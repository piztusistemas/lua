package main

import (
	"strings"
	"testing"
)

func TestOpcionsFormulasNormalizar(t *testing.T) {
	for _, c := range []struct{ en, sae int }{{0, 100}, {90, 100}, {100, 100}, {120, 120}, {125, 130}, {150, 150}, {200, 150}} {
		if g := (opcionsFormulas{Escala: c.en}).normalizar().Escala; g != c.sae {
			t.Errorf("escala %d → %d, esperaba %d", c.en, g, c.sae)
		}
	}
	s := Settings{FormulasEstilo: "grandes", FormulasEscala: 130}
	if o := opcionsFormulasDe(s); !o.Grandes || o.Escala != 130 {
		t.Errorf("das Opcións: %+v", o)
	}
	if o := opcionsFormulasDe(Settings{}); o.Grandes || o.Escala != 100 {
		t.Errorf("por defecto: %+v", o)
	}
}

func TestOpcionsFormulasLatex(t *testing.T) {
	// Por defecto, o mesmo ca antes.
	def := opcionsFormulas{}.latex()
	if !strings.Contains(def, `\def\luaformula#1{$#1$}`) || !strings.Contains(def, `\def\luaformulacentrada#1{\[#1\]}`) {
		t.Errorf("por defecto:\n%s", def)
	}
	grandes := opcionsFormulas{Grandes: true, Escala: 100}.latex()
	if !strings.Contains(grandes, `$\displaystyle #1$`) || strings.Contains(grandes, `scalebox`) {
		t.Errorf("grandes:\n%s", grandes)
	}
	escala := opcionsFormulas{Escala: 120}.latex()
	if !strings.Contains(escala, `\scalebox{1.2}{$#1$}`) || !strings.Contains(escala, `\ifdefined\scalebox`) {
		t.Errorf("escala:\n%s", escala)
	}
}

func TestLatexTramosEscribeLuaformula(t *testing.T) {
	got := latexTramosCrono([]cronoTramo{{Texto: "Divide "}, {Math: `\frac{a}{b}`}, {Math: `E=mc^2`, Display: true}})
	if !strings.Contains(got, `\luaformula{\frac{a}{b}}`) || !strings.Contains(got, `\luaformulacentrada{E=mc^2}`) {
		t.Errorf("tramos: %q", got)
	}
}

func TestFormularioLevaAsOpcionsDasFormulas(t *testing.T) {
	f, d := formularioExame()
	p := PedidoFormulario{Formulario: f, Datos: d}
	tex, _, err := latexFormulario(p, "pdflatex", nil, t.TempDir(), agoraFormulario, nil, "exame")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(tex, `\def\luaformula#1{$#1$}`) {
		t.Error("sen opcións, o formulario ten que levar as definicións por defecto")
	}
	p.formulas = opcionsFormulas{Grandes: true, Escala: 130}
	tex, _, err = latexFormulario(p, "pdflatex", nil, t.TempDir(), agoraFormulario, nil, "exame")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(tex, `\scalebox{1.3}{$\displaystyle #1$}`) {
		t.Error("o formulario non leva as opcións das fórmulas")
	}
	if i, j := strings.Index(tex, `\begin{document}`), strings.Index(tex, `\def\luaformula#1`); i < 0 || j < i {
		t.Error("as definicións van no corpo, para que cheguen tamén coa plantilla")
	}
}

func TestPezasFormulasOfimatica(t *testing.T) {
	tr := []cronoTramo{{Math: `\frac{a}{b}`}, {Math: `x`, Display: true}}
	p := pezasFormulas(tr, opcionsFormulas{})
	if p[0] != `$\frac{a}{b}$` || p[1] != `$\displaystyle x$` {
		t.Errorf("por defecto: %q", p)
	}
	p = pezasFormulas(tr, opcionsFormulas{Grandes: true})
	if p[0] != `$\displaystyle \frac{a}{b}$` {
		t.Errorf("grandes: %q", p)
	}
}
