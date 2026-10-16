package main

// DESEÑO DO PDF COMA DATOS → LaTeX.
//
// A plantilla garda o deseño coma datos (Deseno, edicion/plantilla/deseno.go)
// e aquí sae o LaTeX: letras, tamaño, interliñado, marxes, cores e cada
// elemento do documento (títulos, etiquetas, meses, trimestres, tarefas,
// actas, sinaturas). Un elemento pode levar o seu propio LaTeX: substitúe só
// esa parte.
//
// Prioridade: o Estilo (LaTeX libre) da plantilla > o seu Deseno > os
// «Axustes do texto» da xanela principal > o deseño de Lúa.
//
// Os macros (\chip, \tarefa, \trimestre, \actacab...) son os que usa o
// contido que xera Lúa; os seus nomes e argumentos non cambian aínda que cambie
// o deseño. Cada elemento deixa unha marca de posición (\luamarca) que a vista
// previa usa para saber que elemento hai debaixo do rato (deseno_marcas.go).

import (
	"edicion/idioma"
	"edicion/latex"
	"edicion/plantilla"
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Valores por defecto do deseño de Lúa.
const (
	tamanoDesenoDefecto   = 11.0
	marxeDesenoDefecto    = 20.0
	parrafoDesenoDefecto  = 0.55
	corPrincipalDefecto   = "4F46E5" // tc-indigo
	corEscuraDefecto      = "3730A3" // tt-indigo
	corSuaveDefecto       = "C7D2FE" // tb-indigo
	contrasteMinimoTextos = 4.5      // WCAG AA para texto normal
)

// desenoResolto é o deseño xa con todo decidido: valores por defecto,
// interruptores aplicados e cores corrixidas.
type desenoResolto struct {
	d           Deseno
	letras      latex.Letras
	tamano      float64 // pt do corpo
	k           float64 // tamano / 11: escala de todo
	kt          float64 // k * escala dos títulos
	interlinado float64
	parrafo     float64
	marxe       float64
	principal   string
	escura      string
	suave       string
	texto       string
	// libre: a plantilla ten Estilo en LaTeX libre (manda el).
	libre bool
	// tamanoNoCorpo: o tamaño non se puido poñer no preámbulo (falta scrextend).
	tamanoNoCorpo bool
	// fontes: as letras que leva Lúa e que usa o documento.
	fontes []string
	avisos []string
	// arranxos: o que o corrector cambiou por conta propia.
	arranxos []string
}

// desenoDaPlantilla devolve o deseño que se aplica: o da plantilla ou, se non
// ten, o que din os Axustes do texto (tip).
func desenoDaPlantilla(p *PlantillaLatex, tip *TipografiaOpts) (Deseno, []string) {
	var avisos []string
	tipCheo := tip != nil && (strings.TrimSpace(tip.Fonte) != "" || tip.TamanoPt > 0 || tip.Interlinado > 0)
	if p != nil && p.Deseno != nil {
		if tipCheo {
			avisos = append(avisos, idioma.T("go.deseno.axustes_ignorados", "a plantilla ten o seu deseño: os «Axustes do texto» non se aplican (cambia a letra e o tamaño no editor da plantilla)"))
		}
		return *normalizarDeseno(p.Deseno), avisos
	}
	d := Deseno{}
	if tip != nil {
		comb, sistema := latex.CombinacionDePreset(tip.Fonte)
		d.Letra, d.LetraCorpo = comb, sistema
		d.Tamano, d.Interlinado = tip.TamanoPt, tip.Interlinado
	}
	return *normalizarDeseno(&d), avisos
}

// resolverDeseno aplica valores por defecto, interruptores e corrector.
func resolverDeseno(p *PlantillaLatex, tip *TipografiaOpts) desenoResolto {
	d, avisos := desenoDaPlantilla(p, tip)
	r := desenoResolto{d: d, avisos: avisos}
	r.libre = p != nil && strings.TrimSpace(p.Estilo) != ""
	r.letras = latex.LetrasDe(d.Letra, d.LetraCorpo, d.LetraTitulos, d.LetraEtiquetas, d.LetraMates)
	r.tamano = valorOu(d.Tamano, tamanoDesenoDefecto)
	r.interlinado = valorOu(d.Interlinado, 1)
	r.parrafo = valorOu(d.Parrafo, parrafoDesenoDefecto)
	r.marxe = valorOu(d.Marxe, marxeDesenoDefecto)
	escala := valorOu(d.Escala, 1)

	// Interruptores do mundo real: cambian o resultado, non os valores
	// gardados (ao apagalos todo volve ao de antes).
	if d.AforrarPapel {
		r.marxe = math.Min(r.marxe, 14)
		r.parrafo = math.Min(r.parrafo, 0.25)
		r.interlinado = math.Min(r.interlinado, 1)
		escala *= 0.88
	}
	if d.LecturaFacil {
		r.tamano = math.Max(r.tamano, 13)
		r.interlinado = math.Max(r.interlinado, 1.35)
		r.parrafo = math.Max(r.parrafo, 0.9)
	}
	r.k = r.tamano / tamanoDesenoDefecto
	r.kt = r.k * escala

	r.principal = valorTxt(d.CorPrincipal, corPrincipalDefecto)
	switch {
	case d.CorEscura != "":
		r.escura = d.CorEscura
	case d.CorPrincipal != "":
		r.escura = mesturarCor(r.principal, "000000", 0.3)
	default:
		r.escura = corEscuraDefecto
	}
	if d.CorPrincipal != "" {
		r.suave = mesturarCor(r.principal, "FFFFFF", 0.7)
	} else {
		r.suave = corSuaveDefecto
	}
	r.texto = valorTxt(d.CorTexto, "000000")
	if d.LecturaFacil {
		r.texto = "000000"
	}

	// Corrector: os textos de cor teñen que lerse sobre branco.
	r.escura = r.corrixirContraste(r.escura, idioma.T("go.deseno.que_rotulos", "dos rótulos"))
	r.texto = r.corrixirContraste(r.texto, idioma.T("go.deseno.que_texto", "do texto"))
	for id, e := range d.Elementos {
		if e.Cor != "" {
			e.Cor = r.corrixirContraste(e.Cor, idioma.T("go.deseno.que_elemento", "do elemento «%s»", id))
			r.d.Elementos[id] = e
		}
	}
	return r
}

func valorOu(v, def float64) float64 {
	if v <= 0 {
		return def
	}
	return v
}

func valorTxt(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}

// corrixirContraste escurece unha cor de texto ata que se le ben sobre branco
// (WCAG AA) e anótao.
func (r *desenoResolto) corrixirContraste(cor, que string) string {
	if contrasteCor(cor, "FFFFFF") >= contrasteMinimoTextos {
		return cor
	}
	orixe := cor
	for i := 0; i < 20 && contrasteCor(cor, "FFFFFF") < contrasteMinimoTextos; i++ {
		cor = mesturarCor(cor, "000000", 0.12)
	}
	r.arranxos = append(r.arranxos, idioma.T("go.deseno.escurecida", "escurecín a cor %s (#%s → #%s) para que se lea ben", que, orixe, cor))
	return cor
}

// elemento devolve o axuste dun elemento.
func (r desenoResolto) elemento(id string) ElementoDeseno {
	return r.d.Elementos[id]
}

// pt devolve un tamaño de letra escalado co corpo.
func (r desenoResolto) pt(v float64) string { return trimFloat(v * r.k) }

// ptT devolve un tamaño de título (escala dos títulos e do elemento).
func (r desenoResolto) ptT(id string, v float64) string {
	return trimFloat(v * r.kt * valorOu(r.elemento(id).Escala, 1))
}

// mm devolve unha medida en mm dun elemento escalada coma a súa letra (as
// caixas das etiquetas e dos meses medran co texto, se non o texto sae fóra).
func (r desenoResolto) mm(id string, v float64) string {
	return trimFloat(v*r.k*valorOu(r.elemento(id).Escala, 1)) + "mm"
}

// ptE devolve un tamaño dun elemento que non é título.
func (r desenoResolto) ptE(id string, v float64) string {
	return trimFloat(v * r.k * valorOu(r.elemento(id).Escala, 1))
}

// corElemento devolve a cor LaTeX dun elemento ("" se non ten propia: daquela
// vai `def`).
func (r desenoResolto) corElemento(id, def string) string {
	if c := r.elemento(id).Cor; c != "" {
		return "lua-el-" + strings.ReplaceAll(id, "_", "-")
	}
	return def
}

// maius envolve o texto en \MakeUppercase se o elemento o pide.
func (r desenoResolto) maius(id, texto string) string {
	if r.elemento(id).Maiusculas {
		return `\MakeUppercase{` + texto + `}`
	}
	return texto
}

// filete devolve o LaTeX do filete ou "" se o elemento vai sen el.
func (r desenoResolto) filete(id, latexFilete string) string {
	if r.elemento(id).SenFilete {
		return ""
	}
	return latexFilete
}

// ---------- Cores ----------

func hexARGB(h string) (float64, float64, float64) {
	h = strings.TrimPrefix(h, "#")
	if len(h) != 6 {
		return 0, 0, 0
	}
	v, err := strconv.ParseUint(h, 16, 32)
	if err != nil {
		return 0, 0, 0
	}
	return float64(v>>16&0xFF) / 255, float64(v>>8&0xFF) / 255, float64(v&0xFF) / 255
}

func rgbAHex(r, g, b float64) string {
	c := func(x float64) int { return int(math.Round(math.Max(0, math.Min(1, x)) * 255)) }
	return fmt.Sprintf("%02X%02X%02X", c(r), c(g), c(b))
}

// mesturarCor mestura `a` con `b` (t = canto de b).
func mesturarCor(a, b string, t float64) string {
	ar, ag, ab := hexARGB(a)
	br, bg, bb := hexARGB(b)
	return rgbAHex(ar+(br-ar)*t, ag+(bg-ag)*t, ab+(bb-ab)*t)
}

func luminancia(h string) float64 {
	r, g, b := hexARGB(h)
	lin := func(c float64) float64 {
		if c <= 0.03928 {
			return c / 12.92
		}
		return math.Pow((c+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(r) + 0.7152*lin(g) + 0.0722*lin(b)
}

// contrasteCor é a razón de contraste WCAG entre dúas cores.
func contrasteCor(a, b string) float64 {
	la, lb := luminancia(a), luminancia(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// gris converte unha cor no gris da mesma luminosidade (branco e negro).
func gris(h string) string {
	r, g, b := hexARGB(h)
	y := 0.299*r + 0.587*g + 0.114*b
	return rgbAHex(y, y, y)
}

// ---------- LaTeX ----------

// latexPreambuloDeseno: o que vai no preámbulo de Lúa xusto despois de babel:
// tamaño do corpo (scrextend redefine \normalsize e todos os tamaños
// relativos), interliñado e letras.
func (r *desenoResolto) latexPreambulo(engine string) (string, []string) {
	var b strings.Builder
	if math.Abs(r.tamano-tamanoDesenoDefecto) > 0.01 {
		if latex.Dispoñible("scrextend.sty") {
			fmt.Fprintf(&b, "\\usepackage[fontsize=%spt]{scrextend}\n", trimFloat(r.tamano))
		} else {
			// Sen KOMA-Script (TeX Live mínimo): o tamaño vai tras
			// \begin{document}, coma antes (non chega a \normalsize).
			r.tamanoNoCorpo = true
		}
	}
	if math.Abs(r.interlinado-1) > 0.001 {
		fmt.Fprintf(&b, "\\linespread{%s}\n", trimFloat(r.interlinado))
	}
	x := latex.XerarLetras(r.letras, engine)
	b.WriteString(x.Preambulo)
	b.WriteString(x.Mates)
	r.fontes = x.Ficheiros
	avisos := x.Avisos
	if x.Sistema {
		avisos = append(avisos, idioma.T("go.deseno.letra_sistema", "o deseño usa unha letra do sistema: noutro equipo sen ela o PDF non sairá igual"))
	}
	return b.String(), avisos
}

// latexCores define as cores do deseño e a paleta das tarefas e dos meses
// (en gris se o deseño é para branco e negro).
func (r desenoResolto) latexCores() string {
	c := func(h string) string {
		if r.d.BrancoNegro {
			return gris(h)
		}
		return h
	}
	var p strings.Builder
	for _, t := range paletaTarefasCrono {
		fmt.Fprintf(&p, "\\definecolor{tf-%s}{HTML}{%s}\\definecolor{tt-%s}{HTML}{%s}\\definecolor{tb-%s}{HTML}{%s}\\definecolor{tc-%s}{HTML}{%s}\n",
			t.Clave, c(t.Fondo), t.Clave, c(t.Texto), t.Clave, c(t.Bordo), t.Clave, c(vivosTarefasCrono[t.Clave]))
	}
	for i, m := range coresMesCrono {
		fmt.Fprintf(&p, "\\definecolor{mes-%d}{HTML}{%s}\\definecolor{mesf-%d}{HTML}{%s}\n", i, c(m), i, c(coresMesVivasCrono[i]))
	}
	fmt.Fprintf(&p, "\\definecolor{lua-principal}{HTML}{%s}\\definecolor{lua-escura}{HTML}{%s}\\definecolor{lua-suave}{HTML}{%s}\\definecolor{lua-texto}{HTML}{%s}\n",
		c(r.principal), c(r.escura), c(r.suave), c(r.texto))
	for _, e := range ElementosDesenoIDs() {
		if cor := r.elemento(e).Cor; cor != "" {
			fmt.Fprintf(&p, "\\definecolor{lua-el-%s}{HTML}{%s}\n", strings.ReplaceAll(e, "_", "-"), c(cor))
		}
	}
	fmt.Fprintf(&p, "\\definecolor{gris-linea}{HTML}{%s}\n\\definecolor{gris-texto}{HTML}{%s}\n", c("E7E5E4"), c("44403C"))
	return p.String()
}

// latexMarcas define \luamarca: deixa en doc.lpos a páxina e a posición de
// cada elemento (deseno_marcas.go). Sen \pdfsavepos (raro) non fai nada.
const latexMarcas = `\makeatletter
\ifdefined\pdfsavepos\else\ifdefined\savepos\let\pdfsavepos\savepos\let\pdflastxpos\lastxpos\let\pdflastypos\lastypos\fi\fi
\ifdefined\pdfsavepos
  \newwrite\lua@pos\immediate\openout\lua@pos=\jobname.lpos
  \AtBeginDocument{\immediate\write\lua@pos{paxina \the\paperwidth\space\the\paperheight}}
  \newcommand{\luamarca}[1]{\pdfsavepos\write\lua@pos{#1 \thepage\space\the\pdflastxpos\space\the\pdflastypos}}
\else
  \newcommand{\luamarca}[1]{}
\fi
\makeatother
`

// ElementosDesenoIDs devolve os ids dos elementos, na orde da interface.
func ElementosDesenoIDs() []string {
	out := make([]string, 0, len(plantilla.ElementosDeseno))
	for _, e := range plantilla.ElementosDeseno {
		out = append(out, e.ID)
	}
	return out
}

func elementosDeseno() []string { return ElementosDesenoIDs() }

// latexElemento devolve o LaTeX xerado dun elemento (sen ter en conta o
// LaTeX propio que poida levar).
func (r desenoResolto) latexElemento(id string) string {
	switch id {
	case "texto":
		return r.latexTexto()
	case "titulos":
		return r.latexTitulos()
	case "etiquetas":
		return r.latexEtiquetas()
	case "meses":
		return r.latexMeses()
	case "trimestres":
		return r.latexTrimestres()
	case "tarefas":
		return r.latexTarefas()
	case "actas_cabeceira":
		return r.latexActasCabeceira()
	case "actas_apartados":
		return r.latexActasApartados()
	case "actas_materias":
		return r.latexActasMaterias()
	case "sinaturas":
		return r.latexSinaturas()
	}
	return ""
}

// latexEstilo é o deseño enteiro: cada elemento, o xerado ou o propio.
func (r desenoResolto) latexEstilo() string {
	var b strings.Builder
	b.WriteString("% DESEÑO de Lúa, xerado a partir do deseño da plantilla.\n")
	for _, id := range elementosDeseno() {
		propio := strings.TrimSpace(r.elemento(id).Latex)
		fmt.Fprintf(&b, "%% ── %s ──\n", id)
		if propio != "" {
			b.WriteString(propio + "\n")
			continue
		}
		b.WriteString(r.latexElemento(id))
	}
	return b.String()
}

func (r desenoResolto) latexTexto() string {
	var b strings.Builder
	fmt.Fprintf(&b, "\\setlength{\\parindent}{0pt}\n\\setlength{\\parskip}{%sem}\n", trimFloat(r.parrafo))
	b.WriteString("\\setlist{itemsep=1pt,topsep=2pt,leftmargin=*}\n\\setcounter{tocdepth}{2}\n\\urlstyle{same}\n")
	// Corrector: nin a primeira nin a última liña dun parágrafo soltas
	// noutra páxina.
	b.WriteString("\\clubpenalty=10000\n\\widowpenalty=10000\n\\displaywidowpenalty=10000\n")
	if r.texto != "000000" {
		b.WriteString("\\AtBeginDocument{\\color{lua-texto}}\n")
	}
	if r.d.LecturaFacil {
		// Lectura fácil: sen xustificar e sen cortar palabras.
		b.WriteString("\\AtBeginDocument{\\raggedright}\n\\hyphenpenalty=10000\n\\exhyphenpenalty=10000\n")
	}
	return b.String()
}

func (r desenoResolto) latexTitulos() string {
	cor := r.corElemento("titulos", "")
	corTxt := ""
	if cor != "" {
		corTxt = `\color{` + cor + `}`
	}
	antes := `\luamarca{titulos}`
	if r.elemento("titulos").Maiusculas {
		antes += `\MakeUppercase`
	}
	fil := r.filete("titulos", `[{\vspace{1.2mm}\color{lua-principal}\rule{16mm}{2pt}}]`)
	return fmt.Sprintf(`\titleformat{\section}{\sffamily\fontsize{%s}{%s}\selectfont\bfseries%s}{\thesection.}{0.5em}{%s}%s
\titleformat{\subsection}{\sffamily\fontsize{%s}{%s}\selectfont\bfseries%s}{\thesubsection.}{0.5em}{%s}
\titlespacing*{\section}{0pt}{0pt}{%spt}
\titlespacing*{\subsection}{0pt}{%spt}{%spt}
`, r.ptT("titulos", 20), r.ptT("titulos", 24), corTxt, antes, fil,
		r.ptT("titulos", 14), r.ptT("titulos", 17), corTxt, antes,
		r.pt(12), r.pt(14), r.pt(4))
}

func (r desenoResolto) latexEtiquetas() string {
	var estilo string
	switch r.d.Etiquetas {
	case "planas":
		estilo = `fill=tc-#1,text=white`
	case "contorno":
		estilo = `draw=tc-#1,line width=0.7pt,fill=tf-#1,text=tt-#1`
	default:
		estilo = `draw=none,left color=tc-#1,right color=tt-#1,text=white`
	}
	texto := r.maius("etiquetas", "#2")
	return fmt.Sprintf(`\ifdefined\chipbox\else\newsavebox{\chipbox}\fi
\newlength{\chippad}\setlength{\chippad}{%[7]s}
\newlength{\chipmax}
\newlength{\chiph}
\newcommand{\chipsize}{\fontsize{%[1]s}{%[2]s}\selectfont}
\tikzset{chipstyle/.style={%[3]s,rounded corners=%[8]s,inner xsep=\chippad,inner ysep=1.5pt,minimum height=%[9]s,font=\linespread{1}\fontchip\chipsize\bfseries}}
%% Etiqueta (píldora) dunha tarefa no cronograma anual. Se o título non cabe
%% nunha liña, pártese dentro da propia etiqueta. O puntal invisible separa as
%% liñas de etiquetas.
\newcommand{\chip}[2]{%%
  \begingroup\fontchip\chipsize\bfseries\luamarca{etiquetas}%%
  \sbox{\chipbox}{%[4]s}%%
  \setlength{\chipmax}{\dimexpr\linewidth-2\chippad-1pt\relax}%%
  \ifdim\wd\chipbox>\chipmax
    \setbox\chipbox\vbox{\hsize=\chipmax\raggedright %[4]s\par}%%
    \setlength{\chiph}{\dimexpr(\ht\chipbox+\dp\chipbox)/2+1.5pt+2mm\relax}%%
    \tikz[baseline=(chipn.center)]\node[chipstyle={#1},text width=\chipmax,align=left](chipn){%[4]s};%%
  \else
    \setlength{\chiph}{%[10]s}%%
    \tikz[baseline=(chipn.center)]\node[chipstyle={#1}](chipn){%[4]s};%%
  \fi
  \rule[-\chiph]{0pt}{\dimexpr2\chiph\relax}%%
  \endgroup}
`, r.ptE("etiquetas", 7.5), r.ptE("etiquetas", 9), estilo, texto, "", "", r.mm("etiquetas", 3.2), r.mm("etiquetas", 2.6), r.mm("etiquetas", 5.2), r.mm("etiquetas", 4.6))
}

func (r desenoResolto) latexMeses() string {
	n1, n2 := r.ptE("meses", 8), r.ptE("meses", 9)
	c1, c2 := r.ptE("meses", 5.5), r.ptE("meses", 6)
	return fmt.Sprintf(`%% Etiqueta do mes: cor forte co nome e, debaixo, cantas tarefas ten; sen
%% tarefas, en gris. Todas co ancho do nome de mes máis longo (\meswidth).
\ifdefined\chipbox\else\newsavebox{\chipbox}\fi
\newlength{\meswidth}
\tikzset{mesbadge/.style={rounded corners=%[5]s,minimum width=\meswidth,minimum height=%[6]s,inner xsep=%[7]s,inner ysep=1mm,text=white,align=center,font=\fontchip\linespread{1}}}
\newcommand{\medirmes}[2]{\sbox{\chipbox}{\tikz\node[mesbadge,minimum width=0pt]{{\bfseries\fontsize{%[1]s}{%[2]s}\selectfont #1}\ifx\relax#2\relax\else\\[-0.3mm]{\fontsize{%[3]s}{%[4]s}\selectfont #2}\fi};}%%
  \ifdim\wd\chipbox>\meswidth\setlength{\meswidth}{\wd\chipbox}\fi}
\newcommand{\mesbadge}[3]{\luamarca{meses}\tikz[baseline=(mb.center)]\node[mesbadge,fill=mesf-#1](mb){{\bfseries\fontsize{%[1]s}{%[2]s}\selectfont #2}\ifx\relax#3\relax\else\\[-0.3mm]{\fontsize{%[3]s}{%[4]s}\selectfont #3}\fi};}
\newcommand{\mesvacio}[2]{\luamarca{meses}\tikz[baseline=(mb.center)]\node[mesbadge,fill=black!6,text=black!55](mb){{\bfseries\fontsize{%[1]s}{%[2]s}\selectfont #2}};}
`, n1, n2, c1, c2, r.mm("meses", 2.4), r.mm("meses", 9.5), r.mm("meses", 3))
}

func (r desenoResolto) latexTrimestres() string {
	cor := r.corElemento("trimestres", "lua-escura")
	fil := r.filete("trimestres", `\vspace{0.8mm}\noindent{\color{lua-suave}\rule{\linewidth}{0.8pt}}\par`)
	return fmt.Sprintf(`%% Un trimestre do cronograma anual: nome, meses, liña e táboa. Todo nunha
%% caixa que non se parte: se non cabe, pasa enteiro á páxina seguinte.
\newenvironment{trimestre}[2]{\par\vspace{5mm}\noindent\begin{minipage}{\linewidth}%%
  \luamarca{trimestres}{\fontchip\bfseries\fontsize{%s}{%s}\selectfont\color{%s}%s}\hspace{2.5mm}{\fontchip\fontsize{%s}{%s}\selectfont\color{gris-texto}#2}\par
  %s\vspace{0.5mm}}{\end{minipage}\par}
`, r.ptT("trimestres", 10), r.ptT("trimestres", 12), cor, r.maius("trimestres", "#1"), r.pt(8), r.pt(10), fil)
}

func (r desenoResolto) latexTarefas() string {
	corTxt := ""
	if c := r.corElemento("tarefas", ""); c != "" {
		corTxt = `\color{` + c + `}`
	}
	return fmt.Sprintf(`%% Título dunha tarefa no desenvolvemento: cadradiño da súa cor + negriña.
\newcommand{\tarefa}[2]{\par\vspace{0.8ex}{\sffamily\fontsize{%s}{%s}\selectfont\bfseries\luamarca{tarefas}\textcolor{tc-#1}{\rule{2.4mm}{2.4mm}}\hspace{2mm}%s#2\par}\nopagebreak[4]\vspace{0.2ex}}
`, r.ptT("tarefas", 12), r.ptT("tarefas", 14.5), corTxt)
}

func (r desenoResolto) latexActasCabeceira() string {
	cor := r.corElemento("actas_cabeceira", "lua-escura")
	fil := r.filete("actas_cabeceira", `\vspace{1.5mm}\noindent{\color{lua-principal}\rule{16mm}{2pt}}\par`)
	return fmt.Sprintf(`%% Cabeceira dunha acta: rótulo pequeno, título, subtítulo e un filete.
\newcommand{\actacab}[3]{\par\noindent\luamarca{actas_cabeceira}{\fontchip\bfseries\fontsize{%s}{%s}\selectfont\color{%s}\MakeUppercase{#1}}\par\vspace{1mm}\noindent{\sffamily\bfseries\fontsize{%s}{%s}\selectfont #2\par}\vspace{0.5mm}\noindent{\fontchip\fontsize{%s}{%s}\selectfont\color{gris-texto}#3\par}%s\vspace{2mm}}
%% Un dato da acta (data, hora, lugar): rótulo en negriña e valor.
\newcommand{\actadato}[2]{\noindent{\fontchip\bfseries\fontsize{%s}{%s}\selectfont\color{gris-texto}\makebox[22mm][l]{#1}}#2\par}
`, r.pt(8), r.pt(10), cor, r.ptT("actas_cabeceira", 22), r.ptT("actas_cabeceira", 26), r.pt(10.5), r.pt(13), fil, r.pt(9), r.pt(12))
}

func (r desenoResolto) latexActasApartados() string {
	cor := r.corElemento("actas_apartados", "lua-escura")
	fil := r.filete("actas_apartados", `\nopagebreak\vspace{0.6mm}\nopagebreak\noindent{\color{lua-suave}\rule{\linewidth}{0.8pt}}\par`)
	return fmt.Sprintf(`%% Título dun apartado da acta (asistentes, orde do día, acordos...).
\newcommand{\actasecc}[1]{\par\vspace{3.5mm}\noindent\luamarca{actas_apartados}{\fontchip\bfseries\fontsize{%s}{%s}\selectfont\color{%s}%s}\par%s\nopagebreak\vspace{1mm}}
%% Un campo do seguimento dunha materia (rótulo pequeno e o texto debaixo).
\newcommand{\actacampo}[2]{\par\smallskip\noindent{\fontchip\bfseries\fontsize{%s}{%s}\selectfont\color{gris-texto}#1}\par\nopagebreak\noindent#2}
%% Grao de cumprimento: rótulo, barra e porcentaxe (0-100).
\newcommand{\actaprogreso}[2]{\par\noindent{\fontchip\bfseries\fontsize{%s}{%s}\selectfont\color{gris-texto}#1}\hspace{3mm}\tikz[baseline=-0.6ex]{\fill[black!8,rounded corners=1pt](0,-0.09) rectangle (4,0.09);\fill[lua-principal,rounded corners=1pt](0,-0.09) rectangle ({0.04*#2},0.09);}\hspace{2mm}{\fontchip\bfseries\small #2\,\%%}\par}
`, r.ptT("actas_apartados", 10.5), r.ptT("actas_apartados", 13), cor, r.maius("actas_apartados", "#1"), fil, r.pt(8.5), r.pt(11), r.pt(8.5), r.pt(11))
}

func (r desenoResolto) latexActasMaterias() string {
	cor := r.corElemento("actas_materias", "lua-principal")
	return fmt.Sprintf(`%% Cabeceira do seguimento dunha materia: cadradiño, nome e, á dereita, docente e grupos.
\newcommand{\actamateria}[2]{\par\vspace{2.4mm}\noindent\luamarca{actas_materias}{\sffamily\bfseries\fontsize{%s}{%s}\selectfont\textcolor{%s}{\rule{2.4mm}{2.4mm}}\hspace{2mm}#1}\hfill{\fontchip\small\color{gris-texto}#2}\par\nopagebreak[4]\vspace{0.4mm}}
`, r.ptT("actas_materias", 12), r.ptT("actas_materias", 14.5), cor)
}

func (r desenoResolto) latexSinaturas() string {
	a, b := r.ptE("sinaturas", 8.5), r.ptE("sinaturas", 10)
	c, d := r.ptE("sinaturas", 10), r.ptE("sinaturas", 12)
	col := func(n1, n2 string) string {
		return fmt.Sprintf(`\begin{minipage}[t]{0.47\linewidth}\centering{\fontchip\fontsize{%s}{%s}\selectfont\color{gris-texto}#%s\par}\vspace{14mm}{\color{black!40}\rule{55mm}{0.4pt}}\par\vspace{1.5mm}{\fontchip\bfseries\fontsize{%s}{%s}\selectfont #%s\par}\end{minipage}`, a, b, n1, c, d, n2)
	}
	return `% Sinaturas do peche da acta: quen redacta e a xefatura, cada un coa súa liña.
\newcommand{\actasinaturas}[4]{\par\vspace{9mm}\noindent\begin{minipage}{\linewidth}\luamarca{sinaturas}%
  ` + col("1", "2") + `\hfill
  ` + col("3", "4") + `
\end{minipage}\par}
`
}

// latexDinamico son as definicións que o deseño (o xerado ou o LaTeX libre)
// dá por feitas: cores, \luamarca e \logocentro.
func (r desenoResolto) latexDinamico(logo *imaxeCrono) string {
	var p strings.Builder
	p.WriteString(r.latexCores())
	p.WriteString(latexMarcas)
	// O logotipo do curso: co macro as plantillas non teñen que saber onde
	// está o ficheiro nin que pasa se o curso non ten logotipo.
	p.WriteString("\\newif\\ifhaylogocentro\n")
	if logo != nil {
		p.WriteString("\\haylogocentrotrue\n\\newcommand{\\logocentro}[1][]{\\includegraphics[#1]{" + logo.Ficheiro + "}}\n")
	} else {
		p.WriteString("\\haylogocentrofalse\n\\newcommand{\\logocentro}[1][]{}\n")
	}
	return p.String()
}

// desenoDoc é todo o LaTeX que o deseño achega a un documento.
type desenoDoc struct {
	// Preambulo vai despois de babel no preámbulo de Lúa (tamaño, interliñado, letras).
	Preambulo string
	// Bloque vai despois dos paquetes: cores, marcas, logotipo e estilo.
	Bloque string
	// BloqueCompleta é o que se inxecta nunha plantilla con \documentclass
	// propio: as letras só se ela non fixa as súas.
	BloqueCompleta string
	// Corpo vai xusto despois de \begin{document}.
	Corpo string
	// Marxe en mm para geometry.
	Marxe float64
	// Fontes que hai que deixar en fonts/ (escribirFontesInformeCrono).
	Fontes []string
	// PortadaDef é a portada do modelo do deseño; SenPortada, que non leva.
	PortadaDef string
	SenPortada bool
	// CabeceiraPe vai ao principio do corpo (con {{VARIABLES}} sen substituír).
	CabeceiraPe string
	Avisos      []string
}

// compoñerDeseno resolve o deseño dunha plantilla e devolve as súas partes.
func compoñerDeseno(engine string, tip *TipografiaOpts, logo *imaxeCrono, p *PlantillaLatex) desenoDoc {
	r := resolverDeseno(p, tip)
	pre, avisosLetras := r.latexPreambulo(engine)
	estilo := r.latexEstilo()
	if r.libre {
		estilo = p.Estilo + "\n"
	}
	bloque := r.latexDinamico(logo) + estilo
	dd := desenoDoc{Preambulo: pre, Bloque: bloque, Marxe: r.marxe, Fontes: r.fontes,
		PortadaDef: r.latexPortada(), SenPortada: r.d.Portada == "sen", CabeceiraPe: r.latexCabeceiraPe()}
	if r.tamanoNoCorpo {
		dd.Corpo = tipografiaCorpo(&TipografiaOpts{TamanoPt: r.tamano, Interlinado: siDistinto(r.interlinado, 1)})
	}
	dd.Avisos = append(dd.Avisos, r.avisos...)
	dd.Avisos = append(dd.Avisos, avisosLetras...)
	if len(r.arranxos) > 0 {
		dd.Avisos = append(dd.Avisos, idioma.T("go.deseno.arranxos", "Arranxei %d cousa(s) do deseño: %s", len(r.arranxos), strings.Join(r.arranxos, "; ")))
	}

	// Plantilla completa: o preámbulo é dela. As letras inxéctanse só se non
	// fixa as súas (e avísase se as fixa e alguén escolleu outras); o tamaño
	// e o interliñado van tras \begin{document}, porque a súa clase pode non
	// aceptar scrextend.
	if p != nil && strings.Contains(p.Latex, `\documentclass`) {
		preambuloPlantilla := p.Latex
		if i := strings.Index(preambuloPlantilla, `\begin{document}`); i >= 0 {
			preambuloPlantilla = preambuloPlantilla[:i]
		}
		x := latex.XerarLetras(r.letras, engine)
		letras := x.Preambulo + x.Mates
		if latex.DeclaraLetras(preambuloPlantilla) {
			letras = "\\providecommand{\\fontchip}{\\sffamily}\n"
			if r.d.Letra != "" || r.d.LetraCorpo != "" || (tip != nil && strings.TrimSpace(tip.Fonte) != "") {
				dd.Avisos = append(dd.Avisos, idioma.T("go.deseno.plantilla_fixa_letra", "a plantilla fixa a súa propia letra no LaTeX: a letra escollida no deseño non se aplica"))
			}
		}
		dd.BloqueCompleta = letras + bloque
		dd.Corpo = tipografiaCorpo(&TipografiaOpts{TamanoPt: siDistinto(r.tamano, tamanoDesenoDefecto), Interlinado: siDistinto(r.interlinado, 1)})
	}
	return dd
}

func siDistinto(v, def float64) float64 {
	if math.Abs(v-def) < 0.001 {
		return 0
	}
	return v
}

// EstiloElementoDeseno devolve o LaTeX que Lúa xera para un elemento co
// deseño dado: o punto de partida de «Editar en LaTeX» no editor.
func (a *App) EstiloElementoDeseno(d Deseno, id string) string {
	d.Elementos = nil
	p := &PlantillaLatex{Deseno: &d}
	return resolverDeseno(p, nil).latexElemento(id)
}

// CombinacionsLetras devolve as combinacións de letras para a interface.
func (a *App) CombinacionsLetras() []latex.Combinacion {
	return latex.Combinacions()
}

// ofimaticaDoDeseno enche o que o bloque de Word/OpenDocument deixa baleiro
// coa letra, o tamaño, o interliñado e a cor do deseño do PDF: así un mesmo
// deseño sae parecido en todos os formatos. O que estea escrito no bloque de
// Word manda.
func ofimaticaDoDeseno(o OfimaticaPlantilla, d *Deseno) OfimaticaPlantilla {
	if d == nil {
		return o
	}
	if o.Fonte == "" {
		letras := latex.LetrasDe(d.Letra, d.LetraCorpo, d.LetraTitulos, d.LetraEtiquetas, d.LetraMates)
		if f, ok := latex.FamiliaPorID(letras.Corpo); ok {
			o.Fonte = f.Word
		} else if letras.Corpo != "" {
			o.Fonte = letras.Corpo // fonte do sistema
		}
	}
	if o.TamanoCorpo == 0 && d.Tamano > 0 {
		o.TamanoCorpo = d.Tamano
	}
	if o.Interlinado == 0 && d.Interlinado > 0 {
		o.Interlinado = d.Interlinado
	}
	if o.CorPrincipal == "" && d.CorPrincipal != "" {
		o.CorPrincipal = d.CorPrincipal
		if o.CorSecundaria == "" {
			o.CorSecundaria = mesturarCor(d.CorPrincipal, "000000", 0.3)
		}
	}
	if o.CorTexto == "" && d.CorTexto != "" {
		o.CorTexto = d.CorTexto
	}
	if strings.TrimSpace(o.Cabeceira) == "" {
		o.Cabeceira = zonasAOfimatica(d.Cabeceira)
	}
	if strings.TrimSpace(o.Pe) == "" {
		o.Pe = zonasAOfimatica(d.Pe)
	}
	return o
}

//go:embed skills/deseno-documentos/SKILL.md
var skillDeseno string

// SkillDeseno devolve a «skill» de deseño para calquera IA: as regras
// (skills/deseno-documentos/SKILL.md) e o JSON Schema do deseño. O editor
// cópiaa xunto co deseño actual para levalo a outra IA.
func (a *App) SkillDeseno() string {
	esq, _ := json.MarshalIndent(plantilla.EsquemaDeseno(), "", "  ")
	return strings.TrimSpace(skillDeseno) + "\n\n## Esquema JSON do deseño\n\n```json\n" + string(esq) + "\n```\n"
}

// esqDeseno é o esquema do campo deseno (o mesmo que documenta a skill de deseño).
func esqDeseno() map[string]any {
	e := plantilla.EsquemaDeseno()
	e["description"] = "DESEÑO DO PDF coma datos (o camiño recomendado para letra, tamaño, interliñado, marxes, cores, aspecto das etiquetas e de cada parte do documento): só os campos que cambian; \"\" ou 0 = o de Lúa. Non toques os campos de deseno.fixados. Non funciona se o campo estilo (LaTeX libre) ten algo."
	return e
}
