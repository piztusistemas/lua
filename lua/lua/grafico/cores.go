package grafico

import (
	"math"
	"strconv"
	"strings"
)

// PaletaDefecto son as cores das series, na orde en que se usan: as primeiras
// son as máis distintas entre si (tamén para daltonismo e en escala de grises).
var PaletaDefecto = []string{"3B5BDB", "F08C00", "2F9E44", "E64980", "15AABF", "AE3EC9", "F03E3E", "868E96"}

const (
	corTexto      = "292524"
	corTextoSuave = "57534E"
	corGrella     = "E7E5E4"
	corEixe       = "A8A29E"
	corBlanco     = "FFFFFF"
)

func (s Spec) cor(i int) string {
	p := s.Paleta
	if len(p) == 0 {
		p = PaletaDefecto
	}
	return strings.TrimPrefix(strings.ToUpper(p[i%len(p)]), "#")
}

func canal(hex string, desde int) float64 {
	v, err := strconv.ParseUint(hex[desde:desde+2], 16, 8)
	if err != nil {
		return 0
	}
	c := float64(v) / 255
	if c <= 0.03928 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

func luminancia(hex string) float64 {
	if len(hex) != 6 {
		return 0.5
	}
	return 0.2126*canal(hex, 0) + 0.7152*canal(hex, 2) + 0.0722*canal(hex, 4)
}

// corSobre elixe branco ou texto escuro, o que máis contraste dea sobre `fondo`.
func corSobre(fondo string) string {
	l := luminancia(fondo)
	if (1.05)/(l+0.05) >= (l+0.05)/(luminancia(corTexto)+0.05) {
		return corBlanco
	}
	return corTexto
}
