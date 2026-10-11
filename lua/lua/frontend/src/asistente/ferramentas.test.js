import { describe, it, expect, vi } from 'vitest'
import { crearFerramentas, vistaCurso } from './ferramentas.js'
import { normalizarPlan, novoPlan, gardarTarefa } from '../informes/cronograma-modelo.js'

const HOXE = new Date(2025, 9, 15)

function planBase() {
  let p = normalizarPlan(novoPlan(HOXE)) // 2025-09 .. 2026-06
  p = gardarTarefa(p, '2025-09', { id: 'a', title: 'Avaliacion', description: 'Texto\n\n![foto](img:i1 =60%)', color: 'blue', images: { i1: 'data:image/png;base64,AAAA' } })
  p = gardarTarefa(p, '2025-09', { id: 'b', title: 'Titorias', description: '', color: 'amber' })
  return p
}

function montar(plan = planBase(), extra = {}) {
  const estado = { plan, historial: [] }
  const informes = {
    getPlan: () => estado.plan,
    substituirPlan: (novo, motivo) => { estado.plan = novo; estado.historial.push(motivo) },
    abrirTarefa: vi.fn(),
    novo: vi.fn(async () => {}), abrir: vi.fn(async () => {}),
    gardar: vi.fn(async () => true), gardarComo: vi.fn(async () => true),
    xerar: vi.fn(async () => true),
    ultimoXerar: () => ({ ok: true, avisos: ['aviso1'], paxinas: 7, erro: '' }),
    ultimoErroExportar: () => '',
    exportarPDF: vi.fn(async () => '/tmp/x.pdf'), exportarTex: vi.fn(async () => '/tmp/x.tex'),
    exportarMarkdown: vi.fn(async () => null), exportarDocx: vi.fn(async () => '/tmp/x.docx'), exportarOdt: vi.fn(async () => '/tmp/x.odt'),
    desfacer: vi.fn(), refacer: vi.fn(),
    ...extra.informes,
  }
  let tipografia = { tamanoPt: 0, interlinado: 0, fonte: '' }
  const ui = {
    desacoplarResultado: vi.fn(async () => {}), acoplarResultado: vi.fn(async () => {}),
    escollerPlantilla: vi.fn(async () => {}),
    getTipografia: () => tipografia, setTipografia: (t) => { tipografia = t },
    avisar: vi.fn(), preguntar: vi.fn(async () => 'Outubro'),
    refrescarPlantillas: vi.fn(), recargarOpcions: vi.fn(), abrirTrasladar: vi.fn(async () => {}), reutilizar: vi.fn(async () => {}),
    ...extra.ui,
  }
  const api = {
    IAChamarFerramenta: vi.fn(async (nome, args) => ({ nome, contido: 'ok ' + args, erro: false })),
    AdxuntoDataURI: vi.fn(async (id, max) => `data:image/png;base64,IMG-${id}-${max}`),
    ...extra.api,
  }
  const { executar, nomesFrontend } = crearFerramentas({ informes, api, ui })
  return { estado, informes, ui, api, executar, nomesFrontend }
}
const json = (r) => JSON.parse(r.contido)

describe('curso_ler', () => {
  it('devolve o curso sen imaxes en base64, con ids e o número de imaxes', async () => {
    const { executar } = montar()
    const r = await executar('curso_ler', {})
    expect(r.erro).toBe(false)
    const v = json(r)
    expect(v.curso.anoAcademico).toBeTruthy()
    expect(v.totalTarefas).toBe(2)
    const set = v.meses.find((m) => m.mes === '2025-09')
    expect(set.tarefas.map((t) => t.id)).toEqual(['a', 'b'])
    expect(set.tarefas[0].imaxes).toBe(1)
    expect(r.contido).not.toContain('base64')
    expect('ficheiro' in v).toBe(false) // o curso vive en Lúa, non nun ficheiro
  })
  it('permite pedir só a configuración, ou só un mes', async () => {
    const { executar } = montar()
    expect(json(await executar('curso_ler', { ambito: 'config' })).meses).toBeUndefined()
    const so = json(await executar('curso_ler', { ambito: 'tarefas', mes: '2025-09' }))
    expect(so.curso).toBeUndefined()
    expect(so.meses).toHaveLength(1)
    const mal = await executar('curso_ler', { mes: '1999-01' })
    expect(mal.erro).toBe(true)
    expect(mal.contido).toMatch(/rango/)
  })
  it('avisa das tarefas fóra do rango de datas', () => {
    const p = gardarTarefa(planBase(), '2030-01', { id: 'z', title: 'Lonxe', description: '', color: 'blue' })
    expect(vistaCurso(p).avisoForaDeRango).toMatch(/1 tarefa/)
  })
})

describe('curso_configurar', () => {
  it('cambia só os campos indicados e entra no historial', async () => {
    const { executar, estado } = montar()
    const r = await executar('curso_configurar', { centro: 'IES Novo', xefe: 'Ana', introducion: 'Ola.' })
    expect(r.erro).toBe(false)
    expect(estado.plan.config.schoolName).toBe('IES Novo')
    expect(estado.plan.config.headOfDepartment).toBe('Ana')
    expect(estado.plan.config.introText).toBe('Ola.')
    expect(estado.plan.config.startDate).toBe('2025-09-08') // non tocada
    expect(estado.historial).toEqual(['estrutura'])
  })
  it('valida as datas e non cambia nada se son malas', async () => {
    const { executar, estado } = montar()
    const antes = estado.plan
    const r = await executar('curso_configurar', { dataFin: '2020-01-01' })
    expect(r.erro).toBe(true)
    expect(r.contido).toMatch(/anterior/)
    expect(estado.plan).toBe(antes)
  })
  it('sen campos dá erro; bloqueado só deixa desbloquear', async () => {
    const { executar, estado } = montar()
    expect((await executar('curso_configurar', {})).erro).toBe(true)
    expect((await executar('curso_configurar', { bloqueado: true })).erro).toBe(false)
    expect(estado.plan.isLocked).toBe(true)
    expect((await executar('curso_configurar', { centro: 'X' })).erro).toBe(true)
    expect((await executar('curso_configurar', { bloqueado: false, centro: 'X' })).erro).toBe(false)
    expect(estado.plan.isLocked).toBe(false)
    expect(estado.plan.config.schoolName).toBe('X')
  })
})

describe('tarefas', () => {
  it('engadir crea a tarefa co id devolto, cor por defecto e valida mes/título/cor', async () => {
    const { executar, estado } = montar()
    const r = await executar('tarefa_engadir', { mes: '2025-10', titulo: 'Nova', descricion: '**x**', cor: 'rose' })
    expect(r.erro).toBe(false)
    const { id } = json(r)
    expect(estado.plan.monthsData['2025-10'].tasks[0]).toMatchObject({ id, title: 'Nova', description: '**x**', color: 'rose' })
    const semCor = await executar('tarefa_engadir', { mes: '2025-10', titulo: 'Outra' })
    expect(estado.plan.monthsData['2025-10'].tasks[1].color).toBe('blue')
    expect(semCor.erro).toBe(false)
    expect((await executar('tarefa_engadir', { mes: '2031-01', titulo: 'X' })).contido).toMatch(/non está no rango/)
    expect((await executar('tarefa_engadir', { mes: '2025-10', titulo: '  ' })).erro).toBe(true)
    expect((await executar('tarefa_engadir', { mes: '2025-10', titulo: 'X', cor: 'verde-lima' })).contido).toMatch(/cor non válida/)
  })
  it('editar cambia campos, conserva as imaxes que a descrición aínda usa e avisa se se perden', async () => {
    const { executar, estado } = montar()
    let r = await executar('tarefa_editar', { id: 'a', titulo: 'Avaliación', descricion: 'Novo texto\n\n![foto](img:i1 =60%)' })
    expect(r.erro).toBe(false)
    expect(estado.plan.monthsData['2025-09'].tasks[0]).toMatchObject({ title: 'Avaliación', description: expect.stringContaining('Novo texto') })
    expect(estado.plan.monthsData['2025-09'].tasks[0].images.i1).toBeTruthy()
    expect(json(r).avisos).toBeUndefined()

    r = await executar('tarefa_editar', { id: 'a', descricion: 'Sen imaxe' })
    expect(estado.plan.monthsData['2025-09'].tasks[0].images).toBeUndefined()
    expect(json(r).avisos[0]).toMatch(/imaxe/)
  })
  it('editar pode cambiar a cor e mover de mes; valida id e cor', async () => {
    const { executar, estado } = montar()
    const r = await executar('tarefa_editar', { id: 'b', cor: 'purple', mes: '2025-11' })
    expect(r.erro).toBe(false)
    expect(estado.plan.monthsData['2025-09'].tasks.map((t) => t.id)).toEqual(['a'])
    expect(estado.plan.monthsData['2025-11'].tasks[0]).toMatchObject({ id: 'b', color: 'purple' })
    expect((await executar('tarefa_editar', { id: 'nada', titulo: 'x' })).contido).toMatch(/non existe/)
    expect((await executar('tarefa_editar', { id: 'a' })).erro).toBe(true)
    expect((await executar('tarefa_editar', { id: 'a', cor: 'xx' })).erro).toBe(true)
    expect((await executar('tarefa_editar', { id: 'a', mes: '2040-01' })).erro).toBe(true)
  })
  it('eliminar e mover', async () => {
    const { executar, estado } = montar()
    expect(json(await executar('tarefa_mover', { id: 'b', direccion: 'arriba' })).movida).toBe(true)
    expect(estado.plan.monthsData['2025-09'].tasks.map((t) => t.id)).toEqual(['b', 'a'])
    expect(json(await executar('tarefa_mover', { id: 'b', direccion: 'arriba' })).movida).toBe(false)
    expect((await executar('tarefa_mover', { id: 'b', direccion: 'esquerda' })).erro).toBe(true)
    expect((await executar('tarefa_eliminar', { id: 'a' })).erro).toBe(false)
    expect(estado.plan.monthsData['2025-09'].tasks.map((t) => t.id)).toEqual(['b'])
  })
  it('con o curso bloqueado ningunha edición de tarefas pasa', async () => {
    const { executar } = montar({ ...planBase(), isLocked: true })
    for (const [n, a] of [['tarefa_engadir', { mes: '2025-09', titulo: 'x' }], ['tarefa_editar', { id: 'a', titulo: 'x' }], ['tarefa_eliminar', { id: 'a' }], ['tarefa_mover', { id: 'a', direccion: 'abaixo' }]]) {
      const r = await executar(n, a)
      expect(r.erro, n).toBe(true)
      expect(r.contido).toMatch(/bloqueado/)
    }
  })
  it('tarefa_abrir pide á interface abrir a pantalla da tarefa', async () => {
    const { executar, informes } = montar()
    await executar('tarefa_abrir', { id: 'b' })
    expect(informes.abrirTarefa).toHaveBeenCalledWith('2025-09', expect.objectContaining({ id: 'b' }))
  })
})

describe('imaxes adxuntas que pasan ao JSON do curso', () => {
  it('curso_logotipo garda o data URI (reducido a 900 px) no curso e pódese quitar', async () => {
    const { executar, estado, api } = montar()
    const r = await executar('curso_logotipo', { adxunto: 'a1b2' })
    expect(r.erro).toBe(false)
    expect(api.AdxuntoDataURI).toHaveBeenCalledWith('a1b2', 900)
    expect(estado.plan.config.schoolLogo).toBe('data:image/png;base64,IMG-a1b2-900')
    expect(estado.historial).toEqual(['estrutura'])
    expect((await executar('curso_logotipo', { quitar: true })).erro).toBe(false)
    expect(estado.plan.config.schoolLogo).toBe('')
    expect((await executar('curso_logotipo', {})).contido).toMatch(/adxunto/)
  })
  it('curso_logotipo devolve o erro de Go (non é imaxe) sen tocar o curso; e respecta o bloqueo', async () => {
    const { executar, estado } = montar(planBase(), { api: { AdxuntoDataURI: vi.fn(async () => { throw new Error('«a.txt» non é unha imaxe') }) } })
    const antes = estado.plan
    const r = await executar('curso_logotipo', { adxunto: 'a9' })
    expect(r).toMatchObject({ erro: true, contido: expect.stringMatching(/non é unha imaxe/) })
    expect(estado.plan).toBe(antes)
    const b = montar({ ...planBase(), isLocked: true })
    expect((await b.executar('curso_logotipo', { adxunto: 'a1' })).contido).toMatch(/bloqueado/)
  })
  it('tarefa_imaxe garda a imaxe no JSON da tarefa e insire a liña na descrición', async () => {
    const { executar, estado, api } = montar()
    const r = await executar('tarefa_imaxe', { id: 'b', adxunto: 'a5', largura: 45, alt: 'Cartaz [1]' })
    expect(r.erro).toBe(false)
    const { imaxeId } = json(r)
    expect(api.AdxuntoDataURI).toHaveBeenCalledWith('a5', 1600)
    const t = estado.plan.monthsData['2025-09'].tasks.find((x) => x.id === 'b')
    expect(t.images[imaxeId]).toBe('data:image/png;base64,IMG-a5-1600')
    expect(t.description).toBe(`![Cartaz  1](img:${imaxeId} =45%)`)
    expect(estado.historial).toEqual(['estrutura'])
  })
  it('tarefa_imaxe respecta a descrición existente, a posición e o aliñamento; as imaxes vellas quedan', async () => {
    const { executar, estado } = montar()
    await executar('tarefa_imaxe', { id: 'a', adxunto: 'a5', posicion: 'inicio', alinear: 'dereita' })
    const t = estado.plan.monthsData['2025-09'].tasks.find((x) => x.id === 'a')
    expect(t.description.startsWith(':::dereita\n![](img:i')).toBe(true)
    expect(t.description).toContain('=60%)\n:::\n\nTexto\n\n![foto](img:i1 =60%)')
    expect(Object.keys(t.images)).toHaveLength(2)
    expect(t.images.i1).toBe('data:image/png;base64,AAAA')
    // Ao final, a largura queda entre 5 e 100.
    await executar('tarefa_imaxe', { id: 'a', adxunto: 'a6', largura: 900 })
    const t2 = estado.plan.monthsData['2025-09'].tasks.find((x) => x.id === 'a')
    expect(t2.description.endsWith('=100%)')).toBe(true)
  })
  it('tarefa_imaxe con tarefa inexistente non pide a imaxe', async () => {
    const { executar, api } = montar()
    const r = await executar('tarefa_imaxe', { id: 'nada', adxunto: 'a5' })
    expect(r.erro).toBe(true)
    expect(api.AdxuntoDataURI).not.toHaveBeenCalled()
  })
  it('as imaxes que devolve unha ferramenta de Go pasan ao resultado', async () => {
    const { executar } = montar(planBase(), { api: { IAChamarFerramenta: vi.fn(async () => ({ contido: 'páxina 2', erro: false, imaxes: ['a8'] })) } })
    expect(await executar('adxunto_ver', { id: 'a1' })).toEqual({ contido: 'páxina 2', erro: false, imaxes: ['a8'] })
  })
})

describe('informe, ficheiros e interface', () => {
  it('informe_xerar devolve páxinas e avisos; se falla, o erro', async () => {
    const { executar, informes } = montar()
    expect(json(await executar('informe_xerar', {}))).toEqual({ ok: true, paxinas: 7, avisos: ['aviso1'] })
    informes.xerar.mockResolvedValueOnce(false)
    informes.ultimoXerar = () => ({ ok: false, avisos: [], paxinas: 0, erro: 'Falta LaTeX' })
    const r = await executar('informe_xerar', {})
    expect(r.erro).toBe(true)
    expect(r.contido).toMatch(/^Falta LaTeX .*rexistro_ler/)
  })
  it('informe_exportar chama a exportación do formato e devolve a ruta', async () => {
    const { executar, informes } = montar()
    expect(json(await executar('informe_exportar', { formato: 'docx' }))).toEqual({ exportado: true, ruta: '/tmp/x.docx' })
    expect(informes.exportarDocx).toHaveBeenCalled()
    expect(json(await executar('informe_exportar', { formato: 'md' })).exportado).toBe(false) // cancelou
    expect((await executar('informe_exportar', { formato: 'rtf' })).erro).toBe(true)
    informes.exportarOdt.mockResolvedValueOnce(null)
    informes.ultimoErroExportar = () => 'sen espazo'
    expect((await executar('informe_exportar', { formato: 'odt' })).contido).toBe('sen espazo')
  })
  it('catalogo_reutilizar pon «form:» aos ids dos formularios', async () => {
    const { executar, ui } = montar()
    expect(json(await executar('catalogo_reutilizar', { ids: ['exame', 'cronograma', 'form:ficha'] })).reutilizados).toEqual(['form:exame', 'cronograma', 'form:ficha'])
    expect(ui.reutilizar).toHaveBeenCalledWith(['form:exame', 'cronograma', 'form:ficha'])
    expect((await executar('catalogo_reutilizar', { ids: [] })).erro).toBe(true)
  })
  it('curso_gardar, historial e resultado_xanela', async () => {
    const { executar, informes, ui } = montar()
    expect(json(await executar('curso_gardar', {})).gardado).toBe(true)
    expect(informes.gardar).toHaveBeenCalled()
    await executar('curso_gardar', { comoNovo: true })
    expect(ui.abrirTrasladar).toHaveBeenCalledWith('sacar')
    await executar('curso_abrir', {})
    expect(ui.abrirTrasladar).toHaveBeenCalledWith('meter')
    await executar('historial', { accion: 'desfacer' })
    expect(informes.desfacer).toHaveBeenCalled()
    expect((await executar('historial', { accion: 'foo' })).erro).toBe(true)
    await executar('resultado_xanela', { accion: 'desacoplar' })
    expect(ui.desacoplarResultado).toHaveBeenCalled()
    await executar('resultado_xanela', { accion: 'acoplar' })
    expect(ui.acoplarResultado).toHaveBeenCalled()
  })
  it('axustes_texto le e valida', async () => {
    const { executar } = montar()
    expect(json(await executar('axustes_texto', {}))).toEqual({ tamanoPt: 0, interlinado: 0, fonte: '' })
    expect(json(await executar('axustes_texto', { tamanoPt: 12, fonte: 'sans' }))).toEqual({ tamanoPt: 12, interlinado: 0, fonte: 'sans' })
    expect((await executar('axustes_texto', { tamanoPt: 400 })).erro).toBe(true)
    expect((await executar('axustes_texto', { interlinado: 'moito' })).erro).toBe(true)
  })
  it('usuario_avisar e usuario_preguntar', async () => {
    const { executar, ui } = montar()
    await executar('usuario_avisar', { mensaxe: 'Ollo', tipo: 'aviso' })
    expect(ui.avisar).toHaveBeenCalledWith('Ollo', 'aviso')
    expect((await executar('usuario_avisar', {})).erro).toBe(true)
    expect(json(await executar('usuario_preguntar', { pregunta: 'Que mes?', opcions: ['Setembro', 'Outubro'] }))).toEqual({ resposta: 'Outubro' })
    expect(ui.preguntar).toHaveBeenCalledWith('Que mes?', ['Setembro', 'Outubro'])
    ui.preguntar.mockResolvedValueOnce(null)
    expect(json(await executar('usuario_preguntar', { pregunta: 'x' })).resposta).toBeNull()
  })
  it('plantilla_escoller pasa polo cambio de plantilla da interface', async () => {
    const { executar, ui } = montar()
    await executar('plantilla_escoller', { id: 'abc' })
    expect(ui.escollerPlantilla).toHaveBeenCalledWith('abc')
    expect((await executar('plantilla_escoller', {})).erro).toBe(true)
  })
})

describe('ferramentas de backend', () => {
  it('as descoñecidas ao frontend van a Go co JSON dos argumentos', async () => {
    const { executar, api } = montar()
    const r = await executar('plantillas_listar', {})
    expect(api.IAChamarFerramenta).toHaveBeenCalledWith('plantillas_listar', '{}')
    expect(r).toEqual({ contido: 'ok {}', erro: false, imaxes: [] })
  })
  it('refresca o selector de plantillas tras cambialas e recarga as Opcións tras cambialas', async () => {
    const { executar, ui } = montar()
    await executar('plantilla_gardar', { nome: 'x' })
    await executar('plantilla_eliminar', { id: 'x' })
    expect(ui.refrescarPlantillas).toHaveBeenCalledTimes(2)
    // Gardar logotipos nunha plantilla de exemplo crea unha copia propia: o selector actualízase.
    await executar('adxunto_gardar_en_plantilla', { plantilla: 'x', ids: ['a1', 'a2'] })
    expect(ui.refrescarPlantillas).toHaveBeenCalledTimes(3)
    await executar('plantilla_ler', { id: 'x' })
    expect(ui.refrescarPlantillas).toHaveBeenCalledTimes(3)
    await executar('opcions_cambiar', { idioma: 'es' })
    expect(ui.recargarOpcions).toHaveBeenCalled()
  })
  it('un erro de Go (erro=true) non refresca nada e chega á IA', async () => {
    const { executar, ui } = montar(planBase(), { api: { IAChamarFerramenta: vi.fn(async () => ({ contido: 'non compila', erro: true })) } })
    const r = await executar('plantilla_gardar', {})
    expect(r).toEqual({ contido: 'non compila', erro: true, imaxes: [] })
    expect(ui.refrescarPlantillas).not.toHaveBeenCalled()
  })
  it('se o binding lanza unha excepción, tamén vai coma erro (non rompe o bucle)', async () => {
    const { executar } = montar(planBase(), { api: { IAChamarFerramenta: vi.fn(async () => { throw new Error('rede caída') }) } })
    expect(await executar('web_buscar', { consulta: 'x' })).toEqual({ contido: 'rede caída', erro: true })
  })
  it('un resultado enorme córtase', async () => {
    const enorme = 'x'.repeat(50000)
    const { executar } = montar(planBase(), { api: { IAChamarFerramenta: vi.fn(async () => ({ contido: enorme, erro: false })) } })
    const r = await executar('web_consultar', { url: 'https://x' })
    expect(r.contido.length).toBeLessThan(31000)
    expect(r.contido).toMatch(/cortado/)
  })
})

describe('PRIVACIDADE: formulario en modo produción', () => {
  // Coa ferramenta activa nun formulario en produción (datos reais) o asistente xeral
  // non toca nada: nin le, nin xera, nin exporta, nin chega ao backend.
  const enProduccion = (extra = {}) => montar(planBase(), { informes: { ferramenta: { id: 'form:notas', esFormulario: true, enProduccion: true }, ...extra } })

  it('nega todas as ferramentas, tamén as de backend (nin chega a chamalas)', async () => {
    const { executar, api, informes } = enProduccion()
    for (const nome of ['curso_ler', 'informe_xerar', 'informe_exportar', 'historial', 'resultado_xanela', 'curso_gardar', 'plantillas_listar', 'web_consultar', 'ferramenta_descoñecida']) {
      const r = await executar(nome, { formato: 'pdf', accion: 'desfacer', url: 'https://x' })
      expect(r.erro, nome).toBe(true)
      expect(r.contido, nome).toMatch(/PRODUCIÓN/)
    }
    expect(api.IAChamarFerramenta).not.toHaveBeenCalled()
    expect(informes.xerar).not.toHaveBeenCalled()
    expect(informes.exportarPDF).not.toHaveBeenCalled()
    expect(informes.desfacer).not.toHaveBeenCalled()
  })

  it('só pode marchar a outra ferramenta (ferramenta_activa non se nega)', async () => {
    const { executar } = enProduccion()
    const r = await executar('ferramenta_activa', { ferramenta: 'cronograma' })
    expect(r.contido).not.toMatch(/PRODUCIÓN/)
  })

  it('cos formularios en deseño e as demais ferramentas, todo igual ca sempre', async () => {
    const dese = montar(planBase(), { informes: { ferramenta: { id: 'form:notas', esFormulario: true, enProduccion: false } } })
    expect((await dese.executar('informe_xerar', {})).contido).not.toMatch(/PRODUCIÓN/)
    const normal = montar()
    expect((await normal.executar('informe_xerar', {})).erro).toBe(false)
  })
})

describe('cobertura do catálogo', () => {
  it('as ferramentas de frontend que declara Go teñen código aquí', () => {
    // Lista espello de ferramentasFrontend en asistente_ferramentas.go: se se
    // engade unha alí sen código aquí, o asistente recibiría "descoñecida".
    const declaradas = ['curso_ler', 'curso_configurar', 'tarefa_engadir', 'tarefa_editar', 'tarefa_eliminar', 'tarefa_mover',
      'tarefa_abrir', 'curso_novo', 'curso_abrir', 'curso_gardar', 'informe_xerar', 'informe_exportar', 'historial',
      'resultado_xanela', 'plantilla_escoller', 'axustes_texto', 'usuario_avisar', 'usuario_preguntar', 'curso_logotipo', 'tarefa_imaxe',
      'ferramenta_activa', 'actas_ler', 'actas_configurar', 'actas_membros', 'actas_materias', 'acta_gardar', 'acta_eliminar', 'acta_seguimento', 'actas_ambito', 'catalogo_reutilizar']
    const { nomesFrontend } = montar()
    expect([...nomesFrontend].sort()).toEqual([...declaradas].sort())
  })
})
