// abrirDisenador - o deseñador gráfico de formularios: unha paleta de campos
// (esquerda) que se arrastran ao lenzo (centro), onde se reordenan, e o panel
// de propiedades do campo escollido (dereita). Gardar manda o esquema a Go
// (GardarFormulario), que lle pon os identificadores e o valida.
//
// Todo se pode facer sen arrastrar: premer nun tipo da paleta engádeo, e cada
// tarxeta ten botóns para subir, baixar, duplicar e quitar.
import { el } from '../informes/dom.js'
import { iniciarArrastre } from './arrastrar.js'
import { svgTipo } from './iconas.js'
import {
  TIPOS, TIPOS_COLUMNA, NOME_TIPO, novoEsquema, novoCampo, inserirCampo, moverCampo, eliminarCampo, cambiarCampo,
  duplicarCampo, problemasEsquema, completarIds,
} from './modelo.js'
import { montarFluxoEditor } from './fluxo-editor.js'
import { fluxoIgual } from './fluxo-modelo.js'
import { crearPanelAsistente } from '../asistente/panel.js'
import { crearFerramentasFormulario } from '../asistente/ferramentas-formulario.js'
import { TAREFAS_FORMULARIO } from '../asistente/tarefas.js'
import { logoProvedor } from '../asistente/logos.js'

const ICONA_X = '<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round"><path d="M5 5l10 10M15 5L5 15"/></svg>'
const ICONA_SUBIR = '<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="m5 12 5-5 5 5"/></svg>'
const ICONA_BAIXAR = '<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="m5 8 5 5 5-5"/></svg>'
const ICONA_COPIAR = '<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><rect x="7" y="7" width="9" height="9" rx="1.5"/><path d="M13 7V5.5A1.5 1.5 0 0 0 11.5 4h-6A1.5 1.5 0 0 0 4 5.5v6A1.5 1.5 0 0 0 5.5 13H7"/></svg>'
const ASA = '<svg viewBox="0 0 20 20" fill="currentColor" aria-hidden="true"><circle cx="7.5" cy="5" r="1.5"/><circle cx="12.5" cy="5" r="1.5"/><circle cx="7.5" cy="10" r="1.5"/><circle cx="12.5" cy="10" r="1.5"/><circle cx="7.5" cy="15" r="1.5"/><circle cx="12.5" cy="15" r="1.5"/></svg>'

// abrirDisenador devolve unha promesa que se resolve ao pechar coa acción:
// {accion: 'gardado', esquema} | {accion: 'eliminado', id} | null (cancelado).
//   esquema: o formulario a editar, ou null/undefined para un novo.
//   api: {GardarFormulario, EliminarFormulario, CatalogoFluxo, FluxoAutomaticoDe,
//        SincronizarFluxoFormulario, AvaliarFluxoFormulario}; confirmar(mensaxe) -> Promise<boolean>.
//   obterDatos(): os datos do documento que hai aberto deste formulario ({contexto, valores}),
//        para ver nas tuberías o que dan cos datos de agora.
//   asistente: {api, idioma} co que se monta o asistente de IA (ámbito «formulario»); sen el, non hai asistente.
//   alvo: un elemento onde pintarse a pantalla completa (a xanela propia do deseñador, ver
//        xanelaformulario.go); sen el, píntase coma unha capa sobre a xanela actual.
//   aoSucio(bool): hai (ou xa non hai) cambios sen gardar. aoPreparado({pedirPechar}): para que a
//        xanela poida pedir o pechado (con confirmación) cando o xestor de xanelas o intenta.
export function abrirDisenador({ t: tIn, esquema, api, confirmar, obterDatos = () => ({}), asistente: opcAsist = null, alvo = null, aoSucio, aoPreparado }) {
  const t = tIn || ((clave, fallback, vars) => {
    let val = fallback ?? clave
    if (vars) for (const k in vars) val = val.replaceAll(`{${k}}`, vars[k])
    return val
  })
  const nome = (tipo) => t('formularios.tipo.' + tipo, NOME_TIPO[tipo])
  const existente = !!(esquema && esquema.id)
  let est = JSON.parse(JSON.stringify(esquema || novoEsquema()))
  est.campos = est.campos || []
  let sel = est.campos.length ? 0 : -1
  let sucio = false
  let gardando = false
  let pestana = 'campos' // 'campos' | 'fluxo'
  let catalogo = null
  let fluxoEd = null
  let asistente = null
  const pilaIA = [] // instantáneas do borrador antes de cada cambio da IA
  const marcas = {} // as caixas «Con portada» e «Con sinatura»

  return new Promise((resolve) => {
    const overlay = el('div', { class: 'overlay dis-overlay', role: 'dialog', 'aria-modal': 'true', 'aria-label': t('formularios.dis.titulo', 'Deseñar formulario') })
    const caixa = el('div', { class: 'dis' + (alvo ? ' dis--xanela' : '') })
    if (alvo) { caixa.setAttribute('role', 'dialog'); caixa.setAttribute('aria-label', t('formularios.dis.titulo', 'Deseñar formulario')) } else overlay.append(caixa)

    const erro = el('p', { class: 'crono-erro dis-erro', role: 'alert', hidden: true })
    const paleta = el('div', { class: 'dis-paleta' })
    const lenzo = el('div', { class: 'dis-lenzo', role: 'list' })
    const props = el('div', { class: 'dis-props' })

    const marcar = () => { if (!sucio) aoSucio?.(true); sucio = true }
    const mostrarErro = (msg) => { erro.textContent = msg || ''; erro.hidden = !msg }

    // ---------- Cabeceira ----------
    const inNome = el('input', { type: 'text', class: 'form-entrada', maxlength: '80', placeholder: t('formularios.dis.nome.ph', 'Nome do formulario (p. ex. Informe de titoría)'), 'aria-label': t('formularios.dis.nome', 'Nome do formulario') })
    inNome.value = est.nome || ''
    inNome.addEventListener('input', () => { est.nome = inNome.value; marcar(); mostrarErro('') })
    const inDescricion = el('input', { type: 'text', class: 'form-entrada', maxlength: '200', placeholder: t('formularios.dis.descricion.ph', 'Descrición (opcional: sae baixo o título do documento)'), 'aria-label': t('formularios.dis.descricion', 'Descrición') })
    inDescricion.value = est.descricion || ''
    inDescricion.addEventListener('input', () => { est.descricion = inDescricion.value; marcar() })
    const marca = (clave, etiqueta, axuda) => {
      const i = el('input', { type: 'checkbox', class: 'form-marca' })
      i.checked = !!est[clave]
      i.addEventListener('change', () => { est[clave] = i.checked; marcar() })
      marcas[clave] = i
      return el('label', { class: 'dis-opcion', title: axuda }, i, el('span', { text: etiqueta }))
    }

    // Pestanas (Campos | Tuberías) e botóns do asistente.
    const tabCampos = el('button', { type: 'button', class: 'dis-tab dis-tab--sel', role: 'tab', 'aria-selected': 'true', onclick: () => cambiarPestana('campos') }, t('formularios.dis.tab.campos', 'Campos'))
    const tabFluxo = el('button', { type: 'button', class: 'dis-tab', role: 'tab', 'aria-selected': 'false', onclick: () => cambiarPestana('fluxo'), title: t('formularios.dis.tab.tuberias.title', 'Cálculos, filtros e a orde do documento') }, t('formularios.dis.tab.tuberias', 'Tubaxes'))
    const btnDesfacerIA = el('button', { type: 'button', class: 'dis-desfacer-ia', hidden: true, onclick: () => desfacerIA(), title: t('formularios.dis.desfacerIA.title', 'Desfai o último cambio que fixo o asistente') }, t('formularios.dis.desfacerIA', 'Desfacer cambio da IA'))
    const logoAsistente = el('span', { class: 'tarefa-asistente__logo' })
    const btnAsistente = opcAsist ? el('button', { type: 'button', class: 'tarefa-asistente', onclick: () => alternarAsistente(), title: t('formularios.dis.asistente.title', 'Asistente de IA: deseña o formulario e as tubaxes desde unha descrición ou un documento xa feito') },
      logoAsistente, el('span', { text: t('formularios.dis.asistente.label', 'Asistente deseño') })) : null
    const cabeceira = el('div', { class: 'dis-cab' },
      el('div', { class: 'dis-cab__fila' },
        el('div', { class: 'dis-cab__nome' }, inNome, inDescricion),
        el('button', { type: 'button', class: 'dis-x', 'aria-label': t('dialog.cancel', 'Cancelar'), title: t('dialog.cancel', 'Cancelar'), html: ICONA_X, onclick: () => pedirPechar() })),
      el('div', { class: 'dis-cab__fila dis-cab__fila--opcions' },
        el('div', { class: 'dis-opcions' },
          marca('portada', t('formularios.dis.portada', 'Con portada'), t('formularios.dis.portada.title', 'Engade a portada da plantilla ao documento')),
          marca('sinatura', t('formularios.dis.sinatura', 'Con sinatura'), t('formularios.dis.sinatura.title', 'Engade ao final o lugar, a data e a sinatura de quen asina')))),
      el('div', { class: 'dis-cab__fila dis-cab__fila--tabs' },
        el('div', { class: 'dis-tabs', role: 'tablist' }, tabCampos, tabFluxo),
        el('span', { class: 'dis-cab__espazo' }), btnDesfacerIA, ...(btnAsistente ? [btnAsistente] : [])))

    // ---------- Paleta ----------
    paleta.append(el('h3', { class: 'dis-tit', text: t('formularios.dis.campos', 'Campos') }),
      el('p', { class: 'dis-axuda', text: t('formularios.dis.campos.axuda', 'Arrastra un campo ao formulario, ou preme nel para engadilo ao final.') }))
    for (const tipo of TIPOS) {
      const b = el('button', { type: 'button', class: 'dis-tipo', 'data-tipo': tipo, html: `${svgTipo(tipo)}<span>${nome(tipo)}</span>` })
      b.addEventListener('click', () => engadir(tipo, est.campos.length))
      b.addEventListener('pointerdown', (ev) => iniciarArrastre(ev, {
        texto: nome(tipo),
        aoMover: (x, y) => marcarDestino(x, y),
        aoSoltar: (x, y) => { const i = indiceDestino(x, y); if (i >= 0) engadir(tipo, i) },
        aoRematar: () => limparMarcas(),
      }))
      paleta.append(b)
    }

    // ---------- Lenzo ----------
    function indiceDestino(x, y) {
      const r = lenzo.getBoundingClientRect()
      if (x < r.left || x > r.right || y < r.top - 20 || y > r.bottom + 20) return -1
      const tarxetas = [...lenzo.querySelectorAll('.dis-campo')]
      for (let i = 0; i < tarxetas.length; i++) {
        const b = tarxetas[i].getBoundingClientRect()
        if (y < b.top + b.height / 2) return i
      }
      return tarxetas.length
    }
    function limparMarcas() {
      lenzo.classList.remove('dis-lenzo--sobre')
      lenzo.querySelectorAll('.dis-campo--antes, .dis-campo--despois').forEach((n) => n.classList.remove('dis-campo--antes', 'dis-campo--despois'))
    }
    function marcarDestino(x, y) {
      limparMarcas()
      const i = indiceDestino(x, y)
      if (i < 0) return
      lenzo.classList.add('dis-lenzo--sobre')
      const tarxetas = [...lenzo.querySelectorAll('.dis-campo')]
      if (i < tarxetas.length) tarxetas[i].classList.add('dis-campo--antes')
      else if (tarxetas.length) tarxetas[tarxetas.length - 1].classList.add('dis-campo--despois')
    }

    function resumoCampo(c) {
      const partes = [nome(c.tipo)]
      if (c.obrigatorio) partes.push(t('formularios.obrigatorio', 'Obrigatorio'))
      if (c.agochado) partes.push(t('formularios.dis.agochado', 'Agochado'))
      let detalle = ''
      if (c.tipo === 'escolla') detalle = (c.opcions || []).join(' · ')
      if (c.tipo === 'taboa') detalle = (c.columnas || []).map((x) => x.etiqueta).join(' | ')
      return { linha: partes.join(' · '), detalle }
    }

    function pintarLenzo() {
      lenzo.replaceChildren()
      if (!est.campos.length) {
        lenzo.append(el('div', { class: 'dis-lenzo__baleiro' },
          el('p', { text: t('formularios.dis.baleiro', 'Aínda non hai campos.') }),
          el('p', { text: t('formularios.dis.baleiro.axuda', 'Arrastra un tipo de campo desde a esquerda ata aquí.') })))
        return
      }
      est.campos.forEach((c, i) => {
        const { linha, detalle } = resumoCampo(c)
        const acc = (icona, titulo, fn, desactivado) => el('button', {
          type: 'button', class: 'form-icona-btn', title: titulo, 'aria-label': titulo, html: icona, disabled: desactivado || undefined,
          onclick: (e) => { e.stopPropagation(); fn() },
        })
        const asa = el('span', { class: 'dis-campo__asa', title: t('formularios.dis.arrastrar', 'Arrastrar para reordenar'), html: ASA })
        asa.addEventListener('pointerdown', (ev) => {
          sel = i
          iniciarArrastre(ev, {
            texto: c.etiqueta || nome(c.tipo),
            aoMover: (x, y) => marcarDestino(x, y),
            aoSoltar: (x, y) => {
              const d = indiceDestino(x, y)
              if (d < 0) return
              est.campos = moverCampo(est.campos, i, d)
              sel = est.campos.findIndex((k) => k === c)
              marcar(); pintar()
            },
            aoRematar: () => limparMarcas(),
          })
        })
        const tarxeta = el('div', {
          class: 'dis-campo' + (i === sel ? ' dis-campo--sel' : '') + (c.tipo === 'seccion' ? ' dis-campo--seccion' : ''), role: 'listitem', tabindex: '0',
          'aria-label': `${c.etiqueta} (${nome(c.tipo)})`,
          onclick: () => { sel = i; pintar() },
          onkeydown: (e) => {
            if (e.target !== tarxeta) return
            if (e.key === 'Delete') { e.preventDefault(); quitar(i) }
            if (e.key === 'ArrowUp' && e.altKey) { e.preventDefault(); subir(i, -1) }
            if (e.key === 'ArrowDown' && e.altKey) { e.preventDefault(); subir(i, 1) }
          },
        },
        asa,
        el('span', { class: 'dis-campo__icona', html: svgTipo(c.tipo) }),
        el('div', { class: 'dis-campo__corpo' },
          el('strong', { text: c.etiqueta || t('formularios.dis.semEtiqueta', '(sen etiqueta)') }),
          el('small', { text: linha }),
          detalle ? el('small', { class: 'dis-campo__detalle', text: detalle }) : null),
        el('div', { class: 'dis-campo__acc' },
          acc(ICONA_SUBIR, t('formularios.dis.subir', 'Subir'), () => subir(i, -1), i === 0),
          acc(ICONA_BAIXAR, t('formularios.dis.baixar', 'Baixar'), () => subir(i, 1), i === est.campos.length - 1),
          acc(ICONA_COPIAR, t('formularios.dis.duplicar', 'Duplicar'), () => { est.campos = duplicarCampo(est.campos, i); sel = i + 1; marcar(); pintar() }),
          acc(ICONA_X, t('formularios.dis.quitar', 'Quitar campo'), () => quitar(i))))
        lenzo.append(tarxeta)
      })
    }

    function engadir(tipo, indice) {
      est.campos = inserirCampo(est.campos, novoCampo(tipo, nome(tipo)), indice)
      sel = Math.min(indice, est.campos.length - 1)
      marcar(); mostrarErro(''); pintar()
      lenzo.querySelector('.dis-campo--sel')?.scrollIntoView?.({ block: 'nearest' })
    }
    function quitar(i) {
      est.campos = eliminarCampo(est.campos, i)
      sel = est.campos.length ? Math.min(i, est.campos.length - 1) : -1
      marcar(); pintar()
    }
    function subir(i, d) {
      const j = i + d
      if (j < 0 || j >= est.campos.length) return
      est.campos = moverCampo(est.campos, i, d > 0 ? j + 1 : j)
      sel = j
      marcar(); pintar()
    }

    // ---------- Propiedades ----------
    function pintarPropiedades() {
      props.replaceChildren()
      props.append(el('h3', { class: 'dis-tit', text: t('formularios.dis.propiedades', 'Propiedades') }))
      const c = est.campos[sel]
      if (!c) {
        props.append(el('p', { class: 'dis-axuda', text: t('formularios.dis.propiedades.axuda', 'Escolle un campo para cambiar o seu nome, axuda ou opcións.') }))
        return
      }
      const cambiar = (cambios, repintar = false) => {
        est.campos = cambiarCampo(est.campos, sel, cambios)
        marcar(); mostrarErro('')
        if (repintar) pintar(); else pintarLenzo()
      }
      const texto = (etiqueta, valor, fn, extra = {}) => {
        const i = el('input', { type: 'text', class: 'form-entrada', ...extra })
        i.value = valor || ''
        i.addEventListener('input', () => fn(i.value))
        return el('label', { class: 'dis-prop' }, el('span', { text: etiqueta }), i)
      }
      const marcaProp = (etiqueta, valor, fn, axuda) => {
        const i = el('input', { type: 'checkbox', class: 'form-marca' })
        i.checked = !!valor
        i.addEventListener('change', () => fn(i.checked))
        return el('label', { class: 'dis-opcion', title: axuda }, i, el('span', { text: etiqueta }))
      }
      props.append(el('p', { class: 'dis-props__tipo', html: `${svgTipo(c.tipo)}<span>${nome(c.tipo)}</span>` }))
      props.append(texto(c.tipo === 'seccion' ? t('formularios.dis.titulo.seccion', 'Título da sección') : t('formularios.dis.etiqueta', 'Etiqueta'), c.etiqueta, (v) => cambiar({ etiqueta: v }), { maxlength: '120' }))
      if (c.tipo !== 'seccion') {
        props.append(texto(t('formularios.dis.axuda', 'Axuda (opcional)'), c.axuda, (v) => cambiar({ axuda: v }), { maxlength: '200' }))
        props.append(marcaProp(t('formularios.dis.obrigatorio', 'Obrigatorio'), c.obrigatorio, (v) => cambiar({ obrigatorio: v }), t('formularios.dis.obrigatorio.title', 'Se falta, Lúa avisa ao xerar (pero xera o documento igual)')))
        props.append(marcaProp(t('formularios.dis.agochado.opcion', 'Non sacar no documento'), c.agochado, (v) => cambiar({ agochado: v }), t('formularios.dis.agochado.title', 'O dato só se usa na plantilla (cabeceira, pé…), non no corpo')))
      }
      if (c.tipo === 'escolla') {
        const a = el('textarea', { class: 'form-entrada', rows: '5', 'aria-label': t('formularios.dis.opcions', 'Opcións (unha por liña)') })
        a.value = (c.opcions || []).join('\n')
        a.addEventListener('input', () => cambiar({ opcions: a.value.split('\n') }))
        props.append(el('label', { class: 'dis-prop' }, el('span', { text: t('formularios.dis.opcions', 'Opcións (unha por liña)') }), a))
      }
      if (c.tipo === 'taboa') props.append(editorColumnas(c, cambiar))
      if (c.id) {
        props.append(el('p', { class: 'dis-axuda dis-props__id' }, t('formularios.dis.variable', 'Variable da plantilla: '), el('code', { text: '{{CAMPO_' + c.id.toUpperCase() + '}}' })))
      }
    }

    function editorColumnas(c, cambiar) {
      const caixa = el('div', { class: 'dis-cols' }, el('span', { class: 'dis-cols__tit', text: t('formularios.dis.columnas', 'Columnas da táboa') }))
      const columnas = c.columnas || []
      // As columnas lense do estado NO MOMENTO de cada cambio: `columnas` é a
      // lista de cando se pintou o panel e, se non se repinta (tecleo), a
      // segunda edición pisaría a primeira.
      const actuais = () => est.campos[sel]?.columnas || []
      const gardar = (novas, repintar = true) => cambiar({ columnas: novas }, repintar)
      columnas.forEach((col, k) => {
        const i = el('input', { type: 'text', class: 'form-entrada form-entrada--compacta', maxlength: '80', 'aria-label': t('formularios.dis.columna', 'Nome da columna') })
        i.value = col.etiqueta || ''
        i.addEventListener('input', () => gardar(actuais().map((x, j) => (j === k ? { ...x, etiqueta: i.value } : x)), false))
        const s = el('select', { class: 'form-entrada form-entrada--compacta', 'aria-label': t('formularios.dis.columna.tipo', 'Tipo da columna') })
        for (const tp of TIPOS_COLUMNA) s.append(new Option(nome(tp), tp))
        s.value = col.tipo
        s.addEventListener('change', () => {
          const novo = { ...actuais()[k], tipo: s.value }
          if (s.value === 'escolla' && !(novo.opcions || []).length) novo.opcions = ['Opción 1', 'Opción 2']
          gardar(actuais().map((x, j) => (j === k ? novo : x)))
        })
        const fila = el('div', { class: 'dis-cols__fila' }, i, s, el('button', {
          type: 'button', class: 'form-icona-btn', title: t('formularios.dis.columna.quitar', 'Quitar columna'), 'aria-label': t('formularios.dis.columna.quitar', 'Quitar columna'),
          html: ICONA_X, disabled: columnas.length <= 1 || undefined, onclick: () => gardar(actuais().filter((_, j) => j !== k)),
        }))
        caixa.append(fila)
        if (col.tipo === 'escolla') {
          const o = el('input', { type: 'text', class: 'form-entrada form-entrada--compacta dis-cols__ops', 'aria-label': t('formularios.dis.columna.opcions', 'Opcións, separadas por punto e coma'), placeholder: t('formularios.dis.columna.opcions', 'Opcións, separadas por punto e coma') })
          o.value = (col.opcions || []).join('; ')
          o.addEventListener('input', () => gardar(actuais().map((x, j) => (j === k ? { ...x, opcions: o.value.split(';') } : x)), false))
          caixa.append(o)
        }
      })
      if (columnas.length < 12) {
        caixa.append(el('button', {
          type: 'button', class: 'crono-btn crono-btn--pequeno', onclick: () => gardar([...actuais(), novoCampo('texto', t('formularios.dis.columna.nova', 'Columna {n}', { n: actuais().length + 1 }))]),
        }, t('formularios.dis.columna.engadir', '+ Engadir columna')))
      }
      return caixa
    }

    function pintar() { pintarLenzo(); pintarPropiedades() }

    // ---------- Pé ----------
    const btnGardar = el('button', { type: 'button', class: 'dis-gardar', onclick: () => gardar() }, t('formularios.dis.gardar', 'Gardar formulario'))
    const btnEliminar = existente ? el('button', { type: 'button', class: 'dis-eliminar', onclick: () => eliminar() }, t('formularios.dis.eliminar', 'Eliminar formulario')) : null
    const pe = el('div', { class: 'dis-pe' }, btnEliminar, el('span', { class: 'dis-pe__espazo' }),
      el('button', { type: 'button', onclick: () => pedirPechar() }, t('dialog.cancel', 'Cancelar')), btnGardar)

    const vistaCampos = el('div', { class: 'dis-corpo' }, paleta, lenzo, props)
    const vistaFluxo = el('div', { class: 'dis-fluxo', hidden: true })
    const zonaAsist = el('div', { class: 'dis-asist' })
    caixa.append(cabeceira, erro, el('div', { class: 'dis-medio' }, el('div', { class: 'dis-central' }, vistaCampos, vistaFluxo), zonaAsist), pe)

    // ---------- Tuberías ----------
    async function cambiarPestana(nova) {
      if (nova === pestana) return
      pestana = nova
      for (const [b, id] of [[tabCampos, 'campos'], [tabFluxo, 'fluxo']]) {
        b.classList.toggle('dis-tab--sel', id === nova)
        b.setAttribute('aria-selected', String(id === nova))
      }
      vistaCampos.hidden = nova !== 'campos'
      vistaFluxo.hidden = nova !== 'fluxo'
      if (nova === 'fluxo') await mostrarFluxo()
      else { fluxoEd?.destroy(); fluxoEd = null }
    }

    async function mostrarFluxo() {
      try {
        est.campos = completarIds(est.campos)
        if (!catalogo) catalogo = await api.CatalogoFluxo()
        if (est.fluxo) {
          // Os campos puideron cambiar desde a última vez: axústase a tubería.
          const sinc = await api.SincronizarFluxoFormulario(limpar(est))
          if (!fluxoIgual(sinc, est.fluxo)) { est.fluxo = sinc; marcar() }
          montarEditorFluxo()
        } else {
          await mostrarFluxoAutomatico()
        }
      } catch (err) {
        mostrarErro(String(err?.message || err))
      }
    }

    // montarEditorFluxo pinta o deseño das tubaxes. Con `previa` (a tubaxe do
    // cableado automático), o formulario segue en automático: amósase o deseño
    // pero non se garda ata o primeiro cambio, que o personaliza.
    function montarEditorFluxo(previa) {
      fluxoEd?.destroy()
      fluxoEd = montarFluxoEditor(vistaFluxo, {
        t, api, catalogo,
        obterEsquema: () => (est.fluxo || !previa ? est : { ...est, fluxo: previa }),
        personalizado: () => !!est.fluxo,
        obterDatos: () => obterDatos() || {},
        aoCambiar: (f) => { est.fluxo = f; marcar(); mostrarErro('') },
        aoAutomatico: async () => {
          if (confirmar && !await confirmar(t('formularios.flx.automatico.confirmar', 'Vanse descartar as tubaxes que fixeches e o documento volverá ao cableado automático. Continuar?'))) return
          fluxoEd?.destroy()
          fluxoEd = null
          delete est.fluxo
          marcar()
          await mostrarFluxoAutomatico()
        },
      })
    }

    // mostrarFluxoAutomatico amosa directamente o deseño que corresponde ao
    // cableado automático (sen texto explicativo nin botón).
    async function mostrarFluxoAutomatico() {
      try {
        est.campos = completarIds(est.campos)
        montarEditorFluxo(await api.FluxoAutomaticoDe(limpar(est)))
      } catch (err) { mostrarErro(String(err?.message || err)) }
    }

    // ---------- Asistente de IA ----------
    // O borrador que ve e cambia a IA: sempre con ids completos.
    const editorParaIA = {
      borrador() {
        est.campos = completarIds(est.campos)
        return { ...limpar(est), ...(existente ? { id: est.id } : {}) }
      },
      cambiar(novo) {
        pilaIA.push(JSON.parse(JSON.stringify(est)))
        btnDesfacerIA.hidden = false
        aplicarBorrador(novo)
      },
      datos: () => obterDatos() || {},
    }

    // aplicarBorrador substitúe o formulario polo que deu a IA e repinta todo.
    function aplicarBorrador(novo) {
      const id = est.id // un formulario novo non ten id ata que se garda
      est = JSON.parse(JSON.stringify(novo))
      est.id = id
      est.campos = est.campos || []
      inNome.value = est.nome || ''
      inDescricion.value = est.descricion || ''
      for (const k of ['portada', 'sinatura']) if (marcas[k]) marcas[k].checked = !!est[k]
      sel = est.campos.length ? Math.min(Math.max(sel, 0), est.campos.length - 1) : -1
      marcar()
      mostrarErro('')
      pintar()
      if (pestana === 'fluxo') {
        if (est.fluxo) { if (fluxoEd) fluxoEd.setFluxo(est.fluxo); else montarEditorFluxo() } else { fluxoEd?.destroy(); fluxoEd = null; mostrarFluxoAutomatico() }
      }
    }

    function desfacerIA() {
      const antes = pilaIA.pop()
      if (!antes) return
      aplicarBorrador(antes)
      btnDesfacerIA.hidden = pilaIA.length === 0
    }

    function resumoParaIA() {
      const b = est
      const nodos = b.fluxo ? b.fluxo.nodos.length : 0
      const datos = obterDatos() || {}
      return [
        `- Formulario «${b.nome || '(sen nome)'}» ${existente ? `(gardado, id ${est.id})` : '(novo, aínda sen gardar)'}; ${b.campos.length} campo(s); portada: ${b.portada ? 'si' : 'non'}; sinatura: ${b.sinatura ? 'si' : 'non'}.`,
        `- Tubaxe: ${b.fluxo ? `propia (${nodos} nodos, ${b.fluxo.documento.length} bloques no documento)` : 'automática (sen tubaxe propia)'}.`,
        `- ${sucio ? 'Hai cambios sen gardar no deseñador.' : 'Sen cambios pendentes.'} Datos do documento aberto: ${Object.keys(datos.valores || {}).length ? 'si (formulario_ler ensínaos)' : 'non hai'}.`,
      ].join('\n')
    }

    function montarAsistente() {
      const opc = opcAsist
      const ferr = crearFerramentasFormulario({
        editor: editorParaIA,
        api: { IAChamarFerramenta: opc.api.IAChamarFerramenta, FluxoAutomaticoDe: api.FluxoAutomaticoDe },
        ui: { avisar: (m) => mostrarErro(m), preguntar: (p, o) => asistente.preguntar(p, o) },
      })
      asistente = crearPanelAsistente({
        t,
        api: {
          ...opc.api, // inclúe os adxuntos (clip, arrastrar, pegar)
          IAFerramentas: () => opc.api.IAFerramentasDoAmbito('formulario'),
          IATurnoAsistente: (req) => opc.api.IATurnoAsistente({ ...req, ambito: 'formulario', idioma: opc.idioma?.() || '' }),
        },
        documentoBoveda: 'formulario',
        executar: ferr.executar,
        estadoPrograma: resumoParaIA,
        tarefas: TAREFAS_FORMULARIO,
        claveTarefas: 'lua.asistente.tarefas.formulario',
        montarEn: zonaAsist,
        textos: {
          titulo: t('asistente.tf.titulo', 'Asistente deseño'),
          benvida: t('asistente.tf.benvida', 'Deseño formularios e tubaxes. Xunta cun clip un documento que xa teñas feito (un informe, unha acta, unha folla de notas…) e deduzo del os campos, as táboas e os cálculos; ou descríbeme o que necesitas. Os cambios entran no borrador do deseñador e ti decides se os gardas.'),
          benvidaNota: t('asistente.tf.nota', 'Podes desfacer o último cambio co botón «Desfacer cambio da IA». Se o documento ten cálculos, mírao ben: proba sempre co que che dea o asistente.'),
          caixaPh: t('asistente.tf.ph', 'Describe o formulario que queres, ou pídeme cambios: «engade unha táboa de alumnado», «calcula a nota media»… Se premes unha ficha, isto engádese como instrución adicional.'),
        },
        aoCambiar: ({ ocupado: ocup, preset }) => {
          logoAsistente.innerHTML = logoProvedor(preset)
          btnAsistente?.classList.toggle('tarefa-asistente--ocupado', ocup)
        },
      })
      asistente.pechar()
    }

    function alternarAsistente() {
      if (!asistente) return
      asistente.alternar()
      caixa.classList.toggle('dis--con-asist', asistente.estaAberto())
    }
    if (opcAsist) montarAsistente()

    // ---------- Accións ----------
    const pechar = (r) => {
      document.removeEventListener('keydown', teclas, true)
      fluxoEd?.destroy()
      asistente?.destruir()
      ;(alvo ? caixa : overlay).remove()
      resolve(r)
    }
    async function pedirPechar() {
      if (gardando) return
      if (sucio && confirmar && !await confirmar(t('formularios.dis.descartar', 'Hai cambios sen gardar no formulario. Descartalos?'))) return
      pechar(null)
    }
    async function gardar() {
      if (gardando) return
      const problema = problemasEsquema(est)
      if (problema) {
        const msgs = {
          nome: t('formularios.dis.erro.nome', 'Ponlle un nome ao formulario.'),
          etiqueta: t('formularios.dis.erro.etiqueta', 'Hai un campo (ou columna) sen etiqueta.'),
          opcions: t('formularios.dis.erro.opcions', 'Unha escolla ten que ter polo menos unha opción.'),
          columnas: t('formularios.dis.erro.columnas', 'Unha táboa ten que ter polo menos unha columna.'),
        }
        mostrarErro(msgs[problema])
        if (problema === 'nome') inNome.focus()
        return
      }
      gardando = true
      btnGardar.disabled = true
      try {
        est.campos = completarIds(est.campos)
        // Se ten tubería propia, axústase aos campos (os que se engadiron ou quitaron desde que se editou).
        if (est.fluxo) est.fluxo = await api.SincronizarFluxoFormulario(limpar(est))
        const gardado = await api.GardarFormulario(limpar(est))
        pechar({ accion: 'gardado', esquema: gardado, novo: !existente })
      } catch (err) {
        mostrarErro(String(err?.message || err))
      } finally {
        gardando = false
        btnGardar.disabled = false
      }
    }
    async function eliminar() {
      if (gardando) return
      if (confirmar && !await confirmar(t('formularios.dis.eliminar.confirmar', 'Vaise eliminar o formulario «{nome}». Os documentos que gardaches noutros ficheiros non se tocan. Continuar?', { nome: est.nome }))) return
      try {
        await api.EliminarFormulario(est.id)
        pechar({ accion: 'eliminado', id: est.id })
      } catch (err) {
        mostrarErro(String(err?.message || err))
      }
    }
    function teclas(e) {
      // Dentro do asistente, Esc pecha só o asistente (ver panel.js).
      if (e.target.closest?.('.asistente')) return
      if (e.key === 'Escape' && !document.querySelector('.overlay--dialog')) { e.stopPropagation(); pedirPechar() }
    }
    document.addEventListener('keydown', teclas, true)

    if (alvo) alvo.append(caixa)
    else document.body.append(overlay)
    aoPreparado?.({ pedirPechar })
    pintar()
    ;(existente ? lenzo.querySelector('.dis-campo') : inNome)?.focus()
  })
}

// limpar deixa o esquema listo para Go: sen opcións baleiras nin espazos.
function limpar(e) {
  const opcions = (l) => (l || []).map((o) => String(o).trim()).filter(Boolean)
  const campo = (c) => {
    const out = { ...c, etiqueta: String(c.etiqueta || '').trim() }
    if (c.tipo === 'escolla') out.opcions = opcions(c.opcions)
    else delete out.opcions
    if (c.tipo === 'taboa') out.columnas = (c.columnas || []).map(campo)
    else delete out.columnas
    return out
  }
  return { ...e, nome: String(e.nome || '').trim(), campos: e.campos.map(campo) }
}
