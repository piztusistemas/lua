package main

// Instalación baixo demanda dun paquete de LaTeX que falta: a lóxica xenérica
// (recoñecer no log o ficheiro que faltou, traducilo a un comando de
// instalación do sistema, executalo e comprobar que quedou instalado) vive
// unha soa vez en edicion/latex/paquetes, compartida con Yang e cos módulos
// futuros. Aquí queda o encaixe coa aplicación: o rexistro global, os dous
// bindings de Wails e as dúas costuras que o módulo non pode coñecer (o
// evento de progreso da instalación e kpsewhich).

import "edicion/latex/paquetes"

// rexistroPaquetes garda o paquete que faltou na última compilación fallida.
// O módulo xa o protexe cun mutex.
var rexistroPaquetes paquetes.Rexistro

// PaqueteLatexPendente é o que se lle amosa ao profesorado tras un fallo.
// Alias do tipo do módulo (non unha copia), para que os bindings de Wails e o
// frontend (ofrecerInstalarPaqueteLatex en main.js) sigan co mesmo nome e os
// mesmos campos JSON.
type PaqueteLatexPendente = paquetes.Pendente

// anotarPaqueteQueFalta gárdase desde TODOS os sitios que compilan
// (latexdoc.go e plantillas.go): o log non chega ao frontend cando hai erro,
// así que este é o único xeito de que a información sobreviva á chamada.
func anotarPaqueteQueFalta(log string) { rexistroPaquetes.Anotar(log) }

// limparPaqueteQueFalta bórrao tras unha compilación correcta (ou tras
// instalalo).
func limparPaqueteQueFalta() { rexistroPaquetes.Limpar() }

// PaqueteLatexQueFalta devólvelle ao frontend o que quedou anotado na última
// compilación. Ficheiro baleiro = non hai nada que ofrecer.
func (a *App) PaqueteLatexQueFalta() PaqueteLatexPendente {
	return rexistroPaquetes.Pendente()
}

// InstalarPaqueteLatex instala o paquete anotado na última compilación
// fallida. Mesmo contrato ca InstallLatex/InstallPandoc (installer.go):
// devolve InstallStatus e vai informando do progreso polo mesmo evento, así
// que o frontend non precisa nada novo para amosar "instalando...".
func (a *App) InstalarPaqueteLatex() InstallStatus {
	info := rexistroPaquetes.Pendente()
	if info.Ficheiro == "" {
		return InstallStatus{Log: "Non hai ningún paquete pendente de instalar."}
	}
	if !info.PodeInstalar {
		return InstallStatus{Log: "Non se pode instalar automaticamente: " + info.Motivo}
	}
	log, ok := rexistroPaquetes.Instalar(paquetes.Instalacion{
		NomeApp:           "Lúa",
		InformarProgreso:  informarProgresoInstalacion,
		FicheiroInstalado: kpsewhichFound,
	})
	if ok {
		return InstallStatus{Success: true, Log: log}
	}
	return InstallStatus{Log: log}
}
