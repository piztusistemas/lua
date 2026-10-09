package main

// Ferramentas de web do asistente: buscar e ler páxinas públicas (a
// lexislación vive en sedes oficiais: BOE, DOG, portais das consellerías...).
// Só se executan cando a IA as chama, e só saen da máquina o texto da busca e
// os enderezos que pide a IA.
//
// O contido que volve é DATO, nunca instrucións: o prompt do asistente di que
// o que veña da web non se obedece. Ademais, estas ferramentas só falan con
// enderezos públicos (http/https): non se poden usar para chegar a servizos
// da propia máquina ou da rede local.

import (
	"bytes"
	"context"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	xhtml "golang.org/x/net/html"
)

const (
	webMaxBytes         = 4 << 20 // máximo que se descarga dunha páxina
	webCaracteresPorDef = 12000
	webCaracteresMax    = 40000
	webUserAgent        = "Mozilla/5.0 (X11; Linux x86_64) Lua/1.0 (asistente de informes)"
)

// urlBuscador é o buscador que usa web_buscar (unha variable para as probas).
var urlBuscador = "https://html.duckduckgo.com/html/"

// webPermitirLocal só o pon a true un test, para falar cun servidor local.
var webPermitirLocal = false

// enderezoPublico di se unha IP é de Internet (non local, non privada).
func enderezoPublico(ip netip.Addr) bool {
	ip = ip.Unmap()
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() || ip.IsUnspecified() {
		return false
	}
	if ip.Is4() {
		b := ip.As4()
		if b[0] == 100 && b[1]&0xC0 == 64 { // 100.64.0.0/10 (CGNAT)
			return false
		}
		if b[0] == 0 || b[0] >= 240 {
			return false
		}
	}
	return true
}

// comprobarURLPublica valida que a URL sexa http(s) e que o servidor resolva
// só a enderezos públicos.
func comprobarURLPublica(u *url.URL) error {
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("só se poden consultar enderezos http(s)")
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("enderezo sen servidor")
	}
	if webPermitirLocal {
		return nil
	}
	if ip, err := netip.ParseAddr(host); err == nil {
		if !enderezoPublico(ip) {
			return fmt.Errorf("non se pode consultar un enderezo local ou privado")
		}
		return nil
	}
	if strings.EqualFold(host, "localhost") || strings.HasSuffix(strings.ToLower(host), ".local") || strings.HasSuffix(strings.ToLower(host), ".internal") {
		return fmt.Errorf("non se pode consultar un enderezo local ou privado")
	}
	ips, err := net.DefaultResolver.LookupNetIP(context.Background(), "ip", host)
	if err != nil {
		return fmt.Errorf("non se puido resolver %s: %w", host, err)
	}
	for _, ip := range ips {
		if !enderezoPublico(ip) {
			return fmt.Errorf("non se pode consultar un enderezo local ou privado")
		}
	}
	return nil
}

func clienteWeb() *http.Client {
	return &http.Client{
		Timeout: 25 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 6 {
				return fmt.Errorf("demasiadas redireccións")
			}
			return comprobarURLPublica(req.URL)
		},
	}
}

// descargarWeb devolve o corpo (limitado) e o tipo de contido dunha URL.
func descargarWeb(rawURL string) ([]byte, string, error) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, "", fmt.Errorf("enderezo non válido")
	}
	if err := comprobarURLPublica(u); err != nil {
		return nil, "", err
	}
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", webUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/pdf,text/plain;q=0.9,*/*;q=0.5")
	req.Header.Set("Accept-Language", "gl,es;q=0.9,en;q=0.5")
	resp, err := clienteWeb().Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("non se puido abrir a páxina: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, "", fmt.Errorf("a páxina respondeu HTTP %d", resp.StatusCode)
	}
	corpo, err := io.ReadAll(io.LimitReader(resp.Body, webMaxBytes))
	if err != nil {
		return nil, "", err
	}
	return corpo, resp.Header.Get("Content-Type"), nil
}

// textoDeHTML extrae o texto lexible dun HTML: sen scripts, estilos nin
// navegación, cos parágrafos e as liñas de lista separados.
func textoDeHTML(corpo []byte) string {
	tok := xhtml.NewTokenizer(bytes.NewReader(corpo))
	var b strings.Builder
	saltar := 0 // profundidade dentro de etiquetas que non levan texto útil
	omitidas := map[string]bool{"script": true, "style": true, "noscript": true, "head": true, "svg": true, "nav": true, "footer": true, "iframe": true}
	bloque := map[string]bool{"p": true, "div": true, "br": true, "li": true, "tr": true, "h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true, "section": true, "article": true, "table": true, "ul": true, "ol": true, "blockquote": true}
	for {
		tt := tok.Next()
		switch tt {
		case xhtml.ErrorToken:
			return normalizarTextoWeb(b.String())
		case xhtml.StartTagToken, xhtml.SelfClosingTagToken:
			nome, _ := tok.TagName()
			n := string(nome)
			if omitidas[n] && tt == xhtml.StartTagToken {
				saltar++
			}
			if saltar == 0 && bloque[n] {
				b.WriteString("\n")
			}
			if saltar == 0 && n == "li" {
				b.WriteString("- ")
			}
		case xhtml.EndTagToken:
			nome, _ := tok.TagName()
			n := string(nome)
			if omitidas[n] && saltar > 0 {
				saltar--
			}
			if saltar == 0 && bloque[n] {
				b.WriteString("\n")
			}
		case xhtml.TextToken:
			if saltar == 0 {
				b.Write(tok.Text())
			}
		}
	}
}

var (
	espazosWeb   = regexp.MustCompile(`[ \t\x{00a0}]+`)
	saltosWeb    = regexp.MustCompile(`\n\s*\n\s*\n+`)
	resultadoDD  = regexp.MustCompile(`(?s)<a[^>]*class="result__a"[^>]*href="([^"]+)"[^>]*>(.*?)</a>.*?(?:class="result__snippet"[^>]*>(.*?)</a>|class="result__snippet"[^>]*>(.*?)</(?:td|div)>)`)
	etiquetasWeb = regexp.MustCompile(`<[^>]+>`)
)

func normalizarTextoWeb(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	linhas := strings.Split(s, "\n")
	for i, l := range linhas {
		linhas[i] = strings.TrimSpace(espazosWeb.ReplaceAllString(l, " "))
	}
	s = strings.Join(linhas, "\n")
	s = saltosWeb.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

// webConsultar descarga unha páxina (HTML, texto ou PDF) e devolve o seu texto,
// desde o carácter `inicio` e ata `maxCaracteres` (para ler documentos longos
// por partes). Avisa ao final se queda máis texto.
func webConsultar(rawURL string, inicio, maxCaracteres int) (string, error) {
	corpo, tipo, err := descargarWeb(rawURL)
	if err != nil {
		return "", err
	}
	var texto string
	tipoMin := strings.ToLower(tipo)
	switch {
	case strings.Contains(tipoMin, "pdf") || bytes.HasPrefix(corpo, []byte("%PDF")):
		texto, err = textoDePDFBytes(corpo)
		if err != nil {
			return "", err
		}
	case strings.Contains(tipoMin, "html") || strings.Contains(tipoMin, "xml") || bytes.Contains(corpo[:min(len(corpo), 1024)], []byte("<html")):
		texto = textoDeHTML(corpo)
	default:
		texto = string(corpo)
	}
	if maxCaracteres <= 0 {
		maxCaracteres = webCaracteresPorDef
	}
	if maxCaracteres > webCaracteresMax {
		maxCaracteres = webCaracteresMax
	}
	runas := []rune(texto)
	total := len(runas)
	if inicio < 0 {
		inicio = 0
	}
	if inicio >= total {
		return fmt.Sprintf("[a páxina ten %d caracteres; o inicio pedido (%d) está fóra]", total, inicio), nil
	}
	fin := min(inicio+maxCaracteres, total)
	saida := string(runas[inicio:fin])
	if fin < total {
		saida += fmt.Sprintf("\n\n[... continúa: amosáronse os caracteres %d-%d de %d. Para seguir, chama de novo con inicio=%d ...]", inicio, fin, total, fin)
	}
	return saida, nil
}

// textoDePDFBytes le un PDF descargado con pdftotext.
func textoDePDFBytes(datos []byte) (string, error) {
	if !commandExists("pdftotext") {
		return "", fmt.Errorf("a páxina é un PDF e non se atopou pdftotext (paquete poppler-utils) para lelo")
	}
	tmp, err := os.MkdirTemp("", "lua-web-pdf-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)
	ruta := filepath.Join(tmp, "doc.pdf")
	if err := os.WriteFile(ruta, datos, 0o600); err != nil {
		return "", err
	}
	cmd := exec.Command("pdftotext", "-layout", ruta, "-")
	ocultarConsola(cmd)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("pdftotext: %w", err)
	}
	return normalizarTextoWeb(string(out)), nil
}

// resultadoBusca é unha liña dos resultados do buscador.
type resultadoBusca struct {
	Titulo, URL, Resumo string
}

func limparHTMLInline(s string) string {
	return strings.TrimSpace(html.UnescapeString(etiquetasWeb.ReplaceAllString(s, "")))
}

// enderezoRealDD desfai o enderezo de redirección do buscador
// ("//duckduckgo.com/l/?uddg=<url codificada>&rut=...").
func enderezoRealDD(href string) string {
	href = html.UnescapeString(href)
	if strings.HasPrefix(href, "//") {
		href = "https:" + href
	}
	u, err := url.Parse(href)
	if err != nil {
		return href
	}
	if v := u.Query().Get("uddg"); v != "" {
		return v
	}
	return href
}

// parsearResultadosDD extrae os resultados da páxina HTML do buscador.
func parsearResultadosDD(corpo string, max int) []resultadoBusca {
	var out []resultadoBusca
	for _, m := range resultadoDD.FindAllStringSubmatch(corpo, -1) {
		resumo := m[3]
		if resumo == "" {
			resumo = m[4]
		}
		out = append(out, resultadoBusca{
			Titulo: limparHTMLInline(m[2]),
			URL:    enderezoRealDD(m[1]),
			Resumo: limparHTMLInline(resumo),
		})
		if len(out) >= max {
			break
		}
	}
	return out
}

// webBuscar pregunta ao buscador e devolve os primeiros resultados.
func webBuscar(consulta string, max int) (string, error) {
	consulta = strings.TrimSpace(consulta)
	if consulta == "" {
		return "", fmt.Errorf("falta a consulta")
	}
	if max <= 0 || max > 10 {
		max = 8
	}
	u, err := url.Parse(urlBuscador)
	if err != nil {
		return "", err
	}
	if err := comprobarURLPublica(u); err != nil {
		return "", err
	}
	req, err := http.NewRequest(http.MethodPost, urlBuscador, strings.NewReader(url.Values{"q": {consulta}, "kl": {"es-es"}}.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", webUserAgent)
	resp, err := clienteWeb().Do(req)
	if err != nil {
		return "", fmt.Errorf("non se puido contactar co buscador: %w", err)
	}
	defer resp.Body.Close()
	corpo, _ := io.ReadAll(io.LimitReader(resp.Body, webMaxBytes))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("o buscador respondeu HTTP %d", resp.StatusCode)
	}
	res := parsearResultadosDD(string(corpo), max)
	if len(res) == 0 {
		return "", fmt.Errorf("o buscador non devolveu resultados (ou cambiou o seu formato). Proba con web_consultar sobre un enderezo oficial (boe.es, xunta.gal, edu.xunta.gal)")
	}
	var b strings.Builder
	for i, r := range res {
		fmt.Fprintf(&b, "%d. %s\n   %s\n", i+1, r.Titulo, r.URL)
		if r.Resumo != "" {
			fmt.Fprintf(&b, "   %s\n", r.Resumo)
		}
	}
	return strings.TrimSpace(b.String()), nil
}
