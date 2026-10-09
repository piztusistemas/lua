package xerar

import (
	"strings"
	"testing"
	"time"
)

func TestProblemaSenMaxima(t *testing.T) {
	m, err := NovoMotor(Opcions{}, false, 0)
	if err != nil {
		t.Fatal(err)
	}
	p := Problema{
		Enunciado: "Un coche percorre {d} km en {t} h. Velocidade?",
		Solucion:  "$v = d/t$, logo $v = {v}$ km/h.",
		Variables: []VariableProblema{{Nome: "d", Modo: VariableFixa, Valor: "120"}, {Nome: "t", Modo: VariableAleatoria, De: "2", Ata: "4"}},
		Calculos:  []CalculoProblema{{Nome: "v", Expr: "d/t"}},
	}
	a := m.Calcular(p, SementeProblema(7, 0))
	if a.Erro != "" {
		t.Fatal(a.Erro)
	}
	if !strings.Contains(a.Enunciado, "120") || strings.Contains(a.Enunciado, "{t}") {
		t.Fatalf("enunciado: %q", a.Enunciado)
	}
	if a.Valores["v"].Texto == "" {
		t.Fatalf("sen v: %+v", a.Valores)
	}
	// A mesma semente dá o mesmo; a variable fixa non cambia nunca.
	b := m.Calcular(p, SementeProblema(7, 0))
	if a.Solucion != b.Solucion {
		t.Fatalf("a mesma semente deu %q e %q", a.Solucion, b.Solucion)
	}
	vistos := map[string]bool{}
	for s := 1; s < 30; s++ {
		c := m.Calcular(p, SementeProblema(s, 0))
		if c.Valores["d"].Texto != "120" {
			t.Fatalf("a fixa cambiou: %+v", c.Valores["d"])
		}
		vistos[c.Valores["t"].Texto] = true
	}
	if len(vistos) < 2 {
		t.Fatalf("a variable non varía: %v", vistos)
	}
}

func TestProblemaConMaxima(t *testing.T) {
	precisa(t, "maxima")
	m, err := NovoMotor(Opcions{}, true, 20*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if !m.TenMaxima() {
		t.Skip("sen Maxima")
	}
	p := Problema{
		Enunciado: "Resolve $ {P} = 0 $.",
		Solucion:  "As solucións son $ {S} $. Derivada: {'diff(P,x)}.",
		Resultado: "{S}",
		Variables: []VariableProblema{{Nome: "a", Modo: VariableFixa, Valor: "1"}, {Nome: "r", Modo: VariableAleatoria, De: "1", Ata: "5"}},
		Calculos:  []CalculoProblema{{Nome: "P", Expr: "expand(a*(x-r)*(x+2))"}, {Nome: "S", Expr: "solve(P=0,x)"}},
	}
	c := m.Calcular(p, SementeProblema(3, 1))
	if c.Erro != "" {
		t.Fatal(c.Erro)
	}
	if !c.Maxima || !strings.Contains(c.Enunciado, "x^2") || !strings.Contains(c.Solucion, "x=") {
		t.Fatalf("%+v", c)
	}
	// Unha expresión que Maxima non entende dá aviso, non pecha o motor.
	p.Solucion = "{solve(}"
	c = m.Calcular(p, 1)
	if !m.TenMaxima() {
		t.Fatal("Maxima pechouse por un erro de sintaxe")
	}
	if len(c.Avisos) == 0 {
		t.Fatalf("sen avisos: %+v", c)
	}
}

func TestProblemaComprobacions(t *testing.T) {
	precisa(t, "maxima")
	m, err := NovoMotor(Opcions{}, true, 20*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	p := Problema{
		Enunciado: "Resolve $ {P} = 0 $.",
		Variables: []VariableProblema{{Nome: "r", Modo: VariableAleatoria, De: "1", Ata: "5"}},
		Calculos:  []CalculoProblema{{Nome: "P", Expr: "expand((x-r)*(x+2))"}, {Nome: "S", Expr: "solve(P=0,x)"}},
		// Ben: as dúas solucións anulan P. Mal: unha condición falsa.
		Comprobacions: []string{"every(lambda([s], ratsimp(subst(s, P))=0), S)"},
	}
	if f := m.Probar(p, []int{1, 2, 3}); len(f) > 0 {
		t.Fatalf("fallos inesperados: %v", f)
	}
	p.Comprobacions = []string{"r > 10"}
	if f := m.Probar(p, []int{1, 2}); len(f) == 0 {
		t.Fatal("unha condición falsa non deu fallo")
	}
	p.Comprobacions = nil
	p.Calculos = append(p.Calculos, CalculoProblema{Nome: "Q", Expr: "1/(r-r)"})
	if f := m.Probar(p, []int{1}); len(f) == 0 {
		t.Fatal("unha división por cero non deu fallo")
	}
}

func TestChavesLatexNonSonVariables(t *testing.T) {
	m, _ := NovoMotor(Opcions{}, false, 0)
	p := Problema{Enunciado: "$\\dfrac{d}{t} = \\dfrac{ {d} }{ {t} }$, $x^{t}$ e $ {t} $",
		Variables: []VariableProblema{{Nome: "d", Modo: VariableFixa, Valor: "8"}, {Nome: "t", Modo: VariableFixa, Valor: "2"}}}
	c := m.Calcular(p, 1)
	want := "$\\dfrac{d}{t} = \\dfrac{ {8} }{ {2} }$, $x^{t}$ e $ {2} $"
	if c.Enunciado != want {
		t.Fatalf("\n%q\n%q", c.Enunciado, want)
	}
}

func TestProblemaCircuitoCru(t *testing.T) {
	m, _ := NovoMotor(Opcions{}, false, 0)
	p := Problema{Enunciado: "Calcula a intensidade:\n\n```circuitikz\n\\draw (0,0) to[R, l=$R_1 = {R}\\,\\Omega$] (2,0);\n\\node at (1,1) {texto};\n```\n\nCon {R} ohms.",
		Variables: []VariableProblema{{Nome: "R", Modo: VariableFixa, Valor: "10"}}}
	c := m.Calcular(p, 1)
	if !strings.Contains(c.Enunciado, "```circuitikz\n\\draw (0,0) to[R, l=$R_1 = 10\\,\\Omega$] (2,0);\n\\node at (1,1) {texto};\n```") {
		t.Fatalf("%q", c.Enunciado)
	}
	if len(c.Avisos) > 0 || !strings.Contains(c.Enunciado, "Con $10$ ohms") {
		t.Fatalf("%q %v", c.Enunciado, c.Avisos)
	}
}
