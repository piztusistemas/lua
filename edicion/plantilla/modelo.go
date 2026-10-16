// Package plantilla contén o modelo e a maquinaria das plantillas de documento
// que compartían Yang e Lua: o tipo superconxunto, o almacén (con versión de
// formato), a fábrica, o intercambio entre compañeiros e a xeración con IA.
//
// A aplicación aporta o dominio (as súas plantillas de fábrica, o catálogo de
// textos, as rutas de configuración) e o módulo a mecánica. Os tipos que
// cruzan a fronteira da aplicación expóñense alí coma ALIAS
// (`type PlantillaLatex = plantilla.Plantilla`), de xeito que os bindings de
// Wails manteñen o nome e os campos JSON.
package plantilla

import (
	"fmt"
	"strings"
)

// Var é unha variable que o profesorado enche unha soa vez na plantilla e o
// motor substitúe en cada xeración: {{CENTRO}}, {{MATERIA}}... O nome vai
// sempre en MAIÚSCULAS sen espazos (NormalizarNombreVar), que é o que fai
// recoñecible un marcador dentro de código LaTeX sen confundilo cos {} propios
// de LaTeX.
type Var struct {
	Nome     string `json:"nome"`
	Etiqueta string `json:"etiqueta"` // como se amosa na UI ("Centro educativo")
	Valor    string `json:"valor"`
}

// Plantilla é unha plantilla completa da biblioteca. É o SUPERCONXUNTO das
// dúas aplicacións: os campos que só usa unha (Portada/Sinatura/Introducion/
// Estilo/Textos/Ofimatica, de Lua) quedan co valor cero na outra, e os ficheiros
// xa gardados seguen abrindo sen perda.
type Plantilla struct {
	ID         string `json:"id"`
	Nome       string `json:"nome"`
	Descricion string `json:"descricion"`
	// Latex é a plantilla en si (completa ou fragmento). Ten que conter
	// {{CORPO}}.
	Latex string `json:"latex"`
	// Portada e Sinatura son o deseño, en LaTeX, da primeira e da última
	// páxina do documento. Van á parte do Latex para poder editalas sen tocar
	// a maqueta (cabeceira, pé, marxes), e entran onde a plantilla poña
	// {{PORTADA}} e {{SINATURA}}. Levan as mesmas variables ca o resto máis o
	// macro \logocentro, o logotipo do curso. Sen ningún dos dous marcadores a
	// plantilla é "clásica" (o motor pon a portada e a sinatura por defecto).
	Portada  string `json:"portada"`
	Sinatura string `json:"sinatura"`
	// Introducion é o texto da introdución (con {{VARIABLES}}: o centro, o
	// curso, o departamento...). Baleiro = o texto por defecto.
	Introducion string `json:"introducion"`
	// Estilo é o DESEÑO do documento en LaTeX libre (títulos, etiquetas,
	// cabeceiras). Se ten algo, manda sobre Deseno. Baleiro = Deseno ou, sen
	// el, o deseño por defecto da aplicación.
	Estilo string `json:"estilo"`
	// Deseno é o deseño do PDF coma datos (deseno.go): o que edita o editor
	// visual e a IA. nil = o deseño de Lúa.
	Deseno *Deseno `json:"deseno,omitempty"`
	// Textos son os textos fixos do documento que esta plantilla cambia
	// (clave -> texto). Os que faltan usan o valor por defecto da aplicación.
	Textos map[string]string `json:"textos,omitempty"`
	// Markdown é a versión equivalente para .md/.docx/.odt: mesmo {{CORPO}} e
	// mesmas variables, pero en Markdown. Baleiro = esas exportacións saen sen
	// maqueta, coma se non houbese plantilla.
	Markdown string `json:"markdown"`
	// Ofimatica é o deseño da plantilla para Word (.docx) e OpenDocument
	// (.odt): tipo de letra, cores, portada, cabeceira e pé.
	Ofimatica Ofimatica `json:"ofimatica"`
	// Motor forza o compilador desta plantilla ("xelatex" para unha que use
	// fontspec/fontes do sistema, por exemplo). Baleiro = o de Opcións.
	Motor string `json:"motor"`
	// Variables son as definidas POLA plantilla. As automáticas ({{DATA}},
	// {{FICHEIRO}}, {{TITULO}}...) non se listan aquí: dáas a aplicación, e
	// unha variable con ese mesmo nome definida aquí gaña se ten valor.
	Variables []Var `json:"variables"`
	// Imaxes é sempre o contido REAL do cartafol de imaxes da plantilla:
	// recalcúlase en cada listaxe/gardado en vez de fiarse do que houbese no
	// JSON, así non se desincroniza se alguén toca o cartafol á man ou copia un
	// plantillas.json doutro equipo.
	Imaxes     []string `json:"imaxes"`
	Creado     string   `json:"creado"`
	Modificado string   `json:"modificado"`
	// DeFabrica marca as plantillas de exemplo que a aplicación sementa a
	// primeira vez. Só serve para que a UI as poida distinguir; edítanse e
	// bórranse coma calquera outra.
	DeFabrica bool `json:"deFabrica"`
}

// Ofimatica é o deseño da plantilla para Word (.docx) e OpenDocument (.odt).
// Os valores baleiros/cero significan "o por defecto da aplicación".
type Ofimatica struct {
	Fonte         string  `json:"fonte"`         // "" = Arial
	CorPrincipal  string  `json:"corPrincipal"`  // hex sen #: títulos e portada ("" = 1F3864)
	CorSecundaria string  `json:"corSecundaria"` // subtítulos ("" = 2E5496)
	CorTexto      string  `json:"corTexto"`      // corpo ("" = 212529)
	CorApagada    string  `json:"corApagada"`    // textos secundarios ("" = 595959)
	CorLinha      string  `json:"corLinha"`      // filetes e bordos ("" = 8EAADB)
	TamanoCorpo   float64 `json:"tamanoCorpo"`   // pt; 0 = 11
	Portada       string  `json:"portada"`       // "" = a por defecto
	Cabeceira     string  `json:"cabeceira"`     // "" = sen cabeceira
	Pe            string  `json:"pe"`            // texto ao pé, á beira do número
	// Interlinado é o espazo entre liñas do texto (1 = sinxelo, 1.5...); 0 = o
	// por defecto (1.15). As marxes van en cm; 0 = a por defecto de cada formato.
	Interlinado   float64 `json:"interlinado"`
	MarxeSuperior float64 `json:"marxeSuperior"`
	MarxeInferior float64 `json:"marxeInferior"`
	MarxeEsquerda float64 `json:"marxeEsquerda"`
	MarxeDereita  float64 `json:"marxeDereita"`
}

// Almacen é o ficheiro completo (plantillas.json): as plantillas máis o estado
// da escolla. Vai TODO nun ficheiro propio (non en settings.json) por unha
// razón práctica: gardar os axustes substitúe o Settings enteiro polo que manda
// o diálogo de Opcións, así que gardar aquí a plantilla activa faríaa
// desaparecer cada vez que alguén premese "Gardar" en Opcións.
type Almacen struct {
	Plantillas []Plantilla `json:"plantillas"`
	// Activa é a plantilla que se aplica agora mesmo ("" = ningunha).
	Activa string `json:"activa"`
	// PorFicheiro lembra que plantilla usaba cada documento gardado (ruta
	// absoluta -> ID), para que abrilo restaure a súa maqueta sen tocar o
	// formato do ficheiro. Límpase soa: as entradas de ficheiros que xa non
	// existen bótanse ao gardar.
	PorFicheiro map[string]string `json:"porFicheiro"`
	// Sementado marca que xa se pasou algunha vez por aquí. Xa non decide el
	// só se hai que sementar: consérvase para non romper un plantillas.json
	// escrito por unha versión anterior.
	Sementado bool `json:"sementado"`
	// FabricaBorradas son as IDs de plantillas DE FÁBRICA que o profesorado
	// eliminou. Sen esta lista non se podería distinguir "esta plantilla é
	// nova nesta versión, hai que dala de alta" de "esta xa a borrou a
	// propietaria e non a quere ver máis" - e volverían aparecer soas en cada
	// arranque.
	FabricaBorradas []string `json:"fabricaBorradas"`
	// Version é a versión do formato do ficheiro. Engadiuse despois (os
	// ficheiros vellos veñen sen ela e len como 0), e serve para rexeitar
	// claramente un ficheiro feito por unha versión máis nova en vez de
	// interpretalo mal.
	Version int `json:"version"`
}

// VersionAlmacen é a versión do formato de plantillas.json que escribe esta
// versión do módulo.
const VersionAlmacen = 1

// ComprobarVersion rexeita un plantillas.json feito por unha versión máis nova.
func (a Almacen) ComprobarVersion() error {
	if a.Version > VersionAlmacen {
		return fmt.Errorf("plantillas.json fíxose cunha versión máis nova da aplicación (%d > %d): actualiza para abrilo", a.Version, VersionAlmacen)
	}
	return nil
}

// MarcarVersion deixa o almacén coa versión de formato actual. Chámase sempre
// antes de gardar, para que un ficheiro vello quede migrado en canto se toque.
func (a *Almacen) MarcarVersion() {
	a.Version = VersionAlmacen
}

// Estado é o que precisa a barra de ferramentas dunha soa vez: a lista enteira
// e cal está activa.
type Estado struct {
	Plantillas []Plantilla `json:"plantillas"`
	Activa     string      `json:"activa"`
}

// Marcadores reservados: o corpo do documento e as dúas partes que poden ir
// aparte (portada e sinatura). Non son variables normais e
// NormalizarVariables non deixa que ninguén as defina.
const (
	MarcadorCorpo    = "{{CORPO}}"
	MarcadorPortada  = "{{PORTADA}}"
	MarcadorSinatura = "{{SINATURA}}"
)

// MarcadoresDeParte son os marcadores que a substitución de variables non debe
// tocar: o seu contido insírese despois, xa coas variables substituídas.
func MarcadoresDeParte(nome string) bool {
	return nome == "CORPO" || nome == "PORTADA" || nome == "SINATURA"
}

// NormalizarNombreVar converte o nome dunha variable ao formato canónico:
// MAIÚSCULAS, só letras/números e guións baixos. É o que fai que {{centro}} e
// {{CENTRO}} sexan a mesma cousa e que un marcador non se confunda cun grupo
// LaTeX.
func NormalizarNombreVar(nome string) string {
	nome = strings.ToUpper(strings.TrimSpace(nome))
	var b strings.Builder
	for _, r := range nome {
		switch {
		case r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case strings.ContainsRune("ÁÉÍÓÚÜÑ", r):
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '_':
			b.WriteRune('_')
		}
	}
	return strings.Trim(b.String(), "_")
}

// NormalizarVariables limpa a lista de variables da plantilla: normaliza os
// nomes, descarta os marcadores reservados ({{CORPO}}, {{PORTADA}},
// {{SINATURA}}), quita as duplicadas (gaña a primeira) e ignora as que quedan
// sen nome.
func NormalizarVariables(vars []Var) []Var {
	out := make([]Var, 0, len(vars))
	vistas := map[string]bool{}
	for _, v := range vars {
		v.Nome = NormalizarNombreVar(v.Nome)
		if v.Nome == "" || MarcadoresDeParte(v.Nome) || vistas[v.Nome] {
			continue
		}
		vistas[v.Nome] = true
		out = append(out, v)
	}
	return out
}
