// @vitest-environment jsdom
import { describe, it, expect, afterEach } from 'vitest'
import { escollerIcona } from './escoller-icona.js'
import { CLAVES_ICONA, CLAVES_COR } from './iconas.js'

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
    expect(await p).toEqual({ icona: 'estrela', cor: CLAVES_COR[0] })
    expect(q('.escoller-icona')).toBeNull()
  })

  it('premer unha cor márcaa e pinta as iconas nela; Gardar devolve a icona actual coa cor nova', async () => {
    const p = escollerIcona({ t, nome: 'X', actual: 'libro', cor: 'azul' })
    expect(q('.dis-cor--sel').dataset.cor).toBe('azul')
    q('.dis-cor[data-cor="rosa"]').click()
    expect(qa('.dis-cor--sel').map((b) => b.dataset.cor)).toEqual(['rosa'])
    expect(q('.escoller-icona')).not.toBeNull() // a cor non pecha
    expect(q('.escoller-icona__grella').style.getPropertyValue('--cor-formulario')).toMatch(/oklch/)
    q('.escoller-icona .modal-actions button.primary').click()
    expect(await p).toEqual({ icona: 'libro', cor: 'rosa' })
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
