package main

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Licenza de Buxo (MIT) e avisos de terceiros (MarkItDown e os paquetes que leva
// o conversor): van dentro do executable para poder consultalos desde a
// interface, e tamén no zip do módulo.
//
//go:embed LICENSE
var textoLicenza string

//go:embed THIRD_PARTY_LICENSES.txt
var textoAvisosTerceiros string

// Eventos que recibe o frontend.
const (
	eventoCola     = "buxo:cola"
	eventoRexistro = "buxo:rexistro"
	eventoEstado   = "buxo:estado"
	eventoTarefas  = "buxo:tarefas"
)

func init() {
	application.RegisterEvent[[]Item](eventoCola)
	application.RegisterEvent[Liña](eventoRexistro)
	application.RegisterEvent[EstadoApp](eventoEstado)
	application.RegisterEvent[[]Tarefa](eventoTarefas)
}

// Argumentos co que a aplicación anfitrioa (Lua) lanza Buxo.
type Argumentos struct {
	Boveda  string // cartafol da bóveda do proxecto
	Materia string // tema do proxecto (para o esquema e os prompts da IA)
	ID      string // id da bóveda na API da anfitrioa
}

// EstadoApp é o que ve a interface do estado de Buxo.
type EstadoApp struct {
	Boveda        string `json:"boveda"`
	Materia       string `json:"materia"`
	ModoAPI       bool   `json:"modoApi"`       // hai API da aplicación anfitrioa
	IAConfigurada bool   `json:"iaConfigurada"` // e ten unha IA lista
	Modelo        string `json:"modelo"`
	Erro          string `json:"erro,omitempty"`
	Construindo   bool   `json:"construindo"`
	Fontes        int    `json:"fontes"` // .md en raw/
}

// App é o servizo que expón Buxo ao frontend.
type App struct {
	args Argumentos

	mestre *Mestre
	log    *Rexistro
	ia     *ClienteIA

	mu          sync.Mutex
	iaEstado    EstadoIA
	construindo bool
	cancelar    context.CancelFunc
}

func NewApp(args Argumentos) *App {
	return &App{args: args}
}

func emitir(nome string, datos any) {
	if app := application.Get(); app != nil {
		app.Event.Emit(nome, datos)
	}
}

// ServiceStartup arrinca o mestre e consulta a IA da anfitrioa.
func (a *App) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	if a.args.Boveda == "" {
		return nil
	}
	a.log = NovoRexistro(filepathJoin(a.args.Boveda, ficheiroLog), func(l Liña) { emitir(eventoRexistro, l) })
	a.mestre = NovoMestre(a.args.Boveda, a.args.Materia, a.log, func() { emitir(eventoCola, a.mestre.Items()); emitir(eventoEstado, a.Estado()) })
	a.mestre.cambioTarefas = func() { emitir(eventoTarefas, a.mestre.Tarefas()) }
	a.ia = IADeEntorno(a.args.ID)
	a.mestre.ia = a.ia
	a.refrescarIA(ctx)
	a.mestre.Recuperar()
	return nil
}

func (a *App) ServiceShutdown() error {
	a.mu.Lock()
	c := a.cancelar
	a.mu.Unlock()
	if c != nil {
		c()
	}
	if a.mestre != nil {
		a.mestre.Pechar()
	}
	return nil
}

func (a *App) refrescarIA(ctx context.Context) {
	if a.ia == nil {
		return
	}
	e := a.ia.Estado(ctx)
	a.mu.Lock()
	a.iaEstado = e
	a.mu.Unlock()
}

// Estado devolve o estado actual.
func (a *App) Estado() EstadoApp {
	a.mu.Lock()
	defer a.mu.Unlock()
	e := EstadoApp{
		Boveda: a.args.Boveda, Materia: a.args.Materia, ModoAPI: a.ia != nil,
		IAConfigurada: a.iaEstado.Configurada, Modelo: a.iaEstado.Modelo, Erro: a.iaEstado.Erro,
		Construindo: a.construindo,
	}
	switch {
	case a.args.Boveda == "":
		e.Erro = "Buxo ten que abrirse desde un proxecto de Lua (falta o cartafol da bóveda)."
	case a.ia == nil:
		e.Erro = "Buxo non está conectado a Lua: ábreo desde Lua."
	}
	if a.args.Boveda != "" {
		e.Fontes = fontesRaw(a.args.Boveda)
	}
	return e
}

// Cola devolve os ficheiros a tratar.
func (a *App) Cola() []Item {
	if a.mestre == nil {
		return []Item{}
	}
	return a.mestre.Items()
}

// Tarefas devolve o progreso da última construción.
func (a *App) Tarefas() []Tarefa {
	if a.mestre == nil {
		return []Tarefa{}
	}
	return a.mestre.Tarefas()
}

// Historial devolve o rexistro acumulado (ao abrir a xanela).
func (a *App) Historial() []Liña {
	if a.log == nil {
		return []Liña{}
	}
	return a.log.Historial()
}

// CambiarModo fixa como se converten os ficheiros que se engadan (auto,
// vision ou texto; ver formatos.go).
func (a *App) CambiarModo(modo string) {
	if a.mestre != nil {
		a.mestre.CambiarModo(modo)
	}
}

// EscollerPDFs abre o selector de ficheiros e engade á cola os PDF, imaxes,
// Markdown e documentos de oficina escollidos.
func (a *App) EscollerPDFs() (int, error) {
	if a.mestre == nil {
		return 0, fmt.Errorf("non hai bóveda")
	}
	dialogo := application.Get().Dialog.OpenFile()
	// Ancorado á xanela (en macOS, coma folla): buxo é un executable solto
	// lanzado por Lua, non un .app, e un panel sen xanela pai créase pero
	// queda fóra da pantalla — o botón parecía non facer nada.
	if w, ok := application.Get().Window.GetByName("buxo"); ok {
		dialogo = dialogo.AttachToWindow(w)
	}
	rutas, err := dialogo.
		SetTitle("Escolle ficheiros").
		AddFilter("PDF, imaxes, Markdown e documentos", patronsAceptados()).
		AddFilter("Ficheiros PDF (*.pdf)", "*.pdf;*.PDF").
		PromptForMultipleSelection()
	if err != nil || len(rutas) == 0 {
		return 0, err
	}
	a.mestre.EngadirLocais(rutas)
	return len(rutas), nil
}

// EngadirURLs engade URL (unha por liña ou separadas por espazos): as que son
// un PDF descárganse; nas demais a IA procura PDF, filtrándoos segundo a
// instrución (opcional) que escribiu o usuario.
func (a *App) EngadirURLs(texto, instrucion string) error {
	if a.mestre == nil {
		return fmt.Errorf("non hai bóveda")
	}
	urls := strings.Fields(texto)
	if len(urls) == 0 {
		return fmt.Errorf("escribe polo menos unha URL")
	}
	go a.mestre.ProcurarPDFs(context.Background(), a.ia, urls, instrucion)
	return nil
}

// Quitar saca un ficheiro da lista.
func (a *App) Quitar(id int) {
	if a.mestre != nil {
		a.mestre.Quitar(id)
	}
}

// Construir lanza (en fondo) a construción da wiki coa IA.
func (a *App) Construir() error {
	if a.mestre == nil {
		return fmt.Errorf("non hai bóveda")
	}
	a.mu.Lock()
	if a.construindo {
		a.mu.Unlock()
		return fmt.Errorf("a bóveda xa se está construíndo")
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.construindo, a.cancelar = true, cancel
	a.mu.Unlock()
	emitir(eventoEstado, a.Estado())
	go func() {
		if err := a.mestre.Construir(ctx, a.ia); err != nil && ctx.Err() == nil {
			a.log.Erro("rex.erro", "%v", err)
		}
		a.mu.Lock()
		a.construindo, a.cancelar = false, nil
		a.mu.Unlock()
		cancel()
		emitir(eventoEstado, a.Estado())
	}()
	return nil
}

// Parar cancela a construción en curso.
func (a *App) Parar() {
	a.mu.Lock()
	c := a.cancelar
	a.mu.Unlock()
	if c != nil {
		c()
	}
}

// LicenzaTexto devolve a licenza de Buxo e os avisos de terceiros, para amosalos.
func (a *App) LicenzaTexto() string {
	return strings.TrimSpace(textoLicenza) + "\n\n" + strings.Repeat("=", 60) + "\n\n" + strings.TrimSpace(textoAvisosTerceiros) + "\n"
}
