package main

// Biblioteca de formularios: como se ORGANIZAN os formularios (non o que
// conteñen, iso é formulario.go). Ten tres cousas:
//
//   - CARTAFOLES para agrupar os formularios (un formulario está, como moito,
//     nun cartafol; sen cartafol vai á raíz). Borrar un cartafol NUNCA borra os
//     formularios: pasan á raíz.
//   - A ZONA FIXA da columna esquerda, en orde: o cronograma, as actas e os
//     formularios que o usuario fixe alí. O que está na zona fixa non se pode
//     borrar, só ocultar ou quitar da zona.
//   - Os AGOCHADOS: ferramentas que non se amosan na columna (seguen na
//     biblioteca e pódense volver amosar).
//
// Gárdase nun ficheiro á parte (biblioteca_formularios.json, ao lado da
// carpeta dos formularios, non dentro dela: alí todo .json é un formulario).
// Os ids das ferramentas son os do frontend: «cronograma», «actas» e
// «form:<id>».

import (
	"edicion/idioma"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	maxCarpetasFormularios   = 100
	maxNomeCarpetaFormulario = 60
	maxDescricionFormulario  = 400
)

// CarpetaFormularios é un cartafol da biblioteca.
type CarpetaFormularios struct {
	ID   string `json:"id"`
	Nome string `json:"nome"`
}

// BibliotecaFormularios é a organización dos formularios.
type BibliotecaFormularios struct {
	Carpetas []CarpetaFormularios `json:"carpetas"`
	// Carpeta: id do cartafol de cada formulario (clave = id do formulario).
	// Un formulario que non está aquí vai á raíz.
	Carpeta map[string]string `json:"carpeta"`
	// Fixos: a zona fixa da columna esquerda, en orde.
	Fixos []string `json:"fixos"`
	// Agochados: ferramentas que non se amosan na columna.
	Agochados []string `json:"agochados"`
}

func idFerramentaForm(id string) string { return "form:" + id }

// bibliotecaPorDefecto: sen cartafoles. Fóra dun proxecto o cronograma e as
// actas só están na biblioteca (agochados) ata que o usuario os active; dentro
// dun proxecto a columna amosa as ferramentas do proxecto.
func bibliotecaPorDefecto() BibliotecaFormularios {
	b := BibliotecaFormularios{Carpetas: []CarpetaFormularios{}, Carpeta: map[string]string{}, Fixos: []string{}, Agochados: []string{}}
	if id := idProxectoActivo(); id != "" {
		if p, err := lerProxecto(id); err == nil {
			b.Fixos = append(b.Fixos, p.Ferramentas...)
			for _, f := range ferramentasIntegradas {
				if !contemString(p.Ferramentas, f) {
					b.Agochados = append(b.Agochados, f)
				}
			}
			return b
		}
	}
	b.Agochados = append(b.Agochados, ferramentasIntegradas...)
	return b
}

// ferramentasIntegradas son as que sempre existen e non se poden borrar.
var ferramentasIntegradas = []string{ferramentaCronograma, ferramentaActas}

// rutaBiblioteca: ao lado da carpeta dos formularios.
func rutaBiblioteca() (string, error) {
	dir, err := dirFormularios()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(dir), "biblioteca_formularios.json"), nil
}

// normalizarBiblioteca limpa a biblioteca: descarta o que apunta a formularios
// ou cartafoles que xa non existen, desfai as repeticións e garante que o
// cronograma e as actas seguen na zona fixa. `existentes` son os ids dos
// formularios da biblioteca.
func normalizarBiblioteca(b BibliotecaFormularios, existentes map[string]bool) BibliotecaFormularios {
	out := BibliotecaFormularios{Carpetas: []CarpetaFormularios{}, Carpeta: map[string]string{}, Fixos: []string{}, Agochados: []string{}}

	usados := map[string]bool{}
	for _, c := range b.Carpetas {
		if id := strings.TrimSpace(c.ID); idFormularioRe.MatchString(id) {
			usados[id] = true
		}
	}
	vistos := map[string]bool{}
	for _, c := range b.Carpetas {
		c.Nome = strings.TrimSpace(c.Nome)
		if len([]rune(c.Nome)) > maxNomeCarpetaFormulario {
			c.Nome = string([]rune(c.Nome)[:maxNomeCarpetaFormulario])
		}
		c.ID = strings.TrimSpace(c.ID)
		if !idFormularioRe.MatchString(c.ID) || vistos[c.ID] {
			c.ID = idLibre(slugFormulario(c.Nome), usados)
		}
		vistos[c.ID] = true
		out.Carpetas = append(out.Carpetas, c)
	}

	for form, carpeta := range b.Carpeta {
		if existentes[form] && vistos[carpeta] {
			out.Carpeta[form] = carpeta
		}
	}

	coñecida := func(ferr string) bool {
		if ferr == ferramentaCronograma || ferr == ferramentaActas {
			return true
		}
		id, ok := idFerramentaFormulario(ferr)
		return ok && existentes[id]
	}
	unicos := func(lista []string) []string {
		res := []string{}
		visto := map[string]bool{}
		for _, f := range lista {
			if coñecida(f) && !visto[f] {
				visto[f] = true
				res = append(res, f)
			}
		}
		return res
	}
	out.Fixos = unicos(b.Fixos)
	out.Agochados = unicos(b.Agochados)
	return out
}

func contemString(l []string, s string) bool {
	for _, x := range l {
		if x == s {
			return true
		}
	}
	return false
}

// validarBiblioteca devolve o primeiro problema (xa normalizada), ou nil.
func validarBiblioteca(b BibliotecaFormularios, existentes map[string]bool) error {
	if len(b.Carpetas) > maxCarpetasFormularios {
		return fmt.Errorf("hai máis de %d cartafoles", maxCarpetasFormularios)
	}
	nomes := map[string]bool{}
	for _, c := range b.Carpetas {
		if c.Nome == "" {
			return errors.New("un cartafol non pode ter o nome baleiro")
		}
		k := strings.ToLower(c.Nome)
		if nomes[k] {
			return idioma.Errorf("go.cartafol.repetido", "xa hai un cartafol chamado «%s»", c.Nome)
		}
		nomes[k] = true
	}
	return nil
}

func idsFormularios() (map[string]bool, error) {
	lista, err := listarFormularios()
	if err != nil {
		return nil, err
	}
	ids := make(map[string]bool, len(lista))
	for _, f := range lista {
		ids[f.ID] = true
	}
	return ids, nil
}

// lerBiblioteca le a biblioteca (a de por defecto se aínda non hai ficheiro ou
// está estragado: nunca impide arrincar).
func lerBiblioteca() (BibliotecaFormularios, error) {
	ids, err := idsFormularios()
	if err != nil {
		return bibliotecaPorDefecto(), err
	}
	b := bibliotecaPorDefecto()
	ruta, err := rutaBiblioteca()
	if err != nil {
		return b, err
	}
	if raw, err := os.ReadFile(ruta); err == nil {
		var lida BibliotecaFormularios
		if json.Unmarshal(raw, &lida) == nil {
			b = lida
		}
	}
	return normalizarBiblioteca(b, ids), nil
}

// gardarBiblioteca normaliza, valida e garda a biblioteca. Devolve como quedou.
func gardarBiblioteca(b BibliotecaFormularios) (BibliotecaFormularios, error) {
	ids, err := idsFormularios()
	if err != nil {
		return b, err
	}
	b = normalizarBiblioteca(b, ids)
	if err := validarBiblioteca(b, ids); err != nil {
		return b, err
	}
	return b, escribirBiblioteca(b)
}

func escribirBiblioteca(b BibliotecaFormularios) error {
	ruta, err := rutaBiblioteca()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return err
	}
	tmp := ruta + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, ruta)
}

// crearCarpetaFormularios engade un cartafol (co id que lle toca) e devolve a biblioteca.
func crearCarpetaFormularios(nome string) (BibliotecaFormularios, error) {
	nome = strings.TrimSpace(nome)
	if nome == "" {
		return BibliotecaFormularios{}, idioma.Errorf("go.cartafol.sen_nome", "o cartafol necesita un nome")
	}
	b, err := lerBiblioteca()
	if err != nil {
		return b, err
	}
	usados := map[string]bool{}
	for _, c := range b.Carpetas {
		usados[c.ID] = true
	}
	base := slugFormulario(nome)
	if base == "" {
		base = "cartafol"
	}
	b.Carpetas = append(b.Carpetas, CarpetaFormularios{ID: idLibre(base, usados), Nome: nome})
	return gardarBiblioteca(b)
}

// quitarDaBiblioteca esquece un formulario borrado (cartafol, zona fixa, ocultos).
func quitarDaBiblioteca(id string) {
	b, err := lerBiblioteca()
	if err != nil {
		return
	}
	delete(b.Carpeta, id)
	ferr := idFerramentaForm(id)
	quitar := func(l []string) []string {
		res := []string{}
		for _, x := range l {
			if x != ferr {
				res = append(res, x)
			}
		}
		return res
	}
	b.Fixos, b.Agochados = quitar(b.Fixos), quitar(b.Agochados)
	_ = escribirBiblioteca(b)
}

// estaFixadoNaBiblioteca di se o formulario está na zona fixa (e polo tanto non se pode borrar).
func estaFixadoNaBiblioteca(id string) bool {
	b, err := lerBiblioteca()
	return err == nil && contemString(b.Fixos, idFerramentaForm(id))
}

// actualizarDescricionFormulario cambia SÓ a descrición dun formulario. É un
// dato da biblioteca, non do esquema: vale en calquera modo (tamén en produción).
// reClaveIcona: as claves de icona do frontend (formulario, estrela, reloxo...).
var reClaveIcona = regexp.MustCompile(`^[a-z0-9_-]{1,40}$`)

// actualizarIconaFormulario cambia só a icona da columna esquerda: vale en
// calquera modo, porque non toca os campos.
func actualizarIconaFormulario(id, icona string) (Formulario, error) {
	icona = strings.TrimSpace(icona)
	if !reClaveIcona.MatchString(icona) {
		return Formulario{}, fmt.Errorf("icona non válida: %q", icona)
	}
	f, err := lerFormulario(id)
	if err != nil {
		return Formulario{}, err
	}
	f.Icona = icona
	return f, escribirFormulario(f)
}

func actualizarDescricionFormulario(id, descricion string) (Formulario, error) {
	f, err := lerFormulario(id)
	if err != nil {
		return Formulario{}, err
	}
	descricion = strings.TrimSpace(descricion)
	if len([]rune(descricion)) > maxDescricionFormulario {
		descricion = string([]rune(descricion)[:maxDescricionFormulario])
	}
	f.Descricion = descricion
	return f, escribirFormulario(f)
}
