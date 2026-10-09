package main

import (
	"strings"
	"testing"
)

// TestMotoresParaProbarPreferidoInexistente is the regression test for the
// "usuario ten mal seleccionado o compilador" case: if the configured
// engine doesn't exist as a command at all, motoresParaProbar must still
// offer whatever real engines ARE installed (as fallback candidates),
// instead of returning a list that only ever contains the broken choice.
func TestMotoresParaProbarPreferidoInexistente(t *testing.T) {
	candidatos := motoresParaProbar("motor-que-non-existe-xyz")
	for _, m := range candidatos {
		if m == "motor-que-non-existe-xyz" {
			t.Fatalf("motoresParaProbar non debería devolver un motor que non existe: %v", candidatos)
		}
	}
	// Se hai polo menos un motor LaTeX real instalado nesta máquina, ten
	// que aparecer na lista (é o que permite o fallback).
	for _, m := range []string{"pdflatex", "xelatex", "lualatex"} {
		if commandExists(m) {
			found := false
			for _, c := range candidatos {
				if c == m {
					found = true
				}
			}
			if !found {
				t.Errorf("%s está instalado pero non aparece entre os candidatos: %v", m, candidatos)
			}
		}
	}
}

// TestExtraerErroPDFLatexLocalizacionTrasInsertedText: os erros do tipo
// "Missing ... inserted" meten "<inserted text>" entre o "!" e a liña
// "l.NN", así que a localización non vai pegada ao erro. Log real de
// xelatex - sen isto, o profesorado recibía o erro sen dicir en que liña.
func TestExtraerErroPDFLatexLocalizacionTrasInsertedText(t *testing.T) {
	log := `Overfull \hbox (1.0pt too wide) in paragraph at lines 4--5
! Missing \endgroup inserted.
<inserted text> 
                \endgroup 
l.42 \end{tabular}
                  
No pages of output.`
	got := extraerErroPDFLatex(log)
	if !strings.Contains(got, `! Missing \endgroup inserted.`) {
		t.Errorf("falta a liña do erro: %q", got)
	}
	if !strings.Contains(got, "l.42") {
		t.Errorf("falta a localización l.42 (é o que fai o erro diagnosticable): %q", got)
	}
}

// TestExtraerErroPDFLatexNonMesturaErros: cada "!" queda coa SÚA
// localización; se un erro non ten "l.NN" antes do vindeiro "!", non se lle
// pode pegar a do erro seguinte.
func TestExtraerErroPDFLatexNonMesturaErros(t *testing.T) {
	log := `! Undefined control sequence.
l.10 \descoñecido
! Missing $ inserted.
<inserted text> 
                $
l.20 \end{document}`
	got := extraerErroPDFLatex(log)
	want := `! Undefined control sequence. | l.10 \descoñecido | ! Missing $ inserted. | l.20 \end{document}`
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

// TestExtraerErroPDFLatexSenLocalizacion: hai erros que nunca chegan a dar
// "l.NN" (o ficheiro rematou mentres se lía unha macro, por exemplo). Aí
// devólvese só a liña do erro, non a primeira "l." que apareza páxinas
// despois.
func TestExtraerErroPDFLatexSenLocalizacion(t *testing.T) {
	log := `! File ended while scanning use of \align.
<inserted text> 
                \par 
<*> doc.tex
          
` + strings.Repeat("algunha liña de recheo\n", 10) + `l.99 non me corresponde`
	got := extraerErroPDFLatex(log)
	if strings.Contains(got, "l.99") {
		t.Errorf("colleu unha localización allea moi afastada: %q", got)
	}
}

// TestExtraerErroPDFLatexContinuacion: os erros de PAQUETE parten a
// explicación en liñas "(paquete)  ..." e deixan a primeira baleira tras os
// dous puntos. Report real: o profesorado recibía "! Package fontspec
// Error:" a secas, sen o motivo. Teñen que vir xuntos nunha soa frase.
func TestExtraerErroPDFLatexContinuacion(t *testing.T) {
	log := `This is XeTeX
(/usr/local/texlive/2026/texmf-dist/tex/latex/base/article.cls)
! Package fontspec Error: 
(fontspec)                The font "Fira Sans" cannot be found; this may be
(fontspec)                but usually is not a fontspec bug.

l.12 \setsansfont{Fira Sans}
`
	got := extraerErroPDFLatex(log)
	if !strings.Contains(got, `The font "Fira Sans" cannot be found`) {
		t.Errorf("perdeuse o motivo real do erro: %q", got)
	}
	if !strings.Contains(got, "l.12") {
		t.Errorf("perdeuse a localización: %q", got)
	}
}

// TestPistaLatexAccionable: os dous fallos típicos dunha plantilla escrita
// pola IA (fonte ou paquete que ese equipo non ten) teñen que traducirse a
// unha frase que diga que facer; calquera outro erro non se toca.
func TestPistaLatexAccionable(t *testing.T) {
	fonte := pistaLatexAccionable(`(fontspec)  The font "Fira Sans" cannot be found; this may be`)
	if !strings.Contains(fonte, "Fira Sans") || !strings.Contains(fonte, "non está instalada") {
		t.Errorf("pista de fonte pouco útil: %q", fonte)
	}
	paquete := pistaLatexAccionable("! LaTeX Error: File `tcolorbox.sty' not found.")
	if !strings.Contains(paquete, "tcolorbox") || !strings.Contains(paquete, "tlmgr install tcolorbox") {
		t.Errorf("pista de paquete pouco útil: %q", paquete)
	}
	if outro := pistaLatexAccionable("! Undefined control sequence.\nl.4 \\foo"); outro != "" {
		t.Errorf("un erro non recoñecido non debe inventar pista: %q", outro)
	}
	amb := pistaLatexAccionable("! LaTeX Error: Environment cases undefined.\nl.45 \\begin{cases}")
	if !strings.Contains(amb, "cases") || !strings.Contains(amb, "<SISTEMA>") {
		t.Errorf("pista de ambiente indefinido pouco útil: %q", amb)
	}
}
