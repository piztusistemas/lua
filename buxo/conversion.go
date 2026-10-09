package main

// Conversión de PDF a Markdown co sidecar de markitdown (sidecar/md_convert.py
// empaquetado con PyInstaller; vai no zip de Buxo, ao lado do executable).
//
// O sidecar recibe un cartafol de saída e varios PDF e escribe unha liña JSON
// por ficheiro: {"input","output","ok","error"}. Un PDF que falla non para os
// demais.

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"edicion/plataforma"
)

// ResultadoConversion é unha liña de saída do sidecar.
type ResultadoConversion struct {
	Entrada string `json:"input"`
	Saida   string `json:"output"`
	Ok      bool   `json:"ok"`
	Erro    string `json:"error"`
}

// rutaSidecar busca o sidecar: BUXO_SIDECAR, ou ao lado do executable. En
// macOS buxo é universal e o zip leva un sidecar nativo por arquitectura
// (markitdown-sidecar-arm64 / -amd64; PyInstaller non pode facelo universal
// porque numpy non ten wheels universal2): cóllese o desta CPU, para non
// pasar por Rosetta. Se non está, o markitdown-sidecar de sempre.
func rutaSidecar() (string, error) {
	if r := os.Getenv("BUXO_SIDECAR"); r != "" {
		return r, nil
	}
	nome := "markitdown-sidecar"
	if runtime.GOOS == "windows" {
		nome += ".exe"
	}
	nomes := []string{nome}
	if runtime.GOOS == "darwin" {
		nomes = []string{nome + "-" + runtime.GOARCH, nome}
	}
	exe, err := os.Executable()
	if err == nil {
		for _, n := range nomes {
			p := filepath.Join(filepath.Dir(exe), n)
			if _, err := os.Stat(p); err == nil {
				return p, nil
			}
		}
	}
	return "", fmt.Errorf("non se atopa o conversor %s (ten que ir ao lado de buxo)", nome)
}

// ConverterPDFs converte os PDF a Markdown en saida (un .md por PDF, mesmo
// nome). Chama a cada resultado segundo vai chegando. Devolve erro só se o
// sidecar non se pode lanzar ou morre sen dar resultado a algún ficheiro.
func ConverterPDFs(ctx context.Context, sidecar, saida string, pdfs []string, cadaResultado func(ResultadoConversion)) error {
	if len(pdfs) == 0 {
		return nil
	}
	cmd := exec.CommandContext(ctx, sidecar, append([]string{saida}, pdfs...)...)
	plataforma.OcultarConsola(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("non se puido lanzar o conversor: %w", err)
	}
	dados := map[string]bool{}
	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		var r ResultadoConversion
		if json.Unmarshal(sc.Bytes(), &r) != nil || r.Entrada == "" {
			continue
		}
		dados[r.Entrada] = true
		cadaResultado(r)
	}
	espera := cmd.Wait()
	// Os PDF sen resposta (o conversor morreu, ou cancelouse) dánse por fallidos.
	for _, p := range pdfs {
		if !dados[p] {
			motivo := "o conversor rematou sen dar resultado"
			if ctx.Err() != nil {
				motivo = "cancelado"
			} else if espera != nil {
				motivo = "o conversor fallou: " + espera.Error()
			}
			cadaResultado(ResultadoConversion{Entrada: p, Erro: motivo})
		}
	}
	return nil
}

// cabeceiraFonte engade ao Markdown convertido unha cabeceira YAML co orixe
// (para que a IA poida citar a fonte).
func cabeceiraFonte(md, pdf, urlOrixe string) string {
	return cabeceiraFonteCon(md, pdf, urlOrixe, "")
}

// cabeceiraFonteCon: e, ademais, como se converteu (markitdown, «ia
// (visión)»...), para saber se as fórmulas e os debuxos son de fiar.
func cabeceiraFonteCon(md, orixinal, urlOrixe, conversion string) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("pdf: " + yamlTexto(dirPDF+"/"+orixinal) + "\n")
	if urlOrixe != "" {
		b.WriteString("url: " + yamlTexto(urlOrixe) + "\n")
	}
	if conversion != "" {
		b.WriteString("conversion: " + yamlTexto(conversion) + "\n")
	}
	b.WriteString("---\n\n")
	b.WriteString(md)
	return b.String()
}

func yamlTexto(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", " ").Replace(s) + `"`
}
