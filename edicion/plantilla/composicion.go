package plantilla

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// varRe recoñece un marcador {{NOME}}. Esixir MAIÚSCULAS (e que empece por
// letra) é o que evita confundir unha variable cun grupo LaTeX lexítimo:
// \frac{{a+b}}{2} non ten ningún marcador, \textbf{ {{CENTRO}} } si.
var varRe = regexp.MustCompile(`\{\{\s*([A-ZÁÉÍÓÚÜÑ][A-Z0-9_ÁÉÍÓÚÜÑ]*)\s*\}\}`)

// SubstituirVariables cambia cada {{NOME}} polo seu valor. Un marcador sen valor
// NON se deixa tal cal (aparecería literalmente no PDF): queda baleiro e
// avísase, que é o que deixa ver o erro sen estragar o documento.
// {{CORPO}}/{{PORTADA}}/{{SINATURA}} non se tocan aquí - insírense despois,
// para que un {{X}} escrito dentro do propio documento non se interprete coma
// variable da plantilla.
//
// `escapar` (non nil só no camiño LaTeX) escapa o VALOR de cada variable antes
// de metelo na maqueta: o valor é texto plano dun formulario (nome do centro,
// materia) ou o nome do ficheiro, e un "_", "&", "%", "→"... sen escapar
// rompía a compilación con "Missing $ inserted" (report real: un boletín
// titulado "unidade_2" -> "\bfseries unidade_2" -> erro). A maqueta en si NON se
// escapa (leva LaTeX a propósito); en Markdown non fai falla, así que alí vai
// nil.
func SubstituirVariables(texto string, p *Plantilla, auto map[string]string, baleiro string, escapar func(string) string) (string, []string) {
	valores := map[string]string{}
	for k, v := range auto {
		valores[k] = v
	}
	if p != nil {
		for _, v := range p.Variables {
			// Un valor baleiro NON pisa o automático: unha plantilla cun CENTRO
			// sen encher usa o nome do centro do propio curso.
			if _, hai := valores[v.Nome]; hai && strings.TrimSpace(v.Valor) == "" {
				continue
			}
			valores[v.Nome] = v.Valor
		}
	}
	var avisos []string
	vistos := map[string]bool{}
	out := varRe.ReplaceAllStringFunc(texto, func(m string) string {
		nome := varRe.FindStringSubmatch(m)[1]
		if MarcadoresDeParte(nome) {
			return m
		}
		val, ok := valores[nome]
		if !ok {
			if !vistos[nome] {
				vistos[nome] = true
				avisos = append(avisos, fmt.Sprintf("a plantilla usa {{%s}}, que non está definida: quedou baleira", nome))
			}
			return baleiro
		}
		if strings.TrimSpace(val) == "" {
			return baleiro
		}
		if escapar != nil {
			return escapar(val)
		}
		return val
	})
	return out, avisos
}

// usepackageRe recoñece as ordes que cargan un paquete, xunto coas súas
// opcións.
var usepackageRe = regexp.MustCompile(`\\(?:usepackage|RequirePackage)\s*(?:\[[^\]]*\])?\s*\{([^}]*)\}`)

// PaquetesDeclarados recolle os nomes de paquete que o LaTeX xa carga, para non
// duplicarllos (un \usepackage repetido con opcións distintas é un erro duro de
// LaTeX: "Option clash for package ...").
func PaquetesDeclarados(latex string) map[string]bool {
	out := map[string]bool{}
	for _, m := range usepackageRe.FindAllStringSubmatch(latex, -1) {
		for _, nome := range strings.Split(m[1], ",") {
			out[strings.TrimSpace(nome)] = true
		}
	}
	return out
}

// MotorParaPlantilla: unha plantilla que use fontspec/fontes do sistema precisa
// xelatex aínda que en Opcións estea pdflatex, e ao revés. Baleiro = manda
// Opcións, coma sempre.
func MotorParaPlantilla(p *Plantilla, porDefecto string) string {
	if p != nil && strings.TrimSpace(p.Motor) != "" {
		return strings.TrimSpace(p.Motor)
	}
	return porDefecto
}

// CopiarImaxesPlantilla deixa as imaxes da plantilla no cartafol de destino tal
// cal e mais en images/, para que o mesmo nome valga no .tex (\includegraphics
// resólveo desde o cartafol de compilación) e nas imaxes de <PLOT>/<IMG>.
func (d *Deposito) CopiarImaxesPlantilla(p *Plantilla, destDir string) error {
	return d.EscribirImaxesPlantilla(p, destDir, filepath.Join(destDir, "images"))
}

// CopiarImaxesPlantillaMarkdown só usa images/: a diferenza do .tex (que se
// compila nun temporal), o .md exportado escríbese no cartafol REAL do
// profesorado, e non hai por que espallarlle alí o logo solto cando o Markdown
// xa o referencia coma images/logo.png.
func (d *Deposito) CopiarImaxesPlantillaMarkdown(p *Plantilla, destDir string) error {
	return d.EscribirImaxesPlantilla(p, filepath.Join(destDir, "images"))
}

// EscribirImaxesPlantilla copia as imaxes da plantilla en cada un dos
// cartafoles indicados (creándoos se fai falla). Unha plantilla sen imaxes (o
// normal) non é erro.
func (d *Deposito) EscribirImaxesPlantilla(p *Plantilla, dirs ...string) error {
	if p == nil {
		return nil
	}
	orixe, err := d.DirImaxes(p.ID)
	if err != nil {
		return err
	}
	entradas, err := os.ReadDir(orixe)
	if err != nil {
		return nil // a plantilla non ten imaxes
	}
	for _, e := range entradas {
		if e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(orixe, e.Name()))
		if err != nil {
			return err
		}
		for _, dir := range dirs {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(dir, e.Name()), data, 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}

// DocumentoMarkdownConPlantilla é o equivalente de documentoLatexConPlantilla
// para .md/.docx/.odt. Sen plantilla (ou cunha plantilla sen parte Markdown)
// devolve o corpo tal cal; con ela, substitúe as variables da parte Markdown e
// mete o corpo onde vaia {{CORPO}}.
func DocumentoMarkdownConPlantilla(corpo string, p *Plantilla, auto map[string]string) (string, []string) {
	if p == nil || strings.TrimSpace(p.Markdown) == "" {
		return corpo, nil
	}
	texto, avisos := SubstituirVariables(p.Markdown, p, auto, "", nil)
	return strings.ReplaceAll(texto, MarcadorCorpo, corpo), avisos
}

// PaqueteRequirido é un paquete que o documento precisa: nome (o que aparece en
// \usepackage{nome}) e a liña completa que o carga.
type PaqueteRequirido struct{ Nome, Linha string }

// Preambulo reúne o que o módulo necesita da aplicación para inxectar o
// preámbulo obrigatorio: o coñecemento do motor, as declaracións Unicode, a
// liña de babel do idioma activo e a marca que se pon diante do bloque.
type Preambulo struct {
	// UnicodeNativo di se o motor xa entende UTF-8 (xelatex/lualatex).
	UnicodeNativo func(engine string) bool
	// DeclaracionsUnicode devolve o bloque que hai que engadir nos motores que
	// non entenden UTF-8.
	DeclaracionsUnicode func() string
	// Babel devolve a liña de babel do idioma activo ("" se non hai).
	Babel func() string
	// Marca é o comentario que se pon diante do bloque inxectado.
	Marca string
}

// InxectarPreambuloObrigatorio engade, xusto antes do \begin{document} dunha
// plantilla COMPLETA, o que fai falla para que o documento siga funcionando: os
// paquetes que faltan (os que pase a aplicación máis os do motor e babel), o
// soporte de UTF-8 propio de cada motor e mailo bloque extra (a xeometría da
// maqueta, se a hai). Non toca nada do que a plantilla xa declare - a idea é que
// o profesorado poida cambiar clase, marxes e fontes libremente.
//
// Se non falta nada e non hai bloque extra, o documento devólvese intacto: non
// se lle mete un comentario a unha plantilla que xa está completa.
func InxectarPreambuloObrigatorio(doc, engine string, paquetes []PaqueteRequirido, bloque string, pre Preambulo) string {
	pos := strings.Index(doc, `\begin{document}`)
	if pos < 0 {
		// Sen \begin{document} non hai onde inxectar (plantilla a medio
		// escribir): déixase tal cal e que falle o compilador, que dá unha
		// mensaxe moito máis clara ca calquera adiviña nosa.
		return doc
	}
	declarados := PaquetesDeclarados(doc[:pos])
	var faltan []string
	if pre.UnicodeNativo != nil && pre.UnicodeNativo(engine) {
		if !declarados["fontspec"] {
			faltan = append(faltan, `\usepackage{fontspec}`)
		}
	} else {
		if !declarados["inputenc"] {
			faltan = append(faltan, `\usepackage[utf8]{inputenc}`)
		}
		if !declarados["fontenc"] {
			faltan = append(faltan, `\usepackage[T1]{fontenc}`)
		}
		if pre.DeclaracionsUnicode != nil {
			faltan = append(faltan, strings.TrimRight(pre.DeclaracionsUnicode(), "\n"))
		}
	}
	for _, p := range paquetes {
		if !declarados[p.Nome] {
			faltan = append(faltan, p.Linha)
		}
	}
	if !declarados["babel"] && pre.Babel != nil {
		if b := strings.TrimRight(pre.Babel(), "\n"); b != "" {
			faltan = append(faltan, b)
		}
	}
	if len(faltan) == 0 && strings.TrimSpace(bloque) == "" {
		return doc
	}
	if b := strings.TrimRight(bloque, "\n"); b != "" {
		faltan = append(faltan, b)
	}
	return doc[:pos] + pre.Marca + "\n" + strings.Join(faltan, "\n") + "\n" + doc[pos:]
}
