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

func TestMoverProxectoAOutroCartafol(t *testing.T) {
	arrel := proxectosTemporal(t)
	destino := filepath.Join(arrel, "Clases")
	if err := os.MkdirAll(destino, 0o755); err != nil {
		t.Fatal(err)
	}
	p, _ := crearProxecto("Gramática: 1º/ESO", "", nil)
	orixe, _ := dirProxecto(p.ID)
	if _, err := moverProxecto(p.ID, destino); err != nil {
		t.Fatal(err)
	}
	dir, _ := dirProxecto(p.ID)
	if dir != filepath.Join(destino, "Gramática 1º ESO") {
		t.Fatalf("o proxecto vai nun subcartafol co seu nome: %s", dir)
	}
	if _, err := os.Stat(filepath.Join(dir, "proxecto.json")); err != nil {
		t.Errorf("os ficheiros deben estar no cartafol novo: %v", err)
	}
	if _, err := os.Stat(orixe); !os.IsNotExist(err) {
		t.Errorf("o cartafol vello debe desaparecer: %v", err)
	}
	l, _ := listarProxectos()
	if len(l) != 1 || l[0].Ubicacion != destino || l[0].Ruta != dir {
		t.Errorf("a lista debe dicir onde está: %+v", l)
	}
	// Outro co mesmo nome no mesmo sitio non pisa o primeiro.
	p2, _ := crearProxecto("Gramática: 1º/ESO", "", nil)
	if p2.ID == p.ID {
		t.Fatal("un proxecto doutro cartafol non debe perder o seu id")
	}
	moverProxecto(p2.ID, destino)
	if d2, _ := dirProxecto(p2.ID); d2 != filepath.Join(destino, "Gramática 1º ESO (2)") {
		t.Errorf("nome repetido: %s", d2)
	}
	// Volver ao cartafol de Lúa.
	if _, err := moverProxecto(p2.ID, ""); err != nil {
		t.Fatal(err)
	}
	if d2, _ := dirProxecto(p2.ID); d2 != filepath.Join(arrel, "proxectos", p2.ID) {
		t.Errorf("de volta: %s", d2)
	}
	// Eliminar borra o subcartafol, pero non o cartafol escollido.
	if err := eliminarProxecto(p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Error("eliminar debe borrar o proxecto")
	}
	if _, err := os.Stat(destino); err != nil {
		t.Error("eliminar non debe borrar o cartafol escollido")
	}
	if l, _ := listarProxectos(); len(l) != 1 || l[0].ID != p2.ID || l[0].Ubicacion != "" {
		t.Errorf("despois de eliminar: %+v", l)
	}
}

func TestMoverProxectoRexeita(t *testing.T) {
	arrel := proxectosTemporal(t)
	p, _ := crearProxecto("Un", "", nil)
	dir, _ := dirProxecto(p.ID)
	if _, err := moverProxecto(p.ID, "relativo/dir"); err == nil {
		t.Error("unha ruta relativa debe fallar")
	}
	if _, err := moverProxecto(p.ID, filepath.Join(arrel, "non-existe")); err == nil {
		t.Error("un cartafol que non existe debe fallar")
	}
	if _, err := moverProxecto(p.ID, filepath.Join(dir, "boveda")); err == nil {
		t.Error("non se pode mover dentro de si mesmo")
	}
	activarProxecto(p.ID)
	if _, err := moverProxecto(p.ID, arrel); err == nil {
		t.Error("un proxecto aberto non se move")
	}
	activarProxecto("")
	if d, _ := dirProxecto(p.ID); d != dir {
		t.Errorf("tras os erros segue onde estaba: %s", d)
	}
}

// Cada proxecto pode levar a cor que escolla o usuario: só #rrggbb (gárdase en
// minúsculas) ou "" para volver á de por defecto.
func TestCorDoProxecto(t *testing.T) {
	proxectosTemporal(t)
	p, _ := crearProxecto("P", "", nil)
	q, err := fixarCorProxecto(p.ID, " #3A7BD5 ")
	if err != nil || q.Cor != "#3a7bd5" {
		t.Fatalf("cor: %+v %v", q, err)
	}
	if l, _ := listarProxectos(); len(l) != 1 || l[0].Cor != "#3a7bd5" {
		t.Errorf("a cor ten que gardarse na ficha: %+v", l)
	}
	for _, mala := range []string{"azul", "#12345", "#gggggg", "red;}", "#1234567"} {
		if _, err := fixarCorProxecto(p.ID, mala); err == nil {
			t.Errorf("debería rexeitar %q", mala)
		}
	}
	if q, _ := lerProxecto(p.ID); q.Cor != "#3a7bd5" {
		t.Errorf("unha cor mala non cambia a que había: %q", q.Cor)
	}
	if q, err := fixarCorProxecto(p.ID, ""); err != nil || q.Cor != "" {
		t.Errorf("baleira = por defecto: %+v %v", q, err)
	}
	if _, err := fixarCorProxecto("nonexiste", "#000000"); err == nil {
		t.Error("un proxecto que non existe non ten cor")
	}
	// Unha ficha que chega doutro sitio (un paquete importado) cunha cor mala
	// lese coa de por defecto.
	q, _ = lerProxecto(p.ID)
	q.Cor = "red;}"
	if err := escribirProxecto(q); err != nil {
		t.Fatal(err)
	}
	if q, _ := lerProxecto(p.ID); q.Cor != "" {
		t.Errorf("cor mala lida: %q", q.Cor)
	}
}
