// O control dun campo «problemas» na pantalla de datos dun formulario: dúas
// columnas paralelas, unha fila por pregunta.
//
//   PREGUNTAS (esquerda)                 SOLUCIÓNS (dereita)
//   enunciado (Markdown + $LaTeX$)       baleira ata que se pide: resolver
//   vista coas fórmulas compostas        coa IA + Maxima, só coa IA, ou á man
//   variables: fixa ⇄ variable           a resolución calculada nesta semente
//   cálculos, imaxe, puntos              resultado, comprobacións, fallos
//
// Cambiar unha variable de fixa a variable, o seu rango ou a semente
// recalcula enunciado e solución (Go + Maxima, CalcularProblemas): a
// solución é un modelo cos mesmos {…}, non números copiados.
//
// O control non fala cos bindings: recibe o estado (o calculado, que se
// está a resolver...) e as accións do editor (editor.js).
import { el } from '../informes/dom.js'
import { normalizarImaxe } from '../informes/tarefa-editor.js'
import { htmlMates } from './mates-vista.js'
import { abrirEditorFormulas } from './editor-formulas.js'
import { formulasDe, formulaEn, envolver, substituir } from './formulas-modelo.js'
import {
  MODO_FIXA, MODO_VARIABLE, novoProblema, problemasDe, numerosAVariables, cambiarModo,
  variablesUsadas, nomesDeclarados, solucionDesactualizada, temSolucion,
} from './problemas-modelo.js'

const SVG = (d) => `<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${d}</svg>`
const ICONAS = {
  arriba: SVG('<path d="m5 12 5-5 5 5"/>'),
  abaixo: SVG('<path d="m5 8 5 5 5-5"/>'),
  lixo: SVG('<path d="M4.5 5.5h11"/><path d="M8 5.5V4a1 1 0 0 1 1-1h2a1 1 0 0 1 1 1v1.5"/><path d="M5.5 5.5 6.2 16a1 1 0 0 0 1 .9h5.6a1 1 0 0 0 1-.9l.7-10.5"/>'),
  mais: SVG('<path d="M10 4.5v11M4.5 10h11"/>'),
  imaxe: SVG('<rect x="3.5" y="4.5" width="13" height="11" rx="1.8"/><circle cx="7.5" cy="8.5" r="1.3"/><path d="m4 14 4-3.5 3 2.5 2.5-2 3 3"/>'),
  dado: SVG('<rect x="3.5" y="3.5" width="13" height="13" rx="2.5"/><circle cx="7.3" cy="7.3" r=".9" fill="currentColor"/><circle cx="12.7" cy="12.7" r=".9" fill="currentColor"/><circle cx="10" cy="10" r=".9" fill="currentColor"/>'),
  maxima: SVG('<path d="M4 15.5 7.5 4.5l3 7 2-4 3.5 8"/>'),
  ia: SVG('<path d="M10 2.8 11.6 7.4 16.2 9 11.6 10.6 10 15.2 8.4 10.6 3.8 9 8.4 7.4z"/><path d="M15.5 13.5l.6 1.6 1.6.6-1.6.6-.6 1.6-.6-1.6-1.6-.6 1.6-.6z"/>'),
  formula: SVG('<path d="M3.5 11h2l2 4.5 3.5-11h6"/><path d="m12 9.5 4 4M16 9.5l-4 4"/>'),
  desanclar: SVG('<path d="M11.5 3.5h5v5"/><path d="m16.5 3.5-6 6"/><path d="M8.5 3.5h-4a1 1 0 0 0-1 1v11a1 1 0 0 0 1 1h11a1 1 0 0 0 1-1v-4"/>'),
  anclar: SVG('<path d="M9.5 10.5h-5v-5"/><path d="m4.5 10.5 6-6"/><path d="M11.5 3.5h4a1 1 0 0 1 1 1v11a1 1 0 0 1-1 1h-11a1 1 0 0 1-1-1v-4"/>'),
  man: SVG('<path d="M4 16 14.5 5.5a1.4 1.4 0 0 1 2 2L6 18l-3 1z"/>'),
}

// crearControlProblemas devolve o elemento do campo.
//   campo          o campo do esquema
//   valor          a lista de preguntas
//   estado         { calc: Map id→calculado, maxima, avisoMaxima, motor ('maxima'|'autonomo'),
//                    debuxos: {clave: {imaxe, erro}} (circuítos compilados por Go),
//                    resolvendo: Set ids, informes: Map id→{rexistro, fallos, erro}, editando: Set ids,
//                    iaDispoñible }
//   aoCambiar(lista, motivo)   'texto' (tecleo) ou 'estrutura' (repinta)
//   accions        { resolver(ids, {instrucions}), cambiarMotor(m), instalarMaxima(), indice(id) }
export function crearControlProblemas({ t, campo, valor, estado, aoCambiar, accions }) {
  let lista = problemasDe(valor)
  const mudar = (nova, motivo = 'estrutura') => { lista = nova; aoCambiar(nova, motivo) }
  const mudarUn = (id, f, motivo) => mudar(lista.map((p) => (p.id === id ? f(p) : p)), motivo)

  const marco = el('div', { class: 'prob', 'data-campo': campo.id })

  // ---------- Cabeceira do campo: resolver as súas preguntas ----------
  // (O motor, Maxima e «Resolver o exame» van na cabeceira do formulario:
  // crearBarraProblemas, máis abaixo.)
  const semSolucion = lista.filter((p) => (p.enunciado || '').trim() && !temSolucion(p))
  if (semSolucion.length) {
    marco.append(el('div', { class: 'prob-cab' },
      el('span', { class: 'prob-espazo' }),
      el('button', {
        type: 'button', class: 'crono-btn crono-btn--pequeno', 'data-accion': 'resolver-campo', disabled: !estado.iaDispoñible || estado.resolvendo.size > 0,
        title: estado.iaDispoñible ? t('problemas.resolverCampo.title', 'Resolve as preguntas deste campo que aínda non teñen solución') : t('problemas.semIA', 'Configura un provedor de IA en Opcións para resolver con IA'),
        html: ICONAS.ia + `<span>${semSolucion.length === 1 ? t('problemas.resolverProblema', 'Resolver problema') : t('problemas.resolverProblemas', 'Resolver problemas ({n})', { n: semSolucion.length })}</span>`,
        onclick: () => accions.resolver(semSolucion.map((p) => p.id)),
      })))
  }

  // ---------- Reixa ----------
  const reixa = el('div', { class: 'prob-reixa' },
    el('div', { class: 'prob-col-cab', text: t('problemas.preguntas', 'Preguntas') }),
    el('div', { class: 'prob-col-cab prob-col-cab--sol', text: t('problemas.solucions', 'Solucións') }))
  // Unha pregunta desanclada (estado.solta) vai cos seus dous cadros nun
  // panel que cobre o panel de edición, para escribir con espazo; a barra
  // (Asistente, Xerar) e o documento xerado seguen á vista. Esc ou «Anclar»
  // devólvena á reixa.
  if (estado.solta && !lista.some((p) => p.id === estado.solta)) estado.solta = null
  let panelSolto = null
  lista.forEach((p, i) => {
    if (p.id !== estado.solta) { reixa.append(pregunta(p, i), solucion(p, i)); return }
    panelSolto = el('div', { class: 'prob-solta', role: 'dialog', 'aria-modal': 'true', 'aria-label': t('problemas.pregunta', 'Pregunta {n}', { n: i + 1 }) },
      pregunta(p, i), solucion(p, i))
    panelSolto.addEventListener('keydown', (e) => {
      if (e.key === 'Escape' && !e.defaultPrevented) { e.preventDefault(); alternarSolta(p.id) }
    })
  })
  marco.append(reixa)
  if (panelSolto) { marco.append(panelSolto); requestAnimationFrame(() => encaixarNoEditor(panelSolto)) }
  if (!lista.length) marco.append(el('p', { class: 'form-taboa__baleira', text: t('problemas.baleiro', 'Aínda non hai preguntas.') }))
  marco.append(el('div', { class: 'prob-pe' },
    el('button', {
      type: 'button', class: 'crono-btn crono-btn--pequeno', 'data-accion': 'engadir-pregunta', html: ICONAS.mais + `<span>${t('problemas.engadir', 'Engadir pregunta')}</span>`,
      onclick: () => { const n = novoProblema(); estado.enfocar = n.id; mudar([...lista, n]) },
    }),
    el('small', { class: 'form-camp__axuda', text: t('problemas.axuda', 'Fórmulas con $…$ (en liña) ou $$…$$ (centradas). Unha variable vai entre chaves: {m}. Dentro dunha fórmula, $\\dfrac{m}{V}$ son as letras e $\\dfrac{ {m} }{ {V} }$ os valores.') })))
  return marco

  // ---------- Pregunta ----------
  function pregunta(p, i) {
    const calc = estado.calc.get(p.id)
    const solta = estado.solta === p.id
    const caixa = el('div', { class: 'prob-preg', 'data-prob': p.id })
    const puntos = el('input', { type: 'text', inputmode: 'decimal', class: 'form-entrada prob-puntos', value: p.puntos || '', 'aria-label': t('problemas.puntos', 'Puntos'), placeholder: t('problemas.puntos', 'Puntos') })
    puntos.addEventListener('input', () => mudarUn(p.id, (x) => ({ ...x, puntos: puntos.value }), 'texto'))
    const botonIcona = (icona, titulo, accion, disabled = false) => el('button', { type: 'button', class: 'form-icona-btn', title: titulo, 'aria-label': titulo, html: icona, disabled, onclick: accion })
    caixa.append(el('div', { class: 'prob-preg__cab' },
      el('strong', { text: t('problemas.pregunta', 'Pregunta {n}', { n: i + 1 }) }),
      el('label', { class: 'prob-puntos__lbl' }, el('span', { text: t('problemas.puntos', 'Puntos') }), puntos),
      el('span', { class: 'prob-espazo' }),
      solta && el('button', {
        type: 'button', class: 'crono-btn crono-btn--pequeno prob-btn-asis', 'data-accion': 'asistente-pregunta', 'aria-pressed': String(!!estado.asistente?.get(p.id)?.aberto),
        disabled: !estado.iaDispoñible,
        title: estado.iaDispoñible ? t('problemas.asis.title', 'Asistente da pregunta: mellora a redacción e as fórmulas, pasa os datos a variables ou crea unha pregunta parecida') : t('problemas.semIA', 'Configura un provedor de IA en Opcións para resolver con IA'),
        html: ICONAS.ia + `<span>${t('problemas.asis', 'Asistente')}</span>`,
        onclick: () => { const st = estadoAsis(p.id); st.aberto = !st.aberto; repintar() },
      }),
      el('button', {
        type: 'button', class: 'crono-btn crono-btn--pequeno prob-btn-solta', 'data-accion': 'desanclar',
        title: solta ? t('problemas.anclar.title', 'Volver a pregunta á súa fila (Esc)') : t('problemas.desanclar.title', 'Editar esta pregunta e a súa solución a pantalla completa'),
        html: (solta ? ICONAS.anclar : ICONAS.desanclar) + `<span>${solta ? t('problemas.anclar', 'Anclar') : t('problemas.desanclar', 'Desanclar')}</span>`,
        onclick: () => alternarSolta(p.id),
      }),
      botonIcona(ICONAS.arriba, t('problemas.subir', 'Subir'), () => mover(i, -1), i === 0),
      botonIcona(ICONAS.abaixo, t('problemas.baixar', 'Baixar'), () => mover(i, 1), i === lista.length - 1),
      botonIcona(ICONAS.lixo, t('problemas.quitar', 'Quitar a pregunta'), () => mudar(lista.filter((x) => x.id !== p.id)))))

    if (solta && estado.asistente?.get(p.id)?.aberto) caixa.append(asistentePregunta(p))

    const texto = el('textarea', {
      class: 'form-entrada form-entrada--longa prob-enunciado', rows: '4', spellcheck: 'true', 'data-foco': 'enunciado:' + p.id,
      'aria-label': t('problemas.enunciado', 'Enunciado da pregunta {n}', { n: i + 1 }),
      placeholder: t('problemas.enunciado.ph', 'Ex.: Un corpo de {m} kg acelera a {a} m/s². Calcula a forza: $F = m\\,a$.'),
    })
    texto.value = p.enunciado || ''
    texto.addEventListener('input', () => mudarUn(p.id, (x) => ({ ...x, enunciado: texto.value }), 'texto'))

    // Editor visual de fórmulas: o botón (ou Ctrl+M) edita a fórmula onde
    // está o cursor ou mete unha nova; tamén se abre cun clic na vista.
    const btnFormula = el('button', {
      type: 'button', class: 'crono-btn crono-btn--pequeno prob-btn-formula', 'data-accion': 'formula', html: ICONAS.formula + '<span></span>',
      title: t('formulas.boton.title', 'Escribe a fórmula de forma visual, coma no papel (Ctrl+M)'),
      onmousedown: (e) => e.preventDefault(), // o cursor do texto non se perde
      onclick: () => editarFormula(texto, p.id),
    })
    const rotularBoton = () => {
      btnFormula.lastChild.textContent = formulaEn(texto.value, texto.selectionStart) ? t('formulas.editar', 'Editar fórmula') : t('formulas.inserir', 'Fórmula')
    }
    rotularBoton()
    for (const ev of ['keyup', 'click', 'input', 'focus']) texto.addEventListener(ev, rotularBoton)
    texto.addEventListener('keydown', (e) => {
      if ((e.ctrlKey || e.metaKey) && !e.shiftKey && !e.altKey && e.key.toLowerCase() === 'm') { e.preventDefault(); editarFormula(texto, p.id) }
    })
    caixa.append(el('div', { class: 'prob-ferr' }, btnFormula,
      el('small', { class: 'prob-ferr__axuda', text: t('formulas.boton.axuda', 'ou fai clic nunha fórmula da vista para cambiala') })))
    caixa.append(texto)
    const vista = el('div', { class: 'prob-vista prob-vista--editable', 'data-vista': 'enunciado:' + p.id, 'aria-live': 'polite', html: vistaEnunciado(p, calc) })
    vista.addEventListener('click', (e) => {
      const k = e.target.closest?.('.katex')
      if (!k || !vista.contains(k)) return
      const todas = [...vista.querySelectorAll('.katex')].filter((x) => !x.parentElement.closest('.katex'))
      const f = formulasDe(texto.value)[todas.indexOf(k)]
      if (f) editarFormula(texto, p.id, f)
    })
    caixa.append(vista)
    if (p.imaxe) {
      caixa.append(el('div', { class: 'prob-imaxe' },
        el('img', { src: p.imaxe, alt: t('problemas.imaxe', 'Imaxe da pregunta {n}', { n: i + 1 }) }),
        el('button', { type: 'button', class: 'crono-btn crono-btn--pequeno', text: t('formularios.imaxe.quitar', 'Quitar'), onclick: () => mudarUn(p.id, (x) => ({ ...x, imaxe: '' })) })))
    }
    caixa.append(variables(p, calc))
    caixa.append(calculos(p, 'calculos', t('problemas.calculos', 'Cálculos do enunciado'), t('problemas.calculos.axuda', 'Valores que usa o enunciado, en Maxima: P = expand((x-r1)*(x-r2)) e no enunciado $ {P} = 0 $.')))

    const ficheiro = el('input', { type: 'file', accept: 'image/*', class: 'sr-only', tabindex: '-1' })
    ficheiro.addEventListener('change', async () => {
      const f = ficheiro.files?.[0]
      if (!f) return
      try {
        const uri = await normalizarImaxe(f)
        mudarUn(p.id, (x) => ({ ...x, imaxe: uri }))
      } catch { /* a imaxe non se puido ler: non se cambia nada */ }
    })
    caixa.append(el('div', { class: 'prob-preg__pe' },
      el('button', { type: 'button', class: 'crono-btn crono-btn--pequeno', html: ICONAS.imaxe + `<span>${p.imaxe ? t('formularios.imaxe.cambiar', 'Cambiar imaxe…') : t('problemas.imaxe.engadir', 'Imaxe…')}</span>`, onclick: () => ficheiro.click() }),
      el('button', {
        type: 'button', class: 'crono-btn crono-btn--pequeno', 'data-accion': 'numeros-variables', html: ICONAS.dado + `<span>${t('problemas.numerosVariables', 'Números → variables')}</span>`,
        title: t('problemas.numerosVariables.title', 'Converte os números das fórmulas do enunciado en variables (fixas, co mesmo valor). Despois podes facer variables as que queiras.'),
        onclick: () => {
          const { problema, creadas } = numerosAVariables(lista.find((x) => x.id === p.id))
          if (creadas) mudarUn(p.id, () => problema)
        },
      }),
      ficheiro))
    return caixa
  }

  // alternarSolta desancla a pregunta id (ou ánclaa se xa o está) e leva o
  // foco ao seu enunciado.
  function alternarSolta(id) {
    estado.solta = estado.solta === id ? null : id
    estado.enfocar = id
    repintar()
  }

  // repintar: só cambia a vista (o plan non); sen accions.repintar, coma antes.
  function repintar() {
    if (accions.repintar) accions.repintar()
    else mudar(lista)
  }

  // ---------- Asistente da pregunta (problemas_asistente.go) ----------
  // A IA propón e o docente decide: nada cambia ata «Aceptar». O estado
  // (aberto, ocupado, proposta...) vive en estado.asistente para sobrevivir
  // aos repintados.
  function estadoAsis(id) {
    if (!estado.asistente) estado.asistente = new Map()
    if (!estado.asistente.has(id)) estado.asistente.set(id, { aberto: false, ocupado: false, erro: '', proposta: null, accion: '', instrucions: '', peticion: 0 })
    return estado.asistente.get(id)
  }

  async function pedirAsistente(id, accion) {
    const st = estadoAsis(id)
    const n = ++st.peticion
    Object.assign(st, { ocupado: true, erro: '', proposta: null, accion })
    repintar()
    try {
      const r = await accions.asistentePregunta(id, { accion, instrucions: st.instrucions })
      if (st.peticion !== n) return // cancelouse ou pediuse outra
      st.proposta = r
    } catch (err) {
      if (st.peticion !== n) return
      st.erro = String(err?.message || err)
    }
    st.ocupado = false
    repintar()
  }

  function aceptarProposta(p) {
    const st = estadoAsis(p.id)
    const r = st.proposta
    if (!r) return
    const datos = { enunciado: r.enunciado, variables: r.variables || [], calculos: r.calculos || [] }
    st.proposta = null
    if (st.accion === 'parecida') {
      const nova = { ...novoProblema(), ...datos, puntos: p.puntos || '' }
      const i = lista.findIndex((x) => x.id === p.id)
      mudar([...lista.slice(0, i + 1), nova, ...lista.slice(i + 1)])
    } else {
      mudarUn(p.id, (x) => ({ ...x, ...datos }))
    }
  }

  function asistentePregunta(p) {
    const st = estadoAsis(p.id)
    const caixa = el('div', { class: 'prob-asis', role: 'region', 'aria-label': t('problemas.asis.titulo', 'Asistente da pregunta') })
    const accionsRapidas = [
      ['mellorar', t('problemas.asis.mellorar', 'Mellorar a redacción')],
      ['variables', t('problemas.asis.variables', 'Números → variables')],
      ['formulas', t('problemas.asis.formulas', 'Revisar as fórmulas')],
      ['parecida', t('problemas.asis.parecida', 'Crear unha parecida')],
    ]
    caixa.append(el('div', { class: 'prob-asis__accions' }, ...accionsRapidas.map(([id, nome]) => el('button', {
      type: 'button', class: 'prob-asis__chip', 'data-asis': id, text: nome, disabled: st.ocupado,
      onclick: () => pedirAsistente(p.id, id),
    }))))
    const instr = el('input', {
      type: 'text', class: 'form-entrada prob-asis__instr', value: st.instrucions, disabled: st.ocupado,
      placeholder: t('problemas.asis.ph', 'Ou pídelle outra cousa: «faina máis difícil, con rozamento»…'),
      'aria-label': t('problemas.asis.instr', 'Indicacións para o asistente'),
    })
    instr.addEventListener('input', () => { st.instrucions = instr.value })
    instr.addEventListener('keydown', (e) => { if (e.key === 'Enter' && instr.value.trim()) { e.preventDefault(); pedirAsistente(p.id, 'libre') } })
    caixa.append(el('div', { class: 'prob-asis__fila' }, instr,
      el('button', { type: 'button', class: 'crono-btn crono-btn--pequeno', 'data-asis': 'libre', disabled: st.ocupado, text: t('problemas.asis.pedir', 'Pedir'), onclick: () => { if (instr.value.trim()) pedirAsistente(p.id, 'libre') } })))
    if (st.ocupado) {
      caixa.append(el('div', { class: 'prob-asis__ocupado', role: 'status' },
        el('span', { text: t('problemas.asis.ocupado', 'A IA está a preparar a proposta e Lúa a comprobala…') }),
        el('button', { type: 'button', class: 'crono-btn crono-btn--pequeno', 'data-accion': 'asis-cancelar', text: t('problemas.asis.cancelar', 'Cancelar'), onclick: () => { st.peticion++; st.ocupado = false; repintar() } })))
    }
    if (st.erro) caixa.append(el('p', { class: 'prob-erro', role: 'alert', text: st.erro }))
    const r = st.proposta
    if (r) {
      const vars = (r.variables || []).map((v) => (v.modo === MODO_VARIABLE ? `${v.nome}: ${v.de}–${v.ata}` : `${v.nome} = ${v.valor}`))
      caixa.append(el('div', { class: 'prob-asis__proposta' },
        el('div', { class: 'prob-asis__tit', text: st.accion === 'parecida' ? t('problemas.asis.proposta.nova', 'Pregunta nova proposta') : t('problemas.asis.proposta', 'Proposta') }),
        r.explicacion && el('p', { class: 'prob-asis__expl', text: r.explicacion }),
        el('div', { class: 'prob-vista', html: htmlMates(r.calculado?.enunciado || r.enunciado, estado.debuxos) }),
        vars.length ? el('div', { class: 'prob-asis__vars' }, ...vars.map((v) => el('code', { text: v }))) : null,
        r.fallos?.length ? el('p', { class: 'prob-erro', text: t('problemas.asis.fallos', 'Ollo, Lúa atopou isto: {f}', { f: r.fallos.join(' · ') }) }) : null,
        el('div', { class: 'prob-asis__botons' },
          el('button', { type: 'button', class: 'crono-btn crono-btn--pequeno', 'data-accion': 'asis-descartar', text: t('problemas.asis.descartar', 'Descartar'), onclick: () => { st.proposta = null; repintar() } }),
          el('button', {
            type: 'button', class: 'crono-btn crono-btn--pequeno prob-asis__aceptar', 'data-accion': 'asis-aceptar',
            text: st.accion === 'parecida' ? t('problemas.asis.engadir', 'Engadir coma pregunta nova') : t('problemas.asis.aceptar', 'Aceptar e substituír'),
            onclick: () => aceptarProposta(p),
          }))))
    }
    return caixa
  }

  // editarFormula abre o editor visual coa fórmula f (ou a do cursor, ou o
  // texto seleccionado) e pon o resultado no enunciado coma se se tecleara.
  async function editarFormula(texto, id, f = null) {
    const ini = texto.selectionStart
    const fin = texto.selectionEnd
    f = f || formulaEn(texto.value, ini)
    const rango = f || { ini, fin, tex: texto.value.slice(ini, fin).trim(), centrada: false }
    const p = lista.find((x) => x.id === id)
    const res = await abrirEditorFormulas({ t, tex: rango.tex, centrada: rango.centrada, variables: p ? [...nomesDeclarados(p)] : [] })
    if (!res || (!f && !res.tex)) { texto.focus(); return }
    const { texto: novo, cursor } = substituir(texto.value, envolver(res.tex, res.centrada), { ini: rango.ini, fin: rango.fin, centrada: res.centrada })
    texto.value = novo
    texto.focus()
    texto.setSelectionRange(cursor, cursor)
    texto.dispatchEvent(new Event('input', { bubbles: true }))
  }

  function vistaEnunciado(p, calc) {
    if (!(p.enunciado || '').trim()) return `<p class="prob-vista__baleira">${t('problemas.vista.baleira', 'Aquí verás a pregunta coas fórmulas e os valores.')}</p>`
    if (calc?.erro) return `<p class="prob-erro">${escapar(calc.erro)}</p>` + htmlMates(p.enunciado, estado.debuxos)
    return htmlMates(calc ? calc.enunciado : p.enunciado, estado.debuxos)
  }

  function mover(i, d) {
    const j = i + d
    if (j < 0 || j >= lista.length) return
    const nova = [...lista];
    [nova[i], nova[j]] = [nova[j], nova[i]]
    mudar(nova)
  }

  // ---------- Variables ----------
  function variables(p, calc) {
    // Unha variable úsase se sae no enunciado ou nun cálculo do enunciado.
    const usadas = new Set([...variablesUsadas(p.enunciado), ...(p.calculos || []).flatMap((c) => String(c.expr || '').match(/[A-Za-z_]\w*/g) || [])])
    const vars = p.variables || []
    const caixa = el('div', { class: 'prob-vars' })
    if (!vars.length) {
      caixa.append(el('small', { class: 'form-camp__axuda', text: t('problemas.vars.ningunha', 'Sen variables. Escribe {nome} no enunciado (ou usa «Números → variables») e poderás facelas fixas ou variables.') }))
      return caixa
    }
    caixa.append(el('div', { class: 'prob-vars__tit', text: t('problemas.vars', 'Variables') }))
    vars.forEach((v, j) => {
      const poñer = (nv, motivo = 'estrutura') => mudarUn(p.id, (x) => ({ ...x, variables: x.variables.map((y, k) => (k === j ? nv : y)) }), motivo)
      const agora = calc?.valores?.[v.nome]?.texto
      const fila = el('div', { class: 'prob-var' + (usadas.has(v.nome) || nomesSolucion(p).has(v.nome) ? '' : ' prob-var--sen-uso'), 'data-variable': v.nome })
      fila.append(el('code', { class: 'prob-var__nome', text: v.nome }))
      // Interruptor fixa / variable (dous botóns, coma un segmentado).
      const seg = el('div', { class: 'prob-seg', role: 'group', 'aria-label': t('problemas.var.modo', 'Modo de {v}', { v: v.nome }) })
      for (const [modo, nome, axuda] of [
        [MODO_FIXA, t('problemas.var.fixa', 'Fixa'), t('problemas.var.fixa.title', 'Sempre o mesmo valor')],
        [MODO_VARIABLE, t('problemas.var.variable', 'Variable'), t('problemas.var.variable.title', 'Un valor ao chou do rango en cada semente (cada copia do exame, un)')],
      ]) {
        seg.append(el('button', {
          type: 'button', class: 'prob-seg__btn', 'aria-pressed': String((v.modo || MODO_FIXA) === modo), title: axuda, text: nome,
          onclick: () => { if ((v.modo || MODO_FIXA) !== modo) poñer(cambiarModo(v, agora)) },
        }))
      }
      fila.append(seg)
      const campoNum = (clave, etiqueta, ancho = '') => {
        const i = el('input', { type: 'text', inputmode: 'decimal', class: 'form-entrada prob-var__num' + ancho, value: v[clave] ?? '', 'aria-label': `${etiqueta} (${v.nome})`, placeholder: etiqueta })
        i.addEventListener('input', () => { v = { ...v, [clave]: i.value }; poñer(v, 'texto') })
        return i
      }
      if ((v.modo || MODO_FIXA) === MODO_VARIABLE) {
        fila.append(el('span', { class: 'prob-var__rango' },
          el('span', { text: t('problemas.var.de', 'de') }), campoNum('de', t('problemas.var.de', 'de')),
          el('span', { text: t('problemas.var.ata', 'ata') }), campoNum('ata', t('problemas.var.ata', 'ata')),
          el('span', { text: t('problemas.var.paso', 'paso') }), campoNum('paso', t('problemas.var.paso', 'paso'), ' prob-var__num--paso')))
      } else {
        fila.append(el('span', { text: '=' }), campoNum('valor', t('problemas.var.valor', 'valor'), ' prob-var__num--valor'))
      }
      fila.append(el('span', { class: 'prob-var__agora', 'data-agora': p.id + ':' + v.nome, title: t('problemas.var.agora.title', 'O valor nesta semente'), text: agora !== undefined ? '→ ' + agora : '' }))
      if (!usadas.has(v.nome)) {
        fila.append(el('button', {
          type: 'button', class: 'form-icona-btn', html: ICONAS.lixo, title: t('problemas.var.quitar', 'O enunciado xa non usa {v}: quitala', { v: v.nome }),
          'aria-label': t('problemas.var.quitar', 'O enunciado xa non usa {v}: quitala', { v: v.nome }),
          onclick: () => mudarUn(p.id, (x) => ({ ...x, variables: x.variables.filter((_, k) => k !== j) })),
        }))
      }
      caixa.append(fila)
    })
    return caixa
  }

  function nomesSolucion(p) {
    const s = p.solucion
    if (!s) return new Set()
    return new Set([...variablesUsadas(s.texto || ''), ...variablesUsadas(s.resultado || ''), ...(s.calculos || []).flatMap((c) => String(c.expr || '').match(/[A-Za-z_]\w*/g) || [])])
  }

  // ---------- Cálculos (do enunciado ou da solución) ----------
  function calculos(p, onde, titulo, axuda) {
    const lerLista = (x) => (onde === 'calculos' ? x.calculos : x.solucion?.calculos) || []
    const gardar = (x, cs) => (onde === 'calculos' ? { ...x, calculos: cs } : { ...x, solucion: { ...x.solucion, calculos: cs } })
    const cs = lerLista(p)
    const det = el('details', { class: 'prob-calc' })
    det.open = cs.length > 0 && onde !== 'calculos'
    det.append(el('summary', { text: `${titulo}${cs.length ? ` (${cs.length})` : ''}` }), el('small', { class: 'form-camp__axuda', text: axuda }))
    cs.forEach((c, j) => {
      const nome = el('input', { type: 'text', class: 'form-entrada prob-calc__nome', value: c.nome || '', 'aria-label': t('problemas.calc.nome', 'Nome'), placeholder: 'F' })
      const expr = el('input', { type: 'text', class: 'form-entrada prob-calc__expr', value: c.expr || '', spellcheck: 'false', 'aria-label': t('problemas.calc.expr', 'Expresión'), placeholder: 'm*a' })
      const poñer = (motivo) => mudarUn(p.id, (x) => gardar(x, lerLista(x).map((y, k) => (k === j ? { nome: nome.value, expr: expr.value } : y))), motivo)
      nome.addEventListener('input', () => poñer('texto'))
      expr.addEventListener('input', () => poñer('texto'))
      det.append(el('div', { class: 'prob-calc__fila' }, nome, el('span', { text: '=' }), expr,
        el('button', { type: 'button', class: 'form-icona-btn', html: ICONAS.lixo, title: t('problemas.calc.quitar', 'Quitar'), 'aria-label': t('problemas.calc.quitar', 'Quitar'), onclick: () => mudarUn(p.id, (x) => gardar(x, lerLista(x).filter((_, k) => k !== j))) })))
    })
    det.append(el('button', {
      type: 'button', class: 'crono-btn crono-btn--pequeno', html: ICONAS.mais + `<span>${t('problemas.calc.engadir', 'Engadir cálculo')}</span>`,
      onclick: () => mudarUn(p.id, (x) => gardar(x, [...lerLista(x), { nome: '', expr: '' }])),
    }))
    return det
  }

  // ---------- Solución ----------
  function solucion(p, i) {
    const calc = estado.calc.get(p.id)
    const inf = estado.informes.get(p.id)
    const ocupado = estado.resolvendo.has(p.id)
    const caixa = el('div', { class: 'prob-sol' + (temSolucion(p) ? '' : ' prob-sol--baleira'), 'data-sol': p.id })
    const cab = el('div', { class: 'prob-preg__cab' }, el('strong', { text: t('problemas.solucion', 'Solución {n}', { n: i + 1 }) }))
    if (p.solucion?.orixe) {
      const nomes = { ia: t('problemas.orixe.ia', 'IA'), 'ia+maxima': t('problemas.orixe.iamaxima', 'IA + Maxima'), man: t('problemas.orixe.man', 'Á man') }
      cab.append(el('span', { class: 'prob-chip prob-chip--' + (p.solucion.orixe === 'ia+maxima' ? 'ok' : 'neutro'), text: nomes[p.solucion.orixe] || p.solucion.orixe }))
    }
    caixa.append(cab)

    if (estado.cola?.has(p.id)) {
      caixa.append(el('p', { class: 'prob-vista__baleira', role: 'status', text: t('problemas.enCola', 'En cola: resólvese despois das anteriores…') }))
      return caixa
    }
    if (ocupado) {
      caixa.append(el('p', { class: 'prob-ocupado', role: 'status', text: estado.motor === 'maxima' ? t('problemas.resolvendo.maxima', 'A IA está a resolver e Maxima a comprobar…') : t('problemas.resolvendo', 'A IA está a resolver…') }))
      return caixa
    }
    if (inf?.erro) caixa.append(el('p', { class: 'prob-erro', role: 'alert', text: inf.erro }))

    if (!temSolucion(p) && !estado.editando.has(p.id)) {
      const instr = el('input', { type: 'text', class: 'form-entrada prob-instr', placeholder: t('problemas.instrucions.ph', 'Indicacións para a IA (opcional): «resolve por enerxías»…'), 'aria-label': t('problemas.instrucions', 'Indicacións para a IA') })
      const sen = !(p.enunciado || '').trim() && !p.imaxe
      caixa.append(
        el('p', { class: 'prob-vista__baleira', text: t('problemas.sol.baleira', 'Sen solución. Énchese só cando o pidas.') }),
        instr,
        el('div', { class: 'prob-sol__accions' },
          el('button', {
            type: 'button', class: 'crono-btn', 'data-accion': 'resolver', disabled: sen || !estado.iaDispoñible || estado.resolvendo.size > 0,
            title: estado.iaDispoñible ? '' : t('problemas.semIA', 'Configura un provedor de IA en Opcións para resolver con IA'),
            html: ICONAS.ia + `<span>${estado.motor === 'maxima' ? t('problemas.resolver.maxima', 'Resolver (IA + Maxima)') : t('problemas.resolver.ia', 'Resolver coa IA')}</span>`,
            onclick: () => accions.resolver([p.id], { instrucions: instr.value }),
          }),
          el('button', {
            type: 'button', class: 'crono-btn crono-btn--pequeno', html: ICONAS.man + `<span>${t('problemas.aman', 'Escribila eu')}</span>`,
            onclick: () => { estado.editando.add(p.id); mudarUn(p.id, (x) => ({ ...x, solucion: { texto: '', resultado: '', orixe: 'man', parametrica: true } })) },
          })))
      return caixa
    }

    if (estado.editando.has(p.id)) {
      const s = p.solucion || {}
      const area = (clave, etiqueta, filas, ph) => {
        const a = el('textarea', { class: 'form-entrada form-entrada--longa', rows: String(filas), spellcheck: 'true', 'aria-label': etiqueta, placeholder: ph })
        a.value = s[clave] || ''
        a.addEventListener('input', () => mudarUn(p.id, (x) => ({ ...x, solucion: { ...x.solucion, [clave]: a.value } }), 'texto'))
        return el('label', { class: 'form-camp' }, el('span', { class: 'form-camp__cab' }, el('span', { class: 'form-camp__etq', text: etiqueta })), a)
      }
      const comp = el('textarea', { class: 'form-entrada', rows: '2', spellcheck: 'false', 'aria-label': t('problemas.comprobacions', 'Comprobacións') })
      comp.value = (s.comprobacions || []).join('\n')
      comp.addEventListener('input', () => mudarUn(p.id, (x) => ({ ...x, solucion: { ...x.solucion, comprobacions: comp.value.split('\n').map((l) => l.trim()).filter(Boolean) } }), 'texto'))
      caixa.append(
        area('texto', t('problemas.sol.texto', 'Resolución'), 6, t('problemas.sol.texto.ph', 'Paso a paso, con {variables} e $fórmulas$: $F = m\\,a = {F}$ N')),
        area('resultado', t('problemas.sol.resultado', 'Resultado'), 2, '$F = {F}$ N'),
        calculos(p, 'solucion', t('problemas.sol.calculos', 'Cálculos da solución'), t('problemas.sol.calculos.axuda', 'Cada cálculo dá un valor que podes poñer coma {nome}: F = m*a. Con Maxima vale todo (solve, diff, factor...).')),
        el('details', { class: 'prob-calc' }, el('summary', { text: t('problemas.comprobacions', 'Comprobacións') }),
          el('small', { class: 'form-camp__axuda', text: t('problemas.comprobacions.axuda', 'Unha condición por liña que ten que cumprirse sempre (Maxima): F > 0') }), comp),
        el('div', { class: 'prob-vista prob-vista--sol', 'data-vista': 'solucion:' + p.id, html: vistaSolucion(p, calc) }),
        el('div', { class: 'prob-sol__accions' },
          el('button', { type: 'button', class: 'crono-btn crono-btn--pequeno', text: t('problemas.sol.listo', 'Listo'), onclick: () => { estado.editando.delete(p.id); mudar(lista) } })))
      return caixa
    }

    caixa.append(el('div', { class: 'prob-vista prob-vista--sol', 'data-vista': 'solucion:' + p.id, html: vistaSolucion(p, calc) }))
    if (solucionDesactualizada(p, calc)) {
      caixa.append(el('p', { class: 'prob-aviso', role: 'status', text: t('problemas.sol.vella', 'Esta solución non é paramétrica e resolveuse con outros valores: resólvea de novo ou fai fixas as variables.') }))
    } else if (p.solucion && !p.solucion.parametrica && (p.variables || []).some((v) => v.modo === MODO_VARIABLE)) {
      caixa.append(el('p', { class: 'prob-aviso', text: t('problemas.sol.nonparam', 'A IA non a puido facer paramétrica: só vale para estes valores.') }))
    }
    const fallos = [...(calc?.fallos || []), ...(inf?.fallos || [])]
    if (fallos.length) caixa.append(el('ul', { class: 'prob-fallos', role: 'status' }, ...[...new Set(fallos)].map((f) => el('li', { text: f }))))
    if (inf?.rexistro?.length) caixa.append(el('details', { class: 'prob-rexistro' }, el('summary', { text: t('problemas.rexistro', 'Como se resolveu') }), ...inf.rexistro.map((r) => el('p', { text: r }))))
    caixa.append(el('div', { class: 'prob-sol__accions' },
      el('button', { type: 'button', class: 'crono-btn crono-btn--pequeno', html: ICONAS.man + `<span>${t('problemas.sol.editar', 'Editar')}</span>`, onclick: () => { estado.editando.add(p.id); mudar(lista) } }),
      el('button', {
        type: 'button', class: 'crono-btn crono-btn--pequeno', 'data-accion': 'resolver', disabled: !estado.iaDispoñible || estado.resolvendo.size > 0,
        html: ICONAS.ia + `<span>${t('problemas.sol.outra', 'Resolver de novo')}</span>`, onclick: () => accions.resolver([p.id]),
      }),
      el('button', { type: 'button', class: 'crono-btn crono-btn--pequeno', html: ICONAS.lixo + `<span>${t('problemas.sol.borrar', 'Borrar')}</span>`, onclick: () => { estado.informes.delete(p.id); mudarUn(p.id, (x) => { const { solucion: _s, ...resto } = x; return resto }) } })))
    return caixa
  }

  function vistaSolucion(p, calc) {
    const s = p.solucion || {}
    return htmlSolucion(calc ? calc.solucion : s.texto, calc ? calc.resultado : s.resultado, t, estado.debuxos) ||
      `<p class="prob-vista__baleira">${t('problemas.sol.vista.baleira', 'Escribe a resolución: aquí verás como queda.')}</p>`
  }

}

// htmlSolucion: a resolución e o resultado coas fórmulas compostas.
function htmlSolucion(texto, resultado, t, debuxos) {
  let h = String(texto || '').trim() ? htmlMates(texto, debuxos) : ''
  if (String(resultado || '').trim()) {
    h += `<div class="prob-resultado"><strong>${escapar(t('problemas.sol.resultado', 'Resultado'))}:</strong> ${htmlMates(resultado, debuxos)}</div>`
  }
  return h
}

// cssEsc: CSS.escape onde exista (no WebView sempre; en jsdom non).
export const cssEsc = (s) => (globalThis.CSS?.escape ? globalThis.CSS.escape(s) : String(s).replace(/["\\]/g, '\\$&'))

// crearBarraProblemas: a cabeceira común do formulario cando ten campos de
// problemas. Maxima, como resolver (IA + Maxima ou só a IA) e «Resolver o
// exame», que resolve dunha vez TODAS as preguntas sen solución de TODOS os
// campos de problemas. `pendentes` son os seus ids.
export function crearBarraProblemas({ t, estado, accions, pendentes }) {
  const motorSel = el('select', { class: 'form-entrada prob-motor', 'aria-label': t('problemas.motor', 'Como resolver') })
  motorSel.append(
    new Option(t('problemas.motor.maxima', 'IA + Maxima (cálculo exacto e comprobado)'), 'maxima'),
    new Option(t('problemas.motor.autonomo', 'Só a IA (sen Maxima)'), 'autonomo'),
  )
  motorSel.value = estado.motor
  motorSel.addEventListener('change', () => accions.cambiarMotor(motorSel.value))
  const chipMaxima = !estado.maximaComprobado
    ? el('span', { class: 'prob-chip', html: ICONAS.maxima + `<span>${t('problemas.maxima.comprobando', 'Calculando…')}</span>` })
    : estado.maxima
      ? el('span', { class: 'prob-chip prob-chip--ok', title: t('problemas.maxima.ok.title', 'Os cálculos fanse con Maxima, coma en Yang'), html: ICONAS.maxima + `<span>${t('problemas.maxima.ok', 'Maxima activo')}</span>` })
      : el('span', { class: 'prob-chip prob-chip--aviso' },
        el('span', { html: ICONAS.maxima }),
        el('span', { text: estado.avisoMaxima || t('problemas.maxima.non', 'Sen Maxima: só números e fórmulas simples') }),
        el('button', { type: 'button', class: 'crono-btn crono-btn--pequeno', 'data-accion': 'instalar-maxima', text: t('problemas.maxima.instalar', 'Instalar Maxima'), onclick: () => accions.instalarMaxima() }))
  const faltan = estado.resolvendo.size + (estado.cola?.size || 0)
  const ocupado = faltan > 0
  return el('div', { class: 'prob-barra' },
    chipMaxima,
    el('span', { class: 'prob-espazo' }),
    motorSel,
    el('button', {
      type: 'button', class: 'crono-btn crono-btn--primario', 'data-accion': 'resolver-exame', disabled: !pendentes.length || !estado.iaDispoñible || ocupado,
      title: !estado.iaDispoñible ? t('problemas.semIA', 'Configura un provedor de IA en Opcións para resolver con IA')
        : pendentes.length ? t('problemas.resolverExame.title', 'Resolve dunha vez todas as preguntas do formulario que aínda non teñen solución') : t('problemas.resolverExame.feito', 'Todas as preguntas teñen solución'),
      html: ICONAS.ia + `<span>${ocupado ? t('problemas.resolvendoExame', 'Resolvendo… ({n} pendentes)', { n: faltan }) : t('problemas.resolverExame', 'Resolver o exame')}${!ocupado && pendentes.length ? ` (${pendentes.length})` : ''}</span>`,
      onclick: () => accions.resolver(pendentes),
    }))
}

// encaixarNoEditor pon o panel dunha pregunta desanclada enriba do panel de
// edición da xanela (.pane-editor) e segue o seu tamaño (divisor, asistente,
// xanela). Sen .pane-editor (probas) cobre a xanela enteira, polo CSS.
function encaixarNoEditor(panel) {
  const pane = panel.isConnected && panel.closest('.pane-editor')
  if (!pane) return
  const ro = typeof ResizeObserver !== 'undefined' ? new ResizeObserver(() => poñer()) : null
  function poñer() {
    if (!panel.isConnected) { ro?.disconnect(); window.removeEventListener('resize', poñer); return }
    const r = pane.getBoundingClientRect()
    Object.assign(panel.style, { top: r.top + 'px', left: r.left + 'px', width: r.width + 'px', height: r.height + 'px', right: 'auto', bottom: 'auto' })
  }
  poñer()
  ro?.observe(pane)
  window.addEventListener('resize', poñer)
}

function escapar(s) {
  return String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

// actualizarVistasProblemas repinta só as vistas calculadas (enunciados,
// solucións e valores de agora) sen tocar os textos que se están a escribir.
export function actualizarVistasProblemas(container, lista, estado, t) {
  for (const p of lista) {
    const calc = estado.calc.get(p.id)
    const ve = container.querySelector(`[data-vista="enunciado:${cssEsc(p.id)}"]`)
    if (ve) {
      ve.innerHTML = !(p.enunciado || '').trim()
        ? `<p class="prob-vista__baleira">${t('problemas.vista.baleira', 'Aquí verás a pregunta coas fórmulas e os valores.')}</p>`
        : (calc?.erro ? `<p class="prob-erro">${escapar(calc.erro)}</p>` + htmlMates(p.enunciado, estado.debuxos) : htmlMates(calc ? calc.enunciado : p.enunciado, estado.debuxos))
    }
    const vs = container.querySelector(`[data-vista="solucion:${cssEsc(p.id)}"]`)
    if (vs && calc) {
      const h = htmlSolucion(calc.solucion, calc.resultado, t, estado.debuxos)
      if (h) vs.innerHTML = h
    }
    for (const v of p.variables || []) {
      const a = container.querySelector(`[data-agora="${cssEsc(p.id + ':' + v.nome)}"]`)
      const val = calc?.valores?.[v.nome]?.texto
      if (a) a.textContent = val !== undefined ? '→ ' + val : ''
    }
  }
}

export { nomesDeclarados }
