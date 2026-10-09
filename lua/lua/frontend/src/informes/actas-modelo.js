// Modelo de datos da ferramenta "Actas do departamento" - funcións PURAS (sen
// DOM nin bindings), para poder probalas soas. É o JSON que espera o backend
// (PlanActas en informe_actas.go):
//
//   { id, academicYear, isLocked,
//     config: { startDate, endDate, schoolName, location, department,
//               headOfDepartment, secretary, signatoryRole, schoolLogo,
//               meetingTime, meetingPlace,
//               members:  [{ id, name, role }],        // persoas do departamento
//               subjects: [{ id, name, teacher, groups }] },  // materias que imparte
//     monthsData: { "AAAA-MM": { year, month,
//        acta?: { id, date, startTime, endTime, place, present: [idMembro],
//                 agenda, development, agreements, questions },   // Markdown
//        reports?: { [idMateria]: { teacher?, groups?, progress?, contents?,
//                    results?, difficulties?, measures? } } } } }
//
// Cada mes ten como moito UNHA acta (a acta mensual) e, por cada materia,
// un informe de seguimento cuxos datos recolle a acta.
//
// O ficheiro leva {"ferramenta":"actas","actas":[curso]} para non confundirse co
// do cronograma. Ningunha función muta o que recibe: devolven un plan novo (o
// editor garda instantáneas para desfacer/refacer).
import {
  MESES, calendario, claveMes, cursoPorDefecto, novoId,
} from './cronograma-modelo.js'

export { MESES, calendario, claveMes, validarDatas, MAX_MESES } from './cronograma-modelo.js'
export const FERRAMENTA_ID = 'actas'

export function novoMembro() { return { id: novoId(), name: '', role: '' } }
export function novaMateria() { return { id: novoId(), name: '', teacher: '', groups: '' } }

export function novoPlanActas(hoxe = new Date()) {
  const { inicio, fin } = cursoPorDefecto(hoxe)
  return {
    id: novoId(),
    academicYear: `${inicio}/${fin}`,
    isLocked: false,
    config: {
      startDate: `${inicio}-09-08`,
      endDate: `${fin}-06-19`,
      schoolName: '',
      location: '',
      department: '',
      headOfDepartment: '',
      secretary: '',
      signatoryRole: '',
      schoolLogo: '',
      meetingTime: '',
      meetingPlace: '',
      members: [],
      subjects: [],
    },
    monthsData: {},
  }
}

// ---------- Lectura / escritura de ficheiros ----------

const txt = (v) => String(v ?? '')

function normalizarSeguimento(s) {
  const out = {}
  for (const k of ['teacher', 'groups', 'contents', 'results', 'difficulties', 'measures']) {
    if (txt(s?.[k]).trim() !== '') out[k] = txt(s[k])
  }
  const p = s?.progress
  if (p !== null && p !== undefined && p !== '' && Number.isFinite(Number(p))) out.progress = Math.min(100, Math.max(0, Math.round(Number(p))))
  return out
}

function normalizarActa(a) {
  return {
    id: a?.id ? String(a.id) : novoId(),
    date: txt(a?.date),
    startTime: txt(a?.startTime),
    endTime: txt(a?.endTime),
    place: txt(a?.place),
    present: Array.isArray(a?.present) ? a.present.map(String) : [],
    agenda: txt(a?.agenda),
    development: txt(a?.development),
    agreements: txt(a?.agreements),
    questions: txt(a?.questions),
  }
}

// normalizarPlanActas garante os invariantes mínimos de calquera cousa que
// pareza un curso de actas: tolerante, un ficheiro a medias non debe impedir
// abrilo.
export function normalizarPlanActas(bruto) {
  const base = novoPlanActas()
  const cfg = bruto?.config || {}
  const plan = {
    id: bruto?.id ? String(bruto.id) : base.id,
    academicYear: txt(bruto?.academicYear ?? base.academicYear),
    isLocked: !!bruto?.isLocked,
    config: {
      startDate: txt(cfg.startDate || base.config.startDate),
      endDate: txt(cfg.endDate || base.config.endDate),
      schoolName: txt(cfg.schoolName),
      location: txt(cfg.location),
      department: txt(cfg.department),
      headOfDepartment: txt(cfg.headOfDepartment),
      secretary: txt(cfg.secretary),
      signatoryRole: txt(cfg.signatoryRole),
      schoolLogo: txt(cfg.schoolLogo),
      meetingTime: txt(cfg.meetingTime),
      meetingPlace: txt(cfg.meetingPlace),
      members: (Array.isArray(cfg.members) ? cfg.members : []).map((m) => ({ id: m?.id ? String(m.id) : novoId(), name: txt(m?.name), role: txt(m?.role) })),
      subjects: (Array.isArray(cfg.subjects) ? cfg.subjects : []).map((m) => ({
        id: m?.id ? String(m.id) : novoId(), name: txt(m?.name), teacher: txt(m?.teacher), groups: txt(m?.groups),
      })),
    },
    monthsData: {},
  }
  for (const [clave, mes] of Object.entries(bruto?.monthsData || {})) {
    const [ano, m] = clave.split('-').map((x) => parseInt(x, 10))
    const novo = {
      year: Number.isFinite(mes?.year) ? mes.year : ano,
      month: Number.isFinite(mes?.month) ? mes.month : m - 1,
    }
    if (mes?.acta) novo.acta = normalizarActa(mes.acta)
    const reports = {}
    for (const [id, r] of Object.entries(mes?.reports || {})) {
      const n = normalizarSeguimento(r)
      if (Object.keys(n).length) reports[id] = n
    }
    if (Object.keys(reports).length) novo.reports = reports
    if (novo.acta || novo.reports) plan.monthsData[clave] = novo
  }
  return plan
}

// eFicheiroActas di se un JSON xa lido é un ficheiro desta ferramenta.
export function eFicheiroActas(dato) {
  return !!dato && typeof dato === 'object' && !Array.isArray(dato) && (dato.ferramenta === FERRAMENTA_ID || Array.isArray(dato.actas))
}

// lerFicheiroActas devolve a lista de cursos dun ficheiro; lanza Error con
// mensaxe en galego se non é un ficheiro de actas válido.
export function lerFicheiroActas(texto) {
  let dato
  try {
    dato = JSON.parse(texto)
  } catch {
    throw new Error('O ficheiro non é un JSON válido.')
  }
  if (!eFicheiroActas(dato)) {
    throw new Error('O ficheiro non é un curso de actas (parece un cronograma). Ábreo desde a ferramenta do cronograma.')
  }
  const brutos = Array.isArray(dato.actas) ? dato.actas : []
  if (brutos.length === 0) throw new Error('O ficheiro non contén ningún curso.')
  return brutos.map(normalizarPlanActas)
}

// escribirFicheiroActas serializa un curso. `extra` engade campos á raíz (o
// borrador garda alí a ruta do ficheiro aberto).
export function escribirFicheiroActas(plan, extra = {}) {
  return JSON.stringify({ ferramenta: FERRAMENTA_ID, ...extra, actas: [plan] }, null, 2)
}

// nomeFicheiroSuxeridoActas: "actas-2025-2026" a partir do curso.
export function nomeFicheiroSuxeridoActas(plan) {
  const curso = String(plan?.academicYear || '').replace(/[^\p{L}\p{N}]+/gu, '-').replace(/^-+|-+$/g, '')
  return `actas${curso ? '-' + curso : ''}`
}

// ---------- Operacións (inmutables) ----------

function conMes(plan, clave, fn) {
  const [ano, m] = clave.split('-').map((x) => parseInt(x, 10))
  const actual = plan.monthsData[clave] || { year: ano, month: m - 1 }
  const novo = fn({ ...actual, ...(actual.reports ? { reports: { ...actual.reports } } : {}) })
  const monthsData = { ...plan.monthsData }
  if (!novo.acta && !(novo.reports && Object.keys(novo.reports).length)) delete monthsData[clave]
  else {
    if (novo.reports && Object.keys(novo.reports).length === 0) delete novo.reports
    monthsData[clave] = novo
  }
  return { ...plan, monthsData }
}

// novaActa: unha acta en branco cos valores habituais do departamento (hora e
// lugar das reunións) e todos os membros como asistentes.
export function novaActa(config) {
  return {
    id: novoId(),
    date: '',
    startTime: config?.meetingTime || '',
    endTime: '',
    place: config?.meetingPlace || '',
    present: (config?.members || []).map((m) => m.id),
    agenda: '', development: '', agreements: '', questions: '',
  }
}

export function gardarActa(plan, clave, acta) {
  return conMes(plan, clave, (m) => ({ ...m, acta }))
}

export function eliminarActa(plan, clave) {
  return conMes(plan, clave, (m) => { const { acta, ...resto } = m; return resto })
}

export function seguimentoBaleiro(s) {
  return !s || Object.keys(normalizarSeguimento(s)).length === 0
}

// gardarSeguimento garda o informe dunha materia nun mes (un informe baleiro bórrase).
export function gardarSeguimento(plan, clave, idMateria, seguimento) {
  const n = normalizarSeguimento(seguimento)
  return conMes(plan, clave, (m) => {
    const reports = { ...(m.reports || {}) }
    if (Object.keys(n).length) reports[idMateria] = n
    else delete reports[idMateria]
    return { ...m, reports }
  })
}

// eliminarMateria quita a materia e os seus informes de todos os meses.
export function eliminarMateria(plan, id) {
  const monthsData = {}
  for (const [clave, m] of Object.entries(plan.monthsData)) {
    const reports = { ...(m.reports || {}) }
    delete reports[id]
    const novo = { ...m }
    if (Object.keys(reports).length) novo.reports = reports
    else delete novo.reports
    if (novo.acta || novo.reports) monthsData[clave] = novo
  }
  return { ...plan, config: { ...plan.config, subjects: plan.config.subjects.filter((s) => s.id !== id) }, monthsData }
}

// eliminarMembro quita a persoa e a súa asistencia das actas.
export function eliminarMembro(plan, id) {
  const monthsData = {}
  for (const [clave, m] of Object.entries(plan.monthsData)) {
    monthsData[clave] = m.acta ? { ...m, acta: { ...m.acta, present: m.acta.present.filter((x) => x !== id) } } : m
  }
  return { ...plan, config: { ...plan.config, members: plan.config.members.filter((x) => x.id !== id) }, monthsData }
}

// ---------- Consultas ----------

export function totalActas(plan) {
  return Object.values(plan.monthsData).filter((m) => m.acta).length
}

export function totalSeguimentos(plan) {
  return Object.values(plan.monthsData).reduce((n, m) => n + Object.keys(m.reports || {}).length, 0)
}

// mesesDoCurso: os meses do rango de datas coa súa acta e os seus informes.
export function mesesDoCurso(plan) {
  return calendario(plan.config, {}).map((m) => ({
    ...m,
    acta: plan.monthsData[m.clave]?.acta || null,
    reports: plan.monthsData[m.clave]?.reports || {},
  }))
}

// ambitosDisponibles: as opcións de "que imprimir": o curso enteiro (todas as
// actas) ou unha acta, numerada como no documento.
export function ambitosDisponibles(plan) {
  const out = []
  let num = 0
  for (const m of mesesDoCurso(plan)) {
    if (!m.acta) continue
    num++
    out.push({ valor: m.clave, num, nome: m.nome, ano: m.ano })
  }
  return out
}

// actasForaDeRango: as actas e informes en meses que o rango de datas actual
// non inclúe (non sairían no documento).
export function actasForaDeRango(plan) {
  const dentro = new Set(calendario(plan.config, {}).map((m) => m.clave))
  return Object.entries(plan.monthsData)
    .filter(([clave, m]) => !dentro.has(clave) && (m.acta || Object.keys(m.reports || {}).length))
    .length
}

// materiaDeSeguimento devolve o docente e os grupos efectivos dun informe (os
// da materia se o informe non os cambia).
export function docenteEGrupos(materia, informe) {
  return { teacher: informe?.teacher || materia.teacher || '', groups: informe?.groups || materia.groups || '' }
}
