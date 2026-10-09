package main

// Gráficos dos formularios: o nodo de destino «Gráfico» da tubería e as súas
// saídas. O debuxo vive no paquete grafico (escena única → TikZ para o PDF,
// PNG para Word, OpenDocument e Markdown); aquí só se conectan os datos do
// fluxo co motor e se escribe o resultado en cada formato.
//
// O nodo acepta dúas formas de datos, nunca as dúas á vez:
//   - unha TÁBOA: unha columna de categorías e unha ou varias de valores (cada
//     columna de valores é unha serie);
//   - CANTIDADES soltas (portos v1, v2...): cada unha é unha categoría, co
//     rótulo do porto (ou o do campo de orixe), coma na ficha.

import (
	"encoding/base64"
	"fmt"
	"sort"
	"strings"
	"sync"

	"lua-wails/grafico"
)

const (
	nodoGrafico   = "grafico"
	bloqueGrafico = "grafico"
)

// maxCantidadesGrafico é o número máximo de portos de cantidades soltas.
const maxCantidadesGrafico = 12

// catalogoTiposGrafico son os tipos de gráfico que se ofrecen (ids e nomes).
func catalogoTiposGrafico() (ids []string, nomes map[string]string) {
	nomes = map[string]string{}
	for _, t := range grafico.Tipos() {
		ids = append(ids, t.ID)
		nomes[t.ID] = t.Nome
	}
	return ids, nomes
}

// tipoNodoGrafico é a entrada do catálogo de tuberías.
func tipoNodoGrafico() TipoNodo {
	ids, nomes := catalogoTiposGrafico()
	axuda := "Un gráfico. Escolle o tipo (tarta, rosca, barras, liñas...) e dálle datos: ou unha táboa (columna de categorías + columnas de valores) ou varias cantidades soltas, cada unha co seu rótulo. Non sae se non hai datos."
	return TipoNodo{
		Tipo: nodoGrafico, Nome: "Gráfico", Categoria: "destino", Axuda: axuda,
		PortosEtiqueta: true,
		Entradas:       []PortoNodo{{Nome: "taboa", Tipo: dTaboa, Opcional: true, Axuda: "Alternativa ás cantidades soltas."}, portoSe},
		Dinamicas:      nomesPortos("v", maxCantidadesGrafico), DinTipo: dNumero,
		Config: []ConfigNodo{
			{Clave: "tipo", Etiqueta: "Tipo de gráfico", Tipo: "escolla", Opcions: ids, Nomes: nomes, Defecto: grafico.Tarta},
			confTitulo,
			{Clave: "columna", Etiqueta: "Categorías (columna da táboa)", Tipo: "columna", Axuda: "Só se conectas unha táboa."},
			{Clave: "columnas", Etiqueta: "Valores (columnas da táboa)", Tipo: "columnas", Axuda: "Cada columna é unha serie. A tarta e a rosca usan só a primeira."},
			{Clave: "etiquetas", Etiqueta: "Etiquetas da tarta", Tipo: "escolla", Opcions: []string{"porcentaxe", "valor", "ambos"},
				Nomes:   map[string]string{"porcentaxe": "Porcentaxe", "valor": "Valor", "ambos": "Valor e porcentaxe"},
				Defecto: "porcentaxe", Axuda: "Só na tarta e na rosca."},
		},
	}
}

func tipoGraficoDe(n *NodoFluxo) string {
	if t := cfgTexto(n, "tipo"); t != "" {
		return t
	}
	return grafico.Tarta
}

// portosCantidades devolve os portos de cantidades soltas do nodo, en orde.
func portosCantidades(n *NodoFluxo) []string {
	var out []string
	for p := range n.Entradas {
		if p != "se" && p != "taboa" {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return ordePorto(out[i]) < ordePorto(out[j]) })
	return out
}

// analizarGrafico é a comprobación de configuración do nodo (sen mirar datos).
func analizarGrafico(n *NodoFluxo, ent map[string]*InfoNodo) string {
	if !grafico.ExisteTipo(tipoGraficoDe(n)) {
		return fmt.Sprintf("tipo de gráfico descoñecido: «%s»", tipoGraficoDe(n))
	}
	soltas := len(portosCantidades(n))
	_, taboa := n.Entradas["taboa"]
	switch {
	case taboa && soltas > 0:
		return "usa unha táboa ou cantidades soltas, non as dúas cousas á vez"
	case !taboa && soltas == 0:
		return "conecta unha táboa ou polo menos unha cantidade"
	case taboa:
		cols := ent["taboa"].Columnas
		if _, ok := columnaPorId(cols, cfgTexto(n, "columna")); !ok {
			return "escolle a columna das categorías"
		}
		ids := listaConfig(n, "columnas")
		if len(ids) == 0 {
			return "escolle polo menos unha columna de valores"
		}
		for _, id := range ids {
			c, ok := columnaPorId(cols, id)
			if !ok {
				return fmt.Sprintf("a columna «%s» non existe na táboa", id)
			}
			if c.Tipo != campoNumero && c.Tipo != campoTexto {
				return fmt.Sprintf("a columna «%s» non é numérica", c.Etiqueta)
			}
		}
	}
	return ""
}

// especGrafico converte os datos do fluxo na especificación do gráfico. O
// segundo valor é false se non hai nada que debuxar (o bloque simplemente non sae).
func (e *avaliador) especGrafico(n *NodoFluxo) (grafico.Spec, bool) {
	spec := grafico.Spec{
		Tipo:          tipoGraficoDe(n),
		Etiquetas:     cfgTexto(n, "etiquetas"),
		EtiquetaTotal: strings.TrimSpace(e.tx.T("FORM_GRAFICO_TOTAL")),
	}
	if orixe, ok := n.Entradas["taboa"]; ok {
		v := e.valor(orixe)
		if v.Vacio || len(v.Filas) == 0 {
			return spec, false
		}
		cat, _ := columnaPorId(v.Cols, cfgTexto(n, "columna"))
		for _, f := range v.Filas {
			s, _ := celaFormatada(cat, f[cat.ID], e.tx)
			spec.Categorias = append(spec.Categorias, s)
		}
		malas := 0
		for _, id := range listaConfig(n, "columnas") {
			col, _ := columnaPorId(v.Cols, id)
			serie := grafico.Serie{Nome: col.Etiqueta}
			for _, f := range v.Filas {
				x, ok := valorNumeroFloat(f[col.ID])
				if !ok && textoValor(f[col.ID]) != "" {
					malas++
				}
				serie.Valores = append(serie.Valores, x)
			}
			spec.Series = append(spec.Series, serie)
		}
		if malas > 0 {
			e.avisos = append(e.avisos, fmt.Sprintf("o gráfico «%s»: %d valores da táboa non son números e contan como 0", rotuloNodo(*n), malas))
		}
		return spec, len(spec.Series) > 0
	}
	serie := grafico.Serie{}
	for _, p := range portosCantidades(n) {
		v := e.valor(n.Entradas[p])
		if v.Vacio || v.Erro != "" {
			continue
		}
		etq := cfgTexto(n, "etq_"+p)
		if etq == "" {
			if o := e.a.nodos[n.Entradas[p]]; o != nil && o.Tipo == nodoCampo {
				etq = e.a.campo[o.Campo].Etiqueta
			} else {
				etq = p
			}
		}
		spec.Categorias = append(spec.Categorias, etq)
		serie.Valores = append(serie.Valores, v.Num)
	}
	if len(serie.Valores) == 0 {
		return spec, false
	}
	spec.Series = []grafico.Serie{serie}
	return spec, true
}

// destinoGrafico é o bloque que dá o nodo: a escena xa xerada. Se os datos
// non serven (valores negativos nunha tarta, todo a 0...) o bloque non sae e
// avísase.
func (e *avaliador) destinoGrafico(n *NodoFluxo, cab []bloqueDoc) []bloqueDoc {
	spec, ok := e.especGrafico(n)
	if !ok {
		return nil
	}
	esc, err := grafico.Xerar(spec, medidorGrafico())
	if err != nil {
		e.avisos = append(e.avisos, fmt.Sprintf("o gráfico «%s» non sae: %s", rotuloNodo(*n), err.Error()))
		return nil
	}
	b := bloqueDoc{Tipo: bloqueGrafico, Grafico: esc, Texto: nomeTipoGrafico(spec.Tipo)}
	for i, c := range spec.Categorias {
		vals := make([]string, len(spec.Series))
		for j, s := range spec.Series {
			vals[j], _ = valorNumero(s.Valores[i])
		}
		b.Pares = append(b.Pares, [2]string{c, strings.Join(vals, " / ")})
	}
	return append(cab, b)
}

func nomeTipoGrafico(id string) string {
	for _, t := range grafico.Tipos() {
		if t.ID == id {
			return t.Nome
		}
	}
	return id
}

// ---------- Fontes e PNG ----------

var (
	fontesGraficoUnha sync.Once
	fontesGraficoVal  grafico.Fontes
	medidorGraficoVal grafico.Medidor
)

func iniciarFontesGrafico() {
	fontesGraficoUnha.Do(func() {
		fontesGraficoVal.Regular, _ = fontesInformeFS.ReadFile("fontes_informe/OpenSans-Regular.ttf")
		fontesGraficoVal.Negrita, _ = fontesInformeFS.ReadFile("fontes_informe/OpenSans-Semibold.ttf")
		medidorGraficoVal = grafico.NovoMedidor(fontesGraficoVal)
	})
}

// fontesGrafico son as fontes (Open Sans, embebida) co que se debuxan os PNG.
func fontesGrafico() grafico.Fontes {
	iniciarFontesGrafico()
	return fontesGraficoVal
}

// medidorGrafico mide o texto cas mesmas fontes: a escena cadra co PNG.
func medidorGrafico() grafico.Medidor {
	iniciarFontesGrafico()
	return medidorGraficoVal
}

// pngGrafico rasteriza o gráfico a 300 ppp (Word, OpenDocument, Markdown).
func pngGrafico(bl bloqueDoc) ([]byte, error) {
	if bl.Grafico == nil {
		return nil, fmt.Errorf("gráfico baleiro")
	}
	return bl.Grafico.PNG(300, fontesGrafico())
}

// ---------- Escritores ----------

// latexGrafico escribe o gráfico coma TikZ (vectorial). O texto compóno
// LaTeX co tipo de letra da plantilla (\fontchip).
func latexGrafico(bl bloqueDoc) string {
	if bl.Grafico == nil {
		return ""
	}
	tikz := bl.Grafico.TikZ(grafico.OpcionsTikZ{Escape: EscapeLatexProse, Fonte: `\fontchip`})
	return "\\par\\medskip\\begin{center}\n" + tikz + "\n\\end{center}\n"
}

// mdGrafico escribe o gráfico coma unha imaxe PNG incrustada no Markdown.
func mdGrafico(bl bloqueDoc, esc *escritorImaxesCrono) string {
	datos, err := pngGrafico(bl)
	if err != nil {
		esc.avisos = append(esc.avisos, "un gráfico non se puido debuxar: omitiuse")
		return ""
	}
	im := esc.escribir("data:image/png;base64,"+base64.StdEncoding.EncodeToString(datos), bl.Texto)
	if im == nil {
		return ""
	}
	return "![" + mdEscaparCrono(bl.Grafico.Descricion) + "](" + im.Ficheiro + "){width=" + mmLatexCrono(bl.Grafico.Ancho) + "}\n\n"
}

// imaxeGraficoOfi é o PNG do gráfico listo para Word/OpenDocument, co seu
// tamaño en mm.
func imaxeGraficoOfi(bl bloqueDoc, d *datosOfimatica) (*imaxeOfi, float64, float64, bool) {
	datos, err := pngGrafico(bl)
	if err == nil {
		var im *imaxeOfi
		if im, err = imaxeOfiDeBytes(datos); err == nil && im.W > 0 && im.H > 0 {
			return im, bl.Grafico.Ancho, bl.Grafico.Alto, true
		}
	}
	d.Avisos = engadirSenRepetir(d.Avisos, []string{"un gráfico do formulario non se puido debuxar: omitiuse"})
	return nil, 0, 0, false
}

func (c *construtorDocx) graficoFormulario(d *datosOfimatica, bl bloqueDoc) {
	im, w, h, ok := imaxeGraficoOfi(bl, d)
	if !ok {
		return
	}
	cx, cy := int(w*docxEmuPorMM), int(h*docxEmuPorMM)
	if rel, ok := c.engadirImaxe(im, cx, cy); ok {
		c.corpo.WriteString(paraImaxeDocx(rel, cx, cy, cronoAlinDefecto))
	}
}

func (c *construtorOdt) graficoFormulario(d *datosOfimatica, bl bloqueDoc) {
	im, w, h, ok := imaxeGraficoOfi(bl, d)
	if !ok {
		return
	}
	if o, ok := c.engadirImaxe(im, w/10, h/10); ok {
		c.pImaxe(o, "Figure")
	}
}
