package main

// Bindings de Wails de TRASLADAR (spec trasladar). A lóxica está en
// trasladar*.go; aquí só van os diálogos e o encaixe coa aplicación.

import (
	"edicion/idioma"
	"os"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// eventoFicheirosSoltos chega ao frontend cando se solta algo na xanela.
const eventoFicheirosSoltos = "lua:trasladar-soltados"

// ArboreTraslado é o que se pode escoller en «Sacar».
type ArboreTraslado struct {
	Proxectos   []ItemPaquete `json:"proxectos"`
	Formularios []ItemPaquete `json:"formularios"`
	Plantillas  []ItemPaquete `json:"plantillas"`
}

// ResultadoSacar di onde se gardou ("" = cancelado).
type ResultadoSacar struct {
	Ruta string `json:"ruta"`
}

// asegurarInstalacion devolve (e crea a primeira vez) o id da instalación.
func (a *App) asegurarInstalacion() string {
	if a.settings.Instalacion == "" {
		a.settings.Instalacion = novaInstalacion()
		_ = saveSettingsToDisk(a.settings)
	}
	return a.settings.Instalacion
}

// ArboreTraslado lista proxectos, formularios da biblioteca e plantillas.
func (a *App) ArboreTraslado() (ArboreTraslado, error) {
	out := ArboreTraslado{Proxectos: []ItemPaquete{}, Formularios: []ItemPaquete{}, Plantillas: []ItemPaquete{}}
	ps, err := listarProxectos()
	if err != nil {
		return out, err
	}
	for _, p := range ps {
		out.Proxectos = append(out.Proxectos, ItemPaquete{ID: p.ID, Nome: p.Nome})
	}
	fs, err := formulariosBiblioteca()
	if err != nil {
		return out, err
	}
	for _, f := range fs {
		out.Formularios = append(out.Formularios, ItemPaquete{ID: f.ID, Nome: f.Nome})
	}
	alm, err := loadAlmacenPlantillas()
	if err != nil {
		return out, err
	}
	for _, p := range alm.Plantillas {
		out.Plantillas = append(out.Plantillas, ItemPaquete{ID: p.ID, Nome: p.Nome})
	}
	return out, nil
}

// SacarTenDatosProducion: o que se vai sacar leva datos que poden ser reais.
func (a *App) SacarTenDatosProducion(sel SeleccionSacar) bool { return tenDatosProducion(sel) }

// SacarTraslado pide onde gardar e escribe o paquete .lua.
func (a *App) SacarTraslado(sel SeleccionSacar) (ResultadoSacar, error) {
	a.asegurarInstalacion()
	base := "lua-" + time.Now().Format("2006-01-02")
	switch {
	case sel.Tipo == "todo":
		base = idioma.T("go.trasladar.nome_copia", "copia-de-lua") + "-" + time.Now().Format("2006-01-02")
	case len(sel.Proxectos) == 1 && len(sel.Formularios)+len(sel.Plantillas) == 0:
		if p, err := lerProxecto(sel.Proxectos[0]); err == nil {
			base = nomeFicheiroPlantilla(p.Nome)
		}
	}
	ruta, err := saveFileDialogCompat(idioma.T("go.trasladar.titulo_sacar", "Sacar de Lúa"), "", base+extensionPaquete, "Lúa (*"+extensionPaquete+")", "*"+extensionPaquete)
	if err != nil || ruta == "" {
		return ResultadoSacar{}, err
	}
	if !strings.HasSuffix(strings.ToLower(ruta), extensionPaquete) {
		ruta += extensionPaquete
	}
	if _, err := escribirPaqueteEn(ruta, sel, a.settings); err != nil {
		return ResultadoSacar{}, err
	}
	return ResultadoSacar{Ruta: ruta}, nil
}

// SacarDocumentoJSON garda o documento aberto coma .json editable.
func (a *App) SacarDocumentoJSON(ferramenta, contido, nome string) (ResultadoSacar, error) {
	d, err := construirDocumentoTraslado(ferramenta, contido, nome, a.asegurarInstalacion())
	if err != nil {
		return ResultadoSacar{}, err
	}
	base := nomeFicheiroPlantilla(d.Nome)
	if base == "" {
		base = "documento"
	}
	ruta, err := saveFileDialogCompat(idioma.T("go.trasladar.titulo_sacar", "Sacar de Lúa"), "", base+".json", "JSON (*.json)", "*.json")
	if err != nil || ruta == "" {
		return ResultadoSacar{}, err
	}
	if !strings.HasSuffix(strings.ToLower(ruta), ".json") {
		ruta += ".json"
	}
	if err := escribirXSONAtomico(ruta, d); err != nil {
		return ResultadoSacar{}, err
	}
	return ResultadoSacar{Ruta: ruta}, nil
}

// EscollerFicheiroMeter abre o selector de ficheiros ("" = cancelado).
func (a *App) EscollerFicheiroMeter() (string, error) {
	return application.Get().Dialog.OpenFile().
		SetTitle(idioma.T("go.trasladar.titulo_meter", "Meter en Lúa")).
		AddFilter("Lúa (*.lua, *.json, *.pdf, *.md)", "*.lua;*.json;*.pdf;*.md;*.markdown").
		PromptForSingleSelection()
}

// EscollerCartafolMeter abre o selector de cartafoles ("" = cancelado).
func (a *App) EscollerCartafolMeter() (string, error) {
	return application.Get().Dialog.OpenFile().
		SetTitle(idioma.T("go.trasladar.titulo_meter", "Meter en Lúa")).
		CanChooseDirectories(true).CanChooseFiles(false).CanCreateDirectories(false).
		PromptForSingleSelection()
}

// AnalizarMeter di que leva un ficheiro e onde iría, sen cambiar nada.
func (a *App) AnalizarMeter(ruta string) (ResumoMeter, error) { return a.clasificarMeter(ruta) }

// AplicarMeter mete o ficheiro coas escollas do resumo.
func (a *App) AplicarMeter(ruta string, op OpcionsMeter) (ResultadoMeter, error) {
	a.asegurarInstalacion()
	return a.aplicarMeter(ruta, op)
}

// DesfacerTraslado volve ao estado de antes dunha importación.
func (a *App) DesfacerTraslado(version string) error {
	if err := restaurarVersion(version); err != nil {
		return err
	}
	if id := idProxectoActivo(); id != "" && proxectoAbertoValido() == "" {
		_, _ = activarProxecto("")
	}
	return nil
}

// MigracionTraballoSolto devolve, unha soa vez, o que se moveu do traballo
// solto a un proxecto ao arrincar (Proxecto "" = nada). Se aínda non se fixo
// (por exemplo, fallou ao arrincar), inténtao agora.
func (a *App) MigracionTraballoSolto() (ResultadoMigracion, error) {
	a.migracionMu.Lock()
	defer a.migracionMu.Unlock()
	if a.migracion != nil {
		m := *a.migracion
		a.migracion = &ResultadoMigracion{Ferramentas: []string{}}
		return m, nil
	}
	a.migracion = &ResultadoMigracion{Ferramentas: []string{}}
	return migrarTraballoSolto()
}

// rexistrarSoltarFicheiros avisa ao frontend cando se soltan ficheiros.
func rexistrarSoltarFicheiros(app *application.App, w *application.WebviewWindow) {
	w.OnWindowEvent(events.Common.WindowFilesDropped, func(e *application.WindowEvent) {
		ficheiros := e.Context().DroppedFiles()
		existentes := []string{}
		for _, f := range ficheiros {
			if _, err := os.Stat(f); err == nil {
				existentes = append(existentes, f)
			}
		}
		if len(existentes) > 0 {
			app.Event.Emit(eventoFicheirosSoltos, existentes)
		}
	})
}
