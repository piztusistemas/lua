package main

import (
	"os/exec"

	"edicion/plataforma"
	"edicion/ui"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Ponte co módulo común `edicion`: a implementación destes axudantes vive unha
// soa vez en edicion/plataforma, compartida con Yang e cos módulos futuros.
// Consérvanse estes nomes locais, dunha liña cada un, para non tocar as
// chamadas que xa existen no proxecto (incluídos os seus tests).
func commandExists(name string) bool { return plataforma.CommandExists(name) }

func ocultarConsola(cmd *exec.Cmd) { plataforma.OcultarConsola(cmd) }

// ocultarXanelaDaBarraDeTarefas: truco GTK de Linux para que a xanela de
// resultado non apareza na barra de tarefas; a lóxica vive unha soa vez en
// edicion/ui, co caso baleiro nas demais plataformas. Wails só aparece aquí,
// para extraer o punteiro nativo: o módulo común non importa Wails.
func ocultarXanelaDaBarraDeTarefas(w *application.WebviewWindow) {
	ui.OcultarXanelaDaBarraDeTarefas(w.NativeWindow())
}

// refrescarPathDendeRexistro: en macOS/Windows o PATH que herda Lua ao
// arrincar dende o Finder/Explorer é unha instantánea incompleta; ver
// edicion/plataforma/pathrefresh_*.go.
func refrescarPathDendeRexistro() { plataforma.RefrescarPathDendeRexistro() }

// mbDescargaEnCurso: MB xa descargados pola instalación en marcha (só macOS).
func mbDescargaEnCurso() int64 { return plataforma.MbDescargaEnCurso() }

// bundle/brew: relanzar dentro do .app de macOS, re-asinar o bundle tras
// substituír o binario, e preparar o entorno de instalación con Homebrew.
func bundleQueContenA(exe string) string { return plataforma.BundleQueContenA(exe) }

func comandoRelanzar(exe string) *exec.Cmd { return plataforma.ComandoRelanzar(exe) }

func resinarBundleMacOS(exe string) error { return plataforma.ResinarBundleMacOS(exe) }

func prepararEntornoInstalacion(mensaxe string) ([]string, func()) {
	return plataforma.PrepararEntornoInstalacion(mensaxe)
}

func asegurarHomebrew() (string, error) { return plataforma.AsegurarHomebrew() }

// ListarFontesSistema e AbrirFicheiro seguen sendo métodos de *App porque
// Wails expón ao frontend os métodos exportados do servizo; a lóxica está no
// módulo común.
func (a *App) ListarFontesSistema() []string {
	return plataforma.FontesDoSistema()
}

func (a *App) AbrirFicheiro(ruta string) error {
	return plataforma.AbrirFicheiro(ruta)
}
