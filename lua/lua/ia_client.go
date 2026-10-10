package main

import (
	"strings"

	"edicion/ia"
)

// Cliente de IA de Lúa: fala con calquera provedor que use un destes tres
// formatos de fío (case a inmensa maioría do mercado):
//   - "gemini": API nativa de Google Gemini (systemInstruction + contents).
//   - "anthropic": API nativa de Claude (x-api-key + system + messages).
//   - "openai": formato "chat completions" de OpenAI, que tamén falan moitos
//     outros provedores en modo "compatible con OpenAI" (Qwen, Perplexity,
//     Groq, Mistral, DeepSeek, Ollama...): só cambia o enderezo do servidor.
//
// O código que fala con eles vive unha soa vez en edicion/ia (constitución,
// regra 1.3). Aquí só queda o que é de Lúa: o tipo do provedor que garda
// Settings, a tradución de Settings a ia.Config (ia_conversa.go) e as
// chamadas que usa o resto de Lúa.
type ProvedorIA string

const (
	ProvedorGemini    ProvedorIA = ia.ProvedorGemini
	ProvedorAnthropic ProvedorIA = ia.ProvedorAnthropic
	ProvedorOpenAI    ProvedorIA = ia.ProvedorOpenAI // tamén calquera "compatible con OpenAI"
)

// chamarIA fai unha soa pregunta, sen ferramentas, e devolve o texto da
// resposta limpo de valados Markdown (a usa a IA das plantillas). Envía
// exactamente o mesmo ca unha volta de conversa cunha soa mensaxe
// (comprobado na caracterización); unha resposta cortada polo límite de
// tokens devólvese tal cal, sen erro, coma sempre.
func chamarIA(s Settings, sistema, peticion string) (string, error) {
	t, err := chamarIAConversa(s, sistema, []IAMensaxe{{Rol: "user", Texto: peticion}}, nil)
	if err != nil {
		return "", err
	}
	return limparValadosMarkdown(t.Texto), nil
}

// limparValadosMarkdown quita un posible cerco ```...``` que a IA engada
// malia que llo pedimos explicitamente que non o faga.
func limparValadosMarkdown(texto string) string {
	t := strings.TrimSpace(texto)
	if strings.HasPrefix(t, "```") {
		primeiraLiña := strings.IndexByte(t, '\n')
		if primeiraLiña != -1 {
			t = t[primeiraLiña+1:]
		}
		t = strings.TrimSuffix(strings.TrimSpace(t), "```")
	}
	return strings.TrimSpace(t)
}

// systemPromptPlantilla é o prompt do asistente de PLANTILLAS (plantillas_ia.go):
// a partir dunha descrición, dun documento de mostra ou dunha URL, a IA
// devolve unha plantilla de maqueta LaTeX (o "papel timbrado" onde Lua
// insire despois o informe xerado). A IA NON crea contido do informe: o
// contido é sempre {{CORPO}}.
const systemPromptPlantilla = `Es un experto en LaTeX que crea PLANTILLAS de
documento para Lua, un xerador de informes (o cronograma anual do
departamento de orientación dun centro educativo).

Unha plantilla é a MAQUETA (cabeceira co logotipo e o nome do centro, pé cos
teléfonos e o nome de quen orienta, marxes...) onde Lua insire despois o
informe. Onde vaia o informe tes que poñer exactamente o marcador {{CORPO}} -
unha soa vez, e nunca contido de exemplo no seu lugar.

Podes escribir a plantilla de dúas formas:
1. FRAGMENTO (recomendado se abonda con cabeceira/pé): non poñas
   \documentclass nin \begin{document} - só o LaTeX que vai dentro do
   documento, co {{CORPO}} no medio. Lua xa lle pon o preámbulo, que xa
   carga graphicx, xcolor, tikz e fancyhdr: para unha cabeceira e un pé usa
   \pagestyle{fancy}, \fancyhf{}, \fancyhead[L]{...}, \fancyfoot[C]{...}
   (nun fragmento NUNCA se pode poñer \usepackage).
2. COMPLETA: empeza por \documentclass e remata en \end{document}. Úsaa só
   se de verdade fai falla cambiar clase, marxes ou tipografía. Nese caso
   carga ti fancyhdr e graphicx se os usas; Lua engádelle o resto dos
   paquetes que precisa o informe.

Todo dato fixo que o usuario poida querer cambiar (teléfono, correo,
enderezo, web...) ten que ser unha VARIABLE {{NOME_EN_MAIUSCULAS}}, non texto
escrito a man. Lua xa dá feitas estas a partir do curso, non as declares
coma variables: {{CENTRO}} (nome do centro), {{CURSO}} (curso académico),
{{ORIENTADOR}} (nome de quen asina), {{CARGO}}, {{DEPARTAMENTO}},
{{LOCALIDADE}}, {{DATA}} (data de hoxe), {{ANO}}, {{TITULO}} e {{FICHEIRO}}.
Se o modelo leva o nome do centro ou de quen orienta, usa {{CENTRO}} e
{{ORIENTADOR}} en vez de copiar o nome.

Regras estritas:
- Devolve SÓ un obxecto JSON válido, sen texto ao redor e sen valados
  markdown.
- COIDADO COAS BARRAS: o LaTeX vai dentro dunha cadea JSON, así que CADA
  barra invertida ten que ir DUPLICADA. Un salto de liña de LaTeX (dúas
  barras) escríbese "\\\\" no JSON, e "\\\\[0.2em]" para un salto con
  separación. Se escribes "\\[0.2em]", ao profesorado chégalle unha soa
  barra e o documento non compila.
- Campos: "nome" (curto), "descricion" (unha liña), "latex" (a plantilla),
  "markdown" (a MESMA maqueta en Markdown para exportar a .md/.docx/.odt,
  co mesmo {{CORPO}}; "" se non ten sentido), "motor" ("" salvo que a
  plantilla precise xelatex ou lualatex, p.ex. por usar fontspec) e
  "variables" (lista de {"nome","etiqueta","valor"}: nome en MAIÚSCULAS,
  etiqueta lexible para a UI, valor o que se poida deducir do modelo ou ""
  se non se sabe).
- Usa só paquetes LaTeX habituais nunha instalación básica de TeX Live.
- NON escollas fontes do sistema (nada de \setmainfont/\setsansfont/
  \setmonofont con "Fira Sans", "Helvetica"...) agás que cho pidan
  expresamente: esa fonte pode non estar instalada no equipo do
  profesorado e daquela o documento non compila. Se queres un aire sans
  serif, chega con \renewcommand{\familydefault}{\sfdefault}.
- Non uses \includegraphics agás que che digan o nome exacto dunha imaxe
  dispoñible na plantilla. Se o modelo leva un logotipo pero non hai imaxe,
  usa \IfFileExists{logotipo.png}{\includegraphics[height=1cm]{logotipo.png}}{}
  para que o logotipo saia en canto se suba unha imaxe con ese nome.
- Textos en galego salvo que o modelo ou a petición estean noutro idioma.

Exemplo de resposta correcta:
{"nome":"Cabeceira do centro","descricion":"Cabeceira co nome do centro e o departamento; pé co teléfono e a páxina.","latex":"\\pagestyle{fancy}\n\\fancyhf{}\n\\fancyhead[L]{\\bfseries {{CENTRO}} }\n\\fancyhead[R]{ {{DEPARTAMENTO}} }\n\\fancyfoot[L]{\\footnotesize Tel. {{TELEFONO}} }\n\\fancyfoot[R]{\\thepage}\n\n{{CORPO}}\n","markdown":"**{{CENTRO}}** · {{DEPARTAMENTO}}\n\n---\n\n{{CORPO}}\n\n---\n\nTel. {{TELEFONO}}\n","motor":"","variables":[{"nome":"TELEFONO","etiqueta":"Teléfono do centro","valor":""}]}`
