package main

// Proxectos: unha documentación concreta feita con formularios da biblioteca.
//
// A biblioteca (formulario.go, biblioteca_formularios.go) garda os formularios
// reutilizables. Un PROXECTO é outra cousa: elixe os que vai usar e leva a súa
// propia copia deles e dos datos que se enchen, de modo que:
//
//   - os datos son do proxecto e non da biblioteca (non se mesturan entre
//     proxectos nin co traballo solto);
//   - cambiar ou borrar un formulario da biblioteca non toca un proxecto xa
//     feito: segue podendo reimprimirse tal e como estaba;
//   - o proxecto pode ter a súa bóveda de coñecemento (módulo buxo).
//
// Cartafol dun proxecto (<config>/lua/proxectos/<id>/):
//
//	proxecto.json              nome, materia, ferramentas que usa
//	formularios/<id>.json      copia dos formularios usados
//	datos/<ferramenta>.json    os datos de cada ferramenta (o «borrador»)
//	biblioteca_formularios.json   a columna esquerda do proxecto
//	boveda/                    bóveda do proxecto (pdf/, raw/, wiki/, buxo.md)
//
// Cando hai un proxecto ACTIVO, o resto de Lua traballa «dentro» dil sen saber:
// dirFormularios e borradorFerramentaPath apuntan aos seus cartafoles. As
// ferramentas son «cronograma», «actas» (integradas: só gardan datos) e
// «form:<id>».

import (
	"edicion/idioma"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	maxNomeProxecto    = 80
	maxMateriaProxecto = 200
	maxProxectos       = 500
)

// Proxecto é a ficha dun proxecto.
type Proxecto struct {
	ID          string   `json:"id"`
	Nome        string   `json:"nome"`
	Materia     string   `json:"materia,omitempty"`
	Creado      string   `json:"creado"`
	Ferramentas []string `json:"ferramentas"`
	// BovedaDesconectada: o usuario desconectou a bóveda do proxecto do
	// asistente. Por defecto, se o proxecto ten bóveda, o asistente úsaa.
	BovedaDesconectada bool `json:"bovedaDesconectada,omitempty"`
	// Carpeta: id do cartafol que agrupa o proxecto por temática ("" = sen cartafol).
	Carpeta string `json:"carpeta,omitempty"`
}

// ProxectoInfo é o que ve a lista de proxectos: a ficha e o estado da bóveda.
type ProxectoInfo struct {
	Proxecto
	Activo bool `json:"activo"`
	// Boveda: hai bóveda (cartafol boveda/ con raw/ e wiki/); Fontes e Wiki, os
	// .md de cada capa.
	Boveda bool `json:"boveda"`
	Fontes int  `json:"fontes"`
	Wiki   int  `json:"wiki"`
}

// dirProxectos é o cartafol de todos os proxectos (variable para os tests).
var dirProxectos = func() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "lua", "proxectos"), nil
}

var (
	proxectoActivoMu sync.RWMutex
	proxectoActivoID string
)

func idProxectoActivo() string {
	proxectoActivoMu.RLock()
	defer proxectoActivoMu.RUnlock()
	return proxectoActivoID
}

// dirProxecto devolve o cartafol dun proxecto (sen mirar se existe).
func dirProxecto(id string) (string, error) {
	if !idFormularioRe.MatchString(id) {
		return "", fmt.Errorf("identificador de proxecto non válido: %q", id)
	}
	base, err := dirProxectos()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, id), nil
}

// dirProxectoActivo devolve o cartafol do proxecto activo ("" se non hai).
func dirProxectoActivo() string {
	id := idProxectoActivo()
	if id == "" {
		return ""
	}
	d, err := dirProxecto(id)
	if err != nil {
		return ""
	}
	return d
}

// ferramentaProxectoValida: «cronograma», «actas» ou «form:<id>».
func ferramentaProxectoValida(f string) bool {
	if f == ferramentaCronograma || f == ferramentaActas {
		return true
	}
	_, ok := idFerramentaFormulario(f)
	return ok
}

func lerProxecto(id string) (Proxecto, error) {
	dir, err := dirProxecto(id)
	if err != nil {
		return Proxecto{}, err
	}
	d, err := os.ReadFile(filepath.Join(dir, "proxecto.json"))
	if err != nil {
		return Proxecto{}, fmt.Errorf("non existe o proxecto «%s»", id)
	}
	var p Proxecto
	if err := json.Unmarshal(d, &p); err != nil {
		return Proxecto{}, fmt.Errorf("a ficha do proxecto «%s» está estragada", id)
	}
	p.ID = id
	if p.Ferramentas == nil {
		p.Ferramentas = []string{}
	}
	return p, nil
}

func escribirProxecto(p Proxecto) error {
	dir, err := dirProxecto(p.ID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	d, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(dir, "proxecto.json.tmp")
	if err := os.WriteFile(tmp, d, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, "proxecto.json"))
}

func limparTexto(s string, max int) string {
	s = strings.TrimSpace(strings.Join(strings.Fields(s), " "))
	if r := []rune(s); len(r) > max {
		s = string(r[:max])
	}
	return s
}

// copiarFormularioAoProxecto copia un formulario da BIBLIOTECA (non do contexto
// activo) ao cartafol de formularios do proxecto.
func copiarFormularioAoProxecto(proxecto, fid string) error {
	if !idFormularioRe.MatchString(fid) {
		return fmt.Errorf("identificador de formulario non válido: %q", fid)
	}
	lib, err := dirBibliotecaFormularios()
	if err != nil {
		return err
	}
	d, err := os.ReadFile(filepath.Join(lib, fid+".json"))
	if err != nil {
		return fmt.Errorf("o formulario «%s» non está na biblioteca", fid)
	}
	var f Formulario
	if err := json.Unmarshal(d, &f); err != nil || f.ID != fid {
		return fmt.Errorf("o formulario «%s» da biblioteca está estragado", fid)
	}
	dir, err := dirProxecto(proxecto)
	if err != nil {
		return err
	}
	dest := filepath.Join(dir, "formularios")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dest, fid+".json"), d, 0o644)
}

// crearProxecto crea un proxecto cos formularios (e ferramentas integradas)
// escollidos. Os formularios cópianse da biblioteca; os datos empezan baleiros.
func crearProxecto(nome, materia string, ferramentas []string) (Proxecto, error) {
	nome = limparTexto(nome, maxNomeProxecto)
	if nome == "" {
		return Proxecto{}, errors.New("o proxecto necesita un nome")
	}
	base, err := dirProxectos()
	if err != nil {
		return Proxecto{}, err
	}
	existentes, _ := os.ReadDir(base)
	if len(existentes) >= maxProxectos {
		return Proxecto{}, fmt.Errorf("hai máis de %d proxectos", maxProxectos)
	}
	usados := map[string]bool{}
	for _, e := range existentes {
		usados[e.Name()] = true
	}
	slug := slugFormulario(nome)
	if slug == "" {
		slug = "proxecto"
	}
	p := Proxecto{
		ID: idLibre(slug, usados), Nome: nome, Materia: limparTexto(materia, maxMateriaProxecto),
		Creado: time.Now().Format("2006-01-02"), Ferramentas: []string{},
	}
	vistas := map[string]bool{}
	for _, f := range ferramentas {
		if !ferramentaProxectoValida(f) {
			return Proxecto{}, fmt.Errorf("ferramenta non válida: %q", f)
		}
		if !vistas[f] {
			vistas[f] = true
			p.Ferramentas = append(p.Ferramentas, f)
		}
	}
	// Primeiro as copias e ao final a ficha: un proxecto a medias non aparece.
	for _, f := range p.Ferramentas {
		if fid, ok := idFerramentaFormulario(f); ok {
			if err := copiarFormularioAoProxecto(p.ID, fid); err != nil {
				dir, _ := dirProxecto(p.ID)
				_ = os.RemoveAll(dir)
				return Proxecto{}, err
			}
		}
	}
	if err := escribirProxecto(p); err != nil {
		return Proxecto{}, err
	}
	return p, nil
}

// engadirAoProxecto engade unha ferramenta a un proxecto (copiando o formulario).
func engadirAoProxecto(id, ferramenta string) (Proxecto, error) {
	p, err := lerProxecto(id)
	if err != nil {
		return p, err
	}
	if !ferramentaProxectoValida(ferramenta) {
		return p, fmt.Errorf("ferramenta non válida: %q", ferramenta)
	}
	if contemString(p.Ferramentas, ferramenta) {
		return p, nil
	}
	if fid, ok := idFerramentaFormulario(ferramenta); ok {
		if err := copiarFormularioAoProxecto(id, fid); err != nil {
			return p, err
		}
	}
	p.Ferramentas = append(p.Ferramentas, ferramenta)
	if err := escribirProxecto(p); err != nil {
		return p, err
	}
	// A columna do proxecto amosa a nova ferramenta.
	if idProxectoActivo() == id {
		if b, err := lerBiblioteca(); err == nil {
			b.Agochados = quitarString(b.Agochados, ferramenta)
			if !contemString(b.Fixos, ferramenta) {
				b.Fixos = append(b.Fixos, ferramenta)
			}
			_ = escribirBiblioteca(b)
		}
	}
	return p, nil
}

func quitarString(l []string, s string) []string {
	res := []string{}
	for _, x := range l {
		if x != s {
			res = append(res, x)
		}
	}
	return res
}

// quitarDoProxecto saca unha ferramenta do proxecto e borra a súa copia e os
// seus datos.
func quitarDoProxecto(id, ferramenta string) (Proxecto, error) {
	p, err := lerProxecto(id)
	if err != nil {
		return p, err
	}
	if !contemString(p.Ferramentas, ferramenta) {
		return p, nil
	}
	dir, err := dirProxecto(id)
	if err != nil {
		return p, err
	}
	p.Ferramentas = quitarString(p.Ferramentas, ferramenta)
	if err := escribirProxecto(p); err != nil {
		return p, err
	}
	if fid, ok := idFerramentaFormulario(ferramenta); ok {
		_ = os.Remove(filepath.Join(dir, "formularios", fid+".json"))
	}
	_ = os.Remove(filepath.Join(dir, "datos", borradorFerramentaNome(ferramenta)))
	if idProxectoActivo() == id {
		if b, err := lerBiblioteca(); err == nil {
			b.Fixos, b.Agochados = quitarString(b.Fixos, ferramenta), quitarString(b.Agochados, ferramenta)
			_ = escribirBiblioteca(b)
		}
	}
	return p, nil
}

func listarProxectos() ([]ProxectoInfo, error) {
	base, err := dirProxectos()
	if err != nil {
		return nil, err
	}
	ents, err := os.ReadDir(base)
	if os.IsNotExist(err) {
		return []ProxectoInfo{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := []ProxectoInfo{}
	activo := idProxectoActivo()
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		p, err := lerProxecto(e.Name())
		if err != nil {
			continue // un proxecto estragado non impide ver os demais
		}
		out = append(out, infoProxecto(p, activo == p.ID))
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Nome) < strings.ToLower(out[j].Nome)
	})
	return out, nil
}

func infoProxecto(p Proxecto, activo bool) ProxectoInfo {
	i := ProxectoInfo{Proxecto: p, Activo: activo}
	if raiz := bovedaDoProxecto(p.ID); raiz != "" {
		i.Boveda = true
		for _, f := range (boveda{raiz}).inventario() {
			switch {
			case strings.HasPrefix(f.Rel, "raw/") && f.eMD():
				i.Fontes++
			case strings.HasPrefix(f.Rel, "wiki/") && f.eMD():
				i.Wiki++
			}
		}
	}
	return i
}

// rutaBovedaProxecto é onde vive (ou vivirá) a bóveda do proxecto.
func rutaBovedaProxecto(id string) (string, error) {
	d, err := dirProxecto(id)
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "boveda"), nil
}

// bovedaDoProxecto devolve a ruta da bóveda do proxecto se está construída
// (con raw/ e wiki/), ou "".
func bovedaDoProxecto(id string) string {
	r, err := rutaBovedaProxecto(id)
	if err != nil {
		return ""
	}
	if _, err := validarRaizBoveda(r); err != nil {
		return ""
	}
	return r
}

func eliminarProxecto(id string) error {
	dir, err := dirProxecto(id)
	if err != nil {
		return err
	}
	if _, err := lerProxecto(id); err != nil {
		return err
	}
	if idProxectoActivo() == id {
		return errors.New("o proxecto está aberto: péchao antes de borralo")
	}
	return os.RemoveAll(dir)
}

// activarProxecto fai que Lua traballe dentro do proxecto ("" = fóra de
// calquera). Quen chama ten que volcar antes os borradores en curso: dende
// aquí os datos van a outro sitio.
func activarProxecto(id string) (Proxecto, error) {
	var p Proxecto
	if id != "" {
		var err error
		if p, err = lerProxecto(id); err != nil {
			return p, err
		}
	}
	proxectoActivoMu.Lock()
	proxectoActivoID = id
	proxectoActivoMu.Unlock()
	return p, nil
}

// bovedaDesconectadaDoProxecto di se o proxecto ten a bóveda desconectada.
func bovedaDesconectadaDoProxecto(id string) bool {
	p, err := lerProxecto(id)
	return err == nil && p.BovedaDesconectada
}

// desconectarBovedaDoProxecto desconecta (ou volve conectar) a bóveda do
// proxecto do asistente. Non borra nada.
func desconectarBovedaDoProxecto(id string, desconectar bool) error {
	p, err := lerProxecto(id)
	if err != nil {
		return err
	}
	p.BovedaDesconectada = desconectar
	return escribirProxecto(p)
}

// ---------- Cartafoles de proxectos ----------
// Agrupan os proxectos por temática. Gárdanse en <proxectos>/carpetas.json; un
// proxecto só garda o id do seu cartafol. Borrar un cartafol NUNCA borra
// proxectos: pasan a «sen cartafol».

const (
	maxCarpetasProxectos   = 100
	maxNomeCarpetaProxecto = 60
)

// CarpetaProxectos é un cartafol de proxectos.
type CarpetaProxectos struct {
	ID   string `json:"id"`
	Nome string `json:"nome"`
}

func rutaCarpetasProxectos() (string, error) {
	base, err := dirProxectos()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "carpetas.json"), nil
}

func lerCarpetasProxectos() ([]CarpetaProxectos, error) {
	ruta, err := rutaCarpetasProxectos()
	if err != nil {
		return nil, err
	}
	d, err := os.ReadFile(ruta)
	if err != nil {
		return []CarpetaProxectos{}, nil
	}
	var c []CarpetaProxectos
	if json.Unmarshal(d, &c) != nil {
		return []CarpetaProxectos{}, nil // estragado: non impide ver os proxectos
	}
	if c == nil {
		c = []CarpetaProxectos{}
	}
	return c, nil
}

func gardarCarpetasProxectos(c []CarpetaProxectos) error {
	ruta, err := rutaCarpetasProxectos()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		return err
	}
	d, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := ruta + ".tmp"
	if err := os.WriteFile(tmp, d, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, ruta)
}

func nomeCarpetaRepetido(c []CarpetaProxectos, nome, excepto string) bool {
	for _, x := range c {
		if x.ID != excepto && strings.EqualFold(x.Nome, nome) {
			return true
		}
	}
	return false
}

func crearCarpetaProxectos(nome string) ([]CarpetaProxectos, error) {
	nome = limparTexto(nome, maxNomeCarpetaProxecto)
	if nome == "" {
		return nil, idioma.Errorf("go.cartafol.sen_nome", "o cartafol necesita un nome")
	}
	c, _ := lerCarpetasProxectos()
	if len(c) >= maxCarpetasProxectos {
		return c, fmt.Errorf("hai máis de %d cartafoles", maxCarpetasProxectos)
	}
	if nomeCarpetaRepetido(c, nome, "") {
		return c, idioma.Errorf("go.cartafol.repetido", "xa hai un cartafol chamado «%s»", nome)
	}
	usados := map[string]bool{}
	for _, x := range c {
		usados[x.ID] = true
	}
	base := slugFormulario(nome)
	if base == "" {
		base = "cartafol"
	}
	c = append(c, CarpetaProxectos{ID: idLibre(base, usados), Nome: nome})
	return c, gardarCarpetasProxectos(c)
}

func renomearCarpetaProxectos(id, nome string) ([]CarpetaProxectos, error) {
	nome = limparTexto(nome, maxNomeCarpetaProxecto)
	if nome == "" {
		return nil, idioma.Errorf("go.cartafol.sen_nome", "o cartafol necesita un nome")
	}
	c, _ := lerCarpetasProxectos()
	if nomeCarpetaRepetido(c, nome, id) {
		return c, idioma.Errorf("go.cartafol.repetido", "xa hai un cartafol chamado «%s»", nome)
	}
	for i := range c {
		if c[i].ID == id {
			c[i].Nome = nome
			return c, gardarCarpetasProxectos(c)
		}
	}
	return c, fmt.Errorf("non existe o cartafol «%s»", id)
}

// eliminarCarpetaProxectos borra o cartafol; os seus proxectos pasan a «sen cartafol».
func eliminarCarpetaProxectos(id string) ([]CarpetaProxectos, error) {
	c, _ := lerCarpetasProxectos()
	novo := []CarpetaProxectos{}
	atopado := false
	for _, x := range c {
		if x.ID == id {
			atopado = true
			continue
		}
		novo = append(novo, x)
	}
	if !atopado {
		return c, fmt.Errorf("non existe o cartafol «%s»", id)
	}
	if err := gardarCarpetasProxectos(novo); err != nil {
		return c, err
	}
	base, _ := dirProxectos()
	ents, _ := os.ReadDir(base)
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		if p, err := lerProxecto(e.Name()); err == nil && p.Carpeta == id {
			p.Carpeta = ""
			_ = escribirProxecto(p)
		}
	}
	return novo, nil
}

// moverProxecto pon un proxecto nun cartafol ("" = sen cartafol).
func moverProxecto(id, carpeta string) (Proxecto, error) {
	p, err := lerProxecto(id)
	if err != nil {
		return p, err
	}
	if carpeta != "" {
		c, _ := lerCarpetasProxectos()
		ok := false
		for _, x := range c {
			ok = ok || x.ID == carpeta
		}
		if !ok {
			return p, fmt.Errorf("non existe o cartafol «%s»", carpeta)
		}
	}
	p.Carpeta = carpeta
	return p, escribirProxecto(p)
}
