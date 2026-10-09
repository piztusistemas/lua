package main

// Deseño por defecto da PORTADA e da SINATURA (peche) do informe, e o
// texto por defecto da introdución. Son LaTeX/texto con {{VARIABLES}} igual
// que calquera plantilla: unha plantilla pode substituílos co seu propio
// deseño (campos Portada, Sinatura e Introducion de PlantillaLatex; ver
// plantillas.go) e as plantillas de fábrica levan estes mesmos textos para
// que se poidan abrir, ver como están feitos e modificar.
//
// Variables dispoñibles (as enche Lua co curso e o centro, ver
// variablesAutomaticasCrono): {{CENTRO}}, {{LOCALIDADE}}, {{CURSO}},
// {{DEPARTAMENTO}}, {{XEFE}} (= {{ORIENTADOR}}), {{CARGO}}, {{TITULO}} (o título
// do documento: «Cronograma anual» ou «Actas do departamento»), {{DATA}}, {{DATA_LONGA}}, {{DATA_INICIO}}, {{DATA_FIN}}, {{ANO}} e
// {{FICHEIRO}}. O logotipo do curso vai co macro \logocentro[opcións] (nada
// se o curso non ten logotipo; \ifhaylogocentro di se o hai) e as cores da
// paleta do informe son tt-*/tc-*/tf-*/tb-* (ver bloqueInformeCrono).

// FragmentosInforme son os deseños por defecto que usa o editor de plantillas
// para "Inserir o deseño de Lua" (e o catálogo de textos fixos, cos seus
// valores por defecto).
type FragmentosInforme struct {
	Portada     string `json:"portada"`
	Sinatura    string `json:"sinatura"`
	Introducion string `json:"introducion"`
	// PortadaOfimatica é a portada por defecto de Word/OpenDocument (liñas, ver
	// plantillas_ofimatica.go).
	PortadaOfimatica string `json:"portadaOfimatica"`
	// Estilo é o deseño LaTeX por defecto (estilo_doc.go) e Textos o catálogo de
	// textos fixos do documento (textos_doc.go).
	Estilo string     `json:"estilo"`
	Textos []TextoDoc `json:"textos"`
}

// FragmentosInformePorDefecto devolve a portada, a sinatura e a introdución
// por defecto, para que o editor de plantillas os ofreza como punto de
// partida.
func (a *App) FragmentosInformePorDefecto() FragmentosInforme {
	return FragmentosInforme{
		Portada:          portadaPorDefectoLatex,
		Sinatura:         sinaturaPorDefectoLatex,
		Introducion:      introPorDefectoCrono,
		PortadaOfimatica: portadaOfimaticaPorDefecto,
		Estilo:           estiloPorDefectoLatex,
		Textos:           textosDoc,
	}
}

// introPorDefectoCrono é a introdución cando nin o curso nin a plantilla
// levan a súa. Dá contexto ao documento e nomea o centro, o curso e o
// departamento. Os parágrafos sepáranse cunha liña en branco.
const introPorDefectoCrono = `Este documento recolle o cronograma anual do {{DEPARTAMENTO}} do {{CENTRO}} para o curso académico {{CURSO}}. Nel establécese, mes a mes, a distribución temporal das actividades, actuacións e tarefas que o departamento desenvolverá ao longo do curso, coa finalidade de planificar e coordinar de xeito eficiente o traballo previsto.

A programación ten en conta as características e as necesidades do {{CENTRO}} e do seu alumnado. Ten carácter flexible: poderá revisarse e adaptarse ao longo do curso segundo as demandas e a evolución do propio centro.`

// portadaPorDefectoLatex: unha portada moderna, a toda páxina. Bloque de cor
// con formas xeométricas arriba, título grande, o curso nunha etiqueta, o
// nome do centro e, embaixo, a franxa de cores das tarefas e o lugar e a
// data. Todo se debuxa respecto ás esquinas da páxina, así vale para A4 e
// para Letter. Pide dúas pasadas de LaTeX (as coordenadas "current page"),
// que xa fai Lua.
const portadaPorDefectoLatex = `% PORTADA — deseño de Lua. Cambia o que queiras: textos, cores (tt-indigo, tc-cyan...) e formas.
\thispagestyle{empty}\noindent
\begin{tikzpicture}[remember picture,overlay,every node/.append style={inner sep=0pt}]
  % Bloque de cor superior con formas
  \begin{scope}
    \clip (current page.north west) rectangle ([yshift=-132mm]current page.north east);
    \fill[tt-indigo] (current page.north west) rectangle ([yshift=-132mm]current page.north east);
    \fill[tc-indigo] ([shift={(-28mm,-10mm)}]current page.north east) circle (64mm);
    \fill[tt-indigo] ([shift={(-28mm,-10mm)}]current page.north east) circle (38mm);
    \fill[tc-cyan] ([shift={(-70mm,-126mm)}]current page.north east) circle (30mm);
    \fill[tc-rose] ([shift={(-16mm,-104mm)}]current page.north east) circle (11mm);
    \fill[tc-amber] ([shift={(-46mm,-64mm)}]current page.north east) circle (6mm);
  \end{scope}
  % Logotipo do curso (nunha tarxeta branca para que se vexa sobre a cor)
  \ifhaylogocentro
    \node[anchor=north west,fill=white,rounded corners=4mm,inner sep=3.5mm]
      at ([shift={(20mm,-20mm)}]current page.north west) {\logocentro[height=18mm]};
  \fi
  % Título
  \node[anchor=south west,text=white,font=\sffamily\bfseries\normalsize,align=left]
    at ([shift={(20mm,-66mm)}]current page.north west) {\MakeUppercase{{{DEPARTAMENTO}}}};
  \node[anchor=south west,text=white,font=\sffamily\bfseries,text width=165mm,align=left]
    at ([shift={(20mm,-108mm)}]current page.north west) {\hyphenpenalty=10000 \fontsize{50}{54}\selectfont {{TITULO}}};
  \node[anchor=south west,fill=white,text=tt-indigo,rounded corners=3.5mm,inner xsep=5mm,inner ysep=2.6mm,font=\sffamily\bfseries\Large]
    at ([shift={(20mm,-124mm)}]current page.north west) {Curso {{{CURSO}}}};
  % Centro
  \node[anchor=north west,text=black,font=\sffamily\bfseries,text width=170mm,align=left]
    at ([shift={(20mm,-152mm)}]current page.north west) {\fontsize{24}{28}\selectfont {{{CENTRO}}}};
  \fill[tc-indigo] ([shift={(20mm,-172mm)}]current page.north west) rectangle ++(18mm,1.6mm);
  \node[anchor=north west,text=gris-texto,font=\sffamily\large,text width=170mm,align=left]
    at ([shift={(20mm,-178mm)}]current page.north west) {\relax{{{XEFE}}}\\[1mm]\small{{{CARGO}}}};
  % Franxa de cores (as das tarefas do cronograma)
  \foreach \c [count=\i from 0] in {blue,emerald,amber,rose,purple,cyan,indigo,stone}{%
    \fill[rounded corners=1.6mm,tc-\c] ([shift={({(20+\i*22)*1mm},32mm)}]current page.south west) rectangle ++(19mm,3.2mm);
  }
  \node[anchor=south west,text=gris-texto,font=\sffamily\small]
    at ([shift={(20mm,15mm)}]current page.south west) {\relax{{{LOCALIDADE}}} \textperiodcentered{} {{{DATA_LONGA}}}};
\end{tikzpicture}
\clearpage
`

// sinaturaPorDefectoLatex: o peche do documento. Un bloque pequeno e sobrio,
// sen sección nin páxina propias: lugar e data, un espazo para asinar, unha
// liña fina e, debaixo, o nome e o cargo de quen asina. Vai xunto (nunca
// partido entre dúas páxinas) despois do último contido. Se a queres nunha
// folla aparte, empeza o fragmento con \clearpage.
const sinaturaPorDefectoLatex = `% SINATURA — deseño de Lua: bloque compacto ao final do documento.
\par\vspace{12mm}\noindent
\begin{minipage}{\linewidth}
  {\fontchip\fontsize{8.5}{10}\selectfont\color{gris-texto}{{{LOCALIDADE}}}, {{{DATA_LONGA}}}\par}
  \vspace{13mm}
  {\color{black!40}\rule{58mm}{0.4pt}}\par\vspace{1.8mm}
  {\fontchip\bfseries\fontsize{10.5}{12.5}\selectfont {{{XEFE}}}\par}
  {\fontchip\fontsize{8.5}{10.5}\selectfont\color{gris-texto}{{{CARGO}}}\par}
\end{minipage}
`
