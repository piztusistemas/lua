package grafico

import (
	"bytes"
	"errors"
	"image/png"
	"math"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
)

// PNG rasteriza a escena a `dpi` puntos por polgada (300 para imprimir) sobre
// fondo transparente. O texto debúxase con Open Sans (Fontes).
func (e *Escena) PNG(dpi float64, f Fontes) ([]byte, error) {
	if dpi <= 0 {
		dpi = 300
	}
	reg, err := lerTTF(f.Regular)
	if err != nil {
		return nil, errors.New("gráfico: falta a fonte para o texto")
	}
	neg := reg
	if g, err := lerTTF(f.Negrita); err == nil {
		neg = g
	}
	k := dpi / 25.4 // píxeles por mm
	dc := gg.NewContext(int(math.Ceil(e.Ancho*k)), int(math.Ceil(e.Alto*k)))

	type clave struct {
		tam float64
		neg bool
	}
	caras := map[clave]font.Face{}
	cara := func(tam float64, negrita bool) font.Face {
		c := clave{tam, negrita}
		if fc, ok := caras[c]; ok {
			return fc
		}
		fonte := reg
		if negrita {
			fonte = neg
		}
		fc := truetype.NewFace(fonte, &truetype.Options{Size: tam, DPI: dpi, Hinting: font.HintingNone})
		caras[c] = fc
		return fc
	}

	for _, el := range e.Elementos {
		switch x := el.(type) {
		case Rect:
			dc.SetHexColor(x.Cor)
			if x.Radio > 0 {
				dc.DrawRoundedRectangle(x.X*k, x.Y*k, x.W*k, x.H*k, x.Radio*k)
			} else {
				dc.DrawRectangle(x.X*k, x.Y*k, x.W*k, x.H*k)
			}
			dc.Fill()
		case Circulo:
			dc.SetHexColor(x.Cor)
			dc.DrawCircle(x.Cx*k, x.Cy*k, x.R*k)
			dc.Fill()
		case Linha:
			if len(x.Pts) < 2 {
				continue
			}
			dc.SetHexColor(x.Cor)
			dc.SetLineWidth(x.Grosor * k)
			dc.SetLineCap(gg.LineCapRound)
			dc.SetLineJoin(gg.LineJoinRound)
			if x.Trazos {
				dc.SetDash(2*k, 1.5*k)
			} else {
				dc.SetDash()
			}
			dc.MoveTo(x.Pts[0][0]*k, x.Pts[0][1]*k)
			for _, p := range x.Pts[1:] {
				dc.LineTo(p[0]*k, p[1]*k)
			}
			dc.Stroke()
			dc.SetDash()
		case Sector:
			caminoSector(dc, x, k)
			dc.SetHexColor(x.Cor)
			if x.Borde != "" && x.BordeGrosor > 0 {
				dc.FillPreserve()
				dc.SetHexColor(x.Borde)
				dc.SetLineWidth(x.BordeGrosor * k)
				dc.SetLineJoin(gg.LineJoinRound)
				dc.Stroke()
			} else {
				dc.Fill()
			}
		case Texto:
			dc.SetFontFace(cara(x.Tam, x.Negrita))
			dc.SetHexColor(x.Cor)
			w, _ := dc.MeasureString(x.S)
			px := x.X * k
			switch x.Ancora {
			case AncoraCentro:
				px -= w / 2
			case AncoraDta:
				px -= w
			}
			dc.DrawString(x.S, px, x.Y*k)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, dc.Image()); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// caminoSector traza o contorno dunha porción (ou dun anel) como poligonal fina.
func caminoSector(dc *gg.Context, s Sector, k float64) {
	n := int(math.Max(2, math.Ceil(math.Abs(s.A1-s.A0)/1.2)))
	pt := func(r, a float64) (float64, float64) {
		x, y := puntoPolar(s.Cx, s.Cy, r, a)
		return x * k, y * k
	}
	ang := func(i int) float64 { return s.A0 + (s.A1-s.A0)*float64(i)/float64(n) }
	if math.Abs(s.A1-s.A0) >= 359.99 {
		// Círculo ou anel completo: sen radios de pechamento (deixarían unha
		// costura ao debuxar o filete). O interior vai en sentido contrario.
		circulo := func(r float64, reves bool) {
			const pasos = 360
			dc.NewSubPath()
			for i := 0; i <= pasos; i++ {
				j := i
				if reves {
					j = pasos - i
				}
				x, y := pt(r, s.A0+360*float64(j)/pasos)
				if i == 0 {
					dc.MoveTo(x, y)
				} else {
					dc.LineTo(x, y)
				}
			}
			dc.ClosePath()
		}
		circulo(s.R, false)
		if s.R0 > 0 {
			circulo(s.R0, true)
		}
		return
	}
	dc.NewSubPath()
	if s.R0 > 0 {
		x, y := pt(s.R, s.A0)
		dc.MoveTo(x, y)
		for i := 1; i <= n; i++ {
			dc.LineTo(pt(s.R, ang(i)))
		}
		for i := n; i >= 0; i-- {
			dc.LineTo(pt(s.R0, ang(i)))
		}
	} else {
		dc.MoveTo(s.Cx*k, s.Cy*k)
		for i := 0; i <= n; i++ {
			dc.LineTo(pt(s.R, ang(i)))
		}
	}
	dc.ClosePath()
}
