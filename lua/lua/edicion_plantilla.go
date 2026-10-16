package main

// Ponte co módulo común `edicion`: o modelo das plantillas, o ficheiro da
// biblioteca (plantillas.json), a fábrica, o intercambio entre compañeiros e a
// xeración con IA viven en edicion/plantilla.
//
// Os tipos expóñense coma ALIAS (non coma copias) para que o resto do proxecto
// e os bindings de Wails manteñan os mesmos nomes e os mesmos campos JSON.

import "edicion/plantilla"

type (
	PlantillaVar       = plantilla.Var
	PlantillaLatex     = plantilla.Plantilla
	PlantillasEstado   = plantilla.Estado
	almacenPlantillas  = plantilla.Almacen
	bundlePlantilla    = plantilla.Bundle
	imaxeBundle        = plantilla.ImaxeBundle
	OfimaticaPlantilla = plantilla.Ofimatica
	Deseno             = plantilla.Deseno
	ElementoDeseno     = plantilla.Elemento
	Zonas              = plantilla.Zonas
)

// normalizarDeseno: adaptador (ver edicion/plantilla/deseno.go).
func normalizarDeseno(d *Deseno) *Deseno { return plantilla.NormalizarDeseno(d) }

const (
	marcadorCorpo            = plantilla.MarcadorCorpo
	marcadorPortada          = plantilla.MarcadorPortada
	marcadorSinatura         = plantilla.MarcadorSinatura
	formatoBundlePlantilla   = plantilla.FormatoBundle
	versionBundlePlantilla   = plantilla.VersionBundle
	extensionBundlePlantilla = plantilla.ExtensionBundle
)

// opcionsPlantilla é o que o módulo necesita de Lua para o intercambio: as
// rutas de configuración e o catálogo de textos de documento (textos_doc.go),
// que é de dominio e por iso non vive no módulo.
var opcionsPlantilla = plantilla.Opcions{
	Nome:             "Lúa",
	DirImaxes:        plantillaImaxesDir,
	NormalizarTextos: normalizarTextosDoc,
}

// depositoPlantillas é a biblioteca de Lua: o módulo pon a maquinaria (ler e
// gardar o almacén, sementar e reparar a fábrica, retirar a obsoleta,
// bifurcar, podar) e Lua as costuras: as súas rutas de configuración, a súa
// fábrica de exemplos, as súas plantillas obsoletas, a comprobación de que un
// documento segue no disco e o catálogo de textos de documento.
var depositoPlantillas = plantilla.Deposito{
	RutaAlmacen:     plantillasPath,
	DirImaxes:       plantillaImaxesDir,
	Fabrica:         plantillasDeFabrica,
	FabricaObsoleta: fabricaObsoleta,
	ExisteFicheiro:  fileExists,
	TextosIguais:    textosDocIguais,
}

// Adaptadores da biblioteca (a implementación está no módulo).
func loadAlmacenPlantillas() (almacenPlantillas, error) { return depositoPlantillas.Ler() }

func saveAlmacenPlantillas(alm almacenPlantillas) error { return depositoPlantillas.Gardar(alm) }

func imaxesDaPlantilla(id string) []string { return depositoPlantillas.Imaxes(id) }

func copiarCartafolImaxes(orixeID, destinoID string) error {
	return depositoPlantillas.CopiarImaxes(orixeID, destinoID)
}

// marcadoresDeParte delega no módulo.
func marcadoresDeParte(nome string) bool { return plantilla.MarcadoresDeParte(nome) }

// normalizarNomeVar e normalizarVariables delegan no módulo: a súa
// implementación vive unha soa vez en edicion/plantilla.
func normalizarNomeVar(nome string) string { return plantilla.NormalizarNombreVar(nome) }

func normalizarVariables(vars []PlantillaVar) []PlantillaVar {
	return plantilla.NormalizarVariables(vars)
}

// nomeFicheiroPlantilla delega no módulo (nome de ficheiro seguro para o
// diálogo de gardar).
func nomeFicheiroPlantilla(nome string) string { return plantilla.NombreFicheiro(nome) }

// importadorPlantillas adapta os métodos de *App ao que o módulo necesita para
// dar de alta unha plantilla importada: gardala coa validación da aplicación e
// escribir as súas imaxes.
type importadorPlantillas struct{ a *App }

func (i importadorPlantillas) Gardar(p PlantillaLatex) (PlantillaLatex, error) {
	return i.a.GardarPlantilla(p)
}

func (i importadorPlantillas) Imaxes(id string) []string { return imaxesDaPlantilla(id) }

func (i importadorPlantillas) SubirImaxe(id, nome, dataB64 string) error {
	_, err := i.a.SubirImaxePlantilla(SubirImaxePlantillaRequest{ID: id, FileName: nome, DataB64: dataB64})
	return err
}

// ---------- Adaptadores de composición ----------
// A implementación está en edicion/plantilla (composicion.go).

func substituirVariables(texto string, p *PlantillaLatex, auto map[string]string, baleiro string, escapar func(string) string) (string, []string) {
	return plantilla.SubstituirVariables(texto, p, auto, baleiro, escapar)
}

func paquetesDeclarados(latex string) map[string]bool { return plantilla.PaquetesDeclarados(latex) }

func motorParaPlantilla(p *PlantillaLatex, porDefecto string) string {
	return plantilla.MotorParaPlantilla(p, porDefecto)
}

func copiarImaxesPlantilla(p *PlantillaLatex, destDir string) error {
	return depositoPlantillas.CopiarImaxesPlantilla(p, destDir)
}

func copiarImaxesPlantillaMarkdown(p *PlantillaLatex, destDir string) error {
	return depositoPlantillas.CopiarImaxesPlantillaMarkdown(p, destDir)
}

func escribirImaxesPlantilla(p *PlantillaLatex, dirs ...string) error {
	return depositoPlantillas.EscribirImaxesPlantilla(p, dirs...)
}

// documentoMarkdownConPlantilla: adaptador (a composición está en edicion/plantilla).
func documentoMarkdownConPlantilla(corpo string, p *PlantillaLatex, auto map[string]string) (string, []string) {
	return plantilla.DocumentoMarkdownConPlantilla(corpo, p, auto)
}

// ---------- Preámbulo obrigatorio ----------

// paqueteRequirido é o tipo do módulo: un paquete que o documento precisa.
type paqueteRequirido = plantilla.PaqueteRequirido

// preambuloInforme son as costuras de Lua para inxectar o preámbulo.
var preambuloInforme = plantilla.Preambulo{
	UnicodeNativo:       engineUnicodeNativo,
	DeclaracionsUnicode: func() string { return latexUnicodeDeclarations },
	Babel:               babelLine,
	Marca:               "% --- engadido por Lua para o informe ---",
}

// inxectarPreambuloObrigatorio: adaptador (a lóxica está en edicion/plantilla).
func inxectarPreambuloObrigatorio(doc, engine string, paquetes []paqueteRequirido, bloque string) string {
	return plantilla.InxectarPreambuloObrigatorio(doc, engine, paquetes, bloque, preambuloInforme)
}
