import { describe, it, expect, vi } from 'vitest'
import { crearFerramentas } from './ferramentas.js'
import { vistaActas } from './ferramentas-actas.js'
import { novoPlanActas, normalizarPlanActas, gardarActa, novaActa, gardarSeguimento } from '../informes/actas-modelo.js'
import { FERRAMENTA_ACTAS } from '../informes/ferramenta-actas.js'
import { FERRAMENTA_CRONOGRAMA } from '../informes/ferramenta-cronograma.js'

const HOXE = new Date(2025, 9, 15)

function montar(plan = normalizarPlanActas(novoPlanActas(HOXE)), { ferramenta = FERRAMENTA_ACTAS } = {}) {
  const estado = { plan, ambito: 'curso', ferramenta: ferramenta.id }
  const informes = {
    get ferramenta() { return estado.ferramenta === 'actas' ? FERRAMENTA_ACTAS : FERRAMENTA_CRONOGRAMA },
    getPlan: () => estado.plan,
    substituirPlan: (novo) => { estado.plan = novo },
    ruta: () => '/tmp/actas.json',
    getAmbito: () => estado.ambito, setAmbito: (a) => { estado.ambito = a },
    xerar: vi.fn(async () => true),
  }
  const ui = {
    ferramentaActiva: () => estado.ferramenta,
    cambiarFerramenta: vi.fn(async (id) => { estado.ferramenta = id }),
    avisar: vi.fn(), preguntar: vi.fn(),
  }
  const { executar } = crearFerramentas({ informes, api: { IAChamarFerramenta: vi.fn() }, ui })
  const chamar = async (nome, args = {}) => {
    const r = await executar(nome, args)
    return { ...r, dato: r.erro ? null : JSON.parse(r.contido) }
  }
  return { estado, executar, chamar, ui }
}

describe('ferramentas do asistente para as actas', () => {
  it('configura o departamento, os membros e as materias, conservando ids ao volver definilos', async () => {
    const { chamar, estado } = montar()
    expect((await chamar('actas_configurar', { departamento: 'Matemáticas', xefe: 'Ana', secretario: 'Marta', horaHabitual: '16:00', lugarHabitual: 'Aula 3' })).dato.cambiados).toHaveLength(5)
    expect(estado.plan.config).toMatchObject({ department: 'Matemáticas', headOfDepartment: 'Ana', secretary: 'Marta', meetingTime: '16:00', meetingPlace: 'Aula 3' })
    const m1 = await chamar('actas_membros', { membros: [{ nome: 'Ana', cargo: 'Xefatura' }, 'Marta'] })
    expect(m1.dato.membros.map((m) => m.nome)).toEqual(['Ana', 'Marta'])
    const idAna = estado.plan.config.members[0].id
    await chamar('actas_membros', { membros: [{ nome: 'ana' }, { nome: 'Xoán' }] }) // mesmo membro (sen distinguir maiúsculas)
    expect(estado.plan.config.members[0]).toMatchObject({ id: idAna, name: 'ana', role: 'Xefatura' })
    await chamar('actas_materias', { materias: [{ nome: 'Mates', docente: 'Ana', grupos: '1º' }, 'Física'] })
    expect(estado.plan.config.subjects.map((s) => s.name)).toEqual(['Mates', 'Física'])
    const idMates = estado.plan.config.subjects[0].id
    await chamar('actas_materias', { materias: [{ nome: 'MATES' }] })
    expect(estado.plan.config.subjects).toEqual([{ id: idMates, name: 'MATES', teacher: 'Ana', groups: '1º' }])
  })

  it('crea a acta dun mes, con asistentes por nome, e valida o mes e a data', async () => {
    const { chamar, estado } = montar()
    await chamar('actas_membros', { membros: ['Ana', 'Marta', 'Xoán'] })
    await chamar('actas_configurar', { horaHabitual: '16:00' })
    const r = await chamar('acta_gardar', { mes: '2025-10', data: '2025-10-16', horaFin: '17:00', asistentes: ['ana', 'Marta'], ordeDia: '1. Un', acordos: '- Feito' })
    expect(r.dato).toMatchObject({ mes: '2025-10', creada: true })
    const a = estado.plan.monthsData['2025-10'].acta
    expect(a).toMatchObject({ date: '2025-10-16', startTime: '16:00', endTime: '17:00', agenda: '1. Un', agreements: '- Feito' })
    expect(a.present).toHaveLength(2)
    // Modificar só un campo.
    const r2 = await chamar('acta_gardar', { mes: '2025-10', lugar: 'Biblioteca', asistentes: 'todos' })
    expect(r2.dato.creada).toBe(false)
    expect(estado.plan.monthsData['2025-10'].acta).toMatchObject({ place: 'Biblioteca', agenda: '1. Un' })
    expect(estado.plan.monthsData['2025-10'].acta.present).toHaveLength(3)
    // Erros claros.
    expect((await chamar('acta_gardar', { mes: '2030-01' })).contido).toMatch(/non está no rango/)
    expect((await chamar('acta_gardar', { mes: '2025-10', data: '16/10/2025' })).contido).toMatch(/AAAA-MM-DD/)
    expect((await chamar('acta_gardar', { mes: '2025-10', asistentes: ['Descoñecido'] })).contido).toMatch(/non existe o membro/)
    expect((await chamar('acta_gardar', { mes: '2025-10' })).contido).toMatch(/ningún campo/)
  })

  it('crea informes de seguimento por materia (polo nome) e valida o progreso', async () => {
    const { chamar, estado } = montar()
    await chamar('actas_materias', { materias: ['Mates'] })
    const r = await chamar('acta_seguimento', { mes: '2025-11', materia: 'mates', progreso: 80, contidos: 'Unidade 2' })
    expect(r.dato).toMatchObject({ materia: 'Mates', cambiados: ['contidos', 'progreso'] })
    const id = estado.plan.config.subjects[0].id
    expect(estado.plan.monthsData['2025-11'].reports[id]).toEqual({ contents: 'Unidade 2', progress: 80 })
    await chamar('acta_seguimento', { mes: '2025-11', materia: id, progreso: null })
    expect(estado.plan.monthsData['2025-11'].reports[id]).toEqual({ contents: 'Unidade 2' })
    expect((await chamar('acta_seguimento', { mes: '2025-11', materia: 'mates', progreso: 120 })).contido).toMatch(/entre 0 e 100/)
    expect((await chamar('acta_seguimento', { mes: '2025-11', materia: 'Historia', contidos: 'x' })).contido).toMatch(/non existe a materia/)
  })

  it('actas_ler amosa as actas numeradas e os seguimentos; acta_eliminar quita só a acta', async () => {
    let p = normalizarPlanActas(novoPlanActas(HOXE))
    p = { ...p, config: { ...p.config, members: [{ id: 'm1', name: 'Ana', role: '' }], subjects: [{ id: 's1', name: 'Mates', teacher: 'Ana', groups: '' }] } }
    p = gardarActa(p, '2025-10', { ...novaActa(p.config), date: '2025-10-16', agenda: 'Orde' })
    p = gardarActa(p, '2025-12', { ...novaActa(p.config) })
    p = gardarSeguimento(p, '2025-12', 's1', { progress: 50 })
    const { chamar, estado } = montar(p)
    const v = (await chamar('actas_ler')).dato
    expect(v.membros).toEqual([{ id: 'm1', nome: 'Ana', cargo: '' }])
    expect(v.meses.find((m) => m.mes === '2025-12').acta.numero).toBe(2)
    expect(v.meses.find((m) => m.mes === '2025-12').seguimentos[0]).toMatchObject({ materia: 'Mates', progreso: 50 })
    expect((await chamar('actas_ler', { mes: '2025-10' })).dato.meses).toHaveLength(1)
    expect(vistaActas(p).totalActas).toBe(2)
    await chamar('acta_eliminar', { mes: '2025-12' })
    expect(estado.plan.monthsData['2025-12'].acta).toBeUndefined()
    expect(estado.plan.monthsData['2025-12'].reports.s1.progress).toBe(50)
    expect((await chamar('acta_eliminar', { mes: '2025-11' })).contido).toMatch(/non ten acta/)
  })

  it('o ámbito do documento: curso ou un mes con acta', async () => {
    let p = normalizarPlanActas(novoPlanActas(HOXE))
    p = gardarActa(p, '2025-10', novaActa(p.config))
    const { chamar, estado } = montar(p)
    expect((await chamar('actas_ambito')).dato).toEqual({ ambito: 'curso', 'dispoñibles': ['curso', '2025-10'] })
    await chamar('actas_ambito', { ambito: '2025-10' })
    expect(estado.ambito).toBe('2025-10')
    expect((await chamar('actas_ambito', { ambito: '2025-11' })).contido).toMatch(/non ten acta/)
  })

  it('un curso bloqueado non se edita', async () => {
    const { chamar } = montar({ ...normalizarPlanActas(novoPlanActas(HOXE)), isLocked: true })
    expect((await chamar('acta_gardar', { mes: '2025-10', lugar: 'x' })).contido).toMatch(/bloqueado/)
  })
})

describe('as dúas ferramentas non se mesturan', () => {
  it('coas actas activas as ferramentas do cronograma dan un erro claro', async () => {
    const { chamar } = montar()
    for (const nome of ['curso_ler', 'tarefa_engadir', 'curso_configurar']) {
      const r = await chamar(nome, { mes: '2025-10', titulo: 'x', xefe: 'y' })
      expect(r.erro, nome).toBe(true)
      expect(r.contido).toMatch(/ACTAS/)
    }
  })

  it('co cronograma activo as ferramentas das actas piden cambiar de ferramenta', async () => {
    const { chamar, ui, estado } = montar(normalizarPlanActas(novoPlanActas(HOXE)), { ferramenta: FERRAMENTA_CRONOGRAMA })
    const r = await chamar('actas_ler')
    expect(r.erro).toBe(true)
    expect(r.contido).toMatch(/ferramenta_activa/)
    expect((await chamar('ferramenta_activa')).dato).toEqual({ activa: 'cronograma', 'dispoñibles': ['cronograma', 'actas'] })
    expect((await chamar('ferramenta_activa', { ferramenta: 'actas' })).dato).toEqual({ activa: 'actas' })
    expect(ui.cambiarFerramenta).toHaveBeenCalledWith('actas')
    expect(estado.ferramenta).toBe('actas')
    expect((await chamar('ferramenta_activa', { ferramenta: 'outra' })).erro).toBe(true)
  })
})
