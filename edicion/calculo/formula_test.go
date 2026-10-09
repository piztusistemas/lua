package calculo

import (
	"math"
	"testing"
)

func TestAvaliar(t *testing.T) {
	casos := map[string]float64{
		"a*0.4 + b*0.6":            7.8,
		"se(a >= 5, 1, 0)":         1,
		"round(10/3, 2)":           3.33,
		"max(a, b, 2) - min(a, b)": 3,
		"media(a, b)":              7.5,
	}
	for f, esperado := range casos {
		v, err := Avaliar(f, map[string]float64{"a": 6, "b": 9 - 1 + 1})
		if err != nil || math.Abs(v-esperado) > 1e-9 {
			t.Errorf("%s = %v, %v (esperaba %v)", f, v, err, esperado)
		}
	}
	if _, err := Avaliar("a/0", map[string]float64{"a": 1}); err == nil {
		t.Error("división por cero sen erro")
	}
}
