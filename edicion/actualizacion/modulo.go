package actualizacion

// Instalación e actualización dun MÓDULO descargado (coma buxo): a diferenza
// da autoactualización dunha aplicación, aquí non se substitúe o executable en
// execución senón que se instala un cartafol enteiro (executable, conversor,
// modulo.json...) onde a aplicación anfitrioa vai buscalo.

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// VersionDeModulo le o campo "version" do modulo.json do cartafol do módulo
// ("" se non hai módulo instalado ou o ficheiro non vale).
func VersionDeModulo(dirModulo string) string {
	d, err := os.ReadFile(filepath.Join(dirModulo, "modulo.json"))
	if err != nil {
		return ""
	}
	var m struct {
		Version string `json:"version"`
	}
	if json.Unmarshal(d, &m) != nil {
		return ""
	}
	return strings.TrimSpace(m.Version)
}

// extraerTodo extrae o zip en destino rexeitando rutas que saian del (zip
// slip). Se todo o contido vai dentro dun único cartafol raíz, quítase.
func extraerTodo(zipRuta, destino string) error {
	zr, err := zip.OpenReader(zipRuta)
	if err != nil {
		return fmt.Errorf("o ficheiro descargado non é un zip válido: %w", err)
	}
	defer zr.Close()

	raiz := ""
	unico := true
	for _, f := range zr.File {
		nome := filepath.ToSlash(f.Name)
		primeiro, resto, ten := strings.Cut(nome, "/")
		if !ten || resto == "" && !f.FileInfo().IsDir() {
			unico = false
			break
		}
		if raiz == "" {
			raiz = primeiro
		} else if raiz != primeiro {
			unico = false
			break
		}
	}
	if !unico {
		raiz = ""
	}

	destAbs, err := filepath.Abs(destino)
	if err != nil {
		return err
	}
	n := 0
	for _, f := range zr.File {
		nome := filepath.ToSlash(f.Name)
		if raiz != "" {
			nome = strings.TrimPrefix(nome, raiz+"/")
		}
		if nome == "" {
			continue
		}
		alvo := filepath.Join(destAbs, filepath.FromSlash(nome))
		if alvo != destAbs && !strings.HasPrefix(alvo, destAbs+string(filepath.Separator)) {
			return fmt.Errorf("o zip contén unha ruta non permitida: %q", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(alvo, 0o755); err != nil {
				return err
			}
			continue
		}
		if !f.Mode().IsRegular() {
			continue // sen ligazóns simbólicas nin ficheiros especiais
		}
		if err := os.MkdirAll(filepath.Dir(alvo), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		perm := os.FileMode(0o644)
		if f.Mode()&0o111 != 0 {
			perm = 0o755
		}
		out, err := os.OpenFile(alvo, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		rc.Close()
		if e := out.Close(); err == nil {
			err = e
		}
		if err != nil {
			return err
		}
		n++
	}
	if n == 0 {
		return fmt.Errorf("o zip está baleiro")
	}
	return nil
}

// InstalarModulo descarga o .zip da Release, verifica o seu SHA-256 (obrigatorio)
// e instala o seu contido en dirModulo, substituíndo o que houbese. A
// substitución é atómica: instálase nun cartafol veciño e só ao final se
// cambia; se algo falla, dirModulo queda coma estaba. Despois comproba que
// está o executable de ax e que modulo.json leva a versión nova.
func InstalarModulo(ax Axustes, dirModulo, urlZip, urlChecksum, versionNova string) error {
	pai := filepath.Dir(dirModulo)
	if err := os.MkdirAll(pai, 0o755); err != nil {
		return fmt.Errorf("non se puido crear %q: %w", pai, err)
	}
	tmpDescarga, err := os.MkdirTemp("", ax.Executable+"-modulo-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDescarga)

	zipRuta, err := descargar(urlZip, tmpDescarga, "modulo.zip")
	if err != nil {
		return err
	}
	if err := comprobarChecksum(zipRuta, urlChecksum); err != nil {
		return err
	}

	novo, err := os.MkdirTemp(pai, "."+filepath.Base(dirModulo)+"-novo-")
	if err != nil {
		return fmt.Errorf("non se puido preparar a instalación en %q: %w", pai, err)
	}
	defer os.RemoveAll(novo) // non-op se se renomeou
	if err := extraerTodo(zipRuta, novo); err != nil {
		return err
	}
	exe := filepath.Join(novo, ax.NomeExecutable())
	if st, err := os.Stat(exe); err != nil || !st.Mode().IsRegular() {
		return fmt.Errorf("o zip do módulo non contén %q — revisa como o empaquetaches", ax.NomeExecutable())
	}
	if runtime.GOOS != "windows" {
		_ = os.Chmod(exe, 0o755)
	}
	if versionNova != "" {
		actualizarVersionModuloJSON(novo, versionNova)
	}

	anterior := dirModulo + ".anterior"
	_ = os.RemoveAll(anterior)
	teniaAnterior := false
	if _, err := os.Stat(dirModulo); err == nil {
		if err := os.Rename(dirModulo, anterior); err != nil {
			return fmt.Errorf("non se puido apartar a versión anterior: %w", err)
		}
		teniaAnterior = true
	}
	if err := os.Rename(novo, dirModulo); err != nil {
		if teniaAnterior {
			_ = os.Rename(anterior, dirModulo)
		}
		return fmt.Errorf("non se puido instalar o módulo: %w", err)
	}
	_ = os.RemoveAll(anterior)
	return nil
}
