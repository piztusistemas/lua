package main

// Subconxunto de Markdown das descricións das tarefas dun cronograma. É unha
// porta de cronograma/internal/export/markdown.go (a app Piztu Orienta) para
// que os ficheiros gardados coa versión de escritorio orixinal se vexan igual
// aquí: títulos (#, ##, ###), listas (-, *, 1.), citas (>), **negrita**,
// *cursiva*, __negriña__, _cursiva_, ligazóns [texto](url), imaxes
// ![alt](img:ID =60%) e bloques de aliñamento cercados con ":::centro" ...
// ":::". O tamaño de letra vai como [fs=peq|gr|xgr]texto[/fs].
//
// O parser é o mesmo para as tres saídas (LaTeX, Markdown/Pandoc); o que
// cambia é só o renderizador. Ver informe_cronograma.go.

import (
	"strconv"
	"strings"
	"unicode"
)

type cronoTipoBloque int

const (
	cronoParrafo cronoTipoBloque = iota
	cronoTitulo
	cronoItem
	cronoCita
	cronoImaxe
	cronoDebuxo // ```tikz ou ```circuitikz (os circuítos dos problemas)
)

type cronoAlinamento int

const (
	// cronoAlinDefecto deixa que cada saída aplique o seu criterio: os
	// parágrafos van xustificados e as imaxes centradas.
	cronoAlinDefecto cronoAlinamento = iota
	cronoAlinEsquerda
	cronoAlinCentro
	cronoAlinDereita
	cronoAlinXustificado
)

// cronoTramo é un fragmento de texto cun estilo concreto.
type cronoTramo struct {
	Texto   string
	Negrita bool
	Cursiva bool
	URL     string  // se non está baleiro, o tramo é unha ligazón
	Escala  float64 // multiplicador do tamaño de letra (0 ou 1 = normal)
	// Math: unha fórmula LaTeX ($…$, ou $$…$$ se Display). Vai tal cal ao
	// LaTeX e ao Markdown; Word e OpenDocument póñena coma imaxe.
	Math    string
	Display bool
}

// cronoImaxeRef describe unha imaxe incrustada na descrición dunha tarefa. Ref
// identifica a imaxe: unha entrada do mapa Images da tarefa (ou un data URI
// completo). As dimensións son opcionais.
type cronoImaxeRef struct {
	Alt      string
	Ref      string
	AnchoPct float64 // ancho como porcentaxe da columna de texto (0 = sen indicar)
	AnchoMM  float64
	AltoMM   float64
}

type cronoBloque struct {
	Tipo     cronoTipoBloque
	Nivel    int  // títulos: 1..3
	Ordenada bool // listas: numerada ou con viñeta
	Alin     cronoAlinamento
	Tramos   []cronoTramo
	Imaxe    cronoImaxeRef // só para cronoImaxe
	// Debuxo: o tipo (tikz, circuitikz) e o código, só para cronoDebuxo.
	Debuxo, Codigo string
}

// cronoEscalasTamano son os tamaños de letra que se poden aplicar a un tramo.
var cronoEscalasTamano = map[string]float64{"peq": 0.8, "gr": 1.3, "xgr": 1.6}

// cronoTexto devolve o texto que se amosa nun tramo. Engade a URL entre
// parénteses se difire do texto visible, para que quede reflectida nos
// documentos impresos (unha ligazón non se pode premer no papel).
func (t cronoTramo) cronoTexto() string {
	if t.URL != "" && t.URL != t.Texto {
		if t.Texto == "" {
			return t.URL
		}
		return t.Texto + " (" + t.URL + ")"
	}
	return t.Texto
}

// parseMarkdownCrono converte Markdown nunha lista de bloques.
func parseMarkdownCrono(md string) []cronoBloque {
	var bloques []cronoBloque
	lineas := strings.Split(strings.ReplaceAll(md, "\r\n", "\n"), "\n")

	alin := cronoAlinDefecto
	var paraBuf []string
	pecharParrafo := func() {
		if len(paraBuf) == 0 {
			return
		}
		texto := strings.TrimSpace(strings.Join(paraBuf, " "))
		paraBuf = paraBuf[:0]
		if texto == "" {
			return
		}
		bloques = append(bloques, cronoBloque{Tipo: cronoParrafo, Alin: alin, Tramos: parseInlineCrono(texto)})
	}

	enMath := false // dentro dun bloque $$ … $$ de varias liñas
	var mathBuf []string
	debuxo := "" // dentro dun ```tikz / ```circuitikz
	var codigo []string
	for _, linea := range lineas {
		t := strings.TrimSpace(linea)
		if debuxo != "" {
			if strings.HasPrefix(t, "```") {
				bloques = append(bloques, cronoBloque{Tipo: cronoDebuxo, Alin: alin, Debuxo: debuxo, Codigo: strings.Join(codigo, "\n")})
				debuxo, codigo = "", nil
			} else {
				codigo = append(codigo, linea)
			}
			continue
		}
		if l := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(t, "```"))); strings.HasPrefix(t, "```") && (l == debuxoTikz || l == debuxoCircuitikz) {
			pecharParrafo()
			debuxo = l
			continue
		}
		if enMath {
			if strings.HasSuffix(t, "$$") {
				mathBuf = append(mathBuf, strings.TrimSuffix(t, "$$"))
				enMath = false
				bloques = append(bloques, cronoBloque{Tipo: cronoParrafo, Alin: alin, Tramos: []cronoTramo{{Math: strings.TrimSpace(strings.Join(mathBuf, "\n")), Display: true}}})
				mathBuf = nil
			} else {
				mathBuf = append(mathBuf, t)
			}
			continue
		}
		if strings.HasPrefix(t, "$$") && (t == "$$" || !strings.Contains(t[2:], "$$")) {
			pecharParrafo()
			enMath = true
			mathBuf = []string{strings.TrimPrefix(t, "$$")}
			continue
		}
		switch {
		case t == "":
			pecharParrafo()
		case strings.HasPrefix(t, ":::"):
			pecharParrafo()
			alin = parseValaCrono(t)
		case strings.HasPrefix(t, "### "):
			pecharParrafo()
			bloques = append(bloques, cronoBloque{Tipo: cronoTitulo, Nivel: 3, Alin: alin, Tramos: parseInlineCrono(t[4:])})
		case strings.HasPrefix(t, "## "):
			pecharParrafo()
			bloques = append(bloques, cronoBloque{Tipo: cronoTitulo, Nivel: 2, Alin: alin, Tramos: parseInlineCrono(t[3:])})
		case strings.HasPrefix(t, "# "):
			pecharParrafo()
			bloques = append(bloques, cronoBloque{Tipo: cronoTitulo, Nivel: 1, Alin: alin, Tramos: parseInlineCrono(t[2:])})
		case strings.HasPrefix(t, "> "):
			pecharParrafo()
			bloques = append(bloques, cronoBloque{Tipo: cronoCita, Alin: alin, Tramos: parseInlineCrono(t[2:])})
		case strings.HasPrefix(t, "- "), strings.HasPrefix(t, "* "):
			pecharParrafo()
			bloques = append(bloques, cronoBloque{Tipo: cronoItem, Alin: alin, Tramos: parseInlineCrono(t[2:])})
		case itemOrdenadoCrono(t):
			pecharParrafo()
			resto := t[strings.Index(t, ". ")+2:]
			bloques = append(bloques, cronoBloque{Tipo: cronoItem, Ordenada: true, Alin: alin, Tramos: parseInlineCrono(resto)})
		default:
			if img, ok := parseLinaImaxeCrono(t); ok {
				pecharParrafo()
				bloques = append(bloques, cronoBloque{Tipo: cronoImaxe, Alin: alin, Imaxe: img})
				continue
			}
			paraBuf = append(paraBuf, t)
		}
	}
	pecharParrafo()
	if enMath { // sen peche: o que había vai coma texto
		bloques = append(bloques, cronoBloque{Tipo: cronoParrafo, Alin: alin, Tramos: parseInlineCrono("$$ " + strings.Join(mathBuf, " "))})
	}
	return bloques
}

// parseValaCrono interpreta unha liña ":::nome". O peche (":::" soa) devolve
// o aliñamento por defecto, e un nome descoñecido tamén, para non perder o
// texto seguinte.
func parseValaCrono(linea string) cronoAlinamento {
	nome := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(linea, ":::")))
	switch nome {
	case "esquerda", "left":
		return cronoAlinEsquerda
	case "centro", "center", "centrado":
		return cronoAlinCentro
	case "dereita", "right":
		return cronoAlinDereita
	case "xustificado", "justify":
		return cronoAlinXustificado
	default:
		return cronoAlinDefecto
	}
}

func itemOrdenadoCrono(s string) bool {
	idx := strings.Index(s, ". ")
	if idx <= 0 {
		return false
	}
	for _, c := range s[:idx] {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// parseLinaImaxeCrono analiza unha liña que contén só unha imaxe:
// "![alt](ref)" con dimensións opcionais: "![alt](ref =60%)",
// "![alt](ref =120x80)", "![alt](ref =120x)" ou "![alt](ref =x80)" (en mm).
func parseLinaImaxeCrono(s string) (cronoImaxeRef, bool) {
	if !strings.HasPrefix(s, "![") {
		return cronoImaxeRef{}, false
	}
	runas := []rune(s)
	alt, destino, fin, ok := parseLigazonCrono(runas, 1) // salta o '!' e empeza no '['
	if !ok || fin != len(runas)-1 {
		return cronoImaxeRef{}, false // ten que consumir a liña enteira
	}
	img := cronoImaxeRef{Alt: alt, Ref: destino}
	if i := strings.LastIndex(destino, " ="); i >= 0 {
		img.Ref = strings.TrimSpace(destino[:i])
		parseTamanoImaxeCrono(&img, destino[i+2:])
	}
	img.Ref = strings.TrimPrefix(img.Ref, "img:")
	if img.Ref == "" {
		return cronoImaxeRef{}, false
	}
	return img, true
}

func parseTamanoImaxeCrono(img *cronoImaxeRef, tam string) {
	tam = strings.TrimSpace(tam)
	if pct, ok := strings.CutSuffix(tam, "%"); ok {
		if v, err := strconv.ParseFloat(pct, 64); err == nil && v > 0 {
			img.AnchoPct = v
		}
		return
	}
	w, h, haiX := strings.Cut(tam, "x")
	if !haiX {
		h = ""
	}
	if v, err := strconv.ParseFloat(w, 64); err == nil && v > 0 {
		img.AnchoMM = v
	}
	if v, err := strconv.ParseFloat(h, 64); err == nil && v > 0 {
		img.AltoMM = v
	}
}

// parseInlineCrono procesa as marcas **/__ (negrita), */_ (cursiva),
// [fs=..]..[/fs] (tamaño) e [texto](url) (ligazón).
func parseInlineCrono(s string) []cronoTramo {
	var tramos []cronoTramo
	var buf strings.Builder
	negrita, cursiva := false, false

	pechar := func() {
		if buf.Len() > 0 {
			tramos = append(tramos, cronoTramo{Texto: buf.String(), Negrita: negrita, Cursiva: cursiva})
			buf.Reset()
		}
	}

	runas := []rune(s)
	for i := 0; i < len(runas); i++ {
		c := runas[i]
		if c == '\\' && i+1 < len(runas) && runas[i+1] == '$' {
			buf.WriteRune('$')
			i++
			continue
		}
		// \( … \) e \[ … \] tamén son fórmulas (as de LaTeX).
		if c == '\\' && i+1 < len(runas) && (runas[i+1] == '(' || runas[i+1] == '[') {
			pecha := `\)`
			if runas[i+1] == '[' {
				pecha = `\]`
			}
			if j := strings.Index(string(runas[i+2:]), pecha); j > 0 {
				tex := string(runas[i+2:])[:j]
				pechar()
				tramos = append(tramos, cronoTramo{Math: strings.TrimSpace(tex), Display: runas[i+1] == '[', Negrita: negrita, Cursiva: cursiva})
				i += 2 + len([]rune(tex)) + 1
				continue
			}
		}
		if c == '$' {
			if tex, display, fin, ok := parseMathCrono(runas, i); ok {
				pechar()
				tramos = append(tramos, cronoTramo{Math: tex, Display: display, Negrita: negrita, Cursiva: cursiva})
				i = fin
				continue
			}
		}
		if c == '[' {
			if interior, escala, fin, ok := parseTamanoLetraCrono(runas, i); ok {
				pechar()
				for _, tr := range parseInlineCrono(interior) {
					tr.Negrita = tr.Negrita || negrita
					tr.Cursiva = tr.Cursiva || cursiva
					tr.Escala = escala
					tramos = append(tramos, tr)
				}
				i = fin
				continue
			}
			if texto, url, fin, ok := parseLigazonCrono(runas, i); ok {
				pechar()
				tramos = append(tramos, cronoTramo{Texto: texto, URL: url, Negrita: negrita, Cursiva: cursiva})
				i = fin
				continue
			}
		}
		if (c == '*' || c == '_') && i+1 < len(runas) && runas[i+1] == c {
			if negrita || enxeAbre(runas, i, true) {
				pechar()
				negrita = !negrita
				i++
				continue
			}
		} else if c == '*' || c == '_' {
			if cursiva || enxeAbre(runas, i, false) {
				pechar()
				cursiva = !cursiva
				continue
			}
		}
		buf.WriteRune(c)
	}
	pechar()
	if len(tramos) == 0 {
		tramos = []cronoTramo{{Texto: ""}}
	}
	return tramos
}

// parseMathCrono le unha fórmula que empeza en runas[inicio] (un «$»): $$…$$
// ou $…$ coas regras de Pandoc (o «$» que abre non vai seguido de espazo, o
// que pecha non vai despois dun espazo nin antes dun díxito), para que «custa
// 5$ ou 6$» siga sendo texto. Devolve o TeX, se é display e onde remata.
func parseMathCrono(runas []rune, inicio int) (tex string, display bool, fin int, ok bool) {
	if inicio+1 < len(runas) && runas[inicio+1] == '$' {
		for j := inicio + 2; j+1 < len(runas); j++ {
			if runas[j] == '$' && runas[j+1] == '$' && runas[j-1] != '\\' {
				t := strings.TrimSpace(string(runas[inicio+2 : j]))
				if t == "" {
					return "", false, 0, false
				}
				return t, true, j + 1, true
			}
		}
		return "", false, 0, false
	}
	if inicio+1 >= len(runas) {
		return "", false, 0, false
	}
	for j := inicio + 1; j < len(runas); j++ {
		if runas[j] != '$' || runas[j-1] == '\\' {
			continue
		}
		if j+1 < len(runas) && unicode.IsDigit(runas[j+1]) {
			continue
		}
		tex := string(runas[inicio+1 : j])
		// Con espazos nas beiras ($ {m} $) só se é LaTeX de verdade: así
		// «custa 5$ ou 6$» segue sendo texto.
		if (unicode.IsSpace(runas[inicio+1]) || unicode.IsSpace(runas[j-1])) && !pareceLatex(tex) {
			continue
		}
		if strings.TrimSpace(tex) == "" {
			return "", false, 0, false
		}
		return strings.TrimSpace(tex), false, j, true
	}
	return "", false, 0, false
}

// pareceLatex: o texto ten algo que só pode ser unha fórmula.
func pareceLatex(s string) bool { return strings.ContainsAny(s, "\\{}^_=") }

// enxeAbre di se a marca de énfase en runas[i] abre de verdade unha negrita
// (dobre) ou cursiva: ten que haber despois outra marca que a peche, e un "_"
// dentro dunha palabra (nome_ficheiro, 1_2) é literal. Sen isto, un "_" ou "*"
// solto poñía en cursiva o resto do parágrafo. Só se chama para abrir: unha
// marca que xa está aberta pécheas sempre.
func enxeAbre(runas []rune, i int, dobre bool) bool {
	c := runas[i]
	esAlfaNum := func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }
	if c == '_' && i > 0 && i+1 < len(runas) && esAlfaNum(runas[i-1]) && esAlfaNum(runas[i+1]) {
		return false
	}
	ini := i + 1
	if dobre {
		ini = i + 2
	}
	for j := ini; j < len(runas); j++ {
		if runas[j] != c {
			continue
		}
		parDobre := j+1 < len(runas) && runas[j+1] == c
		if dobre && parDobre {
			return true
		}
		if !dobre && !parDobre && !(j > 0 && runas[j-1] == c) {
			return true
		}
		if parDobre {
			j++ // salta a segunda marca do par
		}
	}
	return false
}

// parseTamanoLetraCrono analiza "[fs=clave]...[/fs]" que empeza en `inicio`.
func parseTamanoLetraCrono(runas []rune, inicio int) (interior string, escala float64, fin int, ok bool) {
	const abre = "[fs="
	if inicio+len(abre) >= len(runas) || string(runas[inicio:inicio+len(abre)]) != abre {
		return "", 0, 0, false
	}
	i := inicio + len(abre)
	var clave strings.Builder
	for i < len(runas) && runas[i] != ']' {
		clave.WriteRune(runas[i])
		i++
	}
	if i >= len(runas) {
		return "", 0, 0, false
	}
	escala = cronoEscalasTamano[clave.String()]
	if escala == 0 {
		return "", 0, 0, false
	}
	i++ // salta ']'
	pecha := []rune("[/fs]")
	for j := i; j+len(pecha) <= len(runas); j++ {
		if string(runas[j:j+len(pecha)]) == string(pecha) {
			return string(runas[i:j]), escala, j + len(pecha) - 1, true
		}
	}
	return "", 0, 0, false
}

// parseLigazonCrono analiza "[texto](url)" que empeza en `inicio` (onde
// runas[inicio]=='['). `fin` é o índice do ')' final.
func parseLigazonCrono(runas []rune, inicio int) (texto, url string, fin int, ok bool) {
	if inicio >= len(runas) || runas[inicio] != '[' {
		return "", "", 0, false
	}
	i := inicio + 1
	var t strings.Builder
	for i < len(runas) && runas[i] != ']' {
		t.WriteRune(runas[i])
		i++
	}
	if i >= len(runas) || runas[i] != ']' {
		return "", "", 0, false
	}
	i++
	if i >= len(runas) || runas[i] != '(' {
		return "", "", 0, false
	}
	i++
	var u strings.Builder
	for i < len(runas) && runas[i] != ')' {
		u.WriteRune(runas[i])
		i++
	}
	if i >= len(runas) || runas[i] != ')' {
		return "", "", 0, false
	}
	return t.String(), u.String(), i, true
}

// tamanoMM (método de cronoImaxeRef) resolve as dimensións finais en mm dunha imaxe, dado o ancho
// da columna de texto e o tamaño natural en píxeles. Se só se indica unha
// dimensión, a outra calcúlase mantendo a proporción; nunca excede o ancho da
// columna nin os 180 mm de alto.
func (r cronoImaxeRef) tamanoMM(colMM float64, wpx, hpx int) (w, h float64) {
	const maxAltoImaxeMM = 180.0
	if wpx <= 0 || hpx <= 0 {
		return 0, 0
	}
	razon := float64(hpx) / float64(wpx)
	switch {
	case r.AnchoPct > 0:
		w = colMM * r.AnchoPct / 100
		h = w * razon
	case r.AnchoMM > 0 && r.AltoMM > 0:
		w, h = r.AnchoMM, r.AltoMM
	case r.AnchoMM > 0:
		w = r.AnchoMM
		h = w * razon
	case r.AltoMM > 0:
		h = r.AltoMM
		w = h / razon
	default:
		// Tamaño natural a 96 dpi, limitado en alto para que unha foto
		// vertical non ocupe unha páxina enteira.
		w, h = float64(wpx)*25.4/96.0, float64(hpx)*25.4/96.0
		if h > maxAltoImaxeMM {
			w = w * maxAltoImaxeMM / h
			h = maxAltoImaxeMM
		}
	}
	if w > colMM {
		h = h * colMM / w
		w = colMM
	}
	return w, h
}

// ---------- Renderizado a LaTeX ----------

// latexTramosCrono escribe os tramos dun bloque como LaTeX. O texto vai escapado
// coma prosa (EscapeLatexProse, o mesmo que o resto de Lua).
func latexTramosCrono(tramos []cronoTramo) string {
	var b strings.Builder
	for _, t := range tramos {
		if t.Math != "" {
			// As macros defínense no corpo (formulas_estilo.go).
			if t.Display {
				b.WriteString(`\luaformulacentrada{` + t.Math + `}`)
			} else {
				b.WriteString(`\luaformula{` + t.Math + `}`)
			}
			continue
		}
		if t.cronoTexto() == "" {
			continue
		}
		var abre, pecha string
		if t.Escala > 0 && t.Escala != 1 {
			cmd := `\large `
			switch {
			case t.Escala < 1:
				cmd = `\small `
			case t.Escala >= 1.5:
				cmd = `\Large `
			}
			abre += "{" + cmd
			pecha = "}" + pecha
		}
		if t.Negrita {
			abre += `\textbf{`
			pecha = "}" + pecha
		}
		if t.Cursiva {
			abre += `\textit{`
			pecha = "}" + pecha
		}
		if t.URL != "" {
			// A ligazón vai activa no PDF e ademais impresa entre parénteses
			// (ver cronoTexto): no papel a URL non se pode premer.
			b.WriteString(abre)
			b.WriteString(`\href{` + urlLatexCrono(t.URL) + `}{` + EscapeLatexProse(t.Texto) + `}`)
			if t.URL != t.Texto && t.Texto != "" {
				b.WriteString(` (\nolinkurl{` + urlLatexCrono(t.URL) + `})`)
			}
			b.WriteString(pecha)
			continue
		}
		b.WriteString(abre + EscapeLatexProse(t.Texto) + pecha)
	}
	return b.String()
}

// urlLatexCrono escapa o mínimo que \href/\nolinkurl precisan.
func urlLatexCrono(u string) string {
	r := strings.NewReplacer(`\`, `/`, `%`, `\%`, `#`, `\#`, `{`, `%7B`, `}`, `%7D`)
	return r.Replace(strings.TrimSpace(u))
}

// envolverAlinCrono aplica o aliñamento dun bloque a un parágrafo de LaTeX.
func envolverAlinCrono(a cronoAlinamento, texto string) string {
	switch a {
	case cronoAlinCentro:
		return `{\centering ` + texto + `\par}` + "\n"
	case cronoAlinDereita:
		return `{\raggedleft ` + texto + `\par}` + "\n"
	case cronoAlinEsquerda:
		return `{\raggedright ` + texto + `\par}` + "\n"
	default:
		return texto + "\n\n"
	}
}

// ---------- Renderizado a Markdown (Pandoc) ----------

// mdEscaparCrono protexe o texto literal para que Pandoc non o tome por
// marcado. Só se escapan os caracteres que poden cambiar o sentido dentro dun
// parágrafo; os das nosas propias marcas xa se consumiron ao parsear.
func mdEscaparCrono(s string) string {
	r := strings.NewReplacer(
		`\`, `\\`, "`", "\\`", `*`, `\*`, `_`, `\_`, `[`, `\[`, `]`, `\]`,
		`<`, `\<`, `>`, `\>`, `#`, `\#`, `|`, `\|`,
	)
	return r.Replace(s)
}

func mdTramosCrono(tramos []cronoTramo) string {
	var b strings.Builder
	for _, t := range tramos {
		if t.Math != "" {
			if t.Display {
				b.WriteString("$$" + t.Math + "$$")
			} else {
				b.WriteString("$" + t.Math + "$")
			}
			continue
		}
		if t.cronoTexto() == "" {
			continue
		}
		var txt string
		switch {
		case t.URL != "":
			etiqueta := t.Texto
			if etiqueta == "" {
				etiqueta = t.URL
			}
			txt = "[" + mdEscaparCrono(etiqueta) + "](" + strings.ReplaceAll(t.URL, " ", "%20") + ")"
		default:
			txt = mdEscaparCrono(t.Texto)
		}
		// Os espazos das beiras fóra das marcas: "**a **b" non se le como
		// negrita en Markdown.
		esq := txt[:len(txt)-len(strings.TrimLeft(txt, " "))]
		dta := txt[len(strings.TrimRight(txt, " ")):]
		nucleo := strings.TrimSpace(txt)
		if nucleo == "" {
			b.WriteString(txt)
			continue
		}
		if t.Negrita {
			nucleo = "**" + nucleo + "**"
		}
		if t.Cursiva {
			nucleo = "*" + nucleo + "*"
		}
		b.WriteString(esq + nucleo + dta)
	}
	return b.String()
}
