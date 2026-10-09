package lectura

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"edicion/ia"
)

const mostras = "../../luayang/mostras"

func mostra(t *testing.T, contén string) string {
	t.Helper()
	if !HaiPoppler() {
		t.Skip("sen poppler")
	}
	l, _ := filepath.Glob(filepath.Join(mostras, "*.pdf"))
	for _, p := range l {
		if strings.Contains(p, contén) {
			return p
		}
	}
	t.Skipf("non está a mostra %s", contén)
	return ""
}

func TestClasificarMostras(t *testing.T) {
	casos := map[string][]string{
		"lenguacastellana":   {MetodoVision, MetodoTexto},
		"matematicasiiex426": {MetodoVision, MetodoVision, MetodoVision, MetodoVision},
		"quimica":            {MetodoVision, MetodoVision, MetodoVision, MetodoVision},
		"inglesex4rev":       {"", MetodoVision},
	}
	for nome, esperado := range casos {
		pax, err := Clasificar(mostra(t, nome))
		if err != nil {
			t.Fatal(err)
		}
		if len(pax) != len(esperado) {
			t.Fatalf("%s: %d páxinas", nome, len(pax))
		}
		for i, e := range esperado {
			if e != "" && pax[i].Metodo != e {
				t.Errorf("%s páxina %d: %s (%s), esperaba %s", nome, i+1, pax[i].Metodo, pax[i].Motivo, e)
			}
		}
	}
	pax, _ := Clasificar(mostra(t, "matematicasiiex426"))
	e := Estimar(pax, 3, 15, "USD")
	if e.PaxinasVision != 4 || !e.ConCusto || e.Custo <= 0 || e.Custo > 1 {
		t.Errorf("estimación: %+v", e)
	}
}

func TestCompararDetectaCambios(t *testing.T) {
	capa := "Calcule os gramos de H2S que se introduciron o inicio. Razoe cal ten maior radio atómico."
	dub, cob := Comparar(capa, "Calcule os gramos de $\\ce{H2S}$ que se introduciron ao inicio. Razoe cal ten maior raio atómico.")
	if !containsAll(dub, "radio") || cob >= 1 {
		t.Errorf("non detectou o cambio: %v %.2f", dub, cob)
	}
	dub, cob = Comparar("A área é 10 unidades", "A área é $10$ unidades")
	if len(dub) > 0 || cob != 1 {
		t.Errorf("falso positivo: %v %v", dub, cob)
	}
}

func containsAll(l []string, ws ...string) bool {
	for _, w := range ws {
		ok := false
		for _, x := range l {
			if x == w {
				ok = true
			}
		}
		if !ok {
			return false
		}
	}
	return true
}

// iaFalsa responde coma Anthropic: a transcrición manual da proba e unha
// estrutura cun exercicio.
func iaFalsa(t *testing.T, transcricion string) (ia.Config, func()) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var corpo map[string]any
		_ = json.Unmarshal(b, &corpo)
		sis, _ := json.Marshal(corpo["system"])
		var resposta any
		if strings.Contains(string(sis), "transcritor") {
			resposta = map[string]any{"markdown": transcricion + "\n\n[[FIGURA-1]]", "figuras": []any{map[string]any{"descricion": "gráfica", "caixa": []float64{0.1, 0.1, 0.5, 0.3}, "tipo": "plot", "codigo": "plot2d(x^2,[x,-1,1])"}}}
		} else {
			resposta = map[string]any{"titulo": "Matemáticas II", "materia": "Matemáticas II", "linguas": []string{"gl"}, "exercicios": []any{
				map[string]any{"titulo": "Pregunta 2", "apartado": "2", "puntos": 2, "paxina": 1, "enunciado": "**2.2.** Discuta, segundo os valores do parámetro $m$, o sistema", "conceptos": []string{"Sistemas de ecuacións"}},
			}}
		}
		j, _ := json.Marshal(resposta)
		out, _ := json.Marshal(map[string]any{"content": []any{map[string]any{"type": "text", "text": string(j)}}, "stop_reason": "end_turn"})
		_, _ = w.Write(out)
	}))
	return ia.Config{Provedor: ia.ProvedorAnthropic, BaseURL: srv.URL, APIKey: "k", Modelo: "m"}, srv.Close
}

func TestLerConVisionFalsa(t *testing.T) {
	pdf := mostra(t, "matematicasiiex426")
	manual, err := os.ReadFile("../../luayang/proba_lectura/matematicas_ii_vision.md")
	if err != nil {
		t.Skip("sen a transcrición manual")
	}
	corpo := string(manual)
	if i := strings.Index(corpo[3:], "---"); i >= 0 {
		corpo = corpo[i+6:]
	}
	cfg, pechar := iaFalsa(t, corpo)
	defer pechar()
	pax, _ := Clasificar(pdf)
	dir := t.TempDir()
	lidas, err := Ler(context.Background(), pdf, pax[:2], Opcions{IA: cfg, Dir: dir, Prefixo: "mates"})
	if err != nil {
		t.Fatal(err)
	}
	p1 := lidas[0]
	if p1.Metodo != MetodoVision || !strings.Contains(p1.Markdown, `\begin{array}`) || !strings.Contains(p1.Markdown, "](raw/imaxes/mates-p1-f1.png)") {
		t.Fatalf("páxina 1: %s\n%s", p1.Metodo, p1.Markdown[:300])
	}
	if _, err := os.Stat(filepath.Join(dir, "imaxes", "mates-p1-f1.png")); err != nil {
		t.Error("non se recortou a figura")
	}
	// A transcrición manual cubre as páxinas 1 e 2: na páxina 1 a cobertura
	// ten que ser alta.
	if p1.Cobertura < 0.85 {
		t.Errorf("cobertura %.2f, dúbidas %v", p1.Cobertura, p1.Dubidas)
	}
	if p1.Imaxe == "" {
		t.Error("sen imaxe da páxina para a revisión")
	}
	e, err := Estruturar(cfg, lidas, "matematicasii")
	if err != nil || len(e.Exercicios) != 1 || e.Exercicios[0].Conceptos[0] != "Sistemas de ecuacións" {
		t.Fatalf("%+v %v", e, err)
	}
}

func TestEstruturarSenIA(t *testing.T) {
	pax := []PaxinaLida{
		{Num: 1, Markdown: "PAU 2026\n\nO exame consta de 4 preguntas.\n\n**PREGUNTA 1. ESTATÍSTICA**\n\nTexto 1\n\n**PREGUNTA 2. ÁLXEBRA**\n\nTexto 2a"},
		{Num: 2, Markdown: "continuación 2\n\nPREGUNTA 3. ANÁLISE\n\nTexto 3"},
	}
	e := EstruturarSenIA(pax, "x")
	if len(e.Exercicios) != 3 || !strings.Contains(e.Exercicios[1].Enunciado, "continuación 2") || e.Exercicios[2].Paxina != 2 || !strings.Contains(e.Instrucions, "4 preguntas") {
		t.Fatalf("%+v", e)
	}
}

func TestTextoAMarkdown(t *testing.T) {
	pdf := mostra(t, "lenguacastellana")
	capa, _ := executar("pdftotext", "-f", "1", "-l", "1", pdf, "-")
	md := textoAMarkdown(capa)
	if strings.Contains(md, "\n7\n") || strings.Contains(md, "\n12\n") {
		t.Errorf("quedaron os números de liña:\n%s", md[:600])
	}
	if !strings.Contains(md, "**PAU**") || !strings.Contains(md, "**PREGUNTA 1.") {
		t.Errorf("sen títulos:\n%s", md[:800])
	}
}

func TestEstruturarNumeradasConPuntos(t *testing.T) {
	pax := []PaxinaLida{{Num: 1, Markdown: "**1. Read the text. (3 points)**\n\nText\n\n**2. Answer the two subsections. (2.5 points)**\n\n1. It is said that...\n2. The referee...\n\n3. Fill each gap. (2 points)\n\ngaps"}}
	e := EstruturarSenIA(pax, "x")
	if len(e.Exercicios) != 3 || !strings.Contains(e.Exercicios[1].Enunciado, "The referee") {
		t.Fatalf("%+v", e.Exercicios)
	}
}
