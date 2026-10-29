import { describe, it, expect } from 'vitest'
import { formulasDe, formulaEn, texAEditor, editorATex, envolver, substituir, variablesDoEditor, nomeValido } from './formulas-modelo.js'

describe('formulasDe', () => {
  it('atopa as fórmulas en liña e centradas coas súas posicións', () => {
    const md = 'Forza $F = m\\,a$ e\n\n$$E = mc^2$$'
    const fs = formulasDe(md)
    expect(fs).toHaveLength(2)
    expect(fs[0]).toMatchObject({ tex: 'F = m\\,a', centrada: false })
    expect(md.slice(fs[0].ini, fs[0].fin)).toBe('$F = m\\,a$')
    expect(fs[1]).toMatchObject({ tex: 'E = mc^2', centrada: true })
  })

  it('non conta os \\$ escapados nin os debuxos nin os prezos', () => {
    expect(formulasDe('Custa 5\\$ e $x$')).toEqual([expect.objectContaining({ tex: 'x' })])
    expect(formulasDe('```tikz\n\\node at (0,0) {$a$};\n```\n$b$').map((f) => f.tex)).toEqual(['b'])
    expect(formulasDe('5$ ou 6$')).toEqual([])
  })

  it('le as variables con espazo nas beiras', () => {
    expect(formulasDe('Masa $ {m} $ kg')[0].tex).toBe('{m}')
  })
})

describe('formulaEn', () => {
  it('devolve a fórmula do cursor, delimitadores incluídos', () => {
    const md = 'a $x+1$ b'
    expect(formulaEn(md, 2)?.tex).toBe('x+1')
    expect(formulaEn(md, 5)?.tex).toBe('x+1')
    expect(formulaEn(md, 7)?.tex).toBe('x+1')
    expect(formulaEn(md, 0)).toBeNull()
    expect(formulaEn(md, 9)).toBeNull()
  })
})

describe('variables entre o enunciado e o editor', () => {
  it('as chaves con espazo son variables e as pegadas son LaTeX', () => {
    expect(texAEditor('\\dfrac{ {m} }{ {V} } = \\dfrac{m}{V}')).toBe('\\dfrac{ \\variable{m} }{ \\variable{V} } = \\dfrac{m}{V}')
    expect(texAEditor('{m}\\cdot {a}')).toBe('\\variable{m}\\cdot \\variable{a}')
    expect(texAEditor('x^{n} + x_{i}')).toBe('x^{n} + x_{i}')
  })

  it('cada \\variable volve ser {x} con espazo', () => {
    expect(editorATex('F=\\variable{m}\\cdot\\variable{a}')).toBe('F= {m} \\cdot {a}')
    expect(editorATex('\\frac{\\variable{m}}{\\variable{V}}')).toBe('\\frac{ {m} }{ {V} }')
    expect(editorATex('x^{\\variable{n}}')).toBe('x^{ {n} }')
  })

  it('ida e volta conserva as variables (os espazos non contan en LaTeX)', () => {
    const sen = (s) => s.replace(/\s+/g, '')
    for (const tex of ['F = {m}\\cdot {a}', '\\dfrac{ {m} }{ {V} }', 'x^{ {n} } + \\sqrt{ {d} }']) {
      expect(sen(texAEditor(editorATex(texAEditor(tex))))).toBe(sen(texAEditor(tex)))
      expect(variablesDoEditor(texAEditor(tex)).length).toBeGreaterThan(0)
    }
  })

  it('variablesDoEditor non repite', () => {
    expect(variablesDoEditor('\\variable{m}+\\variable{m}\\cdot\\variable{g}')).toEqual(['m', 'g'])
  })
})

describe('envolver e substituir', () => {
  it('pon os delimitadores e o espazo xunto ás variables das beiras', () => {
    expect(envolver('x+1', false)).toBe('$x+1$')
    expect(envolver('{m}', false)).toBe('$ {m} $')
    expect(envolver('F = {m}', false)).toBe('$F = {m} $')
    expect(envolver('\\frac12\\sqrt{x}', false)).toBe('$\\frac12\\sqrt{x}$')
    expect(envolver('E=mc^2', true)).toBe('$$E=mc^2$$')
    expect(envolver('  ', false)).toBe('')
  })

  it('mete a fórmula no cursor separada das palabras', () => {
    expect(substituir('Forza:fin', '$F$', { ini: 6, fin: 6 })).toEqual({ texto: 'Forza: $F$ fin', cursor: 11 })
    expect(substituir('a $x$.', '$y$', { ini: 2, fin: 5 }).texto).toBe('a $y$.')
  })

  it('unha fórmula centrada vai no seu parágrafo', () => {
    expect(substituir('Calcula a forza. Despois', '$$F=ma$$', { ini: 16, fin: 16, centrada: true }).texto)
      .toBe('Calcula a forza.\n\n$$F=ma$$\n\nDespois')
  })

  it('valida os nomes das variables', () => {
    expect(nomeValido('m_1')).toBe(true)
    expect(nomeValido('1m')).toBe(false)
    expect(nomeValido('a b')).toBe(false)
  })
})
