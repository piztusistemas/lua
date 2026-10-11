// @vitest-environment jsdom
import { describe, it, expect, vi, afterEach } from 'vitest'
import { abrirProxectos, opcionsFerramentas } from './proxectos.js'

afterEach(() => { document.body.replaceChildren() })
const t = (k, fb, v) => String(fb).replace(/\{(\w+)\}/g, (_, n) => v?.[n] ?? '')
const P = (id, nome, carpeta = '') => ({ id, nome, materia: '', creado: '2026-10-03', ferramentas: [], activo: false, boveda: false, fontes: 0, wiki: 0, carpeta })

function montar(extra = {}) {
  const estado = { proxectos: [P('a', 'Inglés 1', 'ing'), P('b', 'Mates'), P('c', 'Inglés 2', 'ing')], carpetas: [{ id: 'ing', nome: 'Inglés' }, { id: 'vaz', nome: 'Baleiro' }] }
  const accions = {
    listar: vi.fn(async () => estado.proxectos),
    formularios: vi.fn(async () => []),
    integradas: [],
    crear: vi.fn(async () => P('n', 'Novo')),
    abrir: vi.fn(async () => {}), pechar: vi.fn(async () => {}),
    engadir: vi.fn(), quitar: vi.fn(),
    eliminar: vi.fn(async (id) => { estado.proxectos = estado.proxectos.filter((p) => p.id !== id) }),
    buxo: vi.fn(async () => ({ activo: false, instalado: false, iaConfigurada: false })),
    abrirBuxo: vi.fn(),
    carpetas: vi.fn(async () => estado.carpetas),
    crearCarpeta: vi.fn(async (n) => { estado.carpetas = [...estado.carpetas, { id: 'novo', nome: n }] }),
    renomearCarpeta: vi.fn(), eliminarCarpeta: vi.fn(),
    mover: vi.fn(async () => {}),
    preguntar: vi.fn(async () => 'Física'),
    confirmar: vi.fn(async () => true),
    ...extra,
  }
  abrirProxectos({ t, accions })
  return { accions, estado }
}
const tarxetas = () => [...document.querySelectorAll('.prx-tarxeta strong')].map((e) => e.textContent)
const despregar = (nome) => [...document.querySelectorAll('.prx-tarxeta__cab')].find((b) => b.textContent.includes(nome)).click()
const botonCarp = (txt) => [...document.querySelectorAll('.prx-carpetas .bib-carp')].find((b) => b.textContent.startsWith(txt))

describe('proxectos: cartafoles', () => {
  it('filtra por cartafol e conta os proxectos de cada un', async () => {
    montar()
    await vi.waitFor(() => expect(tarxetas().length).toBe(3))
    expect(botonCarp('Todos').textContent).toMatch(/3/)
    botonCarp('Inglés').click()
    expect(tarxetas()).toEqual(['Inglés 1', 'Inglés 2'])
    botonCarp('Sen cartafol').click()
    expect(tarxetas()).toEqual(['Mates'])
    botonCarp('Baleiro').click()
    expect(document.querySelector('.bib-baleira').textContent).toMatch(/baleiro/)
  })
  it('crear cartafol pregunta o nome; mover usa o selector de cada proxecto', async () => {
    const { accions } = montar()
    await vi.waitFor(() => expect(tarxetas().length).toBe(3))
    document.querySelector('.prx-carpetas .bib-nova').click()
    await vi.waitFor(() => expect(accions.crearCarpeta).toHaveBeenCalledWith('Física'))
    await vi.waitFor(() => expect(tarxetas().length).toBe(3))
    despregar('Mates')
    const sel = document.querySelector('.prx-carpeta')
    sel.value = 'vaz'
    sel.dispatchEvent(new Event('change'))
    await vi.waitFor(() => expect(accions.mover).toHaveBeenCalledWith('b', 'vaz'))
  })
  it('eliminar un proxecto pide confirmación; un cartafol avisa de que non borra proxectos', async () => {
    const { accions } = montar()
    await vi.waitFor(() => expect(tarxetas().length).toBe(3))
    despregar('Mates')
    document.querySelector('.prx-eliminar').click()
    await vi.waitFor(() => expect(accions.eliminar).toHaveBeenCalledWith('b'))
    expect(accions.confirmar.mock.calls[0][0]).toMatch(/Mates/)
    const fila = botonCarp('Inglés').closest('.bib-carp-fila')
    fila.querySelectorAll('button')[2].click()
    await vi.waitFor(() => expect(accions.eliminarCarpeta).toHaveBeenCalledWith('ing'))
    expect(accions.confirmar.mock.calls[1][0]).toMatch(/NON se borran/)
  })
  it('os proxectos saen pregados e desprégase ao premer', async () => {
    montar()
    await vi.waitFor(() => expect(tarxetas().length).toBe(3))
    expect(document.querySelector('.prx-acc')).toBe(null)
    expect(document.body.textContent).not.toContain('Ferramentas…')
    despregar('Mates')
    expect(document.querySelectorAll('.prx-acc').length).toBe(1)
    expect(document.querySelector('.prx-tarxeta__cab[aria-expanded="true"]').textContent).toContain('Mates')
    despregar('Mates')
    expect(document.querySelector('.prx-acc')).toBe(null)
  })
  it('crear un proxecto só pide o nome: nace baleiro', async () => {
    const { accions } = montar()
    await vi.waitFor(() => expect(tarxetas().length).toBe(3))
    expect(document.querySelector('.prx-novo input[type=checkbox]')).toBe(null)
    const nome = document.querySelector('.prx-novo input')
    nome.value = 'Física 2º'
    ;[...document.querySelectorAll('.prx-novo button')].find((b) => b.textContent === 'Crear e abrir').click()
    await vi.waitFor(() => expect(accions.crear).toHaveBeenCalledWith('Física 2º', ''))
    expect(accions.abrir).toHaveBeenCalledWith('n')
  })
  it('opcionsFerramentas pon as integradas antes dos formularios', () => {
    expect(opcionsFerramentas([{ id: 'actas', nome: 'Actas' }], [{ id: 'x', nome: 'X', icona: 'i' }]).map((o) => o.id)).toEqual(['actas', 'form:x'])
  })
})
