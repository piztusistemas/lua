import { describe, it, expect } from 'vitest'
import {
  opcionsSacar, opcionInicial, ofreceDatos, seleccionSacar, seleccionBaleira, textoElemento,
  opcionsDestino, textoDestino, confirmacionPrevia, opcionsMeter, podeMeter,
} from './modelo.js'

const t = (_c, f, v) => { let s = f; for (const k in v || {}) s = s.replaceAll(`{${k}}`, v[k]); return s }
const ctx = { proxecto: { id: 'p1', nome: '3º ESO' }, documento: { ferramenta: 'cronograma', nome: 'Cronograma' } }

describe('Sacar', () => {
  it('preselecciona o proxecto aberto e despois o documento', () => {
    expect(opcionsSacar(ctx).map((o) => o.id)).toEqual(['proxecto', 'documento', 'todo', 'escoller'])
    expect(opcionInicial(ctx)).toBe('proxecto')
    expect(opcionInicial(ctx, 'documento')).toBe('documento')
    expect(opcionInicial({ proxecto: null, documento: null })).toBe('todo')
    expect(opcionInicial(ctx, 'inexistente')).toBe('proxecto')
  })

  it('arma a selección para Go', () => {
    expect(seleccionSacar('proxecto', ctx, null, true)).toEqual({ tipo: 'proxecto', proxectos: ['p1'], formularios: [], plantillas: [], datos: true })
    expect(seleccionSacar('todo', ctx, null, false).datos).toBe(true)
    const s = seleccionSacar('escoller', ctx, { proxectos: new Set(['a']), formularios: new Set(), plantillas: new Set(['x']) }, false)
    expect(s).toEqual({ tipo: 'seleccion', proxectos: ['a'], formularios: [], plantillas: ['x'], datos: false })
    expect(seleccionBaleira(seleccionSacar('escoller', ctx, { proxectos: new Set(), formularios: new Set(), plantillas: new Set() }))).toBe(true)
  })

  it('a marca de datos só sae onde ten sentido', () => {
    expect(ofreceDatos('proxecto')).toBe(true)
    expect(ofreceDatos('escoller')).toBe(true)
    expect(ofreceDatos('todo')).toBe(false)
    expect(ofreceDatos('documento')).toBe(false)
  })
})

describe('Meter', () => {
  const doc = { tipo: 'documento', nome: 'Cronograma', destino: 'aberto', podeAberto: true, nomeProxectoAberto: '3º ESO', cursos: [] }

  it('describe cada elemento', () => {
    expect(textoElemento({ tipo: 'proxecto', nome: 'A', detalle: '4/12' }, t)).toBe('Proxecto «A»: 4 formularios, 12 PDF na bóveda')
    expect(textoElemento({ tipo: 'boveda', nome: 'Leis', detalle: '2/3' }, t)).toBe('Fontes para a bóveda: 2 PDF e 3 Markdown')
  })

  it('só deixa escoller destino cando hai dous posibles', () => {
    expect(opcionsDestino(doc)).toEqual(['aberto', 'novo'])
    expect(opcionsDestino({ ...doc, podeAberto: false, destino: 'novo' })).toEqual([])
    expect(opcionsDestino({ ...doc, volta: { nomeProxecto: 'X' } })).toEqual([])
    expect(opcionsDestino({ ...doc, tipo: 'curso', cursos: [{}, {}] })).toEqual([])
    expect(opcionsDestino({ tipo: 'paquete', destino: 'novo' })).toEqual([])
  })

  it('di onde vai', () => {
    expect(textoDestino(doc, {}, t)).toBe('Vai ao proxecto aberto, «3º ESO».')
    expect(textoDestino(doc, { destino: 'novo' }, t)).toContain('proxecto novo')
    const volta = { ...doc, destino: 'volta', volta: { nomeProxecto: 'Orixe' } }
    expect(textoDestino(volta, {}, t)).toContain('Actualiza «Cronograma» do proxecto «Orixe»')
    expect(textoDestino(volta, { accion: 'copia' }, t)).toContain('proxecto novo')
    expect(textoDestino({ tipo: 'copia', destino: 'restaurar' }, {}, t)).toContain('Restaura TODO')
  })

  it('pregunta só por datos reais ou por restaurar', () => {
    expect(confirmacionPrevia(doc)).toBe(null)
    expect(confirmacionPrevia({ ...doc, produccion: true })).toBe('produccion')
    expect(confirmacionPrevia({ tipo: 'copia' })).toBe('restaurar')
  })

  it('arma as opcións para Go', () => {
    expect(opcionsMeter(doc, { destino: 'novo' })).toEqual({ destino: 'novo', cursos: null, accion: '' })
    expect(opcionsMeter({ ...doc, podeAberto: false }, { destino: 'aberto' }).destino).toBe('')
    const varios = { tipo: 'curso', destino: 'novo', cursos: [{}, {}, {}] }
    expect(opcionsMeter(varios, { cursos: new Set([2, 0]) }).cursos).toEqual([0, 2])
    expect(podeMeter(varios, { cursos: new Set() })).toBe(false)
    expect(opcionsMeter({ ...doc, volta: { nomeProxecto: 'X' } }, {}).accion).toBe('actualizar')
  })
})
