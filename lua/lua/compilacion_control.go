package main

// Control das compilacións de LaTeX: tempo máximo (constitución 5.4) e
// cancelación da vista previa (constitución 7.1).
//
// Cada compilación vai nun cartafol temporal propio. Quen quere poder
// cancelala (a vista previa do editor de plantillas) rexistra o seu cartafol
// cun contexto; executarLatexCrono lanza LaTeX con ese contexto e cun tempo
// máximo, así cancelar mata o proceso de verdade.

import (
	"context"
	"edicion/idioma"
	"errors"
	"sync"
	"time"
)

// tempoMaximoLatex é o tempo máximo dunha pasada de LaTeX. Un informe normal
// tarda segundos; isto só para un documento que entrou nun bucle. Variable
// para os tests.
var tempoMaximoLatex = 3 * time.Minute

// errCompilacionCancelada: o usuario cancelou (ou unha vista previa nova
// substituíu a anterior).
var errCompilacionCancelada = errors.New("compilación cancelada")

var (
	compilacionsMu sync.Mutex
	compilacions   = map[string]context.Context{}
	vistasPrevias  = map[string]context.CancelFunc{}
)

// rexistrarVistaPrevia deixa cancelable a compilación que se fai en `dir`.
// Devolve a función que hai que chamar ao rematar.
func rexistrarVistaPrevia(dir string) func() {
	ctx, cancel := context.WithCancel(context.Background())
	compilacionsMu.Lock()
	compilacions[dir] = ctx
	vistasPrevias[dir] = cancel
	compilacionsMu.Unlock()
	return func() {
		compilacionsMu.Lock()
		delete(compilacions, dir)
		delete(vistasPrevias, dir)
		compilacionsMu.Unlock()
		cancel()
	}
}

// contextoCompilacion devolve o contexto da compilación en `dir` (o rexistrado
// ou un novo) xa co tempo máximo.
func contextoCompilacion(dir string) (context.Context, context.CancelFunc) {
	compilacionsMu.Lock()
	pai, ok := compilacions[dir]
	compilacionsMu.Unlock()
	if !ok {
		pai = context.Background()
	}
	return context.WithTimeout(pai, tempoMaximoLatex)
}

// CancelarVistaPrevia para as vistas previas que se están compilando (o botón
// «Cancelar» da vista previa do editor de plantillas).
func (a *App) CancelarVistaPrevia() int {
	compilacionsMu.Lock()
	defer compilacionsMu.Unlock()
	n := 0
	for _, cancel := range vistasPrevias {
		cancel()
		n++
	}
	return n
}

// erroContexto traduce o motivo polo que parou unha compilación.
func erroContexto(ctx context.Context) error {
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return erroTempoLatex{idioma.Errorf("go.latex.tempo_maximo", "LaTeX tardou máis de %s e parouse: revisa o LaTeX da plantilla (pode ter un bucle)", tempoMaximoLatex)}
	case errors.Is(ctx.Err(), context.Canceled):
		return errCompilacionCancelada
	}
	return nil
}

// erroTempoLatex: LaTeX superou o tempo máximo.
type erroTempoLatex struct{ erro error }

func (e erroTempoLatex) Error() string { return e.erro.Error() }

// compilacionParada di se un erro é dunha compilación cancelada ou parada por tempo.
func compilacionParada(err error) bool {
	var t erroTempoLatex
	return errors.Is(err, errCompilacionCancelada) || errors.As(err, &t)
}
