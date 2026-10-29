package main

// Informe "Cronograma anual" (departamento/centro educativo): o que xera
// Lua. O usuario introduce as tarefas de cada mes (ver frontend/src/informes/)
// e aquí compóñese o informe completo -portada, índice, introdución,
// cronograma anual, desenvolvemento por meses e sinatura- en LaTeX, que se
// compila co motor escollido en Opcións. A biblioteca de plantillas
// (plantillas.go) pode envolver o informe cunha cabeceira e un pé propios de
// cada centro.
//
// O modelo de datos (PlanCrono) é o mesmo JSON ca o da app de escritorio
// Piztu Orienta (cronograma/internal/model), así que os cursos gardados alí
// abren aquí sen conversión. As descricións das tarefas son Markdown (ver
// informe_md.go).

import (
	"bytes"
	"crypto/sha1"
	"edicion/idioma"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/gif"
	_ "image/jpeg"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ---------- Modelo ----------

// TarefaCrono é unha tarefa dentro dun mes. A descrición é Markdown; as
// imaxes que leva inseridas (`![alt](img:ID)`) van no mapa Images como data
// URI, para que o ficheiro do curso sexa autocontido.
type TarefaCrono struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Color       string            `json:"color"`
	Images      map[string]string `json:"images,omitempty"`
}

// MesCrono agrupa as tarefas dun mes. Month vai de 0 (xaneiro) a 11.
type MesCrono struct {
	Year  int           `json:"year"`
	Month int           `json:"month"`
	Tasks []TarefaCrono `json:"tasks"`
}

// ConfigCrono son os datos do centro e o rango temporal do curso.
type ConfigCrono struct {
	StartDate  string `json:"startDate"` // AAAA-MM-DD
	EndDate    string `json:"endDate"`   // AAAA-MM-DD
	SchoolName string `json:"schoolName"`
	Location   string `json:"location"`
	// Department é o nome do departamento ("Departamento de Matemáticas"...):
	// Lua serve para calquera departamento didáctico. Baleiro = "Departamento".
	Department       string `json:"department"`
	HeadOfDepartment string `json:"headOfDepartment"`
	SchoolLogo       string `json:"schoolLogo"` // data URI
	IntroText        string `json:"introText"`
	SignatoryRole    string `json:"signatoryRole"`
}

// PlanCrono é un curso académico completo.
type PlanCrono struct {
	ID           string              `json:"id"`
	AcademicYear string              `json:"academicYear"`
	IsLocked     bool                `json:"isLocked"`
	Config       ConfigCrono         `json:"config"`
	MonthsData   map[string]MesCrono `json:"monthsData"` // clave "AAAA-MM"
}

// XerarInformeCronogramaRequest é o que manda o frontend ao premer "Xerar".
type XerarInformeCronogramaRequest struct {
	Plan       PlanCrono       `json:"plan"`
	Tipografia *TipografiaOpts `json:"tipografia"`
	// NomeBase é o nome (sen extensión) do curso aberto, ou o suxerido se
	// aínda non se gardou. Serve para {{FICHEIRO}} das plantillas e para o
	// nome que se propón ao exportar dende a xanela de resultado desacoplada.
	NomeBase string `json:"nomeBase"`
	// Ferramenta di que documento é: "cronograma" (Plan; o de sempre, tamén se
	// vai baleiro) ou "actas" (Actas). Ámbito, só para as actas, é "curso" (todas
	// as do curso) ou "AAAA-MM" (a dun mes).
	Ferramenta string     `json:"ferramenta"`
	Actas      *PlanActas `json:"actas,omitempty"`
	Ambito     string     `json:"ambito"`
	// Formulario: cando Ferramenta é "formulario", o formulario propio e os
	// seus datos (ver formulario.go).
	Formulario *PedidoFormulario `json:"formulario,omitempty"`
}

// eFormulario di se a petición é dun formulario propio do usuario.
func (r XerarInformeCronogramaRequest) eFormulario() bool {
	return r.Ferramenta == ferramentaFormulario && r.Formulario != nil
}

// eActas di se a petición é do documento de actas.
func (r XerarInformeCronogramaRequest) eActas() bool {
	return r.Ferramenta == ferramentaActas && r.Actas != nil
}

// ---------- Paleta ----------

// corTarefaCrono é o aspecto dunha tarefa: fondo, texto e bordo (hex sen #).
type corTarefaCrono struct {
	Clave, Nome, Fondo, Texto, Bordo string
}

// paletaTarefasCrono debe coincidir coa de frontend/src/informes/cronograma-modelo.js.
var paletaTarefasCrono = []corTarefaCrono{
	{"blue", "Azul", "EFF6FF", "1E40AF", "BFDBFE"},
	{"emerald", "Verde", "ECFDF5", "065F46", "A7F3D0"},
	{"amber", "Ámbar", "FFFBEB", "92400E", "FDE68A"},
	{"rose", "Rosa", "FFF1F2", "9F1239", "FECDD3"},
	{"purple", "Morado", "FAF5FF", "6B21A8", "E9D5FF"},
	{"cyan", "Cian", "ECFEFF", "155E75", "A5F3FC"},
	{"indigo", "Índigo", "EEF2FF", "3730A3", "C7D2FE"},
	{"stone", "Gris", "FAFAF9", "292524", "E7E5E4"},
}

// resolverCorCrono devolve a cor dunha clave, tolerando os valores herdados da
// versión web ("bg-blue-50 text-blue-800 border-blue-200") e caendo na
// primeira da paleta se non a recoñece.
func resolverCorCrono(clave string) corTarefaCrono {
	if clave == "" {
		return paletaTarefasCrono[0]
	}
	for _, c := range paletaTarefasCrono {
		if c.Clave == clave {
			return c
		}
	}
	for _, c := range paletaTarefasCrono {
		if strings.Contains(clave, c.Clave) {
			return c
		}
	}
	return paletaTarefasCrono[0]
}

// coresMesCrono é a cor de fondo da banda de cada mes (0 = xaneiro).
var coresMesCrono = [12]string{
	"DBEAFE", "CFFAFE", "ECFDF5", "D1FAE5", "DCFCE7", "FEFCE8",
	"FEF9C3", "FFEDD5", "FFF7ED", "FEF3C7", "E7E5E4", "E2E8F0",
}

var nomesMesesCrono = [12]string{
	"Xaneiro", "Febreiro", "Marzo", "Abril", "Maio", "Xuño",
	"Xullo", "Agosto", "Setembro", "Outubro", "Novembro", "Decembro",
}

var nomesMesesMinusculaCrono = [12]string{
	"xaneiro", "febreiro", "marzo", "abril", "maio", "xuño",
	"xullo", "agosto", "setembro", "outubro", "novembro", "decembro",
}

// ---------- Calendario ----------

// mesCalendario é un mes resolto do calendario, listo para amosar.
type mesCalendario struct {
	Ano, Mes int // Mes 0-11
	Clave    string
	Nome     string
	Tarefas  []TarefaCrono
}

func claveMesCrono(ano, mes int) string { return fmt.Sprintf("%04d-%02d", ano, mes+1) }

// anoMesCrono extrae ano e mes (0-11) dunha data "AAAA-MM-DD".
func anoMesCrono(data string) (ano, mes int, ok bool) {
	partes := strings.Split(strings.TrimSpace(data), "-")
	if len(partes) < 2 {
		return 0, 0, false
	}
	a, err1 := strconv.Atoi(partes[0])
	m, err2 := strconv.Atoi(partes[1])
	if err1 != nil || err2 != nil || m < 1 || m > 12 {
		return 0, 0, false
	}
	return a, m - 1, true
}

// calendarioCrono devolve os meses entre a data de inicio e a de fin, en
// orde, coas tarefas que teña cada un.
func calendarioCrono(cfg ConfigCrono, datos map[string]MesCrono) []mesCalendario {
	a0, m0, ok1 := anoMesCrono(cfg.StartDate)
	a1, m1, ok2 := anoMesCrono(cfg.EndDate)
	if !ok1 || !ok2 {
		return nil
	}
	total := (a1-a0)*12 + (m1 - m0) + 1
	if total <= 0 {
		return nil
	}
	meses := make([]mesCalendario, 0, total)
	for i := 0; i < total; i++ {
		ano := a0 + (m0+i)/12
		mes := (m0 + i) % 12
		clave := claveMesCrono(ano, mes)
		meses = append(meses, mesCalendario{
			Ano: ano, Mes: mes, Clave: clave, Nome: nomesMesesCrono[mes],
			Tarefas: datos[clave].Tasks,
		})
	}
	return meses
}

// validarDatasCrono devolve unha mensaxe de erro en galego, ou "" se as datas
// son válidas e o fin non é anterior ao inicio.
func validarDatasCrono(inicio, fin string) string {
	i, err := time.Parse("2006-01-02", strings.TrimSpace(inicio))
	if err != nil {
		return "A data de inicio non é válida. Usa o formato AAAA-MM-DD (ex.: 2025-09-08)."
	}
	f, err := time.Parse("2006-01-02", strings.TrimSpace(fin))
	if err != nil {
		return "A data de fin non é válida. Usa o formato AAAA-MM-DD (ex.: 2026-06-19)."
	}
	if f.Before(i) {
		return "A data de fin non pode ser anterior á de inicio."
	}
	return ""
}

// subtituloMesCrono é o texto pequeno da etiqueta dun mes ("3 tarefas"; baleiro
// se non ten ningunha). O texto vén dos textos da plantilla (TAREFA_UNHA e
// TAREFAS_N).
func subtituloMesCrono(tx textosDocumento, n int) string {
	switch n {
	case 0:
		return ""
	case 1:
		return tx.T("TAREFA_UNHA")
	}
	return tx.T("TAREFAS_N", map[string]string{"N": strconv.Itoa(n)})
}

// trimestreCrono é un bloque do cronograma anual. Mesos son os meses que
// lle tocan; Titulo ("1.º trimestre") e Meses ("Setembro – Decembro") son o
// texto da cabeceira.
type trimestreCrono struct {
	Numero int
	Titulo string
	Meses  string
	Mesos  []mesCalendario
}

// numeroTrimestreCrono di a que trimestre do curso pertence un mes (0-11):
// setembro a decembro é o 1.º, xaneiro a marzo o 2.º e abril en diante (ata
// xuño, e xullo e agosto se o curso os leva) o 3.º.
func numeroTrimestreCrono(mes int) int {
	switch {
	case mes >= 8:
		return 1
	case mes <= 2:
		return 2
	}
	return 3
}

// trimestresCrono reparte os meses do curso, en orde, por trimestres. Un
// trimestre soa cos meses seguidos: un curso de máis de 12 meses volve abrir o
// 1.º cando chega outra vez setembro.
func trimestresCrono(meses []mesCalendario) []trimestreCrono {
	var out []trimestreCrono
	ultimo := 0
	for _, m := range meses {
		n := numeroTrimestreCrono(m.Mes)
		if len(out) == 0 || n != ultimo {
			out = append(out, trimestreCrono{Numero: n, Titulo: fmt.Sprintf("%d.º trimestre", n)})
			ultimo = n
		}
		g := &out[len(out)-1]
		g.Mesos = append(g.Mesos, m)
	}
	for i := range out {
		g := &out[i]
		g.Meses = g.Mesos[0].Nome
		if len(g.Mesos) > 1 {
			g.Meses += " – " + g.Mesos[len(g.Mesos)-1].Nome
		}
	}
	return out
}

func mesesConTarefasCrono(meses []mesCalendario) []mesCalendario {
	var out []mesCalendario
	for _, m := range meses {
		if len(m.Tarefas) > 0 {
			out = append(out, m)
		}
	}
	return out
}

// nomesMesesInformeCrono devolve o título de cada mes do desenvolvemento. Só
// leva o ano se o rango de datas repite un nome de mes (cursos de máis de 12
// meses), para non ter dous "Setembro" indistinguibles no índice.
func nomesMesesInformeCrono(meses []mesCalendario) []string {
	vistos := map[int]int{}
	for _, m := range meses {
		vistos[m.Mes]++
	}
	out := make([]string, len(meses))
	for i, m := range meses {
		out[i] = m.Nome
		if vistos[m.Mes] > 1 {
			out[i] = fmt.Sprintf("%s %d", m.Nome, m.Ano)
		}
	}
	return out
}

// ---------- Textos por defecto ----------

// departamentoPorDefectoCrono é o nome do departamento se o curso non pon
// ningún: ningún (queda en branco; non se inventa «Departamento»).
const departamentoPorDefectoCrono = ""

// nomeBaseCronoPorDefecto é o nome que se propón ao exportar se o frontend
// non mandou ningún.
const nomeBaseCronoPorDefecto = "cronograma"

func porDefectoCrono(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}

// introCrono devolve o texto da introdución con as variables enchidas: o do
// propio curso se o hai, senón o da plantilla e, senón, o por defecto. Devolve
// tamén os avisos das variables que non se poden encher.
func introCrono(p PlanCrono, plantilla *PlantillaLatex, vars map[string]string) (string, []string) {
	texto := p.Config.IntroText
	if strings.TrimSpace(texto) == "" && plantilla != nil {
		texto = plantilla.Introducion
	}
	if strings.TrimSpace(texto) == "" {
		texto = introPorDefectoCrono
	}
	if plantilla == nil {
		plantilla = &PlantillaLatex{}
	}
	return substituirVariables(texto, plantilla, vars, "", nil)
}

// departamentoCrono é o nome do departamento do curso.
func departamentoCrono(p PlanCrono) string {
	return porDefectoCrono(p.Config.Department, departamentoPorDefectoCrono)
}

// cargoSinaturaCrono é o cargo de quen asina: o do curso ou, se non se puxo,
// «Xefe/a de <departamento>»; sen departamento tampouco hai cargo inventado.
func cargoSinaturaCrono(p PlanCrono) string {
	if strings.TrimSpace(p.Config.SignatoryRole) == "" && departamentoCrono(p) == "" {
		return ""
	}
	return porDefectoCrono(p.Config.SignatoryRole, "Xefe/a de "+departamentoCrono(p))
}

func dataGalegaCrono(t time.Time) string {
	return fmt.Sprintf("%d de %s de %d", t.Day(), nomesMesesMinusculaCrono[int(t.Month())-1], t.Year())
}

// variablesAutomaticasCrono son as variables que Lua enche soa nas
// plantillas a partir do curso: {{CENTRO}}, {{LOCALIDADE}}, {{CURSO}},
// {{ORIENTADOR}} e {{XEFE}} (quen asina, o mesmo dato), {{CARGO}},
// {{DEPARTAMENTO}}, {{TITULO}} (o texto TITULO_CRONOGRAMA), {{FICHEIRO}}, {{DATA}}, {{DATA_LONGA}}
// ("18 de setembro de 2026"), {{DATA_INICIO}}, {{DATA_FIN}} e {{ANO}}. Unha variable co mesmo nome definida na
// propia plantilla gaña se ten valor (ver substituirVariables).
func variablesAutomaticasCrono(p PlanCrono, nomeDoc string, agora time.Time) map[string]string {
	// nomeDoc pode ser unha ruta completa (a que propón o frontend para
	// exportar): {{FICHEIRO}} é só o nome do ficheiro.
	if strings.TrimSpace(nomeDoc) != "" {
		nomeDoc = filepath.Base(nomeDoc)
	}
	v := variablesAutomaticas(nomeDoc)
	v["DATA"] = agora.Format("02/01/2006")
	v["ANO"] = agora.Format("2006")
	v["TITULO"] = textosDocPorClave["TITULO_CRONOGRAMA"].Defecto
	v["CURSO_ACADEMICO"] = "Curso académico " + p.AcademicYear
	v["DEPARTAMENTO"] = departamentoCrono(p)
	v["CARGO"] = cargoSinaturaCrono(p)
	v["DATA_LONGA"] = dataGalegaCrono(agora)
	for nome, data := range map[string]string{"DATA_INICIO": p.Config.StartDate, "DATA_FIN": p.Config.EndDate} {
		if t, err := time.Parse("2006-01-02", strings.TrimSpace(data)); err == nil {
			v[nome] = t.Format("02/01/2006")
		}
	}
	for nome, valor := range map[string]string{
		"CENTRO":     p.Config.SchoolName,
		"LOCALIDADE": p.Config.Location,
		"CURSO":      p.AcademicYear,
		"ORIENTADOR": p.Config.HeadOfDepartment,
		"XEFE":       p.Config.HeadOfDepartment,
	} {
		if strings.TrimSpace(valor) != "" {
			v[nome] = valor
		}
	}
	return v
}

// ---------- Imaxes ----------

// imaxeCrono é unha imaxe xa escrita en disco, lista para \includegraphics.
type imaxeCrono struct {
	Ficheiro string // relativo ao directorio de compilación ("images/x.png")
	W, H     int    // píxeles
}

// escritorImaxesCrono decodifica as imaxes (data URI) do informe e escríbeas
// nun directorio, unha soa vez cada unha. pdflatex só le PNG e JPEG: o GIF
// convértese a PNG e o resto (WebP, BMP...) descártase con aviso; o frontend
// xa normaliza a PNG ao inserilas, así que isto só afecta a ficheiros
// importados doutras fontes.
type escritorImaxesCrono struct {
	dir    string
	cache  map[string]*imaxeCrono
	avisos []string
}

func novoEscritorImaxesCrono(dir string) *escritorImaxesCrono {
	return &escritorImaxesCrono{dir: dir, cache: map[string]*imaxeCrono{}}
}

func decodificarDataURICrono(uri string) ([]byte, error) {
	i := strings.Index(uri, ",")
	if !strings.HasPrefix(uri, "data:") || i < 0 {
		return nil, fmt.Errorf("non é un data URI")
	}
	if !strings.Contains(uri[:i], ";base64") {
		return nil, fmt.Errorf("data URI sen base64")
	}
	raw, err := base64.StdEncoding.DecodeString(uri[i+1:])
	if err != nil || len(raw) == 0 {
		return nil, fmt.Errorf("base64 non válido")
	}
	return raw, nil
}

// escribir devolve a imaxe escrita en disco para `uri`, ou nil (e un aviso)
// se non se pode usar. `nome` só serve para a mensaxe de aviso.
func (e *escritorImaxesCrono) escribir(uri, nome string) *imaxeCrono {
	if uri == "" {
		return nil
	}
	suma := sha1.Sum([]byte(uri))
	chave := hex.EncodeToString(suma[:6])
	if im, ok := e.cache[chave]; ok {
		return im
	}
	e.cache[chave] = nil // tamén se lembran os fallos, para avisar unha soa vez

	raw, err := decodificarDataURICrono(uri)
	if err != nil {
		e.avisos = append(e.avisos, idioma.T("go.latex.imaxe_non_lida", "a imaxe «%s» non se puido ler (%v): omitiuse", nome, err))
		return nil
	}
	cfg, formato, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		e.avisos = append(e.avisos, idioma.T("go.latex.imaxe_formato", "a imaxe «%s» ten un formato que LaTeX non le (só PNG, JPEG e GIF): omitiuse", nome))
		return nil
	}
	ext := "png"
	switch formato {
	case "png":
	case "jpeg":
		ext = "jpg"
	case "gif":
		g, err := gif.Decode(bytes.NewReader(raw))
		if err != nil {
			e.avisos = append(e.avisos, idioma.T("go.latex.imaxe_gif", "a imaxe «%s» (GIF) non se puido converter: omitiuse", nome))
			return nil
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, g); err != nil {
			e.avisos = append(e.avisos, idioma.T("go.latex.imaxe_gif", "a imaxe «%s» (GIF) non se puido converter: omitiuse", nome))
			return nil
		}
		raw = buf.Bytes()
	default:
		e.avisos = append(e.avisos, fmt.Sprintf("a imaxe «%s» ten un formato non soportado (%s): omitiuse", nome, formato))
		return nil
	}
	if err := os.MkdirAll(filepath.Join(e.dir, "images"), 0o755); err != nil {
		e.avisos = append(e.avisos, "non se puido crear o cartafol de imaxes: "+err.Error())
		return nil
	}
	ficheiro := "images/img-" + chave + "." + ext
	if err := os.WriteFile(filepath.Join(e.dir, ficheiro), raw, 0o644); err != nil {
		e.avisos = append(e.avisos, "non se puido escribir a imaxe «"+nome+"»: "+err.Error())
		return nil
	}
	im := &imaxeCrono{Ficheiro: ficheiro, W: cfg.Width, H: cfg.Height}
	e.cache[chave] = im
	return im
}

// resolver busca a imaxe dunha descrición: `ref` é a clave do mapa da tarefa
// ou un data URI directamente.
func (e *escritorImaxesCrono) resolver(ref string, imaxes map[string]string) *imaxeCrono {
	uri := ref
	if !strings.HasPrefix(ref, "data:") {
		var ok bool
		if uri, ok = imaxes[ref]; !ok {
			e.avisos = append(e.avisos, "unha descrición fai referencia a unha imaxe («"+ref+"») que non existe: omitiuse")
			return nil
		}
	}
	return e.escribir(uri, ref)
}

// ---------- LaTeX ----------

// larguraColumnaMMCrono é o ancho útil da páxina: A4 (210 mm) con marxes de 20 mm.
const larguraColumnaMMCrono = 170.0

func mmLatexCrono(v float64) string { return strconv.FormatFloat(v, 'f', 1, 64) + "mm" }

// latexBloquesCrono converte os bloques dunha descrición en LaTeX.
func latexBloquesCrono(bloques []cronoBloque, imaxes map[string]string, esc *escritorImaxesCrono) string {
	var b strings.Builder
	for i := 0; i < len(bloques); {
		bl := bloques[i]
		switch bl.Tipo {
		case cronoItem:
			// Agrupa os elementos consecutivos do mesmo tipo nunha soa lista.
			j := i
			for j < len(bloques) && bloques[j].Tipo == cronoItem && bloques[j].Ordenada == bl.Ordenada {
				j++
			}
			env := "itemize"
			if bl.Ordenada {
				env = "enumerate"
			}
			b.WriteString(`\begin{` + env + "}\n")
			for _, it := range bloques[i:j] {
				b.WriteString(`\item ` + latexTramosCrono(it.Tramos) + "\n")
			}
			b.WriteString(`\end{` + env + "}\n")
			i = j
			continue
		case cronoTitulo:
			tam := map[int]string{1: `\large`, 2: `\normalsize`, 3: `\small`}[bl.Nivel]
			b.WriteString(`\par\smallskip` + envolverAlinCrono(bl.Alin, `{`+tam+`\bfseries `+latexTramosCrono(bl.Tramos)+`}`))
			b.WriteString(`\nopagebreak[3]` + "\n")
		case cronoCita:
			b.WriteString(`\begin{quote}\itshape ` + latexTramosCrono(bl.Tramos) + `\end{quote}` + "\n")
		case cronoDebuxo:
			b.WriteString("\\par\\medskip\n\\begin{center}\n" + envolverDebuxo(bl.Debuxo, bl.Codigo) + "\n\\end{center}\n\\medskip\n")
		case cronoImaxe:
			im := esc.resolver(bl.Imaxe.Ref, imaxes)
			if im == nil {
				break
			}
			w, h := bl.Imaxe.tamanoMM(larguraColumnaMMCrono, im.W, im.H)
			if w <= 0 || h <= 0 {
				break
			}
			pos := "c"
			switch bl.Alin {
			case cronoAlinEsquerda:
				pos = "l"
			case cronoAlinDereita:
				pos = "r"
			}
			b.WriteString(`\par\medskip\noindent\makebox[\linewidth][` + pos + `]{\includegraphics[width=` +
				mmLatexCrono(w) + `,height=` + mmLatexCrono(h) + `]{` + im.Ficheiro + `}}\par\medskip` + "\n")
		default:
			b.WriteString(envolverAlinCrono(bl.Alin, latexTramosCrono(bl.Tramos)))
		}
		i++
	}
	return b.String()
}

// preambuloInformeCrono monta o preámbulo segundo o motor. O de Lua para
// exames carga paquetes de gráficos (circuitikz, gnuplot...) que un informe
// non precisa: aquí só o necesario, así compila máis rápido e falla menos.
// `logo` é o logotipo do curso (nil se non ten), que vai no macro \logocentro;
// `plantilla` pon o deseño (Estilo) que se aplica ao documento.
func preambuloInformeCrono(engine string, tip *TipografiaOpts, logo *imaxeCrono, plantilla *PlantillaLatex) string {
	return preambuloInformeCronoCon(engine, tip, logo, plantilla, "")
}

// preambuloInformeCronoCon é preambuloInformeCrono con `extra` (LaTeX) despois
// do deseño: os documentos que levan macros propios (os formularios) engádeos
// aí sen que cronograma nin actas cambien.
func preambuloInformeCronoCon(engine string, tip *TipografiaOpts, logo *imaxeCrono, plantilla *PlantillaLatex, extra string) string {
	return preambuloDeseno(engine, compoñerDeseno(engine, tip, logo, plantilla), extra)
}

// preambuloDeseno é o preámbulo de Lúa co deseño xa resolto (dd).
func preambuloDeseno(engine string, dd desenoDoc, extra string) string {
	var p strings.Builder
	p.WriteString("\\documentclass[11pt,a4paper]{article}\n")
	if engineUnicodeNativo(engine) {
		p.WriteString("\\usepackage{fontspec}\n")
	} else {
		p.WriteString("\\usepackage[utf8]{inputenc}\n\\usepackage[T1]{fontenc}\n")
		p.WriteString(latexUnicodeDeclarations)
	}
	p.WriteString(babelLine())
	// Tamaño, interliñado e letras de todos os papeis (deseno_latex.go).
	p.WriteString(dd.Preambulo)
	fmt.Fprintf(&p, "\\usepackage[a4paper,margin=%smm]{geometry}\n", trimFloat(dd.Marxe))
	p.WriteString(`\usepackage{graphicx}
\usepackage[table]{xcolor}
\usepackage{array,longtable}
\usepackage{enumitem}
\usepackage{tikz}
\usepackage{circuitikz}
\usetikzlibrary{babel}
\usepackage{titlesec}
\usepackage{amsmath}
\usepackage{amssymb}
\usepackage{fancyhdr}
\usepackage[hidelinks]{hyperref}
`)
	p.WriteString(dd.Bloque)
	p.WriteString(extra)
	return p.String()
}

// paquetesInformeCrono son os paquetes que precisa o informe. Só se usan
// cando unha plantilla COMPLETA (co seu propio \documentclass) non os carga
// ela (ver inxectarPreambuloObrigatorio); no preámbulo de Lua van directos.
var paquetesInformeCrono = []paqueteRequirido{
	{Nome: "graphicx", Linha: `\usepackage{graphicx}`},
	{Nome: "xcolor", Linha: `\usepackage{xcolor}`},
	{Nome: "colortbl", Linha: `\usepackage{colortbl}`}, // \cellcolor e cor das liñas das táboas
	{Nome: "array", Linha: `\usepackage{array}`},
	{Nome: "longtable", Linha: `\usepackage{longtable}`},
	{Nome: "enumitem", Linha: `\usepackage{enumitem}`},
	{Nome: "tikz", Linha: `\usepackage{tikz}`},
	{Nome: "circuitikz", Linha: `\usepackage{circuitikz}`}, // os circuítos dos problemas
	{Nome: "titlesec", Linha: `\usepackage{titlesec}`},
	{Nome: "amsmath", Linha: `\usepackage{amsmath}`}, // as fórmulas dos problemas (\dfrac, align...)
	{Nome: "amssymb", Linha: `\usepackage{amssymb}`},
	{Nome: "hyperref", Linha: `\usepackage[hidelinks]{hyperref}`},
	{Nome: "fancyhdr", Linha: `\usepackage{fancyhdr}`}, // a cabeceira e o pé do deseño
}

// latexInformeCrono compón o documento LaTeX completo do informe e devolve
// tamén os avisos (imaxes descartadas, variables da plantilla sen valor...).
// As imaxes escríbense en `dir`. Con `plantilla` (non nil) o informe vai
// envolto na súa maqueta e coa súa portada e sinatura (ver
// documentoLatexConPlantilla); `nomeDoc` enche {{FICHEIRO}}. O único erro
// posible é unha plantilla sen {{CORPO}}.
func latexInformeCrono(p PlanCrono, engine string, tip *TipografiaOpts, dir string, agora time.Time, plantilla *PlantillaLatex, nomeDoc string) (string, []string, error) {
	esc := novoEscritorImaxesCrono(dir)
	meses := calendarioCrono(p.Config, p.MonthsData)
	conTarefas := mesesConTarefasCrono(meses)
	nomesInforme := nomesMesesInformeCrono(conTarefas)
	escapar := EscapeLatexProse
	logo := esc.escribir(p.Config.SchoolLogo, "logotipo")
	auto := variablesCorpoCrono(p, nomeDoc, agora)
	tx := novosTextosDoc(plantilla, auto)
	tx.engadirVariables(auto, "TITULO_CRONOGRAMA")

	// b é o CORPO (índice, introdución, cronograma e desenvolvemento): a
	// portada e a sinatura vanse a parte, ver partesInformeCrono.
	var b strings.Builder
	dd := compoñerDeseno(engine, tip, logo, plantilla)
	if engineUnicodeNativo(engine) {
		if err := escribirFontesInformeCrono(dir, dd.Fontes); err != nil {
			esc.avisos = append(esc.avisos, "non se puido escribir a fonte do cronograma: "+err.Error())
		}
	}
	b.WriteString(babelListquotFix)
	b.WriteString(formulasDefLatexPorDefecto)
	b.WriteString(dd.Corpo)
	b.WriteString("\\renewcommand{\\contentsname}{" + tx.L("INDICE") + "}\n")

	// --- Índice ---
	b.WriteString("\\tableofcontents\n\\clearpage\n")

	// --- 1. Introdución ---
	intro, avisosIntro := introCrono(p, plantilla, auto)
	b.WriteString("\\section{" + tx.L("INTRODUCION") + "}\n")
	b.WriteString(envolverAlinCrono(cronoAlinXustificado, escapar(intro)))
	b.WriteString("\\clearpage\n")

	// --- 2. Cronograma anual: só o cronograma ---
	b.WriteString("\\section{" + tx.L("CRONOGRAMA") + "}\n")
	if len(meses) == 0 {
		b.WriteString("\\textit{" + tx.L("SEN_MESES") + "}\n")
	} else {
		b.WriteString(`\arrayrulecolor{gris-linea}\setlength{\meswidth}{0pt}` + "\n")
		// Mide todas as etiquetas dos meses antes de nada: a columna do mes é
		// tan ancha coma a máis larga, igual en todos os trimestres.
		for _, m := range meses {
			b.WriteString(`\medirmes{` + escapar(m.Nome) + `}{` + escapar(subtituloMesCrono(tx, len(m.Tarefas))) + "}\n")
		}
		// Un bloque por trimestre. Cada trimestre é unha caixa que non se parte
		// (ambiente trimestre, ver estilo_doc.go): se non cabe no que queda de
		// páxina pasa enteiro á seguinte, así nunca se corta un trimestre polo
		// medio e cada páxina leva os que collen.
		for _, g := range trimestresCrono(meses) {
			b.WriteString(`\begin{trimestre}{` + tx.L("TRIMESTRE", map[string]string{"N": strconv.Itoa(g.Numero)}) + `}{` + escapar(g.Meses) + "}\n")
			// Dúas columnas p{} (non m{}): a etiqueta do mes e a primeira liña de
			// píldoras comparten liña de base, e as dúas celas empezan co mesmo
			// puntal para que todas as filas teñan a mesma altura mínima.
			b.WriteString(`\begin{tabular}{@{}p{\meswidth}@{\hspace{2.5mm}}>{\raggedright\arraybackslash}p{\dimexpr\linewidth-\meswidth-2.5mm\relax}@{}}` + "\n")
			for _, m := range g.Mesos {
				// Puntal de 13.5 mm: a etiqueta do mes (9.5 mm) e 2 mm de aire por riba
				// e por baixo, así entre dous meses queda unha pequena separación.
				b.WriteString(`\rule[-6.75mm]{0pt}{13.5mm}`)
				if n := len(m.Tarefas); n == 0 {
					b.WriteString(fmt.Sprintf(`\mesvacio{%d}{%s}`, m.Mes, escapar(m.Nome)))
				} else {
					b.WriteString(fmt.Sprintf(`\mesbadge{%d}{%s}{%s}`, m.Mes, escapar(m.Nome), escapar(subtituloMesCrono(tx, n))))
				}
				b.WriteString(` & \baselineskip=7.4mm\lineskiplimit=-20mm\relax\rule[-6.75mm]{0pt}{13.5mm}`)
				if len(m.Tarefas) == 0 {
					b.WriteString(`{\fontchip\fontsize{7}{8}\selectfont\itshape\color{black!40} ` + tx.L("SEN_TAREFAS") + `}`)
				}
				for j, t := range m.Tarefas {
					if j > 0 {
						b.WriteString(`\hspace{2mm}`)
					}
					b.WriteString(`\chip{` + resolverCorCrono(t.Color).Clave + `}{` + escapar(porDefectoCrono(t.Title, tx.T("SEN_TITULO"))) + `}`)
				}
				b.WriteString("\\\\\n\\hline\n")
			}
			b.WriteString("\\end{tabular}\n\\end{trimestre}\n")
		}
	}
	b.WriteString("\\clearpage\n")

	// --- 3. Desenvolvemento por meses ---
	b.WriteString("\\section{" + tx.L("DESENVOLVEMENTO") + "}\n")
	if len(conTarefas) == 0 {
		b.WriteString("\\textit{" + tx.L("SEN_TAREFAS_CURSO") + "}\n")
	}
	for i, m := range conTarefas {
		b.WriteString(`\subsection{` + escapar(nomesInforme[i]) + "}\n")
		for _, t := range m.Tarefas {
			b.WriteString(`\tarefa{` + resolverCorCrono(t.Color).Clave + `}{` + escapar(porDefectoCrono(t.Title, tx.T("SEN_TITULO"))) + "}\n")
			bloques := parseMarkdownCrono(t.Description)
			if len(bloques) == 0 {
				b.WriteString(`{\itshape\color{gray} ` + tx.L("SEN_DESCRICION") + `}` + "\n\n")
			} else {
				b.WriteString(latexBloquesCrono(bloques, t.Images, esc))
			}
		}
	}

	doc, avisosPlantilla, err := documentoLatexConPlantilla(
		preambuloDeseno(engine, dd, ""),
		partesInformeCrono{PortadaDef: dd.PortadaDef, CabeceiraPe: dd.CabeceiraPe, Corpo: b.String(), SinaturaDef: sinaturaPorDefectoLatex, SenPortada: dd.SenPortada},
		engine, plantilla, auto, paquetesInformeCrono, dd.BloqueCompleta)
	if err != nil {
		return "", nil, err
	}
	return doc, engadirSenRepetir(esc.avisos, avisosIntro, avisosPlantilla, dd.Avisos), nil
}

// variablesCorpoCrono son as variables automáticas máis un valor de mostra
// para os datos do centro que o curso deixe en branco ("Nome do centro"...),
// para que a portada e a sinatura non saian con ocos.
func variablesCorpoCrono(p PlanCrono, nomeDoc string, agora time.Time) map[string]string {
	v := variablesAutomaticasCrono(p, nomeDoc, agora)
	for nome, defecto := range map[string]string{
		"CENTRO": "Nome do centro", "LOCALIDADE": "Localidade",
		"ORIENTADOR": "Nome e apelidos", "XEFE": "Nome e apelidos",
	} {
		if strings.TrimSpace(v[nome]) == "" {
			v[nome] = defecto
		}
	}
	return v
}

// ---------- Compilación ----------

// executarLatexCrono lanza o motor unha vez sobre dir/doc.tex.
func executarLatexCrono(dir, engine string) (string, error) {
	args := []string{"-interaction=nonstopmode", "-halt-on-error", "doc.tex"}
	if motorMiKTeX(engine) {
		args = append([]string{"--enable-installer"}, args...)
	}
	// Tempo máximo e cancelación (compilacion_control.go).
	ctx, cancel := contextoCompilacion(dir)
	defer cancel()
	cmd := exec.CommandContext(ctx, engine, args...)
	cmd.Dir = dir
	ocultarConsola(cmd)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if e := erroContexto(ctx); e != nil {
		return out.String(), e
	}
	return out.String(), err
}

// motoresMiKTeX lembra, por motor, se é de MiKTeX (a resposta non cambia
// mentres Lua está aberto).
var motoresMiKTeX sync.Map

// motorMiKTeX indica se o motor LaTeX é de MiKTeX. MiKTeX instala os paquetes
// que faltan, pero por defecto PREGUNTA cunha xanela; lanzado por Lua (sen
// consola) esa xanela non se pode abrir («GUI framework cannot be
// initialized») e o motor morre con «exit status 1» antes de escribir no log
// o «File `X.sty' not found» co que Lua ofrecería instalalo. Con
// --enable-installer instala sen preguntar. TeX Live non coñece esa opción.
func motorMiKTeX(engine string) bool {
	if v, ok := motoresMiKTeX.Load(engine); ok {
		return v.(bool)
	}
	cmd := exec.Command(engine, "--version")
	ocultarConsola(cmd)
	out, _ := cmd.Output()
	e := strings.Contains(string(out), "MiKTeX")
	motoresMiKTeX.Store(engine, e)
	return e
}

// compilarLatexCrono escribe o .tex e compílao dúas veces (a primeira
// enche o índice, a segunda pon os números de páxina), devolvendo o PDF e o
// log da última pasada. Se a segunda falla, vale o PDF da primeira.
func compilarLatexCrono(dir, tex, engine string) (pdf []byte, log string, err error) {
	// Ficheiros auxiliares dun intento anterior con outro motor: confundirían
	// ao novo.
	for _, ext := range []string{"aux", "toc", "out", "log", "pdf"} {
		_ = os.Remove(filepath.Join(dir, "doc."+ext))
	}
	if err := os.WriteFile(filepath.Join(dir, "doc.tex"), []byte(tex), 0o644); err != nil {
		return nil, "", err
	}
	var primeiro []byte
	for pasada := 0; pasada < 2; pasada++ {
		l, runErr := executarLatexCrono(dir, engine)
		if runErr != nil && compilacionParada(runErr) {
			// Cancelada ou parada por tempo: non ten sentido outra pasada.
			return nil, l, runErr
		}
		data, readErr := os.ReadFile(filepath.Join(dir, "doc.pdf"))
		if readErr != nil || runErr != nil {
			if pasada == 1 && primeiro != nil {
				return primeiro, l, nil
			}
			if pista := pistaLatexAccionable(l); pista != "" {
				return nil, l, fmt.Errorf("%s: %s", pista, extraerErroPDFLatex(l))
			}
			if detalle := extraerErroPDFLatex(l); detalle != "" {
				return nil, l, fmt.Errorf("%s fallou: %s", engine, detalle)
			}
			if runErr != nil {
				return nil, l, fmt.Errorf("%s fallou: %w", engine, runErr)
			}
			return nil, l, readErr
		}
		primeiro, log = data, l
	}
	pdf, _ = os.ReadFile(filepath.Join(dir, "doc.pdf"))
	if len(pdf) == 0 {
		pdf = primeiro
	}
	return pdf, log, nil
}

// resultadoInformeCrono é o que devolve compilarInformeCrono.
type resultadoInformeCrono struct {
	PDF    []byte
	Tex    string
	Log    string
	Motor  string
	Avisos []string
}

// compilarInformeCrono compila o informe co motor preferido (ou o que esixa a
// plantilla) e, se falla, cos outros instalados (mesma idea ca antes: un
// pdflatex que non entende un carácter Unicode non debe deixar o informe sen
// xerar). As imaxes da plantilla cópianse a carón do doc.tex.
func compilarInformeCrono(p PlanCrono, preferido string, tip *TipografiaOpts, dir string, plantilla *PlantillaLatex, nomeDoc string) (resultadoInformeCrono, error) {
	if msg := validarDatasCrono(p.Config.StartDate, p.Config.EndDate); msg != "" {
		return resultadoInformeCrono{}, fmt.Errorf("%s", msg)
	}
	return compilarComMotores(preferido, tip, dir, plantilla, func(motor string) (string, []string, error) {
		return latexInformeCrono(p, motor, tip, dir, time.Now(), plantilla, nomeDoc)
	})
}

// compilarComMotores é o núcleo compartido polo cronograma e polas actas:
// escolle o motor (o da plantilla, o de Opcións ou xelatex se cómpre), copia as
// imaxes da plantilla, e proba cada motor instalado ata que un compila.
// `xerar(motor)` devolve o .tex para ese motor (e os seus avisos).
func compilarComMotores(preferido string, tip *TipografiaOpts, dir string, plantilla *PlantillaLatex, xerar func(motor string) (string, []string, error)) (resultadoInformeCrono, error) {
	res := resultadoInformeCrono{}
	preferido = strings.TrimSpace(motorParaPlantilla(plantilla, preferido))
	if preferido == "" {
		preferido = "pdflatex"
	}
	// Unha fonte do sistema (Axustes do texto) só a pode usar fontspec.
	if tip != nil && esFonteDoSistema(tip.Fonte) && !engineUnicodeNativo(preferido) {
		preferido = "xelatex"
	}
	// As etiquetas usan Open Sans (fontspec): se a plantilla non fixa o motor,
	// o documento compílase con xelatex sempre que o equipo o teña.
	if (plantilla == nil || strings.TrimSpace(plantilla.Motor) == "") && !engineUnicodeNativo(preferido) && commandExists("xelatex") {
		preferido = "xelatex"
	}
	candidatos := motoresParaProbar(preferido)
	if len(candidatos) == 0 {
		pacote := "texlive-latex-base"
		switch preferido {
		case "xelatex":
			pacote = "texlive-xetex"
		case "lualatex":
			pacote = "texlive-luatex"
		}
		return res, idioma.Errorf("go.latex.motor_falta", "non se atopou %s. Instala unha distribución LaTeX (p.ex. 'sudo apt install %s')", preferido, pacote)
	}
	// As imaxes (logotipos) da plantilla van a carón do doc.tex: así o
	// \includegraphics{logotipo.png} que escribiu quen fixo a plantilla
	// atópaas sen rutas. Ver copiarImaxesPlantilla.
	if err := copiarImaxesPlantilla(plantilla, dir); err != nil {
		return res, idioma.Errorf("go.latex.imaxes_plantilla", "non se puideron copiar as imaxes da plantilla: %w", err)
	}

	var primeiroErro error
	for i, motor := range candidatos {
		tex, avisos, err := xerar(motor)
		if err != nil {
			return res, err // a plantilla é inválida: outro motor non o arranxa
		}
		pdf, log, err := compilarLatexCrono(dir, tex, motor)
		if err != nil && compilacionParada(err) {
			return res, err // cancelada ou parada por tempo: outro motor non o arranxa
		}
		if i == 0 {
			res.Tex, res.Log, res.Avisos = tex, log, avisos
		}
		if err == nil {
			res.PDF, res.Tex, res.Log, res.Motor, res.Avisos = pdf, tex, log, motor, avisos
			if motor != preferido {
				res.Avisos = append(res.Avisos, idioma.T("go.latex.motor_cambiado", "%s non conseguiu compilar este informe; usouse %s automaticamente. Podes cambiar o motor por defecto en Opcións.", preferido, motor))
			}
			return res, nil
		}
		if i == 0 {
			primeiroErro = err
		}
	}
	return res, fmt.Errorf("erro ao compilar o informe: %w", primeiroErro)
}

// xerarPDFCrono é o núcleo compartido por XerarInformeCronograma e
// PrevisualizarPlantilla: compila o informe en `dir` e devolve o PDF, o .tex,
// o log e as páxinas coma imaxes (o mesmo formato para o visor da
// previsualización).
func (a *App) xerarPDFCrono(plan PlanCrono, tip *TipografiaOpts, plantilla *PlantillaLatex, nomeDoc, dir string) (GeneratePDFResult, error) {
	return a.xerarPDFPedido(XerarInformeCronogramaRequest{Plan: plan, Tipografia: tip, NomeBase: nomeDoc}, plantilla, dir)
}

// xerarPDFPedido compila o documento que pide `req` (cronograma ou actas).
func (a *App) xerarPDFPedido(req XerarInformeCronogramaRequest, plantilla *PlantillaLatex, dir string) (GeneratePDFResult, error) {
	req = a.conMotorProblemas(req)
	var res resultadoInformeCrono
	var err error
	if req.eFormulario() {
		res, err = compilarFormulario(*req.Formulario, a.settings.LatexEngine, req.Tipografia, dir, plantilla, req.NomeBase)
	} else if req.eActas() {
		res, err = compilarInformeActas(*req.Actas, req.Ambito, a.settings.LatexEngine, req.Tipografia, dir, plantilla, req.NomeBase)
	} else {
		res, err = compilarInformeCrono(req.Plan, a.settings.LatexEngine, req.Tipografia, dir, plantilla, req.NomeBase)
	}
	rexistrarCompilacion(req, res, err) // para que o asistente o poida ler (rexistro.go)
	result := GeneratePDFResult{}
	result.Log = res.Log
	result.LatexSource = res.Tex
	result.Warnings = append(result.Warnings, res.Avisos...)
	if err != nil {
		// Se o que faltou foi un paquete de LaTeX déixase anotado para que o
		// frontend poida ofrecer instalalo (ver paquetelatex.go): o log non
		// chega ao frontend cando isto devolve erro.
		anotarPaqueteQueFalta(res.Log)
		return result, err
	}
	limparPaqueteQueFalta()
	result.PDFBase64 = base64.StdEncoding.EncodeToString(res.PDF)
	// Inspector da vista previa e corrector (deseno_marcas.go).
	result.Marcas = lerMarcasDeseno(filepath.Join(dir, "doc.lpos"))
	if aviso := avisoMarxe(res.Log); aviso != "" {
		result.Warnings = append(result.Warnings, aviso)
	}

	if commandExists("pdftoppm") {
		imaxes, imgErr := pdfPageImages(dir, filepath.Join(dir, "doc.pdf"))
		if imgErr != nil {
			result.Warnings = append(result.Warnings, idioma.T("go.latex.vista_previa", "non se puido xerar a vista previa das páxinas: %v", imgErr))
		} else {
			result.PageImages = imaxes
		}
	} else {
		result.Warnings = append(result.Warnings, "non se atopou pdftoppm (paquete poppler-utils): sen vista previa páxina a páxina")
	}
	return result, nil
}

// XerarInformeCronograma compila o documento (o cronograma ou, se a petición é
// de actas, as actas) coa plantilla activa e devolve o PDF e as páxinas como
// imaxes para a previsualización. Tamén deixa o resultado gardado para a
// xanela desacoplada (ver xanelaresultado.go), que se actualiza soa se está
// aberta.
func (a *App) XerarInformeCronograma(req XerarInformeCronogramaRequest) (GeneratePDFResult, error) {
	dir, err := os.MkdirTemp("", "lua-informe-*")
	if err != nil {
		return GeneratePDFResult{}, err
	}
	defer os.RemoveAll(dir)

	plantilla := a.plantillaActiva()
	result, err := a.xerarPDFPedido(req, plantilla, dir)
	if err != nil {
		return result, err
	}
	a.publicarResultado(req, plantilla, result)
	return result, nil
}

// exportarTexCrono garda o .tex do informe, co seu cartafol images/ á beira
// (o logotipo e as imaxes das tarefas) e as imaxes da plantilla, para que o
// ficheiro compile solto. Non compila nada: só compón o LaTeX.
func (a *App) exportarTexCrono(req XerarInformeCronogramaRequest, plantilla *PlantillaLatex, defaultName string) (GenerateDocResult, error) {
	req = a.conMotorProblemas(req)
	result := GenerateDocResult{}
	dir, base := splitDialogDefault(defaultName)
	path, err := saveFileDialogCompat("Exportar código LaTeX", dir, base, "LaTeX (*.tex)", "*.tex")
	if err != nil || path == "" {
		return result, err
	}
	motor := strings.TrimSpace(motorParaPlantilla(plantilla, a.settings.LatexEngine))
	if motor == "" {
		motor = "pdflatex"
	}
	var tex string
	var avisos []string
	if req.eFormulario() {
		tex, avisos, err = latexFormulario(*req.Formulario, motor, req.Tipografia, filepath.Dir(path), time.Now(), plantilla, req.NomeBase)
	} else if req.eActas() {
		tex, avisos, err = latexInformeActas(*req.Actas, req.Ambito, motor, req.Tipografia, filepath.Dir(path), time.Now(), plantilla, req.NomeBase)
	} else {
		tex, avisos, err = latexInformeCrono(req.Plan, motor, req.Tipografia, filepath.Dir(path), time.Now(), plantilla, req.NomeBase)
	}
	if err != nil {
		return result, err
	}
	result.Warnings = avisos
	if err := escribirImaxesPlantilla(plantilla, filepath.Dir(path)); err != nil {
		return result, idioma.Errorf("go.latex.imaxes_plantilla", "non se puideron copiar as imaxes da plantilla: %w", err)
	}
	if err := os.WriteFile(path, []byte(tex), 0o644); err != nil {
		return result, err
	}
	result.Path = path
	return result, nil
}

// ExportarInformeTex garda o .tex do informe coa plantilla activa.
func (a *App) ExportarInformeTex(req XerarInformeCronogramaRequest, defaultName string) (GenerateDocResult, error) {
	return a.exportarTexCrono(req, a.plantillaActiva(), defaultName)
}

// ---------- Markdown / Word / OpenDocument ----------

// mdBloquesCrono converte os bloques dunha descrición en Markdown. `nivelBase`
// desprace os títulos (#, ##, ###) por debaixo do título da tarefa.
func mdBloquesCrono(bloques []cronoBloque, imaxes map[string]string, esc *escritorImaxesCrono, nivelBase int) string {
	var b strings.Builder
	for i := 0; i < len(bloques); {
		bl := bloques[i]
		switch bl.Tipo {
		case cronoItem:
			j := i
			for j < len(bloques) && bloques[j].Tipo == cronoItem && bloques[j].Ordenada == bl.Ordenada {
				j++
			}
			for _, it := range bloques[i:j] {
				marca := "- "
				if bl.Ordenada {
					marca = "1. "
				}
				b.WriteString(marca + mdTramosCrono(it.Tramos) + "\n")
			}
			b.WriteString("\n")
			i = j
			continue
		case cronoTitulo:
			b.WriteString(strings.Repeat("#", nivelBase+bl.Nivel-1) + " " + mdTramosCrono(bl.Tramos) + "\n\n")
		case cronoCita:
			b.WriteString("> " + mdTramosCrono(bl.Tramos) + "\n\n")
		case cronoDebuxo:
			// LaTeX cru (Pandoc pásao tal cal ao PDF).
			b.WriteString("```{=latex}\n\\begin{center}\n" + envolverDebuxo(bl.Debuxo, bl.Codigo) + "\n\\end{center}\n```\n\n")
		case cronoImaxe:
			im := esc.resolver(bl.Imaxe.Ref, imaxes)
			if im == nil {
				break
			}
			w, h := bl.Imaxe.tamanoMM(larguraColumnaMMCrono, im.W, im.H)
			if w <= 0 || h <= 0 {
				break
			}
			b.WriteString("![" + mdEscaparCrono(bl.Imaxe.Alt) + "](" + im.Ficheiro + "){width=" + mmLatexCrono(w) + " height=" + mmLatexCrono(h) + "}\n\n")
		default:
			b.WriteString(mdTramosCrono(bl.Tramos) + "\n\n")
		}
		i++
	}
	return b.String()
}

// saltoPaxinaMdCrono é o salto entre portada, índice e seccións. Só Word ten
// un salto de páxina que Pandoc poida emitir sen máis; nos outros formatos
// pónse unha liña horizontal.
func saltoPaxinaMdCrono(formato string) string {
	if formato == "docx" {
		return "```{=openxml}\n<w:p><w:r><w:br w:type=\"page\"/></w:r></w:p>\n```\n\n"
	}
	return "---\n\n"
}

// markdownInformeCrono compón o informe en Markdown. `formato` é "md",
// "docx" ou "odt" (só cambia o xeito de separar páxinas). As imaxes escríbense
// en `dir`/images.
func markdownInformeCrono(p PlanCrono, formato, dir string, agora time.Time, plantilla *PlantillaLatex) (string, []string) {
	esc := novoEscritorImaxesCrono(dir)
	meses := calendarioCrono(p.Config, p.MonthsData)
	conTarefas := mesesConTarefasCrono(meses)
	nomesInforme := nomesMesesInformeCrono(conTarefas)
	centro := porDefectoCrono(p.Config.SchoolName, "Nome do centro")
	localidade := porDefectoCrono(p.Config.Location, "Localidade")
	data := dataGalegaCrono(agora)
	salto := saltoPaxinaMdCrono(formato)
	vars := variablesCorpoCrono(p, "", agora)
	tx := novosTextosDoc(plantilla, vars)
	tx.engadirVariables(vars, "TITULO_CRONOGRAMA")
	intro, avisosIntro := introCrono(p, plantilla, vars)

	var b strings.Builder
	if im := esc.escribir(p.Config.SchoolLogo, "logotipo"); im != nil {
		b.WriteString("![Logotipo](" + im.Ficheiro + "){width=45mm}\n\n")
	}
	b.WriteString("# " + mdEscaparCrono(strings.ToUpper(vars["TITULO"])) + "\n\n")
	b.WriteString("**" + mdEscaparCrono(vars["DEPARTAMENTO"]) + "**\n\n")
	b.WriteString("**" + mdEscaparCrono(centro) + "**\n\n")
	b.WriteString(mdEscaparCrono(tx.T("CURSO_ACADEMICO")) + "\n\n")
	b.WriteString(mdEscaparCrono(localidade+", "+data) + "\n\n")
	b.WriteString(salto)

	b.WriteString("# " + mdEscaparCrono(tx.T("INDICE")) + "\n\n")
	b.WriteString("1. " + mdEscaparCrono(tx.T("INTRODUCION")) + "\n2. " + mdEscaparCrono(tx.T("CRONOGRAMA")) + "\n3. " + mdEscaparCrono(tx.T("DESENVOLVEMENTO")) + "\n")
	for i := range conTarefas {
		b.WriteString(fmt.Sprintf("    - 3.%d. %s\n", i+1, mdEscaparCrono(nomesInforme[i])))
	}
	b.WriteString("\n")
	b.WriteString(salto)

	b.WriteString("# 1. " + mdEscaparCrono(tx.T("INTRODUCION")) + "\n\n" + mdEscaparCrono(intro) + "\n\n")
	b.WriteString(salto)

	// O cronograma anual é só a imaxe (a mesma que vai a Word e OpenDocument).
	b.WriteString("# 2. " + mdEscaparCrono(tx.T("CRONOGRAMA")) + "\n\n")
	if len(meses) == 0 {
		b.WriteString("*" + mdEscaparCrono(tx.T("SEN_MESES")) + "*\n\n")
	} else if png, err := renderCronogramaPNGCrono(p, meses, tx); err != nil {
		esc.avisos = append(esc.avisos, "non se puido debuxar o cronograma anual: "+err.Error())
	} else if err := os.MkdirAll(filepath.Join(dir, "images"), 0o755); err != nil {
		esc.avisos = append(esc.avisos, "non se puido crear o cartafol de imaxes: "+err.Error())
	} else if err := os.WriteFile(filepath.Join(dir, "images", "cronograma.png"), png, 0o644); err != nil {
		esc.avisos = append(esc.avisos, "non se puido escribir a imaxe do cronograma: "+err.Error())
	} else {
		b.WriteString("![" + mdEscaparCrono(tx.T("CRONOGRAMA")) + "](images/cronograma.png)\n\n")
	}
	b.WriteString(salto)

	b.WriteString("# 3. " + mdEscaparCrono(tx.T("DESENVOLVEMENTO")) + "\n\n")
	if len(conTarefas) == 0 {
		b.WriteString("*" + mdEscaparCrono(tx.T("SEN_TAREFAS_CURSO")) + "*\n\n")
	}
	for i, m := range conTarefas {
		b.WriteString(fmt.Sprintf("## 3.%d. %s\n\n", i+1, mdEscaparCrono(nomesInforme[i])))
		for _, t := range m.Tarefas {
			b.WriteString("### " + mdEscaparCrono(porDefectoCrono(t.Title, tx.T("SEN_TITULO"))) + "\n\n")
			bloques := parseMarkdownCrono(t.Description)
			if len(bloques) == 0 {
				b.WriteString("*" + mdEscaparCrono(tx.T("SEN_DESCRICION")) + "*\n\n")
			} else {
				b.WriteString(mdBloquesCrono(bloques, t.Images, esc, 4))
			}
		}
	}
	b.WriteString(salto)

	// Peche: lugar e data, un espazo para asinar, unha liña e o nome e o cargo.
	b.WriteString(mdEscaparCrono(localidade+", "+data) + "\n\n\n\n")
	b.WriteString("\\_\\_\\_\\_\\_\\_\\_\\_\\_\\_\\_\\_\\_\\_\\_\\_\\_\\_\\_\\_\\_\\_\\_\\_\\_\\_\n\n")
	b.WriteString("**" + mdEscaparCrono(porDefectoCrono(p.Config.HeadOfDepartment, "Nome e apelidos")) + "**  \n")
	b.WriteString(mdEscaparCrono(cargoSinaturaCrono(p)) + "\n")
	return b.String(), engadirSenRepetir(esc.avisos, avisosIntro)
}

// markdownInformeConPlantilla compón o informe en Markdown e, se a plantilla
// ten parte Markdown, envólveo nela (cabeceira, pé...).
func markdownInformeConPlantilla(req XerarInformeCronogramaRequest, plantilla *PlantillaLatex, formato, dir string) (string, []string) {
	agora := time.Now()
	var md string
	var avisos []string
	var vars map[string]string
	if req.eFormulario() {
		var err error
		if md, avisos, err = markdownFormulario(*req.Formulario, formato, dir, agora, plantilla); err != nil {
			return "# " + mdEscaparCrono(err.Error()) + "\n", []string{err.Error()}
		}
		vars = variablesFormulario(normalizarFormulario(req.Formulario.Formulario), req.Formulario.Datos, req.NomeBase, agora, plantilla)
		novosTextosDoc(plantilla, vars).engadirVariables(vars, "TITULO_FORMULARIO")
	} else if req.eActas() {
		md, avisos = markdownInformeActas(*req.Actas, req.Ambito, formato, dir, agora, plantilla)
		vars = variablesActas(*req.Actas, req.NomeBase, agora)
		novosTextosDoc(plantilla, vars).engadirVariables(vars, "TITULO_ACTAS")
	} else {
		md, avisos = markdownInformeCrono(req.Plan, formato, dir, agora, plantilla)
		vars = variablesAutomaticasCrono(req.Plan, req.NomeBase, agora)
		novosTextosDoc(plantilla, vars).engadirVariables(vars, "TITULO_CRONOGRAMA")
	}
	md, avisosPlantilla := documentoMarkdownConPlantilla(md, plantilla, vars)
	return md, append(avisos, avisosPlantilla...)
}

// exportarMarkdownCrono garda o informe como .md co seu cartafol images/ á beira.
func (a *App) exportarMarkdownCrono(req XerarInformeCronogramaRequest, plantilla *PlantillaLatex, defaultName string) (GenerateMarkdownResult, error) {
	req = a.conMotorProblemas(req)
	result := GenerateMarkdownResult{}
	dir, base := splitDialogDefault(defaultName)
	path, err := saveFileDialogCompat("Exportar Markdown", dir, base, "Markdown (*.md)", "*.md")
	if err != nil || path == "" {
		return result, err
	}
	md, avisos := markdownInformeConPlantilla(req, plantilla, "md", filepath.Dir(path))
	result.Warnings = avisos
	if err := copiarImaxesPlantillaMarkdown(plantilla, filepath.Dir(path)); err != nil {
		return result, idioma.Errorf("go.latex.imaxes_plantilla", "non se puideron copiar as imaxes da plantilla: %w", err)
	}
	if err := os.WriteFile(path, []byte(md), 0o644); err != nil {
		return result, err
	}
	result.Path = path
	return result, nil
}

// ExportarInformeMarkdown garda o informe como .md coa plantilla activa.
func (a *App) ExportarInformeMarkdown(req XerarInformeCronogramaRequest, defaultName string) (GenerateMarkdownResult, error) {
	return a.exportarMarkdownCrono(req, a.plantillaActiva(), defaultName)
}

// exportarInformeOfimaticaCrono escribe o informe como Word ("docx") ou
// OpenDocument ("odt") co escritor nativo (informe_docx.go, informe_odt.go):
// sae xa cos estilos do informe (títulos, corpo, citas, portada, número de
// páxina), sen depender de Pandoc. A plantilla só achega o texto da
// introdución e as variables; a súa maqueta LaTeX/Markdown non se aplica.
func exportarInformeOfimaticaCrono(req XerarInformeCronogramaRequest, plantilla *PlantillaLatex, path, formato string) (GenerateDocResult, error) {
	result := GenerateDocResult{}
	datos, avisos, err := datosOfimaticaDe(req, plantilla, formato)
	if err != nil {
		return result, err
	}
	if err := escribirAtomicoCrono(path, datos); err != nil {
		return result, err
	}
	result.Warnings = avisos
	result.Path = path
	return result, nil
}

// datosOfimaticaDe compón o documento que pide `req` en Word ("docx") ou
// OpenDocument ("odt") e devolve os bytes e os avisos.
func datosOfimaticaDe(req XerarInformeCronogramaRequest, plantilla *PlantillaLatex, formato string) ([]byte, []string, error) {
	if formato != "docx" && formato != "odt" {
		return nil, nil, fmt.Errorf("formato non soportado: %s", formato)
	}
	var d datosOfimatica
	var err error
	if req.eFormulario() {
		d, err = prepararOfimaticaFormulario(req, plantilla, time.Now())
	} else if req.eActas() {
		d, err = prepararOfimaticaActas(req, plantilla, time.Now())
	} else {
		d, err = prepararOfimatica(req, plantilla, time.Now())
	}
	if err != nil {
		return nil, nil, err
	}
	var datos []byte
	switch {
	case formato == "docx" && req.eFormulario():
		datos, err = escribirDocxFormulario(&d)
	case req.eFormulario():
		datos, err = escribirOdtFormulario(&d)
	case formato == "docx" && req.eActas():
		datos, err = escribirDocxActas(&d)
	case formato == "docx":
		datos, err = escribirDocxCrono(&d)
	case req.eActas():
		datos, err = escribirOdtActas(&d)
	default:
		datos, err = escribirOdtCrono(&d)
	}
	return datos, d.Avisos, err
}

func (a *App) exportarDocxCrono(req XerarInformeCronogramaRequest, plantilla *PlantillaLatex, defaultName string) (GenerateDocResult, error) {
	req = a.conMotorProblemas(req)
	dir, base := splitDialogDefault(defaultName)
	path, err := saveFileDialogCompat("Exportar Word (.docx)", dir, base, "Word (*.docx)", "*.docx")
	if err != nil || path == "" {
		return GenerateDocResult{}, err
	}
	return exportarInformeOfimaticaCrono(req, plantilla, path, "docx")
}

func (a *App) exportarOdtCrono(req XerarInformeCronogramaRequest, plantilla *PlantillaLatex, defaultName string) (GenerateDocResult, error) {
	req = a.conMotorProblemas(req)
	dir, base := splitDialogDefault(defaultName)
	path, err := saveFileDialogCompat("Exportar OpenDocument (.odt)", dir, base, "OpenDocument (*.odt)", "*.odt")
	if err != nil || path == "" {
		return GenerateDocResult{}, err
	}
	return exportarInformeOfimaticaCrono(req, plantilla, path, "odt")
}

// ExportarInformeDocx garda o informe como Word (.docx) coa plantilla activa.
func (a *App) ExportarInformeDocx(req XerarInformeCronogramaRequest, defaultName string) (GenerateDocResult, error) {
	return a.exportarDocxCrono(req, a.plantillaActiva(), defaultName)
}

// ExportarInformeOdt garda o informe como OpenDocument (.odt) coa plantilla activa.
func (a *App) ExportarInformeOdt(req XerarInformeCronogramaRequest, defaultName string) (GenerateDocResult, error) {
	return a.exportarOdtCrono(req, a.plantillaActiva(), defaultName)
}

// planDemoCrono é o curso de mentira co que se previsualiza unha plantilla no
// seu editor: un centro inventado e tres meses con tarefas (unha con lista)
// para que se vexa como queda o informe dentro da maqueta. Datas fixas, non
// relativas a hoxe: a vista previa ten que ser sempre a mesma.
func planDemoCrono() PlanCrono {
	return PlanCrono{
		ID:           "demo",
		AcademicYear: "2025-2026",
		Config: ConfigCrono{
			StartDate:        "2025-09-01",
			EndDate:          "2025-11-30",
			SchoolName:       "Centro de exemplo",
			Location:         "Localidade",
			Department:       "Departamento de exemplo",
			HeadOfDepartment: "Nome e apelidos",
		},
		MonthsData: map[string]MesCrono{
			"2025-09": {Year: 2025, Month: 8, Tasks: []TarefaCrono{
				{ID: "d1", Title: "Acollida do alumnado", Color: "blue",
					Description: "Recepción do alumnado de novo ingreso e presentación do departamento.\n\n- Visita ás instalacións\n- Reunión coas familias"},
				{ID: "d2", Title: "Avaliación inicial", Color: "emerald", Description: "Recollida de datos do alumnado."},
			}},
			"2025-10": {Year: 2025, Month: 9, Tasks: []TarefaCrono{
				{ID: "d3", Title: "Plan de acción titorial", Color: "purple", Description: "Coordinación co profesorado titor."},
			}},
			"2025-11": {Year: 2025, Month: 10, Tasks: []TarefaCrono{
				{ID: "d4", Title: "Orientación académica e profesional", Color: "amber", Description: "Sesións con 4º da ESO e 2º de Bacharelato."},
			}},
		},
	}
}

// ---------- Ficheiros do curso ----------

// escribirAtomicoCrono escribe nun temporal e renomea: unha caída a medio
// gardar non deixa o curso do usuario cortado.
func escribirAtomicoCrono(path string, datos []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, datos, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// GardarBorradorInforme garda o curso en edición da ferramenta (para que non se
// perda ao pechar Lua sen gardar: mesma idea ca data.json da app orixinal). O
// frontend chámao con debounce ao editar.
func (a *App) GardarBorradorInforme(ferramenta, content string) error {
	path, err := borradorFerramentaPath(ferramenta)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := escribirAtomicoCrono(path, []byte(content)); err != nil {
		return err
	}
	// Se quedaba un borrador na localización histórica (con "Lua" en
	// maiúscula), xa non é o que se le: retírao para que non reapareza vello
	// se algún día se borra o novo.
	if dirProxectoActivo() == "" {
		eliminarBorradorVello(ferramenta, path)
	}
	return nil
}

// LerBorradorInforme devolve o último curso en edición da ferramenta, ou "" se
// non hai (ou non é JSON válido: un borrador corrupto non debe bloquear o
// arranque).
func (a *App) LerBorradorInforme(ferramenta string) (string, error) {
	path, err := borradorFerramentaPath(ferramenta)
	if err != nil {
		return "", nil
	}
	if datos, ok := lerBorradorDe(path); ok {
		return datos, nil
	}
	// Migración da localización histórica: se aínda non hai borrador no sitio
	// novo pero si no vello, lese e múdase. Nun sistema de ficheiros
	// insensible a maiúsculas as dúas rutas son o mesmo ficheiro, así que este
	// ramal non se alcanzaría; en Linux si, e é o caso que importa.
	if dirProxectoActivo() != "" {
		return "", nil // un proxecto non herda os borradores de antes dos proxectos
	}
	vello, err := borradorFerramentaPathVello(ferramenta)
	if err != nil {
		return "", nil
	}
	datos, ok := lerBorradorDe(vello)
	if !ok {
		return "", nil
	}
	migrarBorradorVello(vello, path)
	return datos, nil
}

// lerBorradorDe le un borrador e di se serve: un ficheiro que non existe ou
// que non é JSON válido non debe bloquear o arranque.
func lerBorradorDe(path string) (string, bool) {
	datos, err := os.ReadFile(path)
	if err != nil || !json.Valid(datos) {
		return "", false
	}
	return string(datos), true
}

// migrarBorradorVello move o borrador histórico á localización canónica.
// Mellor esforzo: se falla (permisos, ou as dúas rutas son o mesmo ficheiro
// nun sistema insensible a maiúsculas), o borrador xa lido úsase igual.
func migrarBorradorVello(vello, canonico string) {
	if err := os.MkdirAll(filepath.Dir(canonico), 0o755); err != nil {
		return
	}
	_ = os.Rename(vello, canonico)
}

// eliminarBorradorVello borra o borrador histórico despois de gardar o novo.
// Non fai nada se as dúas rutas apuntan ao MESMO ficheiro (macOS, Windows):
// alí <config>/Lua e <config>/lua son o mesmo cartafol, e borrar "o vello"
// borraría o que se acaba de escribir.
func eliminarBorradorVello(ferramenta, canonico string) {
	vello, err := borradorFerramentaPathVello(ferramenta)
	if err != nil {
		return
	}
	fiCanonico, errCanonico := os.Stat(canonico)
	fiVello, errVello := os.Stat(vello)
	if errCanonico != nil || errVello != nil || os.SameFile(fiCanonico, fiVello) {
		return
	}
	_ = os.Remove(vello)
}
