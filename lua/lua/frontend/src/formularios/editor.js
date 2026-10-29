// createFormularioEditor - a pantalla de datos dun formulario propio: xérase
// do esquema (un control por campo) e deixa encher os datos á man ou, nas
// táboas, importar un CSV. Máis arriba están os datos do documento (centro,
// curso, quen asina, logotipo), que enchen as variables da plantilla.
//
// Mesmo contrato ca cronograma-editor.js e actas-editor.js: factoría illada
// (recibe o contedor e as dependencias, nunca fala cos bindings), estado
// inmutable e `onChange(plan, motivo)` con motivo 'texto' (tecleo: o
// controlador agrupa estes para desfacer) ou 'estrutura'.
import { el } from '../informes/dom.js'
import { normalizarImaxe } from '../informes/tarefa-editor.js'
import { decodificarCSV, parseCSV, taboaDeCSV, amosarReferencias } from './modelo.js'
import { crearPanelAsistente } from '../asistente/panel.js'
import { crearFerramentasFormularioDatos } from '../asistente/ferramentas-formulario-datos.js'
import { TAREFAS_FORMULARIO_DATOS } from '../asistente/tarefas.js'
import { logoProvedor } from '../asistente/logos.js'
import { crearControlProblemas, actualizarVistasProblemas, crearBarraProblemas, cssEsc } from './problemas.js'
import { problemasDe, listaCalculo, paraMotor, haiProblemas, xeracionDe, sincronizarVariables, temSolucion } from './problemas-modelo.js'

const ICONAS = {
  editar: '<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M4 16 14.5 5.5a1.4 1.4 0 0 1 2 2L6 18l-3 1z"/></svg>',
  lixo: '<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M4.5 5.5h11"/><path d="M8 5.5V4a1 1 0 0 1 1-1h2a1 1 0 0 1 1 1v1.5"/><path d="M5.5 5.5 6.2 16a1 1 0 0 0 1 .9h5.6a1 1 0 0 0 1-.9l.7-10.5"/></svg>',
  mais: '<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"><path d="M10 4.5v11M4.5 10h11"/></svg>',
  csv: '<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M10 3v9M6.5 8.5 10 12l3.5-3.5"/><path d="M4 14.5V16a1 1 0 0 0 1 1h10a1 1 0 0 0 1-1v-1.5"/></svg>',
}

// AAAA-MM-DD é o que entende <input type="date">; se o valor ven doutro sitio
// (un CSV dá DD/MM/AAAA) tradúcese para amosalo.
function dataParaInput(v) {
  const s = String(v ?? '').trim()
  const m = /^(\d{1,2})\/(\d{1,2})\/(\d{4})$/.exec(s)
  return m ? `${m[3]}-${m[2].padStart(2, '0')}-${m[1].padStart(2, '0')}` : s
}

// Opcións:
//   obterEsquema()    o esquema vixente (co seu `modo`: 'deseno' ou 'produccion')
//   aoDeseñar()       abre o deseñador (só se ofrece en modo deseño)
//   aoCambiarModo(m)  pide pasar a 'deseno' ou 'produccion' (quen chama confirma e garda)
//   aoBorrarDatosProba()  pide borrar os datos de proba da IA (o botón sae en produción se os hai)
//   avisar(m, tipo)   amosa unha mensaxe do asistente (usuario_avisar)
//   problemas         {api: {CalcularProblemas, ResolverProblema, InstallMaxima, IAAsistenteEstado}, idioma}
//                     para os campos «problemas» (cálculo con Maxima e resolución coa IA)
//   asistente         {api, idioma} do asistente de datos ficticios (ámbito
//                     «formulario_datos»); só se monta en modo deseño. En
//                     produción non hai asistente nin deseñador: a IA non pode
//                     ver datos reais.
export function createFormularioEditor(container, {
  t: tIn, onChange, obterEsquema, lembrarContexto, aoDeseñar, aoCambiarModo, aoBorrarDatosProba, avisar, asistente: asistenteOpc = null,
  problemas: problemasOpc = null,
} = {}) {
  const t = tIn || ((clave, fallback, vars) => {
    let val = fallback ?? clave
    if (vars) for (const k in vars) val = val.replaceAll(`{${k}}`, vars[k])
    return val
  })

  let plan = null
  let ctxAberto = null // null = decídese polo contido; despois, o que faga o usuario
  const mensaxeCsv = {} // id da táboa -> última mensaxe da importación
  container.classList.add('crono-editor', 'form-editor')

  const aplicar = (novo, motivo = 'texto') => {
    plan = novo
    if (onChange) onChange(plan, motivo)
    programarRecalculo()
  }
  const poñerValor = (id, v, motivo) => aplicar({ ...plan, valores: { ...plan.valores, [id]: v } }, motivo)
  const enProduccion = () => obterEsquema().modo === 'produccion'
  const poñerContexto = (k, v, motivo) => {
    aplicar({ ...plan, contexto: { ...plan.contexto, [k]: v } }, motivo)
    lembrarContexto?.(plan.contexto)
  }

  // ---------- Controles ----------

  // entrada crea o control dun valor simple (campo ou cela de táboa). `aoCambiar(valor, motivo)`.
  function entrada(c, valor, aoCambiar, { compacto = false, aria } = {}) {
    const comuns = { 'aria-label': aria || c.etiqueta, 'data-campo': c.id }
    switch (c.tipo) {
      case 'textolongo': {
        const a = el('textarea', { ...comuns, class: 'form-entrada form-entrada--longa', rows: '6', spellcheck: 'true', placeholder: t('formularios.markdown', 'Texto (admite Markdown: **negriña**, listas con «-»…)') })
        a.value = valor ?? ''
        a.addEventListener('input', () => aoCambiar(a.value, 'texto'))
        return a
      }
      case 'escolla': {
        const s = el('select', { ...comuns, class: 'form-entrada' })
        s.append(new Option('—', ''))
        const ops = [...(c.opcions || [])]
        if (valor && !ops.includes(valor)) ops.push(valor) // un valor importado non se perde
        for (const o of ops) s.append(new Option(o, o))
        s.value = valor ?? ''
        s.addEventListener('change', () => aoCambiar(s.value, 'estrutura'))
        return s
      }
      case 'verificacion': {
        const i = el('input', { ...comuns, type: 'checkbox', class: 'form-marca' })
        i.checked = valor === true || valor === 'true'
        i.addEventListener('change', () => aoCambiar(i.checked, 'estrutura'))
        return i
      }
      case 'data': {
        const i = el('input', { ...comuns, type: 'date', class: 'form-entrada' })
        i.value = dataParaInput(valor)
        i.addEventListener('input', () => aoCambiar(i.value, 'texto'))
        return i
      }
      case 'numero': {
        const i = el('input', { ...comuns, type: 'text', inputmode: 'decimal', class: 'form-entrada', autocomplete: 'off' })
        i.value = valor ?? ''
        i.addEventListener('input', () => aoCambiar(i.value, 'texto'))
        return i
      }
      default: {
        const i = el('input', { ...comuns, type: 'text', class: 'form-entrada' + (compacto ? ' form-entrada--compacta' : ''), spellcheck: 'true', autocomplete: 'off' })
        i.value = valor ?? ''
        i.addEventListener('input', () => aoCambiar(i.value, 'texto'))
        return i
      }
    }
  }

  // imaxe: escoller, ver e quitar unha imaxe (gárdase coma data URI).
  function controlImaxe(valor, aoCambiar, etiqueta) {
    const caixa = el('div', { class: 'form-imaxe' })
    const erro = el('p', { class: 'crono-erro', hidden: true })
    if (valor) {
      caixa.append(
        el('img', { class: 'form-imaxe__vista', src: valor, alt: etiqueta }),
        el('button', { type: 'button', class: 'crono-btn crono-btn--pequeno', onclick: () => aoCambiar('', 'estrutura') }, t('formularios.imaxe.quitar', 'Quitar')),
      )
    }
    const ficheiro = el('input', { type: 'file', accept: 'image/*', class: 'sr-only', tabindex: '-1' })
    ficheiro.addEventListener('change', async () => {
      const f = ficheiro.files?.[0]
      if (!f) return
      try {
        aoCambiar(await normalizarImaxe(f), 'estrutura')
      } catch (err) {
        erro.textContent = String(err?.message || err)
        erro.hidden = false
      }
    })
    caixa.append(
      el('button', { type: 'button', class: 'crono-btn crono-btn--pequeno', onclick: () => ficheiro.click() },
        valor ? t('formularios.imaxe.cambiar', 'Cambiar imaxe…') : t('formularios.imaxe.escoller', 'Escoller imaxe…')),
      ficheiro, erro,
    )
    return caixa
  }

  // taboa: filas repetibles das columnas do campo, con importación de CSV.
  function controlTaboa(c, filas, enfocarFila = -1) {
    const columnas = c.columnas || []
    const marco = el('div', { class: 'form-taboa' })
    const mudar = (novas, motivo = 'estrutura', enfocar = -1) => {
      poñerValor(c.id, novas, motivo)
      if (motivo === 'estrutura') render(c.id, enfocar)
    }
    const tabla = el('table', { class: 'form-taboa__tabla' })
    tabla.append(el('thead', {}, el('tr', {},
      ...columnas.map((col) => el('th', { text: col.etiqueta })),
      el('th', { class: 'form-taboa__acc' }))))
    const corpo = el('tbody')
    filas.forEach((fila, i) => {
      const tr = el('tr', { 'data-fila': String(i) })
      for (const col of columnas) {
        const td = el('td', { class: col.tipo === 'verificacion' ? 'form-taboa__marca' : null })
        td.append(entrada(col, fila[col.id], (v, motivo) => {
          const novas = filas.map((f, j) => (j === i ? { ...f, [col.id]: v } : f))
          filas = novas
          mudar(novas, motivo)
        }, { compacto: true, aria: `${col.etiqueta} (${i + 1})` }))
        tr.append(td)
      }
      tr.append(el('td', { class: 'form-taboa__acc' }, el('button', {
        type: 'button', class: 'form-icona-btn', title: t('formularios.taboa.quitarFila', 'Quitar a fila'),
        'aria-label': t('formularios.taboa.quitarFila', 'Quitar a fila'), html: ICONAS.lixo,
        onclick: () => mudar(filas.filter((_, j) => j !== i)),
      })))
      corpo.append(tr)
    })
    tabla.append(corpo)
    marco.append(el('div', { class: 'form-taboa__marco' }, tabla))
    if (!filas.length) marco.append(el('p', { class: 'form-taboa__baleira', text: t('formularios.taboa.baleira', 'Sen filas. Engade unha ou importa un CSV.') }))

    const ficheiro = el('input', { type: 'file', accept: '.csv,.tsv,.txt,text/csv', class: 'sr-only', tabindex: '-1' })
    ficheiro.addEventListener('change', async () => {
      const f = ficheiro.files?.[0]
      ficheiro.value = ''
      if (!f) return
      try {
        const csv = parseCSV(decodificarCSV(await f.arrayBuffer()))
        const r = taboaDeCSV(csv, columnas)
        if (!r.filas.length || r.casadas === 0) {
          mensaxeCsv[c.id] = { erro: true, texto: t('formularios.csv.nada', 'Non se atopou ningunha fila ou columna que encaixe co formulario.') }
        } else {
          mensaxeCsv[c.id] = { erro: false, texto: t('formularios.csv.ok', '{n} filas importadas ({c} de {m} columnas). {como}', {
            n: r.filas.length, c: r.casadas, m: columnas.length,
            como: r.cabeceira ? t('formularios.csv.porNome', 'Columnas casadas polo nome da cabeceira.') : t('formularios.csv.porOrde', 'Sen cabeceira: columnas casadas por orde.'),
          }) }
          mudar([...filas, ...r.filas], 'estrutura')
          return
        }
      } catch (err) {
        mensaxeCsv[c.id] = { erro: true, texto: t('formularios.csv.erro', 'Non se puido ler o CSV: {e}', { e: String(err?.message || err) }) }
      }
      render(c.id)
    })
    const msg = mensaxeCsv[c.id]
    marco.append(el('div', { class: 'form-taboa__pe' },
      el('button', {
        type: 'button', class: 'crono-btn crono-btn--pequeno', html: ICONAS.mais + `<span>${t('formularios.taboa.engadir', 'Engadir fila')}</span>`,
        onclick: () => mudar([...filas, {}], 'estrutura', filas.length),
      }),
      el('button', {
        type: 'button', class: 'crono-btn crono-btn--pequeno', html: ICONAS.csv + `<span>${t('formularios.taboa.csv', 'Importar CSV…')}</span>`,
        title: t('formularios.taboa.csvAxuda', 'Cada fila do CSV pasa a ser unha fila da táboa. Se a primeira fila ten os nomes das columnas, casan polo nome; se non, por orde.'),
        onclick: () => ficheiro.click(),
      }),
      ficheiro,
      msg ? el('span', { class: 'form-taboa__msg' + (msg.erro ? ' form-taboa__msg--erro' : ''), role: 'status', text: msg.texto }) : null,
    ))
    if (enfocarFila >= 0) queueMicrotask(() => marco.querySelector(`tr[data-fila="${enfocarFila}"] input, tr[data-fila="${enfocarFila}"] select`)?.focus())
    return marco
  }

  // campo devolve a tarxeta dun campo do esquema.
  function campo(c) {
    if (c.tipo === 'seccion') return el('h3', { class: 'form-seccion', text: c.etiqueta })
    const valor = plan.valores[c.id]
    const etq = el('span', { class: 'form-camp__etq' }, c.etiqueta, c.obrigatorio ? el('span', { class: 'form-camp__obr', title: t('formularios.obrigatorio', 'Obrigatorio'), text: ' *' }) : null)
    const axuda = c.axuda ? el('small', { class: 'form-camp__axuda', text: c.axuda }) : null
    const marcaAgochado = c.agochado ? el('small', { class: 'form-camp__axuda', text: t('formularios.agochado', 'Non sae no documento (só se usa na plantilla).') }) : null
    if (c.tipo === 'taboa') {
      return el('div', { class: 'form-camp form-camp--taboa' }, el('div', { class: 'form-camp__cab' }, etq), axuda, controlTaboa(c, Array.isArray(valor) ? valor : []), marcaAgochado)
    }
    if (c.tipo === 'problemas') return campoProblemas(c, etq, axuda)
    if (c.tipo === 'imaxe') {
      return el('div', { class: 'form-camp' }, el('div', { class: 'form-camp__cab' }, etq), axuda,
        controlImaxe(typeof valor === 'string' ? valor : '', (v, m) => { poñerValor(c.id, v, m); render(c.id) }, c.etiqueta), marcaAgochado)
    }
    const control = entrada(c, valor, (v, m) => poñerValor(c.id, v, m))
    if (c.tipo === 'verificacion') {
      return el('label', { class: 'form-camp form-camp--marca' }, control, etq, axuda, marcaAgochado)
    }
    return el('label', { class: 'form-camp' }, el('span', { class: 'form-camp__cab' }, etq), axuda, control, marcaAgochado)
  }

  // ---------- Problemas (campos «problemas») ----------
  //
  // O calculado (enunciados e solucións con valores, na semente do documento)
  // vive aquí, fóra do plan: vén de Go (CalcularProblemas) e recalcúlase cando
  // cambia unha pregunta, unha variable ou a semente.
  const CLAVE_MOTOR = 'lua.problemas.motor'
  const estProb = {
    calc: new Map(), debuxos: {}, cola: new Set(), maxima: false, avisoMaxima: '', resolvendo: new Set(), informes: new Map(), editando: new Set(),
    iaDispoñible: true, enfocar: null, solta: null, asistente: new Map(),
    motor: (() => { try { return localStorage.getItem(CLAVE_MOTOR) === 'autonomo' ? 'autonomo' : 'maxima' } catch { return 'maxima' } })(),
  }
  let firmaCalc = ''
  let seqCalc = 0
  let temporizadorCalc = null
  let maximaComprobado = false

  function programarRecalculo(inmediato = false) {
    if (!problemasOpc?.api?.CalcularProblemas || !plan || !haiProblemas(obterEsquema())) return
    clearTimeout(temporizadorCalc)
    temporizadorCalc = setTimeout(recalcular, inmediato ? 0 : 350)
  }

  // sincronizarTodas declara as variables novas que se escribiron nos
  // enunciados ({x} sen declarar). Corre co recálculo (a pausa ao teclear).
  function sincronizarTodas() {
    let novos = null
    for (const c of obterEsquema().campos) {
      if (c.tipo !== 'problemas') continue
      const lista = problemasDe(plan.valores?.[c.id])
      const sinc = lista.map(sincronizarVariables)
      if (sinc.some((p, i) => p !== lista[i])) (novos ??= { ...plan.valores })[c.id] = sinc
    }
    return novos ? { ...plan, valores: novos } : null
  }

  async function recalcular() {
    const sinc = sincronizarTodas()
    if (sinc) {
      plan = sinc
      if (onChange) onChange(plan, 'estrutura')
      render() // volve chamar a programarRecalculo
      return
    }
    const elems = listaCalculo(obterEsquema(), plan.valores)
    const pedido = { semente: xeracionDe(plan).semente, problemas: elems.map((e) => ({ indice: e.indice, problema: paraMotor(e.problema) })) }
    const firma = JSON.stringify(pedido)
    if (firma === firmaCalc) return
    firmaCalc = firma
    const n = ++seqCalc
    try {
      const r = await problemasOpc.api.CalcularProblemas(pedido)
      if (n !== seqCalc) return
      const mudouMaxima = !maximaComprobado || estProb.maxima !== !!r.maxima || estProb.avisoMaxima !== (r.aviso || '')
      maximaComprobado = true
      estProb.maximaComprobado = true
      estProb.maxima = !!r.maxima
      estProb.avisoMaxima = r.aviso || ''
      elems.forEach((e, i) => { if (r.calculados?.[i]) estProb.calc.set(e.problema.id, r.calculados[i]) })
      Object.assign(estProb.debuxos, r.debuxos || {})
      if (mudouMaxima) render()
      else actualizarVistas()
    } catch {
      firmaCalc = ''
    }
  }

  function actualizarVistas() {
    for (const c of obterEsquema().campos) {
      if (c.tipo !== 'problemas') continue
      const marco = container.querySelector(`.prob[data-campo="${cssEsc(c.id)}"]`)
      if (marco) actualizarVistasProblemas(marco, problemasDe(plan.valores[c.id]), estProb, t)
    }
  }

  // As preguntas levan un id (só para a interface: casar cada pregunta coa
  // súa solución e co seu calculado). Se falta (datos da IA, ficheiros
  // vellos), pónse sen marcar o documento coma cambiado.
  function asegurarIds() {
    let novos = null
    for (const c of obterEsquema().campos) {
      if (c.tipo !== 'problemas') continue
      const v = plan.valores?.[c.id]
      if (Array.isArray(v) && v.some((p) => p && !p.id)) (novos ??= { ...plan.valores })[c.id] = problemasDe(v)
    }
    if (novos) plan = { ...plan, valores: novos }
  }

  async function comprobarIA() {
    try {
      const e = await problemasOpc?.api?.IAAsistenteEstado?.()
      const ok = e ? !!e.configurada : true
      if (ok !== estProb.iaDispoñible) { estProb.iaDispoñible = ok; if (plan) render() }
    } catch { /* sen info: déixase activo e o erro sae ao resolver */ }
  }

  const accionsProblemas = {
    cambiarMotor(m) {
      estProb.motor = m === 'autonomo' ? 'autonomo' : 'maxima'
      try { localStorage.setItem(CLAVE_MOTOR, estProb.motor) } catch { /* sen persistencia */ }
      render()
    },
    async instalarMaxima() {
      if (!problemasOpc?.api?.InstallMaxima) return
      avisar?.(t('problemas.maxima.instalando', 'Instalando Maxima… (pode pedir o contrasinal e tardar uns minutos)'), 'info')
      try {
        const r = await problemasOpc.api.InstallMaxima()
        avisar?.(r?.success ? t('problemas.maxima.instalado', 'Maxima instalado.') : t('problemas.maxima.fallou', 'Non se puido instalar Maxima: {e}', { e: (r?.log || '').slice(-400) }), r?.success ? 'info' : 'aviso')
      } catch (err) {
        avisar?.(String(err?.message || err), 'aviso')
      }
      firmaCalc = ''
      maximaComprobado = false
      estProb.maximaComprobado = false
      programarRecalculo(true)
    },
    // asistentePregunta: a proposta da IA para unha pregunta (problemas_asistente.go).
    // Non cambia nada: o control amósaa e o docente acéptaa ou descártaa.
    async asistentePregunta(id, { accion, instrucions = '' }) {
      if (!problemasOpc?.api?.AsistentePregunta) throw new Error(t('problemas.asis.semApi', 'O asistente da pregunta non está dispoñible nesta xanela.'))
      const esq = obterEsquema()
      const e = listaCalculo(esq, plan.valores).find((x) => x.problema.id === id)
      if (!e) throw new Error(t('problemas.asis.senPregunta', 'Esta pregunta xa non existe.'))
      const campo = esq.campos.find((c) => c.id === e.campo)
      return problemasOpc.api.AsistentePregunta({
        problema: paraMotor(e.problema), accion, instrucions,
        contexto: [esq.nome, esq.descricion, campo?.etiqueta].filter(Boolean).join(' — '), idioma: problemasOpc.idioma?.() || '',
      })
    },
    // repintar: cambios só da vista (panel aberto, proposta…), sen tocar o plan.
    repintar: () => render(),
    async resolver(ids, { instrucions = '' } = {}) {
      if (!problemasOpc?.api?.ResolverProblema) return
      // Todas quedan «en cola» e resólvense unha tras outra.
      ids = ids.filter((id) => !estProb.cola.has(id) && !estProb.resolvendo.has(id))
      for (const id of ids) estProb.cola.add(id)
      for (const id of ids) {
        estProb.cola.delete(id)
        const esq = obterEsquema()
        const e = listaCalculo(esq, plan.valores).find((x) => x.problema.id === id)
        if (!e) continue
        const campo = esq.campos.find((c) => c.id === e.campo)
        estProb.resolvendo.add(id)
        estProb.informes.delete(id)
        render()
        try {
          const r = await problemasOpc.api.ResolverProblema({
            problema: paraMotor(e.problema), indice: e.indice, semente: xeracionDe(plan).semente, modo: estProb.motor,
            contexto: [esq.nome, esq.descricion, campo?.etiqueta].filter(Boolean).join(' — '), instrucions, idioma: problemasOpc.idioma?.() || '',
          })
          const lista = problemasDe(plan.valores[e.campo]).map((p) => (p.id === id ? { ...p, solucion: r.solucion } : p))
          if (r.calculado) estProb.calc.set(id, r.calculado)
          Object.assign(estProb.debuxos, r.debuxos || {})
          estProb.informes.set(id, { rexistro: r.rexistro || [], fallos: r.fallos || [] })
          estProb.editando.delete(id)
          estProb.resolvendo.delete(id)
          aplicar({ ...plan, valores: { ...plan.valores, [e.campo]: lista } }, 'estrutura')
        } catch (err) {
          estProb.informes.set(id, { erro: String(err?.message || err) })
        } finally {
          estProb.resolvendo.delete(id)
          render()
        }
      }
    },
  }

  function campoProblemas(c, etq, axuda) {
    return el('div', { class: 'form-camp form-camp--problemas' },
      el('div', { class: 'form-camp__cab' }, etq), axuda,
      crearControlProblemas({
        t, campo: c, valor: plan.valores[c.id], estado: estProb, accions: accionsProblemas,
        aoCambiar: (lista, motivo) => {
          poñerValor(c.id, lista, motivo)
          if (motivo === 'estrutura') render()
        },
      }))
  }

  // ---------- Datos do documento ----------

  function bloqueContexto() {
    const cx = plan.contexto
    const det = el('details', { class: 'form-card form-ctx' })
    // Pechado por defecto: ábreo quen o precise (enche variables da plantilla).
    det.open = ctxAberto ?? false
    det.addEventListener('toggle', () => { ctxAberto = det.open })
    det.append(el('summary', {}, el('strong', { text: t('formularios.ctx.titulo', 'Datos do documento') }),
      el('span', { class: 'form-ctx__resumo', text: t('formularios.ctx.resumo', 'centro, curso, quen asina, logotipo…') })))
    const reixa = el('div', { class: 'form-ctx__reixa' })
    const simples = [
      ['centro', t('formularios.ctx.centro', 'Centro')], ['localidade', t('formularios.ctx.localidade', 'Localidade')],
      ['curso', t('formularios.ctx.curso', 'Curso académico')], ['departamento', t('formularios.ctx.departamento', 'Departamento')],
      ['responsable', t('formularios.ctx.responsable', 'Quen asina')], ['cargo', t('formularios.ctx.cargo', 'Cargo')],
    ]
    for (const [k, etiqueta] of simples) {
      reixa.append(el('label', { class: 'form-camp' }, el('span', { class: 'form-camp__cab' }, el('span', { class: 'form-camp__etq', text: etiqueta })),
        entrada({ id: 'ctx_' + k, tipo: 'texto', etiqueta }, cx[k], (v, m) => poñerContexto(k, v, m))))
    }
    det.append(reixa, el('div', { class: 'form-camp' }, el('div', { class: 'form-camp__cab' }, el('span', { class: 'form-camp__etq', text: t('formularios.ctx.logo', 'Logotipo do centro') })),
      controlImaxe(cx.logo || '', (v, m) => { poñerContexto('logo', v, m); render() }, t('formularios.ctx.logo', 'Logotipo do centro'))))
    return det
  }

  // ---------- Modo (deseño / produción) e asistente ----------

  // O interruptor de modo: sempre visible (é o único que se pode tocar en
  // produción), coloreado coma a icona da columna esquerda: vermello = deseño,
  // verde = produción. A cor non é o único indicio: leva texto.
  function botonModo(produccion) {
    return el('button', {
      type: 'button', class: `form-modo form-modo--${produccion ? 'produccion' : 'deseno'}`,
      'aria-label': produccion ? t('formularios.modo.aria.produccion', 'Modo produción. Preme para pasar a deseño') : t('formularios.modo.aria.deseno', 'Modo deseño. Preme para pasar a produción'),
      title: produccion
        ? t('formularios.modo.produccion.title', 'Produción: datos reais, sen asistente de IA nin deseñador, para que a IA non poida ver datos privados. Preme para volver a deseño.')
        : t('formularios.modo.deseno.title', 'Deseño: podes cambiar o formulario e usar o asistente de IA con datos ficticios. Preme para pasar a produción.'),
      onclick: () => aoCambiarModo?.(produccion ? 'deseno' : 'produccion'),
    }, el('span', { class: 'form-modo__punto', 'aria-hidden': 'true' }),
    el('span', { text: produccion ? t('formularios.modo.produccion', 'Produción') : t('formularios.modo.deseno', 'Deseño') }))
  }

  function botonDeseñar() {
    return el('button', {
      type: 'button', class: 'crono-btn', html: ICONAS.editar + `<span>${t('formularios.deseñar', 'Deseñar formulario')}</span>`,
      title: t('formularios.deseñar.title', 'Cambiar os campos deste formulario: engadir, mover, quitar ou cambiar o seu tipo'),
      onclick: () => aoDeseñar?.(),
    })
  }

  // O «Asistente datos»: un panel lateral que vive fóra do que pinta render()
  // (zonaAsist), así non se perde ao repintar. Só existe en modo deseño: en
  // produción o botón non se pinta e o panel destrúese. Pódeselle xuntar
  // documentos (adxuntos) para que os teña en conta ao encher datos ficticios.
  let asistente = null
  let logoAsist = ''
  let asistOcupado = false
  const zonaAsist = el('div', { class: 'form-asist' })

  function botonAsistente() {
    if (!asistente) return null
    const logo = el('span', { class: 'tarefa-asistente__logo', html: logoAsist })
    return el('button', {
      type: 'button', class: 'tarefa-asistente' + (asistOcupado ? ' tarefa-asistente--ocupado' : ''),
      title: t('formularios.asistente.title', 'Asistente de IA deste formulario: enche datos ficticios ou cambia os que hai escritos, tendo en conta os documentos que lle xuntes (só en modo deseño)'),
      onclick: () => asistente?.alternar(),
    }, logo, el('span', { text: t('formularios.asistente.label', 'Asistente datos') }))
  }

  function resumoParaIA() {
    const esquema = obterEsquema()
    const cubertos = esquema.campos.filter((c) => c.tipo !== 'seccion' && plan?.valores?.[c.id] !== undefined && plan.valores[c.id] !== '' && !(Array.isArray(plan.valores[c.id]) && !plan.valores[c.id].length)).length
    return [
      `- Formulario «${esquema.nome}»: ${esquema.campos.length} campo(s), ${cubertos} con datos. MODO DESEÑO: os datos son de proba (ficticios).`,
    ].join('\n')
  }

  function montarAsistente() {
    const opc = asistenteOpc
    const ferr = crearFerramentasFormularioDatos({
      esquema: obterEsquema,
      plan: () => plan,
      aplicar: (novo) => { aplicar(novo, 'estrutura'); render() },
      api: { IAChamarFerramenta: opc.api.IAChamarFerramenta },
      xerar: opc.xerar || null,
      ui: { avisar: (m, tipo) => avisar?.(m, tipo), preguntar: (p, o) => asistente.preguntar(p, o) },
    })
    asistente = crearPanelAsistente({
      t,
      api: {
        ...opc.api, // inclúe os adxuntos (clip, arrastrar, pegar): documentos de referencia para encher datos
        IAFerramentas: () => opc.api.IAFerramentasDoAmbito('formulario_datos'),
        IATurnoAsistente: (req) => opc.api.IATurnoAsistente({ ...req, ambito: 'formulario_datos', idioma: opc.idioma?.() || '' }),
      },
      documentoBoveda: 'formulario',
      executar: ferr.executar,
      estadoPrograma: resumoParaIA,
      tarefas: TAREFAS_FORMULARIO_DATOS,
      claveTarefas: 'lua.asistente.tarefas.formulario_datos',
      montarEn: zonaAsist,
      textos: {
        titulo: t('asistente.tfd.titulo', 'Asistente datos'),
        benvida: t('asistente.tfd.benvida', 'Enchote o formulario con datos ficticios para probalo, ou cambio os que hai escritos. Podes xuntarme cun clip documentos (un temario, un exame xa feito…) para que os teña en conta, útil para deseñar exames e prácticas. Só vexo este formulario, os seus datos e os documentos que me xuntes, e só mentres estea en modo deseño.'),
        benvidaNota: t('asistente.tfd.nota', 'Podes desfacer o que faga co historial do programa (Ctrl+Z). Cando o formulario pase a produción, este asistente desaparece.'),
        caixaPh: t('asistente.tfd.ph', 'Pídeme datos de proba: «enche o formulario con 12 alumnos», «pon as notas máis baixas»… Se premes unha ficha, isto engádese como instrución adicional.'),
      },
      aoCambiar: ({ ocupado, preset }) => {
        logoAsist = logoProvedor(preset)
        asistOcupado = ocupado
        const b = container.querySelector('.form-editor__cab .tarefa-asistente')
        if (b) {
          b.querySelector('.tarefa-asistente__logo').innerHTML = logoAsist
          b.classList.toggle('tarefa-asistente--ocupado', ocupado)
        }
      },
    })
    asistente.pechar()
  }

  function sincronizarAsistente() {
    const quere = !!asistenteOpc && !enProduccion()
    if (quere && !asistente) montarAsistente()
    else if (!quere && asistente) { asistente.destruir(); asistente = null; zonaAsist.replaceChildren() }
  }

  // Datos de proba: se a IA encheu datos en deseño e o formulario pasou a produción,
  // ofrécese un botón para borralos (senón quedarían mesturados cos reais).
  function avisoDatosProba(produccion) {
    if (!produccion || !plan?.datosProba) return null
    return el('div', { class: 'form-proba', role: 'status' },
      el('span', { class: 'form-proba__txt', text: t('formularios.proba.aviso', 'Este formulario ten datos de proba que puxo a IA en modo deseño.') }),
      el('button', {
        type: 'button', class: 'crono-btn', 'data-accion': 'borrar-proba', text: t('formularios.proba.borrar', 'Borrar datos de proba'),
        title: t('formularios.proba.borrar.title', 'Baleira os datos do formulario para empezar cos reais'),
        onclick: () => aoBorrarDatosProba?.(),
      }))
  }

  // ---------- Pintado ----------

  function render(enfocarId = null, enfocarFila = -1) {
    const esquema = obterEsquema()
    const produccion = esquema.modo === 'produccion'
    sincronizarAsistente()
    asegurarIds()
    // O foco dentro dos problemas (o repintado substitúe os nodos).
    const activo = document.activeElement
    const foco = activo && container.contains(activo) && activo.dataset?.foco
      ? { clave: activo.dataset.foco, ini: activo.selectionStart, fin: activo.selectionEnd } : null
    const y = container.scrollTop
    const cab = el('header', { class: 'form-editor__cab' },
      el('div', { class: 'form-editor__titulo' },
        el('h2', { text: esquema.nome }),
        esquema.descricion ? el('p', { text: esquema.descricion }) : null),
      el('div', { class: 'form-editor__acc' }, botonModo(produccion), botonAsistente(), produccion ? null : botonDeseñar()))
    const corpo = []
    if (!esquema.campos.length) {
      corpo.push(el('div', { class: 'form-baleiro' },
        el('p', { text: t('formularios.baleiro', 'Este formulario aínda non ten campos.') }),
        produccion ? null : el('button', { type: 'button', class: 'crono-btn', onclick: () => aoDeseñar?.() }, t('formularios.baleiro.boton', 'Engadir campos'))))
    } else {
      for (const c of esquema.campos) corpo.push(campo(c))
    }
    // (filter: replaceChildren(null) escribiría a palabra «null»)
    // (O antigo check das referencias [PAU 2021 · ordinaria] quitouse: os textos saen
    // tal e como están escritos. Un documento que as tiña ocultas recupéraas.)
    if (plan?.senReferencias) queueMicrotask(() => { if (plan?.senReferencias) { aplicar(amosarReferencias(plan), 'referencias'); render() } })
    // Cos campos de problemas: a barra común (Maxima, motor e «Resolver o exame»).
    const barraProb = problemasOpc && haiProblemas(esquema)
      ? crearBarraProblemas({
        t, estado: estProb, accions: accionsProblemas,
        pendentes: listaCalculo(esquema, plan.valores).map((e) => e.problema).filter((p) => (p.enunciado || '').trim() && !temSolucion(p)).map((p) => p.id),
      })
      : null
    container.replaceChildren(...[cab, barraProb, avisoDatosProba(produccion), bloqueContexto(), el('div', { class: 'form-campos' }, corpo), zonaAsist].filter(Boolean))
    container.scrollTop = y
    if (foco) {
      const n = container.querySelector(`[data-foco="${cssEsc(foco.clave)}"]`)
      if (n) { n.focus(); try { n.setSelectionRange(foco.ini, foco.fin) } catch { /* non é un texto */ } }
    }
    if (estProb.enfocar) {
      container.querySelector(`[data-foco="${cssEsc('enunciado:' + estProb.enfocar)}"]`)?.focus()
      estProb.enfocar = null
    }
    programarRecalculo()
    if (enfocarId && enfocarFila < 0) container.querySelector(`[data-campo="${enfocarId}"]`)?.focus()
    if (enfocarId && enfocarFila >= 0) {
      const marco = [...container.querySelectorAll('.form-taboa')].find((m) => m.querySelector(`tr[data-fila="${enfocarFila}"]`))
      marco?.querySelector(`tr[data-fila="${enfocarFila}"] input, tr[data-fila="${enfocarFila}"] select`)?.focus()
    }
  }

  if (problemasOpc) comprobarIA()

  return {
    getValue: () => plan,
    // setValue substitúe o documento enteiro (abrir, novo, desfacer...) e NON
    // avisa a onChange: quen o chama sabe que cambiou.
    setValue(novo) { plan = novo; render() },
    // Para a barra de xeración (semente, copias, solucións): o editor recalcula.
    recalcularProblemas: () => programarRecalculo(true),
    focus() { container.querySelector('input, textarea, select, button')?.focus() },
    refrescar: () => { firmaCalc = ''; maximaComprobado = false; if (problemasOpc) comprobarIA(); if (plan) render() },
    destroy() { clearTimeout(temporizadorCalc); asistente?.destruir(); asistente = null; container.replaceChildren(); container.classList.remove('crono-editor', 'form-editor') },
  }
}
