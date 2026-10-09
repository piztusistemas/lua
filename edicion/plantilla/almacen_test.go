package plantilla

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// depositoProbas arma un Depósito sobre un cartafol temporal, coa fábrica e as
// obsoletas que lle pase cada proba.
func depositoProbas(t *testing.T, fabrica func() []Plantilla, obsoleta map[string]string) *Deposito {
	t.Helper()
	base := t.TempDir()
	return &Deposito{
		RutaAlmacen:     func() (string, error) { return filepath.Join(base, "plantillas.json"), nil },
		DirImaxes:       func(id string) (string, error) { return filepath.Join(base, "plantillas", id), nil },
		Fabrica:         fabrica,
		FabricaObsoleta: obsoleta,
		ExisteFicheiro:  func(string) bool { return true },
	}
}

func plantillaFabrica(id, nome string) Plantilla {
	return Plantilla{ID: id, Nome: nome, Latex: "{{CORPO}} de " + nome, DeFabrica: true}
}

func fabricaDeUn() func() []Plantilla {
	return func() []Plantilla { return []Plantilla{plantillaFabrica("fabrica-un", "Un")} }
}

// TestDepositoSementa: cun ficheiro inexistente, Ler créao coas plantillas de
// fábrica e coa versión de formato actual.
func TestDepositoSementa(t *testing.T) {
	d := depositoProbas(t, fabricaDeUn(), nil)
	alm, err := d.Ler()
	if err != nil {
		t.Fatalf("Ler: %v", err)
	}
	if len(alm.Plantillas) != 1 || alm.Plantillas[0].ID != "fabrica-un" {
		t.Fatalf("non se sementou a fábrica: %+v", alm.Plantillas)
	}
	if !alm.Sementado {
		t.Error("o almacén debería quedar marcado como sementado")
	}
	ruta, _ := d.RutaAlmacen()
	gardado, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatalf("o ficheiro non se creou: %v", err)
	}
	var cru map[string]any
	if err := json.Unmarshal(gardado, &cru); err != nil {
		t.Fatal(err)
	}
	if v, _ := cru["version"].(float64); int(v) != VersionAlmacen {
		t.Errorf("version = %v, esperaba %d", cru["version"], VersionAlmacen)
	}

	// Ler outra vez non duplica nada.
	alm2, err := d.Ler()
	if err != nil {
		t.Fatal(err)
	}
	if len(alm2.Plantillas) != 1 {
		t.Errorf("a segunda lectura duplicou a fábrica: %+v", alm2.Plantillas)
	}
}

// TestDepositoMeterBifurcaFabrica: gardar por riba dunha de fábrica non a pisa,
// crea unha copia propia e deixa a orixinal intacta.
func TestDepositoMeterBifurcaFabrica(t *testing.T) {
	d := depositoProbas(t, fabricaDeUn(), nil)
	if _, err := d.Ler(); err != nil {
		t.Fatal(err)
	}

	p := plantillaFabrica("fabrica-un", "Un")
	p.Nome = "Editada polo profesorado"
	nova, err := d.Meter(p)
	if err != nil {
		t.Fatalf("Meter: %v", err)
	}
	if nova.ID == "fabrica-un" || nova.DeFabrica {
		t.Fatalf("debería bifurcar a unha plantilla propia: %+v", nova)
	}
	if nova.Nome != "Editada polo profesorado" {
		t.Errorf("perdeuse o nome editado: %+v", nova)
	}

	alm, err := d.Ler()
	if err != nil {
		t.Fatal(err)
	}
	if len(alm.Plantillas) != 2 {
		t.Fatalf("esperaba a de fábrica + a copia: %+v", alm.Plantillas)
	}
	for _, x := range alm.Plantillas {
		if x.ID == "fabrica-un" && (x.Nome != "Un" || !x.DeFabrica) {
			t.Errorf("a de fábrica quedou modificada: %+v", x)
		}
	}
}

// TestDepositoMeterCreaEActualiza: cunha ID baleira crea; cunha ID existente
// actualiza conservando Creado.
func TestDepositoMeterCreaEActualiza(t *testing.T) {
	d := depositoProbas(t, nil, nil)
	nova, err := d.Meter(Plantilla{Nome: "Miña", Latex: "{{CORPO}}"})
	if err != nil {
		t.Fatalf("Meter (crear): %v", err)
	}
	if nova.ID == "" || nova.Creado == "" {
		t.Fatalf("a plantilla nova quedou sen ID ou sen data: %+v", nova)
	}

	nova.Latex = "{{CORPO}} cambiado"
	outra, err := d.Meter(nova)
	if err != nil {
		t.Fatalf("Meter (actualizar): %v", err)
	}
	if outra.ID != nova.ID || outra.Creado != nova.Creado {
		t.Errorf("actualizar debería conservar ID e Creado: %+v vs %+v", outra, nova)
	}
	alm, _ := d.Ler()
	if len(alm.Plantillas) != 1 {
		t.Errorf("actualizar non pode duplicar: %+v", alm.Plantillas)
	}
}

// TestDepositoEliminarFabricaNonRevive: unha de fábrica borrada a propósito non
// pode reaparecer na seguinte lectura.
func TestDepositoEliminarFabricaNonRevive(t *testing.T) {
	d := depositoProbas(t, fabricaDeUn(), nil)
	if _, err := d.Ler(); err != nil {
		t.Fatal(err)
	}
	if err := d.Eliminar("fabrica-un"); err != nil {
		t.Fatalf("Eliminar: %v", err)
	}
	alm, err := d.Ler()
	if err != nil {
		t.Fatal(err)
	}
	if len(alm.Plantillas) != 0 {
		t.Fatalf("a plantilla de fábrica borrada reviviu: %+v", alm.Plantillas)
	}
	if len(alm.FabricaBorradas) != 1 || alm.FabricaBorradas[0] != "fabrica-un" {
		t.Errorf("non se anotou como borrada: %+v", alm.FabricaBorradas)
	}
}

// TestDepositoRetiraObsoleta: unha plantilla de fábrica vella que segue intacta
// e sen imaxes bórrase; se ten cambios ou imaxes, queda pero xa non coma de
// exemplo.
func TestDepositoRetiraObsoleta(t *testing.T) {
	latex := "{{CORPO}} vella"
	soma := sha256.Sum256([]byte(latex))
	obsoleta := map[string]string{"fabrica-vella": hex.EncodeToString(soma[:])[:16]}

	d := depositoProbas(t, fabricaDeUn(), obsoleta)
	base := t.TempDir()
	d.RutaAlmacen = func() (string, error) { return filepath.Join(base, "plantillas.json"), nil }
	d.DirImaxes = func(id string) (string, error) { return filepath.Join(base, "plantillas", id), nil }

	// Ficheiro cunha plantilla vella intacta e outra tocada.
	alm := Almacen{Sementado: true, Plantillas: []Plantilla{
		{ID: "fabrica-vella", Nome: "Vella", Latex: latex, DeFabrica: true},
		{ID: "fabrica-tocada", Nome: "Tocada", Latex: "outro", DeFabrica: true},
	}}
	alm.FabricaBorradas = nil
	if err := d.Gardar(alm); err != nil {
		t.Fatal(err)
	}
	// A "tocada" non está na táboa de obsoletas, así que se conserva.
	post, err := d.RetirarFabricaObsoleta(alm)
	if err != nil {
		t.Fatal(err)
	}
	if len(post.Plantillas) != 1 || post.Plantillas[0].ID != "fabrica-tocada" {
		t.Fatalf("a vella intacta debería retirarse: %+v", post.Plantillas)
	}

	// Cunha imaxe, a vella queda (pero xa non coma de exemplo).
	d2 := depositoProbas(t, fabricaDeUn(), obsoleta)
	base2 := t.TempDir()
	d2.RutaAlmacen = func() (string, error) { return filepath.Join(base2, "plantillas.json"), nil }
	d2.DirImaxes = func(id string) (string, error) { return filepath.Join(base2, "plantillas", id), nil }
	dir, _ := d2.DirImaxes("fabrica-vella")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "logo.png"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	alm2 := Almacen{Sementado: true, Plantillas: []Plantilla{
		{ID: "fabrica-vella", Nome: "Vella", Latex: latex, DeFabrica: true},
	}}
	post2, err := d2.RetirarFabricaObsoleta(alm2)
	if err != nil {
		t.Fatal(err)
	}
	if len(post2.Plantillas) != 1 || post2.Plantillas[0].DeFabrica {
		t.Fatalf("cunha imaxe debería quedar, sen ser de fábrica: %+v", post2.Plantillas)
	}
}

// TestDepositoReversionFutura: un plantillas.json feito por unha versión máis
// nova rexéitase con claridade.
func TestDepositoReversionFutura(t *testing.T) {
	d := depositoProbas(t, nil, nil)
	ruta, _ := d.RutaAlmacen()
	if err := os.WriteFile(ruta, []byte(`{"plantillas":[],"version":99}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Ler(); err == nil {
		t.Fatal("esperaba erro cunha versión de formato futura")
	}
}

// TestDepositoImaxes: copiar (incluídas as que non existen), listar e borrar.
func TestDepositoImaxes(t *testing.T) {
	d := depositoProbas(t, nil, nil)
	if len(d.Imaxes("ningunha")) != 0 {
		t.Error("unha plantilla sen cartafol non ten imaxes")
	}
	if err := d.CopiarImaxes("ningunha", "outra"); err != nil {
		t.Errorf("copiar sen orixe non pode ser erro: %v", err)
	}

	dirOrixe, _ := d.DirImaxes("orixe")
	if err := os.MkdirAll(dirOrixe, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, nome := range []string{"b.png", "a.png"} {
		if err := os.WriteFile(filepath.Join(dirOrixe, nome), []byte(nome), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := d.CopiarImaxes("orixe", "destino"); err != nil {
		t.Fatalf("CopiarImaxes: %v", err)
	}
	nomes := d.Imaxes("destino")
	if len(nomes) != 2 || nomes[0] != "a.png" || nomes[1] != "b.png" {
		t.Fatalf("as imaxes copiadas non son as agardadas (e ordenadas): %+v", nomes)
	}
	if err := d.BorrarImaxes("destino"); err != nil {
		t.Fatalf("BorrarImaxes: %v", err)
	}
	if len(d.Imaxes("destino")) != 0 {
		t.Error("as imaxes deberían quedar borradas")
	}
}

// TestDepositoEstado: a listaxe vai ordenada por nome, coas imaxes resoltas e a
// plantilla activa.
func TestDepositoEstado(t *testing.T) {
	d := depositoProbas(t, nil, nil)
	for _, nome := range []string{"Zebra", "alfa"} {
		if _, err := d.Meter(Plantilla{Nome: nome, Latex: "{{CORPO}}"}); err != nil {
			t.Fatal(err)
		}
	}
	alm, _ := d.Ler()
	alm.Activa = alm.Plantillas[0].ID
	if err := d.Gardar(alm); err != nil {
		t.Fatal(err)
	}
	estado, err := d.Estado()
	if err != nil {
		t.Fatalf("Estado: %v", err)
	}
	if len(estado.Plantillas) != 2 || estado.Plantillas[0].Nome != "alfa" || estado.Plantillas[1].Nome != "Zebra" {
		t.Fatalf("a listaxe non vai ordenada por nome: %+v", estado.Plantillas)
	}
	if estado.Activa != alm.Activa {
		t.Errorf("a plantilla activa perdeuse: %q vs %q", estado.Activa, alm.Activa)
	}
}

// TestDepositoFabricaFoiEditada: a comparación cobre todos os campos que o
// profesorado pode cambiar desde o editor (incluído Ofimatica e Textos) e
// ignora os metadatos que toca a propia aplicación (Creado, Modificado,
// Imaxes).
func TestDepositoFabricaFoiEditada(t *testing.T) {
	d := depositoProbas(t, nil, nil)
	base := Plantilla{
		ID: "fabrica-x", Nome: "X", Descricion: "d", Latex: "{{CORPO}}", Markdown: "{{CORPO}}",
		Portada: "P", Sinatura: "S", Introducion: "I", Estilo: "E",
		Textos: map[string]string{"SAUDO": "Ola"}, Motor: "xelatex",
		Variables: []Var{{Nome: "CENTRO", Valor: "IES"}}, DeFabrica: true,
	}
	if d.FabricaFoiEditada(base, base) {
		t.Error("unha plantilla idéntica non está editada")
	}

	// Metadatos que NON contan.
	metadatos := base
	metadatos.Creado, metadatos.Modificado = "2020", "2021"
	metadatos.Imaxes = []string{"logo.png"}
	if d.FabricaFoiEditada(metadatos, base) {
		t.Error("Creado/Modificado/Imaxes non poden contar coma edición")
	}

	// Campos que SI contan.
	casos := map[string]func(p *Plantilla){
		"nome":        func(p *Plantilla) { p.Nome = "outro" },
		"descricion":  func(p *Plantilla) { p.Descricion = "outra" },
		"latex":       func(p *Plantilla) { p.Latex = "outro" },
		"markdown":    func(p *Plantilla) { p.Markdown = "outro" },
		"portada":     func(p *Plantilla) { p.Portada = "outra" },
		"sinatura":    func(p *Plantilla) { p.Sinatura = "outra" },
		"introducion": func(p *Plantilla) { p.Introducion = "outra" },
		"estilo":      func(p *Plantilla) { p.Estilo = "outro" },
		"motor":       func(p *Plantilla) { p.Motor = "pdflatex" },
		"textos":      func(p *Plantilla) { p.Textos = map[string]string{"SAUDO": "Ola!"} },
		"ofimatica":   func(p *Plantilla) { p.Ofimatica.Pe = "pé" },
		"variables":   func(p *Plantilla) { p.Variables = []Var{{Nome: "CENTRO", Valor: "outro"}} },
	}
	for nome, muta := range casos {
		p := base
		muta(&p)
		if !d.FabricaFoiEditada(p, base) {
			t.Errorf("cambiar %s ten que contar coma edición", nome)
		}
	}

	// Co comparador de textos inxectado (o catálogo é de dominio), úsase ese.
	d.TextosIguais = func(a, b map[string]string) bool { return true }
	p := base
	p.Textos = map[string]string{"OUTRA": "cousa"}
	if d.FabricaFoiEditada(p, base) {
		t.Error("co comparador inxectado, os textos non deberían contar")
	}
}

// TestDepositoActivaEPorFicheiro: a escolla activa, a memoria "que plantilla
// usaba cada documento" e a súa resolución.
func TestDepositoActivaEPorFicheiro(t *testing.T) {
	d := depositoProbas(t, nil, nil)
	p, err := d.Meter(Plantilla{Nome: "Unha", Latex: "{{CORPO}}"})
	if err != nil {
		t.Fatal(err)
	}
	if d.Activa() != nil {
		t.Error("sen ningunha activa, Activa() ten que devolver nil")
	}

	if err := d.Activar(p.ID, "/tmp/curso.matex"); err != nil {
		t.Fatal(err)
	}
	if a := d.Activa(); a == nil || a.ID != p.ID {
		t.Fatalf("Activa() = %+v", a)
	}
	if id, err := d.ParaFicheiro("/tmp/curso.matex"); err != nil || id != p.ID {
		t.Fatalf("ParaFicheiro lembrado = %q, %v", id, err)
	}
	// Un documento descoñecido mantén a activa actual.
	if id, err := d.ParaFicheiro("/tmp/outro.matex"); err != nil || id != p.ID {
		t.Errorf("un documento sen plantilla ten que manter a activa: %q, %v", id, err)
	}

	// Se a plantilla activa desaparece, xérase sen plantilla en vez de fallar.
	if err := d.Eliminar(p.ID); err != nil {
		t.Fatal(err)
	}
	if d.Activa() != nil {
		t.Error("tras eliminar a activa, Activa() ten que devolver nil")
	}
	if id, err := d.ParaFicheiro("/tmp/curso.matex"); err != nil || id != "" {
		t.Errorf("a plantilla lembrada xa non existe: %q, %v", id, err)
	}

	// Activar cun id baleiro esquece a escolla e a memoria do documento.
	q, err := d.Meter(Plantilla{Nome: "Dúas", Latex: "{{CORPO}}"})
	if err != nil {
		t.Fatal(err)
	}
	if err := d.Activar(q.ID, "/tmp/outro.matex"); err != nil {
		t.Fatal(err)
	}
	if err := d.Activar("", "/tmp/outro.matex"); err != nil {
		t.Fatal(err)
	}
	alm, _ := d.Ler()
	if alm.Activa != "" {
		t.Errorf("Activar(\"\") ten que deixar ningunha activa: %q", alm.Activa)
	}
	if _, hai := alm.PorFicheiro["/tmp/outro.matex"]; hai {
		t.Error("Activar(\"\") ten que esquecer a memoria dese documento")
	}
}
