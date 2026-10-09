package plantilla

import (
	"strings"
	"testing"
)

func TestTextoDeJSON(t *testing.T) {
	casos := []struct {
		entrada  string
		agardado string
	}{
		{`"ola"`, "ola"},
		{`12`, "12"},
		{`6.5`, "6.5"},
		{`true`, "true"},
		{`null`, ""},
		{``, ""},
		{`["un", "dous", ""]`, "un\ndous"},
		{`{"a":1}`, `{"a":1}`},
	}
	for _, c := range casos {
		got, err := textoDeJSON([]byte(c.entrada))
		if err != nil {
			t.Errorf("textoDeJSON(%q): %v", c.entrada, err)
			continue
		}
		if got != c.agardado {
			t.Errorf("textoDeJSON(%q) = %q, esperaba %q", c.entrada, got, c.agardado)
		}
	}
}

func TestExtraerObjetoJSON(t *testing.T) {
	casos := []struct{ entrada, agardado string }{
		{`Aquí tes: {"a":1} Espero que che sirva.`, `{"a":1}`},
		{`{"a":"pecha } dentro dunha cadea"}`, `{"a":"pecha } dentro dunha cadea"}`},
		{`{"a":{"b":2}}`, `{"a":{"b":2}}`},
		{`sen ningún obxecto`, `sen ningún obxecto`},
		{`{"a":"escapado \" e }"}`, `{"a":"escapado \" e }"}`},
	}
	for _, c := range casos {
		if got := ExtraerObjetoJSON(c.entrada); got != c.agardado {
			t.Errorf("ExtraerObjetoJSON(%q) = %q, esperaba %q", c.entrada, got, c.agardado)
		}
	}
}

func TestTextoVisibleHTML(t *testing.T) {
	out := TextoVisibleHTML(`<html><head><style>p{color:#fff}</style><script>var x=1</script></head><body><h1>Ola</h1><p>Mundo</p></body></html>`)
	if !strings.Contains(out, "Ola") || !strings.Contains(out, "Mundo") {
		t.Errorf("perdeuse o texto visible: %q", out)
	}
	if strings.Contains(out, "color") || strings.Contains(out, "var x") {
		t.Errorf("colóuselle o script/style: %q", out)
	}
}

func TestRepararBarrasLatex(t *testing.T) {
	casos := []struct{ entrada, agardado string }{
		{`{\Large {{EMPRESA}} }\[0.2em]`, `{\Large {{EMPRESA}} }\\[0.2em]`},
		{`\footnotesize\bfseries FACTURA\[-0.2ex]`, `\footnotesize\bfseries FACTURA\\[-0.2ex]`},
		{"unha liña\\\noutra liña", "unha liña\\\\\noutra liña"},
		// Xa correcto: non se toca.
		{`fila un\\[4pt]`, `fila un\\[4pt]`},
		{"fila un\\\\\nfila dous", "fila un\\\\\nfila dous"},
		// Display math de verdade (pecha con \]) queda intacto.
		{`\[ x^2 + 1 \]`, `\[ x^2 + 1 \]`},
		// Comandos normais non se tocan.
		{`\textbf{ola} \begin{center}`, `\textbf{ola} \begin{center}`},
		{`$x_1 + y^2$`, `$x_1 + y^2$`},
	}
	for _, c := range casos {
		if got := RepararBarrasLatex(c.entrada); got != c.agardado {
			t.Errorf("RepararBarrasLatex(%q) = %q, esperaba %q", c.entrada, got, c.agardado)
		}
	}
}

// TestXeradorIAParsear: a resposta da IA tómase con tolerancia (números onde se
// pediu texto), engádese {{CORPO}} se falta, dedúcese o motor Unicode e
// normalízanse as variables.
func TestXeradorIAParsear(t *testing.T) {
	x := &XeradorIA{}
	raw := "```json\n" + `{
      "nome": "Circular",
      "descricion": "do centro",
      "latex": "\\documentclass{article}\n\\begin{document}\nOla\n\\end{document}",
      "variables": [
        {"nome": "centro", "etiqueta": "Centro", "valor": 12},
        {"nome": "corpo", "etiqueta": "reservado", "valor": "x"}
      ]
    }` + "\n```"
	p, err := x.Parsear(raw)
	if err != nil {
		t.Fatalf("Parsear: %v", err)
	}
	if !strings.Contains(p.Latex, MarcadorCorpo) {
		t.Errorf("debería engadir %s: %q", MarcadorCorpo, p.Latex)
	}
	if !strings.Contains(p.Latex, `\end{document}`) {
		t.Errorf("o marcador ten que ir antes de \\end{document}: %q", p.Latex)
	}
	if p.Nome != "Circular" || p.Descricion != "do centro" {
		t.Errorf("nome/descrición mal: %+v", p)
	}
	if len(p.Variables) != 1 || p.Variables[0].Nome != "CENTRO" || p.Variables[0].Valor != "12" {
		t.Errorf("as variables non se normalizaron (nin o número se fixo texto): %+v", p.Variables)
	}

	// fontspec implica xelatex.
	font, err := x.Parsear(`{"nome":"X","latex":"\\usepackage{fontspec}\n{{CORPO}}"}`)
	if err != nil {
		t.Fatal(err)
	}
	if font.Motor != "xelatex" {
		t.Errorf("con fontspec o motor ten que ser xelatex: %q", font.Motor)
	}

	// Sen nome, un por defecto; e unha resposta ilexible dá un erro explicado.
	if sen, err := x.Parsear(`{"latex":"{{CORPO}}"}`); err != nil || sen.Nome == "" {
		t.Errorf("unha plantilla sen nome debería recibir un por defecto: %+v, %v", sen, err)
	}
	if _, err := x.Parsear("isto non é json"); err == nil {
		t.Error("unha resposta ilexible ten que dar erro")
	}
	if _, err := x.Parsear(`{"latex":""}`); err == nil {
		t.Error("sen LaTeX ten que dar erro")
	}
}

// TestXeradorIAXerar: as fontes xúntanse na petición e o resultado paréase; sen
// fontes ou sen provedor, erro claro.
func TestXeradorIAXerar(t *testing.T) {
	var sistema, peticion string
	x := &XeradorIA{
		Sistema: "SISTEMA",
		ChamarIA: func(s, p string) (string, error) {
			sistema, peticion = s, p
			return `{"nome":"N","latex":"{{CORPO}}"}`, nil
		},
	}
	p, err := x.Xerar(PlantillaIARequest{Descricion: "unha circular", Imaxes: []string{"logo.png"}})
	if err != nil {
		t.Fatalf("Xerar: %v", err)
	}
	if p.Nome != "N" || sistema != "SISTEMA" {
		t.Errorf("resultado/sistema mal: %+v %q", p, sistema)
	}
	if !strings.Contains(peticion, "unha circular") || !strings.Contains(peticion, "logo.png") {
		t.Errorf("a petición non leva as fontes: %q", peticion)
	}

	// Sen fontes non se chama á IA.
	if _, err := x.Xerar(PlantillaIARequest{}); err == nil {
		t.Error("sen fontes ten que dar erro")
	}
	// Sen provedor tampouco.
	if _, err := (&XeradorIA{}).Xerar(PlantillaIARequest{Descricion: "algo"}); err == nil {
		t.Error("sen provedor ten que dar erro")
	}
}

// TestXeradorIADocumentosMostra: os ficheiros de texto van tal cal; un binario
// que non é PDF rexéitase; e os PDF pasan pola costura.
func TestXeradorIADocumentosMostra(t *testing.T) {
	x := &XeradorIA{
		ExtraerTextoPDF: func(b64 string) (string, error) { return "texto do pdf", nil },
		TruncarRunas:    func(s string, n int) string { return s },
	}
	texto, err := x.textoDeDocumentosMostra([]DocumentoMostra{
		{FileName: "modelo.tex", DataB64: "dW5oIG1vZGVsbyBlbiB0ZXh0bw=="}, // "un modelo en texto"
		{FileName: "modelo.pdf", DataB64: "cGRm"},                         // calquera cousa: a costura responde
	})
	if err != nil {
		t.Fatalf("textoDeDocumentosMostra: %v", err)
	}
	if !strings.Contains(texto, "[modelo.tex]") || !strings.Contains(texto, "texto do pdf") {
		t.Errorf("faltan fontes no texto: %q", texto)
	}

	// Un binario que non sexa UTF-8 nin PDF rexéitase.
	if _, err := x.textoDeDocumentosMostra([]DocumentoMostra{{FileName: "cousa.bin", DataB64: "//8="}}); err == nil {
		t.Error("un binario non-PDF ten que rexeitarse")
	}
}
