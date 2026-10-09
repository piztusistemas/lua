package boveda

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"edicion/ficha"
	"edicion/lectura"
)

func TestBovedaFichas(t *testing.T) {
	b, err := Crear(t.TempDir(), "Matemáticas")
	if err != nil {
		t.Fatal(err)
	}
	if d, _ := os.ReadFile(filepath.Join(b.Raiz, FicheiroEsquema)); !strings.Contains(string(d), "Matemáticas") {
		t.Error("buxo.md sen a materia")
	}
	f, err := b.Nova(ficha.TipoExercicio, "Área do triángulo")
	if err != nil || f.ID != "area-do-triangulo" {
		t.Fatalf("%v %v", f, err)
	}
	g, _ := b.Nova(ficha.TipoExercicio, "Área do triángulo")
	if g.ID != "area-do-triangulo-2" {
		t.Errorf("id repetido: %s", g.ID)
	}
	// Unha nota normal de Obsidian non é unha ficha.
	_ = os.WriteFile(filepath.Join(b.Raiz, "wiki", "nota.md"), []byte("# Nota\n"), 0o644)
	l, err := b.Fichas()
	if err != nil || len(l) != 2 || l[0].Titulo != "Área do triángulo" {
		t.Fatalf("%+v %v", l, err)
	}
	lida, mod, err := b.Ler("area-do-triangulo")
	if err != nil {
		t.Fatal(err)
	}
	lida.Corpo = "## Enunciado\n\nCambiado\n"
	mod2, err := b.Gardar(lida, mod)
	if err != nil {
		t.Fatal(err)
	}
	// Outro cambia o ficheiro por fóra: o seguinte gardado non pode pisalo.
	time.Sleep(20 * time.Millisecond)
	ruta, _ := b.Ruta("area-do-triangulo")
	_ = os.WriteFile(ruta, []byte("---\nluayang: 1\ntipo: exercicio\n---\n\nObsidian\n"), 0o644)
	if _, err := b.Gardar(lida, mod2); !errors.Is(err, ErrCambiouNoDisco) {
		t.Errorf("esperaba ErrCambiouNoDisco, deu %v", err)
	}
	if err := b.Eliminar("area-do-triangulo-2"); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Ruta("area-do-triangulo-2"); err == nil {
		t.Error("a ficha eliminada segue aí")
	}
	if _, err := b.Ruta("../../etc/passwd"); err == nil {
		t.Error("ruta fóra da bóveda")
	}
}

func TestGardarImportacionRenomea(t *testing.T) {
	b, _ := Crear(t.TempDir(), "")
	texto := "<EX>uno</EX>\n<EX>dous</EX>\n"
	for i := 0; i < 2; i++ {
		imp, err := ficha.ImportarMatex(texto, "exame")
		if err != nil {
			t.Fatal(err)
		}
		ids, err := b.GardarImportacion(imp)
		if err != nil {
			t.Fatal(err)
		}
		if i == 1 && (ids[0] == "exame" || ids[1] == "exame-1") {
			t.Errorf("non se renomeou: %v", ids)
		}
		ex, _, err := b.Ler(ids[0])
		if err != nil {
			t.Fatal(err)
		}
		for _, id := range ids[1:] {
			if !strings.Contains(ex.Corpo, "[["+id+"]]") {
				t.Errorf("o exame non liga %s: %q", id, ex.Corpo)
			}
		}
	}
}

func TestGardarLectura(t *testing.T) {
	b, _ := Crear(t.TempDir(), "")
	tmp := t.TempDir()
	pdf := filepath.Join(tmp, "Exame Química.pdf")
	_ = os.WriteFile(pdf, []byte("%PDF-1.4 proba"), 0o644)
	_ = os.MkdirAll(filepath.Join(tmp, "imaxes"), 0o755)
	_ = os.WriteFile(filepath.Join(tmp, "imaxes", "exame-quimica-p2-f1.png"), []byte("png"), 0o644)
	pax := []lectura.PaxinaLida{{Num: 1, Metodo: "texto", Markdown: "Instrucións", Cobertura: -1}, {Num: 2, Metodo: "vision", Markdown: "**3.2.3.** Complete ![molécula](raw/imaxes/exame-quimica-p2-f1.png)", Cobertura: 0.97}}
	e := lectura.Estrutura{Titulo: "Química", Materia: "Química", Linguas: []string{"gl", "es"}, Instrucions: "Instrucións",
		Exercicios: []lectura.ExercicioLido{{Titulo: "Pregunta 3", Apartado: "3", Puntos: 2.5, Paxina: 2, Enunciado: "**3.2.3.** Complete ![molécula](raw/imaxes/exame-quimica-p2-f1.png)", Outras: map[string]string{"es": "**3.2.3.** Complete"}, Conceptos: []string{"Química orgánica"}}}}
	r, err := b.GardarLectura(pdf, pax, e, tmp)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"pdf/exame-quimica.pdf", "raw/exame-quimica.md", "raw/imaxes/exame-quimica-p2-f1.png"} {
		if _, err := os.Stat(filepath.Join(b.Raiz, f)); err != nil {
			t.Errorf("falta %s", f)
		}
	}
	ex, _, err := b.Ler(r.Exercicios[0])
	if err != nil {
		t.Fatal(err)
	}
	ss := ex.Seccions()
	if ex.Puntos != 2.5 || ex.Conceptos[0] != "[[Química orgánica]]" || ss[1].Lingua != "es" || ex.Fonte.Paxina != 2 {
		t.Errorf("%+v %+v", ex, ss)
	}
	exame, _, _ := b.Ler(r.Exame)
	if !strings.Contains(exame.Corpo, "![["+r.Exercicios[0]+"]]") || !strings.Contains(exame.Corpo, "Instrucións") {
		t.Errorf("exame: %q", exame.Corpo)
	}
}

func timeZero() time.Time { return time.Time{} }
