package main

// Intercambio de PLANTILLAS entre compañeiros: a lóxica (o bundle .json con
// todo dentro, o marcador de formato, as imaxes en base64) vive unha soa vez en
// edicion/plantilla, compartida con Yang. Aquí queda o encaixe coa aplicación:
// os diálogos de Wails e os dous bindings.
//
// Unha plantilla non vive nun só ficheiro: o texto vai en plantillas.json e as
// imaxes nun cartafol á parte (<config>/lua/plantillas/<id>/), así que para
// compartila hai que xuntar as dúas cousas nun ficheiro portable. O que marca
// que ese .json é unha plantilla non é a extensión senón o campo "formato" de
// dentro: as versións novas escriben un marcador neutro e aceptan tamén os
// históricos ("yang-plantilla" e "lua-plantilla"), así que un bundle exportado
// por calquera das dúas aplicacións ábrese na outra.

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"edicion/plantilla"

	"github.com/wailsapp/wails/v3/pkg/application"
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

// ExportarPlantilla garda unha plantilla da biblioteca nun ficheiro .json
// (texto + imaxes en base64) para compartila cos compañeiros. Devolve a ruta
// escollida, ou "" se se cancelou o diálogo.
func (a *App) ExportarPlantilla(id string) (string, error) {
	alm, err := loadAlmacenPlantillas()
	if err != nil {
		return "", err
	}
	var atopada *PlantillaLatex
	for i := range alm.Plantillas {
		if alm.Plantillas[i].ID == id {
			atopada = &alm.Plantillas[i]
			break
		}
	}
	if atopada == nil {
		return "", fmt.Errorf("non existe esa plantilla")
	}

	bundle, err := construirBundlePlantilla(*atopada)
	if err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return "", err
	}

	base := nomeFicheiroPlantilla(atopada.Nome) + extensionBundlePlantilla
	path, err := saveFileDialogCompat("Exportar plantilla", "", base,
		"Plantilla de Lúa (*"+extensionBundlePlantilla+")", "*"+extensionBundlePlantilla)
	if err != nil || path == "" {
		return "", err
	}
	if !strings.HasSuffix(strings.ToLower(path), extensionBundlePlantilla) {
		path += extensionBundlePlantilla
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// ImportarPlantilla abre un diálogo para escoller un ficheiro .json e engádeo á
// biblioteca coma plantilla nova (ver importarBundlePlantilla). Devolve a
// plantilla gardada; se se cancela o diálogo devolve unha PlantillaLatex baleira
// (ID "") sen erro, para que o frontend o distinga.
func (a *App) ImportarPlantilla() (PlantillaLatex, error) {
	path, err := application.Get().Dialog.OpenFile().
		SetTitle("Importar plantilla").
		AddFilter("Plantilla (*"+extensionBundlePlantilla+")", "*"+extensionBundlePlantilla).
		PromptForSingleSelection()
	if err != nil || path == "" {
		return PlantillaLatex{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return PlantillaLatex{}, err
	}
	return a.importarBundlePlantilla(data)
}
