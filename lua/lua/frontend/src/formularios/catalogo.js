// A ventá «DESEÑAR FORMULARIO» (spec reutilizar-formularios): o catálogo común
// de todos os formularios creados (e o cronograma e as actas), en cartafoles.
// Márcanse os que se queren e «Reutilizar» méteos no proxecto aberto, cada un
// coa súa copia. Tamén abre o deseñador para un formulario novo. É o único sitio
// onde se escollen as ferramentas dun proxecto.
//
//   abrirCatalogo({ t, accions }) -> Promise (resolve ao pechar)
//   accions: {
//     catalogo()                 Promise<{integradas, formularios, carpetas, noProxecto}>
//     nomeIntegrada(id)          nome do cronograma / das actas
//     reutilizar(ferramentas)    Promise (mete no proxecto aberto)
//     novo()                     abre o deseñador para un formulario novo
//     crearCarpeta(nome) / renomearCarpeta(id, nome) / eliminarCarpeta(id)   Promise
//     mover(idFormulario, carpeta)   Promise ('' = sen cartafol)
//     preguntar(msg, valor) / confirmar(msg)
//   }
import { el } from '../informes/dom.js'
import { svgIcona, coresFormularios, valorCor } from './iconas.js'

const ICONA_MAIS = '<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"><path d="M10 4.5v11M4.5 10h11"/></svg>'
const ICONA_LAPIS = '<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M4 16 14.5 5.5a1.4 1.4 0 0 1 2 2L6 18l-3 1z"/></svg>'
const ICONA_LIXO = '<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M4.5 5.5h11"/><path d="M8 5.5V4a1 1 0 0 1 1-1h2a1 1 0 0 1 1 1v1.5"/><path d="M5.5 5.5 6.2 16a1 1 0 0 0 1 .9h5.6a1 1 0 0 0 1-.9l.7-10.5"/></svg>'
const ICONA_X = '<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M5 5l10 10M15 5L5 15"/></svg>'

// filtrarCatalogo: os formularios do cartafol escollido (null = todos, '' =
// sen cartafol) que casan coa busca (por nome ou descrición, sen maiúsculas).
export function filtrarCatalogo(formularios, carpeta, busca) {
  const q = String(busca || '').trim().toLowerCase()
  return formularios.filter((f) => (carpeta === null || (f.carpeta || '') === carpeta)
    && (!q || f.nome.toLowerCase().includes(q) || (f.descricion || '').toLowerCase().includes(q)))
}

export function abrirCatalogo({ t, accions }) {
  return new Promise((resolve) => {
    const previo = document.activeElement
    let cat = { integradas: [], formularios: [], carpetas: [], noProxecto: [] }
    let carpetaSel = null
    let busca = ''
    const marcados = new Set()

    const erro = el('p', { class: 'crono-erro bib-erro', role: 'alert', hidden: true })
    const mostrarErro = (m) => { erro.textContent = m || ''; erro.hidden = !m }
    async function executar(fn) {
      mostrarErro('')
      try { await fn() } catch (err) { mostrarErro(String(err?.message || err)) }
    }
    async function recargar() {
      try {
        cat = await accions.catalogo()
        if (carpetaSel && !cat.carpetas.some((c) => c.id === carpetaSel)) carpetaSel = null
      } catch (err) { mostrarErro(String(err?.message || err)) }
      pintar()
    }

    const lateral = el('nav', { class: 'bib-carpetas', 'aria-label': t('catalogo.carpetas', 'Cartafoles') })
    const lista = el('div', { class: 'bib-lista', role: 'list' })
    const btnReutilizar = el('button', { type: 'button', class: 'crono-btn primary', onclick: () => executar(reutilizar) })
    const inBusca = el('input', {
      type: 'search', class: 'form-entrada bib-busca', placeholder: t('catalogo.buscar', 'Buscar formularios'), 'aria-label': t('catalogo.buscar', 'Buscar formularios'),
      oninput: () => { busca = inBusca.value; pintarLista() },
    })

    const noProxecto = (id) => cat.noProxecto.includes(id)

    function fila({ id, nome, descricion, icona, cor, carpeta, formulario }) {
      const xa = noProxecto(id)
      const cb = el('input', { type: 'checkbox', checked: xa || marcados.has(id), disabled: xa, 'aria-label': t('catalogo.marcar', 'Marcar «{nome}»', { nome }) })
      cb.addEventListener('change', () => { if (cb.checked) marcados.add(id); else marcados.delete(id); actualizarBoton() })
      const sel = formulario ? el('select', {
        class: 'form-entrada form-entrada--compacta bib-sel', 'aria-label': t('catalogo.carpetaDe', 'Cartafol de «{nome}»', { nome }),
        onchange: (e) => executar(async () => { await accions.mover(formulario, e.target.value); await recargar() }),
      }, new Option(t('catalogo.senCartafol', 'Sen cartafol'), ''), ...cat.carpetas.map((c) => new Option(c.nome, c.id))) : null
      if (sel) sel.value = carpeta || ''
      return el('div', { class: 'bib-form' + (xa ? ' cat-form--xa' : ''), role: 'listitem' },
        el('label', { class: 'cat-marca' }, cb,
          el('span', { class: 'bib-form__icona', style: `--cor-formulario: ${valorCor(cor)}`, html: svgIcona(icona || (formulario ? 'formulario' : id === 'actas' ? 'persoas' : 'calendario')), 'aria-hidden': 'true' }),
          el('span', { class: 'bib-form__texto' },
            el('span', { class: 'bib-form__tit' }, el('strong', { text: nome }),
              xa ? el('span', { class: 'bib-chip', text: t('catalogo.xaNoProxecto', 'Xa no proxecto') }) : null),
            descricion ? el('small', { class: 'prx-resumo', text: descricion }) : null)),
        sel ? el('div', { class: 'bib-form__acc' }, sel) : null)
    }

    function pintarLista() {
      const cores = coresFormularios(cat.formularios)
      const filas = []
      if (carpetaSel === null && !busca.trim()) {
        filas.push(el('h3', { class: 'cat-seccion', text: t('catalogo.integradas', 'Do programa') }))
        for (const id of cat.integradas) filas.push(fila({ id, nome: accions.nomeIntegrada(id) }))
        filas.push(el('h3', { class: 'cat-seccion', text: t('catalogo.formularios', 'Formularios creados') }))
      }
      const vistos = filtrarCatalogo(cat.formularios, carpetaSel, busca)
      for (const f of vistos) filas.push(fila({ id: 'form:' + f.id, nome: f.nome, descricion: f.descricion, icona: f.icona, cor: cores[f.id], carpeta: f.carpeta, formulario: f.id }))
      if (!vistos.length) filas.push(el('p', { class: 'bib-baleira', text: cat.formularios.length ? t('catalogo.ningun', 'Ningún formulario aquí.') : t('catalogo.baleiro', 'Aínda non hai formularios creados. Crea o primeiro con «Novo formulario».') }))
      lista.replaceChildren(...filas)
      actualizarBoton()
    }

    function actualizarBoton() {
      const n = [...marcados].filter((id) => !noProxecto(id)).length
      btnReutilizar.textContent = n ? t('catalogo.reutilizarN', 'Reutilizar ({n})', { n }) : t('catalogo.reutilizar', 'Reutilizar')
      btnReutilizar.disabled = n === 0
    }

    function pintarCarpetas() {
      const contar = (v) => cat.formularios.filter((f) => v === null || (f.carpeta || '') === v).length
      const boton = (texto, valor) => el('button', {
        type: 'button', class: 'bib-carp', 'aria-pressed': String(carpetaSel === valor),
        onclick: () => { carpetaSel = valor; pintar() },
      }, el('span', { class: 'bib-carp__nome', text: texto }), el('span', { class: 'bib-carp__n', text: String(contar(valor)) }))
      const filas = [boton(t('catalogo.todos', 'Todos'), null), boton(t('catalogo.senCartafol', 'Sen cartafol'), '')]
      for (const c of cat.carpetas) {
        filas.push(el('div', { class: 'bib-carp-fila' }, boton(c.nome, c.id),
          el('button', { type: 'button', class: 'form-icona-btn', html: ICONA_LAPIS, title: t('catalogo.carpeta.renomear', 'Cambiar o nome do cartafol'), 'aria-label': t('catalogo.carpeta.renomear.aria', 'Cambiar o nome do cartafol {nome}', { nome: c.nome }), onclick: () => renomear(c) }),
          el('button', { type: 'button', class: 'form-icona-btn', html: ICONA_LIXO, title: t('catalogo.carpeta.eliminar', 'Eliminar o cartafol (os formularios non se borran)'), 'aria-label': t('catalogo.carpeta.eliminar.aria', 'Eliminar o cartafol {nome}', { nome: c.nome }), onclick: () => eliminar(c) })))
      }
      filas.push(el('button', { type: 'button', class: 'bib-nova', html: ICONA_MAIS + `<span>${t('catalogo.carpeta.nova', 'Novo cartafol')}</span>`, onclick: () => nova() }))
      lateral.replaceChildren(...filas)
    }

    function pintar() {
      pintarCarpetas()
      pintarLista()
    }

    async function nova() {
      const nome = (await accions.preguntar(t('catalogo.carpeta.nova.pregunta', 'Nome do cartafol novo (p. ex. unha materia ou un tema):'), ''))?.trim()
      if (!nome) return
      await executar(async () => { await accions.crearCarpeta(nome); await recargar() })
    }
    async function renomear(c) {
      const nome = (await accions.preguntar(t('catalogo.carpeta.renomear.pregunta', 'Novo nome do cartafol:'), c.nome))?.trim()
      if (!nome || nome === c.nome) return
      await executar(async () => { await accions.renomearCarpeta(c.id, nome); await recargar() })
    }
    async function eliminar(c) {
      const n = cat.formularios.filter((f) => f.carpeta === c.id).length
      const msg = n
        ? t('catalogo.carpeta.eliminar.confirmar', 'Eliminar o cartafol «{nome}»? Os {n} formularios que ten NON se borran: pasan a «Sen cartafol».', { nome: c.nome, n })
        : t('catalogo.carpeta.eliminar.confirmarBaleiro', 'Eliminar o cartafol «{nome}»?', { nome: c.nome })
      if (!await accions.confirmar(msg)) return
      if (carpetaSel === c.id) carpetaSel = null
      await executar(async () => { await accions.eliminarCarpeta(c.id); await recargar() })
    }

    async function reutilizar() {
      const ids = [...marcados].filter((id) => !noProxecto(id))
      if (!ids.length) return
      await accions.reutilizar(ids)
      marcados.clear()
      pechar()
    }

    const caixa = el('div', { class: 'bib cat' },
      el('header', { class: 'bib__cab' },
        el('h2', { text: t('catalogo.titulo', 'Formularios') }),
        inBusca,
        el('button', { type: 'button', class: 'crono-btn', html: ICONA_MAIS + `<span>${t('catalogo.novo', 'Novo formulario')}</span>`, onclick: () => { pechar(); accions.novo() } }),
        el('button', { type: 'button', class: 'dis-x', 'aria-label': t('proxectos.pechar.panel', 'Pechar'), title: t('proxectos.pechar.panel', 'Pechar'), html: ICONA_X, onclick: () => pechar() })),
      erro,
      el('div', { class: 'bib__corpo' }, lateral, lista),
      el('footer', { class: 'bib-pe cat-pe' },
        el('span', { class: 'bib-pe__axuda', text: t('catalogo.axuda', 'Marca o que queres usar neste proxecto e preme «Reutilizar»: o proxecto leva a súa copia. Quitar unha ferramenta do proxecto: botón dereito sobre ela na columna.') }),
        btnReutilizar))
    const overlay = el('div', { class: 'overlay bib-overlay', role: 'dialog', 'aria-modal': 'true', 'aria-label': t('catalogo.titulo', 'Formularios') }, caixa)

    function tecla(e) {
      if (e.key !== 'Escape') return
      if (document.querySelector('.overlay--dialog')) return
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
    inBusca.focus()
  })
}
