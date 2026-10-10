package main

import (
	"context"
	"slices"
	"testing"

	"edicion/idioma"
)

// TestIdiomaGoSegueOIdiomaEfectivo: as mensaxes que xera o Go usan o mesmo
// idioma ca interface (Settings.Idioma ou, se está baleiro, o de Piztu), ao
// arrincar e cada vez que se gardan as Opcións (tamén desde o asistente).
func TestIdiomaGoSegueOIdiomaEfectivo(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	idioma.Restablecer()
	t.Cleanup(idioma.Restablecer)

	if err := saveSettingsToDisk(Settings{Idioma: "pt"}); err != nil {
		t.Fatal(err)
	}
	a := NewApp()
	a.startup(context.Background())
	if got := idioma.Actual(); got != "pt" {
		t.Errorf("ao arrincar: idioma do Go = %q, quería pt", got)
	}
	// Os dicionarios de Lua quedan rexistrados: unha clave coñecida tradúcese.
	if got := idioma.T("about.version", "Versión"); got != "Versão" {
		t.Errorf("dicionario pt sen rexistrar: %q", got)
	}

	if err := a.SaveSettings(Settings{Idioma: "en"}); err != nil {
		t.Fatal(err)
	}
	if got := idioma.Actual(); got != "en" {
		t.Errorf("tras gardar Opcións: %q, quería en", got)
	}

	a.idiomaPiztu = "es"
	if err := a.SaveSettings(Settings{Idioma: ""}); err != nil {
		t.Fatal(err)
	}
	if got := idioma.Actual(); got != "es" {
		t.Errorf("sen idioma propio: %q, quería o de Piztu (es)", got)
	}
}

// Plan buxo-idiomas T009: Lua pásalle a Buxo o idioma efectivo en
// BUXO_IDIOMA (variable de entorno: un Buxo vello ignóraa).
func TestBuxoIdioma(t *testing.T) {
	a := NewApp()
	a.settings = Settings{Idioma: "pt"}
	if env := a.entornoBuxo("http://127.0.0.1:1", "tok"); !slices.Contains(env, "BUXO_IDIOMA=pt") {
		t.Errorf("con Opcións en pt, falta BUXO_IDIOMA=pt")
	}
	a.settings = Settings{}
	a.idiomaPiztu = "es"
	env := a.entornoBuxo("http://127.0.0.1:1", "tok")
	if !slices.Contains(env, "BUXO_IDIOMA=es") {
		t.Errorf("sen idioma propio, falta o de Piztu (BUXO_IDIOMA=es)")
	}
	if !slices.Contains(env, "PIZTU_API_TOKEN=tok") {
		t.Errorf("o entorno perdeu o token da API")
	}
}
