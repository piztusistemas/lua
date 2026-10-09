package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Cliente de IA xenérico: fala con calquera provedor que use un destes tres
// formatos de fío (case a inmensa maioría do mercado):
//   - "gemini": API nativa de Google Gemini (systemInstruction + contents).
//   - "anthropic": API nativa de Claude (x-api-key + system + messages).
//   - "openai": formato "chat completions" de OpenAI - o que tamén falan de
//     fábrica moitos outros provedores en modo "compatible con OpenAI"
//     (Qwen/DashScope, Perplexity, Groq, Mistral, DeepSeek, Ollama en
//     local...), así que este mesmo camiño cobre ChatGPT E calquera destes
//     só cambiando o enderezo do servidor (ver baseURLPorDefecto/Settings.
//     IABaseURL) - por iso "xenérico/universal": non hai que escribir un
//     adaptador novo por cada marca, só para os poucos formatos de fío
//     realmente distintos que existen.
//
// O profesorado escolle un provedor por nome nun selector (Opcións, ver
// frontend/index.html) que xa pon o formato de fío e o enderezo por
// defecto correctos - nunca ten que sabelo nin escribilo á man.
type ProvedorIA string

const (
	ProvedorGemini    ProvedorIA = "gemini"
	ProvedorAnthropic ProvedorIA = "anthropic"
	ProvedorOpenAI    ProvedorIA = "openai" // tamén calquera "compatible con OpenAI"
)

// iaHTTPClient é o cliente por defecto (fallback dos tests e de calquera
// chamada sen cliente propio). chamarIA constrúe un cliente por chamada co
// timeout de Settings.IATimeoutResolto (Opcións).
var iaHTTPClient = &http.Client{Timeout: 90 * time.Second}

// baseURLPorDefecto devolve o enderezo oficial do provedor cando
// Settings.IABaseURL queda baleiro - só fai falla escribir un enderezo á
// man para un provedor "compatible con OpenAI" que non sexa a propia
// OpenAI (Qwen, Perplexity...), e aínda así o selector do frontend xa trae
// eses enderezos coma preset.
func baseURLPorDefecto(provedor ProvedorIA) string {
	switch provedor {
	case ProvedorGemini:
		return "https://generativelanguage.googleapis.com/v1beta"
	case ProvedorAnthropic:
		return "https://api.anthropic.com/v1"
	default:
		return "https://api.openai.com/v1"
	}
}

// chamarIA é o punto de entrada único que usa XerarPlantillaIA
// (plantillas_ia.go): decide o formato de fío polo
// provedor configurado e devolve o texto de resposta xa limpo de valados
// Markdown. `s` é a configuración completa (non só as claves) para que
// engadir un catro provedor no futuro só toque este switch, non cada
// chamador.
func chamarIA(s Settings, sistema, peticion string) (string, error) {
	// Timeout por chamada (Opcións -> "Tempo máximo de espera pola IA").
	provedor, baseURL, cliente, err := resolverIA(s)
	if err != nil {
		return "", err
	}

	// IAMaxTokens (Opcións): límite de tokens de resposta configurable pola
	// usuaria - 0/sen configurar deixa cada provedor no seu comportamento de
	// sempre (ver Settings.IAMaxTokens/IAMaxTokensAnthropicResolto,
	// settings.go). Anthropic esixe sempre un número en max_tokens
	// (herdando os 8192 antes fixos coma valor por defecto); Gemini/
	// "compatible con OpenAI" simplemente OMITEN o parámetro cando é 0,
	// coma antes desta opción existir.
	switch provedor {
	case ProvedorAnthropic:
		return chamarAnthropic(baseURL, s.IAAPIKey, s.IAModel, sistema, peticion, s.IAMaxTokensAnthropicResolto(), cliente)
	case ProvedorGemini:
		return chamarGemini(baseURL, s.IAAPIKey, s.IAModel, sistema, peticion, s.IAMaxTokensResolto(), cliente)
	default:
		return chamarOpenAICompatible(baseURL, s.IAAPIKey, s.IAModel, sistema, peticion, s.IAMaxTokensResolto(), cliente)
	}
}

// ── Gemini (nativo) ──────────────────────────────────────────────────────

type geminiPeticion struct {
	SystemInstruction *geminiContido          `json:"systemInstruction,omitempty"`
	Contents          []geminiContido         `json:"contents"`
	GenerationConfig  *geminiGenerationConfig `json:"generationConfig,omitempty"`
}

// geminiGenerationConfig só leva MaxOutputTokens por agora - Lua non
// expón ningún outro parámetro de xeración (temperature, topP...) na UI,
// así que non fai falla modelar máis ca isto.
type geminiGenerationConfig struct {
	MaxOutputTokens int `json:"maxOutputTokens,omitempty"`
}

type geminiContido struct {
	Role  string        `json:"role,omitempty"`
	Parts []geminiParte `json:"parts"`
}

type geminiParte struct {
	Text string `json:"text,omitempty"`
}

type geminiResposta struct {
	Candidates []struct {
		Content geminiContido `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// chamarGemini envía `sistema` coma systemInstruction e `peticion` coma único
// contido do usuario. Idéntica xestión de erros ca en Xesta (mesmas
// mensaxes, mesmo comportamento comprobado empiricamente contra a API real:
// systemInstruction precisa Role:"user" ou a API devolve 400 sen máis
// detalle). maxTokens <= 0 omite generationConfig por completo (deixa o
// límite de resposta por defecto do propio modelo, comportamento previo a
// que isto fose configurable - ver Settings.IAMaxTokens).
func chamarGemini(baseURL, apiKey, model, sistema, peticion string, maxTokens int, cliente *http.Client) (string, error) {
	if cliente == nil {
		cliente = iaHTTPClient
	}
	corpo := geminiPeticion{
		SystemInstruction: &geminiContido{Role: "user", Parts: []geminiParte{{Text: sistema}}},
		Contents: []geminiContido{
			{Role: "user", Parts: []geminiParte{{Text: peticion}}},
		},
	}
	if maxTokens > 0 {
		corpo.GenerationConfig = &geminiGenerationConfig{MaxOutputTokens: maxTokens}
	}
	dados, err := json.Marshal(corpo)
	if err != nil {
		return "", err
	}

	url := baseURL + "/models/" + model + ":generateContent"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(dados))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", apiKey)

	resp, err := cliente.Do(req)
	if err != nil {
		return "", fmt.Errorf("non se puido contactar coa IA: %w", err)
	}
	defer resp.Body.Close()

	corpoResposta, _ := io.ReadAll(resp.Body)
	var r geminiResposta
	if err := json.Unmarshal(corpoResposta, &r); err != nil {
		return "", fmt.Errorf("resposta non válida da IA (HTTP %d)", resp.StatusCode)
	}
	if r.Error != nil {
		return "", fmt.Errorf("erro da IA: %s", r.Error.Message)
	}
	if resp.StatusCode != http.StatusOK || len(r.Candidates) == 0 || len(r.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("a IA non devolveu resultado (HTTP %d)", resp.StatusCode)
	}

	return limparValadosMarkdown(r.Candidates[0].Content.Parts[0].Text), nil
}

// ── Anthropic (nativo, Claude) ───────────────────────────────────────────

type anthropicPeticion struct {
	Model     string                  `json:"model"`
	MaxTokens int                     `json:"max_tokens"`
	System    []anthropicSystemBloque `json:"system,omitempty"`
	Messages  []anthropicMensaxe      `json:"messages"`
}

// anthropicSystemBloque é un anaco do "system" da API de Claude. Vai en
// slice (en vez de string simple) para poder marcar CacheControl: o
// "sistema" que mandamos (chuletaLua/chuletaLuaPara + instrución) é
// idéntico en moitas chamadas seguidas (mesmo tipo de anaco, mesmo
// provedor) - marcándoo coma "ephemeral" Claude cachea eses tokens no seu
// lado e cobra ~un 10% neles nas chamadas seguintes en vez do prezo enteiro,
// sen cambiar en nada a resposta (non é compresión con perdas, é caché
// exacta). Gemini/OpenAI non necesitan isto: cachean prefixos repetidos
// automaticamente sen que o cliente marque nada.
type anthropicSystemBloque struct {
	Type         string                 `json:"type"`
	Text         string                 `json:"text"`
	CacheControl *anthropicCacheControl `json:"cache_control,omitempty"`
}

type anthropicCacheControl struct {
	Type string `json:"type"`
}

type anthropicMensaxe struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResposta struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// chamarAnthropic fala coa API de Messages de Claude (api.anthropic.com) -
// autenticación por cabeceira x-api-key (non Bearer) e cabeceira obrigatoria
// anthropic-version, a diferenza dos outros dous formatos. maxTokens chega
// xa resolto (Settings.IAMaxTokensAnthropicResolto) - a diferenza de
// Gemini/OpenAI, esta API esixe SEMPRE un valor numérico en max_tokens, non
// se pode omitir.
func chamarAnthropic(baseURL, apiKey, model, sistema, peticion string, maxTokens int, cliente *http.Client) (string, error) {
	if cliente == nil {
		cliente = iaHTTPClient
	}
	corpo := anthropicPeticion{
		Model:     model,
		MaxTokens: maxTokens,
		Messages:  []anthropicMensaxe{{Role: "user", Content: peticion}},
	}
	if sistema != "" {
		// CacheControl "ephemeral": chuletaLua/chuletaLuaPara + a
		// instrución que segue son idénticas en moitas chamadas seguidas
		// (mesmo tipo de anaco) - Claude cachea eses tokens no seu lado
		// (5 min de TTL por defecto) e cobra menos por eles nas chamadas
		// seguintes. Se sistema queda por baixo do mínimo cacheable do
		// modelo (~1024-2048 tokens), a API simplemente ignora a marca e
		// segue igual, sen erro.
		corpo.System = []anthropicSystemBloque{{
			Type:         "text",
			Text:         sistema,
			CacheControl: &anthropicCacheControl{Type: "ephemeral"},
		}}
	}
	dados, err := json.Marshal(corpo)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+"/messages", bytes.NewReader(dados))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := cliente.Do(req)
	if err != nil {
		return "", fmt.Errorf("non se puido contactar coa IA: %w", err)
	}
	defer resp.Body.Close()

	corpoResposta, _ := io.ReadAll(resp.Body)
	var r anthropicResposta
	if err := json.Unmarshal(corpoResposta, &r); err != nil {
		return "", fmt.Errorf("resposta non válida da IA (HTTP %d)", resp.StatusCode)
	}
	if r.Error != nil {
		return "", fmt.Errorf("erro da IA: %s", r.Error.Message)
	}
	if resp.StatusCode != http.StatusOK || len(r.Content) == 0 {
		return "", fmt.Errorf("a IA non devolveu resultado (HTTP %d)", resp.StatusCode)
	}

	var texto strings.Builder
	for _, bloque := range r.Content {
		if bloque.Type == "text" {
			texto.WriteString(bloque.Text)
		}
	}
	return limparValadosMarkdown(texto.String()), nil
}

// ── OpenAI / "compatible con OpenAI" (ChatGPT, Qwen, Perplexity, Groq...) ──

type openAIPeticion struct {
	Model     string          `json:"model"`
	Messages  []openAIMensaxe `json:"messages"`
	MaxTokens int             `json:"max_tokens,omitempty"`
}

type openAIMensaxe struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResposta struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// chamarOpenAICompatible fala o formato "chat completions" (POST
// {baseURL}/chat/completions, Authorization: Bearer <clave>) - o mesmo que
// falan de fábrica moitos outros provedores cando se lles pasa o enderezo
// axeitado (ver baseURLPorDefecto e os presets do selector no frontend).
// maxTokens <= 0 omite max_tokens por completo (omitempty, openAIPeticion) -
// deixa o límite de resposta por defecto do propio modelo, comportamento
// previo a que isto fose configurable (ver Settings.IAMaxTokens).
func chamarOpenAICompatible(baseURL, apiKey, model, sistema, peticion string, maxTokens int, cliente *http.Client) (string, error) {
	if cliente == nil {
		cliente = iaHTTPClient
	}
	corpo := openAIPeticion{
		Model: model,
		Messages: []openAIMensaxe{
			{Role: "system", Content: sistema},
			{Role: "user", Content: peticion},
		},
		MaxTokens: maxTokens,
	}
	dados, err := json.Marshal(corpo)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(dados))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := cliente.Do(req)
	if err != nil {
		return "", fmt.Errorf("non se puido contactar coa IA: %w", err)
	}
	defer resp.Body.Close()

	corpoResposta, _ := io.ReadAll(resp.Body)
	var r openAIResposta
	if err := json.Unmarshal(corpoResposta, &r); err != nil {
		return "", fmt.Errorf("resposta non válida da IA (HTTP %d)", resp.StatusCode)
	}
	if r.Error != nil {
		return "", fmt.Errorf("erro da IA: %s", r.Error.Message)
	}
	if resp.StatusCode != http.StatusOK || len(r.Choices) == 0 {
		return "", fmt.Errorf("a IA non devolveu resultado (HTTP %d)", resp.StatusCode)
	}

	return limparValadosMarkdown(r.Choices[0].Message.Content), nil
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
