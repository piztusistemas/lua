package main

// Recortes de LaTeX: fórmulas e debuxos (TikZ, circuitikz) compilados a PNG.
// Serve a Word e OpenDocument (formulas_ofi.go), que non teñen editor de
// ecuacións nin TikZ, e á vista do formulario (os circuítos dos problemas,
// coma os <TIKZ> de Yang). Todas as pezas van nun único LaTeX co paquete
// preview (unha páxina axustada por peza); se a compilación falla, cada
// peza compílase soa para saber cal ten o erro.

import (
	"bytes"
	"crypto/sha256"
	"edicion/idioma"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// recorteLatex é unha peza xa compilada.
type recorteLatex struct {
	im  *imaxeOfi
	svg []byte // en vez de im, se se pediu SVG (pp = 0)
	// anchoPt: o ancho real da peza (para amosala ao seu tamaño).
	anchoPt float64
	// Alto e profundidade en puntos (a liña base, para as fórmulas en liña).
	altoPt, fondoPt float64
	erro            string
}

// medidaRe: a liña que escribe documentoRecortes con alto, profundidade e
// ancho de cada peza (en pt).
var medidaRe = regexp.MustCompile(`LUAMEDIDA (\d+) (-?[\d.]+)pt/(-?[\d.]+)pt/(-?[\d.]+)pt`)

// motorRecortes: o primeiro motor de LaTeX que haxa (pdflatex é o máis rápido).
func motorRecortes() string {
	for _, m := range []string{"pdflatex", "xelatex", "lualatex"} {
		if commandExists(m) {
			return m
		}
	}
	return ""
}

func documentoRecortes(motor string, pezas []string) string {
	var b strings.Builder
	b.WriteString("\\documentclass[10pt]{article}\n")
	if motor == "pdflatex" {
		b.WriteString("\\usepackage[utf8]{inputenc}\n\\usepackage[T1]{fontenc}\n" + latexUnicodeDeclarations)
	}
	b.WriteString("\\usepackage{amsmath,amssymb}\n\\usepackage{tikz}\n\\usepackage{circuitikz}\n\\usepackage[active,tightpage]{preview}\n\\setlength\\PreviewBorder{0.5pt}\n\\pagestyle{empty}\n\\begin{document}\n")
	for i, p := range pezas {
		// Medida da peza (para a liña base das fórmulas e o tamaño real).
		fmt.Fprintf(&b, "\\setbox0=\\hbox{%s}\\typeout{LUAMEDIDA %d \\the\\ht0/\\the\\dp0/\\the\\wd0}\n", p, i+1)
		b.WriteString("\\begin{preview}" + p + "\\end{preview}\n\n")
	}
	b.WriteString("\\end{document}\n")
	return b.String()
}

// compilarRecortes compila as pezas (LaTeX que vai dentro dun preview) a
// PNG de `pp` puntos por polgada, ou a SVG (vectorial) se pp é 0. O erro é só se non hai LaTeX ou pdftoppm;
// o dunha peza vai no seu recorte.
func compilarRecortes(pezas []string, pp int) ([]recorteLatex, error) {
	out := make([]recorteLatex, len(pezas))
	if len(pezas) == 0 {
		return out, nil
	}
	motor := motorRecortes()
	if motor == "" || !commandExists("pdftoppm") {
		return out, fmt.Errorf("cómpre LaTeX e pdftoppm")
	}
	if _, err := compilarLote(motor, pezas, pp, out); err == nil {
		return out, nil
	}
	// Algunha peza ten un erro: cada unha soa (e a que falla, ao rexistro).
	for i, p := range pezas {
		un := make([]recorteLatex, 1)
		if log, err := compilarLote(motor, []string{p}, pp, un); err != nil {
			out[i] = recorteLatex{erro: err.Error()}
			rexistrar(EntradaRexistro{Tipo: "debuxo", Documento: "fórmula ou debuxo TikZ", Motor: motor, Erro: err.Error(), Log: log, Tex: documentoRecortes(motor, []string{p})})
			continue
		}
		out[i] = un[0]
	}
	return out, nil
}

func compilarLote(motor string, pezas []string, pp int, out []recorteLatex) (string, error) {
	dir, err := os.MkdirTemp("", "lua-recortes-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)
	pdf, log, err := compilarLatexCrono(dir, documentoRecortes(motor, pezas), motor)
	if err != nil || len(pdf) == 0 {
		if err == nil {
			err = idioma.Errorf("go.latex.sen_paxinas", "LaTeX non deu ningunha páxina")
		}
		return log, err
	}
	dims := map[int][3]float64{}
	for _, m := range medidaRe.FindAllStringSubmatch(log, -1) {
		n, _ := strconv.Atoi(m[1])
		h, _ := strconv.ParseFloat(m[2], 64)
		d, _ := strconv.ParseFloat(m[3], 64)
		w, _ := strconv.ParseFloat(m[4], 64)
		dims[n] = [3]float64{h, d, w + 1} // + o PreviewBorder (0,5 pt por lado)
	}
	if pp == 0 {
		// SVG: unha chamada a pdftocairo por páxina.
		for i := range pezas {
			n := strconv.Itoa(i + 1)
			ruta := filepath.Join(dir, "f-"+n+".svg")
			cmd := exec.Command("pdftocairo", "-svg", "-f", n, "-l", n, filepath.Join(dir, "doc.pdf"), ruta)
			ocultarConsola(cmd)
			var erro bytes.Buffer
			cmd.Stderr = &erro
			if err := cmd.Run(); err != nil {
				return log, fmt.Errorf("pdftocairo: %v %s", err, strings.TrimSpace(erro.String()))
			}
			datos, err := os.ReadFile(ruta)
			if err != nil {
				return log, err
			}
			d := dims[i+1]
			out[i] = recorteLatex{svg: datos, altoPt: d[0], fondoPt: d[1], anchoPt: d[2]}
		}
		return log, nil
	}
	cmd := exec.Command("pdftoppm", "-r", strconv.Itoa(pp), "-png", filepath.Join(dir, "doc.pdf"), filepath.Join(dir, "f"))
	ocultarConsola(cmd)
	var erro bytes.Buffer
	cmd.Stderr = &erro
	if err := cmd.Run(); err != nil {
		return log, fmt.Errorf("pdftoppm: %v %s", err, strings.TrimSpace(erro.String()))
	}
	pngs, _ := filepath.Glob(filepath.Join(dir, "f-*.png"))
	sort.Slice(pngs, func(i, j int) bool { return numeroPaxina(pngs[i]) < numeroPaxina(pngs[j]) })
	if len(pngs) != len(pezas) {
		return log, fmt.Errorf("saíron %d páxinas de %d", len(pngs), len(pezas))
	}
	for i, ruta := range pngs {
		datos, err := os.ReadFile(ruta)
		if err != nil {
			return log, err
		}
		im, err := imaxeOfiDeURI("data:image/png;base64," + base64.StdEncoding.EncodeToString(datos))
		if err != nil {
			return log, err
		}
		d := dims[i+1]
		out[i] = recorteLatex{im: im, altoPt: d[0], fondoPt: d[1], anchoPt: d[2]}
	}
	return log, nil
}

func numeroPaxina(ruta string) int {
	base := strings.TrimSuffix(filepath.Base(ruta), ".png")
	n, _ := strconv.Atoi(base[strings.LastIndex(base, "-")+1:])
	return n
}

// ---------- Debuxos (```tikz e ```circuitikz) ----------

// Tipos de debuxo dun bloque con valla.
const (
	debuxoTikz       = "tikz"
	debuxoCircuitikz = "circuitikz"
)

// etiquetaConComaRe: unha etiqueta de circuitikz (l=, v=, i=, a=...) entre
// $…$ sen chaves. Se leva unha coma (\, ou 1,5) a coma partiría as opcións.
var etiquetaConComaRe = regexp.MustCompile(`\b([lvia][_^]?)=(\$[^$]*,[^$]*\$)`)

// envolverDebuxo pon o contorno se o código non o trae e protexe as
// etiquetas con comas: l=$7\,\Omega$ -> l={$7\,\Omega$}.
func envolverDebuxo(tipo, codigo string) string {
	c := etiquetaConComaRe.ReplaceAllString(strings.TrimSpace(codigo), "$1={$2}")
	if strings.Contains(c, `\begin{tikzpicture}`) || strings.Contains(c, `\begin{circuitikz}`) {
		return c
	}
	if tipo == debuxoCircuitikz {
		// Símbolos europeos (resistencia rectangular), os do ensino de aquí.
		return "\\begin{circuitikz}[european, scale=0.9, transform shape]\n" + c + "\n\\end{circuitikz}"
	}
	return "\\begin{tikzpicture}\n" + c + "\n\\end{tikzpicture}"
}

var valadoDebuxoRe = regexp.MustCompile("(?s)```(tikz|circuitikz)[ \\t]*\\n(.*?)\\n```")

// debuxosDe: os debuxos dun texto Markdown (tipo e código), na orde.
func debuxosDe(md string) [][2]string {
	var out [][2]string
	for _, m := range valadoDebuxoRe.FindAllStringSubmatch(md, -1) {
		out = append(out, [2]string{m[1], m[2]})
	}
	return out
}

// claveDebuxo identifica un debuxo (tipo e código) para a caché e a vista.
func claveDebuxo(tipo, codigo string) string {
	h := sha256.Sum256([]byte(tipo + "\n" + strings.TrimSpace(codigo)))
	return hex.EncodeToString(h[:12])
}

// DebuxoVista: un debuxo para o formulario (imaxe ou erro).
type DebuxoVista struct {
	Imaxe string `json:"imaxe,omitempty"` // data URI (SVG vectorial; PNG se non hai pdftocairo)
	// AnchoPx: o ancho real en px CSS, para velo co tamaño que ten no documento.
	AnchoPx float64 `json:"anchoPx,omitempty"`
	Erro    string  `json:"erro,omitempty"`
}

// cacheDebuxos garda os debuxos xa compilados (a vista recalcula a cada
// pausa ao teclear; un circuíto que non cambia non se volve compilar).
var cacheDebuxos = struct {
	sync.Mutex
	m map[string]DebuxoVista
}{m: map[string]DebuxoVista{}}

// corTextoVista: a cor do texto da interface (--text-dark de style.css).
const corTextoVista = "#22252f"

// ppVista: resolución dos debuxos na vista se non hai pdftocairo (con el
// van en SVG, vectoriais: nítidos a calquera tamaño).
const ppVista = 300

// debuxarParaVista compila os debuxos que faltan e devolve todos por clave.
func debuxarParaVista(debuxos [][2]string) map[string]DebuxoVista {
	out := map[string]DebuxoVista{}
	var pendentes [][2]string
	var claves []string
	cacheDebuxos.Lock()
	for _, d := range debuxos {
		k := claveDebuxo(d[0], d[1])
		if v, ok := cacheDebuxos.m[k]; ok {
			out[k] = v
		} else if _, xa := out[k]; !xa {
			out[k] = DebuxoVista{}
			pendentes = append(pendentes, d)
			claves = append(claves, k)
		}
	}
	cacheDebuxos.Unlock()
	if len(pendentes) == 0 {
		return out
	}
	pezas := make([]string, len(pendentes))
	for i, d := range pendentes {
		pezas[i] = envolverDebuxo(d[0], d[1])
	}
	pp := ppVista
	if commandExists("pdftocairo") {
		pp = 0
	}
	recortes, err := compilarRecortes(pezas, pp)
	cacheDebuxos.Lock()
	defer cacheDebuxos.Unlock()
	for i, k := range claves {
		v := DebuxoVista{}
		switch {
		case err != nil:
			v.Erro = "Non se pode debuxar: " + err.Error()
		case recortes[i].erro != "":
			v.Erro = recortes[i].erro
		case recortes[i].svg != nil:
			// Fondo transparente e o negro coa cor do texto da app: o debuxo
			// intégrase no formulario en vez de parecer unha captura pegada.
			svg := strings.ReplaceAll(string(recortes[i].svg), "rgb(0%, 0%, 0%)", corTextoVista)
			v.Imaxe = "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(svg))
			v.AnchoPx = recortes[i].anchoPt * 96 / 72.27
		default:
			v.Imaxe = "data:image/png;base64," + base64.StdEncoding.EncodeToString(recortes[i].im.Datos)
			v.AnchoPx = float64(recortes[i].im.W) * 96 / float64(pp)
		}
		out[k] = v
		if err == nil { // sen LaTeX non se garda: pódese instalar despois
			if len(cacheDebuxos.m) > 500 {
				cacheDebuxos.m = map[string]DebuxoVista{}
			}
			cacheDebuxos.m[k] = v
		}
	}
	return out
}
