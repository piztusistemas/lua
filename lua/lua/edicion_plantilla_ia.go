package main

// Xeración de plantillas coa IA: a lóxica (as tres fontes —descrición,
// documentos de mostra e URL—, a descarga da páxina, a extracción de texto, o
// parseo tolerante da resposta e a reparación das barras do LaTeX) vive en
// edicion/plantilla. Aquí queda o encaixe coa aplicación: o provedor de IA cos
// seus axustes, o prompt de sistema, pdftotext e os axudantes de resposta.

import "edicion/plantilla"

type (
	DocumentoMostra    = plantilla.DocumentoMostra
	PlantillaIARequest = plantilla.PlantillaIARequest
)

// xeradorPlantillaIA arma o xerador do módulo coas costuras de Lua.
func (a *App) xeradorPlantillaIA() *plantilla.XeradorIA {
	return &plantilla.XeradorIA{
		Sistema:         systemPromptPlantilla,
		ChamarIA:        func(sistema, peticion string) (string, error) { return chamarIA(a.settings, sistema, peticion) },
		ExtraerTextoPDF: extraerTextoPDF,
		LimparValados:   limparValadosMarkdown,
		TruncarRunas:    truncarRunas,
	}
}

// XerarPlantillaIA constrúe a petición coas fontes dispoñibles e devolve a
// plantilla proposta, SEN gardala.
func (a *App) XerarPlantillaIA(req PlantillaIARequest) (PlantillaLatex, error) {
	return a.xeradorPlantillaIA().Xerar(req)
}

// parsearPlantillaIA le o obxecto JSON da resposta e valida o mínimo, sen
// chamar a ningún provedor (para as probas).
func parsearPlantillaIA(raw string) (PlantillaLatex, error) {
	return (&plantilla.XeradorIA{LimparValados: limparValadosMarkdown}).Parsear(raw)
}

// repararBarrasLatex delega no módulo.
func repararBarrasLatex(latex string) string { return plantilla.RepararBarrasLatex(latex) }

// textoVisibleHTML delega no módulo.
func textoVisibleHTML(fonte string) string { return plantilla.TextoVisibleHTML(fonte) }
