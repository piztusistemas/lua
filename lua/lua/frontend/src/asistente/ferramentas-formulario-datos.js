// Ferramentas do asistente dos DATOS dun formulario en modo deseño (ámbito
// "formulario_datos", catálogo en formulario_datos_ia.go). Ler e escribir os
// datos do documento aberto, e ler os documentos que se lle acheguen (útil
// para exames e prácticas a partir dun temario ou dun modelo xa feito). Cada
// valor pasa pola validación contra o esquema antes de entrar (a IA
// equivócase con tipos, opcións e ids, e a mensaxe exacta é o que lle permite
// corrixir); se algún falla, non cambia nada.
//
// PRIVACIDADE: en modo produción estas ferramentas negan todo. O frontend
// tampouco monta o asistente nese modo; isto é o segundo seguro, por se o
// modo cambia co panel aberto.

const TAMANO_MAX = 30000
const MAX_TEXTO = 20000
const MAX_FILAS = 500
const CLAVES_CONTEXTO = ['centro', 'localidade', 'curso', 'departamento', 'responsable', 'cargo']
const erro = (m) => { throw new Error(m) }

// dataISOValida comproba AAAA-MM-DD contra o calendario (un 31 de abril non vale).
function dataISOValida(s) {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(s)
  if (!m) return false
  const d = new Date(Date.UTC(+m[1], +m[2] - 1, +m[3]))
  return d.getUTCFullYear() === +m[1] && d.getUTCMonth() === +m[2] - 1 && d.getUTCDate() === +m[3]
}

// valorParaCampo valida e normaliza un valor segundo o tipo do campo (ou
// columna). `null`/'' baleiran. Devolve o valor tal e como se garda no documento
// (o mesmo que escribiría o usuario: números e datas son cadeas, as marcas booleanos).
export function valorParaCampo(c, v, ruta = c.id) {
  if (v === null || v === undefined || v === '') return c.tipo === 'verificacion' ? false : ''
  switch (c.tipo) {
    case 'texto':
    case 'textolongo': {
      if (typeof v === 'object') erro(`«${ruta}» é un texto: recibiu ${Array.isArray(v) ? 'unha lista' : 'un obxecto'}`)
      const s = String(v)
      if (s.length > MAX_TEXTO) erro(`«${ruta}»: texto demasiado longo (máximo ${MAX_TEXTO} caracteres)`)
      return s
    }
    case 'numero': {
      const s = String(v).trim()
      if (typeof v === 'object' || !/^-?\d+([.,]\d+)?$/.test(s)) erro(`«${ruta}» é un número: «${typeof v === 'object' ? JSON.stringify(v) : s}» non vale (usa 7,5 ou 7.5)`)
      return s
    }
    case 'data': {
      const s = String(v).trim()
      if (!dataISOValida(s)) erro(`«${ruta}» é unha data: «${s}» non vale (usa AAAA-MM-DD, p. ex. 2026-03-15)`)
      return s
    }
    case 'escolla': {
      const s = String(v)
      if (!(c.opcions || []).includes(s)) erro(`«${ruta}» é unha escolla: «${s}» non é unha das opcións (${(c.opcions || []).map((o) => `«${o}»`).join(', ')})`)
      return s
    }
    case 'verificacion': {
      if (v === true || v === 'true') return true
      if (v === false || v === 'false') return false
      return erro(`«${ruta}» é unha marca: usa true ou false`)
    }
    case 'problemas':
      return valorParaProblemas(v, ruta)
    case 'imaxe':
      return erro(`«${ruta}» é unha imaxe: non podo poñer imaxes, só o usuario`)
    case 'seccion':
      return erro(`«${ruta}» é un título de sección: non ten valor`)
    default:
      return erro(`«${ruta}»: tipo de campo descoñecido «${c.tipo}»`)
  }
}

// valorParaProblemas: preguntas cun enunciado (Markdown + $LaTeX$ + {variables}),
// puntos e variables fixas ou variables. As solucións non: fanse co resolvedor
// (IA + Maxima, comprobadas) desde a pantalla, e as imaxes só as pon o usuario.
function valorParaProblemas(v, ruta) {
  if (!Array.isArray(v)) erro(`«${ruta}» son problemas: manda unha lista [{enunciado, puntos, variables: [{nome, modo: "fixa"|"variable", valor, de, ata, paso}]}]`)
  return v.map((p, i) => {
    if (!p || typeof p !== 'object' || Array.isArray(p)) erro(`«${ruta}», pregunta ${i + 1}: ten que ser un obxecto`)
    const enunciado = String(p.enunciado ?? '')
    if (enunciado.length > MAX_TEXTO) erro(`«${ruta}», pregunta ${i + 1}: enunciado demasiado longo`)
    const variables = (Array.isArray(p.variables) ? p.variables : []).map((x, j) => {
      const nome = String(x?.nome ?? '').trim()
      if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(nome)) erro(`«${ruta}», pregunta ${i + 1}, variable ${j + 1}: nome «${nome}» non vale (letras, díxitos e _)`)
      if (x.modo === 'variable') return { nome, modo: 'variable', de: String(x.de ?? '1'), ata: String(x.ata ?? '10'), paso: String(x.paso ?? '1') }
      return { nome, modo: 'fixa', valor: String(x.valor ?? '1') }
    })
    return { ...(p.id ? { id: String(p.id) } : {}), enunciado, puntos: String(p.puntos ?? ''), variables, ...(p.solucion && typeof p.solucion === 'object' ? { solucion: p.solucion } : {}) }
  })
}

function valorParaTaboa(c, v) {
  if (!Array.isArray(v)) erro(`«${c.id}» é unha táboa: manda unha lista de filas [{${(c.columnas || []).map((k) => k.id).join(', ')}}]`)
  if (v.length > MAX_FILAS) erro(`«${c.id}»: demasiadas filas (máximo ${MAX_FILAS})`)
  const cols = new Map((c.columnas || []).map((k) => [k.id, k]))
  return v.map((fila, i) => {
    if (!fila || typeof fila !== 'object' || Array.isArray(fila)) erro(`«${c.id}», fila ${i + 1}: ten que ser un obxecto {id_columna: valor}`)
    const out = {}
    for (const [k, x] of Object.entries(fila)) {
      const col = cols.get(k)
      if (!col) erro(`«${c.id}», fila ${i + 1}: non existe a columna «${k}» (as columnas son: ${[...cols.keys()].join(', ')})`)
      const val = valorParaCampo(col, x, `${c.id}.${k} (fila ${i + 1})`)
      // Unha cela baleira non fai falta gardala.
      if (val !== '' && val !== false) out[k] = val
    }
    return out
  })
}

// vistaDatos: os datos en forma lixeira para a IA (as imaxes nunca viaxan: só se di se hai).
// As táboas grandes non caben nunha lectura (TAMANO_MAX): amósanse as primeiras
// FILAS_LECTURA e dise canto falta; con `campo`, `desde` e `cantidade` lense a
// táboa por tramos, con cada fila numerada (`fila`, desde 1) para poder editar
// unha cela ou quitar unha fila sen remandar a táboa enteira.
const FILAS_LECTURA = 40
export function vistaDatos(esquema, plan, { campo = null, desde = 1, cantidade = FILAS_LECTURA } = {}) {
  const campos = esquema.campos.map((c) => {
    const o = { id: c.id, tipo: c.tipo, etiqueta: c.etiqueta }
    if (c.obrigatorio) o.obrigatorio = true
    if (c.opcions?.length) o.opcions = c.opcions
    if (c.tipo === 'taboa') o.columnas = (c.columnas || []).map((k) => ({ id: k.id, tipo: k.tipo, etiqueta: k.etiqueta, ...(k.opcions?.length ? { opcions: k.opcions } : {}) }))
    return o
  })
  const valores = {}
  const taboas = {}
  for (const c of esquema.campos) {
    if (c.tipo === 'seccion') continue
    const v = plan.valores?.[c.id]
    if (c.tipo === 'imaxe') { if (v) valores[c.id] = '[imaxe: non visible]'; continue }
    if (c.tipo === 'problemas' && Array.isArray(v)) { valores[c.id] = v.map(({ imaxe, ...p }) => (imaxe ? { ...p, imaxe: '[imaxe: non visible]' } : p)); continue }
    if (campo && c.id !== campo) continue
    if (c.tipo === 'taboa' && Array.isArray(v)) {
      const ini = Math.max(1, Math.floor(Number(desde)) || 1)
      const n = Math.min(MAX_FILAS, Math.max(1, Math.floor(Number(cantidade)) || FILAS_LECTURA))
      if (campo || v.length > FILAS_LECTURA) {
        const tramo = v.slice(ini - 1, ini - 1 + n).map((fila, i) => ({ fila: ini + i, ...fila }))
        valores[c.id] = tramo
        const ata = ini - 1 + tramo.length
        if (v.length > tramo.length || ini > 1) taboas[c.id] = { filas_total: v.length, amosadas: tramo.length ? `${ini}-${ata}` : 'ningunha', ...(ata < v.length ? { seguinte: `chama a formulario_datos_ler con campo="${c.id}" e desde=${ata + 1}` } : {}) }
        continue
      }
    }
    if (v !== undefined && v !== '') valores[c.id] = v
  }
  const contexto = {}
  for (const k of CLAVES_CONTEXTO) if (String(plan.contexto?.[k] ?? '').trim()) contexto[k] = plan.contexto[k]
  const saida = { formulario: esquema.nome, campos: campo ? campos.filter((c) => c.id === campo) : campos, valores, contexto, logotipo: !!plan.contexto?.logo }
  if (Object.keys(taboas).length) saida.taboas = taboas
  return saida
}

// crearFerramentasFormularioDatos devolve `executar(nome, args) -> {contido, erro, imaxes}`.
//   esquema()          o esquema do formulario (con `modo`)
//   plan()             o documento aberto {formulario, contexto, valores}
//   aplicar(novo)      substitúe o documento (pasa polo historial do controlador: desfacer)
//   api                { IAChamarFerramenta } (para ler os adxuntos)
//   ui                 { avisar(mensaxe, tipo), preguntar(pregunta, opcions) }
export function crearFerramentasFormularioDatos({ esquema, plan, aplicar, api, ui, xerar = null }) {
  const enProduccion = () => esquema().modo === 'produccion'
  // Só ler os adxuntos (documentos de referencia): nin gardalos nin buscar na web.
  const LIBRES = new Set(['adxuntos_listar', 'adxunto_ler', 'adxunto_ver', 'adxunto_extraer_imaxes',
    // Bóveda de Obsidian (só chegan á IA se o documento ten bóveda conectada; ver boveda.go)
    'boveda_info', 'boveda_listar', 'boveda_ler', 'boveda_buscar', 'boveda_enlaces', 'boveda_pendentes', 'boveda_lint', 'boveda_escribir'])
  const chamarGo = async (nome, args) => {
    const r = await api.IAChamarFerramenta(nome, JSON.stringify(args))
    const contido = r.contido ?? ''
    return {
      contido: contido.length <= TAMANO_MAX ? contido : contido.slice(0, TAMANO_MAX) + `\n[... resultado cortado: ${contido.length - TAMANO_MAX} caracteres máis ...]`,
      erro: !!r.erro, imaxes: r.imaxes || [],
    }
  }

  const frontend = {
    formulario_datos_ler(a) {
      const esq = esquema()
      if (a.campo !== undefined && !esq.campos.some((c) => c.id === a.campo && c.tipo !== 'seccion')) erro(`non existe ningún campo co id «${a.campo}»`)
      return vistaDatos(esq, plan(), { campo: a.campo, desde: a.desde, cantidade: a.cantidade })
    },

    formulario_datos_cambiar(a) {
      const dados = ['valores', 'celas', 'quitar_filas', 'contexto', 'limpar'].filter((k) => a[k] !== undefined)
      if (!dados.length) erro('non indicaches nada a cambiar (valores, celas, quitar_filas, contexto, limpar)')
      const desc = Object.keys(a).filter((k) => !['valores', 'celas', 'quitar_filas', 'contexto', 'limpar'].includes(k))
      const esq = esquema()
      const campos = new Map(esq.campos.map((c) => [c.id, c]))
      const actual = plan()
      const limpar = a.limpar === true || a.limpar === 'true'
      let valores = limpar ? {} : { ...actual.valores }
      let contexto = limpar ? { ...actual.contexto, ...Object.fromEntries(CLAVES_CONTEXTO.map((k) => [k, ''])) } : { ...actual.contexto }

      // Primeiro valídase TODO; só se todo vale se aplica.
      const novosValores = {}
      if (a.valores !== undefined) {
        if (!a.valores || typeof a.valores !== 'object' || Array.isArray(a.valores)) erro('valores ten que ser un obxecto {id_do_campo: valor}')
        for (const [id, v] of Object.entries(a.valores)) {
          const c = campos.get(id)
          if (!c) erro(`non existe ningún campo co id «${id}» (os ids son: ${esq.campos.filter((k) => k.tipo !== 'seccion').map((k) => k.id).join(', ')}); le o formulario con formulario_datos_ler`)
          novosValores[id] = c.tipo === 'taboa' ? valorParaTaboa(c, v ?? []) : valorParaCampo(c, v)
        }
      }
      const novoContexto = {}
      if (a.contexto !== undefined) {
        if (!a.contexto || typeof a.contexto !== 'object' || Array.isArray(a.contexto)) erro('contexto ten que ser un obxecto {centro, localidade, curso, departamento, responsable, cargo}')
        for (const [k, v] of Object.entries(a.contexto)) {
          if (k === 'logo') erro('o logotipo non se pode cambiar desde o asistente')
          if (!CLAVES_CONTEXTO.includes(k)) erro(`o dato do documento «${k}» non existe (os posibles son: ${CLAVES_CONTEXTO.join(', ')})`)
          novoContexto[k] = v === null || v === undefined ? '' : String(v).slice(0, 500)
        }
      }
      valores = { ...valores, ...novosValores }

      // Celas soltas e filas a quitar: sobre o estado que queda tras 'valores'. Os
      // números de fila (desde 1) refírense a esa táboa; quitar_filas aplícase o último.
      const taboaDe = (id) => {
        const c = campos.get(id)
        if (!c || c.tipo !== 'taboa') erro(`«${id}» non é unha táboa (as táboas son: ${esq.campos.filter((k) => k.tipo === 'taboa').map((k) => k.id).join(', ') || 'ningunha'})`)
        return c
      }
      const filaValida = (id, filas, n) => {
        if (!Number.isInteger(n) || n < 1 || n > filas.length) erro(`«${id}»: a fila ${n} non existe (a táboa ten ${filas.length} fila(s), numeradas desde 1)`)
      }
      const copias = {}
      const filasDe = (id) => (copias[id] ??= (Array.isArray(valores[id]) ? valores[id] : []).map((f) => ({ ...f })))
      if (a.celas !== undefined) {
        if (!Array.isArray(a.celas)) erro('celas ten que ser unha lista [{campo, fila, columna, valor}]')
        a.celas.forEach((x, i) => {
          if (!x || typeof x !== 'object') erro(`celas[${i}]: ten que ser un obxecto {campo, fila, columna, valor}`)
          const c = taboaDe(x.campo)
          const filas = filasDe(c.id)
          filaValida(c.id, filas, Number(x.fila))
          const col = (c.columnas || []).find((k) => k.id === x.columna)
          if (!col) erro(`«${c.id}»: non existe a columna «${x.columna}» (as columnas son: ${(c.columnas || []).map((k) => k.id).join(', ')})`)
          const val = valorParaCampo(col, x.valor, `${c.id}.${col.id} (fila ${x.fila})`)
          const fila = filas[Number(x.fila) - 1]
          if (val === '' || val === false) delete fila[col.id]
          else fila[col.id] = val
        })
      }
      if (a.quitar_filas !== undefined) {
        if (!a.quitar_filas || typeof a.quitar_filas !== 'object' || Array.isArray(a.quitar_filas)) erro('quitar_filas ten que ser un obxecto {id_da_taboa: [números de fila]}')
        for (const [id, nums] of Object.entries(a.quitar_filas)) {
          const c = taboaDe(id)
          const filas = filasDe(c.id)
          if (!Array.isArray(nums)) erro(`quitar_filas.${id} ten que ser unha lista de números de fila`)
          nums.forEach((n) => filaValida(c.id, filas, Number(n)))
          const fora = new Set(nums.map(Number))
          copias[c.id] = filas.filter((_, i) => !fora.has(i + 1))
        }
      }
      for (const [id, filas] of Object.entries(copias)) valores[id] = filas
      const tocados = [...new Set([...Object.keys(novosValores), ...Object.keys(copias)])]
      contexto = { ...contexto, ...novoContexto }
      // O documento pasa a levar datos de proba postos pola IA (en produción ofrécese borralos);
      // se só se baleirou, xa non os leva.
      const puxoDatos = tocados.length + Object.keys(novoContexto).length > 0
      const { datosProba: _antes, ...resto } = actual
      const datosProba = puxoDatos || (!limpar && !!actual.datosProba)
      aplicar({ ...resto, valores, contexto, ...(datosProba ? { datosProba: true } : {}) })
      const saida = { limpado: limpar, campos: tocados, contexto: Object.keys(novoContexto) }
      if (desc.length) saida.ignorados = desc
      return saida
    },

    // Compila o documento coma o botón Xerar; os detalles dun erro quedan no
    // rexistro interno (rexistro_listar / rexistro_ler).
    async formulario_xerar() {
      if (!xerar) erro('non se pode xerar desde aquí')
      const r = await xerar()
      const saida = { ok: !!r.ok, paxinas: r.paxinas || 0, avisos: r.avisos || [] }
      if (!r.ok) {
        saida.erro = r.erro || 'non se puido xerar'
        saida.seguinte = 'le o detalle con rexistro_ler (sen parte: o erro de LaTeX e o .tex arredor)'
      }
      return saida
    },

    usuario_avisar(a) {
      const m = String(a.mensaxe ?? '').trim()
      if (!m) erro('falta a mensaxe')
      ui.avisar(m, a.tipo === 'aviso' ? 'aviso' : 'info')
      return { amosado: true }
    },

    async usuario_preguntar(a) {
      const pregunta = String(a.pregunta ?? '').trim()
      if (!pregunta) erro('falta a pregunta')
      const opcions = Array.isArray(a.opcions) ? a.opcions.map(String).filter(Boolean) : []
      const r = await ui.preguntar(pregunta, opcions)
      return r === null || r === undefined ? { resposta: null, nota: 'o usuario non respondeu' } : { resposta: r }
    },
  }

  async function executar(nome, args) {
    if (enProduccion()) {
      return { contido: 'este formulario está en modo PRODUCIÓN: o asistente non ten acceso aos seus datos. Só o usuario pode pasalo a deseño.', erro: true }
    }
    try {
      if (frontend[nome]) {
        const v = await frontend[nome](args || {})
        const s = typeof v === 'string' ? v : JSON.stringify(v, null, 2)
        return { contido: s.length <= TAMANO_MAX ? s : s.slice(0, TAMANO_MAX) + '\n[... cortado ...]', erro: false }
      }
      if (LIBRES.has(nome)) return await chamarGo(nome, args || {})
      return { contido: `a ferramenta «${nome}» non está dispoñible no asistente dos datos do formulario`, erro: true }
    } catch (err) {
      return { contido: String(err?.message || err), erro: true }
    }
  }
  return { executar, nomesFrontend: Object.keys(frontend) }
}
