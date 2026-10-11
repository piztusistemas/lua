// @vitest-environment jsdom
import { describe, it, expect, vi, afterEach } from 'vitest'
import { abrirCatalogo, filtrarCatalogo } from './catalogo.js'

afterEach(() => { document.body.replaceChildren() })
const t = (k, fb, v) => String(fb).replace(/\{(\w+)\}/g, (_, n) => v?.[n] ?? '')
const F = (id, nome, carpeta = '') => ({ id, nome, carpeta, descricion: '' })

function montar(extra = {}) {
  const estado = {
    integradas: ['cronograma', 'actas'],
    formularios: [F('a', 'Exame tipo test', 'mat'), F('b', 'Ficha de gramática'), F('c', 'Práctica Linux', 'mat')],
    carpetas: [{ id: 'mat', nome: 'Matemáticas' }],
    noProxecto: ['form:c', 'actas'],
  }
  const accions = {
    catalogo: vi.fn(async () => estado),
    nomeIntegrada: (id) => (id === 'actas' ? 'Actas' : 'Cronograma'),
    reutilizar: vi.fn(async () => {}),
    novo: vi.fn(),
    crearCarpeta: vi.fn(async () => {}), renomearCarpeta: vi.fn(async () => {}), eliminarCarpeta: vi.fn(async () => {}),
    mover: vi.fn(async () => {}),
    preguntar: vi.fn(async () => 'Lingua'),
    confirmar: vi.fn(async () => true),
    ...extra,
  }
  const pechada = abrirCatalogo({ t, accions })
  return { accions, estado, pechada }
}
const filas = () => [...document.querySelectorAll('.bib-form strong')].map((e) => e.textContent)
const marcar = (nome) => [...document.querySelectorAll('.bib-form')].find((f) => f.textContent.includes(nome)).querySelector('input[type=checkbox]')
const boton = (txt) => [...document.querySelectorAll('button')].find((b) => b.textContent.startsWith(txt))

describe('ventá de formularios', () => {
  it('amosa o cronograma, as actas e os formularios; os do proxecto saen marcados e sen poder tocalos', async () => {
    montar()
    await vi.waitFor(() => expect(filas().length).toBe(5))
    expect(filas()).toEqual(['Cronograma', 'Actas', 'Exame tipo test', 'Ficha de gramática', 'Práctica Linux'])
    expect(marcar('Actas').disabled).toBe(true)
    expect(marcar('Práctica Linux').checked).toBe(true)
    expect(document.body.textContent).toContain('Xa no proxecto')
  })

  it('marcar e premer Reutilizar mete só os novos e pecha', async () => {
    const { accions, pechada } = montar()
    await vi.waitFor(() => expect(filas().length).toBe(5))
    expect(boton('Reutilizar').disabled).toBe(true)
    marcar('Cronograma').click()
    marcar('Ficha de gramática').click()
    expect(boton('Reutilizar').textContent).toBe('Reutilizar (2)')
    boton('Reutilizar').click()
    await pechada
    expect(accions.reutilizar).toHaveBeenCalledWith(['cronograma', 'form:b'])
    expect(document.querySelector('.cat')).toBe(null)
  })

  it('filtra por cartafol e por busca; o cartafol cámbiase co selector', async () => {
    const { accions } = montar()
    await vi.waitFor(() => expect(filas().length).toBe(5))
    ;[...document.querySelectorAll('.bib-carp')].find((b) => b.textContent.startsWith('Matemáticas')).click()
    expect(filas()).toEqual(['Exame tipo test', 'Práctica Linux'])
    const busca = document.querySelector('.bib-busca')
    busca.value = 'linux'
    busca.dispatchEvent(new Event('input'))
    expect(filas()).toEqual(['Práctica Linux'])
    const sel = document.querySelector('.bib-sel')
    sel.value = ''
    sel.dispatchEvent(new Event('change'))
    await vi.waitFor(() => expect(accions.mover).toHaveBeenCalledWith('c', ''))
  })

  it('crear, renomear e eliminar cartafoles (eliminar avisa de que non borra formularios)', async () => {
    const { accions } = montar()
    await vi.waitFor(() => expect(filas().length).toBe(5))
    document.querySelector('.bib-nova').click()
    await vi.waitFor(() => expect(accions.crearCarpeta).toHaveBeenCalledWith('Lingua'))
    const fila = [...document.querySelectorAll('.bib-carp-fila')][0]
    fila.querySelectorAll('button')[2].click()
    await vi.waitFor(() => expect(accions.eliminarCarpeta).toHaveBeenCalledWith('mat'))
    expect(accions.confirmar.mock.calls[0][0]).toMatch(/NON se borran/)
  })

  it('«Novo formulario» pecha a ventá e abre o deseñador', async () => {
    const { accions, pechada } = montar()
    await vi.waitFor(() => expect(filas().length).toBe(5))
    boton('Novo formulario').click()
    await pechada
    expect(accions.novo).toHaveBeenCalled()
  })

  it('filtrarCatalogo', () => {
    const fs = [F('a', 'Alfa', 'x'), F('b', 'Beta')]
    expect(filtrarCatalogo(fs, null, '').length).toBe(2)
    expect(filtrarCatalogo(fs, '', '').map((f) => f.id)).toEqual(['b'])
    expect(filtrarCatalogo(fs, null, 'ALF').map((f) => f.id)).toEqual(['a'])
  })
})
