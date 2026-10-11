package main

// Intercambio de PLANTILLAS entre compañeiros: a lóxica (o bundle .json con
// todo dentro, o marcador de formato, as imaxes en base64) vive unha soa vez en
// edicion/plantilla, compartida con Yang. Aquí queda o encaixe coa aplicación.
// Sacar e meter plantillas fai Trasladar (trasladar.go), que usa estas funcións.
//
// Unha plantilla non vive nun só ficheiro: o texto vai en plantillas.json e as
// imaxes nun cartafol á parte (<config>/lua/plantillas/<id>/), así que para
// compartila hai que xuntar as dúas cousas nun ficheiro portable. O que marca
// que ese .json é unha plantilla non é a extensión senón o campo "formato" de
// dentro: as versións novas escriben un marcador neutro e aceptan tamén os
// históricos ("yang-plantilla" e "lua-plantilla"), así que un bundle exportado
// por calquera das dúas aplicacións ábrese na outra.

import (
	"edicion/plantilla"
)

// construirBundlePlantilla arma o bundle dunha plantilla xa gardada: o seu
// contido editable, sen os metadatos locais, e as súas imaxes lidas do cartafol
// e postas en base64.
func construirBundlePlantilla(p PlantillaLatex) (bundlePlantilla, error) {
	return plantilla.ConstruirBundle(p, opcionsPlantilla)
}

// importarBundlePlantilla valida un .json de plantilla xa lido e gárdao coma
// plantilla NOVA neste equipo: ID nova por GardarPlantilla (que reaproveita a
// validación de nome e {{CORPO}}), e despois as imaxes unha a unha polo mesmo
// camiño ca "Engadir imaxe…" do editor. Devolve a plantilla gardada, xa coa ID
// e coas imaxes resoltas.
func (a *App) importarBundlePlantilla(data []byte) (PlantillaLatex, error) {
	return plantilla.ImportarBundle(data, opcionsPlantilla, importadorPlantillas{a: a})
}
