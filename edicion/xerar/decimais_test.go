package xerar

import (
	"strings"
	"testing"
	"time"

	"edicion/ficha"
)

// problemaOhm é o caso do informe: I aleatoria de 0,1 a 0,5 con paso 0,1
// daba 0.30000000000000004 (0,1 + 2·0,1 en coma flotante) e V = I·R
// 54.00000000000001.
func problemaOhm() Problema {
	return Problema{
		Enunciado: "Unha resistencia de {R} $\\Omega$ e unha intensidade de {I} A.",
		Solucion:  "$V = {V}$ V e $P = {P}$ W",
		Variables: []VariableProblema{{Nome: "R", Modo: VariableAleatoria, De: "20", Ata: "200", Paso: "20"}, {Nome: "I", Modo: VariableAleatoria, De: "0,1", Ata: "0.5", Paso: "0.1"}},
		Calculos:  []CalculoProblema{{Nome: "V", Expr: "I*R"}, {Nome: "P", Expr: "V*I"}},
	}
}

func comprobarRangoOhm(t *testing.T, m *Motor) {
	t.Helper()
	validos := map[string]bool{"0,1": true, "0,2": true, "0,3": true, "0,4": true, "0,5": true}
	vistos := map[string]bool{}
	for s := 1; s <= 60; s++ {
		c := m.Calcular(problemaOhm(), SementeProblema(s, 0))
		if c.Erro != "" {
			t.Fatal(c.Erro)
		}
		i := c.Valores["I"].Texto
		if !validos[i] {
			t.Fatalf("semente %d: I = %q", s, i)
		}
		vistos[i] = true
		todo := c.Enunciado + " " + c.Solucion
		if strings.Contains(todo, "0000") || strings.Contains(todo, "9999") {
			t.Fatalf("semente %d: erro de coma flotante no texto: %q", s, todo)
		}
		if strings.Contains(c.Enunciado, "0.") || !strings.Contains(c.Enunciado, "0{,}") {
			t.Fatalf("semente %d: sen coma decimal: %q", s, c.Enunciado)
		}
	}
	if !vistos["0,5"] {
		t.Fatalf("o extremo 0,5 non sae nunca: %v", vistos)
	}
}

func TestRangoDecimalSenErroFlotante(t *testing.T) {
	m, _ := NovoMotor(Opcions{}, false, 0)
	comprobarRangoOhm(t, m)
}

func TestRangoDecimalSenErroFlotanteMaxima(t *testing.T) {
	precisa(t, "maxima")
	m, err := NovoMotor(Opcions{}, true, 20*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if !m.TenMaxima() {
		t.Skip("sen Maxima")
	}
	// Con Maxima, V = I·R calcúlao Maxima (fórmula con Maxima).
	comprobarRangoOhm(t, m)
}

func TestDecimaisSenMaxima(t *testing.T) {
	m, _ := NovoMotor(Opcions{Decimais: 2}, false, 0)
	p := Problema{Enunciado: "{v} e {10/3}", Variables: []VariableProblema{{Nome: "d", Modo: VariableFixa, Valor: "10"}, {Nome: "t", Modo: VariableFixa, Valor: "3"}},
		Calculos: []CalculoProblema{{Nome: "v", Expr: "d/t"}}}
	c := m.Calcular(p, 1)
	if c.Valores["v"].Texto != "3,33" || c.Enunciado != "$3{,}33$ e $3{,}33$" {
		t.Fatalf("%q %+v", c.Enunciado, c.Valores)
	}
}

func TestDecimaisConMaxima(t *testing.T) {
	precisa(t, "maxima")
	m, err := NovoMotor(Opcions{Decimais: 2}, true, 20*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if !m.TenMaxima() {
		t.Skip("sen Maxima")
	}
	p := Problema{Enunciado: "{v}", Variables: []VariableProblema{{Nome: "d", Modo: VariableFixa, Valor: "10.0"}, {Nome: "t", Modo: VariableFixa, Valor: "3"}},
		Calculos: []CalculoProblema{{Nome: "v", Expr: "d/t"}}}
	c := m.Calcular(p, 1)
	if c.Enunciado != "$3{,}33$" {
		t.Fatalf("%q %+v", c.Enunciado, c)
	}
}

func TestNumerosTex(t *testing.T) {
	for in, want := range map[string]string{
		"54.00000000000001":           "54",
		"16.200000000000003":          "16{,}2",
		"80.0":                        "80",
		"3.56":                        "3{,}56",
		"V=0.30000000000000004":       "V=0{,}3",
		"\\frac{6}{25}":               "\\frac{6}{25}",
		"1.5\\cdot 10^{-7}":           "1{,}5\\cdot 10^{-7}",
		"0.3333333333333333":          "0{,}333333333333",
		"\\left[ x=-2 , x=3 \\right]": "\\left[ x=-2 , x=3 \\right]",
	} {
		if got := numerosTex(in); got != want {
			t.Errorf("numerosTex(%q) = %q, quería %q", in, got, want)
		}
	}
}

func TestLiñaMaximaRangoDecimal(t *testing.T) {
	c := ficha.Campo{Nome: "I", Orixe: ficha.OrixeAleatorio, Xerador: ficha.XeradorRango, De: "0.1", Ata: "0.5", Paso: "0.1"}
	if got, want := liñaMaxima(c, nil), "I: float((1 + 1*random(5))/10)$"; got != want {
		t.Fatalf("%q, quería %q", got, want)
	}
	c = ficha.Campo{Nome: "R", Orixe: ficha.OrixeAleatorio, Xerador: ficha.XeradorRango, De: "20", Ata: "200", Paso: "20"}
	if got, want := liñaMaxima(c, nil), "R: 20 + 20*random(10)$"; got != want {
		t.Fatalf("%q, quería %q", got, want)
	}
}
