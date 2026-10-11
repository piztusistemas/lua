package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// trasladarTemporal aísla proxectos, biblioteca e <config>/lua.
func trasladarTemporal(t *testing.T) string {
	t.Helper()
	arrel := proxectosTemporal(t)
	oCfg := dirConfigLua
	cfg := filepath.Join(arrel, "cfglua")
	dirConfigLua = func() (string, error) { return cfg, nil }
	t.Cleanup(func() { dirConfigLua = oCfg })
	return arrel
}

func escribir(t *testing.T, p, s string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}

func ler(t *testing.T, p string) string {
	t.Helper()
	d, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(d)
}

func TestVersionGardaERestaura(t *testing.T) {
	arrel := trasladarTemporal(t)
	f := filepath.Join(arrel, "a", "doc.json")
	dir := filepath.Join(arrel, "b")
	novo := filepath.Join(arrel, "c.json")
	escribir(t, f, "vello")
	escribir(t, filepath.Join(dir, "x", "y.txt"), "y")
	id, err := gardarVersion("actualizar", "proba", []string{f, dir, novo})
	if err != nil {
		t.Fatal(err)
	}
	escribir(t, f, "novo")
	_ = os.RemoveAll(dir)
	escribir(t, novo, "creado despois")
	if err := restaurarVersion(id); err != nil {
		t.Fatal(err)
	}
	if ler(t, f) != "vello" || ler(t, filepath.Join(dir, "x", "y.txt")) != "y" {
		t.Fatal("non se restaurou o contido")
	}
	if _, err := os.Stat(novo); !os.IsNotExist(err) {
		t.Fatal("unha ruta que non existía tiña que borrarse ao desfacer")
	}
}

func TestVersionGardaSoCincoMaisNovas(t *testing.T) {
	arrel := trasladarTemporal(t)
	f := filepath.Join(arrel, "doc.json")
	escribir(t, f, "x")
	ids := []string{}
	for i := 0; i < maxVersions+2; i++ {
		id, err := gardarVersion("proba", "", []string{f})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	base, _ := dirVersions()
	ents, _ := os.ReadDir(base)
	if len(ents) != maxVersions {
		t.Fatalf("hai %d versións, agardaba %d", len(ents), maxVersions)
	}
	if err := restaurarVersion(ids[0]); err == nil {
		t.Fatal("a versión máis vella tiña que estar borrada")
	}
	if err := restaurarVersion(ids[len(ids)-1]); err != nil {
		t.Fatal(err)
	}
}

func TestVersionRexeitaIDPerigoso(t *testing.T) {
	trasladarTemporal(t)
	if err := restaurarVersion("../../etc"); err == nil {
		t.Fatal("tiña que rexeitar un id con ..")
	}
}

func TestMigracionTraballoSolto(t *testing.T) {
	trasladarTemporal(t)
	f := formularioDeProba(t, "Alfa")
	cfg, _ := dirConfigLua()
	inf := filepath.Join(cfg, "informes")
	escribir(t, filepath.Join(inf, "cronograma-borrador.json"), `{"path":"/vello/curso.json","plans":[{"academicYear":"2025-2026"}]}`)
	escribir(t, filepath.Join(inf, "actas-borrador.json"), `{"ferramenta":"actas","actas":[{}]}`)
	escribir(t, filepath.Join(inf, "formulario-"+f.ID+"-borrador.json"), `{"ferramenta":"formulario","formulario":"`+f.ID+`","datos":{"valores":{"x":"y"}}}`)
	escribir(t, filepath.Join(inf, "formulario-borrado-borrador.json"), `{}`)

	res, err := migrarTraballoSolto()
	if err != nil {
		t.Fatal(err)
	}
	if res.Proxecto == "" || res.Nome != "O meu traballo" || len(res.Ferramentas) != 3 {
		t.Fatalf("migración: %+v", res)
	}
	doc, _ := rutaDocumentoProxecto(res.Proxecto, "cronograma")
	contido := ler(t, doc)
	if !strings.Contains(contido, "2025-2026") || strings.Contains(contido, "/vello/curso.json") {
		t.Fatalf("o cronograma non migrou ben: %s", contido)
	}
	if _, err := os.Stat(filepath.Join(inf, "cronograma-borrador.json")); !os.IsNotExist(err) {
		t.Fatal("o borrador solto tiña que saír de alí")
	}
	if _, err := os.Stat(filepath.Join(inf, "formulario-borrado-borrador.json")); err != nil {
		t.Fatal("os datos dun formulario que xa non existe quedan onde estaban")
	}
	// Idempotente: a segunda vez non fai nada.
	if res2, err := migrarTraballoSolto(); err != nil || res2.Proxecto != "" {
		t.Fatalf("segunda vez: %+v %v", res2, err)
	}
	// Desfacer volve deixar os borradores soltos.
	if err := restaurarVersion(res.Version); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(inf, "cronograma-borrador.json")); err != nil {
		t.Fatal("desfacer tiña que devolver o borrador")
	}
}

func TestMigracionSenNadaSoMarca(t *testing.T) {
	trasladarTemporal(t)
	res, err := migrarTraballoSolto()
	if err != nil || res.Proxecto != "" {
		t.Fatalf("%+v %v", res, err)
	}
	if ps, _ := listarProxectos(); len(ps) != 0 {
		t.Fatal("sen traballo solto non se crea ningún proxecto")
	}
}
