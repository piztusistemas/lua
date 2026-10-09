// Package plataforma reúne os axudantes de sistema que as aplicacións
// comparten: localización de binarios no PATH, execución de procesos externos
// sen xanela en Windows e mailo resto de utilidades por plataforma.
package plataforma

import "os/exec"

// CommandExists di se un executable está dispoñible no PATH do proceso.
//
// En macOS unha app lanzada dende o Finder recibe de launchd un PATH mínimo:
// quen chame a isto debe asegurarse antes de refrescar o PATH (ver
// pathrefresh_*.go), ou pdflatex/pdftoppm/maxima darán "non atopado" aínda
// estando instalados.
func CommandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
