package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestPlantillasJSONRealAbreEGardaSenPerda colle o plantillas.json REAL deste
// equipo (se existe), ábreo e vólveo gardar nun contorno illado, e comproba que
// non se perdeu nada: nin unha plantilla, nin os seus campos. Salta se non hai
// ficheiro real, que é o normal nunha máquina de integración.
//
// É a comprobación de compatibilidade da fase 2: o ficheiro que a xente xa ten
// no disco ten que seguir abrindo (agora co campo "version" engadido) e gardar
// sen perda.
func TestPlantillasJSONRealAbreEGardaSenPerda(t *testing.T) {
	real, err := os.UserConfigDir()
	if err != nil {
		t.Skipf("sen cartafol de configuración: %v", err)
	}
	orixe := filepath.Join(real, "lua", "plantillas.json")
	datos, err := os.ReadFile(orixe)
	if err != nil {
		t.Skipf("non hai plantillas.json real en %s: %v", orixe, err)
	}

	configIllado(t)
	destino := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "lua")
	if err := os.MkdirAll(destino, 0o755); err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(destino, "plantillas.json")
	if err := os.WriteFile(ruta, datos, 0o644); err != nil {
		t.Fatal(err)
	}

	alm1, err := loadAlmacenPlantillas()
	if err != nil {
		t.Fatalf("abrir o plantillas.json real: %v", err)
	}
	if len(alm1.Plantillas) == 0 {
		t.Fatal("o plantillas.json real quedou sen plantillas")
	}
	if err := saveAlmacenPlantillas(alm1); err != nil {
		t.Fatalf("gardar: %v", err)
	}
	alm2, err := loadAlmacenPlantillas()
	if err != nil {
		t.Fatalf("reabrir o ficheiro gardado: %v", err)
	}

	porID := map[string]PlantillaLatex{}
	for _, p := range alm2.Plantillas {
		porID[p.ID] = p
	}
	for _, p := range alm1.Plantillas {
		outra, ok := porID[p.ID]
		if !ok {
			t.Errorf("a plantilla %q (%s) desapareceu ao gardar", p.Nome, p.ID)
			continue
		}
		if outra.Nome != p.Nome || outra.Latex != p.Latex || outra.Markdown != p.Markdown ||
			outra.Motor != p.Motor || outra.Portada != p.Portada || outra.Sinatura != p.Sinatura ||
			outra.Introducion != p.Introducion || outra.Estilo != p.Estilo ||
			outra.Ofimatica != p.Ofimatica ||
			!reflect.DeepEqual(outra.Variables, p.Variables) || !reflect.DeepEqual(outra.Textos, p.Textos) {
			t.Errorf("a plantilla %q perdeu contido ao gardar", p.ID)
		}
	}

	// O ficheiro gardado ten que quedar xa coa versión de formato actual.
	gardado, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	var cru map[string]any
	if err := json.Unmarshal(gardado, &cru); err != nil {
		t.Fatalf("o ficheiro gardado non é JSON válido: %v", err)
	}
	if v, _ := cru["version"].(float64); int(v) != 1 {
		t.Errorf("o plantillas.json gardado debería levar version 1, levou %v", cru["version"])
	}
}
