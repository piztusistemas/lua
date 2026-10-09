// @vitest-environment jsdom
import { describe, it, expect, vi, afterEach } from 'vitest'
import { createFormularioEditor } from './editor.js'

afterEach(() => { document.body.replaceChildren() })
const agardar = (ms = 420) => new Promise((r) => setTimeout(r, ms))

const esquema = { id: 'exame', nome: 'Exame', modo: 'deseno', campos: [{ id: 'preg', tipo: 'problemas', etiqueta: 'Preguntas' }] }

function montar(valores) {
  const container = document.createElement('div')
  document.body.append(container)
  let plan = { formulario: 'exame', contexto: {}, valores }
  const calcular = vi.fn(async (p) => ({
    maxima: true,
    calculados: p.problemas.map(({ problema }) => ({
      enunciado: problema.enunciado.replace('{m}', String(p.semente * 10)),
      solucion: problema.solucion?.texto?.replace('{F}', '99') || '',
      resultado: '',
      valores: { m: { texto: String(p.semente * 10), tex: String(p.semente * 10) } },
    })),
  }))
  const resolver = vi.fn(async () => ({
    solucion: { texto: 'Forza $F = {F}$', calculos: [{ nome: 'F', expr: 'm*a' }], orixe: 'ia+maxima', parametrica: true },
    calculado: { enunciado: 'x', solucion: 'Forza $F = 99$', resultado: '', valores: {} },
    rexistro: ['Ronda 1: Maxima comprobou a solución en 5 semente(s) sen fallos.'],
  }))
  const editor = createFormularioEditor(container, {
    onChange: (p) => { plan = p },
    obterEsquema: () => esquema,
    problemas: { api: { CalcularProblemas: calcular, ResolverProblema: resolver, IAAsistenteEstado: async () => ({ configurada: true }) }, idioma: () => 'gl' },
  })
  editor.setValue(plan)
  return { container, editor, calcular, resolver, plan: () => plan }
}

describe('campo problemas', () => {
  it('calcula, pinta as fórmulas e recalcula ao facer variable unha variable', async () => {
    const m = montar({ preg: [{ id: 'a', enunciado: 'Masa {m} kg e $F = m\\,a$', variables: [{ nome: 'm', modo: 'fixa', valor: '2' }] }] })
    await agardar()
    expect(m.calcular).toHaveBeenCalledTimes(1)
    const vista = m.container.querySelector('[data-vista="enunciado:a"]')
    expect(vista.textContent).toContain('Masa 10 kg')
    expect(vista.querySelector('.katex')).not.toBeNull()
    // Fixa -> variable: cambia o plan e volve calcular.
    const botonVariable = [...m.container.querySelectorAll('.prob-seg__btn')].find((b) => b.textContent === 'Variable')
    botonVariable.click()
    expect(m.plan().valores.preg[0].variables[0]).toMatchObject({ modo: 'variable', de: '1', ata: '4' })
    await agardar()
    expect(m.calcular).toHaveBeenCalledTimes(2)
    // A semente (barra de xeración) tamén recalcula.
    m.editor.setValue({ ...m.plan(), xeracion: { semente: 3 } })
    await agardar()
    expect(m.calcular).toHaveBeenLastCalledWith(expect.objectContaining({ semente: 3 }))
    expect(m.container.querySelector('[data-vista="enunciado:a"]').textContent).toContain('Masa 30 kg')
  })

  it('declara as variables novas que se escriben no enunciado', async () => {
    const m = montar({ preg: [{ id: 'a', enunciado: '', variables: [] }] })
    await agardar()
    const area = m.container.querySelector('[data-foco="enunciado:a"]')
    area.value = 'Un corpo de {m} kg'
    area.dispatchEvent(new Event('input'))
    await agardar()
    expect(m.plan().valores.preg[0].variables).toEqual([{ nome: 'm', modo: 'fixa', valor: '1' }])
    expect(m.container.querySelector('[data-variable="m"]')).not.toBeNull()
  })

  it('a solución só se enche cando se pide, e entra no documento', async () => {
    const m = montar({ preg: [{ id: 'a', enunciado: 'Masa {m}', variables: [{ nome: 'm', modo: 'fixa', valor: '2' }] }] })
    await agardar()
    expect(m.container.querySelector('.prob-sol--baleira')).not.toBeNull()
    m.container.querySelector('.prob-sol [data-accion="resolver"]').click()
    await agardar(30)
    expect(m.resolver).toHaveBeenCalledWith(expect.objectContaining({ modo: 'maxima', indice: 0, problema: expect.objectContaining({ enunciado: 'Masa {m}' }) }))
    expect(m.plan().valores.preg[0].solucion).toMatchObject({ orixe: 'ia+maxima', parametrica: true })
    expect(m.container.querySelector('[data-vista="solucion:a"]').textContent).toContain('Forza')
    expect(m.container.querySelector('.prob-rexistro').textContent).toContain('Maxima comprobou')
  })
})

describe('resolver o exame', () => {
  it('o botón da cabeceira resolve as preguntas de todos os campos', async () => {
    const container = document.createElement('div')
    document.body.append(container)
    const esq = { id: 'ex', nome: 'Exame', modo: 'deseno', campos: [{ id: 'p1', tipo: 'problemas', etiqueta: 'P1' }, { id: 'p2', tipo: 'problemas', etiqueta: 'P2' }] }
    let plan = { formulario: 'ex', contexto: {}, valores: { p1: [{ id: 'a', enunciado: 'Uno' }], p2: [{ id: 'b', enunciado: 'Dous' }, { id: 'c', enunciado: 'Tres', solucion: { texto: 'xa' } }] } }
    const resolver = vi.fn(async (p) => ({ solucion: { texto: 'Sol ' + p.problema.enunciado, parametrica: true }, rexistro: [] }))
    const ed = createFormularioEditor(container, {
      onChange: (p) => { plan = p }, obterEsquema: () => esq,
      problemas: { api: { CalcularProblemas: async (p) => ({ maxima: true, calculados: p.problemas.map(() => ({ enunciado: '', solucion: '', valores: {} })) }), ResolverProblema: resolver, IAAsistenteEstado: async () => ({ configurada: true }) } },
    })
    ed.setValue(plan)
    await agardar(50)
    const boton = container.querySelector('[data-accion="resolver-exame"]')
    expect(boton.textContent).toContain('Resolver o exame (2)')
    // Cada campo di «Resolver problema», non «Resolver o exame».
    expect([...container.querySelectorAll('[data-accion="resolver-campo"]')].map((b) => b.textContent)).toEqual(['Resolver problema', 'Resolver problema'])
    boton.click()
    await agardar(80)
    expect(resolver).toHaveBeenCalledTimes(2)
    expect(plan.valores.p1[0].solucion.texto).toBe('Sol Uno')
    expect(plan.valores.p2[0].solucion.texto).toBe('Sol Dous')
    expect(container.querySelector('[data-accion="resolver-exame"]').disabled).toBe(true)
  })
})
