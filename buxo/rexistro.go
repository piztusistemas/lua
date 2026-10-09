package main

// Rexistro: o terminal da interface (mesmo aspecto có de Lua) e o ficheiro de
// log da bóveda. Cada liña vai aos dous sitios.

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
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

func (r *Rexistro) escribir(nivel, formato string, a ...any) {
	l := Liña{Hora: time.Now().Format("15:04:05"), Nivel: nivel, Texto: fmt.Sprintf(formato, a...)}
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
			fmt.Fprintf(f, "%s %-5s %s\n", time.Now().Format("2006-01-02 15:04:05"), nivel, l.Texto)
			f.Close()
		}
	}
	if emitir != nil {
		emitir(l)
	}
}

func (r *Rexistro) Info(formato string, a ...any)  { r.escribir("info", formato, a...) }
func (r *Rexistro) Aviso(formato string, a ...any) { r.escribir("aviso", formato, a...) }
func (r *Rexistro) Erro(formato string, a ...any)  { r.escribir("erro", formato, a...) }

// Historial devolve as liñas gardadas en memoria.
func (r *Rexistro) Historial() []Liña {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Liña(nil), r.historial...)
}
