package main

// Descarga e copia de PDF á carpeta pdf/ da bóveda.

import (
	"bytes"
	"context"
	"edicion/idioma"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	maxTamanoPDF    = 300 << 20
	timeoutDescarga = 3 * time.Minute
	axenteUsuario   = "Mozilla/5.0 (compatible; Buxo/1.0; +https://github.com/piztusistemas/lua)"
)

var reNomeInseguro = regexp.MustCompile(`[^\p{L}\p{N}._ -]+`)

// nomePDFSeguro devolve un nome de ficheiro .pdf válido en todos os sistemas.
func nomePDFSeguro(nome string) string {
	nome = filepath.Base(strings.ReplaceAll(nome, "\\", "/"))
	nome = strings.TrimSuffix(nome, filepath.Ext(nome))
	nome = strings.TrimSpace(reNomeInseguro.ReplaceAllString(nome, "_"))
	nome = strings.Trim(nome, "._ ")
	if len([]rune(nome)) > 120 {
		nome = string([]rune(nome)[:120])
	}
	if nome == "" {
		nome = "documento"
	}
	return nome + ".pdf"
}

// nomeLibre devolve un nome que non existe en dir (engade -1, -2...).
func nomeLibre(dir, nome string) string {
	ext := filepath.Ext(nome)
	base := strings.TrimSuffix(nome, ext)
	cand := nome
	for i := 1; ; i++ {
		if _, err := os.Stat(filepath.Join(dir, cand)); err != nil {
			return cand
		}
		cand = fmt.Sprintf("%s-%d%s", base, i, ext)
	}
}

// ePDF comproba a cabeceira «%PDF-» (os lectores toleran lixo nos primeiros
// 1024 bytes).
func ePDF(inicio []byte) bool {
	if len(inicio) > 1024 {
		inicio = inicio[:1024]
	}
	return bytes.Contains(inicio, []byte("%PDF-"))
}

// nomeDeResposta saca o nome do PDF da cabeceira Content-Disposition ou, se
// non hai, do final da URL.
func nomeDeResposta(r *http.Response, u *url.URL) string {
	if cd := r.Header.Get("Content-Disposition"); cd != "" {
		if _, p, err := mime.ParseMediaType(cd); err == nil && p["filename"] != "" {
			return p["filename"]
		}
	}
	b := path.Base(u.Path)
	if b == "." || b == "/" || b == "" {
		return u.Hostname()
	}
	if dec, err := url.PathUnescape(b); err == nil {
		b = dec
	}
	return b
}

// DescargarPDF descarga a URL a dir e devolve o nome co que quedou. Falla se o
// que se descarga non é un PDF ou pasa de maxTamanoPDF.
func DescargarPDF(ctx context.Context, cliente *http.Client, enderezo, dir string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(enderezo))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", idioma.Errorf("erro.url_non_valida", "a URL «%s» non é válida (só http e https)", enderezo)
	}
	if cliente == nil {
		cliente = &http.Client{Timeout: timeoutDescarga}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", axenteUsuario)
	req.Header.Set("Accept", "application/pdf,*/*;q=0.5")
	resp, err := cliente.Do(req)
	if err != nil {
		return "", idioma.Errorf("erro.descarga", "non se puido descargar: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", idioma.Errorf("erro.servidor", "o servidor respondeu %d", resp.StatusCode)
	}
	cab := make([]byte, 1024)
	n, _ := io.ReadFull(resp.Body, cab)
	cab = cab[:n]
	if !ePDF(cab) {
		return "", idioma.Errorf("erro.non_pdf_contido", "non é un PDF (contido: %s)", resp.Header.Get("Content-Type"))
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	nome := nomeLibre(dir, nomePDFSeguro(nomeDeResposta(resp, u)))
	destino := filepath.Join(dir, nome)
	tmp := destino + ".parte"
	f, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	_, err = io.Copy(f, io.MultiReader(bytes.NewReader(cab), io.LimitReader(resp.Body, maxTamanoPDF-int64(len(cab))+1)))
	cerrar := f.Close()
	if err == nil {
		err = cerrar
	}
	if err == nil {
		if st, e := os.Stat(tmp); e == nil && st.Size() > maxTamanoPDF {
			err = idioma.Errorf("erro.pdf_grande", "o PDF pasa de %d MB", maxTamanoPDF>>20)
		}
	}
	if err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	if err := os.Rename(tmp, destino); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	return nome, nil
}

// CopiarPDF copia un PDF local a dir e devolve o nome co que quedou.
func CopiarPDF(orixe, dir string) (string, error) {
	f, err := os.Open(orixe)
	if err != nil {
		return "", err
	}
	defer f.Close()
	cab := make([]byte, 1024)
	n, _ := io.ReadFull(f, cab)
	if !ePDF(cab[:n]) {
		return "", idioma.Errorf("erro.non_pdf", "«%s» non é un PDF", filepath.Base(orixe))
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	nome := nomeLibre(dir, nomePDFSeguro(filepath.Base(orixe)))
	destino := filepath.Join(dir, nome)
	tmp := destino + ".parte"
	g, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	_, err = io.Copy(g, f)
	if e := g.Close(); err == nil {
		err = e
	}
	if err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	return nome, os.Rename(tmp, destino)
}
