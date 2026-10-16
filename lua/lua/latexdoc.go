package main

import (
	"edicion/latex"
	"encoding/base64"
	"os"
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
	// Marcas: onde está cada elemento do deseño nas páxinas (deseno_marcas.go).
	Marcas []MarcaDeseno `json:"marcas,omitempty"`
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

// Auxiliares de LaTeX comúns: viven en edicion/latex (constitución, regra 1.3).
// Aquí só quedan adaptadores dunha liña para non cambiar as chamadas.
type TipografiaOpts = latex.TipografiaOpts

func babelLine() string                                   { return latex.BabelLine() }
func continuacionPaquete(liña string) string              { return latex.ContinuacionPaquete(liña) }
func engineUnicodeNativo(engine string) bool              { return latex.EngineUnicodeNativo(engine) }
func esFonteDoSistema(f string) bool                      { return latex.EsFonteDoSistema(f) }
func extraerErroPDFLatex(log string) string               { return latex.ExtraerErroPDFLatex(log) }
func kpsewhichFound(file string) bool                     { return latex.KpsewhichFound(file) }
func motoresParaProbar(preferido string) []string         { return latex.MotoresParaProbar(preferido) }
func pdfPageImages(dir, pdfPath string) ([]string, error) { return latex.PdfPageImages(dir, pdfPath) }
func pistaLatexAccionable(log string) string              { return latex.PistaLatexAccionable(log) }
func setmainfontTeXGyre(base string) string               { return latex.SetmainfontTeXGyre(base) }
func tipografiaCorpo(t *TipografiaOpts) string            { return latex.TipografiaCorpo(t) }
func tipografiaPreambulo(t *TipografiaOpts, engine string) string {
	return latex.TipografiaPreambulo(t, engine)
}
func trimFloat(f float64) string { return latex.TrimFloat(f) }
