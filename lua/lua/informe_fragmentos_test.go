package main

import (
	"strings"
	"testing"
	"time"
)

var agoraFragmentos = time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC)

func texInformeCon(t *testing.T, plantilla *PlantillaLatex) (string, []string) {
	t.Helper()
	tex, avisos, err := latexInformeCrono(planMostraCrono(), "pdflatex", nil, t.TempDir(), agoraFragmentos, plantilla, "")
	if err != nil {
		t.Fatal(err)
	}
	return tex, avisos
}

// Sen plantilla, ou cunha "clásica" (sen {{PORTADA}} nin {{SINATURA}}), o
// informe leva a portada e a sinatura por defecto de Lua.
func TestPortadaSinaturaPorDefecto(t *testing.T) {
	clasica := &PlantillaLatex{Nome: "Clásica", Latex: "\\pagestyle{fancy}\n{{CORPO}}\n"}
	for nome, p := range map[string]*PlantillaLatex{"sen plantilla": nil, "clásica": clasica} {
		tex, _ := texInformeCon(t, p)
		for _, quero := range []string{`Cronograma anual}`, `Curso {2025/2026}`, `\rule{58mm}{0.4pt}`, `María Pérez Souto`} {
			if !strings.Contains(tex, quero) {
				t.Errorf("%s: o .tex non contén %q", nome, quero)
			}
		}
		if strings.Index(tex, `Curso {2025/2026}`) > strings.Index(tex, `\tableofcontents`) {
			t.Errorf("%s: a portada ten que ir antes do índice", nome)
		}
		if strings.Contains(tex, "Sitio para a sinatura") || strings.Contains(tex, `\section{Sinatura}`) {
			t.Errorf("%s: a sinatura non leva sección nin textos de mostra", nome)
		}
		if strings.LastIndex(tex, `\rule{58mm}{0.4pt}`) < strings.Index(tex, `\section{Desenvolvemento por meses}`) {
			t.Errorf("%s: a sinatura ten que ir despois do desenvolvemento", nome)
		}
	}
}

// Cos marcadores, a portada e a sinatura son as da plantilla, coas variables
// do curso e do centro enchidas; un marcador que falta quita esa parte.
func TestPortadaSinaturaDaPlantilla(t *testing.T) {
	p := &PlantillaLatex{
		Nome:     "Propia",
		Latex:    "{{PORTADA}}\n{{CORPO}}\n{{SINATURA}}\n",
		Portada:  `PORTADA-PROPIA {{CENTRO}} / {{CURSO}} / {{DEPARTAMENTO}} / {{XEFE}} / {{CARGO}} / {{DATA_LONGA}} \logocentro[height=1cm]` + "\n",
		Sinatura: `SINATURA-PROPIA {{XEFE}} {{CARGO}}` + "\n",
	}
	tex, avisos := texInformeCon(t, p)
	if len(avisos) != 0 {
		t.Errorf("avisos inesperados: %v", avisos)
	}
	for _, quero := range []string{
		`PORTADA-PROPIA IES Castelao \& Fillos, 100\% público / 2025/2026 / Departamento de Orientación / María Pérez Souto / Xefe/a de Departamento de Orientación / 18 de setembro de 2026`,
		`SINATURA-PROPIA María Pérez Souto Xefe/a de Departamento de Orientación`,
		`\newcommand{\logocentro}[1][]{\includegraphics[#1]{images/img-`,
	} {
		if !strings.Contains(tex, quero) {
			t.Errorf("o .tex non contén %q", quero)
		}
	}
	if strings.Contains(tex, `\rule{58mm}{0.4pt}`) || strings.Contains(tex, `Curso {2025/2026}`) {
		t.Errorf("con portada e sinatura propias non deben saír as de Lua")
	}
	if strings.Contains(tex, marcadorPortada) || strings.Contains(tex, marcadorSinatura) || strings.Contains(tex, marcadorCorpo) {
		t.Errorf("quedou algún marcador sen substituír")
	}

	// Sen {{SINATURA}}: o informe remata sen sinatura (e avisa se a plantilla
	// tiña unha escrita).
	p2 := *p
	p2.Latex = "{{PORTADA}}\n{{CORPO}}\n"
	tex2, avisos2 := texInformeCon(t, &p2)
	if strings.Contains(tex2, "SINATURA-PROPIA") || strings.Contains(tex2, `\rule{58mm}{0.4pt}`) {
		t.Errorf("sen {{SINATURA}} non debe haber sinatura")
	}
	if len(avisos2) != 1 || !strings.Contains(avisos2[0], marcadorSinatura) {
		t.Errorf("debía avisar de que a sinatura da plantilla non se usa: %v", avisos2)
	}

	// Fragmento baleiro cun marcador presente: vale o deseño por defecto.
	p3 := &PlantillaLatex{Nome: "Baleira", Latex: "{{PORTADA}}\n{{CORPO}}\n"}
	tex3, _ := texInformeCon(t, p3)
	if !strings.Contains(tex3, `Curso {2025/2026}`) {
		t.Errorf("unha portada baleira debía usar o deseño por defecto")
	}
}

// Unha variable da propia plantilla con valor gaña ao dato do curso tamén na
// portada (o mesmo criterio ca na cabeceira).
func TestPortadaVariableDaPlantillaGaña(t *testing.T) {
	p := &PlantillaLatex{
		Nome: "Fixa", Latex: "{{PORTADA}}\n{{CORPO}}\n", Portada: "DEP={{DEPARTAMENTO}}\n",
		Variables: []PlantillaVar{{Nome: "DEPARTAMENTO", Valor: "Departamento de Matemáticas"}},
	}
	tex, _ := texInformeCon(t, p)
	if !strings.Contains(tex, "DEP=Departamento de Matemáticas") {
		t.Errorf("a variable da plantilla debía gañar")
	}
}

func TestIntroducion(t *testing.T) {
	p := planMostraCrono()
	vars := variablesCorpoCrono(p, "", agoraFragmentos)

	// Por defecto: nomea o centro, o curso e o departamento.
	texto, avisos := introCrono(p, nil, vars)
	for _, quero := range []string{"Departamento de Orientación", "IES Castelao & Fillos, 100% público", "2025/2026"} {
		if !strings.Contains(texto, quero) {
			t.Errorf("a introdución por defecto non leva %q:\n%s", quero, texto)
		}
	}
	if strings.Contains(texto, "{{") || len(avisos) != 0 {
		t.Errorf("variables sen substituír ou avisos: %q %v", texto, avisos)
	}

	// A da plantilla substitúe a por defecto; a do curso gaña a ambas.
	pl := &PlantillaLatex{Introducion: "Intro de plantilla de {{CENTRO}}."}
	if texto, _ := introCrono(p, pl, vars); texto != "Intro de plantilla de IES Castelao & Fillos, 100% público." {
		t.Errorf("intro da plantilla: %q", texto)
	}
	p.Config.IntroText = "Intro do curso {{CURSO}}."
	if texto, _ := introCrono(p, pl, vars); texto != "Intro do curso 2025/2026." {
		t.Errorf("intro do curso: %q", texto)
	}

	// Un centro sen nome no curso non deixa un oco: sae o dato de mostra.
	p.Config.IntroText = ""
	p.Config.SchoolName = ""
	vars = variablesCorpoCrono(p, "", agoraFragmentos)
	if texto, _ := introCrono(p, nil, vars); !strings.Contains(texto, "Nome do centro") {
		t.Errorf("sen nome de centro debía saír 'Nome do centro': %q", texto)
	}
}

func TestVariablesNovasDoCurso(t *testing.T) {
	v := variablesAutomaticasCrono(planMostraCrono(), "", agoraFragmentos)
	for nome, esperado := range map[string]string{
		"DATA_LONGA": "18 de setembro de 2026", "XEFE": "María Pérez Souto", "ORIENTADOR": "María Pérez Souto",
		"DEPARTAMENTO": "Departamento de Orientación",
	} {
		if v[nome] != esperado {
			t.Errorf("%s = %q, esperaba %q", nome, v[nome], esperado)
		}
	}
	if v["DATA_INICIO"] == "" || v["DATA_FIN"] == "" {
		t.Errorf("DATA_INICIO/DATA_FIN sen valor: %q %q", v["DATA_INICIO"], v["DATA_FIN"])
	}
}

func TestTrimestresCrono(t *testing.T) {
	p := planMostraCrono() // setembro 2025 a xuño 2026
	meses := calendarioCrono(p.Config, p.MonthsData)
	g := trimestresCrono(meses)
	if len(g) != 3 {
		t.Fatalf("trimestres = %d, esperaba 3", len(g))
	}
	for i, esperado := range []struct {
		titulo, rango string
		n             int
	}{
		{"1.º trimestre", "Setembro – Decembro", 4},
		{"2.º trimestre", "Xaneiro – Marzo", 3},
		{"3.º trimestre", "Abril – Xuño", 3},
	} {
		if g[i].Titulo != esperado.titulo || g[i].Meses != esperado.rango || len(g[i].Mesos) != esperado.n {
			t.Errorf("trimestre %d = %q %q (%d meses), esperaba %q %q (%d)", i+1, g[i].Titulo, g[i].Meses, len(g[i].Mesos), esperado.titulo, esperado.rango, esperado.n)
		}
	}

	// Un curso que empeza en xaneiro non ten 1.º trimestre; un de máis de 12
	// meses volve abrir o 1.º cando chega outra vez setembro.
	p.Config.StartDate, p.Config.EndDate = "2026-01-10", "2026-04-30"
	if g := trimestresCrono(calendarioCrono(p.Config, p.MonthsData)); len(g) != 2 || g[0].Titulo != "2.º trimestre" || g[1].Meses != "Abril" {
		t.Errorf("curso de xaneiro a abril: %+v", g)
	}
	p.Config.StartDate, p.Config.EndDate = "2025-09-01", "2026-10-31"
	if g := trimestresCrono(calendarioCrono(p.Config, p.MonthsData)); len(g) != 4 || g[3].Titulo != "1.º trimestre" || g[3].Meses != "Setembro – Outubro" {
		t.Errorf("curso de máis dun ano: %+v", g)
	}
}

// O cronograma anual leva un bloque por trimestre: cada un é unha caixa que non
// se parte (ambiente trimestre) e a columna do mes mídese co texto dos meses.
// Debaixo do título da sección non hai ningún texto: só o cronograma.
func TestCronogramaAnualPorTrimestres(t *testing.T) {
	tex, _ := texInformeCon(t, nil)
	if n := strings.Count(tex, `\begin{trimestre}`); n != 3 {
		t.Errorf("trimestres = %d, esperaba 3", n)
	}
	if n := strings.Count(tex, `\begin{tabular}{@{}p{\meswidth}`); n != 3 {
		t.Errorf("táboas = %d, esperaba 3", n)
	}
	if strings.Contains(tex, `\begin{longtable}`) || strings.Contains(tex, "\\\\*\n\\hline") {
		t.Errorf("os trimestres non deben ir en longtable (pódese partir entre páxinas)")
	}
	if !strings.Contains(tex, `\begin{trimestre}{1.º trimestre}{Setembro – Decembro}`) {
		t.Errorf("falta a cabeceira do 1.º trimestre")
	}
	if !strings.Contains(tex, `\medirmes{Setembro}{3 tarefas}`) || !strings.Contains(tex, `p{\meswidth}`) {
		t.Errorf("a columna do mes debía medirse co texto dos meses")
	}
	if strings.Contains(tex, "Distribución temporal") {
		t.Errorf("a sección do cronograma non leva texto explicativo")
	}
	sec := tex[strings.Index(tex, `\section{Cronograma anual}`):]
	sec = sec[:strings.Index(sec, `\section{Desenvolvemento por meses}`)]
	antes := sec[len(`\section{Cronograma anual}`):strings.Index(sec, `\arrayrulecolor`)]
	if strings.TrimSpace(antes) != "" {
		t.Errorf("entre o título e o cronograma hai texto: %q", antes)
	}
}
