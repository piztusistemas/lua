package ficha

// Importar formularios de Lua. Os campos pasan a ser campos «escrito» co
// mesmo id, tipo e axuda; os atributos do formulario van en `formulario`; e
// a tubería (fluxo) viaxa tal cal en `calculo` ata que o corpo da ficha a
// substitúa (fase 5 da especificación). FormularioLua devolve o mesmo JSON.

import (
	"encoding/json"
	"errors"
	"fmt"
)

type formularioLua struct {
	ID         string     `json:"id"`
	Nome       string     `json:"nome"`
	Icona      string     `json:"icona,omitempty"`
	Descricion string     `json:"descricion,omitempty"`
	Modo       string     `json:"modo,omitempty"`
	Titulo     string     `json:"titulo,omitempty"`
	Portada    bool       `json:"portada,omitempty"`
	Sinatura   bool       `json:"sinatura,omitempty"`
	Campos     []campoLua `json:"campos"`
	Fluxo      any        `json:"fluxo,omitempty"`
}

type campoLua struct {
	ID          string     `json:"id"`
	Tipo        string     `json:"tipo"`
	Etiqueta    string     `json:"etiqueta"`
	Axuda       string     `json:"axuda,omitempty"`
	Obrigatorio bool       `json:"obrigatorio,omitempty"`
	Opcions     []string   `json:"opcions,omitempty"`
	Columnas    []campoLua `json:"columnas,omitempty"`
	Agochado    bool       `json:"agochado,omitempty"`
}

// ImportarFormularioLua converte o JSON dun formulario de Lua nunha ficha.
func ImportarFormularioLua(datos []byte) (*Ficha, error) {
	var fl formularioLua
	if err := json.Unmarshal(datos, &fl); err != nil {
		return nil, fmt.Errorf("o formulario non é JSON válido: %w", err)
	}
	if fl.ID == "" {
		return nil, errors.New("o formulario non ten id")
	}
	f := &Ficha{
		Luayang:   Version,
		ID:        fl.ID,
		Tipo:      TipoFormulario,
		Titulo:    fl.Nome,
		Importado: &Importado{De: "lua", ID: fl.ID},
		Calculo:   fl.Fluxo,
	}
	op := OpcionsFormulario{Icona: fl.Icona, Descricion: fl.Descricion, Modo: fl.Modo, Titulo: fl.Titulo, Portada: fl.Portada, Sinatura: fl.Sinatura}
	if op != (OpcionsFormulario{}) {
		f.Formulario = &op
	}
	for _, c := range fl.Campos {
		f.Campos = append(f.Campos, campoDeLua(c, true))
	}
	return f, nil
}

func campoDeLua(c campoLua, conOrixe bool) Campo {
	out := Campo{Nome: c.ID, Tipo: c.Tipo, Etiqueta: c.Etiqueta, Axuda: c.Axuda, Obrigatorio: c.Obrigatorio, Opcions: c.Opcions, Agochado: c.Agochado}
	if conOrixe {
		out.Orixe = OrixeEscrito
	}
	for _, col := range c.Columnas {
		out.Columnas = append(out.Columnas, campoDeLua(col, false))
	}
	return out
}

func campoALua(c Campo) campoLua {
	out := campoLua{ID: c.Nome, Tipo: c.Tipo, Etiqueta: c.Etiqueta, Axuda: c.Axuda, Obrigatorio: c.Obrigatorio, Opcions: c.Opcions, Agochado: c.Agochado}
	for _, col := range c.Columnas {
		out.Columnas = append(out.Columnas, campoALua(col))
	}
	return out
}

// FormularioLua devolve o JSON de Lua dunha ficha formulario.
func FormularioLua(f *Ficha) ([]byte, error) {
	if f.Tipo != TipoFormulario {
		return nil, fmt.Errorf("a ficha é de tipo %s, non un formulario", f.Tipo)
	}
	fl := formularioLua{ID: f.ID, Nome: f.Titulo, Campos: []campoLua{}, Fluxo: f.Calculo}
	if f.Importado != nil && f.Importado.ID != "" {
		fl.ID = f.Importado.ID
	}
	if op := f.Formulario; op != nil {
		fl.Icona, fl.Descricion, fl.Modo, fl.Titulo, fl.Portada, fl.Sinatura = op.Icona, op.Descricion, op.Modo, op.Titulo, op.Portada, op.Sinatura
	}
	for _, c := range f.Campos {
		if c.Orixe != OrixeEscrito {
			return nil, fmt.Errorf("campo «%s»: Lua só ten campos escritos (este é %s)", c.Nome, c.Orixe)
		}
		fl.Campos = append(fl.Campos, campoALua(c))
	}
	return json.MarshalIndent(fl, "", "  ")
}
