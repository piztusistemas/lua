package main

// Extracción de texto e de imaxes dos adxuntos: PDF (pdftotext/pdfimages/
// pdftoppm, de poppler-utils, que xa é dependencia de Lua), Word e
// OpenDocument (o texto sácase directamente do XML do ZIP, sen ferramentas
// externas), presentacións, follas de cálculo, RTF e texto. Os formatos
// antigos (.doc, .xls, .ppt) pásanse antes por LibreOffice se está
// instalado.

import (
	"archive/zip"
	"bytes"
	"context"
	"edicion/idioma"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const maxTextoAdxunto = 4_000_000 // caracteres que se gardan en memoria por adxunto

// tipoDentroDoZip mira o contido dun ZIP sen extensión coñecida.
func tipoDentroDoZip(d []byte) string {
	zr, err := zip.NewReader(bytes.NewReader(d), int64(len(d)))
	if err != nil {
		return ""
	}
	for _, f := range zr.File {
		switch {
		case f.Name == "word/document.xml":
			return tipoAdxDocx
		case strings.HasPrefix(f.Name, "ppt/slides/"):
			return tipoAdxPres
		case strings.HasPrefix(f.Name, "xl/worksheets/"):
			return tipoAdxFolla
		case f.Name == "mimetype":
			rc, err := f.Open()
			if err != nil {
				continue
			}
			b, _ := io.ReadAll(io.LimitReader(rc, 200))
			rc.Close()
			m := string(b)
			switch {
			case strings.Contains(m, "opendocument.text"):
				return tipoAdxOdt
			case strings.Contains(m, "opendocument.presentation"):
				return tipoAdxPres
			case strings.Contains(m, "opendocument.spreadsheet"):
				return tipoAdxFolla
			}
		}
	}
	return ""
}

// executarComTempo lanza unha ferramenta externa co seu límite de tempo.
func executarComTempo(seg int, nome string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(seg)*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, nome, args...)
	ocultarConsola(cmd)
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	out, err := cmd.Output()
	if err != nil {
		return out, fmt.Errorf("%s: %w %s", nome, err, strings.TrimSpace(errBuf.String()))
	}
	return out, nil
}

// escribirTemp garda datos nun cartafol temporal e devolve a ruta e a función de limpeza.
func escribirTemp(datos []byte, nome string) (string, func(), error) {
	dir, err := os.MkdirTemp("", "lua-adx-*")
	if err != nil {
		return "", nil, err
	}
	ruta := filepath.Join(dir, nome)
	if err := os.WriteFile(ruta, datos, 0o600); err != nil {
		os.RemoveAll(dir)
		return "", nil, err
	}
	return ruta, func() { os.RemoveAll(dir) }, nil
}

var paxinasRe = regexp.MustCompile(`(?m)^Pages:\s+(\d+)`)

// paxinasDoPDFBytes conta as páxinas (0 se non hai pdfinfo).
func paxinasDoPDFBytes(d []byte) int {
	if !commandExists("pdfinfo") {
		return 0
	}
	ruta, limpar, err := escribirTemp(d, "doc.pdf")
	if err != nil {
		return 0
	}
	defer limpar()
	out, err := executarComTempo(30, "pdfinfo", ruta)
	if err != nil {
		return 0
	}
	if m := paxinasRe.FindSubmatch(out); m != nil {
		n, _ := strconv.Atoi(string(m[1]))
		return n
	}
	return 0
}

// textoDoAdxunto devolve o texto que leva un adxunto (co que xa se calculou
// gardado en memoria). As imaxes non teñen texto: hai que velas (adxunto_ver).
func textoDoAdxunto(ax Adxunto, datos []byte) (string, error) {
	adxuntos.mu.Lock()
	t, feito := adxuntos.textos[ax.ID]
	adxuntos.mu.Unlock()
	if feito {
		return t, nil
	}
	var texto string
	var err error
	switch ax.Tipo {
	case tipoAdxImaxe:
		return "", fmt.Errorf("«%s» é unha imaxe: non ten texto que ler. Vela con adxunto_ver", ax.Nome)
	case tipoAdxPDF:
		texto, err = textoDePDFAdxunto(datos)
	case tipoAdxDocx:
		texto, err = textoDeZipOOXML(datos, "word/document.xml")
	case tipoAdxPres:
		texto, err = textoDePresentacion(datos)
	case tipoAdxFolla:
		texto, err = textoDeFolla(datos)
	case tipoAdxOdt:
		texto, err = textoDeODF(datos)
	case tipoAdxRtf:
		texto = textoDeRTF(string(datos))
	case tipoAdxTexto:
		texto = textoPlanoUTF8(datos)
		if strings.HasSuffix(strings.ToLower(ax.Nome), ".html") || strings.HasSuffix(strings.ToLower(ax.Nome), ".htm") {
			texto = textoDeHTML(datos)
		}
		if n := strings.ToLower(ax.Nome); strings.HasSuffix(n, ".md") || strings.HasSuffix(n, ".markdown") {
			texto = estruturaMarkdown(texto) + texto
		}
	case tipoAdxAntigo:
		texto, err = textoViaLibreOffice(ax, datos)
	default:
		return "", fmt.Errorf("non se sabe ler o formato de «%s»", ax.Nome)
	}
	if err != nil {
		return "", err
	}
	texto = strings.TrimSpace(texto)
	if len([]rune(texto)) > maxTextoAdxunto {
		texto = string([]rune(texto)[:maxTextoAdxunto])
	}
	adxuntos.mu.Lock()
	adxuntos.textos[ax.ID] = texto
	adxuntos.mu.Unlock()
	return texto, nil
}

func textoPlanoUTF8(d []byte) string {
	if utf8.Valid(d) {
		return strings.TrimPrefix(string(d), "\ufeff")
	}
	// Latin-1 (moito .txt e .csv de Windows).
	r := make([]rune, len(d))
	for i, b := range d {
		r[i] = rune(b)
	}
	return string(r)
}

func textoDePDFAdxunto(d []byte) (string, error) {
	if !commandExists("pdftotext") {
		return "", idioma.Errorf("go.ler.sen_pdftotext", "non se atopou pdftotext (paquete poppler-utils) para ler o PDF")
	}
	ruta, limpar, err := escribirTemp(d, "doc.pdf")
	if err != nil {
		return "", err
	}
	defer limpar()
	out, err := executarComTempo(90, "pdftotext", "-layout", ruta, "-")
	if err != nil {
		return "", err
	}
	t := strings.TrimSpace(strings.ReplaceAll(string(out), "\f", "\n\n"))
	if t == "" {
		return "", fmt.Errorf("o PDF non ten texto seleccionable (é unha imaxe escaneada?): mira as páxinas con adxunto_ver")
	}
	return t, nil
}

// ---------- Zip / XML ----------

func abrirZip(d []byte) (*zip.Reader, error) {
	zr, err := zip.NewReader(bytes.NewReader(d), int64(len(d)))
	if err != nil {
		return nil, idioma.Errorf("go.ler.ficheiro_danado", "o ficheiro está danado ou non é un documento válido: %w", err)
	}
	return zr, nil
}

func lerEntradaZip(zr *zip.Reader, nome string) ([]byte, error) {
	for _, f := range zr.File {
		if f.Name == nome {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(io.LimitReader(rc, 200<<20))
		}
	}
	return nil, fmt.Errorf("falta %s dentro do documento", nome)
}

func nomeLocal(n xml.Name) string { return n.Local }

// saidaTexto é un buffer ao que se lle pode quitar o último carácter: as
// celas e filas das táboas rematan o seu parágrafo cun salto que sobra.
type saidaTexto struct{ bytes.Buffer }

func (s *saidaTexto) quitarFinal(c byte) {
	b := s.Bytes()
	if n := len(b); n > 0 && b[n-1] == c {
		s.Truncate(n - 1)
	}
}

// cerrarCela e cerrarFila poñen os separadores de táboa (tabulador entre
// celas, salto entre filas) sen deixar liñas soltas dentro das celas.
func (s *saidaTexto) cerrarCela() { s.quitarFinal('\n'); s.WriteByte('\t') }
func (s *saidaTexto) cerrarFila() { s.quitarFinal('\t'); s.WriteByte('\n') }

// textoDeZipOOXML extrae o texto dun .docx (parágrafos, táboas, tabuladores).
func textoDeZipOOXML(d []byte, parte string) (string, error) {
	zr, err := abrirZip(d)
	if err != nil {
		return "", err
	}
	principal, err := lerEntradaZip(zr, parte)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString(textoWordXML(principal))
	// Cabeceiras, pés e notas: moitas plantillas de centro levan aí os datos.
	var extras []string
	for _, f := range zr.File {
		if regexp.MustCompile(`^word/(header|footer|footnotes|endnotes)\d*\.xml$`).MatchString(f.Name) {
			extras = append(extras, f.Name)
		}
	}
	sort.Strings(extras)
	for _, n := range extras {
		bx, err := lerEntradaZip(zr, n)
		if err != nil {
			continue
		}
		if t := strings.TrimSpace(textoWordXML(bx)); t != "" {
			b.WriteString("\n\n[" + strings.TrimSuffix(path.Base(n), ".xml") + "]\n" + t)
		}
	}
	return b.String(), nil
}

func textoWordXML(x []byte) string {
	d := xml.NewDecoder(bytes.NewReader(x))
	var b saidaTexto
	enTexto, enTabs := false, false
	for {
		tok, err := d.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch nomeLocal(t.Name) {
			case "tabs":
				enTabs = true // as paradas de tabulación dos estilos non son texto
			case "t":
				enTexto = true
			case "tab":
				if !enTabs {
					b.WriteByte('\t')
				}
			case "br", "cr":
				b.WriteByte('\n')
			}
		case xml.EndElement:
			switch nomeLocal(t.Name) {
			case "tabs":
				enTabs = false
			case "t":
				enTexto = false
			case "p":
				b.WriteByte('\n')
			case "tc":
				b.cerrarCela()
			case "tr":
				b.cerrarFila()
			}
		case xml.CharData:
			if enTexto {
				b.Write(t)
			}
		}
	}
	return normalizarSaltos(b.String())
}

func normalizarSaltos(s string) string {
	linhas := strings.Split(s, "\n")
	for i, l := range linhas {
		linhas[i] = strings.TrimRight(l, " \t")
	}
	s = strings.Join(linhas, "\n")
	return strings.TrimSpace(regexp.MustCompile(`\n{3,}`).ReplaceAllString(s, "\n\n"))
}

// textoDeODF extrae o texto dun documento OpenDocument (odt, ods, odp).
func textoDeODF(d []byte) (string, error) {
	zr, err := abrirZip(d)
	if err != nil {
		return "", err
	}
	x, err := lerEntradaZip(zr, "content.xml")
	if err != nil {
		return "", err
	}
	return textoODFXML(x), nil
}

func textoODFXML(x []byte) string {
	d := xml.NewDecoder(bytes.NewReader(x))
	var b saidaTexto
	saltar := 0
	omitidos := map[string]bool{"automatic-styles": true, "font-face-decls": true, "scripts": true, "styles": true}
	for {
		tok, err := d.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			n := nomeLocal(t.Name)
			if omitidos[n] {
				saltar++
				continue
			}
			if saltar > 0 {
				continue
			}
			switch n {
			case "tab":
				b.WriteByte('\t')
			case "line-break":
				b.WriteByte('\n')
			case "s":
				c := 1
				for _, a := range t.Attr {
					if a.Name.Local == "c" {
						c, _ = strconv.Atoi(a.Value)
					}
				}
				b.WriteString(strings.Repeat(" ", max(1, min(c, 50))))
			case "page":
				b.WriteString("\n--- Diapositiva ---\n")
			}
		case xml.EndElement:
			n := nomeLocal(t.Name)
			if omitidos[n] {
				saltar--
				continue
			}
			if saltar > 0 {
				continue
			}
			switch n {
			case "p", "h":
				b.WriteByte('\n')
			case "table-cell":
				b.cerrarCela()
			case "table-row":
				b.cerrarFila()
			}
		case xml.CharData:
			if saltar == 0 {
				b.Write(t)
			}
		}
	}
	return normalizarSaltos(b.String())
}

func textoDePresentacion(d []byte) (string, error) {
	zr, err := abrirZip(d)
	if err != nil {
		return "", err
	}
	if _, err := lerEntradaZip(zr, "content.xml"); err == nil { // OpenDocument (.odp)
		return textoDeODF(d)
	}
	type dia struct {
		n   int
		nom string
	}
	var dias []dia
	re := regexp.MustCompile(`^ppt/slides/slide(\d+)\.xml$`)
	for _, f := range zr.File {
		if m := re.FindStringSubmatch(f.Name); m != nil {
			n, _ := strconv.Atoi(m[1])
			dias = append(dias, dia{n, f.Name})
		}
	}
	sort.Slice(dias, func(i, j int) bool { return dias[i].n < dias[j].n })
	if len(dias) == 0 {
		return "", fmt.Errorf("a presentación non ten diapositivas lexibles")
	}
	var b strings.Builder
	for _, s := range dias {
		x, err := lerEntradaZip(zr, s.nom)
		if err != nil {
			continue
		}
		fmt.Fprintf(&b, "--- Diapositiva %d ---\n%s\n\n", s.n, textoDrawingML(x))
		if notas, err := lerEntradaZip(zr, fmt.Sprintf("ppt/notesSlides/notesSlide%d.xml", s.n)); err == nil {
			if t := strings.TrimSpace(textoDrawingML(notas)); t != "" {
				fmt.Fprintf(&b, "[Notas]\n%s\n\n", t)
			}
		}
	}
	return b.String(), nil
}

func textoDrawingML(x []byte) string {
	d := xml.NewDecoder(bytes.NewReader(x))
	var b strings.Builder
	enTexto := false
	for {
		tok, err := d.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if nomeLocal(t.Name) == "t" {
				enTexto = true
			}
		case xml.EndElement:
			switch nomeLocal(t.Name) {
			case "t":
				enTexto = false
			case "p":
				b.WriteByte('\n')
			}
		case xml.CharData:
			if enTexto {
				b.Write(t)
			}
		}
	}
	return normalizarSaltos(b.String())
}

// textoDeFolla le un .xlsx (cadeas compartidas + valores) ou un .ods.
func textoDeFolla(d []byte) (string, error) {
	zr, err := abrirZip(d)
	if err != nil {
		return "", err
	}
	if _, err := lerEntradaZip(zr, "content.xml"); err == nil { // .ods
		return textoDeODF(d)
	}
	var compartidas []string
	if x, err := lerEntradaZip(zr, "xl/sharedStrings.xml"); err == nil {
		compartidas = cadeasCompartidasXLSX(x)
	}
	type folla struct {
		n   int
		nom string
	}
	var follas []folla
	re := regexp.MustCompile(`^xl/worksheets/sheet(\d+)\.xml$`)
	for _, f := range zr.File {
		if m := re.FindStringSubmatch(f.Name); m != nil {
			n, _ := strconv.Atoi(m[1])
			follas = append(follas, folla{n, f.Name})
		}
	}
	sort.Slice(follas, func(i, j int) bool { return follas[i].n < follas[j].n })
	if len(follas) == 0 {
		return "", fmt.Errorf("a folla de cálculo non ten follas lexibles")
	}
	var b strings.Builder
	for _, f := range follas {
		x, err := lerEntradaZip(zr, f.nom)
		if err != nil {
			continue
		}
		fmt.Fprintf(&b, "--- Folla %d ---\n%s\n\n", f.n, textoFollaXLSX(x, compartidas))
	}
	return b.String(), nil
}

func cadeasCompartidasXLSX(x []byte) []string {
	d := xml.NewDecoder(bytes.NewReader(x))
	var out []string
	var cur strings.Builder
	enSi, enT := false, false
	for {
		tok, err := d.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch nomeLocal(t.Name) {
			case "si":
				enSi = true
				cur.Reset()
			case "t":
				enT = enSi
			}
		case xml.EndElement:
			switch nomeLocal(t.Name) {
			case "t":
				enT = false
			case "si":
				out = append(out, cur.String())
				enSi = false
			}
		case xml.CharData:
			if enT {
				cur.Write(t)
			}
		}
	}
	return out
}

func textoFollaXLSX(x []byte, compartidas []string) string {
	d := xml.NewDecoder(bytes.NewReader(x))
	var b strings.Builder
	tipoCelda := ""
	enV, enT := false, false
	var v strings.Builder
	filas := 0
	for {
		tok, err := d.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch nomeLocal(t.Name) {
			case "c":
				tipoCelda = ""
				for _, a := range t.Attr {
					if a.Name.Local == "t" {
						tipoCelda = a.Value
					}
				}
				v.Reset()
			case "v":
				enV = true
			case "t":
				enT = true
			}
		case xml.EndElement:
			switch nomeLocal(t.Name) {
			case "v":
				enV = false
			case "t":
				enT = false
			case "c":
				val := v.String()
				if tipoCelda == "s" {
					if i, err := strconv.Atoi(strings.TrimSpace(val)); err == nil && i >= 0 && i < len(compartidas) {
						val = compartidas[i]
					}
				}
				b.WriteString(val)
				b.WriteByte('\t')
			case "row":
				b.WriteByte('\n')
				filas++
				if filas > 2000 {
					b.WriteString("[... máis filas omitidas ...]\n")
					return normalizarSaltos(b.String())
				}
			}
		case xml.CharData:
			if enV || enT {
				v.Write(t)
			}
		}
	}
	return normalizarSaltos(strings.ReplaceAll(b.String(), "\t\n", "\n"))
}

var (
	rtfCtrlRe = regexp.MustCompile(`\\'([0-9a-fA-F]{2})`)
	rtfPalRe  = regexp.MustCompile(`\\[a-zA-Z]+-?\d* ?`)
)

// textoDeRTF quita o marcado dun RTF (basta para ler o texto).
func textoDeRTF(s string) string {
	s = strings.ReplaceAll(s, "\\par", "\n")
	s = rtfCtrlRe.ReplaceAllStringFunc(s, func(m string) string {
		n, _ := strconv.ParseUint(m[2:], 16, 8)
		return string(rune(n))
	})
	s = rtfPalRe.ReplaceAllString(s, "")
	s = strings.NewReplacer("{", "", "}", "", "\\\\", "\\").Replace(s)
	return normalizarSaltos(s)
}

// textoViaLibreOffice le formatos antigos (.doc, .xls, .ppt...) convertíndoos a PDF.
func textoViaLibreOffice(ax Adxunto, datos []byte) (string, error) {
	pdf, limpar, err := converterAPDFLibreOffice(ax, datos)
	if err != nil {
		return "", err
	}
	defer limpar()
	b, err := os.ReadFile(pdf)
	if err != nil {
		return "", err
	}
	return textoDePDFAdxunto(b)
}

func comandoLibreOffice() string {
	for _, c := range []string{"soffice", "libreoffice"} {
		if commandExists(c) {
			return c
		}
	}
	return ""
}

// converterAPDFLibreOffice converte un documento de oficina a PDF nun cartafol
// temporal. Devolve a ruta do PDF e a función que borra o cartafol.
func converterAPDFLibreOffice(ax Adxunto, datos []byte) (string, func(), error) {
	cmd := comandoLibreOffice()
	if cmd == "" {
		return "", nil, idioma.Errorf("go.ler.sen_libreoffice", "para ler «%s» fai falta LibreOffice, que non está instalado: convérteo a PDF, DOCX ou ODT", ax.Nome)
	}
	dir, err := os.MkdirTemp("", "lua-lo-*")
	if err != nil {
		return "", nil, err
	}
	limpar := func() { os.RemoveAll(dir) }
	nome := "doc" + strings.ToLower(filepath.Ext(ax.Nome))
	if filepath.Ext(nome) == "" {
		nome = "doc.bin"
	}
	ruta := filepath.Join(dir, nome)
	if err := os.WriteFile(ruta, datos, 0o600); err != nil {
		limpar()
		return "", nil, err
	}
	perfil := "-env:UserInstallation=file://" + filepath.ToSlash(filepath.Join(dir, "perfil"))
	if _, err := executarComTempo(120, cmd, perfil, "--headless", "--convert-to", "pdf", "--outdir", dir, ruta); err != nil {
		limpar()
		return "", nil, idioma.Errorf("go.ler.libreoffice_fallou", "LibreOffice non puido converter «%s»: %w", ax.Nome, err)
	}
	pdf := filepath.Join(dir, strings.TrimSuffix(nome, filepath.Ext(nome))+".pdf")
	if _, err := os.Stat(pdf); err != nil {
		limpar()
		return "", nil, fmt.Errorf("LibreOffice non xerou o PDF de «%s»", ax.Nome)
	}
	return pdf, limpar, nil
}

// ---------- Imaxes dentro dos documentos ----------

type imaxeExtraida struct {
	Nome  string
	Datos []byte
}

var extImaxeRe = regexp.MustCompile(`(?i)\.(png|jpe?g|gif|bmp|tiff?|webp)$`)

// imaxesDoAdxunto saca as imaxes incrustadas dun documento (o logotipo dunha
// cabeceira de Word, as fotos dun PDF...). Como moito `max`.
func imaxesDoAdxunto(ax Adxunto, datos []byte, max int) ([]imaxeExtraida, error) {
	switch ax.Tipo {
	case tipoAdxPDF:
		return imaxesDePDF(datos, max)
	case tipoAdxDocx, tipoAdxOdt, tipoAdxPres, tipoAdxFolla:
		zr, err := abrirZip(datos)
		if err != nil {
			return nil, err
		}
		var out []imaxeExtraida
		for _, f := range zr.File {
			if f.FileInfo().IsDir() || !extImaxeRe.MatchString(f.Name) {
				continue
			}
			if !strings.Contains(f.Name, "/media/") && !strings.HasPrefix(f.Name, "Pictures/") {
				continue
			}
			rc, err := f.Open()
			if err != nil {
				continue
			}
			b, _ := io.ReadAll(io.LimitReader(rc, 30<<20))
			rc.Close()
			out = append(out, imaxeExtraida{Nome: path.Base(f.Name), Datos: b})
			if len(out) >= max {
				break
			}
		}
		return out, nil
	}
	return nil, fmt.Errorf("«%s» (%s) non ten imaxes que extraer", ax.Nome, ax.Tipo)
}

func imaxesDePDF(datos []byte, max int) ([]imaxeExtraida, error) {
	if !commandExists("pdfimages") {
		return nil, fmt.Errorf("non se atopou pdfimages (paquete poppler-utils) para sacar as imaxes do PDF")
	}
	ruta, limpar, err := escribirTemp(datos, "doc.pdf")
	if err != nil {
		return nil, err
	}
	defer limpar()
	dir := filepath.Dir(ruta)
	if _, err := executarComTempo(120, "pdfimages", "-png", ruta, filepath.Join(dir, "img")); err != nil {
		return nil, err
	}
	nomes, _ := filepath.Glob(filepath.Join(dir, "img-*.png"))
	sort.Strings(nomes)
	var out []imaxeExtraida
	for _, n := range nomes {
		b, err := os.ReadFile(n)
		if err != nil {
			continue
		}
		// Saltar as miudezas (filetes, puntos): menos de 48 px por algún lado.
		if cfg, _, err := imageConfig(b); err != nil || cfg.Width < 48 || cfg.Height < 48 {
			continue
		}
		out = append(out, imaxeExtraida{Nome: filepath.Base(n), Datos: b})
		if len(out) >= max {
			break
		}
	}
	return out, nil
}

// renderizarPaxina debuxa a páxina `pax` (desde 1) dun adxunto en PDF, ou de
// calquera documento de oficina (convertido antes con LibreOffice), como PNG.
func renderizarPaxina(ax Adxunto, datos []byte, pax int) ([]byte, error) {
	if !commandExists("pdftoppm") {
		return nil, idioma.Errorf("go.ler.sen_pdftoppm", "non se atopou pdftoppm (paquete poppler-utils) para ver as páxinas")
	}
	if pax < 1 {
		pax = 1
	}
	pdfDatos := datos
	if ax.Tipo != tipoAdxPDF {
		pdf, limpar, err := converterAPDFLibreOffice(ax, datos)
		if err != nil {
			return nil, err
		}
		defer limpar()
		if pdfDatos, err = os.ReadFile(pdf); err != nil {
			return nil, err
		}
	}
	ruta, limpar, err := escribirTemp(pdfDatos, "doc.pdf")
	if err != nil {
		return nil, err
	}
	defer limpar()
	dir := filepath.Dir(ruta)
	if _, err := executarComTempo(90, "pdftoppm", "-f", strconv.Itoa(pax), "-l", strconv.Itoa(pax), "-r", "110", "-png", ruta, filepath.Join(dir, "pax")); err != nil {
		return nil, err
	}
	nomes, _ := filepath.Glob(filepath.Join(dir, "pax*.png"))
	if len(nomes) == 0 {
		return nil, fmt.Errorf("a páxina %d non existe", pax)
	}
	return os.ReadFile(nomes[0])
}

var reTituloMD = regexp.MustCompile(`(?m)^(#{1,6})\s+(.+?)\s*#*\s*$`)

// estruturaMarkdown devolve unha cabeceira co que ten un Markdown adxunto
// (frontmatter YAML, índice de títulos e enlaces [[wiki]]/locais) para que a
// IA entenda a súa estrutura e as relacións con outros ficheiros; "" se non
// ten nada disto.
func estruturaMarkdown(t string) string {
	var b strings.Builder
	cuerpo := t
	if strings.HasPrefix(t, "---\n") || strings.HasPrefix(t, "---\r\n") {
		if i := strings.Index(t[3:], "\n---"); i >= 0 {
			fm := strings.TrimSpace(t[3 : 3+i])
			if len([]rune(fm)) > 800 {
				fm = string([]rune(fm)[:800]) + "…"
			}
			b.WriteString("Frontmatter:\n" + fm + "\n")
			cuerpo = t[3+i+4:]
		}
	}
	var tit []string
	for _, m := range reTituloMD.FindAllStringSubmatch(cuerpo, 40) {
		tit = append(tit, strings.Repeat("  ", len(m[1])-1)+m[2])
	}
	if len(tit) > 0 {
		b.WriteString("Títulos:\n" + strings.Join(tit, "\n") + "\n")
	}
	if ls := enlacesDe(cuerpo); len(ls) > 0 {
		if len(ls) > 40 {
			ls = ls[:40]
		}
		b.WriteString("Enlaces: " + strings.Join(ls, ", ") + "\n")
	}
	if b.Len() == 0 {
		return ""
	}
	return "[Estrutura do Markdown]\n" + b.String() + "[Texto]\n"
}
