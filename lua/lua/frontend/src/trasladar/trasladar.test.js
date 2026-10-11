// @vitest-environment jsdom
import { describe, it, expect, vi, afterEach } from 'vitest'
import { abrirTrasladar, trasladarAberta } from './trasladar.js'

const t = (_c, f, v) => { let s = f ?? _c; for (const k in v || {}) s = s.replaceAll(`{${k}}`, v[k]); return s }
const agarda = () => new Promise((r) => setTimeout(r, 0))

function apiFalsa(over = {}) {
  return {
    ArboreTraslado: vi.fn(async () => ({ proxectos: [{ id: 'p1', nome: 'P1' }], formularios: [], plantillas: [] })),
    SacarTenDatosProducion: vi.fn(async () => false),
    SacarTraslado: vi.fn(async () => ({ ruta: '/tmp/p1.lua' })),
    SacarDocumentoJSON: vi.fn(async () => ({ ruta: '/tmp/doc.json' })),
    contidoDocumento: () => '{"plans":[]}',
    EscollerFicheiroMeter: vi.fn(async () => '/tmp/x.json'),
    EscollerCartafolMeter: vi.fn(async () => ''),
    AnalizarMeter: vi.fn(async () => ({ tipo: 'documento', nome: 'Crono', elementos: [{ tipo: 'documento', nome: 'Crono' }], destino: 'aberto', podeAberto: true, nomeProxectoAberto: 'P1', cursos: [] })),
    AplicarMeter: vi.fn(async () => ({ proxectos: ['p1'], version: 'v1', recargar: true })),
    DesfacerTraslado: vi.fn(async () => {}),
    ...over,
  }
}
const contexto = { proxecto: { id: 'p1', nome: 'P1' }, documento: { ferramenta: 'cronograma', nome: 'Cronograma' } }

afterEach(() => {
  document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
  document.body.replaceChildren()
})

describe('Trasladar', () => {
  it('saca o proxecto aberto cun clic', async () => {
    const api = apiFalsa()
    abrirTrasladar({ t, api, contexto, confirmar: async () => true })
    expect(trasladarAberta()).toBe(true)
    const marcado = document.querySelector('input[name="trasOpcion"]:checked')
    expect(marcado.value).toBe('proxecto')
    ;[...document.querySelectorAll('button')].find((b) => b.textContent === 'Sacar…').click()
    await agarda()
    expect(api.SacarTraslado).toHaveBeenCalledWith({ tipo: 'proxecto', proxectos: ['p1'], formularios: [], plantillas: [], datos: true })
    expect(document.body.textContent).toContain('Gardado en /tmp/p1.lua')
  })

  it('o documento sae coma .json editable', async () => {
    const api = apiFalsa()
    abrirTrasladar({ t, api, contexto, opcion: 'documento', confirmar: async () => true })
    expect(document.querySelector('#trasDatos').closest('label').hidden).toBe(true)
    ;[...document.querySelectorAll('button')].find((b) => b.textContent === 'Sacar…').click()
    await agarda()
    expect(api.SacarDocumentoJSON).toHaveBeenCalledWith('cronograma', '{"plans":[]}', 'Cronograma')
  })

  it('con datos de produción pregunta e, se non, saca sen datos', async () => {
    const api = apiFalsa({ SacarTenDatosProducion: vi.fn(async () => true) })
    const confirmar = vi.fn(async () => false)
    abrirTrasladar({ t, api, contexto, confirmar })
    ;[...document.querySelectorAll('button')].find((b) => b.textContent === 'Sacar…').click()
    await agarda(); await agarda()
    expect(confirmar).toHaveBeenCalled()
    expect(api.SacarTraslado.mock.calls[0][0].datos).toBe(false)
  })

  it('mete: resume, aplica e permite desfacer', async () => {
    const api = apiFalsa()
    const antes = vi.fn(async () => {})
    const despois = vi.fn(async () => {})
    abrirTrasladar({ t, api, contexto, lapela: 'meter', confirmar: async () => true, antesDeMeter: antes, despoisDeMeter: despois })
    ;[...document.querySelectorAll('button')].find((b) => b.textContent === 'Escoller ficheiro…').click()
    await agarda(); await agarda()
    expect(document.body.textContent).toContain('Vai ao proxecto aberto, «P1».')
    ;[...document.querySelectorAll('button')].find((b) => b.textContent === 'Meter').click()
    await agarda(); await agarda()
    expect(antes).toHaveBeenCalled()
    expect(api.AplicarMeter).toHaveBeenCalledWith('/tmp/x.json', { destino: 'aberto', cursos: null, accion: '' })
    expect(despois).toHaveBeenCalled()
    ;[...document.querySelectorAll('button')].find((b) => b.textContent === 'Desfacer').click()
    await agarda(); await agarda()
    expect(api.DesfacerTraslado).toHaveBeenCalledWith('v1')
  })

  it('soltar un ficheiro coa xanela aberta lévao a Meter', async () => {
    const api = apiFalsa()
    abrirTrasladar({ t, api, contexto, confirmar: async () => true })
    abrirTrasladar({ t, api, contexto, ruta: '/tmp/soltado.lua' })
    await agarda(); await agarda()
    expect(api.AnalizarMeter).toHaveBeenCalledWith('/tmp/soltado.lua')
    expect(document.querySelectorAll('.tras').length).toBe(1)
  })

  it('Escape pecha a xanela', () => {
    abrirTrasladar({ t, api: apiFalsa(), contexto, confirmar: async () => true })
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    expect(trasladarAberta()).toBe(false)
    expect(document.querySelector('.tras')).toBe(null)
  })
})
