// createActasEditor - a interface da ferramenta "Actas do departamento": datos do
// departamento e das materias que imparte e, por cada mes, a súa acta mensual
// e os informes de seguimento de cada materia (cuxos datos recolle a acta).
//
// Mesmo contrato "factoría illada" ca cronograma-editor.js: recibe o contedor
// e unhas dependencias (t, onChange, lerImaxe), nunca fala cos bindings Wails
// - iso é cousa de informes/controlador.js. O estado é un PlanActas (ver
// actas-modelo.js) e NUNCA se muta: cada cambio constrúe un plan novo e avisa
// a onChange(plan, motivo), onde `motivo` é 'texto' (tecleo nun campo, o
// controlador agrupa estes para desfacer) ou 'estrutura'.
//
// A acta e cada informe de materia edítanse en pantalla completa dentro desta
// mesma xanela (mesmo aspecto ca a pantalla de edición das tarefas).
import { confirmar } from '../comun/dialogs.js'
import {
  MAX_MESES, calendario, claveMes, validarDatas,
  novoMembro, novaMateria, novaActa, gardarActa, eliminarActa, gardarSeguimento, seguimentoBaleiro,
  eliminarMateria, eliminarMembro, totalActas, totalSeguimentos, mesesDoCurso, actasForaDeRango, docenteEGrupos,
} from './actas-modelo.js'
import { el } from './dom.js'

const ICONA_X = '<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round"><path d="M5 5l10 10M15 5L5 15"/></svg>'
const ICONAS = {
  editar: '<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M4 16 14.5 5.5a1.4 1.4 0 0 1 2 2L6 18l-3 1z"/></svg>',
  lixo: '<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M4.5 5.5h11"/><path d="M8 5.5V4a1 1 0 0 1 1-1h2a1 1 0 0 1 1 1v1.5"/><path d="M5.5 5.5 6.2 16a1 1 0 0 0 1 .9h5.6a1 1 0 0 0 1-.9l.7-10.5"/></svg>',
  mais: '<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"><path d="M10 4.5v11M4.5 10h11"/></svg>',
}

// resumo: as primeiras palabras dun texto Markdown sen marcas, para
// recoñecer unha acta na lista sen abrila.
function resumo(md, max = 120) {
  const plano = String(md || '')
    .replace(/^:::.*$/gm, ' ').replace(/!\[[^\]]*\]\([^)]*\)/g, ' ')
    .replace(/\[([^\]]*)\]\([^)]*\)/g, '$1').replace(/^[#>\-*\d. ]+/gm, '').replace(/[*_]/g, '')
    .replace(/\s+/g, ' ').trim()
  return plano.length > max ? plano.slice(0, max - 1) + '…' : plano
}

// dataCurta: "16/10/2025" dun AAAA-MM-DD ('' se non é válida).
function dataCurta(d) {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(String(d || ''))
  return m ? `${m[3]}/${m[2]}/${m[1]}` : ''
}

export function createActasEditor(container, {
  t: tIn, onChange, hoxe = () => new Date(),
} = {}) {
  const t = tIn || ((clave, fallback, vars) => {
    let val = fallback ?? clave
    if (vars) for (const k in vars) val = val.replaceAll(`{${k}}`, vars[k])
    return val
  })

  let plan = null
  let destruido = false
  container.classList.add('crono-editor')

  const aplicar = (novo, motivo = 'estrutura') => {
    plan = novo
    if (onChange) onChange(plan, motivo)
  }
  const mutarConfig = (parcial, motivo) => aplicar({ ...plan, config: { ...plan.config, ...parcial } }, motivo)
  // estruturar: un cambio que altera as listas do curso (hai que repintalas).
  const estruturar = (novo) => { aplicar(novo); renderMeses(); actualizarResumoCurso() }

  const refs = {}

  function camp(etiqueta, control, ancho) {
    return el('label', { class: 'crono-camp' + (ancho ? ' crono-camp--' + ancho : '') },
      el('span', { class: 'crono-camp__etq', text: etiqueta }), control)
  }

  function entrada(campo, { tipo = 'text', placeholder = '' } = {}) {
    return el('input', {
      type: tipo, value: plan.config[campo] ?? '', placeholder,
      oninput: (e) => mutarConfig({ [campo]: e.target.value }, 'texto'),
    })
  }

  // ---------- Datos do departamento ----------

  function actualizarErroDatas() {
    const erro = validarDatas(plan.config.startDate, plan.config.endDate)
    const n = calendario(plan.config, {}).length
    refs.erroDatas.textContent = erro || (n > MAX_MESES
      ? t('informes.datas.demasiados', 'O rango ten {n} meses (máximo {max}). Revisa as datas.', { n, max: MAX_MESES })
      : '')
    refs.erroDatas.hidden = !refs.erroDatas.textContent
  }

  function cambioData(campo, valor) {
    if (valor === '') {
      refs.erroDatas.textContent = t('informes.datas.incompleta', 'Completa a data.')
      refs.erroDatas.hidden = false
      return
    }
    mutarConfig({ [campo]: valor }, 'estrutura')
    actualizarErroDatas()
    renderMeses()
    actualizarResumoCurso()
  }

  function actualizarResumoCurso() {
    const na = totalActas(plan)
    const ns = plan.config.subjects.length
    refs.resumoCurso.textContent = [
      plan.academicYear || '',
      plan.config.department || '',
      t(na === 1 ? 'actas.actas.unha' : 'actas.actas.n', na === 1 ? '1 acta' : '{n} actas', { n: na }),
      t(ns === 1 ? 'actas.materias.unha' : 'actas.materias.n', ns === 1 ? '1 materia' : '{n} materias', { n: ns }),
    ].filter(Boolean).join(' · ')
  }

  // Listas editables (membros e materias): unha fila por elemento, co seu
  // botón de eliminar, e un botón para engadir.
  function listaEditable({ clave, titulo, engadirTexto, novo, campos, eliminar }) {
    const caixa = el('div', { class: 'crono-camp' }, el('span', { class: 'crono-camp__etq', text: titulo }))
    const filas = el('div', { class: 'actas-lista' })
    const pintar = () => {
      filas.replaceChildren()
      for (const item of plan.config[clave]) {
        const fila = el('div', { class: 'actas-lista__fila' })
        for (const [prop, ph, ancho] of campos) {
          fila.append(el('input', {
            type: 'text', value: item[prop] ?? '', placeholder: ph, class: ancho ? 'actas-lista__' + ancho : '',
            oninput: (e) => {
              mutarConfig({ [clave]: plan.config[clave].map((x) => (x.id === item.id ? { ...x, [prop]: e.target.value } : x)) }, 'texto')
              actualizarResumoCurso()
              if (clave === 'subjects') renderMeses()
            },
          }))
        }
        fila.append(el('button', {
          type: 'button', class: 'crono-icona', html: ICONAS.lixo, title: t('informes.eliminar', 'Eliminar'),
          'aria-label': t('informes.eliminar', 'Eliminar'),
          onclick: async () => {
            const nome = item.name || t('actas.sen.nome', 'Sen nome')
            const ok = await confirmar(t(clave === 'subjects' ? 'actas.materia.eliminarConfirmar' : 'actas.membro.eliminarConfirmar',
              clave === 'subjects' ? 'Eliminar a materia «{nome}» e os seus informes de seguimento?' : 'Eliminar a «{nome}» do departamento?', { nome }),
            { aceptar: t('informes.eliminar', 'Eliminar'), cancelar: t('dialog.cancel', 'Cancelar') })
            if (!ok) return
            aplicar(eliminar(plan, item.id))
            pintar(); renderMeses(); actualizarResumoCurso()
          },
        }))
        filas.append(fila)
      }
      if (!plan.config[clave].length) filas.append(el('p', { class: 'crono-baleiro', text: t('actas.lista.baleira', 'Aínda non hai ningún.') }))
    }
    caixa.append(filas, el('div', {}, el('button', {
      type: 'button', class: 'crono-btn crono-btn--pequeno', html: ICONAS.mais,
      onclick: () => { aplicar({ ...plan, config: { ...plan.config, [clave]: [...plan.config[clave], novo()] } }); pintar(); renderMeses(); actualizarResumoCurso() },
    }, el('span', { text: engadirTexto }))))
    pintar()
    return caixa
  }

  function renderConfig() {
    const c = plan.config
    refs.erroDatas = el('p', { class: 'crono-erro', role: 'alert', hidden: true })
    refs.resumoCurso = el('span', { class: 'crono-config__resumo' })
    const bloqueo = el('input', {
      type: 'checkbox', checked: plan.isLocked || undefined,
      onchange: (e) => { aplicar({ ...plan, isLocked: e.target.checked }); render() },
    })

    const detalles = el('details', { class: 'crono-config' },
      el('summary', {}, el('span', { class: 'crono-config__titulo', text: t('actas.config.titulo', 'Datos do departamento e materias impartidas') }), refs.resumoCurso),
      el('div', { class: 'crono-config__corpo' },
        el('div', { class: 'crono-fila' },
          camp(t('informes.curso', 'Curso académico'), el('input', {
            type: 'text', value: plan.academicYear, placeholder: '2025/2026',
            oninput: (e) => { aplicar({ ...plan, academicYear: e.target.value }, 'texto'); actualizarResumoCurso() },
          }), 'curto'),
          camp(t('informes.inicio', 'Inicio'), el('input', { type: 'date', value: c.startDate, onchange: (e) => cambioData('startDate', e.target.value) }), 'data'),
          camp(t('informes.fin', 'Fin'), el('input', { type: 'date', value: c.endDate, onchange: (e) => cambioData('endDate', e.target.value) }), 'data'),
        ),
        refs.erroDatas,
        camp(t('informes.centro', 'Nome do centro'), entrada('schoolName', { placeholder: t('informes.centro.ph', 'IES …') })),
        camp(t('informes.departamento', 'Departamento'), entrada('department', { placeholder: t('informes.departamento.ph', 'Departamento de Matemáticas') })),
        el('div', { class: 'crono-fila' },
          camp(t('informes.localidade', 'Localidade'), entrada('location')),
          camp(t('informes.xefe', 'Xefe/a do departamento'), entrada('headOfDepartment')),
          camp(t('actas.secretario', 'Secretario/a (redacta as actas)'), entrada('secretary')),
        ),
        camp(t('informes.cargo', 'Cargo que figura na sinatura'), entrada('signatoryRole', { placeholder: t('informes.cargo.ph', 'Xefe/a de Departamento') })),
        el('div', { class: 'crono-fila' },
          camp(t('actas.hora.habitual', 'Hora habitual das reunións'), entrada('meetingTime', { tipo: 'time' }), 'data'),
          camp(t('actas.lugar.habitual', 'Lugar habitual'), entrada('meetingPlace', { placeholder: t('actas.lugar.ph', 'Sala de profesorado') })),
        ),
        listaEditable({
          clave: 'members', titulo: t('actas.membros', 'Membros do departamento'), engadirTexto: t('actas.membro.engadir', 'Engadir membro'),
          novo: novoMembro, eliminar: eliminarMembro,
          campos: [['name', t('actas.membro.nome', 'Nome e apelidos'), ''], ['role', t('actas.membro.cargo', 'Cargo (opcional)'), 'curto']],
        }),
        listaEditable({
          clave: 'subjects', titulo: t('actas.materias', 'Materias impartidas'), engadirTexto: t('actas.materia.engadir', 'Engadir materia'),
          novo: novaMateria, eliminar: eliminarMateria,
          campos: [['name', t('actas.materia.nome', 'Materia'), ''], ['teacher', t('actas.materia.docente', 'Docente'), ''], ['groups', t('actas.materia.grupos', 'Grupos (ex.: 1º ESO A)'), 'curto']],
        }),
        el('label', { class: 'crono-camp crono-camp--check' }, bloqueo,
          el('span', { text: t('informes.bloquear', 'Bloquear a edición deste curso') })),
      ),
    )
    // Un curso novo (sen departamento nin actas) ábrese co formulario á vista.
    detalles.open = totalActas(plan) === 0 && !plan.config.department
    actualizarErroDatas()
    actualizarResumoCurso()
    return detalles
  }

  // ---------- Meses: a acta e os informes das materias ----------

  function renderActa(mes, num) {
    const bloq = plan.isLocked
    const a = mes.acta
    const data = dataCurta(a.date)
    const titulo = t('actas.acta.titulo', 'Acta n.º {n}', { n: num }) + (data ? ' · ' + data : '')
    const cuerpo = resumo(a.agenda || a.development || a.agreements)
    return el('div', { class: 'crono-tarefa actas-acta', 'data-acta': mes.clave },
      el('button', {
        type: 'button', class: 'crono-tarefa__corpo', onclick: () => abrirActa(mes.clave),
        title: bloq ? t('actas.acta.ver', 'Ver a acta') : t('actas.acta.editar', 'Editar a acta'),
      },
      el('span', { class: 'crono-tarefa__titulo', text: titulo }),
      cuerpo ? el('span', { class: 'crono-tarefa__resumo', text: cuerpo }) : null),
      el('span', { class: 'crono-tarefa__accions' },
        el('button', {
          type: 'button', class: 'crono-icona', html: ICONAS.lixo, title: t('informes.eliminar', 'Eliminar'), 'aria-label': t('informes.eliminar', 'Eliminar'),
          disabled: bloq || undefined,
          onclick: async (e) => {
            e.stopPropagation()
            const ok = await confirmar(t('actas.acta.eliminarConfirmar', 'Eliminar a acta de {mes}? Os informes das materias consérvanse.', { mes: mes.nome }),
              { aceptar: t('informes.eliminar', 'Eliminar'), cancelar: t('dialog.cancel', 'Cancelar') })
            if (ok) estruturar(eliminarActa(plan, mes.clave))
          },
        })))
  }

  function renderSeguimentos(mes) {
    const materias = plan.config.subjects
    if (!materias.length) {
      return el('p', { class: 'actas-aviso', text: t('actas.sen.materias', 'Engade as materias do departamento nos datos de arriba para crear os seus informes de seguimento.') })
    }
    return el('div', { class: 'actas-materias' },
      materias.map((m) => {
        const informe = mes.reports[m.id]
        const feito = !seguimentoBaleiro(informe)
        const nome = m.name || t('actas.sen.nome', 'Sen nome')
        return el('button', {
          type: 'button', class: 'actas-materia' + (feito ? ' actas-materia--feita' : ''), 'data-materia': m.id,
          title: t('actas.informe.editar', 'Informe de seguimento de {materia}', { materia: nome }),
          onclick: () => abrirSeguimento(mes.clave, m.id),
        },
        el('span', { class: 'actas-materia__nome', text: nome }),
        informe && informe.progress !== undefined ? el('span', { class: 'actas-materia__pct', text: `${informe.progress}%` }) : null)
      }))
  }

  function renderMes(mes, actual, num) {
    const nSeg = Object.keys(mes.reports).filter((id) => !seguimentoBaleiro(mes.reports[id])).length
    return el('section', {
      class: 'crono-mes actas-mes' + (mes.clave === actual ? ' crono-mes--actual' : ''), 'data-mes': mes.clave,
      style: `--cor-mes:${['#DBEAFE', '#CFFAFE', '#ECFDF5', '#D1FAE5', '#DCFCE7', '#FEFCE8', '#FEF9C3', '#FFEDD5', '#FFF7ED', '#FEF3C7', '#E7E5E4', '#E2E8F0'][mes.mes]}`,
    },
    el('header', { class: 'crono-mes__cab' },
      el('h3', {}, el('span', { text: mes.nome }), el('small', { text: String(mes.ano) }),
        mes.clave === actual ? el('em', { class: 'crono-mes__actual', text: t('informes.mes.actual', 'Mes actual') }) : null),
      el('span', { class: 'crono-mes__n', text: plan.config.subjects.length ? `${nSeg}/${plan.config.subjects.length}` : '' }),
      mes.acta ? null : el('button', {
        type: 'button', class: 'crono-btn crono-btn--pequeno', html: ICONAS.mais,
        disabled: plan.isLocked || undefined, onclick: () => abrirActa(mes.clave),
      }, el('span', { text: t('actas.acta.engadir', 'Engadir acta') }))),
    el('div', { class: 'actas-mes__corpo' },
      mes.acta ? renderActa(mes, num) : null,
      renderSeguimentos(mes)))
  }

  function renderMeses() {
    if (!refs.meses) return
    const meses = mesesDoCurso(plan)
    refs.meses.replaceChildren()
    const fora = actasForaDeRango(plan)
    if (fora > 0) {
      refs.meses.append(el('p', { class: 'crono-aviso', text: t('actas.fora', '{n} mes(es) con actas ou informes están fóra do rango de datas e non sairán nos documentos.', { n: fora }) }))
    }
    if (meses.length === 0 || meses.length > MAX_MESES) {
      refs.meses.append(el('p', { class: 'crono-baleiro', text: t('informes.sen.meses', 'Indica un rango de datas válido para ver os meses.') }))
      return
    }
    const actual = claveMes(hoxe().getFullYear(), hoxe().getMonth())
    let num = 0
    for (const m of meses) {
      if (m.acta) num++
      refs.meses.append(renderMes(m, actual, num))
    }
  }

  function render() {
    if (destruido) return
    container.replaceChildren()
    if (!plan) return
    container.classList.toggle('crono-editor--bloqueado', !!plan.isLocked)
    refs.meses = el('div', { class: 'crono-meses' })
    container.append(renderConfig(), refs.meses)
    if (plan.isLocked) {
      container.querySelectorAll('.crono-config__corpo input:not([type=checkbox]), .crono-config__corpo textarea, .crono-config__corpo button').forEach((n) => { n.disabled = true })
    }
    renderMeses()
  }

  // ---------- Pantallas de edición ----------

  // pantalla monta unha pantalla completa (cabeceira coa X, corpo e pé con
  // Cancelar / Aplicar / Gardar) sobre esta xanela. `ler()` devolve o que se
  // garda; `aoGardar(datos)` mételo no curso.
  function pantalla({ titulo, corpo, instantanea, aoGardar, extraPe = [] }) {
    const overlay = el('div', { class: 'overlay overlay--dialog crono-overlay tarefa-overlay actas-overlay' })
    const modal = el('div', { class: 'modal crono-modal tarefa-pantalla', role: 'dialog', 'aria-modal': 'true' })
    overlay.append(modal)
    document.body.append(overlay)
    let inicial = instantanea()
    const estado = el('span', { class: 'tarefa-cab__estado', 'aria-live': 'polite' })
    const cambiado = () => instantanea() !== inicial
    const pechar = () => { document.removeEventListener('keydown', onKey, true); overlay.remove() }
    const cancelar = async () => {
      if (cambiado() && !await confirmar(t('informes.descartarCambios', 'Hai cambios sen gardar. Descartalos?'),
        { aceptar: t('informes.descartar', 'Descartar'), cancelar: t('informes.seguir', 'Seguir editando') })) return
      pechar()
    }
    const gardar = async (pecharDespois) => {
      try {
        await aoGardar()
        inicial = instantanea()
        estado.textContent = t('plantillas.aplicado', '✓ Gardado')
        if (pecharDespois) pechar()
      } catch (err) {
        estado.textContent = String(err?.message || err)
      }
    }
    function onKey(e) {
      if (document.querySelector('.overlay--dialog:not(.actas-overlay)')) return
      if (e.key === 'Escape') { e.preventDefault(); e.stopPropagation(); cancelar() }
      else if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) { e.preventDefault(); e.stopPropagation(); gardar(true) }
      else if ((e.key === 's' || e.key === 'S') && (e.ctrlKey || e.metaKey)) { e.preventDefault(); e.stopPropagation(); gardar(false) }
    }
    document.addEventListener('keydown', onKey, true)
    const botonX = el('button', { type: 'button', class: 'tarefa-x', html: ICONA_X, title: t('informes.tarefa.pechar', 'Pechar'), 'aria-label': t('informes.tarefa.pechar', 'Pechar'), onclick: cancelar })
    const pe = el('div', { class: 'modal-actions tarefa-pe' },
      ...extraPe,
      el('button', { type: 'button', text: t('dialog.cancel', 'Cancelar'), onclick: cancelar }),
      plan.isLocked ? null : el('button', { type: 'button', class: 'tarefa-aplicar', text: t('plantillas.apply', 'Aplicar'), onclick: () => gardar(false) }),
      plan.isLocked ? null : el('button', { type: 'button', class: 'primary', text: t('plantillas.save', 'Gardar'), onclick: () => gardar(true) }))
    modal.append(
      el('header', { class: 'tarefa-cab' }, el('h2', { class: 'tarefa-cab__titulo', text: titulo }), estado, botonX),
      el('div', { class: 'tarefa-medio' }, el('div', { class: 'tarefa-corpo actas-corpo' }, corpo)),
      pe)
    return { pechar }
  }

  const area = (valor, placeholder, filas = 5) => {
    const a = el('textarea', { class: 'actas-md', rows: String(filas), spellcheck: 'true', placeholder, disabled: plan.isLocked || undefined })
    a.value = valor || ''
    return a
  }

  function abrirActa(clave) {
    const mes = mesesDoCurso(plan).find((m) => m.clave === clave)
    if (!mes) return
    const nova = !mes.acta
    const a = { ...(mes.acta || novaActa(plan.config)) }
    const bloq = plan.isLocked
    const num = mesesDoCurso(plan).filter((m) => m.acta || m.clave === clave).findIndex((m) => m.clave === clave) + 1
    const present = new Set(a.present)

    const data = el('input', { type: 'date', value: a.date, disabled: bloq || undefined })
    const inicio = el('input', { type: 'time', value: a.startTime, disabled: bloq || undefined })
    const fin = el('input', { type: 'time', value: a.endTime, disabled: bloq || undefined })
    const lugar = el('input', { type: 'text', value: a.place, placeholder: t('actas.lugar.ph', 'Sala de profesorado'), disabled: bloq || undefined })
    const orde = area(a.agenda, t('actas.orde.ph', 'Orde do día en Markdown:\n1. Lectura da acta anterior\n2. Seguimento das programacións\n3. Rogos e preguntas'), 6)
    const desenvolvemento = area(a.development, t('actas.desenvolvemento.ph', 'Desenvolvemento da sesión'), 8)
    const acordos = area(a.agreements, t('actas.acordos.ph', 'Acordos adoptados (unha lista con -)'), 5)
    const rogos = area(a.questions, t('actas.rogos.ph', 'Rogos e preguntas'), 4)

    const asistentes = el('div', { class: 'actas-asistentes' })
    const pintarAsistentes = () => {
      asistentes.replaceChildren()
      if (!plan.config.members.length) {
        asistentes.append(el('p', { class: 'actas-aviso', text: t('actas.sen.membros', 'Engade os membros do departamento nos datos do curso para marcar quen asiste.') }))
        return
      }
      for (const m of plan.config.members) {
        asistentes.append(el('label', { class: 'actas-asistente' },
          el('input', {
            type: 'checkbox', checked: present.has(m.id) || undefined, disabled: bloq || undefined,
            onchange: (e) => { if (e.target.checked) present.add(m.id); else present.delete(m.id) },
          }),
          el('span', { text: m.name || t('actas.sen.nome', 'Sen nome') + (m.role ? ` (${m.role})` : '') })))
      }
    }
    pintarAsistentes()
    const todos = (marcar) => {
      present.clear()
      if (marcar) plan.config.members.forEach((m) => present.add(m.id))
      pintarAsistentes()
    }

    // Os informes das materias deste mes, só de lectura: edítanse na lista do mes.
    const seguimento = el('div', { class: 'actas-resumo-seg' })
    if (!plan.config.subjects.length) seguimento.append(el('p', { class: 'actas-aviso', text: t('actas.sen.materias.curto', 'Non hai materias configuradas.') }))
    for (const m of plan.config.subjects) {
      const inf = mes.reports[m.id]
      const { teacher } = docenteEGrupos(m, inf)
      const linha = seguimentoBaleiro(inf)
        ? t('actas.seguimento.sen', 'Sen informe')
        : [inf.progress !== undefined ? `${inf.progress}%` : '', resumo(inf.contents || inf.results || inf.difficulties || inf.measures, 80)].filter(Boolean).join(' · ')
      seguimento.append(el('div', { class: 'actas-resumo-seg__fila' },
        el('strong', { text: m.name || t('actas.sen.nome', 'Sen nome') }),
        el('span', { class: 'actas-resumo-seg__doc', text: teacher }),
        el('span', { class: 'actas-resumo-seg__txt', text: linha })))
    }

    const corpo = el('div', { class: 'actas-form' },
      el('div', { class: 'crono-fila' },
        camp(t('actas.data', 'Data'), data, 'data'),
        camp(t('actas.hora.inicio', 'Hora de inicio'), inicio, 'data'),
        camp(t('actas.hora.fin', 'Hora de fin'), fin, 'data'),
        camp(t('actas.lugar', 'Lugar'), lugar)),
      el('div', { class: 'crono-camp' },
        el('span', { class: 'crono-camp__etq', text: t('actas.asistentes', 'Asistentes') }),
        asistentes,
        bloq ? null : el('div', { class: 'actas-asistentes__acc' },
          el('button', { type: 'button', class: 'crono-btn crono-btn--pequeno', text: t('actas.todos', 'Todos'), onclick: () => todos(true) }),
          el('button', { type: 'button', class: 'crono-btn crono-btn--pequeno', text: t('actas.ninguen', 'Ninguén'), onclick: () => todos(false) }))),
      camp(t('actas.orde', 'Orde do día'), orde),
      el('div', { class: 'crono-camp' },
        el('span', { class: 'crono-camp__etq', text: t('actas.seguimento', 'Seguimento das materias (informes deste mes)') }),
        seguimento),
      camp(t('actas.desenvolvemento', 'Desenvolvemento da sesión'), desenvolvemento),
      camp(t('actas.acordos', 'Acordos'), acordos),
      camp(t('actas.rogos', 'Rogos e preguntas'), rogos))

    const ler = () => ({
      ...a, date: data.value, startTime: inicio.value, endTime: fin.value, place: lugar.value,
      present: plan.config.members.map((m) => m.id).filter((id) => present.has(id)),
      agenda: orde.value, development: desenvolvemento.value, agreements: acordos.value, questions: rogos.value,
    })
    pantalla({
      titulo: t('actas.acta.pantalla', 'Acta de {mes} {ano}', { mes: mes.nome, ano: mes.ano }) + (nova ? ' · ' + t('actas.nova', 'nova') : '') + ` · n.º ${num}`,
      corpo, instantanea: () => JSON.stringify(ler()),
      aoGardar: () => { estruturar(gardarActa(plan, clave, ler())) },
    })
    // Unha acta nova non existe no curso ata que se garda.
  }

  function abrirSeguimento(clave, idMateria) {
    const mes = mesesDoCurso(plan).find((m) => m.clave === clave)
    const materia = plan.config.subjects.find((m) => m.id === idMateria)
    if (!mes || !materia) return
    const inf = mes.reports[idMateria] || {}
    const bloq = plan.isLocked
    const docente = el('input', { type: 'text', value: inf.teacher || '', placeholder: materia.teacher || t('actas.materia.docente', 'Docente'), disabled: bloq || undefined })
    const grupos = el('input', { type: 'text', value: inf.groups || '', placeholder: materia.groups || t('actas.materia.grupos', 'Grupos'), disabled: bloq || undefined })
    const pct = el('input', {
      type: 'number', min: '0', max: '100', step: '5', value: inf.progress !== undefined ? String(inf.progress) : '',
      placeholder: '—', disabled: bloq || undefined,
    })
    const contidos = area(inf.contents, t('actas.contidos.ph', 'Unidades e contidos traballados este mes'), 5)
    const resultados = area(inf.results, t('actas.resultados.ph', 'Resultados da avaliación e observacións'), 5)
    const dificultades = area(inf.difficulties, t('actas.dificultades.ph', 'Dificultades detectadas'), 4)
    const medidas = area(inf.measures, t('actas.medidas.ph', 'Medidas adoptadas e propostas de mellora'), 4)
    const corpo = el('div', { class: 'actas-form' },
      el('div', { class: 'crono-fila' },
        camp(t('actas.materia.docente', 'Docente'), docente),
        camp(t('actas.materia.grupos', 'Grupos'), grupos),
        camp(t('actas.cumprimento', 'Grao de cumprimento da programación (%)'), pct, 'data')),
      camp(t('actas.contidos', 'Contidos traballados'), contidos),
      camp(t('actas.resultados', 'Resultados e observacións'), resultados),
      camp(t('actas.dificultades', 'Dificultades detectadas'), dificultades),
      camp(t('actas.medidas', 'Medidas e propostas de mellora'), medidas))
    const ler = () => ({
      teacher: docente.value, groups: grupos.value, progress: pct.value === '' ? undefined : Number(pct.value),
      contents: contidos.value, results: resultados.value, difficulties: dificultades.value, measures: medidas.value,
    })
    let cerrar = null
    const limpar = el('button', {
      type: 'button', class: 'crono-btn crono-btn--suave', text: t('actas.informe.limpar', 'Baleirar o informe'), disabled: bloq || undefined,
      onclick: async () => {
        const ok = await confirmar(t('actas.informe.limparConfirmar', 'Baleirar o informe de {materia} deste mes?', { materia: materia.name }),
          { aceptar: t('actas.informe.baleirar', 'Baleirar'), cancelar: t('dialog.cancel', 'Cancelar') })
        if (!ok) return
        estruturar(gardarSeguimento(plan, clave, idMateria, {}))
        cerrar?.pechar()
      },
    })
    cerrar = pantalla({
      titulo: t('actas.informe.pantalla', 'Informe de seguimento: {materia} · {mes} {ano}', { materia: materia.name || t('actas.sen.nome', 'Sen nome'), mes: mes.nome, ano: mes.ano }),
      corpo, instantanea: () => JSON.stringify(ler()), extraPe: [limpar],
      aoGardar: () => { estruturar(gardarSeguimento(plan, clave, idMateria, ler())) },
    })
  }

  // ---------- API ----------

  return {
    getValue: () => plan,
    // setValue substitúe o curso enteiro (abrir, novo, desfacer...) e NON
    // avisa a onChange: quen o chama sabe que cambiou.
    setValue(novo) { plan = novo; render() },
    focus() { const p = container.querySelector('input, button'); if (p) p.focus() },
    refrescar: render,
    abrirActa,
    abrirSeguimento,
    destroy() { destruido = true; container.replaceChildren(); container.classList.remove('crono-editor') },
    // Só para probas.
    _totais: () => ({ actas: totalActas(plan), informes: totalSeguimentos(plan) }),
  }
}
