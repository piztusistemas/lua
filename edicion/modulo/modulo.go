// Package modulo xestiona un módulo descargable dunha aplicación (Buxo en Lua
// e en Yang): onde está instalado, onde se instala, e a descarga/actualización
// desde as Releases de GitHub (ver edicion/actualizacion.InstalarModulo). A
// lóxica é a mesma para todas as aplicacións; cada unha só aporta o seu
// cartafol de configuración e os Axustes do módulo.
//
// Un Xestor nunca bloquea o arrinque: os fallos de rede quedan no seu Estado.
package modulo

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"edicion/actualizacion"
)

// Estado é o estado do módulo para a interface.
type Estado struct {
	Instalado     bool   `json:"instalado"`
	Version       string `json:"version"`
	VersionRemota string `json:"versionRemota"`
	Descargando   bool   `json:"descargando"`
	Erro          string `json:"erro,omitempty"`
}

// Xestor leva o estado dun módulo.
type Xestor struct {
	// Axustes do módulo (repositorio, prefixo do asset, executable).
	Axustes actualizacion.Axustes
	// Dirs devolve os cartafoles de módulos da aplicación, por orde de
	// preferencia (ver DirsModulos). Se é nil úsase DirsModulos("").
	Dirs func() []string

	mu            sync.Mutex
	descargando   bool
	erro          string
	versionRemota string
}

// NovoXestor crea o xestor dun módulo da aplicación `app` (o nome do seu
// cartafol de configuración: «lua», «yang»).
func NovoXestor(app string, ax actualizacion.Axustes) *Xestor {
	return &Xestor{Axustes: ax, Dirs: func() []string { return DirsModulos(app) }}
}

// DirsModulos son os cartafoles de módulos dunha aplicación, por orde de
// preferencia: «modulos» ao lado do executable (instalación de Piztu,
// /opt/piztu/modulos/<app>/modulos) e, se alí non se pode escribir, na
// configuración do usuario (<config>/<app>/modulos).
func DirsModulos(app string) []string {
	var out []string
	if exe, err := os.Executable(); err == nil {
		if real, err := filepath.EvalSymlinks(exe); err == nil {
			exe = real
		}
		out = append(out, filepath.Join(filepath.Dir(exe), "modulos"))
	}
	if cfg, err := os.UserConfigDir(); err == nil && app != "" {
		out = append(out, filepath.Join(cfg, app, "modulos"))
	}
	return out
}

func (x *Xestor) dirs() []string {
	if x.Dirs != nil {
		return x.Dirs()
	}
	return DirsModulos("")
}

// nomeDir é o nome do cartafol do módulo dentro de cada cartafol de módulos.
func (x *Xestor) nomeDir() string { return x.Axustes.Executable }

// DirInstalado devolve o cartafol onde o módulo está instalado ("" se non).
func (x *Xestor) DirInstalado() string {
	for _, d := range x.dirs() {
		dir := filepath.Join(d, x.nomeDir())
		if st, err := os.Stat(filepath.Join(dir, x.Axustes.NomeExecutable())); err == nil && st.Mode().IsRegular() {
			return dir
		}
	}
	return ""
}

// DirDestino devolve onde instalar o módulo: onde xa está, se se pode
// escribir alí, ou o primeiro cartafol de módulos con permiso.
func (x *Xestor) DirDestino() (string, error) {
	if d := x.DirInstalado(); d != "" && escribible(filepath.Dir(d)) {
		return d, nil
	}
	for _, d := range x.dirs() {
		if escribible(d) {
			return filepath.Join(d, x.nomeDir()), nil
		}
	}
	return "", fmt.Errorf("non hai ningún cartafol de módulos onde se poida escribir")
}

func escribible(dir string) bool {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return false
	}
	f, err := os.CreateTemp(dir, ".escribible-")
	if err != nil {
		return false
	}
	f.Close()
	_ = os.Remove(f.Name())
	return true
}

// Estado devolve o estado actual (sen tocar a rede).
func (x *Xestor) Estado() Estado {
	x.mu.Lock()
	e := Estado{Descargando: x.descargando, Erro: x.erro, VersionRemota: x.versionRemota}
	x.mu.Unlock()
	if d := x.DirInstalado(); d != "" {
		e.Instalado = true
		e.Version = actualizacion.VersionDeModulo(d)
	}
	return e
}

// Sincronizar instala o módulo se falta ou hai unha versión nova. aoCambiar
// (pode ser nil) avisa cando cambia o estado (empeza e remata a descarga).
// Se xa hai unha descarga en curso non fai nada. Devolve o erro, que tamén
// queda no Estado.
func (x *Xestor) Sincronizar(aoCambiar func()) error {
	x.mu.Lock()
	if x.descargando {
		x.mu.Unlock()
		return nil
	}
	x.descargando, x.erro = true, ""
	x.mu.Unlock()
	if aoCambiar != nil {
		aoCambiar()
	}

	err := x.instalarSeFalta()

	x.mu.Lock()
	x.descargando = false
	if err != nil {
		x.erro = err.Error()
	}
	x.mu.Unlock()
	if aoCambiar != nil {
		aoCambiar()
	}
	return err
}

func (x *Xestor) instalarSeFalta() error {
	instalado := x.DirInstalado()
	local := ""
	if instalado != "" {
		local = actualizacion.VersionDeModulo(instalado)
	}
	est := actualizacion.Comprobar(x.Axustes, local)
	x.mu.Lock()
	x.versionRemota = est.VersionRemota
	x.mu.Unlock()
	if est.URLDescarga == "" {
		if instalado == "" {
			if x.Axustes.Release != "" {
				return fmt.Errorf("a Release %s de %s non leva ningún «%s*.zip» para esta plataforma (¿sen rede?)", x.Axustes.Release, x.Axustes.Repo, x.Axustes.PrefixoAsset)
			}
			if x.Axustes.SoUltimaRelease {
				return fmt.Errorf("a última Release de %s non leva ningún «%s*.zip» para esta plataforma (¿sen rede?)", x.Axustes.Repo, x.Axustes.PrefixoAsset)
			}
			return fmt.Errorf("non se atopou ningún «%s*.zip» para esta plataforma nas Releases de %s (¿sen rede?)", x.Axustes.PrefixoAsset, x.Axustes.Repo)
		}
		return nil // xa o temos e non hai nada que comprobar
	}
	if instalado != "" && !est.Disponible {
		return nil
	}
	destino, err := x.DirDestino()
	if err != nil {
		return err
	}
	return actualizacion.InstalarModulo(x.Axustes, destino, est.URLDescarga, est.URLChecksum, est.VersionRemota)
}
