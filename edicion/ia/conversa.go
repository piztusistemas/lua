package ia

// Cliente de IA CONVERSACIONAL con chamadas a ferramentas ("function
// calling"), para o asistente de Lua (asistente.go). Fala os mesmos tres
// formatos de fío ca chamarIA (ia_client.go: Gemini, Anthropic, "compatible
// con OpenAI"), pero en vez dunha soa pregunta/resposta leva unha conversa
// enteira: mensaxes do usuario, respostas da IA (con texto e/ou chamadas a
// ferramentas) e resultados das ferramentas.
//
// O bucle (pedir -> executar as ferramentas -> volver pedir...) vive no
// frontend (frontend/src/asistente/), porque a maioría das ferramentas tocan
// o curso en edición, que só existe alí. Este ficheiro fai UN turno: recibe a
// conversa ata agora e devolve o que respondeu a IA.
//
// Cada resposta da IA gárdase tamén "en bruto" (IAMensaxe.Bruto, o JSON do
// provedor) e devólvese tal cal ao pedir o turno seguinte: hai provedores que
// esixen recibir de volta campos que Lua nin le (as "sinaturas de
// pensamento" de Gemini, o razoamento de DeepSeek...) e sen eles rexeitan a
// conversa.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// IAChamada é unha petición da IA para executar unha ferramenta.
type IAChamada struct {
	ID   string `json:"id"`
	Nome string `json:"nome"`
	Args string `json:"args"` // JSON co obxecto de argumentos
}

// IAResultado é o que devolveu unha ferramenta.
type IAResultado struct {
	ChamadaID string `json:"chamadaId"`
	Nome      string `json:"nome"`
	Contido   string `json:"contido"`
	Erro      bool   `json:"erro"`
	// Imaxes son ids de adxuntos (imaxes) que a ferramenta lle quere amosar á
	// IA (adxunto_ver, plantilla_previsualizar...). Lua resólveas a imaxes
	// reais ao pedir cada turno (resolverAdxuntosMensaxes).
	Imaxes []string `json:"imaxes"`
	// ImaxesIA son esas imaxes xa resoltas para o provedor.
	ImaxesIA []Imaxe `json:"-"`
}

// IAMensaxe é unha mensaxe da conversa. Rol: "user", "assistant" ou "tool"
// (os resultados das ferramentas que pediu a mensaxe "assistant" anterior).
type IAMensaxe struct {
	Rol        string        `json:"rol"`
	Texto      string        `json:"texto"`
	Chamadas   []IAChamada   `json:"chamadas"`
	Resultados []IAResultado `json:"resultados"`
	// Bruto é a resposta orixinal do provedor (JSON), só nas mensaxes
	// "assistant"; Provedor di de que formato é. Se cambia o provedor entre
	// dous turnos, o bruto descártase e reconstrúese a partir de Texto e
	// Chamadas.
	Bruto    string `json:"bruto"`
	Provedor string `json:"provedor"`
	// Adxuntos son os ids dos ficheiros que o usuario xuntou a ESTA mensaxe
	// (adxuntos.go). As imaxes van á vista da IA; dos documentos vai o
	// contido se é curto, ou o aviso de que os lea con adxunto_ler.
	Adxuntos []string `json:"adxuntos"`
	// ImaxesIA: as imaxes que van con esta mensaxe.
	ImaxesIA []Imaxe `json:"-"`
}

// IAFerramenta describe unha ferramenta que a IA pode chamar (o equivalente
// dunha entrada de tools/list en MCP).
type IAFerramenta struct {
	Nome       string `json:"nome"`
	Descricion string `json:"descricion"`
	// Parametros é un JSON Schema (obxecto) cos argumentos.
	Parametros string `json:"parametros"`
	// Tipo: "lectura" (só mira), "escritura" (cambia algo que se pode
	// desfacer) ou "confirmar" (cambio que non se desfai: o frontend
	// pregunta antes).
	Tipo string `json:"tipo"`
	// Onde se executa: "frontend" (toca o curso ou a interface) ou "backend".
	Onde string `json:"onde"`
}

// IATurno é o que respondeu a IA nun turno.
type IATurno struct {
	Texto    string      `json:"texto"`
	Chamadas []IAChamada `json:"chamadas"`
	Bruto    string      `json:"bruto"`
	Provedor string      `json:"provedor"`
	// Truncado: a resposta cortouse polo límite de tokens.
	Truncado bool `json:"truncado"`
}

// postJSONIA fai unha petición POST con cabeceiras e devolve o corpo. Os
// erros de rede e os HTTP que non sexan 2xx sen corpo JSON útil saen cunha
// mensaxe comprensible.
func postJSONIA(cliente *http.Client, url string, cabeceiras map[string]string, corpo any) ([]byte, int, error) {
	if cliente == nil {
		cliente = clientePorDefecto
	}
	dados, err := json.Marshal(corpo)
	if err != nil {
		return nil, 0, err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(dados))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range cabeceiras {
		req.Header.Set(k, v)
	}
	resp, err := cliente.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("non se puido contactar coa IA: %w", err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return b, resp.StatusCode, nil
}

// argsOuBaleiro devolve "{}" se a IA mandou uns argumentos baleiros.
func argsOuBaleiro(s string) string {
	if strings.TrimSpace(s) == "" || strings.TrimSpace(s) == "null" {
		return "{}"
	}
	return s
}

// esquemaOuBaleiro devolve o JSON Schema dunha ferramenta, ou un obxecto
// baleiro válido se non ten parámetros.
func esquemaOuBaleiro(f IAFerramenta) json.RawMessage {
	if strings.TrimSpace(f.Parametros) == "" {
		return json.RawMessage(`{"type":"object","properties":{}}`)
	}
	return json.RawMessage(f.Parametros)
}

func esquemaTenPropiedades(f IAFerramenta) bool {
	var e struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	return json.Unmarshal([]byte(f.Parametros), &e) == nil && len(e.Properties) > 0
}

// ── OpenAI / compatible ─────────────────────────────────────────────────

type openAITurnoResposta struct {
	Choices []struct {
		Message      json.RawMessage `json:"message"`
		FinishReason string          `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type openAIMsgSaida struct {
	Content   *string `json:"content"`
	ToolCalls []struct {
		ID       string `json:"id"`
		Type     string `json:"type"`
		Function struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"function"`
	} `json:"tool_calls"`
	ReasoningContent *string `json:"reasoning_content"`
}

// contidoOpenAI é o "content" dunha mensaxe de usuario: texto simple ou, se
// leva imaxes, a lista de partes text + image_url.
func contidoOpenAI(texto string, imaxes []Imaxe) any {
	if len(imaxes) == 0 {
		return texto
	}
	partes := []any{map[string]any{"type": "text", "text": texto}}
	for _, im := range imaxes {
		partes = append(partes, map[string]any{"type": "image_url", "image_url": map[string]any{"url": "data:" + im.Mime + ";base64," + im.B64}})
	}
	return partes
}

func turnoOpenAI(baseURL, apiKey, model, sistema string, msgs []IAMensaxe, ferramentas []IAFerramenta, maxTokens int, cliente *http.Client) (IATurno, error) {
	var mensaxes []any
	mensaxes = append(mensaxes, map[string]any{"role": "system", "content": sistema})
	for _, m := range msgs {
		switch m.Rol {
		case "user":
			mensaxes = append(mensaxes, map[string]any{"role": "user", "content": contidoOpenAI(m.Texto, m.ImaxesIA)})
		case "assistant":
			if m.Bruto != "" && m.Provedor == string(ProvedorOpenAI) {
				mensaxes = append(mensaxes, json.RawMessage(m.Bruto))
				continue
			}
			msg := map[string]any{"role": "assistant", "content": m.Texto}
			if len(m.Chamadas) > 0 {
				var tcs []any
				for _, c := range m.Chamadas {
					tcs = append(tcs, map[string]any{"id": c.ID, "type": "function",
						"function": map[string]any{"name": c.Nome, "arguments": argsOuBaleiro(c.Args)}})
				}
				msg["tool_calls"] = tcs
			}
			mensaxes = append(mensaxes, msg)
		case "tool":
			var imaxes []Imaxe
			for _, r := range m.Resultados {
				mensaxes = append(mensaxes, map[string]any{"role": "tool", "tool_call_id": r.ChamadaID, "content": r.Contido})
				imaxes = append(imaxes, r.ImaxesIA...)
			}
			// As mensaxes "tool" só levan texto: as imaxes que devolveron as
			// ferramentas van nunha mensaxe de usuario a continuación.
			if len(imaxes) > 0 {
				mensaxes = append(mensaxes, map[string]any{"role": "user", "content": contidoOpenAI("Imaxes devoltas polas ferramentas anteriores:", imaxes)})
			}
		}
	}
	corpo := map[string]any{"model": model, "messages": mensaxes}
	if maxTokens > 0 {
		corpo["max_tokens"] = maxTokens
	}
	if len(ferramentas) > 0 {
		var ts []any
		for _, f := range ferramentas {
			ts = append(ts, map[string]any{"type": "function", "function": map[string]any{
				"name": f.Nome, "description": f.Descricion, "parameters": esquemaOuBaleiro(f)}})
		}
		corpo["tools"] = ts
	}
	b, estado, err := postJSONIA(cliente, baseURL+"/chat/completions", map[string]string{"Authorization": "Bearer " + apiKey}, corpo)
	if err != nil {
		return IATurno{}, err
	}
	var r openAITurnoResposta
	if err := json.Unmarshal(b, &r); err != nil {
		return IATurno{}, fmt.Errorf("resposta non válida da IA (HTTP %d)", estado)
	}
	if r.Error != nil {
		return IATurno{}, fmt.Errorf("erro da IA: %s", r.Error.Message)
	}
	if estado != http.StatusOK || len(r.Choices) == 0 {
		return IATurno{}, fmt.Errorf("a IA non devolveu resultado (HTTP %d)", estado)
	}
	var m openAIMsgSaida
	if err := json.Unmarshal(r.Choices[0].Message, &m); err != nil {
		return IATurno{}, fmt.Errorf("resposta non válida da IA: %w", err)
	}
	t := IATurno{Truncado: r.Choices[0].FinishReason == "length"}
	if m.Content != nil {
		t.Texto = *m.Content
	}
	// O que se garda en bruto é só o que hai que devolver: role, content,
	// tool_calls (e o razoamento, se o provedor o manda: DeepSeek esíxeo).
	bruto := map[string]any{"role": "assistant", "content": t.Texto}
	if len(m.ToolCalls) > 0 {
		var tcs []any
		for i, c := range m.ToolCalls {
			id := c.ID
			if id == "" {
				id = fmt.Sprintf("chamada_%d", i)
			}
			t.Chamadas = append(t.Chamadas, IAChamada{ID: id, Nome: c.Function.Name, Args: argsOuBaleiro(c.Function.Arguments)})
			tcs = append(tcs, map[string]any{"id": id, "type": "function",
				"function": map[string]any{"name": c.Function.Name, "arguments": argsOuBaleiro(c.Function.Arguments)}})
		}
		bruto["tool_calls"] = tcs
	}
	if m.ReasoningContent != nil {
		bruto["reasoning_content"] = *m.ReasoningContent
	}
	bb, _ := json.Marshal(bruto)
	t.Bruto = string(bb)
	return t, nil
}

// ── Anthropic ───────────────────────────────────────────────────────────

type anthropicTurnoResposta struct {
	Content    json.RawMessage `json:"content"`
	StopReason string          `json:"stop_reason"`
	Error      *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func bloqueImaxeAnthropic(im Imaxe) map[string]any {
	return map[string]any{"type": "image", "source": map[string]any{"type": "base64", "media_type": im.Mime, "data": im.B64}}
}

// contidoAnthropic: texto ou, con imaxes, bloques image + text.
func contidoAnthropic(texto string, imaxes []Imaxe) any {
	if len(imaxes) == 0 {
		return texto
	}
	var bl []any
	for _, im := range imaxes {
		bl = append(bl, bloqueImaxeAnthropic(im))
	}
	return append(bl, map[string]any{"type": "text", "text": texto})
}

// contidoAnthropicResultado: o contido dun tool_result que devolve imaxes.
func contidoAnthropicResultado(texto string, imaxes []Imaxe) any {
	bl := []any{map[string]any{"type": "text", "text": texto}}
	for _, im := range imaxes {
		bl = append(bl, bloqueImaxeAnthropic(im))
	}
	return bl
}

func turnoAnthropic(baseURL, apiKey, model, sistema string, msgs []IAMensaxe, ferramentas []IAFerramenta, maxTokens int, cliente *http.Client) (IATurno, error) {
	var mensaxes []any
	for _, m := range msgs {
		switch m.Rol {
		case "user":
			mensaxes = append(mensaxes, map[string]any{"role": "user", "content": contidoAnthropic(m.Texto, m.ImaxesIA)})
		case "assistant":
			if m.Bruto != "" && m.Provedor == string(ProvedorAnthropic) {
				mensaxes = append(mensaxes, map[string]any{"role": "assistant", "content": json.RawMessage(m.Bruto)})
				continue
			}
			var bloques []any
			if strings.TrimSpace(m.Texto) != "" {
				bloques = append(bloques, map[string]any{"type": "text", "text": m.Texto})
			}
			for _, c := range m.Chamadas {
				bloques = append(bloques, map[string]any{"type": "tool_use", "id": c.ID, "name": c.Nome, "input": json.RawMessage(argsOuBaleiro(c.Args))})
			}
			if len(bloques) == 0 {
				bloques = append(bloques, map[string]any{"type": "text", "text": "…"})
			}
			mensaxes = append(mensaxes, map[string]any{"role": "assistant", "content": bloques})
		case "tool":
			var bloques []any
			for _, r := range m.Resultados {
				var contido any = r.Contido
				if len(r.ImaxesIA) > 0 {
					contido = contidoAnthropicResultado(r.Contido, r.ImaxesIA)
				}
				bloques = append(bloques, map[string]any{"type": "tool_result", "tool_use_id": r.ChamadaID, "content": contido, "is_error": r.Erro})
			}
			mensaxes = append(mensaxes, map[string]any{"role": "user", "content": bloques})
		}
	}
	corpo := map[string]any{"model": model, "max_tokens": maxTokens, "messages": mensaxes}
	if sistema != "" {
		// "ephemeral": o sistema e as ferramentas repítense en cada turno do
		// bucle - Claude cachea eses tokens e cobra menos por eles.
		corpo["system"] = []any{map[string]any{"type": "text", "text": sistema, "cache_control": map[string]any{"type": "ephemeral"}}}
	}
	if len(ferramentas) > 0 {
		var ts []any
		for _, f := range ferramentas {
			ts = append(ts, map[string]any{"name": f.Nome, "description": f.Descricion, "input_schema": esquemaOuBaleiro(f)})
		}
		corpo["tools"] = ts
	}
	b, estado, err := postJSONIA(cliente, baseURL+"/messages", map[string]string{"x-api-key": apiKey, "anthropic-version": "2023-06-01"}, corpo)
	if err != nil {
		return IATurno{}, err
	}
	var r anthropicTurnoResposta
	if err := json.Unmarshal(b, &r); err != nil {
		return IATurno{}, fmt.Errorf("resposta non válida da IA (HTTP %d)", estado)
	}
	if r.Error != nil {
		return IATurno{}, fmt.Errorf("erro da IA: %s", r.Error.Message)
	}
	if estado != http.StatusOK || len(r.Content) == 0 {
		return IATurno{}, fmt.Errorf("a IA non devolveu resultado (HTTP %d)", estado)
	}
	var bloques []struct {
		Type  string          `json:"type"`
		Text  string          `json:"text"`
		ID    string          `json:"id"`
		Name  string          `json:"name"`
		Input json.RawMessage `json:"input"`
	}
	if err := json.Unmarshal(r.Content, &bloques); err != nil {
		return IATurno{}, fmt.Errorf("resposta non válida da IA: %w", err)
	}
	t := IATurno{Bruto: string(r.Content), Truncado: r.StopReason == "max_tokens"}
	var texto strings.Builder
	for _, bl := range bloques {
		switch bl.Type {
		case "text":
			texto.WriteString(bl.Text)
		case "tool_use":
			t.Chamadas = append(t.Chamadas, IAChamada{ID: bl.ID, Nome: bl.Name, Args: argsOuBaleiro(string(bl.Input))})
		}
	}
	t.Texto = texto.String()
	return t, nil
}

// ── Gemini ──────────────────────────────────────────────────────────────

type geminiTurnoResposta struct {
	Candidates []struct {
		Content struct {
			Parts json.RawMessage `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func turnoGemini(baseURL, apiKey, model, sistema string, msgs []IAMensaxe, ferramentas []IAFerramenta, maxTokens int, cliente *http.Client) (IATurno, error) {
	var contidos []any
	for _, m := range msgs {
		switch m.Rol {
		case "user":
			partes := []any{map[string]any{"text": m.Texto}}
			for _, im := range m.ImaxesIA {
				partes = append(partes, map[string]any{"inlineData": map[string]any{"mimeType": im.Mime, "data": im.B64}})
			}
			contidos = append(contidos, map[string]any{"role": "user", "parts": partes})
		case "assistant":
			if m.Bruto != "" && m.Provedor == string(ProvedorGemini) {
				contidos = append(contidos, map[string]any{"role": "model", "parts": json.RawMessage(m.Bruto)})
				continue
			}
			var partes []any
			if strings.TrimSpace(m.Texto) != "" {
				partes = append(partes, map[string]any{"text": m.Texto})
			}
			for _, c := range m.Chamadas {
				partes = append(partes, map[string]any{"functionCall": map[string]any{"name": c.Nome, "args": json.RawMessage(argsOuBaleiro(c.Args))}})
			}
			if len(partes) == 0 {
				partes = append(partes, map[string]any{"text": "…"})
			}
			contidos = append(contidos, map[string]any{"role": "model", "parts": partes})
		case "tool":
			var partes []any
			for _, r := range m.Resultados {
				clave := "resultado"
				if r.Erro {
					clave = "erro"
				}
				partes = append(partes, map[string]any{"functionResponse": map[string]any{
					"name": r.Nome, "response": map[string]any{clave: r.Contido}}})
			}
			for _, r := range m.Resultados {
				for _, im := range r.ImaxesIA {
					partes = append(partes, map[string]any{"inlineData": map[string]any{"mimeType": im.Mime, "data": im.B64}})
				}
			}
			contidos = append(contidos, map[string]any{"role": "user", "parts": partes})
		}
	}
	corpo := map[string]any{
		"systemInstruction": map[string]any{"role": "user", "parts": []any{map[string]any{"text": sistema}}},
		"contents":          contidos,
	}
	if maxTokens > 0 {
		corpo["generationConfig"] = map[string]any{"maxOutputTokens": maxTokens}
	}
	if len(ferramentas) > 0 {
		var decl []any
		for _, f := range ferramentas {
			d := map[string]any{"name": f.Nome, "description": f.Descricion}
			// Gemini rexeita un obxecto de parámetros sen propiedades.
			if esquemaTenPropiedades(f) {
				d["parameters"] = esquemaOuBaleiro(f)
			}
			decl = append(decl, d)
		}
		corpo["tools"] = []any{map[string]any{"functionDeclarations": decl}}
	}
	b, estado, err := postJSONIA(cliente, baseURL+"/models/"+model+":generateContent", map[string]string{"x-goog-api-key": apiKey}, corpo)
	if err != nil {
		return IATurno{}, err
	}
	var r geminiTurnoResposta
	if err := json.Unmarshal(b, &r); err != nil {
		return IATurno{}, fmt.Errorf("resposta non válida da IA (HTTP %d)", estado)
	}
	if r.Error != nil {
		return IATurno{}, fmt.Errorf("erro da IA: %s", r.Error.Message)
	}
	if estado != http.StatusOK || len(r.Candidates) == 0 {
		return IATurno{}, fmt.Errorf("a IA non devolveu resultado (HTTP %d)", estado)
	}
	var partes []struct {
		Text         string `json:"text"`
		Thought      bool   `json:"thought"`
		FunctionCall *struct {
			ID   string          `json:"id"`
			Name string          `json:"name"`
			Args json.RawMessage `json:"args"`
		} `json:"functionCall"`
	}
	brutoPartes := r.Candidates[0].Content.Parts
	if len(brutoPartes) == 0 || json.Unmarshal(brutoPartes, &partes) != nil {
		// Sen partes: bloqueada ou baleira (a mensaxe tampouco leva bruto que devolver).
		return IATurno{Truncado: r.Candidates[0].FinishReason == "MAX_TOKENS"}, nil
	}
	t := IATurno{Bruto: string(brutoPartes), Truncado: r.Candidates[0].FinishReason == "MAX_TOKENS"}
	var texto strings.Builder
	for i, p := range partes {
		switch {
		case p.FunctionCall != nil:
			id := p.FunctionCall.ID
			if id == "" {
				id = fmt.Sprintf("chamada_%d", i)
			}
			t.Chamadas = append(t.Chamadas, IAChamada{ID: id, Nome: p.FunctionCall.Name, Args: argsOuBaleiro(string(p.FunctionCall.Args))})
		case p.Text != "" && !p.Thought:
			texto.WriteString(p.Text)
		}
	}
	t.Texto = texto.String()
	return t, nil
}
