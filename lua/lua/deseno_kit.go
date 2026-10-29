package main

// «Kit do centro»: un deseño que se prepara unha vez e se reparte. Gárdase
// nun ficheiro .luadeseno (só o deseño, en JSON) que outra persoa abre no seu
// editor, e pódese aplicar dunha vez a todas as plantillas propias.

import (
	"edicion/idioma"
	"encoding/json"
	"os"
	"strings"
	"time"
)

const formatoFicheiroDeseno = "lua-deseno"

// ficheiroDeseno é o contido dun .luadeseno.
type ficheiroDeseno struct {
	Formato string `json:"formato"`
	Version int    `json:"version"`
	Nome    string `json:"nome,omitempty"`
	Deseno  Deseno `json:"deseno"`
}

// codificarDeseno devolve o JSON dun .luadeseno.
func codificarDeseno(d Deseno, nome string) ([]byte, error) {
	d = *normalizarDeseno(&d)
	return json.MarshalIndent(ficheiroDeseno{Formato: formatoFicheiroDeseno, Version: 1, Nome: strings.TrimSpace(nome), Deseno: d}, "", "  ")
}

// LerDesenoFicheiro le o contido dun .luadeseno (o texto do ficheiro) e
// devolve o deseño xa validado.
func (a *App) LerDesenoFicheiro(texto string) (Deseno, error) {
	var f ficheiroDeseno
	if err := json.Unmarshal([]byte(texto), &f); err != nil || f.Formato != formatoFicheiroDeseno {
		return Deseno{}, idioma.Errorf("go.deseno.ficheiro_non_valido", "o ficheiro non é un deseño de Lúa (.luadeseno)")
	}
	if f.Version > 1 {
		return Deseno{}, idioma.Errorf("go.deseno.ficheiro_novo", "este deseño fíxose cunha versión máis nova de Lúa: actualiza para abrilo")
	}
	return *normalizarDeseno(&f.Deseno), nil
}

// GardarDesenoFicheiro garda o deseño nun .luadeseno (diálogo de gardar).
// Devolve a ruta ("" se se cancelou).
func (a *App) GardarDesenoFicheiro(d Deseno, nome string) (string, error) {
	base := strings.TrimSpace(nome)
	if base == "" {
		base = "deseno"
	}
	ruta, err := saveFileDialogCompat("Gardar o deseño", "", nomeFicheiroPlantilla(base)+".luadeseno", "Deseño de Lúa (*.luadeseno)", "*.luadeseno")
	if err != nil || ruta == "" {
		return "", err
	}
	datos, err := codificarDeseno(d, nome)
	if err != nil {
		return "", err
	}
	return ruta, os.WriteFile(ruta, datos, 0o644)
}

// AplicarDesenoATodas pon o deseño en todas as plantillas propias (as de
// exemplo non se tocan) e devolve cantas cambiou. As que teñen LaTeX libre no
// Estilo tamén levan o deseño, pero alí segue mandando o seu LaTeX.
func (a *App) AplicarDesenoATodas(d Deseno) (int, error) {
	alm, err := loadAlmacenPlantillas()
	if err != nil {
		return 0, err
	}
	norm := normalizarDeseno(&d)
	n := 0
	agora := time.Now().Format(time.RFC3339)
	for i := range alm.Plantillas {
		if alm.Plantillas[i].DeFabrica {
			continue
		}
		copia := *norm
		alm.Plantillas[i].Deseno = &copia
		alm.Plantillas[i].Modificado = agora
		n++
	}
	if n == 0 {
		return 0, nil
	}
	return n, saveAlmacenPlantillas(alm)
}

// usoElementoDeseno é un anaco de documento que usa os macros de cada
// elemento, para comprobar o LaTeX propio dese elemento.
var usoElementoDeseno = map[string]string{
	"texto":           "Texto de proba.\\par Outro parágrafo.",
	"titulos":         "\\section{Título}\\subsection{Subtítulo}Texto.",
	"etiquetas":       "\\chip{blue}{Tarefa de proba}",
	"meses":           "\\medirmes{Setembro}{2 tarefas}\\mesbadge{0}{Setembro}{2 tarefas}\\mesvacio{1}{Outubro}",
	"trimestres":      "\\begin{trimestre}{1.º trimestre}{Setembro – Novembro}Contido\\end{trimestre}",
	"tarefas":         "\\tarefa{blue}{Tarefa de proba}",
	"actas_cabeceira": "\\actacab{Departamento}{Acta n.º 1}{Reunión ordinaria}\\actadato{Data}{1 de outubro}",
	"actas_apartados": "\\actasecc{Asistentes}\\actacampo{Campo}{Texto}\\actaprogreso{Grao}{50}",
	"actas_materias":  "\\actamateria{Materia}{Docente}",
	"sinaturas":       "\\actasinaturas{Redacta}{Nome}{Visto e prace}{Nome}",
}

// ComprobarElementoDeseno compila un documento mínimo co LaTeX propio dun
// elemento e devolve o erro explicado ("" se compila). Así o erro sae ao
// carón do elemento e non ao xerar o documento enteiro.
func (a *App) ComprobarElementoDeseno(d Deseno, id, latexPropio string) (string, error) {
	uso, ok := usoElementoDeseno[id]
	if !ok {
		return "", idioma.Errorf("go.deseno.elemento_inexistente", "a parte «%s» non existe", id)
	}
	motor := "xelatex"
	if !commandExists(motor) {
		motor = "pdflatex"
		if !commandExists(motor) {
			return "", idioma.Errorf("go.latex.motor_falta", "non se atopou %s. Instala unha distribución LaTeX (p.ex. 'sudo apt install %s')", "xelatex", "texlive-xetex")
		}
	}
	els := map[string]ElementoDeseno{}
	for k, v := range d.Elementos {
		els[k] = v
	}
	e := els[id]
	e.Latex = latexPropio
	els[id] = e
	d.Elementos = els
	dir, err := os.MkdirTemp("", "lua-elemento-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)
	dd := compoñerDeseno(motor, nil, nil, &PlantillaLatex{Deseno: &d})
	if engineUnicodeNativo(motor) {
		_ = escribirFontesInformeCrono(dir, dd.Fontes)
	}
	tex := preambuloDeseno(motor, dd, "") + "\\begin{document}\n" + babelListquotFix + formulasDefLatexPorDefecto + uso + "\n\\end{document}\n"
	if _, log, err := compilarLatexCrono(dir, tex, motor); err != nil {
		if pista := pistaLatexAccionable(log); pista != "" {
			return pista, nil
		}
		if det := extraerErroPDFLatex(log); det != "" {
			return det, nil
		}
		return err.Error(), nil
	}
	return "", nil
}
