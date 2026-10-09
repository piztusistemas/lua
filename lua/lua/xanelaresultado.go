package main

import (
	"errors"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

var errXeraAntes = errors.New("xera antes de exportar")

// xanelaresultado.go implementa o "modo dúas xanelas": ademais da xanela
// única integrada (editor + resultado, o modo por defecto e o único que
// arrinca así), pódese DESACOPLAR o resultado (botón "Desacoplar" na cabeceira
// da previsualización) para telo nunha xanela á parte (Imprimir/PDF/TeX/MD/
// DOCX/ODT), arrastrable a un segundo monitor, e ACOPLALO de volta cando se
// queira (botón "Acoplar" nesa xanela, ou a X do xestor de xanelas) -
// decisión de sesión, non un axuste persistente (Lua arrinca sempre en
// xanela única).
//
// DESEÑO (Wails v3): permite varias WebviewWindow no MESMO proceso (ver
// AbrirXanelaResultado). O último informe xerado vive en App.resultado
// (protexido por resultadoMu): cada XerarInformeCronograma correcto chama a
// publicarResultado, que o garda e avisa coma evento
// (lua:resultado-actualizado) para que a xanela de resultado, se está
// aberta, se repinte; ao abrirse, esa xanela pídeo con GetEstadoResultado.
// As exportacións dende a xanela de resultado (AccionResultado*) usan a MESMA
// petición e a MESMA plantilla coas que se xerou o resultado, non o que haxa
// agora no editor - así o que se exporta é o que se está a ver.
const nomeXanelaResultado = "results"

// resultadoPublicado é o último informe xerado con éxito.
type resultadoPublicado struct {
	Hai        bool
	Version    int
	PageImages []string
	Warnings   []string
	PDFBase64  string
	TexSource  string
	Req        XerarInformeCronogramaRequest
	Plantilla  *PlantillaLatex
}

// snapshotResultado é o que ve o frontend da xanela de resultado.
type snapshotResultado struct {
	Version    int      `json:"version"`
	PageImages []string `json:"pageImages"`
	Warnings   []string `json:"warnings"`
	HasPDF     bool     `json:"hasPdf"`
	HasTex     bool     `json:"hasTex"`
}

func (r resultadoPublicado) snapshot() snapshotResultado {
	return snapshotResultado{
		Version: r.Version, PageImages: r.PageImages, Warnings: r.Warnings,
		HasPDF: r.PDFBase64 != "", HasTex: r.TexSource != "",
	}
}

// publicarResultado garda o informe recén xerado e avisa á xanela de
// resultado. Barato e inofensivo aínda sen modo dúas xanelas activo (o
// evento simplemente non ten quen o escoite).
func (a *App) publicarResultado(req XerarInformeCronogramaRequest, plantilla *PlantillaLatex, r GeneratePDFResult) {
	a.resultadoMu.Lock()
	a.resultado = resultadoPublicado{
		Hai: true, Version: a.resultado.Version + 1,
		PageImages: r.PageImages, Warnings: r.Warnings,
		PDFBase64: r.PDFBase64, TexSource: r.LatexSource,
		Req: req, Plantilla: plantilla,
	}
	snap := a.resultado.snapshot()
	a.resultadoMu.Unlock()
	// application.Get() é nil nun test que chama este binding directamente
	// sen pasar por todo o ciclo de vida de Wails - Emit sobre unha app
	// inexistente petaría.
	if app := application.Get(); app != nil {
		app.Event.Emit("lua:resultado-actualizado", snap)
	}
}

// GetEstadoResultado é o que chama a xanela de resultado ao montar
// (hidratación inicial).
func (a *App) GetEstadoResultado() snapshotResultado {
	a.resultadoMu.Lock()
	defer a.resultadoMu.Unlock()
	return a.resultado.snapshot()
}

// resultadoActual devolve unha copia do último informe, ou errXeraAntes se
// aínda non se xerou ningún.
func (a *App) resultadoActual() (resultadoPublicado, error) {
	a.resultadoMu.Lock()
	defer a.resultadoMu.Unlock()
	if !a.resultado.Hai {
		return resultadoPublicado{}, errXeraAntes
	}
	return a.resultado, nil
}

// accionResultado é a resposta común ás AccionResultado* - warnings sempre
// que o backend as dea (ex. imaxe non atopada), path só se de verdade se
// gardou algo (pódese cancelar o diálogo "Gardar como").
type accionResultado struct {
	Path     string   `json:"path"`
	Warnings []string `json:"warnings"`
}

// nomeBaseResultado é o nome (sen extensión) que se propón ao exportar.
func nomeBaseResultado(r resultadoPublicado) string {
	if n := r.Req.NomeBase; n != "" {
		return n
	}
	return nomeBaseCronoPorDefecto
}

// AccionResultadoPDF/Tex len os bytes xa xerados - non recompilan.
func (a *App) AccionResultadoPDF() (accionResultado, error) {
	r, err := a.resultadoActual()
	if err != nil {
		return accionResultado{}, err
	}
	path, err := a.SavePDFDialog(nomeBaseResultado(r)+".pdf", r.PDFBase64)
	return accionResultado{Path: path}, err
}

// AccionResultadoTex compón de novo o .tex (e escribe as imaxes a carón), en
// vez de gardar o texto xa xerado: o ficheiro exportado ten que compilar
// solto, e iso esixe as imaxes do logotipo e das tarefas no mesmo cartafol.
func (a *App) AccionResultadoTex() (accionResultado, error) {
	r, err := a.resultadoActual()
	if err != nil {
		return accionResultado{}, err
	}
	res, err := a.exportarTexCrono(r.Req, r.Plantilla, nomeBaseResultado(r)+".tex")
	return accionResultado{Path: res.Path, Warnings: res.Warnings}, err
}

func (a *App) AccionResultadoMD() (accionResultado, error) {
	r, err := a.resultadoActual()
	if err != nil {
		return accionResultado{}, err
	}
	res, err := a.exportarMarkdownCrono(r.Req, r.Plantilla, nomeBaseResultado(r)+".md")
	return accionResultado{Path: res.Path, Warnings: res.Warnings}, err
}

func (a *App) AccionResultadoDocx() (accionResultado, error) {
	r, err := a.resultadoActual()
	if err != nil {
		return accionResultado{}, err
	}
	res, err := a.exportarDocxCrono(r.Req, r.Plantilla, nomeBaseResultado(r)+".docx")
	return accionResultado{Path: res.Path, Warnings: res.Warnings}, err
}

func (a *App) AccionResultadoOdt() (accionResultado, error) {
	r, err := a.resultadoActual()
	if err != nil {
		return accionResultado{}, err
	}
	res, err := a.exportarOdtCrono(r.Req, r.Plantilla, nomeBaseResultado(r)+".odt")
	return accionResultado{Path: res.Path, Warnings: res.Warnings}, err
}

// AbrirXanelaResultado (re)abre a xanela de resultado coma toplevel
// independente - dous WebviewWindow do MESMO proceso, iso é o que permite
// v3. Idempotente: se xa está aberta, só a trae ao foco. Chamado dende o
// botón "Desacoplar" da xanela principal, e tamén dispoñible coma botón
// manual "Reabrir xanela de resultado" por se o profesorado a pechou sen
// querer.
func (a *App) AbrirXanelaResultado() error {
	app := application.Get()
	if w, ok := app.Window.GetByName(nomeXanelaResultado); ok {
		w.Show()
		w.Focus()
		return nil
	}

	// A xanela de resultado NON arrinca maximizada: o obxectivo do modo
	// dúas xanelas é ter as dúas visibles á vez (nun segundo monitor, ou a
	// carón da outra) - se arrincase maximizada tapando a principal, o
	// profesorado tería que desmaximizala igualmente antes de poder
	// arrastrala a onde lle interese.
	w := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             nomeXanelaResultado,
		Title:            "Lua — Resultado",
		URL:              "/?resultado=1",
		Width:            900,
		Height:           700,
		BackgroundColour: application.NewRGB(23, 24, 28),
		// HiddenOnTaskbar: equivalente en Windows ao truco GTK de
		// edicion/ui/xanela_gtk_linux.go/ocultarXanelaDaBarraDeTarefas - Wails
		// xa o implementa internamente (WS_EX_TOOLWINDOW), non fai falla
		// código propio. Ignorado sen efecto en Linux/macOS (ver
		// edicion/ui/xanela_other.go).
		Windows: application.WindowsWindow{HiddenOnTaskbar: true},
		// ZoomControlEnabled: en Windows delega directamente no propio
		// WebView2 (CoreWebView2Settings.IsZoomControlEnabled) - dálle á
		// xanela de resultado o mesmo Ctrl+/Ctrl-/Ctrl+0/Ctrl+roda que
		// calquera páxina en Edge/Chrome, sen código propio ningún.
		ZoomControlEnabled: true,
		// Minimise/MaximiseButtonState: por defecto deberían amosarse
		// (ButtonEnabled é o valor cero), pero un real report en Windows
		// atopou só o botón de pechar na xanela desacoplada - fixándoos
		// explícitos evita depender de calquera comportamento por defecto
		// da plataforma nesta versión beta de Wails v3.
		MinimiseButtonState: application.ButtonEnabled,
		MaximiseButtonState: application.ButtonEnabled,
	})

	// Peche (botón "Acoplar" na xanela de resultado, OU a X do xestor de
	// xanelas - mesmo camiño de volta para os dous) -> avisar ao editor.
	w.OnWindowEvent(events.Common.WindowClosing, func(_ *application.WindowEvent) {
		application.Get().Event.Emit("lua:resultado-pechado")
	})

	// Fóra da barra de tarefas - ver edicion/ui/xanela_gtk_linux.go. Sen isto,
	// o modo dúas xanelas amosaría dous "aplicativos" distintos na barra, moi
	// incómodo; a xanela de resultado é auxiliar (coma unha paleta
	// flotante), non precisa a súa propia entrada.
	ocultarXanelaDaBarraDeTarefas(w)

	return nil
}

// DockResultado é o binding que chama o botón "Acoplar" da xanela de
// resultado - simplemente pecha o toplevel; OnWindowEvent(WindowClosing)
// enriba xa fai o resto (emitir lua:resultado-pechado). Un só camiño de
// retorno, tanto para o botón coma para a X do xestor de xanelas.
func (a *App) DockResultado() {
	if w, ok := application.Get().Window.GetByName(nomeXanelaResultado); ok {
		w.Close()
	}
}

func (a *App) XanelaResultadoAberta() bool {
	_, ok := application.Get().Window.GetByName(nomeXanelaResultado)
	return ok
}

func init() {
	application.RegisterEvent[snapshotResultado]("lua:resultado-actualizado")
	// application.Void (non "any") é o tipo correcto para un evento sen
	// datos - con "any", Emit sen dato provoca un nil pointer dereference
	// interno que mata toda a app.
	application.RegisterEvent[application.Void]("lua:resultado-pechado")
}
