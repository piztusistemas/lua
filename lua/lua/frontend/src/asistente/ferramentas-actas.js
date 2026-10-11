// Ferramentas do asistente para a ferramenta «Actas do departamento»: le e
// edita o curso de actas (datos do departamento, membros, materias, a acta de
// cada mes e os informes de seguimento das materias). O catálogo vive en Go
// (asistente_ferramentas.go, ferramentasFrontend); aquí só está o código. Mesmo
// contrato ca ferramentas.js: cada función devolve o dato ou lanza un Error
// cunha mensaxe que se lle amosa á IA para que o corrixa.
import {
  calendario, validarDatas, novoMembro, novaMateria, novaActa, gardarActa, eliminarActa, gardarSeguimento,
  seguimentoBaleiro, totalActas, totalSeguimentos, mesesDoCurso, ambitosDisponibles, actasForaDeRango,
} from '../informes/actas-modelo.js'

const erro = (m) => { throw new Error(m) }
const txt = (v) => (v === undefined || v === null ? undefined : String(v))
const nonBaleiro = (v, nome) => {
  const s = (txt(v) ?? '').trim()
  if (!s) erro(`falta ${nome}`)
  return s
}
const norm = (s) => String(s || '').trim().toLowerCase()

// vistaActas: o curso de actas en forma lixeira para a IA (sen o logotipo en base64).
export function vistaActas(plan, { mes } = {}) {
  const c = plan.config
  const out = {
    curso: {
      anoAcademico: plan.academicYear, dataInicio: c.startDate, dataFin: c.endDate, centro: c.schoolName, localidade: c.location,
      departamento: c.department, xefe: c.headOfDepartment, secretario: c.secretary, cargoSinatura: c.signatoryRole,
      horaHabitual: c.meetingTime, lugarHabitual: c.meetingPlace, bloqueado: !!plan.isLocked, tenLogotipo: !!c.schoolLogo,
    },
    membros: c.members.map((m) => ({ id: m.id, nome: m.name, cargo: m.role })),
    materias: c.subjects.map((m) => ({ id: m.id, nome: m.name, docente: m.teacher, grupos: m.groups })),
  }
  let meses = mesesDoCurso(plan)
  if (mes) {
    meses = meses.filter((m) => m.clave === mes)
    if (!meses.length) erro(`o mes ${mes} non está no rango do curso (${mesesDoCurso(plan).map((m) => m.clave).join(', ') || 'sen rango válido'})`)
  }
  let num = 0
  const numeros = {}
  for (const m of mesesDoCurso(plan)) if (m.acta) numeros[m.clave] = ++num
  out.meses = meses.map((m) => ({
    mes: m.clave, nome: `${m.nome} ${m.ano}`,
    acta: m.acta ? {
      numero: numeros[m.clave], data: m.acta.date, horaInicio: m.acta.startTime, horaFin: m.acta.endTime, lugar: m.acta.place,
      asistentes: c.members.filter((x) => m.acta.present.includes(x.id)).map((x) => x.name),
      ordeDia: m.acta.agenda, desenvolvemento: m.acta.development, acordos: m.acta.agreements, rogos: m.acta.questions,
    } : null,
    seguimentos: c.subjects.filter((s) => !seguimentoBaleiro(m.reports[s.id])).map((s) => {
      const r = m.reports[s.id]
      return { materia: s.name, id: s.id, docente: r.teacher, grupos: r.groups, progreso: r.progress, contidos: r.contents, resultados: r.results, dificultades: r.difficulties, medidas: r.measures }
    }),
  }))
  out.totalActas = totalActas(plan)
  out.totalSeguimentos = totalSeguimentos(plan)
  const fora = actasForaDeRango(plan)
  if (fora > 0) out.avisoForaDeRango = `${fora} mes(es) con actas ou informes están fóra do rango de datas e non saen nos documentos`
  return out
}

// crearFerramentasActas devolve as ferramentas (nome -> función) para mesturalas
// coas de ferramentas.js.
//   informes  controlador da ferramenta activa
//   ui        {cambiarFerramenta(id), ferramentaActiva()}
export function crearFerramentasActas({ informes, ui }) {
  const plan = () => informes.getPlan()
  const aplicar = (novo) => informes.substituirPlan(novo, 'estrutura')
  const soActas = () => {
    if (informes.ferramenta?.id !== 'actas') erro('a ferramenta activa non son as actas: chama antes a ferramenta_activa con ferramenta="actas"')
  }
  const editable = () => {
    soActas()
    if (plan().isLocked) erro('o curso está bloqueado: para editalo primeiro hai que desbloquealo (actas_configurar con bloqueado=false)')
  }
  const mesValido = (mes) => {
    const m = nonBaleiro(mes, 'o mes')
    const validos = calendario(plan().config, {}).map((x) => x.clave)
    if (!validos.includes(m)) erro(`o mes ${m} non está no rango do curso. Meses válidos: ${validos.join(', ') || '(ningún: revisa as datas do curso)'}`)
    return m
  }
  // materia: polo id ou polo nome (sen distinguir maiúsculas).
  const atoparMateria = (ref) => {
    const r = nonBaleiro(ref, 'a materia (id ou nome)')
    const p = plan()
    const m = p.config.subjects.find((x) => x.id === r) || p.config.subjects.find((x) => norm(x.name) === norm(r))
    if (!m) erro(`non existe a materia «${r}». Materias: ${p.config.subjects.map((x) => x.name).join(', ') || '(ningunha: defíneas con actas_materias)'}`)
    return m
  }

  return {
    ferramenta_activa(a) {
      const actual = ui.ferramentaActiva()
      if (a.ferramenta === undefined || a.ferramenta === '') return { activa: actual, dispoñibles: ['cronograma', 'actas'] }
      if (a.ferramenta !== 'cronograma' && a.ferramenta !== 'actas') erro("ferramenta ten que ser 'cronograma' ou 'actas'")
      return Promise.resolve(ui.cambiarFerramenta(a.ferramenta)).then(() => ({ activa: a.ferramenta }))
    },

    actas_ler(a) {
      soActas()
      return vistaActas(plan(), { mes: txt(a.mes) })
    },

    actas_configurar(a) {
      soActas()
      const p = plan()
      const config = { ...p.config }
      const cambios = []
      let novo = { ...p }
      if (a.bloqueado !== undefined) {
        novo.isLocked = a.bloqueado === true || a.bloqueado === 'true'
        cambios.push('bloqueado')
      }
      if (p.isLocked && novo.isLocked) editable()
      const mapa = {
        dataInicio: 'startDate', dataFin: 'endDate', centro: 'schoolName', localidade: 'location', departamento: 'department',
        xefe: 'headOfDepartment', secretario: 'secretary', cargoSinatura: 'signatoryRole', horaHabitual: 'meetingTime', lugarHabitual: 'meetingPlace',
      }
      for (const [k, campo] of Object.entries(mapa)) {
        if (a[k] === undefined) continue
        config[campo] = txt(a[k])
        cambios.push(k)
      }
      if (a.anoAcademico !== undefined) { novo.academicYear = txt(a.anoAcademico); cambios.push('anoAcademico') }
      if (a.dataInicio !== undefined || a.dataFin !== undefined) {
        const e = validarDatas(config.startDate, config.endDate)
        if (e) erro(e)
      }
      if (cambios.length === 0) erro('non indicaches ningún campo a cambiar')
      aplicar({ ...novo, config })
      return { cambiados: cambios }
    },

    // Substitúe a lista enteira; os que xa existen (polo nome) conservan o seu id
    // e así non se perden as súas asistencias nin os seus informes.
    actas_membros(a) {
      editable()
      if (!Array.isArray(a.membros)) erro('falta a lista «membros» ([{nome, cargo}])')
      const p = plan()
      const membros = a.membros.map((x) => {
        const nome = nonBaleiro(typeof x === 'string' ? x : x?.nome, 'o nome dun membro')
        const antigo = p.config.members.find((m) => norm(m.name) === norm(nome))
        return { ...(antigo || novoMembro()), name: nome, role: txt(x?.cargo) ?? antigo?.role ?? '' }
      })
      const ids = new Set(membros.map((m) => m.id))
      const monthsData = {}
      for (const [k, m] of Object.entries(p.monthsData)) monthsData[k] = m.acta ? { ...m, acta: { ...m.acta, present: m.acta.present.filter((id) => ids.has(id)) } } : m
      aplicar({ ...p, config: { ...p.config, members: membros }, monthsData })
      return { membros: membros.map((m) => ({ id: m.id, nome: m.name, cargo: m.role })) }
    },

    actas_materias(a) {
      editable()
      if (!Array.isArray(a.materias)) erro('falta a lista «materias» ([{nome, docente, grupos}])')
      const p = plan()
      const materias = a.materias.map((x) => {
        const nome = nonBaleiro(typeof x === 'string' ? x : x?.nome, 'o nome dunha materia')
        const antiga = p.config.subjects.find((m) => norm(m.name) === norm(nome))
        return { ...(antiga || novaMateria()), name: nome, teacher: txt(x?.docente) ?? antiga?.teacher ?? '', groups: txt(x?.grupos) ?? antiga?.groups ?? '' }
      })
      const ids = new Set(materias.map((m) => m.id))
      const monthsData = {}
      for (const [k, m] of Object.entries(p.monthsData)) {
        const reports = Object.fromEntries(Object.entries(m.reports || {}).filter(([id]) => ids.has(id)))
        const novo = { ...m }
        if (Object.keys(reports).length) novo.reports = reports
        else delete novo.reports
        if (novo.acta || novo.reports) monthsData[k] = novo
      }
      aplicar({ ...p, config: { ...p.config, subjects: materias }, monthsData })
      return { materias: materias.map((m) => ({ id: m.id, nome: m.name, docente: m.teacher, grupos: m.groups })) }
    },

    // Crea ou modifica a acta dun mes (só os campos indicados; os textos van COMPLETOS).
    acta_gardar(a) {
      editable()
      const p = plan()
      const mes = mesValido(a.mes)
      const existe = p.monthsData[mes]?.acta
      const acta = { ...(existe || novaActa(p.config)) }
      const cambios = []
      const mapa = { data: 'date', horaInicio: 'startTime', horaFin: 'endTime', lugar: 'place', ordeDia: 'agenda', desenvolvemento: 'development', acordos: 'agreements', rogos: 'questions' }
      for (const [k, campo] of Object.entries(mapa)) {
        if (a[k] === undefined) continue
        acta[campo] = txt(a[k])
        cambios.push(k)
      }
      if (a.data !== undefined && acta.date && !/^\d{4}-\d{2}-\d{2}$/.test(acta.date)) erro('data ten que ir en formato AAAA-MM-DD')
      if (a.asistentes !== undefined) {
        if (a.asistentes === 'todos') acta.present = p.config.members.map((m) => m.id)
        else if (Array.isArray(a.asistentes)) {
          acta.present = a.asistentes.map((n) => {
            const m = p.config.members.find((x) => x.id === n || norm(x.name) === norm(n))
            if (!m) erro(`non existe o membro «${n}». Membros: ${p.config.members.map((x) => x.name).join(', ') || '(ningún: defíneos con actas_membros)'}`)
            return m.id
          })
        } else erro("asistentes ten que ser 'todos' ou unha lista de nomes")
        cambios.push('asistentes')
      }
      if (existe && cambios.length === 0) erro('non indicaches ningún campo a cambiar')
      aplicar(gardarActa(p, mes, acta))
      return { mes, creada: !existe, cambiados: cambios }
    },

    acta_eliminar(a) {
      editable()
      const p = plan()
      const mes = mesValido(a.mes)
      if (!p.monthsData[mes]?.acta) erro(`o mes ${mes} non ten acta`)
      aplicar(eliminarActa(p, mes))
      return { eliminada: mes, nota: 'os informes das materias dese mes consérvanse' }
    },

    // Informe de seguimento dunha materia nun mes: só se cambian os campos indicados.
    acta_seguimento(a) {
      editable()
      const p = plan()
      const mes = mesValido(a.mes)
      const materia = atoparMateria(a.materia)
      const actual = { ...(p.monthsData[mes]?.reports?.[materia.id] || {}) }
      const cambios = []
      const mapa = { docente: 'teacher', grupos: 'groups', contidos: 'contents', resultados: 'results', dificultades: 'difficulties', medidas: 'measures' }
      for (const [k, campo] of Object.entries(mapa)) {
        if (a[k] === undefined) continue
        actual[campo] = txt(a[k])
        cambios.push(k)
      }
      if (a.progreso !== undefined) {
        if (a.progreso === null || a.progreso === '') delete actual.progress
        else {
          const n = Number(a.progreso)
          if (!Number.isFinite(n) || n < 0 || n > 100) erro('progreso ten que ser un número entre 0 e 100')
          actual.progress = n
        }
        cambios.push('progreso')
      }
      if (cambios.length === 0) erro('non indicaches ningún campo a cambiar')
      aplicar(gardarSeguimento(p, mes, materia.id, actual))
      return { mes, materia: materia.name, cambiados: cambios }
    },

    // Que documento se xera e se imprime: todas as actas ou a dun mes.
    actas_ambito(a) {
      soActas()
      if (a.ambito === undefined) return { ambito: informes.getAmbito(), dispoñibles: ['curso', ...ambitosDisponibles(plan()).map((o) => o.valor)] }
      const ambito = String(a.ambito)
      if (ambito !== 'curso' && !ambitosDisponibles(plan()).some((o) => o.valor === ambito)) {
        erro(`ese mes non ten acta. Ámbitos: curso, ${ambitosDisponibles(plan()).map((o) => o.valor).join(', ') || '(ningunha acta aínda)'}`)
      }
      informes.setAmbito(ambito)
      return { ambito }
    },
  }
}
