package api

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func servidorProba(t *testing.T) (*Servidor, string) {
	t.Helper()
	s := Novo("proba")
	s.Rexistrar("GET /eco/{id}", "ler", func(p *Peticion) (any, error) {
		return map[string]any{"id": p.Param("id"), "q": p.Param("q"), "n": p.Enteiro("n"), "token": p.Token}, nil
	})
	s.Rexistrar("PUT /eco", "escribir", func(p *Peticion) (any, error) {
		var c struct{ X string }
		if err := p.Corpo(&c); err != nil {
			return nil, err
		}
		return c.X, nil
	})
	s.Rexistrar("GET /erro", "", func(p *Peticion) (any, error) { return nil, NonAtopado("non hai «%s»", "x") })
	s.Rexistrar("GET /erro-simple", "", func(p *Peticion) (any, error) { return nil, errors.New("roto") })
	u, err := s.Iniciar()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Parar)
	return s, u
}

func TestRotaECliente(t *testing.T) {
	s, u := servidorProba(t)
	c := &Cliente{URL: u, Token: s.NovoToken("t", "ler", "escribir")}
	var r map[string]any
	if err := c.Obter(context.Background(), "/eco/abc", url.Values{"q": {"hola mundo"}, "n": {"7"}}, &r); err != nil {
		t.Fatal(err)
	}
	if r["id"] != "abc" || r["q"] != "hola mundo" || r["n"] != float64(7) || r["token"] != "t" {
		t.Errorf("resposta inesperada: %v", r)
	}
	var s2 string
	if err := c.Facer(context.Background(), http.MethodPut, "/eco", nil, map[string]string{"X": "valor"}, &s2); err != nil || s2 != "valor" {
		t.Errorf("PUT: %q %v", s2, err)
	}
}

func TestPermisosETokens(t *testing.T) {
	s, u := servidorProba(t)
	ctx := context.Background()

	if err := (&Cliente{URL: u, Token: "inventado"}).Obter(ctx, "/erro", nil, nil); !comCodigo(err, 401) {
		t.Errorf("token falso debería dar 401: %v", err)
	}
	if err := (&Cliente{URL: u}).Obter(ctx, "/erro", nil, nil); !comCodigo(err, 401) {
		t.Errorf("sen token debería dar 401: %v", err)
	}
	soLer := &Cliente{URL: u, Token: s.NovoToken("so-ler", "ler")}
	if err := soLer.Facer(ctx, http.MethodPut, "/eco", nil, map[string]string{"X": "a"}, nil); !comCodigo(err, 403) {
		t.Errorf("sen o permiso debería dar 403: %v", err)
	}
	if err := soLer.Obter(ctx, "/eco/a", nil, nil); err != nil {
		t.Errorf("co permiso debería valer: %v", err)
	}
	s.Revogar("so-ler")
	if err := soLer.Obter(ctx, "/eco/a", nil, nil); !comCodigo(err, 401) {
		t.Errorf("token revogado debería dar 401: %v", err)
	}
}

func TestErrosDosManexadores(t *testing.T) {
	s, u := servidorProba(t)
	c := &Cliente{URL: u, Token: s.NovoToken("t")}
	err := c.Obter(context.Background(), "/erro", nil, nil)
	if !comCodigo(err, 404) || !strings.Contains(err.Error(), "non hai «x»") {
		t.Errorf("404 con mensaxe: %v", err)
	}
	if err := c.Obter(context.Background(), "/erro-simple", nil, nil); !comCodigo(err, 400) {
		t.Errorf("erro simple debería dar 400: %v", err)
	}
}

func TestRexeitaOrixeEHostAlleo(t *testing.T) {
	s, u := servidorProba(t)
	tk := s.NovoToken("t", "ler")
	// Unha páxina web (Origin) non pode usar a API aínda que coñeza o token.
	req, _ := http.NewRequest("GET", u+"/v1/eco/a", nil)
	req.Header.Set("Authorization", "Bearer "+tk)
	req.Header.Set("Origin", "https://exemplo.org")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 403 {
		t.Errorf("con Origin debería dar 403, deu %d", resp.StatusCode)
	}
	// DNS rebinding: Host distinto do do servidor.
	req, _ = http.NewRequest("GET", u+"/v1/eco/a", nil)
	req.Header.Set("Authorization", "Bearer "+tk)
	req.Host = "malo.exemplo.org"
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 403 {
		t.Errorf("con Host alleo debería dar 403, deu %d", resp.StatusCode)
	}
}

func TestSoLoopback(t *testing.T) {
	_, u := servidorProba(t)
	if !strings.HasPrefix(u, "http://127.0.0.1:") {
		t.Errorf("tería que escoitar só en 127.0.0.1: %s", u)
	}
}

func TestClienteDeEntorno(t *testing.T) {
	t.Setenv(EnvURL, "")
	t.Setenv(EnvToken, "")
	if _, err := ClienteDeEntorno(); err == nil {
		t.Error("sen variables debería fallar")
	}
	t.Setenv(EnvURL, "http://127.0.0.1:1")
	t.Setenv(EnvToken, "x")
	c, err := ClienteDeEntorno()
	if err != nil || c.Token != "x" {
		t.Errorf("%v %v", c, err)
	}
}

func comCodigo(err error, codigo int) bool {
	var e *Erro
	return errors.As(err, &e) && e.Codigo == codigo
}
