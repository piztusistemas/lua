package main

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"regexp"
	"strings"
	"testing"
	"time"
)

func lerZip(t *testing.T, datos []byte) map[string]string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(datos), int64(len(datos)))
	if err != nil {
		t.Fatalf("non é un ZIP válido: %v", err)
	}
	out := map[string]string{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(rc)
		rc.Close()
		out[f.Name] = string(b)
	}
	return out
}

func xmlBenFormado(t *testing.T, nome, contido string) {
	t.Helper()
	d := xml.NewDecoder(strings.NewReader(contido))
	for {
		if _, err := d.Token(); err != nil {
			if err == io.EOF {
				return
			}
			t.Fatalf("%s: XML mal formado: %v", nome, err)
		}
	}
}

func pedidoOfimatica(p PlanCrono) XerarInformeCronogramaRequest {
	return XerarInformeCronogramaRequest{Plan: p, NomeBase: "cronograma-proba"}
}

func docxDe(t *testing.T, p PlanCrono, plantilla *PlantillaLatex) (map[string]string, datosOfimatica) {
	t.Helper()
	d, err := prepararOfimatica(pedidoOfimatica(p), plantilla, time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	b, err := escribirDocxCrono(&d)
	if err != nil {
		t.Fatal(err)
	}
	return lerZip(t, b), d
}

func odtDe(t *testing.T, p PlanCrono, plantilla *PlantillaLatex) (map[string]string, []byte) {
	t.Helper()
	d, err := prepararOfimatica(pedidoOfimatica(p), plantilla, time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	b, err := escribirOdtCrono(&d)
	if err != nil {
		t.Fatal(err)
	}
	return lerZip(t, b), b
}

var reEstiloDocx = regexp.MustCompile(`<w:pStyle w:val="([^"]+)"/>`)
var reIdEstiloDocx = regexp.MustCompile(`<w:style w:type="paragraph"[^>]*w:styleId="([^"]+)"`)

func TestDocxNativoTemEstilosEXmlValido(t *testing.T) {
	pk, _ := docxDe(t, planMostraCrono(), nil)
	for _, obrigatorio := range []string{"[Content_Types].xml", "_rels/.rels", "word/document.xml", "word/styles.xml", "word/_rels/document.xml.rels", "word/footer1.xml"} {
		if _, ok := pk[obrigatorio]; !ok {
			t.Errorf("falta %s no paquete", obrigatorio)
		}
	}
	for nome, c := range pk {
		if strings.HasSuffix(nome, ".xml") || strings.HasSuffix(nome, ".rels") {
			xmlBenFormado(t, nome, c)
		}
	}
	// Todo estilo que o documento usa está definido na folla de estilos: este é
	// o punto do cambio (un documento con estilos, non sen eles).
	definidos := map[string]bool{}
	for _, m := range reIdEstiloDocx.FindAllStringSubmatch(pk["word/styles.xml"], -1) {
		definidos[m[1]] = true
	}
	usados := map[string]bool{}
	for _, m := range reEstiloDocx.FindAllStringSubmatch(pk["word/document.xml"]+pk["word/footer1.xml"], -1) {
		usados[m[1]] = true
	}
	for e := range usados {
		if !definidos[e] {
			t.Errorf("o documento usa o estilo %q sen definilo", e)
		}
	}
	for _, e := range []string{"Title", "Heading1", "Heading2", "BodyText", "ListBullet", "Quote", "TaskTitle", "SignName"} {
		if !usados[e] {
			t.Errorf("o informe de mostra debería usar o estilo %s", e)
		}
	}
	// Sen formato directo de fonte/cor nos parágrafos de corpo: manda o estilo.
	if strings.Contains(pk["word/document.xml"], `w:rFonts`) {
		t.Error("o documento non debería levar fontes en liña: as fontes van nos estilos")
	}
	// Imaxes: logotipo, cronograma anual e a imaxe da tarefa.
	n := 0
	for nome := range pk {
		if strings.HasPrefix(nome, "word/media/") {
			n++
		}
	}
	if n != 3 {
		t.Errorf("esperaba 3 imaxes (logotipo, cronograma, tarefa), hai %d", n)
	}
	if !strings.Contains(pk["[Content_Types].xml"], `Extension="jpg"`) || !strings.Contains(pk["[Content_Types].xml"], `Extension="png"`) {
		t.Error("faltan os tipos de contido de imaxe")
	}
}

func TestDocxTextoEscapadoEContido(t *testing.T) {
	pk, d := docxDe(t, planMostraCrono(), nil)
	doc := pk["word/document.xml"]
	for _, quero := range []string{"Cronograma anual", "IES Castelao &amp; Fillos, 100% público", "María Pérez Souto", "Avaliación inicial", "18 de setembro de 2026", "Setembro", "Outubro", "Marzo"} {
		if !strings.Contains(doc, quero) {
			t.Errorf("falta %q no documento", quero)
		}
	}
	// A negriña e a cursiva da descrición chegan coma formato de texto.
	if !strings.Contains(doc, "<w:b/>") || !strings.Contains(doc, "<w:i/>") {
		t.Error("a negriña/cursiva da descrición non se converteu")
	}
	// A URL da ligazón queda visible no papel.
	if !strings.Contains(doc, "https://www.edu.xunta.gal/portal/?a=1&amp;b=2#x") {
		t.Error("a URL da ligazón non aparece")
	}
	// Tamaño de letra grande: 11 pt x 1,3 = 14,3 -> 29 medios-puntos.
	if !strings.Contains(doc, `<w:sz w:val="29"/>`) {
		t.Error("o tamaño de letra [fs=gr] non se aplicou")
	}
	if len(d.ConTarefas) != 3 {
		t.Errorf("meses con tarefas = %d", len(d.ConTarefas))
	}
	// Saltos de páxina antes de cada sección (índice, introdución, cronograma, meses, sinatura).
	if got := strings.Count(doc, "<w:pageBreakBefore/>"); got != 5 {
		t.Errorf("saltos de páxina = %d, esperaba 5", got)
	}
}

func TestDocxCursoBaleiroNonFalla(t *testing.T) {
	p := PlanCrono{AcademicYear: "2025/2026", Config: ConfigCrono{StartDate: "2025-09-08", EndDate: "2026-06-19"}, MonthsData: map[string]MesCrono{}}
	pk, _ := docxDe(t, p, nil)
	xmlBenFormado(t, "document.xml", pk["word/document.xml"])
	if !strings.Contains(pk["word/document.xml"], "Aínda non hai tarefas") {
		t.Error("un curso sen tarefas debería dicilo")
	}
	// Sen datas válidas tamén ten que xerar algo.
	p.Config.StartDate = ""
	pk, _ = docxDe(t, p, nil)
	xmlBenFormado(t, "document.xml", pk["word/document.xml"])
}

func TestDocxUsaAIntroducionDaPlantillaEAsVariables(t *testing.T) {
	pl := &PlantillaLatex{Introducion: "Programación de {{CENTRO}} para o curso {{CURSO}}."}
	pk, _ := docxDe(t, planMostraCrono(), pl)
	if !strings.Contains(pk["word/document.xml"], "Programación de IES Castelao &amp; Fillos, 100% público para o curso 2025/2026.") {
		t.Errorf("a introdución da plantilla non se aplicou:\n%s", pk["word/document.xml"][:600])
	}
	// A do propio curso manda sobre a da plantilla.
	p := planMostraCrono()
	p.Config.IntroText = "Introdución propia do curso."
	pk, _ = docxDe(t, p, pl)
	if !strings.Contains(pk["word/document.xml"], "Introdución propia do curso.") || strings.Contains(pk["word/document.xml"], "Programación de") {
		t.Error("a introdución do curso debería gañar á da plantilla")
	}
}

func TestDocxIntroducionEnVariosParagrafos(t *testing.T) {
	p := planMostraCrono()
	p.Config.IntroText = "Primeiro.\n\nSegundo parágrafo.\nContinúa."
	pk, _ := docxDe(t, p, nil)
	if !strings.Contains(pk["word/document.xml"], ">Primeiro.<") || !strings.Contains(pk["word/document.xml"], ">Segundo parágrafo. Continúa.<") {
		t.Error("os parágrafos da introdución non se separaron")
	}
}

func TestDocxImaxeJPEGConservaFormato(t *testing.T) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 40, 20)), nil); err != nil {
		t.Fatal(err)
	}
	p := planMostraCrono()
	p.Config.SchoolLogo = "data:image/jpeg;base64," + b64(buf.Bytes())
	pk, _ := docxDe(t, p, nil)
	achado := false
	for nome, c := range pk {
		if strings.HasPrefix(nome, "word/media/") && strings.HasSuffix(nome, ".jpg") && strings.HasPrefix(c, "\xff\xd8") {
			achado = true
		}
	}
	if !achado {
		t.Error("o logotipo JPEG debería ir tal cal (sen pasar a PNG)")
	}
}

func TestImaxeRotaOuInexistenteDaAvisoENonFalla(t *testing.T) {
	p := planMostraCrono()
	p.Config.SchoolLogo = "data:image/png;base64,AAAA"
	m := p.MonthsData["2025-09"]
	m.Tasks[0].Description = "Texto\n\n![x](img:non-existe)\n\n![y](data:image/png;base64,!!!)"
	p.MonthsData["2025-09"] = m
	d, err := prepararOfimatica(pedidoOfimatica(p), nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	b, err := escribirDocxCrono(&d)
	if err != nil {
		t.Fatal(err)
	}
	xmlBenFormado(t, "document.xml", lerZip(t, b)["word/document.xml"])
	todos := strings.Join(d.Avisos, "\n")
	for _, quero := range []string{"logotipo", "non-existe"} {
		if !strings.Contains(todos, quero) {
			t.Errorf("falta o aviso sobre %q: %v", quero, d.Avisos)
		}
	}
}

func TestOdtNativoEstruturaEEstilos(t *testing.T) {
	pk, bruto := odtDe(t, planMostraCrono(), nil)
	// O mimetype ten que ir primeiro e sen comprimir.
	zr, _ := zip.NewReader(bytes.NewReader(bruto), int64(len(bruto)))
	if zr.File[0].Name != "mimetype" || zr.File[0].Method != zip.Store {
		t.Errorf("o primeiro ficheiro debe ser mimetype sen comprimir: %s %d", zr.File[0].Name, zr.File[0].Method)
	}
	if pk["mimetype"] != "application/vnd.oasis.opendocument.text" {
		t.Errorf("mimetype = %q", pk["mimetype"])
	}
	for nome, c := range pk {
		if strings.HasSuffix(nome, ".xml") {
			xmlBenFormado(t, nome, c)
		}
	}
	definidos := map[string]bool{}
	for _, m := range regexp.MustCompile(`<style:style style:name="([^"]+)"`).FindAllStringSubmatch(pk["styles.xml"]+pk["content.xml"], -1) {
		definidos[m[1]] = true
	}
	for _, m := range regexp.MustCompile(`text:style-name="([^"]+)"`).FindAllStringSubmatch(pk["content.xml"], -1) {
		if !definidos[m[1]] {
			t.Errorf("content.xml usa o estilo %q sen definilo", m[1])
		}
	}
	for _, e := range []string{"Title", "TitleFirst", "Heading1", "TextBody", "Quote", "ListItem", "FS_gr", "TextBodyCenter"} {
		if !definidos[e] {
			t.Errorf("falta o estilo %s", e)
		}
	}
	// Imaxes no manifesto co seu tipo.
	man := pk["META-INF/manifest.xml"]
	for nome := range pk {
		if strings.HasPrefix(nome, "Pictures/") && !strings.Contains(man, nome) {
			t.Errorf("a imaxe %s non está no manifesto", nome)
		}
	}
	if !strings.Contains(man, "image/png") {
		t.Error("o manifesto debe indicar image/png")
	}
	if !strings.Contains(pk["styles.xml"], "text:page-number") {
		t.Error("o pé de páxina debe levar o número")
	}
}

func TestOdtSenLogotipoUsaTitleFirst(t *testing.T) {
	p := planMostraCrono()
	p.Config.SchoolLogo = ""
	pk, _ := odtDe(t, p, nil)
	if !strings.Contains(pk["content.xml"], `text:style-name="TitleFirst"`) {
		t.Error("sen logotipo o título ten que levar a páxina mestra da portada")
	}
	pk, _ = odtDe(t, planMostraCrono(), nil)
	if !strings.Contains(pk["content.xml"], `text:style-name="FigureCover"`) {
		t.Error("co logotipo, a figura leva a páxina mestra da portada")
	}
}

func TestEscaparXMLQuitaControlesInvalidos(t *testing.T) {
	if got := escaparXMLOfi("a\x00b\x08c<&>\"'\t\n"); got != "abc&lt;&amp;&gt;&quot;&apos;\t\n" {
		t.Errorf("escaparXMLOfi = %q", got)
	}
}

func TestPngDoCronogramaTenAsCoresEUnAltoRazoable(t *testing.T) {
	meses := calendarioCrono(planMostraCrono().Config, planMostraCrono().MonthsData)
	png, err := renderCronogramaPNGCrono(planMostraCrono(), meses, novosTextosDoc(nil, nil))
	if err != nil {
		t.Fatal(err)
	}
	im, err := imaxeOfiDeBytes(png)
	if err != nil || im.W != 1240 || im.H < 400 || im.H > 2500 {
		t.Fatalf("imaxe inesperada: %v %+v", err, im)
	}
	img, _, _ := image.Decode(bytes.NewReader(png))
	// A banda de setembro (mes 8) ten a cor propia do mes, non o branco do fondo.
	c := color.NRGBAModel.Convert(img.At(60, 90)).(color.NRGBA)
	if c == (color.NRGBA{255, 255, 255, 255}) {
		t.Error("a banda do mes non está coloreada")
	}
}

func TestExportarOfimaticaEscribeFicheiroAtomico(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{"docx", "odt"} {
		res, err := exportarInformeOfimaticaCrono(pedidoOfimatica(planMostraCrono()), nil, dir+"/saida."+f, f)
		if err != nil || res.Path == "" {
			t.Fatalf("%s: %v", f, err)
		}
	}
	if _, err := exportarInformeOfimaticaCrono(pedidoOfimatica(planMostraCrono()), nil, dir+"/x.rtf", "rtf"); err == nil {
		t.Error("un formato descoñecido debería dar erro")
	}
}

func b64(b []byte) string { return base64.StdEncoding.EncodeToString(b) }

// TestOdtMimetypeSegundoODF: ODF esixe que 'mimetype' vaia primeiro, sen
// comprimir e SEN descritor de datos (bit 3); LibreOffice 26 rexeita o
// paquete se non. As datas teñen que ser válidas (mes 1-12).
func TestOdtMimetypeSegundoODF(t *testing.T) {
	datos, _, err := datosOfimaticaDe(pedidoOfiFormulario(t, formularioMostra(), datosMostra(t)), nil, "odt")
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(datos), int64(len(datos)))
	if err != nil {
		t.Fatal(err)
	}
	m := zr.File[0]
	if m.Name != "mimetype" || m.Method != zip.Store {
		t.Fatalf("primeira entrada = %q (método %d), esperaba mimetype sen comprimir", m.Name, m.Method)
	}
	if m.Flags&0x8 != 0 {
		t.Errorf("mimetype leva descritor de datos (flags %#x)", m.Flags)
	}
	if !bytes.HasPrefix(datos[30:], []byte("mimetype")) || !bytes.HasPrefix(datos[38:], []byte("application/vnd.oasis.opendocument.text")) {
		t.Errorf("o tipo MIME non está en claro no desprazamento 38")
	}
	for _, f := range zr.File {
		if f.Modified.Year() < 1980 || f.ModifiedDate>>5&0xF == 0 {
			t.Errorf("%s: data ZIP non válida (%v)", f.Name, f.Modified)
		}
	}
}
