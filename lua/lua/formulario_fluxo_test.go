package main

import (
	"bytes"
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestFormula(t *testing.T) {
	vars := map[string]float64{"a": 6, "b": 4, "c": 0}
	casos := map[string]float64{
		"1+2*3":                      7,
		"(1+2)*3":                    9,
		"a*0.4 + b*0.6":              4.8,
		"2^3^2":                      512, // asociativa pola dereita
		"-a + 10":                    4,
		"10 % 4":                     2,
		"min(a,b)":                   4,
		"max(a; b; 9)":               9,
		"media(a,b)":                 5,
		"suma(1,2,3)":                6,
		"redondear(3.14159, 2)":      3.14,
		"round(2.5)":                 3,
		"abs(-7)":                    7,
		"sqrt(16)":                   4,
		"se(a>b, 1, 0)":              1,
		"se(a<=b, 1, 0)":             0,
		"A + B":                      10, // sen distinguir maiúsculas
		"se(c==0, 100, a/c)":         100,
		"a>b":                        1,
		"a != 6":                     0,
		"se(c!=0, a/c, -1) + 1":      0,
		"se(a>b, se(c==0,7,a/c), 0)": 7,
	}
	for f, esperado := range casos {
		got, err := avaliarFormula(f, vars)
		if err != nil || math.Abs(got-esperado) > 1e-9 {
			t.Errorf("%q = %v, %v; quería %v", f, got, err, esperado)
		}
	}
	erros := map[string]string{
		"":                  "baleira",
		"a +":               "completar",
		"a / c":             "cero",
		"x + 1":             "non existe",
		"0,5*a":             "sobra",
		"foo(1)":            "non existe",
		"min()":             "argumentos",
		"(a + b":            "parénteses",
		"sqrt(-1)":          "negativo",
		"a $ b":             "sobra",
		"se(c==0, a/c, 1)":  "cero", // a rama elixida si falla
		"se(a/c > 1, 1, 2)": "cero", // e a condición tamén
		"1.2.3":             "punto decimal",
		"se(1,2)":           "argumentos",
	}
	for f, quere := range erros {
		if _, err := avaliarFormula(f, vars); err == nil || !strings.Contains(err.Error(), quere) {
			t.Errorf("%q debería fallar con «%s», deu %v", f, quere, err)
		}
	}
	// A comprobación estática tolera a división por cero (con datos non pasará).
	if err := comprobarFormula("1/(a-b)", []string{"a", "b"}); err != nil {
		t.Errorf("1/(a-b) é unha fórmula válida: %v", err)
	}
	if err := comprobarFormula("a+z", []string{"a"}); err == nil {
		t.Error("z non existe")
	}
}

func txProba() textosDocumento { return novosTextosDoc(nil, map[string]string{}) }

// O cableado automático coma grafo ten que dar EXACTAMENTE os mesmos bloques
// ca o camiño sen tubería: senón «editar as tuberías» cambiaría o documento.
func TestFluxoAutomaticoEquivaleAoCableado(t *testing.T) {
	f := normalizarFormulario(formularioMostra())
	f.Campos = append(f.Campos, CampoFormulario{ID: "oculto", Tipo: campoTexto, Etiqueta: "Oculto", Agochado: true},
		CampoFormulario{ID: "final", Tipo: campoTexto, Etiqueta: "Ao final"})
	datos := map[string]DatosFormulario{
		"cheo":    datosMostra(t),
		"baleiro": {Valores: map[string]any{}},
		"parcial": {Valores: map[string]any{"grupo": "1.º A", "observacions": "Só un texto", "oculto": "non", "final": "si"}},
	}
	for nome, d := range datos {
		esperado, avEsp := bloquesFormulario(f, d, txProba())
		g := f
		g.Fluxo = fluxoAutomatico(f)
		if err := validarFluxo(g); err != nil {
			t.Fatalf("o fluxo automático non valida: %v", err)
		}
		obtido, avObt := bloquesFormulario(g, d, txProba())
		if !reflect.DeepEqual(esperado, obtido) {
			t.Errorf("%s: os bloques difiren.\nautomático: %+v\ntubería:    %+v", nome, esperado, obtido)
		}
		if !reflect.DeepEqual(avEsp, avObt) {
			t.Errorf("%s: os avisos difiren: %v / %v", nome, avEsp, avObt)
		}
	}
}

// formularioNotas: unha táboa de notas e os cálculos típicos dun docente.
func formularioNotas() Formulario {
	return normalizarFormulario(Formulario{
		Nome: "Notas",
		Campos: []CampoFormulario{
			{Tipo: campoTexto, Etiqueta: "Materia"},
			{Tipo: campoNumero, Etiqueta: "Peso do exame"},
			{Tipo: campoTaboa, Etiqueta: "Alumnado", Columnas: []CampoFormulario{
				{Tipo: campoTexto, Etiqueta: "Nome"},
				{Tipo: campoNumero, Etiqueta: "Nota"},
				{Tipo: campoVerificacion, Etiqueta: "Repite"},
			}},
		},
	})
}

func datosNotas() DatosFormulario {
	return DatosFormulario{Valores: map[string]any{
		"materia":       "Matemáticas",
		"peso_do_exame": "0,6",
		"alumnado": []any{
			map[string]any{"nome": "Uxía", "nota": "8", "repite": false},
			map[string]any{"nome": "Brais", "nota": "4,5", "repite": true},
			map[string]any{"nome": "Anxo", "nota": "6,5", "repite": false},
			map[string]any{"nome": "Carme", "nota": "3", "repite": true},
		},
	}}
}

func fluxoNotas() *Fluxo {
	return &Fluxo{
		Nodos: []NodoFluxo{
			{ID: "c_materia", Tipo: nodoCampo, Campo: "materia"},
			{ID: "c_peso", Tipo: nodoCampo, Campo: "peso_do_exame"},
			{ID: "c_alu", Tipo: nodoCampo, Campo: "alumnado"},
			{ID: "media", Tipo: nodoAgregar, Entradas: map[string]string{"taboa": "c_alu"}, Config: map[string]any{"operacion": "media", "columna": "nota", "decimais": 2}},
			{ID: "corte", Tipo: nodoConstante, Config: map[string]any{"tipo": "numero", "valor": "5"}},
			{ID: "pond", Tipo: nodoCalcular, Entradas: map[string]string{"a": "media", "b": "c_peso"}, Config: map[string]any{"formula": "a*b", "decimais": 2}},
			{ID: "aprobado", Tipo: nodoComparar, Entradas: map[string]string{"a": "media", "b": "corte"}, Config: map[string]any{"operador": ">="}},
			{ID: "repiten", Tipo: nodoFiltrar, Entradas: map[string]string{"taboa": "c_alu"}, Config: map[string]any{"columna": "repite", "operador": "=", "valor": "si"}},
			{ID: "orde", Tipo: nodoOrdenar, Entradas: map[string]string{"taboa": "c_alu"}, Config: map[string]any{"columna": "nota", "sentido": "desc"}},
			{ID: "resumo", Tipo: nodoUnir, Entradas: map[string]string{"a": "c_materia", "b": "media", "c": "pond"},
				Config: map[string]any{"modelo": "{{a}}: media {{b}} (ponderada {{c}})"}},
			{ID: "d_titulo", Tipo: nodoTitulo, Config: map[string]any{"texto": "Resultados", "nivel": "2"}},
			{ID: "d_texto", Tipo: nodoParagrafo, Entradas: map[string]string{"texto": "resumo"}},
			{ID: "d_lista", Tipo: nodoLista, Entradas: map[string]string{"taboa": "repiten"}, Config: map[string]any{"columna": "nome", "titulo": "Repetidores"}},
			{ID: "d_taboa", Tipo: nodoTaboa, Entradas: map[string]string{"taboa": "orde"}, Config: map[string]any{"titulo": "Por nota"}},
			{ID: "d_salto", Tipo: nodoSalto},
			{ID: "d_aviso", Tipo: nodoParagrafo, Entradas: map[string]string{"texto": "aviso", "se": "aprobado"}},
			{ID: "aviso", Tipo: nodoConstante, Config: map[string]any{"tipo": "texto", "valor": "A media chega ao aprobado."}},
		},
		Documento: []string{"d_titulo", "d_texto", "d_lista", "d_taboa", "d_salto", "d_aviso"},
	}
}

func TestFluxoCalculosFiltrosEOrdenacion(t *testing.T) {
	f := formularioNotas()
	f.Fluxo = fluxoNotas()
	if err := validarFormulario(f); err != nil {
		t.Fatal(err)
	}
	bloques, res := avaliarFluxoDoc(f, datosNotas(), txProba())
	if res.Erro != "" || len(res.Avisos) != 0 {
		t.Fatalf("erro=%q avisos=%v", res.Erro, res.Avisos)
	}
	// media = (8+4,5+6,5+3)/4 = 5,5 ; ponderada = 5,5*0,6 = 3,3
	if v := res.Nodos["media"].Resumo; v != "5,5" {
		t.Errorf("media = %q", v)
	}
	if v := res.Nodos["pond"].Resumo; v != "3,3" {
		t.Errorf("ponderada = %q", v)
	}
	if v := res.Nodos["aprobado"].Resumo; v != "Si" {
		t.Errorf("aprobado = %q", v)
	}
	if v := res.Nodos["repiten"].Resumo; v != "2 filas" {
		t.Errorf("repiten = %q", v)
	}
	var tipos []string
	for _, b := range bloques {
		tipos = append(tipos, b.Tipo)
	}
	// título, texto, [título+lista], [título+táboa], salto, texto (aprobado é si)
	if got := strings.Join(tipos, ","); got != "titulo,markdown,titulo,lista,titulo,taboa,salto,markdown" {
		t.Fatalf("bloques = %s", got)
	}
	if bloques[1].Texto != "Matemáticas: media 5,5 (ponderada 3,3)" {
		t.Errorf("texto = %q", bloques[1].Texto)
	}
	if !reflect.DeepEqual(bloques[3].Elementos, []string{"Brais", "Carme"}) {
		t.Errorf("lista = %v", bloques[3].Elementos)
	}
	// Ordenada por nota descendente: Uxía 8, Anxo 6,5, Brais 4,5, Carme 3.
	var nomes []string
	for _, fila := range bloques[5].Filas {
		nomes = append(nomes, fila[0])
	}
	if !reflect.DeepEqual(nomes, []string{"Uxía", "Anxo", "Brais", "Carme"}) {
		t.Errorf("orde = %v", nomes)
	}

	// Con notas que non chegan ao aprobado, o bloque condicionado non sae.
	d := datosNotas()
	d.Valores["alumnado"] = []any{map[string]any{"nome": "Ana", "nota": "3", "repite": false}}
	bloques, res = avaliarFluxoDoc(f, d, txProba())
	if res.Nodos["aprobado"].Resumo != "Non" || bloques[len(bloques)-1].Tipo != bloqueSalto {
		t.Errorf("o aviso «aprobado» non debía saír: %+v / %s", bloques, res.Nodos["aprobado"].Resumo)
	}
}

func TestFluxoDatosBaleirosNonRompen(t *testing.T) {
	f := formularioNotas()
	f.Fluxo = fluxoNotas()
	bloques, res := avaliarFluxoDoc(f, DatosFormulario{Valores: map[string]any{}}, txProba())
	if res.Erro != "" || len(res.Avisos) != 0 {
		t.Fatalf("erro=%q avisos=%v", res.Erro, res.Avisos)
	}
	// Só o título fixo e o salto: o resto está baleiro e non sae.
	var tipos []string
	for _, b := range bloques {
		tipos = append(tipos, b.Tipo)
	}
	if got := strings.Join(tipos, ","); got != "titulo,salto" {
		t.Errorf("bloques = %s", got)
	}
	if res.Nodos["media"].Vacio != true || res.Nodos["pond"].Vacio != true {
		t.Errorf("sen datos, os cálculos son baleiros: %+v", res.Nodos["media"])
	}
}

func TestFluxoErrosNosNodos(t *testing.T) {
	f := formularioNotas()
	f.Fluxo = fluxoNotas()
	// Fórmula rota e columna que non existe: o nodo queda en erro e o seu bloque non sae, con aviso.
	for i := range f.Fluxo.Nodos {
		switch f.Fluxo.Nodos[i].ID {
		case "pond":
			f.Fluxo.Nodos[i].Config["formula"] = "a*"
		case "repiten":
			f.Fluxo.Nodos[i].Config["columna"] = "non_existe"
		}
	}
	if err := validarFormulario(f); err != nil {
		t.Fatalf("un erro de configuración non debe impedir gardar: %v", err)
	}
	bloques, res := avaliarFluxoDoc(f, datosNotas(), txProba())
	if !strings.Contains(res.Nodos["pond"].Erro, "fórmula") || !strings.Contains(res.Nodos["repiten"].Erro, "columna") {
		t.Errorf("erros dos nodos: %+v", res.Nodos)
	}
	joined := strings.Join(res.Avisos, "|")
	if !strings.Contains(joined, "Repetidores") {
		t.Errorf("avisos: %v", res.Avisos)
	}
	for _, b := range bloques {
		if b.Tipo == bloqueLista {
			t.Error("a lista con erro non debe saír")
		}
		if b.Tipo == bloqueMarkdown && strings.Contains(b.Texto, "ponderada") {
			t.Error("o texto que depende dunha fórmula rota non debe saír")
		}
	}
	// A división por cero tamén é un erro do nodo, non un pánico.
	f.Fluxo.Nodos[5].Config["formula"] = "a/(b-b)"
	_, res = avaliarFluxoDoc(f, datosNotas(), txProba())
	if !strings.Contains(res.Nodos["pond"].Erro, "cero") {
		t.Errorf("división por cero: %q", res.Nodos["pond"].Erro)
	}
}

func TestValidarFluxoEstrutura(t *testing.T) {
	base := func() Formulario {
		f := formularioNotas()
		f.Fluxo = fluxoNotas()
		return f
	}
	casos := map[string]func(*Formulario){
		"tipo descoñecido":   func(f *Formulario) { f.Fluxo.Nodos[3].Tipo = "magico" },
		"id repetido":        func(f *Formulario) { f.Fluxo.Nodos[1].ID = "c_materia" },
		"id malo":            func(f *Formulario) { f.Fluxo.Nodos[1].ID = "Malo Id" },
		"orixe inexistente":  func(f *Formulario) { f.Fluxo.Nodos[3].Entradas["taboa"] = "nada" },
		"porto inexistente":  func(f *Formulario) { f.Fluxo.Nodos[3].Entradas["saco"] = "c_alu" },
		"desde un destino":   func(f *Formulario) { f.Fluxo.Nodos[3].Entradas["taboa"] = "d_titulo" },
		"documento con nodo": func(f *Formulario) { f.Fluxo.Documento = append(f.Fluxo.Documento, "media") },
		"documento inexiste": func(f *Formulario) { f.Fluxo.Documento = append(f.Fluxo.Documento, "nada") },
		"documento repetido": func(f *Formulario) { f.Fluxo.Documento = append(f.Fluxo.Documento, "d_titulo") },
		"campo sen campo":    func(f *Formulario) { f.Fluxo.Nodos[0].Campo = "" },
		"ciclo":              func(f *Formulario) { f.Fluxo.Nodos[5].Entradas["b"] = "pond" },
		"entrada baleira":    func(f *Formulario) { f.Fluxo.Nodos[3].Entradas["taboa"] = "" },
	}
	for nome, muda := range casos {
		f := base()
		muda(&f)
		// A entrada baleira normalízase (desaparece): valida despois de normalizar só nos demais.
		if nome != "entrada baleira" {
			f = normalizarFormulario(f)
		}
		err := validarFluxo(f)
		if nome == "entrada baleira" {
			if err == nil {
				t.Errorf("%s: sen normalizar debería fallar", nome)
			}
			continue
		}
		if err == nil {
			t.Errorf("%s: debería fallar", nome)
		}
	}
	if err := validarFormulario(base()); err != nil {
		t.Errorf("o fluxo bo debería validar: %v", err)
	}
}

func TestFluxoTiposDasLigazons(t *testing.T) {
	f := formularioNotas()
	f.Fluxo = fluxoNotas()
	// Un texto non entra nunha fórmula, nin unha táboa nun párrafo.
	f.Fluxo.Nodos[5].Entradas["a"] = "c_materia"
	f.Fluxo.Nodos[11].Entradas["texto"] = "c_alu"
	_, res := avaliarFluxoDoc(f, datosNotas(), txProba())
	if !strings.Contains(res.Nodos["pond"].Erro, "número") {
		t.Errorf("texto na fórmula: %q", res.Nodos["pond"].Erro)
	}
	joined := strings.Join(res.Avisos, "|")
	if !strings.Contains(joined, "táboa") {
		t.Errorf("táboa nun párrafo: %v", res.Avisos)
	}
}

func TestSincronizarFluxo(t *testing.T) {
	f := formularioNotas()
	f.Fluxo = fluxoNotas()
	// Engádese un campo e quítase o peso.
	f.Campos = append(f.Campos, CampoFormulario{ID: "notas", Tipo: campoTextoLongo, Etiqueta: "Notas"})
	var campos []CampoFormulario
	for _, c := range f.Campos {
		if c.ID != "peso_do_exame" {
			campos = append(campos, c)
		}
	}
	f.Campos = campos
	novo := sincronizarFluxo(f)
	tem := func(id string) bool {
		for _, n := range novo.Nodos {
			if n.ID == id {
				return true
			}
		}
		return false
	}
	if tem("c_peso") {
		t.Error("o campo quitado debe saír do fluxo")
	}
	if !tem("c_notas") {
		t.Error("o campo novo debe entrar no fluxo")
	}
	for _, n := range novo.Nodos {
		if n.ID == "pond" {
			if _, hai := n.Entradas["b"]; hai {
				t.Error("a ligazón que viña do campo quitado debe desaparecer")
			}
		}
	}
	// O destino novo sae ao final do documento, e o que había segue na súa orde.
	if novo.Documento[0] != "d_titulo" || len(novo.Documento) != len(fluxoNotas().Documento)+1 {
		t.Errorf("documento = %v", novo.Documento)
	}
	f.Fluxo = novo
	if err := validarFormulario(normalizarFormulario(f)); err != nil {
		t.Errorf("o fluxo sincronizado debería validar: %v", err)
	}
	// Sen fluxo, devolve o automático.
	g := formularioNotas()
	if a := sincronizarFluxo(g); len(a.Documento) == 0 {
		t.Error("sen fluxo debería saír o automático")
	}
}

func TestFluxoGardaseEVaiEnLatexEMarkdown(t *testing.T) {
	dir := t.TempDir()
	orixinal := dirFormularios
	dirFormularios = func() (string, error) { return dir, nil }
	defer func() { dirFormularios = orixinal }()

	f := formularioNotas()
	f.Fluxo = fluxoNotas()
	g, err := gardarFormulario(f)
	if err != nil {
		t.Fatal(err)
	}
	l, err := lerFormulario(g.ID)
	if err != nil || l.Fluxo == nil || len(l.Fluxo.Nodos) != len(f.Fluxo.Nodos) || len(l.Fluxo.Documento) != 6 {
		t.Fatalf("o fluxo non se gardou ben: %+v, %v", l.Fluxo, err)
	}

	p := PedidoFormulario{Formulario: l, Datos: datosNotas()}
	md, avisos, err := markdownFormulario(p, "md", t.TempDir(), agoraFormulario, nil)
	if err != nil || len(avisos) != 0 {
		t.Fatalf("err=%v avisos=%v", err, avisos)
	}
	for _, quere := range []string{"## Resultados", "Matemáticas: media 5,5 (ponderada 3,3)", "### Repetidores", "- Brais", "- Carme", "| Uxía | 8 | Non |", "---"} {
		if !strings.Contains(md, quere) {
			t.Errorf("falta %q en:\n%s", quere, md)
		}
	}
	tex, _, err := latexFormulario(p, "pdflatex", nil, t.TempDir(), agoraFormulario, nil, "")
	if err != nil || !strings.Contains(tex, `\begin{itemize}`) || !strings.Contains(tex, `\item Brais`) || !strings.Contains(tex, `\clearpage`) {
		t.Errorf("LaTeX sen lista ou salto (err=%v)", err)
	}
	if !commandExists("xelatex") && !commandExists("pdflatex") {
		return
	}
	res, err := compilarFormulario(p, "xelatex", nil, t.TempDir(), nil, "ficha")
	if err != nil || !bytes.HasPrefix(res.PDF, []byte("%PDF")) {
		t.Fatalf("non compila: %v\n%s", err, res.Log)
	}
}

func TestFluxoInvalidoNonRompeOXerar(t *testing.T) {
	f := formularioNotas()
	f.Fluxo = &Fluxo{Nodos: []NodoFluxo{{ID: "a", Tipo: "raro"}}}
	// Chegado do disco (sen pasar por validarFormulario): o documento sae cun aviso.
	p := PedidoFormulario{Formulario: f, Datos: datosNotas()}
	_, avisos, err := markdownFormulario(p, "md", t.TempDir(), agoraFormulario, nil)
	if err == nil {
		t.Errorf("normalizar e validar deberían rexeitar o formulario (avisos %v)", avisos)
	}
	bloques, av := bloquesFormulario(f, datosNotas(), txProba())
	if len(bloques) != 0 || !strings.Contains(strings.Join(av, "|"), "tubería non é válida") {
		t.Errorf("bloques=%+v avisos=%v", bloques, av)
	}
}

func TestCatalogoNodosCoherente(t *testing.T) {
	vistos := map[string]bool{}
	for _, n := range catalogoNodos {
		if vistos[n.Tipo] {
			t.Errorf("tipo repetido %s", n.Tipo)
		}
		vistos[n.Tipo] = true
		if n.Categoria != "orixe" && n.Categoria != "transformacion" && n.Categoria != "destino" {
			t.Errorf("%s: categoría %q", n.Tipo, n.Categoria)
		}
		if n.Nome == "" || n.Axuda == "" {
			t.Errorf("%s sen nome ou axuda", n.Tipo)
		}
		if n.Categoria == "transformacion" && n.Saida == "" {
			t.Errorf("%s: unha transformación ten saída", n.Tipo)
		}
		if n.eDestino() && n.Saida != "" {
			t.Errorf("%s: un destino non ten saída", n.Tipo)
		}
	}
	// Todo destino que produce un fluxo automático está no catálogo.
	for _, n := range fluxoAutomatico(normalizarFormulario(formularioMostra())).Nodos {
		if !vistos[n.Tipo] {
			t.Errorf("o fluxo automático usa %s, que non está no catálogo", n.Tipo)
		}
	}
}
