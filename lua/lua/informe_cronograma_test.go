package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestParseMarkdownCrono(t *testing.T) {
	md := "# Título\n\nUn **negro** e _cursivo_ con [web](https://x.gal).\n\n- un\n- dous\n1. tres\n\n> cita\n\n:::centro\ncentrado\n:::\n![logo](img:a1 =60%)\n![f](img:b =120x80)"
	bl := parseMarkdownCrono(md)

	tipos := []cronoTipoBloque{}
	for _, b := range bl {
		tipos = append(tipos, b.Tipo)
	}
	esperado := []cronoTipoBloque{cronoTitulo, cronoParrafo, cronoItem, cronoItem, cronoItem, cronoCita, cronoParrafo, cronoImaxe, cronoImaxe}
	if len(tipos) != len(esperado) {
		t.Fatalf("tipos = %v, esperaba %v", tipos, esperado)
	}
	for i := range esperado {
		if tipos[i] != esperado[i] {
			t.Fatalf("bloque %d: tipo %v, esperaba %v", i, tipos[i], esperado[i])
		}
	}
	if bl[0].Nivel != 1 {
		t.Errorf("nivel do título = %d", bl[0].Nivel)
	}
	if !bl[2].Ordenada && bl[4].Ordenada == bl[2].Ordenada {
		t.Errorf("o terceiro item debería ser ordenado e os dous primeiros non")
	}
	// O parágrafo dentro de :::centro vai centrado; o peche ":::" devolve o
	// aliñamento por defecto, así que a imaxe posterior non.
	if bl[6].Alin != cronoAlinCentro || bl[7].Alin != cronoAlinDefecto {
		t.Errorf("bloque :::centro mal aplicado: %v %v", bl[6].Alin, bl[7].Alin)
	}
	if bl[7].Imaxe.Ref != "a1" || bl[7].Imaxe.AnchoPct != 60 {
		t.Errorf("imaxe 1 = %+v", bl[7].Imaxe)
	}
	if bl[8].Imaxe.AnchoMM != 120 || bl[8].Imaxe.AltoMM != 80 {
		t.Errorf("imaxe 2 = %+v", bl[8].Imaxe)
	}

	tr := bl[1].Tramos
	var negro, cursivo, ligazon bool
	for _, x := range tr {
		if x.Texto == "negro" && x.Negrita {
			negro = true
		}
		if x.Texto == "cursivo" && x.Cursiva {
			cursivo = true
		}
		if x.Texto == "web" && x.URL == "https://x.gal" {
			ligazon = true
		}
	}
	if !negro || !cursivo || !ligazon {
		t.Errorf("inline: negro=%v cursivo=%v ligazon=%v en %+v", negro, cursivo, ligazon, tr)
	}
}

func TestTamanoLetraCrono(t *testing.T) {
	tr := parseInlineCrono("a [fs=gr]grande **negro**[/fs] b")
	var gr bool
	for _, x := range tr {
		if x.Escala == 1.3 && strings.Contains(x.Texto, "grande") {
			gr = true
		}
		if x.Texto == "negro" && (x.Escala != 1.3 || !x.Negrita) {
			t.Errorf("o tramo interior perdeu o tamaño ou a negriña: %+v", x)
		}
	}
	if !gr {
		t.Errorf("non se atopou o tramo grande: %+v", tr)
	}
	// Clave descoñecida: fica como texto, non se perde.
	tr = parseInlineCrono("[fs=zzz]x[/fs]")
	if len(tr) == 0 || !strings.Contains(tr[0].Texto, "fs=zzz") {
		t.Errorf("clave descoñecida: %+v", tr)
	}
}

func TestTamanoImaxeCrono(t *testing.T) {
	col := 170.0
	if w, h := (cronoImaxeRef{AnchoPct: 50}).tamanoMM(col, 200, 100); w != 85 || h != 42.5 {
		t.Errorf("50%%: %v x %v", w, h)
	}
	if w, h := (cronoImaxeRef{}).tamanoMM(col, 4000, 100); w != col || h > 5 {
		t.Errorf("non debe pasar da columna: %v x %v", w, h)
	}
	if w, h := (cronoImaxeRef{}).tamanoMM(col, 100, 4000); h != 180 || w > 5 {
		t.Errorf("non debe pasar de 180 mm de alto: %v x %v", w, h)
	}
	if w, h := (cronoImaxeRef{}).tamanoMM(col, 0, 0); w != 0 || h != 0 {
		t.Errorf("sen dimensións: %v x %v", w, h)
	}
}

func TestCalendarioCrono(t *testing.T) {
	cfg := ConfigCrono{StartDate: "2025-09-08", EndDate: "2026-06-19"}
	datos := map[string]MesCrono{"2025-10": {Tasks: []TarefaCrono{{Title: "a"}}}}
	m := calendarioCrono(cfg, datos)
	if len(m) != 10 {
		t.Fatalf("meses = %d, esperaba 10", len(m))
	}
	if m[0].Nome != "Setembro" || m[0].Ano != 2025 || m[4].Nome != "Xaneiro" || m[4].Ano != 2026 || m[9].Nome != "Xuño" {
		t.Errorf("secuencia mal: %+v", m)
	}
	if len(m[1].Tarefas) != 1 {
		t.Errorf("as tarefas de outubro non chegaron: %+v", m[1])
	}
	if calendarioCrono(ConfigCrono{StartDate: "x", EndDate: "2026-06-19"}, nil) != nil {
		t.Errorf("datas inválidas deberían dar nil")
	}
	if calendarioCrono(ConfigCrono{StartDate: "2026-06-01", EndDate: "2025-01-01"}, nil) != nil {
		t.Errorf("fin anterior a inicio debería dar nil")
	}
}

func TestNomesMesesInformeCrono(t *testing.T) {
	// Un curso de máis de 12 meses repite nomes: entón vai co ano.
	cfg := ConfigCrono{StartDate: "2025-09-01", EndDate: "2026-10-30"}
	todos := calendarioCrono(cfg, nil)
	nomes := nomesMesesInformeCrono(todos)
	if nomes[0] != "Setembro 2025" || nomes[13] != "Outubro 2026" {
		t.Errorf("nomes = %v", nomes)
	}
	// Un curso normal non leva o ano.
	nomes = nomesMesesInformeCrono(calendarioCrono(ConfigCrono{StartDate: "2025-09-01", EndDate: "2026-06-30"}, nil))
	if nomes[0] != "Setembro" {
		t.Errorf("nomes = %v", nomes)
	}
}

func TestValidarDatasCrono(t *testing.T) {
	casos := []struct {
		a, b string
		erro bool
	}{
		{"2025-09-08", "2026-06-19", false},
		{"2025-09-08", "2025-09-08", false},
		{"2025-9-8x", "2026-06-19", true},
		{"2025-09-08", "", true},
		{"2026-06-19", "2025-09-08", true},
	}
	for _, c := range casos {
		if got := validarDatasCrono(c.a, c.b) != ""; got != c.erro {
			t.Errorf("validarDatasCrono(%q,%q): erro=%v, esperaba %v", c.a, c.b, got, c.erro)
		}
	}
}

func TestResolverCorCrono(t *testing.T) {
	if resolverCorCrono("rose").Clave != "rose" {
		t.Error("rose")
	}
	if resolverCorCrono("bg-emerald-50 text-emerald-800 border-emerald-200").Clave != "emerald" {
		t.Error("formato herdado da versión web")
	}
	if resolverCorCrono("").Clave != "blue" || resolverCorCrono("nada").Clave != "blue" {
		t.Error("por defecto debe ser a primeira da paleta")
	}
}

// pngDataURI xera un PNG sólido dun tamaño dado, coma data URI.
func pngDataURI(w, h int, c color.NRGBA) string {
	im := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			im.SetNRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, im)
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

// planMostraCrono é un curso realista que usa case todas as funcións.
func planMostraCrono() PlanCrono {
	return PlanCrono{
		ID:           "1",
		AcademicYear: "2025/2026",
		Config: ConfigCrono{
			StartDate:        "2025-09-08",
			EndDate:          "2026-06-19",
			SchoolName:       "IES Castelao & Fillos, 100% público",
			Location:         "Vigo",
			Department:       "Departamento de Orientación",
			HeadOfDepartment: "María Pérez Souto",
			SchoolLogo:       pngDataURI(300, 150, color.NRGBA{R: 30, G: 64, B: 175, A: 255}),
		},
		MonthsData: map[string]MesCrono{
			"2025-09": {Year: 2025, Month: 8, Tasks: []TarefaCrono{
				{ID: "a", Title: "Avaliación inicial", Color: "blue",
					Description: "# Obxectivos\n\nRealizar a **avaliación inicial** do alumnado de 1º de ESO e detectar _necesidades_ de apoio.\n\n- Recollida de informes\n- Reunión con titores\n- Entrevistas con familias\n\n1. Primeiro paso\n2. Segundo paso\n\n> A orientación é un proceso continuo.\n\n:::centro\nTexto centrado con [fs=gr]tamaño grande[/fs]\n:::\n![cor](img:i1 =40%)\n\nMáis info en [Xunta](https://www.edu.xunta.gal/portal/?a=1&b=2#x).",
					Images:      map[string]string{"i1": pngDataURI(400, 200, color.NRGBA{R: 220, G: 38, B: 38, A: 255})}},
				{ID: "b", Title: "Reunión co equipo docente para a coordinación de titorías e plan de acción titorial do curso", Color: "emerald", Description: ""},
				{ID: "c", Title: "Plan de acción titorial", Color: "amber", Description: "Sen máis."},
			}},
			"2025-10": {Year: 2025, Month: 9, Tasks: []TarefaCrono{
				{ID: "d", Title: "Programa de acollida", Color: "rose", Description: "Símbolos: ≤ ≥ ± → e €. Caracteres $ # _ ^ ~ { } \\ % &."},
			}},
			"2026-03": {Year: 2026, Month: 2, Tasks: []TarefaCrono{
				{ID: "e", Title: "Orientación académica e profesional", Color: "purple", Description: "## Subtítulo\n\nTexto."},
				{ID: "f", Title: "", Color: "stone", Description: ""},
			}},
		},
	}
}

func TestLatexInformeCronoContido(t *testing.T) {
	dir := t.TempDir()
	tex, avisos, err := latexInformeCrono(planMostraCrono(), "pdflatex", nil, dir, time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC), nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(avisos) != 0 {
		t.Errorf("avisos inesperados: %v", avisos)
	}
	for _, quero := range []string{
		`\section{Introdución}`, `\section{Cronograma anual}`, `\section{Desenvolvemento por meses}`,
		`\subsection{Setembro}`, `\subsection{Outubro}`, `\subsection{Marzo}`,
		`\chip{blue}{Avaliación inicial}`, `\chip{stone}{Tarefa sen título}`,
		`IES Castelao \& Fillos, 100\% público`,
		`{Vigo}`, `{18 de setembro de 2026}`,
		`\begin{enumerate}`, `\begin{itemize}`, `\begin{quote}`,
		`\href{https://www.edu.xunta.gal/portal/?a=1&b=2\#x}`,
		`María Pérez Souto`,
		`Xefe/a de Departamento de Orientación`,
	} {
		if !strings.Contains(tex, quero) {
			t.Errorf("o .tex non contén %q", quero)
		}
	}
	// Só os meses con tarefas van ao desenvolvemento (Novembro non).
	if strings.Contains(tex, `\subsection{Novembro}`) {
		t.Errorf("un mes sen tarefas non debe ter subsección")
	}
	// Pero o cronograma anual leva todos os meses do rango.
	if n := strings.Count(tex, `\mesbadge{`) + strings.Count(tex, `\mesvacio{`); n != 10 {
		t.Errorf("filas do cronograma anual = %d, esperaba 10", n)
	}
	// O logotipo e a imaxe da tarefa quedan en disco.
	imaxes, _ := filepath.Glob(filepath.Join(dir, "images", "img-*.png"))
	if len(imaxes) != 2 {
		t.Errorf("imaxes escritas = %d, esperaba 2 (logotipo + a da tarefa)", len(imaxes))
	}
	// Unha imaxe nunca leva dimensións en 0.
	if strings.Contains(tex, "width=0.0mm") || strings.Contains(tex, "height=0.0mm") {
		t.Errorf("imaxe con dimensións nulas")
	}
}

func TestLatexInformeCronoImaxeQueFalta(t *testing.T) {
	p := planMostraCrono()
	p.MonthsData["2025-10"] = MesCrono{Tasks: []TarefaCrono{{Title: "x", Description: "![a](img:non-existe)\n\nTexto"}}}
	p.Config.SchoolLogo = "data:image/webp;base64,AAAA"
	tex, avisos, err := latexInformeCrono(p, "pdflatex", nil, t.TempDir(), time.Now(), nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(avisos) < 2 {
		t.Errorf("esperaba avisos pola imaxe inexistente e polo logotipo inválido: %v", avisos)
	}
	if !strings.Contains(tex, "Texto") {
		t.Errorf("a descrición debe seguir aparecendo aínda que falte a imaxe")
	}
}

func TestMarkdownInformeCrono(t *testing.T) {
	dir := t.TempDir()
	md, avisos := markdownInformeCrono(planMostraCrono(), "docx", dir, time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC), nil)
	if len(avisos) != 0 {
		t.Errorf("avisos: %v", avisos)
	}
	for _, quero := range []string{
		"# CRONOGRAMA ANUAL", "# 1. Introdución", "# 2. Cronograma anual", "## 3.1. Setembro",
		"### Avaliación inicial", "#### Obxectivos", "- Recollida de informes", "> A orientación",
		"![Cronograma anual](images/cronograma.png)",
		`{=openxml}`, "Vigo, 18 de setembro de 2026",
	} {
		if !strings.Contains(md, quero) {
			t.Errorf("o Markdown non contén %q", quero)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "images", "cronograma.png")); err != nil {
		t.Errorf("falta a imaxe do cronograma: %v", err)
	}
	// O cronograma é só a imaxe: nin a frase nin a táboa de texto.
	if strings.Contains(md, "Distribución temporal") || strings.Contains(md, "| Mes |") {
		t.Errorf("o Markdown do cronograma debe levar só a imaxe")
	}
	if !regexp.MustCompile(`!\[Logotipo\]\(images/img-[0-9a-f]+\.png\)\{width=45mm\}`).MatchString(md) {
		t.Errorf("falta o logotipo")
	}
	// En odt/md non hai raw openxml.
	odt, _ := markdownInformeCrono(planMostraCrono(), "odt", t.TempDir(), time.Now(), nil)
	if strings.Contains(odt, "openxml") {
		t.Errorf("o formato odt non debe levar raw openxml")
	}
}

func TestPlanCronoJSONCompatible(t *testing.T) {
	// Formato exacto que escribe Piztu Orienta (cronograma/internal/model).
	orixinal := `{"plans":[{"id":"1","academicYear":"2025/2026","isLocked":false,
	 "config":{"startDate":"2025-09-08","endDate":"2026-06-19","schoolName":"IES","location":"Vigo",
	 "headOfDepartment":"X","schoolLogo":"","introText":"","signatoryRole":""},
	 "monthsData":{"2025-09":{"year":2025,"month":8,"tasks":[{"id":"t","title":"T","description":"d","color":"bg-blue-50 text-blue-800"}]}}}]}`
	var db struct{ Plans []PlanCrono }
	if err := json.Unmarshal([]byte(orixinal), &db); err != nil {
		t.Fatal(err)
	}
	p := db.Plans[0]
	if p.AcademicYear != "2025/2026" || p.MonthsData["2025-09"].Tasks[0].Title != "T" || p.Config.SchoolName != "IES" {
		t.Errorf("non se leu ben: %+v", p)
	}
	if resolverCorCrono(p.MonthsData["2025-09"].Tasks[0].Color).Clave != "blue" {
		t.Errorf("cor herdada")
	}
}

// compilarMostra compila o curso de mostra co motor dado e devolve o PDF.
func compilarMostra(t *testing.T, motor string) []byte {
	t.Helper()
	if !commandExists(motor) {
		t.Skipf("%s non está instalado", motor)
	}
	dir := t.TempDir()
	// Unha plantilla que fixa o motor: sen ela, un pdflatex pedido sobe a
	// xelatex para usar Open Sans no cronograma (ver TestCompilarSobeAXelatex).
	fixo := &PlantillaLatex{Nome: "motor fixo", Latex: "{{CORPO}}", Motor: motor}
	res, err := compilarInformeCrono(planMostraCrono(), motor, nil, dir, fixo, "")
	if err != nil {
		t.Fatalf("compilar con %s: %v\n--- log ---\n%s", motor, err, res.Log)
	}
	if res.Motor != motor {
		t.Fatalf("compilouse con %s en vez de %s (avisos: %v)", res.Motor, motor, res.Avisos)
	}
	if !bytes.HasPrefix(res.PDF, []byte("%PDF")) {
		t.Fatalf("a saída non é un PDF")
	}
	if saida := os.Getenv("YANG_INFORME_OUT"); saida != "" {
		_ = os.MkdirAll(saida, 0o755)
		_ = os.WriteFile(filepath.Join(saida, "informe-"+motor+".pdf"), res.PDF, 0o644)
		_ = os.WriteFile(filepath.Join(saida, "informe-"+motor+".tex"), []byte(res.Tex), 0o644)
	}
	return res.PDF
}

func numeroPaxinas(t *testing.T, pdf []byte) int {
	t.Helper()
	if !commandExists("pdfinfo") {
		t.Skip("pdfinfo non está instalado")
	}
	f := filepath.Join(t.TempDir(), "x.pdf")
	_ = os.WriteFile(f, pdf, 0o644)
	out, err := exec.Command("pdfinfo", f).Output()
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`Pages:\s+(\d+)`).FindSubmatch(out)
	if m == nil {
		t.Fatalf("pdfinfo sen Pages: %s", out)
	}
	n := 0
	for _, c := range m[1] {
		n = n*10 + int(c-'0')
	}
	return n
}

func TestCompilarInformeCronoPdflatex(t *testing.T) {
	pdf := compilarMostra(t, "pdflatex")
	// portada + índice + introdución + cronograma + desenvolvemento (a sinatura vai ao final)
	if n := numeroPaxinas(t, pdf); n < 6 {
		t.Errorf("páxinas = %d, esperaba polo menos 6", n)
	}
}

func TestCompilarInformeCronoXelatex(t *testing.T) {
	pdf := compilarMostra(t, "xelatex")
	if n := numeroPaxinas(t, pdf); n < 6 {
		t.Errorf("páxinas = %d, esperaba polo menos 6", n)
	}
}

// Sen motor fixado pola plantilla, o informe usa xelatex (Open Sans embebida
// no cronograma) aínda que se pida pdflatex, e leva a fonte no PDF.
func TestCompilarSobeAXelatex(t *testing.T) {
	if !commandExists("xelatex") {
		t.Skip("xelatex non está instalado")
	}
	res, err := compilarInformeCrono(planMostraCrono(), "pdflatex", nil, t.TempDir(), nil, "")
	if err != nil {
		t.Fatalf("compilar: %v\n%s", err, res.Log)
	}
	if res.Motor != "xelatex" {
		t.Errorf("motor = %s, esperaba xelatex", res.Motor)
	}
	if !strings.Contains(res.Tex, "OpenSans-Regular") {
		t.Errorf("o .tex non usa Open Sans")
	}
	if !commandExists("pdffonts") {
		return
	}
	f := filepath.Join(t.TempDir(), "x.pdf")
	_ = os.WriteFile(f, res.PDF, 0o644)
	if out, _ := exec.Command("pdffonts", f).Output(); !strings.Contains(string(out), "OpenSans") {
		t.Errorf("o PDF non leva Open Sans:\n%s", out)
	}
}

func TestCompilarInformeCronoDatasInvalidas(t *testing.T) {
	p := planMostraCrono()
	p.Config.EndDate = "2020-01-01"
	_, err := compilarInformeCrono(p, "pdflatex", nil, t.TempDir(), nil, "")
	if err == nil || !strings.Contains(err.Error(), "fin") {
		t.Errorf("esperaba un erro de datas, deu %v", err)
	}
}

func TestCompilarInformeCronoBaleiro(t *testing.T) {
	// Un curso recén creado (sen tarefas nin logotipo) tamén ten que compilar.
	if !commandExists("pdflatex") {
		t.Skip("pdflatex non está instalado")
	}
	p := PlanCrono{AcademicYear: "2025/2026", Config: ConfigCrono{StartDate: "2025-09-08", EndDate: "2026-06-19"}, MonthsData: map[string]MesCrono{}}
	res, err := compilarInformeCrono(p, "pdflatex", nil, t.TempDir(), nil, "")
	if err != nil {
		t.Fatalf("%v\n%s", err, res.Log)
	}
}

func TestExportarInformeOfimatica(t *testing.T) {
	dir := t.TempDir()
	for _, formato := range []string{"docx", "odt"} {
		saida := filepath.Join(dir, "informe."+formato)
		res, err := exportarInformeOfimaticaCrono(XerarInformeCronogramaRequest{Plan: planMostraCrono()}, nil, saida, formato)
		if err != nil {
			t.Fatalf("%s: %v", formato, err)
		}
		if info, err := os.Stat(res.Path); err != nil || info.Size() < 5000 {
			t.Errorf("%s: ficheiro ausente ou demasiado pequeno (%v)", formato, err)
		}
		if s := os.Getenv("YANG_INFORME_OUT"); s != "" {
			data, _ := os.ReadFile(saida)
			_ = os.WriteFile(filepath.Join(s, "informe."+formato), data, 0o644)
		}
	}
}

// textoPlanoCrono xunta os tramos e marca negrita/cursiva con [b] [i] para
// poder comparar o resultado do parser dunha ollada.
func textoMarcadoCrono(tr []cronoTramo) string {
	var b strings.Builder
	for _, x := range tr {
		p := x.Texto
		if x.Negrita {
			p = "[b]" + p + "[/b]"
		}
		if x.Cursiva {
			p = "[i]" + p + "[/i]"
		}
		b.WriteString(p)
	}
	return b.String()
}

func TestParseInlineEnxeCrono(t *testing.T) {
	casos := []struct{ entrada, esperado string }{
		{"un **negro** e *cursivo*", "un [b]negro[/b] e [i]cursivo[/i]"},
		{"un __negro__ e _cursivo_", "un [b]negro[/b] e [i]cursivo[/i]"},
		{"**a *b* c**", "[b]a [/b][i][b]b[/b][/i][b] c[/b]"},
		// Marcas sen par: literais, sen poñer en cursiva o resto do parágrafo.
		{"caracteres _ ^ ~ e mais", "caracteres _ ^ ~ e mais"},
		{"2 * 3 = 6", "2 * 3 = 6"},
		{"un **negro sen pechar", "un **negro sen pechar"},
		// "_" dentro dunha palabra é literal (nomes de ficheiro, 1_2...).
		{"ver nome_ficheiro_final.txt", "ver nome_ficheiro_final.txt"},
		{"ver _cursiva_ e nome_x", "ver [i]cursiva[/i] e nome_x"},
		// "*" dentro dunha palabra si pode marcar énfase (Markdown normal).
		{"in*tra*palabra", "in[i]tra[/i]palabra"},
	}
	for _, c := range casos {
		if got := textoMarcadoCrono(parseInlineCrono(c.entrada)); got != c.esperado {
			t.Errorf("parseInlineCrono(%q)\n  = %q\n  esperaba %q", c.entrada, got, c.esperado)
		}
	}
}

// Sen nome de departamento non se inventa «Departamento» nin «Xefe/a de Departamento»: queda en branco.
func TestDepartamentoBaleiroQuedaEnBranco(t *testing.T) {
	p := PlanCrono{}
	if departamentoCrono(p) != "" || cargoSinaturaCrono(p) != "" {
		t.Errorf("departamento %q, cargo %q", departamentoCrono(p), cargoSinaturaCrono(p))
	}
	p.Config.Department = "Departamento de Inglés"
	if cargoSinaturaCrono(p) != "Xefe/a de Departamento de Inglés" {
		t.Errorf("cargo co departamento: %q", cargoSinaturaCrono(p))
	}
	p.Config.SignatoryRole = "Orientadora"
	if cargoSinaturaCrono(p) != "Orientadora" {
		t.Error("o cargo escrito gaña")
	}
}
