package main

// Tipo de letra do cronograma anual do informe: Open Sans, embebida en Lua
// (Apache 2.0, ver fontes_informe/) para que o PDF saia igual en calquera
// equipo sen instalar nada. Só a pode usar fontspec, é dicir xelatex ou
// lualatex: con pdflatex o cronograma cae na sans normal do documento (ver
// \fontchip en bloqueInformeCrono).

import (
	"embed"
	"os"
	"path/filepath"
)

//go:embed fontes_informe/OpenSans-*.ttf
var fontesInformeFS embed.FS

// escribirFontesInformeCrono deixa as fontes en dir/fonts/, onde as vai buscar
// o .tex (Path=fonts/).
func escribirFontesInformeCrono(dir string) error {
	destino := filepath.Join(dir, "fonts")
	if err := os.MkdirAll(destino, 0o755); err != nil {
		return err
	}
	entradas, err := fontesInformeFS.ReadDir("fontes_informe")
	if err != nil {
		return err
	}
	for _, e := range entradas {
		datos, err := fontesInformeFS.ReadFile("fontes_informe/" + e.Name())
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(destino, e.Name()), datos, 0o644); err != nil {
			return err
		}
	}
	return nil
}
