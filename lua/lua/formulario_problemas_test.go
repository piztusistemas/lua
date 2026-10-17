package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"edicion/xerar"
)

func TestParseMathCrono(t *testing.T) {
	casos := []struct {
		md, tex string
		display bool
	}{
		{"A área é $\\pi r^2$ m².", "\\pi r^2", false},
		{"Vale $$x = \\frac{1}{2}$$ sempre.", "x = \\frac{1}{2}", true},
		{"Fórmula \\(a_1 * b_2\\) en liña.", "a_1 * b_2", false},
		{"Con espazos $ {x^2} = 0 $ tamén.", "{x^2} = 0", false},
	}
	for _, c := range casos {
		var atopada *cronoTramo
		for _, tr := range parseInlineCrono(c.md) {
			if tr.Math != "" {
				tr := tr
				atopada = &tr
			}
		}
		if atopada == nil || atopada.Math != c.tex || atopada.Display != c.display {
			t.Errorf("%q: %+v", c.md, atopada)
		}
	}
	// Prezos con $ non son fórmulas.
	for _, tr := range parseInlineCrono("custa 5$ ou 6$ e \\$7") {
		if tr.Math != "" {
			t.Errorf("fórmula falsa: %+v", tr)
		}
	}
	// Bloque de varias liñas.
	bs := parseMarkdownCrono("Antes\n\n$$\na + b\n= c\n$$\n\nDespois")
	if len(bs) != 3 || bs[1].Tramos[0].Math != "a + b\n= c" || !bs[1].Tramos[0].Display {
		t.Fatalf("%+v", bs)
	}
	l := latexBloquesCrono(bs, nil, nil)
	if !strings.Contains(l, `\[a + b`) || strings.Contains(l, `\$`) {
		t.Fatalf("LaTeX: %s", l)
	}
	if md := mdBloquesCrono(parseMarkdownCrono("Sexa $x_1^2$."), nil, nil, 2); !strings.Contains(md, "$x_1^2$") {
		t.Fatalf("Markdown: %s", md)
	}
}

func formularioExame() (Formulario, DatosFormulario) {
	f := normalizarFormulario(Formulario{Nome: "Exame de física", Campos: []CampoFormulario{
		{ID: "alumno", Tipo: campoTexto, Etiqueta: "Alumno/a"},
		{ID: "preguntas", Tipo: campoProblemas, Etiqueta: "Preguntas"},
	}})
	ps := []ProblemaFormulario{
		{Enunciado: "Un móbil percorre {d} m en {t} s. Calcula a velocidade media $v_m$.", Puntos: "2",
			Variables: []xerar.VariableProblema{{Nome: "d", Modo: xerar.VariableFixa, Valor: "120"}, {Nome: "t", Modo: xerar.VariableAleatoria, De: "2", Ata: "6"}},
			Solucion:  &SolucionProblema{Texto: "$v_m = \\dfrac{d}{t} = \\dfrac{ {d} }{ {t} }$", Resultado: "{v} m/s", Calculos: []xerar.CalculoProblema{{Nome: "v", Expr: "d/t"}}, Parametrica: true}},
		{Enunciado: "Pregunta sen solución: deriva $x^2$."},
	}
	d := DatosFormulario{Formulario: f.ID, Valores: map[string]any{"alumno": "", "preguntas": ps}}
	return f, d
}

func TestExpandirProblemas(t *testing.T) {
	f, d := formularioExame()
	tx := novosTextosDoc(nil, map[string]string{})
	m, _ := xerar.NovoMotor(xerar.Opcions{}, false, 0)
	d.Xeracion = &XeracionProblemas{Semente: 4, Variantes: 3, Solucions: solucionsFinal}
	bloques, _ := bloquesFormulario(f, d, tx)
	out, avisos := expandirProblemas(bloques, d.Xeracion, m, tx)
	var saltos, solucions, enunciados int
	for _, b := range out {
		switch b.Tipo {
		case bloqueSalto:
			saltos++
		case bloqueSolucion:
			solucions++
		case bloqueMarkdown:
			if strings.Contains(b.Texto, "móbil") {
				enunciados++
				if !strings.HasPrefix(b.Texto, "**1.** *(2 pt)*") || strings.Contains(b.Texto, "{d}") {
					t.Errorf("enunciado: %q", b.Texto)
				}
			}
		case bloqueProblemas:
			t.Fatal("quedou un bloque sen expandir")
		}
	}
	// 3 copias (2 saltos entre elas) + 1 salto antes do solucionario.
	if enunciados != 3 || saltos != 3 || solucions != 3 {
		t.Fatalf("enunciados %d, saltos %d, solucións %d\n%+v", enunciados, saltos, solucions, out)
	}
	if len(avisos) == 0 || !strings.Contains(strings.Join(avisos, "|"), "aínda non ten solución") {
		t.Fatalf("avisos: %v", avisos)
	}
	// A mesma semente dá o mesmo documento; a pregunta 1 é a mesma en editor e documento.
	r := (&App{}).calcularCon(m, PedidoCalcularProblemas{Semente: 4, Problemas: []ProblemaParaCalcular{{Indice: 0, Problema: d.Valores["preguntas"].([]ProblemaFormulario)[0]}}})
	if !strings.Contains(out[1].Texto, r.Calculados[0].Enunciado) && !strings.Contains(out[2].Texto, r.Calculados[0].Enunciado) {
		t.Fatalf("o editor e o documento non casan:\n%q\n%+v", r.Calculados[0].Enunciado, out[:4])
	}
}

func TestFormularioProblemasPDF(t *testing.T) {
	if !commandExists("xelatex") && !commandExists("pdflatex") {
		t.Skip("sen LaTeX")
	}
	f, d := formularioExame()
	d.Xeracion = &XeracionProblemas{Semente: 1, Variantes: 2, Solucions: solucionsXunto}
	p := PedidoFormulario{Formulario: f, Datos: d}
	if ruta := findMaxima(); ruta != "" {
		m, err := xerar.NovoMotor(xerar.Opcions{Maxima: ruta}, true, 20*time.Second)
		if err != nil {
			t.Fatal(err)
		}
		defer m.Close()
		p.motor = m
	}
	for _, motor := range []string{"pdflatex", "xelatex"} {
		if !commandExists(motor) {
			continue
		}
		res, err := compilarFormulario(p, motor, nil, t.TempDir(), nil, "exame")
		if err != nil {
			t.Fatalf("%s: %v\n%s", motor, err, res.Log)
		}
		if dir := os.Getenv("LUA_PDF_PROBA"); dir != "" {
			_ = os.WriteFile(filepath.Join(dir, "problemas-"+motor+".pdf"), res.PDF, 0o644)
		}
	}
	md, _, err := markdownFormulario(p, "md", t.TempDir(), agoraFormulario, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(md, `$v_m = \dfrac{d}{t}`) || !strings.Contains(md, "> **Solución**") {
		t.Fatalf("Markdown:\n%s", md)
	}
}

func TestResolverProblemaTandem(t *testing.T) {
	ruta := findMaxima()
	if ruta == "" {
		t.Skip("sen Maxima")
	}
	m, err := xerar.NovoMotor(xerar.Opcions{Maxima: ruta}, true, 20*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	// A IA de proba equivócase na primeira ronda (unha comprobación falsa) e
	// arránxao cando Lua lle devolve o fallo de Maxima.
	var rondas []string
	orixinal := chamarIAProblema
	defer func() { chamarIAProblema = orixinal }()
	chamarIAProblema = func(s Settings, sistema string, msgs []IAMensaxe) (string, error) {
		rondas = append(rondas, msgs[len(msgs)-1].Texto)
		if len(rondas) == 1 {
			return `{"solucion":"$x = {S}$","resultado":"{S}","calculos":[{"nome":"P","expr":"expand((x-{r})*(x+2))"},{"nome":"S","expr":"solve(P=0,x)"}],"comprobacions":["length(S) = 3"],"parametrica":true}`, nil
		}
		return "```json\n" + `{"solucion":"Factorizamos: $ {F} = 0$, logo $ {S} $.","resultado":"$ {S} $","calculos":[{"nome":"P","expr":"expand((x-r)*(x+2))"},{"nome":"F","expr":"factor(P)"},{"nome":"S","expr":"solve(P=0,x)"}],"comprobacions":["every(lambda([s], ratsimp(subst(s, P))=0), S)"]}` + "\n```", nil
	}
	p := PedidoResolverProblema{Modo: resolverMaxima, Semente: 2, Problema: ProblemaFormulario{
		Enunciado: "Resolve $ {P} = 0 $.",
		Variables: []xerar.VariableProblema{{Nome: "r", Modo: xerar.VariableAleatoria, De: "1", Ata: "6"}},
		Calculos:  []xerar.CalculoProblema{{Nome: "P", Expr: "expand((x-r)*(x+2))"}},
	}}
	r, err := resolverCon(Settings{}, m, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rondas) != 2 || !strings.Contains(rondas[1], "length(S) = 3") {
		t.Fatalf("rondas: %q", rondas)
	}
	if len(r.Fallos) > 0 || !r.Solucion.Parametrica || r.Solucion.Orixe != "ia+maxima" || !strings.Contains(r.Calculado.Solucion, "x=") {
		t.Fatalf("%+v", r)
	}
	// Os {r} dentro das expresións quítanse.
	if limparCalculos([]xerar.CalculoProblema{{Nome: "{F}", Expr: "{m}*{a}"}})[0].Expr != "m*a" {
		t.Fatal("limparCalculos")
	}
}

func TestProblemasEnWordEOpenDocument(t *testing.T) {
	if !commandExists("pdflatex") || !commandExists("pdftoppm") {
		t.Skip("sen LaTeX ou pdftoppm")
	}
	f, d := formularioExame()
	d.Xeracion = &XeracionProblemas{Semente: 1, Solucions: solucionsXunto}
	for _, formato := range []string{"docx", "odt"} {
		datos, avisos, err := datosOfimaticaDe(pedidoOfiFormulario(t, f, d), nil, formato)
		if err != nil {
			t.Fatal(err)
		}
		for _, a := range avisos {
			if strings.Contains(a, "fórmulas") {
				t.Fatalf("%s: %s", formato, a)
			}
		}
		// As fórmulas van coma imaxes (v_m, a fracción, o resultado, x^2...).
		if n := strings.Count(string(datos), ".png"); n < 4 {
			t.Fatalf("%s: só %d imaxes", formato, n)
		}
		if dir := os.Getenv("LUA_PDF_PROBA"); dir != "" {
			_ = os.WriteFile(filepath.Join(dir, "problemas."+formato), datos, 0o644)
		}
	}
}

func TestProblemaConCircuito(t *testing.T) {
	if !commandExists("pdflatex") || !commandExists("pdftoppm") {
		t.Skip("sen LaTeX")
	}
	f := normalizarFormulario(Formulario{Nome: "Electricidade", Campos: []CampoFormulario{{ID: "preg", Tipo: campoProblemas, Etiqueta: "Preguntas"}}})
	circuito := "```circuitikz\n\\draw (0,0) to[battery1, l=${V}\\,\\mathrm{V}$] (0,3) to[R, l=$R_1 = {R1}\\,\\Omega$] (4,3) to[R, l=$R_2$] (4,0) -- (0,0);\n```"
	ps := []ProblemaFormulario{{Enunciado: "No circuíto da figura, calcula a intensidade.\n\n" + circuito,
		Variables: []xerar.VariableProblema{{Nome: "V", Modo: xerar.VariableFixa, Valor: "12"}, {Nome: "R1", Modo: xerar.VariableAleatoria, De: "2", Ata: "8"}}}}
	d := DatosFormulario{Formulario: f.ID, Valores: map[string]any{"preg": ps}}
	// Vista: o circuíto compílase e sae coma imaxe.
	m, _ := xerar.NovoMotor(xerar.Opcions{}, false, 0)
	r := (&App{}).calcularCon(m, PedidoCalcularProblemas{Semente: 1, Problemas: []ProblemaParaCalcular{{Problema: ps[0]}}})
	r.Debuxos = debuxosVista(r.Calculados...)
	if len(r.Debuxos) != 1 {
		t.Fatalf("debuxos: %+v", r.Debuxos)
	}
	for k, v := range r.Debuxos {
		if v.Erro != "" || !strings.HasPrefix(v.Imaxe, "data:image/") || v.AnchoPx < 50 || !strings.HasPrefix(k, "circuitikz\n\\draw") || strings.Contains(k, "{R1}") {
			t.Fatalf("%q: %+v", k, v.Erro)
		}
		if commandExists("pdftocairo") && !strings.HasPrefix(v.Imaxe, "data:image/svg+xml") {
			t.Fatal("con pdftocairo a vista ten que ser vectorial (SVG)")
		}
		if dir := os.Getenv("LUA_PDF_PROBA"); dir != "" {
			_ = os.WriteFile(filepath.Join(dir, "vista.html"), []byte(fmt.Sprintf(`<body style="margin:20px;background:#fff"><div style="width:420px"><img src="%s" style="width:%.0fpx;max-width:100%%"></div></body>`, v.Imaxe, v.AnchoPx)), 0o644)
		}
	}
	// Un debuxo con erro dío, sen romper os demais.
	mal := debuxarParaVista([][2]string{{"tikz", "\\draw (0,0) -- (1,1);"}, {"tikz", "\\draww (0,0);"}})
	if mal[claveDebuxo("tikz", "\\draw (0,0) -- (1,1);")].Imaxe == "" || mal[claveDebuxo("tikz", "\\draww (0,0);")].Erro == "" {
		t.Fatalf("%+v", mal)
	}
	// PDF.
	p := PedidoFormulario{Formulario: f, Datos: d}
	res, err := compilarFormulario(p, "pdflatex", nil, t.TempDir(), nil, "circuito")
	if err != nil {
		t.Fatalf("%v\n%s", err, res.Log)
	}
	if dir := os.Getenv("LUA_PDF_PROBA"); dir != "" {
		_ = os.WriteFile(filepath.Join(dir, "circuito.pdf"), res.PDF, 0o644)
	}
	if !strings.Contains(res.Tex, `\begin{circuitikz}[european`) {
		t.Fatal("o PDF non leva o circuíto")
	}
	// Word: o circuíto é unha imaxe máis.
	datos, avisos, err := datosOfimaticaDe(pedidoOfiFormulario(t, f, d), nil, "docx")
	if err != nil || len(avisos) > 0 {
		t.Fatalf("%v %v", err, avisos)
	}
	if strings.Count(string(datos), "media/image") < 2 {
		t.Fatal("o Word non leva o circuíto")
	}
}

func TestUnidadesNonSonVariables(t *testing.T) {
	c := CampoFormulario{ID: "p", Etiqueta: "P"}
	v := []ProblemaFormulario{{Enunciado: "Circula $I = {I}\\,\\mathrm{mA}$ e $m = {m}\\,\\text{kg}$, $x^{n}$, $P = 5\\,{kW}$; e $ {q} $.",
		Variables: []xerar.VariableProblema{{Nome: "I", Modo: xerar.VariableFixa, Valor: "50"}, {Nome: "m", Modo: xerar.VariableFixa, Valor: "2"}}}}
	av := avisosProblemas(c, v)
	if len(av) != 1 || !strings.Contains(av[0], "{q}") {
		t.Fatalf("só debería avisar de {q}: %v", av)
	}
}

func TestSolucionsSoResultado(t *testing.T) {
	f, d := formularioExame()
	tx := novosTextosDoc(nil, map[string]string{})
	m, _ := xerar.NovoMotor(xerar.Opcions{}, false, 0)
	pasos := func(out []bloqueDoc) (resultados, conPasos int, titulo string) {
		for _, b := range out {
			if b.Tipo == bloqueSolucion {
				if strings.Contains(b.Texto, "dfrac") {
					conPasos++
				} else if strings.Contains(b.Texto, "m/s") {
					resultados++
				}
			}
			if b.Tipo == bloqueTitulo {
				titulo = b.Texto
			}
		}
		return
	}
	for _, c := range []struct {
		modo, titulo string
		final        bool
	}{{solucionsResultadoXunto, "", false}, {solucionsResultadoFinal, "Resultados", true}} {
		d.Xeracion = &XeracionProblemas{Semente: 4, Solucions: c.modo}
		if d.Xeracion.normalizar().Solucions != c.modo {
			t.Fatalf("%s: o modo non se conserva", c.modo)
		}
		bloques, _ := bloquesFormulario(f, d, tx)
		out, avisos := expandirProblemas(bloques, d.Xeracion, m, tx)
		r, p, titulo := pasos(out)
		if r != 1 || p != 0 {
			t.Fatalf("%s: %d resultados, %d con resolución\n%+v", c.modo, r, p, out)
		}
		if c.final && titulo != c.titulo {
			t.Fatalf("%s: título final %q", c.modo, titulo)
		}
		if !c.final && out[len(out)-1].Tipo == bloqueSalto {
			t.Fatalf("%s: debaixo non leva solucionario ao final", c.modo)
		}
		if !strings.Contains(strings.Join(avisos, "|"), "aínda non ten solución") {
			t.Fatalf("%s: avisos %v", c.modo, avisos)
		}
	}
}

// Cada copia (iteración) leva por defecto a súa semente: semente, semente+1...
// e, con ela, os seus valores.
func TestVariantesSementesDistintas(t *testing.T) {
	f, d := formularioExame()
	ps := d.Valores["preguntas"].([]ProblemaFormulario)
	ps[0].Variables[1].De, ps[0].Variables[1].Ata = "1", "1000000"
	tx := novosTextosDoc(nil, map[string]string{})
	m, _ := xerar.NovoMotor(xerar.Opcions{}, false, 0)
	d.Xeracion = &XeracionProblemas{Semente: 4821, Variantes: 3}
	bloques, _ := bloquesFormulario(f, d, tx)
	out, _ := expandirProblemas(bloques, d.Xeracion, m, tx)
	titulos, enunciados := map[string]bool{}, map[string]bool{}
	for _, b := range out {
		if b.Tipo == bloqueTitulo && b.Nivel == 2 {
			titulos[b.Texto] = true
		}
		if b.Tipo == bloqueMarkdown && strings.Contains(b.Texto, "móbil") {
			enunciados[b.Texto] = true
		}
	}
	if len(titulos) != 3 || len(enunciados) != 3 {
		t.Fatalf("as 3 copias deberían ser distintas: títulos %v, enunciados %v", titulos, enunciados)
	}
}
