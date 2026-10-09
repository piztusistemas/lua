// Menú contextual (botón dereito ou menú «⋯») propio, en HTML: os das webviews
// de escritorio non se poden personalizar e difiren entre sistemas.
//
//   abrirMenu({ x, y, items, aoPechar })
//   items: [{ texto, aoPremer, desactivado?, titulo?, perigo?, separador? }]
//
// Acesible: role=menu/menuitem, frechas, Inicio/Fin, Esc, e devolve o foco a
// onde estaba. Un item desactivado segue visible (co seu `titulo` a explicar
// por que) pero non fai nada.
import { el } from '../informes/dom.js'

let aberto = null

export function pecharMenu() { aberto?.pechar() }

export function abrirMenu({ x, y, items, aoPechar, etiqueta = '' }) {
  pecharMenu()
  const previo = document.activeElement
  const menu = el('div', { class: 'menu-ctx', role: 'menu', 'aria-label': etiqueta || undefined })
  const botons = []
  let cerrado = false

  const pechar = (devolverFoco = true) => {
    if (cerrado) return
    cerrado = true
    document.removeEventListener('pointerdown', fora, true)
    document.removeEventListener('keydown', tecla, true)
    window.removeEventListener('blur', aoPerderFoco, true)
    window.removeEventListener('resize', pecharSen, true)
    window.removeEventListener('scroll', pecharSen, true)
    menu.remove()
    if (aberto && aberto.menu === menu) aberto = null
    if (devolverFoco && previo && previo.isConnected) previo.focus?.()
    aoPechar?.()
  }
  const pecharSen = () => pechar(false)
  // O blur en captura tamén salta cando muda o foco entre os itens: só conta o da xanela.
  const aoPerderFoco = (e) => { if (e.target === window) pechar(false) }
  const fora = (e) => { if (!menu.contains(e.target)) pechar(false) }
  const enfocar = (i) => {
    const n = botons.length
    if (!n) return
    botons[((i % n) + n) % n].focus()
  }
  const tecla = (e) => {
    const i = botons.indexOf(document.activeElement)
    if (e.key === 'Escape') { e.preventDefault(); e.stopPropagation(); pechar() }
    else if (e.key === 'ArrowDown') { e.preventDefault(); enfocar(i + 1) }
    else if (e.key === 'ArrowUp') { e.preventDefault(); enfocar(i < 0 ? -1 : i - 1) }
    else if (e.key === 'Home') { e.preventDefault(); enfocar(0) }
    else if (e.key === 'End') { e.preventDefault(); enfocar(-1) }
    else if (e.key === 'Tab') { e.preventDefault(); pechar() }
  }

  for (const it of items) {
    if (it.separador) { menu.append(el('div', { class: 'menu-ctx__sep', role: 'separator' })); continue }
    const b = el('button', {
      type: 'button', role: 'menuitem', class: 'menu-ctx__item' + (it.perigo ? ' menu-ctx__item--perigo' : '') + (it.desactivado ? ' menu-ctx__item--desactivado' : ''),
      'aria-disabled': it.desactivado ? 'true' : undefined, title: it.titulo || undefined, text: it.texto,
    })
    b.addEventListener('click', (e) => {
      e.stopPropagation()
      if (it.desactivado) return
      pechar(false)
      it.aoPremer?.()
    })
    botons.push(b)
    menu.append(b)
  }

  document.body.append(menu)
  // Dentro da xanela: se non cabe á dereita ou abaixo, ábrese cara ao outro lado.
  const r = menu.getBoundingClientRect()
  const ancho = window.innerWidth || 1024
  const alto = window.innerHeight || 768
  menu.style.left = `${Math.max(4, Math.min(x, ancho - r.width - 4))}px`
  menu.style.top = `${Math.max(4, Math.min(y, alto - r.height - 4))}px`

  document.addEventListener('pointerdown', fora, true)
  document.addEventListener('keydown', tecla, true)
  window.addEventListener('blur', aoPerderFoco, true)
  window.addEventListener('resize', pecharSen, true)
  window.addEventListener('scroll', pecharSen, true)
  aberto = { menu, pechar }
  enfocar(botons.findIndex((b) => b.getAttribute('aria-disabled') !== 'true'))
  return { pechar }
}
