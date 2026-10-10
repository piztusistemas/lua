// O panel do asistente de IA: unha columna á dereita coas TAREFAS que sabe
// facer (fichas), unha caixa para pedirlle o que se queira e a conversa, coas
// ferramentas que vai usando. O bucle está en bucle.js e as ferramentas en
// ferramentas.js; aquí só se pinta e se pregunta ao usuario.
import { el } from '../informes/dom.js'
import { markdownAHtml } from '../informes/cronograma-md.js'
import { executarConversa, retomar } from './bucle.js'
import { logoProvedor, nomeProvedor } from './logos.js'
import { TAREFAS_BASE, lerPropias, engadirPropia, eliminarPropia, instrucionFinal } from './tarefas.js'

const CLAVE_CONFIRMAR = 'lua.asistente.confirmarTodo'

// Nomes amosables das ferramentas (a clave i18n é asistente.ferr.<nome>).
export const ETIQUETAS_FERRAMENTAS = {
  curso_ler: 'Ler o curso', curso_configurar: 'Cambiar datos do curso', tarefa_engadir: 'Engadir tarefa',
  tarefa_editar: 'Editar tarefa', tarefa_eliminar: 'Eliminar tarefa', tarefa_mover: 'Mover tarefa',
  tarefa_abrir: 'Abrir tarefa', curso_novo: 'Novo curso', curso_abrir: 'Abrir curso', curso_gardar: 'Gardar curso',
  informe_xerar: 'Xerar o informe', informe_exportar: 'Exportar o informe', historial: 'Desfacer / refacer',
  resultado_xanela: 'Xanela do resultado', plantilla_escoller: 'Escoller plantilla', axustes_texto: 'Axustes do texto',
  usuario_avisar: 'Avisar', usuario_preguntar: 'Preguntar', plantillas_listar: 'Listar plantillas',
  plantilla_ler: 'Ler plantilla', plantilla_gardar: 'Gardar plantilla', plantilla_comprobar: 'Comprobar plantilla',
  plantilla_duplicar: 'Duplicar plantilla', plantilla_eliminar: 'Eliminar plantilla', opcions_ler: 'Ler as Opcións',
  opcions_cambiar: 'Cambiar Opcións', sistema_estado: 'Estado do sistema', web_buscar: 'Buscar na web',
  web_consultar: 'Ler unha páxina web',
  plantilla_editor_ler: 'Ler a plantilla do editor', plantilla_editor_cambiar: 'Cambiar a plantilla do editor',
  plantilla_previsualizar: 'Previsualizar a plantilla', adxuntos_listar: 'Listar adxuntos', adxunto_ler: 'Ler un adxunto',
  adxunto_ver: 'Ver un adxunto', adxunto_extraer_imaxes: 'Sacar as imaxes dun adxunto', adxunto_eliminar: 'Eliminar un adxunto',
  adxunto_gardar_en_plantilla: 'Gardar imaxes na plantilla', plantilla_imaxe_eliminar: 'Eliminar unha imaxe da plantilla',
  boveda_info: 'Resumo da bóveda', boveda_listar: 'Listar a bóveda', boveda_ler: 'Ler da bóveda', boveda_buscar: 'Buscar na bóveda',
  boveda_enlaces: 'Enlaces da bóveda', boveda_pendentes: 'Fontes sen procesar', boveda_lint: 'Revisar a wiki', boveda_escribir: 'Escribir na wiki',
}

const CLAVES_DETALLE = ['ruta', 'carpeta', 'titulo', 'nome', 'pregunta', 'consulta', 'url', 'formato', 'accion', 'mes', 'id']

function corta(s, n) {
  s = String(s ?? '')
  return s.length > n ? s.slice(0, n - 1) + '…' : s
}

const ICONAS = {
  pechar: '<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M5 5l10 10M15 5L5 15"/></svg>',
  nova: '<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M4 10a6 6 0 1 1 1.8 4.3"/><path d="M4 15.5V11h4.5"/></svg>',
  enviar: '<svg viewBox="0 0 20 20" fill="currentColor"><path d="M3.2 3.6a.8.8 0 0 1 1.1-.9l13 6.4a.8.8 0 0 1 0 1.4l-13 6.4a.8.8 0 0 1-1.1-.9l1.7-5.3a.8.8 0 0 1 .7-.5h5.4a.4.4 0 0 0 0-.8H5.6a.8.8 0 0 1-.7-.6z"/></svg>',
  parar: '<svg viewBox="0 0 20 20" fill="currentColor"><rect x="5" y="5" width="10" height="10" rx="1.6"/></svg>',
  mais: '<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"><path d="M10 4.5v11M4.5 10h11"/></svg>',
  clip: '<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><path d="M16 9.2 9.6 15.6a3.6 3.6 0 0 1-5.1-5.1l6.9-6.9a2.4 2.4 0 0 1 3.4 3.4L8 13.9a1.2 1.2 0 0 1-1.7-1.7l6.2-6.2"/></svg>',
}

// crearPanelAsistente monta o panel e devolve a súa API.
//   t                tradución
//   api              bindings: IAAsistenteEstado, IAFerramentas, IATurnoAsistente
//   executar         (nome, args) -> {contido, erro} (ferramentas.js)
//   estadoPrograma   () -> string co resumo do programa (vai no prompt)
//   abrirOpcions     () abre o diálogo de Opcións (para configurar a IA)
//   aoCambiar        ({ocupado, configurada, preset}) para a barra (logo, indicador)
//   tarefas          fichas de fábrica (por defecto as do asistente principal)
//   claveTarefas     chave de localStorage das fichas propias deste ámbito
//   montarEn         onde se pon o panel: document.body (panel lateral flotante) ou
//                    un elemento (incrustado, coma na pantalla de edición dunha tarefa)
//   textos           {titulo, benvida, benvidaNota, caixaPh} para cambiar os textos
//   abrirOpcions é opcional: sen el non se amosa o botón «Abrir Opcións»
//   documentoBoveda  "cronograma" | "actas" | "plantilla" | "formulario" (ou unha función que o devolve): activa a
//                    barra para conectar unha bóveda de Obsidian (api.BovedaEstadoDe, BovedaConectar,
//                    BovedaEscoller, IAFerramentasBoveda; ver boveda.go)
//   api.AdxuntoGardar (+ AdxuntoEliminar, AdxuntosLimpar, AdxuntoVista, AdxuntoDataURI)
//                    activan os ADXUNTOS: clip, arrastrar e pegar ficheiros na conversa
export function crearPanelAsistente({
  t, api, executar, estadoPrograma, abrirOpcions, aoCambiar = () => {}, almacen = globalThis.localStorage,
  tarefas = TAREFAS_BASE, claveTarefas, montarEn = document.body, textos = {}, documentoBoveda = null,
}) {
  const incrustado = montarEn !== document.body
  const txt = (k, def) => textos[k] ?? def
  const raiz = el('aside', {
    id: incrustado ? undefined : 'asistentePanel', class: 'asistente hidden' + (incrustado ? ' asistente--incrustado' : ''),
    'aria-label': t('asistente.titulo', 'Asistente de IA'),
  })
  let historial = []
  let ferramentas = []
  let estadoIA = { configurada: false, provedor: '', preset: '', modelo: '' }
  let ocupado = false
  let sinal = { abortado: false }
  let pendentes = 0 // tarxetas que agardan resposta do usuario
  const adxApi = api.AdxuntoGardar ? api : null
  let adxPend = [] // adxuntos preparados para a seguinte mensaxe: {id, nome, tipo, tamano, vista}
  let confirmarTodo = false
  try { confirmarTodo = almacen.getItem(CLAVE_CONFIRMAR) === '1' } catch { /* sen persistencia */ }

  // ---------- Elementos fixos ----------

  const logo = el('span', { class: 'asistente__logo' })
  const titulo = el('strong', { class: 'asistente__titulo' })
  const subtitulo = el('small', { class: 'asistente__subtitulo' })
  const btnNova = el('button', { type: 'button', class: 'asistente__ico', html: ICONAS.nova, onclick: () => novaConversa() })
  const btnPechar = el('button', { type: 'button', class: 'asistente__ico asistente__ico--x', html: ICONAS.pechar, onclick: () => pechar() })
  const aviso = el('div', { class: 'asistente__aviso', hidden: true })
  const secBoveda = el('section', { class: 'asistente__boveda', hidden: true })
  const zonaTarefas = el('div', { class: 'asistente__chips' })
  const secTarefas = el('section', { class: 'asistente__tarefas' })
  const conversa = el('div', { class: 'asistente__conversa', 'aria-live': 'polite' })
  const caixa = el('textarea', { class: 'asistente__caixa', rows: '3' })
  const chkConfirmar = el('input', { type: 'checkbox', checked: confirmarTodo || undefined })
  const btnEnviar = el('button', { type: 'button', class: 'asistente__enviar', onclick: () => enviarLibre() })
  const btnParar = el('button', { type: 'button', class: 'asistente__parar', hidden: true, onclick: () => parar() })
  const zonaAdx = el('div', { class: 'asistente__adxuntos', hidden: true })
  const erroAdx = el('p', { class: 'asistente__erro-adx', hidden: true })
  const entradaFicheiros = el('input', { type: 'file', multiple: true, hidden: true })
  const menuClip = el('div', { class: 'asistente__menu-clip', hidden: true, role: 'menu' })
  const btnClip = adxApi ? el('button', { type: 'button', class: 'asistente__clip', html: ICONAS.clip, onclick: () => { menuClip.hidden = !menuClip.hidden } }) : null

  chkConfirmar.addEventListener('change', () => {
    confirmarTodo = chkConfirmar.checked
    try { almacen.setItem(CLAVE_CONFIRMAR, confirmarTodo ? '1' : '0') } catch { /* nada */ }
  })
  caixa.addEventListener('keydown', (e) => {
    if (e.key === 'Enter' && !e.shiftKey && !e.isComposing) { e.preventDefault(); enviarLibre() }
  })
  // Os atallos globais (Ctrl+X = Xerar, Ctrl+S...) non actúan mentres se
  // escribe aquí: Ctrl+X/C/V/Z sobre o texto da caixa son os de sempre.
  raiz.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') { e.stopPropagation(); pechar(); return }
    if ((e.ctrlKey || e.metaKey) && /^(INPUT|TEXTAREA)$/.test(e.target.tagName)) e.stopPropagation()
  })

  // Adxuntos: escoller (clip), arrastrar sobre o panel e pegar na caixa.
  if (adxApi) {
    entradaFicheiros.addEventListener('change', () => { engadirFicheiros([...entradaFicheiros.files]); entradaFicheiros.value = '' })
    caixa.addEventListener('paste', (e) => {
      const fs = [...(e.clipboardData?.files || [])]
      if (fs.length) { e.preventDefault(); engadirFicheiros(fs) }
    })
    let contadorArrastre = 0
    const teFicheiros = (e) => [...(e.dataTransfer?.types || [])].includes('Files')
    raiz.addEventListener('dragenter', (e) => { if (teFicheiros(e)) { contadorArrastre++; raiz.classList.add('asistente--arrastrando') } })
    raiz.addEventListener('dragover', (e) => { if (teFicheiros(e)) e.preventDefault() })
    raiz.addEventListener('dragleave', (e) => { if (teFicheiros(e) && --contadorArrastre <= 0) { contadorArrastre = 0; raiz.classList.remove('asistente--arrastrando') } })
    raiz.addEventListener('drop', (e) => {
      if (!teFicheiros(e)) return
      e.preventDefault()
      contadorArrastre = 0
      raiz.classList.remove('asistente--arrastrando')
      engadirFicheiros([...e.dataTransfer.files])
    })
  }

  const pe = el('footer', { class: 'asistente__pe' },
    zonaAdx, erroAdx, entradaFicheiros, menuClip,
    caixa,
    el('div', { class: 'asistente__accions' },
      btnClip,
      el('label', { class: 'asistente__confirmar' }, chkConfirmar, el('span', { class: 'asistente__confirmar-txt' })),
      el('span', { class: 'asistente__espazo' }), btnParar, btnEnviar))

  const cab = el('header', { class: 'asistente__cab' },
    logo, el('div', { class: 'asistente__cab-txt' }, titulo, subtitulo), el('span', { class: 'asistente__espazo' }), btnNova, btnPechar)
  secTarefas.append(el('h3', { class: 'asistente__h', 'data-h': 'tarefas' }), zonaTarefas)
  raiz.append(cab, aviso, secBoveda, secTarefas, conversa, pe)
  montarEn.append(raiz)

  // ---------- Texto traducible ----------

  function pintarTextos() {
    raiz.setAttribute('aria-label', t('asistente.titulo', 'Asistente de IA'))
    titulo.textContent = txt('titulo', t('asistente.titulo', 'Asistente de IA'))
    btnNova.title = btnNova.ariaLabel = t('asistente.nova', 'Nova conversa')
    btnPechar.title = btnPechar.ariaLabel = t('asistente.pechar', 'Pechar o asistente')
    secTarefas.querySelector('[data-h]').textContent = t('asistente.tarefas', 'Tarefas')
    caixa.placeholder = txt('caixaPh', t('asistente.caixa.ph', 'Pídeme o que queiras: «reescribe a introdución máis formal», «que lei regula…?». Se premes unha tarefa, isto engádese como instrución adicional.'))
    chkConfirmar.parentElement.querySelector('.asistente__confirmar-txt').textContent = t('asistente.confirmar', 'Confirmar cada cambio')
    if (btnClip) btnClip.title = btnClip.ariaLabel = t('asistente.adx.clip', 'Adxuntar ficheiros: imaxes, PDF, Word, OpenDocument… (tamén podes arrastralos ou pegalos aquí)')
    btnEnviar.innerHTML = `${ICONAS.enviar}<span>${t('asistente.enviar', 'Enviar')}</span>`
    btnParar.innerHTML = `${ICONAS.parar}<span>${t('asistente.parar', 'Parar')}</span>`
    pintarTarefas()
    pintarEstadoIA()
    pintarBoveda()
  }

  function pintarEstadoIA() {
    logo.innerHTML = logoProvedor(estadoIA.preset)
    const nome = nomeProvedor(estadoIA.preset)
    subtitulo.textContent = estadoIA.configurada ? `${nome} · ${estadoIA.modelo}` : t('asistente.sen.ia.curto', 'Sen IA configurada')
    aviso.replaceChildren()
    aviso.hidden = estadoIA.configurada
    if (!estadoIA.configurada) {
      aviso.append(
        el('p', { text: abrirOpcions
          ? t('asistente.sen.ia', 'Para usar o asistente hai que configurar unha IA en Opcións (provedor, clave da API e modelo).')
          : t('asistente.sen.ia.xanela', 'Para usar o asistente hai que configurar unha IA en Opcións (na xanela principal de Lúa: provedor, clave da API e modelo). Despois volve abrir este panel.') }),
        ...(abrirOpcions ? [el('button', { type: 'button', class: 'asistente__btn-opcions', text: t('asistente.abrir.opcions', 'Abrir Opcións'), onclick: () => abrirOpcions() })] : []))
    }
    const bloqueado = !estadoIA.configurada || ocupado
    btnEnviar.disabled = bloqueado
    if (btnClip) btnClip.disabled = !estadoIA.configurada
    zonaTarefas.querySelectorAll('button.asistente__chip').forEach((b) => { b.disabled = bloqueado })
    aoCambiar({ ocupado, configurada: estadoIA.configurada, preset: estadoIA.preset })
  }

  // Selector do tipo de adxunto: PDF, Markdown ou outros (imaxes, Word...).
  const TIPOS_ADX = [
    { clave: 'pdf', accept: '.pdf,application/pdf', def: 'PDF (.pdf)' },
    { clave: 'md', accept: '.md,.markdown,text/markdown', def: 'Markdown (.md)' },
    { clave: 'outros', accept: '', def: 'Imaxes, Word, OpenDocument…' },
  ]
  function pintarMenuClip() {
    menuClip.replaceChildren(...TIPOS_ADX.map((x) => el('button', {
      type: 'button', role: 'menuitem', text: t('asistente.adx.tipo.' + x.clave, x.def),
      onclick: () => { menuClip.hidden = true; entradaFicheiros.accept = x.accept; entradaFicheiros.click() },
    })))
  }
  if (adxApi) {
    pintarMenuClip()
    document.addEventListener('click', (e) => { if (!menuClip.hidden && !menuClip.contains(e.target) && !btnClip.contains(e.target)) menuClip.hidden = true })
  }

  // ---------- Bóveda de Obsidian ----------

  const documentoDaBoveda = () => (typeof documentoBoveda === 'function' ? documentoBoveda() : documentoBoveda)

  async function pintarBoveda() {
    const doc = documentoDaBoveda()
    if (!doc || !api.BovedaEstadoDe) { secBoveda.hidden = true; return }
    let e
    try { e = await api.BovedaEstadoDe(doc) } catch { e = { conectada: false } }
    secBoveda.hidden = false
    secBoveda.replaceChildren()
    const erro = el('p', { class: 'asistente__boveda-erro', hidden: true })
    const escoller = async () => {
      erro.hidden = true
      try {
        const ruta = await api.BovedaEscoller()
        if (!ruta) return
        await api.BovedaConectar(doc, ruta)
      } catch (err) { erro.textContent = String(err?.message || err); erro.hidden = false; secBoveda.append(erro); return }
      pintarBoveda()
    }
    const reconectar = e.proxectoDesconectada && api.BovedaProxectoConectar
      ? el('button', {
        type: 'button', class: 'asistente__boveda-conectar', text: '📚 ' + t('asistente.boveda.reconectar', 'Reconectar a bóveda do proxecto'),
        onclick: async () => { try { await api.BovedaProxectoConectar(true) } catch { /* nada */ } pintarBoveda() },
      })
      : null
    if (!e.conectada) {
      secBoveda.append(...[reconectar, el('button', {
        type: 'button', class: 'asistente__boveda-conectar', onclick: escoller,
        title: t('asistente.boveda.conectar.title', 'Conecta unha bóveda de Obsidian (cartafoles raw/ e wiki/ e un CLAUDE.md) para que o asistente a use como contexto'),
      }, el('span', { text: '📚 ' + t('asistente.boveda.conectar', 'Conectar unha bóveda de Obsidian') }))].filter(Boolean))
      return
    }
    const info = e.erro
      ? e.erro
      : t('asistente.boveda.info', '{n} fontes · {w} páxinas wiki', { n: e.raw, w: e.wiki }) + (e.avisos?.length ? ' · ' + e.avisos.join(' ') : '')
    // A bóveda do proxecto conéctase soa en canto existe; só se pode desconectar.
    const doProxecto = e.proxecto && api.BovedaProxectoConectar
    secBoveda.append(...[
      el('div', { class: 'asistente__boveda-txt', title: e.ruta },
        el('strong', { text: '📚 ' + (doProxecto ? t('asistente.boveda.proxecto', 'Bóveda do proxecto') : e.nome) }),
        el('small', { class: e.erro ? 'asistente__boveda-erro' : '', text: info })),
      doProxecto ? null : el('button', { type: 'button', text: t('asistente.boveda.cambiar', 'Cambiar'), onclick: escoller }),
      el('button', {
        type: 'button', text: t('asistente.boveda.desconectar', 'Desconectar'),
        onclick: async () => {
          try { await (doProxecto ? api.BovedaProxectoConectar(false) : api.BovedaConectar(doc, '')) } catch { /* nada */ }
          pintarBoveda()
        },
      }),
      reconectar,
      erro].filter(Boolean))
  }

  // A bóveda pode crearse noutra xanela (Buxo): ao volver aquí actualízase.
  window.addEventListener('focus', () => { if (!secBoveda.hidden || documentoDaBoveda()) pintarBoveda() })

  // ---------- Tarefas (fichas) ----------

  let formularioAberto = false

  function pintarTarefas() {
    zonaTarefas.replaceChildren()
    const todas = [...tarefas, ...lerPropias(almacen, claveTarefas)]
    for (const tf of todas) {
      const nome = tf.propia ? tf.titulo : t(tf.clave, tf.titulo)
      const chip = el('button', {
        type: 'button', class: 'asistente__chip' + (tf.propia ? ' asistente__chip--propia' : ''),
        title: corta(tf.instrucion, 400), onclick: () => executarTarefa(tf),
      }, el('span', { class: 'asistente__chip-ico', text: tf.icona || '★' }), el('span', { text: nome }))
      if (tf.propia) {
        const xa = el('span', { class: 'asistente__chip-x', role: 'button', tabindex: '0', title: t('asistente.tarefa.eliminar', 'Eliminar esta tarefa'), html: ICONAS.pechar })
        const borrar = (e) => { e.stopPropagation(); e.preventDefault(); eliminarPropia(tf.id, almacen, claveTarefas); pintarTarefas() }
        xa.addEventListener('click', borrar)
        xa.addEventListener('keydown', (e) => { if (e.key === 'Enter' || e.key === ' ') borrar(e) })
        chip.append(xa)
      }
      zonaTarefas.append(chip)
    }
    zonaTarefas.append(el('button', {
      type: 'button', class: 'asistente__chip asistente__chip--nova', html: ICONAS.mais + `<span>${t('asistente.tarefa.nova', 'Nova tarefa')}</span>`,
      title: t('asistente.tarefa.nova.title', 'Engade unha tarefa propia á lista: un nome e a instrución que se lle manda á IA'),
      onclick: () => { formularioAberto = !formularioAberto; pintarFormulario() },
    }))
    pintarFormulario()
    const bloqueado = !estadoIA.configurada || ocupado
    zonaTarefas.querySelectorAll('button.asistente__chip:not(.asistente__chip--nova)').forEach((b) => { b.disabled = bloqueado })
  }

  function pintarFormulario() {
    secTarefas.querySelector('.asistente__nova')?.remove()
    if (!formularioAberto) return
    const nome = el('input', { type: 'text', placeholder: t('asistente.tarefa.nome.ph', 'Nome da tarefa (ex.: Resumir en tres liñas)') })
    const instrucion = el('textarea', { rows: '3', placeholder: t('asistente.tarefa.instrucion.ph', 'Que lle pides á IA (ex.: Resume cada tarefa en tres liñas e mantén o formato)') })
    const erro = el('p', { class: 'asistente__erro-form', hidden: true })
    const gardar = () => {
      try {
        engadirPropia(nome.value, instrucion.value, almacen, claveTarefas)
        formularioAberto = false
        pintarTarefas()
      } catch (e) {
        erro.textContent = String(e.message || e)
        erro.hidden = false
      }
    }
    secTarefas.append(el('div', { class: 'asistente__nova' }, nome, instrucion, erro,
      el('div', { class: 'asistente__nova-acc' },
        el('button', { type: 'button', text: t('dialog.cancel', 'Cancelar'), onclick: () => { formularioAberto = false; pintarFormulario() } }),
        el('button', { type: 'button', class: 'primary', text: t('asistente.tarefa.gardar', 'Gardar tarefa'), onclick: gardar }))))
    nome.focus()
  }

  // ---------- Conversa ----------

  const baixar = () => { conversa.scrollTop = conversa.scrollHeight }

  function pintarMensaxe(clase, ...contidos) {
    const n = el('div', { class: 'asis-msg ' + clase }, ...contidos)
    conversa.append(n)
    baixar()
    return n
  }

  const pintarUsuario = (texto, adxuntos = []) => pintarMensaxe('asis-msg--user',
    el('div', { class: 'asis-txt', text: texto }),
    adxuntos.length ? el('div', { class: 'asis-msg__adx' }, adxuntos.map((a) => el('span', { class: 'asis-adx', title: a.nome, text: '📎 ' + corta(a.nome, 28) }))) : null)
  function pintarIA(texto) {
    const c = el('div', { class: 'asis-md' })
    c.innerHTML = markdownAHtml(texto, {})
    return pintarMensaxe('asis-msg--ia', c)
  }
  const pintarAviso = (texto, tipo = 'aviso') => pintarMensaxe('asis-msg--' + tipo, el('div', { class: 'asis-txt', text: texto }))

  let pensando = null
  function mostrarPensando(si) {
    pensando?.remove()
    pensando = null
    if (!si) return
    pensando = el('div', { class: 'asis-pensando' }, el('span'), el('span'), el('span'))
    conversa.append(pensando)
    baixar()
  }

  const etiquetaFerr = (nome) => t('asistente.ferr.' + nome, ETIQUETAS_FERRAMENTAS[nome] || nome)
  function detalleFerr(args) {
    for (const k of CLAVES_DETALLE) if (args && args[k]) return corta(args[k], 60)
    return ''
  }

  // Cada chamada a unha ferramenta pintase como unha liña despregable.
  const ferrPintadas = new Map()
  function pintarFerramenta(ev) {
    const estado = el('span', { class: 'asis-ferr__estado', text: '…' })
    const detalle = detalleFerr(ev.args)
    const pre = el('pre', { class: 'asis-ferr__pre', text: JSON.stringify(ev.args || {}, null, 2) })
    const d = el('details', { class: 'asis-ferr asis-ferr--traballando' },
      el('summary', {}, el('span', { class: 'asis-ferr__ico', text: '⚙' }),
        el('span', { class: 'asis-ferr__nome', text: etiquetaFerr(ev.chamada.nome) }),
        detalle ? el('span', { class: 'asis-ferr__detalle', text: detalle }) : null, estado),
      pre)
    conversa.append(d)
    ferrPintadas.set(ev.chamada.id, { d, estado, pre })
    baixar()
  }
  function rematarFerramenta(ev) {
    const p = ferrPintadas.get(ev.chamada.id)
    if (!p) return
    p.d.classList.remove('asis-ferr--traballando')
    const r = ev.resultado
    p.d.classList.add(r.erro ? 'asis-ferr--erro' : 'asis-ferr--ok')
    p.estado.textContent = r.rexeitada ? '✋' : r.erro ? '✗' : '✓'
    p.d.append(el('pre', { class: 'asis-ferr__pre asis-ferr__res', text: corta(r.contido, 4000) }))
    // As imaxes que se lle amosan á IA (adxunto_ver, plantilla_previsualizar...)
    // vense tamén aquí, para que o usuario saiba o que está mirando.
    if ((r.imaxes || []).length && api.AdxuntoDataURI) {
      const galeria = el('div', { class: 'asis-ferr__imaxes' })
      p.d.append(galeria)
      for (const id of r.imaxes) {
        api.AdxuntoDataURI(id, 640).then((uri) => {
          galeria.append(el('img', { class: 'asis-ferr__img', src: uri, alt: id }))
          baixar()
        }).catch(() => {})
      }
    }
    baixar()
  }

  function aoEvento(ev) {
    switch (ev.tipo) {
      case 'pensando': mostrarPensando(true); break
      case 'texto': mostrarPensando(false); pintarIA(ev.texto); break
      case 'ferramenta': mostrarPensando(false); pintarFerramenta(ev); break
      case 'resultado': rematarFerramenta(ev); break
      case 'aviso': mostrarPensando(false); pintarAviso(ev.texto); break
      default: break
    }
  }

  // ---------- Tarxetas interactivas (confirmar / preguntar) ----------

  function tarxeta(clase, ...fillos) {
    pendentes++
    abrir()
    const n = el('div', { class: 'asis-msg asis-tarxeta ' + clase }, ...fillos)
    conversa.append(n)
    baixar()
    return { n, feita: () => { pendentes--; n.classList.add('asis-tarxeta--feita'); n.querySelectorAll('button, input').forEach((x) => { x.disabled = true }) } }
  }

  function resumoConfirmacion(def, args) {
    if (def.nome === 'plantilla_gardar') {
      const campos = Object.keys(args).filter((k) => k !== 'id')
      const accion = args.id ? t('asistente.conf.modificar', 'Modificar a plantilla') : t('asistente.conf.crear', 'Crear unha plantilla')
      return `${accion}${args.nome ? ` «${args.nome}»` : ''}${args.id ? ` (${args.id})` : ''}. ${t('asistente.conf.campos', 'Campos')}: ${campos.join(', ') || '—'}.`
    }
    if (def.nome === 'plantilla_eliminar') return `${t('asistente.conf.eliminar', 'Eliminar a plantilla')} ${args.id}. ${t('asistente.conf.irreversible', 'Non se pode desfacer.')}`
    if (def.nome === 'curso_novo') return t('asistente.conf.novo', 'Crear un curso novo baleiro (o actual péchase).')
    return `${etiquetaFerr(def.nome)}: ${corta(JSON.stringify(args), 300)}`
  }

  function confirmar(def, args) {
    return new Promise((resolve) => {
      const { n, feita } = tarxeta('asis-tarxeta--conf',
        el('div', { class: 'asis-tarxeta__tit', text: t('asistente.conf.tit', 'O asistente quere facer isto:') }),
        el('div', { class: 'asis-tarxeta__txt', text: resumoConfirmacion(def, args) }))
      const fin = (ok) => { feita(); n.append(el('div', { class: 'asis-tarxeta__res', text: ok ? '✓ ' + t('asistente.conf.permitido', 'Permitido') : '✋ ' + t('asistente.conf.rexeitado', 'Rexeitado') })); resolve(ok) }
      n.append(el('div', { class: 'asis-tarxeta__acc' },
        el('button', { type: 'button', text: t('asistente.conf.rexeitar', 'Rexeitar'), onclick: () => fin(false) }),
        el('button', { type: 'button', class: 'primary', text: t('asistente.conf.permitir', 'Permitir'), onclick: () => fin(true) })))
      n.querySelector('.primary').focus()
    })
  }

  function preguntar(pregunta, opcions = []) {
    return new Promise((resolve) => {
      const { n, feita } = tarxeta('asis-tarxeta--preg',
        el('div', { class: 'asis-tarxeta__tit', text: t('asistente.preg.tit', 'O asistente pregunta:') }),
        el('div', { class: 'asis-tarxeta__txt', text: pregunta }))
      const fin = (r) => { feita(); n.append(el('div', { class: 'asis-tarxeta__res', text: r === null ? '— ' + t('asistente.preg.sen', 'Sen resposta') : '→ ' + r })); resolve(r) }
      if (opcions.length) {
        n.append(el('div', { class: 'asis-tarxeta__opcions' }, opcions.map((o) => el('button', { type: 'button', text: o, onclick: () => fin(o) }))))
      }
      const campo = el('input', { type: 'text', placeholder: opcions.length ? t('asistente.preg.outra', 'Outra resposta…') : t('asistente.preg.resposta', 'A túa resposta…') })
      const ok = () => { if (campo.value.trim()) fin(campo.value.trim()) }
      campo.addEventListener('keydown', (e) => { if (e.key === 'Enter' && !e.isComposing) { e.preventDefault(); ok() } })
      n.append(el('div', { class: 'asis-tarxeta__acc' }, campo,
        el('button', { type: 'button', text: t('asistente.preg.non', 'Non responder'), onclick: () => fin(null) }),
        el('button', { type: 'button', class: 'primary', text: t('asistente.preg.enviar', 'Responder'), onclick: ok })))
      campo.focus()
    })
  }

  // ---------- Execución ----------

  function fixarOcupado(v) {
    ocupado = v
    btnParar.hidden = !v
    btnEnviar.hidden = v
    caixa.disabled = false
    raiz.classList.toggle('asistente--ocupado', v)
    pintarEstadoIA()
  }

  async function correr(fn) {
    if (ocupado) return
    sinal = { abortado: false }
    fixarOcupado(true)
    try {
      await fn()
    } catch (err) {
      mostrarPensando(false)
      const e = pintarMensaxe('asis-msg--erro',
        el('div', { class: 'asis-txt', text: '⚠ ' + String(err?.message || err) }))
      e.append(el('button', { type: 'button', class: 'asis-reintentar', text: t('asistente.reintentar', 'Reintentar'), onclick: () => { e.remove(); correr(() => retomar(paramsBucle())) } }))
    } finally {
      mostrarPensando(false)
      fixarOcupado(false)
    }
  }

  function paramsBucle() {
    return {
      api, executar, ferramentas, historial, estado: estadoPrograma, aoEvento, confirmar,
      confirmarTodo: () => confirmarTodo, sinal,
    }
  }

  async function carregarFerramentas() {
    if (ferramentas.length) return
    try { ferramentas = (await api.IAFerramentas()) || [] } catch { ferramentas = [] }
    // As boveda_* só as ve a IA se hai bóveda, pero o bucle precisa o seu tipo.
    try { if (api.IAFerramentasBoveda) ferramentas = [...ferramentas, ...((await api.IAFerramentasBoveda()) || [])] } catch { /* sen bóveda */ }
  }

  // Se hai adxuntos e non se escribiu nada, a mensaxe di só que os mire.
  const TEXTO_SO_ADXUNTOS = () => t('asistente.adx.sen.texto', 'Mira os ficheiros adxuntos.')

  async function enviar(texto, mostrar) {
    await carregarFerramentas()
    const adx = adxPend
    adxPend = []
    pintarAdxuntos()
    pintarUsuario(mostrar ?? texto, adx)
    await correr(() => executarConversa({ ...paramsBucle(), texto, adxuntos: adx.map((a) => a.id) }))
  }

  function enviarLibre() {
    let texto = caixa.value.trim()
    if ((!texto && !adxPend.length) || ocupado || !estadoIA.configurada) return
    if (!texto) texto = TEXTO_SO_ADXUNTOS()
    caixa.value = ''
    enviar(texto)
  }

  // ---------- Adxuntos ----------

  const TAMANO_MAX_ADX = 30 * 1024 * 1024
  const tamanoLexible = (n) => (n >= 1048576 ? `${(n / 1048576).toFixed(1)} MB` : n >= 1024 ? `${Math.round(n / 1024)} KB` : `${n} B`)
  const ICONA_TIPO = { imaxe: '🖼', pdf: '📕', docx: '📘', odt: '📄', presentacion: '📊', folla: '📈', texto: '📝', rtf: '📄', antigo: '📄', outro: '📎' }

  function pintarAdxuntos() {
    zonaAdx.replaceChildren()
    zonaAdx.hidden = adxPend.length === 0
    for (const a of adxPend) {
      const icona = a.vista
        ? el('img', { class: 'asis-adx-chip__mini', src: a.vista, alt: '' })
        : el('span', { class: 'asis-adx-chip__ico', text: ICONA_TIPO[a.tipo] || '📎' })
      zonaAdx.append(el('span', { class: 'asis-adx-chip', title: `${a.nome} (${a.tipo}, ${tamanoLexible(a.tamano)}) · ${a.id}` },
        icona, el('span', { class: 'asis-adx-chip__nome', text: corta(a.nome, 26) }),
        el('span', { class: 'asis-adx-chip__tam', text: tamanoLexible(a.tamano) }),
        el('button', {
          type: 'button', class: 'asis-adx-chip__x', html: ICONAS.pechar, title: t('asistente.adx.quitar', 'Quitar este adxunto'),
          onclick: () => { adxPend = adxPend.filter((x) => x.id !== a.id); pintarAdxuntos(); adxApi.AdxuntoEliminar?.(a.id).catch(() => {}) },
        })))
    }
  }

  const lerBase64 = (file) => new Promise((resolve, reject) => {
    const r = new FileReader()
    r.onerror = () => reject(new Error('non se puido ler o ficheiro'))
    r.onload = () => resolve(String(r.result).slice(String(r.result).indexOf(',') + 1))
    r.readAsDataURL(file)
  })

  // engadirFicheiros sube cada ficheiro ao almacén temporal (Go) e amósao coma
  // adxunto pendente da seguinte mensaxe.
  async function engadirFicheiros(ficheiros) {
    if (!adxApi || !estadoIA.configurada) return
    erroAdx.hidden = true
    const erros = []
    for (const f of ficheiros) {
      const nome = f.name || 'imaxe-pegada.png'
      try {
        if (f.size > TAMANO_MAX_ADX) throw new Error(t('asistente.adx.gordo', 'pesa máis de 30 MB'))
        const ax = await adxApi.AdxuntoGardar(nome, await lerBase64(f))
        let vista = ''
        if (ax.tipo === 'imaxe' && adxApi.AdxuntoVista) { try { vista = await adxApi.AdxuntoVista(ax.id) } catch { /* sen miniatura */ } }
        adxPend.push({ id: ax.id, nome: ax.nome, tipo: ax.tipo, tamano: ax.tamano, vista })
        pintarAdxuntos()
      } catch (err) {
        erros.push(`${nome}: ${String(err?.message || err)}`)
      }
    }
    if (erros.length) { erroAdx.textContent = erros.join(' · '); erroAdx.hidden = false }
  }

  function executarTarefa(tf) {
    if (ocupado || !estadoIA.configurada) return
    const extra = caixa.value
    caixa.value = ''
    const nome = tf.propia ? tf.titulo : t(tf.clave, tf.titulo)
    enviar(instrucionFinal(tf, extra), extra.trim() ? `${nome} — ${extra.trim()}` : nome)
  }

  function parar() {
    sinal.abortado = true
    btnParar.disabled = true
    setTimeout(() => { btnParar.disabled = false }, 1500)
  }

  function novaConversa() {
    if (ocupado) sinal.abortado = true
    historial = []
    adxPend = []
    pintarAdxuntos()
    erroAdx.hidden = true
    adxApi?.AdxuntosLimpar?.().catch(() => {}) // os adxuntos son da sesión: bórranse coa conversa
    ferrPintadas.clear()
    conversa.replaceChildren()
    mostrarBenvida()
  }

  function mostrarBenvida() {
    conversa.append(el('div', { class: 'asis-benvida' },
      el('p', { text: txt('benvida', t('asistente.benvida', 'Escolle unha tarefa ou escribe o que necesitas. Podo ler e editar o curso, xerar e exportar o informe, xestionar plantillas, cambiar Opcións e buscar lexislación na web.')) }),
      el('p', { class: 'asis-benvida__nota', text: txt('benvidaNota', t('asistente.benvida.nota', 'Os cambios no curso pódense desfacer co botón Desfacer.')) })))
  }

  // ---------- API ----------

  async function refrescar() {
    try { estadoIA = await api.IAAsistenteEstado() } catch { /* mantense o anterior */ }
    pintarEstadoIA()
  }

  function abrir() {
    raiz.classList.remove('hidden')
    if (!incrustado) document.body.classList.add('asistente-aberto')
    refrescar()
    pintarBoveda()
    setTimeout(() => caixa.focus(), 0)
  }
  function pechar() {
    if (pendentes > 0) return // hai unha pregunta agardando resposta: non se pode agochar
    raiz.classList.add('hidden')
    if (!incrustado) document.body.classList.remove('asistente-aberto')
  }
  const estaAberto = () => !raiz.classList.contains('hidden')

  pintarTextos()
  mostrarBenvida()
  refrescar()

  return {
    abrir, pechar, refrescar, estaAberto, refrescarBoveda: pintarBoveda,
    // Quita o panel e para o bucle se estaba a medias (a pantalla que o aloxaba pechouse).
    destruir() { sinal.abortado = true; raiz.remove() },
    alternar: () => (estaAberto() ? pechar() : abrir()),
    refrescarIdioma() {
      pintarTextos()
      if (!historial.length) { conversa.replaceChildren(); mostrarBenvida() }
    },
    // Para as ferramentas usuario_preguntar e as confirmacións.
    preguntar, confirmar,
    // Para probas.
    _historial: () => historial,
    _enviar: (texto) => enviar(texto),
    _ocupado: () => ocupado,
  }
}
