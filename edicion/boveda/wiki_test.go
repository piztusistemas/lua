package boveda

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"edicion/ficha"
	"edicion/ia"
)

// Unha IA falsa (formato OpenAI) que fai o ingest chamando ás ferramentas.
func TestConstruirWiki(t *testing.T) {
	b, _ := Crear(t.TempDir(), "Química")
	_ = os.WriteFile(filepath.Join(b.Raiz, "raw", "quimica-ex2.md"), []byte("---\nfonte: \"[[quimica-ex2.pdf]]\"\n---\n\nPREGUNTA 4. Ácido débil HA, Ka = 3·10⁻⁶. Calcule o pH.\n"), 0o644)
	f := &ficha.Ficha{Luayang: 1, ID: "quimica-ex2-4", Tipo: "exercicio", Titulo: "Ácido débil", Conceptos: []string{"[[Equilibrio ácido-base]]"}, Corpo: "## Enunciado\n\nx\n"}
	if _, err := b.Gardar(f, timeZero()); err != nil {
		t.Fatal(err)
	}
	chamada := func(id, nome string, args any) string {
		a, _ := json.Marshal(args)
		r, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": "", "tool_calls": []any{map[string]any{"id": id, "type": "function", "function": map[string]any{"name": nome, "arguments": string(a)}}}}, "finish_reason": "tool_calls"}}})
		return string(r)
	}
	texto := func(s string) string {
		r, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": s}, "finish_reason": "stop"}}})
		return string(r)
	}
	pasos := []string{
		chamada("c1", "boveda_info", map[string]any{}),
		chamada("c2", "boveda_fichas", map[string]any{}),
		chamada("c3", "boveda_escribir", map[string]any{"ruta": "wiki/fontes/quimica-ex2.md", "contido": "# Química ex2\n\nFonte: [[quimica-ex2]]. Conceptos: [[Equilibrio ácido-base]].\n"}),
		chamada("c4", "boveda_escribir", map[string]any{"ruta": "wiki/conceptos/Equilibrio ácido-base.md", "contido": "# Equilibrio ácido-base\n\nVer [[quimica-ex2-4]].\n"}),
		chamada("c5", "boveda_escribir", map[string]any{"ruta": "raw/trampa.md", "contido": "non"}),
		texto("Feito: 2 páxinas."),
		texto("Lint feito."),
	}
	var n int32
	var pecharErro string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		corpo, _ := io.ReadAll(r.Body)
		i := atomic.AddInt32(&n, 1) - 1
		if i == 5 && !strings.Contains(string(corpo), "só se pode escribir dentro de wiki/") {
			pecharErro = "non chegou o erro de escribir en raw/"
		}
		if int(i) >= len(pasos) {
			_, _ = w.Write([]byte(texto("ok")))
			return
		}
		_, _ = w.Write([]byte(pasos[i]))
	}))
	defer srv.Close()
	cfg := ia.Config{Provedor: ia.ProvedorOpenAI, BaseURL: srv.URL, APIKey: "k", Modelo: "m"}
	var avisos []string
	res, err := b.Construir(context.Background(), cfg, "Química", func(s string) { avisos = append(avisos, s) }, nil)
	if err != nil || !strings.Contains(res, "Feito") {
		t.Fatalf("%q %v\n%v", res, err, avisos)
	}
	if pecharErro != "" {
		t.Error(pecharErro)
	}
	if _, err := os.Stat(filepath.Join(b.Raiz, "raw", "trampa.md")); err == nil {
		t.Error("a IA escribiu en raw/")
	}
	if p := b.Pendentes(); len(p) != 0 {
		t.Errorf("seguen pendentes: %v", p)
	}
	cs, err := b.Conceptos()
	if err != nil {
		t.Fatal(err)
	}
	var atopado bool
	for _, c := range cs {
		if c.Nome == "Equilibrio ácido-base" {
			atopado = c.Paxina == "wiki/conceptos/Equilibrio ácido-base.md" && len(c.Fichas) == 1
		}
	}
	if !atopado {
		t.Errorf("o concepto non une wiki e ficha: %+v", cs)
	}
	nota, err := b.LerNota("Equilibrio ácido-base")
	if err != nil || !strings.Contains(nota, "quimica-ex2-4") {
		t.Errorf("%q %v", nota, err)
	}
	if l := b.Lint(); len(l.Rotos) != 0 {
		t.Errorf("enlaces rotos: %v", l.Rotos)
	}
	if r := b.Buscar("ácido pH", 5); len(r) == 0 || r[0].Ruta != "raw/quimica-ex2.md" {
		t.Errorf("busca: %+v", r)
	}
	_ = fmt.Sprint()
}
