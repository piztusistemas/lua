package calculo

// Avaliador de fórmulas dos nodos «calcular» das tuberías: aritmética sobre as
// variables que chegan polas entradas (a, b, c...). Sen dependencias e sen
// executar nada do usuario: é un analizador descendente recursivo.
//
//	expr   = cmp
//	cmp    = suma [(< | <= | > | >= | == | !=) suma]     -> 1 ou 0
//	suma   = prod {(+ | -) prod}
//	prod   = pot {(* | / | %) pot}
//	pot    = unario [^ pot]
//	unario = [-] primario
//	primar = número | variable | función(args) | (expr)
//
// Números con PUNTO decimal (a coma separa argumentos). Funcións: min, max,
// abs, sqrt, round/redondear(x[, decimais]), suma, media e se(condición, a, b).

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

var errDivisionCero = errors.New("división por cero")

type formulaP struct {
	s    []rune
	i    int
	vars map[string]float64
	// soft > 0 dentro dos argumentos de se(): unha división por cero non corta
	// a análise, anótase en softErr e só é erro se esa rama é a que se usa
	// (así se(b==0, 0, a/b) non falla cando b é 0).
	soft    int
	softErr error
}

// avaliarFormula calcula `formula` coas variables dadas. Os nomes das
// variables non distinguen maiúsculas.
func avaliarFormula(formula string, vars map[string]float64) (float64, error) {
	p := &formulaP{s: []rune(formula), vars: map[string]float64{}}
	for k, v := range vars {
		p.vars[strings.ToLower(k)] = v
	}
	if strings.TrimSpace(formula) == "" {
		return 0, errors.New("a fórmula está baleira")
	}
	v, err := p.expr()
	if err != nil {
		return 0, err
	}
	p.espazos()
	if p.i < len(p.s) {
		return 0, fmt.Errorf("sobra «%s» ao final da fórmula", string(p.s[p.i:]))
	}
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, errors.New("o resultado non é un número")
	}
	return v, nil
}

// comprobarFormula valida a sintaxe e que só use as variables `nomes` (sen
// datos: proba con 1 en cada unha e ignora a división por cero).
func comprobarFormula(formula string, nomes []string) error {
	vars := map[string]float64{}
	for _, n := range nomes {
		vars[n] = 1
	}
	if _, err := avaliarFormula(formula, vars); err != nil && !errors.Is(err, errDivisionCero) {
		return err
	}
	return nil
}

func (p *formulaP) espazos() {
	for p.i < len(p.s) && unicode.IsSpace(p.s[p.i]) {
		p.i++
	}
}

func (p *formulaP) peek() rune {
	p.espazos()
	if p.i < len(p.s) {
		return p.s[p.i]
	}
	return 0
}

func (p *formulaP) consumir(op string) bool {
	p.espazos()
	r := []rune(op)
	if p.i+len(r) > len(p.s) || string(p.s[p.i:p.i+len(r)]) != op {
		return false
	}
	p.i += len(r)
	return true
}

func bool01(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func (p *formulaP) expr() (float64, error) {
	a, err := p.suma()
	if err != nil {
		return 0, err
	}
	for _, op := range []string{"<=", ">=", "==", "!=", "<", ">"} {
		if p.consumir(op) {
			b, err := p.suma()
			if err != nil {
				return 0, err
			}
			switch op {
			case "<=":
				return bool01(a <= b), nil
			case ">=":
				return bool01(a >= b), nil
			case "==":
				return bool01(a == b), nil
			case "!=":
				return bool01(a != b), nil
			case "<":
				return bool01(a < b), nil
			default:
				return bool01(a > b), nil
			}
		}
	}
	return a, nil
}

func (p *formulaP) suma() (float64, error) {
	a, err := p.prod()
	if err != nil {
		return 0, err
	}
	for {
		switch {
		case p.consumir("+"):
			b, err := p.prod()
			if err != nil {
				return 0, err
			}
			a += b
		case p.consumir("-"):
			b, err := p.prod()
			if err != nil {
				return 0, err
			}
			a -= b
		default:
			return a, nil
		}
	}
}

func (p *formulaP) prod() (float64, error) {
	a, err := p.pot()
	if err != nil {
		return 0, err
	}
	for {
		switch {
		case p.consumir("*"):
			b, err := p.pot()
			if err != nil {
				return 0, err
			}
			a *= b
		case p.consumir("/"):
			b, err := p.pot()
			if err != nil {
				return 0, err
			}
			if b == 0 {
				if p.soft > 0 {
					p.softErr = errDivisionCero
					a = 0
					continue
				}
				return 0, errDivisionCero
			}
			a /= b
		case p.consumir("%"):
			b, err := p.pot()
			if err != nil {
				return 0, err
			}
			if b == 0 {
				if p.soft > 0 {
					p.softErr = errDivisionCero
					a = 0
					continue
				}
				return 0, errDivisionCero
			}
			a = math.Mod(a, b)
		default:
			return a, nil
		}
	}
}

func (p *formulaP) pot() (float64, error) {
	a, err := p.unario()
	if err != nil {
		return 0, err
	}
	if p.consumir("^") {
		b, err := p.pot() // asociativa pola dereita
		if err != nil {
			return 0, err
		}
		return math.Pow(a, b), nil
	}
	return a, nil
}

func (p *formulaP) unario() (float64, error) {
	if p.consumir("-") {
		v, err := p.unario()
		return -v, err
	}
	if p.consumir("+") {
		return p.unario()
	}
	return p.primario()
}

func (p *formulaP) primario() (float64, error) {
	c := p.peek()
	switch {
	case c == 0:
		return 0, errors.New("a fórmula remata sen completar")
	case c == '(':
		p.i++
		v, err := p.expr()
		if err != nil {
			return 0, err
		}
		if !p.consumir(")") {
			return 0, errors.New("falta pechar un parénteses")
		}
		return v, nil
	case unicode.IsDigit(c) || c == '.':
		ini := p.i
		for p.i < len(p.s) && (unicode.IsDigit(p.s[p.i]) || p.s[p.i] == '.') {
			p.i++
		}
		v, err := strconv.ParseFloat(string(p.s[ini:p.i]), 64)
		if err != nil {
			return 0, fmt.Errorf("número non válido «%s» (usa punto decimal: 0.4)", string(p.s[ini:p.i]))
		}
		return v, nil
	case unicode.IsLetter(c) || c == '_':
		ini := p.i
		for p.i < len(p.s) && (unicode.IsLetter(p.s[p.i]) || unicode.IsDigit(p.s[p.i]) || p.s[p.i] == '_') {
			p.i++
		}
		nome := strings.ToLower(string(p.s[ini:p.i]))
		if p.peek() == '(' {
			p.i++
			return p.funcion(nome)
		}
		v, ok := p.vars[nome]
		if !ok {
			return 0, fmt.Errorf("a variable «%s» non existe (as entradas chámanse a, b, c...)", nome)
		}
		return v, nil
	}
	return 0, fmt.Errorf("carácter inesperado «%c» na fórmula", c)
}

func (p *formulaP) funcion(nome string) (float64, error) {
	var args []float64
	var errsArgs []error
	perezoso := nome == "se" || nome == "if"
	if !p.consumir(")") {
		for {
			if perezoso {
				p.soft++
				anterior := p.softErr
				p.softErr = nil
				v, err := p.expr()
				errsArgs = append(errsArgs, p.softErr)
				p.softErr = anterior
				p.soft--
				if err != nil {
					return 0, err
				}
				args = append(args, v)
			} else {
				v, err := p.expr()
				if err != nil {
					return 0, err
				}
				args = append(args, v)
			}
			if p.consumir(",") || p.consumir(";") {
				continue
			}
			if p.consumir(")") {
				break
			}
			return 0, fmt.Errorf("na función %s: agardábase «,» ou «)»", nome)
		}
	}
	necesita := func(min, max int) error {
		if len(args) < min || (max >= 0 && len(args) > max) {
			return fmt.Errorf("a función %s non admite %d argumentos", nome, len(args))
		}
		return nil
	}
	switch nome {
	case "min", "max":
		if err := necesita(1, -1); err != nil {
			return 0, err
		}
		r := args[0]
		for _, a := range args[1:] {
			if (nome == "min" && a < r) || (nome == "max" && a > r) {
				r = a
			}
		}
		return r, nil
	case "abs":
		if err := necesita(1, 1); err != nil {
			return 0, err
		}
		return math.Abs(args[0]), nil
	case "sqrt", "raiz":
		if err := necesita(1, 1); err != nil {
			return 0, err
		}
		if args[0] < 0 {
			return 0, errors.New("a raíz dun número negativo non existe")
		}
		return math.Sqrt(args[0]), nil
	case "round", "redondear":
		if err := necesita(1, 2); err != nil {
			return 0, err
		}
		n := 0.0
		if len(args) == 2 {
			n = args[1]
		}
		k := math.Pow(10, n)
		return math.Round(args[0]*k) / k, nil
	case "suma", "sum", "media", "avg":
		if err := necesita(1, -1); err != nil {
			return 0, err
		}
		s := 0.0
		for _, a := range args {
			s += a
		}
		if nome == "media" || nome == "avg" {
			s /= float64(len(args))
		}
		return s, nil
	case "se", "if":
		if err := necesita(3, 3); err != nil {
			return 0, err
		}
		elixida := 2
		if args[0] != 0 {
			elixida = 1
		}
		// Un erro na condición ou na rama que se usa é un erro; na outra, non.
		for _, i := range []int{0, elixida} {
			if errsArgs[i] != nil {
				if p.soft > 0 {
					p.softErr = errsArgs[i]
					return 0, nil
				}
				return 0, errsArgs[i]
			}
		}
		return args[elixida], nil
	}
	return 0, fmt.Errorf("a función «%s» non existe (hai min, max, abs, sqrt, round, suma, media, se)", nome)
}

// Avaliar calcula unha fórmula coas variables dadas (o avaliador dos nodos
// «calcular» de Lua: aritmética, comparacións, min, max, abs, sqrt, round,
// suma, media e se()).
func Avaliar(formula string, vars map[string]float64) (float64, error) {
	return avaliarFormula(formula, vars)
}

// Comprobar valida a sintaxe dunha fórmula e que só use os nomes dados.
func Comprobar(formula string, nomes []string) error { return comprobarFormula(formula, nomes) }
