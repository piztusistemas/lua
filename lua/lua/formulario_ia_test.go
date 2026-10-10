package main

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"
)

func nomesFerramentas(l []IAFerramenta) map[string]bool {
	m := map[string]bool{}
	for _, f := range l {
		m[f.Nome] = true
	}
	return m
}

func TestAmbitoFormularioFerramentas(t *testing.T) {
	a := &App{}
	dentro := nomesFerramentas(a.IAFerramentasDoAmbito(ambitoFormulario))
	for _, quere := range []string{"formulario_ler", "formulario_cambiar", "formulario_axuda", "formulario_probar", "adxunto_ler", "adxunto_ver", "adxunto_extraer_imaxes", "usuario_preguntar", "web_buscar"} {
		if !dentro[quere] {
			t.Errorf("o ámbito formulario debería ter %s", quere)
		}
	}
	// Non ve o curso, as plantillas, as Opcións nin o que garda ficheiros; nin a validación interna.
	for _, non := range []string{"curso_ler", "curso_configurar", "plantilla_gardar", "opcions_cambiar", "adxunto_gardar_en_plantilla", "actas_ler", "formulario_validar"} {
		if dentro[non] {
			t.Errorf("o ámbito formulario NON debería ter %s", non)
		}
	}
	// E o asistente principal non ve as do deseñador (traballan sobre o seu borrador).
	for n := range nomesFerramentas(a.IAFerramentas()) {
		if strings.HasPrefix(n, "formulario_") {
			t.Errorf("o asistente principal non debe ter %s", n)
		}
	}
	// Cada ferramenta ten un esquema JSON válido e o «onde» correcto.
	for _, f := range a.IAFerramentasDoAmbito(ambitoFormulario) {
		if f.Parametros != "" && !json.Valid([]byte(f.Parametros)) {
			t.Errorf("%s: parametros non é JSON", f.Nome)
		}
	}
	if !strings.Contains(sistemaAsistenteFormulario("- estado x", "gl"), "DESEÑADOR DE FORMULARIOS") {
		t.Error("o prompt do ámbito formulario non sae")
	}
}

// campos do exemplo da axuda: se o exemplo non valida ou non calcula ben, a IA copiaría un erro.
func formularioDoExemploAxuda() Formulario {
	var fl Fluxo
	if err := json.Unmarshal([]byte(exemploFluxoAxuda), &fl); err != nil {
		panic(err)
	}
	return Formulario{Nome: "Exemplo", Campos: []CampoFormulario{
		{ID: "alumnado", Tipo: campoTaboa, Etiqueta: "Alumnado", Columnas: []CampoFormulario{
			{ID: "nome", Tipo: campoTexto, Etiqueta: "Nome"},
			{ID: "exame", Tipo: campoNumero, Etiqueta: "Exame"},
			{ID: "traballos", Tipo: campoNumero, Etiqueta: "Traballos"},
		}},
		{ID: "peso_exame", Tipo: campoNumero, Etiqueta: "Peso do exame"},
	}, Fluxo: &fl}
}

func TestExemploDaAxudaValidaECalcula(t *testing.T) {
	f := normalizarFormulario(formularioDoExemploAxuda())
	if err := validarFormulario(f); err != nil {
		t.Fatalf("o exemplo da axuda non valida: %v", err)
	}
	if p := problemasDoFormulario(f); len(p) != 0 {
		t.Fatalf("o exemplo da axuda ten problemas: %v", p)
	}
	d := DatosFormulario{Valores: map[string]any{
		"peso_exame": "0,6",
		"alumnado": []any{
			map[string]any{"nome": "A", "exame": "4", "traballos": "9"},
			map[string]any{"nome": "B", "exame": "8", "traballos": "7"},
		},
	}}
	_, res := avaliarFluxoDoc(f, d, txProba())
	if res.Erro != "" || len(res.Avisos) != 0 {
		t.Fatalf("erro=%q avisos=%v", res.Erro, res.Avisos)
	}
	// media exame 6, traballos 8; final = 6*0.6 + 8*0.4 = 6.8 -> aproba.
	if v := res.Nodos["final"].Resumo; v != "6,8" {
		t.Errorf("final = %q", v)
	}
	var textos []string
	for _, b := range res.Bloques {
		textos = append(textos, b.Texto)
	}
	joined := strings.Join(textos, "|")
	for _, quere := range []string{"Resultados", "Nota final do grupo: 6,8", "Alumnado con exame suspenso", "O grupo aproba."} {
		if !strings.Contains(joined, quere) {
			t.Errorf("falta %q en %q", quere, joined)
		}
	}
	// A axuda leva o catálogo enteiro e o método.
	ax := axudaFormulario()
	for _, quere := range []string{`"tipo":"calcular"`, `"tipo":"filtrar"`, "MÉTODO PARA DESEÑAR", "usuario_preguntar", "seccion:"} {
		if !strings.Contains(ax, quere) {
			t.Errorf("a axuda non di %q", quere)
		}
	}
}

func TestValidarFormularioIA(t *testing.T) {
	bo := map[string]any{"formulario": map[string]any{
		"nome": "Ficha",
		"campos": []any{
			map[string]any{"tipo": "texto", "etiqueta": "Nome"}, // sen id: Lua póno
			map[string]any{"id": "nota", "tipo": "numero", "etiqueta": "Nota"},
		},
	}}
	r, err := validarFormularioIA(bo)
	if err != nil {
		t.Fatal(err)
	}
	m := r.(map[string]any)
	f := m["formulario"].(Formulario)
	if m["ok"] != true || f.ID != "ficha" || f.Campos[0].ID != "nome" || m["automatico"] != true {
		t.Errorf("resultado: %+v", m)
	}

	// Un erro non é un erro de Go: vai no resultado, para que a IA o vexa e corrixa.
	malo := map[string]any{"formulario": map[string]any{"nome": "X", "campos": []any{map[string]any{"id": "a", "tipo": "magico", "etiqueta": "A"}}}}
	r, err = validarFormularioIA(malo)
	m = r.(map[string]any)
	if err != nil || m["ok"] != false || !strings.Contains(m["erro"].(string), "magico") {
		t.Errorf("erro esperado: %+v, %v", m, err)
	}
	if _, err := validarFormularioIA(map[string]any{}); err == nil {
		t.Error("sen formulario debería dar erro")
	}

	// Con tubería rota nun nodo, valida pero informa dos problemas.
	f2 := formularioDoExemploAxuda()
	for i := range f2.Fluxo.Nodos {
		if f2.Fluxo.Nodos[i].ID == "final" {
			f2.Fluxo.Nodos[i].Config["formula"] = "a*"
		}
	}
	raw, _ := json.Marshal(f2)
	var obx map[string]any
	_ = json.Unmarshal(raw, &obx)
	r, _ = validarFormularioIA(map[string]any{"formulario": obx})
	m = r.(map[string]any)
	if m["ok"] != true || m["problemas"] == nil {
		t.Errorf("debería validar cun problema de nodo: %+v", m)
	}

	// sincronizar: un campo novo entra na tubería propia.
	f3 := formularioDoExemploAxuda()
	f3.Campos = append(f3.Campos, CampoFormulario{ID: "obs", Tipo: campoTextoLongo, Etiqueta: "Observacións"})
	raw, _ = json.Marshal(f3)
	_ = json.Unmarshal(raw, &obx)
	r, _ = validarFormularioIA(map[string]any{"formulario": obx, "sincronizar": true})
	nf := r.(map[string]any)["formulario"].(Formulario)
	tem := false
	for _, n := range nf.Fluxo.Nodos {
		if n.Tipo == nodoCampo && n.Campo == "obs" {
			tem = true
		}
	}
	if !tem {
		t.Error("sincronizar debería engadir o nodo do campo novo")
	}
}

func TestProbarFormularioIABloques(t *testing.T) {
	a := &App{}
	raw, _ := json.Marshal(formularioDoExemploAxuda())
	var f map[string]any
	_ = json.Unmarshal(raw, &f)
	r, err := a.probarFormularioIA(map[string]any{
		"formulario": f, "formato": "bloques",
		"datos": map[string]any{"valores": map[string]any{
			"peso_exame": 0.5,
			"alumnado":   []any{map[string]any{"nome": "A", "exame": 8, "traballos": 6}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	m := r.(map[string]any)
	nodos := m["nodos"].(map[string]NodoAvaliado)
	if nodos["final"].Resumo != "7" {
		t.Errorf("final = %q (8*0.5 + 6*0.5 = 7)", nodos["final"].Resumo)
	}
	if _, hai := m["erro_pdf"]; hai {
		t.Error("en formato bloques non se compila o PDF")
	}
	// Datos que non cadran non rompen: avisos.
	r, err = a.probarFormularioIA(map[string]any{"formulario": f, "formato": "bloques", "datos": "isto non é un obxecto"})
	if err == nil {
		t.Errorf("datos mal formados deberían dar erro, deu %v", r)
	}
	// Un formulario inválido dá un erro que a IA le.
	if _, err := a.probarFormularioIA(map[string]any{"formulario": map[string]any{"nome": ""}}); err == nil || !strings.Contains(err.Error(), "non é válido") {
		t.Errorf("erro esperado: %v", err)
	}
}

func TestProbarFormularioIAPDF(t *testing.T) {
	if !commandExists("xelatex") && !commandExists("pdflatex") {
		t.Skip("sen LaTeX")
	}
	if !commandExists("pdftoppm") {
		t.Skip("sen pdftoppm")
	}
	t.Setenv("XDG_CACHE_HOME", t.TempDir()) // o almacén temporal dos adxuntos
	a := &App{}
	raw, _ := json.Marshal(formularioDoExemploAxuda())
	var f map[string]any
	_ = json.Unmarshal(raw, &f)
	r, err := a.probarFormularioIA(map[string]any{
		"formulario": f,
		"datos": map[string]any{"valores": map[string]any{
			"peso_exame": "0,6",
			"alumnado": []any{
				map[string]any{"nome": "Uxía", "exame": "4", "traballos": "9"},
				map[string]any{"nome": "Brais", "exame": "8", "traballos": "7"},
			},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	res, ok := r.(resultadoComImaxes)
	if !ok || len(res.Imaxes) != 1 {
		t.Fatalf("debería devolver unha imaxe da páxina: %#v", r)
	}
	if !strings.Contains(res.Texto, "Nota final do grupo") || !strings.Contains(res.Texto, "Páxina 1 de") {
		t.Errorf("texto: %s", res.Texto)
	}
}

// A lista de iconas que se ensina á IA (Go) ten que ser a que pinta o frontend:
// unha clave que só exista nun sitio daría un formulario con icona rota.
func TestIconasFormularioCoincidenCoFrontend(t *testing.T) {
	raw, err := os.ReadFile("frontend/src/formularios/iconas.js")
	if err != nil {
		t.Skip("sen o código do frontend")
	}
	s := string(raw)
	ini := strings.Index(s, "const ICONAS = {")
	fin := strings.Index(s[ini:], "\n}\n")
	if ini < 0 || fin < 0 {
		t.Fatal("non se atopou o obxecto ICONAS en iconas.js")
	}
	var js []string
	for _, l := range strings.Split(s[ini:ini+fin], "\n") {
		if m := regexp.MustCompile(`^  (\w+): '`).FindStringSubmatch(l); m != nil {
			js = append(js, m[1])
		}
	}
	if strings.Join(js, ",") != strings.Join(iconasFormulario, ",") {
		t.Errorf("iconas distintas\n  frontend: %v\n  Go:       %v", js, iconasFormulario)
	}
}

// O asistente dos datos dun formulario le e escribe os datos, e pode ler
// documentos adxuntos (para exames e prácticas); nada do deseñador, web,
// curso, plantillas ou Opcións.
func TestAmbitoFormularioDatosFerramentas(t *testing.T) {
	a := &App{}
	dentro := nomesFerramentas(a.IAFerramentasDoAmbito(ambitoFormularioDatos))
	for _, quere := range []string{
		"formulario_datos_ler", "formulario_datos_cambiar", "usuario_preguntar", "usuario_avisar",
		"adxuntos_listar", "adxunto_ler", "adxunto_ver", "adxunto_extraer_imaxes",
		// Compilar o documento e ler o rexistro interno (para arranxar o que non compila).
		"formulario_xerar", "rexistro_listar", "rexistro_ler",
	} {
		if !dentro[quere] {
			t.Errorf("o ámbito formulario_datos debería ter %s", quere)
		}
	}
	if len(dentro) != 11 {
		t.Errorf("o ámbito formulario_datos ten máis ferramentas do debido: %v", dentro)
	}
	for _, non := range []string{"formulario_ler", "formulario_cambiar", "formulario_probar", "formulario_axuda", "adxunto_eliminar", "adxunto_gardar_en_plantilla", "web_buscar", "web_consultar", "curso_ler", "informe_xerar", "plantilla_gardar", "opcions_cambiar", "ferramenta_activa"} {
		if dentro[non] {
			t.Errorf("o ámbito formulario_datos NON debería ter %s", non)
		}
	}
	// Nin o asistente principal nin o do deseñador as ven.
	for _, ambito := range []string{"", ambitoFormulario, ambitoPlantilla, ambitoTarefa} {
		for n := range nomesFerramentas(a.IAFerramentasDoAmbito(ambito)) {
			if strings.HasPrefix(n, "formulario_datos_") {
				t.Errorf("o ámbito %q non debe ter %s", ambito, n)
			}
		}
	}
	for _, f := range a.IAFerramentasDoAmbito(ambitoFormularioDatos) {
		if f.Parametros != "" && !json.Valid([]byte(f.Parametros)) {
			t.Errorf("%s: parametros non é JSON", f.Nome)
		}
	}
	p := sistemaAsistenteFormularioDatos("- estado x", "gl")
	if !strings.Contains(p, "DATOS dun formulario") || !strings.Contains(p, "- estado x") {
		t.Error("o prompt do ámbito formulario_datos non sae")
	}
}

// O mesmo coas cores: unha clave só en Go deixaría a icona coa cor por defecto.
func TestCoresFormularioCoincidenCoFrontend(t *testing.T) {
	raw, err := os.ReadFile("frontend/src/formularios/iconas.js")
	if err != nil {
		t.Skip("sen o código do frontend")
	}
	s := string(raw)
	ini := strings.Index(s, "const CORES = {")
	fin := strings.Index(s[ini:], "\n}\n")
	if ini < 0 || fin < 0 {
		t.Fatal("non se atopou o obxecto CORES en iconas.js")
	}
	var js []string
	for _, l := range strings.Split(s[ini:ini+fin], "\n") {
		if m := regexp.MustCompile(`^  (\w+): '`).FindStringSubmatch(l); m != nil {
			js = append(js, m[1])
		}
	}
	if strings.Join(js, ",") != strings.Join(coresFormulario, ",") {
		t.Errorf("cores distintas\n  frontend: %v\n  Go:       %v", js, coresFormulario)
	}
}

func TestNormalizarFormularioCor(t *testing.T) {
	if f := normalizarFormulario(Formulario{Nome: "X", Cor: " rosa "}); f.Cor != "rosa" {
		t.Errorf("cor válida: %q", f.Cor)
	}
	if f := normalizarFormulario(Formulario{Nome: "X", Cor: "vermello"}); f.Cor != "" {
		t.Errorf("unha cor descoñecida debería quedar baleira: %q", f.Cor)
	}
}
