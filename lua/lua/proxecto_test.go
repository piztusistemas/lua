package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// proxectosTemporal aísla biblioteca de formularios, proxectos e configuración.
func proxectosTemporal(t *testing.T) string {
	t.Helper()
	arrel := t.TempDir()
	oFormularios, oBib, oProx := dirFormularios, dirBibliotecaFormularios, dirProxectos
	lib := filepath.Join(arrel, "formularios")
	dirBibliotecaFormularios = func() (string, error) { return lib, nil }
	dirFormularios = func() (string, error) {
		if d := dirProxectoActivo(); d != "" {
			return filepath.Join(d, "formularios"), nil
		}
		return lib, nil
	}
	dirProxectos = func() (string, error) { return filepath.Join(arrel, "proxectos"), nil }
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(arrel, "cfg"))
	t.Setenv("HOME", arrel)
	t.Cleanup(func() {
		dirFormularios, dirBibliotecaFormularios, dirProxectos = oFormularios, oBib, oProx
		_, _ = activarProxecto("")
	})
	return arrel
}

func TestCrearProxectoCopiaFormularios(t *testing.T) {
	proxectosTemporal(t)
	f := formularioDeProba(t, "Alfa")
	p, err := crearProxecto("  Titoría   1º ESO ", "Orientación", []string{"form:" + f.ID, "actas", "form:" + f.ID})
	if err != nil {
		t.Fatal(err)
	}
	if p.Nome != "Titoría 1º ESO" || !reflect.DeepEqual(p.Ferramentas, []string{"form:" + f.ID, "actas"}) {
		t.Errorf("proxecto: %+v", p)
	}
	dir, _ := dirProxecto(p.ID)
	if _, err := os.Stat(filepath.Join(dir, "formularios", f.ID+".json")); err != nil {
		t.Error("o formulario ten que copiarse ao proxecto")
	}
	// Non se pode crear sen nome, con ferramentas inventadas nin con formularios que non existen.
	for _, c := range []struct {
		n string
		f []string
	}{{"", nil}, {"X", []string{"inventada"}}, {"X", []string{"form:nonexiste"}}} {
		if _, err := crearProxecto(c.n, "", c.f); err == nil {
			t.Errorf("debería fallar: %+v", c)
		}
	}
	if l, _ := listarProxectos(); len(l) != 1 {
		t.Errorf("un crear fallido non deixa proxecto a medias: %+v", l)
	}
	// Mesmo nome: id distinto.
	p2, err := crearProxecto("Titoría 1º ESO", "", nil)
	if err != nil || p2.ID == p.ID {
		t.Errorf("ids: %q %q %v", p.ID, p2.ID, err)
	}
}

// Os datos e os formularios son do proxecto: mudar a biblioteca non os toca, e
// fóra do proxecto non se ven.
func TestProxectoAillaDatosEFormularios(t *testing.T) {
	proxectosTemporal(t)
	f := formularioDeProba(t, "Alfa")
	p, _ := crearProxecto("P", "", []string{"form:" + f.ID, "cronograma"})

	// Datos soltos (sen proxecto).
	soltos, _ := borradorFerramentaPath("form:" + f.ID)
	if err := os.MkdirAll(filepath.Dir(soltos), 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(soltos, []byte(`{"soltos":1}`), 0o644)

	if _, err := activarProxecto(p.ID); err != nil {
		t.Fatal(err)
	}
	dentro, _ := borradorFerramentaPath("form:" + f.ID)
	dir, _ := dirProxecto(p.ID)
	if !strings.HasPrefix(dentro, dir) || dentro == soltos {
		t.Errorf("os datos do proxecto van ao proxecto: %s", dentro)
	}
	a := &App{}
	if d, _ := a.LerBorradorInforme("form:" + f.ID); d != "" {
		t.Errorf("o proxecto non herda os datos soltos: %q", d)
	}
	if err := a.GardarBorradorInforme("form:"+f.ID, `{"do":"proxecto"}`); err != nil {
		t.Fatal(err)
	}
	if d, _ := os.ReadFile(soltos); string(d) != `{"soltos":1}` {
		t.Error("gardar dentro do proxecto non pode tocar os datos soltos")
	}
	// Dentro, a lista de formularios é a do proxecto (copia).
	if l, _ := listarFormularios(); len(l) != 1 || l[0].ID != f.ID {
		t.Errorf("formularios do proxecto: %+v", l)
	}
	// A biblioteca pode cambiar ou perder o formulario sen afectar ao proxecto.
	lib, _ := dirBibliotecaFormularios()
	os.Remove(filepath.Join(lib, f.ID+".json"))
	if l, _ := listarFormularios(); len(l) != 1 {
		t.Error("o proxecto conserva a súa copia")
	}
	// A columna do proxecto: as súas ferramentas á vista; as integradas que non usa, agochadas.
	b, _ := lerBiblioteca()
	if !reflect.DeepEqual(b.Fixos, []string{"form:" + f.ID, "cronograma"}) || !reflect.DeepEqual(b.Agochados, []string{"actas"}) {
		t.Errorf("columna do proxecto: %+v", b)
	}
	// Ao saír volven os datos soltos e a biblioteca.
	activarProxecto("")
	if d, _ := a.LerBorradorInforme("form:" + f.ID); d != `{"soltos":1}` {
		t.Errorf("fóra do proxecto: %q", d)
	}
	if l, _ := listarFormularios(); len(l) != 0 {
		t.Errorf("a biblioteca xa non ten o formulario: %+v", l)
	}
	// Reabrir: os datos do proxecto seguen aí.
	activarProxecto(p.ID)
	if d, _ := a.LerBorradorInforme("form:" + f.ID); d != `{"do":"proxecto"}` {
		t.Errorf("os datos do proxecto non se perderon: %q", d)
	}
}

func TestEngadirQuitarEEliminarProxecto(t *testing.T) {
	proxectosTemporal(t)
	f := formularioDeProba(t, "Alfa")
	g := formularioDeProba(t, "Beta")
	p, _ := crearProxecto("P", "", []string{"form:" + f.ID})
	p, err := engadirAoProxecto(p.ID, "form:"+g.ID)
	if err != nil || len(p.Ferramentas) != 2 {
		t.Fatalf("engadir: %+v %v", p, err)
	}
	activarProxecto(p.ID)
	a := &App{}
	a.GardarBorradorInforme("form:"+g.ID, `{"x":1}`)
	dir, _ := dirProxecto(p.ID)
	if p, err = quitarDoProxecto(p.ID, "form:"+g.ID); err != nil || len(p.Ferramentas) != 1 {
		t.Fatalf("quitar: %+v %v", p, err)
	}
	for _, r := range []string{"formularios/" + g.ID + ".json", "datos/formulario-" + g.ID + "-borrador.json"} {
		if _, err := os.Stat(filepath.Join(dir, r)); err == nil {
			t.Errorf("%s debería borrarse", r)
		}
	}
	if err := eliminarProxecto(p.ID); err == nil {
		t.Error("un proxecto aberto non se borra")
	}
	activarProxecto("")
	if err := eliminarProxecto(p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); err == nil {
		t.Error("o cartafol do proxecto debe desaparecer")
	}
}

// Dentro dun proxecto con bóveda, o asistente usa a bóveda do proxecto (pola API).
func TestBovedaDoProxectoNoAsistente(t *testing.T) {
	proxectosTemporal(t)
	p, _ := crearProxecto("P", "", nil)
	a := &App{}
	t.Cleanup(func() {
		if a.api != nil {
			a.api.Parar()
		}
	})
	if id, ruta := a.bovedaDe("cronograma"); id != "cronograma" || ruta != "" {
		t.Errorf("sen proxecto nin bóveda: %q %q", id, ruta)
	}
	raiz, _ := rutaBovedaProxecto(p.ID)
	for _, d := range []string{"raw", "wiki"} {
		os.MkdirAll(filepath.Join(raiz, d), 0o755)
	}
	os.WriteFile(filepath.Join(raiz, "raw", "a.md"), []byte("# A\nfonte"), 0o644)
	os.WriteFile(filepath.Join(raiz, "buxo.md"), []byte("# Esquema buxo"), 0o644)

	if l, _ := listarProxectos(); !l[0].Boveda || l[0].Fontes != 1 {
		t.Errorf("info: %+v", l)
	}
	activarProxecto(p.ID)
	id, ruta := a.bovedaDe("formulario")
	if id != "proxecto:"+p.ID || ruta == "" {
		t.Fatalf("bóveda do proxecto: %q %q", id, ruta)
	}
	if prompt := a.prepararBovedaTurno("", "cronograma"); !strings.Contains(prompt, "Esquema buxo") {
		t.Errorf("o esquema buxo.md debe ir ao prompt:\n%s", prompt)
	}
	if a.bovedaTurnoID != "proxecto:"+p.ID {
		t.Errorf("id do turno: %q", a.bovedaTurnoID)
	}
	res, err := execBoveda("boveda_ler")(a, map[string]any{"ruta": "raw/a.md"})
	if err != nil || !strings.Contains(res.(map[string]any)["texto"].(string), "fonte") {
		t.Fatalf("ler pola API: %v %v", res, err)
	}
	if _, err := execBoveda("boveda_escribir")(a, map[string]any{"ruta": "wiki/p.md", "contido": "x"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(raiz, "wiki", "p.md")); err != nil {
		t.Error("o asistente debe poder escribir na wiki do proxecto")
	}
	// Con proxecto, as ferramentas boveda_* aparecen aínda que non haxa bóveda conectada á man.
	n := 0
	for _, f := range a.ferramentasDoTurno("", "cronograma") {
		if ferramentasBoveda[f.Nome] {
			n++
		}
	}
	if n != len(ferramentasBoveda) {
		t.Errorf("ferramentas de bóveda: %d", n)
	}
}

// A bóveda do proxecto conéctase soa e pódese desconectar (e volver conectar).
func TestBovedaDoProxectoConectaSoaEPodeDesconectarse(t *testing.T) {
	proxectosTemporal(t)
	p, _ := crearProxecto("P", "", nil)
	raiz, _ := rutaBovedaProxecto(p.ID)
	for _, d := range []string{"raw", "wiki"} {
		os.MkdirAll(filepath.Join(raiz, d), 0o755)
	}
	a := &App{}
	activarProxecto(p.ID)
	if e := a.BovedaEstadoDe("formulario"); !e.Conectada || !e.Proxecto || e.ProxectoDesconectada {
		t.Fatalf("debe estar conectada soa: %+v", e)
	}
	if err := a.BovedaProxectoConectar(false); err != nil {
		t.Fatal(err)
	}
	if e := a.BovedaEstadoDe("formulario"); e.Conectada || !e.ProxectoDesconectada {
		t.Errorf("desconectada: %+v", e)
	}
	if prompt := a.prepararBovedaTurno("", "cronograma"); prompt != "" {
		t.Error("desconectada, o asistente non ve a bóveda")
	}
	if err := a.BovedaProxectoConectar(true); err != nil {
		t.Fatal(err)
	}
	if e := a.BovedaEstadoDe("formulario"); !e.Conectada || !e.Proxecto {
		t.Errorf("reconectada: %+v", e)
	}
	activarProxecto("")
	if err := a.BovedaProxectoConectar(true); err == nil {
		t.Error("sen proxecto aberto debe fallar")
	}
}

func TestCarpetasDeProxectos(t *testing.T) {
	proxectosTemporal(t)
	p1, _ := crearProxecto("Un", "", nil)
	p2, _ := crearProxecto("Dous", "", nil)
	c, err := crearCarpetaProxectos("  Inglés  2º ")
	if err != nil || len(c) != 1 || c[0].Nome != "Inglés 2º" || !idFormularioRe.MatchString(c[0].ID) {
		t.Fatalf("crear: %+v %v", c, err)
	}
	if _, err := crearCarpetaProxectos("inglés 2º"); err == nil {
		t.Error("nome repetido (sen mirar maiúsculas) debe fallar")
	}
	if _, err := crearCarpetaProxectos("  "); err == nil {
		t.Error("sen nome debe fallar")
	}
	id := c[0].ID
	if _, err := moverProxecto(p1.ID, "inexistente"); err == nil {
		t.Error("mover a un cartafol que non existe debe fallar")
	}
	moverProxecto(p1.ID, id)
	moverProxecto(p2.ID, id)
	if l, _ := listarProxectos(); l[0].Carpeta != id || l[1].Carpeta != id {
		t.Errorf("os proxectos deben levar o cartafol: %+v", l)
	}
	if c, err = renomearCarpetaProxectos(id, "Inglés"); err != nil || c[0].Nome != "Inglés" {
		t.Errorf("renomear: %+v %v", c, err)
	}
	// Borrar o cartafol non borra proxectos: pasan a sen cartafol.
	if c, err = eliminarCarpetaProxectos(id); err != nil || len(c) != 0 {
		t.Fatalf("eliminar: %+v %v", c, err)
	}
	l, _ := listarProxectos()
	if len(l) != 2 || l[0].Carpeta != "" || l[1].Carpeta != "" {
		t.Errorf("os proxectos quedan sen cartafol: %+v", l)
	}
	if _, err := eliminarCarpetaProxectos(id); err == nil {
		t.Error("borrar un cartafol que non existe debe fallar")
	}
	// carpetas.json non se confunde cun proxecto.
	if len(l) != 2 {
		t.Error("carpetas.json non é un proxecto")
	}
}
