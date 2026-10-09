package ficha

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// idaEVolta escribe a ficha, lea e volve escribir: os dous textos teñen que
// ser idénticos e o corpo, exacto.
func idaEVolta(t *testing.T, f *Ficha) *Ficha {
	t.Helper()
	a, err := Escribir(f)
	if err != nil {
		t.Fatalf("%s: Escribir: %v", f.ID, err)
	}
	g, err := Ler(a)
	if err != nil {
		t.Fatalf("%s: Ler: %v\n%s", f.ID, err, a)
	}
	b, err := Escribir(g)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Fatalf("%s: a segunda escritura difire:\n--- 1\n%s\n--- 2\n%s", f.ID, a, b)
	}
	if g.Corpo != f.Corpo {
		t.Fatalf("%s: o corpo cambiou:\n%q\n%q", f.ID, f.Corpo, g.Corpo)
	}
	return g
}

func TestExemplos(t *testing.T) {
	ficheiros, _ := filepath.Glob("testdata/exemplos/*.md")
	if len(ficheiros) == 0 {
		t.Fatal("non hai exemplos")
	}
	for _, p := range ficheiros {
		datos, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		f, err := Ler(datos)
		if err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		if ps := Validar(f); len(ps) > 0 {
			t.Errorf("%s: %v", p, ps)
		}
		idaEVolta(t, f)
	}
}

func TestAreaParabola(t *testing.T) {
	datos, _ := os.ReadFile("testdata/exemplos/area-parabola.md")
	f, err := Ler(datos)
	if err != nil {
		t.Fatal(err)
	}
	orde, err := OrdeCalculo(f)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(orde, []string{"solA", "k", "S", "f"}) {
		t.Errorf("orde = %v", orde)
	}
	var roles []string
	for _, s := range f.Seccions() {
		roles = append(roles, s.Rol)
		if s.Rol == RolEnunciado && !strings.Contains(s.Texto, "${S}$") {
			t.Errorf("enunciado sen {S}: %q", s.Texto)
		}
	}
	if !reflect.DeepEqual(roles, []string{RolEnunciado, RolResultado, RolResolucion}) {
		t.Errorf("roles = %v", roles)
	}
	if f.Fonte == nil || f.Fonte.Paxina != 2 || f.Campos[0].De != "1" {
		t.Errorf("metadatos mal lidos: %+v %+v", f.Fonte, f.Campos[0])
	}
}

func TestOrdeCalculo(t *testing.T) {
	f := &Ficha{Tipo: TipoExercicio, Campos: []Campo{
		{Nome: "c", Orixe: OrixeCalculado, Expr: "a+b"},
		{Nome: "a", Orixe: OrixeAleatorio, Xerador: "n1"},
		{Nome: "f(x)", Orixe: OrixeCalculado, Expr: "c*x^2", Funcion: true},
		{Nome: "b", Orixe: OrixeAleatorio, Xerador: XeradorRango, De: "a", Ata: "a+10"},
		{Nome: "g", Orixe: OrixeCalculado, Expr: `f(2) + concat("a", "b")`},
	}}
	orde, err := OrdeCalculo(f)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(orde, []string{"a", "b", "c", "f", "g"}) {
		t.Errorf("orde = %v", orde)
	}
	f.Campos[1] = Campo{Nome: "a", Orixe: OrixeCalculado, Expr: "g"}
	if _, err := OrdeCalculo(f); err == nil || !strings.Contains(err.Error(), "círculo") {
		t.Errorf("esperaba un ciclo, deu %v", err)
	}
}

func TestValidar(t *testing.T) {
	f := &Ficha{Tipo: "receita", Campos: []Campo{
		{Nome: "1a", Orixe: OrixeFixo, Valor: "2"},
		{Nome: "x", Orixe: OrixeAleatorio, Xerador: "dado"},
		{Nome: "x", Orixe: OrixeFixo},
		{Nome: "t", Orixe: OrixeTaboa},
		{Nome: "s"},
	}}
	ps := Validar(f)
	texto := ""
	for _, p := range ps {
		texto += p.String() + "\n"
	}
	for _, esperado := range []string{"tipo «receita»", "«1a»: o nome", "xerador «dado»", "«x»: o nome está repetido", "precisa «valor»", "precisa «banco»", "«mostra»", "falta a «orixe»"} {
		if !strings.Contains(texto, esperado) {
			t.Errorf("falta %q en:\n%s", esperado, texto)
		}
	}
}

// TestCorpoExacto: o corpo sae igual (os finais de liña de Windows
// normalízanse a \n ao ler, a propósito).
func TestCorpoExacto(t *testing.T) {
	for _, corpo := range []string{"", "\n\ninicio con liñas", "sen salto final", "---\nunha liña de guións no corpo\n---\n"} {
		f := &Ficha{Luayang: 1, Tipo: TipoExame, Corpo: corpo}
		g := idaEVolta(t, f)
		if g.Corpo != corpo {
			t.Errorf("%q -> %q", corpo, g.Corpo)
		}
	}
}

func TestClavesDescoñecidasConsérvanse(t *testing.T) {
	datos := []byte("---\nluayang: 1\ntipo: exercicio\nautor: Ana\netiquetas: [a, b]\n---\n\ntexto\n")
	f, err := Ler(datos)
	if err != nil {
		t.Fatal(err)
	}
	if f.Extra["autor"] != "Ana" {
		t.Fatalf("Extra = %v", f.Extra)
	}
	out, _ := Escribir(f)
	if !strings.Contains(string(out), "autor: Ana") || !strings.Contains(string(out), "etiquetas:") {
		t.Errorf("perdéronse claves:\n%s", out)
	}
}

func TestNonEFicha(t *testing.T) {
	if _, err := Ler([]byte("# Unha nota de Obsidian\n")); err != ErrSenFrontMatter {
		t.Errorf("err = %v", err)
	}
	if _, err := Ler([]byte("---\ntags: [x]\n---\nnota\n")); err == nil {
		t.Error("unha nota con front matter pero sen «luayang» non é unha ficha")
	}
}

// ---------- Yang ----------

func matexTaboa(t *testing.T, modelo any) string {
	t.Helper()
	j, err := json.Marshal(modelo)
	if err != nil {
		t.Fatal(err)
	}
	return "<!--yang:tabla:2 " + base64.StdEncoding.EncodeToString(j) + " -->\n<EX>corpo xerado</EX>\n"
}

// normalizado: o modelo tal como o deixa normalizeModel, co texto sen as
// liñas en branco dos extremos (Yang tamén as quita ao xerar).
func normalizado(m modeloTaboa) modeloTaboa {
	for i := range m.Exercicios {
		e := &m.Exercicios[i]
		e.Enunciado = strings.Trim(e.Enunciado, "\n")
		e.Resposta = strings.Trim(e.Resposta, "\n")
		e.Solucion = strings.Trim(e.Solucion, "\n")
		if strings.TrimSpace(e.Resposta) == "" {
			e.Resposta = ""
		}
		if strings.TrimSpace(e.Solucion) == "" {
			e.Solucion = ""
		}
	}
	return m
}

// comprobarTaboa: importar, pasar cada ficha por disco e reconstruír o
// modelo. Ten que ser igual ao orixinal.
func comprobarTaboa(t *testing.T, nome, texto string) *Importacion {
	t.Helper()
	imp, err := ImportarMatex(texto, nome)
	if err != nil {
		t.Fatalf("%s: %v", nome, err)
	}
	var lidas []*Ficha
	for _, f := range append([]*Ficha{imp.Exame}, imp.Exercicios...) {
		if ps := Validar(f); len(ps) > 0 {
			t.Errorf("%s/%s: %v", nome, f.ID, ps)
		}
		g := idaEVolta(t, f)
		if g.Tipo == TipoExercicio {
			lidas = append(lidas, g)
		}
	}
	m := embedRe.FindStringSubmatch(strings.ReplaceAll(texto, "\r\n", "\n"))
	orixinal, err := lerModeloTaboa(m[2])
	if err != nil {
		t.Fatal(err)
	}
	if got, want := ModeloTaboa(lidas), normalizado(orixinal); !reflect.DeepEqual(got, want) {
		g, _ := json.MarshalIndent(got, "", " ")
		w, _ := json.MarshalIndent(want, "", " ")
		t.Fatalf("%s: o modelo non volve igual:\n--- volta\n%s\n--- orixinal\n%s", nome, g, w)
	}
	return imp
}

func TestYangTaboa(t *testing.T) {
	modelo := map[string]any{"v": 2, "exercicios": []any{
		map[string]any{
			"id": "e1", "titulo": "Área do triángulo",
			"enunciado": "\nCalcula a área dun triángulo de base {b} e altura {h}.\n<TIKZ>\\draw (0,0) -- ({b},0);</TIKZ>",
			"variables": []any{
				map[string]any{"nome": "b", "modo": "aleatoria", "xer": "rango", "min": "4", "max": "20", "paso": "2"},
				map[string]any{"nome": "h", "modo": "aleatoria", "xer": "n1"},
				map[string]any{"nome": "g", "modo": "fixa", "valor": 9.8},
				map[string]any{"nome": "f(x)", "modo": "auto", "expr": "x^2", "defOp": ":="},
				map[string]any{"nome": "k", "modo": "estraño"},
			},
			"resultados": []any{
				map[string]any{"etiqueta": "a)", "nome": "res_1", "formula": "b*h/2"},
				map[string]any{"etiqueta": "", "nome": "", "formula": ""},
			},
			"resposta": "", "solucion": "<p>Área = <MAT>b*h/2</MAT> = {res_1}</p>",
		},
		map[string]any{"id": "e2", "titulo": "", "enunciado": "<p><b>Exercicio 2:</b> deriva <MAT>f(x)</MAT></p>", "variables": []any{}, "resposta": "{{literal}}", "solucion": " "},
	}}
	imp := comprobarTaboa(t, "Exame de proba", matexTaboa(t, modelo))
	if len(imp.Exercicios) != 2 || imp.Exame.Corpo != "![[exame-de-proba-1]]\n\n![[exame-de-proba-2]]\n" {
		t.Fatalf("exame: %q", imp.Exame.Corpo)
	}
	e1 := imp.Exercicios[0]
	if e1.Campos[0].Paso != "2" || e1.Campos[2].Valor != "9.8" || !e1.Campos[3].Funcion || e1.Campos[4].Xerador != "n1" || e1.Campos[5].Rol != RolResultado {
		t.Errorf("campos: %+v", e1.Campos)
	}
	if imp.Exercicios[1].Titulo != "Exercicio 2: deriva f(x)" {
		t.Errorf("título sacado do texto: %q", imp.Exercicios[1].Titulo)
	}
}

func TestYangTexto(t *testing.T) {
	ficheiros, _ := filepath.Glob("testdata/yang/*.matex")
	for _, p := range ficheiros {
		comprobarTexto(t, p)
	}
}

func comprobarTexto(t *testing.T, p string) *Importacion {
	t.Helper()
	datos, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	orixinal := strings.ReplaceAll(string(datos), "\r\n", "\n")
	nome := strings.TrimSuffix(filepath.Base(p), ".matex")
	imp, err := ImportarMatex(orixinal, nome)
	if err != nil {
		t.Fatalf("%s: %v", p, err)
	}
	porID := []*Ficha{}
	for _, f := range imp.Exercicios {
		porID = append(porID, idaEVolta(t, f))
	}
	exame := idaEVolta(t, imp.Exame)
	volta, err := TextoMatex(exame, porID)
	if err != nil {
		t.Fatal(err)
	}
	if volta != orixinal {
		t.Fatalf("%s: o .matex non volve igual", p)
	}
	return imp
}

// ---------- Lua ----------

// mesmoJSON compara dous JSON polo seu significado.
func mesmoJSON(t *testing.T, a, b []byte) bool {
	t.Helper()
	var x, y any
	if err := json.Unmarshal(a, &x); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &y); err != nil {
		t.Fatal(err)
	}
	return reflect.DeepEqual(x, y)
}

func comprobarLua(t *testing.T, nome string, datos []byte) *Ficha {
	t.Helper()
	f, err := ImportarFormularioLua(datos)
	if err != nil {
		t.Fatalf("%s: %v", nome, err)
	}
	if ps := Validar(f); len(ps) > 0 {
		t.Errorf("%s: %v", nome, ps)
	}
	g := idaEVolta(t, f)
	volta, err := FormularioLua(g)
	if err != nil {
		t.Fatal(err)
	}
	if !mesmoJSON(t, datos, volta) {
		t.Fatalf("%s: o formulario non volve igual:\n%s", nome, volta)
	}
	return g
}

func TestLua(t *testing.T) {
	form := `{"id":"ficha_gramatica","nome":"Ficha de gramática","icona":"lapis","modo":"deseno","titulo":"Grammar practice",
	"campos":[{"id":"contido","tipo":"texto","etiqueta":"Tema"},
	{"id":"nivel","tipo":"escolla","etiqueta":"Nivel","opcions":["A1","A2"],"obrigatorio":true},
	{"id":"exercicios","tipo":"taboa","etiqueta":"Exercicios","columnas":[{"id":"enunciado","tipo":"texto","etiqueta":"Enunciado"},{"id":"ok","tipo":"verificacion","etiqueta":"Feito"}]},
	{"id":"oculto","tipo":"texto","etiqueta":"Variable","agochado":true}],
	"fluxo":{"nodos":[{"id":"c1","tipo":"campo","campo":"contido","x":120.5,"y":40},{"id":"t1","tipo":"titulo","config":{"nivel":"3"},"entradas":{"texto":"c1"}}],"documento":["t1"]}}`
	f := comprobarLua(t, "proba", []byte(form))
	if f.Campos[2].Columnas[1].Orixe != "" || f.Formulario.Titulo != "Grammar practice" {
		t.Errorf("%+v", f)
	}
}

// ---------- ficheiros reais ----------

// TestReais proba ficheiros de verdade que non van no repositorio (poden ter
// datos persoais). LUAYANG_REAIS: lista de ficheiros ou cartafoles separados
// por «:» (.matex de Yang e .json de formularios de Lua).
func TestReais(t *testing.T) {
	lista := os.Getenv("LUAYANG_REAIS")
	if lista == "" {
		t.Skip("sen LUAYANG_REAIS")
	}
	for _, r := range filepath.SplitList(lista) {
		var ficheiros []string
		if st, err := os.Stat(r); err == nil && st.IsDir() {
			m1, _ := filepath.Glob(filepath.Join(r, "*.matex"))
			m2, _ := filepath.Glob(filepath.Join(r, "*.json"))
			ficheiros = append(m1, m2...)
		} else {
			ficheiros = []string{r}
		}
		for _, p := range ficheiros {
			t.Run(filepath.Base(p), func(t *testing.T) {
				datos, err := os.ReadFile(p)
				if err != nil {
					t.Fatal(err)
				}
				switch {
				case strings.HasSuffix(p, ".json"):
					var probe struct {
						Campos []any `json:"campos"`
					}
					if json.Unmarshal(datos, &probe) != nil || probe.Campos == nil {
						t.Skip("non é un formulario de Lua")
					}
					f := comprobarLua(t, p, datos)
					t.Logf("formulario: %d campos, tubería: %v", len(f.Campos), f.Calculo != nil)
				case embedRe.MatchString(strings.ReplaceAll(string(datos), "\r\n", "\n")):
					imp := comprobarTaboa(t, strings.TrimSuffix(filepath.Base(p), ".matex"), string(datos))
					t.Logf("táboa: %d exercicios %v", len(imp.Exercicios), imp.Avisos)
				default:
					imp := comprobarTexto(t, p)
					t.Logf("texto: %d exercicios %v", len(imp.Exercicios), imp.Avisos)
				}
			})
		}
	}
}

func TestSeccionsLingua(t *testing.T) {
	f := &Ficha{Tipo: TipoExercicio, Corpo: "## Enunciado\n\nGalego\n\n## Enunciado [es]\n\nCastellano\n\n## Resolución [es]\n\nx\n"}
	ss := f.Seccions()
	if len(ss) != 3 || ss[1].Lingua != "es" || ss[1].Rol != RolEnunciado || ss[2].Rol != RolResolucion || ss[0].Lingua != "" {
		t.Fatalf("%+v", ss)
	}
}
