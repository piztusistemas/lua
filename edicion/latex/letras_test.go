package latex

import (
	"strings"
	"testing"
)

// conFicheiros fai que Dispoñible responda segundo `ten` durante o test.
func conFicheiros(t *testing.T, ten func(string) bool) {
	t.Helper()
	orixinal := Dispoñible
	Dispoñible = ten
	t.Cleanup(func() { Dispoñible = orixinal })
}

func todos(string) bool { return true }

// TestTipografiaPapeis: escoller unha letra cambia corpo, títulos, etiquetas e
// fórmulas (antes só o corpo: os títulos quedaban en Latin Modern Sans).
func TestTipografiaPapeis(t *testing.T) {
	conFicheiros(t, todos)
	got := TipografiaPreambulo(&TipografiaOpts{Fonte: "serif"}, "xelatex")
	for _, want := range []string{`\setmainfont{texgyretermes}`, `\setsansfont{texgyretermes}`, `\newfontfamily\fontchip{texgyreheros}`, `\setmathfont{texgyretermes-math.otf}`} {
		if !strings.Contains(got, want) {
			t.Errorf("falta %q en:\n%s", want, got)
		}
	}
	// Unha fonte do sistema vai a todos os papeis de texto.
	got = TipografiaPreambulo(&TipografiaOpts{Fonte: "DejaVu Serif"}, "xelatex")
	for _, want := range []string{`\setmainfont{DejaVu Serif}`, `\setsansfont{DejaVu Serif}`, `\newfontfamily\fontchip{DejaVu Serif}`} {
		if !strings.Contains(got, want) {
			t.Errorf("fonte do sistema: falta %q en:\n%s", want, got)
		}
	}
	if got := TipografiaPreambulo(&TipografiaOpts{Fonte: "DejaVu Serif"}, "pdflatex"); got != "" {
		t.Errorf("fonte do sistema en pdflatex debería dar '': %q", got)
	}
	// pdflatex: familias NFSS para corpo e títulos.
	got = TipografiaPreambulo(&TipografiaOpts{Fonte: "palatino"}, "pdflatex")
	if !strings.Contains(got, `\renewcommand{\rmdefault}{qpl}`) || !strings.Contains(got, `\renewcommand{\sfdefault}{qpl}`) {
		t.Errorf("pdflatex palatino: %q", got)
	}
	// Inxección de LaTeX no nome.
	got = TipografiaPreambulo(&TipografiaOpts{Fonte: "X}\\input{/etc/passwd}{"}, "xelatex")
	if strings.Contains(got, `\input`) {
		t.Errorf("nome de fonte sen sanear: %q", got)
	}
}

// TestLetrasSubstitutas: se falta o ficheiro dunha familia úsase a substituta
// e avísase; se falta a letra das fórmulas, quedan as de sempre.
func TestLetrasSubstitutas(t *testing.T) {
	conFicheiros(t, func(f string) bool { return strings.HasPrefix(f, "texgyre") && !strings.Contains(f, "math") })
	// Source Serif non a leva Lúa: sen ela en TeX Live, a substituta.
	x := XerarLetras(LetrasDe("clasica", "sourceserif", "", "", ""), "xelatex")
	if strings.Contains(x.Preambulo, "SourceSerif4") || !strings.Contains(x.Preambulo, `\setmainfont{texgyretermes}`) {
		t.Errorf("Source Serif debería substituírse por Termes:\n%s", x.Preambulo)
	}
	if len(x.Avisos) == 0 {
		t.Error("a substitución ten que avisar")
	}
	// Sen unicode-math nin a letra das fórmulas en TeX Live, as de sempre.
	if x.Mates != "" {
		t.Errorf("sen Termes Math non se carga unicode-math: %q", x.Mates)
	}
	// Fira vai con Lúa: úsase se está en fonts/ e, se non (un exame), Heros.
	x = XerarLetras(LetrasDe("moderna", "", "", "", ""), "xelatex")
	if !strings.Contains(x.Preambulo, `\IfFileExists{fonts/FiraSans-Regular.otf}{\setmainfont{FiraSans}[Path=fonts/`) ||
		!strings.Contains(x.Preambulo, `}{\setmainfont{texgyreheros}`) {
		t.Errorf("Fira desde fonts/ con Heros de reserva:\n%s", x.Preambulo)
	}
	if len(x.Ficheiros) != 4 {
		t.Errorf("os ficheiros de Fira que hai que escribir: %v", x.Ficheiros)
	}
	// Open Sans vai sempre (Lúa escríbea en fonts/).
	x = XerarLetras(LetrasDe("lua", "", "", "", ""), "xelatex")
	if !strings.Contains(x.Preambulo, `\IfFileExists{fonts/OpenSans-Regular.ttf}{\newfontfamily\fontchip{OpenSans}[Path=fonts/,`) {
		t.Errorf("as etiquetas de Lúa van en Open Sans:\n%s", x.Preambulo)
	}
}

func TestLetrasDePapeis(t *testing.T) {
	l := LetrasDe("clasica", "", "heros", "", "lm")
	if l.Corpo != "termes" || l.Titulos != "heros" || l.Etiquetas != "heros" || l.Mates != "" {
		t.Errorf("escollas por papel: %+v", l)
	}
	if c, ok := CombinacionPorID("inventada"); ok || c.ID != "lua" {
		t.Errorf("unha combinación descoñecida é a de Lúa: %+v %v", c, ok)
	}
	if !EsFonteDoSistema("Cambria") || EsFonteDoSistema("clasica") || EsFonteDoSistema("serif") || EsFonteDoSistema("") {
		t.Error("EsFonteDoSistema")
	}
}

func TestDeclaraLetras(t *testing.T) {
	if !DeclaraLetras("\\usepackage{fontspec}\n\\setmainfont{Georgia}") || !DeclaraLetras("\\usepackage{mathptmx}") {
		t.Error("unha plantilla con \\setmainfont ou mathptmx fixa a súa letra")
	}
	if DeclaraLetras("\\usepackage{fontspec}\\usepackage{geometry}") {
		t.Error("fontspec só non fixa ningunha letra")
	}
}
