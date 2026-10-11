package main

// CATÁLOGO COMÚN de formularios (spec reutilizar-formularios): a biblioteca
// de todos os formularios creados, sexa no proxecto que sexa, organizada en
// cartafoles. Desde a ventá «Deseñar formulario» márcanse e reutilízanse no
// proxecto aberto (cada proxecto leva a súa copia, como sempre).
//
// Os formularios viven en dirBibliotecaFormularios() e os cartafoles no
// biblioteca_formularios.json que está ao seu lado (o mesmo que usaba Lúa
// fóra dos proxectos, así que os cartafoles de antes consérvanse).

import (
	"edicion/idioma"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FormularioCatalogo é unha liña da ventá de formularios.
type FormularioCatalogo struct {
	ID         string `json:"id"`
	Nome       string `json:"nome"`
	Icona      string `json:"icona,omitempty"`
	Cor        string `json:"cor,omitempty"`
	Descricion string `json:"descricion,omitempty"`
	Modo       string `json:"modo,omitempty"`
	Carpeta    string `json:"carpeta"` // "" = sen cartafol
}

// CatalogoFormularios é o que amosa a ventá de formularios.
type CatalogoFormularios struct {
	// Integradas: cronograma e actas (sempre se poden reutilizar).
	Integradas  []string             `json:"integradas"`
	Formularios []FormularioCatalogo `json:"formularios"`
	Carpetas    []CarpetaFormularios `json:"carpetas"`
	// NoProxecto: as ferramentas que xa ten o proxecto aberto.
	NoProxecto []string `json:"noProxecto"`
}

// rutaOrganizacionCatalogo é o ficheiro dos cartafoles do catálogo.
func rutaOrganizacionCatalogo() (string, error) {
	lib, err := dirBibliotecaFormularios()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(lib), "biblioteca_formularios.json"), nil
}

// lerOrganizacionCatalogo le os cartafoles (sen erro se aínda non hai). Os
// demais campos do ficheiro consérvanse ao escribir.
func lerOrganizacionCatalogo() (BibliotecaFormularios, error) {
	b := BibliotecaFormularios{Carpetas: []CarpetaFormularios{}, Carpeta: map[string]string{}, Fixos: []string{}, Agochados: []string{}}
	ruta, err := rutaOrganizacionCatalogo()
	if err != nil {
		return b, err
	}
	if raw, err := os.ReadFile(ruta); err == nil {
		var lida BibliotecaFormularios
		if json.Unmarshal(raw, &lida) == nil {
			b = lida
		}
	}
	if b.Carpetas == nil {
		b.Carpetas = []CarpetaFormularios{}
	}
	if b.Carpeta == nil {
		b.Carpeta = map[string]string{}
	}
	return b, nil
}

func escribirOrganizacionCatalogo(b BibliotecaFormularios) error {
	ruta, err := rutaOrganizacionCatalogo()
	if err != nil {
		return err
	}
	return escribirXSONAtomico(ruta, b)
}

// catalogoFormularios arma o catálogo para a ventá.
func catalogoFormularios() (CatalogoFormularios, error) {
	c := CatalogoFormularios{Integradas: append([]string(nil), ferramentasIntegradas...), Formularios: []FormularioCatalogo{}, NoProxecto: []string{}}
	fs, err := formulariosBiblioteca()
	if err != nil {
		return c, err
	}
	org, err := lerOrganizacionCatalogo()
	if err != nil {
		return c, err
	}
	c.Carpetas = org.Carpetas
	carpetas := map[string]bool{}
	for _, cp := range org.Carpetas {
		carpetas[cp.ID] = true
	}
	for _, f := range fs {
		carp := org.Carpeta[f.ID]
		if !carpetas[carp] {
			carp = ""
		}
		c.Formularios = append(c.Formularios, FormularioCatalogo{ID: f.ID, Nome: f.Nome, Icona: f.Icona, Cor: f.Cor, Descricion: f.Descricion, Modo: f.Modo, Carpeta: carp})
	}
	sort.Slice(c.Formularios, func(i, j int) bool {
		return strings.ToLower(c.Formularios[i].Nome) < strings.ToLower(c.Formularios[j].Nome)
	})
	if pid := proxectoAbertoValido(); pid != "" {
		if p, err := lerProxecto(pid); err == nil {
			c.NoProxecto = append(c.NoProxecto, p.Ferramentas...)
		}
	}
	return c, nil
}

// crearCarpetaCatalogo engade un cartafol e devolve o seu id.
func crearCarpetaCatalogo(nome string) (string, error) {
	nome = limparTexto(nome, maxNomeCarpetaFormulario)
	if nome == "" {
		return "", idioma.Errorf("go.cartafol.sen_nome", "o cartafol necesita un nome")
	}
	b, err := lerOrganizacionCatalogo()
	if err != nil {
		return "", err
	}
	if len(b.Carpetas) >= maxCarpetasFormularios {
		return "", fmt.Errorf("hai máis de %d cartafoles", maxCarpetasFormularios)
	}
	usados := map[string]bool{}
	for _, cp := range b.Carpetas {
		if strings.EqualFold(cp.Nome, nome) {
			return cp.ID, nil // xa existe: non se duplica
		}
		usados[cp.ID] = true
	}
	base := slugFormulario(nome)
	if base == "" {
		base = "cartafol"
	}
	id := idLibre(base, usados)
	b.Carpetas = append(b.Carpetas, CarpetaFormularios{ID: id, Nome: nome})
	return id, escribirOrganizacionCatalogo(b)
}

func renomearCarpetaCatalogo(id, nome string) error {
	nome = limparTexto(nome, maxNomeCarpetaFormulario)
	if nome == "" {
		return idioma.Errorf("go.cartafol.sen_nome", "o cartafol necesita un nome")
	}
	b, err := lerOrganizacionCatalogo()
	if err != nil {
		return err
	}
	for i := range b.Carpetas {
		if b.Carpetas[i].ID == id {
			b.Carpetas[i].Nome = nome
			return escribirOrganizacionCatalogo(b)
		}
	}
	return idioma.Errorf("go.cartafol.non_existe", "non existe o cartafol «%s»", id)
}

// eliminarCarpetaCatalogo borra o cartafol; os seus formularios pasan a
// «Sen cartafol» (nunca se borra un formulario).
func eliminarCarpetaCatalogo(id string) error {
	b, err := lerOrganizacionCatalogo()
	if err != nil {
		return err
	}
	novas := []CarpetaFormularios{}
	for _, cp := range b.Carpetas {
		if cp.ID != id {
			novas = append(novas, cp)
		}
	}
	b.Carpetas = novas
	for fid, carp := range b.Carpeta {
		if carp == id {
			delete(b.Carpeta, fid)
		}
	}
	return escribirOrganizacionCatalogo(b)
}

// moverFormularioCatalogo pon un formulario nun cartafol ("" = sen cartafol).
func moverFormularioCatalogo(fid, carpeta string) error {
	if _, err := lerFormularioBiblioteca(fid); err != nil {
		return idioma.Errorf("go.catalogo.sen_formulario", "o formulario «%s» non está na biblioteca", fid)
	}
	b, err := lerOrganizacionCatalogo()
	if err != nil {
		return err
	}
	if carpeta == "" {
		delete(b.Carpeta, fid)
		return escribirOrganizacionCatalogo(b)
	}
	for _, cp := range b.Carpetas {
		if cp.ID == carpeta {
			b.Carpeta[fid] = carpeta
			return escribirOrganizacionCatalogo(b)
		}
	}
	return idioma.Errorf("go.cartafol.non_existe", "non existe o cartafol «%s»", carpeta)
}

// reutilizarNoProxecto engade as ferramentas («cronograma», «actas» ou
// «form:<id>» do catálogo) ao proxecto aberto, cunha copia de cada formulario.
func reutilizarNoProxecto(ferramentas []string) (Proxecto, error) {
	pid := proxectoAbertoValido()
	if pid == "" {
		return Proxecto{}, idioma.Errorf("go.catalogo.sen_proxecto", "non hai ningún proxecto aberto")
	}
	var p Proxecto
	var err error
	for _, f := range ferramentas {
		if p, err = engadirAoProxecto(pid, f); err != nil {
			return p, err
		}
	}
	return lerProxecto(pid)
}

// sincronizarFormularioCatalogo deixa no catálogo común a copia dun
// formulario gardado nun proxecto: actualiza o do mesmo id, agás se o do
// catálogo está en produción (entón entra coma copia, sen pisalo).
func sincronizarFormularioCatalogo(f Formulario) error {
	lib, err := dirBibliotecaFormularios()
	if err != nil {
		return err
	}
	if actual, err := lerFormularioBiblioteca(f.ID); err == nil && actual.enProduccion() {
		_, _, err := meterFormularioBiblioteca(f)
		return err
	}
	f.Modo = modoDeseno
	return escribirXSONAtomico(filepath.Join(lib, f.ID+".json"), f)
}
