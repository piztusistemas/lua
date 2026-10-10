package main

import (
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// xanelaplantilla.go: o editor de plantillas en xanela propia. Ábrese
// maximizado, en bloques e columnas, e coma calquera xanela do sistema pódese
// levar a outro monitor. O profesorado modifica a plantilla, preme Aplicar (que
// a garda SEN pechar) e ve o resultado na vista previa ou na xanela principal,
// sen ter que pechar, probar, abrir e volver modificar. Cada vez que se aplica
// avísase á xanela principal (lua:plantilla-aplicada) para que refresque a
// biblioteca e o informe se a plantilla é a activa.
//
// Segue o mesmo esquema ca xanelatarefa.go (aquí o estado é o mesmo tipo).
// Cada plantilla ten a súa xanela (nome "plantilla-<id>"; "plantilla-nova" para
// as que aínda non se gardaron): se xa está aberta, só se trae ao primeiro plano.

const prefixoXanelaPlantilla = "plantilla-"

// plantillaAplicadaEvento é o que viaxa da xanela do editor á principal.
type plantillaAplicadaEvento struct {
	// ID da plantilla tal como quedou gardada (unha de exemplo bifúrcase ao
	// editala: o id pode ser distinto do que se abriu).
	ID string `json:"id"`
}

type plantillaPedirPecharEvento struct {
	ID string `json:"id"`
}

func nomeXanelaPlantilla(id string) (nome, idLimpo string) {
	idLimpo = idTarefaValido.ReplaceAllString(id, "_")
	if idLimpo == "" {
		idLimpo = "nova"
	}
	return prefixoXanelaPlantilla + idLimpo, idLimpo
}

// AbrirXanelaPlantilla abre (ou trae ao primeiro plano) o editor da plantilla
// `id` (baleiro para unha nova) co JSON `datos` que a pantalla necesita. Devolve
// false, sen erro, cando non se pode abrir unha xanela nativa (proba en
// navegador): o frontend abre daquela o editor a pantalla completa dentro da
// xanela principal.
func (a *App) AbrirXanelaPlantilla(id, datos string) (bool, error) {
	app := application.Get()
	if app == nil || application.System.IsServer() {
		return false, nil
	}
	nome, idLimpo := nomeXanelaPlantilla(id)
	if w, ok := app.Window.GetByName(nome); ok {
		w.Show()
		w.Focus()
		return true, nil
	}

	a.plantillas.mu.Lock()
	a.plantillas.init()
	a.plantillas.datos[idLimpo] = datos
	a.plantillas.sucias[idLimpo] = false
	a.plantillas.pechando[idLimpo] = false
	a.plantillas.mu.Unlock()

	w := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:                nome,
		Title:               "Piztu Lúa — Editar plantilla",
		URL:                 "/?plantilla=" + idLimpo,
		Width:               1500,
		Height:              900,
		StartState:          application.WindowStateMaximised,
		BackgroundColour:    application.NewRGB(23, 24, 28),
		ZoomControlEnabled:  true,
		MinimiseButtonState: application.ButtonEnabled,
		MaximiseButtonState: application.ButtonEnabled,
	})

	// Co botón do xestor de xanelas e cambios sen aplicar, non se pecha: a
	// páxina pregunta (mesmo aviso ca a "X" vermella).
	w.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		a.plantillas.mu.Lock()
		sucia := a.plantillas.sucias[idLimpo] && !a.plantillas.pechando[idLimpo]
		a.plantillas.mu.Unlock()
		if sucia {
			e.Cancel()
			application.Get().Event.Emit("lua:plantilla-pedir-pechar", plantillaPedirPecharEvento{ID: idLimpo})
		}
	})
	w.OnWindowEvent(events.Common.WindowClosing, func(_ *application.WindowEvent) {
		a.plantillas.mu.Lock()
		delete(a.plantillas.datos, idLimpo)
		delete(a.plantillas.sucias, idLimpo)
		delete(a.plantillas.pechando, idLimpo)
		a.plantillas.mu.Unlock()
	})
	return true, nil
}

// GetDatosXanelaPlantilla é o que pide a xanela do editor ao montarse.
func (a *App) GetDatosXanelaPlantilla(id string) string {
	_, idLimpo := nomeXanelaPlantilla(id)
	a.plantillas.mu.Lock()
	defer a.plantillas.mu.Unlock()
	return a.plantillas.datos[idLimpo]
}

// PlantillaXanelaAplicada avisa á xanela principal de que a plantilla se
// gardou (Aplicar ou Gardar) para que refresque o que dependa dela.
func (a *App) PlantillaXanelaAplicada(idPlantilla string) {
	if app := application.Get(); app != nil {
		app.Event.Emit("lua:plantilla-aplicada", plantillaAplicadaEvento{ID: idPlantilla})
	}
}

// MarcarXanelaPlantillaSucia di se o editor ten cambios sen aplicar.
func (a *App) MarcarXanelaPlantillaSucia(id string, sucia bool) {
	_, idLimpo := nomeXanelaPlantilla(id)
	a.plantillas.mu.Lock()
	a.plantillas.init()
	a.plantillas.sucias[idLimpo] = sucia
	a.plantillas.mu.Unlock()
}

// PecharXanelaPlantilla pecha a xanela sen máis preguntas.
func (a *App) PecharXanelaPlantilla(id string) {
	nome, idLimpo := nomeXanelaPlantilla(id)
	a.plantillas.mu.Lock()
	a.plantillas.init()
	a.plantillas.pechando[idLimpo] = true
	a.plantillas.mu.Unlock()
	if app := application.Get(); app != nil {
		if w, ok := app.Window.GetByName(nome); ok {
			w.Close()
		}
	}
}

func init() {
	application.RegisterEvent[plantillaAplicadaEvento]("lua:plantilla-aplicada")
	application.RegisterEvent[plantillaPedirPecharEvento]("lua:plantilla-pedir-pechar")
}
