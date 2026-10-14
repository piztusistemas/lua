package main

// O paquete .lua: un zip con manifest.json e o contido baixo proxectos/,
// formularios/, plantillas/ e (só na copia completa) configuracion/. Ao lelo
// rexéitanse as rutas perigosas (absolutas, con «..», ligazóns) e ponse
// límite ao número de entradas e ao tamaño descomprimido.

import (
	"archive/zip"
	"edicion/idioma"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	maxEntradasPaquete = 50000
	maxBytesPaquete    = 4 << 30 // descomprimido, en total
	maxBytesEntrada    = 1 << 30
	maxBytesManifesto  = 4 << 20
)

// eZip di se o ficheiro empeza coma un zip.
func eZip(ruta string) bool {
	f, err := os.Open(ruta)
	if err != nil {
		return false
	}
	defer f.Close()
	cab := make([]byte, 4)
	if _, err := io.ReadFull(f, cab); err != nil {
		return false
	}
	return string(cab) == "PK\x03\x04"
}

// ---------- Escribir ----------

// resolverSeleccion completa a selección: na copia completa vai todo.
func resolverSeleccion(sel SeleccionSacar) (SeleccionSacar, error) {
	switch sel.Tipo {
	case "todo":
		sel.Datos = true
		sel.Proxectos, sel.Formularios, sel.Plantillas = nil, nil, nil
		if ps, err := listarProxectos(); err == nil {
			for _, p := range ps {
				sel.Proxectos = append(sel.Proxectos, p.ID)
			}
		}
		fs, err := formulariosBiblioteca()
		if err != nil {
			return sel, err
		}
		for _, f := range fs {
			sel.Formularios = append(sel.Formularios, f.ID)
		}
		alm, err := loadAlmacenPlantillas()
		if err != nil {
			return sel, err
		}
		for _, p := range alm.Plantillas {
			sel.Plantillas = append(sel.Plantillas, p.ID)
		}
	case "proxecto", "seleccion":
	default:
		return sel, fmt.Errorf("tipo de selección descoñecido: %q", sel.Tipo)
	}
	if len(sel.Proxectos)+len(sel.Formularios)+len(sel.Plantillas) == 0 {
		return sel, idioma.Errorf("go.trasladar.nada", "non escolliches nada que sacar")
	}
	return sel, nil
}

// escribirPaqueteEn escribe o paquete na ruta (temporal e renomear: un erro a
// metade non deixa un ficheiro incompleto).
func escribirPaqueteEn(ruta string, sel SeleccionSacar, s Settings) (ManifestoPaquete, error) {
	tmp := ruta + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return ManifestoPaquete{}, err
	}
	m, err := escribirPaquete(f, sel, s)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.Remove(tmp)
		return m, err
	}
	if err := os.Rename(tmp, ruta); err != nil {
		_ = os.Remove(tmp)
		return m, err
	}
	return m, nil
}

func escribirPaquete(w io.Writer, sel SeleccionSacar, s Settings) (ManifestoPaquete, error) {
	sel, err := resolverSeleccion(sel)
	if err != nil {
		return ManifestoPaquete{}, err
	}
	m := ManifestoPaquete{
		Formato: formatoPaquete, Version: versionPaquete, Creado: time.Now().Format(time.RFC3339),
		Instalacion: s.Instalacion, Completa: sel.Tipo == "todo", Datos: sel.Datos,
		Proxectos: []ItemPaquete{}, Formularios: []ItemPaquete{}, Plantillas: []ItemPaquete{},
		PlantillaDocumento: map[string]string{},
	}
	alm, err := loadAlmacenPlantillas()
	if err != nil {
		return m, err
	}
	plantillas := map[string]bool{}
	for _, id := range sel.Plantillas {
		plantillas[id] = true
	}
	zw := zip.NewWriter(w)
	vistos := map[string]bool{}
	for _, id := range sel.Proxectos {
		if vistos[id] {
			continue
		}
		vistos[id] = true
		p, err := lerProxecto(id)
		if err != nil {
			return m, err
		}
		dir, _ := dirProxecto(id)
		if err := engadirCartafolZip(zw, dir, "proxectos/"+id, func(rel string) bool {
			return !sel.Datos && (rel == "datos" || strings.HasPrefix(rel, "datos/"))
		}); err != nil {
			return m, err
		}
		m.Proxectos = append(m.Proxectos, ItemPaquete{ID: id, Nome: p.Nome})
		// A plantilla de cada documento do proxecto viaxa con el.
		for _, f := range p.Ferramentas {
			if pl := plantillaDeDocumento(alm, filepath.Join(dir, "datos", borradorFerramentaNome(f))); pl != nil {
				plantillas[pl.ID] = true
				m.PlantillaDocumento[id+"/"+f] = pl.ID
			}
		}
	}
	for _, id := range sel.Formularios {
		f, err := lerFormularioBiblioteca(id)
		if err != nil {
			return m, err
		}
		f.Modo = ""
		if err := engadirJSONZip(zw, "formularios/"+id+".json", f); err != nil {
			return m, err
		}
		m.Formularios = append(m.Formularios, ItemPaquete{ID: id, Nome: f.Nome})
	}
	for _, id := range ordenados(plantillas) {
		var atopada *PlantillaLatex
		for i := range alm.Plantillas {
			if alm.Plantillas[i].ID == id {
				atopada = &alm.Plantillas[i]
			}
		}
		if atopada == nil {
			continue
		}
		b, err := construirBundlePlantilla(*atopada)
		if err != nil {
			return m, err
		}
		if err := engadirJSONZip(zw, "plantillas/"+id+".json", b); err != nil {
			return m, err
		}
		m.Plantillas = append(m.Plantillas, ItemPaquete{ID: id, Nome: atopada.Nome})
	}
	if m.Completa {
		// Opcións SEN claves (constitución 5.1).
		s.IAAPIKey = ""
		if err := engadirJSONZip(zw, "configuracion/settings.json", s); err != nil {
			return m, err
		}
		if lib, err := dirBibliotecaFormularios(); err == nil {
			_ = engadirFicheiroZip(zw, filepath.Join(filepath.Dir(lib), "biblioteca_formularios.json"), "configuracion/biblioteca_formularios.json")
		}
	}
	if err := engadirJSONZip(zw, "manifest.json", m); err != nil {
		return m, err
	}
	return m, zw.Close()
}

func engadirJSONZip(zw *zip.Writer, nome string, v any) error {
	d, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	w, err := zw.Create(nome)
	if err != nil {
		return err
	}
	_, err = w.Write(d)
	return err
}

// engadirFicheiroZip engade un ficheiro se existe (se non, non fai nada).
func engadirFicheiroZip(zw *zip.Writer, orixe, nome string) error {
	f, err := os.Open(orixe)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()
	w, err := zw.Create(nome)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, f)
	return err
}

// engadirCartafolZip engade un cartafol enteiro baixo `prefixo` (sen ligazóns
// nin temporais). `saltar` recibe a ruta relativa con barras.
func engadirCartafolZip(zw *zip.Writer, dir, prefixo string, saltar func(string) bool) error {
	return filepath.WalkDir(dir, func(p string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil || rel == "." {
			return err
		}
		rel = filepath.ToSlash(rel)
		if saltar != nil && saltar(rel) {
			if e.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if e.IsDir() || e.Type()&fs.ModeSymlink != 0 || strings.HasSuffix(e.Name(), ".tmp") {
			return nil
		}
		return engadirFicheiroZip(zw, p, prefixo+"/"+rel)
	})
}

// ---------- Ler ----------

// paqueteAberto é un paquete xa validado.
type paqueteAberto struct {
	zr      *zip.ReadCloser
	m       ManifestoPaquete
	porNome map[string]*zip.File
}

func (p *paqueteAberto) Close() error { return p.zr.Close() }

// nomeSeguroZip: ruta relativa, con barras, sen «..», sen letra de unidade.
func nomeSeguroZip(n string) bool {
	if n == "" || strings.ContainsAny(n, "\\:\x00") || strings.HasPrefix(n, "/") {
		return false
	}
	limpo := strings.TrimSuffix(n, "/")
	if path.Clean(limpo) != limpo {
		return false
	}
	for _, parte := range strings.Split(limpo, "/") {
		if parte == ".." || parte == "." || parte == "" {
			return false
		}
	}
	return true
}

func abrirPaquete(ruta string) (*paqueteAberto, error) {
	zr, err := zip.OpenReader(ruta)
	if err != nil {
		return nil, idioma.Errorf("go.trasladar.danado", "o ficheiro parece de Lúa pero está danado: %w", err)
	}
	p := &paqueteAberto{zr: zr, porNome: map[string]*zip.File{}}
	fallo := func(err error) (*paqueteAberto, error) {
		zr.Close()
		return nil, err
	}
	if len(zr.File) > maxEntradasPaquete {
		return fallo(idioma.Errorf("go.trasladar.moitos", "o ficheiro ten demasiados elementos (máximo %d)", maxEntradasPaquete))
	}
	var total uint64
	for _, f := range zr.File {
		if !nomeSeguroZip(f.Name) || f.Mode()&fs.ModeSymlink != 0 {
			return fallo(idioma.Errorf("go.trasladar.perigoso", "o ficheiro ten rutas non permitidas e non se mete por seguridade"))
		}
		if f.UncompressedSize64 > maxBytesEntrada {
			return fallo(idioma.Errorf("go.ficheiro.grande", "o ficheiro é demasiado grande (máximo %d MB)", maxBytesEntrada>>20))
		}
		total += f.UncompressedSize64
		p.porNome[strings.TrimSuffix(f.Name, "/")] = f
	}
	if total > maxBytesPaquete {
		return fallo(idioma.Errorf("go.ficheiro.grande", "o ficheiro é demasiado grande (máximo %d MB)", maxBytesPaquete>>20))
	}
	d, err := p.ler("manifest.json", maxBytesManifesto)
	if err != nil {
		return fallo(idioma.Errorf("go.trasladar.non_lua", "este ficheiro non é de Lúa nin un formato que Lúa saiba ler (ou está danado)"))
	}
	if err := json.Unmarshal(d, &p.m); err != nil || p.m.Formato != formatoPaquete {
		return fallo(idioma.Errorf("go.trasladar.non_lua", "este ficheiro non é de Lúa nin un formato que Lúa saiba ler (ou está danado)"))
	}
	if p.m.Version < 1 || p.m.Version > versionPaquete {
		return fallo(idioma.Errorf("go.trasladar.version", "o ficheiro ten a versión %d do formato e este Lúa só entende ata a %d: actualiza Lúa", p.m.Version, versionPaquete))
	}
	for _, it := range p.m.Proxectos {
		if !idFormularioRe.MatchString(it.ID) {
			return fallo(idioma.Errorf("go.trasladar.perigoso", "o ficheiro ten rutas non permitidas e non se mete por seguridade"))
		}
	}
	for _, it := range p.m.Formularios {
		if !idFormularioRe.MatchString(it.ID) {
			return fallo(idioma.Errorf("go.trasladar.perigoso", "o ficheiro ten rutas non permitidas e non se mete por seguridade"))
		}
	}
	return p, nil
}

// ler devolve unha entrada (como moito max bytes).
func (p *paqueteAberto) ler(nome string, max int64) ([]byte, error) {
	f, ok := p.porNome[nome]
	if !ok {
		return nil, fmt.Errorf("falta %s", nome)
	}
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	d, err := io.ReadAll(io.LimitReader(rc, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(d)) > max {
		return nil, errors.New("entrada demasiado grande")
	}
	return d, nil
}

// entradas devolve os nomes (ficheiros) baixo `prefixo/`, en orde.
func (p *paqueteAberto) entradas(prefixo string) []string {
	out := []string{}
	for n, f := range p.porNome {
		if strings.HasPrefix(n, prefixo+"/") && !f.FileInfo().IsDir() {
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out
}

// extraer copia as entradas de `prefixo/` a `destino`.
func (p *paqueteAberto) extraer(prefixo, destino string) error {
	for _, n := range p.entradas(prefixo) {
		rel := strings.TrimPrefix(n, prefixo+"/")
		dest := filepath.Join(destino, filepath.FromSlash(rel))
		if !dentroDe(destino, dest) {
			return idioma.Errorf("go.trasladar.perigoso", "o ficheiro ten rutas non permitidas e non se mete por seguridade")
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		rc, err := p.porNome[n].Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(out, io.LimitReader(rc, maxBytesEntrada))
		rc.Close()
		if cerr := out.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func dentroDe(base, p string) bool {
	rel, err := filepath.Rel(base, p)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

// clasificarPaquete resume un paquete .lua (proxecto, selección ou copia).
func (a *App) clasificarPaquete(r ResumoMeter, ruta string) (ResumoMeter, error) {
	p, err := abrirPaquete(ruta)
	if err != nil {
		return r, err
	}
	defer p.Close()
	r.Tipo, r.Destino = "paquete", "novo"
	if p.m.Completa {
		r.Tipo, r.Destino = "copia", "restaurar"
	}
	r.ConDatos = p.m.Datos
	for _, it := range p.m.Proxectos {
		pdfs := len(p.entradas("proxectos/" + it.ID + "/boveda/pdf"))
		forms := len(p.entradas("proxectos/" + it.ID + "/formularios"))
		r.Elementos = append(r.Elementos, ElementoMeter{Tipo: "proxecto", Nome: it.Nome, Detalle: fmt.Sprintf("%d/%d", forms, pdfs)})
		if p.m.Datos && proxectoPaqueteEnProduccion(p, it.ID) {
			r.Produccion = true
		}
	}
	for _, it := range p.m.Formularios {
		r.Elementos = append(r.Elementos, ElementoMeter{Tipo: "formulario", Nome: it.Nome})
	}
	for _, it := range p.m.Plantillas {
		r.Elementos = append(r.Elementos, ElementoMeter{Tipo: "plantilla", Nome: it.Nome})
	}
	return r, nil
}

// proxectoPaqueteEnProduccion: algún formulario do proxecto está en produción.
func proxectoPaqueteEnProduccion(p *paqueteAberto, id string) bool {
	for _, n := range p.entradas("proxectos/" + id + "/formularios") {
		d, err := p.ler(n, maxBytesBundleFormulario)
		if err != nil {
			continue
		}
		var f Formulario
		if json.Unmarshal(d, &f) == nil && f.enProduccion() {
			return true
		}
	}
	return false
}
