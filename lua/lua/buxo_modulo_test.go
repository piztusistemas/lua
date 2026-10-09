package main

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"edicion/api"
	"edicion/modulo"
)

func appConBuxoFalso(t *testing.T) (*App, string, string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("executable falso en shell")
	}
	modulos := t.TempDir()
	dir := filepath.Join(modulos, "buxo")
	os.MkdirAll(dir, 0o755)
	saida := filepath.Join(t.TempDir(), "saida.txt")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + saida + "\nprintf 'URL=%s\\nTOKEN=%s\\n' \"$PIZTU_API_URL\" \"$PIZTU_API_TOKEN\" >> " + saida + "\nsleep 1\n"
	os.WriteFile(filepath.Join(dir, "buxo"), []byte(script), 0o755)
	os.WriteFile(filepath.Join(dir, "modulo.json"), []byte(`{"id":"buxo","version":"26.10"}`), 0o644)
	a := &App{}
	a.buxo = modulo.NovoXestor("lua", axustesBuxo)
	a.buxo.Dirs = func() []string { return []string{modulos} }
	a.settings.IAAPIKey, a.settings.IAModel = "k", "m"
	t.Cleanup(func() {
		if a.api != nil {
			a.api.Parar()
		}
	})
	return a, dir, saida
}

func TestBuxoEstado(t *testing.T) {
	a := &App{}
	a.buxo = modulo.NovoXestor("lua", axustesBuxo)
	d := t.TempDir()
	a.buxo.Dirs = func() []string { return []string{d} }
	e := a.BuxoEstadoActual()
	if !e.Activo || e.Instalado || e.IAConfigurada {
		t.Errorf("estado inicial: %+v", e)
	}
	a2, _, _ := appConBuxoFalso(t)
	e = a2.BuxoEstadoActual()
	if !e.Instalado || e.Version != "26.10" || !e.IAConfigurada {
		t.Errorf("estado con buxo: %+v", e)
	}
	a2.settings.BuxoDesactivado = true
	if a2.BuxoEstadoActual().Activo {
		t.Error("desactivado")
	}
}

func TestBuxoSoSeAbreComIAEActivo(t *testing.T) {
	a, _, _ := appConBuxoFalso(t)
	raiz := filepath.Join(t.TempDir(), "p")
	a.settings.IAAPIKey = ""
	if err := a.abrirBuxo(raiz, "x", "p"); err == nil || !strings.Contains(err.Error(), "IA") {
		t.Errorf("sen IA non debe abrirse: %v", err)
	}
	a.settings.IAAPIKey = "k"
	a.settings.BuxoDesactivado = true
	if err := a.abrirBuxo(raiz, "x", "p"); err == nil || !strings.Contains(err.Error(), "desactivado") {
		t.Errorf("desactivado non debe abrirse: %v", err)
	}
	b := &App{}
	b.buxo = modulo.NovoXestor("lua", axustesBuxo)
	db := t.TempDir()
	b.buxo.Dirs = func() []string { return []string{db} }
	b.settings.IAAPIKey, b.settings.IAModel = "k", "m"
	if err := b.abrirBuxo(raiz, "x", "p"); err == nil || !strings.Contains(err.Error(), "instalado") {
		t.Errorf("sen instalar non debe abrirse: %v", err)
	}
}

func TestAbrirBuxoDaArgumentosETokenLimitado(t *testing.T) {
	a, _, saida := appConBuxoFalso(t)
	raiz := filepath.Join(t.TempDir(), "proxecto")
	if err := a.abrirBuxo(raiz, "Fotosíntese", "proxecto-1"); err != nil {
		t.Fatal(err)
	}
	var txt string
	for i := 0; i < 100; i++ {
		if b, err := os.ReadFile(saida); err == nil && strings.Contains(string(b), "TOKEN=") {
			txt = string(b)
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	for _, c := range []string{"--boveda\n" + raiz, "--materia\nFotosíntese", "--id\nproxecto-1", "URL=http://127.0.0.1:"} {
		if !strings.Contains(txt, c) {
			t.Fatalf("falta %q en:\n%s", c, txt)
		}
	}
	if st, err := os.Stat(raiz); err != nil || !st.IsDir() {
		t.Error("o cartafol da bóveda ten que existir")
	}
	// O token que recibiu buxo vale para a API e a súa bóveda está rexistrada.
	var url, token string
	for _, l := range strings.Split(txt, "\n") {
		if v, ok := strings.CutPrefix(l, "URL="); ok {
			url = v
		}
		if v, ok := strings.CutPrefix(l, "TOKEN="); ok {
			token = v
		}
	}
	c := &api.Cliente{URL: url, Token: token}
	var est map[string]any
	if err := c.Obter(context.Background(), "/ia/estado", nil, &est); err != nil {
		t.Fatal(err)
	}
	if est["configurada"] != true {
		t.Errorf("ia/estado: %v", est)
	}
	if a.rutaBovedaAPI("proxecto-1") != raiz {
		t.Error("a bóveda debe quedar rexistrada na API co id do proxecto")
	}
	// Pechado buxo, o token deixa de valer.
	esperarCond(t, func() bool {
		return codigoAPI(c.Obter(context.Background(), "/ia/estado", nil, nil)) == 401
	})
}

func esperarCond(t *testing.T, f func() bool) {
	t.Helper()
	for i := 0; i < 100; i++ {
		if f() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("tempo esgotado")
}

func TestBuxoActivarPersiste(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	a := &App{}
	a.buxo = modulo.NovoXestor("lua", axustesBuxo)
	d := t.TempDir()
	a.buxo.Dirs = func() []string { return []string{d} }
	if err := a.BuxoActivar(false); err != nil {
		t.Fatal(err)
	}
	if loadSettings().BuxoDesactivado != true {
		t.Error("a desactivación debe gardarse")
	}
	// Opcións non pisa o estado de buxo.
	if err := a.SaveSettings(Settings{}); err != nil {
		t.Fatal(err)
	}
	if !a.settings.BuxoDesactivado {
		t.Error("SaveSettings non debe pisar BuxoDesactivado")
	}
}
