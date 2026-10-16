package main

import (
	"archive/zip"
	"bytes"
	"edicion/latex"
	"encoding/base64"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func plantillaConDeseno(d Deseno) *PlantillaLatex {
	return &PlantillaLatex{Nome: "deseño", Latex: "{{CORPO}}", Deseno: &d}
}

// TestPdffontsCombinacion (AC1): coa combinación Clásica o PDF leva Termes no
// corpo e nos títulos, Heros nas etiquetas e Termes Math nas fórmulas; nada de
// Latin Modern Sans (o que saía antes nos títulos).
func TestPdffontsCombinacion(t *testing.T) {
	if !commandExists("xelatex") || !commandExists("pdffonts") {
		t.Skip("fan falta xelatex e pdffonts")
	}
	dir := t.TempDir()
	res, err := compilarInformeCrono(planMostraCrono(), "xelatex", nil, dir, plantillaConDeseno(Deseno{Letra: "clasica"}), "")
	if err != nil {
		t.Fatalf("compilar: %v\n%s", err, res.Log)
	}
	pdf := filepath.Join(dir, "saida.pdf")
	if err := os.WriteFile(pdf, res.PDF, 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command("pdffonts", pdf).Output()
	if err != nil {
		t.Fatal(err)
	}
	fontes := string(out)
	for _, want := range []string{"TeXGyreTermes-Regular", "TeXGyreTermes-Bold", "TeXGyreHeros"} {
		if !strings.Contains(fontes, want) {
			t.Errorf("falta %s no PDF:\n%s", want, fontes)
		}
	}
	for _, non := range []string{"LMSans", "OpenSans"} {
		if strings.Contains(fontes, non) {
			t.Errorf("o PDF non debería levar %s:\n%s", non, fontes)
		}
	}
}

// TestTamanoNormalsize (AC2): o tamaño cambia \normalsize (scrextend) e os
// títulos medran na mesma proporción.
func TestTamanoNormalsize(t *testing.T) {
	tex, _, err := latexInformeCrono(planMostraCrono(), "xelatex", nil, t.TempDir(), time.Date(2026, 10, 16, 10, 0, 0, 0, time.UTC), plantillaConDeseno(Deseno{Tamano: 13.2}), "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(tex, `\usepackage[fontsize=13.2pt]{scrextend}`) {
		t.Error("o tamaño ten que ir en scrextend")
	}
	// 20pt * 13.2/11 = 24pt
	if !strings.Contains(tex, `\titleformat{\section}{\sffamily\fontsize{24}{28.8}`) {
		t.Errorf("os títulos teñen que escalar co corpo")
	}
	// Xa non se cambia o tamaño tras \begin{document} (non chegaba a \normalsize).
	corpo := tex[strings.Index(tex, `\begin{document}`):]
	if strings.Contains(corpo, `\fontsize{13.2pt}`) {
		t.Error("o tamaño non vai no corpo co preámbulo de Lúa")
	}
}

// TestTamanoCompila: o tamaño e as letras compilan de verdade.
func TestTamanoCompila(t *testing.T) {
	if !commandExists("xelatex") {
		t.Skip("xelatex non está instalado")
	}
	for _, d := range []Deseno{{Tamano: 13, Letra: "lexible", Interlinado: 1.3}, {Letra: "moderna", Etiquetas: "contorno"}, {Letra: "academica", AforrarPapel: true}, {Letra: "elegante", LecturaFacil: true, BrancoNegro: true}} {
		res, err := compilarInformeCrono(planMostraCrono(), "xelatex", nil, t.TempDir(), plantillaConDeseno(d), "")
		if err != nil {
			t.Errorf("deseño %+v non compila: %v\n%s", d, err, res.Log)
		}
	}
}

// TestDesenoBrancoNegro (AC4): en branco e negro ningunha cor ten saturación.
func TestDesenoBrancoNegro(t *testing.T) {
	r := resolverDeseno(plantillaConDeseno(Deseno{BrancoNegro: true, CorPrincipal: "D97706"}), nil)
	for _, liña := range strings.Split(r.latexCores(), `\definecolor`) {
		i := strings.Index(liña, "{HTML}{")
		if i < 0 {
			continue
		}
		hex := liña[i+7 : i+13]
		if hex[0:2] != hex[2:4] || hex[2:4] != hex[4:6] {
			t.Errorf("cor con saturación en branco e negro: %s (%s)", hex, liña)
		}
	}
}

// TestDesenoAforrar: aforrar papel reduce marxe e espazos sen tocar os valores gardados.
func TestDesenoAforrar(t *testing.T) {
	d := Deseno{Marxe: 25, AforrarPapel: true}
	r := resolverDeseno(plantillaConDeseno(d), nil)
	if r.marxe != 14 || r.parrafo > 0.25 {
		t.Errorf("aforrar: marxe %v, parágrafo %v", r.marxe, r.parrafo)
	}
	if d.Marxe != 25 {
		t.Error("o interruptor non cambia o deseño gardado")
	}
	lf := resolverDeseno(plantillaConDeseno(Deseno{LecturaFacil: true}), nil)
	if lf.tamano < 13 || !strings.Contains(lf.latexTexto(), `\raggedright`) {
		t.Errorf("lectura fácil: %v", lf.tamano)
	}
}

// TestCorrectorContraste (AC5): unha cor principal clara escurécese para os
// rótulos e avísase.
func TestCorrectorContraste(t *testing.T) {
	r := resolverDeseno(plantillaConDeseno(Deseno{CorPrincipal: "FDE68A"}), nil)
	if contrasteCor(r.escura, "FFFFFF") < contrasteMinimoTextos {
		t.Errorf("os rótulos seguen sen contraste: %s", r.escura)
	}
	if len(r.arranxos) == 0 {
		t.Error("o arranxo ten que contarse")
	}
	dd := compoñerDeseno("xelatex", nil, nil, plantillaConDeseno(Deseno{CorPrincipal: "FDE68A"}))
	if !strings.Contains(strings.Join(dd.Avisos, "\n"), "Arranxei") {
		t.Errorf("o aviso ten que dicir que se arranxou: %v", dd.Avisos)
	}
	if bo := resolverDeseno(nil, nil); len(bo.arranxos) != 0 {
		t.Errorf("o deseño de Lúa non precisa arranxos: %v", bo.arranxos)
	}
}

// TestElementoLatexPropio: o LaTeX propio dun elemento substitúe só ese elemento.
func TestElementoLatexPropio(t *testing.T) {
	propio := `\newcommand{\tarefa}[2]{\par\textbf{PROPIO #2}\par}`
	r := resolverDeseno(plantillaConDeseno(Deseno{Elementos: map[string]ElementoDeseno{"tarefas": {Latex: propio}}}), nil)
	e := r.latexEstilo()
	if !strings.Contains(e, propio) || strings.Count(e, `\newcommand{\tarefa}`) != 1 {
		t.Error("o LaTeX propio ten que substituír o xerado")
	}
	if !strings.Contains(e, `\newcommand{\chip}`) {
		t.Error("o resto dos elementos segue xerándose")
	}
}

// TestEstiloDesenoPorDefecto: o deseño de Lúa leva todos os macros que usa o
// contido, e o Estilo libre segue mandando.
func TestEstiloDesenoPorDefecto(t *testing.T) {
	for _, m := range []string{`\newcommand{\chip}`, `\newcommand{\mesbadge}`, `\newcommand{\medirmes}`, `\newcommand{\mesvacio}`, `\newenvironment{trimestre}`,
		`\newcommand{\tarefa}`, `\newcommand{\actacab}`, `\newcommand{\actadato}`, `\newcommand{\actasecc}`, `\newcommand{\actamateria}`,
		`\newcommand{\actacampo}`, `\newcommand{\actaprogreso}`, `\newcommand{\actasinaturas}`, `\titleformat{\section}`} {
		if !strings.Contains(estiloPorDefectoLatex, m) {
			t.Errorf("o deseño de Lúa non define %s", m)
		}
	}
	libre := &PlantillaLatex{Latex: "{{CORPO}}", Estilo: "% LIBRE\n", Deseno: &Deseno{Letra: "clasica"}}
	dd := compoñerDeseno("xelatex", nil, nil, libre)
	if !strings.Contains(dd.Bloque, "% LIBRE") || strings.Contains(dd.Bloque, `\newcommand{\chip}`) {
		t.Error("o Estilo libre substitúe o deseño xerado")
	}
}

// TestPlantillaCompletaAvisaLetra: unha plantilla completa con \setmainfont
// conserva a súa letra e avísase; sen ela, inxéctase a do deseño.
func TestPlantillaCompletaAvisaLetra(t *testing.T) {
	coLetra := &PlantillaLatex{Latex: "\\documentclass{article}\n\\usepackage{fontspec}\\setmainfont{Georgia}\n\\begin{document}{{CORPO}}\\end{document}", Deseno: &Deseno{Letra: "clasica"}}
	dd := compoñerDeseno("xelatex", nil, nil, coLetra)
	if strings.Contains(dd.BloqueCompleta, `\setmainfont{texgyretermes}`) {
		t.Error("non se pisa a letra da plantilla")
	}
	if !strings.Contains(strings.Join(dd.Avisos, "\n"), "fixa a súa propia letra") {
		t.Errorf("hai que avisar: %v", dd.Avisos)
	}
	senLetra := &PlantillaLatex{Latex: "\\documentclass{article}\n\\begin{document}{{CORPO}}\\end{document}", Deseno: &Deseno{Letra: "clasica", Tamano: 12}}
	dd = compoñerDeseno("xelatex", nil, nil, senLetra)
	if !strings.Contains(dd.BloqueCompleta, `\setmainfont{texgyretermes}`) {
		t.Error("sen letra propia, a plantilla completa leva a do deseño")
	}
	if !strings.Contains(dd.Corpo, `\fontsize{12pt}`) {
		t.Errorf("o tamaño vai tras \\begin{document} nunha plantilla completa: %q", dd.Corpo)
	}
}

// TestAxustesTextoSenDeseno: sen deseño na plantilla, os Axustes do texto
// seguen mandando; con deseño, avísase de que non se aplican.
func TestAxustesTextoSenDeseno(t *testing.T) {
	tip := &TipografiaOpts{Fonte: "palatino", TamanoPt: 12}
	r := resolverDeseno(nil, tip)
	if r.letras.Corpo != "pagella" || r.tamano != 12 {
		t.Errorf("axustes do texto: %+v", r.letras)
	}
	r = resolverDeseno(plantillaConDeseno(Deseno{Letra: "moderna"}), tip)
	if r.letras.Corpo != "firasans" || len(r.avisos) == 0 {
		t.Errorf("o deseño da plantilla manda e avisa: %+v %v", r.letras, r.avisos)
	}
}

// TestMarcasPosicion (AC6): a vista previa sabe onde está cada elemento.
func TestMarcasPosicion(t *testing.T) {
	dir := t.TempDir()
	lpos := "paxina 597.50787pt 845.04684pt\ntitulos 2 4718592 50000000\ntitulos 2 4718592 50000000\nlixo\netiquetas 3 0 0\n"
	if err := os.WriteFile(filepath.Join(dir, "doc.lpos"), []byte(lpos), 0o644); err != nil {
		t.Fatal(err)
	}
	m := lerMarcasDeseno(filepath.Join(dir, "doc.lpos"))
	if len(m) != 2 || m[0].Elemento != "titulos" || m[0].Paxina != 2 || m[0].X < 0.11 || m[0].X > 0.13 || m[1].Y != 1 {
		t.Fatalf("marcas: %+v", m)
	}
	if lerMarcasDeseno(filepath.Join(dir, "non-existe")) != nil {
		t.Error("sen ficheiro non hai marcas")
	}
	if !commandExists("xelatex") {
		return
	}
	a := &App{}
	res, err := a.xerarPDFPedido(XerarInformeCronogramaRequest{Plan: planMostraCrono()}, plantillaConDeseno(Deseno{}), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	vistos := map[string]bool{}
	for _, mk := range res.Marcas {
		vistos[mk.Elemento] = true
		if mk.X < 0 || mk.X > 1 || mk.Y < 0 || mk.Y > 1 {
			t.Errorf("marca fóra da páxina: %+v", mk)
		}
	}
	for _, e := range []string{"titulos", "etiquetas", "meses", "trimestres", "tarefas"} {
		if !vistos[e] {
			t.Errorf("falta a marca de %s (%v)", e, vistos)
		}
	}
}

func TestCorrectorOverfull(t *testing.T) {
	log := "Overfull \\hbox (0.5pt too wide) in paragraph\nOverfull \\hbox (25.3pt too wide) in paragraph\nOverfull \\hbox (3pt too wide)"
	if a := avisoMarxe(log); !strings.HasPrefix(a, "2 liña") {
		t.Errorf("aviso: %q", a)
	}
	if avisoMarxe("todo ben") != "" {
		t.Error("sen overfull non hai aviso")
	}
}

// TestCoresDoLogo (AC8): dun logotipo con fondo branco, texto negro e unha
// cor sae esa cor, non un gris.
func TestCoresDoLogo(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 200; x++ {
			c := color.RGBA{255, 255, 255, 255}
			switch {
			case x < 60:
				c = color.RGBA{0x0E, 0x74, 0x90, 255} // azul petróleo
			case x < 80:
				c = color.RGBA{20, 20, 20, 255}
			case x < 90:
				c = color.RGBA{0xDC, 0x26, 0x26, 255} // vermello, menos
			}
			img.Set(x, y, c)
		}
	}
	cores := coresDaImaxe(img, 5)
	if len(cores) != 2 || cores[0] != "0E7490" || cores[1] != "DC2626" {
		t.Fatalf("cores: %v", cores)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	a := &App{}
	got, err := a.CoresDaImaxe("data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()))
	if err != nil || len(got) == 0 || got[0] != "0E7490" {
		t.Errorf("binding: %v %v", got, err)
	}
	if _, err := a.CoresDaImaxe("non é unha imaxe"); err == nil {
		t.Error("unha imaxe rota ten que dar erro")
	}
}

func docxMostra(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	escribir := func(nome, texto string) {
		w, err := zw.Create(nome)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(texto))
	}
	escribir("word/styles.xml", `<?xml version="1.0"?><w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:docDefaults><w:rPrDefault><w:rPr><w:rFonts w:asciiTheme="minorHAnsi"/><w:sz w:val="24"/></w:rPr></w:rPrDefault></w:docDefaults>
<w:style w:type="paragraph" w:styleId="Heading1"><w:rPr><w:rFonts w:ascii="Arial"/><w:color w:val="1F3864"/></w:rPr></w:style>
</w:styles>`)
	escribir("word/theme/theme1.xml", `<a:theme xmlns:a="x"><a:themeElements><a:clrScheme><a:accent1><a:srgbClr val="4472C4"/></a:accent1></a:clrScheme><a:fontScheme><a:majorFont><a:latin typeface="Calibri Light"/></a:majorFont><a:minorFont><a:latin typeface="Book Antiqua"/></a:minorFont></a:fontScheme></a:themeElements></a:theme>`)
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestImportarDesenoDocx(t *testing.T) {
	a := &App{}
	res, err := a.ImportarDesenoDeFicheiro("modelo.docx", base64.StdEncoding.EncodeToString(docxMostra(t)))
	if err != nil {
		t.Fatal(err)
	}
	d := res.Deseno
	if d.Letra != "elegante" || d.LetraTitulos != "heros" || d.Tamano != 12 || d.CorPrincipal != "1F3864" {
		t.Errorf("deseño do Word: %+v\n%v", d, res.Explicacion)
	}
	if len(res.Explicacion) < 3 {
		t.Errorf("hai que explicar de onde sae cada cousa: %v", res.Explicacion)
	}
	if _, err := a.ImportarDesenoDeFicheiro("x.odt", ""); err == nil {
		t.Error("só PDF e Word")
	}
}

// TestImportarDesenoPdf (AC9): un PDF en Palatino propón «Elegante».
func TestImportarDesenoPdf(t *testing.T) {
	if !commandExists("xelatex") || !commandExists("pdffonts") {
		t.Skip("fan falta xelatex e pdffonts")
	}
	dir := t.TempDir()
	tex := "\\documentclass{article}\\usepackage{fontspec}\\usepackage{xcolor}" + latex.SetmainfontTeXGyre("texgyrepagella") +
		"\\begin{document}\\colorbox[HTML]{B91C1C}{\\Huge\\color{white} TITULO MOI GRANDE DO DOCUMENTO}\\par Texto en Palatino.\\end{document}"
	if err := os.WriteFile(filepath.Join(dir, "m.tex"), []byte(tex), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("xelatex", "-interaction=nonstopmode", "m.tex")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "m.pdf"))
	a := &App{}
	res, err := a.ImportarDesenoDeFicheiro("m.pdf", base64.StdEncoding.EncodeToString(data))
	if err != nil {
		t.Fatal(err)
	}
	if res.Deseno.Letra != "elegante" {
		t.Errorf("letra: %+v %v", res.Deseno, res.Explicacion)
	}
	if commandExists("pdftoppm") && res.Deseno.CorPrincipal == "" {
		t.Errorf("a cor vermella tería que saír: %v", res.Explicacion)
	}
}

func TestOfimaticaDoDeseno(t *testing.T) {
	o := ofimaticaDoDeseno(OfimaticaPlantilla{}, &Deseno{Letra: "elegante", Tamano: 12, CorPrincipal: "0E7490"})
	if o.Fonte != "Palatino Linotype" || o.TamanoCorpo != 12 || o.CorPrincipal != "0E7490" || o.CorSecundaria == "" {
		t.Errorf("Word herda o deseño: %+v", o)
	}
	o = ofimaticaDoDeseno(OfimaticaPlantilla{Fonte: "Arial"}, &Deseno{Letra: "elegante"})
	if o.Fonte != "Arial" {
		t.Error("o escrito no bloque de Word manda")
	}
}

// TestDesenoSenPaquetesOpcionais: nun TeX Live mínimo sen scrextend nin
// unicode-math o documento segue compilando (o tamaño vai no corpo e as
// fórmulas na letra de sempre).
func TestDesenoSenPaquetesOpcionais(t *testing.T) {
	orixinal := latex.Dispoñible
	latex.Dispoñible = func(f string) bool { return f != "scrextend.sty" && f != "unicode-math.sty" }
	t.Cleanup(func() { latex.Dispoñible = orixinal })
	dd := compoñerDeseno("xelatex", nil, nil, plantillaConDeseno(Deseno{Letra: "clasica", Tamano: 13}))
	if strings.Contains(dd.Preambulo, "scrextend") || strings.Contains(dd.Preambulo, "unicode-math") {
		t.Errorf("sen os paquetes non se poden cargar:\n%s", dd.Preambulo)
	}
	if !strings.Contains(dd.Corpo, `\fontsize{13pt}`) {
		t.Errorf("o tamaño vai tras \\begin{document}: %q", dd.Corpo)
	}
}

// TestPortadaEspazosECor: o título da portada saía «Cronogramaanual» (o
// tamaño cambiaba dentro do nodo de TikZ e as palabras quedaban sen espazo), e
// a portada ía sempre en índigo aínda que o deseño tivese outra cor.
func TestPortadaEspazosECor(t *testing.T) {
	if !strings.Contains(portadaPorDefectoLatex, `\fill[lua-principal]`) || strings.Contains(portadaPorDefectoLatex, "tt-indigo") {
		t.Error("a portada ten que usar as cores do deseño")
	}
	if !commandExists("xelatex") || !commandExists("pdftotext") {
		return
	}
	dir := t.TempDir()
	res, err := compilarInformeCrono(planMostraCrono(), "xelatex", nil, dir, plantillaConDeseno(Deseno{CorPrincipal: "0E7490"}), "")
	if err != nil {
		t.Fatalf("%v\n%s", err, res.Log)
	}
	pdf := filepath.Join(dir, "p.pdf")
	_ = os.WriteFile(pdf, res.PDF, 0o644)
	out, err := exec.Command("pdftotext", "-f", "1", "-l", "1", pdf, "-").Output()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "Cronogramaanual") || !strings.Contains(string(out), "anual") {
		t.Errorf("o título da portada perde os espazos:\n%s", out)
	}
}

// TestLatexTempoMaximo: un documento que non remata párase ao tempo máximo e
// non se proba outro motor.
func TestLatexTempoMaximo(t *testing.T) {
	if !commandExists("xelatex") {
		t.Skip("xelatex non está instalado")
	}
	orixinal := tempoMaximoLatex
	tempoMaximoLatex = 2 * time.Second
	t.Cleanup(func() { tempoMaximoLatex = orixinal })
	tex := `\documentclass{article}\begin{document}\def\x{\x}\x\end{document}`
	inicio := time.Now()
	_, err := compilarComMotores("xelatex", nil, t.TempDir(), nil, func(string) (string, []string, error) { return tex, nil, nil })
	if err == nil || !compilacionParada(err) {
		t.Fatalf("esperaba parada por tempo, obtido %v", err)
	}
	if time.Since(inicio) > 10*time.Second {
		t.Errorf("tardou %v: non se parou", time.Since(inicio))
	}
}

// TestCancelarVistaPrevia: cancelar mata a compilación da vista previa.
func TestCancelarVistaPrevia(t *testing.T) {
	if !commandExists("xelatex") {
		t.Skip("xelatex non está instalado")
	}
	dir := t.TempDir()
	defer rexistrarVistaPrevia(dir)()
	a := &App{}
	go func() {
		time.Sleep(800 * time.Millisecond)
		a.CancelarVistaPrevia()
	}()
	tex := `\documentclass{article}\begin{document}\def\x{\x}\x\end{document}`
	inicio := time.Now()
	_, err := compilarComMotores("xelatex", nil, dir, nil, func(string) (string, []string, error) { return tex, nil, nil })
	if !errors.Is(err, errCompilacionCancelada) {
		t.Fatalf("esperaba cancelada, obtido %v", err)
	}
	if time.Since(inicio) > 8*time.Second {
		t.Errorf("a cancelación tardou %v", time.Since(inicio))
	}
}

// TestFontesEmbebidas: Fira e Libertinus van dentro de Lúa e ao carón do .tex
// só se escriben as que usa o documento.
func TestFontesEmbebidas(t *testing.T) {
	for _, f := range []string{"FiraSans-Regular.otf", "FiraSans-BoldItalic.otf", "FiraMath-Regular.otf", "LibertinusSerif-Regular.otf", "LibertinusMath-Regular.otf", "OpenSans-Regular.ttf"} {
		if _, err := fontesInformeFS.ReadFile("fontes_informe/" + f); err != nil {
			t.Errorf("Lúa non leva %s", f)
		}
	}
	dir := t.TempDir()
	if _, _, err := latexInformeCrono(planMostraCrono(), "xelatex", nil, dir, time.Now(), plantillaConDeseno(Deseno{Letra: "moderna"}), ""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "fonts", "FiraSans-Bold.otf")); err != nil {
		t.Error("falta Fira en fonts/")
	}
	if _, err := os.Stat(filepath.Join(dir, "fonts", "LibertinusSerif-Regular.otf")); err == nil {
		t.Error("Libertinus non se usa: non se escribe")
	}
}

// TestPdffontsModerna: «Moderna» sae en Fira Sans aínda que TeX Live non a
// teña (vén de fonts/).
func TestPdffontsModerna(t *testing.T) {
	if !commandExists("xelatex") || !commandExists("pdffonts") {
		t.Skip("fan falta xelatex e pdffonts")
	}
	orixinal := latex.Dispoñible
	latex.Dispoñible = func(f string) bool { return !strings.HasPrefix(f, "Fira") && orixinal(f) }
	t.Cleanup(func() { latex.Dispoñible = orixinal })
	dir := t.TempDir()
	res, err := compilarInformeCrono(planMostraCrono(), "xelatex", nil, dir, plantillaConDeseno(Deseno{Letra: "moderna"}), "")
	if err != nil {
		t.Fatalf("%v\n%s", err, res.Log)
	}
	pdf := filepath.Join(dir, "m.pdf")
	_ = os.WriteFile(pdf, res.PDF, 0o644)
	out, _ := exec.Command("pdffonts", pdf).Output()
	if !strings.Contains(string(out), "FiraSans") {
		t.Errorf("o PDF non leva Fira Sans:\n%s", out)
	}
}

// TestPortadaModelos: cada modelo de portada compila, leva o título e o
// logotipo vai onde di o deseño; «sen» quita a portada.
func TestPortadaModelos(t *testing.T) {
	for _, m := range []string{"", "sobria", "oficial"} {
		r := resolverDeseno(plantillaConDeseno(Deseno{Portada: m, LogoPortada: "dereita"}), nil)
		p := r.latexPortada()
		if !strings.Contains(p, "{{TITULO}}") || !strings.Contains(p, `anchor=north east`) {
			t.Errorf("modelo %q: falta o título ou o logotipo á dereita", m)
		}
	}
	dd := compoñerDeseno("xelatex", nil, nil, plantillaConDeseno(Deseno{Portada: "sen"}))
	if !dd.SenPortada {
		t.Error("«sen» quita a portada")
	}
	if !commandExists("xelatex") || !commandExists("pdftotext") {
		return
	}
	for _, m := range []string{"sobria", "oficial"} {
		dir := t.TempDir()
		res, err := compilarInformeCrono(planMostraCrono(), "xelatex", nil, dir, plantillaConDeseno(Deseno{Portada: m}), "")
		if err != nil {
			t.Fatalf("%s: %v\n%s", m, err, res.Log)
		}
		pdf := filepath.Join(dir, "p.pdf")
		_ = os.WriteFile(pdf, res.PDF, 0o644)
		out, _ := exec.Command("pdftotext", "-f", "1", "-l", "1", pdf, "-").Output()
		if !strings.Contains(string(out), "Curso") {
			t.Errorf("%s: a portada non sae:\n%s", m, out)
		}
	}
	tex, _, _ := latexInformeCrono(planMostraCrono(), "xelatex", nil, t.TempDir(), time.Now(), plantillaConDeseno(Deseno{Portada: "sen"}), "")
	if strings.Contains(tex, "PORTADA") {
		t.Error("sen portada non debe levar portada")
	}
}

// TestCabeceiraPeDeseno: as zonas levan texto escapado, variables enchidas,
// logotipo e número de páxina, e van despois da maqueta.
func TestCabeceiraPeDeseno(t *testing.T) {
	p := &PlantillaLatex{Nome: "x", Latex: "\\pagestyle{fancy}\\fancyhead[L]{MAQUETA}\n{{CORPO}}", Deseno: &Deseno{
		Cabeceira:    Zonas{Esquerda: "@logo {{CENTRO}}", Dereita: "Dpto. 50% & máis"},
		Pe:           Zonas{Esquerda: "{{CURSO}}"},
		NumeroPaxina: "centro",
	}}
	tex, _, err := latexInformeCrono(planMostraCrono(), "xelatex", nil, t.TempDir(), time.Now(), p, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`\fancyhead[L]{\fontchip\small\color{gris-texto}\logocentro[height=7mm] `, `Dpto. 50\% \& máis`, `\fancyfoot[C]{\fontchip\small\color{gris-texto}\thepage}`} {
		if !strings.Contains(tex, want) {
			t.Errorf("falta %q", want)
		}
	}
	if strings.Contains(tex, "{{CENTRO}}") || strings.Contains(tex, "{{CURSO}}") {
		t.Error("as variables teñen que estar enchidas")
	}
	if strings.Index(tex, "MAQUETA") > strings.Index(tex, "% Cabeceira e pé do deseño") {
		t.Error("a cabeceira do deseño vai despois da maqueta")
	}
	if !commandExists("xelatex") {
		return
	}
	if res, err := compilarInformeCrono(planMostraCrono(), "xelatex", nil, t.TempDir(), p, ""); err != nil {
		t.Fatalf("non compila: %v\n%s", err, res.Log)
	}
}

func TestOfimaticaCabeceiraDoDeseno(t *testing.T) {
	o := ofimaticaDoDeseno(OfimaticaPlantilla{}, &Deseno{Cabeceira: Zonas{Centro: "@logo {{CENTRO}}\n@imaxe escudo.png"}, Pe: Zonas{Dereita: "{{CURSO}}"}})
	if o.Cabeceira != "{{CENTRO}}\n@imaxe escudo.png 8\n@centro" || o.Pe != "{{CURSO}}\n@dereita" {
		t.Errorf("cabeceira de Word: %q / %q", o.Cabeceira, o.Pe)
	}
	o = ofimaticaDoDeseno(OfimaticaPlantilla{Cabeceira: "PROPIA"}, &Deseno{Cabeceira: Zonas{Centro: "x"}})
	if o.Cabeceira != "PROPIA" {
		t.Error("a de Word manda")
	}
}

func TestFicheiroDeseno(t *testing.T) {
	a := &App{}
	datos, err := codificarDeseno(Deseno{Letra: "elegante", CorPrincipal: "#0e7490", Tamano: 99}, "Centro")
	if err != nil {
		t.Fatal(err)
	}
	d, err := a.LerDesenoFicheiro(string(datos))
	if err != nil || d.Letra != "elegante" || d.CorPrincipal != "0E7490" || d.Tamano != 20 {
		t.Errorf("ida e volta: %+v %v", d, err)
	}
	if _, err := a.LerDesenoFicheiro(`{"formato":"outro"}`); err == nil {
		t.Error("un ficheiro que non é un deseño ten que dar erro")
	}
	if _, err := a.LerDesenoFicheiro(`{"formato":"lua-deseno","version":9}`); err == nil {
		t.Error("unha versión máis nova ten que dar erro")
	}
}

func TestAplicarDesenoATodas(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())
	alm := almacenPlantillas{Plantillas: []PlantillaLatex{
		{ID: "a", Nome: "A", Latex: "{{CORPO}}"},
		{ID: "b", Nome: "B", Latex: "{{CORPO}}", Deseno: &Deseno{Letra: "moderna"}},
		{ID: "f", Nome: "Fábrica", Latex: "{{CORPO}}", DeFabrica: true},
	}}
	if err := saveAlmacenPlantillas(alm); err != nil {
		t.Fatal(err)
	}
	a := &App{}
	n, err := a.AplicarDesenoATodas(Deseno{Letra: "clasica"})
	if err != nil || n != 2 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	volta, _ := loadAlmacenPlantillas()
	for _, p := range volta.Plantillas {
		if p.DeFabrica && p.Deseno != nil {
			t.Errorf("as de exemplo non se tocan: %s", p.ID)
		}
		if !p.DeFabrica && (p.Deseno == nil || p.Deseno.Letra != "clasica") {
			t.Errorf("%s non levou o deseño", p.ID)
		}
	}
}

func TestComprobarElementoDeseno(t *testing.T) {
	if !commandExists("xelatex") {
		t.Skip("xelatex non está instalado")
	}
	a := &App{}
	bo := `\newcommand{\tarefa}[2]{\par\textbf{#2}\par}`
	if msg, err := a.ComprobarElementoDeseno(Deseno{}, "tarefas", bo); err != nil || msg != "" {
		t.Errorf("un LaTeX bo non dá erro: %q %v", msg, err)
	}
	malo := `\newcommand{\tarefa}[2]{\par\textbf{#2}\comandoQueNonExiste\par}`
	if msg, _ := a.ComprobarElementoDeseno(Deseno{}, "tarefas", malo); !strings.Contains(msg, "comandoQueNonExiste") && !strings.Contains(strings.ToLower(msg), "undefined") {
		t.Errorf("o erro ten que dicir que falla: %q", msg)
	}
	if _, err := a.ComprobarElementoDeseno(Deseno{}, "inventado", ""); err == nil {
		t.Error("unha parte que non existe dá erro")
	}
}
