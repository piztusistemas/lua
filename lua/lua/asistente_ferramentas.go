package main

// Catálogo de ferramentas do asistente (ver asistente.go). Aquí están as
// DEFINICIÓNS de todas (nome, descrición, argumentos) e o código das de
// backend; as de frontend (Onde == "frontend") execútaas
// frontend/src/asistente/ferramentas.js co mesmo nome.

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
)

// ---------- Axudas para escribir esquemas ----------

func esqTxt(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }
func esqNum(desc string) map[string]any { return map[string]any{"type": "number", "description": desc} }
func esqBool(desc string) map[string]any {
	return map[string]any{"type": "boolean", "description": desc}
}
func esqEnum(desc string, valores ...string) map[string]any {
	return map[string]any{"type": "string", "description": desc, "enum": valores}
}

// esqObx constrúe o JSON Schema dun obxecto de argumentos.
func esqObx(props map[string]any, obrigatorios ...string) string {
	e := map[string]any{"type": "object", "properties": props}
	if len(obrigatorios) > 0 {
		e["required"] = obrigatorios
	}
	b, _ := json.Marshal(e)
	return string(b)
}

const (
	tipoLectura   = "lectura"
	tipoEscritura = "escritura"
	tipoConfirmar = "confirmar"
)

var camposPlantilla = map[string]any{
	"nome":        esqTxt("Nome da plantilla."),
	"descricion":  esqTxt("Descrición dunha liña."),
	"latex":       esqTxt("LaTeX da maqueta (cabeceira, pé, marxes). Ten que levar {{CORPO}} unha soa vez."),
	"portada":     esqTxt("LaTeX da portada (entra onde a plantilla poña {{PORTADA}})."),
	"sinatura":    esqTxt("LaTeX da última folla (entra onde a plantilla poña {{SINATURA}})."),
	"introducion": esqTxt("Texto da introdución por defecto, con variables {{CENTRO}}, {{CURSO}}..."),
	"estilo":      esqTxt("DESEÑO do documento en LaTeX (formato dos títulos, etiquetas do cronograma, cabeceira das actas...): substitúe o de Lúa enteiro; baleiro = o de Lúa. Parte do estilo por defecto (plantilla_editor_ler dá estiloPorDefecto)."),
	"textos": map[string]any{
		"type": "object",
		"description": "Textos fixos do documento (títulos de sección, «Sen tarefas», rótulos das actas...): clave -> texto; só se cambian os que indiques (un texto baleiro volve ao de Lúa). " +
			"Claves: " + clavesTextosDoc() + ". Levan variables {{CENTRO}}, {{CURSO}}, {{DEPARTAMENTO}}... e as propias de cada un (ver plantilla_editor_ler: textosDispoñibles).",
		"additionalProperties": map[string]any{"type": "string"},
	},
	"markdown": esqTxt("Maqueta equivalente en Markdown para exportar a .md (co mesmo {{CORPO}})."),
	"motor":    esqTxt("Compilador que esixe esta plantilla: pdflatex, xelatex ou lualatex; baleiro = o de Opcións."),
	"ofimatica": map[string]any{
		"type": "object",
		"description": "Deseño para Word (.docx) e OpenDocument (.odt): só os campos que indiques (os demais consérvanse). " +
			"portada usa liñas: '@logo 35' (logotipo do curso, alto en mm), '@imaxe nome.png 30' (imaxe do cartafol da plantilla), " +
			"'@espazo 12', '# Título grande', '## Subtítulo', '---' (liña), '### Liña destacada (centro)' e liñas normais; con variables {{CENTRO}}, {{CURSO}}...",
		"properties": map[string]any{
			"fonte":         esqTxt("Tipo de letra (p.ex. Calibri, Georgia). Baleiro = Arial."),
			"corPrincipal":  esqTxt("Cor dos títulos e da portada, hexadecimal sen # (p.ex. 1F3864)."),
			"corSecundaria": esqTxt("Cor dos subtítulos, hexadecimal."),
			"corTexto":      esqTxt("Cor do corpo do texto, hexadecimal."),
			"corApagada":    esqTxt("Cor dos textos secundarios, hexadecimal."),
			"corLinha":      esqTxt("Cor dos filetes e bordos, hexadecimal."),
			"tamanoCorpo":   esqNum("Tamaño do corpo en pt (8-16; 0 = 11)."),
			"interlinado":   esqNum("Interliñado do texto (1 = sinxelo, 1.15 = por defecto, 1.5, 2...; entre 1 e 3; 0 = o por defecto)."),
			"marxeSuperior": esqNum("Marxe superior da páxina en cm (0.5-6; 0 = a por defecto: 2.5 en Word, 2 en OpenDocument)."),
			"marxeInferior": esqNum("Marxe inferior en cm (0.5-6; 0 = por defecto)."),
			"marxeEsquerda": esqNum("Marxe esquerda en cm (0.5-6; 0 = por defecto)."),
			"marxeDereita":  esqNum("Marxe dereita en cm (0.5-6; 0 = por defecto)."),
			"portada":       esqTxt("Portada de Word/OpenDocument (ver a descrición do campo ofimatica)."),
			"cabeceira":     esqTxt("Cabeceira de todas as páxinas menos a portada: unha liña por elemento, en orde e nunha soa liña de papel: '@imaxe logo1.png 10', '@imaxe logo2.pdf 10' (varios logotipos, alto en mm; PNG, JPEG ou PDF do cartafol de imaxes da plantilla), texto con variables ({{CENTRO}}...), e '@esquerda'/'@centro'/'@dereita' para o aliñamento (por defecto dereita)."),
			"pe":            esqTxt("Pé: mesmo formato ca a cabeceira (imaxes '@imaxe', texto, aliñamento; por defecto centro); o número de páxina engádese ao final."),
		},
	},
	"variables": map[string]any{
		"type": "array", "description": "Variables propias da plantilla (teléfono, correo...).",
		"items": map[string]any{"type": "object", "properties": map[string]any{
			"nome":     esqTxt("Nome en MAIÚSCULAS sen espazos, p.ex. TELEFONO."),
			"etiqueta": esqTxt("Como se amosa na interface."),
			"valor":    esqTxt("Valor actual."),
		}, "required": []string{"nome"}},
	},
}

func esqPlantilla(extra map[string]any, obrigatorios ...string) string {
	props := map[string]any{}
	for k, v := range camposPlantilla {
		props[k] = v
	}
	for k, v := range extra {
		props[k] = v
	}
	return esqObx(props, obrigatorios...)
}

// ---------- Ferramentas de frontend (só a definición) ----------

var ferramentasFrontend = []IAFerramenta{
	{
		Nome: "curso_ler", Tipo: tipoLectura, Onde: "frontend",
		Descricion: "Le o curso que hai agora en edición: datos do curso e do centro, e os meses coas súas tarefas (id, título, descrición, cor, mes). As imaxes non se inclúen (só se indica cantas ten cada tarefa). Chámaa sempre antes de editar.",
		Parametros: esqObx(map[string]any{
			"ambito": esqEnum("Que ler: 'todo' (por defecto), 'config' (só os datos do curso) ou 'tarefas'.", "todo", "config", "tarefas"),
			"mes":    esqTxt("Opcional: só as tarefas deste mes, formato AAAA-MM."),
		}),
	},
	{
		Nome: "curso_configurar", Tipo: tipoEscritura, Onde: "frontend",
		Descricion: "Cambia datos do curso e do centro (só os campos que indiques). As datas van en formato AAAA-MM-DD. O texto da introdución é o do propio curso e ten prioridade sobre o da plantilla.",
		Parametros: esqObx(map[string]any{
			"anoAcademico":  esqTxt("Curso académico, p.ex. 2025/2026."),
			"dataInicio":    esqTxt("Data de inicio AAAA-MM-DD."),
			"dataFin":       esqTxt("Data de fin AAAA-MM-DD."),
			"centro":        esqTxt("Nome do centro."),
			"localidade":    esqTxt("Localidade."),
			"departamento":  esqTxt("Nome do departamento, p.ex. Departamento de Matemáticas (Lúa serve para calquera departamento)."),
			"xefe":          esqTxt("Nome de quen asina (xefe/a do departamento)."),
			"cargoSinatura": esqTxt("Cargo que figura na sinatura."),
			"introducion":   esqTxt("Texto da introdución do informe (baleiro = usar o da plantilla ou o por defecto)."),
			"bloqueado":     esqBool("Bloquear a edición do curso."),
		}),
	},
	{
		Nome: "ferramenta_activa", Tipo: tipoEscritura, Onde: "frontend",
		Descricion: "Lúa ten varias ferramentas (columna da esquerda): 'cronograma' (o cronograma anual do departamento) e 'actas' (as actas mensuais do departamento e os informes de seguimento das materias). Sen argumentos di cal está activa; con 'ferramenta' cambia a ela. As ferramentas curso_* e tarefa_* traballan co cronograma; as actas_* e acta_* coas actas. curso_novo, curso_abrir, curso_gardar, informe_xerar, informe_exportar e historial valen para a ferramenta activa.",
		Parametros: esqObx(map[string]any{"ferramenta": esqEnum("Ferramenta á que cambiar.", "cronograma", "actas")}),
	},
	{
		Nome: "actas_ler", Tipo: tipoLectura, Onde: "frontend",
		Descricion: "Le o curso de ACTAS que hai en edición (ferramenta 'actas'): datos do departamento, membros, materias impartidas e, por mes, a acta (data, horas, asistentes, orde do día, desenvolvemento, acordos, rogos) e os informes de seguimento de cada materia. Chámaa sempre antes de editar.",
		Parametros: esqObx(map[string]any{"mes": esqTxt("Opcional: só este mes, formato AAAA-MM.")}),
	},
	{
		Nome: "actas_configurar", Tipo: tipoEscritura, Onde: "frontend",
		Descricion: "Cambia os datos do departamento e do curso de actas (só os campos que indiques). As datas van en AAAA-MM-DD e as horas en HH:MM.",
		Parametros: esqObx(map[string]any{
			"anoAcademico":  esqTxt("Curso académico, p.ex. 2025/2026."),
			"dataInicio":    esqTxt("Data de inicio AAAA-MM-DD."),
			"dataFin":       esqTxt("Data de fin AAAA-MM-DD."),
			"centro":        esqTxt("Nome do centro."),
			"localidade":    esqTxt("Localidade."),
			"departamento":  esqTxt("Nome do departamento, p.ex. Departamento de Matemáticas."),
			"xefe":          esqTxt("Xefe/a do departamento (asina co Vº e Prace)."),
			"secretario":    esqTxt("Secretario/a: quen redacta as actas."),
			"cargoSinatura": esqTxt("Cargo do xefe/a que figura na sinatura."),
			"horaHabitual":  esqTxt("Hora habitual das reunións (HH:MM); entra nas actas novas."),
			"lugarHabitual": esqTxt("Lugar habitual das reunións; entra nas actas novas."),
			"bloqueado":     esqBool("Bloquear a edición do curso."),
		}),
	},
	{
		Nome: "actas_membros", Tipo: tipoEscritura, Onde: "frontend",
		Descricion: "Define a lista COMPLETA de membros do departamento (substitúe a actual; os que xa existen polo nome conservan a súa asistencia). Cada membro: nome e, opcional, cargo.",
		Parametros: esqObx(map[string]any{"membros": map[string]any{
			"type": "array", "description": "Membros do departamento.",
			"items": map[string]any{"type": "object", "properties": map[string]any{"nome": esqTxt("Nome e apelidos."), "cargo": esqTxt("Cargo (opcional).")}, "required": []string{"nome"}},
		}}, "membros"),
	},
	{
		Nome: "actas_materias", Tipo: tipoEscritura, Onde: "frontend",
		Descricion: "Define a lista COMPLETA de materias que imparte o departamento (substitúe a actual; as que xa existen polo nome conservan os seus informes de seguimento; as que se quitan perden os seus informes). Por cada materia hai un informe de seguimento en cada mes.",
		Parametros: esqObx(map[string]any{"materias": map[string]any{
			"type": "array", "description": "Materias impartidas.",
			"items": map[string]any{"type": "object", "properties": map[string]any{
				"nome": esqTxt("Nome da materia."), "docente": esqTxt("Docente que a imparte."), "grupos": esqTxt("Grupos ou cursos, p.ex. 1º ESO A."),
			}, "required": []string{"nome"}},
		}}, "materias"),
	},
	{
		Nome: "acta_gardar", Tipo: tipoEscritura, Onde: "frontend",
		Descricion: "Crea (ou modifica, se xa existe) a acta MENSUAL dun mes: só os campos que indiques. Unha acta nova trae a hora e o lugar habituais e todos os membros como asistentes. Os textos (ordeDia, desenvolvemento, acordos, rogos) son Markdown e substitúense enteiros. A acta recolle automaticamente os informes de seguimento das materias dese mes (acta_seguimento).",
		Parametros: esqObx(map[string]any{
			"mes":             esqTxt("Mes, formato AAAA-MM (dentro do rango do curso)."),
			"data":            esqTxt("Data da reunión AAAA-MM-DD."),
			"horaInicio":      esqTxt("Hora de inicio HH:MM."),
			"horaFin":         esqTxt("Hora de fin HH:MM."),
			"lugar":           esqTxt("Lugar da reunión."),
			"asistentes":      map[string]any{"description": "'todos' ou a lista de nomes dos membros que asisten (os demais figuran coma ausentes).", "oneOf": []any{map[string]any{"type": "string"}, map[string]any{"type": "array", "items": map[string]any{"type": "string"}}}},
			"ordeDia":         esqTxt("Orde do día (Markdown, p.ex. lista numerada)."),
			"desenvolvemento": esqTxt("Desenvolvemento da sesión (Markdown)."),
			"acordos":         esqTxt("Acordos adoptados (Markdown)."),
			"rogos":           esqTxt("Rogos e preguntas (Markdown)."),
		}, "mes"),
	},
	{
		Nome: "acta_eliminar", Tipo: tipoEscritura, Onde: "frontend",
		Descricion: "Elimina a acta dun mes (os informes de seguimento das materias dese mes consérvanse; o usuario pode desfacelo co historial). Fai isto só se llo piden.",
		Parametros: esqObx(map[string]any{"mes": esqTxt("Mes AAAA-MM.")}, "mes"),
	},
	{
		Nome: "acta_seguimento", Tipo: tipoEscritura, Onde: "frontend",
		Descricion: "Crea ou modifica o informe de seguimento dunha MATERIA nun mes (só os campos que indiques; os textos son Markdown e substitúense enteiros). É o que recolle a acta do mes. A materia ten que estar definida (actas_materias).",
		Parametros: esqObx(map[string]any{
			"mes":          esqTxt("Mes AAAA-MM."),
			"materia":      esqTxt("Nome ou id da materia."),
			"progreso":     esqNum("Grao de cumprimento da programación, 0-100 (null = sen dato)."),
			"contidos":     esqTxt("Contidos traballados."),
			"resultados":   esqTxt("Resultados e observacións."),
			"dificultades": esqTxt("Dificultades detectadas."),
			"medidas":      esqTxt("Medidas adoptadas e propostas de mellora."),
			"docente":      esqTxt("Docente deste mes (baleiro = o da materia)."),
			"grupos":       esqTxt("Grupos deste mes (baleiro = os da materia)."),
		}, "mes", "materia"),
	},
	{
		Nome: "actas_ambito", Tipo: tipoEscritura, Onde: "frontend",
		Descricion: "Escolle que documento de actas se xera e se imprime (informe_xerar, informe_exportar): 'curso' (todas as actas do curso, con portada e índice) ou un mes AAAA-MM (só a acta dese mes). Sen argumentos di o ámbito actual e os posibles.",
		Parametros: esqObx(map[string]any{"ambito": esqTxt("'curso' ou AAAA-MM.")}),
	},
	{
		Nome: "tarefa_engadir", Tipo: tipoEscritura, Onde: "frontend",
		Descricion: "Engade unha tarefa a un mes do curso. Devolve o id da nova tarefa.",
		Parametros: esqObx(map[string]any{
			"mes":        esqTxt("Mes, formato AAAA-MM (ten que estar dentro do rango de datas do curso)."),
			"titulo":     esqTxt("Título da tarefa."),
			"descricion": esqTxt("Descrición en Markdown (ver formato admitido)."),
			"cor":        esqEnum("Cor da tarefa.", "blue", "emerald", "amber", "rose", "purple", "cyan", "indigo", "stone"),
		}, "mes", "titulo"),
	},
	{
		Nome: "tarefa_editar", Tipo: tipoEscritura, Onde: "frontend",
		Descricion: "Modifica unha tarefa existente (só os campos que indiques). A descrición substitúese enteira: le antes a actual (curso_ler) e devolve o texto completo, mantendo as imaxes (liñas ![..](img:..)). Con 'mes' cámbiaa de mes.",
		Parametros: esqObx(map[string]any{
			"id":         esqTxt("Id da tarefa (de curso_ler)."),
			"titulo":     esqTxt("Novo título."),
			"descricion": esqTxt("Nova descrición COMPLETA en Markdown."),
			"cor":        esqEnum("Nova cor.", "blue", "emerald", "amber", "rose", "purple", "cyan", "indigo", "stone"),
			"mes":        esqTxt("Mover a este mes, AAAA-MM."),
		}, "id"),
	},
	{
		Nome: "tarefa_eliminar", Tipo: tipoEscritura, Onde: "frontend",
		Descricion: "Elimina unha tarefa do curso (o usuario pode desfacelo co botón Desfacer). Fai isto só se llo piden.",
		Parametros: esqObx(map[string]any{"id": esqTxt("Id da tarefa.")}, "id"),
	},
	{
		Nome: "tarefa_mover", Tipo: tipoEscritura, Onde: "frontend",
		Descricion: "Cambia a orde dunha tarefa dentro do seu mes.",
		Parametros: esqObx(map[string]any{
			"id":        esqTxt("Id da tarefa."),
			"direccion": esqEnum("Cara onde moverla.", "arriba", "abaixo"),
		}, "id", "direccion"),
	},
	{
		Nome: "tarefa_abrir", Tipo: tipoEscritura, Onde: "frontend",
		Descricion: "Abre a pantalla de edición dunha tarefa para que o usuario a revise ou a modifique el mesmo.",
		Parametros: esqObx(map[string]any{"id": esqTxt("Id da tarefa.")}, "id"),
	},
	{
		Nome: "curso_logotipo", Tipo: tipoEscritura, Onde: "frontend",
		Descricion: "GARDA NO JSON DO CURSO o logotipo do centro: usa unha imaxe adxunta (adxuntos_listar) ou quita o logotipo actual. O logotipo do curso sae na portada dos informes que o usen (macro \\logocentro no PDF, @logo en Word/OpenDocument). Pódese desfacer co historial.",
		Parametros: esqObx(map[string]any{
			"adxunto": esqTxt("Id do adxunto (unha imaxe) que pasa a ser o logotipo."),
			"quitar":  esqBool("Quitar o logotipo actual (en vez de poñer un)."),
		}),
	},
	{
		Nome: "tarefa_imaxe", Tipo: tipoEscritura, Onde: "frontend",
		Descricion: "GARDA NO JSON DO CURSO unha imaxe adxunta nunha tarefa e insire na descrición a liña ![alt](img:ID =60%). Podes elixir a largura (en % da columna), o texto alternativo, o aliñamento e se vai ao principio ou ao final da descrición. Pódese desfacer co historial.",
		Parametros: esqObx(map[string]any{
			"id":       esqTxt("Id da tarefa (de curso_ler)."),
			"adxunto":  esqTxt("Id do adxunto (unha imaxe)."),
			"largura":  esqNum("Largura en % da columna (por defecto 60)."),
			"alt":      esqTxt("Texto alternativo / pé da imaxe."),
			"alinear":  esqEnum("Aliñamento da imaxe.", "centro", "esquerda", "dereita"),
			"posicion": esqEnum("Onde ir na descrición.", "final", "inicio"),
		}, "id", "adxunto"),
	},
	{
		Nome: "curso_novo", Tipo: tipoConfirmar, Onde: "frontend",
		Descricion: "Crea un curso novo baleiro (o que hai agora péchase; se ten cambios sen gardar o programa pregunta ao usuario).",
	},
	{
		Nome: "catalogo_reutilizar", Tipo: tipoEscritura, Onde: "frontend",
		Descricion: "Mete no proxecto aberto formularios do catálogo común (ids de catalogo_ler), o cronograma («cronograma») ou as actas («actas»): o proxecto leva unha copia de cada un e aparecen na columna da esquerda.",
		Parametros: esqObx(map[string]any{"ids": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Ids dos formularios, «cronograma» ou «actas»."}}, "ids"),
	},
	{
		Nome: "curso_abrir", Tipo: tipoEscritura, Onde: "frontend",
		Descricion: "Abre a xanela Trasladar na lapela «Meter» para que o usuario meta en Lúa un ficheiro (.lua, .json dun curso, formulario ou plantilla, PDF ou Markdown). Lúa di que leva e onde vai antes de metelo.",
	},
	{
		Nome: "curso_gardar", Tipo: tipoEscritura, Onde: "frontend",
		Descricion: "O curso xa se garda só en Lúa (dentro do proxecto); isto escríbeo agora mesmo. Con comoNovo=true abre ademais a xanela Trasladar para sacalo a un ficheiro.",
		Parametros: esqObx(map[string]any{"comoNovo": esqBool("Abrir Trasladar para sacar o curso a un ficheiro.")}),
	},
	{
		Nome: "informe_xerar", Tipo: tipoEscritura, Onde: "frontend",
		Descricion: "Xera o informe (compila o LaTeX coa plantilla activa) e actualiza a vista previa. Devolve se saíu ben, os avisos e o número de páxinas.",
	},
	{
		Nome: "informe_exportar", Tipo: tipoEscritura, Onde: "frontend",
		Descricion: "Exporta o informe a un ficheiro. Abre o diálogo de Gardar para que o usuario escolla onde. Formatos: pdf, tex, md (Markdown), docx (Word), odt (OpenDocument).",
		Parametros: esqObx(map[string]any{"formato": esqEnum("Formato.", "pdf", "tex", "md", "docx", "odt")}, "formato"),
	},
	{
		Nome: "historial", Tipo: tipoEscritura, Onde: "frontend",
		Descricion: "Desfai ou refai o último cambio no curso.",
		Parametros: esqObx(map[string]any{"accion": esqEnum("Acción.", "desfacer", "refacer")}, "accion"),
	},
	{
		Nome: "resultado_xanela", Tipo: tipoEscritura, Onde: "frontend",
		Descricion: "Leva a vista previa do informe a unha xanela á parte (desacoplar, útil con dous monitores) ou volve acoplala.",
		Parametros: esqObx(map[string]any{"accion": esqEnum("Acción.", "desacoplar", "acoplar")}, "accion"),
	},
	{
		Nome: "plantilla_escoller", Tipo: tipoEscritura, Onde: "frontend",
		Descricion: "Escolle a plantilla activa para este curso (id de plantillas_listar; cadea baleira = sen plantilla) e volve xerar o informe se xa había un.",
		Parametros: esqObx(map[string]any{"id": esqTxt("Id da plantilla.")}, "id"),
	},
	{
		Nome: "axustes_texto", Tipo: tipoEscritura, Onde: "frontend",
		Descricion: "Le ou cambia os axustes do texto do PDF: tamaño de letra, interliñado e familia tipográfica. Sen argumentos só le os valores actuais.",
		Parametros: esqObx(map[string]any{
			"tamanoPt":    esqNum("Tamaño do corpo en pt (0 = por defecto)."),
			"interlinado": esqNum("Factor de interliñado (0 = por defecto, 1.0)."),
			"fonte":       esqTxt("Familia: '' (por defecto), 'serif', 'sans', 'palatino' ou o nome dunha fonte do sistema (esixe xelatex)."),
		}),
	},
	{
		Nome: "usuario_avisar", Tipo: tipoLectura, Onde: "frontend",
		Descricion: "Mostra unha mensaxe ao usuario na propia interface do programa (barra de estado e avisos), fóra do chat. Úsaa para avisos importantes; para responder xa che chega co texto.",
		Parametros: esqObx(map[string]any{
			"mensaxe": esqTxt("Texto do aviso."),
			"tipo":    esqEnum("Tipo.", "info", "aviso"),
		}, "mensaxe"),
	},
	{
		Nome: "usuario_preguntar", Tipo: tipoLectura, Onde: "frontend",
		Descricion: "Pregunta algo ao usuario e devolve a súa resposta. Se pasas 'opcions' escolle unha delas; se non, escribe texto libre. Úsaa cando falte un dato ou haxa varias formas de facer algo.",
		Parametros: esqObx(map[string]any{
			"pregunta": esqTxt("A pregunta."),
			"opcions":  map[string]any{"type": "array", "description": "Respostas posibles (opcional).", "items": map[string]any{"type": "string"}},
		}, "pregunta"),
	},
}

// ferramentasTarefa son as ferramentas do asistente da PANTALLA DE EDICIÓN DUNHA
// TAREFA (ámbito "tarefa", ver ambitosAsistente): tocan só o borrador que hai no
// editor, non o curso. O usuario decide despois se o aplica (Aplicar/Gardar).
var ferramentasTarefa = []IAFerramenta{
	{
		Nome: "editor_ler", Tipo: tipoLectura, Onde: "frontend",
		Descricion: "Le a tarefa que está aberta no editor, tal como está agora (o borrador, con cambios sen aplicar incluídos): título, descrición en Markdown, cor, mes, se é nova e que imaxes leva. Chámaa sempre antes de cambiar nada.",
	},
	{
		Nome: "editor_cambiar", Tipo: tipoEscritura, Onde: "frontend",
		Descricion: "Cambia o borrador da tarefa no editor (só os campos que indiques). A descrición substitúese enteira: devolve o texto COMPLETO, conservando as liñas de imaxe ![..](img:..). O cambio non se garda no curso ata que o usuario prema Aplicar ou Gardar, e el pode desfacelo co botón «Desfacer cambio da IA».",
		Parametros: esqObx(map[string]any{
			"titulo":     esqTxt("Novo título."),
			"descricion": esqTxt("Nova descrición COMPLETA en Markdown."),
			"cor":        esqEnum("Nova cor.", "blue", "emerald", "amber", "rose", "purple", "cyan", "indigo", "stone"),
		}),
	},
}

// ferramentasPlantillaEditor son as do asistente do EDITOR DE PLANTILLAS (ámbito
// "plantilla", ver ambitosAsistente): tocan só o borrador que hai no editor,
// non o disco. O usuario decide despois se o aplica (Aplicar/Gardar) e ve o
// resultado na vista previa do propio editor.
var ferramentasPlantillaEditor = []IAFerramenta{
	{
		Nome: "plantilla_editor_ler", Tipo: tipoLectura, Onde: "frontend",
		Descricion: "Le a plantilla que está aberta no editor, tal como está agora (o borrador, con cambios sen aplicar incluídos): nome, LaTeX da maqueta, portada, sinatura, introdución, Markdown, compilador, variables, imaxes do seu cartafol e o deseño de Word/OpenDocument (fonte, tamaño, interliñado, marxes, cores, cabeceira, pé e portada). Chámaa sempre antes de cambiar nada.",
	},
	{
		Nome: "plantilla_editor_cambiar", Tipo: tipoEscritura, Onde: "frontend",
		Descricion: "Cambia o borrador da plantilla no editor: só os campos que indiques (nos textos vai o valor COMPLETO; en 'ofimatica' só os subcampos que cambien; 'variables' substitúe a lista enteira). O cambio non se garda ata que o usuario prema Aplicar ou Gardar, e el pode desfacelo co botón «Desfacer cambio da IA». Ten que seguir habendo {{CORPO}} no LaTeX. Despois comproba o resultado con plantilla_previsualizar.",
		Parametros: esqPlantilla(nil),
	},
}

// ferramentasDoAmbitoPlantilla: ademais das de ferramentasPlantillaEditor, o
// asistente do editor de plantillas pode probar o borrador, mirar e gardar
// adxuntos (logotipos, documentos de mostra) e buscar na web.
var ferramentasDoAmbitoPlantilla = map[string]bool{
	"plantilla_previsualizar": true, "plantilla_comprobar": true,
	"adxuntos_listar": true, "adxunto_ler": true, "adxunto_ver": true, "adxunto_extraer_imaxes": true,
	"adxunto_gardar_en_plantilla": true, "plantilla_imaxe_eliminar": true,
	"web_buscar": true, "web_consultar": true, "usuario_preguntar": true, "usuario_avisar": true,
}

// ambitoTarefa: só isto ve e pode facer o asistente dunha tarefa.
var ferramentasDoAmbitoTarefa = map[string]bool{
	"editor_ler": true, "editor_cambiar": true, "web_buscar": true, "web_consultar": true,
	"usuario_preguntar": true, "usuario_avisar": true,
}

// ---------- Ferramentas de backend ----------

type ferramentaBackend struct {
	def  IAFerramenta
	exec func(a *App, args map[string]any) (any, error)
}

var ferramentasBackend = []ferramentaBackend{
	{
		def: IAFerramenta{
			Nome: "plantillas_listar", Tipo: tipoLectura, Onde: "backend",
			Descricion: "Lista as plantillas da biblioteca (id, nome, descrición, se é a activa, variables). Para ver o LaTeX dunha, usa plantilla_ler.",
		},
		exec: func(a *App, _ map[string]any) (any, error) {
			est, err := a.ListarPlantillas()
			if err != nil {
				return nil, err
			}
			type resumo struct {
				ID          string         `json:"id"`
				Nome        string         `json:"nome"`
				Descricion  string         `json:"descricion"`
				Activa      bool           `json:"activa"`
				DeFabrica   bool           `json:"deFabrica"`
				Motor       string         `json:"motor"`
				Variables   []PlantillaVar `json:"variables"`
				TenPortada  bool           `json:"tenPortada"`
				TenSinatura bool           `json:"tenSinatura"`
				Imaxes      []string       `json:"imaxes"`
			}
			out := []resumo{}
			for _, p := range est.Plantillas {
				out = append(out, resumo{p.ID, p.Nome, p.Descricion, p.ID == est.Activa, p.DeFabrica, p.Motor,
					p.Variables, strings.TrimSpace(p.Portada) != "", strings.TrimSpace(p.Sinatura) != "", p.Imaxes})
			}
			return map[string]any{"plantillas": out, "activa": est.Activa}, nil
		},
	},
	{
		def: IAFerramenta{
			Nome: "plantilla_ler", Tipo: tipoLectura, Onde: "backend",
			Descricion: "Le unha plantilla completa: LaTeX da maqueta, portada, sinatura, introdución, Markdown, motor e variables.",
			Parametros: esqObx(map[string]any{"id": esqTxt("Id da plantilla (de plantillas_listar).")}, "id"),
		},
		exec: func(a *App, args map[string]any) (any, error) {
			return plantillaPorID(a, argTxt(args, "id"))
		},
	},
	{
		def: IAFerramenta{
			Nome: "plantilla_gardar", Tipo: tipoConfirmar, Onde: "backend",
			Descricion: "Crea unha plantilla nova (sen id) ou modifica unha existente (con id; só se cambian os campos que indiques). O usuario ten que confirmar. Despois proba con plantilla_comprobar.",
			Parametros: esqPlantilla(map[string]any{"id": esqTxt("Id da plantilla a modificar; baleiro para crear unha nova.")}),
		},
		exec: func(a *App, args map[string]any) (any, error) {
			p, err := plantillaDeArgs(a, args)
			if err != nil {
				return nil, err
			}
			gardada, err := a.GardarPlantilla(p)
			if err != nil {
				return nil, err
			}
			res := map[string]any{"gardada": true, "id": gardada.ID, "nome": gardada.Nome}
			if id := strings.TrimSpace(argTxt(args, "id")); id != "" && id != gardada.ID {
				// Editar unha plantilla de exemplo non a pisa: crea unha copia propia.
				res["nota"] = "a plantilla orixinal é de exemplo e queda intacta: os cambios gardáronse nunha copia co id novo indicado (e pasa a ser a activa se a orixinal o era)"
			}
			return res, nil
		},
	},
	{
		def: IAFerramenta{
			Nome: "plantilla_comprobar", Tipo: tipoLectura, Onde: "backend",
			Descricion: "Compila unha plantilla cun curso de exemplo para ver se funciona, sen gardala. Pasa o id dunha existente (e opcionalmente campos para probar cambios) ou os campos completos dunha nova. Devolve se compilou, os avisos e, se fallou, o erro de LaTeX.",
			Parametros: esqPlantilla(map[string]any{
				"id":        esqTxt("Id da plantilla base (opcional)."),
				"documento": esqEnum("Documento de exemplo sobre o que se proba: 'cronograma' (por defecto) ou 'actas'.", "cronograma", "actas"),
			}),
		},
		exec: func(a *App, args map[string]any) (any, error) {
			p, err := plantillaDeArgs(a, args)
			if err != nil {
				return nil, err
			}
			p.Textos = normalizarTextosDoc(p.Textos)
			r, err := a.PrevisualizarPlantillaDe(p, documentoDeMostra(args))
			if err != nil {
				return nil, fmt.Errorf("non compila: %s", finalDe(err.Error(), 3000))
			}
			return map[string]any{"compila": true, "paxinas": len(r.PageImages), "avisos": r.Warnings}, nil
		},
	},
	{
		def: IAFerramenta{
			Nome: "plantilla_duplicar", Tipo: tipoEscritura, Onde: "backend",
			Descricion: "Fai unha copia dunha plantilla (para probar cambios sen perder a orixinal). Devolve o id da copia.",
			Parametros: esqObx(map[string]any{"id": esqTxt("Id da plantilla.")}, "id"),
		},
		exec: func(a *App, args map[string]any) (any, error) {
			p, err := a.DuplicarPlantilla(argTxt(args, "id"))
			if err != nil {
				return nil, err
			}
			return map[string]any{"id": p.ID, "nome": p.Nome}, nil
		},
	},
	{
		def: IAFerramenta{
			Nome: "plantilla_eliminar", Tipo: tipoConfirmar, Onde: "backend",
			Descricion: "Elimina unha plantilla da biblioteca. Non se pode desfacer; o usuario ten que confirmar.",
			Parametros: esqObx(map[string]any{"id": esqTxt("Id da plantilla.")}, "id"),
		},
		exec: func(a *App, args map[string]any) (any, error) {
			if err := a.EliminarPlantilla(argTxt(args, "id")); err != nil {
				return nil, err
			}
			return "plantilla eliminada", nil
		},
	},
	{
		def: IAFerramenta{
			Nome: "opcions_ler", Tipo: tipoLectura, Onde: "backend",
			Descricion: "Le as Opcións do programa: idioma, compilador LaTeX, xerar ao gardar, botóns de exportación visibles e configuración da IA (sen a clave).",
		},
		exec: func(a *App, _ map[string]any) (any, error) { return opcionsVisibles(a.settings), nil },
	},
	{
		def: IAFerramenta{
			Nome: "opcions_cambiar", Tipo: tipoEscritura, Onde: "backend",
			Descricion: "Cambia Opcións do programa (só as que indiques). A clave da API e o provedor de IA non se poden cambiar dende aquí.",
			Parametros: esqObx(map[string]any{
				"idioma":            esqEnum("Idioma da interface.", "gl", "es", "en", "pt"),
				"latexEngine":       esqEnum("Compilador LaTeX ('' = pdflatex).", "pdflatex", "xelatex", "lualatex"),
				"xerarAoGardar":     esqBool("Xerar tamén o informe ao gardar."),
				"exportarPdf":       esqBool("Amosar o botón PDF."),
				"exportarTex":       esqBool("Amosar o botón TeX."),
				"exportarMarkdown":  esqBool("Amosar o botón MD."),
				"exportarDocx":      esqBool("Amosar o botón DOCX."),
				"exportarOdt":       esqBool("Amosar o botón ODT."),
				"iaModel":           esqTxt("Nome do modelo de IA."),
				"iaMaxTokens":       esqNum("Límite de tokens por resposta (0 = o do provedor)."),
				"iaTimeoutSegundos": esqNum("Tempo máximo de espera por cada chamada á IA (mínimo 30)."),
			}),
		},
		exec: func(a *App, args map[string]any) (any, error) { return cambiarOpcions(a, args) },
	},
	{
		def: IAFerramenta{
			Nome: "sistema_estado", Tipo: tipoLectura, Onde: "backend",
			Descricion: "Comproba se o equipo ten o necesario para compilar o PDF (motor LaTeX e pdftoppm) e devolve datos do sistema. Útil se xerar o informe falla.",
		},
		exec: func(a *App, _ map[string]any) (any, error) {
			return map[string]any{"latex": a.CheckLatexDeps(), "sistema": runtime.GOOS}, nil
		},
	},
	{
		def: IAFerramenta{
			Nome: "web_buscar", Tipo: tipoLectura, Onde: "backend",
			Descricion: "Busca na web (p.ex. lexislación educativa: 'DOG decreto orientación educativa Galicia'). Devolve título, enderezo e un resumo dos primeiros resultados. Le a fonte con web_consultar.",
			Parametros: esqObx(map[string]any{
				"consulta":   esqTxt("O que buscar."),
				"resultados": esqNum("Cantos resultados (1-10, por defecto 8)."),
			}, "consulta"),
		},
		exec: func(a *App, args map[string]any) (any, error) {
			return webBuscar(argTxt(args, "consulta"), argInt(args, "resultados"))
		},
	},
	{
		def: IAFerramenta{
			Nome: "web_consultar", Tipo: tipoLectura, Onde: "backend",
			Descricion: "Le o texto dunha páxina web pública ou dun PDF (BOE, DOG, edu.xunta.gal...). Para documentos longos devolve unha parte; usa 'inicio' para seguir lendo.",
			Parametros: esqObx(map[string]any{
				"url":           esqTxt("Enderezo http(s) completo."),
				"inicio":        esqNum("Carácter desde o que ler (por defecto 0)."),
				"maxCaracteres": esqNum("Cantos caracteres devolver (por defecto 12000, máximo 40000)."),
			}, "url"),
		},
		exec: func(a *App, args map[string]any) (any, error) {
			return webConsultar(argTxt(args, "url"), argInt(args, "inicio"), argInt(args, "maxCaracteres"))
		},
	},
}

// ---------- Axudas das ferramentas ----------

func argTxt(m map[string]any, k string) string {
	if v, ok := m[k]; ok {
		if s, ok := v.(string); ok {
			return s
		}
		if v != nil {
			return fmt.Sprint(v)
		}
	}
	return ""
}

func argInt(m map[string]any, k string) int {
	switch v := m[k].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case string:
		var n int
		fmt.Sscanf(strings.TrimSpace(v), "%d", &n)
		return n
	}
	return 0
}

// finalDe devolve os últimos n caracteres de s (nos erros de LaTeX, o que
// importa está ao final).
func finalDe(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return "…" + string(r[len(r)-n:])
}

func plantillaPorID(a *App, id string) (PlantillaLatex, error) {
	if strings.TrimSpace(id) == "" {
		return PlantillaLatex{}, fmt.Errorf("falta o id da plantilla")
	}
	est, err := a.ListarPlantillas()
	if err != nil {
		return PlantillaLatex{}, err
	}
	for _, p := range est.Plantillas {
		if p.ID == id {
			return p, nil
		}
	}
	return PlantillaLatex{}, fmt.Errorf("non existe ningunha plantilla co id %q (usa plantillas_listar)", id)
}

// plantillaDeArgs constrúe unha plantilla a partir dos argumentos: parte da
// existente (se hai id) e pisa só os campos indicados.
func plantillaDeArgs(a *App, args map[string]any) (PlantillaLatex, error) {
	var p PlantillaLatex
	if id := strings.TrimSpace(argTxt(args, "id")); id != "" {
		var err error
		if p, err = plantillaPorID(a, id); err != nil {
			return p, err
		}
	}
	for campo, destino := range map[string]*string{
		"nome": &p.Nome, "descricion": &p.Descricion, "latex": &p.Latex, "portada": &p.Portada,
		"sinatura": &p.Sinatura, "introducion": &p.Introducion, "markdown": &p.Markdown, "motor": &p.Motor,
		"estilo": &p.Estilo,
	} {
		if v, ok := args[campo]; ok {
			if s, ok := v.(string); ok {
				*destino = s
			}
		}
	}
	if v, ok := args["textos"]; ok && v != nil {
		// Só se cambian os indicados: partimos dos actuais e un baleiro quita o texto.
		novos := map[string]string{}
		for k, val := range p.Textos {
			novos[k] = val
		}
		b, _ := json.Marshal(v)
		var dados map[string]string
		if err := json.Unmarshal(b, &dados); err != nil {
			return p, fmt.Errorf("o campo textos ten que ser un obxecto clave -> texto: %w", err)
		}
		for k, val := range dados {
			clave := strings.ToUpper(strings.TrimSpace(k))
			if _, ok := textosDocPorClave[clave]; !ok {
				return p, fmt.Errorf("o texto %q non existe. Claves: %s", k, clavesTextosDoc())
			}
			if strings.TrimSpace(val) == "" {
				delete(novos, clave)
			} else {
				novos[clave] = val
			}
		}
		p.Textos = normalizarTextosDoc(novos)
	}
	if v, ok := args["ofimatica"]; ok && v != nil {
		// Só se pisan os campos indicados: partimos do bloque actual.
		b, _ := json.Marshal(v)
		if err := json.Unmarshal(b, &p.Ofimatica); err != nil {
			return p, fmt.Errorf("o campo ofimatica non ten o formato esperado: %w", err)
		}
	}
	if v, ok := args["variables"]; ok && v != nil {
		b, _ := json.Marshal(v)
		var vars []PlantillaVar
		if err := json.Unmarshal(b, &vars); err != nil {
			return p, fmt.Errorf("as variables non teñen o formato esperado: %w", err)
		}
		p.Variables = vars
	}
	return p, nil
}

// opcionsVisibles é Settings SEN a clave da API (nin o enderezo do servidor).
func opcionsVisibles(s Settings) map[string]any {
	return map[string]any{
		"idioma":            s.Idioma,
		"latexEngine":       s.LatexEngine,
		"xerarAoGardar":     s.XerarAoGardarResolto(),
		"exportarPdf":       s.ExportarPDFResolto(),
		"exportarTex":       s.ExportarTexResolto(),
		"exportarMarkdown":  s.ExportarMarkdownResolto(),
		"exportarDocx":      s.ExportarDocx,
		"exportarOdt":       s.ExportarOdt,
		"iaProvedor":        s.IAProvedor,
		"iaPreset":          s.IAPreset,
		"iaModel":           s.IAModel,
		"iaMaxTokens":       s.IAMaxTokens,
		"iaTimeoutSegundos": s.IATimeoutSegundos,
		"iaTenClave":        strings.TrimSpace(s.IAAPIKey) != "",
	}
}

func argBool(v any) (bool, bool) {
	switch x := v.(type) {
	case bool:
		return x, true
	case string:
		switch strings.ToLower(strings.TrimSpace(x)) {
		case "true", "si", "sí", "1":
			return true, true
		case "false", "non", "0":
			return false, true
		}
	}
	return false, false
}

// cambiarOpcions aplica só as Opcións permitidas. A clave, o provedor e o
// enderezo de IA quedan fóra a propósito (ver a nota de seguridade en
// asistente.go).
func cambiarOpcions(a *App, args map[string]any) (any, error) {
	s := a.settings
	cambios := []string{}
	for k, v := range args {
		switch k {
		case "idioma":
			x := argTxt(args, k)
			if x != "gl" && x != "es" && x != "en" && x != "pt" {
				return nil, fmt.Errorf("idioma non válido: %q (gl, es, en ou pt)", x)
			}
			s.Idioma = x
		case "latexEngine":
			x := argTxt(args, k)
			if x != "" && x != "pdflatex" && x != "xelatex" && x != "lualatex" {
				return nil, fmt.Errorf("compilador non válido: %q", x)
			}
			s.LatexEngine = x
		case "xerarAoGardar", "exportarPdf", "exportarTex", "exportarMarkdown", "exportarDocx", "exportarOdt":
			b, ok := argBool(v)
			if !ok {
				return nil, fmt.Errorf("%s ten que ser true ou false", k)
			}
			switch k {
			case "xerarAoGardar":
				s.XerarAoGardar = &b
			case "exportarPdf":
				s.ExportarPDF = &b
			case "exportarTex":
				s.ExportarTex = &b
			case "exportarMarkdown":
				s.ExportarMarkdown = &b
			case "exportarDocx":
				s.ExportarDocx = b
			case "exportarOdt":
				s.ExportarOdt = b
			}
		case "iaModel":
			x := strings.TrimSpace(argTxt(args, k))
			if x == "" {
				return nil, fmt.Errorf("o modelo de IA non pode quedar baleiro")
			}
			s.IAModel = x
		case "iaMaxTokens":
			n := argInt(args, k)
			if n < 0 {
				return nil, fmt.Errorf("iaMaxTokens non pode ser negativo")
			}
			s.IAMaxTokens = n
		case "iaTimeoutSegundos":
			n := argInt(args, k)
			if n < 0 {
				return nil, fmt.Errorf("iaTimeoutSegundos non pode ser negativo")
			}
			s.IATimeoutSegundos = n
		default:
			return nil, fmt.Errorf("a opción %q non se pode cambiar dende o asistente", k)
		}
		cambios = append(cambios, k)
	}
	if len(cambios) == 0 {
		return nil, fmt.Errorf("non indicaches ningunha opción a cambiar")
	}
	if err := a.SaveSettings(s); err != nil {
		return nil, err
	}
	return map[string]any{"cambiadas": cambios, "opcions": opcionsVisibles(s)}, nil
}
