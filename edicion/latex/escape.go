// Package latex reúne a infraestrutura LaTeX común: escape de texto, o
// rexistro de fragmentos opacos e (a partir da fase 3) preámbulos,
// compilación multi-motor, tipografía e PDF→imaxes.
package latex

import "strings"

var latexEscaper = strings.NewReplacer(
	`\`, `\textbackslash{}`,
	`&`, `\&`,
	`%`, `\%`,
	`$`, `\$`,
	`#`, `\#`,
	`_`, `\_`,
	`{`, `\{`,
	`}`, `\}`,
	`~`, `\textasciitilde{}`,
	`^`, `\textasciicircum{}`,
)

// EscapeLatex escapa os caracteres especiais de LaTeX nun texto plano que NON
// é xa fonte TeX (mensaxes de erro, texto corrente).
//
// OLLO: non distingue contexto matemático. Para texto que poida conter
// matemáticas hai que protexer primeiro as partes xa en LaTeX con
// RawRegistry.Protect e resolver ao final con Resolve; ver EscapeLatexProse.
func EscapeLatex(s string) string {
	return latexEscaper.Replace(s)
}

// CollapseWhitespace achata cada secuencia de espazos nun só e recorta os
// extremos, de xeito que o resultado non pode conter unha liña baleira.
//
// Existe por un fallo concreto que mata o documento: Maxima informa dun erro
// de sintaxe en varias liñas e a liña do "caret" vai seguida dunha baleira.
// Unha liña baleira é un \par para TeX, e o argumento de \textcolor non é
// \long, así que pdflatex aborta con "! Paragraph ended before \@textcolor
// was complete"; con -halt-on-error iso remata a compilación enteira, de
// xeito que un único <EVAL> mal escrito nun exercicio facía fallar TODO o
// exame no canto de amosar ese erro en vermello onde corresponde.
// Non se perde nada útil: o aliñamento do caret só significa algo nun
// terminal monoespazado, e isto vaise compor como parágrafo normal, onde
// LaTeX xa colapsa os espazos.
func CollapseWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// mathTextSymbols mapea os caracteres Unicode matemáticos, relacionais e
// gregos que aparecen coma PROSA (escritos á man ou pegados dun procesador de
// textos) a un comando LaTeX que si os debuxa, en vez do codepoint crú. A
// fonte de TEXTO de LaTeX (Computer Modern/Latin Modern) non ten eses glifos:
// pasalos tal cal aborta o PDF con "Unicode character ... not set up for use
// with LaTeX" (pdflatex con -halt-on-error: falla o documento enteiro) ou
// perde o glifo cun "Missing character" (xelatex/lualatex, cuxa fonte por
// defecto ten o mesmo oco). Encaminalos por \ensuremath{...} usa a fonte
// MATEMÁTICA, que os tres motores teñen, así que o arranxo vale igual para
// todos en vez de duplicar un axuste por motor.
var mathTextSymbols = map[rune]string{
	'−': `\ensuremath{-}`, // U+2212 MINUS SIGN (not the ASCII hyphen)
	'×': `\ensuremath{\times}`,
	'÷': `\ensuremath{\div}`,
	'±': `\ensuremath{\pm}`,
	'≈': `\ensuremath{\approx}`,
	'≤': `\ensuremath{\leq}`,
	'≥': `\ensuremath{\geq}`,
	'≠': `\ensuremath{\neq}`,
	'∞': `\ensuremath{\infty}`,
	'∑': `\ensuremath{\sum}`,
	'√': `\ensuremath{\surd}`,
	'∪': `\ensuremath{\cup}`,
	'∩': `\ensuremath{\cap}`,
	'∈': `\ensuremath{\in}`,
	'∉': `\ensuremath{\notin}`,
	'⊂': `\ensuremath{\subset}`,
	'⊆': `\ensuremath{\subseteq}`,
	'∅': `\ensuremath{\emptyset}`,
	'∖': `\ensuremath{\setminus}`,
	'ℝ': `\ensuremath{\mathbb{R}}`,
	'ℕ': `\ensuremath{\mathbb{N}}`,
	'ℤ': `\ensuremath{\mathbb{Z}}`,
	'ℚ': `\ensuremath{\mathbb{Q}}`,
	'ℂ': `\ensuremath{\mathbb{C}}`,
	'→': `\ensuremath{\rightarrow}`,
	'←': `\ensuremath{\leftarrow}`,
	'↔': `\ensuremath{\leftrightarrow}`,
	'⇒': `\ensuremath{\Rightarrow}`,
	'⇐': `\ensuremath{\Leftarrow}`,
	'⇔': `\ensuremath{\Leftrightarrow}`,
	'°': `\textdegree{}`,
	'α': `\ensuremath{\alpha}`, 'β': `\ensuremath{\beta}`, 'γ': `\ensuremath{\gamma}`,
	'δ': `\ensuremath{\delta}`, 'ε': `\ensuremath{\varepsilon}`, 'ζ': `\ensuremath{\zeta}`,
	'η': `\ensuremath{\eta}`, 'θ': `\ensuremath{\theta}`, 'ι': `\ensuremath{\iota}`,
	'κ': `\ensuremath{\kappa}`, 'λ': `\ensuremath{\lambda}`, 'μ': `\ensuremath{\mu}`,
	'ν': `\ensuremath{\nu}`, 'ξ': `\ensuremath{\xi}`, 'π': `\ensuremath{\pi}`,
	'ρ': `\ensuremath{\rho}`, 'σ': `\ensuremath{\sigma}`, 'τ': `\ensuremath{\tau}`,
	'υ': `\ensuremath{\upsilon}`, 'φ': `\ensuremath{\varphi}`, 'χ': `\ensuremath{\chi}`,
	'ψ': `\ensuremath{\psi}`, 'ω': `\ensuremath{\omega}`,
	'Γ': `\ensuremath{\Gamma}`, 'Δ': `\ensuremath{\Delta}`, 'Θ': `\ensuremath{\Theta}`,
	'Λ': `\ensuremath{\Lambda}`, 'Ξ': `\ensuremath{\Xi}`, 'Π': `\ensuremath{\Pi}`,
	'Σ': `\ensuremath{\Sigma}`, 'Φ': `\ensuremath{\Phi}`, 'Ψ': `\ensuremath{\Psi}`,
	'Ω': `\ensuremath{\Omega}`,
}

// EscapeLatexProse é EscapeLatex máis a substitución de mathTextSymbols: é o
// que hai que usar para o texto do documento (nomes, títulos, valores das
// variables dunha plantilla, descricións...).
//
// Non distingue contexto matemático, así que un "$x_1$" xa en LaTeX sairía
// escapado ("\$x\_1\$"). Quen teña texto con matemáticas debe protexer esas
// partes con RawRegistry.Protect antes e chamar a Resolve despois.
// Mantense separada de EscapeLatex porque esta última úsase tal cal en
// cadeas curtas de diagnóstico, onde eses símbolos practicamente non
// aparecen e meter \ensuremath sería inútil.
func EscapeLatexProse(s string) string {
	var b strings.Builder
	for _, r := range s {
		if repl, ok := mathTextSymbols[r]; ok {
			b.WriteString(repl)
			continue
		}
		b.WriteString(latexEscaper.Replace(string(r)))
	}
	return b.String()
}
