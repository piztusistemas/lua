package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeometriaDaMaqueta(t *testing.T) {
	if g := geometriaDaMaqueta("\\pagestyle{fancy}\n"); g != "" {
		t.Errorf("sen medidas non hai nada que arranxar: %q", g)
	}
	g := geometriaDaMaqueta("\\setlength{\\headheight}{ 52pt }\n\\setlength{\\footskip}{14mm}\n\\setlength{\\headheight}{40pt}\n\\setlength{\\headsep}{\\baselineskip}")
	// A última medida de cada tipo gaña; as que non son literais ignóranse.
	if g != "\\ifdefined\\geometry\\geometry{headheight=40pt,footskip=14mm,includehead}\\fi\n" {
		t.Errorf("geometría = %q", g)
	}
}

const maquetaCompletaPe = `\documentclass[a4paper,11pt]{article}
\usepackage[left=20mm,right=20mm,top=20mm,bottom=22mm]{geometry}
\usepackage{fancyhdr}
\begin{document}
\pagestyle{fancy}
\fancyhf{}
\setlength{\headheight}{52pt}
\setlength{\footskip}{14mm}
\fancyhead[C]{CABECEIRA-PROBA\\segunda liña\\terceira liña}
\fancyfoot[L]{PE-PROBA}
\fancyfoot[R]{\thepage}
{{PORTADA}}
{{CORPO}}
\end{document}
`

// Unha maqueta que cambia \headheight/\footskip despois de \begin{document} non
// debe sacar o pé de páxina (co número) fóra da folla nin deixar o texto invadir a
// marxe de abaixo: Lua leva esas medidas ao geometry.
func TestMaquetaCoPeAposBeginDocumentSenTraerOPeFora(t *testing.T) {
	if !commandExists("pdflatex") || !commandExists("pdftotext") {
		t.Skip("sen pdflatex/pdftotext")
	}
	pl := &PlantillaLatex{Nome: "Pe", Latex: maquetaCompletaPe}
	tex, _, err := latexInformeCrono(planMostraCrono(), "pdflatex", nil, t.TempDir(), agoraFragmentos, pl, "")
	if err != nil {
		t.Fatal(err)
	}
	if i, j := strings.Index(tex, `\geometry{headheight=52pt,footskip=14mm,includehead}`), strings.Index(tex, `\begin{document}`); i < 0 || i > j {
		t.Fatalf("falta o \\geometry antes de \\begin{document}")
	}
	dir := t.TempDir()
	res, err := compilarInformeCrono(planMostraCrono(), "pdflatex", nil, dir, pl, "")
	if err != nil {
		t.Fatalf("%v\n%s", err, res.Log)
	}
	pdf := filepath.Join(dir, "x.pdf")
	_ = os.WriteFile(pdf, res.PDF, 0o644)
	n := numeroPaxinas(t, res.PDF)
	saida, err := exec.Command("pdftotext", "-layout", pdf, "-").Output()
	if err != nil {
		t.Fatal(err)
	}
	// O pé sae en todas as páxinas menos na portada.
	if c := strings.Count(string(saida), "PE-PROBA"); c < n-1 {
		t.Errorf("o pé sae en %d páxinas de %d", c, n)
	}
}
