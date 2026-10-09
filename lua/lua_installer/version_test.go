package main

import (
	"os"
	"strings"
	"testing"
)

// TestLuaEmbebidaEAVersionDoCodigo: o instalador ten que levar dentro o Lua
// do código actual. Se files/ quedou dunha compilación vella (pasou o
// 2026-10-10: instalábase a 26.10.2, sen os problemas), este test falla.
// Arranxo: `make build`, que recompila Lua antes do instalador.
func TestLuaEmbebidaEAVersionDoCodigo(t *testing.T) {
	esperada, err := os.ReadFile("../lua/VERSION")
	if err != nil {
		t.Skip("sen ../lua/VERSION:", err)
	}
	if _, err := luaFiles.ReadDir("files"); err != nil {
		t.Skip("files/ baleiro: aínda non se compilou Lua para o instalador")
	}
	got := versionLuaEmbebida()
	if got != strings.TrimSpace(string(esperada)) {
		t.Errorf("o instalador leva Lua %q, pero o código de Lua é %q: executa `make build`", got, strings.TrimSpace(string(esperada)))
	}
}

func TestVersionLuaDoInfoPlist(t *testing.T) {
	plist := `<dict><key>CFBundleName</key><string>Lua</string>
        <key>CFBundleShortVersionString</key>
            <string>26.10.3</string></dict>`
	if v := versionDoInfoPlist([]byte(plist)); v != "26.10.3" {
		t.Errorf("versión = %q", v)
	}
	if v := versionDoInfoPlist([]byte("<dict/>")); v != "" {
		t.Errorf("sen clave debería ser baleira: %q", v)
	}
}
