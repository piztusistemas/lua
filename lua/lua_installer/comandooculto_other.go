//go:build !windows

package main

import "os/exec"

// comandoOculto é exec.Command tal cal fóra de Windows: só alí fai falta
// pedir explicitamente que non se abra unha xanela de consola para o proceso
// fillo (ver comandooculto_windows.go, que é onde vive a versión real).
//
// install.go chámao sen guarda de plataforma nos camiños de winget
// (wingetPackageInstalled, instalarDependenciasWindows,
// desinstalarDependenciasWindows): ese código só se executa en Windows -
// runtime.GOOS decide antes de chegar - pero compílase en todas as
// plataformas, así que o símbolo ten que existir en todas elas.
func comandoOculto(name string, arg ...string) *exec.Cmd {
	return exec.Command(name, arg...)
}
