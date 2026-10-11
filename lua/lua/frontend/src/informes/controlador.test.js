// @vitest-environment jsdom
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { createInformes } from './controlador.js'
import { escribirFicheiro, novoPlan, normalizarPlan, gardarTarefa, novaTarefa } from './cronograma-modelo.js'
import { FERRAMENTA_ACTAS } from './ferramenta-actas.js'
import { escribirFicheiroActas, novoPlanActas, normalizarPlanActas, gardarActa, novaActa } from './actas-modelo.js'

function novoContexto(apiExtra = {}, uiExtra = {}, opcions = {}) {
  const container = document.createElement('div')
  document.body.append(container)
  const api = {
    LerBorradorInforme: vi.fn(async () => ''),
    GardarBorradorInforme: vi.fn(async () => {}),
    XerarInformeCronograma: vi.fn(async () => ({ pdfBase64: 'UERG', pageImages: ['data:image/png;base64,AA'], warnings: ['aviso'], log: 'log' })),
    SavePDFDialog: vi.fn(async () => '/tmp/x.pdf'),
    ExportarInformeTex: vi.fn(async () => ({ path: '/tmp/x.tex', warnings: [] })),
    ExportarInformeMarkdown: vi.fn(async () => ({ path: '/tmp/x.md', warnings: [] })),
    ExportarInformeDocx: vi.fn(async () => ({ path: '/tmp/x.docx', warnings: [] })),
    ExportarInformeOdt: vi.fn(async () => ({ path: '/tmp/x.odt', warnings: [] })),
    ...apiExtra,
  }
  const ui = {
    setStatus: vi.fn(), setStatusGardado: vi.fn(), showWarnings: vi.fn(), log: vi.fn(), logAvisos: vi.fn(),
    mensaxeErro: (e) => (e && e.message) || String(e),
    previsualizar: vi.fn(), limparResultado: vi.fn(), setUltimoPDF: vi.fn(),
    confirmar: vi.fn(async () => true), ocupado: vi.fn(), ocupadoBoton: vi.fn(), logLatex: vi.fn(), aoFallar: vi.fn(),
    ...uiExtra,
  }
  const inf = createInformes({ container, api, ui, ...opcions })
  return { inf, api, ui, container }
}

// Mes de inicio do curso por defecto (depende da data de hoxe): as tarefas
// de proba van nel para que caian dentro do rango de datas.
const mesInicio = (plan) => plan.config.startDate.slice(0, 7)
const planCunhaTarefa = (titulo, mes = 0) => {
  const base = normalizarPlan(novoPlan())
  const [a, m] = mesInicio(base).split('-').map(Number)
  const clave = `${a + Math.floor((m - 1 + mes) / 12)}-${String(((m - 1 + mes) % 12) + 1).padStart(2, '0')}`
  return gardarTarefa(base, clave, { ...novaTarefa(), title: titulo })
}

afterEach(() => { document.body.replaceChildren(); vi.useRealTimers() })

describe('arranque e borrador', () => {
  it('sen borrador arranca cun curso novo baleiro e sen sucio', async () => {
    const { inf, ui } = novoContexto()
    await inf.iniciar()
    const e = inf._estado()
    expect(e.plan.monthsData).toEqual({})
    expect(e.sucio).toBe(false)
    expect(ui.setStatus).toHaveBeenLastCalledWith('Gardado en Lúa')
  })

  it('recupera o documento gardado en Lúa (un «path» vello ignórase)', async () => {
    const plan = planCunhaTarefa('Recuperada')
    const { inf, container } = novoContexto({
      LerBorradorInforme: vi.fn(async () => escribirFicheiro(plan, { path: '/tmp/curso.json' })),
    })
    await inf.iniciar()
    expect(inf._estado().sucio).toBe(false)
    expect(container.textContent).toContain('Recuperada')
  })

  it('un borrador corrupto non impide arrancar', async () => {
    const { inf } = novoContexto({ LerBorradorInforme: vi.fn(async () => '{isto non é json') })
    await inf.iniciar()
    expect(inf._estado().plan.monthsData).toEqual({})
  })

})

describe('descartar (ferramenta borrada)', () => {
  it('despois de descartar o borrador NUNCA volve escribirse (nin polo debounce nin ao volcar)', async () => {
    vi.useFakeTimers()
    const { inf, api, container } = novoContexto()
    await inf.iniciar()
    const centro = [...container.querySelectorAll('.crono-camp input[type=text]')].find((i) => i.placeholder.startsWith('IES'))
    centro.value = 'DATO REAL'
    centro.dispatchEvent(new Event('input')) // deixa un gardado pendente
    inf.descartar()
    vi.advanceTimersByTime(2000)
    await inf.volcarBorrador()
    expect(api.GardarBorradorInforme).not.toHaveBeenCalled()
    // Se o borrado falla, pódese reactivar e volve gardar.
    inf.descartar(false)
    await inf.volcarBorrador()
    expect(api.GardarBorradorInforme).toHaveBeenCalledTimes(1)
  })
})

describe('edición, desfacer e borrador automático', () => {
  it('editar un campo marca sucio, garda o borrador con debounce e permite desfacer', async () => {
    vi.useFakeTimers()
    const { inf, api, container } = novoContexto()
    await inf.iniciar()
    const centro = [...container.querySelectorAll('.crono-camp input[type=text]')].find((i) => i.placeholder.startsWith('IES'))
    centro.value = 'IES Nova'
    centro.dispatchEvent(new Event('input'))
    expect(inf._estado().sucio).toBe(true)
    expect(inf._estado().plan.config.schoolName).toBe('IES Nova')
    expect(api.GardarBorradorInforme).not.toHaveBeenCalled()
    vi.advanceTimersByTime(800)
    expect(api.GardarBorradorInforme).toHaveBeenCalledTimes(1)
    expect(JSON.parse(api.GardarBorradorInforme.mock.calls[0][0]).plans[0].config.schoolName).toBe('IES Nova')

    inf.desfacer()
    expect(inf._estado().plan.config.schoolName).toBe('')
    inf.refacer()
    expect(inf._estado().plan.config.schoolName).toBe('IES Nova')
  })

  it('o tecleo seguido nun campo é un só paso de desfacer', async () => {
    vi.useFakeTimers()
    const { inf, container } = novoContexto()
    await inf.iniciar()
    const centro = [...container.querySelectorAll('.crono-camp input[type=text]')].find((i) => i.placeholder.startsWith('IES'))
    for (const v of ['I', 'IE', 'IES']) { centro.value = v; centro.dispatchEvent(new Event('input')); vi.advanceTimersByTime(100) }
    expect(inf._estado().pila).toBe(2) // orixe + un paso
    inf.desfacer()
    expect(inf._estado().plan.config.schoolName).toBe('')
  })

  it('desfacer/refacer nos bordos non fan nada', async () => {
    const { inf } = novoContexto()
    await inf.iniciar()
    inf.desfacer()
    inf.refacer()
    expect(inf._estado().idx).toBe(0)
  })
})

describe('documento', () => {
  it('novo pide confirmación se hai contido e pódese desfacer', async () => {
    const plan = planCunhaTarefa('X')
    const { inf, ui, api } = novoContexto({ LerBorradorInforme: vi.fn(async () => escribirFicheiro(plan)) })
    await inf.iniciar()
    ui.confirmar.mockResolvedValueOnce(false)
    await inf.novo()
    expect(ui.confirmar).toHaveBeenCalledTimes(1)
    expect(Object.keys(inf._estado().plan.monthsData)).toHaveLength(1) // cancelado: non se tocou
    await inf.novo() // confirmar devolve true por defecto
    expect(inf._estado().plan.monthsData).toEqual({})
    expect(api.GardarBorradorInforme).toHaveBeenCalled()
    inf.desfacer()
    expect(Object.keys(inf._estado().plan.monthsData)).toHaveLength(1)
    await inf.novo() // con contido outra vez: pregunta
    await inf.novo() // baleiro: nin pregunta
    expect(ui.confirmar).toHaveBeenCalledTimes(3)
  })

  it('gardar escribe xa en Lúa, sen diálogos', async () => {
    const { inf, api, ui } = novoContexto()
    await inf.iniciar()
    expect(await inf.gardar()).toBe(true)
    expect(api.GardarBorradorInforme).toHaveBeenCalledWith(expect.stringContaining('"plans"'))
    expect(api.GardarBorradorInforme.mock.calls[0][0]).not.toContain('"path"')
    expect(ui.setStatus).toHaveBeenLastCalledWith('Gardado en Lúa')
    expect(inf.abrir).toBeUndefined()
    expect(inf.gardarComo).toBeUndefined()
  })

  it('un erro ao gardar dío', async () => {
    const { inf, ui } = novoContexto({ GardarBorradorInforme: vi.fn(async () => { throw new Error('sen permiso') }) })
    await inf.iniciar()
    expect(await inf.gardar()).toBe(false)
    expect(ui.setStatus).toHaveBeenLastCalledWith('Erro ao gardar')
  })

  it('contidoDocumento é o mesmo JSON que se garda en Lúa', async () => {
    const plan = planCunhaTarefa('Para sacar')
    const { inf } = novoContexto({ LerBorradorInforme: vi.fn(async () => escribirFicheiro(plan)) })
    await inf.iniciar()
    expect(JSON.parse(inf.contidoDocumento()).plans[0].monthsData).toEqual(plan.monthsData)
  })
})

describe('xerar', () => {
  it('manda o curso, amosa as páxinas e os avisos e garda o PDF para imprimir', async () => {
    const { inf, api, ui } = novoContexto()
    await inf.iniciar()
    expect(await inf.xerar()).toBe(true)
    expect(api.XerarInformeCronograma.mock.calls[0][0].plan.academicYear).toMatch(/^\d{4}\/\d{4}$/)
    expect(ui.previsualizar).toHaveBeenCalledWith(['data:image/png;base64,AA'])
    // Os avisos van ao rexistro, despois do «Feito.»; a barra amarela queda baleira.
    expect(ui.logAvisos).toHaveBeenLastCalledWith(['aviso'])
    expect(ui.log.mock.invocationCallOrder.at(-1)).toBeLessThan(ui.logAvisos.mock.invocationCallOrder.at(-1))
    expect(ui.showWarnings).toHaveBeenLastCalledWith([])
    expect(ui.setStatus).toHaveBeenLastCalledWith(expect.stringMatching(/1/))
    expect(ui.setUltimoPDF).toHaveBeenLastCalledWith('UERG')
    expect(inf.temResultado()).toBe(true)
    expect(ui.ocupado.mock.calls).toEqual([[true], [false, true]]) // o 2.º dato di se saíu ben (a barra de progreso aprende o tempo)
  })

  it('con datas inválidas non chama ao backend e explica o motivo', async () => {
    const { inf, api, ui, container } = novoContexto()
    await inf.iniciar()
    const plan = inf._estado().plan
    plan.config.endDate = '2000-01-01'
    expect(await inf.xerar()).toBe(false)
    expect(api.XerarInformeCronograma).not.toHaveBeenCalled()
    expect(ui.showWarnings).toHaveBeenLastCalledWith([expect.stringMatching(/anterior/)])
  })

  it('un erro do backend amósase, chama a aoFallar e libera o botón', async () => {
    const { inf, ui } = novoContexto({ XerarInformeCronograma: vi.fn(async () => { throw new Error('pdflatex fallou') }) })
    await inf.iniciar()
    expect(await inf.xerar()).toBe(false)
    expect(ui.showWarnings).toHaveBeenLastCalledWith(['pdflatex fallou'])
    expect(ui.aoFallar).toHaveBeenCalled()
    expect(ui.ocupado).toHaveBeenLastCalledWith(false, false)
  })

  it('non arranca dúas xeracións á vez', async () => {
    let solta
    const { inf, api } = novoContexto({ XerarInformeCronograma: vi.fn(() => new Promise((r) => { solta = r })) })
    await inf.iniciar()
    const a = inf.xerar()
    expect(await inf.xerar()).toBe(false)
    solta({ pdfBase64: 'x', pageImages: [], warnings: [], log: '' })
    await a
    expect(api.XerarInformeCronograma).toHaveBeenCalledTimes(1)
  })
})

describe('exportar', () => {
  it('PDF: xera se non hai resultado e despois pide onde gardalo', async () => {
    const { inf, api } = novoContexto()
    await inf.iniciar()
    await inf.exportarPDF()
    expect(api.XerarInformeCronograma).toHaveBeenCalledTimes(1)
    expect(api.SavePDFDialog).toHaveBeenCalledWith(expect.stringMatching(/^cronograma-.*\.pdf$/), 'UERG')
  })

  it('PDF: non recompila se o curso non cambiou, pero si se cambiou', async () => {
    vi.useFakeTimers()
    const { inf, api, container } = novoContexto()
    await inf.iniciar()
    await inf.xerar()
    await inf.exportarPDF()
    expect(api.XerarInformeCronograma).toHaveBeenCalledTimes(1)
    const centro = [...container.querySelectorAll('.crono-camp input[type=text]')].find((i) => i.placeholder.startsWith('IES'))
    centro.value = 'Cambiado'
    centro.dispatchEvent(new Event('input'))
    await inf.exportarPDF()
    expect(api.XerarInformeCronograma).toHaveBeenCalledTimes(2)
  })

  it('PDF: se a xeración falla non intenta gardar', async () => {
    const { inf, api } = novoContexto({ XerarInformeCronograma: vi.fn(async () => { throw new Error('mal') }) })
    await inf.iniciar()
    await inf.exportarPDF()
    expect(api.SavePDFDialog).not.toHaveBeenCalled()
  })

  it('TeX/MD/DOCX/ODT chaman ao seu método co nome suxerido e amosan a ruta', async () => {
    const { inf, api, ui } = novoContexto()
    await inf.iniciar()
    await inf.exportarTex('b')
    await inf.exportarMarkdown('b')
    await inf.exportarDocx('b')
    await inf.exportarOdt('b')
    expect(api.ExportarInformeTex.mock.calls[0][1]).toMatch(/^cronograma-\d{4}-\d{4}\.tex$/)
    expect(api.ExportarInformeMarkdown.mock.calls[0][1]).toMatch(/\.md$/)
    expect(api.ExportarInformeDocx.mock.calls[0][1]).toMatch(/\.docx$/)
    expect(api.ExportarInformeOdt.mock.calls[0][1]).toMatch(/\.odt$/)
    expect(ui.setStatusGardado).toHaveBeenLastCalledWith(expect.stringContaining('/tmp/x.odt'), '/tmp/x.odt')
    expect(ui.ocupadoBoton).toHaveBeenCalledWith('b', true)
    expect(ui.ocupadoBoton).toHaveBeenLastCalledWith('b', false)
  })

  it('un erro na exportación avisa e libera o botón', async () => {
    const { inf, ui } = novoContexto({ ExportarInformeDocx: vi.fn(async () => { throw new Error('non hai pandoc') }) })
    await inf.iniciar()
    await inf.exportarDocx('b')
    expect(ui.showWarnings).toHaveBeenLastCalledWith(['non hai pandoc'])
    expect(ui.ocupadoBoton).toHaveBeenLastCalledWith('b', false)
  })
})

describe('plantillas (estilos) por curso', () => {
  it('a petición leva nomeBase (o nome suxerido polo curso)', async () => {
    const { inf, api } = novoContexto()
    await inf.iniciar()
    await inf.xerar()
    expect(api.XerarInformeCronograma.mock.calls[0][0].nomeBase).toMatch(/^cronograma-\d{4}-\d{4}$/)
  })

  it('as exportacións MD/DOCX/ODT mandan a petición completa (curso, tipografía e nomeBase)', async () => {
    const { inf, api } = novoContexto()
    await inf.iniciar()
    await inf.exportarMarkdown('b')
    await inf.exportarDocx('b')
    await inf.exportarOdt('b')
    for (const f of [api.ExportarInformeMarkdown, api.ExportarInformeDocx, api.ExportarInformeOdt]) {
      const req = f.mock.calls[0][0]
      expect(req.plan).toBeDefined()
      expect(req.nomeBase).toMatch(/^cronograma-\d{4}-\d{4}$/)
      expect('tipografia' in req).toBe(true)
    }
  })
})


describe('a ferramenta de actas co mesmo controlador', () => {
  const actas = (apiExtra = {}, uiExtra = {}) => novoContexto(apiExtra, uiExtra, { ferramenta: FERRAMENTA_ACTAS })
  const planCunhaActa = () => {
    const p = normalizarPlanActas(novoPlanActas())
    const [a, m] = p.config.startDate.slice(0, 7).split('-')
    return gardarActa(p, `${a}-${m}`, { ...novaActa(p.config), id: 'a1' })
  }

  it('arranca cun curso de actas e xera co pedido de actas (ferramenta, curso e ámbito)', async () => {
    const { inf, api } = actas()
    await inf.iniciar()
    expect(inf._estado().plan.config.members).toEqual([])
    expect(inf.getAmbito()).toBe('curso')
    await inf.xerar()
    const pedido = api.XerarInformeCronograma.mock.calls[0][0]
    expect(pedido).toMatchObject({ ferramenta: 'actas', ambito: 'curso' })
    expect(pedido.actas.config.subjects).toEqual([])
    expect(pedido.nomeBase).toMatch(/^actas-\d{4}-\d{4}$/)
  })

  it('cambiar o ámbito invalida o resultado, cambia o nome ao exportar e non marca o curso coma cambiado', async () => {
    const { inf, api } = actas()
    await inf.iniciar()
    await inf.xerar()
    expect(inf.estaAoDia()).toBe(true)
    inf.setAmbito('2025-11')
    expect(inf.estaAoDia()).toBe(false)
    expect(inf._estado().sucio).toBe(false)
    await inf.exportarPDF()
    // Como o resultado xa non estaba ao día, volveuse xerar co novo ámbito.
    expect(api.XerarInformeCronograma).toHaveBeenCalledTimes(2)
    expect(api.XerarInformeCronograma.mock.calls[1][0]).toMatchObject({ ambito: '2025-11' })
    expect(api.SavePDFDialog.mock.calls[0][0]).toMatch(/-2025-11\.pdf$/)
    inf.setAmbito('2025-11') // o mesmo: non fai nada
    expect(inf.estaAoDia()).toBe(true)
  })

  it('o documento que se garda en Lúa é de actas', async () => {
    const plan = planCunhaActa()
    const { inf, api } = actas()
    await inf.iniciar()
    inf.substituirPlan(plan)
    await inf.volcarBorrador()
    const gardado = JSON.parse(api.GardarBorradorInforme.mock.calls.at(-1)[0])
    expect(gardado.ferramenta).toBe('actas')
    expect(gardado.actas[0].monthsData).toBeTruthy()
  })

  it('recupera o borrador de actas ao arrancar', async () => {
    const plan = planCunhaActa()
    const { inf } = actas({ LerBorradorInforme: vi.fn(async () => escribirFicheiroActas(plan, { path: '/tmp/actas.json' })) })
    await inf.iniciar()
    expect(Object.keys(inf._estado().plan.monthsData)).toHaveLength(1)
    expect(inf._estado().sucio).toBe(false)
  })

  it('avisa a ui de cada cambio do curso (para refrescar a lista de actas do ámbito)', async () => {
    const aoCambiar = vi.fn()
    const { inf } = actas({}, { aoCambiar })
    await inf.iniciar()
    aoCambiar.mockClear()
    inf.substituirPlan(planCunhaActa())
    expect(aoCambiar).toHaveBeenCalled()
    expect(inf.ferramenta.ambitosDisponibles(inf.getPlan())).toHaveLength(1)
  })
})
