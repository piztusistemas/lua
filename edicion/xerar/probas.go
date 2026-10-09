package xerar

// Probas dunha ficha (a lapela «Test» de Yang, para todas as fichas): xérase
// en N sementes e compróbase que ningún valor dá erro, infinito ou
// indeterminación, que se cumpren as condicións da ficha e que as variantes
// son de verdade distintas.

import (
	"fmt"
	"math"
	"os"
	"regexp"
	"sort"
	"strings"

	"edicion/cas"
	"edicion/ficha"
	"edicion/latex"
)

// Fallo nunha semente.
type Fallo struct {
	Semente int    `json:"semente"`
	Campo   string `json:"campo"`
	Mensaxe string `json:"mensaxe"`
}

// InformeProbas: o resultado das probas.
type InformeProbas struct {
	Sementes  int      `json:"sementes"`
	Fallos    []Fallo  `json:"fallos"`
	Distintas int      `json:"distintas"` // variantes distintas
	Campos    []string `json:"campos"`
	// Mostra: os valores das primeiras sementes (para a táboa da UI).
	Mostra []map[string]string `json:"mostra"`
	Avisos []string            `json:"avisos"`
}

// Ben: sen fallos.
func (i InformeProbas) Ben() bool { return len(i.Fallos) == 0 }

// Resumo en texto (para devolverllo á IA cando ten que arranxar a ficha).
func (i InformeProbas) Resumo() string {
	if i.Ben() {
		return fmt.Sprintf("%d sementes sen erros; %d variantes distintas.", i.Sementes, i.Distintas)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%d fallos en %d sementes:\n", len(i.Fallos), i.Sementes)
	vistos := map[string]bool{}
	for _, f := range i.Fallos {
		k := f.Campo + f.Mensaxe
		if vistos[k] {
			continue
		}
		vistos[k] = true
		fmt.Fprintf(&b, "- semente %d, %s: %s\n", f.Semente, f.Campo, f.Mensaxe)
	}
	return b.String()
}

var malRe = regexp.MustCompile(`(?i)\\infty|\binf\b|\bminf\b|\bund\b|\bind\b|\binfinity\b|\bnan\b|division by zero|división por cero|expt: undefined|\berror\b`)

// Probar xera a ficha en n sementes (1..n) e devolve o informe.
func Probar(f *ficha.Ficha, fichas Fichas, o Opcions, n int) (InformeProbas, error) {
	if n <= 0 {
		n = 25
		if f.Probas != nil && f.Probas.Sementes > 0 {
			n = f.Probas.Sementes
		}
	}
	inf := InformeProbas{Sementes: n}
	if fichas == nil {
		fichas = func(string) (*ficha.Ficha, error) { return nil, fmt.Errorf("sen bóveda") }
	}
	if ps := ficha.Validar(f); len(ps) > 0 {
		for _, p := range ps {
			inf.Fallos = append(inf.Fallos, Fallo{Campo: p.Campo, Mensaxe: p.Mensaxe})
		}
		return inf, nil
	}
	dir, err := os.MkdirTemp("", "luayang-probas-*")
	if err != nil {
		return inf, err
	}
	defer os.RemoveAll(dir)
	x := &xerador{o: o, fichas: fichas, dir: dir, reg: latex.NovoRawRegistry()}
	if PrecisaMaxima(f) {
		ruta := RutaMaxima(o.Maxima)
		if ruta == "" {
			return inf, fmt.Errorf("esta ficha precisa Maxima e non está instalado")
		}
		sess, err := cas.Open(ruta)
		if err != nil {
			return inf, err
		}
		x.m = cas.NewMaxima(sess)
		defer x.m.Close()
		x.m.SetOutDir(dir)
		x.m.Format = cas.FormatLatex
		x.m.RenderMode = "proba"
		if _, err := x.m.Send("linel:1024;display2d:false;simp:true;", false); err != nil {
			return inf, err
		}
	}
	condicions := []string{}
	if f.Probas != nil {
		condicions = f.Probas.Condicions
	}
	distintas := map[string]bool{}
	for s := 1; s <= n; s++ {
		if err := x.iniciarVariante(s); err != nil {
			return inf, err
		}
		var vals map[string]Valor
		var err error
		if f.FormatoCorpo() == ficha.FormatoMatex {
			vals, err = x.probarMatex(f, s, &inf)
		} else {
			vals, err = x.valoresMarkdown(f)
		}
		if err != nil {
			inf.Fallos = append(inf.Fallos, Fallo{Semente: s, Mensaxe: err.Error()})
			continue
		}
		fila := map[string]string{}
		var chave []string
		for _, k := range nomesOrdenados(vals) {
			v := vals[k]
			t := v.Texto
			if len(v.Filas) > 0 {
				var ids []string
				for _, fl := range v.Filas {
					ids = append(ids, primeiroValor(fl))
				}
				t = strings.Join(ids, " · ")
			}
			fila[k] = t
			chave = append(chave, k+"="+t)
			if v.EsNum && (math.IsNaN(v.Num) || math.IsInf(v.Num, 0)) {
				inf.Fallos = append(inf.Fallos, Fallo{Semente: s, Campo: k, Mensaxe: "non é un número finito"})
			} else if malRe.MatchString(v.Tex) || (!v.EsNum && len(v.Filas) == 0 && malRe.MatchString(v.Texto)) {
				inf.Fallos = append(inf.Fallos, Fallo{Semente: s, Campo: k, Mensaxe: "dá " + recortar(v.Texto, 80)})
			}
		}
		distintas[strings.Join(chave, "|")] = true
		if len(inf.Mostra) < 6 {
			inf.Mostra = append(inf.Mostra, fila)
		}
		for _, c := range condicions {
			ok, err := x.condicion(c, vals)
			if err != nil {
				inf.Fallos = append(inf.Fallos, Fallo{Semente: s, Campo: "condición", Mensaxe: fmt.Sprintf("%s: %v", c, err)})
			} else if !ok {
				inf.Fallos = append(inf.Fallos, Fallo{Semente: s, Campo: "condición", Mensaxe: "non se cumpre " + c})
			}
		}
	}
	inf.Distintas = len(distintas)
	for _, c := range f.Campos {
		inf.Campos = append(inf.Campos, ficha.NomeBase(c.Nome))
	}
	sort.Strings(inf.Campos)
	if n >= 5 && inf.Distintas*2 < n && len(f.Campos) > 0 {
		inf.Avisos = append(inf.Avisos, fmt.Sprintf("só %d variantes distintas en %d sementes: amplía os rangos ou as listas", inf.Distintas, n))
	}
	return inf, nil
}

func primeiroValor(f map[string]string) string {
	var ks []string
	for k := range f {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	if len(ks) == 0 {
		return ""
	}
	return recortar(f[ks[0]], 30)
}

func recortar(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}

// condicion avalía unha condición (Maxima se hai sesión, se non fórmula).
func (x *xerador) condicion(c string, vals map[string]Valor) (bool, error) {
	if x.m != nil {
		out, err := x.m.Send("is("+c+")", true)
		if err != nil {
			return false, err
		}
		switch ultimoValor(out) {
		case "true":
			return true, nil
		case "false":
			return false, nil
		default:
			return false, fmt.Errorf("non se puido decidir (%s)", ultimoValor(out))
		}
	}
	v, err := numeroFormula(c, vals)
	return v != 0, err
}

// probarMatex: unha ficha de Yang avalíase enteira en modo «proba» (con
// resultados e solucións) e búscanse os erros que marca cas.
func (x *xerador) probarMatex(f *ficha.Ficha, semente int, inf *InformeProbas) (map[string]Valor, error) {
	fixos, err := valoresGo(f, x.rng, x.bancos, x.o.Datos, false)
	if err != nil {
		return nil, err
	}
	x.m.ProtectRaw = nil
	html, err := x.m.ParseText(textoMatex(f, fixos))
	if err != nil {
		return nil, err
	}
	if strings.Contains(html, cas.MarcaErroInline) || malRe.MatchString(html) {
		trozo := html
		if i := malRe.FindStringIndex(html); i != nil {
			trozo = html[max(0, i[0]-60):min(len(html), i[1]+60)]
		}
		inf.Fallos = append(inf.Fallos, Fallo{Semente: semente, Campo: "texto", Mensaxe: "erro ou valor non válido: …" + strings.TrimSpace(trozo) + "…"})
	}
	vals := map[string]Valor{}
	for _, c := range f.Campos {
		if c.Funcion || strings.Contains(c.Nome, "(") || c.Orixe == ficha.OrixeTaboa || c.Orixe == ficha.OrixeEscrito {
			continue
		}
		n := ficha.NomeBase(c.Nome)
		if x.senValor(n) {
			inf.Fallos = append(inf.Fallos, Fallo{Semente: semente, Campo: n, Mensaxe: "non se puido calcular (quedou sen valor: división por cero ou erro de Maxima?)"})
			continue
		}
		tex, err := x.tex(n, true)
		if err != nil {
			inf.Fallos = append(inf.Fallos, Fallo{Semente: semente, Campo: n, Mensaxe: err.Error()})
			continue
		}
		v := Valor{Tex: tex, Texto: tex, Mat: true}
		if num, err := x.numero(n); err == nil {
			v.Num, v.EsNum = num, true
		}
		vals[n] = v
	}
	for k, v := range fixos {
		vals[k] = v
	}
	return vals, nil
}

// senValor: a variable de Maxima non ten valor (vale o seu propio nome).
func (x *xerador) senValor(nome string) bool {
	out, err := x.m.Send("string("+nome+")", true)
	if err != nil {
		return true
	}
	return strings.Trim(ultimoValor(out), `"`) == nome
}
