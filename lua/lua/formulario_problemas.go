package main

// Problemas: o que Lua colle de Yang. Un campo «problemas» dun formulario é
// unha lista de preguntas (matemáticas, física, química, tecnoloxía...) que
// se enchen coma o resto dos datos: un enunciado en Markdown con fórmulas
// $LaTeX$, unha imaxe, os puntos e as VARIABLES que aparecen no enunciado
// ({a}, {m}, {v0}...), cada unha fixa ou variable (aleatoria nun rango).
//
// A cada pregunta corresponde unha SOLUCIÓN, que se enche só cando se pide
// (á man ou coa IA, ver problemas_ia.go). A solución non leva números
// copiados: leva os mesmos {…} ca o enunciado e uns CÁLCULOS en Maxima
// (ou fórmulas simples sen Maxima). Así:
//
//   - cambiar unha variable de fixa a variable (ou a semente) recalcula a
//     solución ao instante, sen volver chamar á IA;
//   - cada variante (as «iteracións» de Yang, unha semente por copia) sae
//     coa súa solución correcta;
//   - Maxima comproba a solución da IA en varias sementes antes de aceptala.
//
// O cálculo vai no motor común de Yang (edicion/xerar.Motor). No documento,
// cada campo de problemas é un bloque bloqueProblemas que expandirProblemas
// converte nos bloques de sempre (Markdown con fórmulas, imaxes, solucións)
// xusto antes de escribir o PDF, o Markdown, o Word ou o OpenDocument.

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"edicion/xerar"
)

// campoProblemas é o tipo do campo.
const campoProblemas = "problemas"

// Tipos de bloque dos problemas.
const (
	bloqueProblemas = "problemas" // sen expandir (só existe ata expandirProblemas)
	bloqueSolucion  = "solucion"  // a solución dunha pregunta: Texto é Markdown
)

// Onde van as solucións no documento.
const (
	solucionsNon   = "non"   // só os enunciados (o exame para o alumnado)
	solucionsXunto = "xunto" // debaixo de cada pregunta
	solucionsFinal = "final" // nun solucionario ao final
)

// maxVariantes é o tope de copias dun documento (o mesmo ca Yang).
const maxVariantes = 50

// SolucionProblema é a solución dunha pregunta.
type SolucionProblema struct {
	// Texto e Resultado: Markdown + $LaTeX$ + {…} (as regras de Yang).
	Texto     string `json:"texto,omitempty"`
	Resultado string `json:"resultado,omitempty"`
	// Calculos: valores auxiliares (Maxima ou fórmula) que usa a solución.
	Calculos []xerar.CalculoProblema `json:"calculos,omitempty"`
	// Comprobacions: condicións que Maxima verifica en cada semente.
	Comprobacions []string `json:"comprobacions,omitempty"`
	// Orixe: quen a fixo («ia», «ia+maxima» ou «man»).
	Orixe string `json:"orixe,omitempty"`
	// Parametrica: a solución depende das variables (con {…}). Se non o é,
	// só vale para os valores con que se resolveu (Valores).
	Parametrica bool              `json:"parametrica,omitempty"`
	Valores     map[string]string `json:"valores,omitempty"`
}

// ProblemaFormulario é unha pregunta.
type ProblemaFormulario struct {
	ID        string                   `json:"id,omitempty"`
	Enunciado string                   `json:"enunciado"`
	Imaxe     string                   `json:"imaxe,omitempty"` // data URI
	Puntos    string                   `json:"puntos,omitempty"`
	Variables []xerar.VariableProblema `json:"variables,omitempty"`
	// Calculos: valores que xa usa o enunciado (p. ex. un polinomio feito
	// coas raíces {r1}, {r2}).
	Calculos []xerar.CalculoProblema `json:"calculos,omitempty"`
	Solucion *SolucionProblema       `json:"solucion,omitempty"`
}

// motor é o problema tal e como o calcula o motor común.
func (p ProblemaFormulario) motor() xerar.Problema {
	out := xerar.Problema{Enunciado: p.Enunciado, Variables: p.Variables, Calculos: append([]xerar.CalculoProblema(nil), p.Calculos...)}
	if s := p.Solucion; s != nil {
		out.Solucion, out.Resultado = s.Texto, s.Resultado
		out.Calculos = append(out.Calculos, s.Calculos...)
		out.Comprobacions = s.Comprobacions
	}
	return out
}

// XeracionProblemas son as opcións de xeración dun documento con problemas
// (a barra de Yang: semente e iteracións, e que solucións saen).
type XeracionProblemas struct {
	Semente   int    `json:"semente"`
	Variantes int    `json:"variantes,omitempty"` // as «iteracións»: 0 ou 1 = unha copia
	Solucions string `json:"solucions,omitempty"` // solucionsNon (por defecto), solucionsXunto ou solucionsFinal
}

func (x *XeracionProblemas) normalizar() XeracionProblemas {
	out := XeracionProblemas{Variantes: 1, Solucions: solucionsNon}
	if x == nil {
		return out
	}
	out.Semente = max(x.Semente, 0)
	out.Variantes = min(max(x.Variantes, 1), maxVariantes)
	switch x.Solucions {
	case solucionsXunto, solucionsFinal:
		out.Solucions = x.Solucions
	}
	return out
}

// problemasDe le o valor dun campo problemas (veña coma veña do JSON).
func problemasDe(v any) []ProblemaFormulario {
	switch x := v.(type) {
	case nil:
		return nil
	case []ProblemaFormulario:
		return x
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var out []ProblemaFormulario
	if json.Unmarshal(b, &out) != nil {
		return nil
	}
	return out
}

// problemasNonVacios son as preguntas con algo escrito.
func problemasNonVacios(v any) []ProblemaFormulario {
	var out []ProblemaFormulario
	for _, p := range problemasDe(v) {
		if strings.TrimSpace(p.Enunciado) != "" || p.Imaxe != "" {
			out = append(out, p)
		}
	}
	return out
}

// temProblemas di se o formulario ten algún campo de problemas.
func (f Formulario) temProblemas() bool {
	for _, c := range f.Campos {
		if c.Tipo == campoProblemas {
			return true
		}
	}
	return false
}

// desprazamentosProblemas: para cada campo de problemas, cantas preguntas
// hai antes del (na orde dos campos). Fai que cada pregunta teña o seu
// índice no documento e, con el, a súa semente (xerar.SementeProblema), a
// mesma no editor e no documento.
func desprazamentosProblemas(f Formulario, d DatosFormulario) map[string]int {
	out := map[string]int{}
	n := 0
	for _, c := range f.Campos {
		if c.Tipo == campoProblemas {
			out[c.ID] = n
			n += len(problemasDe(d.Valores[c.ID]))
		}
	}
	return out
}

// ---------- Avisos ----------

var chaveVariableRe = regexp.MustCompile(`\{\s*([A-Za-z_][A-Za-z0-9_]*)\s*\}`)

// avisosProblemas revisa as preguntas: variables sen declarar, rangos
// baleiros, solucións que xa non valen para os valores de agora.
func avisosProblemas(c CampoFormulario, v any) []string {
	var avisos []string
	for i, p := range problemasDe(v) {
		onde := fmt.Sprintf("«%s», pregunta %d", c.Etiqueta, i+1)
		declaradas := map[string]bool{}
		for _, va := range p.Variables {
			declaradas[strings.TrimSpace(va.Nome)] = true
			if va.Modo == xerar.VariableAleatoria {
				de, e1 := strconv.ParseFloat(strings.Replace(strings.TrimSpace(va.De), ",", ".", 1), 64)
				ata, e2 := strconv.ParseFloat(strings.Replace(strings.TrimSpace(va.Ata), ",", ".", 1), 64)
				if e1 == nil && e2 == nil && ata < de {
					avisos = append(avisos, fmt.Sprintf("%s: a variable %s ten un rango baleiro (de %s ata %s)", onde, va.Nome, va.De, va.Ata))
				}
			}
		}
		for _, cl := range p.Calculos {
			declaradas[nomeCalculo(cl.Nome)] = true
		}
		if p.Solucion != nil {
			for _, cl := range p.Solucion.Calculos {
				declaradas[nomeCalculo(cl.Nome)] = true
			}
		}
		for _, nome := range variablesEnFormulas(p.Enunciado) {
			if !declaradas[nome] && !unidadesComuns[nome] {
				avisos = append(avisos, fmt.Sprintf("%s: {%s} está nunha fórmula pero non é ningunha variable", onde, nome))
				declaradas[nome] = true
			}
		}
	}
	return avisos
}

func nomeCalculo(n string) string {
	n = strings.TrimSpace(n)
	if i := strings.Index(n, "("); i >= 0 {
		n = n[:i]
	}
	return strings.TrimSpace(n)
}

// unidadesComuns: unidades que poden ir entre chaves nunha fórmula ($50\,{mA}$)
// e que non son variables sen declarar.
var unidadesComuns = func() map[string]bool {
	m := map[string]bool{}
	// Só as de dúas letras ou máis: as dunha (m, g, V, A...) adoitan ser variables.
	for _, u := range strings.Fields(`mA kA uA mV kV mW kW MW Wh kWh kJ MJ cal kcal Hz kHz MHz GHz
		mF uF nF pF mH mC uC ohm Ohm kohm Mohm ms min cm mm km kg mg kN Pa kPa hPa MPa atm bar
		mL ml mol rpm dB lx cd`) {
		m[u] = true
	}
	return m
}()

// variablesEnFormulas: os {nome} das fórmulas que son variables. Unhas chaves
// pegadas a un comando (\mathrm{mA}, \text{kg}), a «^», a «_» ou a outras
// chaves son LaTeX (a mesma regra ca o motor e a vista).
func variablesEnFormulas(md string) []string {
	var out []string
	for _, f := range matematicasRe.FindAllString(md, -1) {
		for _, loc := range chaveVariableRe.FindAllStringSubmatchIndex(f, -1) {
			if loc[0] > 0 && strings.ContainsRune("^_}", rune(f[loc[0]-1])) || loc[0] > 0 && unicode.IsLetter(rune(f[loc[0]-1])) {
				continue
			}
			out = append(out, f[loc[2]:loc[3]])
		}
	}
	return out
}

var matematicasRe = regexp.MustCompile(`(?s)\$\$.+?\$\$|\$[^$\n]+?\$`)

// soMatematicas: só o que hai entre $…$ (fóra, un {x} pode ser unha
// expresión de Maxima e non unha variable).
func soMatematicas(s string) string {
	return strings.Join(matematicasRe.FindAllString(s, -1), " ")
}

// ---------- Documento ----------

// motorProblemasDoc: o que precisa a expansión (o *xerar.Motor ou un dobre
// nas probas).
type motorProblemasDoc interface {
	Calcular(p xerar.Problema, semente int) xerar.ProblemaCalculado
}

// expandirProblemas converte os bloques de problemas nos bloques de sempre.
// Con varias variantes, o documento enteiro repítese unha vez por semente
// (coma as iteracións de Yang), separado por saltos de páxina. Sen bloques
// de problemas devolve os bloques tal cal.
func expandirProblemas(bloques []bloqueDoc, xer *XeracionProblemas, m motorProblemasDoc, tx textosDocumento) ([]bloqueDoc, []string) {
	haiProblemas := false
	for _, b := range bloques {
		if b.Tipo == bloqueProblemas {
			haiProblemas = true
			break
		}
	}
	if !haiProblemas {
		return bloques, nil
	}
	x := xer.normalizar()
	var avisos []string
	avisar := func(s string) { avisos = engadirSenRepetir(avisos, []string{s}) }
	var out, solucionario []bloqueDoc
	for v := 0; v < x.Variantes; v++ {
		semente := x.Semente + v
		if v > 0 {
			out = append(out, bloqueDoc{Tipo: bloqueSalto})
		}
		if x.Variantes > 1 {
			out = append(out, bloqueDoc{Tipo: bloqueTitulo, Nivel: 2, Texto: tx.T("FORM_VARIANTE", map[string]string{"N": strconv.Itoa(v + 1), "SEMENTE": strconv.Itoa(semente)})})
			if x.Solucions == solucionsFinal {
				solucionario = append(solucionario, bloqueDoc{Tipo: bloqueTitulo, Nivel: 3, Texto: tx.T("FORM_VARIANTE", map[string]string{"N": strconv.Itoa(v + 1), "SEMENTE": strconv.Itoa(semente)})})
			}
		}
		numero := 0
		for _, b := range bloques {
			if b.Tipo != bloqueProblemas {
				out = append(out, b)
				continue
			}
			for i, p := range b.Problemas {
				if strings.TrimSpace(p.Enunciado) == "" && p.Imaxe == "" {
					continue
				}
				numero++
				num := strconv.Itoa(numero)
				c := m.Calcular(p.motor(), xerar.SementeProblema(semente, b.Desprazamento+i))
				if c.Erro != "" {
					avisar(fmt.Sprintf("pregunta %s (semente %d): %s", num, semente, c.Erro))
				}
				for _, a := range append(c.Avisos, c.Fallos...) {
					avisar(fmt.Sprintf("pregunta %s: %s", num, a))
				}
				cab := "**" + num + ".**"
				if pt := strings.TrimSpace(p.Puntos); pt != "" {
					cab += " *(" + tx.T("FORM_PUNTOS", map[string]string{"N": pt}) + ")*"
				}
				out = append(out, bloqueDoc{Tipo: bloqueMarkdown, Texto: cab + " " + strings.TrimSpace(c.Enunciado)})
				if p.Imaxe != "" {
					out = append(out, bloqueDoc{Tipo: bloqueImaxe, Imaxe: p.Imaxe})
				}
				sol := textoSolucion(c, tx)
				if sol != "" && p.Solucion != nil && !p.Solucion.Parametrica && len(p.Solucion.Valores) > 0 && !mesmosValores(p.Solucion.Valores, c.Valores) {
					avisar(fmt.Sprintf("pregunta %s: a solución resolveuse con outros valores das variables; resólvea de novo ou fai fixas as variables", num))
				}
				switch {
				case x.Solucions == solucionsNon:
				case sol == "":
					avisar(fmt.Sprintf("pregunta %s: aínda non ten solución", num))
				case x.Solucions == solucionsXunto:
					out = append(out, bloqueDoc{Tipo: bloqueSolucion, Rotulo: tx.T("FORM_SOLUCION"), Texto: sol})
				default:
					solucionario = append(solucionario, bloqueDoc{Tipo: bloqueSolucion, Rotulo: num + ".", Texto: sol})
				}
			}
		}
	}
	if len(solucionario) > 0 {
		out = append(out, bloqueDoc{Tipo: bloqueSalto}, bloqueDoc{Tipo: bloqueTitulo, Nivel: 2, Texto: tx.T("FORM_SOLUCIONARIO")})
		out = append(out, solucionario...)
	}
	return out, avisos
}

// textoSolucion: o Markdown da solución calculada (resolución e resultado).
func textoSolucion(c xerar.ProblemaCalculado, tx textosDocumento) string {
	sol := strings.TrimSpace(c.Solucion)
	if r := strings.TrimSpace(c.Resultado); r != "" {
		if sol != "" {
			sol += "\n\n"
		}
		sol += "**" + tx.T("FORM_RESULTADO") + ":** " + r
	}
	return sol
}

// mesmosValores compara os valores cos que se resolveu cos de agora.
func mesmosValores(antes map[string]string, agora map[string]xerar.ValorProblema) bool {
	for k, v := range antes {
		if a, ok := agora[k]; ok && strings.TrimSpace(a.Texto) != strings.TrimSpace(v) {
			return false
		}
	}
	return true
}

// bloquesDocumentoFormulario: os bloques do documento cos problemas xa
// expandidos (o que escriben LaTeX, Markdown, Word e OpenDocument).
func bloquesDocumentoFormulario(p PedidoFormulario, f Formulario, d DatosFormulario, tx textosDocumento) ([]bloqueDoc, []string) {
	bloques, avisos := bloquesFormulario(f, d, tx)
	m := p.motor
	if m == nil {
		gm, _ := xerar.NovoMotor(xerar.Opcions{}, false, 0)
		m = gm
	}
	exp, av := expandirProblemas(bloques, d.Xeracion, m, tx)
	return exp, engadirSenRepetir(avisos, av)
}

// conMotorProblemas pon o motor de Maxima na petición se o formulario ten
// problemas (copia o pedido: o do chamador non se toca).
func (a *App) conMotorProblemas(req XerarInformeCronogramaRequest) XerarInformeCronogramaRequest {
	if req.eFormulario() && req.Formulario.motor == nil && req.Formulario.Formulario.temProblemas() {
		p := *req.Formulario
		p.motor, _ = a.motorProblemas()
		req.Formulario = &p
	}
	return req
}

// ---------- Bindings ----------

// PedidoCalcularProblemas: as preguntas que o editor quere ver calculadas.
type PedidoCalcularProblemas struct {
	Semente   int                    `json:"semente"`
	Problemas []ProblemaParaCalcular `json:"problemas"`
}

// ProblemaParaCalcular: unha pregunta e o seu índice no documento.
type ProblemaParaCalcular struct {
	Indice   int                `json:"indice"`
	Problema ProblemaFormulario `json:"problema"`
}

// ResultadoCalcularProblemas: o calculado, na mesma orde.
type ResultadoCalcularProblemas struct {
	Maxima     bool                      `json:"maxima"`
	Aviso      string                    `json:"aviso,omitempty"`
	Calculados []xerar.ProblemaCalculado `json:"calculados"`
	// Debuxos: os circuítos e debuxos TikZ dos textos calculados, xa
	// compilados (clave: tipo + "\n" + código, ver claveVistaDebuxo).
	Debuxos map[string]DebuxoVista `json:"debuxos"`
}

// CalcularProblemas calcula as preguntas na semente (a primeira variante):
// é o que ve o editor mentres se escribe, e o que se recalcula ao cambiar
// unha variable de fixa a variable ou a semente.
func (a *App) CalcularProblemas(p PedidoCalcularProblemas) ResultadoCalcularProblemas {
	m, aviso := a.motorProblemas()
	out := a.calcularCon(m, p)
	out.Aviso = aviso
	return out
}

func (a *App) calcularCon(m *xerar.Motor, p PedidoCalcularProblemas) ResultadoCalcularProblemas {
	out := ResultadoCalcularProblemas{Maxima: m.TenMaxima(), Calculados: []xerar.ProblemaCalculado{}}
	for _, pr := range p.Problemas {
		out.Calculados = append(out.Calculados, m.Calcular(pr.Problema.motor(), xerar.SementeProblema(max(p.Semente, 0), pr.Indice)))
	}
	out.Debuxos = debuxosVista(out.Calculados...)
	return out
}

// claveVistaDebuxo: a clave co que o frontend busca un debuxo (o mesmo
// texto que ve: tipo, salto de liña e o código sen espazos nas beiras).
func claveVistaDebuxo(tipo, codigo string) string { return tipo + "\n" + strings.TrimSpace(codigo) }

// debuxosVista compila os debuxos dos problemas calculados para a vista.
func debuxosVista(cs ...xerar.ProblemaCalculado) map[string]DebuxoVista {
	var ds [][2]string
	for _, c := range cs {
		for _, t := range []string{c.Enunciado, c.Solucion, c.Resultado} {
			ds = append(ds, debuxosDe(t)...)
		}
	}
	out := map[string]DebuxoVista{}
	if len(ds) == 0 {
		return out
	}
	porClave := debuxarParaVista(ds)
	for _, d := range ds {
		out[claveVistaDebuxo(d[0], d[1])] = porClave[claveDebuxo(d[0], d[1])]
	}
	return out
}
