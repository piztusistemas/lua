package main

import (
	"strings"
	"testing"
)

func TestRexistroInterno(t *testing.T) {
	log := "This is pdfTeX\n(./doc.tex\n! Undefined control sequence.\nl.4 Masa \\maza\n                {m}\n?\n"
	tex := "\\documentclass{article}\n\\begin{document}\nOla\nMasa \\maza{m}\nFin\n\\end{document}"
	id := rexistrar(EntradaRexistro{Tipo: "compilacion", Documento: "formulario «Exame»", Erro: "pdflatex fallou", Log: log, Tex: tex})
	priv := rexistrar(EntradaRexistro{Tipo: "compilacion", Documento: "formulario «Notas»", Erro: "x", Log: "segredo", Tex: "segredo", Privado: true})

	r, err := lerRexistro(map[string]any{"id": float64(id)})
	if err != nil {
		t.Fatal(err)
	}
	m := r.(map[string]any)
	if !strings.Contains(m["erro_latex"].(string), "Undefined control sequence") || m["liña_tex"] != 4 || !strings.Contains(m["tex_arredor"].(string), "    4| Masa \\maza{m}") {
		t.Fatalf("%+v", m)
	}
	r, _ = lerRexistro(map[string]any{"id": float64(id), "parte": "tex", "buscar": "maza"})
	if a := r.(map[string]any)["achados"].([]string); len(a) != 1 {
		t.Fatalf("%v", a)
	}
	// Un formulario en produción: nin log nin .tex para a IA.
	r, _ = lerRexistro(map[string]any{"id": float64(priv), "parte": "log"})
	if s := strings.Join(strings.Fields(strings.ReplaceAll(strings.TrimSpace(func() string {
		b := ""
		for k, v := range r.(map[string]any) {
			b += k + "=" + toS(v) + " "
		}
		return b
	}()), "\n", " ")), " "); strings.Contains(s, "segredo") {
		t.Fatalf("filtrouse o privado: %s", s)
	}
	// As ferramentas están nos ámbitos dos asistentes.
	for _, amb := range []string{ambitoFormularioDatos, ambitoPlantilla, ""} {
		nomes := map[string]bool{}
		for _, f := range (&App{}).IAFerramentasDoAmbito(amb) {
			nomes[f.Nome] = true
		}
		if !nomes["rexistro_ler"] || !nomes["rexistro_listar"] {
			t.Errorf("ámbito %q sen as ferramentas do rexistro", amb)
		}
	}
}

func toS(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// A API local deixa que un módulo (Buxo) mande imaxes nos turnos.
func TestImaxesDaAPI(t *testing.T) {
	msgs := []IAMensaxe{{Rol: "user", Texto: "le", Imaxes: []string{"data:image/png;base64,iVBORw0KGgo="}}}
	if err := imaxesDaAPI(msgs); err != nil || len(msgs[0].ImaxesIA) != 1 || msgs[0].ImaxesIA[0].Mime != "image/png" || msgs[0].Imaxes != nil {
		t.Fatalf("%v %+v", err, msgs[0])
	}
	if err := imaxesDaAPI([]IAMensaxe{{Imaxes: []string{"/etc/passwd"}}}); err == nil {
		t.Error("unha ruta non é un data URI")
	}
	moitas := make([]string, maxImaxesAPI+1)
	for i := range moitas {
		moitas[i] = "data:image/png;base64,AA=="
	}
	if err := imaxesDaAPI([]IAMensaxe{{Imaxes: moitas}}); err == nil {
		t.Error("tope de imaxes")
	}
}
