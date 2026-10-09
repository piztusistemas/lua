// Package lectura le exames e materiais en PDF por capas: as páxinas de
// texto simple coa capa de texto do PDF (exacta e de balde) e as que teñen
// fórmulas, figuras ou están escaneadas coa visión da IA, transcritas
// literalmente a Markdown + LaTeX. Despois compara as dúas lecturas para
// marcar o que a IA puido inventar ou cambiar. Ver
// luayang/proba_lectura/README.md.
package lectura

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"edicion/plataforma"
)

// Métodos de lectura dunha páxina.
const (
	MetodoTexto  = "texto"
	MetodoVision = "vision"
)

// Paxina: o que se sabe dunha páxina antes de lela.
type Paxina struct {
	Num        int    `json:"num"`
	Caracteres int    `json:"caracteres"` // na capa de texto
	FontesMath bool   `json:"fontesMath"`
	Imaxes     int    `json:"imaxes"` // imaxes que non se repiten noutras páxinas
	Escaneada  bool   `json:"escaneada"`
	Metodo     string `json:"metodo"`
	Motivo     string `json:"motivo"`
}

func executar(nome string, args ...string) (string, error) {
	cmd := exec.Command(nome, args...)
	plataforma.OcultarConsola(cmd)
	var out, erro bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &erro
	if err := cmd.Run(); err != nil {
		return out.String(), fmt.Errorf("%s: %v: %s", nome, err, strings.TrimSpace(erro.String()))
	}
	return out.String(), nil
}

// HaiPoppler di se están as ferramentas de poppler.
func HaiPoppler() bool {
	for _, c := range []string{"pdfinfo", "pdftotext", "pdffonts", "pdfimages", "pdftoppm"} {
		if !plataforma.CommandExists(c) {
			return false
		}
	}
	return true
}

var paxinasRe = regexp.MustCompile(`(?m)^Pages:\s+(\d+)`)

// NumPaxinas dun PDF.
func NumPaxinas(pdf string) (int, error) {
	out, err := executar("pdfinfo", pdf)
	if err != nil {
		return 0, err
	}
	m := paxinasRe.FindStringSubmatch(out)
	if m == nil {
		return 0, fmt.Errorf("non se puido ler o número de páxinas")
	}
	return strconv.Atoi(m[1])
}

// fontesMath recoñece as fontes de fórmulas (editor de ecuacións de Word,
// LaTeX, MathType…).
var fontesMathRe = regexp.MustCompile(`(?i)math|cmmi|cmsy|cmex|msam|msbm|stix|symbol|mtextra|euler|rsfs`)

// Clasificar decide como ler cada páxina, sen IA.
func Clasificar(pdf string) ([]Paxina, error) {
	if !HaiPoppler() {
		return nil, fmt.Errorf("falta poppler (pdfinfo, pdftotext, pdftoppm…): instálao para ler PDF")
	}
	n, err := NumPaxinas(pdf)
	if err != nil {
		return nil, err
	}
	// Imaxes que se repiten (o logotipo da cabeceira) non contan coma figuras.
	tamanos := map[string]int{}
	imaxesPorPax := map[int][]string{}
	if out, err := executar("pdfimages", "-list", pdf); err == nil {
		for _, l := range strings.Split(out, "\n")[2:] {
			c := strings.Fields(l)
			if len(c) < 5 || c[2] != "image" {
				continue
			}
			p, _ := strconv.Atoi(c[0])
			k := c[3] + "x" + c[4]
			tamanos[k]++
			imaxesPorPax[p] = append(imaxesPorPax[p], k)
		}
	}
	var out []Paxina
	for i := 1; i <= n; i++ {
		p := Paxina{Num: i}
		txt, _ := executar("pdftotext", "-f", strconv.Itoa(i), "-l", strconv.Itoa(i), pdf, "-")
		p.Caracteres = len(strings.TrimSpace(txt))
		if f, err := executar("pdffonts", "-f", strconv.Itoa(i), "-l", strconv.Itoa(i), pdf); err == nil {
			for _, l := range strings.Split(f, "\n")[2:] {
				if c := strings.Fields(l); len(c) > 0 && fontesMathRe.MatchString(c[0]) {
					p.FontesMath = true
				}
			}
		}
		for _, k := range imaxesPorPax[i] {
			// Unha imaxe que se repite e é baixiña é o logotipo da cabeceira;
			// unha que se repite pero é alta pode ser a mesma figura na
			// versión noutra lingua.
			ancho, alto := 0, 0
			if x := strings.Index(k, "x"); x > 0 {
				ancho, _ = strconv.Atoi(k[:x])
				alto, _ = strconv.Atoi(k[x+1:])
			}
			// Moi ancha e baixa (máis de 4:1 e menos de 120 px): un logotipo.
			logotipo := alto < 120 && ancho > 4*alto
			if !logotipo && (tamanos[k] == 1 || alto > 100) {
				p.Imaxes++
			}
		}
		// Unha imaxe grande e sen texto: páxina escaneada ou fotografada.
		p.Escaneada = p.Caracteres < 40
		switch {
		case p.Escaneada:
			p.Metodo, p.Motivo = MetodoVision, "sen capa de texto (escaneada ou foto)"
		case p.FontesMath:
			p.Metodo, p.Motivo = MetodoVision, "ten fórmulas"
		case p.Imaxes > 0:
			p.Metodo, p.Motivo = MetodoVision, "ten figuras"
		case ocosOuColumnas(txt) != "":
			p.Metodo, p.Motivo = MetodoVision, ocosOuColumnas(txt)
		default:
			p.Metodo, p.Motivo = MetodoTexto, "texto simple"
		}
		out = append(out, p)
	}
	return out, nil
}

// ocosOuColumnas: sinais de que a orde da capa de texto non é fiable
// (exercicios de ocos con liñas «______», táboas en columnas). Nas mostras,
// o exame de inglés tiña o texto de ocos desordenado.
func ocosOuColumnas(txt string) string {
	if strings.Count(txt, "____") >= 3 || strings.Count(txt, "………") >= 3 || strings.Count(txt, "....") >= 6 {
		return "ten ocos para encher (a capa de texto desordénaos)"
	}
	curtas, numeros := 0, 0
	liñas := 0
	for _, l := range strings.Split(txt, "\n") {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		liñas++
		if len([]rune(l)) <= 12 {
			curtas++
			if _, err := strconv.Atoi(l); err == nil {
				numeros++
			}
		}
	}
	// Moitas liñas dunha soa palabra seguidas: unha táboa ou unha caixa de
	// palabras lida por columnas.
	switch {
	case numeros >= 8:
		return "ten as liñas numeradas nunha columna á parte"
	case liñas > 20 && curtas*3 > liñas:
		return "ten táboas ou columnas (a capa de texto desordénaas)"
	}
	return ""
}

// Estimacion do custo de ler un PDF coa IA.
type Estimacion struct {
	Paxinas       int     `json:"paxinas"`
	PaxinasVision int     `json:"paxinasVision"`
	TokensEntrada int     `json:"tokensEntrada"`
	TokensSaida   int     `json:"tokensSaida"`
	Custo         float64 `json:"custo"`
	Moeda         string  `json:"moeda"`
	ConCusto      bool    `json:"conCusto"` // falso se non hai prezos configurados
}

// Cifras aproximadas por páxina (imaxe a 150 ppp + instrucións; resposta en
// Markdown). A estruturación do documento enteiro conta á parte.
const (
	tokensImaxePaxina    = 1700
	tokensInstrucions    = 900
	tokensSaidaVision    = 1400
	tokensTextoPaxina    = 900
	tokensSaidaEstrutura = 500
)

// Estimar calcula canto pode custar ler as páxinas (prezos por millón).
func Estimar(pax []Paxina, prezoEntrada, prezoSaida float64, moeda string) Estimacion {
	e := Estimacion{Paxinas: len(pax), Moeda: moeda}
	for _, p := range pax {
		if p.Metodo == MetodoVision {
			e.PaxinasVision++
			e.TokensEntrada += tokensImaxePaxina + tokensInstrucions + tokensTextoPaxina/2
			e.TokensSaida += tokensSaidaVision
		}
		// Estruturar: todo o texto pasa unha vez máis.
		e.TokensEntrada += tokensTextoPaxina
		e.TokensSaida += tokensSaidaEstrutura
	}
	if prezoEntrada > 0 || prezoSaida > 0 {
		e.ConCusto = true
		e.Custo = float64(e.TokensEntrada)/1e6*prezoEntrada + float64(e.TokensSaida)/1e6*prezoSaida
	}
	return e
}
