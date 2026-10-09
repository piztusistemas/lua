package main

// Escritor Markdown dos bloques dun formulario. Serve para exportar .md e, máis
// adiante, de base a Word e OpenDocument (o mesmo camiño que cronograma e actas).

import (
	"fmt"
	"strings"
	"time"
)

// mdCelaFormulario escapa unha cela de táboa Markdown: o «|» cortaríaa e un
// salto de liña rompería a fila.
func mdCelaFormulario(s string) string {
	s = mdEscaparCrono(strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", " "), "\n", " "))
	return strings.ReplaceAll(s, "|", `\|`)
}

// mdBloquesFormulario escribe os bloques en Markdown.
func mdBloquesFormulario(bloques []bloqueDoc, esc *escritorImaxesCrono, formato string) string {
	var b strings.Builder
	for _, bl := range bloques {
		switch bl.Tipo {
		case bloqueTitulo:
			b.WriteString(strings.Repeat("#", max(bl.Nivel, 2)) + " " + mdEscaparCrono(bl.Texto) + "\n\n")
		case bloqueFicha:
			for _, par := range bl.Pares {
				valor := strings.Join(strings.Fields(strings.ReplaceAll(par[1], "\n", " ")), " ")
				b.WriteString("**" + mdEscaparCrono(par[0]) + ":** " + mdEscaparCrono(valor) + "  \n")
			}
			b.WriteString("\n")
		case bloqueMarkdown:
			if md := parseMarkdownCrono(bl.Texto); len(md) > 0 {
				b.WriteString(mdBloquesCrono(md, nil, esc, 2))
			}
		case bloqueTaboa:
			if len(bl.Cabeceira) == 0 {
				continue
			}
			cab := make([]string, len(bl.Cabeceira))
			sep := make([]string, len(bl.Cabeceira))
			for i, c := range bl.Cabeceira {
				cab[i], sep[i] = mdCelaFormulario(c), "---"
			}
			b.WriteString("| " + strings.Join(cab, " | ") + " |\n| " + strings.Join(sep, " | ") + " |\n")
			for _, fila := range bl.Filas {
				celas := make([]string, len(bl.Cabeceira))
				for i := range celas {
					if i < len(fila) {
						celas[i] = mdCelaFormulario(fila[i])
					}
				}
				b.WriteString("| " + strings.Join(celas, " | ") + " |\n")
			}
			b.WriteString("\n")
		case bloqueLista:
			for _, el := range bl.Elementos {
				b.WriteString("- " + mdEscaparCrono(strings.Join(strings.Fields(el), " ")) + "\n")
			}
			b.WriteString("\n")
		case bloqueSalto:
			b.WriteString(saltoPaxinaMdCrono(formato))
		case bloqueSolucion:
			b.WriteString("> **" + mdEscaparCrono(bl.Rotulo) + "**\n>\n")
			for _, l := range strings.Split(strings.TrimSpace(mdBloquesCrono(parseMarkdownCrono(bl.Texto), nil, esc, 3)), "\n") {
				b.WriteString(strings.TrimRight("> "+l, " ") + "\n")
			}
			b.WriteString("\n")
		case bloqueGrafico:
			b.WriteString(mdGrafico(bl, esc))
		case bloqueImaxe:
			if im := esc.escribir(bl.Imaxe, bl.Texto); im != nil {
				w := float64(im.W) * 25.4 / 150
				w = min(w, larguraImaxeMaxMMFormulario)
				b.WriteString("![" + mdEscaparCrono(bl.Texto) + "](" + im.Ficheiro + "){width=" + mmLatexCrono(w) + "}\n\n")
			}
		}
	}
	return b.String()
}

// markdownFormulario compón o formulario enchido en Markdown. `formato` é "md",
// "docx" ou "odt" (só cambia o xeito de separar páxinas, ver saltoPaxinaMdCrono).
// Devolve o corpo sen envolver na plantilla, coma markdownInformeCrono.
func markdownFormulario(p PedidoFormulario, formato, dir string, agora time.Time, plantilla *PlantillaLatex) (string, []string, error) {
	f, d := normalizarFormulario(p.Formulario), p.Datos
	if err := validarFormulario(f); err != nil {
		return "", nil, fmt.Errorf("formulario non válido: %w", err)
	}
	esc := novoEscritorImaxesCrono(dir)
	auto := variablesFormulario(f, d, "", agora, plantilla)
	tx := novosTextosDoc(plantilla, auto)
	tx.engadirVariables(auto, "TITULO_FORMULARIO")
	bloques, avisosDatos := bloquesDocumentoFormulario(p, f, d, tx)

	var b strings.Builder
	if f.Portada {
		if im := esc.escribir(d.Contexto.Logo, "logotipo"); im != nil {
			b.WriteString("![Logotipo](" + im.Ficheiro + "){width=45mm}\n\n")
		}
		b.WriteString("# " + mdEscaparCrono(strings.ToUpper(auto["TITULO"])) + "\n\n")
		b.WriteString("**" + mdEscaparCrono(auto["DEPARTAMENTO"]) + "**\n\n")
		b.WriteString("**" + mdEscaparCrono(auto["CENTRO"]) + "**\n\n")
		b.WriteString(mdEscaparCrono(auto["LOCALIDADE"]+", "+auto["DATA_LONGA"]) + "\n\n")
		b.WriteString(saltoPaxinaMdCrono(formato))
	}
	b.WriteString(mdBloquesFormulario(bloques, esc, formato))
	if f.Sinatura {
		b.WriteString("\n\n" + mdEscaparCrono(auto["LOCALIDADE"]+", "+auto["DATA_LONGA"]) + "\n\n\n\n")
		b.WriteString("\\_\\_\\_\\_\\_\\_\\_\\_\\_\\_\\_\\_\\_\\_\\_\\_\\_\\_\\_\\_\\_\\_\\_\\_\\_\\_\n\n")
		b.WriteString("**" + mdEscaparCrono(auto["XEFE"]) + "**  \n" + mdEscaparCrono(auto["CARGO"]) + "\n")
	}

	// A cabeceira e o pé da plantilla engádeos markdownInformeConPlantilla.
	return b.String(), engadirSenRepetir(avisosDatos, esc.avisos), nil
}
