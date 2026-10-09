package documento

import (
	"fmt"
	"strings"
	"time"

	"edicion/cas"
	"edicion/plantilla"
)

// Composición do .tex final coa plantilla do centro (vén de plantillas.go de
// Yang; o modelo de plantilla xa é o común de edicion/plantilla).

var paquetesMatexe = []plantilla.PaqueteRequirido{
	{Nome: "amsmath", Linha: `\usepackage{amsmath,amssymb}`},
	{Nome: "graphicx", Linha: `\usepackage{graphicx}`},
	{Nome: "tikz", Linha: `\usepackage{tikz}`},
	{Nome: "gnuplot-lua-tikz", Linha: `\usepackage{gnuplot-lua-tikz}`},
	{Nome: "circuitikz", Linha: `\usepackage{circuitikz}`},
	{Nome: "xcolor", Linha: `\usepackage{xcolor}`},
	{Nome: "longtable", Linha: "\\usepackage{longtable,booktabs,array,calc}\n\\makeatletter\\@ifundefined{c@none}{\\newcounter{none}}{}\\makeatother\n\\providecommand{\\tightlist}{\\setlength{\\itemsep}{0pt}\\setlength{\\parskip}{0pt}}"},
	{Nome: "mhchem", Linha: `\usepackage[version=4]{mhchem}`},
	{Nome: "chemfig", Linha: `\usepackage{chemfig}`},
	{Nome: "hyperref", Linha: `\usepackage[hidelinks]{hyperref}`},
}

var preambuloMatexe = plantilla.Preambulo{
	UnicodeNativo:       engineUnicodeNativo,
	DeclaracionsUnicode: func() string { return latexUnicodeDeclarations },
	Babel:               babelLine,
	Marca:               "% --- engadido por LuaYang ---",
}

// variablesAutomaticas son as que enche Yang soa en cada xeración. Unha
// variable definida na plantilla co mesmo nome ten prioridade (ex.: fixar
// un {{TITULO}} propio en vez do nome do ficheiro).
func variablesAutomaticas(nomeDoc string) map[string]string {
	nomeDoc = strings.TrimSpace(nomeDoc)
	return map[string]string{
		"DATA":     time.Now().Format("02/01/2006"),
		"FICHEIRO": nomeDoc,
		"TITULO":   nomeDoc,
		"ANO":      time.Now().Format("2006"),
	}
}

// baleiroLatex é o que se pon no sitio dunha variable SEN valor cando se
// está a compoñer LaTeX. Non pode ser a cadea baleira: unha liña de
// plantilla coma "{\\LARGE {{EMPRESA}} }\\\\[4pt]" quedaría cun \\\\ sen nada
// diante e pdflatex morre con "There's no line here to end" - un orzamento
// recén creado, coas variables aínda por encher, non compilaría. \\mbox{}
// non se ve no PDF pero xa é "algo" para LaTeX. En Markdown non fai falla
// nada disto (unha liña baleira alí non é erro), así que alí vai baleiro.
const baleiroLatex = `\mbox{}`

// As funcións de composición (substituír variables, paquetes declarados, motor da
// plantilla e copia de imaxes) viven en edicion/plantilla; as de abaixo son os
// adaptadores da aplicación.

// paqueteRequirido e paquetesMatexe viven en edicion_plantilla.go (adaptadores).

// inxectarPreambuloObrigatorio: adaptador (ver edicion/plantilla).

// documentoLatexConPlantilla compón o .tex final a partir do corpo xa
// xerado (o exame/documento pasado por Maxima) e da plantilla escollida.
// Con plantilla nil devolve exactamente o de sempre: preámbulo por defecto
// + corpo + \end{document}.
func documentoLatexConPlantilla(corpo, engine string, p *plantilla.Plantilla, nomeDoc string, tip *TipografiaOpts) (string, []string, error) {
	// tamaño/interliñado van xusto tras \begin{document}: aplícanse sempre,
	// aínda que a plantilla traia \documentclass propio. babelListquotFix
	// (latexdoc.go) vai igual de "sempre" - arranxa un choque real entre
	// babel galego/castelán e enumitem que rompe calquera <ol>/<ul> de dous
	// ou máis elementos ("Incomplete \iffalse"), tanto co preámbulo por
	// defecto coma cunha plantilla propia.
	corpo = babelListquotFix + tipografiaCorpo(tip) + corpo
	// familia de fonte: vai no preámbulo, canda a liña de babel. Só cando o
	// preámbulo o pon Yang - se a plantilla ten \documentclass propio, é ela
	// quen manda na fonte (só se lle aplica o tamaño/interliñado de arriba).
	fonte := tipografiaPreambulo(tip, engine)
	if p == nil {
		return fmt.Sprintf(preambuloPorDefecto(engine), babelLine()+fonte) + corpo + latexPostamble, nil, nil
	}
	if !strings.Contains(p.Latex, plantilla.MarcadorCorpo) {
		return "", nil, fmt.Errorf("a plantilla %q non leva %s", p.Nome, plantilla.MarcadorCorpo)
	}
	texto, avisos := plantilla.SubstituirVariables(p.Latex, p, variablesAutomaticas(nomeDoc), baleiroLatex, cas.EscapeLatexProse)
	// O corpo insérese DESPOIS de substituír as variables: así un {{...}}
	// que aparecese no propio documento do profesorado queda intacto.
	doc := strings.ReplaceAll(texto, plantilla.MarcadorCorpo, corpo)
	if strings.Contains(doc, `\documentclass`) {
		return plantilla.InxectarPreambuloObrigatorio(doc, engine, paquetesMatexe, "", preambuloMatexe), avisos, nil
	}
	// Fragmento: a plantilla é só cabeceira/pé, o preámbulo pono Yang.
	return fmt.Sprintf(preambuloPorDefecto(engine), babelLine()+fonte) + doc + latexPostamble, avisos, nil
}
