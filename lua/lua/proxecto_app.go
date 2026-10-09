package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Métodos de App para os proxectos (ver proxecto.go).

// ListarProxectos devolve os proxectos, por nome.
func (a *App) ListarProxectos() ([]ProxectoInfo, error) {
	return listarProxectos()
}

// ListarFormulariosBiblioteca devolve os formularios da BIBLIOTECA aínda que
// haxa un proxecto aberto (ListarFormularios devolve os do proxecto): serve
// para escoller con que formularios se crea ou se amplía un proxecto.
func (a *App) ListarFormulariosBiblioteca() ([]Formulario, error) {
	dir, err := dirBibliotecaFormularios()
	if err != nil {
		return nil, err
	}
	ents, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return []Formulario{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := []Formulario{}
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		d, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var f Formulario
		if json.Unmarshal(d, &f) != nil || !idFormularioRe.MatchString(f.ID) {
			continue
		}
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Nome) < strings.ToLower(out[j].Nome)
	})
	return out, nil
}

// CrearProxecto crea un proxecto cos formularios («form:<id>») e ferramentas
// integradas («cronograma», «actas») escollidos.
func (a *App) CrearProxecto(nome, materia string, ferramentas []string) (Proxecto, error) {
	return crearProxecto(nome, materia, ferramentas)
}

// ActivarProxecto fai que Lua traballe dentro do proxecto ("" = saír del). O
// frontend ten que volcar antes os borradores en curso e recargar despois.
func (a *App) ActivarProxecto(id string) (Proxecto, error) {
	return activarProxecto(id)
}

// ProxectoActivo devolve o id do proxecto aberto ("" se ningún).
func (a *App) ProxectoActivo() string {
	return idProxectoActivo()
}

// EngadirAoProxecto engade unha ferramenta (copiando o formulario da biblioteca).
func (a *App) EngadirAoProxecto(id, ferramenta string) (Proxecto, error) {
	return engadirAoProxecto(id, ferramenta)
}

// QuitarDoProxecto saca unha ferramenta do proxecto, con copia e datos.
func (a *App) QuitarDoProxecto(id, ferramenta string) (Proxecto, error) {
	return quitarDoProxecto(id, ferramenta)
}

// EliminarProxecto borra un proxecto enteiro (datos, copias e bóveda).
func (a *App) EliminarProxecto(id string) error {
	return eliminarProxecto(id)
}

// AbrirBuxoDoProxecto abre Buxo sobre a bóveda do proxecto (créase ao engadir
// o primeiro PDF).
func (a *App) AbrirBuxoDoProxecto(id string) error {
	p, err := lerProxecto(id)
	if err != nil {
		return err
	}
	raiz, err := rutaBovedaProxecto(id)
	if err != nil {
		return err
	}
	materia := p.Nome
	if p.Materia != "" {
		materia += " — " + p.Materia
	}
	return a.abrirBuxo(raiz, materia, idBovedaProxecto(id))
}

// idBovedaProxecto é o id da bóveda dun proxecto na API.
func idBovedaProxecto(id string) string { return "proxecto:" + id }

// bovedaDe devolve o id (na API) e a ruta da bóveda que usa un tipo de
// documento: a do proxecto aberto, se xa ten bóveda, e se non a conectada a
// ese tipo de documento ("" se ningunha).
func (a *App) bovedaDe(documento string) (id, ruta string) {
	if pid := idProxectoActivo(); pid != "" {
		if r := bovedaDoProxecto(pid); r != "" && !bovedaDesconectadaDoProxecto(pid) {
			return idBovedaProxecto(pid), r
		}
	}
	return documento, a.settings.IABovedas[documento]
}

// BovedaProxectoConectar conecta (true) ou desconecta (false) do asistente a
// bóveda do proxecto aberto. Por defecto está conectada en canto existe.
func (a *App) BovedaProxectoConectar(conectar bool) error {
	pid := idProxectoActivo()
	if pid == "" {
		return fmt.Errorf("non hai ningún proxecto aberto")
	}
	return desconectarBovedaDoProxecto(pid, !conectar)
}

// ListarCarpetasProxectos devolve os cartafoles de proxectos.
func (a *App) ListarCarpetasProxectos() ([]CarpetaProxectos, error) { return lerCarpetasProxectos() }

// CrearCarpetaProxectos crea un cartafol e devolve a lista.
func (a *App) CrearCarpetaProxectos(nome string) ([]CarpetaProxectos, error) {
	return crearCarpetaProxectos(nome)
}

// RenomearCarpetaProxectos cambia o nome dun cartafol.
func (a *App) RenomearCarpetaProxectos(id, nome string) ([]CarpetaProxectos, error) {
	return renomearCarpetaProxectos(id, nome)
}

// EliminarCarpetaProxectos borra un cartafol (os proxectos non se borran).
func (a *App) EliminarCarpetaProxectos(id string) ([]CarpetaProxectos, error) {
	return eliminarCarpetaProxectos(id)
}

// MoverProxecto pon un proxecto nun cartafol ("" = sen cartafol).
func (a *App) MoverProxecto(id, carpeta string) (Proxecto, error) {
	return moverProxecto(id, carpeta)
}
