package main

import "edicion/latex"

// O escape de texto para LaTeX vive agora no módulo común
// (edicion/latex), compartido con Yang e cos módulos futuros. Consérvase o
// nome para non tocar as chamadas deste proxecto (informe_actas.go,
// informe_md.go, informe_cronograma.go, plantillas.go, textos_doc.go).
//
// OLLO: EscapeLatexProse non distingue contexto matemático; se algún día Lua
// compón texto con LaTeX dentro, hai que protexer eses fragmentos con
// latex.RawRegistry.Protect e resolver ao final con Resolve.
func EscapeLatexProse(s string) string { return latex.EscapeLatexProse(s) }
