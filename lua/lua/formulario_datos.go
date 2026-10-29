package main

// Motor de formularios (capa 2: os DATOS). Un DatosFormulario é o que o
// usuario énchea para un formulario: os valores de cada campo (á man, por CSV
// ou coa IA: todos acaban neste mesmo JSON) e o contexto do documento (centro,
// curso, logotipo...), que é o que enche {{CENTRO}}, {{CURSO}} e demais
// variables da plantilla.
//
// Os valores viaxan coma JSON solto (map[string]any): texto, número, bool,
// e para as táboas unha lista de filas {columna: valor}. Aquí non hai
// validación estrita que impida xerar: un dato malo dá un AVISO e o documento
// sae igual (un docente prefire ver o PDF cun aviso a un erro que non o deixa
// ver nada).

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ContextoFormulario son os datos do documento que non son campos do
// formulario: quen o asina e onde.
type ContextoFormulario struct {
	Centro       string `json:"centro,omitempty"`
	Localidade   string `json:"localidade,omitempty"`
	Curso        string `json:"curso,omitempty"` // "2025-2026"
	Departamento string `json:"departamento,omitempty"`
	Responsable  string `json:"responsable,omitempty"` // quen asina ({{XEFE}}, {{ORIENTADOR}})
	Cargo        string `json:"cargo,omitempty"`
	Logo         string `json:"logo,omitempty"` // data URI
}

// DatosFormulario é un documento enchido.
type DatosFormulario struct {
	Formulario string             `json:"formulario"` // id do formulario
	Contexto   ContextoFormulario `json:"contexto"`
	Valores    map[string]any     `json:"valores"`
	// SenReferencias: o usuario desactivou as referencias ([PAU 2021 · ordinaria],
	// [clave OFICIAL…]) que a IA deixa dentro dos textos: non saen no documento.
	// Os datos non se tocan; só se quitan ao xerar.
	SenReferencias bool `json:"senReferencias,omitempty"`
	// Xeracion: semente, variantes e solucións dos campos de problemas
	// (formulario_problemas.go). nil = unha copia, semente 0, sen solucións.
	Xeracion *XeracionProblemas `json:"xeracion,omitempty"`
}

// PedidoFormulario é o que manda o frontend ao xerar un formulario: o esquema
// vai xunto cos datos (así a vista previa funciona co borrador aínda sen
// gardar, e o backend non depende de que a biblioteca coincida).
type PedidoFormulario struct {
	Formulario Formulario      `json:"formulario"`
	Datos      DatosFormulario `json:"datos"`
	// motor: o que calcula os problemas (o pon a App ao xerar; nil = sen
	// problemas que calcular, ou un motor só en Go).
	motor motorProblemasDoc
	// formulas: estilo e tamaño das fórmulas (Opcións; o pon a App ao xerar).
	formulas opcionsFormulas
}

// textoValor devolve o valor coma texto recortado, sen formatar.
func textoValor(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(x)
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case bool:
		return strconv.FormatBool(x)
	default:
		return strings.TrimSpace(fmt.Sprint(x))
	}
}

// valorBool interpreta un valor de verificación: bool, ou o texto que sae dun
// CSV ou da IA («si», «sí», «true», «1», «x»...).
func valorBool(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case float64:
		return x != 0
	case string:
		switch strings.ToLower(strings.TrimSpace(x)) {
		case "true", "si", "sí", "s", "1", "x", "yes", "y", "verdadeiro", "certo":
			return true
		}
	}
	return false
}

// valorNumero le un número con coma ou punto decimal («3,5», «3.5», 3.5) e
// devolve o seu texto galego (coma decimal, sen ceros de máis). ok é falso se
// non é un número.
func valorNumero(v any) (string, bool) {
	var f float64
	switch x := v.(type) {
	case float64:
		f = x
	case int:
		f = float64(x)
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return "", true
		}
		// «1.234,5» (miles co punto, decimal coa coma) e «1234.5» valen os dous.
		if strings.Contains(s, ",") {
			s = strings.ReplaceAll(s, ".", "")
			s = strings.ReplaceAll(s, ",", ".")
		}
		var err error
		if f, err = strconv.ParseFloat(s, 64); err != nil {
			return x, false
		}
	case nil:
		return "", true
	default:
		return textoValor(v), false
	}
	return strings.ReplaceAll(strconv.FormatFloat(f, 'f', -1, 64), ".", ","), true
}

// valorData le AAAA-MM-DD (ou DD/MM/AAAA, que é o que pon quen teclea) e
// devolve DD/MM/AAAA. ok é falso se non é unha data.
func valorData(v any) (string, bool) {
	s := textoValor(v)
	if s == "" {
		return "", true
	}
	for _, formato := range []string{"2006-01-02", "02/01/2006", "2/1/2006"} {
		if t, err := time.Parse(formato, s); err == nil {
			return t.Format("02/01/2006"), true
		}
	}
	return s, false
}

// filasDatos devolve as filas dunha táboa tal e como chegan (lista de
// obxectos {columna: valor}); ignora o que non sexa unha fila.
func filasDatos(v any) []map[string]any {
	var out []map[string]any
	switch x := v.(type) {
	case []map[string]any:
		out = x
	case []any:
		for _, it := range x {
			if m, ok := it.(map[string]any); ok {
				out = append(out, m)
			}
		}
	}
	return out
}

// celaFormatada devolve o valor dunha columna (ou dun campo simple) listo para
// imprimir. Os textos fixos («Si», «Non») veñen dos textos do documento.
// Devolve tamén se o valor é aceptable para o seu tipo.
func celaFormatada(c CampoFormulario, v any, tx textosDocumento) (string, bool) {
	switch c.Tipo {
	case campoNumero:
		return valorNumero(v)
	case campoData:
		return valorData(v)
	case campoVerificacion:
		if valorBool(v) {
			return tx.T("FORM_SI"), true
		}
		return tx.T("FORM_NON"), true
	case campoEscolla:
		s := textoValor(v)
		if s == "" {
			return "", true
		}
		for _, o := range c.Opcions {
			if strings.EqualFold(o, s) {
				return o, true
			}
		}
		return s, false
	default:
		return textoValor(v), true
	}
}

// valorVacio di se un campo non ten nada que amosar. Unha verificación nunca
// está baleira: «Non» tamén é unha resposta.
func valorVacio(c CampoFormulario, v any) bool {
	switch c.Tipo {
	case campoVerificacion:
		return false
	case campoTaboa:
		return len(filasNonVacias(c, v)) == 0
	case campoProblemas:
		return len(problemasNonVacios(v)) == 0
	default:
		return textoValor(v) == ""
	}
}

// filasNonVacias son as filas da táboa con algo escrito nalgunha columna.
func filasNonVacias(c CampoFormulario, v any) []map[string]any {
	var out []map[string]any
	for _, fila := range filasDatos(v) {
		for _, col := range c.Columnas {
			if col.Tipo == campoVerificacion {
				if valorBool(fila[col.ID]) {
					out = append(out, fila)
					break
				}
				continue
			}
			if textoValor(fila[col.ID]) != "" {
				out = append(out, fila)
				break
			}
		}
	}
	return out
}

// avisosDatos revisa os datos contra o esquema e devolve o que non cadra:
// obrigatorios sen valor, números e datas que non o son, escollas fóra das
// opcións. Non impide xerar.
func avisosDatos(f Formulario, d DatosFormulario, tx textosDocumento) []string {
	var avisos []string
	for _, c := range f.Campos {
		if c.Tipo == campoSeccion {
			continue
		}
		v := d.Valores[c.ID]
		if c.Obrigatorio && valorVacio(c, v) {
			avisos = append(avisos, fmt.Sprintf("o campo obrigatorio «%s» está sen cubrir", c.Etiqueta))
			continue
		}
		if c.Tipo == campoProblemas {
			avisos = append(avisos, avisosProblemas(c, v)...)
			continue
		}
		if c.Tipo == campoTaboa {
			for i, fila := range filasNonVacias(c, v) {
				for _, col := range c.Columnas {
					if _, ok := celaFormatada(col, fila[col.ID], tx); !ok {
						avisos = append(avisos, fmt.Sprintf("táboa «%s», fila %d: «%s» non ten un valor válido (%s)", c.Etiqueta, i+1, col.Etiqueta, textoValor(fila[col.ID])))
					}
				}
			}
			continue
		}
		if _, ok := celaFormatada(c, v, tx); !ok {
			avisos = append(avisos, fmt.Sprintf("o campo «%s» non ten un valor válido (%s)", c.Etiqueta, textoValor(v)))
		}
	}
	return avisos
}

// variablesFormulario son as variables automáticas do documento: as do curso
// ({{CENTRO}}, {{CURSO}}, {{DEPARTAMENTO}}, {{DATA_LONGA}}...) a partir do
// contexto, {{FORMULARIO}} e {{CAMPO_<ID>}} por cada campo simple con valor.
// Os campos gañan sobre ningunha outra variable automática: o seu nome leva o
// prefixo CAMPO_ para non pisalas.
func variablesFormulario(f Formulario, d DatosFormulario, nomeDoc string, agora time.Time, plantilla *PlantillaLatex) map[string]string {
	// Os «Si»/«Non» das verificacións son textos da plantilla; aínda non hai
	// variables para eles, así que se resolven sen ningunha.
	tx := novosTextosDoc(plantilla, map[string]string{})
	c := d.Contexto
	v := variablesCorpoCrono(PlanCrono{
		AcademicYear: c.Curso,
		Config: ConfigCrono{
			SchoolName: c.Centro, Location: c.Localidade, Department: c.Departamento,
			HeadOfDepartment: c.Responsable, SignatoryRole: c.Cargo, SchoolLogo: c.Logo,
		},
	}, nomeDoc, agora)
	v["FORMULARIO"] = f.tituloDoc()
	v["TITULO"] = f.tituloDoc()
	for _, campo := range f.Campos {
		switch campo.Tipo {
		case campoSeccion, campoTaboa, campoImaxe, campoTextoLongo, campoProblemas:
			continue
		}
		s, _ := celaFormatada(campo, d.Valores[campo.ID], tx)
		v["CAMPO_"+strings.ToUpper(campo.ID)] = s
	}
	return v
}

// reReferencia é unha referencia: texto entre corchetes dunha soa liña (≥3
// caracteres) que non é unha ligazón Markdown ([texto](url) nin [texto][id]).
var reReferencia = regexp.MustCompile(`[ \t]*\[[^\[\]\n]{3,}\]`)

// quitarReferencias quita as referencias dun texto e deixa o resto ordenado
// (sen espazos soltos ao final das liñas).
func quitarReferencias(s string) string {
	if !strings.Contains(s, "[") {
		return s
	}
	var b strings.Builder
	rest := s
	for {
		loc := reReferencia.FindStringIndex(rest)
		if loc == nil {
			b.WriteString(rest)
			break
		}
		fin := loc[1]
		if fin < len(rest) && (rest[fin] == '(' || rest[fin] == '[') { // ligazón: queda
			b.WriteString(rest[:fin])
			rest = rest[fin:]
			continue
		}
		b.WriteString(rest[:loc[0]])
		rest = rest[fin:]
	}
	linhas := strings.Split(b.String(), "\n")
	for i, l := range linhas {
		linhas[i] = strings.TrimRight(l, " \t")
	}
	return strings.Join(linhas, "\n")
}

func quitarReferenciasEn(v any) any {
	switch x := v.(type) {
	case string:
		return quitarReferencias(x)
	case []any:
		out := make([]any, len(x))
		for i, e := range x {
			out[i] = quitarReferenciasEn(e)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, e := range x {
			out[k] = quitarReferenciasEn(e)
		}
		return out
	}
	return v
}

// aplicarReferencias devolve os datos tal e como van ao documento: sen as
// referencias se o usuario as desactivou. Non muda os datos orixinais.
func (d DatosFormulario) aplicarReferencias() DatosFormulario {
	if !d.SenReferencias {
		return d
	}
	novo := d
	novo.Valores = make(map[string]any, len(d.Valores))
	for k, v := range d.Valores {
		novo.Valores[k] = quitarReferenciasEn(v)
	}
	return novo
}
