// createCronogramaEditor - a interface do informe "Cronograma anual": datos do
// curso e, por cada mes, as súas tarefas. Substitúe (no modo Cronograma) ao
// editor de exames da parte esquerda de Lua.
//
// Mesmo contrato "factoría illada" ca editor.js/explorer.js: recibe o
// contedor e unhas dependencias (t, onChange, lerImaxe), nunca fala cos
// bindings Wails - iso é cousa de informes/controlador.js. O estado é un
// PlanCrono (ver cronograma-modelo.js) e NUNCA se muta: cada cambio constrúe
// un plan novo e avisa a onChange(plan, motivo), onde `motivo` é 'texto'
// (tecleo nun campo, o controlador agrupa estes para desfacer) ou
// 'estrutura' (engadir/borrar/mover...).
import { confirmar } from '../comun/dialogs.js'
import {
  PALETA, COR_MES, MAX_MESES, resolverCor, calendario, claveMes, validarDatas,
  novaTarefa, gardarTarefa, eliminarTarefa, moverTarefa, duplicarTarefa, moverTarefaAMes,
  totalTarefas, tarefasForaDeRango,
} from './cronograma-modelo.js'
import { el } from './dom.js'
import { montarEditorTarefa, normalizarImaxe } from './tarefa-editor.js'

export { normalizarImaxe }

const ICONAS = {
  arriba: '<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12l5-5 5 5"/></svg>',
  abaixo: '<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M5 8l5 5 5-5"/></svg>',
  copiar: '<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><rect x="7" y="7" width="9" height="9" rx="1.4"/><path d="M4 13V5.4A1.4 1.4 0 0 1 5.4 4H13"/></svg>',
  editar: '<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M4 16 14.5 5.5a1.4 1.4 0 0 1 2 2L6 18l-3 1z"/></svg>',
  lixo: '<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M4.5 5.5h11"/><path d="M8 5.5V4a1 1 0 0 1 1-1h2a1 1 0 0 1 1 1v1.5"/><path d="M5.5 5.5 6.2 16a1 1 0 0 0 1 .9h5.6a1 1 0 0 0 1-.9l.7-10.5"/></svg>',
  mais: '<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"><path d="M10 4.5v11M4.5 10h11"/></svg>',
}

// resumo: as primeiras palabras da descrición sen marcas Markdown, para
// recoñecer unha tarefa na lista sen abrila.
function resumo(desc, max = 110) {
  const plano = String(desc)
    .replace(/^:::.*$/gm, ' ').replace(/!\[[^\]]*\]\([^)]*\)/g, ' ')
    .replace(/\[([^\]]*)\]\([^)]*\)/g, '$1').replace(/\[\/?fs[^\]]*\]/g, '')
    .replace(/^[#>\-*\d. ]+/gm, '').replace(/[*_]/g, '')
    .replace(/\s+/g, ' ').trim()
  return plano.length > max ? plano.slice(0, max - 1) + '…' : plano
}

// abrirXanelaTarefa (opcional): async (datos) -> true se a tarefa se abriu
// nunha xanela propia do sistema (que se pode levar a outro monitor); false
// ou un erro -> ábrese a pantalla completa dentro desta mesma xanela.
// asistenteTarefa (opcional): {api, idioma} co asistente de IA da pantalla da tarefa
// (ver tarefa-editor.js); sen el a pantalla non leva o botón do asistente.
export function createCronogramaEditor(container, {
  t: tIn, onChange, lerImaxe = normalizarImaxe, hoxe = () => new Date(), abrirXanelaTarefa = null, asistenteTarefa = null,
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
  // estruturar: un cambio que altera a lista de meses/tarefas (a lista e o
  // resumo do curso hai que repintalos; os campos do formulario non).
  const estruturar = (novo) => { aplicar(novo); renderMeses(); actualizarResumoCurso() }

  // ---------- Datos do curso ----------

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

  function actualizarErroDatas() {
    const erro = validarDatas(plan.config.startDate, plan.config.endDate)
    const n = calendario(plan.config, plan.monthsData).length
    refs.erroDatas.textContent = erro || (n > MAX_MESES
      ? t('informes.datas.demasiados', 'O rango ten {n} meses (máximo {max}). Revisa as datas.', { n, max: MAX_MESES })
      : '')
    refs.erroDatas.hidden = !refs.erroDatas.textContent
  }

  function cambioData(campo, valor) {
    // Un <input type=date> incompleto dá '' - non se pisa a data boa mentres
    // o usuario aínda está escribindo.
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
    const meses = calendario(plan.config, plan.monthsData)
    const n = totalTarefas(plan)
    const rango = meses.length ? `${meses[0].nome} ${meses[0].ano} – ${meses[meses.length - 1].nome} ${meses[meses.length - 1].ano}` : t('informes.datas.sen', 'Sen rango de datas válido')
    refs.resumoCurso.textContent = `${plan.academicYear || ''} · ${rango} · ${t(n === 1 ? 'informes.tarefas.unha' : 'informes.tarefas.n', n === 1 ? '1 tarefa' : '{n} tarefas', { n })}`
  }

  function renderConfig() {
    const c = plan.config
    refs.erroDatas = el('p', { class: 'crono-erro', role: 'alert', hidden: true })
    refs.resumoCurso = el('span', { class: 'crono-config__resumo' })

    const bloqueo = el('input', {
      type: 'checkbox', checked: plan.isLocked || undefined,
      onchange: (e) => { aplicar({ ...plan, isLocked: e.target.checked }); render() },
    })

    const introTextarea = el('textarea', {
      rows: '4', placeholder: t('informes.intro.placeholder', 'Se está baleiro, úsase un texto por defecto.'),
      oninput: (e) => mutarConfig({ introText: e.target.value }, 'texto'),
    })
    introTextarea.value = c.introText

    const detalles = el('details', { class: 'crono-config' },
      el('summary', {}, el('span', { class: 'crono-config__titulo', text: t('informes.config.titulo', 'Datos do curso, do centro e do departamento') }), refs.resumoCurso),
      el('div', { class: 'crono-config__corpo' },
        el('div', { class: 'crono-fila' },
          camp(t('informes.curso', 'Curso académico'), el('input', {
            type: 'text', value: plan.academicYear, placeholder: '2025/2026',
            oninput: (e) => { aplicar({ ...plan, academicYear: e.target.value }, 'texto'); actualizarResumoCurso() },
          }), 'curto'),
          camp(t('informes.inicio', 'Inicio'), el('input', {
            type: 'date', value: c.startDate, onchange: (e) => cambioData('startDate', e.target.value),
          }), 'data'),
          camp(t('informes.fin', 'Fin'), el('input', {
            type: 'date', value: c.endDate, onchange: (e) => cambioData('endDate', e.target.value),
          }), 'data'),
        ),
        refs.erroDatas,
        camp(t('informes.centro', 'Nome do centro'), entrada('schoolName', { placeholder: t('informes.centro.ph', 'IES …') })),
        camp(t('informes.departamento', 'Departamento'), entrada('department', { placeholder: t('informes.departamento.ph', 'Departamento de Matemáticas') })),
        el('div', { class: 'crono-fila' },
          camp(t('informes.localidade', 'Localidade'), entrada('location')),
          camp(t('informes.xefe', 'Xefe/a do departamento'), entrada('headOfDepartment')),
        ),
        camp(t('informes.cargo', 'Cargo que figura na sinatura'), entrada('signatoryRole', { placeholder: t('informes.cargo.ph', 'Xefe/a de Departamento') })),
        camp(t('informes.intro', 'Texto da introdución'), introTextarea),
        el('label', { class: 'crono-camp crono-camp--check' }, bloqueo,
          el('span', { text: t('informes.bloquear', 'Bloquear a edición deste curso') })),
      ),
    )
    // Un curso novo (sen tarefas nin centro) ábrese co formulario á vista.
    detalles.open = totalTarefas(plan) === 0 && !plan.config.schoolName
    refs.detalles = detalles
    actualizarErroDatas()
    actualizarResumoCurso()
    return detalles
  }

  // ---------- Meses e tarefas ----------

  function renderTarefa(mes, tarefa, i, total) {
    const cor = resolverCor(tarefa.color)
    const bloq = plan.isLocked
    const boton = (icona, titulo, fn, desactivado) => el('button', {
      type: 'button', class: 'crono-icona', html: icona, title: titulo, 'aria-label': titulo,
      disabled: (desactivado || bloq) || undefined, onclick: (e) => { e.stopPropagation(); fn() },
    })
    return el('li', {
      class: 'crono-tarefa', 'data-id': tarefa.id,
      style: `--tf:${cor.fondo};--tt:${cor.texto};--tb:${cor.bordo}`,
    },
    el('button', {
      type: 'button', class: 'crono-tarefa__corpo', onclick: () => abrirTarefa(mes.clave, tarefa),
      title: bloq ? t('informes.tarefa.ver', 'Ver a tarefa') : t('informes.tarefa.editar', 'Editar a tarefa'),
    },
    el('span', { class: 'crono-tarefa__titulo', text: tarefa.title || t('informes.tarefa.sen', 'Tarefa sen título') }),
    resumo(tarefa.description) ? el('span', { class: 'crono-tarefa__resumo', text: resumo(tarefa.description) }) : null),
    el('span', { class: 'crono-tarefa__accions' },
      boton(ICONAS.arriba, t('informes.tarefa.subir', 'Subir'), () => estruturar(moverTarefa(plan, mes.clave, tarefa.id, -1)), i === 0),
      boton(ICONAS.abaixo, t('informes.tarefa.baixar', 'Baixar'), () => estruturar(moverTarefa(plan, mes.clave, tarefa.id, 1)), i === total - 1),
      boton(ICONAS.copiar, t('informes.tarefa.duplicar', 'Duplicar'), () => estruturar(duplicarTarefa(plan, mes.clave, tarefa.id))),
      boton(ICONAS.lixo, t('informes.tarefa.eliminar', 'Eliminar'), async () => {
        const ok = await confirmar(t('informes.tarefa.eliminarConfirmar', 'Eliminar a tarefa «{titulo}»?', { titulo: tarefa.title || t('informes.tarefa.sen', 'Tarefa sen título') }),
          { aceptar: t('informes.eliminar', 'Eliminar'), cancelar: t('dialog.cancel', 'Cancelar') })
        if (ok) estruturar(eliminarTarefa(plan, mes.clave, tarefa.id))
      }),
    ))
  }

  function renderMes(mes, actual) {
    const tarefas = mes.tarefas
    const n = tarefas.length
    return el('section', {
      class: 'crono-mes' + (mes.clave === actual ? ' crono-mes--actual' : ''), 'data-mes': mes.clave,
      style: `--cor-mes:${COR_MES[mes.mes]}`,
    },
    el('header', { class: 'crono-mes__cab' },
      el('h3', {}, el('span', { text: mes.nome }), el('small', { text: String(mes.ano) }),
        mes.clave === actual ? el('em', { class: 'crono-mes__actual', text: t('informes.mes.actual', 'Mes actual') }) : null),
      el('span', { class: 'crono-mes__n', text: n ? t(n === 1 ? 'informes.tarefas.unha' : 'informes.tarefas.n', n === 1 ? '1 tarefa' : '{n} tarefas', { n }) : '' }),
      el('button', {
        type: 'button', class: 'crono-btn crono-btn--pequeno', html: ICONAS.mais,
        disabled: plan.isLocked || undefined, onclick: () => abrirTarefa(mes.clave, null),
      }, el('span', { text: t('informes.tarefa.engadir', 'Engadir tarefa') }))),
    n ? el('ul', { class: 'crono-tarefas' }, tarefas.map((tf, i) => renderTarefa(mes, tf, i, n))) : null)
  }

  function renderMeses() {
    if (!refs.meses) return
    const meses = calendario(plan.config, plan.monthsData)
    refs.meses.replaceChildren()
    const fora = tarefasForaDeRango(plan)
    if (fora > 0) {
      refs.meses.append(el('p', { class: 'crono-aviso', text: t('informes.fora', '{n} tarefa(s) están en meses fóra do rango de datas e non sairán no informe. Amplía o rango para velas.', { n: fora }) }))
    }
    if (meses.length === 0 || meses.length > MAX_MESES) {
      refs.meses.append(el('p', { class: 'crono-baleiro', text: t('informes.sen.meses', 'Indica un rango de datas válido para ver os meses.') }))
      return
    }
    const actual = claveMes(hoxe().getFullYear(), hoxe().getMonth())
    for (const m of meses) refs.meses.append(renderMes(m, actual))
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

  // ---------- Pantalla de tarefa ----------

  // datosTarefa reúne o que precisa a pantalla "Editar tarefa", esteña onde
  // esteña (só datos serializables: tamén viaxan á xanela propia).
  function datosTarefa(clave, tarefaOrixe) {
    return {
      nova: !tarefaOrixe,
      tarefa: tarefaOrixe ? { ...tarefaOrixe } : novaTarefa(PALETA[0].clave),
      clave,
      meses: calendario(plan.config, plan.monthsData).map((m) => ({ clave: m.clave, nome: m.nome, ano: m.ano })),
      bloq: !!plan.isLocked,
      // Para o asistente da tarefa (a normativa depende do centro e do curso).
      contexto: { centro: plan.config.schoolName || '', localidade: plan.config.location || '', anoAcademico: plan.academicYear || '' },
    }
  }

  // gardarTarefaNoPlan mete a tarefa no curso (e cámbiaa de mes se cómpre).
  // Serve para a pantalla desta xanela e para a xanela propia (que avisa por
  // aplicarTarefaExterna).
  function gardarTarefaNoPlan({ tarefa, claveOrixe, mesDestino }) {
    let novo = gardarTarefa(plan, claveOrixe, tarefa)
    if (mesDestino !== claveOrixe) novo = moverTarefaAMes(novo, claveOrixe, mesDestino, tarefa.id)
    estruturar(novo)
  }

  function abrirTarefaNestaXanela(datos) {
    const overlay = el('div', { class: 'overlay overlay--dialog crono-overlay tarefa-overlay' })
    const modal = el('div', { class: 'modal crono-modal' })
    overlay.append(modal)
    document.body.append(overlay)
    montarEditorTarefa(modal, {
      t, lerImaxe, ...datos, asistente: asistenteTarefa,
      aoAplicar: (r) => gardarTarefaNoPlan(r),
      aoPechar: () => overlay.remove(),
    })
  }

  function abrirTarefa(clave, tarefaOrixe) {
    const datos = datosTarefa(clave, tarefaOrixe)
    if (!abrirXanelaTarefa) { abrirTarefaNestaXanela(datos); return }
    Promise.resolve()
      .then(() => abrirXanelaTarefa(datos))
      .then((aberta) => { if (!aberta) abrirTarefaNestaXanela(datos) })
      .catch(() => abrirTarefaNestaXanela(datos))
  }

  // ---------- API ----------

  return {
    getValue: () => plan,
    // setValue substitúe o curso enteiro (abrir, novo, desfacer...) e NON
    // avisa a onChange: quen o chama sabe que cambiou.
    setValue(novo) { plan = novo; render() },
    focus() { const p = container.querySelector('input, button'); if (p) p.focus() },
    // refrescar volve pintar todo (p.ex. ao cambiar o idioma).
    refrescar: render,
    // aplicarTarefaExterna recibe unha tarefa gardada dende a xanela propia
    // de edición (ou a fai o asistente) e métea no curso. NON avisa a
    // onChange coma un cambio de texto: é estrutura e entra no historial.
    aplicarTarefaExterna: gardarTarefaNoPlan,
    // Abre a pantalla de edición dunha tarefa existente ou nova (clave do mes).
    abrirTarefa,
    destroy() { destruido = true; container.replaceChildren(); container.classList.remove('crono-editor') },
  }
}
