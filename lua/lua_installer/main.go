package main

import (
	"embed"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

// versionInstalador é a versión deste instalador (non confundir coa de Lua
// en si — lua/modulo.json → "version" — aínda que hoxe se publiquen sempre
// xuntas). Ver App.Version en app.go. Mesmo patrón que installer/main.go.
//go:embed VERSION
var versionInstalador string

func main() {
	// En macOS, un .app aberto con dobre clic recibe de launchd un PATH
	// mínimo que non inclúe /opt/homebrew/bin: sen isto, `brew` non se
	// atoparía e o paso de dependencias fallaría sempre (no-op fóra de
	// macOS, ver macos_darwin.go).
	refrescarPathMacOS()

	if saiuOK, permisoDenegado := intentarElevar(); saiuOK {
		return
	} else {
		app := NewApp()
		app.permisoDenegado = permisoDenegado

		wailsApp := application.New(application.Options{
			Name:        "Lua Installer",
			Description: "Instalador de Lua",
			Services: []application.Service{
				application.NewService(app),
			},
			Assets: application.AssetOptions{
				Handler: application.AssetFileServerFS(assets),
			},
			Icon: appIcon,
			Linux: application.LinuxOptions{
				ProgramName: "Lua Installer",
			},
			// Unha soa xanela: ao pechala remata o instalador.
			Mac: application.MacOptions{
				ApplicationShouldTerminateAfterLastWindowClosed: true,
			},
		})

		wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
			Title:            "Lua — Instalador",
			Width:            1024,
			Height:           720,
			StartState:       application.WindowStateMaximised,
			BackgroundColour: application.NewRGB(245, 245, 244), // --bg-main
			URL:              "/",
		})

		err := wailsApp.Run()
		if err != nil {
			println("Error:", err.Error())
		}
	}
}
