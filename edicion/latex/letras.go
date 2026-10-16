package latex

// Letras do documento por PAPEIS. Un documento ten catro papeis
// tipográficos e cada un leva unha soa familia:
//
//   - corpo: o texto (\rmfamily, \setmainfont);
//   - títulos: títulos e destacados (\sffamily, \setsansfont);
//   - etiquetas: etiquetas do cronograma, rótulos das actas (\fontchip);
//   - matemáticas: as fórmulas (unicode-math, \setmathfont).
//
// Antes só se cambiaba o corpo, e un documento con «Georgia» saía cos títulos
// en Latin Modern Sans e as etiquetas en Open Sans: tres letras mesturadas.
// Aquí unha COMBINACIÓN (Clásica, Elegante...) dá as catro de vez.
//
// As familias cárganse polo NOME DO FICHEIRO (TeX Live ou o cartafol fonts/
// que deixa Lúa ao carón do .tex), nunca polo nome do sistema: así o PDF sae
// igual en macOS, Windows e Linux. Se un ficheiro falta (un TeX Live mínimo),
// úsase a familia substituta e avísase.

import (
	"edicion/idioma"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Familia é unha familia de letra que Lúa sabe cargar.
type Familia struct {
	ID   string
	Nome string // como se amosa
	// Base e Ficheiros: os catro ficheiros (normal, negra, cursiva, negra
	// cursiva) para fontspec; "*" substitúese por Base.
	Base      string
	Ext       string
	Normal    string
	Negra     string
	Cursiva   string
	NegraCurs string
	// Ruta (opcional) é o Path= de fontspec (as fontes que leva Lúa en fonts/).
	Ruta string
	// NFSS é o código de familia para pdflatex (qtm, qhv...).
	NFSS string
	// Substituta é a familia que se usa se faltan os ficheiros.
	Substituta string
	// Serifa: true se é unha letra con serifa (para escoller a de Word).
	Serifa bool
	// Word é a letra máis parecida que adoita haber nun equipo con Word/LibreOffice.
	Word string
}

// FamiliaMates é unha letra matemática (OpenType MATH) para unicode-math.
type FamiliaMates struct {
	ID, Nome, Ficheiro string
	// Ruta: as que leva Lúa (fonts/ ao carón do .tex).
	Ruta string
}

// Combinacion é un xogo de letras con nome: o que escolle o profesorado.
type Combinacion struct {
	ID        string `json:"id"`
	Nome      string `json:"nome"`
	Corpo     string `json:"corpo"`
	Titulos   string `json:"titulos"`
	Etiquetas string `json:"etiquetas"`
	Mates     string `json:"mates"` // "" = Latin Modern (as de sempre)
	// Descricion é unha liña para a interface.
	Descricion string `json:"descricion"`
}

var familias = map[string]Familia{
	"termes":   gyre("termes", "TeX Gyre Termes (tipo Times)", "texgyretermes", "qtm", true, "Times New Roman"),
	"pagella":  gyre("pagella", "TeX Gyre Pagella (tipo Palatino)", "texgyrepagella", "qpl", true, "Palatino Linotype"),
	"schola":   gyre("schola", "TeX Gyre Schola (tipo Century)", "texgyreschola", "qcs", true, "Century Schoolbook"),
	"bonum":    gyre("bonum", "TeX Gyre Bonum (tipo Bookman)", "texgyrebonum", "qbk", true, "Bookman Old Style"),
	"heros":    gyre("heros", "TeX Gyre Heros (tipo Helvetica)", "texgyreheros", "qhv", false, "Arial"),
	"adventor": gyre("adventor", "TeX Gyre Adventor (tipo Avant Garde)", "texgyreadventor", "qag", false, "Century Gothic"),
	"opensans": {
		ID: "opensans", Nome: "Open Sans", Base: "OpenSans", Ext: ".ttf",
		Normal: "*-Regular", Negra: "*-Semibold", Cursiva: "*-Italic", NegraCurs: "*-SemiboldItalic",
		Ruta: "fonts/", NFSS: "qhv", Substituta: "heros", Word: "Open Sans",
	},
	"firasans": {
		ID: "firasans", Nome: "Fira Sans", Base: "FiraSans", Ext: ".otf",
		Normal: "*-Regular", Negra: "*-Bold", Cursiva: "*-Italic", NegraCurs: "*-BoldItalic",
		Ruta: "fonts/", NFSS: "qhv", Substituta: "heros", Word: "Calibri",
	},
	"libertinus": {
		ID: "libertinus", Nome: "Libertinus Serif", Base: "LibertinusSerif", Ext: ".otf",
		Normal: "*-Regular", Negra: "*-Bold", Cursiva: "*-Italic", NegraCurs: "*-BoldItalic",
		Ruta: "fonts/", NFSS: "qtm", Substituta: "termes", Serifa: true, Word: "Cambria",
	},
	"sourceserif": {
		ID: "sourceserif", Nome: "Source Serif", Base: "SourceSerif4", Ext: ".otf",
		Normal: "*-Regular", Negra: "*-Bold", Cursiva: "*-It", NegraCurs: "*-BoldIt",
		NFSS: "qtm", Substituta: "termes", Serifa: true, Word: "Georgia",
	},
}

func gyre(id, nome, base, nfss string, serifa bool, word string) Familia {
	return Familia{ID: id, Nome: nome, Base: base, Ext: ".otf",
		Normal: "*-regular", Negra: "*-bold", Cursiva: "*-italic", NegraCurs: "*-bolditalic",
		NFSS: nfss, Serifa: serifa, Word: word}
}

var familiasMates = map[string]FamiliaMates{
	"termes":     {"termes", "TeX Gyre Termes Math", "texgyretermes-math.otf", ""},
	"pagella":    {"pagella", "TeX Gyre Pagella Math", "texgyrepagella-math.otf", ""},
	"schola":     {"schola", "TeX Gyre Schola Math", "texgyreschola-math.otf", ""},
	"bonum":      {"bonum", "TeX Gyre Bonum Math", "texgyrebonum-math.otf", ""},
	"fira":       {"fira", "Fira Math", "FiraMath-Regular.otf", "fonts/"},
	"libertinus": {"libertinus", "Libertinus Math", "LibertinusMath-Regular.otf", "fonts/"},
}

// combinacions, na orde na que se amosan. A primeira é a de Lúa: a que saía
// ata agora (Helvetica + Open Sans), pero cos títulos na mesma letra ca o corpo.
var combinacions = []Combinacion{
	{ID: "lua", Nome: "Lúa", Corpo: "heros", Titulos: "heros", Etiquetas: "opensans", Descricion: "Clara e actual (a de sempre)"},
	{ID: "clasica", Nome: "Clásica", Corpo: "termes", Titulos: "termes", Etiquetas: "heros", Mates: "termes", Descricion: "Formal, tipo Times"},
	{ID: "elegante", Nome: "Elegante", Corpo: "pagella", Titulos: "pagella", Etiquetas: "pagella", Mates: "pagella", Descricion: "Coidada, tipo Palatino"},
	{ID: "lexible", Nome: "Lexible", Corpo: "schola", Titulos: "heros", Etiquetas: "heros", Mates: "schola", Descricion: "Moi fácil de ler, tipo libro escolar"},
	{ID: "moderna", Nome: "Moderna", Corpo: "firasans", Titulos: "firasans", Etiquetas: "firasans", Mates: "fira", Descricion: "Sen serifa, limpa"},
	{ID: "academica", Nome: "Académica", Corpo: "libertinus", Titulos: "libertinus", Etiquetas: "heros", Mates: "libertinus", Descricion: "De libro, boa para fórmulas"},
}

// Combinacions devolve as combinacións na orde da interface.
func Combinacions() []Combinacion {
	return append([]Combinacion(nil), combinacions...)
}

// CombinacionPorID devolve a combinación (a de Lúa se o id non existe).
func CombinacionPorID(id string) (Combinacion, bool) {
	for _, c := range combinacions {
		if c.ID == id {
			return c, true
		}
	}
	return combinacions[0], false
}

// FamiliaPorID devolve unha familia do catálogo.
func FamiliaPorID(id string) (Familia, bool) {
	f, ok := familias[id]
	return f, ok
}

// FamiliaMatesPorID devolve unha letra matemática do catálogo.
func FamiliaMatesPorID(id string) (FamiliaMates, bool) {
	f, ok := familiasMates[id]
	return f, ok
}

// IDsFamilias devolve os ids das familias (ordenados), para o esquema e a UI.
func IDsFamilias() []string {
	out := make([]string, 0, len(familias))
	for k := range familias {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// IDsFamiliasMates devolve os ids das letras matemáticas (ordenados).
func IDsFamiliasMates() []string {
	out := make([]string, 0, len(familiasMates))
	for k := range familiasMates {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Letras é o xogo de letras xa resolto para un documento: unha familia por
// papel. Unha cadea que non é do catálogo é unha fonte do sistema (só
// xelatex/lualatex).
type Letras struct {
	Corpo, Titulos, Etiquetas, Mates string
}

// LetrasDe resolve unha combinación e as escollas por papel ("" = a da
// combinación).
func LetrasDe(combinacion, corpo, titulos, etiquetas, mates string) Letras {
	c, _ := CombinacionPorID(combinacion)
	l := Letras{Corpo: c.Corpo, Titulos: c.Titulos, Etiquetas: c.Etiquetas, Mates: c.Mates}
	if s := strings.TrimSpace(corpo); s != "" {
		l.Corpo = s
		// Unha fonte do sistema escollida para o corpo leva tamén os títulos e
		// as etiquetas se non se dixo outra cousa: é o que espera quen escolle
		// «Georgia» (que todo saia en Georgia).
		if _, ok := familias[s]; !ok {
			if strings.TrimSpace(titulos) == "" {
				l.Titulos = s
			}
			if strings.TrimSpace(etiquetas) == "" {
				l.Etiquetas = s
			}
		}
	}
	if s := strings.TrimSpace(titulos); s != "" {
		l.Titulos = s
	}
	if s := strings.TrimSpace(etiquetas); s != "" {
		l.Etiquetas = s
	}
	if s := strings.TrimSpace(mates); s != "" {
		if s == "lm" {
			s = ""
		}
		l.Mates = s
	}
	return l
}

// Dispoñible di se un ficheiro de fonte se pode atopar (kpsewhich, unha vez
// por ficheiro). É unha variable para que os tests non dependan do TeX Live do
// equipo.
var Dispoñible = func(ficheiro string) bool {
	if v, ok := dispoñibles.Load(ficheiro); ok {
		return v.(bool)
	}
	v := KpsewhichFound(ficheiro)
	dispoñibles.Store(ficheiro, v)
	return v
}

var dispoñibles sync.Map

// ficheiro devolve o nome do ficheiro normal da familia (para kpsewhich).
func (f Familia) ficheiro() string {
	return strings.ReplaceAll(f.Normal, "*", f.Base) + f.Ext
}

// resolver devolve a familia que se pode usar de verdade: a pedida ou, se
// falta o ficheiro, a substituta. As de Lúa (Ruta fonts/) sempre están: Lúa
// escríbeas ao carón do .tex.
func resolver(id string, avisos *[]string) (Familia, bool) {
	f, ok := familias[id]
	if !ok {
		return Familia{}, false
	}
	if f.Ruta != "" || Dispoñible(f.ficheiro()) {
		return f, true
	}
	if s, ok := familias[f.Substituta]; ok && f.Substituta != "" {
		*avisos = append(*avisos, idioma.T("go.deseno.letra_substituta", "a letra «%s» non está instalada neste equipo: usouse «%s»", f.Nome, s.Nome))
		return s, true
	}
	return Familia{}, false
}

// opcionsFontspec é a lista [..] de fontspec para cargar a familia polos
// seus ficheiros.
func (f Familia) opcionsFontspec() string {
	op := fmt.Sprintf("Extension=%s,UprightFont=%s,BoldFont=%s,ItalicFont=%s,BoldItalicFont=%s", f.Ext, f.Normal, f.Negra, f.Cursiva, f.NegraCurs)
	if f.Ruta != "" {
		op = "Path=" + f.Ruta + "," + op
	}
	return op
}

// nomeSeguro limpa o nome dunha fonte do sistema antes de metelo en LaTeX.
func nomeSeguro(nome string) string {
	return strings.TrimSpace(fontNameSafeRe.ReplaceAllString(nome, ""))
}

// LatexLetras son as liñas de LaTeX que poñen as letras.
//   - Preambulo vai despois de fontspec (ou de fontenc en pdflatex):
//     \setmainfont, \setsansfont e \fontchip.
//   - Mates carga unicode-math ao final do preámbulo (ten que ir despois de
//     amsmath); pódese poñer en calquera sitio do preámbulo.
type LatexLetras struct {
	Preambulo string
	Mates     string
	Avisos    []string
	// Sistema: hai algunha fonte do sistema (o PDF non será igual noutro equipo).
	Sistema bool
	// Ficheiros: as fontes que leva a aplicación e que hai que deixar en
	// fonts/ ao carón do .tex.
	Ficheiros []string
}

// ficheiros devolve os catro ficheiros da familia.
func (f Familia) ficheiros() []string {
	var out []string
	for _, p := range []string{f.Normal, f.Negra, f.Cursiva, f.NegraCurs} {
		out = append(out, strings.ReplaceAll(p, "*", f.Base)+f.Ext)
	}
	return out
}

// XerarLetras compón as liñas de LaTeX das letras para o motor dado.
func XerarLetras(l Letras, engine string) LatexLetras {
	var out LatexLetras
	unicode := EngineUnicodeNativo(engine)
	var b strings.Builder
	type papel struct {
		id   string
		cmd  string // \setmainfont / \setsansfont / fontchip
		nfss string // \rmdefault / \sfdefault / ""
	}
	papeis := []papel{{l.Corpo, `\setmainfont`, `\rmdefault`}, {l.Titulos, `\setsansfont`, `\sfdefault`}, {l.Etiquetas, `fontchip`, ""}}
	chipDefinido := false
	for _, p := range papeis {
		id := strings.TrimSpace(p.id)
		if id == "" {
			continue
		}
		f, ok := resolver(id, &out.Avisos)
		if !ok {
			// Fonte do sistema: só con fontspec.
			nome := nomeSeguro(id)
			if !unicode || nome == "" {
				if nome != "" {
					out.Avisos = append(out.Avisos, idioma.T("go.deseno.letra_so_xelatex", "a letra «%s» só se pode usar con xelatex ou lualatex", nome))
				}
				continue
			}
			out.Sistema = true
			if p.cmd == "fontchip" {
				fmt.Fprintf(&b, "\\newfontfamily\\fontchip{%s}\n", nome)
				chipDefinido = true
			} else {
				fmt.Fprintf(&b, "%s{%s}\n", p.cmd, nome)
			}
			continue
		}
		if unicode {
			def := func(f Familia) string {
				if p.cmd == "fontchip" {
					return fmt.Sprintf("\\newfontfamily\\fontchip{%s}[%s]", f.Base, f.opcionsFontspec())
				}
				return fmt.Sprintf("%s{%s}[%s]", p.cmd, f.Base, f.opcionsFontspec())
			}
			chipDefinido = chipDefinido || p.cmd == "fontchip"
			if f.Ruta != "" {
				out.Ficheiros = append(out.Ficheiros, f.ficheiros()...)
				// As fontes que leva Lúa só están se a aplicación as deixou ao
				// carón do .tex (os informes si; un exame non): se non, a substituta.
				s, ok := familias[f.Substituta]
				if !ok {
					s = familias["heros"]
				}
				fmt.Fprintf(&b, "\\IfFileExists{%s%s}{%s}{%s}\n", f.Ruta, f.ficheiro(), def(f), def(s))
				continue
			}
			b.WriteString(def(f) + "\n")
			continue
		}
		if p.cmd == "fontchip" {
			fmt.Fprintf(&b, "\\newcommand{\\fontchip}{\\fontfamily{%s}\\selectfont}\n", f.NFSS)
			chipDefinido = true
		} else {
			fmt.Fprintf(&b, "\\renewcommand{%s}{%s}\n", p.nfss, f.NFSS)
		}
	}
	if !chipDefinido {
		b.WriteString("\\newcommand{\\fontchip}{\\sffamily}\n")
	}
	// Se o corpo é sen serifa, o documento enteiro vai nela tamén en pdflatex
	// (\familydefault): en xelatex xa o fai \setmainfont.
	out.Preambulo = b.String()

	if m := strings.TrimSpace(l.Mates); m != "" && unicode {
		if fm, ok := familiasMates[m]; ok {
			if fm.Ruta != "" && Dispoñible("unicode-math.sty") {
				// As que leva Lúa: só se están en fonts/ (os informes si; un exame non).
				out.Ficheiros = append(out.Ficheiros, fm.Ficheiro)
				out.Mates = "\\IfFileExists{" + fm.Ruta + fm.Ficheiro + "}{\\AddToHook{begindocument/before}{\\usepackage{unicode-math}\\setmathfont{" + fm.Ficheiro + "}[Path=" + fm.Ruta + "]}}{}\n"
			} else if Dispoñible(fm.Ficheiro) && Dispoñible("unicode-math.sty") {
				// Ao final do preámbulo: unicode-math ten que cargar despois de
				// amsmath, e o preámbulo de cada documento pon amsmath nun sitio
				// distinto.
				out.Mates = "\\AddToHook{begindocument/before}{\\usepackage{unicode-math}\\setmathfont{" + fm.Ficheiro + "}}\n"
			} else {
				out.Avisos = append(out.Avisos, idioma.T("go.deseno.mates_falta", "a letra das fórmulas «%s» non está instalada: as fórmulas saen na letra de sempre", fm.Nome))
			}
		}
	}
	out.Ficheiros = senRepetir(out.Ficheiros)
	return out
}

func senRepetir(l []string) []string {
	vistos := map[string]bool{}
	var out []string
	for _, x := range l {
		if !vistos[x] {
			vistos[x] = true
			out = append(out, x)
		}
	}
	return out
}

// DeclaraLetras di se un LaTeX (o preámbulo dunha plantilla completa) xa fixa
// as súas letras: daquela mandan as da plantilla.
func DeclaraLetras(preambulo string) bool {
	for _, m := range []string{`\setmainfont`, `\setsansfont`, `\familydefault`, `\rmdefault`, `\sfdefault`,
		`{mathptmx}`, `{helvet}`, `{mathpazo}`, `{tgtermes}`, `{tgpagella}`, `{tgheros}`, `{tgschola}`,
		`{lmodern}`, `{times}`, `{palatino}`, `{libertine}`, `{libertinus}`, `{opensans}`, `{FiraSans}`, `{sourceserifpro}`, `unicode-math`} {
		if strings.Contains(preambulo, m) {
			return true
		}
	}
	return false
}

// CombinacionDePreset traduce os valores vellos do panel «Axustes do texto»
// (serif, sans, palatino) á combinación equivalente.
func CombinacionDePreset(fonte string) (combinacion, sistema string) {
	switch strings.TrimSpace(fonte) {
	case "":
		return "", ""
	case "serif":
		return "clasica", ""
	case "sans":
		return "lua", ""
	case "palatino":
		return "elegante", ""
	}
	f := strings.TrimSpace(fonte)
	if _, ok := CombinacionPorID(f); ok {
		return f, ""
	}
	return "", f
}
