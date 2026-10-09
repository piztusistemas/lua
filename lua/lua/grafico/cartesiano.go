package grafico

import (
	"fmt"
	"math"
)

func puntoPolar(cx, cy, r, graos float64) (x, y float64) {
	a := graos * math.Pi / 180
	return cx + r*math.Sin(a), cy - r*math.Cos(a)
}

func nomeSerie(s Spec, i int) string {
	if s.Series[i].Nome != "" {
		return s.Series[i].Nome
	}
	return fmt.Sprintf("Serie %d", i+1)
}

// lenda pon as series (cadro de cor + nome) en filas que se rompen ao chegar
// ao ancho maxW. Devolve o alto que ocupa.
func lenda(e *Escena, m Medidor, s Spec, x, y, maxW float64) float64 {
	const (
		tam   = 8.5
		fila  = 5.4
		cadro = 3.0
	)
	cx, cy := x, y
	for i := range s.Series {
		nome := cortar(m, nomeSerie(s, i), tam, false, maxW-cadro-1.8)
		w := cadro + 1.8 + m.Ancho(nome, tam, false)
		if cx > x && cx+w > x+maxW {
			cx, cy = x, cy+fila
		}
		e.rect(cx, cy+(fila-cadro)/2, cadro, cadro, s.cor(i))
		e.texto(cx+cadro+1.8, cy+fila/2+centroBase*tam, nome, tam, false, AncoraEsq, corTexto)
		cx += w + 6
	}
	return cy - y + fila
}

// marco é a zona de debuxo dun gráfico cartesiano vertical (categorías en x,
// valores en y) e a escala dos valores.
type marco struct {
	x0, x1, y0, y1 float64
	lo, hi, paso   float64
	slot           float64
}

func (k marco) y(v float64) float64  { return k.y1 - (v-k.lo)/(k.hi-k.lo)*(k.y1-k.y0) }
func (k marco) centro(i int) float64 { return k.x0 + (float64(i)+0.5)*k.slot }

// marcoVertical calcula o marco e debuxa o que non depende das series: a
// lenda (se `conLenda`), a grella, os valores do eixe e os nomes das
// categorías (en varias liñas ou máis pequenos se non caben).
func marcoVertical(e *Escena, m Medidor, s Spec, vmin, vmax float64, conLenda bool) marco {
	const (
		tamT    = 8.0
		altoMin = 72.0
		plotMin = 34.0
	)
	e.Ancho = anchoDefecto
	if s.Ancho > 0 {
		e.Ancho = s.Ancho
	}
	lo, hi, paso := escalaRango(vmin, vmax)
	dec := decimaisPaso(paso)
	nTicks := int(math.Round((hi-lo)/paso)) + 1
	ticks := make([]string, nTicks)
	wT := 0.0
	for i := range ticks {
		ticks[i] = fmtNum(lo+float64(i)*paso, dec)
		wT = math.Max(wT, m.Ancho(ticks[i], tamT, false))
	}
	k := marco{lo: lo, hi: hi, paso: paso, x0: marxe + wT + 2.5, x1: e.Ancho - marxe}
	n := len(s.Categorias)
	k.slot = (k.x1 - k.x0) / float64(n)

	// Nomes das categorías: baixa o corpo ata que caiban en ≤ 3 liñas.
	tamC := 8.5
	var linhas [][]string
	for ; ; tamC -= 1 {
		linhas = linhas[:0]
		cabe := true
		for _, c := range s.Categorias {
			l := quebrar(m, c, tamC, false, k.slot-1.5, 3)
			linhas = append(linhas, l)
			for _, t := range l {
				if m.Ancho(t, tamC, false) > k.slot-1.5 {
					cabe = false
				}
			}
		}
		if cabe || tamC <= 6.5 {
			break
		}
	}
	maxL := 1
	for _, l := range linhas {
		maxL = max(maxL, len(l))
	}
	hCat := float64(maxL)*tamC*mmPorPt*1.25 + 2.2

	yTop := marxe
	if conLenda {
		yTop += lenda(e, m, s, marxe, marxe, e.Ancho-2*marxe) + 2
	}
	e.Alto = math.Max(altoMin, yTop+3.5+plotMin+hCat+marxe)
	if s.Alto > 0 {
		e.Alto = math.Max(s.Alto, yTop+3.5+plotMin+hCat+marxe)
	}
	k.y0 = yTop + 3.5 // aire para os valores sobre as barras
	k.y1 = e.Alto - marxe - hCat

	for i, t := range ticks {
		y := k.y(lo + float64(i)*paso)
		cor := corGrella
		if math.Abs(lo+float64(i)*paso) < 1e-9 {
			cor = corEixe
		}
		e.linha(cor, 0.2, false, [2]float64{k.x0, y}, [2]float64{k.x1, y})
		e.texto(k.x0-2, y+centroBase*tamT, t, tamT, false, AncoraDta, corTextoSuave)
	}
	for i, l := range linhas {
		for j, t := range l {
			e.texto(k.centro(i), k.y1+3.6+float64(j)*tamC*mmPorPt*1.25, t, tamC, false, AncoraCentro, corTexto)
		}
	}
	return k
}

// intervaloValores devolve o mínimo e o máximo de todas as series.
func intervaloValores(s Spec) (min, max float64) {
	min, max = math.Inf(1), math.Inf(-1)
	for _, se := range s.Series {
		for _, v := range se.Valores {
			min, max = math.Min(min, v), math.Max(max, v)
		}
	}
	return min, max
}

func haiNegativos(s Spec) (string, bool) {
	for _, se := range s.Series {
		for i, v := range se.Valores {
			if v < 0 {
				return s.Categorias[i], true
			}
		}
	}
	return "", false
}
