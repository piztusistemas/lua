// O panel de PROXECTOS: crear, abrir e ordenar proxectos. Un proxecto leva a
// súa copia dos formularios e os seus propios datos (ver proxecto.go), e pode
// ter a súa bóveda de coñecemento (módulo Buxo). Os formularios dun proxecto
// escóllense na ventá «Deseñar formulario» (catalogo.js), non aquí: un proxecto
// novo nace baleiro. Os proxectos saen pregados; ao premer, desprégase o resto.
//
//   abrirProxectos({ t, accions }) -> Promise (resolve ao pechar)
//   accions: {
//     listar()                 [{id, nome, materia, ferramentas, activo, boveda, fontes, wiki}]
//     formularios()            esquemas da BIBLIOTECA (non os do proxecto aberto)
//     integradas               [{id, nome}] cronograma e actas
//     crear(nome, materia)     Promise<Proxecto> (baleiro)
//     abrir(id)                entra no proxecto (main.js recarga as ferramentas); Promise
//     eliminar(id)             Promise
//     buxo()                   Promise<{activo, instalado, iaConfigurada}>
//     abrirBuxo(id)            Promise
//     carpetas()               [{id, nome}]
//     crearCarpeta(nome) / renomearCarpeta(id, nome) / eliminarCarpeta(id)   Promise (eliminar NON borra proxectos)
//     mover(id, carpeta)       pon o proxecto nun cartafol ('' = sen cartafol)
//     preguntar(msg, valor)    Promise<string|null>
//     confirmar(msg)           Promise<boolean>
//   }
import { el } from '../informes/dom.js'
import { coresFormularios } from './iconas.js'

const ICONA_MAIS = '<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"><path d="M10 4.5v11M4.5 10h11"/></svg>'
const ICONA_LAPIS = '<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M4 16 14.5 5.5a1.4 1.4 0 0 1 2 2L6 18l-3 1z"/></svg>'
const ICONA_LIXO = '<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M4.5 5.5h11"/><path d="M8 5.5V4a1 1 0 0 1 1-1h2a1 1 0 0 1 1 1v1.5"/><path d="M5.5 5.5 6.2 16a1 1 0 0 0 1 .9h5.6a1 1 0 0 0 1-.9l.7-10.5"/></svg>'
const ICONA_X = '<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M5 5l10 10M15 5L5 15"/></svg>'

// opcionsFerramentas: o que se pode meter nun proxecto, en orde (integradas e
// despois formularios por nome).
export function opcionsFerramentas(integradas, formularios) {
  const cores = coresFormularios(formularios)
  return [
    ...integradas.map((f) => ({ id: f.id, nome: f.nome, icona: null })),
    ...formularios.map((f) => ({ id: 'form:' + f.id, nome: f.nome, icona: f.icona, cor: cores[f.id] })),
  ]
}

// resumoBoveda: o texto do estado da bóveda dun proxecto.
export function resumoBoveda(p, t) {
  if (!p.boveda) return t('proxectos.boveda.non', 'Sen bóveda de coñecemento')
  return t('proxectos.boveda.info', 'Bóveda: {f} fontes · {w} páxinas wiki', { f: p.fontes, w: p.wiki })
}

export function abrirProxectos({ t, accions }) {
  return new Promise((resolve) => {
    const previo = document.activeElement
    let proxectos = []
    let carpetas = []
    let carpetaSel = null // null = todos, '' = sen cartafol, id = ese cartafol
    let formularios = []
    let buxo = { activo: false, instalado: false, iaConfigurada: false }
    const abertos = new Set() // proxectos despregados

    const overlay = el('div', { class: 'overlay bib-overlay', role: 'dialog', 'aria-modal': 'true', 'aria-label': t('proxectos.titulo', 'Proxectos') })
    const erro = el('p', { class: 'crono-erro bib-erro', role: 'alert', hidden: true })
    const lista = el('div', { class: 'prx-lista' })
    const lateral = el('nav', { class: 'bib-carpetas prx-carpetas', 'aria-label': t('proxectos.carpetas', 'Cartafoles') })
    const mostrarErro = (m) => { erro.textContent = m || ''; erro.hidden = !m }
    async function executar(fn) {
      mostrarErro('')
      try { await fn() } catch (err) { mostrarErro(String(err?.message || err)) }
      await recargar()
    }
    async function recargar() {
      try {
        ;[proxectos, formularios, buxo, carpetas] = await Promise.all([accions.listar(), accions.formularios(), accions.buxo().catch(() => buxo), accions.carpetas ? accions.carpetas() : []])
        carpetas = carpetas || []
        if (carpetaSel && !carpetas.some((c) => c.id === carpetaSel)) carpetaSel = null
      } catch (err) { mostrarErro(String(err?.message || err)) }
      pintar()
    }

    // ---------- Novo proxecto ----------
    const inNome = el('input', { type: 'text', class: 'form-entrada', maxlength: '80', placeholder: t('proxectos.novo.nome', 'Nome do proxecto'), 'aria-label': t('proxectos.novo.nome', 'Nome do proxecto') })
    const inMateria = el('input', { type: 'text', class: 'form-entrada', maxlength: '200', placeholder: t('proxectos.novo.materia', 'Materia ou tema (opcional: axuda á IA da bóveda)'), 'aria-label': t('proxectos.novo.materia', 'Materia ou tema') })
    const btnCrear = el('button', {
      type: 'button', class: 'crono-btn primary', text: t('proxectos.novo.crear', 'Crear e abrir'),
      onclick: () => executar(async () => {
        const nome = inNome.value.trim()
        if (!nome) throw new Error(t('proxectos.novo.sen.nome', 'O proxecto necesita un nome.'))
        const p = await accions.crear(nome, inMateria.value.trim())
        if (carpetaSel && accions.mover) await accions.mover(p.id, carpetaSel)
        inNome.value = ''
        inMateria.value = ''
        await accions.abrir(p.id)
        pechar()
      }),
    })
    const novo = el('section', { class: 'prx-novo' },
      el('h3', { text: t('proxectos.novo.titulo', 'Novo proxecto') }),
      el('p', { class: 'prx-axuda', text: t('proxectos.novo.axudaBaleiro', 'Crea un proxecto baleiro. Despois engádelle formularios, o cronograma ou as actas co botón + da columna (Deseñar formulario).') }),
      inNome, inMateria, btnCrear)

    // ---------- Lista ----------
    function tarxeta(p) {
      const opcions = opcionsFerramentas(accions.integradas, formularios)
      const nomes = new Map(opcions.map((o) => [o.id, o.nome]))
      const resumo = p.ferramentas.map((f) => nomes.get(f) || f).join(' · ') || t('proxectos.sen.ferramentas', 'Sen ferramentas')
      const buxoOfrecido = buxo.activo && buxo.iaConfigurada
      const despregado = abertos.has(p.id)
      const acc = [
        p.activo
          ? el('span', { class: 'bib-chip', text: t('proxectos.aberto', 'Aberto') })
          : el('button', { type: 'button', class: 'crono-btn primary', text: t('proxectos.abrir', 'Abrir'), onclick: () => executar(async () => { await accions.abrir(p.id); pechar() }) }),
        // Buxo ofrécese se está activo e hai unha IA configurada; se aínda non
        // está instalado (descárgase das Releases) o botón sae apagado e di por que.
        buxoOfrecido ? el('button', {
          type: 'button', class: 'crono-btn', text: t('proxectos.buxo', 'Fontes de coñecemento'), disabled: !buxo.instalado,
          title: buxo.instalado
            ? t('proxectos.buxo.title', 'Abre Buxo: engade PDF e constrúe a bóveda de coñecemento do proxecto')
            : buxo.erro ? t('proxectos.buxo.erro', 'Buxo non está instalado: {msg}', { msg: buxo.erro }) : t('proxectos.buxo.non', 'Buxo aínda non está instalado: Lúa descárgao ao arrincar (mira Opcións → Módulos)'),
          onclick: () => executar(() => accions.abrirBuxo(p.id)),
        }) : null,
        el('button', {
          type: 'button', class: 'crono-btn prx-eliminar', text: t('proxectos.eliminar.boton', 'Eliminar'), title: p.activo ? t('proxectos.eliminar.aberto', 'Pecha o proxecto antes de eliminalo') : t('proxectos.eliminar', 'Eliminar o proxecto'),
          'aria-label': t('proxectos.eliminar.aria', 'Eliminar o proxecto {nome}', { nome: p.nome }), disabled: p.activo,
          onclick: () => executar(async () => {
            if (!await accions.confirmar(t('proxectos.eliminar.confirmar', 'Eliminar o proxecto «{nome}»? Bórranse os seus datos, as copias dos formularios e a súa bóveda. Non se pode desfacer.', { nome: p.nome }))) return
            await accions.eliminar(p.id)
          }),
        }),
      ]
      const selCarpeta = accions.mover ? el('select', {
        class: 'form-entrada form-entrada--compacta prx-carpeta', 'aria-label': t('proxectos.carpeta.aria', 'Cartafol de {nome}', { nome: p.nome }),
        title: t('proxectos.carpeta.title', 'Cartafol do proxecto'),
      }, new Option(t('proxectos.senCartafol', 'Sen cartafol'), ''), ...carpetas.map((c) => new Option(c.nome, c.id))) : null
      if (selCarpeta) {
        selCarpeta.value = carpetas.some((c) => c.id === p.carpeta) ? p.carpeta : ''
        selCarpeta.addEventListener('change', () => executar(() => accions.mover(p.id, selCarpeta.value)))
      }
      acc.unshift(selCarpeta)
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
      return el('article', { class: 'prx-tarxeta' + (p.activo ? ' prx-tarxeta--activo' : '') + (despregado ? ' prx-tarxeta--aberta' : '') },
        cab,
        despregado ? el('div', { class: 'prx-tarxeta__texto' },
          el('small', { class: 'prx-resumo', text: resumo }),
          el('small', { class: 'prx-resumo', text: resumoBoveda(p, t) })) : null,
        despregado ? el('div', { class: 'bib-form__acc prx-acc' }, acc.filter(Boolean)) : null)
    }

    const daCarpeta = (p) => (carpetas.some((c) => c.id === p.carpeta) ? p.carpeta : '')

    function pintarCarpetas() {
      const n = { todos: proxectos.length, raiz: proxectos.filter((p) => daCarpeta(p) === '').length }
      for (const c of carpetas) n[c.id] = proxectos.filter((p) => daCarpeta(p) === c.id).length
      const boton = (texto, valor, cantidade) => el('button', {
        type: 'button', class: 'bib-carp', 'aria-pressed': String(carpetaSel === valor),
        onclick: () => { carpetaSel = valor; pintar() },
      }, el('span', { class: 'bib-carp__nome', text: texto }), el('span', { class: 'bib-carp__n', text: String(cantidade) }))
      const filas = [boton(t('proxectos.todos', 'Todos'), null, n.todos), boton(t('proxectos.senCartafol', 'Sen cartafol'), '', n.raiz)]
      for (const c of carpetas) {
        filas.push(el('div', { class: 'bib-carp-fila' },
          boton(c.nome, c.id, n[c.id] ?? 0),
          el('button', {
            type: 'button', class: 'form-icona-btn', html: ICONA_LAPIS, title: t('proxectos.carpeta.renomear', 'Cambiar o nome do cartafol'),
            'aria-label': t('proxectos.carpeta.renomear.aria', 'Cambiar o nome do cartafol {nome}', { nome: c.nome }),
            onclick: () => renomearCarp(c),
          }),
          el('button', {
            type: 'button', class: 'form-icona-btn', html: ICONA_LIXO, title: t('proxectos.carpeta.eliminar', 'Eliminar o cartafol (os proxectos non se borran)'),
            'aria-label': t('proxectos.carpeta.eliminar.aria', 'Eliminar o cartafol {nome}', { nome: c.nome }),
            onclick: () => eliminarCarp(c, n[c.id] ?? 0),
          })))
      }
      if (accions.crearCarpeta) {
        filas.push(el('button', { type: 'button', class: 'bib-nova', html: ICONA_MAIS + `<span>${t('proxectos.carpeta.nova', 'Novo cartafol')}</span>`, onclick: () => novaCarp() }))
      }
      lateral.replaceChildren(...filas)
    }

    async function novaCarp() {
      const nome = (await accions.preguntar(t('proxectos.carpeta.nova.pregunta', 'Nome do cartafol novo (p. ex. unha temática):'), ''))?.trim()
      if (!nome) return
      await executar(async () => { await accions.crearCarpeta(nome) })
    }
    async function renomearCarp(c) {
      const nome = (await accions.preguntar(t('proxectos.carpeta.renomear.pregunta', 'Novo nome do cartafol:'), c.nome))?.trim()
      if (!nome || nome === c.nome) return
      await executar(async () => { await accions.renomearCarpeta(c.id, nome) })
    }
    async function eliminarCarp(c, cantidade) {
      const msg = cantidade
        ? t('proxectos.carpeta.eliminar.confirmar', 'Eliminar o cartafol «{nome}»? Os {n} proxectos que ten NON se borran: pasan a «Sen cartafol».', { nome: c.nome, n: cantidade })
        : t('proxectos.carpeta.eliminar.confirmarBaleiro', 'Eliminar o cartafol «{nome}»?', { nome: c.nome })
      if (!await accions.confirmar(msg)) return
      if (carpetaSel === c.id) carpetaSel = null
      await executar(async () => { await accions.eliminarCarpeta(c.id) })
    }

    function pintar() {
      pintarCarpetas()
      const vistos = proxectos.filter((p) => carpetaSel === null || daCarpeta(p) === carpetaSel)
      lista.replaceChildren(...(vistos.length
        ? vistos.map(tarxeta)
        : [el('p', { class: 'bib-baleira', text: proxectos.length ? t('proxectos.carpeta.baleira', 'Este cartafol está baleiro. Move proxectos aquí co selector de cada un.') : t('proxectos.baleiro', 'Aínda non hai proxectos. Crea o primeiro co formulario da dereita.') })]))
    }

    const caixa = el('div', { class: 'bib prx' },
      el('header', { class: 'bib__cab' },
        el('h2', { text: t('proxectos.titulo', 'Proxectos') }),
        el('button', { type: 'button', class: 'dis-x', 'aria-label': t('proxectos.pechar.panel', 'Pechar'), title: t('proxectos.pechar.panel', 'Pechar'), html: ICONA_X, onclick: () => pechar() })),
      erro,
      el('div', { class: 'prx-corpo' }, lateral, lista, novo))
    overlay.append(caixa)

    function tecla(e) {
      if (e.key !== 'Escape') return
      if (document.querySelector('.menu-ctx, .overlay--dialog')) return
      e.stopPropagation()
      pechar()
    }
    function pechar() {
      document.removeEventListener('keydown', tecla, true)
      overlay.remove()
      if (previo && previo.isConnected) previo.focus?.()
      resolve()
    }
    document.addEventListener('keydown', tecla, true)
    document.body.append(overlay)
    recargar()
    inNome.focus()
  })
}
