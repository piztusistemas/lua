package main

// Estética de Word (.docx) e OpenDocument (.odt) DENDE A PLANTILLA.
//
// Lua xera toda a información do documento (datos, textos, tarefas, datas);
// o aspecto -tipo de letra, cores, portada, cabeceira e pé, logotipos- é cousa
// da plantilla. Para o PDF iso xa vai no LaTeX da plantilla (maqueta, {{PORTADA}},
// {{SINATURA}}); para Word e OpenDocument, que non usan LaTeX, cada plantilla
// leva este bloque (PlantillaLatex.Ofimatica). Baleiro = o deseño por defecto.
//
// A PORTADA escríbese cunhas liñas de texto co seu significado (as variables
// {{CENTRO}}, {{CURSO}}... enchénse coma no resto da plantilla):
//
//	@logo 35            logotipo do curso (alto en mm, opcional)
//	@imaxe cabeceira.png 30   imaxe do cartafol de imaxes da plantilla (alto en mm)
//	@espazo 12          espazo vertical en mm
//	# CRONOGRAMA ANUAL  título grande
//	## {{DEPARTAMENTO}}  subtítulo
//	---                 liña separadora
//	### {{CENTRO}}      liña destacada (nome do centro)
//	Curso académico {{CURSO}}    liña normal centrada
//
// A CABECEIRA e o PÉ son unha soa liña con elementos en orde: texto (con
// variables) e imaxes. Cada liña da plantilla é un elemento:
//
//	@imaxe logo1.png 10   imaxe da plantilla (alto en mm; PDF tamén) - pódense poñer varias
//	@imaxe logo2.pdf 10
//	{{CENTRO}} · {{CURSO}}  texto
//	@dereita              aliñamento: @esquerda, @centro ou @dereita (cabeceira: dereita; pé: centro)
//
// Ao pé engádese despois o número de páxina.

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// OfimaticaPlantilla está aliasado en edicion_plantilla.go (tipo do módulo).

// portadaOfimaticaPorDefecto é a portada que se usa se a plantilla non ten a súa.
const portadaOfimaticaPorDefecto = `@logo 35
# {{TITULO}}
## {{DEPARTAMENTO}}
---
### {{CENTRO}}
{{CURSO_ACADEMICO}}
{{LOCALIDADE}}, {{DATA_LONGA}}`

const (
	ofiFontePorDefecto     = "Arial"
	ofiPrincipalPorDefecto = "1F3864"
	ofiSecundariaDefecto   = "2E5496"
	ofiTextoPorDefecto     = "212529"
	ofiApagadaPorDefecto   = "595959"
	ofiLinhaPorDefecto     = "8EAADB"
	ofiTamanoPorDefecto    = 11.0
)

// estiloOfi é o estilo xa resolto (con todos os valores por defecto postos).
type estiloOfi struct {
	Fonte, Principal, Secundaria, Texto, Apagada, Linha string
	TamanoPt                                            float64
	// Interlinado (0 = por defecto) e marxes en cm (0 = por defecto do formato).
	Interlinado                            float64
	MarxeSup, MarxeInf, MarxeEsq, MarxeDer float64
}

var corHexRe = regexp.MustCompile(`^[0-9A-Fa-f]{6}$`)

// normalizarCorHex acepta "#1f3864", "1F3864" ou "abc" e devolve "1F3864".
func normalizarCorHex(s string) (string, error) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if s == "" {
		return "", nil
	}
	if len(s) == 3 {
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	}
	if !corHexRe.MatchString(s) {
		return "", fmt.Errorf("cor non válida: %q (usa un valor hexadecimal coma 1F3864)", s)
	}
	return strings.ToUpper(s), nil
}

var fonteInvalidaRe = regexp.MustCompile(`[<>&"'\\/;{}]`)

// normalizarOfimatica valida e limpa o bloque (cores, fonte, tamaño).
func normalizarOfimatica(o OfimaticaPlantilla) (OfimaticaPlantilla, error) {
	o.Fonte = strings.TrimSpace(fonteInvalidaRe.ReplaceAllString(o.Fonte, ""))
	for _, c := range []*string{&o.CorPrincipal, &o.CorSecundaria, &o.CorTexto, &o.CorApagada, &o.CorLinha} {
		v, err := normalizarCorHex(*c)
		if err != nil {
			return o, err
		}
		*c = v
	}
	if o.TamanoCorpo != 0 && (o.TamanoCorpo < 8 || o.TamanoCorpo > 16) {
		return o, fmt.Errorf("o tamaño do corpo de Word/OpenDocument ten que estar entre 8 e 16 pt (ou 0 para o por defecto)")
	}
	if o.Interlinado != 0 && (o.Interlinado < 1 || o.Interlinado > 3) {
		return o, fmt.Errorf("o interliñado de Word/OpenDocument ten que estar entre 1 e 3 (ou 0 para o por defecto, 1.15)")
	}
	for nome, m := range map[string]float64{"superior": o.MarxeSuperior, "inferior": o.MarxeInferior, "esquerda": o.MarxeEsquerda, "dereita": o.MarxeDereita} {
		if m != 0 && (m < 0.5 || m > 6) {
			return o, fmt.Errorf("a marxe %s de Word/OpenDocument ten que estar entre 0.5 e 6 cm (ou 0 para a por defecto)", nome)
		}
	}
	o.Portada = strings.ReplaceAll(o.Portada, "\r\n", "\n")
	return o, nil
}

// estiloOfimatica resolve o estilo de Word/OpenDocument cun OfimaticaPlantilla
// (tipo do módulo común), enchendo os valores por defecto. É unha función e non
// un método porque o tipo vive en edicion/plantilla.
func estiloOfimatica(o OfimaticaPlantilla) estiloOfi {
	pon := func(v, def string) string {
		if v == "" {
			return def
		}
		return v
	}
	e := estiloOfi{
		Fonte: pon(o.Fonte, ofiFontePorDefecto), Principal: pon(o.CorPrincipal, ofiPrincipalPorDefecto),
		Secundaria: pon(o.CorSecundaria, ofiSecundariaDefecto), Texto: pon(o.CorTexto, ofiTextoPorDefecto),
		Apagada: pon(o.CorApagada, ofiApagadaPorDefecto), Linha: pon(o.CorLinha, ofiLinhaPorDefecto),
		TamanoPt: o.TamanoCorpo, Interlinado: o.Interlinado,
		MarxeSup: o.MarxeSuperior, MarxeInf: o.MarxeInferior, MarxeEsq: o.MarxeEsquerda, MarxeDer: o.MarxeDereita,
	}
	if e.TamanoPt == 0 {
		e.TamanoPt = ofiTamanoPorDefecto
	}
	return e
}

// aplicarEstiloXML cambia no XML de estilos por defecto os valores fixos
// (fonte, cores, tamaño do corpo) polos da plantilla. As cores por defecto
// son cadeas hexadecimais únicas, así que abonda con substituílas.
func (e estiloOfi) aplicarEstiloXML(xml string) string {
	r := strings.NewReplacer(
		ofiPrincipalPorDefecto, e.Principal,
		ofiSecundariaDefecto, e.Secundaria,
		ofiTextoPorDefecto, e.Texto,
		ofiApagadaPorDefecto, e.Apagada,
		ofiLinhaPorDefecto, e.Linha,
		ofiFontePorDefecto, e.Fonte,
	)
	return r.Replace(xml)
}

// Valores por defecto que se cambian por cadea nos XML base (docx/odt).
const (
	docxMarxesPorDefecto   = `w:top="1418" w:right="1418" w:bottom="1418" w:left="1418"`
	docxInterlinadoDefecto = `<w:spacing w:after="160" w:line="276" w:lineRule="auto"/></w:pPr></w:pPrDefault>`
	odtMarxesPorDefecto    = `fo:margin-top="2cm" fo:margin-bottom="2cm" fo:margin-left="2cm" fo:margin-right="2cm"/>`
	odtInterlinadoDefecto  = `fo:margin-bottom="0.25cm" fo:line-height="115%" fo:text-align="start"/>`
)

// cmATwips: 1 cm = 567 twips (as unidades de Word).
func cmATwips(cm float64) int { return int(cm*567 + 0.5) }

// docxPagina devolve o sectPr co tamaño de marxes da plantilla (as que valgan 0
// quedan coma están: 2,5 cm).
func (e estiloOfi) docxPagina(sect string) string {
	twips := func(cm float64) int {
		if cm == 0 {
			return 1418
		}
		return cmATwips(cm)
	}
	novo := fmt.Sprintf(`w:top="%d" w:right="%d" w:bottom="%d" w:left="%d"`,
		twips(e.MarxeSup), twips(e.MarxeDer), twips(e.MarxeInf), twips(e.MarxeEsq))
	return strings.Replace(sect, docxMarxesPorDefecto, novo, 1)
}

// docxInterlinado aplica o interliñado ao estilo por defecto dos parágrafos.
func (e estiloOfi) docxInterlinado(xml string) string {
	if e.Interlinado == 0 {
		return xml
	}
	novo := fmt.Sprintf(`<w:spacing w:after="160" w:line="%d" w:lineRule="auto"/></w:pPr></w:pPrDefault>`, int(e.Interlinado*240+0.5))
	return strings.Replace(xml, docxInterlinadoDefecto, novo, 1)
}

// odtPaxina aplica as marxes e o interliñado da plantilla ao styles.xml do ODT.
func (e estiloOfi) odtPaxina(xml string) string {
	cm := func(v, def float64) float64 {
		if v == 0 {
			return def
		}
		return v
	}
	novo := fmt.Sprintf(`fo:margin-top="%.2fcm" fo:margin-bottom="%.2fcm" fo:margin-left="%.2fcm" fo:margin-right="%.2fcm"/>`,
		cm(e.MarxeSup, 2), cm(e.MarxeInf, 2), cm(e.MarxeEsq, 2), cm(e.MarxeDer, 2))
	xml = strings.Replace(xml, odtMarxesPorDefecto, novo, 1)
	if e.Interlinado != 0 {
		xml = strings.Replace(xml, odtInterlinadoDefecto,
			fmt.Sprintf(`fo:margin-bottom="0.25cm" fo:line-height="%d%%" fo:text-align="start"/>`, int(e.Interlinado*100+0.5)), 1)
	}
	return xml
}

// ---------- Portada ----------

type linhaPortada struct {
	Tipo   string  // logo, imaxe, espazo, titulo, subtitulo, organizacion, regra, meta
	Texto  string  // xa con variables enchidas
	Nome   string  // imaxe
	AltoMM float64 // logo, imaxe, espazo (0 = por defecto)
}

// parsearPortadaOfi le a portada (ver o comentario do ficheiro). Unha liña
// baleira non fai nada; `subst` enche as variables de cada texto.
func parsearPortadaOfi(dsl string, subst func(string) string) []linhaPortada {
	var out []linhaPortada
	for _, bruta := range strings.Split(strings.ReplaceAll(dsl, "\r\n", "\n"), "\n") {
		l := strings.TrimSpace(bruta)
		switch {
		case l == "":
			continue
		case strings.HasPrefix(l, "@"):
			campos := strings.Fields(l[1:])
			if len(campos) == 0 {
				continue
			}
			alto := func(i int) float64 {
				if len(campos) > i {
					if v, err := strconv.ParseFloat(strings.TrimSuffix(campos[i], "mm"), 64); err == nil && v > 0 && v < 250 {
						return v
					}
				}
				return 0
			}
			switch strings.ToLower(campos[0]) {
			case "logo":
				out = append(out, linhaPortada{Tipo: "logo", AltoMM: alto(1)})
			case "imaxe", "imagen", "image":
				if len(campos) > 1 {
					out = append(out, linhaPortada{Tipo: "imaxe", Nome: campos[1], AltoMM: alto(2)})
				}
			case "espazo", "espacio", "space":
				out = append(out, linhaPortada{Tipo: "espazo", AltoMM: alto(1)})
			}
		case l == "---" || l == "***":
			out = append(out, linhaPortada{Tipo: "regra"})
		case strings.HasPrefix(l, "### "):
			out = append(out, linhaPortada{Tipo: "organizacion", Texto: subst(l[4:])})
		case strings.HasPrefix(l, "## "):
			out = append(out, linhaPortada{Tipo: "subtitulo", Texto: subst(l[3:])})
		case strings.HasPrefix(l, "# "):
			out = append(out, linhaPortada{Tipo: "titulo", Texto: subst(l[2:])})
		default:
			out = append(out, linhaPortada{Tipo: "meta", Texto: subst(l)})
		}
	}
	return out
}

// elementoCaixa é unha peza da cabeceira ou do pé: texto ou imaxe.
type elementoCaixa struct {
	Tipo   string // texto, imaxe
	Texto  string
	Nome   string  // imaxe
	AltoMM float64 // imaxe (0 = 10 mm)
}

// caixaOfi é a cabeceira ou o pé xa lidos: elementos e aliñamento.
type caixaOfi struct {
	Elementos []elementoCaixa
	Alin      string // esquerda, centro, dereita
}

func (c caixaOfi) baleira() bool { return len(c.Elementos) == 0 }

// textoPlano xunta os textos (sen imaxes), separados por espazos.
func (c caixaOfi) textoPlano() string {
	var t []string
	for _, e := range c.Elementos {
		if e.Tipo == "texto" {
			t = append(t, e.Texto)
		}
	}
	return strings.Join(t, "  ")
}

// parsearCaixaOfi le a cabeceira ou o pé (ver o comentario do ficheiro).
func parsearCaixaOfi(dsl string, subst func(string) string, alinPorDefecto string) caixaOfi {
	c := caixaOfi{Alin: alinPorDefecto}
	for _, bruta := range strings.Split(strings.ReplaceAll(dsl, "\r\n", "\n"), "\n") {
		l := strings.TrimSpace(bruta)
		if l == "" {
			continue
		}
		if !strings.HasPrefix(l, "@") {
			if t := subst(l); t != "" {
				c.Elementos = append(c.Elementos, elementoCaixa{Tipo: "texto", Texto: t})
			}
			continue
		}
		campos := strings.Fields(l[1:])
		switch strings.ToLower(campos[0]) {
		case "esquerda", "izquierda", "left":
			c.Alin = "esquerda"
		case "centro", "center":
			c.Alin = "centro"
		case "dereita", "derecha", "right":
			c.Alin = "dereita"
		case "imaxe", "imagen", "image", "logo":
			if len(campos) < 2 {
				continue
			}
			e := elementoCaixa{Tipo: "imaxe", Nome: campos[1]}
			if len(campos) > 2 {
				if v, err := strconv.ParseFloat(strings.TrimSuffix(campos[2], "mm"), 64); err == nil && v > 0 && v < 60 {
					e.AltoMM = v
				}
			}
			c.Elementos = append(c.Elementos, e)
		}
	}
	return c
}

// nomesImaxesOfi devolve as imaxes da plantilla que usa a portada, a cabeceira ou o pé.
func nomesImaxesOfi(portada []linhaPortada, caixas ...caixaOfi) []string {
	vistos := map[string]bool{}
	var out []string
	add := func(n string) {
		if n != "" && !vistos[n] {
			vistos[n] = true
			out = append(out, n)
		}
	}
	for _, l := range portada {
		if l.Tipo == "imaxe" {
			add(l.Nome)
		}
	}
	for _, c := range caixas {
		for _, e := range c.Elementos {
			if e.Tipo == "imaxe" {
				add(e.Nome)
			}
		}
	}
	return out
}

// PrevisualizarPlantillaOfimatica renderiza o deseño de Word ou OpenDocument
// (formato "docx" ou "odt") da plantilla, sen gardala, cun curso de exemplo:
// devolve as primeiras páxinas (PNG en data URI) para a vista previa do
// editor. Precisa LibreOffice (converte o documento a PDF) e pdftoppm.
func (a *App) PrevisualizarPlantillaOfimatica(p PlantillaLatex, formato string) (GeneratePDFResult, error) {
	return a.PrevisualizarPlantillaOfimaticaDe(p, formato, "")
}

// PrevisualizarPlantillaOfimaticaDe é o anterior co documento de mostra da
// ferramenta indicada ("cronograma" ou "actas").
func (a *App) PrevisualizarPlantillaOfimaticaDe(p PlantillaLatex, formato, ferramenta string) (GeneratePDFResult, error) {
	return a.previsualizarOfimaticaPedido(p, formato, pedidoDemo(ferramenta))
}

// PrevisualizarPlantillaOfimaticaDocumento é o anterior co documento que o
// usuario ten en edición.
func (a *App) PrevisualizarPlantillaOfimaticaDocumento(p PlantillaLatex, formato string, req XerarInformeCronogramaRequest) (GeneratePDFResult, error) {
	return a.previsualizarOfimaticaPedido(p, formato, req)
}

func (a *App) previsualizarOfimaticaPedido(p PlantillaLatex, formato string, pedido XerarInformeCronogramaRequest) (GeneratePDFResult, error) {
	formato = strings.ToLower(strings.TrimSpace(formato))
	if formato != "odt" {
		formato = "docx"
	}
	ofi, err := normalizarOfimatica(p.Ofimatica)
	if err != nil {
		return GeneratePDFResult{}, err
	}
	p.Ofimatica = ofi
	p.Variables = normalizarVariables(p.Variables)
	p.Textos = normalizarTextosDoc(p.Textos)
	tipo := tipoAdxDocx
	if formato == "odt" {
		tipo = tipoAdxOdt
	}
	datos, avisosOfi, err := datosOfimaticaDe(pedido, &p, formato)
	if err != nil {
		return GeneratePDFResult{}, err
	}
	if !commandExists("pdftoppm") {
		return GeneratePDFResult{}, fmt.Errorf("non se atopou pdftoppm (paquete poppler-utils) para debuxar a vista previa")
	}
	pdf, limpar, err := converterAPDFLibreOffice(Adxunto{ID: "previa", Nome: "previa." + formato, Tipo: tipo}, datos)
	if err != nil {
		return GeneratePDFResult{}, err
	}
	defer limpar()
	dir, err := os.MkdirTemp("", "lua-previa-ofi-*")
	if err != nil {
		return GeneratePDFResult{}, err
	}
	defer os.RemoveAll(dir)
	if _, err := executarComTempo(90, "pdftoppm", "-f", "1", "-l", "6", "-r", "80", "-png", pdf, filepath.Join(dir, "pax")); err != nil {
		return GeneratePDFResult{}, err
	}
	nomes, _ := filepath.Glob(filepath.Join(dir, "pax*.png"))
	sort.Strings(nomes) // pdftoppm enche o número con ceros: a orde alfabética é a das páxinas
	res := GeneratePDFResult{Warnings: avisosOfi}
	for _, n := range nomes {
		b, err := os.ReadFile(n)
		if err != nil {
			return GeneratePDFResult{}, err
		}
		res.PageImages = append(res.PageImages, "data:image/png;base64,"+base64.StdEncoding.EncodeToString(b))
	}
	return res, nil
}
