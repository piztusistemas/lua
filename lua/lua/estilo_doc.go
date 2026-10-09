package main

// DESEÑO dos documentos (cronograma e actas): vai en LaTeX, no bloque «Estilo»
// da plantilla, e pódese cambiar por completo dende o editor de plantillas.
// Lua só xera o contido; os macros de aquí (\chip, \tarefa, \trimestre,
// \actacab, \actasecc...) son os que usa ese contido, así que o aspecto do
// documento é cousa do LaTeX e non do código de Lua.
//
// Dúas partes:
//
//   - o que depende dos DATOS e vai sempre igual (bloqueDinamicoInforme): as
//     cores das tarefas e dos meses (a paleta), o macro \logocentro (o
//     logotipo do curso), as cores de texto e o tipo de letra das etiquetas;
//   - o DESEÑO (estiloPorDefectoLatex): marxes de parágrafo, formato dos
//     títulos, etiquetas, cabeceiras de trimestre, cabeceira e rótulos das
//     actas... Se a plantilla ten o seu Estilo substitúe este enteiro; baleiro
//     = este. O editor ofrece «Usar o estilo de Lua» para partir del.

import (
	"fmt"
	"strings"
)

// estiloPorDefectoLatex é o deseño por defecto. Pode usar as cores do bloque
// dinámico (tf-*, tt-*, tb-*, tc-*, mes-*, mesf-*, gris-linea, gris-texto) e
// \fontchip (o tipo de letra das etiquetas).
const estiloPorDefectoLatex = `% ESTILO — deseño de Lua. Cambia o que queiras: parágrafos, títulos, etiquetas, cores...
\setlength{\parindent}{0pt}
\setlength{\parskip}{0.55em}
\setlist{itemsep=1pt,topsep=2pt,leftmargin=*}
\setcounter{tocdepth}{2}
\urlstyle{same}
\titleformat{\section}{\sffamily\fontsize{20}{24}\selectfont\bfseries}{\thesection.}{0.5em}{}[{\vspace{1.2mm}\color{tc-indigo}\rule{16mm}{2pt}}]
\titleformat{\subsection}{\sffamily\fontsize{14}{17}\selectfont\bfseries}{\thesubsection.}{0.5em}{}
\titlespacing*{\section}{0pt}{0pt}{12pt}
\titlespacing*{\subsection}{0pt}{14pt}{4pt}
\newsavebox{\chipbox}
\newlength{\chippad}\setlength{\chippad}{3.2mm}
\newlength{\chipmax}
\newlength{\chiph}
\newcommand{\chipsize}{\fontsize{7.5}{9}\selectfont}
\tikzset{chipstyle/.style={draw=none,left color=tc-#1,right color=tt-#1,text=white,rounded corners=2.6mm,inner xsep=\chippad,inner ysep=1.5pt,minimum height=5.2mm,font=\fontchip\chipsize\bfseries}}
% Etiqueta (píldora con degradado) dunha tarefa no cronograma anual. Se o
% título non cabe nunha liña do ancho dispoñible, o texto pártese dentro da
% propia etiqueta. Leva un puntal invisible do seu alto máis 2 mm por riba e
% por baixo, así as tarefas nunca tocan as liñas da táboa e as liñas de
% etiquetas quedan un pouco separadas.
\newcommand{\chip}[2]{%
  \begingroup\fontchip\chipsize\bfseries
  \sbox{\chipbox}{#2}%
  \setlength{\chipmax}{\dimexpr\linewidth-2\chippad-1pt\relax}%
  \ifdim\wd\chipbox>\chipmax
    \setbox\chipbox\vbox{\hsize=\chipmax\raggedright #2\par}%
    \setlength{\chiph}{\dimexpr(\ht\chipbox+\dp\chipbox)/2+1.5pt+2mm\relax}%
    \tikz[baseline=(chipn.center)]\node[chipstyle={#1},text width=\chipmax,align=left](chipn){#2};%
  \else
    \setlength{\chiph}{4.6mm}%
    \tikz[baseline=(chipn.center)]\node[chipstyle={#1}](chipn){#2};%
  \fi
  \rule[-\chiph]{0pt}{\dimexpr2\chiph\relax}%
  \endgroup}
% Etiqueta do mes no cronograma anual: cor forte co nome e, debaixo, cantas
% tarefas ten; sen tarefas queda en gris.
% A etiqueta ten o ancho do nome de mes máis longo (\meswidth, que mide
% \medirmes para cada mes antes das táboas): a columna non ocupa máis sitio ca
% o necesario e as tarefas teñen todo o resto.
\newlength{\meswidth}
\tikzset{mesbadge/.style={rounded corners=2.4mm,minimum width=\meswidth,minimum height=9.5mm,inner xsep=3mm,inner ysep=1mm,text=white,align=center,font=\fontchip}}
\newcommand{\medirmes}[2]{\sbox{\chipbox}{\tikz\node[mesbadge,minimum width=0pt]{{\bfseries\fontsize{8}{9}\selectfont #1}\ifx\relax#2\relax\else\\[-0.3mm]{\fontsize{5.5}{6}\selectfont #2}\fi};}%
  \ifdim\wd\chipbox>\meswidth\setlength{\meswidth}{\wd\chipbox}\fi}
\newcommand{\mesbadge}[3]{\tikz[baseline=(mb.center)]\node[mesbadge,fill=mesf-#1](mb){{\bfseries\fontsize{8}{9}\selectfont #2}\ifx\relax#3\relax\else\\[-0.3mm]{\fontsize{5.5}{6}\selectfont #3}\fi};}
\newcommand{\mesvacio}[2]{\tikz[baseline=(mb.center)]\node[mesbadge,fill=black!6,text=black!55](mb){{\bfseries\fontsize{8}{9}\selectfont #2}};}
% Un trimestre do cronograma anual: nome e meses, unha liña e a táboa dos
% seus meses. Todo vai nunha caixa que non se parte: se non cabe na páxina,
% pasa enteiro á seguinte (un trimestre nunca queda cortado polo medio).
\newenvironment{trimestre}[2]{\par\vspace{5mm}\noindent\begin{minipage}{\linewidth}%
  {\fontchip\bfseries\fontsize{10}{12}\selectfont\color{tt-indigo}#1}\hspace{2.5mm}{\fontchip\fontsize{8}{10}\selectfont\color{gris-texto}#2}\par
  \vspace{0.8mm}\noindent{\color{tb-indigo}\rule{\linewidth}{0.8pt}}\par\vspace{0.5mm}}{\end{minipage}\par}
% Título dunha tarefa no desenvolvemento: cadradiño da súa cor + negriña.
\newcommand{\tarefa}[2]{\par\vspace{0.8ex}{\sffamily\large\bfseries\textcolor{tc-#1}{\rule{2.4mm}{2.4mm}}\hspace{2mm}#2\par}\nopagebreak[4]\vspace{0.2ex}}
% ── Actas do departamento ──
% Cabeceira dunha acta: rótulo pequeno, título, subtítulo e un filete.
\newcommand{\actacab}[3]{\par\noindent{\fontchip\bfseries\fontsize{8}{10}\selectfont\color{tt-indigo}\MakeUppercase{#1}}\par\vspace{1mm}\noindent{\sffamily\bfseries\fontsize{22}{26}\selectfont #2\par}\vspace{0.5mm}\noindent{\fontchip\fontsize{10.5}{13}\selectfont\color{gris-texto}#3\par}\vspace{1.5mm}\noindent{\color{tc-indigo}\rule{16mm}{2pt}}\par\vspace{2mm}}
% Un dato da acta (data, hora, lugar): rótulo en negriña e valor.
\newcommand{\actadato}[2]{\noindent{\fontchip\bfseries\fontsize{9}{12}\selectfont\color{gris-texto}\makebox[22mm][l]{#1}}#2\par}
% Título dun apartado da acta (asistentes, orde do día, acordos...).
\newcommand{\actasecc}[1]{\par\vspace{3.5mm}\noindent{\fontchip\bfseries\fontsize{10.5}{13}\selectfont\color{tt-indigo}#1}\par\nopagebreak\vspace{0.6mm}\nopagebreak\noindent{\color{tb-indigo}\rule{\linewidth}{0.8pt}}\par\nopagebreak\vspace{1mm}}
% Cabeceira do seguimento dunha materia: cadradiño, nome e, á dereita, docente e grupos.
\newcommand{\actamateria}[2]{\par\vspace{2.4mm}\noindent{\sffamily\bfseries\large\textcolor{tc-indigo}{\rule{2.4mm}{2.4mm}}\hspace{2mm}#1}\hfill{\fontchip\small\color{gris-texto}#2}\par\nopagebreak[4]\vspace{0.4mm}}
% Un campo do seguimento dunha materia (rótulo pequeno e o texto debaixo).
\newcommand{\actacampo}[2]{\par\smallskip\noindent{\fontchip\bfseries\fontsize{8.5}{11}\selectfont\color{gris-texto}#1}\par\nopagebreak\noindent#2}
% Grao de cumprimento: rótulo, barra e porcentaxe (0-100).
\newcommand{\actaprogreso}[2]{\par\noindent{\fontchip\bfseries\fontsize{8.5}{11}\selectfont\color{gris-texto}#1}\hspace{3mm}\tikz[baseline=-0.6ex]{\fill[black!8,rounded corners=1pt](0,-0.09) rectangle (4,0.09);\fill[tc-emerald,rounded corners=1pt](0,-0.09) rectangle ({0.04*#2},0.09);}\hspace{2mm}{\fontchip\bfseries\small #2\,\%}\par}
% Sinaturas do peche da acta: quen redacta e a xefatura, cada un coa súa liña. Van xuntas.
\newcommand{\actasinaturas}[4]{\par\vspace{9mm}\noindent\begin{minipage}{\linewidth}%
  \begin{minipage}[t]{0.47\linewidth}\centering{\fontchip\fontsize{8.5}{10}\selectfont\color{gris-texto}#1\par}\vspace{14mm}{\color{black!40}\rule{55mm}{0.4pt}}\par\vspace{1.5mm}{\fontchip\bfseries\fontsize{10}{12}\selectfont #2\par}\end{minipage}\hfill
  \begin{minipage}[t]{0.47\linewidth}\centering{\fontchip\fontsize{8.5}{10}\selectfont\color{gris-texto}#3\par}\vspace{14mm}{\color{black!40}\rule{55mm}{0.4pt}}\par\vspace{1.5mm}{\fontchip\bfseries\fontsize{10}{12}\selectfont #4\par}\end{minipage}
\end{minipage}\par}
`

// estiloInforme devolve o deseño LaTeX que se aplica: o da plantilla se o ten,
// senón o de Lua.
func estiloInforme(p *PlantillaLatex) string {
	if p != nil && strings.TrimSpace(p.Estilo) != "" {
		return p.Estilo
	}
	return estiloPorDefectoLatex
}

// vivosTarefasCrono é o extremo vivo do degradado das etiquetas de cada cor de
// tarefa (o outro extremo é a cor de texto da paleta). Só vale para o PDF.
var vivosTarefasCrono = map[string]string{
	"blue": "2563EB", "emerald": "059669", "amber": "D97706", "rose": "E11D48",
	"purple": "9333EA", "cyan": "0891B2", "indigo": "4F46E5", "stone": "57534E",
}

// coresMesVivasCrono é a cor forte da etiqueta de cada mes (0 = xaneiro), a
// versión saturada de coresMesCrono: estacións de fondo cálido no outono e
// frío no inverno.
var coresMesVivasCrono = [12]string{
	"1D4ED8", "0E7490", "0F766E", "15803D", "4D7C0F", "A16207",
	"B45309", "B91C1C", "C2410C", "B45309", "57534E", "334155",
}

// bloqueDinamicoInforme son as definicións que dependen dos datos e que o
// deseño da plantilla dá por feitas: as cores da paleta (tarefas e meses), o
// macro \logocentro (o logotipo do curso; nada se o curso non ten), as cores
// de texto e o tipo de letra das etiquetas (\fontchip: Open Sans se o motor é
// xelatex/lualatex e a fonte está en fonts/; se non, a sans normal).
func bloqueDinamicoInforme(logo *imaxeCrono, fontes bool) string {
	var p strings.Builder
	for _, c := range paletaTarefasCrono {
		fmt.Fprintf(&p, "\\definecolor{tf-%s}{HTML}{%s}\\definecolor{tt-%s}{HTML}{%s}\\definecolor{tb-%s}{HTML}{%s}\\definecolor{tc-%s}{HTML}{%s}\n",
			c.Clave, c.Fondo, c.Clave, c.Texto, c.Clave, c.Bordo, c.Clave, vivosTarefasCrono[c.Clave])
	}
	for i, c := range coresMesCrono {
		fmt.Fprintf(&p, "\\definecolor{mes-%d}{HTML}{%s}\\definecolor{mesf-%d}{HTML}{%s}\n", i, c, i, coresMesVivasCrono[i])
	}
	// O logotipo do curso: co macro as plantillas non teñen que saber onde
	// está o ficheiro nin que pasa se o curso non ten logotipo.
	p.WriteString("\\newif\\ifhaylogocentro\n")
	if logo != nil {
		p.WriteString("\\haylogocentrotrue\n\\newcommand{\\logocentro}[1][]{\\includegraphics[#1]{" + logo.Ficheiro + "}}\n")
	} else {
		p.WriteString("\\haylogocentrofalse\n\\newcommand{\\logocentro}[1][]{}\n")
	}
	p.WriteString(`\definecolor{gris-linea}{HTML}{E7E5E4}
\definecolor{gris-texto}{HTML}{44403C}
% Tipo de letra das etiquetas: Open Sans (a que embebe Lua en fonts/) se o
% motor é xelatex/lualatex; con pdflatex, a sans normal.
\let\fontchip\sffamily
`)
	if fontes {
		// fontspec xa está cargado (Lua e inxectarPreambuloObrigatorio cárganno
		// cos motores unicode). Se falta o ficheiro, queda a sans normal.
		p.WriteString(`\ifdefined\newfontfamily
  \IfFileExists{fonts/OpenSans-Regular.ttf}{%
    \renewfontfamily\fontchip[Path=fonts/,Extension=.ttf,UprightFont=OpenSans-Regular,BoldFont=OpenSans-Semibold,ItalicFont=OpenSans-Italic,BoldItalicFont=OpenSans-SemiboldItalic]{OpenSans-Regular}}{}
\fi
`)
	}
	return p.String()
}

// bloqueInformeCrono son as definicións que dan ao documento o seu aspecto:
// as que dependen dos datos e, a continuación, o deseño (Estilo) da plantilla
// ou o de Lua. Vai igual no preámbulo de Lua ca inxectado nunha plantilla
// completa: así o documento non cambia de cara segundo o tipo de plantilla.
func bloqueInformeCrono(logo *imaxeCrono, fontes bool, plantilla *PlantillaLatex) string {
	return bloqueDinamicoInforme(logo, fontes) + estiloInforme(plantilla)
}
