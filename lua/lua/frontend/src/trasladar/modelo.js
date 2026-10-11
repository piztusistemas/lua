// Lóxica pura da xanela TRASLADAR (spec trasladar): que se preselecciona ao
// sacar, que selección se lle pasa a Go e como se le o resumo de «Meter».
// Sen DOM nin bindings, para probala en modelo.test.js.

// opcionsSacar: as opcións de «Sacar» segundo o que hai aberto.
//   contexto: { proxecto: {id, nome} | null, documento: {ferramenta, nome} | null }
export function opcionsSacar(contexto) {
  const out = []
  if (contexto.proxecto) out.push({ id: 'proxecto', nome: contexto.proxecto.nome })
  if (contexto.documento) out.push({ id: 'documento', nome: contexto.documento.nome })
  out.push({ id: 'todo' }, { id: 'escoller' })
  return out
}

// opcionInicial: o que vén marcado (o máis concreto do que está aberto).
export function opcionInicial(contexto, pedida) {
  const ids = opcionsSacar(contexto).map((o) => o.id)
  if (pedida && ids.includes(pedida)) return pedida
  return ids[0]
}

// ofreceDatos: a opción ofrece a marca «Incluír os datos escritos».
export function ofreceDatos(opcion) {
  return opcion === 'proxecto' || opcion === 'escoller'
}

// seleccionSacar arma a SeleccionSacar de Go. `marcados` é a árbore escollida:
// { proxectos: Set, formularios: Set, plantillas: Set }.
export function seleccionSacar(opcion, contexto, marcados, datos) {
  switch (opcion) {
    case 'proxecto':
      return { tipo: 'proxecto', proxectos: [contexto.proxecto.id], formularios: [], plantillas: [], datos: Boolean(datos) }
    case 'todo':
      return { tipo: 'todo', proxectos: [], formularios: [], plantillas: [], datos: true }
    case 'escoller':
      return {
        tipo: 'seleccion',
        proxectos: [...(marcados?.proxectos || [])],
        formularios: [...(marcados?.formularios || [])],
        plantillas: [...(marcados?.plantillas || [])],
        datos: Boolean(datos),
      }
  }
  return null
}

// seleccionBaleira: non hai nada marcado na árbore.
export function seleccionBaleira(sel) {
  return !sel || (sel.proxectos.length + sel.formularios.length + sel.plantillas.length === 0)
}

// textoElemento: unha liña do resumo de «Meter».
export function textoElemento(e, t) {
  switch (e.tipo) {
    case 'proxecto': {
      const [forms, pdfs] = String(e.detalle || '0/0').split('/').map((n) => parseInt(n, 10) || 0)
      return t('trasladar.el.proxecto', 'Proxecto «{nome}»: {f} formularios, {p} PDF na bóveda', { nome: e.nome, f: forms, p: pdfs })
    }
    case 'formulario':
      return t('trasladar.el.formulario', 'Formulario «{nome}»', { nome: e.nome })
    case 'plantilla':
      return t('trasladar.el.plantilla', 'Plantilla «{nome}»', { nome: e.nome })
    case 'boveda': {
      const [pdfs, mds] = String(e.detalle || '0/0').split('/').map((n) => parseInt(n, 10) || 0)
      return t('trasladar.el.boveda', 'Fontes para a bóveda: {p} PDF e {m} Markdown', { p: pdfs, m: mds })
    }
    case 'documento':
      return t('trasladar.el.documento', 'Documento «{nome}»', { nome: e.nome })
  }
  return e.nome
}

// opcionsDestino: entre que destinos pode escoller o usuario (baleiro = ningún:
// o destino é fixo).
export function opcionsDestino(resumo) {
  if (!resumo) return []
  if (['documento', 'curso', 'boveda'].includes(resumo.tipo) && !resumo.volta && resumo.podeAberto && (resumo.cursos || []).length <= 1) {
    return ['aberto', 'novo']
  }
  return []
}

// textoDestino: onde vai ir o que se mete.
export function textoDestino(resumo, escolla, t) {
  const destino = escolla.destino || resumo.destino
  const nome = resumo.nomeProxectoAberto || ''
  if (resumo.volta && escolla.accion !== 'copia') {
    return t('trasladar.destino.volta', 'Actualiza «{doc}» do proxecto «{nome}». A versión anterior gárdase e podes desfacer.', { doc: resumo.nome, nome: resumo.volta.nomeProxecto })
  }
  switch (destino) {
    case 'aberto': return t('trasladar.destino.aberto', 'Vai ao proxecto aberto, «{nome}».', { nome })
    case 'novo':
    case 'volta': return t('trasladar.destino.novo', 'Lúa crea un proxecto novo para el. Non se toca nada do que xa tes.')
    case 'biblioteca': return t('trasladar.destino.biblioteca', 'Vai á biblioteca. Non se toca nada do que xa tes.')
    case 'boveda': return t('trasladar.destino.boveda', 'Vai á bóveda do proxecto aberto, «{nome}». Os nomes repetidos entran coma copia.', { nome })
    case 'restaurar': return t('trasladar.destino.restaurar', 'Restaura TODO Lúa tal como estaba na copia. O estado actual gárdase antes e podes desfacer.')
  }
  return ''
}

// confirmacionPrevia: a única pregunta que pode haber antes de meter.
export function confirmacionPrevia(resumo) {
  if (!resumo) return null
  if (resumo.tipo === 'copia') return 'restaurar'
  if (resumo.produccion) return 'produccion'
  return null
}

// opcionsMeter: as OpcionsMeter de Go a partir da escolla da xanela.
export function opcionsMeter(resumo, escolla) {
  const op = { destino: '', cursos: null, accion: '' }
  if (opcionsDestino(resumo).includes(escolla.destino)) op.destino = escolla.destino
  if ((resumo.cursos || []).length > 1) op.cursos = [...(escolla.cursos || [])].sort((a, b) => a - b)
  if (resumo.volta) op.accion = escolla.accion === 'copia' ? 'copia' : 'actualizar'
  return op
}

// podeMeter: hai algo que meter coa escolla actual.
export function podeMeter(resumo, escolla) {
  if (!resumo) return false
  if ((resumo.cursos || []).length > 1) return (escolla.cursos || []).length > 0
  return true
}
