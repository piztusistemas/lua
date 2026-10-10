package main

// Métodos de App que expón o motor de formularios ao frontend (Wails). A
// xeración de PDF, TeX e Markdown non ten métodos propios: entra polos mesmos
// XerarInformeCronograma / ExportarInforme* que o cronograma e as actas, cunha
// petición de ferramenta «formulario» (ver XerarInformeCronogramaRequest).

// ListarFormularios devolve a biblioteca de formularios (unha icona na columna
// de ferramentas por cada un), ordenada por nome.
func (a *App) ListarFormularios() ([]Formulario, error) {
	return listarFormularios()
}

// LerFormulario devolve un formulario da biblioteca polo seu id.
func (a *App) LerFormulario(id string) (Formulario, error) {
	return lerFormulario(id)
}

// GardarFormulario normaliza, valida e garda un formulario (novo ou existente)
// e devolve como quedou: co id e os ids dos campos que Lua lles puxo.
func (a *App) GardarFormulario(f Formulario) (Formulario, error) {
	return gardarFormulario(f)
}

// CambiarModoFormulario pasa un formulario a modo «deseno» ou «produccion» e
// devolve como quedou. Ao pasalo a produción péchase a xanela do deseñador, se
// está aberta (e falla se ten cambios sen gardar: primeiro hai que decidir que
// facer con eles). É o ÚNICO camiño para cambiar o modo: gardar non o toca.
func (a *App) CambiarModoFormulario(id, modo string) (Formulario, error) {
	if modo == modoProduccion {
		if err := a.pecharDeseñadorFormulario(id); err != nil {
			return Formulario{}, err
		}
	}
	return cambiarModoFormulario(id, modo)
}

// EliminarFormulario borra un formulario da biblioteca.
func (a *App) EliminarFormulario(id string) error {
	return eliminarFormulario(id)
}

// LerBibliotecaFormularios devolve a organización dos formularios: cartafoles,
// zona fixa da columna esquerda e ferramentas agochadas.
func (a *App) LerBibliotecaFormularios() (BibliotecaFormularios, error) {
	return lerBiblioteca()
}

// GardarBibliotecaFormularios garda a organización completa (a que devolve
// LerBibliotecaFormularios, modificada) e devolve como quedou, xa limpa.
func (a *App) GardarBibliotecaFormularios(b BibliotecaFormularios) (BibliotecaFormularios, error) {
	return gardarBiblioteca(b)
}

// CrearCarpetaFormularios engade un cartafol e devolve a biblioteca co seu id.
func (a *App) CrearCarpetaFormularios(nome string) (BibliotecaFormularios, error) {
	return crearCarpetaFormularios(nome)
}

// GardarDescricionFormulario cambia só a descrición dun formulario (vale en
// calquera modo, tamén en produción: non toca os campos).
func (a *App) GardarDescricionFormulario(id, descricion string) (Formulario, error) {
	return actualizarDescricionFormulario(id, descricion)
}

// GardarIconaFormulario cambia só a icona e a cor do formulario na columna
// esquerda (menú do botón dereito → «Cambiar icona»). Vale en calquera modo.
func (a *App) GardarIconaFormulario(id, icona, cor string) (Formulario, error) {
	return actualizarIconaFormulario(id, icona, cor)
}

// TiposCampoFormulario é a paleta de campos que pode arrastrar o deseñador.
func (a *App) TiposCampoFormulario() []string {
	return append([]string(nil), tiposCampoFormulario...)
}
