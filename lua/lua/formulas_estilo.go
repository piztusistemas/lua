package main

// Tamaño e estilo das fórmulas nos documentos (Opcións → «Fórmulas nos
// documentos»). Unha fracción dentro dunha liña compóñena LaTeX en tamaño
// «de texto», moi pequena; o docente pode escoller:
//
//   - estilo: compactas (o de sempre) ou a tamaño completo (\displaystyle,
//     coma \dfrac), aínda que vaian no medio dunha liña;
//   - tamaño respecto ao texto: do 100 % ao 150 %. O texto non cambia.
//
// Cada fórmula escríbese coma \luaformula{…} ou \luaformulacentrada{…}
// (latexTramosCrono) e as macros defínense ao principio do corpo, así chegan
// tamén cunha plantilla con \documentclass propio. A escala vai con
// \scalebox, que non depende das letras nin do motor; sen graphicx,
// ignórase. Word e OpenDocument levan as fórmulas coma imaxes compiladas
// por LaTeX (formulas_ofi.go): mesmo estilo e a imaxe escalada.

import (
	"fmt"
	"strconv"
)

const (
	formulasCompactas = "compactas"
	formulasGrandes   = "grandes"
)

// opcionsFormulas: o que se escolleu en Opcións. Escala en %, 100–150.
type opcionsFormulas struct {
	Grandes bool
	Escala  int
}

// normalizar leva a escala ao rango e a pasos de 10 (0 = 100 %).
func (o opcionsFormulas) normalizar() opcionsFormulas {
	o.Escala = (o.Escala + 5) / 10 * 10
	o.Escala = max(100, min(150, o.Escala))
	return o
}

func opcionsFormulasDe(s Settings) opcionsFormulas {
	return opcionsFormulas{Grandes: s.FormulasEstilo == formulasGrandes, Escala: s.FormulasEscala}.normalizar()
}

// factor: a escala coma número para LaTeX («1.2»).
func (o opcionsFormulas) factor() string {
	return strconv.FormatFloat(float64(o.normalizar().Escala)/100, 'f', -1, 64)
}

// latex: as definicións de \luaformula e \luaformulacentrada.
func (o opcionsFormulas) latex() string {
	o = o.normalizar()
	enLiña := "$#1$"
	if o.Grandes {
		enLiña = `$\displaystyle #1$`
	}
	if o.Escala == 100 {
		return fmt.Sprintf("\\def\\luaformula#1{%s}\n\\def\\luaformulacentrada#1{\\[#1\\]}\n", enLiña)
	}
	f := o.factor()
	return fmt.Sprintf("\\def\\luaformula#1{\\ifdefined\\scalebox\\scalebox{%s}{%s}\\else%s\\fi}\n"+
		"\\def\\luaformulacentrada#1{\\ifdefined\\scalebox\\[\\scalebox{%s}{$\\displaystyle #1$}\\]\\else\\[#1\\]\\fi}\n",
		f, enLiña, enLiña, f)
}

// formulasDefLatexPorDefecto: para os documentos que non reciben as Opcións
// (cronograma, actas): as fórmulas coma sempre.
var formulasDefLatexPorDefecto = opcionsFormulas{}.latex()

// opcionsFormulas: as das Opcións do usuario.
func (a *App) opcionsFormulas() opcionsFormulas {
	return opcionsFormulasDe(a.settings)
}
