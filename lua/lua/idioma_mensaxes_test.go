package main

import (
	"strings"
	"testing"

	"edicion/idioma"
)

// conIdioma deixa o Go de Lua no idioma pedido, cos dicionarios reais, e
// volve ao galego ao rematar o test.
func conIdioma(t *testing.T, codigo string) {
	t.Helper()
	idioma.Restablecer()
	rexistrarIdiomasGo()
	idioma.SetIdioma(codigo)
	t.Cleanup(idioma.Restablecer)
}

// Spec mensaxes-go-idiomas, aceptación 3: falta un paquete de LaTeX, Lua en
// portugués → a pista está en portugués e leva o nome exacto do paquete.
func TestPistaPaqueteLatexEnPortugues(t *testing.T) {
	conIdioma(t, "pt")
	log := "! LaTeX Error: File `circuitikz.sty' not found."
	got := pistaLatexAccionable(log)
	if !strings.Contains(got, "circuitikz") || !strings.Contains(got, "pacote") {
		t.Errorf("pista en portugués co nome do paquete, obtido %q", got)
	}
}

// Spec mensaxes-go-idiomas, situación 1: o aviso de cambio automático de
// motor sae no idioma activo.
func TestAvisoCambioDeMotorEnIngles(t *testing.T) {
	if !commandExists("xelatex") || !commandExists("pdflatex") {
		t.Skip("precisa xelatex e pdflatex")
	}
	conIdioma(t, "en")
	// Un documento que se nega a compilar con xelatex e compila con pdflatex.
	tex := `\documentclass{article}\ifdefined\XeTeXversion\errmessage{non con xelatex}\fi\begin{document}Ola\end{document}`
	res, err := compilarComMotores("xelatex", nil, t.TempDir(), nil, func(string) (string, []string, error) { return tex, nil, nil })
	if err != nil {
		t.Fatalf("compilar: %v", err)
	}
	if res.Motor != "pdflatex" {
		t.Fatalf("motor = %s, quería pdflatex", res.Motor)
	}
	aviso := strings.Join(res.Avisos, "\n")
	if !strings.Contains(aviso, "xelatex could not compile this report") {
		t.Errorf("aviso en inglés, obtido %q", aviso)
	}
}

// Spec mensaxes-go-idiomas, aceptación 2: abrir un ficheiro que non é un
// formulario válido, con Lua en castelán → o erro está en castelán.
func TestFicheiroNonValidoEnCastelan(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	conIdioma(t, "es")
	_, err := importarBundleFormulario([]byte(`{"ola": 1}`))
	if err == nil {
		t.Fatal("un JSON que non é un formulario debería dar erro")
	}
	if !strings.Contains(err.Error(), "no es un formulario de Lua") {
		t.Errorf("erro en castelán, obtido %q", err.Error())
	}
	_, err = importarBundleFormulario([]byte(`non é JSON`))
	if err == nil || !strings.Contains(err.Error(), "no es un JSON válido") {
		t.Errorf("JSON roto: erro en castelán, obtido %v", err)
	}
}

// Spec mensaxes-go-idiomas, aceptación 7: sen clave de API e Lua en inglés,
// o erro do asistente está en inglés.
func TestIASenClaveEnIngles(t *testing.T) {
	conIdioma(t, "en")
	_, _, _, err := resolverIA(Settings{})
	if err == nil {
		t.Fatal("sen clave de API debería dar erro")
	}
	if !strings.Contains(err.Error(), "AI API key is missing") {
		t.Errorf("erro en inglés, obtido %q", err.Error())
	}
}
