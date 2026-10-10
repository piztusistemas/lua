// Package idioma traduce as mensaxes que xera o Go (erros e avisos) ao idioma
// activo da aplicación: gl, es, en ou pt.
//
// O galego é a fonte e a reserva. Cada mensaxe leva no propio código a súa
// clave e o seu formato galego, coma o t(clave, galego) do frontend; a
// aplicación rexistra ao arrincar os dicionarios dos outros idiomas e di cal
// está activo. Unha aplicación que non rexistra nada (Yang, os tests) segue
// exactamente en galego.
//
// O estado é global ao proceso, coma o idioma da interface.
package idioma

import (
	"errors"
	"fmt"
	"sync"
)

var idiomasValidos = map[string]bool{"gl": true, "es": true, "en": true, "pt": true}

var (
	mu          sync.RWMutex
	actual      = "gl"
	dicionarios = map[string]map[string]string{}
)

// Rexistrar engade (ou substitúe) o dicionario clave → formato dun idioma.
func Rexistrar(codigo string, dicionario map[string]string) {
	copia := make(map[string]string, len(dicionario))
	for clave, formato := range dicionario {
		copia[clave] = formato
	}
	mu.Lock()
	dicionarios[codigo] = copia
	mu.Unlock()
}

// SetIdioma fixa o idioma activo. Un código baleiro ou descoñecido é "gl".
func SetIdioma(codigo string) {
	if !idiomasValidos[codigo] {
		codigo = "gl"
	}
	mu.Lock()
	actual = codigo
	mu.Unlock()
}

// Actual devolve o código do idioma activo.
func Actual() string {
	mu.RLock()
	defer mu.RUnlock()
	return actual
}

// Restablecer volve ao estado inicial: galego e sen dicionarios. Para os tests.
func Restablecer() {
	mu.Lock()
	actual = "gl"
	dicionarios = map[string]map[string]string{}
	mu.Unlock()
}

// formato devolve o formato da clave no idioma activo ou, se non o hai, o galego.
func formato(clave, galego string) string {
	mu.RLock()
	defer mu.RUnlock()
	if f, ok := dicionarios[actual][clave]; ok && actual != "gl" {
		return f
	}
	return galego
}

// T devolve a mensaxe no idioma activo, cos argumentos inseridos coma en
// fmt.Sprintf. En galego é exactamente fmt.Sprintf(galego, args...).
func T(clave, galego string, args ...any) string {
	return fmt.Sprintf(formato(clave, galego), args...)
}

// erroTraducible é un erro que se traduce ao ler o seu texto, non ao crealo:
// se o idioma cambia mentres viaxa polas capas, sae no idioma novo.
type erroTraducible struct {
	clave, galego string
	args          []any
}

// Errorf crea un erro coma fmt.Errorf (incluído %w) cuxo texto sae no idioma
// activo cando se le.
func Errorf(clave, galego string, args ...any) error {
	return &erroTraducible{clave: clave, galego: galego, args: args}
}

func (e *erroTraducible) Error() string {
	return fmt.Errorf(formato(e.clave, e.galego), e.args...).Error()
}

// Unwrap devolve o erro envolto con %w, coma fmt.Errorf.
func (e *erroTraducible) Unwrap() error {
	return errors.Unwrap(fmt.Errorf(e.galego, e.args...))
}

// textoGalego é un erro que só leva un texto xa resolto en galego.
type textoGalego string

func (t textoGalego) Error() string { return string(t) }

// Galego devolve o texto en galego dun erro, aínda que o idioma activo sexa
// outro: para os ficheiros de log, que van sempre en galego. Un erro que non
// creou este paquete devolve o seu Error() tal cal.
func Galego(err error) string {
	if err == nil {
		return ""
	}
	e, ok := err.(*erroTraducible)
	if !ok {
		return err.Error()
	}
	args := make([]any, len(e.args))
	for i, a := range e.args {
		if inner, ok := a.(error); ok {
			args[i] = textoGalego(Galego(inner))
		} else {
			args[i] = a
		}
	}
	return fmt.Errorf(e.galego, args...).Error()
}
