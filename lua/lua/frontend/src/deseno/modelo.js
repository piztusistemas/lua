// MODELO DO DESEÑO DO PDF (funcións puras, con probas en modelo.test.js).
//
// O deseño é un obxecto de datos (edicion/plantilla/deseno.go): letra,
// tamaño, cores, espazos, interruptores e axustes por elemento. Aquí está o
// que a interface fai con el sen falar co backend:
//   - as barras (formal, aire, cor), que moven varios valores á vez;
//   - as variantes (escoller entre dúas propostas);
//   - validar e aplicar o que propón a IA (respectando os campos fixados);
//   - atopar o elemento que hai baixo o rato na vista previa.

export const COMBINACIONS = ['lua', 'clasica', 'elegante', 'lexible', 'moderna', 'academica']
export const ETIQUETAS = ['degradado', 'planas', 'contorno']
export const ELEMENTOS = ['texto', 'titulos', 'etiquetas', 'meses', 'trimestres', 'tarefas', 'actas_cabeceira', 'actas_apartados', 'actas_materias', 'sinaturas']
// Que controis ten cada elemento (o mesmo ca ElementosDeseno en Go).
export const CONTROIS_ELEMENTO = {
  texto: [],
  titulos: ['cor', 'escala', 'maiusculas', 'filete'],
  etiquetas: ['escala', 'maiusculas'],
  meses: ['escala'],
  trimestres: ['cor', 'escala', 'maiusculas', 'filete'],
  tarefas: ['cor', 'escala'],
  actas_cabeceira: ['cor', 'escala', 'filete'],
  actas_apartados: ['cor', 'escala', 'maiusculas', 'filete'],
  actas_materias: ['cor', 'escala'],
  sinaturas: ['escala'],
}
// Rangos numéricos (os mesmos ca NormalizarDeseno en Go).
export const RANGOS = {
  tamano: [8, 20], escala: [0.6, 1.8], interlinado: [0.9, 2.2], parrafo: [0, 3], marxe: [8, 40],
}
export const POR_DEFECTO = { tamano: 11, escala: 1, interlinado: 1, parrafo: 0.55, marxe: 20, corPrincipal: '4F46E5' }
const CORES = ['corPrincipal', 'corEscura', 'corTexto']
const BOOLS = ['brancoNegro', 'aforrarPapel', 'lecturaFacil']
const LETRAS = ['letraCorpo', 'letraTitulos', 'letraEtiquetas', 'letraMates']
const HEX = /^[0-9A-Fa-f]{6}$/
const ENUMS = { portada: ['', 'sobria', 'oficial', 'sen'], logoPortada: ['', 'esquerda', 'centro', 'dereita'], numeroPaxina: ['', 'esquerda', 'centro', 'non'] }

const erro = (m) => { throw new Error(m) }
const copia = (d) => JSON.parse(JSON.stringify(d || {}))

// valor devolve o valor efectivo dun campo (o gardado ou o por defecto).
export function valor(d, campo) {
  const v = d?.[campo]
  if (v === undefined || v === '' || v === 0 || v === null) return POR_DEFECTO[campo]
  return v
}

export function normalizarCor(c) {
  let s = String(c ?? '').trim().replace(/^#/, '')
  if (s.length === 3) s = s.replace(/./g, (x) => x + x)
  return HEX.test(s) ? s.toUpperCase() : ''
}

// limpar quita os valores baleiros (o deseño gárdase sen ruído).
export function limpar(d) {
  const o = {}
  for (const [k, v] of Object.entries(d || {})) {
    if (v === '' || v === 0 || v === false || v === null || v === undefined) continue
    if (k === 'elementos') {
      const els = {}
      for (const [id, e] of Object.entries(v || {})) {
        const le = limpar(e)
        if (Object.keys(le).length) els[id] = le
      }
      if (Object.keys(els).length) o.elementos = els
      continue
    }
    if (k === 'barras') {
      const b = limpar({ ...v, base: v.base && Object.keys(v.base).length ? v.base : undefined })
      if (Object.keys(b).length) o.barras = b
      continue
    }
    if (Array.isArray(v) && v.length === 0) continue
    o[k] = v
  }
  return o
}

export const fixado = (d, campo) => (d?.fixados || []).includes(campo)

// poñer cambia un campo se non está fixado.
function poñer(d, campo, v) {
  if (fixado(d, campo)) return
  d[campo] = v
}

// aplicarBarra move unha barra (-2..2) e cambia os valores que dependen dela.
// Formal: letra e escala dos títulos e aspecto das etiquetas. Aire:
// interliñado, parágrafo e marxes. Cor: cor principal e etiquetas.
// Campos que move cada barra (para lembrar os valores de partida).
const CAMPOS_BARRA = { formal: ['letra', 'escala', 'etiquetas'], aire: ['interlinado', 'parrafo', 'marxe'] }

export function aplicarBarra(deseno, barra, nivel) {
  const d = copia(deseno)
  const n = Math.max(-2, Math.min(2, Math.round(Number(nivel) || 0)))
  d.barras = { ...(d.barras || {}), [barra]: n }
  // Ao saír do medio, a barra lembra os valores que había; ao volver ao
  // medio, restáuraos (non os de Lúa).
  if (CAMPOS_BARRA[barra]) {
    const base = { ...(d.barras.base || {}) }
    const antes = deseno?.barras?.[barra] || 0
    if (antes === 0 && n !== 0) for (const c of CAMPOS_BARRA[barra]) base[c] = deseno?.[c] ?? ''
    if (n === 0) {
      for (const c of CAMPOS_BARRA[barra]) {
        if (c in base) { if (!fixado(d, c)) d[c] = base[c]; delete base[c] }
      }
      d.barras.base = base
      return limpar(d)
    }
    d.barras.base = base
  }
  if (barra === 'formal') {
    poñer(d, 'letra', ['moderna', 'lua', '', 'clasica', 'elegante'][n + 2])
    poñer(d, 'escala', [1.15, 1.07, 0, 0.95, 0.9][n + 2])
    poñer(d, 'etiquetas', ['degradado', 'degradado', '', 'planas', 'contorno'][n + 2])
  } else if (barra === 'aire') {
    poñer(d, 'interlinado', [0.95, 1, 0, 1.2, 1.35][n + 2])
    poñer(d, 'parrafo', [0.2, 0.35, 0, 0.8, 1.1][n + 2])
    poñer(d, 'marxe', [13, 16, 0, 24, 28][n + 2])
  } else if (barra === 'cor') {
    // A barra lembra a cor de partida (barras.corBase) para volver a ela no 0.
    const b = d.barras
    if (!b.corBase) b.corBase = normalizarCor(deseno?.corPrincipal) || POR_DEFECTO.corPrincipal
    const base = b.corBase
    if (n < 0) {
      poñer(d, 'corPrincipal', n === -2 ? '334155' : mesturar(base, '64748B', 0.6))
      poñer(d, 'etiquetas', 'contorno')
    } else if (n > 0) {
      poñer(d, 'corPrincipal', saturar(base, n))
      poñer(d, 'etiquetas', 'degradado')
    } else {
      poñer(d, 'corPrincipal', b.corBase === POR_DEFECTO.corPrincipal ? '' : b.corBase)
      poñer(d, 'etiquetas', '')
      delete b.corBase
    }
  } else {
    erro(`barra descoñecida: ${barra}`)
  }
  return limpar(d)
}

// ---------- Cores ----------
function rgb(h) {
  const v = parseInt(h, 16)
  return [(v >> 16) & 255, (v >> 8) & 255, v & 255].map((x) => x / 255)
}
function hex(r, g, b) {
  return [r, g, b].map((x) => Math.round(Math.max(0, Math.min(1, x)) * 255).toString(16).padStart(2, '0')).join('').toUpperCase()
}
export function mesturar(a, b, t) {
  const [ar, ag, ab] = rgb(a)
  const [br, bg, bb] = rgb(b)
  return hex(ar + (br - ar) * t, ag + (bg - ag) * t, ab + (bb - ab) * t)
}
function saturar(h, n) {
  const [r, g, b] = rgb(h)
  const m = (r + g + b) / 3
  const k = 1 + 0.25 * n
  return hex(m + (r - m) * k, m + (g - m) * k, m + (b - m) * k)
}
// contraste WCAG sobre branco (o corrector de Go escurece por debaixo de 4.5).
export function contrasteBranco(h) {
  const lin = (c) => (c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4)
  const [r, g, b] = rgb(h).map(lin)
  const l = 0.2126 * r + 0.7152 * g + 0.0722 * b
  return 1.05 / (l + 0.05)
}

// ---------- Variantes («como no oculista») ----------
// Cada rolda explora unha dimensión e propón dúas versións que só se
// distinguen nela. As dimensións fixadas sáltanse.
const DIMENSIONS = [
  { id: 'letra', campo: 'letra' },
  { id: 'cor', campo: 'corPrincipal' },
  { id: 'aire', campo: 'interlinado' },
  { id: 'etiquetas', campo: 'etiquetas' },
  { id: 'tamano', campo: 'tamano' },
]
const CORES_VARIANTES = ['1D4ED8', '0F766E', 'B91C1C', '7C3AED', 'C2410C', '334155', '15803D', 'BE185D']

// variantes(deseno, rolda, coresLogo) -> { dimension, a, b } ou null se todo está fixado.
export function variantes(deseno, rolda = 0, coresLogo = []) {
  const libres = DIMENSIONS.filter((x) => !fixado(deseno, x.campo))
  if (!libres.length) return null
  const dim = libres[((rolda % libres.length) + libres.length) % libres.length]
  const a = copia(deseno)
  const b = copia(deseno)
  const xiro = Math.floor(rolda / libres.length)
  switch (dim.id) {
    case 'letra': {
      const actual = deseno?.letra || 'lua'
      const outras = COMBINACIONS.filter((c) => c !== actual)
      a.letra = outras[(xiro * 2) % outras.length]
      b.letra = outras[(xiro * 2 + 1) % outras.length]
      break
    }
    case 'cor': {
      const actual = normalizarCor(deseno?.corPrincipal) || POR_DEFECTO.corPrincipal
      const lista = [...coresLogo.map(normalizarCor).filter(Boolean), ...CORES_VARIANTES].filter((c, i, l) => c !== actual && l.indexOf(c) === i)
      a.corPrincipal = lista[(xiro * 2) % lista.length]
      b.corPrincipal = lista[(xiro * 2 + 1) % lista.length]
      break
    }
    case 'aire': {
      a.interlinado = 1; a.parrafo = 0.3
      b.interlinado = 1.3; b.parrafo = 0.9
      break
    }
    case 'etiquetas': {
      const actual = deseno?.etiquetas || 'degradado'
      const outras = ETIQUETAS.filter((e) => e !== actual)
      a.etiquetas = outras[0]
      b.etiquetas = outras[1]
      break
    }
    case 'tamano': {
      const t = valor(deseno, 'tamano')
      a.tamano = Math.max(9, t - 1)
      b.tamano = Math.min(14, t + 1.5)
      break
    }
  }
  return { dimension: dim.id, a: limpar(a), b: limpar(b) }
}

// ---------- IA ----------
// validarParche comproba o que propón a IA para o deseño. Lanza un erro que
// explica o que falla (a IA leo e corrixe); devolve o parche limpo e avisos.
export function validarParche(p, deseno = {}) {
  if (!p || typeof p !== 'object' || Array.isArray(p)) erro('deseno ten que ser un obxecto')
  const out = {}
  const avisos = []
  for (const [k, v] of Object.entries(p)) {
    if (fixado(deseno, k)) { avisos.push(`«${k}» está fixado polo usuario: non se cambiou`); continue }
    if (k === 'letra') {
      if (v !== '' && !COMBINACIONS.includes(v)) erro(`letra «${v}» non existe (${COMBINACIONS.join(', ')})`)
      out.letra = v
    } else if (LETRAS.includes(k)) {
      const s = String(v ?? '').trim()
      if (s && !/^[\p{L}\p{N} .,'&()+_-]{1,80}$/u.test(s)) erro(`${k}: nome de letra non válido «${v}»`)
      out[k] = s
    } else if (k in RANGOS) {
      const n = Number(v)
      if (Number.isNaN(n)) erro(`${k}: ten que ser un número`)
      const [min, max] = RANGOS[k]
      if (n !== 0 && (n < min || n > max)) erro(`${k}: ${n} está fóra de rango (${min}-${max}; 0 = o de Lúa)`)
      out[k] = n
    } else if (CORES.includes(k)) {
      const c = normalizarCor(v)
      if (String(v ?? '').trim() && !c) erro(`${k}: cor non válida «${v}» (hexadecimal, p.ex. 1F3864)`)
      out[k] = c
    } else if (k === 'etiquetas') {
      if (v !== '' && !ETIQUETAS.includes(v)) erro(`etiquetas: «${v}» non existe (${ETIQUETAS.join(', ')})`)
      out.etiquetas = v
    } else if (BOOLS.includes(k)) {
      out[k] = !!v
    } else if (k in ENUMS) {
      if (!ENUMS[k].includes(v ?? '')) erro(`${k}: «${v}» non existe (${ENUMS[k].filter(Boolean).join(', ')} ou "")`)
      out[k] = v ?? ''
    } else if (k === 'cabeceira' || k === 'pe') {
      if (!v || typeof v !== 'object' || Array.isArray(v)) erro(`${k} ten que ser un obxecto {esquerda, centro, dereita}`)
      const z = {}
      for (const [zk, zv] of Object.entries(v)) {
        if (!['esquerda', 'centro', 'dereita'].includes(zk)) erro(`${k}.${zk} non existe (esquerda, centro, dereita)`)
        z[zk] = String(zv ?? '').slice(0, 200)
      }
      out[k] = { ...(deseno?.[k] || {}), ...z }
    } else if (k === 'elementos') {
      if (!v || typeof v !== 'object') erro('elementos ten que ser un obxecto id -> axustes')
      out.elementos = {}
      for (const [id, e] of Object.entries(v)) {
        if (!ELEMENTOS.includes(id)) erro(`elemento «${id}» non existe (${ELEMENTOS.join(', ')})`)
        if (e === null) { out.elementos[id] = null; continue }
        const el = {}
        for (const [ck, cv] of Object.entries(e || {})) {
          if (ck === 'cor') {
            const c = normalizarCor(cv)
            if (String(cv ?? '').trim() && !c) erro(`elementos.${id}.cor non válida «${cv}»`)
            el.cor = c
          } else if (ck === 'escala') {
            const n = Number(cv)
            if (Number.isNaN(n) || (n !== 0 && (n < 0.5 || n > 2.5))) erro(`elementos.${id}.escala: entre 0.5 e 2.5`)
            el.escala = n
          } else if (ck === 'maiusculas' || ck === 'senFilete') {
            el[ck] = !!cv
          } else if (ck === 'latex') {
            el.latex = String(cv ?? '')
          } else {
            avisos.push(`elementos.${id}.${ck} non existe: ignorado`)
          }
        }
        out.elementos[id] = el
      }
    } else if (k === 'fixados') {
      avisos.push('os campos fixados só os cambia o usuario')
    } else {
      avisos.push(`«${k}» non é un campo do deseño: ignorado`)
    }
  }
  return { parche: out, avisos }
}

// aplicarParche mestura un parche no deseño: os elementos mestúranse campo a
// campo e un elemento a null quítase.
export function aplicarParche(deseno, parche) {
  const d = copia(deseno)
  for (const [k, v] of Object.entries(parche || {})) {
    if (k !== 'elementos') { d[k] = v; continue }
    d.elementos = d.elementos || {}
    for (const [id, e] of Object.entries(v || {})) {
      if (e === null) delete d.elementos[id]
      else d.elementos[id] = { ...(d.elementos[id] || {}), ...e }
    }
  }
  return limpar(d)
}

// ---------- Inspector ----------
// elementoEnPunto: as marcas son a esquina de arriba á esquerda (aprox.) de
// cada elemento na páxina. O elemento baixo o rato é a marca máis próxima que
// está por riba del (ou un pouco por debaixo) na mesma páxina. Sen ningunha
// preto, é o texto.
export function marcaEnPunto(marcas, paxina, x, y) {
  let mellor = null
  let dist = Infinity
  for (const m of marcas || []) {
    if (m.paxina !== paxina) continue
    const dy = y - m.y
    if (dy < -0.015 || dy > 0.06) continue // o punto ten que estar xusto debaixo da marca
    const dx = Math.max(0, m.x - x) * 2 + Math.max(0, x - m.x) * 0.15
    const d = Math.abs(dy) + dx
    if (d < dist) { dist = d; mellor = m }
  }
  return mellor
}

export function elementoEnPunto(marcas, paxina, x, y) {
  return marcaEnPunto(marcas, paxina, x, y)?.elemento || 'texto'
}

// caixaMarca: a zona aproximada que ocupa unha parte (fraccións da páxina),
// para resaltala na vista previa: desde un pouco por riba da marca ata a
// seguinte marca de debaixo (como moito 6 % da páxina).
export function caixaMarca(marcas, m) {
  const seguinte = (marcas || [])
    .filter((o) => o.paxina === m.paxina && o.y > m.y + 0.005)
    .reduce((min, o) => Math.min(min, o.y), Infinity)
  const top = Math.max(0, m.y - 0.022)
  const fondo = Math.min(seguinte - 0.022, m.y + 0.04)
  return { x: Math.max(0, m.x - 0.01), y: top, w: Math.min(1, 0.92 - m.x + 0.01), h: Math.max(0.02, fondo - top) }
}

// parteDeCampo: que parte do documento se ve mellor ao cambiar un campo (a
// vista previa salta á súa páxina).
export function parteDeCampo(campo) {
  if (ELEMENTOS.includes(campo)) return campo
  return {
    letra: 'titulos', letraCorpo: 'titulos', letraTitulos: 'titulos', tamano: 'titulos', escala: 'titulos', corPrincipal: 'etiquetas',
    etiquetas: 'etiquetas', brancoNegro: 'etiquetas', interlinado: 'titulos', parrafo: 'titulos', marxe: 'titulos',
    portada: 'portada', logoPortada: 'portada', cabeceira: 'tarefas', pe: 'tarefas', numeroPaxina: 'tarefas',
  }[campo] || ''
}

// describir resume o deseño nunha liña para o asistente e as variantes.
export function describir(d, t = (k, def) => def) {
  const p = []
  p.push(t(`deseno.letra.${d?.letra || 'lua'}`, d?.letra || 'lua'))
  p.push(`${valor(d, 'tamano')} pt`)
  if (d?.corPrincipal) p.push('#' + d.corPrincipal)
  if (d?.brancoNegro) p.push(t('deseno.bn', 'branco e negro'))
  if (d?.aforrarPapel) p.push(t('deseno.aforrar', 'aforrar papel'))
  if (d?.lecturaFacil) p.push(t('deseno.lectura', 'lectura fácil'))
  return p.join(' · ')
}

// paxinaRepresentativa: a páxina (1, 2...) con máis partes do deseño, para
// amosar nas variantes a que máis se nota (a portada ou o índice non serven).
export function paxinaRepresentativa(res) {
  const conta = new Map()
  for (const m of res?.marcas || []) conta.set(m.paxina, (conta.get(m.paxina) || 0) + 1)
  let mellor = 1
  let max = -1
  for (const [p, n] of conta) if (n > max || (n === max && p < mellor)) { mellor = p; max = n }
  const total = (res?.pageImages || []).length
  return total ? Math.min(Math.max(1, mellor), total) : 1
}
