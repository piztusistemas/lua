package main

// Marcas de posición e corrector despois de compilar.
//
// Cada elemento do deseño escribe, ao saír na páxina, unha liña en doc.lpos
// (\luamarca, deseno_latex.go): «elemento páxina x y», coas coordenadas en sp
// desde a esquina de abaixo á esquerda. A vista previa recibe as marcas
// normalizadas (0..1 desde arriba á esquerda) e, cando o profesorado preme
// nun punto da páxina, abre o elemento que hai alí.

import (
	"bufio"
	"edicion/idioma"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// MarcaDeseno é a posición dun elemento nunha páxina da vista previa.
type MarcaDeseno struct {
	Elemento string  `json:"elemento"`
	Paxina   int     `json:"paxina"` // 1, 2...
	X        float64 `json:"x"`      // 0..1 desde a esquerda
	Y        float64 `json:"y"`      // 0..1 desde arriba
}

// lerMarcasDeseno le o ficheiro de marcas. Sen ficheiro ou sen tamaño de
// páxina devolve nil (a vista previa queda sen inspector, non é un erro).
func lerMarcasDeseno(ruta string) []MarcaDeseno {
	f, err := os.Open(ruta)
	if err != nil {
		return nil
	}
	defer f.Close()
	var ancho, alto float64 // en sp
	var out []MarcaDeseno
	vistas := map[string]bool{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		campos := strings.Fields(sc.Text())
		if len(campos) == 3 && campos[0] == "paxina" {
			ancho, alto = ptASp(campos[1]), ptASp(campos[2])
			continue
		}
		if len(campos) != 4 || ancho <= 0 || alto <= 0 {
			continue
		}
		pax, e1 := strconv.Atoi(campos[1])
		x, e2 := strconv.ParseFloat(campos[2], 64)
		y, e3 := strconv.ParseFloat(campos[3], 64)
		if e1 != nil || e2 != nil || e3 != nil || pax < 1 {
			continue
		}
		m := MarcaDeseno{Elemento: campos[0], Paxina: pax, X: redondear(x / ancho), Y: redondear(1 - y/alto)}
		// A segunda pasada de LaTeX repite as marcas: só unha de cada.
		clave := fmt.Sprintf("%s|%d|%v|%v", m.Elemento, m.Paxina, m.X, m.Y)
		if vistas[clave] {
			continue
		}
		vistas[clave] = true
		out = append(out, m)
	}
	return out
}

// ptASp converte "597.50787pt" a sp.
func ptASp(s string) float64 {
	v, err := strconv.ParseFloat(strings.TrimSuffix(s, "pt"), 64)
	if err != nil {
		return 0
	}
	return v * 65536
}

func redondear(v float64) float64 {
	return float64(int(v*10000+0.5)) / 10000
}

// overfullRe recoñece unha liña que sae da marxe no log de LaTeX.
var overfullRe = regexp.MustCompile(`Overfull \\hbox \(([0-9.]+)pt too wide\)`)

// avisoMarxe conta as liñas que saen da marxe máis de 2 pt (as de menos non se
// ven) e devolve o aviso, ou "".
func avisoMarxe(log string) string {
	n := 0
	for _, m := range overfullRe.FindAllStringSubmatch(log, -1) {
		if v, err := strconv.ParseFloat(m[1], 64); err == nil && v > 2 {
			n++
		}
	}
	if n == 0 {
		return ""
	}
	return idioma.T("go.deseno.overfull", "%d liña(s) saen da marxe: adoita ser un texto moi longo sen espazos (unha ligazón, un correo) ou unha táboa moi ancha", n)
}
