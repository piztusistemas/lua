package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func novaBovedaProba(t *testing.T) *boveda {
	t.Helper()
	d := t.TempDir()
	ficheiros := map[string]string{
		"raw/fotosintese.md":  "# Fotosíntese\nA planta usa luz.",
		"raw/sen-procesar.md": "# Outra fonte\nTexto solto.",
		"wiki/index.md":       "# Índice\n- [[Fotosintese]]\n",
		"wiki/fotosintese.md": "Fonte: [[raw/fotosintese.md]]. Ver [[clorofila]] e [[inexistente]].",
		"wiki/clorofila.md":   "Pigmento. Volver a [[fotosintese]].",
		"wiki/orfa.md":        "Ninguén me enlaza.",
		"wiki/log.md":         "## [2026-01-01] ingest | fotosintese\n",
		"CLAUDE.md":           "# Esquema\nClaude Code mantén a wiki. Edita CLAUDE.md.",
	}
	for r, c := range ficheiros {
		p := filepath.Join(d, filepath.FromSlash(r))
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	raiz, err := validarRaizBoveda(d)
	if err != nil {
		t.Fatal(err)
	}
	return &boveda{raiz}
}

func TestBovedaValidarExixeRawEWiki(t *testing.T) {
	if _, err := validarRaizBoveda(t.TempDir()); err == nil {
		t.Fatal("un cartafol baleiro non é unha bóveda")
	}
}

func TestBovedaRutasSeguras(t *testing.T) {
	b := novaBovedaProba(t)
	for _, r := range []string{"../x.md", "/etc/passwd", "wiki/../../x.md", ""} {
		if _, err := b.rutaSegura(r); err == nil {
			t.Errorf("a ruta %q debería rexeitarse", r)
		}
	}
	fora := t.TempDir()
	if err := os.Symlink(fora, filepath.Join(b.raiz, "wiki", "saida")); err == nil {
		if _, err := b.rutaSegura("wiki/saida/x.md"); err == nil {
			t.Error("unha ligazón simbólica que sae da bóveda debería rexeitarse")
		}
	}
}

func TestBovedaEscribirSoWiki(t *testing.T) {
	b := novaBovedaProba(t)
	for _, r := range []string{"raw/fotosintese.md", "CLAUDE.md", "LUA.md", "wiki/x.txt"} {
		if _, err := escribirNaBoveda(b, r, "x", false); err == nil {
			t.Errorf("non se debería poder escribir %s", r)
		}
	}
	if _, err := escribirNaBoveda(b, "wiki/conceptos/novo.md", "Ola", false); err != nil {
		t.Fatal(err)
	}
	if _, err := escribirNaBoveda(b, "wiki/log.md", "## [2026-02-02] query | x", true); err != nil {
		t.Fatal(err)
	}
	l, _ := b.ler("wiki/log.md")
	if !strings.Contains(l, "ingest | fotosintese") || !strings.Contains(l, "query | x") {
		t.Errorf("engadir debe conservar o log: %q", l)
	}
}

func TestBovedaGrafoLintEPendentes(t *testing.T) {
	b := novaBovedaProba(t)
	g := b.construirGrafo()
	if got := g.saintes["wiki/fotosintese.md"]; len(got) != 2 {
		t.Errorf("enlaces saíntes: %v", got)
	}
	if got := g.entrantes["wiki/fotosintese.md"]; len(got) != 2 { // index + clorofila
		t.Errorf("retroenlaces: %v", got)
	}
	if got := g.rotos["wiki/fotosintese.md"]; len(got) != 1 || got[0] != "inexistente" {
		t.Errorf("rotos: %v", got)
	}
	p := fontesPendentes(g)
	if len(p) != 1 || p[0] != "raw/sen-procesar.md" {
		t.Errorf("pendentes: %v", p)
	}
	l := lintBoveda(g)
	orfas := l["paxinasOrfas"].([]string)
	if len(orfas) != 1 || orfas[0] != "wiki/orfa.md" {
		t.Errorf("orfas: %v", orfas)
	}
}

func TestBovedaBuscar(t *testing.T) {
	b := novaBovedaProba(t)
	r, err := buscarNaBoveda(b, "pigmento", "wiki", 5)
	if err != nil {
		t.Fatal(err)
	}
	if r.(map[string]any)["total"].(int) != 1 {
		t.Errorf("busca: %v", r)
	}
}

func TestBovedaEsquemaLuaMD(t *testing.T) {
	b := novaBovedaProba(t)
	s := Settings{IAModel: "modelo-x"}
	txt, err := b.sincronizarEsquema(s)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(txt[strings.Index(txt, "# Esquema"):], "Claude") || !strings.Contains(txt, "Edita LUA.md") {
		t.Errorf("a copia non está adaptada:\n%s", txt)
	}
	if !strings.Contains(txt, "boveda_buscar") {
		t.Error("falta a adaptación ás ferramentas de Lua")
	}
	// CLAUDE.md orixinal intacto.
	if o, _ := os.ReadFile(filepath.Join(b.raiz, "CLAUDE.md")); !strings.Contains(string(o), "Claude Code") {
		t.Error("CLAUDE.md non se debe modificar")
	}
	// Sen cambios non se reescribe; se CLAUDE.md cambia, rexenérase.
	_ = os.WriteFile(filepath.Join(b.raiz, "CLAUDE.md"), []byte("# Novo esquema"), 0o644)
	txt2, _ := b.sincronizarEsquema(s)
	if !strings.Contains(txt2, "Novo esquema") {
		t.Error("LUA.md debería rexenerarse")
	}
}

func TestFerramentasBovedaSoComBoveda(t *testing.T) {
	a := &App{}
	for _, f := range a.ferramentasDoTurno("", "cronograma") {
		if ferramentasBoveda[f.Nome] {
			t.Errorf("%s non debería estar sen bóveda", f.Nome)
		}
	}
	a.settings.IABovedas = map[string]string{"cronograma": "/x"}
	n := 0
	for _, f := range a.ferramentasDoTurno("tarefa", "") {
		if ferramentasBoveda[f.Nome] {
			n++
		}
	}
	if n != len(ferramentasBoveda) {
		t.Errorf("co ámbito tarefa debería haber %d ferramentas boveda_*, hai %d", len(ferramentasBoveda), n)
	}
}

func TestEstruturaMarkdownAdxunto(t *testing.T) {
	e := estruturaMarkdown("---\ntags: [a]\n---\n# Tema\n## Parte\nVer [[outra]] e [x](y.md).")
	for _, c := range []string{"Frontmatter", "tags: [a]", "Tema", "  Parte", "outra", "y.md"} {
		if !strings.Contains(e, c) {
			t.Errorf("falta %q en:\n%s", c, e)
		}
	}
	if estruturaMarkdown("só texto") != "" {
		t.Error("sen estrutura debe devolver baleiro")
	}
}
