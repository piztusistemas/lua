package main

// Ferramentas do asistente para o CATÁLOGO COMÚN de formularios (spec
// reutilizar-formularios): ler o catálogo e ordenalo en cartafoles. Reutilizar
// no proxecto aberto é unha ferramenta de frontend (catalogo_reutilizar),
// porque despois hai que recargar as ferramentas da columna.

import (
	"fmt"
	"strings"
)

var ferramentasCatalogo = []ferramentaBackend{
	{
		def: IAFerramenta{
			Nome: "catalogo_ler", Tipo: tipoLectura, Onde: "backend",
			Descricion: "Le o catálogo común de formularios (todos os creados, sexa no proxecto que sexa): id, nome, descrición e cartafol de cada un, os cartafoles, e que ferramentas ten xa o proxecto aberto. O cronograma e as actas sempre se poden reutilizar.",
		},
		exec: func(a *App, _ map[string]any) (any, error) { return catalogoFormularios() },
	},
	{
		def: IAFerramenta{
			Nome: "catalogo_mover", Tipo: tipoEscritura, Onde: "backend",
			Descricion: "Pon un ou varios formularios do catálogo nun cartafol (polo nome; se non existe, créase). Cartafol baleiro = sen cartafol. Os ids saen de catalogo_ler.",
			Parametros: esqObx(map[string]any{
				"ids":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Ids dos formularios (sen «form:»)."},
				"cartafol": esqTxt("Nome do cartafol (créase se non existe); baleiro = sen cartafol."),
			}, "ids"),
		},
		exec: func(a *App, args map[string]any) (any, error) {
			ids := argLista(args, "ids")
			if len(ids) == 0 {
				return nil, fmt.Errorf("faltan os ids dos formularios")
			}
			carpeta := ""
			if nome := strings.TrimSpace(argTxt(args, "cartafol")); nome != "" {
				id, err := crearCarpetaCatalogo(nome)
				if err != nil {
					return nil, err
				}
				carpeta = id
			}
			for _, id := range ids {
				if err := moverFormularioCatalogo(strings.TrimPrefix(id, "form:"), carpeta); err != nil {
					return nil, err
				}
			}
			return map[string]any{"movidos": len(ids), "cartafol": carpeta}, nil
		},
	},
	{
		def: IAFerramenta{
			Nome: "catalogo_cartafol_renomear", Tipo: tipoEscritura, Onde: "backend",
			Descricion: "Cambia o nome dun cartafol do catálogo de formularios (id de catalogo_ler).",
			Parametros: esqObx(map[string]any{"id": esqTxt("Id do cartafol."), "nome": esqTxt("Nome novo.")}, "id", "nome"),
		},
		exec: func(a *App, args map[string]any) (any, error) {
			return map[string]any{"renomeado": true}, renomearCarpetaCatalogo(argTxt(args, "id"), argTxt(args, "nome"))
		},
	},
	{
		def: IAFerramenta{
			Nome: "catalogo_cartafol_eliminar", Tipo: tipoConfirmar, Onde: "backend",
			Descricion: "Elimina un cartafol do catálogo (id de catalogo_ler). Os formularios NON se borran: quedan sen cartafol. Pide confirmación ao usuario.",
			Parametros: esqObx(map[string]any{"id": esqTxt("Id do cartafol.")}, "id"),
		},
		exec: func(a *App, args map[string]any) (any, error) {
			return map[string]any{"eliminado": true}, eliminarCarpetaCatalogo(argTxt(args, "id"))
		},
	},
}

func init() { ferramentasBackend = append(ferramentasBackend, ferramentasCatalogo...) }
