// @vitest-environment jsdom
import { describe, it, expect, vi, afterEach } from 'vitest'
import { createFormularioEditor } from './editor.js'
import { crearFerramentaFormulario } from './ferramenta.js'
import { novoPlanFormulario } from './modelo.js'

afterEach(() => { document.body.replaceChildren(); localStorage.clear() })

const tick = () => new Promise((r) => setTimeout(r, 0))
const esquemaBase = (modo) => ({
  id: 'notas', nome: 'Notas', modo,
  campos: [{ id: 'grupo', tipo: 'texto', etiqueta: 'Grupo' }, { id: 'nota', tipo: 'numero', etiqueta: 'Nota' }],
})

function apiIA(turnos = []) {
  return {
    api: {
      IAAsistenteEstado: vi.fn(async () => ({ configurada: true, provedor: 'openai', preset: 'openai', modelo: 'm' })),
      IAFerramentasDoAmbito: vi.fn(async () => [
        { nome: 'formulario_datos_ler', tipo: 'lectura' }, { nome: 'formulario_datos_cambiar', tipo: 'escritura' },
      ]),
      IATurnoAsistente: vi.fn(async () => turnos.shift() || { texto: 'feito', chamadas: [] }),
    },
    idioma: () => 'gl',
  }
}

function montar(modo, extra = {}) {
  const container = document.createElement('div')
  document.body.append(container)
  const estado = { esquema: esquemaBase(modo), plan: null }
  const opcs = {
    onChange: vi.fn((p) => { estado.plan = p }),
    obterEsquema: () => estado.esquema,
    aoDeseñar: vi.fn(), aoCambiarModo: vi.fn(), avisar: vi.fn(),
    asistente: apiIA(), ...extra,
  }
  const editor = createFormularioEditor(container, opcs)
  estado.plan = novoPlanFormulario('notas')
  editor.setValue(estado.plan)
  return { container, editor, estado, opcs }
}

const botons = (c) => [...c.querySelectorAll('.form-editor__cab button')].map((b) => b.className.split(' ')[0])

describe('editor de formulario: modo deseño', () => {
  it('ten o interruptor de modo, o asistente e Deseñar formulario, e o asistente vai ao lado do deseñador', () => {
    const { container } = montar('deseno')
    expect(botons(container)).toEqual(['form-modo', 'tarefa-asistente', 'crono-btn'])
    expect(container.querySelector('.form-modo').className).toMatch(/form-modo--deseno/)
    expect(container.querySelector('.form-modo').textContent).toBe('Deseño')
    expect(container.querySelector('.crono-btn').textContent).toMatch(/Deseñar formulario/)
    expect(container.querySelector('.form-asist .asistente')).not.toBeNull()
  })

  it('un formulario sen modo (de antes) é de deseño', () => {
    const { container } = montar(undefined)
    expect(botons(container)).toEqual(['form-modo', 'tarefa-asistente', 'crono-btn'])
  })

  it('«Deseñar formulario» e o interruptor avisan a quen chama', () => {
    const { container, opcs } = montar('deseno')
    container.querySelector('.crono-btn').click()
    expect(opcs.aoDeseñar).toHaveBeenCalledTimes(1)
    container.querySelector('.form-modo').click()
    expect(opcs.aoCambiarModo).toHaveBeenCalledWith('produccion')
  })

  it('o asistente ábrese e pecha co seu botón', () => {
    const { container } = montar('deseno')
    const panel = container.querySelector('.form-asist .asistente')
    expect(panel.classList.contains('hidden')).toBe(true)
    container.querySelector('.tarefa-asistente').click()
    expect(panel.classList.contains('hidden')).toBe(false)
    container.querySelector('.tarefa-asistente').click()
    expect(panel.classList.contains('hidden')).toBe(true)
  })

  it('o Asistente datos ten a barra para conectar unha bóveda de Obsidian', async () => {
    const opc = apiIA()
    opc.api.BovedaEstadoDe = vi.fn(async () => ({ conectada: false }))
    const { container } = montar('deseno', { asistente: opc })
    container.querySelector('.tarefa-asistente').click()
    await vi.waitFor(() => expect(container.querySelector('.form-asist .asistente__boveda-conectar')).not.toBeNull())
    expect(opc.api.BovedaEstadoDe).toHaveBeenCalledWith('formulario')
  })

  it('sen opcións de asistente (p. ex. sen IA) non hai botón', () => {
    const { container } = montar('deseno', { asistente: null })
    expect(botons(container)).toEqual(['form-modo', 'crono-btn'])
  })

  it('un formulario sen campos ofrece «Engadir campos»', () => {
    const { container, estado, editor } = montar('deseno')
    estado.esquema = { ...estado.esquema, campos: [] }
    editor.refrescar()
    expect(container.querySelector('.form-baleiro button')).not.toBeNull()
  })

  it('o asistente escribe datos ficticios no documento (de punta a punta) e pasa polo onChange', async () => {
    const turnos = [
      { texto: '', chamadas: [{ id: '1', nome: 'formulario_datos_cambiar', args: JSON.stringify({ valores: { grupo: '1º A', nota: 8.5 } }) }] },
      { texto: 'Enchín o grupo e a nota.', chamadas: [] },
    ]
    const ia = apiIA(turnos)
    const { container, opcs, estado } = montar('deseno', { asistente: ia })
    container.querySelector('.tarefa-asistente').click()
    await tick() // o panel le o estado da IA ao abrirse
    const caixa = container.querySelector('.form-asist .asistente__caixa')
    caixa.value = 'enche o formulario'
    container.querySelector('.form-asist .asistente__enviar').click()
    await vi.waitFor(() => expect(estado.plan.valores.grupo).toBe('1º A'))
    expect(estado.plan.valores.nota).toBe('8.5')
    // Entra polo historial do controlador (desfacer) e o editor amosa o valor.
    expect(opcs.onChange).toHaveBeenLastCalledWith(expect.objectContaining({ valores: { grupo: '1º A', nota: '8.5' } }), 'estrutura')
    expect(container.querySelector('[data-campo="grupo"]').value).toBe('1º A')
    // O turno vai co ámbito propio (nunca o xeral nin o do deseñador).
    expect(ia.api.IATurnoAsistente.mock.calls[0][0].ambito).toBe('formulario_datos')
    expect(ia.api.IAFerramentasDoAmbito).toHaveBeenCalledWith('formulario_datos')
  })
})

describe('editor de formulario: nada de «null» na pantalla', () => {
  it('nin en deseño nin en produción, con ou sen aviso de datos de proba', () => {
    for (const modo of ['deseno', 'produccion']) {
      const { container, estado, editor } = montar(modo)
      expect(container.textContent, modo).not.toMatch(/null|undefined/)
      editor.setValue({ ...estado.plan, datosProba: true })
      expect(container.textContent, modo + ' con proba').not.toMatch(/null|undefined/)
    }
  })
})

describe('editor de formulario: modo PRODUCIÓN (a IA non ve datos privados)', () => {
  it('só queda o interruptor: nin asistente nin Deseñar formulario', () => {
    const { container } = montar('produccion')
    expect(botons(container)).toEqual(['form-modo'])
    expect(container.querySelector('.form-modo').className).toMatch(/form-modo--produccion/)
    expect(container.querySelector('.form-modo').textContent).toBe('Produción')
    expect(container.querySelector('.asistente, .tarefa-asistente, .form-asist .asistente')).toBeNull()
    expect(container.textContent).not.toMatch(/Deseñar formulario|Asistente/)
  })

  it('o interruptor pide volver a deseño', () => {
    const { container, opcs } = montar('produccion')
    container.querySelector('.form-modo').click()
    expect(opcs.aoCambiarModo).toHaveBeenCalledWith('deseno')
  })

  it('un formulario sen campos non ofrece «Engadir campos» (é o deseñador)', () => {
    const { container, estado, editor } = montar('produccion')
    estado.esquema = { ...estado.esquema, campos: [] }
    editor.refrescar()
    expect(container.querySelector('.form-baleiro')).not.toBeNull()
    expect(container.querySelector('.form-baleiro button')).toBeNull()
  })

  it('ao pasar de deseño a produción o asistente DESAPARECE (destruído, non só agochado)', () => {
    const { container, estado, editor } = montar('deseno')
    container.querySelector('.tarefa-asistente').click() // aberto
    expect(container.querySelector('.asistente')).not.toBeNull()
    estado.esquema = { ...estado.esquema, modo: 'produccion' }
    editor.refrescar()
    expect(container.querySelector('.asistente')).toBeNull()
    expect(container.querySelector('.tarefa-asistente')).toBeNull()
    expect(botons(container)).toEqual(['form-modo'])
    // E ao volver a deseño, volve (limpo).
    estado.esquema = { ...estado.esquema, modo: 'deseno' }
    editor.refrescar()
    expect(botons(container)).toEqual(['form-modo', 'tarefa-asistente', 'crono-btn'])
    expect(container.querySelector('.asistente').classList.contains('hidden')).toBe(true)
  })

  it('unha conversa en curso pararase se pasa a produción: as ferramentas negan e non escriben', async () => {
    let liberar
    const ia = apiIA()
    ia.api.IATurnoAsistente = vi.fn()
      .mockImplementationOnce(() => new Promise((r) => { liberar = () => r({ texto: '', chamadas: [{ id: '1', nome: 'formulario_datos_cambiar', args: JSON.stringify({ valores: { grupo: 'FILTRADO' } }) }] }) }))
      .mockResolvedValue({ texto: 'ok', chamadas: [] })
    const { container, estado, editor, opcs } = montar('deseno', { asistente: ia })
    container.querySelector('.tarefa-asistente').click()
    await tick() // o panel le o estado da IA ao abrirse
    container.querySelector('.form-asist .asistente__caixa').value = 'enche'
    container.querySelector('.form-asist .asistente__enviar').click()
    await vi.waitFor(() => expect(typeof liberar).toBe('function')) // o turno está agardando
    estado.esquema = { ...estado.esquema, modo: 'produccion' }
    editor.refrescar()
    liberar()
    await tick(); await tick()
    expect(estado.plan.valores.grupo).toBeUndefined()
    expect(opcs.onChange).not.toHaveBeenCalled()
  })
})

describe('memoria do contexto entre formularios', () => {
  it('só os formularios en produción len e gardan o contexto lembrado; os de deseño nin o len nin o pisan', () => {
    localStorage.setItem('lua.formularios.contexto', JSON.stringify({ centro: 'Centro REAL', responsable: 'Ana Real' }))
    const prod = crearFerramentaFormulario(esquemaBase('produccion'))
    expect(prod.novoPlan().contexto.centro).toBe('Centro REAL')
    const dese = crearFerramentaFormulario(esquemaBase('deseno'))
    expect(dese.novoPlan().contexto.centro).toBe('')
    expect(dese.novoPlan().contexto.responsable).toBe('')

    // O deseño non pisa o que lembra a produción.
    const contedor = document.createElement('div')
    document.body.append(contedor)
    const ed = dese.crearEditor(contedor, { t: (k, f) => f ?? k, onChange: () => {} })
    ed.setValue(dese.novoPlan())
    const centro = contedor.querySelector('[data-campo="ctx_centro"]')
    centro.value = 'Centro FICTICIO'
    centro.dispatchEvent(new Event('input', { bubbles: true }))
    expect(JSON.parse(localStorage.getItem('lua.formularios.contexto')).centro).toBe('Centro REAL')
    // En produción si se lembra.
    const cp = document.createElement('div')
    document.body.append(cp)
    const edp = prod.crearEditor(cp, { t: (k, f) => f ?? k, onChange: () => {} })
    edp.setValue(prod.novoPlan())
    const cp2 = cp.querySelector('[data-campo="ctx_centro"]')
    cp2.value = 'Outro real'
    cp2.dispatchEvent(new Event('input', { bubbles: true }))
    expect(JSON.parse(localStorage.getItem('lua.formularios.contexto')).centro).toBe('Outro real')
  })

  it('a ferramenta di se está en produción segundo o esquema vixente', () => {
    const f = crearFerramentaFormulario(esquemaBase('deseno'))
    expect(f.enProduccion).toBe(false)
    f.actualizarEsquema(esquemaBase('produccion'))
    expect(f.enProduccion).toBe(true)
  })
})

describe('datos de proba da IA (botón en produción)', () => {
  const conProba = (modo, extra = {}) => {
    const r = montar(modo, { aoBorrarDatosProba: vi.fn(), ...extra })
    r.editor.setValue({ ...r.estado.plan, valores: { grupo: 'Proba' }, datosProba: true })
    return r
  }
  it('en produción, con datos de proba, sae o aviso co botón «Borrar datos de proba»', () => {
    const { container, opcs } = conProba('produccion')
    const boton = container.querySelector('.form-proba [data-accion="borrar-proba"]')
    expect(boton.textContent).toBe('Borrar datos de proba')
    boton.click()
    expect(opcs.aoBorrarDatosProba).toHaveBeenCalledTimes(1)
  })
  it('sen a marca de datos de proba non sae, nin en deseño (alí está o asistente)', () => {
    expect(montar('produccion').container.querySelector('.form-proba')).toBeNull()
    expect(conProba('deseno').container.querySelector('.form-proba')).toBeNull()
  })
  it('o aviso desaparece cando o documento xa non leva a marca (despois de borrar)', () => {
    const { container, estado, editor } = conProba('produccion')
    expect(container.querySelector('.form-proba')).not.toBeNull()
    editor.setValue({ ...estado.plan, valores: {}, datosProba: undefined })
    expect(container.querySelector('.form-proba')).toBeNull()
  })
  it('a marca sobrevive ao teclear (datos reais por riba dos de proba)', () => {
    const { container, opcs } = conProba('produccion')
    const campo = container.querySelector('[data-campo="nota"]')
    campo.value = '7'
    campo.dispatchEvent(new Event('input', { bubbles: true }))
    expect(opcs.onChange).toHaveBeenLastCalledWith(expect.objectContaining({ datosProba: true }), 'texto')
  })
  it('o asistente pon a marca ao escribir datos, e retírase ao baleirar', async () => {
    const turnos = [
      { texto: '', chamadas: [{ id: '1', nome: 'formulario_datos_cambiar', args: JSON.stringify({ valores: { grupo: '1º A' } }) }] },
      { texto: 'feito', chamadas: [] },
    ]
    const { container, estado } = montar('deseno', { asistente: apiIA(turnos) })
    container.querySelector('.tarefa-asistente').click()
    await tick()
    container.querySelector('.form-asist .asistente__caixa').value = 'enche'
    container.querySelector('.form-asist .asistente__enviar').click()
    await vi.waitFor(() => expect(estado.plan.datosProba).toBe(true))
  })
})

describe('referencias e datos do documento', () => {
  it('xa non hai check das referencias: os textos saen tal cal', () => {
    const m = montar('deseno')
    expect(m.container.querySelector('input[data-accion="referencias"]')).toBeNull()
    expect(m.container.textContent).not.toContain('Amosar as referencias')
  })

  it('un documento que as tiña ocultas recupéraas', async () => {
    const m = montar('deseno')
    const plan = { ...m.estado.plan, valores: { grupo: 'A', nota: 7 }, senReferencias: true, referenciasOcultas: { grupo: 'A [PAU 2021 · ordinaria]' } }
    m.editor.setValue(plan)
    await tick()
    expect(m.estado.plan.senReferencias).toBe(false)
    expect(m.container.querySelector('[data-campo="grupo"]').value).toBe('A [PAU 2021 · ordinaria]')
  })

  it('«Datos do documento» sae pechado e ábrese ao premelo', () => {
    const m = montar('deseno')
    const det = m.container.querySelector('details.form-ctx')
    expect(det.open).toBe(false)
    det.open = true
    det.dispatchEvent(new Event('toggle'))
    m.editor.refrescar()
    expect(m.container.querySelector('details.form-ctx').open).toBe(true)
  })
})
