package main

// Escritor LaTeX (PDF) dos bloques dun formulario, e a súa compilación.
//
// O deseño non vai aquí: os bloques escríbense con macros \form* que teñen un
// aspecto por defecto (estiloFormularioPorDefectoLatex) e que a plantilla pode
// redefinir no seu bloque «Estilo». Van con \providecommand e DESPOIS do Estilo
// da plantilla: así unha plantilla con Estilo propio (que non coñece estes
// macros) segue compilando, e a que os define ganan os seus.

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// estiloFormularioPorDefectoLatex son os macros dos bloques de formulario. Usan
// as cores e \fontchip do bloque dinámico (ver estilo_doc.go), que sempre están.
const estiloFormularioPorDefectoLatex = `% FORMULARIOS — macros dos bloques dun formulario. Redefínea no Estilo da plantilla con \newcommand.
% Título do documento (se non leva portada): título, subtítulo e un filete.
\providecommand{\formtitulo}[2]{\par\noindent{\sffamily\bfseries\fontsize{22}{26}\selectfont #1\par}\ifx\relax#2\relax\else\vspace{0.5mm}\noindent{\fontchip\fontsize{10.5}{13}\selectfont\color{gris-texto}#2\par}\fi\vspace{1.5mm}\noindent{\color{tc-indigo}\rule{16mm}{2pt}}\par\vspace{2mm}}
% Sección (campo «sección»): título con filete.
\providecommand{\formseccion}[1]{\par\vspace{3.5mm}\noindent{\fontchip\bfseries\fontsize{10.5}{13}\selectfont\color{tt-indigo}#1}\par\nopagebreak\vspace{0.6mm}\nopagebreak\noindent{\color{tb-indigo}\rule{\linewidth}{0.8pt}}\par\nopagebreak\vspace{1mm}}
% Título pequeno dun texto longo, dunha táboa ou dunha imaxe.
\providecommand{\formsubseccion}[1]{\par\smallskip\noindent{\fontchip\bfseries\fontsize{8.5}{11}\selectfont\color{gris-texto}#1}\par\nopagebreak}
% Un dato da ficha: rótulo e valor na MESMA liña (comparten liña base; unha
% minipage por columna non aliña ben) con sangría francesa para as liñas
% seguintes. Se o rótulo non cabe na columna, vai enriba do valor.
\newsavebox{\formcaixa}
\providecommand{\formdato}[2]{\par\noindent
  \sbox{\formcaixa}{{\fontchip\bfseries\fontsize{9}{12}\selectfont\color{gris-texto}#1}}%
  \ifdim\wd\formcaixa>34mm
    \usebox{\formcaixa}\par\nopagebreak\noindent #2\par
  \else
    \hangindent=38mm\hangafter=1\relax
    \makebox[38mm][l]{\usebox{\formcaixa}}#2\par
  \fi}
% Cela de cabeceira dunha táboa (o fondo da fila vai co propio bloque).
\providecommand{\formcabeceira}[1]{{\fontchip\bfseries\fontsize{8.5}{11}\selectfont\color{tt-indigo}#1}}
% Pé dunha imaxe.
\providecommand{\formpe}[1]{{\fontchip\fontsize{8.5}{11}\selectfont\color{gris-texto}#1}}
% Solución dunha pregunta (campo «problemas»): rótulo e texto, cun filete á esquerda.
\providecommand{\formsolucion}[2]{\par\smallskip\noindent{\fontchip\bfseries\fontsize{9}{12}\selectfont\color{tt-indigo}#1}\par\nopagebreak\begin{list}{}{\setlength{\leftmargin}{5mm}\setlength{\topsep}{0pt}\setlength{\itemsep}{0pt}}\item[]#2\end{list}\par\medskip}
`

// bloqueInformeFormulario é bloqueInformeCrono máis os macros dos formularios
// (despois do Estilo, ver a nota do principio).
func bloqueInformeFormulario(logo *imaxeCrono, fontes bool, plantilla *PlantillaLatex) string {
	return bloqueInformeCrono(logo, fontes, plantilla) + estiloFormularioPorDefectoLatex
}

// larguraImaxeMaxMMFormulario é o ancho máximo dunha imaxe dun formulario.
const larguraImaxeMaxMMFormulario = 120.0

// latexBloquesFormulario escribe os bloques en LaTeX.
func latexBloquesFormulario(bloques []bloqueDoc, esc *escritorImaxesCrono) string {
	var b strings.Builder
	for _, bl := range bloques {
		switch bl.Tipo {
		case bloqueTitulo:
			macro := `\formsubseccion`
			if bl.Nivel <= 2 {
				macro = `\formseccion`
			}
			b.WriteString(macro + `{` + EscapeLatexProse(bl.Texto) + "}\n")
		case bloqueFicha:
			for _, par := range bl.Pares {
				b.WriteString(`\formdato{` + EscapeLatexProse(par[0]) + `}{` + latexTextoSimpleFormulario(par[1]) + "}\n")
			}
		case bloqueMarkdown:
			if blocosMd := parseMarkdownCrono(bl.Texto); len(blocosMd) > 0 {
				b.WriteString(latexBloquesCrono(blocosMd, nil, esc))
			}
		case bloqueTaboa:
			b.WriteString(latexTaboaFormulario(bl))
		case bloqueLista:
			b.WriteString("\\begin{itemize}\n")
			for _, el := range bl.Elementos {
				b.WriteString(`\item ` + latexTextoSimpleFormulario(el) + "\n")
			}
			b.WriteString("\\end{itemize}\n")
		case bloqueSalto:
			b.WriteString("\\clearpage\n")
		case bloqueSolucion:
			b.WriteString(`\formsolucion{` + EscapeLatexProse(bl.Rotulo) + `}{` + latexBloquesCrono(parseMarkdownCrono(bl.Texto), nil, esc) + "}\n")
		case bloqueGrafico:
			b.WriteString(latexGrafico(bl))
		case bloqueImaxe:
			im := esc.escribir(bl.Imaxe, bl.Texto)
			if im == nil || im.W <= 0 || im.H <= 0 {
				continue
			}
			// 150 ppp como referencia; nunca máis ancha ca o máximo nin ca a columna.
			w := float64(im.W) * 25.4 / 150
			if w > larguraImaxeMaxMMFormulario {
				w = larguraImaxeMaxMMFormulario
			}
			if w > larguraColumnaMMCrono {
				w = larguraColumnaMMCrono
			}
			if bl.Texto != "" {
				b.WriteString(`\formsubseccion{` + EscapeLatexProse(bl.Texto) + "}\n")
			}
			b.WriteString(`\par\medskip\noindent\begin{center}\includegraphics[width=` + mmLatexCrono(w) + `]{` + im.Ficheiro + `}\end{center}` + "\n")
		}
	}
	return b.String()
}

// latexTextoSimpleFormulario escapa un valor dunha ficha. Un salto de liña do
// dato pasa a salto de liña de LaTeX (dentro da minipage do macro).
func latexTextoSimpleFormulario(s string) string {
	linhas := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	for i, l := range linhas {
		linhas[i] = EscapeLatexProse(strings.TrimSpace(l))
	}
	return strings.Join(linhas, `\newline `)
}

// latexTaboaFormulario escribe unha táboa (longtable: parte a páxina se cómpre
// e repite a cabeceira) con columnas do mesmo ancho.
func latexTaboaFormulario(bl bloqueDoc) string {
	n := len(bl.Cabeceira)
	if n == 0 {
		return ""
	}
	nc := strconv.Itoa(n)
	// Ancho de cada columna: o da páxina menos os espazos e filetes, entre n.
	ancho := `p{\dimexpr(\linewidth-` + strconv.Itoa(2*n) + `\tabcolsep-` + strconv.Itoa(n+1) + `\arrayrulewidth)/` + nc + `\relax}`
	var b strings.Builder
	b.WriteString(`\begingroup\arrayrulecolor{tb-indigo}\renewcommand{\arraystretch}{1.25}\par\smallskip` + "\n")
	b.WriteString(`\begin{longtable}{|` + strings.Repeat(ancho+"|", n) + "}\n\\hline\n")
	celas := make([]string, n)
	for i, c := range bl.Cabeceira {
		celas[i] = `\formcabeceira{` + EscapeLatexProse(c) + `}`
	}
	cab := `\rowcolor{tf-indigo}` + strings.Join(celas, " & ") + " \\\\\n\\hline\n"
	b.WriteString(cab + "\\endfirsthead\n" + cab + "\\endhead\n")
	for _, fila := range bl.Filas {
		celas := make([]string, n)
		for i := range celas {
			if i < len(fila) {
				celas[i] = latexTextoSimpleFormulario(fila[i])
			}
		}
		b.WriteString(strings.Join(celas, " & ") + " \\\\\n\\hline\n")
	}
	b.WriteString("\\end{longtable}\\endgroup\n")
	return b.String()
}

// latexFormulario compón o documento LaTeX dun formulario enchido e devolve
// tamén os avisos (datos que non cadran, imaxes descartadas, variables da
// plantilla sen valor...). As imaxes escríbense en `dir`.
func latexFormulario(p PedidoFormulario, engine string, tip *TipografiaOpts, dir string, agora time.Time, plantilla *PlantillaLatex, nomeDoc string) (string, []string, error) {
	f, d := normalizarFormulario(p.Formulario), p.Datos
	if err := validarFormulario(f); err != nil {
		return "", nil, fmt.Errorf("formulario non válido: %w", err)
	}
	esc := novoEscritorImaxesCrono(dir)
	logo := esc.escribir(d.Contexto.Logo, "logotipo")
	auto := variablesFormulario(f, d, nomeDoc, agora, plantilla)
	tx := novosTextosDoc(plantilla, auto)
	tx.engadirVariables(auto, "TITULO_FORMULARIO")
	if engineUnicodeNativo(engine) {
		if err := escribirFontesInformeCrono(dir); err != nil {
			esc.avisos = append(esc.avisos, "non se puido escribir a fonte do documento: "+err.Error())
		}
	}
	bloques, avisosDatos := bloquesDocumentoFormulario(p, f, d, tx)

	var b strings.Builder
	b.WriteString(babelListquotFix)
	b.WriteString(tipografiaCorpo(tip))
	// Sen título automático: o corpo é só o que ten o formulario. O título vai
	// na plantilla (portada, cabeceira: {{TITULO}}) ou nun campo do formulario.
	b.WriteString(latexBloquesFormulario(bloques, esc))

	bloque := bloqueInformeFormulario(logo, engineUnicodeNativo(engine), plantilla)
	doc, avisosPlantilla, err := documentoLatexConPlantilla(
		preambuloInformeCronoCon(engine, tip, logo, plantilla, estiloFormularioPorDefectoLatex),
		partesInformeCrono{PortadaDef: portadaPorDefectoLatex, Corpo: b.String(), SinaturaDef: sinaturaPorDefectoLatex, SenPortada: !f.Portada, SenSinatura: !f.Sinatura},
		engine, plantilla, auto, paquetesInformeCrono, bloque)
	if err != nil {
		return "", nil, err
	}
	return doc, engadirSenRepetir(engadirSenRepetir(avisosDatos, esc.avisos), avisosPlantilla), nil
}

// compilarFormulario compila o formulario co motor preferido (ver compilarComMotores).
func compilarFormulario(p PedidoFormulario, preferido string, tip *TipografiaOpts, dir string, plantilla *PlantillaLatex, nomeDoc string) (resultadoInformeCrono, error) {
	return compilarComMotores(preferido, tip, dir, plantilla, func(motor string) (string, []string, error) {
		return latexFormulario(p, motor, tip, dir, time.Now(), plantilla, nomeDoc)
	})
}
