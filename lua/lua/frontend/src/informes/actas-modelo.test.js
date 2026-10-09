import { describe, it, expect } from 'vitest'
import {
  novoPlanActas, normalizarPlanActas, lerFicheiroActas, escribirFicheiroActas, nomeFicheiroSuxeridoActas, eFicheiroActas,
  novoMembro, novaMateria, novaActa, gardarActa, eliminarActa, gardarSeguimento, seguimentoBaleiro,
  eliminarMateria, eliminarMembro, totalActas, totalSeguimentos, mesesDoCurso, ambitosDisponibles, actasForaDeRango, docenteEGrupos,
} from './actas-modelo.js'
import { escribirFicheiro, novoPlan } from './cronograma-modelo.js'

const HOXE = new Date(2025, 9, 15)
const plan0 = () => normalizarPlanActas(novoPlanActas(HOXE)) // curso 2025/2026, sen actas

function plan2() {
  let p = plan0()
  p = { ...p, config: { ...p.config, members: [{ id: 'm1', name: 'Ana', role: '' }, { id: 'm2', name: 'Marta', role: 'Xefatura' }], subjects: [{ id: 's1', name: 'Mates', teacher: 'Ana', groups: '1º' }] } }
  p = gardarActa(p, '2025-10', { ...novaActa(p.config), id: 'a1', date: '2025-10-16' })
  p = gardarActa(p, '2025-12', { ...novaActa(p.config), id: 'a2' })
  p = gardarSeguimento(p, '2025-10', 's1', { progress: 70, contents: 'Unidade 1' })
  return p
}

describe('curso novo e ficheiros', () => {
  it('un curso novo ten o rango do curso, sen actas, membros nin materias', () => {
    const p = plan0()
    expect(p.academicYear).toBe('2025/2026')
    expect(p.config.startDate).toBe('2025-09-08')
    expect(p.config).toMatchObject({ department: '', members: [], subjects: [], secretary: '' })
    expect(p.monthsData).toEqual({})
    expect(totalActas(p)).toBe(0)
  })

  it('escribe e le o mesmo curso, cunha marca propia para non confundilo co cronograma', () => {
    const p = plan2()
    const texto = escribirFicheiroActas(p, { path: '/tmp/x.json' })
    expect(JSON.parse(texto)).toMatchObject({ ferramenta: 'actas', path: '/tmp/x.json' })
    const [q] = lerFicheiroActas(texto)
    expect(q).toEqual(p)
    expect(eFicheiroActas(JSON.parse(texto))).toBe(true)
  })

  it('un ficheiro de cronograma non se abre coma actas, nin ao revés', () => {
    expect(() => lerFicheiroActas(escribirFicheiro(novoPlan()))).toThrow(/cronograma/)
    expect(() => lerFicheiroActas('isto non é json')).toThrow(/JSON/)
    expect(() => lerFicheiroActas('{"ferramenta":"actas","actas":[]}')).toThrow(/ningún curso/)
  })

  it('tolera un ficheiro a medias: enche os valores por defecto e descarta o baleiro', () => {
    const p = normalizarPlanActas({
      config: { members: [{ name: 'X' }], subjects: [{ name: 'Y' }] },
      monthsData: {
        '2025-11': { acta: { agenda: 'a' }, reports: { s1: { progress: '150', contents: '  ' }, s2: {} } },
        '2025-12': { reports: { s9: {} } },
      },
    })
    expect(p.config.members[0].id).toBeTruthy()
    expect(p.config.subjects[0]).toMatchObject({ name: 'Y', teacher: '', groups: '' })
    expect(p.monthsData['2025-11'].acta).toMatchObject({ agenda: 'a', present: [] })
    expect(p.monthsData['2025-11'].reports).toEqual({ s1: { progress: 100 } }) // limitado a 100; o baleiro sae
    expect(p.monthsData['2025-12']).toBeUndefined()
  })

  it('o nome de ficheiro suxerido leva o curso', () => {
    expect(nomeFicheiroSuxeridoActas({ academicYear: '2025/2026' })).toBe('actas-2025-2026')
    expect(nomeFicheiroSuxeridoActas({})).toBe('actas')
  })
})

describe('actas e informes de materias', () => {
  it('unha acta nova trae a hora e o lugar habituais e todos os membros como asistentes', () => {
    const c = { meetingTime: '16:00', meetingPlace: 'Aula 3', members: [{ id: 'a' }, { id: 'b' }] }
    expect(novaActa(c)).toMatchObject({ startTime: '16:00', place: 'Aula 3', present: ['a', 'b'], agenda: '' })
  })

  it('gardar e eliminar unha acta non toca os informes do mes nin muta o curso orixinal', () => {
    const p = plan2()
    const orixinal = JSON.stringify(p)
    const q = eliminarActa(p, '2025-10')
    expect(JSON.stringify(p)).toBe(orixinal)
    expect(q.monthsData['2025-10'].acta).toBeUndefined()
    expect(q.monthsData['2025-10'].reports.s1.progress).toBe(70)
    expect(totalActas(q)).toBe(1)
    // Sen acta nin informes o mes desaparece do ficheiro.
    expect(eliminarActa(q, '2025-12').monthsData['2025-12']).toBeUndefined()
  })

  it('un informe baleiro borra o informe; un informe con datos gárdase normalizado', () => {
    let p = plan2()
    expect(totalSeguimentos(p)).toBe(1)
    p = gardarSeguimento(p, '2025-10', 's1', { progress: '', contents: '', teacher: '  ' })
    expect(totalSeguimentos(p)).toBe(0)
    p = gardarSeguimento(p, '2025-11', 's1', { progress: 30.6, results: 'Ben' })
    expect(p.monthsData['2025-11'].reports.s1).toEqual({ progress: 31, results: 'Ben' })
    expect(p.monthsData['2025-11'].acta).toBeUndefined()
    expect(seguimentoBaleiro({ progress: 0 })).toBe(false) // o 0 % é un dato
    expect(seguimentoBaleiro(undefined)).toBe(true)
  })

  it('eliminar unha materia leva os seus informes; eliminar un membro quita a súa asistencia', () => {
    const p = plan2()
    const q = eliminarMateria(p, 's1')
    expect(q.config.subjects).toEqual([])
    expect(totalSeguimentos(q)).toBe(0)
    expect(q.monthsData['2025-10'].acta).toBeTruthy()
    const r = eliminarMembro(p, 'm1')
    expect(r.config.members.map((m) => m.id)).toEqual(['m2'])
    expect(r.monthsData['2025-10'].acta.present).toEqual(['m2'])
  })

  it('crea membros e materias con id propio', () => {
    expect(novoMembro().id).not.toBe(novoMembro().id)
    expect(novaMateria()).toMatchObject({ name: '', teacher: '', groups: '' })
  })

  it('o docente e os grupos do informe pisan os da materia só se están escritos', () => {
    const m = { teacher: 'Ana', groups: '1º' }
    expect(docenteEGrupos(m, {})).toEqual({ teacher: 'Ana', groups: '1º' })
    expect(docenteEGrupos(m, { teacher: 'Substituta' })).toEqual({ teacher: 'Substituta', groups: '1º' })
  })
})

describe('meses e ámbitos de impresión', () => {
  it('os meses do curso levan a súa acta e os seus informes', () => {
    const meses = mesesDoCurso(plan2())
    expect(meses).toHaveLength(10)
    expect(meses[1]).toMatchObject({ clave: '2025-10', nome: 'Outubro' })
    expect(meses[1].acta.id).toBe('a1')
    expect(meses[1].reports.s1.progress).toBe(70)
    expect(meses[0].acta).toBeNull()
  })

  it('os ámbitos son as actas do curso, numeradas en orde', () => {
    expect(ambitosDisponibles(plan2())).toEqual([
      { valor: '2025-10', num: 1, nome: 'Outubro', ano: 2025 },
      { valor: '2025-12', num: 2, nome: 'Decembro', ano: 2025 },
    ])
    expect(ambitosDisponibles(plan0())).toEqual([])
  })

  it('avisa das actas que caen fóra do rango de datas', () => {
    const p = plan2()
    expect(actasForaDeRango(p)).toBe(0)
    expect(actasForaDeRango({ ...p, config: { ...p.config, endDate: '2025-11-30' } })).toBe(1)
  })
})
