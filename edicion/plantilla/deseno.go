package plantilla

// Deseno é o DESEÑO do PDF coma datos: letras, tamaños, cores, espazos e o
// aspecto de cada elemento do documento. A aplicación xera o LaTeX a partir
// del (Lúa: deseno_latex.go), así o profesorado non ten que escribir LaTeX e
// a IA non pode romper a compilación: só cambia valores que se validan aquí.
//
// Todos os campos baleiros/cero significan «o de Lúa». O control fino non se
// perde: cada elemento pode levar o seu propio LaTeX (Elemento.Latex), que
// substitúe só ese elemento.

import (
	"edicion/latex"
	"math"
	"regexp"
	"sort"
	"strings"
)

// VersionDeseno é a versión do formato do deseño.
const VersionDeseno = 1

type Deseno struct {
	Version int `json:"version,omitempty"`
	// Letra é a combinación de letras (latex.Combinacions: lua, clasica...).
	Letra string `json:"letra,omitempty"`
	// Letras por papel ("" = a da combinación): id do catálogo ou nome dunha
	// fonte do sistema (esta só con xelatex/lualatex e non viaxa a outro equipo).
	LetraCorpo     string `json:"letraCorpo,omitempty"`
	LetraTitulos   string `json:"letraTitulos,omitempty"`
	LetraEtiquetas string `json:"letraEtiquetas,omitempty"`
	LetraMates     string `json:"letraMates,omitempty"` // "lm" = as de sempre
	// Tamano do corpo en pt (0 = 11). Todo escala con el.
	Tamano float64 `json:"tamano,omitempty"`
	// Escala dos títulos respecto do corpo (0 = 1).
	Escala float64 `json:"escala,omitempty"`
	// Interlinado (factor de \linespread; 0 = 1).
	Interlinado float64 `json:"interlinado,omitempty"`
	// Parrafo é o espazo entre parágrafos en em (0 = 0.55).
	Parrafo float64 `json:"parrafo,omitempty"`
	// Marxe en mm (0 = 20).
	Marxe float64 `json:"marxe,omitempty"`
	// Cores en hex sen #. Principal: títulos, filetes, rótulos ("" = índigo de
	// Lúa). Escura: rótulos e texto destacado ("" = derivada da principal).
	// Texto: o corpo ("" = negro).
	CorPrincipal string `json:"corPrincipal,omitempty"`
	CorEscura    string `json:"corEscura,omitempty"`
	CorTexto     string `json:"corTexto,omitempty"`
	// Etiquetas: aspecto das etiquetas do cronograma: "degradado" (por
	// defecto), "planas" ou "contorno".
	Etiquetas string `json:"etiquetas,omitempty"`
	// Barras gárdaas a interface (onde quedou cada barra). O xerador non as le:
	// as barras só cambian os outros campos.
	Barras Barras `json:"barras,omitempty"`
	// Interruptores do mundo real: aplícanse ao xerar, sen tocar os valores.
	BrancoNegro  bool `json:"brancoNegro,omitempty"`
	AforrarPapel bool `json:"aforrarPapel,omitempty"`
	LecturaFacil bool `json:"lecturaFacil,omitempty"`
	// Portada: modelo da primeira páxina: "" (a de cor de Lúa), "sobria",
	// "oficial" ou "sen" (sen portada). O LaTeX da plantilla (Portada) manda
	// sobre isto.
	Portada string `json:"portada,omitempty"`
	// LogoPortada: onde vai o logotipo na portada ("" = o do modelo,
	// "esquerda", "centro", "dereita").
	LogoPortada string `json:"logoPortada,omitempty"`
	// Cabeceira e Pe: texto de cada zona, con variables ({{CENTRO}}...),
	// @logo (o logotipo do curso) e @imaxe nome.png. Baleiras = as da maqueta.
	Cabeceira Zonas `json:"cabeceira,omitzero"`
	Pe        Zonas `json:"pe,omitzero"`
	// NumeroPaxina: onde vai o número de páxina se o deseño pon o pé: ""
	// (dereita), "esquerda", "centro" ou "non".
	NumeroPaxina string `json:"numeroPaxina,omitempty"`
	// Elementos: axustes de cada elemento (ElementosDeseno).
	Elementos map[string]Elemento `json:"elementos,omitempty"`
	// Fixados: campos que as barras, as variantes e a IA non deben tocar.
	Fixados []string `json:"fixados,omitempty"`
}

// Barras: -2..2. Formal (+) / informal (-); aireado (+) / compacto (-);
// colorido (+) / sobrio (-).
type Barras struct {
	Formal int `json:"formal,omitempty"`
	Aire   int `json:"aire,omitempty"`
	Cor    int `json:"cor,omitempty"`
	// CorBase é a cor principal que había antes de mover a barra da cor
	// (para volver a ela ao deixala no medio).
	CorBase string `json:"corBase,omitempty"`
	// Base son os valores que había antes de mover as barras formal e aire
	// (campo -> valor), para restauralos ao volver ao medio.
	Base map[string]any `json:"base,omitempty"`
}

// Zonas son as tres partes dunha cabeceira ou dun pé.
type Zonas struct {
	Esquerda string `json:"esquerda,omitempty"`
	Centro   string `json:"centro,omitempty"`
	Dereita  string `json:"dereita,omitempty"`
}

// Baleira di se ningunha zona ten texto.
func (z Zonas) Baleira() bool {
	return strings.TrimSpace(z.Esquerda+z.Centro+z.Dereita) == ""
}

func limparZona(s string) string {
	s = strings.ReplaceAll(strings.TrimSpace(s), "\r\n", "\n")
	if r := []rune(s); len(r) > 200 {
		s = string(r[:200])
	}
	return s
}

// Elemento é o axuste dun elemento do documento.
type Elemento struct {
	// Cor do elemento (hex; "" = a principal do deseño).
	Cor string `json:"cor,omitempty"`
	// Escala do tamaño do elemento (0 = 1).
	Escala float64 `json:"escala,omitempty"`
	// Maiusculas: o texto do elemento en maiúsculas.
	Maiusculas bool `json:"maiusculas,omitempty"`
	// SenFilete: quita a liña de cor que acompaña ao elemento.
	SenFilete bool `json:"senFilete,omitempty"`
	// Latex substitúe enteira a definición do elemento ("" = a xerada).
	Latex string `json:"latex,omitempty"`
}

// ElementoDeseno describe un elemento que se pode axustar.
type ElementoDeseno struct {
	ID string `json:"id"`
	// Usa: que controis ten sentido amosar.
	Cor, Escala, Maiusculas, Filete bool
}

// ElementosDeseno son os elementos do documento, na orde da interface.
var ElementosDeseno = []ElementoDeseno{
	{ID: "texto"},
	{ID: "titulos", Cor: true, Escala: true, Maiusculas: true, Filete: true},
	{ID: "etiquetas", Escala: true, Maiusculas: true},
	{ID: "meses", Escala: true},
	{ID: "trimestres", Cor: true, Escala: true, Maiusculas: true, Filete: true},
	{ID: "tarefas", Cor: true, Escala: true},
	{ID: "actas_cabeceira", Cor: true, Escala: true, Filete: true},
	{ID: "actas_apartados", Cor: true, Escala: true, Maiusculas: true, Filete: true},
	{ID: "actas_materias", Cor: true, Escala: true},
	{ID: "sinaturas", Escala: true},
}

// EsElemento di se o id é dun elemento coñecido.
func EsElemento(id string) bool {
	for _, e := range ElementosDeseno {
		if e.ID == id {
			return true
		}
	}
	return false
}

var hexRe = regexp.MustCompile(`^[0-9A-Fa-f]{6}$`)

// NormalizarCor deixa unha cor en hex de 6 díxitos en maiúsculas sen #, ou ""
// se non é válida.
func NormalizarCor(c string) string {
	c = strings.TrimPrefix(strings.TrimSpace(c), "#")
	if len(c) == 3 {
		c = string([]byte{c[0], c[0], c[1], c[1], c[2], c[2]})
	}
	if !hexRe.MatchString(c) {
		return ""
	}
	return strings.ToUpper(c)
}

func limitar(v, min, max float64) float64 {
	if v == 0 || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return math.Max(min, math.Min(max, v))
}

func limitarInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// letraValida acepta un id do catálogo ou un nome de fonte do sistema
// razoable (só letras, díxitos, espazos e puntuación inofensiva).
var letraSistemaRe = regexp.MustCompile(`^[\p{L}\p{N} .,'&()+_-]{1,80}$`)

func letraValida(s string, mates bool) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if mates {
		if s == "lm" {
			return s
		}
		if _, ok := latex.FamiliaMatesPorID(s); ok {
			return s
		}
		return ""
	}
	if _, ok := latex.FamiliaPorID(s); ok {
		return s
	}
	if letraSistemaRe.MatchString(s) {
		return s
	}
	return ""
}

// NormalizarDeseno valida e limpa un deseño: valores fóra de rango lévanse ao
// límite, cores e ids inválidos quedan baleiros, elementos descoñecidos
// bótanse. Un deseño nil segue sendo nil.
func NormalizarDeseno(d *Deseno) *Deseno {
	if d == nil {
		return nil
	}
	o := *d
	o.Version = VersionDeseno
	if _, ok := latex.CombinacionPorID(strings.TrimSpace(o.Letra)); !ok {
		o.Letra = ""
	} else {
		o.Letra = strings.TrimSpace(o.Letra)
	}
	o.LetraCorpo = letraValida(o.LetraCorpo, false)
	o.LetraTitulos = letraValida(o.LetraTitulos, false)
	o.LetraEtiquetas = letraValida(o.LetraEtiquetas, false)
	o.LetraMates = letraValida(o.LetraMates, true)
	o.Tamano = limitar(o.Tamano, 8, 20)
	o.Escala = limitar(o.Escala, 0.6, 1.8)
	o.Interlinado = limitar(o.Interlinado, 0.9, 2.2)
	o.Parrafo = limitar(o.Parrafo, 0.01, 3)
	o.Marxe = limitar(o.Marxe, 8, 40)
	o.CorPrincipal = NormalizarCor(o.CorPrincipal)
	o.CorEscura = NormalizarCor(o.CorEscura)
	o.CorTexto = NormalizarCor(o.CorTexto)
	switch o.Etiquetas {
	case "degradado", "planas", "contorno":
	default:
		o.Etiquetas = ""
	}
	switch o.Portada {
	case "sobria", "oficial", "sen":
	default:
		o.Portada = ""
	}
	switch o.LogoPortada {
	case "esquerda", "centro", "dereita":
	default:
		o.LogoPortada = ""
	}
	switch o.NumeroPaxina {
	case "esquerda", "centro", "non":
	default:
		o.NumeroPaxina = ""
	}
	o.Cabeceira = Zonas{limparZona(o.Cabeceira.Esquerda), limparZona(o.Cabeceira.Centro), limparZona(o.Cabeceira.Dereita)}
	o.Pe = Zonas{limparZona(o.Pe.Esquerda), limparZona(o.Pe.Centro), limparZona(o.Pe.Dereita)}
	o.Barras.Formal = limitarInt(o.Barras.Formal, -2, 2)
	o.Barras.Aire = limitarInt(o.Barras.Aire, -2, 2)
	o.Barras.Cor = limitarInt(o.Barras.Cor, -2, 2)
	o.Barras.CorBase = NormalizarCor(o.Barras.CorBase)
	if len(o.Barras.Base) > 0 {
		base := map[string]any{}
		for k, v := range o.Barras.Base {
			switch v.(type) {
			case string, float64:
				if camposFixables[k] {
					base[k] = v
				}
			}
		}
		o.Barras.Base = base
		if len(base) == 0 {
			o.Barras.Base = nil
		}
	}
	if len(o.Elementos) > 0 {
		els := map[string]Elemento{}
		for id, e := range o.Elementos {
			if !EsElemento(id) {
				continue
			}
			e.Cor = NormalizarCor(e.Cor)
			e.Escala = limitar(e.Escala, 0.5, 2.5)
			if e != (Elemento{}) {
				els[id] = e
			}
		}
		o.Elementos = els
		if len(els) == 0 {
			o.Elementos = nil
		}
	}
	if len(o.Fixados) > 0 {
		vistos := map[string]bool{}
		var fix []string
		for _, f := range o.Fixados {
			f = strings.TrimSpace(f)
			if f != "" && !vistos[f] && camposFixables[f] {
				vistos[f] = true
				fix = append(fix, f)
			}
		}
		sort.Strings(fix)
		o.Fixados = fix
	}
	return &o
}

// camposFixables son os campos que se poden fixar.
var camposFixables = map[string]bool{
	"letra": true, "tamano": true, "escala": true, "interlinado": true, "parrafo": true,
	"marxe": true, "corPrincipal": true, "corEscura": true, "corTexto": true, "etiquetas": true,
	"portada": true, "cabeceira": true, "pe": true,
}

// EsquemaDeseno é o JSON Schema do deseño: o que recibe a IA (calquera
// provedor) e o que documenta DESENO-SKILL.md.
func EsquemaDeseno() map[string]any {
	num := func(desc string, min, max float64) map[string]any {
		return map[string]any{"type": "number", "description": desc, "minimum": min, "maximum": max}
	}
	cor := func(desc string) map[string]any {
		return map[string]any{"type": "string", "description": desc + " Hex de 6 díxitos sen #.", "pattern": "^[0-9A-Fa-f]{6}$"}
	}
	var combs, fams, mates []string
	for _, c := range latex.Combinacions() {
		combs = append(combs, c.ID)
	}
	fams = latex.IDsFamilias()
	mates = append([]string{"lm"}, latex.IDsFamiliasMates()...)
	letra := func(desc string) map[string]any {
		return map[string]any{"type": "string", "description": desc + " Id do catálogo (" + strings.Join(fams, ", ") + ") ou o nome dunha fonte do sistema (só xelatex, non viaxa a outro equipo)."}
	}
	zonas := func(desc string) map[string]any {
		txt := func(z string) map[string]any {
			return map[string]any{"type": "string", "description": "Texto da zona " + z + ": variables ({{CENTRO}}, {{CURSO}}, {{DEPARTAMENTO}}, {{TITULO}}...), @logo (logotipo do curso) e @imaxe nome.png (imaxe da plantilla)."}
		}
		return map[string]any{"type": "object", "description": desc, "properties": map[string]any{
			"esquerda": txt("esquerda"), "centro": txt("central"), "dereita": txt("dereita"),
		}, "additionalProperties": false}
	}
	elementoIDs := make([]string, 0, len(ElementosDeseno))
	for _, e := range ElementosDeseno {
		elementoIDs = append(elementoIDs, e.ID)
	}
	elemento := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"cor":        cor("Cor do elemento."),
			"escala":     num("Tamaño do elemento respecto do normal.", 0.5, 2.5),
			"maiusculas": map[string]any{"type": "boolean", "description": "Texto en maiúsculas."},
			"senFilete":  map[string]any{"type": "boolean", "description": "Sen a liña de cor."},
			"latex":      map[string]any{"type": "string", "description": "LaTeX que substitúe a definición do elemento. Só se o usuario o pide expresamente."},
		},
		"additionalProperties": false,
	}
	elementosProps := map[string]any{}
	for _, id := range elementoIDs {
		elementosProps[id] = elemento
	}
	return map[string]any{
		"type":        "object",
		"description": "Deseño do PDF como datos. Campo baleiro ou 0 = o de Lúa.",
		"properties": map[string]any{
			"letra":          map[string]any{"type": "string", "enum": combs, "description": "Combinación de letras (corpo, títulos, etiquetas e fórmulas)."},
			"letraCorpo":     letra("Letra do texto."),
			"letraTitulos":   letra("Letra dos títulos."),
			"letraEtiquetas": letra("Letra das etiquetas e rótulos."),
			"letraMates":     map[string]any{"type": "string", "enum": mates, "description": "Letra das fórmulas (lm = a de sempre)."},
			"tamano":         num("Tamaño do texto en pt.", 8, 20),
			"escala":         num("Tamaño dos títulos respecto do texto.", 0.6, 1.8),
			"interlinado":    num("Espazo entre liñas (1 = sinxelo).", 0.9, 2.2),
			"parrafo":        num("Espazo entre parágrafos en em.", 0, 3),
			"marxe":          num("Marxe da páxina en mm.", 8, 40),
			"corPrincipal":   cor("Cor principal (títulos, liñas, rótulos)."),
			"corEscura":      cor("Cor escura para rótulos."),
			"corTexto":       cor("Cor do texto."),
			"etiquetas":      map[string]any{"type": "string", "enum": []string{"degradado", "planas", "contorno"}, "description": "Aspecto das etiquetas do cronograma."},
			"brancoNegro":    map[string]any{"type": "boolean", "description": "Preparar para imprimir en branco e negro."},
			"aforrarPapel":   map[string]any{"type": "boolean", "description": "Compactar para gastar menos follas."},
			"lecturaFacil":   map[string]any{"type": "boolean", "description": "Letra grande, máis espazo, sen xustificar."},
			"elementos":      map[string]any{"type": "object", "properties": elementosProps, "additionalProperties": false},
			"portada":        map[string]any{"type": "string", "enum": []string{"", "sobria", "oficial", "sen"}, "description": "Modelo da portada: \"\" (a de cor de Lúa), sobria, oficial ou sen (sen portada)."},
			"logoPortada":    map[string]any{"type": "string", "enum": []string{"", "esquerda", "centro", "dereita"}, "description": "Posición do logotipo na portada."},
			"cabeceira":      zonas("Cabeceira de todas as páxinas menos a portada."),
			"pe":             zonas("Pé de todas as páxinas menos a portada."),
			"numeroPaxina":   map[string]any{"type": "string", "enum": []string{"", "esquerda", "centro", "non"}, "description": "Onde vai o número de páxina (\"\" = dereita) cando o deseño pon o pé."},
			"fixados":        map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Campos que non se deben cambiar."},
		},
		"additionalProperties": false,
	}
}
