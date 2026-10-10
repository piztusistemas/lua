// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { crearColumna } from './columna.js'
import { bibliotecaBaleira as bibliotecaPorDefecto, INTEGRADAS, fixar, agochar } from './biblioteca-modelo.js'

// Fixture: o cronograma e as actas activos na zona fixa (o por defecto é só na biblioteca).
const bibliotecaBaleira = () => ({ ...bibliotecaPorDefecto(), fixos: [...INTEGRADAS], agochados: [] })

const IDS = ['cronograma', 'actas', 'form:a', 'form:b', 'form:c']
const NOMES = { cronograma: 'Cronograma', actas: 'Actas', 'form:a': 'Alfa', 'form:b': 'Beta', 'form:c': 'Gamma' }
const t = (k, f) => f ?? k

let nav, zonaFixa, zonaResto, separador, bib, gardar, eliminar, cambiarIcona, col

function montar(inicial = bibliotecaBaleira()) {
  document.body.innerHTML = ''
  nav = document.createElement('nav')
  zonaFixa = document.createElement('div')
  separador = document.createElement('span')
  zonaResto = document.createElement('div')
  nav.append(zonaFixa, separador, zonaResto)
  document.body.append(nav)
  for (const id of IDS) {
    const b = document.createElement('button')
    b.className = 'ferramenta-btn'
    b.dataset.ferramenta = id
    zonaResto.append(b)
  }
  bib = inicial
  gardar = vi.fn(async (nova) => { bib = nova; col.pintar() })
  eliminar = vi.fn(async () => {})
  cambiarIcona = vi.fn(async () => {})
  col = crearColumna({ nav, zonaFixa, zonaResto, separador, t, biblioteca: () => bib, existentes: () => IDS, nomeDe: (id) => NOMES[id], gardar, eliminar, cambiarIcona })
  col.pintar()
}

const btn = (id) => nav.querySelector(`[data-ferramenta="${id}"]`)
const ids = (zona) => [...zona.querySelectorAll('.ferramenta-btn')].filter((b) => !b.hidden).map((b) => b.dataset.ferramenta)
const menu = () => document.querySelector('.menu-ctx')
const itensMenu = () => [...document.querySelectorAll('.menu-ctx__item')]
const dereito = (id) => btn(id).dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 30, clientY: 100 }))

beforeEach(() => montar())
afterEach(() => { col.destruir(); document.body.innerHTML = '' })

describe('pintar', () => {
  it('reparte os botóns nas zonas, coa orde da biblioteca', () => {
    expect(ids(zonaFixa)).toEqual(['cronograma', 'actas'])
    expect(ids(zonaResto)).toEqual(['form:a', 'form:b', 'form:c'])
    expect(separador.hidden).toBe(false)
  })
  it('un formulario fixado pasa á zona fixa, e os agochados non se ven (pero seguen no DOM)', () => {
    bib = agochar(fixar(bib, 'form:b'), 'actas')
    col.pintar()
    expect(ids(zonaFixa)).toEqual(['cronograma', 'form:b'])
    expect(ids(zonaResto)).toEqual(['form:a', 'form:c'])
    expect(btn('actas').hidden).toBe(true)
    expect(btn('actas').isConnected).toBe(true)
  })
  it('o separador só se ve se hai algo en cada zona', () => {
    bib = { ...bib, agochados: ['form:a', 'form:b', 'form:c'] }
    col.pintar()
    expect(separador.hidden).toBe(true)
  })
})

describe('menú do botón dereito', () => {
  it('nun formulario de abaixo: fixar, ocultar, cambiar icona e eliminar', () => {
    dereito('form:b')
    expect(itensMenu().map((i) => i.textContent)).toEqual(['Fixar na zona superior', 'Ocultar da columna', 'Cambiar icona…', 'Eliminar formulario…'])
    expect(itensMenu()[3].getAttribute('aria-disabled')).toBeNull()
  })
  it('Eliminar chama a eliminar co id (a confirmación é de quen o implementa) e pecha o menú', () => {
    dereito('form:b')
    itensMenu()[3].click()
    expect(eliminar).toHaveBeenCalledWith('form:b')
    expect(menu()).toBeNull()
  })
  it('Fixar e Ocultar piden gardar o novo estado', () => {
    dereito('form:a')
    itensMenu()[0].click()
    expect(gardar.mock.calls[0][0].fixos).toEqual(['cronograma', 'actas', 'form:a'])
    dereito('form:c')
    itensMenu()[1].click()
    expect(gardar.mock.calls[1][0].agochados).toEqual(['form:c'])
  })
  it('nun formulario FIXO: quitar da zona fixa, e Eliminar desactivado (non fai nada)', () => {
    bib = fixar(bib, 'form:a'); col.pintar()
    dereito('form:a')
    expect(itensMenu().map((i) => i.textContent)).toEqual(['Quitar da zona fixa', 'Ocultar da columna', 'Cambiar icona…', 'Eliminar formulario…'])
    const borrar = itensMenu()[3]
    expect(borrar.getAttribute('aria-disabled')).toBe('true')
    expect(borrar.title).toMatch(/zona fixa/)
    borrar.click()
    expect(eliminar).not.toHaveBeenCalled()
    expect(menu()).not.toBeNull() // segue aberto: non se executou
    itensMenu()[0].click()
    expect(gardar.mock.calls[0][0].fixos).toEqual(['cronograma', 'actas'])
  })
  it('o cronograma e as actas só se poden ocultar (nunca borrar)', () => {
    dereito('cronograma')
    expect(itensMenu().map((i) => i.textContent)).toEqual(['Ocultar da columna'])
    itensMenu()[0].click()
    expect(gardar.mock.calls[0][0].agochados).toEqual(['cronograma'])
  })
  it('pecha con Esc e ao premer fóra', () => {
    dereito('form:a')
    expect(menu()).not.toBeNull()
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    expect(menu()).toBeNull()
    dereito('form:a')
    document.body.dispatchEvent(new MouseEvent('pointerdown', { bubbles: true }))
    expect(menu()).toBeNull()
  })
  it('só hai un menú aberto á vez, e fóra dos botóns non sae ningún', () => {
    dereito('form:a'); dereito('form:b')
    expect(document.querySelectorAll('.menu-ctx').length).toBe(1)
    nav.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true }))
    expect(document.querySelectorAll('.menu-ctx').length).toBe(1)
  })
  it('as frechas moven o foco', () => {
    dereito('form:a')
    expect(document.activeElement).toBe(itensMenu()[0])
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowDown', bubbles: true }))
    expect(document.activeElement).toBe(itensMenu()[1])
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowUp', bubbles: true }))
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowUp', bubbles: true }))
    expect(document.activeElement).toBe(itensMenu().at(-1)) // dá a volta á última
  })
})

describe('arrastre entre zonas', () => {
  // A xeometría de jsdom é toda ceros: póñense retángulos de verdade.
  function xeometria() {
    const rect = (top, bottom, left = 0, right = 56) => ({ top, bottom, left, right, width: right - left, height: bottom - top, x: left, y: top })
    nav.getBoundingClientRect = () => rect(0, 600)
    separador.getBoundingClientRect = () => rect(198, 202)
    let y = 10
    for (const b of zonaFixa.querySelectorAll('.ferramenta-btn')) { const t0 = y; b.getBoundingClientRect = () => rect(t0, t0 + 40); y += 50 }
    y = 220
    for (const b of zonaResto.querySelectorAll('.ferramenta-btn')) { const t0 = y; b.getBoundingClientRect = () => rect(t0, t0 + 40); y += 50 }
  }
  const punteiro = (tipo, x, y, alvo = window) => alvo.dispatchEvent(new MouseEvent(tipo, { bubbles: true, cancelable: true, button: 0, clientX: x, clientY: y }))
  function arrastrar(id, ata) {
    xeometria()
    const b = btn(id)
    const r = b.getBoundingClientRect()
    punteiro('pointerdown', 20, (r.top + r.bottom) / 2, b)
    punteiro('pointermove', 20, (r.top + r.bottom) / 2 + 12)
    punteiro('pointermove', ata[0], ata[1])
    punteiro('pointerup', ata[0], ata[1])
  }

  it('un formulario de abaixo, solto na zona fixa, queda fixado onde caia (entre o cronograma e as actas)', () => {
    arrastrar('form:b', [20, 60]) // 60 > mediaY do cronograma (30) e < mediaY das actas (80)
    expect(gardar).toHaveBeenCalledTimes(1)
    expect(gardar.mock.calls[0][0].fixos).toEqual(['cronograma', 'form:b', 'actas'])
    expect(ids(zonaFixa)).toEqual(['cronograma', 'form:b', 'actas'])
  })
  it('solto ao final da zona fixa, queda ao final', () => {
    arrastrar('form:a', [20, 150])
    expect(gardar.mock.calls[0][0].fixos).toEqual(['cronograma', 'actas', 'form:a'])
  })
  it('un formulario fixado arrastrado á zona de abaixo sae da zona fixa', () => {
    bib = fixar(bib, 'form:a'); col.pintar()
    arrastrar('form:a', [20, 400])
    expect(gardar.mock.calls[0][0].fixos).toEqual(['cronograma', 'actas'])
  })
  it('o cronograma tamén pode saír da zona fixa arrastrándoo abaixo', () => {
    arrastrar('cronograma', [20, 400])
    expect(gardar.mock.calls[0][0].fixos).toEqual(['actas'])
  })
  it('reordenar dentro da zona fixa', () => {
    arrastrar('actas', [20, 12])
    expect(gardar.mock.calls[0][0].fixos).toEqual(['actas', 'cronograma'])
  })
  it('soltar fóra da columna, ou premer sen arrastrar, non fai nada', () => {
    arrastrar('form:b', [400, 60])
    expect(gardar).not.toHaveBeenCalled()
    xeometria()
    punteiro('pointerdown', 20, 240, btn('form:a'))
    punteiro('pointerup', 20, 240)
    expect(gardar).not.toHaveBeenCalled()
  })
  it('durante o arrastre marca a zona e o sitio onde caería, e limpa ao rematar', () => {
    xeometria()
    punteiro('pointerdown', 20, 240, btn('form:a'))
    punteiro('pointermove', 20, 60)
    expect(zonaFixa.classList.contains('ferramentas-nav__zona--soltar')).toBe(true)
    expect(btn('actas').classList.contains('ferramenta-btn--soltar-antes')).toBe(true)
    punteiro('pointermove', 20, 400) // abaixo: un formulario que xa está abaixo non ten efecto: sen marca
    expect(zonaFixa.classList.contains('ferramentas-nav__zona--soltar')).toBe(false)
    expect(zonaResto.classList.contains('ferramentas-nav__zona--soltar')).toBe(false)
    punteiro('pointermove', 20, 60)
    punteiro('pointerup', 20, 60)
    expect(nav.querySelector('.ferramenta-btn--soltar-antes')).toBeNull()
    expect(nav.classList.contains('ferramentas-nav--arrastrando')).toBe(false)
  })
})

describe('cambiar icona', () => {
  it('o botón dereito dun formulario ofrece «Cambiar icona…» e chama con ese formulario', async () => {
    // Un arrastre dun test anterior engule o clic seguinte ata o próximo ciclo (arrastrar.js).
    await new Promise((r) => setTimeout(r, 0))
    dereito('form:b')
    const item = itensMenu().find((i) => /Cambiar icona/.test(i.textContent))
    expect(item).toBeTruthy()
    item.click()
    expect(cambiarIcona).toHaveBeenCalledWith('form:b')
  })
  it('o cronograma e as actas non teñen esa opción (a súa icona é fixa)', () => {
    dereito('cronograma')
    expect(itensMenu().some((i) => /Cambiar icona/.test(i.textContent))).toBe(false)
  })
})
