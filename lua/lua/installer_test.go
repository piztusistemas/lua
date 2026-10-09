package main

import "testing"

// TestLinuxLatexInstallCommand só comproba a lóxica de *selección* (que
// xestor de paquetes gaña, que argv se constrúe): nunca executa pkexec/apt,
// así que se pode correr en calquera sitio sen tocar o sistema real.
func TestLinuxLatexInstallCommand(t *testing.T) {
	argv, mgr, ok := linuxLatexInstallCommand()
	if !ok {
		t.Skip("no known package manager on this machine")
	}
	if len(argv) == 0 {
		t.Fatalf("%s: empty argv", mgr)
	}
	t.Logf("detected %s -> %v", mgr, argv)
}

// TestCheckLatexDeps runs against whatever is actually installed on this
// machine (no mocking - commandExists shells out to exec.LookPath for
// real) - CI/dev machines running the rest of this test suite already need
// a working pdflatex + pdftoppm (see latexdoc_test.go's TestGeneratePDF),
// so this should report OK here; it's really locking in the "engine
// re-derived from settings" behaviour, not the detection itself.
func TestCheckLatexDeps(t *testing.T) {
	a := &App{}

	a.settings.LatexEngine = ""
	status := a.CheckLatexDeps()
	if status.Engine != "pdflatex" {
		t.Errorf("empty LatexEngine setting should default to pdflatex, got %q", status.Engine)
	}
	if !commandExists("pdflatex") {
		t.Skip("pdflatex not found on this machine, skipping the rest")
	}
	if !status.EngineFound {
		t.Error("pdflatex is on PATH but EngineFound is false")
	}
	if !status.PdftoppmFound && commandExists("pdftoppm") {
		t.Error("pdftoppm is on PATH but PdftoppmFound is false")
	}

	a.settings.LatexEngine = "xelatex"
	status = a.CheckLatexDeps()
	if status.Engine != "xelatex" {
		t.Errorf("LatexEngine=xelatex should be reflected as-is, got %q", status.Engine)
	}

	a.settings.LatexEngine = "pdflatex"
	if !a.CheckLatexDeps().OK() {
		t.Error("OK() should be true when both engine and pdftoppm are found")
	}
}
