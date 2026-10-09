//go:build !darwin

package plataforma

// Talóns para as plataformas que non son macOS: os case "darwin" de
// InstallMaxima/InstallLatex/InstallPandoc (installer.go) compílanse en
// todas elas aínda que só se executen nun Mac, así que estes símbolos teñen
// que existir sempre. Ver brew_darwin.go para o que fan de verdade.

// PrepararEntornoInstalacion: fóra de macOS non hai que preparar entorno
// ningún (nil = o proceso fillo herda o de Yang). En Linux o contrasinal
// pídeo o axente de polkit vía pkexec e en Windows pídeo o propio UAC.
func PrepararEntornoInstalacion(mensaxe string) (env []string, limpar func()) {
	return nil, func() {}
}

// AsegurarHomebrew: Homebrew só se usa no camiño de macOS.
func AsegurarHomebrew() (string, error) { return "", nil }
