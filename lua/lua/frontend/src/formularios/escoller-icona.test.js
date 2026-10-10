// @vitest-environment jsdom
import { describe, it, expect, afterEach } from 'vitest'
import { escollerIcona } from './escoller-icona.js'
import { CLAVES_ICONA } from './iconas.js'

const t = (_k, galego, vars) => (vars ? galego.replace('{nome}', vars.nome) : galego)
const q = (s) => document.querySelector(s)
const qa = (s) => [...document.querySelectorAll(s)]

describe('xanela «Cambiar icona»', () => {
  afterEach(() => { document.body.innerHTML = '' })

  it('amosa todas as iconas, marca a actual e devolve a escollida', async () => {
    const p = escollerIcona({ t, nome: 'Informe de titoría', actual: 'reloxo' })
    expect(q('.escoller-icona').textContent).toMatch(/Informe de titoría/)
    expect(qa('.escoller-icona .dis-icona').length).toBe(CLAVES_ICONA.length)
    expect(q('.dis-icona--sel').dataset.icona).toBe('reloxo')
    q('.dis-icona[data-icona="estrela"]').click()
    expect(await p).toBe('estrela')
    expect(q('.escoller-icona')).toBeNull()
  })

  it('sen icona propia marca a do programa; Cancelar e Escape non cambian nada', async () => {
    let p = escollerIcona({ t, nome: 'X', actual: '' })
    expect(q('.dis-icona--sel').dataset.icona).toBe('formulario')
    q('.escoller-icona .modal-actions button').click()
    expect(await p).toBeNull()
    p = escollerIcona({ t, nome: 'X', actual: 'lista' })
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    expect(await p).toBeNull()
  })
})
