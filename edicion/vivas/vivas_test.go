package vivas

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"

	"edicion/ficha"
	"edicion/ia"
	"edicion/xerar"
)

// A IA falsa primeiro propón unha ficha que divide por cero e, ao recibir o
// erro, arránxaa.
func TestEditarArranxaTrasProbas(t *testing.T) {
	if _, err := exec.LookPath("maxima"); err != nil {
		t.Skip("sen maxima")
	}
	mala := "---\nluayang: 1\ntipo: exercicio\ntitulo: Cociente\ncampos:\n  - nome: b\n    orixe: aleatorio\n    xerador: rango\n    de: 0\n    ata: 3\n  - nome: r\n    orixe: calculado\n    motor: maxima\n    expr: \"12/b\"\n    rol: resultado\n---\n\n## Enunciado\n\nCalcula 12/{b}.\n"
	boa := strings.Replace(mala, "de: 0", "de: 1", 1)
	var n int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		corpo, _ := io.ReadAll(r.Body)
		txt := "```markdown\n" + mala + "```"
		if atomic.AddInt32(&n, 1) > 1 {
			if !strings.Contains(string(corpo), "As probas fallan") {
				t.Errorf("non chegou o resumo das probas: %s", corpo)
			}
			txt = boa
		}
		out, _ := json.Marshal(map[string]any{"content": []any{map[string]any{"type": "text", "text": txt}}, "stop_reason": "end_turn"})
		_, _ = w.Write(out)
	}))
	defer srv.Close()
	c := ia.Config{Provedor: ia.ProvedorAnthropic, BaseURL: srv.URL, APIKey: "k", Modelo: "m"}
	actual := &ficha.Ficha{Luayang: 1, ID: "cociente", Tipo: ficha.TipoExercicio, Titulo: "Cociente", Corpo: "## Enunciado\n\nCalcula 12/4.\n"}
	p, err := Editar(c, actual, PedidoVivificar, func(f *ficha.Ficha) (xerar.InformeProbas, error) {
		return xerar.Probar(f, nil, xerar.Opcions{}, 20)
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !p.Valida || p.Rondas != 2 || !strings.Contains(p.Texto, "de: \"1\"") && !strings.Contains(p.Texto, "de: 1") {
		t.Fatalf("%+v", p)
	}
	if !strings.Contains(p.Texto, "id: cociente") {
		t.Errorf("perdeuse o id:\n%s", p.Texto)
	}
}
