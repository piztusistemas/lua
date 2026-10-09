package plantilla

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSubstituirVariables(t *testing.T) {
	p := &Plantilla{
		Latex: "x",
		Variables: []Var{
			{Nome: "CENTRO", Valor: "IES Xistral"},
			{Nome: "VAZIA", Valor: "   "},
			{Nome: "UNIDADE", Valor: "unidade_2"},
		},
	}
	auto := map[string]string{"DATA": "01/01/2026", "VAZIA": "do curso"}

	out, avisos := SubstituirVariables("{{CENTRO}} · {{DATA}} · {{VAZIA}} · {{SEN_DEFINIR}}", p, auto, "\\mbox{}", nil)
	if !strings.Contains(out, "IES Xistral") || !strings.Contains(out, "01/01/2026") {
		t.Errorf("non substituíu as variables: %q", out)
	}
	// Un valor baleiro na plantilla NON pisa o automático.
	if !strings.Contains(out, "do curso") {
		t.Errorf("a variable baleira debería deixar o valor automático: %q", out)
	}
	// Un marcador sen definir queda co recheo e avisa (unha soa vez).
	if !strings.Contains(out, "\\mbox{}") {
		t.Errorf("o marcador sen definir debería quedar co recheo: %q", out)
	}
	if len(avisos) != 1 || !strings.Contains(avisos[0], "SEN_DEFINIR") {
		t.Errorf("avisos inesperados: %+v", avisos)
	}

	// Os marcadores de parte non se tocan aquí.
	out, _ = SubstituirVariables("{{CORPO}} {{PORTADA}} {{SINATURA}}", p, nil, "", nil)
	if out != "{{CORPO}} {{PORTADA}} {{SINATURA}}" {
		t.Errorf("os marcadores de parte non se poden substituír: %q", out)
	}

	// O escapador aplícase ao VALOR, non á maqueta.
	out, _ = SubstituirVariables(`\textbf{{{UNIDADE}}}`, p, nil, "", func(s string) string {
		return strings.ReplaceAll(s, "_", "\\_")
	})
	if out != `\textbf{unidade\_2}` {
		t.Errorf("o valor non se escapou: %q", out)
	}

	// Sen plantilla, só as automáticas.
	out, avisos = SubstituirVariables("{{DATA}}", nil, auto, "", nil)
	if out != "01/01/2026" || len(avisos) != 0 {
		t.Errorf("sen plantilla: %q %+v", out, avisos)
	}
}

func TestPaquetesDeclarados(t *testing.T) {
	latex := `\usepackage{graphicx}
\usepackage[utf8]{inputenc}
\RequirePackage{tikz, pgfplots}
% \usepackage{comentado}`
	paquetes := PaquetesDeclarados(latex)
	for _, nome := range []string{"graphicx", "inputenc", "tikz", "pgfplots"} {
		if !paquetes[nome] {
			t.Errorf("falta o paquete %q: %+v", nome, paquetes)
		}
	}
	// A expresión regular non entende comentarios de LaTeX: un paquete
	// comentado tamén conta (comportamento histórico, consérvase tal cal).
	if !paquetes["comentado"] {
		t.Error("cambio de comportamento: a expresión xa non colle os paquetes comentados")
	}
	if len(PaquetesDeclarados("sen paquetes")) != 0 {
		t.Error("sen \\usepackage non pode haber paquetes")
	}
}

func TestMotorParaPlantilla(t *testing.T) {
	if got := MotorParaPlantilla(&Plantilla{Motor: "lualatex"}, "pdflatex"); got != "lualatex" {
		t.Errorf("o motor da plantilla ten prioridade: %q", got)
	}
	if got := MotorParaPlantilla(&Plantilla{}, "pdflatex"); got != "pdflatex" {
		t.Errorf("sen motor manda o por defecto: %q", got)
	}
	if got := MotorParaPlantilla(nil, "xelatex"); got != "xelatex" {
		t.Errorf("sen plantilla manda o por defecto: %q", got)
	}
}

func TestDepositoEscribirImaxesPlantilla(t *testing.T) {
	d := depositoProbas(t, nil, nil)
	if err := d.EscribirImaxesPlantilla(nil, t.TempDir()); err != nil {
		t.Errorf("unha plantilla nil non é erro: %v", err)
	}
	if err := d.EscribirImaxesPlantilla(&Plantilla{ID: "sen-imaxes"}, t.TempDir()); err != nil {
		t.Errorf("unha plantilla sen imaxes non é erro: %v", err)
	}

	dirOrixe, _ := d.DirImaxes("con-imaxes")
	if err := os.MkdirAll(dirOrixe, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dirOrixe, "logo.png"), []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	destino := t.TempDir()
	if err := d.CopiarImaxesPlantilla(&Plantilla{ID: "con-imaxes"}, destino); err != nil {
		t.Fatalf("CopiarImaxesPlantilla: %v", err)
	}
	for _, ruta := range []string{filepath.Join(destino, "logo.png"), filepath.Join(destino, "images", "logo.png")} {
		if _, err := os.Stat(ruta); err != nil {
			t.Errorf("falta %s: %v", ruta, err)
		}
	}
	// A variante de Markdown só escribe en images/.
	destinoMd := t.TempDir()
	if err := d.CopiarImaxesPlantillaMarkdown(&Plantilla{ID: "con-imaxes"}, destinoMd); err != nil {
		t.Fatalf("CopiarImaxesPlantillaMarkdown: %v", err)
	}
	if _, err := os.Stat(filepath.Join(destinoMd, "images", "logo.png")); err != nil {
		t.Errorf("falta a imaxe en images/: %v", err)
	}
	if _, err := os.Stat(filepath.Join(destinoMd, "logo.png")); !os.IsNotExist(err) {
		t.Errorf("a variante de Markdown non debe copiar o logo solto (err=%v)", err)
	}
}

func TestDocumentoMarkdownConPlantilla(t *testing.T) {
	auto := map[string]string{"DATA": "01/01/2026"}
	p := &Plantilla{Markdown: "# {{TITULO}}\n\n{{CORPO}}\n\n{{DATA}}"}

	// Sen plantilla ou sen parte Markdown, o corpo tal cal.
	if out, avisos := DocumentoMarkdownConPlantilla("corpo", nil, auto); out != "corpo" || avisos != nil {
		t.Errorf("sen plantilla: %q %+v", out, avisos)
	}
	if out, _ := DocumentoMarkdownConPlantilla("corpo", &Plantilla{}, auto); out != "corpo" {
		t.Errorf("sen parte Markdown: %q", out)
	}

	out, avisos := DocumentoMarkdownConPlantilla("O MEU CORPO", p, auto)
	if !strings.Contains(out, "O MEU CORPO") {
		t.Errorf("o corpo non entrou: %q", out)
	}
	if strings.Contains(out, "{{CORPO}}") {
		t.Errorf("o marcador do corpo tiña que quedar substituído: %q", out)
	}
	if !strings.Contains(out, "01/01/2026") {
		t.Errorf("non substituíu as automáticas: %q", out)
	}
	// {{TITULO}} non está definida: queda baleira e avisa (nun paso posterior,
	// sobre o texto xa composto, non se reinterpreta o corpo).
	if len(avisos) != 1 || !strings.Contains(avisos[0], "TITULO") {
		t.Errorf("avisos inesperados: %+v", avisos)
	}
}

func preambuloProbas() Preambulo {
	return Preambulo{
		UnicodeNativo:       func(engine string) bool { return engine == "xelatex" },
		DeclaracionsUnicode: func() string { return "\\DeclareUnicodeCharacter{2212}{-}\n" },
		Babel:               func() string { return "\\usepackage[galician]{babel}\n" },
		Marca:               "% --- engadido por Probe ---",
	}
}

func TestInxectarPreambuloObrigatorio(t *testing.T) {
	pre := preambuloProbas()
	doc := "\\documentclass{article}\n\\begin{document}\n{{CORPO}}\n\\end{document}"

	out := InxectarPreambuloObrigatorio(doc, "pdflatex",
		[]PaqueteRequirido{{"graphicx", `\usepackage{graphicx}`}}, "", pre)
	for _, quero := range []string{
		`\usepackage[utf8]{inputenc}`, `\usepackage[T1]{fontenc}`,
		`\usepackage{graphicx}`, `\usepackage[galician]{babel}`,
		`\DeclareUnicodeCharacter{2212}{-}`, "% --- engadido por Probe ---",
	} {
		if !strings.Contains(out, quero) {
			t.Errorf("falta %q en:\n%s", quero, out)
		}
	}
	if strings.Index(out, "engadido por Probe") > strings.Index(out, `\begin{document}`) {
		t.Error("o bloque ten que ir antes de \\begin{document}")
	}
	if !strings.HasPrefix(out, "\\documentclass{article}\n") {
		t.Error("a cabeceira do documento non se pode tocar")
	}

	// Motor Unicode: fontspec no canto de inputenc/fontenc.
	outU := InxectarPreambuloObrigatorio(doc, "xelatex", nil, "", pre)
	if !strings.Contains(outU, `\usepackage{fontspec}`) || strings.Contains(outU, "inputenc") {
		t.Errorf("con motor Unicode vai fontspec: %s", outU)
	}

	// Un paquete xa declarado non se duplica.
	doc2 := "\\documentclass{article}\n\\usepackage{graphicx}\n\\begin{document}\nx\\end{document}"
	out2 := InxectarPreambuloObrigatorio(doc2, "pdflatex",
		[]PaqueteRequirido{{"graphicx", `\usepackage{graphicx}`}}, "", pre)
	if strings.Count(out2, `\usepackage{graphicx}`) != 1 {
		t.Errorf("duplicouse un paquete xa declarado:\n%s", out2)
	}

	// Nunha plantilla completa (nada que engadir) o documento devólvese intacto.
	completo := "\\documentclass{article}\n\\usepackage{fontspec}\n\\usepackage{babel}\n\\begin{document}\nx\\end{document}"
	if got := InxectarPreambuloObrigatorio(completo, "xelatex", nil, "", pre); got != completo {
		t.Errorf("unha plantilla completa non se toca:\n%s", got)
	}

	// O bloque extra (xeometría da maqueta) inxéctase aínda que non falte nada.
	conBloque := InxectarPreambuloObrigatorio(completo, "xelatex", nil, `\geometry{headheight=52pt}`, pre)
	if !strings.Contains(conBloque, `\geometry{headheight=52pt}`) {
		t.Errorf("o bloque extra non se inxectou:\n%s", conBloque)
	}

	// Sen \begin{document} non hai onde inxectar.
	senInicio := "\\documentclass{article}\n\\usepackage{tikz}\n"
	if got := InxectarPreambuloObrigatorio(senInicio, "pdflatex", nil, "", pre); got != senInicio {
		t.Errorf("sen \\begin{document} hai que devolver o documento tal cal:\n%s", got)
	}
}
