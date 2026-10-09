package xerar

import "edicion/calculo"

func avaliarFormula(expr string, vars map[string]float64) (float64, error) {
	return calculo.Avaliar(expr, vars)
}
