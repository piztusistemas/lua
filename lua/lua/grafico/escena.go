// Package grafico é o motor de figuras de Lua: converte unha especificación de
// gráfico (tipo, categorías, series) nunha ESCENA —primitivas xeométricas en
// milímetros, sen formato— e a escena serialízase a TikZ (PDF/TeX) ou a PNG
// (Word, OpenDocument, Markdown). A xeometría calcúlase unha soa vez: un tipo
// de gráfico novo só precisa o seu xerador e sae igual en todos os formatos.
//
// O paquete non coñece formularios, plantillas nin a app: recibe os datos, as
// fontes (para medir texto) e a paleta por parámetro.
package grafico

// Ancoraxe horizontal dun texto respecto do seu punto.
const (
	AncoraEsq = iota
	AncoraCentro
	AncoraDta
)

// Escena é unha figura: un rectángulo Ancho×Alto (mm) co orixe arriba á
// esquerda e o eixe y cara abaixo (como unha imaxe).
type Escena struct {
	Ancho, Alto float64
	Descricion  string // texto alternativo (accesibilidade)
	Elementos   []Elemento
}

// Elemento é unha primitiva da escena.
type Elemento interface{ elemento() }

// Sector é unha porción de círculo ou de anel (R0 > 0). Os ángulos van en
// graos, desde as 12 en punto e en sentido horario.
type Sector struct {
	Cx, Cy, R, R0, A0, A1 float64
	Cor                   string // «RRGGBB»
	Borde                 string // cor do filete («» = sen filete)
	BordeGrosor           float64
}

// Rect é un rectángulo.
type Rect struct {
	X, Y, W, H float64
	Cor        string
	Radio      float64
}

// Linha é unha poligonal.
type Linha struct {
	Pts    [][2]float64
	Cor    string
	Grosor float64
	Trazos bool
}

// Circulo é un círculo cheo (marcadores).
type Circulo struct {
	Cx, Cy, R float64
	Cor       string
}

// Texto é unha liña de texto cuxa liña base pasa por (X, Y). Tam en puntos.
type Texto struct {
	X, Y    float64
	S       string
	Tam     float64
	Negrita bool
	Ancora  int
	Cor     string
}

func (Sector) elemento()  {}
func (Rect) elemento()    {}
func (Linha) elemento()   {}
func (Circulo) elemento() {}
func (Texto) elemento()   {}

// Constrúe a escena. Métodos curtos para que os xeradores lean coma unha receita.

func (e *Escena) rect(x, y, w, h float64, cor string) {
	e.Elementos = append(e.Elementos, Rect{X: x, Y: y, W: w, H: h, Cor: cor})
}

func (e *Escena) linha(cor string, grosor float64, trazos bool, pts ...[2]float64) {
	e.Elementos = append(e.Elementos, Linha{Pts: pts, Cor: cor, Grosor: grosor, Trazos: trazos})
}

func (e *Escena) texto(x, y float64, s string, tam float64, negrita bool, ancora int, cor string) {
	if s == "" {
		return
	}
	e.Elementos = append(e.Elementos, Texto{X: x, Y: y, S: s, Tam: tam, Negrita: negrita, Ancora: ancora, Cor: cor})
}
