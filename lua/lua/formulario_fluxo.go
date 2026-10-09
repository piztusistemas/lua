package main

// Motor de formularios (capa 3: as TUBERÍAS). Un Fluxo é un grafo de nodos que
// leva os datos do formulario ata os bloques do documento:
//
//	ORIXES          campo (o valor dun campo do formulario), constante
//	TRANSFORMACIÓNS calcular, agregar, filtrar, ordenar, columnas, unir, comparar
//	DESTINOS        titulo, paragrafo, ficha, taboa, lista, imaxe, salto
//
// Cada nodo ten UNHA saída (as orixes e as transformacións) e as entradas que
// diga o catálogo (catalogoNodos): unha ligazón vai dunha saída a unha entrada
// e vive na entrada (NodoFluxo.Entradas: porto -> id do nodo de orixe). Os
// destinos colócanse no documento na orde de Fluxo.Documento.
//
// Sen Fluxo (nil) o formulario usa o cableado automático de
// formulario_bloques.go. fluxoAutomatico devolve ese mesmo cableado coma
// grafo, para poder editalo: un test garante que dá os mesmos bloques.
//
// TODO o avaliado ocorre aquí, en Go: a vista previa, as exportacións, o
// asistente e o editor gráfico (que só le o resultado en AvaliacionFluxo)
// ven sempre o mesmo.

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Tipos de nodo.
const (
	nodoCampo     = "campo"
	nodoConstante = "constante"
	nodoCalcular  = "calcular"
	nodoAgregar   = "agregar"
	nodoFiltrar   = "filtrar"
	nodoOrdenar   = "ordenar"
	nodoColumnas  = "columnas"
	nodoUnir      = "unir"
	nodoComparar  = "comparar"
	nodoTitulo    = "titulo"
	nodoParagrafo = "paragrafo"
	nodoFicha     = "ficha"
	nodoTaboa     = "taboa"
	nodoLista     = "lista"
	nodoImaxe     = "imaxe"
	nodoSalto     = "salto"
	nodoProblemas = "problemas"
)

// Tipos de dato que viaxan polas ligazóns.
const (
	dTexto  = "texto"
	dNumero = "numero"
	dData   = "data"
	dBool   = "bool"
	dTaboa  = "taboa"
	dImaxe  = "imaxe"
	// dProblemas: as preguntas dun campo de problemas (só van a un destino Problemas).
	dProblemas = "problemas"
)

const maxNodosFluxo = 300

// NodoFluxo é un nodo do grafo.
type NodoFluxo struct {
	ID   string  `json:"id"`
	Tipo string  `json:"tipo"`
	X    float64 `json:"x,omitempty"` // posición no lenzo (só a usa o editor)
	Y    float64 `json:"y,omitempty"`
	// Campo: só «campo», o id do campo do formulario.
	Campo string `json:"campo,omitempty"`
	// Config: os parámetros do nodo (ver ConfigNodo no catálogo).
	Config map[string]any `json:"config,omitempty"`
	// Entradas: porto -> id do nodo cuxa saída chega a ese porto.
	Entradas map[string]string `json:"entradas,omitempty"`
}

// Fluxo é o grafo completo dun formulario.
type Fluxo struct {
	Nodos []NodoFluxo `json:"nodos"`
	// Documento: os ids dos destinos, na orde en que saen no documento.
	Documento []string `json:"documento"`
}

// ---------- Catálogo ----------

// PortoNodo é unha entrada dun nodo.
type PortoNodo struct {
	Nome     string `json:"nome"`
	Tipo     string `json:"tipo"` // dTexto, dNumero, dData, dBool, dTaboa, dImaxe ou «calquera» (calquera valor simple)
	Opcional bool   `json:"opcional,omitempty"`
	Axuda    string `json:"axuda,omitempty"`
}

// ConfigNodo é un parámetro dun nodo.
type ConfigNodo struct {
	Clave    string   `json:"clave"`
	Etiqueta string   `json:"etiqueta"`
	Tipo     string   `json:"tipo"` // texto, textolongo, numero, escolla, columna, columnas, bool
	Opcions  []string `json:"opcions,omitempty"`
	// Nomes: como se chama cada opción na interface (se falta, o propio id).
	Nomes   map[string]string `json:"nomes,omitempty"`
	Defecto any               `json:"defecto,omitempty"`
	Axuda   string            `json:"axuda,omitempty"`
}

// TipoNodo describe un tipo de nodo: serve á validación, ao editor gráfico e
// ao asistente (que o le con formulario_axuda).
type TipoNodo struct {
	Tipo      string       `json:"tipo"`
	Nome      string       `json:"nome"`
	Categoria string       `json:"categoria"` // orixe, transformacion, destino
	Axuda     string       `json:"axuda"`
	Entradas  []PortoNodo  `json:"entradas,omitempty"`
	Dinamicas []string     `json:"dinamicas,omitempty"` // nomes de porto que se poden ir engadindo (a, b, c...), todos do tipo DinTipo
	DinTipo   string       `json:"dinTipo,omitempty"`
	Saida     string       `json:"saida,omitempty"` // tipo da saída; «» nos destinos; nos «campo»/«constante» depende do nodo
	Config    []ConfigNodo `json:"config,omitempty"`
	// PortosEtiqueta: cada porto dinámico leva un rótulo editable (config «etq_<porto>»), coma na ficha.
	PortosEtiqueta bool `json:"portosEtiqueta,omitempty"`
}

func nomesPortos(prefixo string, n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = prefixo + strconv.Itoa(i+1)
	}
	return out
}

var letrasPortos = []string{"a", "b", "c", "d", "e", "f", "g", "h"}

var (
	portoSe     = PortoNodo{Nome: "se", Tipo: dBool, Opcional: true, Axuda: "Se se conecta e vale «non», o bloque non sae no documento."}
	confTitulo  = ConfigNodo{Clave: "titulo", Etiqueta: "Título encima (opcional)", Tipo: "texto"}
	opsComparar = []string{"=", "!=", ">", ">=", "<", "<=", "contén", "baleiro", "non baleiro"}
)

// catalogoNodos é a definición de todos os tipos de nodo.
var catalogoNodos = []TipoNodo{
	{Tipo: nodoCampo, Nome: "Campo", Categoria: "orixe", Axuda: "O valor dun campo do formulario. Créanse sós, un por campo.",
		Config: []ConfigNodo{{Clave: "campo", Etiqueta: "Campo", Tipo: "texto"}}},
	{Tipo: nodoConstante, Nome: "Valor fixo", Categoria: "orixe", Axuda: "Un texto ou un número que escribes ti (p. ex. un peso 0.4 ou unha nota de corte 5).", Saida: "",
		Config: []ConfigNodo{
			{Clave: "tipo", Etiqueta: "Tipo", Tipo: "escolla", Opcions: []string{dTexto, dNumero}, Defecto: dNumero},
			{Clave: "valor", Etiqueta: "Valor", Tipo: "texto"}}},

	{Tipo: nodoCalcular, Nome: "Calcular", Categoria: "transformacion", Saida: dNumero,
		Axuda:     "Fai unha conta cos números que entran (a, b, c...). Operadores + - * / % ^ e comparacións; funcións min, max, abs, sqrt, round(x,n), suma, media, se(cond,a,b). Punto decimal: 0.4.",
		Dinamicas: letrasPortos, DinTipo: dNumero,
		Config: []ConfigNodo{
			{Clave: "formula", Etiqueta: "Fórmula", Tipo: "texto", Defecto: "a", Axuda: "Ex.: a*0.4 + b*0.6"},
			{Clave: "decimais", Etiqueta: "Decimais", Tipo: "numero", Defecto: 2}}},
	{Tipo: nodoAgregar, Nome: "Resumir táboa", Categoria: "transformacion", Saida: dNumero,
		Axuda:    "Saca un número dunha táboa: suma, media, mínimo, máximo ou cantas filas hai.",
		Entradas: []PortoNodo{{Nome: "taboa", Tipo: dTaboa}},
		Config: []ConfigNodo{
			{Clave: "operacion", Etiqueta: "Operación", Tipo: "escolla", Opcions: []string{"suma", "media", "min", "max", "contar"}, Defecto: "media"},
			{Clave: "columna", Etiqueta: "Columna", Tipo: "columna", Axuda: "Non fai falla para «contar»."},
			{Clave: "decimais", Etiqueta: "Decimais", Tipo: "numero", Defecto: 2}}},
	{Tipo: nodoFiltrar, Nome: "Filtrar filas", Categoria: "transformacion", Saida: dTaboa,
		Axuda:    "Deixa só as filas da táboa que cumpren unha condición.",
		Entradas: []PortoNodo{{Nome: "taboa", Tipo: dTaboa}},
		Config: []ConfigNodo{
			{Clave: "columna", Etiqueta: "Columna", Tipo: "columna"},
			{Clave: "operador", Etiqueta: "Condición", Tipo: "escolla", Opcions: opsComparar, Defecto: "="},
			{Clave: "valor", Etiqueta: "Valor", Tipo: "texto"}}},
	{Tipo: nodoOrdenar, Nome: "Ordenar filas", Categoria: "transformacion", Saida: dTaboa,
		Axuda:    "Ordena as filas da táboa por unha columna.",
		Entradas: []PortoNodo{{Nome: "taboa", Tipo: dTaboa}},
		Config: []ConfigNodo{
			{Clave: "columna", Etiqueta: "Columna", Tipo: "columna"},
			{Clave: "sentido", Etiqueta: "Sentido", Tipo: "escolla", Opcions: []string{"asc", "desc"}, Defecto: "asc"}}},
	{Tipo: nodoColumnas, Nome: "Escoller columnas", Categoria: "transformacion", Saida: dTaboa,
		Axuda:    "Deixa só algunhas columnas da táboa (e na orde que digas).",
		Entradas: []PortoNodo{{Nome: "taboa", Tipo: dTaboa}},
		Config:   []ConfigNodo{{Clave: "columnas", Etiqueta: "Columnas", Tipo: "columnas"}}},
	{Tipo: nodoUnir, Nome: "Unir texto", Categoria: "transformacion", Saida: dTexto,
		Axuda:     "Compón unha frase cos valores que entran: no modelo escribe {{a}}, {{b}}... onde vaia cada un.",
		Dinamicas: letrasPortos, DinTipo: "calquera",
		Config: []ConfigNodo{{Clave: "modelo", Etiqueta: "Modelo", Tipo: "textolongo", Defecto: "{{a}}", Axuda: "Ex.: Nota final: {{a}} ({{b}})"}}},
	{Tipo: nodoComparar, Nome: "Comparar", Categoria: "transformacion", Saida: dBool,
		Axuda:    "Dá si ou non segundo compare dous valores. Serve para ocultar un bloque (entrada «se» dos destinos).",
		Entradas: []PortoNodo{{Nome: "a", Tipo: "calquera"}, {Nome: "b", Tipo: "calquera", Opcional: true}},
		Config:   []ConfigNodo{{Clave: "operador", Etiqueta: "Condición", Tipo: "escolla", Opcions: opsComparar, Defecto: ">="}, {Clave: "valor", Etiqueta: "Valor fixo (se non hai «b»)", Tipo: "texto"}}},

	{Tipo: nodoTitulo, Nome: "Título", Categoria: "destino", Axuda: "Un título no documento. Escribe o texto fixo ou conecta un texto.",
		Entradas: []PortoNodo{{Nome: "texto", Tipo: dTexto, Opcional: true}, portoSe},
		Config: []ConfigNodo{{Clave: "texto", Etiqueta: "Texto fixo", Tipo: "texto"},
			{Clave: "nivel", Etiqueta: "Nivel", Tipo: "escolla", Opcions: []string{"2", "3"}, Defecto: "2", Axuda: "2 = sección, 3 = subtítulo"}}},
	{Tipo: nodoParagrafo, Nome: "Texto", Categoria: "destino", Axuda: "Un texto en Markdown. Non sae se está baleiro.",
		Entradas: []PortoNodo{{Nome: "texto", Tipo: dTexto}, portoSe}, Config: []ConfigNodo{confTitulo}},
	{Tipo: nodoFicha, Nome: "Ficha de datos", Categoria: "destino", Axuda: "Parellas «rótulo: valor» unha debaixo doutra. Os valores baleiros non saen.", PortosEtiqueta: true,
		Dinamicas: nomesPortos("v", 30), DinTipo: "calquera", Entradas: []PortoNodo{portoSe}},
	{Tipo: nodoTaboa, Nome: "Táboa", Categoria: "destino", Axuda: "Amosa unha táboa. Non sae se non ten filas.",
		Entradas: []PortoNodo{{Nome: "taboa", Tipo: dTaboa}, portoSe}, Config: []ConfigNodo{confTitulo}},
	{Tipo: nodoLista, Nome: "Lista", Categoria: "destino", Axuda: "Unha lista con viñetas cos valores dunha columna da táboa.",
		Entradas: []PortoNodo{{Nome: "taboa", Tipo: dTaboa}, portoSe},
		Config:   []ConfigNodo{{Clave: "columna", Etiqueta: "Columna", Tipo: "columna"}, confTitulo}},
	{Tipo: nodoImaxe, Nome: "Imaxe", Categoria: "destino", Axuda: "Amosa unha imaxe. Non sae se non hai.",
		Entradas: []PortoNodo{{Nome: "imaxe", Tipo: dImaxe}, portoSe}, Config: []ConfigNodo{confTitulo}},
	{Tipo: nodoProblemas, Nome: "Problemas", Categoria: "destino", Axuda: "As preguntas dun campo de problemas, numeradas, coas fórmulas calculadas na semente de cada copia e, se se pide, coas solucións.",
		Entradas: []PortoNodo{{Nome: "problemas", Tipo: dProblemas}, portoSe}},
	tipoNodoGrafico(),
	{Tipo: nodoSalto, Nome: "Salto de páxina", Categoria: "destino", Axuda: "Empeza unha páxina nova.", Entradas: []PortoNodo{portoSe}},
}

var catalogoPorTipo = func() map[string]TipoNodo {
	m := map[string]TipoNodo{}
	for _, t := range catalogoNodos {
		m[t.Tipo] = t
	}
	return m
}()

func (t TipoNodo) eDestino() bool { return t.Categoria == "destino" }

// tipoPorto devolve o tipo do porto `nome` do tipo de nodo (estático ou dinámico).
func (t TipoNodo) tipoPorto(nome string) (string, bool, bool) {
	for _, p := range t.Entradas {
		if p.Nome == nome {
			return p.Tipo, p.Opcional, true
		}
	}
	for _, d := range t.Dinamicas {
		if d == nome {
			return t.DinTipo, true, true
		}
	}
	return "", false, false
}

// aceptaTipo di se un porto de tipo `entrada` recibe unha saída de tipo `saida`.
func aceptaTipo(entrada, saida string) bool {
	switch entrada {
	case "calquera":
		return saida != dTaboa && saida != dImaxe && saida != dProblemas
	case dTexto:
		return saida == dTexto || saida == dNumero || saida == dData || saida == dBool
	}
	return entrada == saida
}

// ---------- Análise estática (tipos, columnas e erros) ----------

// InfoNodo é o que se sabe dun nodo sen mirar os datos: o tipo da súa saída,
// as columnas se é unha táboa e o erro que lle impide funcionar.
type InfoNodo struct {
	Tipo     string            `json:"tipo,omitempty"`
	Columnas []CampoFormulario `json:"columnas,omitempty"`
	Erro     string            `json:"erro,omitempty"`
}

type analise struct {
	f     Formulario
	nodos map[string]*NodoFluxo
	campo map[string]CampoFormulario
	info  map[string]*InfoNodo
	fase  map[string]int // 0 sen visitar, 1 en curso, 2 feito
}

func tipoDeCampo(c CampoFormulario) string {
	switch c.Tipo {
	case campoNumero:
		return dNumero
	case campoData:
		return dData
	case campoVerificacion:
		return dBool
	case campoImaxe:
		return dImaxe
	case campoTaboa:
		return dTaboa
	case campoProblemas:
		return dProblemas
	}
	return dTexto
}

func novaAnalise(f Formulario) *analise {
	a := &analise{f: f, nodos: map[string]*NodoFluxo{}, campo: map[string]CampoFormulario{}, info: map[string]*InfoNodo{}, fase: map[string]int{}}
	for _, c := range f.Campos {
		a.campo[c.ID] = c
	}
	if f.Fluxo != nil {
		for i := range f.Fluxo.Nodos {
			n := &f.Fluxo.Nodos[i]
			a.nodos[n.ID] = n
		}
	}
	return a
}

func cfgTexto(n *NodoFluxo, clave string) string {
	if n.Config == nil {
		return ""
	}
	return strings.TrimSpace(textoValor(n.Config[clave]))
}

func cfgNumero(n *NodoFluxo, clave string, defecto float64) float64 {
	if n.Config == nil {
		return defecto
	}
	if s, ok := valorNumeroFloat(n.Config[clave]); ok {
		return s
	}
	return defecto
}

// valorNumeroFloat le un número de calquera valor solto («3,5», 3.5...).
func valorNumeroFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case int:
		return float64(x), true
	case bool:
		return 0, false
	}
	s := textoValor(v)
	if s == "" {
		return 0, false
	}
	if strings.Contains(s, ",") {
		s = strings.ReplaceAll(strings.ReplaceAll(s, ".", ""), ",", ".")
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, false
	}
	return f, true
}

func columnaPorId(cols []CampoFormulario, id string) (CampoFormulario, bool) {
	for _, c := range cols {
		if c.ID == id {
			return c, true
		}
	}
	return CampoFormulario{}, false
}

// analizar calcula (con memoria) a InfoNodo de `id`.
func (a *analise) analizar(id string) *InfoNodo {
	switch a.fase[id] {
	case 2:
		return a.info[id]
	case 1:
		// Volvemos a un nodo que aínda estamos analizando: ciclo. Ten que ser
		// un obxecto novo (o do nodo en curso séguese enchendo máis arriba).
		return &InfoNodo{Erro: "hai un ciclo: o nodo depende de si mesmo"}
	}
	inf := &InfoNodo{}
	a.info[id] = inf
	n, ok := a.nodos[id]
	if !ok {
		inf.Erro = fmt.Sprintf("o nodo «%s» non existe", id)
		a.fase[id] = 2
		return inf
	}
	a.fase[id] = 1
	defer func() { a.fase[id] = 2 }()

	t, ok := catalogoPorTipo[n.Tipo]
	if !ok {
		inf.Erro = fmt.Sprintf("tipo de nodo descoñecido: «%s»", n.Tipo)
		return inf
	}
	// Entradas: existen, cadran de tipo e non teñen erro.
	ent := map[string]*InfoNodo{}
	for porto, orixe := range n.Entradas {
		tp, _, existe := t.tipoPorto(porto)
		if !existe {
			inf.Erro = fmt.Sprintf("o nodo %s non ten unha entrada «%s»", t.Nome, porto)
			return inf
		}
		io := a.analizar(orixe)
		if io.Erro != "" {
			inf.Erro = "unha entrada ten erro: " + io.Erro
			return inf
		}
		if !aceptaTipo(tp, io.Tipo) {
			inf.Erro = fmt.Sprintf("a entrada «%s» agarda %s e chega %s", porto, nomeTipoDato(tp), nomeTipoDato(io.Tipo))
			return inf
		}
		ent[porto] = io
	}
	for _, p := range t.Entradas {
		if _, hai := ent[p.Nome]; !hai && !p.Opcional {
			inf.Erro = fmt.Sprintf("falta conectar a entrada «%s»", p.Nome)
			return inf
		}
	}

	switch n.Tipo {
	case nodoCampo:
		c, ok := a.campo[n.Campo]
		if !ok {
			inf.Erro = fmt.Sprintf("o campo «%s» xa non existe no formulario", n.Campo)
			return inf
		}
		if c.Tipo == campoSeccion {
			inf.Erro = "unha sección non ten valor"
			return inf
		}
		inf.Tipo = tipoDeCampo(c)
		if c.Tipo == campoTaboa {
			inf.Columnas = c.Columnas
		}
	case nodoConstante:
		inf.Tipo = dNumero
		if cfgTexto(n, "tipo") == dTexto {
			inf.Tipo = dTexto
		}
		if inf.Tipo == dNumero && cfgTexto(n, "valor") != "" {
			if _, ok := valorNumeroFloat(n.Config["valor"]); !ok {
				inf.Erro = fmt.Sprintf("«%s» non é un número", cfgTexto(n, "valor"))
			}
		}
	case nodoCalcular:
		inf.Tipo = dNumero
		var nomes []string
		for p := range n.Entradas {
			nomes = append(nomes, p)
		}
		sort.Strings(nomes)
		if err := comprobarFormula(cfgTexto(n, "formula"), nomes); err != nil {
			inf.Erro = "fórmula: " + err.Error()
		}
	case nodoAgregar, nodoFiltrar, nodoOrdenar, nodoColumnas, nodoLista:
		cols := ent["taboa"].Columnas
		inf.Tipo = dTaboa
		if n.Tipo == nodoAgregar {
			inf.Tipo = dNumero
		}
		if n.Tipo == nodoLista {
			inf.Tipo = ""
		}
		inf.Columnas = cols
		switch n.Tipo {
		case nodoAgregar:
			inf.Columnas = nil
			op := cfgTexto(n, "operacion")
			if !contemTexto([]string{"suma", "media", "min", "max", "contar"}, op) {
				inf.Erro = fmt.Sprintf("operación descoñecida: «%s»", op)
			} else if op != "contar" {
				if c, ok := columnaPorId(cols, cfgTexto(n, "columna")); !ok {
					inf.Erro = "escolle unha columna da táboa"
				} else if c.Tipo != campoNumero && c.Tipo != campoTexto {
					inf.Erro = fmt.Sprintf("a columna «%s» non é numérica", c.Etiqueta)
				}
			}
		case nodoFiltrar:
			if _, ok := columnaPorId(cols, cfgTexto(n, "columna")); !ok {
				inf.Erro = "escolle unha columna da táboa"
			} else if !contemTexto(opsComparar, cfgTexto(n, "operador")) {
				inf.Erro = fmt.Sprintf("condición descoñecida: «%s»", cfgTexto(n, "operador"))
			}
		case nodoOrdenar:
			if _, ok := columnaPorId(cols, cfgTexto(n, "columna")); !ok {
				inf.Erro = "escolle unha columna da táboa"
			}
		case nodoColumnas:
			ids := listaConfig(n, "columnas")
			if len(ids) == 0 {
				inf.Erro = "escolle polo menos unha columna"
			}
			var novas []CampoFormulario
			for _, id := range ids {
				c, ok := columnaPorId(cols, id)
				if !ok {
					inf.Erro = fmt.Sprintf("a columna «%s» non existe na táboa", id)
					break
				}
				novas = append(novas, c)
			}
			inf.Columnas = novas
		case nodoLista:
			if _, ok := columnaPorId(cols, cfgTexto(n, "columna")); !ok {
				inf.Erro = "escolle a columna da lista"
			}
		}
	case nodoUnir:
		inf.Tipo = dTexto
	case nodoComparar:
		inf.Tipo = dBool
		if !contemTexto(opsComparar, cfgTexto(n, "operador")) {
			inf.Erro = fmt.Sprintf("condición descoñecida: «%s»", cfgTexto(n, "operador"))
		}
	case nodoTitulo:
		if cfgTexto(n, "texto") == "" && n.Entradas["texto"] == "" {
			inf.Erro = "escribe o texto do título ou conecta un texto"
		}
	case nodoFicha:
		if len(n.Entradas) == 0 || (len(n.Entradas) == 1 && n.Entradas["se"] != "") {
			inf.Erro = "conecta polo menos un valor"
		}
	case nodoGrafico:
		inf.Erro = analizarGrafico(n, ent)
	}
	return inf
}

func nomeTipoDato(t string) string {
	switch t {
	case dTexto:
		return "un texto"
	case dNumero:
		return "un número"
	case dData:
		return "unha data"
	case dBool:
		return "un si/non"
	case dTaboa:
		return "unha táboa"
	case dImaxe:
		return "unha imaxe"
	case dProblemas:
		return "uns problemas"
	case "calquera":
		return "un valor simple"
	}
	return "«" + t + "»"
}

func contemTexto(l []string, s string) bool {
	for _, x := range l {
		if x == s {
			return true
		}
	}
	return false
}

// listaConfig le un parámetro que é unha lista de textos (ou texto con comas).
func listaConfig(n *NodoFluxo, clave string) []string {
	if n.Config == nil {
		return nil
	}
	var out []string
	switch v := n.Config[clave].(type) {
	case []any:
		for _, x := range v {
			if s := strings.TrimSpace(textoValor(x)); s != "" {
				out = append(out, s)
			}
		}
	case []string:
		for _, s := range v {
			if s = strings.TrimSpace(s); s != "" {
				out = append(out, s)
			}
		}
	case string:
		for _, s := range strings.Split(v, ",") {
			if s = strings.TrimSpace(s); s != "" {
				out = append(out, s)
			}
		}
	}
	return out
}

// ---------- Validación ----------

// validarFluxo devolve o primeiro problema ESTRUTURAL do grafo (ids, tipos,
// ligazóns, documento). Os erros de configuración dun nodo (columna sen
// escoller, fórmula mal escrita...) non impiden gardar: o nodo queda en
// vermello e o seu bloque non sae, con aviso.
func validarFluxo(f Formulario) error {
	fl := f.Fluxo
	if fl == nil {
		return nil
	}
	if len(fl.Nodos) > maxNodosFluxo {
		return fmt.Errorf("a tubería ten máis de %d nodos", maxNodosFluxo)
	}
	vistos := map[string]bool{}
	for i, n := range fl.Nodos {
		if !idFormularioRe.MatchString(n.ID) {
			return fmt.Errorf("nodo %d: o identificador %q non vale (minúsculas, díxitos e _)", i+1, n.ID)
		}
		if vistos[n.ID] {
			return fmt.Errorf("nodo %d: o identificador %q está repetido", i+1, n.ID)
		}
		vistos[n.ID] = true
		t, ok := catalogoPorTipo[n.Tipo]
		if !ok {
			return fmt.Errorf("nodo «%s»: tipo descoñecido «%s» (hai: %s)", n.ID, n.Tipo, nomesTiposNodo())
		}
		if n.Tipo == nodoCampo && n.Campo == "" {
			return fmt.Errorf("nodo «%s»: un nodo campo ten que dicir de que campo (campo)", n.ID)
		}
		for porto, orixe := range n.Entradas {
			if _, _, ok := t.tipoPorto(porto); !ok {
				return fmt.Errorf("nodo «%s» (%s): non ten unha entrada «%s»", n.ID, n.Tipo, porto)
			}
			if orixe == "" {
				return fmt.Errorf("nodo «%s»: a entrada «%s» está baleira (quítaa ou conéctaa)", n.ID, porto)
			}
		}
	}
	for _, n := range fl.Nodos {
		for porto, orixe := range n.Entradas {
			if !vistos[orixe] {
				return fmt.Errorf("nodo «%s»: a entrada «%s» ven de «%s», que non existe", n.ID, porto, orixe)
			}
			if o := catalogoPorTipo[nodoPorId(fl, orixe).Tipo]; o.eDestino() {
				return fmt.Errorf("nodo «%s»: unha entrada non pode vir dun destino («%s»)", n.ID, orixe)
			}
		}
	}
	// Ciclos.
	a := novaAnalise(f)
	for _, n := range fl.Nodos {
		if i := a.analizar(n.ID); strings.Contains(i.Erro, "ciclo") {
			return fmt.Errorf("nodo «%s»: %s", n.ID, i.Erro)
		}
	}
	docs := map[string]bool{}
	for _, id := range fl.Documento {
		n := nodoPorId(fl, id)
		if n.ID == "" {
			return fmt.Errorf("o documento leva «%s», que non existe", id)
		}
		if !catalogoPorTipo[n.Tipo].eDestino() {
			return fmt.Errorf("o documento leva «%s» (%s), que non é un destino", id, n.Tipo)
		}
		if docs[id] {
			return fmt.Errorf("o documento leva «%s» dúas veces", id)
		}
		docs[id] = true
	}
	return nil
}

func nomesTiposNodo() string {
	var out []string
	for _, t := range catalogoNodos {
		out = append(out, t.Tipo)
	}
	return strings.Join(out, ", ")
}

func nodoPorId(fl *Fluxo, id string) NodoFluxo {
	for _, n := range fl.Nodos {
		if n.ID == id {
			return n
		}
	}
	return NodoFluxo{}
}

// normalizarFluxo limpa o que chega do frontend ou da IA: ids en minúsculas,
// entradas baleiras fóra e destinos que faltan no documento ao final.
func normalizarFluxo(f Formulario) *Fluxo {
	if f.Fluxo == nil {
		return nil
	}
	out := &Fluxo{}
	for _, n := range f.Fluxo.Nodos {
		n.ID = strings.TrimSpace(n.ID)
		n.Tipo = strings.TrimSpace(n.Tipo)
		n.Campo = strings.TrimSpace(n.Campo)
		ent := map[string]string{}
		for p, o := range n.Entradas {
			if o = strings.TrimSpace(o); o != "" {
				ent[strings.TrimSpace(p)] = o
			}
		}
		n.Entradas = nil
		if len(ent) > 0 {
			n.Entradas = ent
		}
		out.Nodos = append(out.Nodos, n)
	}
	for _, id := range f.Fluxo.Documento {
		out.Documento = append(out.Documento, strings.TrimSpace(id))
	}
	// Un destino que non está no documento non sairía nunca: engádese ao final.
	en := map[string]bool{}
	for _, id := range out.Documento {
		en[id] = true
	}
	for _, n := range out.Nodos {
		if t, ok := catalogoPorTipo[n.Tipo]; ok && t.eDestino() && !en[n.ID] {
			out.Documento = append(out.Documento, n.ID)
		}
	}
	return out
}

// ---------- Cableado automático como grafo ----------

func idNodoCampo(id string) string { return "c_" + id }

// fluxoAutomatico devolve o cableado automático do formulario coma grafo
// editable: un nodo campo por campo e os destinos que xeraría bloquesFormulario.
func fluxoAutomatico(f Formulario) *Fluxo {
	fl := &Fluxo{}
	usados := map[string]bool{}
	novoID := func(base string) string { return idLibre(base, usados) }
	engadirCampo := func(c CampoFormulario) string {
		id := idNodoCampo(c.ID)
		usados[id] = true
		fl.Nodos = append(fl.Nodos, NodoFluxo{ID: id, Tipo: nodoCampo, Campo: c.ID})
		return id
	}
	destino := func(tipo string, cfg map[string]any, ent map[string]string) {
		id := novoID("d_" + tipo)
		fl.Nodos = append(fl.Nodos, NodoFluxo{ID: id, Tipo: tipo, Config: cfg, Entradas: ent})
		fl.Documento = append(fl.Documento, id)
	}
	var ficha map[string]string
	var fichaCfg map[string]any
	nFicha := 0
	pecharFicha := func() {
		if len(ficha) > 0 {
			destino(nodoFicha, fichaCfg, ficha)
		}
		ficha, fichaCfg, nFicha = nil, nil, 0
	}
	for _, c := range f.Campos {
		if c.Agochado {
			continue
		}
		switch c.Tipo {
		case campoSeccion:
			pecharFicha()
			destino(nodoTitulo, map[string]any{"texto": c.Etiqueta, "nivel": "2"}, nil)
		case campoTextoLongo:
			pecharFicha()
			destino(nodoParagrafo, map[string]any{"titulo": c.Etiqueta}, map[string]string{"texto": engadirCampo(c)})
		case campoImaxe:
			pecharFicha()
			destino(nodoImaxe, map[string]any{"titulo": c.Etiqueta}, map[string]string{"imaxe": engadirCampo(c)})
		case campoTaboa:
			pecharFicha()
			destino(nodoTaboa, map[string]any{"titulo": c.Etiqueta}, map[string]string{"taboa": engadirCampo(c)})
		case campoProblemas:
			pecharFicha()
			destino(nodoProblemas, nil, map[string]string{"problemas": engadirCampo(c)})
		default:
			if ficha == nil {
				ficha, fichaCfg = map[string]string{}, map[string]any{}
			}
			nFicha++
			porto := "v" + strconv.Itoa(nFicha)
			ficha[porto] = engadirCampo(c)
			fichaCfg["etq_"+porto] = c.Etiqueta
			if nFicha == 30 {
				pecharFicha()
			}
		}
	}
	pecharFicha()
	return fl
}

// sincronizarFluxo axusta un fluxo existente aos campos do formulario: os
// campos novos entran (co seu destino automático, ao final do documento), os
// que xa non existen saen (e as ligazóns que viñan deles). Sen fluxo devolve o
// automático.
func sincronizarFluxo(f Formulario) *Fluxo {
	if f.Fluxo == nil || len(f.Fluxo.Nodos) == 0 {
		return fluxoAutomatico(f)
	}
	fl := &Fluxo{Documento: append([]string(nil), f.Fluxo.Documento...)}
	campos := map[string]bool{}
	for _, c := range f.Campos {
		campos[c.ID] = true
	}
	quitados := map[string]bool{}
	tenNodo := map[string]bool{}
	for _, n := range f.Fluxo.Nodos {
		if n.Tipo == nodoCampo {
			if !campos[n.Campo] {
				quitados[n.ID] = true
				continue
			}
			tenNodo[n.Campo] = true
		}
		fl.Nodos = append(fl.Nodos, n)
	}
	for i := range fl.Nodos {
		var ent map[string]string
		for p, o := range fl.Nodos[i].Entradas {
			if !quitados[o] {
				if ent == nil {
					ent = map[string]string{}
				}
				ent[p] = o
			}
		}
		fl.Nodos[i].Entradas = ent
	}
	// Campos novos: o seu cableado automático, sen repetir os nodos que xa hai.
	falta := f
	falta.Campos = nil
	for _, c := range f.Campos {
		if !tenNodo[c.ID] && c.Tipo != campoSeccion && !c.Agochado {
			falta.Campos = append(falta.Campos, c)
		}
	}
	if len(falta.Campos) > 0 {
		usados := map[string]bool{}
		for _, n := range fl.Nodos {
			usados[n.ID] = true
		}
		extra := fluxoAutomatico(falta)
		remap := map[string]string{}
		for _, n := range extra.Nodos {
			novo := idLibre(n.ID, usados)
			remap[n.ID] = novo
			n.ID = novo
			fl.Nodos = append(fl.Nodos, n)
		}
		for i := len(fl.Nodos) - len(extra.Nodos); i < len(fl.Nodos); i++ {
			for p, o := range fl.Nodos[i].Entradas {
				fl.Nodos[i].Entradas[p] = remap[o]
			}
		}
		for _, id := range extra.Documento {
			fl.Documento = append(fl.Documento, remap[id])
		}
	}
	// O documento non pode levar destinos que se foron.
	var doc []string
	tenID := map[string]bool{}
	for _, n := range fl.Nodos {
		tenID[n.ID] = true
	}
	for _, id := range fl.Documento {
		if tenID[id] {
			doc = append(doc, id)
		}
	}
	fl.Documento = doc
	return fl
}

// ---------- Avaliación ----------

// valorFluxo é o que sae dun nodo.
type valorFluxo struct {
	Tipo  string
	Vacio bool
	Texto string
	Num   float64
	Bool  bool
	Cols  []CampoFormulario
	Filas []map[string]any
	Imaxe string
	Erro  string
	// Problemas: o valor dun campo de problemas e o índice da súa primeira pregunta.
	Problemas     []ProblemaFormulario
	Desprazamento int
}

// AvaliacionFluxo é o que ve o editor gráfico e o asistente: o estado de cada
// nodo, os bloques resultantes e os avisos.
type AvaliacionFluxo struct {
	Nodos   map[string]NodoAvaliado `json:"nodos"`
	Bloques []BloqueVista           `json:"bloques"`
	Avisos  []string                `json:"avisos"`
	Erro    string                  `json:"erro,omitempty"`
}

// NodoAvaliado é o estado dun nodo despois de avaliar.
type NodoAvaliado struct {
	Tipo     string            `json:"tipo,omitempty"`
	Columnas []CampoFormulario `json:"columnas,omitempty"`
	Erro     string            `json:"erro,omitempty"`
	Resumo   string            `json:"resumo,omitempty"` // o valor, en curto, para amosalo no nodo
	Vacio    bool              `json:"vacio,omitempty"`
}

// BloqueVista é un bloque do documento en forma sinxela (para a vista previa
// e para o asistente).
type BloqueVista struct {
	Tipo      string      `json:"tipo"`
	Nivel     int         `json:"nivel,omitempty"`
	Texto     string      `json:"texto,omitempty"`
	Pares     [][2]string `json:"pares,omitempty"`
	Cabeceira []string    `json:"cabeceira,omitempty"`
	Filas     [][]string  `json:"filas,omitempty"`
	Elementos []string    `json:"elementos,omitempty"`
	Nodo      string      `json:"nodo,omitempty"` // o destino que o xerou
}

type avaliador struct {
	avisos []string // problemas dos datos que impiden un bloque (ver destinoGrafico)
	a      *analise
	d      DatosFormulario
	tx     textosDocumento
	cache  map[string]valorFluxo
}

func textoResumo(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if len(r) > max {
		return string(r[:max-1]) + "…"
	}
	return s
}

// formatarNumero: coma decimal, sen ceros de máis, con `decimais` como máximo.
func formatarNumero(v float64, decimais int) string {
	if decimais < 0 {
		decimais = 0
	}
	if decimais > 8 {
		decimais = 8
	}
	s := strconv.FormatFloat(v, 'f', decimais, 64)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	}
	if s == "-0" {
		s = "0"
	}
	return strings.ReplaceAll(s, ".", ",")
}

// valor devolve o valor do nodo `id` (con memoria). Os erros e os baleiros
// propáganse: un nodo cunha entrada baleira dá baleiro.
func (e *avaliador) valor(id string) valorFluxo {
	if v, ok := e.cache[id]; ok {
		return v
	}
	v := e.calcular(id)
	e.cache[id] = v
	return v
}

func (e *avaliador) calcular(id string) valorFluxo {
	n := e.a.nodos[id]
	inf := e.a.analizar(id)
	if inf.Erro != "" {
		return valorFluxo{Erro: inf.Erro}
	}
	entrada := func(p string) (valorFluxo, bool) {
		o, ok := n.Entradas[p]
		if !ok {
			return valorFluxo{}, false
		}
		return e.valor(o), true
	}
	switch n.Tipo {
	case nodoCampo:
		c := e.a.campo[n.Campo]
		v := e.d.Valores[c.ID]
		switch c.Tipo {
		case campoTaboa:
			return valorFluxo{Tipo: dTaboa, Cols: c.Columnas, Filas: filasNonVacias(c, v), Vacio: valorVacio(c, v)}
		case campoImaxe:
			s := textoValor(v)
			return valorFluxo{Tipo: dImaxe, Imaxe: s, Vacio: s == ""}
		case campoProblemas:
			ps := problemasDe(v)
			n := len(problemasNonVacios(v))
			return valorFluxo{Tipo: dProblemas, Problemas: ps, Desprazamento: desprazamentosProblemas(e.a.f, e.d)[c.ID], Vacio: n == 0, Texto: fmt.Sprintf("%d preguntas", n)}
		case campoNumero:
			s, _ := valorNumero(v)
			f, ok := valorNumeroFloat(v)
			return valorFluxo{Tipo: dNumero, Texto: s, Num: f, Vacio: !ok && textoValor(v) == "", Erro: erroSe(!ok && textoValor(v) != "", "«"+textoValor(v)+"» non é un número")}
		case campoData:
			s, ok := valorData(v)
			return valorFluxo{Tipo: dData, Texto: s, Vacio: s == "", Erro: erroSe(!ok, "«"+s+"» non é unha data")}
		case campoVerificacion:
			b := valorBool(v)
			return valorFluxo{Tipo: dBool, Bool: b, Texto: e.tx.T(mapaSiNon(b))}
		default:
			s, _ := celaFormatada(c, v, e.tx)
			return valorFluxo{Tipo: dTexto, Texto: s, Vacio: s == ""}
		}
	case nodoConstante:
		s := cfgTexto(n, "valor")
		if inf.Tipo == dNumero {
			f, ok := valorNumeroFloat(n.Config["valor"])
			return valorFluxo{Tipo: dNumero, Num: f, Texto: formatarNumero(f, 8), Vacio: !ok}
		}
		return valorFluxo{Tipo: dTexto, Texto: s, Vacio: s == ""}
	case nodoCalcular:
		vars := map[string]float64{}
		for p := range n.Entradas {
			v, _ := entrada(p)
			if v.Erro != "" {
				return valorFluxo{Erro: v.Erro}
			}
			if v.Vacio {
				return valorFluxo{Tipo: dNumero, Vacio: true}
			}
			vars[p] = v.Num
		}
		r, err := avaliarFormula(cfgTexto(n, "formula"), vars)
		if err != nil {
			return valorFluxo{Erro: err.Error()}
		}
		return numeroSaida(r, int(cfgNumero(n, "decimais", 2)))
	case nodoAgregar:
		v, _ := entrada("taboa")
		if v.Erro != "" {
			return valorFluxo{Erro: v.Erro}
		}
		op := cfgTexto(n, "operacion")
		if op == "contar" {
			return numeroSaida(float64(len(v.Filas)), 0)
		}
		var xs []float64
		for _, f := range v.Filas {
			if x, ok := valorNumeroFloat(f[cfgTexto(n, "columna")]); ok {
				xs = append(xs, x)
			}
		}
		if len(xs) == 0 {
			if op == "suma" {
				return numeroSaida(0, 0)
			}
			return valorFluxo{Tipo: dNumero, Vacio: true}
		}
		r := xs[0]
		s := 0.0
		for _, x := range xs {
			s += x
			if op == "min" && x < r {
				r = x
			}
			if op == "max" && x > r {
				r = x
			}
		}
		switch op {
		case "suma":
			r = s
		case "media":
			r = s / float64(len(xs))
		}
		return numeroSaida(r, int(cfgNumero(n, "decimais", 2)))
	case nodoFiltrar, nodoOrdenar, nodoColumnas:
		v, _ := entrada("taboa")
		if v.Erro != "" {
			return valorFluxo{Erro: v.Erro}
		}
		out := valorFluxo{Tipo: dTaboa, Cols: inf.Columnas}
		switch n.Tipo {
		case nodoFiltrar:
			col, _ := columnaPorId(v.Cols, cfgTexto(n, "columna"))
			for _, f := range v.Filas {
				if cumpreCondicion(col, f[col.ID], cfgTexto(n, "operador"), cfgTexto(n, "valor")) {
					out.Filas = append(out.Filas, f)
				}
			}
		case nodoOrdenar:
			col, _ := columnaPorId(v.Cols, cfgTexto(n, "columna"))
			out.Filas = append([]map[string]any(nil), v.Filas...)
			desc := cfgTexto(n, "sentido") == "desc"
			sort.SliceStable(out.Filas, func(i, j int) bool {
				c := compararCela(col, out.Filas[i][col.ID], out.Filas[j][col.ID])
				if desc {
					return c > 0
				}
				return c < 0
			})
		default:
			out.Filas = v.Filas
		}
		out.Vacio = len(out.Filas) == 0
		return out
	case nodoUnir:
		modelo := cfgTexto(n, "modelo")
		if modelo == "" {
			modelo = "{{a}}"
		}
		baleiro := true
		for p := range n.Entradas {
			v, _ := entrada(p)
			if v.Erro != "" {
				return valorFluxo{Erro: v.Erro}
			}
			if !v.Vacio {
				baleiro = false
			}
			modelo = strings.ReplaceAll(modelo, "{{"+p+"}}", textoDeValor(v))
		}
		// Un marcador sen entrada queda baleiro.
		for _, p := range letrasPortos {
			modelo = strings.ReplaceAll(modelo, "{{"+p+"}}", "")
		}
		return valorFluxo{Tipo: dTexto, Texto: strings.TrimSpace(modelo), Vacio: baleiro}
	case nodoComparar:
		a, _ := entrada("a")
		if a.Erro != "" {
			return valorFluxo{Erro: a.Erro}
		}
		op := cfgTexto(n, "operador")
		b, hai := entrada("b")
		if hai && b.Erro != "" {
			return valorFluxo{Erro: b.Erro}
		}
		col := CampoFormulario{Tipo: campoTexto}
		if a.Tipo == dNumero {
			col.Tipo = campoNumero
		} else if a.Tipo == dData {
			col.Tipo = campoData
		}
		ref := cfgTexto(n, "valor")
		if hai {
			ref = textoDeValor(b)
		}
		ok := cumpreCondicion(col, textoDeValor(a), op, ref)
		return valorFluxo{Tipo: dBool, Bool: ok, Texto: e.tx.T(mapaSiNon(ok))}
	}
	return valorFluxo{Erro: "nodo sen avaliación: " + n.Tipo}
}

func erroSe(cond bool, msg string) string {
	if cond {
		return msg
	}
	return ""
}

func mapaSiNon(b bool) string {
	if b {
		return "FORM_SI"
	}
	return "FORM_NON"
}

func numeroSaida(r float64, decimais int) valorFluxo {
	if decimais < 0 || decimais > 8 {
		decimais = 2
	}
	k := math.Pow(10, float64(decimais))
	r = math.Round(r*k) / k
	return valorFluxo{Tipo: dNumero, Num: r, Texto: formatarNumero(r, decimais)}
}

// textoDeValor é o valor dun nodo simple coma texto para imprimir.
func textoDeValor(v valorFluxo) string {
	if v.Vacio {
		return ""
	}
	return v.Texto
}

// chaveCela devolve o valor dunha cela nunha forma comparable: número, data
// (AAAAMMDD) ou texto en minúsculas.
func chaveCela(col CampoFormulario, v any) (float64, bool) {
	switch col.Tipo {
	case campoNumero:
		if f, ok := valorNumeroFloat(v); ok {
			return f, true
		}
	case campoData:
		if d, ok := valorData(v); ok && d != "" {
			if t, err := time.Parse("02/01/2006", d); err == nil {
				return float64(t.Unix()), true
			}
		}
	case campoTexto:
		// Un texto que é un número compárase coma número (p. ex. notas importadas de CSV).
		if f, ok := valorNumeroFloat(v); ok {
			return f, true
		}
	}
	return 0, false
}

func compararCela(col CampoFormulario, a, b any) int {
	fa, na := chaveCela(col, a)
	fb, nb := chaveCela(col, b)
	if na && nb {
		switch {
		case fa < fb:
			return -1
		case fa > fb:
			return 1
		}
		return 0
	}
	return strings.Compare(strings.ToLower(textoValor(a)), strings.ToLower(textoValor(b)))
}

// cumpreCondicion aplica «celda OP valor». Con números e datas compara como
// tales; co resto, sen distinguir maiúsculas.
func cumpreCondicion(col CampoFormulario, cela any, op, ref string) bool {
	s := strings.TrimSpace(textoValor(cela))
	switch op {
	case "baleiro":
		return s == ""
	case "non baleiro":
		return s != ""
	case "contén":
		return strings.Contains(strings.ToLower(s), strings.ToLower(ref))
	}
	if col.Tipo == campoVerificacion {
		eq := valorBool(cela) == valorBool(ref)
		if op == "!=" {
			return !eq
		}
		return eq
	}
	c := compararCela(col, cela, ref)
	switch op {
	case "=":
		return c == 0
	case "!=":
		return c != 0
	case ">":
		return c > 0
	case ">=":
		return c >= 0
	case "<":
		return c < 0
	case "<=":
		return c <= 0
	}
	return false
}

// resumoValor é o valor dun nodo en curto (para amosalo no editor).
func resumoValor(v valorFluxo) string {
	if v.Erro != "" {
		return ""
	}
	if v.Vacio {
		return ""
	}
	switch v.Tipo {
	case dTaboa:
		if len(v.Filas) == 1 {
			return "1 fila"
		}
		return fmt.Sprintf("%d filas", len(v.Filas))
	case dImaxe:
		return "imaxe"
	case dBool:
		return v.Texto
	}
	return textoResumo(v.Texto, 40)
}

// avaliarFluxoDoc avalía o grafo e devolve os bloques do documento e o estado
// de cada nodo.
func avaliarFluxoDoc(f Formulario, d DatosFormulario, tx textosDocumento) ([]bloqueDoc, AvaliacionFluxo) {
	res := AvaliacionFluxo{Nodos: map[string]NodoAvaliado{}}
	if f.Fluxo == nil {
		res.Erro = "o formulario non ten tubería"
		return nil, res
	}
	if err := validarFluxo(f); err != nil {
		res.Erro = err.Error()
		return nil, res
	}
	a := novaAnalise(f)
	ev := &avaliador{a: a, d: d, tx: tx, cache: map[string]valorFluxo{}}
	for _, n := range f.Fluxo.Nodos {
		inf := a.analizar(n.ID)
		if catalogoPorTipo[n.Tipo].eDestino() {
			res.Nodos[n.ID] = NodoAvaliado{Erro: inf.Erro}
			continue
		}
		v := ev.valor(n.ID)
		erro := inf.Erro
		if erro == "" {
			erro = v.Erro
		}
		res.Nodos[n.ID] = NodoAvaliado{Tipo: inf.Tipo, Columnas: inf.Columnas, Erro: erro, Resumo: resumoValor(v), Vacio: v.Vacio && erro == ""}
	}
	var bloques []bloqueDoc
	for _, id := range f.Fluxo.Documento {
		n := a.nodos[id]
		if n == nil {
			continue
		}
		etq := rotuloNodo(*n)
		if inf := a.analizar(id); inf.Erro != "" {
			res.Avisos = append(res.Avisos, fmt.Sprintf("o bloque «%s» non sae: %s", etq, inf.Erro))
			continue
		}
		if err := ev.erroEntradas(n); err != "" {
			res.Avisos = append(res.Avisos, fmt.Sprintf("o bloque «%s» non sae: %s", etq, err))
			continue
		}
		if s, ok := n.Entradas["se"]; ok {
			if v := ev.valor(s); v.Erro == "" && v.Tipo == dBool && !v.Bool {
				continue
			}
		}
		novos := ev.destino(n)
		bloques = append(bloques, novos...)
		res.Avisos = engadirSenRepetir(res.Avisos, ev.avisos)
		ev.avisos = nil
		nd := res.Nodos[id]
		nd.Resumo = resumoBloques(novos)
		nd.Vacio = len(novos) == 0
		res.Nodos[id] = nd
	}
	if len(bloques) == 0 {
		if t := strings.TrimSpace(tx.T("FORM_SEN_CONTIDO")); t != "" {
			bloques = append(bloques, bloqueDoc{Tipo: bloqueMarkdown, Texto: t})
		}
	}
	for _, b := range bloques {
		res.Bloques = append(res.Bloques, vistaBloque(b))
	}
	return bloques, res
}

// erroEntradas devolve o erro (en tempo de avaliación) dunha entrada do destino.
func (e *avaliador) erroEntradas(n *NodoFluxo) string {
	for p, o := range n.Entradas {
		if v := e.valor(o); v.Erro != "" {
			return fmt.Sprintf("a entrada «%s» ten un erro: %s", p, v.Erro)
		}
	}
	return ""
}

func rotuloNodo(n NodoFluxo) string {
	if t := cfgTexto(&n, "titulo"); t != "" {
		return t
	}
	if t := cfgTexto(&n, "texto"); t != "" {
		return t
	}
	return catalogoPorTipo[n.Tipo].Nome
}

func resumoBloques(bs []bloqueDoc) string {
	if len(bs) == 0 {
		return ""
	}
	b := bs[len(bs)-1]
	switch b.Tipo {
	case bloqueFicha:
		return fmt.Sprintf("%d datos", len(b.Pares))
	case bloqueTaboa:
		return fmt.Sprintf("%d filas", len(b.Filas))
	case bloqueLista:
		return fmt.Sprintf("%d elementos", len(b.Elementos))
	case bloqueImaxe:
		return "imaxe"
	case bloqueGrafico:
		return fmt.Sprintf("%s · %d datos", strings.ToLower(b.Texto), len(b.Pares))
	case bloqueSalto:
		return "páxina nova"
	}
	return textoResumo(b.Texto, 40)
}

func vistaBloque(b bloqueDoc) BloqueVista {
	v := BloqueVista{Tipo: b.Tipo, Nivel: b.Nivel, Texto: b.Texto, Pares: b.Pares, Cabeceira: b.Cabeceira, Filas: b.Filas, Elementos: b.Elementos}
	if b.Tipo == bloqueImaxe {
		v.Texto = b.Texto // só o pé: os bytes da imaxe non van á vista previa
	}
	return v
}

// destino converte un destino en bloques (0, 1 ou máis; 0 = non sae).
func (e *avaliador) destino(n *NodoFluxo) []bloqueDoc {
	entrada := func(p string) (valorFluxo, bool) {
		o, ok := n.Entradas[p]
		if !ok {
			return valorFluxo{}, false
		}
		return e.valor(o), true
	}
	cab := func() []bloqueDoc {
		if t := cfgTexto(n, "titulo"); t != "" {
			return []bloqueDoc{{Tipo: bloqueTitulo, Nivel: 3, Texto: t}}
		}
		return nil
	}
	switch n.Tipo {
	case nodoTitulo:
		txt := cfgTexto(n, "texto")
		if v, ok := entrada("texto"); ok {
			if v.Vacio {
				return nil
			}
			txt = v.Texto
		}
		nivel := 2
		if cfgTexto(n, "nivel") == "3" {
			nivel = 3
		}
		if txt == "" {
			return nil
		}
		return []bloqueDoc{{Tipo: bloqueTitulo, Nivel: nivel, Texto: txt}}
	case nodoParagrafo:
		v, _ := entrada("texto")
		if v.Vacio || strings.TrimSpace(v.Texto) == "" {
			return nil
		}
		return append(cab(), bloqueDoc{Tipo: bloqueMarkdown, Texto: v.Texto})
	case nodoFicha:
		var portos []string
		for p := range n.Entradas {
			if p != "se" {
				portos = append(portos, p)
			}
		}
		sort.Slice(portos, func(i, j int) bool { return ordePorto(portos[i]) < ordePorto(portos[j]) })
		var pares [][2]string
		for _, p := range portos {
			v := e.valor(n.Entradas[p])
			if v.Vacio || (v.Tipo != dBool && strings.TrimSpace(v.Texto) == "") {
				continue
			}
			etq := cfgTexto(n, "etq_"+p)
			if etq == "" {
				if o := e.a.nodos[n.Entradas[p]]; o != nil && o.Tipo == nodoCampo {
					etq = e.a.campo[o.Campo].Etiqueta
				} else {
					etq = p
				}
			}
			pares = append(pares, [2]string{etq, v.Texto})
		}
		if len(pares) == 0 {
			return nil
		}
		return []bloqueDoc{{Tipo: bloqueFicha, Pares: pares}}
	case nodoTaboa:
		v, _ := entrada("taboa")
		if v.Vacio || len(v.Filas) == 0 {
			return nil
		}
		b := bloqueDoc{Tipo: bloqueTaboa, Texto: cfgTexto(n, "titulo")}
		for _, c := range v.Cols {
			b.Cabeceira = append(b.Cabeceira, c.Etiqueta)
		}
		for _, f := range v.Filas {
			linha := make([]string, len(v.Cols))
			for j, c := range v.Cols {
				linha[j], _ = celaFormatada(c, f[c.ID], e.tx)
			}
			b.Filas = append(b.Filas, linha)
		}
		return append(cab(), b)
	case nodoLista:
		v, _ := entrada("taboa")
		col, _ := columnaPorId(v.Cols, cfgTexto(n, "columna"))
		var els []string
		for _, f := range v.Filas {
			if s, _ := celaFormatada(col, f[col.ID], e.tx); s != "" {
				els = append(els, s)
			}
		}
		if len(els) == 0 {
			return nil
		}
		return append(cab(), bloqueDoc{Tipo: bloqueLista, Elementos: els})
	case nodoGrafico:
		return e.destinoGrafico(n, cab())
	case nodoImaxe:
		v, _ := entrada("imaxe")
		if v.Vacio || v.Imaxe == "" {
			return nil
		}
		return []bloqueDoc{{Tipo: bloqueImaxe, Texto: cfgTexto(n, "titulo"), Imaxe: v.Imaxe}}
	case nodoSalto:
		return []bloqueDoc{{Tipo: bloqueSalto}}
	case nodoProblemas:
		v, _ := entrada("problemas")
		if v.Vacio {
			return nil
		}
		return []bloqueDoc{{Tipo: bloqueProblemas, Problemas: v.Problemas, Desprazamento: v.Desprazamento}}
	}
	return nil
}

// ordePorto ordena v2 antes ca v10.
func ordePorto(p string) int {
	if len(p) > 1 && p[0] == 'v' {
		if n, err := strconv.Atoi(p[1:]); err == nil {
			return n
		}
	}
	return 1000 + int(p[0])
}

// bloquesDoFluxo é bloquesFormulario cando o formulario ten tubería.
func bloquesDoFluxo(f Formulario, d DatosFormulario, tx textosDocumento) ([]bloqueDoc, []string) {
	d = d.aplicarReferencias()
	avisos := avisosDatos(f, d, tx)
	bloques, res := avaliarFluxoDoc(f, d, tx)
	if res.Erro != "" {
		avisos = append(avisos, "a tubería non é válida: "+res.Erro)
		if t := strings.TrimSpace(tx.T("FORM_SEN_CONTIDO")); t != "" {
			return []bloqueDoc{{Tipo: bloqueMarkdown, Texto: t}}, avisos
		}
		return nil, avisos
	}
	return bloques, engadirSenRepetir(avisos, res.Avisos)
}
