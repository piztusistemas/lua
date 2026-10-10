package main

import (
	"embed"
	"encoding/json"

	"edicion/idioma"
)

// Os mesmos dicionarios que importa o frontend (idiomas/*.json) serven tamén
// para as mensaxes que xera o Go: as claves "go.*" só as usa o Go, e o resto
// queda dispoñible por se algunha mensaxe reutiliza un texto da interface.
//
//go:embed idiomas/*.json
var idiomasFS embed.FS

// rexistrarIdiomasGo le os tres dicionarios embebidos. Faise ao arrincar e
// cada vez que se gardan as Opcións: custa pouco e así ningún estado vello
// (un test que restableceu o tradutor, por exemplo) deixa o Go sen eles.
func rexistrarIdiomasGo() {
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
}

// idiomaEfectivo é o idioma da interface: o escollido en Opcións ou, se non
// hai, o de Piztu. É o mesmo criterio que usa o frontend.
func (a *App) idiomaEfectivo() string {
	if a.settings.Idioma != "" {
		return a.settings.Idioma
	}
	return a.idiomaPiztu
}

// aplicarIdiomaGo fai que os erros e avisos do Go saian no idioma efectivo.
func (a *App) aplicarIdiomaGo() {
	rexistrarIdiomasGo()
	idioma.SetIdioma(a.idiomaEfectivo())
}
