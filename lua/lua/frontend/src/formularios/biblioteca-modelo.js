// Modelo da BIBLIOTECA de formularios (funcións puras, sen DOM): cartafoles,
// zona fixa da columna esquerda e ferramentas agochadas. Espello do que
// garda Go (biblioteca_formularios.go, que ademais normaliza e valida): aquí
// só se calcula o novo estado; quen chama envíao a Go e usa o que devolve.
//
// Os ids das ferramentas son os da columna: 'cronograma', 'actas' e 'form:<id>'.

export const INTEGRADAS = ['cronograma', 'actas']

// Por defecto o cronograma e as actas só están na biblioteca (agochadas): o usuario activa as que quere.
export const bibliotecaBaleira = () => ({ carpetas: [], carpeta: {}, fixos: [], agochados: [...INTEGRADAS] })

// completar acepta o que devolve Go (ou nada) e garante todas as claves.
export function completar(b) {
  const o = b && typeof b === 'object' ? b : {}
  return {
    carpetas: Array.isArray(o.carpetas) ? o.carpetas : [],
    carpeta: o.carpeta && typeof o.carpeta === 'object' ? o.carpeta : {},
    fixos: Array.isArray(o.fixos) ? o.fixos : [],
    agochados: Array.isArray(o.agochados) ? o.agochados : [],
  }
}

export const esIntegrada = (id) => INTEGRADAS.includes(id)
export const idDeFormulario = (ferr) => (ferr.startsWith('form:') ? ferr.slice(5) : null)
export const estaFixa = (b, id) => b.fixos.includes(id)
export const estaAgochada = (b, id) => b.agochados.includes(id)

// Os formularios da zona fixa non se poden borrar (as ferramentas integradas
// tampouco: nin sequera son formularios).
export const podeBorrar = (b, id) => !esIntegrada(id) && !estaFixa(b, id)

// columna reparte as ferramentas que existen en zona fixa (na orde de b.fixos)
// e resto (os formularios que non están fixados, na orde recibida), sen as agochadas.
//   existentes: ids en orde (integradas primeiro, despois formularios por nome).
export function columna(b, existentes) {
  const hai = new Set(existentes)
  const agochada = (id) => b.agochados.includes(id)
  const fixas = b.fixos.filter((id) => hai.has(id) && !agochada(id))
  const resto = existentes.filter((id) => !b.fixos.includes(id) && !agochada(id))
  return { fixas, resto }
}

const sen = (lista, id) => lista.filter((x) => x !== id)

// fixar deixa `id` na zona fixa, diante de `antesDe` (ou ao final). Se xa estaba, móveo.
export function fixar(b, id, antesDe = null) {
  const resto = sen(b.fixos, id)
  const i = antesDe ? resto.indexOf(antesDe) : -1
  const fixos = i < 0 ? [...resto, id] : [...resto.slice(0, i), id, ...resto.slice(i)]
  return { ...b, fixos }
}

// desfixar quita unha ferramenta da zona fixa (pasa á de abaixo).
export function desfixar(b, id) {
  return { ...b, fixos: sen(b.fixos, id) }
}

export function agochar(b, id, agochado = true) {
  return { ...b, agochados: agochado ? [...sen(b.agochados, id), id] : sen(b.agochados, id) }
}

// moverACarpeta: carpetaId baleiro = sen cartafol (a raíz).
export function moverACarpeta(b, formId, carpetaId) {
  const carpeta = { ...b.carpeta }
  if (carpetaId) carpeta[formId] = carpetaId
  else delete carpeta[formId]
  return { ...b, carpeta }
}

export function renomearCarpeta(b, carpetaId, nome) {
  return { ...b, carpetas: b.carpetas.map((c) => (c.id === carpetaId ? { ...c, nome } : c)) }
}

// eliminarCarpeta NUNCA borra formularios: pasan á raíz.
export function eliminarCarpeta(b, carpetaId) {
  const carpeta = Object.fromEntries(Object.entries(b.carpeta).filter(([, c]) => c !== carpetaId))
  return { ...b, carpetas: b.carpetas.filter((c) => c.id !== carpetaId), carpeta }
}

// nomeRepetido: existe xa un cartafol con ese nome (sen mirar maiúsculas)? `exceptoId` para renomear.
export function nomeRepetido(b, nome, exceptoId = null) {
  const n = String(nome).trim().toLowerCase()
  return b.carpetas.some((c) => c.id !== exceptoId && c.nome.trim().toLowerCase() === n)
}

// filtrar devolve os formularios dunha vista da biblioteca.
//   carpeta: null (todos) | '' (sen cartafol) | id do cartafol
//   busca: texto (nome ou descrición, sen mirar maiúsculas nin acentos)
const plano = (s) => String(s ?? '').normalize('NFD').replace(/[̀-ͯ]/g, '').toLowerCase()
export function filtrar(b, formularios, { carpeta = null, busca = '' } = {}) {
  const q = plano(busca).trim()
  return formularios.filter((f) => {
    if (carpeta !== null && (b.carpeta[f.id] || '') !== carpeta) return false
    return !q || plano(f.nome).includes(q) || plano(f.descricion).includes(q)
  })
}

export function contarPorCarpeta(b, formularios) {
  const n = { todos: formularios.length, raiz: 0 }
  for (const c of b.carpetas) n[c.id] = 0
  for (const f of formularios) {
    const c = b.carpeta[f.id]
    if (c && c in n) n[c]++
    else n.raiz++
  }
  return n
}

// destinoDeSoltar decide onde cae un arrastre na columna. `xeo` describe o que se ve:
//   { dentro: bool (o punteiro está sobre a columna), ySeparador: y do límite entre zonas,
//     fixas: [{id, mediaY}] (botóns visibles da zona fixa, de arriba abaixo) }
// Devolve { zona: 'fixa', antesDe: id|null } | { zona: 'resto' } | null (fóra da columna).
export function destinoDeSoltar(y, xeo) {
  if (!xeo.dentro) return null
  if (y >= xeo.ySeparador) return { zona: 'resto' }
  const seguinte = xeo.fixas.find((f) => y < f.mediaY)
  return { zona: 'fixa', antesDe: seguinte ? seguinte.id : null }
}

// accionDeSoltar traduce (o que se arrastra, onde cae) no novo estado da
// biblioteca, ou null se non hai nada que facer (un arrastre sen efecto).
export function accionDeSoltar(b, id, destino) {
  if (!destino) return null
  if (destino.zona === 'fixa') {
    const fixos = b.fixos
    const i = fixos.indexOf(id)
    // Soltar un fixo no seu propio sitio: nada.
    if (i >= 0) {
      const seguinte = fixos[i + 1] ?? null
      if (destino.antesDe === id || destino.antesDe === seguinte) return null
    }
    return fixar(b, id, destino.antesDe)
  }
  // Á zona de abaixo: só sae da zona fixa unha ferramenta fixada.
  if (!estaFixa(b, id)) return null
  return desfixar(b, id)
}
