package latex

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestBabelGalegoConservaPuntoEnMatematicas: babel-galician con
// shorthands=off deixa o «.» activo en modo matemático e sen definición,
// así que $12.75$ saía «1275» no PDF (e $0.3$, «03»).
func TestBabelGalegoConservaPuntoEnMatematicas(t *testing.T) {
	for _, ferramenta := range []string{"pdflatex", "pdftotext", "kpsewhich"} {
		if _, err := exec.LookPath(ferramenta); err != nil {
			t.Skip("sen " + ferramenta)
		}
	}
	if !KpsewhichFound("galician.ldf") {
		t.Skip("sen galician.ldf")
	}
	dir := t.TempDir()
	doc := "\\documentclass{article}\n" + BabelLine() + "\\usepackage{amsmath}\n\\begin{document}\nA $12.75$ B $0{,}3$ C $a<b$\n\n\\selectlanguage{galician}D $1.5$\n\\end{document}\n"
	if err := os.WriteFile(filepath.Join(dir, "p.tex"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "pdflatex", "-interaction=nonstopmode", "-halt-on-error", "p.tex")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("pdflatex: %v\n%s", err, out)
	}
	out, err := exec.CommandContext(ctx, "pdftotext", filepath.Join(dir, "p.pdf"), "-").Output()
	if err != nil {
		t.Fatal(err)
	}
	texto := strings.Join(strings.Fields(string(out)), " ")
	for _, quero := range []string{"A 12.75", "B 0,3", "a < b", "D 1.5"} {
		if !strings.Contains(texto, quero) {
			t.Errorf("falta %q no PDF: %q", quero, texto)
		}
	}
}
