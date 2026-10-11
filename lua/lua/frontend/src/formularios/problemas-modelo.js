// Modelo dos campos «problemas» (funcións puras, sen DOM). Unha pregunta é
// o mesmo JSON que garda Go (formulario_problemas.go):
//
//   { id, enunciado, imaxe, puntos,
//     variables: [{ nome, modo: 'fixa'|'variable', valor, de, ata, paso }],
//     calculos:  [{ nome, expr }],
//     solucion:  { texto, resultado, calculos, comprobacions, orixe, parametrica, valores } }
//
// As regras dos {…} son as de Yang: {nome} é o valor dunha variable ou dun
// cálculo; dentro dunha fórmula, unhas chaves pegadas a un comando, a «^», a
// «_» ou a outras chaves son LaTeX ($\dfrac{m}{V}$ son as letras), e o valor
// vai con espazo ($\dfrac{ {m} }{ {V} }$ ou $ {m} $).

export const MODO_FIXA = 'fixa'
export const MODO_VARIABLE = 'variable'

let contador = 0
export function idProblema() {
  contador += 1
  return 'p' + Date.now().toString(36) + contador.toString(36) + Math.random().toString(36).slice(2, 6)
}

export const novoProblema = () => ({ id: idProblema(), enunciado: '', puntos: '', variables: [] })

// problemasDe le o valor dun campo (sempre unha lista de preguntas con id).
export function problemasDe(v) {
  return (Array.isArray(v) ? v : []).filter((p) => p && typeof p === 'object').map((p) => (p.id ? p : { ...p, id: idProblema() }))
}

const IDENT = /^[A-Za-z_][A-Za-z0-9_]*$/
const MATES = /\$\$[\s\S]+?\$\$|\$[^$\n]+?\$|\\\([\s\S]+?\\\)|\\\[[\s\S]+?\\\]/g

// partesMates divide un texto en anacos de prosa e de fórmula.
export function partesMates(s) {
  const out = []
  let ultimo = 0
  for (const m of String(s ?? '').matchAll(MATES)) {
    if (m.index > ultimo) out.push({ mates: false, texto: s.slice(ultimo, m.index) })
    out.push({ mates: true, texto: m[0] })
    ultimo = m.index + m[0].length
  }
  if (ultimo < String(s ?? '').length) out.push({ mates: false, texto: s.slice(ultimo) })
  return out
}

const grupoLatex = (c) => /[A-Za-z^_}]/.test(c || '')

// variablesUsadas devolve os nomes {x} que o texto usa coma variables (na
// orde en que aparecen, sen repetir). Fóra das fórmulas, {expr} pode ser
// unha expresión de Maxima: só conta se é un identificador.
export function variablesUsadas(texto) {
  const out = []
  // Nun debuxo (```circuitikz, ```tikz) as chaves son de TikZ: só contan as
  // das etiquetas en fórmula, l=$R_1 = {R1}\,\Omega$.
  const debuxos = []
  const sen = String(texto ?? '').replace(/```(tikz|circuitikz)[ \t]*\n([\s\S]*?)\n```/g, (_m, _t, codigo) => {
    debuxos.push(...partesMates(codigo).filter((p) => p.mates).map((p) => p.texto))
    return '\n'
  })
  for (const parte of [...partesMates(sen), ...debuxos.map((texto) => ({ mates: true, texto }))]) {
    for (const m of parte.texto.matchAll(/\{\s*([A-Za-z_][A-Za-z0-9_]*)\s*\}/g)) {
      if (parte.mates && grupoLatex(parte.texto[m.index - 1])) continue
      if (!parte.mates && (parte.texto[m.index - 1] === '{' || parte.texto[m.index + m[0].length] === '}')) continue // {{ }} literais
      if (!out.includes(m[1])) out.push(m[1])
    }
  }
  return out
}

// nomesDeclarados: variables e cálculos (da pregunta e da solución).
export function nomesDeclarados(p) {
  const nomes = new Set()
  for (const v of p.variables || []) if (v.nome) nomes.add(v.nome.trim())
  for (const c of [...(p.calculos || []), ...(p.solucion?.calculos || [])]) if (c.nome) nomes.add(c.nome.split('(')[0].trim())
  return nomes
}

// sincronizarVariables engade as variables que o enunciado usa e aínda non
// están declaradas (fixas, con valor 1). Non quita ningunha: unha que xa non
// se usa márcaa a interface e quítaa o usuario.
// UNIDADES: as que poden ir entre chaves nunha fórmula ($50\,{mA}$) e non
// son variables (a mesma lista ca Go, formulario_problemas.go).
// Só as de dúas letras ou máis: as dunha (m, g, V, A...) adoitan ser variables.
export const UNIDADES = new Set(`mA kA uA mV kV mW kW MW Wh kWh kJ MJ cal kcal Hz kHz MHz GHz
  mF uF nF pF mH mC uC ohm Ohm kohm Mohm ms min cm mm km kg mg kN Pa kPa hPa MPa atm bar
  mL ml mol rpm dB lx cd`.split(/\s+/))

export function sincronizarVariables(p) {
  const declarados = nomesDeclarados(p)
  const novas = variablesUsadas(p.enunciado).filter((n) => IDENT.test(n) && !declarados.has(n) && !UNIDADES.has(n))
  if (!novas.length) return p
  return { ...p, variables: [...(p.variables || []), ...novas.map((nome) => ({ nome, modo: MODO_FIXA, valor: '1' }))] }
}

const LETRAS = 'abcdefghkmnpqrstuvwz'

// numerosAVariables converte os números das fórmulas do enunciado en
// variables fixas (co mesmo valor), para logo poder facelas variables. Non
// toca expoñentes nin subíndices (x^2, a_1) nin os números dos comandos
// (\sqrt[3]{}). Devolve a pregunta nova e cantas variables creou.
export function numerosAVariables(p) {
  const usados = nomesDeclarados(p)
  for (const n of variablesUsadas(p.enunciado)) usados.add(n)
  const libre = () => {
    for (const l of LETRAS) if (!usados.has(l)) { usados.add(l); return l }
    for (let i = 1; ; i++) if (!usados.has('k' + i)) { usados.add('k' + i); return 'k' + i }
  }
  const novas = []
  // Os debuxos (```circuitikz, ```tikz) non se tocan: os seus números son coordenadas.
  const enunciado = String(p.enunciado ?? '').split(/(```(?:tikz|circuitikz)[ \t]*\n[\s\S]*?\n```)/).map((anaco, i) => (i % 2 ? anaco : partesMates(anaco).map((parte) => {
    if (!parte.mates) return parte.texto
    return parte.texto.replace(/(\d+(?:[.,]\d+)?)/g, (num, _g, pos, s) => {
      const antes = s.slice(0, pos).replace(/\s+$/, '')
      const ult = antes.slice(-1)
      if (ult === '^' || ult === '_' || ult === '[' || /\d|[.,]/.test(s[pos - 1] || '') || /[A-Za-z]$/.test(s.slice(0, pos))) return num
      if (/[\^_]\{$/.test(antes)) return num
      const nome = libre()
      novas.push({ nome, modo: MODO_FIXA, valor: num.replace('.', ',') })
      return ` {${nome}} `
    }).replace(/ {2,}/g, ' ')
  }).join(''))).join('')
  if (!novas.length) return { problema: p, creadas: 0 }
  return { problema: { ...p, enunciado, variables: [...(p.variables || []), ...novas] }, creadas: novas.length }
}

// cambiarModo pasa unha variable de fixa a variable ou ao revés. Ao fixala
// queda co valor que tiña agora (o que se ve), e ao facela variable propón
// un rango arredor dese valor.
export function cambiarModo(v, valorAgora) {
  if (v.modo === MODO_VARIABLE) {
    return { nome: v.nome, modo: MODO_FIXA, valor: valorAgora ?? v.valor ?? '1' }
  }
  const n = Number(String(v.valor ?? '').replace(',', '.'))
  if (Number.isFinite(n) && Number.isInteger(n)) {
    const de = n > 0 ? Math.max(1, Math.round(n / 2)) : n - 5
    const ata = n > 0 ? Math.max(de + 1, n * 2) : n + 5
    return { nome: v.nome, modo: MODO_VARIABLE, valor: v.valor, de: String(de), ata: String(ata), paso: '1' }
  }
  if (Number.isFinite(n)) {
    return { nome: v.nome, modo: MODO_VARIABLE, valor: v.valor, de: String(+(n / 2).toFixed(2)).replace('.', ','), ata: String(+(n * 2).toFixed(2)).replace('.', ','), paso: '0,1' }
  }
  return { nome: v.nome, modo: MODO_VARIABLE, valor: v.valor, de: '1', ata: '10', paso: '1' }
}

// Os rangos van á ficha con punto decimal (Go e Maxima).
const numGo = (s) => String(s ?? '').trim().replace(',', '.')

// paraMotor: a pregunta tal e como a calcula Go (comas decimais a puntos).
export function paraMotor(p) {
  return {
    ...p,
    variables: (p.variables || []).map((v) => (v.modo === MODO_VARIABLE
      ? { nome: v.nome, modo: MODO_VARIABLE, de: numGo(v.de), ata: numGo(v.ata), paso: numGo(v.paso) || '1' }
      : { nome: v.nome, modo: MODO_FIXA, valor: String(v.valor ?? '').trim() })),
  }
}

// listaCalculo: as preguntas de TODOS os campos de problemas na orde dos
// campos, co seu índice no documento (o que dá a súa semente, igual ca en Go).
export function listaCalculo(esquema, valores) {
  const out = []
  let n = 0
  for (const c of esquema.campos || []) {
    if (c.tipo !== 'problemas') continue
    for (const p of problemasDe(valores?.[c.id])) out.push({ campo: c.id, indice: n++, problema: p })
  }
  return out
}

// solucionDesactualizada: unha solución que non é paramétrica só vale para
// os valores con que se resolveu.
export function solucionDesactualizada(p, calculado) {
  const s = p.solucion
  if (!s || s.parametrica || !s.valores || !calculado?.valores) return false
  return Object.entries(s.valores).some(([k, v]) => calculado.valores[k] && String(calculado.valores[k].texto).trim() !== String(v).trim())
}

export const temSolucion = (p) => Boolean(p.solucion && (String(p.solucion.texto || '').trim() || String(p.solucion.resultado || '').trim()))

// haiProblemas: o esquema ten algún campo de problemas.
export const haiProblemas = (esquema) => (esquema?.campos || []).some((c) => c.tipo === 'problemas')

// xeracionDe: as opcións de xeración do documento (semente, copias, solucións).
export function xeracionDe(plan) {
  const x = plan?.xeracion || {}
  return {
    semente: Number.isFinite(Number(x.semente)) ? Math.max(0, Math.trunc(Number(x.semente))) : 1,
    variantes: Math.min(50, Math.max(1, Math.trunc(Number(x.variantes)) || 1)),
    solucions: ['non', 'xunto', 'final', 'resultado-xunto', 'resultado-final'].includes(x.solucions) ? x.solucions : 'non',
  }
}
