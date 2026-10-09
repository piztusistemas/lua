package web

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSincronizarEVerificar comproba o circuíto completo nun cartafol temporal:
// o que Sincronizar escribe, Verificar acéptao sen queixarse.
func TestSincronizarEVerificar(t *testing.T) {
	destino := t.TempDir()
	if err := Sincronizar(destino); err != nil {
		t.Fatalf("Sincronizar: %v", err)
	}
	if err := Verificar(destino); err != nil {
		t.Fatalf("Verificar logo de Sincronizar: %v", err)
	}

	nomes, err := Ficheiros()
	if err != nil {
		t.Fatalf("Ficheiros: %v", err)
	}
	if len(nomes) == 0 {
		t.Fatal("a fonte do frontend común está baleira")
	}
	for _, nome := range nomes {
		if _, err := os.Stat(filepath.Join(destino, filepath.FromSlash(nome))); err != nil {
			t.Errorf("falta o ficheiro copiado %s: %v", nome, err)
		}
	}

	// O manifesto ten unha liña "<sha256>  <nome>" por ficheiro.
	manifest, err := os.ReadFile(filepath.Join(destino, NomeManifest))
	if err != nil {
		t.Fatalf("ler %s: %v", NomeManifest, err)
	}
	liñas := strings.Count(strings.TrimSpace(string(manifest)), "\n") + 1
	if liñas != len(nomes) {
		t.Errorf("%s ten %d liñas, agardábanse %d (unha por ficheiro)", NomeManifest, liñas, len(nomes))
	}
	for _, nome := range nomes {
		h, err := Hash(nome)
		if err != nil {
			t.Fatalf("Hash(%s): %v", nome, err)
		}
		if !strings.Contains(string(manifest), h+"  "+nome) {
			t.Errorf("%s non rexistra %s (%s)", NomeManifest, nome, h[:12])
		}
	}
}

// TestVerificarDetectaContidoAlterado é a garda contra a deriva de verdade: se
// alguén edita a copia xerada, Verificar ten que fallar.
func TestVerificarDetectaContidoAlterado(t *testing.T) {
	destino := t.TempDir()
	if err := Sincronizar(destino); err != nil {
		t.Fatalf("Sincronizar: %v", err)
	}
	nomes, err := Ficheiros()
	if err != nil {
		t.Fatalf("Ficheiros: %v", err)
	}
	ruta := filepath.Join(destino, filepath.FromSlash(nomes[0]))
	contido, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatalf("ler %s: %v", ruta, err)
	}
	if err := os.WriteFile(ruta, append(contido, []byte("\n// edición local\n")...), 0o644); err != nil {
		t.Fatalf("editar %s: %v", ruta, err)
	}
	err = Verificar(destino)
	if err == nil {
		t.Fatal("Verificar aceptou unha copia editada a man")
	}
	if !strings.Contains(err.Error(), ComandoSincronizar) {
		t.Errorf("a mensaxe de erro non di como rexenerar a copia: %v", err)
	}
}

func TestVerificarDetectaFicheiroQueFalta(t *testing.T) {
	destino := t.TempDir()
	if err := Sincronizar(destino); err != nil {
		t.Fatalf("Sincronizar: %v", err)
	}
	nomes, err := Ficheiros()
	if err != nil {
		t.Fatalf("Ficheiros: %v", err)
	}
	if err := os.Remove(filepath.Join(destino, filepath.FromSlash(nomes[0]))); err != nil {
		t.Fatalf("eliminar a copia: %v", err)
	}
	if err := Verificar(destino); err == nil {
		t.Fatal("Verificar aceptou unha copia incompleta")
	}
}

func TestVerificarDetectaFicheiroDeMais(t *testing.T) {
	destino := t.TempDir()
	if err := Sincronizar(destino); err != nil {
		t.Fatalf("Sincronizar: %v", err)
	}
	sobra := filepath.Join(destino, "motor-duplicado.js")
	if err := os.WriteFile(sobra, []byte("// código de motor fóra do módulo\n"), 0o644); err != nil {
		t.Fatalf("crear %s: %v", sobra, err)
	}
	if err := Verificar(destino); err == nil {
		t.Fatal("Verificar aceptou código alleo dentro da copia xerada")
	}
}

// TestSincronizarRetiraCopiasMortas comproba que un ficheiro que xa non existe
// na fonte desaparece do destino (e que os cartafoles que quedan baleiros se
// retiran), para que non queden copias mortas dunha fase anterior.
func TestSincronizarRetiraCopiasMortas(t *testing.T) {
	destino := t.TempDir()
	if err := Sincronizar(destino); err != nil {
		t.Fatalf("Sincronizar: %v", err)
	}
	morto := filepath.Join(destino, "asistente", "vello.js")
	if err := os.MkdirAll(filepath.Dir(morto), 0o755); err != nil {
		t.Fatalf("crear cartafol: %v", err)
	}
	if err := os.WriteFile(morto, []byte("// xa non existe na fonte\n"), 0o644); err != nil {
		t.Fatalf("crear %s: %v", morto, err)
	}
	if err := Sincronizar(destino); err != nil {
		t.Fatalf("segunda Sincronizar: %v", err)
	}
	if _, err := os.Stat(morto); !os.IsNotExist(err) {
		t.Errorf("a copia morta %s non se retirou (err=%v)", morto, err)
	}
	if _, err := os.Stat(filepath.Dir(morto)); !os.IsNotExist(err) {
		t.Errorf("o cartafol baleiro %s non se retirou (err=%v)", filepath.Dir(morto), err)
	}
	if err := Verificar(destino); err != nil {
		t.Fatalf("Verificar despois de limpar: %v", err)
	}
}
