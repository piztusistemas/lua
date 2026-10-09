package main

// Word (.docx) e OpenDocument (.odt) dos formularios propios. Usan os mesmos
// bloques ca o PDF e o Markdown (formulario_bloques.go) e a mesma maquinaria
// que o cronograma e as actas (informe_ofimatica.go): a folla de estilos, a
// portada, a cabeceira e o pé veñen do bloque «Word e OpenDocument» da
// PLANTILLA; Lua só pon o contido.
//
// Os formularios traen unha cousa que o cronograma e as actas non tiñan:
// TÁBOAS (docx: w:tbl; odt: table:table) e listas de datos.

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// docFormulario é o formulario xa resolto para os escritores de Word e ODT.
type docFormulario struct {
	Titulo            string
	Portada, Sinatura bool
	Bloques           []bloqueDoc
}

// prepararOfimaticaFormulario resolve todo o que precisan os escritores DOCX/ODT
// para un formulario: variables, textos, bloques e o deseño da plantilla.
func prepararOfimaticaFormulario(req XerarInformeCronogramaRequest, plantilla *PlantillaLatex, agora time.Time) (datosOfimatica, error) {
	f, dat := normalizarFormulario(req.Formulario.Formulario), req.Formulario.Datos
	if err := validarFormulario(f); err != nil {
		return datosOfimatica{}, fmt.Errorf("formulario non válido: %w", err)
	}
	vars := variablesFormulario(f, dat, req.NomeBase, agora, plantilla)
	tx := novosTextosDoc(plantilla, vars)
	tx.engadirVariables(vars, "TITULO_FORMULARIO")
	bloques, avisos := bloquesDocumentoFormulario(*req.Formulario, f, dat, tx)
	d := baseOfimatica(plantilla, vars, tx, dat.Contexto.Logo, avisos)
	d.Centro = vars["CENTRO"]
	d.Localidade = vars["LOCALIDADE"]
	d.Curso = vars["CURSO"]
	d.Data = vars["DATA_LONGA"]
	d.Xefe = vars["XEFE"]
	d.Cargo = vars["CARGO"]
	d.Departamento = vars["DEPARTAMENTO"]
	d.Formulario = &docFormulario{
		Titulo: vars["TITULO"], Portada: f.Portada, Sinatura: f.Sinatura, Bloques: bloques,
	}
	fs, err := compoñerFormulas(formulasDeBloques(bloques), d.Estilo.TamanoPt)
	if err != nil {
		d.Avisos = engadirSenRepetir(d.Avisos, []string{err.Error() + ": as fórmulas van coma texto"})
	}
	if err := fs.engadirDebuxos(bloques); err != nil {
		d.Avisos = engadirSenRepetir(d.Avisos, []string{err.Error()})
	}
	d.Formulas = fs
	return d, nil
}

// tintarHex mestura unha cor hexadecimal (sen #) con branco: k=0 é a cor, k=1 é branco.
func tintarHex(hex string, k float64) string {
	if len(hex) != 6 {
		return "EEF2FF"
	}
	var out strings.Builder
	for i := 0; i < 6; i += 2 {
		v, err := strconv.ParseUint(hex[i:i+2], 16, 8)
		if err != nil {
			return "EEF2FF"
		}
		c := float64(v) + (255-float64(v))*k
		fmt.Fprintf(&out, "%02X", int(math.Round(c)))
	}
	return out.String()
}

// anchoImaxeFormularioMM é o ancho dunha imaxe dun formulario: o seu tamaño a
// 150 ppp con tope (o mesmo criterio ca no PDF e no Markdown).
func anchoImaxeFormularioMM(im *imaxeOfi) (w, h float64) {
	w = math.Min(float64(im.W)*25.4/150, larguraImaxeMaxMMFormulario)
	return w, w * float64(im.H) / float64(im.W)
}

// ---------- Word ----------

// runsLinhasDocx escribe un texto con saltos de liña coma runs de Word.
func runsLinhasDocx(texto string, negrita bool, mediosPt int) []string {
	linhas := strings.Split(strings.ReplaceAll(texto, "\r\n", "\n"), "\n")
	var runs []string
	for i, l := range linhas {
		if i > 0 {
			runs = append(runs, `<w:r><w:br/></w:r>`)
		}
		runs = append(runs, runDocx(strings.TrimSpace(l), negrita, false, "", mediosPt))
	}
	return runs
}

// taboaDocx escribe unha táboa de Word: cabeceira repetida en cada páxina,
// filas que non se cortan e bordos coa cor de liña da plantilla.
func taboaDocx(cab []string, filas [][]string, e estiloOfi) string {
	n := len(cab)
	if n == 0 {
		return ""
	}
	bordo := func(nome string) string {
		return `<w:` + nome + ` w:val="single" w:sz="4" w:space="0" w:color="` + e.Linha + `"/>`
	}
	ancho := strconv.Itoa(docxColumnaEMU / 635 / n) // EMU -> twips
	var b strings.Builder
	b.WriteString(`<w:tbl><w:tblPr><w:tblW w:w="5000" w:type="pct"/><w:tblBorders>` +
		bordo("top") + bordo("left") + bordo("bottom") + bordo("right") + bordo("insideH") + bordo("insideV") +
		`</w:tblBorders><w:tblCellMar><w:top w:w="40" w:type="dxa"/><w:left w:w="100" w:type="dxa"/>` +
		`<w:bottom w:w="40" w:type="dxa"/><w:right w:w="100" w:type="dxa"/></w:tblCellMar></w:tblPr><w:tblGrid>`)
	for range cab {
		b.WriteString(`<w:gridCol w:w="` + ancho + `"/>`)
	}
	b.WriteString(`</w:tblGrid>`)
	fila := func(celas []string, cabeceira bool) {
		if cabeceira {
			b.WriteString(`<w:tr><w:trPr><w:cantSplit/><w:tblHeader/></w:trPr>`)
		} else {
			b.WriteString(`<w:tr><w:trPr><w:cantSplit/></w:trPr>`)
		}
		for i := 0; i < n; i++ {
			txt := ""
			if i < len(celas) {
				txt = celas[i]
			}
			b.WriteString(`<w:tc><w:tcPr><w:tcW w:w="` + ancho + `" w:type="dxa"/>`)
			if cabeceira {
				b.WriteString(`<w:shd w:val="clear" w:color="auto" w:fill="` + tintarHex(e.Principal, 0.88) + `"/>`)
			}
			b.WriteString(`</w:tcPr>`)
			b.WriteString(paraDocx("BodyText", `<w:spacing w:before="30" w:after="30"/><w:jc w:val="left"/>`, runsLinhasDocx(txt, cabeceira, 20)...))
			b.WriteString(`</w:tc>`)
		}
		b.WriteString(`</w:tr>`)
	}
	fila(cab, true)
	for _, f := range filas {
		fila(f, false)
	}
	b.WriteString(`</w:tbl>`)
	// Unha táboa non pode ser o último do corpo e, ademais, precisa un respiro.
	b.WriteString(paraDocx("BodyText", `<w:spacing w:before="0" w:after="60"/>`, runDocx("", false, false, "", 0)))
	return b.String()
}

// bloquesFormulario escribe os bloques do documento en Word.
func (c *construtorDocx) bloquesFormulario(d *datosOfimatica) {
	for _, bl := range d.Formulario.Bloques {
		switch bl.Tipo {
		case bloqueTitulo:
			estilo := "Heading2"
			if bl.Nivel >= 3 {
				estilo = "Subheading"
			}
			c.texto(estilo, bl.Texto)
		case bloqueFicha:
			for _, par := range bl.Pares {
				runs := append([]string{runDocx(par[0]+": ", true, false, "", 0)}, runsLinhasDocx(par[1], false, 0)...)
				c.corpo.WriteString(paraDocx("BodyText", `<w:keepLines/><w:spacing w:after="60"/><w:jc w:val="left"/>`, runs...))
			}
		case bloqueMarkdown:
			if md := parseMarkdownCrono(bl.Texto); len(md) > 0 {
				c.bloques(md, nil, &d.Avisos)
			}
		case bloqueSolucion:
			c.texto("Subheading", bl.Rotulo)
			c.bloques(parseMarkdownCrono(bl.Texto), nil, &d.Avisos)
		case bloqueTaboa:
			c.corpo.WriteString(taboaDocx(bl.Cabeceira, bl.Filas, d.Estilo))
		case bloqueLista:
			for _, el := range bl.Elementos {
				runs := append([]string{runDocx("•\t", false, false, "", 0)}, runsLinhasDocx(el, false, 0)...)
				c.corpo.WriteString(paraDocx("ListBullet", "", runs...))
			}
		case bloqueImaxe:
			im, err := imaxeOfiDeURI(bl.Imaxe)
			if err != nil || im.W <= 0 || im.H <= 0 {
				d.Avisos = engadirSenRepetir(d.Avisos, []string{"unha imaxe do formulario non se puido usar: omitiuse"})
				continue
			}
			if bl.Texto != "" {
				c.texto("Subheading", bl.Texto)
			}
			w, h := anchoImaxeFormularioMM(im)
			cx, cy := int(w*docxEmuPorMM), int(h*docxEmuPorMM)
			if rel, ok := c.engadirImaxe(im, cx, cy); ok {
				c.corpo.WriteString(paraImaxeDocx(rel, cx, cy, cronoAlinDefecto))
			}
		case bloqueGrafico:
			c.graficoFormulario(d, bl)
		case bloqueSalto:
			// Un parágrafo mínimo que abre páxina nova (sen deixar un oco visible).
			c.corpo.WriteString(paraDocx("BodyText", `<w:pageBreakBefore/><w:spacing w:before="0" w:after="0" w:line="20" w:lineRule="exact"/>`, runDocx("", false, false, "", 2)))
		}
	}
}

// escribirDocxFormulario compón o formulario en Word e devolve os bytes do .docx.
func escribirDocxFormulario(d *datosOfimatica) ([]byte, error) {
	c := &construtorDocx{baseMedios: int(d.Estilo.TamanoPt*2 + 0.5), formulas: d.Formulas}
	fm := d.Formulario
	// Sen título automático (vai na plantilla ou nun campo): tras a portada, só o salto.
	if fm.Portada {
		c.portada(d)
		c.corpo.WriteString(paraDocx("BodyText", `<w:pageBreakBefore/><w:spacing w:before="0" w:after="0" w:line="20" w:lineRule="exact"/>`, runDocx("", false, false, "", 2)))
	}
	c.bloquesFormulario(d)
	if fm.Sinatura {
		c.texto("BodyText", "")
		c.texto("SignName", d.Xefe)
		c.texto("SignMeta", d.Cargo)
		c.texto("SignMeta", d.Tx.T("LUGAR_DATA"))
	}
	return montarDocx(c.corpo.String(), c.imaxes, d)
}

// ---------- OpenDocument ----------

// linhasOdt escribe un texto con saltos de liña coma contido de parágrafo ODT.
func linhasOdt(texto string) string {
	linhas := strings.Split(strings.ReplaceAll(texto, "\r\n", "\n"), "\n")
	for i, l := range linhas {
		linhas[i] = escaparXMLOfi(strings.TrimSpace(l))
	}
	return strings.Join(linhas, `<text:line-break/>`)
}

// estilosFormularioOdt son os estilos automáticos (content.xml) que usan as
// táboas e as fichas dos formularios. `columnas` son os números de columnas
// das táboas do documento (un estilo de columna por cada un).
func estilosFormularioOdt(e estiloOfi, columnas map[int]bool) string {
	var b strings.Builder
	b.WriteString(`<style:style style:name="FormField" style:family="paragraph" style:parent-style-name="TextBody">` +
		`<style:paragraph-properties fo:margin-bottom="0.1cm" fo:text-align="start"/></style:style>`)
	b.WriteString(`<style:style style:name="TableText" style:family="paragraph" style:parent-style-name="TextBody">` +
		`<style:paragraph-properties fo:margin-top="0.05cm" fo:margin-bottom="0.05cm" fo:text-align="start"/>` +
		`<style:text-properties fo:font-size="10pt"/></style:style>`)
	b.WriteString(`<style:style style:name="LuaTaboa" style:family="table"><style:table-properties style:width="` +
		fmt.Sprintf("%.2fcm", odtColumnaCm) + `" style:rel-width="100%" table:align="margins" fo:margin-top="0.1cm" fo:margin-bottom="0.3cm"/></style:style>`)
	borde := `fo:border="0.5pt solid #` + e.Linha + `"`
	b.WriteString(`<style:style style:name="LuaCela" style:family="table-cell"><style:table-cell-properties fo:padding="0.1cm" ` + borde + `/></style:style>`)
	b.WriteString(`<style:style style:name="LuaCelaCab" style:family="table-cell"><style:table-cell-properties fo:padding="0.1cm" ` + borde +
		` fo:background-color="#` + tintarHex(e.Principal, 0.88) + `"/></style:style>`)
	for n := range columnas {
		b.WriteString(fmt.Sprintf(`<style:style style:name="LuaCol%d" style:family="table-column"><style:table-column-properties style:column-width="%.3fcm" style:rel-column-width="%d*"/></style:style>`,
			n, odtColumnaCm/float64(n), 65535/n))
	}
	return b.String()
}

// taboaOdt escribe unha táboa de OpenDocument.
func (c *construtorOdt) taboaOdt(cab []string, filas [][]string, columnas map[int]bool) {
	n := len(cab)
	if n == 0 {
		return
	}
	columnas[n] = true
	c.nTaboas++
	c.corpo.WriteString(fmt.Sprintf(`<table:table table:name="Taboa%d" table:style-name="LuaTaboa"><table:table-column table:style-name="LuaCol%d" table:number-columns-repeated="%d"/>`, c.nTaboas, n, n))
	fila := func(celas []string, cabeceira bool) {
		c.corpo.WriteString(`<table:table-row>`)
		for i := 0; i < n; i++ {
			txt := ""
			if i < len(celas) {
				txt = celas[i]
			}
			estilo, contido := "LuaCela", linhasOdt(txt)
			if cabeceira {
				estilo, contido = "LuaCelaCab", `<text:span text:style-name="Strong">`+linhasOdt(txt)+`</text:span>`
			}
			c.corpo.WriteString(`<table:table-cell table:style-name="` + estilo + `" office:value-type="string"><text:p text:style-name="TableText">` + contido + `</text:p></table:table-cell>`)
		}
		c.corpo.WriteString(`</table:table-row>`)
	}
	c.corpo.WriteString(`<table:table-header-rows>`)
	fila(cab, true)
	c.corpo.WriteString(`</table:table-header-rows>`)
	for _, f := range filas {
		fila(f, false)
	}
	c.corpo.WriteString(`</table:table>`)
}

// bloquesFormulario escribe os bloques do documento en OpenDocument.
func (c *construtorOdt) bloquesFormulario(d *datosOfimatica, columnas map[int]bool) {
	for _, bl := range d.Formulario.Bloques {
		switch bl.Tipo {
		case bloqueTitulo:
			estilo := "Heading2"
			if bl.Nivel >= 3 {
				estilo = "Subheading"
			}
			c.p(estilo, bl.Texto)
		case bloqueFicha:
			for _, par := range bl.Pares {
				c.corpo.WriteString(`<text:p text:style-name="FormField"><text:span text:style-name="Strong">` + escaparXMLOfi(par[0]+": ") + `</text:span>` + linhasOdt(par[1]) + `</text:p>`)
			}
		case bloqueMarkdown:
			if md := parseMarkdownCrono(bl.Texto); len(md) > 0 {
				c.bloques(md, nil, &d.Avisos)
			}
		case bloqueSolucion:
			c.p("Subheading", bl.Rotulo)
			c.bloques(parseMarkdownCrono(bl.Texto), nil, &d.Avisos)
		case bloqueTaboa:
			c.taboaOdt(bl.Cabeceira, bl.Filas, columnas)
		case bloqueLista:
			for _, el := range bl.Elementos {
				c.corpo.WriteString(`<text:p text:style-name="ListItem">•<text:tab/>` + linhasOdt(el) + `</text:p>`)
			}
		case bloqueImaxe:
			im, err := imaxeOfiDeURI(bl.Imaxe)
			if err != nil || im.W <= 0 || im.H <= 0 {
				d.Avisos = engadirSenRepetir(d.Avisos, []string{"unha imaxe do formulario non se puido usar: omitiuse"})
				continue
			}
			if bl.Texto != "" {
				c.p("Subheading", bl.Texto)
			}
			w, h := anchoImaxeFormularioMM(im)
			if o, ok := c.engadirImaxe(im, w/10, h/10); ok {
				c.pImaxe(o, "Figure")
			}
		case bloqueGrafico:
			c.graficoFormulario(d, bl)
		case bloqueSalto:
			c.corpo.WriteString(`<text:p text:style-name="PaxinaNova"/>`)
		}
	}
}

// escribirOdtFormulario compón o formulario en OpenDocument e devolve os bytes do .odt.
func escribirOdtFormulario(d *datosOfimatica) ([]byte, error) {
	c := &construtorOdt{imaxesPlantilla: d.ImaxesPlantilla, formulas: d.Formulas}
	fm := d.Formulario
	extraEstilos := ""
	if fm.Portada {
		extraEstilos = c.portada(d)
		c.corpo.WriteString(`<text:p text:style-name="PaxinaNova"/>`)
	}
	columnas := map[int]bool{}
	c.bloquesFormulario(d, columnas)
	if fm.Sinatura {
		c.p("TextBody", "")
		c.p("SignName", d.Xefe)
		c.p("SignMeta", d.Cargo)
		c.p("SignMeta", d.Tx.T("LUGAR_DATA"))
	}
	d.cabOdt = c.caixaOdt(d.Cabeceira, "Header")
	d.peOdt = c.caixaOdt(d.Pe, "Footer")
	estilos := extraEstilos + estilosFormularioOdt(d.Estilo, columnas) +
		`<style:style style:name="PaxinaNova" style:family="paragraph" style:parent-style-name="TextBody"><style:paragraph-properties fo:break-before="page" fo:margin-top="0cm" fo:margin-bottom="0cm"/><style:text-properties fo:font-size="1pt"/></style:style>`
	contido := odtCabeceiraContido + odtEstilosAutomaticosCon(d.Estilo, estilos) +
		`<office:body><office:text>` + c.corpo.String() + `</office:text></office:body></office:document-content>`
	return montarOdt(contido, c.imaxes, d)
}
