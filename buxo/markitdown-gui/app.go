package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// The markitdown sidecar (PyInstaller bundle with markitdown and all its
// Python dependencies) is built by sidecar/build.sh into ./embedded.
//
//go:embed embedded
var sidecarFS embed.FS

// App is bound to the frontend.
type App struct {
	ctx context.Context
}

// ConvertResult describes the outcome for one file.
type ConvertResult struct {
	Input  string `json:"input"`
	Output string `json:"output"`
	Ok     bool   `json:"ok"`
	Error  string `json:"error"`
}

func NewApp() *App { return &App{} }

func (a *App) startup(ctx context.Context) { a.ctx = ctx }

// SelectPDFs opens a native multi-file dialog filtered to PDFs.
func (a *App) SelectPDFs() ([]string, error) {
	return wruntime.OpenMultipleFilesDialog(a.ctx, wruntime.OpenDialogOptions{
		Title:   "Selecciona ficheiros PDF",
		Filters: []wruntime.FileFilter{{DisplayName: "Ficheiros PDF (*.pdf)", Pattern: "*.pdf;*.PDF"}},
	})
}

// SelectOutputDir opens a native directory chooser.
func (a *App) SelectOutputDir(current string) (string, error) {
	return wruntime.OpenDirectoryDialog(a.ctx, wruntime.OpenDialogOptions{
		Title:            "Selecciona o cartafol de destino",
		DefaultDirectory: current,
	})
}

// extractSidecar writes the embedded binary to the user cache dir (once per
// content hash) and returns its path.
func extractSidecar() (string, error) {
	name := "markitdown-sidecar"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	data, err := sidecarFS.ReadFile("embedded/" + name)
	if err != nil {
		return "", fmt.Errorf("o motor markitdown non está incrustado (executa sidecar/build.sh): %w", err)
	}
	sum := sha256.Sum256(data)
	cache, err := os.UserCacheDir()
	if err != nil {
		cache = os.TempDir()
	}
	dir := filepath.Join(cache, "markitdown-gui", hex.EncodeToString(sum[:8]))
	path := filepath.Join(dir, name)
	if st, err := os.Stat(path); err == nil && st.Size() == int64(len(data)) {
		return path, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o755); err != nil {
		return "", err
	}
	return path, os.Rename(tmp, path)
}

// Convert runs markitdown over the files, emitting a "progress" event per file.
func (a *App) Convert(files []string, outDir string) ([]ConvertResult, error) {
	if len(files) == 0 {
		return nil, fmt.Errorf("non hai ficheiros para converter")
	}
	if outDir == "" {
		return nil, fmt.Errorf("selecciona un cartafol de destino")
	}
	bin, err := extractSidecar()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(bin, append([]string{outDir}, files...)...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	var results []ConvertResult
	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		var r ConvertResult
		if json.Unmarshal(sc.Bytes(), &r) != nil {
			continue
		}
		results = append(results, r)
		wruntime.EventsEmit(a.ctx, "progress", r)
	}
	if err := cmd.Wait(); err != nil && len(results) == 0 {
		return nil, fmt.Errorf("fallou o motor de conversión: %w", err)
	}
	return results, nil
}
