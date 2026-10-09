import { describe, it, expect } from 'vitest'
import {
  PALETA, resolverCor, calendario, validarDatas, novoPlan, cursoPorDefecto, claveMes,
  normalizarPlan, lerFicheiro, escribirFicheiro, nomeFicheiroSuxerido,
  novaTarefa, gardarTarefa, eliminarTarefa, moverTarefa, duplicarTarefa, moverTarefaAMes,
  totalTarefas, tarefasForaDeRango, imaxesUsadas,
} from './cronograma-modelo.js'

const t = (title, id = title) => ({ id, title, description: '', color: 'blue' })

describe('calendario', () => {
  const cfg = { startDate: '2025-09-08', endDate: '2026-06-19' }
  it('xera os 10 meses do curso, cruzando de ano', () => {
    const m = calendario(cfg, {})
    expect(m).toHaveLength(10)
    expect(m[0]).toMatchObject({ nome: 'Setembro', ano: 2025, clave: '2025-09' })
    expect(m[4]).toMatchObject({ nome: 'Xaneiro', ano: 2026, clave: '2026-01' })
    expect(m[9].nome).toBe('Xuño')
  })
  it('mete as tarefas de cada mes e deixa [] nos baleiros', () => {
    const m = calendario(cfg, { '2025-10': { year: 2025, month: 9, tasks: [t('a')] } })
    expect(m[1].tarefas).toHaveLength(1)
    expect(m[2].tarefas).toEqual([])
  })
  it('datas inválidas ou invertidas dan lista baleira', () => {
    expect(calendario({ startDate: 'x', endDate: '2026-06-19' })).toEqual([])
    expect(calendario({ startDate: '2026-06-01', endDate: '2025-01-01' })).toEqual([])
  })
  it('claveMes enche con ceros', () => {
    expect(claveMes(2025, 0)).toBe('2025-01')
  })
})

describe('validarDatas', () => {
  it('acepta datas boas, incluída a mesma', () => {
    expect(validarDatas('2025-09-08', '2026-06-19')).toBe('')
    expect(validarDatas('2025-09-08', '2025-09-08')).toBe('')
  })
  it('rexeita formato, dato imposible e orde', () => {
    expect(validarDatas('2025-9-8', '2026-06-19')).toMatch(/inicio/)
    expect(validarDatas('2025-02-31', '2026-06-19')).toMatch(/inicio/)
    expect(validarDatas('2025-09-08', '')).toMatch(/fin/)
    expect(validarDatas('2026-06-19', '2025-09-08')).toMatch(/anterior/)
  })
})

describe('curso por defecto', () => {
  it('de xullo en diante é o curso seguinte', () => {
    expect(cursoPorDefecto(new Date(2026, 6, 1))).toEqual({ inicio: 2026, fin: 2027 })
    expect(cursoPorDefecto(new Date(2026, 8, 18))).toEqual({ inicio: 2026, fin: 2027 })
    expect(cursoPorDefecto(new Date(2026, 5, 30))).toEqual({ inicio: 2025, fin: 2026 })
  })
  it('novoPlan é un curso baleiro e válido', () => {
    const p = novoPlan(new Date(2026, 8, 18))
    expect(p.academicYear).toBe('2026/2027')
    expect(validarDatas(p.config.startDate, p.config.endDate)).toBe('')
    expect(p.monthsData).toEqual({})
  })
})

describe('paleta', () => {
  it('resolverCor tolera valores herdados e descoñecidos', () => {
    expect(resolverCor('rose').clave).toBe('rose')
    expect(resolverCor('bg-emerald-50 text-emerald-800').clave).toBe('emerald')
    expect(resolverCor('').clave).toBe(PALETA[0].clave)
    expect(resolverCor('nada').clave).toBe(PALETA[0].clave)
  })
})

describe('ficheiros', () => {
  const orixinal = {
    plans: [{
      id: '1', academicYear: '2025/2026', isLocked: false,
      config: { startDate: '2025-09-08', endDate: '2026-06-19', schoolName: 'IES', location: 'Vigo', headOfDepartment: 'X', schoolLogo: '', introText: '', signatoryRole: '' },
      monthsData: { '2025-09': { year: 2025, month: 8, tasks: [{ id: 't', title: 'T', description: 'd', color: 'blue' }] } },
    }],
  }
  it('le o formato {plans:[...]} de Piztu Orienta', () => {
    const [p] = lerFicheiro(JSON.stringify(orixinal))
    expect(p.config.schoolName).toBe('IES')
    expect(p.monthsData['2025-09'].tasks[0].title).toBe('T')
  })
  it('le o array da versión web e un curso solto', () => {
    expect(lerFicheiro(JSON.stringify(orixinal.plans))).toHaveLength(1)
    expect(lerFicheiro(JSON.stringify(orixinal.plans[0]))).toHaveLength(1)
  })
  it('le todos os cursos dunha copia de seguridade con varios', () => {
    const dous = { plans: [orixinal.plans[0], { ...orixinal.plans[0], id: '2', academicYear: '2026/2027' }] }
    expect(lerFicheiro(JSON.stringify(dous)).map((p) => p.academicYear)).toEqual(['2025/2026', '2026/2027'])
  })
  it('rexeita o que non é un curso, con mensaxe en galego', () => {
    expect(() => lerFicheiro('non json')).toThrow(/JSON/)
    expect(() => lerFicheiro('{"a":1}')).toThrow(/ningún curso/)
    expect(() => lerFicheiro('[]')).toThrow(/ningún curso/)
  })
  it('normalizarPlan completa o que falte sen petar', () => {
    const p = normalizarPlan({ config: { schoolName: 'X' }, monthsData: { '2025-09': { tasks: [{ title: 'sen id' }] } } })
    expect(p.config.startDate).toBeTruthy()
    expect(p.monthsData['2025-09']).toMatchObject({ year: 2025, month: 8 })
    expect(p.monthsData['2025-09'].tasks[0].id).toBeTruthy()
  })
  it('escribir e ler dá o mesmo curso, e o formato é {plans:[...]}', () => {
    const p = normalizarPlan(orixinal.plans[0])
    const texto = escribirFicheiro(p)
    expect(Object.keys(JSON.parse(texto))).toEqual(['plans'])
    expect(lerFicheiro(texto)[0]).toEqual(p)
  })
  it('o borrador pode levar campos extra na raíz sen romper a lectura', () => {
    const texto = escribirFicheiro(normalizarPlan(orixinal.plans[0]), { path: '/tmp/x.json' })
    expect(JSON.parse(texto).path).toBe('/tmp/x.json')
    expect(lerFicheiro(texto)).toHaveLength(1)
  })
  it('nome suxerido a partir do curso', () => {
    expect(nomeFicheiroSuxerido({ academicYear: '2025/2026' })).toBe('cronograma-2025-2026')
    expect(nomeFicheiroSuxerido({ academicYear: '' })).toBe('cronograma')
  })
})

describe('operacións sobre tarefas', () => {
  const base = () => normalizarPlan({ monthsData: {} })

  it('gardarTarefa engade e despois substitúe polo id', () => {
    let p = gardarTarefa(base(), '2025-09', t('a'))
    p = gardarTarefa(p, '2025-09', t('b'))
    expect(p.monthsData['2025-09'].tasks.map((x) => x.title)).toEqual(['a', 'b'])
    p = gardarTarefa(p, '2025-09', { ...t('a'), title: 'A2' })
    expect(p.monthsData['2025-09'].tasks.map((x) => x.title)).toEqual(['A2', 'b'])
    expect(p.monthsData['2025-09']).toMatchObject({ year: 2025, month: 8 })
  })
  it('nunca muta o plan de entrada', () => {
    const p0 = gardarTarefa(base(), '2025-09', t('a'))
    const copia = JSON.stringify(p0)
    gardarTarefa(p0, '2025-09', t('b'))
    eliminarTarefa(p0, '2025-09', 'a')
    moverTarefa(p0, '2025-09', 'a', 1)
    duplicarTarefa(p0, '2025-09', 'a')
    expect(JSON.stringify(p0)).toBe(copia)
  })
  it('eliminar a última tarefa quita o mes do mapa', () => {
    const p = eliminarTarefa(gardarTarefa(base(), '2025-09', t('a')), '2025-09', 'a')
    expect(p.monthsData['2025-09']).toBeUndefined()
  })
  it('moverTarefa reordena e respecta os bordos', () => {
    let p = ['a', 'b', 'c'].reduce((acc, x) => gardarTarefa(acc, '2025-09', t(x)), base())
    p = moverTarefa(p, '2025-09', 'a', 1)
    expect(p.monthsData['2025-09'].tasks.map((x) => x.id)).toEqual(['b', 'a', 'c'])
    p = moverTarefa(p, '2025-09', 'b', -1)
    expect(p.monthsData['2025-09'].tasks.map((x) => x.id)).toEqual(['b', 'a', 'c'])
    p = moverTarefa(p, '2025-09', 'c', 1)
    expect(p.monthsData['2025-09'].tasks.map((x) => x.id)).toEqual(['b', 'a', 'c'])
  })
  it('duplicarTarefa insire a copia xusto despois, con id novo', () => {
    let p = ['a', 'b'].reduce((acc, x) => gardarTarefa(acc, '2025-09', t(x)), base())
    p = duplicarTarefa(p, '2025-09', 'a')
    const ts = p.monthsData['2025-09'].tasks
    expect(ts.map((x) => x.title)).toEqual(['a', 'a (copia)', 'b'])
    expect(new Set(ts.map((x) => x.id)).size).toBe(3)
  })
  it('moverTarefaAMes leva a tarefa ao final doutro mes', () => {
    let p = gardarTarefa(gardarTarefa(base(), '2025-09', t('a')), '2025-10', t('z'))
    p = moverTarefaAMes(p, '2025-09', '2025-10', 'a')
    expect(p.monthsData['2025-09']).toBeUndefined()
    expect(p.monthsData['2025-10'].tasks.map((x) => x.id)).toEqual(['z', 'a'])
    expect(moverTarefaAMes(p, '2025-10', '2025-10', 'a')).toBe(p)
  })
  it('totalTarefas e tarefasForaDeRango', () => {
    let p = gardarTarefa(gardarTarefa(base(), '2025-09', t('a')), '2030-01', t('lonxe'))
    p = { ...p, config: { ...p.config, startDate: '2025-09-08', endDate: '2026-06-19' } }
    expect(totalTarefas(p)).toBe(2)
    expect(tarefasForaDeRango(p)).toBe(1)
  })
  it('novaTarefa ten id e cor por defecto', () => {
    expect(novaTarefa().color).toBe(PALETA[0].clave)
    expect(novaTarefa('rose').color).toBe('rose')
  })
  it('imaxesUsadas descarta as que a descrición xa non usa', () => {
    const imgs = { a: 'data:image/png;base64,AA', b: 'data:image/png;base64,BB' }
    expect(imaxesUsadas('texto ![x](img:a =40%) fin', imgs)).toEqual({ a: imgs.a })
    expect(imaxesUsadas('nada', imgs)).toEqual({})
  })
})
