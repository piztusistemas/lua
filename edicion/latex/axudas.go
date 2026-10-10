package latex

// Auxiliares de LaTeX comúns a Lúa e a edicion/documento: tipografía
// (fontes e tamaños), motores dispoñibles e pistas para os erros de LaTeX.
// Movidos aquí desde as dúas copias que había (constitución, regra 1.3).

import (
	"edicion/idioma"
	"edicion/plataforma"
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

// EngineUnicodeNativo di se `engine` xa traballa en UTF-8/fontes do sistema
// de fábrica (xelatex, lualatex) fronte ao pdflatex por defecto, que
// necesita inputenc/fontenc - determina que preámbulo usar.
func EngineUnicodeNativo(engine string) bool {
	return engine == "xelatex" || engine == "lualatex"
}

// BabelLine picks a babel language the installed TeX Live can actually
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
func BabelLine() string {
	for _, lang := range []string{"galician", "spanish"} {
		if KpsewhichFound(lang + ".ldf") {
			return `\usepackage[` + lang + `,shorthands=off]{babel}` + "\n"
		}
	}
	return ""
}

func KpsewhichFound(file string) bool {
	if !plataforma.CommandExists("kpsewhich") {
		return false
	}
	cmd := exec.Command("kpsewhich", file)
	plataforma.OcultarConsola(cmd)
	return cmd.Run() == nil
}

// MotoresParaProbar devolve, en orde, os motores LaTeX candidatos para
// compilar un documento: primeiro o preferido (o escollido en Opcións),
// despois os outros dous coma fallback transparente - só os que estean
// realmente instalados no sistema. Isto é o que permite a compileLatexAuto
// recuperarse dun motor mal escollido (ou non instalado) sen que o usuario
// teña que decatarse: se o preferido non chega a xerar PDF, próbanse os
// demais antes de darse por vencido.
func MotoresParaProbar(preferido string) []string {
	todos := []string{"pdflatex", "xelatex", "lualatex"}
	orde := todos
	if preferido != "" {
		orde = append([]string{preferido}, todos...)
	}
	var dispoñibles []string
	xaProbado := map[string]bool{}
	for _, m := range orde {
		if xaProbado[m] || !plataforma.CommandExists(m) {
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

// PistaLatexAccionable traduce un log de LaTeX a unha frase que diga QUE
// hai que facer, ou "" se non recoñece o caso (entón queda o erro cru de
// sempre, que para un erro de sintaxe é máis útil ca calquera paráfrase).
func PistaLatexAccionable(log string) string {
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

// ExtraerErroPDFLatex busca as liñas de erro estándar de pdflatex (as que
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

// ContinuacionPaquete devolve o texto dunha liña de continuación de erro de
// paquete ("(fontspec)      The font ... cannot be found"), ou "" se a liña
// non o é. O formato é sempre (nome-do-paquete) + recheo de espazos.
var continuacionRe = regexp.MustCompile(`^\(([A-Za-z0-9@_.-]+)\)\s+(\S.*)$`)

func ContinuacionPaquete(liña string) string {
	m := continuacionRe.FindStringSubmatch(strings.TrimRight(liña, "\r"))
	if m == nil {
		return ""
	}
	return strings.TrimSpace(m[2])
}

func ExtraerErroPDFLatex(log string) string {
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
			resto := ContinuacionPaquete(liñas[j])
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

// PdfPageImages rasterizes every page of a compiled PDF to PNG via pdftoppm
// (poppler-utils - the same tool cas.tikz already relies on for its HTML
// preview), so the webview can show a faithful print preview: WebKitGTK's
// embedded webview (what Wails uses on Linux) has no reliable native PDF
// renderer to point an <iframe>/<embed> at, but any webview can show <img>
// tags. Returns one data: URI per page, in page order.
func PdfPageImages(dir, pdfPath string) ([]string, error) {
	prefix := filepath.Join(dir, "page")
	cmd := exec.Command("pdftoppm", "-png", "-r", "100", pdfPath, prefix)
	plataforma.OcultarConsola(cmd)
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

// SetmainfontTeXGyre carga unha familia TeX Gyre polos seus ficheiros .otf
// de TeX Live (texgyreheros-regular.otf...). Polo nome («TeX Gyre Heros»)
// fontspec só a atopa se está no sistema, e en macOS non o está.
func SetmainfontTeXGyre(base string) string {
	return "\\setmainfont{" + base + "}[Extension=.otf,UprightFont=*-regular,BoldFont=*-bold,ItalicFont=*-italic,BoldItalicFont=*-bolditalic]\n"
}

// EsFonteDoSistema di se `f` é un nome de familia do sistema (fc-list), é
// dicir: non baleiro e non un dos presets. Estas fontes SÓ as pode usar
// xelatex/lualatex - GeneratePDF forza o motor cando aparece unha.
func EsFonteDoSistema(f string) bool {
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

// TipografiaPreambulo: liña(s) que van no PREÁMBULO (carga da familia de
// fonte). "" se non hai fonte escollida, ou se é unha fonte do sistema pero
// o motor non é Unicode (nese caso GeneratePDF xa debería telo forzado).
func TipografiaPreambulo(t *TipografiaOpts, engine string) string {
	if t == nil || strings.TrimSpace(t.Fonte) == "" {
		return ""
	}
	f := strings.TrimSpace(t.Fonte)
	if p, ok := presetsFonte[f]; ok {
		if EngineUnicodeNativo(engine) {
			if p.unicode == "" {
				return ""
			}
			return SetmainfontTeXGyre(p.unicode)
		}
		return p.pdflatex + "\n"
	}
	// Fonte do sistema: só fontspec (xelatex/lualatex).
	if !EngineUnicodeNativo(engine) {
		return ""
	}
	nome := strings.TrimSpace(fontNameSafeRe.ReplaceAllString(f, ""))
	if nome == "" {
		return ""
	}
	return "\\setmainfont{" + nome + "}\n"
}

// TipografiaCorpo: comandos que van XUSTO DESPOIS de \begin{document}
// (tamaño e interliñado). "" se ambos son os de por defecto.
func TipografiaCorpo(t *TipografiaOpts) string {
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
			TrimFloat(t.TamanoPt), TrimFloat(t.TamanoPt*1.2*inter))
	}
	return fmt.Sprintf("\\linespread{%s}\\selectfont\n", TrimFloat(inter))
}

// TrimFloat formatea un float a 2 decimais como moito e sen ceros de cola
// ("13", "1.5", "15.6") - evita que 12*1.2*1.5 saia como
// "21.599999999999998pt" no .tex.
func TrimFloat(f float64) string {
	s := strconv.FormatFloat(f, 'f', 2, 64)
	s = strings.TrimRight(s, "0")
	return strings.TrimRight(s, ".")
}

// PresetsFonte devolve as claves das fontes predefinidas (serif, sans...), en orde.
func PresetsFonte() []string {
	out := make([]string, 0, len(presetsFonte))
	for k := range presetsFonte {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
