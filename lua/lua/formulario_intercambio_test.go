package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// entornoIntercambio aparta a biblioteca E os borradores nun cartafol de proba
// (importar deixa os datos coma borrador: nunca na configuración real).
func entornoIntercambio(t *testing.T) string {
	t.Helper()
	bibliotecaTemporal(t)
	dir := t.TempDir()
	orixinal := rutaBorradorFormulario
	rutaBorradorFormulario = func(id string) (string, error) { return filepath.Join(dir, "borrador-"+id+".json"), nil }
	t.Cleanup(func() { rutaBorradorFormulario = orixinal })
	return dir
}

// formularioIntercambio: un formulario coma o do docente, con táboa, tubería e gráfico.
func formularioIntercambio(t *testing.T) Formulario {
	t.Helper()
	f := formularioGrafico("barras", true)
	f.Descricion = "Resultados por grupo"
	g, err := gardarFormulario(f)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

// mesmoJSON compara como se gardan os datos: un mapa nil e un baleiro son o mesmo no ficheiro.
func mesmoJSON(a, b any) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}

func exportar(t *testing.T, f Formulario, datos *DatosIntercambioFormulario) []byte {
	t.Helper()
	b, err := escribirBundleFormulario(construirBundleFormulario(f, datos))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestIntercambioFormularioSenDatosDaOMesmoFormulario(t *testing.T) {
	entornoIntercambio(t)
	f := formularioIntercambio(t)
	ficheiro := exportar(t, f, nil)
	if strings.Contains(string(ficheiro), `"modo"`) || strings.Contains(string(ficheiro), `"datos"`) {
		t.Errorf("o ficheiro non debe levar modo nin datos:\n%s", ficheiro)
	}

	// Noutro equipo (biblioteca baleira): entra igual, en deseño.
	entornoIntercambio(t)
	r, err := importarBundleFormulario(ficheiro)
	if err != nil {
		t.Fatal(err)
	}
	g := r.Formulario
	if g.ID != f.ID || g.Nome != f.Nome || g.Descricion != f.Descricion || g.Modo != modoDeseno {
		t.Errorf("importado = %+v", g)
	}
	if !mesmoJSON(g.Campos, f.Campos) || !mesmoJSON(g.Fluxo, f.Fluxo) {
		t.Error("os campos ou a tubería cambiaron ao importar")
	}
	if r.Datos != nil || len(r.Avisos) != 0 {
		t.Errorf("datos=%v avisos=%v", r.Datos, r.Avisos)
	}
	// E quedou gardado na biblioteca.
	l, err := lerFormulario(g.ID)
	if err != nil || l.Nome != f.Nome {
		t.Errorf("non se gardou: %v", err)
	}
	// O formulario importado xera o mesmo documento.
	b1, _ := bloquesFormulario(f, datosGrafico(), txProba())
	b2, _ := bloquesFormulario(l, datosGrafico(), txProba())
	if len(b1) != len(b2) || len(b1) == 0 {
		t.Errorf("bloques %d e %d", len(b1), len(b2))
	}
}

func TestImportarDuasVecesNonPisaEntraCoaCopia(t *testing.T) {
	entornoIntercambio(t)
	f := formularioIntercambio(t)
	ficheiro := exportar(t, f, nil)
	r, err := importarBundleFormulario(ficheiro)
	if err != nil {
		t.Fatal(err)
	}
	if r.Formulario.ID == f.ID || r.Formulario.ID != f.ID+"_2" || r.Formulario.Nome != f.Nome+" (2)" {
		t.Errorf("a copia = %q / %q", r.Formulario.ID, r.Formulario.Nome)
	}
	if len(r.Avisos) != 1 || !strings.Contains(r.Avisos[0], "coma copia") {
		t.Errorf("avisos = %v", r.Avisos)
	}
	// O orixinal segue igual e agora hai dous.
	if o, err := lerFormulario(f.ID); err != nil || o.Nome != f.Nome {
		t.Errorf("o orixinal cambiou: %+v %v", o, err)
	}
	if r3, err := importarBundleFormulario(ficheiro); err != nil || r3.Formulario.ID != f.ID+"_3" || r3.Formulario.Nome != f.Nome+" (3)" {
		t.Errorf("terceira: %+v %v", r3.Formulario.ID, err)
	}
	if l, _ := listarFormularios(); len(l) != 3 {
		t.Errorf("formularios = %d", len(l))
	}
}

func TestModoNonViaxaNunhaProduccionSenDatosImportaEnDeseno(t *testing.T) {
	entornoIntercambio(t)
	f := formularioIntercambio(t)
	f, err := cambiarModoFormulario(f.ID, modoProduccion)
	if err != nil {
		t.Fatal(err)
	}
	ficheiro := exportar(t, f, nil)
	if strings.Contains(string(ficheiro), "produccion") {
		t.Error("o modo produción non debe viaxar")
	}
	entornoIntercambio(t)
	r, err := importarBundleFormulario(ficheiro)
	if err != nil || r.Formulario.Modo != modoDeseno {
		t.Errorf("modo = %q (%v)", r.Formulario.Modo, err)
	}
}

func datosIntercambio(proba bool) *DatosIntercambioFormulario {
	return &DatosIntercambioFormulario{
		Contexto: ContextoFormulario{Centro: "IES de Probas", Curso: "2026-2027"},
		Valores: map[string]any{
			"suspensos": "2", "notables": "3",
			"por_grupo":  []any{map[string]any{"grupo": "1º A", "aprobados": "22", "suspensos": "3"}},
			"non_existe": "sobra",
		},
		Proba: proba,
	}
}

func TestImportarDatosDeProbaEntraEnDeseno(t *testing.T) {
	entornoIntercambio(t)
	f := formularioIntercambio(t)
	ficheiro := exportar(t, f, datosIntercambio(true))
	// O que non é un campo do formulario nin se exporta.
	if strings.Contains(string(ficheiro), "non_existe") {
		t.Error("un valor sen campo non debería exportarse")
	}
	entornoIntercambio(t)
	r, err := importarBundleFormulario(ficheiro)
	if err != nil {
		t.Fatal(err)
	}
	if r.Formulario.Modo != modoDeseno || r.Datos == nil || !r.Datos.Proba {
		t.Fatalf("modo=%q datos=%+v", r.Formulario.Modo, r.Datos)
	}
	if r.Datos.Valores["suspensos"] != "2" || r.Datos.Contexto.Centro != "IES de Probas" {
		t.Errorf("datos = %+v", r.Datos)
	}
	if len(r.Avisos) != 0 {
		t.Errorf("avisos = %v", r.Avisos)
	}
}

// Os datos importados quedan coma borrador do formulario novo, no formato que
// le a app ao abrilo (o do documento dun formulario), e sen datos non hai borrador.
func TestImportarDeixaOsDatosCommaBorrador(t *testing.T) {
	entornoIntercambio(t)
	f := formularioIntercambio(t)
	ficheiro := exportar(t, f, datosIntercambio(true))
	dir := entornoIntercambio(t) // o equipo que recibe o ficheiro: outra biblioteca e outros borradores
	r, err := importarBundleFormulario(ficheiro)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "borrador-"+r.Formulario.ID+".json"))
	if err != nil {
		t.Fatalf("non hai borrador: %v", err)
	}
	var doc struct {
		Ferramenta string                     `json:"ferramenta"`
		Formulario string                     `json:"formulario"`
		Datos      DatosIntercambioFormulario `json:"datos"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Ferramenta != "formulario" || doc.Formulario != r.Formulario.ID || doc.Datos.Valores["notables"] != "3" ||
		doc.Datos.Contexto.Centro != "IES de Probas" || !doc.Datos.Proba {
		t.Errorf("borrador = %s", raw)
	}
	// Un formulario sen datos non deixa borrador.
	sen := exportar(t, f, nil)
	r2, err := importarBundleFormulario(sen)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "borrador-"+r2.Formulario.ID+".json")); !os.IsNotExist(err) {
		t.Error("un formulario sen datos non debe deixar borrador")
	}
}

func TestImportarDatosReaisEntraEnProduccionSenIA(t *testing.T) {
	entornoIntercambio(t)
	f := formularioIntercambio(t)
	ficheiro := exportar(t, f, datosIntercambio(false))
	entornoIntercambio(t)
	r, err := importarBundleFormulario(ficheiro)
	if err != nil {
		t.Fatal(err)
	}
	if r.Formulario.Modo != modoProduccion || r.Datos == nil {
		t.Fatalf("modo=%q datos=%+v", r.Formulario.Modo, r.Datos)
	}
	if !strings.Contains(strings.Join(r.Avisos, "|"), "PRODUCIÓN") {
		t.Errorf("avisos = %v", r.Avisos)
	}
	// E queda así no disco, non só na resposta.
	if l, _ := lerFormulario(r.Formulario.ID); !l.enProduccion() {
		t.Error("o formulario importado con datos reais non quedou en produción")
	}
}

func TestDatosSenContidoNonVan(t *testing.T) {
	f := formularioIntercambio(t)
	baleiros := &DatosIntercambioFormulario{Valores: map[string]any{"suspensos": "", "por_grupo": []any{map[string]any{"grupo": " "}}, "notables": nil}}
	if b := construirBundleFormulario(f, baleiros); b.Datos != nil {
		t.Errorf("datos baleiros non deben exportarse: %+v", b.Datos)
	}
	// Un formulario en produción sen contido segue sen arrastrar modo ningún.
	if b := construirBundleFormulario(f, nil); b.Datos != nil || b.Formulario.Modo != "" {
		t.Errorf("bundle = %+v", b)
	}
}

func TestImportarRexeitaFicheirosMalos(t *testing.T) {
	entornoIntercambio(t)
	f := formularioIntercambio(t)
	bom := construirBundleFormulario(f, nil)
	con := func(cambia func(*BundleFormulario)) []byte {
		b := bom
		cambia(&b)
		raw, _ := json.Marshal(b)
		return raw
	}
	casos := map[string]struct {
		datos []byte
		quere string
	}{
		"non é json":            {[]byte("isto non é json"), "JSON válido"},
		"outro formato":         {[]byte(`{"formato":"lua-plantilla","version":1}`), "non é un formulario de Lua"},
		"un documento de datos": {[]byte(`{"ferramenta":"formulario","formulario":"exame","datos":{}}`), "DATOS dun formulario"},
		"versión futura":        {con(func(b *BundleFormulario) { b.Version = 2 }), "actualiza Lua"},
		"sen versión":           {con(func(b *BundleFormulario) { b.Version = 0 }), "versión"},
		"sen nome":              {con(func(b *BundleFormulario) { b.Formulario.Nome = "" }), "non é válido"},
		"campo raro":            {con(func(b *BundleFormulario) { b.Formulario.Campos[0].Tipo = "boton" }), "non é válido"},
		"demasiado grande":      {make([]byte, maxBytesBundleFormulario+1), "demasiado grande"},
	}
	for nome, c := range casos {
		r, err := importarBundleFormulario(c.datos)
		if err == nil || !strings.Contains(err.Error(), c.quere) {
			t.Errorf("%s: err=%v, agardábase %q", nome, err, c.quere)
		}
		if r.Formulario.ID != "" {
			t.Errorf("%s: non debería devolver un formulario", nome)
		}
	}
	// Nada se gardou polo camiño: só o formulario orixinal.
	if l, _ := listarFormularios(); len(l) != 1 {
		t.Errorf("a biblioteca ten %d formularios", len(l))
	}
}

// Un formulario gardado polo deseñador de verdade (con tubería) sobrevive
// exportar -> importar sen perder nada nos cálculos.
func TestIntercambioConservaAsTuberias(t *testing.T) {
	entornoIntercambio(t)
	f := formularioNotas()
	f.Fluxo = fluxoNotas()
	g, err := gardarFormulario(f)
	if err != nil {
		t.Fatal(err)
	}
	ficheiro := exportar(t, g, nil)
	entornoIntercambio(t)
	r, err := importarBundleFormulario(ficheiro)
	if err != nil {
		t.Fatal(err)
	}
	if !mesmoJSON(r.Formulario.Fluxo, g.Fluxo) {
		t.Error("a tubería cambiou")
	}
	b1, _ := bloquesFormulario(g, datosNotas(), txProba())
	b2, _ := bloquesFormulario(r.Formulario, datosNotas(), txProba())
	if !reflect.DeepEqual(b1, b2) {
		t.Errorf("o documento xerado é distinto:\n%+v\n%+v", b1, b2)
	}
}
