package main

// Imaxe PNG do cronograma anual, que levan os documentos Word e OpenDocument
// (o PDF ten a súa propia maqueta en LaTeX, ver estilo_doc.go). Un mes
// por fila: banda de cor co nome e, á dereita, unha "pílula" coloreada por
// cada tarefa. Porta de cronograma/internal/export/cronopng.go.

import (
	"bytes"
	"image/color"
	"image/png"
	"strconv"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
)

// Constantes de maquetación da imaxe do cronograma (píxeles).
const (
	cronoPngAncho  = 1240.0
	cronoPngMarxe  = 40.0
	cronoPngBanda  = 170.0
	cronoPngOco    = 18.0
	cronoPngPildA  = 38.0 // alto da pílula
	cronoPngPildPX = 14.0 // recheo horizontal dentro da pílula
	cronoPngPildGX = 10.0
	cronoPngPildGY = 10.0
	cronoPngFilaPY = 14.0
)

func facePngCrono(ttf []byte, puntos float64) font.Face {
	f, err := truetype.Parse(ttf)
	if err != nil {
		return nil
	}
	return truetype.NewFace(f, &truetype.Options{Size: puntos})
}

// corHexCrono le "RRGGBB" (sen #); gris se non é válido.
func corHexCrono(s string) color.NRGBA {
	if len(s) != 6 {
		return color.NRGBA{R: 128, G: 128, B: 128, A: 255}
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return color.NRGBA{R: 128, G: 128, B: 128, A: 255}
	}
	return color.NRGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 255}
}

type pildoraPngCrono struct {
	x, y, w float64 // y relativo ao inicio da fila
	texto   string
	cor     corTarefaCrono
}

type filaPngCrono struct {
	nome     string
	mes      int
	pildoras []pildoraPngCrono
	alto     float64
}

// renderCronogramaPNGCrono debuxa o cronograma anual como imaxe PNG: só os
// meses e as súas tarefas (o nome do departamento, o centro e o curso van no
// resto do documento, non na imaxe). `tx` dá o texto das tarefas sen título.
func renderCronogramaPNGCrono(p PlanCrono, meses []mesCalendario, tx textosDocumento) ([]byte, error) {
	faceMes := facePngCrono(gobold.TTF, 16)
	facePild := facePngCrono(gobold.TTF, 14)

	areaX := cronoPngMarxe + cronoPngBanda + cronoPngOco
	areaW := cronoPngAncho - cronoPngMarxe - areaX

	// Contexto só para medir texto.
	mc := gg.NewContext(1, 1)
	mc.SetFontFace(facePild)

	// Pase 1: maquetación.
	filas := make([]filaPngCrono, 0, len(meses))
	for _, m := range meses {
		fila := filaPngCrono{nome: m.Nome, mes: m.Mes}
		cursorX := areaX
		topLinha := cronoPngFilaPY
		liñas := 1
		for _, t := range m.Tarefas {
			titulo := porDefectoCrono(t.Title, tx.T("SEN_TITULO"))
			tw, _ := mc.MeasureString(titulo)
			w := tw + 2*cronoPngPildPX
			if w > areaW {
				w = areaW
			}
			if cursorX+w > areaX+areaW && cursorX > areaX {
				cursorX = areaX
				topLinha += cronoPngPildA + cronoPngPildGY
				liñas++
			}
			fila.pildoras = append(fila.pildoras, pildoraPngCrono{
				x: cursorX, y: topLinha, w: w, texto: titulo, cor: resolverCorCrono(t.Color),
			})
			cursorX += w + cronoPngPildGX
		}
		contido := float64(liñas)*cronoPngPildA + float64(liñas-1)*cronoPngPildGY
		fila.alto = contido + 2*cronoPngFilaPY
		if minimo := cronoPngPildA + 2*cronoPngFilaPY; fila.alto < minimo {
			fila.alto = minimo
		}
		filas = append(filas, fila)
	}

	total := 2 * cronoPngMarxe
	for _, f := range filas {
		total += f.alto
	}

	// Pase 2: debuxo.
	dc := gg.NewContext(int(cronoPngAncho), int(total))
	dc.SetColor(color.White)
	dc.Clear()

	y := cronoPngMarxe
	for _, f := range filas {
		// Banda do mes.
		dc.SetColor(corHexCrono(coresMesCrono[f.mes]))
		dc.DrawRectangle(cronoPngMarxe, y, cronoPngBanda, f.alto)
		dc.Fill()
		dc.SetFontFace(faceMes)
		dc.SetColor(rgbCrono(0x44, 0x40, 0x3c))
		dc.DrawStringAnchored(f.nome, cronoPngMarxe+cronoPngBanda/2, y+f.alto/2, 0.5, 0.5)

		// Pílulas.
		dc.SetFontFace(facePild)
		for _, c := range f.pildoras {
			cx, cy := c.x, y+c.y
			dc.SetColor(corHexCrono(c.cor.Fondo))
			dc.DrawRoundedRectangle(cx, cy, c.w, cronoPngPildA, 8)
			dc.Fill()
			dc.SetColor(corHexCrono(c.cor.Bordo))
			dc.DrawRoundedRectangle(cx, cy, c.w, cronoPngPildA, 8)
			dc.SetLineWidth(1)
			dc.Stroke()
			dc.SetColor(corHexCrono(c.cor.Texto))
			dc.DrawStringAnchored(recortarAnchoCrono(dc, c.texto, c.w-2*cronoPngPildPX), cx+cronoPngPildPX, cy+cronoPngPildA/2, 0, 0.5)
		}

		// Separador inferior.
		dc.SetColor(rgbCrono(0xf0, 0xf0, 0xf0))
		dc.DrawRectangle(cronoPngMarxe, y+f.alto, cronoPngAncho-2*cronoPngMarxe, 1)
		dc.Fill()
		y += f.alto
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, dc.Image()); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func rgbCrono(r, g, b uint8) color.NRGBA { return color.NRGBA{R: r, G: g, B: b, A: 255} }

// recortarAnchoCrono recorta o texto engadindo "…" se non cabe no ancho dado.
func recortarAnchoCrono(dc *gg.Context, s string, maxW float64) string {
	if w, _ := dc.MeasureString(s); w <= maxW {
		return s
	}
	runas := []rune(s)
	for len(runas) > 1 {
		runas = runas[:len(runas)-1]
		cand := string(runas) + "…"
		if w, _ := dc.MeasureString(cand); w <= maxW {
			return cand
		}
	}
	return "…"
}
