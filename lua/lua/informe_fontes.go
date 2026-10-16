package main

// Letras que leva Lúa dentro (ver fontes_informe/LEEME.txt): Open Sans
// (Apache 2.0), Fira Sans e Fira Math, Libertinus Serif e Libertinus Math (OFL),
// para que o PDF saia igual en calquera equipo sen instalar nada. Só a pode usar fontspec, é dicir xelatex ou
// lualatex: con pdflatex o cronograma cae na sans normal do documento (ver
// \fontchip en bloqueInformeCrono).

import (
	"embed"
	"os"
	"path/filepath"
)

//go:embed fontes_informe/*.ttf fontes_informe/*.otf
var fontesInformeFS embed.FS

// escribirFontesInformeCrono deixa en dir/fonts/ as fontes que usa o
// documento (as que leva Lúa: Open Sans, Fira, Libertinus), onde as vai buscar
// o .tex (Path=fonts/). Un nome que Lúa non leva ignórase.
func escribirFontesInformeCrono(dir string, ficheiros []string) error {
	if len(ficheiros) == 0 {
		return nil
	}
	destino := filepath.Join(dir, "fonts")
	if err := os.MkdirAll(destino, 0o755); err != nil {
		return err
	}
	for _, nome := range ficheiros {
		datos, err := fontesInformeFS.ReadFile("fontes_informe/" + nome)
		if err != nil {
			continue
		}
		if err := os.WriteFile(filepath.Join(destino, nome), datos, 0o644); err != nil {
			return err
		}
	}
	return nil
}
