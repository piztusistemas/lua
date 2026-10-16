// Package paquetes instala baixo demanda un paquete de LaTeX que falta.
//
// Unha distribución de LaTeX completa son varios GB, así que as aplicacións
// nunca a instalan enteira: poñen o subconxunto que precisan e listo. O prezo
// é que, tarde ou cedo, alguén escribe (ou a IA xera) un documento ou unha
// plantilla que usa un paquete que ese equipo non ten, e a compilación morre
// cun "File `tcolorbox.sty' not found" que non lle di nada ao profesorado.
//
// Este paquete pecha ese oco: cando unha compilación falla por un paquete que
// falta, Anotar garda cal foi e a aplicación ofrece un botón para instalalo,
// co mesmo camiño de permisos que xa usa a instalación de LaTeX (pkexec en
// Linux, diálogo nativo de administrador en macOS).
//
// Do exterior só precisan dúas costuras: quen emite o progreso da instalación
// (o módulo non coñece Wails) e como se comproba que o ficheiro xa queda
// dispoñible (kpsewhich, que vive no motor LaTeX da aplicación).
package paquetes

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"edicion/plataforma"
)

// Detección UNIVERSAL de "falta algo de LaTeX que se pode instalar". Un log de
// LaTeX pártese en liñas de ~80 caracteres, así que "not found" pode aparecer
// cortado ("not foun\nd"); Anotar normaliza os espazos antes de aplicar estas
// expresións, por iso aquí non se depende de que dúas palabras vaian xuntas.
var (
	// ficheiroQueFaltaRe: "File `X.EXT' not found" para calquera extensión que
	// un paquete de TeX Live poida fornecer (paquete .sty, clase .cls, idioma
	// de babel .ldf, definicións .def/.fd/.clo, config .cfg, fontes
	// .ttf/.otf, codificación .enc, input .tex).
	ficheiroQueFaltaRe = regexp.MustCompile(
		"File `([A-Za-z0-9@_.+ -]+\\.(?:sty|cls|ldf|def|fd|clo|sto|cfg|cmap|enc|ttf|otf|tex|bst|bbx|cbx))' not found")
	// babelIdiomaRe: babel dío doutro xeito cando falta o soporte dun idioma
	// -> "the language definition file spanish.ldf was not found" (ou o
	// galician.ldf, que á súa vez necesita spanish.ldf). Basta con recoñecer
	// "language definition file X.ldf": esa frase só aparece neste erro.
	babelIdiomaRe = regexp.MustCompile(`language definition file ([a-z@]+)\.ldf`)
	// fontspecFaltaRe: xelatex/lualatex non atopan unha fonte pedida por
	// fontspec (\setmainfont, \setmonofont...) nunha plantilla.
	fontspecFaltaRe = regexp.MustCompile(`The font "([^"\n]+)" cannot be found`)
	// espazosLogRe normaliza calquera bloque de espazos/saltos a un só espazo,
	// para que o corte de liña do log non rompa as expresións de enriba.
	espazosLogRe = regexp.MustCompile(`\s+`)
)

// Pendente é o que se lle amosa ao profesorado tras un fallo: que falta, se se
// pode instalar neste equipo, e QUE comando vai executar exactamente (amósase
// sempre antes de premer: instalar software de sistema non se fai ás
// agachadas).
type Pendente struct {
	Ficheiro     string `json:"ficheiro"` // "tcolorbox.sty"
	Nome         string `json:"nome"`     // "tcolorbox"
	PodeInstalar bool   `json:"podeInstalar"`
	Comando      string `json:"comando"`
	// Motivo explica por que NON se pode instalar automaticamente, cando
	// PodeInstalar é false (sen xestor de paquetes coñecido, sen pkexec...).
	Motivo string `json:"motivo"`
}

// Rexistro garda o ficheiro de LaTeX que faltou na última compilación. É
// seguro para uso concorrente: as compilacións poden correr en goroutines
// distintas.
type Rexistro struct {
	mu       sync.Mutex
	ficheiro string // nome do ficheiro, "" se non hai nada pendente
}

// Anotar mira o log dunha compilación fallida e garda o ficheiro que faltou,
// para que a interface poida ofrecer instalalo. Chámase desde TODOS os sitios
// que compilan: o log non chega ao frontend cando hai erro (o binding de Wails
// só entrega o error), así que este é o único xeito de que a información
// sobreviva á chamada.
func (r *Rexistro) Anotar(log string) {
	norm := espazosLogRe.ReplaceAllString(log, " ")
	r.mu.Lock()
	defer r.mu.Unlock()
	switch {
	case ficheiroQueFaltaRe.MatchString(norm):
		r.ficheiro = strings.TrimSpace(ficheiroQueFaltaRe.FindStringSubmatch(norm)[1])
	case babelIdiomaRe.MatchString(norm):
		// Gárdase co sufixo .ldf para que comandoInstalar saiba que é un
		// idioma de babel (paquete texlive-lang-*), non un .sty.
		r.ficheiro = babelIdiomaRe.FindStringSubmatch(norm)[1] + ".ldf"
	case fontspecFaltaRe.MatchString(norm):
		// Sufixo .font: pseudo-extensión interna (non existe tal ficheiro),
		// só para que o resto do código saiba que é unha fonte.
		r.ficheiro = strings.TrimSpace(fontspecFaltaRe.FindStringSubmatch(norm)[1]) + ".font"
	default:
		r.ficheiro = ""
	}
}

// Limpar bórrao tras unha compilación correcta (ou tras instalalo): senón o
// botón "instalar" quedaría pendurado despois de que o problema xa non exista.
func (r *Rexistro) Limpar() {
	r.mu.Lock()
	r.ficheiro = ""
	r.mu.Unlock()
}

// Pendente devolve o que quedou anotado na última compilación. Ficheiro
// baleiro = non hai nada que ofrecer.
func (r *Rexistro) Pendente() Pendente {
	r.mu.Lock()
	ficheiro := r.ficheiro
	r.mu.Unlock()
	if ficheiro == "" {
		return Pendente{}
	}
	nome := NomeDesdeFicheiro(ficheiro)
	info := Pendente{Ficheiro: ficheiro, Nome: nome}
	argv, descricion, motivo := comandoInstalar(ficheiro, nome)
	if argv == nil {
		info.Motivo = motivo
		return info
	}
	info.PodeInstalar = true
	info.Comando = descricion
	return info
}

// Instalacion reúne as costuras que o módulo non pode resolver por si mesmo.
type Instalacion struct {
	// NomeApp é o nome visible da aplicación, para a mensaxe de permisos.
	NomeApp string
	// InformarProgreso emite o progreso da instalación e devolve a función que
	// o detén. A aplicación é quen ten o emisor de eventos.
	InformarProgreso func(que string) (parar func())
	// FicheiroInstalado comproba se LaTeX atopa xa o ficheiro (kpsewhich).
	FicheiroInstalado func(ficheiro string) bool
}

// Instalar executa o comando de instalación do paquete anotado na última
// compilación fallida. Devolve o log da instalación e se quedou instalado de
// verdade; se non hai nada pendente ou non se pode instalar automaticamente,
// devolve ("", false) sen executar nada.
func (r *Rexistro) Instalar(inst Instalacion) (log string, ok bool) {
	info := r.Pendente()
	if info.Ficheiro == "" || !info.PodeInstalar {
		return "", false
	}
	argv, _, _ := comandoInstalar(info.Ficheiro, info.Nome)
	if argv == nil {
		return "", false
	}

	env, limpar := plataforma.PrepararEntornoInstalacion(
		fmt.Sprintf("%s precisa permisos de administrador para instalar o paquete de LaTeX «%s».", inst.NomeApp, info.Nome))
	defer limpar()

	cmd := exec.Command(argv[0], argv[1:]...)
	if env != nil {
		cmd.Env = env
	}
	plataforma.OcultarConsola(cmd)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	parar := func() {}
	if inst.InformarProgreso != nil {
		parar = inst.InformarProgreso("paquete-latex")
	}
	runErr := cmd.Run()
	parar()

	log = strings.TrimSpace(out.String())
	// Comprobación real por riba do código de saída: o que importa é que LaTeX
	// atope xa o ficheiro (algúns xestores devolven != 0 por avisos). Para
	// .sty/.cls/.ldf/.def... kpsewhich sábeo; para unha fonte (.font é unha
	// pseudo-extensión interna, non hai tal ficheiro) hai que fiarse do código
	// de saída do xestor de paquetes.
	ext := strings.ToLower(filepath.Ext(info.Ficheiro))
	instalado := false
	if inst.FicheiroInstalado != nil {
		instalado = inst.FicheiroInstalado(info.Ficheiro)
	}
	if ext == ".font" || ext == ".ttf" || ext == ".otf" {
		instalado = runErr == nil
	}
	if instalado {
		r.Limpar()
		return log, true
	}
	if runErr != nil {
		log = strings.TrimSpace(log + "\n" + runErr.Error())
	}
	return log, false
}

// paquetesDebian traduce un paquete de CTAN ao paquete de Debian/Ubuntu que o
// contén. Debian NON reparte TeX Live por paquete de CTAN (a diferenza de
// openSUSE ou Fedora), senón en coleccións grandes, así que hai que mapear:
// aquí van as coleccións, e o que non estea nesta táboa cae en
// texlive-latex-extra, que é onde vive a inmensa maioría do que lle falta a un
// documento normal. Non se adiviña máis ca isto a propósito: un nome de
// paquete inventado faría fallar o apt-get enteiro.
var paquetesDebian = map[string]string{
	// Debuxo e gráficas (texlive-pictures).
	"pgfplots": "texlive-pictures", "circuitikz": "texlive-pictures",
	"tikz": "texlive-pictures", "pgf": "texlive-pictures",
	"gnuplot-lua-tikz": "texlive-pictures",
	// Ciencias (texlive-science).
	"siunitx": "texlive-science", "mhchem": "texlive-science",
	"chemfig": "texlive-science", "circuitikz-science": "texlive-science",
	// Fontes (texlive-fonts-extra).
	"fontawesome": "texlive-fonts-extra", "fontawesome5": "texlive-fonts-extra",
	"roboto": "texlive-fonts-extra", "inconsolata": "texlive-fonts-extra",
	"fira": "texlive-fonts-extra", "lato": "texlive-fonts-extra",
	// Básicos que xa deberían estar, pero por se acaso.
	"amsmath": "texlive-latex-recommended", "graphicx": "texlive-latex-base",
	"geometry": "texlive-latex-base", "hyperref": "texlive-latex-base",
	"babel": "texlive-latex-base", "fancyhdr": "texlive-latex-recommended",
	"booktabs": "texlive-latex-recommended",
	// Familias de fonte do botón "Axustes do texto" (ver presetsFonte no motor
	// LaTeX da aplicación).
	"mathptmx": "texlive-latex-recommended", "helvet": "texlive-latex-recommended",
	"mathpazo": "texlive-latex-recommended",
	// Deseño do PDF (letras por papel, tamaño): ver edicion/latex/letras.go.
	"scrextend": "texlive-latex-recommended", "unicode-math": "texlive-latex-extra",
	"texgyretermes-math": "fonts-texgyre-math", "texgyrepagella-math": "fonts-texgyre-math",
	"texgyreschola-math": "fonts-texgyre-math", "texgyrebonum-math": "fonts-texgyre-math",
	"FiraMath-Regular": "texlive-fonts-extra", "LibertinusMath-Regular": "texlive-fonts-extra",
	"LibertinusSerif-Regular": "texlive-fonts-extra", "FiraSans-Regular": "texlive-fonts-extra",
	"SourceSerif4-Regular": "texlive-fonts-extra",
}

// PaqueteDebianPorDefecto é a colección grande na que cae o que non estea na
// táboa.
const PaqueteDebianPorDefecto = "texlive-latex-extra"

// NomeDesdeFicheiro saca o nome "limpo" (sen extensión) de calquera dos
// ficheiros que Anotar pode gardar: "tcolorbox.sty" -> "tcolorbox",
// "spanish.ldf" -> "spanish", "Fira Sans.font" -> "Fira Sans".
func NomeDesdeFicheiro(f string) string {
	return strings.TrimSuffix(f, filepath.Ext(f))
}

// grupoBabel mapea un idioma de babel á colección de TeX Live que o trae.
// Debian/Arch reparten os idiomas por grupos (texlive-lang-spanish,
// texlive-lang-european...), non un por idioma; o que non teña grupo propio
// vive en "european" (galician, catalán, éuscaro, holandés, nórdicos,
// centroeuropeos...).
func grupoBabel(lang string) string {
	switch lang {
	case "spanish":
		return "spanish"
	case "french", "francais", "acadian", "canadien":
		return "french"
	case "portuguese", "portuges", "brazil", "brazilian":
		return "portuguese"
	case "german", "ngerman", "austrian", "naustrian", "swissgerman":
		return "german"
	case "english", "usenglish", "american", "british", "ukenglish", "canadian", "australian", "newzealand":
		return "english"
	case "italian":
		return "italian"
	case "russian", "ukrainian", "bulgarian", "serbianc":
		return "cyrillic"
	case "greek", "polutonikogreek", "ibygreek":
		return "greek"
	case "arabic", "farsi":
		return "arabic"
	case "japanese":
		return "japanese"
	case "chinese":
		return "chinese"
	case "korean":
		return "korean"
	default:
		return "european"
	}
}

// PaqueteDebianPara traduce "que ficheiro falta" -> "paquete apt". Distingue
// por extensión: idiomas de babel (.ldf) van en texlive-lang-<grupo>, as
// fontes en texlive-fonts-extra, e o resto (.sty/.cls/.def/...) pasa pola
// táboa paquetesDebian ou cae na colección grande.
func PaqueteDebianPara(ext, nome string) string {
	switch ext {
	case ".ldf":
		return "texlive-lang-" + grupoBabel(nome)
	case ".font", ".ttf", ".otf", ".enc":
		return "texlive-fonts-extra"
	}
	if deb := paquetesDebian[nome]; deb != "" {
		return deb
	}
	return PaqueteDebianPorDefecto
}

// comandoInstalar arma o comando de instalación para ESTE equipo. Devolve
// argv=nil e un motivo lexible cando non hai forma de facelo automaticamente.
//
// Cada sistema resólvese do xeito máis exacto que permita:
//   - Fedora ten "provides" virtuais por ficheiro: dnf install "tex(x.sty)"
//     acerta sempre, sen táboas.
//   - openSUSE reparte TeX Live por paquete de CTAN: texlive-<nome>.
//   - Debian/Ubuntu reparte por coleccións: ver paquetesDebian.
//   - macOS (MacTeX/BasicTeX) e calquera TeX Live oficial traen tlmgr, que
//     instala por nome de paquete de CTAN.
func comandoInstalar(ficheiro, nome string) (argv []string, descricion, motivo string) {
	ext := strings.ToLower(filepath.Ext(ficheiro))
	// alvoCTAN: nome do paquete tal e como o coñecen tlmgr/mpm (CTAN). Para un
	// idioma de babel é "babel-<idioma>"; para unha fonte, a mellor
	// aproximación é o nome en minúsculas sen espazos.
	alvoCTAN := nome
	switch ext {
	case ".ldf":
		alvoCTAN = "babel-" + nome
	case ".font", ".ttf", ".otf":
		alvoCTAN = strings.ToLower(strings.ReplaceAll(nome, " ", ""))
	}

	switch runtime.GOOS {
	case "linux":
		var base []string
		switch {
		case plataforma.CommandExists("apt-get"):
			base = []string{"sh", "-c", "apt-get update && apt-get install -y " + PaqueteDebianPara(ext, nome)}
		case plataforma.CommandExists("dnf"):
			// Fedora ten "provides" virtuais por ficheiro para .sty/.cls/.ldf/
			// .def/.fd... - acerta sen táboas. As fontes van pola colección.
			if ext == ".font" || ext == ".ttf" || ext == ".otf" {
				base = []string{"dnf", "install", "-y", "texlive-collection-fontsextra"}
			} else {
				base = []string{"dnf", "install", "-y", fmt.Sprintf("tex(%s)", ficheiro)}
			}
		case plataforma.CommandExists("zypper"):
			switch ext {
			case ".ldf":
				base = []string{"zypper", "install", "-y", "texlive-babel-" + nome}
			case ".font", ".ttf", ".otf":
				base = []string{"zypper", "install", "-y", "texlive-fonts-extra"}
			default:
				base = []string{"zypper", "install", "-y", "texlive-" + nome}
			}
		case plataforma.CommandExists("pacman"):
			switch ext {
			case ".ldf":
				base = []string{"pacman", "-Sy", "--noconfirm", "texlive-lang" + grupoBabel(nome)}
			case ".font", ".ttf", ".otf":
				base = []string{"pacman", "-Sy", "--noconfirm", "texlive-fontsextra"}
			default:
				base = []string{"pacman", "-Sy", "--noconfirm", "texlive-latexextra"}
			}
		default:
			return nil, "", "non se atopou un xestor de paquetes coñecido (apt/dnf/zypper/pacman)"
		}
		if !plataforma.CommandExists("pkexec") {
			return nil, "", "non se atopou 'pkexec' para pedir permisos de administrador; instálao nun terminal"
		}
		return append([]string{"pkexec"}, base...), strings.Join(base, " "), ""

	case "darwin":
		if !plataforma.CommandExists("tlmgr") {
			return nil, "", "non se atopou tlmgr (instala MacTeX dende Opcións)"
		}
		// sudo -A: MacTeX vive en /usr/local/texlive, propiedade de root. O -A
		// colle o askpass que prepara PrepararEntornoInstalacion
		// (edicion/plataforma/brew_darwin.go), o mesmo diálogo nativo que xa se
		// usa para instalar MacTeX ou Maxima.
		return []string{"sudo", "-A", "tlmgr", "install", alvoCTAN},
			"sudo tlmgr install " + alvoCTAN, ""

	case "windows":
		// MiKTeX xa instala paquetes que faltan el só (por defecto,
		// preguntando); se chegou aquí é que esa opción está desactivada ou
		// que é unha instalación de TeX Live. mpm é o xestor de MiKTeX.
		if plataforma.CommandExists("mpm") {
			return []string{"mpm", "--install=" + alvoCTAN}, "mpm --install=" + alvoCTAN, ""
		}
		if plataforma.CommandExists("tlmgr") {
			return []string{"tlmgr", "install", alvoCTAN}, "tlmgr install " + alvoCTAN, ""
		}
		return nil, "", "non se atopou mpm nin tlmgr (o xestor de paquetes de MiKTeX/TeX Live)"
	}
	return nil, "", "sistema operativo non recoñecido"
}
