package modulo

import (
	"os"
	"path/filepath"
	"testing"

	"edicion/actualizacion"
)

func xestorProba(t *testing.T) (*Xestor, string) {
	t.Helper()
	base := t.TempDir()
	x := &Xestor{
		Axustes: actualizacion.Axustes{Nome: "Probex", Repo: "x/y", PrefixoAsset: "probex_", Executable: "probex"},
		Dirs:    func() []string { return []string{base} },
	}
	return x, base
}

func TestDirInstaladoEEstado(t *testing.T) {
	x, base := xestorProba(t)
	if e := x.Estado(); e.Instalado || e.Version != "" {
		t.Errorf("sen instalar: %+v", e)
	}
	dir := filepath.Join(base, "probex")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "probex"), []byte("x"), 0o755)
	os.WriteFile(filepath.Join(dir, "modulo.json"), []byte(`{"version":"26.10.1"}`), 0o644)
	if x.DirInstalado() != dir {
		t.Errorf("dir instalado: %q", x.DirInstalado())
	}
	if e := x.Estado(); !e.Instalado || e.Version != "26.10.1" {
		t.Errorf("instalado: %+v", e)
	}
}

func TestDirDestinoPrefereOPrimeiroEscribible(t *testing.T) {
	x, base := xestorProba(t)
	d, err := x.DirDestino()
	if err != nil || d != filepath.Join(base, "probex") {
		t.Errorf("destino: %q %v", d, err)
	}
	x.Dirs = func() []string { return []string{"/proc/nada-escribible", base} }
	if d, _ := x.DirDestino(); d != filepath.Join(base, "probex") {
		t.Errorf("debe saltar o que non se pode escribir: %q", d)
	}
	x.Dirs = func() []string { return nil }
	if _, err := x.DirDestino(); err == nil {
		t.Error("sen cartafoles debe fallar")
	}
}

func TestDirsModulos(t *testing.T) {
	// XDG_CONFIG_HOME só manda en Linux; en macOS e Windows os.UserConfigDir
	// devolve o cartafol do sistema, así que se compara co que el diga.
	t.Setenv("XDG_CONFIG_HOME", "/cfg")
	cfg, err := os.UserConfigDir()
	if err != nil {
		t.Skip("sen cartafol de configuración:", err)
	}
	d := DirsModulos("yang")
	if len(d) != 2 || d[1] != filepath.Join(cfg, "yang", "modulos") || filepath.Base(d[0]) != "modulos" {
		t.Errorf("dirs: %v", d)
	}
}
