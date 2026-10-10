package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
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

// Spec buxo-idiomas, aceptación 6: o panel vai no idioma de Lua e buxo.log
// sempre en galego, tamén cando un argumento é un erro traducible.
func TestRexistroIdiomas(t *testing.T) {
	t.Cleanup(idioma.Restablecer)
	idioma.Restablecer()
	idioma.Rexistrar("pt", map[string]string{
		"rex.proba.convertido": "Convertido em raw/%s",
		"rex.proba.saltase":    "%s: %v (ignorado)",
		"erro.proba.falta":     "não se encontrou %s",
	})
	idioma.SetIdioma("pt")

	ficheiro := filepath.Join(t.TempDir(), "buxo.log")
	var panel []string
	r := NovoRexistro(ficheiro, func(l Liña) { panel = append(panel, l.Texto) })
	r.Info("rex.proba.convertido", "Convertido raw/%s", "a.md")
	r.Erro("rex.proba.saltase", "%s: %v (sáltase)", "b.pdf", idioma.Errorf("erro.proba.falta", "non se atopou %s", "pdftoppm"))

	quere := []string{"Convertido em raw/a.md", "b.pdf: não se encontrou pdftoppm (ignorado)"}
	if !slices.Equal(panel, quere) {
		t.Errorf("panel = %q, quería %q", panel, quere)
	}
	datos, err := os.ReadFile(ficheiro)
	if err != nil {
		t.Fatal(err)
	}
	log := string(datos)
	for _, galego := range []string{"Convertido raw/a.md", "b.pdf: non se atopou pdftoppm (sáltase)"} {
		if !strings.Contains(log, galego) {
			t.Errorf("buxo.log non ten %q en galego:\n%s", galego, log)
		}
	}
}

// Spec buxo-idiomas, aceptación 2: un erro de conversión, con Lua en inglés,
// sae en inglés e conserva o nome do ficheiro.
func TestErroConversionEnIngles(t *testing.T) {
	t.Cleanup(idioma.Restablecer)
	idioma.Restablecer()
	t.Setenv("BUXO_IDIOMA", "en")
	configurarIdioma()
	orixe := filepath.Join(t.TempDir(), "notas.xyz")
	if err := os.WriteFile(orixe, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := CopiarFicheiro(orixe, t.TempDir())
	if err == nil {
		t.Fatal("un .xyz debería dar erro")
	}
	if got := err.Error(); !strings.Contains(got, "notas.xyz") || !strings.Contains(got, "does not know how to handle") {
		t.Errorf("erro en inglés co nome, obtido %q", got)
	}
	if m := motivoVision("ola", 3, 0); m == nil || !strings.Contains(m.Error(), "little extractable text") {
		t.Errorf("motivo en inglés, obtido %v", m)
	}
}
