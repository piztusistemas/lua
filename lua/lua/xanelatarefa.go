package main

import (
	"regexp"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// xanelatarefa.go: a pantalla "Editar tarefa" en xanela propia. Ao premer
// nunha tarefa a xanela ábrese maximizada, ocupando a pantalla enteira, e
// coma calquera xanela do sistema pódese desmaximizar e levar a un segundo
// monitor. O curso en edición vive na xanela principal: esta só recibe a
// tarefa (JSON, cos datos que precisa, ver cronograma-editor.js/datosTarefa)
// e, cada vez que o profesorado preme Aplicar ou Gardar, devolve o resultado
// coma evento (lua:tarefa-aplicada) que a xanela principal mete no curso.
//
// Cada tarefa ten a súa propia xanela (nome "tarefa-<id>"): se xa está aberta,
// só se trae ao primeiro plano.

const prefixoXanelaTarefa = "tarefa-"

var idTarefaValido = regexp.MustCompile(`[^A-Za-z0-9_-]`)

// nomeXanelaTarefa devolve o nome da xanela dunha tarefa, e o id xa limpo de
// caracteres que non poden ir nun nome de xanela nin nunha URL.
func nomeXanelaTarefa(id string) (nome, idLimpo string) {
	idLimpo = idTarefaValido.ReplaceAllString(id, "_")
	if idLimpo == "" {
		idLimpo = "nova"
	}
	return prefixoXanelaTarefa + idLimpo, idLimpo
}

// tarefaAplicadaEvento é o que viaxa da xanela de tarefa á principal.
type tarefaAplicadaEvento struct {
	ID string `json:"id"`
	// Resultado é o JSON {tarefa, claveOrixe, mesDestino} tal como o mandou a
	// pantalla; a xanela principal é a única que o interpreta.
	Resultado string `json:"resultado"`
}

type tarefaPedirPecharEvento struct {
	ID string `json:"id"`
}

// estadoXanelasTarefa é o que Go lembra das xanelas de tarefa abertas.
type estadoXanelasTarefa struct {
	mu       sync.Mutex
	datos    map[string]string // id -> JSON inicial
	sucias   map[string]bool   // id -> hai cambios sen aplicar
	pechando map[string]bool   // id -> pechado confirmado, deixar pasar o peche
}

func (e *estadoXanelasTarefa) init() {
	if e.datos == nil {
		e.datos = map[string]string{}
		e.sucias = map[string]bool{}
		e.pechando = map[string]bool{}
	}
}

// AbrirXanelaTarefa abre (ou trae ao primeiro plano) a xanela de edición da
// tarefa `id`, co JSON `datos` que a pantalla necesita para pintarse. Devolve
// false, sen erro, cando non se pode abrir unha xanela nativa (proba en
// navegador, por exemplo): o frontend abre daquela a pantalla completa dentro
// da xanela principal.
func (a *App) AbrirXanelaTarefa(id, datos string) (bool, error) {
	app := application.Get()
	// En modo servidor (a app nun navegador, para probas) non hai xanelas do
	// sistema: o frontend usa a pantalla completa dentro da propia páxina.
	if app == nil || application.System.IsServer() {
		return false, nil
	}
	nome, idLimpo := nomeXanelaTarefa(id)
	if w, ok := app.Window.GetByName(nome); ok {
		w.Show()
		w.Focus()
		return true, nil
	}

	a.tarefas.mu.Lock()
	a.tarefas.init()
	a.tarefas.datos[idLimpo] = datos
	a.tarefas.sucias[idLimpo] = false
	a.tarefas.pechando[idLimpo] = false
	a.tarefas.mu.Unlock()

	w := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             nome,
		Title:            "Piztu Lúa — Editar tarefa",
		URL:              "/?tarefa=" + idLimpo,
		Width:            1280,
		Height:           860,
		StartState:       application.WindowStateMaximised,
		BackgroundColour: application.NewRGB(23, 24, 28),
		// Zoom coma nas outras xanelas (Ctrl+/Ctrl-/Ctrl+0).
		ZoomControlEnabled:  true,
		MinimiseButtonState: application.ButtonEnabled,
		MaximiseButtonState: application.ButtonEnabled,
	})

	// Se se pecha co botón do xestor de xanelas e hai cambios sen aplicar, non
	// se pecha: pídeselle á páxina que pregunte (mesmo aviso ca a "X" vermella).
	w.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		a.tarefas.mu.Lock()
		sucia := a.tarefas.sucias[idLimpo] && !a.tarefas.pechando[idLimpo]
		a.tarefas.mu.Unlock()
		if sucia {
			e.Cancel()
			application.Get().Event.Emit("lua:tarefa-pedir-pechar", tarefaPedirPecharEvento{ID: idLimpo})
		}
	})
	w.OnWindowEvent(events.Common.WindowClosing, func(_ *application.WindowEvent) {
		a.tarefas.mu.Lock()
		delete(a.tarefas.datos, idLimpo)
		delete(a.tarefas.sucias, idLimpo)
		delete(a.tarefas.pechando, idLimpo)
		a.tarefas.mu.Unlock()
	})
	return true, nil
}

// GetDatosXanelaTarefa é o que pide a xanela de tarefa ao montarse.
func (a *App) GetDatosXanelaTarefa(id string) string {
	_, idLimpo := nomeXanelaTarefa(id)
	a.tarefas.mu.Lock()
	defer a.tarefas.mu.Unlock()
	return a.tarefas.datos[idLimpo]
}

// AplicarTarefaXanela leva o resultado da xanela de tarefa á principal.
func (a *App) AplicarTarefaXanela(id, resultado string) error {
	_, idLimpo := nomeXanelaTarefa(id)
	if app := application.Get(); app != nil {
		app.Event.Emit("lua:tarefa-aplicada", tarefaAplicadaEvento{ID: idLimpo, Resultado: resultado})
	}
	return nil
}

// MarcarXanelaTarefaSucia di se a xanela ten cambios sen aplicar (para non
// pechala sen preguntar co botón do xestor de xanelas).
func (a *App) MarcarXanelaTarefaSucia(id string, sucia bool) {
	_, idLimpo := nomeXanelaTarefa(id)
	a.tarefas.mu.Lock()
	a.tarefas.init()
	a.tarefas.sucias[idLimpo] = sucia
	a.tarefas.mu.Unlock()
}

// PecharXanelaTarefa pecha a xanela sen máis preguntas (a páxina xa
// confirmou o descarte, ou xa gardou).
func (a *App) PecharXanelaTarefa(id string) {
	nome, idLimpo := nomeXanelaTarefa(id)
	a.tarefas.mu.Lock()
	a.tarefas.init()
	a.tarefas.pechando[idLimpo] = true
	a.tarefas.mu.Unlock()
	if app := application.Get(); app != nil {
		if w, ok := app.Window.GetByName(nome); ok {
			w.Close()
		}
	}
}

func init() {
	application.RegisterEvent[tarefaAplicadaEvento]("lua:tarefa-aplicada")
	application.RegisterEvent[tarefaPedirPecharEvento]("lua:tarefa-pedir-pechar")
}
