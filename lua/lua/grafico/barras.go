package grafico

import (
	"errors"
	"fmt"
	"math"
)

// xerarBarras debuxa barras verticais: agrupadas (unha por serie) ou, con
// `apiladas`, unha soa barra por categoría cos segmentos das series.
func xerarBarras(s Spec, m Medidor, apiladas bool) (*Escena, error) {
	if c, mal := haiNegativos(s); mal {
		return nil, fmt.Errorf("«%s» ten un valor negativo: as barras só admiten valores 0 ou maiores", c)
	}
	nSer, n := len(s.Series), len(s.Categorias)
	max := 0.0
	if apiladas {
		for i := 0; i < n; i++ {
			suma := 0.0
			for _, se := range s.Series {
				suma += se.Valores[i]
			}
			max = math.Max(max, suma)
		}
	} else {
		_, max = intervaloValores(s)
	}
	if max <= 0 {
		return nil, errors.New("todos os valores son 0: non hai nada que debuxar")
	}
	e := &Escena{}
	k := marcoVertical(e, m, s, 0, max, nSer > 1)
	const tamV = 8.0

	if apiladas {
		bw := math.Min(k.slot*0.62, 18)
		for i := 0; i < n; i++ {
			acc := 0.0
			for j, se := range s.Series {
				v := se.Valores[i]
				if v <= 0 {
					continue
				}
				yTop, yBot := k.y(acc+v), k.y(acc)
				e.rect(k.centro(i)-bw/2, yTop, bw, yBot-yTop, s.cor(j))
				txt := fmtNum(v, s.Decimais)
				if yBot-yTop >= 4.6 && m.Ancho(txt, tamV, true) <= bw-1 {
					e.texto(k.centro(i), (yTop+yBot)/2+centroBase*tamV, txt, tamV, true, AncoraCentro, corSobre(s.cor(j)))
				}
				acc += v
			}
		}
		return e, nil
	}

	grupo := math.Min(k.slot*0.72, float64(nSer)*14)
	if nSer == 1 {
		grupo = math.Min(k.slot*0.6, 16)
	}
	const oco = 0.6
	bw := (grupo - float64(nSer-1)*oco) / float64(nSer)
	for i := 0; i < n; i++ {
		x := k.centro(i) - grupo/2
		for j, se := range s.Series {
			v := se.Valores[i]
			yTop := k.y(v)
			if v > 0 {
				e.rect(x, yTop, bw, k.y1-yTop, s.cor(j))
			}
			txt := fmtNum(v, s.Decimais)
			if m.Ancho(txt, tamV, true) <= bw+1.6 || nSer == 1 {
				e.texto(x+bw/2, yTop-1.3, txt, tamV, true, AncoraCentro, corTexto)
			}
			x += bw + oco
		}
	}
	return e, nil
}

// xerarBarrasH debuxa barras horizontais: os nomes á esquerda, os valores no
// fin de cada barra. O alto da figura crece coas categorías.
func xerarBarrasH(s Spec, m Medidor) (*Escena, error) {
	if c, mal := haiNegativos(s); mal {
		return nil, fmt.Errorf("«%s» ten un valor negativo: as barras só admiten valores 0 ou maiores", c)
	}
	_, max := intervaloValores(s)
	if max <= 0 {
		return nil, errors.New("todos os valores son 0: non hai nada que debuxar")
	}
	const (
		tamC   = 8.5
		tamV   = 8.0
		tamT   = 8.0
		maxNom = 48.0
	)
	nSer, n := len(s.Series), len(s.Categorias)
	e := &Escena{Ancho: anchoDefecto}
	if s.Ancho > 0 {
		e.Ancho = s.Ancho
	}
	nomes := make([]string, n)
	wNome := 0.0
	for i, c := range s.Categorias {
		nomes[i] = cortar(m, c, tamC, false, maxNom)
		wNome = math.Max(wNome, m.Ancho(nomes[i], tamC, false))
	}
	lo, hi, paso := escalaRango(0, max)
	dec := decimaisPaso(paso)
	wVal := 0.0
	for _, se := range s.Series {
		for _, v := range se.Valores {
			wVal = math.Max(wVal, m.Ancho(fmtNum(v, s.Decimais), tamV, true))
		}
	}
	x0 := marxe + wNome + 2.5
	x1 := e.Ancho - marxe - wVal - 2

	y := marxe
	if nSer > 1 {
		y += lenda(e, m, s, marxe, marxe, e.Ancho-2*marxe) + 2
	}
	bh, oco, aire := 4.6, 0.5, 3.4
	if nSer > 1 {
		bh = 3.4
	}
	slot := float64(nSer)*bh + float64(nSer-1)*oco + aire
	y0 := y
	y1 := y0 + float64(n)*slot
	e.Alto = y1 + 5.6 + marxe
	xv := func(v float64) float64 { return x0 + (v-lo)/(hi-lo)*(x1-x0) }

	nTicks := int(math.Round((hi-lo)/paso)) + 1
	for i := 0; i < nTicks; i++ {
		v := lo + float64(i)*paso
		cor := corGrella
		if i == 0 {
			cor = corEixe
		}
		e.linha(cor, 0.2, false, [2]float64{xv(v), y0}, [2]float64{xv(v), y1})
		e.texto(xv(v), y1+4.2, fmtNum(v, dec), tamT, false, AncoraCentro, corTextoSuave)
	}
	for i := 0; i < n; i++ {
		yg := y0 + float64(i)*slot + aire/2
		yc := yg + (float64(nSer)*bh+float64(nSer-1)*oco)/2
		e.texto(x0-2.5, yc+centroBase*tamC, nomes[i], tamC, false, AncoraDta, corTexto)
		for j, se := range s.Series {
			v := se.Valores[i]
			yb := yg + float64(j)*(bh+oco)
			if v > 0 {
				e.rect(x0, yb, xv(v)-x0, bh, s.cor(j))
			}
			e.texto(xv(v)+1.4, yb+bh/2+centroBase*tamV, fmtNum(v, s.Decimais), tamV, true, AncoraEsq, corTexto)
		}
	}
	return e, nil
}
