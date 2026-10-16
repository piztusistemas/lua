package plantilla

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNormalizarDeseno(t *testing.T) {
	if NormalizarDeseno(nil) != nil {
		t.Fatal("nil segue sendo nil")
	}
	d := NormalizarDeseno(&Deseno{
		Letra: "inventada", LetraCorpo: "Georgia", LetraTitulos: `X}\input{y`, LetraMates: "pagella",
		Tamano: 50, Escala: 0.1, Interlinado: 1.3, Marxe: -5,
		CorPrincipal: "#1d4ed8", CorEscura: "verde", CorTexto: "abc",
		Etiquetas: "neon", Barras: Barras{Formal: 9, Aire: -9},
		Elementos: map[string]Elemento{
			"titulos":    {Cor: "#ff0000", Escala: 9},
			"inventado":  {Cor: "FF0000"},
			"etiquetas":  {},
			"trimestres": {Maiusculas: true},
		},
		Fixados: []string{"letra", "letra", "outro"},
	})
	if d.Version != VersionDeseno {
		t.Errorf("versión: %d", d.Version)
	}
	if d.Letra != "" || d.LetraCorpo != "Georgia" || d.LetraTitulos != "" || d.LetraMates != "pagella" {
		t.Errorf("letras: %+v", d)
	}
	if d.Tamano != 20 || d.Escala != 0.6 || d.Interlinado != 1.3 || d.Marxe != 8 {
		t.Errorf("rangos: %+v", d)
	}
	if d.CorPrincipal != "1D4ED8" || d.CorEscura != "" || d.CorTexto != "AABBCC" {
		t.Errorf("cores: %q %q %q", d.CorPrincipal, d.CorEscura, d.CorTexto)
	}
	if d.Etiquetas != "" || d.Barras.Formal != 2 || d.Barras.Aire != -2 {
		t.Errorf("etiquetas/barras: %+v", d)
	}
	if len(d.Elementos) != 2 || d.Elementos["titulos"].Cor != "FF0000" || d.Elementos["titulos"].Escala != 2.5 || !d.Elementos["trimestres"].Maiusculas {
		t.Errorf("elementos: %+v", d.Elementos)
	}
	if len(d.Fixados) != 1 || d.Fixados[0] != "letra" {
		t.Errorf("fixados: %v", d.Fixados)
	}
}

// TestEsquemaDeseno: o esquema é JSON válido e cobre todos os campos que
// serializa o tipo (se alguén engade un campo e esquece o esquema, a IA non o
// coñecería).
func TestEsquemaDeseno(t *testing.T) {
	esq := EsquemaDeseno()
	if _, err := json.Marshal(esq); err != nil {
		t.Fatal(err)
	}
	props := esq["properties"].(map[string]any)
	cheo := Deseno{Letra: "x", LetraCorpo: "x", LetraTitulos: "x", LetraEtiquetas: "x", LetraMates: "x", Tamano: 1, Escala: 1,
		Interlinado: 1, Parrafo: 1, Marxe: 1, CorPrincipal: "x", CorEscura: "x", CorTexto: "x", Etiquetas: "x",
		BrancoNegro: true, AforrarPapel: true, LecturaFacil: true, Elementos: map[string]Elemento{"x": {}}, Fixados: []string{"x"},
		Portada: "x", LogoPortada: "x", Cabeceira: Zonas{Esquerda: "x"}, Pe: Zonas{Dereita: "x"}, NumeroPaxina: "x"}
	data, _ := json.Marshal(cheo)
	var campos map[string]any
	_ = json.Unmarshal(data, &campos)
	for k := range campos {
		if k == "version" || k == "barras" {
			continue // internos: a IA non os toca
		}
		if _, ok := props[k]; !ok {
			t.Errorf("o esquema non ten %q", k)
		}
	}
}

func TestPlantillaLevaDesenoEnJSON(t *testing.T) {
	p := Plantilla{Nome: "x", Deseno: &Deseno{Letra: "clasica"}}
	data, _ := json.Marshal(p)
	var volta Plantilla
	if err := json.Unmarshal(data, &volta); err != nil || volta.Deseno == nil || volta.Deseno.Letra != "clasica" {
		t.Fatalf("o deseño non fai a viaxe de ida e volta: %s", data)
	}
	b, err := ConstruirBundle(p, Opcions{})
	if err != nil || b.Plantilla.Deseno == nil || b.Plantilla.Deseno.Letra != "clasica" {
		t.Fatalf("o deseño ten que viaxar ao exportar: %+v %v", b.Plantilla.Deseno, err)
	}
}

func TestNormalizarPortadaECabeceira(t *testing.T) {
	d := NormalizarDeseno(&Deseno{Portada: "barroca", LogoPortada: "centro", NumeroPaxina: "arriba",
		Cabeceira: Zonas{Esquerda: "  @logo {{CENTRO}}  ", Dereita: strings.Repeat("a", 300)}})
	if d.Portada != "" || d.LogoPortada != "centro" || d.NumeroPaxina != "" {
		t.Errorf("enums: %+v", d)
	}
	if d.Cabeceira.Esquerda != "@logo {{CENTRO}}" || len([]rune(d.Cabeceira.Dereita)) != 200 || !d.Pe.Baleira() {
		t.Errorf("zonas: %+v", d.Cabeceira)
	}
}
