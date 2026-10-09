package main

import (
	"testing"

	"edicion/web"
)

// TestFrontendComunSincronizado garda contra a deriva do frontend compartido:
// os ficheiros de frontend/src/comun/ son copias xeradas de
// edicion/web/estatico/ (fonte única de verdade), así que este test falla se
// alguén edita unha copia a man, se engade código de motor dentro de comun/ ou
// se esquece de sincronizar tras cambiar a fonte.
//
// A orde exacta para rexenerar a copia vai na mensaxe de erro e está en
// edicion/web/web.go (ComandoSincronizar).
func TestFrontendComunSincronizado(t *testing.T) {
	if err := web.Verificar("frontend/src/comun"); err != nil {
		t.Fatalf("frontend común sen sincronizar: %v", err)
	}
}
