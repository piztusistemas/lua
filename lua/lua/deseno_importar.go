package main

// «Fai que se pareza a isto»: a partir dun documento que lle gusta ao
// profesorado (PDF ou Word), Lúa propón un deseño: a combinación de letras
// máis parecida, as cores e o tamaño. O resultado é un deseño normal que se
// pode seguir editando; a explicación di de onde saíu cada cousa.

import (
	"archive/zip"
	"bytes"
	"context"
	"edicion/idioma"
	"edicion/latex"
	"encoding/base64"
	"encoding/xml"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ImportacionDeseno é a proposta de deseño e de onde saíu.
type ImportacionDeseno struct {
	Deseno      Deseno   `json:"deseno"`
	Explicacion []string `json:"explicacion"`
}

// combinacionParecida devolve a combinación máis parecida a unha letra polo
// seu nome ("" se non a recoñece).
func combinacionParecida(nome string) string {
	n := strings.ToLower(strings.ReplaceAll(nome, " ", ""))
	grupos := []struct {
		comb   string
		claves []string
	}{
		{"elegante", []string{"palatino", "pagella", "bookantiqua", "palladio", "pplr"}},
		{"lexible", []string{"century", "schola", "schoolbook", "atkinson", "andika"}},
		{"academica", []string{"libertin", "latinmodern", "lmroman", "cmr", "cambria", "georgia", "garamond", "minion", "sourceserif", "charter"}},
		{"clasica", []string{"times", "termes", "liberationserif", "nimbusrom", "tinos", "ptmr"}},
		{"moderna", []string{"fira", "opensans", "calibri", "roboto", "segoe", "sourcesans", "lato", "notosans", "verdana", "carlito", "inter"}},
		{"lua", []string{"helvetica", "arial", "heros", "liberationsans", "nimbussans", "arimo", "phvr"}},
	}
	for _, g := range grupos {
		for _, c := range g.claves {
			if strings.Contains(n, c) {
				return g.comb
			}
		}
	}
	return ""
}

// ---------- Word ----------

type docxRPr struct {
	Fonts *struct {
		ASCII      string `xml:"ascii,attr"`
		ASCIITheme string `xml:"asciiTheme,attr"`
	} `xml:"rFonts"`
	Sz *struct {
		Val string `xml:"val,attr"`
	} `xml:"sz"`
	Color *struct {
		Val string `xml:"val,attr"`
	} `xml:"color"`
}

type docxStyles struct {
	Defaults struct {
		RPr docxRPr `xml:"rPrDefault>rPr"`
	} `xml:"docDefaults"`
	Styles []struct {
		ID  string  `xml:"styleId,attr"`
		RPr docxRPr `xml:"rPr"`
	} `xml:"style"`
}

var (
	temaLatinRe  = regexp.MustCompile(`<a:(major|minor)Font>\s*<a:latin typeface="([^"]*)"`)
	temaAccentRe = regexp.MustCompile(`<a:accent1>\s*<a:srgbClr val="([0-9A-Fa-f]{6})"`)
)

func lerZipDeseno(zr *zip.Reader, nome string) []byte {
	for _, f := range zr.File {
		if f.Name == nome {
			rc, err := f.Open()
			if err != nil {
				return nil
			}
			defer rc.Close()
			data, _ := io.ReadAll(io.LimitReader(rc, 20<<20))
			return data
		}
	}
	return nil
}

// importarDesenoDocx le as letras, o tamaño e a cor dos títulos dun .docx.
func importarDesenoDocx(data []byte) (ImportacionDeseno, error) {
	var out ImportacionDeseno
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return out, idioma.Errorf("go.deseno.docx_non_valido", "non é un documento de Word (.docx) válido: %w", err)
	}
	var st docxStyles
	if s := lerZipDeseno(zr, "word/styles.xml"); s != nil {
		_ = xml.Unmarshal(s, &st)
	}
	tema := map[string]string{}
	acento := ""
	if t := lerZipDeseno(zr, "word/theme/theme1.xml"); t != nil {
		for _, m := range temaLatinRe.FindAllStringSubmatch(string(t), -1) {
			tema[m[1]] = m[2]
		}
		if m := temaAccentRe.FindStringSubmatch(string(t)); m != nil {
			acento = strings.ToUpper(m[1])
		}
	}
	fonte := func(r docxRPr) string {
		if r.Fonts == nil {
			return ""
		}
		if r.Fonts.ASCII != "" {
			return r.Fonts.ASCII
		}
		switch {
		case strings.HasPrefix(r.Fonts.ASCIITheme, "major"):
			return tema["major"]
		case strings.HasPrefix(r.Fonts.ASCIITheme, "minor"):
			return tema["minor"]
		}
		return ""
	}
	corpo := fonte(st.Defaults.RPr)
	if corpo == "" {
		corpo = tema["minor"]
	}
	titulos, corTitulos := "", ""
	for _, s := range st.Styles {
		if s.ID == "Normal" {
			if f := fonte(s.RPr); f != "" {
				corpo = f
			}
			if s.RPr.Sz != nil && st.Defaults.RPr.Sz == nil {
				st.Defaults.RPr.Sz = s.RPr.Sz
			}
		}
		id := strings.ToLower(s.ID)
		if id == "heading1" || id == "ttulo1" || id == "titulo1" || id == "título1" {
			titulos = fonte(s.RPr)
			if s.RPr.Color != nil {
				corTitulos = s.RPr.Color.Val
			}
		}
	}
	if titulos == "" {
		titulos = tema["major"]
	}
	if c := combinacionParecida(corpo); c != "" {
		out.Deseno.Letra = c
		out.Explicacion = append(out.Explicacion, idioma.T("go.deseno.imp_texto", "o texto vai en «%s»: propoño a letra «%s»", corpo, c))
	} else if corpo != "" {
		out.Explicacion = append(out.Explicacion, idioma.T("go.deseno.imp_texto_sen_letra", "o texto vai en «%s», que non coñezo: deixo a letra de Lúa", corpo))
	}
	if ct := combinacionParecida(titulos); ct != "" && ct != out.Deseno.Letra {
		if comb, ok := latex.CombinacionPorID(ct); ok {
			out.Deseno.LetraTitulos = comb.Titulos
			out.Explicacion = append(out.Explicacion, idioma.T("go.deseno.imp_titulos", "os títulos van en «%s»: póñoos coma en «%s»", titulos, ct))
		}
	}
	if st.Defaults.RPr.Sz != nil {
		if medios, err := strconv.ParseFloat(st.Defaults.RPr.Sz.Val, 64); err == nil && medios > 0 {
			out.Deseno.Tamano = medios / 2
			out.Explicacion = append(out.Explicacion, idioma.T("go.deseno.imp_tamano", "o texto ten %s pt", trimFloat(medios/2)))
		}
	}
	cor, _ := normalizarCorHex(corTitulos)
	if cor == "" {
		cor, _ = normalizarCorHex(acento)
	}
	if cor != "" && cor != "000000" {
		out.Deseno.CorPrincipal = cor
		out.Explicacion = append(out.Explicacion, idioma.T("go.deseno.imp_cor_titulos", "a cor dos títulos é #%s", cor))
	}
	return out, nil
}

// ---------- PDF ----------

var pdffontsNomeRe = regexp.MustCompile(`(?m)^(?:[A-Z]{6}\+)?([^\s]+)`)

// importarDesenoPdf le as letras (pdffonts) e as cores da primeira páxina
// (pdftoppm) dun PDF.
func importarDesenoPdf(data []byte) (ImportacionDeseno, error) {
	var out ImportacionDeseno
	if !commandExists("pdffonts") {
		return out, idioma.Errorf("go.deseno.pdffonts_falta", "para ler un PDF fai falla pdffonts (Poppler): instala poppler-utils")
	}
	dir, err := os.MkdirTemp("", "lua-importar-*")
	if err != nil {
		return out, err
	}
	defer os.RemoveAll(dir)
	pdf := filepath.Join(dir, "mostra.pdf")
	if err := os.WriteFile(pdf, data, 0o644); err != nil {
		return out, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "pdffonts", "-l", "3", pdf)
	ocultarConsola(cmd)
	saida, err := cmd.Output()
	if err != nil {
		return out, idioma.Errorf("go.deseno.pdf_ler", "non se puido ler o PDF: %w", err)
	}
	liñas := strings.Split(string(saida), "\n")
	// Contan as letras que non son de símbolos nin matemáticas; a primeira
	// recoñecida é a do texto (pdffonts lista na orde de aparición).
	var nomes []string
	for _, l := range liñas[min(2, len(liñas)):] {
		if m := pdffontsNomeRe.FindStringSubmatch(l); m != nil {
			n := m[1]
			if strings.Contains(strings.ToLower(n), "math") || strings.Contains(strings.ToLower(n), "symbol") {
				continue
			}
			nomes = append(nomes, n)
		}
	}
	for _, n := range nomes {
		if c := combinacionParecida(n); c != "" {
			out.Deseno.Letra = c
			out.Explicacion = append(out.Explicacion, idioma.T("go.deseno.imp_pdf_letra", "o documento usa «%s»: propoño a letra «%s»", n, c))
			break
		}
	}
	if out.Deseno.Letra == "" && len(nomes) > 0 {
		out.Explicacion = append(out.Explicacion, idioma.T("go.deseno.imp_pdf_sen_letra", "non recoñezo a letra «%s»: deixo a de Lúa", nomes[0]))
	}
	if commandExists("pdftoppm") {
		ctx2, cancel2 := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel2()
		c := exec.CommandContext(ctx2, "pdftoppm", "-r", "40", "-f", "1", "-l", "1", "-png", pdf, filepath.Join(dir, "p"))
		ocultarConsola(c)
		if c.Run() == nil {
			if imxs, _ := filepath.Glob(filepath.Join(dir, "p*.png")); len(imxs) > 0 {
				if f, err := os.Open(imxs[0]); err == nil {
					if img, err := png.Decode(f); err == nil {
						if cores := coresDaImaxe(img, 3); len(cores) > 0 {
							out.Deseno.CorPrincipal = cores[0]
							out.Explicacion = append(out.Explicacion, idioma.T("go.deseno.imp_cor", "a cor que máis se ve é #%s", cores[0]))
						}
					}
					f.Close()
				}
			}
		}
	}
	return out, nil
}

// ImportarDesenoDeFicheiro propón un deseño a partir dun PDF ou dun Word
// (nome do ficheiro e contido en base64).
func (a *App) ImportarDesenoDeFicheiro(nome, datosB64 string) (ImportacionDeseno, error) {
	data, err := base64.StdEncoding.DecodeString(datosB64)
	if err != nil {
		return ImportacionDeseno{}, idioma.Errorf("go.deseno.ficheiro_codificado", "o ficheiro non está ben codificado: %w", err)
	}
	var res ImportacionDeseno
	switch strings.ToLower(filepath.Ext(nome)) {
	case ".docx":
		res, err = importarDesenoDocx(data)
	case ".pdf":
		res, err = importarDesenoPdf(data)
	default:
		return ImportacionDeseno{}, idioma.Errorf("go.deseno.so_pdf_docx", "só se pode copiar o deseño dun PDF ou dun Word (.docx)")
	}
	if err != nil {
		return res, err
	}
	res.Deseno = *normalizarDeseno(&res.Deseno)
	if len(res.Explicacion) == 0 {
		res.Explicacion = []string{idioma.T("go.deseno.imp_nada", "non atopei nada que copiar neste documento")}
	}
	return res, nil
}
