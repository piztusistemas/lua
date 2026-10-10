// Modelo dos formularios propios (funcións puras, sen DOM): o esquema que
// deseña o usuario, o documento que enche (o «plan» da ferramenta) e a
// importación de CSV. O esquema é o mesmo JSON que garda Go (formulario.go):
// os nomes dos campos coinciden.
//
// Todo é inmutable: cada función devolve un obxecto novo.

// Orde da paleta do deseñador: primeiro a estrutura (sección, táboa), despois
// os datos simples e ao final o cálculo (o campo «problemas»), destacado.
export const TIPOS = ['seccion', 'taboa', 'texto', 'textolongo', 'numero', 'data', 'escolla', 'verificacion', 'imaxe', 'problemas']
// Tipos que poden ser columna dunha táboa (Go valida o mesmo).
export const TIPOS_COLUMNA = ['texto', 'numero', 'data', 'escolla', 'verificacion']

// Nome por defecto de cada tipo (a interface traduce con t('formularios.tipo.<tipo>')).
export const NOME_TIPO = {
  texto: 'Texto curto', textolongo: 'Texto longo', numero: 'Número', data: 'Data', escolla: 'Escolla',
  verificacion: 'Si / Non', imaxe: 'Imaxe', taboa: 'Táboa', problemas: 'Cálculo', seccion: 'Sección',
}

const SEN_ACENTO = { á: 'a', à: 'a', â: 'a', ã: 'a', ä: 'a', é: 'e', è: 'e', ê: 'e', ë: 'e', í: 'i', ì: 'i', î: 'i', ï: 'i', ó: 'o', ò: 'o', ô: 'o', õ: 'o', ö: 'o', ú: 'u', ù: 'u', û: 'u', ü: 'u', ñ: 'n', ç: 'c' }

// semAcentos: minúsculas e sen acentos, para comparar cabeceiras de CSV.
export function normalizarTexto(s) {
  return [...String(s ?? '').toLowerCase()].map((c) => SEN_ACENTO[c] ?? c).join('').replace(/[^a-z0-9]+/g, ' ').trim()
}

// slug é o identificador que Go tira dunha etiqueta (o mesmo criterio que slugFormulario).
export function slug(s) {
  let out = [...String(s ?? '').toLowerCase()].map((c) => SEN_ACENTO[c] ?? c).join('').replace(/[^a-z0-9]+/g, '_').replace(/^_+|_+$/g, '')
  if (/^[0-9]/.test(out)) out = 'c_' + out
  return out.slice(0, 40).replace(/_+$/g, '')
}

// ---------- Esquema ----------

export const novoEsquema = () => ({
  id: '', nome: '', icona: 'formulario', descricion: '', titulo: '', portada: false, sinatura: false, campos: [],
})

// novoCampo devolve un campo do tipo dado. O id queda baleiro: Go pono ao
// gardar (a partir da etiqueta) e despois xa non cambia.
export function novoCampo(tipo, etiqueta = '') {
  const c = { id: '', tipo, etiqueta: etiqueta || NOME_TIPO[tipo] || tipo }
  if (tipo === 'escolla') c.opcions = ['Opción 1', 'Opción 2']
  if (tipo === 'taboa') c.columnas = [novoCampo('texto', 'Columna 1'), novoCampo('texto', 'Columna 2')]
  return c
}

export function inserirCampo(campos, campo, indice) {
  const i = Math.max(0, Math.min(indice ?? campos.length, campos.length))
  return [...campos.slice(0, i), campo, ...campos.slice(i)]
}

// moverCampo move o campo `de` ao sitio `a` (índice de inserción na lista ORIXINAL: 0..n).
export function moverCampo(campos, de, a) {
  if (de < 0 || de >= campos.length) return campos
  const destino = a > de ? a - 1 : a
  if (destino === de) return campos
  const copia = [...campos]
  const [c] = copia.splice(de, 1)
  copia.splice(Math.max(0, Math.min(destino, copia.length)), 0, c)
  return copia
}

export const eliminarCampo = (campos, i) => campos.filter((_, j) => j !== i)
export const cambiarCampo = (campos, i, cambios) => campos.map((c, j) => (j === i ? { ...c, ...cambios } : c))

// duplicarCampo copia un campo baixo o orixinal, sen id (Go dálle un novo).
export function duplicarCampo(campos, i) {
  const c = { ...campos[i], id: '' }
  if (c.columnas) c.columnas = c.columnas.map((x) => ({ ...x, id: '' }))
  return inserirCampo(campos, c, i + 1)
}

// completarIds pon id aos campos (e ás columnas das táboas) que aínda non o
// teñen, a partir da etiqueta (o mesmo criterio ca Go). Un id xa posto NUNCA
// cambia: as tuberías e as variables da plantilla refírense a el. Faise no
// momento en que se precisa (abrir as tuberías, gardar, ler a IA), non ao
// escribir, para que a etiqueta xa sexa a boa.
export function completarIds(campos) {
  const usados = new Set(campos.map((c) => c.id).filter(Boolean))
  const unico = (base, conxunto) => {
    let id = base || 'campo'
    for (let n = 2; conxunto.has(id); n++) id = `${base || 'campo'}_${n}`
    conxunto.add(id)
    return id
  }
  return campos.map((c) => {
    const out = { ...c, id: c.id || unico(slug(c.etiqueta), usados) }
    if (c.tipo === 'taboa' && Array.isArray(c.columnas)) {
      const cols = new Set(c.columnas.map((x) => x.id).filter(Boolean))
      out.columnas = c.columnas.map((x) => (x.id ? x : { ...x, id: unico(slug(x.etiqueta) || 'columna', cols) }))
    }
    return out
  })
}

// problemasEsquema avisa do que Go rexeitaría, para dicilo antes de gardar.
export function problemasEsquema(e) {
  if (!String(e.nome || '').trim()) return 'nome'
  for (const c of e.campos) {
    if (!String(c.etiqueta || '').trim()) return 'etiqueta'
    if (c.tipo === 'escolla' && !(c.opcions || []).some((o) => String(o).trim())) return 'opcions'
    if (c.tipo === 'taboa') {
      if (!(c.columnas || []).length) return 'columnas'
      if ((c.columnas || []).some((x) => !String(x.etiqueta || '').trim())) return 'etiqueta'
    }
  }
  return ''
}

// ---------- Documento (o plan da ferramenta) ----------

export const contextoBaleiro = () => ({ centro: '', localidade: '', curso: '', departamento: '', responsable: '', cargo: '', logo: '' })

export function novoPlanFormulario(id, contexto = {}) {
  return { formulario: id, contexto: { ...contextoBaleiro(), ...contexto }, valores: {} }
}

const FERRAMENTA = 'formulario'

// escribirFicheiroFormulario serializa o documento. `extra` engade campos á raíz
// (o borrador garda alí a ruta do ficheiro aberto).
export function escribirFicheiroFormulario(plan, extra = {}) {
  // `proba`: hai datos ficticios que puxo a IA (en produción ofrécese o botón para borralos).
  return JSON.stringify({ ferramenta: FERRAMENTA, formulario: plan.formulario, ...extra, datos: { contexto: plan.contexto, valores: plan.valores, ...(plan.datosProba ? { proba: true } : {}), ...(plan.senReferencias ? { senReferencias: true } : {}), ...(plan.referenciasOcultas && Object.keys(plan.referenciasOcultas).length ? { referenciasOcultas: plan.referenciasOcultas } : {}), ...(plan.xeracion ? { xeracion: plan.xeracion } : {}) } }, null, 2)
}

// lerFicheiroFormulario le un documento e devolve [plan] (o controlador agarda
// unha lista de «cursos»). Falla se o ficheiro é doutro formulario ou doutra ferramenta.
export function lerFicheiroFormulario(texto, id, nome = '') {
  let dato
  try {
    dato = JSON.parse(texto)
  } catch {
    throw new Error('O ficheiro non é un JSON válido.')
  }
  if (!dato || dato.ferramenta !== FERRAMENTA) {
    throw new Error('O ficheiro non é un documento de formulario. Ábreo desde a ferramenta á que pertence.')
  }
  if (dato.formulario !== id) {
    throw new Error(`Este ficheiro é doutro formulario («${dato.formulario}»${nome ? `, e este é «${nome}»` : ''}).`)
  }
  const d = dato.datos || {}
  return [{
    formulario: id,
    contexto: { ...contextoBaleiro(), ...(d.contexto && typeof d.contexto === 'object' ? d.contexto : {}) },
    valores: d.valores && typeof d.valores === 'object' && !Array.isArray(d.valores) ? d.valores : {},
    ...(d.proba === true ? { datosProba: true } : {}),
    ...(d.senReferencias === true ? { senReferencias: true } : {}),
    ...(d.referenciasOcultas && typeof d.referenciasOcultas === 'object' && !Array.isArray(d.referenciasOcultas) ? { referenciasOcultas: d.referenciasOcultas } : {}),
    // Semente, copias e solucións dos problemas (a barra de xeración).
    ...(d.xeracion && typeof d.xeracion === 'object' && !Array.isArray(d.xeracion) ? { xeracion: d.xeracion } : {}),
  }]
}

// datosParaExportar serializa os datos do documento para levalos dentro dun
// formulario exportado ({contexto, valores, proba}, o mesmo que «datos» do
// ficheiro do documento). «» se non hai nada escrito: un formulario sen datos
// exporta só a estrutura.
export function datosParaExportar(plan) {
  if (!haiContidoFormulario(plan)) return ''
  return JSON.stringify({ contexto: plan.contexto || {}, valores: plan.valores || {}, ...(plan.datosProba ? { proba: true } : {}) })
}

export function nomeFicheiroSuxeridoFormulario(plan, esquema) {
  const curso = String(plan?.contexto?.curso || '').replace(/[^\p{L}\p{N}]+/gu, '-').replace(/^-+|-+$/g, '')
  return `${esquema?.id || plan?.formulario || 'formulario'}${curso ? '-' + curso : ''}`
}

const valorHaiContido = (v) => {
  if (Array.isArray(v)) return v.some((f) => f && typeof f === 'object' && Object.values(f).some((x) => valorHaiContido(x)))
  if (typeof v === 'boolean') return v
  return String(v ?? '').trim() !== ''
}

export function haiContidoFormulario(plan) {
  return Object.values(plan?.valores || {}).some(valorHaiContido)
    || Object.values(plan?.contexto || {}).some((v) => String(v ?? '').trim() !== '')
}

// ---------- CSV ----------

// decodificarCSV le os bytes dun CSV: UTF-8 (co BOM que pon Excel, se o ten) e,
// se non é UTF-8 válido, Windows-1252 (o que garda Excel en galego/castelán).
export function decodificarCSV(buffer) {
  try {
    return new TextDecoder('utf-8', { fatal: true }).decode(buffer).replace(/^﻿/, '')
  } catch {
    return new TextDecoder('windows-1252').decode(buffer)
  }
}

// separadorCSV escolle «;», «,» ou tabulador: o que máis se repite na primeira
// liña fóra de comiñas (Excel en galego usa «;» porque a coma é a decimal).
export function separadorCSV(texto) {
  const conta = { ';': 0, ',': 0, '\t': 0 }
  let dentro = false
  for (const c of texto) {
    if (c === '"') dentro = !dentro
    else if (!dentro && (c === '\n' || c === '\r')) break
    else if (!dentro && c in conta) conta[c]++
  }
  const [mellor, n] = Object.entries(conta).sort((a, b) => b[1] - a[1])[0]
  return n > 0 ? mellor : ','
}

// parseCSV devolve as filas (listas de cadeas) sen as liñas baleiras. Vale
// para comiñas dobres, «""» escapado e saltos de liña dentro de comiñas.
export function parseCSV(texto, separador = separadorCSV(texto)) {
  const filas = []
  let fila = []
  let cela = ''
  let dentro = false
  const pecharCela = () => { fila.push(cela); cela = '' }
  const pecharFila = () => {
    pecharCela()
    if (fila.some((c) => c.trim() !== '')) filas.push(fila)
    fila = []
  }
  for (let i = 0; i < texto.length; i++) {
    const c = texto[i]
    if (dentro) {
      if (c === '"') {
        if (texto[i + 1] === '"') { cela += '"'; i++ } else dentro = false
      } else cela += c
    } else if (c === '"' && cela === '') dentro = true
    else if (c === separador) pecharCela()
    else if (c === '\n') pecharFila()
    else if (c === '\r') { if (texto[i + 1] === '\n') i++; pecharFila() }
    else cela += c
  }
  if (cela !== '' || fila.length) pecharFila()
  return filas
}

// taboaDeCSV converte as filas dun CSV en filas dunha táboa do formulario.
// Se algunha cela da primeira fila coincide coa etiqueta dunha columna, é a
// cabeceira e as columnas casan polo nome; se non, casan por orde. Devolve
// {filas, cabeceira, casadas} para dicirlle ao usuario que se fixo.
export function taboaDeCSV(csv, columnas) {
  if (!csv.length || !columnas.length) return { filas: [], cabeceira: false, casadas: 0 }
  const etiquetas = columnas.map((c) => normalizarTexto(c.etiqueta))
  const primeira = csv[0].map(normalizarTexto)
  const cabeceira = primeira.some((h) => h && etiquetas.includes(h))
  let indices // indices[k] = posición no CSV da columna k da táboa (ou -1)
  if (cabeceira) {
    indices = etiquetas.map((e) => (e ? primeira.indexOf(e) : -1))
  } else {
    indices = columnas.map((_, k) => (k < csv[0].length ? k : -1))
  }
  const filas = csv.slice(cabeceira ? 1 : 0).map((linha) => {
    const f = {}
    columnas.forEach((col, k) => {
      const i = indices[k]
      const v = i >= 0 ? String(linha[i] ?? '').trim() : ''
      f[col.id] = col.tipo === 'verificacion' ? ['si', 'sí', 'true', '1', 'x', 'yes'].includes(v.toLowerCase()) : v
    })
    return f
  })
  return { filas, cabeceira, casadas: indices.filter((i) => i >= 0).length }
}

// ---------- Referencias entre [corchetes] ----------
// A IA deixa dentro dos textos referencias coma [PAU 2021 · ordinaria] ou
// [clave OFICIAL 2023: …]. O check do editor quítaas de TODOS os textos ao
// momento (e así tampouco saen no documento) e, ao volver activalo, repóñenas.
// Para poder repoñelas, ao quitalas gárdase o texto orixinal de cada valor en
// `referenciasOcultas` (ruta -> texto); só se repón onde o texto segue sendo o
// quitado (se o usuario o editou mentres, respéctase o que escribiu).
// É o mesmo criterio ca Go (quitarReferencias, formulario_datos.go).

const RE_REFERENCIA = /[ \t]*\[[^[\]\n]{3,}\](?![([])/g

export function quitarReferencias(s) {
  if (typeof s !== 'string' || !s.includes('[')) return s
  return s.replace(RE_REFERENCIA, '').split('\n').map((l) => l.replace(/[ \t]+$/, '')).join('\n')
}

const SEP = '\u0001'

function percorrer(v, ruta, f) {
  if (typeof v === 'string') return f(v, ruta)
  if (Array.isArray(v)) return v.map((x, i) => percorrer(x, [...ruta, i], f))
  if (v && typeof v === 'object') return Object.fromEntries(Object.entries(v).map(([k, x]) => [k, percorrer(x, [...ruta, k], f)]))
  return v
}

// ocultarReferencias quita as referencias de todos os textos e lembra os orixinais.
export function ocultarReferencias(plan) {
  const ocultas = { ...(plan.referenciasOcultas || {}) }
  const valores = percorrer(plan.valores || {}, [], (txt, ruta) => {
    const sen = quitarReferencias(txt)
    if (sen !== txt) ocultas[ruta.join(SEP)] = txt
    return sen
  })
  return { ...plan, valores, senReferencias: true, referenciasOcultas: ocultas }
}

// amosarReferencias repón os textos orixinais onde non se tocaron.
export function amosarReferencias(plan) {
  const ocultas = plan.referenciasOcultas || {}
  const valores = percorrer(plan.valores || {}, [], (txt, ruta) => {
    const orixinal = ocultas[ruta.join(SEP)]
    return orixinal !== undefined && quitarReferencias(orixinal) === txt ? orixinal : txt
  })
  const { referenciasOcultas, ...resto } = plan // eslint-disable-line no-unused-vars
  return { ...resto, valores, senReferencias: false }
}
