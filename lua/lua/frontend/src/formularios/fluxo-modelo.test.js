import { describe, it, expect } from 'vitest'
import {
  aceptaTipo, portosDe, novoNodo, conectar, desconectar, eliminarNodo, moverNoDocumento, poñerConfig,
  nodosPorColumna, creaCiclo, idNodoLibre, fluxoIgual, ordePorto, posicionInicial, profundidades,
} from './fluxo-modelo.js'

const CAT = [
  { tipo: 'campo', nome: 'Campo', categoria: 'orixe' },
  { tipo: 'constante', nome: 'Valor fixo', categoria: 'orixe', config: [{ clave: 'tipo', defecto: 'numero' }] },
  { tipo: 'calcular', nome: 'Calcular', categoria: 'transformacion', saida: 'numero', dinamicas: ['a', 'b', 'c'], dinTipo: 'numero', config: [{ clave: 'formula', defecto: 'a' }, { clave: 'decimais', defecto: 2 }] },
  { tipo: 'agregar', nome: 'Resumir', categoria: 'transformacion', saida: 'numero', entradas: [{ nome: 'taboa', tipo: 'taboa' }] },
  { tipo: 'paragrafo', nome: 'Texto', categoria: 'destino', entradas: [{ nome: 'texto', tipo: 'texto' }, { nome: 'se', tipo: 'bool', opcional: true }] },
  { tipo: 'titulo', nome: 'Título', categoria: 'destino' },
]
const base = () => ({
  nodos: [
    { id: 'c_a', tipo: 'campo', campo: 'a' },
    { id: 'c_t', tipo: 'campo', campo: 't' },
  ],
  documento: [],
})

describe('tipos', () => {
  it('aceptaTipo espella a regra de Go', () => {
    expect(aceptaTipo('numero', 'numero')).toBe(true)
    expect(aceptaTipo('numero', 'texto')).toBe(false)
    expect(aceptaTipo('texto', 'numero')).toBe(true)
    expect(aceptaTipo('texto', 'taboa')).toBe(false)
    expect(aceptaTipo('calquera', 'bool')).toBe(true)
    expect(aceptaTipo('calquera', 'imaxe')).toBe(false)
    expect(aceptaTipo('taboa', 'taboa')).toBe(true)
    expect(aceptaTipo('numero', undefined)).toBe(true) // aínda non se avaliou
  })
})

describe('portos', () => {
  it('un nodo dinámico ensina os ligados e un libre', () => {
    const cat = CAT[2]
    expect(portosDe({ id: 'x', entradas: {} }, cat).map((p) => p.nome)).toEqual(['a'])
    expect(portosDe({ id: 'x', entradas: { a: 'c_a' } }, cat).map((p) => [p.nome, p.ligado])).toEqual([['a', true], ['b', false]])
    expect(portosDe({ id: 'x', entradas: { a: 'c_a', b: 'c_a', c: 'c_a' } }, cat).map((p) => p.nome)).toEqual(['a', 'b', 'c']) // sen máis libres
  })
  it('un nodo con portos fixos non ten dinámicos', () => {
    expect(portosDe({ entradas: {} }, CAT[4]).map((p) => p.nome)).toEqual(['texto', 'se'])
  })
  it('ordePorto compara polo número', () => {
    expect(['v10', 'v2', 'v1'].sort((a, b) => ordePorto(a) - ordePorto(b))).toEqual(['v1', 'v2', 'v10'])
  })
})

describe('editar o grafo', () => {
  it('novoNodo pon ids únicos, os defectos e os destinos no documento', () => {
    const orixinal = base()
    let f = orixinal
    let r = novoNodo(f, CAT, 'calcular')
    expect(r.id).toBe('n_calcular')
    expect(r.fluxo.nodos.find((n) => n.id === r.id).config).toEqual({ formula: 'a', decimais: 2 })
    expect(r.fluxo.documento).toEqual([])
    f = r.fluxo
    r = novoNodo(f, CAT, 'calcular')
    expect(r.id).toBe('n_calcular_2')
    r = novoNodo(r.fluxo, CAT, 'paragrafo')
    expect(r.id).toBe('d_paragrafo')
    expect(r.fluxo.documento).toEqual(['d_paragrafo'])
    expect(() => novoNodo(f, CAT, 'raro')).toThrow()
    expect(orixinal.nodos.length).toBe(2) // inmutable
  })

  it('conectar valida tipos, portos e ciclos', () => {
    let f = novoNodo(base(), CAT, 'calcular').fluxo
    f = novoNodo(f, CAT, 'paragrafo').fluxo
    expect(conectar(f, CAT, 'c_a', 'n_calcular', 'a', 'numero').fluxo.nodos.find((n) => n.id === 'n_calcular').entradas).toEqual({ a: 'c_a' })
    expect(conectar(f, CAT, 'c_t', 'n_calcular', 'a', 'texto').erro).toBe('tipo')
    expect(conectar(f, CAT, 'c_a', 'n_calcular', 'z', 'numero').erro).toBe('porto')
    expect(conectar(f, CAT, 'd_paragrafo', 'n_calcular', 'a', 'texto').erro).toBe('destino')
    expect(conectar(f, CAT, 'nada', 'n_calcular', 'a', 'numero').erro).toBe('nodo')
    // texto acepta un número
    expect(conectar(f, CAT, 'n_calcular', 'd_paragrafo', 'texto', 'numero').fluxo).toBeTruthy()
    // ciclo: calcular -> calcular2 -> calcular
    let g = novoNodo(f, CAT, 'calcular').fluxo // n_calcular_2
    g = conectar(g, CAT, 'n_calcular', 'n_calcular_2', 'a', 'numero').fluxo
    expect(creaCiclo(g, 'n_calcular_2', 'n_calcular')).toBe(true)
    expect(conectar(g, CAT, 'n_calcular_2', 'n_calcular', 'a', 'numero').erro).toBe('ciclo')
    expect(conectar(g, CAT, 'n_calcular', 'n_calcular', 'a', 'numero').erro).toBe('ciclo') // a si mesmo
  })

  it('conectar substitúe a ligazón do porto e desconectar quítaa', () => {
    let f = novoNodo(base(), CAT, 'calcular').fluxo
    f = conectar(f, CAT, 'c_a', 'n_calcular', 'a', 'numero').fluxo
    f = conectar(f, CAT, 'c_a', 'n_calcular', 'a', 'numero').fluxo
    f = desconectar(f, 'n_calcular', 'a')
    expect(nodoDe(f, 'n_calcular').entradas).toEqual({})
  })

  it('eliminarNodo limpa as ligazóns que saían del e o documento', () => {
    let f = novoNodo(base(), CAT, 'calcular').fluxo
    f = novoNodo(f, CAT, 'paragrafo').fluxo
    f = conectar(f, CAT, 'n_calcular', 'd_paragrafo', 'texto', 'numero').fluxo
    const sen = eliminarNodo(f, 'n_calcular')
    expect(sen.nodos.map((n) => n.id)).toEqual(['c_a', 'c_t', 'd_paragrafo'])
    expect(nodoDe(sen, 'd_paragrafo').entradas).toEqual({})
    const senD = eliminarNodo(f, 'd_paragrafo')
    expect(senD.documento).toEqual([])
  })

  it('moverNoDocumento e poñerConfig', () => {
    let f = novoNodo(base(), CAT, 'titulo').fluxo
    f = novoNodo(f, CAT, 'paragrafo').fluxo
    expect(f.documento).toEqual(['d_titulo', 'd_paragrafo'])
    expect(moverNoDocumento(f, 'd_paragrafo', -1).documento).toEqual(['d_paragrafo', 'd_titulo'])
    expect(moverNoDocumento(f, 'd_titulo', -1)).toBe(f) // xa é o primeiro
    expect(poñerConfig(f, 'd_titulo', 'texto', 'Oi').nodos.find((n) => n.id === 'd_titulo').config.texto).toBe('Oi')
  })
})

describe('profundidades', () => {
  it('conta os saltos desde os campos e tolera ciclos', () => {
    const f = {
      nodos: [
        { id: 'c_a', tipo: 'campo', campo: 'a' },
        { id: 'x', tipo: 'calcular', entradas: { a: 'c_a' } },
        { id: 'y', tipo: 'calcular', entradas: { a: 'x', b: 'c_a' } },
        { id: 'z', tipo: 'calcular', entradas: { a: 'y', b: 'x' } },
        { id: 'k', tipo: 'constante' },
        { id: 'p', tipo: 'calcular', entradas: { a: 'q' } },
        { id: 'q', tipo: 'calcular', entradas: { a: 'p' } },
      ],
      documento: [],
    }
    const p = profundidades(f)
    expect([p.get('c_a'), p.get('x'), p.get('y'), p.get('z'), p.get('k')]).toEqual([0, 1, 2, 3, 1])
    expect(p.get('p')).toBeGreaterThanOrEqual(0) // non se colga cun ciclo
  })
})

describe('columnas do editor', () => {
  it('reparte datos, medio e documento (con orde) e non pisa posicións', () => {
    let f = novoNodo(base(), CAT, 'calcular').fluxo
    f = novoNodo(f, CAT, 'titulo').fluxo
    f = novoNodo(f, CAT, 'paragrafo').fluxo
    f = { ...f, documento: ['d_paragrafo', 'd_titulo'] }
    const { datos, medio, documento } = nodosPorColumna(f, CAT)
    expect(datos.map((n) => n.id)).toEqual(['c_a', 'c_t'])
    expect(medio.map((n) => n.id)).toEqual(['n_calcular'])
    expect(documento.map((n) => n.id)).toEqual(['d_paragrafo', 'd_titulo'])
    // Un destino que non estea no documento aparece igualmente (ao final).
    expect(nodosPorColumna({ ...f, documento: [] }, CAT).documento.length).toBe(2)
    const p1 = posicionInicial(f, CAT)
    const p2 = posicionInicial(novoNodo(f, CAT, 'calcular').fluxo, CAT)
    expect(p1).not.toEqual(p2)
  })
  it('fluxoIgual ignora as posicións', () => {
    const f = base()
    const g = { ...f, nodos: f.nodos.map((n) => ({ ...n, x: 5, y: 9 })) }
    expect(fluxoIgual(f, g)).toBe(true)
    expect(fluxoIgual(f, { ...f, documento: ['x'] })).toBe(false)
    expect(idNodoLibre(f, 'c_a')).toBe('c_a_2')
  })
})

function nodoDe(f, id) { return f.nodos.find((n) => n.id === id) }
