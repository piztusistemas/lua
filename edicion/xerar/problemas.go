package xerar

// Problemas: o motor de Yang servido peza a peza, para as aplicacións que
// non traballan con fichas da bóveda senón cos seus propios formularios (Lua).
// Un problema é un enunciado e unha solución en Markdown + $LaTeX$, coas
// regras dos {…} de Yang (ver markdown.go), unhas variables (fixas ou
// aleatorias) e uns cálculos auxiliares (o <HIDE> de Yang).
//
// Un Motor ten unha sesión de Maxima aberta (se a hai) e calcula cada
// problema nunha semente: devolve o Markdown xa substituído (cos valores e as
// expresións en $…$) que a aplicación pinta ou converte ao seu documento. Sen
// Maxima calcula en Go o que pode (números, rangos e fórmulas simples) e avisa
// do resto.

import (
	"edicion/idioma"
	"fmt"
	"strings"
	"sync"
	"time"

	"edicion/cas"
	"edicion/ficha"
	"edicion/latex"
)

// Modos dunha variable dun problema.
const (
	VariableFixa      = "fixa"
	VariableAleatoria = "variable"
)

// VariableProblema é un dato do enunciado.
type VariableProblema struct {
	Nome  string `json:"nome"`
	Modo  string `json:"modo"`            // VariableFixa ou VariableAleatoria
	Valor string `json:"valor,omitempty"` // a fixa
	De    string `json:"de,omitempty"`    // a aleatoria: rango [De, Ata] con Paso
	Ata   string `json:"ata,omitempty"`
	Paso  string `json:"paso,omitempty"`
}

// CalculoProblema é un valor auxiliar (Maxima ou fórmula) que se pode usar
// coma {nome} no enunciado ou na solución. Nome pode ser unha función
// «f(x)», que se define con «:=».
type CalculoProblema struct {
	Nome string `json:"nome"`
	Expr string `json:"expr"`
}

// Problema é o que a aplicación lle pasa ao motor.
type Problema struct {
	Enunciado string             `json:"enunciado"`
	Solucion  string             `json:"solucion,omitempty"`
	Resultado string             `json:"resultado,omitempty"`
	Variables []VariableProblema `json:"variables,omitempty"`
	Calculos  []CalculoProblema  `json:"calculos,omitempty"`
	// Comprobacions: condicións que teñen que cumprirse en toda semente (en
	// Maxima, «is(...)»; sen Maxima, unha fórmula distinta de 0). Por
	// exemplo, que a solución verifique a ecuación.
	Comprobacions []string `json:"comprobacions,omitempty"`
}

// ValorProblema é o valor dunha variable ou dun cálculo nunha semente.
type ValorProblema struct {
	Texto string `json:"texto"` // para amosar (coma decimal)
	Tex   string `json:"tex"`   // para dentro de $…$
}

// ProblemaCalculado é o problema nunha semente: o Markdown xa substituído.
type ProblemaCalculado struct {
	Enunciado string                   `json:"enunciado"`
	Solucion  string                   `json:"solucion"`
	Resultado string                   `json:"resultado"`
	Valores   map[string]ValorProblema `json:"valores"`
	Maxima    bool                     `json:"maxima"` // calculouse con Maxima
	Avisos    []string                 `json:"avisos,omitempty"`
	Erro      string                   `json:"erro,omitempty"` // non se puido calcular (os textos van sen substituír)
	// Fallos: comprobacións que non se cumpren e valores que dan infinito,
	// indeterminación ou erro.
	Fallos []string `json:"fallos,omitempty"`
}

// Motor calcula problemas. É seguro usalo desde varias gorrutinas (unha
// sesión de Maxima só atende unha cousa de cada vez).
type Motor struct {
	mu sync.Mutex
	x  *xerador
}

// NovoMotor abre Maxima se `o.Maxima` (ou o do PATH) existe e `conMaxima`;
// se non, o motor calcula só en Go.
func NovoMotor(o Opcions, conMaxima bool, timeout time.Duration) (*Motor, error) {
	x := &xerador{o: o, reg: latex.NovoRawRegistry()}
	if conMaxima {
		if ruta := RutaMaxima(o.Maxima); ruta != "" {
			sess, err := cas.Open(ruta)
			if err != nil {
				return nil, idioma.Errorf("go.maxima.non_inicia", "non se puido iniciar Maxima: %w", err)
			}
			x.m = cas.NewMaxima(sess)
			if timeout > 0 {
				x.m.SetTimeout(timeout)
			}
			x.m.Format = cas.FormatLatex
			x.m.ForzarDecimal = o.ForzarDecimal
			x.m.RenderMode = "solucions"
			if _, err := x.m.Send("linel:1024;display2d:false;simp:true;", false); err != nil {
				x.m.Close()
				return nil, err
			}
		}
	}
	return &Motor{x: x}, nil
}

// TenMaxima di se o motor calcula con Maxima.
func (m *Motor) TenMaxima() bool { return m != nil && m.x.m != nil }

// Close pecha Maxima.
func (m *Motor) Close() {
	if m == nil || m.x.m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.x.m.Close()
	m.x.m = nil
}

// SementeProblema é a semente do problema `indice` (desde 0) na variante que
// empeza en `semente`: cada problema leva a súa, así dous problemas coas
// mesmas variables non saen cos mesmos números.
func SementeProblema(semente, indice int) int { return semente*1000 + indice }

// fichaDe converte o problema nunha ficha de exercicio (o modelo común).
func fichaDe(p Problema, conMaxima bool) *ficha.Ficha {
	f := &ficha.Ficha{Luayang: ficha.Version, Tipo: ficha.TipoExercicio, Formato: ficha.FormatoMarkdown}
	motor := ficha.MotorFormula
	if conMaxima {
		motor = ficha.MotorMaxima
	}
	for _, v := range p.Variables {
		nome := strings.TrimSpace(v.Nome)
		if nome == "" {
			continue
		}
		if v.Modo == VariableAleatoria {
			f.Campos = append(f.Campos, ficha.Campo{Nome: nome, Orixe: ficha.OrixeAleatorio, Xerador: ficha.XeradorRango,
				De: strings.TrimSpace(v.De), Ata: strings.TrimSpace(v.Ata), Paso: strings.TrimSpace(v.Paso)})
			continue
		}
		f.Campos = append(f.Campos, ficha.Campo{Nome: nome, Orixe: ficha.OrixeFixo, Valor: numeroMaxima(v.Valor)})
	}
	for _, c := range p.Calculos {
		nome := strings.TrimSpace(c.Nome)
		if nome == "" || strings.TrimSpace(c.Expr) == "" {
			continue
		}
		f.Campos = append(f.Campos, ficha.Campo{Nome: nome, Orixe: ficha.OrixeCalculado, Motor: motor, Expr: strings.TrimSpace(c.Expr),
			Funcion: strings.Contains(nome, "(")})
	}
	return f
}

// numeroMaxima: «2,5» -> «2.5» (a coma decimal galega non é Maxima).
func numeroMaxima(s string) string {
	s = strings.TrimSpace(s)
	if strings.Count(s, ",") == 1 && !strings.ContainsAny(s, "[]()") {
		if t := strings.Replace(s, ",", ".", 1); valorTexto(t).EsNum {
			return t
		}
	}
	return s
}

// Calcular resolve o problema na semente dada. Non devolve erro: se algo
// falla, vai en Erro e os textos saen sen substituír, para que a aplicación
// os poida amosar igual.
func (m *Motor) Calcular(p Problema, semente int) ProblemaCalculado {
	m.mu.Lock()
	defer m.mu.Unlock()
	x := m.x
	x.avisos = nil
	out := ProblemaCalculado{Enunciado: p.Enunciado, Solucion: p.Solucion, Resultado: p.Resultado, Valores: map[string]ValorProblema{}, Maxima: x.m != nil}
	f := fichaDe(p, x.m != nil)
	for _, pr := range ficha.Validar(f) {
		out.Avisos = append(out.Avisos, pr.String())
	}
	if err := x.iniciarVariante(semente); err != nil {
		out.Erro = err.Error()
		m.reiniciarSeCaeu(err)
		return out
	}
	vals, err := x.valoresProblema(f)
	if err != nil {
		out.Erro = err.Error()
		m.reiniciarSeCaeu(err)
		return out
	}
	for k, v := range vals {
		if v.Filas != nil {
			continue
		}
		vp := ValorProblema{Texto: v.Texto, Tex: v.Tex}
		if v.EsNum {
			vp.Texto, vp.Tex = formatarNumeroDecimais(v.Num, x.o.Decimais)
		}
		if vp.Tex == "" {
			vp.Tex = `\text{` + latex.EscapeLatex(v.Texto) + `}`
		}
		out.Valores[k] = vp
	}
	cx := &contextoMD{vals: vals, reg: x.reg, avisos: &x.avisos, valadosCrus: true, decimais: x.o.Decimais}
	if x.m != nil {
		cx.maxima = x.tex
	}
	out.Enunciado = cx.substituir(p.Enunciado)
	out.Solucion = cx.substituir(p.Solucion)
	out.Resultado = cx.substituir(p.Resultado)
	out.Avisos = append(out.Avisos, x.avisos...)
	for _, k := range nomesOrdenados(vals) {
		if malRe.MatchString(vals[k].Tex + " " + vals[k].Texto) {
			out.Fallos = append(out.Fallos, fmt.Sprintf("%s vale %s", k, vals[k].Texto))
		}
	}
	for _, c := range p.Comprobacions {
		if c = strings.TrimSpace(c); c != "" {
			if f := x.comprobar(c, vals); f != "" {
				out.Fallos = append(out.Fallos, f)
			}
		}
	}
	return out
}

// comprobar avalía unha condición; devolve "" se se cumpre ou o motivo.
func (x *xerador) comprobar(c string, vals map[string]Valor) string {
	if x.m == nil {
		n, err := numeroFormula(c, vals)
		if err != nil {
			return fmt.Sprintf("non se puido comprobar «%s»: %v", c, err)
		}
		if n == 0 {
			return idioma.T("go.problemas.condicion", "non se cumpre «%s»", c)
		}
		return ""
	}
	out, err := x.m.Send("is("+c+")", true)
	if err != nil {
		return fmt.Sprintf("non se puido comprobar «%s»: %v", c, err)
	}
	if r := ultimoValor(out); r != "true" {
		return idioma.T("go.problemas.condicion_maxima", "non se cumpre «%s» (Maxima di %s)", c, r)
	}
	return ""
}

// Probar calcula o problema nas sementes dadas e devolve os problemas
// atopados (erros, avisos e fallos), cada un coa súa semente. Baleiro = ben.
func (m *Motor) Probar(p Problema, sementes []int) []string {
	var out []string
	vistos := map[string]bool{}
	engadir := func(sem int, msg string) {
		if !vistos[msg] {
			vistos[msg] = true
			out = append(out, fmt.Sprintf("semente %d: %s", sem, msg))
		}
	}
	for _, s := range sementes {
		c := m.Calcular(p, s)
		if c.Erro != "" {
			engadir(s, c.Erro)
			continue
		}
		for _, a := range c.Avisos {
			engadir(s, a)
		}
		for _, f := range c.Fallos {
			engadir(s, f)
		}
	}
	return out
}

// valoresProblema: coma valoresMarkdown, pero un problema que non precisa
// Maxima calcúlase sempre en Go (así a mesma semente dá os mesmos números
// con e sen Maxima instalado).
func (x *xerador) valoresProblema(f *ficha.Ficha) (map[string]Valor, error) {
	if x.m == nil || !PrecisaMaxima(f) {
		return x.conDecimais(valoresGo(f, x.rng, nil, nil, true))
	}
	return x.valoresMarkdown(f)
}

// reiniciarSeCaeu: se Maxima morreu (tempo esgotado, erro de proceso), o
// motor segue en Go en vez de quedar roto.
func (m *Motor) reiniciarSeCaeu(err error) {
	if m.x.m == nil || err == nil {
		return
	}
	if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "tempo") || strings.Contains(err.Error(), "EOF") || strings.Contains(err.Error(), "broken pipe") {
		m.x.m.Close()
		m.x.m = nil
	}
}
