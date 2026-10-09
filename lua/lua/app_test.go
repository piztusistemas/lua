package main

import (
	"path/filepath"
	"testing"
)

// TestSplitDialogDefault fixa o erro real: pasarlle unha ruta COMPLETA a un
// diálogo de gardar coma "nome suxerido" rompía o diálogo nativo -
// DefaultFilename mapea a gtk_file_chooser_set_current_name() en Linux, que
// só admite un nome de ficheiro, sen separadores de ruta.
func TestSplitDialogDefault(t *testing.T) {
	existingDir := t.TempDir()

	dir, base := splitDialogDefault(filepath.Join(existingDir, "cronograma1.pdf"))
	if dir != existingDir || base != "cronograma1.pdf" {
		t.Errorf("existing dir: got (%q, %q), want (%q, %q)", dir, base, existingDir, "cronograma1.pdf")
	}

	// Un directorio que xa non existe (ex.: o .json movéuse/borrouse dende
	// que se abriu) non debe facer fallar o diálogo enteiro - DefaultDirectory
	// baleiro deixa que o selector use o seu propio por defecto.
	dir, base = splitDialogDefault("/non/existe/xamais/cronograma1.pdf")
	if dir != "" || base != "cronograma1.pdf" {
		t.Errorf("non-existent dir: got (%q, %q), want (\"\", %q)", dir, base, "cronograma1.pdf")
	}

	// Un nome solto, sen ruta (o caso "documento novo, sen gardar aínda").
	dir, base = splitDialogDefault("sen-nome.json")
	if dir != "" || base != "sen-nome.json" {
		t.Errorf("bare filename: got (%q, %q), want (\"\", %q)", dir, base, "sen-nome.json")
	}
}
