package main

// DESEÑO dos documentos (cronograma, actas e formularios). O deseño vive na
// plantilla coma datos (Deseno) e deseno_latex.go xera o LaTeX; se a plantilla
// ten Estilo (LaTeX libre), manda ese. Lúa só xera o contido; os macros
// (\chip, \tarefa, \trimestre, \actacab, \actasecc...) son os que usa ese
// contido, así que o aspecto do documento é cousa do deseño.

// estiloPorDefectoLatex é o deseño de Lúa en LaTeX: o que xera deseno_latex.go
// cun deseño baleiro. É o punto de partida de «Usar o estilo de Lúa» no bloque
// de LaTeX libre do editor. Pode usar as cores lua-* e da paleta (tf-*, tt-*,
// tb-*, tc-*, mes-*, mesf-*, gris-linea, gris-texto), \fontchip e \luamarca.
var estiloPorDefectoLatex = resolverDeseno(nil, nil).latexEstilo()

// vivosTarefasCrono é o extremo vivo do degradado das etiquetas de cada cor de
// tarefa (o outro extremo é a cor de texto da paleta). Só vale para o PDF.
var vivosTarefasCrono = map[string]string{
	"blue": "2563EB", "emerald": "059669", "amber": "D97706", "rose": "E11D48",
	"purple": "9333EA", "cyan": "0891B2", "indigo": "4F46E5", "stone": "57534E",
}

// coresMesVivasCrono é a cor forte da etiqueta de cada mes (0 = xaneiro), a
// versión saturada de coresMesCrono: estacións de fondo cálido no outono e
// frío no inverno.
var coresMesVivasCrono = [12]string{
	"1D4ED8", "0E7490", "0F766E", "15803D", "4D7C0F", "A16207",
	"B45309", "B91C1C", "C2410C", "B45309", "57534E", "334155",
}

// bloqueInformeCrono son as definicións que dan ao documento o seu aspecto e
// que se inxectan nunha plantilla COMPLETA (co seu \documentclass): cores,
// marcas, logotipo, letras (se a plantilla non fixa as súas) e o deseño. Así o
// documento non cambia de cara segundo o tipo de plantilla.
func bloqueInformeCrono(logo *imaxeCrono, engine string, tip *TipografiaOpts, plantilla *PlantillaLatex) string {
	return compoñerDeseno(engine, tip, logo, plantilla).BloqueCompleta
}
