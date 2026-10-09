// Modelo das tuberías (funcións puras, sen DOM). O grafo é o mesmo JSON que
// avalía Go (formulario_fluxo.go): {nodos:[{id,tipo,x,y,campo,config,entradas}],
// documento:[ids dos destinos]}. `catalogo` é a lista de tipos de nodo que dá
// Go (CatalogoFluxo): entradas, parámetros, categoría.
//
// Todo é inmutable: cada función devolve un fluxo novo.

export const indexarCatalogo = (catalogo) => Object.fromEntries(catalogo.map((t) => [t.tipo, t]))

// aceptaTipo di se un porto de tipo `entrada` recibe unha saída de tipo `saida`
// (o mesmo que aceptaTipo en Go).
export function aceptaTipo(entrada, saida) {
  if (!saida) return true // aínda non se sabe: Go dirá se non cadra
  if (entrada === 'calquera') return saida !== 'taboa' && saida !== 'imaxe'
  if (entrada === 'texto') return ['texto', 'numero', 'data', 'bool'].includes(saida)
  return entrada === saida
}

export const eDestino = (tipoCat) => tipoCat?.categoria === 'destino'
export const nodoPorId = (fluxo, id) => fluxo.nodos.find((n) => n.id === id)

// idNodoLibre: base, base_2, base_3... que non use xa outro nodo.
export function idNodoLibre(fluxo, base) {
  const usados = new Set(fluxo.nodos.map((n) => n.id))
  let id = base
  for (let n = 2; usados.has(id); n++) id = `${base}_${n}`
  return id
}

// portosDe devolve as entradas que se pintan nun nodo: as fixas do catálogo e,
// nos nodos con portos dinámicos (a, b, c...), os que xa teñen ligazón máis UN
// libre para engadir o seguinte (mentres quede algún).
export function portosDe(nodo, tipoCat) {
  if (!tipoCat) return []
  const ent = nodo.entradas || {}
  const portos = (tipoCat.entradas || []).map((p) => ({ ...p, dinamico: false, ligado: !!ent[p.nome] }))
  const dins = tipoCat.dinamicas || []
  if (dins.length) {
    for (const nome of dins) if (ent[nome]) portos.push({ nome, tipo: tipoCat.dinTipo, opcional: true, dinamico: true, ligado: true })
    const libre = dins.find((nome) => !ent[nome])
    if (libre) portos.push({ nome: libre, tipo: tipoCat.dinTipo, opcional: true, dinamico: true, ligado: false })
  }
  return portos
}

// Os portos dinámicos «v1, v2, v10» ordénanse polo número, non polo texto.
export const ordePorto = (nome) => {
  const m = /^v(\d+)$/.exec(nome)
  return m ? Number(m[1]) : 1000 + nome.charCodeAt(0)
}

// valoresPorDefecto: o `defecto` de cada parámetro do catálogo.
export function configPorDefecto(tipoCat) {
  const c = {}
  for (const p of tipoCat.config || []) if (p.defecto !== undefined) c[p.clave] = p.defecto
  return c
}

// novoNodo engade un nodo do tipo dado. Os destinos entran ao final do documento.
// Devolve {fluxo, id}.
export function novoNodo(fluxo, catalogo, tipo, { x = 30, y = 30, config = {}, campo } = {}) {
  const cat = indexarCatalogo(catalogo)[tipo]
  if (!cat) throw new Error('tipo de nodo descoñecido: ' + tipo)
  const id = idNodoLibre(fluxo, (eDestino(cat) ? 'd_' : 'n_') + tipo)
  const nodo = { id, tipo, x: Math.round(x), y: Math.round(y), config: { ...configPorDefecto(cat), ...config } }
  if (campo) nodo.campo = campo
  return {
    id,
    fluxo: { nodos: [...fluxo.nodos, nodo], documento: eDestino(cat) ? [...fluxo.documento, id] : fluxo.documento },
  }
}

// creaCiclo: conectar `orixe` a `destino` pecharía un ciclo se `orixe` xa
// depende (directa ou indirectamente) de `destino`.
export function creaCiclo(fluxo, orixeId, destinoId) {
  if (orixeId === destinoId) return true
  const vistos = new Set()
  const depende = (id) => {
    if (id === destinoId) return true
    if (vistos.has(id)) return false
    vistos.add(id)
    const n = nodoPorId(fluxo, id)
    return Object.values(n?.entradas || {}).some(depende)
  }
  return depende(orixeId)
}

// conectar liga a saída de `orixeId` ao porto `porto` de `destinoId` (substitúe o que houbese).
// Devolve {fluxo} ou {erro: 'ciclo' | 'destino' | 'tipo' | 'porto' | 'nodo'}.
export function conectar(fluxo, catalogo, orixeId, destinoId, porto, tipoSaida) {
  const idx = indexarCatalogo(catalogo)
  const o = nodoPorId(fluxo, orixeId)
  const d = nodoPorId(fluxo, destinoId)
  if (!o || !d) return { erro: 'nodo' }
  if (eDestino(idx[o.tipo])) return { erro: 'destino' }
  const cat = idx[d.tipo]
  const p = portosDe(d, cat).find((x) => x.nome === porto)
  if (!p) return { erro: 'porto' }
  if (!aceptaTipo(p.tipo, tipoSaida)) return { erro: 'tipo' }
  if (creaCiclo(fluxo, orixeId, destinoId)) return { erro: 'ciclo' }
  return {
    fluxo: { ...fluxo, nodos: fluxo.nodos.map((n) => (n.id === destinoId ? { ...n, entradas: { ...(n.entradas || {}), [porto]: orixeId } } : n)) },
  }
}

export function desconectar(fluxo, destinoId, porto) {
  return {
    ...fluxo,
    nodos: fluxo.nodos.map((n) => {
      if (n.id !== destinoId) return n
      const { [porto]: _quitado, ...resto } = n.entradas || {}
      return { ...n, entradas: resto }
    }),
  }
}

// eliminarNodo quita o nodo, as ligazóns que saían del e o seu sitio no documento.
export function eliminarNodo(fluxo, id) {
  return {
    nodos: fluxo.nodos.filter((n) => n.id !== id).map((n) => {
      if (!n.entradas || !Object.values(n.entradas).includes(id)) return n
      return { ...n, entradas: Object.fromEntries(Object.entries(n.entradas).filter(([, o]) => o !== id)) }
    }),
    documento: fluxo.documento.filter((d) => d !== id),
  }
}

export function moverNoDocumento(fluxo, id, delta) {
  const i = fluxo.documento.indexOf(id)
  const j = i + delta
  if (i < 0 || j < 0 || j >= fluxo.documento.length) return fluxo
  const doc = [...fluxo.documento]
  ;[doc[i], doc[j]] = [doc[j], doc[i]]
  return { ...fluxo, documento: doc }
}

export const poñerConfig = (fluxo, id, clave, valor) => ({
  ...fluxo, nodos: fluxo.nodos.map((n) => (n.id === id ? { ...n, config: { ...(n.config || {}), [clave]: valor } } : n)),
})

export const poñerPosicion = (fluxo, id, x, y) => ({
  ...fluxo, nodos: fluxo.nodos.map((n) => (n.id === id ? { ...n, x: Math.round(x), y: Math.round(y) } : n)),
})

// nodosPorColumna reparte os nodos nas tres columnas do editor: os campos
// (datos) á esquerda, os destinos (documento, en orde) á dereita e o resto
// (transformacións e valores fixos) no medio, libres.
export function nodosPorColumna(fluxo, catalogo) {
  const idx = indexarCatalogo(catalogo)
  const datos = fluxo.nodos.filter((n) => n.tipo === 'campo')
  const medio = fluxo.nodos.filter((n) => n.tipo !== 'campo' && !eDestino(idx[n.tipo]))
  const enDoc = new Set(fluxo.documento)
  const documento = [
    ...fluxo.documento.map((id) => nodoPorId(fluxo, id)).filter(Boolean),
    ...fluxo.nodos.filter((n) => eDestino(idx[n.tipo]) && !enDoc.has(n.id)),
  ]
  return { datos, medio, documento }
}

// profundidades: cantos «saltos» hai desde os datos ata cada nodo (os campos = 0).
// Serve para colocar as transformacións de esquerda a dereita segundo dependan
// unhas das outras. Tolera ciclos (cortan en 0).
export function profundidades(fluxo) {
  const prof = new Map()
  const fondo = (id, ruta = new Set()) => {
    if (prof.has(id)) return prof.get(id)
    const n = nodoPorId(fluxo, id)
    if (!n || ruta.has(id)) return 0
    if (n.tipo === 'campo') { prof.set(id, 0); return 0 }
    ruta.add(id)
    const d = 1 + Math.max(0, ...Object.values(n.entradas || {}).map((o) => fondo(o, ruta)))
    ruta.delete(id)
    prof.set(id, d)
    return d
  }
  for (const n of fluxo.nodos) fondo(n.id)
  return prof
}

// posicionInicial: onde poñer un nodo novo no medio sen tapar os que hai.
export function posicionInicial(fluxo, catalogo) {
  const { medio } = nodosPorColumna(fluxo, catalogo)
  const n = medio.length
  return { x: 24 + (n % 4) * 34, y: 24 + Math.floor(n / 4) * 30 + (n % 4) * 90 }
}

// fluxoIgual compara dous fluxos ignorando as posicións (para saber se cambiou o que importa).
export function fluxoIgual(a, b) {
  const limpo = (f) => JSON.stringify(f && { ...f, nodos: f.nodos.map(({ x: _x, y: _y, ...r }) => r) })
  return limpo(a) === limpo(b)
}
