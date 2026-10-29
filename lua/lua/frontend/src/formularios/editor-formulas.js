// Editor visual de fórmulas (ao estilo de MathType ou CalcMe) para os
// enunciados dos problemas. Escríbese a fórmula coma no papel, cunha paleta
// de modelos e atallos de teclado, e o resultado volve ao enunciado coma
// LaTeX ($…$): o texto segue a ser a fonte da verdade (constitución 3.3).
//
// O campo de escritura é MathLive (<math-field>, MIT), que se carga só a
// primeira vez que se abre o editor. As letras son as de KaTeX, que xa
// trae a vista (mates-vista.js): MathLive non descarga ningunha.
//
//   abrirEditorFormulas({ t, tex, centrada, variables })
//     → Promise<{ tex, centrada } | null>    (null se se cancela)
//
// tex vai e volve na forma do enunciado ({m} con espazo = variable).
import katex from 'katex'
import { el } from '../informes/dom.js'
import { texAEditor, editorATex, variablesDoEditor, nomeValido } from './formulas-modelo.js'

// A pastilla dunha variable: un bloque que non se parte, coa cor da app.
const MACRO_VARIABLE = { args: 1, def: '\\colorbox{#e0e7ff}{\\textcolor{#3730a3}{\\textsf{#1}}}', captureSelection: true, expand: false }

// Paleta: [título, LaTeX que se insire (#0 = o seleccionado, #? = un oco,
// #@ = o que hai antes), mostra no botón (se non, a propia inserción),
// false se o título non se traduce (letras gregas, unidades)].
export const PALETA = [
  ['basico', 'Básico', [
    ['Fracción', '\\frac{#0}{#?}', '\\dfrac{a}{b}'],
    ['Potencia', '#@^{#?}', 'x^{n}'],
    ['Subíndice', '#@_{#?}', 'x_{i}'],
    ['Raíz cadrada', '\\sqrt{#0}', '\\sqrt{x}'],
    ['Raíz n-ésima', '\\sqrt[#?]{#0}', '\\sqrt[n]{x}'],
    ['Parénteses', '\\left(#0\\right)', '(\\,)'],
    ['Corchetes', '\\left[#0\\right]', '[\\,]'],
    ['Valor absoluto', '\\left|#0\\right|', '|x|'],
    ['Por (punto)', '\\cdot'],
    ['Por (aspa)', '\\times'],
    ['Entre', '\\div'],
    ['Máis menos', '\\pm'],
    ['Distinto', '\\neq'],
    ['Aproximadamente', '\\approx'],
    ['Menor ou igual', '\\leq'],
    ['Maior ou igual', '\\geq'],
    ['Infinito', '\\infty'],
    ['Graos', '^{\\circ}', '30^{\\circ}'],
    ['Porcentaxe', '\\%'],
  ]],
  ['gregas', 'Gregas', [
    ...['alpha', 'beta', 'gamma', 'delta', 'varepsilon', 'theta', 'lambda', 'mu', 'pi', 'rho', 'sigma', 'tau', 'varphi', 'omega']
      .map((g) => [g, `\\${g}`, null, false]),
    ...['Delta', 'Sigma', 'Omega', 'Phi', 'Pi'].map((g) => [g, `\\${g}`, null, false]),
  ]],
  ['simbolos', 'Símbolos', [
    ['Frecha', '\\rightarrow'],
    ['Implica', '\\Rightarrow'],
    ['Equivale', '\\Leftrightarrow'],
    ['Proporcional', '\\propto'],
    ['Pertence', '\\in'],
    ['Non pertence', '\\notin'],
    ['Subconxunto', '\\subset'],
    ['Unión', '\\cup'],
    ['Intersección', '\\cap'],
    ['Para todo', '\\forall'],
    ['Existe', '\\exists'],
    ['Reais', '\\mathbb{R}'],
    ['Naturais', '\\mathbb{N}'],
    ['Enteiros', '\\mathbb{Z}'],
    ['Perpendicular', '\\perp'],
    ['Paralela', '\\parallel'],
    ['Ángulo', '\\angle'],
    ['Tres puntos', '\\ldots'],
  ]],
  ['calculo', 'Cálculo', [
    ['Sumatorio', '\\sum_{#?}^{#?}#0', '\\sum_{i=1}^{n}'],
    ['Integral', '\\int #0\\,d#?', '\\int f\\,dx'],
    ['Integral definida', '\\int_{#?}^{#?}#0\\,d#?', '\\int_a^b'],
    ['Límite', '\\lim_{#?\\to #?}#0', '\\lim_{x\\to a}'],
    ['Derivada', '\\frac{d#0}{d#?}', '\\dfrac{dy}{dx}'],
    ['Derivada parcial', '\\frac{\\partial #0}{\\partial #?}', '\\frac{\\partial f}{\\partial x}'],
    ['Logaritmo', '\\log_{#?}\\left(#0\\right)', '\\log_b'],
    ['Logaritmo neperiano', '\\ln\\left(#0\\right)', '\\ln'],
    ['Exponencial', 'e^{#0}', 'e^{x}'],
    ['Seno', '\\sin\\left(#0\\right)', '\\sin'],
    ['Coseno', '\\cos\\left(#0\\right)', '\\cos'],
    ['Tanxente', '\\tan\\left(#0\\right)', '\\tan'],
  ]],
  ['fisica', 'Vectores e unidades', [
    ['Vector', '\\vec{#0}', '\\vec{v}'],
    ['Incremento', '\\Delta #0', '\\Delta x'],
    ['Media (barra)', '\\overline{#0}', '\\overline{x}'],
    ['Unitario', '\\hat{#0}', '\\hat{u}'],
    ['Derivada temporal', '\\dot{#0}', '\\dot{x}'],
    ['Notación científica', '\\cdot 10^{#?}', '\\cdot 10^{n}'],
    ...['m', 'kg', 's', 'm/s', 'm/s^2', 'N', 'J', 'W', 'Pa', 'K', '^{\\circ}C', 'mol', 'A', 'V']
      .map((u) => [u.replace('^{\\circ}', '°'), `\\,\\mathrm{${u}}`, `\\mathrm{${u}}`, false]),
    ['Ohm', '\\,\\Omega', '\\Omega'],
  ]],
  ['matrices', 'Matrices', [
    ['Matriz 2×2', '\\begin{pmatrix}#?&#?\\\\#?&#?\\end{pmatrix}', '\\begin{pmatrix}a&b\\\\c&d\\end{pmatrix}'],
    ['Matriz 3×3', '\\begin{pmatrix}#?&#?&#?\\\\#?&#?&#?\\\\#?&#?&#?\\end{pmatrix}', '\\begin{pmatrix}\\cdot&\\cdot&\\cdot\\\\\\cdot&\\cdot&\\cdot\\\\\\cdot&\\cdot&\\cdot\\end{pmatrix}'],
    ['Determinante', '\\begin{vmatrix}#?&#?\\\\#?&#?\\end{vmatrix}', '\\begin{vmatrix}a&b\\\\c&d\\end{vmatrix}'],
    ['Vector columna', '\\begin{pmatrix}#?\\\\#?\\end{pmatrix}', '\\begin{pmatrix}x\\\\y\\end{pmatrix}'],
    ['Sistema', '\\begin{cases}#?\\\\#?\\end{cases}', '\\begin{cases}\\cdots\\\\\\cdots\\end{cases}'],
  ]],
]

// Atallos co Ctrl (⌘ en macOS), os de MathType. «/» (Maiús+7 no teclado
// galego e castelán) fai unha fracción co que hai antes, coma en MathLive.
const ATALLOS = {
  f: '\\frac{#0}{#?}',
  r: '\\sqrt{#0}',
  h: '#@^{#?}',
  l: '#@_{#?}',
}

let mathlive = null
async function cargarMathlive() {
  if (!mathlive) {
    mathlive = await import('mathlive')
    // Sen descargas: as letras son as de KaTeX (xa cargadas) e sen sons.
    mathlive.MathfieldElement.fontsDirectory = null
    mathlive.MathfieldElement.soundsDirectory = null
  }
  return mathlive
}

// chave: «Raíz cadrada» → «raiz-cadrada», para as claves de idiomas.
export const chave = (s) => s.normalize('NFD').replace(/[\u0300-\u036f]/g, '').toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')

const mostra = (tex) => {
  try { return katex.renderToString(tex, { throwOnError: false, strict: 'ignore', output: 'html' }) } catch { return tex }
}

export async function abrirEditorFormulas({ t, tex = '', centrada = false, variables = [] }) {
  const { MathfieldElement } = await cargarMathlive()
  const anterior = document.activeElement

  return new Promise((resolve) => {
    const mf = new MathfieldElement()
    mf.className = 'formula-campo'
    mf.setAttribute('aria-label', t('formulas.campo', 'Fórmula'))

    // ---------- Paleta ----------
    const pestanas = el('div', { class: 'formula-pestanas', role: 'tablist', 'aria-label': t('formulas.paleta', 'Paleta de símbolos') })
    const paineis = el('div', { class: 'formula-paineis' })
    PALETA.forEach(([id, nome, items], n) => {
      const panel = el('div', { class: 'formula-panel', role: 'tabpanel', id: 'formula-panel-' + id, hidden: n !== 0 })
      for (const [titulo, ins, ver, traducir = true] of items) {
        const tit = traducir ? t('formulas.p.' + chave(titulo), titulo) : titulo
        panel.append(el('button', {
          type: 'button', class: 'formula-btn', title: tit, 'aria-label': tit, html: mostra(ver ?? ins),
          onclick: () => inserir(ins),
        }))
      }
      const tab = el('button', {
        type: 'button', class: 'formula-pestana', role: 'tab', 'aria-selected': String(n === 0), 'aria-controls': panel.id,
        tabindex: n === 0 ? '0' : '-1', text: t(`formulas.tab.${id}`, nome),
        onclick: () => escollerPestana(n),
      })
      pestanas.append(tab)
      paineis.append(panel)
    })
    function escollerPestana(n) {
      ;[...pestanas.children].forEach((b, i) => { b.setAttribute('aria-selected', String(i === n)); b.tabIndex = i === n ? 0 : -1 })
      ;[...paineis.children].forEach((p, i) => { p.hidden = i !== n })
    }
    pestanas.addEventListener('keydown', (e) => {
      const tabs = [...pestanas.children]
      const i = tabs.indexOf(document.activeElement)
      if (i < 0 || !['ArrowLeft', 'ArrowRight'].includes(e.key)) return
      e.preventDefault()
      const j = (i + (e.key === 'ArrowRight' ? 1 : -1) + tabs.length) % tabs.length
      escollerPestana(j)
      tabs[j].focus()
    })

    // ---------- Variables ----------
    const nomeNova = el('input', {
      type: 'text', class: 'form-entrada formula-var__nova', spellcheck: 'false',
      placeholder: t('formulas.var.nova.ph', 'nome'), 'aria-label': t('formulas.var.nova', 'Nome da variable nova'),
    })
    const chips = el('span', { class: 'formula-vars__lista' })
    const coñecidas = new Set(variables.filter(nomeValido))
    const pintarChips = () => {
      for (const n of variablesDoEditor(mf.value)) coñecidas.add(n)
      chips.replaceChildren(...[...coñecidas].map((n) => el('button', {
        type: 'button', class: 'formula-var', title: t('formulas.var.inserir', 'Inserir a variable {v}', { v: n }), text: n,
        onclick: () => inserirVariable(n),
      })))
    }
    const novaVariable = () => {
      const n = nomeNova.value.trim()
      if (!nomeValido(n)) { nomeNova.setAttribute('aria-invalid', 'true'); nomeNova.focus(); return }
      nomeNova.removeAttribute('aria-invalid')
      nomeNova.value = ''
      inserirVariable(n)
    }
    nomeNova.addEventListener('keydown', (e) => { if (e.key === 'Enter') { e.preventDefault(); e.stopPropagation(); novaVariable() } })
    const varsFila = el('div', { class: 'formula-vars' },
      el('span', { class: 'formula-vars__tit', text: t('formulas.vars', 'Variables') }),
      chips,
      el('span', { class: 'formula-vars__nova' }, nomeNova,
        el('button', { type: 'button', class: 'crono-btn crono-btn--pequeno', text: t('formulas.var.engadir', '+ Variable'), title: t('formulas.var.engadir.title', 'Unha variable cambia de valor en cada copia do exame (Ctrl+K)'), onclick: novaVariable })))

    // ---------- LaTeX (para quen o prefira) ----------
    const fonte = el('textarea', { class: 'form-entrada formula-fonte', rows: '2', spellcheck: 'false', 'aria-label': t('formulas.latex', 'LaTeX') })
    const detFonte = el('details', { class: 'formula-latex' }, el('summary', { text: t('formulas.latex.ver', 'Ver e editar o LaTeX') }), fonte)
    const sincronizarFonte = () => { if (document.activeElement !== fonte) fonte.value = editorATex(mf.value) }
    fonte.addEventListener('input', () => { mf.value = texAEditor(fonte.value); pintarChips() })
    mf.addEventListener('input', () => { sincronizarFonte(); pintarChips() })

    const chkCentrada = el('input', { type: 'checkbox' })
    chkCentrada.checked = centrada

    const aceptar = el('button', { type: 'button', class: 'primary', text: t('formulas.aceptar', 'Inserir'), title: 'Ctrl+Enter', onclick: () => pechar({ tex: editorATex(mf.value), centrada: chkCentrada.checked }) })
    const modal = el('div', { class: 'modal formula-modal' },
      el('h2', { text: t('formulas.titulo', 'Editor de fórmulas') }),
      pestanas, paineis,
      el('div', { class: 'formula-marco' }, mf),
      varsFila,
      el('small', { class: 'form-camp__axuda formula-atallos', text: t('formulas.atallos', '/ (Maiús+7) fracción · ^ potencia · _ subíndice · Ctrl+R raíz · Ctrl+K variable · sqrt, pi, alpha… escritos convértense · Espazo ou → para saír dun oco') }),
      detFonte,
      el('div', { class: 'modal-actions' },
        el('label', { class: 'formula-centrada' }, chkCentrada, el('span', { text: t('formulas.centrada', 'Centrada na súa liña') })),
        el('span', { class: 'prob-espazo' }),
        el('button', { type: 'button', text: t('formulas.cancelar', 'Cancelar'), onclick: () => pechar(null) }),
        aceptar))
    const overlay = el('div', { class: 'overlay formula-overlay', role: 'dialog', 'aria-modal': 'true', 'aria-label': t('formulas.titulo', 'Editor de fórmulas') }, modal)

    function inserir(ins) {
      mf.executeCommand(['insert', ins, { selectionMode: 'placeholder', focus: true }])
      mf.focus()
    }
    function inserirVariable(n) {
      coñecidas.add(n)
      inserir(`\\variable{${n}}`)
      pintarChips()
    }

    // Atallos dentro do campo. En captura no propio <math-field>: chegan
    // antes ca os de MathLive (que van dentro da súa sombra).
    mf.addEventListener('keydown', (e) => {
      const mod = e.ctrlKey || e.metaKey
      let feito = true
      if (mod && e.key === 'Enter') aceptar.click()
      else if (mod && !e.shiftKey && !e.altKey && ATALLOS[e.key.toLowerCase()]) inserir(ATALLOS[e.key.toLowerCase()])
      else if (mod && e.key.toLowerCase() === 'k') nomeNova.focus()
      else if (!mod && !e.altKey && e.key === '/' && mf.mode === 'math') inserir('\\frac{#@}{#?}')
      else feito = false
      if (feito) { e.preventDefault(); e.stopPropagation() }
    }, true)

    let rematado = false
    function onKeydown(e) {
      if (e.key === 'Escape' && mf.mode !== 'latex') { e.preventDefault(); e.stopPropagation(); pechar(null) }
      else if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) { e.preventDefault(); e.stopPropagation(); aceptar.click() }
    }
    function pechar(res) {
      if (rematado) return
      rematado = true
      document.removeEventListener('keydown', onKeydown, true)
      overlay.remove()
      anterior?.focus?.()
      resolve(res)
    }
    document.addEventListener('keydown', onKeydown, true)
    overlay.addEventListener('mousedown', (e) => { if (e.target === overlay) pechar(null) })

    // MathLive só acepta opcións e valor xa montado no documento.
    document.body.append(overlay)
    mf.smartFence = true
    mf.mathVirtualKeyboardPolicy = 'manual'
    mf.macros = { ...mf.macros, variable: MACRO_VARIABLE }
    mf.value = texAEditor(tex)
    sincronizarFonte()
    pintarChips()
    requestAnimationFrame(() => { mf.focus(); mf.executeCommand('moveToMathfieldEnd') })
  })
}
