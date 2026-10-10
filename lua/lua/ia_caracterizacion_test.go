package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
)

// Caracterización do cliente de IA (spec sen-duplicados-ia-latex): grava o que
// Lúa envía a cada provedor e o que interpreta da resposta, e compárao co
// ficheiro de referencia. A refactorización non pode cambiar nin un byte.
// Para rexenerar a referencia (só antes de refactorizar): CARACTERIZAR=1.

const ficheiroCaracterizacionIA = "testdata/ia_caracterizacion.golden.json"

type peticionGravada struct {
	Metodo    string            `json:"metodo"`
	Ruta      string            `json:"ruta"`
	Query     string            `json:"query"`
	Cabeceira map[string]string `json:"cabeceiras"`
	Corpo     any               `json:"corpo"`
}

type casoGravado struct {
	Nome      string            `json:"nome"`
	Peticions []peticionGravada `json:"peticions"`
	Resultado any               `json:"resultado"`
}

var cabeceirasGravadas = []string{"Authorization", "X-Api-Key", "X-Goog-Api-Key", "Anthropic-Version", "Content-Type"}

func servidorIAProba(t *testing.T) (*httptest.Server, func() []peticionGravada) {
	t.Helper()
	var mu sync.Mutex
	var gravadas []peticionGravada
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		datos, _ := io.ReadAll(r.Body)
		var corpo any
		_ = json.Unmarshal(datos, &corpo)
		cab := map[string]string{}
		for _, h := range cabeceirasGravadas {
			if v := r.Header.Get(h); v != "" {
				cab[h] = v
			}
		}
		mu.Lock()
		gravadas = append(gravadas, peticionGravada{r.Method, r.URL.Path, r.URL.RawQuery, cab, corpo})
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, ":generateContent"):
			io.WriteString(w, `{"candidates":[{"content":{"role":"model","parts":[{"text":"ola gemini"},{"functionCall":{"name":"f","args":{"a":1}}}]},"finishReason":"STOP"}]}`)
		case strings.HasSuffix(r.URL.Path, "/messages"):
			io.WriteString(w, "{\"content\":[{\"type\":\"text\",\"text\":\"```\\nola claude\\n```\"},{\"type\":\"tool_use\",\"id\":\"t1\",\"name\":\"f\",\"input\":{\"a\":1}}],\"stop_reason\":\"tool_use\"}")
		default:
			io.WriteString(w, `{"choices":[{"message":{"content":"ola openai","tool_calls":[{"id":"c1","type":"function","function":{"name":"f","arguments":"{\"a\":1}"}}]},"finish_reason":"tool_calls"}]}`)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, func() []peticionGravada {
		mu.Lock()
		defer mu.Unlock()
		out := gravadas
		gravadas = nil
		return out
	}
}

// conImaxesIA pon imaxes xa resoltas nunha mensaxe e nos seus resultados (o
// que fai resolverAdxuntosMensaxes antes de cada turno).
func conImaxesIA(m IAMensaxe, ims []imaxeIA) IAMensaxe {
	m.imaxes = ims
	for i := range m.Resultados {
		m.Resultados[i].imaxesIA = ims
	}
	return m
}

func casosCaracterizacionIA(t *testing.T) []casoGravado {
	srv, gravadas := servidorIAProba(t)
	im := []imaxeIA{{Mime: "image/png", B64: "QUJD"}}
	ferr := []IAFerramenta{
		{Nome: "f", Descricion: "unha ferramenta", Parametros: `{"type":"object","properties":{"a":{"type":"number"}}}`},
		{Nome: "g", Descricion: "sen parámetros", Parametros: ``},
	}
	msgs := []IAMensaxe{
		conImaxesIA(IAMensaxe{Rol: "user", Texto: "mira isto"}, im),
		{Rol: "assistant", Texto: "vou mirar", Chamadas: []IAChamada{{ID: "c1", Nome: "f", Args: `{"a":1}`}}},
		conImaxesIA(IAMensaxe{Rol: "user", Resultados: []IAResultado{{ChamadaID: "c1", Nome: "f", Contido: "resultado", Imaxes: []string{"x"}}}}, im),
	}
	var casos []casoGravado
	for _, prov := range []string{"gemini", "anthropic", "openai"} {
		for _, max := range []int{0, 1234} {
			s := Settings{IAProvedor: prov, IABaseURL: srv.URL + "/", IAAPIKey: "clave", IAModel: "modelo", IAMaxTokens: max}
			sufixo := prov + "/max" + map[bool]string{true: "0", false: "1234"}[max == 0]

			texto, err := chamarIA(s, "sistema", "petición")
			casos = append(casos, casoGravado{"simple/" + sufixo, gravadas(), map[string]any{"texto": texto, "erro": erroTexto(err)}})

			turno, err := chamarIAConversa(s, "sistema", msgs, ferr)
			casos = append(casos, casoGravado{"conversa/" + sufixo, gravadas(), map[string]any{"turno": turno, "erro": erroTexto(err)}})
		}
	}
	// O provedor responde cun erro HTTP: mesmas mensaxes nos dous camiños.
	erroSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		io.WriteString(w, `{"error":{"message":"cota esgotada"}}`)
	}))
	defer erroSrv.Close()
	for _, prov := range []string{"gemini", "anthropic", "openai"} {
		s := Settings{IAProvedor: prov, IABaseURL: erroSrv.URL, IAAPIKey: "clave", IAModel: "modelo"}
		_, e1 := chamarIA(s, "sistema", "petición")
		_, e2 := chamarIAConversa(s, "sistema", []IAMensaxe{{Rol: "user", Texto: "petición"}}, nil)
		casos = append(casos, casoGravado{"erros/http/" + prov, nil, map[string]any{"simple": erroTexto(e1), "conversa": erroTexto(e2)}})
	}
	// Sen clave: erro sen chegar á rede.
	_, err := chamarIA(Settings{IAModel: "m"}, "s", "p")
	_, err2 := chamarIAConversa(Settings{IAAPIKey: "k"}, "s", nil, nil)
	casos = append(casos, casoGravado{"erros/configuracion", gravadas(), map[string]any{"simple": erroTexto(err), "conversa": erroTexto(err2)}})
	return casos
}

func erroTexto(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func TestCaracterizacionClienteIA(t *testing.T) {
	casos := casosCaracterizacionIA(t)
	sort.Slice(casos, func(i, j int) bool { return casos[i].Nome < casos[j].Nome })
	actual, err := json.MarshalIndent(casos, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if os.Getenv("CARACTERIZAR") == "1" {
		_ = os.MkdirAll(filepath.Dir(ficheiroCaracterizacionIA), 0o755)
		if err := os.WriteFile(ficheiroCaracterizacionIA, append(actual, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("referencia gravada en %s (%d casos)", ficheiroCaracterizacionIA, len(casos))
		return
	}
	esperado, err := os.ReadFile(ficheiroCaracterizacionIA)
	if err != nil {
		t.Fatalf("falta a referencia (%v): xérase con CARACTERIZAR=1 antes de refactorizar", err)
	}
	if strings.TrimSpace(string(esperado)) != strings.TrimSpace(string(actual)) {
		ruta := filepath.Join(t.TempDir(), "actual.json")
		_ = os.WriteFile(ruta, actual, 0o644)
		t.Errorf("o cliente de IA cambiou o que envía ou o que interpreta.\n  referencia: %s\n  actual:     %s\n  (diff -u entre os dous)", ficheiroCaracterizacionIA, ruta)
	}
}
