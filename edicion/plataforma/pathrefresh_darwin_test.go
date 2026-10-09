//go:build darwin

package plataforma

import (
	"slices"
	"strings"
	"testing"
)

// TestPathHelperDirsLeEtcPaths comproba que se len as fontes oficiais de
// /usr/libexec/path_helper: é por aí por onde MacTeX (/etc/paths.d/TeX) se
// declara, e non pola lista fixa do código.
func TestPathHelperDirsLeEtcPaths(t *testing.T) {
	dirs := pathHelperDirsMacOS()
	if len(dirs) == 0 {
		t.Fatal("esperaba polo menos o contido de /etc/paths")
	}
	// /usr/bin está en /etc/paths en calquera macOS.
	if !slices.Contains(dirs, "/usr/bin") {
		t.Errorf("esperaba /usr/bin dende /etc/paths, atopei %v", dirs)
	}
	for _, d := range dirs {
		if strings.TrimSpace(d) != d || d == "" {
			t.Errorf("directorio mal recortado: %q", d)
		}
	}
}
