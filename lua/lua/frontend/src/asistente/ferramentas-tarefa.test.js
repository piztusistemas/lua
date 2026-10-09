import { describe, it, expect, vi } from 'vitest'
import { crearFerramentasTarefa } from './ferramentas-tarefa.js'

function montar({ bloqueado = false, desc = 'Texto\n\n![foto](img:i1 =60%)' } = {}) {
  const estado = { titulo: 'Avaliacion', descricion: desc, cor: 'blue', mes: '2025-09', nova: false, cambiosSenAplicar: false, imaxes: ['i1'] }
  const editor = {
    bloqueado,
    ler: () => ({ ...estado }),
    cambiar: vi.fn((c) => { if (c.titulo !== undefined) estado.titulo = c.titulo; if (c.descricion !== undefined) estado.descricion = c.descricion; if (c.cor !== undefined) estado.cor = c.cor }),
  }
  const api = { IAChamarFerramenta: vi.fn(async (nome, args) => ({ contido: `web ${nome} ${args}`, erro: false })) }
  const ui = { avisar: vi.fn(), preguntar: vi.fn(async () => 'Setembro') }
  return { ...crearFerramentasTarefa({ editor, api, ui }), editor, api, ui, estado }
}
const json = (r) => JSON.parse(r.contido)

describe('ferramentas do editor da tarefa', () => {
  it('editor_ler devolve o borrador', async () => {
    const { executar } = montar()
    const r = await executar('editor_ler', {})
    expect(r.erro).toBe(false)
    expect(json(r)).toMatchObject({ titulo: 'Avaliacion', cor: 'blue', mes: '2025-09', imaxes: ['i1'] })
  })
  it('editor_cambiar cambia só o indicado', async () => {
    const { executar, editor } = montar()
    const r = await executar('editor_cambiar', { titulo: '  Avaliación  ', cor: 'rose', descricion: 'Novo\n\n![foto](img:i1 =60%)' })
    expect(r.erro).toBe(false)
    expect(json(r)).toEqual({ cambiados: ['titulo', 'descricion', 'cor'] })
    expect(editor.cambiar).toHaveBeenCalledWith({ titulo: 'Avaliación', descricion: 'Novo\n\n![foto](img:i1 =60%)', cor: 'rose' })
  })
  it('valida: sen campos, título baleiro, cor mala, curso bloqueado', async () => {
    const { executar, editor } = montar()
    expect((await executar('editor_cambiar', {})).erro).toBe(true)
    expect((await executar('editor_cambiar', { titulo: '   ' })).contido).toMatch(/título/)
    expect((await executar('editor_cambiar', { cor: 'lila' })).contido).toMatch(/cor non válida/)
    expect(editor.cambiar).not.toHaveBeenCalled()
    const b = montar({ bloqueado: true })
    expect((await b.executar('editor_cambiar', { titulo: 'x' })).contido).toMatch(/bloqueado/)
    expect(b.editor.cambiar).not.toHaveBeenCalled()
  })
  it('avisa se a nova descrición perde unha imaxe', async () => {
    const { executar } = montar()
    const r = await executar('editor_cambiar', { descricion: 'Sen imaxe' })
    expect(json(r).avisos[0]).toMatch(/imaxe/)
  })
  it('usuario_avisar e usuario_preguntar', async () => {
    const { executar, ui } = montar()
    await executar('usuario_avisar', { mensaxe: 'Ollo', tipo: 'aviso' })
    expect(ui.avisar).toHaveBeenCalledWith('Ollo', 'aviso')
    expect(json(await executar('usuario_preguntar', { pregunta: 'Mes?', opcions: ['Setembro'] }))).toEqual({ resposta: 'Setembro' })
  })
})

describe('a web e nada máis do backend', () => {
  it('web_buscar e web_consultar van a Go', async () => {
    const { executar, api } = montar()
    const r = await executar('web_buscar', { consulta: 'site:boe.es orientación' })
    expect(r.erro).toBe(false)
    expect(api.IAChamarFerramenta).toHaveBeenCalledWith('web_buscar', '{"consulta":"site:boe.es orientación"}')
    await executar('web_consultar', { url: 'https://www.boe.es/x' })
    expect(api.IAChamarFerramenta).toHaveBeenCalledTimes(2)
  })
  it('calquera outra ferramenta (curso, plantillas, Opcións...) está bloqueada', async () => {
    const { executar, api } = montar()
    for (const n of ['curso_ler', 'tarefa_eliminar', 'plantilla_gardar', 'opcions_cambiar', 'informe_exportar', 'curso_novo']) {
      const r = await executar(n, {})
      expect(r.erro, n).toBe(true)
      expect(r.contido).toMatch(/non está dispoñible/)
    }
    expect(api.IAChamarFerramenta).not.toHaveBeenCalled()
  })
  it('un fallo da web chega á IA coma erro', async () => {
    const { executar, api } = montar()
    api.IAChamarFerramenta.mockRejectedValueOnce(new Error('sen rede'))
    expect(await executar('web_consultar', { url: 'https://x' })).toEqual({ contido: 'sen rede', erro: true })
  })
})
