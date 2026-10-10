package main

import (
	"edicion/idioma"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// latexPostamble pecha o documento.
const latexPostamble = "\n\\end{document}\n"

// latexUnicodeDeclarations son as declaracións \DeclareUnicodeCharacter que
// necesita pdflatex (inputenc) para non petar cun símbolo Unicode solto
// (≈, ±, →, letras gregas...) nun texto: a fonte Computer Modern non ten eses
// glifos en modo texto. xelatex/lualatex non as precisan (son Unicode
// nativos). Vai no preámbulo do informe e inxéctase tamén nas plantillas
// con \documentclass propio (ver inxectarPreambuloObrigatorio).
const latexUnicodeDeclarations = `\DeclareUnicodeCharacter{2212}{\ensuremath{-}}
\DeclareUnicodeCharacter{2248}{\ensuremath{\approx}}
\DeclareUnicodeCharacter{2264}{\ensuremath{\leq}}
\DeclareUnicodeCharacter{2265}{\ensuremath{\geq}}
\DeclareUnicodeCharacter{2260}{\ensuremath{\neq}}
\DeclareUnicodeCharacter{00B1}{\ensuremath{\pm}}
\DeclareUnicodeCharacter{00D7}{\ensuremath{\times}}
\DeclareUnicodeCharacter{00F7}{\ensuremath{\div}}
\DeclareUnicodeCharacter{00B0}{\textdegree}
\DeclareUnicodeCharacter{2192}{\ensuremath{\rightarrow}}
\DeclareUnicodeCharacter{2190}{\ensuremath{\leftarrow}}
\DeclareUnicodeCharacter{2194}{\ensuremath{\leftrightarrow}}
\DeclareUnicodeCharacter{21D2}{\ensuremath{\Rightarrow}}
\DeclareUnicodeCharacter{21D0}{\ensuremath{\Leftarrow}}
\DeclareUnicodeCharacter{21D4}{\ensuremath{\Leftrightarrow}}
\DeclareUnicodeCharacter{221E}{\ensuremath{\infty}}
\DeclareUnicodeCharacter{2211}{\ensuremath{\sum}}
\DeclareUnicodeCharacter{221A}{\ensuremath{\surd}}
\DeclareUnicodeCharacter{222A}{\ensuremath{\cup}}
\DeclareUnicodeCharacter{2229}{\ensuremath{\cap}}
\DeclareUnicodeCharacter{2208}{\ensuremath{\in}}
\DeclareUnicodeCharacter{2209}{\ensuremath{\notin}}
\DeclareUnicodeCharacter{2282}{\ensuremath{\subset}}
\DeclareUnicodeCharacter{2286}{\ensuremath{\subseteq}}
\DeclareUnicodeCharacter{2205}{\ensuremath{\emptyset}}
\DeclareUnicodeCharacter{2216}{\ensuremath{\setminus}}
\DeclareUnicodeCharacter{211D}{\ensuremath{\mathbb{R}}}
\DeclareUnicodeCharacter{2115}{\ensuremath{\mathbb{N}}}
\DeclareUnicodeCharacter{2124}{\ensuremath{\mathbb{Z}}}
\DeclareUnicodeCharacter{211A}{\ensuremath{\mathbb{Q}}}
\DeclareUnicodeCharacter{2102}{\ensuremath{\mathbb{C}}}
\DeclareUnicodeCharacter{03B1}{\ensuremath{\alpha}}
\DeclareUnicodeCharacter{03B2}{\ensuremath{\beta}}
\DeclareUnicodeCharacter{03B3}{\ensuremath{\gamma}}
\DeclareUnicodeCharacter{03B4}{\ensuremath{\delta}}
\DeclareUnicodeCharacter{03B5}{\ensuremath{\varepsilon}}
\DeclareUnicodeCharacter{03B6}{\ensuremath{\zeta}}
\DeclareUnicodeCharacter{03B7}{\ensuremath{\eta}}
\DeclareUnicodeCharacter{03B8}{\ensuremath{\theta}}
\DeclareUnicodeCharacter{03B9}{\ensuremath{\iota}}
\DeclareUnicodeCharacter{03BA}{\ensuremath{\kappa}}
\DeclareUnicodeCharacter{03BB}{\ensuremath{\lambda}}
\DeclareUnicodeCharacter{03BC}{\ensuremath{\mu}}
\DeclareUnicodeCharacter{03BD}{\ensuremath{\nu}}
\DeclareUnicodeCharacter{03BE}{\ensuremath{\xi}}
\DeclareUnicodeCharacter{03C0}{\ensuremath{\pi}}
\DeclareUnicodeCharacter{03C1}{\ensuremath{\rho}}
\DeclareUnicodeCharacter{03C3}{\ensuremath{\sigma}}
\DeclareUnicodeCharacter{03C4}{\ensuremath{\tau}}
\DeclareUnicodeCharacter{03C5}{\ensuremath{\upsilon}}
\DeclareUnicodeCharacter{03C6}{\ensuremath{\varphi}}
\DeclareUnicodeCharacter{03C7}{\ensuremath{\chi}}
\DeclareUnicodeCharacter{03C8}{\ensuremath{\psi}}
\DeclareUnicodeCharacter{03C9}{\ensuremath{\omega}}
\DeclareUnicodeCharacter{0393}{\ensuremath{\Gamma}}
\DeclareUnicodeCharacter{0394}{\ensuremath{\Delta}}
\DeclareUnicodeCharacter{0398}{\ensuremath{\Theta}}
\DeclareUnicodeCharacter{039B}{\ensuremath{\Lambda}}
\DeclareUnicodeCharacter{039E}{\ensuremath{\Xi}}
\DeclareUnicodeCharacter{03A0}{\ensuremath{\Pi}}
\DeclareUnicodeCharacter{03A3}{\ensuremath{\Sigma}}
\DeclareUnicodeCharacter{03A6}{\ensuremath{\Phi}}
\DeclareUnicodeCharacter{03A8}{\ensuremath{\Psi}}
`

// engineUnicodeNativo di se `engine` xa traballa en UTF-8/fontes do sistema
// de fábrica (xelatex, lualatex) fronte ao pdflatex por defecto, que
// necesita inputenc/fontenc - determina que preámbulo usar.
func engineUnicodeNativo(engine string) bool {
	return engine == "xelatex" || engine == "lualatex"
}

// babelLine picks a babel language the installed TeX Live can actually
// find (galician.ldf and even spanish.ldf require the texlive-lang-*
// packages, which a base/minimal install won't have - only english.ldf is
// always guaranteed to exist). Skipping babel entirely when nothing fits
// still produces a perfectly good PDF, just without language-aware
// hyphenation.
//
// "shorthands=off": galician/spanish babel fan ACTIVOS varios caracteres
// (<, >, ", ~, ...) para atallos tipográficos («», ñ...). Nun exame eses
// atallos non se usan (o contido xa vén en UTF-8 con «, ñ... de verdade) e
// en troques rebentan a compilación en canto aparece un "<", ">", "->" ou
// "<=" no contido - dentro dun <TIKZ>, dun <TEX>, ou nun texto que a IA non
// escapou - con erros crípticos ("Argument of \language@active@arg> has an
// extra }", "Incomplete \iffalse"). Apagalos de raíz é máis seguro e non
// perde nada visible. (A libraría tikz "babel", ver tikzBabelLine, xa
// cubría o caso dos tikzpicture; isto cobre todo o demais.)
func babelLine() string {
	for _, lang := range []string{"galician", "spanish"} {
		if kpsewhichFound(lang + ".ldf") {
			return `\usepackage[` + lang + `,shorthands=off]{babel}` + "\n"
		}
	}
	return ""
}

// babelListquotFix neutraliza un segundo xeito (independente de
// "shorthands=off", babelLine enriba) de que babel-galician/babel-spanish
// rompan calquera <ol>/<ul> de dous ou máis elementos con "Incomplete
// \iffalse; all text was ignored after line N" - SEN que faga falla
// <TIKZ> nin ningún carácter activo (<, >...) no medio: abonda con DOUS
// "\item" en calquera lista, mesmo "\item $x+y=1$" repetido dúas veces
// (report real - unha lista numerada con varios pasos abonda).
//
// Motivo (galician.ldf/spanish.ldf, función gl@enumdef/\gl@itemize):
// \labelenumN e \labelitemN redefínense para chamaren \gl@listquot (ou
// es@listquot), pensado para acentos de peche de cita cando un \item
// continúa un diálogo dentro dun ambiente "quoting" - Lua nunca usa
// "quoting", pero \gl@listquot chámase IGUALMENTE en cada \item, e a súa
// interacción coas modificacións de \item de enumitem (\usepackage
// {enumitem} enriba, necesario para <ol type="a">) é o que desincroniza o
// contador de condicionais de TeX e xera o \iffalse fantasma.
//
// A neutralización ten que ir DESPOIS de \begin{document} (non no
// preámbulo): babel volve activar \gl@listquot ao seleccionar o idioma,
// nun hook que se executa XUSTO AO EMPEZAR O DOCUMENTO - calquera
// \renewcommand no preámbulo queda pisado por ese hook e non serve de
// nada (comprobado). \@ifundefined fai que isto sexa inofensivo cando
// babel non cargou galego/castelán (inglés, ou sen babel ningún): non
// toca nada que non exista. \gl@listquot/es@listquot non levan
// argumentos e non se usan para máis nada no documento (a numeración en
// si - "1.", "a)"...) - queda intacta, só desaparece o efecto colateral
// que rompía a compilación.
//
// Insírese xusto tras \begin{document}, canda tipografiaCorpo (ver
// latexInformeCrono, informe_cronograma.go) - así chega igual con
// plantilla fragmento ca con plantilla con \documentclass propio.
const babelListquotFix = `\makeatletter
\@ifundefined{gl@listquot}{}{\renewcommand{\gl@listquot}{}}
\@ifundefined{es@listquot}{}{\renewcommand{\es@listquot}{}}
\makeatother
`

func kpsewhichFound(file string) bool {
	if !commandExists("kpsewhich") {
		return false
	}
	cmd := exec.Command("kpsewhich", file)
	ocultarConsola(cmd)
	return cmd.Run() == nil
}

// motoresParaProbar devolve, en orde, os motores LaTeX candidatos para
// compilar un documento: primeiro o preferido (o escollido en Opcións),
// despois os outros dous coma fallback transparente - só os que estean
// realmente instalados no sistema. Isto é o que permite a compileLatexAuto
// recuperarse dun motor mal escollido (ou non instalado) sen que o usuario
// teña que decatarse: se o preferido non chega a xerar PDF, próbanse os
// demais antes de darse por vencido.
func motoresParaProbar(preferido string) []string {
	todos := []string{"pdflatex", "xelatex", "lualatex"}
	orde := todos
	if preferido != "" {
		orde = append([]string{preferido}, todos...)
	}
	var dispoñibles []string
	xaProbado := map[string]bool{}
	for _, m := range orde {
		if xaProbado[m] || !commandExists(m) {
			continue
		}
		xaProbado[m] = true
		dispoñibles = append(dispoñibles, m)
	}
	return dispoñibles
}

// fonteQueFaltaRe/paqueteQueFaltaRe recoñecen os dous fallos que máis se
// dan cunha PLANTILLA do profesorado (plantillas.go), sobre todo se a
// escribiu a IA a partir dun modelo bonito: pide unha fonte do sistema ou
// un paquete de LaTeX que ese equipo non ten. O erro cru de LaTeX non lle
// di a ninguén que facer; estas dúas pistas si.
var fonteQueFaltaRe = regexp.MustCompile(`The font "([^"]+)" cannot be found`)
var paqueteQueFaltaRe = regexp.MustCompile("File `([A-Za-z0-9@_.-]+)\\.sty' not found")

// ambienteIndefinidoRe: "! LaTeX Error: Environment foobar undefined." -
// case típico de contido da IA que escribe \begin{algo} cun ambiente que non
// existe (ou que non ten sentido nese sitio). sanearAmbientesLatex xa
// desactiva os \begin/\end SEN "{", pero un \begin{align}/\begin{aligned}/…
// mal colocado aínda chega aquí.
var ambienteIndefinidoRe = regexp.MustCompile(`Environment (\S+) undefined`)

// pistaLatexAccionable traduce un log de LaTeX a unha frase que diga QUE
// hai que facer, ou "" se non recoñece o caso (entón queda o erro cru de
// sempre, que para un erro de sintaxe é máis útil ca calquera paráfrase).
func pistaLatexAccionable(log string) string {
	if m := fonteQueFaltaRe.FindStringSubmatch(log); m != nil {
		return idioma.T("go.latex.fonte_falta", "a plantilla pide a fonte «%s», que non está instalada neste equipo: instálaa, ou cambia a fonte (\\setmainfont/\\setsansfont/\\setmonofont) no editor de plantillas", m[1])
	}
	if m := paqueteQueFaltaRe.FindStringSubmatch(log); m != nil {
		return idioma.T("go.latex.paquete_falta", "falta o paquete de LaTeX «%s»: instálao (p.ex. 'sudo tlmgr install %s') ou quítao da plantilla", m[1], m[1])
	}
	if m := ambienteIndefinidoRe.FindStringSubmatch(log); m != nil {
		return idioma.T("go.latex.ambiente_inexistente", "o documento usa un ambiente LaTeX que non existe («%s»); adoita vir da IA ao mesturar \\begin{...} coas etiquetas <...>. Rexenera ese exercicio, ou na súa resolución usa <SISTEMA> para os sistemas e <MAT>matrix(...)</MAT> para as matrices en vez de \\begin{...}", strings.Trim(m[1], `.`))
	}
	return ""
}

// extraerErroPDFLatex busca as liñas de erro estándar de pdflatex (as que
// empezan por "!", ex. "! Undefined control sequence.", "! Package ...
// Error: ...") máis a liña "l.NN ..." que indica ONDE - mesmo patrón que xa
// usa cas.tikz para os seus propios avisos.
//
// A liña "l.NN" non sempre vai xusto debaixo do "!": erros coma "! Missing
// \endgroup inserted." ou "! Missing $ inserted." meten primeiro o que TeX
// inseriu ("<inserted text>", "<template>", "<recently read>"...) e só
// despois a localización. Buscando só na liña seguinte quedaba fóra
// precisamente nos erros que máis falla fan de localizar, e o profesorado
// recibía un "! Missing \endgroup inserted." pelado, sen liña ningunha.
// Por iso ollamos unhas cantas liñas cara adiante, ata a primeira "l.NN" ou
// ata o vindeiro "!" (que xa é outro erro distinto).
const maxLiñasAtaLocalizacion = 8

// maxLiñasContinuacion limita canto se colle das liñas "(paquete) ..." que
// seguen a un erro de paquete: as primeiras din o problema, o resto adoita
// ser a lea xenérica de LaTeX ("this may be but usually is not a bug...").
const maxLiñasContinuacion = 4

// continuacionPaquete devolve o texto dunha liña de continuación de erro de
// paquete ("(fontspec)      The font ... cannot be found"), ou "" se a liña
// non o é. O formato é sempre (nome-do-paquete) + recheo de espazos.
var continuacionRe = regexp.MustCompile(`^\(([A-Za-z0-9@_.-]+)\)\s+(\S.*)$`)

func continuacionPaquete(liña string) string {
	m := continuacionRe.FindStringSubmatch(strings.TrimRight(liña, "\r"))
	if m == nil {
		return ""
	}
	return strings.TrimSpace(m[2])
}

func extraerErroPDFLatex(log string) string {
	liñas := strings.Split(log, "\n")
	var atopadas []string
	for i, l := range liñas {
		if !strings.HasPrefix(strings.TrimSpace(l), "!") {
			continue
		}
		erro := strings.TrimSpace(l)
		// Continuación "(paquete)   ...": os erros de paquete parten a
		// explicación en varias liñas prefixadas co seu nome, e a PRIMEIRA
		// (a que empeza por "!") pode quedar baleira despois dos dous
		// puntos. Report real: "! Package fontspec Error:" a secas, cando o
		// que fai falla saber ("The font "Fira Sans" cannot be found") ía
		// nas liñas de abaixo. Únense nunha soa frase, colapsando o
		// recheo de espazos co que LaTeX as aliña.
		for j := i + 1; j < len(liñas) && j <= i+maxLiñasContinuacion; j++ {
			resto := continuacionPaquete(liñas[j])
			if resto == "" {
				break
			}
			erro = strings.TrimSpace(erro) + " " + resto
		}
		atopadas = append(atopadas, erro)
		for j := i + 1; j < len(liñas) && j <= i+maxLiñasAtaLocalizacion; j++ {
			seguinte := strings.TrimSpace(liñas[j])
			if strings.HasPrefix(seguinte, "!") {
				break
			}
			if strings.HasPrefix(seguinte, "l.") {
				atopadas = append(atopadas, seguinte)
				break
			}
		}
	}
	return strings.Join(atopadas, " | ")
}

// pdfPageImages rasterizes every page of a compiled PDF to PNG via pdftoppm
// (poppler-utils - the same tool cas.tikz already relies on for its HTML
// preview), so the webview can show a faithful print preview: WebKitGTK's
// embedded webview (what Wails uses on Linux) has no reliable native PDF
// renderer to point an <iframe>/<embed> at, but any webview can show <img>
// tags. Returns one data: URI per page, in page order.
func pdfPageImages(dir, pdfPath string) ([]string, error) {
	prefix := filepath.Join(dir, "page")
	cmd := exec.Command("pdftoppm", "-png", "-r", "100", pdfPath, prefix)
	ocultarConsola(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("pdftoppm: %w: %s", err, out)
	}

	matches, err := filepath.Glob(prefix + "-*.png")
	if err != nil {
		return nil, err
	}
	// pdftoppm's own numeric suffix (page-1.png, page-2.png, ...) doesn't
	// zero-pad below 10 pages, so a plain string sort would put page-10.png
	// before page-2.png - sort by the parsed page number instead.
	pageNumRe := regexp.MustCompile(`-(\d+)\.png$`)
	pageNum := func(name string) int {
		m := pageNumRe.FindStringSubmatch(name)
		if m == nil {
			return 0
		}
		n, _ := strconv.Atoi(m[1])
		return n
	}
	sort.Slice(matches, func(i, j int) bool { return pageNum(matches[i]) < pageNum(matches[j]) })

	images := make([]string, 0, len(matches))
	for _, m := range matches {
		data, err := os.ReadFile(m)
		if err != nil {
			return nil, err
		}
		images = append(images, "data:image/png;base64,"+base64.StdEncoding.EncodeToString(data))
	}
	return images, nil
}

// TipografiaOpts leva os tres axustes globais do botón "Axustes do texto"
// (frontend/src/main.js). Aplícanse SÓ á saída PDF (é o que se
// previsualiza); a plantilla, se ten \documentclass propio, segue mandando
// na súa fonte (só se lle aplica o tamaño/interliñado).
type TipografiaOpts struct {
	TamanoPt    float64 `json:"tamanoPt"`    // corpo do texto en pt; 0 = por defecto
	Interlinado float64 `json:"interlinado"` // factor de \linespread; 0 = 1.0 (sinxelo)
	Fonte       string  `json:"fonte"`       // id dun preset (ver presetsFonte); "" = por defecto
}

// presetsFonte: id -> paquete a cargar en pdflatex / base do ficheiro TeX Gyre para
// \setmainfont en xelatex+lualatex. Só familias moi estendidas (viven en
// texlive-latex-recommended / texlive-fonts-recommended); se faltan, o aviso
// de "instalar paquete de LaTeX" (paquetelatex.go) xa o cobre.
var presetsFonte = map[string]struct{ pdflatex, unicode string }{
	"serif":    {"\\usepackage{mathptmx}", "texgyretermes"},
	"sans":     {"\\usepackage[scaled]{helvet}\n\\renewcommand\\familydefault{\\sfdefault}", "texgyreheros"},
	"palatino": {"\\usepackage{mathpazo}", "texgyrepagella"},
}

// setmainfontTeXGyre carga unha familia TeX Gyre polos seus ficheiros .otf
// de TeX Live (texgyreheros-regular.otf...). Polo nome («TeX Gyre Heros»)
// fontspec só a atopa se está no sistema, e en macOS non o está.
func setmainfontTeXGyre(base string) string {
	return "\\setmainfont{" + base + "}[Extension=.otf,UprightFont=*-regular,BoldFont=*-bold,ItalicFont=*-italic,BoldItalicFont=*-bolditalic]\n"
}

// esFonteDoSistema di se `f` é un nome de familia do sistema (fc-list), é
// dicir: non baleiro e non un dos presets. Estas fontes SÓ as pode usar
// xelatex/lualatex - GeneratePDF forza o motor cando aparece unha.
func esFonteDoSistema(f string) bool {
	f = strings.TrimSpace(f)
	if f == "" {
		return false
	}
	_, ok := presetsFonte[f]
	return !ok
}

// fontNameSafeRe limpa o nome de familia que chega do frontend antes de
// metelo nun \setmainfont{...}: só letras, díxitos, espazos e puntuación
// inofensiva (os nomes de fonte reais están dentro diso). Evita inxección de
// LaTeX a través do valor do selector.
var fontNameSafeRe = regexp.MustCompile(`[^\p{L}\p{N} .,'&()+_-]`)

// tipografiaPreambulo: liña(s) que van no PREÁMBULO (carga da familia de
// fonte). "" se non hai fonte escollida, ou se é unha fonte do sistema pero
// o motor non é Unicode (nese caso GeneratePDF xa debería telo forzado).
func tipografiaPreambulo(t *TipografiaOpts, engine string) string {
	if t == nil || strings.TrimSpace(t.Fonte) == "" {
		return ""
	}
	f := strings.TrimSpace(t.Fonte)
	if p, ok := presetsFonte[f]; ok {
		if engineUnicodeNativo(engine) {
			if p.unicode == "" {
				return ""
			}
			return setmainfontTeXGyre(p.unicode)
		}
		return p.pdflatex + "\n"
	}
	// Fonte do sistema: só fontspec (xelatex/lualatex).
	if !engineUnicodeNativo(engine) {
		return ""
	}
	nome := strings.TrimSpace(fontNameSafeRe.ReplaceAllString(f, ""))
	if nome == "" {
		return ""
	}
	return "\\setmainfont{" + nome + "}\n"
}

// tipografiaCorpo: comandos que van XUSTO DESPOIS de \begin{document}
// (tamaño e interliñado). "" se ambos son os de por defecto.
func tipografiaCorpo(t *TipografiaOpts) string {
	if t == nil || (t.TamanoPt <= 0 && t.Interlinado <= 0) {
		return ""
	}
	inter := t.Interlinado
	if inter <= 0 {
		inter = 1.0
	}
	if t.TamanoPt > 0 {
		// baseline = tamaño * 1.2 (factor estándar de TeX) * interliñado.
		return fmt.Sprintf("\\fontsize{%spt}{%spt}\\selectfont\n",
			trimFloat(t.TamanoPt), trimFloat(t.TamanoPt*1.2*inter))
	}
	return fmt.Sprintf("\\linespread{%s}\\selectfont\n", trimFloat(inter))
}

// trimFloat formatea un float a 2 decimais como moito e sen ceros de cola
// ("13", "1.5", "15.6") - evita que 12*1.2*1.5 saia como
// "21.599999999999998pt" no .tex.
func trimFloat(f float64) string {
	s := strconv.FormatFloat(f, 'f', 2, 64)
	s = strings.TrimRight(s, "0")
	return strings.TrimRight(s, ".")
}

type GeneratePDFResult struct {
	PDFBase64 string `json:"pdfBase64"`
	// LatexSource is the exact .tex source pdflatex compiled (preamble +
	// body + \end{document}) - plain text, no base64 needed. Lets the user
	// export/edit the LaTeX directly instead of only the compiled PDF.
	LatexSource string `json:"latexSource"`
	// PageImages holds one data: URI (PNG) per PDF page, for previewing the
	// compiled document in the webview - see pdfPageImages. Empty if
	// pdftoppm isn't installed; the PDF itself still generates fine, it
	// just can't be shown page-by-page in the preview pane.
	PageImages []string `json:"pageImages"`
	Warnings   []string `json:"warnings"`
	Log        string   `json:"log"` // pdflatex's own output, for troubleshooting a failed compile
}

// SavePDFDialog shows a native "save as" dialog and writes base64-encoded
// PDF bytes (as returned by GeneratePDF) to the chosen path.
func (a *App) SavePDFDialog(defaultName, pdfBase64 string) (string, error) {
	dir, base := splitDialogDefault(defaultName)
	path, err := saveFileDialogCompat("Exportar PDF", dir, base, "PDF (*.pdf)", "*.pdf")
	if err != nil || path == "" {
		return "", err
	}
	data, err := base64.StdEncoding.DecodeString(pdfBase64)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// SaveTexDialog shows a native "save as" dialog and writes the .tex source
// (as returned by GeneratePDF's LatexSource) to the chosen path - plain
// text, unlike SavePDFDialog there's no base64 decoding step.
func (a *App) SaveTexDialog(defaultName, content string) (string, error) {
	dir, base := splitDialogDefault(defaultName)
	path, err := saveFileDialogCompat("Exportar código LaTeX", dir, base, "LaTeX (*.tex)", "*.tex")
	if err != nil || path == "" {
		return "", err
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", err
	}
	return path, nil
}
