package main

import (
	"archive/zip"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// proxectoCompleto crea un proxecto con cronograma, un formulario con datos e
// unha bóveda cun PDF. Devolve o proxecto e o formulario.
func proxectoCompleto(t *testing.T) (Proxecto, Formulario) {
	t.Helper()
	f := formularioDeProba(t, "Titoría")
	p, err := crearProxecto("3º ESO", "Mates", []string{"cronograma", "form:" + f.ID})
	if err != nil {
		t.Fatal(err)
	}
	dir, _ := dirProxecto(p.ID)
	escribir(t, filepath.Join(dir, "datos", "cronograma-borrador.json"), `{"plans":[{"academicYear":"2025-2026"}]}`)
	escribir(t, filepath.Join(dir, "datos", borradorFerramentaNome("form:"+f.ID)), `{"ferramenta":"formulario","formulario":"`+f.ID+`","datos":{"valores":{"x":"dato"}}}`)
	escribir(t, filepath.Join(dir, "boveda", "pdf", "lei.pdf"), "%PDF-1.4 proba")
	escribir(t, filepath.Join(dir, "boveda", "raw", "lei.md"), "# Lei")
	escribir(t, filepath.Join(dir, "boveda", "wiki", "index.md"), "# Índice")
	return p, f
}

// sacarA escribe un paquete nun ficheiro fóra das rutas de Lúa.
func sacarA(t *testing.T, sel SeleccionSacar, s Settings) string {
	t.Helper()
	ruta := filepath.Join(t.TempDir(), "saida"+extensionPaquete)
	if _, err := escribirPaqueteEn(ruta, sel, s); err != nil {
		t.Fatal(err)
	}
	return ruta
}

func TestPaqueteProxectoIdaEVolta(t *testing.T) {
	trasladarTemporal(t)
	p, f := proxectoCompleto(t)
	pl, err := (&App{}).GardarPlantilla(PlantillaLatex{Nome: "Centro", Latex: "{{CORPO}}"})
	if err != nil {
		t.Fatal(err)
	}
	docCrono, _ := rutaDocumentoProxecto(p.ID, "cronograma")
	if err := lembrarPlantillaDocumento(docCrono, pl.ID); err != nil {
		t.Fatal(err)
	}
	ruta := sacarA(t, SeleccionSacar{Tipo: "proxecto", Proxectos: []string{p.ID}, Datos: true}, Settings{})

	trasladarTemporal(t) // outro equipo
	_, _ = activarProxecto("")
	a := &App{}
	r, err := a.clasificarMeter(ruta)
	if err != nil {
		t.Fatal(err)
	}
	if r.Tipo != "paquete" || r.Destino != "novo" || len(r.Elementos) != 2 || r.Elementos[0].Nome != "3º ESO" || r.Elementos[1].Tipo != "plantilla" {
		t.Fatalf("resumo: %+v", r)
	}
	res, err := a.aplicarMeter(ruta, OpcionsMeter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Proxectos) != 1 || res.AbrirProxecto != res.Proxectos[0] || res.Version == "" {
		t.Fatalf("resultado: %+v", res)
	}
	novo, err := lerProxecto(res.Proxectos[0])
	if err != nil || novo.Nome != "3º ESO" || len(novo.Ferramentas) != 2 {
		t.Fatalf("proxecto: %+v %v", novo, err)
	}
	dir, _ := dirProxecto(novo.ID)
	if ler(t, filepath.Join(dir, "boveda", "pdf", "lei.pdf")) != "%PDF-1.4 proba" || ler(t, filepath.Join(dir, "boveda", "wiki", "index.md")) != "# Índice" {
		t.Fatal("a bóveda non chegou igual")
	}
	if !strings.Contains(ler(t, filepath.Join(dir, "datos", borradorFerramentaNome("form:"+f.ID))), "dato") {
		t.Fatal("os datos do formulario non chegaron")
	}
	if _, err := os.Stat(filepath.Join(dir, "formularios", f.ID+".json")); err != nil {
		t.Fatal("falta a copia do formulario no proxecto")
	}
	alm, _ := loadAlmacenPlantillas()
	if usada := plantillaDeDocumento(alm, filepath.Join(dir, "datos", "cronograma-borrador.json")); usada == nil || usada.Nome != "Centro" {
		t.Fatalf("a plantilla do cronograma non viaxou co proxecto: %+v", usada)
	}
	// Meter outra vez non pisa: entra coma «3º ESO (2)».
	res2, err := a.aplicarMeter(ruta, OpcionsMeter{})
	if err != nil {
		t.Fatal(err)
	}
	if p2, _ := lerProxecto(res2.Proxectos[0]); p2.Nome != "3º ESO (2)" || p2.ID == novo.ID {
		t.Fatalf("segunda importación: %+v", p2)
	}
	// Desfacer a segunda borra o proxecto novo.
	if err := restaurarVersion(res2.Version); err != nil {
		t.Fatal(err)
	}
	if _, err := lerProxecto(res2.Proxectos[0]); err == nil {
		t.Fatal("desfacer tiña que borrar o proxecto importado")
	}
}

func TestPaqueteSenDatosNonLevaDatos(t *testing.T) {
	trasladarTemporal(t)
	p, _ := proxectoCompleto(t)
	ruta := sacarA(t, SeleccionSacar{Tipo: "proxecto", Proxectos: []string{p.ID}}, Settings{})
	zr, err := zip.OpenReader(ruta)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	pdf := false
	for _, f := range zr.File {
		if strings.Contains(f.Name, "/datos/") {
			t.Fatalf("sen datos e leva %s", f.Name)
		}
		pdf = pdf || strings.HasSuffix(f.Name, "lei.pdf")
	}
	if !pdf {
		t.Fatal("a bóveda vai sempre")
	}
}

func TestPaqueteSenClaves(t *testing.T) {
	trasladarTemporal(t)
	proxectoCompleto(t)
	ruta := sacarA(t, SeleccionSacar{Tipo: "todo"}, Settings{IAAPIKey: "sk-secreta-123", IAProvedor: "anthropic"})
	zr, err := zip.OpenReader(ruta)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	for _, f := range zr.File {
		rc, _ := f.Open()
		buf := make([]byte, f.UncompressedSize64)
		_, _ = rc.Read(buf)
		rc.Close()
		if strings.Contains(string(buf), "sk-secreta-123") {
			t.Fatalf("a clave saíu en %s", f.Name)
		}
	}
}

func TestPaqueteRutasPerigosas(t *testing.T) {
	trasladarTemporal(t)
	for _, nome := range []string{"../fora.txt", "/abs.txt", "a/../../b", `c:\x`, "a//b"} {
		ruta := filepath.Join(t.TempDir(), "mal.lua")
		out, _ := os.Create(ruta)
		zw := zip.NewWriter(out)
		w, _ := zw.Create("manifest.json")
		_, _ = w.Write([]byte(`{"formato":"lua-paquete","version":1}`))
		w, _ = zw.Create(nome)
		_, _ = w.Write([]byte("x"))
		_ = zw.Close()
		_ = out.Close()
		if _, err := abrirPaquete(ruta); err == nil {
			t.Fatalf("aceptou a ruta %q", nome)
		}
	}
}

func TestMeterDanadoNonCambiaNada(t *testing.T) {
	trasladarTemporal(t)
	a := &App{}
	dir := t.TempDir()
	for nome, contido := range map[string]string{
		"roto.json":   `{"plans": [`,
		"raro.json":   `{"ola": 1}`,
		"futuro.lua":  "PK\x03\x04lixo",
		"futuro.json": `{"formato":"lua-documento","version":99,"ferramenta":"cronograma","documento":{}}`,
	} {
		ruta := filepath.Join(dir, nome)
		escribir(t, ruta, contido)
		if _, err := a.aplicarMeter(ruta, OpcionsMeter{}); err == nil {
			t.Fatalf("%s: tiña que fallar", nome)
		}
	}
	if ps, _ := listarProxectos(); len(ps) != 0 {
		t.Fatalf("creáronse proxectos: %+v", ps)
	}
}

func TestClasificarFormatosVellos(t *testing.T) {
	trasladarTemporal(t)
	a := &App{}
	dir := t.TempDir()
	f := formularioDeProba(t, "Alfa")
	casos := map[string]struct{ contido, tipo, ferramenta string }{
		"crono-obx.json":   {`{"plans":[{"academicYear":"2024-2025"}]}`, "curso", "cronograma"},
		"crono-array.json": {`[{"academicYear":"a"},{"academicYear":"b"}]`, "curso", "cronograma"},
		"crono-un.json":    {`{"config":{},"monthsData":{}}`, "curso", "cronograma"},
		"actas.json":       {`{"ferramenta":"actas","actas":[{}]}`, "curso", "actas"},
		"form-doc.json":    {`{"ferramenta":"formulario","formulario":"` + f.ID + `","datos":{"valores":{}}}`, "documento", "form:" + f.ID},
		"form.json":        {`{"formato":"lua-formulario","version":1,"formulario":{"id":"beta","nome":"Beta","campos":[{"tipo":"texto","etiqueta":"Y"}]}}`, "formulario", ""},
		"plantilla.json":   {`{"formato":"lua-plantilla","version":1,"plantilla":{"nome":"Centro","latex":"{{CORPO}}"}}`, "plantilla", ""},
	}
	for nome, c := range casos {
		ruta := filepath.Join(dir, nome)
		escribir(t, ruta, c.contido)
		r, err := a.clasificarMeter(ruta)
		if err != nil {
			t.Fatalf("%s: %v", nome, err)
		}
		if r.Tipo != c.tipo || r.Ferramenta != c.ferramenta {
			t.Fatalf("%s: %+v", nome, r)
		}
	}
	// Sen proxecto aberto, un curso vai a un proxecto novo.
	ruta := filepath.Join(dir, "crono-obx.json")
	if r, _ := a.clasificarMeter(ruta); r.Destino != "novo" {
		t.Fatalf("destino sen proxecto: %s", r.Destino)
	}
	// Cun proxecto aberto que aínda non ten cronograma, vai ao aberto.
	p, _ := crearProxecto("Aberto", "", nil)
	if _, err := activarProxecto(p.ID); err != nil {
		t.Fatal(err)
	}
	if r, _ := a.clasificarMeter(ruta); r.Destino != "aberto" || !r.PodeAberto {
		t.Fatalf("destino co proxecto aberto: %+v", r)
	}
}

func TestMeterVariosCursos(t *testing.T) {
	trasladarTemporal(t)
	a := &App{}
	ruta := filepath.Join(t.TempDir(), "piztu.json")
	escribir(t, ruta, `{"plans":[{"academicYear":"2022-2023"},{"academicYear":"2023-2024"},{"academicYear":"2024-2025"}]}`)
	r, err := a.clasificarMeter(ruta)
	if err != nil || len(r.Cursos) != 3 || r.Destino != "novo" {
		t.Fatalf("%+v %v", r, err)
	}
	res, err := a.aplicarMeter(ruta, OpcionsMeter{Cursos: []int{0, 2}})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Proxectos) != 2 {
		t.Fatalf("agardaba dous proxectos: %+v", res)
	}
	p, _ := lerProxecto(res.Proxectos[1])
	if p.Nome != "Cronograma 2024-2025" || !contemString(p.Ferramentas, "cronograma") {
		t.Fatalf("proxecto: %+v", p)
	}
	doc, _ := rutaDocumentoProxecto(p.ID, "cronograma")
	if !strings.Contains(ler(t, doc), "2024-2025") {
		t.Fatal("o curso non quedou no proxecto")
	}
}

func TestMeterFormularioConESenDatos(t *testing.T) {
	trasladarTemporal(t)
	a := &App{}
	dir := t.TempDir()
	sen := filepath.Join(dir, "sen.json")
	escribir(t, sen, `{"formato":"lua-formulario","version":1,"formulario":{"id":"beta","nome":"Beta","campos":[{"tipo":"texto","etiqueta":"Y"}]}}`)
	res, err := a.aplicarMeter(sen, OpcionsMeter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Proxectos) != 0 || len(res.Formularios) != 1 {
		t.Fatalf("sen datos: %+v", res)
	}
	if _, err := lerFormularioBiblioteca("beta"); err != nil {
		t.Fatal("non está na biblioteca")
	}
	// Idéntico: reutilízase, non se duplica.
	if res, err = a.aplicarMeter(sen, OpcionsMeter{}); err != nil || res.Formularios[0] != "beta" {
		t.Fatalf("reutilizar: %+v %v", res, err)
	}
	con := filepath.Join(dir, "con.json")
	escribir(t, con, `{"formato":"lua-formulario","version":1,"formulario":{"id":"gamma","nome":"Gamma","campos":[{"id":"y","tipo":"texto","etiqueta":"Y"}]},"datos":{"contexto":{},"valores":{"y":"real"}}}`)
	r, _ := a.clasificarMeter(con)
	if r.Destino != "novo" || !r.Produccion {
		t.Fatalf("resumo con datos: %+v", r)
	}
	res, err = a.aplicarMeter(con, OpcionsMeter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Proxectos) != 1 {
		t.Fatalf("con datos: %+v", res)
	}
	p, _ := lerProxecto(res.Proxectos[0])
	if p.Nome != "Gamma" || !contemString(p.Ferramentas, "form:gamma") {
		t.Fatalf("proxecto: %+v", p)
	}
	pdir, _ := dirProxecto(p.ID)
	var copia Formulario
	_ = json.Unmarshal([]byte(ler(t, filepath.Join(pdir, "formularios", "gamma.json"))), &copia)
	if copia.Modo != modoProduccion {
		t.Fatal("con datos reais, a copia do proxecto vai en produción")
	}
	if _, err := lerFormularioBiblioteca("gamma"); err != nil {
		t.Fatal("o formulario tamén queda na biblioteca")
	}
}

func TestDocumentoVoltaActualizaEDesfai(t *testing.T) {
	trasladarTemporal(t)
	a := &App{settings: Settings{Instalacion: "esta"}}
	p, _ := proxectoCompleto(t)
	if _, err := activarProxecto(p.ID); err != nil {
		t.Fatal(err)
	}
	doc, _ := rutaDocumentoProxecto(p.ID, "cronograma")
	d, err := construirDocumentoTraslado("cronograma", ler(t, doc), "Cronograma", "esta")
	if err != nil {
		t.Fatal(err)
	}
	// Edítase fóra.
	d.Documento = json.RawMessage(`{"plans":[{"academicYear":"EDITADO"}]}`)
	ruta := filepath.Join(t.TempDir(), "crono.json")
	b, _ := json.Marshal(d)
	escribir(t, ruta, string(b))
	r, err := a.clasificarMeter(ruta)
	if err != nil || r.Volta == nil || r.Destino != "volta" {
		t.Fatalf("volta: %+v %v", r, err)
	}
	res, err := a.aplicarMeter(ruta, OpcionsMeter{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(ler(t, doc), "EDITADO") || !res.Recargar {
		t.Fatalf("non se actualizou: %+v", res)
	}
	if err := restaurarVersion(res.Version); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(ler(t, doc), "2025-2026") {
		t.Fatal("desfacer non volveu ao de antes")
	}
}

func TestDocumentoVoltaComoCopia(t *testing.T) {
	trasladarTemporal(t)
	a := &App{settings: Settings{Instalacion: "esta"}}
	p, _ := proxectoCompleto(t)
	_, _ = activarProxecto(p.ID)
	doc, _ := rutaDocumentoProxecto(p.ID, "cronograma")
	d, _ := construirDocumentoTraslado("cronograma", ler(t, doc), "Cronograma 2025", "esta")
	d.Documento = json.RawMessage(`{"plans":[{"academicYear":"COPIA"}]}`)
	ruta := filepath.Join(t.TempDir(), "crono.json")
	b, _ := json.Marshal(d)
	escribir(t, ruta, string(b))
	res, err := a.aplicarMeter(ruta, OpcionsMeter{Accion: "copia"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(ler(t, doc), "COPIA") {
		t.Fatal("o orixinal non pode cambiar")
	}
	if len(res.Proxectos) != 1 || res.Proxectos[0] == p.ID || res.AbrirProxecto == "" {
		t.Fatalf("copia: %+v", res)
	}
}

func TestDocumentoDeFormularioLevaEstrutura(t *testing.T) {
	trasladarTemporal(t)
	p, f := proxectoCompleto(t)
	_, _ = activarProxecto(p.ID)
	doc, _ := rutaDocumentoProxecto(p.ID, "form:"+f.ID)
	d, err := construirDocumentoTraslado("form:"+f.ID, ler(t, doc), "", "esta")
	if err != nil {
		t.Fatal(err)
	}
	if d.Formulario == nil || d.Formulario.Nome != "Titoría" || d.Nome != "Titoría" || d.Formulario.Modo != "" {
		t.Fatalf("documento: %+v", d)
	}
	// Noutro equipo, sen o formulario: entra cun proxecto novo e o formulario.
	ruta := filepath.Join(t.TempDir(), "doc.json")
	b, _ := json.Marshal(d)
	escribir(t, ruta, string(b))
	trasladarTemporal(t)
	a := &App{settings: Settings{Instalacion: "outra"}}
	res, err := a.aplicarMeter(ruta, OpcionsMeter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Proxectos) != 1 || len(res.Formularios) != 1 {
		t.Fatalf("%+v", res)
	}
	if _, err := lerFormularioBiblioteca(res.Formularios[0]); err != nil {
		t.Fatal("o formulario tiña que entrar na biblioteca")
	}
}

func TestSacarTenDatosProducion(t *testing.T) {
	trasladarTemporal(t)
	p, f := proxectoCompleto(t)
	sel := SeleccionSacar{Tipo: "proxecto", Proxectos: []string{p.ID}, Datos: true}
	if tenDatosProducion(sel) {
		t.Fatal("en deseño non hai datos reais")
	}
	dir, _ := dirProxecto(p.ID)
	ruta := filepath.Join(dir, "formularios", f.ID+".json")
	var copia Formulario
	_ = json.Unmarshal([]byte(ler(t, ruta)), &copia)
	copia.Modo = modoProduccion
	b, _ := json.Marshal(copia)
	escribir(t, ruta, string(b))
	if !tenDatosProducion(sel) {
		t.Fatal("en produción con datos tiña que avisar")
	}
	sel.Datos = false
	if tenDatosProducion(sel) {
		t.Fatal("sen datos non hai nada que avisar")
	}
}

func TestCopiaCompletaRestaurarEDesfacer(t *testing.T) {
	trasladarTemporal(t)
	a := &App{settings: Settings{IAAPIKey: "clave-local"}}
	p, _ := proxectoCompleto(t)
	ruta := sacarA(t, SeleccionSacar{Tipo: "todo"}, Settings{IAAPIKey: "clave-local", Idioma: "es"})
	// Despois da copia: créase outro proxecto e bórrase o primeiro.
	outro, _ := crearProxecto("Despois", "", nil)
	_ = eliminarProxecto(p.ID)
	r, err := a.clasificarMeter(ruta)
	if err != nil || r.Tipo != "copia" || r.Destino != "restaurar" {
		t.Fatalf("%+v %v", r, err)
	}
	res, err := a.aplicarMeter(ruta, OpcionsMeter{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lerProxecto(p.ID); err != nil {
		t.Fatal("o proxecto da copia tiña que volver")
	}
	if _, err := lerProxecto(outro.ID); err == nil {
		t.Fatal("o estado ten que ser o da copia")
	}
	if a.settings.IAAPIKey != "clave-local" || a.settings.Idioma != "es" || !res.Reiniciar {
		t.Fatalf("opcións: %+v", a.settings)
	}
	if err := restaurarVersion(res.Version); err != nil {
		t.Fatal(err)
	}
	if _, err := lerProxecto(outro.ID); err != nil {
		t.Fatal("desfacer tiña que recuperar o estado anterior")
	}
}

func TestMeterBoveda(t *testing.T) {
	trasladarTemporal(t)
	a := &App{}
	fontes := filepath.Join(t.TempDir(), "Leis")
	escribir(t, filepath.Join(fontes, "a.pdf"), "%PDF a")
	escribir(t, filepath.Join(fontes, "sub", "b.md"), "# b")
	escribir(t, filepath.Join(fontes, "c.txt"), "non")
	// Sen proxecto aberto: proxecto novo co nome do cartafol.
	res, err := a.aplicarMeter(fontes, OpcionsMeter{})
	if err != nil {
		t.Fatal(err)
	}
	p, _ := lerProxecto(res.Proxectos[0])
	if p.Nome != "Leis" {
		t.Fatalf("proxecto: %+v", p)
	}
	raiz, _ := rutaBovedaProxecto(p.ID)
	if ler(t, filepath.Join(raiz, "pdf", "a.pdf")) != "%PDF a" || ler(t, filepath.Join(raiz, "raw", "b.md")) != "# b" {
		t.Fatal("as fontes non quedaron na bóveda")
	}
	// Co proxecto aberto: á súa bóveda, sen pisar.
	_, _ = activarProxecto(p.ID)
	res, err = a.aplicarMeter(filepath.Join(fontes, "a.pdf"), OpcionsMeter{})
	if err != nil || !res.Recargar {
		t.Fatalf("%+v %v", res, err)
	}
	if ler(t, filepath.Join(raiz, "pdf", "a (2).pdf")) != "%PDF a" {
		t.Fatal("un nome repetido entra coma «a (2).pdf»")
	}
}

func TestPlantillaLembradaPorDocumento(t *testing.T) {
	trasladarTemporal(t)
	a := &App{}
	p, _ := proxectoCompleto(t)
	_, _ = activarProxecto(p.ID)
	pl, err := a.GardarPlantilla(PlantillaLatex{Nome: "Centro", Latex: "{{CORPO}}"})
	if err != nil {
		t.Fatal(err)
	}
	if err := a.EscollerPlantillaDocumento(pl.ID, "cronograma"); err != nil {
		t.Fatal(err)
	}
	if err := a.EscollerPlantillaDocumento("", "actas"); err != nil {
		t.Fatal(err)
	}
	if id, err := a.PlantillaDoDocumento("cronograma"); err != nil || id != pl.ID {
		t.Fatalf("o cronograma lembra %q (%v)", id, err)
	}
}

func TestMeterNonPisaODocumentoDoProxectoAberto(t *testing.T) {
	trasladarTemporal(t)
	a := &App{}
	p, _ := proxectoCompleto(t) // xa ten cronograma con datos
	_, _ = activarProxecto(p.ID)
	ruta := filepath.Join(t.TempDir(), "outro.json")
	escribir(t, ruta, `{"plans":[{"academicYear":"ALLEO"}]}`)
	r, _ := a.clasificarMeter(ruta)
	if r.Destino != "novo" || r.PodeAberto {
		t.Fatalf("o aberto xa ten cronograma: %+v", r)
	}
	// Aínda que se pida o proxecto aberto, non se pisa: vai a un novo.
	res, err := a.aplicarMeter(ruta, OpcionsMeter{Destino: "aberto"})
	if err != nil {
		t.Fatal(err)
	}
	doc, _ := rutaDocumentoProxecto(p.ID, "cronograma")
	if strings.Contains(ler(t, doc), "ALLEO") || len(res.Proxectos) != 1 || res.Proxectos[0] == p.ID {
		t.Fatalf("pisou o documento do proxecto aberto: %+v", res)
	}
}
