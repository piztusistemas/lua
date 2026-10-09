package main

// Escritor nativo de Word (.docx). O documento apóiase en estilos nomeados
// (definidos en styles.xml): cada parágrafo referencia un estilo semántico
// (Title, Heading1, BodyText...) en vez de levar formato directo, de xeito que
// o resultado é consistente, editable e imprimible sen retoques. Ver
// informe_ofimatica.go.

import (
	"archive/zip"
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

const (
	docxEmuPorPx = 9525
	docxEmuPorMM = 36000
	// A4 (210 mm) con marxes de 25 mm.
	docxColumnaMM  = 160.0
	docxColumnaEMU = int(docxColumnaMM * docxEmuPorMM)
	// Alto máximo dunha figura (unha páxina A4 menos marxes e a lenda).
	docxAltoMaxEMU = 22 * 10 * docxEmuPorMM
)

type imaxeDocx struct {
	relID    string
	ficheiro string // "media/image2.png"
	im       *imaxeOfi
	cx, cy   int // EMU
}

type construtorDocx struct {
	formulas   *formulasOfi // as fórmulas xa compoñidas (formulas_ofi.go)
	corpo      strings.Builder
	imaxes     []imaxeDocx
	baseMedios int // tamaño do corpo en medios-puntos (para [fs=..])
}

// engadirImaxe incorpora unha imaxe ao paquete co tamaño exacto (EMU).
func (c *construtorDocx) engadirImaxe(im *imaxeOfi, cx, cy int) (string, bool) {
	if im == nil || cx <= 0 || cy <= 0 {
		return "", false
	}
	n := len(c.imaxes) + 1
	rel := fmt.Sprintf("rIdImg%d", n)
	c.imaxes = append(c.imaxes, imaxeDocx{relID: rel, ficheiro: fmt.Sprintf("media/image%d.%s", n, im.Ext), im: im, cx: cx, cy: cy})
	return rel, true
}

// engadirImaxeNatural usa o tamaño natural (96 dpi) limitado ao ancho da
// columna e a `maxAlto` EMU (0 = sen límite de alto).
func (c *construtorDocx) engadirImaxeNatural(im *imaxeOfi, maxAlto int) (rel string, cx, cy int, ok bool) {
	cx, cy = im.W*docxEmuPorPx, im.H*docxEmuPorPx
	if cx > docxColumnaEMU {
		cy = cy * docxColumnaEMU / cx
		cx = docxColumnaEMU
	}
	if maxAlto > 0 && cy > maxAlto {
		cx = cx * maxAlto / cy
		cy = maxAlto
	}
	rel, ok = c.engadirImaxe(im, cx, cy)
	return rel, cx, cy, ok
}

// portada escribe a portada segundo as liñas da plantilla (ver
// plantillas_ofimatica.go): logotipo do curso, imaxes da plantilla, títulos...
func (c *construtorDocx) portada(d *datosOfimatica) {
	for _, l := range d.Portada {
		switch l.Tipo {
		case "logo", "imaxe":
			im := d.Logo
			if l.Tipo == "imaxe" {
				im = d.ImaxesPlantilla[l.Nome]
			}
			if im == nil {
				continue
			}
			rel, cx, cy, ok := c.engadirImaxeNatural(im, 0)
			if !ok {
				continue
			}
			// Con alto explícito (@logo 30) a imaxe ten EXACTAMENTE ese alto;
			// sen el, o seu tamaño natural con tope de 35 mm.
			maxH := 35 * docxEmuPorMM
			if l.AltoMM > 0 {
				cy = int(l.AltoMM * docxEmuPorMM)
				cx = cy * im.W / im.H
				maxH = cy
			}
			if cx > docxColumnaEMU {
				cy = cy * docxColumnaEMU / cx
				cx = docxColumnaEMU
			}
			if cy > maxH {
				cx = cx * maxH / cy
				cy = maxH
			}
			c.imaxes[len(c.imaxes)-1].cx, c.imaxes[len(c.imaxes)-1].cy = cx, cy
			c.corpo.WriteString(paraImaxeDocx(rel, cx, cy, cronoAlinDefecto))
		case "espazo":
			twips := int(max(l.AltoMM, 1) * 56.7)
			c.corpo.WriteString(paraDocx("Normal", `<w:spacing w:before="`+strconv.Itoa(twips)+`" w:after="0"/>`, runDocx("", false, false, "", 0)))
		case "titulo":
			c.texto("Title", l.Texto)
		case "subtitulo":
			c.texto("Subtitle", l.Texto)
		case "regra":
			c.texto("CoverRule", "")
		case "organizacion":
			c.texto("CoverOrg", l.Texto)
		default:
			c.texto("CoverMeta", l.Texto)
		}
	}
}

func (c *construtorDocx) p(estilo string, runs ...string) {
	c.corpo.WriteString(paraDocx(estilo, "", runs...))
}
func (c *construtorDocx) pSalto(estilo string, runs ...string) {
	c.corpo.WriteString(paraDocx(estilo, "<w:pageBreakBefore/>", runs...))
}
func (c *construtorDocx) texto(estilo, t string) { c.p(estilo, runDocx(t, false, false, "", 0)) }
func (c *construtorDocx) textoSalto(estilo, t string) {
	c.pSalto(estilo, runDocx(t, false, false, "", 0))
}

// escribirDocxCrono compón o informe completo e devolve os bytes do .docx.
func escribirDocxCrono(d *datosOfimatica) ([]byte, error) {
	c := &construtorDocx{baseMedios: int(d.Estilo.TamanoPt*2 + 0.5)}

	// --- Portada (do deseño da plantilla) ---
	c.portada(d)

	// --- Índice ---
	c.textoSalto("Heading1", d.Tx.T("INDICE"))
	c.texto("TOCEntry", "1. "+d.Tx.T("INTRODUCION"))
	c.texto("TOCEntry", "2. "+d.Tx.T("CRONOGRAMA"))
	c.texto("TOCEntry", "3. "+d.Tx.T("DESENVOLVEMENTO"))
	for i := range d.ConTarefas {
		c.texto("TOCSub", fmt.Sprintf("3.%d. %s", i+1, d.Nomes[i]))
	}
	c.texto("TOCEntry", "4. "+d.Tx.T("SINATURA_TITULO"))

	// --- 1. Introdución ---
	c.textoSalto("Heading1", "1. "+d.Tx.T("INTRODUCION"))
	for _, par := range paragrafosOfi(d.Intro) {
		c.texto("BodyText", par)
	}

	// --- 2. Cronograma ---
	c.textoSalto("Heading1", "2. "+d.Tx.T("CRONOGRAMA"))
	if im, err := imaxeOfiDeBytes(d.CronoPNG); err == nil {
		if rel, cx, cy, ok := c.engadirImaxeNatural(im, docxAltoMaxEMU); ok {
			c.corpo.WriteString(paraImaxeDocx(rel, cx, cy, cronoAlinDefecto))
		}
	}

	// --- 3. Desenvolvemento por meses ---
	c.textoSalto("Heading1", "3. "+d.Tx.T("DESENVOLVEMENTO"))
	if len(d.ConTarefas) == 0 {
		c.p("BodyText", runDocx(d.Tx.T("SEN_TAREFAS_CURSO"), false, true, "808080", 0))
	}
	for i, m := range d.ConTarefas {
		c.texto("Heading2", fmt.Sprintf("3.%d. %s", i+1, d.Nomes[i]))
		for _, t := range m.Tarefas {
			c.texto("TaskTitle", porDefectoCrono(t.Title, d.Tx.T("SEN_TITULO")))
			bloques := parseMarkdownCrono(t.Description)
			if len(bloques) == 0 {
				c.p("BodyText", runDocx(d.Tx.T("SEN_DESCRICION"), false, true, "808080", 0))
			} else {
				c.bloques(bloques, t.Images, &d.Avisos)
			}
		}
	}

	// --- 4. Sinatura ---
	c.textoSalto("Heading1", "4. "+d.Tx.T("SINATURA_TITULO"))
	c.texto("BodyText", d.Tx.T("SINATURA_TEXTO"))
	c.texto("SignName", d.Xefe)
	c.texto("SignMeta", d.Cargo)
	c.texto("SignMeta", d.Tx.T("LUGAR_DATA"))

	return montarDocx(c.corpo.String(), c.imaxes, d)
}

// paragrafosOfi divide un texto en parágrafos (liñas en branco), para que unha
// introdución en varios parágrafos non quede nun bloque único.
func paragrafosOfi(texto string) []string {
	var out []string
	for _, par := range strings.Split(strings.ReplaceAll(texto, "\r\n", "\n"), "\n\n") {
		if par = strings.TrimSpace(strings.ReplaceAll(par, "\n", " ")); par != "" {
			out = append(out, par)
		}
	}
	if len(out) == 0 {
		out = []string{""}
	}
	return out
}

// bloques renderiza os bloques da descrición dunha tarefa.
func (c *construtorDocx) bloques(bloques []cronoBloque, imaxes map[string]string, avisos *[]string) {
	orde := 0
	for _, bl := range bloques {
		if bl.Tipo != cronoItem || !bl.Ordenada {
			orde = 0
		}
		switch bl.Tipo {
		case cronoTitulo:
			c.corpo.WriteString(paraDocx("Subheading", propAlinDocx(bl.Alin), runDocx(textoTramosOfi(bl.Tramos), false, false, "", 0)))
		case cronoItem:
			marca := "•\t"
			if bl.Ordenada {
				orde++
				marca = fmt.Sprintf("%d.\t", orde)
			}
			runs := append([]string{runDocx(marca, false, false, "", 0)}, c.runsTramos(bl.Tramos, false)...)
			c.corpo.WriteString(paraDocx("ListBullet", propAlinDocx(bl.Alin), runs...))
		case cronoCita:
			c.corpo.WriteString(paraDocx("Quote", propAlinDocx(bl.Alin), c.runsTramos(bl.Tramos, true)...))
		case cronoDebuxo:
			im, w, h, ok := c.formulas.debuxo(bl)
			if !ok {
				continue
			}
			cx, cy := int(w*914400), int(h*914400)
			if cx > docxColumnaEMU {
				cy = cy * docxColumnaEMU / cx
				cx = docxColumnaEMU
			}
			if rel, ok := c.engadirImaxe(im, cx, cy); ok {
				c.corpo.WriteString(paraImaxeDocx(rel, cx, cy, cronoAlinDefecto))
			}
		case cronoImaxe:
			im := resolverImaxeOfi(bl.Imaxe.Ref, imaxes, avisos)
			if im == nil {
				continue
			}
			w, h := bl.Imaxe.tamanoMM(docxColumnaMM, im.W, im.H)
			cx, cy := int(w*docxEmuPorMM), int(h*docxEmuPorMM)
			if rel, ok := c.engadirImaxe(im, cx, cy); ok {
				c.corpo.WriteString(paraImaxeDocx(rel, cx, cy, alinImaxeOfi(bl.Alin)))
			}
		default:
			c.corpo.WriteString(paraDocx("BodyText", propAlinDocx(bl.Alin), c.runsTramos(bl.Tramos, false)...))
		}
	}
}

// ---------- XML ----------

// runDocx crea un run que herda tipo de letra e tamaño do estilo do parágrafo;
// só engade negriña/cursiva/cor/tamaño (en medios-puntos) cando cómpre.
func runDocx(texto string, negrita, cursiva bool, cor string, mediosPt int) string {
	var rpr strings.Builder
	if negrita || cursiva || cor != "" || mediosPt > 0 {
		rpr.WriteString("<w:rPr>")
		if negrita {
			rpr.WriteString("<w:b/>")
		}
		if cursiva {
			rpr.WriteString("<w:i/>")
		}
		if cor != "" {
			rpr.WriteString(`<w:color w:val="` + cor + `"/>`)
		}
		if mediosPt > 0 {
			sz := strconv.Itoa(mediosPt)
			rpr.WriteString(`<w:sz w:val="` + sz + `"/><w:szCs w:val="` + sz + `"/>`)
		}
		rpr.WriteString("</w:rPr>")
	}
	return "<w:r>" + rpr.String() + `<w:t xml:space="preserve">` + escaparXMLOfi(texto) + "</w:t></w:r>"
}

func runsTramosDocx(tramos []cronoTramo, forzarCursiva bool, baseMedios int) []string {
	runs := make([]string, 0, len(tramos))
	for _, t := range tramos {
		txt := t.cronoTexto()
		if txt == "" {
			continue
		}
		mediosPt := 0
		if t.Escala > 0 && t.Escala != 1 {
			mediosPt = int(float64(baseMedios)*t.Escala + 0.5)
		}
		runs = append(runs, runDocx(txt, t.Negrita, t.Cursiva || forzarCursiva, "", mediosPt))
	}
	return runs
}

// propAlinDocx traduce un aliñamento a <w:jc>. O por defecto non emite nada
// para que mande o estilo do parágrafo.
func propAlinDocx(a cronoAlinamento) string {
	switch a {
	case cronoAlinEsquerda:
		return `<w:jc w:val="left"/>`
	case cronoAlinCentro:
		return `<w:jc w:val="center"/>`
	case cronoAlinDereita:
		return `<w:jc w:val="right"/>`
	case cronoAlinXustificado:
		return `<w:jc w:val="both"/>`
	}
	return ""
}

func paraDocx(estilo, extraPr string, runs ...string) string {
	var b strings.Builder
	b.WriteString(`<w:p><w:pPr><w:pStyle w:val="` + estilo + `"/>` + extraPr + `</w:pPr>`)
	for _, r := range runs {
		b.WriteString(r)
	}
	b.WriteString("</w:p>")
	return b.String()
}

// paraImaxeDocx emite o parágrafo dunha imaxe. O estilo Figure xa centra, así
// que só se engade <w:jc> para esquerda ou dereita.
func paraImaxeDocx(rel string, cx, cy int, a cronoAlinamento) string {
	id := strings.TrimPrefix(rel, "rIdImg")
	return `<w:p><w:pPr><w:pStyle w:val="Figure"/>` + propAlinDocx(alinImaxeOfi(a)) + `</w:pPr>` + runImaxeDocx(rel, id, cx, cy) + `</w:p>`
}

// runImaxeDocx é a imaxe en liña (un run); `id` é o identificador numérico
// (único no documento) do obxecto de debuxo.
func runImaxeDocx(rel, id string, cx, cy int) string {
	plantilla := `<w:r><w:drawing>` +
		`<wp:inline distT="0" distB="0" distL="0" distR="0">` +
		`<wp:extent cx="%d" cy="%d"/>` +
		`<wp:docPr id="%s" name="Imaxe%s"/>` +
		`<a:graphic xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">` +
		`<a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/picture">` +
		`<pic:pic xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture">` +
		`<pic:nvPicPr><pic:cNvPr id="%s" name="Imaxe%s"/><pic:cNvPicPr/></pic:nvPicPr>` +
		`<pic:blipFill><a:blip r:embed="%s"/><a:stretch><a:fillRect/></a:stretch></pic:blipFill>` +
		`<pic:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="%d" cy="%d"/></a:xfrm>` +
		`<a:prstGeom prst="rect"><a:avLst/></a:prstGeom></pic:spPr>` +
		`</pic:pic></a:graphicData></a:graphic></wp:inline></w:drawing></w:r>`
	return fmt.Sprintf(plantilla, cx, cy, id, id, id, id, rel, cx, cy)
}

// tamanoCaixaMM calcula o tamaño (mm) dunha imaxe de cabeceira ou pé: o alto
// pedido (10 mm por defecto) e o ancho segundo a proporción, con tope de 120 mm.
func tamanoCaixaMM(im *imaxeOfi, altoMM float64) (w, h float64) {
	if altoMM <= 0 {
		altoMM = 10
	}
	h = altoMM
	w = h * float64(im.W) / float64(max(im.H, 1))
	if w > 120 {
		h, w = h*120/w, 120
	}
	return w, h
}

// caixaDocx constrúe o parágrafo dunha cabeceira ou pé: as imaxes e os textos
// en orde, nunha soa liña; ao pé (`numero`) engádese o número de páxina.
// `prefixo` distingue as imaxes desta peza no paquete ("hdr" ou "ftr").
func caixaDocx(c caixaOfi, imaxes map[string]*imaxeOfi, prefixo, estilo string, numero bool) (string, []imaxeDocx) {
	var runs strings.Builder
	var ims []imaxeDocx
	primeiro := true
	sep := func() {
		if !primeiro {
			runs.WriteString(`<w:r><w:t xml:space="preserve">  </w:t></w:r>`)
		}
		primeiro = false
	}
	for _, e := range c.Elementos {
		switch e.Tipo {
		case "texto":
			sep()
			runs.WriteString(runDocx(e.Texto, false, false, "", 0))
		case "imaxe":
			im := imaxes[e.Nome]
			if im == nil {
				continue
			}
			w, h := tamanoCaixaMM(im, e.AltoMM)
			n := len(ims) + 1
			rel := fmt.Sprintf("rId%s%d", strings.ToUpper(prefixo), n)
			ims = append(ims, imaxeDocx{relID: rel, ficheiro: fmt.Sprintf("media/%s%d.%s", prefixo, n, im.Ext), im: im, cx: int(w * docxEmuPorMM), cy: int(h * docxEmuPorMM)})
			sep()
			base := 1000
			if prefixo == "ftr" {
				base = 2000
			}
			runs.WriteString(runImaxeDocx(rel, strconv.Itoa(base+n), int(w*docxEmuPorMM), int(h*docxEmuPorMM)))
		}
	}
	if numero {
		if !primeiro {
			runs.WriteString(`<w:r><w:t xml:space="preserve">  ·  </w:t></w:r>`)
		}
		runs.WriteString(`<w:r><w:fldChar w:fldCharType="begin"/></w:r><w:r><w:instrText xml:space="preserve"> PAGE </w:instrText></w:r>` +
			`<w:r><w:fldChar w:fldCharType="separate"/></w:r><w:r><w:t>1</w:t></w:r><w:r><w:fldChar w:fldCharType="end"/></w:r>`)
	}
	jc := map[string]string{"esquerda": "left", "centro": "center", "dereita": "right"}[c.Alin]
	return `<w:p><w:pPr><w:pStyle w:val="` + estilo + `"/><w:jc w:val="` + jc + `"/></w:pPr>` + runs.String() + `</w:p>`, ims
}

// relsImaxesDocx é o ficheiro de relacións das imaxes dunha cabeceira ou pé.
func relsImaxesDocx(ims []imaxeDocx) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`)
	for _, im := range ims {
		b.WriteString(fmt.Sprintf(`<Relationship Id="%s" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="%s"/>`, im.relID, im.ficheiro))
	}
	b.WriteString(`</Relationships>`)
	return b.String()
}

func escaparXMLOfi(s string) string {
	// Os caracteres de control (agás tab e saltos) non son válidos en XML 1.0.
	s = strings.Map(func(r rune) rune {
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' {
			return -1
		}
		return r
	}, s)
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;").Replace(s)
}

// ---------- Paquete ZIP ----------

func montarDocx(corpo string, imaxes []imaxeDocx, d *datosOfimatica) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	escribir := func(nome, contido string) error {
		w, err := zw.Create(nome)
		if err != nil {
			return err
		}
		_, err = w.Write([]byte(contido))
		return err
	}

	cabeceira := !d.Cabeceira.baleira()
	tipos := docxContentTypes
	if cabeceira {
		tipos = strings.Replace(tipos, `</Types>`,
			`<Override PartName="/word/header1.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.header+xml"/>`+
				`<Override PartName="/word/header2.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.header+xml"/></Types>`, 1)
	}
	if err := escribir("[Content_Types].xml", tipos); err != nil {
		return nil, err
	}
	if err := escribir("_rels/.rels", docxRelsRaiz); err != nil {
		return nil, err
	}

	var rels strings.Builder
	rels.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
		`<Relationship Id="rIdStyles" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>` +
		`<Relationship Id="rIdFooter1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/footer" Target="footer1.xml"/>` +
		`<Relationship Id="rIdFooter2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/footer" Target="footer2.xml"/>`)
	if cabeceira {
		rels.WriteString(`<Relationship Id="rIdHeader1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/header" Target="header1.xml"/>` +
			`<Relationship Id="rIdHeader2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/header" Target="header2.xml"/>`)
	}
	for _, im := range imaxes {
		rels.WriteString(fmt.Sprintf(
			`<Relationship Id="%s" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="%s"/>`,
			im.relID, im.ficheiro))
	}
	rels.WriteString(`</Relationships>`)
	sect := d.Estilo.docxPagina(docxPeDoc)
	// O pé ten sempre o número de páxina; a cabeceira só se a plantilla a define.
	parPe, imsPe := caixaDocx(d.Pe, d.ImaxesPlantilla, "ftr", "Footer", true)
	partes := [][2]string{
		{"word/_rels/document.xml.rels", rels.String()},
		{"word/styles.xml", d.Estilo.aplicarEstiloXML(d.Estilo.docxInterlinado(docxEstilosXML(d.Estilo)))},
		{"word/footer1.xml", docxContedorPe(parPe)},
		{"word/footer2.xml", docxPeBaleiroXML},
	}
	imaxesCaixas := imsPe
	if len(imsPe) > 0 {
		partes = append(partes, [2]string{"word/_rels/footer1.xml.rels", relsImaxesDocx(imsPe)})
	}
	if cabeceira {
		parCab, imsCab := caixaDocx(d.Cabeceira, d.ImaxesPlantilla, "hdr", "Header", false)
		sect = strings.Replace(sect, `<w:footerReference w:type="default"`, `<w:headerReference w:type="default" r:id="rIdHeader1"/><w:headerReference w:type="first" r:id="rIdHeader2"/><w:footerReference w:type="default"`, 1)
		partes = append(partes,
			[2]string{"word/header1.xml", docxContedorCabeceira(parCab)},
			[2]string{"word/header2.xml", docxContedorCabeceira(`<w:p/>`)})
		if len(imsCab) > 0 {
			partes = append(partes, [2]string{"word/_rels/header1.xml.rels", relsImaxesDocx(imsCab)})
		}
		imaxesCaixas = append(imaxesCaixas, imsCab...)
	}
	partes = append(partes, [2]string{"word/document.xml", docxCabeceiraDoc + corpo + sect})
	for _, par := range partes {
		if err := escribir(par[0], par[1]); err != nil {
			return nil, err
		}
	}
	for _, im := range append(append([]imaxeDocx{}, imaxes...), imaxesCaixas...) {
		w, err := zw.Create("word/" + im.ficheiro)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(im.im.Datos); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

const docxContentTypes = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
	`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
	`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
	`<Default Extension="xml" ContentType="application/xml"/>` +
	`<Default Extension="png" ContentType="image/png"/>` +
	`<Default Extension="jpg" ContentType="image/jpeg"/>` +
	`<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>` +
	`<Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>` +
	`<Override PartName="/word/footer1.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.footer+xml"/>` +
	`<Override PartName="/word/footer2.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.footer+xml"/>` +
	`</Types>`

const docxRelsRaiz = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
	`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
	`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>` +
	`</Relationships>`

const docxCabeceiraDoc = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
	`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" ` +
	`xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" ` +
	`xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing" ` +
	`xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" ` +
	`xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture">` +
	`<w:body>`

// O sectPr referencia o pé de páxina (numeración) e activa titlePg para que a
// portada non leve número.
const docxPeDoc = `<w:sectPr>` +
	`<w:footerReference w:type="default" r:id="rIdFooter1"/>` +
	`<w:footerReference w:type="first" r:id="rIdFooter2"/>` +
	`<w:pgSz w:w="11906" w:h="16838"/>` +
	`<w:pgMar w:top="1418" w:right="1418" w:bottom="1418" w:left="1418" w:header="709" w:footer="709" w:gutter="0"/>` +
	`<w:titlePg/>` +
	`</w:sectPr></w:body></w:document>`

const docxPeXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
	`<w:ftr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" ` +
	`xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">` +
	`<w:p><w:pPr><w:pStyle w:val="Footer"/></w:pPr>` +
	`<w:r><w:fldChar w:fldCharType="begin"/></w:r>` +
	`<w:r><w:instrText xml:space="preserve"> PAGE </w:instrText></w:r>` +
	`<w:r><w:fldChar w:fldCharType="separate"/></w:r>` +
	`<w:r><w:t>1</w:t></w:r>` +
	`<w:r><w:fldChar w:fldCharType="end"/></w:r>` +
	`</w:p></w:ftr>`

// docxContedorPe e docxContedorCabeceira envolven o parágrafo xa construído.
func docxContedorPe(par string) string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<w:ftr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" ` +
		`xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" ` +
		`xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing" ` +
		`xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" ` +
		`xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture">` + par + `</w:ftr>`
}

func docxContedorCabeceira(par string) string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<w:hdr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" ` +
		`xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" ` +
		`xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing" ` +
		`xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" ` +
		`xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture">` + par + `</w:hdr>`
}

const docxPeBaleiroXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
	`<w:ftr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` +
	`<w:p><w:pPr><w:pStyle w:val="Footer"/></w:pPr></w:p></w:ftr>`

// docxEstilosXML define a folla de estilos nomeados: toda a tipografía e cor
// do informe está controlada desde aquí (editando un estilo en Word cambia
// todo o documento).
const docxEstilosBase = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
	`<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` +
	`<w:docDefaults><w:rPrDefault><w:rPr>` +
	`<w:rFonts w:ascii="Arial" w:hAnsi="Arial" w:cs="Arial"/>` +
	`<w:color w:val="212529"/><w:sz w:val="22"/><w:szCs w:val="22"/><w:lang w:val="gl-ES"/>` +
	`</w:rPr></w:rPrDefault>` +
	`<w:pPrDefault><w:pPr><w:spacing w:after="160" w:line="276" w:lineRule="auto"/></w:pPr></w:pPrDefault>` +
	`</w:docDefaults>` +
	// Normal (base)
	`<w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/></w:style>` +
	// Portada
	`<w:style w:type="paragraph" w:styleId="Title"><w:name w:val="Title"/><w:basedOn w:val="Normal"/>` +
	`<w:pPr><w:spacing w:before="2200" w:after="120"/><w:jc w:val="center"/></w:pPr>` +
	`<w:rPr><w:b/><w:color w:val="1F3864"/><w:sz w:val="60"/><w:szCs w:val="60"/></w:rPr></w:style>` +
	`<w:style w:type="paragraph" w:styleId="Subtitle"><w:name w:val="Subtitle"/><w:basedOn w:val="Normal"/>` +
	`<w:pPr><w:spacing w:after="120"/><w:jc w:val="center"/></w:pPr>` +
	`<w:rPr><w:color w:val="595959"/><w:sz w:val="30"/><w:szCs w:val="30"/></w:rPr></w:style>` +
	`<w:style w:type="paragraph" w:styleId="CoverRule"><w:name w:val="Cover Rule"/><w:basedOn w:val="Normal"/>` +
	`<w:pPr><w:spacing w:before="120" w:after="280"/><w:jc w:val="center"/>` +
	`<w:pBdr><w:bottom w:val="single" w:sz="6" w:space="1" w:color="8EAADB"/></w:pBdr>` +
	`<w:ind w:left="2600" w:right="2600"/></w:pPr>` +
	`<w:rPr><w:sz w:val="2"/><w:szCs w:val="2"/></w:rPr></w:style>` +
	`<w:style w:type="paragraph" w:styleId="CoverOrg"><w:name w:val="Cover Org"/><w:basedOn w:val="Normal"/>` +
	`<w:pPr><w:spacing w:before="120" w:after="40"/><w:jc w:val="center"/></w:pPr>` +
	`<w:rPr><w:b/><w:color w:val="1F3864"/><w:sz w:val="28"/><w:szCs w:val="28"/></w:rPr></w:style>` +
	`<w:style w:type="paragraph" w:styleId="CoverMeta"><w:name w:val="Cover Meta"/><w:basedOn w:val="Normal"/>` +
	`<w:pPr><w:spacing w:after="40"/><w:jc w:val="center"/></w:pPr>` +
	`<w:rPr><w:color w:val="595959"/><w:sz w:val="24"/><w:szCs w:val="24"/></w:rPr></w:style>` +
	// Encabezados
	`<w:style w:type="paragraph" w:styleId="Heading1"><w:name w:val="heading 1"/><w:basedOn w:val="Normal"/><w:next w:val="BodyText"/>` +
	`<w:pPr><w:keepNext/><w:spacing w:before="360" w:after="160"/><w:outlineLvl w:val="0"/>` +
	`<w:pBdr><w:bottom w:val="single" w:sz="6" w:space="6" w:color="8EAADB"/></w:pBdr></w:pPr>` +
	`<w:rPr><w:b/><w:color w:val="1F3864"/><w:sz w:val="32"/><w:szCs w:val="32"/></w:rPr></w:style>` +
	`<w:style w:type="paragraph" w:styleId="Heading2"><w:name w:val="heading 2"/><w:basedOn w:val="Normal"/><w:next w:val="BodyText"/>` +
	`<w:pPr><w:keepNext/><w:spacing w:before="240" w:after="100"/><w:outlineLvl w:val="1"/></w:pPr>` +
	`<w:rPr><w:b/><w:color w:val="2E5496"/><w:sz w:val="26"/><w:szCs w:val="26"/></w:rPr></w:style>` +
	`<w:style w:type="paragraph" w:styleId="TaskTitle"><w:name w:val="Task Title"/><w:basedOn w:val="Normal"/><w:next w:val="BodyText"/>` +
	`<w:pPr><w:keepNext/><w:spacing w:before="160" w:after="40"/><w:outlineLvl w:val="2"/></w:pPr>` +
	`<w:rPr><w:b/><w:color w:val="212529"/><w:sz w:val="24"/><w:szCs w:val="24"/></w:rPr></w:style>` +
	`<w:style w:type="paragraph" w:styleId="Subheading"><w:name w:val="Subheading"/><w:basedOn w:val="Normal"/><w:next w:val="BodyText"/>` +
	`<w:pPr><w:keepNext/><w:spacing w:before="120" w:after="40"/></w:pPr>` +
	`<w:rPr><w:b/><w:color w:val="2E5496"/><w:sz w:val="24"/><w:szCs w:val="24"/></w:rPr></w:style>` +
	// Corpo e listas
	`<w:style w:type="paragraph" w:styleId="BodyText"><w:name w:val="Body Text"/><w:basedOn w:val="Normal"/>` +
	`<w:pPr><w:spacing w:after="160"/><w:jc w:val="both"/></w:pPr></w:style>` +
	`<w:style w:type="paragraph" w:styleId="ListBullet"><w:name w:val="List Bullet"/><w:basedOn w:val="BodyText"/>` +
	`<w:pPr><w:spacing w:after="60"/><w:jc w:val="left"/><w:ind w:left="567" w:hanging="284"/>` +
	`<w:tabs><w:tab w:val="left" w:pos="567"/></w:tabs></w:pPr></w:style>` +
	`<w:style w:type="paragraph" w:styleId="Quote"><w:name w:val="Quote"/><w:basedOn w:val="BodyText"/>` +
	`<w:pPr><w:spacing w:before="120" w:after="160"/><w:ind w:left="567"/>` +
	`<w:pBdr><w:left w:val="single" w:sz="18" w:space="8" w:color="8EAADB"/></w:pBdr></w:pPr>` +
	`<w:rPr><w:i/><w:color w:val="595959"/></w:rPr></w:style>` +
	// Índice
	`<w:style w:type="paragraph" w:styleId="TOCEntry"><w:name w:val="TOC Entry"/><w:basedOn w:val="Normal"/>` +
	`<w:pPr><w:spacing w:after="80"/></w:pPr><w:rPr><w:b/></w:rPr></w:style>` +
	`<w:style w:type="paragraph" w:styleId="TOCSub"><w:name w:val="TOC Sub"/><w:basedOn w:val="Normal"/>` +
	`<w:pPr><w:spacing w:after="40"/><w:ind w:left="567"/></w:pPr><w:rPr><w:color w:val="595959"/></w:rPr></w:style>` +
	// Figuras e pés
	`<w:style w:type="paragraph" w:styleId="Figure"><w:name w:val="Figure"/><w:basedOn w:val="Normal"/>` +
	`<w:pPr><w:spacing w:before="160" w:after="80"/><w:jc w:val="center"/></w:pPr></w:style>` +
	`<w:style w:type="paragraph" w:styleId="Caption"><w:name w:val="Caption"/><w:basedOn w:val="Normal"/>` +
	`<w:pPr><w:spacing w:after="200"/><w:jc w:val="center"/></w:pPr>` +
	`<w:rPr><w:i/><w:color w:val="808080"/><w:sz w:val="18"/><w:szCs w:val="18"/></w:rPr></w:style>` +
	// Sinatura
	`<w:style w:type="paragraph" w:styleId="SignName"><w:name w:val="Sign Name"/><w:basedOn w:val="Normal"/>` +
	`<w:pPr><w:spacing w:before="1400" w:after="40"/><w:jc w:val="center"/></w:pPr>` +
	`<w:rPr><w:b/><w:color w:val="1F3864"/><w:sz w:val="28"/><w:szCs w:val="28"/></w:rPr></w:style>` +
	`<w:style w:type="paragraph" w:styleId="SignMeta"><w:name w:val="Sign Meta"/><w:basedOn w:val="Normal"/>` +
	`<w:pPr><w:spacing w:after="40"/><w:jc w:val="center"/></w:pPr><w:rPr><w:color w:val="595959"/></w:rPr></w:style>` +
	// Pé de páxina
	`<w:style w:type="paragraph" w:styleId="Footer"><w:name w:val="footer"/><w:basedOn w:val="Normal"/>` +
	`<w:pPr><w:spacing w:after="0" w:line="240" w:lineRule="auto"/><w:jc w:val="center"/></w:pPr>` +
	`<w:rPr><w:color w:val="808080"/><w:sz w:val="18"/><w:szCs w:val="18"/></w:rPr></w:style>` +
	// Cabeceira
	`<w:style w:type="paragraph" w:styleId="Header"><w:name w:val="header"/><w:basedOn w:val="Normal"/>` +
	`<w:pPr><w:spacing w:after="0" w:line="240" w:lineRule="auto"/><w:jc w:val="right"/>` +
	`<w:pBdr><w:bottom w:val="single" w:sz="4" w:space="4" w:color="8EAADB"/></w:pBdr></w:pPr>` +
	`<w:rPr><w:color w:val="808080"/><w:sz w:val="18"/><w:szCs w:val="18"/></w:rPr></w:style>` +
	`</w:styles>`

// docxEstilosXML devolve a folla de estilos co tamaño do corpo da plantilla
// (o resto de valores, fonte e cores, cámbianse con aplicarEstiloXML).
func docxEstilosXML(e estiloOfi) string {
	medios := strconv.Itoa(int(e.TamanoPt*2 + 0.5))
	return strings.Replace(docxEstilosBase,
		`<w:color w:val="212529"/><w:sz w:val="22"/><w:szCs w:val="22"/>`,
		`<w:color w:val="212529"/><w:sz w:val="`+medios+`"/><w:szCs w:val="`+medios+`"/>`, 1)
}
