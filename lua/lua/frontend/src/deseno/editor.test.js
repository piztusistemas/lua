// @vitest-environment jsdom
//
// Probas do bloque «Deseño» (deseno/editor.js) e da súa integración no editor
// de plantillas: letra, tamaño, barras, interruptores, elementos con LaTeX
// propio, desfacer, aviso de LaTeX libre, variantes e inspector da vista previa.
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { montarBloqueDeseno } from './editor.js'
import { montarEditorPlantilla } from '../plantillas-editor.js'

const t = (clave, fallback, vars) => {
  let val = fallback ?? clave
  if (vars) for (const k in vars) val = val.replaceAll(`{${k}}`, vars[k])
  return val
}

function montar(inicial = {}, extra = {}) {
  let deseno = inicial
  const nodo = document.createElement('div')
  document.body.replaceChildren(nodo)
  const cambiar = vi.fn((d) => { deseno = d })
  const api = {
    combinacions: vi.fn(async () => null),
    estiloElemento: vi.fn(async () => '\\newcommand{\\tarefa}[2]{XERADO}'),
    coresImaxe: vi.fn(async () => ['0E7490', 'DC2626']),
    importarDeseno: vi.fn(async () => ({ deseno: { letra: 'elegante', corPrincipal: 'B91C1C' }, explicacion: ['de proba'] })),
    ...extra.api,
  }
  const ui = montarBloqueDeseno(nodo, { t, api, ler: () => deseno, cambiar, ...extra })
  return { nodo, ui, cambiar, api, deseno: () => deseno }
}

const botonCo = (nodo, texto) => [...nodo.querySelectorAll('button')].find((b) => b.textContent.includes(texto))

describe('bloque Deseño', () => {
  beforeEach(() => { document.body.replaceChildren() })

  it('escoller unha letra e o tamaño cambia o deseño, e desfacer volve atrás', () => {
    const { nodo, deseno, cambiar } = montar()
    nodo.querySelector('[role="radio"]:nth-child(3)').click() // Elegante
    expect(deseno().letra).toBe('elegante')
    botonCo(nodo, '+').click()
    expect(deseno().tamano).toBe(11.5)
    botonCo(nodo, 'Desfacer').click()
    expect(deseno().tamano).toBeUndefined()
    expect(deseno().letra).toBe('elegante')
    expect(cambiar).toHaveBeenCalledTimes(3)
  })

  it('a letra escollida sae marcada', () => {
    const { nodo } = montar({ letra: 'lexible' })
    const marcada = nodo.querySelector('[role="radio"][aria-checked="true"]')
    expect(marcada.textContent).toMatch(/lexible/i)
  })

  it('as barras e os interruptores cambian os valores', () => {
    const { nodo, deseno } = montar()
    const barra = nodo.querySelectorAll('.deseno-barra input')[1] // aire
    barra.value = '2'
    barra.dispatchEvent(new Event('change'))
    expect(deseno().interlinado).toBe(1.35)
    const bn = nodo.querySelector('.deseno-interruptor input')
    bn.checked = true
    bn.dispatchEvent(new Event('change'))
    expect(deseno().brancoNegro).toBe(true)
  })

  it('o cadeado fixa un campo e as barras xa non o tocan', () => {
    const { nodo, deseno } = montar({ letra: 'lexible' })
    nodo.querySelector('.deseno-cadeado').click() // o da letra
    expect(deseno().fixados).toEqual(['letra'])
    const formal = nodo.querySelector('.deseno-barra input')
    formal.value = '2'
    formal.dispatchEvent(new Event('change'))
    expect(deseno().letra).toBe('lexible')
  })

  it('un elemento ábrese e pode levar o seu LaTeX, que parte do xerado', async () => {
    const { nodo, ui, deseno, api } = montar()
    ui.abrirElemento('tarefas')
    const det = nodo.querySelector('[data-elemento="tarefas"] .deseno-latex')
    expect(det).toBeTruthy()
    det.open = true
    det.dispatchEvent(new Event('toggle'))
    await vi.waitFor(() => expect(det.querySelector('textarea').value).toMatch(/XERADO/))
    expect(api.estiloElemento).toHaveBeenCalled()
    det.querySelector('textarea').value = '\\newcommand{\\tarefa}[2]{PROPIO}'
    botonCo(det, 'Usar este LaTeX').click()
    await vi.waitFor(() => expect(deseno().elementos?.tarefas?.latex).toMatch(/PROPIO/))
    // Marca visible na lista.
    expect(nodo.querySelector('[data-elemento="tarefas"] .deseno-elemento__marca').textContent).toMatch(/LaTeX/)
  })

  it('avisa se a plantilla ten LaTeX libre e deixa pasar ao editor visual', () => {
    let libre = '% libre'
    const quitar = vi.fn(() => { libre = '' })
    const { nodo } = montar({}, { estiloLibre: () => libre, quitarEstiloLibre: quitar })
    expect(nodo.querySelector('.deseno-aviso')).toBeTruthy()
    botonCo(nodo, 'Pasar ao editor visual').click()
    expect(quitar).toHaveBeenCalled()
    expect(nodo.querySelector('.deseno-aviso')).toBeNull()
  })

  it('aplica un JSON pegado doutra IA validándoo', () => {
    const { nodo, deseno } = montar()
    nodo.querySelector('.deseno-detalles').open = true
    const area = nodo.querySelector('.deseno-json textarea')
    area.value = 'Aquí tes:\n{"letra": "moderna", "tamano": 12}\n'
    botonCo(nodo, 'Aplicar este JSON').click()
    expect(deseno()).toMatchObject({ letra: 'moderna', tamano: 12 })
    // O bloque volveuse pintar: a caixa é outra.
    nodo.querySelector('.deseno-json textarea').value = '{"tamano": 99}'
    botonCo(nodo, 'Aplicar este JSON').click()
    expect(nodo.textContent).toMatch(/fóra de rango/)
    expect(deseno().tamano).toBe(12)
  })

  it('variantes: compila as dúas propostas e escoller unha cámbiaa', async () => {
    const previsualizarCon = vi.fn(async () => ({ pageImages: ['data:image/png;base64,AA'], marcas: [] }))
    const { nodo, deseno } = montar({}, { previsualizarCon })
    botonCo(nodo, 'Probar variantes').click()
    await vi.waitFor(() => expect(previsualizarCon).toHaveBeenCalledTimes(2))
    const dialogo = document.querySelector('.deseno-variantes')
    await vi.waitFor(() => expect(dialogo.querySelectorAll('img').length).toBe(2))
    const escoller = [...dialogo.querySelectorAll('button')].filter((b) => b.textContent === 'Prefiro esta')
    escoller[1].click()
    expect(deseno().letra).toBeTruthy()
    // Pasa á seguinte rolda (a cor) e compila outras dúas.
    await vi.waitFor(() => expect(previsualizarCon).toHaveBeenCalledTimes(4))
    botonCo(dialogo, 'Rematar').click()
    expect(document.querySelector('.deseno-variantes')).toBeNull()
  })
})

describe('melloras do bloque', () => {
  beforeEach(() => { document.body.replaceChildren() })

  it('as frechas moven a escolla de letra (só a marcada recibe o tabulador)', async () => {
    const { nodo, deseno } = montar()
    const radios = [...nodo.querySelectorAll('.deseno-letras [role="radio"]')]
    expect(radios.filter((r) => r.tabIndex === 0)).toHaveLength(1)
    radios[0].focus()
    radios[0].dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowRight', bubbles: true }))
    expect(deseno().letra).toBe('clasica')
  })

  it('escoller un modelo de portada e o logotipo, e a cabeceira por zonas', () => {
    const { nodo, deseno, cambiar } = montar()
    nodo.querySelector('[data-radio="portada-oficial"]').click()
    expect(deseno().portada).toBe('oficial')
    expect(cambiar).toHaveBeenLastCalledWith(expect.anything(), 'portada')
    const sel = [...nodo.querySelectorAll('select')].find((s) => [...s.options].some((o) => o.value === 'dereita'))
    sel.value = 'dereita'
    sel.dispatchEvent(new Event('change'))
    expect(deseno().logoPortada).toBe('dereita')
    const engadir = () => nodo.querySelector('.deseno-folla--cabeceira .deseno-folla__zona--esquerda select')
    engadir().value = '@logo'
    engadir().dispatchEvent(new Event('change'))
    engadir().value = '{{CENTRO}}'
    engadir().dispatchEvent(new Event('change'))
    expect(deseno().cabeceira).toEqual({ esquerda: '@logo\n{{CENTRO}}' })
    const filete = nodo.querySelector('.deseno-folla--cabeceira input[type="checkbox"]')
    filete.checked = true
    filete.dispatchEvent(new Event('change'))
    expect(deseno().cabeceiraFilete).toBe(true)
    for (let i = 0; i < 2; i++) nodo.querySelector('.deseno-folla--cabeceira .deseno-peza__x').click()
    expect(deseno().cabeceira).toBeUndefined()
  })

  it('o LaTeX dunha parte compróbase antes de usalo e o erro sae ao carón', async () => {
    const comprobarElemento = vi.fn(async (d, id, latex) => (latex.includes('MAL') ? 'Undefined control sequence \\MAL' : ''))
    const { nodo, ui, deseno } = montar({}, { api: { comprobarElemento } })
    ui.abrirElemento('tarefas')
    const det = nodo.querySelector('[data-elemento="tarefas"] .deseno-latex')
    det.querySelector('textarea').value = '\\newcommand{\\tarefa}[2]{\\MAL}'
    botonCo(det, 'Usar este LaTeX').click()
    await vi.waitFor(() => expect(det.querySelector('.deseno-nota').textContent).toMatch(/Non compila/))
    expect(deseno().elementos).toBeUndefined()
    botonCo(det, 'Usar igualmente').click()
    expect(deseno().elementos.tarefas.latex).toMatch(/MAL/)
  })

  it('kit do centro: gardar, abrir e aplicar a todas', async () => {
    const api = {
      gardarDesenoFicheiro: vi.fn(async () => '/tmp/centro.luadeseno'),
      lerDesenoFicheiro: vi.fn(async () => ({ letra: 'academica' })),
      aplicarDesenoATodas: vi.fn(async () => 3),
    }
    const { nodo } = montar({ letra: 'clasica' }, { api, nomePlantilla: () => 'Centro' })
    botonCo(nodo, 'Gardar nun ficheiro').click()
    await vi.waitFor(() => expect(api.gardarDesenoFicheiro).toHaveBeenCalledWith({ letra: 'clasica' }, 'Centro'))
    await vi.waitFor(() => expect(nodo.textContent).toMatch(/centro\.luadeseno/))
    expect(botonCo(nodo, 'Abrir un ficheiro')).toBeTruthy()
    expect(botonCo(nodo, 'Aplicar a todas')).toBeTruthy()
  })
})

describe('editor de plantillas co deseño', () => {
  beforeEach(() => { document.body.replaceChildren(); localStorage.clear() })

  function apiFalsa() {
    return {
      listar: vi.fn(async () => ({ plantillas: [], activa: '' })),
      gardar: vi.fn(async (p) => ({ ...p, id: p.id || 'nova-id', imaxes: [] })),
      previsualizar: vi.fn(async () => ({
        pageImages: ['data:image/png;base64,AA'], warnings: [],
        marcas: [{ elemento: 'titulos', paxina: 1, x: 0.1, y: 0.1 }],
      })),
      fragmentosPorDefecto: vi.fn(async () => ({ portada: '', sinatura: '', introducion: '', estilo: '', textos: [] })),
    }
  }

  it('o bloque Deseño vai á vista e o deseño gárdase coa plantilla', async () => {
    const api = apiFalsa()
    const c = document.createElement('div')
    document.body.appendChild(c)
    const ed = await montarEditorPlantilla(c, { t, api, plantilla: { id: 'p1', nome: 'X', latex: '{{CORPO}}', variables: [], imaxes: [], ofimatica: {} } })
    const bloque = c.querySelector('section[data-bloque="deseno"]')
    expect(bloque.hidden).toBe(false)
    bloque.querySelector('[role="radio"]:nth-child(2)').click() // Clásica
    expect(ed.cambiado()).toBe(true)
    await ed.aplicar()
    expect(api.gardar.mock.calls[0][0].deseno).toEqual({ letra: 'clasica' })
  })

  it('a vista previa faise ao abrir, ten Cancelar mentres compila e resalta a parte baixo o rato', async () => {
    const api = apiFalsa()
    let soltar
    api.previsualizar = vi.fn(() => new Promise((r) => { soltar = () => r({ pageImages: ['data:image/png;base64,AA'], warnings: [], marcas: [{ elemento: 'titulos', paxina: 1, x: 0.1, y: 0.1 }] }) }))
    api.cancelarVista = vi.fn(async () => 1)
    const c = document.createElement('div')
    document.body.appendChild(c)
    await montarEditorPlantilla(c, { t, api, plantilla: { id: 'p1', nome: 'X', latex: '{{CORPO}}', variables: [], imaxes: [], ofimatica: {} } })
    await vi.waitFor(() => expect(api.previsualizar).toHaveBeenCalledTimes(1))
    const cancelar = c.querySelector('.plantilla-vista__cancelar')
    expect(cancelar.hidden).toBe(false)
    cancelar.click()
    expect(api.cancelarVista).toHaveBeenCalled()
    soltar()
    await vi.waitFor(() => expect(c.querySelector('.plantilla-vista__cancelar').hidden).toBe(true))
    const im = c.querySelector('.plantilla-previa__paxina')
    im.getBoundingClientRect = () => ({ left: 0, top: 0, width: 100, height: 100 })
    im.dispatchEvent(new MouseEvent('mousemove', { bubbles: true, clientX: 30, clientY: 11 }))
    const res = c.querySelector('.plantilla-previa__resalte')
    expect(res.hidden).toBe(false)
    expect(res.textContent).toBe('titulos') // (o t de proba non traduce)
  })

  it('unha plantilla nova abre tamén a identidade (precisa nome)', async () => {
    const c = document.createElement('div')
    document.body.appendChild(c)
    await montarEditorPlantilla(c, { t, api: apiFalsa(), plantilla: null })
    await new Promise((r) => requestAnimationFrame(r))
    expect(c.querySelector('section[data-bloque="identidade"]').hidden).toBe(false)
  })

  it('premer nun título da vista previa abre o elemento Títulos', async () => {
    const api = apiFalsa()
    const c = document.createElement('div')
    document.body.appendChild(c)
    await montarEditorPlantilla(c, { t, api, plantilla: { id: 'p1', nome: 'X', latex: '{{CORPO}}', variables: [], imaxes: [], ofimatica: {} } })
    ;[...c.querySelectorAll('button')].find((b) => b.textContent.includes('Vista previa') && !b.classList.contains('plantilla-cab-btn')).click()
    const im = await vi.waitFor(() => {
      const i = c.querySelector('.plantilla-previa__paxina')
      if (!i) throw new Error('sen páxina')
      return i
    })
    im.getBoundingClientRect = () => ({ left: 0, top: 0, width: 100, height: 100 })
    im.dispatchEvent(new MouseEvent('click', { bubbles: true, clientX: 30, clientY: 11 }))
    const el = c.querySelector('[data-elemento="titulos"] .deseno-elemento__cab')
    expect(el.getAttribute('aria-expanded')).toBe('true')
  })
})
