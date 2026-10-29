package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Settings persiste as opcións do usuario entre sesións (diálogo Opcións,
// frontend/src/main.js). Vai en <config>/lua/settings.json. Un ficheiro
// gardado por unha versión anterior (con campos que xa non existen) le sen
// problema: json ignora os campos descoñecidos, e desaparecen ao volver
// gardar.
type Settings struct {
	// IAProvedor/IAAPIKey/IAModel/IABaseURL configuran o asistente de IA
	// opcional (xerar plantillas a partir dun documento modelo, ver
	// plantillas_ia.go e ia_client.go) - calquera provedor que fale un dos
	// tres formatos de fío soportados (Gemini nativo, Anthropic nativo, ou
	// "compatible con OpenAI" - que cobre ChatGPT e moitos outros, ver
	// chamarIA). IAAPIKey baleira = asistente desactivado, o resto de Lua
	// funciona igual sen el.
	IAProvedor string `json:"iaProvedor"` // "gemini" | "anthropic" | "openai"
	IAAPIKey   string `json:"iaApiKey"`
	IAModel    string `json:"iaModel"`
	// IABaseURL só fai falla para un provedor "compatible con OpenAI" que
	// non sexa a propia OpenAI (Qwen, Perplexity...) - baleiro = enderezo
	// oficial do provedor (ver ia.BaseURLPorDefecto en edicion/ia). O selector do frontend xa
	// o prerreche cos presets coñecidos, o profesorado non ten por que velo.
	IABaseURL string `json:"iaBaseUrl"`
	// IAPreset non o le ningunha chamada á IA - é só o valor do selector do
	// frontend (que nome de provedor amosar), gardado á parte de
	// IAProvedor/IABaseURL para poder redebuxar exactamente o mesmo preset
	// ao reabrir Opcións (ex. distinguir "Qwen" de "Outro" aínda que os dous
	// resolvan a IAProvedor="openai" cun IABaseURL parecido).
	IAPreset string `json:"iaPreset"`
	// IAMaxTokens limita canto pode chegar a escribir a IA nunha soa
	// resposta - 0/sen configurar deixa o comportamento de sempre, distinto
	// por provedor (ver IAMaxTokensResolto/IAMaxTokensAnthropicResolto).
	// Subilo require que o MODELO escollido admita ese límite: Lua non o
	// comproba, un valor demasiado alto para un modelo pequeno faría fallar
	// a chamada coa súa propia mensaxe de erro do provedor.
	IAMaxTokens int `json:"iaMaxTokens"`
	// IATimeoutSegundos limita canto agarda Lua por CADA chamada á IA antes
	// de dala por perdida. 0/sen configurar -> 240s; valores por debaixo de
	// 30 trátanse coma sen configurar (ver IATimeoutResolto).
	IATimeoutSegundos int `json:"iaTimeoutSegundos"`
	// IABovedas liga cada tipo de documento (cronograma, actas, plantilla,
	// formulario) á ruta da súa bóveda de Obsidian (ver boveda.go).
	// Xestiónase con BovedaConectar, non dende Opcións.
	IABovedas map[string]string `json:"iaBovedas,omitempty"`
	// BuxoDesactivado: o usuario desactivou o módulo buxo en Opcións (por
	// defecto está activo e Lua o descarga e actualiza ao arrincar). Xestiónase
	// con BuxoActivar.
	BuxoDesactivado bool `json:"buxoDesactivado,omitempty"`
	// Instalacion identifica esta instalación de Lúa (aleatorio, non secreto):
	// recoñece os documentos .json que saíron de aquí e volven (trasladar.go).
	Instalacion string `json:"instalacion,omitempty"`
	// LatexEngine escolle o compilador: "" ou "pdflatex" (por defecto),
	// "xelatex" ou "lualatex" (soporte Unicode/fontes do sistema nativo,
	// útil para símbolos especiais coma letras gregas ou fontes propias).
	LatexEngine string `json:"latexEngine"`
	// Idioma escolle a lingua da interface: "" ou "gl" (galego, por
	// defecto), "es", "en" ou "pt". Os dicionarios en si viven en
	// lua/idiomas/*.json, importados directamente polo frontend
	// (frontend/src/main.js) - este campo só persiste a escolla do
	// profesorado entre sesións, o propio Go nunca traduce nada.
	Idioma string `json:"idioma"`
	// XerarAoGardar, cando activo (por defecto), fai que Gardar/Gardar como
	// (Ctrl+S) tamén recompile o informe (mesmo que premer Xerar/Ctrl+X) -
	// así a vista previa nunca queda desincronizada do que se acaba de
	// gardar. Un *bool (non un bool a secas) porque o valor por defecto é
	// true: un settings.json vello (ou un profesor que nunca abriu Opcións)
	// non ten esta chave, e un bool normal léraa como false. Ver
	// XerarAoGardarResolto.
	XerarAoGardar *bool `json:"xerarAoGardar,omitempty"`
	// ExportarPDF/ExportarTex/ExportarMarkdown/ExportarDocx/ExportarOdt
	// deciden que botóns de exportación amosa a barra da previsualización -
	// quen só use PDF pode agochar os demais. PDF/Tex/Markdown son *bool
	// (mesmo motivo ca XerarAoGardar: por defecto ACTIVADOS, así que "sen
	// configurar" (nil) ten que distinguirse de "false" explícito).
	// Docx/Odt son bool normal porque por defecto van DESACTIVADOS (quen só
	// entrega PDF non os precisa) - o valor cero de Go (false) xa é o defecto
	// correcto.
	ExportarPDF      *bool `json:"exportarPdf,omitempty"`
	ExportarTex      *bool `json:"exportarTex,omitempty"`
	ExportarMarkdown *bool `json:"exportarMarkdown,omitempty"`
	ExportarDocx     bool  `json:"exportarDocx"`
	ExportarOdt      bool  `json:"exportarOdt"`

	// MaximaPath é a ruta do executable de Maxima (o motor de cálculo dos
	// problemas, coma en Yang). Baleira = buscalo (ver findMaxima). Sen Maxima
	// os problemas calcúlanse en Go: números, rangos e fórmulas simples.
	MaximaPath string `json:"maximaPath,omitempty"`
	// MaximaDecimais: cifras decimais dos resultados de Maxima (0 = as de Maxima).
	MaximaDecimais int `json:"maximaDecimais,omitempty"`

	// FormulasEstilo e FormulasEscala: como saen as fórmulas nos documentos
	// (formulas_estilo.go). Estilo "compactas" (ou baleiro) ou "grandes";
	// escala en % respecto ao texto, 100–150 (0 = 100).
	FormulasEstilo string `json:"formulasEstilo,omitempty"`
	FormulasEscala int    `json:"formulasEscala,omitempty"`
}

// XerarAoGardarResolto devolve XerarAoGardar coa regra de "sen configurar
// (nil) -> true" xa aplicada.
func (s Settings) XerarAoGardarResolto() bool {
	return s.XerarAoGardar == nil || *s.XerarAoGardar
}

// ExportarPDFResolto/ExportarTexResolto/ExportarMarkdownResolto aplican a
// mesma regra de "sen configurar (nil) -> true" ca XerarAoGardarResolto - un
// settings.json vello, ou quen nunca abriu Opcións, non ten estas chaves e
// os botóns de exportación deben seguir visibles coma sempre.
func (s Settings) ExportarPDFResolto() bool {
	return s.ExportarPDF == nil || *s.ExportarPDF
}

func (s Settings) ExportarTexResolto() bool {
	return s.ExportarTex == nil || *s.ExportarTex
}

func (s Settings) ExportarMarkdownResolto() bool {
	return s.ExportarMarkdown == nil || *s.ExportarMarkdown
}

// IAMaxTokensResolto devolve IAMaxTokens coa regra de "<=0 -> 0" (omitir o
// parámetro por completo) - para Gemini e «compatible con OpenAI»
// (edicion/ia, a través de configDe en ia_conversa.go), cuxas APIs aceptan omitir o límite de resposta e deixar
// que o decida o propio modelo.
func (s Settings) IAMaxTokensResolto() int {
	if s.IAMaxTokens <= 0 {
		return 0
	}
	return s.IAMaxTokens
}

// IATimeoutResolto devolve IATimeoutSegundos coa regra "< 30s -> 240s"
// aplicada (0/sen configurar e calquera valor demasiado baixo caen no
// mesmo), listo para http.Client.Timeout en chamarIA.
func (s Settings) IATimeoutResolto() time.Duration {
	if s.IATimeoutSegundos < 30 {
		return 240 * time.Second
	}
	return time.Duration(s.IATimeoutSegundos) * time.Second
}

// IAMaxTokensAnthropicResolto devolve IAMaxTokens coa regra de "<=0 ->
// 8192" - para Anthropic (configDe, ia_conversa.go), cuxa API esixe SEMPRE un
// valor numérico en max_tokens (non se pode omitir coma en Gemini/OpenAI).
func (s Settings) IAMaxTokensAnthropicResolto() int {
	if s.IAMaxTokens <= 0 {
		return 8192
	}
	return s.IAMaxTokens
}

func settingsPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "lua")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "settings.json"), nil
}

func loadSettings() Settings {
	var s Settings
	path, err := settingsPath()
	if err != nil {
		return s
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return s
	}
	if err := json.Unmarshal(b, &s); err != nil {
		s = Settings{}
	}
	migrarAxustesIAVellos(b, &s)
	return s
}

// migrarAxustesIAVellos le un settings.json gardado por unha versión
// anterior (só tiña geminiApiKey/geminiModel, sempre Gemini, sen os campos
// IA* xenéricos de enriba) e completa eses campos novos se aínda están
// baleiros - así quen xa tiña a IA configurada non ten que volver escribir a
// clave despois de actualizar Lua. Non toca nada se xa hai IAAPIKey.
func migrarAxustesIAVellos(b []byte, s *Settings) {
	if s.IAAPIKey != "" {
		return
	}
	var vello struct {
		GeminiAPIKey string `json:"geminiApiKey"`
		GeminiModel  string `json:"geminiModel"`
	}
	if json.Unmarshal(b, &vello) != nil || vello.GeminiAPIKey == "" {
		return
	}
	s.IAProvedor = string(ProvedorGemini)
	s.IAPreset = string(ProvedorGemini)
	s.IAAPIKey = vello.GeminiAPIKey
	s.IAModel = vello.GeminiModel
}

func saveSettingsToDisk(s Settings) error {
	path, err := settingsPath()
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// GetSettings devolve as opcións activas, para o diálogo de Opcións.
func (a *App) GetSettings() Settings {
	return a.settings
}

// SaveSettings garda as opcións novas e aplícaas ao momento.
func (a *App) SaveSettings(s Settings) error {
	if s.IABovedas == nil { // Opcións non as xestiona: consérvanse
		s.IABovedas = a.settings.IABovedas
	}
	s.BuxoDesactivado = a.settings.BuxoDesactivado // non o xestiona o diálogo de Opcións
	s.Instalacion = a.settings.Instalacion         // nin o identificador da instalación
	cambiaMaxima := s.MaximaPath != a.settings.MaximaPath || s.MaximaDecimais != a.settings.MaximaDecimais
	a.settings = s
	a.aplicarIdiomaGo()
	if cambiaMaxima {
		a.pecharMotorProblemas()
	}
	return saveSettingsToDisk(s)
}
