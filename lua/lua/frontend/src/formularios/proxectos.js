// A pantalla de INICIO: crear, abrir e ordenar proxectos. Lúa arrinca nela,
// sen ningún proxecto aberto: no panel da esquerda vai «Novo proxecto» e,
// debaixo, os proxectos creados ou metidos con Trasladar; os que non caben
// seguen no panel da dereita. Ao abrir un proxecto, os dous paneis pasan a
// ser o editor e o resultado; o botón Proxectos da columna volve aquí.
//
// Un proxecto leva a súa copia dos formularios e os seus propios datos (ver
// proxecto.go), e pode ter a súa bóveda de coñecemento (módulo Buxo). Os
// formularios dun proxecto escóllense na ventá «Deseñar formulario»
// (catalogo.js), non aquí: un proxecto novo nace baleiro. Os proxectos saen
// pregados; ao premer, desprégase o resto. Cada proxecto gárdase nun cartafol
// do equipo (por defecto, o de Lúa) e ten a cor que escolla o usuario.
//
//   crearInicio({ t, accions, esquerda, dereita }) -> { recargar(), enfocar(), repartir() }
//   esquerda, dereita: os contedores dos dous paneis (enchéos o inicio)
//   accions: {
//     listar()                 [{id, nome, materia, cor, ferramentas, activo, boveda, fontes, wiki, ruta, ubicacion}]
//     formularios()            esquemas da BIBLIOTECA (non os do proxecto aberto)
//     integradas               [{id, nome}] cronograma e actas
//     crear(nome, materia)     Promise<Proxecto> (baleiro; non o abre: despois configúrase)
//     abrir(id)                entra no proxecto (ou volve a el, se xa está aberto); Promise
//     eliminar(id)             Promise
//     fixarCor(id, cor)        '#rrggbb' ou '' (a de por defecto); Promise
//     buxo()                   Promise<{activo, instalado, iaConfigurada}>
//     abrirBuxo(id)            Promise
//     escollerCartafol()       Promise<string> selector de cartafoles do sistema ('' = cancelado)
//     mover(id, destino)       leva os ficheiros a <destino>/<Nome> ('' = o cartafol de Lúa)
//     confirmar(msg)           Promise<boolean>
//   }
import { el } from '../informes/dom.js'
import { coresFormularios } from './iconas.js'

// opcionsFerramentas: o que se pode meter nun proxecto, en orde (integradas e
// despois formularios por nome).
export function opcionsFerramentas(integradas, formularios) {
  const cores = coresFormularios(formularios)
  return [
    ...integradas.map((f) => ({ id: f.id, nome: f.nome, icona: null })),
    ...formularios.map((f) => ({ id: 'form:' + f.id, nome: f.nome, icona: f.icona, cor: cores[f.id] })),
  ]
}

// nomeCartafol: o último tramo dunha ruta (vale para / e \\).
export function nomeCartafol(ruta) {
  return String(ruta).replace(/[\\/]+$/, '').split(/[\\/]/).pop() || ruta
}

// resumoBoveda: o texto do estado da bóveda dun proxecto.
export function resumoBoveda(p, t) {
  if (!p.boveda) return t('proxectos.boveda.non', 'Sen bóveda de coñecemento')
  return t('proxectos.boveda.info', 'Bóveda: {f} fontes · {w} páxinas wiki', { f: p.fontes, w: p.wiki })
}

// Cores dos proxectos que o usuario aínda non cambiou. Cada un colle unha
// segundo o seu id, así non cambia cando nacen ou se borran outros.
export const CORES_PROXECTO = ['#3a6fd8', '#d9822b', '#8e44ad', '#1f9bb4', '#c2407a', '#c99a1c', '#4b4fc4', '#8a5a3c']

// corProxecto: a cor do proxecto (a súa ou a de por defecto), en #rrggbb.
export function corProxecto(p) {
  if (/^#[0-9a-f]{6}$/i.test(p?.cor || '')) return p.cor.toLowerCase()
  let h = 0
  for (const c of String(p?.id || '')) h = (h * 31 + c.codePointAt(0)) >>> 0
  return CORES_PROXECTO[h % CORES_PROXECTO.length]
}

// cantasCaben: cantas tarxetas (polas súas alturas, en orde) caben nun alto
// dispoñible, cun oco entre elas. As que non caben van ao panel da dereita.
export function cantasCaben(alturas, disponible, oco = 0) {
  let usado = 0
  for (let i = 0; i < alturas.length; i++) {
    usado += alturas[i] + (i ? oco : 0)
    if (usado > disponible) return i
  }
  return alturas.length
}

// motivoSenBuxo: por que non se poden engadir fontes de coñecemento agora
// mesmo e que facer ('' = pódese).
export function motivoSenBuxo(buxo, t) {
  if (!buxo?.activo) return t('proxectos.buxo.inactivo', 'As fontes de coñecemento precisan o módulo Buxo, que está desactivado. Actívao en Opcións → Módulos.')
  if (!buxo.iaConfigurada) return t('proxectos.buxo.senIA', 'As fontes de coñecemento precisan unha IA. Escolle o provedor e pon a clave en Opcións → Asistente de IA.')
  if (!buxo.instalado) {
    return buxo.erro
      ? t('proxectos.buxo.erro', 'As fontes de coñecemento (Buxo) non están instaladas: {msg}', { msg: buxo.erro })
      : t('proxectos.buxo.non', 'Buxo aínda non está instalado: Lúa descárgao ao arrincar (mira Opcións → Módulos)')
  }
  return ''
}

export function crearInicio({ t, accions, esquerda, dereita }) {
  let proxectos = []
  let carpetas = [] // as ubicacións distintas dos proxectos (rutas), por nome
  let carpetaSel = null // null = todos, '' = o cartafol de Lúa, ruta = ese cartafol
  let formularios = []
  let buxo = { activo: false, instalado: false, iaConfigurada: false }
  const abertos = new Set() // proxectos despregados

  const erro = el('p', { class: 'crono-erro bib-erro', role: 'alert', hidden: true })
  const filtro = el('nav', { class: 'prx-carpetas', 'aria-label': t('proxectos.carpetas', 'Cartafoles') })
  const listaEsq = el('div', { class: 'prx-lista' })
  const listaDer = el('div', { class: 'prx-lista', 'aria-label': t('proxectos.mais', 'Máis proxectos') })
  const mostrarErro = (m) => { erro.textContent = m || ''; erro.hidden = !m }
  async function executar(fn) {
    mostrarErro('')
    try { await fn() } catch (err) { mostrarErro(String(err?.message || err)) }
    await recargar()
  }
  async function recargar() {
    try {
      ;[proxectos, formularios, buxo] = await Promise.all([accions.listar(), accions.formularios(), accions.buxo().catch(() => buxo)])
      proxectos = proxectos || []
      carpetas = [...new Set(proxectos.map((p) => p.ubicacion || '').filter(Boolean))]
        .sort((a, b) => nomeCartafol(a).localeCompare(nomeCartafol(b)) || a.localeCompare(b))
      if (carpetaSel === '' ? !proxectos.some((p) => !p.ubicacion) : carpetaSel && !carpetas.includes(carpetaSel)) carpetaSel = null
    } catch (err) { mostrarErro(String(err?.message || err)) }
    pintar()
  }

  // ---------- Novo proxecto ----------
  // Dous pasos: «Crear» crea o proxecto (sen abrilo) e a sección pasa a
  // «Configurar»: cartafol e fontes de coñecemento; despois, «Abrir proxecto».
  let creado = null // id do proxecto acabado de crear, mentres se configura
  const inNome = el('input', { type: 'text', class: 'form-entrada', maxlength: '80', placeholder: t('proxectos.novo.nome', 'Nome do proxecto'), 'aria-label': t('proxectos.novo.nome', 'Nome do proxecto') })
  const inMateria = el('input', { type: 'text', class: 'form-entrada', maxlength: '200', placeholder: t('proxectos.novo.materia', 'Materia ou tema (opcional: axuda á IA da bóveda)'), 'aria-label': t('proxectos.novo.materia', 'Materia ou tema') })
  const crear = () => executar(async () => {
    const nome = inNome.value.trim()
    if (!nome) throw new Error(t('proxectos.novo.sen.nome', 'O proxecto necesita un nome.'))
    const p = await accions.crear(nome, inMateria.value.trim())
    if (carpetaSel && accions.mover) await accions.mover(p.id, carpetaSel)
    inNome.value = ''
    inMateria.value = ''
    creado = p.id
    abertos.add(p.id) // a súa tarxeta sae despregada: é a que se vai configurar
  })
  for (const i of [inNome, inMateria]) i.addEventListener('keydown', (e) => { if (e.key === 'Enter') { e.preventDefault(); crear() } })
  const btnCrear = el('button', { type: 'button', class: 'crono-btn primary', text: t('proxectos.novo.crear', 'Crear'), onclick: crear })
  const novo = el('section', { class: 'prx-novo' })

  function pintarNovo() {
    if (!creado) {
      novo.replaceChildren(
        el('h3', { text: t('proxectos.novo.titulo', 'Novo proxecto') }),
        el('p', { class: 'prx-axuda', text: t('proxectos.novo.axudaBaleiro', 'Crea un proxecto baleiro. Despois engádelle formularios, o cronograma ou as actas co botón + da columna (Deseñar formulario).') }),
        inNome, inMateria, btnCrear)
      return
    }
    const p = proxectos.find((x) => x.id === creado)
    if (!p) { creado = null; pintarNovo(); return }
    const fillos = [
      el('h3', { text: t('proxectos.config.titulo', 'Configurar «{nome}»', { nome: p.nome }) }),
      el('p', { class: 'prx-axuda', text: t('proxectos.config.axuda', 'O proxecto está creado. Antes de abrilo, escolle onde gardalo e engade as fontes de coñecemento coas que se formará a súa bóveda.') }),
    ]
    if (accions.mover) {
      fillos.push(el('div', { class: 'prx-config__campo' },
        el('span', { text: t('proxectos.ubicacion.gardarEn', 'Gárdase en') }),
        el('code', { class: 'prx-ruta', text: p.ruta || '', title: p.ruta || '' }),
        botonMover(p)))
    }
    fillos.push(botonFontes(p))
    fillos.push(el('div', { class: 'prx-config__botons' },
      el('button', { type: 'button', class: 'crono-btn primary', text: t('proxectos.config.abrir', 'Abrir proxecto'), onclick: () => executar(async () => { creado = null; await accions.abrir(p.id) }) }),
      el('button', { type: 'button', class: 'crono-btn', text: t('proxectos.config.despois', 'Agora non'), title: t('proxectos.config.despois.title', 'Volver sen abrir o proxecto'), onclick: () => { creado = null; pintar(); inNome.focus() } })))
    novo.replaceChildren(...fillos)
  }

  // ---------- Tarxetas ----------
  function tarxeta(p) {
    const opcions = opcionsFerramentas(accions.integradas, formularios)
    const nomes = new Map(opcions.map((o) => [o.id, o.nome]))
    const resumo = p.ferramentas.map((f) => nomes.get(f) || f).join(' · ') || t('proxectos.sen.ferramentas', 'Sen ferramentas')
    const despregado = abertos.has(p.id)
    const cor = corProxecto(p)
    const acc = [
      el('button', {
        type: 'button', class: 'crono-btn primary',
        text: p.activo ? t('proxectos.volver', 'Volver ao proxecto') : t('proxectos.abrir', 'Abrir'),
        onclick: () => executar(() => accions.abrir(p.id)),
      }),
      accions.mover ? botonMover(p) : null,
      botonFontes(p),
      accions.fixarCor ? escollerCor(p, cor) : null,
      el('button', {
        type: 'button', class: 'crono-btn prx-eliminar', text: t('proxectos.eliminar.boton', 'Eliminar'), title: p.activo ? t('proxectos.eliminar.aberto', 'Pecha o proxecto antes de eliminalo') : t('proxectos.eliminar', 'Eliminar o proxecto'),
        'aria-label': t('proxectos.eliminar.aria', 'Eliminar o proxecto {nome}', { nome: p.nome }), disabled: p.activo,
        onclick: () => executar(async () => {
          if (!await accions.confirmar(t('proxectos.eliminar.confirmar', 'Eliminar o proxecto «{nome}»? Bórranse os seus datos, as copias dos formularios e a súa bóveda. Non se pode desfacer.', { nome: p.nome }))) return
          await accions.eliminar(p.id)
        }),
      }),
    ]
    const alternar = () => { if (despregado) abertos.delete(p.id); else abertos.add(p.id); pintar() }
    const cab = el('button', {
      type: 'button', class: 'prx-tarxeta__cab', 'aria-expanded': String(despregado),
      title: despregado ? t('proxectos.pregar', 'Pregar') : t('proxectos.despregar', 'Ver as ferramentas e as accións do proxecto'),
      onclick: alternar,
    },
    el('span', { class: 'prx-tarxeta__frecha', 'aria-hidden': 'true', text: despregado ? '▾' : '▸' }),
    el('strong', { text: p.nome }),
    p.activo ? el('span', { class: 'bib-chip bib-chip--produccion', text: t('proxectos.aberto', 'Aberto') }) : null,
    el('small', { class: 'bib-form__n', text: p.creado }),
    p.materia ? el('span', { class: 'prx-materia', text: p.materia }) : null)
    return el('article', {
      'data-id': p.id, style: `--cor-proxecto: ${cor}`,
      class: 'prx-tarxeta' + (p.activo ? ' prx-tarxeta--activo' : '') + (despregado ? ' prx-tarxeta--aberta' : ''),
    },
    cab,
    despregado ? el('div', { class: 'prx-tarxeta__texto' },
      el('small', { class: 'prx-resumo', text: resumo }),
      el('small', { class: 'prx-resumo', text: resumoBoveda(p, t) }),
      p.ruta ? el('small', { class: 'prx-resumo prx-ruta', title: p.ruta, text: t('proxectos.ubicacion.en', 'Gardado en {ruta}', { ruta: p.ruta }) }) : null) : null,
    despregado ? el('div', { class: 'bib-form__acc prx-acc' }, acc.filter(Boolean)) : null)
  }

  // botonFontes: «Fontes de coñecemento» sae en todos os proxectos. Se Buxo
  // aínda non se pode usar (desactivado, sen IA ou sen instalar), premelo di
  // que falta e onde se arranxa, en vez de agochar o botón.
  function botonFontes(p) {
    const motivo = motivoSenBuxo(buxo, t)
    return el('button', {
      type: 'button', class: 'crono-btn prx-fontes' + (motivo ? ' prx-fontes--sen' : ''), text: t('proxectos.buxo', 'Fontes de coñecemento'),
      title: motivo || t('proxectos.buxo.title', 'Abre Buxo: engade PDF e constrúe a bóveda de coñecemento do proxecto'),
      onclick: () => executar(async () => {
        const agora = motivoSenBuxo(await accions.buxo().catch(() => buxo), t)
        if (agora) throw new Error(agora)
        await accions.abrirBuxo(p.id)
      }),
    })
  }

  // escollerCor: o selector de cor do sistema e, se o usuario xa a cambiou,
  // volver á de por defecto. Gárdase ao escoller (change), non mentres se arrastra.
  function escollerCor(p, cor) {
    const entrada = el('input', {
      type: 'color', class: 'prx-cor__entrada', value: cor,
      'aria-label': t('proxectos.cor.aria', 'Cor do proxecto {nome}', { nome: p.nome }),
    })
    entrada.addEventListener('change', () => executar(() => accions.fixarCor(p.id, entrada.value)))
    return el('span', { class: 'prx-cor' },
      el('label', { class: 'prx-cor__etiqueta', title: t('proxectos.cor.title', 'Escolle a cor coa que se ve este proxecto') },
        el('span', { text: t('proxectos.cor', 'Cor') }), entrada),
      p.cor ? el('button', {
        type: 'button', class: 'crono-btn prx-cor__defecto', text: t('proxectos.cor.defecto', 'Cor por defecto'),
        onclick: () => executar(() => accions.fixarCor(p.id, '')),
      }) : null)
  }

  // botonMover: escolle un cartafol do equipo e leva alí o proxecto.
  function botonMover(p) {
    return el('button', {
      type: 'button', class: 'crono-btn prx-mover', text: t('proxectos.ubicacion.escoller', 'Escoller cartafol…'),
      title: p.activo ? t('proxectos.ubicacion.aberto', 'Pecha o proxecto antes de movelo') : t('proxectos.ubicacion.title', 'Garda o proxecto noutro cartafol do equipo (os ficheiros móvense alí)'),
      'aria-label': t('proxectos.ubicacion.aria', 'Escoller onde gardar o proxecto {nome}', { nome: p.nome }), disabled: p.activo,
      onclick: () => executar(async () => {
        const destino = await accions.escollerCartafol()
        if (!destino) return
        await accions.mover(p.id, destino)
        if (carpetaSel !== null) carpetaSel = destino // a lista segue ao proxecto
      }),
    })
  }

  const daCarpeta = (p) => p.ubicacion || ''

  // pintarCarpetas: o filtro por cartafol só sae se hai proxectos gardados fóra
  // do cartafol de Lúa (senón todos estarían no mesmo).
  function pintarCarpetas() {
    filtro.hidden = carpetas.length === 0
    const n = { todos: proxectos.length, raiz: proxectos.filter((p) => daCarpeta(p) === '').length }
    const boton = (texto, valor, cantidade, title) => el('button', {
      type: 'button', class: 'bib-carp', 'aria-pressed': String(carpetaSel === valor), title,
      onclick: () => { carpetaSel = valor; pintar() },
    }, el('span', { class: 'bib-carp__nome', text: texto }), el('span', { class: 'bib-carp__n', text: String(cantidade) }))
    const filas = [boton(t('proxectos.todos', 'Todos'), null, n.todos)]
    if (n.raiz) filas.push(boton(t('proxectos.ubicacion.lua', 'Cartafol de Lúa'), '', n.raiz, t('proxectos.ubicacion.lua.title', 'Os proxectos que Lúa garda na súa configuración')))
    for (const c of carpetas) filas.push(boton(nomeCartafol(c), c, proxectos.filter((p) => daCarpeta(p) === c).length, c))
    filtro.replaceChildren(...filas)
  }

  function pintar() {
    pintarCarpetas()
    pintarNovo()
    const vistos = proxectos.filter((p) => carpetaSel === null || daCarpeta(p) === carpetaSel)
    listaDer.replaceChildren()
    listaEsq.replaceChildren(...(vistos.length
      ? vistos.map(tarxeta)
      : [el('p', { class: 'bib-baleira', text: t('proxectos.baleiro', 'Aínda non hai proxectos. Crea o primeiro aquí enriba ou mete un con Trasladar.') })]))
    repartir()
    if (creado) [...listaEsq.children, ...listaDer.children].find((e) => e.dataset.id === creado)?.scrollIntoView?.({ block: 'nearest' })
  }

  // repartir: as tarxetas que non caben no panel da esquerda (debaixo de «Novo
  // proxecto») pasan ao da dereita. Se o da dereita non se ve (resultado noutra
  // xanela) ou aínda non hai medidas, todas quedan á esquerda, con scroll.
  function repartir() {
    const tarxetas = [...listaEsq.children, ...listaDer.children].filter((e) => e.classList.contains('prx-tarxeta'))
    listaEsq.append(...tarxetas)
    if (!tarxetas.length || !esquerda.clientHeight || !dereita.clientHeight || !dereita.isConnected) return
    const px = (v) => parseFloat(v) || 0
    const estilo = getComputedStyle(listaEsq)
    const disponible = esquerda.clientHeight - px(getComputedStyle(esquerda).paddingBottom) -
      listaEsq.offsetTop - px(estilo.paddingTop) - px(estilo.paddingBottom)
    const oco = px(estilo.rowGap)
    const n = cantasCaben(tarxetas.map((e) => e.offsetHeight), disponible, oco)
    listaDer.append(...tarxetas.slice(n))
  }
  let pendente = 0
  const repartirLogo = () => { cancelAnimationFrame(pendente); pendente = requestAnimationFrame(repartir) }
  if (typeof ResizeObserver === 'function') {
    const ro = new ResizeObserver(repartirLogo)
    ro.observe(esquerda)
    ro.observe(dereita)
  }

  esquerda.replaceChildren(
    el('h2', { class: 'prx-inicio__titulo', text: t('proxectos.titulo', 'Proxectos') }),
    novo, erro, filtro, listaEsq)
  dereita.replaceChildren(listaDer)

  return {
    recargar,
    repartir,
    enfocar: () => { if (!creado) inNome.focus() },
  }
}
