package grafico

// xerarLinhas debuxa unha poligonal por serie sobre as categorías, en orde.
// Con unha soa serie escríbese o valor
// de cada punto (ata 14).
func xerarLinhas(s Spec, m Medidor) (*Escena, error) {
	min, max := intervaloValores(s)
	// O eixe abarca o cero, agás se os valores están xuntos lonxe del (medias
	// entre 5,8 e 7,1): nese caso empeza preto dos datos para que se vexa a evolución.
	if min > 0 && (max-min) >= 0.5*max {
		min = 0
	}
	if max < 0 {
		max = 0
	}
	e := &Escena{}
	k := marcoVertical(e, m, s, min, max, len(s.Series) > 1)
	n := len(s.Categorias)
	const tamV = 8.0
	for j, se := range s.Series {
		pts := make([][2]float64, n)
		for i, v := range se.Valores {
			pts[i] = [2]float64{k.centro(i), k.y(v)}
		}
		if n > 1 {
			e.linha(s.cor(j), 0.55, false, pts...)
		}
		for _, p := range pts {
			e.Elementos = append(e.Elementos, Circulo{Cx: p[0], Cy: p[1], R: 1.5, Cor: corBlanco}, Circulo{Cx: p[0], Cy: p[1], R: 1.0, Cor: s.cor(j)})
		}
		if len(s.Series) == 1 && n <= 14 {
			for i, v := range se.Valores {
				e.texto(pts[i][0], pts[i][1]-2.6, fmtNum(v, s.Decimais), tamV, true, AncoraCentro, corTexto)
			}
		}
	}
	return e, nil
}
