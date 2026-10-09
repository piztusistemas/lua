package main

// Motor de formularios (capa 1: o ESQUEMA). O docente deseña os seus propios
// formularios coa lista de campos que queira; Lua garda o esquema nun JSON da
// biblioteca (unha icona na columna esquerda por formulario), os datos entran
// á man, por CSV ou coa IA (formulario_datos.go) e o documento sae dun fluxo
// que os converte en bloques (formulario_bloques.go) que escriben o PDF
// (formulario_latex.go) e o Markdown (formulario_md.go).
//
// O principio de sempre: o formulario e os datos dan o CONTIDO; o deseño e os
// textos fixos son da plantilla (ver textos_doc.go, «formulario»).

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// ferramentaFormulario é o valor de XerarInformeCronogramaRequest.Ferramenta
// dos formularios propios (as outras, cronograma e actas, están en informe_actas.go).
const ferramentaFormulario = "formulario"

// idFerramentaFormulario devolve o id do formulario dunha «ferramenta» do
// frontend («form:<id>»: cada formulario é unha ferramenta na columna esquerda,
// co seu ficheiro e o seu borrador).
func idFerramentaFormulario(ferramenta string) (string, bool) {
	id, ok := strings.CutPrefix(ferramenta, "form:")
	if !ok || !idFormularioRe.MatchString(id) {
		return "", false
	}
	return id, true
}

// Tipos de campo.
const (
	campoTexto        = "texto"        // liña curta
	campoTextoLongo   = "textolongo"   // Markdown
	campoNumero       = "numero"       // número (coma ou punto decimal)
	campoData         = "data"         // AAAA-MM-DD
	campoEscolla      = "escolla"      // unha das Opcions
	campoVerificacion = "verificacion" // si / non
	campoImaxe        = "imaxe"        // data URI
	campoTaboa        = "taboa"        // filas repetibles das Columnas
	campoSeccion      = "seccion"      // título que agrupa os seguintes (sen valor)
	// campoProblemas (ver formulario_problemas.go): preguntas con fórmulas,
	// variables e solución.
)

// tiposCampoFormulario é o catálogo, en orde de aparición na paleta do deseñador.
var tiposCampoFormulario = []string{
	campoTexto, campoTextoLongo, campoNumero, campoData, campoEscolla,
	campoVerificacion, campoImaxe, campoTaboa, campoProblemas, campoSeccion,
}

// maxCamposFormulario e maxColumnasTaboa poñen un tope sensato: un formulario
// non é unha base de datos e o JSON viaxa enteiro ao frontend.
const (
	maxCamposFormulario = 200
	maxColumnasTaboa    = 12
)

// CampoFormulario é un elemento do formulario.
type CampoFormulario struct {
	// ID é a clave dos datos e da variable {{CAMPO_<ID>}} da plantilla
	// (minúsculas, díxitos e _; empeza por letra).
	ID          string   `json:"id"`
	Tipo        string   `json:"tipo"`
	Etiqueta    string   `json:"etiqueta"`
	Axuda       string   `json:"axuda,omitempty"`
	Obrigatorio bool     `json:"obrigatorio,omitempty"`
	Opcions     []string `json:"opcions,omitempty"` // só «escolla»
	// Columnas: só «taboa». Cada columna é un campo simple (texto, número,
	// data, escolla ou verificación).
	Columnas []CampoFormulario `json:"columnas,omitempty"`
	// Agochado: o campo enche a variable da plantilla pero non sae no corpo.
	Agochado bool `json:"agochado,omitempty"`
}

// Modos dun formulario. En DESEÑO (o de partida) o docente pode cambiar os
// campos e traballar co asistente de IA con datos ficticios; en PRODUCIÓN
// úsase con datos reais e a IA non ten acceso ningún: nin asistente nin
// deseñador. O modo só se cambia con CambiarModoFormulario, nunca gardando o
// esquema (así nin o deseñador nin a IA poden sacar un formulario de produción).
const (
	modoDeseno     = "deseno"
	modoProduccion = "produccion"
)

// Formulario é o esquema completo: o que se garda na biblioteca.
type Formulario struct {
	ID         string `json:"id"`
	Nome       string `json:"nome"`
	Icona      string `json:"icona,omitempty"` // clave do selector de iconas do frontend
	Descricion string `json:"descricion,omitempty"`
	// Modo: modoDeseno ou modoProduccion. Baleiro (un formulario de antes) vale coma deseño.
	Modo string `json:"modo,omitempty"`
	// Titulo é o do documento ({{TITULO}}); baleiro = o nome do formulario.
	Titulo string `json:"titulo,omitempty"`
	// Portada e Sinatura piden a portada e o peche de sinatura da plantilla.
	// Un formulario pequeno (unha ficha, un informe curto) non os quere.
	Portada  bool              `json:"portada,omitempty"`
	Sinatura bool              `json:"sinatura,omitempty"`
	Campos   []CampoFormulario `json:"campos"`
	// Fluxo: as tuberías que levan os datos ata o documento (formulario_fluxo.go).
	// nil = cableado automático: cada campo sae no documento segundo o seu tipo.
	Fluxo *Fluxo `json:"fluxo,omitempty"`
}

// enProduccion di se o formulario está en modo produción.
func (f Formulario) enProduccion() bool { return f.Modo == modoProduccion }

var idFormularioRe = regexp.MustCompile(`^[a-z][a-z0-9_]{0,39}$`)

// tituloDoc é o título do documento.
func (f Formulario) tituloDoc() string {
	if t := strings.TrimSpace(f.Titulo); t != "" {
		return t
	}
	return strings.TrimSpace(f.Nome)
}

// semAcento son as letras con acento (ou ñ, ç) que poden saír nunha etiqueta
// en galego, castelán, portugués ou francés, coa súa letra base.
var semAcento = map[rune]rune{
	'á': 'a', 'à': 'a', 'â': 'a', 'ã': 'a', 'ä': 'a', 'é': 'e', 'è': 'e', 'ê': 'e', 'ë': 'e',
	'í': 'i', 'ì': 'i', 'î': 'i', 'ï': 'i', 'ó': 'o', 'ò': 'o', 'ô': 'o', 'õ': 'o', 'ö': 'o',
	'ú': 'u', 'ù': 'u', 'û': 'u', 'ü': 'u', 'ñ': 'n', 'ç': 'c',
}

// slugFormulario converte un texto libre nun identificador: sen acentos, en
// minúsculas, cos separadores en «_» e empezando por letra. Baleiro se non
// queda nada aproveitable.
func slugFormulario(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if base, ok := semAcento[r]; ok {
			r = base
		}
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			if b.Len() > 0 && !strings.HasSuffix(b.String(), "_") {
				b.WriteByte('_')
			}
		}
	}
	out := strings.Trim(b.String(), "_")
	if out != "" && out[0] >= '0' && out[0] <= '9' {
		out = "c_" + out
	}
	if len(out) > 40 {
		out = strings.Trim(out[:40], "_")
	}
	return out
}

// idLibre devolve `base` (ou base_2, base_3...) que non estea en `usados`.
func idLibre(base string, usados map[string]bool) string {
	if base == "" {
		base = "campo"
	}
	id := base
	for n := 2; usados[id]; n++ {
		suf := fmt.Sprintf("_%d", n)
		id = base
		if len(id)+len(suf) > 40 {
			id = id[:40-len(suf)]
		}
		id += suf
	}
	usados[id] = true
	return id
}

// normalizarFormulario limpa o esquema que chega do frontend ou da IA: recorta
// textos, pon os identificadores que faltan (a partir da etiqueta) e desfai as
// repeticións. Non valida (iso é validarFormulario).
func normalizarFormulario(f Formulario) Formulario {
	f.Nome = strings.TrimSpace(f.Nome)
	f.Descricion = strings.TrimSpace(f.Descricion)
	f.Titulo = strings.TrimSpace(f.Titulo)
	f.Icona = strings.TrimSpace(f.Icona)
	if f.Modo != modoProduccion {
		f.Modo = modoDeseno
	}
	if f.ID = strings.TrimSpace(f.ID); f.ID == "" {
		f.ID = slugFormulario(f.Nome)
	}
	usados := map[string]bool{}
	f.Campos = normalizarCampos(f.Campos, usados, false)
	f.Fluxo = normalizarFluxo(f)
	return f
}

func normalizarCampos(campos []CampoFormulario, usados map[string]bool, columnas bool) []CampoFormulario {
	if len(campos) == 0 {
		return nil
	}
	out := make([]CampoFormulario, 0, len(campos))
	// Primeiro reservan o seu id os que xa o traen: un id novo nunca lle rouba
	// o sitio a un que o formulario tiña.
	for _, c := range campos {
		if id := strings.TrimSpace(c.ID); id != "" && idFormularioRe.MatchString(id) {
			usados[id] = true
		}
	}
	vistos := map[string]bool{}
	for _, c := range campos {
		c.Tipo = strings.TrimSpace(c.Tipo)
		c.Etiqueta = strings.TrimSpace(c.Etiqueta)
		c.Axuda = strings.TrimSpace(c.Axuda)
		id := strings.TrimSpace(c.ID)
		if id == "" || !idFormularioRe.MatchString(id) || vistos[id] {
			id = idLibre(slugFormulario(c.Etiqueta), usados)
		}
		vistos[id] = true
		c.ID = id
		var opcions []string
		for _, o := range c.Opcions {
			if o = strings.TrimSpace(o); o != "" {
				opcions = append(opcions, o)
			}
		}
		c.Opcions = opcions
		if c.Tipo == campoTaboa && !columnas {
			c.Columnas = normalizarCampos(c.Columnas, map[string]bool{}, true)
		} else {
			c.Columnas = nil
		}
		out = append(out, c)
	}
	return out
}

func tipoCampoValido(t string) bool {
	for _, x := range tiposCampoFormulario {
		if x == t {
			return true
		}
	}
	return false
}

// validarFormulario devolve o primeiro problema do esquema (xa normalizado), ou nil.
func validarFormulario(f Formulario) error {
	if f.Nome == "" {
		return errors.New("o formulario non ten nome")
	}
	if !idFormularioRe.MatchString(f.ID) {
		return fmt.Errorf("o identificador %q do formulario non vale (minúsculas, díxitos e _, empezando por letra)", f.ID)
	}
	if len(f.Campos) > maxCamposFormulario {
		return fmt.Errorf("o formulario ten máis de %d campos", maxCamposFormulario)
	}
	usados := map[string]bool{}
	for i, c := range f.Campos {
		if err := validarCampo(c, false); err != nil {
			return fmt.Errorf("campo %d: %w", i+1, err)
		}
		if usados[c.ID] {
			return fmt.Errorf("campo %d: o identificador %q está repetido", i+1, c.ID)
		}
		usados[c.ID] = true
	}
	return validarFluxo(f)
}

func validarCampo(c CampoFormulario, columna bool) error {
	if !tipoCampoValido(c.Tipo) {
		return fmt.Errorf("tipo %q descoñecido", c.Tipo)
	}
	if !idFormularioRe.MatchString(c.ID) {
		return fmt.Errorf("o identificador %q non vale", c.ID)
	}
	if c.Etiqueta == "" {
		return fmt.Errorf("«%s» non ten etiqueta", c.ID)
	}
	switch c.Tipo {
	case campoEscolla:
		if len(c.Opcions) == 0 {
			return fmt.Errorf("«%s» é unha escolla sen opcións", c.Etiqueta)
		}
	case campoTaboa:
		if columna {
			return fmt.Errorf("«%s»: unha táboa non pode ir dentro doutra", c.Etiqueta)
		}
		if len(c.Columnas) == 0 {
			return fmt.Errorf("a táboa «%s» non ten columnas", c.Etiqueta)
		}
		if len(c.Columnas) > maxColumnasTaboa {
			return fmt.Errorf("a táboa «%s» ten máis de %d columnas", c.Etiqueta, maxColumnasTaboa)
		}
		usadas := map[string]bool{}
		for j, col := range c.Columnas {
			if err := validarCampo(col, true); err != nil {
				return fmt.Errorf("táboa «%s», columna %d: %w", c.Etiqueta, j+1, err)
			}
			if usadas[col.ID] {
				return fmt.Errorf("táboa «%s»: a columna %q está repetida", c.Etiqueta, col.ID)
			}
			usadas[col.ID] = true
		}
	}
	if columna {
		switch c.Tipo {
		case campoSeccion, campoImaxe, campoTextoLongo, campoProblemas:
			return fmt.Errorf("«%s»: ese tipo non pode ser columna dunha táboa", c.Etiqueta)
		}
	}
	return nil
}

// ---------- Biblioteca ----------

// dirFormularios é o cartafol da biblioteca (un .json por formulario). É unha
// variable para poder apuntala a un cartafol temporal nos tests.
var dirFormularios = func() (string, error) {
	// Dentro dun proxecto (proxecto.go) os formularios son as copias del.
	if d := dirProxectoActivo(); d != "" {
		return filepath.Join(d, "formularios"), nil
	}
	return dirBibliotecaFormularios()
}

// dirBibliotecaFormularios é SEMPRE o cartafol da biblioteca, esteamos ou non
// dentro dun proxecto (os proxectos cópianse dela).
var dirBibliotecaFormularios = func() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "lua", "formularios"), nil
}

func rutaFormulario(id string) (string, error) {
	if !idFormularioRe.MatchString(id) {
		return "", fmt.Errorf("identificador de formulario non válido: %q", id)
	}
	dir, err := dirFormularios()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, id+".json"), nil
}

// lerFormulario le un formulario da biblioteca.
func lerFormulario(id string) (Formulario, error) {
	ruta, err := rutaFormulario(id)
	if err != nil {
		return Formulario{}, err
	}
	raw, err := os.ReadFile(ruta)
	if err != nil {
		return Formulario{}, err
	}
	var f Formulario
	if err := json.Unmarshal(raw, &f); err != nil {
		return Formulario{}, fmt.Errorf("o formulario %q está estragado: %w", id, err)
	}
	return f, nil
}

// gardarFormulario normaliza, valida e garda o formulario (crea o ficheiro ou
// substitúeo). Devolve o formulario tal e como quedou.
//
// O modo NON se cambia por aquí: un formulario novo nace sempre en deseño e un
// que xa existe conserva o seu (para pasalo de modo está CambiarModoFormulario).
// Un formulario en produción non se pode gardar: o seu esquema queda conxelado
// ata que o docente o pase a deseño.
func gardarFormulario(f Formulario) (Formulario, error) {
	f = normalizarFormulario(f)
	if err := validarFormulario(f); err != nil {
		return f, err
	}
	f.Modo = modoDeseno
	if actual, err := lerFormulario(f.ID); err == nil {
		if actual.enProduccion() {
			return f, fmt.Errorf("o formulario «%s» está en modo produción: pásao a deseño para cambialo", actual.Nome)
		}
	}
	return f, escribirFormulario(f)
}

// escribirFormulario garda o ficheiro do formulario (xa validado).
func escribirFormulario(f Formulario) error {
	ruta, err := rutaFormulario(f.ID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	// Escritura atómica: un corte de luz non pode deixar o formulario a medias.
	tmp := ruta + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, ruta)
}

// cambiarModoFormulario pasa un formulario da biblioteca a deseño ou a produción.
func cambiarModoFormulario(id, modo string) (Formulario, error) {
	if modo != modoDeseno && modo != modoProduccion {
		return Formulario{}, fmt.Errorf("modo descoñecido %q (usa %q ou %q)", modo, modoDeseno, modoProduccion)
	}
	f, err := lerFormulario(id)
	if err != nil {
		return Formulario{}, err
	}
	f.Modo = modo
	return f, escribirFormulario(f)
}

// eliminarFormulario borra un formulario da biblioteca (non os datos que o
// usuario gardase noutros ficheiros).
func eliminarFormulario(id string) error {
	ruta, err := rutaFormulario(id)
	if err != nil {
		return err
	}
	// Os da zona fixa da columna esquerda non se poden borrar (só ocultar ou quitar da zona).
	if estaFixadoNaBiblioteca(id) {
		return errors.New("este formulario está na zona fixa da columna esquerda: quítao de alí antes de borralo (ou ocúltao)")
	}
	if err := os.Remove(ruta); err != nil && !os.IsNotExist(err) {
		return err
	}
	// O traballo en curso dese formulario (o borrador) vai con el.
	if bor, err := borradorFerramentaPath("form:" + id); err == nil {
		_ = os.Remove(bor)
	}
	quitarDaBiblioteca(id)
	return nil
}

// listarFormularios devolve os formularios da biblioteca ordenados por nome. Un
// ficheiro estragado non impide ver os demais: sáltase.
func listarFormularios() ([]Formulario, error) {
	dir, err := dirFormularios()
	if err != nil {
		return nil, err
	}
	entradas, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Formulario
	for _, e := range entradas {
		nome := e.Name()
		if e.IsDir() || !strings.HasSuffix(nome, ".json") {
			continue
		}
		f, err := lerFormulario(strings.TrimSuffix(nome, ".json"))
		if err != nil {
			continue
		}
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := strings.ToLower(out[i].Nome), strings.ToLower(out[j].Nome)
		if a != b {
			return a < b
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}
