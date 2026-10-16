package main

// Exportación NATIVA a Word (.docx) e OpenDocument (.odt): o informe escríbese
// directamente en XML sobre unha folla de estilos propia (Título, Encabezado 1,
// Corpo de texto, Cita, Lenda...) en vez de pasar por Pandoc, así o documento
// sae xa con formato coidado e imprimible, sen ter que darlle estilos á man.
// É a mesma maqueta que xera a app de escritorio Piztu Orienta
// (cronograma/internal/export/docx.go e odt.go), adaptada aos tipos de Lua.
//
// Este ficheiro ten o que comparten os dous escritores: os datos xa resoltos
// do informe (datosOfimatica), as imaxes e o cronograma anual coma PNG
// (informe_cronopng.go). Ver informe_docx.go e informe_odt.go.

import (
	"bytes"
	"edicion/idioma"
	"fmt"
	"image"
	"image/gif"
	_ "image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// imaxeOfi é unha imaxe lista para meter nun paquete DOCX/ODT. Só se
// incrustan PNG e JPEG tal cales; un GIF convértese a PNG.
type imaxeOfi struct {
	Datos []byte
	Ext   string // "png" ou "jpg"
	Mime  string
	W, H  int // píxeles
}

// imaxeOfiDeURI decodifica un data URI. Devolve erro se non é unha imaxe que
// Word/LibreOffice lean sen máis (PNG, JPEG ou GIF).
func imaxeOfiDeURI(uri string) (*imaxeOfi, error) {
	raw, err := decodificarDataURICrono(uri)
	if err != nil {
		return nil, err
	}
	return imaxeOfiDeBytes(raw)
}

func imaxeOfiDeBytes(raw []byte) (*imaxeOfi, error) {
	cfg, formato, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("formato de imaxe non recoñecido")
	}
	switch formato {
	case "png":
		return &imaxeOfi{Datos: raw, Ext: "png", Mime: "image/png", W: cfg.Width, H: cfg.Height}, nil
	case "jpeg":
		return &imaxeOfi{Datos: raw, Ext: "jpg", Mime: "image/jpeg", W: cfg.Width, H: cfg.Height}, nil
	case "gif":
		g, err := gif.Decode(bytes.NewReader(raw))
		if err != nil {
			return nil, fmt.Errorf("GIF non lexible")
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, g); err != nil {
			return nil, err
		}
		return &imaxeOfi{Datos: buf.Bytes(), Ext: "png", Mime: "image/png", W: cfg.Width, H: cfg.Height}, nil
	}
	return nil, fmt.Errorf("formato de imaxe non soportado (%s)", formato)
}

// resolverImaxeOfi busca a imaxe dunha descrición: `ref` é a clave do mapa da
// tarefa ou un data URI directamente. Se non se pode usar engade un aviso e
// devolve nil.
func resolverImaxeOfi(ref string, imaxes map[string]string, avisos *[]string) *imaxeOfi {
	uri := ref
	if !strings.HasPrefix(ref, "data:") {
		var ok bool
		if uri, ok = imaxes[ref]; !ok {
			*avisos = engadirSenRepetir(*avisos, []string{"unha descrición fai referencia a unha imaxe («" + ref + "») que non existe: omitiuse"})
			return nil
		}
	}
	im, err := imaxeOfiDeURI(uri)
	if err != nil {
		*avisos = engadirSenRepetir(*avisos, []string{idioma.T("go.ofi.imaxe_non_usada", "a imaxe «%s» non se puido usar (%v): omitiuse", ref, err)})
		return nil
	}
	return im
}

// alinImaxeOfi devolve o aliñamento dunha figura: o centrado é o normal e xa
// vén no estilo, así que só se distinguen esquerda e dereita.
func alinImaxeOfi(a cronoAlinamento) cronoAlinamento {
	if a == cronoAlinEsquerda || a == cronoAlinDereita {
		return a
	}
	return cronoAlinDefecto
}

// textoTramosOfi xunta o texto visible dos tramos (sen negriña nin cursiva),
// coa URL entre parénteses nas ligazóns: no papel non se pode premer.
func textoTramosOfi(tramos []cronoTramo) string {
	var b strings.Builder
	for _, t := range tramos {
		b.WriteString(t.cronoTexto())
	}
	return b.String()
}

// datosOfimatica é o informe xa resolto: textos, meses e imaxes.
type datosOfimatica struct {
	Plan         PlanCrono
	Centro       string
	Localidade   string
	Curso        string
	Data         string // "18 de setembro de 2026"
	Xefe         string
	Cargo        string
	Departamento string
	// Tx resolve os textos fixos do documento (os da plantilla ou os de Lua).
	Tx         textosDocumento
	Intro      string
	Meses      []mesCalendario
	ConTarefas []mesCalendario
	Nomes      []string // título de cada mes de ConTarefas no desenvolvemento
	Logo       *imaxeOfi
	CronoPNG   []byte
	Avisos     []string
	// Deseño que vén da plantilla (plantillas_ofimatica.go).
	Estilo    estiloOfi
	Portada   []linhaPortada
	Cabeceira caixaOfi
	Pe        caixaOfi
	// ImaxesPlantilla: as imaxes do cartafol da plantilla que usa a portada.
	ImaxesPlantilla map[string]*imaxeOfi
	// cabOdt e peOdt: cabeceira e pé xa construídos para o ODT (informe_odt.go).
	cabOdt, peOdt string
	// Só nas actas (informe_actas.go): as actas xa resoltas e se o documento é
	// o curso enteiro (con portada e índice) ou unha soa acta.
	Actas          []actaResolta
	ActasCurso     bool
	ActaSecretario string
	// Só nos formularios propios (informe_formulario_ofi.go).
	Formulario *docFormulario
	// Formulas: as fórmulas do documento xa compoñidas coma imaxes.
	Formulas *formulasOfi
}

// baseOfimatica resolve o que comparten o cronograma e as actas: o deseño de
// Word/OpenDocument da plantilla (estilo, portada, cabeceira e pé), o
// logotipo e as imaxes da plantilla que usa ese deseño.
func baseOfimatica(plantilla *PlantillaLatex, vars map[string]string, tx textosDocumento, logo string, avisos []string) datosOfimatica {
	ofi := OfimaticaPlantilla{}
	if plantilla != nil {
		ofi = ofimaticaDoDeseno(plantilla.Ofimatica, plantilla.Deseno)
	}
	pl := plantilla
	if pl == nil {
		pl = &PlantillaLatex{}
	}
	subst := func(t string) string {
		r, _ := substituirVariables(t, pl, vars, "", nil)
		return strings.TrimSpace(r)
	}
	dslPortada := ofi.Portada
	if strings.TrimSpace(dslPortada) == "" {
		dslPortada = portadaOfimaticaPorDefecto
	}
	d := datosOfimatica{
		Estilo:    estiloOfimatica(ofi),
		Portada:   parsearPortadaOfi(dslPortada, subst),
		Cabeceira: parsearCaixaOfi(ofi.Cabeceira, subst, "dereita"),
		Pe:        parsearCaixaOfi(ofi.Pe, subst, "centro"),
		Tx:        tx,
		Avisos:    avisos,
	}
	if logo != "" {
		im, err := imaxeOfiDeURI(logo)
		if err != nil {
			d.Avisos = append(d.Avisos, idioma.T("go.ofi.logotipo", "o logotipo non se puido usar (%v): omitiuse", err))
		} else {
			d.Logo = im
		}
	}
	// Imaxes da plantilla que piden a portada, a cabeceira ou o pé (@imaxe nome.png).
	for _, nome := range nomesImaxesOfi(d.Portada, d.Cabeceira, d.Pe) {
		if d.ImaxesPlantilla == nil {
			d.ImaxesPlantilla = map[string]*imaxeOfi{}
		}
		im, err := imaxeDaPlantillaOfi(pl, nome)
		if err != nil {
			d.Avisos = append(d.Avisos, idioma.T("go.ofi.imaxe_deseno", "o deseño de Word/OpenDocument da plantilla usa a imaxe «%s», que non se puido usar (%v): omitiuse", nome, err))
			continue
		}
		d.ImaxesPlantilla[nome] = im
	}
	return d
}

// prepararOfimatica resolve todo o que precisan os escritores DOCX/ODT.
func prepararOfimatica(req XerarInformeCronogramaRequest, plantilla *PlantillaLatex, agora time.Time) (datosOfimatica, error) {
	p := req.Plan
	vars := variablesCorpoCrono(p, req.NomeBase, agora)
	tx := novosTextosDoc(plantilla, vars)
	tx.engadirVariables(vars, "TITULO_CRONOGRAMA")
	intro, avisos := introCrono(p, plantilla, vars)
	meses := calendarioCrono(p.Config, p.MonthsData)
	conTarefas := mesesConTarefasCrono(meses)
	d := baseOfimatica(plantilla, vars, tx, p.Config.SchoolLogo, avisos)
	d.Plan = p
	d.Centro = porDefectoCrono(p.Config.SchoolName, "Nome do centro")
	d.Localidade = porDefectoCrono(p.Config.Location, "Localidade")
	d.Curso = p.AcademicYear
	d.Data = dataGalegaCrono(agora)
	d.Xefe = porDefectoCrono(p.Config.HeadOfDepartment, "Nome e apelidos")
	d.Cargo = cargoSinaturaCrono(p)
	d.Departamento = vars["DEPARTAMENTO"]
	d.Intro = intro
	d.Meses = meses
	d.ConTarefas = conTarefas
	d.Nomes = nomesMesesInformeCrono(conTarefas)
	png, err := renderCronogramaPNGCrono(p, meses, tx)
	if err != nil {
		return d, err
	}
	d.CronoPNG = png
	return d, nil
}

// imaxeDaPlantillaOfi le unha imaxe do cartafol de imaxes da plantilla.
func imaxeDaPlantillaOfi(p *PlantillaLatex, nome string) (*imaxeOfi, error) {
	if strings.TrimSpace(p.ID) == "" {
		return nil, idioma.Errorf("go.ofi.sen_imaxes", "a plantilla aínda non ten imaxes (gárdaa primeiro)")
	}
	dir, err := plantillaImaxesDir(p.ID)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(filepath.Join(dir, filepath.Base(nome)))
	if err != nil {
		return nil, fmt.Errorf("non existe no cartafol de imaxes da plantilla")
	}
	// Word e OpenDocument non incrustan PDF: un logotipo en PDF (vectorial
	// para o LaTeX) úsase aquí coma unha imaxe de píxeles a 300 ppp.
	if bytes.HasPrefix(b, []byte("%PDF")) {
		if b, err = pdfAPNGLogo(b, 1, 300); err != nil {
			return nil, err
		}
	}
	return imaxeOfiDeBytes(b)
}
