// createInformes - o controlador do modo "Informes" de Lua. Fai de pegamento
// entre o editor do cronograma (cronograma-editor.js), o backend Go
// (informe_cronograma.go) e a chrome de main.js (barra, estado, avisos,
// panel de resultado). Concentra aquí todo o que main.js non debe saber:
// o curso en edición, o ficheiro aberto, desfacer/refacer, o borrador
// automático e as exportacións.
//
// Igual ca editor.js/explorer.js é unha factoría illada: recibe `api`
// (os bindings de Wails que precisa) e `ui` (callbacks de main.js), e nunca
// importa nada de main.js - así se pode probar en controlador.test.js.
//
// Serve para calquera ferramenta (cronograma, actas...): cada unha achega unha
// "ferramenta" (o seu modelo de datos, o seu editor e a súa petición ao
// backend, ver ferramenta-cronograma.js e ferramenta-actas.js) e o controlador
// leva o curso en edición, os ficheiros, desfacer/refacer, o borrador e as
// exportacións igual para todas.
import { FERRAMENTA_CRONOGRAMA } from './ferramenta-cronograma.js'

// Límites do historial de desfacer: os cursos levan imaxes en base64 e cada
// instantánea é o JSON enteiro, así que se limita tamén polo tamaño total.
const HISTORIAL_MAX_ENTRADAS = 60
const HISTORIAL_MAX_CARACTERES = 40_000_000
const AGRUPAR_TECLEO_MS = 1200
const BORRADOR_MS = 700

export function createInformes({
  container, api, ui, t: tIn, tipografia = () => null, lerImaxe, abrirXanelaTarefa, asistenteTarefa,
  ferramenta = FERRAMENTA_CRONOGRAMA,
}) {
  const { novoPlan, lerFicheiro, escribirFicheiro, nomeFicheiroSuxerido } = ferramenta
  const t = tIn || ((clave, fallback, vars) => {
    let val = fallback ?? clave
    if (vars) for (const k in vars) val = val.replaceAll(`{${k}}`, vars[k])
    return val
  })

  let editor = null
  let plan = novoPlan()
  // ambito: que se xera/imprime (só ten sentido nas ferramentas con varios
  // documentos, coma as actas: "curso" ou o mes dunha acta).
  let ambito = ferramenta.ambitoPorDefecto || ''
  let ruta = null // ficheiro aberto/gardado, ou null se é un curso novo
  let sucio = false // cambios desde o último gardar/abrir
  let versionPlan = 0 // sobe en cada cambio; permite saber se o resultado está ao día
  let versionResultado = -1
  let ultimoPDF = null
  let ocupado = false
  // Resultado do último Xerar (para o asistente: avisos, páxinas, erro).
  let ultimoXerar = { ok: false, avisos: [], paxinas: 0, erro: '' }
  let ultimoErroExportar = ''

  // ---------- Historial (desfacer/refacer) ----------

  let pila = []
  let idx = -1
  let ultimoMotivo = ''
  let ultimoTempo = 0

  function reiniciarHistorial() {
    pila = [JSON.stringify(plan)]
    idx = 0
    ultimoMotivo = ''
  }

  function rexistrar(motivo) {
    const s = JSON.stringify(plan)
    if (s === pila[idx]) return
    const agora = Date.now()
    // O tecleo seguido nun campo é UN paso de desfacer, non un por letra.
    if (motivo === 'texto' && ultimoMotivo === 'texto' && agora - ultimoTempo < AGRUPAR_TECLEO_MS) {
      pila[idx] = s
    } else {
      pila.length = idx + 1
      pila.push(s)
      idx = pila.length - 1
    }
    ultimoMotivo = motivo
    ultimoTempo = agora
    let total = pila.reduce((n, x) => n + x.length, 0)
    while (pila.length > 1 && (pila.length > HISTORIAL_MAX_ENTRADAS || total > HISTORIAL_MAX_CARACTERES)) {
      total -= pila.shift().length
      idx--
    }
  }

  function irAoHistorial(novoIdx) {
    if (novoIdx < 0 || novoIdx >= pila.length) return
    idx = novoIdx
    ultimoMotivo = ''
    plan = JSON.parse(pila[idx])
    editor.setValue(plan)
    marcarCambio()
  }

  // ---------- Cambios e borrador ----------

  let temporizadorBorrador = null
  // descartado: a ferramenta borrouse (un formulario eliminado): o borrador NON
  // pode volver a escribirse (podería levar datos reais dun formulario que xa non existe).
  let descartado = false

  const haiContido = () => ferramenta.haiContido(plan)

  function marcarCambio() {
    sucio = true
    versionPlan++
    clearTimeout(temporizadorBorrador)
    temporizadorBorrador = setTimeout(gardarBorrador, BORRADOR_MS)
    ui.aoCambiar?.()
  }

  // O borrador é o traballo en curso: se Lua se pecha sen gardar, ao volver
  // abrir o curso está como estaba (mesma idea ca data.json da app orixinal).
  async function gardarBorrador() {
    clearTimeout(temporizadorBorrador)
    temporizadorBorrador = null
    if (descartado) return
    try {
      await api.GardarBorradorInforme(escribirFicheiro(plan, { path: ruta || '' }))
    } catch {
      // Rede de seguridade, non crítica: se falla non se molesta ao usuario.
    }
  }

  function aoCambiarEditor(novo, motivo) {
    plan = novo
    rexistrar(motivo)
    marcarCambio()
  }

  function cargar(novoPlanCargado, novaRuta, { sucioInicial = false } = {}) {
    plan = novoPlanCargado
    ruta = novaRuta
    editor.setValue(plan)
    reiniciarHistorial()
    sucio = sucioInicial
    versionPlan++
    ultimoPDF = null
    versionResultado = -1
    ui.limparResultado?.()
    ui.setUltimoPDF?.(null)
    ui.aoCambiar?.()
  }

  // ---------- Arranque ----------

  // cargarBorrador carga o borrador da ferramenta (ou un plan novo se non hai).
  async function cargarBorrador() {
    let recuperado = false
    try {
      const borrador = await api.LerBorradorInforme()
      if (borrador) {
        const dato = JSON.parse(borrador)
        const [p] = lerFicheiro(borrador)
        cargar(p, dato.path || null, { sucioInicial: true })
        recuperado = true
      }
    } catch {
      // Un borrador ilegible non debe impedir arrincar.
    }
    if (!recuperado) cargar(novoPlan(), null)
    ui.setStatus(ruta || t('status.unsaved', 'sen gardar'))
  }

  async function iniciar() {
    editor = ferramenta.crearEditor(container, { t, onChange: aoCambiarEditor, lerImaxe, abrirXanelaTarefa, asistenteTarefa })
    await cargarBorrador()
  }

  // recargar volve ler o borrador sen refacer o editor: serve cando os datos
  // cambian de sitio (entrar ou saír dun proxecto). Quen chama volca ANTES o
  // borrador actual (volcarBorrador), porque os datos novos veñen doutro sitio.
  async function recargar() {
    clearTimeout(temporizadorBorrador)
    temporizadorBorrador = null
    await cargarBorrador()
  }

  // ---------- Ficheiros ----------

  async function confirmarPerdaDeCambios() {
    if (!sucio || !haiContido()) return true
    return ui.confirmar(t('confirm.discardChanges', 'Vanse perder os cambios non gardados. Continuar?'))
  }

  async function novo() {
    if (!await confirmarPerdaDeCambios()) return
    cargar(novoPlan(), null)
    gardarBorrador()
    ui.showWarnings([])
    ui.setStatus(t('status.unsaved', 'sen gardar'))
  }

  // escollerCurso: un ficheiro pode traer varios cursos (copia de seguridade
  // completa de Piztu Orienta). Devolve o escollido ou null se se cancela.
  function escollerCurso(cursos) {
    if (cursos.length === 1) return Promise.resolve(cursos[0])
    return new Promise((resolve) => {
      const overlay = document.createElement('div')
      overlay.className = 'overlay overlay--dialog'
      const modal = document.createElement('div')
      modal.className = 'modal modal--dialog'
      const msg = document.createElement('p')
      msg.className = 'dialog-message'
      msg.textContent = t('informes.escoller.curso', 'Este ficheiro contén {n} cursos. Cal queres abrir?', { n: cursos.length })
      modal.append(msg)
      const lista = document.createElement('div')
      lista.style.cssText = 'display:flex;flex-direction:column;gap:0.4rem;max-height:50vh;overflow:auto;margin:0.6rem 0'
      const pechar = (r) => { overlay.remove(); resolve(r) }
      for (const c of cursos) {
        const b = document.createElement('button')
        b.type = 'button'
        b.className = 'crono-btn'
        b.textContent = ferramenta.descricionCurso(c, t)
        b.addEventListener('click', () => pechar(c))
        lista.append(b)
      }
      modal.append(lista)
      const acc = document.createElement('div')
      acc.className = 'modal-actions'
      const cancelar = document.createElement('button')
      cancelar.type = 'button'
      cancelar.textContent = t('dialog.cancel', 'Cancelar')
      cancelar.addEventListener('click', () => pechar(null))
      acc.append(cancelar)
      modal.append(acc)
      overlay.append(modal)
      overlay.addEventListener('click', (e) => { if (e.target === overlay) pechar(null) })
      document.body.append(overlay)
    })
  }

  async function abrir() {
    if (!await confirmarPerdaDeCambios()) return
    try {
      const res = await api.AbrirInformeDialog()
      if (!res) return
      const cursos = lerFicheiro(res.content)
      const curso = await escollerCurso(cursos)
      if (!curso) return
      // Un ficheiro con varios cursos non se pode sobrescribir con Ctrl+S
      // sen perder os outros: gárdase como copia nova (ruta null).
      const vario = cursos.length > 1
      cargar(curso, vario ? null : res.path)
      // Cada curso lembra a súa plantilla (estilo): recupérase antes de xerar.
      if (!vario) await ui.aoAbrirFicheiro?.(res.path)
      ui.showWarnings([])
      ui.setStatus(vario ? t('status.unsaved', 'sen gardar') : res.path)
      gardarBorrador()
      await xerar()
    } catch (err) {
      ui.setStatus(t('status.errorOpening', 'Erro ao abrir'))
      ui.showWarnings([ui.mensaxeErro(err)])
    }
  }

  async function gardarComo() {
    try {
      const suxerido = (ruta ? ruta.replace(/\.json$/i, '') : nomeFicheiroSuxerido(plan)) + '.json'
      const path = await api.GardarInformeDialog(suxerido, escribirFicheiro(plan))
      if (!path) return false
      ruta = path
      sucio = false
      ui.setStatus(t('status.saved', 'Gardado: ') + path)
      gardarBorrador()
      // "Gardar como" estrea ruta: anótase que ESE curso usa a plantilla activa.
      await ui.aoGardarComo?.(path)
      return true
    } catch (err) {
      ui.setStatus(t('status.errorSaving', 'Erro ao gardar'))
      ui.showWarnings([ui.mensaxeErro(err)])
      return false
    }
  }

  async function gardar() {
    if (!ruta) return gardarComo()
    try {
      await api.SaveFile(ruta, escribirFicheiro(plan))
      sucio = false
      ui.setStatus(t('status.saved', 'Gardado: ') + ruta)
      gardarBorrador()
      return true
    } catch (err) {
      ui.setStatus(t('status.errorSaving', 'Erro ao gardar'))
      ui.showWarnings([ui.mensaxeErro(err)])
      return false
    }
  }

  // ---------- Xerar e exportar ----------

  function peticion() {
    // nomeBase (ruta do curso sen extensión, ou o nome suxerido) enche
    // {{FICHEIRO}} das plantillas e é o nome que propón a xanela de
    // resultado desacoplada ao exportar.
    return ferramenta.peticion(plan, { tipografia: tipografia(), nomeBase: nomeBase(''), ambito })
  }

  function nomeBase(ext) {
    const base = ruta ? ruta.replace(/\.json$/i, '') : nomeFicheiroSuxerido(plan)
    // Unha soa acta: o nome leva o mes (actas-2025-2026-2025-11).
    const sufixo = ambito && ambito !== 'curso' ? '-' + ambito : ''
    return base + sufixo + ext
  }

  async function xerar() {
    if (ocupado) return false
    const erroDatas = ferramenta.validarDatas(plan)
    if (erroDatas) {
      ui.setStatus(t('status.error', 'Erro'))
      ui.showWarnings([erroDatas])
      ultimoXerar = { ok: false, avisos: [], paxinas: 0, erro: erroDatas }
      return false
    }
    ocupado = true
    ui.ocupado?.(true)
    const versionAoInicio = versionPlan
    ui.setStatus(t('informes.status.compilando', 'Compilando o documento…'))
    ui.showWarnings([])
    ui.log('[XERAR] Compilando o documento (LaTeX)...')
    try {
      const r = await api.XerarInformeCronograma(peticion())
      ultimoPDF = r.pdfBase64
      versionResultado = versionAoInicio
      ui.setUltimoPDF?.(r.pdfBase64)
      ui.previsualizar(r.pageImages)
      ui.showWarnings(r.warnings)
      ui.logLatex?.(r.log)
      ui.setStatus(t('status.done', 'Feito.'))
      ui.log('[XERAR] Feito.')
      ultimoXerar = { ok: true, avisos: r.warnings || [], paxinas: (r.pageImages || []).length, erro: '' }
      return true
    } catch (err) {
      ui.setStatus(t('status.error', 'Erro'))
      ui.showWarnings([ui.mensaxeErro(err)])
      ui.log('[XERAR] ERRO: ' + err)
      ultimoXerar = { ok: false, avisos: [], paxinas: 0, erro: ui.mensaxeErro(err) }
      ui.aoFallar?.()
      return false
    } finally {
      ocupado = false
      ui.ocupado?.(false, ultimoXerar.ok)
    }
  }

  // exportar: envolve o patrón repetido (estado "compilando", aviso, erro).
  async function exportar(boton, estado, erroClave, erroTexto, fn) {
    ultimoErroExportar = ''
    ui.ocupadoBoton?.(boton, true)
    ui.setStatus(estado)
    ui.showWarnings([])
    ui.log('[XERAR] ' + estado)
    try {
      const r = await fn()
      ui.showWarnings(r?.warnings || [])
      const path = r?.path
      ui.setStatusGardado(path ? t('status.exported', 'Exportado: ') + path : t('status.unsaved', 'sen gardar'), path)
      ui.log('[XERAR] ' + (path ? 'Exportado: ' + path : 'Feito.'))
      return path || null
    } catch (err) {
      ui.setStatus(t(erroClave, erroTexto))
      ui.showWarnings([ui.mensaxeErro(err)])
      ui.log('[XERAR] ERRO: ' + err)
      ultimoErroExportar = ui.mensaxeErro(err)
      return null
    } finally {
      ui.ocupadoBoton?.(boton, false)
    }
  }

  async function exportarPDF(boton) {
    ultimoErroExportar = ''
    // Reutiliza o PDF xa xerado se o curso non cambiou desde entón.
    if (ultimoPDF === null || versionResultado !== versionPlan) {
      if (!await xerar()) { ultimoErroExportar = ultimoXerar.erro; return null }
    }
    ui.ocupadoBoton?.(boton, true)
    try {
      const path = await api.SavePDFDialog(nomeBase('.pdf'), ultimoPDF)
      ui.setStatusGardado(path ? t('status.pdfSaved', 'PDF gardado: ') + path : t('status.done', 'Feito.'), path)
      ui.log('[XERAR] ' + (path ? 'PDF gardado: ' + path : 'Feito.'))
      return path || null
    } catch (err) {
      ui.setStatus(t('status.errorCompilingPdf', 'Erro ao compilar PDF'))
      ui.showWarnings([ui.mensaxeErro(err)])
      ultimoErroExportar = ui.mensaxeErro(err)
      return null
    } finally {
      ui.ocupadoBoton?.(boton, false)
    }
  }

  const exportarTex = (b) => exportar(b, t('status.exporting', 'Exportando...'), 'status.errorExporting', 'Erro ao exportar',
    () => api.ExportarInformeTex(peticion(), nomeBase('.tex')))
  const exportarMarkdown = (b) => exportar(b, t('status.compilingMarkdown', 'Compilando Markdown...'), 'status.errorExportingMarkdown', 'Erro ao exportar Markdown',
    () => api.ExportarInformeMarkdown(peticion(), nomeBase('.md')))
  const exportarDocx = (b) => exportar(b, t('status.compilingDocx', 'Compilando Word (.docx)...'), 'status.errorExportingDocx', 'Erro ao exportar Word',
    () => api.ExportarInformeDocx(peticion(), nomeBase('.docx')))
  const exportarOdt = (b) => exportar(b, t('status.compilingOdt', 'Compilando OpenDocument (.odt)...'), 'status.errorExportingOdt', 'Erro ao exportar OpenDocument',
    () => api.ExportarInformeOdt(peticion(), nomeBase('.odt')))

  return {
    iniciar,
    recargar,
    // A petición que se mandaría ao xerar o documento actual (vista previa da plantilla).
    peticionActual: peticion,
    novo,
    abrir,
    gardar,
    gardarComo,
    xerar,
    exportarPDF,
    exportarTex,
    exportarMarkdown,
    exportarDocx,
    exportarOdt,
    desfacer: () => irAoHistorial(idx - 1),
    refacer: () => irAoHistorial(idx + 1),
    focus: () => editor?.focus(),
    // ── Para o asistente de IA (asistente/ferramentas.js) ──
    // O curso actual (obxecto vivo: non se debe mutar, só ler).
    getPlan: () => plan,
    // Substitúe o curso enteiro por un novo (construído coas funcións de
    // cronograma-modelo.js): entra no historial de desfacer coma unha
    // edición máis e marca o curso coma cambiado.
    substituirPlan(novo, motivo = 'estrutura') {
      plan = novo
      editor.setValue(plan)
      rexistrar(motivo)
      marcarCambio()
    },
    // Abre a pantalla de edición dunha tarefa (existente ou nova).
    abrirTarefa: (clave, tarefa) => editor?.abrirTarefa(clave, tarefa),
    ultimoXerar: () => ultimoXerar,
    ultimoErroExportar: () => ultimoErroExportar,
    refrescarIdioma: () => editor?.refrescar(),
    // Tarefa gardada nunha xanela propia de edición: entra no curso.
    aplicarTarefaExterna: (r) => editor?.aplicarTarefaExterna(r),
    ruta: () => ruta,
    ferramenta,
    // Que se xera e imprime (ver `ambito` arriba). Cambialo invalida o
    // resultado xa xerado, pero non marca o curso coma cambiado.
    getAmbito: () => ambito,
    setAmbito(novo) {
      if (novo === ambito) return
      ambito = novo
      versionPlan++
    },
    // Para main.js: hai un PDF ao día que se poida imprimir?
    temResultado: () => ultimoPDF !== null,
    // O resultado xerado corresponde ao curso e ao ámbito de agora?
    estaAoDia: () => ultimoPDF !== null && versionResultado === versionPlan,
    // Gardar o borrador sen agardar polo debounce (ao pechar ou trocar de modo).
    volcarBorrador: gardarBorrador,
    // Para main.js ao borrar un formulario: para o gardado automático do borrador para sempre.
    descartar(activo = true) { descartado = activo; if (activo) { clearTimeout(temporizadorBorrador); temporizadorBorrador = null } },
    // Só para probas.
    _estado: () => ({ plan, ruta, sucio, idx, pila: pila.length }),
  }
}
