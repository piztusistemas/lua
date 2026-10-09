package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lua-wails/grafico"
)

// formularioGrafico: cinco cantidades (as notas agrupadas) e unha táboa de
// categorías/cantidades, que son as dúas formas de darlle datos a un gráfico.
func formularioGrafico(tipo string, taboa bool) Formulario {
	f := normalizarFormulario(Formulario{
		Nome: "Resultados",
		Campos: []CampoFormulario{
			{Tipo: campoNumero, Etiqueta: "Suspensos"},
			{Tipo: campoNumero, Etiqueta: "Suficientes"},
			{Tipo: campoNumero, Etiqueta: "Bens"},
			{Tipo: campoNumero, Etiqueta: "Notables"},
			{Tipo: campoNumero, Etiqueta: "Sobresaíntes"},
			{Tipo: campoTaboa, Etiqueta: "Por grupo", Columnas: []CampoFormulario{
				{Tipo: campoTexto, Etiqueta: "Grupo"},
				{Tipo: campoNumero, Etiqueta: "Aprobados"},
				{Tipo: campoNumero, Etiqueta: "Suspensos"},
			}},
		},
	})
	nodos := []NodoFluxo{}
	graf := NodoFluxo{ID: "g", Tipo: nodoGrafico, Config: map[string]any{"tipo": tipo, "titulo": "Distribución das notas"}, Entradas: map[string]string{}}
	if taboa {
		nodos = append(nodos, NodoFluxo{ID: "c_taboa", Tipo: nodoCampo, Campo: f.Campos[5].ID})
		graf.Entradas["taboa"] = "c_taboa"
		graf.Config["columna"] = f.Campos[5].Columnas[0].ID
		graf.Config["columnas"] = []any{f.Campos[5].Columnas[1].ID, f.Campos[5].Columnas[2].ID}
	} else {
		for i := 0; i < 5; i++ {
			id := "c" + string(rune('a'+i))
			nodos = append(nodos, NodoFluxo{ID: id, Tipo: nodoCampo, Campo: f.Campos[i].ID})
			graf.Entradas["v"+string(rune('1'+i))] = id
		}
	}
	nodos = append(nodos, graf)
	f.Fluxo = &Fluxo{Nodos: nodos, Documento: []string{"g"}}
	return f
}

func datosGrafico() DatosFormulario {
	return DatosFormulario{Valores: map[string]any{
		"suspensos": "2", "suficientes": "1", "bens": "1", "notables": "3", "sobresaintes": "1",
		"por_grupo": []any{
			map[string]any{"grupo": "1º A", "aprobados": "22", "suspensos": "3"},
			map[string]any{"grupo": "1º B", "aprobados": "19", "suspensos": "6"},
			map[string]any{"grupo": "1º C", "aprobados": "24", "suspensos": "1"},
		},
	}}
}

func bloqueGraficoDe(t *testing.T, f Formulario, d DatosFormulario) (bloqueDoc, []string) {
	t.Helper()
	bloques, avisos := bloquesFormulario(f, d, txProba())
	for _, b := range bloques {
		if b.Tipo == bloqueGrafico {
			return b, avisos
		}
	}
	t.Fatalf("non hai bloque de gráfico: %+v (avisos %v)", bloques, avisos)
	return bloqueDoc{}, nil
}

func TestGraficoComCantidadesSoltas(t *testing.T) {
	b, avisos := bloqueGraficoDe(t, formularioGrafico("tarta", false), datosGrafico())
	if len(avisos) != 0 {
		t.Errorf("avisos: %v", avisos)
	}
	if b.Grafico == nil || b.Grafico.Ancho <= 0 {
		t.Fatal("o bloque non leva escena")
	}
	// A vista previa leva categoría e valor; o rótulo vén do campo de orixe.
	if len(b.Pares) != 5 || b.Pares[0] != [2]string{"Suspensos", "2"} || b.Pares[4] != [2]string{"Sobresaíntes", "1"} {
		t.Errorf("pares = %v", b.Pares)
	}
	// O título da tubería sae como subtítulo antes do gráfico.
	bloques, _ := bloquesFormulario(formularioGrafico("tarta", false), datosGrafico(), txProba())
	if len(bloques) != 2 || bloques[0].Tipo != bloqueTitulo || bloques[0].Texto != "Distribución das notas" {
		t.Errorf("bloques = %+v", bloques)
	}
}

func TestGraficoComTaboaEVariasSeries(t *testing.T) {
	b, avisos := bloqueGraficoDe(t, formularioGrafico("barras", true), datosGrafico())
	if len(avisos) != 0 {
		t.Errorf("avisos: %v", avisos)
	}
	if len(b.Pares) != 3 || b.Pares[1] != [2]string{"1º B", "19 / 6"} {
		t.Errorf("pares = %v", b.Pares)
	}
	// Con dúas series a escena leva lenda: dous cadros de cor polo menos.
	cadros := map[string]bool{}
	for _, el := range b.Grafico.Elementos {
		if r, ok := el.(grafico.Rect); ok && r.W < 4 && r.H < 4 {
			cadros[r.Cor] = true
		}
	}
	if len(cadros) != 2 {
		t.Errorf("a lenda debería ter 2 cores, hai %v", cadros)
	}
}

func TestGraficoAnaliseEDatosMalos(t *testing.T) {
	// As dúas formas de datos á vez.
	f := formularioGrafico("tarta", false)
	f.Fluxo.Nodos[len(f.Fluxo.Nodos)-1].Entradas["taboa"] = "c_taboa"
	f.Fluxo.Nodos = append([]NodoFluxo{{ID: "c_taboa", Tipo: nodoCampo, Campo: "por_grupo"}}, f.Fluxo.Nodos...)
	_, res := avaliarFluxoDoc(f, datosGrafico(), txProba())
	if !strings.Contains(strings.Join(res.Avisos, "|"), "non as dúas cousas") {
		t.Errorf("avisos = %v", res.Avisos)
	}
	// Tipo descoñecido e nodo sen datos conectados.
	g := formularioGrafico("fatia", false)
	if _, res := avaliarFluxoDoc(g, datosGrafico(), txProba()); !strings.Contains(strings.Join(res.Avisos, "|"), "tipo de gráfico descoñecido") {
		t.Errorf("avisos = %v", res.Avisos)
	}
	// Datos que o gráfico non admite (negativo nunha tarta): non sae e avisa.
	d := datosGrafico()
	d.Valores["notables"] = "-3"
	bloques, avisos := bloquesFormulario(formularioGrafico("tarta", false), d, txProba())
	for _, b := range bloques {
		if b.Tipo == bloqueGrafico {
			t.Error("un gráfico con valores negativos non debería saír")
		}
	}
	if !strings.Contains(strings.Join(avisos, "|"), "valor negativo") {
		t.Errorf("avisos = %v", avisos)
	}
	// Sen datos: simplemente non sae, sen avisos.
	bloques, avisos = bloquesFormulario(formularioGrafico("tarta", false), DatosFormulario{Valores: map[string]any{}}, txProba())
	if len(avisos) != 0 || len(bloques) != 0 {
		t.Errorf("bloques=%+v avisos=%v", bloques, avisos)
	}
}

func TestCatalogoGraficoOfreceTodosOsTipos(t *testing.T) {
	var cfg ConfigNodo
	for _, c := range catalogoPorTipo[nodoGrafico].Config {
		if c.Clave == "tipo" {
			cfg = c
		}
	}
	if len(cfg.Opcions) != len(grafico.Tipos()) || cfg.Nomes["rosca"] == "" {
		t.Errorf("o selector de tipo non ofrece os tipos do motor: %+v", cfg)
	}
	if !catalogoPorTipo[nodoGrafico].PortosEtiqueta {
		t.Error("o gráfico ten portos con rótulo editable")
	}
}

// Cada tipo sae en LaTeX (TikZ), Markdown, Word e ODT; e o PDF compila cos dous motores.
func TestGraficoEnTodosOsFormatos(t *testing.T) {
	saida := os.Getenv("GRAFICO_SAIDA")
	for _, ti := range grafico.Tipos() {
		f, d := formularioGrafico(ti.ID, false), datosGrafico()
		if ti.Series {
			f, d = formularioGrafico(ti.ID, true), datosGrafico()
		}
		p := PedidoFormulario{Formulario: f, Datos: d}

		tex, avisos, err := latexFormulario(p, "xelatex", nil, t.TempDir(), agoraFormulario, nil, "")
		if err != nil || strings.Contains(strings.Join(avisos, "|"), "gráfico") {
			t.Fatalf("%s: err=%v avisos=%v", ti.ID, err, avisos)
		}
		if !strings.Contains(tex, `\begin{tikzpicture}`) || !strings.Contains(tex, `\node[anchor=`) {
			t.Fatalf("%s: o LaTeX non leva o gráfico en TikZ", ti.ID)
		}
		if strings.Contains(tex, "% ") || strings.Contains(tex, " ") {
			t.Errorf("%s: espazo duro sen traducir no TikZ", ti.ID)
		}

		md, _, err := markdownFormulario(p, "md", t.TempDir(), agoraFormulario, nil)
		if err != nil || !strings.Contains(md, "![") || !strings.Contains(md, "{width=") {
			t.Errorf("%s: Markdown sen imaxe (err=%v):\n%s", ti.ID, err, md)
		}

		for _, formato := range []string{"docx", "odt"} {
			datos, avisos, err := datosOfimaticaDe(pedidoOfiFormulario(t, f, d), nil, formato)
			if err != nil || strings.Contains(strings.Join(avisos, "|"), "gráfico") {
				t.Fatalf("%s/%s: err=%v avisos=%v", ti.ID, formato, err, avisos)
			}
			z := lerZip(t, datos)
			temImaxe := false
			for nome, c := range z {
				if (strings.HasPrefix(nome, "word/media/") || strings.HasPrefix(nome, "Pictures/")) && strings.HasPrefix(c, "\x89PNG") {
					temImaxe = true
				}
			}
			if !temImaxe {
				t.Errorf("%s/%s: o PNG do gráfico non entrou no paquete", ti.ID, formato)
			}
			if formato == "docx" {
				xmlBenFormado(t, "word/document.xml", z["word/document.xml"])
			} else {
				xmlBenFormado(t, "content.xml", z["content.xml"])
			}
			if saida != "" {
				os.WriteFile(filepath.Join(saida, "form_"+ti.ID+"."+formato), datos, 0o644)
			}
		}

		if !commandExists("xelatex") && !commandExists("pdflatex") {
			continue
		}
		for _, motor := range []string{"pdflatex", "xelatex"} {
			if !commandExists(motor) {
				continue
			}
			res, err := compilarFormulario(p, motor, nil, t.TempDir(), nil, "resultados")
			if err != nil || !bytes.HasPrefix(res.PDF, []byte("%PDF")) {
				t.Fatalf("%s con %s non compila: %v\n%s", ti.ID, motor, err, res.Log)
			}
			if saida != "" && motor == "xelatex" {
				os.WriteFile(filepath.Join(saida, "form_"+ti.ID+".pdf"), res.PDF, 0o644)
			}
		}
	}
}

func TestGraficoOfimaticaAbreEnLibreOffice(t *testing.T) {
	if !commandExists("soffice") || !commandExists("pdftotext") {
		t.Skip("sen LibreOffice ou pdftotext")
	}
	f, d := formularioGrafico("rosca", false), datosGrafico()
	for _, formato := range []string{"docx", "odt"} {
		datos, _, err := datosOfimaticaDe(pedidoOfiFormulario(t, f, d), nil, formato)
		if err != nil {
			t.Fatal(err)
		}
		tipo := tipoAdxDocx
		if formato == "odt" {
			tipo = tipoAdxOdt
		}
		pdf, limpar, err := converterAPDFLibreOffice(Adxunto{ID: "x", Nome: "f." + formato, Tipo: tipo}, datos)
		if err != nil {
			t.Fatalf("%s: LibreOffice non o abre: %v", formato, err)
		}
		if fi, err := os.Stat(pdf); err != nil || fi.Size() == 0 {
			t.Errorf("%s: LibreOffice non deixou un PDF", formato)
		}
		limpar()
	}
}
