package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func pedidoOfiFormulario(t *testing.T, f Formulario, d DatosFormulario) XerarInformeCronogramaRequest {
	return XerarInformeCronogramaRequest{Ferramenta: ferramentaFormulario, NomeBase: "ficha", Formulario: &PedidoFormulario{Formulario: f, Datos: d}}
}

func TestDocxDoFormulario(t *testing.T) {
	d := datosMostra(t)
	// Un valor con salto de liña, caracteres de XML e unha lista, unha lista e un salto de páxina pola tubería.
	d.Valores["grupo"] = "2.º B & <C>\nsegunda liña"
	f := formularioMostra()
	f.Sinatura = true
	datos, avisos, err := datosOfimaticaDe(pedidoOfiFormulario(t, f, d), nil, "docx")
	if err != nil || len(avisos) != 0 {
		t.Fatalf("err=%v avisos=%v", err, avisos)
	}
	z := lerZip(t, datos)
	doc := z["word/document.xml"]
	xmlBenFormado(t, "word/document.xml", doc)
	for _, quere := range []string{
		"Datos do grupo", "Grupo: ", "2.º B &amp; &lt;C&gt;", "<w:br/>", "segunda liña",
		"<w:tbl>", "<w:tblHeader/>", "Brais _Q_", "7,25", "<w:drawing>",
		"Ana Pérez", // quen asina (sinatura)
		"<w:pBdr>",  /* estilo dos títulos */
	} {
		if !strings.Contains(doc, quere) && !strings.Contains(z["word/styles.xml"], quere) {
			t.Errorf("falta %q no Word", quere)
		}
	}
	for _, etiqueta := range []string{"Observacións", "Alumnado con medidas", "Foto do grupo"} {
		if strings.Contains(doc, etiqueta) {
			t.Errorf("a etiqueta %q non debe saír no Word", etiqueta)
		}
	}
	if n := strings.Count(doc, "<w:tr>"); n != 3 { // cabeceira + 2 filas (a baleira non sae)
		t.Errorf("filas da táboa = %d", n)
	}
	// A imaxe vai no paquete.
	temMedia := false
	for nome := range z {
		if strings.HasPrefix(nome, "word/media/") {
			temMedia = true
		}
	}
	if !temMedia {
		t.Error("a imaxe do formulario non entrou no paquete")
	}
}

func TestOdtDoFormulario(t *testing.T) {
	d := datosMostra(t)
	d.Valores["grupo"] = "2.º B & <C>\nsegunda liña"
	f := formularioMostra()
	datos, avisos, err := datosOfimaticaDe(pedidoOfiFormulario(t, f, d), nil, "odt")
	if err != nil || len(avisos) != 0 {
		t.Fatalf("err=%v avisos=%v", err, avisos)
	}
	z := lerZip(t, datos)
	xmlBenFormado(t, "content.xml", z["content.xml"])
	xmlBenFormado(t, "styles.xml", z["styles.xml"])
	c := z["content.xml"]
	for _, quere := range []string{
		"2.º B &amp; &lt;C&gt;", "<text:line-break/>", `<table:table table:name="Taboa1"`, "<table:table-header-rows>",
		`table:number-columns-repeated="3"`, "LuaCol3", "Brais _Q_", "7,25", "<draw:image",
	} {
		if !strings.Contains(c, quere) {
			t.Errorf("falta %q no ODT", quere)
		}
	}
	if n := strings.Count(c, "<table:table-row>"); n != 3 {
		t.Errorf("filas = %d", n)
	}
}

func TestOfimaticaFormularioListaSaltoEDoisTaboas(t *testing.T) {
	f := normalizarFormulario(formularioNotas())
	f.Fluxo = fluxoNotas() // ten lista, táboa, salto de páxina e un bloque condicionado
	req := pedidoOfiFormulario(t, f, datosNotas())
	dx, _, err := datosOfimaticaDe(req, nil, "docx")
	if err != nil {
		t.Fatal(err)
	}
	doc := lerZip(t, dx)["word/document.xml"]
	xmlBenFormado(t, "docx", doc)
	for _, quere := range []string{"Repetidores", "ListBullet", "Brais", "Carme", "<w:pageBreakBefore/>", "A media chega ao aprobado."} {
		if !strings.Contains(doc, quere) {
			t.Errorf("falta %q no Word", quere)
		}
	}
	do, _, err := datosOfimaticaDe(req, nil, "odt")
	if err != nil {
		t.Fatal(err)
	}
	c := lerZip(t, do)["content.xml"]
	xmlBenFormado(t, "odt", c)
	for _, quere := range []string{"Repetidores", `text:style-name="ListItem"`, "PaxinaNova", "A media chega ao aprobado."} {
		if !strings.Contains(c, quere) {
			t.Errorf("falta %q no ODT", quere)
		}
	}
}

func TestOfimaticaFormularioSenDatosEInvalidoNonRompe(t *testing.T) {
	f := formularioMostra()
	for _, formato := range []string{"docx", "odt"} {
		b, _, err := datosOfimaticaDe(pedidoOfiFormulario(t, f, DatosFormulario{Valores: map[string]any{}}), nil, formato)
		if err != nil || len(b) < 500 {
			t.Errorf("%s baleiro: %v", formato, err)
		}
		if _, _, err := datosOfimaticaDe(pedidoOfiFormulario(t, Formulario{}, DatosFormulario{}), nil, formato); err == nil {
			t.Errorf("%s: un formulario sen nome non debería exportarse", formato)
		}
	}
}

func TestTintarHex(t *testing.T) {
	if got := tintarHex("1F3864", 0); got != "1F3864" {
		t.Errorf("k=0: %s", got)
	}
	if got := tintarHex("1F3864", 1); got != "FFFFFF" {
		t.Errorf("k=1: %s", got)
	}
	if got := tintarHex("nada", 0.5); got != "EEF2FF" {
		t.Errorf("cor mala: %s", got)
	}
}

// Con LibreOffice de verdade: o que escribimos ten que ABRIRSE e ter o texto.
func TestFormularioOfimaticaAbreEnLibreOffice(t *testing.T) {
	if !commandExists("soffice") || !commandExists("pdftotext") {
		t.Skip("sen LibreOffice ou pdftotext")
	}
	d := datosMostra(t)
	f := formularioMostra()
	for _, formato := range []string{"docx", "odt"} {
		datos, _, err := datosOfimaticaDe(pedidoOfiFormulario(t, f, d), nil, formato)
		if err != nil {
			t.Fatal(err)
		}
		tipo := tipoAdxDocx
		if formato == "odt" {
			tipo = tipoAdxOdt
		}
		pdf, limpar, err := converterAPDFLibreOffice(Adxunto{ID: "x", Nome: "f." + formato, Tipo: tipo}, datos)
		if err != nil {
			t.Fatalf("%s: LibreOffice non o abre: %v", formato, err)
		}
		defer limpar()
		txt := filepath.Join(t.TempDir(), "o.txt")
		if _, err := executarComTempo(60, "pdftotext", "-layout", pdf, txt); err != nil {
			t.Fatal(err)
		}
		b, _ := os.ReadFile(txt)
		// Sen "Informe de titoría": o corpo non leva título automático (ver
		// TestFormularioSenTituloAutomatico); o título vai na plantilla.
		for _, quere := range []string{"Datos do grupo", "2.º B", "Uxía", "7,25", "Nota", "Adaptación"} {
			if !strings.Contains(string(b), quere) {
				t.Errorf("%s: falta %q no PDF de LibreOffice:\n%s", formato, quere, b)
			}
		}
	}
}
