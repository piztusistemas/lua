package lectura

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"edicion/ia"
)

// PaxinaLida: o resultado de ler unha páxina.
type PaxinaLida struct {
	Num    int    `json:"num"`
	Metodo string `json:"metodo"`
	Motivo string `json:"motivo"`
	// Markdown: a transcrición (Markdown + $LaTeX$). As figuras van coma
	// ![descrición](raw/imaxes/…png).
	Markdown string `json:"markdown"`
	// TextoCapa: o que di a capa de texto do PDF (para comparar).
	TextoCapa string `json:"-"`
	// Dubidas: palabras e números da capa de texto que non aparecen na
	// transcrición da IA (pode que os cambiase ou que os inventase).
	Dubidas []string `json:"dubidas"`
	// Cobertura: fracción da capa de texto que está na transcrición (0-1);
	// -1 se a páxina non ten capa de texto.
	Cobertura float64  `json:"cobertura"`
	Figuras   []Figura `json:"figuras"`
	// Imaxe: a páxina en PNG (data URI) para a revisión lado a lado.
	Imaxe string `json:"imaxe"`
	Erro  string `json:"erro"`
}

// Figura recortada dunha páxina.
type Figura struct {
	Ficheiro    string `json:"ficheiro"` // nome do PNG (en raw/imaxes/)
	Descricion  string `json:"descricion"`
	Tipo        string `json:"tipo"`        // plot, tikz, chemfig ou ""
	Reconstruir string `json:"reconstruir"` // código proposto (opcional)
	ruta        string
}

// Opcions da lectura.
type Opcions struct {
	IA      ia.Config
	Dir     string // onde deixar as imaxes (as figuras van a Dir/imaxes)
	Prefixo string // para os nomes das figuras (o slug do PDF)
	Aviso   func(string)
	Avance  func(feito, total int)
}

const ppVision = 150 // resolución da imaxe que ve a IA

// Ler le todas as páxinas co método decidido en Clasificar.
func Ler(ctx context.Context, pdf string, pax []Paxina, o Opcions) ([]PaxinaLida, error) {
	if err := os.MkdirAll(filepath.Join(o.Dir, "imaxes"), 0o755); err != nil {
		return nil, err
	}
	aviso := o.Aviso
	if aviso == nil {
		aviso = func(string) {}
	}
	var out []PaxinaLida
	for i, p := range pax {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		if o.Avance != nil {
			o.Avance(i, len(pax))
		}
		pl := PaxinaLida{Num: p.Num, Metodo: p.Metodo, Motivo: p.Motivo, Cobertura: -1}
		num := strconv.Itoa(p.Num)
		capa, _ := executar("pdftotext", "-f", num, "-l", num, pdf, "-")
		pl.TextoCapa = strings.TrimSpace(capa)
		png, err := renderizar(pdf, p.Num, ppVision, filepath.Join(o.Dir, "paxina-"+num))
		if err == nil {
			datos, _ := os.ReadFile(png)
			pl.Imaxe = "data:image/png;base64," + base64.StdEncoding.EncodeToString(datos)
		}
		if p.Metodo == MetodoVision && !o.IA.Configurada() {
			aviso(fmt.Sprintf("páxina %d: %s, pero non hai IA configurada; úsase a capa de texto", p.Num, p.Motivo))
			pl.Metodo = MetodoTexto
		}
		if pl.Metodo == MetodoTexto || err != nil {
			pl.Markdown = textoAMarkdown(pl.TextoCapa)
			if p.Metodo == MetodoVision && err != nil {
				pl.Erro = "non se puido renderizar a páxina: " + err.Error()
			}
			out = append(out, pl)
			continue
		}
		aviso(fmt.Sprintf("páxina %d: lendo con visión (%s)…", p.Num, p.Motivo))
		lida, err := lerVision(o.IA, png, pl.TextoCapa)
		if err != nil {
			pl.Erro = err.Error()
			pl.Markdown = textoAMarkdown(pl.TextoCapa)
			pl.Metodo = MetodoTexto
			aviso(fmt.Sprintf("páxina %d: a IA fallou (%v); úsase a capa de texto", p.Num, err))
			out = append(out, pl)
			continue
		}
		pl.Markdown = lida.Markdown
		for j, f := range lida.Figuras {
			nome := fmt.Sprintf("%s-p%d-f%d.png", o.Prefixo, p.Num, j+1)
			ruta := filepath.Join(o.Dir, "imaxes", nome)
			if err := recortar(png, f.Caixa, ruta); err != nil {
				aviso(fmt.Sprintf("páxina %d: non se puido recortar a figura %d: %v", p.Num, j+1, err))
				continue
			}
			pl.Figuras = append(pl.Figuras, Figura{Ficheiro: nome, Descricion: f.Descricion, Tipo: f.Tipo, Reconstruir: f.Codigo, ruta: ruta})
			marca := fmt.Sprintf("[[FIGURA-%d]]", j+1)
			img := fmt.Sprintf("![%s](raw/imaxes/%s)", limparAlt(f.Descricion), nome)
			if strings.TrimSpace(f.Codigo) != "" && (f.Tipo == "plot" || f.Tipo == "tikz" || f.Tipo == "chemfig") {
				// A proposta vai nun comentario: vese en Obsidian pero non no
				// PDF. Para usala, cámbiase a imaxe por un bloque ```tipo.
				img += fmt.Sprintf("\n<!-- LuaYang: proposta de reconstrución (%s)\n%s\n-->", f.Tipo, strings.ReplaceAll(strings.TrimSpace(f.Codigo), "-->", "- ->"))
			}
			if strings.Contains(pl.Markdown, marca) {
				pl.Markdown = strings.ReplaceAll(pl.Markdown, marca, img)
			} else {
				pl.Markdown += "\n\n" + img + "\n"
			}
		}
		pl.Markdown = regexp.MustCompile(`\[\[FIGURA-\d+\]\]`).ReplaceAllString(pl.Markdown, "")
		if pl.TextoCapa != "" {
			pl.Dubidas, pl.Cobertura = Comparar(pl.TextoCapa, pl.Markdown)
		}
		out = append(out, pl)
	}
	if o.Avance != nil {
		o.Avance(len(pax), len(pax))
	}
	return out, nil
}

func limparAlt(s string) string {
	return strings.NewReplacer("[", "(", "]", ")", "\n", " ").Replace(strings.TrimSpace(s))
}

// renderizar unha páxina a PNG; devolve a ruta.
func renderizar(pdf string, num, ppp int, prefixo string) (string, error) {
	n := strconv.Itoa(num)
	if _, err := executar("pdftoppm", "-png", "-r", strconv.Itoa(ppp), "-f", n, "-l", n, "-singlefile", pdf, prefixo); err != nil {
		return "", err
	}
	return prefixo + ".png", nil
}

// recortar unha caixa (fraccións 0-1 da páxina: x0, y0, x1, y1) do PNG.
func recortar(png string, caixa [4]float64, dest string) error {
	f, err := os.Open(png)
	if err != nil {
		return err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return err
	}
	b := img.Bounds()
	marxe := 0.01
	x0 := clamp(caixa[0]-marxe) * float64(b.Dx())
	y0 := clamp(caixa[1]-marxe) * float64(b.Dy())
	x1 := clamp(caixa[2]+marxe) * float64(b.Dx())
	y1 := clamp(caixa[3]+marxe) * float64(b.Dy())
	if x1-x0 < 8 || y1-y0 < 8 {
		return fmt.Errorf("caixa baleira %v", caixa)
	}
	sub, ok := img.(interface {
		SubImage(r image.Rectangle) image.Image
	})
	if !ok {
		return fmt.Errorf("non se pode recortar este tipo de imaxe")
	}
	r := image.Rect(int(x0), int(y0), int(x1), int(y1)).Add(b.Min)
	return gardarPNG(sub.SubImage(r), dest)
}

func clamp(v float64) float64 {
	switch {
	case v < 0:
		return 0
	case v > 1:
		return 1
	}
	return v
}

// ---------- visión ----------

const sistemaVision = `Es un transcritor de exames e materiais docentes. Recibes a imaxe dunha páxina e tes que devolver o seu contido en Markdown con fórmulas LaTeX.

REGRAS (importantes):
- Transcribe LITERALMENTE. Non corrixas ortografía, gramática, puntuación nin palabras, aínda que estean mal ou sexan dialectais. Non traduzas. Non resumas. Non engadas nada que non estea na páxina.
- Fórmulas en LaTeX: $...$ en liña e $$...$$ á parte. Sistemas con \left\{\begin{array}...\end{array}\right. ou cases; matrices con pmatrix; límites con \lim_{x\to a}.
- Química: \ce{...} (mhchem) para fórmulas e reaccións (\ce{H2O}, \ce{CO2 + H2 <=> CO + H2O}).
- Títulos de pregunta en negra (**PREGUNTA 1. ...**), listas e apartados como no orixinal.
- Táboas en Markdown. Ocos para encher: «______».
- Ignora cabeceiras e pés que se repiten (logotipos, nome da institución, número de páxina).
- Cada figura, gráfico, debuxo ou esquema: pon no texto, no sitio onde vai, a marca [[FIGURA-n]] (n = 1, 2...) e descríbea en "figuras" coa súa caixa en fraccións da páxina (x0, y0 arriba á esquerda; x1, y1 abaixo á dereita). Se se pode reconstruír, propón o código: "plot" (orde plot2d de Maxima), "tikz" (só o corpo do tikzpicture) ou "chemfig" (o argumento de \chemfig). Senón, tipo "".

Responde SÓ cun obxecto JSON:
{"markdown": "...", "figuras": [{"descricion": "...", "caixa": [x0, y0, x1, y1], "tipo": "plot|tikz|chemfig|", "codigo": "..."}]}`

type respostaVision struct {
	Markdown string `json:"markdown"`
	Figuras  []struct {
		Descricion string     `json:"descricion"`
		Caixa      [4]float64 `json:"caixa"`
		Tipo       string     `json:"tipo"`
		Codigo     string     `json:"codigo"`
	} `json:"figuras"`
}

func lerVision(c ia.Config, png, capa string) (respostaVision, error) {
	datos, err := os.ReadFile(png)
	if err != nil {
		return respostaVision{}, err
	}
	pedido := "Transcribe esta páxina."
	if capa != "" {
		// A capa de texto axuda a non confundir letras, pero a orde e as
		// fórmulas mándaas a imaxe.
		pedido += "\n\nPara axudarte, este é o texto que trae o PDF (pode ter a orde mal e as fórmulas rotas; a referencia é a imaxe):\n<capa>\n" + recortarTexto(capa, 6000) + "\n</capa>"
	}
	var r respostaVision
	var ultimo error
	for intento := 0; intento < 2; intento++ {
		txt, err := ia.Preguntar(c, sistemaVision, pedido, ia.ImaxeDeBytes("image/png", datos))
		if err != nil {
			ultimo = err
			continue
		}
		if err := json.Unmarshal([]byte(ia.ExtraerJSON(txt)), &r); err != nil {
			ultimo = fmt.Errorf("a IA non devolveu JSON válido: %v", err)
			continue
		}
		if strings.TrimSpace(r.Markdown) == "" {
			ultimo = fmt.Errorf("a IA devolveu unha transcrición baleira")
			continue
		}
		return r, nil
	}
	return r, ultimo
}

func recortarTexto(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}

// textoAMarkdown: a capa de texto en Markdown, conservando a estrutura que
// se pode deducir sen IA: quítanse as columnas de números de liña, os títulos
// de pregunta van en parágrafo propio e en negra, e as liñas curtas (fin de
// parágrafo, títulos, versos) conservan o salto.
func textoAMarkdown(t string) string {
	t = strings.ReplaceAll(t, "\f", "\n")
	liñas := strings.Split(t, "\n")
	for i := range liñas {
		liñas[i] = strings.TrimRight(liñas[i], " \t")
	}
	liñas = quitarNumeracion(liñas)
	maxLen := 0
	for _, l := range liñas {
		maxLen = max(maxLen, len([]rune(strings.TrimSpace(l))))
	}
	esc := strings.NewReplacer("*", `\*`, "_", `\_`, "#", `\#`, "$", `\$`, "{", "{{", "}", "}}")
	var b strings.Builder
	parágrafo := false
	for i, l := range liñas {
		l = strings.TrimSpace(l)
		if l == "" {
			if parágrafo {
				b.WriteString("\n\n")
				parágrafo = false
			}
			continue
		}
		if tituloPreguntaRe.MatchString(l) || (l == strings.ToUpper(l) && len([]rune(l)) >= 3 && strings.ContainsFunc(l, unicode.IsLetter)) {
			if parágrafo {
				b.WriteString("\n\n")
			}
			b.WriteString("**" + esc.Replace(l) + "**\n\n")
			parágrafo = false
			continue
		}
		b.WriteString(esc.Replace(l))
		parágrafo = true
		seguinte := ""
		if i+1 < len(liñas) {
			seguinte = strings.TrimSpace(liñas[i+1])
		}
		switch {
		case seguinte == "":
		case len([]rune(l)) < maxLen*65/100 || apartadoRe.MatchString(seguinte):
			b.WriteString("\\\n") // salto de liña duro (pandoc)
		default:
			b.WriteString("\n")
		}
	}
	out := strings.TrimSpace(b.String())
	// Un salto duro xusto antes dun parágrafo novo sobra (pandoc pintaría «\\»).
	out = regexp.MustCompile(`\\\n(\s*\n|\*\*)`).ReplaceAllString(out, "\n$1")
	return strings.TrimSuffix(out, `\`)
}

var apartadoRe = regexp.MustCompile(`^(\d+(\.\d+)*\.|[a-z]\)|[-–•])\s`)

// quitarNumeracion quita as tiradas de 5 ou máis liñas que son só números
// consecutivos (a columna de números de liña dun texto).
func quitarNumeracion(liñas []string) []string {
	var out []string
	for i := 0; i < len(liñas); {
		j := i
		esperado := -1
		for j < len(liñas) {
			n, err := strconv.Atoi(strings.TrimSpace(liñas[j]))
			if err != nil || (esperado >= 0 && n != esperado) {
				break
			}
			esperado = n + 1
			j++
		}
		if j-i >= 5 {
			i = j
			continue
		}
		out = append(out, liñas[i])
		i++
	}
	return out
}

// ---------- dobre lectura ----------

var (
	comandoTexRe = regexp.MustCompile(`\\[a-zA-Z]+`)
	palabraRe    = regexp.MustCompile(`[\p{L}]{4,}|\d+(?:[.,]\d+)*`)
)

func normalizar(s string) string {
	s = strings.Map(func(r rune) rune {
		switch r {
		case '𝑎', '𝐴':
			return 'a'
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return ' '
	}, s)
	return s
}

// Comparar: que palabras (4+ letras) e números da capa de texto non están na
// transcrición. Devolve as dúbidas (sen repetir) e a cobertura.
func Comparar(capa, transcricion string) ([]string, float64) {
	t := comandoTexRe.ReplaceAllString(transcricion, " ")
	t = normalizar(t)
	tSenEspazos := strings.ReplaceAll(t, " ", "")
	conxunto := map[string]bool{}
	for _, w := range strings.Fields(t) {
		conxunto[w] = true
	}
	total, atopadas := 0, 0
	var dubidas []string
	vistas := map[string]bool{}
	for _, w := range palabraRe.FindAllString(capa, -1) {
		n := strings.TrimSpace(normalizar(w))
		if n == "" {
			continue
		}
		// As letras matemáticas de Unicode (𝑥, 𝑃…) non son palabras.
		if strings.ContainsFunc(w, func(r rune) bool { return r >= 0x1D400 && r <= 0x1D7FF }) {
			continue
		}
		total++
		partes := strings.Fields(n)
		ok := true
		for _, p := range partes {
			if !conxunto[p] && !strings.Contains(tSenEspazos, p) {
				ok = false
			}
		}
		if ok {
			atopadas++
		} else if !vistas[w] {
			vistas[w] = true
			dubidas = append(dubidas, w)
		}
	}
	if total == 0 {
		return nil, -1
	}
	return dubidas, float64(atopadas) / float64(total)
}
