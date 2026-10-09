package main

import (
	"errors"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// xanelaformulario.go: o DESEÑADOR DE FORMULARIOS en xanela propia. Ábrese
// maximizado, a pantalla completa, e coma calquera xanela do sistema pódese
// levar a outro monitor: o profesorado pode ter o deseñador nunha pantalla e o
// formulario e o seu resultado noutra. Ao gardar (ou eliminar) avísase á
// xanela principal (lua:formulario-gardado / lua:formulario-eliminado) para que
// pinte a icona nova, refresque o formulario e o seu resultado.
//
// Segue o mesmo esquema ca xanelaplantilla.go. Cada formulario ten a súa xanela
// (nome «formulario-<id>»; «formulario-novo» para os que aínda non se
// gardaron): se xa está aberta, só se trae ao primeiro plano.

const prefixoXanelaFormulario = "formulario-"

// formularioResultadoEvento é o que viaxa da xanela do deseñador á principal:
// o resultado do deseñador en JSON ({accion:'gardado', esquema, novo} ou
// {accion:'eliminado', id}).
type formularioResultadoEvento struct {
	Resultado string `json:"resultado"`
}

type formularioPedirPecharEvento struct {
	ID string `json:"id"`
}

func nomeXanelaFormulario(id string) (nome, idLimpo string) {
	idLimpo = idTarefaValido.ReplaceAllString(id, "_")
	if idLimpo == "" {
		idLimpo = "novo"
	}
	return prefixoXanelaFormulario + idLimpo, idLimpo
}

// AbrirXanelaFormulario abre (ou trae ao primeiro plano) o deseñador do
// formulario `id` (baleiro para un novo) co JSON `datos` que necesita
// ({esquema, datos}). Devolve false, sen erro, cando non se pode abrir unha
// xanela nativa (proba en navegador): o frontend abre daquela o deseñador
// coma unha capa dentro da xanela principal.
func (a *App) AbrirXanelaFormulario(id, datos string) (bool, error) {
	// Un formulario en produción non se deseña: nin sequera se lle amosa a
	// xanela (o frontend tampouco ofrece o botón, isto é o seguro de Go).
	if id != "" {
		if f, err := lerFormulario(id); err == nil && f.enProduccion() {
			return false, errors.New("o formulario está en modo produción: pásao a deseño para cambialo")
		}
	}
	app := application.Get()
	if app == nil || application.System.IsServer() {
		return false, nil
	}
	nome, idLimpo := nomeXanelaFormulario(id)
	if w, ok := app.Window.GetByName(nome); ok {
		w.Show()
		w.Focus()
		return true, nil
	}

	a.formularios.mu.Lock()
	a.formularios.init()
	a.formularios.datos[idLimpo] = datos
	a.formularios.sucias[idLimpo] = false
	a.formularios.pechando[idLimpo] = false
	a.formularios.mu.Unlock()

	w := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:                nome,
		Title:               "Lua — Deseñar formulario",
		URL:                 "/?formulario=" + idLimpo,
		Width:               1500,
		Height:              900,
		StartState:          application.WindowStateMaximised,
		BackgroundColour:    application.NewRGB(23, 24, 28),
		ZoomControlEnabled:  true,
		MinimiseButtonState: application.ButtonEnabled,
		MaximiseButtonState: application.ButtonEnabled,
	})

	// Co botón do xestor de xanelas e cambios sen gardar, non se pecha: a
	// páxina pregunta (mesmo aviso ca a «X» do deseñador).
	w.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		a.formularios.mu.Lock()
		sucia := a.formularios.sucias[idLimpo] && !a.formularios.pechando[idLimpo]
		a.formularios.mu.Unlock()
		if sucia {
			e.Cancel()
			application.Get().Event.Emit("lua:formulario-pedir-pechar", formularioPedirPecharEvento{ID: idLimpo})
		}
	})
	w.OnWindowEvent(events.Common.WindowClosing, func(_ *application.WindowEvent) {
		a.formularios.mu.Lock()
		delete(a.formularios.datos, idLimpo)
		delete(a.formularios.sucias, idLimpo)
		delete(a.formularios.pechando, idLimpo)
		a.formularios.mu.Unlock()
	})
	return true, nil
}

// GetDatosXanelaFormulario é o que pide a xanela do deseñador ao montarse.
func (a *App) GetDatosXanelaFormulario(id string) string {
	_, idLimpo := nomeXanelaFormulario(id)
	a.formularios.mu.Lock()
	defer a.formularios.mu.Unlock()
	return a.formularios.datos[idLimpo]
}

// FormularioXanelaGardado avisa á xanela principal de que o formulario se
// gardou: `resultado` é o JSON {accion:'gardado', esquema, novo}.
func (a *App) FormularioXanelaGardado(resultado string) {
	if app := application.Get(); app != nil {
		app.Event.Emit("lua:formulario-gardado", formularioResultadoEvento{Resultado: resultado})
	}
}

// FormularioXanelaEliminado avisa de que o formulario se eliminou: `resultado` é {accion:'eliminado', id}.
func (a *App) FormularioXanelaEliminado(resultado string) {
	if app := application.Get(); app != nil {
		app.Event.Emit("lua:formulario-eliminado", formularioResultadoEvento{Resultado: resultado})
	}
}

// MarcarXanelaFormularioSucia di se o deseñador ten cambios sen gardar.
func (a *App) MarcarXanelaFormularioSucia(id string, sucia bool) {
	_, idLimpo := nomeXanelaFormulario(id)
	a.formularios.mu.Lock()
	a.formularios.init()
	a.formularios.sucias[idLimpo] = sucia
	a.formularios.mu.Unlock()
}

// PecharXanelaFormulario pecha a xanela sen máis preguntas.
func (a *App) PecharXanelaFormulario(id string) {
	nome, idLimpo := nomeXanelaFormulario(id)
	a.formularios.mu.Lock()
	a.formularios.init()
	a.formularios.pechando[idLimpo] = true
	a.formularios.mu.Unlock()
	if app := application.Get(); app != nil {
		if w, ok := app.Window.GetByName(nome); ok {
			w.Close()
		}
	}
}

// pecharDeseñadorFormulario pecha a xanela do deseñador do formulario `id` se
// está aberta. Se ten cambios sen gardar devolve un erro en vez de pechala.
func (a *App) pecharDeseñadorFormulario(id string) error {
	nome, idLimpo := nomeXanelaFormulario(id)
	app := application.Get()
	if app == nil {
		return nil
	}
	if _, ok := app.Window.GetByName(nome); !ok {
		return nil
	}
	a.formularios.mu.Lock()
	sucia := a.formularios.sucias[idLimpo]
	a.formularios.mu.Unlock()
	if sucia {
		return errors.New("o deseñador deste formulario está aberto con cambios sen gardar: gárdaos ou péchao antes de pasalo a produción")
	}
	a.PecharXanelaFormulario(id)
	return nil
}

func init() {
	application.RegisterEvent[formularioResultadoEvento]("lua:formulario-gardado")
	application.RegisterEvent[formularioResultadoEvento]("lua:formulario-eliminado")
	application.RegisterEvent[formularioPedirPecharEvento]("lua:formulario-pedir-pechar")
}
