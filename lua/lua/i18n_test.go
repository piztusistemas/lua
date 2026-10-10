package main

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Constitución, regra 3.2: ningún texto de interface vai cableado. Estes
// tests garanten que toda clave que usa o frontend existe en gl.json e que
// es/en/pt teñen as mesmas claves e as mesmas variables ca o galego.

var idiomasLua = []string{"gl", "es", "en", "pt"}

func lerIdioma(t *testing.T, codigo string) map[string]string {
	t.Helper()
	datos, err := os.ReadFile(filepath.Join("idiomas", codigo+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]string
	if err := json.Unmarshal(datos, &m); err != nil {
		t.Fatalf("%s.json: %v", codigo, err)
	}
	return m
}

// Unha clave estática nunha chamada t('clave', ...), t("clave"...) ou
// t(`clave`...). As que se constrúen ('prefixo.' + x, `prefixo.${x}`) non
// se poden comprobar aquí e quedan fóra.
var reClaveT = regexp.MustCompile("\\bt\\(\\s*(['\"`])([A-Za-z][A-Za-z0-9_.\\-]*)(['\"`])")
var reClaveDOM = regexp.MustCompile(`data-i18n(?:-[a-z-]+)?="([^"]+)"`)

func clavesDoFrontend(t *testing.T) map[string][]string {
	t.Helper()
	usadas := map[string][]string{}
	engadir := func(clave, ficheiro string) {
		if !slices.Contains(usadas[clave], ficheiro) {
			usadas[clave] = append(usadas[clave], ficheiro)
		}
	}
	ler := func(ruta string) {
		datos, err := os.ReadFile(ruta)
		if err != nil {
			t.Fatal(err)
		}
		s := string(datos)
		for _, m := range reClaveT.FindAllStringSubmatch(s, -1) {
			if m[1] != m[3] || strings.HasSuffix(m[2], ".") {
				continue
			}
			engadir(m[2], ruta)
		}
		for _, m := range reClaveDOM.FindAllStringSubmatch(s, -1) {
			engadir(m[1], ruta)
		}
	}
	ler(filepath.Join("frontend", "index.html"))
	err := filepath.WalkDir(filepath.Join("frontend", "src"), func(ruta string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(ruta, ".js") && !strings.HasSuffix(ruta, ".test.js") {
			ler(ruta)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return usadas
}

func TestClavesDoFrontendEstanEnGalego(t *testing.T) {
	gl := lerIdioma(t, "gl")
	usadas := clavesDoFrontend(t)
	if len(usadas) < 500 {
		t.Fatalf("só se atoparon %d claves: o escáner do frontend non funciona", len(usadas))
	}
	var faltan []string
	for clave, ficheiros := range usadas {
		if _, ok := gl[clave]; !ok {
			faltan = append(faltan, clave+"  ("+strings.Join(ficheiros, ", ")+")")
		}
	}
	sort.Strings(faltan)
	if len(faltan) > 0 {
		t.Errorf("%d claves usadas no frontend non están en idiomas/gl.json:\n  %s", len(faltan), strings.Join(faltan, "\n  "))
	}
}

// As variables {x} do frontend poden ir en calquera orde; os verbos de
// formato do Go (%s, %d, %v, %q, %w) insírense por posición, así que teñen
// que ir na mesma orde en todos os idiomas.
var reVariable = regexp.MustCompile(`\{[A-Za-z_0-9]+\}`)
var reVerbo = regexp.MustCompile(`%[sdvqw]`)

func TestIdiomasTenAsMesmasClavesEVariables(t *testing.T) {
	gl := lerIdioma(t, "gl")
	for _, codigo := range idiomasLua[1:] {
		outro := lerIdioma(t, codigo)
		for clave, valor := range gl {
			trad, ok := outro[clave]
			if !ok {
				t.Errorf("%s.json: falta %q", codigo, clave)
				continue
			}
			a, b := reVariable.FindAllString(valor, -1), reVariable.FindAllString(trad, -1)
			sort.Strings(a)
			sort.Strings(b)
			if !slices.Equal(a, b) {
				t.Errorf("%s.json %q: variables %v, en galego %v", codigo, clave, b, a)
			}
			if va, vb := reVerbo.FindAllString(valor, -1), reVerbo.FindAllString(trad, -1); !slices.Equal(va, vb) {
				t.Errorf("%s.json %q: verbos de formato %v, en galego %v (mesma orde)", codigo, clave, vb, va)
			}
		}
		for clave := range outro {
			if _, ok := gl[clave]; !ok {
				t.Errorf("%s.json: sobra %q (non está en gl.json)", codigo, clave)
			}
		}
	}
}

// Unha chamada idioma.T("clave", "galego", ...) ou idioma.Errorf(...) no Go.
var reChamadaGo = regexp.MustCompile("idioma\\.(?:T|Errorf)\\(\\s*\"([^\"]+)\"\\s*,\\s*(\"(?:\\\\.|[^\"\\\\])*\"|`[^`]*`)?")

// TestMensaxesDoGoEstanEnGalego (spec mensaxes-go-idiomas, aceptación 5):
// toda mensaxe do Go traducible de Lua e de edicion ten a súa clave en
// gl.json, e o texto galego do código é o mesmo ca o do JSON.
func TestMensaxesDoGoEstanEnGalego(t *testing.T) {
	gl := lerIdioma(t, "gl")
	var problemas []string
	chamadas := 0
	revisar := func(ruta string) {
		datos, err := os.ReadFile(ruta)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range reChamadaGo.FindAllStringSubmatch(string(datos), -1) {
			chamadas++
			clave := m[1]
			if m[2] == "" {
				problemas = append(problemas, ruta+": "+clave+": o texto galego ten que ser un literal")
				continue
			}
			galego, err := strconv.Unquote(m[2])
			if err != nil {
				problemas = append(problemas, ruta+": "+clave+": literal non válido")
				continue
			}
			valor, ok := gl[clave]
			switch {
			case !ok:
				problemas = append(problemas, ruta+": falta "+clave+" en gl.json")
			case valor != galego:
				problemas = append(problemas, ruta+": "+clave+": o galego do código non é o de gl.json\n    código: "+galego+"\n    json:   "+valor)
			}
		}
	}
	for _, raiz := range []string{".", filepath.Join("..", "..", "edicion")} {
		err := filepath.WalkDir(raiz, func(ruta string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() && (d.Name() == "node_modules" || d.Name() == "frontend" || d.Name() == "build") {
				return fs.SkipDir
			}
			if !d.IsDir() && strings.HasSuffix(ruta, ".go") && !strings.HasSuffix(ruta, "_test.go") {
				revisar(ruta)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	sort.Strings(problemas)
	if len(problemas) > 0 {
		t.Errorf("%d problemas en %d chamadas:\n  %s", len(problemas), chamadas, strings.Join(problemas, "\n  "))
	}
}
