package main

// Textos fixos dos documentos (títulos de sección, «Sen tarefas», rótulos das
// actas...). Lua xera TODO o contido do documento, pero ningún texto vai
// escrito a lume no código: cada un ten unha clave, un valor por defecto e un
// campo no editor de plantillas («Textos do documento») onde se pode cambiar.
// A plantilla garda só os que se cambian (PlantillaLatex.Textos); baleiro =
// o valor por defecto de aquí.
//
// Un texto pode levar {{VARIABLES}} (as do curso e da plantilla: {{CENTRO}},
// {{DEPARTAMENTO}}...) e as propias do sitio onde se usa (poñeo a axuda de cada
// unha: {{N}} no trimestre, {{HORA_FIN}} no peche da acta...).
//
// O DESEÑO (cores, formato de títulos, etiquetas...) non está aquí: vai en
// LaTeX no bloque «Estilo» da plantilla (ver estilo_doc.go).

import (
	"strings"
)

// TextoDoc describe un texto fixo configurable.
type TextoDoc struct {
	Clave    string `json:"clave"`
	Grupo    string `json:"grupo"` // "comun", "cronograma", "actas" ou "formulario"
	Etiqueta string `json:"etiqueta"`
	Defecto  string `json:"defecto"`
	// Axuda lista as variables propias deste texto, se as ten.
	Axuda string `json:"axuda,omitempty"`
}

// textosDoc é o catálogo de textos, en orde de aparición no editor.
var textosDoc = []TextoDoc{
	// ── Comúns ──
	{"INDICE", "comun", "Título do índice", "Índice", ""},
	{"INTRODUCION", "comun", "Título da sección de introdución", "Introdución", ""},
	{"CURSO_ACADEMICO", "comun", "Rótulo do curso na portada de Markdown/Word/OpenDocument", "Curso académico {{CURSO}}", ""},
	{"SINATURA_TITULO", "comun", "Título da sección de sinatura (Word/OpenDocument)", "Sinatura", ""},
	{"LUGAR_DATA", "comun", "Lugar e data do peche", "En {{LOCALIDADE}}, a {{DATA_LONGA}}", ""},

	// ── Cronograma ──
	{"TITULO_CRONOGRAMA", "cronograma", "Título do documento (portada: {{TITULO}})", "Cronograma anual", ""},
	{"CRONOGRAMA", "cronograma", "Título da sección do cronograma", "Cronograma anual", ""},
	{"DESENVOLVEMENTO", "cronograma", "Título da sección de desenvolvemento", "Desenvolvemento por meses", ""},
	{"TRIMESTRE", "cronograma", "Nome de cada trimestre", "{{N}}.º trimestre", "{{N}}: número do trimestre"},
	{"SEN_TAREFAS", "cronograma", "Mes sen tarefas (no cronograma)", "Sen tarefas", ""},
	{"TAREFA_UNHA", "cronograma", "Etiqueta dun mes cunha tarefa", "1 tarefa", ""},
	{"TAREFAS_N", "cronograma", "Etiqueta dun mes con varias tarefas", "{{N}} tarefas", "{{N}}: cantas hai"},
	{"SEN_TITULO", "cronograma", "Tarefa sen título", "Tarefa sen título", ""},
	{"SEN_DESCRICION", "cronograma", "Tarefa sen descrición", "Sen descrición detallada.", ""},
	{"SEN_TAREFAS_CURSO", "cronograma", "Curso sen ningunha tarefa", "Aínda non hai tarefas cargadas en ningún mes.", ""},
	{"SEN_MESES", "cronograma", "Rango de datas sen meses", "O rango de datas do curso non contén ningún mes.", ""},
	{"SINATURA_TEXTO", "cronograma", "Frase antes da sinatura (Word/OpenDocument)", "E, para que conste, asina a presente programación anual:", ""},

	// ── Actas ──
	{"TITULO_ACTAS", "actas", "Título do documento (portada: {{TITULO}})", "Actas do departamento", ""},
	{"ACTAS_LISTA", "actas", "Título do índice das actas", "Actas do curso {{CURSO}}", ""},
	{"ACTA_KICKER", "actas", "Rótulo pequeno sobre o título da acta", "{{DEPARTAMENTO}}", ""},
	{"ACTA_TITULO", "actas", "Título de cada acta", "Acta n.º {{NUM}}", "{{NUM}}: número da acta no curso"},
	{"ACTA_SUBTITULO", "actas", "Subtítulo de cada acta", "Reunión ordinaria de {{MES_NOME}} de {{ANO_ACTA}}", "{{MES_NOME}}, {{ANO_ACTA}}"},
	{"ACTA_ENTRADA_INDICE", "actas", "Entrada de cada acta no índice", "Acta n.º {{NUM}} · {{MES_NOME}} de {{ANO_ACTA}}", "{{NUM}}, {{MES_NOME}}, {{ANO_ACTA}}"},
	{"ACTA_DATA", "actas", "Rótulo da data", "Data", ""},
	{"ACTA_HORA", "actas", "Rótulo da hora", "Hora", ""},
	{"ACTA_HORAS", "actas", "Intervalo de horas", "{{HORA_INICIO}} – {{HORA_FIN}}", "{{HORA_INICIO}}, {{HORA_FIN}}"},
	{"ACTA_LUGAR", "actas", "Rótulo do lugar", "Lugar", ""},
	{"ACTA_ASISTENTES", "actas", "Título dos asistentes", "Asistentes", ""},
	{"ACTA_AUSENTES", "actas", "Rótulo dos ausentes", "Ausentes", ""},
	{"ACTA_SEN_AUSENTES", "actas", "Cando non falta ninguén", "Ningún", ""},
	{"ACTA_ORDE", "actas", "Título da orde do día", "Orde do día", ""},
	{"ACTA_SEGUIMENTO", "actas", "Título do seguimento das materias", "Seguimento das programacións didácticas", ""},
	{"ACTA_SEN_SEGUIMENTO", "actas", "Mes sen informes de materias", "Non hai informes de seguimento das materias neste mes.", ""},
	{"ACTA_DESENVOLVEMENTO", "actas", "Título do desenvolvemento da sesión", "Desenvolvemento da sesión", ""},
	{"ACTA_ACORDOS", "actas", "Título dos acordos", "Acordos", ""},
	{"ACTA_ROGOS", "actas", "Título de rogos e preguntas", "Rogos e preguntas", ""},
	{"ACTA_SEN_CONTIDO", "actas", "Apartado da acta sen contido", "Sen contido.", ""},
	{"ACTA_PECHE", "actas", "Peche da acta", "Sen máis asuntos que tratar, levántase a sesión ás {{HORA_FIN}} do {{DATA_ACTA_LONGA}}.", "{{HORA_FIN}}, {{DATA_ACTA_LONGA}}"},
	{"ACTA_SECRETARIO", "actas", "Rótulo da sinatura de quen redacta", "O/A secretario/a", ""},
	{"ACTA_VISTO", "actas", "Rótulo da sinatura da xefatura", "Vº e Prace, {{CARGO}}", ""},
	{"SEG_MATERIA", "actas", "Rótulo da materia", "Materia", ""},
	{"SEG_DOCENTE", "actas", "Rótulo do docente", "Docente", ""},
	{"SEG_GRUPOS", "actas", "Rótulo dos grupos", "Grupos", ""},
	{"SEG_CUMPRIMENTO", "actas", "Rótulo do grao de cumprimento", "Grao de cumprimento da programación", ""},
	{"SEG_CONTIDOS", "actas", "Rótulo dos contidos", "Contidos traballados", ""},
	{"SEG_RESULTADOS", "actas", "Rótulo dos resultados", "Resultados e observacións", ""},
	{"SEG_DIFICULTADES", "actas", "Rótulo das dificultades", "Dificultades detectadas", ""},
	{"SEG_MEDIDAS", "actas", "Rótulo das medidas", "Medidas e propostas de mellora", ""},
	{"SEG_SEN_DATOS", "actas", "Materia sen informe do mes", "Sen informe neste mes.", ""},
	{"SEN_ACTAS", "actas", "Curso sen ningunha acta", "Aínda non hai ningunha acta neste curso.", ""},

	// ── Formularios propios ──
	{"TITULO_FORMULARIO", "formulario", "Título do documento (portada: {{TITULO}})", "{{FORMULARIO}}", "{{FORMULARIO}}: o título do formulario"},
	{"FORM_SI", "formulario", "Verificación marcada", "Si", ""},
	{"FORM_NON", "formulario", "Verificación sen marcar", "Non", ""},
	{"FORM_SEN_CONTIDO", "formulario", "Formulario sen ningún dato (baleiro = non sae nada)", "", ""},
	{"FORM_GRAFICO_TOTAL", "formulario", "Rótulo do total no centro dunha rosca", "Total", ""},
	{"FORM_PUNTOS", "formulario", "Puntos dunha pregunta ({{N}}: os puntos)", "{{N}} pt", "{{N}}: os puntos"},
	{"FORM_SOLUCION", "formulario", "Rótulo da solución debaixo dunha pregunta", "Solución", ""},
	{"FORM_RESULTADO", "formulario", "Rótulo do resultado final dunha pregunta", "Resultado", ""},
	{"FORM_SOLUCIONARIO", "formulario", "Título do solucionario ao final", "Solucionario", ""},
	{"FORM_VARIANTE", "formulario", "Título de cada copia cando hai varias ({{N}}: número, {{SEMENTE}}: a semente)", "Modelo {{N}}", "{{N}}: o número da copia; {{SEMENTE}}: a súa semente"},
}

var textosDocPorClave = func() map[string]TextoDoc {
	m := make(map[string]TextoDoc, len(textosDoc))
	for _, t := range textosDoc {
		m[t.Clave] = t
	}
	return m
}()

// normalizarTextosDoc deixa só os textos que existen no catálogo e que teñen
// valor (un baleiro equivale a «o de Lua»).
func normalizarTextosDoc(m map[string]string) map[string]string {
	if len(m) == 0 {
		return nil
	}
	out := map[string]string{}
	for k, v := range m {
		k = strings.ToUpper(strings.TrimSpace(k))
		if _, ok := textosDocPorClave[k]; ok && strings.TrimSpace(v) != "" {
			out[k] = strings.TrimSpace(v)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// textosDocIguais compara dous mapas de textos (para saber se se editou unha
// plantilla de fábrica).
func textosDocIguais(a, b map[string]string) bool {
	a, b = normalizarTextosDoc(a), normalizarTextosDoc(b)
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// textosDocumento resolve os textos dun documento: o valor da plantilla se o
// ten, senón o de Lua, e substitúe as {{VARIABLES}} (as do curso máis as
// propias de cada sitio).
type textosDocumento struct {
	plantilla *PlantillaLatex
	vars      map[string]string
}

func novosTextosDoc(p *PlantillaLatex, vars map[string]string) textosDocumento {
	if p == nil {
		p = &PlantillaLatex{}
	}
	return textosDocumento{plantilla: p, vars: vars}
}

// cru devolve o texto da clave con {{VARIABLES}} sen substituír.
func (d textosDocumento) cru(clave string) string {
	if v := strings.TrimSpace(d.plantilla.Textos[clave]); v != "" {
		return v
	}
	return textosDocPorClave[clave].Defecto
}

// T é o texto resolvido (texto plano). `extra` engade variables propias do
// sitio (p. ex. {"N": "2"}); unha variable sen valor queda baleira.
func (d textosDocumento) T(clave string, extra ...map[string]string) string {
	vars := d.vars
	if len(extra) > 0 {
		vars = make(map[string]string, len(d.vars)+len(extra[0]))
		for k, v := range d.vars {
			vars[k] = v
		}
		for k, v := range extra[0] {
			vars[k] = v
		}
	}
	out, _ := substituirVariables(d.cru(clave), d.plantilla, vars, "", nil)
	return strings.TrimSpace(out)
}

// L é o texto resolvido e escapado para LaTeX.
func (d textosDocumento) L(clave string, extra ...map[string]string) string {
	return EscapeLatexProse(d.T(clave, extra...))
}

// engadirVariables pon en `vars` as variables que veñen dos textos do
// documento: {{TITULO}} (o texto `tituloClave`) e {{CURSO_ACADEMICO}}. Cámbiaas
// a plantilla ao cambiar eses textos, e valen na portada e na cabeceira.
func (d textosDocumento) engadirVariables(vars map[string]string, tituloClave string) {
	vars["TITULO"] = d.T(tituloClave)
	vars["CURSO_ACADEMICO"] = d.T("CURSO_ACADEMICO")
}

// clavesTextosDoc lista as claves dos textos fixos (para a axuda das ferramentas).
func clavesTextosDoc() string {
	nomes := make([]string, len(textosDoc))
	for i, t := range textosDoc {
		nomes[i] = t.Clave
	}
	return strings.Join(nomes, ", ")
}
