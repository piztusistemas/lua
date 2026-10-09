package xerar

// Fichas en formato matex: compóñense coma o .matex que xeraba Yang
// (modelToBody de table-serialize.js, portado aquí) e pásanse polo mesmo
// motor (cas + documento). Así un exercicio importado de Yang sae igual ca
// en Yang.

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"edicion/ficha"
)

var (
	identRe = regexp.MustCompile(`^[A-Za-z_]\w*$`)
	tokenRe = regexp.MustCompile(`\{\{|\}\}|\{([^{}\n]+)\}`)
	// bloquesCodigo: etiquetas con código dentro (TikZ, LaTeX, Maxima) onde
	// unha chave é sintaxe da linguaxe e non un token.
	bloquesCodigoRe  = regexp.MustCompile(`(?s)<(TIKZ|TEX|PLOT|MAT|EVAL|HIDE|SISTEMA)>.*?</(TIKZ|TEX|PLOT|MAT|EVAL|HIDE|SISTEMA)>`)
	etiquetaInicioRe = regexp.MustCompile(`^<([A-Za-z]+)>`)
)

var maximaBloques = map[string]bool{"MAT": true, "EVAL": true, "HIDE": true, "SISTEMA": true}

// tokensAEval: «área de {b}» -> «área de <EVAL>b</EVAL>» (tokensToEval de
// Yang). Na prosa toda chave é un token; dentro dos bloques de código só os
// identificadores declarados.
func tokensAEval(texto string, declarados map[string]bool) string {
	var out strings.Builder
	ultimo := 0
	for _, m := range bloquesCodigoRe.FindAllStringIndex(texto, -1) {
		out.WriteString(tokensProsa(texto[ultimo:m[0]]))
		bloque := texto[m[0]:m[1]]
		tag := etiquetaInicioRe.FindStringSubmatch(bloque)[1]
		out.WriteString(tokensCodigo(bloque, declarados, maximaBloques[strings.ToUpper(tag)]))
		ultimo = m[1]
	}
	out.WriteString(tokensProsa(texto[ultimo:]))
	return out.String()
}

func tokensProsa(s string) string {
	return tokenRe.ReplaceAllStringFunc(s, func(m string) string {
		switch m {
		case "{{":
			return "{"
		case "}}":
			return "}"
		}
		return "<EVAL>" + strings.TrimSpace(m[1:len(m)-1]) + "</EVAL>"
	})
}

func tokensCodigo(s string, declarados map[string]bool, maxima bool) string {
	return tokenRe.ReplaceAllStringFunc(s, func(m string) string {
		switch m {
		case "{{":
			return "{"
		case "}}":
			return "}"
		}
		t := strings.TrimSpace(m[1 : len(m)-1])
		if identRe.MatchString(t) && declarados[t] {
			if maxima {
				return t
			}
			return "<EVAL>" + t + "</EVAL>"
		}
		return m
	})
}

// liñaMaxima: o que vai no <HIDE> para un campo (varToMaxima de Yang, máis
// as orixes novas). Os valores que xa se sortearon en Go (táboas, campos
// escritos) chegan en `fixos`.
func liñaMaxima(c ficha.Campo, fixos map[string]Valor) string {
	nome := strings.TrimSpace(c.Nome)
	if nome == "" {
		return ""
	}
	if v, ok := fixos[ficha.NomeBase(nome)]; ok {
		if v.EsNum {
			return fmt.Sprintf("%s: %s$", nome, strconv.FormatFloat(v.Num, 'g', -1, 64))
		}
		return fmt.Sprintf("%s: %s$", nome, cadeaMaxima(v.Texto))
	}
	switch c.Orixe {
	case ficha.OrixeFixo:
		if strings.TrimSpace(c.Valor) == "" {
			return ""
		}
		return fmt.Sprintf("%s: %s$", nome, strings.TrimSpace(c.Valor))
	case ficha.OrixeCalculado:
		ex := strings.TrimSpace(c.Expr)
		if ex == "" {
			return ""
		}
		if c.Funcion || strings.Contains(nome, "(") {
			return fmt.Sprintf("%s := %s$", nome, ex)
		}
		return fmt.Sprintf("%s: %s$", nome, ex)
	case ficha.OrixeAleatorio:
		switch c.Xerador {
		case ficha.XeradorRango:
			min, max, paso := porDefecto(c.De, "1"), porDefecto(c.Ata, "10"), porDefecto(c.Paso, "1")
			if paso == "1" {
				n := fmt.Sprintf("((%s)-(%s)+1)", max, min)
				if a, e1 := strconv.ParseFloat(min, 64); e1 == nil {
					if b, e2 := strconv.ParseFloat(max, 64); e2 == nil {
						n = strconv.FormatFloat(b-a+1, 'g', -1, 64)
					}
				}
				return fmt.Sprintf("%s: (%s) + random(%s)$", nome, min, n)
			}
			return fmt.Sprintf("%s: (%s) + (%s)*random(floor(((%s)-(%s))/(%s))+1)$", nome, min, paso, max, min, paso)
		case ficha.XeradorLista:
			elems := make([]string, len(c.Valores))
			for i, v := range c.Valores {
				if _, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
					elems[i] = strings.TrimSpace(v)
				} else {
					elems[i] = cadeaMaxima(v)
				}
			}
			return fmt.Sprintf("%s: [%s][1+random(%d)]$", nome, strings.Join(elems, ", "), len(elems))
		default:
			x := c.Xerador
			if x == "" {
				x = "n1"
			}
			return fmt.Sprintf("%s: %s()$", nome, x)
		}
	}
	return ""
}

func cadeaMaxima(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

func porDefecto(s, d string) string {
	if strings.TrimSpace(s) == "" {
		return d
	}
	return strings.TrimSpace(s)
}

// textoMatex compón o .matex dun exercicio en formato matex (o <EX> sen as
// etiquetas de fóra).
func textoMatex(f *ficha.Ficha, fixos map[string]Valor) string {
	seccions := f.Seccions()
	// Exercicio antigo (sen títulos de sección): o texto vai tal cal.
	if len(seccions) <= 1 && (len(seccions) == 0 || seccions[0].Titulo == "") && len(f.Campos) == 0 {
		return f.Corpo
	}
	var partes []string
	var hide []string
	declarados := map[string]bool{}
	orde, _ := ficha.OrdeCalculo(f)
	porNome := map[string]ficha.Campo{}
	for _, c := range f.Campos {
		porNome[ficha.NomeBase(c.Nome)] = c
		declarados[ficha.NomeBase(c.Nome)] = true
	}
	// As variables do docente primeiro e os resultados ao final (orde do
	// <HIDE> de Yang), respectando as dependencias.
	for _, resultado := range []bool{false, true} {
		for _, n := range orde {
			c := porNome[n]
			if (c.Rol == ficha.RolResultado) != resultado {
				continue
			}
			if l := liñaMaxima(c, fixos); l != "" {
				hide = append(hide, l)
			}
		}
	}
	if len(hide) > 0 {
		partes = append(partes, "<HIDE>"+strings.Join(hide, " ")+"</HIDE>")
	}
	var resposta, solucion string
	for _, s := range seccions {
		switch s.Rol {
		case ficha.RolResultado:
			resposta = s.Texto
		case ficha.RolResolucion:
			solucion = s.Texto
		default:
			if t := strings.TrimSpace(tokensAEval(s.Texto, declarados)); t != "" {
				partes = append(partes, t)
			}
		}
	}
	if strings.TrimSpace(resposta) == "" {
		resposta = respostaDesdeResultados(f)
	}
	if strings.TrimSpace(resposta) != "" {
		partes = append(partes, "<RESP>"+strings.TrimSpace(tokensAEval(resposta, declarados))+"</RESP>")
	}
	if strings.TrimSpace(solucion) != "" {
		partes = append(partes, "<SOL>"+strings.TrimSpace(tokensAEval(solucion, declarados))+"</SOL>")
	}
	return "\n" + strings.Join(partes, "\n") + "\n"
}

// respostaDesdeResultados: «etiqueta: <MAT>fórmula</MAT> = {res_n}» por cada
// resultado (respostaDesdeResultados de Yang).
func respostaDesdeResultados(f *ficha.Ficha) string {
	var liñas []string
	for _, c := range f.Campos {
		if c.Rol != ficha.RolResultado || strings.TrimSpace(c.Expr) == "" {
			continue
		}
		pref := ""
		if et := strings.TrimSpace(c.Etiqueta); et != "" {
			pref = "<b>" + strings.NewReplacer("{", "{{", "}", "}}").Replace(et) + ":</b> "
		}
		liñas = append(liñas, fmt.Sprintf("<p>%s<MAT>%s</MAT> = {%s}</p>", pref, strings.TrimSpace(c.Expr), c.Nome))
	}
	return strings.Join(liñas, "\n")
}
