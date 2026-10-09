import { describe, it, expect } from 'vitest'
import { textoEstado, textoAxudaURLs, resumoProgreso, fraccionTarefa } from './util.js'

describe('util', () => {
  it('pide configurar a IA', () => {
    expect(textoEstado({ iaConfigurada: false, fontes: 3 })).toMatch(/clave de IA/)
  })
  it('pide PDF se non hai fontes', () => {
    expect(textoEstado({ iaConfigurada: true, fontes: 0 })).toMatch(/engade/)
  })
  it('axuda das URL segundo a IA', () => {
    expect(textoAxudaURLs({ iaConfigurada: true })).toMatch(/IA busca/)
    expect(textoAxudaURLs({ iaConfigurada: false })).toMatch(/Sen IA/)
  })
  it('o progreso total suma o camiño de cada tarefa', () => {
    const r = resumoProgreso([
      { estado: 'feita', pasos: 3 }, { estado: 'erro', pasos: 1 }, { estado: 'executando', pasos: 6 }, { estado: 'pendente', pasos: 0 },
    ])
    expect(r).toEqual({ total: 4, rematadas: 2, erros: 1, pct: Math.round(((1 + 1 + 0.5 + 0) / 4) * 100) })
    expect(resumoProgreso([])).toEqual({ total: 0, rematadas: 0, erros: 0, pct: 0 })
  })
  it('unha tarefa en curso nunca chega ao 100 %', () => {
    expect(fraccionTarefa({ estado: 'executando', pasos: 500 })).toBeLessThan(1)
    expect(fraccionTarefa({ estado: 'executando', pasos: 0 })).toBe(0)
  })
})
