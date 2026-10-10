// A COLUMNA ESQUERDA de ferramentas: dúas zonas (a fixa, con cronograma, actas
// e os formularios fixados, e a de abaixo, cos demais formularios), os botóns
// agochados, o menú do botón dereito e o arrastre entre zonas. O estado
// (cartafoles, fixos, agochados) vive en Go; aquí só se pinta e se pide cambialo.
//
//   nav, zonaFixa, zonaResto, separador   elementos do HTML
//   biblioteca()                          estado actual (biblioteca-modelo.js)
//   existentes()                          ids en orde: cronograma, actas e 'form:<id>' por nome
//   nomeDe(id)                            nome amosable (para o «fantasma» do arrastre)
//   gardar(nova)                          persiste un novo estado (Promise; main.js trata os erros)
//   eliminar(id)                          borra un formulario (con confirmación)
//   exportar(id)                          (opcional) exporta un formulario a un ficheiro
//   cambiarIcona(id)                      (opcional) escolle a icona dun formulario
import { columna, accionDeSoltar, destinoDeSoltar } from './biblioteca-modelo.js'
import { itemsDeFerramenta } from './accions-ferramenta.js'
import { abrirMenu } from './menu-contextual.js'
import { iniciarArrastre } from './arrastrar.js'

const SELECTOR = '.ferramenta-btn[data-ferramenta]'
const MARXE_X = 24 // canto se pode saír da columna sen perder o destino

export function crearColumna({ nav, zonaFixa, zonaResto, separador, t, biblioteca, existentes, nomeDe, gardar, eliminar, exportar, cambiarIcona }) {
  const botons = () => new Map([...nav.querySelectorAll(SELECTOR)].map((b) => [b.dataset.ferramenta, b]))

  // pintar reparte os botóns nas zonas (na orde da biblioteca) e agocha os que toquen.
  function pintar() {
    const todos = botons()
    const { fixas, resto } = columna(biblioteca(), existentes())
    const visibles = new Set([...fixas, ...resto])
    for (const [id, b] of todos) b.hidden = !visibles.has(id)
    const en = (ids) => ids.map((id) => todos.get(id)).filter(Boolean)
    const agochados = [...todos.entries()].filter(([id]) => !visibles.has(id)).map(([, b]) => b)
    zonaFixa.replaceChildren(...en(fixas))
    zonaResto.replaceChildren(...en(resto), ...agochados)
    separador.hidden = !(fixas.length && resto.length)
  }

  // ---------- Menú do botón dereito ----------
  function aoMenu(e) {
    const b = e.target.closest(SELECTOR)
    if (!b) return
    e.preventDefault()
    const id = b.dataset.ferramenta
    // Ábrese á dereita da icona, non debaixo do punteiro (alí o tooltip da icona taparíao);
    // así vale igual co rato ca coa tecla de menú.
    const r = b.getBoundingClientRect()
    abrirMenu({
      x: r.right + 6, y: r.top,
      etiqueta: nomeDe(id),
      items: itemsDeFerramenta(id, { t, biblioteca, gardar, eliminar, exportar, cambiarIcona }),
    })
  }

  // ---------- Arrastre entre zonas ----------
  function xeometria(x, y) {
    const r = nav.getBoundingClientRect()
    const s = separador.getBoundingClientRect()
    return {
      dentro: x >= r.left - MARXE_X && x <= r.right + MARXE_X && y >= r.top && y <= r.bottom,
      ySeparador: (s.top + s.bottom) / 2,
      fixas: [...zonaFixa.querySelectorAll(SELECTOR)].filter((b) => !b.hidden).map((b) => {
        const br = b.getBoundingClientRect()
        return { id: b.dataset.ferramenta, mediaY: (br.top + br.bottom) / 2 }
      }),
    }
  }
  function limparMarcas() {
    nav.classList.remove('ferramentas-nav--arrastrando')
    zonaFixa.classList.remove('ferramentas-nav__zona--soltar')
    zonaResto.classList.remove('ferramentas-nav__zona--soltar')
    nav.querySelectorAll('.ferramenta-btn--soltar-antes, .ferramenta-btn--soltar-despois').forEach((b) => b.classList.remove('ferramenta-btn--soltar-antes', 'ferramenta-btn--soltar-despois'))
  }
  function marcar(id, x, y) {
    limparMarcas()
    nav.classList.add('ferramentas-nav--arrastrando')
    const destino = destinoDeSoltar(y, xeometria(x, y))
    // Só se marca onde o arrastre ten efecto: o resto ensina que non caerá.
    if (!accionDeSoltar(biblioteca(), id, destino)) return
    if (destino.zona === 'resto') { zonaResto.classList.add('ferramentas-nav__zona--soltar'); return }
    zonaFixa.classList.add('ferramentas-nav__zona--soltar')
    const fixas = [...zonaFixa.querySelectorAll(SELECTOR)].filter((b) => !b.hidden && b.dataset.ferramenta !== id)
    const ref = fixas.find((b) => b.dataset.ferramenta === destino.antesDe)
    if (ref) ref.classList.add('ferramenta-btn--soltar-antes')
    else fixas[fixas.length - 1]?.classList.add('ferramenta-btn--soltar-despois')
  }
  function aoPremer(e) {
    const b = e.target.closest(SELECTOR)
    if (!b) return
    const id = b.dataset.ferramenta
    iniciarArrastre(e, {
      texto: nomeDe(id),
      aoMover: (x, y) => marcar(id, x, y),
      aoSoltar: (x, y) => {
        const nova = accionDeSoltar(biblioteca(), id, destinoDeSoltar(y, xeometria(x, y)))
        if (nova) gardar(nova)
      },
      aoRematar: limparMarcas,
    })
  }

  nav.addEventListener('contextmenu', aoMenu)
  nav.addEventListener('pointerdown', aoPremer)
  return {
    pintar,
    destruir() { nav.removeEventListener('contextmenu', aoMenu); nav.removeEventListener('pointerdown', aoPremer) },
  }
}
