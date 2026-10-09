package main

// Escritor nativo de OpenDocument Text (.odt, LibreOffice). Mesma maqueta ca o
// de Word (informe_docx.go): estilos de parágrafo e de texto nomeados
// (Title, Heading1, TextBody, Quote...) e dúas páxinas mestras (a portada sen
// numerar e o resto co número de páxina ao pé). Ver informe_ofimatica.go.

import (
	"archive/zip"
	"bytes"
	"fmt"
	"hash/crc32"
	"io"
	"strings"
	"time"
)

const (
	// A4 (210 mm) con marxes de 20 mm.
	odtColumnaCm = 17.0
	// Alto máximo dunha figura (unha páxina A4 menos marxes e a lenda).
	odtAltoMaxCm = 22.0
	// Tamaño do corpo en puntos, base dos tamaños de letra.
	odtCorpoPt = 11.0
)

// odtOrdeTamanos fixa a orde (determinista) dos estilos de tamaño de letra.
var odtOrdeTamanos = []string{"peq", "gr", "xgr"}

type imaxeOdt struct {
	href     string
	im       *imaxeOfi
	wcm, hcm float64
}

type construtorOdt struct {
	formulas        *formulasOfi // as fórmulas xa compoñidas (formulas_ofi.go)
	corpo           strings.Builder
	imaxes          []imaxeOdt
	imaxesPlantilla map[string]*imaxeOfi
	nTaboas         int // para dar un nome único a cada táboa (formularios)
}

func pxACm(px int) float64 { return float64(px) * 2.54 / 96.0 }

func (c *construtorOdt) engadirImaxe(im *imaxeOfi, wcm, hcm float64) (imaxeOdt, bool) {
	if im == nil || wcm <= 0 || hcm <= 0 {
		return imaxeOdt{}, false
	}
	o := imaxeOdt{href: fmt.Sprintf("Pictures/image%d.%s", len(c.imaxes)+1, im.Ext), im: im, wcm: wcm, hcm: hcm}
	c.imaxes = append(c.imaxes, o)
	return o, true
}

// engadirImaxeNatural usa o tamaño natural limitado ao ancho da columna e, se
// maxAlto > 0, a ese alto en cm.
func (c *construtorOdt) engadirImaxeNatural(im *imaxeOfi, maxAlto float64) (imaxeOdt, bool) {
	w, h := pxACm(im.W), pxACm(im.H)
	if w > odtColumnaCm {
		h = h * odtColumnaCm / w
		w = odtColumnaCm
	}
	if maxAlto > 0 && h > maxAlto {
		w = w * maxAlto / h
		h = maxAlto
	}
	return c.engadirImaxe(im, w, h)
}

func (c *construtorOdt) p(estilo, texto string) {
	c.corpo.WriteString(`<text:p text:style-name="` + estilo + `">` + escaparXMLOfi(texto) + `</text:p>`)
}

func (c *construtorOdt) pImaxe(o imaxeOdt, estilo string) {
	c.corpo.WriteString(`<text:p text:style-name="` + estilo + `">` + marcoOdt(o) + `</text:p>`)
}

func marcoOdt(o imaxeOdt) string {
	return fmt.Sprintf(
		`<draw:frame draw:style-name="Fr" text:anchor-type="as-char" svg:width="%.2fcm" svg:height="%.2fcm">`+
			`<draw:image xlink:href="%s" xlink:type="simple" xlink:show="embed" xlink:actuate="onLoad"/>`+
			`</draw:frame>`, o.wcm, o.hcm, o.href)
}

// portada escribe a portada segundo o deseño da plantilla e devolve os estilos
// extra que precisan os seus espazos.
func (c *construtorOdt) portada(d *datosOfimatica) string {
	// O primeiro parágrafo do documento fixa a páxina mestra "First_Page" (sen
	// cabeceira nin pé); o resto encadea automaticamente coa mestra "Standard".
	primeiro := true
	var extraEstilos strings.Builder
	titulo := func(estilo, texto string) {
		if primeiro {
			// O primeiro parágrafo leva a mestra da portada.
			switch estilo {
			case "Title":
				estilo = "TitleFirst"
			default:
				estilo = "CoverFirst"
			}
			primeiro = false
		}
		c.p(estilo, texto)
	}
	for i, l := range d.Portada {
		switch l.Tipo {
		case "logo", "imaxe":
			im := d.Logo
			if l.Tipo == "imaxe" {
				im = d.ImaxesPlantilla[l.Nome]
			}
			if im == nil {
				continue
			}
			// Con alto explícito (@logo 30) a imaxe ten EXACTAMENTE ese alto;
			// sen el, o seu tamaño natural con tope de 3,5 cm.
			w, h := pxACm(im.W), pxACm(im.H)
			if l.AltoMM > 0 {
				h = l.AltoMM / 10
				w = h * float64(im.W) / float64(im.H)
			}
			if w > odtColumnaCm {
				h, w = h*odtColumnaCm/w, odtColumnaCm
			}
			if tope := 3.5; l.AltoMM == 0 && h > tope {
				w, h = w*tope/h, tope
			}
			if o, ok := c.engadirImaxe(im, w, h); ok {
				estilo := "Figure"
				if primeiro {
					estilo, primeiro = "FigureCover", false
				}
				c.pImaxe(o, estilo)
			}
		case "espazo":
			nome := fmt.Sprintf("Gap%d", i)
			extraEstilos.WriteString(fmt.Sprintf(`<style:style style:name="%s" style:family="paragraph" style:parent-style-name="CoverMeta"><style:paragraph-properties fo:margin-top="%.2fcm" fo:margin-bottom="0cm"/><style:text-properties fo:font-size="2pt"/></style:style>`, nome, max(l.AltoMM, 1)/10))
			if primeiro {
				nome = nome + "F"
				extraEstilos.WriteString(fmt.Sprintf(`<style:style style:name="%s" style:family="paragraph" style:parent-style-name="%s" style:master-page-name="First_Page"/>`, nome, fmt.Sprintf("Gap%d", i)))
				primeiro = false
			}
			c.p(nome, "")
		case "titulo":
			titulo("Title", l.Texto)
		case "subtitulo":
			titulo("Subtitle", l.Texto)
		case "regra":
			titulo("CoverRule", "")
		case "organizacion":
			titulo("CoverOrg", l.Texto)
		default:
			titulo("CoverMeta", l.Texto)
		}
	}
	return extraEstilos.String()
}

// escribirOdtCrono compón o informe completo e devolve os bytes do .odt.
func escribirOdtCrono(d *datosOfimatica) ([]byte, error) {
	c := &construtorOdt{imaxesPlantilla: d.ImaxesPlantilla}

	// --- Portada (do deseño da plantilla) ---
	extraEstilos := c.portada(d)

	// --- Índice ---
	c.p("Heading1", d.Tx.T("INDICE"))
	c.p("TOCEntry", "1. "+d.Tx.T("INTRODUCION"))
	c.p("TOCEntry", "2. "+d.Tx.T("CRONOGRAMA"))
	c.p("TOCEntry", "3. "+d.Tx.T("DESENVOLVEMENTO"))
	for i := range d.ConTarefas {
		c.p("TOCSub", fmt.Sprintf("3.%d. %s", i+1, d.Nomes[i]))
	}
	c.p("TOCEntry", "4. "+d.Tx.T("SINATURA_TITULO"))

	// --- 1. Introdución ---
	c.p("Heading1", "1. "+d.Tx.T("INTRODUCION"))
	for _, par := range paragrafosOfi(d.Intro) {
		c.p("TextBody", par)
	}

	// --- 2. Cronograma ---
	c.p("Heading1", "2. "+d.Tx.T("CRONOGRAMA"))
	if im, err := imaxeOfiDeBytes(d.CronoPNG); err == nil {
		if o, ok := c.engadirImaxeNatural(im, odtAltoMaxCm); ok {
			c.pImaxe(o, "Figure")
		}
	}

	// --- 3. Desenvolvemento por meses ---
	c.p("Heading1", "3. "+d.Tx.T("DESENVOLVEMENTO"))
	if len(d.ConTarefas) == 0 {
		c.corpo.WriteString(`<text:p text:style-name="TextBody"><text:span text:style-name="Muted">` + escaparXMLOfi(d.Tx.T("SEN_TAREFAS_CURSO")) + `</text:span></text:p>`)
	}
	for i, m := range d.ConTarefas {
		c.p("Heading2", fmt.Sprintf("3.%d. %s", i+1, d.Nomes[i]))
		for _, t := range m.Tarefas {
			c.p("TaskTitle", porDefectoCrono(t.Title, d.Tx.T("SEN_TITULO")))
			bloques := parseMarkdownCrono(t.Description)
			if len(bloques) == 0 {
				c.corpo.WriteString(`<text:p text:style-name="TextBody"><text:span text:style-name="Muted">` + escaparXMLOfi(d.Tx.T("SEN_DESCRICION")) + `</text:span></text:p>`)
			} else {
				c.bloques(bloques, t.Images, &d.Avisos)
			}
		}
	}

	// --- 4. Sinatura ---
	c.p("Heading1", "4. "+d.Tx.T("SINATURA_TITULO"))
	c.p("TextBody", d.Tx.T("SINATURA_TEXTO"))
	c.p("SignName", d.Xefe)
	c.p("SignMeta", d.Cargo)
	c.p("SignMeta", d.Tx.T("LUGAR_DATA"))

	// Cabeceira e pé (co seu logotipos) da plantilla: as imaxes entran no paquete.
	d.cabOdt = c.caixaOdt(d.Cabeceira, "Header")
	d.peOdt = c.caixaOdt(d.Pe, "Footer")

	contido := odtCabeceiraContido + odtEstilosAutomaticosCon(d.Estilo, extraEstilos) +
		`<office:body><office:text>` + c.corpo.String() + `</office:text></office:body></office:document-content>`
	return montarOdt(contido, c.imaxes, d)
}

// estiloAlinOdt devolve o estilo de parágrafo que combina un estilo base cun
// aliñamento (os derivados están en odtEstilosAutomaticos).
func estiloAlinOdt(base string, a cronoAlinamento) string {
	switch a {
	case cronoAlinEsquerda:
		return base + "Left"
	case cronoAlinCentro:
		return base + "Center"
	case cronoAlinDereita:
		return base + "Right"
	case cronoAlinXustificado:
		return base + "Justify"
	}
	return base
}

func (c *construtorOdt) bloques(bloques []cronoBloque, imaxes map[string]string, avisos *[]string) {
	orde := 0
	for _, bl := range bloques {
		if bl.Tipo != cronoItem || !bl.Ordenada {
			orde = 0
		}
		switch bl.Tipo {
		case cronoTitulo:
			c.corpo.WriteString(`<text:p text:style-name="` + estiloAlinOdt("Subheading", bl.Alin) + `">` + escaparXMLOfi(textoTramosOfi(bl.Tramos)) + `</text:p>`)
		case cronoItem:
			marca := "•"
			if bl.Ordenada {
				orde++
				marca = fmt.Sprintf("%d.", orde)
			}
			c.corpo.WriteString(`<text:p text:style-name="` + estiloAlinOdt("ListItem", bl.Alin) + `">` + escaparXMLOfi(marca) + `<text:tab/>` + c.tramos(bl.Tramos) + `</text:p>`)
		case cronoCita:
			c.corpo.WriteString(`<text:p text:style-name="` + estiloAlinOdt("Quote", bl.Alin) + `">` + c.tramos(bl.Tramos) + `</text:p>`)
		case cronoDebuxo:
			im, w, h, ok := c.formulas.debuxo(bl)
			if !ok {
				continue
			}
			wcm, hcm := w*2.54, h*2.54
			if wcm > odtColumnaCm {
				hcm = hcm * odtColumnaCm / wcm
				wcm = odtColumnaCm
			}
			if o, ok := c.engadirImaxe(im, wcm, hcm); ok {
				c.pImaxe(o, "Figure")
			}
		case cronoImaxe:
			im := resolverImaxeOfi(bl.Imaxe.Ref, imaxes, avisos)
			if im == nil {
				continue
			}
			w, h := bl.Imaxe.tamanoMM(odtColumnaCm*10, im.W, im.H)
			if o, ok := c.engadirImaxe(im, w/10, h/10); ok {
				// O estilo Figure xa centra; só esquerda/dereita precisan derivado.
				c.pImaxe(o, estiloAlinOdt("Figure", alinImaxeOfi(bl.Alin)))
			}
		default:
			c.corpo.WriteString(`<text:p text:style-name="` + estiloAlinOdt("TextBody", bl.Alin) + `">` + c.tramos(bl.Tramos) + `</text:p>`)
		}
	}
}

func tramosOdt(tramos []cronoTramo) string {
	var b strings.Builder
	for _, t := range tramos {
		txt := t.cronoTexto()
		if txt == "" {
			continue
		}
		var estilo string
		switch {
		case t.Negrita && t.Cursiva:
			estilo = "StrongEmphasis"
		case t.Negrita:
			estilo = "Strong"
		case t.Cursiva:
			estilo = "Emphasis"
		}
		interior := escaparXMLOfi(txt)
		if estilo != "" {
			interior = `<text:span text:style-name="` + estilo + `">` + interior + `</text:span>`
		}
		// O tamaño vai nun span externo: combínase co estilo de negriña/
		// cursiva sen multiplicar as combinacións posibles.
		if nome := estiloTamanoOdt(t.Escala); nome != "" {
			interior = `<text:span text:style-name="` + nome + `">` + interior + `</text:span>`
		}
		b.WriteString(interior)
	}
	return b.String()
}

func estiloTamanoOdt(escala float64) string {
	if escala <= 0 || escala == 1 {
		return ""
	}
	for _, clave := range odtOrdeTamanos {
		if cronoEscalasTamano[clave] == escala {
			return "FS_" + clave
		}
	}
	return ""
}

func montarOdt(contido string, imaxes []imaxeOdt, d *datosOfimatica) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	// Datas ZIP válidas: sen elas Go escribe 1980-00-00, que non é unha data.
	agora := time.Now()

	// O 'mimetype' debe ir primeiro, SEN comprimir e SEN descritor de datos
	// (ODF 1.2, parte 3, §3.3): LibreOffice 26 non abre o paquete se non.
	// CreateRaw é o único xeito de que Go non engada o descritor.
	mime := []byte("application/vnd.oasis.opendocument.text")
	cab := &zip.FileHeader{
		Name:               "mimetype",
		Method:             zip.Store,
		CRC32:              crc32.ChecksumIEEE(mime),
		CompressedSize64:   uint64(len(mime)),
		UncompressedSize64: uint64(len(mime)),
	}
	cab.SetModTime(agora) //nolint:staticcheck // CreateRaw, a diferenza de CreateHeader, non traduce Modified á data DOS
	mw, err := zw.CreateRaw(cab)
	if err != nil {
		return nil, err
	}
	if _, err := mw.Write(mime); err != nil {
		return nil, err
	}
	crear := func(nome string) (io.Writer, error) {
		return zw.CreateHeader(&zip.FileHeader{Name: nome, Method: zip.Deflate, Modified: agora})
	}
	escribir := func(nome, datos string) error {
		w, err := crear(nome)
		if err != nil {
			return err
		}
		_, err = w.Write([]byte(datos))
		return err
	}
	if err := escribir("content.xml", contido); err != nil {
		return nil, err
	}
	if err := escribir("styles.xml", odtEstilosCon(d)); err != nil {
		return nil, err
	}
	if err := escribir("meta.xml", odtMetaXML); err != nil {
		return nil, err
	}
	var manifesto strings.Builder
	manifesto.WriteString(odtManifestoCabeceira)
	for _, o := range imaxes {
		manifesto.WriteString(fmt.Sprintf(`<manifest:file-entry manifest:full-path="%s" manifest:media-type="%s"/>`, o.href, o.im.Mime))
	}
	manifesto.WriteString(`</manifest:manifest>`)
	if err := escribir("META-INF/manifest.xml", manifesto.String()); err != nil {
		return nil, err
	}
	for _, o := range imaxes {
		w, err := crear(o.href)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(o.im.Datos); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

const odtCabeceiraContido = `<?xml version="1.0" encoding="UTF-8" standalone="no"?>` +
	`<office:document-content ` +
	`xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0" ` +
	`xmlns:style="urn:oasis:names:tc:opendocument:xmlns:style:1.0" ` +
	`xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0" ` +
	`xmlns:table="urn:oasis:names:tc:opendocument:xmlns:table:1.0" ` +
	`xmlns:draw="urn:oasis:names:tc:opendocument:xmlns:drawing:1.0" ` +
	`xmlns:fo="urn:oasis:names:tc:opendocument:xmlns:xsl-fo-compatible:1.0" ` +
	`xmlns:xlink="http://www.w3.org/1999/xlink" ` +
	`xmlns:svg="urn:oasis:names:tc:opendocument:xmlns:svg-compatible:1.0" ` +
	`office:version="1.2">`

// odtEstilosAutomaticosCon define, en content.xml, as variantes aliñadas dos
// estilos de parágrafo do corpo. Herdan do estilo base e só cambian
// fo:text-align, de xeito que unha tarefa pode centrar ou xustificar un
// parágrafo sen perder o resto do formato (marxes, sangrías, bordos...).
// `extra` son estilos propios da portada da plantilla (espazos).
func odtEstilosAutomaticosCon(e estiloOfi, extra string) string {
	var b strings.Builder
	b.WriteString(`<office:automatic-styles>`)
	alins := [][2]string{{"Left", "start"}, {"Center", "center"}, {"Right", "end"}, {"Justify", "justify"}}
	for _, base := range []string{"TextBody", "ListItem", "Quote", "Subheading"} {
		for _, a := range alins {
			b.WriteString(estiloAlinOdtXML(base+a[0], base, a[1]))
		}
	}
	b.WriteString(estiloAlinOdtXML("FigureLeft", "Figure", "start"))
	b.WriteString(estiloAlinOdtXML("FigureRight", "Figure", "end"))
	for _, clave := range odtOrdeTamanos {
		b.WriteString(fmt.Sprintf(
			`<style:style style:name="FS_%s" style:family="text"><style:text-properties fo:font-size="%.1fpt"/></style:style>`,
			clave, e.TamanoPt*cronoEscalasTamano[clave]))
	}
	b.WriteString(extra)
	b.WriteString(`</office:automatic-styles>`)
	return b.String()
}

func estiloAlinOdtXML(nome, pai, alin string) string {
	return `<style:style style:name="` + nome + `" style:family="paragraph" style:parent-style-name="` + pai + `">` +
		`<style:paragraph-properties fo:text-align="` + alin + `"/></style:style>`
}

// odtEstilosXML define a folla de estilos do documento: estilos comúns
// nomeados, estilos de texto, deseño de páxina e as páxinas mestras (portada
// sen número e contido numerado).
const odtEstilosBase = `<?xml version="1.0" encoding="UTF-8" standalone="no"?>` +
	`<office:document-styles ` +
	`xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0" ` +
	`xmlns:style="urn:oasis:names:tc:opendocument:xmlns:style:1.0" ` +
	`xmlns:fo="urn:oasis:names:tc:opendocument:xmlns:xsl-fo-compatible:1.0" ` +
	`xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0" ` +
	`xmlns:draw="urn:oasis:names:tc:opendocument:xmlns:drawing:1.0" ` +
	`xmlns:svg="urn:oasis:names:tc:opendocument:xmlns:svg-compatible:1.0" ` +
	`xmlns:xlink="http://www.w3.org/1999/xlink" office:version="1.2">` +
	`<office:font-face-decls>` +
	`<style:font-face style:name="Arial" svg:font-family="Arial, &apos;Liberation Sans&apos;, sans-serif" style:font-family-generic="swiss" style:font-pitch="variable"/>` +
	`</office:font-face-decls>` +
	`<office:styles>` +
	`<style:default-style style:family="paragraph">` +
	`<style:paragraph-properties fo:margin-bottom="0.25cm" fo:line-height="115%" fo:text-align="start"/>` +
	`<style:text-properties style:font-name="Arial" fo:font-size="11pt" fo:color="#212529" fo:language="gl" fo:country="ES"/>` +
	`</style:default-style>` +
	// Portada
	`<style:style style:name="Title" style:display-name="Título" style:family="paragraph" style:class="text">` +
	`<style:paragraph-properties fo:text-align="center" fo:margin-top="3.5cm" fo:margin-bottom="0.15cm"/>` +
	`<style:text-properties style:font-name="Arial" fo:font-size="26pt" fo:font-weight="bold" fo:color="#1F3864"/></style:style>` +
	`<style:style style:name="TitleFirst" style:family="paragraph" style:master-page-name="First_Page" style:class="text">` +
	`<style:paragraph-properties fo:text-align="center" fo:margin-top="3.5cm" fo:margin-bottom="0.15cm"/>` +
	`<style:text-properties style:font-name="Arial" fo:font-size="26pt" fo:font-weight="bold" fo:color="#1F3864"/></style:style>` +
	`<style:style style:name="Subtitle" style:display-name="Subtítulo" style:family="paragraph" style:class="text">` +
	`<style:paragraph-properties fo:text-align="center" fo:margin-bottom="0.15cm"/>` +
	`<style:text-properties style:font-name="Arial" fo:font-size="15pt" fo:color="#595959"/></style:style>` +
	`<style:style style:name="CoverFirst" style:family="paragraph" style:master-page-name="First_Page" style:parent-style-name="CoverMeta"/>` +
	`<style:style style:name="HeaderLeft" style:family="paragraph" style:parent-style-name="Header"><style:paragraph-properties fo:text-align="start"/></style:style>` +
	`<style:style style:name="HeaderCenter" style:family="paragraph" style:parent-style-name="Header"><style:paragraph-properties fo:text-align="center"/></style:style>` +
	`<style:style style:name="Header" style:family="paragraph">` +
	`<style:paragraph-properties fo:text-align="end" fo:padding-bottom="0.1cm" fo:border-bottom="0.5pt solid #8EAADB"/>` +
	`<style:text-properties style:font-name="Arial" fo:font-size="9pt" fo:color="#808080"/></style:style>` +
	`<style:style style:name="CoverRule" style:family="paragraph" style:class="text">` +
	`<style:paragraph-properties fo:text-align="center" fo:margin-left="6cm" fo:margin-right="6cm" fo:margin-top="0.2cm" fo:margin-bottom="0.5cm" fo:border-bottom="0.5pt solid #8EAADB"/>` +
	`<style:text-properties fo:font-size="2pt"/></style:style>` +
	`<style:style style:name="CoverOrg" style:family="paragraph" style:class="text">` +
	`<style:paragraph-properties fo:text-align="center" fo:margin-top="0.2cm" fo:margin-bottom="0.08cm"/>` +
	`<style:text-properties style:font-name="Arial" fo:font-size="14pt" fo:font-weight="bold" fo:color="#1F3864"/></style:style>` +
	`<style:style style:name="CoverMeta" style:family="paragraph" style:class="text">` +
	`<style:paragraph-properties fo:text-align="center" fo:margin-bottom="0.08cm"/>` +
	`<style:text-properties style:font-name="Arial" fo:font-size="12pt" fo:color="#595959"/></style:style>` +
	// Encabezados
	`<style:style style:name="Heading1" style:display-name="Encabezado 1" style:family="paragraph" style:class="text" style:default-outline-level="1">` +
	`<style:paragraph-properties fo:break-before="page" fo:text-align="start" fo:margin-top="0.3cm" fo:margin-bottom="0.25cm" fo:padding-bottom="0.08cm" fo:border-bottom="0.5pt solid #8EAADB" fo:keep-with-next="always"/>` +
	`<style:text-properties style:font-name="Arial" fo:font-size="16pt" fo:font-weight="bold" fo:color="#1F3864"/></style:style>` +
	`<style:style style:name="Heading2" style:display-name="Encabezado 2" style:family="paragraph" style:class="text" style:default-outline-level="2">` +
	`<style:paragraph-properties fo:text-align="start" fo:margin-top="0.4cm" fo:margin-bottom="0.15cm" fo:keep-with-next="always"/>` +
	`<style:text-properties style:font-name="Arial" fo:font-size="13pt" fo:font-weight="bold" fo:color="#2E5496"/></style:style>` +
	`<style:style style:name="TaskTitle" style:display-name="Título de tarefa" style:family="paragraph" style:class="text">` +
	`<style:paragraph-properties fo:text-align="start" fo:margin-top="0.25cm" fo:margin-bottom="0.05cm" fo:keep-with-next="always"/>` +
	`<style:text-properties style:font-name="Arial" fo:font-size="12pt" fo:font-weight="bold" fo:color="#212529"/></style:style>` +
	`<style:style style:name="Subheading" style:family="paragraph" style:class="text">` +
	`<style:paragraph-properties fo:text-align="start" fo:margin-top="0.2cm" fo:margin-bottom="0.05cm" fo:keep-with-next="always"/>` +
	`<style:text-properties style:font-name="Arial" fo:font-size="12pt" fo:font-weight="bold" fo:color="#2E5496"/></style:style>` +
	// Corpo, listas, citas
	`<style:style style:name="TextBody" style:display-name="Corpo de texto" style:family="paragraph" style:class="text">` +
	`<style:paragraph-properties fo:text-align="justify" fo:margin-bottom="0.25cm"/>` +
	`<style:text-properties style:font-name="Arial" fo:font-size="11pt" fo:color="#212529"/></style:style>` +
	`<style:style style:name="ListItem" style:display-name="Elemento de lista" style:family="paragraph" style:class="list">` +
	`<style:paragraph-properties fo:text-align="start" fo:margin-left="0.8cm" fo:text-indent="-0.4cm" fo:margin-bottom="0.1cm">` +
	`<style:tab-stops><style:tab-stop style:position="0.8cm"/></style:tab-stops></style:paragraph-properties>` +
	`<style:text-properties style:font-name="Arial" fo:font-size="11pt" fo:color="#212529"/></style:style>` +
	`<style:style style:name="Quote" style:display-name="Cita" style:family="paragraph" style:class="text">` +
	`<style:paragraph-properties fo:text-align="justify" fo:margin-left="0.7cm" fo:margin-top="0.2cm" fo:margin-bottom="0.25cm" fo:padding-left="0.3cm" fo:border-left="0.06cm solid #8EAADB"/>` +
	`<style:text-properties style:font-name="Arial" fo:font-size="11pt" fo:font-style="italic" fo:color="#595959"/></style:style>` +
	// Índice
	`<style:style style:name="TOCEntry" style:family="paragraph" style:class="index">` +
	`<style:paragraph-properties fo:text-align="start" fo:margin-bottom="0.12cm"/>` +
	`<style:text-properties style:font-name="Arial" fo:font-size="11pt" fo:font-weight="bold" fo:color="#212529"/></style:style>` +
	`<style:style style:name="TOCSub" style:family="paragraph" style:class="index">` +
	`<style:paragraph-properties fo:text-align="start" fo:margin-left="1cm" fo:margin-bottom="0.06cm"/>` +
	`<style:text-properties style:font-name="Arial" fo:font-size="11pt" fo:color="#595959"/></style:style>` +
	// Figuras, pés, sinatura
	`<style:style style:name="Figure" style:family="paragraph" style:class="text">` +
	`<style:paragraph-properties fo:text-align="center" fo:margin-top="0.3cm" fo:margin-bottom="0.1cm"/></style:style>` +
	`<style:style style:name="FigureCover" style:family="paragraph" style:master-page-name="First_Page" style:class="text">` +
	`<style:paragraph-properties fo:text-align="center" fo:margin-top="0.3cm" fo:margin-bottom="0.1cm"/></style:style>` +
	`<style:style style:name="Caption" style:display-name="Lenda" style:family="paragraph" style:class="text">` +
	`<style:paragraph-properties fo:text-align="center" fo:margin-bottom="0.4cm"/>` +
	`<style:text-properties style:font-name="Arial" fo:font-size="9pt" fo:font-style="italic" fo:color="#808080"/></style:style>` +
	`<style:style style:name="SignName" style:family="paragraph" style:class="text">` +
	`<style:paragraph-properties fo:text-align="center" fo:margin-top="2.4cm" fo:margin-bottom="0.05cm"/>` +
	`<style:text-properties style:font-name="Arial" fo:font-size="14pt" fo:font-weight="bold" fo:color="#1F3864"/></style:style>` +
	`<style:style style:name="SignMeta" style:family="paragraph" style:class="text">` +
	`<style:paragraph-properties fo:text-align="center" fo:margin-bottom="0.05cm"/>` +
	`<style:text-properties style:font-name="Arial" fo:font-size="11pt" fo:color="#595959"/></style:style>` +
	`<style:style style:name="FooterLeft" style:family="paragraph" style:parent-style-name="Footer"><style:paragraph-properties fo:text-align="start"/></style:style>` +
	`<style:style style:name="FooterRight" style:family="paragraph" style:parent-style-name="Footer"><style:paragraph-properties fo:text-align="end"/></style:style>` +
	`<style:style style:name="Footer" style:family="paragraph">` +
	`<style:paragraph-properties fo:text-align="center"/>` +
	`<style:text-properties style:font-name="Arial" fo:font-size="9pt" fo:color="#808080"/></style:style>` +
	// Estilos de texto (en liña)
	`<style:style style:name="Strong" style:family="text"><style:text-properties fo:font-weight="bold"/></style:style>` +
	`<style:style style:name="Emphasis" style:family="text"><style:text-properties fo:font-style="italic"/></style:style>` +
	`<style:style style:name="StrongEmphasis" style:family="text"><style:text-properties fo:font-weight="bold" fo:font-style="italic"/></style:style>` +
	`<style:style style:name="Muted" style:family="text"><style:text-properties fo:font-style="italic" fo:color="#808080"/></style:style>` +
	// Estilo gráfico para as imaxes
	`<style:style style:name="Fr" style:family="graphic">` +
	`<style:graphic-properties text:anchor-type="as-char" svg:x="0cm" svg:y="0cm" style:wrap="none" style:vertical-pos="middle" style:vertical-rel="text" fo:border="none"/></style:style>` +
	// As fórmulas (formulas_ofi.go): a súa liña base casa coa do texto.
	`<style:style style:name="FrFormula" style:family="graphic">` +
	`<style:graphic-properties text:anchor-type="as-char" svg:x="0cm" style:wrap="none" style:vertical-pos="from-top" style:vertical-rel="baseline" fo:border="none"/></style:style>` +
	`</office:styles>` +
	`<office:automatic-styles>` +
	`<style:page-layout style:name="Mpm1">` +
	`<style:page-layout-properties fo:page-width="21cm" fo:page-height="29.7cm" style:print-orientation="portrait" ` +
	`fo:margin-top="2cm" fo:margin-bottom="2cm" fo:margin-left="2cm" fo:margin-right="2cm"/>` +
	`<style:header-style><style:header-footer-properties fo:min-height="0.6cm" fo:margin-bottom="0.4cm"/></style:header-style>` +
	`<style:footer-style><style:header-footer-properties fo:min-height="0.6cm" fo:margin-top="0.4cm"/></style:footer-style>` +
	`</style:page-layout>` +
	`</office:automatic-styles>` +
	`<office:master-styles>` +
	`<style:master-page style:name="First_Page" style:page-layout-name="Mpm1" style:next-style-name="Standard"/>` +
	`<style:master-page style:name="Standard" style:page-layout-name="Mpm1">` +
	`@@CABECEIRA@@` +
	`@@PE@@` +
	`</style:master-page>` +
	`</office:master-styles>` +
	`</office:document-styles>`

const odtMetaXML = `<?xml version="1.0" encoding="UTF-8" standalone="no"?>` +
	`<office:document-meta ` +
	`xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0" ` +
	`xmlns:meta="urn:oasis:names:tc:opendocument:xmlns:meta:1.0" office:version="1.2">` +
	`<office:meta><meta:generator>Lua</meta:generator></office:meta>` +
	`</office:document-meta>`

const odtManifestoCabeceira = `<?xml version="1.0" encoding="UTF-8" standalone="no"?>` +
	`<manifest:manifest xmlns:manifest="urn:oasis:names:tc:opendocument:xmlns:manifest:1.0" manifest:version="1.2">` +
	`<manifest:file-entry manifest:full-path="/" manifest:media-type="application/vnd.oasis.opendocument.text"/>` +
	`<manifest:file-entry manifest:full-path="content.xml" manifest:media-type="text/xml"/>` +
	`<manifest:file-entry manifest:full-path="styles.xml" manifest:media-type="text/xml"/>` +
	`<manifest:file-entry manifest:full-path="meta.xml" manifest:media-type="text/xml"/>`

// caixaOdt constrúe a cabeceira (`<style:header>`) ou o pé (`<style:footer>`)
// cos elementos da plantilla: imaxes (que se engaden ao paquete) e textos, e
// ao pé o número de páxina. `base` é "Header" ou "Footer".
func (c *construtorOdt) caixaOdt(cx caixaOfi, base string) string {
	if base == "Header" && cx.baleira() {
		return ""
	}
	estilo := base
	switch cx.Alin {
	case "esquerda":
		estilo = base + "Left"
	case "centro":
		if base == "Header" {
			estilo = "HeaderCenter"
		}
	case "dereita":
		if base == "Footer" {
			estilo = "FooterRight"
		}
	}
	var b strings.Builder
	primeiro := true
	sep := func() {
		if !primeiro {
			b.WriteString(`<text:s text:c="2"/>`)
		}
		primeiro = false
	}
	for _, e := range cx.Elementos {
		switch e.Tipo {
		case "texto":
			sep()
			b.WriteString(escaparXMLOfi(e.Texto))
		case "imaxe":
			im := c.imaxesPlantilla[e.Nome]
			if im == nil {
				continue
			}
			w, h := tamanoCaixaMM(im, e.AltoMM)
			if o, ok := c.engadirImaxe(im, w/10, h/10); ok {
				sep()
				b.WriteString(marcoOdt(o))
			}
		}
	}
	if base == "Footer" {
		if !primeiro {
			b.WriteString("  ·  ")
		}
		b.WriteString(`<text:page-number text:select-page="current"/>`)
	}
	etq := "style:header"
	if base == "Footer" {
		etq = "style:footer"
	}
	return `<` + etq + `><text:p text:style-name="` + estilo + `">` + b.String() + `</text:p></` + etq + `>`
}

// odtEstilosCon devolve styles.xml co estilo da plantilla: fonte, cores, tamaño
// do corpo, cabeceira e pé (xa construídos en d.cabOdt e d.peOdt).
func odtEstilosCon(d *datosOfimatica) string {
	x := odtEstilosBase
	x = strings.Replace(x, "@@CABECEIRA@@", d.cabOdt, 1)
	x = strings.Replace(x, "@@PE@@", d.peOdt, 1)
	x = strings.ReplaceAll(x, `fo:font-size="11pt"`, fmt.Sprintf(`fo:font-size="%.1fpt"`, d.Estilo.TamanoPt))
	return d.Estilo.aplicarEstiloXML(d.Estilo.odtPaxina(x))
}
