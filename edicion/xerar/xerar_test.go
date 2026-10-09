package xerar

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"edicion/documento"
	"edicion/ficha"
)

func lerFicha(t *testing.T, ruta string) *ficha.Ficha {
	t.Helper()
	d, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	f, err := ficha.Ler(d)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// resolverDir: as fichas dun cartafol polo seu nome de ficheiro.
func resolverDir(dir string) Fichas {
	return func(id string) (*ficha.Ficha, error) {
		d, err := os.ReadFile(filepath.Join(dir, id+".md"))
		if err != nil {
			return nil, err
		}
		return ficha.Ler(d)
	}
}

func precisa(t *testing.T, cmds ...string) {
	t.Helper()
	for _, c := range cmds {
		if _, err := exec.LookPath(c); err != nil {
			t.Skipf("falta %s", c)
		}
	}
}

func compilar(t *testing.T, dir, corpo string) {
	t.Helper()
	if _, err := exec.LookPath("xelatex"); err != nil {
		return
	}
	r, err := documento.Compilar(dir, corpo, "xelatex", nil, nil, "proba", nil)
	if err != nil {
		t.Fatalf("%v\n%s", err, ultimasLiñas(r.Log, 30))
	}
	if len(r.PDF) < 1000 {
		t.Fatal("PDF baleiro")
	}
	if os.Getenv("LUAYANG_GARDAR_PDF") != "" {
		_ = os.WriteFile(filepath.Join(os.Getenv("LUAYANG_GARDAR_PDF"), t.Name()+".pdf"), r.PDF, 0o644)
	}
}

func ultimasLiñas(s string, n int) string {
	l := strings.Split(s, "\n")
	if len(l) > n {
		l = l[len(l)-n:]
	}
	return strings.Join(l, "\n")
}

func TestAreaParabolaMaxima(t *testing.T) {
	precisa(t, "maxima", "pandoc")
	f := lerFicha(t, "../ficha/testdata/exemplos/area-parabola.md")
	dir := t.TempDir()
	r, err := Corpo(f, nil, Opcions{Variantes: 3, Semente: 1, Modo: ModoResolucion}, dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Avisos) > 0 {
		t.Errorf("avisos: %v", r.Avisos)
	}
	if strings.Contains(r.Corpo, "{S}") || strings.Contains(r.Corpo, "{solA}") || strings.Contains(r.Corpo, "MATEXERAWBLOCK") {
		t.Fatalf("quedaron campos sen substituír:\n%s", r.Corpo)
	}
	// O resultado ten que cadrar: S = solA*k^3/6 en cada variante.
	for i, v := range r.Valores {
		if !v["S"].EsNum || !v["solA"].EsNum || !v["k"].EsNum {
			t.Fatalf("variante %d: valores non numéricos %+v", i, v)
		}
		k := v["k"].Num
		if d := v["S"].Num - v["solA"].Num*k*k*k/6; d > 1e-9 || d < -1e-9 {
			t.Errorf("variante %d: S=%v solA=%v k=%v", i, v["S"].Num, v["solA"].Num, k)
		}
	}
	if !strings.Contains(r.Corpo, "Resolución") || !strings.Contains(r.Corpo, `\includegraphics`) && !strings.Contains(r.Corpo, "images/img_") {
		t.Errorf("falta a resolución ou o gráfico:\n%s", r.Corpo)
	}
	compilar(t, dir, r.Corpo)
}

func TestEnunciadosSenSolucions(t *testing.T) {
	precisa(t, "maxima", "pandoc")
	f := lerFicha(t, "../ficha/testdata/exemplos/area-parabola.md")
	r, err := Corpo(f, nil, Opcions{Semente: 4, Modo: ModoEnunciados}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(r.Corpo, "Resultado") || strings.Contains(r.Corpo, "Resolución") {
		t.Errorf("a folla do alumnado leva solucións:\n%s", r.Corpo)
	}
}

func TestInglesBancoSenMaxima(t *testing.T) {
	precisa(t, "pandoc")
	dir := "../ficha/testdata/exemplos"
	f := lerFicha(t, filepath.Join(dir, "transformacions-ingles.md"))
	if PrecisaMaxima(f) {
		t.Fatal("unha ficha de letras non debería precisar Maxima")
	}
	tmp := t.TempDir()
	r1, err := Corpo(f, resolverDir(dir), Opcions{Semente: 7, Modo: ModoResultado}, tmp)
	if err != nil {
		t.Fatal(err)
	}
	r2, _ := Corpo(f, resolverDir(dir), Opcions{Semente: 7, Modo: ModoResultado}, t.TempDir())
	if r1.Corpo != r2.Corpo {
		t.Error("a mesma semente ten que dar o mesmo documento")
	}
	filas := r1.Valores[0]["frases"].Filas
	if len(filas) != 6 {
		t.Fatalf("filas = %d", len(filas))
	}
	vistas := map[string]bool{}
	for _, fl := range filas {
		if vistas[fl["estrutura"]] {
			t.Errorf("estrutura repetida: %s", fl["estrutura"])
		}
		vistas[fl["estrutura"]] = true
		if !strings.Contains(r1.Corpo, strings.Split(fl["pista"], "'")[0]) {
			t.Errorf("falta a pista %q", fl["pista"])
		}
	}
	if !strings.Contains(r1.Corpo, `\begin{enumerate}`) {
		t.Errorf("a lista non saíu:\n%s", r1.Corpo)
	}
	compilar(t, tmp, r1.Corpo)
}

func TestExameMatexDeYang(t *testing.T) {
	precisa(t, "maxima")
	datos, err := os.ReadFile("../ficha/testdata/yang/bloques.matex")
	if err != nil {
		t.Fatal(err)
	}
	imp, err := ficha.ImportarMatex(string(datos), "bloques")
	if err != nil {
		t.Fatal(err)
	}
	porID := map[string]*ficha.Ficha{}
	for _, e := range imp.Exercicios {
		porID[e.ID] = e
	}
	res := func(id string) (*ficha.Ficha, error) {
		if f, ok := porID[id]; ok {
			return f, nil
		}
		return nil, fmt.Errorf("non existe %s", id)
	}
	dir := t.TempDir()
	r, err := Corpo(imp.Exame, res, Opcions{Semente: 1}, dir)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r.Corpo, "Exercicio 1") || !strings.Contains(r.Corpo, "tikzpicture") {
		t.Errorf("corpo:\n%s", r.Corpo)
	}
	compilar(t, dir, r.Corpo)
}

func TestTaboaYangConSolucions(t *testing.T) {
	precisa(t, "maxima")
	modelo := `<!--yang:tabla:2 ` + b64(`{"v":2,"exercicios":[{"id":"e1","titulo":"Triángulo","enunciado":"Calcula a área dun triángulo de base {b} cm e altura {h} cm.","variables":[{"nome":"b","modo":"aleatoria","xer":"rango","min":"4","max":"20","paso":"2"},{"nome":"h","modo":"aleatoria","xer":"n1"}],"resultados":[{"etiqueta":"Área","nome":"res_1","formula":"b*h/2"}],"resposta":"","solucion":"<p>Área = <MAT>b*h/2</MAT> = {res_1} cm²</p>"}]}`) + " -->\n"
	imp, err := ficha.ImportarMatex(modelo, "triangulo")
	if err != nil {
		t.Fatal(err)
	}
	e := imp.Exercicios[0]
	for _, modo := range []string{ModoEnunciados, ModoResultado, ModoResolucion} {
		dir := t.TempDir()
		r, err := Corpo(e, nil, Opcions{Semente: 3, Modo: modo, Variantes: 2}, dir)
		if err != nil {
			t.Fatal(modo, err)
		}
		temArea := strings.Contains(r.Corpo, "Área")
		if (modo == ModoEnunciados) == temArea {
			t.Errorf("%s: corpo\n%s", modo, r.Corpo)
		}
		compilar(t, dir, r.Corpo)
	}
}

func b64(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

func TestProbas(t *testing.T) {
	precisa(t, "maxima")
	f := lerFicha(t, "../ficha/testdata/exemplos/area-parabola.md")
	inf, err := Probar(f, nil, Opcions{}, 12)
	if err != nil {
		t.Fatal(err)
	}
	if !inf.Ben() || inf.Distintas < 6 || len(inf.Mostra) == 0 {
		t.Fatalf("%+v", inf)
	}
	// Unha ficha que divide por cero nalgunha semente.
	mala := &ficha.Ficha{Luayang: 1, Tipo: ficha.TipoExercicio, Campos: []ficha.Campo{
		{Nome: "a", Orixe: ficha.OrixeAleatorio, Xerador: ficha.XeradorRango, De: "0", Ata: "2"},
		{Nome: "r", Orixe: ficha.OrixeCalculado, Motor: ficha.MotorMaxima, Expr: "1/a"},
	}, Probas: &ficha.Probas{Condicions: []string{"a > 0"}}, Corpo: "## Enunciado\n\n{r}\n"}
	inf, err = Probar(mala, nil, Opcions{}, 15)
	if err != nil {
		t.Fatal(err)
	}
	if inf.Ben() || !strings.Contains(inf.Resumo(), "«r»") {
		t.Fatalf("non detectou os fallos: %s", inf.Resumo())
	}
}

func TestProbasSenMaxima(t *testing.T) {
	precisa(t, "pandoc")
	dir := "../ficha/testdata/exemplos"
	f := lerFicha(t, filepath.Join(dir, "transformacions-ingles.md"))
	inf, err := Probar(f, resolverDir(dir), Opcions{}, 10)
	if err != nil || !inf.Ben() || inf.Distintas < 5 {
		t.Fatalf("%+v %v", inf, err)
	}
	f.Campos[0].Mostra = 9
	inf, _ = Probar(f, resolverDir(dir), Opcions{}, 3)
	if inf.Ben() {
		t.Fatal("o banco non ten 9 filas de estruturas distintas")
	}
}

func TestProbasMatex(t *testing.T) {
	precisa(t, "maxima")
	modelo := `<!--yang:tabla:2 ` + b64(`{"v":2,"exercicios":[{"id":"e1","titulo":"Cociente","enunciado":"Calcula {a}/{b}.","variables":[{"nome":"a","modo":"aleatoria","xer":"n1"},{"nome":"b","modo":"aleatoria","xer":"rango","min":"0","max":"3"}],"resultados":[{"etiqueta":"","nome":"res_1","formula":"a/b"}],"resposta":"","solucion":""}]}`) + " -->\n"
	imp, _ := ficha.ImportarMatex(modelo, "c")
	inf, err := Probar(imp.Exercicios[0], nil, Opcions{}, 20)
	if err != nil {
		t.Fatal(err)
	}
	if inf.Ben() {
		t.Fatalf("b pode ser 0 e non se detectou: %+v", inf)
	}
}

func TestComentarioNonSaeNoPDF(t *testing.T) {
	precisa(t, "pandoc")
	f := &ficha.Ficha{Luayang: 1, Tipo: ficha.TipoExercicio, Corpo: "## Enunciado\n\nTexto.\n<!-- LuaYang: proposta de reconstrución (plot)\nplot2d(x^2,[x,-1,1])\n-->\n"}
	r, err := Corpo(f, nil, Opcions{}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(r.Corpo, "plot2d") || strings.Contains(r.Corpo, "LuaYang") {
		t.Errorf("o comentario saíu no documento:\n%s", r.Corpo)
	}
}
