// @vitest-environment jsdom
//
// Probas do editor de plantillas en bloques e columnas (plantillas-editor.js):
// Aplicar garda SEN pechar e deixa seguir editando, a vista previa (PDF e
// Word/ODT), o aviso de cambios sen aplicar, a xanela propia e o asistente.
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { abrirEditorPlantilla } from './plantillas.js'
import { montarEditorPlantilla } from './plantillas-editor.js'

const t = (clave, fallback, vars) => {
  let val = fallback ?? clave
  if (vars) for (const k in vars) val = val.replaceAll(`{${k}}`, vars[k])
  return val
}

function apiFalsa() {
  return {
    listar: vi.fn(async () => ({ plantillas: [], activa: '' })),
    gardar: vi.fn(async (p) => ({ ...p, id: p.id || 'nova-id', imaxes: p.imaxes || [] })),
    subirImaxe: vi.fn(async ({ fileName }) => fileName),
    eliminarImaxe: vi.fn(async () => {}),
    previsualizar: vi.fn(async () => ({ pageImages: ['data:image/png;base64,AA'], warnings: [] })),
    previsualizarOfi: vi.fn(async () => ({ pageImages: ['data:image/png;base64,BB', 'data:image/png;base64,CC'], warnings: [] })),
    fragmentosPorDefecto: vi.fn(async () => ({
      portada: 'PORTADA-YANG', sinatura: 'SINATURA-YANG', introducion: 'INTRO-YANG', portadaOfimatica: '@logo 35',
      estilo: '% ESTILO-YANG\n\\newcommand{\\actacab}',
      textos: [
        { clave: 'INDICE', grupo: 'comun', etiqueta: 'Título do índice', defecto: 'Índice' },
        { clave: 'SEN_TAREFAS', grupo: 'cronograma', etiqueta: 'Mes sen tarefas', defecto: 'Sen tarefas' },
        { clave: 'TRIMESTRE', grupo: 'cronograma', etiqueta: 'Nome de cada trimestre', defecto: '{{N}}.º trimestre', axuda: '{{N}}: número do trimestre' },
        { clave: 'ACTA_ORDE', grupo: 'actas', etiqueta: 'Título da orde do día', defecto: 'Orde do día' },
      ],
    })),
    aplicada: vi.fn(async () => {}),
  }
}

const P = { id: 'p1', nome: 'Centro', descricion: '', latex: 'CAB\n{{CORPO}}', markdown: '', motor: '', variables: [], imaxes: [], ofimatica: { fonte: 'Georgia' } }

const visible = () => document.querySelector('.overlay:not(.hidden) .modal')
const botonDe = (raiz, txt) => [...raiz.querySelectorAll('button')].find((b) => b.textContent.includes(txt))
const campo = (raiz, etiqueta) => [...raiz.querySelectorAll('label')].find((l) => l.firstChild && l.firstChild.textContent === etiqueta)?.querySelector('input,textarea,select')

beforeEach(() => {
  localStorage.clear()
  document.querySelectorAll('.overlay').forEach((o) => o.classList.add('hidden'))
})
afterEach(() => {
  document.querySelectorAll('.overlay').forEach((o) => o.classList.add('hidden'))
})

describe('editor en bloques e columnas', () => {
  it('ten catro columnas (o deseño primeiro), a vista previa e o pé con Aplicar / Gardar / Cancelar', async () => {
    await abrirEditorPlantilla({ t, api: apiFalsa(), plantilla: { ...P } })
    const m = visible()
    expect(m.classList.contains('plantilla-pantalla')).toBe(true)
    const cols = m.querySelectorAll('.plantilla-columna')
    expect(cols).toHaveLength(4)
    expect([...cols].map((c) => c.querySelector('.plantilla-columna__tit').textContent)).toEqual(['Deseño', 'Datos e recursos', 'PDF e LaTeX', 'Word e OpenDocument'])
    for (const c of [...cols].slice(1)) expect(c.querySelectorAll('.plantilla-bloque').length).toBeGreaterThanOrEqual(2)
    expect(m.querySelector('.plantilla-vista')).toBeTruthy()
    const nomes = [...m.querySelectorAll('.modal-actions button')].map((b) => b.textContent)
    expect(nomes).toEqual(expect.arrayContaining(['Cancelar', 'Aplicar', 'Gardar']))
    expect(m.querySelector('.tarefa-x')).toBeTruthy() // a X vermella
  })

  it('o deseño de Word/OpenDocument ten letra, tamaño, interliñado e as catro marxes', async () => {
    const api = apiFalsa()
    await abrirEditorPlantilla({ t, api, plantilla: { ...P } })
    const m = visible()
    expect(campo(m, 'Tipo de letra').value).toBe('Georgia')
    const inter = campo(m, 'Interliñado (1 = sinxelo)')
    inter.value = '1.5'
    inter.dispatchEvent(new Event('input', { bubbles: true }))
    for (const [etq, v] of [['Marxe superior (cm)', '3'], ['Marxe inferior (cm)', '2'], ['Marxe esquerda (cm)', '3.5'], ['Marxe dereita (cm)', '1.5']]) {
      const c = campo(m, etq)
      c.value = v
      c.dispatchEvent(new Event('input', { bubbles: true }))
    }
    botonDe(m.querySelector('.modal-actions'), 'Aplicar').click()
    await vi.waitFor(() => expect(api.gardar).toHaveBeenCalled())
    expect(api.gardar.mock.calls[0][0].ofimatica).toMatchObject({
      fonte: 'Georgia', interlinado: 1.5, marxeSuperior: 3, marxeInferior: 2, marxeEsquerda: 3.5, marxeDereita: 1.5,
    })
  })
})

describe('Aplicar: gardar sen pechar e seguir editando', () => {
  it('garda, avisa á xanela principal, deixa a pantalla aberta e actualiza a vista previa', async () => {
    const api = apiFalsa()
    await abrirEditorPlantilla({ t, api, plantilla: { ...P } })
    const m = visible()
    m.querySelector('.plantilla-codigo').value = 'NOVO\n{{CORPO}}'
    botonDe(m.querySelector('.modal-actions'), 'Aplicar').click()
    await vi.waitFor(() => expect(api.aplicada).toHaveBeenCalled())
    expect(api.gardar.mock.calls[0][0].latex).toBe('NOVO\n{{CORPO}}')
    expect(api.aplicada.mock.calls[0][0].id).toBe('p1')
    expect(document.querySelector('.overlay:not(.hidden)')).toBeTruthy() // non se pechou
    // Unha ao abrir e outra tras aplicar ("Actualizar ao aplicar", activo por defecto).
    await vi.waitFor(() => expect(api.previsualizar).toHaveBeenCalledTimes(2))
    await vi.waitFor(() => expect(m.querySelectorAll('.plantilla-previa__paxina')).toHaveLength(1))
    expect(m.querySelector('.tarefa-cab__estado').textContent).toBe('✓ Gardado')
  })

  it('despois de aplicar, os cambios de Word/ODT seguen chegando ao seguinte gardado (o obxecto ofimatica non se perde)', async () => {
    const api = apiFalsa()
    api.gardar = vi.fn(async (p) => ({ ...p, id: 'p1', imaxes: [], ofimatica: { ...p.ofimatica, fonte: (p.ofimatica.fonte || '').toUpperCase() } }))
    await abrirEditorPlantilla({ t, api, plantilla: { ...P } })
    const m = visible()
    const accions = m.querySelector('.modal-actions')
    botonDe(accions, 'Aplicar').click()
    await vi.waitFor(() => expect(api.gardar).toHaveBeenCalledTimes(1))
    await vi.waitFor(() => expect(campo(m, 'Tipo de letra').value).toBe('GEORGIA')) // o valor normalizado volve aos campos
    const cab = campo(m, 'Cabeceira (en todas as páxinas menos a portada)')
    cab.value = '{{CENTRO}}'
    cab.dispatchEvent(new Event('input', { bubbles: true }))
    const cor = m.querySelector('.plantilla-ofi-cor input[type=color]')
    cor.value = '#aa0000'
    cor.dispatchEvent(new Event('input', { bubbles: true }))
    await vi.waitFor(() => expect(botonDe(accions, 'Aplicar').disabled).toBe(false))
    botonDe(accions, 'Aplicar').click()
    await vi.waitFor(() => expect(api.gardar).toHaveBeenCalledTimes(2))
    expect(api.gardar.mock.calls[1][0].ofimatica).toMatchObject({ cabeceira: '{{CENTRO}}', corPrincipal: 'AA0000' })
  })

  it('Gardar garda e pecha; sen nome ou sen {{CORPO}} non garda e di por que', async () => {
    const api = apiFalsa()
    await abrirEditorPlantilla({ t, api, plantilla: { ...P, nome: '' } })
    const m = visible()
    botonDe(m.querySelector('.modal-actions'), 'Gardar').click()
    await vi.waitFor(() => expect(m.querySelector('.plantilla-estado').textContent).toContain('nome'))
    expect(api.gardar).not.toHaveBeenCalled()
    campo(m, 'Nome').value = 'Con nome'
    m.querySelector('.plantilla-codigo').value = 'sen corpo'
    botonDe(m.querySelector('.modal-actions'), 'Aplicar').click()
    await vi.waitFor(() => expect(m.querySelector('.plantilla-estado').textContent).toContain('CORPO'))
    m.querySelector('.plantilla-codigo').value = 'ben {{CORPO}}'
    botonDe(m.querySelector('.modal-actions'), 'Gardar').click()
    await vi.waitFor(() => expect(document.querySelector('.overlay:not(.hidden)')).toBeNull())
    expect(api.gardar).toHaveBeenCalledTimes(1)
  })
})

describe('cambios sen aplicar e peche', () => {
  it('con cambios, Cancelar / a X preguntan; sen cambios pechan directamente', async () => {
    await abrirEditorPlantilla({ t, api: apiFalsa(), plantilla: { ...P } })
    let m = visible()
    m.querySelector('.tarefa-x').click() // sen cambios: pecha
    await vi.waitFor(() => expect(document.querySelector('.overlay:not(.hidden)')).toBeNull())

    await abrirEditorPlantilla({ t, api: apiFalsa(), plantilla: { ...P } })
    m = visible()
    const area = m.querySelector('.plantilla-codigo')
    area.value = 'CAMBIADO {{CORPO}}'
    area.dispatchEvent(new Event('input', { bubbles: true }))
    expect(m.querySelector('.tarefa-cab__estado').textContent).toContain('sen aplicar')
    botonDe(m.querySelector('.modal-actions'), 'Cancelar').click()
    const dialogo = await vi.waitUntil(() => document.querySelector('.overlay--dialog'))
    expect(dialogo.textContent).toContain('cambios sen gardar')
    // "Seguir editando": non se pecha
    ;[...dialogo.querySelectorAll('button')].find((b) => b.textContent === 'Seguir editando').click()
    await vi.waitFor(() => expect(document.querySelector('.overlay--dialog')).toBeNull())
    expect(document.querySelector('.overlay:not(.hidden)')).toBeTruthy()
    // Descartar: pecha
    m.querySelector('.tarefa-x').click()
    const d2 = await vi.waitUntil(() => document.querySelector('.overlay--dialog'))
    ;[...d2.querySelectorAll('button')].find((b) => b.textContent === 'Descartar').click()
    await vi.waitFor(() => expect(document.querySelector('.overlay:not(.hidden)')).toBeNull())
  })

  it('avisa a quen o monta cando hai/deixa de haber cambios (para que Go non deixe pechar a xanela) e ao aplicar volve a false', async () => {
    const contedor = document.createElement('div')
    document.body.append(contedor)
    const aoCambiarSucio = vi.fn()
    const ed = await montarEditorPlantilla(contedor, { t, api: apiFalsa(), plantilla: { ...P }, aoCambiarSucio })
    const area = contedor.querySelector('.plantilla-codigo')
    area.value = 'X {{CORPO}}'
    area.dispatchEvent(new Event('input', { bubbles: true }))
    expect(aoCambiarSucio).toHaveBeenLastCalledWith(true)
    expect(ed.cambiado()).toBe(true)
    expect(await ed.aplicar()).toBe(true)
    expect(aoCambiarSucio).toHaveBeenLastCalledWith(false)
    expect(ed.cambiado()).toBe(false)
    ed.destruir()
    contedor.remove()
  })
})

describe('vista previa', () => {
  it('o selector de formato usa previsualizarOfi para Word e OpenDocument e amosa as súas páxinas', async () => {
    const api = apiFalsa()
    await abrirEditorPlantilla({ t, api, plantilla: { ...P } })
    const m = visible()
    const sel = m.querySelector('.plantilla-vista__formato')
    sel.value = 'docx'
    sel.dispatchEvent(new Event('change', { bubbles: true }))
    await vi.waitFor(() => expect(api.previsualizarOfi).toHaveBeenCalledWith(expect.objectContaining({ id: 'p1' }), 'docx', 'cronograma', null))
    await vi.waitFor(() => expect(m.querySelectorAll('.plantilla-previa__paxina')).toHaveLength(2))
    expect(api.previsualizar).not.toHaveBeenCalled()
  })

  it('avisa de que a vista previa quedou vella cando se cambia algo despois, e "Actualizar ao aplicar" pódese desactivar', async () => {
    const api = apiFalsa()
    await abrirEditorPlantilla({ t, api, plantilla: { ...P } })
    const m = visible()
    botonDe(m.querySelector('.plantilla-vista__cab'), 'Actualizar').click()
    await vi.waitFor(() => expect(m.querySelectorAll('.plantilla-previa__paxina')).toHaveLength(1))
    expect(m.querySelector('.plantilla-vista__aviso').hidden).toBe(true)
    const area = m.querySelector('.plantilla-codigo')
    area.value = 'OUTRO {{CORPO}}'
    area.dispatchEvent(new Event('input', { bubbles: true }))
    expect(m.querySelector('.plantilla-vista__aviso').hidden).toBe(false)
    const auto = m.querySelector('.plantilla-vista__auto input')
    auto.checked = false
    auto.dispatchEvent(new Event('change'))
    botonDe(m.querySelector('.modal-actions'), 'Aplicar').click()
    await vi.waitFor(() => expect(api.gardar).toHaveBeenCalled())
    await new Promise((r) => setTimeout(r, 30))
    expect(api.previsualizar).toHaveBeenCalledTimes(2) // a de abrir e a do botón: non se volveu compilar
    expect(localStorage.getItem('lua.plantilla.autoVista')).toBe('0')
  })
})

describe('xanela propia do sistema', () => {
  it('se hai xanela nativa (api.abrirXanela devolve true) non se abre o overlay e pásanse os datos', async () => {
    const api = apiFalsa()
    api.abrirXanela = vi.fn(async () => true)
    await abrirEditorPlantilla({ t, api, plantilla: { ...P }, abrirIA: true })
    expect(api.abrirXanela).toHaveBeenCalledWith('p1', JSON.stringify({ plantilla: { ...P }, abrirIA: true, documento: null }))
    expect(document.querySelector('.overlay:not(.hidden)')).toBeNull()
  })

  it('unha plantilla nova pide a xanela co id baleiro; se non hai xanelas, cae ao overlay a pantalla completa', async () => {
    const api = apiFalsa()
    api.abrirXanela = vi.fn(async () => false)
    await abrirEditorPlantilla({ t, api, plantilla: null })
    expect(api.abrirXanela.mock.calls[0][0]).toBe('')
    const ov = document.querySelector('.overlay:not(.hidden)')
    expect(ov.classList.contains('tarefa-overlay')).toBe(true)
  })
})

describe('asistente da plantilla', () => {
  it('só aparece o botón do asistente se hai IA (api.asistente)', async () => {
    await abrirEditorPlantilla({ t, api: apiFalsa(), plantilla: { ...P } })
    expect(visible().querySelector('.tarefa-asistente')).toBeNull()
    const api = apiFalsa()
    api.asistente = {
      api: {
        IAAsistenteEstado: vi.fn(async () => ({ configurada: true, provedor: 'openai', preset: 'openai', modelo: 'x' })),
        IAFerramentasDoAmbito: vi.fn(async () => []), IATurnoAsistente: vi.fn(), IAChamarFerramenta: vi.fn(),
      },
      idioma: () => 'gl',
    }
    await abrirEditorPlantilla({ t, api, plantilla: { ...P } })
    const m = visible()
    expect(m.querySelector('.tarefa-asistente')).toBeTruthy()
    m.querySelector('.tarefa-asistente').click()
    // Fichas por defecto: portada, letra, interliñado, marxes, cabeceira, pé...
    const fichas = [...m.querySelectorAll('.asistente .asistente__chip')].map((x) => x.textContent)
    for (const quero of ['Modificar a portada', 'Cambiar a letra', 'Axustar o interliñado', 'Axustar as marxes', 'Modificar a cabeceira', 'Modificar o pé de páxina']) {
      expect(fichas.some((f) => f.includes(quero)), quero).toBe(true)
    }
    // Unha ficha manda a súa instrución co ámbito "plantilla" (e as ferramentas dese ámbito).
    api.asistente.api.IATurnoAsistente.mockResolvedValue({ texto: 'Feito.', chamadas: [], bruto: '', provedor: 'openai' })
    ;[...m.querySelectorAll('.asistente .asistente__chip')].find((x) => x.textContent.includes('Axustar as marxes')).click()
    await vi.waitFor(() => expect(api.asistente.api.IATurnoAsistente).toHaveBeenCalled())
    const req = api.asistente.api.IATurnoAsistente.mock.calls[0][0]
    expect(req.ambito).toBe('plantilla')
    expect(req.mensaxes[0].texto).toContain('MARXES')
    expect(req.estado).toContain('«Centro»') // resumo do borrador no prompt
    expect(api.asistente.api.IAFerramentasDoAmbito).toHaveBeenCalledWith('plantilla')
  })
})


describe('lanzador de bloques (só se despregan os que se chaman)', () => {
  const abertos = (m) => [...m.querySelectorAll('.plantilla-bloque:not([hidden])')].map((b) => b.dataset.bloque)
  const chip = (m, id) => m.querySelector(`.plantilla-chip[data-bloque="${id}"]`)

  it('de primeiras só está á vista o deseño, e cada bloque ten o seu botón', async () => {
    await abrirEditorPlantilla({ t, api: apiFalsa(), plantilla: { ...P } })
    const m = visible()
    expect(abertos(m).sort()).toEqual(['deseno'])
    const ids = [...m.querySelectorAll('.plantilla-bloque')].map((b) => b.dataset.bloque)
    expect(ids).toHaveLength(15)
    for (const id of ids) expect(chip(m, id), id).toBeTruthy()
    // As columnas sen nada á vista agóchanse: só queda a do deseño.
    const cols = m.querySelectorAll('.plantilla-columna')
    expect(cols[0].hidden).toBe(false)
    expect([...cols].slice(1).every((c) => c.hidden)).toBe(true)
    expect(chip(m, 'deseno').getAttribute('aria-pressed')).toBe('true')
    expect(chip(m, 'latex').getAttribute('aria-pressed')).toBe('false')
    expect(chip(m, 'estilo').getAttribute('aria-pressed')).toBe('false')
  })

  it('premer un botón despraga o bloque e outra vez pécha o; o ✕ do bloque tamén', async () => {
    await abrirEditorPlantilla({ t, api: apiFalsa(), plantilla: { ...P } })
    const m = visible()
    chip(m, 'cores').click()
    expect(abertos(m)).toContain('cores')
    expect(m.querySelectorAll('.plantilla-columna')[3].hidden).toBe(false)
    chip(m, 'cores').click()
    expect(abertos(m)).not.toContain('cores')
    expect(m.querySelectorAll('.plantilla-columna')[3].hidden).toBe(true)
    chip(m, 'estilo').click()
    m.querySelector('.plantilla-bloque[data-bloque="estilo"] .plantilla-bloque__x').click()
    expect(abertos(m)).not.toContain('estilo')
  })

  it('lembra que bloques había abertos', async () => {
    await abrirEditorPlantilla({ t, api: apiFalsa(), plantilla: { ...P } })
    chip(visible(), 'textos').click()
    document.querySelectorAll('.overlay').forEach((o) => o.classList.add('hidden'))
    await abrirEditorPlantilla({ t, api: apiFalsa(), plantilla: { ...P } })
    expect(abertos(visible())).toContain('textos')
  })
})

describe('Estilo e Textos do documento (o deseño e os textos son da plantilla)', () => {
  it('«Usar o estilo de Lúa» enche o estilo e gárdase coa plantilla', async () => {
    const api = apiFalsa()
    await abrirEditorPlantilla({ t, api, plantilla: { ...P } })
    const m = visible()
    const bloque = m.querySelector('.plantilla-bloque[data-bloque="estilo"]')
    expect(bloque.querySelector('textarea').value).toBe('')
    botonDe(bloque, 'Usar o estilo de Lúa').click()
    expect(bloque.querySelector('textarea').value).toContain('ESTILO-YANG')
    botonDe(m.querySelector('.modal-actions'), 'Aplicar').click()
    await vi.waitFor(() => expect(api.gardar).toHaveBeenCalled())
    expect(api.gardar.mock.calls[0][0].estilo).toContain('ESTILO-YANG')
  })

  it('amosa todos os textos con o valor de Lua de exemplo e só garda os que se cambian', async () => {
    const api = apiFalsa()
    await abrirEditorPlantilla({ t, api, plantilla: { ...P, textos: { SEN_TAREFAS: 'Libre' } } })
    const m = visible()
    const bloque = m.querySelector('.plantilla-bloque[data-bloque="textos"]')
    const inputs = [...bloque.querySelectorAll('.plantilla-texto input')]
    expect(inputs).toHaveLength(4)
    const por = (etq) => [...bloque.querySelectorAll('.plantilla-texto')].find((f) => f.querySelector('.plantilla-texto__etq').textContent === etq)
    expect(por('Título do índice').querySelector('input').placeholder).toBe('Índice')
    expect(por('Título do índice').querySelector('input').value).toBe('')
    expect(por('Mes sen tarefas').querySelector('input').value).toBe('Libre')
    expect(por('Nome de cada trimestre').textContent).toContain('{{N}}: número do trimestre')

    const indice = por('Título do índice').querySelector('input')
    indice.value = 'Contido'
    indice.dispatchEvent(new Event('input', { bubbles: true }))
    // O ↺ devolve un texto ao de Lua.
    botonDe(por('Mes sen tarefas'), '↺').click()
    botonDe(m.querySelector('.modal-actions'), 'Aplicar').click()
    await vi.waitFor(() => expect(api.gardar).toHaveBeenCalled())
    expect(api.gardar.mock.calls[0][0].textos).toEqual({ INDICE: 'Contido' })
  })

  it('un texto en branco non se garda (vale o de Lua) e o buscador filtra', async () => {
    const api = apiFalsa()
    await abrirEditorPlantilla({ t, api, plantilla: { ...P } })
    const m = visible()
    const bloque = m.querySelector('.plantilla-bloque[data-bloque="textos"]')
    const primeiro = bloque.querySelector('.plantilla-texto input')
    primeiro.value = '   '
    primeiro.dispatchEvent(new Event('input', { bubbles: true }))
    const filtro = bloque.querySelector('input[type="search"]')
    filtro.value = 'orde'
    filtro.dispatchEvent(new Event('input', { bubbles: true }))
    const visibles = [...bloque.querySelectorAll('.plantilla-texto')].filter((f) => !f.hidden)
    expect(visibles.map((f) => f.querySelector('.plantilla-texto__etq').textContent)).toEqual(['Título da orde do día'])
    botonDe(m.querySelector('.modal-actions'), 'Aplicar').click()
    await vi.waitFor(() => expect(api.gardar).toHaveBeenCalled())
    expect(api.gardar.mock.calls[0][0].textos || {}).toEqual({})
  })

  it('a vista previa faise sempre co documento en edición, sen selector de mostra', async () => {
    const api = apiFalsa()
    api.documentoActual = () => ({ ferramenta: 'formulario', nomeBase: 'boletin' })
    await abrirEditorPlantilla({ t, api, plantilla: { ...P } })
    const m = visible()
    expect(m.querySelector('.plantilla-vista__mostra')).toBeNull()
    expect(m.querySelector('.plantilla-vista select[title="Documento de mostra da vista previa"]')).toBeNull()
    m.querySelector('.plantilla-vista__cab button').click()
    await vi.waitFor(() => expect(api.previsualizar).toHaveBeenCalledWith(expect.objectContaining({ id: 'p1' }), 'actual', { ferramenta: 'formulario', nomeBase: 'boletin' }))
  })
  it('sen documento aberto cae ao cronograma de exemplo', async () => {
    const api = apiFalsa()
    await abrirEditorPlantilla({ t, api, plantilla: { ...P } })
    visible().querySelector('.plantilla-vista__cab button').click()
    await vi.waitFor(() => expect(api.previsualizar).toHaveBeenCalledWith(expect.objectContaining({ id: 'p1' }), 'cronograma', null))
  })
})
