package main

import (
	"embed"
	"flag"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/linux/icon.png
var appIcon []byte

func filepathJoin(a ...string) string { return filepath.Join(a...) }

// Buxo lánzao a aplicación anfitrioa (Lua; despois Yang) para un proxecto:
//
//	buxo --boveda <cartafol> --materia <texto> --id <id da bóveda na API>
//
// e pásalle PIZTU_API_URL e PIZTU_API_TOKEN para falar coa súa API.
func main() {
	var args Argumentos
	flag.StringVar(&args.Boveda, "boveda", os.Getenv("BUXO_BOVEDA"), "cartafol da bóveda do proxecto")
	flag.StringVar(&args.Materia, "materia", os.Getenv("BUXO_MATERIA"), "tema do proxecto")
	flag.StringVar(&args.ID, "id", os.Getenv("BUXO_BOVEDA_ID"), "id da bóveda na API da aplicación anfitrioa")
	flag.Parse()
	configurarIdioma()
	if args.Boveda != "" {
		if abs, err := filepath.Abs(args.Boveda); err == nil {
			args.Boveda = abs
		}
	}

	app := application.New(application.Options{
		Name:        "Buxo",
		Description: "Bóveda de coñecemento a partir de PDF (método Karpathy)",
		Services:    []application.Service{application.NewService(NewApp(args))},
		Assets:      application.AssetOptions{Handler: application.AssetFileServerFS(assets)},
		Icon:        appIcon,
		Linux:       application.LinuxOptions{ProgramName: "Buxo"},
	})
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             "buxo",
		Title:            "Buxo",
		Width:            1100,
		Height:           760,
		BackgroundColour: application.NewRGB(23, 24, 28),
		URL:              "/",
	})
	if err := app.Run(); err != nil {
		println("Error:", err.Error())
	}
}
