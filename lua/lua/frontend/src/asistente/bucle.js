// O bucle do asistente: pídelle un turno á IA (IATurnoAsistente, Go), executa
// as ferramentas que pida, devólvelle os resultados... ata que a IA responde
// só con texto. Non toca o DOM: informa do que pasa con `aoEvento` e pídelle
// permiso ao usuario con `confirmar`, así se pode probar (bucle.test.js) e o
// panel só se ocupa de pintar.
//
// A conversa (`historial`) é a que se lle manda a Go en cada turno. Debe
// quedar SEMPRE consistente: unha mensaxe da IA con chamadas a ferramentas vai
// seguida dunha mensaxe "tool" cun resultado por cada chamada, aínda que o
// usuario interrompa ou unha ferramenta falle; senón os provedores rexeitan
// a conversa enteira no turno seguinte.

export const MAX_PASOS = 25

const RECHAZADA = 'O usuario rexeitou esta acción. Non a repitas; pregúntalle que prefire.'
const INTERROMPIDA = 'Interrompido polo usuario antes de executarse.'

// parsearArgs le os argumentos JSON dunha chamada (a IA ás veces manda texto roto).
function parsearArgs(bruto) {
  if (bruto === undefined || bruto === null || String(bruto).trim() === '') return { args: {} }
  try {
    const v = JSON.parse(bruto)
    if (v && typeof v === 'object' && !Array.isArray(v)) return { args: v }
    return { erro: 'os argumentos ten que ser un obxecto JSON' }
  } catch (e) {
    return { erro: `os argumentos non son JSON válido: ${e.message}` }
  }
}

// executarConversa: engade `texto` do usuario ao historial e corre o bucle.
//   api          { IATurnoAsistente }
//   executar     (nome, args) -> Promise<{contido, erro}>
//   ferramentas  catálogo (IAFerramentas) para saber o tipo de cada unha
//   historial    array mutable de mensaxes (ver ia_conversa.go, IAMensaxe)
//   adxuntos     ids dos ficheiros que o usuario xunta a esta mensaxe (adxuntos.go)
//   estado       () -> string co resumo do programa (vai no prompt do sistema)
//   aoEvento     (evento) para pintar: pensando | texto | ferramenta | resultado | aviso
//   confirmar    async (def, args) -> bool; só se chama para ferramentas que o precisan
//   confirmarTodo  () -> bool: pedir confirmación tamén nas de escritura
//   sinal        { abortado: bool }
// Devolve { pasos, interrompido, agotado }. Se a chamada á IA falla, lanza o
// erro (o historial queda consistente e pódese reintentar con retomar()).
export async function executarConversa({
  api, executar, ferramentas, historial, texto, adxuntos = [], estado = () => '', aoEvento = () => {},
  confirmar = async () => true, confirmarTodo = () => false, sinal = { abortado: false }, maxPasos = MAX_PASOS,
}) {
  const haiTexto = texto !== undefined && texto !== null && String(texto).trim() !== ''
  if (haiTexto || adxuntos.length) {
    historial.push({ rol: 'user', texto: haiTexto ? String(texto) : '', chamadas: [], resultados: [], bruto: '', provedor: '', adxuntos: [...adxuntos] })
  }
  return retomar({ api, executar, ferramentas, historial, estado, aoEvento, confirmar, confirmarTodo, sinal, maxPasos })
}

// retomar continúa o bucle co historial tal como está (tamén para reintentar
// despois dun erro de rede sen repetir a mensaxe do usuario).
export async function retomar({
  api, executar, ferramentas, historial, estado = () => '', aoEvento = () => {},
  confirmar = async () => true, confirmarTodo = () => false, sinal = { abortado: false }, maxPasos = MAX_PASOS,
}) {
  const defs = new Map((ferramentas || []).map((f) => [f.nome, f]))
  let pasos = 0
  while (pasos < maxPasos) {
    if (sinal.abortado) return { pasos, interrompido: true, agotado: false }
    pasos++
    aoEvento({ tipo: 'pensando' })
    const turno = await api.IATurnoAsistente({ mensaxes: historial, estado: estado() })
    const chamadas = turno.chamadas || []
    historial.push({
      rol: 'assistant', texto: turno.texto || '', chamadas, resultados: [],
      bruto: turno.bruto || '', provedor: turno.provedor || '', adxuntos: [],
    })
    if (turno.texto && turno.texto.trim()) aoEvento({ tipo: 'texto', texto: turno.texto })
    if (turno.truncado) aoEvento({ tipo: 'aviso', texto: 'A resposta cortouse polo límite de tokens (Opcións → IA). Podes pedirlle que continúe.' })

    if (chamadas.length === 0) {
      if (!(turno.texto && turno.texto.trim()) && !turno.truncado) aoEvento({ tipo: 'aviso', texto: 'A IA non devolveu ningunha resposta. Proba de novo ou reformula a petición.' })
      return { pasos, interrompido: false, agotado: false }
    }

    const resultados = []
    for (const chamada of chamadas) {
      const def = defs.get(chamada.nome)
      const { args, erro: erroArgs } = parsearArgs(chamada.args)
      aoEvento({ tipo: 'ferramenta', chamada, def, args })
      let r
      if (sinal.abortado) {
        r = { contido: INTERROMPIDA, erro: true }
      } else if (!def) {
        r = { contido: `ferramenta descoñecida: ${chamada.nome}`, erro: true }
      } else if (erroArgs) {
        r = { contido: erroArgs, erro: true }
      } else {
        const precisaPermiso = def.tipo === 'confirmar' || (def.tipo === 'escritura' && confirmarTodo())
        let permitido = true
        if (precisaPermiso) permitido = await confirmar(def, args)
        if (!permitido) r = { contido: RECHAZADA, erro: true, rexeitada: true }
        else r = await executar(chamada.nome, args)
      }
      resultados.push({ chamadaId: chamada.id, nome: chamada.nome, contido: r.contido ?? '', erro: !!r.erro, imaxes: r.imaxes || [] })
      aoEvento({ tipo: 'resultado', chamada, def, args, resultado: r })
    }
    historial.push({ rol: 'tool', texto: '', chamadas: [], resultados, bruto: '', provedor: '', adxuntos: [] })
  }
  aoEvento({ tipo: 'aviso', texto: `O asistente parou tras ${maxPasos} pasos seguidos. Podes pedirlle que continúe.` })
  return { pasos, interrompido: false, agotado: true }
}
