package plantilla

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Intercambio de plantillas entre compañeiros. Unha plantilla non vive nun só
// ficheiro: o texto vai en plantillas.json e as imaxes nun cartafol á parte, así
// que para compartila hai que xuntar as dúas cousas nun ficheiro portable.
//
// É un .json normal con TODO dentro —só o que o profesorado edita (nada de ID,
// datas nin DeFabrica: eses póñoos de novo o equipo que a importe) e as imaxes
// en base64—. Texto plano a propósito, para poder revisalo (e, se fai falla,
// editalo) antes de mandarllo a ninguén.
//
// O que marca que ese .json é unha plantilla deste módulo non é a extensión
// senón o campo "formato" de dentro. As versións novas escriben o marcador
// NEUTRO (FormatoBundle) e mais aceptan os dous marcadores históricos, de xeito
// que un bundle exportado por calquera das aplicacións ábrese na outra.
const (
	// FormatoBundle é o marcador neutro que escriben as versións novas.
	FormatoBundle = "piztu-plantilla"
	// formatoBundleYang e formatoBundleLua son os marcadores das versións
	// anteriores (cando cada aplicación tiña o seu): acéptanse ao importar.
	formatoBundleYang = "yang-plantilla"
	formatoBundleLua  = "lua-plantilla"

	// VersionBundle é a versión do formato do ficheiro de intercambio.
	VersionBundle = 1
	// ExtensionBundle é a extensión que se propón ao exportar e ao importar.
	ExtensionBundle = ".json"
)

// ImaxeBundle é unha imaxe da plantilla metida no propio ficheiro de
// intercambio (mesmo par nome/base64 ca DocumentoMostra).
type ImaxeBundle struct {
	Nome    string `json:"nome"`
	DataB64 string `json:"dataB64"`
}

// Bundle é o contido dun ficheiro .json de plantilla.
type Bundle struct {
	Formato string `json:"formato"`
	Version int    `json:"version"`
	// Plantilla reutiliza Plantilla, pero os campos locais a cada equipo (ID,
	// Creado, Modificado, DeFabrica, Imaxes) van sempre baleiros: escríbeos de
	// novo o equipo que importe.
	Plantilla Plantilla     `json:"plantilla"`
	Imaxes    []ImaxeBundle `json:"imaxes"`
}

// Opcions reúne o que o módulo necesita da aplicación para o intercambio.
type Opcions struct {
	// Nome é o nome visible da aplicación, para as mensaxes de erro.
	Nome string
	// DirImaxes devolve o cartafol de imaxes dunha plantilla (as rutas de
	// configuración son da aplicación).
	DirImaxes func(id string) (string, error)
	// NormalizarTextos limpa o mapa de textos da plantilla. O catálogo de
	// claves coñecidas vive na aplicación; se é nil, cópiase tal cal.
	NormalizarTextos func(map[string]string) map[string]string
}

// Importador é o que o módulo necesita da aplicación para dar de alta unha
// plantilla importada.
type Importador interface {
	// Gardar crea unha plantilla nova (ID baleira) coa validación da
	// aplicación (nome, {{CORPO}}...) e devolve a gardada.
	Gardar(p Plantilla) (Plantilla, error)
	// Imaxes devolve os nomes das imaxes gardadas desa plantilla.
	Imaxes(id string) []string
	// SubirImaxe garda unha imaxe da plantilla.
	SubirImaxe(id, nome, dataB64 string) error
}

// ConstruirBundle arma o bundle dunha plantilla xa gardada: o seu contido
// editable, sen os metadatos locais, e as súas imaxes lidas do cartafol e postas
// en base64.
func ConstruirBundle(p Plantilla, o Opcions) (Bundle, error) {
	textos := p.Textos
	if o.NormalizarTextos != nil {
		textos = o.NormalizarTextos(p.Textos)
	}
	b := Bundle{
		Formato: FormatoBundle,
		Version: VersionBundle,
		Plantilla: Plantilla{
			Nome:        strings.TrimSpace(p.Nome),
			Descricion:  strings.TrimSpace(p.Descricion),
			Latex:       p.Latex,
			Portada:     p.Portada,
			Sinatura:    p.Sinatura,
			Introducion: p.Introducion,
			Estilo:      p.Estilo,
			Textos:      textos,
			Markdown:    p.Markdown,
			Ofimatica:   p.Ofimatica,
			Motor:       p.Motor,
			Variables:   NormalizarVariables(p.Variables),
		},
	}

	if o.DirImaxes == nil {
		return b, nil
	}
	dir, err := o.DirImaxes(p.ID)
	if err != nil {
		return Bundle{}, err
	}
	entradas, err := os.ReadDir(dir)
	if err != nil {
		return b, nil // a plantilla non ten imaxes
	}
	for _, e := range entradas {
		if e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return Bundle{}, err
		}
		b.Imaxes = append(b.Imaxes, ImaxeBundle{
			Nome:    e.Name(),
			DataB64: base64.StdEncoding.EncodeToString(data),
		})
	}
	return b, nil
}

// ImportarBundle valida un .json de plantilla xa lido e gárdao coma plantilla
// NOVA neste equipo: ID nova por Gardar (que reaproveita a validación de nome e
// {{CORPO}}), e despois as imaxes unha a unha polo mesmo camiño ca "Engadir
// imaxe…" do editor. Devolve a plantilla gardada, xa coa ID e coas imaxes
// resoltas.
func ImportarBundle(data []byte, o Opcions, imp Importador) (Plantilla, error) {
	var b Bundle
	if err := json.Unmarshal(data, &b); err != nil {
		return Plantilla{}, fmt.Errorf("o ficheiro non é unha plantilla de %s válida: %w", o.Nome, err)
	}
	if !formatoAceptado(b.Formato) {
		return Plantilla{}, fmt.Errorf("o ficheiro non é unha plantilla de %s", o.Nome)
	}
	if b.Version > VersionBundle {
		return Plantilla{}, fmt.Errorf("esta plantilla fíxose cunha versión máis nova de %s: actualiza %s para importala", o.Nome, o.Nome)
	}

	p := b.Plantilla
	p.ID = ""
	p.DeFabrica = false
	p.Creado = ""
	p.Modificado = ""
	p.Imaxes = nil
	if strings.TrimSpace(p.Nome) == "" {
		p.Nome = "Plantilla importada"
	}

	gardada, err := imp.Gardar(p)
	if err != nil {
		return Plantilla{}, err
	}

	for _, im := range b.Imaxes {
		if err := imp.SubirImaxe(gardada.ID, im.Nome, im.DataB64); err != nil {
			// A plantilla xa está gardada: devólvese igual, avisando de que
			// unha imaxe non entrou, en vez de perder o texto (o caro de
			// reescribir) por un logo que se pode volver subir.
			gardada.Imaxes = imp.Imaxes(gardada.ID)
			return gardada, fmt.Errorf("plantilla importada, pero a imaxe %q non se puido gardar: %w", im.Nome, err)
		}
	}
	gardada.Imaxes = imp.Imaxes(gardada.ID)
	return gardada, nil
}

// formatoAceptado di se o marcador dun bundle é dunha versión coñecida: o
// neutro actual ou calquera dos dous históricos.
func formatoAceptado(formato string) bool {
	switch formato {
	case FormatoBundle, formatoBundleYang, formatoBundleLua:
		return true
	}
	return false
}

// NombreFicheiro converte o nome dunha plantilla nun nome de ficheiro razoable
// para propoñer no diálogo de gardar ("Circular do centro" -> "Circular do
// centro"). Quita só o que non pode ir nun nome de ficheiro en ningún sistema;
// o profesorado sempre pode renomear no propio diálogo.
func NombreFicheiro(nome string) string {
	nome = strings.TrimSpace(nome)
	var b strings.Builder
	for _, r := range nome {
		if strings.ContainsRune(`/\:*?"<>|`, r) || r < 0x20 {
			b.WriteRune('-')
			continue
		}
		b.WriteRune(r)
	}
	limpo := strings.Trim(strings.TrimSpace(b.String()), ".")
	if limpo == "" {
		return "plantilla"
	}
	return limpo
}
