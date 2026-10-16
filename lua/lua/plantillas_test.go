package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// configIllado illa TODA a configuración de Lua (settings.json,
// plantillas.json e o cartafol de imaxes das plantillas)
// nun temporal. XDG_CONFIG_HOME abonda en Linux, pero os.UserConfigDir
// ignórao en macOS (alí é $HOME/Library/Application Support), así que hai
// que fixar as dúas variables para que a proba non escriba na
// configuración real de quen a executa.
func configIllado(t *testing.T) {
	t.Helper()
	// A CACHÉ (a diferenza da configuración) séguese compartindo coa real,
	// e a propósito: en macOS TeX Live cachea fontes en os.UserCacheDir() e
	// colga de HOME - cun HOME falso e baleiro, pdflatex tardaría moito máis
	// en compilar. Ligar o cartafol de caché ao real deixa iso funcionando
	// sen sacar a configuración do temporal.
	cacheReal, errCache := os.UserCacheDir()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
	if errCache != nil {
		return
	}
	novaCache, err := os.UserCacheDir()
	if err != nil || novaCache == cacheReal {
		return
	}
	if err := os.MkdirAll(filepath.Dir(novaCache), 0o755); err == nil {
		_ = os.Symlink(cacheReal, novaCache)
	}
}

func TestPlantillasRoundTrip(t *testing.T) {
	configIllado(t)
	a := &App{}

	// A biblioteca arranca coas plantillas de exemplo sementadas.
	estado, err := a.ListarPlantillas()
	if err != nil {
		t.Fatalf("listar: %v", err)
	}
	if len(estado.Plantillas) != len(plantillasDeFabrica()) {
		t.Fatalf("esperaba as plantillas de fábrica sementadas, atopei %d", len(estado.Plantillas))
	}
	if estado.Activa != "" {
		t.Errorf("por defecto non debería haber plantilla activa, hai %q", estado.Activa)
	}

	if _, err := a.GardarPlantilla(PlantillaLatex{Nome: "", Latex: "{{CORPO}}"}); err == nil {
		t.Error("esperaba erro sen nome")
	}
	if _, err := a.GardarPlantilla(PlantillaLatex{Nome: "Sen corpo", Latex: `\section{Ola}`}); err == nil {
		t.Error("esperaba erro sen o marcador {{CORPO}}")
	}
	if _, err := a.GardarPlantilla(PlantillaLatex{Nome: "MD sen corpo", Latex: "{{CORPO}}", Markdown: "# Ola"}); err == nil {
		t.Error("esperaba erro co Markdown sen {{CORPO}}")
	}

	nova, err := a.GardarPlantilla(PlantillaLatex{
		Nome:      "Circular",
		Latex:     "{{CENTRO}}\n\n{{CORPO}}\n",
		Variables: []PlantillaVar{{Nome: "centro educativo", Etiqueta: "Centro", Valor: "IES Xistral"}},
	})
	if err != nil {
		t.Fatalf("gardar: %v", err)
	}
	if nova.ID == "" {
		t.Fatal("esperaba unha ID")
	}
	if len(nova.Variables) != 1 || nova.Variables[0].Nome != "CENTRO_EDUCATIVO" {
		t.Errorf("o nome da variable debería normalizarse: %+v", nova.Variables)
	}

	// Editar (mesma ID) actualiza en vez de duplicar.
	nova.Nome = "Circular do centro"
	if _, err := a.GardarPlantilla(nova); err != nil {
		t.Fatalf("actualizar: %v", err)
	}
	estado, _ = a.ListarPlantillas()
	if len(estado.Plantillas) != len(plantillasDeFabrica())+1 {
		t.Fatalf("actualizar non debería crear outra plantilla: %d", len(estado.Plantillas))
	}

	// Activar, e lembrar por ficheiro.
	if err := a.EscollerPlantilla(nova.ID, ""); err != nil {
		t.Fatalf("escoller: %v", err)
	}
	if p := a.plantillaActiva(); p == nil || p.ID != nova.ID {
		t.Fatalf("plantillaActiva non devolveu a escollida: %+v", p)
	}

	if err := a.EliminarPlantilla(nova.ID); err != nil {
		t.Fatalf("eliminar: %v", err)
	}
	if p := a.plantillaActiva(); p != nil {
		t.Errorf("ao borrar a plantilla activa debe quedar ningunha, quedou %+v", p)
	}
}

// TestPlantillaPorFicheiro: cada documento lembra a súa plantilla, e abrir
// un que nunca tivo ningunha non desfai a escolla actual.
func TestPlantillaPorFicheiro(t *testing.T) {
	configIllado(t)
	a := &App{}
	p1, err := a.GardarPlantilla(PlantillaLatex{Nome: "Exame", Latex: "{{CORPO}}"})
	if err != nil {
		t.Fatalf("gardar: %v", err)
	}

	// O ficheiro ten que existir de verdade: podarPorFicheiro limpa as
	// entradas de documentos borrados.
	doc := t.TempDir() + "/curso.json"
	if err := writeFileParaTest(doc); err != nil {
		t.Fatal(err)
	}
	if err := a.EscollerPlantilla(p1.ID, doc); err != nil {
		t.Fatalf("escoller: %v", err)
	}
	if err := a.EscollerPlantilla("", ""); err != nil {
		t.Fatalf("desactivar: %v", err)
	}

	id, err := a.PlantillaParaFicheiro(doc)
	if err != nil {
		t.Fatalf("plantillaParaFicheiro: %v", err)
	}
	if id != p1.ID {
		t.Errorf("esperaba recuperar %q para %s, obtiven %q", p1.ID, doc, id)
	}
	if p := a.plantillaActiva(); p == nil || p.ID != p1.ID {
		t.Error("abrir o documento debería activar a súa plantilla")
	}

	outro := t.TempDir() + "/outro.json"
	if err := writeFileParaTest(outro); err != nil {
		t.Fatal(err)
	}
	id, _ = a.PlantillaParaFicheiro(outro)
	if id != p1.ID {
		t.Errorf("un documento sen plantilla lembrada debe manter a activa, deu %q", id)
	}
}

func TestSubstituirVariables(t *testing.T) {
	p := &PlantillaLatex{Variables: []PlantillaVar{
		{Nome: "CENTRO", Valor: "IES Xistral"},
		{Nome: "TITULO", Valor: "Exame de setembro"},
	}}
	auto := map[string]string{"DATA": "01/09/2026", "TITULO": "exame"}

	out, avisos := substituirVariables("{{CENTRO}} - {{DATA}} - {{TITULO}} - {{CORPO}} - {{NADA}}", p, auto, "", nil)
	if want := "IES Xistral - 01/09/2026 - Exame de setembro - {{CORPO}} - "; out != want {
		t.Errorf("substitución incorrecta:\n  obtido: %q\n  esperado: %q", out, want)
	}
	if len(avisos) != 1 || !strings.Contains(avisos[0], "NADA") {
		t.Errorf("esperaba un aviso por {{NADA}}, obtiven %v", avisos)
	}

	// En LaTeX, unha variable sen valor (definida ou non) ten que deixar un
	// \mbox{}: unha liña baleira antes dun \\ mata a compilación enteira.
	p.Variables = append(p.Variables, PlantillaVar{Nome: "PE", Valor: "   "})
	out, _ = substituirVariables("{{PE}}|{{NADA}}", p, auto, baleiroLatex, nil)
	if out != `\mbox{}|\mbox{}` {
		t.Errorf("unha variable baleira en LaTeX ten que dar \\mbox{}: %q", out)
	}

	// Un grupo LaTeX normal non é unha variable.
	out, _ = substituirVariables(`\frac{{a+b}}{2}`, p, auto, baleiroLatex, nil)
	if out != `\frac{{a+b}}{2}` {
		t.Errorf("non debería tocar código LaTeX normal: %q", out)
	}

	// Camiño LaTeX: o VALOR dunha variable con caracteres especiais escápase
	// (senón "unidade_2" -> "Missing $ inserted"). A maqueta non se toca.
	out, _ = substituirVariables(`\bfseries {{TITULO}} & {{DATA}}`, &PlantillaLatex{
		Variables: []PlantillaVar{{Nome: "TITULO", Valor: "unidade_2"}},
	}, map[string]string{"DATA": "01/09/2026"}, baleiroLatex, EscapeLatexProse)
	if want := `\bfseries unidade\_2 & 01/09/2026`; out != want {
		t.Errorf("o valor da variable debía ir escapado:\n  obtido: %q\n  esperado: %q", out, want)
	}
}

// componInforme é o atallo das probas de composición: preámbulo de Lua,
// corpo dado e as definicións do informe, coma fai latexInformeCrono.
func componInforme(corpo, engine string, p *PlantillaLatex, auto map[string]string) (string, []string, error) {
	return documentoLatexConPlantilla(preambuloInformeCrono(engine, nil, nil, p), partesInformeCrono{Corpo: corpo}, engine, p, auto, paquetesInformeCrono, bloqueInformeCrono(nil, engine, nil, p))
}

func TestDocumentoLatexConPlantillaFragmento(t *testing.T) {
	p := &PlantillaLatex{
		Nome:      "Cabeceira",
		Latex:     "\\begin{center}{{CENTRO}}\\end{center}\n\n{{CORPO}}\n",
		Variables: []PlantillaVar{{Nome: "CENTRO", Valor: "IES Xistral"}},
	}
	doc, _, err := componInforme("CORPO XERADO", "pdflatex", p, variablesAutomaticas("curso"))
	if err != nil {
		t.Fatalf("compoñer: %v", err)
	}
	if !strings.Contains(doc, `\documentclass`) || !strings.Contains(doc, `\begin{document}`) {
		t.Error("un fragmento ten que ir envolto no preámbulo do informe")
	}
	if !strings.Contains(doc, "IES Xistral") || !strings.Contains(doc, "CORPO XERADO") {
		t.Error("faltan a variable ou o corpo no documento composto")
	}
	if !strings.Contains(doc, `\usepackage{fancyhdr}`) {
		t.Error("o preámbulo do informe ten que traer fancyhdr para as cabeceiras dos fragmentos")
	}
	if strings.Contains(doc, marcadorCorpo) {
		t.Error("quedou un {{CORPO}} sen substituír")
	}
	// O fragmento vai DENTRO do documento, non antes do preámbulo.
	if strings.Index(doc, "IES Xistral") < strings.Index(doc, `\begin{document}`) {
		t.Error("a cabeceira do fragmento quedou antes de \\begin{document}")
	}
}

func TestDocumentoLatexConPlantillaCompleta(t *testing.T) {
	p := &PlantillaLatex{
		Nome: "Centro",
		Latex: `\documentclass[11pt]{article}
\usepackage[margin=3cm]{geometry}
\usepackage{graphicx}
\begin{document}
{{CORPO}}
\end{document}
`,
	}
	doc, _, err := componInforme("CORPO", "pdflatex", p, variablesAutomaticas("curso"))
	if err != nil {
		t.Fatalf("compoñer: %v", err)
	}
	if strings.Count(doc, `\documentclass`) != 1 {
		t.Error("unha plantilla completa manda: non se lle pode engadir outro \\documentclass")
	}
	if !strings.Contains(doc, `[margin=3cm]{geometry}`) {
		t.Error("perdéronse as marxes propias da plantilla")
	}
	if strings.Count(doc, `\usepackage{graphicx}`) != 1 {
		t.Error("graphicx xa estaba declarado: non se debe duplicar (Option clash)")
	}
	for _, agardado := range []string{`\usepackage{longtable}`, `\usepackage{titlesec}`, `\newcommand{\chip}`, `\definecolor{tf-blue}`} {
		if !strings.Contains(doc, agardado) {
			t.Errorf("falta o que precisa o informe: %s", agardado)
		}
	}
	if !strings.Contains(doc, `\DeclareUnicodeCharacter`) {
		t.Error("con pdflatex tamén hai que inxectar as declaracións Unicode")
	}
	// Todo o inxectado ten que quedar ANTES do \begin{document}.
	if strings.Index(doc, `\usepackage{longtable}`) > strings.Index(doc, `\begin{document}`) ||
		strings.Index(doc, `\newcommand{\chip}`) > strings.Index(doc, `\begin{document}`) {
		t.Error("o preámbulo inxectado quedou despois de \\begin{document}")
	}
}

// Cun motor Unicode (xelatex/lualatex) NON se pode inxectar inputenc: é
// erro duro nese motor. Debe ir fontspec no seu lugar.
func TestInxeccionSegundoMotor(t *testing.T) {
	base := "\\documentclass{article}\n\\begin{document}\n{{CORPO}}\n\\end{document}\n"
	p := &PlantillaLatex{Nome: "x", Latex: base}

	pdf, _, _ := componInforme("c", "pdflatex", p, nil)
	if !strings.Contains(pdf, `\usepackage[utf8]{inputenc}`) {
		t.Error("con pdflatex fai falla inputenc")
	}
	xe, _, _ := componInforme("c", "xelatex", p, nil)
	if strings.Contains(xe, "inputenc") {
		t.Error("con xelatex NON se pode inxectar inputenc")
	}
	if !strings.Contains(xe, `\usepackage{fontspec}`) {
		t.Error("con xelatex fai falla fontspec")
	}
}

func TestPlantillaSenCorpoDaErro(t *testing.T) {
	if _, _, err := componInforme("c", "pdflatex", &PlantillaLatex{Nome: "x", Latex: "sen marcador"}, nil); err == nil {
		t.Error("unha plantilla sen {{CORPO}} ten que dar erro")
	}
}

// Sen plantilla, o documento é o de sempre: preámbulo + corpo.
func TestDocumentoLatexSenPlantilla(t *testing.T) {
	doc, avisos, err := componInforme("CORPO", "pdflatex", nil, nil)
	if err != nil || len(avisos) != 0 {
		t.Fatalf("erro=%v avisos=%v", err, avisos)
	}
	if !strings.HasSuffix(doc, "CORPO"+latexPostamble) || !strings.Contains(doc, `\begin{document}`) {
		t.Errorf("composición sen plantilla incorrecta:\n%s", doc)
	}
}

func TestDocumentoMarkdownConPlantilla(t *testing.T) {
	corpo := "Informe 1"
	if out, _ := documentoMarkdownConPlantilla(corpo, nil, nil); out != corpo {
		t.Errorf("sen plantilla o corpo non se toca: %q", out)
	}
	p := &PlantillaLatex{Latex: "{{CORPO}}"} // sen parte Markdown
	if out, _ := documentoMarkdownConPlantilla(corpo, p, nil); out != corpo {
		t.Errorf("sen parte Markdown o corpo non se toca: %q", out)
	}
	p.Markdown = "# {{TITULO}}\n\n{{CORPO}}\n"
	out, _ := documentoMarkdownConPlantilla(corpo, p, map[string]string{"TITULO": "Cronograma 25-26"})
	if out != "# Cronograma 25-26\n\nInforme 1\n" {
		t.Errorf("composición Markdown incorrecta: %q", out)
	}
}

// Unha variable da plantilla SEN valor non pisa a automática (o nome do
// centro do propio curso); con valor, si a pisa.
func TestVariableBaleiraNonPisaAutomatica(t *testing.T) {
	auto := map[string]string{"CENTRO": "IES do curso"}
	out, _ := substituirVariables("{{CENTRO}}", &PlantillaLatex{Variables: []PlantillaVar{{Nome: "CENTRO", Valor: ""}}}, auto, "", nil)
	if out != "IES do curso" {
		t.Errorf("valor baleiro debía usar o automático: %q", out)
	}
	out, _ = substituirVariables("{{CENTRO}}", &PlantillaLatex{Variables: []PlantillaVar{{Nome: "CENTRO", Valor: "IES da plantilla"}}}, auto, "", nil)
	if out != "IES da plantilla" {
		t.Errorf("a plantilla con valor debía gañar: %q", out)
	}
}

func TestNormalizarNomeVar(t *testing.T) {
	casos := map[string]string{
		"centro educativo": "CENTRO_EDUCATIVO",
		"  Curso  ":        "CURSO",
		"n.º de orzamento": "N_DE_ORZAMENTO",
		"{{raro}}":         "RARO",
		"":                 "",
	}
	for entrada, esperado := range casos {
		if got := normalizarNomeVar(entrada); got != esperado {
			t.Errorf("normalizarNomeVar(%q) = %q, esperaba %q", entrada, got, esperado)
		}
	}
}

func TestParsearPlantillaIA(t *testing.T) {
	raw := "Aquí tes a plantilla:\n```json\n{\"nome\":\"Circular\",\"latex\":\"\\\\section{Ola}\\n{{CORPO}}\",\"markdown\":\"\",\"variables\":[{\"nome\":\"centro\",\"etiqueta\":\"Centro\",\"valor\":\"\"}]}\n```\nEspero que che sirva!"
	p, err := parsearPlantillaIA(raw)
	if err != nil {
		t.Fatalf("parsear: %v", err)
	}
	if p.Nome != "Circular" || !strings.Contains(p.Latex, marcadorCorpo) {
		t.Errorf("plantilla mal parseada: %+v", p)
	}
	if len(p.Variables) != 1 || p.Variables[0].Nome != "CENTRO" {
		t.Errorf("variables mal normalizadas: %+v", p.Variables)
	}

	// Sen {{CORPO}}: recupérase engadíndoo en vez de tirar a resposta.
	p, err = parsearPlantillaIA(`{"nome":"X","latex":"\\documentclass{article}\n\\begin{document}\nOla\n\\end{document}"}`)
	if err != nil {
		t.Fatalf("parsear sen marcador: %v", err)
	}
	if !strings.Contains(p.Latex, marcadorCorpo) {
		t.Errorf("debería engadirse {{CORPO}}: %q", p.Latex)
	}
	if strings.Index(p.Latex, marcadorCorpo) > strings.Index(p.Latex, `\end{document}`) {
		t.Error("{{CORPO}} ten que quedar DENTRO do documento")
	}

	if _, err := parsearPlantillaIA("non hai json ningún aquí"); err == nil {
		t.Error("esperaba erro cunha resposta sen JSON")
	}
}

func TestTextoVisibleHTML(t *testing.T) {
	html := `<html><head><style>p{color:red}</style><script>var x=1</script></head>
	<body><h1>Instituto</h1><p>Modelo de exame</p></body></html>`
	out := textoVisibleHTML(html)
	if !strings.Contains(out, "Instituto") || !strings.Contains(out, "Modelo de exame") {
		t.Errorf("perdeuse texto visible: %q", out)
	}
	if strings.Contains(out, "color:red") || strings.Contains(out, "var x") {
		t.Errorf("style/script non deberían aparecer: %q", out)
	}
}

// writeFileParaTest crea un ficheiro baleiro (as probas de PorFicheiro
// precisan que a ruta exista de verdade, ver podarPorFicheiro).
func writeFileParaTest(path string) error {
	return os.WriteFile(path, []byte("<EX></EX>\n"), 0o644)
}

// TestPrevisualizarPlantillasDeFabrica compila DE VERDADE cada plantilla de
// exemplo co cronograma de mostra - é a única forma de saber que o LaTeX que
// Lua sementa non peta na cara de quen preme "Vista previa" a primeira vez.
func TestPrevisualizarPlantillasDeFabrica(t *testing.T) {
	if !commandExists("pdflatex") {
		t.Skip("sen pdflatex neste equipo")
	}
	configIllado(t)
	a := &App{}
	for _, p := range plantillasDeFabrica() {
		t.Run(p.Nome, func(t *testing.T) {
			res, err := a.PrevisualizarPlantilla(p)
			if err != nil {
				t.Fatalf("non compilou: %v\nlog:\n%s", err, res.Log)
			}
			if res.PDFBase64 == "" {
				t.Error("compilou sen devolver PDF")
			}
			if strings.Contains(res.LatexSource, marcadorCorpo) || strings.Contains(res.LatexSource, marcadorPortada) || strings.Contains(res.LatexSource, marcadorSinatura) {
				t.Error("quedou un marcador ({{CORPO}}, {{PORTADA}} ou {{SINATURA}}) sen substituír no .tex final")
			}
			if !strings.Contains(res.LatexSource, "Centro de exemplo") {
				t.Error("o informe de mostra non chegou ao .tex final")
			}
		})
	}
}

// TestXerarInformeConPlantillaActiva é a proba de integración de todo o
// camiño: escoller unha plantilla e premer "Xerar" ten que dar un PDF co
// informe DENTRO da maqueta (pdflatex real, coma na app).
func TestXerarInformeConPlantillaActiva(t *testing.T) {
	if !commandExists("pdflatex") {
		t.Skip("sen pdflatex neste equipo")
	}
	configIllado(t)
	a := &App{}
	a.startup(context.Background())

	p, err := a.GardarPlantilla(PlantillaLatex{
		Nome: "Cabeceira do centro",
		Latex: `\pagestyle{fancy}\fancyhf{}
\fancyhead[L]{ {{CENTRO}} }\fancyhead[R]{ Tel. {{TELEFONO}} }
\fancyfoot[L]{ {{ORIENTADOR}} }\fancyfoot[R]{\thepage}
{{CORPO}}
`,
		Markdown:  "# {{CENTRO}}\n\n{{CORPO}}\n",
		Variables: []PlantillaVar{{Nome: "TELEFONO", Valor: "986 000 000"}},
	})
	if err != nil {
		t.Fatalf("gardar plantilla: %v", err)
	}

	plan := planDemoCrono()
	req := XerarInformeCronogramaRequest{Plan: plan, NomeBase: "cronograma-2025-2026"}

	// Sen plantilla activa: o informe sae coma sempre.
	sen, err := a.XerarInformeCronograma(req)
	if err != nil {
		t.Fatalf("xerar sen plantilla: %v\nlog:\n%s", err, sen.Log)
	}
	if strings.Contains(sen.LatexSource, "986 000 000") {
		t.Error("sen plantilla activa non debería aparecer a cabeceira")
	}

	if err := a.EscollerPlantilla(p.ID, ""); err != nil {
		t.Fatalf("escoller: %v", err)
	}
	con, err := a.XerarInformeCronograma(req)
	if err != nil {
		t.Fatalf("xerar con plantilla: %v\nlog:\n%s", err, con.Log)
	}
	pdf, err := base64.StdEncoding.DecodeString(con.PDFBase64)
	if err != nil || !bytes.HasPrefix(pdf, []byte("%PDF")) {
		t.Fatalf("non saíu un PDF válido (%v)", err)
	}
	// CENTRO e ORIENTADOR veñen do propio curso; TELEFONO, da plantilla.
	for _, agardado := range []string{"Centro de exemplo", "Nome e apelidos", "986 000 000", "ACOLLIDA DO ALUMNADO", "Acollida do alumnado"} {
		if strings.EqualFold(agardado, "ACOLLIDA DO ALUMNADO") {
			continue
		}
		if !strings.Contains(con.LatexSource, agardado) {
			t.Errorf("falta %q no .tex final", agardado)
		}
	}
	if strings.Contains(con.LatexSource, marcadorCorpo) {
		t.Error("quedou {{CORPO}} sen substituír")
	}

	// O resultado queda gardado para a xanela desacoplada, coa plantilla.
	r, err := a.resultadoActual()
	if err != nil || r.Plantilla == nil || r.Plantilla.ID != p.ID || r.Req.NomeBase != "cronograma-2025-2026" {
		t.Errorf("o resultado publicado non garda a plantilla/nome: %+v (err %v)", r, err)
	}
	if snap := a.GetEstadoResultado(); !snap.HasPDF || snap.Version != 2 {
		t.Errorf("snapshot inesperado: %+v", snap)
	}

	// Markdown coa plantilla.
	md, _ := markdownInformeConPlantilla(req, r.Plantilla, "md", t.TempDir())
	if !strings.HasPrefix(md, "# Centro de exemplo") {
		t.Errorf("falta a cabeceira Markdown da plantilla:\n%.200s", md)
	}
	if strings.Contains(md, marcadorCorpo) {
		t.Error("quedou {{CORPO}} sen substituír no Markdown")
	}
}

// TestRepararBarrasLatex cobre o fallo real que rompeu unha plantilla
// xerada dende unha URL: o modelo sub-escapou os \\ do JSON e chegaron coma
// unha soa barra ("\[0.2em]"), que en LaTeX abre modo matemático.
func TestRepararBarrasLatex(t *testing.T) {
	casos := []struct{ entrada, esperado string }{
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
	}
	for _, c := range casos {
		if got := repararBarrasLatex(c.entrada); got != c.esperado {
			t.Errorf("repararBarrasLatex(%q)\n  = %q\n  esperaba %q", c.entrada, got, c.esperado)
		}
	}
}

// TestMotorDeducidoDaIA: unha plantilla con fontspec só compila con
// xelatex/lualatex. Se a IA non o marca no campo "motor", dedúcese do
// código - senón, quen teña pdflatex en Opcións vería fallar a plantilla
// cun erro que non menciona a causa.
func TestMotorDeducidoDaIA(t *testing.T) {
	con, err := parsearPlantillaIA(`{"nome":"X","latex":"\\documentclass{article}\n\\usepackage{fontspec}\n\\begin{document}\n{{CORPO}}\n\\end{document}"}`)
	if err != nil {
		t.Fatalf("parsear: %v", err)
	}
	if con.Motor != "xelatex" {
		t.Errorf("con fontspec o motor debería ser xelatex, foi %q", con.Motor)
	}

	sen, err := parsearPlantillaIA(`{"nome":"X","latex":"\\section{Ola}\n{{CORPO}}"}`)
	if err != nil {
		t.Fatalf("parsear: %v", err)
	}
	if sen.Motor != "" {
		t.Errorf("sen fontspec non se debe forzar motor, foi %q", sen.Motor)
	}

	// Un motor que xa veña da IA mándase el.
	explicito, err := parsearPlantillaIA(`{"nome":"X","motor":"lualatex","latex":"\\usepackage{fontspec}\n{{CORPO}}"}`)
	if err != nil {
		t.Fatalf("parsear: %v", err)
	}
	if explicito.Motor != "lualatex" {
		t.Errorf("o motor explícito da IA non se pode pisar, foi %q", explicito.Motor)
	}
}

// TestPlantillasDeFabricaNovasChegorAQuenXaTiñaLua é a proba de que se
// poden REPARTIR plantillas: unha que se engada a plantillas_fabrica.go
// nunha versión nova ten que aparecerlle tamén a quen xa tiña Lua
// instalado (biblioteca xa sementada), sen pisar o que teña editado nin
// resucitar o que borrase.
func TestPlantillasDeFabricaNovasChegorAQuenXaTiñaLua(t *testing.T) {
	configIllado(t)
	a := &App{}

	// Estado "instalación vella": xa sementada, pero só cunha das de
	// fábrica, e ademais EDITADA polo profesorado.
	fabrica := plantillasDeFabrica()
	editada := fabrica[0]
	editada.Latex = "O MEU DESEÑO PROPIO\n{{CORPO}}\n"
	editada.Nome = "O meu exame"
	if err := saveAlmacenPlantillas(almacenPlantillas{
		Plantillas: []PlantillaLatex{editada},
		Sementado:  true,
	}); err != nil {
		t.Fatal(err)
	}

	estado, err := a.ListarPlantillas()
	if err != nil {
		t.Fatalf("listar: %v", err)
	}
	// As tres de fábrica MÁIS a copia propia que saca repararFabricaEditadas
	// da que estaba editada en sitio. Antes eran só tres, porque o traballo
	// do profesorado ocupaba unha das IDs de fábrica - que é precisamente o
	// que xa non pasa (ver repararFabricaEditadas, plantillas.go).
	if len(estado.Plantillas) != len(fabrica)+1 {
		t.Fatalf("esperaba %d plantillas (as %d de fábrica + a copia do traballo editado), atopei %d",
			len(fabrica)+1, len(fabrica), len(estado.Plantillas))
	}
	// A garantía de sempre segue en pé: o traballo do profesorado non se
	// perde nunha actualización. O que cambia é onde vive - nunha plantilla
	// propia, non pisando a de fábrica.
	traballo := false
	for _, p := range estado.Plantillas {
		if p.Nome == "O meu exame" && strings.Contains(p.Latex, "O MEU DESEÑO PROPIO") {
			traballo = true
			if p.DeFabrica {
				t.Error("o traballo do profesorado non pode quedar marcado coma de fábrica")
			}
			if p.ID == editada.ID {
				t.Error("tiña que mudar de ID para deixar libre a de fábrica")
			}
		}
	}
	if !traballo {
		t.Error("unha actualización non pode facer desaparecer unha plantilla xa editada")
	}

	// Borrar unha de fábrica ten que ser definitivo.
	borrada := fabrica[1].ID
	if err := a.EliminarPlantilla(borrada); err != nil {
		t.Fatalf("eliminar: %v", err)
	}
	estado, err = a.ListarPlantillas() // "reinicio": volve pasar por sementarSeFai
	if err != nil {
		t.Fatalf("listar tras eliminar: %v", err)
	}
	for _, p := range estado.Plantillas {
		if p.ID == borrada {
			t.Errorf("unha plantilla de fábrica borrada non pode volver soa: %q", p.Nome)
		}
	}
	// -1 pola borrada, +1 pola copia propia da reparación (que segue aí:
	// borrar unha de fábrica non pode levar por diante o traballo propio).
	if len(estado.Plantillas) != len(fabrica) {
		t.Errorf("esperaba %d tras eliminar, atopei %d", len(fabrica), len(estado.Plantillas))
	}
}

// TestEditarDeFabricaBifurca cobre o arranxo de bifurcarDeFabrica: gardar
// por riba dunha plantilla de fábrica NON a pisa. Antes si, e iso deixaba a
// ID de fábrica ocupada para sempre (sementarSeFai só repón as que faltan),
// así que esa plantilla xa nunca recibía as melloras dunha versión nova de
// Lua - e ademais gardaba datos privados baixo unha entrada marcada coma
// "de exemplo".
func TestEditarDeFabricaBifurca(t *testing.T) {
	configIllado(t)
	a := &App{}

	estado, err := a.ListarPlantillas() // sementa as de fábrica
	if err != nil {
		t.Fatal(err)
	}
	orixinal := estado.Plantillas[0]
	if !orixinal.DeFabrica {
		t.Fatalf("esperaba unha de fábrica, dáme %+v", orixinal.Nome)
	}
	latexFabrica := orixinal.Latex

	// O documento e a escolla apuntan á de fábrica, coma quen leva tempo
	// traballando con ela.
	doc := filepath.Join(t.TempDir(), "meu.json")
	if err := os.WriteFile(doc, []byte("ola"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := a.EscollerPlantilla(orixinal.ID, doc); err != nil {
		t.Fatal(err)
	}

	// Editar e gardar, coma quen lle mete os datos do seu centro.
	editada := orixinal
	editada.Latex = "DATOS PRIVADOS DO MEU CENTRO\n{{CORPO}}\n"
	gardada, err := a.GardarPlantilla(editada)
	if err != nil {
		t.Fatalf("gardar: %v", err)
	}

	if gardada.ID == orixinal.ID {
		t.Error("gardar sobre unha de fábrica ten que crear unha copia, non pisala")
	}
	if gardada.DeFabrica {
		t.Error("a copia do profesorado non pode quedar marcada coma de fábrica")
	}

	estado, err = a.ListarPlantillas()
	if err != nil {
		t.Fatal(err)
	}
	var fabricaAgora, miña *PlantillaLatex
	for i := range estado.Plantillas {
		switch estado.Plantillas[i].ID {
		case orixinal.ID:
			fabricaAgora = &estado.Plantillas[i]
		case gardada.ID:
			miña = &estado.Plantillas[i]
		}
	}
	if fabricaAgora == nil {
		t.Fatal("a de fábrica ten que sobrevivir á edición")
	}
	if fabricaAgora.Latex != latexFabrica {
		t.Error("a de fábrica quedou modificada; tiña que seguir intacta")
	}
	if strings.Contains(fabricaAgora.Latex, "DATOS PRIVADOS") {
		t.Error("datos privados filtrados na plantilla marcada coma de exemplo")
	}
	if miña == nil || !strings.Contains(miña.Latex, "DATOS PRIVADOS DO MEU CENTRO") {
		t.Fatal("a copia ten que levar o que se escribiu")
	}

	// A bifurcación herda a escolla: o traballo segue a aplicarse ao mesmo
	// documento, que é o que espera quen acaba de premer "Gardar".
	if estado.Activa != gardada.ID {
		t.Errorf("a activa tiña que pasar á copia, quedou en %q", estado.Activa)
	}
	id, err := a.PlantillaParaFicheiro(doc)
	if err != nil {
		t.Fatal(err)
	}
	if id != gardada.ID {
		t.Errorf("o documento tiña que quedar apuntando á copia, apunta a %q", id)
	}

	// Editar OUTRA VEZ a copia xa non bifurca: só a primeira vez.
	miña.Latex = "SEGUNDA VOLTA\n{{CORPO}}\n"
	outra, err := a.GardarPlantilla(*miña)
	if err != nil {
		t.Fatal(err)
	}
	if outra.ID != gardada.ID {
		t.Error("editar unha plantilla propia non pode seguir creando copias")
	}
}

// TestRepararFabricaEditadaAnterior cobre a reparación retroactiva: un
// plantillas.json que xa vén estropeado dunha versión anterior de Lua
// (unha de fábrica pisada en sitio) arránxase só ao cargalo, sen perder
// nada e sen pedirlle nada ao profesorado.
func TestRepararFabricaEditadaAnterior(t *testing.T) {
	configIllado(t)
	a := &App{}

	fabrica := plantillasDeFabrica()
	orixinal := fabrica[0]

	// Estado herdado: a de fábrica pisada, aínda marcada DeFabrica, e todo
	// (activa + documento) apuntando a ela.
	estropeada := orixinal
	estropeada.Nome = "Exame do meu centro"
	estropeada.Latex = "CENTRO PRIVADO\n{{CORPO}}\n"
	doc := "/tmp/vello.json"
	if err := saveAlmacenPlantillas(almacenPlantillas{
		Plantillas:  append([]PlantillaLatex{estropeada}, fabrica[1:]...),
		Activa:      orixinal.ID,
		PorFicheiro: map[string]string{doc: orixinal.ID},
		Sementado:   true,
	}); err != nil {
		t.Fatal(err)
	}

	estado, err := a.ListarPlantillas() // pasa por repararFabricaEditadas
	if err != nil {
		t.Fatal(err)
	}

	var deFabrica, propia *PlantillaLatex
	for i := range estado.Plantillas {
		switch {
		case estado.Plantillas[i].ID == orixinal.ID:
			deFabrica = &estado.Plantillas[i]
		case estado.Plantillas[i].Nome == "Exame do meu centro":
			propia = &estado.Plantillas[i]
		}
	}
	if deFabrica == nil {
		t.Fatal("a de fábrica ten que seguir existindo coa súa ID")
	}
	if deFabrica.Latex != orixinal.Latex || deFabrica.Nome != orixinal.Nome {
		t.Errorf("a de fábrica tiña que volver ao orixinal, quedou en %q", deFabrica.Nome)
	}
	if strings.Contains(deFabrica.Latex, "CENTRO PRIVADO") {
		t.Error("datos privados aínda baixo a entrada marcada de fábrica")
	}
	if propia == nil {
		t.Fatal("o traballo editado tiña que sobrevivir nunha plantilla propia")
	}
	if propia.DeFabrica || !strings.Contains(propia.Latex, "CENTRO PRIVADO") {
		t.Errorf("a copia propia non conserva o traballo: %+v", propia.Nome)
	}
	if estado.Activa != propia.ID {
		t.Error("a activa tiña que pasar á copia propia")
	}
	id, err := a.PlantillaParaFicheiro(doc)
	if err != nil {
		t.Fatal(err)
	}
	if id != propia.ID {
		t.Error("o documento tiña que quedar coa copia propia")
	}

	// Idempotente: cargar outra vez non pode seguir creando copias.
	antes := len(estado.Plantillas)
	estado, err = a.ListarPlantillas()
	if err != nil {
		t.Fatal(err)
	}
	if len(estado.Plantillas) != antes {
		t.Errorf("a reparación repetiuse: %d -> %d plantillas", antes, len(estado.Plantillas))
	}
}

// TestRetirarFabricaObsoleta: as plantillas de exemplo dos tempos do
// xerador de exames desaparecen se ninguén as tocou, pero unha que ten
// cambios ou imaxes queda (xa non coma "de exemplo") e a activa non queda
// apuntando a unha plantilla que xa non existe.
func TestRetirarFabricaObsoleta(t *testing.T) {
	configIllado(t)
	a := &App{}

	exame := PlantillaLatex{ID: "fabrica-exame", Nome: "Exame de instituto", DeFabrica: true, Latex: exameLatexVello}
	boletin := PlantillaLatex{ID: "fabrica-boletin", Nome: "Boletín", DeFabrica: true, Latex: "O MEU BOLETÍN\n{{CORPO}}\n"}
	propia := PlantillaLatex{ID: "propia", Nome: "IES Ramón Caamaño", Latex: "{{CORPO}}"}
	if err := saveAlmacenPlantillas(almacenPlantillas{
		Plantillas: []PlantillaLatex{exame, boletin, propia},
		Activa:     "fabrica-exame",
		Sementado:  true,
	}); err != nil {
		t.Fatal(err)
	}

	estado, err := a.ListarPlantillas()
	if err != nil {
		t.Fatal(err)
	}
	vistas := map[string]PlantillaLatex{}
	for _, p := range estado.Plantillas {
		vistas[p.ID] = p
	}
	if _, hai := vistas["fabrica-exame"]; hai {
		t.Error("a plantilla de exame sen tocar tiña que retirarse")
	}
	if b, hai := vistas["fabrica-boletin"]; !hai || b.DeFabrica {
		t.Errorf("unha plantilla vella con cambios queda, pero xa non coma de fábrica: %+v", b)
	}
	if _, hai := vistas["propia"]; !hai {
		t.Error("as plantillas propias non se tocan")
	}
	if estado.Activa != "" {
		t.Errorf("a activa apuntaba a unha retirada, ten que quedar ningunha: %q", estado.Activa)
	}
	for _, p := range plantillasDeFabrica() {
		if _, hai := vistas[p.ID]; !hai {
			t.Errorf("faltan as plantillas de fábrica novas: %s", p.ID)
		}
	}
}

// exameLatexVello é o LaTeX orixinal da plantilla "Exame de instituto" das
// versións vellas: só o hash (fabricaObsoleta) importa para recoñecela.
const exameLatexVello = `\begin{center}
  {\Large\bfseries {{CENTRO}} }\\[2pt]
  {\large {{MATERIA}} \textendash{} {{CURSO}} }\\[2pt]
  {{AVALIACION}}
\end{center}

\vspace{2mm}
\noindent Nome e apelidos: \hrulefill\hspace{4mm} Data: {{DATA}}
\vspace{2mm}

\hrule
\vspace{6mm}

{{CORPO}}
`

// pngMinimo é un PNG de 1x1 píxeles: abonda para probar que o logotipo dunha
// plantilla chega ao directorio de compilación.
const pngMinimo = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII="

// TestXerarInformeConPlantillaCompletaDeCentro é o caso real dun centro: unha
// plantilla COMPLETA (\documentclass propio, marxes, fancyhdr, logotipo na
// cabeceira e variables propias) ten que envolver o informe sen que se
// rompan nin a plantilla nin o informe (táboa do cronograma, títulos...), con
// pdflatex e con xelatex.
func TestXerarInformeConPlantillaCompletaDeCentro(t *testing.T) {
	if !commandExists("pdflatex") {
		t.Skip("sen pdflatex neste equipo")
	}
	configIllado(t)
	a := &App{}
	a.startup(context.Background())

	p, err := a.GardarPlantilla(PlantillaLatex{
		Nome: "IES de proba",
		Latex: `\documentclass[letterpaper,10pt]{article}
\usepackage[top=1in,bottom=2in,left=1in,right=1in]{geometry}
\usepackage{graphicx}
\usepackage{fancyhdr}
\pagestyle{fancy}
\fancyhf{}
\fancyhead[L]{\bfseries {{DEPARTAMENTO}}}
\fancyhead[R]{\includegraphics[height=1.5cm]{logotipo.png}}
\fancyfoot[L]{\footnotesize\itshape {{PE}}}
\fancyfoot[C]{\thepage}
\setlength{\headheight}{45pt}
\begin{document}
{{CORPO}}
\end{document}
`,
		Variables: []PlantillaVar{{Nome: "PE", Valor: "Orientadora: Ana Pérez, tel. 986 111 222"}},
	})
	if err != nil {
		t.Fatalf("gardar: %v", err)
	}
	if _, err := a.SubirImaxePlantilla(SubirImaxePlantillaRequest{ID: p.ID, FileName: "logotipo.png", DataB64: pngMinimo}); err != nil {
		t.Fatalf("subir logotipo: %v", err)
	}
	if err := a.EscollerPlantilla(p.ID, ""); err != nil {
		t.Fatal(err)
	}

	for _, motor := range []string{"pdflatex", "xelatex"} {
		if !commandExists(motor) {
			continue
		}
		t.Run(motor, func(t *testing.T) {
			a.settings.LatexEngine = motor
			res, err := a.XerarInformeCronograma(XerarInformeCronogramaRequest{Plan: planDemoCrono(), NomeBase: "x"})
			if err != nil {
				t.Fatalf("non compilou: %v\nlog:\n%s", err, res.Log)
			}
			if res.PDFBase64 == "" || len(res.PageImages) < 4 {
				t.Errorf("PDF ou páxinas inesperados (%d páxinas)", len(res.PageImages))
			}
			for _, agardado := range []string{"Departamento de exemplo", "Ana Pérez, tel. 986 111 222", `\chip{`} {
				if !strings.Contains(res.LatexSource, agardado) {
					t.Errorf("falta %q no .tex final", agardado)
				}
			}
			if strings.Count(res.LatexSource, `\documentclass`) != 1 {
				t.Error("máis dun \\documentclass")
			}
		})
	}
}
