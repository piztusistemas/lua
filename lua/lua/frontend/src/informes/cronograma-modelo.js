// Modelo de datos do informe "Cronograma anual" - funcións PURAS (sen DOM nin
// bindings), para poder probalas soas. O formato é o mesmo JSON ca o da app
// de escritorio Piztu Orienta (cronograma/internal/model) e ca o struct
// PlanCrono de informe_cronograma.go, así que os cursos gardados alí abren
// aquí sen conversión (e viceversa):
//
//   { id, academicYear, isLocked,
//     config: { startDate, endDate, schoolName, location, department,
//               headOfDepartment, schoolLogo, introText, signatoryRole },
//     monthsData: { "AAAA-MM": { year, month, tasks: [
//        { id, title, description, color, images? } ] } } }
//
// Ningunha función muta o que recibe: devolven un plan novo (o editor garda
// instantáneas para desfacer/refacer).

export const MESES = [
  'Xaneiro', 'Febreiro', 'Marzo', 'Abril', 'Maio', 'Xuño',
  'Xullo', 'Agosto', 'Setembro', 'Outubro', 'Novembro', 'Decembro',
]

// PALETA debe coincidir coa de paletaTarefasCrono (informe_cronograma.go):
// as mesmas claves e cores, para que o que se ve aquí sexa o que sae no PDF.
export const PALETA = [
  { clave: 'blue', nome: 'Azul', fondo: '#EFF6FF', texto: '#1E40AF', bordo: '#BFDBFE' },
  { clave: 'emerald', nome: 'Verde', fondo: '#ECFDF5', texto: '#065F46', bordo: '#A7F3D0' },
  { clave: 'amber', nome: 'Ámbar', fondo: '#FFFBEB', texto: '#92400E', bordo: '#FDE68A' },
  { clave: 'rose', nome: 'Rosa', fondo: '#FFF1F2', texto: '#9F1239', bordo: '#FECDD3' },
  { clave: 'purple', nome: 'Morado', fondo: '#FAF5FF', texto: '#6B21A8', bordo: '#E9D5FF' },
  { clave: 'cyan', nome: 'Cian', fondo: '#ECFEFF', texto: '#155E75', bordo: '#A5F3FC' },
  { clave: 'indigo', nome: 'Índigo', fondo: '#EEF2FF', texto: '#3730A3', bordo: '#C7D2FE' },
  { clave: 'stone', nome: 'Gris', fondo: '#FAFAF9', texto: '#292524', bordo: '#E7E5E4' },
]

// resolverCor: a cor dunha clave. Tolera os valores herdados da versión web
// ("bg-blue-50 text-blue-800 border-blue-200") e cae na primeira da paleta.
export function resolverCor(clave) {
  if (!clave) return PALETA[0]
  const exacta = PALETA.find((c) => c.clave === clave)
  if (exacta) return exacta
  return PALETA.find((c) => String(clave).includes(c.clave)) || PALETA[0]
}

// Cor de fondo da banda de cada mes (0 = xaneiro), como en coresMesCrono (Go).
export const COR_MES = [
  '#DBEAFE', '#CFFAFE', '#ECFDF5', '#D1FAE5', '#DCFCE7', '#FEFCE8',
  '#FEF9C3', '#FFEDD5', '#FFF7ED', '#FEF3C7', '#E7E5E4', '#E2E8F0',
]

export function novoId() {
  return Date.now().toString(36) + Math.random().toString(36).slice(2, 8)
}

const dous = (n) => String(n).padStart(2, '0')

export function claveMes(ano, mes) {
  return `${String(ano).padStart(4, '0')}-${dous(mes + 1)}`
}

// cursoAcademicoPorDefecto: o curso que corresponde a `hoxe` (de xullo en
// diante xa é o seguinte). Devolve os anos de inicio e fin.
export function cursoPorDefecto(hoxe = new Date()) {
  const inicio = hoxe.getMonth() >= 6 ? hoxe.getFullYear() : hoxe.getFullYear() - 1
  return { inicio, fin: inicio + 1 }
}

export function novoPlan(hoxe = new Date()) {
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
      schoolLogo: '',
      introText: '',
      signatoryRole: '',
    },
    monthsData: {},
  }
}

// DEPARTAMENTO_HERDADO: os cursos gardados antes de que Lua servise para
// calquera departamento (sen o campo `department`) eran do Departamento de
// Orientación: consérvase ese nome ao abrilos.
export const DEPARTAMENTO_HERDADO = 'Departamento de Orientación'

function anoMes(data) {
  const m = /^(\d{4})-(\d{1,2})/.exec(String(data || '').trim())
  if (!m) return null
  const mes = parseInt(m[2], 10)
  if (mes < 1 || mes > 12) return null
  return { ano: parseInt(m[1], 10), mes: mes - 1 }
}

// calendario: os meses entre a data de inicio e a de fin, en orde, coas
// tarefas de cada un. Igual ca calendarioCrono (Go).
export function calendario(config, monthsData = {}) {
  const a = anoMes(config?.startDate)
  const b = anoMes(config?.endDate)
  if (!a || !b) return []
  const total = (b.ano - a.ano) * 12 + (b.mes - a.mes) + 1
  if (total <= 0) return []
  const meses = []
  for (let i = 0; i < total; i++) {
    const ano = a.ano + Math.floor((a.mes + i) / 12)
    const mes = (a.mes + i) % 12
    const clave = claveMes(ano, mes)
    meses.push({ ano, mes, clave, nome: MESES[mes], tarefas: monthsData[clave]?.tasks || [] })
  }
  return meses
}

function dataValida(s) {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(String(s || '').trim())
  if (!m) return null
  const d = new Date(Date.UTC(+m[1], +m[2] - 1, +m[3]))
  // Descarta 2025-02-31 (o Date rolaría a marzo sen erro).
  if (d.getUTCFullYear() !== +m[1] || d.getUTCMonth() !== +m[2] - 1 || d.getUTCDate() !== +m[3]) return null
  return d
}

// validarDatas devolve unha mensaxe de erro en galego ou '' se todo está ben.
export function validarDatas(inicio, fin) {
  const i = dataValida(inicio)
  if (!i) return 'A data de inicio non é válida. Usa o formato AAAA-MM-DD (ex.: 2025-09-08).'
  const f = dataValida(fin)
  if (!f) return 'A data de fin non é válida. Usa o formato AAAA-MM-DD (ex.: 2026-06-19).'
  if (f < i) return 'A data de fin non pode ser anterior á de inicio.'
  return ''
}

// Un curso de máis de MAX_MESES é case seguro un erro de teclado (2025 vs
// 2205) e a interface pintaría centos de meses.
export const MAX_MESES = 36

// ---------- Lectura / escritura de ficheiros ----------

function normalizarTarefa(t) {
  const out = {
    id: t?.id ? String(t.id) : novoId(),
    title: String(t?.title ?? ''),
    description: String(t?.description ?? ''),
    color: String(t?.color ?? ''),
  }
  if (t?.images && typeof t.images === 'object' && Object.keys(t.images).length) out.images = { ...t.images }
  return out
}

// normalizarPlan garante os invariantes mínimos (mapas non nulos, datas por
// defecto...) de calquera cousa que pareza un curso. Tolerante: un ficheiro
// a medias non debe impedir abrilo.
export function normalizarPlan(bruto) {
  const base = novoPlan()
  const cfg = bruto?.config || {}
  const plan = {
    id: bruto?.id ? String(bruto.id) : base.id,
    academicYear: String(bruto?.academicYear ?? base.academicYear),
    isLocked: !!bruto?.isLocked,
    config: {
      startDate: String(cfg.startDate || base.config.startDate),
      endDate: String(cfg.endDate || base.config.endDate),
      schoolName: String(cfg.schoolName ?? ''),
      location: String(cfg.location ?? ''),
      department: cfg.department === undefined ? DEPARTAMENTO_HERDADO : String(cfg.department ?? ''),
      headOfDepartment: String(cfg.headOfDepartment ?? ''),
      schoolLogo: String(cfg.schoolLogo ?? ''),
      introText: String(cfg.introText ?? ''),
      signatoryRole: String(cfg.signatoryRole ?? ''),
    },
    monthsData: {},
  }
  for (const [clave, mes] of Object.entries(bruto?.monthsData || {})) {
    const [ano, m] = clave.split('-').map((x) => parseInt(x, 10))
    plan.monthsData[clave] = {
      year: Number.isFinite(mes?.year) ? mes.year : ano,
      month: Number.isFinite(mes?.month) ? mes.month : m - 1,
      tasks: (Array.isArray(mes?.tasks) ? mes.tasks : []).map(normalizarTarefa),
    }
  }
  return plan
}

// lerFicheiro interpreta o contido dun ficheiro e devolve a lista de cursos
// que ten. Acepta: {"plans":[...]} (formato de Piztu Orienta), un array de
// cursos (versión web antiga) ou un curso solto. Lanza Error con mensaxe en
// galego se non é un cursos válido.
export function lerFicheiro(texto) {
  let dato
  try {
    dato = JSON.parse(texto)
  } catch {
    throw new Error('O ficheiro non é un JSON válido.')
  }
  const brutos = Array.isArray(dato) ? dato
    : Array.isArray(dato?.plans) ? dato.plans
      : dato && typeof dato === 'object' && (dato.config || dato.monthsData) ? [dato]
        : null
  if (!brutos || brutos.length === 0) throw new Error('O ficheiro non contén ningún curso.')
  return brutos.map(normalizarPlan)
}

// escribirFicheiro serializa un curso no formato de Piztu Orienta. `extra`
// engade campos á raíz (o borrador garda alí a ruta do ficheiro aberto);
// ignóranos quen le o formato orixinal.
export function escribirFicheiro(plan, extra = {}) {
  return JSON.stringify({ ...extra, plans: [plan] }, null, 2)
}

// nomeFicheiroSuxerido: "cronograma-2025-2026.json" a partir do curso.
export function nomeFicheiroSuxerido(plan) {
  const curso = String(plan?.academicYear || '').replace(/[^\p{L}\p{N}]+/gu, '-').replace(/^-+|-+$/g, '')
  return `cronograma${curso ? '-' + curso : ''}`
}

// ---------- Operacións sobre tarefas (inmutables) ----------

function conMes(plan, clave, fn) {
  const [ano, m] = clave.split('-').map((x) => parseInt(x, 10))
  const actual = plan.monthsData[clave] || { year: ano, month: m - 1, tasks: [] }
  const tarefas = fn([...actual.tasks])
  const monthsData = { ...plan.monthsData }
  if (tarefas.length === 0) delete monthsData[clave]
  else monthsData[clave] = { ...actual, tasks: tarefas }
  return { ...plan, monthsData }
}

export function novaTarefa(cor = PALETA[0].clave) {
  return { id: novoId(), title: '', description: '', color: cor }
}

// gardarTarefa engade a tarefa ao mes, ou substitúe a que teña o mesmo id.
export function gardarTarefa(plan, clave, tarefa) {
  return conMes(plan, clave, (ts) => {
    const i = ts.findIndex((t) => t.id === tarefa.id)
    if (i >= 0) ts[i] = tarefa
    else ts.push(tarefa)
    return ts
  })
}

export function eliminarTarefa(plan, clave, id) {
  return conMes(plan, clave, (ts) => ts.filter((t) => t.id !== id))
}

// moverTarefa cambia a orde dentro do mes: dir = -1 (cara arriba) ou +1.
export function moverTarefa(plan, clave, id, dir) {
  return conMes(plan, clave, (ts) => {
    const i = ts.findIndex((t) => t.id === id)
    const j = i + dir
    if (i < 0 || j < 0 || j >= ts.length) return ts
    ;[ts[i], ts[j]] = [ts[j], ts[i]]
    return ts
  })
}

// duplicarTarefa insire unha copia xusto despois da orixinal.
export function duplicarTarefa(plan, clave, id) {
  return conMes(plan, clave, (ts) => {
    const i = ts.findIndex((t) => t.id === id)
    if (i < 0) return ts
    const orixe = ts[i]
    const copia = {
      ...orixe,
      id: novoId(),
      title: orixe.title ? `${orixe.title} (copia)` : '',
      ...(orixe.images ? { images: { ...orixe.images } } : {}),
    }
    ts.splice(i + 1, 0, copia)
    return ts
  })
}

// moverTarefaAMes leva a tarefa ao final doutro mes.
export function moverTarefaAMes(plan, desde, ata, id) {
  if (desde === ata) return plan
  const tarefa = (plan.monthsData[desde]?.tasks || []).find((t) => t.id === id)
  if (!tarefa) return plan
  return gardarTarefa(eliminarTarefa(plan, desde, id), ata, tarefa)
}

// totalTarefas: cantas hai en todo o curso (incluídos meses fóra do rango
// actual de datas, que o informe non amosa pero o ficheiro conserva).
export function totalTarefas(plan) {
  return Object.values(plan.monthsData).reduce((n, m) => n + m.tasks.length, 0)
}

// tarefasForaDeRango: as que están en meses que o rango de datas actual non
// inclúe. Se se acurta o curso, esas tarefas non saen no informe: convén
// avisar en vez de facelas desaparecer en silencio.
export function tarefasForaDeRango(plan) {
  const dentro = new Set(calendario(plan.config, plan.monthsData).map((m) => m.clave))
  return Object.entries(plan.monthsData)
    .filter(([clave, m]) => !dentro.has(clave) && m.tasks.length > 0)
    .reduce((n, [, m]) => n + m.tasks.length, 0)
}

// imaxesUsadas descarta do mapa as imaxes que a descrición xa non referencia
// (`![alt](img:ID)`), para que o ficheiro non acumule imaxes mortas.
export function imaxesUsadas(descricion, imaxes = {}) {
  const usadas = {}
  for (const m of String(descricion).matchAll(/!\[[^\]]*\]\(img:([^\s)]+)/g)) {
    if (imaxes[m[1]]) usadas[m[1]] = imaxes[m[1]]
  }
  return usadas
}
