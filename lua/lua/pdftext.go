package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// extraerTextoPDF decodifica un PDF en base64 nun ficheiro temporal e
// executa pdftotext -layout sobre el (mantén a disposición de columnas/táboas
// mellor cá modo por defecto). pdftotext vén con poppler-utils, que xa é
// unha dependencia de Lua (ver installer.go/InstallLatex). Úsase para ler un
// PDF de mostra ao crear unha plantilla coa IA (plantillas_ia.go).
func extraerTextoPDF(dataB64 string) (string, error) {
	if !commandExists("pdftotext") {
		return "", fmt.Errorf("non se atopou pdftotext (paquete poppler-utils) para ler o PDF de mostra")
	}
	data, err := base64.StdEncoding.DecodeString(dataB64)
	if err != nil {
		return "", fmt.Errorf("PDF non válido: %w", err)
	}
	tmpDir, err := os.MkdirTemp("", "lua-pdf-modelo-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	pdfPath := filepath.Join(tmpDir, "modelo.pdf")
	if err := os.WriteFile(pdfPath, data, 0o644); err != nil {
		return "", err
	}

	cmd := exec.Command("pdftotext", "-layout", pdfPath, "-")
	ocultarConsola(cmd)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("pdftotext: %w", err)
	}
	return string(out), nil
}

// truncarRunas corta `s` a como moito `limite` runas, avisando ao final se
// tivo que facelo (para que a IA e quen usa Lua saiban que o documento
// achegado era máis longo có que se lle mandou).
func truncarRunas(s string, limite int) string {
	r := []rune(s)
	if len(r) <= limite {
		return s
	}
	return string(r[:limite]) + "\n[...documento truncado, era máis longo...]"
}
