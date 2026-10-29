// Modelo do editor visual de fórmulas (funcións puras, sen DOM nin MathLive).
//
// O enunciado segue a ser Markdown con $…$ (constitución 3.3): o editor
// visual só le unha fórmula dese texto e devolve outra. Estas funcións
// atopan as fórmulas no texto e traducen as variables entre as dúas formas:
//
//   no enunciado    $F = {m}\cdot {a}$       (chaves con espazo: o valor)
//   no editor       F = \variable{m}\cdot \variable{a}
//
// \variable é unha macro do editor que debuxa a variable coma unha pastilla
// que non se pode partir. As chaves pegadas a un comando, a «^», a «_» ou a
// outras chaves son LaTeX ($\dfrac{m}{V}$) e quedan coma están.

// As mesmas fórmulas que compón a vista (mates-vista.js) e no mesmo orde.
const MATES = /\$\$([\s\S]+?)\$\$|\\\[([\s\S]+?)\\\]|\\\(([\s\S]+?)\\\)|\$([^\s$](?:[^$\n]*?[^\s$])?|\s[^$\n]*?[\\{}^_=][^$\n]*?|[^$\n]*?[\\{}^_=][^$\n]*?\s)\$(?!\d)/g
const DEBUXO = /```(tikz|circuitikz)[ \t]*\n([\s\S]*?)\n```/g
const VARIABLE = /\{\s*([A-Za-z_][A-Za-z0-9_]*)\s*\}/g
const grupoLatex = (c) => /[A-Za-z^_}]/.test(c || '')

// formulasDe devolve as fórmulas dun texto: [{ ini, fin, tex, centrada }],
// con ini/fin as posicións dos delimitadores (fin exclusivo). Os debuxos
// e os \$ escapados non contan.
export function formulasDe(md) {
  const texto = String(md ?? '')
  // Tapar sen mover posicións: os debuxos con espazos e \$ con dous «x».
  const tapado = texto
    .replace(DEBUXO, (m) => m.replace(/[^\n]/g, ' '))
    .replace(/\\\$/g, 'xx')
  const out = []
  for (const m of tapado.matchAll(MATES)) {
    const [, d1, d2, i1, i2] = m
    const centrada = (d1 ?? d2) !== undefined
    const ini = m.index
    const fin = ini + m[0].length
    const ancho = m[0].startsWith('$$') || m[0].startsWith('\\') ? 2 : 1
    out.push({ ini, fin, tex: texto.slice(ini + ancho, fin - ancho).trim(), centrada })
  }
  return out
}

// formulaEn devolve a fórmula que contén a posición do cursor (incluídos os
// delimitadores), ou null.
export function formulaEn(md, pos) {
  return formulasDe(md).find((f) => pos >= f.ini && pos <= f.fin) || null
}

// texAEditor pasa o LaTeX dunha fórmula do enunciado ao editor visual.
export function texAEditor(tex) {
  return String(tex ?? '').replace(VARIABLE, (m, nome, i, s) => (grupoLatex(s[i - 1]) ? m : `\\variable{${nome}}`))
}

// editorATex fai o contrario: cada \variable{x} volve ser « {x} », co espazo
// diante que a distingue dun grupo de LaTeX.
export function editorATex(latex) {
  return String(latex ?? '')
    .replace(/\\variable\s*\{\s*([A-Za-z_][A-Za-z0-9_]*)\s*\}/g, ' {$1} ')
    .replace(/\{ +\{/g, '{ {')
    .replace(/\} +\}/g, '} }')
    .replace(/ {2,}/g, ' ')
    .trim()
}

// envolver pon os delimitadores. Unha fórmula que empeza ou remata nunha
// variable leva espazo xunto ao $ ($ {m} $), coma no resto de Lúa.
export function envolver(tex, centrada) {
  const t = String(tex ?? '').trim()
  if (!t) return ''
  if (centrada) return `$$${t}$$`
  const ini = /^\{\s*[A-Za-z_]\w*\s*\}/.test(t) ? ' ' : ''
  const ult = /\{\s*[A-Za-z_]\w*\s*\}$/.exec(t)
  const fin = ult && !grupoLatex(t[ult.index - 1]) ? ' ' : ''
  return `$${ini}${t}${fin}$`
}

// substituir mete a fórmula nova no texto: no lugar da que se editaba
// (f = { ini, fin }) ou no cursor. Unha fórmula centrada vai no seu propio
// parágrafo. Devolve { texto, cursor } co cursor xusto despois da fórmula.
export function substituir(md, nova, { ini, fin, centrada = false }) {
  const texto = String(md ?? '')
  let antes = texto.slice(0, ini)
  let despois = texto.slice(fin)
  let bloque = nova
  if (centrada && nova) {
    if (antes && !antes.endsWith('\n\n')) antes = antes.replace(/[ \t]*\n?$/, '') + '\n\n'
    if (despois && !despois.startsWith('\n\n')) despois = '\n\n' + despois.replace(/^[ \t]*\n?/, '')
  } else if (nova) {
    if (antes && !/\s$/.test(antes)) bloque = ' ' + bloque
    if (despois && !/^[\s.,;:!?)]/.test(despois)) bloque += ' '
  }
  return { texto: antes + bloque + despois, cursor: antes.length + bloque.length }
}

// variablesDoEditor devolve os nomes das \variable{…} dunha fórmula.
export function variablesDoEditor(latex) {
  return [...new Set([...String(latex ?? '').matchAll(/\\variable\s*\{\s*([A-Za-z_][A-Za-z0-9_]*)\s*\}/g)].map((m) => m[1]))]
}

export const nomeValido = (s) => /^[A-Za-z_][A-Za-z0-9_]*$/.test(String(s ?? '').trim())
