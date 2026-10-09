package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/gif"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestNormalizarOfimatica(t *testing.T) {
	o, err := normalizarOfimatica(OfimaticaPlantilla{Fonte: ` Ca"lib<ri> `, CorPrincipal: "#1f3864", CorSecundaria: "abc", TamanoCorpo: 12, Portada: "a\r\nb"})
	if err != nil {
		t.Fatal(err)
	}
	if o.Fonte != "Calibri" || o.CorPrincipal != "1F3864" || o.CorSecundaria != "AABBCC" || o.Portada != "a\nb" {
		t.Errorf("%+v", o)
	}
	for _, mala := range []OfimaticaPlantilla{{CorPrincipal: "azul"}, {CorTexto: "12345"}, {TamanoCorpo: 40}, {TamanoCorpo: 3}} {
		if _, err := normalizarOfimatica(mala); err == nil {
			t.Errorf("%+v debería dar erro", mala)
		}
	}
}

func TestEstiloPorDefectoEPropio(t *testing.T) {
	d := estiloOfimatica(OfimaticaPlantilla{})
	if d.Fonte != "Arial" || d.Principal != "1F3864" || d.TamanoPt != 11 {
		t.Errorf("%+v", d)
	}
	e := estiloOfimatica(OfimaticaPlantilla{Fonte: "Georgia", CorPrincipal: "AA0000", TamanoCorpo: 12})
	if e.Fonte != "Georgia" || e.Principal != "AA0000" || e.Secundaria != "2E5496" || e.TamanoPt != 12 {
		t.Errorf("%+v", e)
	}
}

func TestParsearPortadaOfi(t *testing.T) {
	subst := func(s string) string { return strings.ReplaceAll(s, "{{CENTRO}}", "IES Rosalía") }
	ls := parsearPortadaOfi("@logo 40\n@imaxe cab.png 25mm\n@espazo 12\n\n# Título\n## Sub\n---\n### {{CENTRO}}\nCurso {{CENTRO}}\n@raro\n@imaxe", subst)
	quero := []linhaPortada{
		{Tipo: "logo", AltoMM: 40}, {Tipo: "imaxe", Nome: "cab.png", AltoMM: 25}, {Tipo: "espazo", AltoMM: 12},
		{Tipo: "titulo", Texto: "Título"}, {Tipo: "subtitulo", Texto: "Sub"}, {Tipo: "regra"},
		{Tipo: "organizacion", Texto: "IES Rosalía"}, {Tipo: "meta", Texto: "Curso IES Rosalía"},
	}
	if len(ls) != len(quero) {
		t.Fatalf("%d liñas: %+v", len(ls), ls)
	}
	for i := range quero {
		if ls[i] != quero[i] {
			t.Errorf("liña %d: %+v, esperaba %+v", i, ls[i], quero[i])
		}
	}
	// A por defecto leva o mesmo que a portada fixa de antes.
	def := parsearPortadaOfi(portadaOfimaticaPorDefecto, subst)
	if len(def) != 7 || def[0].Tipo != "logo" || def[1].Texto != "{{TITULO}}" {
		t.Errorf("%+v", def)
	}
}

func planeTemplateOfi(ofi OfimaticaPlantilla) *PlantillaLatex {
	return &PlantillaLatex{Nome: "proba", Latex: "{{CORPO}}", Ofimatica: ofi,
		Variables: []PlantillaVar{{Nome: "TELEFONO", Etiqueta: "Teléfono", Valor: "986 11 22 33"}}}
}

func TestDocxTomaOEstiloDaPlantilla(t *testing.T) {
	pl := planeTemplateOfi(OfimaticaPlantilla{
		Fonte: "Georgia", CorPrincipal: "AA0000", CorSecundaria: "00AA00", CorTexto: "111111", CorApagada: "222222", CorLinha: "0000AA", TamanoCorpo: 12,
		Portada:   "# {{CENTRO}}\n@espazo 20\n### Curso {{CURSO}}\nTel. {{TELEFONO}}\n---",
		Cabeceira: "{{CENTRO}} · {{DEPARTAMENTO}}", Pe: "Tel. {{TELEFONO}}",
	})
	pk, _ := docxDe(t, planMostraCrono(), pl)
	for nome, c := range pk {
		if strings.HasSuffix(nome, ".xml") || strings.HasSuffix(nome, ".rels") {
			xmlBenFormado(t, nome, c)
		}
	}
	est := pk["word/styles.xml"]
	for _, quero := range []string{`w:ascii="Georgia"`, `w:val="AA0000"`, `w:val="00AA00"`, `w:val="111111"`, `w:val="222222"`, `w:color="0000AA"`, `<w:sz w:val="24"/>`} {
		if !strings.Contains(est, quero) {
			t.Errorf("styles.xml non ten %s", quero)
		}
	}
	for _, non := range []string{"Arial", "1F3864", "2E5496", "8EAADB"} {
		if strings.Contains(est, non) {
			t.Errorf("styles.xml aínda ten o valor por defecto %s", non)
		}
	}
	doc := pk["word/document.xml"]
	// A portada é a da plantilla: título co centro, espazo, liña destacada e teléfono; sen "CRONOGRAMA ANUAL".
	i := strings.Index(doc, "IES Castelao &amp; Fillos, 100% público")
	if i < 0 || i > strings.Index(doc, "Índice") {
		t.Error("a portada non leva o centro")
	}
	for _, quero := range []string{"Curso 2025/2026", "Tel. 986 11 22 33", `w:before="1134"`} {
		if !strings.Contains(doc[:strings.Index(doc, "Índice")], quero) {
			t.Errorf("a portada non ten %q", quero)
		}
	}
	if strings.Contains(doc[:strings.Index(doc, "Índice")], "CRONOGRAMA ANUAL") {
		t.Error("a portada da plantilla substitúe á por defecto")
	}
	// Cabeceira e pé.
	if !strings.Contains(pk["word/header1.xml"], "IES Castelao &amp; Fillos, 100% público · Departamento de Orientación") {
		t.Errorf("cabeceira: %s", pk["word/header1.xml"])
	}
	if strings.Contains(pk["word/header2.xml"], "Castelao") {
		t.Error("a cabeceira da portada debe ir baleira")
	}
	if !strings.Contains(pk["word/footer1.xml"], "Tel. 986 11 22 33") || !strings.Contains(pk["word/footer1.xml"], "PAGE") {
		t.Errorf("pé: %s", pk["word/footer1.xml"])
	}
	if !strings.Contains(doc, `w:headerReference w:type="default"`) || !strings.Contains(pk["[Content_Types].xml"], "header1.xml") || !strings.Contains(pk["word/_rels/document.xml.rels"], "header1.xml") {
		t.Error("a cabeceira non está enganchada ao documento")
	}
}

func TestDocxSenCabeceiraNonLevaPartesDeCabeceira(t *testing.T) {
	pk, _ := docxDe(t, planMostraCrono(), nil)
	if _, hai := pk["word/header1.xml"]; hai || strings.Contains(pk["word/document.xml"], "headerReference") {
		t.Error("sen cabeceira na plantilla non debe haber cabeceira")
	}
}

func TestOdtTomaOEstiloDaPlantilla(t *testing.T) {
	pl := planeTemplateOfi(OfimaticaPlantilla{
		Fonte: "Georgia", CorPrincipal: "AA0000", TamanoCorpo: 12,
		Portada: "@espazo 15\n# {{CENTRO}}\nTel. {{TELEFONO}}", Cabeceira: "{{CENTRO}}", Pe: "Tel. {{TELEFONO}}",
	})
	pk, _ := odtDe(t, planMostraCrono(), pl)
	for nome, c := range pk {
		if strings.HasSuffix(nome, ".xml") {
			xmlBenFormado(t, nome, c)
		}
	}
	est := pk["styles.xml"]
	for _, quero := range []string{"Georgia", "#AA0000", `fo:font-size="12.0pt"`, "<style:header>", "Tel. 986 11 22 33  ·  <text:page-number"} {
		if !strings.Contains(est, quero) {
			t.Errorf("styles.xml non ten %q", quero)
		}
	}
	if strings.Contains(est, "Arial") || strings.Contains(est, "1F3864") {
		t.Error("aínda hai valores por defecto")
	}
	// A portada: o primeiro parágrafo é o espazo (mestra da portada) e despois o título co centro.
	c := pk["content.xml"]
	if !strings.Contains(c, `text:style-name="Gap0F"`) || strings.Contains(c, "CRONOGRAMA ANUAL") || !strings.Contains(c, "Tel. 986 11 22 33") {
		t.Errorf("portada: %.500s", c)
	}
	// Todos os estilos usados están definidos.
	definidos := map[string]bool{}
	for _, m := range regexpEstiloOdt.FindAllStringSubmatch(est+c, -1) {
		definidos[m[1]] = true
	}
	for _, m := range regexpUsoOdt.FindAllStringSubmatch(c, -1) {
		if !definidos[m[1]] {
			t.Errorf("estilo usado e non definido: %s", m[1])
		}
	}
}

func TestPortadaConImaxeDaPlantillaEmDocxEOdt(t *testing.T) {
	aillarAdxuntos(t)
	a := &App{}
	est, _ := a.ListarPlantillas()
	propia, _ := a.DuplicarPlantilla(est.Plantillas[0].ID)
	dir, _ := plantillaImaxesDir(propia.ID)
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "cabeceira.png"), pngProba(300, 100, color.NRGBA{R: 0, G: 100, B: 200, A: 255}), 0o644)
	propia.Ofimatica = OfimaticaPlantilla{Portada: "@imaxe cabeceira.png 30\n@imaxe non-existe.png\n# Ola"}
	d, err := prepararOfimatica(XerarInformeCronogramaRequest{Plan: planMostraCrono()}, &propia, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if d.ImaxesPlantilla["cabeceira.png"] == nil || d.ImaxesPlantilla["non-existe.png"] != nil {
		t.Fatalf("imaxes: %+v", d.ImaxesPlantilla)
	}
	if !strings.Contains(strings.Join(d.Avisos, "|"), "non-existe.png") {
		t.Errorf("falta o aviso da imaxe que non existe: %v", d.Avisos)
	}
	b, err := escribirDocxCrono(&d)
	if err != nil {
		t.Fatal(err)
	}
	pk := lerZip(t, b)
	n := 0
	for nome := range pk {
		if strings.HasPrefix(nome, "word/media/") {
			n++
		}
	}
	if n != 3 { // imaxe da portada + cronograma anual + imaxe da tarefa
		t.Errorf("imaxes no docx = %d", n)
	}
	// O alto respectou os 30 mm (1.080.000 EMU).
	if !strings.Contains(pk["word/document.xml"], `cy="1080000"`) {
		t.Error("a imaxe da portada non tomou o alto de 30 mm")
	}
	bo, err := escribirOdtCrono(&d)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(lerZip(t, bo)["content.xml"], `svg:height="3.00cm"`) {
		t.Error("a imaxe da portada non tomou o alto de 30 mm no ODT")
	}
}

func TestGardarPlantillaValidaOfimaticaEConservaAsImaxes(t *testing.T) {
	aillarAdxuntos(t)
	a := &App{}
	p, err := a.GardarPlantilla(PlantillaLatex{Nome: "Con Word", Latex: "{{CORPO}}", Ofimatica: OfimaticaPlantilla{CorPrincipal: "#abc123", Fonte: "Calibri"}})
	if err != nil || p.Ofimatica.CorPrincipal != "ABC123" {
		t.Fatalf("%v %+v", err, p.Ofimatica)
	}
	if _, err := a.GardarPlantilla(PlantillaLatex{Nome: "Mala", Latex: "{{CORPO}}", Ofimatica: OfimaticaPlantilla{CorPrincipal: "verde"}}); err == nil {
		t.Error("unha cor mala debería rexeitarse ao gardar")
	}
	// Exportar e importar leva o bloque.
	b, err := construirBundlePlantilla(p)
	if err != nil || b.Plantilla.Ofimatica.Fonte != "Calibri" {
		t.Fatalf("bundle: %v %+v", err, b.Plantilla.Ofimatica)
	}
	// Está na lista de campos que marcan unha plantilla de exemplo coma editada.
	orix := p
	orix.Ofimatica.Pe = "outro"
	if !depositoPlantillas.FabricaFoiEditada(orix, p) {
		t.Error("cambiar Ofimatica conta coma edición")
	}
}

func TestPlantillaGardarPorFerramentaMezclaOFimaticaParcial(t *testing.T) {
	aillarAdxuntos(t)
	a := &App{}
	p, _ := a.GardarPlantilla(PlantillaLatex{Nome: "Propia", Latex: "{{CORPO}}", Ofimatica: OfimaticaPlantilla{Fonte: "Calibri", CorPrincipal: "112233", Pe: "Pé antigo"}})
	r := chamarFerr(a, "plantilla_gardar", map[string]any{"id": p.ID, "ofimatica": map[string]any{"pe": "Pé novo", "tamanoCorpo": 12}})
	if r.Erro {
		t.Fatal(r.Contido)
	}
	q, _ := plantillaPorID(a, p.ID)
	if q.Ofimatica.Pe != "Pé novo" || q.Ofimatica.TamanoCorpo != 12 || q.Ofimatica.Fonte != "Calibri" || q.Ofimatica.CorPrincipal != "112233" {
		t.Errorf("só se pisan os campos indicados: %+v", q.Ofimatica)
	}
	if r := chamarFerr(a, "plantilla_gardar", map[string]any{"id": p.ID, "ofimatica": map[string]any{"corPrincipal": "morado"}}); !r.Erro {
		t.Error("cor inválida")
	}
	// plantilla_ler devolve o bloque.
	if r := chamarFerr(a, "plantilla_ler", map[string]any{"id": p.ID}); !strings.Contains(r.Contido, `"ofimatica"`) || !strings.Contains(r.Contido, "Calibri") {
		t.Errorf("plantilla_ler: %s", r.Contido)
	}
	// O esquema da ferramenta expón o bloque.
	for _, f := range a.IAFerramentas() {
		if f.Nome == "plantilla_gardar" {
			var e map[string]any
			json.Unmarshal([]byte(f.Parametros), &e)
			if e["properties"].(map[string]any)["ofimatica"] == nil {
				t.Error("plantilla_gardar non declara ofimatica")
			}
		}
	}
}

var (
	regexpEstiloOdt = regexp.MustCompile(`<style:style style:name="([^"]+)"`)
	regexpUsoOdt    = regexp.MustCompile(`text:style-name="([^"]+)"`)
)

// TestMostraOfimaticaPersonalizada escribe un docx/odt cunha plantilla
// personalizada para velos a ollo (YANG_INFORME_OUT=<cartafol>).
func TestMostraOfimaticaPersonalizada(t *testing.T) {
	dir := os.Getenv("YANG_INFORME_OUT")
	if dir == "" {
		t.Skip("defínese YANG_INFORME_OUT para escribir a mostra")
	}
	aillarAdxuntos(t)
	a := &App{}
	est, _ := a.ListarPlantillas()
	propia, _ := a.DuplicarPlantilla(est.Plantillas[0].ID)
	imDir, _ := plantillaImaxesDir(propia.ID)
	os.MkdirAll(imDir, 0o755)
	os.WriteFile(filepath.Join(imDir, "banda.png"), pngProba(900, 120, color.NRGBA{R: 122, G: 31, B: 52, A: 255}), 0o644)
	propia.Variables = append(propia.Variables, PlantillaVar{Nome: "TELEFONO", Valor: "986 11 22 33"})
	propia.Ofimatica = OfimaticaPlantilla{
		Fonte: "Georgia", CorPrincipal: "7A1F34", CorSecundaria: "A34A5E", CorLinha: "D9A7B3", TamanoCorpo: 11.5,
		Portada:   "@imaxe banda.png 18\n@espazo 30\n# {{CENTRO}}\n## Cronograma anual de {{DEPARTAMENTO}}\n---\n### Curso {{CURSO}}\n{{LOCALIDADE}} · Tel. {{TELEFONO}}\n@espazo 10\n@logo 28",
		Cabeceira: "@imaxe banda.png 8\n@imaxe logo.pdf 8\n{{CENTRO}} · {{DEPARTAMENTO}}", Pe: "@imaxe banda.png 4\nCurso {{CURSO}}",
	}
	os.WriteFile(filepath.Join(imDir, "logo.pdf"), []byte(pdfProba), 0o644)
	req := XerarInformeCronogramaRequest{Plan: planMostraCrono()}
	for _, f := range []string{"docx", "odt"} {
		if _, err := exportarInformeOfimaticaCrono(req, &propia, filepath.Join(dir, "personalizada."+f), f); err != nil {
			t.Fatal(err)
		}
	}
}

// ---------- Subir imaxes dende o editor de plantillas ----------

func subirProba(t *testing.T, a *App, id, nome string, datos []byte) (string, error) {
	t.Helper()
	return a.SubirImaxePlantilla(SubirImaxePlantillaRequest{ID: id, FileName: nome, DataB64: base64.StdEncoding.EncodeToString(datos)})
}

func TestEditorSubeUnPDFVectorialTalCal(t *testing.T) {
	aillarAdxuntos(t)
	a := &App{}
	p, _ := a.GardarPlantilla(PlantillaLatex{Nome: "Con PDF", Latex: "{{CORPO}}"})
	nome, err := subirProba(t, a, p.ID, "Logo do Centro (vectorial).pdf", []byte(pdfProba))
	if err != nil || nome != "Logo-do-Centro-vectorial.pdf" {
		t.Fatalf("%q %v", nome, err)
	}
	dir, _ := plantillaImaxesDir(p.ID)
	b, _ := os.ReadFile(filepath.Join(dir, nome))
	if string(b) != pdfProba {
		t.Error("o PDF non debe modificarse: é vectorial")
	}
	// Un PDF sen extensión .pdf tamén se recoñece polo contido.
	if n, err := subirProba(t, a, p.ID, "logo-sen-extension", []byte(pdfProba)); err != nil || !strings.HasSuffix(n, ".pdf") {
		t.Errorf("%q %v", n, err)
	}
	if l := imaxesDaPlantilla(p.ID); len(l) != 2 {
		t.Errorf("imaxes: %v", l)
	}
}

func TestEditorConverteOsFormatosQueLaTeXNonLe(t *testing.T) {
	aillarAdxuntos(t)
	a := &App{}
	p, _ := a.GardarPlantilla(PlantillaLatex{Nome: "Formatos", Latex: "{{CORPO}}"})
	var g bytes.Buffer
	gif.Encode(&g, image.NewPaletted(image.Rect(0, 0, 8, 8), color.Palette{color.Black, color.White}), nil)
	nome, err := subirProba(t, a, p.ID, "animado.gif", g.Bytes())
	if err != nil || nome != "animado.png" {
		t.Fatalf("o GIF debe pasar a PNG: %q %v", nome, err)
	}
	dir, _ := plantillaImaxesDir(p.ID)
	if b, _ := os.ReadFile(filepath.Join(dir, nome)); !bytes.HasPrefix(b, []byte("\x89PNG")) {
		t.Error("non é un PNG")
	}
	// PNG e JPEG quedan tal cales (e co nome seguro).
	png := pngProba(10, 10, color.NRGBA{A: 255})
	if n, _ := subirProba(t, a, p.ID, "Meu Logo.PNG", png); n != "Meu-Logo.png" {
		t.Errorf("nome: %q", n)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "Meu-Logo.png")); !bytes.Equal(b, png) {
		t.Error("un PNG non se debe recodificar")
	}
	if n, _ := subirProba(t, a, p.ID, "foto.jpeg", jpegProba(20, 10)); n != "foto.jpg" {
		t.Errorf("o JPEG queda coma .jpg: %q", n)
	}
	// SVG con Inkscape/ImageMagick, se hai.
	if commandExists("inkscape") || comandoImageMagick() != "" {
		svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="80" height="40"><rect width="80" height="40" fill="#06c"/></svg>`)
		if n, err := subirProba(t, a, p.ID, "logo.svg", svg); err != nil || n != "logo.png" {
			t.Errorf("SVG: %q %v", n, err)
		}
	}
	// Un RAW sen conversor dá unha mensaxe clara; e non deixa nada no cartafol.
	if _, err := subirProba(t, a, p.ID, "foto.nef", []byte("II*\x00 non é un RAW")); err != nil && !strings.Contains(err.Error(), "NEF") {
		t.Errorf("a mensaxe debe nomear o formato: %v", err)
	}
	// O que non é imaxe (unha fonte, un .sty) vai tal cal, co nome orixinal.
	if n, err := subirProba(t, a, p.ID, "miña fonte.ttf", []byte("\x00\x01\x00\x00fonte")); err != nil || n != "miña fonte.ttf" {
		t.Errorf("ficheiro non imaxe: %q %v", n, err)
	}
	if _, err := subirProba(t, a, p.ID, "baleiro.png", nil); err == nil {
		t.Error("un ficheiro baleiro debe rexeitarse")
	}
	if _, err := subirProba(t, a, "", "a.png", png); err == nil {
		t.Error("sen plantilla gardada non hai onde subir")
	}
}

func TestEditorMiniaturaDePDFEImaxe(t *testing.T) {
	aillarAdxuntos(t)
	a := &App{}
	p, _ := a.GardarPlantilla(PlantillaLatex{Nome: "Mini", Latex: "{{CORPO}}"})
	subirProba(t, a, p.ID, "a.png", pngProba(600, 200, color.NRGBA{R: 200, A: 255}))
	subirProba(t, a, p.ID, "fonte.ttf", []byte("x"))
	v, err := a.PlantillaImaxeVista(p.ID, "a.png")
	if err != nil || !strings.HasPrefix(v, "data:image/") {
		t.Fatalf("%v", err)
	}
	if raw, _ := decodificarDataURICrono(v); func() int { c, _, _ := image.DecodeConfig(bytes.NewReader(raw)); return c.Width }() != 160 {
		t.Error("a miniatura debe reducirse a 160 px")
	}
	if v, _ := a.PlantillaImaxeVista(p.ID, "fonte.ttf"); v != "" {
		t.Errorf("un ficheiro que non é imaxe non ten miniatura: %.30s", v)
	}
	if commandExists("pdftoppm") {
		subirProba(t, a, p.ID, "logo.pdf", []byte(pdfProba))
		if v, err := a.PlantillaImaxeVista(p.ID, "logo.pdf"); err != nil || !strings.HasPrefix(v, "data:image/") {
			t.Errorf("miniatura do PDF: %v", err)
		}
	}
	// Un nome con ../ non sae do cartafol da plantilla.
	if _, err := a.PlantillaImaxeVista(p.ID, "../../plantillas.json"); err == nil {
		t.Error("non debe ler fóra do cartafol")
	}
}

func TestInterlinadoEMarxesDaPlantillaEnDocxEOdt(t *testing.T) {
	pl := planeTemplateOfi(OfimaticaPlantilla{Interlinado: 1.5, MarxeSuperior: 3, MarxeInferior: 2, MarxeEsquerda: 3.5, MarxeDereita: 1.5})
	pk, _ := docxDe(t, planMostraCrono(), pl)
	if est := pk["word/styles.xml"]; !strings.Contains(est, `w:line="360" w:lineRule="auto"/></w:pPr></w:pPrDefault>`) {
		t.Error("o interliñado 1.5 non chegou ao estilo por defecto do docx")
	}
	if doc := pk["word/document.xml"]; !strings.Contains(doc, `w:top="1701" w:right="851" w:bottom="1134" w:left="1985"`) {
		t.Errorf("marxes do docx: %s", doc[strings.LastIndex(doc, "<w:sectPr"):])
	}
	po, _ := odtDe(t, planMostraCrono(), pl)
	est := po["styles.xml"]
	if !strings.Contains(est, `fo:line-height="150%"`) || !strings.Contains(est, `fo:margin-top="3.00cm" fo:margin-bottom="2.00cm" fo:margin-left="3.50cm" fo:margin-right="1.50cm"`) {
		t.Error("interliñado ou marxes do odt non aplicados")
	}
	// Por defecto todo queda igual ca antes.
	pk2, _ := docxDe(t, planMostraCrono(), planeTemplateOfi(OfimaticaPlantilla{}))
	if !strings.Contains(pk2["word/document.xml"], `w:top="1418" w:right="1418" w:bottom="1418" w:left="1418"`) || !strings.Contains(pk2["word/styles.xml"], `w:line="276"`) {
		t.Error("o valor por defecto cambiou")
	}
	if _, err := normalizarOfimatica(OfimaticaPlantilla{Interlinado: 5}); err == nil {
		t.Error("interliñado fóra de rango aceptado")
	}
	if _, err := normalizarOfimatica(OfimaticaPlantilla{MarxeEsquerda: 20}); err == nil {
		t.Error("marxe fóra de rango aceptada")
	}
}

func TestPrevisualizarPlantillaOfimaticaDevolvePaxinas(t *testing.T) {
	if comandoLibreOffice() == "" || !commandExists("pdftoppm") {
		t.Skip("precisa LibreOffice e pdftoppm")
	}
	a := &App{}
	for _, formato := range []string{"docx", "odt"} {
		r, err := a.PrevisualizarPlantillaOfimatica(PlantillaLatex{Nome: "p", Latex: "{{CORPO}}", Ofimatica: OfimaticaPlantilla{Interlinado: 1.5, MarxeEsquerda: 4}}, formato)
		if err != nil {
			t.Fatalf("%s: %v", formato, err)
		}
		if len(r.PageImages) < 2 || len(r.PageImages) > 6 || !strings.HasPrefix(r.PageImages[0], "data:image/png;base64,") {
			t.Errorf("%s: %d páxinas", formato, len(r.PageImages))
		}
	}
	if _, err := a.PrevisualizarPlantillaOfimatica(PlantillaLatex{Ofimatica: OfimaticaPlantilla{Interlinado: 9}}, "docx"); err == nil {
		t.Error("aceptou un interliñado inválido")
	}
}
