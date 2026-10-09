package grafico

import (
	"errors"
	"fmt"
)

// xerarTarta debuxa unha tarta ou, con `rosca`, un anel co total no centro. A
// lenda vai á dereita (nome + valor, aliñados) e cada porción grande leva o seu
// porcentaxe dentro: sen liñas guía nin etiquetas que se pisen. A cor depende
// da posición da categoría, así que a mesma categoría ten a mesma cor en todas
// as tartas dun documento aínda que unha porción sexa 0.
func xerarTarta(s Spec, m Medidor, rosca bool) (*Escena, error) {
	vals := s.Series[0].Valores
	total := 0.0
	for i, v := range vals {
		if v < 0 {
			return nil, fmt.Errorf("«%s» ten un valor negativo: non se pode representar nunha tarta", s.Categorias[i])
		}
		total += v
	}
	if total <= 0 {
		return nil, errors.New("todos os valores son 0: non hai nada que repartir")
	}
	if len(vals) > 12 {
		return nil, fmt.Errorf("demasiadas porcións (%d): o máximo é 12, agrupa as pequenas", len(vals))
	}
	pct := porcentaxes(vals)

	const (
		R      = 23.0
		tamL   = 9.0 // lenda
		tamP   = 8.0 // porcentaxe dentro da porción
		filaH  = 6.4
		cadro  = 3.4
		oco    = 11.0 // separación entre a tarta e a lenda
		maxNom = 62.0
	)
	R0 := 0.0
	if rosca {
		R0 = 0.58 * R
	}
	valorTxt := func(i int) string {
		switch s.Etiquetas {
		case "valor":
			return fmtNum(vals[i], s.Decimais)
		case "ambos":
			return fmtNum(vals[i], s.Decimais) + nbsp + "·" + nbsp + fmtPct(pct[i])
		}
		return fmtPct(pct[i])
	}
	nomes := make([]string, len(vals))
	valores := make([]string, len(vals))
	var wNome, wVal float64
	for i := range vals {
		nomes[i] = cortar(m, s.Categorias[i], tamL, false, maxNom)
		valores[i] = valorTxt(i)
		wNome = max(wNome, m.Ancho(nomes[i], tamL, false))
		wVal = max(wVal, m.Ancho(valores[i], tamL, true))
	}

	alto := max(2*R, float64(len(vals))*filaH) + 2*marxe
	x0 := marxe + 2*R + oco // onde empeza a lenda
	ancho := x0 + cadro + 2 + wNome + 6 + wVal + marxe
	e := &Escena{Ancho: ancho, Alto: alto}
	cx, cy := marxe+R, alto/2

	// Porcións.
	ang := 0.0
	for i, v := range vals {
		if v <= 0 {
			continue
		}
		da := v / total * 360
		e.Elementos = append(e.Elementos, Sector{Cx: cx, Cy: cy, R: R, R0: R0, A0: ang, A1: ang + da, Cor: s.cor(i), Borde: corBlanco, BordeGrosor: 0.5})
		ang += da
	}
	// Porcentaxe dentro das porcións que o permiten (>= 7 %).
	ang = 0.0
	rt := (R + R0) / 2
	if !rosca {
		rt = 0.68 * R
	}
	for i, v := range vals {
		if v <= 0 {
			continue
		}
		da := v / total * 360
		if pct[i] >= 7 {
			x, y := puntoPolar(cx, cy, rt, ang+da/2)
			e.texto(x, y+centroBase*tamP, fmtPct(pct[i]), tamP, true, AncoraCentro, corSobre(s.cor(i)))
		}
		ang += da
	}
	if rosca {
		e.texto(cx, cy+centroBase*15-0.6, fmtNum(total, s.Decimais), 15, true, AncoraCentro, corTexto)
		if s.EtiquetaTotal != "" {
			e.texto(cx, cy+centroBase*15+3.6, s.EtiquetaTotal, 7.5, false, AncoraCentro, corTextoSuave)
		}
	}

	// Lenda, centrada en vertical.
	y := alto/2 - float64(len(vals))*filaH/2
	for i := range vals {
		yc := y + filaH/2
		e.rect(x0, yc-cadro/2, cadro, cadro, s.cor(i))
		e.texto(x0+cadro+2, yc+centroBase*tamL, nomes[i], tamL, false, AncoraEsq, corTexto)
		e.texto(ancho-marxe, yc+centroBase*tamL, valores[i], tamL, true, AncoraDta, corTexto)
		y += filaH
	}
	return e, nil
}
