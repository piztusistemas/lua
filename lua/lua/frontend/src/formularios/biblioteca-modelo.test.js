import { describe, it, expect } from 'vitest'
import {
  bibliotecaBaleira as bibliotecaPorDefecto, INTEGRADAS, completar, columna, fixar, desfixar, agochar, moverACarpeta, eliminarCarpeta, renomearCarpeta,
  nomeRepetido, filtrar, contarPorCarpeta, podeBorrar, destinoDeSoltar, accionDeSoltar,
} from './biblioteca-modelo.js'

// Fixture: o cronograma e as actas activos na zona fixa (o por defecto é só na biblioteca).
const bibliotecaBaleira = () => ({ ...bibliotecaPorDefecto(), fixos: [...INTEGRADAS], agochados: [] })

const EXIST = ['cronograma', 'actas', 'form:a', 'form:b', 'form:c']

describe('columna', () => {
  it('por defecto: cronograma e actas fixos e os formularios abaixo', () => {
    expect(columna(bibliotecaBaleira(), EXIST)).toEqual({ fixas: ['cronograma', 'actas'], resto: ['form:a', 'form:b', 'form:c'] })
  })
  it('un formulario fixado sae da zona de abaixo e entra na fixa; os agochados non se ven', () => {
    let b = fixar(bibliotecaBaleira(), 'form:b')
    b = agochar(b, 'actas')
    b = agochar(b, 'form:c')
    expect(columna(b, EXIST)).toEqual({ fixas: ['cronograma', 'form:b'], resto: ['form:a'] })
  })
  it('ignora ids que xa non existen', () => {
    const b = { ...bibliotecaBaleira(), fixos: ['cronograma', 'form:fantasma', 'actas'] }
    expect(columna(b, EXIST).fixas).toEqual(['cronograma', 'actas'])
  })
})

describe('fixar / desfixar / agochar', () => {
  it('fixar insire diante de outro ou ao final, e move se xa estaba', () => {
    let b = fixar(bibliotecaBaleira(), 'form:a', 'actas')
    expect(b.fixos).toEqual(['cronograma', 'form:a', 'actas'])
    b = fixar(b, 'form:b')
    expect(b.fixos).toEqual(['cronograma', 'form:a', 'actas', 'form:b'])
    b = fixar(b, 'form:b', 'cronograma')
    expect(b.fixos).toEqual(['form:b', 'cronograma', 'form:a', 'actas'])
  })
  it('desfixar quita calquera ferramenta da zona fixa (tamén o cronograma e as actas)', () => {
    const b = fixar(bibliotecaBaleira(), 'form:a')
    expect(desfixar(b, 'form:a').fixos).toEqual(['cronograma', 'actas'])
    expect(desfixar(b, 'actas').fixos).toEqual(['cronograma', 'form:a'])
  })
  it('por defecto o cronograma e as actas só están na biblioteca (agochadas), e ao activalas saen na zona de abaixo', () => {
    const b = bibliotecaPorDefecto()
    expect(b.fixos).toEqual([])
    expect(b.agochados).toEqual(['cronograma', 'actas'])
    expect(columna(b, EXIST)).toEqual({ fixas: [], resto: ['form:a', 'form:b', 'form:c'] })
    expect(columna(agochar(b, 'actas', false), EXIST)).toEqual({ fixas: [], resto: ['actas', 'form:a', 'form:b', 'form:c'] })
  })
  it('agochar e amosar non repiten', () => {
    let b = agochar(agochar(bibliotecaBaleira(), 'form:a'), 'form:a')
    expect(b.agochados).toEqual(['form:a'])
    b = agochar(b, 'form:a', false)
    expect(b.agochados).toEqual([])
  })
  it('non muta o orixinal', () => {
    const b = bibliotecaBaleira()
    fixar(b, 'form:a'); agochar(b, 'form:a'); moverACarpeta(b, 'a', 'c1')
    expect(b).toEqual(bibliotecaBaleira())
  })
})

describe('borrar', () => {
  it('só se poden borrar os formularios que non están na zona fixa', () => {
    const b = fixar(bibliotecaBaleira(), 'form:a')
    expect(podeBorrar(b, 'form:a')).toBe(false)
    expect(podeBorrar(b, 'form:b')).toBe(true)
    expect(podeBorrar(b, 'cronograma')).toBe(false)
    expect(podeBorrar(b, 'actas')).toBe(false)
  })
})

describe('cartafoles', () => {
  const base = { ...bibliotecaBaleira(), carpetas: [{ id: 'c1', nome: 'Titoría' }, { id: 'c2', nome: 'Exames' }], carpeta: { a: 'c1', b: 'c1', c: 'c2' } }
  const forms = [{ id: 'a', nome: 'Acta de titoría', descricion: 'Reunión' }, { id: 'b', nome: 'Ficha', descricion: 'Datos do alumnado' }, { id: 'c', nome: 'Exame', descricion: '' }, { id: 'd', nome: 'Solto', descricion: 'Sen cartafol' }]
  it('mover a un cartafol e á raíz', () => {
    expect(moverACarpeta(base, 'd', 'c2').carpeta.d).toBe('c2')
    expect('a' in moverACarpeta(base, 'a', '').carpeta).toBe(false)
  })
  it('borrar un cartafol NON borra formularios: pasan á raíz', () => {
    const b = eliminarCarpeta(base, 'c1')
    expect(b.carpetas.map((c) => c.id)).toEqual(['c2'])
    expect(b.carpeta).toEqual({ c: 'c2' })
  })
  it('renomear e nomes repetidos (sen mirar maiúsculas)', () => {
    expect(renomearCarpeta(base, 'c1', 'Novo').carpetas[0].nome).toBe('Novo')
    expect(nomeRepetido(base, ' titoría ')).toBe(true)
    expect(nomeRepetido(base, 'Titoría', 'c1')).toBe(false)
    expect(nomeRepetido(base, 'Outro')).toBe(false)
  })
  it('filtrar por cartafol, sen cartafol e por texto (sen acentos)', () => {
    expect(filtrar(base, forms).length).toBe(4)
    expect(filtrar(base, forms, { carpeta: 'c1' }).map((f) => f.id)).toEqual(['a', 'b'])
    expect(filtrar(base, forms, { carpeta: '' }).map((f) => f.id)).toEqual(['d'])
    expect(filtrar(base, forms, { busca: 'TUTORIA' }).map((f) => f.id)).toEqual([]) // titoría != tutoría
    expect(filtrar(base, forms, { busca: 'titoria' }).map((f) => f.id)).toEqual(['a'])
    expect(filtrar(base, forms, { busca: 'alumnado', carpeta: 'c1' }).map((f) => f.id)).toEqual(['b'])
  })
  it('contar por cartafol (un formulario cun cartafol descoñecido conta na raíz)', () => {
    expect(contarPorCarpeta({ ...base, carpeta: { ...base.carpeta, d: 'nonexiste' } }, forms)).toEqual({ todos: 4, raiz: 1, c1: 2, c2: 1 })
  })
})

describe('arrastrar na columna', () => {
  const xeo = { dentro: true, ySeparador: 200, fixas: [{ id: 'cronograma', mediaY: 40 }, { id: 'actas', mediaY: 90 }, { id: 'form:a', mediaY: 140 }] }
  it('destino: diante do primeiro botón que queda por debaixo do punteiro, ou ao final', () => {
    expect(destinoDeSoltar(30, xeo)).toEqual({ zona: 'fixa', antesDe: 'cronograma' })
    expect(destinoDeSoltar(100, xeo)).toEqual({ zona: 'fixa', antesDe: 'form:a' })
    expect(destinoDeSoltar(180, xeo)).toEqual({ zona: 'fixa', antesDe: null })
    expect(destinoDeSoltar(250, xeo)).toEqual({ zona: 'resto' })
    expect(destinoDeSoltar(30, { ...xeo, dentro: false })).toBeNull()
  })
  it('un formulario de abaixo solto na zona fixa queda fixado onde caia', () => {
    const b = accionDeSoltar(bibliotecaBaleira(), 'form:b', { zona: 'fixa', antesDe: 'actas' })
    expect(b.fixos).toEqual(['cronograma', 'form:b', 'actas'])
  })
  it('reordenar dentro da zona fixa; soltalo onde xa está non fai nada', () => {
    const b = fixar(fixar(bibliotecaBaleira(), 'form:a'), 'form:b')
    expect(accionDeSoltar(b, 'form:b', { zona: 'fixa', antesDe: 'cronograma' }).fixos).toEqual(['form:b', 'cronograma', 'actas', 'form:a'])
    expect(accionDeSoltar(b, 'actas', { zona: 'fixa', antesDe: 'actas' })).toBeNull()
    expect(accionDeSoltar(b, 'cronograma', { zona: 'fixa', antesDe: 'actas' })).toBeNull() // xa está diante das actas
    expect(accionDeSoltar(b, 'form:b', { zona: 'fixa', antesDe: null })).toBeNull() // xa é o último
  })
  it('unha ferramenta fixada arrastrada abaixo sae da zona; unha que xa está abaixo non fai nada', () => {
    const b = fixar(bibliotecaBaleira(), 'form:a')
    expect(accionDeSoltar(b, 'form:a', { zona: 'resto' }).fixos).toEqual(['cronograma', 'actas'])
    expect(accionDeSoltar(b, 'cronograma', { zona: 'resto' }).fixos).toEqual(['actas', 'form:a'])
    expect(accionDeSoltar(b, 'form:b', { zona: 'resto' })).toBeNull() // xa estaba abaixo
    expect(accionDeSoltar(b, 'form:a', null)).toBeNull()
  })
  it('completar acepta null e o que devolve Go sen algunha clave', () => {
    expect(completar(null)).toEqual({ ...bibliotecaPorDefecto(), fixos: [], agochados: [] })
    expect(completar({ carpetas: [{ id: 'x', nome: 'X' }] }).fixos).toEqual([])
  })
})
