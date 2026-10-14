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
	"regexp"
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
	// BovedaAutomatica: formularios que se engaden á bóveda en cada Xerar
	// (boveda_traballos.go). Por defecto, ningún.
	BovedaAutomatica []string `json:"bovedaAutomatica,omitempty"`
	// Cor: a que escolleu o usuario para o proxecto (#rrggbb). Baleira = a de
	// por defecto, que o frontend reparte entre os proxectos.
	Cor string `json:"cor,omitempty"`
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
	// Ruta: o cartafol do proxecto. Ubicacion: o cartafol que escolleu o
	// usuario para gardalo ("" = o cartafol de Lúa).
	Ruta      string `json:"ruta"`
	Ubicacion string `json:"ubicacion,omitempty"`
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

// dirProxecto devolve o cartafol dun proxecto (sen mirar se existe): o que
// escolleu o usuario ou, se non escolleu ningún, <proxectos>/<id>.
func dirProxecto(id string) (string, error) {
	if !idFormularioRe.MatchString(id) {
		return "", fmt.Errorf("identificador de proxecto non válido: %q", id)
	}
	if r, ok := lerUbicacions()[id]; ok {
		return r, nil
	}
	base, err := dirProxectos()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, id), nil
}

// idsProxectos: os cartafoles de <proxectos> e os proxectos gardados fóra.
// Non mira se son proxectos de verdade (iso faino lerProxecto).
func idsProxectos() []string {
	vistos := map[string]bool{}
	if base, err := dirProxectos(); err == nil {
		ents, _ := os.ReadDir(base)
		for _, e := range ents {
			if e.IsDir() {
				vistos[e.Name()] = true
			}
		}
	}
	for id := range lerUbicacions() {
		vistos[id] = true
	}
	return ordenados(vistos)
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
	if !corProxectoRe.MatchString(p.Cor) {
		p.Cor = "" // unha cor mala (a man, nun paquete) vale a de por defecto
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
	existentes := idsProxectos()
	if len(existentes) >= maxProxectos {
		return Proxecto{}, fmt.Errorf("hai máis de %d proxectos", maxProxectos)
	}
	usados := map[string]bool{}
	for _, id := range existentes {
		usados[id] = true
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
	if _, err := dirProxectos(); err != nil {
		return nil, err
	}
	out := []ProxectoInfo{}
	activo := idProxectoActivo()
	for _, id := range idsProxectos() {
		p, err := lerProxecto(id)
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
	i.Ruta, _ = dirProxecto(p.ID)
	if _, fora := lerUbicacions()[p.ID]; fora {
		i.Ubicacion = filepath.Dir(i.Ruta)
	}
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
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	ubicacionsMu.Lock()
	defer ubicacionsMu.Unlock()
	u := lerUbicacions()
	if _, ok := u[id]; !ok {
		return nil
	}
	delete(u, id)
	return gardarUbicacions(u)
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

var corProxectoRe = regexp.MustCompile(`^#[0-9a-f]{6}$`)

// fixarCorProxecto garda a cor do proxecto (#rrggbb; "" = a de por defecto).
func fixarCorProxecto(id, cor string) (Proxecto, error) {
	cor = strings.ToLower(strings.TrimSpace(cor))
	if cor != "" && !corProxectoRe.MatchString(cor) {
		return Proxecto{}, idioma.Errorf("go.proxecto.cor_mala", "a cor do proxecto ten que ser da forma #rrggbb (por exemplo, #3a7bd5)")
	}
	p, err := lerProxecto(id)
	if err != nil {
		return p, err
	}
	p.Cor = cor
	return p, escribirProxecto(p)
}

// ---------- Onde se garda cada proxecto ----------
// Un proxecto vive en <proxectos>/<id> ou, se o usuario escolleu outro
// cartafol do equipo, en <ese cartafol>/<Nome do proxecto>. Estes últimos
// anótanse en <proxectos>/ubicacions.json (id → ruta absoluta).

var ubicacionsMu sync.Mutex

func rutaUbicacions() (string, error) {
	base, err := dirProxectos()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "ubicacions.json"), nil
}

// lerUbicacions devolve os proxectos gardados fóra (nunca nil).
func lerUbicacions() map[string]string {
	u := map[string]string{}
	ruta, err := rutaUbicacions()
	if err != nil {
		return u
	}
	d, err := os.ReadFile(ruta)
	if err != nil || json.Unmarshal(d, &u) != nil || u == nil {
		return map[string]string{} // estragado: os proxectos de Lúa seguen víndose
	}
	for id, r := range u {
		if !idFormularioRe.MatchString(id) || !filepath.IsAbs(r) {
			delete(u, id)
		}
	}
	return u
}

func gardarUbicacions(u map[string]string) error {
	ruta, err := rutaUbicacions()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		return err
	}
	return escribirXSONAtomico(ruta, u)
}

// nomeCartafolProxecto: o nome do proxecto sen os caracteres que non admiten
// os sistemas de ficheiros.
func nomeCartafolProxecto(nome string) string {
	n := strings.Map(func(r rune) rune {
		if strings.ContainsRune(`/\:*?"<>|`, r) || r < 32 {
			return ' '
		}
		return r
	}, nome)
	n = strings.Trim(strings.Join(strings.Fields(n), " "), ". ")
	if n == "" {
		n = "proxecto"
	}
	return n
}

// moverProxecto leva os ficheiros dun proxecto a <destino>/<Nome do proxecto>
// (destino "" = o cartafol de Lúa). Un nome repetido vai coma «Nome (2)».
func moverProxecto(id, destino string) (Proxecto, error) {
	p, err := lerProxecto(id)
	if err != nil {
		return p, err
	}
	if idProxectoActivo() == id {
		return p, idioma.Errorf("go.proxecto.mover_aberto", "o proxecto está aberto: péchao antes de movelo")
	}
	base, err := dirProxectos()
	if err != nil {
		return p, err
	}
	orixe, _ := dirProxecto(id)
	destino = strings.TrimSpace(destino)
	if destino != "" {
		if !filepath.IsAbs(destino) {
			return p, idioma.Errorf("go.proxecto.ruta_relativa", "escolle un cartafol co selector: «%s» non é unha ruta completa", destino)
		}
		destino = filepath.Clean(destino)
		if st, err := os.Stat(destino); err != nil || !st.IsDir() {
			return p, idioma.Errorf("go.cartafol.non_existe", "non existe o cartafol «%s»", destino)
		}
		if dentroDe(orixe, destino) {
			return p, idioma.Errorf("go.proxecto.dentro_si", "non se pode gardar un proxecto dentro do seu propio cartafol")
		}
		if destino == filepath.Clean(base) {
			destino = ""
		}
	}
	novo := filepath.Join(base, id)
	if destino != "" {
		if filepath.Dir(orixe) == destino {
			return p, nil // xa está aí
		}
		nome := nomeCartafolProxecto(p.Nome)
		novo = filepath.Join(destino, nome)
		for n := 2; ; n++ {
			if _, err := os.Lstat(novo); os.IsNotExist(err) {
				break
			}
			novo = filepath.Join(destino, fmt.Sprintf("%s (%d)", nome, n))
		}
	}
	if novo == orixe {
		return p, nil
	}
	ubicacionsMu.Lock()
	defer ubicacionsMu.Unlock()
	if err := moverOuCopiar(orixe, novo); err != nil {
		_ = os.RemoveAll(novo)
		return p, err
	}
	u := lerUbicacions()
	if destino == "" {
		delete(u, id)
	} else {
		u[id] = novo
	}
	if err := gardarUbicacions(u); err != nil {
		// Que non quede perdido: se se copiou, o orixinal segue enteiro.
		if _, e := os.Stat(orixe); e == nil {
			_ = os.RemoveAll(novo)
		} else {
			_ = moverOuCopiar(novo, orixe)
		}
		return p, err
	}
	_ = os.RemoveAll(orixe) // se se copiou (outro disco), aínda está
	return p, nil
}
