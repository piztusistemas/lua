//go:build windows

package plataforma

import (
	"os/exec"
	"syscall"
)

// OcultarConsola evita que cmd abra a súa propia xanela de consola visible en
// Windows. Real report: xerar un documento facía aparecer varias "pantallas
// negras", unha por cada pdflatex/pdftoppm/maxima.bat/pandoc lanzado. A app é
// unha GUI sen consola propia (ldflags -H windowsgui), así que calquera fillo
// de consola abre entón a SÚA, xa que non hai ningunha que herdar.
// CREATE_NO_WINDOW dille a CreateProcess que non cree consola ningunha para ese
// proceso - stdin/stdout/stderr seguen funcionando igual (xa van redirixidos a
// pipes/buffers, nunca a esa consola). NON usar con procesos GUI (buxo, o
// propio executable relanzado...): HideWindow pon SW_HIDE no STARTUPINFO e a
// primeira ShowWindow do fillo obedéceo, así que a súa xanela queda oculta.
func OcultarConsola(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = true
	cmd.SysProcAttr.CreationFlags |= 0x08000000 // CREATE_NO_WINDOW
}
