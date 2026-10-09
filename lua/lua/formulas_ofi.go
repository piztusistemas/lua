package main

// Fórmulas en Word e OpenDocument. Os escritores nativos de Lua non teñen
// un editor de ecuacións: cada fórmula ($…$ ou $$…$$) compílase con LaTeX
// (o mesmo que fai o PDF) e vai coma imaxe en liña, á altura do texto. Así
// unha fórmula sae compoñida igual ca no PDF, nunca coma texto plano.
//
// Todas as fórmulas dun documento compílanse nun único LaTeX co paquete
// preview (unha páxina axustada por fórmula) e pásanse a PNG con pdftoppm.

import (
	"fmt"
	"strconv"
	"strings"
)

// ppFormula: resolución das imaxes das fórmulas.
const ppFormula = 300

// formulaOfi é unha fórmula xa compoñida.
type formulaOfi struct {
	im *imaxeOfi
	// Alto e profundidade en puntos (a 10 pt, o tamaño do LaTeX): a
	// profundidade baixa a imaxe para que a liña base case co texto.
	altoPt, fondoPt float64
}

// formulasOfi son as fórmulas dun documento, pola súa clave.
type formulasOfi struct {
	porClave map[string]formulaOfi
	debuxos  map[string]*imaxeOfi // ```tikz/```circuitikz, por claveDebuxo
	escala   float64              // tamaño do corpo do documento / 10 pt
}

func claveFormula(tex string, display bool) string {
	if display {
		return "D|" + tex
	}
	return "I|" + tex
}

// formulasDeBloques recolle as fórmulas dos textos Markdown dos bloques.
func formulasDeBloques(bloques []bloqueDoc) []cronoTramo {
	var out []cronoTramo
	vistas := map[string]bool{}
	for _, b := range bloques {
		if b.Tipo != bloqueMarkdown && b.Tipo != bloqueSolucion {
			continue
		}
		for _, bl := range parseMarkdownCrono(b.Texto) {
			for _, t := range bl.Tramos {
				if k := claveFormula(t.Math, t.Display); t.Math != "" && !vistas[k] {
					vistas[k] = true
					out = append(out, t)
				}
			}
		}
	}
	return out
}

// compoñerFormulas compila as fórmulas e devolve as imaxes. Se non hai
// LaTeX ou pdftoppm, devolve o erro (o chamador avisa e escribe o TeX coma
// texto); unha fórmula con erro tamén vai coma texto.
func compoñerFormulas(tramos []cronoTramo, tamanoPt float64) (*formulasOfi, error) {
	out := &formulasOfi{porClave: map[string]formulaOfi{}, escala: 1}
	if tamanoPt > 0 {
		out.escala = tamanoPt / 10
	}
	if len(tramos) == 0 {
		return out, nil
	}
	pezas := make([]string, len(tramos))
	for i, t := range tramos {
		if t.Display {
			pezas[i] = "$\\displaystyle " + t.Math + "$"
		} else {
			pezas[i] = "$" + t.Math + "$"
		}
	}
	recortes, err := compilarRecortes(pezas, ppFormula)
	if err != nil {
		return out, fmt.Errorf("para pór as fórmulas en Word e OpenDocument %v", err)
	}
	var mal int
	for i, r := range recortes {
		if r.im == nil {
			mal++
			continue
		}
		out.porClave[claveFormula(tramos[i].Math, tramos[i].Display)] = formulaOfi{im: r.im, altoPt: r.altoPt, fondoPt: r.fondoPt}
	}
	if mal > 0 {
		return out, fmt.Errorf("%d fórmula(s) teñen un erro de LaTeX", mal)
	}
	return out, nil
}

// engadirDebuxos compila os debuxos dos bloques (a 300 ppp). Devolve o
// aviso se algún non se puido debuxar.
func (f *formulasOfi) engadirDebuxos(bloques []bloqueDoc) error {
	var debuxos [][2]string
	vistos := map[string]bool{}
	for _, b := range bloques {
		if b.Tipo != bloqueMarkdown && b.Tipo != bloqueSolucion {
			continue
		}
		for _, d := range debuxosDe(b.Texto) {
			if k := claveDebuxo(d[0], d[1]); !vistos[k] {
				vistos[k] = true
				debuxos = append(debuxos, d)
			}
		}
	}
	if len(debuxos) == 0 {
		return nil
	}
	pezas := make([]string, len(debuxos))
	for i, d := range debuxos {
		pezas[i] = envolverDebuxo(d[0], d[1])
	}
	recortes, err := compilarRecortes(pezas, ppFormula)
	if err != nil {
		return fmt.Errorf("para pór os debuxos en Word e OpenDocument %v", err)
	}
	if f.debuxos == nil {
		f.debuxos = map[string]*imaxeOfi{}
	}
	mal := 0
	for i, r := range recortes {
		if r.im == nil {
			mal++
			continue
		}
		f.debuxos[claveDebuxo(debuxos[i][0], debuxos[i][1])] = r.im
	}
	if mal > 0 {
		return fmt.Errorf("%d debuxo(s) teñen un erro de TikZ e non saen", mal)
	}
	return nil
}

// debuxo devolve a imaxe dun debuxo e o seu tamaño en polgadas (sen
// escalar: un circuíto mide o que mide).
func (f *formulasOfi) debuxo(bl cronoBloque) (*imaxeOfi, float64, float64, bool) {
	if f == nil {
		return nil, 0, 0, false
	}
	im, ok := f.debuxos[claveDebuxo(bl.Debuxo, bl.Codigo)]
	if !ok {
		return nil, 0, 0, false
	}
	return im, float64(im.W) / ppFormula, float64(im.H) / ppFormula, true
}

// tamano: ancho e alto en polgadas no documento (escalado ao tamaño do corpo).
func (f *formulasOfi) tamano(fo formulaOfi) (w, h float64) {
	return float64(fo.im.W) / ppFormula * f.escala, float64(fo.im.H) / ppFormula * f.escala
}

func (f *formulasOfi) busca(t cronoTramo) (formulaOfi, bool) {
	if f == nil {
		return formulaOfi{}, false
	}
	fo, ok := f.porClave[claveFormula(t.Math, t.Display)]
	return fo, ok
}

// ---------- Word ----------

// runFormulaDocx: a fórmula coma imaxe en liña, baixada a súa profundidade
// (<w:position> en medios puntos) para que a liña base case co texto.
func (c *construtorDocx) runFormulaDocx(t cronoTramo) string {
	fo, ok := c.formulas.busca(t)
	if !ok {
		return runDocx(textoFormula(t), false, true, "", 0)
	}
	w, h := c.formulas.tamano(fo)
	cx, cy := int(w*914400), int(h*914400)
	rel, ok := c.engadirImaxe(fo.im, cx, cy)
	if !ok {
		return runDocx(textoFormula(t), false, true, "", 0)
	}
	id := strings.TrimPrefix(rel, "rIdImg")
	r := runImaxeDocx(rel, id, cx, cy)
	if fo.fondoPt > 0 && !t.Display {
		baixar := int(fo.fondoPt*c.formulas.escala*2 + 0.5)
		r = strings.Replace(r, "<w:r>", `<w:r><w:rPr><w:position w:val="-`+strconv.Itoa(baixar)+`"/></w:rPr>`, 1)
	}
	return r
}

// runsTramos son os runs dun bloque, coas fórmulas compoñidas.
func (c *construtorDocx) runsTramos(tramos []cronoTramo, forzarCursiva bool) []string {
	var out []string
	for _, t := range tramos {
		if t.Math != "" {
			out = append(out, c.runFormulaDocx(t))
			continue
		}
		out = append(out, runsTramosDocx([]cronoTramo{t}, forzarCursiva, c.baseMedios)...)
	}
	return out
}

// ---------- OpenDocument ----------

func (c *construtorOdt) tramos(ts []cronoTramo) string {
	var b strings.Builder
	for _, t := range ts {
		if t.Math == "" {
			b.WriteString(tramosOdt([]cronoTramo{t}))
			continue
		}
		fo, ok := c.formulas.busca(t)
		if !ok {
			b.WriteString(`<text:span text:style-name="Emphasis">` + escaparXMLOfi(textoFormula(t)) + `</text:span>`)
			continue
		}
		w, h := c.formulas.tamano(fo)
		o, ok := c.engadirImaxe(fo.im, w*2.54, h*2.54)
		if !ok {
			continue
		}
		marco := marcoOdt(o)
		if fo.fondoPt > 0 && !t.Display {
			// Ancorada coma carácter: svg:y negativo sobe, positivo baixa
			// respecto da liña base (estilo FrFormula).
			marco = strings.Replace(marco, `draw:style-name="Fr"`, fmt.Sprintf(`draw:style-name="FrFormula" svg:y="-%.3fcm"`, (fo.altoPt*c.formulas.escala)/72*2.54), 1)
		}
		b.WriteString(marco)
	}
	return b.String()
}

// textoFormula: o TeX, se a fórmula non se puido compoñer.
func textoFormula(t cronoTramo) string {
	if t.Display {
		return "$$" + t.Math + "$$"
	}
	return "$" + t.Math + "$"
}
