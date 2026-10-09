package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"regexp"
	"strings"
	"testing"
)

var regexpNomeFerramenta = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

func ferramentaProba() []IAFerramenta {
	return []IAFerramenta{{
		Nome: "curso_ler", Descricion: "le", Tipo: "lectura", Onde: "frontend",
		Parametros: esqObx(map[string]any{"mes": esqTxt("mes")}),
	}, {
		Nome: "sen_args", Descricion: "sen argumentos", Tipo: "lectura", Onde: "frontend",
	}}
}

func lerCorpo(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	b, _ := io.ReadAll(r.Body)
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("corpo non válido: %v\n%s", err, b)
	}
	return m
}

// conversaProba: o usuario pide algo, a IA chama unha ferramenta, o
// resultado volve, e a IA responde. Serve para os tres formatos.
func conversaProba(bruto, provedor string) []IAMensaxe {
	return []IAMensaxe{
		{Rol: "user", Texto: "corrixe o curso"},
		{Rol: "assistant", Texto: "Vou mirar.", Bruto: bruto, Provedor: provedor,
			Chamadas: []IAChamada{{ID: "c1", Nome: "curso_ler", Args: `{"mes":"2025-09"}`}}},
		{Rol: "tool", Resultados: []IAResultado{{ChamadaID: "c1", Nome: "curso_ler", Contido: "{\"ok\":true}"}}},
	}
}

func TestTurnoOpenAIChamadasAFerramentas(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" || r.Header.Get("Authorization") != "Bearer clave" {
			t.Errorf("ruta/cabeceira inesperadas: %s %q", r.URL.Path, r.Header.Get("Authorization"))
		}
		c := lerCorpo(t, r)
		msgs := c["messages"].([]any)
		if msgs[0].(map[string]any)["role"] != "system" {
			t.Error("a primeira mensaxe ten que ser a do sistema")
		}
		ultima := msgs[len(msgs)-1].(map[string]any)
		if ultima["role"] != "tool" || ultima["tool_call_id"] != "c1" {
			t.Errorf("o resultado da ferramenta non chegou: %+v", ultima)
		}
		// A mensaxe do asistente vai en bruto, tal cal (con reasoning_content).
		asis := msgs[2].(map[string]any)
		if asis["reasoning_content"] != "pensei" {
			t.Errorf("o bruto non se devolveu: %+v", asis)
		}
		ts := c["tools"].([]any)
		if len(ts) != 2 || ts[0].(map[string]any)["type"] != "function" {
			t.Errorf("tools mal formadas: %+v", ts)
		}
		w.Write([]byte(`{"choices":[{"finish_reason":"tool_calls","message":{"role":"assistant","content":null,
			"tool_calls":[{"id":"call_9","type":"function","function":{"name":"sen_args","arguments":""}}],
			"reasoning_content":"outro"}}]}`))
	}))
	defer srv.Close()

	bruto := `{"role":"assistant","content":"Vou mirar.","reasoning_content":"pensei","tool_calls":[{"id":"c1","type":"function","function":{"name":"curso_ler","arguments":"{\"mes\":\"2025-09\"}"}}]}`
	tr, err := turnoOpenAI(srv.URL, "clave", "m", "sistema", conversaProba(bruto, "openai"), ferramentaProba(), 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(tr.Chamadas) != 1 || tr.Chamadas[0].Nome != "sen_args" || tr.Chamadas[0].Args != "{}" || tr.Chamadas[0].ID != "call_9" {
		t.Errorf("chamadas mal lidas: %+v", tr.Chamadas)
	}
	if !strings.Contains(tr.Bruto, `"reasoning_content":"outro"`) {
		t.Errorf("o bruto perdeu o razoamento: %s", tr.Bruto)
	}
}

func TestTurnoOpenAIReconstruyeSenBruto(t *testing.T) {
	// Se cambia o provedor a mitade da conversa, non hai bruto que valga: a
	// mensaxe reconstrúese con texto e chamadas.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		msgs := lerCorpo(t, r)["messages"].([]any)
		asis := msgs[2].(map[string]any)
		tcs := asis["tool_calls"].([]any)
		f := tcs[0].(map[string]any)["function"].(map[string]any)
		if f["name"] != "curso_ler" || f["arguments"] != `{"mes":"2025-09"}` {
			t.Errorf("chamada mal reconstruída: %+v", asis)
		}
		w.Write([]byte(`{"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"feito"}}]}`))
	}))
	defer srv.Close()
	tr, err := turnoOpenAI(srv.URL, "clave", "m", "s", conversaProba(`{"rexeitado":true}`, "gemini"), nil, 100, nil)
	if err != nil || tr.Texto != "feito" || len(tr.Chamadas) != 0 {
		t.Fatalf("%v %+v", err, tr)
	}
}

func TestTurnoAnthropicChamadasAFerramentas(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/messages" || r.Header.Get("x-api-key") != "clave" || r.Header.Get("anthropic-version") == "" {
			t.Errorf("ruta/cabeceiras inesperadas: %s", r.URL.Path)
		}
		c := lerCorpo(t, r)
		msgs := c["messages"].([]any)
		ult := msgs[len(msgs)-1].(map[string]any)
		bl := ult["content"].([]any)[0].(map[string]any)
		if ult["role"] != "user" || bl["type"] != "tool_result" || bl["tool_use_id"] != "c1" {
			t.Errorf("tool_result mal formado: %+v", ult)
		}
		asis := msgs[1].(map[string]any)["content"].([]any)
		if asis[1].(map[string]any)["type"] != "tool_use" {
			t.Errorf("o bruto do asistente non se devolveu: %+v", asis)
		}
		tools := c["tools"].([]any)
		if tools[0].(map[string]any)["input_schema"] == nil {
			t.Errorf("falta input_schema: %+v", tools)
		}
		w.Write([]byte(`{"stop_reason":"tool_use","content":[{"type":"text","text":"Mirando"},{"type":"tool_use","id":"toolu_1","name":"curso_ler","input":{"mes":"2025-10"}}]}`))
	}))
	defer srv.Close()

	bruto := `[{"type":"text","text":"Vou mirar."},{"type":"tool_use","id":"c1","name":"curso_ler","input":{"mes":"2025-09"}}]`
	tr, err := turnoAnthropic(srv.URL, "clave", "m", "sistema", conversaProba(bruto, "anthropic"), ferramentaProba(), 4096, nil)
	if err != nil {
		t.Fatal(err)
	}
	if tr.Texto != "Mirando" || len(tr.Chamadas) != 1 || tr.Chamadas[0].ID != "toolu_1" || tr.Chamadas[0].Args != `{"mes":"2025-10"}` {
		t.Errorf("turno mal lido: %+v", tr)
	}
}

func TestTurnoAnthropicTruncado(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"stop_reason":"max_tokens","content":[{"type":"text","text":"cortado"}]}`))
	}))
	defer srv.Close()
	tr, err := turnoAnthropic(srv.URL, "c", "m", "", []IAMensaxe{{Rol: "user", Texto: "hola"}}, nil, 10, nil)
	if err != nil || !tr.Truncado || tr.Texto != "cortado" {
		t.Fatalf("%v %+v", err, tr)
	}
}

func TestTurnoGeminiChamadasAFerramentas(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models/m:generateContent" || r.Header.Get("x-goog-api-key") != "clave" {
			t.Errorf("ruta/cabeceira inesperadas: %s", r.URL.Path)
		}
		c := lerCorpo(t, r)
		conts := c["contents"].([]any)
		ult := conts[len(conts)-1].(map[string]any)
		fr := ult["parts"].([]any)[0].(map[string]any)["functionResponse"].(map[string]any)
		if ult["role"] != "user" || fr["name"] != "curso_ler" || fr["response"].(map[string]any)["resultado"] == nil {
			t.Errorf("functionResponse mal formado: %+v", ult)
		}
		// O bruto (con thoughtSignature) tense que devolver sen tocar.
		asis := conts[1].(map[string]any)
		if asis["role"] != "model" || asis["parts"].([]any)[0].(map[string]any)["thoughtSignature"] != "SIG" {
			t.Errorf("o bruto de Gemini non se devolveu: %+v", asis)
		}
		decl := c["tools"].([]any)[0].(map[string]any)["functionDeclarations"].([]any)
		if _, ten := decl[1].(map[string]any)["parameters"]; ten {
			t.Error("unha ferramenta sen propiedades non debe levar parameters en Gemini")
		}
		w.Write([]byte(`{"candidates":[{"content":{"parts":[{"thought":true,"text":"pensando"},{"text":"Ola"},{"functionCall":{"name":"curso_ler","args":{"mes":"2025-11"}},"thoughtSignature":"S2"}]}}]}`))
	}))
	defer srv.Close()

	bruto := `[{"functionCall":{"name":"curso_ler","args":{"mes":"2025-09"}},"thoughtSignature":"SIG"}]`
	tr, err := turnoGemini(srv.URL, "clave", "m", "sistema", conversaProba(bruto, "gemini"), ferramentaProba(), 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if tr.Texto != "Ola" {
		t.Errorf("o texto de pensamento non se descarta: %q", tr.Texto)
	}
	if len(tr.Chamadas) != 1 || tr.Chamadas[0].Nome != "curso_ler" || tr.Chamadas[0].Args != `{"mes":"2025-11"}` {
		t.Errorf("chamadas mal lidas: %+v", tr.Chamadas)
	}
	if !strings.Contains(tr.Bruto, "S2") {
		t.Errorf("o bruto perdeu a sinatura: %s", tr.Bruto)
	}
}

func TestTurnoErrosDoProvedor(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		w.Write([]byte(`{"error":{"message":"modelo inexistente"}}`))
	}))
	defer srv.Close()
	msgs := []IAMensaxe{{Rol: "user", Texto: "hola"}}
	for nome, fn := range map[string]func() error{
		"openai":    func() error { _, e := turnoOpenAI(srv.URL, "c", "m", "s", msgs, nil, 0, nil); return e },
		"anthropic": func() error { _, e := turnoAnthropic(srv.URL, "c", "m", "s", msgs, nil, 10, nil); return e },
		"gemini":    func() error { _, e := turnoGemini(srv.URL, "c", "m", "s", msgs, nil, 0, nil); return e },
	} {
		if err := fn(); err == nil || !strings.Contains(err.Error(), "modelo inexistente") {
			t.Errorf("%s: erro = %v", nome, err)
		}
	}
}

func TestChamarIAConversaExixeConfiguracion(t *testing.T) {
	if _, err := chamarIAConversa(Settings{}, "s", []IAMensaxe{{Rol: "user", Texto: "x"}}, nil); err == nil || !strings.Contains(err.Error(), "clave") {
		t.Errorf("sen clave debería fallar pedindo a clave: %v", err)
	}
	if _, err := chamarIAConversa(Settings{IAAPIKey: "k"}, "s", nil, nil); err == nil || !strings.Contains(err.Error(), "modelo") {
		t.Errorf("sen modelo debería fallar pedindo o modelo: %v", err)
	}
}

func TestIATurnoAsistenteEnviaSistemaEFerramentas(t *testing.T) {
	var sistema string
	var nTools int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := lerCorpo(t, r)
		sistema = c["messages"].([]any)[0].(map[string]any)["content"].(string)
		nTools = len(c["tools"].([]any))
		w.Write([]byte(`{"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"Ola!"}}]}`))
	}))
	defer srv.Close()
	a := &App{settings: Settings{IAProvedor: "openai", IABaseURL: srv.URL, IAAPIKey: "k", IAModel: "m"}}
	tr, err := a.IATurnoAsistente(IATurnoRequest{Mensaxes: []IAMensaxe{{Rol: "user", Texto: "ola"}}, Estado: "Curso 2025/2026 con 3 tarefas"})
	if err != nil || tr.Texto != "Ola!" || tr.Provedor != "openai" {
		t.Fatalf("%v %+v", err, tr)
	}
	if !strings.Contains(sistema, "Curso 2025/2026 con 3 tarefas") || !strings.Contains(sistema, "ferramentas") {
		t.Errorf("o prompt non leva o estado: %q", sistema[:200])
	}
	if nTools != len(a.IAFerramentas()) || nTools < 20 {
		t.Errorf("as ferramentas non chegaron todas: %d", nTools)
	}
}

// ---------- Catálogo de ferramentas ----------

func TestFerramentasCatalogoCoherente(t *testing.T) {
	a := &App{}
	vistos := map[string]bool{}
	for _, f := range a.IAFerramentas() {
		if vistos[f.Nome] {
			t.Errorf("nome duplicado: %s", f.Nome)
		}
		vistos[f.Nome] = true
		if !regexpNomeFerramenta.MatchString(f.Nome) {
			t.Errorf("nome non válido para os provedores: %q", f.Nome)
		}
		if strings.TrimSpace(f.Descricion) == "" {
			t.Errorf("%s sen descrición", f.Nome)
		}
		if f.Onde != "frontend" && f.Onde != "backend" {
			t.Errorf("%s: Onde = %q", f.Nome, f.Onde)
		}
		if f.Tipo != tipoLectura && f.Tipo != tipoEscritura && f.Tipo != tipoConfirmar {
			t.Errorf("%s: Tipo = %q", f.Nome, f.Tipo)
		}
		if f.Parametros != "" {
			var e map[string]any
			if err := json.Unmarshal([]byte(f.Parametros), &e); err != nil || e["type"] != "object" {
				t.Errorf("%s: parámetros non son un JSON Schema de obxecto: %v", f.Nome, err)
			}
		}
	}
	// As que controlan o programa polo menos: curso, tarefas, informe, plantillas, opcións, web, usuario.
	for _, n := range []string{"curso_ler", "tarefa_editar", "informe_xerar", "informe_exportar", "plantilla_gardar", "opcions_cambiar", "web_buscar", "usuario_preguntar", "usuario_avisar"} {
		if !vistos[n] {
			t.Errorf("falta a ferramenta %s", n)
		}
	}
}

func TestFerramentasQueNonSeDesfanPidenConfirmacion(t *testing.T) {
	tipos := map[string]string{}
	for _, f := range (&App{}).IAFerramentas() {
		tipos[f.Nome] = f.Tipo
	}
	for _, n := range []string{"plantilla_gardar", "plantilla_eliminar", "curso_novo"} {
		if tipos[n] != tipoConfirmar {
			t.Errorf("%s ten que pedir confirmación, é %q", n, tipos[n])
		}
	}
}

func TestIAChamarFerramentaDescoñecidaEArgsInvalidos(t *testing.T) {
	a := &App{}
	if r := a.IAChamarFerramenta("non_existe", "{}"); !r.Erro || !strings.Contains(r.Contido, "descoñecida") {
		t.Errorf("%+v", r)
	}
	if r := a.IAChamarFerramenta("opcions_ler", "{isto non é json"); !r.Erro {
		t.Errorf("uns argumentos rotos deberían dar erro: %+v", r)
	}
}

func TestOpcionsLerNonExponAClave(t *testing.T) {
	a := &App{settings: Settings{IAAPIKey: "SEGREDO-123", IABaseURL: "https://exemplo.test", IAModel: "m", IAProvedor: "openai"}}
	r := a.IAChamarFerramenta("opcions_ler", "")
	if r.Erro {
		t.Fatal(r.Contido)
	}
	if strings.Contains(r.Contido, "SEGREDO-123") || strings.Contains(r.Contido, "exemplo.test") {
		t.Errorf("opcions_ler filtrou a clave ou o enderezo: %s", r.Contido)
	}
	if !strings.Contains(r.Contido, `"iaTenClave": true`) {
		t.Errorf("debería dicir que hai clave: %s", r.Contido)
	}
}

func TestOpcionsCambiarSoAceptaAsPermitidas(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	a := &App{settings: Settings{IAAPIKey: "SEGREDO", IABaseURL: "https://oficial.test", IAProvedor: "openai", IAModel: "m"}}

	// A clave, o provedor e o enderezo non se poden tocar (un texto lido da
	// web non pode desviar a clave a outro servidor).
	for _, args := range []string{`{"iaApiKey":"x"}`, `{"iaBaseUrl":"https://malo.test"}`, `{"iaProvedor":"gemini"}`, `{"iaPreset":"outro"}`} {
		if r := a.IAChamarFerramenta("opcions_cambiar", args); !r.Erro {
			t.Errorf("%s debería rexeitarse: %+v", args, r)
		}
	}
	if a.settings.IAAPIKey != "SEGREDO" || a.settings.IABaseURL != "https://oficial.test" || a.settings.IAProvedor != "openai" {
		t.Errorf("os datos da IA cambiaron: %+v", a.settings)
	}

	r := a.IAChamarFerramenta("opcions_cambiar", `{"idioma":"es","latexEngine":"xelatex","exportarDocx":true,"xerarAoGardar":false,"iaMaxTokens":4000}`)
	if r.Erro {
		t.Fatal(r.Contido)
	}
	s := a.settings
	if s.Idioma != "es" || s.LatexEngine != "xelatex" || !s.ExportarDocx || s.XerarAoGardarResolto() || s.IAMaxTokens != 4000 {
		t.Errorf("non se aplicaron: %+v", s)
	}
	if s.IAAPIKey != "SEGREDO" {
		t.Error("a clave desapareceu")
	}
	// Persistiu en disco.
	if cargada := loadSettings(); cargada.Idioma != "es" || !cargada.ExportarDocx {
		t.Errorf("non se gardou en disco: %+v", cargada)
	}

	for _, args := range []string{`{"idioma":"klingon"}`, `{"latexEngine":"word"}`, `{"exportarOdt":"quizais"}`, `{"iaModel":"  "}`, `{}`} {
		if r := a.IAChamarFerramenta("opcions_cambiar", args); !r.Erro {
			t.Errorf("%s debería dar erro", args)
		}
	}
}

func TestPlantillasComoFerramentas(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	a := &App{}
	r := a.IAChamarFerramenta("plantillas_listar", "")
	if r.Erro || !strings.Contains(r.Contido, `"plantillas"`) {
		t.Fatalf("%+v", r)
	}
	var lista struct {
		Plantillas []struct {
			ID   string `json:"id"`
			Nome string `json:"nome"`
		} `json:"plantillas"`
	}
	if err := json.Unmarshal([]byte(r.Contido), &lista); err != nil || len(lista.Plantillas) == 0 {
		t.Fatalf("non hai plantillas de fábrica: %v %s", err, r.Contido)
	}
	id := lista.Plantillas[0].ID

	// Ler completa.
	r = a.IAChamarFerramenta("plantilla_ler", `{"id":"`+id+`"}`)
	if r.Erro || !strings.Contains(r.Contido, "{{CORPO}}") {
		t.Fatalf("plantilla_ler: %+v", r)
	}
	if r := a.IAChamarFerramenta("plantilla_ler", `{"id":"non-existe"}`); !r.Erro {
		t.Error("un id inexistente debería dar erro")
	}

	// Modificar só un campo conserva os outros.
	r = a.IAChamarFerramenta("plantilla_gardar", `{"id":"`+id+`","descricion":"Descrición nova pola IA"}`)
	if r.Erro {
		t.Fatalf("plantilla_gardar: %s", r.Contido)
	}
	var gardada struct {
		ID   string `json:"id"`
		Nota string `json:"nota"`
	}
	json.Unmarshal([]byte(r.Contido), &gardada)
	// A primeira é de exemplo: editala crea unha copia propia e a orixinal queda intacta.
	if gardada.ID == id || gardada.Nota == "" {
		t.Errorf("editar unha de exemplo debería bifurcar e avisalo: %s", r.Contido)
	}
	p, err := plantillaPorID(a, gardada.ID)
	if err != nil || p.Descricion != "Descrición nova pola IA" || !strings.Contains(p.Latex, "{{CORPO}}") {
		t.Fatalf("a modificación perdeu campos: %v %+v", err, p.Descricion)
	}
	if orixinal, err := plantillaPorID(a, id); err != nil || orixinal.Descricion == "Descrición nova pola IA" {
		t.Errorf("a plantilla de exemplo non debería cambiar: %v", err)
	}
	// Unha propia (non de exemplo) edítase en sitio.
	r = a.IAChamarFerramenta("plantilla_gardar", `{"id":"`+gardada.ID+`","descricion":"Outra vez"}`)
	if r.Erro || !strings.Contains(r.Contido, gardada.ID) || strings.Contains(r.Contido, "nota") {
		t.Errorf("unha propia debería editarse no sitio: %+v", r)
	}
	id = gardada.ID

	// Un LaTeX sen {{CORPO}} rexéitase coa mensaxe do backend.
	if r := a.IAChamarFerramenta("plantilla_gardar", `{"id":"`+id+`","latex":"\\pagestyle{empty}"}`); !r.Erro {
		t.Error("unha plantilla sen {{CORPO}} non se pode gardar")
	}

	// Crear unha nova, duplicar e eliminar.
	r = a.IAChamarFerramenta("plantilla_gardar", `{"nome":"Nova da IA","latex":"\\pagestyle{empty}\n{{CORPO}}","variables":[{"nome":"telefono","etiqueta":"Teléfono","valor":"986"}]}`)
	if r.Erro {
		t.Fatal(r.Contido)
	}
	var nova struct{ ID string }
	json.Unmarshal([]byte(r.Contido), &nova)
	if nova.ID == "" {
		t.Fatalf("sen id: %s", r.Contido)
	}
	if r := a.IAChamarFerramenta("plantilla_duplicar", `{"id":"`+nova.ID+`"}`); r.Erro {
		t.Fatal(r.Contido)
	}
	if r := a.IAChamarFerramenta("plantilla_eliminar", `{"id":"`+nova.ID+`"}`); r.Erro {
		t.Fatal(r.Contido)
	}
	if _, err := plantillaPorID(a, nova.ID); err == nil {
		t.Error("a plantilla eliminada aínda existe")
	}
}

// ---------- Web ----------

func TestEnderezoPublico(t *testing.T) {
	for ip, quero := range map[string]bool{
		"8.8.8.8": true, "93.184.216.34": true, "2606:4700::1111": true,
		"127.0.0.1": false, "10.0.0.5": false, "192.168.1.1": false, "172.16.0.1": false, "169.254.169.254": false,
		"100.64.0.1": false, "0.0.0.0": false, "::1": false, "fe80::1": false, "fd00::1": false, "::ffff:127.0.0.1": false,
	} {
		if got := enderezoPublico(netip.MustParseAddr(ip)); got != quero {
			t.Errorf("enderezoPublico(%s) = %v, quería %v", ip, got, quero)
		}
	}
}

func TestWebConsultarRexeitaEnderezosLocais(t *testing.T) {
	for _, u := range []string{
		"http://127.0.0.1:8080/", "http://localhost/admin", "http://192.168.0.1/", "http://[::1]/",
		"http://169.254.169.254/latest/meta-data/", "file:///etc/passwd", "ftp://exemplo.org/x", "gopher://x", "http://servidor.local/",
	} {
		if _, err := webConsultar(u, 0, 0); err == nil {
			t.Errorf("%s debería rexeitarse", u)
		}
	}
}

func TestWebConsultarLeHTMLeParteAsPaxinas(t *testing.T) {
	webPermitirLocal = true
	defer func() { webPermitirLocal = false }()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redir" {
			http.Redirect(w, r, "/lei", http.StatusFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(`<html><head><title>x</title><style>.a{}</style></head><body><nav>MENU</nav>
			<script>var x = "SCRIPT";</script><h1>Decreto 1/2025</h1><p>Artigo   1.  Obxecto.</p>
			<ul><li>Primeiro</li><li>Segundo</li></ul><footer>PE</footer></body></html>`))
	}))
	defer srv.Close()

	txt, err := webConsultar(srv.URL+"/redir", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, quero := range []string{"Decreto 1/2025", "Artigo 1. Obxecto.", "- Primeiro", "- Segundo"} {
		if !strings.Contains(txt, quero) {
			t.Errorf("falta %q en %q", quero, txt)
		}
	}
	for _, non := range []string{"MENU", "SCRIPT", "PE\n", ".a{}"} {
		if strings.Contains(txt, non) {
			t.Errorf("non debería levar %q: %q", non, txt)
		}
	}
	// En partes: lese un anaco e avísase de como seguir.
	parte, err := webConsultar(srv.URL, 0, 10)
	if err != nil || !strings.Contains(parte, "inicio=10") {
		t.Errorf("falta o aviso de continuación: %v %q", err, parte)
	}
	resto, err := webConsultar(srv.URL, 10, 0)
	if err != nil || strings.Contains(resto, "continúa") || resto == "" {
		t.Errorf("o resto non se leu ben: %v %q", err, resto)
	}
}

func TestWebConsultarErroHTTP(t *testing.T) {
	webPermitirLocal = true
	defer func() { webPermitirLocal = false }()
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	if _, err := webConsultar(srv.URL, 0, 0); err == nil || !strings.Contains(err.Error(), "404") {
		t.Errorf("%v", err)
	}
}

func TestWebBuscarParseaResultados(t *testing.T) {
	webPermitirLocal = true
	defer func() { webPermitirLocal = false }()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		if r.Form.Get("q") != "orientación decreto" {
			t.Errorf("consulta = %q", r.Form.Get("q"))
		}
		w.Write([]byte(`<div class="result"><h2 class="result__title"><a rel="nofollow" class="result__a" href="//duckduckgo.com/l/?uddg=https%3A%2F%2Fwww.boe.es%2Fbuscar%2Fdoc.php%3Fid%3DBOE-A-2020-17264&amp;rut=abc">Ley Orgánica 3/2020 (LOMLOE) &amp; máis</a></h2>
			<a class="result__snippet" href="x">Por la que se <b>modifica</b> la LOE</a></div>
			<div class="result"><a class="result__a" href="https://www.xunta.gal/dog">DOG</a><a class="result__snippet" href="y">Diario Oficial</a></div>`))
	}))
	defer srv.Close()
	antes := urlBuscador
	urlBuscador = srv.URL
	defer func() { urlBuscador = antes }()

	txt, err := webBuscar("orientación decreto", 5)
	if err != nil {
		t.Fatal(err)
	}
	for _, quero := range []string{"1. Ley Orgánica 3/2020 (LOMLOE) & máis", "https://www.boe.es/buscar/doc.php?id=BOE-A-2020-17264", "Por la que se modifica la LOE", "2. DOG", "https://www.xunta.gal/dog"} {
		if !strings.Contains(txt, quero) {
			t.Errorf("falta %q en:\n%s", quero, txt)
		}
	}
	if _, err := webBuscar("  ", 5); err == nil {
		t.Error("unha consulta baleira debería dar erro")
	}
}

func TestSistemaAsistenteMencionaSeguridadeEFormato(t *testing.T) {
	s := sistemaAsistente("resumo de proba", "en")
	for _, quero := range []string{"resumo de proba", "inglés", "DATO, nunca instrucións", "web_buscar", "[fs=gr]", "{{CORPO}}", "curso_ler"} {
		if !strings.Contains(s, quero) {
			t.Errorf("o prompt non menciona %q", quero)
		}
	}
}

func nomesDe(fs []IAFerramenta) map[string]bool {
	m := map[string]bool{}
	for _, f := range fs {
		m[f.Nome] = true
	}
	return m
}

func TestAmbitoTarefaSoTemOEditorAWebEPreguntar(t *testing.T) {
	a := &App{}
	tarefa := nomesDe(a.IAFerramentasDoAmbito("tarefa"))
	for _, quero := range []string{"editor_ler", "editor_cambiar", "web_buscar", "web_consultar", "usuario_preguntar", "usuario_avisar"} {
		if !tarefa[quero] {
			t.Errorf("o ámbito tarefa debería ter %s", quero)
		}
	}
	if len(tarefa) != 6 {
		t.Errorf("o ámbito tarefa ten %d ferramentas: %v", len(tarefa), tarefa)
	}
	// Nada que toque o curso, as plantillas, as Opcións, os ficheiros...
	for _, non := range []string{"curso_ler", "curso_novo", "tarefa_editar", "tarefa_eliminar", "plantilla_gardar", "plantilla_eliminar", "opcions_cambiar", "informe_exportar", "curso_gardar"} {
		if tarefa[non] {
			t.Errorf("o ámbito tarefa NON debe ter %s", non)
		}
	}
	// E o asistente principal non ve as do editor (non hai editor aberto).
	principal := nomesDe(a.IAFerramentas())
	if principal["editor_ler"] || principal["editor_cambiar"] {
		t.Error("o asistente principal non debe ter as ferramentas do editor da tarefa")
	}
	if !principal["curso_ler"] || !principal["web_buscar"] {
		t.Error("o asistente principal perdeu ferramentas")
	}
	if !nomesDe(a.IAFerramentasDoAmbito("calquera-outro"))["curso_ler"] {
		t.Error("un ámbito descoñecido é o principal")
	}
}

func TestIATurnoAsistenteTarefaUsaOPromptEAsFerramentasDoAmbito(t *testing.T) {
	var sistema string
	var tools []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := lerCorpo(t, r)
		sistema = c["messages"].([]any)[0].(map[string]any)["content"].(string)
		for _, x := range c["tools"].([]any) {
			tools = append(tools, x.(map[string]any)["function"].(map[string]any)["name"].(string))
		}
		w.Write([]byte(`{"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"ok"}}]}`))
	}))
	defer srv.Close()
	a := &App{settings: Settings{IAProvedor: "openai", IABaseURL: srv.URL, IAAPIKey: "k", IAModel: "m"}}
	_, err := a.IATurnoAsistente(IATurnoRequest{Mensaxes: []IAMensaxe{{Rol: "user", Texto: "x"}}, Estado: "Tarefa «Avaliación»", Idioma: "es", Ambito: "tarefa"})
	if err != nil {
		t.Fatal(err)
	}
	for _, quero := range []string{"EDICIÓN DUNHA TAREFA", "Tarefa «Avaliación»", "editor_ler", "site:boe.es", "DOG", "castelán", "Referencias normativas"} {
		if !strings.Contains(sistema, quero) {
			t.Errorf("o prompt da tarefa non menciona %q", quero)
		}
	}
	if strings.Contains(sistema, "curso_ler") {
		t.Error("o prompt da tarefa non debe falar das ferramentas do curso")
	}
	if len(tools) != 6 {
		t.Errorf("ferramentas enviadas: %v", tools)
	}
}

func TestAmbitoPlantillaSoTenAsFerramentasDoEditor(t *testing.T) {
	a := &App{}
	nomes := map[string]bool{}
	for _, f := range a.IAFerramentasDoAmbito(ambitoPlantilla) {
		nomes[f.Nome] = true
	}
	for _, quero := range []string{"plantilla_editor_ler", "plantilla_editor_cambiar", "plantilla_previsualizar", "plantilla_comprobar",
		"adxunto_gardar_en_plantilla", "web_buscar", "usuario_preguntar"} {
		if !nomes[quero] {
			t.Errorf("falta %s no ámbito plantilla", quero)
		}
	}
	// Nada do curso, das Opcións nin do resto da biblioteca (gardar ao disco é decisión do usuario).
	for _, non := range []string{"plantilla_gardar", "plantilla_eliminar", "plantilla_escoller", "opcions_cambiar", "opcions_ler",
		"curso_ler", "tarefa_editar", "editor_cambiar", "informe_xerar", "sistema_estado"} {
		if nomes[non] {
			t.Errorf("%s non debe estar no ámbito plantilla", non)
		}
	}
	// As do editor de plantilla non se ven no asistente principal nin no da tarefa.
	for _, ambito := range []string{"", ambitoTarefa} {
		for _, f := range a.IAFerramentasDoAmbito(ambito) {
			if strings.HasPrefix(f.Nome, "plantilla_editor_") {
				t.Errorf("%s aparece no ámbito %q", f.Nome, ambito)
			}
		}
	}
	if p := sistemaAsistentePlantilla("estado", "gl"); !strings.Contains(p, "plantilla_editor_ler") || !strings.Contains(p, "interlinado") {
		t.Error("o prompt do asistente da plantilla está incompleto")
	}
}
