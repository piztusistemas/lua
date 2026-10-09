package latex

import (
	"strings"
	"testing"
)

func TestEscapeLatexCaracteresEspeciais(t *testing.T) {
	casos := map[string]string{
		`\`: `\textbackslash{}`,
		`&`: `\&`,
		`%`: `\%`,
		`$`: `\$`,
		`#`: `\#`,
		`_`: `\_`,
		`{`: `\{`,
		`}`: `\}`,
		`~`: `\textasciitilde{}`,
		`^`: `\textasciicircum{}`,
	}
	for entrada, esperado := range casos {
		if got := EscapeLatex(entrada); got != esperado {
			t.Errorf("EscapeLatex(%q) = %q, esperaba %q", entrada, got, esperado)
		}
	}
}

func TestEscapeLatexProseSimbolosUnicode(t *testing.T) {
	casos := map[string]string{
		"−":     `\ensuremath{-}`,
		"π":     `\ensuremath{\pi}`,
		"°":     `\textdegree{}`,
		"ℝ":     `\ensuremath{\mathbb{R}}`,
		"x ≤ 3": `x \ensuremath{\leq} 3`,
	}
	for entrada, esperado := range casos {
		if got := EscapeLatexProse(entrada); got != esperado {
			t.Errorf("EscapeLatexProse(%q) = %q, esperaba %q", entrada, got, esperado)
		}
	}
}

// TestEscapeLatexProseNonTocaTextoNormal: a prosa corrente ten que saír
// intacta (acentos incluídos) - se isto falla, todos os informes rompen.
func TestEscapeLatexProseNonTocaTextoNormal(t *testing.T) {
	const entrada = "Departamento de Matemáticas, curso 2026/2027"
	if got := EscapeLatexProse(entrada); got != entrada {
		t.Errorf("o texto normal non debería cambiar: %q", got)
	}
}

// TestEscapeLatexProseEscapaCifraoEnProsa: un "$" solto na prosa SI ten que
// escapar; se non, LaTeX abre un modo matemático que nunca pecha.
func TestEscapeLatexProseEscapaCifraoEnProsa(t *testing.T) {
	if got := EscapeLatexProse("custa 5$"); got != `custa 5\$` {
		t.Errorf("EscapeLatexProse(%q) = %q", "custa 5$", got)
	}
}

func TestRawRegistryProtexeEResolve(t *testing.T) {
	reg := NovoRawRegistry()
	token := reg.Protect(`$x_1 + y^2$`)
	if token == `$x_1 + y^2$` {
		t.Fatal("Protect debería devolver un token, non o LaTeX orixinal")
	}
	if got := reg.Resolve("antes " + token + " despois"); got != `antes $x_1 + y^2$ despois` {
		t.Errorf("Resolve = %q", got)
	}
}

// TestProtexerEscaparResolverConservaAsMatematicas é a proba que xustifica que
// o escaper e o rexistro viaxen xuntos: sen protexer primeiro,
// EscapeLatexProse destrozaría as matemáticas ("\$x\_1 + y\^2\$"). Hoxe
// ningún test de Yang nin de Lua cubría esta combinación.
func TestProtexerEscaparResolverConservaAsMatematicas(t *testing.T) {
	const orixinal = "A área é $x_1 + y^2$ e o 50% dela"
	reg := NovoRawRegistry()

	// 1) protexer as matemáticas
	protexido := strings.Replace(orixinal, `$x_1 + y^2$`, reg.Protect(`$x_1 + y^2$`), 1)
	// 2) escapar a prosa (os tokens son alfanuméricos: o escaper non os toca)
	escapado := EscapeLatexProse(protexido)
	// 3) resolver
	final := reg.Resolve(escapado)

	const esperado = `A área é $x_1 + y^2$ e o 50\% dela`
	if final != esperado {
		t.Fatalf("circuíto protexer→escapar→resolver = %q, esperaba %q", final, esperado)
	}
	// Contraexemplo, para que quede documentado o que pasaría sen o par:
	if so := EscapeLatexProse(orixinal); so == final {
		t.Error("sen protexer, o resultado debería ser distinto (as matemáticas escaparían)")
	}
}

func TestCollapseWhitespace(t *testing.T) {
	entrada := "erro de sintaxe:\n\narredondar(ev(x^2+1;\n   ^\n\n(%i96)"
	got := CollapseWhitespace(entrada)
	if strings.Contains(got, "\n") {
		t.Errorf("CollapseWhitespace deixou un salto de liña: %q", got)
	}
	if strings.Contains(got, "  ") {
		t.Errorf("CollapseWhitespace deixou espazos dobres: %q", got)
	}
}
