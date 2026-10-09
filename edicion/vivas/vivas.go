// Package vivas: a IA propón cambios a unha ficha (facela viva con
// variables e solucións calculadas, encher un banco, corrixila…) e cada
// proposta valídase e próbase en varias sementes antes de chegar á persoa.
// Se falla, a IA recibe a mensaxe exacta e ten outra rolda para arranxala
// (o método de correccion_ia.go de Yang).
package vivas

import (
	"fmt"
	"strings"

	"edicion/ficha"
	"edicion/ia"
	"edicion/xerar"
)

// Peticións predefinidas.
const (
	PedidoVivificar = "Fai viva esta ficha: converte os datos numéricos do enunciado en campos aleatorios (sortea a SOLUCIÓN e calcula os datos, para que o resultado sexa sempre bonito), escribe o resultado final como campos calculados con rol resultado e a resolución paso a paso usando os campos. Mantén o texto do enunciado igual salvo os números que pasan a ser {campos}."
	PedidoBanco     = "Engade 10 filas novas ao banco, do mesmo nivel e estilo ca as que xa ten, sen repetir ningunha. Non cambies as columnas."
	PedidoResolver  = "Escribe a sección Resultado e a sección Resolución (paso a paso) deste exercicio. Se hai cálculos, faios con campos calculados de Maxima para que non poidan estar mal."
)

const sistema = `Es o asistente de fichas de LuaYang, un programa para docentes. Unha ficha é un ficheiro Markdown cun front matter YAML.

FORMATO DA FICHA
---
luayang: 1
tipo: exercicio          # exercicio | exame | banco | formulario
titulo: ...
materia: ...
nivel: ...
conceptos: ["[[Concepto]]"]
campos:
  - nome: solA           # identificador (letras sen acentos, díxitos, _)
    orixe: aleatorio     # aleatorio | calculado | fixo | taboa | escrito
    xerador: rango       # rango (de, ata, paso) | lista (valores) | n0 n1 n2 (naturais) z0 z1 z2 (enteiros) q0 q1 q2 (fraccións)
    de: 1
    ata: 6
  - nome: k
    orixe: aleatorio
    xerador: lista
    valores: [1, 2, 3, 4]
  - nome: S
    orixe: calculado
    motor: maxima        # maxima (para matemáticas/ciencias) ou formula (aritmética simple)
    expr: "solA*k^3/6"
  - nome: res_1
    orixe: calculado
    motor: maxima
    expr: "solA"
    rol: resultado       # é o resultado final dun apartado
    etiqueta: "a)"
  - nome: frases         # para letras: filas sorteadas dun banco
    orixe: taboa
    banco: "[[nome-do-banco]]"
    mostra: 6
probas:
  sementes: 25
  condicions: ["S > 0"]  # o que ten que cumprirse en todas as variantes
---

## Enunciado

Texto en Markdown. {nome} pon o valor dun campo; dentro de $...$ úsase igual: $f(x)={f}$. {'expr} é unha expresión de Maxima sen avaliar. Listas dunha táboa: {frases | lista: "{orixinal} (*{pista}*)"}.
Gráficos: un bloque ` + "```plot" + ` coa orde plot2d de Maxima; debuxos: ` + "```tikz" + `.

## Resultado

Só vai no solucionario.

## Resolución

Paso a paso, só no solucionario.

UN BANCO (tipo: banco) leva "columnas: [a, b, c]" e o corpo é unha táboa Markdown con esas columnas.

REGRAS
- Devolve SEMPRE a ficha COMPLETA (front matter e corpo), e nada máis: nin explicacións nin valados de código arredor.
- Non cambies o que non se pide. Conserva id, conceptos, fonte e o resto de metadatos.
- Para matemáticas e ciencias usa motor maxima. Sortea a solución e calcula os datos: así o resultado sempre é exacto e bonito. Evita divisións por cero e raíces negativas (usa condicions).
- Se a ficha ten "formato: matex", o texto está no dialecto de Yang (HTML con <MAT>, <EVAL>...): mantén ese formato.
- O texto do docente é sagrado: non corrixas o seu estilo nin a súa lingua.`

// Proposta da IA.
type Proposta struct {
	Texto  string              `json:"texto"` // a ficha proposta (.md)
	Probas xerar.InformeProbas `json:"probas"`
	Rondas int                 `json:"rondas"`
	Avisos []string            `json:"avisos"`
	Valida bool                `json:"valida"`
}

// Probador proba unha ficha (xerar.Probar coa bóveda).
type Probador func(f *ficha.Ficha) (xerar.InformeProbas, error)

// Editar pídelle á IA o cambio e próbao.
func Editar(c ia.Config, actual *ficha.Ficha, pedido string, probar Probador, aviso func(string)) (Proposta, error) {
	if aviso == nil {
		aviso = func(string) {}
	}
	orixinal, err := ficha.Escribir(actual)
	if err != nil {
		return Proposta{}, err
	}
	msgs := []ia.IAMensaxe{{Rol: "user", Texto: "PETICIÓN: " + pedido + "\n\nFICHA ACTUAL:\n" + string(orixinal)}}
	var p Proposta
	for ronda := 1; ronda <= 3; ronda++ {
		p.Rondas = ronda
		aviso(fmt.Sprintf("Rolda %d: pedindo a proposta á IA…", ronda))
		t, err := ia.Turno(c, sistema, msgs, nil)
		if err != nil {
			return p, err
		}
		msgs = append(msgs, ia.IAMensaxe{Rol: "assistant", Texto: t.Texto, Bruto: t.Bruto, Provedor: t.Provedor})
		texto := limparResposta(t.Texto)
		f, err := ficha.Ler([]byte(texto))
		var problema string
		switch {
		case err != nil:
			problema = "O texto non é unha ficha válida: " + err.Error()
		default:
			if f.ID == "" {
				f.ID = actual.ID
			}
			if ps := ficha.Validar(f); len(ps) > 0 {
				var l []string
				for _, x := range ps {
					l = append(l, x.String())
				}
				problema = "A ficha ten erros:\n- " + strings.Join(l, "\n- ")
			} else if probar != nil {
				aviso("Probando a proposta en varias sementes…")
				inf, err := probar(f)
				p.Probas = inf
				if err != nil {
					problema = "Non se puido probar: " + err.Error()
				} else if !inf.Ben() {
					problema = "As probas fallan:\n" + inf.Resumo()
				}
			}
		}
		if d, err := ficha.Escribir(f); err == nil && f != nil {
			p.Texto = string(d)
		} else {
			p.Texto = texto
		}
		if problema == "" {
			p.Valida = true
			p.Avisos = append(p.Avisos, p.Probas.Avisos...)
			return p, nil
		}
		aviso("A proposta ten problemas; pídese que os arranxe.")
		p.Avisos = append(p.Avisos, fmt.Sprintf("rolda %d: %s", ronda, primeiraLiña(problema)))
		msgs = append(msgs, ia.IAMensaxe{Rol: "user", Texto: problema + "\n\nArránxao e devolve de novo a ficha COMPLETA."})
	}
	return p, nil
}

func primeiraLiña(s string) string {
	l := strings.SplitN(s, "\n", 3)
	if len(l) > 1 {
		return l[0] + " " + l[1]
	}
	return l[0]
}

// limparResposta quita valados de código e texto arredor do front matter.
func limparResposta(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		if i := strings.Index(s, "\n"); i >= 0 {
			s = s[i+1:]
		}
		if j := strings.LastIndex(s, "```"); j >= 0 {
			s = s[:j]
		}
	}
	if i := strings.Index(s, "---\n"); i > 0 {
		s = s[i:]
	}
	return strings.TrimSpace(s) + "\n"
}
