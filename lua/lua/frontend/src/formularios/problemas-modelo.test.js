import { describe, it, expect } from 'vitest'
import {
  variablesUsadas, sincronizarVariables, numerosAVariables, cambiarModo, paraMotor, listaCalculo,
  solucionDesactualizada, xeracionDe, partesMates,
} from './problemas-modelo.js'

describe('problemas-modelo', () => {
  it('distingue variables de grupos de LaTeX', () => {
    expect(variablesUsadas('Masa {m} kg; $\\dfrac{m}{V} = \\dfrac{ {m} }{ {V} }$, $x^{n}$ e $ {n} $; {{literal}}'))
      .toEqual(['m', 'V', 'n'])
    expect(variablesUsadas("Calcula {diff(f,x)} e {'f}")).toEqual([])
    expect(variablesUsadas('Circuíto:\n\n```circuitikz\n\\draw (0,0) to[R, l=$R_1 = {R1}\\,\\mathrm{V}$] (2,0);\n\\node at (1,1) {texto};\n```')).toEqual(['R1'])
  })

  it('non declara as unidades coma variables', () => {
    expect(sincronizarVariables({ enunciado: 'Circula $I = {I}\\,{mA}$ e $\\mathrm{kW}$', variables: [] }).variables).toEqual([{ nome: 'I', modo: 'fixa', valor: '1' }])
  })

  it('declara as variables novas sen tocar as que hai', () => {
    const p = sincronizarVariables({ enunciado: '{a} e {b}', variables: [{ nome: 'a', modo: 'variable', de: '1', ata: '3' }] })
    expect(p.variables).toEqual([{ nome: 'a', modo: 'variable', de: '1', ata: '3' }, { nome: 'b', modo: 'fixa', valor: '1' }])
  })

  it('converte os números das fórmulas en variables, non os expoñentes', () => {
    const { problema, creadas } = numerosAVariables({ enunciado: 'Resolve $2x^2 - 3,5x + 1 = 0$ (ex. 1)', variables: [] })
    expect(creadas).toBe(4)
    expect(problema.enunciado).toContain('x^2')
    expect(problema.enunciado).toContain('(ex. 1)')
    expect(problema.variables.map((v) => v.valor)).toEqual(['2', '3,5', '1', '0'])
    expect(variablesUsadas(problema.enunciado)).toEqual(problema.variables.map((v) => v.nome))
  })

  it('non toca os números dos debuxos', () => {
    const c = '```circuitikz\n\\draw (0,0) to[R, l=$4\\,\\Omega$] (2,0);\n```'
    const { problema, creadas } = numerosAVariables({ enunciado: 'Con $V = 9$:\n\n' + c, variables: [] })
    expect(creadas).toBe(1)
    expect(problema.enunciado).toContain(c)
  })

  it('cambia o modo conservando o valor visible', () => {
    const v = cambiarModo({ nome: 'a', modo: 'fixa', valor: '6' })
    expect(v).toMatchObject({ modo: 'variable', de: '3', ata: '12' })
    expect(cambiarModo(v, '7')).toEqual({ nome: 'a', modo: 'fixa', valor: '7' })
  })

  it('pasa comas a puntos para Go', () => {
    expect(paraMotor({ variables: [{ nome: 'k', modo: 'variable', de: '0,5', ata: '2', paso: '0,1' }] }).variables[0]).toEqual({ nome: 'k', modo: 'variable', de: '0.5', ata: '2', paso: '0.1' })
  })

  it('numera as preguntas de todos os campos na orde do esquema', () => {
    const esq = { campos: [{ id: 'b', tipo: 'problemas' }, { id: 'x', tipo: 'texto' }, { id: 'a', tipo: 'problemas' }] }
    const l = listaCalculo(esq, { a: [{ id: '1' }], b: [{ id: '2' }, { id: '3' }] })
    expect(l.map((e) => [e.campo, e.indice, e.problema.id])).toEqual([['b', 0, '2'], ['b', 1, '3'], ['a', 2, '1']])
  })

  it('detecta solucións non paramétricas vellas', () => {
    const p = { solucion: { texto: 'x', parametrica: false, valores: { a: '2' } } }
    expect(solucionDesactualizada(p, { valores: { a: { texto: '3' } } })).toBe(true)
    expect(solucionDesactualizada(p, { valores: { a: { texto: '2' } } })).toBe(false)
  })

  it('xeración por defecto', () => {
    expect(xeracionDe({})).toEqual({ semente: 1, variantes: 1, solucions: 'non' })
    expect(xeracionDe({ xeracion: { semente: 7, variantes: 99, solucions: 'final' } })).toEqual({ semente: 7, variantes: 50, solucions: 'final' })
    expect(xeracionDe({ xeracion: { solucions: 'resultado-xunto' } }).solucions).toBe('resultado-xunto')
    expect(xeracionDe({ xeracion: { solucions: 'resultado-final' } }).solucions).toBe('resultado-final')
    expect(xeracionDe({ xeracion: { solucions: 'inventada' } }).solucions).toBe('non')
    expect(partesMates('a $x$ b').map((p) => p.mates)).toEqual([false, true, false])
  })
})
