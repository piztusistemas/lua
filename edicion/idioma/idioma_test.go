package idioma

import (
	"errors"
	"fmt"
	"io/fs"
	"sync"
	"testing"
)

// dicionariosProba deixa o tradutor cun estado coñecido e restáurao ao
// rematar: o estado é global ao proceso, e os tests non poden pisarse.
func dicionariosProba(t *testing.T) {
	t.Helper()
	Restablecer()
	Rexistrar("es", map[string]string{
		"go.proba.ficheiro": "no se encontró %s",
		"go.proba.envolto":  "no se pudo leer: %w",
		"go.proba.dous":     "%d de %s",
	})
	Rexistrar("en", map[string]string{
		"go.proba.ficheiro": "%s was not found",
	})
	t.Cleanup(Restablecer)
}

func TestGalegoEIdenticoAFmt(t *testing.T) {
	dicionariosProba(t)
	SetIdioma("gl")
	if got, want := T("go.proba.dous", "%d de %s", 3, "dez"), fmt.Sprintf("%d de %s", 3, "dez"); got != want {
		t.Errorf("T en gl = %q, quería %q", got, want)
	}
	base := errors.New("permiso denegado")
	err := Errorf("go.proba.envolto", "non se puido ler: %w", base)
	if got, want := err.Error(), fmt.Errorf("non se puido ler: %w", base).Error(); got != want {
		t.Errorf("Errorf en gl = %q, quería %q", got, want)
	}
}

func TestTraduceCosArgumentos(t *testing.T) {
	dicionariosProba(t)
	SetIdioma("es")
	if got := T("go.proba.dous", "%d de %s", 3, "dez"); got != "3 de dez" {
		t.Errorf("es = %q", got)
	}
	SetIdioma("en")
	if got := T("go.proba.ficheiro", "non se atopou %s", "xelatex"); got != "xelatex was not found" {
		t.Errorf("en = %q", got)
	}
}

func TestReservaAoGalego(t *testing.T) {
	dicionariosProba(t)
	SetIdioma("en")
	// A clave existe en es pero non en en: vai o galego.
	if got := T("go.proba.dous", "%d de %s", 1, "dous"); got != "1 de dous" {
		t.Errorf("clave sen tradución = %q", got)
	}
	if got := T("go.proba.inexistente", "texto %s", "galego"); got != "texto galego" {
		t.Errorf("clave inexistente = %q", got)
	}
	for _, codigo := range []string{"", "xx", "fr"} {
		SetIdioma(codigo)
		if Actual() != "gl" {
			t.Errorf("SetIdioma(%q) deixou %q, quería gl", codigo, Actual())
		}
	}
}

func TestErroEnvoltoSegueSendoOMesmo(t *testing.T) {
	dicionariosProba(t)
	SetIdioma("es")
	err := Errorf("go.proba.envolto", "non se puido ler: %w", fs.ErrNotExist)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Error("errors.Is non atravesa o erro traducido")
	}
	if errors.Unwrap(err) != fs.ErrNotExist {
		t.Error("Unwrap non devolve o erro envolto")
	}
	if got := err.Error(); got != "no se pudo leer: "+fs.ErrNotExist.Error() {
		t.Errorf("es = %q", got)
	}
}

func TestTraduceAoLerONonAoCrear(t *testing.T) {
	dicionariosProba(t)
	SetIdioma("gl")
	err := Errorf("go.proba.ficheiro", "non se atopou %s", "pandoc")
	SetIdioma("en")
	if got := err.Error(); got != "pandoc was not found" {
		t.Errorf("tras cambiar a en = %q", got)
	}
}

func TestGalegoDunErro(t *testing.T) {
	dicionariosProba(t)
	SetIdioma("en")
	err := fmt.Errorf("contexto: %w", Errorf("go.proba.ficheiro", "non se atopou %s", "pandoc"))
	if got := Galego(Errorf("go.proba.ficheiro", "non se atopou %s", "pandoc")); got != "non se atopou pandoc" {
		t.Errorf("Galego = %q", got)
	}
	if got := Galego(errors.New("simple")); got != "simple" {
		t.Errorf("Galego dun erro normal = %q", got)
	}
	// Un erro normal que envolve un traducible: o texto é o de Error().
	if got := Galego(err); got != err.Error() {
		t.Errorf("Galego dun envoltorio = %q", got)
	}
	if Galego(nil) != "" {
		t.Error("Galego(nil) debería ser baleiro")
	}
}

func TestConcorrencia(t *testing.T) {
	dicionariosProba(t)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() { defer wg.Done(); SetIdioma([]string{"gl", "es", "en"}[i%3]) }()
		go func() { defer wg.Done(); _ = T("go.proba.ficheiro", "non se atopou %s", "x") }()
	}
	wg.Wait()
}

func TestGalegoDeErrosAniñados(t *testing.T) {
	dicionariosProba(t)
	SetIdioma("es")
	dentro := Errorf("go.proba.ficheiro", "non se atopou %s", "pandoc")
	fora := Errorf("go.proba.envolto", "non se puido ler: %w", dentro)
	if got := fora.Error(); got != "no se pudo leer: no se encontró pandoc" {
		t.Errorf("es = %q", got)
	}
	if got := Galego(fora); got != "non se puido ler: non se atopou pandoc" {
		t.Errorf("Galego = %q", got)
	}
	if !errors.Is(fora, dentro) {
		t.Error("errors.Is non chega ao erro de dentro")
	}
}
