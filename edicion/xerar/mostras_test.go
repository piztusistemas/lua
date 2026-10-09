package xerar

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"edicion/boveda"
	"edicion/ia"
	"edicion/lectura"
)

// Os 5 PDF das mostras, sen IA: ler, separar exercicios, gardar e xerar.
func TestMostrasDePuntaAPunta(t *testing.T) {
	precisa(t, "pandoc", "xelatex", "pdftoppm")
	pdfs, _ := filepath.Glob("../../luayang/mostras/*.pdf")
	if len(pdfs) == 0 {
		t.Skip("sen mostras")
	}
	b, err := boveda.Crear(t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	for _, pdf := range pdfs {
		nome := filepath.Base(pdf)
		t.Run(nome[24:], func(t *testing.T) {
			pax, err := lectura.Clasificar(pdf)
			if err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			lidas, err := lectura.Ler(context.Background(), pdf, pax, lectura.Opcions{IA: ia.Config{}, Dir: dir, Prefixo: "m"})
			if err != nil {
				t.Fatal(err)
			}
			e, _ := lectura.Estruturar(ia.Config{}, lidas, nome)
			r, err := b.GardarLectura(pdf, lidas, e, dir)
			if err != nil {
				t.Fatal(err)
			}
			exame, err := b.Resolver(r.Exame)
			if err != nil {
				t.Fatal(err)
			}
			tmp := t.TempDir()
			res, err := Corpo(exame, b.Resolver, Opcions{BaseDir: b.Raiz}, tmp)
			if err != nil {
				t.Fatal(err)
			}
			compilar(t, tmp, res.Corpo)
			t.Logf("%d exercicios, %d caracteres de LaTeX", len(r.Exercicios), len(res.Corpo))
			if len(r.Exercicios) < 2 || strings.TrimSpace(res.Corpo) == "" {
				t.Errorf("non se separaron os exercicios: %+v", r)
			}
		})
	}
	_ = os.Remove
}
