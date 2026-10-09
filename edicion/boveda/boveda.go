// Package boveda é o espazo de traballo de LuaYang: unha bóveda de Obsidian
// coas fontes (pdf/, raw/), a wiki de Buxo (wiki/) e as fichas (fichas/).
package boveda

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"edicion/ficha"
)

// Cartafoles da bóveda.
const (
	DirPDF          = "pdf"
	DirRaw          = "raw"
	DirRawImaxes    = "raw/imaxes"
	DirWiki         = "wiki"
	DirFichas       = "fichas"
	DirSaida        = "saida"
	DirPapeleira    = ".papeleira"
	FicheiroEsquema = "buxo.md"
)

// ErrCambiouNoDisco: o ficheiro cambiou desde que se abriu (outra persoa,
// Obsidian…). Non se sobrescribe.
var ErrCambiouNoDisco = errors.New("o ficheiro cambiou no disco desde que se abriu")

// Boveda aberta.
type Boveda struct {
	Raiz string
}

// Crear crea (ou completa) a estrutura dunha bóveda. Non pisa nada.
func Crear(raiz, materia string) (*Boveda, error) {
	raiz = filepath.Clean(raiz)
	for _, d := range []string{DirPDF, DirRaw, DirRawImaxes, DirWiki, DirWiki + "/fontes", DirWiki + "/conceptos", DirWiki + "/exercicios", DirFichas, DirSaida} {
		if err := os.MkdirAll(filepath.Join(raiz, d), 0o755); err != nil {
			return nil, err
		}
	}
	esquema := filepath.Join(raiz, FicheiroEsquema)
	if _, err := os.Stat(esquema); errors.Is(err, fs.ErrNotExist) {
		if err := os.WriteFile(esquema, []byte(TextoEsquema(materia)), 0o644); err != nil {
			return nil, err
		}
	}
	return &Boveda{Raiz: raiz}, nil
}

// Abrir abre unha bóveda que xa existe (e completa os cartafoles que falten).
func Abrir(raiz string) (*Boveda, error) {
	st, err := os.Stat(raiz)
	if err != nil {
		return nil, fmt.Errorf("non se pode abrir a bóveda: %w", err)
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("%s non é un cartafol", raiz)
	}
	return Crear(raiz, "")
}

// Resumo dunha ficha para as listaxes.
type Resumo struct {
	ID         string    `json:"id"`
	Ruta       string    `json:"ruta"` // relativa á bóveda
	Tipo       string    `json:"tipo"`
	Titulo     string    `json:"titulo"`
	Materia    string    `json:"materia"`
	Nivel      string    `json:"nivel"`
	Formato    string    `json:"formato"`
	Conceptos  []string  `json:"conceptos"`
	Modificado time.Time `json:"modificado"`
	Erro       string    `json:"erro,omitempty"`
}

func (b *Boveda) saltar(rel string, d fs.DirEntry) bool {
	n := d.Name()
	if d.IsDir() {
		return strings.HasPrefix(n, ".") || rel == DirPDF || rel == DirRaw || rel == DirSaida || rel == "node_modules"
	}
	return !strings.HasSuffix(n, ".md")
}

// Fichas lista todas as fichas da bóveda (calquera .md con front matter de
// LuaYang, estea onde estea; as notas normais de Obsidian ignóranse).
func (b *Boveda) Fichas() ([]Resumo, error) {
	out := []Resumo{}
	err := filepath.WalkDir(b.Raiz, func(ruta string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(b.Raiz, ruta)
		if rel == "." {
			return nil
		}
		if b.saltar(filepath.ToSlash(rel), d) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		datos, err := os.ReadFile(ruta)
		if err != nil || !strings.HasPrefix(strings.TrimPrefix(string(datos), "\ufeff"), "---") || !strings.Contains(string(datos[:min(len(datos), 400)]), "luayang:") {
			return nil
		}
		info, _ := d.Info()
		r := Resumo{ID: strings.TrimSuffix(d.Name(), ".md"), Ruta: filepath.ToSlash(rel)}
		if info != nil {
			r.Modificado = info.ModTime()
		}
		f, err := ficha.Ler(datos)
		if err != nil {
			r.Erro = err.Error()
		} else {
			r.Tipo, r.Titulo, r.Materia, r.Nivel, r.Formato, r.Conceptos = f.Tipo, f.Titulo, f.Materia, f.Nivel, f.FormatoCorpo(), f.Conceptos
		}
		out = append(out, r)
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Ruta) < strings.ToLower(out[j].Ruta) })
	return out, err
}

// Ruta absoluta dunha ficha polo seu id (o nome do ficheiro sen .md, como
// nas ligazóns [[…]] de Obsidian). Admite tamén unha ruta relativa.
func (b *Boveda) Ruta(id string) (string, error) {
	id = strings.TrimSuffix(strings.TrimSpace(id), ".md")
	if id == "" {
		return "", errors.New("falta o id da ficha")
	}
	if strings.Contains(id, "/") {
		r := filepath.Join(b.Raiz, filepath.FromSlash(id)+".md")
		if !dentro(b.Raiz, r) {
			return "", errors.New("ruta fóra da bóveda")
		}
		if _, err := os.Stat(r); err != nil {
			return "", fmt.Errorf("non existe a ficha %s", id)
		}
		return r, nil
	}
	var atopada string
	_ = filepath.WalkDir(b.Raiz, func(ruta string, d fs.DirEntry, err error) error {
		if err != nil || atopada != "" {
			return nil
		}
		rel, _ := filepath.Rel(b.Raiz, ruta)
		if rel != "." && b.saltar(filepath.ToSlash(rel), d) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.IsDir() && strings.TrimSuffix(d.Name(), ".md") == id {
			atopada = ruta
		}
		return nil
	})
	if atopada == "" {
		return "", fmt.Errorf("non existe a ficha %s", id)
	}
	return atopada, nil
}

func dentro(raiz, ruta string) bool {
	rel, err := filepath.Rel(raiz, ruta)
	return err == nil && !strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel)
}

// Ler le unha ficha e a hora da súa última modificación.
func (b *Boveda) Ler(id string) (*ficha.Ficha, time.Time, error) {
	r, err := b.Ruta(id)
	if err != nil {
		return nil, time.Time{}, err
	}
	datos, err := os.ReadFile(r)
	if err != nil {
		return nil, time.Time{}, err
	}
	st, _ := os.Stat(r)
	f, err := ficha.Ler(datos)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("%s: %w", id, err)
	}
	if f.ID == "" {
		f.ID = strings.TrimSuffix(filepath.Base(r), ".md")
	}
	return f, st.ModTime(), nil
}

// Resolver: o que precisa o xerador para as ![[ligazóns]] e os bancos.
func (b *Boveda) Resolver(id string) (*ficha.Ficha, error) {
	f, _, err := b.Ler(id)
	return f, err
}

// Gardar escribe a ficha no seu sitio (ou en fichas/<id>.md se é nova).
// `lida` é a hora de modificación de cando se abriu: se o ficheiro cambiou
// despois, non se garda (ErrCambiouNoDisco). Cero = non comprobar.
func (b *Boveda) Gardar(f *ficha.Ficha, lida time.Time) (time.Time, error) {
	if strings.TrimSpace(f.ID) == "" {
		return time.Time{}, errors.New("a ficha non ten id")
	}
	r, err := b.Ruta(f.ID)
	if err != nil {
		r = filepath.Join(b.Raiz, DirFichas, f.ID+".md")
	} else if !lida.IsZero() {
		if st, err := os.Stat(r); err == nil && st.ModTime().After(lida.Add(time.Millisecond)) {
			return time.Time{}, ErrCambiouNoDisco
		}
	}
	datos, err := ficha.Escribir(f)
	if err != nil {
		return time.Time{}, err
	}
	if err := escribirAtomico(r, datos); err != nil {
		return time.Time{}, err
	}
	st, err := os.Stat(r)
	if err != nil {
		return time.Time{}, err
	}
	return st.ModTime(), nil
}

func escribirAtomico(ruta string, datos []byte) error {
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		return err
	}
	tmp := ruta + ".gardando"
	if err := os.WriteFile(tmp, datos, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, ruta)
}

// IDLibre: un id que aínda non usa ningunha ficha («base», «base-2»…).
func (b *Boveda) IDLibre(base string) string {
	base = ficha.Slug(base)
	if base == "" {
		base = "ficha"
	}
	usados := map[string]bool{}
	if l, err := b.Fichas(); err == nil {
		for _, r := range l {
			usados[r.ID] = true
		}
	}
	if !usados[base] {
		return base
	}
	for i := 2; ; i++ {
		if c := fmt.Sprintf("%s-%d", base, i); !usados[c] {
			return c
		}
	}
}

// Nova crea unha ficha baleira do tipo dado en fichas/.
func (b *Boveda) Nova(tipo, titulo string) (*ficha.Ficha, error) {
	f := &ficha.Ficha{Luayang: ficha.Version, Tipo: tipo, Titulo: strings.TrimSpace(titulo)}
	f.ID = b.IDLibre(titulo)
	switch tipo {
	case ficha.TipoExercicio:
		f.Corpo = "## Enunciado\n\n\n\n## Resultado\n\n\n\n## Resolución\n\n"
	case ficha.TipoExame:
		f.Corpo = "Instrucións do exame.\n\n## Pregunta 1\n\n"
	case ficha.TipoBanco:
		f.Columnas = []string{"texto", "solucion"}
		f.Corpo = "| texto | solucion |\n|---|---|\n|  |  |\n"
	case ficha.TipoFormulario:
		f.Corpo = ""
	default:
		return nil, fmt.Errorf("tipo de ficha descoñecido: %s", tipo)
	}
	if _, err := b.Gardar(f, time.Time{}); err != nil {
		return nil, err
	}
	return f, nil
}

// Eliminar move a ficha á papeleira da bóveda (non se perde nada).
func (b *Boveda) Eliminar(id string) error {
	r, err := b.Ruta(id)
	if err != nil {
		return err
	}
	dest := filepath.Join(b.Raiz, DirPapeleira, time.Now().Format("20060102-150405")+"-"+filepath.Base(r))
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	return os.Rename(r, dest)
}

// GardarImportacion escribe as fichas dunha importación nun subcartafol de
// fichas/ (co nome do exame), cambiando os ids que xa existan.
func (b *Boveda) GardarImportacion(imp *ficha.Importacion) ([]string, error) {
	cartafol := filepath.Join(b.Raiz, DirFichas, ficha.Slug(imp.Exame.ID))
	cambios := map[string]string{}
	for _, f := range append([]*ficha.Ficha{imp.Exame}, imp.Exercicios...) {
		novo := b.IDLibre(f.ID)
		for _, outro := range cambios {
			if outro == novo {
				novo = b.IDLibre(novo + "-b")
			}
		}
		cambios[f.ID] = novo
	}
	for vello, novo := range cambios {
		if vello != novo {
			imp.Exame.Corpo = strings.ReplaceAll(imp.Exame.Corpo, "[["+vello+"]]", "[["+novo+"]]")
		}
	}
	var ids []string
	for _, f := range append([]*ficha.Ficha{imp.Exame}, imp.Exercicios...) {
		f.ID = cambios[f.ID]
		datos, err := ficha.Escribir(f)
		if err != nil {
			return ids, err
		}
		if err := escribirAtomico(filepath.Join(cartafol, f.ID+".md"), datos); err != nil {
			return ids, err
		}
		ids = append(ids, f.ID)
	}
	return ids, nil
}
