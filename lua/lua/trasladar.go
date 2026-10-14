package main

// TRASLADAR (spec trasladar): un só sitio para sacar cousas de Lúa e metelas.
//
//   - Sacar: un paquete .lua (zip con manifest.json) que leva calquera
//     combinación de proxectos (formularios, datos e bóveda), formularios da
//     biblioteca e plantillas; ou, para un só documento, un .json editable.
//   - Meter: Lúa le o ficheiro, di que leva e onde vai (clasificarMeter) e só
//     despois o aplica (trasladar_meter.go). Nunca se pisa nada en silencio.
//
// Este ficheiro ten os tipos, o documento .json e a clasificación. O zip está
// en trasladar_paquete.go e a importación en trasladar_meter.go.

import (
	"bytes"
	"crypto/rand"
	"edicion/idioma"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	formatoPaquete     = "lua-paquete"
	versionPaquete     = 1
	extensionPaquete   = ".lua"
	formatoDocumento   = "lua-documento"
	versionDocumento   = 1
	maxBytesJSONMeter  = 200 << 20
	maxFicheirosBoveda = 5000
)

// SeleccionSacar é o que o usuario escolle en «Sacar».
type SeleccionSacar struct {
	// Tipo: "proxecto" (os Proxectos indicados), "todo" (copia de seguridade
	// completa) ou "seleccion" (o escollido na árbore).
	Tipo        string   `json:"tipo"`
	Proxectos   []string `json:"proxectos"`
	Formularios []string `json:"formularios"`
	Plantillas  []string `json:"plantillas"`
	// Datos: van os datos escritos dos proxectos (na copia completa sempre).
	Datos bool `json:"datos"`
}

// ItemPaquete é unha entrada do manifesto.
type ItemPaquete struct {
	ID   string `json:"id"`
	Nome string `json:"nome"`
}

// ManifestoPaquete é o manifest.json dun paquete .lua.
type ManifestoPaquete struct {
	Formato     string        `json:"formato"`
	Version     int           `json:"version"`
	Creado      string        `json:"creado"`
	Instalacion string        `json:"instalacion,omitempty"`
	Completa    bool          `json:"completa,omitempty"`
	Datos       bool          `json:"datos"`
	Proxectos   []ItemPaquete `json:"proxectos"`
	Formularios []ItemPaquete `json:"formularios"`
	Plantillas  []ItemPaquete `json:"plantillas"`
	// PlantillaDocumento: «<proxecto>/<ferramenta>» → id (no paquete) da
	// plantilla que usa ese documento.
	PlantillaDocumento map[string]string `json:"plantillaDocumento,omitempty"`
}

// OrixeDocumento di de onde saíu un documento .json (para recoñecer a volta).
type OrixeDocumento struct {
	Instalacion string `json:"instalacion"`
	Proxecto    string `json:"proxecto,omitempty"`
	Ferramenta  string `json:"ferramenta"`
	Exportado   string `json:"exportado,omitempty"`
}

// DocumentoTraslado é o .json editable dun documento: autosuficiente (datos,
// estrutura do formulario e plantilla).
type DocumentoTraslado struct {
	Formato    string         `json:"formato"`
	Version    int            `json:"version"`
	Orixe      OrixeDocumento `json:"orixe"`
	Ferramenta string         `json:"ferramenta"`
	Nome       string         `json:"nome"`
	// Documento é o mesmo JSON ca o traballo en curso da ferramenta.
	Documento  json.RawMessage  `json:"documento"`
	Formulario *Formulario      `json:"formulario,omitempty"`
	Plantilla  *bundlePlantilla `json:"plantilla,omitempty"`
}

// ElementoMeter é unha liña do resumo de «Meter».
type ElementoMeter struct {
	Tipo    string `json:"tipo"` // proxecto | formulario | plantilla | documento | boveda
	Nome    string `json:"nome"`
	Detalle string `json:"detalle,omitempty"`
}

// VoltaMeter: o documento saíu deste Lúa e o seu documento orixinal segue aquí.
type VoltaMeter struct {
	Proxecto     string `json:"proxecto"`
	NomeProxecto string `json:"nomeProxecto"`
	Ferramenta   string `json:"ferramenta"`
}

// ResumoMeter é o que Lúa amosa antes de meter nada.
type ResumoMeter struct {
	// Tipo: paquete | copia | documento | curso | formulario | plantilla | boveda
	Tipo      string          `json:"tipo"`
	Nome      string          `json:"nome"`
	Elementos []ElementoMeter `json:"elementos"`
	// Ferramenta do documento ou curso («cronograma», «actas», «form:<id>»).
	Ferramenta string `json:"ferramenta,omitempty"`
	// Cursos: un por curso cando o ficheiro trae varios (JSON de cada un).
	Cursos []json.RawMessage `json:"cursos,omitempty"`
	// Destino por defecto: aberto | novo | biblioteca | boveda | restaurar | volta
	Destino string `json:"destino"`
	// PodeAberto: o documento pode ir ao proxecto aberto (non o ten aínda).
	PodeAberto         bool        `json:"podeAberto"`
	ProxectoAberto     string      `json:"proxectoAberto,omitempty"`
	NomeProxectoAberto string      `json:"nomeProxectoAberto,omitempty"`
	Volta              *VoltaMeter `json:"volta,omitempty"`
	// Produccion: trae datos que poden ser reais.
	Produccion bool     `json:"produccion"`
	ConDatos   bool     `json:"conDatos"`
	Avisos     []string `json:"avisos,omitempty"`
}

// ---------- Instalación ----------

// novaInstalacion xera un identificador aleatorio (non é segredo: só serve
// para recoñecer os documentos que saíron deste Lúa).
func novaInstalacion() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ---------- Biblioteca de formularios (sempre a biblioteca, non o proxecto) ----------

func lerFormularioBiblioteca(id string) (Formulario, error) {
	if !idFormularioRe.MatchString(id) {
		return Formulario{}, fmt.Errorf("identificador de formulario non válido: %q", id)
	}
	lib, err := dirBibliotecaFormularios()
	if err != nil {
		return Formulario{}, err
	}
	d, err := os.ReadFile(filepath.Join(lib, id+".json"))
	if err != nil {
		return Formulario{}, err
	}
	var f Formulario
	if err := json.Unmarshal(d, &f); err != nil {
		return Formulario{}, err
	}
	return f, nil
}

func formulariosBiblioteca() ([]Formulario, error) {
	lib, err := dirBibliotecaFormularios()
	if err != nil {
		return nil, err
	}
	ents, err := os.ReadDir(lib)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	out := []Formulario{}
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		if f, err := lerFormularioBiblioteca(strings.TrimSuffix(e.Name(), ".json")); err == nil {
			out = append(out, f)
		}
	}
	return out, nil
}

// mesmoFormulario compara a estrutura (sen o modo).
func mesmoFormulario(a, b Formulario) bool {
	a.Modo, b.Modo = "", ""
	da, _ := json.Marshal(normalizarFormulario(a))
	db, _ := json.Marshal(normalizarFormulario(b))
	return bytes.Equal(da, db)
}

// meterFormularioBiblioteca deixa o formulario na biblioteca: se xa hai un
// idéntico, reutilízao; se o id está ocupado por outro distinto, entra coma
// copia («Nome (2)»). Devolve o formulario tal como quedou.
func meterFormularioBiblioteca(f Formulario) (Formulario, bool, error) {
	f = normalizarFormulario(f)
	if err := validarFormulario(f); err != nil {
		return f, false, idioma.Errorf("go.formulario.non_valido", "o formulario do ficheiro non é válido: %w", err)
	}
	existentes, err := formulariosBiblioteca()
	if err != nil {
		return f, false, err
	}
	ids, nomes := map[string]bool{}, map[string]bool{}
	for _, e := range existentes {
		if mesmoFormulario(e, f) {
			return e, true, nil
		}
		ids[e.ID] = true
		nomes[strings.ToLower(e.Nome)] = true
	}
	if ids[f.ID] {
		f.ID = idLibre(f.ID, ids)
		f.Nome = nomeLibreFormulario(f.Nome, nomes)
	}
	f.Modo = modoDeseno
	lib, err := dirBibliotecaFormularios()
	if err != nil {
		return f, false, err
	}
	return f, false, escribirXSONAtomico(filepath.Join(lib, f.ID+".json"), f)
}

// escribirXSONAtomico garda v como JSON nun temporal e renomea.
func escribirXSONAtomico(ruta string, v any) error {
	d, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		return err
	}
	return escribirAtomicoCrono(ruta, d)
}

// ---------- Proxectos ----------

func idsEProxectos() (map[string]bool, map[string]bool) {
	ids, nomes := map[string]bool{}, map[string]bool{}
	for _, id := range idsProxectos() {
		ids[id] = true
		if p, err := lerProxecto(id); err == nil {
			nomes[strings.ToLower(p.Nome)] = true
		}
	}
	return ids, nomes
}

// proxectoTenDocumento: o proxecto usa a ferramenta e xa ten datos dela.
func proxectoTenDocumento(id, ferramenta string) bool {
	p, err := lerProxecto(id)
	if err != nil || !contemString(p.Ferramentas, ferramenta) {
		return false
	}
	dir, err := dirProxecto(id)
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(dir, "datos", borradorFerramentaNome(ferramenta)))
	return err == nil
}

// rutaDocumentoProxecto é o ficheiro do traballo en curso dun documento.
func rutaDocumentoProxecto(id, ferramenta string) (string, error) {
	dir, err := dirProxecto(id)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "datos", borradorFerramentaNome(ferramenta)), nil
}

// proxectoAbertoValido devolve o proxecto aberto se aínda existe ("" se non).
func proxectoAbertoValido() string {
	id := idProxectoActivo()
	if id == "" {
		return ""
	}
	if _, err := lerProxecto(id); err != nil {
		return ""
	}
	return id
}

// ---------- Plantillas ----------

// plantillaDeDocumento devolve a plantilla lembrada para un documento.
func plantillaDeDocumento(alm almacenPlantillas, rutaDoc string) *PlantillaLatex {
	id, ok := alm.PorFicheiro[rutaDoc]
	if !ok {
		return nil
	}
	for i := range alm.Plantillas {
		if alm.Plantillas[i].ID == id {
			return &alm.Plantillas[i]
		}
	}
	return nil
}

// lembrarPlantillaDocumento anota que un documento usa unha plantilla (sen
// cambiar a activa).
func lembrarPlantillaDocumento(rutaDoc, id string) error {
	alm, err := loadAlmacenPlantillas()
	if err != nil {
		return err
	}
	if alm.PorFicheiro == nil {
		alm.PorFicheiro = map[string]string{}
	}
	alm.PorFicheiro[rutaDoc] = id
	return saveAlmacenPlantillas(alm)
}

// ---------- Documento .json ----------

// construirDocumentoTraslado arma o .json editable do documento `ferramenta`
// do proxecto aberto. `contido` é o JSON do traballo en curso.
func construirDocumentoTraslado(ferramenta, contido, nome, instalacion string) (DocumentoTraslado, error) {
	var d DocumentoTraslado
	if !json.Valid([]byte(contido)) {
		return d, errors.New("o documento non é un JSON válido")
	}
	if ferramenta != ferramentaCronograma && ferramenta != ferramentaActas {
		if _, ok := idFerramentaFormulario(ferramenta); !ok {
			return d, fmt.Errorf("ferramenta non válida: %q", ferramenta)
		}
	}
	d = DocumentoTraslado{
		Formato: formatoDocumento, Version: versionDocumento, Ferramenta: ferramenta, Nome: strings.TrimSpace(nome),
		Orixe:     OrixeDocumento{Instalacion: instalacion, Proxecto: idProxectoActivo(), Ferramenta: ferramenta, Exportado: time.Now().Format(time.RFC3339)},
		Documento: json.RawMessage(sanearContidoDocumento([]byte(contido))),
	}
	if fid, ok := idFerramentaFormulario(ferramenta); ok {
		f, err := lerFormulario(fid)
		if err != nil {
			return d, err
		}
		f.Modo = ""
		d.Formulario = &f
		if d.Nome == "" {
			d.Nome = f.Nome
		}
	}
	if ruta, err := borradorFerramentaPath(ferramenta); err == nil {
		if alm, err := loadAlmacenPlantillas(); err == nil {
			if p := plantillaDeDocumento(alm, ruta); p != nil {
				if b, err := construirBundlePlantilla(*p); err == nil {
					d.Plantilla = &b
				}
			}
		}
	}
	return d, nil
}

// sanearContidoDocumento quita o campo «path» (o ficheiro vinculado das
// versións anteriores): un documento trasladado non apunta a ningún ficheiro.
func sanearContidoDocumento(contido []byte) []byte {
	var m map[string]json.RawMessage
	if json.Unmarshal(contido, &m) != nil {
		return contido
	}
	if _, ok := m["path"]; !ok {
		return contido
	}
	delete(m, "path")
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return contido
	}
	return out
}

// ---------- Clasificación ----------

// cabeceiraJSON son os campos que distinguen os formatos que Lúa le.
type cabeceiraJSON struct {
	Formato    string            `json:"formato"`
	Version    int               `json:"version"`
	Ferramenta string            `json:"ferramenta"`
	Formulario json.RawMessage   `json:"formulario"`
	Plans      []json.RawMessage `json:"plans"`
	Actas      []json.RawMessage `json:"actas"`
	Config     json.RawMessage   `json:"config"`
	MonthsData json.RawMessage   `json:"monthsData"`
	Datos      json.RawMessage   `json:"datos"`
}

var formatosPlantilla = map[string]bool{formatoBundlePlantilla: true, "yang-plantilla": true, "lua-plantilla": true}

// clasificarMeter le a ruta (ficheiro ou cartafol) e di que leva e onde iría.
// Non cambia nada.
func (a *App) clasificarMeter(ruta string) (ResumoMeter, error) {
	r := ResumoMeter{Elementos: []ElementoMeter{}}
	r.ProxectoAberto = proxectoAbertoValido()
	if r.ProxectoAberto != "" {
		if p, err := lerProxecto(r.ProxectoAberto); err == nil {
			r.NomeProxectoAberto = p.Nome
		}
	}
	st, err := os.Stat(ruta)
	if err != nil {
		return r, idioma.Errorf("go.trasladar.non_existe", "non se atopa «%s»", filepath.Base(ruta))
	}
	r.Nome = filepath.Base(ruta)
	if st.IsDir() {
		return clasificarBoveda(r, ruta)
	}
	ext := strings.ToLower(filepath.Ext(ruta))
	switch ext {
	case ".pdf", ".md", ".markdown":
		return clasificarBoveda(r, ruta)
	}
	if eZip(ruta) {
		return a.clasificarPaquete(r, ruta)
	}
	if st.Size() > maxBytesJSONMeter {
		return r, idioma.Errorf("go.ficheiro.grande", "o ficheiro é demasiado grande (máximo %d MB)", maxBytesJSONMeter>>20)
	}
	data, err := os.ReadFile(ruta)
	if err != nil {
		return r, err
	}
	return a.clasificarJSON(r, data)
}

func (a *App) clasificarJSON(r ResumoMeter, data []byte) (ResumoMeter, error) {
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	if !json.Valid(data) {
		return r, idioma.Errorf("go.trasladar.non_lua", "este ficheiro non é de Lúa nin un formato que Lúa saiba ler (ou está danado)")
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) > 0 && trimmed[0] == '[' {
		// Array: curso de cronograma da versión web de Piztu Orienta.
		var cursos []json.RawMessage
		_ = json.Unmarshal(trimmed, &cursos)
		return r.curso(ferramentaCronograma, cursos)
	}
	var c cabeceiraJSON
	if err := json.Unmarshal(data, &c); err != nil {
		return r, idioma.Errorf("go.trasladar.non_lua", "este ficheiro non é de Lúa nin un formato que Lúa saiba ler (ou está danado)")
	}
	switch {
	case c.Formato == formatoDocumento:
		return a.clasificarDocumento(r, data)
	case c.Formato == formatoBundleFormulario:
		return clasificarBundleFormulario(r, data)
	case formatosPlantilla[c.Formato]:
		var b bundlePlantilla
		if err := json.Unmarshal(data, &b); err != nil {
			return r, idioma.Errorf("go.trasladar.danado", "o ficheiro parece de Lúa pero está danado: %w", err)
		}
		r.Tipo, r.Destino = "plantilla", "biblioteca"
		r.Elementos = append(r.Elementos, ElementoMeter{Tipo: "plantilla", Nome: b.Plantilla.Nome})
		return r, nil
	case c.Formato != "":
		return r, idioma.Errorf("go.trasladar.formato_futuro", "o ficheiro é dun formato que este Lúa non coñece («%s»): actualiza Lúa", c.Formato)
	case c.Ferramenta == ferramentaActas:
		return r.curso(ferramentaActas, c.Actas)
	case c.Ferramenta == ferramentaFormulario:
		var fid string
		_ = json.Unmarshal(c.Formulario, &fid)
		return a.clasificarDocumentoFormularioVello(r, fid, data)
	case c.Plans != nil:
		return r.curso(ferramentaCronograma, c.Plans)
	case c.Config != nil || c.MonthsData != nil:
		return r.curso(ferramentaCronograma, []json.RawMessage{json.RawMessage(data)})
	}
	return r, idioma.Errorf("go.trasladar.non_lua", "este ficheiro non é de Lúa nin un formato que Lúa saiba ler (ou está danado)")
}

// curso: un .json vello de cronograma ou de actas (un ou varios cursos).
func (r ResumoMeter) curso(ferramenta string, cursos []json.RawMessage) (ResumoMeter, error) {
	if len(cursos) == 0 {
		return r, idioma.Errorf("go.trasladar.sen_cursos", "o ficheiro non contén ningún curso")
	}
	r.Tipo, r.Ferramenta = "curso", ferramenta
	r.Cursos = cursos
	r.ConDatos = true
	r.Elementos = append(r.Elementos, ElementoMeter{Tipo: "documento", Nome: ferramenta, Detalle: fmt.Sprint(len(cursos))})
	r.destinoDocumento(ferramenta)
	if len(cursos) > 1 {
		r.Destino, r.PodeAberto = "novo", false
	}
	return r, nil
}

// destinoDocumento aplica a regra: o proxecto aberto se aínda non ten ese
// documento; se non, un proxecto novo.
func (r *ResumoMeter) destinoDocumento(ferramenta string) {
	r.PodeAberto = r.ProxectoAberto != "" && !proxectoTenDocumento(r.ProxectoAberto, ferramenta)
	if r.PodeAberto {
		r.Destino = "aberto"
	} else {
		r.Destino = "novo"
	}
}

func (a *App) clasificarDocumento(r ResumoMeter, data []byte) (ResumoMeter, error) {
	var d DocumentoTraslado
	if err := json.Unmarshal(data, &d); err != nil {
		return r, idioma.Errorf("go.trasladar.danado", "o ficheiro parece de Lúa pero está danado: %w", err)
	}
	if d.Version < 1 || d.Version > versionDocumento {
		return r, idioma.Errorf("go.trasladar.version", "o ficheiro ten a versión %d do formato e este Lúa só entende ata a %d: actualiza Lúa", d.Version, versionDocumento)
	}
	if !json.Valid(d.Documento) {
		return r, idioma.Errorf("go.trasladar.danado", "o ficheiro parece de Lúa pero está danado: %w", errors.New("documento"))
	}
	r.Tipo, r.Ferramenta, r.ConDatos = "documento", d.Ferramenta, true
	if d.Nome != "" {
		r.Nome = d.Nome
	}
	r.Elementos = append(r.Elementos, ElementoMeter{Tipo: "documento", Nome: r.Nome})
	if fid, ok := idFerramentaFormulario(d.Ferramenta); ok {
		if d.Formulario == nil {
			if _, err := lerFormularioBiblioteca(fid); err != nil {
				return r, idioma.Errorf("go.trasladar.sen_formulario", "este documento é dun formulario que non tes e o ficheiro non o trae")
			}
		}
		r.Produccion = !datosSonDeProba(d.Documento)
	} else if d.Ferramenta != ferramentaCronograma && d.Ferramenta != ferramentaActas {
		return r, fmt.Errorf("ferramenta non válida: %q", d.Ferramenta)
	}
	if d.Plantilla != nil {
		r.Elementos = append(r.Elementos, ElementoMeter{Tipo: "plantilla", Nome: d.Plantilla.Plantilla.Nome})
	}
	r.destinoDocumento(d.Ferramenta)
	// A volta: saíu deste Lúa e o documento orixinal segue aquí.
	if d.Orixe.Instalacion != "" && d.Orixe.Instalacion == a.settings.Instalacion && d.Orixe.Proxecto != "" &&
		d.Orixe.Ferramenta == d.Ferramenta && proxectoTenDocumento(d.Orixe.Proxecto, d.Ferramenta) {
		p, _ := lerProxecto(d.Orixe.Proxecto)
		r.Volta = &VoltaMeter{Proxecto: p.ID, NomeProxecto: p.Nome, Ferramenta: d.Ferramenta}
		r.Destino = "volta"
	}
	return r, nil
}

// datosSonDeProba: o documento dun formulario marca os seus datos coma de proba.
func datosSonDeProba(doc []byte) bool {
	var d struct {
		Datos struct {
			Proba bool `json:"proba"`
		} `json:"datos"`
	}
	_ = json.Unmarshal(doc, &d)
	return d.Datos.Proba
}

func (a *App) clasificarDocumentoFormularioVello(r ResumoMeter, fid string, data []byte) (ResumoMeter, error) {
	if !idFormularioRe.MatchString(fid) {
		return r, idioma.Errorf("go.trasladar.non_lua", "este ficheiro non é de Lúa nin un formato que Lúa saiba ler (ou está danado)")
	}
	ferr := idFerramentaForm(fid)
	_, errLib := lerFormularioBiblioteca(fid)
	enAberto := false
	if r.ProxectoAberto != "" {
		if dir, err := dirProxecto(r.ProxectoAberto); err == nil {
			_, e := os.Stat(filepath.Join(dir, "formularios", fid+".json"))
			enAberto = e == nil
		}
	}
	if errLib != nil && !enAberto {
		return r, idioma.Errorf("go.trasladar.sen_formulario", "este documento é dun formulario que non tes e o ficheiro non o trae")
	}
	r.Tipo, r.Ferramenta, r.ConDatos = "documento", ferr, true
	r.Produccion = !datosSonDeProba(data)
	r.Elementos = append(r.Elementos, ElementoMeter{Tipo: "documento", Nome: r.Nome})
	r.destinoDocumento(ferr)
	return r, nil
}

func clasificarBundleFormulario(r ResumoMeter, data []byte) (ResumoMeter, error) {
	var b BundleFormulario
	if err := json.Unmarshal(data, &b); err != nil {
		return r, idioma.Errorf("go.trasladar.danado", "o ficheiro parece de Lúa pero está danado: %w", err)
	}
	if b.Version < 1 || b.Version > versionBundleFormulario {
		return r, idioma.Errorf("go.formulario.version", "o ficheiro ten a versión %d do formato e este Lúa só entende ata a %d: actualiza Lúa", b.Version, versionBundleFormulario)
	}
	r.Tipo, r.Nome = "formulario", b.Formulario.Nome
	r.Elementos = append(r.Elementos, ElementoMeter{Tipo: "formulario", Nome: b.Formulario.Nome})
	if haiDatosIntercambio(b.Datos) {
		r.ConDatos, r.Produccion, r.Destino = true, !b.Datos.Proba, "novo"
	} else {
		r.Destino = "biblioteca"
	}
	return r, nil
}

// clasificarBoveda: PDF, Markdown ou un cartafol (bóveda ou fontes soltas).
func clasificarBoveda(r ResumoMeter, ruta string) (ResumoMeter, error) {
	pdfs, mds, err := contarFontesBoveda(ruta)
	if err != nil {
		return r, err
	}
	if pdfs+mds == 0 {
		return r, idioma.Errorf("go.trasladar.baleiro", "en «%s» non hai nada que Lúa poida meter (PDF, Markdown ou unha bóveda)", filepath.Base(ruta))
	}
	r.Tipo, r.Destino = "boveda", "boveda"
	r.Elementos = append(r.Elementos, ElementoMeter{Tipo: "boveda", Nome: r.Nome, Detalle: fmt.Sprintf("%d/%d", pdfs, mds)})
	if r.ProxectoAberto == "" {
		r.Destino = "novo"
	}
	r.PodeAberto = r.ProxectoAberto != ""
	return r, nil
}

func eFonteBoveda(nome string) (pdf, md bool) {
	switch strings.ToLower(filepath.Ext(nome)) {
	case ".pdf":
		return true, false
	case ".md", ".markdown":
		return false, true
	}
	return false, false
}

func contarFontesBoveda(ruta string) (pdfs, mds int, err error) {
	st, err := os.Stat(ruta)
	if err != nil {
		return 0, 0, err
	}
	if !st.IsDir() {
		p, m := eFonteBoveda(ruta)
		if p {
			pdfs = 1
		}
		if m {
			mds = 1
		}
		return pdfs, mds, nil
	}
	n := 0
	err = filepath.WalkDir(ruta, func(p string, e fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if e.IsDir() {
			if strings.HasPrefix(e.Name(), ".") && p != ruta {
				return filepath.SkipDir
			}
			return nil
		}
		if n++; n > maxFicheirosBoveda {
			return idioma.Errorf("go.trasladar.moitos_cartafol", "o cartafol ten demasiados ficheiros (máximo %d)", maxFicheirosBoveda)
		}
		pp, mm := eFonteBoveda(e.Name())
		if pp {
			pdfs++
		}
		if mm {
			mds++
		}
		return nil
	})
	return pdfs, mds, err
}

// nomeProxectoLibre devolve `nome` ou «nome (2)»... se xa hai un proxecto así.
func nomeProxectoLibre(nome string) string {
	_, nomes := idsEProxectos()
	return nomeLibreFormulario(limparTexto(nome, maxNomeProxecto), nomes)
}

// ordenados devolve as claves dun conxunto en orde.
func ordenados(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// tenDatosProducion: o que se vai sacar leva datos de formularios en
// produción (poden ser reais e saen do equipo).
func tenDatosProducion(sel SeleccionSacar) bool {
	sel, err := resolverSeleccion(sel)
	if err != nil || !sel.Datos {
		return false
	}
	for _, id := range sel.Proxectos {
		p, err := lerProxecto(id)
		if err != nil {
			continue
		}
		dir, _ := dirProxecto(id)
		for _, f := range p.Ferramentas {
			fid, ok := idFerramentaFormulario(f)
			if !ok {
				continue
			}
			raw, err := os.ReadFile(filepath.Join(dir, "formularios", fid+".json"))
			if err != nil {
				continue
			}
			var copia Formulario
			if json.Unmarshal(raw, &copia) != nil || !copia.enProduccion() {
				continue
			}
			if _, err := os.Stat(filepath.Join(dir, "datos", borradorFerramentaNome(f))); err == nil {
				return true
			}
		}
	}
	return false
}
