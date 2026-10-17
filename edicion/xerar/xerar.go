// Package xerar converte fichas en documentos: escolle os valores de cada
// variante, compón o texto (Markdown ou matex) e devolve o corpo LaTeX que
// edicion/documento compila coa plantilla.
package xerar

import (
	"edicion/idioma"
	_ "embed"
	"fmt"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"edicion/cas"
	"edicion/documento"
	"edicion/ficha"
	"edicion/latex"
)

//go:embed codeini.ini
var codeIniPorDefecto string

// Modos de xeración (as lapelas de Yang).
const (
	ModoEnunciados = "enunciados" // folla para o alumnado
	ModoResultado  = "resultado"  // + resultado final
	ModoResolucion = "resolucion" // + resolución paso a paso
)

var modoCas = map[string]string{ModoEnunciados: "enunciados", ModoResultado: "resposta", ModoResolucion: "solucions", "": "enunciados"}

// Opcions dunha xeración.
type Opcions struct {
	Variantes     int
	Semente       int
	Modo          string
	Maxima        string // ruta do executable; baleiro = «maxima» do PATH
	CodeIni       string // baleiro = o de Yang
	Decimais      int
	ForzarDecimal bool
	BaseDir       string         // onde buscar as imaxes (a bóveda)
	Lingua        string         // nos exames bilingües; baleira = a primeira da ficha
	Datos         map[string]any // valores dos campos escritos (formularios)
}

// Fichas resolve unha ficha polo seu id (as ![[ligazóns]] dun exame e os
// bancos).
type Fichas func(id string) (*ficha.Ficha, error)

// Resultado dunha xeración.
type Resultado struct {
	Corpo  string // corpo LaTeX (sen preámbulo)
	Avisos []string
	// Valores: os valores dos campos da ficha principal, por variante.
	Valores []map[string]Valor
}

type xerador struct {
	o                Opcions
	fichas           Fichas
	dir              string
	m                *cas.Maxima
	reg              *latex.RawRegistry
	conv             *documento.Conversor
	rng              *rand.Rand
	avisos           []string
	valoresPrincipal map[string]Valor
	nImaxes          int
}

// RutaMaxima busca Maxima (a configurada ou a do PATH).
func RutaMaxima(configurada string) string {
	if strings.TrimSpace(configurada) != "" {
		return configurada
	}
	if p, err := exec.LookPath("maxima"); err == nil {
		return p
	}
	return ""
}

// Corpo xera o corpo LaTeX de `f` en `dir` (onde tamén van as imaxes dos
// gráficos). O chamador compílao con documento.Compilar no mesmo dir.
func Corpo(f *ficha.Ficha, fichas Fichas, o Opcions, dir string) (Resultado, error) {
	if o.Variantes < 1 {
		o.Variantes = 1
	}
	if fichas == nil {
		fichas = func(id string) (*ficha.Ficha, error) { return nil, fmt.Errorf("non hai bóveda aberta") }
	}
	x := &xerador{o: o, fichas: fichas, dir: dir, reg: latex.NovoRawRegistry(), conv: documento.NovoConversor(o.BaseDir, dir)}
	if err := os.MkdirAll(filepath.Join(dir, "images"), 0o755); err != nil {
		return Resultado{}, err
	}
	precisa, err := x.arbrePrecisaMaxima(f, 0)
	if err != nil {
		return Resultado{}, err
	}
	if precisa {
		ruta := RutaMaxima(o.Maxima)
		if ruta == "" {
			return Resultado{}, fmt.Errorf("esta ficha precisa Maxima e non está instalado (ou configura a súa ruta en Opcións)")
		}
		sess, err := cas.Open(ruta)
		if err != nil {
			return Resultado{}, idioma.Errorf("go.maxima.non_inicia", "non se puido iniciar Maxima: %w", err)
		}
		x.m = cas.NewMaxima(sess)
		defer x.m.Close()
		x.m.SetOutDir(dir)
		x.m.Format = cas.FormatLatex
		x.m.ForzarDecimal = o.ForzarDecimal
		x.m.RenderMode = modoCas[o.Modo]
		if _, err := x.m.Send("linel:1024;display2d:false;simp:true;", false); err != nil {
			return Resultado{}, err
		}
	}
	var corpo strings.Builder
	var res Resultado
	for i := 0; i < o.Variantes; i++ {
		semente := o.Semente + i
		if err := x.iniciarVariante(semente); err != nil {
			return res, err
		}
		x.valoresPrincipal = nil
		l, err := x.ficha(f, 0)
		if err != nil {
			return res, fmt.Errorf("variante %d (semente %d): %w", i+1, semente, err)
		}
		if i > 0 {
			corpo.WriteString("\n\\clearpage\n")
		}
		corpo.WriteString(l)
		res.Valores = append(res.Valores, x.valoresPrincipal)
	}
	final := corpo.String()
	for i := 0; i < 5 && strings.Contains(final, "MATEXERAWBLOCK"); i++ {
		final = x.reg.Resolve(final)
	}
	final, av := documento.SanearCorpo(final)
	res.Corpo = final
	res.Avisos = append(append(x.avisos, x.conv.Avisos()...), av...)
	return res, nil
}

func (x *xerador) iniciarVariante(semente int) error {
	x.rng = rand.New(rand.NewPCG(uint64(semente), 0x4c7561596e67))
	if x.m == nil {
		return nil
	}
	if _, err := x.m.Send("reset();kill(all)", true); err != nil {
		return err
	}
	if err := x.m.SetDecimais(x.o.Decimais); err != nil {
		return err
	}
	ini := x.o.CodeIni
	if strings.TrimSpace(ini) == "" {
		ini = codeIniPorDefecto
	}
	for _, l := range strings.Split(strings.ReplaceAll(ini, "\r", ""), "\n") {
		l = strings.TrimSpace(strings.ReplaceAll(l, "\ufeff", ""))
		if l == "" {
			continue
		}
		if _, err := x.m.Send(l, false); err != nil {
			x.avisos = append(x.avisos, fmt.Sprintf("codeini %q: %v", l, err))
		}
	}
	return x.m.SetSeed(semente)
}

var ligazonRe = regexp.MustCompile(`!\[\[([^\]|\n]+)(\|[^\]\n]*)?\]\]`)

func (x *xerador) arbrePrecisaMaxima(f *ficha.Ficha, fondo int) (bool, error) {
	if fondo > 4 {
		return false, fmt.Errorf("demasiados exames dentro de exames (ou unha ligazón circular)")
	}
	if PrecisaMaxima(f) {
		return true, nil
	}
	if f.Tipo != ficha.TipoExame {
		return false, nil
	}
	for _, m := range ligazonRe.FindAllStringSubmatch(f.Corpo, -1) {
		e, err := x.fichas(strings.TrimSpace(m[1]))
		if err != nil {
			continue // xa avisará ao compoñer
		}
		if p, err := x.arbrePrecisaMaxima(e, fondo+1); p || err != nil {
			return p, err
		}
	}
	return false, nil
}

func (x *xerador) ficha(f *ficha.Ficha, fondo int) (string, error) {
	if fondo > 4 {
		return "", fmt.Errorf("demasiados exames dentro de exames (ou unha ligazón circular)")
	}
	if f.Tipo == ficha.TipoExame {
		return x.exame(f, fondo)
	}
	return x.exercicio(f, fondo)
}

func (x *xerador) bancos(nome string) (*ficha.Ficha, error) { return x.fichas(nome) }

func (x *xerador) exercicio(f *ficha.Ficha, fondo int) (string, error) {
	if f.FormatoCorpo() == ficha.FormatoMatex {
		fixos, err := valoresGo(f, x.rng, x.bancos, x.o.Datos, false)
		if err != nil {
			return "", err
		}
		if fondo == 0 {
			x.valoresPrincipal = fixos
		}
		return x.casALatex(textoMatex(f, fixos))
	}
	vals, err := x.valoresMarkdown(f)
	if err != nil {
		return "", err
	}
	if fondo == 0 {
		x.valoresPrincipal = vals
	}
	md := x.seleccionarSeccions(f, vals)
	return x.markdownALatex(md, vals)
}

// valoresMarkdown: en Go se se pode; se non, con Maxima.
func (x *xerador) valoresMarkdown(f *ficha.Ficha) (map[string]Valor, error) {
	if !PrecisaMaxima(f) || x.m == nil {
		return x.conDecimais(valoresGo(f, x.rng, x.bancos, x.o.Datos, true))
	}
	vals, err := valoresGo(f, x.rng, x.bancos, x.o.Datos, false)
	if err != nil {
		return nil, err
	}
	orde, err := ficha.OrdeCalculo(f)
	if err != nil {
		return nil, err
	}
	porNome := map[string]ficha.Campo{}
	for _, c := range f.Campos {
		porNome[ficha.NomeBase(c.Nome)] = c
	}
	for _, n := range orde {
		c := porNome[n]
		if c.Orixe == ficha.OrixeEscrito || c.Orixe == ficha.OrixeTaboa {
			continue
		}
		l := liñaMaxima(c, nil)
		if l == "" {
			continue
		}
		if _, err := x.m.Send(strings.TrimSuffix(l, "$"), false); err != nil {
			return nil, fmt.Errorf("campo «%s»: %w", c.Nome, err)
		}
		if c.Funcion || strings.Contains(c.Nome, "(") {
			continue // unha función non ten un valor que amosar
		}
		if x.senValor(n) {
			return nil, fmt.Errorf("campo «%s»: non se puido calcular %s (división por cero ou erro de Maxima?)", c.Nome, c.Expr)
		}
		tex, err := x.tex(n, true)
		if err != nil {
			return nil, fmt.Errorf("campo «%s»: %w", c.Nome, err)
		}
		v := Valor{Tex: tex, Texto: tex, Mat: true}
		if n, err := x.numero(n); err == nil {
			v.Num, v.EsNum = n, true
		}
		vals[n] = v
	}
	return vals, nil
}

// conDecimais redondea á opción Decimais os números calculados en Go (os de
// Maxima xa os redondea ToTeX, ver cas.Maxima.SetDecimais).
func (x *xerador) conDecimais(vals map[string]Valor, err error) (map[string]Valor, error) {
	if err != nil || x.o.Decimais <= 0 {
		return vals, err
	}
	for k, v := range vals {
		if v.EsNum && !v.Mat {
			v.Texto, v.Tex = formatarNumeroDecimais(v.Num, x.o.Decimais)
			vals[k] = v
		}
	}
	return vals, nil
}

// tex: o TeX dunha expresión (sen os $), ou erro se Maxima non o puido facer.
func (x *xerador) tex(expr string, avaliar bool) (string, error) {
	gardar := x.m.ProtectRaw
	x.m.ProtectRaw = nil
	defer func() { x.m.ProtectRaw = gardar }()
	s, err := x.m.ToTeX(expr, avaliar)
	if err != nil {
		return "", err
	}
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "$") || !strings.HasSuffix(s, "$") {
		return "", fmt.Errorf("Maxima non o puido calcular: %s", strings.TrimSpace(latexATexto(s)))
	}
	return numerosTex(strings.Trim(s, "$")), nil
}

var numeroMaximaRe = regexp.MustCompile(`^-?[0-9]+(\.[0-9]+)?([eEbB][-+]?[0-9]+)?$`)

// numero: o valor numérico (se o ten) dunha variable de Maxima.
func (x *xerador) numero(nome string) (float64, error) {
	out, err := x.m.Send("float("+nome+")", true)
	if err != nil {
		return 0, err
	}
	s := ultimoValor(out)
	if !numeroMaximaRe.MatchString(s) {
		return 0, fmt.Errorf("non é un número: %s", s)
	}
	var n float64
	_, err = fmt.Sscan(strings.NewReplacer("b", "e", "B", "e").Replace(s), &n)
	return n, err
}

var etiquetaSaidaRe = regexp.MustCompile(`\(%o\d+\)\s*`)

func ultimoValor(out string) string {
	partes := etiquetaSaidaRe.Split(out, -1)
	return strings.TrimSpace(partes[len(partes)-1])
}

var comandoLatexRe = regexp.MustCompile(`\\[a-zA-Z]+(\{[^}]*\})?`)

func latexATexto(s string) string {
	return strings.Trim(comandoLatexRe.ReplaceAllString(s, " "), "{} ")
}

// seleccionarSeccions: o Markdown que vai neste modo.
func (x *xerador) seleccionarSeccions(f *ficha.Ficha, vals map[string]Valor) string {
	var b strings.Builder
	var resultado, resolucion string
	lingua := x.o.Lingua
	principal := ""
	if len(f.Linguas) > 0 {
		principal = f.Linguas[0]
	}
	for _, s := range f.Seccions() {
		if s.Lingua != "" && s.Lingua != lingua {
			continue
		}
		if s.Lingua == "" && lingua != "" && lingua != principal && temVersion(f, s.Rol, lingua) {
			continue
		}
		switch s.Rol {
		case ficha.RolResultado:
			resultado = s.Texto
		case ficha.RolResolucion:
			resolucion = s.Texto
		case ficha.RolContido:
			b.WriteString("## " + s.Titulo + "\n\n" + s.Texto + "\n\n")
		default:
			b.WriteString(s.Texto + "\n\n")
		}
	}
	if strings.TrimSpace(resultado) == "" {
		resultado = resultadoAutomatico(f)
	}
	switch x.o.Modo {
	case ModoResultado:
		if strings.TrimSpace(resultado) != "" {
			b.WriteString("**Resultado.** " + resultado + "\n\n")
		}
	case ModoResolucion:
		if strings.TrimSpace(resolucion) == "" {
			resolucion = resultado
		}
		if strings.TrimSpace(resolucion) != "" {
			b.WriteString("**Resolución.** " + resolucion + "\n\n")
		}
	}
	return b.String()
}

// temVersion: a ficha ten unha sección deste rol na lingua dada.
func temVersion(f *ficha.Ficha, rol, lingua string) bool {
	for _, s := range f.Seccions() {
		if s.Rol == rol && s.Lingua == lingua {
			return true
		}
	}
	return false
}

// resultadoAutomatico: «etiqueta: {nome}» por cada campo resultado.
func resultadoAutomatico(f *ficha.Ficha) string {
	var partes []string
	for _, c := range f.Campos {
		if c.Rol != ficha.RolResultado {
			continue
		}
		et := strings.TrimSpace(c.Etiqueta)
		if et == "" {
			et = c.Nome
		}
		partes = append(partes, et+": {"+ficha.NomeBase(c.Nome)+"}")
	}
	return strings.Join(partes, "; ")
}

func (x *xerador) markdownALatex(md string, vals map[string]Valor) (string, error) {
	cx := &contextoMD{vals: vals, reg: x.reg, avisos: &x.avisos, imaxe: x.imaxe, decimais: x.o.Decimais}
	if x.m != nil {
		cx.maxima = x.tex
		cx.plot = func(codigo string) (string, error) {
			gardar := x.m.ProtectRaw
			x.m.ProtectRaw = nil
			defer func() { x.m.ProtectRaw = gardar }()
			return x.m.ParseText("<PLOT>" + codigo + "</PLOT>")
		}
	}
	l, err := mdALatex(cx.substituir(md))
	if err != nil {
		return "", err
	}
	return arranxarPandoc(l), nil
}

var (
	seccionPandocRe = regexp.MustCompile(`\\(section|subsection|subsubsection|paragraph)\{`)
	labelPandocRe   = regexp.MustCompile(`\\label\{[^}]*\}`)
	hipertargetRe   = regexp.MustCompile(`(?s)\\hypertarget\{[^}]*\}\{%\n`)
)

// arranxarPandoc: títulos sen numerar e sen etiquetas (nun exame os números
// pono quen o escribe).
func arranxarPandoc(l string) string {
	l = seccionPandocRe.ReplaceAllString(l, `\$1*{`)
	l = labelPandocRe.ReplaceAllString(l, "")
	if hipertargetRe.MatchString(l) {
		// \hypertarget{x}{% ... } envolve cada título: quítase o envoltorio
		// e a chave que o pecha.
		l = hipertargetRe.ReplaceAllString(l, "{")
	}
	return l
}

// imaxe copia unha imaxe da bóveda ao cartafol da compilación e devolve o
// LaTeX que a inclúe ("" se non se atopa).
func (x *xerador) imaxe(ruta, ancho string) string {
	ruta = strings.TrimSpace(strings.SplitN(ruta, " ", 2)[0])
	if ruta == "" || strings.Contains(ruta, "://") || x.o.BaseDir == "" {
		return ""
	}
	var candidatas []string
	if filepath.IsAbs(ruta) {
		candidatas = []string{ruta}
	} else {
		candidatas = []string{filepath.Join(x.o.BaseDir, ruta), filepath.Join(x.o.BaseDir, "raw", "imaxes", filepath.Base(ruta))}
	}
	for _, c := range candidatas {
		datos, err := os.ReadFile(c)
		if err != nil {
			continue
		}
		x.nImaxes++
		nome := fmt.Sprintf("img-%d%s", x.nImaxes, strings.ToLower(filepath.Ext(c)))
		if err := os.WriteFile(filepath.Join(x.dir, "images", nome), datos, 0o644); err != nil {
			return ""
		}
		if ancho == "" {
			ancho = "0.6\\linewidth"
		}
		return "\\begin{center}\\includegraphics[width=" + ancho + ",keepaspectratio]{images/" + nome + "}\\end{center}"
	}
	x.avisos = append(x.avisos, "non se atopou a imaxe "+ruta)
	return ""
}

// casALatex: texto matex -> Maxima -> HTML -> LaTeX.
func (x *xerador) casALatex(texto string) (string, error) {
	if x.m == nil {
		return "", fmt.Errorf("esta ficha precisa Maxima")
	}
	x.m.ProtectRaw = x.reg.Protect
	html, err := x.m.ParseText(documento.PreprocesarPseudoEtiquetas(texto, x.reg))
	if err != nil {
		return "", err
	}
	return x.conv.HTMLaLatex(html)
}

var calloutRe = regexp.MustCompile(`^>\s*\[!([A-Za-z-]+)\][-+]?\s*(.*)$`)

func (x *xerador) exame(f *ficha.Ficha, fondo int) (string, error) {
	if f.FormatoCorpo() == ficha.FormatoMatex {
		var erros []string
		texto := ligazonRe.ReplaceAllStringFunc(f.Corpo, func(l string) string {
			id := strings.TrimSpace(ligazonRe.FindStringSubmatch(l)[1])
			e, err := x.fichas(id)
			if err != nil {
				erros = append(erros, id)
				return ""
			}
			if e.FormatoCorpo() == ficha.FormatoMatex && e.Tipo != ficha.TipoExame {
				fixos, err := valoresGo(e, x.rng, x.bancos, x.o.Datos, false)
				if err != nil {
					erros = append(erros, id+": "+err.Error())
					return ""
				}
				return "<EX>" + textoMatex(e, fixos) + "</EX>"
			}
			lx, err := x.ficha(e, fondo+1)
			if err != nil {
				erros = append(erros, id+": "+err.Error())
				return ""
			}
			return x.reg.Protect(lx)
		})
		if len(erros) > 0 {
			return "", fmt.Errorf("non se puideron incluír: %s", strings.Join(erros, "; "))
		}
		return x.casALatex(texto)
	}
	vals, err := x.valoresMarkdown(f)
	if err != nil {
		return "", err
	}
	if fondo == 0 {
		x.valoresPrincipal = vals
	}
	var md strings.Builder
	for _, liña := range strings.Split(f.Corpo, "\n") {
		t := strings.TrimSpace(liña)
		if m := calloutRe.FindStringSubmatch(t); m != nil {
			if m[2] != "" {
				md.WriteString("*" + m[2] + "*\n\n")
			}
			continue
		}
		if strings.HasPrefix(t, ">") {
			t = strings.TrimSpace(strings.TrimPrefix(t, ">"))
		}
		if l := ligazonRe.FindStringSubmatch(t); l != nil && l[0] == t {
			e, err := x.fichas(strings.TrimSpace(l[1]))
			if err != nil {
				return "", fmt.Errorf("non se atopa a ficha %s: %w", l[1], err)
			}
			lx, err := x.ficha(e, fondo+1)
			if err != nil {
				return "", fmt.Errorf("%s: %w", l[1], err)
			}
			md.WriteString("\n" + x.reg.Protect(lx) + "\n\n")
			continue
		}
		md.WriteString(liña + "\n")
	}
	return x.markdownALatex(md.String(), vals)
}
