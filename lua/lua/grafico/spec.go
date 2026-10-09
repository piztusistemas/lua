package grafico

import (
	"errors"
	"fmt"
	"math"
)

// Tipos de gráfico. Os ids son estables: gárdanse nos formularios.
const (
	Tarta          = "tarta"
	Rosca          = "rosca"
	Barras         = "barras"
	BarrasH        = "barras_horizontais"
	BarrasApiladas = "barras_apiladas"
	Linhas         = "linas"
)

// TipoInfo describe un tipo de gráfico para o catálogo (editor e asistente).
type TipoInfo struct {
	ID     string
	Nome   string
	Axuda  string
	Series bool // admite varias series (columnas de valores)
}

// Tipos devolve os tipos de gráfico na orde en que se ofrecen.
func Tipos() []TipoInfo {
	return []TipoInfo{
		{Tarta, "Tarta", "Reparto dun total en porcións, cos porcentaxes. Usa só a primeira serie.", false},
		{Rosca, "Rosca", "Coma a tarta, cun oco no medio co total.", false},
		{Barras, "Barras", "Unha barra por categoría; con varias series, agrupadas.", true},
		{BarrasH, "Barras horizontais", "Barras deitadas: bo para nomes longos ou moitas categorías.", true},
		{BarrasApiladas, "Barras apiladas", "As series empílanse nunha soa barra por categoría.", true},
		{Linhas, "Liñas", "Evolución dos valores ao longo das categorías (ordenadas).", true},
	}
}

// ExisteTipo di se `id` é un tipo de gráfico coñecido.
func ExisteTipo(id string) bool {
	for _, t := range Tipos() {
		if t.ID == id {
			return true
		}
	}
	return false
}

// Serie é unha serie de valores (un por categoría).
type Serie struct {
	Nome    string
	Valores []float64
}

// Spec é o que se quere debuxar.
type Spec struct {
	Tipo       string
	Categorias []string
	Series     []Serie
	// Ancho e Alto en mm (0 = o tamaño natural do tipo).
	Ancho, Alto float64
	// Decimais dos valores (0 = os que fagan falta, ata 2).
	Decimais int
	// Etiquetas da tarta e da rosca: «porcentaxe» (por defecto), «valor» ou «ambos».
	Etiquetas string
	// EtiquetaTotal: texto pequeno baixo o total no centro da rosca.
	EtiquetaTotal string
	// Paleta: cores «RRGGBB» das series (baleira = PaletaDefecto).
	Paleta []string
	// Descricion: texto alternativo. Se está baleiro, xérase unha lista dos datos.
	Descricion string
}

const (
	marxe        = 3.0 // mm arredor da figura
	anchoDefecto = 150.0
	maxCategoria = 40
)

// Xerar converte a especificación en escena. m mide o texto (nil = aproximado).
func Xerar(s Spec, m Medidor) (*Escena, error) {
	if m == nil {
		m = medidorAprox{}
	}
	if !ExisteTipo(s.Tipo) {
		return nil, fmt.Errorf("tipo de gráfico descoñecido: «%s»", s.Tipo)
	}
	if len(s.Series) == 0 {
		return nil, errors.New("non hai datos para representar")
	}
	n := len(s.Series[0].Valores)
	if n == 0 {
		return nil, errors.New("non hai datos para representar")
	}
	if n > maxCategoria {
		return nil, fmt.Errorf("demasiadas categorías (%d): o máximo é %d", n, maxCategoria)
	}
	for _, se := range s.Series {
		if len(se.Valores) != n {
			return nil, fmt.Errorf("a serie «%s» ten %d valores e a primeira %d", se.Nome, len(se.Valores), n)
		}
		for _, v := range se.Valores {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return nil, fmt.Errorf("a serie «%s» ten un valor que non é un número", se.Nome)
			}
		}
	}
	// Categorías: as que falten numéranse.
	cats := make([]string, n)
	for i := range cats {
		if i < len(s.Categorias) && s.Categorias[i] != "" {
			cats[i] = s.Categorias[i]
		} else {
			cats[i] = fmt.Sprint(i + 1)
		}
	}
	s.Categorias = cats
	if s.Decimais > 0 {
		s.Decimais = min(s.Decimais, 4)
	} else {
		s.Decimais = -1
	}

	var e *Escena
	var err error
	switch s.Tipo {
	case Tarta:
		e, err = xerarTarta(s, m, false)
	case Rosca:
		e, err = xerarTarta(s, m, true)
	case Barras:
		e, err = xerarBarras(s, m, false)
	case BarrasApiladas:
		e, err = xerarBarras(s, m, true)
	case BarrasH:
		e, err = xerarBarrasH(s, m)
	case Linhas:
		e, err = xerarLinhas(s, m)
	}
	if err != nil {
		return nil, err
	}
	e.Descricion = s.Descricion
	if e.Descricion == "" {
		e.Descricion = descricionPorDefecto(s)
	}
	return e, nil
}

func descricionPorDefecto(s Spec) string {
	out := ""
	for i, c := range s.Categorias {
		if i > 0 {
			out += "; "
		}
		out += c + ": "
		for j, se := range s.Series {
			if j > 0 {
				out += " / "
			}
			out += fmtNum(se.Valores[i], s.Decimais)
		}
	}
	return out
}
