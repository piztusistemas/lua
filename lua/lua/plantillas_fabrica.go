package main

import "time"

// plantillasDeFabrica son as plantillas de exemplo coas que arranca a
// biblioteca (sementadas unha soa vez, ver sementarSeFai). Non son "de só
// lectura": están aí para abrir, ver como se fai unha, e modificalas ou
// duplicalas co nome, o teléfono e o logotipo do propio centro.
//
// Son as dúas FRAGMENTO (sen \documentclass): só cabeceira e pé, o
// preámbulo pono Lua. É o 90% dos casos e non esixe saber LaTeX máis alá do
// texto. Unha plantilla COMPLETA (co seu \documentclass, marxes e fontes) tamén
// vale - ver plantillas.go - pero non hai ningunha de exemplo: adóitase
// crear coa IA a partir dun documento do centro, ou importar doutro equipo.
//
// Ningunha require unha imaxe: unha \includegraphics{logotipo.png} sen o
// ficheiro correspondente faría fallar a compilación nada máis escoller a
// plantilla. Por iso o logotipo vai envolto en \IfFileExists: aparece en canto
// se sobe unha imaxe co nome «logotipo.png» dende "🖼️ Imaxes" no editor de
// plantillas, e se non hai imaxe a cabeceira sae igual, sen el.
//
// CENTRO, CURSO, ORIENTADOR e DEPARTAMENTO enchéos Lua a partir do curso
// aberto (ver variablesAutomaticasCrono); as variables da plantilla co
// mesmo nome e valor baleiro non as pisan, e con valor si - así un centro
// pode fixar o seu nome unha vez na plantilla e non escribilo en cada curso.
func plantillasDeFabrica() []PlantillaLatex {
	agora := time.Now().Format(time.RFC3339)
	// IDs fixas e lexibles (non a marca de tempo que usa GardarPlantilla):
	// así unha plantilla de fábrica sementada nun equipo é a mesma que
	// noutro, e un plantillas.json compartido entre dous ordenadores segue
	// apuntando ao mesmo sitio.
	novo := func(id, nome, desc, latex, markdown string, vars []PlantillaVar) PlantillaLatex {
		return PlantillaLatex{
			ID: id, Nome: nome, Descricion: desc,
			Latex: latex, Markdown: markdown, Variables: vars,
			Creado: agora, Modificado: agora, DeFabrica: true,
		}
	}
	moderna := novo("fabrica-moderna", "Portada moderna e cabeceira do centro",
		"Portada e última folla (sinatura) deseñadas na propia plantilla: editaas na pestana de portada e sinatura, e usan os datos do curso e do centro ({{CENTRO}}, {{XEFE}}, {{CARGO}}...). Ten tamén un texto de introdución co centro, o curso e o departamento.",
		modernaLatex, modernaMarkdown, nil)
	moderna.Portada = portadaPorDefectoLatex
	moderna.Sinatura = sinaturaPorDefectoLatex
	moderna.Introducion = introPorDefectoCrono
	return []PlantillaLatex{
		moderna,
		novo("fabrica-centro", "Cabeceira e pé do centro",
			"Cabeceira co nome do centro, o departamento e o curso (e o logotipo, se subes unha imaxe «logotipo.png»); pé co nome de quen orienta, teléfono e correo.",
			centroLatex, centroMarkdown, []PlantillaVar{
				{Nome: "CENTRO", Etiqueta: "Nome do centro (baleiro = o do curso)", Valor: ""},
				{Nome: "ORIENTADOR", Etiqueta: "Orientador/a (baleiro = o do curso)", Valor: ""},
				{Nome: "TELEFONO", Etiqueta: "Teléfono", Valor: ""},
				{Nome: "CORREO", Etiqueta: "Correo electrónico", Valor: ""},
			}),
		novo("fabrica-pe", "Só pé de páxina",
			"Sen cabeceira: un pé sobrio co centro, o contacto e o número de páxina.",
			peLatex, peMarkdown, []PlantillaVar{
				{Nome: "CENTRO", Etiqueta: "Nome do centro (baleiro = o do curso)", Valor: ""},
				{Nome: "CONTACTO", Etiqueta: "Teléfono / correo", Valor: ""},
			}),
	}
}

const centroLatex = `\pagestyle{fancy}
\fancyhf{}
\renewcommand{\headrulewidth}{0.4pt}
\renewcommand{\footrulewidth}{0.4pt}
\setlength{\headheight}{28pt}
\fancyhead[L]{\IfFileExists{logotipo.png}{\includegraphics[height=9mm]{logotipo.png}\hspace{3mm}}{}\small\bfseries {{CENTRO}} }
\fancyhead[R]{\small {{DEPARTAMENTO}} \textendash{} Curso {{CURSO}} }
\fancyfoot[L]{\footnotesize {{ORIENTADOR}} }
\fancyfoot[C]{\footnotesize Tel. {{TELEFONO}} \textperiodcentered{} {{CORREO}} }
\fancyfoot[R]{\footnotesize\thepage}

{{CORPO}}
`

const centroMarkdown = `**{{CENTRO}}** · {{DEPARTAMENTO}} · Curso {{CURSO}}

---

{{CORPO}}

---

{{ORIENTADOR}} · Tel. {{TELEFONO}} · {{CORREO}}
`

const peLatex = `\pagestyle{fancy}
\fancyhf{}
\renewcommand{\headrulewidth}{0pt}
\renewcommand{\footrulewidth}{0.4pt}
\fancyfoot[L]{\footnotesize {{CENTRO}} }
\fancyfoot[C]{\footnotesize {{CONTACTO}} }
\fancyfoot[R]{\footnotesize\thepage}

{{CORPO}}
`

const peMarkdown = `{{CORPO}}

---

{{CENTRO}} · {{CONTACTO}}
`

// modernaLatex é a maqueta da plantilla "moderna": sen \documentclass (Lua
// pon o preámbulo), cunha cabeceira lixeira e os tres marcadores de parte:
// {{PORTADA}} e {{SINATURA}} levan os deseños do propio editor e {{CORPO}} o
// índice, a introdución, o cronograma e o desenvolvemento.
const modernaLatex = `\pagestyle{fancy}
\fancyhf{}
\renewcommand{\headrulewidth}{0pt}
\renewcommand{\footrulewidth}{0pt}
\setlength{\headheight}{28pt}
\fancyhead[L]{\IfFileExists{logotipo.png}{\includegraphics[height=9mm]{logotipo.png}\hspace{3mm}}{}\sffamily\small\bfseries {{CENTRO}} }
\fancyhead[R]{\sffamily\small\color{gris-texto} {{DEPARTAMENTO}} \textperiodcentered{} Curso {{CURSO}} }
\fancyfoot[R]{\sffamily\footnotesize\thepage}

{{PORTADA}}

{{CORPO}}

{{SINATURA}}
`

const modernaMarkdown = `{{CORPO}}
`
