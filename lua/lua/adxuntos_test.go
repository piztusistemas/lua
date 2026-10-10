package main

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// aillarAdxuntos manda o almacén (e a configuración) a cartafoles temporais.
func aillarAdxuntos(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	adxuntos.mu.Lock()
	adxuntos.textos = map[string]string{}
	adxuntos.pdfs = map[string]string{}
	adxuntos.mu.Unlock()
}

func pngProba(w, h int, c color.NRGBA) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	var b bytes.Buffer
	png.Encode(&b, img)
	return b.Bytes()
}

func jpegProba(w, h int) []byte {
	var b bytes.Buffer
	jpeg.Encode(&b, image.NewRGBA(image.Rect(0, 0, w, h)), nil)
	return b.Bytes()
}

func zipProba(t *testing.T, ficheiros map[string]string) []byte {
	t.Helper()
	var b bytes.Buffer
	zw := zip.NewWriter(&b)
	for n, c := range ficheiros {
		w, _ := zw.Create(n)
		w.Write([]byte(c))
	}
	zw.Close()
	return b.Bytes()
}

func gardarProba(t *testing.T, nome string, datos []byte) Adxunto {
	t.Helper()
	ax, err := (&App{}).AdxuntoGardar(nome, base64.StdEncoding.EncodeToString(datos))
	if err != nil {
		t.Fatalf("AdxuntoGardar(%s): %v", nome, err)
	}
	return ax
}

func TestDetectarTipoAdxunto(t *testing.T) {
	docx := zipProba(t, map[string]string{"word/document.xml": "<x/>"})
	casos := []struct {
		nome  string
		datos []byte
		tipo  string
	}{
		{"a.png", pngProba(2, 2, color.NRGBA{A: 255}), tipoAdxImaxe},
		{"sen-extension", jpegProba(4, 4), tipoAdxImaxe},
		{"x.pdf", []byte("%PDF-1.4 ..."), tipoAdxPDF},
		{"a.docx", docx, tipoAdxDocx},
		{"sen-extension", docx, tipoAdxDocx},
		{"a.odt", []byte("PK\x03\x04..."), tipoAdxOdt},
		{"a.pptx", []byte("PK\x03\x04..."), tipoAdxPres},
		{"a.xlsx", []byte("PK\x03\x04..."), tipoAdxFolla},
		{"a.ods", []byte("PK\x03\x04..."), tipoAdxFolla},
		{"a.rtf", []byte("{\\rtf1 hola}"), tipoAdxRtf},
		{"a.doc", []byte("\xd0\xcf\x11\xe0\xa1\xb1"), tipoAdxAntigo},
		{"a.csv", []byte("a;b\n1;2"), tipoAdxTexto},
		{"notas", []byte("texto sen extensión"), tipoAdxTexto},
		{"binario.bin", []byte("\x00\x01\x02\x00"), tipoAdxOutro},
	}
	for _, c := range casos {
		if got, _ := detectarTipoAdxunto(c.nome, c.datos); got != c.tipo {
			t.Errorf("detectarTipoAdxunto(%q) = %s, esperaba %s", c.nome, got, c.tipo)
		}
	}
}

func TestAlmacenAdxuntosGardarListarEliminar(t *testing.T) {
	aillarAdxuntos(t)
	a := &App{}
	ax := gardarProba(t, "../../logo centro.png", pngProba(120, 60, color.NRGBA{R: 200, A: 255}))
	if ax.Nome != "logo centro.png" || ax.Tipo != tipoAdxImaxe || ax.Ancho != 120 || ax.Alto != 60 {
		t.Errorf("metadatos: %+v", ax)
	}
	if !idAdxuntoValido(ax.ID) {
		t.Errorf("id non válido: %q", ax.ID)
	}
	gardarProba(t, "notas.txt", []byte("hola"))
	if l := a.AdxuntosListar(); len(l) != 2 || l[0].ID != ax.ID {
		t.Errorf("listar: %+v", l)
	}
	if v, err := a.AdxuntoVista(ax.ID); err != nil || !strings.HasPrefix(v, "data:image/jpeg;base64,") {
		t.Errorf("miniatura: %q %v", v[:min(30, len(v))], err)
	}
	if err := a.AdxuntoEliminar(ax.ID); err != nil {
		t.Fatal(err)
	}
	if l := a.AdxuntosListar(); len(l) != 1 {
		t.Errorf("tras eliminar: %+v", l)
	}
	if err := a.AdxuntosLimpar(); err != nil {
		t.Fatal(err)
	}
	if l := a.AdxuntosListar(); len(l) != 0 {
		t.Errorf("tras limpar: %+v", l)
	}
}

func TestAlmacenAdxuntosLimitesESeguridade(t *testing.T) {
	aillarAdxuntos(t)
	a := &App{}
	if _, err := a.AdxuntoGardar("x.txt", ""); err == nil {
		t.Error("un ficheiro baleiro debería rexeitarse")
	}
	if _, err := a.AdxuntoGardar("x.txt", "!!!non-base64"); err == nil {
		t.Error("base64 malo debería rexeitarse")
	}
	if _, err := gardarAdxunto("gordo.bin", make([]byte, adxuntoMaxBytes+1), ""); err == nil || !strings.Contains(err.Error(), "MB") {
		t.Errorf("ficheiro enorme: %v", err)
	}
	if _, err := gardarAdxunto("rota.png", append([]byte("\x89PNG"), 1, 2, 3), ""); err == nil {
		t.Error("unha imaxe corrupta debería rexeitarse")
	}
	// ids con ../ non saen do almacén.
	for _, id := range []string{"../../etc/passwd", "a/b", "", "A1", "x"} {
		if _, _, err := adxuntoPorID(id); err == nil {
			t.Errorf("id %q debería rexeitarse", id)
		}
		if err := a.AdxuntoEliminar(id); err == nil {
			t.Errorf("eliminar %q debería rexeitarse", id)
		}
	}
	if _, _, err := adxuntoPorID("a000000000"); err == nil {
		t.Error("un id inexistente debería dar erro")
	}
}

func TestRedimensionarImaxe(t *testing.T) {
	// Un PNG transparente segue PNG; redúcese ao lado máximo.
	transp := pngProba(400, 200, color.NRGBA{R: 1, G: 2, B: 3, A: 100})
	out, mime, w, h, err := redimensionarImaxe(transp, 100, "")
	if err != nil || mime != "image/png" || w != 100 || h != 50 {
		t.Fatalf("png transparente: %v %s %dx%d", err, mime, w, h)
	}
	if _, _, err := image.DecodeConfig(bytes.NewReader(out)); err != nil {
		t.Error(err)
	}
	// Un JPEG segue JPEG; non se amplía o que xa é pequeno.
	out, mime, w, h, err = redimensionarImaxe(jpegProba(50, 30), 1000, "")
	if err != nil || mime != "image/jpeg" || w != 50 || h != 30 || !bytes.HasPrefix(out, []byte("\xff\xd8")) {
		t.Errorf("jpeg: %v %s %dx%d", err, mime, w, h)
	}
	// Forzar formato.
	if _, mime, _, _, _ := redimensionarImaxe(transp, 0, "jpeg"); mime != "image/jpeg" {
		t.Errorf("forzar jpeg: %s", mime)
	}
	if _, _, _, _, err := redimensionarImaxe([]byte("non é imaxe"), 10, ""); err == nil {
		t.Error("datos non imaxe deberían dar erro")
	}
}

func TestAdxuntoDataURIParaOCurso(t *testing.T) {
	aillarAdxuntos(t)
	a := &App{}
	ax := gardarProba(t, "logo.png", pngProba(3000, 1000, color.NRGBA{B: 200, A: 255}))
	uri, err := a.AdxuntoDataURI(ax.ID, 900)
	if err != nil || !strings.HasPrefix(uri, "data:image/png;base64,") {
		t.Fatalf("%v %q", err, uri[:20])
	}
	raw, err := decodificarDataURICrono(uri)
	if err != nil {
		t.Fatal(err)
	}
	cfg, _, _ := image.DecodeConfig(bytes.NewReader(raw))
	if cfg.Width != 900 || cfg.Height != 300 {
		t.Errorf("dimensións: %dx%d", cfg.Width, cfg.Height)
	}
	// O data URI vale para o logotipo do curso (imaxeOfiDeURI, o mesmo que usa o informe).
	if im, err := imaxeOfiDeURI(uri); err != nil || im.W != 900 {
		t.Errorf("non vale para o informe: %v", err)
	}
	doc := gardarProba(t, "a.txt", []byte("x"))
	if _, err := a.AdxuntoDataURI(doc.ID, 0); err == nil || !strings.Contains(err.Error(), "non é unha imaxe") {
		t.Errorf("un documento non é unha imaxe: %v", err)
	}
}

// ---------- Extracción de texto ----------

func TestTextoDeDocxEOdtXeradosPorLua(t *testing.T) {
	aillarAdxuntos(t)
	d, err := prepararOfimatica(XerarInformeCronogramaRequest{Plan: planMostraCrono()}, nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	docx, _ := escribirDocxCrono(&d)
	odt, _ := escribirOdtCrono(&d)
	for nome, datos := range map[string][]byte{"informe.docx": docx, "informe.odt": odt} {
		ax := gardarProba(t, nome, datos)
		txt, err := textoDoAdxunto(ax, datos)
		if err != nil {
			t.Fatalf("%s: %v", nome, err)
		}
		for _, quero := range []string{"Cronograma anual", "IES Castelao & Fillos, 100% público", "Avaliación inicial", "Primeiro paso", "María Pérez Souto"} {
			if !strings.Contains(txt, quero) {
				t.Errorf("%s: falta %q en el texto:\n%.400s", nome, quero, txt)
			}
		}
		if strings.Contains(txt, "\t\t\t") {
			t.Errorf("%s: sobran tabuladores (paradas dos estilos?)", nome)
		}
	}
}

func TestTextoDeWordComTaboasCabeceiraEPes(t *testing.T) {
	aillarAdxuntos(t)
	ns := `xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"`
	doc := `<w:document ` + ns + `><w:body><w:p><w:pPr><w:tabs><w:tab w:val="left" w:pos="100"/></w:tabs></w:pPr><w:r><w:t>Primeira</w:t></w:r><w:r><w:tab/></w:r><w:r><w:t>liña</w:t></w:r></w:p>` +
		`<w:tbl><w:tr><w:tc><w:p><w:r><w:t>A1</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>B1</w:t></w:r></w:p></w:tc></w:tr></w:tbl></w:body></w:document>`
	cab := `<w:hdr ` + ns + `><w:p><w:r><w:t>IES Rosalía · Tel. 986 000 000</w:t></w:r></w:p></w:hdr>`
	datos := zipProba(t, map[string]string{"word/document.xml": doc, "word/header1.xml": cab})
	ax := gardarProba(t, "modelo.docx", datos)
	txt, err := textoDoAdxunto(ax, datos)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(txt, "Primeira\tliña") || !strings.Contains(txt, "A1\tB1") || !strings.Contains(txt, "[header1]\nIES Rosalía · Tel. 986 000 000") {
		t.Errorf("texto: %q", txt)
	}
}

func TestTextoDePresentacionFollaRTFETexto(t *testing.T) {
	aillarAdxuntos(t)
	a := `xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"`
	pptx := zipProba(t, map[string]string{
		"ppt/slides/slide2.xml": `<p:sld ` + a + `><a:p><a:r><a:t>Segundo</a:t></a:r></a:p></p:sld>`,
		"ppt/slides/slide1.xml": `<p:sld ` + a + `><a:p><a:r><a:t>Primeiro</a:t></a:r></a:p></p:sld>`,
	})
	ax := gardarProba(t, "charla.pptx", pptx)
	txt, err := textoDoAdxunto(ax, pptx)
	if err != nil || strings.Index(txt, "Primeiro") > strings.Index(txt, "Segundo") || !strings.Contains(txt, "--- Diapositiva 2 ---") {
		t.Errorf("pptx: %v %q", err, txt)
	}

	xlsx := zipProba(t, map[string]string{
		"xl/sharedStrings.xml":     `<sst><si><t>Nome</t></si><si><r><t>Ana</t></r><r><t> Pérez</t></r></si></sst>`,
		"xl/worksheets/sheet1.xml": `<worksheet><sheetData><row><c t="s"><v>0</v></c><c><v>Idade</v></c></row><row><c t="s"><v>1</v></c><c><v>14</v></c></row></sheetData></worksheet>`,
	})
	ax = gardarProba(t, "alumnado.xlsx", xlsx)
	txt, err = textoDoAdxunto(ax, xlsx)
	if err != nil || !strings.Contains(txt, "Nome\tIdade") || !strings.Contains(txt, "Ana Pérez\t14") {
		t.Errorf("xlsx: %v %q", err, txt)
	}

	ods := zipProba(t, map[string]string{"content.xml": `<office:document-content xmlns:office="o" xmlns:table="t" xmlns:text="x"><office:body><table:table><table:table-row><table:table-cell><text:p>A</text:p></table:table-cell><table:table-cell><text:p>B</text:p></table:table-cell></table:table-row></table:table></office:body></office:document-content>`})
	ax = gardarProba(t, "folla.ods", ods)
	if txt, err = textoDoAdxunto(ax, ods); err != nil || !strings.Contains(txt, "A\tB") {
		t.Errorf("ods: %v %q", err, txt)
	}

	rtf := []byte(`{\rtf1\ansi{\fonttbl\f0 Arial;}\f0 Ola \b mundo\b0\par Segunda liña \'e1}`)
	ax = gardarProba(t, "carta.rtf", rtf)
	if txt, err = textoDoAdxunto(ax, rtf); err != nil || !strings.Contains(txt, "Ola mundo") || !strings.Contains(txt, "Segunda liña á") {
		t.Errorf("rtf: %v %q", err, txt)
	}

	latin1 := []byte("acci\xf3n;educaci\xf3n")
	ax = gardarProba(t, "datos.csv", latin1)
	if txt, err = textoDoAdxunto(ax, latin1); err != nil || txt != "acción;educación" {
		t.Errorf("latin1: %v %q", err, txt)
	}
	html := []byte("<html><body><h1>Título</h1><script>x()</script><p>Corpo</p></body></html>")
	ax = gardarProba(t, "pax.html", html)
	if txt, err = textoDoAdxunto(ax, html); err != nil || !strings.Contains(txt, "Corpo") || strings.Contains(txt, "x()") {
		t.Errorf("html: %v %q", err, txt)
	}
	img := gardarProba(t, "foto.png", pngProba(4, 4, color.NRGBA{A: 255}))
	if _, err = textoDoAdxunto(img, nil); err == nil || !strings.Contains(err.Error(), "adxunto_ver") {
		t.Errorf("unha imaxe non ten texto: %v", err)
	}
	if _, err := textoDoAdxunto(Adxunto{ID: "abc", Nome: "x.docx", Tipo: tipoAdxDocx}, []byte("non é zip")); err == nil {
		t.Error("un docx corrupto debería dar erro")
	}
}

const pdfProba = "%PDF-1.4\n1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj\n2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj\n" +
	"3 0 obj<</Type/Page/Parent 2 0 R/MediaBox[0 0 300 200]/Contents 4 0 R/Resources<</Font<</F1 5 0 R>>>>>>endobj\n" +
	"4 0 obj<</Length 60>>stream\nBT /F1 18 Tf 20 100 Td (Plan de orientacion 2025) Tj ET\nendstream endobj\n" +
	"5 0 obj<</Type/Font/Subtype/Type1/BaseFont/Helvetica>>endobj\ntrailer<</Root 1 0 R/Size 6>>\n%%EOF\n"

func TestTextoEPaxinasDePDF(t *testing.T) {
	if !commandExists("pdftotext") || !commandExists("pdftoppm") {
		t.Skip("poppler-utils non está instalado")
	}
	aillarAdxuntos(t)
	ax := gardarProba(t, "plan.pdf", []byte(pdfProba))
	if ax.Tipo != tipoAdxPDF {
		t.Fatalf("tipo: %s", ax.Tipo)
	}
	txt, err := textoDoAdxunto(ax, []byte(pdfProba))
	if err != nil || !strings.Contains(txt, "Plan de orientacion 2025") {
		t.Errorf("pdf: %v %q", err, txt)
	}
	png, err := renderizarPaxina(ax, []byte(pdfProba), 1)
	if err != nil {
		t.Fatal(err)
	}
	if cfg, _, err := image.DecodeConfig(bytes.NewReader(png)); err != nil || cfg.Width < 100 {
		t.Errorf("páxina renderizada: %v", err)
	}
	if _, err := renderizarPaxina(ax, []byte(pdfProba), 9); err == nil {
		t.Error("unha páxina que non existe debería dar erro")
	}
}

func TestImaxesDentroDeDocumentos(t *testing.T) {
	aillarAdxuntos(t)
	logo := pngProba(120, 90, color.NRGBA{R: 30, G: 60, B: 200, A: 255})
	docx := zipProba(t, map[string]string{"word/document.xml": "<x/>", "word/media/image1.png": string(logo), "word/media/vector.emf": "emf", "word/theme.png": "no"})
	ax := gardarProba(t, "papel-timbrado.docx", docx)
	ims, err := imaxesDoAdxunto(ax, docx, 10)
	if err != nil || len(ims) != 1 || ims[0].Nome != "image1.png" {
		t.Fatalf("docx: %v %+v", err, ims)
	}
	// Un documento de texto non ten imaxes que extraer.
	txt := gardarProba(t, "a.txt", []byte("x"))
	if _, err := imaxesDoAdxunto(txt, []byte("x"), 5); err == nil {
		t.Error("un .txt non ten imaxes")
	}
}

// ---------- Resolución para a IA ----------

func TestResolverAdxuntosMensaxes(t *testing.T) {
	aillarAdxuntos(t)
	img := gardarProba(t, "modelo.png", pngProba(3000, 2000, color.NRGBA{R: 9, A: 255}))
	curto := gardarProba(t, "datos.txt", []byte("Centro: IES Rosalía"))
	longo := gardarProba(t, "libro.txt", []byte(strings.Repeat("palabra ", 3000)))
	msgs := []IAMensaxe{{Rol: "user", Texto: "mira isto", Adxuntos: []string{img.ID, curto.ID, longo.ID, "a99999", "../../x"}}}
	out := resolverAdxuntosMensaxes(msgs)
	m := out[0]
	if len(m.ImaxesIA) != 1 || m.ImaxesIA[0].Mime != "image/png" && m.ImaxesIA[0].Mime != "image/jpeg" {
		t.Fatalf("imaxes: %+v", m.ImaxesIA)
	}
	raw, _ := base64.StdEncoding.DecodeString(m.ImaxesIA[0].B64)
	if cfg, _, _ := image.DecodeConfig(bytes.NewReader(raw)); max(cfg.Width, cfg.Height) != imaxeIAMaxLado {
		t.Errorf("a imaxe non se reduciu: %dx%d", cfg.Width, cfg.Height)
	}
	for _, quero := range []string{"mira isto", "[Adxuntos desta mensaxe", img.ID, "modelo.png", "vai incluída", "Centro: IES Rosalía", "adxunto_ler", "non dispoñible"} {
		if !strings.Contains(m.Texto, quero) {
			t.Errorf("a nota non ten %q:\n%s", quero, m.Texto)
		}
	}
	// O historial orixinal non se modifica.
	if msgs[0].Texto != "mira isto" || len(msgs[0].ImaxesIA) != 0 {
		t.Error("resolverAdxuntosMensaxes mutou a mensaxe orixinal")
	}
}

func TestResolverAdxuntosLimitaAsImaxesEDeixaAsNovas(t *testing.T) {
	aillarAdxuntos(t)
	var ids []string
	for i := 0; i < maxImaxesPorPeticion+3; i++ {
		ids = append(ids, gardarProba(t, "i.png", pngProba(8, 8, color.NRGBA{A: 255})).ID)
	}
	msgs := []IAMensaxe{
		{Rol: "user", Texto: "vella", Adxuntos: ids[:5]},
		{Rol: "assistant", Texto: "ok"},
		{Rol: "user", Texto: "nova", Adxuntos: ids[5:]},
	}
	out := resolverAdxuntosMensaxes(msgs)
	if len(out[2].ImaxesIA) != maxImaxesPorPeticion-2+2-0 && len(out[2].ImaxesIA) != 10 {
		t.Errorf("as novas deberían entrar todas: %d", len(out[2].ImaxesIA))
	}
	total := 0
	for _, m := range out {
		total += len(m.ImaxesIA)
	}
	if total != maxImaxesPorPeticion {
		t.Errorf("total de imaxes = %d, o máximo é %d", total, maxImaxesPorPeticion)
	}
	if !strings.Contains(out[0].Texto, "hai demasiadas imaxes") {
		t.Errorf("a mensaxe vella debería dicir que non incluíu imaxes:\n%s", out[0].Texto)
	}
}

func TestResultadosDeFerramentasCoImaxes(t *testing.T) {
	aillarAdxuntos(t)
	im := gardarProba(t, "previa.png", pngProba(30, 30, color.NRGBA{A: 255}))
	msgs := []IAMensaxe{
		{Rol: "user", Texto: "x"},
		{Rol: "assistant", Chamadas: []IAChamada{{ID: "c1", Nome: "adxunto_ver"}}},
		{Rol: "tool", Resultados: []IAResultado{{ChamadaID: "c1", Nome: "adxunto_ver", Contido: "ok", Imaxes: []string{im.ID, "a000000000"}}}},
	}
	out := resolverAdxuntosMensaxes(msgs)
	r := out[2].Resultados[0]
	if len(r.ImaxesIA) != 1 || !strings.Contains(r.Contido, "non dispoñible") {
		t.Errorf("%+v", r)
	}
	if len(msgs[2].Resultados[0].ImaxesIA) != 0 || strings.Contains(msgs[2].Resultados[0].Contido, "non dispoñible") {
		t.Error("o historial orixinal mutou")
	}
}

// ---------- Imaxes nos tres provedores ----------

func TestOpenAILevaImaxesDoUsuarioEDasFerramentas(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		msgs := lerCorpo(t, r)["messages"].([]any)
		u := msgs[1].(map[string]any)["content"].([]any)
		if u[0].(map[string]any)["type"] != "text" || u[1].(map[string]any)["type"] != "image_url" {
			t.Errorf("mensaxe do usuario con imaxe: %+v", u)
		}
		if url := u[1].(map[string]any)["image_url"].(map[string]any)["url"].(string); !strings.HasPrefix(url, "data:image/png;base64,QUJD") {
			t.Errorf("url da imaxe: %.40s", url)
		}
		// tool + user con imaxes despois
		if msgs[3].(map[string]any)["role"] != "tool" {
			t.Errorf("esperaba tool: %+v", msgs[3])
		}
		ult := msgs[4].(map[string]any)
		if ult["role"] != "user" || len(ult["content"].([]any)) != 2 {
			t.Errorf("as imaxes da ferramenta van nunha mensaxe de usuario: %+v", ult)
		}
		w.Write([]byte(`{"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"vexo"}}]}`))
	}))
	defer srv.Close()
	im := []imaxeIA{{Mime: "image/png", B64: "QUJD"}}
	msgs := []IAMensaxe{
		{Rol: "user", Texto: "mira", ImaxesIA: im},
		{Rol: "assistant", Chamadas: []IAChamada{{ID: "c1", Nome: "adxunto_ver", Args: "{}"}}},
		{Rol: "tool", Resultados: []IAResultado{{ChamadaID: "c1", Nome: "adxunto_ver", Contido: "ok", ImaxesIA: im}}},
	}
	if _, err := chamarIAConversa(Settings{IAProvedor: "openai", IABaseURL: srv.URL, IAAPIKey: "k", IAModel: "m"}, "s", msgs, nil); err != nil {
		t.Fatal(err)
	}
}

func TestAnthropicLevaImaxesDoUsuarioEDentroDoToolResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		msgs := lerCorpo(t, r)["messages"].([]any)
		u := msgs[0].(map[string]any)["content"].([]any)
		src := u[0].(map[string]any)["source"].(map[string]any)
		if u[0].(map[string]any)["type"] != "image" || src["type"] != "base64" || src["media_type"] != "image/png" || src["data"] != "QUJD" || u[1].(map[string]any)["type"] != "text" {
			t.Errorf("mensaxe do usuario: %+v", u)
		}
		tr := msgs[2].(map[string]any)["content"].([]any)[0].(map[string]any)
		cont := tr["content"].([]any)
		if tr["type"] != "tool_result" || cont[0].(map[string]any)["type"] != "text" || cont[1].(map[string]any)["type"] != "image" {
			t.Errorf("tool_result con imaxes: %+v", tr)
		}
		w.Write([]byte(`{"stop_reason":"end_turn","content":[{"type":"text","text":"vexo"}]}`))
	}))
	defer srv.Close()
	im := []imaxeIA{{Mime: "image/png", B64: "QUJD"}}
	msgs := []IAMensaxe{
		{Rol: "user", Texto: "mira", ImaxesIA: im},
		{Rol: "assistant", Chamadas: []IAChamada{{ID: "c1", Nome: "adxunto_ver", Args: "{}"}}},
		{Rol: "tool", Resultados: []IAResultado{{ChamadaID: "c1", Nome: "adxunto_ver", Contido: "ok", ImaxesIA: im}}},
	}
	if _, err := chamarIAConversa(Settings{IAProvedor: "anthropic", IABaseURL: srv.URL, IAAPIKey: "k", IAModel: "m", IAMaxTokens: 100}, "s", msgs, nil); err != nil {
		t.Fatal(err)
	}
}

func TestGeminiLevaImaxesComoInlineData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := lerCorpo(t, r)["contents"].([]any)
		u := c[0].(map[string]any)["parts"].([]any)
		id := u[1].(map[string]any)["inlineData"].(map[string]any)
		if id["mimeType"] != "image/jpeg" || id["data"] != "QUJD" {
			t.Errorf("inlineData do usuario: %+v", u)
		}
		partes := c[2].(map[string]any)["parts"].([]any)
		if _, ok := partes[0].(map[string]any)["functionResponse"]; !ok {
			t.Errorf("primeiro va o functionResponse: %+v", partes)
		}
		if _, ok := partes[1].(map[string]any)["inlineData"]; !ok {
			t.Errorf("despois a imaxe da ferramenta: %+v", partes)
		}
		w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"vexo"}]}}]}`))
	}))
	defer srv.Close()
	im := []imaxeIA{{Mime: "image/jpeg", B64: "QUJD"}}
	msgs := []IAMensaxe{
		{Rol: "user", Texto: "mira", ImaxesIA: im},
		{Rol: "assistant", Chamadas: []IAChamada{{ID: "c1", Nome: "adxunto_ver", Args: "{}"}}},
		{Rol: "tool", Resultados: []IAResultado{{ChamadaID: "c1", Nome: "adxunto_ver", Contido: "ok", ImaxesIA: im}}},
	}
	if _, err := chamarIAConversa(Settings{IAProvedor: "gemini", IABaseURL: srv.URL, IAAPIKey: "k", IAModel: "m"}, "s", msgs, nil); err != nil {
		t.Fatal(err)
	}
}

func TestIATurnoAsistenteMandaImaxesAdxuntasEExplicaOErroSenVision(t *testing.T) {
	aillarAdxuntos(t)
	ax := gardarProba(t, "papel.png", pngProba(50, 50, color.NRGBA{A: 255}))
	var recibida bool
	falla := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := lerCorpo(t, r)["messages"].([]any)[1].(map[string]any)["content"]
		_, recibida = u.([]any)
		if falla {
			w.WriteHeader(400)
			w.Write([]byte(`{"error":{"message":"this model does not support image input"}}`))
			return
		}
		w.Write([]byte(`{"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"vexo un papel"}}]}`))
	}))
	defer srv.Close()
	a := &App{settings: Settings{IAProvedor: "openai", IABaseURL: srv.URL, IAAPIKey: "k", IAModel: "m"}}
	req := IATurnoRequest{Mensaxes: []IAMensaxe{{Rol: "user", Texto: "que ves?", Adxuntos: []string{ax.ID}}}}
	tr, err := a.IATurnoAsistente(req)
	if err != nil || tr.Texto != "vexo un papel" || !recibida {
		t.Fatalf("%v %+v recibida=%v", err, tr, recibida)
	}
	falla = true
	if _, err := a.IATurnoAsistente(req); err == nil || !strings.Contains(err.Error(), "modelo con visión") {
		t.Errorf("o erro debería explicar o da visión: %v", err)
	}
	// Sen imaxes o erro sae sen a nota.
	if _, err := a.IATurnoAsistente(IATurnoRequest{Mensaxes: []IAMensaxe{{Rol: "user", Texto: "hola"}}}); err == nil || strings.Contains(err.Error(), "visión") {
		t.Errorf("sen imaxes non fai falla a nota: %v", err)
	}
}

// ---------- Ferramentas ----------

func chamarFerr(a *App, nome string, args map[string]any) IAResultado {
	b, _ := json.Marshal(args)
	return a.IAChamarFerramenta(nome, string(b))
}

func TestFerramentasDeAdxuntos(t *testing.T) {
	aillarAdxuntos(t)
	a := &App{}
	if r := chamarFerr(a, "adxuntos_listar", nil); r.Erro || !strings.Contains(r.Contido, `"total": 0`) {
		t.Fatalf("%+v", r)
	}
	txt := gardarProba(t, "libro.txt", []byte(strings.Repeat("abcdefghij", 3000)))
	r := chamarFerr(a, "adxunto_ler", map[string]any{"id": txt.ID, "maxCaracteres": 100})
	if r.Erro || !strings.Contains(r.Contido, "inicio=100") {
		t.Errorf("adxunto_ler en partes: %+v", r)
	}
	r = chamarFerr(a, "adxunto_ler", map[string]any{"id": txt.ID, "inicio": 29990, "maxCaracteres": 100})
	if r.Erro || strings.Contains(r.Contido, "continúa") {
		t.Errorf("a última parte non debe dicir que continúa: %+v", r)
	}
	if r := chamarFerr(a, "adxunto_ler", map[string]any{"id": "a000000000"}); !r.Erro {
		t.Error("id inexistente")
	}

	img := gardarProba(t, "modelo.png", pngProba(20, 20, color.NRGBA{A: 255}))
	r = chamarFerr(a, "adxunto_ver", map[string]any{"id": img.ID})
	if r.Erro || len(r.Imaxes) != 1 || r.Imaxes[0] != img.ID {
		t.Errorf("adxunto_ver imaxe: %+v", r)
	}
	if r := chamarFerr(a, "adxunto_ler", map[string]any{"id": img.ID}); !r.Erro || !strings.Contains(r.Contido, "adxunto_ver") {
		t.Errorf("ler unha imaxe debe apuntar a adxunto_ver: %+v", r)
	}

	// Extraer imaxes dun docx e velas.
	logo := pngProba(100, 100, color.NRGBA{R: 200, A: 255})
	docx := gardarProba(t, "timbrado.docx", zipProba(t, map[string]string{"word/document.xml": "<x/>", "word/media/logo.png": string(logo)}))
	r = chamarFerr(a, "adxunto_extraer_imaxes", map[string]any{"id": docx.ID})
	if r.Erro || !strings.Contains(r.Contido, `"total": 1`) {
		t.Fatalf("extraer: %+v", r)
	}
	if l := a.AdxuntosListar(); len(l) != 4 || l[3].Orixe != docx.ID || l[3].Tipo != tipoAdxImaxe {
		t.Errorf("a imaxe extraída debería ser un adxunto novo con orixe: %+v", l)
	}

	if r := chamarFerr(a, "adxunto_eliminar", map[string]any{"id": txt.ID}); r.Erro {
		t.Error(r.Contido)
	}
}

func TestGardarAdxuntoEnPlantilla(t *testing.T) {
	aillarAdxuntos(t)
	a := &App{}
	est, err := a.ListarPlantillas()
	if err != nil || len(est.Plantillas) == 0 {
		t.Fatal(err)
	}
	fabrica := est.Plantillas[0]
	img := gardarProba(t, "Logo do Centro (1).png", pngProba(200, 100, color.NRGBA{R: 5, A: 255}))

	propia, err := a.DuplicarPlantilla(fabrica.ID)
	if err != nil {
		t.Fatal(err)
	}
	r := chamarFerr(a, "adxunto_gardar_en_plantilla", map[string]any{"id": img.ID, "plantilla": propia.ID})
	if r.Erro {
		t.Fatal(r.Contido)
	}
	_, g0, _, _ := idsDe(t, r)
	var res struct{ Nome string }
	if len(g0) == 1 {
		res.Nome = g0[0].Nome
	}
	if res.Nome != "Logo-do-Centro-1.png" {
		t.Errorf("o nome debe quedar seguro para LaTeX: %q", res.Nome)
	}
	dir, _ := plantillaImaxesDir(propia.ID)
	gardada, err := os.ReadFile(filepath.Join(dir, res.Nome))
	if err != nil || !bytes.HasPrefix(gardada, []byte("\x89PNG")) {
		t.Fatalf("a imaxe non está no cartafol da plantilla: %v", err)
	}
	if p, _ := plantillaPorID(a, propia.ID); !containsStr(p.Imaxes, res.Nome) {
		t.Errorf("plantilla_ler debería listar a imaxe: %v", p.Imaxes)
	}
	// Mesmo nome: crea outro; con substituir, pisa.
	r = chamarFerr(a, "adxunto_gardar_en_plantilla", map[string]any{"id": img.ID, "plantilla": propia.ID, "nome": "logotipo.png"})
	r2 := chamarFerr(a, "adxunto_gardar_en_plantilla", map[string]any{"id": img.ID, "plantilla": propia.ID, "nome": "logotipo.png"})
	r3 := chamarFerr(a, "adxunto_gardar_en_plantilla", map[string]any{"id": img.ID, "plantilla": propia.ID, "nome": "logotipo.png", "substituir": true})
	if !strings.Contains(r.Contido, `"logotipo.png"`) || !strings.Contains(r2.Contido, `"logotipo-1.png"`) || !strings.Contains(r3.Contido, `"logotipo.png"`) {
		t.Errorf("nomes: %s | %s | %s", r.Contido, r2.Contido, r3.Contido)
	}
	// Un documento de texto non vai a unha plantilla.
	doc := gardarProba(t, "a.txt", []byte("x"))
	if r := chamarFerr(a, "adxunto_gardar_en_plantilla", map[string]any{"id": doc.ID, "plantilla": propia.ID}); !r.Erro || !strings.Contains(r.Contido, "adxunto_extraer_imaxes") {
		t.Errorf("un documento non vai a unha plantilla: %+v", r)
	}
	// Eliminar (e non nas de exemplo).
	if r := chamarFerr(a, "plantilla_imaxe_eliminar", map[string]any{"plantilla": propia.ID, "nome": "logotipo-1.png"}); r.Erro {
		t.Error(r.Contido)
	}
	if r := chamarFerr(a, "plantilla_imaxe_eliminar", map[string]any{"plantilla": fabrica.ID, "nome": "x.png"}); !r.Erro {
		t.Error("nas de exemplo non se quitan imaxes")
	}
}

func containsStr(l []string, s string) bool {
	for _, x := range l {
		if x == s {
			return true
		}
	}
	return false
}

func TestGardarAdxuntoWebPConvertidoAPNG(t *testing.T) {
	aillarAdxuntos(t)
	a := &App{}
	est, _ := a.ListarPlantillas()
	propia, _ := a.DuplicarPlantilla(est.Plantillas[0].ID)
	// Un GIF (formato que LaTeX non le) pasa a PNG.
	var gifB bytes.Buffer
	gif.Encode(&gifB, image.NewPaletted(image.Rect(0, 0, 8, 8), color.Palette{color.Black, color.White}), nil)
	ax := gardarProba(t, "animada.gif", gifB.Bytes())
	if ax.Tipo != tipoAdxImaxe {
		t.Skipf("o GIF de proba non se recoñeceu: %+v", ax)
	}
	r := chamarFerr(a, "adxunto_gardar_en_plantilla", map[string]any{"id": ax.ID, "plantilla": propia.ID})
	if r.Erro || !strings.Contains(r.Contido, `"animada.png"`) {
		t.Errorf("o GIF debe gardarse coma PNG: %+v", r)
	}
}

func TestPlantillaPrevisualizarDevolveUnhaImaxe(t *testing.T) {
	if !commandExists("pdflatex") || !commandExists("pdftoppm") {
		t.Skip("LaTeX non está instalado")
	}
	aillarAdxuntos(t)
	a := &App{}
	est, _ := a.ListarPlantillas()
	r := chamarFerr(a, "plantilla_previsualizar", map[string]any{"id": est.Plantillas[0].ID, "paxina": 1})
	if r.Erro || len(r.Imaxes) != 1 {
		t.Fatalf("%+v", r)
	}
	ax, datos, err := adxuntoPorID(r.Imaxes[0])
	if err != nil || ax.Tipo != tipoAdxImaxe || ax.Orixe != "plantilla_previsualizar" || ax.Ancho < 300 {
		t.Fatalf("%v %+v", err, ax)
	}
	if !bytes.HasPrefix(datos, []byte("\x89PNG")) {
		t.Error("non é un PNG")
	}
	// Unha plantilla rota devolve o erro de LaTeX en vez dunha imaxe.
	r = chamarFerr(a, "plantilla_previsualizar", map[string]any{"latex": `\begin{itemize} {{CORPO}}`, "nome": "rota"})
	if !r.Erro || len(r.Imaxes) != 0 {
		t.Errorf("unha plantilla que non compila non ten vista previa: %+v", r)
	}
	if r := chamarFerr(a, "plantilla_previsualizar", map[string]any{"id": est.Plantillas[0].ID, "formato": "rtf"}); !r.Erro {
		t.Error("formato non válido")
	}
}

// ---------- Varios logotipos, PDF, formatos raros e plantillas de exemplo ----------

func idsDe(t *testing.T, r IAResultado) (plantilla string, gardadas []imaxeGardadaEnPlantilla, erros []map[string]string, nota string) {
	t.Helper()
	var res struct {
		Plantilla string                    `json:"plantilla"`
		Gardadas  []imaxeGardadaEnPlantilla `json:"gardadas"`
		Erros     []map[string]string       `json:"erros"`
		Nota      string                    `json:"nota"`
	}
	if err := json.Unmarshal([]byte(r.Contido), &res); err != nil {
		t.Fatalf("%v: %s", err, r.Contido)
	}
	return res.Plantilla, res.Gardadas, res.Erros, res.Nota
}

func TestVariosLogotiposNaMesmaChamadaComPlantillaDeExemplo(t *testing.T) {
	aillarAdxuntos(t)
	a := &App{}
	est, _ := a.ListarPlantillas()
	fabrica := est.Plantillas[0]
	activa := fabrica.ID
	a.EscollerPlantilla(activa, "")
	l1 := gardarProba(t, "logo centro.png", pngProba(200, 80, color.NRGBA{R: 10, A: 255}))
	l2 := gardarProba(t, "logo Xunta.jpg", jpegProba(300, 100))
	// Unha plantilla de EXEMPLO xa non falla: créase unha copia propia (activa) e gárdanse os dous.
	r := chamarFerr(a, "adxunto_gardar_en_plantilla", map[string]any{"plantilla": fabrica.ID, "ids": []string{l1.ID, l2.ID}, "nomes": []string{"logo1.png", "logo2.jpg"}})
	if r.Erro {
		t.Fatal(r.Contido)
	}
	pid, gardadas, erros, nota := idsDe(t, r)
	if pid == fabrica.ID || len(gardadas) != 2 || len(erros) != 0 || !strings.Contains(nota, "copia propia") {
		t.Fatalf("plantilla=%s gardadas=%+v erros=%v nota=%q", pid, gardadas, erros, nota)
	}
	if gardadas[0].Nome != "logo1.png" || gardadas[1].Nome != "logo2.jpg" {
		t.Errorf("nomes: %+v", gardadas)
	}
	nova, err := plantillaPorID(a, pid)
	if err != nil || nova.DeFabrica || !containsStr(nova.Imaxes, "logo1.png") || !containsStr(nova.Imaxes, "logo2.jpg") {
		t.Fatalf("a copia debería ter as dúas imaxes: %v %+v", err, nova.Imaxes)
	}
	est2, _ := a.ListarPlantillas()
	if est2.Activa != pid {
		t.Errorf("a copia debería quedar activa (era a orixinal): %s", est2.Activa)
	}
	if orixinal, _ := plantillaPorID(a, fabrica.ID); len(orixinal.Imaxes) != 0 {
		t.Errorf("a de exemplo non se toca: %v", orixinal.Imaxes)
	}
	// Chamar de novo coa copia propia non crea outra copia.
	r = chamarFerr(a, "adxunto_gardar_en_plantilla", map[string]any{"plantilla": pid, "ids": []string{l1.ID}, "nomes": []string{"logo3.png"}})
	if p2, _, _, nota2 := idsDe(t, r); p2 != pid || nota2 != "" {
		t.Errorf("unha plantilla propia non se copia: %s %q", p2, nota2)
	}
}

func TestUnFicheiroQueFallaNonImpideGardarOsDemais(t *testing.T) {
	aillarAdxuntos(t)
	a := &App{}
	est, _ := a.ListarPlantillas()
	propia, _ := a.DuplicarPlantilla(est.Plantillas[0].ID)
	bo := gardarProba(t, "bo.png", pngProba(20, 20, color.NRGBA{A: 255}))
	txt := gardarProba(t, "notas.txt", []byte("x"))
	r := chamarFerr(a, "adxunto_gardar_en_plantilla", map[string]any{"plantilla": propia.ID, "ids": []string{txt.ID, bo.ID, "a000000000"}})
	if r.Erro {
		t.Fatal(r.Contido)
	}
	_, gardadas, erros, _ := idsDe(t, r)
	if len(gardadas) != 1 || len(erros) != 2 || erros[0]["adxunto"] != txt.ID {
		t.Errorf("gardadas=%+v erros=%+v", gardadas, erros)
	}
	// Se non se garda ningún, é un erro coa razón de cada un.
	r = chamarFerr(a, "adxunto_gardar_en_plantilla", map[string]any{"plantilla": propia.ID, "ids": []string{txt.ID}})
	if !r.Erro || !strings.Contains(r.Contido, "non se gardou ningún") || !strings.Contains(r.Contido, txt.ID) {
		t.Errorf("%+v", r)
	}
	if r := chamarFerr(a, "adxunto_gardar_en_plantilla", map[string]any{"plantilla": propia.ID}); !r.Erro || !strings.Contains(r.Contido, "ids") {
		t.Errorf("sen ids: %+v", r)
	}
}

func TestLogotipoEnPDFGardaseVectorialEOuComoPNG(t *testing.T) {
	if !commandExists("pdftoppm") {
		t.Skip("poppler-utils non está instalado")
	}
	aillarAdxuntos(t)
	a := &App{}
	est, _ := a.ListarPlantillas()
	propia, _ := a.DuplicarPlantilla(est.Plantillas[0].ID)
	pdf := gardarProba(t, "Logo Centro.pdf", []byte(pdfProba))
	// Por defecto: o PDF vai tal cal (vectorial para LaTeX).
	r := chamarFerr(a, "adxunto_gardar_en_plantilla", map[string]any{"plantilla": propia.ID, "id": pdf.ID, "nome": "logo.pdf"})
	if r.Erro {
		t.Fatal(r.Contido)
	}
	_, g, _, _ := idsDe(t, r)
	dir, _ := plantillaImaxesDir(propia.ID)
	b, err := os.ReadFile(filepath.Join(dir, g[0].Nome))
	if err != nil || !bytes.HasPrefix(b, []byte("%PDF")) || g[0].Nome != "logo.pdf" || !strings.Contains(g[0].Nota, "vectorial") {
		t.Fatalf("o PDF debe gardarse tal cal: %v %+v", err, g)
	}
	// formato=png: rasterízase e recórtase.
	r = chamarFerr(a, "adxunto_gardar_en_plantilla", map[string]any{"plantilla": propia.ID, "id": pdf.ID, "formato": "png", "nome": "logo"})
	if r.Erro {
		t.Fatal(r.Contido)
	}
	_, g, _, _ = idsDe(t, r)
	png, err := os.ReadFile(filepath.Join(dir, g[0].Nome))
	if err != nil || g[0].Nome != "logo.png" || !bytes.HasPrefix(png, []byte("\x89PNG")) {
		t.Fatalf("formato=png: %v %+v", err, g)
	}
	cfg, _, _ := image.DecodeConfig(bytes.NewReader(png))
	if cfg.Width >= 300*300/72 { // a páxina enteira a 300 ppp sería 1250 px de ancho
		t.Errorf("o logotipo non se recortou: %dx%d", cfg.Width, cfg.Height)
	}
}

func TestLogotipoPDFPasaAoJSONDoCursoComoPNG(t *testing.T) {
	if !commandExists("pdftoppm") {
		t.Skip("poppler-utils non está instalado")
	}
	aillarAdxuntos(t)
	a := &App{}
	pdf := gardarProba(t, "logo.pdf", []byte(pdfProba))
	uri, err := a.AdxuntoDataURI(pdf.ID, 900)
	if err != nil || !strings.HasPrefix(uri, "data:image/png;base64,") {
		t.Fatalf("%v", err)
	}
	if im, err := imaxeOfiDeURI(uri); err != nil || im.W < 50 {
		t.Errorf("non vale para o informe: %v", err)
	}
}

func TestRecortarBordosBrancos(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 200, 200))
	for i := range img.Pix {
		img.Pix[i] = 255
	}
	for y := 80; y < 120; y++ {
		for x := 60; x < 140; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 200, A: 255})
		}
	}
	var b bytes.Buffer
	png.Encode(&b, img)
	cfg, _, _ := image.DecodeConfig(bytes.NewReader(recortarBordosBrancos(b.Bytes())))
	if cfg.Width > 90 || cfg.Height > 50 || cfg.Width < 80 {
		t.Errorf("recorte: %dx%d", cfg.Width, cfg.Height)
	}
	// Unha imaxe toda branca non se toca.
	branca := image.NewNRGBA(image.Rect(0, 0, 10, 10))
	for i := range branca.Pix {
		branca.Pix[i] = 255
	}
	var bb bytes.Buffer
	png.Encode(&bb, branca)
	if !bytes.Equal(recortarBordosBrancos(bb.Bytes()), bb.Bytes()) {
		t.Error("unha imaxe toda branca non debe cambiar")
	}
}

func TestFormatosRarosConvertenseAoSubirse(t *testing.T) {
	aillarAdxuntos(t)
	a := &App{}
	// SVG con Inkscape ou ImageMagick, se hai.
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="120" height="60"><rect width="120" height="60" fill="#c00"/></svg>`)
	if commandExists("inkscape") || comandoImageMagick() != "" {
		ax, err := a.AdxuntoGardar("logo.svg", base64.StdEncoding.EncodeToString(svg))
		if err != nil {
			t.Fatalf("svg: %v", err)
		}
		if ax.Tipo != tipoAdxImaxe || ax.Formato != "svg" || ax.Mime != "image/png" || ax.Ancho < 100 {
			t.Errorf("o SVG debe quedar coma imaxe PNG: %+v", ax)
		}
		if v, err := a.AdxuntoVista(ax.ID); err != nil || v == "" {
			t.Errorf("miniatura do SVG: %v", err)
		}
	}
	// RAW/HEIC: se hai conversor convértese; se non, o erro di que instalar (nunca "formato descoñecido").
	_, err := a.AdxuntoGardar("foto.cr2", base64.StdEncoding.EncodeToString([]byte("II*\x00 isto non é un RAW de verdade")))
	if err == nil {
		t.Skip("hai un conversor que aceptou o ficheiro de proba")
	}
	if !strings.Contains(err.Error(), "CR2") || !strings.Contains(err.Error(), "dcraw") {
		t.Errorf("o erro debe explicar que instalar: %v", err)
	}
	if l := a.AdxuntosListar(); len(l) > 1 {
		t.Errorf("un ficheiro que non se converteu non debe quedar no almacén: %+v", l)
	}
	for _, n := range []string{"a.svg", "b.CR2", "c.nef", "d.dng", "e.heic", "f.avif"} {
		if !precisaConversorExterno(n) {
			t.Errorf("%s debería pasar polo conversor externo", n)
		}
	}
	for _, n := range []string{"a.png", "b.jpg", "c.webp", "d.tiff", "e.pdf"} {
		if precisaConversorExterno(n) {
			t.Errorf("%s non precisa conversor", n)
		}
	}
}

// ---------- Cabeceira e pé con logotipos ----------

func TestParsearCaixaOfi(t *testing.T) {
	subst := func(s string) string { return strings.ReplaceAll(s, "{{CENTRO}}", "IES Rosalía") }
	c := parsearCaixaOfi("@imaxe logo1.png 10\n@imaxe logo2.pdf 12mm\n@imaxe\n{{CENTRO}} · Orientación\n\n@esquerda", subst, "dereita")
	if c.Alin != "esquerda" || len(c.Elementos) != 3 {
		t.Fatalf("%+v", c)
	}
	if c.Elementos[0] != (elementoCaixa{Tipo: "imaxe", Nome: "logo1.png", AltoMM: 10}) || c.Elementos[1].Nome != "logo2.pdf" || c.Elementos[1].AltoMM != 12 || c.Elementos[2].Texto != "IES Rosalía · Orientación" {
		t.Errorf("%+v", c.Elementos)
	}
	if !parsearCaixaOfi("  \n", subst, "centro").baleira() || parsearCaixaOfi("", subst, "centro").Alin != "centro" {
		t.Error("baleira")
	}
	if c.textoPlano() != "IES Rosalía · Orientación" {
		t.Errorf("textoPlano: %q", c.textoPlano())
	}
}

func plantillaCosLogotipos(t *testing.T, a *App, pdfLogo bool) PlantillaLatex {
	t.Helper()
	est, _ := a.ListarPlantillas()
	propia, _ := a.DuplicarPlantilla(est.Plantillas[0].ID)
	dir, _ := plantillaImaxesDir(propia.ID)
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "centro.png"), pngProba(300, 100, color.NRGBA{R: 122, G: 31, B: 52, A: 255}), 0o644)
	nome := "xunta.png"
	if pdfLogo {
		nome = "xunta.pdf"
		os.WriteFile(filepath.Join(dir, nome), []byte(pdfProba), 0o644)
	} else {
		os.WriteFile(filepath.Join(dir, nome), pngProba(200, 200, color.NRGBA{B: 180, A: 255}), 0o644)
	}
	propia.Ofimatica = OfimaticaPlantilla{
		Cabeceira: "@imaxe centro.png 10\n@imaxe " + nome + " 10\n{{CENTRO}}\n@esquerda",
		Pe:        "@imaxe centro.png 6\nTel. 986 11 22 33",
	}
	return propia
}

func TestDocxCabeceiraEPeCoDousLogotipos(t *testing.T) {
	aillarAdxuntos(t)
	a := &App{}
	for _, pdfLogo := range []bool{false, true} {
		if pdfLogo && !commandExists("pdftoppm") {
			continue
		}
		pl := plantillaCosLogotipos(t, a, pdfLogo)
		pk, d := docxDe(t, planMostraCrono(), &pl)
		for nome, c := range pk {
			if strings.HasSuffix(nome, ".xml") || strings.HasSuffix(nome, ".rels") {
				xmlBenFormado(t, nome, c)
			}
		}
		if len(d.Avisos) != 0 {
			t.Errorf("pdf=%v avisos: %v", pdfLogo, d.Avisos)
		}
		cab := pk["word/header1.xml"]
		if strings.Count(cab, "<w:drawing>") != 2 || !strings.Contains(cab, "IES Castelao") || !strings.Contains(cab, `<w:jc w:val="left"/>`) {
			t.Errorf("pdf=%v cabeceira: %s", pdfLogo, cab)
		}
		rels := pk["word/_rels/header1.xml.rels"]
		if strings.Count(rels, "relationships/image") != 2 || !strings.Contains(rels, "media/hdr1.png") {
			t.Errorf("pdf=%v rels da cabeceira: %s", pdfLogo, rels)
		}
		for _, f := range []string{"word/media/hdr1.png", "word/media/hdr2.png", "word/media/ftr1.png"} {
			if _, ok := pk[f]; !ok {
				t.Errorf("pdf=%v falta %s", pdfLogo, f)
			}
		}
		if pdf := pk["word/media/hdr2.png"]; !strings.HasPrefix(pdf, "\x89PNG") {
			t.Errorf("o logotipo (PDF: %v) ten que ir coma PNG no Word", pdfLogo)
		}
		pe := pk["word/footer1.xml"]
		if strings.Count(pe, "<w:drawing>") != 1 || !strings.Contains(pe, "Tel. 986 11 22 33") || !strings.Contains(pe, "PAGE") {
			t.Errorf("pé: %s", pe)
		}
		if !strings.Contains(pk["word/_rels/footer1.xml.rels"], "media/ftr1.png") {
			t.Error("rels do pé")
		}
		// Os ids dos debuxos non se repiten.
		ids := regexp.MustCompile(`<wp:docPr id="(\d+)"`).FindAllStringSubmatch(pk["word/document.xml"]+cab+pe, -1)
		vistos := map[string]bool{}
		for _, m := range ids {
			if vistos[m[1]] {
				t.Errorf("id de debuxo repetido: %s", m[1])
			}
			vistos[m[1]] = true
		}
	}
}

func TestOdtCabeceiraEPeCoDousLogotipos(t *testing.T) {
	aillarAdxuntos(t)
	a := &App{}
	pl := plantillaCosLogotipos(t, a, false)
	pk, _ := odtDe(t, planMostraCrono(), &pl)
	for nome, c := range pk {
		if strings.HasSuffix(nome, ".xml") {
			xmlBenFormado(t, nome, c)
		}
	}
	est := pk["styles.xml"]
	if strings.Count(est, "<draw:frame") != 3 { // dous na cabeceira, un no pé
		t.Errorf("frames en styles.xml: %d", strings.Count(est, "<draw:frame"))
	}
	if !strings.Contains(est, `<style:header><text:p text:style-name="HeaderLeft">`) || !strings.Contains(est, "Tel. 986 11 22 33  ·  <text:page-number") {
		t.Errorf("cabeceira/pé: %.600s", est)
	}
	n := 0
	for nome := range pk {
		if strings.HasPrefix(nome, "Pictures/") {
			n++
			if !strings.Contains(pk["META-INF/manifest.xml"], nome) {
				t.Errorf("%s non está no manifesto", nome)
			}
		}
	}
	if n < 4 { // 3 + cronograma anual (+ imaxe da tarefa)
		t.Errorf("imaxes no paquete: %d", n)
	}
}

func TestPlantillaGardarAdmiteCabeceiraCoLogotipos(t *testing.T) {
	aillarAdxuntos(t)
	a := &App{}
	r := chamarFerr(a, "plantilla_gardar", map[string]any{"nome": "Con logos", "latex": "{{CORPO}}",
		"ofimatica": map[string]any{"cabeceira": "@imaxe a.png 10\n@imaxe b.pdf 10\nTexto"}})
	if r.Erro {
		t.Fatal(r.Contido)
	}
	var res struct{ ID string }
	json.Unmarshal([]byte(r.Contido), &res)
	p, _ := plantillaPorID(a, res.ID)
	if !strings.Contains(p.Ofimatica.Cabeceira, "@imaxe b.pdf 10") {
		t.Errorf("%+v", p.Ofimatica)
	}
}
