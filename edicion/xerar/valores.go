package xerar

// Valores dos campos nunha variante. Hai dous camiños:
//
//   - Sen Maxima: os campos fixos, os rangos e listas numéricos e as
//     fórmulas simples (o avaliador de Lua) calcúlanse en Go. Vale para case
//     todo o que non é matemáticas (letras, formularios, bancos).
//   - Con Maxima: mándanse as mesmas liñas ca o <HIDE> de Yang e cada valor
//     sae co tex() de Maxima. A mesma semente dá os mesmos números ca Yang.
//
// As táboas sorteadas (bancos) e os campos escritos fanse sempre en Go.

import (
	"fmt"
	"math"
	"math/rand/v2"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"edicion/calculo"
	"edicion/ficha"
)

// Valor dun campo nunha variante.
type Valor struct {
	Texto string  // para a prosa
	Tex   string  // para dentro de $…$ (baleiro = usar Texto escapado)
	Num   float64 // se EsNum
	EsNum bool
	Mat   bool // vai sempre en modo matemático (valores de Maxima)
	// Filas: os campos táboa (filas do banco ou da táboa escrita).
	Filas []map[string]string
}

// Bancos devolve a ficha dun banco polo seu nome ([[nome]] ou nome).
type Bancos func(nome string) (*ficha.Ficha, error)

var xeradoresYang = map[string]bool{"n0": true, "n1": true, "n2": true, "z0": true, "z1": true, "z2": true, "q0": true, "q1": true, "q2": true}

// PrecisaMaxima di se a ficha non se pode xerar sen Maxima.
func PrecisaMaxima(f *ficha.Ficha) bool {
	if f.FormatoCorpo() == ficha.FormatoMatex {
		return true
	}
	for _, c := range f.Campos {
		if c.Orixe == ficha.OrixeCalculado && motorDe(c, f) == ficha.MotorMaxima {
			return true
		}
		if c.Orixe == ficha.OrixeAleatorio && xeradoresYang[c.Xerador] {
			return true
		}
	}
	return strings.Contains(f.Corpo, "```plot") || strings.Contains(f.Corpo, "{'")
}

func motorDe(c ficha.Campo, f *ficha.Ficha) string {
	if c.Motor != "" {
		return c.Motor
	}
	if f.FormatoCorpo() == ficha.FormatoMatex {
		return ficha.MotorMaxima
	}
	return ficha.MotorFormula
}

// formatarNumero: coma decimal galega, sen ceros de máis.
func formatarNumero(n float64) (texto, tex string) {
	return formatarNumeroDecimais(n, 0)
}

// formatarNumeroDecimais: coma formatarNumero, redondeando a `decimais`
// cifras se é > 0 (a opción «Decimais nos resultados»).
func formatarNumeroDecimais(n float64, decimais int) (texto, tex string) {
	if decimais > 0 {
		p := math.Pow10(decimais)
		n = math.Round(n*p) / p
	}
	if n == math.Trunc(n) && math.Abs(n) < 1e15 {
		s := strconv.FormatFloat(n, 'f', 0, 64)
		return s, s
	}
	s := strconv.FormatFloat(senRuidoFlotante(n), 'f', -1, 64)
	if len(s) > 12 {
		s = strings.TrimRight(strings.TrimRight(strconv.FormatFloat(n, 'f', 6, 64), "0"), ".")
	}
	return strings.Replace(s, ".", ",", 1), strings.Replace(s, ".", "{,}", 1)
}

// senRuidoFlotante quita o erro de representación binaria (0,1 + 0,2 =
// 0,30000000000000004; 0,3 · 180 = 54,00000000000001) quedando con 12 cifras
// significativas: abondan para calquera dato de aula e o ruído vai na 16.ª.
func senRuidoFlotante(n float64) float64 {
	r, err := strconv.ParseFloat(strconv.FormatFloat(n, 'g', 12, 64), 64)
	if err != nil {
		return n
	}
	return r
}

var numeroDecimalTexRe = regexp.MustCompile(`[0-9]+\.[0-9]+`)

// numerosTex: os números decimais do TeX de Maxima, sen o ruído do coma
// flotante (senRuidoFlotante), sen ceros de máis («80.0» -> «80») e con coma
// decimal galega en TeX («3.56» -> «3{,}56»). Maxima escribe sempre punto.
func numerosTex(s string) string {
	return numeroDecimalTexRe.ReplaceAllStringFunc(s, func(m string) string {
		n, err := strconv.ParseFloat(m, 64)
		if err != nil {
			return m
		}
		return strings.Replace(strconv.FormatFloat(senRuidoFlotante(n), 'f', -1, 64), ".", "{,}", 1)
	})
}

// rangoDecimal é un rango [de, ata] con paso en enteiros escalados a 10^decimais,
// para que os valores saian exactos: 0,1 + 2·0,1 en coma flotante dá
// 0,30000000000000004, e (0,5 − 0,1)/0,1 = 3,9999999999999996 deixaba fóra
// o extremo.
type rangoDecimal struct {
	de, paso, pasos int64
	decimais        int
}

// lerRango le un rango con números literais («0,1» ou «0.1»); ok = false se
// algún extremo non é un número (unha expresión de Maxima).
func lerRango(de, ata, paso string) (r rangoDecimal, ok bool, err error) {
	textos := []string{numeroMaxima(de), numeroMaxima(ata), numeroMaxima(paso)}
	var nums [3]float64
	for i, t := range textos {
		n, e := strconv.ParseFloat(t, 64)
		if e != nil || math.IsInf(n, 0) || math.IsNaN(n) {
			return r, false, nil
		}
		nums[i] = n
		if p := strings.IndexByte(t, '.'); p >= 0 && !strings.ContainsAny(t, "eE") {
			r.decimais = max(r.decimais, len(t)-p-1)
		}
	}
	if r.decimais > 9 {
		return r, false, nil
	}
	escala := math.Pow10(r.decimais)
	a, b := int64(math.Round(nums[0]*escala)), int64(math.Round(nums[1]*escala))
	r.de, r.paso = a, int64(math.Round(nums[2]*escala))
	if r.paso <= 0 || b < a {
		return r, true, fmt.Errorf("rango baleiro (de %s ata %s, paso %s)", textos[0], textos[1], textos[2])
	}
	r.pasos = (b-a)/r.paso + 1
	return r, true, nil
}

// valor é o k-ésimo valor do rango (desde 0).
func (r rangoDecimal) valor(k int64) float64 {
	return float64(r.de+r.paso*k) / math.Pow10(r.decimais)
}

func valorNumero(n float64) Valor {
	t, x := formatarNumero(n)
	return Valor{Texto: t, Tex: x, Num: n, EsNum: true}
}

func valorTexto(s string) Valor {
	if n, err := strconv.ParseFloat(strings.Replace(strings.TrimSpace(s), ",", ".", 1), 64); err == nil {
		return valorNumero(n)
	}
	return Valor{Texto: s}
}

// sortearFilas escolle `mostra` filas do banco, sen repetir e, se se pide,
// sen dúas co mesmo valor na columna `distintas`.
func sortearFilas(filas []map[string]string, mostra int, distintas string, rng *rand.Rand) ([]map[string]string, error) {
	orde := rng.Perm(len(filas))
	var out []map[string]string
	vistos := map[string]bool{}
	for _, i := range orde {
		if len(out) == mostra {
			break
		}
		if distintas != "" {
			k := filas[i][distintas]
			if vistos[k] {
				continue
			}
			vistos[k] = true
		}
		out = append(out, filas[i])
	}
	if len(out) < mostra {
		return out, fmt.Errorf("o banco só ten %d filas válidas e pídense %d", len(out), mostra)
	}
	return out, nil
}

// FilasBanco le a táboa Markdown do corpo dun banco.
func FilasBanco(b *ficha.Ficha) []map[string]string {
	var cab []string
	var filas []map[string]string
	for _, l := range strings.Split(b.Corpo, "\n") {
		l = strings.TrimSpace(l)
		if !strings.HasPrefix(l, "|") {
			if cab != nil && len(filas) > 0 {
				break
			}
			continue
		}
		celas := partirFila(l)
		if cab == nil {
			cab = celas
			continue
		}
		if separadorRe.MatchString(l) {
			continue
		}
		f := map[string]string{}
		for i, c := range cab {
			if i < len(celas) {
				f[c] = celas[i]
			}
		}
		filas = append(filas, f)
	}
	return filas
}

var separadorRe = regexp.MustCompile(`^\|?\s*:?-{2,}:?\s*(\|\s*:?-{2,}:?\s*)*\|?$`)

func partirFila(l string) []string {
	l = strings.TrimSuffix(strings.TrimPrefix(l, "|"), "|")
	var out []string
	var b strings.Builder
	for i := 0; i < len(l); i++ {
		if l[i] == '\\' && i+1 < len(l) && l[i+1] == '|' {
			b.WriteByte('|')
			i++
			continue
		}
		if l[i] == '|' {
			out = append(out, strings.TrimSpace(b.String()))
			b.Reset()
			continue
		}
		b.WriteByte(l[i])
	}
	return append(out, strings.TrimSpace(b.String()))
}

// nomeBanco: «[[banco-x]]» -> «banco-x».
func nomeBanco(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(strings.TrimPrefix(s, "[["), "]]")
	if i := strings.Index(s, "|"); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSuffix(s, ".md")
}

// valoresGo: os valores que se poden calcular sen Maxima. Os que precisan
// Maxima quedan fóra (o chamador decide). `datos`: os campos escritos.
func valoresGo(f *ficha.Ficha, rng *rand.Rand, bancos Bancos, datos map[string]any, soEnGo bool) (map[string]Valor, error) {
	vals := map[string]Valor{}
	orde, err := ficha.OrdeCalculo(f)
	if err != nil {
		return nil, err
	}
	porNome := map[string]ficha.Campo{}
	for _, c := range f.Campos {
		porNome[ficha.NomeBase(c.Nome)] = c
	}
	nums := func() map[string]float64 {
		m := map[string]float64{}
		for k, v := range vals {
			if v.EsNum {
				m[k] = v.Num
			}
		}
		return m
	}
	numero := func(expr string) (float64, error) {
		return calculo.Avaliar(strings.TrimSpace(expr), nums())
	}
	for _, n := range orde {
		c := porNome[n]
		switch c.Orixe {
		case ficha.OrixeEscrito:
			vals[n] = valorEscrito(c, datos[c.Nome])
		case ficha.OrixeTaboa:
			b, err := bancos(nomeBanco(c.Banco))
			if err != nil {
				return nil, fmt.Errorf("campo «%s»: non se atopa o banco %s: %w", c.Nome, c.Banco, err)
			}
			filas, err := sortearFilas(FilasBanco(b), c.Mostra, c.DistintasPor, rng)
			if err != nil {
				return nil, fmt.Errorf("campo «%s»: %w", c.Nome, err)
			}
			vals[n] = Valor{Filas: filas, Texto: fmt.Sprintf("%d filas", len(filas))}
		case ficha.OrixeFixo:
			if !soEnGo {
				continue
			}
			vals[n] = valorTexto(c.Valor)
		case ficha.OrixeAleatorio:
			if !soEnGo {
				continue
			}
			switch c.Xerador {
			case ficha.XeradorLista:
				if len(c.Valores) == 0 {
					return nil, fmt.Errorf("campo «%s»: lista baleira", c.Nome)
				}
				vals[n] = valorTexto(c.Valores[rng.IntN(len(c.Valores))])
			case ficha.XeradorRango:
				if r, ok, err := lerRango(porDefecto(c.De, "1"), porDefecto(c.Ata, "10"), porDefecto(c.Paso, "1")); ok {
					if err != nil {
						return nil, fmt.Errorf("campo «%s»: %w", c.Nome, err)
					}
					vals[n] = valorNumero(r.valor(rng.Int64N(r.pasos)))
					continue
				}
				a, err1 := numero(porDefecto(c.De, "1"))
				b, err2 := numero(porDefecto(c.Ata, "10"))
				p, err3 := numero(porDefecto(c.Paso, "1"))
				if err := errorsJoin(err1, err2, err3); err != nil {
					return nil, fmt.Errorf("campo «%s»: %w", c.Nome, err)
				}
				if p <= 0 || b < a {
					return nil, fmt.Errorf("campo «%s»: rango baleiro (de %v ata %v, paso %v)", c.Nome, a, b, p)
				}
				pasos := int(math.Floor((b-a)/p+1e-9)) + 1
				vals[n] = valorNumero(a + p*float64(rng.IntN(pasos)))
			default:
				return nil, fmt.Errorf("campo «%s»: o xerador %s precisa Maxima", c.Nome, c.Xerador)
			}
		case ficha.OrixeCalculado:
			if !soEnGo {
				continue
			}
			v, err := numero(c.Expr)
			if err != nil {
				return nil, fmt.Errorf("campo «%s»: %w", c.Nome, err)
			}
			vals[n] = valorNumero(v)
		}
	}
	return vals, nil
}

func errorsJoin(errs ...error) error {
	for _, e := range errs {
		if e != nil {
			return e
		}
	}
	return nil
}

// valorEscrito: o dato dun formulario (texto, número ou táboa de filas).
func valorEscrito(c ficha.Campo, d any) Valor {
	switch x := d.(type) {
	case nil:
		return Valor{}
	case float64:
		return valorNumero(x)
	case bool:
		if x {
			return Valor{Texto: "Si"}
		}
		return Valor{Texto: "Non"}
	case []any:
		var filas []map[string]string
		for _, f := range x {
			m := map[string]string{}
			if fm, ok := f.(map[string]any); ok {
				for k, v := range fm {
					m[k] = fmt.Sprint(v)
				}
			}
			filas = append(filas, m)
		}
		return Valor{Filas: filas, Texto: fmt.Sprintf("%d filas", len(filas))}
	default:
		if c.Tipo == "numero" {
			return valorTexto(fmt.Sprint(x))
		}
		return Valor{Texto: fmt.Sprint(x)}
	}
}

// nomesOrdenados: para mensaxes e probas deterministas.
func nomesOrdenados(m map[string]Valor) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
