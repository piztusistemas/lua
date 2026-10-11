package main

// Migración do TRABALLO SOLTO (spec trasladar): os datos pertencen sempre a un
// proxecto. A primeira vez que arrinca esta versión, os borradores que había
// fóra de proxectos (cronograma, actas e datos de formularios) pasan a un
// proxecto «O meu traballo». Antes gárdase unha versión (pódese desfacer) e ao
// final déixase unha marca para non repetilo.

import (
	"edicion/idioma"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const marcaMigracion = ".migracion-proxectos"

// ResultadoMigracion é o que se moveu (Proxecto "" = non había nada).
type ResultadoMigracion struct {
	Proxecto    string   `json:"proxecto"`
	Nome        string   `json:"nome"`
	Ferramentas []string `json:"ferramentas"`
	Version     string   `json:"version"`
}

// borradoresSoltos atopa os borradores fóra de proxectos: ferramenta → ruta.
func borradoresSoltos() map[string]string {
	out := map[string]string{}
	cfg, err := dirConfigLua()
	if err != nil {
		return out
	}
	dirs := []string{filepath.Join(cfg, "informes")}
	if vello, err := borradorFerramentaPathVello(ferramentaCronograma); err == nil {
		dirs = append(dirs, filepath.Dir(vello))
	}
	for _, dir := range dirs {
		ents, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range ents {
			n := e.Name()
			if e.IsDir() || !strings.HasSuffix(n, "-borrador.json") {
				continue
			}
			var ferr string
			switch {
			case n == "cronograma-borrador.json":
				ferr = ferramentaCronograma
			case n == "actas-borrador.json":
				ferr = ferramentaActas
			case strings.HasPrefix(n, "formulario-"):
				fid := strings.TrimSuffix(strings.TrimPrefix(n, "formulario-"), "-borrador.json")
				if _, err := lerFormularioBiblioteca(fid); err != nil {
					continue // sen o formulario, os datos quedan onde estaban
				}
				ferr = idFerramentaForm(fid)
			default:
				continue
			}
			if _, xa := out[ferr]; !xa {
				out[ferr] = filepath.Join(dir, n)
			}
		}
	}
	return out
}

// migrarTraballoSolto move o traballo solto a «O meu traballo». É idempotente.
func migrarTraballoSolto() (ResultadoMigracion, error) {
	res := ResultadoMigracion{Ferramentas: []string{}}
	cfg, err := dirConfigLua()
	if err != nil {
		return res, err
	}
	marca := filepath.Join(cfg, marcaMigracion)
	if fileExists(marca) {
		return res, nil
	}
	soltos := borradoresSoltos()
	if len(soltos) == 0 {
		return res, escribirMarcaMigracion(marca)
	}
	ferramentas := []string{}
	for _, f := range []string{ferramentaCronograma, ferramentaActas} {
		if _, ok := soltos[f]; ok {
			ferramentas = append(ferramentas, f)
		}
	}
	for _, f := range ordenadosMapa(soltos) {
		if strings.HasPrefix(f, "form:") {
			ferramentas = append(ferramentas, f)
		}
	}
	ids, _ := idsEProxectos()
	nome := nomeProxectoLibre(idioma.T("go.trasladar.meu_traballo", "O meu traballo"))
	id := idLibre(slugFormulario(nome), ids)
	dir, err := dirProxecto(id)
	if err != nil {
		return res, err
	}
	rutas := []string{dir}
	for _, r := range soltos {
		rutas = append(rutas, r)
	}
	ver, err := gardarVersion("migracion", nome, rutas)
	if err != nil {
		return res, err
	}
	fallo := func(err error) (ResultadoMigracion, error) {
		_ = restaurarVersion(ver)
		return ResultadoMigracion{Ferramentas: []string{}}, err
	}
	p := Proxecto{ID: id, Nome: nome, Creado: time.Now().Format("2006-01-02"), Ferramentas: ferramentas}
	for _, f := range ferramentas {
		if fid, ok := idFerramentaFormulario(f); ok {
			if err := copiarFormularioAoProxecto(id, fid); err != nil {
				return fallo(err)
			}
		}
	}
	alm, _ := loadAlmacenPlantillas()
	for _, f := range ferramentas {
		orixe := soltos[f]
		datos, err := os.ReadFile(orixe)
		if err != nil {
			return fallo(err)
		}
		destino := filepath.Join(dir, "datos", borradorFerramentaNome(f))
		if err := os.MkdirAll(filepath.Dir(destino), 0o755); err != nil {
			return fallo(err)
		}
		if err := escribirAtomicoCrono(destino, sanearContidoDocumento(datos)); err != nil {
			return fallo(err)
		}
		// A plantilla que usaba o ficheiro vinculado pasa ao documento.
		if ligado := rutaVinculada(datos); ligado != "" {
			if pl := plantillaDeDocumento(alm, ligado); pl != nil {
				_ = lembrarPlantillaDocumento(destino, pl.ID)
			}
		}
	}
	if err := escribirProxecto(p); err != nil {
		return fallo(err)
	}
	for _, r := range soltos {
		_ = os.Remove(r)
	}
	if err := escribirMarcaMigracion(marca); err != nil {
		return fallo(err)
	}
	res.Proxecto, res.Nome, res.Ferramentas, res.Version = id, nome, ferramentas, ver
	return res, nil
}

func escribirMarcaMigracion(marca string) error {
	if err := os.MkdirAll(filepath.Dir(marca), 0o755); err != nil {
		return err
	}
	return os.WriteFile(marca, []byte(time.Now().Format(time.RFC3339)+"\n"), 0o644)
}

// rutaVinculada le o campo «path» dun borrador (o ficheiro vinculado de antes).
func rutaVinculada(datos []byte) string {
	var d struct {
		Path string `json:"path"`
	}
	_ = json.Unmarshal(datos, &d)
	return d.Path
}

func ordenadosMapa(m map[string]string) []string {
	c := map[string]bool{}
	for k := range m {
		c[k] = true
	}
	return ordenados(c)
}
