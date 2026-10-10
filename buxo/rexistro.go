package main

// Rexistro: o terminal da interface (mesmo aspecto có de Lua) e o ficheiro de
// log da bóveda. Cada liña vai aos dous sitios.

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"edicion/idioma"
)

// Liña é unha liña do rexistro tal e como a recibe o frontend.
type Liña struct {
	Hora  string `json:"hora"`
	Nivel string `json:"nivel"` // info | aviso | erro
	Texto string `json:"texto"`
}

type Rexistro struct {
	mu       sync.Mutex
	ficheiro string
	emitir   func(Liña)
	// historial serve para pintar o terminal ao reabrir a xanela.
	historial []Liña
}

const maxHistorialRexistro = 2000

// NovoRexistro crea un rexistro. ficheiro é a ruta do log en disco ("" = só
// memoria); emitir avisa a interface (pode ser nil).
func NovoRexistro(ficheiro string, emitir func(Liña)) *Rexistro {
	return &Rexistro{ficheiro: ficheiro, emitir: emitir}
}

func (r *Rexistro) escribir(nivel, clave, galego string, a ...any) {
	// O panel vai no idioma de Lua; buxo.log, sempre en galego.
	l := Liña{Hora: time.Now().Format("15:04:05"), Nivel: nivel, Texto: idioma.T(clave, galego, a...)}
	r.mu.Lock()
	r.historial = append(r.historial, l)
	if len(r.historial) > maxHistorialRexistro {
		r.historial = r.historial[len(r.historial)-maxHistorialRexistro:]
	}
	emitir, ficheiro := r.emitir, r.ficheiro
	r.mu.Unlock()
	if ficheiro != "" {
		_ = os.MkdirAll(filepath.Dir(ficheiro), 0o755)
		if f, err := os.OpenFile(ficheiro, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
			fmt.Fprintf(f, "%s %-5s %s\n", time.Now().Format("2006-01-02 15:04:05"), nivel, fmt.Sprintf(galego, enGalego(a)...))
			f.Close()
		}
	}
	if emitir != nil {
		emitir(l)
	}
}

// enGalego substitúe os erros traducibles polo seu texto galego, para o log.
func enGalego(args []any) []any {
	out := make([]any, len(args))
	for i, a := range args {
		if err, ok := a.(error); ok {
			out[i] = idioma.Galego(err)
		} else {
			out[i] = a
		}
	}
	return out
}

// Info, Aviso e Erro escriben unha liña: clave do dicionario, formato galego
// (o mesmo de gl.json) e argumentos, coma en fmt.Sprintf.
func (r *Rexistro) Info(clave, galego string, a ...any)  { r.escribir("info", clave, galego, a...) }
func (r *Rexistro) Aviso(clave, galego string, a ...any) { r.escribir("aviso", clave, galego, a...) }
func (r *Rexistro) Erro(clave, galego string, a ...any)  { r.escribir("erro", clave, galego, a...) }

// Historial devolve as liñas gardadas en memoria.
func (r *Rexistro) Historial() []Liña {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Liña(nil), r.historial...)
}
