package main

// Portada, cabeceira e pé do deseño, sen LaTeX: a portada por modelos con
// nome e a cabeceira e o pé por zonas (esquerda, centro, dereita) con texto,
// variables, @logo e @imaxe. O LaTeX da plantilla (portada, maqueta) segue
// existindo para quen o queira: a portada propia manda sobre o modelo; a
// cabeceira e o pé do deseño, se se enchen, mandan sobre os da maqueta.

import (
	"edicion/plantilla"
	"regexp"
	"strings"
)

// posicionLogoPortada devolve onde vai o logotipo nos modelos de portada.
func posicionLogoPortada(pos, def string, y string, marxe string) string {
	if pos == "" {
		pos = def
	}
	switch pos {
	case "centro":
		return `anchor=north] at ([yshift=-` + y + `]current page.north)`
	case "dereita":
		return `anchor=north east] at ([shift={(-` + marxe + `,-` + y + `)}]current page.north east)`
	}
	return `anchor=north west] at ([shift={(` + marxe + `,-` + y + `)}]current page.north west)`
}

// latexPortada devolve o LaTeX da portada do modelo do deseño ("" se o
// deseño non leva portada).
func (r desenoResolto) latexPortada() string {
	switch r.d.Portada {
	case "sen":
		return ""
	case "sobria":
		return `% PORTADA sobria (deseño de Lúa).
\thispagestyle{empty}\noindent
\begin{tikzpicture}[remember picture,overlay,every node/.append style={inner sep=0pt}]
  \fill[lua-principal] (current page.north west) rectangle ([yshift=-6mm]current page.north east);
  \ifhaylogocentro
    \node[` + posicionLogoPortada(r.d.LogoPortada, "esquerda", "22mm", "25mm") + ` {\logocentro[height=20mm]};
  \fi
  \node[anchor=south west,text=lua-escura,font=\fontchip\bfseries\normalsize] at ([shift={(25mm,-90mm)}]current page.north west) {\MakeUppercase{{{DEPARTAMENTO}}}};
  \node[anchor=north west,text=black,font=\sffamily\bfseries\fontsize{40}{46}\selectfont,text width=160mm,align=left] at ([shift={(25mm,-96mm)}]current page.north west) {\hyphenpenalty=10000 {{TITULO}}};
  \fill[lua-principal] ([shift={(25mm,-142mm)}]current page.north west) rectangle ++(30mm,1.6mm);
  \node[anchor=north west,text=gris-texto,font=\sffamily\Large,text width=160mm,align=left] at ([shift={(25mm,-150mm)}]current page.north west) {Curso {{{CURSO}}}\\[3mm]{\bfseries {{{CENTRO}}}}};
  \node[anchor=south west,text=gris-texto,font=\sffamily\normalsize,text width=160mm,align=left] at ([shift={(25mm,30mm)}]current page.south west) {\relax{{{XEFE}}}\\{\small {{{CARGO}}}}\\[3mm]{\small {{{LOCALIDADE}}} \textperiodcentered{} {{{DATA_LONGA}}}}};
\end{tikzpicture}
\clearpage
`
	case "oficial":
		return `% PORTADA oficial (deseño de Lúa): todo centrado.
\thispagestyle{empty}\noindent
\begin{tikzpicture}[remember picture,overlay,every node/.append style={inner sep=0pt}]
  \ifhaylogocentro
    \node[` + posicionLogoPortada(r.d.LogoPortada, "centro", "25mm", "25mm") + ` {\logocentro[height=24mm]};
  \fi
  \node[anchor=north,text=black,font=\sffamily\bfseries\large,text width=160mm,align=center] at ([yshift=-58mm]current page.north) {{{{CENTRO}}}\\[1.5mm]{\normalfont\normalsize {{{DEPARTAMENTO}}}}};
  \draw[lua-principal,line width=0.8pt] ([shift={(-60mm,-82mm)}]current page.north) -- ++(120mm,0);
  \node[anchor=north,text=lua-escura,font=\sffamily\bfseries\fontsize{34}{40}\selectfont,text width=160mm,align=center] at ([yshift=-100mm]current page.north) {\hyphenpenalty=10000 {{TITULO}}};
  \node[anchor=north,text=gris-texto,font=\sffamily\Large] at ([yshift=-142mm]current page.north) {Curso {{{CURSO}}}};
  \draw[lua-principal,line width=0.8pt] ([shift={(-60mm,-160mm)}]current page.north) -- ++(120mm,0);
  \node[anchor=south,text=black,font=\sffamily\normalsize,text width=160mm,align=center] at ([yshift=35mm]current page.south) {\relax{{{XEFE}}}\\{\small\color{gris-texto} {{{CARGO}}}}\\[4mm]{\small {{{LOCALIDADE}}}, {{{DATA_LONGA}}}}};
\end{tikzpicture}
\clearpage
`
	}
	// A de cor (a de sempre), co logotipo onde diga o deseño.
	if r.d.LogoPortada == "" || r.d.LogoPortada == "esquerda" {
		return portadaPorDefectoLatex
	}
	orixe := `\node[anchor=north west,fill=white,rounded corners=4mm,inner sep=3.5mm]
      at ([shift={(20mm,-20mm)}]current page.north west) {\logocentro[height=18mm]};`
	novo := `\node[fill=white,rounded corners=4mm,inner sep=3.5mm,` + posicionLogoPortada(r.d.LogoPortada, "esquerda", "20mm", "20mm") + ` {\logocentro[height=18mm]};`
	return strings.Replace(portadaPorDefectoLatex, orixe, novo, 1)
}

// tokenZonaRe recoñece o que non se escapa no texto dunha zona: variables,
// @logo e @imaxe nome.
var tokenZonaRe = regexp.MustCompile(`\{\{\s*[A-ZÁÉÍÓÚÜÑ][A-Z0-9_ÁÉÍÓÚÜÑ]*\s*\}\}|@logo\b|@imaxe\s+[\w.-]+`)

// latexZona converte o texto dunha zona en LaTeX: o texto escápase, as
// variables quedan para substituír despois e os saltos de liña son \\.
func latexZona(texto string) string {
	var b strings.Builder
	for i, liña := range strings.Split(strings.TrimSpace(texto), "\n") {
		if i > 0 {
			b.WriteString(`\\`)
		}
		pos := 0
		for _, m := range tokenZonaRe.FindAllStringIndex(liña, -1) {
			b.WriteString(EscapeLatexProse(liña[pos:m[0]]))
			t := liña[m[0]:m[1]]
			switch {
			case t == "@logo":
				b.WriteString(`\logocentro[height=7mm]`)
			case strings.HasPrefix(t, "@imaxe"):
				nome := strings.TrimSpace(strings.TrimPrefix(t, "@imaxe"))
				b.WriteString(`\IfFileExists{` + nome + `}{\includegraphics[height=7mm]{` + nome + `}}{}`)
			default:
				b.WriteString(t)
			}
			pos = m[1]
		}
		b.WriteString(EscapeLatexProse(liña[pos:]))
	}
	return b.String()
}

// latexCabeceiraPe devolve o LaTeX da cabeceira e do pé do deseño (con
// {{VARIABLES}} aínda sen substituír) ou "" se o deseño non os pon. Vai ao
// principio do corpo, despois da maqueta, así manda sobre ela.
func (r desenoResolto) latexCabeceiraPe() string {
	cab, pe := r.d.Cabeceira, r.d.Pe
	if cab.Baleira() && pe.Baleira() && !r.d.CabeceiraFilete && !r.d.PeFilete {
		return ""
	}
	var b strings.Builder
	b.WriteString("% Cabeceira e pé do deseño.\n\\pagestyle{fancy}\n")
	zona := func(cmd, lado, texto string) {
		if strings.TrimSpace(texto) != "" {
			b.WriteString(cmd + "[" + lado + "]{\\fontchip\\small\\color{gris-texto}" + latexZona(texto) + "}\n")
		}
	}
	if !cab.Baleira() || r.d.CabeceiraFilete {
		b.WriteString("\\fancyhead{}\n\\renewcommand{\\headrulewidth}{" + larguraFilete(r.d.CabeceiraFilete) + "}\n")
		if strings.Contains(cab.Esquerda+cab.Centro+cab.Dereita, "@") {
			b.WriteString("\\setlength{\\headheight}{24pt}\n")
		}
		zona(`\fancyhead`, "L", cab.Esquerda)
		zona(`\fancyhead`, "C", cab.Centro)
		zona(`\fancyhead`, "R", cab.Dereita)
	}
	if !pe.Baleira() || r.d.PeFilete {
		b.WriteString("\\fancyfoot{}\n\\renewcommand{\\footrulewidth}{" + larguraFilete(r.d.PeFilete) + "}\n")
		esq, cen, der := pe.Esquerda, pe.Centro, pe.Dereita
		num := `\thepage`
		switch r.d.NumeroPaxina {
		case "non":
		case "esquerda":
			esq = strings.TrimSpace(esq + "\n" + "@@paxina")
		case "centro":
			cen = strings.TrimSpace(cen + "\n" + "@@paxina")
		default:
			der = strings.TrimSpace(der + "\n" + "@@paxina")
		}
		for _, z := range [][2]string{{"L", esq}, {"C", cen}, {"R", der}} {
			lado, texto := z[0], z[1]
			if strings.TrimSpace(texto) == "" {
				continue
			}
			l := latexZona(strings.ReplaceAll(texto, "@@paxina", "LUAPAXINA"))
			l = strings.ReplaceAll(l, "LUAPAXINA", num)
			b.WriteString("\\fancyfoot[" + lado + "]{\\fontchip\\small\\color{gris-texto}" + l + "}\n")
		}
	}
	return b.String()
}

// larguraFilete é o groso da liña da cabeceira ou do pé (0pt = sen liña).
func larguraFilete(si bool) string {
	if si {
		return "0.4pt"
	}
	return "0pt"
}

// zonasAOfimatica traduce unha cabeceira ou un pé do deseño á linguaxe da
// cabeceira de Word/OpenDocument (unha caixa cun aliñamento).
func zonasAOfimatica(z plantilla.Zonas) string {
	var liñas []string
	alin := ""
	for _, p := range []struct{ alin, texto string }{{"@esquerda", z.Esquerda}, {"@centro", z.Centro}, {"@dereita", z.Dereita}} {
		t := strings.TrimSpace(p.texto)
		if t == "" {
			continue
		}
		if alin == "" {
			alin = p.alin
		}
		for _, l := range strings.Split(t, "\n") {
			l = strings.TrimSpace(strings.ReplaceAll(l, "@logo", ""))
			if strings.HasPrefix(l, "@imaxe") {
				l += " 8"
			}
			if l != "" {
				liñas = append(liñas, l)
			}
		}
	}
	if len(liñas) == 0 {
		return ""
	}
	return strings.Join(append(liñas, alin), "\n")
}
