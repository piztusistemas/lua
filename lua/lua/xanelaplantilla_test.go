package main

import "testing"

func TestNomeXanelaPlantilla(t *testing.T) {
	for entrada, quero := range map[string]string{
		"":              "plantilla-nova",
		"abc123":        "plantilla-abc123",
		"a/b c?d=1&e#f": "plantilla-a_b_c_d_1_e_f",
		"../../etc":     "plantilla-______etc",
	} {
		nome, _ := nomeXanelaPlantilla(entrada)
		if nome != quero {
			t.Errorf("nomeXanelaPlantilla(%q) = %q, esperaba %q", entrada, nome, quero)
		}
	}
}

func TestXanelaPlantillaSenAplicacionNonFalla(t *testing.T) {
	a := &App{}
	// Sen xanelas nativas (probas, modo servidor) devolve false e o frontend abre o overlay.
	if ok, err := a.AbrirXanelaPlantilla("p1", "{}"); ok || err != nil {
		t.Errorf("AbrirXanelaPlantilla sen aplicación: %v, %v", ok, err)
	}
	a.MarcarXanelaPlantillaSucia("p1", true)
	if !a.plantillas.sucias["p1"] {
		t.Error("non lembra que hai cambios sen aplicar")
	}
	a.MarcarXanelaPlantillaSucia("p1", false)
	a.plantillas.datos["p1"] = `{"x":1}`
	if a.GetDatosXanelaPlantilla("p1") != `{"x":1}` || a.GetDatosXanelaPlantilla("../p1") != "" {
		t.Error("os datos da xanela non se len por id limpo")
	}
	a.PlantillaXanelaAplicada("p1") // sen aplicación: non debe fallar
	a.PecharXanelaPlantilla("p1")
}
