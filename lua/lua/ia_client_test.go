package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Os formatos de fío viven en edicion/ia; estes tests comproban, a través
// de chamarIA (a entrada que usa Lúa), que cada provedor recibe a forma que
// espera a súa API. A caracterización (ia_caracterizacion_test.go) fixa
// ademais cada byte do que se envía.

func corpoJSON(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	var c map[string]any
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		t.Fatalf("corpo non válido: %v", err)
	}
	return c
}

func settingsProba(provedor, url string) Settings {
	return Settings{IAProvedor: provedor, IABaseURL: url, IAAPIKey: "clave-proba", IAModel: "modelo-proba"}
}

// TestChamarGeminiFormatoFio: URL .../models/<model>:generateContent,
// cabeceira x-goog-api-key e systemInstruction.
func TestChamarGeminiFormatoFio(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models/modelo-proba:generateContent" {
			t.Errorf("ruta inesperada: %s", r.URL.Path)
		}
		if r.Header.Get("x-goog-api-key") != "clave-proba" {
			t.Errorf("falta ou é incorrecta a cabeceira x-goog-api-key: %q", r.Header.Get("x-goog-api-key"))
		}
		si, _ := corpoJSON(t, r)["systemInstruction"].(map[string]any)
		partes, _ := si["parts"].([]any)
		if len(partes) == 0 || partes[0].(map[string]any)["text"] != "sistema-proba" {
			t.Errorf("systemInstruction non chegou coma se esperaba: %+v", si)
		}
		io.WriteString(w, `{"candidates":[{"content":{"parts":[{"text":"resposta da IA"}]}}]}`)
	}))
	defer srv.Close()
	texto, err := chamarIA(settingsProba("gemini", srv.URL), "sistema-proba", "peticion-proba")
	if err != nil || texto != "resposta da IA" {
		t.Errorf("resposta = %q, %v", texto, err)
	}
}

// TestChamarAnthropicFormatoFio: POST .../messages, cabeceiras x-api-key +
// anthropic-version (non Bearer) e o sistema nun bloque con cache_control.
func TestChamarAnthropicFormatoFio(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/messages" {
			t.Errorf("ruta inesperada: %s", r.URL.Path)
		}
		if r.Header.Get("x-api-key") != "clave-proba" {
			t.Errorf("falta a cabeceira x-api-key: %q", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") == "" {
			t.Error("falta a cabeceira anthropic-version")
		}
		system, _ := corpoJSON(t, r)["system"].([]any)
		if len(system) != 1 {
			t.Fatalf("system = %+v, quería exactamente un bloque", system)
		}
		b := system[0].(map[string]any)
		if b["text"] != "sistema-proba" {
			t.Errorf("system[0].text = %v", b["text"])
		}
		if cc, _ := b["cache_control"].(map[string]any); cc["type"] != "ephemeral" {
			t.Errorf("system[0].cache_control = %+v, quería {ephemeral} (para aforrar tokens en chamadas repetidas)", b["cache_control"])
		}
		io.WriteString(w, `{"content":[{"type":"text","text":"resposta da IA"}]}`)
	}))
	defer srv.Close()
	texto, err := chamarIA(settingsProba("anthropic", srv.URL), "sistema-proba", "peticion-proba")
	if err != nil || texto != "resposta da IA" {
		t.Errorf("resposta = %q, %v", texto, err)
	}
}

// TestChamarOpenAICompatibleFormatoFio: formato "chat completions", o que
// fala tanto a propia OpenAI coma calquera «compatible con OpenAI».
func TestChamarOpenAICompatibleFormatoFio(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("ruta inesperada: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer clave-proba" {
			t.Errorf("falta ou é incorrecta a cabeceira Authorization: %q", r.Header.Get("Authorization"))
		}
		msgs, _ := corpoJSON(t, r)["messages"].([]any)
		if len(msgs) != 2 || msgs[0].(map[string]any)["role"] != "system" || msgs[1].(map[string]any)["role"] != "user" {
			t.Errorf("messages non teñen a forma esperada: %+v", msgs)
		}
		io.WriteString(w, `{"choices":[{"message":{"content":"resposta da IA"}}]}`)
	}))
	defer srv.Close()
	texto, err := chamarIA(settingsProba("openai", srv.URL), "sistema-proba", "peticion-proba")
	if err != nil || texto != "resposta da IA" {
		t.Errorf("resposta = %q, %v", texto, err)
	}
}

// TestChamarIAValidacion comproba que chamarIA esixe clave e modelo antes
// de tentar sequera contactar coa rede.
func TestChamarIAValidacion(t *testing.T) {
	if _, err := chamarIA(Settings{}, "sistema", "peticion"); err == nil {
		t.Error("esperaba erro sen clave da API")
	}
	if _, err := chamarIA(Settings{IAAPIKey: "clave"}, "sistema", "peticion"); err == nil {
		t.Error("esperaba erro sen modelo")
	}
}

// TestChamarIADespachoPorProvedor comproba que cada provedor configurado
// chama realmente o formato de fío correspondente, montando un servidor de
// proba distinto para cada un e apuntando IABaseURL cara a el.
func TestChamarIADespachoPorProvedor(t *testing.T) {
	casos := []struct {
		provedor     string
		rutaAgardada string
	}{
		{string(ProvedorGemini), "/models/m:generateContent"},
		{string(ProvedorAnthropic), "/messages"},
		{string(ProvedorOpenAI), "/chat/completions"},
		{"", "/models/m:generateContent"}, // baleiro = Gemini, retrocompatibilidade
	}
	for _, c := range casos {
		var rutaChamada string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rutaChamada = r.URL.Path
			// Resposta mínima válida para calquera dos tres formatos.
			io.WriteString(w, `{"candidates":[{"content":{"parts":[{"text":"ok"}]}}],`+
				`"content":[{"type":"text","text":"ok"}],`+
				`"choices":[{"message":{"content":"ok"}}]}`)
		}))
		s := Settings{IAProvedor: c.provedor, IAAPIKey: "clave", IAModel: "m", IABaseURL: srv.URL}
		if _, err := chamarIA(s, "sistema", "peticion"); err != nil {
			t.Errorf("provedor %q: chamarIA: %v", c.provedor, err)
		}
		if rutaChamada != c.rutaAgardada {
			t.Errorf("provedor %q: ruta chamada = %q, quería %q", c.provedor, rutaChamada, c.rutaAgardada)
		}
		srv.Close()
	}
}
