package main

// Cores dunha imaxe (o logotipo do centro): o «kit do centro» parte delas.
// Bótanse os brancos, os negros e os grises (o fondo e o texto do logotipo) e
// quedan as cores con carácter, ordenadas polo que ocupan.

import (
	"bytes"
	"edicion/idioma"
	"encoding/base64"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"sort"
	"strings"
)

// coresDaImaxe devolve ata `max` cores (hex sen #) da imaxe, da que máis
// ocupa á que menos.
func coresDaImaxe(img image.Image, max int) []string {
	b := img.Bounds()
	if b.Empty() {
		return nil
	}
	// Abonda con mirar uns 40 000 puntos aínda que a imaxe sexa grande.
	paso := int(math.Max(1, math.Sqrt(float64(b.Dx()*b.Dy())/40000)))
	type acum struct{ r, g, b, n float64 }
	caixas := map[int]*acum{}
	for y := b.Min.Y; y < b.Max.Y; y += paso {
		for x := b.Min.X; x < b.Max.X; x += paso {
			r16, g16, b16, a16 := img.At(x, y).RGBA()
			if a16 < 0x8000 {
				continue
			}
			// Desfaise o alfa premultiplicado.
			r, g, bb := float64(r16)/float64(a16), float64(g16)/float64(a16), float64(b16)/float64(a16)
			mx, mn := math.Max(r, math.Max(g, bb)), math.Min(r, math.Min(g, bb))
			l := (mx + mn) / 2
			s := 0.0
			if mx != mn {
				s = (mx - mn) / (1 - math.Abs(2*l-1))
			}
			if l > 0.93 || l < 0.07 || s < 0.25 {
				continue
			}
			clave := int(r*7.99)<<6 | int(g*7.99)<<3 | int(bb*7.99)
			c := caixas[clave]
			if c == nil {
				c = &acum{}
				caixas[clave] = c
			}
			c.r += r
			c.g += g
			c.b += bb
			c.n++
		}
	}
	type cor struct {
		hex        string
		r, g, b, n float64
	}
	var cores []cor
	for _, c := range caixas {
		cores = append(cores, cor{r: c.r / c.n, g: c.g / c.n, b: c.b / c.n, n: c.n})
	}
	sort.Slice(cores, func(i, j int) bool { return cores[i].n > cores[j].n })
	// Xúntanse as caixas veciñas (a mesma cor cun degradado ou antialias).
	var out []cor
	for _, c := range cores {
		xunta := false
		for i := range out {
			d := math.Sqrt(math.Pow(out[i].r-c.r, 2) + math.Pow(out[i].g-c.g, 2) + math.Pow(out[i].b-c.b, 2))
			if d < 0.2 {
				out[i].n += c.n
				xunta = true
				break
			}
		}
		if !xunta {
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].n > out[j].n })
	var hex []string
	for _, c := range out {
		// Unha cor que ocupa moi pouco é ruído (bordos, antialias).
		if c.n < 0.01*out[0].n {
			continue
		}
		hex = append(hex, rgbAHex(c.r, c.g, c.b))
		if len(hex) == max {
			break
		}
	}
	return hex
}

// decodificarImaxeB64 acepta base64 ou un data URI.
func decodificarImaxeB64(datos string) (image.Image, error) {
	if i := strings.Index(datos, ","); strings.HasPrefix(datos, "data:") && i > 0 {
		datos = datos[i+1:]
	}
	bin, err := base64.StdEncoding.DecodeString(strings.TrimSpace(datos))
	if err != nil {
		return nil, idioma.Errorf("go.deseno.imaxe_codificada", "a imaxe non está ben codificada: %w", err)
	}
	img, _, err := image.Decode(bytes.NewReader(bin))
	if err != nil {
		return nil, idioma.Errorf("go.deseno.imaxe_ler", "non se pode ler a imaxe (vale PNG, JPEG ou GIF): %w", err)
	}
	return img, nil
}

// CoresDaImaxe devolve as cores principais dunha imaxe (base64 ou data URI),
// para «Tomar as cores do logotipo» no editor do deseño.
func (a *App) CoresDaImaxe(datos string) ([]string, error) {
	img, err := decodificarImaxeB64(datos)
	if err != nil {
		return nil, err
	}
	return coresDaImaxe(img, 5), nil
}
