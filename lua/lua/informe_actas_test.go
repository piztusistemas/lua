package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var agoraActas = time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC)

// planMostraActas é un curso con dúas actas (outubro e novembro), tres materias
// e informes de seguimento soltos.
func planMostraActas() PlanActas {
	p := planDemoActas()
	m := p.MonthsData["2025-10"]
	nov := m
	a2 := *m.Acta
	a2.ID, a2.Date, a2.EndTime = "a2", "2025-11-20", ""
	a2.Present = []string{"m1", "m2", "m3"}
	nov.Acta = &a2
	nov.Reports = nil
	p.MonthsData["2025-11"] = nov
	// Un mes cun informe pero sen acta (non debe saír).
	p.MonthsData["2025-09"] = MesActas{Year: 2025, Month: 8, Reports: map[string]SeguimentoActas{"s2": {Contents: "Intro"}}}
	p.Config.SchoolName = "IES Castelao & Fillos, 100% público"
	return p
}

func TestActasResoltas(t *testing.T) {
	p := planMostraActas()
	todas := actasResoltas(p, "")
	if len(todas) != 2 || todas[0].Num != 1 || todas[1].Num != 2 || todas[0].MesNome != "Outubro" {
		t.Fatalf("actas = %+v", todas)
	}
	if len(mesesConActa(p)) != 2 {
		t.Errorf("meses con acta = %v", mesesConActa(p))
	}
	// Ámbito dun mes: só esa acta, pero co número que lle toca no curso.
	nov := actasResoltas(p, "2025-11")
	if len(nov) != 1 || nov[0].Num != 2 {
		t.Fatalf("acta de novembro = %+v", nov)
	}
	if len(actasResoltas(p, "2025-09")) != 0 {
		t.Error("un mes sen acta non debe dar ningunha")
	}
	a := todas[0]
	if a.Data != "16 de outubro de 2025" || a.HoraInicio != "16:00" {
		t.Errorf("data/hora: %+v", a)
	}
	if len(a.Asistentes) != 2 || len(a.Ausentes) != 1 || !strings.Contains(a.Ausentes[0], "Xoán") {
		t.Errorf("asistentes %v ausentes %v", a.Asistentes, a.Ausentes)
	}
	if len(a.Materias) != 2 || !a.Materias[0].TenDatos || a.Materias[1].TenDatos {
		t.Errorf("materias = %+v", a.Materias)
	}
	if a.Materias[0].Docente != "Ana Pérez" || a.Materias[0].Grupos != "1º ESO" {
		t.Errorf("a materia debe herdar docente e grupos: %+v", a.Materias[0])
	}
	// O informe do mes pode cambiar o docente da materia.
	m := p.MonthsData["2025-10"]
	m.Reports["s1"] = SeguimentoActas{Teacher: "Substituta", Contents: "x"}
	p.MonthsData["2025-10"] = m
	if d := actasResoltas(p, "2025-10")[0].Materias[0].Docente; d != "Substituta" {
		t.Errorf("docente do informe = %q", d)
	}
}

func TestPlanActasJSON(t *testing.T) {
	p := planMostraActas()
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var q PlanActas
	if err := json.Unmarshal(b, &q); err != nil {
		t.Fatal(err)
	}
	if q.MonthsData["2025-10"].Acta == nil || *q.MonthsData["2025-10"].Reports["s1"].Progress != 80 || len(q.Config.Subjects) != 2 {
		t.Errorf("ida e volta: %+v", q)
	}
	// O progreso ausente non se confunde con 0.
	if strings.Contains(string(b), `"progress":0`) {
		t.Errorf("progreso inventado: %s", b)
	}
}

func TestLatexInformeActasCurso(t *testing.T) {
	dir := t.TempDir()
	tex, avisos, err := latexInformeActas(planMostraActas(), "curso", "pdflatex", nil, dir, agoraActas, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(avisos) != 0 {
		t.Errorf("avisos: %v", avisos)
	}
	for _, quero := range []string{
		`\tableofcontents`, `\renewcommand{\contentsname}{Actas do curso 2025-2026}`,
		`\addcontentsline{toc}{section}{Acta n.º 1 · outubro de 2025}`,
		`\actacab{Departamento de exemplo}{Acta n.º 1}{Reunión ordinaria de outubro de 2025}`,
		`\actacab{Departamento de exemplo}{Acta n.º 2}{Reunión ordinaria de novembro de 2025}`,
		`\actadato{Data}{16 de outubro de 2025}`, `\actadato{Hora}{16:00 – 17:15}`,
		`\actasecc{Asistentes}`, `\item Ana Pérez (Xefatura)`, `\actadato{Ausentes}{Xoán Rial}`,
		`\actasecc{Orde do día}`, `\actasecc{Seguimento das programacións didácticas}`,
		`\actamateria{Materia A}{Ana Pérez · 1º ESO}`, `\actaprogreso{Grao de cumprimento da programación}{80}`,
		`\actacampo{Contidos traballados}`, `\actamateria{Materia B}`, `Sen informe neste mes.`,
		`\actasecc{Acordos}`, `\actasecc{Rogos e preguntas}`,
		`levántase a sesión ás 17:15 do 16 de outubro de 2025`,
		`\actasinaturas{O/A secretario/a}{Nome e apelidos}{Vº e Prace, Xefe/a de Departamento de exemplo}{Nome e apelidos}`,
		`IES Castelao`, `Actas do departamento`,
	} {
		if !strings.Contains(tex, quero) {
			t.Errorf("o .tex non contén %q", quero)
		}
	}
	if n := strings.Count(tex, `\actacab{`); n != 2 {
		t.Errorf("actas = %d, esperaba 2 (a de setembro non ten acta)", n)
	}
	// A segunda acta non ten hora de fin: sen frase de peche.
	if strings.Count(tex, "levántase a sesión") != 1 {
		t.Errorf("a frase de peche só vai nas actas con hora de fin")
	}
	// Só o curso enteiro leva portada: hai portada (tikz de páxina) e índice.
	if !strings.Contains(tex, `current page.north west`) {
		t.Errorf("o curso enteiro leva portada")
	}
}

func TestLatexInformeActaSoa(t *testing.T) {
	tex, _, err := latexInformeActas(planMostraActas(), "2025-11", "pdflatex", nil, t.TempDir(), agoraActas, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(tex, `\actacab{`) != 1 || !strings.Contains(tex, `{Acta n.º 2}`) {
		t.Errorf("debe levar só a acta 2")
	}
	for _, non := range []string{`\tableofcontents`, `current page.north west`, `\addcontentsline`, `Outubro`} {
		if strings.Contains(tex, non) {
			t.Errorf("unha acta soa non leva %q", non)
		}
	}
}

// Sen ningunha acta o documento compila igual, cun aviso no seu sitio.
func TestLatexInformeActasSenActas(t *testing.T) {
	p := planMostraActas()
	p.MonthsData = map[string]MesActas{}
	tex, _, err := latexInformeActas(p, "curso", "pdflatex", nil, t.TempDir(), agoraActas, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(tex, `Aínda non hai ningunha acta neste curso.`) {
		t.Errorf("falta o texto de curso sen actas")
	}
}

// Os textos fixos e o estilo vense afectados pola plantilla: nada está escrito a lume.
func TestTextosEEstiloDaPlantilla(t *testing.T) {
	pl := &PlantillaLatex{
		Nome: "Propia", Latex: "{{CORPO}}",
		Estilo: "% estilo propio\n\\newcommand{\\actacab}[3]{CAB #1|#2|#3}\n",
		Textos: map[string]string{
			"ACTA_TITULO": "Sesión {{NUM}}", "ACTA_ORDE": "Puntos a tratar", "SEN_TAREFAS": "Libre",
			"CRONOGRAMA": "Calendario", "TRIMESTRE": "T{{N}}", "TITULO_ACTAS": "Libro de actas",
		},
	}
	tex, _, err := latexInformeActas(planMostraActas(), "curso", "pdflatex", nil, t.TempDir(), agoraActas, pl, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, quero := range []string{`{Sesión 1}`, `\actasecc{Puntos a tratar}`, `% estilo propio`} {
		if !strings.Contains(tex, quero) {
			t.Errorf("falta %q", quero)
		}
	}
	// O estilo da plantilla substitúe ao de Lua (non queda o \newcommand{\actacab} de Lua).
	if strings.Contains(tex, `\newcommand{\actacab}[3]{\par\noindent`) {
		t.Errorf("o estilo de Lua non debía ir se a plantilla ten o seu")
	}
	// O mesmo mecanismo vale para o cronograma.
	tc, _, err := latexInformeCrono(planMostraCrono(), "pdflatex", nil, t.TempDir(), agoraActas, pl, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, quero := range []string{`\section{Calendario}`, `\begin{trimestre}{T1}`, `Libre`} {
		if !strings.Contains(tc, quero) {
			t.Errorf("cronograma: falta %q", quero)
		}
	}
	if strings.Contains(tc, `\section{Cronograma anual}`) {
		t.Errorf("o título da sección debía cambiar")
	}
}

func TestNormalizarTextosDoc(t *testing.T) {
	n := normalizarTextosDoc(map[string]string{"indice": " Contido ", "INEXISTENTE": "x", "SEN_TITULO": "  "})
	if len(n) != 1 || n["INDICE"] != "Contido" {
		t.Errorf("normalizado = %v", n)
	}
	if normalizarTextosDoc(nil) != nil || normalizarTextosDoc(map[string]string{"X": "y"}) != nil {
		t.Errorf("sen textos válidos debe dar nil")
	}
	if !textosDocIguais(map[string]string{"INDICE": "a"}, map[string]string{"indice": "a", "XX": "b"}) {
		t.Errorf("deben ser iguais")
	}
	// Cada texto do catálogo ten clave única e valor por defecto (agás
	// FORM_SEN_CONTIDO: un formulario baleiro non pon ningún texto xenérico).
	vistos := map[string]bool{}
	for _, x := range textosDoc {
		if x.Clave == "" || (x.Defecto == "" && x.Clave != "FORM_SEN_CONTIDO") || vistos[x.Clave] {
			t.Errorf("texto mal definido: %+v", x)
		}
		vistos[x.Clave] = true
	}
}

func TestCompilarActasPdflatexEXelatex(t *testing.T) {
	for _, motor := range []string{"pdflatex", "xelatex"} {
		if !commandExists(motor) {
			continue
		}
		for _, ambito := range []string{"curso", "2025-10"} {
			t.Run(motor+"/"+ambito, func(t *testing.T) {
				res, err := compilarInformeActas(planMostraActas(), ambito, motor, nil, t.TempDir(), nil, "actas")
				if err != nil {
					t.Fatalf("%v\n%s", err, res.Log)
				}
				n := numeroPaxinas(t, res.PDF)
				if ambito == "curso" && n < 4 {
					t.Errorf("páxinas = %d (portada, índice e dúas actas)", n)
				}
				if ambito != "curso" && (n < 1 || n > 2) {
					t.Errorf("unha acta ocupa %d páxinas", n)
				}
			})
		}
	}
}

func TestCompilarActasDatasInvalidas(t *testing.T) {
	p := planMostraActas()
	p.Config.EndDate = "2020-01-01"
	if _, err := compilarInformeActas(p, "curso", "pdflatex", nil, t.TempDir(), nil, ""); err == nil {
		t.Error("as datas invertidas deben dar erro")
	}
}

func TestMarkdownInformeActas(t *testing.T) {
	md, _ := markdownInformeActas(planMostraActas(), "curso", "md", t.TempDir(), agoraActas, nil)
	for _, quero := range []string{
		"# ACTAS DO DEPARTAMENTO", "# Actas do curso 2025-2026", "1. Acta n.º 1 · outubro de 2025",
		"# Acta n.º 1", "**Data:** 16 de outubro de 2025", "## Asistentes", "- Ana Pérez (Xefatura)",
		"### Materia A", "**Grao de cumprimento da programación:** 80 %", "**Contidos traballados**",
		"*Sen informe neste mes.*", "## Acordos", "**O/A secretario/a**",
	} {
		if !strings.Contains(md, quero) {
			t.Errorf("o Markdown non contén %q", quero)
		}
	}
	soa, _ := markdownInformeActas(planMostraActas(), "2025-11", "md", t.TempDir(), agoraActas, nil)
	if strings.Contains(soa, "# Actas do curso") || strings.Contains(soa, "Acta n.º 1") || !strings.Contains(soa, "# Acta n.º 2") {
		t.Errorf("unha acta soa: %s", soa)
	}
}

func actasDe(t *testing.T, ambito, formato string, plantilla *PlantillaLatex) map[string]string {
	t.Helper()
	p := planMostraActas()
	datos, avisos, err := datosOfimaticaDe(XerarInformeCronogramaRequest{Ferramenta: ferramentaActas, Actas: &p, Ambito: ambito, NomeBase: "actas"}, plantilla, formato)
	if err != nil {
		t.Fatal(err)
	}
	if len(avisos) != 0 {
		t.Errorf("avisos: %v", avisos)
	}
	return lerZip(t, datos)
}

func TestDocxEOdtDeActas(t *testing.T) {
	pk := actasDe(t, "curso", "docx", nil)
	doc := pk["word/document.xml"]
	for _, quero := range []string{"Actas do curso 2025-2026", "Acta n.º 1", "Acta n.º 2", "Ana Pérez (Xefatura)", "Materia A", "80 %", "Sen informe neste mes.", "O/A secretario/a"} {
		if !strings.Contains(doc, quero) {
			t.Errorf("docx: falta %q", quero)
		}
	}
	odt := actasDe(t, "curso", "odt", nil)["content.xml"]
	for _, quero := range []string{"Actas do curso 2025-2026", "Acta n.º 2", "Materia A", "Sen informe neste mes.", "Vº e Prace"} {
		if !strings.Contains(odt, quero) {
			t.Errorf("odt: falta %q", quero)
		}
	}
	// Unha acta soa non leva o índice.
	soa := actasDe(t, "2025-10", "docx", nil)["word/document.xml"]
	if strings.Contains(soa, "Actas do curso") || strings.Contains(soa, "Acta n.º 2") || !strings.Contains(soa, "Acta n.º 1") {
		t.Errorf("docx dunha acta soa incorrecto")
	}
}

func TestExportarActasEscribeFicheiros(t *testing.T) {
	p := planMostraActas()
	req := XerarInformeCronogramaRequest{Ferramenta: ferramentaActas, Actas: &p, Ambito: "curso", NomeBase: "actas"}
	for _, formato := range []string{"docx", "odt"} {
		res, err := exportarInformeOfimaticaCrono(req, nil, filepath.Join(t.TempDir(), "a."+formato), formato)
		if err != nil {
			t.Fatal(err)
		}
		if b, err := os.ReadFile(res.Path); err != nil || len(b) < 1000 {
			t.Errorf("%s non escrito: %v", formato, err)
		}
	}
}

func TestBorradorPorFerramenta(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	a := &App{}
	if b, _ := a.LerBorradorInforme(ferramentaActas); b != "" {
		t.Fatal("non debía haber borrador")
	}
	if err := a.GardarBorradorInforme(ferramentaActas, `{"ferramenta":"actas"}`); err != nil {
		t.Fatal(err)
	}
	if b, _ := a.LerBorradorInforme(ferramentaActas); b != `{"ferramenta":"actas"}` {
		t.Errorf("borrador de actas = %q", b)
	}
	// O cronograma ten o seu propio borrador.
	if b, _ := a.LerBorradorInforme(ferramentaCronograma); b != "" {
		t.Errorf("o borrador do cronograma non debía verse: %q", b)
	}
}

// rutasDeBorradorDistinguenMaiusculas comproba se <config>/Lua e <config>/lua
// son dous cartafoles distintos (Linux) ou o mesmo (macOS/Windows). As probas
// de migración só teñen sentido no primeiro caso.
func rutasDeBorradorDistinguenMaiusculas(t *testing.T, cfg string) bool {
	t.Helper()
	dirVello := filepath.Join(cfg, "Lua", "informes")
	dirNovo := filepath.Join(cfg, "lua", "informes")
	if err := os.MkdirAll(dirVello, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dirNovo, 0o755); err != nil {
		t.Fatal(err)
	}
	fiVello, errVello := os.Stat(dirVello)
	fiNovo, errNovo := os.Stat(dirNovo)
	return errVello == nil && errNovo == nil && !os.SameFile(fiVello, fiNovo)
}

// TestBorradorMigraDaLocalizacionVella cobre o arranxo da ruta dos borradores:
// as versións anteriores gardaban en <config>/Lua/informes/ ("Lua" en
// maiúscula, cando o resto da configuración vive en <config>/lua/). Un
// borrador xa existente non pode desaparecer: lese do sitio vello e múdase ao
// novo.
func TestBorradorMigraDaLocalizacionVella(t *testing.T) {
	configIllado(t)
	cfg, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	if !rutasDeBorradorDistinguenMaiusculas(t, cfg) {
		t.Skip("sistema de ficheiros insensible a maiúsculas: as dúas rutas son o mesmo cartafol")
	}
	vello := filepath.Join(cfg, "Lua", "informes", "cronograma-borrador.json")
	if err := os.WriteFile(vello, []byte(`{"ferramenta":"cronograma"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	a := &App{}
	if b, _ := a.LerBorradorInforme(ferramentaCronograma); b != `{"ferramenta":"cronograma"}` {
		t.Fatalf("debía ler o borrador da localización vella, obtiven %q", b)
	}
	novo := filepath.Join(cfg, "lua", "informes", "cronograma-borrador.json")
	if _, err := os.Stat(novo); err != nil {
		t.Errorf("o borrador debía migrar a %s: %v", novo, err)
	}
	if _, err := os.Stat(vello); !os.IsNotExist(err) {
		t.Errorf("o borrador vello debería ter desaparecido ao migrar (err=%v)", err)
	}
}

// TestGardarBorradorLimpaOLocalizacionVella comproba que gardar no sitio novo
// retira o borrador histórico, para que non quede un ficheiro vello que
// reaparecería se algún día se borra o novo.
func TestGardarBorradorLimpaOLocalizacionVella(t *testing.T) {
	configIllado(t)
	cfg, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	if !rutasDeBorradorDistinguenMaiusculas(t, cfg) {
		t.Skip("sistema de ficheiros insensible a maiúsculas: as dúas rutas son o mesmo cartafol")
	}
	vello := filepath.Join(cfg, "Lua", "informes", "actas-borrador.json")
	if err := os.WriteFile(vello, []byte(`{"ferramenta":"actas","vello":true}`), 0o644); err != nil {
		t.Fatal(err)
	}

	a := &App{}
	if err := a.GardarBorradorInforme(ferramentaActas, `{"ferramenta":"actas","vello":false}`); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(vello); !os.IsNotExist(err) {
		t.Errorf("o borrador vello debería terse retirado ao gardar (err=%v)", err)
	}
	if b, _ := a.LerBorradorInforme(ferramentaActas); b != `{"ferramenta":"actas","vello":false}` {
		t.Errorf("debería lerse o borrador novo, obtiven %q", b)
	}
}

func TestPrevisualizarPlantillaDeActas(t *testing.T) {
	if !commandExists("xelatex") && !commandExists("pdflatex") {
		t.Skip("sen LaTeX")
	}
	a := &App{}
	r, err := a.PrevisualizarPlantillaDe(PlantillaLatex{Nome: "p", Latex: "{{CORPO}}", Textos: map[string]string{"ACTA_TITULO": "Sesión especial"}}, ferramentaActas)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r.LatexSource, "Sesión especial") || strings.Contains(r.LatexSource, "\\tarefa{") {
		t.Errorf("a vista previa debía ser dun documento de actas")
	}
}
