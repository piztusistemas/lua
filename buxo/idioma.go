package main

import (
	"embed"
	"encoding/json"
	"os"

	"edicion/idioma"
)

// Buxo fala o idioma de Lua: Lua pásallo en BUXO_IDIOMA ao lanzalo (unha
// variable de entorno e non un argumento, para que un Buxo vello non se
// peche cun argumento que non coñece). O galego é a fonte e a reserva.
//
//go:embed idiomas/*.json
var idiomasFS embed.FS

// configurarIdioma rexistra os dicionarios de Buxo e fixa o idioma da sesión.
func configurarIdioma() {
	for _, codigo := range []string{"es", "en", "pt"} {
		datos, err := idiomasFS.ReadFile("idiomas/" + codigo + ".json")
		if err != nil {
			continue
		}
		var dicionario map[string]string
		if json.Unmarshal(datos, &dicionario) == nil {
			idioma.Rexistrar(codigo, dicionario)
		}
	}
	idioma.SetIdioma(os.Getenv("BUXO_IDIOMA"))
}
