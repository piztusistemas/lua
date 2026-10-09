//go:build !(linux && gtk3)

package ui

import "unsafe"

// OcultarXanelaDaBarraDeTarefas non fai nada nesta plataforma/build.
//
// En Linux con "-tags gtk3" (xanela_gtk_linux.go) fai falla un truco GTK
// explícito para agochar a xanela de resultado da barra de tarefas. En
// Windows non fai falla: a aplicación xa pide iso na propia creación da
// xanela con application.WebviewWindowOptions{Windows:
// application.WindowsWindow{HiddenOnTaskbar: true}} - Wails aplícao
// internamente coa mesma técnica (WS_EX_TOOLWINDOW), sen precisar código
// propio aquí. En macOS e en Linux sen "-tags gtk3" isto queda sen efecto (a
// xanela de resultado aparece na barra de tarefas coma calquera outra) - non
// impide usar a aplicación, só é menos elegante.
func OcultarXanelaDaBarraDeTarefas(native unsafe.Pointer) {}
