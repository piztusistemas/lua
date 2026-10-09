package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestChamarGeminiFormatoFio comproba que chamarGemini fala o formato de
// fío exacto que espera a API de Gemini (URL .../models/<model>:generate
// Content, cabeceira x-goog-api-key) e que sabe ler unha resposta real.
func TestChamarGeminiFormatoFio(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models/modelo-proba:generateContent" {
			t.Errorf("ruta inesperada: %s", r.URL.Path)
		}
		if r.Header.Get("x-goog-api-key") != "clave-proba" {
			t.Errorf("falta ou é incorrecta a cabeceira x-goog-api-key: %q", r.Header.Get("x-goog-api-key"))
		}
		var corpo geminiPeticion
		if err := json.NewDecoder(r.Body).Decode(&corpo); err != nil {
			t.Fatalf("corpo non válido: %v", err)
		}
		if corpo.SystemInstruction == nil || corpo.SystemInstruction.Parts[0].Text != "sistema-proba" {
			t.Errorf("systemInstruction non chegou coma se esperaba: %+v", corpo.SystemInstruction)
		}
		json.NewEncoder(w).Encode(geminiResposta{
			Candidates: []struct {
				Content geminiContido `json:"content"`
			}{{Content: geminiContido{Parts: []geminiParte{{Text: "resposta da IA"}}}}},
		})
	}))
	defer srv.Close()

	texto, err := chamarGemini(srv.URL, "clave-proba", "modelo-proba", "sistema-proba", "peticion-proba", 0, nil)
	if err != nil {
		t.Fatalf("chamarGemini: %v", err)
	}
	if texto != "resposta da IA" {
		t.Errorf("resposta = %q, quería %q", texto, "resposta da IA")
	}
}

// TestChamarAnthropicFormatoFio comproba o formato de fío de Claude:
// POST .../messages, cabeceiras x-api-key + anthropic-version (non Bearer).
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
		var corpo anthropicPeticion
		if err := json.NewDecoder(r.Body).Decode(&corpo); err != nil {
			t.Fatalf("corpo non válido: %v", err)
		}
		if len(corpo.System) != 1 {
			t.Fatalf("system = %+v, quería exactamente un bloque", corpo.System)
		}
		if corpo.System[0].Text != "sistema-proba" {
			t.Errorf("system[0].text = %q, quería %q", corpo.System[0].Text, "sistema-proba")
		}
		if corpo.System[0].CacheControl == nil || corpo.System[0].CacheControl.Type != "ephemeral" {
			t.Errorf("system[0].cache_control = %+v, quería {ephemeral} (para aforrar tokens en chamadas repetidas)", corpo.System[0].CacheControl)
		}
		json.NewEncoder(w).Encode(anthropicResposta{
			Content: []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{{Type: "text", Text: "resposta da IA"}},
		})
	}))
	defer srv.Close()

	texto, err := chamarAnthropic(srv.URL, "clave-proba", "modelo-proba", "sistema-proba", "peticion-proba", 0, nil)
	if err != nil {
		t.Fatalf("chamarAnthropic: %v", err)
	}
	if texto != "resposta da IA" {
		t.Errorf("resposta = %q, quería %q", texto, "resposta da IA")
	}
}

// TestChamarOpenAICompatibleFormatoFio comproba o formato "chat
// completions" - o que fala tanto a propia OpenAI coma Qwen/Perplexity/
// calquera outro "compatible con OpenAI" (só cambia o enderezo).
func TestChamarOpenAICompatibleFormatoFio(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("ruta inesperada: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer clave-proba" {
			t.Errorf("falta ou é incorrecta a cabeceira Authorization: %q", r.Header.Get("Authorization"))
		}
		var corpo openAIPeticion
		if err := json.NewDecoder(r.Body).Decode(&corpo); err != nil {
			t.Fatalf("corpo non válido: %v", err)
		}
		if len(corpo.Messages) != 2 || corpo.Messages[0].Role != "system" || corpo.Messages[1].Role != "user" {
			t.Errorf("messages non teñen a forma esperada: %+v", corpo.Messages)
		}
		json.NewEncoder(w).Encode(openAIResposta{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{{Message: struct {
				Content string `json:"content"`
			}{Content: "resposta da IA"}}},
		})
	}))
	defer srv.Close()

	texto, err := chamarOpenAICompatible(srv.URL, "clave-proba", "modelo-proba", "sistema-proba", "peticion-proba", 0, nil)
	if err != nil {
		t.Fatalf("chamarOpenAICompatible: %v", err)
	}
	if texto != "resposta da IA" {
		t.Errorf("resposta = %q, quería %q", texto, "resposta da IA")
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
