import { describe, it, expect } from 'vitest'
import {
  aplicarBarra, variantes, validarParche, aplicarParche, elementoEnPunto, limpar, normalizarCor, valor, contrasteBranco,
  caixaMarca, parteDeCampo, paxinaRepresentativa,
} from './modelo.js'

describe('barras', () => {
  it('formal cambia letra, escala e etiquetas, e volve ao de Lúa no medio', () => {
    const f = aplicarBarra({}, 'formal', 2)
    expect(f.letra).toBe('elegante')
    expect(f.etiquetas).toBe('contorno')
    expect(f.barras.formal).toBe(2)
    const medio = aplicarBarra(f, 'formal', 0)
    expect(medio.letra).toBeUndefined()
    expect(medio.etiquetas).toBeUndefined()
  })

  it('respecta os campos fixados', () => {
    const d = aplicarBarra({ letra: 'lexible', fixados: ['letra'] }, 'formal', -2)
    expect(d.letra).toBe('lexible')
    expect(d.escala).toBe(1.15)
  })

  it('aire move interliñado, parágrafo e marxe', () => {
    const d = aplicarBarra({}, 'aire', -2)
    expect([d.interlinado, d.parrafo, d.marxe]).toEqual([0.95, 0.2, 13])
  })

  it('a barra da cor lembra a cor de partida', () => {
    const vivo = aplicarBarra({ corPrincipal: '0E7490' }, 'cor', 2)
    expect(vivo.corPrincipal).not.toBe('0E7490')
    const sobrio = aplicarBarra(vivo, 'cor', -2)
    expect(sobrio.corPrincipal).toBe('334155')
    const volta = aplicarBarra(sobrio, 'cor', 0)
    expect(volta.corPrincipal).toBe('0E7490')
    expect(volta.barras?.corBase).toBeUndefined()
    // Sen cor de partida, volve á de Lúa.
    expect(aplicarBarra(aplicarBarra({}, 'cor', 1), 'cor', 0).corPrincipal).toBeUndefined()
  })
})

describe('variantes', () => {
  it('cada rolda explora unha dimensión e as dúas propostas só se distinguen nela', () => {
    const v = variantes({ tamano: 12 }, 0)
    expect(v.dimension).toBe('letra')
    expect(v.a.letra).not.toBe(v.b.letra)
    expect(v.a.tamano).toBe(12)
    const c = variantes({}, 1, ['#dc2626'])
    expect(c.dimension).toBe('cor')
    expect(c.a.corPrincipal).toBe('DC2626')
  })

  it('salta o fixado e devolve null se todo o está', () => {
    expect(variantes({ fixados: ['letra'] }, 0).dimension).toBe('cor')
    expect(variantes({ fixados: ['letra', 'corPrincipal', 'interlinado', 'etiquetas', 'tamano'] }, 0)).toBeNull()
  })
})

describe('IA', () => {
  it('valida rangos, cores e elementos', () => {
    expect(() => validarParche({ tamano: 40 })).toThrow(/fóra de rango/)
    expect(() => validarParche({ letra: 'comic' })).toThrow(/non existe/)
    expect(() => validarParche({ corPrincipal: 'verde' })).toThrow(/cor non válida/)
    expect(() => validarParche({ elementos: { pe: {} } })).toThrow(/non existe/)
    const { parche, avisos } = validarParche({ corPrincipal: '#abc', inventado: 1, elementos: { titulos: { maiusculas: 1, cor: '0E7490' } } })
    expect(parche.corPrincipal).toBe('AABBCC')
    expect(parche.elementos.titulos).toEqual({ maiusculas: true, cor: '0E7490' })
    expect(avisos.join()).toMatch(/inventado/)
  })

  it('non toca o fixado', () => {
    const { parche, avisos } = validarParche({ letra: 'moderna' }, { fixados: ['letra'] })
    expect(parche.letra).toBeUndefined()
    expect(avisos[0]).toMatch(/fixado/)
  })

  it('mestura os elementos campo a campo e null quita un elemento', () => {
    const d = aplicarParche({ elementos: { titulos: { cor: '111111' }, tarefas: { escala: 1.2 } } },
      { tamano: 12, elementos: { titulos: { maiusculas: true }, tarefas: null } })
    expect(d).toEqual({ tamano: 12, elementos: { titulos: { cor: '111111', maiusculas: true } } })
  })
})

describe('inspector', () => {
  const marcas = [
    { elemento: 'titulos', paxina: 1, x: 0.12, y: 0.1 },
    { elemento: 'etiquetas', paxina: 1, x: 0.3, y: 0.4 },
    { elemento: 'etiquetas', paxina: 1, x: 0.3, y: 0.45 },
    { elemento: 'tarefas', paxina: 2, x: 0.12, y: 0.4 },
  ]
  it('atopa o elemento xusto debaixo da marca', () => {
    expect(elementoEnPunto(marcas, 1, 0.3, 0.11)).toBe('titulos')
    expect(elementoEnPunto(marcas, 1, 0.35, 0.452)).toBe('etiquetas')
    expect(elementoEnPunto(marcas, 2, 0.5, 0.41)).toBe('tarefas')
  })
  it('lonxe de toda marca é o texto', () => {
    expect(elementoEnPunto(marcas, 1, 0.5, 0.8)).toBe('texto')
    expect(elementoEnPunto([], 1, 0.5, 0.5)).toBe('texto')
  })
})

describe('utilidades', () => {
  it('limpa os baleiros e normaliza cores', () => {
    expect(limpar({ letra: '', tamano: 0, brancoNegro: false, elementos: { titulos: {} }, barras: { formal: 0 } })).toEqual({})
    expect(normalizarCor('#0e7')).toBe('00EE77')
    expect(normalizarCor('xx')).toBe('')
    expect(valor({}, 'tamano')).toBe(11)
    expect(contrasteBranco('000000')).toBeCloseTo(21)
  })
})

describe('melloras', () => {
  it('as barras lembran os valores de partida e volven a eles', () => {
    const inicial = { letra: 'lexible', interlinado: 1.15 }
    const formal = aplicarBarra(inicial, 'formal', 2)
    expect(formal.letra).toBe('elegante')
    const outra = aplicarBarra(formal, 'formal', 1) // segue lembrando a de partida
    expect(aplicarBarra(outra, 'formal', 0).letra).toBe('lexible')
    const aire = aplicarBarra(inicial, 'aire', 2)
    expect(aire.interlinado).toBe(1.35)
    const volta = aplicarBarra(aire, 'aire', 0)
    expect(volta.interlinado).toBe(1.15)
    expect(volta.barras?.base).toBeUndefined()
  })

  it('a caixa dunha parte vai da súa marca ata a seguinte', () => {
    const marcas = [{ elemento: 'titulos', paxina: 1, x: 0.1, y: 0.1 }, { elemento: 'etiquetas', paxina: 1, x: 0.1, y: 0.13 }]
    const c = caixaMarca(marcas, marcas[0])
    expect(c.y).toBeCloseTo(0.078)
    expect(c.y + c.h).toBeCloseTo(0.108)
    const sen = caixaMarca(marcas, marcas[1])
    expect(sen.h).toBeCloseTo(0.062)
  })

  it('cada cambio sabe que parte do documento mirar', () => {
    expect(parteDeCampo('letra')).toBe('titulos')
    expect(parteDeCampo('portada')).toBe('portada')
    expect(parteDeCampo('meses')).toBe('meses')
    expect(paxinaRepresentativa({ pageImages: [1, 2, 3], marcas: [{ paxina: 3 }, { paxina: 3 }, { paxina: 2 }] })).toBe(3)
  })
})

describe('IA: portada, cabeceira e pé', () => {
  it('valida os modelos e as zonas', () => {
    expect(() => validarParche({ portada: 'barroca' })).toThrow(/non existe/)
    expect(() => validarParche({ cabeceira: { arriba: 'x' } })).toThrow(/non existe/)
    const { parche } = validarParche({ portada: 'sobria', cabeceira: { dereita: '{{CURSO}}' } }, { cabeceira: { esquerda: '@logo' } })
    expect(parche).toEqual({ portada: 'sobria', cabeceira: { esquerda: '@logo', dereita: '{{CURSO}}' } })
  })
})

describe('elementosDe', () => {
  it('só as partes do tipo de documento, máis as xa cambiadas', async () => {
    const { elementosDe, ELEMENTOS } = await import('./modelo.js')
    expect(elementosDe('actas', {})).toEqual(['texto', 'titulos', 'actas_cabeceira', 'actas_apartados', 'actas_materias', 'sinaturas'])
    expect(elementosDe('formulario', { elementos: { meses: { cor: '111111' } } })).toEqual(['texto', 'titulos', 'meses', 'sinaturas'])
    expect(elementosDe('', {})).toEqual(ELEMENTOS)
  })
})
