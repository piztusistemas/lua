package main

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func pedidoFormularioProba(f Formulario) XerarInformeCronogramaRequest {
	return XerarInformeCronogramaRequest{Ferramenta: ferramentaFormulario, NomeBase: "exame",
		Formulario: &PedidoFormulario{Formulario: f, Datos: DatosFormulario{Formulario: f.ID, Valores: map[string]any{}}}}
}

func TestTraballoCreaABovedaDoProxectoESubstitue(t *testing.T) {
	trasladarTemporal(t)
	a := &App{}
	p, _ := crearProxecto("Mates", "", nil)
	_, _ = activarProxecto(p.ID)
	f := formularioDeProba(t, "Exame tipo test")
	pdf := base64.StdEncoding.EncodeToString([]byte("%PDF-1.4 primeiro"))
	res, err := a.EngadirTraballoBoveda(pedidoFormularioProba(f), pdf, "Exame tipo test")
	if err != nil {
		t.Fatal(err)
	}
	raiz, _ := rutaBovedaProxecto(p.ID)
	raiz, _ = filepath.EvalSymlinks(raiz)
	if !res.Creada || bovedaDoProxecto(p.ID) == "" {
		t.Fatalf("tiña que crear a bóveda do proxecto: %+v", res)
	}
	nota := filepath.Join(raiz, "wiki", "traballos", "Exame tipo test.md")
	if res.Nota != nota {
		t.Fatalf("nota en %q", res.Nota)
	}
	contido := ler(t, nota)
	if !strings.Contains(contido, "[[Exame tipo test.pdf|PDF]]") || !strings.Contains(contido, "tipo: traballo") {
		t.Fatalf("nota: %s", contido)
	}
	if ler(t, filepath.Join(raiz, "wiki", "traballos", "Exame tipo test.pdf")) != "%PDF-1.4 primeiro" {
		t.Fatal("falta o PDF")
	}
	if !strings.Contains(ler(t, filepath.Join(raiz, "wiki", "log.md")), "traballo | Exame tipo test") {
		t.Fatal("falta a liña no log")
	}
	// Segunda vez: substitúe (unha soa nota e un só PDF).
	pdf2 := base64.StdEncoding.EncodeToString([]byte("%PDF-1.4 segundo"))
	res2, err := a.EngadirTraballoBoveda(pedidoFormularioProba(f), pdf2, "Exame tipo test")
	if err != nil || res2.Creada {
		t.Fatalf("%+v %v", res2, err)
	}
	ents, _ := os.ReadDir(filepath.Join(raiz, "wiki", "traballos"))
	n := 0
	for _, e := range ents {
		if !e.IsDir() {
			n++
		}
	}
	if n != 2 || ler(t, filepath.Join(raiz, "wiki", "traballos", "Exame tipo test.pdf")) != "%PDF-1.4 segundo" {
		t.Fatalf("tiña que substituír: %d ficheiros", n)
	}
}

func TestTraballoVaiABovedaConectada(t *testing.T) {
	trasladarTemporal(t)
	externa := t.TempDir()
	for _, d := range []string{"raw", "wiki"} {
		_ = os.MkdirAll(filepath.Join(externa, d), 0o755)
	}
	a := &App{settings: Settings{IABovedas: map[string]string{"formulario": externa}}}
	p, _ := crearProxecto("Sen bóveda", "", nil)
	_, _ = activarProxecto(p.ID)
	f := formularioDeProba(t, "Boletín")
	res, err := a.EngadirTraballoBoveda(pedidoFormularioProba(f), "", "Boletín")
	if err != nil {
		t.Fatal(err)
	}
	real, _ := filepath.EvalSymlinks(externa)
	if res.Boveda != real || res.Creada || bovedaDoProxecto(p.ID) != "" {
		t.Fatalf("tiña que ir á bóveda conectada: %+v", res)
	}
	if _, err := os.Stat(filepath.Join(real, "wiki", "traballos", "Boletín.md")); err != nil {
		t.Fatal("falta a nota na bóveda conectada")
	}
}

func TestTraballoProduccionBloqueado(t *testing.T) {
	trasladarTemporal(t)
	a := &App{}
	p, _ := crearProxecto("Titoría", "", nil)
	_, _ = activarProxecto(p.ID)
	f := formularioDeProba(t, "Informe de avaliación")
	if _, err := cambiarModoFormulario(f.ID, modoProduccion); err != nil {
		t.Fatal(err)
	}
	// Aínda que o frontend o mande sen modo, mírase o gardado.
	f.Modo = ""
	if _, err := a.EngadirTraballoBoveda(pedidoFormularioProba(f), "", "Informe"); err == nil || err.Error() != erroProduccionBoveda().Error() { // no idioma activo
		t.Fatalf("produción tiña que bloquearse: %v", err)
	}
	if bovedaDoProxecto(p.ID) != "" {
		t.Fatal("non se pode crear nada na bóveda")
	}
}

func TestTraballoCronograma(t *testing.T) {
	trasladarTemporal(t)
	a := &App{}
	p, _ := crearProxecto("Dpto", "", nil)
	_, _ = activarProxecto(p.ID)
	res, err := a.EngadirTraballoBoveda(XerarInformeCronogramaRequest{Ferramenta: ferramentaCronograma, Plan: PlanCrono{AcademicYear: "2025/2026"}}, "", "Cronograma anual")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(res.Nota, filepath.Join("traballos", "Cronograma anual.md")) || !strings.Contains(ler(t, res.Nota), "documento: cronograma") {
		t.Fatalf("%+v", res)
	}
}
