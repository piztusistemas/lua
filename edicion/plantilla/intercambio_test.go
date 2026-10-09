package plantilla

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------- modelo ----------

func TestNormalizarNombreVar(t *testing.T) {
	casos := map[string]string{
		"centro":          "CENTRO",
		"  Centro  ":      "CENTRO",
		"curso académico": "CURSO_ACADÉMICO",
		"nome-do-curso":   "NOME_DO_CURSO",
		"a.b":             "AB", // o punto non é válido nun marcador
		"{{X}}":           "X",
		"":                "",
	}
	for entrada, esperado := range casos {
		if got := NormalizarNombreVar(entrada); got != esperado {
			t.Errorf("NormalizarNombreVar(%q) = %q, esperaba %q", entrada, got, esperado)
		}
	}
}

func TestNormalizarVariables(t *testing.T) {
	vars := []Var{
		{Nome: "centro", Valor: "IES Xistral"},
		{Nome: "CENTRO", Valor: "duplicada, gana a primeira"},
		{Nome: "corpo", Valor: "marcador reservado"},
		{Nome: "portada", Valor: "marcador reservado"},
		{Nome: "sinatura", Valor: "marcador reservado"},
		{Nome: "  ", Valor: "sen nome"},
		{Nome: "Curso Académico", Valor: "2026/2027"},
	}
	out := NormalizarVariables(vars)
	if len(out) != 2 {
		t.Fatalf("esperaba 2 variables, obtiven %d: %+v", len(out), out)
	}
	if out[0].Nome != "CENTRO" || out[0].Valor != "IES Xistral" {
		t.Errorf("a primeira variable non é a agardada: %+v", out[0])
	}
	if out[1].Nome != "CURSO_ACADÉMICO" {
		t.Errorf("a segunda variable non é a agardada: %+v", out[1])
	}
}

func TestVersionAlmacen(t *testing.T) {
	var a Almacen
	if err := a.ComprobarVersion(); err != nil {
		t.Errorf("un almacén sen versión (ficheiro vello) ten que abrir: %v", err)
	}
	a.MarcarVersion()
	if a.Version != VersionAlmacen {
		t.Errorf("MarcarVersion deixou %d, esperaba %d", a.Version, VersionAlmacen)
	}
	if err := a.ComprobarVersion(); err != nil {
		t.Errorf("un almacén coa versión actual ten que abrir: %v", err)
	}
	a.Version = VersionAlmacen + 1
	if err := a.ComprobarVersion(); err == nil {
		t.Error("un almacén dunha versión máis nova ten que rexeitarse")
	}
}

// ---------- intercambio ----------

// importadorFalso imita a parte da aplicación que o módulo necesita: valida
// como ela (nome e {{CORPO}}) e garda imaxes por plantilla.
type importadorFalso struct {
	gardadas []Plantilla
	imaxes   map[string][]string
	errSubir error
}

func novoImportadorFalso() *importadorFalso {
	return &importadorFalso{imaxes: map[string][]string{}}
}

func (i *importadorFalso) Gardar(p Plantilla) (Plantilla, error) {
	if strings.TrimSpace(p.Nome) == "" {
		return Plantilla{}, errors.New("dálle un nome á plantilla")
	}
	if !strings.Contains(p.Latex, MarcadorCorpo) {
		return Plantilla{}, fmt.Errorf("o LaTeX da plantilla ten que levar %s", MarcadorCorpo)
	}
	p.ID = fmt.Sprintf("id-%d", len(i.gardadas)+1)
	p.Imaxes = nil
	i.gardadas = append(i.gardadas, p)
	return p, nil
}

func (i *importadorFalso) Imaxes(id string) []string { return i.imaxes[id] }

func (i *importadorFalso) SubirImaxe(id, nome, dataB64 string) error {
	if i.errSubir != nil {
		return i.errSubir
	}
	i.imaxes[id] = append(i.imaxes[id], nome)
	return nil
}

func opcionsProbas(dirImaxes func(string) (string, error)) Opcions {
	return Opcions{
		Nome:      "Probex",
		DirImaxes: dirImaxes,
		NormalizarTextos: func(m map[string]string) map[string]string {
			if len(m) == 0 {
				return nil
			}
			out := map[string]string{}
			for k, v := range m {
				if strings.TrimSpace(v) != "" {
					out[strings.ToUpper(strings.TrimSpace(k))] = strings.TrimSpace(v)
				}
			}
			if len(out) == 0 {
				return nil
			}
			return out
		},
	}
}

// TestBundleRoundTrip: exportar unha plantilla (co seu logo) e importala nun
// equipo "limpo" ten que reproducila enteira - texto, variables e imaxes - pero
// coma plantilla NOVA (ID propia, non de fábrica).
func TestBundleRoundTrip(t *testing.T) {
	dirImaxes := t.TempDir()
	pngB64 := base64.StdEncoding.EncodeToString([]byte("\x89PNG\r\n\x1a\nfake"))
	if err := os.WriteFile(filepath.Join(dirImaxes, "logo.png"), []byte("\x89PNG\r\n\x1a\nfake"), 0o644); err != nil {
		t.Fatal(err)
	}

	orixe := Plantilla{
		ID:          "id-orixe",
		Nome:        "  Circular do centro  ",
		Descricion:  "Cabeceira do IES",
		Latex:       "{{CENTRO}}\n\n{{CORPO}}\n",
		Portada:     "PORTADA {{CENTRO}}",
		Sinatura:    "SINATURA {{XEFE}}",
		Introducion: "INTRO {{CURSO}}",
		Estilo:      "\\renewcommand{\\familydefault}{\\sfdefault}",
		Textos:      map[string]string{"saudo": "Ola", "baleiro": "   "},
		Markdown:    "# {{CENTRO}}\n\n{{CORPO}}\n",
		Motor:       "xelatex",
		Variables:   []Var{{Nome: "centro", Etiqueta: "Centro", Valor: "IES Xistral"}},
		Creado:      "2026-01-01T00:00:00Z",
		Modificado:  "2026-01-02T00:00:00Z",
		DeFabrica:   true,
	}

	bundle, err := ConstruirBundle(orixe, opcionsProbas(func(string) (string, error) { return dirImaxes, nil }))
	if err != nil {
		t.Fatalf("ConstruirBundle: %v", err)
	}
	if bundle.Formato != FormatoBundle || bundle.Version != VersionBundle {
		t.Fatalf("cabeceira do bundle inesperada: %+v", bundle)
	}
	if bundle.Plantilla.ID != "" || bundle.Plantilla.DeFabrica || bundle.Plantilla.Creado != "" || bundle.Plantilla.Modificado != "" {
		t.Errorf("o bundle non debe levar metadatos locais: %+v", bundle.Plantilla)
	}
	if bundle.Plantilla.Nome != "Circular do centro" {
		t.Errorf("o nome non se limpa: %q", bundle.Plantilla.Nome)
	}
	if len(bundle.Plantilla.Variables) != 1 || bundle.Plantilla.Variables[0].Nome != "CENTRO" {
		t.Errorf("as variables non se normalizan: %+v", bundle.Plantilla.Variables)
	}
	if bundle.Plantilla.Textos["SAUDO"] != "Ola" || len(bundle.Plantilla.Textos) != 1 {
		t.Errorf("os textos non se normalizan: %+v", bundle.Plantilla.Textos)
	}
	if len(bundle.Imaxes) != 1 || bundle.Imaxes[0].Nome != "logo.png" || bundle.Imaxes[0].DataB64 != pngB64 {
		t.Fatalf("a imaxe non viaxou no bundle: %+v", bundle.Imaxes)
	}

	data, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	// --- importar nun equipo limpo ---
	imp := novoImportadorFalso()
	gardada, err := ImportarBundle(data, opcionsProbas(func(string) (string, error) { return "", nil }), imp)
	if err != nil {
		t.Fatalf("ImportarBundle: %v", err)
	}
	if gardada.ID == "" || gardada.DeFabrica {
		t.Errorf("a plantilla importada ten que ser nova e propia: %+v", gardada)
	}
	if gardada.Nome != "Circular do centro" || gardada.Motor != "xelatex" {
		t.Errorf("perdéronse campos ao importar: %+v", gardada)
	}
	if gardada.Portada != "PORTADA {{CENTRO}}" || gardada.Sinatura != "SINATURA {{XEFE}}" || gardada.Introducion != "INTRO {{CURSO}}" {
		t.Errorf("a portada, a sinatura e a introdución non viaxaron: %+v", gardada)
	}
	if gardada.Estilo != orixe.Estilo {
		t.Errorf("o estilo non viaxou: %q", gardada.Estilo)
	}
	if len(gardada.Variables) != 1 || gardada.Variables[0].Nome != "CENTRO" {
		t.Errorf("as variables non se conservaron: %+v", gardada.Variables)
	}
	if len(gardada.Imaxes) != 1 || gardada.Imaxes[0] != "logo.png" {
		t.Errorf("a imaxe non se escribiu no equipo que importa: %+v", gardada.Imaxes)
	}
}

// TestImportarBundleAceptaMarcadoresHistoricos é a proba de compatibilidade
// entre aplicacións: o marcador neutro e os dous históricos ("yang-plantilla" e
// "lua-plantilla") teñen que abrir igual.
func TestImportarBundleAceptaMarcadoresHistoricos(t *testing.T) {
	for _, formato := range []string{FormatoBundle, "yang-plantilla", "lua-plantilla"} {
		data, err := json.Marshal(Bundle{
			Formato:   formato,
			Version:   VersionBundle,
			Plantilla: Plantilla{Nome: "Veña de onde veña", Latex: "{{CORPO}}"},
		})
		if err != nil {
			t.Fatal(err)
		}
		imp := novoImportadorFalso()
		gardada, err := ImportarBundle(data, opcionsProbas(nil), imp)
		if err != nil {
			t.Errorf("formato %q: %v", formato, err)
			continue
		}
		if gardada.Nome != "Veña de onde veña" {
			t.Errorf("formato %q: nome mal importado: %+v", formato, gardada)
		}
	}
}

func TestImportarBundleInvalido(t *testing.T) {
	marcar := func(v any) []byte {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	casos := map[string][]byte{
		"non é JSON":          []byte("isto non é json"),
		"formato descoñecido": marcar(map[string]any{"formato": "outra-cousa", "version": 1}),
		"formato baleiro":     marcar(map[string]any{"plantilla": map[string]any{"latex": "{{CORPO}}"}}),
		"versión do futuro":   marcar(map[string]any{"formato": FormatoBundle, "version": VersionBundle + 1}),
		"sen {{CORPO}}": marcar(Bundle{
			Formato: FormatoBundle, Version: VersionBundle,
			Plantilla: Plantilla{Nome: "Rota", Latex: `\section{Ola}`},
		}),
	}
	for nome, data := range casos {
		if _, err := ImportarBundle(data, opcionsProbas(nil), novoImportadorFalso()); err == nil {
			t.Errorf("%s: esperaba erro", nome)
		}
	}

	// Un bundle sen nome recibe un por defecto en vez de fallar.
	ok := marcar(Bundle{
		Formato: FormatoBundle, Version: VersionBundle,
		Plantilla: Plantilla{Latex: "{{CORPO}}"},
	})
	imp, err := ImportarBundle(ok, opcionsProbas(nil), novoImportadorFalso())
	if err != nil {
		t.Fatalf("bundle sen nome: %v", err)
	}
	if strings.TrimSpace(imp.Nome) == "" {
		t.Error("unha plantilla importada sen nome debería recibir un por defecto")
	}
}

// TestImportarBundleImaxeFallida: se unha imaxe non entra, a plantilla xa
// gardada devólvese igual (avísase do erro) en vez de perderse o texto.
func TestImportarBundleImaxeFallida(t *testing.T) {
	data, err := json.Marshal(Bundle{
		Formato:   FormatoBundle,
		Version:   VersionBundle,
		Plantilla: Plantilla{Nome: "Con logo", Latex: "{{CORPO}}"},
		Imaxes:    []ImaxeBundle{{Nome: "logo.png", DataB64: "x"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	imp := novoImportadorFalso()
	imp.errSubir = errors.New("disco cheo")
	gardada, err := ImportarBundle(data, opcionsProbas(nil), imp)
	if err == nil {
		t.Fatal("esperaba erro coa imaxe")
	}
	if gardada.ID == "" {
		t.Error("a plantilla xa gardada ten que devolverse igual")
	}
}

func TestNombreFicheiro(t *testing.T) {
	casos := map[string]string{
		"Circular do centro":  "Circular do centro",
		"Orzamento 2026/2027": "Orzamento 2026-2027",
		`  raro: *?"<>|  `:    "raro- ------",
		"":                    "plantilla",
		"...":                 "plantilla",
	}
	for entrada, esperado := range casos {
		if got := NombreFicheiro(entrada); got != esperado {
			t.Errorf("NombreFicheiro(%q) = %q, esperaba %q", entrada, got, esperado)
		}
	}
}
