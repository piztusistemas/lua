package grafico

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

// OpcionsTikZ configura a saída TikZ.
type OpcionsTikZ struct {
	// Escape protexe un texto para LaTeX (nil: sen escapar).
	Escape func(string) string
	// Fonte son os comandos que escollen a familia do texto (p. ex. `\fontchip`,
	// da plantilla); baleiro = `\sffamily`.
	Fonte string
}

func mm(v float64) string {
	s := strconv.FormatFloat(v, 'f', 3, 64)
	s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	if s == "" || s == "-" || s == "-0" {
		return "0"
	}
	return s
}

// TikZ serializa a escena a un `tikzpicture` (1 unidade = 1 mm, y cara arriba).
// O texto compóno LaTeX, así que sae coa fonte do documento. Cada gráfico é
// autosuficiente: define as súas cores dentro do propio ambiente.
func (e *Escena) TikZ(o OpcionsTikZ) string {
	esc := o.Escape
	if esc == nil {
		esc = func(s string) string { return s }
	}
	fonte := o.Fonte
	if fonte == "" {
		fonte = `\sffamily`
	}
	y := func(v float64) string { return mm(e.Alto - v) }

	usadas := map[string]bool{}
	cor := func(c string) string {
		c = strings.ToUpper(strings.TrimPrefix(c, "#"))
		usadas[c] = true
		return "zg" + c
	}

	var cos strings.Builder
	for _, el := range e.Elementos {
		switch x := el.(type) {
		case Rect:
			r := ""
			if x.Radio > 0 {
				r = ",rounded corners=" + mm(x.Radio) + "mm"
			}
			fmt.Fprintf(&cos, "\\fill[%s%s] (%s,%s) rectangle (%s,%s);\n", cor(x.Cor), r, mm(x.X), y(x.Y+x.H), mm(x.X+x.W), y(x.Y))
		case Circulo:
			fmt.Fprintf(&cos, "\\fill[%s] (%s,%s) circle (%smm);\n", cor(x.Cor), mm(x.Cx), y(x.Cy), mm(x.R))
		case Linha:
			if len(x.Pts) < 2 {
				continue
			}
			est := cor(x.Cor) + ",line width=" + mm(x.Grosor) + "mm,line cap=round,line join=round"
			if x.Trazos {
				est += ",dash pattern=on 2mm off 1.5mm"
			}
			cos.WriteString("\\draw[" + est + "]")
			for i, p := range x.Pts {
				if i > 0 {
					cos.WriteString(" --")
				}
				fmt.Fprintf(&cos, " (%s,%s)", mm(p[0]), y(p[1]))
			}
			cos.WriteString(";\n")
		case Sector:
			cos.WriteString(sectorTikZ(x, e.Alto, cor))
		case Texto:
			ancora := "base west"
			switch x.Ancora {
			case AncoraCentro:
				ancora = "base"
			case AncoraDta:
				ancora = "base east"
			}
			txt := strings.ReplaceAll(esc(x.S), nbsp, "~")
			neg := ""
			if x.Negrita {
				neg = `\bfseries`
			}
			fmt.Fprintf(&cos, "\\node[anchor=%s,text=%s,font=%s%s\\fontsize{%s}{%s}\\selectfont] at (%s,%s) {%s};\n",
				ancora, cor(x.Cor), fonte, neg, mm(x.Tam), mm(x.Tam*1.2), mm(x.X), y(x.Y), txt)
		}
	}

	var b strings.Builder
	b.WriteString("\\begin{tikzpicture}[x=1mm,y=1mm,every node/.style={inner sep=0pt,outer sep=0pt}]\n")
	nomes := make([]string, 0, len(usadas))
	for c := range usadas {
		nomes = append(nomes, c)
	}
	sort.Strings(nomes)
	for _, c := range nomes {
		fmt.Fprintf(&b, "\\definecolor{zg%s}{HTML}{%s}\n", c, c)
	}
	// Reserva o tamaño completo aínda que o debuxado non chegue aos bordes.
	fmt.Fprintf(&b, "\\path[use as bounding box] (0,0) rectangle (%s,%s);\n", mm(e.Ancho), mm(e.Alto))
	b.WriteString(cos.String())
	b.WriteString("\\end{tikzpicture}")
	return b.String()
}

// sectorTikZ traza unha porción ou un anel. TikZ mide os ángulos desde o leste
// en sentido antihorario e a escena desde o norte en sentido horario: θ = 90 − a.
func sectorTikZ(s Sector, alto float64, cor func(string) string) string {
	est := "fill=" + cor(s.Cor)
	if s.Borde != "" && s.BordeGrosor > 0 {
		est += ",draw=" + cor(s.Borde) + ",line width=" + mm(s.BordeGrosor) + "mm,line join=round"
	}
	cy := alto - s.Cy
	pt := func(r, a float64) string {
		x, y := puntoPolar(s.Cx, s.Cy, r, a)
		return "(" + mm(x) + "," + mm(alto-y) + ")"
	}
	if math.Abs(s.A1-s.A0) >= 359.99 {
		if s.R0 > 0 {
			return fmt.Sprintf("\\path[%s,even odd rule] (%s,%s) circle (%smm) (%s,%s) circle (%smm);\n", est, mm(s.Cx), mm(cy), mm(s.R), mm(s.Cx), mm(cy), mm(s.R0))
		}
		return fmt.Sprintf("\\path[%s] (%s,%s) circle (%smm);\n", est, mm(s.Cx), mm(cy), mm(s.R))
	}
	t0, t1 := 90-s.A0, 90-s.A1
	if s.R0 > 0 {
		return fmt.Sprintf("\\path[%s] %s arc[start angle=%s,end angle=%s,radius=%smm] -- %s arc[start angle=%s,end angle=%s,radius=%smm] -- cycle;\n",
			est, pt(s.R, s.A0), mm(t0), mm(t1), mm(s.R), pt(s.R0, s.A1), mm(t1), mm(t0), mm(s.R0))
	}
	return fmt.Sprintf("\\path[%s] (%s,%s) -- %s arc[start angle=%s,end angle=%s,radius=%smm] -- cycle;\n",
		est, mm(s.Cx), mm(cy), pt(s.R, s.A0), mm(t0), mm(t1), mm(s.R))
}
