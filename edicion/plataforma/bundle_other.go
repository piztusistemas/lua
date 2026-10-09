//go:build !darwin

package plataforma

import "os/exec"

// BundleQueContenA: fóra de macOS non hai bundles .app que localizar. Existe
// tamén aquí (e non só en bundle_darwin.go) porque o adaptador das aplicacións
// é común a todas as plataformas e senón non compilaría en Linux/Windows.
// Ver bundle_darwin.go.
func BundleQueContenA(exe string) string { return "" }

// ComandoRelanzar: fóra de macOS o executable é un binario solto e chega con
// executalo. Ver bundle_darwin.go, onde hai que pasar polo .app.
func ComandoRelanzar(exe string) *exec.Cmd { return exec.Command(exe) }

// ResinarBundleMacOS: só macOS ten bundles .app con sinatura de código que
// haxa que refacer despois de substituír o executable. En Linux un binario
// solto substituído funciona tal cal. Ver bundle_darwin.go.
func ResinarBundleMacOS(exe string) error { return nil }
