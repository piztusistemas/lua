package main

// Formularios web para o asistente: moitas fontes de datos públicos (por
// exemplo, as series das estacións meteorolóxicas de datosclima.es) só dan os
// datos despois de enviar un formulario (escoller provincia, estación,
// datas...). web_consultar con formularios=true describe os formularios dunha
// páxina e web_enviar_formulario envíaos. Teñen as mesmas proteccións ca
// web_consultar (pedirWeb): só enderezos públicos, tamaño e tempo máximos.
//
// Non se executa o JavaScript das páxinas: se un script cambia a acción do
// formulario (document.form.action = "..."), lístase para que a IA a use.

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	xhtml "golang.org/x/net/html"
)

const opcionsMaxPorSelect = 400

// accionScriptRe: unha acción de formulario posta por un script.
var accionScriptRe = regexp.MustCompile(`\.action\s*=\s*["']([^"'$<>{}\s]+)["']`)

// webFormularios descarga unha páxina e describe os seus formularios.
func webFormularios(rawURL string) (string, error) {
	corpo, _, final, err := pedirWeb(http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	return formulariosDeHTML(corpo, final), nil
}

// webEnviarFormulario envía os campos (POST por defecto, ou GET) e devolve o
// texto da resposta por partes, coma web_consultar.
func webEnviarFormulario(rawURL, metodo string, campos map[string]any, inicio, maxCaracteres int) (string, error) {
	corpo, tipo, _, err := enviarFormularioWeb(rawURL, metodo, campos)
	if err != nil {
		return "", err
	}
	return textoWebEnPartes(corpo, tipo, inicio, maxCaracteres)
}

// webEnviarFormularioEVerFormularios envía o formulario e describe os
// formularios da resposta (cando un select reenvía a páxina para encher o
// seguinte: provincia -> estacións).
func webEnviarFormularioEVerFormularios(rawURL, metodo string, campos map[string]any) (string, error) {
	corpo, _, final, err := enviarFormularioWeb(rawURL, metodo, campos)
	if err != nil {
		return "", err
	}
	return formulariosDeHTML(corpo, final), nil
}

func enviarFormularioWeb(rawURL, metodo string, campos map[string]any) ([]byte, string, *url.URL, error) {
	metodo = strings.ToUpper(strings.TrimSpace(metodo))
	if metodo == "" {
		metodo = http.MethodPost
	}
	valores := url.Values{}
	for k, v := range campos {
		switch x := v.(type) {
		case []any:
			for _, e := range x {
				valores.Add(k, valorCampoWeb(e))
			}
		default:
			valores.Set(k, valorCampoWeb(x))
		}
	}
	return pedirWeb(metodo, rawURL, valores)
}

// valorCampoWeb: o texto dun valor que manda a IA (os números do JSON veñen
// coma float64: 2005, non 2005.000000).
func valorCampoWeb(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(x)
	default:
		return fmt.Sprint(x)
	}
}

// formulariosDeHTML describe en texto os formularios dunha páxina: acción
// (resolta contra `base`), método e campos, coas opcións dos select. Ao final
// van as accións que poñen os scripts.
func formulariosDeHTML(corpo []byte, base *url.URL) string {
	doc, err := xhtml.Parse(bytes.NewReader(corpo))
	if err != nil {
		return "Non se puido ler o HTML da páxina."
	}
	var b strings.Builder
	var scripts strings.Builder
	n := 0
	var percorrer func(*xhtml.Node)
	percorrer = func(nd *xhtml.Node) {
		if nd.Type == xhtml.ElementNode {
			switch nd.Data {
			case "form":
				n++
				describirFormulario(&b, nd, n, base)
			case "script":
				for c := nd.FirstChild; c != nil; c = c.NextSibling {
					if c.Type == xhtml.TextNode {
						scripts.WriteString(c.Data + "\n")
					}
				}
			}
		}
		for c := nd.FirstChild; c != nil; c = c.NextSibling {
			percorrer(c)
		}
	}
	percorrer(doc)
	if n == 0 {
		b.WriteString("A páxina non ten formularios.\n")
	}
	var accions []string
	vistas := map[string]bool{}
	for _, m := range accionScriptRe.FindAllStringSubmatch(scripts.String(), -1) {
		a := resolverWeb(base, m[1])
		if !vistas[a] {
			vistas[a] = true
			accions = append(accions, a)
		}
	}
	if len(accions) > 0 {
		b.WriteString("\nAccións que poñen os scripts da páxina (o envío de verdade pode ir a unha destas, cos mesmos campos): " + strings.Join(accions, ", ") + "\n")
	}
	s := b.String()
	if r := []rune(s); len(r) > webCaracteresMax {
		s = string(r[:webCaracteresMax]) + "\n[... cortado ...]"
	}
	return s
}

func describirFormulario(b *strings.Builder, form *xhtml.Node, n int, base *url.URL) {
	metodo := strings.ToUpper(strings.TrimSpace(atributo(form, "method")))
	if metodo != http.MethodPost {
		metodo = http.MethodGet
	}
	nome := ""
	if v := atributo(form, "name"); v != "" {
		nome = " «" + v + "»"
	} else if v := atributo(form, "id"); v != "" {
		nome = " «" + v + "»"
	}
	fmt.Fprintf(b, "Formulario %d%s: %s %s\n", n, nome, metodo, resolverWeb(base, atributo(form, "action")))
	var campos func(*xhtml.Node)
	campos = func(nd *xhtml.Node) {
		if nd.Type == xhtml.ElementNode {
			switch nd.Data {
			case "input":
				describirInput(b, nd)
			case "textarea":
				if nc := atributo(nd, "name"); nc != "" {
					fmt.Fprintf(b, "- %s (textarea) = «%s»\n", nc, strings.TrimSpace(textoNodo(nd)))
				}
			case "select":
				describirSelect(b, nd)
				return
			}
		}
		for c := nd.FirstChild; c != nil; c = c.NextSibling {
			campos(c)
		}
	}
	for c := form.FirstChild; c != nil; c = c.NextSibling {
		campos(c)
	}
}

func describirInput(b *strings.Builder, nd *xhtml.Node) {
	nc := atributo(nd, "name")
	tipo := strings.ToLower(atributo(nd, "type"))
	if tipo == "" {
		tipo = "text"
	}
	if nc == "" || tipo == "button" || tipo == "reset" || tipo == "image" || tipo == "file" {
		return
	}
	extra := ""
	if (tipo == "checkbox" || tipo == "radio") && temAtributo(nd, "checked") {
		extra = " (marcada)"
	}
	fmt.Fprintf(b, "- %s (%s) = «%s»%s\n", nc, tipo, atributo(nd, "value"), extra)
}

func describirSelect(b *strings.Builder, nd *xhtml.Node) {
	nc := atributo(nd, "name")
	if nc == "" {
		return
	}
	envia := ""
	if strings.Contains(strings.ToLower(atributo(nd, "onchange")), "submit") {
		envia = " — a páxina reenvía o formulario ao cambiala (envíao con este valor para ver os campos seguintes)"
	}
	var ops []string
	total := 0
	var buscar func(*xhtml.Node)
	buscar = func(o *xhtml.Node) {
		if o.Type == xhtml.ElementNode && o.Data == "option" {
			total++
			if total <= opcionsMaxPorSelect {
				etiqueta := strings.Join(strings.Fields(textoNodo(o)), " ")
				valor := etiqueta
				if temAtributo(o, "value") {
					valor = atributo(o, "value")
				}
				op := "«" + valor + "» = " + etiqueta
				if temAtributo(o, "selected") {
					op += " (escollida)"
				}
				ops = append(ops, op)
			}
			return
		}
		for c := o.FirstChild; c != nil; c = c.NextSibling {
			buscar(c)
		}
	}
	buscar(nd)
	if total > opcionsMaxPorSelect {
		ops = append(ops, fmt.Sprintf("... e %d máis", total-opcionsMaxPorSelect))
	}
	fmt.Fprintf(b, "- %s (select)%s: %s\n", nc, envia, strings.Join(ops, "; "))
}

func atributo(nd *xhtml.Node, nome string) string {
	for _, a := range nd.Attr {
		if strings.EqualFold(a.Key, nome) {
			return a.Val
		}
	}
	return ""
}

func temAtributo(nd *xhtml.Node, nome string) bool {
	for _, a := range nd.Attr {
		if strings.EqualFold(a.Key, nome) {
			return true
		}
	}
	return false
}

func textoNodo(nd *xhtml.Node) string {
	var b strings.Builder
	var f func(*xhtml.Node)
	f = func(n *xhtml.Node) {
		if n.Type == xhtml.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(nd)
	return b.String()
}

// resolverWeb: a URL absoluta dunha ligazón (relativa á páxina).
func resolverWeb(base *url.URL, ref string) string {
	ref = strings.TrimSpace(ref)
	if base == nil {
		return ref
	}
	r, err := url.Parse(ref)
	if err != nil {
		return ref
	}
	return base.ResolveReference(r).String()
}
