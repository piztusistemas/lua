package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCatalogoCartafolesEReutilizar(t *testing.T) {
	trasladarTemporal(t)
	a := &App{}
	alfa := formularioDeProba(t, "Alfa") // na biblioteca (sen proxecto)
	beta := formularioDeProba(t, "Beta")
	p, _ := crearProxecto("Proxecto", "", nil)
	if _, err := activarProxecto(p.ID); err != nil {
		t.Fatal(err)
	}
	c, err := a.CatalogoFormularios()
	if err != nil || len(c.Formularios) != 2 || len(c.Integradas) != 2 || len(c.NoProxecto) != 0 {
		t.Fatalf("catálogo: %+v %v", c, err)
	}
	// Cartafoles: crear (sen duplicar), mover, renomear e eliminar.
	id, err := a.CrearCarpetaCatalogo("Lingua")
	if err != nil {
		t.Fatal(err)
	}
	if id2, _ := a.CrearCarpetaCatalogo("lingua"); id2 != id {
		t.Fatal("un cartafol co mesmo nome non se duplica")
	}
	if err := a.MoverFormularioCatalogo(alfa.ID, id); err != nil {
		t.Fatal(err)
	}
	if err := a.MoverFormularioCatalogo(alfa.ID, "non-existe"); err == nil {
		t.Fatal("non se pode mover a un cartafol que non existe")
	}
	if err := a.RenomearCarpetaCatalogo(id, "Linguas"); err != nil {
		t.Fatal(err)
	}
	c, _ = a.CatalogoFormularios()
	if c.Carpetas[0].Nome != "Linguas" || c.Formularios[0].Carpeta != id {
		t.Fatalf("cartafol: %+v", c)
	}
	// Reutilizar dous no proxecto aberto (e o cronograma).
	pr, err := a.ReutilizarNoProxecto([]string{"form:" + alfa.ID, "form:" + beta.ID, "cronograma"})
	if err != nil || len(pr.Ferramentas) != 3 {
		t.Fatalf("reutilizar: %+v %v", pr, err)
	}
	dir, _ := dirProxecto(p.ID)
	if _, err := os.Stat(filepath.Join(dir, "formularios", beta.ID+".json")); err != nil {
		t.Fatal("falta a copia no proxecto")
	}
	if c, _ = a.CatalogoFormularios(); len(c.NoProxecto) != 3 {
		t.Fatalf("o catálogo ten que saber que xa están no proxecto: %+v", c.NoProxecto)
	}
	// Eliminar o cartafol non borra o formulario.
	if err := a.EliminarCarpetaCatalogo(id); err != nil {
		t.Fatal(err)
	}
	c, _ = a.CatalogoFormularios()
	if len(c.Carpetas) != 0 || len(c.Formularios) != 2 || c.Formularios[0].Carpeta != "" {
		t.Fatalf("despois de eliminar o cartafol: %+v", c)
	}
}

func TestFormularioGardadoNoProxectoChegaAoCatalogo(t *testing.T) {
	trasladarTemporal(t)
	a := &App{}
	p, _ := crearProxecto("Proxecto", "", nil)
	_, _ = activarProxecto(p.ID)
	f, err := a.GardarFormulario(Formulario{Nome: "Novo", Campos: []CampoFormulario{{Tipo: campoTexto, Etiqueta: "X"}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lerFormularioBiblioteca(f.ID); err != nil {
		t.Fatal("o formulario gardado no proxecto tiña que chegar ao catálogo")
	}
	// Gardalo outra vez cun cambio actualiza o mesmo (non crea copias).
	f.Nome = "Novo cambiado"
	if _, err := a.GardarFormulario(f); err != nil {
		t.Fatal(err)
	}
	c, _ := a.CatalogoFormularios()
	if len(c.Formularios) != 1 || c.Formularios[0].Nome != "Novo cambiado" {
		t.Fatalf("catálogo: %+v", c.Formularios)
	}
	// Se o do catálogo está en produción, non se pisa: entra coma copia.
	lib, _ := lerFormularioBiblioteca(f.ID)
	lib.Modo = modoProduccion
	ruta, _ := dirBibliotecaFormularios()
	_ = escribirXSONAtomico(filepath.Join(ruta, f.ID+".json"), lib)
	f.Nome = "Outra versión"
	if _, err := a.GardarFormulario(f); err != nil {
		t.Fatal(err)
	}
	if orixinal, _ := lerFormularioBiblioteca(f.ID); orixinal.Nome != "Novo cambiado" {
		t.Fatal("o do catálogo en produción non se pode pisar")
	}
	if c, _ = a.CatalogoFormularios(); len(c.Formularios) != 2 {
		t.Fatalf("tiña que entrar coma copia: %+v", c.Formularios)
	}
}

func TestReutilizarSenProxectoFalla(t *testing.T) {
	trasladarTemporal(t)
	if _, err := (&App{}).ReutilizarNoProxecto([]string{"cronograma"}); err == nil {
		t.Fatal("sen proxecto aberto non se pode reutilizar")
	}
}

func execCatalogo(t *testing.T, nome string, args map[string]any) any {
	t.Helper()
	for _, f := range ferramentasBackend {
		if f.def.Nome == nome {
			out, err := f.exec(&App{}, args)
			if err != nil {
				t.Fatalf("%s: %v", nome, err)
			}
			return out
		}
	}
	t.Fatalf("non existe a ferramenta %s", nome)
	return nil
}

func TestAsistenteMoveFormularioACartafolNovo(t *testing.T) {
	trasladarTemporal(t)
	f := formularioDeProba(t, "Ficha")
	execCatalogo(t, "catalogo_mover", map[string]any{"ids": []any{"form:" + f.ID}, "cartafol": "Gramática"})
	c := execCatalogo(t, "catalogo_ler", nil).(CatalogoFormularios)
	if len(c.Carpetas) != 1 || c.Carpetas[0].Nome != "Gramática" || c.Formularios[0].Carpeta != c.Carpetas[0].ID {
		t.Fatalf("catálogo: %+v", c)
	}
	execCatalogo(t, "catalogo_cartafol_eliminar", map[string]any{"id": c.Carpetas[0].ID})
	if c = execCatalogo(t, "catalogo_ler", nil).(CatalogoFormularios); len(c.Carpetas) != 0 || len(c.Formularios) != 1 {
		t.Fatalf("despois de eliminar: %+v", c)
	}
	// As do catálogo están no asistente principal (non se filtran coma as formulario_*).
	nomes := map[string]bool{}
	for _, d := range (&App{}).IAFerramentas() {
		nomes[d.Nome] = true
	}
	for _, n := range []string{"catalogo_ler", "catalogo_mover", "catalogo_reutilizar", "catalogo_cartafol_eliminar"} {
		if !nomes[n] {
			t.Errorf("falta %s no asistente principal", n)
		}
	}
}
