package main

// Versións para DESFACER un traslado (spec trasladar): antes de actualizar un
// documento, restaurar unha copia completa ou migrar o traballo solto, Lúa
// copia o que vai cambiar a <config>/lua/versions/<id>/. Restaurar unha
// versión volve deixar cada ruta como estaba (e borra as que non existían).
// Gárdanse as maxVersions últimas; a máis vella bórrase ao crear outra.

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"time"
)

const maxVersions = 5

// dirConfigLua é <config>/lua (variable para os tests).
var dirConfigLua = func() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "lua"), nil
}

func dirVersions() (string, error) {
	d, err := dirConfigLua()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "versions"), nil
}

// rutaVersion é unha ruta gardada nunha versión.
type rutaVersion struct {
	Orixe  string `json:"orixe"`  // ruta absoluta orixinal
	Copia  string `json:"copia"`  // nome dentro da versión ("" se non existía)
	Existe bool   `json:"existe"` // existía cando se gardou
}

// metaVersion describe unha versión.
type metaVersion struct {
	ID         string        `json:"id"`
	Tipo       string        `json:"tipo"`
	Descricion string        `json:"descricion"`
	Creada     string        `json:"creada"`
	Rutas      []rutaVersion `json:"rutas"`
}

var idVersionRe = regexp.MustCompile(`^[0-9]{8}-[0-9]{6}-[0-9]+-[a-z]+$`)

// gardarVersion copia as rutas (ficheiros ou cartafoles) e devolve o id da
// versión. Unha ruta que non existe anótase para borrala ao desfacer.
func gardarVersion(tipo, descricion string, rutas []string) (string, error) {
	base, err := dirVersions()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(base, 0o755); err != nil {
		return "", err
	}
	agora := time.Now()
	id := ""
	// O contador son os nanosegundos (máis un se coinciden): sempre crece
	// dentro do mesmo segundo, aínda que se borrasen versións máis vellas.
	for n := agora.Nanosecond(); ; n++ {
		id = agora.Format("20060102-150405") + "-" + strconv.Itoa(n) + "-" + tipo
		if _, err := os.Stat(filepath.Join(base, id)); os.IsNotExist(err) {
			break
		}
	}
	dir := filepath.Join(base, id)
	if err := os.MkdirAll(filepath.Join(dir, "contido"), 0o755); err != nil {
		return "", err
	}
	meta := metaVersion{ID: id, Tipo: tipo, Descricion: descricion, Creada: agora.Format(time.RFC3339), Rutas: []rutaVersion{}}
	for i, r := range rutas {
		abs, err := filepath.Abs(r)
		if err != nil {
			_ = os.RemoveAll(dir)
			return "", err
		}
		rv := rutaVersion{Orixe: abs}
		if _, err := os.Lstat(abs); err == nil {
			rv.Existe = true
			rv.Copia = strconv.Itoa(i)
			if err := copiarArbore(abs, filepath.Join(dir, "contido", rv.Copia)); err != nil {
				_ = os.RemoveAll(dir)
				return "", err
			}
		}
		meta.Rutas = append(meta.Rutas, rv)
	}
	d, _ := json.MarshalIndent(meta, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "meta.json"), d, 0o644); err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}
	podarVersions(base)
	return id, nil
}

// podarVersions deixa só as maxVersions máis novas.
func podarVersions(base string) {
	ents, err := os.ReadDir(base)
	if err != nil {
		return
	}
	ids := []string{}
	for _, e := range ents {
		if e.IsDir() && idVersionRe.MatchString(e.Name()) {
			ids = append(ids, e.Name())
		}
	}
	sort.Slice(ids, func(i, j int) bool { return menorIDVersion(ids[i], ids[j]) })
	for len(ids) > maxVersions {
		_ = os.RemoveAll(filepath.Join(base, ids[0]))
		ids = ids[1:]
	}
}

// menorIDVersion ordena por data e despois polo contador (numérico).
func menorIDVersion(a, b string) bool {
	if a[:15] != b[:15] {
		return a[:15] < b[:15]
	}
	return contadorVersion(a) < contadorVersion(b)
}

func contadorVersion(id string) int {
	resto := id[16:]
	for i := 0; i < len(resto); i++ {
		if resto[i] == '-' {
			n, _ := strconv.Atoi(resto[:i])
			return n
		}
	}
	return 0
}

// restaurarVersion volve deixar as rutas da versión como estaban.
func restaurarVersion(id string) error {
	if !idVersionRe.MatchString(id) {
		return fmt.Errorf("versión non válida: %q", id)
	}
	base, err := dirVersions()
	if err != nil {
		return err
	}
	dir := filepath.Join(base, id)
	d, err := os.ReadFile(filepath.Join(dir, "meta.json"))
	if err != nil {
		return errors.New("esa versión xa non existe")
	}
	var meta metaVersion
	if err := json.Unmarshal(d, &meta); err != nil {
		return errors.New("a versión está estragada")
	}
	for _, r := range meta.Rutas {
		if !filepath.IsAbs(r.Orixe) {
			return errors.New("a versión está estragada")
		}
		if err := os.RemoveAll(r.Orixe); err != nil {
			return err
		}
		if r.Existe {
			if err := copiarArbore(filepath.Join(dir, "contido", filepath.Base(r.Copia)), r.Orixe); err != nil {
				return err
			}
		}
	}
	return nil
}

// copiarArbore copia un ficheiro ou un cartafol enteiro (sen seguir ligazóns).
func copiarArbore(orixe, destino string) error {
	st, err := os.Lstat(orixe)
	if err != nil {
		return err
	}
	if !st.IsDir() {
		return copiarFicheiro(orixe, destino, st.Mode())
	}
	return filepath.WalkDir(orixe, func(p string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(orixe, p)
		if err != nil {
			return err
		}
		dest := filepath.Join(destino, rel)
		switch {
		case e.IsDir():
			return os.MkdirAll(dest, 0o755)
		case e.Type()&fs.ModeSymlink != 0:
			return nil // as ligazóns non se copian
		default:
			info, err := e.Info()
			if err != nil {
				return err
			}
			return copiarFicheiro(p, dest, info.Mode())
		}
	})
}

func copiarFicheiro(orixe, destino string, modo fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(destino), 0o755); err != nil {
		return err
	}
	in, err := os.Open(orixe)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(destino, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, modo.Perm()|0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
