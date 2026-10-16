import {
  AbrirFicheiro,
  GetSettings, SaveSettings,
  CheckLatexDeps, InstallLatex,
  SavePDFDialog,
  AbrirXanelaResultado, GetEstadoResultado, DockResultado,
  IAAsistenteEstado, IAFerramentas, IAFerramentasDoAmbito, IATurnoAsistente, IAChamarFerramenta,
  AdxuntoGardar, AdxuntoEliminar, AdxuntosLimpar, AdxuntoVista, AdxuntoDataURI,
  BovedaEstadoDe, BovedaProxectoConectar, BovedaConectar, BovedaEscoller, IAFerramentasBoveda,
  AbrirXanelaTarefa, GetDatosXanelaTarefa, AplicarTarefaXanela, MarcarXanelaTarefaSucia, PecharXanelaTarefa,
  AbrirXanelaPlantilla, GetDatosXanelaPlantilla, PlantillaXanelaAplicada, MarcarXanelaPlantillaSucia, PecharXanelaPlantilla,
  PrevisualizarPlantillaOfimaticaDe,
  AccionResultadoPDF, AccionResultadoTex, AccionResultadoMD, AccionResultadoDocx, AccionResultadoOdt,
  GetActualizacionLua, AplicarActualizacionLua,
  IdiomaPiztu,
  ListarPlantillas, FragmentosInformePorDefecto, GardarPlantilla, EliminarPlantilla, DuplicarPlantilla,
  EscollerPlantillaDocumento, PlantillaDoDocumento, SubirImaxePlantilla, PlantillaImaxeVista, EliminarImaxePlantilla,
  PrevisualizarPlantillaDe, PrevisualizarPlantillaDocumento, PrevisualizarPlantillaOfimaticaDocumento, PrevisualizarPlantilla,
  PaqueteLatexQueFalta, InstalarPaqueteLatex,
  ListarFontesSistema,
  CombinacionsLetras, CoresDaImaxe, ImportarDesenoDeFicheiro, EstiloElementoDeseno, SkillDeseno,
  CancelarVistaPrevia, ComprobarElementoDeseno, GardarDesenoFicheiro, LerDesenoFicheiro, AplicarDesenoATodas,
  XerarInformeCronograma, ExportarInformeTex, ExportarInformeMarkdown, ExportarInformeDocx, ExportarInformeOdt,
  GardarBorradorInforme, LerBorradorInforme,
  ListarFormularios, GardarFormulario, EliminarFormulario, CambiarModoFormulario,
  LerBibliotecaFormularios, GardarBibliotecaFormularios, CrearCarpetaFormularios, GardarDescricionFormulario, GardarIconaFormulario,
  ArboreTraslado, SacarTenDatosProducion, SacarTraslado, SacarDocumentoJSON, EscollerFicheiroMeter, EscollerCartafolMeter,
  AnalizarMeter, AplicarMeter, DesfacerTraslado, MigracionTraballoSolto,
  BuxoEstadoActual, BuxoActivar, BuxoActualizar,
  EscollerCartafolProxecto, MoverProxecto, FixarCorProxecto,
  ListarProxectos, ListarFormulariosBiblioteca, CrearProxecto, ActivarProxecto, ProxectoActivo, QuitarDoProxecto, EliminarProxecto, AbrirBuxoDoProxecto,
  EngadirTraballoBoveda, BovedaAutomaticaDe, FixarBovedaAutomatica,
  CatalogoFormularios, CrearCarpetaCatalogo, RenomearCarpetaCatalogo, EliminarCarpetaCatalogo, MoverFormularioCatalogo, ReutilizarNoProxecto,
  CatalogoFluxo, FluxoAutomaticoDe, SincronizarFluxoFormulario, AvaliarFluxoFormulario,
  AbrirXanelaFormulario, GetDatosXanelaFormulario, FormularioXanelaGardado, FormularioXanelaEliminado, MarcarXanelaFormularioSucia, PecharXanelaFormulario,
  CalcularProblemas, ResolverProblema, CheckMaxima, DetectMaxima, InstallMaxima,
} from '../bindings/lua-wails/app.js'
import { Application, Events, Window } from '@wailsio/runtime'
import { haiProblemas, xeracionDe } from './formularios/problemas-modelo.js'
import { abrirBibliotecaPlantillas } from './plantillas.js'
import { abrirTrasladar } from './trasladar/trasladar.js'
import { montarEditorPlantilla } from './plantillas-editor.js'
import { makeDraggable } from './draggable.js'
import { crearProgreso } from './progreso.js'
import { confirmar, preguntar } from './comun/dialogs.js'
import { createInformes } from './informes/controlador.js'
import { FERRAMENTA_CRONOGRAMA } from './informes/ferramenta-cronograma.js'
import { FERRAMENTA_ACTAS } from './informes/ferramenta-actas.js'
import { crearFerramentaFormulario } from './formularios/ferramenta.js'
import { abrirDisenador } from './formularios/disenador.js'
import { svgIcona, coresFormularios, valorCor, aparenciaLibre } from './formularios/iconas.js'
import { haiContidoFormulario, novoEsquema } from './formularios/modelo.js'
import { abrirBibliotecaFormularios } from './formularios/biblioteca.js'
import { crearInicio, corProxecto } from './formularios/proxectos.js'
import { abrirCatalogo } from './formularios/catalogo.js'
import { crearColumna } from './formularios/columna.js'
import { escollerIcona } from './formularios/escoller-icona.js'
import { bibliotecaBaleira, completar, columna as columnaDe, agochar, podeBorrar } from './formularios/biblioteca-modelo.js'
import { montarEditorTarefa, normalizarImaxe } from './informes/tarefa-editor.js'
import { calendario, totalTarefas } from './informes/cronograma-modelo.js'
import { totalActas, totalSeguimentos } from './informes/actas-modelo.js'
import { crearFerramentas } from './asistente/ferramentas.js'
import { crearPanelAsistente } from './asistente/panel.js'
import { logoProvedor } from './asistente/logos.js'
import glDict from '../../idiomas/gl.json'
import esDict from '../../idiomas/es.json'
import enDict from '../../idiomas/en.json'
import ptDict from '../../idiomas/pt.json'

// ── Idioma ───────────────────────────────────────────────────────────────
// gl.json é a fonte da verdade (todas as claves); es/en/pt poden estar
// incompletos (ou baleiros de todo, ver lua/idiomas/README.md) - I18N
// mestura sempre gl por baixo, así calquera clave sen traducir aínda amosa
// o galego en vez dun oco ou da propia clave crúa.
const DICTS = { gl: glDict, es: esDict, en: enDict, pt: ptDict }
let idiomaActual = 'gl'
let I18N = { ...glDict }
// idiomaPiztu: código que Piztu ten escollido (IdiomaPiztu(), ver app.go),
// "" se Lua arrincou sen contexto de Piztu (executable á man). Só un valor
// por defecto — resolto cedo no arranque, antes de que Opcións poida
// abrirse, e usado en openOptions/saveOptions para distinguir "segue a
// Piztu" (idioma: "" en Settings) de "elección explícita".
let idiomaPiztu = ''

function cargarIdioma(code) {
  idiomaActual = DICTS[code] ? code : 'gl'
  I18N = { ...glDict, ...(DICTS[idiomaActual] || {}) }
}

// t busca `clave` no dicionario activo; `fallback` (normalmente o propio
// texto galego xa escrito no HTML/JS) cobre o caso de arrincar antes de
// cargarIdioma() resolver, ou unha clave que nin sequera exista en gl.json
// por un despiste. `vars` interpola {nome} -> valor.
function t(clave, fallback, vars) {
  let val = I18N[clave] ?? fallback ?? clave
  if (vars) {
    for (const k in vars) val = val.replaceAll(`{${k}}`, vars[k])
  }
  return val
}

// aplicarTraducions percorre o DOM estático (index.html) e substitúe cada
// texto marcado por data-i18n* polo valor do idioma activo. O texto galego
// que xa hai escrito en cada elemento serve de fallback (t() cae nel se
// falta a clave), así que non hai "oco" mentres carga nin se falla algo.
function aplicarTraducions() {
  document.querySelectorAll('[data-i18n]').forEach((el) => {
    el.textContent = t(el.dataset.i18n, el.textContent)
  })
  document.querySelectorAll('[data-i18n-title]').forEach((el) => {
    el.title = t(el.dataset.i18nTitle, el.title)
  })
  document.querySelectorAll('[data-i18n-placeholder]').forEach((el) => {
    el.placeholder = t(el.dataset.i18nPlaceholder, el.placeholder)
  })
  document.querySelectorAll('[data-i18n-aria-label]').forEach((el) => {
    el.setAttribute('aria-label', t(el.dataset.i18nAriaLabel, el.getAttribute('aria-label')))
  })
  document.querySelectorAll('[data-i18n-html]').forEach((el) => {
    el.innerHTML = t(el.dataset.i18nHtml, el.innerHTML)
  })
}

const preview = document.getElementById('preview')
const btnNew = document.getElementById('btnNew')
const btnTrasladar = document.getElementById('btnTrasladar')
const btnUndo = document.getElementById('btnUndo')
const btnRedo = document.getElementById('btnRedo')
const btnRun = document.getElementById('btnRun')
const plantillaSelect = document.getElementById('plantillaSelect')
const btnPlantillas = document.getElementById('btnPlantillas')
const btnAxustesTexto = document.getElementById('btnAxustesTexto')
const btnPrint = document.getElementById('btnPrint')
const btnPDF = document.getElementById('btnPDF')
const btnExportTex = document.getElementById('btnExportTex')
const btnExportMarkdown = document.getElementById('btnExportMarkdown')
const btnExportDocx = document.getElementById('btnExportDocx')
const btnExportOdt = document.getElementById('btnExportOdt')
const btnBoveda = document.getElementById('btnBoveda')
const chkBovedaSempre = document.getElementById('chkBovedaSempre')
const bovedaSempreFila = document.getElementById('bovedaSempreFila')
const btnOptions = document.getElementById('btnOptions')
const btnAbout = document.getElementById('btnAbout')
const sobreOverlay = document.getElementById('sobreOverlay')
const btnAboutClose = document.getElementById('btnAboutClose')
const sobreVersion = document.getElementById('sobreVersion')
const sobreActualizacion = document.getElementById('sobreActualizacion')
const btnActualizarLua = document.getElementById('btnActualizarLua')
const statusEl = document.getElementById('status')
const btnAbrirGardado = document.getElementById('btnAbrirGardado')
const warningsEl = document.getElementById('warnings')

const optionsOverlay = document.getElementById('optionsOverlay')
const optIAPreset = document.getElementById('optIAPreset')
const optIAApiKey = document.getElementById('optIAApiKey')
const optBuxoActivo = document.getElementById('optBuxoActivo')
const buxoEstadoEl = document.getElementById('buxoEstado')
const btnBuxoActualizar = document.getElementById('btnBuxoActualizar')
const optIAModel = document.getElementById('optIAModel')
const optIABaseURL = document.getElementById('optIABaseURL')
const optIAMaxTokens = document.getElementById('optIAMaxTokens')
const optIATimeout = document.getElementById('optIATimeout')
const optIABaseURLRow = document.getElementById('optIABaseURLRow')
const optLatexEngine = document.getElementById('optLatexEngine')
const optMaximaPath = document.getElementById('optMaximaPath')
const optMaximaDecimais = document.getElementById('optMaximaDecimais')
const optMaximaEstado = document.getElementById('optMaximaEstado')
const btnDetectarMaxima = document.getElementById('btnDetectarMaxima')
const btnInstalarMaxima = document.getElementById('btnInstalarMaxima')
const optXerarAoGardar = document.getElementById('optXerarAoGardar')
const optIdioma = document.getElementById('optIdioma')
const optExportPDF = document.getElementById('optExportPDF')
const optExportTex = document.getElementById('optExportTex')
const optExportMarkdown = document.getElementById('optExportMarkdown')
const optExportDocx = document.getElementById('optExportDocx')
const optExportOdt = document.getElementById('optExportOdt')
const btnOptionsCancel = document.getElementById('btnOptionsCancel')
const btnOptionsSave = document.getElementById('btnOptionsSave')

const latexWarning = document.getElementById('latexWarning')
const latexWarningText = document.getElementById('latexWarningText')
const btnInstallLatex = document.getElementById('btnInstallLatex')
const btnLatexManual = document.getElementById('btnLatexManual')
const btnLatexDismiss = document.getElementById('btnLatexDismiss')
const latexManualOverlay = document.getElementById('latexManualOverlay')
const btnLatexManualClose = document.getElementById('btnLatexManualClose')


// Modo dúas xanelas - ver xanelaresultado.go. panePreview/
// panePreviewPlaceholder son exclusivos da xanela EDITOR (activarModoDobreEditor
// agocha unha e amosa a outra cando o resultado vive noutra xanela);
// btnReabrirResultado vive dentro do placeholder.
const panePreview = document.querySelector('.pane-preview')
const panePreviewPlaceholder = document.querySelector('.pane-preview-placeholder')
const btnReabrirResultado = document.getElementById('btnReabrirResultado')
const btnDesacoplarResultado = document.getElementById('btnDesacoplarResultado')
// Tira co botón de "ancho completo": só visible co resultado desacoplado.
const tiraAnchoCompleto = document.getElementById('tiraAnchoCompleto')
const btnAnchoCompleto = document.getElementById('btnAnchoCompleto')

// paneEditor/paneSplitter: divisor arrastrable entre editor e resultado -
// ver initPaneSplitter, máis abaixo.
const paneEditor = document.querySelector('.pane-editor')
const paneSplitter = document.getElementById('paneSplitter')

// informes: o controlador da ferramenta ACTIVA (informes/controlador.js); é
// informesCrono ou informesActas, e cambia con cambiarFerramenta. Créanse máis
// abaixo, na xanela EDITOR (a de resultado non ten editor).
let informes = null
let informesCrono = null
let informesActas = null
// controladores: todas as ferramentas (cronograma, actas e un por cada
// formulario propio, con id «form:<id>»).
const controladores = new Map()
let ferramentaActiva = 'cronograma'
let proxectoAberto = ''
const CLAVE_FERRAMENTA = 'lua.ferramenta'
// As páxinas do último resultado de cada ferramenta: ao cambiar de ferramenta
// a previsualización volve amosar as súas.
const paxinasPorFerramenta = { cronograma: [], actas: [] } // e as dos formularios, por id
const ferramentasNav = document.getElementById('ferramentasNav')
const zonaFixaColumna = document.getElementById('ferramentasFixas')
const zonaRestoColumna = document.getElementById('ferramentasResto')
const contedorFerramenta = {
  cronograma: document.getElementById('informeEditor'),
  actas: document.getElementById('actasEditor'),
}
const ambitoWrap = document.getElementById('ambitoWrap')
const ambitoSelect = document.getElementById('ambitoSelect')
// previewZoom: nivel de zoom (Ctrl+/Ctrl-/Ctrl+0) das páxinas dentro do
// iframe de previsualización - ver pagesToPreviewHTML/zoomScript. Vive aquí
// (non dentro do iframe) para sobrevivir a unha re-xeración (Xerar substitúe
// todo o srcdoc, o script inxectado esquecería o nivel); o propio script
// avisa dos cambios por postMessage.
let previewZoom = 1
// xerarAoGardarActivo reflicte Opcións → "Xerar automaticamente ao gardar"
// (Settings.XerarAoGardar, settings.go) - true por defecto. Actualízase ao
// arrincar (GetSettings, ao final deste ficheiro) e cada vez que se garda
// Opcións (saveOptions).
let xerarAoGardarActivo = true
// souXanelaResultado: esta xanela é a de resultado desacoplada (?resultado=1,
// ver AbrirXanelaResultado en xanelaresultado.go), sen editor propio: as
// exportacións reenvíanse ao backend (accionRemota), que xa ten o último
// informe xerado.
const souXanelaResultado = new URLSearchParams(window.location.search).get('resultado') === '1'

// idXanelaTarefa: id da tarefa se ESTA xanela é a de edición dunha tarefa
// (?tarefa=<id>, ver AbrirXanelaTarefa en xanelatarefa.go), ou '' se non.
const idXanelaTarefa = new URLSearchParams(window.location.search).get('tarefa') || ''
const souXanelaTarefa = idXanelaTarefa !== ''

// idXanelaPlantilla: id da plantilla se ESTA xanela é a do editor de plantillas
// (?plantilla=<id|nova>, ver AbrirXanelaPlantilla en xanelaplantilla.go).
const idXanelaPlantilla = new URLSearchParams(window.location.search).get('plantilla') || ''
const souXanelaPlantilla = idXanelaPlantilla !== ''

// idXanelaFormulario: id do formulario se ESTA xanela é a do deseñador de formularios
// (?formulario=<id|novo>, ver AbrirXanelaFormulario en xanelaformulario.go).
const idXanelaFormulario = new URLSearchParams(window.location.search).get('formulario') || ''
const souXanelaFormulario = idXanelaFormulario !== ''

// ── Divisor arrastrable entre editor e resultado ─────────────────────────
// Só ten sentido en modo xanela única (con editor E resultado visibles á
// vez nesta mesma xanela) - en modo dúas xanelas, cada un vive na súa
// propia xanela OS, xa redimensionable arrastrando o propio bordo da
// xanela (ver entrarModoResultado/activarModoDobreEditor, que agochan
// #paneSplitter coma o resto do panel que xa non aplica).
const PANE_SPLITTER_KEY = 'lua.editorWidthPx'
const PANE_MIN_PX = 220 // mesmo mínimo có CSS (.pane-editor/.pane-preview min-width)

function aplicarLarguraEditor(px) {
  paneEditor.style.width = px + 'px'
}

// dragOverlay é un <div> transparente a pantalla completa que só existe
// mentres se arrastra o divisor - sen el, en canto o rato pasa por riba do
// <iframe id="preview"> (á dereita) o arrastre "atascábase": un iframe ten
// o seu propio documento, así que os mousemove que caen enriba del non
// chegan ao listener do documento principal (window). O overlay tápao todo
// (z-index alto) e recibe el os eventos no seu lugar.
let dragOverlay = null
function crearDragOverlay() {
  const div = document.createElement('div')
  div.className = 'pane-splitter-overlay'
  document.body.appendChild(div)
  return div
}

function initPaneSplitter() {
  // Largura gardada de sesións anteriores (localStorage - preferencia
  // puramente visual desta máquina, non fai falla pasar por Opcións/Go).
  let gardada = NaN
  try { gardada = parseInt(localStorage.getItem(PANE_SPLITTER_KEY), 10) } catch { /* sen almacenamento */ }
  if (gardada) aplicarLarguraEditor(gardada)

  let arrastrando = false

  paneSplitter.addEventListener('mousedown', (e) => {
    arrastrando = true
    paneSplitter.classList.add('dragging')
    document.body.style.cursor = 'col-resize'
    document.body.style.userSelect = 'none'
    dragOverlay = crearDragOverlay()
    e.preventDefault()
  })

  window.addEventListener('mousemove', (e) => {
    if (!arrastrando) return
    const panesRect = paneEditor.parentElement.getBoundingClientRect()
    const max = panesRect.width - PANE_MIN_PX - paneSplitter.offsetWidth
    const px = Math.min(Math.max(e.clientX - panesRect.left, PANE_MIN_PX), max)
    aplicarLarguraEditor(px)
  })

  window.addEventListener('mouseup', () => {
    if (!arrastrando) return
    arrastrando = false
    paneSplitter.classList.remove('dragging')
    document.body.style.cursor = ''
    document.body.style.userSelect = ''
    if (dragOverlay) {
      dragOverlay.remove()
      dragOverlay = null
    }
    try { localStorage.setItem(PANE_SPLITTER_KEY, String(paneEditor.getBoundingClientRect().width)) } catch { /* nada */ }
  })
}
initPaneSplitter()

// ── Axustes do texto (botón entre "Plantillas" e "Imprimir") ──────────────
// Panel flotante e arrastrable con tamaño de letra, interliñado e familia
// tipográfica. Aplícase á saída PDF (XerarInformeCronograma.tipografia ->
// latexdoc.go). Persístese en localStorage (preferencia visual local, coma o
// divisor arrastrable - non pasa por Settings/Go).
const AXUSTES_TEXTO_KEY = 'lua.tipografia'
let tipografia = { tamanoPt: 0, interlinado: 0, fonte: '' }
try {
  const g = JSON.parse(localStorage.getItem(AXUSTES_TEXTO_KEY) || '{}')
  tipografia = {
    tamanoPt: Number(g.tamanoPt) || 0,
    interlinado: Number(g.interlinado) || 0,
    // Os valores vellos (serif, sans, palatino) pasan á combinación equivalente.
    fonte: typeof g.fonte === 'string' ? ({ serif: 'clasica', sans: '', palatino: 'elegante' }[g.fonte] ?? g.fonte) : '',
  }
} catch { /* localStorage corrupto: quédanse os valores por defecto */ }

// tipografiaActual devólvese en cada petición de xerar. Os campos a 0/""
// significan "por defecto" tamén no backend (ver TipografiaOpts).
function tipografiaActual() {
  return { ...tipografia }
}

const axustesTextoPanel = document.getElementById('axustesTextoPanel')
const axustesTextoHead = document.getElementById('axustesTextoHead')
const optTextoSize = document.getElementById('optTextoSize')
const optTextoLeading = document.getElementById('optTextoLeading')
const optTextoFont = document.getElementById('optTextoFont')
const optTextoFontRecom = document.getElementById('optTextoFontRecom')
const optTextoFontSistema = document.getElementById('optTextoFontSistema')
if (optTextoFontRecom) optTextoFontRecom.label = t('axustesTexto.font.recom', 'Combinacións')
if (optTextoFontSistema) optTextoFontSistema.label = t('axustesTexto.font.sistema', 'Do sistema (xelatex)')

// fontesSistemaCargadas: as familias que devolve ListarFontesSistema
// (fc-list) só se piden a primeira vez que se abre o panel. null = aínda non
// se pediron; [] = pedíronse e non hai (fc-list ausente).
let fontesSistemaCargadas = null
async function cargarFontesSistema() {
  if (fontesSistemaCargadas !== null || !optTextoFontSistema) return
  try {
    fontesSistemaCargadas = (await ListarFontesSistema()) || []
  } catch {
    fontesSistemaCargadas = []
  }
  optTextoFontSistema.innerHTML = ''
  for (const nome of fontesSistemaCargadas) {
    const o = document.createElement('option')
    o.value = nome
    o.textContent = nome
    optTextoFontSistema.appendChild(o)
  }
  optTextoFontSistema.hidden = fontesSistemaCargadas.length === 0
}

function pintarAxustesTexto() {
  optTextoSize.value = tipografia.tamanoPt ? String(tipografia.tamanoPt) : ''
  optTextoLeading.value = tipografia.interlinado ? String(tipografia.interlinado) : '0'
  // Se a fonte gardada é do sistema e a lista aínda non cargou (ou fallou),
  // engádese como opción solta para que o selector a poida amosar.
  const f = tipografia.fonte || ''
  if (f && !optTextoFont.querySelector(`option[value="${CSS.escape(f)}"]`)) {
    const o = document.createElement('option')
    o.value = f
    o.textContent = f
    ;(optTextoFontSistema || optTextoFont).appendChild(o)
  }
  optTextoFont.value = f
}

function gardarAxustesTexto() {
  tipografia = {
    tamanoPt: parseFloat(optTextoSize.value) || 0,
    interlinado: parseFloat(optTextoLeading.value) || 0,
    fonte: optTextoFont.value || '',
  }
  try { localStorage.setItem(AXUSTES_TEXTO_KEY, JSON.stringify(tipografia)) } catch { /* sen persistencia, non pasa nada */ }
}

if (axustesTextoPanel) {
  makeDraggable(axustesTextoPanel, axustesTextoHead)
  // Non arrancar un arrastre ao premer o ✕ (está dentro da cabeceira-asa).
  document.getElementById('btnAxustesTextoClose').addEventListener('mousedown', (e) => e.stopPropagation())

  const toggleAxustesTexto = async () => {
    const oculto = axustesTextoPanel.classList.toggle('hidden')
    if (oculto) return
    await cargarFontesSistema()
    pintarAxustesTexto()
    optTextoSize.focus()
  }
  btnAxustesTexto && btnAxustesTexto.addEventListener('click', toggleAxustesTexto)
  document.getElementById('btnAxustesTextoClose').addEventListener('click', () => axustesTextoPanel.classList.add('hidden'))

  for (const el of [optTextoSize, optTextoLeading, optTextoFont]) {
    el.addEventListener('change', gardarAxustesTexto)
  }
  document.getElementById('btnTextoXerar').addEventListener('click', () => { gardarAxustesTexto(); run() })
  document.getElementById('btnTextoReset').addEventListener('click', () => {
    tipografia = { tamanoPt: 0, interlinado: 0, fonte: '' }
    try { localStorage.removeItem(AXUSTES_TEXTO_KEY) } catch { /* nada */ }
    pintarAxustesTexto()
    run()
  })
}

function setStatus(text) {
  statusEl.textContent = text
  btnAbrirGardado.hidden = true
}

// setStatusGardado: igual ca setStatus, pero cando hai un ficheiro local
// recén xerado (PDF/tex/md/docx/odt) amosa ademais un botón "Abrir" ao
// carón que o abre coa aplicación por defecto do sistema (AbrirFicheiro,
// abrirficheiro.go).
function setStatusGardado(text, ruta) {
  setStatus(text)
  if (!ruta) return
  btnAbrirGardado.hidden = false
  btnAbrirGardado.onclick = () => AbrirFicheiro(ruta).catch((err) => showWarnings([mensaxeErro(err)]))
}

// mensaxeErro: os erros que chegan dunha chamada ao backend (Go, vía
// bindings de Wails) son un RuntimeError con .message = a mensaxe real -
// mensaxeErro(err) sobre ese obxecto amosaba "RuntimeError: <mensaxe>", un
// prefixo que non aporta nada e fai parecer un erro técnico grave o que
// adoita ser un aviso normal.
function mensaxeErro(err) {
  return err && err.message ? err.message : String(err)
}

// ── Panel de rexistro (← #log-panel de Piztu, mesmo comportamento: engade
// liñas, corta ás LOG_MAX_LINEAS máis recentes, autoscroll). Arrinca
// minimizado (ver .collapsed en index.html) - só se usa para depurar a
// compilación do informe (pdflatex), non é o foco da app. ──
const LOG_MAX_LINEAS = 500
const logOutput = document.getElementById('log-output')

function log(txt, clase) {
  const linea = document.createElement('div')
  linea.textContent = txt
  if (clase) linea.className = clase
  logOutput.appendChild(linea)
  while (logOutput.childElementCount > LOG_MAX_LINEAS) {
    logOutput.removeChild(logOutput.firstChild)
  }
  logOutput.scrollTop = logOutput.scrollHeight
}

const logPanel = document.getElementById('log-panel')
const btnToggleLog = document.getElementById('btn-toggle-log')
btnToggleLog.addEventListener('click', () => {
  const colapsado = logPanel.classList.toggle('collapsed')
  btnToggleLog.textContent = colapsado ? '+' : '−'
})

const btnClearLog = document.getElementById('btn-clear-log')
btnClearLog.addEventListener('click', () => {
  logOutput.innerHTML = ''
})

// logPdflatex vai ao panel de rexistro a saída do compilador (campo `log` do
// resultado) cando hai algunha - liña a liña, co mesmo prefixo cós demais
// eventos do panel.
function logPdflatex(texto) {
  if (!texto) return
  for (const liña of texto.split('\n')) {
    if (liña.trim()) log('[PDFLATEX] ' + liña)
  }
}

// logAvisos: os avisos dunha xeración que saíu ben (pregunta sen solución,
// variable da plantilla sen definir…) van ao rexistro, despois da saída do
// compilador e en ámbar, para distinguilos do resto.
function logAvisos(list) {
  for (const w of list || []) log('[AVISO] ' + w, 'log-aviso')
}

function showWarnings(list) {
  warningsEl.innerHTML = ''
  if (!list || list.length === 0) {
    warningsEl.classList.remove('visible')
    return
  }
  for (const w of list) {
    const div = document.createElement('div')
    div.textContent = '⚠ ' + w
    warningsEl.appendChild(div)
  }
  warningsEl.classList.add('visible')
  ofrecerInstalarPaqueteLatex()
}

// ofrecerInstalarPaqueteLatex: se a compilación fallou porque falta un
// paquete de LaTeX, Lua ofrece instalalo alí mesmo (ver paquetelatex.go).
// Consúltase SEMPRE que se amosan avisos, non só nun sitio concreto: o
// paquete pode faltar ao xerar, ao exportar ou ao previsualizar unha
// plantilla, e o backend xa sabe se hai algo pendente ou non.
//
// O botón amosa o comando exacto que se vai executar (title): instalar
// software de sistema, con permisos de administrador, non se fai sen que se
// vexa o que se fai.
async function ofrecerInstalarPaqueteLatex() {
  let info
  try {
    info = await PaqueteLatexQueFalta()
  } catch {
    return
  }
  if (!info || !info.ficheiro) return
  // showWarnings puido volver correr mentres agardabamos (é asíncrono):
  // se o panel xa non está visible, non hai onde poñer nada.
  if (!warningsEl.classList.contains('visible')) return

  const fila = document.createElement('div')
  fila.className = 'warnings-paquete'
  if (!info.podeInstalar) {
    fila.textContent = t('latexPackage.cannot', 'Falta o paquete de LaTeX «{nome}»: {motivo}.',
      { nome: info.nome, motivo: info.motivo })
    warningsEl.appendChild(fila)
    return
  }

  const btn = document.createElement('button')
  btn.type = 'button'
  btn.className = 'warnings-bugreport-btn'
  btn.textContent = t('latexPackage.install', '⇩ Instalar «{nome}» automaticamente', { nome: info.nome })
  btn.title = t('latexPackage.command', 'Executarase: {comando}', { comando: info.comando })
  btn.addEventListener('click', async () => {
    const textoOriginal = btn.textContent
    btn.disabled = true
    btn.classList.add('is-busy')
    btn.textContent = t('latexPackage.installingBtn', '⏳ Instalando «{nome}»…', { nome: info.nome })
    setStatus(t('latexPackage.installing', 'Instalando o paquete «{nome}»…', { nome: info.nome }))
    log('[LATEX] Instalando ' + info.nome + ': ' + info.comando)
    // Deixar que o navegador pinte o estado "Instalando…" ANTES de arrincar a
    // instalación: pkexec/polkit abre un diálogo nativo que rouba o foco e,
    // en WebKitGTK, pode reter o repintado ata que se pecha - sen esta
    // pausa, o botón quedaba coa lenda vella mentres se pedía o contrasinal.
    await new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r)))
    try {
      const res = await InstalarPaqueteLatex()
      if (res.log) logPdflatex(res.log)
      if (res.success) {
        setStatus(t('latexPackage.installed', 'Paquete «{nome}» instalado', { nome: info.nome }))
        // Volver xerar: é o que quere quen premeu o botón, e así vese
        // decontado se o informe xa compila. (run() reconstrúe #warnings,
        // así que este botón desaparece; non fai falla restaurar a lenda.)
        await run()
      } else {
        setStatus(t('latexPackage.failed', 'Non se puido instalar «{nome}»', { nome: info.nome }))
        btn.disabled = false
        btn.classList.remove('is-busy')
        btn.textContent = textoOriginal
      }
    } catch (err) {
      setStatus(t('latexPackage.failed', 'Non se puido instalar «{nome}»', { nome: info.nome }))
      btn.disabled = false
      btn.classList.remove('is-busy')
      btn.textContent = textoOriginal
      showWarnings([mensaxeErro(err)])
    }
  })
  fila.appendChild(btn)
  warningsEl.appendChild(fila)
}

// ── Previsualización ─────────────────────────────────────────────────────

// zoomScript: Ctrl+/Ctrl-/Ctrl+0 para as páxinas dentro do iframe de
// previsualización (e da xanela de resultado desacoplada, que reusa este
// mesmo HTML). .pdf-page xa se axusta por defecto ao ancho dispoñible
// (max-width:100%), pero nun panel/xanela pequena iso pode quedar demasiado
// pequeno para ler. Ctrl+0 volve a ese axuste automático; o nivel avísase ao
// pai (handleZoomMessage) para sobrevivir a un "Xerar".
function zoomScript(initialZoom) {
  return `
<script>
let zoom = ${initialZoom}
function aplicarZoom() {
  document.querySelectorAll('.pdf-page').forEach((img) => {
    if (zoom === 1) { img.style.maxWidth = '100%'; img.style.width = ''; return }
    const disponible = document.body.clientWidth - 32 // 2x padding:1rem do body
    const axuste = Math.min(img.naturalWidth || disponible, disponible)
    img.style.maxWidth = 'none'
    img.style.width = Math.round(axuste * zoom) + 'px'
  })
}
document.addEventListener('keydown', (e) => {
  if (!(e.ctrlKey || e.metaKey)) return
  let novoZoom = zoom
  if (e.key === '+' || e.key === '=') novoZoom = Math.min(zoom * 1.2, 4)
  else if (e.key === '-' || e.key === '_') novoZoom = Math.max(zoom / 1.2, 0.3)
  else if (e.key === '0') novoZoom = 1
  else return
  e.preventDefault()
  zoom = novoZoom
  aplicarZoom()
  parent.postMessage({ luaSetZoom: zoom }, '*')
})
window.addEventListener('load', aplicarZoom)
</script>`
}

// pagesToPreviewHTML envolve as imaxes das páxinas do PDF (unha PNG por
// páxina, ver GeneratePDFResult.PageImages / pdfPageImages en latexdoc.go)
// nun pequeno documento HTML con scroll para o iframe de previsualización -
// é o que o webview renderiza de verdade, porque WebKitGTK (o motor de Wails
// en Linux) non ten un visor de PDF nativo fiable ao que apuntar un
// <iframe>/<embed>.
function pagesToPreviewHTML(pageImages, zoom = previewZoom) {
  if (!pageImages || pageImages.length === 0) {
    // Mensaxe neutra a propósito: amósase tamén antes de xerar nada (ex. a
    // xanela de resultado desacoplada ábrese xa dende o principio, sen nada
    // aínda) - non se pode saber aquí se é por iso ou por faltar pdftoppm de
    // verdade. O aviso específico de pdftoppm ausente xa chega por separado,
    // no panel de avisos, despois de Xerar.
    return `<!doctype html><html><body><p style="font:14px sans-serif;color:#666;padding:1rem">${t('preview.noPages', 'Aínda non hai nada que amosar. Preme "Xerar" para compilar o informe.')}</p></body></html>`
  }
  const pages = pageImages.map((src, i) => `<img src="${src}" class="pdf-page" data-page="${i + 1}">`).join('\n')
  return `<!doctype html><html><head><meta charset="utf-8">
<style>
  body{margin:0;padding:1rem;background:#787878;display:flex;flex-direction:column;align-items:center;gap:1rem}
  .pdf-page{max-width:100%;box-shadow:0 2px 8px rgba(0,0,0,.35);background:#fff}
  @media print{ body{background:#fff;padding:0;gap:0} .pdf-page{box-shadow:none;max-width:100%;page-break-after:always} }
</style>
</head><body>
${pages}
${zoomScript(zoom)}
</body></html>`
}

// handleZoomMessage recibe o nivel de zoom de zoomScript e gárdao en
// previewZoom para que sobreviva á próxima "Xerar" (pagesToPreviewHTML
// substitúe todo o srcdoc, perdendo calquera estado que só vivise dentro
// dese documento).
window.addEventListener('message', (e) => {
  if (!e.data || typeof e.data.luaSetZoom !== 'number') return
  previewZoom = e.data.luaSetZoom
})

// ── Informe (editor do cronograma + resultado) ───────────────────────────
// O editor da esquerda e toda a lóxica de ficheiros/desfacer/exportar viven
// en informes/controlador.js; aquí só se lle dan os bindings e os callbacks
// da chrome (barra de estado, avisos, panel de resultado).

// Barra de progreso da compilación (botón Xerar): ver progreso.js.
const progresoXerar = crearProgreso({
  contedor: document.getElementById('progresoXerar'),
  enche: document.getElementById('progresoXerarEnche'),
  t: (clave, fallback, vars) => t(clave, fallback, vars),
  almacen: {
    get: (k) => { try { return localStorage.getItem(k) } catch { return null } },
    set: (k, v) => { try { localStorage.setItem(k, v) } catch { /* sen persistencia */ } },
  },
})

// criarControlador monta o controlador dunha ferramenta sobre o seu contedor.
// Os bindings de ficheiros e borrador levan o id da ferramenta (cada unha ten
// o seu borrador e o seu tipo de ficheiro); o resto (xerar, exportar) é o
// mesmo binding, porque a petición xa di de que documento se trata.
function criarControlador(ferramenta) {
  const id = ferramenta.id
  const activo = () => ferramentaActiva === id
  return createInformes({
    container: contedorFerramenta[id],
    ferramenta,
    api: {
      XerarInformeCronograma, ExportarInformeTex, ExportarInformeMarkdown, ExportarInformeDocx, ExportarInformeOdt,
      EngadirTraballoBoveda,
      GardarBorradorInforme: (contido) => GardarBorradorInforme(id, contido),
      LerBorradorInforme: () => LerBorradorInforme(id),
      SavePDFDialog,
    },
    ui: {
      setStatus,
      setStatusGardado,
      showWarnings,
      log,
      logAvisos,
      mensaxeErro,
      confirmar,
      previsualizar: (paxinas) => {
        paxinasPorFerramenta[id] = paxinas
        if (activo()) preview.srcdoc = pagesToPreviewHTML(paxinas)
      },
      limparResultado: () => {
        paxinasPorFerramenta[id] = []
        if (activo()) preview.srcdoc = pagesToPreviewHTML([])
      },
      ocupado: (v, ok) => { btnRun.disabled = v; if (v) progresoXerar.comezar(id); else progresoXerar.rematar(ok !== false) },
      ocupadoBoton: (boton, v) => { if (boton) boton.disabled = v },
      logLatex: logPdflatex,
      aoFallar: () => { checkLatex() },
      aoCambiar: () => { if (activo()) actualizarAmbitos() },
      // «Engadir sempre á bóveda ao xerar» (só formularios en deseño).
      despoisDeXerar: () => { if (activo()) engadirABovedaAutomatico(id).catch(() => {}) },
    },
    t,
    tipografia: tipografiaActual,
    // A tarefa (só no cronograma) ábrese nunha xanela propia (maximizada, que
    // se pode levar a outro monitor); se non se pode, o editor usa a pantalla
    // completa desta.
    abrirXanelaTarefa: id === 'cronograma' ? (datos) => AbrirXanelaTarefa(datos.tarefa.id, JSON.stringify(datos)) : undefined,
    // Asistente de IA dentro da pantalla de edición da tarefa (ámbito: só esa tarefa).
    asistenteTarefa: id === 'cronograma' ? asistenteDaTarefa() : undefined,
  })
}

function crearInformes() {
  informesCrono = criarControlador(FERRAMENTA_CRONOGRAMA)
  informesActas = criarControlador(FERRAMENTA_ACTAS)
  controladores.set('cronograma', informesCrono)
  controladores.set('actas', informesActas)
  columnaUI = crearColumna({
    nav: ferramentasNav, zonaFixa: zonaFixaColumna, zonaResto: zonaRestoColumna, separador: document.getElementById('ferramentasSeparador'),
    t, biblioteca: () => biblioteca, existentes: idsExistentes, nomeDe: nomeDeFerramenta,
    gardar: gardarBibliotecaColumna, eliminar: eliminarFormularioDaColumna, exportar: exportarFormulario,
    cambiarIcona: cambiarIconaFormulario, quitar: (id) => quitarDoProxectoUI(id).catch((err) => showWarnings([mensaxeErro(err)])),
  })
  pintarColumna()
  informesCrono.iniciar()
  informesActas.iniciar()
  let inicial = 'cronograma'
  try { inicial = localStorage.getItem(CLAVE_FERRAMENTA) || 'cronograma' } catch { /* sen persistencia */ }
  ferramentaActiva = inicial === 'actas' ? 'actas' : 'cronograma'
  informes = controladores.get(ferramentaActiva)
  aplicarFerramentaNaInterface()
  // Os formularios propios entran despois (hai que ler a biblioteca); se o
  // último que se usou era un deles, volve a el cando estea listo.
  cargarFormularios(inicial).then(recollerMigracion).then(mostrarInicio).catch((err) => { setStatus(t('formularios.erro.carga', 'Non se puido ler a biblioteca de formularios')); showWarnings([mensaxeErro(err)]) })
}

// aplicarFerramentaNaInterface amosa o editor da ferramenta activa, marca o
// seu botón da columna da esquerda e repón o seu estado (resultado, ámbito).
function aplicarFerramentaNaInterface() {
  // Sen ningunha ferramenta á vista na columna (o cronograma e as actas só están
  // na biblioteca ata que o usuario as activa) amósase a pantalla de benvida.
  const { fixas: fx, resto: rs } = columnaDe(biblioteca, idsExistentes())
  const senFerramenta = fx.length + rs.length === 0
  document.body.classList.toggle('sen-ferramenta', senFerramenta)
  // Benvida: fóra dun proxecto, abrir un; nun proxecto baleiro, crear o seu
  // primeiro formulario (o deseñador trae o asistente) ou engadirlle ferramentas.
  for (const el of document.querySelectorAll('#benvida [data-benvida]')) el.hidden = (el.dataset.benvida === 'proxecto') !== Boolean(proxectoAberto)
  for (const [id, contedor] of Object.entries(contedorFerramenta)) contedor.hidden = senFerramenta || id !== ferramentaActiva
  ferramentasNav?.querySelectorAll('.ferramenta-btn[data-ferramenta]').forEach((b) => {
    b.setAttribute('aria-pressed', String(b.dataset.ferramenta === ferramentaActiva))
  })
  preview.srcdoc = pagesToPreviewHTML((!senFerramenta && paxinasPorFerramenta[ferramentaActiva]) || [])
  actualizarAmbitos()
  actualizarAsistentePorModo()
  actualizarBotonBoveda()
  setStatus(t('status.gardadoLua', 'Gardado en Lúa'))
}

// ── Bóveda (spec traballos-boveda) ──
// O botón «Bóveda» leva o documento á bóveda coma nota Markdown co seu PDF.
// Nos formularios hai ademais a marca «sempre» (engadir en cada Xerar). En
// produción as dúas cousas están desactivadas: o asistente le a bóveda.
const enProducion = (id) => controladores.get(id)?.ferramenta.esquema?.modo === 'produccion'

async function actualizarBotonBoveda() {
  if (!btnBoveda || souXanelaResultado) return
  const id = ferramentaActiva
  const formulario = haiFerramenta() && id.startsWith('form:')
  const producion = formulario && enProducion(id)
  btnBoveda.disabled = producion
  btnBoveda.title = producion
    ? t('preview.boveda.produccion', 'En produción non se engade á bóveda: os datos poden ser reais e o asistente le a bóveda.')
    : t('preview.boveda.title', 'Engadir este documento á bóveda de coñecemento (nota Markdown e PDF en wiki/traballos/). Se o proxecto non ten bóveda, Lúa créaa.')
  bovedaSempreFila.hidden = !formulario
  chkBovedaSempre.disabled = producion
  chkBovedaSempre.checked = false
  if (formulario && !producion) {
    try { chkBovedaSempre.checked = await BovedaAutomaticaDe(id) } catch { /* queda desmarcada */ }
  }
}

async function engadirABovedaAutomatico(id) {
  if (!id.startsWith('form:') || enProducion(id)) return
  const c = controladores.get(id)
  // Só co resultado ao día: así nunca se volve xerar desde aquí.
  if (!c || !c.estaAoDia() || !await BovedaAutomaticaDe(id)) return
  await c.engadirABoveda(nomeDeFerramenta(id))
}

// cambiarFerramenta pasa á ferramenta `id` (botóns da columna da esquerda):
// cada unha conserva o seu curso, o seu ficheiro, o seu desfacer e o seu resultado.
async function cambiarFerramenta(id) {
  if (!contedorFerramenta[id] || !controladores.has(id) || id === ferramentaActiva || !informesCrono) return
  // Ir a unha ferramenta agochada (desde a biblioteca ou co asistente) a amosa de novo.
  if (biblioteca.agochados.includes(id)) {
    try { await gardarBiblioteca(agochar(biblioteca, id, false)) } catch (err) { showWarnings([mensaxeErro(err)]) }
  }
  await informes.volcarBorrador()
  ferramentaActiva = id
  informes = controladores.get(id)
  try { localStorage.setItem(CLAVE_FERRAMENTA, id) } catch { /* sen persistencia */ }
  aplicarFerramentaNaInterface()
  if (asistente) asistente.refrescarBoveda?.() // cada ferramenta ten a súa bóveda
  sincronizarPlantillaDoDocumento()
  informes.focus()
}

// ── Formularios propios ──
// Cada formulario da biblioteca é unha ferramenta máis: un botón na columna
// esquerda, un contedor no panel do editor e un controlador (o mesmo dos
// informes: ficheiros, desfacer, borrador, xerar e exportar). O esquema
// edítase co deseñador (formularios/disenador.js).
const formularioNav = document.getElementById('ferramentasNav')
const btnNovoFormulario = document.getElementById('btnNovoFormulario')
const btnBibliotecaFormularios = document.getElementById('btnBibliotecaFormularios')

// A columna esquerda ten dúas zonas: a FIXA (cronograma, actas e os formularios
// que o usuario fixe: non se poden borrar, só ocultar) e a de abaixo (os demais
// formularios, que si se poden borrar). Como se organiza (zona fixa, ocultos e
// cartafoles da biblioteca) vive en Go (biblioteca_formularios.go).
let biblioteca = bibliotecaBaleira()
let columnaUI = null

const nomeDeFerramenta = (id) => (id === 'cronograma' ? t('ferramentas.cronograma', 'Cronograma anual')
  : id === 'actas' ? t('ferramentas.actas', 'Actas do departamento')
    : controladores.get(id)?.ferramenta.esquema.nome || id)

// idsFormularios: os ids «form:<id>» por nome.
const idsFormularios = () => [...controladores.keys()].filter((id) => id.startsWith('form:'))
  .sort((a, b) => nomeDeFerramenta(a).localeCompare(nomeDeFerramenta(b), undefined, { sensitivity: 'base' }))
const idsExistentes = () => ['cronograma', 'actas', ...idsFormularios()]
const pintarColumna = () => columnaUI?.pintar()

// gardarBiblioteca persiste un novo estado da biblioteca e repinta a columna. Lanza
// se Go o rexeita (p. ex. agochar todo); quen chama decide como amosar o erro.
async function gardarBiblioteca(nova) {
  biblioteca = completar(await GardarBibliotecaFormularios(nova))
  pintarColumna()
  await asegurarFerramentaVisible()
}
async function gardarBibliotecaColumna(nova) {
  try { await gardarBiblioteca(nova) } catch (err) { showWarnings([mensaxeErro(err)]) }
}
async function recargarBiblioteca() {
  try { biblioteca = completar(await LerBibliotecaFormularios()) } catch { /* queda a de antes */ }
  pintarColumna()
}

// Se a ferramenta activa se agocha, vaise á primeira que se vexa.
async function asegurarFerramentaVisible() {
  const { fixas, resto } = columnaDe(biblioteca, idsExistentes())
  const visibles = [...fixas, ...resto]
  if (visibles.includes(ferramentaActiva)) { aplicarFerramentaNaInterface(); return }
  const alt = visibles[0]
  if (alt) await cambiarFerramenta(alt)
  else aplicarFerramentaNaInterface() // ningunha: pantalla de benvida
}

// Os esquemas de todos os formularios cargados (para repartir as cores).
const esquemasFormularios = () => idsFormularios().map((id) => controladores.get(id)?.ferramenta.esquema).filter(Boolean)

// Cada formulario ten a súa icona e a súa cor (iconas.js), para distinguilos
// dunha ollada. O modo vai nun punto na esquina: VERMELLO en deseño (a IA
// pode axudar, con datos ficticios) e VERDE en produción (datos reais, sen
// IA). Tamén vai no texto, para non depender só da cor.
function pintarBotonFormulario(boton, esquema) {
  const produccion = esquema.modo === 'produccion'
  const modo = produccion ? t('formularios.modo.produccion', 'Produción') : t('formularios.modo.deseno', 'Deseño')
  const cor = coresFormularios([...esquemasFormularios().filter((e) => e.id !== esquema.id), esquema])[esquema.id]
  boton.innerHTML = svgIcona(esquema.icona) + '<span class="ferramenta-btn__modo" aria-hidden="true"></span>'
  boton.style.setProperty('--cor-formulario', valorCor(cor))
  boton.classList.toggle('ferramenta-btn--produccion', produccion)
  boton.classList.toggle('ferramenta-btn--deseno', !produccion)
  boton.title = `${esquema.nome} · ${modo}`
  boton.setAttribute('aria-label', `${esquema.nome}, ${t('formularios.modo.etq', 'modo')} ${modo}`)
}

// cambiarIconaFormulario abre a xanela «Cambiar icona» (menú do botón dereito
// ou «⋯» da biblioteca) e garda a icona e a cor escollidas, sen abrir o deseñador.
async function cambiarIconaFormulario(id) {
  const c = controladores.get(id)
  if (!c) return
  const esquema = c.ferramenta.esquema
  const corActual = coresFormularios(esquemasFormularios())[esquema.id]
  const r = await escollerIcona({ t, nome: esquema.nome, actual: esquema.icona, cor: corActual })
  if (!r || (r.icona === (esquema.icona || 'formulario') && r.cor === esquema.cor)) return
  try {
    const novo = await GardarIconaFormulario(esquema.id, r.icona, r.cor)
    c.ferramenta.actualizarEsquema(novo)
    repintarBotonsFormularios()
    pintarColumna()
  } catch (err) {
    showWarnings([mensaxeErro(err)])
  }
}

function engadirFormulario(esquema) {
  const id = 'form:' + esquema.id
  if (controladores.has(id)) return controladores.get(id)
  const contedor = document.createElement('div')
  contedor.className = 'contedor-ferramenta'
  contedor.hidden = true
  document.querySelector('.pane-editor').append(contedor)
  contedorFerramenta[id] = contedor
  paxinasPorFerramenta[id] = []
  const boton = document.createElement('button')
  boton.type = 'button'
  boton.className = 'ferramenta-btn'
  boton.dataset.ferramenta = id
  boton.setAttribute('aria-pressed', 'false')
  pintarBotonFormulario(boton, esquema)
  zonaRestoColumna.append(boton)
  const ferr = crearFerramentaFormulario(esquema, {
    aoDeseñar: (fid) => deseñarFormulario(fid),
    aoCambiarModo: (fid, modo) => cambiarModoFormulario(fid, modo),
    aoBorrarDatosProba: (fid) => borrarDatosProba(fid),
    // O asistente de datos (só existe en modo deseño, ver formularios/editor.js).
    asistente: asistenteDoFormularioDatos(),
    // Os campos «problemas»: Maxima e o resolvedor coa IA (formularios/problemas.js).
    problemas: { api: { CalcularProblemas, ResolverProblema, InstallMaxima, IAAsistenteEstado }, idioma: () => idiomaActual },
    avisar: (mensaxe, tipo) => { setStatus(mensaxe); if (tipo === 'aviso') showWarnings([mensaxe]) },
  })
  const controlador = criarControlador(ferr)
  controladores.set(id, controlador)
  controlador.iniciar()
  pintarColumna()
  return controlador
}

async function cargarFormularios(inicial) {
  const lista = (await ListarFormularios()) || []
  await recargarBiblioteca()
  for (const esquema of lista) engadirFormulario(esquema)
  pintarColumna()
  if (inicial && inicial.startsWith('form:') && controladores.has(inicial) && !biblioteca.agochados.includes(inicial)) await cambiarFerramenta(inicial)
  await asegurarFerramentaVisible()
}

// ── Proxectos ──
// Un proxecto é unha documentación concreta feita con formularios da biblioteca:
// leva a súa copia deles e os seus propios datos (proxecto.go). Entrar nun
// proxecto cambia en Go o sitio onde viven os formularios e os datos; aquí
// recárganse as ferramentas para velo.
const proxectoChip = document.getElementById('proxectoActivo')

// Os datos pertencen sempre a un proxecto (spec trasladar): o chip só di cal
// está aberto, coa súa cor; para cambiar, o inicio (botón Proxectos).
function pintarChipProxecto(p) {
  if (!proxectoChip) return
  proxectoChip.hidden = !proxectoAberto
  proxectoChip.replaceChildren()
  if (!proxectoAberto) return
  proxectoChip.style.setProperty('--cor-proxecto', corProxecto(p))
  proxectoChip.append(t('proxectos.chip', 'Proxecto: {nome}', { nome: p?.nome || '' }))
}

// cambiarProxecto entra no proxecto `id`. Os documentos en curso vólcanse ANTES
// de cambiar o sitio dos datos, agás cando o que hai no disco é máis novo ca o
// que hai en pantalla (despois de meter algo con Trasladar): `volcar: false`.
// Fóra de proxecto (ao arrincar Lúa está no inicio) non hai nada que volcar.
let nomeProxectoAberto = ''
async function cambiarProxecto(id, { volcar } = {}) {
  if (volcar ?? Boolean(proxectoAberto)) for (const c of controladores.values()) await c.volcarBorrador()
  const p = await ActivarProxecto(id)
  proxectoAberto = id
  nomeProxectoAberto = p?.nome || ''
  for (const fid of idsFormularios()) quitarFormulario(fid)
  await informesCrono.recargar()
  await informesActas.recargar()
  await cargarFormularios(null)
  pintarChipProxecto(p)
  // Os controladores dos formularios son OBXECTOS NOVOS: `informes` ten que apuntar
  // ao novo (senón Xerar/Gardar usarían o vello, que xa non ten os datos do editor).
  if (!controladores.has(ferramentaActiva)) ferramentaActiva = 'cronograma'
  informes = controladores.get(ferramentaActiva)
  if (asistente) asistente.refrescarBoveda?.() // a bóveda do proxecto, se a ten
  await asegurarFerramentaVisible()
  await sincronizarPlantillaDoDocumento()
  setStatus(t('status.gardadoLua', 'Gardado en Lúa'))
}

// recollerMigracion: se ao arrincar Go pasou o traballo solto dunha versión
// anterior a un proxecto, avísase unha vez (o proxecto sae no inicio).
let avisoMigracion = ''
async function recollerMigracion() {
  try {
    const m = await MigracionTraballoSolto()
    if (!m?.proxecto) return
    avisoMigracion = t('trasladar.migracion', 'O teu traballo de antes (fóra de proxectos) está agora no proxecto «{nome}». Lúa garda todo só; para levar algo fóra usa Trasladar.', { nome: m.nome })
  } catch (err) {
    showWarnings([mensaxeErro(err)])
  }
}

// ── Inicio ──
// Lúa arrinca sen proxecto aberto, no inicio (formularios/proxectos.js): o
// panel da esquerda leva «Novo proxecto» e os proxectos, e os que non caben
// seguen no da dereita. Ao abrir un, os paneis volven ser editor e resultado.
// O botón Proxectos da columna volve ao inicio sen pechar o proxecto aberto.
let inicio = null
const btnProxectos = document.getElementById('btnProxectos')
const enInicio = () => document.body.classList.contains('inicio')

function mostrarInicio() {
  if (!inicio) {
    inicio = crearInicio({
      t,
      esquerda: document.getElementById('inicioEsquerda'),
      dereita: document.getElementById('inicioDereita'),
      accions: {
        listar: () => ListarProxectos(),
        formularios: async () => (await ListarFormulariosBiblioteca()) || [],
        integradas: [
          { id: 'cronograma', nome: nomeDeFerramenta('cronograma') },
          { id: 'actas', nome: nomeDeFerramenta('actas') },
        ],
        crear: (nome, materia) => CrearProxecto(nome, materia, []),
        abrir: async (id) => { if (id !== proxectoAberto) await cambiarProxecto(id); ocultarInicio() },
        eliminar: (id) => EliminarProxecto(id),
        fixarCor: async (id, cor) => { const p = await FixarCorProxecto(id, cor); if (id === proxectoAberto) pintarChipProxecto(p) },
        buxo: () => BuxoEstadoActual(),
        abrirBuxo: (id) => AbrirBuxoDoProxecto(id),
        escollerCartafol: () => EscollerCartafolProxecto(),
        mover: (id, destino) => MoverProxecto(id, destino),
        confirmar,
      },
    })
  }
  document.body.classList.add('inicio')
  btnProxectos?.setAttribute('aria-pressed', 'true')
  setStatus('')
  if (avisoMigracion) { showWarnings([avisoMigracion]); avisoMigracion = '' }
  return inicio.recargar().then(() => inicio.enfocar())
}

function ocultarInicio() {
  if (!enInicio()) return
  document.body.classList.remove('inicio')
  btnProxectos?.setAttribute('aria-pressed', 'false')
  aplicarFerramentaNaInterface()
}

// O botón Proxectos alterna: do proxecto ao inicio e, se hai un aberto, volta.
btnProxectos?.addEventListener('click', () => {
  if (enInicio() && proxectoAberto) ocultarInicio()
  else mostrarInicio().catch((err) => showWarnings([mensaxeErro(err)]))
})
document.getElementById('btnBenvidaProxectos')?.addEventListener('click', () => mostrarInicio().catch((err) => showWarnings([mensaxeErro(err)])))
document.getElementById('btnBenvidaProxectosEngadir')?.addEventListener('click', () => abrirCatalogoUI())
document.getElementById('btnBenvidaFormulario')?.addEventListener('click', () => deseñarFormulario(null))

function quitarFormulario(id) {
  const controlador = controladores.get(id)
  if (!controlador) return
  controlador.descartar() // o borrador non pode volver a escribirse (podería levar datos reais)
  controladores.delete(id)
  contedorFerramenta[id]?.remove()
  delete contedorFerramenta[id]
  delete paxinasPorFerramenta[id]
  formularioNav.querySelector(`.ferramenta-btn[data-ferramenta="${CSS.escape(id)}"]`)?.remove()
  repintarBotonsFormularios() // os de antes (sen cor gardada) poden cambiar de cor
  pintarColumna()
}

function repintarBotonsFormularios() {
  for (const id of idsFormularios()) {
    const boton = formularioNav.querySelector(`.ferramenta-btn[data-ferramenta="${CSS.escape(id)}"]`)
    const esquema = controladores.get(id)?.ferramenta.esquema
    if (boton && esquema) pintarBotonFormulario(boton, esquema)
  }
}

// eliminarFormularioDaColumna borra un formulario (menú do botón dereito ou biblioteca),
// SEMPRE preguntando antes. Os da zona fixa non se poden borrar. Devolve se se borrou.
async function eliminarFormularioDaColumna(tid) {
  const c = controladores.get(tid)
  if (!c || !tid.startsWith('form:')) return false
  const fid = tid.slice(5)
  const nome = c.ferramenta.esquema.nome
  if (!podeBorrar(biblioteca, tid)) {
    showWarnings([t('formularios.menu.eliminar.fixo', 'Os formularios da zona fixa non se poden borrar: quítao da zona fixa primeiro (ou ocúltao).')])
    return false
  }
  let pregunta = t('formularios.eliminar.confirmar', 'Seguro que queres eliminar o formulario «{nome}»? Non se pode desfacer. Os documentos que gardaras noutros ficheiros non se tocan.', { nome })
  if (haiContidoFormulario(c.getPlan())) pregunta += ' ' + t('formularios.eliminar.datos', 'Os datos que hai escritos nel sen gardar perderanse.')
  if (!await confirmar(pregunta)) return false
  try {
    // Primeiro se para o borrador (senón podería reescribirse ao cambiar de ferramenta) e despois se borra.
    c.descartar()
    if (ferramentaActiva === tid) {
      const { fixas, resto } = columnaDe(biblioteca, idsExistentes().filter((x) => x !== tid))
      await cambiarFerramenta(fixas[0] || resto[0] || 'cronograma')
    }
    await EliminarFormulario(fid)
  } catch (err) {
    c.descartar(false)
    setStatus(t('status.error', 'Erro'))
    showWarnings([mensaxeErro(err)])
    return false
  }
  quitarFormulario(tid)
  await recargarBiblioteca()
  setStatus(t('formularios.eliminado', 'Formulario eliminado'))
  return true
}

// borrarDatosProba baleira o documento dun formulario en produción cando a IA
// deixara nel datos de proba (en deseño). Pódese desfacer co historial.
async function borrarDatosProba(fid) {
  const c = controladores.get('form:' + fid)
  if (!c) return
  if (!await confirmar(t('formularios.proba.confirmar', 'Borrar TODOS os datos deste formulario (os de proba que puxo a IA e calquera outro que escribises despois)? Podes desfacelo con Desfacer.'))) return
  c.substituirPlan(c.ferramenta.novoPlan(), 'estrutura')
  setStatus(t('formularios.proba.borrados', 'Datos de proba borrados'))
}

// exportarFormulario (menú do botón dereito ou «⋯» da biblioteca): abre
// Trasladar co documento dese formulario (estrutura e datos) xa escollido.
async function exportarFormulario(tid) {
  if (!controladores.has(tid) || !tid.startsWith('form:')) return
  if (tid !== ferramentaActiva) await cambiarFerramenta(tid)
  await abrirTrasladarUI({ lapela: 'sacar', opcion: 'documento' })
}

// importarFormulario: «Importar formulario…» da biblioteca abre Trasladar.
// Devolve null: a biblioteca non ten que amosar nada (faino Trasladar).
async function importarFormulario() {
  abrirTrasladarUI({ lapela: 'meter' })
  return null
}

// ── Trasladar ──
// O único sitio para sacar cousas de Lúa e metelas (trasladar/trasladar.js).
// Ábrese co botón ⇅, con Ctrl+O (meter) e Ctrl+Maiús+S (sacar), ao soltar un
// ficheiro na xanela e dende os menús de formularios e plantillas.
function contextoTrasladar() {
  const proxecto = proxectoAberto ? { id: proxectoAberto, nome: nomeProxectoAberto || proxectoAberto } : null
  const documento = haiFerramenta() && informes ? { ferramenta: ferramentaActiva, nome: nomeDeFerramenta(ferramentaActiva) } : null
  return { proxecto, documento }
}

// describirCurso: o texto dun curso dun ficheiro con varios (Piztu Orienta).
function describirCurso(ferramenta, curso, i) {
  try {
    const f = ferramenta === 'actas' ? FERRAMENTA_ACTAS : FERRAMENTA_CRONOGRAMA
    const texto = ferramenta === 'actas' ? JSON.stringify({ ferramenta: 'actas', actas: [curso] }) : JSON.stringify({ plans: [curso] })
    return f.descricionCurso(f.lerFicheiro(texto)[0], t)
  } catch {
    return String(i + 1)
  }
}

// recargarDespoisDeMeter volve ler o que cambiou SEN volcar a pantalla enriba
// do que se acaba de meter.
async function recargarDespoisDeMeter(res) {
  if (res.reiniciar) { location.reload(); return }
  await recargarBiblioteca()
  await refrescarSelectorPlantillas()
  const existentes = ((await ListarProxectos()) || []).map((p) => p.id)
  if (res.abrirProxecto && !res.desfeito && existentes.includes(res.abrirProxecto)) {
    await cambiarProxecto(res.abrirProxecto, { volcar: false })
    ocultarInicio()
  } else if (existentes.includes(proxectoAberto)) {
    await cambiarProxecto(proxectoAberto, { volcar: false })
  } else {
    // O proxecto aberto xa non existe (Desfacer, copia restaurada): ao inicio.
    if (proxectoAberto) await cambiarProxecto('', { volcar: false })
    await mostrarInicio()
  }
  if (enInicio()) await inicio.recargar()
}

function abrirTrasladarUI(opcions = {}) {
  return abrirTrasladar({
    t,
    contexto: contextoTrasladar(),
    confirmar,
    describirCurso,
    api: {
      ArboreTraslado, SacarTenDatosProducion, SacarTraslado, SacarDocumentoJSON,
      contidoDocumento: () => informes.contidoDocumento(),
      EscollerFicheiroMeter, EscollerCartafolMeter, AnalizarMeter, AplicarMeter, DesfacerTraslado,
    },
    antesDeMeter: async () => { for (const c of controladores.values()) await c.volcarBorrador() },
    despoisDeMeter: (res) => recargarDespoisDeMeter(res).catch((err) => showWarnings([mensaxeErro(err)])),
    ...opcions,
  }).catch((err) => showWarnings([mensaxeErro(err)]))
}

// Soltar un ficheiro ou cartafol en calquera parte de Lúa (trasladar_app.go).
Events.On('lua:trasladar-soltados', (e) => {
  const rutas = Array.isArray(e.data) ? e.data : []
  if (!rutas.length || souXanelaResultado || souXanelaTarefa) return
  if (rutas.length > 1) setStatus(t('trasladar.soUn', 'Soltaches {n} elementos: Lúa mete o primeiro. Solta os demais despois.', { n: rutas.length }))
  abrirTrasladarUI({ ruta: rutas[0] })
})

// A biblioteca de formularios: todos, en cartafoles, con descrición (formularios/biblioteca.js).
function verBibliotecaFormularios() {
  return abrirBibliotecaFormularios({
    t,
    accions: {
      formularios: () => idsFormularios().map((id) => controladores.get(id).ferramenta.esquema),
      biblioteca: () => biblioteca,
      integradas: ['cronograma', 'actas'].map((id) => ({ id, nome: nomeDeFerramenta(id) })),
      gardar: gardarBiblioteca,
      crearCarpeta: async (nome) => { biblioteca = completar(await CrearCarpetaFormularios(nome)); pintarColumna() },
      gardarDescricion: async (fid, texto) => {
        const novo = await GardarDescricionFormulario(fid, texto)
        const c = controladores.get('form:' + fid)
        if (c) { c.ferramenta.actualizarEsquema(novo); c.refrescarIdioma() }
      },
      abrir: (tid) => cambiarFerramenta(tid),
      eliminar: eliminarFormularioDaColumna,
      exportar: exportarFormulario,
      cambiarIcona: cambiarIconaFormulario,
      importar: importarFormulario,
      confirmar,
      preguntar,
    },
  })
}
btnBibliotecaFormularios?.addEventListener('click', () => verBibliotecaFormularios())

// cambiarModoFormulario pasa un formulario a modo 'deseno' ou 'produccion'.
// Cara a produción desaparecen o asistente e o deseñador. Cara a deseño, o
// asistente xa podería ver os datos: por iso se baleiran os que haxa (o
// documento en pantalla, non os ficheiros que o usuario gardase) tras avisar.
async function cambiarModoFormulario(fid, modo) {
  const id = 'form:' + fid
  const c = controladores.get(id)
  if (!c) return
  const esquema = c.ferramenta.esquema
  const producion = modo === 'produccion'
  if ((esquema.modo === 'produccion') === producion) return
  const baleirar = !producion && haiContidoFormulario(c.getPlan())
  const pregunta = producion
    ? t('formularios.modo.confirmar.produccion', 'Pasar «{nome}» a PRODUCIÓN? Desaparecen o botón Deseñar formulario e o asistente de IA, e a IA deixa de ter acceso a este formulario. Podes volver a deseño cando queiras.', { nome: esquema.nome })
      + (c.getPlan()?.datosProba ? ' ' + t('formularios.modo.confirmar.proba', 'Os datos de proba que puxo a IA seguen no formulario: en produción terás un botón para borralos.') : '')
    : baleirar
      ? t('formularios.modo.confirmar.desenoBaleirar', 'Pasar «{nome}» a DESEÑO? O asistente de IA poderá ver os datos deste formulario, así que se BALEIRAN os que hai agora en pantalla (os ficheiros que gardaras non se tocan). Continuar?', { nome: esquema.nome })
      : t('formularios.modo.confirmar.deseno', 'Pasar «{nome}» a DESEÑO? Volven o botón Deseñar formulario e o asistente de IA, que poderá ver os datos que escribas aquí. Usa só datos ficticios.', { nome: esquema.nome })
  if (!await confirmar(pregunta)) return
  try {
    const novo = await CambiarModoFormulario(fid, modo)
    c.ferramenta.actualizarEsquema(novo)
    // O documento en pantalla nace de novo co contexto que lle toca ao modo novo.
    if (baleirar) c.substituirPlan(c.ferramenta.novoPlan(), 'estrutura')
    const boton = formularioNav.querySelector(`.ferramenta-btn[data-ferramenta="${CSS.escape(id)}"]`)
    if (boton) pintarBotonFormulario(boton, novo)
    c.refrescarIdioma()
    actualizarAsistentePorModo()
    actualizarBotonBoveda()
    setStatus(producion
      ? t('formularios.modo.aProduccion', 'Formulario en produción: {nome}', { nome: novo.nome })
      : t('formularios.modo.aDeseno', 'Formulario en deseño: {nome}', { nome: novo.nome }))
  } catch (err) {
    setStatus(t('status.error', 'Erro'))
    showWarnings([mensaxeErro(err)])
  }
}

// actualizarAsistentePorModo: coa ferramenta activa nun formulario en PRODUCIÓN
// o asistente xeral non aparece (nin se pode abrir): que a IA non capture datos
// privados. As súas ferramentas tamén o negan (asistente/ferramentas.js).
function actualizarAsistentePorModo() {
  const bloqueado = !!informes?.ferramenta?.enProduccion
  if (btnAsistente) btnAsistente.hidden = bloqueado
  if (bloqueado) asistente?.pechar()
}

// deseñarFormulario abre o deseñador para un formulario da biblioteca (co seu
// id) ou para un novo (sen id).
// As funcións de Go e as opcións do asistente que precisa o deseñador (na xanela principal e na propia).
const apiDisenador = () => ({ GardarFormulario, EliminarFormulario, CatalogoFluxo, FluxoAutomaticoDe, SincronizarFluxoFormulario, AvaliarFluxoFormulario })
const asistenteDoDisenador = () => ({
  api: { IAAsistenteEstado, IAFerramentasDoAmbito, IATurnoAsistente, IAChamarFerramenta, AdxuntoGardar, AdxuntoEliminar, AdxuntosLimpar, AdxuntoVista, AdxuntoDataURI, BovedaEstadoDe, BovedaProxectoConectar, BovedaConectar, BovedaEscoller, IAFerramentasBoveda },
  idioma: () => idiomaActual,
})

// deseñarFormulario abre o deseñador para un formulario da biblioteca (co seu
// id) ou para un novo (sen id). Ábrese nunha XANELA PROPIA a pantalla completa
// (que se pode levar a outro monitor); o resultado chega despois por eventos
// (lua:formulario-gardado / -eliminado, ver aplicarResultadoDisenador). Se non
// hai xanelas nativas (proba en navegador), ábrese coma unha capa aquí mesmo.
async function deseñarFormulario(fid) {
  const id = fid ? 'form:' + fid : null
  // Un formulario novo nace coa forma e a cor que menos se usan.
  const gardado = id ? controladores.get(id)?.ferramenta.esquema : null
  // Un dos de antes (sen cor gardada) leva a que xa amosa: ao gardar queda fixada.
  const actual = gardado ? { ...gardado, cor: coresFormularios(esquemasFormularios())[gardado.id] } : { ...novoEsquema(), ...aparenciaLibre(esquemasFormularios()) }
  // Un formulario en produción non se deseña (o botón nin se ofrece; isto é o seguro).
  if (actual?.modo === 'produccion') {
    showWarnings([t('formularios.modo.bloqueado', 'Este formulario está en modo produción: pásao a deseño para cambialo.')])
    return
  }
  const plan = id ? controladores.get(id)?.getPlan() : null
  const datos = plan ? { contexto: plan.contexto, valores: plan.valores } : {}
  try {
    if (await AbrirXanelaFormulario(fid || '', JSON.stringify({ esquema: actual, datos }))) return
  } catch (err) {
    setStatus(t('status.error', 'Erro'))
    showWarnings([mensaxeErro(err)])
    return
  }
  // Sen xanela nativa: capa dentro desta xanela.
  const r = await abrirDisenador({ t, esquema: actual, confirmar, api: apiDisenador(), obterDatos: () => datos, asistente: asistenteDoDisenador() })
  if (r) await aplicarResultadoDisenador(r)
}

// aplicarResultadoDisenador leva ao programa o que fixo o deseñador: unha icona
// nova, o esquema cambiado dun formulario ou un formulario eliminado.
async function aplicarResultadoDisenador(r) {
  try {
    if (r.accion === 'eliminado') {
      const id = 'form:' + r.id
      // Xa se borrou en Go: o borrador non pode volver a escribirse ao cambiar de ferramenta.
      controladores.get(id)?.descartar()
      if (ferramentaActiva === id) await cambiarFerramenta('cronograma')
      quitarFormulario(id)
      await recargarBiblioteca()
      setStatus(t('formularios.eliminado', 'Formulario eliminado'))
      return
    }
    const novoId = 'form:' + r.esquema.id
    if (r.novo || !controladores.has(novoId)) {
      engadirFormulario(r.esquema)
      await cambiarFerramenta(novoId)
      controladores.get(novoId)?.focus()
    } else {
      // O controlador segue (co seu documento, desfacer e ficheiro): só cambia o esquema.
      const c = controladores.get(novoId)
      c.ferramenta.actualizarEsquema(r.esquema)
      const boton = formularioNav.querySelector(`.ferramenta-btn[data-ferramenta="${CSS.escape(novoId)}"]`)
      if (boton) pintarBotonFormulario(boton, r.esquema)
      pintarColumna() // o nome pode ter cambiado: a orde da zona de abaixo
      c.refrescarIdioma()
      if (c.temResultado() && ferramentaActiva === novoId) run()
    }
    setStatus(t('formularios.gardado', 'Formulario gardado: {nome}', { nome: r.esquema.nome }))
  } catch (err) {
    setStatus(t('status.error', 'Erro'))
    showWarnings([mensaxeErro(err)])
  }
}

// «Deseñar formulario» (botón +): a ventá do catálogo común, onde se reutilizan
// os formularios xa creados (e o cronograma e as actas) ou se crea un novo.
btnNovoFormulario?.addEventListener('click', () => abrirCatalogoUI())

function abrirCatalogoUI() {
  return abrirCatalogo({
    t,
    accions: {
      catalogo: () => CatalogoFormularios(),
      nomeIntegrada: nomeDeFerramenta,
      reutilizar: reutilizarNoProxectoUI,
      novo: () => deseñarFormulario(null),
      crearCarpeta: (nome) => CrearCarpetaCatalogo(nome),
      renomearCarpeta: (id, nome) => RenomearCarpetaCatalogo(id, nome),
      eliminarCarpeta: (id) => EliminarCarpetaCatalogo(id),
      mover: (fid, carpeta) => MoverFormularioCatalogo(fid, carpeta),
      preguntar,
      confirmar,
    },
  }).catch((err) => showWarnings([mensaxeErro(err)]))
}

// reutilizarNoProxectoUI mete no proxecto aberto as ferramentas escollidas e
// vai á última. Os documentos en curso vólcanse antes de recargar.
async function reutilizarNoProxectoUI(ferramentas) {
  for (const c of controladores.values()) await c.volcarBorrador()
  await ReutilizarNoProxecto(ferramentas)
  await cambiarProxecto(proxectoAberto, { volcar: false })
  const ultima = ferramentas.at(-1)
  if (ultima && controladores.has(ultima)) await cambiarFerramenta(ultima)
  setStatus(t('catalogo.reutilizados', 'Engadido ao proxecto: {n}', { n: ferramentas.map(nomeDeFerramenta).join(', ') }))
}

// quitarDoProxectoUI (botón dereito na columna): saca a ferramenta do proxecto
// aberto, coa súa copia e os seus datos. A biblioteca común non cambia.
async function quitarDoProxectoUI(id) {
  const nome = nomeDeFerramenta(id)
  if (!await confirmar(t('proxectos.quitar.confirmar', 'Quitar «{nome}» do proxecto? Bórranse a súa copia e os datos que levase neste proxecto.', { nome }) + ' ' + t('catalogo.quitar.biblioteca', 'A biblioteca común non cambia: podes volver reutilizalo.'))) return
  for (const c of controladores.values()) if (c !== controladores.get(id)) await c.volcarBorrador()
  controladores.get(id)?.descartar()
  await QuitarDoProxecto(proxectoAberto, id)
  await cambiarProxecto(proxectoAberto, { volcar: false })
  setStatus(t('catalogo.quitado', 'Quitado do proxecto: {nome}', { nome }))
}

// ── Ámbito: que se xera e imprime (só nas actas) ──
let ambitosPintados = ''
// ── Barra dos problemas (semente, iteracións, solucións), coma en Yang ──
// Só se ve nun formulario con campos de problemas. Os valores van no
// documento (plan.xeracion): gárdanse co ficheiro e entran no desfacer.
const barraProblemas = document.getElementById('barraProblemas')
const probSemente = document.getElementById('probSemente')
const probIteracions = document.getElementById('probIteracions')
const probSolucions = document.getElementById('probSolucions')

function actualizarBarraProblemas() {
  if (!barraProblemas) return
  const f = informes?.ferramenta
  const ver = Boolean(f?.esFormulario && haiProblemas(f.esquema))
  barraProblemas.hidden = !ver
  if (!ver) return
  const x = xeracionDe(informes.getPlan())
  if (document.activeElement !== probSemente) probSemente.value = String(x.semente)
  if (document.activeElement !== probIteracions) probIteracions.value = String(x.variantes)
  probSolucions.value = x.solucions
}

function cambiarXeracion() {
  if (!informes?.ferramenta?.esFormulario) return
  const plan = informes.getPlan()
  const xeracion = {
    semente: Math.max(0, parseInt(probSemente.value, 10) || 0),
    variantes: Math.min(50, Math.max(1, parseInt(probIteracions.value, 10) || 1)),
    solucions: probSolucions.value,
  }
  const antes = xeracionDe(plan)
  if (antes.semente === xeracion.semente && antes.variantes === xeracion.variantes && antes.solucions === xeracion.solucions) return
  informes.substituirPlan({ ...plan, xeracion })
}
probSemente?.addEventListener('change', cambiarXeracion)
probSemente?.addEventListener('input', () => { if (probSemente.value !== '') cambiarXeracion() })
probIteracions?.addEventListener('change', cambiarXeracion)
probSolucions?.addEventListener('change', cambiarXeracion)

function actualizarAmbitos() {
  actualizarBarraProblemas()
  if (!ambitoWrap || !ambitoSelect || !informes) return
  const f = informes.ferramenta
  ambitoWrap.hidden = !f.ambitosDisponibles
  if (!f.ambitosDisponibles) return
  const opcions = f.ambitosDisponibles(informes.getPlan())
  if (informes.getAmbito() !== 'curso' && !opcions.some((o) => o.valor === informes.getAmbito())) informes.setAmbito('curso')
  const assinatura = JSON.stringify([idiomaActual, informes.getAmbito(), opcions])
  if (assinatura === ambitosPintados) return
  ambitosPintados = assinatura
  ambitoSelect.replaceChildren(
    new Option(t('actas.ambito.curso', 'Todas as actas do curso'), 'curso'),
    ...opcions.map((o) => new Option(t('actas.ambito.acta', 'Acta n.º {n} · {mes} {ano}', { n: o.num, mes: o.nome, ano: o.ano }), o.valor)),
  )
  ambitoSelect.value = informes.getAmbito()
}

if (ambitoSelect) {
  ambitoSelect.addEventListener('change', () => {
    const tiñaResultado = informes.temResultado()
    informes.setAmbito(ambitoSelect.value)
    // Cambiar o que se imprime refai o resultado, se xa se tiña un.
    if (tiñaResultado) run()
  })
}
ferramentasNav?.addEventListener('click', (e) => {
  const b = e.target.closest('.ferramenta-btn[data-ferramenta]')
  if (b) cambiarFerramenta(b.dataset.ferramenta)
})

// run compila o informe co compilador escollido en Opcións (mesmo motor real
// que o PDF final) e amosa cada páxina como imaxe na previsualización - así
// o que se ve é exactamente o que se vai imprimir/entregar.
function run() {
  return informes && haiFerramenta() ? informes.xerar() : Promise.resolve(false)
}

// Sen ningunha ferramenta á vista (pantalla de benvida) a barra vese enteira,
// pero o que traballa sobre un documento (novo, abrir, gardar, desfacer,
// xerar, exportar) non fai nada: non hai documento.
const haiFerramenta = () => !document.body.classList.contains('sen-ferramenta') && !document.body.classList.contains('inicio')
const conFerramenta = (fn) => (...args) => (haiFerramenta() ? fn(...args) : undefined)

function gardadoOk() {
  if (xerarAoGardarActivo) run()
}

async function newFile() {
  await informes.novo()
}

// gardarAgora (Ctrl+S): Lúa xa garda só; isto escribe todo xa e, se está
// activado en Opcións, volve xerar.
async function gardarAgora() {
  for (const c of controladores.values()) if (c !== informes) await c.volcarBorrador()
  if (await informes.gardar()) gardadoOk()
}

async function printPreview() {
  if (!souXanelaResultado && !informes.temResultado()) {
    setStatus(t('status.generateBeforePrint', 'Xera antes de imprimir'))
    return
  }
  // Imprímese o que se ve: se o ámbito ou o curso cambiaron desde o último
  // Xerar, primeiro refaise o resultado.
  if (!souXanelaResultado && !informes.estaAoDia()) {
    if (!await run()) return
  }
  preview.contentWindow.focus()
  preview.contentWindow.print()
}

// accionRemota (só na xanela de RESULTADO desacoplada): chama directamente un
// dos bindings AccionResultado{PDF,Tex,MD,Docx,Odt} (Go, xanelaresultado.go) -
// esta xanela non ten editor propio, así que non pode compilar nada ela soa;
// o backend exporta o último informe xerado e devolve {path, warnings}, ou
// rexeita a promesa se hai erro (ex. "xera antes de exportar").
async function accionRemota(accionFn, btn, statusCompiling, errKey, errFallback) {
  btn.disabled = true
  setStatus(statusCompiling)
  showWarnings([])
  log('[XERAR] ' + statusCompiling)
  try {
    const result = await accionFn()
    setStatusGardado(result.path ? t('status.exported', 'Exportado: ') + result.path : t('status.gardadoLua', 'Gardado en Lúa'), result.path)
    log('[XERAR] ' + (result.path ? 'Exportado: ' + result.path : 'Feito.'))
    logAvisos(result.warnings)
  } catch (err) {
    setStatus(t(errKey, errFallback))
    showWarnings([mensaxeErro(err)])
    log('[XERAR] ERRO: ' + err)
  } finally {
    btn.disabled = false
  }
}

function exportPDF() {
  if (souXanelaResultado) {
    return accionRemota(AccionResultadoPDF, btnPDF, t('status.exporting', 'Exportando...'),
      'status.errorCompilingPdf', 'Erro ao compilar PDF')
  }
  return informes.exportarPDF(btnPDF)
}

function exportTex() {
  if (souXanelaResultado) {
    return accionRemota(AccionResultadoTex, btnExportTex, t('status.exporting', 'Exportando...'),
      'status.errorExporting', 'Erro ao exportar')
  }
  return informes.exportarTex(btnExportTex)
}

function exportMarkdown() {
  if (souXanelaResultado) {
    return accionRemota(AccionResultadoMD, btnExportMarkdown, t('status.compilingMarkdown', 'Compilando Markdown...'),
      'status.errorExportingMarkdown', 'Erro ao exportar Markdown')
  }
  return informes.exportarMarkdown(btnExportMarkdown)
}

function exportDocx() {
  if (souXanelaResultado) {
    return accionRemota(AccionResultadoDocx, btnExportDocx, t('status.compilingDocx', 'Compilando Word (.docx)...'),
      'status.errorExportingDocx', 'Erro ao exportar Word')
  }
  return informes.exportarDocx(btnExportDocx)
}

function exportOdt() {
  if (souXanelaResultado) {
    return accionRemota(AccionResultadoOdt, btnExportOdt, t('status.compilingOdt', 'Compilando OpenDocument (.odt)...'),
      'status.errorExportingOdt', 'Erro ao exportar OpenDocument')
  }
  return informes.exportarOdt(btnExportOdt)
}

// ── Plantillas (estilos: cabeceira, pé, logotipo, datos do centro) ───────
// A plantilla activa vive no backend (plantillas.go), non aquí:
// XerarInformeCronograma e as exportacións resólvena elas soas. O frontend só
// escolle cal é (selector da barra) e amosa a biblioteca. Por iso cambiar de
// plantilla NON toca ningunha das peticións de xeración - só chama a
// EscollerPlantilla e volve xerar.
const apiPlantillas = {
  listar: () => ListarPlantillas(),
  fragmentosPorDefecto: () => FragmentosInformePorDefecto(),
  gardar: (p) => GardarPlantilla(p),
  eliminar: (id) => EliminarPlantilla(id),
  duplicar: (id) => DuplicarPlantilla(id),
  // O segundo argumento é o documento (a ferramenta activa) que lembra a plantilla.
  escoller: (id, ferramenta) => EscollerPlantillaDocumento(id, ferramenta),
  subirImaxe: (req) => SubirImaxePlantilla(req),
  vistaImaxe: (id, nome) => PlantillaImaxeVista(id, nome),
  eliminarImaxe: (id, nome) => EliminarImaxePlantilla(id, nome),
  // `ferramenta` ("cronograma" ou "actas"): o documento de mostra da vista previa.
  // `ferramenta`: "actual" (o documento que se está editando, `documento` é a súa
  // petición), "cronograma" ou "actas" (documentos de mostra).
  previsualizar: (p, ferramenta, documento) => (ferramenta === 'actual' && documento
    ? PrevisualizarPlantillaDocumento(p, documento) : PrevisualizarPlantillaDe(p, ferramenta === 'actas' ? 'actas' : 'cronograma')),
  previsualizarOfi: (p, formato, ferramenta, documento) => (ferramenta === 'actual' && documento
    ? PrevisualizarPlantillaOfimaticaDocumento(p, formato, documento) : PrevisualizarPlantillaOfimaticaDe(p, formato, ferramenta === 'actas' ? 'actas' : 'cronograma')),
  // A petición do documento en edición (só na xanela principal; a do editor recíbea ao abrirse).
  documentoActual: () => (document.body.classList.contains('sen-ferramenta') ? null : (informes?.peticionActual?.() ?? null)),
  fontes: () => ListarFontesSistema(),
  // Deseño do PDF (deseno/editor.js).
  combinacions: () => CombinacionsLetras(),
  coresImaxe: (datos) => CoresDaImaxe(datos),
  importarDeseno: (nome, datosB64) => ImportarDesenoDeFicheiro(nome, datosB64),
  estiloElemento: (d, id) => EstiloElementoDeseno(d, id),
  skillDeseno: () => SkillDeseno(),
  cancelarVista: () => CancelarVistaPrevia(),
  comprobarElemento: (d, id, latex) => ComprobarElementoDeseno(d, id, latex),
  gardarDesenoFicheiro: (d, nome) => GardarDesenoFicheiro(d, nome),
  lerDesenoFicheiro: (texto) => LerDesenoFicheiro(texto),
  aplicarDesenoATodas: (d) => AplicarDesenoATodas(d),
  // O editor abre nunha xanela propia (a nova xanela non abre outras). Sen
  // xanelas nativas (proba en navegador) devolve false e o editor ábrese a
  // pantalla completa nesta mesma xanela.
  abrirXanela: souXanelaPlantilla ? null : (id, datos) => AbrirXanelaPlantilla(id, datos),
  // Cada Aplicar/Gardar do editor avisa á xanela principal para que refresque
  // a biblioteca e o informe (ver plantillaGardada).
  aplicada: (p) => (souXanelaPlantilla ? PlantillaXanelaAplicada(p.id) : plantillaGardada(p.id)),
  // Asistente de IA do editor (ámbito "plantilla"): bindings, adxuntos e idioma.
  asistente: asistenteDaPlantilla(),
  // Exportar e Importar abren Trasladar (o único sitio para sacar e meter).
  exportar: (id) => { abrirTrasladarUI({ lapela: 'sacar', marcados: { plantillas: [id] } }); return '' },
  importar: () => { abrirTrasladarUI({ lapela: 'meter' }); return null },
}

// refrescarSelectorPlantillas volve pintar o desplegable da barra co que
// haxa gardado. Chámase ao arrincar e cada vez que a biblioteca cambia
// (crear/editar/eliminar/importar pode mudar tanto os nomes coma cal está
// activa).
async function refrescarSelectorPlantillas() {
  if (!plantillaSelect) return
  try {
    const estado = await ListarPlantillas()
    plantillaSelect.innerHTML = ''
    const ningunha = document.createElement('option')
    ningunha.value = ''
    ningunha.textContent = t('plantillas.none', 'Ningunha')
    plantillaSelect.appendChild(ningunha)
    // Bifurcar unha de fábrica (ver bifurcarDeFabrica, plantillas.go) deixa
    // dúas entradas co MESMO nome: a copia propia e a de fábrica que agora
    // sobrevive. Na biblioteca distínguense polo distintivo "de exemplo",
    // pero aquí só hai texto, así que se marca a de fábrica - e só cando hai
    // empate, para non ensuciar o caso normal.
    const veces = new Map()
    for (const p of estado.plantillas || []) veces.set(p.nome, (veces.get(p.nome) || 0) + 1)
    for (const p of estado.plantillas || []) {
      const opt = document.createElement('option')
      opt.value = p.id
      opt.textContent = p.deFabrica && veces.get(p.nome) > 1
        ? `${p.nome} (${t('plantillas.builtin', 'de exemplo')})`
        : p.nome
      plantillaSelect.appendChild(opt)
    }
    plantillaSelect.value = estado.activa || ''
  } catch (err) {
    showWarnings([mensaxeErro(err)])
  }
}

// cambiarPlantilla aplica a escolla e volve xerar SE xa había un resultado
// na pantalla: a plantilla é precisamente o que se ve, así que deixar a
// vista previa coa maqueta anterior sería enganoso. Nun curso aínda sen
// xerar non se compila nada.
async function cambiarPlantilla(id) {
  try {
    await EscollerPlantillaDocumento(id, haiFerramenta() ? ferramentaActiva : '')
    setStatus(id
      ? t('status.templateApplied', 'Plantilla aplicada')
      : t('status.templateNone', 'Sen plantilla'))
    if (informes.temResultado()) await run()
  } catch (err) {
    setStatus(t('status.errorTemplate', 'Erro coa plantilla'))
    showWarnings([mensaxeErro(err)])
  }
}

// sincronizarPlantillaDoDocumento: cada documento do proxecto lembra a súa
// plantilla (PlantillaDoDocumento, plantillas.go): ao cambiar de documento ou
// de proxecto recupérase. Un documento que nunca tivo deixa a activa.
async function sincronizarPlantillaDoDocumento() {
  if (!plantillaSelect || !proxectoAberto || !haiFerramenta()) return
  try {
    await PlantillaDoDocumento(ferramentaActiva)
    await refrescarSelectorPlantillas()
  } catch (err) {
    showWarnings([mensaxeErro(err)])
  }
}

// plantillaGardada: o editor de plantillas (esta xanela ou a súa xanela propia)
// gardou `id`. Refréscase o selector e, se é a plantilla en uso (ou a gardada
// pasou a selo, coma ao editar unha de exemplo), volve xerarse o informe para
// que se vexa o cambio sen pechar o editor.
async function plantillaGardada(id) {
  if (!plantillaSelect) return
  const anterior = plantillaSelect.value
  await refrescarSelectorPlantillas()
  const activa = plantillaSelect.value
  if (informes && informes.temResultado() && activa && (activa === id || activa !== anterior)) await run()
}

function abrirPlantillas() {
  abrirBibliotecaPlantillas({
    t,
    api: apiPlantillas,
    ficheiroActual: () => (haiFerramenta() ? ferramentaActiva : ''),
    onEscoller: async () => {
      const anterior = plantillaSelect ? plantillaSelect.value : ''
      await refrescarSelectorPlantillas()
      // Se a biblioteca cambiou a plantilla en uso, aplícase igual ca dende
      // o selector (volver xerar incluído).
      if (plantillaSelect && plantillaSelect.value !== anterior && informes.temResultado()) await run()
    },
  })
}

// ── Opcións ──────────────────────────────────────────────────────────────

// IA_PRESETS mapea cada opción do selector "Provedor de IA" a que formato
// de fío fala (provedor, ver ia_client.go) e que enderezo de servidor usar
// - así só se escolle un nome de marca recoñecible, nunca se escribe un
// enderezo nin se sabe que "Qwen" e "ChatGPT" acaban falando o mesmo
// protocolo por baixo. "outro" queda cun enderezo baleiro a propósito (o
// único caso onde SI hai que escribilo á man: un provedor "compatible con
// OpenAI" que non está na lista).
const IA_PRESETS = {
  gemini: { provedor: 'gemini', baseURL: '', modeloExemplo: 'gemini-2.0-flash' },
  openai: { provedor: 'openai', baseURL: '', modeloExemplo: 'gpt-4o-mini' },
  anthropic: { provedor: 'anthropic', baseURL: '', modeloExemplo: 'claude-3-5-haiku-latest' },
  qwen: { provedor: 'openai', baseURL: 'https://dashscope.aliyuncs.com/compatible-mode/v1', modeloExemplo: 'qwen-plus' },
  perplexity: { provedor: 'openai', baseURL: 'https://api.perplexity.ai', modeloExemplo: 'sonar' },
  outro: { provedor: 'openai', baseURL: '', modeloExemplo: 'o-nome-exacto-do-modelo' },
}

// aplicarPresetIA actualiza o placeholder do modelo e (agás cando se está a
// prerrechear dende axustes xa gardados) o enderezo do servidor, e amosa a
// fila de "Enderezo do servidor" só para "outro" - os demais presets xa
// levan o seu enderezo fixo, non hai nada que ver nin tocar.
function aplicarPresetIA(preset, manterBaseURLGardado) {
  const p = IA_PRESETS[preset] || IA_PRESETS.gemini
  optIAModel.placeholder = p.modeloExemplo
  if (!manterBaseURLGardado) optIABaseURL.value = p.baseURL
  optIABaseURLRow.classList.toggle('hidden', preset !== 'outro')
}

optIAPreset.addEventListener('change', () => aplicarPresetIA(optIAPreset.value, false))

// aplicarVisibilidadeExportadores agocha/amosa os botóns da barra da
// previsualización (btnPDF/btnExportTex/.../btnExportOdt) segundo o que se
// marcou en Opcións. PDF/Tex/Markdown usan "!== false": un settings.json
// vello, ou quen nunca abriu Opcións, non ten estas chaves (undefined) e os
// tres deben seguir visibles coma sempre. DOCX/ODT son o contrario - por
// defecto agochados - así que "!!" abonda.
function aplicarVisibilidadeExportadores(s) {
  btnPDF.classList.toggle('hidden', s.exportarPdf === false)
  btnExportTex.classList.toggle('hidden', s.exportarTex === false)
  btnExportMarkdown.classList.toggle('hidden', s.exportarMarkdown === false)
  btnExportDocx.classList.toggle('hidden', !s.exportarDocx)
  btnExportOdt.classList.toggle('hidden', !s.exportarOdt)
}

// Módulo Buxo (Opcións → Módulos): activar/desactivar e estado da descarga.
function pintarBuxo(e) {
  optBuxoActivo.checked = !!e.activo
  btnBuxoActualizar.disabled = !!e.descargando
  let txt
  if (e.descargando) txt = t('options.buxo.estado.descargando', 'Descargando…')
  else if (e.erro) txt = t('options.buxo.estado.erro', 'Erro: {msg}', { msg: e.erro })
  else if (!e.activo) txt = t('options.buxo.estado.desactivado', 'Desactivado')
  else if (!e.instalado) txt = t('options.buxo.estado.non', 'Non instalado')
  else {
    txt = t('options.buxo.estado.instalado', 'Instalado: versión {v}', { v: e.version || '?' })
    if (e.versionRemota && e.versionRemota !== e.version) txt += ' · ' + t('options.buxo.estado.novo', 'dispoñible a versión {v}', { v: e.versionRemota })
  }
  if (e.activo && !e.iaConfigurada) txt += ' · ' + t('options.buxo.estado.sen_ia', 'Configura unha clave de IA para poder usalo.')
  buxoEstadoEl.textContent = txt
}

async function refrescarBuxo() {
  try { pintarBuxo(await BuxoEstadoActual()) } catch { /* sen módulo: Opcións segue valendo */ }
}

optBuxoActivo.addEventListener('change', async () => {
  try { await BuxoActivar(optBuxoActivo.checked) } catch (err) { showWarnings([mensaxeErro(err)]) }
  refrescarBuxo()
})
btnBuxoActualizar.addEventListener('click', async () => {
  try { await BuxoActualizar() } catch { /* o erro queda no estado */ }
  refrescarBuxo()
})
Events.On('buxo_cambiou', (e) => pintarBuxo(e.data))

// Maxima en Opcións: estado, detectar e instalar (coma LaTeX).
async function pintarEstadoMaxima() {
  if (!optMaximaEstado) return
  try {
    const e = await CheckMaxima()
    optMaximaEstado.textContent = e.atopado ? t('options.maxima.ok', '✓ Maxima atopado: {ruta}', { ruta: e.ruta }) : t('options.maxima.non', 'Maxima non está instalado.')
  } catch { optMaximaEstado.textContent = '' }
}
btnDetectarMaxima?.addEventListener('click', async () => {
  const r = await DetectMaxima()
  optMaximaPath.value = r || ''
  optMaximaEstado.textContent = r ? t('options.maxima.ok', '✓ Maxima atopado: {ruta}', { ruta: r }) : t('options.maxima.non', 'Maxima non está instalado.')
})
btnInstalarMaxima?.addEventListener('click', async () => {
  btnInstalarMaxima.disabled = true
  const texto = btnInstalarMaxima.textContent
  btnInstalarMaxima.textContent = t('status.installingMaxima', '⏳ Instalando… (pode pedir contrasinal)')
  try {
    const r = await InstallMaxima()
    if (!r.success) showWarnings([t('options.maxima.fallou', 'Non se puido instalar Maxima.'), (r.log || '').slice(-600)])
    await pintarEstadoMaxima()
    if (r.success) optMaximaPath.value = (await CheckMaxima()).ruta || ''
    for (const c of controladores.values()) c.refrescarIdioma?.()
  } finally {
    btnInstalarMaxima.disabled = false
    btnInstalarMaxima.textContent = texto
  }
})

async function openOptions() {
  try {
    const s = await GetSettings()
    refrescarBuxo()
    optIAPreset.value = s.iaPreset || 'gemini'
    aplicarPresetIA(optIAPreset.value, true) // manter o enderezo xa gardado, non o do preset
    optIAApiKey.value = s.iaApiKey || ''
    optIAModel.value = s.iaModel || ''
    optIABaseURL.value = s.iaBaseUrl || ''
    optIAMaxTokens.value = s.iaMaxTokens || 0
    optIATimeout.value = s.iaTimeoutSegundos || 240
    optLatexEngine.value = s.latexEngine || ''
    optMaximaPath.value = s.maximaPath || ''
    optMaximaDecimais.value = s.maximaDecimais || ''
    pintarEstadoMaxima()
    optXerarAoGardar.checked = s.xerarAoGardar !== false
    optIdioma.value = s.idioma || idiomaPiztu || 'gl'
    optExportPDF.checked = s.exportarPdf !== false
    optExportTex.checked = s.exportarTex !== false
    optExportMarkdown.checked = s.exportarMarkdown !== false
    optExportDocx.checked = !!s.exportarDocx
    optExportOdt.checked = !!s.exportarOdt
    optionsOverlay.classList.remove('hidden')
  } catch (err) {
    setStatus(t('status.errorLoadingOptions', 'Erro ao cargar opcións'))
    showWarnings([mensaxeErro(err)])
  }
}

function closeOptions() {
  optionsOverlay.classList.add('hidden')
}

// refrescarIdioma aplica un cambio de idioma sen reiniciar Lua: retraduce
// o DOM estático (aplicarTraducions) e o que se pinta en JS (o editor do
// cronograma, o selector de plantillas, os avisos con texto interpolado).
function refrescarIdioma(code) {
  cargarIdioma(code)
  aplicarTraducions()
  checkLatex() // latexWarningText leva texto dinámico interpolado, non só data-i18n
  refrescarSelectorPlantillas() // as opcións do selector píntanse en JS, non levan data-i18n
  pintarActualizacionLua() // sobreVersion/sobreActualizacion tamén levan texto interpolado
  // Os editores das ferramentas pintan as etiquetas en JS.
  for (const c of controladores.values()) c.refrescarIdioma()
  actualizarAmbitos()
  if (asistente) asistente.refrescarIdioma() // o panel do asistente tamén
}

async function saveOptions() {
  try {
    // valorSeleccionado é o que se ve e se aplica; idiomaAGardar é o que se
    // persiste. Se o escollido coincide co idioma que Lua xa herdaría de
    // Piztu, gárdase "" (segue en modo automático: se Piztu cambia de idioma
    // máis adiante, Lua cámbiao con el). Só se garda un valor concreto cando
    // se escolleu algo DISTINTO — iso si é unha elección explícita, que debe
    // prevalecer sempre, mesmo sobre futuros cambios en Piztu.
    const valorSeleccionado = optIdioma.value
    const idiomaAGardar = valorSeleccionado === idiomaPiztu ? '' : valorSeleccionado
    await SaveSettings({
      iaPreset: optIAPreset.value,
      iaProvedor: (IA_PRESETS[optIAPreset.value] || IA_PRESETS.gemini).provedor,
      iaApiKey: optIAApiKey.value.trim(),
      iaModel: optIAModel.value.trim(),
      iaBaseUrl: optIABaseURL.value.trim(),
      iaMaxTokens: parseInt(optIAMaxTokens.value, 10) || 0,
      iaTimeoutSegundos: parseInt(optIATimeout.value, 10) || 240,
      latexEngine: optLatexEngine.value,
      maximaPath: optMaximaPath.value.trim(),
      maximaDecimais: Math.max(0, parseInt(optMaximaDecimais.value, 10) || 0),
      xerarAoGardar: optXerarAoGardar.checked,
      idioma: idiomaAGardar,
      exportarPdf: optExportPDF.checked,
      exportarTex: optExportTex.checked,
      exportarMarkdown: optExportMarkdown.checked,
      exportarDocx: optExportDocx.checked,
      exportarOdt: optExportOdt.checked,
    })
    xerarAoGardarActivo = optXerarAoGardar.checked
    aplicarVisibilidadeExportadores({
      exportarPdf: optExportPDF.checked,
      exportarTex: optExportTex.checked,
      exportarMarkdown: optExportMarkdown.checked,
      exportarDocx: optExportDocx.checked,
      exportarOdt: optExportOdt.checked,
    })
    if (valorSeleccionado !== idiomaActual) refrescarIdioma(valorSeleccionado)
    else for (const c of controladores.values()) if (c.ferramenta?.esFormulario) c.refrescarIdioma?.() // IA e Maxima poden cambiar
    closeOptions()
    checkLatex() // o compilador escollido pode ter cambiado
    if (asistente) asistente.refrescar() // o provedor de IA pode ter cambiado (logo, modelo)
    setStatus(t('status.optionsSaved', 'Opcións gardadas'))
  } catch (err) {
    setStatus(t('status.errorSavingOptions', 'Erro ao gardar opcións'))
    showWarnings([mensaxeErro(err)])
  }
}

// --- Aviso de LaTeX/pdftoppm ---
// Comproba o motor LaTeX escollido en Opcións (XerarInformeCronograma xa
// falla cun erro que menciona ese mesmo motor se falta) e pdftoppm (necesario
// para a vista previa páxina a páxina). Chámase ao arrincar e despois de
// calquera instalación - non hai sondaxe en segundo plano, así que se se
// instala á man nun terminal á parte, o aviso desaparece cando se volva
// comprobar (arrinque seguinte, ou premendo calquera acción que chame
// checkLatex()).
async function checkLatex() {
  try {
    const s = await CheckLatexDeps()
    if (s.engineFound && s.pdftoppmFound) {
      latexWarning.classList.add('hidden')
      return
    }
    const faltan = []
    if (!s.engineFound) faltan.push(s.engine)
    if (!s.pdftoppmFound) faltan.push('pdftoppm (poppler-utils)')
    latexWarningText.textContent = t('status.latexMissing', '⚠ Falta {deps} neste equipo. É necesario para xerar/previsualizar PDFs.', {
      deps: faltan.join(t('status.latexMissing.and', ' e ')),
    })
    latexWarning.classList.remove('hidden')
  } catch (err) {
    // ignore - o aviso simplemente non se amosa se non se puido comprobar
  }
}

async function installLatexNow() {
  btnInstallLatex.disabled = true
  btnInstallLatex.textContent = t('status.installingLatex', '⏳ Instalando... (pode pedir contrasinal, tarda uns minutos)')
  try {
    const result = await InstallLatex()
    if (result.success) {
      setStatus(t('status.latexInstalled', 'LaTeX instalado correctamente.'))
      checkLatex()
    } else {
      setStatus(t('status.autoInstallFailed', 'Non se puido instalar automaticamente'))
      showWarnings([result.log || t('status.unknownError', 'erro descoñecido')])
    }
  } catch (err) {
    setStatus(t('status.errorInstalling', 'Erro ao instalar'))
    showWarnings([mensaxeErro(err)])
  } finally {
    btnInstallLatex.disabled = false
    btnInstallLatex.textContent = t('banner.latex.install', '⇩ Instalar automaticamente')
  }
}

// --- Progreso das instalacións (lua:instalacion-progreso, installer.go) ---
//
// InstallLatex é síncrono: o await de arriba non devolve nada ata que
// remata. Iso vale para un `apt install` de vinte segundos, pero non para `brew install --cask mactex-no-gui` en macOS, que
// son preto de 6 GB e pode pasar media hora sen que Homebrew emita unha soa
// liña. Sen isto, o botón queda cun "⏳ Instalando…" fixo e o natural é
// pensar que Lua colgou e pechalo a metade da descarga.
//
// O proceso Go emite un evento decontado e despois cada 30 segundos; aquí só
// se reescribe o texto do botón correspondente.
const botonsInstalacion = {
  latex: () => btnInstallLatex,
}

Events.On('lua:instalacion-progreso', (e) => {
  const p = e.data || {}
  const boton = botonsInstalacion[p.que]?.()
  // Se xa rematou, non tocar nada: o finally de install*Now() é quen
  // restaura o texto orixinal do botón.
  if (!boton || p.completo) return

  if (p.mb > 0) {
    boton.textContent = t('status.installingProgress.mb', '⏳ Descargando… {mb} MB ({min} min)', {
      mb: p.mb,
      min: p.minutos,
    })
  } else if (p.minutos > 0) {
    boton.textContent = t('status.installingProgress', '⏳ Instalando… ({min} min)', { min: p.minutos })
  }
})

// ── Modo dúas xanelas (botón "Desacoplar", ver xanelaresultado.go) ───────
// Con dous monitores, o resultado pódese levar a unha xanela á parte e
// deixar o editor na outra. Acoplar de volta = pechar a xanela de resultado.

// entrarModoResultado configura ESTA xanela coma a "xanela de resultado":
// agocha todo agás o panel de resultado (que pasa a ocupar a xanela enteira,
// sen editor ao lado) e escoita lua:resultado-actualizado (Events, emitido
// por XerarInformeCronograma no proceso Go cada vez que se xera un informe)
// para manter as páxinas actualizadas.
function entrarModoResultado() {
  latexWarning.classList.add('hidden')
  document.getElementById('toolbar').classList.add('hidden')
  paneEditor.classList.add('hidden')
  paneSplitter.classList.add('hidden')
  // O selector de plantilla, a biblioteca e os axustes do texto viven na
  // cabeceira do resultado, pero na xanela desacoplada non hai nada que
  // escoller (xérase sempre dende a xanela editor): agóchanse.
  document.getElementById('plantillaWrap')?.classList.add('hidden')
  btnPlantillas?.classList.add('hidden')
  btnAxustesTexto?.classList.add('hidden')
  preview.srcdoc = pagesToPreviewHTML([])

  let ultimaVersion = -1
  function aplicarEstado(estado) {
    if (estado.version === ultimaVersion) return
    ultimaVersion = estado.version
    preview.srcdoc = pagesToPreviewHTML(estado.pageImages)
    logAvisos(estado.warnings)
  }

  // Subscribirse ANTES de pedir o snapshot inicial - se chega un Xerar()
  // entre medias, o evento non se perde.
  Events.On('lua:resultado-actualizado', (e) => aplicarEstado(e.data))
  GetEstadoResultado().then(aplicarEstado).catch(() => {})
}

// activarModoDobreEditor agocha o panel de resultado NESTA xanela (o editor)
// cando o resultado vive na outra xanela, amosando no seu lugar un aviso +
// un botón para reabrila por se se pechou sen querer.
function activarModoDobreEditor() {
  panePreview.classList.add('hidden')
  panePreviewPlaceholder.classList.remove('hidden')
  tiraAnchoCompleto.classList.remove('hidden')
}

// ── Editor ao 100% do ancho (botón da tira esquerda) ─────────────────────
// Co resultado noutra xanela, o editor pode ocupar todo o ancho da pantalla:
// agóchase o aviso do resultado e, se a xanela non estaba maximizada, tamén
// se maximiza (e ao volver ao ancho normal desmaximízase só se a maximizou
// este botón, non o profesorado).
let anchoCompleto = false
let maximizadaPorMin = false

async function fixarAnchoCompleto(activo) {
  anchoCompleto = activo
  document.body.classList.toggle('editor-ancho-completo', activo)
  btnAnchoCompleto.setAttribute('aria-pressed', String(activo))
  const titulo = activo
    ? t('preview.remoto.anchoNormal', 'Volver ao ancho normal do editor')
    : t('preview.remoto.anchoCompleto', 'Ampliar o editor ao 100% do ancho da pantalla')
  btnAnchoCompleto.title = titulo
  try {
    if (activo) {
      if (!(await Window.IsMaximised())) { await Window.Maximise(); maximizadaPorMin = true }
    } else if (maximizadaPorMin) {
      maximizadaPorMin = false
      await Window.UnMaximise()
    }
  } catch { /* fóra de Wails (proba en navegador): abonda co CSS */ }
}

btnAnchoCompleto.addEventListener('click', () => fixarAnchoCompleto(!anchoCompleto))

// saírModoDobreEditor reverte activarModoDobreEditor - chámase ao recibir
// lua:resultado-pechado (Events, emitido por WindowClosing en
// xanelaresultado.go ao pechar a xanela de resultado). preview.srcdoc xa
// está actualizado (Xerar sempre o escribe, mesmo con esta xanela agochada),
// así que abonda con amosala de novo.
function saírModoDobreEditor() {
  if (anchoCompleto) fixarAnchoCompleto(false)
  tiraAnchoCompleto.classList.add('hidden')
  panePreviewPlaceholder.classList.add('hidden')
  panePreview.classList.remove('hidden')
}

btnReabrirResultado.addEventListener('click', async () => {
  try {
    await AbrirXanelaResultado()
  } catch (err) {
    setStatus(mensaxeErro(err))
  }
})

// asistenteDaTarefa: o que precisa a pantalla de edición dunha tarefa para o seu
// asistente de IA (tarefa-editor.js): os bindings e o idioma da interface.
function asistenteDaTarefa() {
  return {
    api: { IAAsistenteEstado, IAFerramentasDoAmbito, IATurnoAsistente, IAChamarFerramenta, BovedaEstadoDe, BovedaProxectoConectar, BovedaConectar, BovedaEscoller, IAFerramentasBoveda },
    idioma: () => idiomaActual,
  }
}

// asistenteDoFormularioDatos: o que precisa o «Asistente datos» dun formulario
// (formularios/editor.js, ámbito «formulario_datos»): os bindings normais máis
// os dos adxuntos, para que se lle poidan achegar documentos (por exemplo, un
// temario ou un exame xa feito) que teña en conta ao encher datos de proba.
function asistenteDoFormularioDatos() {
  return {
    // formulario_xerar: compila o documento aberto (o do asistente é o activo).
    xerar: async () => {
      const ok = await informes.xerar()
      return { ok: ok !== false && informes.ultimoXerar().ok, ...informes.ultimoXerar() }
    },
    api: {
      IAAsistenteEstado, IAFerramentasDoAmbito, IATurnoAsistente, IAChamarFerramenta,
      AdxuntoGardar, AdxuntoEliminar, AdxuntosLimpar, AdxuntoVista, AdxuntoDataURI,
      BovedaEstadoDe, BovedaProxectoConectar, BovedaConectar, BovedaEscoller, IAFerramentasBoveda,
    },
    idioma: () => idiomaActual,
  }
}

// asistenteDaPlantilla: o que precisa o editor de plantillas para o seu asistente
// de IA (plantillas-editor.js): os bindings (tamén os dos adxuntos: logotipos,
// documentos modelo) e o idioma da interface.
function asistenteDaPlantilla() {
  return {
    api: {
      IAAsistenteEstado, IAFerramentasDoAmbito, IATurnoAsistente, IAChamarFerramenta,
      AdxuntoGardar, AdxuntoEliminar, AdxuntosLimpar, AdxuntoVista, AdxuntoDataURI,
      BovedaEstadoDe, BovedaProxectoConectar, BovedaConectar, BovedaEscoller, IAFerramentasBoveda,
    },
    idioma: () => idiomaActual,
  }
}

// ── Xanela do editor de plantillas (ver xanelaplantilla.go) ──────────────
// entrarModoPlantilla configura ESTA xanela coma a do editor de plantillas: só
// ten o editor, a tamaño completo, e pódese levar a outro monitor. Aplicar
// garda sen pechar e avisa á xanela principal (lua:plantilla-aplicada).
async function entrarModoPlantilla() {
  document.body.classList.add('xanela-plantilla')
  latexWarning.classList.add('hidden')
  let s = {}
  try { s = await GetSettings() } catch { /* idioma por defecto */ }
  let idioma = s.idioma || ''
  if (!idioma) { try { idioma = await IdiomaPiztu() } catch { /* nada */ } }
  if (idioma) { cargarIdioma(idioma); aplicarTraducions() }

  const contedor = document.createElement('div')
  contedor.id = 'plantillaXanela'
  contedor.className = 'modal modal--plantilla-editor'
  document.body.append(contedor)

  let datos = null
  try { datos = JSON.parse(await GetDatosXanelaPlantilla(idXanelaPlantilla)) } catch { /* abaixo */ }
  if (!datos) {
    contedor.textContent = t('plantillas.sen.datos', 'Non hai datos desta plantilla. Pecha esta xanela e ábrea de novo.')
    return
  }
  const editor = await montarEditorPlantilla(contedor, {
    t, api: apiPlantillas, plantilla: datos.plantilla || null, abrirIA: !!datos.abrirIA, documento: datos.documento || null,
    aoPechar: () => { PecharXanelaPlantilla(idXanelaPlantilla) },
    aoCambiarSucio: (sucia) => { MarcarXanelaPlantillaSucia(idXanelaPlantilla, sucia) },
  })
  // O botón de pechar do xestor de xanelas con cambios sen aplicar: Go xa o
  // cancelou, pregúntase aquí (mesmo aviso ca a "X" vermella).
  Events.On('lua:plantilla-pedir-pechar', (e) => {
    if (e.data && e.data.id === idXanelaPlantilla) editor.pedirPechar()
  })
}

// ── Xanela do deseñador de formularios (ver xanelaformulario.go) ─────────
// entrarModoFormulario configura ESTA xanela coma a do deseñador: só ten o
// deseñador, a pantalla completa, e pódese levar a outro monitor. Ao gardar ou
// eliminar avisa á xanela principal e pecha.
async function entrarModoFormulario() {
  document.body.classList.add('xanela-formulario')
  latexWarning.classList.add('hidden')
  let s = {}
  try { s = await GetSettings() } catch { /* idioma por defecto */ }
  let idioma = s.idioma || ''
  if (!idioma) { try { idioma = await IdiomaPiztu() } catch { /* nada */ } }
  if (idioma) { cargarIdioma(idioma); aplicarTraducions() }

  const contedor = document.createElement('div')
  contedor.id = 'formularioXanela'
  document.body.append(contedor)

  let datos = null
  try { datos = JSON.parse(await GetDatosXanelaFormulario(idXanelaFormulario)) } catch { /* abaixo */ }
  if (!datos) {
    contedor.textContent = t('formularios.sen.datos', 'Non hai datos deste formulario. Pecha esta xanela e ábrea de novo.')
    return
  }
  let controlDisenador = null
  // O botón de pechar do xestor de xanelas con cambios sen gardar: Go xa o
  // cancelou, pregúntase aquí (mesmo aviso ca a «X» do deseñador).
  Events.On('lua:formulario-pedir-pechar', (e) => {
    if (e.data && e.data.id === idXanelaFormulario) controlDisenador?.pedirPechar()
  })
  const r = await abrirDisenador({
    t, esquema: datos.esquema || null, confirmar, api: apiDisenador(), asistente: asistenteDoDisenador(),
    obterDatos: () => datos.datos || {},
    alvo: contedor,
    aoSucio: (sucia) => { MarcarXanelaFormularioSucia(idXanelaFormulario, sucia) },
    aoPreparado: (c) => { controlDisenador = c },
  })
  if (r?.accion === 'gardado') await FormularioXanelaGardado(JSON.stringify(r))
  else if (r?.accion === 'eliminado') await FormularioXanelaEliminado(JSON.stringify(r))
  PecharXanelaFormulario(idXanelaFormulario)
}

// ── Xanela de edición dunha tarefa (ver xanelatarefa.go) ─────────────────
// entrarModoTarefa configura ESTA xanela coma a de edición dunha tarefa: non
// ten barra nin editor de curso, só a pantalla "Editar tarefa" a tamaño
// completo. O curso vive na xanela principal: Aplicar/Gardar mandan a tarefa
// alá (AplicarTarefaXanela -> evento lua:tarefa-aplicada).
async function entrarModoTarefa() {
  document.body.classList.add('xanela-tarefa')
  latexWarning.classList.add('hidden')
  let s = {}
  try { s = await GetSettings() } catch { /* idioma por defecto */ }
  let idioma = s.idioma || ''
  if (!idioma) { try { idioma = await IdiomaPiztu() } catch { /* nada */ } }
  if (idioma) { cargarIdioma(idioma); aplicarTraducions() }

  const contedor = document.createElement('div')
  contedor.id = 'tarefaXanela'
  contedor.className = 'modal crono-modal'
  document.body.append(contedor)

  let datos = null
  try { datos = JSON.parse(await GetDatosXanelaTarefa(idXanelaTarefa)) } catch { /* abaixo */ }
  if (!datos) {
    contedor.textContent = t('informes.tarefa.sen.datos', 'Non hai datos desta tarefa. Pecha esta xanela e ábrea de novo.')
    return
  }
  const editor = montarEditorTarefa(contedor, {
    t, lerImaxe: normalizarImaxe, ...datos, asistente: asistenteDaTarefa(),
    aoAplicar: (r) => AplicarTarefaXanela(idXanelaTarefa, JSON.stringify(r)),
    aoPechar: () => { PecharXanelaTarefa(idXanelaTarefa) },
    aoCambiarSucio: (sucia) => { MarcarXanelaTarefaSucia(idXanelaTarefa, sucia) },
  })
  // O botón de pechar do xestor de xanelas con cambios sen aplicar: Go xa o
  // cancelou, pregúntase aquí (mesmo aviso ca a "X" vermella).
  Events.On('lua:tarefa-pedir-pechar', (e) => {
    if (e.data && e.data.id === idXanelaTarefa) editor.pedirPechar()
  })
}

// ── Asistente de IA ──────────────────────────────────────────────────────
// Botón á esquerda de "Xerar" + panel lateral (asistente/panel.js). A IA é a
// configurada en Opcións. As ferramentas con que controla o programa son as
// de asistente/ferramentas.js (curso, informe, plantillas, Opcións...) e as de
// Go (asistente_ferramentas.go).
const btnAsistente = document.getElementById('btnAsistente')
const btnAsistenteLogo = document.getElementById('btnAsistenteLogo')
let asistente = null

// aplicarTipografia cambia os axustes do texto sen pasar polo panel flotante
// (mesma persistencia ca gardarAxustesTexto).
function aplicarTipografia(nova) {
  tipografia = {
    tamanoPt: Number(nova.tamanoPt) || 0,
    interlinado: Number(nova.interlinado) || 0,
    fonte: typeof nova.fonte === 'string' ? nova.fonte : '',
  }
  try { localStorage.setItem(AXUSTES_TEXTO_KEY, JSON.stringify(tipografia)) } catch { /* sen persistencia */ }
  pintarAxustesTexto()
}

// recargarOpcions aplica á interface as Opcións que cambiou o asistente.
async function recargarOpcions() {
  const s = await GetSettings()
  xerarAoGardarActivo = s.xerarAoGardar !== false
  aplicarVisibilidadeExportadores(s)
  const idioma = s.idioma || idiomaPiztu || 'gl'
  if (idioma !== idiomaActual) refrescarIdioma(idioma)
  checkLatex()
  asistente?.refrescar()
}

// resumoEstadoAsistente: unhas liñas co que hai agora no programa, para o
// prompt do asistente (o detalle pídeo el coas ferramentas).
function resumoEstadoAsistente() {
  const p = informes?.getPlan()
  if (!p) return ''
  if (informes.ferramenta.esFormulario) return resumoEstadoFormulario(p)
  const c = p.config
  if (informes.ferramenta.id === 'actas') return resumoEstadoActas(p)
  const meses = calendario(c, p.monthsData)
  const plantilla = plantillaSelect?.selectedOptions?.[0]?.textContent?.trim()
  return [
    `- Curso académico: ${p.academicYear || '(sen indicar)'}; datas ${c.startDate} a ${c.endDate} (${meses.length} meses).`,
    `- Centro: ${c.schoolName || '(sen indicar)'}; localidade: ${c.location || '(sen indicar)'}; xefe/a: ${c.headOfDepartment || '(sen indicar)'}.`,
    `- Tarefas: ${totalTarefas(p)}${p.isLocked ? '; o curso está BLOQUEADO (non se pode editar)' : ''}; introdución do curso: ${c.introText ? 'escrita' : 'baleira'}.`,
    `- Gardado en Lúa, no proxecto «${nomeProxectoAberto}»; plantilla activa: ${plantilla || 'ningunha'}; informe xerado: ${informes.temResultado() ? 'si' : 'non'}.`,
    `- Idioma da interface: ${idiomaActual}.`,
  ].join('\n')
}

// resumoEstadoFormulario: o estado para o asistente cando a ferramenta activa é un formulario propio.
function resumoEstadoFormulario(p) {
  // En produción o asistente nin sabe canto hai escrito: só que non ten acceso.
  if (informes.ferramenta.enProduccion) {
    return '- FERRAMENTA ACTIVA: un formulario en modo PRODUCIÓN. Non tes acceso a el nin aos seus datos: non chames a ferramenta ningunha sobre el.'
  }
  const e = informes.ferramenta.esquema
  const cubertos = e.campos.filter((c) => c.tipo !== 'seccion' && p.valores[c.id] !== undefined && p.valores[c.id] !== '').length
  return [
    `- FERRAMENTA ACTIVA: o formulario propio «${e.nome}» (as ferramentas curso_*, tarefa_* e actas_* non traballan sobre el).`,
    `- Campos: ${e.campos.length}; con datos: ${cubertos}. Gardado en Lúa, no proxecto «${nomeProxectoAberto}»; documento xerado: ${informes.temResultado() ? 'si' : 'non'}.`,
    `- Idioma da interface: ${idiomaActual}.`,
  ].join('\n')
}

// resumoEstadoActas é o resumo do estado para o asistente cando a ferramenta activa son as actas.
function resumoEstadoActas(p) {
  const c = p.config
  const plantilla = plantillaSelect?.selectedOptions?.[0]?.textContent?.trim()
  return [
    `- FERRAMENTA ACTIVA: actas do departamento (as ferramentas actas_* traballan sobre ela; as do cronograma curso_* e tarefa_* non).`,
    `- Curso académico: ${p.academicYear || '(sen indicar)'}; datas ${c.startDate} a ${c.endDate}.`,
    `- Centro: ${c.schoolName || '(sen indicar)'}; departamento: ${c.department || '(sen indicar)'}; xefe/a: ${c.headOfDepartment || '(sen indicar)'}; secretario/a: ${c.secretary || '(sen indicar)'}.`,
    `- Membros: ${c.members.length}; materias: ${c.subjects.length}; actas: ${totalActas(p)}; informes de seguimento: ${totalSeguimentos(p)}${p.isLocked ? '; o curso está BLOQUEADO (non se pode editar)' : ''}.`,
    `- Ámbito que se xera e imprime: ${informes.getAmbito()}.`,
    `- Gardado en Lúa, no proxecto «${nomeProxectoAberto}»; plantilla activa: ${plantilla || 'ningunha'}; documento xerado: ${informes.temResultado() ? 'si' : 'non'}.`,
    `- Idioma da interface: ${idiomaActual}.`,
  ].join('\n')
}

// informesAtivo reenvía cada chamada ao controlador da ferramenta que estea activa
// agora (o asistente vive máis ca a ferramenta escollida).
const informesAtivo = new Proxy({}, { get: (_, clave) => informes[clave] })

function crearAsistente() {
  const ferr = crearFerramentas({
    informes: informesAtivo,
    api: { IAChamarFerramenta, AdxuntoDataURI },
    ui: {
      desacoplarResultado,
      acoplarResultado,
      escollerPlantilla: async (id) => { await cambiarPlantilla(id); await refrescarSelectorPlantillas() },
      getTipografia: tipografiaActual,
      setTipografia: aplicarTipografia,
      avisar: (mensaxe, tipo) => { setStatus(mensaxe); if (tipo === 'aviso') showWarnings([mensaxe]) },
      preguntar: (pregunta, opcions) => asistente.preguntar(pregunta, opcions),
      refrescarPlantillas: () => refrescarSelectorPlantillas(),
      recargarOpcions,
      cambiarFerramenta,
      ferramentaActiva: () => ferramentaActiva,
      abrirTrasladar: (lapela) => abrirTrasladarUI({ lapela }),
      reutilizar: reutilizarNoProxectoUI,
    },
  })
  asistente = crearPanelAsistente({
    t,
    api: {
      IAAsistenteEstado, IAFerramentas, IATurnoAsistente: (req) => IATurnoAsistente({ ...req, idioma: idiomaActual, documento: ferramentaActiva }),
      // Adxuntos: ficheiros que se xuntan á conversa (almacén temporal, ver adxuntos.go).
      AdxuntoGardar, AdxuntoEliminar, AdxuntosLimpar, AdxuntoVista, AdxuntoDataURI,
      // Bóveda de Obsidian da ferramenta activa (ver boveda.go).
      BovedaEstadoDe, BovedaProxectoConectar, BovedaConectar, BovedaEscoller, IAFerramentasBoveda,
    },
    documentoBoveda: () => (ferramentaActiva.startsWith('form:') ? 'formulario' : ferramentaActiva),
    executar: ferr.executar,
    estadoPrograma: resumoEstadoAsistente,
    abrirOpcions: openOptions,
    aoCambiar: ({ ocupado, preset }) => {
      btnAsistenteLogo.innerHTML = logoProvedor(preset)
      btnAsistente.classList.toggle('btn-asistente--ocupado', ocupado)
    },
  })
}

// --- Actualización de Lua (← botón "i" da barra) ---
// O backend comproba en fondo ao arrincar (evento actualizacion_lua_cambiada,
// ver Events.On abaixo); aquí só se pinta o que xa se sabe. Mesmo mecanismo
// ca "Sobre Piztu" en piztu/frontend/src/main.js.
let actualizacionLua = null

function pintarActualizacionLua() {
  sobreVersion.textContent =
    t('about.version', 'Versión') + ' ' + (actualizacionLua ? actualizacionLua.versionLocal : '')

  const dispo = !!(actualizacionLua && actualizacionLua.disponible)
  btnAbout.classList.toggle('btn-actualizacion-dispo', dispo)

  if (dispo) {
    sobreActualizacion.textContent = t('about.novaversion', 'Hai unha versión nova de Lúa ({v}) para descargar e instalar.', {
      v: actualizacionLua.versionRemota,
    })
    sobreActualizacion.classList.remove('hidden')
    btnActualizarLua.classList.remove('hidden')
    btnActualizarLua.disabled = false
    btnActualizarLua.textContent = t('about.actualizar', '⟳ Actualizar')
  } else {
    sobreActualizacion.classList.add('hidden')
    btnActualizarLua.classList.add('hidden')
  }
}

function aplicarActualizacionLua(estado) {
  actualizacionLua = estado || null
  pintarActualizacionLua()
}

// Instala en quente a última versión de Lua e reinicia (← fondo laranxa no
// botón "i" cando GetActualizacionLua/evento actualizacion_lua_cambiada
// din que hai unha nova). Lua péchase só e reábrese xa actualizado.
btnActualizarLua.addEventListener('click', async () => {
  if (!await confirmar(t('about.actualizar.confirmar',
      'Isto vai reiniciar Lúa para aplicar a actualización. ¿Continuar?'))) {
    return
  }
  btnActualizarLua.disabled = true
  btnActualizarLua.textContent = t('about.actualizando', 'Actualizando...')
  AplicarActualizacionLua()
    .then(() => {
      btnActualizarLua.textContent = t('about.actualizar.ok', '✔ Reiniciando…')
      setTimeout(() => Application.Quit(), 400)
    })
    .catch((err) => {
      btnActualizarLua.disabled = false
      btnActualizarLua.textContent = t('about.actualizar', '⟳ Actualizar')
      showWarnings([mensaxeErro(err)])
    })
})

Events.On('actualizacion_lua_cambiada', (e) => aplicarActualizacionLua(e.data))

// --- Wiring ---

btnNew.addEventListener('click', conFerramenta(newFile))
btnTrasladar?.addEventListener('click', () => abrirTrasladarUI())
btnUndo.addEventListener('click', conFerramenta(() => informes.desfacer()))
btnRedo.addEventListener('click', conFerramenta(() => informes.refacer()))
btnRun.addEventListener('click', run)
if (plantillaSelect) plantillaSelect.addEventListener('change', () => cambiarPlantilla(plantillaSelect.value))
if (btnPlantillas) btnPlantillas.addEventListener('click', abrirPlantillas)
btnPrint.addEventListener('click', conFerramenta(printPreview))
btnPDF.addEventListener('click', conFerramenta(exportPDF))
btnExportTex.addEventListener('click', conFerramenta(exportTex))
btnExportMarkdown.addEventListener('click', conFerramenta(exportMarkdown))
btnExportDocx.addEventListener('click', conFerramenta(exportDocx))
btnExportOdt.addEventListener('click', conFerramenta(exportOdt))
btnBoveda?.addEventListener('click', conFerramenta(() => informes.engadirABoveda(nomeDeFerramenta(ferramentaActiva), btnBoveda)))
chkBovedaSempre?.addEventListener('change', async () => {
  const quere = chkBovedaSempre.checked
  try {
    await FixarBovedaAutomatica(ferramentaActiva, quere)
    setStatus(quere ? t('preview.bovedaSempre.si', 'Este formulario engadirase á bóveda cada vez que se xere.') : t('preview.bovedaSempre.non', 'Este formulario xa non se engade á bóveda ao xerar.'))
  } catch (err) {
    chkBovedaSempre.checked = !quere
    showWarnings([mensaxeErro(err)])
  }
})
if (souXanelaResultado) { if (btnBoveda) btnBoveda.hidden = true; if (bovedaSempreFila) bovedaSempreFila.hidden = true }
btnOptions.addEventListener('click', openOptions)
btnOptionsCancel.addEventListener('click', closeOptions)
btnOptionsSave.addEventListener('click', saveOptions)

btnInstallLatex.addEventListener('click', installLatexNow)
btnLatexManual.addEventListener('click', () => latexManualOverlay.classList.remove('hidden'))
btnLatexManualClose.addEventListener('click', () => latexManualOverlay.classList.add('hidden'))
btnLatexDismiss.addEventListener('click', () => latexWarning.classList.add('hidden'))

btnAbout.addEventListener('click', () => sobreOverlay.classList.remove('hidden'))
btnAboutClose.addEventListener('click', () => sobreOverlay.classList.add('hidden'))

// Ctrl+N/S/O/Enter/X funcionan sexa cal sexa o foco. Ctrl+O (meter) e
// Ctrl+Maiús+S (sacar) abren Trasladar; Ctrl+S garda xa en Lúa. Ctrl+Z/Y sobre o
// historial do curso só se non se está a teclear nun campo (aí manda o
// desfacer nativo do campo). Ctrl+X (Xerar) fai preventDefault() antes de que
// o navegador execute o "cortar" nativo - por deseño: pediuse Ctrl+X para
// Xerar a propósito, sabendo que se perde o atallo para cortar texto (segue
// dispoñible dende o menú contextual).
window.addEventListener('keydown', (e) => {
  // As xanelas de resultado e de tarefa non teñen editor/ficheiro propio -
  // Ctrl+N/S/O/etc non teñen sentido alí.
  if (souXanelaResultado || souXanelaTarefa) return
  const mod = e.ctrlKey || e.metaKey
  if (mod && e.shiftKey && (e.key === 's' || e.key === 'S')) { e.preventDefault(); abrirTrasladarUI({ lapela: 'sacar' }); return }
  if (mod && e.key === 'o') { e.preventDefault(); abrirTrasladarUI({ lapela: 'meter' }); return }
  if (mod && !haiFerramenta()) return // sen documento: Ctrl+N/S/Z/Xerar non fan nada
  if (mod && e.key === 's') { e.preventDefault(); gardarAgora() }
  else if (mod && e.key === 'n') { e.preventDefault(); newFile() }
  else if (mod && (e.key === 'Enter' || e.key === 'x' || e.key === 'X')) { e.preventDefault(); run() }
  else if (mod && (e.key === 'z' || e.key === 'Z' || e.key === 'y') && !/^(INPUT|TEXTAREA|SELECT)$/.test(e.target.tagName)) {
    e.preventDefault()
    if (e.key === 'y' || e.shiftKey) informes.refacer()
    else informes.desfacer()
  }
  else if (e.key === 'Escape') {
    closeOptions()
    latexManualOverlay.classList.add('hidden')
    sobreOverlay.classList.add('hidden')
  }
})

setStatus(t('status.gardadoLua', 'Gardado en Lúa'))
pintarActualizacionLua() // versión local xa dispoñible dende o arranque, aínda sen resposta de GitHub
GetActualizacionLua().then(aplicarActualizacionLua).catch(() => {})

// Idioma gardado: aplícase despois do arranque (en galego por defecto) en
// vez de bloquear a creación inicial do editor nunha chamada async - un
// posible "flash" breve en galego é preferible a reestruturar todo o
// arranque do módulo arredor dun await.
//
// Prioridade (ver saveOptions): 1) Settings.Idioma, se xa se escolleu un
// idioma propio en Opcións - prevalece sempre; 2) IdiomaPiztu(), se Lua
// arrincou dende Piztu e aínda non hai elección propia; 3) galego (xa é o
// que está cargado por defecto, non fai falla refrescar).
Promise.all([GetSettings(), IdiomaPiztu().catch(() => '')]).then(([s, ip]) => {
  idiomaPiztu = ip || ''
  xerarAoGardarActivo = s.xerarAoGardar !== false
  const idiomaEfectivo = s.idioma || idiomaPiztu
  if (idiomaEfectivo && idiomaEfectivo !== 'gl') refrescarIdioma(idiomaEfectivo)
  aplicarVisibilidadeExportadores(s)
  // O selector de plantilla non existe na xanela de resultado (agocha a
  // cabeceira), e alí non hai nada que escoller: xérase sempre dende a
  // xanela editor.
  if (!souXanelaResultado) refrescarSelectorPlantillas()
}).catch(() => {})

// arrincarModoXanelas decide a UI segundo a xanela: Lua arrinca sempre
// acoplado (xanela única) - non hai axuste que cambie iso, desacoplar é unha
// decisión manual de sesión (botón "Desacoplar"). A xanela de resultado non
// compila nada ela soa, así que non comproba LaTeX nin crea o editor.
function arrincarModoXanelas() {
  if (souXanelaResultado) {
    entrarModoResultado()
    return
  }
  if (souXanelaTarefa) {
    entrarModoTarefa()
    return
  }
  if (souXanelaPlantilla) {
    entrarModoPlantilla()
    return
  }
  if (souXanelaFormulario) {
    entrarModoFormulario()
    return
  }
  crearInformes()
  checkLatex()
  Events.On('lua:resultado-pechado', saírModoDobreEditor)
  // O editor de plantillas (na súa xanela propia) gardou: refrescar selector e informe.
  Events.On('lua:plantilla-aplicada', (e) => {
    plantillaGardada(e.data && e.data.id).catch((err) => setStatus(mensaxeErro(err)))
  })
  // O deseñador de formularios (na súa xanela propia) gardou ou eliminou un formulario.
  Events.On('lua:formulario-gardado', (e) => {
    try { aplicarResultadoDisenador(JSON.parse(e.data.resultado)) } catch (err) { setStatus(mensaxeErro(err)) }
  })
  Events.On('lua:formulario-eliminado', (e) => {
    try { aplicarResultadoDisenador(JSON.parse(e.data.resultado)) } catch (err) { setStatus(mensaxeErro(err)) }
  })
  // Unha tarefa aplicada na súa xanela propia entra no curso desta xanela.
  Events.On('lua:tarefa-aplicada', (e) => {
    try { informesCrono.aplicarTarefaExterna(JSON.parse(e.data.resultado)) } catch (err) { setStatus(mensaxeErro(err)) }
  })

  // btnDesacoplarResultado (só existe na xanela editor) - dispoñible en
  // calquera momento. Se AbrirXanelaResultado falla, vólvese ao estado
  // anterior en vez de deixar o panel agochado sen xanela.
  btnDesacoplarResultado.classList.remove('hidden')
  btnDesacoplarResultado.addEventListener('click', desacoplarResultado)
  btnAsistente.addEventListener('click', () => asistente?.alternar())
  crearAsistente()
}

// desacoplarResultado leva o resultado a unha xanela á parte (botón
// "Desacoplar" e ferramenta resultado_xanela do asistente).
async function desacoplarResultado() {
  activarModoDobreEditor()
  try {
    await AbrirXanelaResultado()
  } catch (err) {
    saírModoDobreEditor()
    setStatus(mensaxeErro(err))
  }
}

async function acoplarResultado() {
  await DockResultado() // WindowClosing -> lua:resultado-pechado -> saírModoDobreEditor
}
arrincarModoXanelas()
