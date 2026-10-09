import { describe, it, expect, vi } from 'vitest'
import { crearFerramentasPlantilla, validarOfimatica } from './ferramentas-plantilla.js'

function montar(extra = {}) {
  const estado = { imaxes: ['logo.png'], cambios: [] }
  const editor = {
    ler: vi.fn(() => ({ id: 'p1', nome: 'Centro', imaxes: estado.imaxes })),
    borrador: vi.fn(() => ({ id: 'p1', nome: 'Centro', latex: 'X {{CORPO}}', ofimatica: { fonte: 'Georgia' }, variables: [] })),
    cambiar: vi.fn((c) => estado.cambios.push(c)),
    id: vi.fn(() => 'p1'),
    garantirGardada: vi.fn(async () => true),
    recargarImaxes: vi.fn(async () => {}),
    ...extra,
  }
  const api = { IAChamarFerramenta: vi.fn(async (nome) => ({ contido: `ok ${nome}`, erro: false, imaxes: ['a1'] })) }
  const ui = { avisar: vi.fn(), preguntar: vi.fn(async () => 'Georgia') }
  return { ...crearFerramentasPlantilla({ editor, api, ui }), editor, api, ui, estado }
}

describe('validarOfimatica', () => {
  it('normaliza cores (#, 3 díxitos, minúsculas) e acepta números en rango', () => {
    const { valores } = validarOfimatica({ corPrincipal: '#abc', corTexto: '1f3864', interlinado: 1.5, marxeEsquerda: 3, tamanoCorpo: 12, fonte: 'Calibri' })
    expect(valores).toEqual({ corPrincipal: 'AABBCC', corTexto: '1F3864', interlinado: 1.5, marxeEsquerda: 3, tamanoCorpo: 12, fonte: 'Calibri' })
  })
  it('0 e cor baleira valen (= por defecto); fóra de rango ou cor mala dan un erro explicado', () => {
    expect(validarOfimatica({ interlinado: 0, corLinha: '' }).valores).toEqual({ interlinado: 0, corLinha: '' })
    expect(() => validarOfimatica({ interlinado: 5 })).toThrow(/interlinado.*fóra de rango/)
    expect(() => validarOfimatica({ marxeSuperior: 20 })).toThrow(/marxeSuperior.*0.5-6 cm/)
    expect(() => validarOfimatica({ corPrincipal: 'azul' })).toThrow(/cor non válida/)
    expect(() => validarOfimatica({ tamanoCorpo: 'grande' })).toThrow(/número/)
  })
  it('os campos que non existen devólvense aparte', () => {
    expect(validarOfimatica({ fonte: 'A', invento: 1 }).descoñecidos).toEqual(['invento'])
  })
})

describe('estilo e textos do documento', () => {
  const editorCoaLista = () => ({
    ler: vi.fn(() => ({
      id: 'p1', nome: 'Centro', imaxes: [], textos: { SEN_TAREFAS: 'Libre', INDICE: 'Contido' },
      textosDispoñibles: [{ clave: 'INDICE' }, { clave: 'SEN_TAREFAS' }, { clave: 'ACTA_ORDE' }],
    })),
  })
  it('cambia o estilo (LaTeX) do documento', async () => {
    const f = montar()
    const r = await f.executar('plantilla_editor_cambiar', { estilo: '% meu estilo\n\\newcommand{\\chip}[2]{#2}' })
    expect(r.erro).toBe(false)
    expect(f.estado.cambios[0]).toEqual({ estilo: '% meu estilo\n\\newcommand{\\chip}[2]{#2}' })
  })
  it('só cambia os textos indicados: os demais consérvanse e un baleiro devolve o de Lua', async () => {
    const f = montar(editorCoaLista())
    const r = await f.executar('plantilla_editor_cambiar', { textos: { acta_orde: 'Puntos', SEN_TAREFAS: '' } })
    expect(r.erro).toBe(false)
    expect(f.estado.cambios[0]).toEqual({ textos: { INDICE: 'Contido', ACTA_ORDE: 'Puntos' } })
    expect(JSON.parse(r.contido).cambiados).toEqual(['textos'])
  })
  it('un texto que non existe ou un formato malo dan un erro que explica as claves', async () => {
    const f = montar(editorCoaLista())
    const r = await f.executar('plantilla_editor_cambiar', { textos: { INVENTADO: 'x' } })
    expect(r.erro).toBe(true)
    expect(r.contido).toMatch(/INVENTADO.*non existe.*INDICE, SEN_TAREFAS, ACTA_ORDE/)
    expect((await f.executar('plantilla_editor_cambiar', { textos: ['x'] })).erro).toBe(true)
    expect(f.estado.cambios).toEqual([])
  })
  it('a proba pode facerse sobre un documento de actas', async () => {
    const f = montar()
    await f.executar('plantilla_previsualizar', { formato: 'pdf', documento: 'actas' })
    expect(JSON.parse(f.api.IAChamarFerramenta.mock.calls[0][1])).toMatchObject({ documento: 'actas', latex: 'X {{CORPO}}' })
  })
})

describe('plantilla_editor_cambiar', () => {
  it('cambia só o que se indica, valida e devolve os campos cambiados', async () => {
    const f = montar()
    const r = await f.executar('plantilla_editor_cambiar', { latex: 'A {{CORPO}}', ofimatica: { interlinado: 1.5 }, motor: 'XeLaTeX' })
    expect(r.erro).toBe(false)
    expect(f.estado.cambios[0]).toEqual({ latex: 'A {{CORPO}}', ofimatica: { interlinado: 1.5 }, motor: 'xelatex' })
    expect(JSON.parse(r.contido).cambiados).toEqual(['latex', 'motor', 'ofimatica'])
  })
  it('rexeita un LaTeX sen {{CORPO}} ou con dous, un nome baleiro e un motor descoñecido, sen tocar o borrador', async () => {
    const f = montar()
    for (const args of [{ latex: 'sen corpo' }, { latex: '{{CORPO}} {{CORPO}}' }, { nome: '  ' }, { motor: 'word' }, {}]) {
      const r = await f.executar('plantilla_editor_cambiar', args)
      expect(r.erro, JSON.stringify(args)).toBe(true)
    }
    expect(f.editor.cambiar).not.toHaveBeenCalled()
  })
  it('valida as variables (nome en maiúsculas, sen repetir) e avisa de campos descoñecidos e de imaxes que faltan', async () => {
    const f = montar()
    expect((await f.executar('plantilla_editor_cambiar', { variables: [{ nome: 'telefono', valor: '1' }, { nome: 'CORREO' }] })).erro).toBe(false)
    expect(f.estado.cambios[0].variables).toEqual([{ nome: 'TELEFONO', etiqueta: '', valor: '1' }, { nome: 'CORREO', etiqueta: '', valor: '' }])
    expect((await f.executar('plantilla_editor_cambiar', { variables: [{ nome: 'A B' }] })).erro).toBe(true)
    expect((await f.executar('plantilla_editor_cambiar', { variables: [{ nome: 'A' }, { nome: 'a' }] })).erro).toBe(true)
    const r = await f.executar('plantilla_editor_cambiar', { nome: 'Novo', invento: 1, ofimatica: { cabeceira: '@imaxe falta.png 10', pe: '@imaxe logo.png 5' } })
    const avisos = JSON.parse(r.contido).avisos.join(' | ')
    expect(avisos).toMatch(/invento/)
    expect(avisos).toMatch(/falta\.png/)
    expect(avisos).not.toMatch(/logo\.png/)
  })
  it('un borrador pechado non se pode cambiar (o editor lanza)', async () => {
    const f = montar({ cambiar: vi.fn(() => { throw new Error('o editor desta plantilla xa está pechado') }) })
    const r = await f.executar('plantilla_editor_cambiar', { nome: 'X' })
    expect(r).toEqual({ contido: 'o editor desta plantilla xa está pechado', erro: true })
  })
})

describe('ferramentas de Go neste ámbito', () => {
  it('previsualizar e comprobar levan SEMPRE o borrador do editor (e o id); a IA só escolle formato e páxina', async () => {
    const f = montar()
    const r = await f.executar('plantilla_previsualizar', { formato: 'docx', paxina: 2, id: 'outra', latex: 'colado' })
    expect(r.imaxes).toEqual(['a1'])
    const [nome, args] = f.api.IAChamarFerramenta.mock.calls[0]
    expect(nome).toBe('plantilla_previsualizar')
    expect(JSON.parse(args)).toEqual({ id: 'p1', nome: 'Centro', latex: 'X {{CORPO}}', ofimatica: { fonte: 'Georgia' }, variables: [], formato: 'docx', paxina: 2 })
    await f.executar('plantilla_comprobar', {})
    expect(f.api.IAChamarFerramenta.mock.calls[1][0]).toBe('plantilla_comprobar')
  })
  it('gardar/eliminar imaxes usan a plantilla do editor (non a que diga a IA), gardan antes se hai que gardar e refrescan as imaxes', async () => {
    const f = montar()
    await f.executar('adxunto_gardar_en_plantilla', { ids: ['a1', 'a2'], plantilla: 'outra-plantilla' })
    expect(f.editor.garantirGardada).toHaveBeenCalled()
    expect(JSON.parse(f.api.IAChamarFerramenta.mock.calls[0][1])).toEqual({ ids: ['a1', 'a2'], plantilla: 'p1' })
    expect(f.editor.recargarImaxes).toHaveBeenCalled()
  })
  it('se a plantilla non se pode gardar antes, non se garda nada e dise', async () => {
    const f = montar({ garantirGardada: vi.fn(async () => false) })
    const r = await f.executar('adxunto_gardar_en_plantilla', { ids: ['a1'] })
    expect(r.erro).toBe(true)
    expect(f.api.IAChamarFerramenta).not.toHaveBeenCalled()
  })
  it('as lecturas de adxuntos e a web pasan sen tocar; calquera outra ferramenta de Go queda fóra do ámbito', async () => {
    const f = montar()
    for (const n of ['adxuntos_listar', 'adxunto_ver', 'adxunto_ler', 'adxunto_extraer_imaxes', 'web_buscar', 'web_consultar']) {
      expect((await f.executar(n, { id: 'a1' })).erro).toBe(false)
    }
    for (const n of ['plantilla_gardar', 'plantilla_eliminar', 'opcions_cambiar', 'curso_ler', 'sistema_estado', 'plantilla_escoller']) {
      const r = await f.executar(n, {})
      expect(r.erro, n).toBe(true)
      expect(r.contido).toMatch(/non está dispoñible/)
    }
    expect(f.api.IAChamarFerramenta).toHaveBeenCalledTimes(6)
  })
  it('usuario_preguntar e usuario_avisar van á interface', async () => {
    const f = montar()
    expect(JSON.parse((await f.executar('usuario_preguntar', { pregunta: 'Que letra?', opcions: ['Georgia'] })).contido)).toEqual({ resposta: 'Georgia' })
    await f.executar('usuario_avisar', { mensaxe: 'Feito', tipo: 'aviso' })
    expect(f.ui.avisar).toHaveBeenCalledWith('Feito', 'aviso')
  })
})
