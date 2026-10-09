package ficha

// Validación da ficha e orde de cálculo dos campos. Os erros dinse en
// galego e co nome do campo, porque os le o docente (e a IA, que arranxa a
// ficha a partir da mensaxe exacta).

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Problema é unha falla atopada ao validar.
type Problema struct {
	Campo   string // baleiro se é da ficha
	Mensaxe string
}

func (p Problema) String() string {
	if p.Campo == "" {
		return p.Mensaxe
	}
	return fmt.Sprintf("campo «%s»: %s", p.Campo, p.Mensaxe)
}

var identRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// funcionRe: un nome de función de Maxima coma o de Yang («f(x)», «g(x,y)»).
var funcionRe = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*\(\s*[A-Za-z_][A-Za-z0-9_]*(\s*,\s*[A-Za-z_][A-Za-z0-9_]*)*\s*\)$`)

var tiposDato = []string{"texto", "textolongo", "numero", "data", "escolla", "verificacion", "imaxe", "taboa", "seccion", "expresion"}

// NomeBase: o identificador co que outros campos se refiren a este (o «f»
// de «f(x)»).
func NomeBase(nome string) string {
	if m := funcionRe.FindStringSubmatch(nome); m != nil {
		return m[1]
	}
	return nome
}

// Validar devolve todos os problemas da ficha (baleiro = válida).
func Validar(f *Ficha) []Problema {
	var ps []Problema
	mal := func(campo, formato string, args ...any) {
		ps = append(ps, Problema{Campo: campo, Mensaxe: fmt.Sprintf(formato, args...)})
	}
	if !conten(tiposFicha, f.Tipo) {
		mal("", "tipo «%s» descoñecido (vale: %s)", f.Tipo, strings.Join(tiposFicha, ", "))
	}
	if f.Formato != "" && f.Formato != FormatoMarkdown && f.Formato != FormatoMatex {
		mal("", "formato «%s» descoñecido (vale: markdown, matex)", f.Formato)
	}
	vistos := map[string]bool{}
	for _, c := range f.Campos {
		base := NomeBase(c.Nome)
		switch {
		case c.Nome == "":
			mal("", "hai un campo sen nome")
			continue
		case !identRe.MatchString(c.Nome) && !funcionRe.MatchString(c.Nome):
			mal(c.Nome, "o nome só pode levar letras sen acentos, díxitos e _, e non pode empezar por díxito")
		case vistos[base]:
			mal(c.Nome, "o nome está repetido")
		}
		vistos[base] = true
		validarCampo(c, f, mal)
	}
	if _, err := OrdeCalculo(f); err != nil {
		mal("", "%v", err)
	}
	return ps
}

func validarCampo(c Campo, f *Ficha, mal func(campo, formato string, args ...any)) {
	if c.Tipo != "" && !conten(tiposDato, c.Tipo) {
		mal(c.Nome, "tipo «%s» descoñecido", c.Tipo)
	}
	switch c.Orixe {
	case OrixeEscrito:
		if c.Tipo == "escolla" && len(c.Opcions) == 0 {
			mal(c.Nome, "unha escolla precisa opcións")
		}
		if c.Tipo == "taboa" && len(c.Columnas) == 0 {
			mal(c.Nome, "unha táboa precisa columnas")
		}
	case OrixeFixo:
		if strings.TrimSpace(c.Valor) == "" {
			mal(c.Nome, "un campo fixo precisa «valor»")
		}
	case OrixeAleatorio:
		switch {
		case c.Xerador == XeradorRango:
			if c.De == "" || c.Ata == "" {
				mal(c.Nome, "un rango precisa «de» e «ata»")
			}
		case c.Xerador == XeradorLista:
			if len(c.Valores) == 0 {
				mal(c.Nome, "unha lista precisa «valores»")
			}
		case conten(xeradoresYang, c.Xerador):
		default:
			mal(c.Nome, "xerador «%s» descoñecido (vale: rango, lista, %s)", c.Xerador, strings.Join(xeradoresYang, ", "))
		}
	case OrixeCalculado:
		if strings.TrimSpace(c.Expr) == "" {
			mal(c.Nome, "un campo calculado precisa «expr»")
		}
		if m := motorDe(c, f); m != MotorFormula && m != MotorMaxima {
			mal(c.Nome, "motor «%s» descoñecido (vale: formula, maxima)", m)
		}
	case OrixeTaboa:
		if c.Banco == "" {
			mal(c.Nome, "unha táboa sorteada precisa «banco»")
		}
		if c.Mostra <= 0 {
			mal(c.Nome, "«mostra» ten que ser polo menos 1")
		}
	case "":
		mal(c.Nome, "falta a «orixe» (vale: %s)", strings.Join(orixes, ", "))
	default:
		mal(c.Nome, "orixe «%s» descoñecida (vale: %s)", c.Orixe, strings.Join(orixes, ", "))
	}
}

// motorDe: o motor efectivo dun campo calculado. Sen indicar, nunha ficha
// matex é Maxima (é o que facía Yang) e noutra a fórmula simple.
func motorDe(c Campo, f *Ficha) string {
	if c.Motor != "" {
		return c.Motor
	}
	if f.FormatoCorpo() == FormatoMatex {
		return MotorMaxima
	}
	return MotorFormula
}

var palabraRe = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*`)

// dependencias: os campos da ficha que nomea c nas súas expresións.
func dependencias(c Campo, nomes map[string]bool) []string {
	var deps []string
	visto := map[string]bool{}
	for _, txt := range []string{c.Expr, c.De, c.Ata, c.Paso} {
		for _, p := range palabraRe.FindAllString(sinCadeas(txt), -1) {
			if nomes[p] && p != NomeBase(c.Nome) && !visto[p] {
				visto[p] = true
				deps = append(deps, p)
			}
		}
	}
	return deps
}

// sinCadeas quita os literais "…" para non tomar palabras de dentro por
// nomes de campo.
func sinCadeas(s string) string {
	var b strings.Builder
	dentro := false
	for _, r := range s {
		if r == '"' {
			dentro = !dentro
			continue
		}
		if !dentro {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// OrdeCalculo devolve os nomes dos campos na orde en que hai que calculalos:
// cada un despois dos que usa e, se non hai dependencias que o impidan, na
// orde en que están escritos (a orde do <HIDE> de Yang). Erro se hai un ciclo.
func OrdeCalculo(f *Ficha) ([]string, error) {
	nomes := map[string]bool{}
	pos := map[string]int{}
	for i, c := range f.Campos {
		nomes[NomeBase(c.Nome)] = true
		pos[NomeBase(c.Nome)] = i
	}
	pendentes := map[string]int{}
	usanme := map[string][]string{}
	for _, c := range f.Campos {
		b := NomeBase(c.Nome)
		deps := dependencias(c, nomes)
		pendentes[b] = len(deps)
		for _, d := range deps {
			usanme[d] = append(usanme[d], b)
		}
	}
	var listos, orde []string
	for _, c := range f.Campos {
		if b := NomeBase(c.Nome); pendentes[b] == 0 {
			listos = append(listos, b)
		}
	}
	for len(listos) > 0 {
		sort.SliceStable(listos, func(i, j int) bool { return pos[listos[i]] < pos[listos[j]] })
		n := listos[0]
		listos = listos[1:]
		orde = append(orde, n)
		for _, u := range usanme[n] {
			pendentes[u]--
			if pendentes[u] == 0 {
				listos = append(listos, u)
			}
		}
	}
	if len(orde) < len(pendentes) {
		var ciclo []string
		for _, c := range f.Campos {
			if b := NomeBase(c.Nome); pendentes[b] > 0 {
				ciclo = append(ciclo, b)
			}
		}
		return nil, fmt.Errorf("os campos %s dependen uns doutros en círculo", strings.Join(ciclo, ", "))
	}
	return orde, nil
}
