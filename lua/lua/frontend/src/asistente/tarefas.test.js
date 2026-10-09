import { describe, it, expect } from 'vitest'
import { TAREFAS_BASE, lerPropias, engadirPropia, eliminarPropia, instrucionFinal } from './tarefas.js'
import { logoProvedor, nomeProvedor } from './logos.js'

function almacenFalso() {
  const d = {}
  return { getItem: (k) => (k in d ? d[k] : null), setItem: (k, v) => { d[k] = v } }
}

describe('fichas de tarefas', () => {
  it('as de fábrica cobren o que se pediu e teñen instrución e clave i18n', () => {
    const ids = TAREFAS_BASE.map((t) => t.id)
    for (const necesaria of ['ortografia', 'redaccion', 'introducion', 'lexislacion', 'formato', 'plantilla']) expect(ids).toContain(necesaria)
    for (const t of TAREFAS_BASE) {
      expect(t.instrucion.length).toBeGreaterThan(40)
      expect(t.clave).toBe('asistente.tarefa.' + t.id)
    }
    expect(new Set(ids).size).toBe(ids.length)
  })
  it('as propias engádense, persisten, e bórranse', () => {
    const a = almacenFalso()
    expect(lerPropias(a)).toEqual([])
    engadirPropia('  Resumir ', ' Resume cada tarefa ', a)
    engadirPropia('Traducir', 'Traduce ao inglés', a)
    const l = lerPropias(a)
    expect(l.map((x) => x.titulo)).toEqual(['Resumir', 'Traducir'])
    expect(l[0]).toMatchObject({ instrucion: 'Resume cada tarefa', propia: true })
    const restante = eliminarPropia(l[0].id, a)
    expect(restante.map((x) => x.titulo)).toEqual(['Traducir'])
    expect(lerPropias(a)).toHaveLength(1)
  })
  it('esixe nome e instrución, e tolera un almacén roto', () => {
    const a = almacenFalso()
    expect(() => engadirPropia('', 'x', a)).toThrow()
    expect(() => engadirPropia('x', '  ', a)).toThrow()
    a.setItem('lua.asistente.tarefas', '{isto non é json')
    expect(lerPropias(a)).toEqual([])
    a.setItem('lua.asistente.tarefas', JSON.stringify([{ titulo: 'ok', instrucion: 'ok' }, { titulo: '', instrucion: 'x' }, 5, null]))
    expect(lerPropias(a)).toHaveLength(1)
  })
  it('instrucionFinal engade o texto da caixa coma instrución adicional', () => {
    const t = { instrucion: 'Faino.' }
    expect(instrucionFinal(t, '')).toBe('Faino.')
    expect(instrucionFinal(t, '  só maio  ')).toBe('Faino.\n\nInstrucións adicionais do usuario: só maio')
  })
})

describe('logotipos', () => {
  it('cada provedor ten un SVG e os descoñecidos usan as faíscas xenéricas', () => {
    for (const p of ['gemini', 'anthropic', 'openai', 'qwen', 'perplexity', 'outro', '']) {
      const svg = logoProvedor(p)
      expect(svg.startsWith('<svg')).toBe(true)
      expect(svg).toContain('</svg>')
    }
    expect(logoProvedor('gemini')).not.toBe(logoProvedor('anthropic'))
    expect(logoProvedor('outro')).toContain('linearGradient')
  })
  it('os ids dos degradados non se repiten (dous logos na mesma páxina)', () => {
    const id = (s) => /id="([^"]+)"/.exec(s)[1]
    expect(id(logoProvedor('gemini'))).not.toBe(id(logoProvedor('gemini')))
  })
  it('nomes amosables', () => {
    expect(nomeProvedor('anthropic')).toBe('Claude')
    expect(nomeProvedor('openai')).toBe('ChatGPT')
    expect(nomeProvedor('outro')).toBe('IA')
  })
})
