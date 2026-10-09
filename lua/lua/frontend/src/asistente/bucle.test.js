import { describe, it, expect, vi } from 'vitest'
import { executarConversa, retomar, MAX_PASOS } from './bucle.js'

const DEFS = [
  { nome: 'ler', tipo: 'lectura', onde: 'frontend' },
  { nome: 'editar', tipo: 'escritura', onde: 'frontend' },
  { nome: 'perigo', tipo: 'confirmar', onde: 'backend' },
]

// api falsa: devolve os turnos gardados, en orde; garda o que recibe.
function apiCon(turnos) {
  const recibidos = []
  return {
    recibidos,
    IATurnoAsistente: vi.fn(async (req) => {
      recibidos.push(JSON.parse(JSON.stringify(req.mensaxes)))
      const t = turnos.shift()
      if (t instanceof Error) throw t
      return t
    }),
  }
}
const chamada = (id, nome, args = {}) => ({ id, nome, args: JSON.stringify(args) })

function correr(turnos, extra = {}) {
  const historial = extra.historial || []
  const eventos = []
  const api = apiCon(turnos)
  const executar = extra.executar || vi.fn(async (n) => ({ contido: `res-${n}`, erro: false }))
  const p = executarConversa({
    api, executar, ferramentas: DEFS, historial, texto: extra.texto ?? 'ola', adxuntos: extra.adxuntos, estado: () => 'ESTADO',
    aoEvento: (e) => eventos.push(e), confirmar: extra.confirmar, confirmarTodo: extra.confirmarTodo, sinal: extra.sinal, maxPasos: extra.maxPasos,
  })
  return { p, api, executar, historial, eventos }
}

describe('bucle do asistente', () => {
  it('sen ferramentas: un turno, a resposta queda no historial', async () => {
    const { p, historial, eventos, api } = correr([{ texto: 'Ola!', chamadas: [], bruto: '{"b":1}', provedor: 'openai' }])
    const r = await p
    expect(r).toMatchObject({ pasos: 1, interrompido: false, agotado: false })
    expect(historial.map((m) => m.rol)).toEqual(['user', 'assistant'])
    expect(historial[1]).toMatchObject({ texto: 'Ola!', bruto: '{"b":1}', provedor: 'openai' })
    expect(eventos.map((e) => e.tipo)).toEqual(['pensando', 'texto'])
    expect(api.recibidos[0]).toHaveLength(1)
  })

  it('executa as ferramentas e devolve os resultados na mesma orde, ata que responde só texto', async () => {
    const { p, historial, executar, api } = correr([
      { texto: 'Miro', chamadas: [chamada('c1', 'ler'), chamada('c2', 'editar', { id: 'a' })], bruto: 'B1', provedor: 'openai' },
      { texto: 'Feito', chamadas: [], bruto: 'B2', provedor: 'openai' },
    ])
    const r = await p
    expect(r.pasos).toBe(2)
    expect(executar).toHaveBeenNthCalledWith(1, 'ler', {})
    expect(executar).toHaveBeenNthCalledWith(2, 'editar', { id: 'a' })
    expect(historial.map((m) => m.rol)).toEqual(['user', 'assistant', 'tool', 'assistant'])
    expect(historial[2].resultados).toEqual([
      { chamadaId: 'c1', nome: 'ler', contido: 'res-ler', erro: false, imaxes: [] },
      { chamadaId: 'c2', nome: 'editar', contido: 'res-editar', erro: false, imaxes: [] },
    ])
    // No segundo turno a IA recibe a conversa completa, co bruto do primeiro.
    expect(api.recibidos[1].map((m) => m.rol)).toEqual(['user', 'assistant', 'tool'])
    expect(api.recibidos[1][1].bruto).toBe('B1')
  })

  it('unha ferramenta descoñecida ou uns argumentos rotos dan erro á IA sen romper o bucle', async () => {
    const { p, historial, executar } = correr([
      { texto: '', chamadas: [chamada('c1', 'non_existe'), { id: 'c2', nome: 'ler', args: '{roto' }, { id: 'c3', nome: 'ler', args: '[1]' }], bruto: 'B', provedor: 'x' },
      { texto: 'ok', chamadas: [] },
    ])
    await p
    expect(executar).not.toHaveBeenCalled()
    const rs = historial[2].resultados
    expect(rs.map((r) => r.erro)).toEqual([true, true, true])
    expect(rs[0].contido).toMatch(/descoñecida/)
    expect(rs[1].contido).toMatch(/JSON/)
    expect(rs[2].contido).toMatch(/obxecto/)
  })

  it('as ferramentas "confirmar" piden permiso; se se rexeita non se executan e a IA sábeo', async () => {
    const confirmar = vi.fn(async () => false)
    const { p, historial, executar } = correr([
      { texto: '', chamadas: [chamada('c1', 'perigo', { id: 'x' })], bruto: 'B', provedor: 'x' },
      { texto: 'vale', chamadas: [] },
    ], { confirmar })
    await p
    expect(confirmar).toHaveBeenCalledWith(DEFS[2], { id: 'x' })
    expect(executar).not.toHaveBeenCalled()
    expect(historial[2].resultados[0]).toMatchObject({ erro: true, contido: expect.stringMatching(/rexeitou/) })
  })

  it('se se permite, execútase; as de lectura nunca piden permiso; as de escritura só co modo "confirmar todo"', async () => {
    const confirmar = vi.fn(async () => true)
    const turnos = () => [{ texto: '', chamadas: [chamada('c1', 'ler'), chamada('c2', 'editar'), chamada('c3', 'perigo')], bruto: 'B' }, { texto: 'ok', chamadas: [] }]
    let r = correr(turnos(), { confirmar })
    await r.p
    expect(confirmar).toHaveBeenCalledTimes(1) // só a perigosa
    expect(r.executar).toHaveBeenCalledTimes(3)

    confirmar.mockClear()
    r = correr(turnos(), { confirmar, confirmarTodo: () => true })
    await r.p
    expect(confirmar).toHaveBeenCalledTimes(2) // editar + perigo, non ler
  })

  it('unha ferramenta que devolve erro chega á IA coma erro', async () => {
    const executar = vi.fn(async () => ({ contido: 'mes fóra de rango', erro: true }))
    const { p, historial } = correr([{ texto: '', chamadas: [chamada('c1', 'editar')], bruto: 'B' }, { texto: 'corrixo', chamadas: [] }], { executar })
    await p
    expect(historial[2].resultados[0]).toEqual({ chamadaId: 'c1', nome: 'editar', contido: 'mes fóra de rango', erro: true, imaxes: [] })
  })

  it('interromper deixa o historial consistente: cada chamada pendente ten o seu resultado', async () => {
    const sinal = { abortado: false }
    const executar = vi.fn(async () => { sinal.abortado = true; return { contido: 'feita', erro: false } })
    const { p, historial } = correr([
      { texto: '', chamadas: [chamada('c1', 'ler'), chamada('c2', 'ler'), chamada('c3', 'ler')], bruto: 'B' },
      { texto: 'nunca chega', chamadas: [] },
    ], { executar, sinal })
    const r = await p
    expect(r.interrompido).toBe(true)
    expect(executar).toHaveBeenCalledTimes(1)
    expect(historial.map((m) => m.rol)).toEqual(['user', 'assistant', 'tool'])
    expect(historial[2].resultados.map((x) => x.chamadaId)).toEqual(['c1', 'c2', 'c3'])
    expect(historial[2].resultados[1].contido).toMatch(/Interrompido/)
  })

  it('para tras MAX_PASOS turnos con ferramentas e avisa', async () => {
    const turnos = Array.from({ length: 40 }, (_, i) => ({ texto: '', chamadas: [chamada('c' + i, 'ler')], bruto: 'B' }))
    const { p, eventos, api } = correr(turnos, { maxPasos: 4 })
    const r = await p
    expect(r).toMatchObject({ pasos: 4, agotado: true })
    expect(api.IATurnoAsistente).toHaveBeenCalledTimes(4)
    expect(eventos.at(-1)).toMatchObject({ tipo: 'aviso', texto: expect.stringMatching(/parou/) })
    expect(MAX_PASOS).toBeGreaterThanOrEqual(10)
  })

  it('avisa se a IA non responde nada ou se corta polo límite de tokens', async () => {
    let r = correr([{ texto: '  ', chamadas: [] }])
    await r.p
    expect(r.eventos.some((e) => e.tipo === 'aviso' && /non devolveu/.test(e.texto))).toBe(true)
    r = correr([{ texto: 'a medias', chamadas: [], truncado: true }])
    await r.p
    expect(r.eventos.some((e) => e.tipo === 'aviso' && /límite de tokens/.test(e.texto))).toBe(true)
  })

  it('un erro da IA sobe, pero o historial queda ben para reintentar con retomar()', async () => {
    const historial = []
    const api = apiCon([new Error('sen rede'), { texto: 'agora si', chamadas: [] }])
    const comun = { api, executar: vi.fn(), ferramentas: DEFS, historial, aoEvento: () => {} }
    await expect(executarConversa({ ...comun, texto: 'hola' })).rejects.toThrow('sen rede')
    expect(historial.map((m) => m.rol)).toEqual(['user'])
    const r = await retomar(comun)
    expect(r.pasos).toBe(1)
    expect(historial.map((m) => m.rol)).toEqual(['user', 'assistant']) // sen repetir a mensaxe do usuario
  })

  it('envía o estado do programa en cada turno e non engade unha mensaxe baleira', async () => {
    const { p, api } = correr([{ texto: 'ok', chamadas: [] }], { texto: '   ' })
    await p
    expect(api.IATurnoAsistente.mock.calls[0][0].estado).toBe('ESTADO')
    expect(api.recibidos[0]).toEqual([])
  })

  it('os adxuntos van na mensaxe do usuario, tamén sen texto', async () => {
    const { p, historial, api } = correr([{ texto: 'vexo o adxunto', chamadas: [] }], { texto: '', adxuntos: ['a1', 'a2'] })
    await p
    expect(historial[0]).toMatchObject({ rol: 'user', texto: '', adxuntos: ['a1', 'a2'] })
    expect(api.recibidos[0][0].adxuntos).toEqual(['a1', 'a2'])
    // Sen texto nin adxuntos non se engade nada.
    const v = correr([{ texto: 'ok', chamadas: [] }], { texto: ' ', adxuntos: [] })
    await v.p
    expect(v.api.recibidos[0]).toEqual([])
  })

  it('as imaxes que devolve unha ferramenta viaxan no resultado (ids de adxuntos)', async () => {
    const executar = vi.fn(async () => ({ contido: 'páxina 1', erro: false, imaxes: ['a77'] }))
    const { p, historial } = correr([{ texto: '', chamadas: [chamada('c1', 'ler')], bruto: 'B' }, { texto: 'vexo', chamadas: [] }], { executar })
    await p
    expect(historial[2].resultados[0]).toEqual({ chamadaId: 'c1', nome: 'ler', contido: 'páxina 1', erro: false, imaxes: ['a77'] })
  })
})
