package main

// Módulo buxo (bóveda de coñecemento a partir de PDF, método Karpathy).
//
// Lua cárgao automaticamente: ao arrincar e despois cada intervaloBuxo, se
// está activo, mira a Release de piztusistemas/lua da versión da Lua instalada
// (etiqueta = VERSION, ex. 26.10.1) e as posteriores e, se buxo non está
// instalado ou hai un asset «buxo_<versión>_<so>_<arq>.zip» de versión maior
// (republicado na Release actual ou nunha máis nova), descárgao (co seu
// .sha256) e instálao no cartafol de módulos de Lua. O usuario pode desactivalo en Opcións, que é o
// único sitio onde se xestiona.
//
// Buxo é un executable á parte: Lua lánzao co cartafol da bóveda do proxecto e
// dálle un token da API local (api_boveda.go) cos permisos de bóveda e IA, así
// que non ve a clave da IA nin toca outra cousa ca a súa bóveda.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"edicion/actualizacion"
	"edicion/api"
	"edicion/modulo"

	"github.com/wailsapp/wails/v3/pkg/application"
)

var axustesBuxo = actualizacion.Axustes{
	Nome:         "Buxo",
	Repo:         "piztusistemas/lua",
	PrefixoAsset: "buxo_",
	Executable:   "buxo",
	// Buxo viaxa nas Release de Lua: mírase a da versión desta Lua e as
	// posteriores (ver init e actualizacion.Axustes.Release).
	SoUltimaRelease: true,
}

// intervaloBuxo é cada canto se busca unha versión nova de buxo mentres Lua
// está aberta.
const intervaloBuxo = time.Hour

// buxoAbertos conta as instancias de buxo lanzadas que seguen abertas: con
// algunha aberta non se substitúe o módulo (tócalle na seguinte comprobación).
var buxoAbertos atomic.Int32

const eventoBuxoCambiou = "buxo_cambiou"

func init() {
	axustesBuxo.Release = VersionLocalLua()
	application.RegisterEvent[BuxoEstado](eventoBuxoCambiou)
}

// BuxoEstado é o que ve Opcións do módulo buxo.
type BuxoEstado struct {
	Activo        bool   `json:"activo"`
	Instalado     bool   `json:"instalado"`
	Version       string `json:"version"`
	VersionRemota string `json:"versionRemota"`
	Descargando   bool   `json:"descargando"`
	// IAConfigurada: buxo só se ofrece ao usuario cando hai unha IA lista.
	IAConfigurada bool   `json:"iaConfigurada"`
	Erro          string `json:"erro,omitempty"`
}

// xestorBuxo é o xestor común (edicion/modulo) de Buxo en Lua. Cada App ten o
// seu (a.buxo), creado a primeira vez que se usa.
func (a *App) xestorBuxo() *modulo.Xestor {
	a.buxoOnce.Do(func() {
		if a.buxo == nil {
			a.buxo = modulo.NovoXestor("lua", axustesBuxo)
		}
	})
	return a.buxo
}

func nomeExecutableBuxo() string { return axustesBuxo.NomeExecutable() }

// dirBuxoInstalado devolve o cartafol onde buxo está instalado ("" se non).
func (a *App) dirBuxoInstalado() string { return a.xestorBuxo().DirInstalado() }

func (a *App) buxoActivo() bool { return !a.settings.BuxoDesactivado }

// BuxoEstadoActual devolve o estado do módulo para Opcións.
func (a *App) BuxoEstadoActual() BuxoEstado {
	m := a.xestorBuxo().Estado()
	e := BuxoEstado{
		Activo: a.buxoActivo(), Instalado: m.Instalado, Version: m.Version, VersionRemota: m.VersionRemota,
		Descargando: m.Descargando, Erro: m.Erro,
	}
	_, _, _, err := resolverIA(a.settings)
	e.IAConfigurada = err == nil
	return e
}

func (a *App) avisarBuxo() {
	if app := application.Get(); app != nil {
		app.Event.Emit(eventoBuxoCambiou, a.BuxoEstadoActual())
	}
}

// BuxoActivar activa o desactiva o módulo. Ao activalo comproba a descarga.
func (a *App) BuxoActivar(activo bool) error {
	a.settings.BuxoDesactivado = !activo
	if err := saveSettingsToDisk(a.settings); err != nil {
		return err
	}
	a.avisarBuxo()
	if activo {
		go a.sincronizarBuxo(false)
	}
	return nil
}

// BuxoActualizar força a comprobación e a instalación da última versión.
func (a *App) BuxoActualizar() error {
	return a.sincronizarBuxo(true)
}

// sincronizarBuxo instala buxo se falta ou hai versión nova. Se forzar,
// mesmo estando desactivado. Nunca bloquea o arranque: os fallos de rede quedan
// no estado (Erro).
func (a *App) sincronizarBuxo(forzar bool) error {
	if !forzar && !a.buxoActivo() {
		return nil
	}
	if buxoAbertos.Load() > 0 {
		if forzar {
			return fmt.Errorf("pecha Buxo antes de actualizalo")
		}
		return nil
	}
	return a.xestorBuxo().Sincronizar(a.avisarBuxo)
}

// vixiarBuxo sincroniza buxo ao arrincar e despois cada intervaloBuxo.
func (a *App) vixiarBuxo() {
	_ = a.sincronizarBuxo(false)
	for range time.Tick(intervaloBuxo) {
		_ = a.sincronizarBuxo(false)
	}
}

// abrirBuxo lanza buxo sobre a bóveda de raiz (id é o da bóveda na API). Só se
// pode se buxo está instalado e hai IA configurada.
func (a *App) abrirBuxo(raiz, materia, id string) error {
	if !a.buxoActivo() {
		return fmt.Errorf("o módulo Buxo está desactivado (actívao en Opcións)")
	}
	if _, _, _, err := resolverIA(a.settings); err != nil {
		return fmt.Errorf("Buxo precisa unha IA configurada en Opcións: %w", err)
	}
	dir := a.dirBuxoInstalado()
	if dir == "" {
		return fmt.Errorf("Buxo aínda non está instalado (descárgase ao arrincar Lua; mira Opcións)")
	}
	if strings.TrimSpace(raiz) == "" {
		return fmt.Errorf("falta o cartafol da bóveda")
	}
	if err := os.MkdirAll(raiz, 0o755); err != nil {
		return err
	}
	url, token, err := a.tokenModuloAPI("buxo:"+id, permisoBovedaLer, permisoBovedaEscribir, permisoIA)
	if err != nil {
		return err
	}
	a.registrarBovedaAPI(id, raiz)

	cmd := exec.Command(filepath.Join(dir, nomeExecutableBuxo()), "--boveda", raiz, "--materia", materia, "--id", id)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), api.EnvURL+"="+url, api.EnvToken+"="+token)
	// Sen plataforma.OcultarConsola: buxo é unha GUI, e en Windows HideWindow
	// pasaríalle SW_HIDE no STARTUPINFO, que a súa primeira ShowWindow obedece:
	// a xanela creábase pero quedaba oculta para sempre.
	if err := cmd.Start(); err != nil {
		a.registrarBovedaAPI(id, "")
		a.api.Revogar("buxo:" + id)
		return fmt.Errorf("non se puido lanzar Buxo: %w", err)
	}
	buxoAbertos.Add(1)
	go func() {
		_ = cmd.Wait()
		buxoAbertos.Add(-1)
		// Ao pechar buxo o token deixa de valer; a bóveda segue sendo do proxecto.
		a.api.Revogar("buxo:" + id)
	}()
	return nil
}
