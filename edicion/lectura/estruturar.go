package lectura

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"edicion/ia"
)

// Estrutura dun documento lido: os seus exercicios e metadatos.
type Estrutura struct {
	Titulo      string          `json:"titulo"`
	Materia     string          `json:"materia"`
	Nivel       string          `json:"nivel"`
	Linguas     []string        `json:"linguas"`
	Instrucions string          `json:"instrucions"`
	Exercicios  []ExercicioLido `json:"exercicios"`
	Avisos      []string        `json:"avisos"`
}

// ExercicioLido: un exercicio do documento, co texto literal.
type ExercicioLido struct {
	Titulo    string            `json:"titulo"`
	Apartado  string            `json:"apartado"`
	Puntos    float64           `json:"puntos"`
	Paxina    int               `json:"paxina"`
	Enunciado string            `json:"enunciado"`
	Outras    map[string]string `json:"outras"` // lingua -> enunciado
	Conceptos []string          `json:"conceptos"`
}

// MarkdownCompleto: as páxinas xuntas, con marcas de páxina.
func MarkdownCompleto(pax []PaxinaLida) string {
	var b strings.Builder
	for _, p := range pax {
		fmt.Fprintf(&b, "<!-- páxina %d -->\n\n%s\n\n", p.Num, strings.TrimSpace(p.Markdown))
	}
	return b.String()
}

const sistemaEstrutura = `Recibes a transcrición en Markdown dun exame ou material docente (con marcas <!-- páxina N -->). Tes que separalo en exercicios.

REGRAS:
- Copia o texto de cada exercicio EXACTAMENTE como está (Markdown, LaTeX e ligazóns a imaxes ![...](...) incluídas). Non corrixas, non resumas, non traduzas, non reformules.
- Un exercicio é cada pregunta/problema numerado de primeiro nivel (cos seus apartados dentro). O contexto común dunha pregunta vai co seu enunciado.
- As instrucións xerais do exame (cabeceira, normas) van en "instrucions".
- Se o documento repite o mesmo exame noutra lingua (p.ex. galego e despois castelán), a primeira lingua é a principal: para cada exercicio pon a versión na outra lingua en "outras" co código da lingua ("es", "en", "gl", "pt"...). Non crees exercicios duplicados.
- "conceptos": 1 a 4 conceptos curriculares que traballa o exercicio, en galego, curtos (p.ex. "Distribución normal", "Equilibrio químico").
- "puntos": a puntuación se aparece (número), ou 0.
- "materia" e "nivel" se se deducen (p.ex. "Matemáticas II", "2º Bacharelato").

Responde SÓ cun obxecto JSON:
{"titulo": "...", "materia": "...", "nivel": "...", "linguas": ["gl","es"], "instrucions": "...", "exercicios": [{"titulo": "Pregunta 1. Estatística", "apartado": "1", "puntos": 2, "paxina": 1, "enunciado": "...", "outras": {"es": "..."}, "conceptos": ["..."]}]}`

// Estruturar separa o documento en exercicios coa IA (ou, sen IA, cos
// títulos de pregunta).
func Estruturar(c ia.Config, pax []PaxinaLida, nome string) (Estrutura, error) {
	md := MarkdownCompleto(pax)
	if !c.Configurada() {
		e := EstruturarSenIA(pax, nome)
		e.Avisos = append(e.Avisos, "sen IA: os exercicios separáronse polos títulos «Pregunta N» / «Exercicio N»")
		return e, nil
	}
	txt, err := ia.Preguntar(c, sistemaEstrutura, "Documento: "+nome+"\n\n"+md)
	if err != nil {
		return Estrutura{}, err
	}
	var e Estrutura
	if err := json.Unmarshal([]byte(ia.ExtraerJSON(txt)), &e); err != nil {
		return Estrutura{}, fmt.Errorf("a IA non devolveu JSON válido ao separar os exercicios: %v", err)
	}
	if len(e.Exercicios) == 0 {
		return Estrutura{}, fmt.Errorf("a IA non atopou exercicios")
	}
	// Comprobar que a IA copiou e non reescribiu.
	for i, ex := range e.Exercicios {
		dub, cob := Comparar(ex.Enunciado, md)
		if cob >= 0 && cob < 0.9 {
			e.Avisos = append(e.Avisos, fmt.Sprintf("exercicio %d: o texto non coincide co lido (%.0f %%): revisa %s", i+1, cob*100, strings.Join(primeiras(dub, 6), ", ")))
		}
	}
	if e.Titulo == "" {
		e.Titulo = nome
	}
	return e, nil
}

func primeiras(l []string, n int) []string {
	if len(l) > n {
		return l[:n]
	}
	return l
}

var tituloPreguntaRe = regexp.MustCompile(`(?im)^[ \t]*(?:\*\*|#+\s*)?(?:PREGUNTA|EXERCICIO|EJERCICIO|QUESTION|PROBLEMA|ACTIVIDADE|ACTIVIDAD|TAREFA|CUESTIÓN|CUESTION)\s*\d+`)

// numeradaConPuntosRe: «2. Answer the two subsections… (2.5 points)». Só se
// usa se non hai títulos «Pregunta N»; esixir a puntuación evita partir as
// listas numeradas de dentro dun exercicio.
var numeradaConPuntosRe = regexp.MustCompile(`(?im)^[ \t]*(?:\*\*)?\d{1,2}\.\s[^\n]*\(\s*\d+(?:[.,]\d+)?\s*(?:points?|puntos?|pts?\.?)\s*\)`)

// EstruturarSenIA parte polos títulos de pregunta.
func EstruturarSenIA(pax []PaxinaLida, nome string) Estrutura {
	titulos := tituloPreguntaRe
	if !algunha(pax, tituloPreguntaRe) && algunha(pax, numeradaConPuntosRe) {
		titulos = numeradaConPuntosRe
	}
	e := Estrutura{Titulo: nome}
	type anaco struct {
		texto string
		pax   int
	}
	var anacos []anaco
	var instr strings.Builder
	for _, p := range pax {
		t := p.Markdown
		idx := titulos.FindAllStringIndex(t, -1)
		if len(idx) == 0 {
			if len(anacos) == 0 {
				instr.WriteString(t + "\n\n")
			} else {
				anacos[len(anacos)-1].texto += "\n\n" + t
			}
			continue
		}
		antes := strings.TrimSpace(t[:idx[0][0]])
		if antes != "" {
			if len(anacos) == 0 {
				instr.WriteString(antes + "\n\n")
			} else {
				anacos[len(anacos)-1].texto += "\n\n" + antes
			}
		}
		for i, m := range idx {
			fin := len(t)
			if i+1 < len(idx) {
				fin = idx[i+1][0]
			}
			anacos = append(anacos, anaco{texto: strings.TrimSpace(t[m[0]:fin]), pax: p.Num})
		}
	}
	e.Instrucions = strings.TrimSpace(instr.String())
	if len(anacos) == 0 && e.Instrucions != "" {
		anacos = append(anacos, anaco{texto: e.Instrucions, pax: 1})
		e.Instrucions = ""
	}
	for i, a := range anacos {
		primeira := strings.SplitN(a.texto, "\n", 2)[0]
		e.Exercicios = append(e.Exercicios, ExercicioLido{
			Titulo:    strings.Trim(strings.TrimSpace(primeira), "*# "),
			Apartado:  fmt.Sprint(i + 1),
			Paxina:    a.pax,
			Enunciado: a.texto,
		})
	}
	return e
}

func algunha(pax []PaxinaLida, re *regexp.Regexp) bool {
	for _, p := range pax {
		if re.MatchString(p.Markdown) {
			return true
		}
	}
	return false
}
