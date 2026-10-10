package main

import (
	"encoding/base64"
	"slices"
	"strings"
	"testing"
	"unicode/utf16"
)

// Os accesos directos levan o nome visible «Lúa»; o executable e os
// cartafoles seguen sendo «lua».
func TestAccesosDirectosChamanseLua(t *testing.T) {
	if nomeAtalloWindows != "Lúa.lnk" {
		t.Errorf("atallo de Windows = %q, quería Lúa.lnk", nomeAtalloWindows)
	}
	// Os de instalacións anteriores bórranse ao instalar e ao desinstalar.
	if !slices.Contains(atallosWindowsVellos, "Lua.lnk") {
		t.Errorf("falta Lua.lnk entre os atallos vellos: %v", atallosWindowsVellos)
	}
	d := entradaEscritorioLinux("/opt/piztu/lua", "/usr/share/pixmaps/lua.png")
	for _, quere := range []string{"Name=Lúa\n", "Name[gl_ES]=Lúa\n", "Exec=/opt/piztu/lua/lua\n"} {
		if !strings.Contains(d, quere) {
			t.Errorf("a entrada .desktop non ten %q:\n%s", quere, d)
		}
	}
}

// O script de PowerShell vai codificado (UTF-16LE en base64) para que o til
// de «Lúa» chegue intacto sexa cal sexa a páxina de códigos da consola.
func TestArgsPowerShellConservaOTil(t *testing.T) {
	script := `$sc = $s.CreateShortcut('C:\Users\Ana\Desktop\Lúa.lnk')`
	args := argsPowerShell(script)
	i := slices.Index(args, "-EncodedCommand")
	if i < 0 || i+1 >= len(args) {
		t.Fatalf("faltan -EncodedCommand e o script: %v", args)
	}
	b, err := base64.StdEncoding.DecodeString(args[i+1])
	if err != nil || len(b)%2 != 0 {
		t.Fatalf("base64 non válido: %v", err)
	}
	u := make([]uint16, len(b)/2)
	for j := range u {
		u[j] = uint16(b[2*j]) | uint16(b[2*j+1])<<8
	}
	if got := string(utf16.Decode(u)); got != script {
		t.Errorf("script descodificado = %q", got)
	}
}
