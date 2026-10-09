package actualizacion

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func zipDe(t *testing.T, ficheiros map[string]string) []byte {
	t.Helper()
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	for n, c := range ficheiros {
		h := &zip.FileHeader{Name: n, Method: zip.Deflate}
		h.SetMode(0o755)
		f, err := w.CreateHeader(h)
		if err != nil {
			t.Fatal(err)
		}
		f.Write([]byte(c))
	}
	w.Close()
	return b.Bytes()
}

func servidorModulo(t *testing.T, z []byte, sha string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/m.zip", func(w http.ResponseWriter, r *http.Request) { w.Write(z) })
	mux.HandleFunc("/m.zip.sha256", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(sha + "  m.zip\n")) })
	s := httptest.NewServer(mux)
	t.Cleanup(s.Close)
	return s
}

func hashDe(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

func TestInstalarModulo(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}
	ax := Axustes{Nome: "Buxo", Executable: "buxo", PrefixoAsset: "buxo_"}
	z := zipDe(t, map[string]string{
		"buxo": "exe", "markitdown-sidecar": "sc", "modulo.json": `{"id":"buxo","version":"1.0"}`, "lib/x.txt": "x",
	})
	s := servidorModulo(t, z, hashDe(z))
	dir := filepath.Join(t.TempDir(), "modulos", "buxo")

	if VersionDeModulo(dir) != "" {
		t.Fatal("sen módulo non hai versión")
	}
	if err := InstalarModulo(ax, dir, s.URL+"/m.zip", s.URL+"/m.zip.sha256", "26.11"); err != nil {
		t.Fatal(err)
	}
	if v := VersionDeModulo(dir); v != "26.11" {
		t.Errorf("versión: %q", v)
	}
	for _, f := range []string{"buxo", "markitdown-sidecar", "lib/x.txt"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("falta %s", f)
		}
	}
	if st, _ := os.Stat(filepath.Join(dir, "buxo")); st.Mode()&0o111 == 0 {
		t.Error("o executable debe ser executable")
	}
	// Reinstalar substitúe e non deixa restos.
	if err := InstalarModulo(ax, dir, s.URL+"/m.zip", s.URL+"/m.zip.sha256", "26.12"); err != nil {
		t.Fatal(err)
	}
	if VersionDeModulo(dir) != "26.12" {
		t.Error("debería actualizarse a versión")
	}
	restos, _ := filepath.Glob(filepath.Join(filepath.Dir(dir), "*"))
	if len(restos) != 1 {
		t.Errorf("non deben quedar restos: %v", restos)
	}
}

func TestInstalarModuloRexeitaChecksumMaloEZipSlip(t *testing.T) {
	ax := Axustes{Nome: "Buxo", Executable: "buxo", PrefixoAsset: "buxo_"}
	dir := filepath.Join(t.TempDir(), "buxo")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "vello"), []byte("v"), 0o644)

	z := zipDe(t, map[string]string{"buxo": "exe"})
	s := servidorModulo(t, z, strings.Repeat("0", 64))
	if err := InstalarModulo(ax, dir, s.URL+"/m.zip", s.URL+"/m.zip.sha256", "1"); err == nil {
		t.Error("un checksum que non coincide debe rexeitarse")
	}
	if err := InstalarModulo(ax, dir, s.URL+"/m.zip", "", "1"); err == nil {
		t.Error("sen checksum debe rexeitarse")
	}
	if _, err := os.Stat(filepath.Join(dir, "vello")); err != nil {
		t.Error("se falla, o módulo anterior queda intacto")
	}

	mal := zipDe(t, map[string]string{"buxo": "exe", "../fora.txt": "x"})
	s2 := servidorModulo(t, mal, hashDe(mal))
	if err := InstalarModulo(ax, dir, s2.URL+"/m.zip", s2.URL+"/m.zip.sha256", "1"); err == nil {
		t.Error("zip slip debe rexeitarse")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(dir), "fora.txt")); err == nil {
		t.Error("escribiuse fóra do módulo")
	}

	sen := zipDe(t, map[string]string{"outra": "x"})
	s3 := servidorModulo(t, sen, hashDe(sen))
	if err := InstalarModulo(ax, dir, s3.URL+"/m.zip", s3.URL+"/m.zip.sha256", "1"); err == nil || !strings.Contains(err.Error(), "buxo") {
		t.Errorf("sen executable debe fallar: %v", err)
	}
}

func TestExtraerTodoQuitaCartafolRaiz(t *testing.T) {
	z := zipDe(t, map[string]string{"buxo_1/buxo": "e", "buxo_1/modulo.json": "{}"})
	f := filepath.Join(t.TempDir(), "a.zip")
	os.WriteFile(f, z, 0o644)
	d := t.TempDir()
	if err := extraerTodo(f, d); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(d, "buxo")); err != nil {
		t.Error("debería quitar o cartafol raíz")
	}
}
