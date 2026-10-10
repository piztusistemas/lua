package main

import (
	"testing"

	"edicion/idioma"
)

// Spec buxo-idiomas, aceptacións 1 e 3: Buxo toma o idioma que lle pasa Lua
// en BUXO_IDIOMA; sen el, ou cun descoñecido, queda en galego.
func TestIdiomaArrinque(t *testing.T) {
	t.Cleanup(idioma.Restablecer)
	casos := []struct{ entorno, quere string }{
		{"es", "es"}, {"", "gl"}, {"xx", "gl"}, {"pt", "pt"},
	}
	for _, c := range casos {
		idioma.Restablecer()
		t.Setenv("BUXO_IDIOMA", c.entorno)
		configurarIdioma()
		if got := idioma.Actual(); got != c.quere {
			t.Errorf("BUXO_IDIOMA=%q → %q, quería %q", c.entorno, got, c.quere)
		}
	}
	// Os dicionarios de Buxo quedan rexistrados.
	idioma.Restablecer()
	t.Setenv("BUXO_IDIOMA", "es")
	configurarIdioma()
	if got := idioma.T("ui.construir", "Construír a bóveda"); got != "Construir la bóveda" {
		t.Errorf("dicionario es sen rexistrar: %q", got)
	}
}
