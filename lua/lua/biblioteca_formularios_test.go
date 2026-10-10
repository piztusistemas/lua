package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func formularioDeProba(t *testing.T, nome string) Formulario {
	t.Helper()
	f, err := gardarFormulario(Formulario{Nome: nome, Campos: []CampoFormulario{{Tipo: campoTexto, Etiqueta: "X"}}})
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestBibliotecaPorDefecto(t *testing.T) {
	bibliotecaTemporal(t)
	b, err := lerBiblioteca()
	if err != nil {
		t.Fatal(err)
	}
	// O cronograma e as actas só están na biblioteca (agochados) ata que o usuario os active.
	if len(b.Fixos) != 0 || !reflect.DeepEqual(b.Agochados, []string{"cronograma", "actas"}) || len(b.Carpetas) != 0 || b.Carpeta == nil {
		t.Fatalf("por defecto: %+v", b)
	}
}

func TestCarpetasDaBiblioteca(t *testing.T) {
	bibliotecaTemporal(t)
	a := formularioDeProba(t, "Alfa")

	b, err := crearCarpetaFormularios("  Titoría 1º ESO ")
	if err != nil || len(b.Carpetas) != 1 || b.Carpetas[0].Nome != "Titoría 1º ESO" || !idFormularioRe.MatchString(b.Carpetas[0].ID) {
		t.Fatalf("crear: %+v, %v", b, err)
	}
	id := b.Carpetas[0].ID
	if _, err := crearCarpetaFormularios("titoría 1º eso"); err == nil {
		t.Error("un cartafol co mesmo nome (sen mirar maiúsculas) debería fallar")
	}
	if _, err := crearCarpetaFormularios("   "); err == nil {
		t.Error("un cartafol sen nome debería fallar")
	}
	// Ids distintos aínda que os nomes dean o mesmo slug.
	b2, err := crearCarpetaFormularios("Titoria 1 ESO!")
	if err != nil || b2.Carpetas[1].ID == id {
		t.Fatalf("ids: %+v, %v", b2, err)
	}

	// Mover o formulario a un cartafol e persiste.
	b2.Carpeta[a.ID] = id
	g, err := gardarBiblioteca(b2)
	if err != nil || g.Carpeta[a.ID] != id {
		t.Fatalf("mover: %+v, %v", g, err)
	}
	if l, _ := lerBiblioteca(); l.Carpeta[a.ID] != id {
		t.Fatalf("non persistiu: %+v", l)
	}

	// Borrar o cartafol NON borra o formulario: pasa á raíz.
	g.Carpetas = g.Carpetas[1:]
	g, err = gardarBiblioteca(g)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := g.Carpeta[a.ID]; ok {
		t.Errorf("o formulario debería quedar sen cartafol: %+v", g.Carpeta)
	}
	if _, err := lerFormulario(a.ID); err != nil {
		t.Errorf("o formulario debe seguir existindo: %v", err)
	}
}

func TestNormalizarBibliotecaLimpaReferencias(t *testing.T) {
	bibliotecaTemporal(t)
	a := formularioDeProba(t, "Alfa")
	sucia := BibliotecaFormularios{
		Carpetas:  []CarpetaFormularios{{ID: "c1", Nome: "A"}, {ID: "c1", Nome: "B"}, {Nome: "C"}},
		Carpeta:   map[string]string{a.ID: "c1", "fantasma": "c1", "outro": "nonexiste"},
		Fixos:     []string{"form:" + a.ID, "form:fantasma", "form:" + a.ID, "cronograma"},
		Agochados: []string{"form:fantasma", "actas", "actas", "raro"},
	}
	g, err := gardarBiblioteca(sucia)
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]bool{}
	for _, c := range g.Carpetas {
		if ids[c.ID] || !idFormularioRe.MatchString(c.ID) {
			t.Errorf("ids de cartafol repetidos ou malos: %+v", g.Carpetas)
		}
		ids[c.ID] = true
	}
	if len(g.Carpetas) != 3 {
		t.Errorf("cartafoles: %+v", g.Carpetas)
	}
	if len(g.Carpeta) != 1 || g.Carpeta[a.ID] != "c1" {
		t.Errorf("carpeta: %+v", g.Carpeta)
	}
	// Sen fantasmas nin repeticións; as integradas non se obrigan á zona fixa.
	if !reflect.DeepEqual(g.Fixos, []string{"form:" + a.ID, "cronograma"}) {
		t.Errorf("fixos: %v", g.Fixos)
	}
	if !reflect.DeepEqual(g.Agochados, []string{"actas"}) {
		t.Errorf("agochados: %v", g.Agochados)
	}
}

// Pódese deixar a columna baleira: cronograma e actas só están na biblioteca ata que se activan.
func TestPodeAgocharseTodo(t *testing.T) {
	bibliotecaTemporal(t)
	a := formularioDeProba(t, "Alfa")
	b, _ := lerBiblioteca()
	b.Agochados = []string{"cronograma", "actas", "form:" + a.ID}
	g, err := gardarBiblioteca(b)
	if err != nil || len(g.Agochados) != 3 {
		t.Errorf("agochar todo debería valer: %+v, %v", g, err)
	}
}

// O que está na zona fixa non se pode borrar (só ocultar ou quitar da zona), e ao
// borrar un formulario esquécese o seu sitio na biblioteca.
func TestBorrarFormularioRespectaAZonaFixa(t *testing.T) {
	bibliotecaTemporal(t)
	a := formularioDeProba(t, "Alfa")
	b, _ := lerBiblioteca()
	c, err := crearCarpetaFormularios("Arquivo")
	if err != nil {
		t.Fatal(err)
	}
	b = c
	b.Fixos = append(b.Fixos, "form:"+a.ID)
	b.Carpeta[a.ID] = b.Carpetas[0].ID
	b.Agochados = []string{"form:" + a.ID}
	if _, err := gardarBiblioteca(b); err != nil {
		t.Fatal(err)
	}
	if err := eliminarFormulario(a.ID); err == nil {
		t.Fatal("un formulario da zona fixa non se pode borrar")
	}
	if _, err := lerFormulario(a.ID); err != nil {
		t.Fatalf("non debería ter desaparecido: %v", err)
	}
	// Ocultar e quitar da zona fixa si se pode; despois xa se pode borrar.
	b, _ = lerBiblioteca()
	b.Fixos = []string{"cronograma", "actas"}
	if _, err := gardarBiblioteca(b); err != nil {
		t.Fatal(err)
	}
	if err := eliminarFormulario(a.ID); err != nil {
		t.Fatalf("fóra da zona fixa xa se borra: %v", err)
	}
	if _, err := lerFormulario(a.ID); !os.IsNotExist(err) {
		t.Errorf("o ficheiro debería ter desaparecido: %v", err)
	}
	g, _ := lerBiblioteca()
	if len(g.Carpeta) != 0 || len(g.Agochados) != 0 || contemString(g.Fixos, "form:"+a.ID) {
		t.Errorf("a biblioteca aínda apunta ao formulario borrado: %+v", g)
	}
}

// A biblioteca vive fóra da carpeta dos formularios: non aparece coma un formulario.
func TestBibliotecaNonSeMestura(t *testing.T) {
	bibliotecaTemporal(t)
	formularioDeProba(t, "Alfa")
	if _, err := crearCarpetaFormularios("Unha"); err != nil {
		t.Fatal(err)
	}
	dir, _ := dirFormularios()
	if _, err := os.Stat(filepath.Join(filepath.Dir(dir), "biblioteca_formularios.json")); err != nil {
		t.Fatalf("o ficheiro da biblioteca non está onde debe: %v", err)
	}
	if l, _ := listarFormularios(); len(l) != 1 {
		t.Errorf("a biblioteca non é un formulario: %+v", l)
	}
	// Estragada: cae ao por defecto sen impedir nada.
	ruta, _ := rutaBiblioteca()
	if err := os.WriteFile(ruta, []byte("{isto non"), 0o644); err != nil {
		t.Fatal(err)
	}
	if b, err := lerBiblioteca(); err != nil || len(b.Agochados) != 2 {
		t.Errorf("estragada: %+v, %v", b, err)
	}
}

func TestDescricionDeFormularioVaiEnCalqueraModo(t *testing.T) {
	bibliotecaTemporal(t)
	a := formularioDeProba(t, "Alfa")
	f, err := actualizarDescricionFormulario(a.ID, "  Para as titorías  ")
	if err != nil || f.Descricion != "Para as titorías" {
		t.Fatalf("descrición: %+v, %v", f, err)
	}
	// Tamén en produción (o esquema está conxelado, pero a descrición é da biblioteca)...
	if _, err := cambiarModoFormulario(a.ID, modoProduccion); err != nil {
		t.Fatal(err)
	}
	f, err = actualizarDescricionFormulario(a.ID, "Nova")
	if err != nil || f.Descricion != "Nova" || !f.enProduccion() || len(f.Campos) != 1 {
		t.Fatalf("en produción: %+v, %v", f, err)
	}
	// ...e non se pasa dun límite.
	longa := make([]rune, 1000)
	for i := range longa {
		longa[i] = 'ñ'
	}
	if f, _ := actualizarDescricionFormulario(a.ID, string(longa)); len([]rune(f.Descricion)) != maxDescricionFormulario {
		t.Errorf("a descrición debería cortarse en %d", maxDescricionFormulario)
	}
	if _, err := actualizarDescricionFormulario("non_existe", "x"); err == nil {
		t.Error("un formulario que non existe debería fallar")
	}
}

// A icona da columna esquerda cámbiase desde o menú do botón dereito, sen
// abrir o deseñador: vale en calquera modo e non toca os campos.
func TestIconaDeFormularioVaiEnCalqueraModo(t *testing.T) {
	bibliotecaTemporal(t)
	a := formularioDeProba(t, "Alfa")
	f, err := actualizarIconaFormulario(a.ID, "  estrela ", " rosa ")
	if err != nil || f.Icona != "estrela" || f.Cor != "rosa" {
		t.Fatalf("icona: %+v, %v", f, err)
	}
	if _, err := cambiarModoFormulario(a.ID, modoProduccion); err != nil {
		t.Fatal(err)
	}
	f, err = actualizarIconaFormulario(a.ID, "reloxo", "azul")
	if err != nil || f.Icona != "reloxo" || f.Cor != "azul" || !f.enProduccion() || len(f.Campos) != 1 {
		t.Fatalf("en produción: %+v, %v", f, err)
	}
	if g, _ := lerFormulario(a.ID); g.Icona != "reloxo" || g.Cor != "azul" {
		t.Errorf("non quedou gardada: %q %q", g.Icona, g.Cor)
	}
	if _, err := actualizarIconaFormulario(a.ID, "<script>", ""); err == nil {
		t.Error("unha clave de icona con caracteres raros debería rexeitarse")
	}
	if _, err := actualizarIconaFormulario(a.ID, "estrela", "vermello"); err == nil {
		t.Error("unha cor descoñecida debería rexeitarse")
	}
	if _, err := actualizarIconaFormulario("non_existe", "estrela", ""); err == nil {
		t.Error("un formulario que non existe debería fallar")
	}
}
