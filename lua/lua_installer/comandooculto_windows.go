//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

// comandoOculto é exec.Command + ocultar a xanela de consola (ver
// ocultarConsolaWindows) - winget é un proceso de consola, e este instalador
// tamén é unha app GUI sen consola propia da que herdar unha.
func comandoOculto(name string, arg ...string) *exec.Cmd {
	cmd := exec.Command(name, arg...)
	ocultarConsolaWindows(cmd)
	return cmd
}

func ocultarConsolaWindows(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = true
	cmd.SysProcAttr.CreationFlags |= 0x08000000 // CREATE_NO_WINDOW
}
