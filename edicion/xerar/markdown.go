package xerar

// Fichas en Markdown: substitúense os {campos} co valor da variante e
// pásase o Markdown a LaTeX con pandoc. As pezas que xa son LaTeX final
// (gráficos de Maxima, TikZ, chemfig, exercicios matex dentro dun exame)
// van protexidas co rexistro de edicion/latex e só se resolven ao final.
//
// Regras dos {…} (as de Yang):
//   - na prosa: {nome} é o valor; {expr} avalíase (Maxima ou fórmula);
//     {'expr} é a expresión sen avaliar (Maxima); {nome | lista: "modelo"}
//     fai unha lista coas filas dunha táboa; {{ e }} son chaves literais.
//   - dentro de $…$ e dos bloques de código só se substitúe {nome} se nome
//     é un campo da ficha: o resto son chaves de LaTeX.

import (
	"bytes"
	"edicion/idioma"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"edicion/latex"
	"edicion/plataforma"
)

var (
	valladoRe     = regexp.MustCompile("(?s)(^|\n)(```+)([^\n]*)\n(.*?)\n```+[ \t]*(\n|$)")
	mathRe        = regexp.MustCompile(`(?s)\$\$.+?\$\$|\$[^$\n]+?\$`)
	listaRe       = regexp.MustCompile(`\{\s*([A-Za-z_]\w*)\s*\|\s*lista\s*:\s*"((?:[^"\\]|\\.)*)"\s*\}`)
	identChavesRe = regexp.MustCompile(`\{\s*([A-Za-z_]\w*)\s*\}`)
	imaxeMDRe     = regexp.MustCompile(`!\[([^\]]*)\]\(([^)\s]+(?:\s+"[^"]*")?)\)`)
	imaxeObsRe    = regexp.MustCompile(`!\[\[([^\]|]+\.(?:png|jpe?g|gif|pdf|svg|webp))(\|\d+)?\]\]`)
	campoColRe    = regexp.MustCompile(`\{([A-Za-z_]\w*)\}`)
)

// contextoMD: o que precisa a substitución nunha variante.
type contextoMD struct {
	vals   map[string]Valor
	reg    *latex.RawRegistry
	maxima func(expr string, avaliar bool) (string, error) // tex sen $; nil = sen Maxima
	plot   func(codigo string) (string, error)             // LaTeX final
	imaxe  func(ruta, ancho string) string                 // LaTeX final ou ""
	avisos *[]string
	// valadosCrus: os debuxos (```tikz, ```circuitikz, ```chemfig) quedan
	// coma bloques Markdown, só coas {variables} substituídas, para que a
	// aplicación os debuxe ela (Lua: problemas). Sen isto van protexidos
	// coma LaTeX final.
	valadosCrus bool
	decimais    int // redondeo das contas en Go (0 = sen redondear)
}

// Marca dun debuxo cru mentres se substitúe o resto (ver valadosCrus).
const marcaValado = "\x00VALADO%d\x00"

// substituir aplica as regras ao Markdown dunha ficha.
func (cx *contextoMD) substituir(md string) string {
	var crus []string
	// 1. Bloques con valla.
	md = valladoRe.ReplaceAllStringFunc(md, func(b string) string {
		m := valladoRe.FindStringSubmatch(b)
		pre, info, codigo, post := m[1], strings.TrimSpace(m[3]), m[4], m[5]
		if cx.valadosCrus {
			switch strings.ToLower(info) {
			case "tikz", "circuitikz", "chemfig":
				crus = append(crus, "```"+strings.ToLower(info)+"\n"+cx.soDeclarados(codigo)+"\n```")
				return pre + fmt.Sprintf(marcaValado, len(crus)-1) + post
			}
		}
		switch strings.ToLower(info) {
		case "plot":
			if cx.plot == nil {
				cx.aviso("hai un gráfico ```plot pero Maxima non está dispoñible")
				return pre + post
			}
			l, err := cx.plot(codigo)
			if err != nil {
				cx.aviso("gráfico: " + err.Error())
				return pre + post
			}
			return pre + "\n" + cx.reg.Protect(l) + "\n" + post
		case "tikz", "circuitikz":
			c := cx.soDeclarados(codigo)
			if !strings.Contains(c, `\begin{tikzpicture}`) && !strings.Contains(c, `\begin{circuitikz}`) {
				c = "\\begin{tikzpicture}\n" + c + "\n\\end{tikzpicture}"
			}
			return pre + "\n" + cx.reg.Protect("\\begin{center}\n"+c+"\n\\end{center}") + "\n" + post
		case "chemfig":
			return pre + "\n" + cx.reg.Protect("\\begin{center}\\chemfig{"+strings.TrimSpace(cx.soDeclarados(codigo))+"}\\end{center}") + "\n" + post
		case "{=latex}":
			return pre + "\n" + cx.reg.Protect(cx.soDeclarados(codigo)) + "\n" + post
		}
		return b
	})
	// 2. Imaxes da bóveda: ![texto](ruta) e ![[ficheiro.png]].
	if cx.imaxe != nil {
		md = imaxeMDRe.ReplaceAllStringFunc(md, func(m string) string {
			p := imaxeMDRe.FindStringSubmatch(m)
			if l := cx.imaxe(p[2], ""); l != "" {
				return "\n" + cx.reg.Protect(l) + "\n"
			}
			return m
		})
		md = imaxeObsRe.ReplaceAllStringFunc(md, func(m string) string {
			p := imaxeObsRe.FindStringSubmatch(m)
			ancho := ""
			if p[2] != "" {
				ancho = strings.TrimPrefix(p[2], "|") + "px"
				ancho = cssPx(ancho)
			}
			if l := cx.imaxe(p[1], ancho); l != "" {
				return "\n" + cx.reg.Protect(l) + "\n"
			}
			return m
		})
	}
	// 3. Listas de táboas (levan chaves dentro, van antes ca o resto).
	md = listaRe.ReplaceAllStringFunc(md, func(m string) string {
		l := listaRe.FindStringSubmatch(m)
		return cx.lista(l[1], strings.ReplaceAll(l[2], `\"`, `"`))
	})
	// 4. Matemáticas e prosa.
	var out strings.Builder
	ultimo := 0
	for _, m := range mathRe.FindAllStringIndex(md, -1) {
		out.WriteString(cx.prosa(md[ultimo:m[0]]))
		out.WriteString(cx.soDeclaradosTex(md[m[0]:m[1]]))
		ultimo = m[1]
	}
	out.WriteString(cx.prosa(md[ultimo:]))
	res := out.String()
	for i, c := range crus {
		res = strings.Replace(res, fmt.Sprintf(marcaValado, i), c, 1)
	}
	return res
}

func (cx *contextoMD) aviso(s string) { *cx.avisos = append(*cx.avisos, s) }

// soDeclarados: {nome} -> valor (texto) só para campos; o resto intacto.
// Coma nas fórmulas, unhas chaves pegadas a un comando (\mathrm{V}), a «^»,
// a «_» ou a outras chaves son LaTeX/TikZ, non unha variable.
func (cx *contextoMD) soDeclarados(s string) string {
	var b strings.Builder
	ultimo := 0
	for _, loc := range identChavesRe.FindAllStringIndex(s, -1) {
		m := s[loc[0]:loc[1]]
		b.WriteString(s[ultimo:loc[0]])
		ultimo = loc[1]
		v, ok := cx.vals[strings.TrimSpace(m[1:len(m)-1])]
		switch {
		case !ok || loc[0] > 0 && grupoLatex(s[loc[0]-1]):
			b.WriteString(m)
		case v.Tex != "":
			b.WriteString(v.Tex)
		default:
			b.WriteString(v.Texto)
		}
	}
	b.WriteString(s[ultimo:])
	return b.String()
}

// soDeclaradosTex: igual pero dentro de $…$ (valor en TeX). Unhas chaves
// pegadas a un comando, a «^», a «_» ou a outras chaves son un grupo de
// LaTeX e non se tocan: $\dfrac{d}{t}$ é a fórmula con letras; o valor vai
// con espazo, $\dfrac{ {d} }{ {t} }$ ou $ {d} $.
func (cx *contextoMD) soDeclaradosTex(s string) string {
	var b strings.Builder
	ultimo := 0
	for _, loc := range identChavesRe.FindAllStringIndex(s, -1) {
		m := s[loc[0]:loc[1]]
		b.WriteString(s[ultimo:loc[0]])
		ultimo = loc[1]
		if loc[0] > 0 && grupoLatex(s[loc[0]-1]) {
			b.WriteString(m)
			continue
		}
		t := strings.TrimSpace(m[1 : len(m)-1])
		v, ok := cx.vals[t]
		switch {
		case !ok:
			b.WriteString(m)
		case v.Tex != "":
			b.WriteString("{" + v.Tex + "}")
		default:
			b.WriteString("{\\text{" + latex.EscapeLatex(v.Texto) + "}}")
		}
	}
	b.WriteString(s[ultimo:])
	return b.String()
}

// grupoLatex: o carácter antes dunhas chaves fai delas un argumento de LaTeX.
func grupoLatex(c byte) bool {
	return c == '^' || c == '_' || c == '}' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

func (cx *contextoMD) prosa(s string) string {
	return tokenRe.ReplaceAllStringFunc(s, func(m string) string {
		switch m {
		case "{{":
			return "{"
		case "}}":
			return "}"
		}
		expr := strings.TrimSpace(m[1 : len(m)-1])
		if strings.HasPrefix(expr, "'") {
			return cx.avaliarMaxima(strings.TrimPrefix(expr, "'"), false, m)
		}
		if v, ok := cx.vals[expr]; ok {
			if v.EsNum || v.Mat {
				return "$" + v.Tex + "$"
			}
			return escaparMD(v.Texto)
		}
		if cx.maxima != nil {
			return cx.avaliarMaxima(expr, true, m)
		}
		if n, err := numeroFormula(expr, cx.vals); err == nil {
			_, tex := formatarNumeroDecimais(n, cx.decimais)
			return "$" + tex + "$"
		}
		cx.aviso(fmt.Sprintf("non se puido calcular {%s}", expr))
		return m
	})
}

func (cx *contextoMD) avaliarMaxima(expr string, avaliar bool, orixinal string) string {
	if cx.maxima == nil {
		cx.aviso(fmt.Sprintf("{%s} precisa Maxima", strings.Trim(orixinal, "{}")))
		return orixinal
	}
	tex, err := cx.maxima(expr, avaliar)
	if err != nil {
		cx.aviso(fmt.Sprintf("{%s}: %v", expr, err))
		return orixinal
	}
	return "$" + tex + "$"
}

// lista: unha lista numerada cunha liña por fila da táboa.
func (cx *contextoMD) lista(nome, modelo string) string {
	v, ok := cx.vals[nome]
	if !ok {
		cx.aviso(fmt.Sprintf("{%s | lista}: non hai ningún campo táboa con ese nome", nome))
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\n")
	for i, f := range v.Filas {
		liña := campoColRe.ReplaceAllStringFunc(modelo, func(m string) string {
			col := m[1 : len(m)-1]
			if val, ok := f[col]; ok {
				return strings.NewReplacer("{", "{{", "}", "}}").Replace(escaparMD(val))
			}
			return m
		})
		fmt.Fprintf(&b, "%d. %s\n", i+1, liña)
	}
	return b.String() + "\n"
}

// escaparMD: o texto dun campo non é Markdown (un «*» é un asterisco).
func escaparMD(s string) string {
	return strings.NewReplacer(`\`, `\\`, `*`, `\*`, `_`, `\_`, "`", "\\`", `#`, `\#`, `$`, `\$`, `[`, `\[`, `]`, `\]`).Replace(s)
}

func numeroFormula(expr string, vals map[string]Valor) (float64, error) {
	m := map[string]float64{}
	for k, v := range vals {
		if v.EsNum {
			m[k] = v.Num
		}
	}
	return avaliarFormula(expr, m)
}

// mdALatex pasa un anaco de Markdown a LaTeX con pandoc.
func mdALatex(md string) (string, error) {
	if strings.TrimSpace(md) == "" {
		return "", nil
	}
	if !plataforma.CommandExists("pandoc") {
		return "", idioma.Errorf("go.markdown.sen_pandoc", "non se atopou pandoc (precísase para as fichas en Markdown)")
	}
	cmd := exec.Command("pandoc", "-f", "markdown+tex_math_dollars+raw_tex-implicit_figures", "-t", "latex", "--wrap=preserve")
	plataforma.OcultarConsola(cmd)
	cmd.Stdin = strings.NewReader(md)
	var out, erro bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &erro
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("pandoc: %v: %s", err, strings.TrimSpace(erro.String()))
	}
	return out.String(), nil
}

// cssPx: «300px» -> ancho en cm para \includegraphics (96 px por polgada).
func cssPx(s string) string {
	var n float64
	if _, err := fmt.Sscanf(strings.TrimSuffix(s, "px"), "%g", &n); err != nil || n <= 0 {
		return ""
	}
	return fmt.Sprintf("%.2fcm", n/96*2.54)
}
