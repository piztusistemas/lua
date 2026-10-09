// Markdown das descricións das tarefas: (1) operacións da barra de formato
// sobre o texto dun <textarea> e (2) unha vista previa en HTML.
//
// O subconxunto é o de informe_md.go (parseMarkdownCrono), que é o que de
// verdade compón o PDF: títulos (#, ##, ###), listas (-, *, 1.), citas (>),
// **negrita**, *cursiva*, __negrita__, _cursiva_, [texto](url),
// ![alt](img:ID =60%), bloques de aliñamento ":::centro ... :::" e tamaño de
// letra [fs=peq|gr|xgr]...[/fs]. A vista previa é UNHA APROXIMACIÓN: o
// resultado final é sempre o que se ve na dereita despois de "Xerar". Se se
// cambian as regras de parseo hai que tocar os dous sitios (as probas de
// cronograma-md.test.js e informe_cronograma_test.go comparten casos).

// ---------- Operacións sobre o texto ----------
// Todas reciben e devolven {texto, ini, fin} (fin >= ini, posicións do
// textarea) e non tocan o DOM: o editor aplica o resultado.

export function envolver({ texto, ini, fin }, abre, pecha, marcador = 'texto') {
  const seleccion = texto.slice(ini, fin)
  const miolo = seleccion || marcador
  const novo = texto.slice(0, ini) + abre + miolo + pecha + texto.slice(fin)
  const a = ini + abre.length
  return { texto: novo, ini: a, fin: a + miolo.length }
}

function limitesLinas({ texto, ini, fin }) {
  const desde = texto.lastIndexOf('\n', ini - 1) + 1
  let ata = texto.indexOf('\n', fin > ini ? fin - 1 : fin)
  if (ata < 0) ata = texto.length
  return { desde, ata }
}

// prefixoBloque pon (ou quita, se todas as liñas xa o levan) un prefixo ao
// principio de cada liña seleccionada: "## ", "- ", "1. ", "> ".
export function prefixoBloque(estado, prefixo) {
  const { texto } = estado
  const { desde, ata } = limitesLinas(estado)
  const linas = texto.slice(desde, ata).split('\n')
  const todasCon = linas.every((l) => l.startsWith(prefixo))
  const novas = linas.map((l) => (todasCon ? l.slice(prefixo.length) : prefixo + l))
  const bloque = novas.join('\n')
  return { texto: texto.slice(0, desde) + bloque + texto.slice(ata), ini: desde, fin: desde + bloque.length }
}

const ALIN = ['esquerda', 'centro', 'dereita', 'xustificado']

// alinear envolve as liñas seleccionadas nun bloque ":::nome ... :::". Se xa
// están dentro dun bloque de aliñamento, cambia o nome en vez de aniñar.
export function alinear(estado, nome) {
  if (!ALIN.includes(nome)) return estado
  const { texto } = estado
  const { desde, ata } = limitesLinas(estado)
  const antes = texto.slice(0, desde)
  const cerca = /(^|\n):::(\w+)\n$/.exec(antes)
  const despois = texto.slice(ata)
  if (cerca && ALIN.includes(cerca[2]) && /^\n:::(\n|$)/.test(despois)) {
    const sustituido = antes.slice(0, cerca.index + cerca[1].length) + ':::' + nome + '\n'
    return { texto: sustituido + texto.slice(desde), ini: sustituido.length, fin: sustituido.length + (ata - desde) }
  }
  const abre = (desde > 0 && !antes.endsWith('\n\n') ? '\n' : '') + `:::${nome}\n`
  const pecha = '\n:::' + (despois.startsWith('\n\n') || despois === '' ? '' : '\n')
  const miolo = texto.slice(desde, ata)
  const novo = antes + abre + miolo + pecha + despois
  const a = antes.length + abre.length
  return { texto: novo, ini: a, fin: a + miolo.length }
}

export function tamanoLetra(estado, clave) {
  return envolver(estado, `[fs=${clave}]`, '[/fs]', 'texto')
}

export function ligazon(estado, url) {
  const { texto, ini, fin } = estado
  const etiqueta = texto.slice(ini, fin) || url
  const novo = texto.slice(0, ini) + `[${etiqueta}](${url})` + texto.slice(fin)
  return { texto: novo, ini: ini + 1, fin: ini + 1 + etiqueta.length }
}

// inserirImaxe pon `![alt](img:ref)` nunha liña propia (o parser só recoñece
// imaxes que ocupan a liña enteira).
export function inserirImaxe({ texto, ini, fin }, ref, alt = '') {
  const antes = texto.slice(0, ini)
  const despois = texto.slice(fin)
  const pre = antes === '' || antes.endsWith('\n') ? '' : '\n'
  const pos = despois === '' || despois.startsWith('\n') ? '' : '\n'
  const marca = `![${alt}](img:${ref})`
  const novo = antes + pre + marca + pos + despois
  const p = antes.length + pre.length + marca.length
  return { texto: novo, ini: p, fin: p }
}

// ---------- Parseo (porta de informe_md.go) ----------

const ESCALAS = { peq: 0.8, gr: 1.3, xgr: 1.6 }

const esAlfaNum = (c) => /[\p{L}\p{N}]/u.test(c || '')

// enxeAbre: ver a función homónima en informe_md.go.
function enxeAbre(runas, i, dobre) {
  const c = runas[i]
  if (c === '_' && i > 0 && i + 1 < runas.length && esAlfaNum(runas[i - 1]) && esAlfaNum(runas[i + 1])) return false
  const ini = dobre ? i + 2 : i + 1
  for (let j = ini; j < runas.length; j++) {
    if (runas[j] !== c) continue
    const parDobre = j + 1 < runas.length && runas[j + 1] === c
    if (dobre && parDobre) return true
    if (!dobre && !parDobre && !(j > 0 && runas[j - 1] === c)) return true
    if (parDobre) j++
  }
  return false
}

function parseLigazon(runas, inicio) {
  if (runas[inicio] !== '[') return null
  let i = inicio + 1
  let texto = ''
  while (i < runas.length && runas[i] !== ']') texto += runas[i++]
  if (runas[i] !== ']' || runas[i + 1] !== '(') return null
  i += 2
  let url = ''
  while (i < runas.length && runas[i] !== ')') url += runas[i++]
  if (runas[i] !== ')') return null
  return { texto, url, fin: i }
}

function parseTamanoLetra(runas, inicio) {
  if (runas.slice(inicio, inicio + 4).join('') !== '[fs=') return null
  let i = inicio + 4
  let clave = ''
  while (i < runas.length && runas[i] !== ']') clave += runas[i++]
  if (i >= runas.length || !ESCALAS[clave]) return null
  i++
  const pecha = '[/fs]'
  for (let j = i; j + pecha.length <= runas.length; j++) {
    if (runas.slice(j, j + pecha.length).join('') === pecha) {
      return { interior: runas.slice(i, j).join(''), escala: ESCALAS[clave], fin: j + pecha.length - 1 }
    }
  }
  return null
}

export function parseInline(s) {
  const tramos = []
  let buf = ''
  let negrita = false
  let cursiva = false
  const pechar = () => {
    if (buf) tramos.push({ texto: buf, negrita, cursiva })
    buf = ''
  }
  const runas = Array.from(s)
  for (let i = 0; i < runas.length; i++) {
    const c = runas[i]
    if (c === '[') {
      const tam = parseTamanoLetra(runas, i)
      if (tam) {
        pechar()
        for (const tr of parseInline(tam.interior)) {
          tramos.push({ ...tr, negrita: tr.negrita || negrita, cursiva: tr.cursiva || cursiva, escala: tam.escala })
        }
        i = tam.fin
        continue
      }
      const lig = parseLigazon(runas, i)
      if (lig) {
        pechar()
        tramos.push({ texto: lig.texto, url: lig.url, negrita, cursiva })
        i = lig.fin
        continue
      }
    }
    if ((c === '*' || c === '_') && runas[i + 1] === c) {
      if (negrita || enxeAbre(runas, i, true)) {
        pechar()
        negrita = !negrita
        i++
        continue
      }
    } else if (c === '*' || c === '_') {
      if (cursiva || enxeAbre(runas, i, false)) {
        pechar()
        cursiva = !cursiva
        continue
      }
    }
    buf += c
  }
  pechar()
  return tramos.length ? tramos : [{ texto: '' }]
}

function parseAlin(linea) {
  const nome = linea.replace(/^:::/, '').trim().toLowerCase()
  if (nome === 'esquerda' || nome === 'left') return 'left'
  if (nome === 'centro' || nome === 'center' || nome === 'centrado') return 'center'
  if (nome === 'dereita' || nome === 'right') return 'right'
  if (nome === 'xustificado' || nome === 'justify') return 'justify'
  return ''
}

function parseLiñaImaxe(s) {
  if (!s.startsWith('![')) return null
  const runas = Array.from(s)
  const lig = parseLigazon(runas, 1)
  if (!lig || lig.fin !== runas.length - 1) return null
  const img = { alt: lig.texto, ref: lig.url, anchoPct: 0, anchoMM: 0, altoMM: 0 }
  const i = lig.url.lastIndexOf(' =')
  if (i >= 0) {
    img.ref = lig.url.slice(0, i).trim()
    const tam = lig.url.slice(i + 2).trim()
    if (tam.endsWith('%')) {
      const v = parseFloat(tam.slice(0, -1))
      if (v > 0) img.anchoPct = v
    } else {
      const [w, h] = tam.split('x')
      const wv = parseFloat(w)
      const hv = parseFloat(h)
      if (wv > 0) img.anchoMM = wv
      if (hv > 0) img.altoMM = hv
    }
  }
  img.ref = img.ref.replace(/^img:/, '')
  return img.ref ? img : null
}

export function parseMarkdown(md) {
  const bloques = []
  let alin = ''
  let para = []
  const pecharParrafo = () => {
    const texto = para.join(' ').trim()
    para = []
    if (texto) bloques.push({ tipo: 'p', alin, tramos: parseInline(texto) })
  }
  for (const linea of String(md).replace(/\r\n/g, '\n').split('\n')) {
    const t = linea.trim()
    if (t === '') pecharParrafo()
    else if (t.startsWith(':::')) { pecharParrafo(); alin = parseAlin(t) }
    else if (t.startsWith('### ')) { pecharParrafo(); bloques.push({ tipo: 'h', nivel: 3, alin, tramos: parseInline(t.slice(4)) }) }
    else if (t.startsWith('## ')) { pecharParrafo(); bloques.push({ tipo: 'h', nivel: 2, alin, tramos: parseInline(t.slice(3)) }) }
    else if (t.startsWith('# ')) { pecharParrafo(); bloques.push({ tipo: 'h', nivel: 1, alin, tramos: parseInline(t.slice(2)) }) }
    else if (t.startsWith('> ')) { pecharParrafo(); bloques.push({ tipo: 'q', alin, tramos: parseInline(t.slice(2)) }) }
    else if (t.startsWith('- ') || t.startsWith('* ')) { pecharParrafo(); bloques.push({ tipo: 'li', ordenada: false, alin, tramos: parseInline(t.slice(2)) }) }
    else if (/^\d+\. /.test(t)) { pecharParrafo(); bloques.push({ tipo: 'li', ordenada: true, alin, tramos: parseInline(t.replace(/^\d+\. /, '')) }) }
    else {
      const img = parseLiñaImaxe(t)
      if (img) { pecharParrafo(); bloques.push({ tipo: 'img', alin, imaxe: img }) } else para.push(t)
    }
  }
  pecharParrafo()
  return bloques
}

// ---------- Vista previa en HTML ----------

const escapar = (s) => String(s).replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]))

// Só se aceptan ligazóns con estes protocolos (a vista previa vive na app,
// non se quere un javascript: nun href).
const URL_OK = /^(https?:|mailto:)/i

function htmlTramos(tramos) {
  return tramos.map((t) => {
    let h = escapar(t.texto)
    if (t.url) {
      h = URL_OK.test(t.url.trim()) ? `<a href="${escapar(t.url.trim())}" target="_blank" rel="noopener">${h}</a>` : h
    }
    if (t.negrita) h = `<strong>${h}</strong>`
    if (t.cursiva) h = `<em>${h}</em>`
    if (t.escala && t.escala !== 1) h = `<span style="font-size:${t.escala}em">${h}</span>`
    return h
  }).join('')
}

function estiloAlin(alin) {
  return alin ? ` style="text-align:${alin}"` : ''
}

function htmlImaxe(bl, imaxes) {
  const uri = /^data:image\//.test(bl.imaxe.ref) ? bl.imaxe.ref : imaxes[bl.imaxe.ref]
  if (!uri || !/^data:image\//.test(uri)) {
    return `<p class="md-aviso">[imaxe non atopada: ${escapar(bl.imaxe.ref)}]</p>`
  }
  const est = []
  if (bl.imaxe.anchoPct) est.push(`width:${bl.imaxe.anchoPct}%`)
  else if (bl.imaxe.anchoMM) est.push(`width:${bl.imaxe.anchoMM}mm`)
  if (bl.imaxe.altoMM) est.push(`height:${bl.imaxe.altoMM}mm`)
  est.push('max-width:100%')
  const alin = bl.alin === 'left' || bl.alin === 'right' ? bl.alin : 'center'
  return `<p style="text-align:${alin}"><img src="${escapar(uri)}" alt="${escapar(bl.imaxe.alt)}" style="${est.join(';')}"></p>`
}

export function markdownAHtml(md, imaxes = {}) {
  const bloques = parseMarkdown(md)
  let h = ''
  for (let i = 0; i < bloques.length;) {
    const b = bloques[i]
    if (b.tipo === 'li') {
      let j = i
      while (j < bloques.length && bloques[j].tipo === 'li' && bloques[j].ordenada === b.ordenada) j++
      const etq = b.ordenada ? 'ol' : 'ul'
      h += `<${etq}>${bloques.slice(i, j).map((x) => `<li${estiloAlin(x.alin)}>${htmlTramos(x.tramos)}</li>`).join('')}</${etq}>`
      i = j
      continue
    }
    if (b.tipo === 'h') h += `<h${b.nivel + 3}${estiloAlin(b.alin)}>${htmlTramos(b.tramos)}</h${b.nivel + 3}>`
    else if (b.tipo === 'q') h += `<blockquote${estiloAlin(b.alin)}>${htmlTramos(b.tramos)}</blockquote>`
    else if (b.tipo === 'img') h += htmlImaxe(b, imaxes)
    else h += `<p${estiloAlin(b.alin)}>${htmlTramos(b.tramos)}</p>`
    i++
  }
  return h
}
