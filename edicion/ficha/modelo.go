// Package ficha é o modelo común de LuaYang: a peza única que substitúe ao
// exercicio de Yang e ao formulario de Lua (ver luayang/ESPECIFICACION.md).
//
// Unha ficha é un ficheiro Markdown da bóveda: metadatos e campos en YAML
// (front matter) e un corpo dividido en seccións «## …». Este paquete só
// sabe ler, escribir, validar e importar fichas; non xera documentos nin
// fala con Maxima (iso vai noutros paquetes, para que o modelo non arrastre
// dependencias).
package ficha

// Version é o número que vai na clave `luayang` de toda ficha.
const Version = 1

// Tipos de ficha.
const (
	TipoExercicio  = "exercicio"
	TipoExame      = "exame"
	TipoBanco      = "banco"
	TipoFormulario = "formulario"
	TipoRubrica    = "rubrica"
)

var tiposFicha = []string{TipoExercicio, TipoExame, TipoBanco, TipoFormulario, TipoRubrica}

// Formatos do corpo. En Markdown (o de sempre) o texto é Markdown + $LaTeX$
// + {campos}. En Matex o texto de cada sección é o dialecto de Yang (HTML con
// <MAT>, <EVAL>, <PLOT>…) e xérase co motor de Yang: así os exercicios
// importados saen exactamente igual ca en Yang, sen reescribilos.
const (
	FormatoMarkdown = "markdown"
	FormatoMatex    = "matex"
)

// Orixes dun campo: quen pon o valor.
const (
	OrixeEscrito   = "escrito"   // a persoa, nun formulario (Lua)
	OrixeFixo      = "fixo"      // a ficha, sempre o mesmo (Yang: fixa)
	OrixeAleatorio = "aleatorio" // o azar, coa semente (Yang: aleatoria)
	OrixeCalculado = "calculado" // unha fórmula doutros campos (Yang: auto; Lua: calcular)
	OrixeTaboa     = "taboa"     // filas sorteadas dun banco
)

var orixes = []string{OrixeEscrito, OrixeFixo, OrixeAleatorio, OrixeCalculado, OrixeTaboa}

// Motores de cálculo.
const (
	MotorFormula = "formula" // o avaliador de Lua: Go, sen dependencias
	MotorMaxima  = "maxima"  // Maxima (opcional)
)

// Xeradores de valores aleatorios. Os nX/zX/qX son os de Yang
// (codeini_default.ini): naturais, enteiros e racionais pequenos, medianos e
// grandes.
const (
	XeradorRango = "rango"
	XeradorLista = "lista"
)

var xeradoresYang = []string{"n0", "n1", "n2", "z0", "z1", "z2", "q0", "q1", "q2"}

// Ficha é o contido dun ficheiro .md de LuaYang.
type Ficha struct {
	Luayang   int      `yaml:"luayang" json:"luayang"`
	ID        string   `yaml:"id,omitempty" json:"id,omitempty"`
	Tipo      string   `yaml:"tipo" json:"tipo"`
	Titulo    string   `yaml:"titulo,omitempty" json:"titulo,omitempty"`
	Materia   string   `yaml:"materia,omitempty" json:"materia,omitempty"`
	Nivel     string   `yaml:"nivel,omitempty" json:"nivel,omitempty"`
	Formato   string   `yaml:"formato,omitempty" json:"formato,omitempty"` // baleiro = markdown
	Conceptos []string `yaml:"conceptos,omitempty" json:"conceptos,omitempty"`
	Fonte     *Fonte   `yaml:"fonte,omitempty" json:"fonte,omitempty"`
	Puntos    float64  `yaml:"puntos,omitempty" json:"puntos,omitempty"`
	Linguas   []string `yaml:"linguas,omitempty" json:"linguas,omitempty"`
	Plantilla string   `yaml:"plantilla,omitempty" json:"plantilla,omitempty"`
	// Columnas: só nos bancos (a táboa vai no corpo).
	Columnas []string `yaml:"columnas,omitempty" json:"columnas,omitempty"`
	Campos   []Campo  `yaml:"campos,omitempty" json:"campos,omitempty"`
	Probas   *Probas  `yaml:"probas,omitempty" json:"probas,omitempty"`
	// Formulario: o que só ten sentido nun formulario de Lua.
	Formulario *OpcionsFormulario `yaml:"formulario,omitempty" json:"formulario,omitempty"`
	// Calculo: a tubería de Lua tal cal (nodos e documento). Ata que o corpo
	// a substitúa (fase 5) viaxa sen tocar, para non perder nada.
	Calculo any `yaml:"calculo,omitempty" json:"calculo,omitempty"`
	// Importado: de onde veu a ficha, para poder volver ao formato orixinal.
	Importado *Importado `yaml:"importado,omitempty" json:"importado,omitempty"`
	// Extra: claves que este paquete non coñece (escritas a man en Obsidian,
	// ou dunha versión futura). Consérvanse ao gardar.
	Extra map[string]any `yaml:",inline" json:"extra,omitempty"`

	// Corpo: o Markdown despois do front matter, exacto.
	Corpo string `yaml:"-" json:"corpo"`
}

// Fonte: o documento orixinal dunha ficha importada.
type Fonte struct {
	PDF      string `yaml:"pdf,omitempty" json:"pdf,omitempty"`
	Paxina   int    `yaml:"paxina,omitempty" json:"paxina,omitempty"`
	Apartado string `yaml:"apartado,omitempty" json:"apartado,omitempty"`
}

// Probas: o que se comproba en cada semente (a lapela Test de Yang).
type Probas struct {
	Sementes   int      `yaml:"sementes,omitempty" json:"sementes,omitempty"`
	Condicions []string `yaml:"condicions,omitempty" json:"condicions,omitempty"`
}

// OpcionsFormulario: os atributos dun formulario de Lua que non son campos.
type OpcionsFormulario struct {
	Icona      string `yaml:"icona,omitempty" json:"icona,omitempty"`
	Descricion string `yaml:"descricion,omitempty" json:"descricion,omitempty"`
	// Modo: «deseno» ou «produccion» (en produción a IA non ten acceso).
	Modo     string `yaml:"modo,omitempty" json:"modo,omitempty"`
	Titulo   string `yaml:"titulo,omitempty" json:"titulo,omitempty"` // título do documento; baleiro = o da ficha
	Portada  bool   `yaml:"portada,omitempty" json:"portada,omitempty"`
	Sinatura bool   `yaml:"sinatura,omitempty" json:"sinatura,omitempty"`
}

// Importado: a orixe dunha ficha importada.
type Importado struct {
	De string `yaml:"de" json:"de"`                     // "yang-taboa", "yang", "lua"
	ID string `yaml:"id,omitempty" json:"id,omitempty"` // o id no formato orixinal
	// SenTitulo: o orixinal non tiña título e o da ficha é sacado do texto.
	SenTitulo bool `yaml:"sen_titulo,omitempty" json:"sen_titulo,omitempty"`
}

// Campo: unha entrada da ficha. Que claves usa depende da Orixe.
type Campo struct {
	Nome  string `yaml:"nome" json:"nome"`
	Orixe string `yaml:"orixe,omitempty" json:"orixe,omitempty"` // baleira só nas columnas dunha táboa
	// Tipo de dato (os de Lua): texto, textolongo, numero, data, escolla,
	// verificacion, imaxe, taboa, seccion; e expresion (un obxecto de Maxima).
	Tipo        string   `yaml:"tipo,omitempty" json:"tipo,omitempty"`
	Etiqueta    string   `yaml:"etiqueta,omitempty" json:"etiqueta,omitempty"`
	Axuda       string   `yaml:"axuda,omitempty" json:"axuda,omitempty"`
	Obrigatorio bool     `yaml:"obrigatorio,omitempty" json:"obrigatorio,omitempty"`
	Agochado    bool     `yaml:"agochado,omitempty" json:"agochado,omitempty"`
	Opcions     []string `yaml:"opcions,omitempty" json:"opcions,omitempty"`   // escolla
	Columnas    []Campo  `yaml:"columnas,omitempty" json:"columnas,omitempty"` // taboa
	// Rol: RolResultado marca o resultado final dun apartado (os
	// `resultados` da táboa de Yang): sae na sección Resultado e é a única
	// fonte de verdade da corrección.
	Rol string `yaml:"rol,omitempty" json:"rol,omitempty"`

	// fixo
	Valor string `yaml:"valor,omitempty" json:"valor,omitempty"`

	// aleatorio
	Xerador string   `yaml:"xerador,omitempty" json:"xerador,omitempty"`
	De      string   `yaml:"de,omitempty" json:"de,omitempty"`
	Ata     string   `yaml:"ata,omitempty" json:"ata,omitempty"`
	Paso    string   `yaml:"paso,omitempty" json:"paso,omitempty"` // baleiro = 1
	Valores []string `yaml:"valores,omitempty" json:"valores,omitempty"`

	// calculado
	Motor   string `yaml:"motor,omitempty" json:"motor,omitempty"`
	Expr    string `yaml:"expr,omitempty" json:"expr,omitempty"`
	Formato string `yaml:"formato,omitempty" json:"formato,omitempty"` // latex, texto, numero
	// Funcion: definición de función de Maxima (f(x) := …), coma o defOp
	// «:=» de Yang.
	Funcion bool `yaml:"funcion,omitempty" json:"funcion,omitempty"`

	// taboa
	Banco        string `yaml:"banco,omitempty" json:"banco,omitempty"`
	Mostra       int    `yaml:"mostra,omitempty" json:"mostra,omitempty"`
	DistintasPor string `yaml:"distintas_por,omitempty" json:"distintas_por,omitempty"`
}

// FormatoCorpo devolve o formato efectivo do corpo.
func (f *Ficha) FormatoCorpo() string {
	if f.Formato == "" {
		return FormatoMarkdown
	}
	return f.Formato
}

func conten(lista []string, s string) bool {
	for _, x := range lista {
		if x == s {
			return true
		}
	}
	return false
}
