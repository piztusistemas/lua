// htmlMates converte o Markdown dunha pregunta ou dunha solución (o que xa
// calculou Go: as variables substituídas) en HTML coas fórmulas compostas
// con KaTeX, para velas no formulario mentres se escribe. É só a vista: o
// documento faino LaTeX de verdade.
//
// Markdown mínimo, o mesmo subconxunto que entende Lua (informe_md.go):
// parágrafos, **negriña**, *cursiva*, listas con «-» ou «1.», e fórmulas
// $…$, $$…$$, \( … \) e \[ … \].
import katex from 'katex'
import 'katex/dist/katex.min.css'

const escapar = (s) => String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')

const MATES = /\$\$([\s\S]+?)\$\$|\\\[([\s\S]+?)\\\]|\\\(([\s\S]+?)\\\)|\$([^\s$](?:[^$\n]*?[^\s$])?|\s[^$\n]*?[\\{}^_=][^$\n]*?|[^$\n]*?[\\{}^_=][^$\n]*?\s)\$(?!\d)/g
// (Con espazos nas beiras, $ {m} $, só se leva algo de LaTeX: «5$ ou 6$» é texto, coma en Go.)

function formula(tex, display) {
  try {
    return katex.renderToString(tex, { displayMode: display, throwOnError: false, strict: 'ignore', output: 'html' })
  } catch (err) {
    return `<code class="mates-erro" title="${escapar(err?.message || err)}">${escapar(tex)}</code>`
  }
}

function enLiña(s) {
  return s
    .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
    .replace(/(^|[^*\w])\*(?!\s)(.+?)\*(?!\w)/g, '$1<em>$2</em>')
}

// Os debuxos (```circuitikz e ```tikz) compílaos Go con LaTeX
// (CalcularProblemas): `debuxos` leva cada un pola clave «tipo\ncódigo».
const DEBUXO = /```(tikz|circuitikz)[ \t]*\n([\s\S]*?)\n```/g

function debuxo(tipo, codigo, debuxos) {
  const d = debuxos?.[tipo + '\n' + codigo.trim()]
  // Ao seu tamaño real (o do documento); nunca máis ancho ca a columna.
  const ancho = d?.anchoPx > 0 ? ` style="width:${Math.round(d.anchoPx)}px"` : ''
  if (d?.imaxe) return `<div class="mates-debuxo"><img src="${escapar(d.imaxe)}"${ancho} alt="${tipo === 'circuitikz' ? 'Circuíto' : 'Debuxo'}"></div>`
  if (d?.erro) return `<div class="mates-debuxo mates-debuxo--erro" title="${escapar(d.erro)}">⚠ O debuxo ten un erro de TikZ: ${escapar(d.erro.split('\n')[0].slice(0, 160))}</div>`
  return '<div class="mates-debuxo mates-debuxo--pendente">Debuxando…</div>'
}

export function htmlMates(md, debuxos = null) {
  const formulas = []
  const marca = (html) => { formulas.push(html); return `\u0000${formulas.length - 1}\u0000` }
  let s = String(md ?? '').replace(DEBUXO, (_m, tipo, codigo) => `\n\n${marca(debuxo(tipo, codigo, debuxos))}\n\n`)
  s = s.replace(/\\\$/g, '\u0001')
  s = s.replace(MATES, (_m, d1, d2, i1, i2) => (d1 ?? d2) !== undefined ? marca(formula((d1 ?? d2).trim(), true)) : marca(formula((i1 ?? i2).trim(), false)))
  const html = []
  let lista = null
  const pecharLista = () => { if (lista) { html.push(`</${lista}>`); lista = null } }
  for (const bloque of escapar(s).split(/\n\s*\n/)) {
    for (const liña of bloque.split('\n')) {
      const t = liña.trim()
      const vi = /^[-*]\s+(.*)$/.exec(t)
      const nu = /^\d+[.)]\s+(.*)$/.exec(t)
      if (vi || nu) {
        const tipo = vi ? 'ul' : 'ol'
        if (lista !== tipo) { pecharLista(); html.push(`<${tipo}>`); lista = tipo }
        html.push(`<li>${enLiña((vi || nu)[1])}</li>`)
        continue
      }
      pecharLista()
      if (t) html.push(`<p>${enLiña(t)}</p>`)
    }
    pecharLista()
  }
  return html.join('').replace(/<\/p><p>/g, '<br>').replace(/\u0000(\d+)\u0000/g, (_m, i) => formulas[Number(i)]).replace(/\u0001/g, '$')
}
