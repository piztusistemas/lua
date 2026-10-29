package main

import (
	"strings"
	"testing"

	"edicion/xerar"
)

// iaDeProba substitúe a IA polas respostas dadas, en orde, e garda as
// mensaxes que recibiu.
func iaDeProba(t *testing.T, respostas ...string) *[][]IAMensaxe {
	t.Helper()
	orixinal := chamarIAProblema
	t.Cleanup(func() { chamarIAProblema = orixinal })
	var recibidas [][]IAMensaxe
	chamarIAProblema = func(_ Settings, _ string, msgs []IAMensaxe) (string, error) {
		recibidas = append(recibidas, append([]IAMensaxe(nil), msgs...))
		if len(recibidas) > len(respostas) {
			t.Fatalf("chamada á IA de máis (%d)", len(recibidas))
		}
		return respostas[len(recibidas)-1], nil
	}
	return &recibidas
}

func motorSenMaxima(t *testing.T) *xerar.Motor {
	t.Helper()
	m, err := xerar.NovoMotor(xerar.Opcions{}, false, 0)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestAsistentePreguntaPropostaComprobada(t *testing.T) {
	recibidas := iaDeProba(t, "```json\n"+`{"enunciado": "Un corpo de {m} kg acelera a {a} m/s². Calcula a forza.",
		"variables": [{"nome": "m", "modo": "variable", "de": "2", "ata": "10", "paso": "1"}, {"nome": "a", "modo": "fixa", "valor": "3"}],
		"explicacion": "Os datos pasan a variables."}`+"\n```")
	p := PedidoAsistentePregunta{
		Problema: ProblemaFormulario{Enunciado: "Un corpo de 4 kg acelera a 3 m/s². Calcula a forza."},
		Accion:   accionVariables, Contexto: "Exame de física",
	}
	r, err := asistentePreguntaCon(Settings{}, motorSenMaxima(t), p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r.Enunciado, "{m}") || len(r.Variables) != 2 || r.Explicacion == "" {
		t.Fatalf("proposta: %+v", r)
	}
	if strings.Contains(r.Calculado.Enunciado, "{m}") || len(r.Fallos) > 0 {
		t.Fatalf("a proposta non se calculou: %+v", r)
	}
	// Só viaxa a pregunta, coa acción e o contexto.
	msg := (*recibidas)[0][0].Texto
	for _, x := range []string{"Exame de física", "Un corpo de 4 kg", "variables"} {
		if !strings.Contains(msg, x) {
			t.Errorf("a mensaxe non leva %q:\n%s", x, msg)
		}
	}
}

func TestAsistentePreguntaDevolveOsFallosAIA(t *testing.T) {
	recibidas := iaDeProba(t,
		`{"enunciado": "Masa {m} kg e forza {F} N", "variables": [{"nome": "m", "modo": "fixa", "valor": "2"}]}`,
		`{"enunciado": "Masa {m} kg", "variables": [{"nome": "m", "modo": "fixa", "valor": "2"}], "explicacion": "Quitei {F}."}`)
	r, err := asistentePreguntaCon(Settings{}, motorSenMaxima(t), PedidoAsistentePregunta{Problema: ProblemaFormulario{Enunciado: "Masa 2 kg"}, Accion: accionMellorar})
	if err != nil {
		t.Fatal(err)
	}
	if len(*recibidas) != 2 {
		t.Fatalf("esperaba 2 rondas, houbo %d", len(*recibidas))
	}
	segunda := (*recibidas)[1]
	if ultima := segunda[len(segunda)-1].Texto; !strings.Contains(ultima, "{F}") {
		t.Errorf("a segunda ronda non leva o fallo: %q", ultima)
	}
	if r.Enunciado != "Masa {m} kg" || len(r.Fallos) > 0 {
		t.Fatalf("proposta: %+v", r)
	}
}

func TestAsistentePreguntaQuedaCosFallosTrasAsRondas(t *testing.T) {
	mala := `{"enunciado": "Masa {q}", "variables": []}`
	iaDeProba(t, mala, mala)
	r, err := asistentePreguntaCon(Settings{}, motorSenMaxima(t), PedidoAsistentePregunta{Problema: ProblemaFormulario{Enunciado: "Masa"}, Accion: accionMellorar})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Fallos) == 0 {
		t.Fatal("a proposta tiña que levar os fallos para que o editor avise")
	}
}

func TestAsistentePreguntaRexeitaPedidosEVariablesMalas(t *testing.T) {
	m := motorSenMaxima(t)
	for nome, p := range map[string]PedidoAsistentePregunta{
		"acción descoñecida":   {Problema: ProblemaFormulario{Enunciado: "x"}, Accion: "borrar"},
		"libre sen indicación": {Problema: ProblemaFormulario{Enunciado: "x"}, Accion: accionLibre},
		"mellorar baleira":     {Accion: accionMellorar},
	} {
		if _, err := asistentePreguntaCon(Settings{}, m, p); err == nil {
			t.Errorf("%s: tiña que dar erro", nome)
		}
	}
	// Nome de variable que non vale: a IA ten outra ronda e, se insiste, erro.
	malo := `{"enunciado": "Masa {1m}", "variables": [{"nome": "1m", "modo": "fixa", "valor": "2"}]}`
	iaDeProba(t, malo, malo)
	if _, err := asistentePreguntaCon(Settings{}, m, PedidoAsistentePregunta{Problema: ProblemaFormulario{Enunciado: "Masa 2"}, Accion: accionMellorar}); err == nil {
		t.Error("un nome de variable que non vale tiña que dar erro")
	}
}

func TestAsistentePreguntaParecidaPodeEmpezarBaleira(t *testing.T) {
	iaDeProba(t, `{"enunciado": "Calcula $\\sqrt{ {n} }$.", "variables": [{"nome": "n", "modo": "variable", "de": "1", "ata": "9"}]}`)
	r, err := asistentePreguntaCon(Settings{}, motorSenMaxima(t), PedidoAsistentePregunta{Accion: accionLibre, Instrucions: "unha raíz cadrada"})
	if err != nil {
		t.Fatal(err)
	}
	if r.Variables[0].Modo != xerar.VariableAleatoria || r.Variables[0].Paso != "1" {
		t.Fatalf("variable: %+v", r.Variables[0])
	}
}
