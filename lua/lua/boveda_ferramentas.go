package main

// Ferramentas boveda_* do asistente (ver boveda.go). Só están dispoñibles
// cando hai unha bóveda conectada ao documento (IAFerramentasDoAmbito).

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ferramentasBoveda: nomes das ferramentas que só existen con bóveda.
var ferramentasBoveda = map[string]bool{
	"boveda_info": true, "boveda_listar": true, "boveda_ler": true, "boveda_buscar": true,
	"boveda_enlaces": true, "boveda_pendentes": true, "boveda_lint": true, "boveda_escribir": true,
}

func init() {
	ferramentasBackend = append(ferramentasBackend,
		ferramentaBackend{
			def: IAFerramenta{
				Nome: "boveda_info", Tipo: tipoLectura, Onde: "backend",
				Descricion: "Resumo da bóveda de Obsidian conectada: esquema da wiki (LUA.md), índice, final do log e número de fontes (raw/) e páxinas (wiki/). Chámaa ao comezar a traballar coa bóveda.",
			},
			exec: execBoveda("boveda_info"),
		},
		ferramentaBackend{
			def: IAFerramenta{
				Nome: "boveda_listar", Tipo: tipoLectura, Onde: "backend",
				Descricion: "Lista os ficheiros da bóveda (ruta, tamaño, data) de raw/ ou wiki/, ou dun subcartafol.",
				Parametros: esqObx(map[string]any{
					"carpeta": esqTxt("Cartafol: 'raw', 'wiki' ou un subcartafol (p.ex. wiki/conceptos). Por defecto todos."),
					"max":     esqNum("Máximo de ficheiros (por defecto 200, máximo 1000)."),
				}),
			},
			exec: execBoveda("boveda_listar"),
		},
		ferramentaBackend{
			def: IAFerramenta{
				Nome: "boveda_ler", Tipo: tipoLectura, Onde: "backend",
				Descricion: "Le unha páxina da wiki, unha fonte de raw/ ou un ficheiro especial (index.md, log.md, LUA.md). Devolve o texto (por partes se é longo: usa 'inicio') máis os enlaces saíntes e os retroenlaces para seguir o mapa da wiki.",
				Parametros: esqObx(map[string]any{
					"ruta":          esqTxt("Ruta relativa á bóveda (p.ex. wiki/fotosintese.md) ou o nome dunha páxina ([[nome]])."),
					"inicio":        esqNum("Carácter desde o que ler (por defecto 0)."),
					"maxCaracteres": esqNum("Cantos caracteres devolver (por defecto 12000, máximo 40000)."),
				}, "ruta"),
			},
			exec: execBoveda("boveda_ler"),
		},
		ferramentaBackend{
			def: IAFerramenta{
				Nome: "boveda_enlaces", Tipo: tipoLectura, Onde: "backend",
				Descricion: "Enlaces saíntes e retroenlaces dunha páxina (a vecindade no mapa da wiki), para explorar relacións sen ler todo. Con profundidade 2 inclúe tamén os veciños dos veciños.",
				Parametros: esqObx(map[string]any{
					"ruta":         esqTxt("Ruta ou nome da páxina."),
					"profundidade": esqNum("1 (por defecto) ou 2."),
				}, "ruta"),
			},
			exec: execBoveda("boveda_enlaces"),
		},
		ferramentaBackend{
			def: IAFerramenta{
				Nome: "boveda_buscar", Tipo: tipoLectura, Onde: "backend",
				Descricion: "Busca texto en raw/ e wiki/ (sen distinguir maiúsculas; todas as palabras da consulta teñen que aparecer no ficheiro, ou no nome). Devolve os ficheiros máis relevantes con fragmentos.",
				Parametros: esqObx(map[string]any{
					"consulta": esqTxt("Palabras a buscar."),
					"ambito":   esqEnum("Onde buscar: 'todo' (por defecto), 'raw' ou 'wiki'.", "todo", "raw", "wiki"),
					"max":      esqNum("Máximo de ficheiros (por defecto 10, máximo 30)."),
				}, "consulta"),
			},
			exec: execBoveda("boveda_buscar"),
		},
		ferramentaBackend{
			def: IAFerramenta{
				Nome: "boveda_pendentes", Tipo: tipoLectura, Onde: "backend",
				Descricion: "INGEST: lista as fontes de raw/ (Markdown) que ningunha páxina da wiki cita aínda (por enlace, ruta ou nome de ficheiro). Son as candidatas a procesar e incorporar á wiki.",
			},
			exec: execBoveda("boveda_pendentes"),
		},
		ferramentaBackend{
			def: IAFerramenta{
				Nome: "boveda_lint", Tipo: tipoLectura, Onde: "backend",
				Descricion: "LINT: revisión estrutural da wiki: enlaces rotos, páxinas orfas (sen retroenlaces), páxinas que non figuran no índice e fontes de raw/ sen procesar. As contradicións e lagoas de contido revísaas ti lendo as páxinas.",
			},
			exec: execBoveda("boveda_lint"),
		},
		ferramentaBackend{
			def: IAFerramenta{
				Nome: "boveda_escribir", Tipo: tipoEscritura, Onde: "backend",
				Descricion: "Crea ou actualiza unha páxina da wiki (só dentro de wiki/, ou index.md/log.md na raíz). raw/ e CLAUDE.md nunca se modifican. Para o log usa modo 'engadir'. Se sobrescribes, pasa o contido COMPLETO (le antes a páxina).",
				Parametros: esqObx(map[string]any{
					"ruta":    esqTxt("Ruta relativa á bóveda, p.ex. wiki/conceptos/fotosintese.md (acaba en .md)."),
					"contido": esqTxt("Texto Markdown."),
					"modo":    esqEnum("'sobrescribir' (por defecto) ou 'engadir' (ao final, para log.md).", "sobrescribir", "engadir"),
				}, "ruta", "contido"),
			},
			exec: execBoveda("boveda_escribir"),
		},
	)
}

// resolverRutaBoveda acepta unha ruta da bóveda ou o nome dunha páxina.
func resolverRutaBoveda(b *boveda, g *grafo, entrada string) (string, error) {
	e := strings.TrimSpace(entrada)
	e = strings.TrimSuffix(strings.TrimPrefix(e, "[["), "]]")
	if e == "" {
		return "", fmt.Errorf("falta a ruta")
	}
	if r := g.resolver("", e); r != "" && b.lexible(r) {
		return r, nil
	}
	return "", fmt.Errorf("non existe «%s» na bóveda (usa boveda_buscar ou boveda_listar)", entrada)
}

type resultadoBuscaBoveda struct {
	Ruta       string   `json:"ruta"`
	Puntos     int      `json:"-"`
	Fragmentos []string `json:"fragmentos"`
}

func buscarNaBoveda(b *boveda, consulta, ambito string, max int) (any, error) {
	termos := strings.Fields(strings.ToLower(consulta))
	if len(termos) == 0 {
		return nil, fmt.Errorf("a consulta está baleira")
	}
	if max <= 0 || max > 30 {
		max = 10
	}
	g := b.construirGrafo()
	var res []resultadoBuscaBoveda
	for rel, t := range g.textos {
		if (ambito == "raw" && !strings.HasPrefix(rel, "raw/")) || (ambito == "wiki" && !strings.HasPrefix(rel, "wiki/")) {
			continue
		}
		baixo := strings.ToLower(t)
		nome := strings.ToLower(rel)
		puntos, ok := 0, true
		for _, term := range termos {
			n := strings.Count(baixo, term)
			if strings.Contains(nome, term) {
				n += 5
			}
			if n == 0 {
				ok = false
				break
			}
			puntos += n
		}
		if !ok {
			continue
		}
		r := resultadoBuscaBoveda{Ruta: rel, Puntos: puntos}
		for _, l := range strings.Split(t, "\n") {
			ll := strings.ToLower(l)
			for _, term := range termos {
				if strings.Contains(ll, term) {
					r.Fragmentos = append(r.Fragmentos, recortarRunas(strings.TrimSpace(l), 220))
					break
				}
			}
			if len(r.Fragmentos) >= 3 {
				break
			}
		}
		res = append(res, r)
	}
	sort.Slice(res, func(i, j int) bool {
		if res[i].Puntos != res[j].Puntos {
			return res[i].Puntos > res[j].Puntos
		}
		return res[i].Ruta < res[j].Ruta
	})
	total := len(res)
	if len(res) > max {
		res = res[:max]
	}
	return map[string]any{"resultados": res, "total": total}, nil
}

// fontesPendentes: ficheiros md de raw/ que ningunha páxina da wiki (nin o log)
// cita por enlace, ruta ou nome de ficheiro.
func fontesPendentes(g *grafo) []string {
	var wiki strings.Builder
	for rel, t := range g.textos {
		if strings.HasPrefix(rel, "wiki/") || rel == "index.md" || rel == "log.md" {
			wiki.WriteString(strings.ToLower(t))
			wiki.WriteByte('\n')
		}
	}
	corpo := wiki.String()
	citada := map[string]bool{}
	for rel := range g.textos {
		if strings.HasPrefix(rel, "wiki/") || rel == "index.md" || rel == "log.md" {
			for _, d := range g.saintes[rel] {
				citada[d] = true
			}
		}
	}
	var out []string
	for _, f := range g.ficheiros {
		if !strings.HasPrefix(f.Rel, "raw/") || !f.eMD() || citada[f.Rel] {
			continue
		}
		base := strings.ToLower(filepath.Base(f.Rel))
		if strings.Contains(corpo, strings.ToLower(f.Rel)) || strings.Contains(corpo, base) ||
			strings.Contains(corpo, strings.TrimSuffix(base, filepath.Ext(base))) {
			continue
		}
		out = append(out, f.Rel)
	}
	sort.Strings(out)
	return out
}

func lintBoveda(g *grafo) map[string]any {
	rotos := map[string][]string{}
	for rel, ds := range g.rotos {
		rotos[rel] = ds
	}
	var orfas, foraIndice []string
	indice := ""
	for _, ix := range []string{"wiki/index.md", "index.md"} {
		if t, ok := g.textos[ix]; ok {
			indice = ix
			_ = t
			break
		}
	}
	for _, f := range g.ficheiros {
		if !strings.HasPrefix(f.Rel, "wiki/") || !f.eMD() {
			continue
		}
		if f.Rel == "wiki/index.md" || f.Rel == "wiki/log.md" {
			continue
		}
		if len(g.entrantes[f.Rel]) == 0 {
			orfas = append(orfas, f.Rel)
		}
		if indice != "" {
			citada := false
			for _, o := range g.entrantes[f.Rel] {
				if o == indice {
					citada = true
					break
				}
			}
			if !citada {
				foraIndice = append(foraIndice, f.Rel)
			}
		}
	}
	sort.Strings(orfas)
	sort.Strings(foraIndice)
	res := map[string]any{
		"enlacesRotos": rotos, "paxinasOrfas": orfas, "fontesSenProcesar": fontesPendentes(g),
	}
	if indice == "" {
		res["aviso"] = "non hai wiki/index.md: crea o índice"
	} else {
		res["paxinasForaDoIndice"] = foraIndice
	}
	return res
}

// escribirNaBoveda grava unha páxina da wiki. Só wiki/**.md e index.md/log.md;
// nunca raw/ nin CLAUDE.md/LUA.md.
func escribirNaBoveda(b *boveda, rel, contido string, engadir bool) (any, error) {
	rel = strings.TrimPrefix(filepath.ToSlash(strings.TrimSpace(rel)), "./")
	if !b.escribible(rel) {
		return nil, fmt.Errorf("só se pode escribir en wiki/*.md (e index.md/log.md): «%s» non se pode modificar", rel)
	}
	if len(contido) > bovedaMaxEscritura {
		return nil, fmt.Errorf("o contido é demasiado grande (máximo %d KB)", bovedaMaxEscritura>>10)
	}
	abs, err := b.rutaSegura(rel)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return nil, err
	}
	existia := false
	if _, err := os.Stat(abs); err == nil {
		existia = true
	}
	if engadir && existia {
		if actual, err := os.ReadFile(abs); err == nil {
			sep := "\n"
			if !strings.HasSuffix(string(actual), "\n") {
				sep = "\n\n"
			}
			contido = string(actual) + sep + strings.TrimLeft(contido, "\n")
		}
	}
	if !strings.HasSuffix(contido, "\n") {
		contido += "\n"
	}
	tmp := abs + ".lua-tmp"
	if err := os.WriteFile(tmp, []byte(contido), 0o644); err != nil {
		return nil, err
	}
	if err := os.Rename(tmp, abs); err != nil {
		_ = os.Remove(tmp)
		return nil, err
	}
	acc := "creada"
	if existia {
		acc = "actualizada"
		if engadir {
			acc = "ampliada"
		}
	}
	return fmt.Sprintf("páxina %s: %s", acc, rel), nil
}
