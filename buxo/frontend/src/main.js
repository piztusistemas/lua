import { Events } from '@wailsio/runtime'
import {
  Estado, Cola, Historial, Tarefas, LicenzaTexto, EscollerPDFs, EngadirURLs, Quitar, Construir, Parar, CambiarModo,
} from '../bindings/buxo/app.js'
import { textoEstado, textoAxudaURLs, resumoProgreso, fraccionTarefa } from './util.js'

const $ = (id) => document.getElementById(id)
const el = (tag, props = {}, ...fillos) => {
  const n = document.createElement(tag)
  for (const [k, v] of Object.entries(props)) {
    if (k === 'class') n.className = v
    else if (k === 'text') n.textContent = v
    else if (k.startsWith('on')) n.addEventListener(k.slice(2), v)
    else n.setAttribute(k, v)
  }
  n.append(...fillos)
  return n
}

let estado = null

function pintarEstado(e) {
  estado = e
  $('materia').textContent = e.materia || ''
  const ia = $('estado-ia')
  ia.textContent = e.iaConfigurada ? `IA: ${e.modelo}` : 'IA sen configurar'
  ia.className = 'chip ' + (e.iaConfigurada ? 'on' : 'off')
  const aviso = $('aviso')
  if (e.erro && !e.iaConfigurada) {
    aviso.textContent = e.erro
    aviso.hidden = false
  } else {
    aviso.hidden = true
  }
  $('btn-construir').hidden = e.construindo
  ia.classList.toggle('traballando', !!e.construindo)
  if (e.construindo) ia.textContent = `IA: ${e.modelo} · traballando…`
  $('btn-parar').hidden = !e.construindo
  $('btn-construir').disabled = !e.iaConfigurada || e.fontes === 0
  $('btn-construir').title = textoEstado(e)
  $('axuda-urls').textContent = textoAxudaURLs(e)
  const podeEngadir = !!e.boveda
  $('btn-pdfs').disabled = !podeEngadir
  $('btn-urls').disabled = !podeEngadir
}

const ETIQUETA = { pendente: 'Pendente', descargando: 'Descargando', convertendo: 'Convertendo', feito: 'Feito', erro: 'Erro' }

function pintarCola(items) {
  const cola = $('cola')
  cola.replaceChildren()
  const feitos = items.filter((i) => i.estado === 'feito').length
  $('conta').textContent = items.length ? `${feitos}/${items.length}` : ''
  if (!items.length) {
    cola.append(el('div', { class: 'baleiro', text: 'Aínda non hai ficheiros. Engade PDF, imaxes, Markdown ou URL á esquerda: converteranse a Markdown (coas fórmulas en LaTeX) automaticamente.' }))
    return
  }
  for (const it of items) {
    const nome = it.nome || it.fonte
    cola.append(el('div', { class: 'fila' },
      el('span', { class: 'est ' + it.estado, text: ETIQUETA[it.estado] || it.estado }),
      el('div', { class: 'fila-nome' },
        el('strong', { text: nome, title: it.fonte }),
        it.erro ? el('small', { class: 'erro', text: it.erro }) : el('small', { text: it.detalle || (it.orixe === 'url' ? it.fonte : '') }),
        // Leuno a IA (fórmulas, debuxos, escaneado): dise por que.
        it.vision && !it.erro ? el('small', { class: 'vision', text: `✦ Lido coa IA: ${it.vision}` }) : ''),
      el('button', { text: '✕', title: 'Quitar da lista', onclick: () => Quitar(it.id) })))
  }
}

const ETQ_TAREFA = { pendente: 'Pendente', executando: 'A IA traballa', feita: 'Feita', erro: 'Erro' }

// Progreso múltiple: unha barra total en segmentos (un por tarefa, coa cor do
// seu estado) e, debaixo, unha barra por tarefa co seu camiño.
function pintarTarefas(tarefas) {
  const sec = $('progreso')
  sec.hidden = !tarefas.length
  if (!tarefas.length) return
  const r = resumoProgreso(tarefas)
  $('prog-titulo').textContent = `Progreso · ${r.rematadas}/${r.total} tarefas` + (r.erros ? ` · ${r.erros} con erro` : '')
  $('prog-pct').textContent = `${r.pct} %`
  const seg = $('prog-segmentos')
  seg.setAttribute('aria-valuenow', String(r.pct))
  seg.replaceChildren(...tarefas.map((t) => el('span', { class: 'prog-seg ' + t.estado, title: `${t.nome} · ${ETQ_TAREFA[t.estado]}` })))
  const lista = $('prog-lista')
  const abaixo = lista.scrollHeight - lista.scrollTop - lista.clientHeight < 40
  lista.replaceChildren(...tarefas.map((t) => el('div', { class: 'prog-fila ' + t.estado },
    el('span', { class: 'prog-nome', text: t.nome, title: t.erro || t.nome }),
    el('div', { class: 'prog-barra' }, el('span', { class: 'prog-enche', style: `width:${Math.round(fraccionTarefa(t) * 100)}%` })),
    el('span', { class: 'prog-estado', text: t.estado === 'executando' ? `${ETQ_TAREFA[t.estado]} · ${t.pasos} pasos` : ETQ_TAREFA[t.estado] }))))
  // Segue a tarefa que corre, se non se estaba lendo máis arriba.
  const activa = lista.querySelector('.prog-fila.executando')
  if (activa && abaixo) activa.scrollIntoView({ block: 'nearest' })
}

const terminal = $('log-output')
function pintarLinha(l) {
  const cls = l.nivel === 'erro' ? 'erro-l' : l.nivel === 'aviso' ? 'aviso-l' : ''
  const div = el('div', { class: cls }, el('span', { class: 'hora', text: l.hora }), l.texto)
  const abaixo = terminal.scrollHeight - terminal.scrollTop - terminal.clientHeight < 40
  terminal.append(div)
  if (abaixo) terminal.scrollTop = terminal.scrollHeight
}

async function erroVisible(f) {
  try { return await f() } catch (e) {
    pintarLinha({ hora: new Date().toLocaleTimeString('gl'), nivel: 'erro', texto: e?.message || String(e) })
  }
}

// O modo de lectura lémbrase entre sesións.
const CLAVE_MODO = 'buxo.modo'
try { $('modo').value = localStorage.getItem(CLAVE_MODO) || 'auto' } catch { /* sen persistencia */ }
CambiarModo($('modo').value).catch(() => {})
$('modo').addEventListener('change', () => {
  try { localStorage.setItem(CLAVE_MODO, $('modo').value) } catch { /* sen persistencia */ }
  CambiarModo($('modo').value).catch(() => {})
})
$('btn-pdfs').addEventListener('click', () => erroVisible(EscollerPDFs))
$('btn-urls').addEventListener('click', () => erroVisible(async () => {
  await EngadirURLs($('urls').value, $('instrucion').value)
  $('urls').value = ''
}))
$('btn-construir').addEventListener('click', () => erroVisible(Construir))
$('btn-parar').addEventListener('click', () => Parar())
// Licenza de Buxo e avisos de terceiros.
async function amosarLicenza() {
  $('licenza-texto').textContent = await LicenzaTexto()
  $('licenza').hidden = false
  $('btn-licenza-pechar').focus()
}
$('btn-licenza').addEventListener('click', () => erroVisible(amosarLicenza))
$('btn-licenza-pechar').addEventListener('click', () => { $('licenza').hidden = true })
$('licenza').addEventListener('click', (e) => { if (e.target === $('licenza')) $('licenza').hidden = true })
document.addEventListener('keydown', (e) => { if (e.key === 'Escape') $('licenza').hidden = true })
$('btn-clear-log').addEventListener('click', () => terminal.replaceChildren())
$('btn-toggle-log').addEventListener('click', () => {
  const p = $('log-panel')
  p.classList.toggle('collapsed')
  $('btn-toggle-log').textContent = p.classList.contains('collapsed') ? '+' : '–'
})

Events.On('buxo:cola', (e) => pintarCola(e.data))
Events.On('buxo:rexistro', (e) => pintarLinha(e.data))
Events.On('buxo:estado', (e) => pintarEstado(e.data))
Events.On('buxo:tarefas', (e) => pintarTarefas(e.data || []))

;(async () => {
  pintarEstado(await Estado())
  pintarCola(await Cola())
  pintarTarefas(await Tarefas())
  for (const l of await Historial()) pintarLinha(l)
})()
