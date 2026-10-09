package ficha

// As seccións do corpo: cada título «## …» (fóra dos bloques de código)
// abre unha. O título decide quen a ve: o enunciado vai sempre; o resultado
// e a resolución só no solucionario (a regra das lapelas de Yang).

import (
	"regexp"
	"strings"
	"unicode"
)

// Roles dunha sección.
const (
	RolPreambulo  = "preambulo" // o texto antes do primeiro título
	RolEnunciado  = "enunciado"
	RolResultado  = "resultado"  // resposta final
	RolResolucion = "resolucion" // paso a paso
	RolContido    = "contido"    // calquera outro título: vai sempre
)

// Seccion é un anaco do corpo.
type Seccion struct {
	Titulo string `json:"Titulo"` // sen os «## »; baleiro no preámbulo
	Rol    string `json:"Rol"`
	Texto  string `json:"Texto"` // sen o título nin as liñas en branco dos extremos
	// Lingua: «es» en «## Enunciado [es]» (versión noutra lingua dun exame
	// bilingüe). Baleira = a lingua principal da ficha.
	Lingua string `json:"Lingua,omitempty"`
}

var linguaTituloRe = regexp.MustCompile(`^(.*?)\s*\[([a-z]{2,3})\]$`)

// partirLingua: «Enunciado [es]» -> («Enunciado», «es»).
func partirLingua(t string) (string, string) {
	if m := linguaTituloRe.FindStringSubmatch(strings.TrimSpace(t)); m != nil {
		return m[1], m[2]
	}
	return t, ""
}

// Solucionario di se a sección só vai na versión con solucións.
func (s Seccion) Solucionario() bool {
	return s.Rol == RolResultado || s.Rol == RolResolucion
}

var rolesPorTitulo = map[string]string{
	"enunciado":    RolEnunciado,
	"resultado":    RolResultado,
	"resultados":   RolResultado,
	"resposta":     RolResultado,
	"respostas":    RolResultado,
	"clave":        RolResultado,
	"resolucion":   RolResolucion,
	"solucion":     RolResolucion,
	"solucions":    RolResolucion,
	"solucionario": RolResolucion,
}

// RolDoTitulo: o rol que corresponde a un título de sección.
func RolDoTitulo(titulo string) string {
	titulo, _ = partirLingua(titulo)
	if r, ok := rolesPorTitulo[clave(titulo)]; ok {
		return r
	}
	return RolContido
}

var senAcentos = strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n")

// clave: minúsculas e sen acentos («Resolución» -> «resolucion»).
func clave(s string) string {
	return senAcentos.Replace(strings.Map(unicode.ToLower, strings.TrimSpace(s)))
}

// Seccions parte o corpo. O preámbulo só aparece se ten texto. Nun
// exercicio, un preámbulo sen títulos despois é o enunciado.
func (f *Ficha) Seccions() []Seccion {
	var out []Seccion
	actual := Seccion{Rol: RolPreambulo}
	var linas []string
	pechar := func() {
		actual.Texto = strings.Trim(strings.Join(linas, "\n"), "\n")
		if actual.Rol != RolPreambulo || strings.TrimSpace(actual.Texto) != "" {
			out = append(out, actual)
		}
		linas = nil
	}
	enCodigo := false
	for _, l := range strings.Split(f.Corpo, "\n") {
		if strings.HasPrefix(strings.TrimSpace(l), "```") {
			enCodigo = !enCodigo
		}
		if !enCodigo && strings.HasPrefix(l, "## ") {
			pechar()
			t := strings.TrimSpace(l[3:])
			_, lingua := partirLingua(t)
			actual = Seccion{Titulo: t, Rol: RolDoTitulo(t), Lingua: lingua}
			continue
		}
		linas = append(linas, l)
	}
	pechar()
	if f.Tipo == TipoExercicio && len(out) == 1 && out[0].Rol == RolPreambulo {
		out[0].Rol = RolEnunciado
	}
	return out
}

// CorpoDeSeccions compón un corpo a partir de seccións (o inverso de
// Seccions para seccións con título).
func CorpoDeSeccions(ss []Seccion) string {
	var b strings.Builder
	for i, s := range ss {
		if i > 0 {
			b.WriteString("\n")
		}
		if s.Titulo != "" {
			b.WriteString("## " + s.Titulo + "\n\n")
		}
		if s.Texto != "" {
			b.WriteString(s.Texto + "\n")
		}
	}
	return b.String()
}
