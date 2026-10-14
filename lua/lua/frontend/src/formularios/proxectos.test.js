// @vitest-environment jsdom
import { describe, it, expect, vi, afterEach } from 'vitest'
import { crearInicio, opcionsFerramentas, nomeCartafol, corProxecto, cantasCaben, CORES_PROXECTO } from './proxectos.js'

afterEach(() => { document.body.replaceChildren(); vi.restoreAllMocks() })
const t = (k, fb, v) => String(fb).replace(/\{(\w+)\}/g, (_, n) => v?.[n] ?? '')
const P = (id, nome, ubicacion = '') => ({ id, nome, materia: '', creado: '2026-10-03', ferramentas: [], activo: false, boveda: false, fontes: 0, wiki: 0, ubicacion, ruta: (ubicacion || '/cfg/lua/proxectos') + '/' + (ubicacion ? nome : id) })

function montar(extra = {}) {
  const estado = { proxectos: [P('a', 'Inglés 1', '/home/u/Clases/Inglés'), P('b', 'Mates'), P('c', 'Inglés 2', '/home/u/Clases/Inglés')] }
  const accions = {
    listar: vi.fn(async () => estado.proxectos),
    formularios: vi.fn(async () => []),
    integradas: [],
    crear: vi.fn(async () => { const p = P('n', 'Novo'); estado.proxectos = [...estado.proxectos, p]; return p }),
    abrir: vi.fn(async () => {}), pechar: vi.fn(async () => {}),
    eliminar: vi.fn(async (id) => { estado.proxectos = estado.proxectos.filter((p) => p.id !== id) }),
    buxo: vi.fn(async () => ({ activo: false, instalado: false, iaConfigurada: false })),
    abrirBuxo: vi.fn(),
    escollerCartafol: vi.fn(async () => '/media/usb/Clases'),
    mover: vi.fn(async (id, destino) => { estado.proxectos = estado.proxectos.map((p) => (p.id === id ? { ...p, ubicacion: destino, ruta: destino + '/' + p.nome } : p)) }),
    fixarCor: vi.fn(async (id, cor) => { estado.proxectos = estado.proxectos.map((p) => (p.id === id ? { ...p, cor } : p)) }),
    confirmar: vi.fn(async () => true),
    ...extra,
  }
  const esquerda = document.createElement('div')
  const dereita = document.createElement('div')
  document.body.append(esquerda, dereita)
  const inicio = crearInicio({ t, accions, esquerda, dereita })
  inicio.recargar()
  return { accions, estado, esquerda, dereita, inicio }
}
const tarxetas = () => [...document.querySelectorAll('.prx-tarxeta strong')].map((e) => e.textContent)
const despregar = (nome) => [...document.querySelectorAll('.prx-tarxeta__cab')].find((b) => b.textContent.includes(nome)).click()
const botonNovo = (txt) => [...document.querySelectorAll('.prx-novo button')].find((b) => b.textContent === txt)
const botonCarp = (txt) => [...document.querySelectorAll('.prx-carpetas .bib-carp')].find((b) => b.textContent.startsWith(txt))

describe('proxectos: cartafoles', () => {
  it('agrupa polo cartafol do equipo onde está cada proxecto', async () => {
    montar()
    await vi.waitFor(() => expect(tarxetas().length).toBe(3))
    expect(botonCarp('Todos').textContent).toMatch(/3/)
    expect(botonCarp('Inglés').title).toBe('/home/u/Clases/Inglés')
    botonCarp('Inglés').click()
    expect(tarxetas()).toEqual(['Inglés 1', 'Inglés 2'])
    botonCarp('Cartafol de Lúa').click()
    expect(tarxetas()).toEqual(['Mates'])
    expect(document.querySelector('.prx-carpetas .bib-nova')).toBe(null)
  })
  it('«Escoller cartafol…» leva o proxecto ao cartafol escollido e a lista segue con el', async () => {
    const { accions } = montar()
    await vi.waitFor(() => expect(tarxetas().length).toBe(3))
    botonCarp('Cartafol de Lúa').click()
    despregar('Mates')
    expect(document.querySelector('.prx-tarxeta--aberta').textContent).toContain('/cfg/lua/proxectos/b')
    document.querySelector('.prx-mover').click()
    await vi.waitFor(() => expect(accions.mover).toHaveBeenCalledWith('b', '/media/usb/Clases'))
    await vi.waitFor(() => expect(botonCarp('Clases').getAttribute('aria-pressed')).toBe('true'))
    expect(tarxetas()).toEqual(['Mates'])
  })
  it('cancelar o selector non move nada', async () => {
    const { accions } = montar({ escollerCartafol: vi.fn(async () => '') })
    await vi.waitFor(() => expect(tarxetas().length).toBe(3))
    despregar('Mates')
    document.querySelector('.prx-mover').click()
    await vi.waitFor(() => expect(accions.escollerCartafol).toHaveBeenCalled())
    expect(accions.mover).not.toHaveBeenCalled()
  })
  it('eliminar un proxecto pide confirmación', async () => {
    const { accions } = montar()
    await vi.waitFor(() => expect(tarxetas().length).toBe(3))
    despregar('Mates')
    document.querySelector('.prx-eliminar').click()
    await vi.waitFor(() => expect(accions.eliminar).toHaveBeenCalledWith('b'))
    expect(accions.confirmar.mock.calls[0][0]).toMatch(/Mates/)
  })
  it('os proxectos saen pregados e desprégase ao premer', async () => {
    montar()
    await vi.waitFor(() => expect(tarxetas().length).toBe(3))
    expect(document.querySelector('.prx-acc')).toBe(null)
    expect(document.body.textContent).not.toContain('Ferramentas…')
    despregar('Mates')
    expect(document.querySelectorAll('.prx-acc').length).toBe(1)
    expect(document.querySelector('.prx-tarxeta__cab[aria-expanded="true"]').textContent).toContain('Mates')
    despregar('Mates')
    expect(document.querySelector('.prx-acc')).toBe(null)
  })
  it('crear un proxecto só pide o nome: nace baleiro e non se abre ata configuralo', async () => {
    const { accions } = montar()
    await vi.waitFor(() => expect(tarxetas().length).toBe(3))
    expect(document.querySelector('.prx-novo input[type=checkbox]')).toBe(null)
    const nome = document.querySelector('.prx-novo input')
    nome.value = 'Física 2º'
    botonNovo('Crear').click()
    await vi.waitFor(() => expect(accions.crear).toHaveBeenCalledWith('Física 2º', ''))
    await vi.waitFor(() => expect(botonNovo('Abrir proxecto')).toBeTruthy())
    expect(accions.abrir).not.toHaveBeenCalled()
    expect(document.querySelector('.prx-tarxeta__cab[aria-expanded="true"]').textContent).toContain('Novo')
    botonNovo('Abrir proxecto').click()
    await vi.waitFor(() => expect(accions.abrir).toHaveBeenCalledWith('n'))
  })
  it('despois de crear, escóllese o cartafol e as fontes de coñecemento antes de abrir', async () => {
    const { accions } = montar({ buxo: vi.fn(async () => ({ activo: true, instalado: true, iaConfigurada: true })) })
    await vi.waitFor(() => expect(tarxetas().length).toBe(3))
    botonCarp('Inglés').click()
    document.querySelector('.prx-novo input').value = 'Inglés 3'
    botonNovo('Crear').click()
    await vi.waitFor(() => expect(accions.mover).toHaveBeenCalledWith('n', '/home/u/Clases/Inglés'))
    await vi.waitFor(() => expect(document.querySelector('.prx-novo .prx-ruta').textContent).toBe('/home/u/Clases/Inglés/Novo'))
    botonNovo('Escoller cartafol…').click()
    await vi.waitFor(() => expect(accions.mover).toHaveBeenCalledWith('n', '/media/usb/Clases'))
    await vi.waitFor(() => expect(document.querySelector('.prx-novo .prx-ruta').textContent).toBe('/media/usb/Clases/Novo'))
    botonNovo('Fontes de coñecemento').click()
    await vi.waitFor(() => expect(accions.abrirBuxo).toHaveBeenCalledWith('n'))
    expect(accions.abrir).not.toHaveBeenCalled()
    botonNovo('Agora non').click()
    await vi.waitFor(() => expect(botonNovo('Crear')).toBeTruthy())
    expect(accions.abrir).not.toHaveBeenCalled()
  })
  it('nomeCartafol colle o último tramo da ruta', () => {
    expect(nomeCartafol('/home/u/Clases/')).toBe('Clases')
    expect(nomeCartafol('C:\\Users\\u\\Clases')).toBe('Clases')
  })
  it('opcionsFerramentas pon as integradas antes dos formularios', () => {
    expect(opcionsFerramentas([{ id: 'actas', nome: 'Actas' }], [{ id: 'x', nome: 'X', icona: 'i' }]).map((o) => o.id)).toEqual(['actas', 'form:x'])
  })
})

describe('inicio: os dous paneis', () => {
  it('«Novo proxecto» vai no panel da esquerda, enriba dos proxectos', async () => {
    const { esquerda, dereita } = montar()
    await vi.waitFor(() => expect(tarxetas().length).toBe(3))
    const novo = esquerda.querySelector('.prx-novo')
    const lista = esquerda.querySelector('.prx-lista')
    expect(novo.compareDocumentPosition(lista) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    expect(esquerda.querySelectorAll('.prx-tarxeta').length).toBe(3)
    expect(dereita.querySelectorAll('.prx-tarxeta').length).toBe(0)
  })
  it('os proxectos que non caben á esquerda seguen no panel da dereita, en orde', async () => {
    const { esquerda, dereita, inicio } = montar()
    await vi.waitFor(() => expect(tarxetas().length).toBe(3))
    Object.defineProperty(esquerda, 'clientHeight', { configurable: true, value: 250 })
    Object.defineProperty(dereita, 'clientHeight', { configurable: true, value: 600 })
    vi.spyOn(HTMLElement.prototype, 'offsetHeight', 'get').mockReturnValue(100)
    inicio.repartir()
    const nomes = (c) => [...c.querySelectorAll('.prx-tarxeta strong')].map((e) => e.textContent)
    expect(nomes(esquerda)).toEqual(['Inglés 1', 'Mates'])
    expect(nomes(dereita)).toEqual(['Inglés 2'])
  })
  it('cantasCaben conta as tarxetas que entran co oco entre elas', () => {
    expect(cantasCaben([100, 100, 100], 250)).toBe(2)
    expect(cantasCaben([100, 100], 205, 10)).toBe(1)
    expect(cantasCaben([100, 100], 210, 10)).toBe(2)
    expect(cantasCaben([300], 200)).toBe(0)
    expect(cantasCaben([], 200)).toBe(0)
  })
  it('sen proxectos, o panel da dereita queda baleiro e a esquerda di como empezar', async () => {
    const { dereita } = montar({ listar: vi.fn(async () => []) })
    await vi.waitFor(() => expect(document.querySelector('.bib-baleira')).toBeTruthy())
    expect(dereita.textContent).toBe('')
  })
  it('abrir un proxecto desde a súa tarxeta', async () => {
    const { accions } = montar()
    await vi.waitFor(() => expect(tarxetas().length).toBe(3))
    despregar('Mates')
    ;[...document.querySelectorAll('.prx-acc button')].find((b) => b.textContent === 'Abrir').click()
    await vi.waitFor(() => expect(accions.abrir).toHaveBeenCalledWith('b'))
  })
})

describe('inicio: a cor de cada proxecto', () => {
  it('cada proxecto ten unha cor por defecto estable e a súa, se a escolleu', () => {
    expect(CORES_PROXECTO).toContain(corProxecto({ id: 'abc' }))
    expect(corProxecto({ id: 'abc' })).toBe(corProxecto({ id: 'abc', cor: '' }))
    expect(corProxecto({ id: 'abc', cor: '#FF0000' })).toBe('#ff0000')
    expect(corProxecto({ id: 'abc', cor: 'red;}' })).toBe(corProxecto({ id: 'abc' }))
  })
  it('a tarxeta leva a cor do proxecto', async () => {
    montar({ listar: vi.fn(async () => [{ ...P('b', 'Mates'), cor: '#123456' }]) })
    await vi.waitFor(() => expect(tarxetas().length).toBe(1))
    expect(document.querySelector('.prx-tarxeta').style.getPropertyValue('--cor-proxecto')).toBe('#123456')
  })
  it('escoller unha cor gárdaa e «Cor por defecto» quítaa', async () => {
    const { accions } = montar()
    await vi.waitFor(() => expect(tarxetas().length).toBe(3))
    despregar('Mates')
    const entrada = document.querySelector('.prx-tarxeta--aberta input[type=color]')
    expect(entrada.getAttribute('aria-label')).toMatch(/Mates/)
    expect(document.querySelector('.prx-cor__defecto')).toBe(null)
    entrada.value = '#00aa55'
    entrada.dispatchEvent(new Event('change'))
    await vi.waitFor(() => expect(accions.fixarCor).toHaveBeenCalledWith('b', '#00aa55'))
    await vi.waitFor(() => expect(document.querySelector('.prx-tarxeta--aberta').style.getPropertyValue('--cor-proxecto')).toBe('#00aa55'))
    document.querySelector('.prx-cor__defecto').click()
    await vi.waitFor(() => expect(accions.fixarCor).toHaveBeenCalledWith('b', ''))
  })
})

describe('inicio: fontes de coñecemento', () => {
  const botonFontes = () => [...document.querySelectorAll('.prx-tarxeta--aberta button')].find((b) => b.textContent === 'Fontes de coñecemento')
  const estados = [
    [{ activo: false, instalado: true, iaConfigurada: true }, /Opcións → Módulos/],
    [{ activo: true, instalado: true, iaConfigurada: false }, /IA/],
    [{ activo: true, instalado: false, iaConfigurada: true }, /instalado/],
  ]
  for (const [estado, motivo] of estados) {
    it(`sae en cada proxecto aínda que Buxo non se poida usar, e di por que (${JSON.stringify(estado)})`, async () => {
      const { accions } = montar({ buxo: vi.fn(async () => estado) })
      await vi.waitFor(() => expect(tarxetas().length).toBe(3))
      despregar('Mates')
      expect(botonFontes()).toBeTruthy()
      expect(botonFontes().disabled).toBe(false)
      botonFontes().click()
      await vi.waitFor(() => expect(document.querySelector('.bib-erro').hidden).toBe(false))
      expect(document.querySelector('.bib-erro').textContent).toMatch(motivo)
      expect(accions.abrirBuxo).not.toHaveBeenCalled()
    })
  }
  it('con Buxo listo abre a bóveda do proxecto', async () => {
    const { accions } = montar({ buxo: vi.fn(async () => ({ activo: true, instalado: true, iaConfigurada: true })) })
    await vi.waitFor(() => expect(tarxetas().length).toBe(3))
    despregar('Inglés 2')
    botonFontes().click()
    await vi.waitFor(() => expect(accions.abrirBuxo).toHaveBeenCalledWith('c'))
  })
  it('tamén sae ao configurar un proxecto novo', async () => {
    montar()
    await vi.waitFor(() => expect(tarxetas().length).toBe(3))
    document.querySelector('.prx-novo input').value = 'Física'
    botonNovo('Crear').click()
    await vi.waitFor(() => expect(botonNovo('Fontes de coñecemento')).toBeTruthy())
  })
})

