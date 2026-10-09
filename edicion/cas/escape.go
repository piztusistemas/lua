package cas

import "edicion/latex"

// O escape vive agora no módulo común (edicion/latex), compartido coas
// aplicacións e cos módulos futuros. Consérvanse aquí os mesmos nomes
// -exportados e internos- para non tocar as chamadas de maxima.go, tags.go
// nin os seus tests.
//
// OLLO: EscapeLatexProse non distingue contexto matemático. Para texto que
// poida conter LaTeX hai que protexer eses fragmentos con
// latex.RawRegistry.Protect e resolver ao final con Resolve (ver
// escapeOutsideMath en latexdoc.go).
func EscapeLatex(s string) string { return latex.EscapeLatex(s) }

func escapeLatex(s string) string { return latex.EscapeLatex(s) }

func EscapeLatexProse(s string) string { return latex.EscapeLatexProse(s) }

func collapseWhitespace(s string) string { return latex.CollapseWhitespace(s) }
