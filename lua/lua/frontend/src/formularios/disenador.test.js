// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { abrirDisenador } from './disenador.js'

const q = (s) => document.querySelector(s)
const qa = (s) => [...document.querySelectorAll(s)]
const escribir = (input, valor) => { input.value = valor; input.dispatchEvent(new Event('input', { bubbles: true })) }
const tipo = (t) => q(`.dis-tipo[data-tipo="${t}"]`)

describe('deseñador de formularios', () => {
  let api
  beforeEach(() => {
    document.body.innerHTML = ''
    api = { GardarFormulario: vi.fn(async (e) => ({ ...e, id: 'informe' })), EliminarFormulario: vi.fn(async () => {}) }
  })
  afterEach(() => { document.body.innerHTML = '' })

  it('engade campos premendo na paleta e garda o esquema', async () => {
    const p = abrirDisenador({ esquema: null, api })
    escribir(q('.dis-cab__nome input'), '  Informe ')
    tipo('texto').click()
    tipo('data').click()
    expect(qa('.dis-campo').length).toBe(2)
    q('.dis-gardar').click()
    const r = await p
    expect(r.accion).toBe('gardado')
    expect(r.novo).toBe(true)
    const enviado = api.GardarFormulario.mock.calls[0][0]
    expect(enviado.nome).toBe('Informe') // sen espazos
    expect(enviado.campos.map((c) => c.tipo)).toEqual(['texto', 'data'])
    expect(document.querySelector('.dis-overlay')).toBeNull()
  })

  it('a paleta vai na orde pedida e destaca o campo de cálculo', async () => {
    abrirDisenador({ esquema: null, api })
    expect(qa('.dis-tipo').map((b) => b.dataset.tipo)).toEqual(
      ['seccion', 'taboa', 'texto', 'textolongo', 'numero', 'data', 'escolla', 'verificacion', 'imaxe', 'problemas'])
    expect(qa('.dis-tipo--destacado').map((b) => b.dataset.tipo)).toEqual(['problemas'])
    expect(tipo('problemas').textContent).toBe('Cálculo')
  })

  it('non garda sen nome nin cun campo sen etiqueta, e di por que', async () => {
    const p = abrirDisenador({ esquema: null, api })
    tipo('texto').click()
    q('.dis-gardar').click()
    expect(q('.dis-erro').hidden).toBe(false)
    expect(q('.dis-erro').textContent).toMatch(/nome/i)
    escribir(q('.dis-cab__nome input'), 'X')
    escribir(q('.dis-props input'), '   ') // etiqueta en branco
    q('.dis-gardar').click()
    expect(q('.dis-erro').textContent).toMatch(/etiqueta/i)
    expect(api.GardarFormulario).not.toHaveBeenCalled()
    q('.dis-pe button').click() // cancelar
    expect(await p).toBeNull()
  })

  it('editar dúas columnas seguidas non pisa a primeira (o estado non se le do panel pintado)', async () => {
    const p = abrirDisenador({ esquema: null, api })
    escribir(q('.dis-cab__nome input'), 'X')
    tipo('taboa').click()
    const [c1, c2] = qa('.dis-cols__fila input')
    escribir(c1, 'Nome')
    escribir(c2, 'Nota')
    // O resumo da tarxeta (que sae do estado) ten que levar as dúas.
    expect(q('.dis-campo__detalle').textContent).toBe('Nome | Nota')
    q('.dis-gardar').click()
    await p
    expect(api.GardarFormulario.mock.calls[0][0].campos[0].columnas.map((c) => c.etiqueta)).toEqual(['Nome', 'Nota'])
  })

  it('subir, baixar, duplicar e quitar', async () => {
    abrirDisenador({ esquema: { id: 'f', nome: 'F', campos: [
      { id: 'a', tipo: 'texto', etiqueta: 'A' }, { id: 'b', tipo: 'texto', etiqueta: 'B' }, { id: 'c', tipo: 'texto', etiqueta: 'C' },
    ] }, api })
    const nomes = () => qa('.dis-campo strong').map((n) => n.textContent).join('')
    expect(nomes()).toBe('ABC')
    qa('.dis-campo')[0].querySelector('[aria-label="Baixar"]').click()
    expect(nomes()).toBe('BAC')
    qa('.dis-campo')[2].querySelector('[aria-label="Subir"]').click()
    expect(nomes()).toBe('BCA')
    qa('.dis-campo')[0].querySelector('[aria-label="Duplicar"]').click()
    expect(nomes()).toBe('BBCA')
    qa('.dis-campo')[1].querySelector('[aria-label="Quitar campo"]').click()
    expect(nomes()).toBe('BCA')
  })

  it('un formulario existente ofrece eliminar (con confirmación) e cancelar avisa dos cambios', async () => {
    const confirmar = vi.fn(async () => true)
    const p = abrirDisenador({ esquema: { id: 'f', nome: 'F', campos: [] }, api, confirmar })
    q('.dis-eliminar').click()
    const r = await p
    expect(confirmar).toHaveBeenCalledOnce()
    expect(api.EliminarFormulario).toHaveBeenCalledWith('f')
    expect(r).toEqual({ accion: 'eliminado', id: 'f' })

    const nega = vi.fn(async () => false)
    abrirDisenador({ esquema: { id: 'g', nome: 'G', campos: [] }, api, confirmar: nega })
    escribir(q('.dis-cab__nome input'), 'G2') // hai cambios
    q('.dis-pe button:not(.dis-gardar)').click()
    await Promise.resolve()
    expect(nega).toHaveBeenCalledOnce()
    expect(q('.dis-overlay')).not.toBeNull() // seguiu aberto
  })

  it('mostra o erro de Go sen pechar', async () => {
    api.GardarFormulario = vi.fn(async () => { throw new Error('campo 1: tipo raro') })
    abrirDisenador({ esquema: null, api })
    escribir(q('.dis-cab__nome input'), 'X')
    q('.dis-gardar').click()
    await new Promise((r) => setTimeout(r))
    expect(q('.dis-erro').textContent).toContain('tipo raro')
    expect(q('.dis-overlay')).not.toBeNull()
  })

  it('en modo xanela píntase no contedor (sen capa), avisa dos cambios e deixa pedir o pechado desde fóra', async () => {
    const alvo = document.createElement('div')
    document.body.append(alvo)
    const aoSucio = vi.fn()
    let control = null
    const confirmar = vi.fn(async () => true)
    const p = abrirDisenador({ esquema: { id: 'f', nome: 'F', campos: [] }, api, confirmar, alvo, aoSucio, aoPreparado: (c) => { control = c } })
    expect(alvo.querySelector('.dis.dis--xanela')).not.toBeNull()
    expect(document.querySelector('.dis-overlay')).toBeNull() // non hai capa
    expect(aoSucio).not.toHaveBeenCalled()
    escribir(alvo.querySelector('.dis-cab__nome input'), 'G')
    escribir(alvo.querySelector('.dis-cab__nome input'), 'GG')
    expect(aoSucio).toHaveBeenCalledTimes(1) // só a primeira vez
    expect(aoSucio).toHaveBeenCalledWith(true)
    // O xestor de xanelas quere pechar: pregunta (hai cambios) e, se se confirma, resolve con null.
    await control.pedirPechar()
    expect(confirmar).toHaveBeenCalledOnce()
    expect(await p).toBeNull()
    expect(alvo.querySelector('.dis')).toBeNull() // recolleuse
  })

  it('en modo xanela, gardar resolve co formulario e recolle o deseñador', async () => {
    const alvo = document.createElement('div')
    document.body.append(alvo)
    const p = abrirDisenador({ esquema: null, api, alvo })
    escribir(alvo.querySelector('.dis-cab__nome input'), 'Nova')
    alvo.querySelector('.dis-gardar').click()
    const r = await p
    expect(r.accion).toBe('gardado')
    expect(alvo.querySelector('.dis')).toBeNull()
  })
})

describe('pestana de tubaxes', () => {
  const CAT = [
    { tipo: 'campo', nome: 'Campo', categoria: 'orixe' },
    { tipo: 'titulo', nome: 'Título', categoria: 'destino' },
  ]
  const automatico = () => ({
    nodos: [{ id: 'c_a', tipo: 'campo', campo: 'a' }, { id: 't1', tipo: 'titulo', config: { texto: 'Informe' } }],
    documento: ['t1'],
  })
  let api
  beforeEach(() => {
    document.body.innerHTML = ''
    api = {
      GardarFormulario: vi.fn(async (e) => ({ ...e, id: 'informe' })),
      EliminarFormulario: vi.fn(async () => {}),
      CatalogoFluxo: vi.fn(async () => CAT),
      FluxoAutomaticoDe: vi.fn(async () => automatico()),
      SincronizarFluxoFormulario: vi.fn(async (e) => e.fluxo),
      AvaliarFluxoFormulario: vi.fn(async () => ({ nodos: {}, bloques: [] })),
    }
  })
  afterEach(() => { document.body.innerHTML = '' })

  const abrirTubaxes = async () => {
    const p = abrirDisenador({ esquema: null, api })
    escribir(q('.dis-cab__nome input'), 'Informe')
    tipo('texto').click()
    qa('.dis-tab')[1].click()
    await vi.waitFor(() => expect(q('.flx__barra')).not.toBeNull())
    return { p } // nun obxecto: un await sobre a promesa agardaría a pechar o deseñador
  }

  it('amosa o deseño de inmediato, sen texto explicativo nin botón, e segue en automático', async () => {
    const { p } = await abrirTubaxes()
    expect(q('.dis-fluxo__intro')).toBeNull()
    expect(qa('button').some((b) => /Personalizar/.test(b.textContent))).toBe(false)
    expect(qa('button').some((b) => /cableado automático/.test(b.textContent))).toBe(false)
    q('.dis-gardar').click()
    await p
    expect(api.GardarFormulario.mock.calls[0][0].fluxo).toBeUndefined()
  })

  it('o primeiro cambio no deseño personaliza a tubaxe', async () => {
    const { p } = await abrirTubaxes()
    q('button[aria-label="Quitar o nodo"]').click()
    await vi.waitFor(() => expect(qa('button').some((b) => /cableado automático/.test(b.textContent))).toBe(true))
    q('.dis-gardar').click()
    await p
    const enviado = api.GardarFormulario.mock.calls[0][0]
    expect(enviado.fluxo).toBeDefined()
    expect(enviado.fluxo.nodos.map((n) => n.id)).toEqual(['c_a'])
  })
})
