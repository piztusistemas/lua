// montarEditorTarefa - a pantalla "Editar tarefa" (título, cor, mes e descrición
// en Markdown con vista previa). Ocupa TODA a pantalla ou xanela onde se
// monte: ten cabeceira con "X" vermella arriba á dereita, e ao pé Cancelar,
// Aplicar (garda sen pechar) e Gardar (garda e pecha).
//
// Non sabe onde vive: `container` pode ser un overlay dentro da xanela
// principal (cronograma-editor.js, abrirTarefa) ou a páxina enteira dunha
// xanela propia do sistema, que se pode arrastrar a un segundo monitor
// (main.js, entrarModoTarefa). Tampouco sabe onde vai o plan: cada vez que se
// aplica chama a `aoAplicar` e quen o montou decide que facer con iso.
import { confirmar, preguntar } from '../comun/dialogs.js'
import { PALETA, resolverCor, novoId, imaxesUsadas, MESES } from './cronograma-modelo.js'
import {
  envolver, prefixoBloque, alinear, tamanoLetra, ligazon, inserirImaxe, markdownAHtml,
} from './cronograma-md.js'
import { el } from './dom.js'
import { crearPanelAsistente } from '../asistente/panel.js'
import { crearFerramentasTarefa } from '../asistente/ferramentas-tarefa.js'
import { TAREFAS_TAREFA } from '../asistente/tarefas.js'
import { logoProvedor } from '../asistente/logos.js'

const ICONA_X = '<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round"><path d="M5 5l10 10M15 5L5 15"/></svg>'

// normalizarImaxe le un ficheiro de imaxe e devolve un data URI PNG (ou JPEG
// se `saida` é 'jpeg' e o orixinal era JPEG) reducido a `maxLado` píxeles.
// Pasar sempre polo canvas convirte calquera formato que o navegador
// decodifique (PNG, JPG, GIF, BMP, WEBP, SVG...) a algo que LaTeX le. Sae
// como función exportada para poder substituíla nas probas (jsdom non ten
// canvas).
export async function normalizarImaxe(file, { maxLado = 1600, saida = 'png' } = {}) {
  const url = URL.createObjectURL(file)
  try {
    const img = await new Promise((resolve, reject) => {
      const i = new Image()
      i.onload = () => resolve(i)
      i.onerror = () => reject(new Error('non se puido ler a imaxe'))
      i.src = url
    })
    const w0 = img.naturalWidth || img.width
    const h0 = img.naturalHeight || img.height
    if (!w0 || !h0) throw new Error('a imaxe non ten dimensións')
    const k = Math.min(1, maxLado / Math.max(w0, h0))
    const canvas = document.createElement('canvas')
    canvas.width = Math.max(1, Math.round(w0 * k))
    canvas.height = Math.max(1, Math.round(h0 * k))
    const ctx = canvas.getContext('2d')
    const jpeg = saida === 'jpeg' && /jpe?g/i.test(file.type)
    if (jpeg) { ctx.fillStyle = '#fff'; ctx.fillRect(0, 0, canvas.width, canvas.height) }
    ctx.drawImage(img, 0, 0, canvas.width, canvas.height)
    return jpeg ? canvas.toDataURL('image/jpeg', 0.9) : canvas.toDataURL('image/png')
  } finally {
    URL.revokeObjectURL(url)
  }
}

// montarEditorTarefa(container, opcións):
//   t          función de tradución
//   tarefa     a tarefa (obxecto completo; para unha nova, xa co seu id)
//   nova       true se aínda non existe no curso
//   clave      mes ("AAAA-MM") onde vive (ou onde se vai crear)
//   meses      [{clave, nome, ano}] para o selector de mes
//   bloq       curso bloqueado: só se pode ver
//   lerImaxe   (file, {maxLado, saida}) -> data URI
//   aoAplicar  async ({ tarefa, claveOrixe, mesDestino }) - gardar no curso;
//              se lanza un erro amósase e a pantalla non se pecha
//   aoPechar   () - hai que quitar a pantalla (xa confirmado o descarte)
//   aoCambiarSucio  (bool) - avisa cando hai/deixa de haber cambios sen aplicar
//   contexto   {centro, localidade, anoAcademico} para o asistente (opcional)
//   asistente  {api, idioma} para amosar o asistente de IA desta tarefa (opcional):
//              api = {IAAsistenteEstado, IAFerramentasDoAmbito, IATurnoAsistente,
//              IAChamarFerramenta} (os bindings) e idioma() o código da interface
export function montarEditorTarefa(container, {
  t, tarefa: tarefaIn, nova: novaIn, clave, meses, bloq = false, lerImaxe = normalizarImaxe,
  aoAplicar, aoPechar, aoCambiarSucio, contexto = {}, asistente: asistenteOpc = null,
}) {
  let nova = !!novaIn
  let claveActual = clave
  let tarefa = { ...tarefaIn }
  const imaxes = { ...(tarefa.images || {}) }
  let corClave = resolverCor(tarefa.color).clave
  let mesDestino = clave
  let ocupado = false
  let pechado = false
  const instantanea = () => JSON.stringify([titulo.value, desc.value, corClave, mesDestino])

  container.classList.add('tarefa-pantalla')
  container.setAttribute('role', 'dialog')
  container.setAttribute('aria-modal', 'true')

  const titulo = el('input', { type: 'text', value: tarefa.title, placeholder: t('informes.tarefa.titulo.ph', 'Ex.: Avaliación inicial…'), disabled: bloq || undefined })
  const erro = el('p', { class: 'crono-erro', role: 'alert', hidden: true })
  const desc = el('textarea', {
    class: 'crono-md', spellcheck: 'true', disabled: bloq || undefined,
    placeholder: t('informes.tarefa.desc.ph', 'Descrición en Markdown:\n**negrita**, *cursiva*, listas con -'),
  })
  desc.value = tarefa.description
  const vista = el('div', { class: 'crono-md-vista' })
  const pintarVista = () => {
    vista.innerHTML = markdownAHtml(desc.value, imaxes) || `<p class="md-aviso">${t('informes.vista.baleira', 'A vista previa aparece aquí.')}</p>`
  }
  desc.addEventListener('input', pintarVista)

  // Cores
  const swatches = el('div', { class: 'crono-cores', role: 'radiogroup', 'aria-label': t('informes.cor', 'Cor') })
  const pintarCores = () => {
    swatches.replaceChildren(...PALETA.map((c) => el('button', {
      type: 'button', class: 'crono-cor' + (c.clave === corClave ? ' crono-cor--activa' : ''),
      style: `background:${c.fondo};border-color:${c.bordo};color:${c.texto}`, title: c.nome, role: 'radio',
      'aria-checked': String(c.clave === corClave), disabled: bloq || undefined,
      onclick: () => { corClave = c.clave; pintarCores() },
    }, el('span', { text: 'Aa' }))))
  }
  pintarCores()

  // Mes
  const selMes = el('select', { disabled: bloq || undefined, onchange: (e) => { mesDestino = e.target.value } },
    meses.map((m) => el('option', { value: m.clave, text: `${m.nome} ${m.ano}`, selected: m.clave === clave || undefined })))
  if (![...selMes.options].some((o) => o.value === clave)) {
    const [a, m] = clave.split('-').map((x) => parseInt(x, 10))
    selMes.prepend(el('option', { value: clave, text: `${MESES[m - 1]} ${a}`, selected: true }))
  }

  // Barra de formato
  const aplicarTexto = (fn) => {
    const ini = desc.selectionStart
    const fin = desc.selectionEnd
    const r = fn({ texto: desc.value, ini, fin })
    desc.value = r.texto
    desc.focus()
    desc.setSelectionRange(r.ini, r.fin)
    pintarVista()
  }
  const bfmt = (etiqueta, titulo, fn, clase = '') => el('button', {
    type: 'button', class: 'crono-fmt ' + clase, text: etiqueta, title: titulo, 'aria-label': titulo,
    disabled: bloq || undefined, onmousedown: (e) => e.preventDefault(), onclick: () => aplicarTexto(fn),
  })
  const tamanos = el('select', {
    class: 'crono-fmt crono-fmt--sel', title: t('informes.fmt.tamano', 'Tamaño de letra'), disabled: bloq || undefined,
    onchange: (e) => { const v = e.target.value; e.target.value = ''; if (v) aplicarTexto((s) => tamanoLetra(s, v)) },
  },
  el('option', { value: '', text: t('informes.fmt.tamano.cab', 'Tamaño') }),
  el('option', { value: 'peq', text: t('informes.fmt.peq', 'Pequeno') }),
  el('option', { value: 'gr', text: t('informes.fmt.gr', 'Grande') }),
  el('option', { value: 'xgr', text: t('informes.fmt.xgr', 'Moi grande') }))

  const ficheiroImaxe = el('input', { type: 'file', accept: 'image/*', hidden: true })
  ficheiroImaxe.addEventListener('change', async () => {
    const f = ficheiroImaxe.files[0]
    ficheiroImaxe.value = ''
    if (!f) return
    try {
      const uri = await lerImaxe(f, { maxLado: 1600, saida: 'jpeg' })
      const id = 'i' + novoId()
      imaxes[id] = uri
      const alt = f.name.replace(/\.[^.]+$/, '').replace(/[[\]()]/g, '')
      aplicarTexto((s) => inserirImaxe(s, `${id} =60%`, alt))
    } catch (err) {
      erro.textContent = t('informes.imaxe.erro', 'Non se puido inserir a imaxe: {msg}', { msg: err.message || err })
      erro.hidden = false
    }
  })

  const barra = el('div', { class: 'crono-fmtbarra', role: 'toolbar', 'aria-label': t('informes.fmt.aria', 'Formato') },
    bfmt('N', t('informes.fmt.negrita', 'Negrita'), (s) => envolver(s, '**', '**'), 'crono-fmt--n'),
    bfmt('C', t('informes.fmt.cursiva', 'Cursiva'), (s) => envolver(s, '*', '*'), 'crono-fmt--c'),
    tamanos,
    el('span', { class: 'crono-fmt__sep' }),
    bfmt('T', t('informes.fmt.titulo', 'Título'), (s) => prefixoBloque(s, '## ')),
    bfmt('•', t('informes.fmt.lista', 'Lista'), (s) => prefixoBloque(s, '- ')),
    bfmt('1.', t('informes.fmt.listanum', 'Lista numerada'), (s) => prefixoBloque(s, '1. ')),
    bfmt('“”', t('informes.fmt.cita', 'Cita'), (s) => prefixoBloque(s, '> ')),
    el('span', { class: 'crono-fmt__sep' }),
    bfmt('⇤', t('informes.fmt.esq', 'Aliñar á esquerda'), (s) => alinear(s, 'esquerda')),
    bfmt('↔', t('informes.fmt.cen', 'Centrar'), (s) => alinear(s, 'centro')),
    bfmt('⇥', t('informes.fmt.der', 'Aliñar á dereita'), (s) => alinear(s, 'dereita')),
    bfmt('≡', t('informes.fmt.xus', 'Xustificar'), (s) => alinear(s, 'xustificado')),
    el('span', { class: 'crono-fmt__sep' }),
    el('button', {
      type: 'button', class: 'crono-fmt', text: t('informes.fmt.ligazon', 'Ligazón'), disabled: bloq || undefined,
      onmousedown: (e) => e.preventDefault(),
      onclick: async () => {
        const ini = desc.selectionStart
        const fin = desc.selectionEnd
        const url = await preguntar(t('informes.fmt.ligazon.url', 'Enderezo da ligazón (https://…)'), 'https://')
        if (!url || url === 'https://') { desc.focus(); return }
        desc.setSelectionRange(ini, fin)
        aplicarTexto((s) => ligazon(s, url.trim()))
      },
    }),
    el('button', {
      type: 'button', class: 'crono-fmt', text: t('informes.fmt.imaxe', 'Imaxe'), disabled: bloq || undefined,
      onmousedown: (e) => e.preventDefault(), onclick: () => ficheiroImaxe.click(),
    }),
    ficheiroImaxe,
  )

  // ---------- Borrador para o asistente ----------
  // O asistente traballa sobre o BORRADOR (o que hai agora nos campos), non
  // sobre o curso: cada cambio seu apílase para poder desfacelo.
  const pilaIA = []
  const fixarCampos = (c) => {
    if (c.titulo !== undefined) titulo.value = c.titulo
    if (c.descricion !== undefined) { desc.value = c.descricion; pintarVista() }
    if (c.cor !== undefined) { corClave = resolverCor(c.cor).clave; pintarCores() }
    marcarCambios()
  }
  const btnDesfacerIA = el('button', {
    type: 'button', class: 'tarefa-desfacer-ia', hidden: true,
    title: t('informes.tarefa.desfacer.ia.title', 'Volver ao texto anterior ao último cambio da IA'),
    text: t('informes.tarefa.desfacer.ia', '↶ Desfacer cambio da IA'),
    onclick: () => {
      const anterior = pilaIA.pop()
      if (anterior) fixarCampos(anterior)
      btnDesfacerIA.hidden = pilaIA.length === 0
    },
  })
  const editorParaIA = {
    bloqueado: bloq,
    ler: () => ({
      titulo: titulo.value, descricion: desc.value, cor: corClave, mes: mesDestino, nova,
      cambiosSenAplicar: cambiado(), imaxes: Object.keys(imaxes),
      mesesDisponibles: meses.map((m) => `${m.clave} (${m.nome} ${m.ano})`),
    }),
    cambiar: (c) => {
      if (pechado) throw new Error('a pantalla desta tarefa xa está pechada')
      pilaIA.push({ titulo: titulo.value, descricion: desc.value, cor: corClave })
      btnDesfacerIA.hidden = false
      fixarCampos(c)
    },
  }
  const resumoParaIA = () => {
    const mes = meses.find((m) => m.clave === mesDestino)
    return [
      `- Tarefa ${nova ? 'nova (aínda non gardada no curso)' : 'existente'}, no mes ${mes ? `${mes.nome} ${mes.ano}` : mesDestino}; cor: ${corClave}.`,
      `- Título actual: «${titulo.value}»; a descrición ten ${desc.value.length} caracteres${Object.keys(imaxes).length ? ` e ${Object.keys(imaxes).length} imaxe(s)` : ''}.`,
      `- Curso: ${contexto.anoAcademico || '(sen indicar)'}; centro: ${contexto.centro || '(sen indicar)'}; localidade: ${contexto.localidade || '(sen indicar)'}.`,
    ].join('\n')
  }

  // Cabeceira: título da pantalla, estado e "X" vermella
  const cabTitulo = el('h2', { class: 'tarefa-cab__titulo' })
  const estado = el('span', { class: 'tarefa-cab__estado', 'aria-live': 'polite' })
  const pintarTitulo = () => {
    cabTitulo.textContent = nova ? t('informes.tarefa.nova', 'Nova tarefa')
      : bloq ? t('informes.tarefa.ver', 'Ver a tarefa') : t('informes.tarefa.editar', 'Editar tarefa')
  }
  pintarTitulo()
  const botonX = el('button', {
    type: 'button', class: 'tarefa-x', html: ICONA_X,
    title: t('informes.tarefa.pechar', 'Pechar'), 'aria-label': t('informes.tarefa.pechar', 'Pechar'),
    onclick: () => cancelar(),
  })

  const cambiado = () => instantanea() !== inicial
  let inicial = instantanea()
  let sucioAvisado = false
  const marcarCambios = () => {
    if (bloq) return
    const sucio = cambiado()
    estado.textContent = sucio ? t('informes.tarefa.sen.aplicar', '● Cambios sen aplicar') : ''
    if (sucio !== sucioAvisado) { sucioAvisado = sucio; aoCambiarSucio?.(sucio) }
  }
  container.addEventListener('input', marcarCambios)
  container.addEventListener('change', marcarCambios)
  container.addEventListener('click', () => setTimeout(marcarCambios, 0))

  function pechar() {
    pechado = true
    asistente?.destruir()
    document.removeEventListener('keydown', onKeydown, true)
    container.classList.remove('tarefa-pantalla')
    aoPechar()
  }

  async function cancelar() {
    if (ocupado) return
    if (cambiado() && !bloq) {
      const ok = await confirmar(t('informes.tarefa.descartar', 'Hai cambios sen gardar nesta tarefa. Descartalos?'),
        { aceptar: t('informes.descartar', 'Descartar'), cancelar: t('informes.seguir', 'Seguir editando') })
      if (!ok) return
    }
    pechar()
  }

  // aplicar garda no curso sen pechar. Devolve true se se gardou (ou unha
  // promesa dese valor, se `aoAplicar` é asíncrono: a xanela propia manda a
  // tarefa á principal por un evento). É síncrono cando pode, para que
  // Gardar peche ao instante no caso normal.
  function aplicar() {
    if (bloq || ocupado) return false
    if (!titulo.value.trim()) {
      erro.textContent = t('informes.tarefa.faltaTitulo', 'A tarefa debe ter un título.')
      erro.hidden = false
      titulo.focus()
      return false
    }
    erro.hidden = true
    const novaTarefa = { ...tarefa, title: titulo.value.trim(), description: desc.value, color: corClave }
    const usadas = imaxesUsadas(desc.value, imaxes)
    if (Object.keys(usadas).length) novaTarefa.images = usadas
    else delete novaTarefa.images
    ocupado = true
    const feito = () => {
      ocupado = false
      tarefa = novaTarefa
      claveActual = mesDestino
      nova = false
      pintarTitulo()
      inicial = instantanea()
      estado.textContent = t('informes.tarefa.aplicado', '✓ Aplicado')
      if (sucioAvisado) { sucioAvisado = false; aoCambiarSucio?.(false) }
      return true
    }
    const fallou = (err) => {
      ocupado = false
      erro.textContent = String(err?.message || err)
      erro.hidden = false
      return false
    }
    let r
    try { r = aoAplicar({ tarefa: novaTarefa, claveOrixe: claveActual, mesDestino }) } catch (err) { return fallou(err) }
    if (r && typeof r.then === 'function') return r.then(feito, fallou)
    return feito()
  }

  function gardar() {
    const r = aplicar()
    if (r && typeof r.then === 'function') return r.then((ok) => { if (ok) pechar() })
    if (r) pechar()
    return undefined
  }

  // Esc en captura para gañarlle ao manexador global de main.js; Ctrl+Intro
  // garda e pecha, Ctrl+S aplica.
  function onKeydown(e) {
    // Escribindo no asistente, as teclas son del (Esc pecha só o seu panel).
    if (e.target && e.target.closest && e.target.closest('.asistente')) return
    if (document.querySelector('.overlay--dialog:not(.crono-overlay)')) return // un diálogo (confirmar/preguntar) por riba manda
    if (e.key === 'Escape') { e.preventDefault(); e.stopPropagation(); cancelar() }
    else if (e.key === 'Enter' && (e.ctrlKey || e.metaKey) && !bloq) { e.preventDefault(); e.stopPropagation(); gardar() }
    else if ((e.key === 's' || e.key === 'S') && (e.ctrlKey || e.metaKey) && !bloq) { e.preventDefault(); e.stopPropagation(); aplicar() }
  }
  document.addEventListener('keydown', onKeydown, true)

  let asistente = null
  const logoAsistente = el('span', { class: 'tarefa-asistente__logo' })
  const btnAsistente = asistenteOpc && !bloq ? el('button', {
    type: 'button', class: 'tarefa-asistente',
    title: t('informes.tarefa.asistente.title', 'Asistente de IA para esta tarefa: corrixe, mellora o formato e busca referencias legais (DOG, BOE...)'),
    onclick: () => asistente?.alternar(),
  }, logoAsistente, el('span', { text: t('toolbar.asistente.label', 'Asistente') })) : null
  const corpo = el('div', { class: 'tarefa-corpo' },
      el('div', { class: 'crono-fila crono-fila--centro' },
        el('label', { class: 'crono-camp' }, el('span', { class: 'crono-camp__etq', text: t('informes.tarefa.titulo', 'Título da tarefa') }), titulo),
        el('div', { class: 'crono-camp crono-camp--cores' }, el('span', { class: 'crono-camp__etq', text: t('informes.cor', 'Cor') }), swatches),
        el('label', { class: 'crono-camp crono-camp--mes' }, el('span', { class: 'crono-camp__etq', text: t('informes.mes', 'Mes') }), selMes)),
      el('div', { class: 'crono-md-area' },
        el('div', { class: 'crono-md-editor' },
          el('span', { class: 'crono-camp__etq', text: t('informes.tarefa.desc', 'Descrición') }), barra, desc),
        el('div', { class: 'crono-md-previa' },
          el('span', { class: 'crono-camp__etq', text: t('informes.vista', 'Vista previa') }), vista)),
      erro)
  const medio = el('div', { class: 'tarefa-medio' }, corpo)
  container.replaceChildren(
    el('header', { class: 'tarefa-cab' }, cabTitulo, estado, btnDesfacerIA, btnAsistente, botonX),
    medio,
    el('div', { class: 'modal-actions tarefa-pe' },
      el('button', { type: 'button', text: bloq ? t('informes.pechar', 'Pechar') : t('dialog.cancel', 'Cancelar'), onclick: cancelar }),
      bloq ? null : el('button', {
        type: 'button', class: 'tarefa-aplicar', text: t('informes.aplicar', 'Aplicar'),
        title: t('informes.aplicar.title', 'Gardar os cambios no curso sen pechar esta pantalla (Ctrl+S)'),
        onclick: () => { aplicar() },
      }),
      bloq ? null : el('button', { type: 'button', class: 'primary', text: t('informes.gardar', 'Gardar'), onclick: gardar })),
  )
  pintarVista()
  titulo.focus()
  if (!nova) titulo.select()

  // Asistente de IA da tarefa: panel incrustado á dereita, co ámbito limitado a
  // este borrador (e á web, para as referencias legais).
  if (btnAsistente) {
    const ferr = crearFerramentasTarefa({
      editor: editorParaIA,
      api: { IAChamarFerramenta: asistenteOpc.api.IAChamarFerramenta },
      ui: {
        avisar: (m) => { estado.textContent = m },
        preguntar: (p, o) => asistente.preguntar(p, o),
      },
    })
    asistente = crearPanelAsistente({
      t,
      api: {
        IAAsistenteEstado: asistenteOpc.api.IAAsistenteEstado,
        IAFerramentas: () => asistenteOpc.api.IAFerramentasDoAmbito('tarefa'),
        IATurnoAsistente: (req) => asistenteOpc.api.IATurnoAsistente({ ...req, ambito: 'tarefa', idioma: asistenteOpc.idioma?.() || '' }),
        // Bóveda de Obsidian do cronograma (a tarefa pertence a el).
        BovedaEstadoDe: asistenteOpc.api.BovedaEstadoDe, BovedaConectar: asistenteOpc.api.BovedaConectar,
        BovedaEscoller: asistenteOpc.api.BovedaEscoller, IAFerramentasBoveda: asistenteOpc.api.IAFerramentasBoveda,
      },
      documentoBoveda: 'cronograma',
      executar: ferr.executar,
      estadoPrograma: resumoParaIA,
      tarefas: TAREFAS_TAREFA,
      claveTarefas: 'lua.asistente.tarefas.tarefa',
      montarEn: medio,
      textos: {
        titulo: t('asistente.tt.titulo', 'Asistente da tarefa'),
        benvida: t('asistente.tt.benvida', 'Traballo só sobre esta tarefa: corrixo a ortografía, mellora a redacción e o formato, e busco referencias legais (BOE, DOG...). Os cambios entran no borrador; ti decides se os aplicas.'),
        benvidaNota: t('asistente.tt.nota', 'Podes desfacer o último cambio co botón «Desfacer cambio da IA».'),
        caixaPh: t('asistente.tt.ph', 'Pídeme o que queiras sobre esta tarefa: «faina máis formal», «que decreto regula isto?»… Se premes unha ficha, isto engádese como instrución adicional.'),
      },
      aoCambiar: ({ ocupado: ocup, preset }) => {
        logoAsistente.innerHTML = logoProvedor(preset)
        btnAsistente.classList.toggle('tarefa-asistente--ocupado', ocup)
      },
    })
  }

  return {
    // Só para o asistente e as probas: hai cambios sen aplicar?
    cambiado,
    // Trae ao primeiro plano o campo do título.
    focus: () => titulo.focus(),
    // Quita a pantalla sen preguntar (a xanela pechouse por outro lado).
    destruir() {
      pechado = true
      asistente?.destruir()
      document.removeEventListener('keydown', onKeydown, true)
      container.replaceChildren()
      container.classList.remove('tarefa-pantalla')
    },
    // Repite o pechado normal (con confirmación se hai cambios).
    pedirPechar: cancelar,
  }
}
