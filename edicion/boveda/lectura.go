package boveda

// Gardar un PDF lido (edicion/lectura) na bóveda: o orixinal en pdf/, a
// transcrición literal en raw/ (coas figuras en raw/imaxes/) e unha ficha
// estática por exercicio, máis o exame que as compón.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"edicion/ficha"
	"edicion/lectura"
)

// ResultadoImportacion: o que se creou.
type ResultadoImportacion struct {
	PDF        string   `json:"pdf"`
	Raw        string   `json:"raw"`
	Exame      string   `json:"exame"`
	Exercicios []string `json:"exercicios"`
	Avisos     []string `json:"avisos"`
}

// GardarLectura escribe todo na bóveda. dirTraballo é onde lectura.Ler
// deixou as figuras (dirTraballo/imaxes).
func (b *Boveda) GardarLectura(pdfOrixe string, pax []lectura.PaxinaLida, e lectura.Estrutura, dirTraballo string) (ResultadoImportacion, error) {
	var r ResultadoImportacion
	base := ficha.Slug(strings.TrimSuffix(filepath.Base(pdfOrixe), filepath.Ext(pdfOrixe)))
	if base == "" {
		base = "documento"
	}
	// 1. O PDF orixinal (sen pisar outro co mesmo nome).
	nomePDF := base + ".pdf"
	for i := 2; ; i++ {
		if _, err := os.Stat(filepath.Join(b.Raiz, DirPDF, nomePDF)); os.IsNotExist(err) {
			break
		}
		nomePDF = fmt.Sprintf("%s-%d.pdf", base, i)
	}
	base = strings.TrimSuffix(nomePDF, ".pdf")
	datos, err := os.ReadFile(pdfOrixe)
	if err != nil {
		return r, err
	}
	if err := escribirAtomico(filepath.Join(b.Raiz, DirPDF, nomePDF), datos); err != nil {
		return r, err
	}
	r.PDF = DirPDF + "/" + nomePDF
	// 2. As figuras.
	if ents, err := os.ReadDir(filepath.Join(dirTraballo, "imaxes")); err == nil {
		for _, en := range ents {
			d, err := os.ReadFile(filepath.Join(dirTraballo, "imaxes", en.Name()))
			if err == nil {
				err = escribirAtomico(filepath.Join(b.Raiz, DirRawImaxes, en.Name()), d)
			}
			if err != nil {
				r.Avisos = append(r.Avisos, "figura "+en.Name()+": "+err.Error())
			}
		}
	}
	// 3. A transcrición literal.
	var raw strings.Builder
	fmt.Fprintf(&raw, "---\nfonte: \"[[%s]]\"\nimportado: %s\npaxinas:\n", nomePDF, time.Now().Format("2006-01-02"))
	for _, p := range pax {
		fmt.Fprintf(&raw, "  - {num: %d, metodo: %s", p.Num, p.Metodo)
		if p.Cobertura >= 0 {
			fmt.Fprintf(&raw, ", cobertura: %.2f", p.Cobertura)
		}
		raw.WriteString("}\n")
	}
	raw.WriteString("---\n\n" + lectura.MarkdownCompleto(pax))
	rawRel := DirRaw + "/" + base + ".md"
	if err := escribirAtomico(filepath.Join(b.Raiz, filepath.FromSlash(rawRel)), []byte(raw.String())); err != nil {
		return r, err
	}
	r.Raw = rawRel
	// 4. As fichas: unha por exercicio e o exame.
	cartafol := filepath.Join(b.Raiz, DirFichas, base)
	exame := &ficha.Ficha{
		Luayang: ficha.Version, ID: b.IDLibre(base), Tipo: ficha.TipoExame,
		Titulo: e.Titulo, Materia: e.Materia, Nivel: e.Nivel, Linguas: e.Linguas,
		Fonte: &ficha.Fonte{PDF: "[[" + nomePDF + "]]"},
	}
	usados := map[string]bool{exame.ID: true}
	var corpo strings.Builder
	if strings.TrimSpace(e.Instrucions) != "" {
		corpo.WriteString(strings.TrimSpace(e.Instrucions) + "\n\n")
	}
	for i, ex := range e.Exercicios {
		id := b.IDLibre(fmt.Sprintf("%s-%d", base, i+1))
		for usados[id] {
			id = b.IDLibre(id + "-b")
		}
		usados[id] = true
		f := &ficha.Ficha{
			Luayang: ficha.Version, ID: id, Tipo: ficha.TipoExercicio,
			Titulo: ex.Titulo, Materia: e.Materia, Nivel: e.Nivel, Puntos: ex.Puntos,
			Fonte:   &ficha.Fonte{PDF: "[[" + nomePDF + "]]", Paxina: ex.Paxina, Apartado: ex.Apartado},
			Linguas: e.Linguas,
		}
		for _, c := range ex.Conceptos {
			f.Conceptos = append(f.Conceptos, "[["+strings.TrimSpace(c)+"]]")
		}
		ss := []ficha.Seccion{{Titulo: "Enunciado", Texto: strings.TrimSpace(ex.Enunciado)}}
		for _, l := range e.Linguas {
			if t, ok := ex.Outras[l]; ok && strings.TrimSpace(t) != "" {
				ss = append(ss, ficha.Seccion{Titulo: "Enunciado [" + l + "]", Texto: strings.TrimSpace(t)})
			}
		}
		for l, t := range ex.Outras {
			if !contenLingua(e.Linguas, l) && strings.TrimSpace(t) != "" {
				ss = append(ss, ficha.Seccion{Titulo: "Enunciado [" + l + "]", Texto: strings.TrimSpace(t)})
			}
		}
		ss = append(ss, ficha.Seccion{Titulo: "Resultado"}, ficha.Seccion{Titulo: "Resolución"})
		f.Corpo = ficha.CorpoDeSeccions(ss)
		d, err := ficha.Escribir(f)
		if err != nil {
			return r, err
		}
		if err := escribirAtomico(filepath.Join(cartafol, id+".md"), d); err != nil {
			return r, err
		}
		r.Exercicios = append(r.Exercicios, id)
		corpo.WriteString("![[" + id + "]]\n\n")
	}
	exame.Corpo = corpo.String()
	d, err := ficha.Escribir(exame)
	if err != nil {
		return r, err
	}
	if err := escribirAtomico(filepath.Join(cartafol, exame.ID+".md"), d); err != nil {
		return r, err
	}
	r.Exame = exame.ID
	r.Avisos = append(r.Avisos, e.Avisos...)
	return r, nil
}

func contenLingua(l []string, s string) bool {
	for _, x := range l {
		if x == s {
			return true
		}
	}
	return false
}
