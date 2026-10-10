package main

// API de comunicacións de Lua (ver edicion/api): único camiño polo que se
// accede ás bóvedas de Obsidian conectadas. O asistente de Lua (ferramentas
// boveda_*) e os módulos externos (buxo) pasan polas mesmas rutas e polos
// mesmos permisos:
//
//	GET /v1/bovedas                          bóvedas conectadas
//	GET /v1/bovedas/{id}/{info|listar|ler|enlaces|buscar|pendentes|lint}
//	PUT /v1/bovedas/{id}/escribir            só wiki/*.md (e index.md/log.md)
//
// {id} é o identificador da bóveda: hoxe o tipo de documento ao que está
// conectada (cronograma, actas, plantilla, formulario).

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"edicion/api"
)

const (
	permisoBovedaLer      = "boveda.ler"
	permisoBovedaEscribir = "boveda.escribir"
	permisoIA             = "ia.usar"
	tokenLuaInterno       = "lua"
)

// opsBoveda son as operacións sobre unha bóveda: o corpo de cada ruta da API.
var opsBoveda = map[string]func(b *boveda, args map[string]any) (any, error){
	"info": func(b *boveda, args map[string]any) (any, error) {
		g := b.construirGrafo()
		var nRaw, nWiki int
		for _, f := range g.ficheiros {
			if !f.eMD() {
				continue
			}
			if strings.HasPrefix(f.Rel, "raw/") {
				nRaw++
			} else if strings.HasPrefix(f.Rel, "wiki/") {
				nWiki++
			}
		}
		esq, _ := b.ler("LUA.md")
		res := map[string]any{
			"raiz": filepath.Base(b.raiz), "fontesRaw": nRaw, "paxinasWiki": nWiki,
			"esquema": recortarRunas(esq, bovedaMaxEsquema),
		}
		for _, ix := range []string{"wiki/index.md", "index.md"} {
			if t, err := b.ler(ix); err == nil {
				res["indice"], res["rutaIndice"] = recortarRunas(t, bovedaMaxIndice), ix
				break
			}
		}
		for _, lg := range []string{"wiki/log.md", "log.md"} {
			if t, err := b.ler(lg); err == nil {
				r := []rune(strings.TrimSpace(t))
				if len(r) > 3000 {
					r = r[len(r)-3000:]
				}
				res["finalDoLog"], res["rutaLog"] = string(r), lg
				break
			}
		}
		return res, nil
	},
	"listar": func(b *boveda, args map[string]any) (any, error) {
		pref := strings.Trim(filepath.ToSlash(argTxt(args, "carpeta")), "/")
		if pref != "" && !b.lexible(pref+"/") && !raizsLegiblesBoveda[pref] {
			return nil, fmt.Errorf("só se pode listar raw/ e wiki/")
		}
		n := argInt(args, "max")
		if n <= 0 || n > 1000 {
			n = 200
		}
		type fila struct {
			Ruta   string `json:"ruta"`
			Tamano int64  `json:"tamano"`
			Data   string `json:"data"`
		}
		var out []fila
		total := 0
		for _, f := range b.inventario() {
			if pref != "" && f.Rel != pref && !strings.HasPrefix(f.Rel, pref+"/") {
				continue
			}
			total++
			if len(out) < n {
				out = append(out, fila{f.Rel, f.Tamano, f.Modif.Format("2006-01-02")})
			}
		}
		return map[string]any{"ficheiros": out, "total": total, "amosados": len(out)}, nil
	},
	"ler": func(b *boveda, args map[string]any) (any, error) {
		g := b.construirGrafo()
		rel, err := resolverRutaBoveda(b, g, argTxt(args, "ruta"))
		if err != nil {
			return nil, err
		}
		t, err := b.ler(rel)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"ruta": rel, "texto": paxinarTexto(t, argInt(args, "inicio"), argInt(args, "maxCaracteres")),
			"enlacesSaintes": g.saintes[rel], "retroenlaces": g.entrantes[rel], "enlacesRotos": g.rotos[rel],
		}, nil
	},
	"enlaces": func(b *boveda, args map[string]any) (any, error) {
		g := b.construirGrafo()
		rel, err := resolverRutaBoveda(b, g, argTxt(args, "ruta"))
		if err != nil {
			return nil, err
		}
		res := map[string]any{"ruta": rel, "saintes": g.saintes[rel], "retroenlaces": g.entrantes[rel], "rotos": g.rotos[rel]}
		if argInt(args, "profundidade") >= 2 {
			veciños := map[string]map[string][]string{}
			for _, v := range append(append([]string{}, g.saintes[rel]...), g.entrantes[rel]...) {
				if _, ok := veciños[v]; !ok {
					veciños[v] = map[string][]string{"saintes": g.saintes[v], "retroenlaces": g.entrantes[v]}
				}
			}
			res["veciños"] = veciños
		}
		return res, nil
	},
	"buscar": func(b *boveda, args map[string]any) (any, error) {
		return buscarNaBoveda(b, argTxt(args, "consulta"), argTxt(args, "ambito"), argInt(args, "max"))
	},
	"pendentes": func(b *boveda, args map[string]any) (any, error) {
		p := fontesPendentes(b.construirGrafo())
		return map[string]any{"pendentes": p, "total": len(p)}, nil
	},
	"lint": func(b *boveda, args map[string]any) (any, error) {
		return lintBoveda(b.construirGrafo()), nil
	},
	"escribir": func(b *boveda, args map[string]any) (any, error) {
		return escribirNaBoveda(b, argTxt(args, "ruta"), argTxt(args, "contido"), argTxt(args, "modo") == "engadir")
	},
}

// ---------- Servidor ----------

// servidorAPI devolve o servidor da API de Lua e o cliente interno (co que o
// propio Lua fala con el), arrincándoos a primeira vez que se piden.
func (a *App) servidorAPI() (*api.Servidor, *api.Cliente, error) {
	a.apiMu.Lock()
	defer a.apiMu.Unlock()
	if a.api != nil {
		return a.api, a.apiCliente, nil
	}
	s := api.Novo("lua")
	a.rexistrarRutasAPI(s)
	url, err := s.Iniciar()
	if err != nil {
		return nil, nil, fmt.Errorf("non se puido arrincar a API local: %w", err)
	}
	a.api = s
	a.apiCliente = &api.Cliente{URL: url, Token: s.NovoToken(tokenLuaInterno, permisoBovedaLer, permisoBovedaEscribir, permisoIA)}
	return a.api, a.apiCliente, nil
}

// TokenModuloAPI crea un token para un módulo externo (buxo) con só os
// permisos dados e devolve a URL e o token que se lle pasan ao lanzalo
// (variables PIZTU_API_URL e PIZTU_API_TOKEN).
func (a *App) tokenModuloAPI(modulo string, permisos ...string) (url, token string, err error) {
	s, c, err := a.servidorAPI()
	if err != nil {
		return "", "", err
	}
	return c.URL, s.NovoToken(modulo, permisos...), nil
}

// bovedaPorID resolve o identificador da API nunha bóveda usable.
func (a *App) bovedaPorID(id string) (*boveda, error) {
	ruta := a.rutaBovedaAPI(id)
	if ruta == "" {
		return nil, api.NonAtopado("non hai ningunha bóveda conectada coma «%s»", id)
	}
	raiz, err := validarRaizBoveda(ruta)
	if err != nil {
		return nil, err
	}
	return &boveda{raiz}, nil
}

// rutaBovedaAPI devolve a ruta da bóveda con ese id: unha bóveda rexistrada
// por un módulo (registrarBovedaAPI) ou a conectada a un tipo de documento.
func (a *App) rutaBovedaAPI(id string) string {
	a.apiMu.Lock()
	r := a.bovedasAPI[id]
	a.apiMu.Unlock()
	if r != "" {
		return r
	}
	if pid, ok := strings.CutPrefix(id, "proxecto:"); ok {
		return bovedaDoProxecto(pid)
	}
	return a.settings.IABovedas[id]
}

// registrarBovedaAPI fai accesible pola API a bóveda de ruta co id dado (ruta
// baleira = retirala). Lanzar buxo sobre unha bóveda rexístraa así.
func (a *App) registrarBovedaAPI(id, ruta string) {
	a.apiMu.Lock()
	defer a.apiMu.Unlock()
	if ruta == "" {
		delete(a.bovedasAPI, id)
		return
	}
	if a.bovedasAPI == nil {
		a.bovedasAPI = map[string]string{}
	}
	a.bovedasAPI[id] = ruta
}

func (a *App) rexistrarRutasAPI(s *api.Servidor) {
	s.Rexistrar("GET /bovedas", permisoBovedaLer, func(p *api.Peticion) (any, error) {
		out := []BovedaEstado{}
		vistos := map[string]bool{}
		a.apiMu.Lock()
		extra := map[string]string{}
		for k, v := range a.bovedasAPI {
			extra[k] = v
		}
		a.apiMu.Unlock()
		for id, r := range extra {
			vistos[id] = true
			out = append(out, a.estadoBovedaRuta(id, r))
		}
		for _, d := range documentosBoveda {
			if e := a.estadoBovedaRuta(d, a.settings.IABovedas[d]); e.Conectada && !vistos[d] {
				out = append(out, e)
			}
		}
		return out, nil
	})
	// Definicións das ferramentas boveda_* (nome, descrición, parámetros): así
	// un módulo que fai de «asistente» non as duplica.
	s.Rexistrar("GET /bovedas/ferramentas", permisoBovedaLer, func(p *api.Peticion) (any, error) {
		return a.IAFerramentasBoveda(), nil
	})
	// IA: a clave nunca sae de Lua; os módulos piden turnos de conversa.
	s.Rexistrar("GET /ia/estado", permisoIA, func(p *api.Peticion) (any, error) {
		err := comprobarIA(a.settings)
		res := map[string]any{"configurada": err == nil, "modelo": a.settings.IAModel}
		if err != nil {
			res["erro"] = err.Error()
		}
		return res, nil
	})
	s.Rexistrar("POST /ia/turno", permisoIA, func(p *api.Peticion) (any, error) {
		var pet struct {
			Sistema     string         `json:"sistema"`
			Mensaxes    []IAMensaxe    `json:"mensaxes"`
			Ferramentas []IAFerramenta `json:"ferramentas"`
		}
		if err := p.Corpo(&pet); err != nil {
			return nil, err
		}
		if err := imaxesDaAPI(pet.Mensaxes); err != nil {
			return nil, err
		}
		return chamarIAConversa(a.settings, pet.Sistema, pet.Mensaxes, pet.Ferramentas)
	})
	for nome, op := range opsBoveda {
		metodo, scope := "GET", permisoBovedaLer
		if nome == "escribir" {
			metodo, scope = "PUT", permisoBovedaEscribir
		}
		s.Rexistrar(metodo+" /bovedas/{id}/"+nome, scope, func(p *api.Peticion) (any, error) {
			b, err := a.bovedaPorID(p.Param("id"))
			if err != nil {
				return nil, err
			}
			args := map[string]any{}
			if metodo == "PUT" {
				if err := p.Corpo(&args); err != nil {
					return nil, err
				}
			} else {
				for k, v := range p.R.URL.Query() {
					args[k] = v[0]
				}
			}
			return op(b, args)
		})
	}
}

// maxImaxesAPI: tope de imaxes nun turno pedido pola API (as páxinas dun PDF).
const maxImaxesAPI = 12

// imaxesDaAPI pasa as imaxes (data URI) das mensaxes dun módulo á vista da IA.
func imaxesDaAPI(msgs []IAMensaxe) error {
	n := 0
	for i := range msgs {
		for _, uri := range msgs[i].Imaxes {
			im, ok := imaxeParaIA(uri)
			if !ok {
				return fmt.Errorf("imaxe non válida (ten que ser un data URI en base64)")
			}
			if n++; n > maxImaxesAPI {
				return fmt.Errorf("demasiadas imaxes nun turno (máximo %d)", maxImaxesAPI)
			}
			msgs[i].ImaxesIA = append(msgs[i].ImaxesIA, im)
		}
		msgs[i].Imaxes = nil
	}
	return nil
}

// execBoveda é o corpo das ferramentas boveda_* do asistente: pasa pola API
// local, nunca toca a bóveda directamente. A bóveda é a do turno en curso.
func execBoveda(nome string) func(a *App, args map[string]any) (any, error) {
	op := strings.TrimPrefix(nome, "boveda_")
	return func(a *App, args map[string]any) (any, error) {
		a.bovedaMu.Lock()
		id := a.bovedaTurnoID
		a.bovedaMu.Unlock()
		if id == "" {
			return nil, fmt.Errorf("non hai ningunha bóveda conectada a este documento (conéctaa nas opcións do asistente)")
		}
		_, c, err := a.servidorAPI()
		if err != nil {
			return nil, err
		}
		ruta := "/bovedas/" + url.PathEscape(id) + "/" + op
		var res any
		if op == "escribir" {
			err = c.Facer(context.Background(), "PUT", ruta, nil, args, &res)
		} else {
			q := url.Values{}
			for k, v := range args {
				q.Set(k, argTxt(map[string]any{k: v}, k))
			}
			err = c.Obter(context.Background(), ruta, q, &res)
		}
		return res, err
	}
}
