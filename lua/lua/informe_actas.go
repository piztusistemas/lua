package main

// Informe "Actas do departamento": a segunda ferramenta de Lua. Cada mes do
// curso ten unha acta mensual do departamento e, por cada materia que imparte,
// un informe de seguimento da programación; a acta recolle os datos desses
// informes. Pódese imprimir unha acta soa ou todas as do curso.
//
// Comparte co cronograma a biblioteca de plantillas (cabeceira, pé, portada,
// estilo e textos: plantillas.go), o compilador LaTeX (compilarComMotores) e as
// exportacións a Markdown, Word e OpenDocument. O contido é aquí; o deseño é
// da plantilla (macros \actacab, \actasecc... de estilo_doc.go) e os textos
// fixos, editables, están en textos_doc.go.
//
// O ficheiro do curso é un JSON propio ({"ferramenta":"actas","actas":[...]}),
// distinto do do cronograma para que un non se abra coma o outro.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ---------- Modelo ----------

// MateriaActas é unha materia que imparte o departamento.
type MateriaActas struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Teacher string `json:"teacher"` // docente que a imparte
	Groups  string `json:"groups"`  // grupos ou cursos ("1º ESO A, 1º ESO B")
}

// MembroActas é unha persoa do departamento (asiste ás reunións).
type MembroActas struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"` // opcional: "Xefatura", "Secretaría"...
}

// ConfigActas son os datos do departamento e do centro.
type ConfigActas struct {
	StartDate        string         `json:"startDate"`
	EndDate          string         `json:"endDate"`
	SchoolName       string         `json:"schoolName"`
	Location         string         `json:"location"`
	Department       string         `json:"department"`
	HeadOfDepartment string         `json:"headOfDepartment"`
	Secretary        string         `json:"secretary"` // quen redacta as actas
	SignatoryRole    string         `json:"signatoryRole"`
	SchoolLogo       string         `json:"schoolLogo"` // data URI
	MeetingTime      string         `json:"meetingTime"`
	MeetingPlace     string         `json:"meetingPlace"`
	Members          []MembroActas  `json:"members"`
	Subjects         []MateriaActas `json:"subjects"`
}

// SeguimentoActas é o informe de seguimento dunha materia nun mes: o que
// recolle a acta mensual.
type SeguimentoActas struct {
	Teacher      string `json:"teacher,omitempty"`  // baleiro = o da materia
	Groups       string `json:"groups,omitempty"`   // baleiro = os da materia
	Progress     *int   `json:"progress,omitempty"` // grao de cumprimento da programación, 0-100
	Contents     string `json:"contents,omitempty"`
	Results      string `json:"results,omitempty"`
	Difficulties string `json:"difficulties,omitempty"`
	Measures     string `json:"measures,omitempty"`
}

// baleiro di se o informe non ten ningún dato.
func (s SeguimentoActas) baleiro() bool {
	return s.Progress == nil && strings.TrimSpace(s.Contents+s.Results+s.Difficulties+s.Measures) == ""
}

// ActaActas é a acta dun mes. Os textos son Markdown (ver informe_md.go).
type ActaActas struct {
	ID          string   `json:"id"`
	Date        string   `json:"date"` // AAAA-MM-DD, pode ir baleira
	StartTime   string   `json:"startTime"`
	EndTime     string   `json:"endTime"`
	Place       string   `json:"place"`
	Present     []string `json:"present"` // IDs dos membros que asisten
	Agenda      string   `json:"agenda"`
	Development string   `json:"development"`
	Agreements  string   `json:"agreements"`
	Questions   string   `json:"questions"`
}

// MesActas agrupa o de cada mes: a súa acta (se se creou) e os informes de
// seguimento das materias (clave: ID da materia).
type MesActas struct {
	Year    int                        `json:"year"`
	Month   int                        `json:"month"`
	Acta    *ActaActas                 `json:"acta,omitempty"`
	Reports map[string]SeguimentoActas `json:"reports,omitempty"`
}

// PlanActas é un curso académico completo de actas.
type PlanActas struct {
	ID           string              `json:"id"`
	AcademicYear string              `json:"academicYear"`
	IsLocked     bool                `json:"isLocked"`
	Config       ConfigActas         `json:"config"`
	MonthsData   map[string]MesActas `json:"monthsData"` // clave "AAAA-MM"
}

const (
	ferramentaCronograma = "cronograma"
	ferramentaActas      = "actas"
	ambitoCursoActas     = "curso"
)

// comoCrono devolve os datos comúns co cronograma (centro, departamento, xefe,
// datas...) para reutilizar as variables e as funcións de calendario.
func (p PlanActas) comoCrono() PlanCrono {
	c := p.Config
	return PlanCrono{
		ID: p.ID, AcademicYear: p.AcademicYear, IsLocked: p.IsLocked,
		Config: ConfigCrono{
			StartDate: c.StartDate, EndDate: c.EndDate, SchoolName: c.SchoolName, Location: c.Location,
			Department: c.Department, HeadOfDepartment: c.HeadOfDepartment, SchoolLogo: c.SchoolLogo,
			SignatoryRole: c.SignatoryRole,
		},
	}
}

// variablesActas son as variables automáticas das actas: as do cronograma coa
// variable {{SECRETARIO}} máis e o título dos textos.
func variablesActas(p PlanActas, nomeDoc string, agora time.Time) map[string]string {
	v := variablesCorpoCrono(p.comoCrono(), nomeDoc, agora)
	v["TITULO"] = textosDocPorClave["TITULO_ACTAS"].Defecto
	v["SECRETARIO"] = porDefectoCrono(p.Config.Secretary, "Nome e apelidos")
	return v
}

// ---------- Resolución ----------

// materiaResolta é o seguimento dunha materia xa listo para amosar.
type materiaResolta struct {
	Nome, Docente, Grupos                       string
	Progreso                                    *int
	Contidos, Resultados, Dificultades, Medidas string
	TenDatos                                    bool
}

// actaResolta é unha acta xa cos nomes, as horas e as materias resoltas.
type actaResolta struct {
	Num                        int
	Clave                      string
	Ano, Mes                   int // Mes 0-11
	MesNome                    string
	Data                       string // "12 de novembro de 2025" ou ""
	HoraInicio, HoraFin, Lugar string
	Asistentes, Ausentes       []string
	Orde, Desenvolvemento      string
	Acordos, Rogos             string
	Materias                   []materiaResolta
}

// vars son as variables propias desta acta para os textos fixos.
func (a actaResolta) vars() map[string]string {
	return map[string]string{
		"NUM": strconv.Itoa(a.Num), "MES_NOME": strings.ToLower(a.MesNome), "ANO_ACTA": strconv.Itoa(a.Ano),
		"DATA_ACTA_LONGA": a.Data, "HORA_INICIO": a.HoraInicio, "HORA_FIN": a.HoraFin,
	}
}

func nomeMembroActas(m MembroActas) string {
	n := strings.TrimSpace(m.Name)
	if n == "" {
		n = "Sen nome"
	}
	if r := strings.TrimSpace(m.Role); r != "" {
		n += " (" + r + ")"
	}
	return n
}

// actasResoltas devolve as actas do curso (as dos meses que teñen unha), en
// orde. `ambito` é "curso" (ou baleiro) para todas ou "AAAA-MM" para unha soa;
// o número de cada acta conta sempre desde o principio do curso.
func actasResoltas(p PlanActas, ambito string) []actaResolta {
	meses := calendarioCrono(p.comoCrono().Config, nil)
	var out []actaResolta
	num := 0
	for _, m := range meses {
		md, ok := p.MonthsData[m.Clave]
		if !ok || md.Acta == nil {
			continue
		}
		num++
		if ambito != "" && ambito != ambitoCursoActas && ambito != m.Clave {
			continue
		}
		ac := *md.Acta
		r := actaResolta{
			Num: num, Clave: m.Clave, Ano: m.Ano, Mes: m.Mes, MesNome: m.Nome,
			HoraInicio: strings.TrimSpace(ac.StartTime), HoraFin: strings.TrimSpace(ac.EndTime),
			Lugar: strings.TrimSpace(ac.Place),
			Orde:  ac.Agenda, Desenvolvemento: ac.Development, Acordos: ac.Agreements, Rogos: ac.Questions,
		}
		if t, err := time.Parse("2006-01-02", strings.TrimSpace(ac.Date)); err == nil {
			r.Data = dataGalegaCrono(t)
		}
		presentes := map[string]bool{}
		for _, id := range ac.Present {
			presentes[id] = true
		}
		for _, mb := range p.Config.Members {
			if presentes[mb.ID] {
				r.Asistentes = append(r.Asistentes, nomeMembroActas(mb))
			} else {
				r.Ausentes = append(r.Ausentes, nomeMembroActas(mb))
			}
		}
		for _, mt := range p.Config.Subjects {
			s := md.Reports[mt.ID]
			mr := materiaResolta{
				Nome:     porDefectoCrono(mt.Name, "Materia sen nome"),
				Docente:  strings.TrimSpace(porDefectoCrono(s.Teacher, mt.Teacher)),
				Grupos:   strings.TrimSpace(porDefectoCrono(s.Groups, mt.Groups)),
				Progreso: s.Progress, Contidos: s.Contents, Resultados: s.Results,
				Dificultades: s.Difficulties, Medidas: s.Measures, TenDatos: !s.baleiro(),
			}
			r.Materias = append(r.Materias, mr)
		}
		out = append(out, r)
	}
	return out
}

func limitarPorcentaxe(n int) int {
	if n < 0 {
		return 0
	}
	if n > 100 {
		return 100
	}
	return n
}

// horasActa é o texto do intervalo de horas ("" se non hai ningunha).
func horasActa(tx textosDocumento, a actaResolta) string {
	switch {
	case a.HoraInicio != "" && a.HoraFin != "":
		return tx.T("ACTA_HORAS", a.vars())
	case a.HoraInicio != "":
		return a.HoraInicio
	}
	return a.HoraFin
}

// ---------- LaTeX ----------

func latexTextoActa(md string, tx textosDocumento, esc *escritorImaxesCrono) string {
	bloques := parseMarkdownCrono(md)
	if len(bloques) == 0 {
		return `{\itshape\color{gray} ` + tx.L("ACTA_SEN_CONTIDO") + `}` + "\n\n"
	}
	return latexBloquesCrono(bloques, nil, esc)
}

// latexCampoActa escribe un campo do seguimento dunha materia só se ten texto.
func latexCampoActa(b *strings.Builder, etiqueta, md string, esc *escritorImaxesCrono) {
	bloques := parseMarkdownCrono(md)
	if len(bloques) == 0 {
		return
	}
	b.WriteString(`\actacampo{` + etiqueta + `}{` + latexBloquesCrono(bloques, nil, esc) + "}\n")
}

// latexCorpoActa escribe o corpo dunha acta (sen o \clearpage inicial).
func latexCorpoActa(b *strings.Builder, p PlanActas, a actaResolta, tx textosDocumento, esc *escritorImaxesCrono) {
	v := a.vars()
	b.WriteString(`\actacab{` + tx.L("ACTA_KICKER", v) + `}{` + tx.L("ACTA_TITULO", v) + `}{` + tx.L("ACTA_SUBTITULO", v) + "}\n")
	if a.Data != "" {
		b.WriteString(`\actadato{` + tx.L("ACTA_DATA") + `}{` + EscapeLatexProse(a.Data) + "}\n")
	}
	if h := horasActa(tx, a); h != "" {
		b.WriteString(`\actadato{` + tx.L("ACTA_HORA") + `}{` + EscapeLatexProse(h) + "}\n")
	}
	if a.Lugar != "" {
		b.WriteString(`\actadato{` + tx.L("ACTA_LUGAR") + `}{` + EscapeLatexProse(a.Lugar) + "}\n")
	}

	if len(p.Config.Members) > 0 {
		b.WriteString(`\actasecc{` + tx.L("ACTA_ASISTENTES") + "}\n")
		if len(a.Asistentes) > 0 {
			b.WriteString("\\begin{itemize}[itemsep=0pt,topsep=1pt]\n")
			for _, n := range a.Asistentes {
				b.WriteString(`\item ` + EscapeLatexProse(n) + "\n")
			}
			b.WriteString("\\end{itemize}\n")
		}
		ausentes := tx.T("ACTA_SEN_AUSENTES")
		if len(a.Ausentes) > 0 {
			ausentes = strings.Join(a.Ausentes, "; ")
		}
		b.WriteString(`\actadato{` + tx.L("ACTA_AUSENTES") + `}{` + EscapeLatexProse(ausentes) + "}\n")
	}

	b.WriteString(`\actasecc{` + tx.L("ACTA_ORDE") + "}\n" + latexTextoActa(a.Orde, tx, esc))

	b.WriteString(`\actasecc{` + tx.L("ACTA_SEGUIMENTO") + "}\n")
	if len(a.Materias) == 0 {
		b.WriteString(`{\itshape\color{gray} ` + tx.L("ACTA_SEN_SEGUIMENTO") + `}` + "\n\n")
	}
	for _, m := range a.Materias {
		detalle := m.Docente
		if m.Grupos != "" {
			if detalle != "" {
				detalle += " · "
			}
			detalle += m.Grupos
		}
		b.WriteString(`\actamateria{` + EscapeLatexProse(m.Nome) + `}{` + EscapeLatexProse(detalle) + "}\n")
		if !m.TenDatos {
			b.WriteString(`{\itshape\color{gray} ` + tx.L("SEG_SEN_DATOS") + `}` + "\n\n")
			continue
		}
		if m.Progreso != nil {
			b.WriteString(`\actaprogreso{` + tx.L("SEG_CUMPRIMENTO") + `}{` + strconv.Itoa(limitarPorcentaxe(*m.Progreso)) + "}\n")
		}
		latexCampoActa(b, tx.L("SEG_CONTIDOS"), m.Contidos, esc)
		latexCampoActa(b, tx.L("SEG_RESULTADOS"), m.Resultados, esc)
		latexCampoActa(b, tx.L("SEG_DIFICULTADES"), m.Dificultades, esc)
		latexCampoActa(b, tx.L("SEG_MEDIDAS"), m.Medidas, esc)
	}

	b.WriteString(`\actasecc{` + tx.L("ACTA_DESENVOLVEMENTO") + "}\n" + latexTextoActa(a.Desenvolvemento, tx, esc))
	b.WriteString(`\actasecc{` + tx.L("ACTA_ACORDOS") + "}\n" + latexTextoActa(a.Acordos, tx, esc))
	b.WriteString(`\actasecc{` + tx.L("ACTA_ROGOS") + "}\n" + latexTextoActa(a.Rogos, tx, esc))

	if a.HoraFin != "" {
		b.WriteString(`\par\medskip\noindent ` + tx.L("ACTA_PECHE", v) + "\n")
	}
	b.WriteString(`\actasinaturas{` + tx.L("ACTA_SECRETARIO") + `}{` + EscapeLatexProse(porDefectoCrono(p.Config.Secretary, "Nome e apelidos")) +
		`}{` + tx.L("ACTA_VISTO") + `}{` + EscapeLatexProse(porDefectoCrono(p.Config.HeadOfDepartment, "Nome e apelidos")) + "}\n")
}

// latexInformeActas compón o documento LaTeX das actas: o curso enteiro (con
// portada e índice) ou unha soa acta, segundo `ambito`.
func latexInformeActas(p PlanActas, ambito, engine string, tip *TipografiaOpts, dir string, agora time.Time, plantilla *PlantillaLatex, nomeDoc string) (string, []string, error) {
	esc := novoEscritorImaxesCrono(dir)
	logo := esc.escribir(p.Config.SchoolLogo, "logotipo")
	auto := variablesActas(p, nomeDoc, agora)
	tx := novosTextosDoc(plantilla, auto)
	tx.engadirVariables(auto, "TITULO_ACTAS")
	curso := ambito == "" || ambito == ambitoCursoActas
	actas := actasResoltas(p, ambito)

	dd := compoñerDeseno(engine, tip, logo, plantilla)
	if engineUnicodeNativo(engine) {
		if err := escribirFontesInformeCrono(dir, dd.Fontes); err != nil {
			esc.avisos = append(esc.avisos, "non se puido escribir a fonte do documento: "+err.Error())
		}
	}
	var b strings.Builder
	b.WriteString(babelListquotFix)
	b.WriteString(formulasDefLatexPorDefecto)
	b.WriteString(dd.Corpo)
	if curso {
		b.WriteString("\\renewcommand{\\contentsname}{" + tx.L("ACTAS_LISTA") + "}\n\\tableofcontents\n\\clearpage\n")
	}
	if len(actas) == 0 {
		b.WriteString("\\textit{" + tx.L("SEN_ACTAS") + "}\n")
	}
	for i, a := range actas {
		if i > 0 {
			b.WriteString("\\clearpage\n")
		}
		if curso {
			b.WriteString(`\phantomsection\addcontentsline{toc}{section}{` + tx.L("ACTA_ENTRADA_INDICE", a.vars()) + "}\n")
		}
		latexCorpoActa(&b, p, a, tx, esc)
	}

	doc, avisosPlantilla, err := documentoLatexConPlantilla(
		preambuloDeseno(engine, dd, ""),
		partesInformeCrono{PortadaDef: dd.PortadaDef, CabeceiraPe: dd.CabeceiraPe, Corpo: b.String(), SenPortada: !curso || dd.SenPortada, SenSinatura: true},
		engine, plantilla, auto, paquetesInformeCrono, dd.BloqueCompleta)
	if err != nil {
		return "", nil, err
	}
	return doc, engadirSenRepetir(esc.avisos, avisosPlantilla, dd.Avisos), nil
}

// validarDatasActas é validarDatasCrono para o rango do curso de actas.
func validarDatasActas(p PlanActas) string {
	return validarDatasCrono(p.Config.StartDate, p.Config.EndDate)
}

// compilarInformeActas compila as actas co motor preferido (ver compilarComMotores).
func compilarInformeActas(p PlanActas, ambito, preferido string, tip *TipografiaOpts, dir string, plantilla *PlantillaLatex, nomeDoc string) (resultadoInformeCrono, error) {
	if msg := validarDatasActas(p); msg != "" {
		return resultadoInformeCrono{}, fmt.Errorf("%s", msg)
	}
	return compilarComMotores(preferido, tip, dir, plantilla, func(motor string) (string, []string, error) {
		return latexInformeActas(p, ambito, motor, tip, dir, time.Now(), plantilla, nomeDoc)
	})
}

// ---------- Markdown ----------

func mdCampoActa(b *strings.Builder, etiqueta, md string, esc *escritorImaxesCrono, nivel int) {
	bloques := parseMarkdownCrono(md)
	if len(bloques) == 0 {
		return
	}
	b.WriteString("**" + mdEscaparCrono(etiqueta) + "**\n\n" + mdBloquesCrono(bloques, nil, esc, nivel))
}

func mdTextoActa(md string, tx textosDocumento, esc *escritorImaxesCrono, nivel int) string {
	bloques := parseMarkdownCrono(md)
	if len(bloques) == 0 {
		return "*" + mdEscaparCrono(tx.T("ACTA_SEN_CONTIDO")) + "*\n\n"
	}
	return mdBloquesCrono(bloques, nil, esc, nivel)
}

// markdownInformeActas compón as actas en Markdown.
func markdownInformeActas(p PlanActas, ambito, formato, dir string, agora time.Time, plantilla *PlantillaLatex) (string, []string) {
	esc := novoEscritorImaxesCrono(dir)
	vars := variablesActas(p, "", agora)
	tx := novosTextosDoc(plantilla, vars)
	tx.engadirVariables(vars, "TITULO_ACTAS")
	salto := saltoPaxinaMdCrono(formato)
	curso := ambito == "" || ambito == ambitoCursoActas
	actas := actasResoltas(p, ambito)

	var b strings.Builder
	if curso {
		if im := esc.escribir(p.Config.SchoolLogo, "logotipo"); im != nil {
			b.WriteString("![Logotipo](" + im.Ficheiro + "){width=45mm}\n\n")
		}
		b.WriteString("# " + mdEscaparCrono(strings.ToUpper(vars["TITULO"])) + "\n\n")
		b.WriteString("**" + mdEscaparCrono(vars["DEPARTAMENTO"]) + "**\n\n")
		b.WriteString("**" + mdEscaparCrono(vars["CENTRO"]) + "**\n\n")
		b.WriteString(mdEscaparCrono(tx.T("CURSO_ACADEMICO")) + "\n\n")
		b.WriteString(mdEscaparCrono(vars["LOCALIDADE"]+", "+vars["DATA_LONGA"]) + "\n\n")
		b.WriteString(salto)
		b.WriteString("# " + mdEscaparCrono(tx.T("ACTAS_LISTA")) + "\n\n")
		for i, a := range actas {
			b.WriteString(fmt.Sprintf("%d. %s\n", i+1, mdEscaparCrono(tx.T("ACTA_ENTRADA_INDICE", a.vars()))))
		}
		b.WriteString("\n")
	}
	if len(actas) == 0 {
		b.WriteString("*" + mdEscaparCrono(tx.T("SEN_ACTAS")) + "*\n\n")
	}
	for i, a := range actas {
		if i > 0 || curso {
			b.WriteString(salto)
		}
		v := a.vars()
		b.WriteString("# " + mdEscaparCrono(tx.T("ACTA_TITULO", v)) + "\n\n")
		b.WriteString("*" + mdEscaparCrono(tx.T("ACTA_SUBTITULO", v)) + "*\n\n")
		if a.Data != "" {
			b.WriteString("**" + mdEscaparCrono(tx.T("ACTA_DATA")) + ":** " + mdEscaparCrono(a.Data) + "  \n")
		}
		if h := horasActa(tx, a); h != "" {
			b.WriteString("**" + mdEscaparCrono(tx.T("ACTA_HORA")) + ":** " + mdEscaparCrono(h) + "  \n")
		}
		if a.Lugar != "" {
			b.WriteString("**" + mdEscaparCrono(tx.T("ACTA_LUGAR")) + ":** " + mdEscaparCrono(a.Lugar) + "  \n")
		}
		b.WriteString("\n")
		if len(p.Config.Members) > 0 {
			b.WriteString("## " + mdEscaparCrono(tx.T("ACTA_ASISTENTES")) + "\n\n")
			for _, n := range a.Asistentes {
				b.WriteString("- " + mdEscaparCrono(n) + "\n")
			}
			ausentes := tx.T("ACTA_SEN_AUSENTES")
			if len(a.Ausentes) > 0 {
				ausentes = strings.Join(a.Ausentes, "; ")
			}
			b.WriteString("\n**" + mdEscaparCrono(tx.T("ACTA_AUSENTES")) + ":** " + mdEscaparCrono(ausentes) + "\n\n")
		}
		b.WriteString("## " + mdEscaparCrono(tx.T("ACTA_ORDE")) + "\n\n" + mdTextoActa(a.Orde, tx, esc, 3))
		b.WriteString("## " + mdEscaparCrono(tx.T("ACTA_SEGUIMENTO")) + "\n\n")
		if len(a.Materias) == 0 {
			b.WriteString("*" + mdEscaparCrono(tx.T("ACTA_SEN_SEGUIMENTO")) + "*\n\n")
		}
		for _, m := range a.Materias {
			b.WriteString("### " + mdEscaparCrono(m.Nome) + "\n\n")
			if m.Docente != "" {
				b.WriteString("**" + mdEscaparCrono(tx.T("SEG_DOCENTE")) + ":** " + mdEscaparCrono(m.Docente) + "  \n")
			}
			if m.Grupos != "" {
				b.WriteString("**" + mdEscaparCrono(tx.T("SEG_GRUPOS")) + ":** " + mdEscaparCrono(m.Grupos) + "  \n")
			}
			if !m.TenDatos {
				b.WriteString("\n*" + mdEscaparCrono(tx.T("SEG_SEN_DATOS")) + "*\n\n")
				continue
			}
			if m.Progreso != nil {
				b.WriteString("**" + mdEscaparCrono(tx.T("SEG_CUMPRIMENTO")) + ":** " + strconv.Itoa(limitarPorcentaxe(*m.Progreso)) + " %\n")
			}
			b.WriteString("\n")
			mdCampoActa(&b, tx.T("SEG_CONTIDOS"), m.Contidos, esc, 5)
			mdCampoActa(&b, tx.T("SEG_RESULTADOS"), m.Resultados, esc, 5)
			mdCampoActa(&b, tx.T("SEG_DIFICULTADES"), m.Dificultades, esc, 5)
			mdCampoActa(&b, tx.T("SEG_MEDIDAS"), m.Medidas, esc, 5)
		}
		b.WriteString("## " + mdEscaparCrono(tx.T("ACTA_DESENVOLVEMENTO")) + "\n\n" + mdTextoActa(a.Desenvolvemento, tx, esc, 3))
		b.WriteString("## " + mdEscaparCrono(tx.T("ACTA_ACORDOS")) + "\n\n" + mdTextoActa(a.Acordos, tx, esc, 3))
		b.WriteString("## " + mdEscaparCrono(tx.T("ACTA_ROGOS")) + "\n\n" + mdTextoActa(a.Rogos, tx, esc, 3))
		if a.HoraFin != "" {
			b.WriteString(mdEscaparCrono(tx.T("ACTA_PECHE", v)) + "\n\n")
		}
		b.WriteString("\n\n**" + mdEscaparCrono(tx.T("ACTA_SECRETARIO")) + "**  \n" + mdEscaparCrono(porDefectoCrono(p.Config.Secretary, "Nome e apelidos")) + "\n\n")
		b.WriteString("**" + mdEscaparCrono(tx.T("ACTA_VISTO")) + "**  \n" + mdEscaparCrono(porDefectoCrono(p.Config.HeadOfDepartment, "Nome e apelidos")) + "\n")
	}
	return b.String(), esc.avisos
}

// ---------- Word / OpenDocument ----------

// prepararOfimaticaActas resolve o que precisan os escritores DOCX/ODT das actas.
func prepararOfimaticaActas(req XerarInformeCronogramaRequest, plantilla *PlantillaLatex, agora time.Time) (datosOfimatica, error) {
	p := *req.Actas
	vars := variablesActas(p, req.NomeBase, agora)
	tx := novosTextosDoc(plantilla, vars)
	tx.engadirVariables(vars, "TITULO_ACTAS")
	d := baseOfimatica(plantilla, vars, tx, p.Config.SchoolLogo, nil)
	cc := p.comoCrono()
	d.Plan = cc
	d.Centro = vars["CENTRO"]
	d.Localidade = vars["LOCALIDADE"]
	d.Curso = p.AcademicYear
	d.Data = dataGalegaCrono(agora)
	d.Xefe = porDefectoCrono(p.Config.HeadOfDepartment, "Nome e apelidos")
	d.Cargo = cargoSinaturaCrono(cc)
	d.Departamento = vars["DEPARTAMENTO"]
	d.Actas = actasResoltas(p, req.Ambito)
	d.ActasCurso = req.Ambito == "" || req.Ambito == ambitoCursoActas
	d.ActaSecretario = porDefectoCrono(p.Config.Secretary, "Nome e apelidos")
	return d, nil
}

// escribirDocxActas escribe as actas como Word.
func escribirDocxActas(d *datosOfimatica) ([]byte, error) {
	c := &construtorDocx{baseMedios: int(d.Estilo.TamanoPt*2 + 0.5)}
	tx := d.Tx
	// Unha acta soa non leva portada nin índice: empeza directamente.
	if d.ActasCurso {
		c.portada(d)
		c.textoSalto("Heading1", tx.T("ACTAS_LISTA"))
		for _, a := range d.Actas {
			c.texto("TOCEntry", tx.T("ACTA_ENTRADA_INDICE", a.vars()))
		}
	}
	if len(d.Actas) == 0 {
		c.p("BodyText", runDocx(tx.T("SEN_ACTAS"), false, true, "808080", 0))
	}
	for i, a := range d.Actas {
		v := a.vars()
		if i == 0 && !d.ActasCurso {
			c.texto("Heading1", tx.T("ACTA_TITULO", v))
		} else {
			c.textoSalto("Heading1", tx.T("ACTA_TITULO", v))
		}
		c.p("BodyText", runDocx(tx.T("ACTA_SUBTITULO", v), false, true, "", 0))
		dato := func(etq, val string) {
			c.p("BodyText", runDocx(etq+": ", true, false, "", 0), runDocx(val, false, false, "", 0))
		}
		if a.Data != "" {
			dato(tx.T("ACTA_DATA"), a.Data)
		}
		if h := horasActa(tx, a); h != "" {
			dato(tx.T("ACTA_HORA"), h)
		}
		if a.Lugar != "" {
			dato(tx.T("ACTA_LUGAR"), a.Lugar)
		}
		if len(a.Asistentes) > 0 || len(a.Ausentes) > 0 {
			c.texto("Heading2", tx.T("ACTA_ASISTENTES"))
			for _, n := range a.Asistentes {
				c.corpo.WriteString(paraDocx("ListBullet", "", runDocx("•\t", false, false, "", 0), runDocx(n, false, false, "", 0)))
			}
			ausentes := tx.T("ACTA_SEN_AUSENTES")
			if len(a.Ausentes) > 0 {
				ausentes = strings.Join(a.Ausentes, "; ")
			}
			dato(tx.T("ACTA_AUSENTES"), ausentes)
		}
		texto := func(md string) {
			if bl := parseMarkdownCrono(md); len(bl) > 0 {
				c.bloques(bl, nil, &d.Avisos)
			} else {
				c.p("BodyText", runDocx(tx.T("ACTA_SEN_CONTIDO"), false, true, "808080", 0))
			}
		}
		c.texto("Heading2", tx.T("ACTA_ORDE"))
		texto(a.Orde)
		c.texto("Heading2", tx.T("ACTA_SEGUIMENTO"))
		if len(a.Materias) == 0 {
			c.p("BodyText", runDocx(tx.T("ACTA_SEN_SEGUIMENTO"), false, true, "808080", 0))
		}
		for _, m := range a.Materias {
			titulo := m.Nome
			if m.Docente != "" {
				titulo += " — " + m.Docente
			}
			if m.Grupos != "" {
				titulo += " · " + m.Grupos
			}
			c.texto("TaskTitle", titulo)
			if !m.TenDatos {
				c.p("BodyText", runDocx(tx.T("SEG_SEN_DATOS"), false, true, "808080", 0))
				continue
			}
			if m.Progreso != nil {
				dato(tx.T("SEG_CUMPRIMENTO"), strconv.Itoa(limitarPorcentaxe(*m.Progreso))+" %")
			}
			for _, campo := range []struct{ clave, md string }{
				{"SEG_CONTIDOS", m.Contidos}, {"SEG_RESULTADOS", m.Resultados},
				{"SEG_DIFICULTADES", m.Dificultades}, {"SEG_MEDIDAS", m.Medidas},
			} {
				if bl := parseMarkdownCrono(campo.md); len(bl) > 0 {
					c.p("BodyText", runDocx(tx.T(campo.clave), true, false, "", 0))
					c.bloques(bl, nil, &d.Avisos)
				}
			}
		}
		c.texto("Heading2", tx.T("ACTA_DESENVOLVEMENTO"))
		texto(a.Desenvolvemento)
		c.texto("Heading2", tx.T("ACTA_ACORDOS"))
		texto(a.Acordos)
		c.texto("Heading2", tx.T("ACTA_ROGOS"))
		texto(a.Rogos)
		if a.HoraFin != "" {
			c.texto("BodyText", tx.T("ACTA_PECHE", v))
		}
		c.texto("SignName", d.ActaSecretario)
		c.texto("SignMeta", tx.T("ACTA_SECRETARIO"))
		c.texto("SignName", d.Xefe)
		c.texto("SignMeta", tx.T("ACTA_VISTO"))
	}
	return montarDocx(c.corpo.String(), c.imaxes, d)
}

// escribirOdtActas escribe as actas como OpenDocument.
func escribirOdtActas(d *datosOfimatica) ([]byte, error) {
	c := &construtorOdt{imaxesPlantilla: d.ImaxesPlantilla}
	tx := d.Tx
	extraEstilos := ""
	if d.ActasCurso {
		extraEstilos = c.portada(d)
		c.p("Heading1", tx.T("ACTAS_LISTA"))
		for _, a := range d.Actas {
			c.p("TOCEntry", tx.T("ACTA_ENTRADA_INDICE", a.vars()))
		}
	}
	muted := func(t string) {
		c.corpo.WriteString(`<text:p text:style-name="TextBody"><text:span text:style-name="Muted">` + escaparXMLOfi(t) + `</text:span></text:p>`)
	}
	if len(d.Actas) == 0 {
		muted(tx.T("SEN_ACTAS"))
	}
	for _, a := range d.Actas {
		v := a.vars()
		c.p("Heading1", tx.T("ACTA_TITULO", v))
		muted(tx.T("ACTA_SUBTITULO", v))
		dato := func(etq, val string) {
			c.corpo.WriteString(`<text:p text:style-name="TextBody"><text:span text:style-name="Strong">` + escaparXMLOfi(etq+": ") + `</text:span>` + escaparXMLOfi(val) + `</text:p>`)
		}
		if a.Data != "" {
			dato(tx.T("ACTA_DATA"), a.Data)
		}
		if h := horasActa(tx, a); h != "" {
			dato(tx.T("ACTA_HORA"), h)
		}
		if a.Lugar != "" {
			dato(tx.T("ACTA_LUGAR"), a.Lugar)
		}
		if len(a.Asistentes) > 0 || len(a.Ausentes) > 0 {
			c.p("Heading2", tx.T("ACTA_ASISTENTES"))
			for _, n := range a.Asistentes {
				c.corpo.WriteString(`<text:p text:style-name="ListItem">•<text:tab/>` + escaparXMLOfi(n) + `</text:p>`)
			}
			ausentes := tx.T("ACTA_SEN_AUSENTES")
			if len(a.Ausentes) > 0 {
				ausentes = strings.Join(a.Ausentes, "; ")
			}
			dato(tx.T("ACTA_AUSENTES"), ausentes)
		}
		texto := func(md string) {
			if bl := parseMarkdownCrono(md); len(bl) > 0 {
				c.bloques(bl, nil, &d.Avisos)
			} else {
				muted(tx.T("ACTA_SEN_CONTIDO"))
			}
		}
		c.p("Heading2", tx.T("ACTA_ORDE"))
		texto(a.Orde)
		c.p("Heading2", tx.T("ACTA_SEGUIMENTO"))
		if len(a.Materias) == 0 {
			muted(tx.T("ACTA_SEN_SEGUIMENTO"))
		}
		for _, m := range a.Materias {
			titulo := m.Nome
			if m.Docente != "" {
				titulo += " — " + m.Docente
			}
			if m.Grupos != "" {
				titulo += " · " + m.Grupos
			}
			c.p("TaskTitle", titulo)
			if !m.TenDatos {
				muted(tx.T("SEG_SEN_DATOS"))
				continue
			}
			if m.Progreso != nil {
				dato(tx.T("SEG_CUMPRIMENTO"), strconv.Itoa(limitarPorcentaxe(*m.Progreso))+" %")
			}
			for _, campo := range []struct{ clave, md string }{
				{"SEG_CONTIDOS", m.Contidos}, {"SEG_RESULTADOS", m.Resultados},
				{"SEG_DIFICULTADES", m.Dificultades}, {"SEG_MEDIDAS", m.Medidas},
			} {
				if bl := parseMarkdownCrono(campo.md); len(bl) > 0 {
					c.corpo.WriteString(`<text:p text:style-name="TextBody"><text:span text:style-name="Strong">` + escaparXMLOfi(tx.T(campo.clave)) + `</text:span></text:p>`)
					c.bloques(bl, nil, &d.Avisos)
				}
			}
		}
		c.p("Heading2", tx.T("ACTA_DESENVOLVEMENTO"))
		texto(a.Desenvolvemento)
		c.p("Heading2", tx.T("ACTA_ACORDOS"))
		texto(a.Acordos)
		c.p("Heading2", tx.T("ACTA_ROGOS"))
		texto(a.Rogos)
		if a.HoraFin != "" {
			c.p("TextBody", tx.T("ACTA_PECHE", v))
		}
		c.p("SignName", d.ActaSecretario)
		c.p("SignMeta", tx.T("ACTA_SECRETARIO"))
		c.p("SignName", d.Xefe)
		c.p("SignMeta", tx.T("ACTA_VISTO"))
	}
	d.cabOdt = c.caixaOdt(d.Cabeceira, "Header")
	d.peOdt = c.caixaOdt(d.Pe, "Footer")
	contido := odtCabeceiraContido + odtEstilosAutomaticosCon(d.Estilo, extraEstilos) +
		`<office:body><office:text>` + c.corpo.String() + `</office:text></office:body></office:document-content>`
	return montarOdt(contido, c.imaxes, d)
}

// ---------- Ficheiros ----------

// borradorFerramentaNome é o ficheiro de borrador de cada ferramenta.
func borradorFerramentaNome(ferramenta string) string {
	if ferramenta == ferramentaActas {
		return "actas-borrador.json"
	}
	// Cada formulario propio ten o seu borrador («form:<id>» no frontend).
	if id, ok := idFerramentaFormulario(ferramenta); ok {
		return "formulario-" + id + "-borrador.json"
	}
	return "cronograma-borrador.json"
}

// borradorFerramentaPath: onde se garda o traballo en curso de cada ferramenta.
// A raíz é <config>/lua/, a mesma ca settings.json e plantillas.json.
func borradorFerramentaPath(ferramenta string) (string, error) {
	// Dentro dun proxecto (proxecto.go) os datos son do proxecto.
	if d := dirProxectoActivo(); d != "" {
		return filepath.Join(d, "datos", borradorFerramentaNome(ferramenta)), nil
	}
	dir, err := dirConfigLua()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "informes", borradorFerramentaNome(ferramenta)), nil
}

// borradorFerramentaPathVello é a localización histórica dos borradores:
// <config>/Lua/informes/ — con "Lua" en MAIÚSCULA, inconsistente co resto da
// configuración de Lua (<config>/lua/). En Linux e noutros sistemas de
// ficheiros sensibles a maiúsculas son dous directorios distintos, así que os
// borradores escritos por versións anteriores viven nunha árbore á parte.
// Consérvase só para MIGRALOS (ver LerBorradorInforme): sen iso, cambiar a
// ruta faría desaparecer o traballo en curso de quen xa tiña un borrador.
func borradorFerramentaPathVello(ferramenta string) (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "Lua", "informes", borradorFerramentaNome(ferramenta)), nil
}

// mesesConActa devolve as claves dos meses que teñen acta, en orde.
func mesesConActa(p PlanActas) []string {
	var out []string
	for k, m := range p.MonthsData {
		if m.Acta != nil {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// planDemoActas é o curso de mentira co que se previsualiza unha plantilla no
// seu editor cando se escolle «Actas»: un departamento inventado con tres
// materias e unha acta. Datas fixas para que a vista previa sexa sempre igual.
func planDemoActas() PlanActas {
	uno := 80
	return PlanActas{
		ID:           "demo",
		AcademicYear: "2025-2026",
		Config: ConfigActas{
			StartDate: "2025-09-01", EndDate: "2025-11-30",
			SchoolName: "Centro de exemplo", Location: "Localidade", Department: "Departamento de exemplo",
			HeadOfDepartment: "Nome e apelidos", Secretary: "Nome e apelidos",
			MeetingTime: "16:00", MeetingPlace: "Sala de profesorado",
			Members: []MembroActas{{ID: "m1", Name: "Ana Pérez", Role: "Xefatura"}, {ID: "m2", Name: "Marta Souto"}, {ID: "m3", Name: "Xoán Rial"}},
			Subjects: []MateriaActas{
				{ID: "s1", Name: "Materia A", Teacher: "Ana Pérez", Groups: "1º ESO"},
				{ID: "s2", Name: "Materia B", Teacher: "Marta Souto", Groups: "2º ESO"},
			},
		},
		MonthsData: map[string]MesActas{
			"2025-10": {Year: 2025, Month: 9,
				Acta: &ActaActas{
					ID: "a1", Date: "2025-10-16", StartTime: "16:00", EndTime: "17:15", Place: "Sala de profesorado",
					Present:     []string{"m1", "m2"},
					Agenda:      "1. Lectura e aprobación da acta anterior.\n2. Seguimento das programacións.\n3. Rogos e preguntas.",
					Development: "Revísase o desenvolvemento das programacións e coméntanse as incidencias do mes.",
					Agreements:  "- Coordinar as probas de avaliación inicial.\n- Revisar os criterios de cualificación.",
					Questions:   "Sen rogos nin preguntas.",
				},
				Reports: map[string]SeguimentoActas{
					"s1": {Progress: &uno, Contents: "Unidades 1 e 2.", Results: "Bos resultados xerais.", Difficulties: "Falta de material.", Measures: "- Reforzo en pequeno grupo"},
				},
			},
		},
	}
}
