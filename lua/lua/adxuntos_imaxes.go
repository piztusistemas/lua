package main

// Formatos de imaxe que Go non le por si só (RAW das cámaras, SVG, HEIC,
// AVIF, EPS...) e logotipos en PDF. Convértense a PNG con ferramentas
// externas SE están instaladas (ImageMagick, Inkscape, dcraw, poppler); Lua
// di con claridade cal fai falta se non hai ningunha. Os PNG, JPEG, GIF, BMP,
// TIFF e WebP léense directamente.

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Extensións de imaxes que se converten con ferramentas externas.
var (
	extensionsRaw = map[string]bool{
		"raw": true, "dng": true, "cr2": true, "cr3": true, "crw": true, "nef": true, "nrw": true, "arw": true,
		"srf": true, "sr2": true, "orf": true, "rw2": true, "raf": true, "pef": true, "srw": true, "3fr": true,
		"erf": true, "mrw": true, "kdc": true, "dcr": true, "x3f": true, "iiq": true,
	}
	extensionsExternas = map[string]bool{
		"svg": true, "svgz": true, "heic": true, "heif": true, "avif": true, "eps": true, "ai": true,
		"psd": true, "jp2": true, "j2k": true, "xcf": true, "emf": true, "wmf": true, "ico": true, "tga": true, "pcx": true, "jxl": true,
	}
)

func extensionDe(nome string) string {
	return strings.ToLower(strings.TrimPrefix(filepath.Ext(nome), "."))
}

func precisaConversorExterno(nome string) bool {
	e := extensionDe(nome)
	return extensionsRaw[e] || extensionsExternas[e]
}

func comandoImageMagick() string {
	for _, c := range []string{"magick", "convert"} {
		if commandExists(c) {
			return c
		}
	}
	return ""
}

// converterImaxeExternaAPNG converte calquera imaxe de formato raro a PNG.
func converterImaxeExternaAPNG(datos []byte, nome string) ([]byte, error) {
	ext := extensionDe(nome)
	if ext == "" {
		ext = "bin"
	}
	ruta, limpar, err := escribirTemp(datos, "orixe."+ext)
	if err != nil {
		return nil, err
	}
	defer limpar()
	dir := filepath.Dir(ruta)
	saida := filepath.Join(dir, "saida.png")
	var ultimoErro error
	intentar := func(nomeCmd string, args ...string) bool {
		if !commandExists(nomeCmd) {
			return false
		}
		if _, err := executarComTempo(120, nomeCmd, args...); err != nil {
			ultimoErro = err
			return false
		}
		b, err := os.ReadFile(saida)
		return err == nil && len(b) > 0
	}
	switch {
	case ext == "svg" || ext == "svgz":
		if intentar("inkscape", ruta, "--export-type=png", "--export-width=2000", "--export-filename="+saida) {
			return os.ReadFile(saida)
		}
	case extensionsRaw[ext]:
		// dcraw escribe un TIFF a stdout: pásase a PNG con Go.
		if commandExists("dcraw") {
			if out, err := executarComTempo(180, "dcraw", "-c", "-w", "-T", ruta); err == nil && len(out) > 0 {
				if img, _, err := image.Decode(bytes.NewReader(out)); err == nil {
					var b bytes.Buffer
					if png.Encode(&b, img) == nil {
						return b.Bytes(), nil
					}
				}
			} else if err != nil {
				ultimoErro = err
			}
		}
	}
	if im := comandoImageMagick(); im != "" {
		args := []string{ruta + "[0]", "-auto-orient", "-background", "none", "-resize", "4000x4000>", "png:" + saida}
		if im == "magick" {
			args = append([]string{"convert"}, args...)
			if intentar("magick", args...) {
				return os.ReadFile(saida)
			}
		} else if intentar("convert", args...) {
			return os.ReadFile(saida)
		}
	}
	msg := fmt.Sprintf("non se puido converter «%s» (formato %s)", nome, strings.ToUpper(ext))
	switch {
	case extensionsRaw[ext]:
		msg += ": fan falta dcraw ou ImageMagick con soporte RAW (sudo apt install dcraw imagemagick), ou exporta a foto a PNG/JPEG"
	case ext == "svg" || ext == "svgz":
		msg += ": fai falta Inkscape ou ImageMagick, ou exporta o logotipo a PNG ou PDF"
	default:
		msg += ": fai falta ImageMagick (sudo apt install imagemagick), ou exporta a imaxe a PNG/JPEG"
	}
	if ultimoErro != nil {
		msg += " (" + finalDe(ultimoErro.Error(), 200) + ")"
	}
	return nil, fmt.Errorf("%s", msg)
}

// recortarBordosBrancos quita as beiras brancas ou transparentes dunha imaxe
// (un logotipo en PDF adoita vir nunha páxina enteira). Deixa 1% de marxe.
func recortarBordosBrancos(datos []byte) []byte {
	img, _, err := image.Decode(bytes.NewReader(datos))
	if err != nil {
		return datos
	}
	b := img.Bounds()
	minX, minY, maxX, maxY := b.Max.X, b.Max.Y, b.Min.X, b.Min.Y
	baleiro := func(c color.Color) bool {
		r, g, bl, a := c.RGBA()
		if a < 0x1000 {
			return true
		}
		return r > 0xF8F8 && g > 0xF8F8 && bl > 0xF8F8
	}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if !baleiro(img.At(x, y)) {
				minX, maxX = min(minX, x), max(maxX, x)
				minY, maxY = min(minY, y), max(maxY, y)
			}
		}
	}
	if maxX < minX || maxY < minY {
		return datos // toda branca: non tocar
	}
	marxe := max(2, (maxX-minX)/100)
	r := image.Rect(max(b.Min.X, minX-marxe), max(b.Min.Y, minY-marxe), min(b.Max.X, maxX+marxe+1), min(b.Max.Y, maxY+marxe+1))
	if r.Dx() >= b.Dx() && r.Dy() >= b.Dy() {
		return datos
	}
	sub, ok := img.(interface {
		SubImage(image.Rectangle) image.Image
	})
	if !ok {
		return datos
	}
	var out bytes.Buffer
	if png.Encode(&out, sub.SubImage(r)) != nil {
		return datos
	}
	return out.Bytes()
}

// pdfAPNGLogo debuxa unha páxina dun PDF (por defecto a 1) coma PNG a `dpi`
// puntos por polgada e quítalle as beiras brancas: sirve para usar un
// logotipo en PDF onde só cabe unha imaxe de píxeles (Word, OpenDocument,
// o logotipo do curso).
func pdfAPNGLogo(pdf []byte, pax, dpi int) ([]byte, error) {
	if !commandExists("pdftoppm") {
		return nil, fmt.Errorf("non se atopou pdftoppm (paquete poppler-utils) para usar o PDF coma imaxe")
	}
	if pax < 1 {
		pax = 1
	}
	if dpi < 72 {
		dpi = 300
	}
	ruta, limpar, err := escribirTemp(pdf, "logo.pdf")
	if err != nil {
		return nil, err
	}
	defer limpar()
	dir := filepath.Dir(ruta)
	if _, err := executarComTempo(90, "pdftoppm", "-f", strconv.Itoa(pax), "-l", strconv.Itoa(pax), "-r", strconv.Itoa(dpi), "-png", ruta, filepath.Join(dir, "pax")); err != nil {
		return nil, err
	}
	nomes, _ := filepath.Glob(filepath.Join(dir, "pax*.png"))
	if len(nomes) == 0 {
		return nil, fmt.Errorf("a páxina %d do PDF non existe", pax)
	}
	b, err := os.ReadFile(nomes[0])
	if err != nil {
		return nil, err
	}
	return recortarBordosBrancos(b), nil
}
