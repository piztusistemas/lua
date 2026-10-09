// montarFluxoEditor - o editor gráfico das tuberías dun formulario.
//
// Tres columnas: á esquerda os DATOS (un nodo por campo do formulario), no
// medio as TRANSFORMACIÓNS (cálculos, filtros... libres, arrastrables) e á
// dereita o DOCUMENTO (os bloques que saen, na orde en que saen). As
// ligazóns créanse arrastrando desde o punto de saída dun nodo ata un porto de
// entrada de outro; só deixan soltar onde o tipo cadra (un texto non entra
// nunha fórmula).
//
// O grafo é o JSON que avalía Go: este editor só o edita (fluxo-modelo.js) e
// PIDE a Go que o avalíe (AvaliarFluxoFormulario) para amosar en cada nodo o
// valor que dá cos datos actuais e o erro se o ten. Non calcula nada.
import { el } from '../informes/dom.js'
import { iniciarArrastre } from './arrastrar.js'
import { svgTipo } from './iconas.js'
import {
  indexarCatalogo, aceptaTipo, portosDe, novoNodo, conectar, desconectar, eliminarNodo, moverNoDocumento, poñerConfig,
  poñerPosicion, nodosPorColumna, posicionInicial, ordePorto, nodoPorId, eDestino, profundidades,
} from './fluxo-modelo.js'

const ICONA_X = '<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round"><path d="M5 5l10 10M15 5L5 15"/></svg>'
const ICONA_SUBIR = '<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="m5 12 5-5 5 5"/></svg>'
const ICONA_BAIXAR = '<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="m5 8 5 5 5-5"/></svg>'

const TRANSFORMACIONS = ['calcular', 'agregar', 'filtrar', 'ordenar', 'columnas', 'unir', 'comparar', 'constante']
const BLOQUES = ['titulo', 'paragrafo', 'ficha', 'taboa', 'lista', 'grafico', 'imaxe', 'salto']
const COR = { orixe: 'var(--primary)', transformacion: 'var(--amber)', destino: 'var(--on)' }

// api: {AvaliarFluxoFormulario}; catalogo: lista de tipos de Go.
//   obterEsquema() -> o formulario (co fluxo actual dentro), obterDatos() -> {valores}.
//   aoCambiar(fluxo): o editor mudou o grafo.
export function montarFluxoEditor(container, { t: tIn, api, catalogo, obterEsquema, obterDatos = () => ({}), aoCambiar, aoAutomatico }) {
  const t = tIn || ((clave, fallback, vars) => {
    let val = fallback ?? clave
    if (vars) for (const k in vars) val = val.replaceAll(`{${k}}`, vars[k])
    return val
  })
  const idx = indexarCatalogo(catalogo)
  let fluxo = obterEsquema().fluxo
  let aval = { nodos: {}, bloques: [], avisos: [], erro: '' }
  let destruido = false
  let temporizador = null
  let menuAberto = null

  const nomeTipo = (tipo) => t('formularios.nodo.' + tipo, idx[tipo]?.nome || tipo)
  const axudaTipo = (tipo) => t('formularios.nodo.' + tipo + '.axuda', idx[tipo]?.axuda || '')
  const campoDe = (n) => obterEsquema().campos.find((c) => c.id === n.campo)
  const tipoDeSaida = (id) => aval.nodos[id]?.tipo || ''

  // rotulo: como se chama un nodo cando se mostra de onde ven unha ligazón.
  function rotulo(id) {
    const n = nodoPorId(fluxo, id)
    if (!n) return id
    if (n.tipo === 'campo') return campoDe(n)?.etiqueta || n.campo
    if (n.tipo === 'constante') return `${n.config?.valor ?? ''}`.trim() || nomeTipo('constante')
    return nomeTipo(n.tipo)
  }

  // ---------- Estado e avaliación ----------

  function cambiar(novo, { pintar = true } = {}) {
    fluxo = novo
    aoCambiar?.(fluxo)
    if (pintar) render()
    agendarAvaliar()
  }

  function agendarAvaliar() {
    clearTimeout(temporizador)
    temporizador = setTimeout(avaliar, 200)
  }

  async function avaliar() {
    if (destruido) return
    try {
      const datos = obterDatos() || {}
      const r = await api.AvaliarFluxoFormulario({
        formulario: { ...obterEsquema(), fluxo },
        datos: { formulario: obterEsquema().id || '', contexto: datos.contexto || {}, valores: datos.valores || {} },
      })
      if (destruido) return
      aval = { nodos: r.nodos || {}, bloques: r.bloques || [], avisos: r.avisos || [], erro: r.erro || '' }
    } catch (err) {
      aval = { nodos: {}, bloques: [], avisos: [], erro: String(err?.message || err) }
    }
    actualizarEstados()
  }

  // ---------- Pintado ----------

  const lenzo = el('div', { class: 'flx__lenzo' })
  const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg')
  svg.setAttribute('class', 'flx__ligazons')
  const colDatos = el('div', { class: 'flx__col flx__col--datos' })
  const colMedio = el('div', { class: 'flx__col flx__col--medio' })
  const colDoc = el('div', { class: 'flx__col flx__col--doc' })
  const barra = el('div', { class: 'flx__barra' })
  const estado = el('div', { class: 'flx__estado', role: 'status' })
  lenzo.append(svg, colDatos, colMedio, colDoc)
  container.replaceChildren(barra, estado, lenzo)
  container.classList.add('flx')

  const boton = (texto, fn, extra = {}) => el('button', { type: 'button', class: 'crono-btn crono-btn--pequeno', onclick: fn, ...extra }, texto)

  function menu(titulo, tipos) {
    const caixa = el('div', { class: 'flx__menu-wrap' })
    const b = boton(titulo + ' ▾', (e) => {
      e.stopPropagation()
      const abrir = !lista.hidden ? false : true
      cerrarMenus()
      lista.hidden = !abrir
      menuAberto = abrir ? lista : null
    }, { 'aria-haspopup': 'true' })
    const lista = el('div', { class: 'flx__menu', hidden: true, role: 'menu' })
    for (const tipo of tipos) {
      if (!idx[tipo]) continue
      lista.append(el('button', {
        type: 'button', class: 'flx__menu-item', role: 'menuitem',
        onclick: () => { cerrarMenus(); engadir(tipo) },
      }, el('strong', { text: nomeTipo(tipo) }), el('small', { text: axudaTipo(tipo) })))
    }
    caixa.append(b, lista)
    return caixa
  }
  function cerrarMenus() { container.querySelectorAll('.flx__menu').forEach((m) => { m.hidden = true }); menuAberto = null }
  const cerrarFora = () => cerrarMenus()
  document.addEventListener('click', cerrarFora)

  function engadir(tipo) {
    const { x, y } = posicionInicial(fluxo, catalogo)
    cambiar(novoNodo(fluxo, catalogo, tipo, { x, y }).fluxo)
  }

  function pintarBarra() {
    barra.replaceChildren(
      menu(t('formularios.flx.mais.transformacion', '＋ Transformación'), TRANSFORMACIONS),
      menu(t('formularios.flx.mais.bloque', '＋ Bloque do documento'), BLOQUES),
      el('span', { class: 'flx__barra-espazo' }),
      boton(t('formularios.flx.automatico', 'Volver ao cableado automático'), () => aoAutomatico?.(), {
        title: t('formularios.flx.automatico.title', 'Descarta as tuberías propias: cada campo sae segundo o seu tipo, coma ao principio'),
      }),
    )
  }

  // celaConfig crea o control dun parámetro dun nodo.
  function celaConfig(n, p) {
    const valor = n.config?.[p.clave]
    const grava = (v) => { fluxo = poñerConfig(fluxo, n.id, p.clave, v); aoCambiar?.(fluxo); agendarAvaliar() }
    const etq = t('formularios.cfg.' + n.tipo + '.' + p.clave, p.etiqueta)
    let control
    switch (p.tipo) {
      case 'escolla': {
        control = el('select', { class: 'form-entrada form-entrada--compacta', 'data-cfg': p.clave })
        for (const o of p.opcions || []) control.append(new Option(o === 'asc' ? t('formularios.cfg.asc', 'ascendente') : o === 'desc' ? t('formularios.cfg.desc', 'descendente') : t('formularios.cfg.' + n.tipo + '.' + p.clave + '.' + o, p.nomes?.[o] ?? o), o))
        control.value = String(valor ?? p.defecto ?? '')
        control.addEventListener('change', () => grava(control.value))
        break
      }
      case 'columna': {
        control = el('select', { class: 'form-entrada form-entrada--compacta', 'data-cfg': p.clave, 'data-columna': n.id })
        pintarOpcionsColumna(control, n, valor)
        control.addEventListener('change', () => grava(control.value))
        break
      }
      case 'columnas': {
        control = el('div', { class: 'flx-cols', 'data-columnas': n.id })
        pintarChecksColumnas(control, n, grava)
        break
      }
      case 'textolongo': {
        control = el('textarea', { class: 'form-entrada form-entrada--compacta', rows: '2', 'data-cfg': p.clave, spellcheck: 'true' })
        control.value = valor ?? ''
        control.addEventListener('input', () => grava(control.value))
        break
      }
      default: {
        control = el('input', {
          type: 'text', class: 'form-entrada form-entrada--compacta', 'data-cfg': p.clave, autocomplete: 'off',
          inputmode: p.tipo === 'numero' ? 'decimal' : undefined, spellcheck: p.tipo === 'texto' ? 'true' : undefined,
          placeholder: p.defecto !== undefined && p.defecto !== '' ? String(p.defecto) : undefined,
        })
        control.value = valor ?? ''
        control.addEventListener('input', () => grava(p.tipo === 'numero' && control.value.trim() !== '' && !isNaN(Number(control.value.replace(',', '.'))) ? Number(control.value.replace(',', '.')) : control.value))
      }
    }
    return el('label', { class: 'flx-cfg', title: p.axuda ? t('formularios.cfg.' + n.tipo + '.' + p.clave + '.axuda', p.axuda) : '' }, el('span', { text: etq }), control)
  }

  const columnasDaTaboa = (n) => aval.nodos[n.entradas?.taboa]?.columnas || []

  function pintarOpcionsColumna(select, n, valor) {
    const cols = columnasDaTaboa(n)
    select.replaceChildren(new Option(cols.length ? '—' : t('formularios.flx.columna.sen', '(conecta unha táboa)'), ''))
    for (const c of cols) select.append(new Option(c.etiqueta, c.id))
    select.value = cols.some((c) => c.id === valor) ? valor : ''
  }

  function pintarChecksColumnas(caixa, n, grava) {
    const cols = columnasDaTaboa(n)
    const escollidas = Array.isArray(n.config?.columnas) ? n.config.columnas : []
    caixa.replaceChildren(...(cols.length ? cols.map((c) => {
      const i = el('input', { type: 'checkbox', class: 'form-marca' })
      i.checked = escollidas.includes(c.id)
      i.addEventListener('change', () => {
        const actual = Array.isArray(n.config?.columnas) ? n.config.columnas : []
        grava(i.checked ? [...actual, c.id] : actual.filter((x) => x !== c.id))
      })
      return el('label', { class: 'flx-cols__col' }, i, el('span', { text: c.etiqueta }))
    }) : [el('small', { class: 'flx-nodo__axuda', text: t('formularios.flx.columna.sen', '(conecta unha táboa)') })]))
  }

  function pintarPorto(n, p) {
    const orixe = n.entradas?.[p.nome]
    const punto = el('i', { class: 'flx-punto flx-punto--in', 'data-porto': `${n.id}:${p.nome}`, 'data-tipo': p.tipo, 'data-destino': n.id, 'data-nome': p.nome })
    const partes = [punto]
    const conRotulo = !!idx[n.tipo]?.portosEtiqueta && p.dinamico // portos de valores con rótulo editable (ficha, gráfico)
    const etiquetaPorto = p.dinamico && !conRotulo ? p.nome : t('formularios.flx.porto.' + p.nome, p.nome === 'se' ? 'só se' : p.nome === 'taboa' ? 'táboa' : p.nome)
    if (conRotulo && orixe) {
      const i = el('input', {
        type: 'text', class: 'form-entrada form-entrada--compacta flx-porto__etq', 'aria-label': t('formularios.flx.ficha.rotulo', 'Rótulo'),
        placeholder: rotulo(orixe),
      })
      i.value = n.config?.['etq_' + p.nome] ?? ''
      i.addEventListener('input', () => { fluxo = poñerConfig(fluxo, n.id, 'etq_' + p.nome, i.value); aoCambiar?.(fluxo); agendarAvaliar() })
      partes.push(i)
    } else {
      partes.push(el('span', { class: 'flx-porto__nome', text: etiquetaPorto }))
    }
    if (orixe) {
      if (!conRotulo) partes.push(el('span', { class: 'flx-porto__orixe', title: rotulo(orixe), text: '← ' + rotulo(orixe) }))
      partes.push(el('button', {
        type: 'button', class: 'form-icona-btn flx-porto__x', title: t('formularios.flx.desconectar', 'Desconectar'),
        'aria-label': t('formularios.flx.desconectar', 'Desconectar'), html: ICONA_X, onclick: () => cambiar(desconectar(fluxo, n.id, p.nome)),
      }))
    }
    return el('div', { class: 'flx-porto' + (orixe ? ' flx-porto--ligado' : '') + (p.opcional ? ' flx-porto--opcional' : ''), 'data-porto-fila': `${n.id}:${p.nome}` }, partes)
  }

  function pintarNodo(n, { doc = false, indice = 0, total = 0 } = {}) {
    const cat = idx[n.tipo]
    if (!cat) return el('div', { class: 'flx-nodo flx-nodo--erro' }, `? ${n.tipo}`)
    const campo = n.tipo === 'campo' ? campoDe(n) : null
    const titulo = n.tipo === 'campo' ? (campo?.etiqueta || n.campo) : nomeTipo(n.tipo)
    const cab = el('div', { class: 'flx-nodo__cab' })
    if (n.tipo === 'campo') cab.append(el('span', { class: 'flx-nodo__icona', html: svgTipo(campo?.tipo || 'texto') }))
    cab.append(el('strong', { class: 'flx-nodo__titulo', title: n.tipo === 'campo' ? titulo : `${titulo} — ${axudaTipo(n.tipo)}`, text: titulo }), el('span', { class: 'flx-nodo__res', 'data-res': n.id }))
    if (doc) {
      cab.append(
        el('button', { type: 'button', class: 'form-icona-btn', title: t('formularios.dis.subir', 'Subir'), 'aria-label': t('formularios.dis.subir', 'Subir'), html: ICONA_SUBIR, disabled: indice === 0 || undefined, onclick: () => cambiar(moverNoDocumento(fluxo, n.id, -1)) }),
        el('button', { type: 'button', class: 'form-icona-btn', title: t('formularios.dis.baixar', 'Baixar'), 'aria-label': t('formularios.dis.baixar', 'Baixar'), html: ICONA_BAIXAR, disabled: indice === total - 1 || undefined, onclick: () => cambiar(moverNoDocumento(fluxo, n.id, 1)) }),
      )
    }
    if (n.tipo !== 'campo') {
      cab.append(el('button', {
        type: 'button', class: 'form-icona-btn', title: t('formularios.flx.quitar', 'Quitar o nodo'), 'aria-label': t('formularios.flx.quitar', 'Quitar o nodo'),
        html: ICONA_X, onclick: () => cambiar(eliminarNodo(fluxo, n.id)),
      }))
    }
    const nodo = el('div', { class: `flx-nodo flx-nodo--${cat.categoria}`, 'data-id': n.id }, cab)
    if (cat.categoria !== 'destino') nodo.append(el('i', { class: 'flx-punto flx-punto--out', 'data-saida': n.id, title: t('formularios.flx.saida', 'Arrastra ata unha entrada para conectar') }))
    nodo.append(el('div', { class: 'flx-nodo__erro', 'data-erro': n.id, hidden: true }))
    const corpo = el('div', { class: 'flx-nodo__corpo' })
    let portos = portosDe(n, cat)
    if (cat.portosEtiqueta) portos = [...portos.filter((p) => p.dinamico).sort((a, b) => ordePorto(a.nome) - ordePorto(b.nome)), ...portos.filter((p) => !p.dinamico)]
    for (const p of portos) corpo.append(pintarPorto(n, p))
    for (const p of cat.config || []) {
      if (n.tipo === 'campo') continue
      corpo.append(celaConfig(n, p))
    }
    if (n.tipo === 'calcular') corpo.append(el('small', { class: 'flx-nodo__axuda', text: t('formularios.flx.calcular.axuda', 'Escribe a conta cos nomes das entradas (a, b, c…). Punto decimal: 0.4') }))
    if (corpo.childNodes.length) nodo.append(corpo)
    if (!doc && n.tipo !== 'campo') {
      cab.classList.add('flx-nodo__cab--asa')
      cab.addEventListener('pointerdown', (ev) => arrastrarNodo(ev, n, nodo))
      nodo.style.left = `${n.x || 0}px`
      nodo.style.top = `${n.y || 0}px`
    }
    return nodo
  }

  function render() {
    if (destruido) return
    pintarBarra()
    const { datos, medio, documento } = nodosPorColumna(fluxo, catalogo)
    colDatos.replaceChildren(el('h4', { class: 'flx__tit', text: t('formularios.flx.datos', 'Datos') }), ...datos.map((n) => pintarNodo(n)))
    colMedio.replaceChildren(
      el('h4', { class: 'flx__tit', text: t('formularios.flx.transformacions', 'Transformacións') }),
      ...(medio.length ? [] : [el('p', { class: 'flx__axuda', text: t('formularios.flx.medio.axuda', 'Aquí van os cálculos, filtros e frases. Engade un co botón «＋ Transformación» e conecta os puntos.') })]),
      ...medio.map((n) => pintarNodo(n)),
    )
    colDoc.replaceChildren(
      el('h4', { class: 'flx__tit', text: t('formularios.flx.documento', 'Documento') }),
      ...(documento.length ? [] : [el('p', { class: 'flx__axuda', text: t('formularios.flx.doc.axuda', 'O documento está baleiro: engade bloques co botón «＋ Bloque do documento».') })]),
      ...documento.map((n, i) => pintarNodo(n, { doc: true, indice: i, total: documento.length })),
    )
    // Os nodos que chegan sen posición (un fluxo do asistente ou do disco) colócanse soos.
    autoColocar(medio)
    axustarMedio()
    actualizarEstados()
    requestAnimationFrame(pintarLigazons)
  }

  // axustarMedio fai a columna do medio tan grande coma os nodos libres (están en
  // absoluto e non a ensanchan): se non, taparían a columna do documento. Hai
  // que fixar as pistas da grella; un min-width non basta con minmax(…, 1fr).
  function axustarMedio() {
    const libres = nodosPorColumna(fluxo, catalogo).medio
    colMedio.style.minHeight = `${Math.max(320, ...libres.map((n) => (n.y || 0) + 260))}px`
    const ancho = Math.max(400, ...libres.map((n) => (n.x || 0) + 270))
    lenzo.style.gridTemplateColumns = `240px ${ancho}px 300px`
  }

  // autoColocar dá posición aos nodos do medio que non a traen: en columnas
  // segundo a súa profundidade (o que depende doutro vai á dereita del) e
  // apilados coas súas alturas reais. Son posicións por defecto: non marcan o
  // formulario coma cambiado (se o usuario move un nodo, xa entran todas).
  function autoColocar(medio) {
    const sen = medio.filter((n) => n.x === undefined && n.y === undefined)
    if (!sen.length) return
    const prof = profundidades(fluxo)
    const ordenados = sen.map((n, i) => ({ n, i })).sort((a, b) => (prof.get(a.n.id) - prof.get(b.n.id)) || (a.i - b.i))
    const yCol = {}
    let novo = fluxo
    for (const { n } of ordenados) {
      const c = Math.max(0, (prof.get(n.id) || 1) - 1)
      const elem = colMedio.querySelector(`[data-id="${CSS.escape(n.id)}"]`)
      const x = 8 + c * 268
      const y = yCol[c] ?? 8
      yCol[c] = y + (elem?.offsetHeight || 170) + 14
      novo = poñerPosicion(novo, n.id, x, y)
      if (elem) { elem.style.left = `${x}px`; elem.style.top = `${y}px` }
    }
    fluxo = novo
  }

  function actualizarEstados() {
    if (destruido) return
    for (const nodo of lenzo.querySelectorAll('.flx-nodo[data-id]')) {
      const id = nodo.dataset.id
      const info = aval.nodos[id] || {}
      const res = nodo.querySelector(`[data-res="${id}"]`)
      if (res) {
        res.textContent = info.erro ? '' : (info.resumo || (info.vacio ? t('formularios.flx.baleiro', '(baleiro)') : ''))
        res.classList.toggle('flx-nodo__res--baleiro', !info.resumo)
      }
      const er = nodo.querySelector(`[data-erro="${id}"]`)
      if (er) { er.textContent = info.erro || ''; er.hidden = !info.erro }
      nodo.classList.toggle('flx-nodo--erro', !!info.erro)
    }
    // As columnas dos nodos que dependen dunha táboa poden ter cambiado.
    for (const sel of lenzo.querySelectorAll('select[data-columna]')) {
      const n = nodoPorId(fluxo, sel.dataset.columna)
      if (n) pintarOpcionsColumna(sel, n, n.config?.[sel.dataset.cfg])
    }
    for (const caixa of lenzo.querySelectorAll('[data-columnas]')) {
      const n = nodoPorId(fluxo, caixa.dataset.columnas)
      if (n) pintarChecksColumnas(caixa, n, (v) => { fluxo = poñerConfig(fluxo, n.id, 'columnas', v); aoCambiar?.(fluxo); agendarAvaliar() })
    }
    const msgs = []
    if (aval.erro) msgs.push(el('p', { class: 'crono-erro', text: aval.erro }))
    if (aval.avisos.length) msgs.push(el('ul', { class: 'flx__avisos' }, aval.avisos.map((a) => el('li', { text: a }))))
    estado.replaceChildren(...msgs)
    requestAnimationFrame(pintarLigazons)
  }

  // ---------- Ligazóns ----------

  function centro(elem, ref) {
    const r = elem.getBoundingClientRect()
    return { x: r.left + r.width / 2 - ref.left + lenzo.scrollLeft, y: r.top + r.height / 2 - ref.top + lenzo.scrollTop }
  }
  const curva = (a, b) => {
    const d = Math.max(40, Math.abs(b.x - a.x) * 0.5)
    return `M${a.x},${a.y} C${a.x + d},${a.y} ${b.x - d},${b.y} ${b.x},${b.y}`
  }

  function pintarLigazons() {
    if (destruido) return
    const ref = lenzo.getBoundingClientRect()
    svg.setAttribute('width', String(lenzo.scrollWidth))
    svg.setAttribute('height', String(lenzo.scrollHeight))
    const camiños = []
    for (const n of fluxo.nodos) {
      for (const [porto, orixe] of Object.entries(n.entradas || {})) {
        const a = lenzo.querySelector(`[data-saida="${CSS.escape(orixe)}"]`)
        const b = lenzo.querySelector(`[data-porto="${CSS.escape(n.id + ':' + porto)}"]`)
        if (!a || !b) continue
        const cat = idx[nodoPorId(fluxo, orixe)?.tipo]?.categoria
        const p = document.createElementNS('http://www.w3.org/2000/svg', 'path')
        p.setAttribute('d', curva(centro(a, ref), centro(b, ref)))
        p.setAttribute('stroke', COR[cat] || 'var(--text-muted)')
        p.setAttribute('class', 'flx__ligazon' + (aval.nodos[n.id]?.erro ? ' flx__ligazon--erro' : ''))
        camiños.push(p)
      }
    }
    svg.replaceChildren(...camiños)
  }

  // ---------- Arrastre ----------

  function arrastrarNodo(ev, n, nodo) {
    if (ev.target.closest('button')) return
    const x0 = ev.clientX
    const y0 = ev.clientY
    const ini = { x: n.x || 0, y: n.y || 0 }
    let movido = false
    const mover = (e) => {
      const x = Math.max(0, ini.x + e.clientX - x0)
      const y = Math.max(0, ini.y + e.clientY - y0)
      nodo.style.left = `${x}px`
      nodo.style.top = `${y}px`
      movido = true
      fluxo = poñerPosicion(fluxo, n.id, x, y)
      pintarLigazons()
    }
    const soltar = () => {
      window.removeEventListener('pointermove', mover, true)
      window.removeEventListener('pointerup', soltar, true)
      if (movido) { aoCambiar?.(fluxo); axustarMedio(); pintarLigazons() }
    }
    window.addEventListener('pointermove', mover, true)
    window.addEventListener('pointerup', soltar, true)
    ev.preventDefault()
  }

  // Conectar: arrastrar desde un punto de saída ata un porto de entrada.
  lenzo.addEventListener('pointerdown', (ev) => {
    const saida = ev.target.closest?.('[data-saida]')
    if (!saida || ev.button > 0) return
    ev.preventDefault()
    ev.stopPropagation()
    const orixe = saida.dataset.saida
    const tipoSaida = tipoDeSaida(orixe)
    const ref = lenzo.getBoundingClientRect()
    const inicio = centro(saida, ref)
    const temp = document.createElementNS('http://www.w3.org/2000/svg', 'path')
    temp.setAttribute('class', 'flx__ligazon flx__ligazon--temp')
    svg.append(temp)
    const portos = [...lenzo.querySelectorAll('.flx-punto--in')]
    for (const p of portos) {
      const ok = p.dataset.destino !== orixe && aceptaTipo(p.dataset.tipo, tipoSaida)
      p.classList.add(ok ? 'flx-punto--ok' : 'flx-punto--nok')
    }
    lenzo.classList.add('flx__lenzo--conectando')
    const mover = (e) => {
      const r = lenzo.getBoundingClientRect()
      temp.setAttribute('d', curva(inicio, { x: e.clientX - r.left + lenzo.scrollLeft, y: e.clientY - r.top + lenzo.scrollTop }))
      e.preventDefault()
    }
    const rematar = (e) => {
      window.removeEventListener('pointermove', mover, true)
      window.removeEventListener('pointerup', rematar, true)
      temp.remove()
      lenzo.classList.remove('flx__lenzo--conectando')
      for (const p of portos) p.classList.remove('flx-punto--ok', 'flx-punto--nok')
      const alvo = document.elementFromPoint(e.clientX, e.clientY)?.closest?.('.flx-porto')
      const punto = alvo?.querySelector('.flx-punto--in')
      if (!punto) return
      const r = conectar(fluxo, catalogo, orixe, punto.dataset.destino, punto.dataset.nome, tipoSaida)
      if (r.erro) {
        const msgs = {
          ciclo: t('formularios.flx.erro.ciclo', 'Esa ligazón faría un ciclo (o nodo dependería de si mesmo).'),
          tipo: t('formularios.flx.erro.tipo', 'Ese dato non cadra con esa entrada (p. ex. un texto nunha conta).'),
          destino: t('formularios.flx.erro.destino', 'Un bloque do documento non ten saída.'),
          porto: t('formularios.flx.erro.porto', 'Esa entrada non existe.'),
          nodo: t('formularios.flx.erro.nodo', 'O nodo xa non existe.'),
        }
        estado.replaceChildren(el('p', { class: 'crono-erro', text: msgs[r.erro] || r.erro }))
        return
      }
      cambiar(r.fluxo)
    }
    window.addEventListener('pointermove', mover, true)
    window.addEventListener('pointerup', rematar, true)
  })

  const observador = typeof ResizeObserver === 'function' ? new ResizeObserver(() => pintarLigazons()) : null
  observador?.observe(lenzo)
  lenzo.addEventListener('scroll', pintarLigazons)

  render()
  agendarAvaliar()

  return {
    // recolle un fluxo novo de fóra (o asistente cambiou o borrador, sincronizouse...).
    setFluxo(novo) { fluxo = novo; render(); agendarAvaliar() },
    getFluxo: () => fluxo,
    reavaliar: avaliar,
    avaliacion: () => aval,
    destroy() {
      destruido = true
      clearTimeout(temporizador)
      document.removeEventListener('click', cerrarFora)
      observador?.disconnect()
      container.replaceChildren()
      container.classList.remove('flx')
    },
    // Só para probas.
    _eDestino: (id) => eDestino(idx[nodoPorId(fluxo, id)?.tipo]),
  }
}
