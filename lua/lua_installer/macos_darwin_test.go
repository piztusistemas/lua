//go:build darwin

package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestDestinoMacOS: destDir e nomeExecutable teñen que encaixar entre si —
// filepath.Join(destDir, nomeExecutable()) é a ruta que app.go marca coma
// executable despois de extraer, e se non apunta a Contents/MacOS/lua o
// bundle queda sen permiso de execución e non abre.
func TestDestinoMacOS(t *testing.T) {
	if destDir != "/Applications/Lua.app" {
		t.Errorf("destDir = %q, esperaba /Applications/Lua.app", destDir)
	}
	if !strings.HasSuffix(destDir, ".app") {
		t.Errorf("en macOS o destino ten que ser un bundle .app: %q", destDir)
	}

	exe := filepath.Join(destDir, nomeExecutable())
	if exe != "/Applications/Lua.app/Contents/MacOS/lua" {
		t.Errorf("ruta do executable = %q, esperaba /Applications/Lua.app/Contents/MacOS/lua", exe)
	}
}

// TestDestinoNonEBorrableDeMais: desinstalarLuaCompleto fai
// os.RemoveAll(destDir). Se algunha vez alguén cambiase destDir a
// "/Applications" (por exemplo para que files/ puidese conter un Lua.app/
// enteiro), a desinstalación borraría TODAS as aplicacións do equipo.
func TestDestinoNonEBorrableDeMais(t *testing.T) {
	for _, prohibido := range []string{"/", "/Applications", "/Users", "/System", "/Library"} {
		if filepath.Clean(destDir) == prohibido {
			t.Fatalf("destDir = %q: desinstalarLuaCompleto faría os.RemoveAll aí", destDir)
		}
	}
}

// TestVariantesIdiomaMacOS comproba o mecanismo de claves ".darwin"
// (aplicarVariantesSO, i18n.go): nun Mac non se pode seguir dicindo que hai
// que executar o instalador con sudo nin que Lua vai a /opt/piztu.
func TestVariantesIdiomaMacOS(t *testing.T) {
	for _, code := range []string{"gl", "es", "en", "pt"} {
		t.Run(code, func(t *testing.T) {
			tr := NewTranslator(code)
			todo := tr.All()

			// Ningunha variante pode quedar sen resolver no dicionario que
			// consome o frontend.
			for clave := range todo {
				for _, sufixo := range sufixosSO {
					if strings.HasSuffix(clave, sufixo) {
						t.Errorf("variante sen resolver: %q", clave)
					}
				}
			}

			// As claves que falaban de Linux teñen que ter collido a
			// variante de macOS.
			if got := todo["desinst.desc"]; !strings.Contains(got, "/Applications/Lua.app") {
				t.Errorf("desinst.desc segue sen ser de macOS: %q", got)
			}
			for _, clave := range []string{"benvida.aviso", "desinst.desc", "fin.creados", "benvida.modulo.corpo"} {
				if got := todo[clave]; strings.Contains(got, "/opt/piztu") {
					t.Errorf("%s segue a mencionar /opt/piztu: %q", clave, got)
				}
			}
			// "executa con sudo" é, ademais de falso, un mal consello en
			// macOS: Homebrew négase a funcionar coma root.
			if got := todo["benvida.aviso"]; strings.Contains(strings.ToLower(got), "con sudo") ||
				strings.Contains(strings.ToLower(got), "with sudo") {
				t.Errorf("benvida.aviso segue a recomendar sudo: %q", got)
			}

			// E o T() de Go (InstalacionDesc/CompletadoDesc) ten que ver o
			// mesmo dicionario xa resolto.
			if got := tr.T("fin.desc", destDir); !strings.Contains(got, destDir) {
				t.Errorf("fin.desc non interpolou o destino: %q", got)
			}
		})
	}
}
