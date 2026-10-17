package main

import (
	"strings"
	"testing"

	"edicion/plantilla"
)

func TestCabeceiraPeFilete(t *testing.T) {
	r := desenoResolto{d: plantilla.Deseno{CabeceiraFilete: true, Cabeceira: plantilla.Zonas{Esquerda: "@logo"}}}
	l := r.latexCabeceiraPe()
	if !strings.Contains(l, `\headrulewidth}{0.4pt}`) {
		t.Fatalf("falta a liña da cabeceira:\n%s", l)
	}
	r = desenoResolto{d: plantilla.Deseno{Cabeceira: plantilla.Zonas{Esquerda: "x"}}}
	if l := r.latexCabeceiraPe(); !strings.Contains(l, `\headrulewidth}{0pt}`) {
		t.Fatalf("sobra a liña:\n%s", l)
	}
	if l := (desenoResolto{d: plantilla.Deseno{}}).latexCabeceiraPe(); l != "" {
		t.Fatalf("debería ir baleiro: %q", l)
	}
}
