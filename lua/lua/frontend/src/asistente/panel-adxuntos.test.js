// @vitest-environment jsdom
import { describe, it, expect, vi, afterEach } from 'vitest'
import { crearPanelAsistente } from './panel.js'

afterEach(() => { document.body.replaceChildren() })

const almacen = () => { const d = {}; return { getItem: (k) => d[k] ?? null, setItem: (k, v) => { d[k] = v } } }
const tick = () => new Promise((r) => setTimeout(r, 0))

function montar({ conAdxuntos = true, turnos = [{ texto: 'vexo', chamadas: [] }], extraApi = {} } = {}) {
  const api = {
    IAAsistenteEstado: vi.fn(async () => ({ configurada: true, provedor: 'openai', preset: 'openai', modelo: 'm' })),
    IAFerramentas: vi.fn(async () => []),
    IATurnoAsistente: vi.fn(async () => turnos.shift() || { texto: 'ok', chamadas: [] }),
    ...(conAdxuntos ? {
      AdxuntoGardar: vi.fn(async (nome, b64) => ({ id: 'a' + nome.length + b64.length, nome, tipo: /\.png$/.test(nome) ? 'imaxe' : 'pdf', tamano: 5 })),
      AdxuntoEliminar: vi.fn(async () => {}), AdxuntosLimpar: vi.fn(async () => {}),
      AdxuntoVista: vi.fn(async () => 'data:image/jpeg;base64,MINI'),
      AdxuntoDataURI: vi.fn(async (id) => `data:image/png;base64,GRANDE-${id}`),
    } : {}),
    ...extraApi,
  }
  const panel = crearPanelAsistente({ t: (k, f) => f ?? k, api, executar: vi.fn(async () => ({ contido: 'ok', erro: false })), estadoPrograma: () => '', almacen: almacen() })
  return { api, panel, raiz: document.getElementById('asistentePanel') }
}

const ficheiro = (nome, contido = 'hola', tipo = 'application/octet-stream') => new File([contido], nome, { type: tipo })
async function escoller(raiz, ficheiros) {
  const input = raiz.querySelector('input[type=file]')
  Object.defineProperty(input, 'files', { value: ficheiros, configurable: true })
  input.dispatchEvent(new Event('change'))
  await vi.waitFor(() => expect(raiz.querySelectorAll('.asis-adx-chip').length).toBe(ficheiros.length))
}

describe('adxuntos no panel do asistente', () => {
  it('con API de adxuntos hai botón de clip; sen ela (asistente da tarefa) non', async () => {
    expect(montar().raiz.querySelector('.asistente__clip')).not.toBeNull()
    document.body.replaceChildren()
    expect(montar({ conAdxuntos: false }).raiz.querySelector('.asistente__clip')).toBeNull()
  })

  it('escoller ficheiros: sóbense a Go en base64 e aparecen coma fichas (coa miniatura das imaxes)', async () => {
    const { api, raiz } = montar()
    await vi.waitFor(() => expect(raiz.querySelector('.asistente__clip').disabled).toBe(false))
    await escoller(raiz, [ficheiro('logo.png', 'PNGDATA'), ficheiro('plan.pdf', 'PDFDATA')])
    expect(api.AdxuntoGardar).toHaveBeenNthCalledWith(1, 'logo.png', btoa('PNGDATA'))
    expect(api.AdxuntoGardar).toHaveBeenNthCalledWith(2, 'plan.pdf', btoa('PDFDATA'))
    const fichas = [...raiz.querySelectorAll('.asis-adx-chip')]
    expect(fichas[0].querySelector('img.asis-adx-chip__mini').src).toContain('MINI')
    expect(fichas[1].querySelector('.asis-adx-chip__ico')).not.toBeNull()
    expect(fichas[1].textContent).toContain('plan.pdf')
  })

  it('quitar unha ficha elimínaa tamén do almacén temporal', async () => {
    const { api, raiz } = montar()
    await vi.waitFor(() => expect(raiz.querySelector('.asistente__clip').disabled).toBe(false))
    await escoller(raiz, [ficheiro('a.pdf'), ficheiro('b.pdf', 'bb')])
    raiz.querySelector('.asis-adx-chip__x').click()
    expect(raiz.querySelectorAll('.asis-adx-chip')).toHaveLength(1)
    expect(api.AdxuntoEliminar).toHaveBeenCalledTimes(1)
  })

  it('enviar leva os ids dos adxuntos, amósaos na burbulla e limpa as fichas; con só adxuntos, o texto é un aviso', async () => {
    const { api, raiz } = montar()
    await vi.waitFor(() => expect(raiz.querySelector('.asistente__clip').disabled).toBe(false))
    await escoller(raiz, [ficheiro('modelo.png', 'X')])
    const id = [...raiz.querySelectorAll('.asis-adx-chip')].length && api.AdxuntoGardar.mock.results.length
    expect(id).toBe(1)
    raiz.querySelector('.asistente__enviar').click() // sen texto
    await vi.waitFor(() => expect(api.IATurnoAsistente).toHaveBeenCalled())
    const msgs = api.IATurnoAsistente.mock.calls[0][0].mensaxes
    expect(msgs[0].texto).toBe('Mira os ficheiros adxuntos.')
    expect(msgs[0].adxuntos).toHaveLength(1)
    expect(raiz.querySelectorAll('.asis-adx-chip')).toHaveLength(0)
    expect(raiz.querySelector('.asis-msg--user .asis-adx').textContent).toContain('modelo.png')
    await vi.waitFor(() => expect(raiz.textContent).toContain('vexo'))
  })

  it('unha ficha de tarefa tamén leva os adxuntos pendentes', async () => {
    const { api, raiz } = montar()
    await vi.waitFor(() => expect(raiz.querySelector('.asistente__clip').disabled).toBe(false))
    await escoller(raiz, [ficheiro('modelo.pdf')])
    ;[...raiz.querySelectorAll('.asistente__chip')].find((c) => c.textContent.includes('Corrixir ortografía')).click()
    await vi.waitFor(() => expect(api.IATurnoAsistente).toHaveBeenCalled())
    expect(api.IATurnoAsistente.mock.calls[0][0].mensaxes[0].adxuntos).toHaveLength(1)
  })

  it('arrastrar e pegar ficheiros tamén os adxunta', async () => {
    const { raiz } = montar()
    await vi.waitFor(() => expect(raiz.querySelector('.asistente__clip').disabled).toBe(false))
    const drop = new Event('drop', { bubbles: true, cancelable: true })
    drop.dataTransfer = { types: ['Files'], files: [ficheiro('solto.pdf')] }
    raiz.dispatchEvent(drop)
    await vi.waitFor(() => expect(raiz.querySelectorAll('.asis-adx-chip')).toHaveLength(1))
    const paste = new Event('paste', { bubbles: true, cancelable: true })
    paste.clipboardData = { files: [ficheiro('captura.png', 'IMG')] }
    raiz.querySelector('.asistente__caixa').dispatchEvent(paste)
    await vi.waitFor(() => expect(raiz.querySelectorAll('.asis-adx-chip')).toHaveLength(2))
    expect(paste.defaultPrevented).toBe(true)
    // Arrastrar algo que non son ficheiros (texto) non fai nada.
    const dropTexto = new Event('drop', { bubbles: true, cancelable: true })
    dropTexto.dataTransfer = { types: ['text/plain'], files: [] }
    raiz.dispatchEvent(dropTexto)
    expect(raiz.querySelectorAll('.asis-adx-chip')).toHaveLength(2)
  })

  it('un ficheiro demasiado grande ou un fallo de Go amósase e non rompe os demais', async () => {
    const { raiz } = montar({ extraApi: { AdxuntoGardar: vi.fn(async (n) => { if (n === 'mala.pdf') throw new Error('o ficheiro está baleiro'); return { id: 'ax' + n.length, nome: n, tipo: 'pdf', tamano: 3 } }) } })
    await vi.waitFor(() => expect(raiz.querySelector('.asistente__clip').disabled).toBe(false))
    const gordo = ficheiro('gordo.pdf')
    Object.defineProperty(gordo, 'size', { value: 31 * 1024 * 1024 })
    const input = raiz.querySelector('input[type=file]')
    Object.defineProperty(input, 'files', { value: [gordo, ficheiro('mala.pdf'), ficheiro('boa.pdf')], configurable: true })
    input.dispatchEvent(new Event('change'))
    await vi.waitFor(() => expect(raiz.querySelectorAll('.asis-adx-chip')).toHaveLength(1))
    const erro = raiz.querySelector('.asistente__erro-adx')
    expect(erro.hidden).toBe(false)
    expect(erro.textContent).toMatch(/gordo\.pdf.*30 MB/)
    expect(erro.textContent).toMatch(/mala\.pdf.*baleiro/)
  })

  it('Nova conversa limpa os adxuntos da sesión (almacén temporal)', async () => {
    const { api, raiz } = montar()
    await vi.waitFor(() => expect(raiz.querySelector('.asistente__clip').disabled).toBe(false))
    await escoller(raiz, [ficheiro('a.pdf')])
    raiz.querySelector('.asistente__ico[title]').click()
    expect(api.AdxuntosLimpar).toHaveBeenCalled()
    expect(raiz.querySelectorAll('.asis-adx-chip')).toHaveLength(0)
  })

  it('as imaxes que a IA mira (adxunto_ver) vense na conversa', async () => {
    const chamada = { id: 'c1', nome: 'adxunto_ver', args: '{"id":"a1"}' }
    const { api, panel, raiz } = montar({ turnos: [{ texto: '', chamadas: [chamada], bruto: 'B' }, { texto: 'Xa a vexo', chamadas: [] }] })
    api.IAFerramentas.mockResolvedValue([{ nome: 'adxunto_ver', tipo: 'lectura', onde: 'backend' }])
    panel.abrir()
    const executar = vi.fn(async () => ({ contido: 'Páxina 1', erro: false, imaxes: ['a1'] }))
    // Novo panel co executor que devolve imaxes.
    document.body.replaceChildren()
    const p2 = crearPanelAsistente({ t: (k, f) => f ?? k, api, executar, estadoPrograma: () => '', almacen: almacen() })
    await p2._enviar('mira a páxina')
    await vi.waitFor(() => expect(document.querySelector('.asis-ferr__img')).not.toBeNull())
    expect(document.querySelector('.asis-ferr__img').src).toContain('GRANDE-a1')
    expect(api.AdxuntoDataURI).toHaveBeenCalledWith('a1', 640)
    await tick()
  })

  it('sen IA configurada o clip está desactivado e non se aceptan ficheiros', async () => {
    const { api, raiz } = montar({ extraApi: { IAAsistenteEstado: vi.fn(async () => ({ configurada: false, provedor: '', preset: '', modelo: '' })) } })
    await vi.waitFor(() => expect(raiz.querySelector('.asistente__aviso').hidden).toBe(false))
    expect(raiz.querySelector('.asistente__clip').disabled).toBe(true)
    const input = raiz.querySelector('input[type=file]')
    Object.defineProperty(input, 'files', { value: [ficheiro('a.pdf')], configurable: true })
    input.dispatchEvent(new Event('change'))
    await tick()
    expect(api.AdxuntoGardar).not.toHaveBeenCalled()
  })
})

describe('aviso de IA sen configurar', () => {
  it('sen abrirOpcions (paneis incrustados) non escribe «null» no aviso', async () => {
    const api = { IAAsistenteEstado: vi.fn(async () => ({ configurada: false })), IAFerramentas: vi.fn(async () => []), IATurnoAsistente: vi.fn() }
    const panel = crearPanelAsistente({ t: (k, f) => f ?? k, api, executar: vi.fn(), estadoPrograma: () => '', almacen: almacen(), montarEn: document.body.appendChild(document.createElement('div')) })
    panel.abrir()
    await tick()
    const aviso = document.querySelector('.asistente__aviso')
    expect(aviso.hidden).toBe(false)
    expect(aviso.textContent).not.toMatch(/null/)
    expect(aviso.querySelector('button')).toBeNull()
  })
})
