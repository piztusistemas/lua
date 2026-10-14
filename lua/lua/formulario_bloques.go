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
//   - un texto longo é só o seu Markdown, unha imaxe só a imaxe e unha táboa
//     a cabeceira e as filas con algo escrito: a etiqueta do campo é para o
//     formulario, non sae no documento.
//
// O deseño (cores, tipos de letra, filetes) non está aquí: vai nos macros
// \form* que define a plantilla (ver formulario_latex.go).

import (
	"strings"

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
			bloques = append(bloques, bloqueDoc{Tipo: bloqueMarkdown, Texto: textoValor(v)})
		case campoImaxe:
			pecharFicha()
			if valorVacio(c, v) {
				continue
			}
			bloques = append(bloques, bloqueDoc{Tipo: bloqueImaxe, Imaxe: textoValor(v)})
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
			b := bloqueDoc{Tipo: bloqueTaboa}
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
