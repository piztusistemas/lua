// @vitest-environment jsdom
import { describe, it, expect, afterEach } from 'vitest'
import { createActasEditor } from './actas-editor.js'
import { novoPlanActas, normalizarPlanActas, gardarActa, novaActa, gardarSeguimento } from './actas-modelo.js'

const HOXE = new Date(2025, 9, 15) // outubro de 2025

function plan0() {
  const p = normalizarPlanActas(novoPlanActas(HOXE))
  return { ...p, config: { ...p.config, department: 'Matemáticas', meetingTime: '16:00', meetingPlace: 'Aula 3',
    members: [{ id: 'm1', name: 'Ana', role: '' }, { id: 'm2', name: 'Marta', role: 'Xefatura' }],
    subjects: [{ id: 's1', name: 'Mates', teacher: 'Ana', groups: '1º ESO' }, { id: 's2', name: 'Física', teacher: 'Marta', groups: '' }] } }
}

function montar(planInicial = plan0()) {
  const container = document.createElement('div')
  document.body.append(container)
  const cambios = []
  const ed = createActasEditor(container, { onChange: (p, motivo) => cambios.push({ p, motivo }), hoxe: () => HOXE })
  ed.setValue(planInicial)
  return { ed, container, cambios, ultimo: () => cambios[cambios.length - 1]?.p }
}

const mes = (c, clave) => c.querySelector(`[data-mes="${clave}"]`)
const pantalla = () => document.querySelector('.actas-overlay .tarefa-pantalla')
const botonDe = (raiz, txt) => [...raiz.querySelectorAll('button')].find((b) => b.textContent.trim() === txt)
const escribir = (input, valor) => { input.value = valor; input.dispatchEvent(new Event('input', { bubbles: true })) }
const tick = () => new Promise((r) => setTimeout(r, 0))

afterEach(() => { document.body.replaceChildren() })

describe('lista de meses das actas', () => {
  it('pinta un mes por cada mes do rango co botón para crear a acta e un botón por materia', () => {
    const { container } = montar()
    expect(container.querySelectorAll('.actas-mes')).toHaveLength(10)
    const out = mes(container, '2025-10')
    expect(out.classList.contains('crono-mes--actual')).toBe(true)
    expect(out.querySelector('.crono-mes__cab .crono-btn').textContent).toContain('Engadir acta')
    expect([...out.querySelectorAll('.actas-materia__nome')].map((n) => n.textContent)).toEqual(['Mates', 'Física'])
    expect(out.querySelector('.actas-acta')).toBeNull()
  })

  it('o formulario dos datos do departamento ten membros e materias, e un curso novo ábrese co formulario á vista', () => {
    const { container } = montar(normalizarPlanActas(novoPlanActas(HOXE)))
    const det = container.querySelector('details.crono-config')
    expect(det.open).toBe(true)
    expect(det.querySelector('summary').textContent).toContain('Datos do departamento e materias impartidas')
    expect(container.textContent).toContain('Membros do departamento')
    expect(container.textContent).toContain('Materias impartidas')
    expect(montar().container.querySelector('details.crono-config').open).toBe(false)
  })

  it('engadir membros e materias actualiza o curso e os botóns dos meses', () => {
    const { container, ultimo } = montar()
    const engadirMateria = [...container.querySelectorAll('button')].find((b) => b.textContent.includes('Engadir materia'))
    engadirMateria.click()
    expect(ultimo().config.subjects).toHaveLength(3)
    expect(mes(container, '2025-09').querySelectorAll('.actas-materia')).toHaveLength(3)
    const filas = container.querySelectorAll('.actas-lista__fila')
    const ultimaMateria = [...filas].pop()
    escribir(ultimaMateria.querySelector('input'), 'Química')
    expect(ultimo().config.subjects[2].name).toBe('Química')
    expect([...mes(container, '2025-09').querySelectorAll('.actas-materia__nome')].map((n) => n.textContent)).toContain('Química')
  })
})

describe('a acta mensual', () => {
  it('«Engadir acta» abre a pantalla cos valores habituais e todos os membros; Gardar mete a acta no mes', async () => {
    const { container, ultimo } = montar()
    mes(container, '2025-10').querySelector('.crono-mes__cab .crono-btn').click()
    const p = pantalla()
    expect(p).toBeTruthy()
    expect(p.querySelector('.tarefa-cab__titulo').textContent).toContain('Outubro 2025')
    const [data, inicio, fin, lugar] = p.querySelectorAll('.crono-fila input')
    expect(inicio.value).toBe('16:00')
    expect(lugar.value).toBe('Aula 3')
    const marcas = [...p.querySelectorAll('.actas-asistente input')]
    expect(marcas.map((m) => m.checked)).toEqual([true, true])
    data.value = '2025-10-16'
    fin.value = '17:15'
    marcas[1].checked = false
    marcas[1].dispatchEvent(new Event('change', { bubbles: true }))
    const [orde, desenvolvemento, acordos, rogos] = p.querySelectorAll('textarea')
    orde.value = '1. Lectura da acta'
    acordos.value = '- Coordinar as probas'
    botonDe(p, 'Gardar').click()
    await tick()
    const acta = ultimo().monthsData['2025-10'].acta
    expect(acta).toMatchObject({ date: '2025-10-16', startTime: '16:00', endTime: '17:15', place: 'Aula 3', present: ['m1'], agenda: '1. Lectura da acta', agreements: '- Coordinar as probas' })
    expect(pantalla()).toBeNull() // Gardar pecha
    const card = mes(container, '2025-10').querySelector('.actas-acta')
    expect(card.textContent).toContain('Acta n.º 1 · 16/10/2025')
    expect(card.textContent).toContain('1. Lectura da acta'.replace(/^\d\. /, ''))
    // Xa hai acta: desaparece o botón de engadir.
    expect(mes(container, '2025-10').querySelector('.crono-mes__cab .crono-btn')).toBeNull()
    void desenvolvemento; void rogos
  })

  it('Aplicar garda sen pechar e Cancelar con cambios pide confirmación', async () => {
    const { container, ultimo } = montar()
    mes(container, '2025-11').querySelector('.crono-mes__cab .crono-btn').click()
    const p = pantalla()
    p.querySelector('textarea').value = 'Punto único'
    botonDe(p, 'Aplicar').click()
    await tick()
    expect(ultimo().monthsData['2025-11'].acta.agenda).toBe('Punto único')
    expect(pantalla()).toBeTruthy()
    p.querySelector('textarea').value = 'Outro cambio'
    botonDe(p, 'Cancelar').click()
    await tick()
    const dialogo = document.querySelector('.overlay--dialog:not(.actas-overlay)')
    expect(dialogo).toBeTruthy() // «Hai cambios sen gardar»
    dialogo.querySelector('.modal-actions button:first-child').click() // Seguir editando
    await tick()
    expect(pantalla()).toBeTruthy()
    expect(ultimo().monthsData['2025-11'].acta.agenda).toBe('Punto único')
  })

  it('eliminar a acta pide confirmación e non toca os informes das materias', async () => {
    let p = plan0()
    p = gardarActa(p, '2025-10', { ...novaActa(p.config), id: 'a1' })
    p = gardarSeguimento(p, '2025-10', 's1', { progress: 60 })
    const { container, ultimo } = montar(p)
    mes(container, '2025-10').querySelector('.actas-acta .crono-icona').click()
    await tick()
    document.querySelector('.overlay--dialog .modal-actions button.primary').click()
    await tick()
    expect(ultimo().monthsData['2025-10'].acta).toBeUndefined()
    expect(ultimo().monthsData['2025-10'].reports.s1.progress).toBe(60)
  })
})

describe('os informes de seguimento das materias', () => {
  it('o botón da materia abre o seu informe; Gardar deixa a materia marcada co seu progreso', async () => {
    const { container, ultimo } = montar()
    const botonMateria = mes(container, '2025-10').querySelector('[data-materia="s1"]')
    expect(botonMateria.classList.contains('actas-materia--feita')).toBe(false)
    botonMateria.click()
    const p = pantalla()
    expect(p.querySelector('.tarefa-cab__titulo').textContent).toContain('Mates')
    const [docente, grupos, pct] = p.querySelectorAll('.crono-fila input')
    expect(docente.placeholder).toBe('Ana') // o da materia, ata que se cambie
    expect(grupos.placeholder).toBe('1º ESO')
    pct.value = '75'
    const [contidos] = p.querySelectorAll('textarea')
    contidos.value = 'Unidades 1 e 2'
    botonDe(p, 'Gardar').click()
    await tick()
    expect(ultimo().monthsData['2025-10'].reports.s1).toEqual({ progress: 75, contents: 'Unidades 1 e 2' })
    const feito = mes(container, '2025-10').querySelector('[data-materia="s1"]')
    expect(feito.classList.contains('actas-materia--feita')).toBe(true)
    expect(feito.textContent).toContain('75%')
    expect(mes(container, '2025-10').querySelector('[data-materia="s2"]').classList.contains('actas-materia--feita')).toBe(false)
  })

  it('«Baleirar o informe» quita os datos da materia nese mes', async () => {
    const p = gardarSeguimento(plan0(), '2025-10', 's1', { progress: 60, contents: 'x' })
    const { container, ultimo } = montar(p)
    mes(container, '2025-10').querySelector('[data-materia="s1"]').click()
    botonDe(pantalla(), 'Baleirar o informe').click()
    await tick()
    document.querySelector('.overlay--dialog:not(.actas-overlay) .modal-actions button.primary').click()
    await tick()
    expect(ultimo().monthsData['2025-10']).toBeUndefined()
    expect(pantalla()).toBeNull()
  })

  it('a acta amosa un resumo só de lectura dos informes do mes', () => {
    let p = plan0()
    p = gardarSeguimento(p, '2025-10', 's1', { progress: 60, contents: 'Fraccións' })
    const { container } = montar(p)
    mes(container, '2025-10').querySelector('.crono-mes__cab .crono-btn').click()
    const filas = [...pantalla().querySelectorAll('.actas-resumo-seg__fila')]
    expect(filas).toHaveLength(2)
    expect(filas[0].textContent).toContain('Mates')
    expect(filas[0].textContent).toContain('60% · Fraccións')
    expect(filas[1].textContent).toContain('Sen informe')
  })
})

describe('logotipo', () => {
  it('os datos do departamento non levan logotipo: faise dende a plantilla', () => {
    const { container } = montar()
    expect(container.querySelector('input[type=file]')).toBeNull()
    expect(container.querySelector('.crono-logo')).toBeNull()
  })
})

describe('curso bloqueado', () => {
  it('non deixa crear actas nin gardar cambios', () => {
    const { container } = montar({ ...plan0(), isLocked: true })
    expect(mes(container, '2025-10').querySelector('.crono-mes__cab .crono-btn').disabled).toBe(true)
    mes(container, '2025-10').querySelector('[data-materia="s1"]').click()
    expect(botonDe(pantalla(), 'Gardar')).toBeUndefined()
  })
})
