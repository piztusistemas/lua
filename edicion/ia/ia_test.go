package ia

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestPreguntarAnthropicConImaxe(t *testing.T) {
	var corpo map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "k" || !strings.HasSuffix(r.URL.Path, "/messages") {
			t.Errorf("petición: %s %v", r.URL.Path, r.Header)
		}
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &corpo)
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"$x^2$"}],"stop_reason":"end_turn"}`))
	}))
	defer srv.Close()
	c := Config{Provedor: ProvedorAnthropic, BaseURL: srv.URL, APIKey: "k", Modelo: "m"}
	txt, err := Preguntar(c, "transcribe", "páxina 1", ImaxeDeBytes("image/png", []byte{1, 2, 3}))
	if err != nil || txt != "$x^2$" {
		t.Fatalf("%q %v", txt, err)
	}
	j, _ := json.Marshal(corpo["messages"])
	if !strings.Contains(string(j), `"type":"image"`) || !strings.Contains(string(j), "AQID") {
		t.Errorf("a imaxe non foi: %s", j)
	}
}

func TestAxenteOpenAI(t *testing.T) {
	var n int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		switch atomic.AddInt32(&n, 1) {
		case 1:
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"","tool_calls":[{"id":"c1","type":"function","function":{"name":"sumar","arguments":"{\"a\":2,\"b\":3}"}}]},"finish_reason":"tool_calls"}]}`))
		default:
			if !strings.Contains(string(b), `"tool_call_id":"c1"`) || !strings.Contains(string(b), `5`) {
				t.Errorf("non chegou o resultado da ferramenta: %s", b)
			}
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"Son 5."},"finish_reason":"stop"}]}`))
		}
	}))
	defer srv.Close()
	c := Config{Provedor: ProvedorOpenAI, BaseURL: srv.URL, APIKey: "k", Modelo: "m"}
	sumar := Ferramenta{
		Def: IAFerramenta{Nome: "sumar", Descricion: "suma", Parametros: `{"type":"object","properties":{"a":{"type":"number"},"b":{"type":"number"}}}`},
		Executar: func(args json.RawMessage) (string, []Imaxe, error) {
			var a struct{ A, B float64 }
			_ = json.Unmarshal(args, &a)
			return strings.TrimSpace(strings.Repeat(" ", 0) + jsonNum(a.A+a.B)), nil, nil
		},
	}
	var avisos []string
	txt, err := Axente(context.Background(), c, "sistema", "canto é 2+3?", []Ferramenta{sumar}, 5, func(s string) { avisos = append(avisos, s) })
	if err != nil || txt != "Son 5." || len(avisos) != 1 {
		t.Fatalf("%q %v %v", txt, err, avisos)
	}
}

func jsonNum(f float64) string { b, _ := json.Marshal(f); return string(b) }

func TestExtraerJSON(t *testing.T) {
	for entrada, esperado := range map[string]string{
		"```json\n{\"a\":1}\n```":         `{"a":1}`,
		"Aquí vai: {\"a\":{\"b\":2}} fin": `{"a":{"b":2}}`,
		"[1,2]":                           `[1,2]`,
	} {
		if g := ExtraerJSON(entrada); g != esperado {
			t.Errorf("%q -> %q", entrada, g)
		}
	}
}

// As imaxes que manda un módulo pola API local (Buxo: páxinas dun PDF) viaxan
// no JSON da mensaxe coma "imaxes", igual ca no tipo que usaba Lúa.
func TestIAMensaxeLevaImaxesEnJSON(t *testing.T) {
	m := IAMensaxe{Rol: "user", Texto: "x", Imaxes: []string{"data:image/png;base64,QUJD"}}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	var volta IAMensaxe
	if err := json.Unmarshal(b, &volta); err != nil || len(volta.Imaxes) != 1 || volta.Imaxes[0] != m.Imaxes[0] {
		t.Errorf("imaxes perdidas no JSON: %s", b)
	}
	if b2, _ := json.Marshal(IAMensaxe{Rol: "user"}); strings.Contains(string(b2), `"imaxes"`) {
		t.Errorf("sen imaxes, o campo non debe saír: %s", b2)
	}
}
