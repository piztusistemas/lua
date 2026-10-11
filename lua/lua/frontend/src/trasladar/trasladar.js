// A xanela TRASLADAR (spec trasladar): o único sitio para sacar cousas de Lúa
// e metelas. Dúas lapelas:
//
//   Sacar  — vén marcado o que está aberto (o proxecto, ou o documento coma
//            .json editable); tamén «Todo Lúa» (copia de seguridade) e
//            «Escoller…» (árbore de proxectos, formularios e plantillas).
//   Meter  — soltar ou escoller un ficheiro ou cartafol; Lúa di que leva e
//            onde vai antes de facer nada, e despois permite desfacer.
//
//   abrirTrasladar({ t, api, contexto, lapela, opcion, ruta, marcados, confirmar,
//                    describirCurso, antesDeMeter, despoisDeMeter }) -> Promise (ao pechar)
//   api: ArboreTraslado, SacarTenDatosProducion, SacarTraslado, SacarDocumentoJSON,
//        contidoDocumento(), EscollerFicheiroMeter, EscollerCartafolMeter,
//        AnalizarMeter, AplicarMeter, DesfacerTraslado
//   contexto: { proxecto: {id, nome}|null, documento: {ferramenta, nome}|null }
//
// Se xa está aberta, abrirTrasladar só leva a ruta (un ficheiro soltado) á
// lapela Meter.
import { el } from '../informes/dom.js'
import {
  opcionsSacar, opcionInicial, ofreceDatos, seleccionSacar, seleccionBaleira, textoElemento,
  opcionsDestino, textoDestino, confirmacionPrevia, opcionsMeter, podeMeter,
} from './modelo.js'

const ICONA_X = '<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M5 5l10 10M15 5L5 15"/></svg>'

let aberta = null // { meter(ruta), lapela(id) } da xanela aberta

export function trasladarAberta() {
  return Boolean(aberta)
}

export function abrirTrasladar(o) {
  if (aberta) {
    if (o.ruta) aberta.meter(o.ruta)
    else aberta.lapela(o.lapela || 'sacar')
    return Promise.resolve()
  }
  const { t, api, contexto, confirmar } = o
  return new Promise((resolve) => {
    const previo = document.activeElement
    let ocupado = false
    const erro = el('p', { class: 'crono-erro bib-erro', role: 'alert', hidden: true })
    const mostrarErro = (err) => {
      erro.textContent = err ? String(err?.message || err) : ''
      erro.hidden = !err
    }
    async function executar(fn) {
      if (ocupado) return
      ocupado = true
      caixa.setAttribute('aria-busy', 'true')
      mostrarErro(null)
      try { await fn() } catch (err) { mostrarErro(err) } finally {
        ocupado = false
        caixa.removeAttribute('aria-busy')
      }
    }

    // ---------- Lapelas ----------
    const btnSacar = el('button', { type: 'button', role: 'tab', class: 'tras-lapela', id: 'trasLapelaSacar', text: t('trasladar.sacar', 'Sacar de Lúa'), onclick: () => irA('sacar') })
    const btnMeter = el('button', { type: 'button', role: 'tab', class: 'tras-lapela', id: 'trasLapelaMeter', text: t('trasladar.meter', 'Meter en Lúa'), onclick: () => irA('meter') })
    const corpoSacar = el('section', { class: 'tras-corpo', role: 'tabpanel', 'aria-labelledby': 'trasLapelaSacar' })
    const corpoMeter = el('section', { class: 'tras-corpo', role: 'tabpanel', 'aria-labelledby': 'trasLapelaMeter' })
    function irA(id) {
      const sacar = id === 'sacar'
      btnSacar.setAttribute('aria-selected', String(sacar))
      btnMeter.setAttribute('aria-selected', String(!sacar))
      btnSacar.tabIndex = sacar ? 0 : -1
      btnMeter.tabIndex = sacar ? -1 : 0
      corpoSacar.hidden = !sacar
      corpoMeter.hidden = sacar
      mostrarErro(null)
      const foco = (sacar ? corpoSacar : corpoMeter).querySelector('input:checked, button.primary, button')
      foco?.focus()
    }
    const lapelas = el('div', { class: 'tras-lapelas', role: 'tablist', onkeydown: (e) => {
      if (e.key === 'ArrowRight' || e.key === 'ArrowLeft') {
        e.preventDefault()
        irA(btnSacar.getAttribute('aria-selected') === 'true' ? 'meter' : 'sacar')
        ;(btnSacar.getAttribute('aria-selected') === 'true' ? btnSacar : btnMeter).focus()
      }
    } }, btnSacar, btnMeter)

    // ---------- Sacar ----------
    let opcion = opcionInicial(contexto, o.opcion)
    const marcados = { proxectos: new Set(o.marcados?.proxectos || []), formularios: new Set(o.marcados?.formularios || []), plantillas: new Set(o.marcados?.plantillas || []) }
    if (o.marcados) opcion = 'escoller'
    let arbore = null
    const chkDatos = el('input', { type: 'checkbox', id: 'trasDatos', checked: true })
    const filaDatos = el('label', { class: 'tras-marca', for: 'trasDatos' }, chkDatos, ' ', t('trasladar.datos', 'Incluír os datos escritos'))
    const zonaArbore = el('div', { class: 'tras-arbore' })
    const resultadoSacar = el('p', { class: 'tras-resultado', role: 'status' })
    const btnFacerSacar = el('button', { type: 'button', class: 'crono-btn primary', text: t('trasladar.sacar.boton', 'Sacar…'), onclick: () => executar(sacar) })

    function etiquetaOpcion(op) {
      switch (op.id) {
        case 'proxecto': return t('trasladar.op.proxecto', 'Este proxecto, «{nome}» (formularios, datos e bóveda) · ficheiro .lua', { nome: op.nome })
        case 'documento': return t('trasladar.op.documento', 'Este documento, «{nome}», para editalo fóra · ficheiro .json', { nome: op.nome })
        case 'todo': return t('trasladar.op.todo', 'Todo Lúa: copia de seguridade (proxectos, biblioteca, plantillas e opcións, sen claves) · ficheiro .lua')
        case 'escoller': return t('trasladar.op.escoller', 'Escoller…')
      }
      return op.id
    }
    const radios = el('div', { class: 'tras-opcions', role: 'radiogroup', 'aria-label': t('trasladar.que', 'Que queres levar?') },
      opcionsSacar(contexto).map((op) => el('label', { class: 'tras-opcion' },
        el('input', { type: 'radio', name: 'trasOpcion', value: op.id, checked: op.id === opcion, onchange: () => { opcion = op.id; pintarSacar() } }),
        ' ', etiquetaOpcion(op))))

    async function pintarArbore() {
      if (!arbore) {
        zonaArbore.replaceChildren(el('p', { class: 'prx-axuda', text: t('trasladar.cargando', 'Cargando…') }))
        arbore = await api.ArboreTraslado()
      }
      const grupo = (titulo, lista, conxunto) => el('fieldset', { class: 'tras-grupo' },
        el('legend', { text: titulo }),
        lista.length ? lista.map((it) => el('label', { class: 'tras-opcion' },
          el('input', { type: 'checkbox', checked: conxunto.has(it.id), onchange: (e) => { if (e.target.checked) conxunto.add(it.id); else conxunto.delete(it.id) } }),
          ' ', it.nome)) : el('p', { class: 'prx-axuda', text: t('trasladar.ningun', 'Ningún') }))
      zonaArbore.replaceChildren(
        grupo(t('trasladar.g.proxectos', 'Proxectos'), arbore.proxectos || [], marcados.proxectos),
        grupo(t('trasladar.g.formularios', 'Formularios da biblioteca'), arbore.formularios || [], marcados.formularios),
        grupo(t('trasladar.g.plantillas', 'Plantillas'), arbore.plantillas || [], marcados.plantillas))
    }

    function pintarSacar() {
      filaDatos.hidden = !ofreceDatos(opcion)
      zonaArbore.hidden = opcion !== 'escoller'
      if (opcion === 'escoller') executar(pintarArbore)
    }

    async function sacar() {
      resultadoSacar.textContent = ''
      let r
      if (opcion === 'documento') {
        r = await api.SacarDocumentoJSON(contexto.documento.ferramenta, api.contidoDocumento(), contexto.documento.nome)
      } else {
        const sel = seleccionSacar(opcion, contexto, marcados, chkDatos.checked)
        if (seleccionBaleira(sel)) throw new Error(t('trasladar.nada', 'Non escolliches nada que sacar.'))
        if (sel.datos && await api.SacarTenDatosProducion(sel)) {
          const si = await confirmar(t('trasladar.produccion.sacar', 'Hai formularios en PRODUCIÓN con datos que poden ser reais. Se van no ficheiro, saen do teu equipo. Incluílos?'), {
            aceptar: t('trasladar.produccion.si', 'Incluír os datos'), cancelar: t('trasladar.produccion.non', 'Sen datos'),
          })
          if (!si) sel.datos = false
        }
        r = await api.SacarTraslado(sel)
      }
      if (r && r.ruta) resultadoSacar.textContent = t('trasladar.sacado', 'Gardado en {ruta}', { ruta: r.ruta })
    }

    corpoSacar.append(
      el('h3', { text: t('trasladar.que', 'Que queres levar?') }),
      radios, filaDatos, zonaArbore,
      el('div', { class: 'modal-actions' }, btnFacerSacar),
      resultadoSacar)

    // ---------- Meter ----------
    let ruta = ''
    let resumo = null
    const escolla = { destino: '', cursos: new Set(), accion: 'actualizar' }
    const zonaResumo = el('div', { class: 'tras-resumo', 'aria-live': 'polite' })
    const resultadoMeter = el('div', { class: 'tras-resultado', role: 'status' })
    const soltar = el('div', { class: 'tras-soltar', 'data-file-drop-target': '' },
      el('p', { text: t('trasladar.soltar', 'Solta aquí un ficheiro ou un cartafol (.lua, .json, PDF, Markdown ou unha bóveda), ou escólleo:') }),
      el('div', { class: 'tras-botons' },
        el('button', { type: 'button', class: 'crono-btn', text: t('trasladar.escollerFicheiro', 'Escoller ficheiro…'), onclick: () => executar(async () => { const r = await api.EscollerFicheiroMeter(); if (r) await analizar(r) }) }),
        el('button', { type: 'button', class: 'crono-btn', text: t('trasladar.escollerCartafol', 'Escoller cartafol…'), onclick: () => executar(async () => { const r = await api.EscollerCartafolMeter(); if (r) await analizar(r) }) })))

    async function analizar(r) {
      ruta = r
      resumo = null
      resultadoMeter.replaceChildren()
      zonaResumo.replaceChildren(el('p', { class: 'prx-axuda', text: t('trasladar.lendo', 'Lendo…') }))
      try {
        resumo = await api.AnalizarMeter(ruta)
      } catch (err) {
        zonaResumo.replaceChildren()
        throw err
      }
      escolla.destino = opcionsDestino(resumo).length ? resumo.destino : ''
      escolla.accion = 'actualizar'
      escolla.cursos = new Set((resumo.cursos || []).map((_, i) => i))
      pintarResumo()
    }

    function pintarResumo() {
      if (!resumo) { zonaResumo.replaceChildren(); return }
      const partes = [
        el('h3', { text: t('trasladar.resumo', 'Que leva «{nome}»', { nome: resumo.nome }) }),
        el('ul', { class: 'tras-lista' }, (resumo.elementos || []).filter((e) => resumo.tipo !== 'curso' || e.tipo !== 'documento').map((e) => el('li', { text: textoElemento(e, t) }))),
      ]
      const cursos = resumo.cursos || []
      if (cursos.length > 1) {
        partes.push(el('fieldset', { class: 'tras-grupo' },
          el('legend', { text: t('trasladar.cursos', 'Este ficheiro trae {n} cursos. Cada un escollido entra no seu proxecto:', { n: cursos.length }) }),
          cursos.map((c, i) => el('label', { class: 'tras-opcion' },
            el('input', { type: 'checkbox', checked: escolla.cursos.has(i), onchange: (e) => { if (e.target.checked) escolla.cursos.add(i); else escolla.cursos.delete(i); actualizarBoton() } }),
            ' ', o.describirCurso ? o.describirCurso(resumo.ferramenta, c, i) : String(i + 1)))))
      } else if (cursos.length === 1) {
        partes.push(el('p', { text: o.describirCurso ? o.describirCurso(resumo.ferramenta, cursos[0], 0) : '' }))
      }
      if (resumo.volta) {
        partes.push(el('div', { class: 'tras-opcions', role: 'radiogroup', 'aria-label': t('trasladar.volta.que', 'Este documento saíu de aquí. Que fago?') },
          el('p', { text: t('trasladar.volta.que', 'Este documento saíu de aquí. Que fago?') }),
          ...[['actualizar', t('trasladar.volta.actualizar', 'Actualizar o orixinal (recomendado)')], ['copia', t('trasladar.volta.copia', 'Engadir como copia nun proxecto novo')]].map(([v, txt]) =>
            el('label', { class: 'tras-opcion' },
              el('input', { type: 'radio', name: 'trasVolta', value: v, checked: escolla.accion === v, onchange: () => { escolla.accion = v; pintarResumo() } }), ' ', txt))))
      }
      const destinos = opcionsDestino(resumo)
      if (destinos.length) {
        partes.push(el('label', { class: 'tras-destino' }, t('trasladar.destino', 'Destino'), ' ',
          el('select', { onchange: (e) => { escolla.destino = e.target.value; pintarResumo() } },
            destinos.map((d) => el('option', { value: d, selected: escolla.destino === d, text: d === 'aberto' ? t('trasladar.destino.op.aberto', 'O proxecto aberto, «{nome}»', { nome: resumo.nomeProxectoAberto }) : t('trasladar.destino.op.novo', 'Un proxecto novo') })))))
      }
      partes.push(el('p', { class: 'tras-onde', text: textoDestino(resumo, escolla, t) }))
      if (resumo.produccion) partes.push(el('p', { class: 'bib-aviso', text: t('trasladar.produccion.meter', 'Trae datos que poden ser reais: o formulario entra en PRODUCIÓN, sen asistente de IA.') }))
      for (const a of resumo.avisos || []) partes.push(el('p', { class: 'bib-aviso', text: a }))
      partes.push(el('div', { class: 'modal-actions' }, btnFacerMeter))
      zonaResumo.replaceChildren(...partes)
      actualizarBoton()
    }

    const btnFacerMeter = el('button', { type: 'button', class: 'crono-btn primary', text: t('trasladar.meter.boton', 'Meter'), onclick: () => executar(meter) })
    const actualizarBoton = () => { btnFacerMeter.disabled = !podeMeter(resumo, escolla) }

    async function meter() {
      if (!resumo) return
      const pregunta = confirmacionPrevia(resumo)
      if (pregunta === 'restaurar' && !await confirmar(t('trasladar.restaurar.confirmar', 'Restaurar a copia substitúe TODOS os proxectos, a biblioteca e as plantillas polos da copia. O estado actual gárdase antes e podes desfacer. Continuar?'))) return
      await o.antesDeMeter?.()
      const res = await api.AplicarMeter(ruta, opcionsMeter(resumo, escolla))
      resumo = null
      zonaResumo.replaceChildren()
      const accions = []
      if (res.version) {
        accions.push(el('button', { type: 'button', class: 'crono-btn', text: t('trasladar.desfacer', 'Desfacer'), onclick: () => executar(async () => {
          await o.antesDeMeter?.()
          await api.DesfacerTraslado(res.version)
          resultadoMeter.replaceChildren(el('p', { text: t('trasladar.desfeito', 'Desfeito: todo está como antes.') }))
          await o.despoisDeMeter?.({ ...res, recargar: true, desfeito: true })
        }) }))
      }
      resultadoMeter.replaceChildren(
        el('p', { text: res.reiniciar ? t('trasladar.restaurado', 'Copia restaurada.') : t('trasladar.metido', 'Feito. Xa está en Lúa.') }),
        ...(res.avisos || []).map((a) => el('p', { class: 'bib-aviso', text: a })),
        el('div', { class: 'modal-actions' }, ...accions))
      await o.despoisDeMeter?.(res)
    }

    corpoMeter.append(soltar, zonaResumo, resultadoMeter)

    // ---------- Xanela ----------
    const overlay = el('div', { class: 'overlay bib-overlay', role: 'dialog', 'aria-modal': 'true', 'aria-label': t('trasladar.titulo', 'Trasladar'), 'data-file-drop-target': '' })
    const caixa = el('div', { class: 'bib tras' },
      el('header', { class: 'bib__cab' },
        el('h2', { text: t('trasladar.titulo', 'Trasladar') }),
        lapelas,
        el('button', { type: 'button', class: 'dis-x', 'aria-label': t('proxectos.pechar.panel', 'Pechar'), title: t('proxectos.pechar.panel', 'Pechar'), html: ICONA_X, onclick: () => pechar() })),
      erro, corpoSacar, corpoMeter)
    overlay.append(caixa)

    function tecla(e) {
      if (e.key !== 'Escape') return
      if (document.querySelector('.overlay--dialog')) return
      e.stopPropagation()
      pechar()
    }
    function pechar() {
      document.removeEventListener('keydown', tecla, true)
      overlay.remove()
      aberta = null
      if (previo && previo.isConnected) previo.focus?.()
      resolve()
    }
    aberta = {
      meter: (r) => { irA('meter'); executar(() => analizar(r)) },
      lapela: (id) => irA(id),
    }
    document.addEventListener('keydown', tecla, true)
    document.body.append(overlay)
    pintarSacar()
    if (o.ruta) aberta.meter(o.ruta)
    else irA(o.lapela === 'meter' ? 'meter' : 'sacar')
  })
}
