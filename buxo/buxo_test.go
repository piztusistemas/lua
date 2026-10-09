package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"edicion/api"
)

const pdfMostra = "%PDF-1.4\n1 0 obj<<>>endobj\ntrailer<<>>\n%%EOF\n"

// sidecarFalso imita markitdown-sidecar: escribe «# <nome>» en <saida>/<nome>.md
// e falla cos ficheiros que se chaman «malo».
func sidecarFalso(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("sidecar falso en shell")
	}
	p := filepath.Join(t.TempDir(), "sidecar.sh")
	script := `#!/bin/sh
out="$1"; shift
for f in "$@"; do
  n=$(basename "$f" .pdf)
  case "$n" in
    malo*) printf '{"input":"%s","ok":false,"error":"ValueError: sen texto"}\n' "$f";;
    *) printf '# %s\n\nTexto.\n' "$n" > "$out/$n.md"; printf '{"input":"%s","output":"%s/%s.md","ok":true}\n' "$f" "$out" "$n";;
  esac
done
`
	if err := os.WriteFile(p, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func novoMestreProba(t *testing.T) (*Mestre, *Rexistro) {
	t.Helper()
	raiz := filepath.Join(t.TempDir(), "boveda")
	log := NovoRexistro(filepath.Join(raiz, ficheiroLog), nil)
	m := NovoMestre(raiz, "Fotosíntese", log, nil)
	m.sidecar = sidecarFalso(t)
	t.Cleanup(m.Pechar)
	return m, log
}

func esperar(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("tempo esgotado esperando a condición")
}

func servidorPDF(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/a.pdf", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(pdfMostra)) })
	mux.HandleFunc("/malo.pdf", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(pdfMostra)) })
	mux.HandleFunc("/non-pdf.pdf", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("<html>erro</html>")) })
	mux.HandleFunc("/404.pdf", func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) })
	mux.HandleFunc("/pax", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<a href="a.pdf">Tema A</a> <a href="/malo.pdf">Malo</a> <a href="https://x.org/y">outra</a><a href="a.pdf#f">dup</a>`))
	})
	s := httptest.NewServer(mux)
	t.Cleanup(s.Close)
	return s
}

func TestEstruturaSoCandoHaiPDF(t *testing.T) {
	m, _ := novoMestreProba(t)
	if _, err := os.Stat(m.raiz); err == nil {
		t.Fatal("non se debe crear nada ata que haxa PDF")
	}
	pdf := filepath.Join(t.TempDir(), "tema.pdf")
	os.WriteFile(pdf, []byte(pdfMostra), 0o644)
	m.EngadirLocais([]string{pdf})
	esperar(t, func() bool { return m.Contas()[estFeito] == 1 })
	for _, d := range []string{"pdf/tema.pdf", "raw/tema.md", "buxo.md", "wiki", "buxo.log"} {
		if _, err := os.Stat(filepath.Join(m.raiz, d)); err != nil {
			t.Errorf("falta %s", d)
		}
	}
	md, _ := os.ReadFile(filepath.Join(m.raiz, "raw", "tema.md"))
	if !strings.HasPrefix(string(md), "---\npdf: \"pdf/tema.pdf\"\n---") || !strings.Contains(string(md), "# tema") {
		t.Errorf("raw/tema.md sen cabeceira de fonte:\n%s", md)
	}
	if e, _ := os.ReadFile(filepath.Join(m.raiz, "buxo.md")); !strings.Contains(string(e), "Fotosíntese") {
		t.Error("buxo.md debe levar a materia")
	}
}

func TestFalloNonParaOsDemais(t *testing.T) {
	m, log := novoMestreProba(t)
	s := servidorPDF(t)
	m.EngadirURLs([]string{s.URL + "/a.pdf", s.URL + "/malo.pdf", s.URL + "/non-pdf.pdf", s.URL + "/404.pdf"})
	esperar(t, func() bool { c := m.Contas(); return c[estFeito] == 1 && c[estErro] == 3 })
	var erros int
	for _, l := range log.Historial() {
		if l.Nivel == "erro" {
			erros++
		}
	}
	if erros != 3 {
		t.Errorf("deberían quedar 3 erros no rexistro, hai %d: %v", erros, log.Historial())
	}
	if _, err := os.Stat(filepath.Join(m.raiz, "raw", "a.md")); err != nil {
		t.Error("o PDF bo debe converterse")
	}
	if l, _ := os.ReadFile(filepath.Join(m.raiz, ficheiroLog)); !strings.Contains(string(l), "sen texto") {
		t.Errorf("o log en disco debe ter o erro:\n%s", l)
	}
	if ents, _ := os.ReadDir(filepath.Join(m.raiz, "pdf")); len(ents) != 2 {
		t.Errorf("en pdf/ só deben estar os dous PDF válidos (a.pdf, malo.pdf): %v", ents)
	}
}

func TestNomesDuplicadosENomeSeguro(t *testing.T) {
	if got := nomePDFSeguro("../../etc/pas swd?.PDF"); strings.ContainsAny(got, "/\\?") || !strings.HasSuffix(got, ".pdf") {
		t.Errorf("nome inseguro: %q", got)
	}
	d := t.TempDir()
	os.WriteFile(filepath.Join(d, "a.pdf"), nil, 0o644)
	if got := nomeLibre(d, "a.pdf"); got != "a-1.pdf" {
		t.Errorf("duplicado: %q", got)
	}
	if _, err := DescargarPDF(context.Background(), nil, "file:///etc/passwd", d); err == nil {
		t.Error("só http/https")
	}
}

func TestRecuperarCola(t *testing.T) {
	m, _ := novoMestreProba(t)
	CrearEstrutura(m.raiz, "x")
	os.WriteFile(filepath.Join(m.raiz, "pdf", "feito.pdf"), []byte(pdfMostra), 0o644)
	os.WriteFile(filepath.Join(m.raiz, "raw", "feito.md"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(m.raiz, "pdf", "novo.pdf"), []byte(pdfMostra), 0o644)
	m.Recuperar()
	esperar(t, func() bool { c := m.Contas(); return c[estFeito] == 2 })
}

func TestProcurarPDFsSenIA(t *testing.T) {
	m, _ := novoMestreProba(t)
	s := servidorPDF(t)
	m.ProcurarPDFs(context.Background(), nil, []string{s.URL + "/pax", "isto non é url"}, "")
	esperar(t, func() bool { c := m.Contas(); return c[estFeito] == 1 && c[estErro] == 1 })
	if n := len(m.Items()); n != 2 {
		t.Errorf("a ligazón repetida non debe duplicarse: %d items", n)
	}
}

// ---------- API falsa de Lua ----------

type luaFalso struct {
	mu       sync.Mutex
	turnos   int
	escritas []string
	pendente bool
	url      string
	token    string
}

func novoLuaFalso(t *testing.T) *luaFalso {
	t.Helper()
	l := &luaFalso{pendente: true}
	s := api.Novo("falso")
	s.Rexistrar("GET /ia/estado", "ia.usar", func(p *api.Peticion) (any, error) {
		return map[string]any{"configurada": true, "modelo": "m"}, nil
	})
	s.Rexistrar("GET /bovedas/ferramentas", "boveda.ler", func(p *api.Peticion) (any, error) {
		return []IAFerramenta{{Nome: "boveda_escribir", Descricion: "x", Parametros: `{"type":"object"}`}}, nil
	})
	s.Rexistrar("GET /bovedas/{id}/pendentes", "boveda.ler", func(p *api.Peticion) (any, error) {
		l.mu.Lock()
		defer l.mu.Unlock()
		if l.pendente {
			return map[string]any{"pendentes": []string{"raw/a.md"}, "total": 1}, nil
		}
		return map[string]any{"pendentes": []string{}, "total": 0}, nil
	})
	s.Rexistrar("PUT /bovedas/{id}/escribir", "boveda.escribir", func(p *api.Peticion) (any, error) {
		var m map[string]any
		_ = p.Corpo(&m)
		l.mu.Lock()
		l.escritas = append(l.escritas, p.Param("id")+":"+m["ruta"].(string))
		l.pendente = false
		l.mu.Unlock()
		return "páxina creada", nil
	})
	s.Rexistrar("POST /ia/turno", "ia.usar", func(p *api.Peticion) (any, error) {
		var pet struct{ Mensaxes []IAMensaxe }
		_ = p.Corpo(&pet)
		l.mu.Lock()
		defer l.mu.Unlock()
		l.turnos++
		ultima := pet.Mensaxes[len(pet.Mensaxes)-1]
		if ultima.Rol == "user" && strings.Contains(ultima.Texto, "INGEST") {
			args, _ := json.Marshal(map[string]string{"ruta": "wiki/fontes/a.md", "contido": "# A"})
			return IATurno{Chamadas: []IAChamada{{ID: "1", Nome: "boveda_escribir", Args: string(args)}}, Bruto: "{}", Provedor: "x"}, nil
		}
		return IATurno{Texto: "Feito."}, nil
	})
	u, err := s.Iniciar()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Parar)
	l.url, l.token = u, s.NovoToken("buxo", "boveda.ler", "boveda.escribir", "ia.usar")
	return l
}

func TestConstruirPolaAPI(t *testing.T) {
	l := novoLuaFalso(t)
	m, log := novoMestreProba(t)
	CrearEstrutura(m.raiz, "x")
	os.WriteFile(filepath.Join(m.raiz, "raw", "a.md"), []byte("# A"), 0o644)
	ia := &ClienteIA{C: &api.Cliente{URL: l.url, Token: l.token}, Boveda: "buxo"}
	if err := m.Construir(context.Background(), ia); err != nil {
		t.Fatal(err)
	}
	if len(l.escritas) != 1 || l.escritas[0] != "buxo:wiki/fontes/a.md" {
		t.Errorf("escritas: %v", l.escritas)
	}
	if l.turnos < 3 { // ingest (ferramenta + resposta) + revisión
		t.Errorf("turnos: %d", l.turnos)
	}
	tar := m.Tarefas()
	if len(tar) != 2 || tar[0].Nome != "a.md" || tar[0].Estado != tarFeita || tar[0].Pasos != 1 || tar[1].Estado != tarFeita {
		t.Errorf("tarefas: %+v", tar)
	}
	ult := log.Historial()[len(log.Historial())-1]
	if !strings.Contains(ult.Texto, "Rematado") {
		t.Errorf("último rexistro: %v", ult)
	}
}

func TestConstruirSenPDFOuSenIA(t *testing.T) {
	m, _ := novoMestreProba(t)
	if err := m.Construir(context.Background(), nil); err == nil {
		t.Error("sen IA debe fallar")
	}
	l := novoLuaFalso(t)
	ia := &ClienteIA{C: &api.Cliente{URL: l.url, Token: l.token}, Boveda: "buxo"}
	if err := m.Construir(context.Background(), ia); err == nil || !strings.Contains(err.Error(), "PDF") {
		t.Errorf("sen raw/ debe pedir PDF: %v", err)
	}
	if _, err := os.Stat(m.raiz); err == nil {
		t.Error("non se debe crear a bóveda sen PDF")
	}
}

func TestExtraerLigazons(t *testing.T) {
	base, _ := http_parse("https://exemplo.org/a/b/")
	pdfs, pax := extraerLigazons([]byte(`<a href="x.PDF">X <b>doc</b></a><a href="../y.html">Y</a><a href="mailto:a@b">m</a>`), base)
	if len(pdfs) != 1 || pdfs[0].URL != "https://exemplo.org/a/b/x.PDF" || pdfs[0].Texto != "X doc" {
		t.Errorf("pdfs: %v", pdfs)
	}
	if len(pax) != 1 || pax[0].URL != "https://exemplo.org/a/y.html" {
		t.Errorf("páxinas: %v", pax)
	}
}

func http_parse(s string) (*url.URL, error) { return url.Parse(s) }

// Con o sidecar real (build/bin, ver sidecar/build.sh) e un PDF de verdade.
// Fíxase BUXO_PDF_PROBA para indicar o PDF; sen iso (ou sen sidecar) sáltase.
func TestSidecarReal(t *testing.T) {
	sc := filepath.Join("build", "bin", "markitdown-sidecar")
	pdf := os.Getenv("BUXO_PDF_PROBA")
	if _, err := os.Stat(sc); err != nil || pdf == "" {
		t.Skip("sen sidecar real ou BUXO_PDF_PROBA")
	}
	raiz := filepath.Join(t.TempDir(), "b")
	m := NovoMestre(raiz, "x", NovoRexistro("", nil), nil)
	m.sidecar, _ = filepath.Abs(sc)
	defer m.Pechar()
	m.EngadirLocais([]string{pdf, "/non/existe.pdf"})
	esperar(t, func() bool { c := m.Contas(); return c[estFeito] == 1 && c[estErro] == 1 })
	ents, _ := os.ReadDir(filepath.Join(raiz, "raw"))
	if len(ents) != 1 {
		t.Fatalf("raw/: %v", ents)
	}
	b, _ := os.ReadFile(filepath.Join(raiz, "raw", ents[0].Name()))
	if len(b) < 200 {
		t.Errorf("Markdown demasiado curto: %d bytes", len(b))
	}
}

// Mentres a IA tarda, o rexistro avisa de que segue traballando.
func TestPulsoMentresAIATarda(t *testing.T) {
	antes := pulsoIA
	pulsoIA = 30 * time.Millisecond
	defer func() { pulsoIA = antes }()
	s := api.Novo("lento")
	s.Rexistrar("POST /ia/turno", "ia.usar", func(p *api.Peticion) (any, error) {
		time.Sleep(150 * time.Millisecond)
		return IATurno{Texto: "ok"}, nil
	})
	u, _ := s.Iniciar()
	defer s.Parar()
	ia := &ClienteIA{C: &api.Cliente{URL: u, Token: s.NovoToken("t", "ia.usar")}}
	var avisos []string
	var mu sync.Mutex
	_, err := ia.turnoComPulso(context.Background(), "", []IAMensaxe{{Rol: "user", Texto: "x"}}, nil, func(m string) { mu.Lock(); avisos = append(avisos, m); mu.Unlock() })
	if err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(avisos) < 3 || !strings.Contains(avisos[0], "agardando") || !strings.Contains(avisos[len(avisos)-1], "segue traballando") {
		t.Errorf("avisos: %v", avisos)
	}
}

// Coa instrución do usuario a IA filtra os PDF dunha páxina (só o que escolle entra).
func TestProcurarPDFsConInstrucion(t *testing.T) {
	m, log := novoMestreProba(t)
	s := servidorPDF(t)
	var criterio string
	var mu sync.Mutex
	fal := api.Novo("falso")
	fal.Rexistrar("POST /ia/turno", "ia.usar", func(p *api.Peticion) (any, error) {
		var pet struct{ Mensaxes []IAMensaxe }
		_ = p.Corpo(&pet)
		mu.Lock()
		criterio = pet.Mensaxes[0].Texto
		mu.Unlock()
		return IATurno{Texto: "[0]"}, nil // só o primeiro: a.pdf
	})
	u, _ := fal.Iniciar()
	defer fal.Parar()
	ia := &ClienteIA{C: &api.Cliente{URL: u, Token: fal.NovoToken("t", "ia.usar")}}
	m.ProcurarPDFs(context.Background(), ia, []string{s.URL + "/pax"}, "só o tema A")
	esperar(t, func() bool { return m.Contas()[estFeito] == 1 })
	if n := len(m.Items()); n != 1 {
		t.Errorf("só debía entrar un PDF: %d", n)
	}
	mu.Lock()
	defer mu.Unlock()
	if !strings.Contains(criterio, "CRITERIO DO USUARIO (prevalece): só o tema A") || !strings.Contains(criterio, "Tema A") {
		t.Errorf("o prompt non leva a instrución nin as ligazóns:\n%s", criterio)
	}
	// Sen IA avísase de que a instrución non se aplica.
	m.ProcurarPDFs(context.Background(), nil, []string{s.URL + "/pax"}, "algo")
	var avisou bool
	for _, l := range log.Historial() {
		avisou = avisou || strings.Contains(l.Texto, "Sen IA non se pode aplicar")
	}
	if !avisou {
		t.Error("falta o aviso de instrución sen IA")
	}
}

// A licenza de Buxo (MIT) e o aviso de MarkItDown poden consultarse desde a interface.
func TestLicenzaConsultable(t *testing.T) {
	txt := (&App{}).LicenzaTexto()
	for _, c := range []string{"MIT License", "Martín Pardiñas", "Microsoft Corporation", "markitdown", "PyInstaller"} {
		if !strings.Contains(txt, c) {
			t.Errorf("a licenza non menciona %q", c)
		}
	}
}
