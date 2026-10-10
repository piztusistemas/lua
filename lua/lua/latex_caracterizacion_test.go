package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Caracterización das funcións auxiliares de LaTeX (spec sen-duplicados-ia-latex):
// o texto que xeran para un conxunto fixo de entradas non pode cambiar ao
// movelas a edicion. Rexenerar só antes de refactorizar: CARACTERIZAR=1.

const ficheiroCaracterizacionLatex = "testdata/latex_caracterizacion.golden.json"

func casosCaracterizacionLatex() map[string]any {
	out := map[string]any{}
	for _, e := range []string{"", "pdflatex", "xelatex", "lualatex", "raro"} {
		out["engineUnicodeNativo/"+e] = engineUnicodeNativo(e)
		out["motoresParaProbar/"+e] = motoresParaProbar(e)
		for _, f := range []string{"", "serif", "sans", "palatino", "DejaVu Sans", "Fonte {mala}\\"} {
			for _, tam := range []float64{0, 11, 12.5} {
				t := &TipografiaOpts{Fonte: f, TamanoPt: tam, Interlinado: 1.15}
				out["tipografiaPreambulo/"+e+"/"+f+"/"+trimFloat(tam)] = tipografiaPreambulo(t, e)
			}
		}
		out["tipografiaPreambulo/nil/"+e] = tipografiaPreambulo(nil, e)
	}
	for _, tam := range []float64{0, 10, 11.5} {
		for _, inter := range []float64{0, 1, 1.5} {
			out["tipografiaCorpo/"+trimFloat(tam)+"/"+trimFloat(inter)] = tipografiaCorpo(&TipografiaOpts{TamanoPt: tam, Interlinado: inter})
		}
	}
	out["tipografiaCorpo/nil"] = tipografiaCorpo(nil)
	for _, f := range []string{"", "serif", "Arial"} {
		out["esFonteDoSistema/"+f] = esFonteDoSistema(f)
	}
	for _, b := range []string{"texgyreheros", "texgyretermes"} {
		out["setmainfontTeXGyre/"+b] = setmainfontTeXGyre(b)
	}
	for _, f := range []float64{0, 1, 1.15, 2.5, 10.125} {
		out["trimFloat/"+strings.ReplaceAll(trimFloat(f), ".", "_")] = trimFloat(f)
	}
	out["babelLine"] = babelLine()
	logs := map[string]string{
		"fonte":   "! Package fontspec Error: The font \"Comic Sans\" cannot be found.",
		"paquete": "! LaTeX Error: File `circuitikz.sty' not found.",
		"ambiente": "! LaTeX Error: Environment matrixx undefined.\n" +
			"l.12 \\begin{matrixx}",
		"sintaxe":    "! Undefined control sequence.\nl.5 \\foo\n",
		"baleiro":    "",
		"continuado": "! LaTeX Error: File `pgfplots\n.sty' not found.",
	}
	for k, l := range logs {
		out["pistaLatexAccionable/"+k] = pistaLatexAccionable(l)
		out["extraerErroPDFLatex/"+k] = extraerErroPDFLatex(l)
	}
	for _, l := range []string{"", "abc", "liña con espazos  "} {
		out["continuacionPaquete/"+l] = continuacionPaquete(l)
	}
	return out
}

func TestCaracterizacionLatex(t *testing.T) {
	actual, err := json.MarshalIndent(casosCaracterizacionLatex(), "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if os.Getenv("CARACTERIZAR") == "1" {
		_ = os.MkdirAll(filepath.Dir(ficheiroCaracterizacionLatex), 0o755)
		if err := os.WriteFile(ficheiroCaracterizacionLatex, append(actual, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	esperado, err := os.ReadFile(ficheiroCaracterizacionLatex)
	if err != nil {
		t.Fatalf("falta a referencia (%v): xérase con CARACTERIZAR=1 antes de refactorizar", err)
	}
	if strings.TrimSpace(string(esperado)) != strings.TrimSpace(string(actual)) {
		ruta := filepath.Join(t.TempDir(), "actual.json")
		_ = os.WriteFile(ruta, actual, 0o644)
		t.Errorf("as funcións de LaTeX cambiaron o que xeran.\n  referencia: %s\n  actual:     %s", ficheiroCaracterizacionLatex, ruta)
	}
}
