package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"edicion/api"
)

func TestMotivoVision(t *testing.T) {
	texto := strings.Repeat("Unha frase normal dun documento sen nada especial. ", 20)
	if m := motivoVision(texto, 1, 0); m != "" {
		t.Errorf("texto normal: %q", m)
	}
	if m := motivoVision("ola", 3, 0); !strings.Contains(m, "pouco texto") {
		t.Errorf("escaneado: %q", m)
	}
	if m := motivoVision(texto+"∫ x² dx = √2 ± π", 1, 0); m != "ten fórmulas" {
		t.Errorf("fórmulas: %q", m)
	}
	if m := motivoVision(texto, 1, 2); !strings.Contains(m, "figura") {
		t.Errorf("figuras: %q", m)
	}
	if !mdTenFormulasEnTexto("Resolve x^2 - 4 = 0") || mdTenFormulasEnTexto("Resolve $x^2 - 4 = 0$ e $y$") {
		t.Error("mdTenFormulasEnTexto")
	}
}

// visionFalsa: unha IA que «le» as páxinas (devolve LaTeX e unha figura) e
// pasa fórmulas a LaTeX.
type visionFalsa struct {
	mu     sync.Mutex
	imaxes int
	url    string
	token  string
}

func novaVisionFalsa(t *testing.T) *visionFalsa {
	t.Helper()
	v := &visionFalsa{}
	s := api.Novo("falso")
	s.Rexistrar("GET /ia/estado", "ia.usar", func(p *api.Peticion) (any, error) {
		return map[string]any{"configurada": true}, nil
	})
	s.Rexistrar("POST /ia/turno", "ia.usar", func(p *api.Peticion) (any, error) {
		var pet struct{ Mensaxes []IAMensaxe }
		_ = p.Corpo(&pet)
		u := pet.Mensaxes[len(pet.Mensaxes)-1]
		v.mu.Lock()
		v.imaxes += len(u.Imaxes)
		v.mu.Unlock()
		if len(u.Imaxes) > 0 {
			return IATurno{Texto: "```markdown\n<!-- páxina 1 -->\n# Exame\n\n**1.** Calcula $\\int_0^1 x^2\\,dx$.\n\n![Circuíto con dúas resistencias](figura:1:0.25,0.25,0.75,0.75)\n\n```circuitikz\n\\draw (0,0) to[R] (2,0);\n```\n```"}, nil
		}
		return IATurno{Texto: strings.ReplaceAll(u.Texto, "x^2", "$x^2$")}, nil
	})
	u, err := s.Iniciar()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Parar)
	v.url, v.token = u, s.NovoToken("buxo", "ia.usar")
	return v
}

func escribirPNG(t *testing.T, ruta string) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 200, 300))
	for x := 50; x < 150; x++ {
		img.Set(x, 150, color.Black)
	}
	f, err := os.Create(ruta)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	png.Encode(f, img)
}

func TestImaxeConVision(t *testing.T) {
	v := novaVisionFalsa(t)
	m, _ := novoMestreProba(t)
	m.ia = &ClienteIA{C: &api.Cliente{URL: v.url, Token: v.token}, Boveda: "buxo"}
	foto := filepath.Join(t.TempDir(), "Exame foto.png")
	escribirPNG(t, foto)
	m.EngadirLocais([]string{foto})
	esperar(t, func() bool { c := m.Contas(); return c[estFeito]+c[estErro] == 1 })
	if it := m.Items()[0]; it.Estado != estFeito {
		t.Fatalf("%+v", it)
	}
	md, err := os.ReadFile(filepath.Join(m.raiz, "raw", "Exame foto.md"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(md)
	for _, quere := range []string{`pdf: "pdf/Exame foto.png"`, `conversion: "ia (visión)"`, `$\int_0^1 x^2\,dx$`, "![[Exame foto-p1-1.png]]", "```circuitikz"} {
		if !strings.Contains(s, quere) {
			t.Errorf("falta %q en:\n%s", quere, s)
		}
	}
	recorte, err := os.Open(filepath.Join(m.raiz, "raw", "imaxes", "Exame foto-p1-1.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer recorte.Close()
	c, _, err := image.DecodeConfig(recorte)
	if err != nil || c.Width < 95 || c.Width > 115 || c.Height < 145 || c.Height > 165 {
		t.Errorf("recorte %dx%d (%v)", c.Width, c.Height, err)
	}
}

func TestImaxeSenIA(t *testing.T) {
	m, _ := novoMestreProba(t)
	foto := filepath.Join(t.TempDir(), "a.png")
	escribirPNG(t, foto)
	m.EngadirLocais([]string{foto})
	esperar(t, func() bool { return m.Contas()[estErro] == 1 })
	if !strings.Contains(m.Items()[0].Erro, "IA") {
		t.Errorf("%+v", m.Items()[0])
	}
}

func TestMarkdownTalCalEConLaTeX(t *testing.T) {
	m, _ := novoMestreProba(t)
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "apuntes.md"), []byte("# Tema\n\nTexto con $x$ e $y$ xa en LaTeX $z$ $w$."), 0o644)
	m.EngadirLocais([]string{filepath.Join(dir, "apuntes.md")})
	esperar(t, func() bool { return m.Contas()[estFeito] == 1 })
	d, _ := os.ReadFile(filepath.Join(m.raiz, "raw", "apuntes.md"))
	if !strings.Contains(string(d), `conversion: "markdown"`) || !strings.Contains(string(d), "# Tema") {
		t.Fatalf("%s", d)
	}
	// Con fórmulas en texto plano e IA: pásanse a LaTeX.
	v := novaVisionFalsa(t)
	m.ia = &ClienteIA{C: &api.Cliente{URL: v.url, Token: v.token}, Boveda: "buxo"}
	os.WriteFile(filepath.Join(dir, "boletin.md"), []byte("1. Resolve x^2 - 4 = 0"), 0o644)
	m.EngadirLocais([]string{filepath.Join(dir, "boletin.md")})
	esperar(t, func() bool { return m.Contas()[estFeito] == 2 })
	d, _ = os.ReadFile(filepath.Join(m.raiz, "raw", "boletin.md"))
	if !strings.Contains(string(d), "$x^2$ - 4 = 0") {
		t.Fatalf("%s", d)
	}
}

// Un PDF de verdade con fórmulas: markitdown (falso) dá texto con símbolos e
// a IA le as páxinas.
func TestPDFConFormulasPasaAVision(t *testing.T) {
	if _, err := exec.LookPath("pdflatex"); err != nil {
		t.Skip("sen pdflatex")
	}
	if _, err := exec.LookPath("pdftoppm"); err != nil {
		t.Skip("sen pdftoppm")
	}
	dir := t.TempDir()
	tex := "\\documentclass{article}\\begin{document}Calcula $\\int_0^1 x^2\\,dx$.\\end{document}"
	os.WriteFile(filepath.Join(dir, "exame.tex"), []byte(tex), 0o644)
	cmd := exec.Command("pdflatex", "-interaction=nonstopmode", "exame.tex")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	v := novaVisionFalsa(t)
	m, log := novoMestreProba(t)
	m.ia = &ClienteIA{C: &api.Cliente{URL: v.url, Token: v.token}, Boveda: "buxo"}
	// O sidecar falso escribe «# exame» + «Texto.»: pouco texto para unha páxina.
	m.EngadirLocais([]string{filepath.Join(dir, "exame.pdf")})
	esperar(t, func() bool { c := m.Contas(); return c[estFeito]+c[estErro] == 1 })
	it := m.Items()[0]
	if it.Estado != estFeito || it.Vision == "" {
		t.Fatalf("%+v\n%v", it, log.Historial())
	}
	d, _ := os.ReadFile(filepath.Join(m.raiz, "raw", "exame.md"))
	if !strings.Contains(string(d), `$\int_0^1 x^2\,dx$`) || v.imaxes != 1 {
		t.Fatalf("imaxes %d:\n%s", v.imaxes, d)
	}
	// O modo texto non chama á IA.
	m.CambiarModo(modoTexto)
	cp := filepath.Join(dir, "outro.pdf")
	b, _ := os.ReadFile(filepath.Join(dir, "exame.pdf"))
	os.WriteFile(cp, b, 0o644)
	m.EngadirLocais([]string{cp})
	esperar(t, func() bool { return m.Contas()[estFeito] == 2 })
	if v.imaxes != 1 {
		t.Errorf("en modo texto non debe usarse a IA (imaxes %d)", v.imaxes)
	}
}
