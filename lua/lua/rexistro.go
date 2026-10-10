package main

// Rexistro interno: o que pasou nas últimas compilacións (o log de LaTeX, o
// .tex xerado, os erros e avisos), gardado na memoria do programa para que
// o asistente de IA o poida LER e arranxar a causa. Antes só o vía a persoa
// no panel de rexistro; a IA recibía «non se puido xerar» e pouco máis.
//
// Privacidade: unha compilación dun formulario en PRODUCIÓN (datos reais)
// anótase, pero o seu log e o seu .tex non se lle amosan á IA.

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

// maxEntradasRexistro: só as últimas (cada log pode ter centos de KB).
const maxEntradasRexistro = 30

// EntradaRexistro é unha compilación (ou un debuxo que non compilou).
type EntradaRexistro struct {
	ID        int       `json:"id"`
	Cando     time.Time `json:"cando"`
	Tipo      string    `json:"tipo"`      // "compilacion" ou "debuxo"
	Documento string    `json:"documento"` // cronograma, actas, o nome do formulario...
	Motor     string    `json:"motor,omitempty"`
	OK        bool      `json:"ok"`
	Erro      string    `json:"erro,omitempty"`
	Avisos    []string  `json:"avisos,omitempty"`
	Log       string    `json:"-"`
	Tex       string    `json:"-"`
	Privado   bool      `json:"privado,omitempty"`
}

var rexistroInterno struct {
	sync.Mutex
	seq      int
	entradas []EntradaRexistro
}

// rexistrar anota unha entrada e devolve o seu id.
func rexistrar(e EntradaRexistro) int {
	rexistroInterno.Lock()
	defer rexistroInterno.Unlock()
	rexistroInterno.seq++
	e.ID = rexistroInterno.seq
	if e.Cando.IsZero() {
		e.Cando = time.Now()
	}
	rexistroInterno.entradas = append(rexistroInterno.entradas, e)
	if n := len(rexistroInterno.entradas); n > maxEntradasRexistro {
		rexistroInterno.entradas = rexistroInterno.entradas[n-maxEntradasRexistro:]
	}
	return e.ID
}

func entradaRexistro(id int) (EntradaRexistro, bool) {
	rexistroInterno.Lock()
	defer rexistroInterno.Unlock()
	for i := len(rexistroInterno.entradas) - 1; i >= 0; i-- {
		if e := rexistroInterno.entradas[i]; id == 0 || e.ID == id {
			return e, true
		}
	}
	return EntradaRexistro{}, false
}

// documentoDoPedido: como se chama o documento dunha petición e se é privado.
func documentoDoPedido(req XerarInformeCronogramaRequest) (string, bool) {
	switch {
	case req.eFormulario():
		f := req.Formulario.Formulario
		return "formulario «" + strings.TrimSpace(f.Nome) + "»", f.enProduccion()
	case req.eActas():
		return "actas", false
	}
	return "cronograma", false
}

// rexistrarCompilacion anota o resultado dunha compilación do documento.
func rexistrarCompilacion(req XerarInformeCronogramaRequest, res resultadoInformeCrono, err error) {
	doc, privado := documentoDoPedido(req)
	e := EntradaRexistro{Tipo: "compilacion", Documento: doc, Motor: res.Motor, OK: err == nil, Avisos: res.Avisos, Log: res.Log, Tex: res.Tex, Privado: privado}
	if err != nil {
		e.Erro = err.Error()
	}
	rexistrar(e)
}

// ---------- Lectura para a IA ----------

var liñaErroLatexRe = regexp.MustCompile(`(?m)^l\.(\d+)`)

// resumoEntrada: o que ve a IA na lista.
func resumoEntrada(e EntradaRexistro) map[string]any {
	out := map[string]any{
		"id": e.ID, "cando": e.Cando.Format("15:04:05"), "tipo": e.Tipo, "documento": e.Documento, "ok": e.OK,
	}
	if e.Motor != "" {
		out["motor"] = e.Motor
	}
	if e.Privado {
		out["privado"] = "formulario en produción (datos reais): o log e o .tex non se amosan"
		return out
	}
	if e.Erro != "" {
		out["erro"] = recortar(e.Erro, 600)
	}
	if len(e.Avisos) > 0 {
		out["avisos"] = e.Avisos
	}
	out["liñas_log"] = strings.Count(e.Log, "\n") + 1
	out["liñas_tex"] = strings.Count(e.Tex, "\n") + 1
	return out
}

func recortar(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}

// tramoNumerado devolve as liñas [desde, desde+cantidade) numeradas desde 1.
func tramoNumerado(texto string, desde, cantidade int) (string, int) {
	liñas := strings.Split(texto, "\n")
	desde = max(desde, 1)
	if cantidade <= 0 {
		cantidade = 60
	}
	cantidade = min(cantidade, 400)
	var b strings.Builder
	for i := desde - 1; i < len(liñas) && i < desde-1+cantidade; i++ {
		fmt.Fprintf(&b, "%5d| %s\n", i+1, liñas[i])
	}
	return b.String(), len(liñas)
}

// erroDoLog: o anaco do log co primeiro erro de LaTeX («! …») e a liña do
// .tex onde foi (l.NN).
func erroDoLog(log string) (anaco string, liñaTex int) {
	liñas := strings.Split(log, "\n")
	for i, l := range liñas {
		if strings.HasPrefix(l, "!") {
			fin := min(i+14, len(liñas))
			anaco = strings.Join(liñas[max(i-3, 0):fin], "\n")
			if m := liñaErroLatexRe.FindStringSubmatch(anaco); m != nil {
				fmt.Sscan(m[1], &liñaTex)
			}
			return anaco, liñaTex
		}
	}
	return "", 0
}

func lerRexistro(args map[string]any) (any, error) {
	id := int(numeroArg(args["id"]))
	e, ok := entradaRexistro(id)
	if !ok {
		return nil, fmt.Errorf("non hai ningunha compilación co id %d no rexistro (usa rexistro_listar)", id)
	}
	if e.Privado {
		return resumoEntrada(e), nil
	}
	parte, _ := args["parte"].(string)
	desde, cantidade := int(numeroArg(args["desde"])), int(numeroArg(args["cantidade"]))
	buscar, _ := args["buscar"].(string)
	out := resumoEntrada(e)
	texto := e.Log
	if parte == "tex" {
		texto = e.Tex
	}
	if buscar = strings.TrimSpace(buscar); buscar != "" {
		var achados []string
		for i, l := range strings.Split(texto, "\n") {
			if strings.Contains(strings.ToLower(l), strings.ToLower(buscar)) {
				achados = append(achados, fmt.Sprintf("%5d| %s", i+1, l))
				if len(achados) == 40 {
					break
				}
			}
		}
		out["achados"] = achados
		return out, nil
	}
	if parte == "" || parte == "erro" {
		anaco, liña := erroDoLog(e.Log)
		if anaco == "" {
			out["erro_latex"] = "o log non ten ningunha liña «!» de erro"
			return out, nil
		}
		out["erro_latex"] = anaco
		if liña > 0 && e.Tex != "" {
			// O .tex arredor da liña do erro: é o que di de que texto do documento vén.
			t, _ := tramoNumerado(e.Tex, liña-12, 20)
			out["liña_tex"] = liña
			out["tex_arredor"] = t
		}
		return out, nil
	}
	if desde == 0 && parte == "log" {
		// Por defecto, o final do log (onde están os erros).
		n := strings.Count(e.Log, "\n") + 1
		desde = max(n-max(cantidade, 60)+1, 1)
	}
	t, total := tramoNumerado(texto, desde, cantidade)
	out["parte"] = parte
	out["texto"] = t
	out["total_liñas"] = total
	return out, nil
}

func numeroArg(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case int:
		return float64(x)
	case string:
		var n float64
		fmt.Sscan(x, &n)
		return n
	}
	return 0
}

// ferramentasRexistro: ler o rexistro interno (todas son de lectura).
var ferramentasRexistro = []ferramentaBackend{
	{
		def: IAFerramenta{
			Nome: "rexistro_listar", Tipo: tipoLectura, Onde: "backend",
			Descricion: "Lista as últimas compilacións do documento que fixo Lúa (e os debuxos TikZ que non compilaron), da máis recente á máis antiga: id, hora, documento, se saíron ben, o erro e os avisos. Úsaa SEMPRE que xerar falle ou haxa avisos, antes de cambiar nada.",
		},
		exec: func(_ *App, _ map[string]any) (any, error) {
			rexistroInterno.Lock()
			defer rexistroInterno.Unlock()
			out := []map[string]any{}
			for i := len(rexistroInterno.entradas) - 1; i >= 0; i-- {
				out = append(out, resumoEntrada(rexistroInterno.entradas[i]))
			}
			if len(out) == 0 {
				return map[string]any{"entradas": out, "nota": "aínda non se compilou nada nesta sesión"}, nil
			}
			return map[string]any{"entradas": out}, nil
		},
	},
	{
		def: IAFerramenta{
			Nome: "rexistro_ler", Tipo: tipoLectura, Onde: "backend",
			Descricion: "Le unha compilación do rexistro interno. Sen 'parte' (ou parte='erro') devolve o primeiro erro de LaTeX do log e o .tex xerado arredor da liña do erro (l.NN): con iso sabes que texto do documento o causou (unha fórmula, un circuíto, un carácter...) para arranxalo NA SÚA ORIXE (o dato, a pregunta, a plantilla), nunca no .tex. parte='log' ou 'tex' le o texto completo por tramos numerados ('desde', 'cantidade'; o log, por defecto, o final). 'buscar' devolve as liñas que conteñen ese texto.",
			Parametros: esqObx(map[string]any{
				"id":        esqNum("Id da compilación (de rexistro_listar). 0 ou sen poñer = a última."),
				"parte":     esqEnum("Que ler.", "erro", "log", "tex"),
				"desde":     esqNum("Primeira liña (desde 1)."),
				"cantidade": esqNum("Cantas liñas (por defecto 60, máximo 400)."),
				"buscar":    esqTxt("Texto que buscar no log ou no .tex (p. ex. «Undefined control sequence» ou un anaco do enunciado)."),
			}),
		},
		exec: func(_ *App, args map[string]any) (any, error) { return lerRexistro(args) },
	},
}

func init() {
	ferramentasBackend = append(ferramentasBackend, ferramentasRexistro...)
	for _, m := range []map[string]bool{ferramentasDoAmbitoFormularioDatos, ferramentasDoAmbitoFormulario, ferramentasDoAmbitoPlantilla} {
		m["rexistro_listar"], m["rexistro_ler"] = true, true
	}
}
