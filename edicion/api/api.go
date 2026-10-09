// Package api é a API de comunicacións local das aplicacións de Piztu (Lua,
// Yang) e dos seus módulos (buxo): un servidor HTTP+JSON que só escoita en
// 127.0.0.1, con tokens que levan PERMISOS (scopes), e o cliente para falar con
// el.
//
// É a única porta de entrada aos datos que Lua expón (hoxe, as bóvedas de
// Obsidian): tanto o asistente de Lua coma un módulo externo pasan por aquí,
// así os permisos e as comprobacións viven nun só sitio.
//
// Seguridade:
//
//   - só escoita en loopback, nun porto elixido polo sistema;
//   - cada petición leva «Authorization: Bearer <token>»; un token só vale para
//     os scopes co que se creou;
//   - rexéitase calquera petición cuxo Host non sexa o do servidor (contra DNS
//     rebinding) ou que traia cabeceira Origin (unha páxina web non é un
//     cliente lexítimo: o CORS non o evitaría para peticións «simples»).
//
// O paquete non coñece Wails nin o dominio: as aplicacións rexistran as súas
// rutas con Rexistrar.
package api

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// Variables de contorno co que unha aplicación lanza un módulo para que fale
// coa súa API (ver ClienteDeEntorno).
const (
	EnvURL   = "PIZTU_API_URL"
	EnvToken = "PIZTU_API_TOKEN"
)

// Versión do protocolo: vai no prefixo das rutas («/v1/...»).
const Prefixo = "/v1"

const maxCorpo = 4 << 20

// Erro é un erro da API co seu código HTTP. Un manexador devolve un *Erro para
// escoller o código; calquera outro erro vai coma 400.
type Erro struct {
	Codigo  int
	Mensaxe string
}

func (e *Erro) Error() string { return e.Mensaxe }

// NonAtopado, Prohibido e Invalido constrúen erros habituais.
func NonAtopado(formato string, a ...any) *Erro {
	return &Erro{http.StatusNotFound, fmt.Sprintf(formato, a...)}
}
func Prohibido(formato string, a ...any) *Erro {
	return &Erro{http.StatusForbidden, fmt.Sprintf(formato, a...)}
}
func Invalido(formato string, a ...any) *Erro {
	return &Erro{http.StatusBadRequest, fmt.Sprintf(formato, a...)}
}

// Peticion é o que recibe un manexador.
type Peticion struct {
	R *http.Request
	// Token é o nome co que se creou o token que fixo a petición.
	Token string
}

// Param devolve un parámetro da ruta ({id}) ou, se non hai, da query.
func (p *Peticion) Param(nome string) string {
	if v := p.R.PathValue(nome); v != "" {
		return v
	}
	return p.R.URL.Query().Get(nome)
}

// Enteiro devolve un parámetro numérico (0 se falta ou non é un número).
func (p *Peticion) Enteiro(nome string) int {
	var n int
	_, _ = fmt.Sscanf(p.Param(nome), "%d", &n)
	return n
}

// Corpo decodifica o corpo JSON da petición en v.
func (p *Peticion) Corpo(v any) error {
	d := json.NewDecoder(io.LimitReader(p.R.Body, maxCorpo))
	if err := d.Decode(v); err != nil {
		return Invalido("o corpo non é un JSON válido: %v", err)
	}
	return nil
}

// Manexador atende unha ruta. O valor devolto serialízase coma JSON.
type Manexador func(p *Peticion) (any, error)

type token struct {
	nome    string
	valor   string
	scopes  map[string]bool
	revogar bool
}

// Servidor é a API local dunha aplicación.
type Servidor struct {
	nome string
	mux  *http.ServeMux

	mu      sync.Mutex
	tokens  []*token
	scopeDe map[string]string // "METODO /ruta" -> scope
	srv     *http.Server
	url     string
	host    string
}

// Novo crea un servidor (sen arrincar). nome é o da aplicación («lua»).
func Novo(nome string) *Servidor {
	return &Servidor{nome: nome, mux: http.NewServeMux(), scopeDe: map[string]string{}}
}

// Rexistrar engade unha ruta. patron segue a sintaxe de http.ServeMux
// («GET /bovedas/{id}/info»; o prefixo /v1 engádese aquí). scope é o permiso
// que ten que ter o token ("" = basta un token válido).
func (s *Servidor) Rexistrar(patron, scope string, h Manexador) {
	metodo, ruta, ok := strings.Cut(patron, " ")
	if !ok || !strings.HasPrefix(ruta, "/") {
		panic("api: patrón non válido: " + patron)
	}
	completo := metodo + " " + Prefixo + ruta
	s.mu.Lock()
	s.scopeDe[completo] = scope
	s.mu.Unlock()
	s.mux.HandleFunc(completo, func(w http.ResponseWriter, r *http.Request) {
		tk := s.autenticar(w, r, scope)
		if tk == nil {
			return
		}
		res, err := h(&Peticion{R: r, Token: tk.nome})
		if err != nil {
			var ea *Erro
			if !errors.As(err, &ea) {
				ea = &Erro{http.StatusBadRequest, err.Error()}
			}
			escribirJSON(w, ea.Codigo, map[string]string{"erro": ea.Mensaxe})
			return
		}
		escribirJSON(w, http.StatusOK, res)
	})
}

func escribirJSON(w http.ResponseWriter, codigo int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(codigo)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Servidor) autenticar(w http.ResponseWriter, r *http.Request, scope string) *token {
	s.mu.Lock()
	host := s.host
	s.mu.Unlock()
	if r.Host != host || r.Header.Get("Origin") != "" {
		escribirJSON(w, http.StatusForbidden, map[string]string{"erro": "petición non permitida"})
		return nil
	}
	cab := r.Header.Get("Authorization")
	valor, ok := strings.CutPrefix(cab, "Bearer ")
	if !ok || valor == "" {
		escribirJSON(w, http.StatusUnauthorized, map[string]string{"erro": "falta o token"})
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var atopado *token
	for _, t := range s.tokens {
		// Compáranse todos con tempo constante en canto ao contido.
		if subtle.ConstantTimeCompare([]byte(t.valor), []byte(valor)) == 1 && !t.revogar {
			atopado = t
		}
	}
	if atopado == nil {
		escribirJSON(w, http.StatusUnauthorized, map[string]string{"erro": "token non válido"})
		return nil
	}
	if scope != "" && !atopado.scopes[scope] {
		escribirJSON(w, http.StatusForbidden, map[string]string{"erro": "o token non ten o permiso «" + scope + "»"})
		return nil
	}
	return atopado
}

// NovoToken crea un token co nome e os permisos dados e devolve o seu valor.
// O nome só serve para identificalo (rexistros, Peticion.Token).
func (s *Servidor) NovoToken(nome string, scopes ...string) string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		panic("api: sen aleatoriedade: " + err.Error())
	}
	t := &token{nome: nome, valor: hex.EncodeToString(b), scopes: map[string]bool{}}
	for _, sc := range scopes {
		t.scopes[sc] = true
	}
	s.mu.Lock()
	s.tokens = append(s.tokens, t)
	s.mu.Unlock()
	return t.valor
}

// Revogar invalida todos os tokens con ese nome.
func (s *Servidor) Revogar(nome string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.tokens {
		if t.nome == nome {
			t.revogar = true
		}
	}
}

// Iniciar arrinca o servidor en 127.0.0.1 (porto libre) e devolve a súa URL.
func (s *Servidor) Iniciar() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.srv != nil {
		return s.url, nil
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	s.host = l.Addr().String()
	s.url = "http://" + s.host
	s.srv = &http.Server{Handler: s.mux, ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = s.srv.Serve(l) }()
	return s.url, nil
}

// URL devolve a URL do servidor ("" se non está arrincado).
func (s *Servidor) URL() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.url
}

// Parar detén o servidor.
func (s *Servidor) Parar() {
	s.mu.Lock()
	srv := s.srv
	s.srv, s.url = nil, ""
	s.mu.Unlock()
	if srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}
}

// ---------- Cliente ----------

// Cliente fala coa API dunha aplicación.
type Cliente struct {
	URL   string
	Token string
	HTTP  *http.Client
}

// ClienteDeEntorno constrúe o cliente coas variables PIZTU_API_URL e
// PIZTU_API_TOKEN (as que pon a aplicación ao lanzar un módulo).
func ClienteDeEntorno() (*Cliente, error) {
	u, t := os.Getenv(EnvURL), os.Getenv(EnvToken)
	if u == "" || t == "" {
		return nil, fmt.Errorf("non hai API: faltan %s e %s (o módulo ten que lanzarse desde a aplicación)", EnvURL, EnvToken)
	}
	return &Cliente{URL: u, Token: t}, nil
}

func (c *Cliente) cliente() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 60 * time.Second}
}

// Facer executa unha petición (ruta sen o prefixo /v1). corpo vai coma JSON se
// non é nil; a resposta decodifícase en saida (se non é nil).
func (c *Cliente) Facer(ctx context.Context, metodo, ruta string, query url.Values, corpo, saida any) error {
	var lector io.Reader
	if corpo != nil {
		b, err := json.Marshal(corpo)
		if err != nil {
			return err
		}
		lector = strings.NewReader(string(b))
	}
	u := strings.TrimRight(c.URL, "/") + Prefixo + ruta
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, metodo, u, lector)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	if corpo != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.cliente().Do(req)
	if err != nil {
		return fmt.Errorf("a API non responde: %w", err)
	}
	defer resp.Body.Close()
	datos, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		var e struct {
			Erro string `json:"erro"`
		}
		if json.Unmarshal(datos, &e) == nil && e.Erro != "" {
			return &Erro{resp.StatusCode, e.Erro}
		}
		return &Erro{resp.StatusCode, fmt.Sprintf("a API respondeu %d", resp.StatusCode)}
	}
	if saida == nil {
		return nil
	}
	return json.Unmarshal(datos, saida)
}

// Obter é Facer con GET.
func (c *Cliente) Obter(ctx context.Context, ruta string, query url.Values, saida any) error {
	return c.Facer(ctx, http.MethodGet, ruta, query, nil, saida)
}
