package grafico

import (
	"encoding/binary"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
)

const (
	nbsp = " "
	// Un texto centrado a man: a liña base queda esta fracción do corpo por debaixo do centro.
	centroBase = 0.126 // mm por punto de corpo
	mmPorPt    = 25.4 / 72
)

// Fontes son os TTF co que se mide e se debuxa o texto (Open Sans en Lua).
type Fontes struct{ Regular, Negrita []byte }

// Medidor mide o ancho dun texto en mm.
type Medidor interface {
	Ancho(s string, tamPt float64, negrita bool) float64
}

type medidorAprox struct{}

func (medidorAprox) Ancho(s string, tam float64, negrita bool) float64 {
	k := 0.54
	if negrita {
		k = 0.58
	}
	return float64(len([]rune(s))) * k * tam * mmPorPt
}

type medidorFonte struct {
	mu       sync.Mutex
	reg, neg font.Face
}

// NovoMedidor mide co propio TTF. Sen fonte válida cae a unha aproximación.
func NovoMedidor(f Fontes) Medidor {
	reg := faceMedida(f.Regular)
	if reg == nil {
		return medidorAprox{}
	}
	neg := faceMedida(f.Negrita)
	if neg == nil {
		neg = reg
	}
	return &medidorFonte{reg: reg, neg: neg}
}

// lerTTF le un TTF. freetype rexeita a táboa «kern» de Open Sans («bad kern
// table length»): nese caso renómease a táboa nunha copia para que se ignore
// (só se perde o kerning na medida; o PDF, que compón LaTeX, non a usa).
func lerTTF(b []byte) (*truetype.Font, error) {
	f, err := truetype.Parse(b)
	if err == nil || len(b) < 12 {
		return f, err
	}
	c := append([]byte(nil), b...)
	n := int(binary.BigEndian.Uint16(c[4:6]))
	if len(c) < 12+16*n {
		return nil, err
	}
	for i := 0; i < n; i++ {
		if rec := c[12+16*i : 12+16*i+4]; string(rec) == "kern" {
			copy(rec, "Xern")
		}
	}
	return truetype.Parse(c)
}

func faceMedida(ttf []byte) font.Face {
	if len(ttf) == 0 {
		return nil
	}
	f, err := lerTTF(ttf)
	if err != nil {
		return nil
	}
	return truetype.NewFace(f, &truetype.Options{Size: 100, DPI: 72, Hinting: font.HintingNone})
}

func (m *medidorFonte) Ancho(s string, tam float64, negrita bool) float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	face := m.reg
	if negrita {
		face = m.neg
	}
	pt := float64(font.MeasureString(face, s)) / 64 / 100 * tam
	return pt * mmPorPt
}

// fmtNum escribe un número á galega (coma decimal). dec < 0: os decimais
// mínimos (ata 2) que non cambian o valor.
func fmtNum(v float64, dec int) string {
	if dec < 0 {
		for d := 0; d <= 2; d++ {
			dec = d
			p := math.Pow(10, float64(d))
			if math.Abs(v-math.Round(v*p)/p) < 1e-9 {
				break
			}
		}
	}
	s := strconv.FormatFloat(v, 'f', dec, 64)
	if strings.Trim(s, "-0,.") == "" {
		s = strings.TrimPrefix(s, "-") // «-0» non se escribe
	}
	return strings.Replace(s, ".", ",", 1)
}

func fmtPct(p float64) string { return fmtNum(p, -1) + nbsp + "%" }

// porcentaxes reparte 100 entre os valores cun decimal e o método do maior
// resto: a suma dá sempre 100 exacto (sen «33,3 + 33,3 + 33,3»).
func porcentaxes(vals []float64) []float64 {
	total := 0.0
	for _, v := range vals {
		total += v
	}
	out := make([]float64, len(vals))
	if total <= 0 {
		return out
	}
	type resto struct {
		i    int
		frac float64
	}
	decimas := make([]int, len(vals))
	sobran := 1000
	var restos []resto
	for i, v := range vals {
		x := v / total * 1000
		f := math.Floor(x + 1e-9)
		decimas[i] = int(f)
		sobran -= int(f)
		restos = append(restos, resto{i, x - f})
	}
	sort.SliceStable(restos, func(a, b int) bool { return restos[a].frac > restos[b].frac })
	for k := 0; k < sobran && k < len(restos); k++ {
		decimas[restos[k].i]++
	}
	for i, d := range decimas {
		out[i] = float64(d) / 10
	}
	return out
}

// escalaRango devolve uns límites e un paso «redondos» (1, 2, 2,5, 5 × 10ⁿ)
// que abarcan [min, max] con uns cinco intervalos.
func escalaRango(min, max float64) (lo, hi, paso float64) {
	if max < min {
		min, max = max, min
	}
	span := max - min
	if span == 0 {
		span = math.Abs(max)
	}
	if span == 0 {
		span = 1
	}
	raw := span / 5
	mag := math.Pow(10, math.Floor(math.Log10(raw)))
	norm := raw / mag
	paso = 10 * mag
	for _, n := range []float64{1, 2, 2.5, 5, 10} {
		if norm <= n+1e-9 {
			paso = n * mag
			break
		}
	}
	lo = math.Floor(min/paso+1e-9) * paso
	hi = math.Ceil(max/paso-1e-9) * paso
	if hi <= lo {
		hi = lo + paso
	}
	return lo, hi, paso
}

// decimaisPaso: cantos decimais precisa un paso de escala (0,5 → 1; 0,25 → 2).
func decimaisPaso(paso float64) int {
	for d := 0; d < 4; d++ {
		p := math.Pow(10, float64(d))
		if math.Abs(paso-math.Round(paso*p)/p) < 1e-9 {
			return d
		}
	}
	return 3
}

// cortar acurta o texto con «…» ata que caiba en maxW.
func cortar(m Medidor, s string, tam float64, negrita bool, maxW float64) string {
	if m.Ancho(s, tam, negrita) <= maxW {
		return s
	}
	r := []rune(strings.TrimSpace(s))
	for len(r) > 1 {
		r = r[:len(r)-1]
		t := strings.TrimSpace(string(r)) + "…"
		if m.Ancho(t, tam, negrita) <= maxW {
			return t
		}
	}
	return "…"
}

// quebrar reparte o texto en liñas por palabras, ata maxLinhas; o que non
// caiba na última acurtase con «…».
func quebrar(m Medidor, s string, tam float64, negrita bool, maxW float64, maxLinhas int) []string {
	palabras := strings.Fields(s)
	if len(palabras) == 0 {
		return nil
	}
	var linhas []string
	actual := ""
	for _, p := range palabras {
		prob := p
		if actual != "" {
			prob = actual + " " + p
		}
		if actual == "" || m.Ancho(prob, tam, negrita) <= maxW {
			actual = prob
			continue
		}
		linhas = append(linhas, actual)
		actual = p
	}
	linhas = append(linhas, actual)
	if len(linhas) > maxLinhas {
		resto := strings.Join(linhas[maxLinhas-1:], " ")
		linhas = append(linhas[:maxLinhas-1], resto)
	}
	for i, l := range linhas {
		linhas[i] = cortar(m, l, tam, negrita, maxW)
	}
	return linhas
}
