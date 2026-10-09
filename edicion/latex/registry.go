package latex

import (
	"fmt"
	"strings"
)

// RawRegistry garda fragmentos de LaTeX xa listos (matemáticas de Maxima,
// gráficos, <TAB>, hiperligazóns...) detrás de tokens opacos.
//
// É a metade que lle faltaba ao escaper: EscapeLatexProse non distingue
// contexto matemático, así que un "$x_1$" acabaría como "\$x\_1\$". O
// circuíto correcto é sempre o mesmo, e a orde importa:
//
//  1. PROTEXER — cada fragmento xa en LaTeX pasa por Protect e convértese nun
//     token alfanumérico opaco ("MATEXERAWBLOCK1ZZ"), seguro de pasar polo
//     analizador de HTML e polo escaper.
//  2. COMPOÑER — o texto resultante (tokens + prosa) escápase con
//     EscapeLatexProse sen medo.
//  3. RESOLVER — Resolve devolve o LaTeX real no seu sitio.
//  4. SANEAR — e só entón se neutralizan os \begin/\end malformados que
//     quedaran soltos.
//
// Trocar esa orde (p. ex. resolver antes de escapar) rompe os documentos con
// matemáticas de xeito silencioso.
type RawRegistry struct {
	blocks map[string]string
	n      int
}

// NovoRawRegistry crea un rexistro baleiro.
func NovoRawRegistry() *RawRegistry {
	return &RawRegistry{blocks: map[string]string{}}
}

// Protect garda s e devolve o token que o substitúe.
func (r *RawRegistry) Protect(s string) string {
	r.n++
	token := fmt.Sprintf("MATEXERAWBLOCK%dZZ", r.n)
	r.blocks[token] = s
	return token
}

// Resolve devolve os fragmentos gardados ao seu sitio.
func (r *RawRegistry) Resolve(s string) string {
	for token, real := range r.blocks {
		s = strings.ReplaceAll(s, token, real)
	}
	return s
}
