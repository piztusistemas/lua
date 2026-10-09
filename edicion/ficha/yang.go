package ficha

// Importar .matex de Yang. Hai dous casos:
//
//   - Documento da TÁBOA (leva o comentario <!--yang:tabla:N base64-->): o
//     modelo JSON incrustado ten as variables, os resultados, a resposta e a
//     solución de cada exercicio. Convértese sen perdas: cada exercicio é
//     unha ficha con campos e seccións, e ModeloTaboa reconstrúe o mesmo
//     modelo.
//   - Documento ANTIGO (bloques, texto, IA): só hai texto. Cada <EX>…</EX>
//     pasa a ser unha ficha co seu texto TAL CAL e o resto do documento queda
//     no exame, cunha ligazón ![[…]] onde estaba cada exercicio. TextoMatex
//     volve dar o documento orixinal byte a byte.
//
// Nos dous casos o texto queda no dialecto de Yang (formato matex) e xérase
// co motor de Yang: nada se reescribe, así que nada se estraga.

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Importacion é o resultado de importar un documento: o exame (a
// composición) e os exercicios, na orde do documento.
type Importacion struct {
	Exame      *Ficha
	Exercicios []*Ficha
	Avisos     []string
}

var (
	embedRe    = regexp.MustCompile(`^\x{feff}?\s*<!--\s*yang:tabla:(\d+)\s+([A-Za-z0-9+/=]+)\s*-->\r?\n?`)
	exRe       = regexp.MustCompile(`(?s)<EX>(.*?)</EX>`)
	hideRe     = regexp.MustCompile(`(?s)<HIDE>.*?</HIDE>`)
	ligazonRe  = regexp.MustCompile(`!\[\[([^\]\n]+)\]\]`)
	etiquetaRe = regexp.MustCompile(`(?s)<[^>]*>`)
	bloqueRe   = regexp.MustCompile(`(?i)<br\s*/?>|</(p|li|div|h[1-6]|tr)>|<(PLOT|TIKZ|SISTEMA|RESP|SOL)>`)
	slugMalRe  = regexp.MustCompile(`[^a-z0-9]+`)
)

// modeloTaboa reproduce o modelo de table-serialize.js (versión 2).
type modeloTaboa struct {
	V          int              `json:"v"`
	Exercicios []exercicioTaboa `json:"exercicios"`
}

type exercicioTaboa struct {
	ID         string           `json:"id"`
	Titulo     string           `json:"titulo"`
	Enunciado  string           `json:"enunciado"`
	Variables  []variableTaboa  `json:"variables"`
	Resultados []resultadoTaboa `json:"resultados"`
	Resposta   string           `json:"resposta"`
	Solucion   string           `json:"solucion"`
}

type variableTaboa struct {
	Nome  string `json:"nome"`
	Modo  string `json:"modo"`
	Valor string `json:"valor,omitempty"`
	Expr  string `json:"expr,omitempty"`
	DefOp string `json:"defOp,omitempty"`
	Xer   string `json:"xer,omitempty"`
	Min   string `json:"min,omitempty"`
	Max   string `json:"max,omitempty"`
	Paso  string `json:"paso,omitempty"`
}

type resultadoTaboa struct {
	Etiqueta string `json:"etiqueta"`
	Nome     string `json:"nome"`
	Formula  string `json:"formula"`
}

// textoJS: os campos do modelo poden vir coma número se o JSON se editou a
// man ou o escribiu a IA; table-serialize.js fai String(x). Aquí igual.
type textoJS string

func (t *textoJS) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		*t = textoJS(s)
		return nil
	}
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	if v == nil {
		*t = ""
		return nil
	}
	*t = textoJS(fmt.Sprint(v))
	return nil
}

// ImportarMatex le un .matex. `nome` é o nome do ficheiro sen extensión:
// dá o id do exame e o prefixo dos exercicios.
func ImportarMatex(texto, nome string) (*Importacion, error) {
	texto = strings.ReplaceAll(texto, "\r\n", "\n")
	base := Slug(nome)
	if base == "" {
		base = "exame"
	}
	if m := embedRe.FindStringSubmatch(texto); m != nil {
		modelo, err := lerModeloTaboa(m[2])
		if err != nil {
			return nil, err
		}
		return importarTaboa(modelo, nome, base), nil
	}
	return importarTexto(texto, nome, base), nil
}

// lerModeloTaboa descodifica e normaliza o modelo (normalizeModel de
// table-serialize.js).
func lerModeloTaboa(b64 string) (modeloTaboa, error) {
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return modeloTaboa{}, fmt.Errorf("o modelo da táboa está estragado: %w", err)
	}
	var bruto struct {
		Exercicios []struct {
			ID         textoJS              `json:"id"`
			Titulo     textoJS              `json:"titulo"`
			Enunciado  textoJS              `json:"enunciado"`
			Resposta   textoJS              `json:"resposta"`
			Solucion   textoJS              `json:"solucion"`
			Variables  []map[string]textoJS `json:"variables"`
			Resultados []map[string]textoJS `json:"resultados"`
		} `json:"exercicios"`
	}
	if err := json.Unmarshal(raw, &bruto); err != nil {
		return modeloTaboa{}, fmt.Errorf("o modelo da táboa non é JSON válido: %w", err)
	}
	m := modeloTaboa{V: 2}
	for i, ex := range bruto.Exercicios {
		e := exercicioTaboa{
			ID: string(ex.ID), Titulo: string(ex.Titulo), Enunciado: string(ex.Enunciado),
			Resposta: string(ex.Resposta), Solucion: string(ex.Solucion),
			Variables: []variableTaboa{}, Resultados: []resultadoTaboa{},
		}
		if e.ID == "" {
			e.ID = "e" + strconv.Itoa(i+1)
		}
		for _, v := range ex.Variables {
			e.Variables = append(e.Variables, normalizarVariable(v))
		}
		for j, r := range ex.Resultados {
			formula := strings.TrimSpace(string(r["formula"]))
			if formula == "" {
				continue
			}
			nome := strings.TrimSpace(string(r["nome"]))
			if !identRe.MatchString(nome) {
				nome = "res_" + strconv.Itoa(j+1)
			}
			e.Resultados = append(e.Resultados, resultadoTaboa{Etiqueta: strings.TrimSpace(string(r["etiqueta"])), Nome: nome, Formula: formula})
		}
		m.Exercicios = append(m.Exercicios, e)
	}
	return m, nil
}

func normalizarVariable(v map[string]textoJS) variableTaboa {
	modo := string(v["modo"])
	if modo != "fixa" && modo != "aleatoria" && modo != "auto" {
		modo = "aleatoria"
	}
	out := variableTaboa{Nome: string(v["nome"]), Modo: modo}
	switch modo {
	case "fixa":
		out.Valor = string(v["valor"])
	case "auto":
		out.Expr = string(v["expr"])
		if v["defOp"] == ":=" {
			out.DefOp = ":="
		}
	default:
		if v["xer"] == "rango" {
			out.Xer = "rango"
			out.Min = porDefecto(string(v["min"]), "1")
			out.Max = porDefecto(string(v["max"]), "10")
			out.Paso = porDefecto(string(v["paso"]), "1")
		} else {
			out.Xer = string(v["xer"])
			if !conten(xeradoresYang, out.Xer) {
				out.Xer = "n1"
			}
		}
	}
	return out
}

func porDefecto(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

func importarTaboa(m modeloTaboa, nome, base string) *Importacion {
	imp := &Importacion{}
	exame := &Ficha{Luayang: Version, ID: base, Tipo: TipoExame, Titulo: nome, Importado: &Importado{De: "yang-taboa"}}
	var ligazons []string
	for i, ex := range m.Exercicios {
		f := &Ficha{
			Luayang:   Version,
			ID:        fmt.Sprintf("%s-%d", base, i+1),
			Tipo:      TipoExercicio,
			Titulo:    ex.Titulo,
			Formato:   FormatoMatex,
			Importado: &Importado{De: "yang-taboa", ID: ex.ID},
		}
		if f.Titulo == "" {
			f.Titulo = tituloDoTexto(ex.Enunciado, fmt.Sprintf("%s · exercicio %d", nome, i+1))
			f.Importado.SenTitulo = true
		}
		for _, v := range ex.Variables {
			f.Campos = append(f.Campos, campoDeVariable(v))
		}
		for _, r := range ex.Resultados {
			f.Campos = append(f.Campos, Campo{Nome: r.Nome, Orixe: OrixeCalculado, Motor: MotorMaxima, Expr: r.Formula, Rol: RolResultado, Etiqueta: r.Etiqueta})
		}
		ss := []Seccion{{Titulo: "Enunciado", Texto: strings.Trim(ex.Enunciado, "\n")}}
		if strings.TrimSpace(ex.Resposta) != "" {
			ss = append(ss, Seccion{Titulo: "Resultado", Texto: strings.Trim(ex.Resposta, "\n")})
		}
		if strings.TrimSpace(ex.Solucion) != "" {
			ss = append(ss, Seccion{Titulo: "Resolución", Texto: strings.Trim(ex.Solucion, "\n")})
		}
		f.Corpo = CorpoDeSeccions(ss)
		for _, s := range ss {
			if strings.Contains("\n"+s.Texto, "\n## ") {
				imp.Avisos = append(imp.Avisos, fmt.Sprintf("%s: o texto ten liñas que empezan por «## » e partirían a sección", f.ID))
			}
		}
		imp.Exercicios = append(imp.Exercicios, f)
		ligazons = append(ligazons, "![["+f.ID+"]]")
	}
	exame.Corpo = strings.Join(ligazons, "\n\n") + "\n"
	imp.Exame = exame
	return imp
}

func campoDeVariable(v variableTaboa) Campo {
	c := Campo{Nome: v.Nome}
	switch v.Modo {
	case "fixa":
		c.Orixe, c.Valor = OrixeFixo, v.Valor
	case "auto":
		c.Orixe, c.Motor, c.Expr = OrixeCalculado, MotorMaxima, v.Expr
		c.Funcion = v.DefOp == ":="
	default:
		c.Orixe, c.Xerador = OrixeAleatorio, v.Xer
		if v.Xer == "rango" {
			c.Xerador, c.De, c.Ata = XeradorRango, v.Min, v.Max
			if v.Paso != "1" {
				c.Paso = v.Paso
			}
		}
	}
	return c
}

func variableDeCampo(c Campo) variableTaboa {
	v := variableTaboa{Nome: c.Nome}
	switch c.Orixe {
	case OrixeFixo:
		v.Modo, v.Valor = "fixa", c.Valor
	case OrixeCalculado:
		v.Modo, v.Expr = "auto", c.Expr
		if c.Funcion {
			v.DefOp = ":="
		}
	default:
		v.Modo, v.Xer = "aleatoria", c.Xerador
		if c.Xerador == XeradorRango {
			v.Min, v.Max, v.Paso = c.De, c.Ata, porDefecto(c.Paso, "1")
		}
	}
	return v
}

// ModeloTaboa reconstrúe o modelo da táboa de Yang a partir dos exercicios
// (para comprobar a ida e volta, e para quen queira abrilos en Yang).
func ModeloTaboa(exercicios []*Ficha) modeloTaboa {
	m := modeloTaboa{V: 2}
	for _, f := range exercicios {
		e := exercicioTaboa{Titulo: f.Titulo, Variables: []variableTaboa{}, Resultados: []resultadoTaboa{}}
		if f.Importado != nil {
			e.ID = f.Importado.ID
			if f.Importado.SenTitulo {
				e.Titulo = ""
			}
		}
		for _, c := range f.Campos {
			if c.Rol == RolResultado {
				e.Resultados = append(e.Resultados, resultadoTaboa{Etiqueta: c.Etiqueta, Nome: c.Nome, Formula: c.Expr})
			} else {
				e.Variables = append(e.Variables, variableDeCampo(c))
			}
		}
		for _, s := range f.Seccions() {
			switch s.Rol {
			case RolEnunciado:
				e.Enunciado = s.Texto
			case RolResultado:
				e.Resposta = s.Texto
			case RolResolucion:
				e.Solucion = s.Texto
			}
		}
		m.Exercicios = append(m.Exercicios, e)
	}
	return m
}

func importarTexto(texto, nome, base string) *Importacion {
	imp := &Importacion{}
	exame := &Ficha{Luayang: Version, ID: base, Tipo: TipoExame, Titulo: nome, Formato: FormatoMatex, Importado: &Importado{De: "yang"}}
	n := 0
	exame.Corpo = exRe.ReplaceAllStringFunc(texto, func(bloque string) string {
		n++
		dentro := exRe.FindStringSubmatch(bloque)[1]
		f := &Ficha{
			Luayang:   Version,
			ID:        fmt.Sprintf("%s-%d", base, n),
			Tipo:      TipoExercicio,
			Titulo:    tituloDoTexto(dentro, fmt.Sprintf("%s · exercicio %d", nome, n)),
			Formato:   FormatoMatex,
			Importado: &Importado{De: "yang"},
			Corpo:     dentro,
		}
		if strings.Contains("\n"+dentro, "\n## ") {
			imp.Avisos = append(imp.Avisos, fmt.Sprintf("%s: o texto ten liñas que empezan por «## » e partirían a sección", f.ID))
		}
		imp.Exercicios = append(imp.Exercicios, f)
		return "![[" + f.ID + "]]"
	})
	if n == 0 {
		imp.Avisos = append(imp.Avisos, "o documento non ten exercicios <EX>: todo queda no exame")
	}
	if hideRe.MatchString(exRe.ReplaceAllString(texto, "")) {
		imp.Avisos = append(imp.Avisos, "hai <HIDE> fóra dos exercicios: os exercicios poden usar variables que só existen dentro do exame")
	}
	imp.Exame = exame
	return imp
}

// TextoMatex volve compoñer o .matex dun documento antigo importado.
func TextoMatex(exame *Ficha, exercicios []*Ficha) (string, error) {
	porID := map[string]*Ficha{}
	for _, f := range exercicios {
		porID[f.ID] = f
	}
	var falta []string
	out := ligazonRe.ReplaceAllStringFunc(exame.Corpo, func(l string) string {
		id := ligazonRe.FindStringSubmatch(l)[1]
		f, ok := porID[id]
		if !ok {
			falta = append(falta, id)
			return l
		}
		return "<EX>" + f.Corpo + "</EX>"
	})
	if len(falta) > 0 {
		return "", fmt.Errorf("faltan os exercicios %s", strings.Join(falta, ", "))
	}
	return out, nil
}

// tituloDoTexto: un título curto sacado da primeira liña con texto (sen
// etiquetas), ou `defecto`.
func tituloDoTexto(texto, defecto string) string {
	limpo := hideRe.ReplaceAllString(texto, "")
	limpo = etiquetaRe.ReplaceAllString(bloqueRe.ReplaceAllString(limpo, "\n"), "")
	for _, l := range strings.Split(limpo, "\n") {
		l = strings.Join(strings.Fields(l), " ")
		l = strings.TrimRight(l, ":.")
		if l == "" {
			continue
		}
		if r := []rune(l); len(r) > 70 {
			l = strings.TrimSpace(string(r[:70])) + "…"
		}
		return l
	}
	return defecto
}

// Slug: un nome de ficheiro seguro e lexible («Exame cuarto» -> «exame-cuarto»).
func Slug(s string) string {
	s = slugMalRe.ReplaceAllString(clave(s), "-")
	return strings.Trim(s, "-")
}
