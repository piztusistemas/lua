package documento

// API do paquete: o que usa LuaYang. O resto do paquete é o código de Yang
// (latexdoc.go, plantillas.go, latexenv_*.go) case sen cambios, para que os
// documentos saian exactamente igual ca en Yang.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"edicion/latex"
	"edicion/plantilla"
)

// Tipografia: tamaño de letra, interliñado e fonte (o botón «Axustes do
// texto» de Yang).
type Tipografia = TipografiaOpts

// Conversor pasa a LaTeX o HTML que devolve Maxima (cas.ParseText en
// FormatLatex).
type Conversor struct{ c *latexConverter }

// NovoConversor: baseDir é onde se buscan as imaxes <IMG src> do documento;
// buildDir, o cartafol da compilación.
func NovoConversor(baseDir, buildDir string) *Conversor {
	return &Conversor{c: newLatexConverter(baseDir, buildDir)}
}

// HTMLaLatex converte un fragmento.
func (c *Conversor) HTMLaLatex(fragmento string) (string, error) {
	nodos, err := parseFragment(fragmento)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, n := range nodos {
		b.WriteString(c.c.render(n))
	}
	return b.String(), nil
}

// Avisos: imaxes que non se atoparon, etiquetas HTML sen soporte...
func (c *Conversor) Avisos() []string {
	av := append([]string{}, c.c.warnings...)
	for t := range c.c.unknownTags {
		av = append(av, "etiqueta HTML sen soporte en PDF (mantívose o contido): <"+t+">")
	}
	return av
}

// PreprocesarPseudoEtiquetas substitúe <TAB> e <BP> (sen peche) por LaTeX
// protexido.
func PreprocesarPseudoEtiquetas(s string, reg *latex.RawRegistry) string {
	return preprocessPseudoTags(s, reg)
}

// SanearCorpo neutraliza \begin/\end sen ambiente e \if/\fi desequilibrados
// (contido roto da IA) para que o resto do documento compile.
func SanearCorpo(corpo string) (string, []string) {
	c, a1 := sanearAmbientesLatex(corpo)
	c, a2 := sanearCondicionaisLatex(c)
	return c, append(a1, a2...)
}

// Resultado dunha compilación.
type Resultado struct {
	PDF    []byte
	Tex    string // o .tex completo que se compilou
	Log    string
	Motor  string // o motor que compilou de verdade
	Avisos []string
}

// Compilar compón o .tex coa plantilla (nil = preámbulo por defecto) e
// compílao en dir. Se o motor preferido falla, próbanse os outros
// instalados.
func Compilar(dir, corpo, motor string, p *plantilla.Plantilla, dep *plantilla.Deposito, nomeDoc string, tip *Tipografia) (Resultado, error) {
	if p != nil {
		motor = plantilla.MotorParaPlantilla(p, motor)
	}
	if strings.TrimSpace(motor) == "" {
		motor = "xelatex"
	}
	if tip != nil && esFonteDoSistema(tip.Fonte) && !engineUnicodeNativo(motor) {
		motor = "xelatex"
	}
	pdf, tex, log, usado, avisos, err := compileLatexAuto(dir, corpo, motor, p, dep, nomeDoc, tip)
	r := Resultado{PDF: pdf, Tex: tex, Log: log, Motor: usado, Avisos: avisos}
	if err != nil {
		return r, fmt.Errorf("erro ao compilar o PDF: %w", err)
	}
	if usado != motor {
		r.Avisos = append(r.Avisos, fmt.Sprintf("%s non conseguiu compilar este documento; usouse %s.", motor, usado))
	}
	return r, nil
}

// PaxinasPNG rasteriza o PDF (unha data: URI por páxina) para a vista
// previa. Precisa pdftoppm (poppler).
func PaxinasPNG(dir string, pdf []byte) ([]string, error) {
	if !commandExists("pdftoppm") {
		return nil, fmt.Errorf("non se atopou pdftoppm (poppler): sen vista previa páxina a páxina")
	}
	ruta := filepath.Join(dir, "vista.pdf")
	if err := os.WriteFile(ruta, pdf, 0o644); err != nil {
		return nil, err
	}
	return pdfPageImages(dir, ruta)
}

// HaiLatex di se hai algún motor LaTeX instalado.
func HaiLatex() bool { return len(motoresParaProbar("")) > 0 }
