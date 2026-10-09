package main

// Adxuntos do asistente: ficheiros (imaxes, PDF, Word, OpenDocument, follas,
// presentacións, texto...) que o usuario xunta á conversa para que a IA lea a
// información que levan, ou para usar as imaxes (logotipos...) nas plantillas
// e no curso.
//
// GARDADO TEMPORAL: os adxuntos viven nun cartafol de caché
// (<caché do usuario>/lua/adxuntos) só mentres dura a sesión: bórranse ao
// arrincar Lua e ao premer «Nova conversa». Nada disto forma parte do curso.
//
// GARDADO DEFINITIVO (só cando procede, e sempre por petición explícita):
//   - unha imaxe pode pasar ao cartafol de imaxes dunha PLANTILLA
//     (adxunto_gardar_en_plantilla), onde a usan o LaTeX e o deseño de Word;
//   - unha imaxe pode pasar ao propio JSON do CURSO (logotipo do curso,
//     imaxe dunha tarefa): ferramentas curso_logotipo e tarefa_imaxe, que
//     piden o data URI con AdxuntoDataURI.
// Os documentos (PDF, Word...) non se gardan definitivamente: só se len.

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	_ "golang.org/x/image/bmp"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
	// GIF, JPEG e PNG rexístranse noutros ficheiros do paquete.
)

const (
	adxuntoMaxBytes     = 30 << 20  // por ficheiro
	adxuntoMaxTotal     = 250 << 20 // en toda a sesión
	adxuntoMaxCantidade = 80
	// Lado máximo das imaxes que se lle mandan á IA: abonda para ler texto e
	// ver un deseño, sen pagar tokens de máis.
	imaxeIAMaxLado = 1568
	// Lado máximo ao levar unha imaxe ao curso (logotipo/tarefa) por defecto.
	imaxeCursoMaxLado = 1600
)

// Tipos de adxunto.
const (
	tipoAdxImaxe  = "imaxe"
	tipoAdxPDF    = "pdf"
	tipoAdxDocx   = "docx"
	tipoAdxOdt    = "odt"
	tipoAdxPres   = "presentacion"
	tipoAdxFolla  = "folla"
	tipoAdxTexto  = "texto"
	tipoAdxRtf    = "rtf"
	tipoAdxAntigo = "antigo"
	tipoAdxOutro  = "outro"
)

// Adxunto é un ficheiro do almacén temporal.
type Adxunto struct {
	ID      string `json:"id"`
	Nome    string `json:"nome"`
	Tipo    string `json:"tipo"`
	Mime    string `json:"mime"`
	Tamano  int64  `json:"tamano"`
	Ancho   int    `json:"ancho"`   // imaxes
	Alto    int    `json:"alto"`    // imaxes
	Paxinas int    `json:"paxinas"` // PDF
	// Formato: extensión do ficheiro orixinal se se converteu a PNG ao subilo
	// (RAW, SVG, HEIC...); baleiro nos demais.
	Formato string `json:"formato"`
	// Orixe: de que adxunto saíu (páxina renderizada, imaxe extraída dun
	// documento); baleiro se o subiu o usuario.
	Orixe  string `json:"orixe"`
	Creado string `json:"creado"`
}

// almacenAdxuntos é o almacén da sesión. O cartafol pódese cambiar nas
// probas con XDG_CACHE_HOME.
type almacenAdxuntos struct {
	mu      sync.Mutex
	textos  map[string]string // texto extraído xa calculado
	pdfs    map[string]string // ruta do PDF derivado (documentos de oficina -> PDF)
	limpado bool
}

var adxuntos = &almacenAdxuntos{textos: map[string]string{}, pdfs: map[string]string{}}

func dirAdxuntos() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil || base == "" {
		base = os.TempDir()
	}
	dir := filepath.Join(base, "lua", "adxuntos")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

// limparAdxuntosDaSesion borra todo o almacén (ao arrincar e en «Nova conversa»).
func limparAdxuntosDaSesion() error {
	adxuntos.mu.Lock()
	defer adxuntos.mu.Unlock()
	adxuntos.textos = map[string]string{}
	adxuntos.pdfs = map[string]string{}
	dir, err := dirAdxuntos()
	if err != nil {
		return err
	}
	return os.RemoveAll(dir)
}

func novoIDAdxunto() string {
	b := make([]byte, 5)
	_, _ = rand.Read(b)
	return "a" + hex.EncodeToString(b)
}

func rutaDatosAdxunto(dir, id string) string { return filepath.Join(dir, id+".dat") }
func rutaMetaAdxunto(dir, id string) string  { return filepath.Join(dir, id+".json") }

// idAdxuntoValido evita que un id inventado (../../x) saia do almacén.
func idAdxuntoValido(id string) bool {
	if len(id) < 2 || len(id) > 24 {
		return false
	}
	for _, c := range id {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}

// detectarTipoAdxunto decide o tipo polo contido (e a extensión, para os
// formatos que son un ZIP).
func detectarTipoAdxunto(nome string, d []byte) (tipo, mime string) {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(nome), "."))
	switch {
	case bytes.HasPrefix(d, []byte("%PDF")):
		return tipoAdxPDF, "application/pdf"
	case bytes.HasPrefix(d, []byte("\x89PNG")):
		return tipoAdxImaxe, "image/png"
	case bytes.HasPrefix(d, []byte("\xff\xd8")):
		return tipoAdxImaxe, "image/jpeg"
	case bytes.HasPrefix(d, []byte("GIF8")):
		return tipoAdxImaxe, "image/gif"
	case len(d) > 12 && bytes.Equal(d[:4], []byte("RIFF")) && bytes.Equal(d[8:12], []byte("WEBP")):
		return tipoAdxImaxe, "image/webp"
	case bytes.HasPrefix(d, []byte("BM")) && (ext == "bmp" || ext == ""):
		return tipoAdxImaxe, "image/bmp"
	case bytes.HasPrefix(d, []byte("II*\x00")) || bytes.HasPrefix(d, []byte("MM\x00*")):
		return tipoAdxImaxe, "image/tiff"
	case bytes.HasPrefix(d, []byte("PK\x03\x04")):
		switch ext {
		case "docx", "docm", "dotx":
			return tipoAdxDocx, "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
		case "odt", "ott", "fodt":
			return tipoAdxOdt, "application/vnd.oasis.opendocument.text"
		case "pptx", "ppsx":
			return tipoAdxPres, "application/vnd.openxmlformats-officedocument.presentationml.presentation"
		case "odp", "otp":
			return tipoAdxPres, "application/vnd.oasis.opendocument.presentation"
		case "xlsx", "xlsm":
			return tipoAdxFolla, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
		case "ods", "ots":
			return tipoAdxFolla, "application/vnd.oasis.opendocument.spreadsheet"
		}
		// Sen extensión útil: mirar dentro.
		if zt := tipoDentroDoZip(d); zt != "" {
			return zt, "application/zip"
		}
		return tipoAdxOutro, "application/zip"
	case bytes.HasPrefix(d, []byte("{\\rtf")):
		return tipoAdxRtf, "application/rtf"
	case bytes.HasPrefix(d, []byte("\xd0\xcf\x11\xe0")): // OLE2: .doc/.xls/.ppt antigos
		return tipoAdxAntigo, "application/x-ole-storage"
	}
	switch ext {
	case "txt", "md", "markdown", "csv", "tsv", "json", "xml", "html", "htm", "tex", "log", "yaml", "yml", "ini", "srt":
		return tipoAdxTexto, "text/plain"
	case "rtf":
		return tipoAdxRtf, "application/rtf"
	case "doc", "xls", "ppt", "odg", "wpd", "pages", "numbers", "key":
		return tipoAdxAntigo, "application/octet-stream"
	}
	if utf8Aceptable(d) {
		return tipoAdxTexto, "text/plain"
	}
	return tipoAdxOutro, "application/octet-stream"
}

func utf8Aceptable(d []byte) bool {
	if len(d) > 4096 {
		d = d[:4096]
	}
	if bytes.IndexByte(d, 0) >= 0 {
		return false
	}
	return true
}

// AdxuntoGardar mete un ficheiro (en base64, dende o frontend) no almacén
// temporal da sesión.
func (a *App) AdxuntoGardar(nome, dadosB64 string) (Adxunto, error) {
	datos, err := base64.StdEncoding.DecodeString(dadosB64)
	if err != nil {
		return Adxunto{}, fmt.Errorf("o ficheiro non chegou ben: %w", err)
	}
	return gardarAdxunto(nome, datos, "")
}

func gardarAdxunto(nome string, datos []byte, orixe string) (Adxunto, error) {
	if len(datos) == 0 {
		return Adxunto{}, fmt.Errorf("o ficheiro está baleiro")
	}
	if len(datos) > adxuntoMaxBytes {
		return Adxunto{}, fmt.Errorf("o ficheiro pesa %d MB; o máximo son %d MB", len(datos)>>20, adxuntoMaxBytes>>20)
	}
	adxuntos.mu.Lock()
	defer adxuntos.mu.Unlock()
	dir, err := dirAdxuntos()
	if err != nil {
		return Adxunto{}, err
	}
	existentes := listarAdxuntosDir(dir)
	if len(existentes) >= adxuntoMaxCantidade {
		return Adxunto{}, fmt.Errorf("hai demasiados adxuntos (%d); elimina algúns ou inicia unha conversa nova", adxuntoMaxCantidade)
	}
	var total int64
	for _, e := range existentes {
		total += e.Tamano
	}
	if total+int64(len(datos)) > adxuntoMaxTotal {
		return Adxunto{}, fmt.Errorf("os adxuntos da sesión ocupan demasiado (máximo %d MB); elimina algún", adxuntoMaxTotal>>20)
	}

	nome = strings.TrimSpace(filepath.Base(strings.ReplaceAll(nome, "\\", "/")))
	if nome == "" || nome == "." || nome == "/" {
		nome = "adxunto"
	}
	// Formatos de imaxe que Go non le (RAW, SVG, HEIC...): pásanse a PNG xa
	// ao subilos, así o resto do programa só ve imaxes normais.
	formatoOrixinal := ""
	if precisaConversorExterno(nome) {
		png, err := converterImaxeExternaAPNG(datos, nome)
		if err != nil {
			return Adxunto{}, err
		}
		formatoOrixinal = extensionDe(nome)
		datos = png
	}
	tipo, mime := detectarTipoAdxunto(nome, datos)
	ax := Adxunto{Formato: formatoOrixinal, ID: novoIDAdxunto(), Nome: nome, Tipo: tipo, Mime: mime, Tamano: int64(len(datos)), Orixe: orixe, Creado: time.Now().Format("2006-01-02T15:04:05.000000000Z07:00")}
	if tipo == tipoAdxImaxe && formatoOrixinal == "" && !imaxeLexible(datos) {
		// Unha imaxe que Go non le (un TIFF raro, un JPEG estraño): probar co conversor externo.
		if png, err := converterImaxeExternaAPNG(datos, nome); err == nil {
			datos, formatoOrixinal, mime = png, extensionDe(nome), "image/png"
			ax.Formato, ax.Mime, ax.Tamano = formatoOrixinal, mime, int64(len(datos))
		}
	}
	if tipo == tipoAdxImaxe {
		cfg, _, err := image.DecodeConfig(bytes.NewReader(datos))
		if err != nil {
			return Adxunto{}, fmt.Errorf("non se puido ler a imaxe «%s»: %w", nome, err)
		}
		ax.Ancho, ax.Alto = cfg.Width, cfg.Height
	}
	if tipo == tipoAdxPDF {
		ax.Paxinas = paxinasDoPDFBytes(datos)
	}
	if err := os.WriteFile(rutaDatosAdxunto(dir, ax.ID), datos, 0o600); err != nil {
		return Adxunto{}, err
	}
	meta, _ := json.Marshal(ax)
	if err := os.WriteFile(rutaMetaAdxunto(dir, ax.ID), meta, 0o600); err != nil {
		return Adxunto{}, err
	}
	return ax, nil
}

func listarAdxuntosDir(dir string) []Adxunto {
	entradas, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []Adxunto
	for _, e := range entradas {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var ax Adxunto
		if json.Unmarshal(b, &ax) == nil && idAdxuntoValido(ax.ID) {
			out = append(out, ax)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Creado < out[j].Creado || (out[i].Creado == out[j].Creado && out[i].ID < out[j].ID)
	})
	return out
}

// AdxuntosListar devolve os adxuntos da sesión, do máis vello ao máis novo.
func (a *App) AdxuntosListar() []Adxunto {
	adxuntos.mu.Lock()
	defer adxuntos.mu.Unlock()
	dir, err := dirAdxuntos()
	if err != nil {
		return []Adxunto{}
	}
	out := listarAdxuntosDir(dir)
	if out == nil {
		out = []Adxunto{}
	}
	return out
}

// adxuntoPorID devolve os metadatos e o contido dun adxunto.
func adxuntoPorID(id string) (Adxunto, []byte, error) {
	if !idAdxuntoValido(id) {
		return Adxunto{}, nil, fmt.Errorf("id de adxunto non válido: %q (usa adxuntos_listar)", id)
	}
	adxuntos.mu.Lock()
	defer adxuntos.mu.Unlock()
	dir, err := dirAdxuntos()
	if err != nil {
		return Adxunto{}, nil, err
	}
	mb, err := os.ReadFile(rutaMetaAdxunto(dir, id))
	if err != nil {
		return Adxunto{}, nil, fmt.Errorf("non existe o adxunto %q (usa adxuntos_listar)", id)
	}
	var ax Adxunto
	if err := json.Unmarshal(mb, &ax); err != nil {
		return Adxunto{}, nil, err
	}
	datos, err := os.ReadFile(rutaDatosAdxunto(dir, id))
	if err != nil {
		return Adxunto{}, nil, fmt.Errorf("o adxunto %q xa non ten contido", id)
	}
	return ax, datos, nil
}

// AdxuntoEliminar quita un adxunto do almacén.
func (a *App) AdxuntoEliminar(id string) error {
	if !idAdxuntoValido(id) {
		return fmt.Errorf("id de adxunto non válido")
	}
	adxuntos.mu.Lock()
	defer adxuntos.mu.Unlock()
	dir, err := dirAdxuntos()
	if err != nil {
		return err
	}
	delete(adxuntos.textos, id)
	if p, ok := adxuntos.pdfs[id]; ok {
		_ = os.Remove(p)
		delete(adxuntos.pdfs, id)
	}
	_ = os.Remove(rutaMetaAdxunto(dir, id))
	return os.Remove(rutaDatosAdxunto(dir, id))
}

// AdxuntosLimpar borra todos os adxuntos da sesión («Nova conversa»).
func (a *App) AdxuntosLimpar() error { return limparAdxuntosDaSesion() }

// AdxuntoVista devolve unha miniatura (data URI) para amosar nos adxuntos da
// interface; baleira se non é unha imaxe.
func (a *App) AdxuntoVista(id string) (string, error) {
	ax, datos, err := adxuntoPorID(id)
	if err != nil {
		return "", err
	}
	if ax.Tipo != tipoAdxImaxe {
		return "", nil
	}
	out, mime, _, _, err := redimensionarImaxe(datos, 96, "jpeg")
	if err != nil {
		return "", nil
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(out), nil
}

// AdxuntoDataURI devolve unha imaxe do almacén coma data URI, reducida a
// `maxLado` píxeles (0 = 1600): é o formato co que o curso garda as súas
// imaxes no seu JSON (logotipo, imaxes das tarefas). Os PNG (logotipos,
// transparencias) seguen en PNG; as fotos, en JPEG.
func (a *App) AdxuntoDataURI(id string, maxLado int) (string, error) {
	ax, datos, err := adxuntoPorID(id)
	if err != nil {
		return "", err
	}
	if ax.Tipo == tipoAdxPDF {
		// Un logotipo en PDF pasa a PNG (300 ppp, sen beiras) para o JSON do curso.
		if datos, err = pdfAPNGLogo(datos, 1, 300); err != nil {
			return "", err
		}
	} else if ax.Tipo != tipoAdxImaxe {
		return "", fmt.Errorf("«%s» non é unha imaxe nin un PDF (é %s)", ax.Nome, ax.Tipo)
	}
	if maxLado <= 0 {
		maxLado = imaxeCursoMaxLado
	}
	out, mime, _, _, err := redimensionarImaxe(datos, maxLado, "")
	if err != nil {
		return "", err
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(out), nil
}

// redimensionarImaxe decodifica calquera imaxe soportada, redúceaa se algún
// lado pasa de maxLado e recodifícaa. formato: "png", "jpeg" ou "" (o mesmo
// tipo que a orixe se é JPEG; PNG en calquera outro caso, e JPEG se un PNG
// opaco sae moi pesado).
func redimensionarImaxe(datos []byte, maxLado int, formato string) (out []byte, mime string, w, h int, err error) {
	src, fmtOrixe, err := image.Decode(bytes.NewReader(datos))
	if err != nil {
		return nil, "", 0, 0, fmt.Errorf("imaxe non lexible: %w", err)
	}
	b := src.Bounds()
	w, h = b.Dx(), b.Dy()
	if maxLado > 0 && (w > maxLado || h > maxLado) {
		k := float64(maxLado) / float64(max(w, h))
		nw, nh := max(1, int(float64(w)*k+0.5)), max(1, int(float64(h)*k+0.5))
		dst := image.NewNRGBA(image.Rect(0, 0, nw, nh))
		draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Over, nil)
		src, w, h = dst, nw, nh
	}
	opaca := true
	if o, ok := src.(interface{ Opaque() bool }); ok {
		opaca = o.Opaque()
	}
	usarJPEG := false
	switch formato {
	case "jpeg":
		usarJPEG = true
	case "png":
	default:
		usarJPEG = fmtOrixe == "jpeg"
	}
	if usarJPEG {
		// O JPEG non ten transparencia: pousar sobre branco.
		plano := image.NewRGBA(src.Bounds())
		draw.Draw(plano, plano.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)
		draw.Draw(plano, plano.Bounds(), src, src.Bounds().Min, draw.Over)
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, plano, &jpeg.Options{Quality: 86}); err != nil {
			return nil, "", 0, 0, err
		}
		return buf.Bytes(), "image/jpeg", w, h, nil
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, src); err != nil {
		return nil, "", 0, 0, err
	}
	// Un PNG opaco moi pesado (unha foto) sae moito máis pequeno en JPEG.
	if formato == "" && opaca && buf.Len() > 2<<20 {
		return redimensionarImaxe(datos, maxLado, "jpeg")
	}
	return buf.Bytes(), "image/png", w, h, nil
}

// imaxeIA é unha imaxe lista para mandarlle á IA (base64 + tipo).
type imaxeIA struct {
	Mime string
	B64  string
}

// imaxeDoAdxuntoParaIA le unha imaxe do almacén, reducida para a IA.
func imaxeDoAdxuntoParaIA(id string) (imaxeIA, Adxunto, error) {
	ax, datos, err := adxuntoPorID(id)
	if err != nil {
		return imaxeIA{}, ax, err
	}
	if ax.Tipo != tipoAdxImaxe {
		return imaxeIA{}, ax, fmt.Errorf("«%s» non é unha imaxe", ax.Nome)
	}
	out, mime, _, _, err := redimensionarImaxe(datos, imaxeIAMaxLado, "")
	if err != nil {
		return imaxeIA{}, ax, err
	}
	return imaxeIA{Mime: mime, B64: base64.StdEncoding.EncodeToString(out)}, ax, nil
}

// descricionAdxunto: unha liña para a IA cos datos do adxunto.
func descricionAdxunto(ax Adxunto) string {
	d := fmt.Sprintf("%s — «%s», %s, %s", ax.ID, ax.Nome, ax.Tipo, tamanoLexible(ax.Tamano))
	if ax.Tipo == tipoAdxImaxe {
		d += fmt.Sprintf(", %dx%d px", ax.Ancho, ax.Alto)
	}
	if ax.Paxinas > 0 {
		d += fmt.Sprintf(", %d páxinas", ax.Paxinas)
	}
	if ax.Orixe != "" {
		d += ", sacado de " + ax.Orixe
	}
	return d
}

func tamanoLexible(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/float64(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.0f KB", float64(n)/float64(1<<10))
	}
	return fmt.Sprintf("%d B", n)
}

func imageConfig(b []byte) (image.Config, string, error) {
	return image.DecodeConfig(bytes.NewReader(b))
}

// imaxeLexible di se Go sabe decodificar a imaxe.
func imaxeLexible(d []byte) bool {
	_, _, err := image.DecodeConfig(bytes.NewReader(d))
	return err == nil
}
