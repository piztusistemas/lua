package main

// Lectura con visión (coma NotebookLM): a IA ve as páxinas e transcríbeas a
// Markdown con LaTeX. É o que fai falla cando markitdown (que só extrae
// texto) perde a información:
//
//   - fórmulas e notación matemática ($\frac{a}{b}$, $\sqrt{x}$, $x^2$...),
//   - debuxos, circuítos, gráficas e figuras,
//   - documentos escaneados ou fotos (sen texto que extraer).
//
// As páxinas pásanse a PNG con pdftoppm e vanlle á IA de dúas en dúas pola
// API de Lua. As figuras que sinala a IA (cun recadro en fraccións da páxina)
// recórtanse e gárdanse en raw/imaxes/, e o Markdown enlázaas con ![[...]].
// Os circuítos, ademais, reprodúcense en circuitikz.

import (
	"bytes"
	"context"
	"edicion/idioma"
	"encoding/base64"
	"fmt"
	"image"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"edicion/plataforma"
)

// dirImaxes: as figuras recortadas (dentro de raw/: son parte da fonte).
const dirImaxes = "imaxes"

// ppVision: resolución das páxinas que ve a IA (e das que se recortan as figuras).
const ppVision = 150

// paxinasPorChamada: cantas páxinas ve a IA de cada vez.
const paxinasPorChamada = 2

// ---------- Cando fai falla ----------

// simbolosMates: caracteres que só aparecen en texto con fórmulas.
const simbolosMates = "∫∑∏√∞≈≠≤≥±×÷∂∇∆πθλμσΔΩαβγδεφω→⇒⇔∈∉⊂∪∩′″½¼¾²³⁴⁵⁶⁷⁸⁹⁰₀₁₂₃₄"

// motivoVision di por que un PDF convertido con markitdown precisa a IA
// ("" = o texto abonda). paxinas e figuras veñen de pdfinfo e pdfimages.
func motivoVision(md string, paxinas, figuras int) error {
	texto := strings.TrimSpace(md)
	if paxinas < 1 {
		paxinas = 1
	}
	if len([]rune(texto)) < 150*paxinas {
		return idioma.Errorf("motivo.pouco_texto", "ten pouco texto extraíble (escaneado ou feito de imaxes)")
	}
	if strings.Contains(texto, "(cid:") {
		return idioma.Errorf("motivo.ilexible", "o texto sae con caracteres ilexibles")
	}
	mates := 0
	for _, r := range texto {
		if strings.ContainsRune(simbolosMates, r) {
			mates++
		}
	}
	if mates >= 5 {
		return idioma.Errorf("motivo.formulas", "ten fórmulas")
	}
	// Fraccións e expoñentes esnaquizados: moitas liñas cun só número ou letra.
	curtas, total := 0, 0
	for _, l := range strings.Split(texto, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			total++
			if len([]rune(l)) <= 3 {
				curtas++
			}
		}
	}
	if total > 20 && curtas*100/total >= 12 {
		return idioma.Errorf("motivo.formulas_rotas", "ten fórmulas (o texto sae esnaquizado)")
	}
	if figuras > 0 {
		return idioma.Errorf("motivo.figuras", "ten %d figura(s) ou debuxo(s)", figuras)
	}
	return nil
}

// paxinasPDF conta as páxinas (pdfinfo).
func paxinasPDF(pdf string) int {
	cmd := exec.Command("pdfinfo", pdf)
	plataforma.OcultarConsola(cmd)
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	for _, l := range strings.Split(string(out), "\n") {
		if v, ok := strings.CutPrefix(l, "Pages:"); ok {
			n, _ := strconv.Atoi(strings.TrimSpace(v))
			return n
		}
	}
	return 0
}

// figurasPDF conta as imaxes do PDF que poden ser figuras (pdfimages; as
// pequenas, coma logotipos e iconas, non contan).
func figurasPDF(pdf string) int {
	cmd := exec.Command("pdfimages", "-list", pdf)
	plataforma.OcultarConsola(cmd)
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	n := 0
	for i, l := range strings.Split(string(out), "\n") {
		c := strings.Fields(l)
		if i < 2 || len(c) < 5 {
			continue
		}
		w, _ := strconv.Atoi(c[3])
		h, _ := strconv.Atoi(c[4])
		if w >= 150 && h >= 150 {
			n++
		}
	}
	return n
}

// mdTenFormulasEnTexto: un Markdown con fórmulas escritas en texto plano
// (sen $…$), que paga a pena pasar a LaTeX.
func mdTenFormulasEnTexto(md string) bool {
	if strings.Count(md, "$") >= 4 {
		return false // xa ten LaTeX
	}
	mates := 0
	for _, r := range md {
		if strings.ContainsRune(simbolosMates, r) {
			mates++
		}
	}
	return mates >= 3 || regexp.MustCompile(`\b[a-zA-Z]\^\d|\bsqrt\(|\b\d+/\d+\b`).MatchString(md)
}

// ---------- Páxinas ----------

// paxinaVision: unha páxina en PNG.
type paxinaVision struct {
	Numero int
	Ruta   string
}

// renderizarPDF pasa as páxinas do PDF a PNG en dir.
func renderizarPDF(ctx context.Context, pdf, dir string) ([]paxinaVision, error) {
	if !plataforma.CommandExists("pdftoppm") {
		return nil, idioma.Errorf("erro.sen_pdftoppm", "para que a IA lea as páxinas cómpre pdftoppm (poppler)")
	}
	cmd := exec.CommandContext(ctx, "pdftoppm", "-r", strconv.Itoa(ppVision), "-png", pdf, filepath.Join(dir, "p"))
	plataforma.OcultarConsola(cmd)
	var erro bytes.Buffer
	cmd.Stderr = &erro
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("pdftoppm: %v %s", err, strings.TrimSpace(erro.String()))
	}
	rutas, _ := filepath.Glob(filepath.Join(dir, "p-*.png"))
	var out []paxinaVision
	for _, r := range rutas {
		b := strings.TrimSuffix(filepath.Base(r), ".png")
		n, _ := strconv.Atoi(b[strings.LastIndex(b, "-")+1:])
		out = append(out, paxinaVision{Numero: n, Ruta: r})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Numero < out[j].Numero })
	if len(out) == 0 {
		return nil, idioma.Errorf("erro.pdf_sen_paxinas", "o PDF non ten páxinas")
	}
	return out, nil
}

func dataURI(ruta string) (string, error) {
	d, err := os.ReadFile(ruta)
	if err != nil {
		return "", err
	}
	mime := "image/png"
	switch strings.ToLower(filepath.Ext(ruta)) {
	case ".jpg", ".jpeg":
		mime = "image/jpeg"
	case ".webp":
		mime = "image/webp"
	case ".gif":
		mime = "image/gif"
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(d), nil
}

// ---------- A IA ----------

const sistemaVision = `Es o transcritor de documentos de Buxo. Recibes páxinas dun documento (un exame,
apuntamentos, un boletín de exercicios...) coma imaxes e escribes o seu contido en
Markdown FIEL ao orixinal, para unha bóveda de Obsidian. Non resumas, non resolvas,
non corrixas, non engadas nada que non estea na páxina.

FORMATO
- Texto en Markdown: títulos con #, listas, **negriña**, táboas Markdown.
- TODA a notación matemática, física ou química en LaTeX: $…$ en liña e $$…$$ para
  as fórmulas destacadas. Fraccións \frac{}{}, raíces \sqrt{}, vectores \vec{},
  unidades con \,\mathrm{m/s}, símbolos gregos \alpha... Fórmulas químicas en LaTeX
  ($\mathrm{H_2SO_4}$). Non deixes ningunha fórmula en texto plano.
- Exercicios e preguntas: conserva a numeración e as puntuacións («1.», «a)», «(2 puntos)»).
- FIGURAS (gráficas, debuxos, esquemas, fotos, xeometría, circuítos): onde estea, escribe
  ![descrición breve](figura:P:x0,y0,x1,y1)
  onde P é o número de páxina indicado e x0,y0,x1,y1 o recadro da figura en fraccións do
  ancho e do alto da páxina (0 a 1, desde a esquina de arriba á esquerda), un pouco
  folgado. Debaixo, unha liña en cursiva co que se ve (eixes, valores, etiquetas) se é
  información útil para resolver o exercicio.
- CIRCUÍTOS ELÉCTRICOS: ademais da figura, reprodúceo nun bloque de código circuitikz
  (sen \begin{circuitikz}), coas mesmas etiquetas:
  ` + "```circuitikz" + `
  \draw (0,0) to[battery1, l=$12\,\mathrm{V}$] (0,3) to[R, l=$R_1$] (3,3) -- (3,0) -- (0,0);
  ` + "```" + `
  (compoñentes: R, battery1, V, lamp, ammeter, voltmeter, switch, C, L, D; etiquetas l=$…$).
- Cabeceiras e pés repetidos en todas as páxinas (logotipo, nome do centro, número de
  páxina) non se copian.
- Antes do contido de cada páxina escribe <!-- páxina P -->.
- Se algo non se le, escribe [ilexible] no seu sitio; non o inventes.

Responde SÓ co Markdown, sen comentarios nin bloque de código arredor.`

const sistemaNormalizar = `Es o editor de Buxo. Recibes un anaco dun documento Markdown que ten fórmulas
escritas en texto plano (x^2, sqrt(x), 3/4, símbolos Unicode...). Devolve o MESMO
documento, cos mesmos títulos, listas, texto e orde, pero coa notación matemática en
LaTeX ($…$ en liña, $$…$$ as fórmulas soltas). Non cambies nada máis: nin palabras, nin
números, nin a estrutura. Non resumas nin resolvas. Responde SÓ co Markdown.`

// figuraRe: ![descrición](figura:P:x0,y0,x1,y1)
var figuraRe = regexp.MustCompile(`!\[([^\]]*)\]\(figura:(\d+):\s*([\d.]+)\s*,\s*([\d.]+)\s*,\s*([\d.]+)\s*,\s*([\d.]+)\s*\)`)

// transcribirPaxinas pide á IA o Markdown das páxinas (de dúas en dúas) e
// recorta as figuras. base é o nome do documento (para as imaxes) e
// progreso recibe «páxinas a-b de n».
func (m *Mestre) transcribirPaxinas(ctx context.Context, ia *ClienteIA, paxinas []paxinaVision, base string, progreso func(string)) (string, error) {
	var md strings.Builder
	porNumero := map[int]paxinaVision{}
	for _, p := range paxinas {
		porNumero[p.Numero] = p
	}
	nFig := 0
	for i := 0; i < len(paxinas); i += paxinasPorChamada {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		lote := paxinas[i:min(i+paxinasPorChamada, len(paxinas))]
		var nums []string
		var imaxes []string
		for _, p := range lote {
			uri, err := dataURI(p.Ruta)
			if err != nil {
				return "", err
			}
			imaxes = append(imaxes, uri)
			nums = append(nums, strconv.Itoa(p.Numero))
		}
		desc := fmt.Sprintf("páxina %s de %d", nums[0], len(paxinas))
		// descVisible é o mesmo no idioma de Lua (panel e erros); desc, en
		// galego, vai na petición á IA.
		descVisible := idioma.Errorf("desc.paxina", "páxina %s de %d", nums[0], len(paxinas))
		if len(nums) > 1 {
			desc = fmt.Sprintf("páxinas %s-%s de %d", nums[0], nums[len(nums)-1], len(paxinas))
			descVisible = idioma.Errorf("desc.paxinas", "páxinas %s-%s de %d", nums[0], nums[len(nums)-1], len(paxinas))
		}
		progreso(idioma.T("detalle.ia_le", "a IA le as %s", descVisible))
		pedido := "Transcribe " + desc + ". As imaxes van na orde das páxinas " + strings.Join(nums, ", ") + "."
		t, err := ia.turnoComPulso(ctx, sistemaVision, []IAMensaxe{{Rol: "user", Texto: pedido, Imaxes: imaxes}}, nil, nil)
		if err != nil {
			return "", idioma.Errorf("desc.erro", "%s: %w", descVisible, err)
		}
		if t.Truncado {
			m.log.Aviso("rex.tokens", "%s: a resposta da IA cortouse polo límite de tokens (auméntao nas Opcións de Lua)", descVisible)
		}
		texto := limparValado(t.Texto)
		texto = figuraRe.ReplaceAllStringFunc(texto, func(f string) string {
			g := figuraRe.FindStringSubmatch(f)
			n, _ := strconv.Atoi(g[2])
			p, ok := porNumero[n]
			if !ok && len(lote) == 1 {
				p, ok = lote[0], true
			}
			if !ok {
				return "*[Figura: " + g[1] + "]*"
			}
			caixa := [4]float64{}
			for k := range caixa {
				caixa[k], _ = strconv.ParseFloat(g[3+k], 64)
			}
			nFig++
			nome := fmt.Sprintf("%s-p%d-%d.png", base, p.Numero, nFig)
			if err := m.recortarFigura(p.Ruta, caixa, nome); err != nil {
				m.log.Aviso("rex.figura", "%s: non se puido recortar unha figura (%v): vai a páxina enteira", descVisible, err)
				nome = fmt.Sprintf("%s-p%d.png", base, p.Numero)
				if err := copiarImaxe(p.Ruta, filepath.Join(m.raiz, dirRaw, dirImaxes, nome)); err != nil {
					return "*[Figura: " + g[1] + "]*"
				}
			}
			return "![[" + nome + "]]"
		})
		md.WriteString(strings.TrimSpace(texto) + "\n\n")
	}
	return md.String(), nil
}

// limparValado quita un ```markdown … ``` que a IA poña arredor de todo.
func limparValado(s string) string {
	t := strings.TrimSpace(s)
	if strings.HasPrefix(t, "```") && strings.HasSuffix(t, "```") {
		if i := strings.IndexByte(t, '\n'); i > 0 {
			primeira := strings.TrimSpace(t[3:i])
			if primeira == "" || primeira == "markdown" || primeira == "md" {
				return strings.TrimSpace(strings.TrimSuffix(t[i+1:], "```"))
			}
		}
	}
	return t
}

// recortarFigura garda en raw/imaxes/nome o recadro (fraccións 0-1) da páxina,
// cun pouco de marxe.
func (m *Mestre) recortarFigura(ruta string, c [4]float64, nome string) error {
	f, err := os.Open(ruta)
	if err != nil {
		return err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return err
	}
	b := img.Bounds()
	marxe := 0.015
	x0, y0 := max(c[0]-marxe, 0), max(c[1]-marxe, 0)
	x1, y1 := min(c[2]+marxe, 1), min(c[3]+marxe, 1)
	if x1-x0 < 0.03 || y1-y0 < 0.02 {
		return fmt.Errorf("recadro baleiro")
	}
	r := image.Rect(b.Min.X+int(x0*float64(b.Dx())), b.Min.Y+int(y0*float64(b.Dy())), b.Min.X+int(x1*float64(b.Dx())), b.Min.Y+int(y1*float64(b.Dy())))
	out := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	draw.Draw(out, out.Bounds(), img, r.Min, draw.Src)
	dir := filepath.Join(m.raiz, dirRaw, dirImaxes)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, out); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, nome), buf.Bytes(), 0o644)
}

func copiarImaxe(orixe, destino string) error {
	d, err := os.ReadFile(orixe)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destino), 0o755); err != nil {
		return err
	}
	return os.WriteFile(destino, d, 0o644)
}

// ---------- Conversións coa IA ----------

// converterConVision transcribe un PDF ou unha imaxe e escribe raw/<nome>.md.
func (m *Mestre) converterConVision(ctx context.Context, it *Item, motivo error) error {
	ia := m.iaVision()
	if ia == nil {
		return idioma.Errorf("erro.precisa_ia", "para ler fórmulas e debuxos cómpre a IA (abre Buxo desde Lua cunha IA configurada)")
	}
	orixe := filepath.Join(m.raiz, dirPDF, it.Nome)
	base := strings.TrimSuffix(it.Nome, filepath.Ext(it.Nome))
	tmp, err := os.MkdirTemp("", "buxo-vision-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	var paxinas []paxinaVision
	if tipoDe(it.Nome) == tipoImaxe {
		paxinas = []paxinaVision{{Numero: 1, Ruta: orixe}}
	} else if paxinas, err = renderizarPDF(ctx, orixe, tmp); err != nil {
		return err
	}
	m.log.Info("rex.ia_le", "%s: %s; a IA le %d páxina(s)…", it.Nome, motivo, len(paxinas))
	md, err := m.transcribirPaxinas(ctx, ia, paxinas, base, func(s string) {
		m.poñer(it, func(i *Item) { i.Detalle = s })
	})
	if err != nil {
		return err
	}
	return m.escribirRaw(it, md, "ia (visión)")
}

// normalizarMarkdown pasa a LaTeX as fórmulas en texto plano dun .md (por
// anacos, para non pasar o límite da resposta).
func (m *Mestre) normalizarMarkdown(ctx context.Context, it *Item, md string) (string, error) {
	ia := m.iaVision()
	if ia == nil {
		return md, fmt.Errorf("sen IA")
	}
	var out strings.Builder
	anacos := partirMarkdown(md, 9000)
	for i, a := range anacos {
		m.poñer(it, func(x *Item) {
			x.Detalle = idioma.T("detalle.formulas_latex", "a IA pasa as fórmulas a LaTeX (%d/%d)", i+1, len(anacos))
		})
		t, err := ia.turnoComPulso(ctx, sistemaNormalizar, []IAMensaxe{{Rol: "user", Texto: a}}, nil, nil)
		if err != nil {
			return md, err
		}
		out.WriteString(strings.TrimSpace(limparValado(t.Texto)) + "\n\n")
	}
	return out.String(), nil
}

// partirMarkdown parte nos parágrafos para que cada anaco non pase de n bytes.
func partirMarkdown(md string, n int) []string {
	var out []string
	var b strings.Builder
	for _, p := range strings.Split(md, "\n\n") {
		if b.Len() > 0 && b.Len()+len(p) > n {
			out = append(out, b.String())
			b.Reset()
		}
		b.WriteString(p + "\n\n")
	}
	if strings.TrimSpace(b.String()) != "" {
		out = append(out, b.String())
	}
	return out
}

// escribirRaw garda o Markdown en raw/ coa cabeceira da fonte.
func (m *Mestre) escribirRaw(it *Item, md, conversion string) error {
	if strings.TrimSpace(md) == "" {
		return fmt.Errorf("non saíu ningún texto")
	}
	url := ""
	if it.Orixe == "url" {
		url = it.Fonte
	}
	dst := filepath.Join(m.raiz, dirRaw, nomeMD(it.Nome))
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, []byte(cabeceiraFonteCon(md, it.Nome, url, conversion)), 0o644)
}
