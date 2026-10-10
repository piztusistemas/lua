package main

import (
	"context"
	"edicion/idioma"
	"edicion/xerar"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"edicion/api"
	"edicion/modulo"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// App é o backend Go exposto ao frontend (frontend/src/main.js) mediante os
// bindings de Wails.
type App struct {
	settings Settings

	// bovedaTurno é a ruta da bóveda de Obsidian do turno do asistente en
	// curso (ver boveda.go); "" se o documento non ten bóveda.
	bovedaMu    sync.Mutex
	bovedaTurno string
	// bovedaTurnoID é o id da bóveda do turno na API (ver api_boveda.go).
	bovedaTurnoID string

	// api é a API de comunicacións local (ver api_boveda.go); arrinca a
	// primeira vez que se necesita.
	apiMu      sync.Mutex
	api        *api.Servidor
	apiCliente *api.Cliente
	// buxo leva o estado do módulo buxo (ver buxo_modulo.go).
	buxo     *modulo.Xestor
	buxoOnce sync.Once

	// bovedasAPI: bóvedas que un módulo rexistrou na API (id -> ruta).
	bovedasAPI map[string]string

	// idiomaPiztu é o código de idioma (gl/es/en/pt) que o profesorado
	// escolleu no propio Piztu, aprendido de PIZTU_CONTEXTO (contexto.go) -
	// "" se Lua correu sen ese contexto (instalación solta, sen Piztu). É só
	// un valor POR DEFECTO para o idioma de Lua cando settings.Idioma aínda
	// está baleiro - ver IdiomaPiztu() e frontend/src/main.js.
	idiomaPiztu string

	// actualizacion é a última foto de se hai unha versión nova de Lua
	// publicada en GitHub (ver comprobarActualizacionLua, actualizacion.go).
	// Consultada en fondo ao arrincar; o botón "i" da barra pinta fondo azul
	// se hai actualización (evento actualizacion_lua_cambiada).
	actualizacionMu sync.Mutex
	actualizacion   EstadoActualizacionLua

	// resultado é o último informe xerado: alimenta a xanela de resultado
	// desacoplada (ver xanelaresultado.go).
	resultadoMu sync.Mutex
	resultado   resultadoPublicado

	// tarefas leva a conta das xanelas de edición de tarefa abertas (ver
	// xanelatarefa.go).
	tarefas estadoXanelasTarefa
	// plantillas leva a conta das xanelas do editor de plantillas abertas (ver
	// xanelaplantilla.go).
	plantillas estadoXanelasTarefa
	// formularios leva a conta das xanelas do deseñador de formularios abertas
	// (ver xanelaformulario.go).
	formularios estadoXanelasTarefa

	// problemas é o motor que calcula os problemas mentres se editan (ver
	// maxima.go): leva unha sesión de Maxima aberta para que cada recálculo
	// non pague o arranque.
	problemasMu sync.Mutex
	problemas   *xerar.Motor
	problemasDe string // a ruta de Maxima coa que se abriu ("" = sen Maxima)
}

func NewApp() *App {
	return &App{}
}

// startup mantense coma alias fino para os tests, que chaman a.startup(...)
// directamente sen pasar por todo o ciclo de vida de Wails - deliberadamente
// NON toca a rede (ver ServiceStartup, que si o fai): un test que chame só
// startup() non debe disparar chamadas HTTP de fondo.
func (a *App) startup(ctx context.Context) {
	a.settings = loadSettings()
	if pctx, ok := lerContextoLanzamento(); ok {
		a.idiomaPiztu = pctx.Idioma
	}
	a.aplicarIdiomaGo()
}

// ServiceStartup é o hook de ciclo de vida que Wails v3 chama unha soa vez
// ao arrincar (interface application.ServiceStartup).
func (a *App) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	// En Windows, o PATH que herda Lua ao arrincar (dende un acceso
	// directo/Explorer) é unha instantánea de cando comezou a sesión de
	// escritorio - se MiKTeX/Pandoc se instalaron DESPOIS diso (ex.: co seu
	// propio instalador, non co botón de Lua), Lua seguiría sen velos ata
	// pechar sesión. Refrescar aquí, antes de calquera detección
	// (CheckLatexDeps/CheckDocDeps), evita ese falso "non atopado" - ver
	// pathrefresh_windows.go. No-op noutras plataformas.
	refrescarPathDendeRexistro()

	a.startup(ctx)
	// Os adxuntos do asistente son temporais: os da sesión anterior non serven.
	_ = limparAdxuntosDaSesion()
	go a.comprobarActualizacionLua()
	go a.vixiarBuxo() // buxo: descarga/actualiza se está activo (ao arrincar e cada hora)
	return nil
}

// IdiomaPiztu devolve o código de idioma (gl/es/en/pt) que o profesorado ten
// escollido en ⚙ Aula do propio Piztu, ou "" se Lua arrincou sen ese
// contexto (executable á man, sen Piztu instalado). O frontend só o usa
// coma valor por defecto cando aínda non se escolleu un idioma propio en
// Opcións (Settings.Idioma == "") - unha elección xa feita en Lua sempre
// gaña, ver frontend/src/main.js.
func (a *App) IdiomaPiztu() string {
	return a.idiomaPiztu
}

// comprobarActualizacionLua consulta as Releases de piztusistemas/lua en
// GitHub (ver actualizacion.go) e avisa o frontend (evento
// actualizacion_lua_cambiada) para que o botón "i" amose fondo azul.
func (a *App) comprobarActualizacionLua() {
	estado := ComprobarActualizacionLua()
	a.actualizacionMu.Lock()
	a.actualizacion = estado
	a.actualizacionMu.Unlock()
	// application.Get() só é non-nil cando application.New()/Run() xa
	// correu polo ciclo de vida real de Wails - nun test que chama
	// a.startup(context.Background()) directamente segue sendo nil, e Emit
	// sobre unha app inexistente petaría.
	if app := application.Get(); app != nil {
		app.Event.Emit("actualizacion_lua_cambiada", estado)
	}
}

// GetActualizacionLua devolve a última foto de se hai actualización de Lua
// dispoñible, sen tocar a rede (← diálogo de información ao abrirse).
func (a *App) GetActualizacionLua() EstadoActualizacionLua {
	a.actualizacionMu.Lock()
	defer a.actualizacionMu.Unlock()
	return a.actualizacion
}

// AplicarActualizacionLua descarga a última versión de Lua, substitúe o
// executable en execución e arrinca xa a nova copia (← botón "Actualizar"
// no diálogo de información). O frontend é quen peche esta xanela despois.
func (a *App) AplicarActualizacionLua() error {
	estado := a.GetActualizacionLua()
	if !estado.Disponible {
		return idioma.Errorf("go.act.ningunha", "non hai ningunha actualización dispoñible")
	}
	if err := AplicarActualizacionLuaEnCaliente(estado.URLDescarga, estado.URLChecksum, estado.VersionRemota); err != nil {
		return err
	}
	return ReiniciarLua()
}

// splitDialogDefault separa un "nome suxerido" para un diálogo de gardar en
// directorio + nome de ficheiro - necesario porque DefaultFilename NON
// acepta unha ruta completa: en Linux mapea directamente a
// gtk_file_chooser_set_current_name(), que a documentación de GTK di
// explicitamente que espera só un nome de ficheiro, sen barras - pasarlle
// unha ruta completa rompe o selector (amosaba/tentaba gardar coa ruta
// completa coma se fose o nome, en vez de abrir nesa carpeta cun nome normal
// para editar). O directorio vai en DefaultDirectory (se aínda existe - se
// non, déixase baleiro para que o selector use o seu propio por defecto en
// vez de fallar enteiro, que é o que fai o diálogo se DefaultDirectory non
// existe).
func splitDialogDefault(name string) (dir, base string) {
	base = filepath.Base(name)
	d := filepath.Dir(name)
	if d == "" || d == "." {
		return "", base
	}
	if info, err := os.Stat(d); err != nil || !info.IsDir() {
		return "", base
	}
	return d, base
}

// saveFileDialogCompat constrúe un diálogo "gardar": dir só se fixa se non
// está baleiro (ver splitDialogDefault). Usado por todas as exportacións
// (SavePDFDialog/SaveTexDialog en latexdoc.go, informe_cronograma.go).
func saveFileDialogCompat(title, dir, base, filterName, filterPattern string) (string, error) {
	d := application.Get().Dialog.SaveFile().
		SetMessage(title).
		SetFilename(base).
		AddFilter(filterName, filterPattern)
	if dir != "" {
		d = d.SetDirectory(dir)
	}
	return d.PromptForSingleSelection()
}

// SaveFile escribe content nunha ruta xa coñecida (Ctrl+S sobre un ficheiro
// que xa se abriu ou se gardou unha vez).
func (a *App) SaveFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// dedupeFileName devolve un nome de ficheiro seguro para escribir en dir:
// se fileName xa existe aí, engade "-1", "-2"... antes da extensión ata
// atopar un que non colida. fallback substitúe un nome baleiro (fileName sen
// parte antes da extensión, p.ex. ".png" solto).
func dedupeFileName(dir, fileName, fallback string) string {
	ext := filepath.Ext(fileName)
	base := strings.TrimSuffix(filepath.Base(fileName), ext)
	if base == "" {
		base = fallback
	}
	name := base + ext
	for i := 1; fileExists(filepath.Join(dir, name)); i++ {
		name = fmt.Sprintf("%s-%d%s", base, i, ext)
	}
	return name
}
