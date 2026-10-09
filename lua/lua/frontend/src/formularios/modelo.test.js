import { describe, it, expect } from 'vitest'
import {
  slug, novoCampo, inserirCampo, moverCampo, eliminarCampo, duplicarCampo, problemasEsquema, novoEsquema,
  novoPlanFormulario, escribirFicheiroFormulario, lerFicheiroFormulario, nomeFicheiroSuxeridoFormulario,
  haiContidoFormulario, parseCSV, separadorCSV, decodificarCSV, taboaDeCSV, completarIds,
  datosParaExportar, quitarReferencias, ocultarReferencias, amosarReferencias,
} from './modelo.js'

describe('esquema', () => {
  it('slug coincide co de Go', () => {
    expect(slug('Data da reunión')).toBe('data_da_reunion')
    expect(slug('  Nº de Alumnos! ')).toBe('n_de_alumnos')
    expect(slug('2.º trimestre')).toBe('c_2_trimestre')
    expect(slug('???')).toBe('')
  })
  it('novoCampo trae valores por defecto útiles e sen id', () => {
    expect(novoCampo('texto')).toEqual({ id: '', tipo: 'texto', etiqueta: 'Texto curto' })
    expect(novoCampo('escolla').opcions.length).toBe(2)
    expect(novoCampo('taboa').columnas.length).toBe(2)
  })
  it('mover reordena a partir do índice de inserción da lista orixinal', () => {
    const l = ['a', 'b', 'c', 'd'].map((x) => ({ etiqueta: x }))
    const orde = (x) => x.map((c) => c.etiqueta).join('')
    expect(orde(moverCampo(l, 0, 3))).toBe('bcad') // a vai antes de d
    expect(orde(moverCampo(l, 3, 0))).toBe('dabc')
    expect(orde(moverCampo(l, 1, 1))).toBe('abcd') // no mesmo sitio
    expect(orde(moverCampo(l, 1, 2))).toBe('abcd') // xusto despois de si mesmo
    expect(orde(moverCampo(l, 0, 4))).toBe('bcda')
    expect(moverCampo(l, 9, 0)).toBe(l)
  })
  it('inserir, eliminar e duplicar non mutan', () => {
    const l = [{ id: 'a', tipo: 'texto', etiqueta: 'A' }]
    const m = inserirCampo(l, novoCampo('data'), 0)
    expect(m.map((c) => c.tipo)).toEqual(['data', 'texto'])
    expect(l.length).toBe(1)
    expect(eliminarCampo(m, 0).length).toBe(1)
    const d = duplicarCampo(l, 0)
    expect(d.length).toBe(2)
    expect(d[1].id).toBe('')
    expect(d[0].id).toBe('a')
  })
  it('problemasEsquema avisa do que Go rexeitaría', () => {
    const e = { ...novoEsquema(), nome: 'X' }
    expect(problemasEsquema({ ...e, nome: ' ' })).toBe('nome')
    expect(problemasEsquema({ ...e, campos: [{ tipo: 'texto', etiqueta: '' }] })).toBe('etiqueta')
    expect(problemasEsquema({ ...e, campos: [{ tipo: 'escolla', etiqueta: 'E', opcions: [' '] }] })).toBe('opcions')
    expect(problemasEsquema({ ...e, campos: [{ tipo: 'taboa', etiqueta: 'T', columnas: [] }] })).toBe('columnas')
    expect(problemasEsquema({ ...e, campos: [novoCampo('texto')] })).toBe('')
  })
})

describe('completarIds', () => {
  it('pon ids a partir da etiqueta sen repetir e sen tocar os que xa hai', () => {
    const r = completarIds([
      { id: 'nome', tipo: 'texto', etiqueta: 'Outro nome' }, // xa ten id: queda
      { id: '', tipo: 'texto', etiqueta: 'Nome' },           // «nome» está collido
      { id: '', tipo: 'texto', etiqueta: 'Nome' },
      { id: '', tipo: 'numero', etiqueta: '???' },
      { id: '', tipo: 'taboa', etiqueta: 'Alumnado', columnas: [{ id: '', tipo: 'texto', etiqueta: 'Nome' }, { id: 'n', tipo: 'numero', etiqueta: 'Nota' }, { id: '', tipo: 'texto', etiqueta: 'Nome' }] },
    ])
    expect(r.map((c) => c.id)).toEqual(['nome', 'nome_2', 'nome_3', 'campo', 'alumnado'])
    expect(r[4].columnas.map((c) => c.id)).toEqual(['nome', 'n', 'nome_2'])
  })
  it('non muta a entrada', () => {
    const c = [{ id: '', tipo: 'texto', etiqueta: 'X' }]
    completarIds(c)
    expect(c[0].id).toBe('')
  })
})

describe('ficheiro do documento', () => {
  const plan = { ...novoPlanFormulario('ficha', { curso: '2025-2026' }), valores: { nome: 'Ana' } }
  it('vai e volve', () => {
    const [p] = lerFicheiroFormulario(escribirFicheiroFormulario(plan, { path: '/x' }), 'ficha')
    expect(p).toEqual(plan)
  })
  it('recusa outro formulario ou outra ferramenta', () => {
    expect(() => lerFicheiroFormulario(escribirFicheiroFormulario(plan), 'outra', 'Outra')).toThrow(/doutro formulario/)
    expect(() => lerFicheiroFormulario('{"actas":[]}', 'ficha')).toThrow(/non é un documento de formulario/)
    expect(() => lerFicheiroFormulario('{roto', 'ficha')).toThrow(/JSON/)
  })
  it('tolera datos que faltan', () => {
    const [p] = lerFicheiroFormulario('{"ferramenta":"formulario","formulario":"f"}', 'f')
    expect(p.valores).toEqual({})
    expect(p.contexto.centro).toBe('')
  })
  it('nome suxerido e haiContido', () => {
    expect(nomeFicheiroSuxeridoFormulario(plan, { id: 'ficha' })).toBe('ficha-2025-2026')
    expect(haiContidoFormulario(novoPlanFormulario('f'))).toBe(false)
    expect(haiContidoFormulario({ contexto: {}, valores: { a: false, t: [{ x: '' }] } })).toBe(false)
    expect(haiContidoFormulario({ contexto: {}, valores: { t: [{ x: 'v' }] } })).toBe(true)
    expect(haiContidoFormulario({ contexto: { centro: 'IES' }, valores: {} })).toBe(true)
  })
})

describe('CSV', () => {
  it('detecta o separador de Excel en galego', () => {
    expect(separadorCSV('nome;nota\nAna;7,5')).toBe(';')
    expect(separadorCSV('nome,nota\nAna,7.5')).toBe(',')
    expect(separadorCSV('nome\tnota')).toBe('\t')
    expect(separadorCSV('"a;b",c\n')).toBe(',') // o «;» está entre comiñas
  })
  it('parsea comiñas, escapes e saltos dentro de cela', () => {
    expect(parseCSV('a;b\n"x;1";"di ""ola"""\n"liña1\nliña2";z\n')).toEqual([['a', 'b'], ['x;1', 'di "ola"'], ['liña1\nliña2', 'z']])
  })
  it('salta liñas baleiras e aguanta CRLF', () => {
    expect(parseCSV('a;b\r\n\r\n1;2\r\n;\r\n3;4')).toEqual([['a', 'b'], ['1', '2'], ['3', '4']])
  })
  it('decodifica UTF-8 con BOM e Windows-1252', () => {
    const utf8 = new Uint8Array([0xef, 0xbb, 0xbf, ...new TextEncoder().encode('Uxía')])
    expect(decodificarCSV(utf8.buffer)).toBe('Uxía')
    expect(decodificarCSV(new Uint8Array([0x55, 0x78, 0xed, 0x61]).buffer)).toBe('Uxía') // í = 0xED en cp1252
  })
  const cols = [
    { id: 'nome', tipo: 'texto', etiqueta: 'Nome' },
    { id: 'nota', tipo: 'numero', etiqueta: 'Nota' },
    { id: 'adapt', tipo: 'verificacion', etiqueta: 'Adaptación' },
  ]
  it('casa as columnas polo nome da cabeceira, en calquera orde', () => {
    const r = taboaDeCSV(parseCSV('ADAPTACION;nota;Nome;extra\nsi;7,5;Ana;q\nnon;4;Brais;q'), cols)
    expect(r.cabeceira).toBe(true)
    expect(r.casadas).toBe(3)
    expect(r.filas).toEqual([{ nome: 'Ana', nota: '7,5', adapt: true }, { nome: 'Brais', nota: '4', adapt: false }])
  })
  it('sen cabeceira casa por orde e non perde a primeira fila', () => {
    const r = taboaDeCSV(parseCSV('Ana;7\nBrais;4'), cols)
    expect(r.cabeceira).toBe(false)
    expect(r.casadas).toBe(2)
    expect(r.filas.length).toBe(2)
    expect(r.filas[0]).toEqual({ nome: 'Ana', nota: '7', adapt: false })
  })
})

describe('marca de datos de proba no ficheiro', () => {
  it('vai e volve co documento (e sen marca non ensucia o ficheiro)', () => {
    const plan = { formulario: 'x', contexto: {}, valores: { a: '1' }, datosProba: true }
    const [lido] = lerFicheiroFormulario(escribirFicheiroFormulario(plan), 'x')
    expect(lido.datosProba).toBe(true)
    const sen = escribirFicheiroFormulario({ ...plan, datosProba: undefined })
    expect(sen).not.toMatch(/proba/)
    expect('datosProba' in lerFicheiroFormulario(sen, 'x')[0]).toBe(false)
  })
})

describe('datos que viaxan ao exportar un formulario', () => {
  it('sen nada escrito non se exporta nada (só a estrutura)', () => {
    expect(datosParaExportar(novoPlanFormulario('x'))).toBe('')
    expect(datosParaExportar({ formulario: 'x', contexto: {}, valores: { a: '  ', t: [{ n: '' }], v: false } })).toBe('')
  })
  it('con datos leva contexto e valores, e marca os de proba só se o son', () => {
    const plan = { formulario: 'x', contexto: { centro: 'IES' }, valores: { a: '1' } }
    expect(JSON.parse(datosParaExportar(plan))).toEqual({ contexto: { centro: 'IES' }, valores: { a: '1' } })
    expect(JSON.parse(datosParaExportar({ ...plan, datosProba: true })).proba).toBe(true)
  })
})


describe('referencias entre [corchetes]', () => {
  const plan = () => ({
    ...novoPlanFormulario('x'),
    valores: {
      ex: '1. She began talking. (for)  [PAU 2021 · ordinaria]\n2. Another one [clave OFICIAL 2023: admite as dúas formas]',
      nota: 5,
      fila: [{ c: 'texto [PAU 2023 · extraordinaria]' }, { c: 'sen nada' }],
      lig: 'ver [a guía](http://x.org) e [ok]',
    },
  })
  it('quitar deixa o texto limpo e respecta ligazóns e corchetes curtos', () => {
    expect(quitarReferencias('a  [PAU 2021 · ordinaria]')).toBe('a')
    expect(quitarReferencias('ver [a guía](http://x.org) e [ok]')).toBe('ver [a guía](http://x.org) e [ok]')
    expect(quitarReferencias(7)).toBe(7)
  })
  it('desactivar quita as referencias de TODOS os textos (tamén de táboas) e activar repóneas', () => {
    const p = plan()
    const oculto = ocultarReferencias(p)
    expect(oculto.senReferencias).toBe(true)
    expect(oculto.valores.ex).toBe('1. She began talking. (for)\n2. Another one')
    expect(oculto.valores.fila[0].c).toBe('texto')
    expect(oculto.valores.nota).toBe(5)
    expect(oculto.valores.lig).toBe(p.valores.lig)
    expect(p.valores.ex).toMatch(/PAU 2021/) // inmutable
    const de_novo = amosarReferencias(oculto)
    expect(de_novo.senReferencias).toBe(false)
    expect(de_novo.valores).toEqual(p.valores)
    expect(de_novo.referenciasOcultas).toBeUndefined()
  })
  it('un texto editado mentres estaban agochadas non se pisa ao reactivalas', () => {
    const oculto = ocultarReferencias(plan())
    const editado = { ...oculto, valores: { ...oculto.valores, ex: 'texto novo do usuario' } }
    const r = amosarReferencias(editado)
    expect(r.valores.ex).toBe('texto novo do usuario')
    expect(r.valores.fila[0].c).toMatch(/PAU 2023/) // o que non se tocou si volve
  })
})
