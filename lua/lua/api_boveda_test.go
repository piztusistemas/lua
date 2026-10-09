package main

import (
	"context"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"edicion/api"
)

func appConBoveda(t *testing.T) (*App, *boveda) {
	t.Helper()
	b := novaBovedaProba(t)
	a := &App{}
	a.settings.IABovedas = map[string]string{"cronograma": b.raiz}
	t.Cleanup(func() {
		if a.api != nil {
			a.api.Parar()
		}
	})
	return a, b
}

func codigoAPI(err error) int {
	var e *api.Erro
	if errors.As(err, &e) {
		return e.Codigo
	}
	return 0
}

func TestAPIBovedaLerEBuscar(t *testing.T) {
	a, _ := appConBoveda(t)
	_, c, err := a.servidorAPI()
	if err != nil {
		t.Fatal(err)
	}
	var lista []BovedaEstado
	if err := c.Obter(context.Background(), "/bovedas", nil, &lista); err != nil || len(lista) != 1 || lista[0].Documento != "cronograma" {
		t.Fatalf("bóvedas: %v %v", lista, err)
	}
	var r map[string]any
	if err := c.Obter(context.Background(), "/bovedas/cronograma/ler", url.Values{"ruta": {"wiki/clorofila.md"}}, &r); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r["texto"].(string), "Pigmento") {
		t.Errorf("ler: %v", r)
	}
	var busca map[string]any
	if err := c.Obter(context.Background(), "/bovedas/cronograma/buscar", url.Values{"consulta": {"pigmento"}, "max": {"5"}}, &busca); err != nil || busca["total"] != float64(1) {
		t.Errorf("buscar: %v %v", busca, err)
	}
	if err := c.Obter(context.Background(), "/bovedas/actas/info", nil, nil); codigoAPI(err) != 404 {
		t.Errorf("bóveda sen conectar debería dar 404: %v", err)
	}
	if err := c.Obter(context.Background(), "/bovedas/cronograma/ler", url.Values{"ruta": {"../fora.md"}}, nil); err == nil {
		t.Error("unha ruta fóra da bóveda debería fallar")
	}
}

func TestAPIBovedaEscribirETokenSoLectura(t *testing.T) {
	a, b := appConBoveda(t)
	s, c, err := a.servidorAPI()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := c.Facer(ctx, "PUT", "/bovedas/cronograma/escribir", nil, map[string]any{"ruta": "wiki/nova.md", "contido": "Ola"}, nil); err != nil {
		t.Fatal(err)
	}
	if d, _ := os.ReadFile(filepath.Join(b.raiz, "wiki", "nova.md")); !strings.Contains(string(d), "Ola") {
		t.Error("a páxina non se escribiu")
	}
	// raw/ segue sendo inmutable tamén pola API.
	if err := c.Facer(ctx, "PUT", "/bovedas/cronograma/escribir", nil, map[string]any{"ruta": "raw/x.md", "contido": "x"}, nil); err == nil {
		t.Error("raw/ non se pode escribir")
	}
	// Un módulo con só lectura non escribe.
	url, tk, err := a.tokenModuloAPI("modulo-lector", permisoBovedaLer)
	if err != nil {
		t.Fatal(err)
	}
	lector := &api.Cliente{URL: url, Token: tk}
	if err := lector.Facer(ctx, "PUT", "/bovedas/cronograma/escribir", nil, map[string]any{"ruta": "wiki/y.md", "contido": "y"}, nil); codigoAPI(err) != 403 {
		t.Errorf("sen permiso de escritura debería dar 403: %v", err)
	}
	if err := lector.Obter(ctx, "/bovedas/cronograma/info", nil, nil); err != nil {
		t.Errorf("con permiso de lectura debería valer: %v", err)
	}
	s.Revogar("modulo-lector")
	if err := lector.Obter(ctx, "/bovedas/cronograma/info", nil, nil); codigoAPI(err) != 401 {
		t.Errorf("token revogado debería dar 401: %v", err)
	}
}

// As ferramentas boveda_* do asistente pasan pola API (mesmos permisos e
// mesmas comprobacións), usando a bóveda do turno.
func TestFerramentasBovedaPasanPolaAPI(t *testing.T) {
	a, b := appConBoveda(t)
	if _, err := execBoveda("boveda_ler")(a, map[string]any{"ruta": "wiki/clorofila.md"}); err == nil {
		t.Error("sen bóveda do turno debería fallar")
	}
	a.prepararBovedaTurno("", "cronograma")
	if a.bovedaTurnoID != "cronograma" {
		t.Fatalf("id do turno: %q", a.bovedaTurnoID)
	}
	res, err := execBoveda("boveda_ler")(a, map[string]any{"ruta": "[[clorofila]]", "inicio": 0})
	if err != nil || !strings.Contains(res.(map[string]any)["texto"].(string), "Pigmento") {
		t.Fatalf("boveda_ler: %v %v", res, err)
	}
	if _, err := execBoveda("boveda_escribir")(a, map[string]any{"ruta": "wiki/desde-ia.md", "contido": "x", "modo": "sobrescribir"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(b.raiz, "wiki", "desde-ia.md")); err != nil {
		t.Error("a ferramenta non escribiu pola API")
	}
	if _, err := execBoveda("boveda_escribir")(a, map[string]any{"ruta": "CLAUDE.md", "contido": "x"}); err == nil {
		t.Error("CLAUDE.md non se pode modificar")
	}
}
