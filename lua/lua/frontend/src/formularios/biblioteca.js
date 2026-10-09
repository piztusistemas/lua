// A BIBLIOTECA de formularios: todos os formularios do usuario (tamén os que
// non se amosan na columna esquerda), organizados en cartafoles, con nome,
// descrición e modo. Desde aquí tamén se abren, se fixan na zona superior da
// columna, se agochan e se borran (as mesmas accións ca co botón dereito da
// icona, ver accions-ferramenta.js).
//
//   abrirBibliotecaFormularios({ t, accions }) -> Promise (resolve ao pechar)
//   accions: {
//     formularios()          esquemas actuais, por nome
//     biblioteca()           estado (biblioteca-modelo.js)
//     integradas             [{id, nome}] cronograma e actas (só se poden agochar)
//     gardar(nova)           persiste o novo estado (lanza se Go o rexeita)
//     crearCarpeta(nome)     Promise
//     gardarDescricion(id, texto)
//     abrir(id)              amosa a ferramenta (se estaba agochada) e vai a ela
//     eliminar(id)           borra un formulario (con confirmación); Promise<boolean>
//     exportar(id)           exporta un formulario a un ficheiro (para pasarllo a un compañeiro)
//     importar()             importa un formulario desde un ficheiro; Promise<{avisos}|null> (null se se cancela; os avisos amósanse aquí mesmo)
//     confirmar(msg), preguntar(msg, valor)   diálogos propios
//   }
import { el } from '../informes/dom.js'
import { svgIcona } from './iconas.js'
import {
  filtrar, contarPorCarpeta, estaFixa, estaAgochada, moverACarpeta, renomearCarpeta, eliminarCarpeta, nomeRepetido, agochar,
} from './biblioteca-modelo.js'
import { itemsDeFerramenta } from './accions-ferramenta.js'
import { abrirMenu } from './menu-contextual.js'

const ICONA_X = '<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M5 5l10 10M15 5L5 15"/></svg>'
const ICONA_MAIS = '<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"><path d="M10 4.5v11M4.5 10h11"/></svg>'
const ICONA_LAPIS = '<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M4 16 14.5 5.5a1.4 1.4 0 0 1 2 2L6 18l-3 1z"/></svg>'
const ICONA_LIXO = '<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M4.5 5.5h11"/><path d="M8 5.5V4a1 1 0 0 1 1-1h2a1 1 0 0 1 1 1v1.5"/><path d="M5.5 5.5 6.2 16a1 1 0 0 0 1 .9h5.6a1 1 0 0 0 1-.9l.7-10.5"/></svg>'
const ICONA_MAIS_OPC = '<svg viewBox="0 0 20 20" fill="currentColor"><circle cx="4.5" cy="10" r="1.5"/><circle cx="10" cy="10" r="1.5"/><circle cx="15.5" cy="10" r="1.5"/></svg>'

export function abrirBibliotecaFormularios({ t, accions }) {
  return new Promise((resolve) => {
    const previo = document.activeElement
    let carpetaSel = null // null = todos, '' = sen cartafol, id = ese cartafol
    let busca = ''

    const overlay = el('div', { class: 'overlay bib-overlay', role: 'dialog', 'aria-modal': 'true', 'aria-label': t('formularios.bib.titulo', 'Biblioteca de formularios') })
    const erro = el('p', { class: 'crono-erro bib-erro', role: 'alert', hidden: true })
    const aviso = el('p', { class: 'bib-aviso', role: 'status', hidden: true })
    const lateral = el('nav', { class: 'bib-carpetas', 'aria-label': t('formularios.bib.carpetas', 'Cartafoles') })
    const lista = el('div', { class: 'bib-lista' })
    const pe = el('div', { class: 'bib-pe' })
    const inBusca = el('input', { type: 'search', class: 'form-entrada bib-busca', placeholder: t('formularios.bib.buscar', 'Buscar por nome ou descrición…'), 'aria-label': t('formularios.bib.buscar.aria', 'Buscar formularios') })
    inBusca.addEventListener('input', () => { busca = inBusca.value; pintarLista() })

    const mostrarErro = (m) => { erro.textContent = m || ''; erro.hidden = !m }
    const mostrarAviso = (m) => { aviso.textContent = m || ''; aviso.hidden = !m }
    // executar corre unha acción, repinta e amosa o erro se Go a rexeita.
    async function executar(fn, foco) {
      mostrarErro('')
      try { await fn() } catch (err) { mostrarErro(String(err?.message || err)) }
      pintar(foco)
    }
    const itemsDe = (id) => itemsDeFerramenta(id, {
      t, biblioteca: accions.biblioteca,
      gardar: (nova) => executar(() => accions.gardar(nova)),
      eliminar: (fid) => executar(() => accions.eliminar(fid)),
      exportar: accions.exportar ? (fid) => executar(() => accions.exportar(fid)) : undefined,
    })

    // ---------- Cartafoles ----------
    function pintarCarpetas() {
      const b = accions.biblioteca()
      const n = contarPorCarpeta(b, accions.formularios())
      const boton = (texto, valor, cantidade, foco) => el('button', {
        type: 'button', class: 'bib-carp', 'aria-pressed': String(carpetaSel === valor), 'data-foco': foco,
        onclick: () => { carpetaSel = valor; pintar(foco) },
      }, el('span', { class: 'bib-carp__nome', text: texto }), el('span', { class: 'bib-carp__n', text: String(cantidade) }))
      const filas = [
        boton(t('formularios.bib.todos', 'Todos'), null, n.todos, 'c:todos'),
        boton(t('formularios.bib.senCartafol', 'Sen cartafol'), '', n.raiz, 'c:raiz'),
      ]
      for (const c of b.carpetas) {
        filas.push(el('div', { class: 'bib-carp-fila' },
          boton(c.nome, c.id, n[c.id] ?? 0, 'c:' + c.id),
          el('button', {
            type: 'button', class: 'form-icona-btn', html: ICONA_LAPIS, 'data-foco': 'r:' + c.id,
            title: t('formularios.bib.renomear', 'Cambiar o nome do cartafol'), 'aria-label': t('formularios.bib.renomear.aria', 'Cambiar o nome do cartafol {nome}', { nome: c.nome }),
            onclick: () => renomear(c),
          }),
          el('button', {
            type: 'button', class: 'form-icona-btn', html: ICONA_LIXO, 'data-foco': 'e:' + c.id,
            title: t('formularios.bib.eliminarCarpeta', 'Eliminar o cartafol (os formularios non se borran)'), 'aria-label': t('formularios.bib.eliminarCarpeta.aria', 'Eliminar o cartafol {nome}', { nome: c.nome }),
            onclick: () => eliminarCarp(c, n[c.id] ?? 0),
          })))
      }
      filas.push(el('button', { type: 'button', class: 'bib-nova', html: ICONA_MAIS + `<span>${t('formularios.bib.novaCarpeta', 'Novo cartafol')}</span>`, 'data-foco': 'nova', onclick: () => novaCarpeta() }))
      lateral.replaceChildren(...filas)
    }

    // Importa un formulario dun ficheiro: entra na biblioteca coma un novo e os avisos
    // (copia por id repetido, modo produción por traer datos...) sáen aquí mesmo.
    async function importar() {
      mostrarErro('')
      mostrarAviso('')
      try {
        const r = await accions.importar()
        if (r?.avisos?.length) mostrarAviso(r.avisos.join(' '))
      } catch (err) { mostrarErro(String(err?.message || err)) }
      pintar('imp')
    }

    async function novaCarpeta() {
      const nome = (await accions.preguntar(t('formularios.bib.novaCarpeta.pregunta', 'Nome do cartafol novo:'), ''))?.trim()
      if (!nome) return
      if (nomeRepetido(accions.biblioteca(), nome)) { mostrarErro(t('formularios.bib.carpetaRepetida', 'Xa hai un cartafol chamado «{nome}».', { nome })); return }
      await executar(() => accions.crearCarpeta(nome), 'nova')
    }
    async function renomear(c) {
      const nome = (await accions.preguntar(t('formularios.bib.renomear.pregunta', 'Novo nome do cartafol:'), c.nome))?.trim()
      if (!nome || nome === c.nome) return
      if (nomeRepetido(accions.biblioteca(), nome, c.id)) { mostrarErro(t('formularios.bib.carpetaRepetida', 'Xa hai un cartafol chamado «{nome}».', { nome })); return }
      await executar(() => accions.gardar(renomearCarpeta(accions.biblioteca(), c.id, nome)), 'r:' + c.id)
    }
    async function eliminarCarp(c, cantidade) {
      const msg = cantidade
        ? t('formularios.bib.eliminarCarpeta.confirmar', 'Eliminar o cartafol «{nome}»? Os {n} formularios que ten NON se borran: pasan a «Sen cartafol».', { nome: c.nome, n: cantidade })
        : t('formularios.bib.eliminarCarpeta.confirmarBaleiro', 'Eliminar o cartafol «{nome}»?', { nome: c.nome })
      if (!await accions.confirmar(msg)) return
      if (carpetaSel === c.id) carpetaSel = null
      await executar(() => accions.gardar(eliminarCarpeta(accions.biblioteca(), c.id)), 'c:todos')
    }

    // ---------- Formularios ----------
    function tarxeta(f) {
      const b = accions.biblioteca()
      const id = 'form:' + f.id
      const produccion = f.modo === 'produccion'
      const modo = produccion ? t('formularios.modo.produccion', 'Produción') : t('formularios.modo.deseno', 'Deseño')
      const descricion = el('input', {
        type: 'text', class: 'form-entrada bib-desc', maxlength: '400', value: f.descricion || '', 'data-foco': 'd:' + f.id,
        placeholder: t('formularios.bib.descricion.ph', 'Engadir unha descrición…'), 'aria-label': t('formularios.bib.descricion.aria', 'Descrición de {nome}', { nome: f.nome }),
      })
      descricion.value = f.descricion || ''
      descricion.addEventListener('change', () => executar(() => accions.gardarDescricion(f.id, descricion.value), 'd:' + f.id))
      const carpeta = el('select', { class: 'form-entrada form-entrada--compacta bib-sel', 'data-foco': 's:' + f.id, 'aria-label': t('formularios.bib.carpeta.aria', 'Cartafol de {nome}', { nome: f.nome }) },
        new Option(t('formularios.bib.senCartafol', 'Sen cartafol'), ''), ...b.carpetas.map((c) => new Option(c.nome, c.id)))
      carpeta.value = b.carpeta[f.id] || ''
      carpeta.addEventListener('change', () => executar(() => accions.gardar(moverACarpeta(accions.biblioteca(), f.id, carpeta.value)), 's:' + f.id))
      const cantidade = (f.campos || []).filter((c) => c.tipo !== 'seccion').length

      return el('article', { class: 'bib-form' + (estaAgochada(b, id) ? ' bib-form--agochado' : '') },
        el('span', { class: `bib-form__icona bib-form__icona--${produccion ? 'produccion' : 'deseno'}`, html: svgIcona(f.icona), 'aria-hidden': 'true' }),
        el('div', { class: 'bib-form__texto' },
          el('div', { class: 'bib-form__tit' },
            el('strong', { text: f.nome }),
            el('span', { class: `bib-chip bib-chip--${produccion ? 'produccion' : 'deseno'}`, text: modo }),
            estaFixa(b, id) ? el('span', { class: 'bib-chip', title: t('formularios.bib.fixo.title', 'Está na zona fixa da columna esquerda: non se pode borrar'), text: t('formularios.bib.fixo', 'Fixo') }) : null,
            estaAgochada(b, id) ? el('span', { class: 'bib-chip bib-chip--agochado', title: t('formularios.bib.agochado.title', 'Non se ve na columna esquerda'), text: t('formularios.bib.agochado', 'Agochado') }) : null,
            el('small', { class: 'bib-form__n', text: t('formularios.bib.campos', '{n} campos', { n: cantidade }) })),
          descricion),
        el('div', { class: 'bib-form__acc' },
          carpeta,
          el('button', { type: 'button', class: 'crono-btn', 'data-foco': 'a:' + f.id, text: t('formularios.bib.abrir', 'Abrir'), onclick: () => abrir(id) }),
          el('button', {
            type: 'button', class: 'form-icona-btn bib-mais', html: ICONA_MAIS_OPC, 'data-foco': 'm:' + f.id,
            title: t('formularios.bib.mais', 'Máis accións'), 'aria-label': t('formularios.bib.mais.aria', 'Máis accións sobre {nome}', { nome: f.nome }), 'aria-haspopup': 'menu',
            onclick: (e) => { const r = e.currentTarget.getBoundingClientRect(); abrirMenu({ x: r.right - 200, y: r.bottom + 4, items: itemsDe(id), etiqueta: f.nome }) },
          })))
    }

    async function abrir(id) {
      mostrarErro('')
      try { await accions.abrir(id) } catch (err) { mostrarErro(String(err?.message || err)); return }
      pechar()
    }

    function pintarLista() {
      const todos = accions.formularios()
      const vistos = filtrar(accions.biblioteca(), todos, { carpeta: carpetaSel, busca })
      if (!vistos.length) {
        lista.replaceChildren(el('p', { class: 'bib-baleira', text: !todos.length
          ? t('formularios.bib.baleira', 'Aínda non tes formularios. Crea o primeiro co botón «+» da columna esquerda.')
          : busca ? t('formularios.bib.sen.resultados', 'Ningún formulario coincide coa busca.') : t('formularios.bib.carpeta.baleira', 'Este cartafol está baleiro. Move formularios aquí co selector de cada un.') }))
        return
      }
      lista.replaceChildren(...vistos.map(tarxeta))
    }

    function pintarPe() {
      const b = accions.biblioteca()
      pe.replaceChildren(
        el('strong', { class: 'bib-pe__tit', text: t('formularios.bib.integradas', 'Ferramentas do programa') }),
        ...accions.integradas.map((f) => {
          const oculta = estaAgochada(b, f.id)
          return el('span', { class: 'bib-integrada' },
            el('span', { text: f.nome }),
            el('button', {
              type: 'button', class: 'crono-btn crono-btn--pequeno', 'data-foco': 'i:' + f.id,
              text: oculta ? t('formularios.menu.amosar', 'Amosar na columna') : t('formularios.menu.ocultar', 'Ocultar da columna'),
              onclick: () => executar(() => accions.gardar(agochar(accions.biblioteca(), f.id, !oculta)), 'i:' + f.id),
            }))
        }),
        el('small', { class: 'bib-pe__axuda', text: t('formularios.bib.integradas.axuda', 'Non se poden borrar, só ocultar. Tamén podes arrastrar un formulario da columna á zona de arriba para fixalo alí.') }))
    }

    function pintar(foco) {
      const clave = foco ?? document.activeElement?.dataset?.foco
      pintarCarpetas()
      pintarLista()
      pintarPe()
      if (clave) [...overlay.querySelectorAll('[data-foco]')].find((e) => e.dataset.foco === clave)?.focus()
    }

    // ---------- Montaxe e peche ----------
    const caixa = el('div', { class: 'bib' },
      el('header', { class: 'bib__cab' },
        el('h2', { text: t('formularios.bib.titulo', 'Biblioteca de formularios') }),
        inBusca,
        accions.importar ? el('button', {
          type: 'button', class: 'crono-btn', 'data-foco': 'imp', text: t('formularios.bib.importar', 'Importar formulario…'),
          title: t('formularios.bib.importar.title', 'Engade á biblioteca un formulario que che pasou un compañeiro (ficheiro exportado con «Exportar formulario…»)'),
          onclick: () => importar(),
        }) : null,
        el('button', { type: 'button', class: 'dis-x', 'aria-label': t('formularios.bib.pechar', 'Pechar a biblioteca'), title: t('formularios.bib.pechar', 'Pechar a biblioteca'), html: ICONA_X, onclick: () => pechar() })),
      erro,
      aviso,
      el('div', { class: 'bib__corpo' }, lateral, lista),
      pe)
    overlay.append(caixa)

    function tecla(e) {
      if (e.key !== 'Escape') return
      // Un menú ou un diálogo aberto encima ten prioridade (eles pechan co seu Esc).
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
    pintar()
    inBusca.focus()
  })
}
