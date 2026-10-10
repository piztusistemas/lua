package main

import (
	"edicion/latex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestTipografiaCorpo(t *testing.T) {
	casos := []struct {
		nome string
		in   *TipografiaOpts
		want string
	}{
		{"nil", nil, ""},
		{"todo por defecto", &TipografiaOpts{}, ""},
		{"só tamaño", &TipografiaOpts{TamanoPt: 13}, "\\fontsize{13pt}{15.6pt}\\selectfont\n"},
		{"tamaño + interliñado", &TipografiaOpts{TamanoPt: 12, Interlinado: 1.5}, "\\fontsize{12pt}{21.6pt}\\selectfont\n"},
		{"só interliñado", &TipografiaOpts{Interlinado: 2}, "\\linespread{2}\\selectfont\n"},
	}
	for _, c := range casos {
		if got := tipografiaCorpo(c.in); got != c.want {
			t.Errorf("%s: got %q, want %q", c.nome, got, c.want)
		}
	}
}

func TestTipografiaPreambulo(t *testing.T) {
	if got := tipografiaPreambulo(nil, "pdflatex"); got != "" {
		t.Errorf("nil: got %q", got)
	}
	if got := tipografiaPreambulo(&TipografiaOpts{Fonte: "inventada"}, "pdflatex"); got != "" {
		t.Errorf("preset descoñecido: got %q", got)
	}
	if got := tipografiaPreambulo(&TipografiaOpts{Fonte: "serif"}, "pdflatex"); !strings.Contains(got, "mathptmx") {
		t.Errorf("pdflatex serif debería cargar mathptmx: %q", got)
	}
	if got := tipografiaPreambulo(&TipografiaOpts{Fonte: "serif"}, "xelatex"); !strings.Contains(got, "\\setmainfont{texgyretermes}") {
		t.Errorf("xelatex serif debería usar \\setmainfont: %q", got)
	}
	if got := tipografiaPreambulo(&TipografiaOpts{Fonte: "sans"}, "pdflatex"); !strings.Contains(got, "helvet") {
		t.Errorf("pdflatex sans debería cargar helvet: %q", got)
	}

	// Fonte do sistema (nome de familia calquera): \setmainfont só con
	// motor Unicode; nada con pdflatex.
	if got := tipografiaPreambulo(&TipografiaOpts{Fonte: "DejaVu Serif"}, "xelatex"); got != "\\setmainfont{DejaVu Serif}\n" {
		t.Errorf("fonte do sistema en xelatex: %q", got)
	}
	if got := tipografiaPreambulo(&TipografiaOpts{Fonte: "DejaVu Serif"}, "pdflatex"); got != "" {
		t.Errorf("fonte do sistema en pdflatex debería dar '': %q", got)
	}
	// Limpeza contra inxección de LaTeX no nome.
	if got := tipografiaPreambulo(&TipografiaOpts{Fonte: "X}\\input{/etc/passwd}{"}, "xelatex"); strings.ContainsAny(got, "\\}{") && !strings.HasPrefix(got, "\\setmainfont{") {
		t.Errorf("nome de fonte sen sanear: %q", got)
	}
}

func TestEsFonteDoSistema(t *testing.T) {
	casos := map[string]bool{"": false, "serif": false, "palatino": false, "DejaVu Sans": true, "Cambria": true}
	for f, want := range casos {
		if got := esFonteDoSistema(f); got != want {
			t.Errorf("esFonteDoSistema(%q) = %v, want %v", f, got, want)
		}
	}
}

// TestPresetsFonteCompilanConXelatex: as fontes TeX Gyre viven en TeX Live,
// non no sistema, así que en macOS fontspec non as atopa polo nome. Cada
// preset ten que compilar de verdade con xelatex.
func TestPresetsFonteCompilanConXelatex(t *testing.T) {
	if !commandExists("xelatex") {
		t.Skip("xelatex non está instalado")
	}
	for _, id := range latex.PresetsFonte() {
		dir := t.TempDir()
		tex := "\\documentclass{article}\n\\usepackage{fontspec}\n" +
			tipografiaPreambulo(&TipografiaOpts{Fonte: id}, "xelatex") +
			"\\begin{document}Ola \\textbf{negra} \\textit{cursiva}\\end{document}\n"
		if err := os.WriteFile(filepath.Join(dir, "f.tex"), []byte(tex), 0o644); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command("xelatex", "-interaction=nonstopmode", "-halt-on-error", "f.tex")
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Errorf("preset %q non compila con xelatex: %v\n%s", id, err, out)
		}
	}
}
