package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// InstallStatus reports the outcome of an install attempt back to the UI.
type InstallStatus struct {
	Success bool   `json:"success"`
	Log     string `json:"log"`
}

// eventoProgresoInstalacion é o nome do evento co que InstallLatex/InstallPandoc informan do seu avance mentres corren.
//
// Fai falla porque eses métodos son SÍNCRONOS: o frontend agarda a que
// devolvan InstallStatus, así que ata que rematan non ten nada que amosar
// máis alá dun "⏳ Instalando…" fixo. Iso vale para un `apt install pandoc`
// de vinte segundos, pero non para `brew install --cask mactex-no-gui`, que
// son preto de 6 GB e pode pasar media hora sen dar sinal ningún (ver
// progreso_darwin.go). Sen isto, o botón queda conxelado e o natural é
// pensar que Lua colgou e pechalo a metade.
const eventoProgresoInstalacion = "lua:instalacion-progreso"

// ProgresoInstalacion é o que viaxa no evento. Que é a ferramenta que se
// está a instalar ("latex" ou "pandoc"), para que o frontend saiba
// que botón actualizar.
type ProgresoInstalacion struct {
	Que      string `json:"que"`
	Minutos  int    `json:"minutos"`
	MB       int64  `json:"mb"` // 0 = descoñecido (ver mbDescargaEnCurso)
	Completo bool   `json:"completo"`
}

// informarProgresoInstalacion emite un evento de progreso cada 30 segundos
// mentres dure a instalación, e devolve a función que o detén. O primeiro
// evento vai decontado, para que o botón deixe de amosar un texto fixo sen
// agardar medio minuto.
func informarProgresoInstalacion(que string) (parar func()) {
	feito := make(chan struct{})
	rematado := make(chan struct{})
	inicio := time.Now()

	emitir := func(completo bool) {
		// application.Get() é nil nun test que chame o binding directamente
		// (mesma precaución que en xanelaresultado.go e app.go).
		app := application.Get()
		if app == nil {
			return
		}
		app.Event.Emit(eventoProgresoInstalacion, ProgresoInstalacion{
			Que:      que,
			Minutos:  int(time.Since(inicio).Minutes()),
			MB:       mbDescargaEnCurso(),
			Completo: completo,
		})
	}

	emitir(false)
	go func() {
		defer close(rematado)
		tic := time.NewTicker(30 * time.Second)
		defer tic.Stop()
		for {
			select {
			case <-feito:
				emitir(true)
				return
			case <-tic.C:
				emitir(false)
			}
		}
	}()

	return func() {
		close(feito)
		<-rematado
	}
}

// linuxLatexInstallCommand elixe o primeiro xestor de paquetes dispoñible e
// devolve a liña de comandos completa para instalar a cadea LaTeX. apt leva
// "update &&" por diante porque unha caché de paquetes vella é a razón máis
// común de que falle unha instalación nova. Instala: pdflatex (base+recommended, covers amsmath/amssymb/xcolor/
// geometry), TikZ (its own package on Debian/Ubuntu - texlive-latex-base
// alone does NOT include it, confirmed on this machine's dpkg -S), xelatex/
// lualatex (both optional engines offered in Opcións), and poppler-utils
// (pdftoppm - necesario para a vista previa páxina a páxina).
// Deliberately NOT texlive-full: that pulls in several GB of packages
// (fonts/languages Lua never uses).
//
// texlive-latex-extra (Debian/Ubuntu só) vai polas PLANTILLAS de documento
// (plantillas.go): tcolorbox, titlesec, adjustbox e compañía viven aí, e son
// o pan de cada día de calquera maqueta traballada - incluídas as que
// escribe a IA a partir dun modelo. Mesma lista ca o instalador de fóra
// (lua_installer/install.go), que ten que instalar exactamente o mesmo.
func linuxLatexInstallCommand() (argv []string, mgrName string, ok bool) {
	switch {
	case commandExists("apt-get"):
		return []string{"sh", "-c", "apt-get update && apt-get install -y texlive-latex-base texlive-latex-recommended texlive-pictures texlive-latex-extra texlive-xetex texlive-luatex poppler-utils"}, "apt-get", true
	case commandExists("dnf"):
		return []string{"dnf", "install", "-y", "texlive-latex", "texlive-collection-pictures", "texlive-xetex", "texlive-luatex", "poppler-utils"}, "dnf", true
	case commandExists("pacman"):
		return []string{"pacman", "-Sy", "--noconfirm", "texlive-core", "texlive-pictures", "poppler"}, "pacman", true
	case commandExists("zypper"):
		return []string{"zypper", "install", "-y", "texlive", "texlive-latex", "poppler-tools"}, "zypper", true
	}
	return nil, "", false
}

// LatexDepsStatus indica se están instaladas as ferramentas que Lua precisa
// para compilar e previsualizar PDFs: o motor de LaTeX escollido en Opcións
// (XerarInformeCronograma falla cun erro que menciona ese mesmo motor se
// falta) máis pdftoppm (poppler-utils), necesario para a vista previa
// páxina a páxina (pdfPageImages).
type LatexDepsStatus struct {
	Engine        string `json:"engine"`
	EngineFound   bool   `json:"engineFound"`
	PdftoppmFound bool   `json:"pdftoppmFound"`
}

// OK reports whether everything Lua needs to compile a PDF is present -
// exposed as a method (not just a computed JS boolean) so the frontend
// doesn't have to duplicate this exact condition.
func (s LatexDepsStatus) OK() bool { return s.EngineFound && s.PdftoppmFound }

// CheckLatexDeps re-deriva o motor configurado do mesmo xeito que XerarInformeCronograma
// (setting baleiro = pdflatex) (empty setting = pdflatex), so switching engines in Opcións updates
// what gets checked without needing anything else to change.
func (a *App) CheckLatexDeps() LatexDepsStatus {
	engine := strings.TrimSpace(a.settings.LatexEngine)
	if engine == "" {
		engine = "pdflatex"
	}
	return LatexDepsStatus{
		Engine:        engine,
		EngineFound:   commandExists(engine),
		PdftoppmFound: commandExists("pdftoppm"),
	}
}

// InstallLatex instala a cadea LaTeX co xestor de paquetes nativo da
// plataforma. En Linux pasa por pkexec, para que esta app gráfica nunca
// precise un terminal onde pedir o contrasinal: o axente polkit do escritorio
// amosa o seu propio diálogo de autenticación. Só se executa cando se pide
// explicitamente dende a interface. Ver linuxLatexInstallCommand para o que
// se instala e por que.
func (a *App) InstallLatex() InstallStatus {
	var cmd *exec.Cmd
	// logPrevio arrastra o log de ter instalado Homebrew antes de poder
	// instalar nada con el (só macOS).
	var logPrevio string
	limparEntorno := func() {}
	// limparEntorno só se usa no camiño de macOS (ver brew_darwin.go): borra o
	// programa askpass temporal. Envólvese nun closure porque aínda non ten o
	// seu valor definitivo cando se avalía o defer.
	defer func() { limparEntorno() }()

	switch runtime.GOOS {
	case "linux":
		argv, mgrName, ok := linuxLatexInstallCommand()
		if !ok {
			return InstallStatus{Log: "Non se atopou un xestor de paquetes coñecido (apt/dnf/pacman/zypper). Instala LaTeX manualmente."}
		}
		if !commandExists("pkexec") {
			return InstallStatus{Log: fmt.Sprintf("Non se atopou 'pkexec' para pedir permisos de administrador. Instala manualmente nun terminal cos comandos de %s.", mgrName)}
		}
		cmd = exec.Command("pkexec", argv...)

	case "windows":
		if !commandExists("winget") {
			return InstallStatus{Log: "Non se atopou winget. Instala MiKTeX manualmente desde https://miktex.org"}
		}
		cmd = exec.Command("winget", "install", "--id", "MiKTeX.MiKTeX", "-e",
			"--silent", "--accept-source-agreements", "--accept-package-agreements")
		ocultarConsola(cmd)

	case "darwin":
		logBrew, err := asegurarHomebrew()
		if err != nil {
			return InstallStatus{Log: logBrew + err.Error()}
		}
		logPrevio = logBrew
		// mactex-no-gui é un cask de tipo .pkg: Homebrew instálao chamando a
		// `sudo` por dentro. Sen SUDO_ASKPASS iso falla decontado ("no tty
		// present and no askpass program specified") porque Lua é unha app
		// de GUI e nunca ten terminal - de aí prepararEntornoInstalacion
		// (ver brew_darwin.go), que amosa o diálogo nativo de contrasinal.
		cmd = exec.Command("brew", "install", "--cask", "mactex-no-gui")
		cmd.Env, limparEntorno = prepararEntornoInstalacion(
			"Lua precisa permisos de administrador para instalar MacTeX (a distribución de LaTeX).")

	default:
		return InstallStatus{Log: "Sistema operativo non recoñecido para instalación automática."}
	}

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	pararProgreso := informarProgresoInstalacion("latex")
	runErr := cmd.Run()
	pararProgreso()
	log := strings.TrimSpace(logPrevio + out.String())
	// Ademais do PATH do Rexistro en Windows, isto é o que fai que en macOS
	// se vexa pdflatex xusto despois de instalar MacTeX sen reiniciar Lua:
	// o .pkg crea /Library/TeX/texbin e rexístrao en /etc/paths.d/TeX, pero
	// o PATH deste proceso quedou fixado ao arrincar (ver
	// pathrefresh_darwin.go). Sen isto, CheckLatexDeps de embaixo seguiría a
	// dicir que non hai motor de LaTeX e o botón daría a instalación por
	// fallida aínda tendo ido ben.
	refrescarPathDendeRexistro()

	// Volve comprobar sexa cal sexa o código de saída: algúns xestores de paquetes devolven un código non
	// cero por avisos aínda que a instalación fose ben.
	status := a.CheckLatexDeps()
	if !status.OK() {
		if runErr != nil {
			log += "\n" + runErr.Error()
		}
		return InstallStatus{Success: false, Log: log}
	}
	return InstallStatus{Success: true, Log: log}
}
