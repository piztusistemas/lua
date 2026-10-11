package main

// Formato .json dun FORMULARIO para pasalo entre compañeiros (o das versións
// anteriores a Trasladar). Lúa xa non o escribe: un formulario sae dentro dun
// paquete .lua ou do .json editable dun documento (trasladar.go). Pero segue
// léndoo, así que aquí queda a súa lóxica pura (bytes -> bytes).
//
// O ficheiro é DATOS, nunca código: un paquete co marcador «lua-formulario»,
// a versión do formato, o esquema (campos e tubería) e, se o docente o pide,
// os datos. Ao importar todo pasa polas validacións de sempre
// (normalizarFormulario/validarFormulario) e NUNCA se pisa un formulario que xa
// exista: se o id coincide, entra coma unha copia con outro id e outro nome.
//
// PRIVACIDADE (ver o modo dun formulario en formulario.go): o modo non viaxa no
// ficheiro. Un formulario importado nace en DESEÑO (onde a IA pode axudar con
// datos ficticios), agás se o ficheiro trae datos que non están marcados coma
// de proba: entón entra en PRODUCIÓN, onde a IA non ten acceso ningún.
// Non leva plantillas: van aparte e levan LaTeX e imaxes.

import (
	"edicion/idioma"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	formatoBundleFormulario   = "lua-formulario"
	versionBundleFormulario   = 1
	extensionBundleFormulario = ".formulario.json"
	// maxBytesBundleFormulario limita o que se le ao importar (as imaxes van en base64).
	maxBytesBundleFormulario = 40 << 20
)

// DatosIntercambioFormulario son os datos que pode levar un paquete: o mesmo
// que garda o documento dun formulario («datos» de escribirFicheiroFormulario).
type DatosIntercambioFormulario struct {
	Contexto ContextoFormulario `json:"contexto"`
	Valores  map[string]any     `json:"valores"`
	// Proba: os datos son ficticios (postos pola IA en modo deseño).
	Proba bool `json:"proba,omitempty"`
}

// BundleFormulario é o contido do ficheiro.
type BundleFormulario struct {
	Formato    string                      `json:"formato"`
	Version    int                         `json:"version"`
	Formulario Formulario                  `json:"formulario"`
	Datos      *DatosIntercambioFormulario `json:"datos,omitempty"`
}

// ResultadoImportacionFormulario é o que recibe o frontend tras importar.
type ResultadoImportacionFormulario struct {
	// Formulario, tal e como quedou na biblioteca (id e nome finais, modo).
	Formulario Formulario `json:"formulario"`
	// Datos que trouxo o ficheiro (xa deixados coma borrador do formulario), ou nil.
	Datos  *DatosIntercambioFormulario `json:"datos,omitempty"`
	Avisos []string                    `json:"avisos,omitempty"`
}

// haiDatosIntercambio di se hai algo escrito (un formulario sen contido non leva «datos»).
func haiDatosIntercambio(d *DatosIntercambioFormulario) bool {
	if d == nil {
		return false
	}
	c := d.Contexto
	if strings.TrimSpace(c.Centro+c.Localidade+c.Curso+c.Departamento+c.Responsable+c.Cargo+c.Logo) != "" {
		return true
	}
	for _, v := range d.Valores {
		if valorTemContido(v) {
			return true
		}
	}
	return false
}

func valorTemContido(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(x) != ""
	case bool:
		return x
	case []any:
		for _, f := range x {
			if m, ok := f.(map[string]any); ok {
				for _, c := range m {
					if valorTemContido(c) {
						return true
					}
				}
			}
		}
		return false
	}
	return true
}

// sanearDatosIntercambio deixa só os datos que o formulario recoñece: os
// valores dos seus campos con valor (non seccións) e o contexto tipado. Devolve
// nil se non queda nada. `descartados` é cantos valores sobraban.
func sanearDatosIntercambio(f Formulario, d *DatosIntercambioFormulario) (san *DatosIntercambioFormulario, descartados int) {
	if d == nil {
		return nil, 0
	}
	campos := map[string]bool{}
	for _, c := range f.Campos {
		if c.Tipo != campoSeccion {
			campos[c.ID] = true
		}
	}
	san = &DatosIntercambioFormulario{Contexto: d.Contexto, Valores: map[string]any{}, Proba: d.Proba}
	for id, v := range d.Valores {
		if campos[id] {
			san.Valores[id] = v
		} else {
			descartados++
		}
	}
	if !haiDatosIntercambio(san) {
		return nil, descartados
	}
	return san, descartados
}

// construirBundleFormulario arma o paquete dun formulario xa gardado. O modo
// non viaxa. Os datos, se hai, pasan polo saneado.
func construirBundleFormulario(f Formulario, datos *DatosIntercambioFormulario) BundleFormulario {
	f.Modo = ""
	b := BundleFormulario{Formato: formatoBundleFormulario, Version: versionBundleFormulario, Formulario: f}
	b.Datos, _ = sanearDatosIntercambio(f, datos)
	return b
}

// escribirBundleFormulario serializa un paquete.
func escribirBundleFormulario(b BundleFormulario) ([]byte, error) {
	return json.MarshalIndent(b, "", "  ")
}

// nomeLibreFormulario devolve `nome`, ou «nome (2)», «nome (3)»... se xa hai un igual.
func nomeLibreFormulario(nome string, usados map[string]bool) string {
	if !usados[strings.ToLower(nome)] {
		return nome
	}
	for n := 2; ; n++ {
		c := fmt.Sprintf("%s (%d)", nome, n)
		if !usados[strings.ToLower(c)] {
			return c
		}
	}
}

// importarBundleFormulario le un paquete, valida o formulario e gárdao na
// biblioteca coma formulario NOVO (nunca pisa un existente). Ver a nota de
// privacidade do principio para o modo.
func importarBundleFormulario(data []byte) (ResultadoImportacionFormulario, error) {
	var res ResultadoImportacionFormulario
	if len(data) > maxBytesBundleFormulario {
		return res, idioma.Errorf("go.ficheiro.grande", "o ficheiro é demasiado grande (máximo %d MB)", maxBytesBundleFormulario>>20)
	}
	// Primeiro só a cabeceira: un ficheiro doutra clase (un documento cos datos
	// dun formulario, unha plantilla) ten que dar unha mensaxe clara, non un
	// erro de tipos ao ler o resto.
	var cab struct {
		Formato    string `json:"formato"`
		Version    int    `json:"version"`
		Ferramenta string `json:"ferramenta"`
	}
	if err := json.Unmarshal(data, &cab); err != nil {
		return res, idioma.Errorf("go.ficheiro.json", "o ficheiro non é un JSON válido: %w", err)
	}
	if cab.Ferramenta == "formulario" {
		return res, idioma.Errorf("go.formulario.son_datos", "este ficheiro é un documento cos DATOS dun formulario, non o formulario en si: ábreo desde o formulario ao que pertence")
	}
	if cab.Formato != formatoBundleFormulario {
		return res, idioma.Errorf("go.formulario.sen_marcador", "o ficheiro non é un formulario de Lúa (falta o marcador «%s»)", formatoBundleFormulario)
	}
	if cab.Version < 1 || cab.Version > versionBundleFormulario {
		return res, idioma.Errorf("go.formulario.version", "o ficheiro ten a versión %d do formato e este Lúa só entende ata a %d: actualiza Lúa", cab.Version, versionBundleFormulario)
	}
	var b BundleFormulario
	if err := json.Unmarshal(data, &b); err != nil {
		return res, idioma.Errorf("go.formulario.non_valido", "o formulario do ficheiro non é válido: %w", err)
	}

	f := normalizarFormulario(b.Formulario)
	if err := validarFormulario(f); err != nil {
		return res, idioma.Errorf("go.formulario.non_valido", "o formulario do ficheiro non é válido: %w", err)
	}

	// Nunca se pisa un formulario: se o id xa existe, entra coma copia.
	existentes, err := listarFormularios()
	if err != nil {
		return res, err
	}
	ids, nomes := map[string]bool{}, map[string]bool{}
	for _, e := range existentes {
		ids[e.ID] = true
		nomes[strings.ToLower(e.Nome)] = true
	}
	if ids[f.ID] {
		f.ID = idLibre(f.ID, ids)
		f.Nome = nomeLibreFormulario(f.Nome, nomes)
		res.Avisos = append(res.Avisos, idioma.T("go.formulario.copia", "Xa tiñas un formulario con ese identificador: o importado entrou coma copia, «%s».", f.Nome))
	}

	f, err = gardarFormulario(f)
	if err != nil {
		return res, err
	}

	datos, descartados := sanearDatosIntercambio(f, b.Datos)
	if descartados > 0 {
		res.Avisos = append(res.Avisos, idioma.T("go.formulario.datos_descartados", "%d datos do ficheiro non correspondían a ningún campo do formulario e non se cargaron.", descartados))
	}
	// Datos que non son de proba: o formulario entra en PRODUCIÓN (sen IA).
	if datos != nil && !datos.Proba {
		if f, err = cambiarModoFormulario(f.ID, modoProduccion); err != nil {
			return res, err
		}
		res.Avisos = append(res.Avisos, "O ficheiro trae datos que non están marcados coma de proba: o formulario entrou en modo PRODUCIÓN, sen asistente de IA.")
	}
	if datos != nil {
		if err := escribirBorradorImportado(f.ID, datos); err != nil {
			return res, idioma.Errorf("go.formulario.datos_non_cargados", "o formulario entrou na biblioteca pero non se puideron cargar os seus datos: %w", err)
		}
	}
	res.Formulario, res.Datos = f, datos
	return res, nil
}

// rutaBorradorFormulario é onde a app garda o traballo en curso (o borrador) dun
// formulario. É unha variable para que os tests non escriban na configuración real.
var rutaBorradorFormulario = func(id string) (string, error) {
	return borradorFerramentaPath(idFerramentaForm(id))
}

// escribirBorradorImportado deixa os datos importados coma o borrador do
// formulario novo: cando a app o abre, recupéraos pola mesma vía ca calquera
// traballo en curso (sen gardar) e non hai carreira co arrinque da ferramenta.
// O formato é o do documento dun formulario (escribirFicheiroFormulario, en JS).
func escribirBorradorImportado(id string, d *DatosIntercambioFormulario) error {
	ruta, err := rutaBorradorFormulario(id)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(map[string]any{"ferramenta": "formulario", "formulario": id, "datos": d}, "", "  ")
	if err != nil {
		return err
	}
	return escribirAtomicoCrono(ruta, raw)
}
