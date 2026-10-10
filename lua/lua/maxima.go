package main

// Maxima en Lua: o motor de cálculo dos problemas (formulario_problemas.go),
// o mesmo de Yang (edicion/cas, edicion/xerar). É opcional: sen el os
// problemas calcúlanse en Go (números, rangos e fórmulas simples) e o
// asistente resolve sen comprobar cálculos simbólicos.

import (
	"bytes"
	"edicion/idioma"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"edicion/xerar"
)

// findMaxima busca o executable de Maxima nos sitios habituais e no PATH
// (o mesmo camiño ca Yang).
func findMaxima() string {
	for _, c := range []string{"/usr/bin/maxima", "/usr/local/bin/maxima", "/opt/homebrew/bin/maxima"} {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	// O instalador de Windows (e winget) non sempre engade maxima.bat ao PATH.
	if runtime.GOOS == "windows" {
		var atopados []string
		for _, base := range []string{`C:\`, os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)"), filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs")} {
			if base == "" {
				continue
			}
			for _, patron := range []string{filepath.Join(base, "maxima-*", "bin", "maxima.bat"), filepath.Join(base, "Maxima-*", "bin", "maxima.bat")} {
				m, _ := filepath.Glob(patron)
				atopados = append(atopados, m...)
			}
		}
		if len(atopados) > 0 {
			sort.Strings(atopados)
			return atopados[len(atopados)-1]
		}
	}
	if p, err := exec.LookPath("maxima"); err == nil {
		return p
	}
	return ""
}

// rutaMaxima é a ruta configurada se existe; se non, a que se atope.
func (a *App) rutaMaxima() string {
	if p := strings.TrimSpace(a.settings.MaximaPath); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return findMaxima()
}

// EstadoMaxima é o que a interface amosa de Maxima.
type EstadoMaxima struct {
	Atopado bool   `json:"atopado"`
	Ruta    string `json:"ruta"`
}

// CheckMaxima di se hai Maxima e onde.
func (a *App) CheckMaxima() EstadoMaxima {
	r := a.rutaMaxima()
	return EstadoMaxima{Atopado: r != "", Ruta: r}
}

// DetectMaxima volve buscar Maxima (botón «Detectar» de Opcións).
func (a *App) DetectMaxima() string { return findMaxima() }

// linuxMaximaInstallCommand: Maxima e gnuplot (para os gráficos) co xestor
// de paquetes que haxa.
func linuxMaximaInstallCommand() (argv []string, mgrName string, ok bool) {
	switch {
	case commandExists("apt-get"):
		return []string{"sh", "-c", "apt-get update && apt-get install -y maxima maxima-share gnuplot"}, "apt-get", true
	case commandExists("dnf"):
		return []string{"dnf", "install", "-y", "maxima", "gnuplot"}, "dnf", true
	case commandExists("pacman"):
		return []string{"pacman", "-Sy", "--noconfirm", "maxima", "gnuplot"}, "pacman", true
	case commandExists("zypper"):
		return []string{"zypper", "install", "-y", "maxima", "gnuplot"}, "zypper", true
	}
	return nil, "", false
}

// InstallMaxima instala Maxima co xestor de paquetes do sistema (o mesmo ca
// InstallLatex: pkexec, winget ou Homebrew). Só se executa cando se pide.
func (a *App) InstallMaxima() InstallStatus {
	var cmd *exec.Cmd
	var logPrevio string
	limparEntorno := func() {}
	defer func() { limparEntorno() }()

	switch runtime.GOOS {
	case "linux":
		argv, mgrName, ok := linuxMaximaInstallCommand()
		if !ok {
			return InstallStatus{Log: "Non se atopou un xestor de paquetes coñecido (apt/dnf/pacman/zypper). Instala Maxima manualmente."}
		}
		if !commandExists("pkexec") {
			return InstallStatus{Log: idioma.T("go.instalar.sen_pkexec", "Non se atopou 'pkexec' para pedir permisos de administrador. Instala manualmente nun terminal cos comandos de %s.", mgrName)}
		}
		cmd = exec.Command("pkexec", argv...)
	case "windows":
		if !commandExists("winget") {
			return InstallStatus{Log: "Non se atopou winget. Descarga Maxima desde https://maxima.sourceforge.io"}
		}
		// O id de winget é «MaximaTeam.Maxima» (comprobado en Yang).
		cmd = exec.Command("winget", "install", "--id", "MaximaTeam.Maxima", "-e",
			"--silent", "--accept-source-agreements", "--accept-package-agreements")
		ocultarConsola(cmd)
	case "darwin":
		logBrew, err := asegurarHomebrew()
		if err != nil {
			return InstallStatus{Log: logBrew + err.Error()}
		}
		logPrevio = logBrew
		cmd = exec.Command("brew", "install", "maxima", "gnuplot")
		cmd.Env, limparEntorno = prepararEntornoInstalacion("Lúa precisa permisos de administrador para instalar Maxima.")
	default:
		return InstallStatus{Log: "Sistema operativo non recoñecido para instalación automática."}
	}

	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	pararProgreso := informarProgresoInstalacion("maxima")
	runErr := cmd.Run()
	pararProgreso()
	log := strings.TrimSpace(logPrevio + out.String())
	refrescarPathDendeRexistro()

	// Volve buscar sexa cal sexa o código de saída (hai xestores que dan
	// erro por un aviso).
	if r := findMaxima(); r != "" {
		a.settings.MaximaPath = r
		_ = saveSettingsToDisk(a.settings)
		a.pecharMotorProblemas()
		return InstallStatus{Success: true, Log: log}
	}
	if runErr != nil {
		log += "\n" + runErr.Error()
	}
	return InstallStatus{Log: log}
}

// opcionsXerar son as opcións do motor común segundo as de Lua.
func (a *App) opcionsXerar() xerar.Opcions {
	return xerar.Opcions{Maxima: a.rutaMaxima(), Decimais: a.settings.MaximaDecimais}
}

// tempoMaxima: canto se agarda por unha resposta de Maxima.
const tempoMaxima = 20 * time.Second

// motorProblemas devolve o motor dos recálculos (abríndoo a primeira vez
// ou se cambiou a ruta de Maxima). Se Maxima non arrinca, devolve un motor
// sen Maxima e o erro coma aviso.
func (a *App) motorProblemas() (*xerar.Motor, string) {
	a.problemasMu.Lock()
	defer a.problemasMu.Unlock()
	ruta := a.rutaMaxima()
	if a.problemas != nil && a.problemasDe == ruta {
		return a.problemas, ""
	}
	if a.problemas != nil {
		a.problemas.Close()
	}
	o := a.opcionsXerar()
	m, err := xerar.NovoMotor(o, ruta != "", tempoMaxima)
	aviso := ""
	if err != nil {
		// Non se volve tentar ata que cambie a ruta (Opcións ou instalar).
		aviso = err.Error()
		m, _ = xerar.NovoMotor(o, false, 0)
	}
	a.problemas, a.problemasDe = m, ruta
	return m, aviso
}

// pecharMotorProblemas pecha a sesión de Maxima (ao saír ou ao cambiar a ruta).
func (a *App) pecharMotorProblemas() {
	a.problemasMu.Lock()
	defer a.problemasMu.Unlock()
	if a.problemas != nil {
		a.problemas.Close()
		a.problemas = nil
	}
}
