package main

// Motor de formularios (capa 3: o FLUXO, nesta primeira fase o cableado
// automático). Convirte un formulario e os seus datos nunha lista de BLOQUES
// de documento: unha representación intermedia sen formato que consumen os
// escritores (LaTeX, Markdown; despois Word e OpenDocument). O fluxo avalíase
// aquí, en Go, e só aquí: a vista previa, as exportacións e o asistente ven
// sempre o mesmo resultado.
//
// Regras do cableado automático (o que fai un formulario sen tocar as
// tuberías):
//
//   - un campo «sección» é un título de nivel 2;
//   - os campos simples (texto, número, data, escolla, verificación) seguidos
//     agrúpanse nunha FICHA de pares «etiqueta: valor»; os baleiros omítense;
//   - un texto longo é un título de nivel 3 coa etiqueta e o Markdown debaixo;
//   - unha imaxe leva a etiqueta de pé; unha táboa, a etiqueta de título e as
//     filas con algo escrito;
//   - unha etiqueta sen cambiar (o nome do tipo: «Texto longo») non sae
//     (ver etiquetaDoc).
//
// O deseño (cores, tipos de letra, filetes) non está aquí: vai nos macros
// \form* que define a plantilla (ver formulario_latex.go).

import (
	"encoding/json"
	"strings"
	"sync"

	"lua-wails/grafico"
)

// Tipos de bloque.
const (
	bloqueTitulo   = "titulo"
	bloqueFicha    = "ficha"
	bloqueMarkdown = "markdown"
	bloqueTaboa    = "taboa"
	bloqueImaxe    = "imaxe"
	bloqueLista    = "lista" // viñetas: Elementos
	bloqueSalto    = "salto" // páxina nova
	// bloqueGrafico (ver formulario_grafico.go): Grafico é a escena xa xerada.
)

// bloqueDoc é un bloque do documento xerado.
type bloqueDoc struct {
	Tipo  string
	Nivel int    // títulos: 2 ou 3 (o 1 é o título do documento)
	Texto string // título, Markdown ou pé da imaxe
	// Ficha: pares etiqueta / valor.
	Pares [][2]string
	// Táboa: cabeceira e filas xa formatadas.
	Cabeceira []string
	Filas     [][]string
	// Imaxe: data URI.
	Imaxe string
	// Lista: os elementos.
	Elementos []string
	// Gráfico: a escena (Pares leva categoría e valor para a vista previa).
	Grafico *grafico.Escena
	// Problemas (bloqueProblemas, ver formulario_problemas.go): as preguntas
	// dun campo e cantas hai antes nos campos anteriores (o índice de cada
	// pregunta, que dá a súa semente).
	Problemas     []ProblemaFormulario
	Desprazamento int
	// Rotulo: o da solución (bloqueSolucion): «Solución» ou o número.
	Rotulo string
}

// bloquesFormulario aplica o cableado automático. Devolve os bloques e os
// avisos dos datos (ver avisosDatos).
func bloquesFormulario(f Formulario, d DatosFormulario, tx textosDocumento) ([]bloqueDoc, []string) {
	d = d.aplicarReferencias()
	if f.Fluxo != nil {
		return bloquesDoFluxo(f, d, tx)
	}
	avisos := avisosDatos(f, d, tx)
	desprazamentos := desprazamentosProblemas(f, d)
	var bloques []bloqueDoc
	var ficha [][2]string
	pecharFicha := func() {
		if len(ficha) > 0 {
			bloques = append(bloques, bloqueDoc{Tipo: bloqueFicha, Pares: ficha})
			ficha = nil
		}
	}
	for _, c := range f.Campos {
		if c.Agochado {
			continue
		}
		v := d.Valores[c.ID]
		switch c.Tipo {
		case campoSeccion:
			pecharFicha()
			bloques = append(bloques, bloqueDoc{Tipo: bloqueTitulo, Nivel: 2, Texto: c.Etiqueta})
		case campoTextoLongo:
			pecharFicha()
			if valorVacio(c, v) {
				continue
			}
			if t := etiquetaDoc(c); t != "" {
				bloques = append(bloques, bloqueDoc{Tipo: bloqueTitulo, Nivel: 3, Texto: t})
			}
			bloques = append(bloques, bloqueDoc{Tipo: bloqueMarkdown, Texto: textoValor(v)})
		case campoImaxe:
			pecharFicha()
			if valorVacio(c, v) {
				continue
			}
			bloques = append(bloques, bloqueDoc{Tipo: bloqueImaxe, Texto: etiquetaDoc(c), Imaxe: textoValor(v)})
		case campoProblemas:
			pecharFicha()
			if valorVacio(c, v) {
				continue
			}
			bloques = append(bloques, bloqueDoc{Tipo: bloqueProblemas, Texto: c.Etiqueta, Problemas: problemasDe(v), Desprazamento: desprazamentos[c.ID]})
		case campoTaboa:
			pecharFicha()
			filas := filasNonVacias(c, v)
			if len(filas) == 0 {
				continue
			}
			b := bloqueDoc{Tipo: bloqueTaboa, Texto: etiquetaDoc(c)}
			for _, col := range c.Columnas {
				b.Cabeceira = append(b.Cabeceira, col.Etiqueta)
			}
			for _, fila := range filas {
				linha := make([]string, len(c.Columnas))
				for j, col := range c.Columnas {
					linha[j], _ = celaFormatada(col, fila[col.ID], tx)
				}
				b.Filas = append(b.Filas, linha)
			}
			if b.Texto != "" {
				bloques = append(bloques, bloqueDoc{Tipo: bloqueTitulo, Nivel: 3, Texto: b.Texto})
			}
			bloques = append(bloques, b)
		default:
			if valorVacio(c, v) {
				continue
			}
			s, _ := celaFormatada(c, v, tx)
			ficha = append(ficha, [2]string{c.Etiqueta, s})
		}
	}
	pecharFicha()
	if len(bloques) == 0 {
		if t := strings.TrimSpace(tx.T("FORM_SEN_CONTIDO")); t != "" {
			bloques = append(bloques, bloqueDoc{Tipo: bloqueMarkdown, Texto: t})
		}
	}
	return bloques, avisos
}

// etiquetaDoc é a etiqueta dun campo tal como sae no documento (título dun
// texto longo ou dunha táboa, pé dunha imaxe). Unha etiqueta sen cambiar —o
// nome do tipo que pon o deseñador ao crear o campo, «Texto longo», en
// calquera idioma— non di nada do contido: non sae.
func etiquetaDoc(c CampoFormulario) string {
	t := strings.TrimSpace(c.Etiqueta)
	if t == "" || nomesTipoPorDefecto()[c.Tipo][strings.ToLower(t)] {
		return ""
	}
	return t
}

// nomesTipoPorDefecto: para cada tipo, os nomes que lle dá o deseñador
// (formularios.tipo.<tipo> nos dicionarios embebidos), en minúsculas.
var nomesTipoPorDefecto = sync.OnceValue(func() map[string]map[string]bool {
	nomes := map[string]map[string]bool{}
	for _, codigo := range []string{"gl", "es", "en", "pt"} {
		datos, err := idiomasFS.ReadFile("idiomas/" + codigo + ".json")
		if err != nil {
			continue
		}
		var dicionario map[string]string
		if json.Unmarshal(datos, &dicionario) != nil {
			continue
		}
		for _, tipo := range tiposCampoFormulario {
			if n := strings.TrimSpace(dicionario["formularios.tipo."+tipo]); n != "" {
				if nomes[tipo] == nil {
					nomes[tipo] = map[string]bool{}
				}
				nomes[tipo][strings.ToLower(n)] = true
			}
		}
	}
	return nomes
})
