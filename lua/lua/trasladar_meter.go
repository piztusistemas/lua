package main

// Aplicar «Meter» (spec trasladar). Cada importación garda antes unha versión
// do que pode cambiar (biblioteca, plantillas, os documentos que se
// actualizan e os cartafoles novos, que aínda non existen): se algo falla a
// metade, restáurase e non queda nada a medias; se sae ben, a mesma versión
// serve para «Desfacer».

import (
	"edicion/idioma"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// OpcionsMeter son as escollas do resumo (todas opcionais).
type OpcionsMeter struct {
	// Destino: "aberto" ou "novo" ("" = o que propón o resumo).
	Destino string `json:"destino"`
	// Cursos: índices dos cursos escollidos (nil = todos).
	Cursos []int `json:"cursos"`
	// Accion na volta dun documento: "actualizar" (por defecto) ou "copia".
	Accion string `json:"accion"`
}

// ResultadoMeter é o que se meteu.
type ResultadoMeter struct {
	Proxectos   []string `json:"proxectos"`
	Formularios []string `json:"formularios"`
	Plantillas  []string `json:"plantillas"`
	// Version permite desfacer a importación.
	Version string `json:"version"`
	// AbrirProxecto: proxecto que convén abrir despois ("" = ningún).
	AbrirProxecto string `json:"abrirProxecto,omitempty"`
	// Recargar: cambiou o proxecto aberto (hai que volver lelo).
	Recargar bool `json:"recargar"`
	// Reiniciar: restaurouse unha copia completa (hai que recargar todo).
	Reiniciar bool     `json:"reiniciar"`
	Avisos    []string `json:"avisos,omitempty"`
}

// docMeter é un documento listo para entrar nun proxecto.
type docMeter struct {
	ferramenta string
	contido    []byte
	formulario *Formulario
	plantilla  *bundlePlantilla
	nome       string
	produccion bool
}

// rutasComunsMeter son as que calquera importación pode cambiar.
func rutasComunsMeter() []string {
	out := []string{}
	if lib, err := dirBibliotecaFormularios(); err == nil {
		out = append(out, lib, filepath.Join(filepath.Dir(lib), "biblioteca_formularios.json"))
	}
	if p, err := plantillasPath(); err == nil {
		out = append(out, p)
	}
	if d, err := plantillaImaxesDir("x"); err == nil {
		out = append(out, filepath.Dir(d))
	}
	return out
}

// aplicarMeter mete o ficheiro ou cartafol `ruta`.
func (a *App) aplicarMeter(ruta string, op OpcionsMeter) (ResultadoMeter, error) {
	res := ResultadoMeter{Proxectos: []string{}, Formularios: []string{}, Plantillas: []string{}}
	r, err := a.clasificarMeter(ruta)
	if err != nil {
		return res, err
	}
	switch r.Tipo {
	case "copia":
		return a.restaurarCopia(ruta)
	case "paquete":
		return a.meterPaquete(ruta, r)
	case "boveda":
		return a.meterBoveda(ruta, r, op)
	case "plantilla":
		data, err := os.ReadFile(ruta)
		if err != nil {
			return res, err
		}
		return a.conVersion(r.Nome, nil, &res, func() error {
			id, err := a.meterPlantilla(data)
			res.Plantillas = append(res.Plantillas, id)
			return err
		})
	}
	data, err := os.ReadFile(ruta)
	if err != nil {
		return res, err
	}
	docs, err := docsDe(r, data, op)
	if err != nil {
		return res, err
	}
	if r.Tipo == "formulario" && len(docs) == 0 {
		// Formulario sen datos: só á biblioteca.
		var b BundleFormulario
		if err := json.Unmarshal(data, &b); err != nil {
			return res, err
		}
		return a.conVersion(r.Nome, nil, &res, func() error {
			f, reutilizado, err := meterFormularioBiblioteca(b.Formulario)
			if err == nil {
				res.Formularios = append(res.Formularios, f.ID)
				if reutilizado {
					res.Avisos = append(res.Avisos, idioma.T("go.trasladar.reutilizado", "Xa tiñas un formulario idéntico, «%s»: non se duplicou.", f.Nome))
				}
			}
			return err
		})
	}
	return a.meterDocumentos(r, docs, op)
}

// conVersion garda unha versión de `extra` e das rutas comúns, executa `fn` e,
// se falla, deixa todo como estaba.
func (a *App) conVersion(nome string, extra []string, res *ResultadoMeter, fn func() error) (ResultadoMeter, error) {
	ver, err := gardarVersion("meter", nome, append(rutasComunsMeter(), extra...))
	if err != nil {
		return *res, err
	}
	if err := fn(); err != nil {
		_ = restaurarVersion(ver)
		return *res, err
	}
	res.Version = ver
	return *res, nil
}

// ---------- Documentos ----------

// docsDe converte o ficheiro nos documentos que van entrar.
func docsDe(r ResumoMeter, data []byte, op OpcionsMeter) ([]docMeter, error) {
	switch r.Tipo {
	case "documento":
		var d DocumentoTraslado
		var c cabeceiraJSON
		_ = json.Unmarshal(data, &c)
		if c.Formato != formatoDocumento { // documento vello dun formulario
			return []docMeter{{ferramenta: r.Ferramenta, contido: sanearContidoDocumento(data), nome: strings.TrimSuffix(r.Nome, filepath.Ext(r.Nome)), produccion: r.Produccion}}, nil
		}
		if err := json.Unmarshal(data, &d); err != nil {
			return nil, err
		}
		return []docMeter{{ferramenta: d.Ferramenta, contido: sanearContidoDocumento(d.Documento), formulario: d.Formulario, plantilla: d.Plantilla, nome: d.Nome, produccion: r.Produccion}}, nil
	case "curso":
		escollidos := op.Cursos
		if escollidos == nil {
			for i := range r.Cursos {
				escollidos = append(escollidos, i)
			}
		}
		out := []docMeter{}
		for _, i := range escollidos {
			if i < 0 || i >= len(r.Cursos) {
				return nil, fmt.Errorf("curso %d fóra de rango", i)
			}
			c := r.Cursos[i]
			var contido []byte
			if r.Ferramenta == ferramentaActas {
				contido, _ = json.MarshalIndent(map[string]any{"ferramenta": ferramentaActas, "actas": []json.RawMessage{c}}, "", "  ")
			} else {
				contido, _ = json.MarshalIndent(map[string]any{"plans": []json.RawMessage{c}}, "", "  ")
			}
			out = append(out, docMeter{ferramenta: r.Ferramenta, contido: contido, nome: nomeCurso(r.Ferramenta, c)})
		}
		if len(out) == 0 {
			return nil, idioma.Errorf("go.trasladar.nada_meter", "non escolliches ningún curso")
		}
		return out, nil
	case "formulario":
		var b BundleFormulario
		if err := json.Unmarshal(data, &b); err != nil {
			return nil, err
		}
		if !haiDatosIntercambio(b.Datos) {
			return nil, nil
		}
		contido, _ := json.MarshalIndent(map[string]any{"ferramenta": ferramentaFormulario, "formulario": b.Formulario.ID, "datos": b.Datos}, "", "  ")
		f := b.Formulario
		return []docMeter{{ferramenta: idFerramentaForm(f.ID), contido: contido, formulario: &f, nome: f.Nome, produccion: !b.Datos.Proba}}, nil
	}
	return nil, fmt.Errorf("tipo descoñecido: %q", r.Tipo)
}

// nomeCurso: «Cronograma 2025-2026» / «Actas 2025-2026».
func nomeCurso(ferramenta string, c json.RawMessage) string {
	var p struct {
		AcademicYear string `json:"academicYear"`
	}
	_ = json.Unmarshal(c, &p)
	base := idioma.T("go.trasladar.nome_cronograma", "Cronograma")
	if ferramenta == ferramentaActas {
		base = idioma.T("go.trasladar.nome_actas", "Actas")
	}
	if y := strings.TrimSpace(p.AcademicYear); y != "" {
		return base + " " + y
	}
	return base
}

// meterDocumentos mete un ou varios documentos (volta, proxecto aberto ou novo).
func (a *App) meterDocumentos(r ResumoMeter, docs []docMeter, op OpcionsMeter) (ResultadoMeter, error) {
	res := ResultadoMeter{Proxectos: []string{}, Formularios: []string{}, Plantillas: []string{}}
	aberto := proxectoAbertoValido()
	// Volta: actualizar o documento orixinal.
	if r.Volta != nil && op.Accion != "copia" && len(docs) == 1 {
		destino, err := rutaDocumentoProxecto(r.Volta.Proxecto, r.Volta.Ferramenta)
		if err != nil {
			return res, err
		}
		res.Proxectos = append(res.Proxectos, r.Volta.Proxecto)
		if r.Volta.Proxecto == aberto {
			res.Recargar = true
		} else {
			res.AbrirProxecto = r.Volta.Proxecto
		}
		return a.conVersion(r.Nome, []string{destino}, &res, func() error {
			return escribirAtomicoCrono(destino, docs[0].contido)
		})
	}
	destino := op.Destino
	if destino == "" || destino == "volta" {
		destino = r.Destino
	}
	if destino == "volta" {
		destino = "novo"
		if r.ProxectoAberto != "" && !proxectoTenDocumento(r.ProxectoAberto, docs[0].ferramenta) {
			destino = "aberto"
		}
	}
	if destino == "aberto" && (aberto == "" || len(docs) > 1 || proxectoTenDocumento(aberto, docs[0].ferramenta)) {
		destino = "novo" // nunca se pisa un documento que xa existe
	}
	// Os proxectos novos resérvanse antes, para que «Desfacer» os borre.
	ids, _ := idsEProxectos()
	destinos := make([]string, len(docs))
	extra := []string{}
	for i, d := range docs {
		if destino == "aberto" {
			destinos[i] = aberto
			if ruta, err := rutaDocumentoProxecto(aberto, d.ferramenta); err == nil {
				extra = append(extra, ruta)
			}
			if dir, err := dirProxecto(aberto); err == nil {
				extra = append(extra, filepath.Join(dir, "proxecto.json"), filepath.Join(dir, "formularios"), filepath.Join(dir, "biblioteca_formularios.json"))
			}
			continue
		}
		slug := slugFormulario(d.nome)
		if slug == "" {
			slug = "proxecto"
		}
		id := idLibre(slug, ids)
		ids[id] = true
		destinos[i] = id
		if dir, err := dirProxecto(id); err == nil {
			extra = append(extra, dir)
		}
	}
	return a.conVersion(r.Nome, extra, &res, func() error {
		for i, d := range docs {
			if destino != "aberto" {
				nome := d.nome
				if nome == "" {
					nome = r.Nome
				}
				p := Proxecto{ID: destinos[i], Nome: nomeProxectoLibre(nome), Creado: time.Now().Format("2006-01-02"), Ferramentas: []string{}}
				if err := escribirProxecto(p); err != nil {
					return err
				}
			}
			if err := a.meterDocumentoEn(destinos[i], d, &res); err != nil {
				return err
			}
			res.Proxectos = append(res.Proxectos, destinos[i])
		}
		if destino == "aberto" {
			res.Recargar = true
		} else {
			res.AbrirProxecto = destinos[0]
		}
		return nil
	})
}

// meterDocumentoEn deixa o documento `d` no proxecto `pid`.
func (a *App) meterDocumentoEn(pid string, d docMeter, res *ResultadoMeter) error {
	ferramenta := d.ferramenta
	contido := d.contido
	var copia *Formulario
	if fid, ok := idFerramentaFormulario(ferramenta); ok {
		f := d.formulario
		if f == nil {
			if lf, err := lerFormularioBiblioteca(fid); err == nil {
				f = &lf
			}
		}
		if f == nil {
			// Documento vello dun formulario que só está no proxecto aberto.
			dir, _ := dirProxecto(pid)
			raw, err := os.ReadFile(filepath.Join(dir, "formularios", fid+".json"))
			if err != nil {
				return idioma.Errorf("go.trasladar.sen_formulario", "este documento é dun formulario que non tes e o ficheiro non o trae")
			}
			var pf Formulario
			if err := json.Unmarshal(raw, &pf); err != nil {
				return err
			}
			f = &pf
		}
		lf, _, err := meterFormularioBiblioteca(*f)
		if err != nil {
			return err
		}
		res.Formularios = append(res.Formularios, lf.ID)
		if lf.ID != fid {
			ferramenta = idFerramentaForm(lf.ID)
			contido = cambiarFormularioDoDocumento(contido, lf.ID)
		}
		c := lf
		if d.produccion {
			c.Modo = modoProduccion
		}
		copia = &c
	}
	if err := asegurarFerramentaProxecto(pid, ferramenta, copia); err != nil {
		return err
	}
	ruta, err := rutaDocumentoProxecto(pid, ferramenta)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		return err
	}
	if err := escribirAtomicoCrono(ruta, contido); err != nil {
		return err
	}
	if d.plantilla != nil {
		data, _ := json.Marshal(d.plantilla)
		id, err := a.meterPlantilla(data)
		if err != nil {
			return err
		}
		res.Plantillas = append(res.Plantillas, id)
		return lembrarPlantillaDocumento(ruta, id)
	}
	return nil
}

// cambiarFormularioDoDocumento pon o id do formulario no documento.
func cambiarFormularioDoDocumento(contido []byte, id string) []byte {
	var m map[string]json.RawMessage
	if json.Unmarshal(contido, &m) != nil {
		return contido
	}
	m["formulario"], _ = json.Marshal(id)
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return contido
	}
	return out
}

// asegurarFerramentaProxecto engade a ferramenta ao proxecto (coa copia do
// formulario, se non a ten) e, se o proxecto está aberto, á súa columna.
func asegurarFerramentaProxecto(pid, ferramenta string, copia *Formulario) error {
	p, err := lerProxecto(pid)
	if err != nil {
		return err
	}
	dir, err := dirProxecto(pid)
	if err != nil {
		return err
	}
	if copia != nil {
		ruta := filepath.Join(dir, "formularios", copia.ID+".json")
		if _, err := os.Stat(ruta); os.IsNotExist(err) {
			if err := escribirXSONAtomico(ruta, copia); err != nil {
				return err
			}
		}
	}
	if contemString(p.Ferramentas, ferramenta) {
		return nil
	}
	p.Ferramentas = append(p.Ferramentas, ferramenta)
	if err := escribirProxecto(p); err != nil {
		return err
	}
	if idProxectoActivo() == pid {
		if b, err := lerBiblioteca(); err == nil {
			b.Agochados = quitarString(b.Agochados, ferramenta)
			if !contemString(b.Fixos, ferramenta) {
				b.Fixos = append(b.Fixos, ferramenta)
			}
			_ = escribirBiblioteca(b)
		}
	}
	return nil
}

// ---------- Plantillas ----------

// meterPlantilla mete un paquete de plantilla; se xa hai unha idéntica,
// reutilízaa. Devolve o id local.
func (a *App) meterPlantilla(data []byte) (string, error) {
	var b bundlePlantilla
	if err := json.Unmarshal(data, &b); err != nil {
		return "", idioma.Errorf("go.trasladar.danado", "o ficheiro parece de Lúa pero está danado: %w", err)
	}
	b.Formato = formatoBundlePlantilla
	novo, _ := json.Marshal(b)
	if alm, err := loadAlmacenPlantillas(); err == nil {
		for _, p := range alm.Plantillas {
			if eb, err := construirBundlePlantilla(p); err == nil {
				if actual, _ := json.Marshal(eb); string(actual) == string(novo) {
					return p.ID, nil
				}
			}
		}
	}
	p, err := a.importarBundlePlantilla(data)
	if err != nil {
		return "", err
	}
	return p.ID, nil
}

// ---------- Paquetes ----------

func (a *App) meterPaquete(ruta string, r ResumoMeter) (ResultadoMeter, error) {
	res := ResultadoMeter{Proxectos: []string{}, Formularios: []string{}, Plantillas: []string{}}
	p, err := abrirPaquete(ruta)
	if err != nil {
		return res, err
	}
	defer p.Close()
	base, err := dirProxectos()
	if err != nil {
		return res, err
	}
	ids, _ := idsEProxectos()
	novos := make([]string, len(p.m.Proxectos))
	extra := []string{}
	for i, it := range p.m.Proxectos {
		slug := slugFormulario(it.Nome)
		if slug == "" {
			slug = it.ID
		}
		novos[i] = idLibre(slug, ids)
		ids[novos[i]] = true
		extra = append(extra, filepath.Join(base, novos[i]))
	}
	temporais := []string{}
	defer func() {
		for _, t := range temporais {
			_ = os.RemoveAll(t)
		}
	}()
	return a.conVersion(r.Nome, extra, &res, func() error {
		mapa := map[string]string{}
		for _, it := range p.m.Plantillas {
			data, err := p.ler("plantillas/"+it.ID+".json", maxBytesBundleFormulario)
			if err != nil {
				return err
			}
			id, err := a.meterPlantilla(data)
			if err != nil {
				return err
			}
			mapa[it.ID] = id
			res.Plantillas = append(res.Plantillas, id)
		}
		for _, it := range p.m.Formularios {
			data, err := p.ler("formularios/"+it.ID+".json", maxBytesBundleFormulario)
			if err != nil {
				return err
			}
			var f Formulario
			if err := json.Unmarshal(data, &f); err != nil {
				return idioma.Errorf("go.trasladar.danado", "o ficheiro parece de Lúa pero está danado: %w", err)
			}
			lf, _, err := meterFormularioBiblioteca(f)
			if err != nil {
				return err
			}
			res.Formularios = append(res.Formularios, lf.ID)
		}
		if err := os.MkdirAll(base, 0o755); err != nil {
			return err
		}
		for i, it := range p.m.Proxectos {
			tmp := filepath.Join(base, ".meter-"+novos[i])
			temporais = append(temporais, tmp)
			_ = os.RemoveAll(tmp)
			if err := p.extraer("proxectos/"+it.ID, tmp); err != nil {
				return err
			}
			raw, err := os.ReadFile(filepath.Join(tmp, "proxecto.json"))
			if err != nil {
				return idioma.Errorf("go.trasladar.danado", "o ficheiro parece de Lúa pero está danado: %w", err)
			}
			var pr Proxecto
			if err := json.Unmarshal(raw, &pr); err != nil {
				return idioma.Errorf("go.trasladar.danado", "o ficheiro parece de Lúa pero está danado: %w", err)
			}
			for _, f := range pr.Ferramentas {
				if !ferramentaProxectoValida(f) {
					return idioma.Errorf("go.trasladar.danado", "o ficheiro parece de Lúa pero está danado: %w", fmt.Errorf("ferramenta %q", f))
				}
			}
			pr.ID, pr.Nome, pr.Carpeta = novos[i], nomeProxectoLibre(pr.Nome), ""
			if pr.Ferramentas == nil {
				pr.Ferramentas = []string{}
			}
			if err := escribirXSONAtomico(filepath.Join(tmp, "proxecto.json"), pr); err != nil {
				return err
			}
			if err := os.Rename(tmp, filepath.Join(base, novos[i])); err != nil {
				return err
			}
			for _, f := range pr.Ferramentas {
				if pl, ok := mapa[p.m.PlantillaDocumento[it.ID+"/"+f]]; ok {
					if ruta, err := rutaDocumentoProxecto(novos[i], f); err == nil {
						_ = lembrarPlantillaDocumento(ruta, pl)
					}
				}
			}
			res.Proxectos = append(res.Proxectos, novos[i])
		}
		if len(novos) > 0 {
			res.AbrirProxecto = novos[0]
		}
		return nil
	})
}

// restaurarCopia volve deixar Lúa como na copia completa. Antes garda unha
// versión de todo o que substitúe.
func (a *App) restaurarCopia(ruta string) (ResultadoMeter, error) {
	res := ResultadoMeter{Proxectos: []string{}, Formularios: []string{}, Plantillas: []string{}, Reiniciar: true}
	p, err := abrirPaquete(ruta)
	if err != nil {
		return res, err
	}
	defer p.Close()
	if !p.m.Completa {
		return res, errors.New("o ficheiro non é unha copia completa")
	}
	cfg, err := dirConfigLua()
	if err != nil {
		return res, err
	}
	base, err := dirProxectos()
	if err != nil {
		return res, err
	}
	lib, err := dirBibliotecaFormularios()
	if err != nil {
		return res, err
	}
	// Primeiro todo a un temporal: se o ficheiro está mal, non se toca nada.
	tmp := filepath.Join(cfg, "restaurar.tmp")
	_ = os.RemoveAll(tmp)
	defer os.RemoveAll(tmp)
	if err := p.extraer("proxectos", filepath.Join(tmp, "proxectos")); err != nil {
		return res, err
	}
	if err := p.extraer("formularios", filepath.Join(tmp, "formularios")); err != nil {
		return res, err
	}
	for _, it := range p.m.Proxectos {
		if _, err := os.Stat(filepath.Join(tmp, "proxectos", it.ID, "proxecto.json")); err != nil {
			return res, idioma.Errorf("go.trasladar.danado", "o ficheiro parece de Lúa pero está danado: %w", err)
		}
	}
	rutas := append(rutasComunsMeter(), base)
	ver, err := gardarVersion("restaurar", filepath.Base(ruta), rutas)
	if err != nil {
		return res, err
	}
	_, _ = activarProxecto("")
	fallo := func(err error) (ResultadoMeter, error) {
		_ = restaurarVersion(ver)
		return res, err
	}
	if err := os.RemoveAll(base); err != nil {
		return fallo(err)
	}
	if err := os.MkdirAll(filepath.Dir(base), 0o755); err != nil {
		return fallo(err)
	}
	if err := moverOuCopiar(filepath.Join(tmp, "proxectos"), base); err != nil {
		return fallo(err)
	}
	if d, err := p.ler("configuracion/carpetas.json", maxBytesManifesto); err == nil {
		if c, err := rutaCarpetasProxectos(); err == nil {
			_ = escribirAtomicoCrono(c, d)
		}
	}
	if err := os.RemoveAll(lib); err != nil {
		return fallo(err)
	}
	if err := moverOuCopiar(filepath.Join(tmp, "formularios"), lib); err != nil {
		return fallo(err)
	}
	org := filepath.Join(filepath.Dir(lib), "biblioteca_formularios.json")
	_ = os.Remove(org)
	if d, err := p.ler("configuracion/biblioteca_formularios.json", maxBytesManifesto); err == nil {
		_ = escribirAtomicoCrono(org, d)
	}
	mapa := map[string]string{}
	for _, it := range p.m.Plantillas {
		data, err := p.ler("plantillas/"+it.ID+".json", maxBytesBundleFormulario)
		if err != nil {
			return fallo(err)
		}
		id, err := a.meterPlantilla(data)
		if err != nil {
			return fallo(err)
		}
		mapa[it.ID] = id
	}
	for clave, pl := range p.m.PlantillaDocumento {
		pid, ferr, ok := strings.Cut(clave, "/")
		if !ok {
			continue
		}
		if ruta, err := rutaDocumentoProxecto(pid, ferr); err == nil && mapa[pl] != "" {
			_ = lembrarPlantillaDocumento(ruta, mapa[pl])
		}
	}
	if d, err := p.ler("configuracion/settings.json", maxBytesManifesto); err == nil {
		var s Settings
		if json.Unmarshal(d, &s) == nil {
			// O que é deste equipo non se toca: clave, instalación e bóvedas.
			s.IAAPIKey, s.Instalacion, s.IABovedas = a.settings.IAAPIKey, a.settings.Instalacion, a.settings.IABovedas
			a.settings = s
			_ = saveSettingsToDisk(s)
		}
	}
	for _, it := range p.m.Proxectos {
		res.Proxectos = append(res.Proxectos, it.ID)
	}
	res.Version = ver
	return res, nil
}

// moverOuCopiar move un cartafol (ou o copia se está noutro disco). Se a
// orixe non existe, crea o destino baleiro.
func moverOuCopiar(orixe, destino string) error {
	if _, err := os.Stat(orixe); os.IsNotExist(err) {
		return os.MkdirAll(destino, 0o755)
	}
	if err := os.Rename(orixe, destino); err == nil {
		return nil
	}
	return copiarArbore(orixe, destino)
}

// ---------- Bóveda ----------

// meterBoveda copia PDF e Markdown (ou unha bóveda enteira) á bóveda do
// proxecto aberto, ou dun proxecto novo. Nunca pisa un ficheiro: un nome
// repetido entra coma «nome (2).pdf».
func (a *App) meterBoveda(ruta string, r ResumoMeter, op OpcionsMeter) (ResultadoMeter, error) {
	res := ResultadoMeter{Proxectos: []string{}, Formularios: []string{}, Plantillas: []string{}}
	aberto := proxectoAbertoValido()
	pid := aberto
	novo := aberto == "" || op.Destino == "novo"
	extra := []string{}
	if novo {
		ids, _ := idsEProxectos()
		slug := slugFormulario(strings.TrimSuffix(r.Nome, filepath.Ext(r.Nome)))
		if slug == "" {
			slug = "proxecto"
		}
		pid = idLibre(slug, ids)
		if dir, err := dirProxecto(pid); err == nil {
			extra = append(extra, dir)
		}
	}
	raiz, err := rutaBovedaProxecto(pid)
	if err != nil {
		return res, err
	}
	copias, err := planBoveda(ruta, raiz)
	if err != nil {
		return res, err
	}
	for _, c := range copias {
		extra = append(extra, c[1])
	}
	for _, d := range []string{"raw", "wiki", "pdf"} {
		extra = append(extra, filepath.Join(raiz, d))
	}
	return a.conVersion(r.Nome, extra, &res, func() error {
		if novo {
			p := Proxecto{ID: pid, Nome: nomeProxectoLibre(strings.TrimSuffix(r.Nome, filepath.Ext(r.Nome))), Creado: time.Now().Format("2006-01-02"), Ferramentas: []string{}}
			if err := escribirProxecto(p); err != nil {
				return err
			}
			res.AbrirProxecto = pid
		} else {
			res.Recargar = true
		}
		for _, d := range []string{"raw", "wiki", "pdf"} {
			if err := os.MkdirAll(filepath.Join(raiz, d), 0o755); err != nil {
				return err
			}
		}
		for _, c := range copias {
			if err := copiarFicheiro(c[0], c[1], 0o644); err != nil {
				return err
			}
		}
		res.Proxectos = append(res.Proxectos, pid)
		return nil
	})
}

// planBoveda decide onde vai cada ficheiro: unha bóveda (raw/, wiki/)
// consérvase tal cal; as fontes soltas van a pdf/ (PDF) e raw/ (Markdown).
func planBoveda(orixe, raiz string) ([][2]string, error) {
	usados := map[string]bool{}
	libre := func(dest string) string {
		ext := filepath.Ext(dest)
		base := strings.TrimSuffix(dest, ext)
		c := dest
		for n := 2; usados[c] || fileExists(c); n++ {
			c = fmt.Sprintf("%s (%d)%s", base, n, ext)
		}
		usados[c] = true
		return c
	}
	out := [][2]string{}
	st, err := os.Stat(orixe)
	if err != nil {
		return nil, err
	}
	if !st.IsDir() {
		pdf, _ := eFonteBoveda(orixe)
		sub := "raw"
		if pdf {
			sub = "pdf"
		}
		return [][2]string{{orixe, libre(filepath.Join(raiz, sub, filepath.Base(orixe)))}}, nil
	}
	eBoveda := false
	if _, err := validarRaizBoveda(orixe); err == nil {
		eBoveda = true
	}
	err = filepath.WalkDir(orixe, func(p string, e fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if e.IsDir() {
			if strings.HasPrefix(e.Name(), ".") && p != orixe {
				return filepath.SkipDir
			}
			return nil
		}
		if e.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		if len(out) >= maxFicheirosBoveda {
			return idioma.Errorf("go.trasladar.moitos_cartafol", "o cartafol ten demasiados ficheiros (máximo %d)", maxFicheirosBoveda)
		}
		if eBoveda {
			rel, err := filepath.Rel(orixe, p)
			if err != nil {
				return err
			}
			out = append(out, [2]string{p, libre(filepath.Join(raiz, rel))})
			return nil
		}
		pdf, md := eFonteBoveda(e.Name())
		switch {
		case pdf:
			out = append(out, [2]string{p, libre(filepath.Join(raiz, "pdf", e.Name()))})
		case md:
			out = append(out, [2]string{p, libre(filepath.Join(raiz, "raw", e.Name()))})
		}
		return nil
	})
	return out, err
}
