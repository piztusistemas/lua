package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Spec buxo-idiomas, aceptación 4: ningún texto visible de Buxo sen clave nos
// catro idiomas, e o galego do código igual ca o de gl.json.

func lerDicionarioBuxo(t *testing.T, codigo string) map[string]string {
	t.Helper()
	datos, err := os.ReadFile(filepath.Join("idiomas", codigo+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var d map[string]string
	if err := json.Unmarshal(datos, &d); err != nil {
		t.Fatalf("%s.json: %v", codigo, err)
	}
	return d
}

var (
	// idioma.T("clave", "galego"…), idioma.Errorf(…) e log.Info|Aviso|Erro("clave", "galego"…)
	reChamadaGoBuxo = regexp.MustCompile(`(?:idioma\.(?:T|Errorf)|\.(?:Info|Aviso|Erro))\(\s*"([a-z][a-z_.]*)"\s*,\s*("(?:\\.|[^"\\])*")`)
	// t('clave', 'galego'…) no JS
	reChamadaJS = regexp.MustCompile(`\bt\(\s*'([a-z][a-z_.]*)'\s*,\s*'((?:\\'|[^'])*)'`)
	reDataI18n  = regexp.MustCompile(`data-i18n(?:-[a-z-]+)?="([^"]+)"`)
	reVerboBuxo = regexp.MustCompile(`%[sdvqw]`)
	reVarBuxo   = regexp.MustCompile(`\{[A-Za-z_0-9]+\}`)
)

func TestTextosDeBuxoEnGalego(t *testing.T) {
	gl := lerDicionarioBuxo(t, "gl")
	var problemas []string
	revisar := func(onde, clave, galego string) {
		v, ok := gl[clave]
		switch {
		case !ok:
			problemas = append(problemas, onde+": falta "+clave+" en gl.json")
		case galego != "" && v != galego:
			problemas = append(problemas, onde+": "+clave+": o galego do código non é o de gl.json\n    código: "+galego+"\n    json:   "+v)
		}
	}
	gos, _ := filepath.Glob("*.go")
	chamadas := 0
	for _, f := range gos {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		datos, _ := os.ReadFile(f)
		for _, m := range reChamadaGoBuxo.FindAllStringSubmatch(string(datos), -1) {
			galego, err := strconv.Unquote(m[2])
			if err != nil {
				problemas = append(problemas, f+": "+m[1]+": literal non válido")
				continue
			}
			chamadas++
			revisar(f, m[1], galego)
		}
	}
	js, _ := filepath.Glob(filepath.Join("frontend", "src", "*.js"))
	for _, f := range js {
		if strings.HasSuffix(f, ".test.js") {
			continue
		}
		datos, _ := os.ReadFile(f)
		for _, m := range reChamadaJS.FindAllStringSubmatch(string(datos), -1) {
			chamadas++
			revisar(f, m[1], strings.ReplaceAll(m[2], `\'`, `'`))
		}
	}
	html, _ := os.ReadFile(filepath.Join("frontend", "index.html"))
	for _, m := range reDataI18n.FindAllStringSubmatch(string(html), -1) {
		chamadas++
		revisar("index.html", m[1], "")
	}
	if chamadas < 100 {
		t.Fatalf("só %d textos atopados: o escáner non funciona", chamadas)
	}
	sort.Strings(problemas)
	if len(problemas) > 0 {
		t.Errorf("%d problemas en %d textos:\n  %s", len(problemas), chamadas, strings.Join(problemas, "\n  "))
	}
}

func TestIdiomasDeBuxoCoinciden(t *testing.T) {
	gl := lerDicionarioBuxo(t, "gl")
	for _, codigo := range []string{"es", "en", "pt"} {
		d := lerDicionarioBuxo(t, codigo)
		for clave, valor := range gl {
			trad, ok := d[clave]
			if !ok {
				t.Errorf("%s.json: falta %q", codigo, clave)
				continue
			}
			if a, b := reVerboBuxo.FindAllString(valor, -1), reVerboBuxo.FindAllString(trad, -1); !slices.Equal(a, b) {
				t.Errorf("%s.json %q: verbos %v, en galego %v", codigo, clave, b, a)
			}
			a, b := reVarBuxo.FindAllString(valor, -1), reVarBuxo.FindAllString(trad, -1)
			sort.Strings(a)
			sort.Strings(b)
			if !slices.Equal(a, b) {
				t.Errorf("%s.json %q: variables %v, en galego %v", codigo, clave, b, a)
			}
		}
		for clave := range d {
			if _, ok := gl[clave]; !ok {
				t.Errorf("%s.json: sobra %q", codigo, clave)
			}
		}
	}
}
