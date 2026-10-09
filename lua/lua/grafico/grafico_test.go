package grafico

import (
	"bytes"
	"fmt"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fontesTeste(t *testing.T) Fontes {
	t.Helper()
	reg, err := os.ReadFile("../fontes_informe/OpenSans-Regular.ttf")
	if err != nil {
		t.Skip("sen fontes: " + err.Error())
	}
	neg, _ := os.ReadFile("../fontes_informe/OpenSans-Semibold.ttf")
	return Fontes{Regular: reg, Negrita: neg}
}

// notas é o exemplo real: cantos suspensos, suficientes, bens, notables e sobresaíntes.
func notas() Spec {
	return Spec{
		Categorias: []string{"Suspensos", "Suficientes", "Bens", "Notables", "Sobresaíntes"},
		Series:     []Serie{{Nome: "Matemáticas", Valores: []float64{2, 1, 1, 3, 1}}},
	}
}

func TestPorcentaxesSumanCen(t *testing.T) {
	for _, vals := range [][]float64{{1, 1, 1}, {2, 1, 1, 3, 1}, {7, 7, 7, 7, 7, 7, 7}, {1, 0, 2}, {99.9, 0.1}} {
		suma := 0.0
		for _, p := range porcentaxes(vals) {
			suma += p
		}
		if math.Abs(suma-100) > 1e-9 {
			t.Errorf("%v: a suma dá %v", vals, suma)
		}
	}
	got := porcentaxes([]float64{2, 1, 1, 3, 1})
	want := []float64{25, 12.5, 12.5, 37.5, 12.5}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("porcentaxes = %v, agardábase %v", got, want)
			break
		}
	}
}

func TestFormatoGalego(t *testing.T) {
	casos := map[string]string{
		fmtNum(12.5, -1):   "12,5",
		fmtNum(25, -1):     "25",
		fmtNum(6.4375, -1): "6,44",
		fmtNum(3, 2):       "3,00",
		fmtNum(-0.0001, 1): "0,0",
		fmtPct(37.5):       "37,5" + nbsp + "%",
	}
	for got, want := range casos {
		if got != want {
			t.Errorf("formato: %q, agardábase %q", got, want)
		}
	}
}

func TestEscalaRango(t *testing.T) {
	casos := []struct{ min, max, lo, hi, paso float64 }{
		{0, 3, 0, 3, 1}, {0, 10, 0, 10, 2}, {0, 97, 0, 100, 20}, {0, 0.8, 0, 0.8, 0.2}, {-3, 8, -5, 10, 2.5}, {0, 0, 0, 0.2, 0.2},
	}
	for _, c := range casos {
		lo, hi, paso := escalaRango(c.min, c.max)
		if math.Abs(lo-c.lo) > 1e-9 || math.Abs(hi-c.hi) > 1e-9 || math.Abs(paso-c.paso) > 1e-9 {
			t.Errorf("escalaRango(%v,%v) = %v,%v,%v; agardábase %v,%v,%v", c.min, c.max, lo, hi, paso, c.lo, c.hi, c.paso)
		}
	}
}

func TestErros(t *testing.T) {
	casos := map[string]Spec{
		"tipo":      {Tipo: "fatia", Series: []Serie{{Valores: []float64{1}}}},
		"sen datos": {Tipo: Tarta},
		"negativo":  {Tipo: Tarta, Categorias: []string{"a", "b"}, Series: []Serie{{Valores: []float64{3, -1}}}},
		"todo cero": {Tipo: Barras, Categorias: []string{"a"}, Series: []Serie{{Valores: []float64{0}}}},
		"longo":     {Tipo: Barras, Categorias: []string{"a", "b"}, Series: []Serie{{Valores: []float64{1, 2}}, {Valores: []float64{1}}}},
		"nan":       {Tipo: Linhas, Series: []Serie{{Valores: []float64{math.NaN()}}}},
	}
	for nome, s := range casos {
		if _, err := Xerar(s, nil); err == nil {
			t.Errorf("%s: debería dar erro", nome)
		}
	}
}

func TestTodosOsTiposXeranPNGeTikZ(t *testing.T) {
	f := fontesTeste(t)
	m := NovoMedidor(f)
	for _, ti := range Tipos() {
		s := notas()
		s.Tipo = ti.ID
		if ti.Series {
			s.Series = append(s.Series, Serie{Nome: "Lingua", Valores: []float64{1, 2, 2, 3, 0}})
		}
		e, err := Xerar(s, m)
		if err != nil {
			t.Fatalf("%s: %v", ti.ID, err)
		}
		if e.Ancho <= 0 || e.Alto <= 0 || len(e.Elementos) == 0 {
			t.Errorf("%s: escena baleira", ti.ID)
		}
		// Nada fóra do lenzo (marxe de 0,5 mm por erro de redondeo do texto).
		for _, el := range e.Elementos {
			if r, ok := el.(Rect); ok && (r.X < -0.5 || r.Y < -0.5 || r.X+r.W > e.Ancho+0.5 || r.Y+r.H > e.Alto+0.5) {
				t.Errorf("%s: un rectángulo sae do lenzo: %+v (%.1f×%.1f)", ti.ID, r, e.Ancho, e.Alto)
			}
			if x, ok := el.(Texto); ok {
				w := m.Ancho(x.S, x.Tam, x.Negrita)
				x0, x1 := x.X, x.X+w
				switch x.Ancora {
				case AncoraCentro:
					x0, x1 = x.X-w/2, x.X+w/2
				case AncoraDta:
					x0, x1 = x.X-w, x.X
				}
				if x0 < -0.5 || x1 > e.Ancho+0.5 || x.Y > e.Alto+0.5 || x.Y < 0 {
					t.Errorf("%s: o texto %q sae do lenzo (%.1f..%.1f de %.1f, y=%.1f de %.1f)", ti.ID, x.S, x0, x1, e.Ancho, x.Y, e.Alto)
				}
			}
		}
		datos, err := e.PNG(300, f)
		if err != nil {
			t.Fatalf("%s: PNG: %v", ti.ID, err)
		}
		im, err := png.Decode(bytes.NewReader(datos))
		if err != nil {
			t.Fatalf("%s: PNG non válido: %v", ti.ID, err)
		}
		if got, want := im.Bounds().Dx(), int(math.Ceil(e.Ancho*300/25.4)); got != want {
			t.Errorf("%s: PNG de %d px, agardábase %d", ti.ID, got, want)
		}
		tk := e.TikZ(OpcionsTikZ{})
		if !strings.HasPrefix(tk, `\begin{tikzpicture}`) || !strings.HasSuffix(tk, `\end{tikzpicture}`) {
			t.Errorf("%s: TikZ mal formado", ti.ID)
		}
		if strings.Count(tk, "{") != strings.Count(tk, "}") {
			t.Errorf("%s: chaves sen pechar no TikZ", ti.ID)
		}
		if strings.Contains(tk, nbsp) {
			t.Errorf("%s: quedou un espazo duro sen traducir a «~»", ti.ID)
		}
	}
}

func TestTartaCorEstablePorCategoria(t *testing.T) {
	// Unha porción a 0 non desprazas as cores das demais (mesma categoría, mesma cor).
	a := notas()
	a.Tipo = Tarta
	b := notas()
	b.Tipo = Tarta
	b.Series[0].Valores = []float64{0, 1, 1, 3, 1}
	ea, _ := Xerar(a, nil)
	eb, _ := Xerar(b, nil)
	cores := func(e *Escena) map[string]bool {
		m := map[string]bool{}
		for _, el := range e.Elementos {
			if s, ok := el.(Sector); ok {
				m[s.Cor] = true
			}
		}
		return m
	}
	ca, cb := cores(ea), cores(eb)
	if len(ca) != 5 || len(cb) != 4 {
		t.Fatalf("porcións: %d e %d", len(ca), len(cb))
	}
	for c := range cb {
		if !ca[c] {
			t.Errorf("a cor %s non está na tarta completa", c)
		}
	}
	if cb[PaletaDefecto[0]] {
		t.Error("a categoría a 0 non debe debuxar porción")
	}
}

func TestQuebrarEcortar(t *testing.T) {
	m := medidorAprox{}
	if l := quebrar(m, "Sobresaíntes de todo o grupo", 8.5, false, 20, 2); len(l) != 2 {
		t.Errorf("quebrar: %v", l)
	}
	if c := cortar(m, "Unha etiqueta moi longa", 9, false, 15); !strings.HasSuffix(c, "…") || m.Ancho(c, 9, false) > 15 {
		t.Errorf("cortar: %q", c)
	}
}

// TestMostras escribe todos os gráficos en GRAFICO_SAIDA (PNG e TikZ) para
// miralos. Sen a variable non fai nada.
func TestMostras(t *testing.T) {
	dir := os.Getenv("GRAFICO_SAIDA")
	if dir == "" {
		t.Skip("defina GRAFICO_SAIDA para xerar as mostras")
	}
	f := fontesTeste(t)
	m := NovoMedidor(f)
	var tex strings.Builder
	tex.WriteString("\\documentclass[a4paper]{article}\n\\usepackage{fontspec,xcolor,tikz,graphicx}\n\\setmainfont{OpenSans-Regular.ttf}[Path=./fonts/,BoldFont=OpenSans-Semibold.ttf]\n\\newcommand{\\fontchip}{\\sffamily}\n\\usepackage[margin=20mm]{geometry}\n\\begin{document}\\parindent0pt\n")
	os.MkdirAll(filepath.Join(dir, "fonts"), 0o755)
	os.WriteFile(filepath.Join(dir, "fonts", "OpenSans-Regular.ttf"), f.Regular, 0o644)
	os.WriteFile(filepath.Join(dir, "fonts", "OpenSans-Semibold.ttf"), f.Negrita, 0o644)
	casos := map[string]Spec{}
	for _, ti := range Tipos() {
		s := notas()
		s.Tipo = ti.ID
		s.EtiquetaTotal = "Total"
		casos[ti.ID] = s
		if ti.Series {
			s2 := notas()
			s2.Tipo = ti.ID
			s2.Series = []Serie{{Nome: "Matemáticas", Valores: []float64{2, 1, 1, 3, 1}}, {Nome: "Lingua galega", Valores: []float64{1, 1, 2, 3, 1}}, {Nome: "Inglés", Valores: []float64{1, 2, 1, 3, 1}}}
			casos[ti.ID+"_series"] = s2
		}
	}
	// Casos difíciles: nomes longos, decimais, cero, moitas categorías.
	casos["barras_medias"] = Spec{Tipo: Barras, Categorias: []string{"Matemáticas", "Lingua galega e literatura", "Inglés", "Plástica", "Educación física"},
		Series: []Serie{{Nome: "Nota media", Valores: []float64{6.4375, 6.875, 6.625, 7.75, 8.2}}}}
	casos["linas_evolucion"] = Spec{Tipo: Linhas, Categorias: []string{"1ª aval.", "2ª aval.", "3ª aval."}, Series: []Serie{{Nome: "1º ESO A", Valores: []float64{5.8, 6.4, 7.1}}, {Nome: "1º ESO B", Valores: []float64{6.2, 5.9, 6.6}}}}
	casos["tarta_un_dato"] = Spec{Tipo: Rosca, Categorias: []string{"Aprobados"}, Series: []Serie{{Valores: []float64{18}}}, EtiquetaTotal: "Total"}
	casos["tarta_ambos"] = Spec{Tipo: Tarta, Etiquetas: "ambos", Categorias: notas().Categorias, Series: notas().Series}
	casos["barras_h_moitas"] = Spec{Tipo: BarrasH, Categorias: []string{"Antía Barreiro López", "Brais Cendón Mariño", "Carmen Doval Ferro", "Iago Estévez Rama", "Lucía Fandiño Souto", "Mauro Neira Piñeiro", "Noa Parga Vila", "Uxío Reboiras Caneda"}, Series: []Serie{{Nome: "Nota", Valores: []float64{7.5, 4.5, 9, 6, 8.5, 3.5, 7, 5.5}}}}
	for nome, s := range casos {
		e, err := Xerar(s, m)
		if err != nil {
			t.Fatalf("%s: %v", nome, err)
		}
		datos, err := e.PNG(300, f)
		if err != nil {
			t.Fatal(err)
		}
		os.WriteFile(filepath.Join(dir, nome+".png"), datos, 0o644)
		fmt.Fprintf(&tex, "\\textbf{%s} (%.0f×%.0f mm)\\par\\smallskip\n\\fbox{\\begin{minipage}{%.2fmm}\\begin{center}\n%s\n\\end{center}\\end{minipage}}\\par\\smallskip\n\\fbox{\\includegraphics[width=%.2fmm]{%s.png}}\\par\\bigskip\n",
			strings.ReplaceAll(nome, "_", "\\_"), e.Ancho, e.Alto, e.Ancho, e.TikZ(OpcionsTikZ{Escape: func(s string) string { return strings.ReplaceAll(s, "%", `\%`) }}), e.Ancho, nome)
	}
	tex.WriteString("\\end{document}\n")
	os.WriteFile(filepath.Join(dir, "mostras.tex"), []byte(tex.String()), 0o644)
}
