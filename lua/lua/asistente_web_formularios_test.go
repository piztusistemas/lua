package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// Unha páxina coma a de datosclima.es: o formulario envíase con POST e a
// acción de verdade ponna un script.
const paxinaFormularioProba = `<html><body>
<form name="Seleccion" action="/Aemethistorico/Meteostation.php" method="POST">
<select name="Provincia" onChange="this.form.submit()">
  <option value="">--</option>
  <option value="A CORUÑA" selected>A CORUÑA</option>
  <option value="LUGO">LUGO</option>
</select>
<select name="id_hija"><option value="1399">VIMIANZO</option><option value="1387">A CORUÑA</option></select>
<input type="hidden" name="Indicativo" value="1399">
<input type="text" name="Iyear" maxlength="4" value="">
<textarea name="notas">ola</textarea>
<input type="button" onclick="Valida(this.form)" value="BUSCAR">
</form>
<form action="https://outro.example/busca"><input name="q"></form>
<script>function Valida(form){ document.Seleccion.action ="Tempestad.php"; form.submit() }</script>
</body></html>`

func TestFormulariosDeHTMLDescribeCamposOpcionsEAccionsDosScripts(t *testing.T) {
	base, _ := url.Parse("https://datosclima.es/Aemethistorico/Meteostation.php")
	txt := formulariosDeHTML([]byte(paxinaFormularioProba), base)
	for _, quero := range []string{
		"Formulario 1 «Seleccion»: POST https://datosclima.es/Aemethistorico/Meteostation.php",
		"Provincia (select)", "«A CORUÑA» = A CORUÑA (escollida)", "«LUGO» = LUGO",
		"id_hija (select)", "«1399» = VIMIANZO",
		"Indicativo (hidden) = «1399»", "Iyear (text) = «»", "notas (textarea) = «ola»",
		"Formulario 2: GET https://outro.example/busca", "q (text)",
		// A acción que pon o script, xa resolta contra a páxina.
		"https://datosclima.es/Aemethistorico/Tempestad.php",
	} {
		if !strings.Contains(txt, quero) {
			t.Errorf("falta %q en:\n%s", quero, txt)
		}
	}
	if strings.Contains(txt, "BUSCAR") {
		t.Errorf("os botóns non son campos que enviar:\n%s", txt)
	}
}

func TestFormulariosDeHTMLSenFormularios(t *testing.T) {
	base, _ := url.Parse("https://exemplo.org/")
	if txt := formulariosDeHTML([]byte(`<p>nada</p>`), base); !strings.Contains(txt, "non ten formularios") {
		t.Errorf("%q", txt)
	}
}

func TestWebEnviarFormularioPOSTeGET(t *testing.T) {
	webPermitirLocal = true
	defer func() { webPermitirLocal = false }()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Error(err)
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte("<html><body><p>" + r.Method + " " + r.Header.Get("Content-Type") +
			"</p><p>provincia=" + r.Form.Get("Provincia") + " ano=" + r.Form.Get("Iyear") + "</p></body></html>"))
	}))
	defer srv.Close()

	txt, err := webEnviarFormulario(srv.URL+"/Tempestad.php", "", map[string]any{"Provincia": "A CORUÑA", "Iyear": 2005.0}, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, quero := range []string{"POST application/x-www-form-urlencoded", "provincia=A CORUÑA ano=2005"} {
		if !strings.Contains(txt, quero) {
			t.Errorf("falta %q en %q", quero, txt)
		}
	}
	txt, err = webEnviarFormulario(srv.URL+"/busca?x=1", "get", map[string]any{"Iyear": "2010"}, 0, 0)
	if err != nil || !strings.Contains(txt, "GET") || !strings.Contains(txt, "ano=2010") {
		t.Errorf("GET: %v %q", err, txt)
	}
	// Cos formularios da resposta (o paso seguinte dun formulario encadeado).
	form, err := webEnviarFormularioEVerFormularios(srv.URL+"/x", "POST", map[string]any{"Provincia": "LUGO"})
	if err != nil || !strings.Contains(form, "A páxina non ten formularios") {
		t.Errorf("formularios da resposta: %v %q", err, form)
	}
	if _, err := webEnviarFormulario(srv.URL, "DELETE", nil, 0, 0); err == nil {
		t.Error("só se admiten POST e GET")
	}
}

func TestWebEnviarFormularioRexeitaEnderezosLocais(t *testing.T) {
	for _, u := range []string{"http://127.0.0.1:8080/", "http://localhost/x", "http://192.168.1.1/login", "file:///etc/passwd"} {
		if _, err := webEnviarFormulario(u, "POST", map[string]any{"a": "b"}, 0, 0); err == nil {
			t.Errorf("%s debería rexeitarse", u)
		}
	}
}

func TestWebConsultarFormulariosPolaFerramenta(t *testing.T) {
	webPermitirLocal = true
	defer func() { webPermitirLocal = false }()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(paxinaFormularioProba))
	}))
	defer srv.Close()
	var exec func(*App, map[string]any) (any, error)
	for _, f := range ferramentasBackend {
		if f.def.Nome == "web_consultar" {
			exec = f.exec
		}
	}
	if exec == nil {
		t.Fatal("non hai web_consultar")
	}
	r, err := exec(&App{}, map[string]any{"url": srv.URL + "/Aemethistorico/Meteostation.php", "formularios": true})
	if err != nil {
		t.Fatal(err)
	}
	if s, _ := r.(string); !strings.Contains(s, srv.URL+"/Aemethistorico/Tempestad.php") || !strings.Contains(s, "id_hija (select)") {
		t.Errorf("%v", r)
	}
}

// Os gráficos (pgfplots dentro dun bloque tikz) teñen o paquete no PDF, nas
// plantillas completas e nos recortes de Word/ODT/vista.
func TestPgfplotsNosPreambulos(t *testing.T) {
	if s := documentoRecortes("pdflatex", nil); !strings.Contains(s, `\usepackage{pgfplots}`) || !strings.Contains(s, "1000 sep={}") {
		t.Errorf("os recortes non cargan pgfplots:\n%s", s)
	}
	ten := false
	for _, p := range paquetesInformeCrono {
		if p.Nome == "pgfplots" && strings.Contains(p.Linha, "use comma") {
			ten = true
		}
	}
	if !ten {
		t.Error("paquetesInformeCrono non leva pgfplots")
	}
	tex, _, err := latexInformeCrono(PlanCrono{}, "pdflatex", nil, t.TempDir(), time.Date(2026, 11, 11, 10, 0, 0, 0, time.UTC), nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(tex, `\usepackage{pgfplots}`) {
		t.Error("o preámbulo do PDF non carga pgfplots")
	}
}

// Un gráfico de temperaturas compila de verdade (se hai LaTeX).
func TestGraficoPgfplotsCompila(t *testing.T) {
	if motorRecortes() == "" || !commandExists("pdftoppm") {
		t.Skip("sen LaTeX ou pdftoppm")
	}
	codigo := `\begin{axis}[width=10cm, height=6cm, xlabel={Ano}, ylabel={Temperatura media ($^\circ$C)}, grid=major]
\addplot[mark=*, blue] coordinates {(2000,13.1) (2005,13.4) (2010,13.6) (2015,13.9) (2020,14.2)};
\addplot[red, dashed, domain=2000:2020] {13.1 + 0.055*(x-2000)};
\end{axis}`
	recs, err := compilarRecortes([]string{envolverDebuxo(debuxoTikz, codigo)}, 72)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 || recs[0].erro != "" || recs[0].im == nil {
		t.Fatalf("o gráfico non compila: %+v", recs)
	}
	// Un gráfico de 10 cm ten de ancho uns 280 pt.
	if recs[0].anchoPt < 200 {
		t.Errorf("o gráfico saíu pequeno de máis: %.0f pt", recs[0].anchoPt)
	}
}

// O caso de uso: un formulario cun gráfico de temperaturas nun campo de
// texto longo sae no PDF (cos dous motores).
func TestFormularioConGraficoPgfplotsCompilaPDF(t *testing.T) {
	pedido := PedidoFormulario{Formulario: formularioMostra(), Datos: datosMostra(t)}
	pedido.Datos.Valores["observacions"] = "Temperatura media anual en Vimianzo (fonte: datosclima.es).\n\n```tikz\n" +
		`\begin{axis}[width=12cm, height=6cm, xlabel={Ano}, ylabel={Temperatura media ($^\circ$C)}, grid=major, legend pos=north west]
\addplot[mark=*, blue] coordinates {(2000,13.1) (2004,13.3) (2008,13.2) (2012,13.6) (2016,13.8) (2020,14.1)};
\addlegendentry{Media anual}
\addplot[red, dashed, domain=2000:2020] {13.05 + 0.048*(x-2000)};
\addlegendentry{Tendencia}
\end{axis}` + "\n```\n\nSobe uns 0,5 °C por década."
	probado := false
	for _, motor := range []string{"pdflatex", "xelatex"} {
		if !commandExists(motor) {
			continue
		}
		probado = true
		t.Run(motor, func(t *testing.T) {
			res, err := compilarFormulario(pedido, motor, nil, t.TempDir(), nil, "ficha")
			if err != nil {
				t.Fatalf("non compila: %v\n%s", err, res.Log)
			}
			if !strings.HasPrefix(string(res.PDF), "%PDF") {
				t.Errorf("o resultado non é un PDF")
			}
		})
	}
	if !probado {
		t.Skip("sen LaTeX")
	}
}

// Calquera fonte: as celas dunha táboa HTML saen separadas (a IA ten que
// saber que número vai en que columna).
func TestTextoDeHTMLSeparaAsCelasDasTaboas(t *testing.T) {
	txt := textoDeHTML([]byte(`<table><tr><th>Ano</th><th>Media</th><th>Máxima</th></tr>
<tr><td>2005</td><td>13,4</td><td>35,0</td></tr><tr><td>2006</td><td><b>13,9</b></td><td>36,1</td></tr></table>`))
	for _, quero := range []string{"Ano | Media | Máxima", "2005 | 13,4 | 35,0", "2006 | 13,9 | 36,1"} {
		if !strings.Contains(txt, quero) {
			t.Errorf("falta %q en %q", quero, txt)
		}
	}
}

// CSV e JSON lense tal cal; un ficheiro binario (Excel, ZIP) non se lle
// pasa á IA coma lixo: avísase de que se achegue.
func TestTextoWebCSVeBinarios(t *testing.T) {
	txt, err := textoWebEnPartes([]byte("ano;media\n2005;13,4\n"), "text/csv", 0, 0)
	if err != nil || !strings.Contains(txt, "2005;13,4") {
		t.Errorf("CSV: %v %q", err, txt)
	}
	_, err = textoWebEnPartes([]byte("PK\x03\x04\x00\x00binario"), "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", 0, 0)
	if err == nil || !strings.Contains(err.Error(), "achegue") {
		t.Errorf("un .xlsx debería dar un erro que diga que se achegue: %v", err)
	}
}
