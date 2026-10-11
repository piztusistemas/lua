package main

import (
	"bytes"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var agoraFormulario = time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)

// formularioMostra usa todos os tipos de campo.
func formularioMostra() Formulario {
	return Formulario{
		Nome:       "Informe de titoría",
		Descricion: "Seguimento trimestral do grupo",
		Campos: []CampoFormulario{
			{Tipo: campoSeccion, Etiqueta: "Datos do grupo"},
			{Tipo: campoTexto, Etiqueta: "Grupo", Obrigatorio: true},
			{Tipo: campoNumero, Etiqueta: "Alumnado"},
			{Tipo: campoData, Etiqueta: "Data da reunión"},
			{Tipo: campoEscolla, Etiqueta: "Trimestre", Opcions: []string{"1.º", "2.º", "3.º"}},
			{Tipo: campoVerificacion, Etiqueta: "Reunión coas familias"},
			{Tipo: campoSeccion, Etiqueta: "Seguimento"},
			{Tipo: campoTextoLongo, Etiqueta: "Observacións"},
			{Tipo: campoTaboa, Etiqueta: "Alumnado con medidas", Columnas: []CampoFormulario{
				{Tipo: campoTexto, Etiqueta: "Nome"},
				{Tipo: campoNumero, Etiqueta: "Nota"},
				{Tipo: campoVerificacion, Etiqueta: "Adaptación"},
			}},
			{Tipo: campoImaxe, Etiqueta: "Foto do grupo"},
		},
	}
}

func datosMostra(t *testing.T) DatosFormulario {
	return DatosFormulario{
		Contexto: ContextoFormulario{Centro: "IES Castelao & Fillos, 100% público", Curso: "2025-2026", Departamento: "Orientación", Responsable: "Ana Pérez"},
		Valores: map[string]any{
			"grupo":                 "2.º B",
			"alumnado":              "27",
			"data_da_reunion":       "2026-01-15",
			"trimestre":             "2.º",
			"reunion_coas_familias": true,
			"observacions":          "Grupo **traballador**.\n\n- Mellora en lingua\n- Faltas en matemáticas",
			"alumnado_con_medidas": []any{
				map[string]any{"nome": "Brais _Q_", "nota": "4,5", "adaptacion": true},
				map[string]any{"nome": "Uxía", "nota": 7.25, "adaptacion": false},
				map[string]any{"nome": "", "nota": "", "adaptacion": false}, // fila baleira: non sae
			},
			"foto_do_grupo": pngDataURI(60, 40, color.NRGBA{R: 40, G: 120, B: 200, A: 255}),
		},
	}
}

func TestSlugFormulario(t *testing.T) {
	for entrada, esperado := range map[string]string{
		"Data da reunión":   "data_da_reunion",
		"  Nº de Alumnos! ": "n_de_alumnos",
		"2.º trimestre":     "c_2_trimestre",
		"Ñandú çá":          "nandu_ca",
		"???":               "",
	} {
		if got := slugFormulario(entrada); got != esperado {
			t.Errorf("slugFormulario(%q) = %q, quería %q", entrada, got, esperado)
		}
	}
}

func TestNormalizarFormularioPonIds(t *testing.T) {
	f := normalizarFormulario(Formulario{Nome: " Ficha ", Campos: []CampoFormulario{
		{Tipo: campoTexto, Etiqueta: "Nome"},
		{Tipo: campoTexto, Etiqueta: "Nome"},                   // mesma etiqueta: id distinto
		{ID: "nome", Tipo: campoTexto, Etiqueta: "Outro"},      // xa traía id: non llo rouba o de arriba
		{ID: "Mal Id", Tipo: campoTexto, Etiqueta: "Inválido"}, // id malo: recalcúlase
	}})
	if f.ID != "ficha" || f.Nome != "Ficha" {
		t.Fatalf("id/nome = %q/%q", f.ID, f.Nome)
	}
	vistos := map[string]bool{}
	for _, c := range f.Campos {
		if vistos[c.ID] {
			t.Errorf("id repetido %q en %+v", c.ID, f.Campos)
		}
		vistos[c.ID] = true
	}
	if f.Campos[2].ID != "nome" {
		t.Errorf("o campo con id propio perdeuno: %+v", f.Campos)
	}
	if err := validarFormulario(f); err != nil {
		t.Errorf("o normalizado debería validar: %v", err)
	}
}

func TestValidarFormulario(t *testing.T) {
	casos := map[string]Formulario{
		"sen nome":         {ID: "a"},
		"tipo raro":        {Nome: "A", ID: "a", Campos: []CampoFormulario{{ID: "x", Tipo: "magico", Etiqueta: "X"}}},
		"escolla sen ops":  {Nome: "A", ID: "a", Campos: []CampoFormulario{{ID: "x", Tipo: campoEscolla, Etiqueta: "X"}}},
		"taboa sen cols":   {Nome: "A", ID: "a", Campos: []CampoFormulario{{ID: "x", Tipo: campoTaboa, Etiqueta: "X"}}},
		"sen etiqueta":     {Nome: "A", ID: "a", Campos: []CampoFormulario{{ID: "x", Tipo: campoTexto}}},
		"id repetido":      {Nome: "A", ID: "a", Campos: []CampoFormulario{{ID: "x", Tipo: campoTexto, Etiqueta: "X"}, {ID: "x", Tipo: campoTexto, Etiqueta: "Y"}}},
		"taboa dentro":     {Nome: "A", ID: "a", Campos: []CampoFormulario{{ID: "t", Tipo: campoTaboa, Etiqueta: "T", Columnas: []CampoFormulario{{ID: "u", Tipo: campoTaboa, Etiqueta: "U"}}}}},
		"imaxe en columna": {Nome: "A", ID: "a", Campos: []CampoFormulario{{ID: "t", Tipo: campoTaboa, Etiqueta: "T", Columnas: []CampoFormulario{{ID: "u", Tipo: campoImaxe, Etiqueta: "U"}}}}},
	}
	for nome, f := range casos {
		if err := validarFormulario(f); err == nil {
			t.Errorf("%s: debería fallar", nome)
		}
	}
}

func TestBibliotecaFormularios(t *testing.T) {
	dir := t.TempDir()
	orixinal := dirFormularios
	dirFormularios = func() (string, error) { return filepath.Join(dir, "formularios"), nil }
	defer func() { dirFormularios = orixinal }()

	if l, err := listarFormularios(); err != nil || len(l) != 0 {
		t.Fatalf("biblioteca nova: %v, %v", l, err)
	}
	b, err := gardarFormulario(Formulario{Nome: "Beta", Campos: []CampoFormulario{{Tipo: campoTexto, Etiqueta: "Nome"}}})
	if err != nil || b.ID != "beta" {
		t.Fatalf("gardar: %+v, %v", b, err)
	}
	if _, err := gardarFormulario(Formulario{Nome: "alfa"}); err != nil {
		t.Fatal(err)
	}
	// Un ficheiro estragado non impide ver os demais.
	if err := os.WriteFile(filepath.Join(dir, "formularios", "roto.json"), []byte("{isto non"), 0o644); err != nil {
		t.Fatal(err)
	}
	l, err := listarFormularios()
	if err != nil || len(l) != 2 || l[0].ID != "alfa" || l[1].ID != "beta" {
		t.Fatalf("listar: %+v, %v", l, err)
	}
	if _, err := gardarFormulario(Formulario{Nome: "Malo", Campos: []CampoFormulario{{Tipo: "raro", Etiqueta: "x"}}}); err == nil {
		t.Error("gardar un formulario inválido debería fallar")
	}
	if _, err := lerFormulario("../fóra"); err == nil {
		t.Error("un id con ruta non debe valer")
	}
	if err := eliminarFormulario("beta"); err != nil {
		t.Fatal(err)
	}
	if err := eliminarFormulario("beta"); err != nil {
		t.Errorf("eliminar dúas veces non debería fallar: %v", err)
	}
	if l, _ := listarFormularios(); len(l) != 1 {
		t.Errorf("tras eliminar: %+v", l)
	}
}

func TestValoresFormatados(t *testing.T) {
	for entrada, esperado := range map[any]string{"3.5": "3,5", "3,5": "3,5", 3.5: "3,5", "1.234,5": "1234,5", "7": "7", 27.0: "27"} {
		if got, ok := valorNumero(entrada); !ok || got != esperado {
			t.Errorf("valorNumero(%v) = %q,%v; quería %q", entrada, got, ok, esperado)
		}
	}
	if _, ok := valorNumero("doce"); ok {
		t.Error("«doce» non é un número")
	}
	for entrada, esperado := range map[string]string{"2026-01-15": "15/01/2026", "15/01/2026": "15/01/2026", "5/1/2026": "05/01/2026"} {
		if got, ok := valorData(entrada); !ok || got != esperado {
			t.Errorf("valorData(%q) = %q,%v; quería %q", entrada, got, ok, esperado)
		}
	}
	if _, ok := valorData("ontem"); ok {
		t.Error("«ontem» non é unha data")
	}
	for _, v := range []any{true, "si", "Sí", "TRUE", "x", 1.0} {
		if !valorBool(v) {
			t.Errorf("valorBool(%v) debería ser certo", v)
		}
	}
	for _, v := range []any{false, "non", "", nil, 0.0} {
		if valorBool(v) {
			t.Errorf("valorBool(%v) debería ser falso", v)
		}
	}
}

func TestBloquesFormularioCableadoAutomatico(t *testing.T) {
	f := normalizarFormulario(formularioMostra())
	d := datosMostra(t)
	tx := novosTextosDoc(nil, map[string]string{})
	bloques, avisos := bloquesFormulario(f, d, tx)
	if len(avisos) != 0 {
		t.Errorf("avisos inesperados: %v", avisos)
	}
	var tipos []string
	for _, b := range bloques {
		tipos = append(tipos, b.Tipo)
	}
	// sección, ficha (grupo…familias), sección, título+markdown, título+táboa, imaxe.
	esperado := "titulo,ficha,titulo,titulo,markdown,titulo,taboa,imaxe"
	if strings.Join(tipos, ",") != esperado {
		t.Fatalf("bloques = %s, quería %s", strings.Join(tipos, ","), esperado)
	}
	ficha := bloques[1].Pares
	if len(ficha) != 5 || ficha[1] != [2]string{"Alumnado", "27"} || ficha[2] != [2]string{"Data da reunión", "15/01/2026"} || ficha[4] != [2]string{"Reunión coas familias", "Si"} {
		t.Errorf("ficha = %v", ficha)
	}
	taboa := bloques[6]
	if len(taboa.Filas) != 2 || taboa.Filas[0][1] != "4,5" || taboa.Filas[1][1] != "7,25" || taboa.Filas[1][2] != "Non" {
		t.Errorf("táboa (a fila baleira non debe saír) = %v", taboa.Filas)
	}
}

func TestBloquesFormularioAvisosEBaleiro(t *testing.T) {
	f := normalizarFormulario(formularioMostra())
	tx := novosTextosDoc(nil, map[string]string{})
	bloques, avisos := bloquesFormulario(f, DatosFormulario{Valores: map[string]any{
		"alumnado": "moitos", "data_da_reunion": "ontem", "trimestre": "4.º",
	}}, tx)
	joined := strings.Join(avisos, "|")
	for _, quere := range []string{"obrigatorio «Grupo»", "«Alumnado» non ten un valor válido", "«Data da reunión»", "«Trimestre»"} {
		if !strings.Contains(joined, quere) {
			t.Errorf("falta o aviso %q en %q", quere, joined)
		}
	}
	// Ningún texto longo, imaxe nin táboa: sen eses bloques; as verificacións
	// dan «Non» (nunca están baleiras).
	for _, b := range bloques {
		if b.Tipo == bloqueTaboa || b.Tipo == bloqueImaxe || b.Tipo == bloqueMarkdown {
			t.Errorf("bloque de máis: %+v", b)
		}
	}

	solo := Formulario{ID: "s", Nome: "S", Campos: []CampoFormulario{{ID: "a", Tipo: campoTexto, Etiqueta: "A"}}}
	bloques, _ = bloquesFormulario(solo, DatosFormulario{}, tx)
	if len(bloques) != 0 {
		t.Errorf("un formulario sen datos non debe poñer ningún texto xenérico: %+v", bloques)
	}
}

func TestVariablesFormulario(t *testing.T) {
	f := normalizarFormulario(formularioMostra())
	v := variablesFormulario(f, datosMostra(t), "ficha.json", agoraFormulario, nil)
	for clave, esperado := range map[string]string{
		"CAMPO_GRUPO": "2.º B", "CAMPO_DATA_DA_REUNION": "15/01/2026", "CAMPO_ALUMNADO": "27",
		"CENTRO": "IES Castelao & Fillos, 100% público", "CURSO": "2025-2026", "FORMULARIO": "Informe de titoría",
		"DATA_LONGA": "20 de setembro de 2026",
	} {
		if v[clave] != esperado {
			t.Errorf("%s = %q, quería %q", clave, v[clave], esperado)
		}
	}
	if _, hai := v["CAMPO_OBSERVACIONS"]; hai {
		t.Error("un texto longo non é variable de plantilla")
	}
}

func TestMarkdownFormulario(t *testing.T) {
	p := PedidoFormulario{Formulario: formularioMostra(), Datos: datosMostra(t)}
	dir := t.TempDir()
	md, avisos, err := markdownFormulario(p, "md", dir, agoraFormulario, nil)
	if err != nil || len(avisos) != 0 {
		t.Fatalf("err=%v avisos=%v", err, avisos)
	}
	for _, quere := range []string{
		"## Datos do grupo",
		"**Grupo:** 2.º B", "**Data da reunión:** 15/01/2026", "**Reunión coas familias:** Si",
		"### Observacións", "**traballador**", "| Nome | Nota | Adaptación |", "| Uxía | 7,25 | Non |", "![Foto do grupo](",
	} {
		if !strings.Contains(md, quere) {
			t.Errorf("falta %q no Markdown:\n%s", quere, md)
		}
	}
	if strings.Contains(md, "|  |  |") {
		t.Error("a fila baleira da táboa non debe saír")
	}
	if strings.Contains(md, "Seguimento trimestral") {
		t.Error("a descrición do formulario (é da biblioteca) non debe saír no documento")
	}
	if _, err := os.Stat(filepath.Join(dir, "images")); err != nil {
		t.Errorf("a imaxe debería escribirse en images/: %v", err)
	}
}

func TestMarkdownFormularioConPlantillaEDespacho(t *testing.T) {
	// Pasa polo mesmo camiño ca as exportacións: a plantilla envolve UNHA vez.
	pl := &PlantillaLatex{Nome: "t", Latex: "{{CORPO}}", Markdown: "CABECEIRA {{CENTRO}}\n\n{{CORPO}}\n\nPE"}
	req := XerarInformeCronogramaRequest{
		Ferramenta: ferramentaFormulario, NomeBase: "ficha",
		Formulario: &PedidoFormulario{Formulario: formularioMostra(), Datos: datosMostra(t)},
	}
	md, _ := markdownInformeConPlantilla(req, pl, "md", t.TempDir())
	if strings.Count(md, "CABECEIRA") != 1 || !strings.Contains(md, "IES Castelao") || !strings.Contains(md, "**Grupo:** 2.º B") {
		t.Errorf("envoltorio raro:\n%s", md)
	}
	// Word e OpenDocument saen polo mesmo despacho (ver informe_formulario_ofi_test.go).
	for _, formato := range []string{"docx", "odt"} {
		if b, _, err := datosOfimaticaDe(req, pl, formato); err != nil || len(b) < 500 {
			t.Errorf("%s: %d bytes, %v", formato, len(b), err)
		}
	}
}

func TestLatexFormularioTemOsMacros(t *testing.T) {
	p := PedidoFormulario{Formulario: formularioMostra(), Datos: datosMostra(t)}
	tex, avisos, err := latexFormulario(p, "pdflatex", nil, t.TempDir(), agoraFormulario, nil, "ficha")
	if err != nil || len(avisos) != 0 {
		t.Fatalf("err=%v avisos=%v", err, avisos)
	}
	for _, quere := range []string{
		`\formseccion{Datos do grupo}`,
		`\formdato{Data da reunión}{15/01/2026}`, `\begin{longtable}`, `\rowcolor{tf-indigo}`, `\includegraphics[width=`,
		`\providecommand{\formdato}`,
	} {
		if !strings.Contains(tex, quere) {
			t.Errorf("falta %q no LaTeX", quere)
		}
	}
	// Sen portada nin sinatura (o formulario non as pide).
	if strings.Contains(tex, "tikzpicture}[remember picture") || strings.Contains(tex, `\rule{58mm}`) {
		t.Error("o formulario non pediu portada nin sinatura")
	}
	// Con portada e sinatura si.
	p.Formulario.Portada, p.Formulario.Sinatura = true, true
	tex, _, err = latexFormulario(p, "pdflatex", nil, t.TempDir(), agoraFormulario, nil, "ficha")
	if err != nil || !strings.Contains(tex, "remember picture") || !strings.Contains(tex, `\rule{58mm}`) {
		t.Errorf("faltan a portada ou a sinatura (err=%v)", err)
	}
	if strings.Contains(tex, `\formtitulo{`) && strings.Count(tex, `\formtitulo{Informe`) > 0 {
		t.Error("con portada o título non se repite no corpo")
	}
}

func TestLatexFormularioEscapaOTexto(t *testing.T) {
	p := PedidoFormulario{
		Formulario: Formulario{ID: "e", Nome: "A & B_100%", Campos: []CampoFormulario{{ID: "x", Tipo: campoTexto, Etiqueta: "Custo (€) #1"}}},
		Datos:      DatosFormulario{Valores: map[string]any{"x": `a_b & c% {d} \ ~ ^`}},
	}
	tex, _, err := latexFormulario(p, "pdflatex", nil, t.TempDir(), agoraFormulario, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(tex, "A & B_100%") || strings.Contains(tex, "a_b & c%") {
		t.Errorf("texto sen escapar:\n%s", tex)
	}
}

func TestCompilarFormularioPDF(t *testing.T) {
	if !commandExists("xelatex") && !commandExists("pdflatex") {
		t.Skip("sen LaTeX")
	}
	pedido := PedidoFormulario{Formulario: formularioMostra(), Datos: datosMostra(t)}
	comPortada := pedido
	comPortada.Formulario.Portada, comPortada.Formulario.Sinatura = true, true

	// O caso importante: unha plantilla co seu propio Estilo (que non coñece
	// os macros \form*) ten que seguir compilando.
	estiloPropio := &PlantillaLatex{Nome: "estilo propio", Latex: "{{CORPO}}", Estilo: "% deseño propio, sen macros de formulario\n\\setlength{\\parindent}{0pt}\n"}
	// E unha que redefine un macro.
	estiloRedefine := &PlantillaLatex{Nome: "redefine", Latex: "{{CORPO}}", Estilo: "\\newcommand{\\formseccion}[1]{\\par\\textbf{>>> #1 <<<}\\par}\n"}

	casos := map[string]struct {
		p  PedidoFormulario
		pl *PlantillaLatex
	}{
		"sen plantilla":       {pedido, nil},
		"portada e sinatura":  {comPortada, nil},
		"estilo propio":       {pedido, estiloPropio},
		"estilo que redefine": {pedido, estiloRedefine},
	}
	for nome, c := range casos {
		for _, motor := range []string{"pdflatex", "xelatex"} {
			if !commandExists(motor) {
				continue
			}
			t.Run(nome+"/"+motor, func(t *testing.T) {
				res, err := compilarFormulario(c.p, motor, nil, t.TempDir(), c.pl, "ficha")
				if err != nil {
					t.Fatalf("non compila: %v\n%s", err, res.Log)
				}
				if !bytes.HasPrefix(res.PDF, []byte("%PDF")) {
					t.Errorf("o resultado non é un PDF (%d bytes)", len(res.PDF))
				}
			})
		}
	}
}

func TestBorradorDeCadaFormulario(t *testing.T) {
	if borradorFerramentaNome("form:informe_titoria") != "formulario-informe_titoria-borrador.json" {
		t.Error("cada formulario ten o seu borrador")
	}
	// Un id malo ou raro non debe poder saír do cartafol nin pisar o do cronograma.
	for _, malo := range []string{"form:../x", "form:", "form:A B", "form:a/b"} {
		if borradorFerramentaNome(malo) != "cronograma-borrador.json" {
			t.Errorf("%q non é unha ferramenta de formulario válida", malo)
		}
	}
	if borradorFerramentaNome("actas") != "actas-borrador.json" || borradorFerramentaNome("cronograma") != "cronograma-borrador.json" {
		t.Error("o das actas e o do cronograma non cambian")
	}
}

func TestNomeXanelaFormulario(t *testing.T) {
	if n, id := nomeXanelaFormulario("informe_titoria"); n != "formulario-informe_titoria" || id != "informe_titoria" {
		t.Errorf("%q %q", n, id)
	}
	if n, id := nomeXanelaFormulario(""); n != "formulario-novo" || id != "novo" {
		t.Errorf("un formulario sen id é «novo»: %q %q", n, id)
	}
	// Un id raro non pode sacar o nome do sitio nin coar caracteres de ruta.
	if n, _ := nomeXanelaFormulario("../x y"); strings.ContainsAny(n, "/. ") {
		t.Errorf("nome sen limpar: %q", n)
	}
	// Sen xanela nativa (probas, servidor) non hai erro: devolve false e o frontend usa a capa.
	if ok, err := (&App{}).AbrirXanelaFormulario("x", "{}"); ok || err != nil {
		t.Errorf("sen aplicación non debería abrir nada: %v %v", ok, err)
	}
}

func bibliotecaTemporal(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	orixinal := dirFormularios
	dirFormularios = func() (string, error) { return filepath.Join(dir, "formularios"), nil }
	t.Cleanup(func() { dirFormularios = orixinal })
}

// O modo (deseño/produción) só cambia por cambiarModoFormulario: gardar non o
// toca, nin sequera se o esquema que chega di outra cousa, e un formulario en
// produción non se pode gardar.
func TestModoFormulario(t *testing.T) {
	bibliotecaTemporal(t)
	campos := []CampoFormulario{{Tipo: campoTexto, Etiqueta: "Nome"}}

	f, err := gardarFormulario(Formulario{Nome: "Ficha", Campos: campos})
	if err != nil || f.Modo != modoDeseno || f.enProduccion() {
		t.Fatalf("un formulario novo nace en deseño: %+v, %v", f, err)
	}
	// Un esquema novo que pida nacer en produción (a IA, un ficheiro manipulado...) nace igualmente en deseño.
	g, err := gardarFormulario(Formulario{Nome: "Colado", Modo: modoProduccion, Campos: campos})
	if err != nil || g.Modo != modoDeseno {
		t.Fatalf("un formulario novo non pode nacer en produción: %+v, %v", g, err)
	}

	if _, err := cambiarModoFormulario("ficha", "vermello"); err == nil {
		t.Error("un modo descoñecido debería fallar")
	}
	if _, err := cambiarModoFormulario("non_existe", modoProduccion); err == nil {
		t.Error("cambiar o modo dun formulario que non existe debería fallar")
	}
	p, err := cambiarModoFormulario("ficha", modoProduccion)
	if err != nil || !p.enProduccion() {
		t.Fatalf("pasar a produción: %+v, %v", p, err)
	}
	// O modo persiste e o resto do esquema queda igual.
	l, err := lerFormulario("ficha")
	if err != nil || !l.enProduccion() || l.Nome != "Ficha" || len(l.Campos) != 1 {
		t.Fatalf("o modo non persistiu: %+v, %v", l, err)
	}
	// En produción o esquema está conxelado: gardar falla e non toca o ficheiro.
	if _, err := gardarFormulario(Formulario{ID: "ficha", Nome: "Outro nome", Modo: modoDeseno, Campos: campos}); err == nil {
		t.Error("gardar un formulario en produción debería fallar")
	}
	if l, _ := lerFormulario("ficha"); l.Nome != "Ficha" || !l.enProduccion() {
		t.Errorf("o intento de gardar cambiou o formulario: %+v", l)
	}
	// Nin sequera se abre o deseñador (o seguro de Go, ademais de non ofrecer o botón).
	if _, err := (&App{}).AbrirXanelaFormulario("ficha", "{}"); err == nil {
		t.Error("abrir o deseñador dun formulario en produción debería fallar")
	}
	// Volver a deseño permite gardar, e gardar non cambia o modo.
	if d, err := cambiarModoFormulario("ficha", modoDeseno); err != nil || d.enProduccion() {
		t.Fatalf("pasar a deseño: %+v, %v", d, err)
	}
	if s, err := gardarFormulario(Formulario{ID: "ficha", Nome: "Ficha 2", Campos: campos}); err != nil || s.Modo != modoDeseno {
		t.Fatalf("gardar en deseño: %+v, %v", s, err)
	}
}

// Un ficheiro de antes (sen o campo «modo») é un formulario en deseño.
func TestFormularioSenModoEDeseno(t *testing.T) {
	bibliotecaTemporal(t)
	dir, _ := dirFormularios()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "vello.json"), []byte(`{"id":"vello","nome":"Vello","campos":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := lerFormulario("vello")
	if err != nil || f.enProduccion() {
		t.Fatalf("un formulario sen modo debe valer coma deseño: %+v, %v", f, err)
	}
	if _, err := gardarFormulario(f); err != nil {
		t.Errorf("gardar un formulario sen modo: %v", err)
	}
}

func TestQuitarReferencias(t *testing.T) {
	casos := map[string]string{
		"1. She began talking an hour ago. (for)  [PAU 2021 · ordinaria]":    "1. She began talking an hour ago. (for)",
		"Resposta: since [clave OFICIAL 2023: admite as dúas formas] e máis": "Resposta: since e máis",
		"Ver [a documentación](http://x.org) e [ref][1] e [ok]":              "Ver [a documentación](http://x.org) e [ref][1] e [ok]",
		"sen corchetes": "sen corchetes",
		"liña 1 [PAU 2020 · extraordinaria]\nliña 2 [PAU 2021 · ordinaria]  \nz": "liña 1\nliña 2\nz",
	}
	for in, esperado := range casos {
		if got := quitarReferencias(in); got != esperado {
			t.Errorf("%q → %q (esperado %q)", in, got, esperado)
		}
	}
	d := DatosFormulario{SenReferencias: true, Valores: map[string]any{
		"a": "texto [PAU 2021 · ordinaria]", "n": 3.0,
		"t": []any{map[string]any{"c": "fila [clave OFICIAL 2023]"}},
	}}
	l := d.aplicarReferencias()
	if l.Valores["a"] != "texto" || l.Valores["n"] != 3.0 || l.Valores["t"].([]any)[0].(map[string]any)["c"] != "fila" {
		t.Errorf("aplicar: %+v", l.Valores)
	}
	if d.Valores["a"] != "texto [PAU 2021 · ordinaria]" {
		t.Error("os datos orixinais non se tocan")
	}
	d.SenReferencias = false
	if d.aplicarReferencias().Valores["a"] != "texto [PAU 2021 · ordinaria]" {
		t.Error("coas referencias activas non se quita nada")
	}
}

// O corpo dun formulario non leva títulos nin textos automáticos: o título vai
// na plantilla ({{TITULO}}) ou nun campo.
func TestFormularioSenTituloAutomatico(t *testing.T) {
	p := PedidoFormulario{Formulario: formularioMostra(), Datos: datosMostra(t)}
	tex, _, err := latexFormulario(p, "pdflatex", nil, t.TempDir(), agoraFormulario, nil, "ficha")
	if err != nil || strings.Contains(tex, `\formtitulo{Informe`) {
		t.Errorf("o LaTeX leva título automático (err=%v)", err)
	}
	md, _, _ := markdownFormulario(p, "md", t.TempDir(), agoraFormulario, nil)
	if strings.Contains(md, "# Informe de titoría") {
		t.Error("o Markdown leva título automático")
	}
}
