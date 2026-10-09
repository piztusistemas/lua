// @vitest-environment jsdom
import { describe, it, expect, afterEach, vi } from 'vitest'
import { createCronogramaEditor } from './cronograma-editor.js'
import { normalizarPlan, novoPlan, gardarTarefa } from './cronograma-modelo.js'

const HOXE = new Date(2025, 9, 15) // outubro de 2025

function plan0() {
  const p = normalizarPlan(novoPlan(HOXE)) // curso 2025/2026
  return p
}

function montar(planInicial = plan0(), opcions = {}) {
  const container = document.createElement('div')
  document.body.append(container)
  const cambios = []
  const ed = createCronogramaEditor(container, {
    onChange: (p, motivo) => cambios.push({ p, motivo }),
    hoxe: () => HOXE,
    ...opcions,
  })
  ed.setValue(planInicial)
  return { ed, container, cambios }
}

const modal = () => document.querySelector('.crono-modal')
const mes = (container, clave) => container.querySelector(`[data-mes="${clave}"]`)
const titulos = (container, clave) => [...mes(container, clave).querySelectorAll('.crono-tarefa__titulo')].map((n) => n.textContent)
const tick = () => new Promise((r) => setTimeout(r, 0))

// Engade unha tarefa polo botón do mes e a modal, como o usuario.
function engadirTarefa(container, clave, titulo, descricion = '') {
  mes(container, clave).querySelector('.crono-mes__cab .crono-btn').click()
  modal().querySelector('input[type=text]').value = titulo
  if (descricion) {
    const ta = modal().querySelector('textarea.crono-md')
    ta.value = descricion
    ta.dispatchEvent(new Event('input'))
  }
  ;[...modal().querySelectorAll('.modal-actions button')].find((b) => b.classList.contains('primary')).click()
}

afterEach(() => { document.body.replaceChildren() })

describe('lista de meses', () => {
  it('pinta un mes por cada mes do rango, en orde, e marca o actual', () => {
    const { container } = montar()
    const nomes = [...container.querySelectorAll('.crono-mes h3 span:first-child')].map((n) => n.textContent)
    expect(nomes).toEqual(['Setembro', 'Outubro', 'Novembro', 'Decembro', 'Xaneiro', 'Febreiro', 'Marzo', 'Abril', 'Maio', 'Xuño'])
    expect(container.querySelectorAll('.crono-mes--actual')).toHaveLength(1)
    expect(mes(container, '2025-10').classList.contains('crono-mes--actual')).toBe(true)
  })

  it('un curso novo ten o formulario aberto; un con tarefas, pechado', () => {
    const novo = montar().container
    expect(novo.querySelector('details').open).toBe(true)
    const cheo = montar(gardarTarefa(plan0(), '2025-09', { id: 'a', title: 'X', description: '', color: 'blue' })).container
    expect(cheo.querySelector('details').open).toBe(false)
  })

  it('con datas inválidas non pinta meses e di por que', () => {
    const p = plan0()
    p.config.endDate = '2000-01-01'
    const { container } = montar(p)
    expect(container.querySelectorAll('.crono-mes')).toHaveLength(0)
    expect(container.querySelector('.crono-erro').textContent).toMatch(/anterior/)
  })

  it('avisa das tarefas que quedan fóra do rango de datas', () => {
    const p = gardarTarefa(plan0(), '2030-01', { id: 'a', title: 'Lonxe', description: '', color: 'blue' })
    const { container } = montar(p)
    expect(container.querySelector('.crono-aviso').textContent).toMatch(/1 tarefa/)
  })
})

describe('datas', () => {
  it('cambiar unha data válida repinta os meses e avisa', () => {
    const { container, cambios } = montar()
    const fin = container.querySelectorAll('input[type=date]')[1]
    fin.value = '2025-12-20'
    fin.dispatchEvent(new Event('change'))
    expect(container.querySelectorAll('.crono-mes')).toHaveLength(4)
    expect(cambios.at(-1).p.config.endDate).toBe('2025-12-20')
  })

  it('unha data incompleta non pisa a boa', () => {
    const { container, cambios } = montar()
    const n = cambios.length
    const inicio = container.querySelectorAll('input[type=date]')[0]
    inicio.value = ''
    inicio.dispatchEvent(new Event('change'))
    expect(cambios.length).toBe(n)
    expect(container.querySelectorAll('.crono-mes')).toHaveLength(10)
    expect(container.querySelector('.crono-erro:not([hidden])').textContent).toMatch(/Completa/)
  })
})

describe('tarefas', () => {
  it('engadir unha tarefa dende a modal', () => {
    const { container, cambios } = montar()
    engadirTarefa(container, '2025-09', 'Avaliación inicial', '**Importante**')
    expect(titulos(container, '2025-09')).toEqual(['Avaliación inicial'])
    expect(modal()).toBeNull()
    const ultimo = cambios.at(-1)
    expect(ultimo.motivo).toBe('estrutura')
    expect(ultimo.p.monthsData['2025-09'].tasks[0]).toMatchObject({ title: 'Avaliación inicial', description: '**Importante**', color: 'blue' })
  })

  it('sen título non garda e avisa', () => {
    const { container, cambios } = montar()
    const n = cambios.length
    engadirTarefa(container, '2025-09', '   ')
    expect(modal()).not.toBeNull()
    expect(modal().querySelector('.crono-erro:not([hidden])').textContent).toMatch(/título/)
    expect(cambios.length).toBe(n)
  })

  it('o título con HTML móstrase coma texto, non se interpreta', () => {
    const { container } = montar()
    engadirTarefa(container, '2025-09', '<img src=x onerror=alert(1)>', '<script>alert(1)</script>')
    expect(container.querySelector('img[src=x]')).toBeNull()
    expect(container.querySelector('.crono-tarefa__titulo').textContent).toBe('<img src=x onerror=alert(1)>')
  })

  it('editar unha tarefa: a modal ábrese cos seus datos e gárdase o cambio', () => {
    const p = gardarTarefa(plan0(), '2025-09', { id: 'a', title: 'Vella', description: 'desc', color: 'rose' })
    const { container, cambios } = montar(p)
    container.querySelector('.crono-tarefa__corpo').click()
    expect(modal().querySelector('input[type=text]').value).toBe('Vella')
    expect(modal().querySelector('textarea.crono-md').value).toBe('desc')
    expect(modal().querySelector('.crono-cor--activa').title).toBe('Rosa')
    modal().querySelector('input[type=text]').value = 'Nova'
    modal().querySelectorAll('.crono-cor')[3 + 1].click() // purple
    ;[...modal().querySelectorAll('.modal-actions button')].find((b) => b.classList.contains('primary')).click()
    expect(cambios.at(-1).p.monthsData['2025-09'].tasks[0]).toMatchObject({ id: 'a', title: 'Nova', color: 'purple' })
  })

  it('cambiar o mes na modal move a tarefa', () => {
    const p = gardarTarefa(plan0(), '2025-09', { id: 'a', title: 'Móvese', description: '', color: 'blue' })
    const { container } = montar(p)
    container.querySelector('.crono-tarefa__corpo').click()
    const sel = modal().querySelector('select')
    sel.value = '2025-11'
    sel.dispatchEvent(new Event('change'))
    ;[...modal().querySelectorAll('.modal-actions button')].find((b) => b.classList.contains('primary')).click()
    expect(titulos(container, '2025-09')).toEqual([])
    expect(titulos(container, '2025-11')).toEqual(['Móvese'])
  })

  it('subir, baixar e duplicar', () => {
    let p = plan0()
    for (const x of ['a', 'b', 'c']) p = gardarTarefa(p, '2025-09', { id: x, title: x, description: '', color: 'blue' })
    const { container } = montar(p)
    const accion = (i, k) => mes(container, '2025-09').querySelectorAll('.crono-tarefa')[i].querySelectorAll('.crono-icona')[k].click()
    accion(0, 1) // baixar a
    expect(titulos(container, '2025-09')).toEqual(['b', 'a', 'c'])
    accion(2, 0) // subir c
    expect(titulos(container, '2025-09')).toEqual(['b', 'c', 'a'])
    accion(0, 2) // duplicar b
    expect(titulos(container, '2025-09')).toEqual(['b', 'b (copia)', 'c', 'a'])
    const primeira = mes(container, '2025-09').querySelectorAll('.crono-tarefa')[0].querySelectorAll('.crono-icona')
    expect(primeira[0].disabled).toBe(true) // a primeira non sobe
  })

  it('eliminar pide confirmación', async () => {
    const p = gardarTarefa(plan0(), '2025-09', { id: 'a', title: 'Borrar', description: '', color: 'blue' })
    const { container } = montar(p)
    const borrar = () => container.querySelector('.crono-tarefa .crono-icona:nth-child(4)').click()
    borrar()
    await tick()
    document.querySelector('.overlay--dialog .modal-actions button:first-child').click() // Cancelar
    await tick()
    expect(titulos(container, '2025-09')).toEqual(['Borrar'])
    borrar()
    await tick()
    document.querySelector('.overlay--dialog .modal-actions button.primary').click()
    await tick()
    expect(titulos(container, '2025-09')).toEqual([])
  })

  it('cancelar unha modal con cambios pide confirmación; sen cambios pecha directamente', async () => {
    const { container } = montar()
    mes(container, '2025-09').querySelector('.crono-mes__cab .crono-btn').click()
    ;[...modal().querySelectorAll('.modal-actions button')].find((b) => !b.classList.contains('primary')).click()
    await tick()
    expect(modal()).toBeNull() // sen cambios: pecha

    mes(container, '2025-09').querySelector('.crono-mes__cab .crono-btn').click()
    modal().querySelector('input[type=text]').value = 'a medias'
    ;[...modal().querySelectorAll('.modal-actions button')].find((b) => !b.classList.contains('primary')).click()
    await tick()
    expect(modal()).not.toBeNull()
    document.querySelector('.overlay--dialog:not(.crono-overlay) .modal-actions button:first-child').click() // Seguir editando
    await tick()
    expect(modal()).not.toBeNull()
  })
})

describe('barra de formato e vista previa', () => {
  it('a vista previa segue o texto', () => {
    const { container } = montar()
    mes(container, '2025-09').querySelector('.crono-mes__cab .crono-btn').click()
    const ta = modal().querySelector('textarea.crono-md')
    ta.value = '# Título\n\n- un\n- dous'
    ta.dispatchEvent(new Event('input'))
    expect(modal().querySelector('.crono-md-vista h4').textContent).toBe('Título')
    expect(modal().querySelectorAll('.crono-md-vista li')).toHaveLength(2)
  })

  it('o botón Negrita envolve a selección', () => {
    const { container } = montar()
    mes(container, '2025-09').querySelector('.crono-mes__cab .crono-btn').click()
    const ta = modal().querySelector('textarea.crono-md')
    ta.value = 'ola mundo'
    ta.setSelectionRange(4, 9)
    modal().querySelector('.crono-fmt--n').click()
    expect(ta.value).toBe('ola **mundo**')
  })

  it('inserir unha imaxe engade a referencia e gárdase co mapa de imaxes', async () => {
    const lerImaxe = vi.fn(async () => 'data:image/jpeg;base64,/9j/AAAA')
    const { container, cambios } = montar(plan0(), { lerImaxe })
    mes(container, '2025-09').querySelector('.crono-mes__cab .crono-btn').click()
    modal().querySelector('input[type=text]').value = 'Con imaxe'
    const ficheiro = modal().querySelector('input[type=file]')
    const arquivo = new File(['x'], 'foto (1).jpg', { type: 'image/jpeg' })
    Object.defineProperty(ficheiro, 'files', { value: [arquivo], configurable: true })
    ficheiro.dispatchEvent(new Event('change'))
    await vi.waitFor(() => expect(modal().querySelector('textarea.crono-md').value).toMatch(/!\[foto 1\]\(img:i\w+ =60%\)/))
    expect(lerImaxe).toHaveBeenCalledWith(arquivo, { maxLado: 1600, saida: 'jpeg' })
    expect(modal().querySelector('.crono-md-vista img')).not.toBeNull()
    ;[...modal().querySelectorAll('.modal-actions button')].find((b) => b.classList.contains('primary')).click()
    const t = cambios.at(-1).p.monthsData['2025-09'].tasks[0]
    expect(Object.keys(t.images)).toHaveLength(1)
    expect(t.description).toContain(`img:${Object.keys(t.images)[0]}`)
  })

  it('unha imaxe que xa non se referencia non se garda', async () => {
    const lerImaxe = vi.fn(async () => 'data:image/png;base64,AAAA')
    const { container, cambios } = montar(plan0(), { lerImaxe })
    mes(container, '2025-09').querySelector('.crono-mes__cab .crono-btn').click()
    modal().querySelector('input[type=text]').value = 'Sen imaxe'
    const ficheiro = modal().querySelector('input[type=file]')
    Object.defineProperty(ficheiro, 'files', { value: [new File(['x'], 'a.png', { type: 'image/png' })], configurable: true })
    ficheiro.dispatchEvent(new Event('change'))
    await vi.waitFor(() => expect(modal().querySelector('textarea.crono-md').value).toContain('img:'))
    modal().querySelector('textarea.crono-md').value = 'borrei a referencia'
    ;[...modal().querySelectorAll('.modal-actions button')].find((b) => b.classList.contains('primary')).click()
    expect(cambios.at(-1).p.monthsData['2025-09'].tasks[0].images).toBeUndefined()
  })

  it('un erro ao ler a imaxe móstrase na modal', async () => {
    const lerImaxe = vi.fn(async () => { throw new Error('formato raro') })
    const { container } = montar(plan0(), { lerImaxe })
    mes(container, '2025-09').querySelector('.crono-mes__cab .crono-btn').click()
    const ficheiro = modal().querySelector('input[type=file]')
    Object.defineProperty(ficheiro, 'files', { value: [new File(['x'], 'a.bmp')], configurable: true })
    ficheiro.dispatchEvent(new Event('change'))
    await vi.waitFor(() => expect(modal().querySelector('.crono-erro:not([hidden])').textContent).toMatch(/formato raro/))
  })
})

describe('logotipo', () => {
  it('o logotipo faise dende a plantilla: os datos do curso non teñen onde escollelo, pero un que xa haxa consérvase', () => {
    const p = { ...plan0(), config: { ...plan0().config, schoolLogo: 'data:image/png;base64,LOGO' } }
    const { container, ed } = montar(p)
    expect(container.querySelector('input[type=file]')).toBeNull()
    expect(container.querySelector('.crono-logo')).toBeNull()
    expect(ed.getValue().config.schoolLogo).toBe('data:image/png;base64,LOGO')
  })
})

describe('curso bloqueado', () => {
  it('deshabilita os botóns e a modal só deixa ver', () => {
    const p = gardarTarefa(plan0(), '2025-09', { id: 'a', title: 'Fixa', description: 'x', color: 'blue' })
    p.isLocked = true
    const { container } = montar(p)
    expect(mes(container, '2025-09').querySelector('.crono-mes__cab .crono-btn').disabled).toBe(true)
    expect([...container.querySelectorAll('.crono-tarefa .crono-icona')].every((b) => b.disabled)).toBe(true)
    expect(container.querySelector('.crono-config__corpo input[type=text]').disabled).toBe(true)
    container.querySelector('.crono-tarefa__corpo').click()
    expect(modal().querySelector('input[type=text]').disabled).toBe(true)
    expect(modal().querySelector('.modal-actions .primary')).toBeNull()
  })

  it('o interruptor de bloqueo cambia o curso', () => {
    const { container, cambios } = montar()
    const check = container.querySelector('.crono-camp--check input')
    check.checked = true
    check.dispatchEvent(new Event('change'))
    expect(cambios.at(-1).p.isLocked).toBe(true)
    expect(container.classList.contains('crono-editor--bloqueado')).toBe(true)
  })
})

describe('campos do curso', () => {
  it('escribir nun campo avisa cun cambio de tipo "texto" sen repintar (non perde o foco)', () => {
    const { container, cambios } = montar()
    const campo = [...container.querySelectorAll('input[type=text]')].find((i) => i.placeholder.startsWith('IES'))
    campo.value = 'IES Castelao'
    campo.dispatchEvent(new Event('input'))
    expect(cambios.at(-1)).toMatchObject({ motivo: 'texto' })
    expect(cambios.at(-1).p.config.schoolName).toBe('IES Castelao')
    expect(container.contains(campo)).toBe(true)
  })
})

describe('pantalla completa da tarefa', () => {
  const botons = () => [...modal().querySelectorAll('.modal-actions button')]
  const botonAplicar = () => modal().querySelector('.tarefa-aplicar')
  const abrirNova = (container) => mes(container, '2025-09').querySelector('.crono-mes__cab .crono-btn').click()

  it('ocupa a pantalla e ten unha X vermella na cabeceira', () => {
    const { container } = montar()
    abrirNova(container)
    expect(modal().classList.contains('tarefa-pantalla')).toBe(true)
    const x = modal().querySelector('.tarefa-cab .tarefa-x')
    expect(x).not.toBeNull()
    expect(x.getAttribute('aria-label')).toBeTruthy()
  })

  it('a X pecha sen preguntar se non hai cambios, e pregunta se os hai', async () => {
    const { container } = montar()
    abrirNova(container)
    modal().querySelector('.tarefa-x').click()
    await tick()
    expect(modal()).toBeNull()

    abrirNova(container)
    modal().querySelector('input[type=text]').value = 'a medias'
    modal().querySelector('.tarefa-x').click()
    await tick()
    expect(modal()).not.toBeNull()
    document.querySelector('.overlay--dialog:not(.crono-overlay) .modal-actions button:first-child').click() // Seguir editando
    await tick()
    expect(modal()).not.toBeNull()
  })

  it('Aplicar garda no curso SEN pechar; despois Gardar actualiza a mesma tarefa', () => {
    const { container, cambios } = montar()
    abrirNova(container)
    modal().querySelector('input[type=text]').value = 'Primeira'
    botonAplicar().click()
    expect(modal()).not.toBeNull()
    expect(titulos(container, '2025-09')).toEqual(['Primeira'])
    expect(cambios.at(-1).motivo).toBe('estrutura')
    expect(modal().querySelector('h2').textContent).toMatch(/Editar/)

    modal().querySelector('input[type=text]').value = 'Segunda'
    ;[...botons()].find((b) => b.classList.contains('primary')).click()
    expect(modal()).toBeNull()
    expect(titulos(container, '2025-09')).toEqual(['Segunda']) // a mesma tarefa, non unha copia
  })

  it('Aplicar sen título non garda e avisa', () => {
    const { container, cambios } = montar()
    const n = cambios.length
    abrirNova(container)
    botonAplicar().click()
    expect(cambios.length).toBe(n)
    expect(modal().querySelector('.crono-erro:not([hidden])').textContent).toMatch(/título/)
  })

  it('tras Aplicar, pechar co X xa non pregunta (o cambio está gardado)', async () => {
    const { container } = montar()
    abrirNova(container)
    modal().querySelector('input[type=text]').value = 'Gardada'
    botonAplicar().click()
    modal().querySelector('.tarefa-x').click()
    await tick()
    expect(modal()).toBeNull()
  })

  it('Aplicar tamén move a tarefa se se cambia o mes', () => {
    const { container } = montar()
    abrirNova(container)
    modal().querySelector('input[type=text]').value = 'Viaxeira'
    botonAplicar().click()
    const sel = modal().querySelector('select')
    sel.value = '2025-11'
    sel.dispatchEvent(new Event('change'))
    botonAplicar().click()
    expect(titulos(container, '2025-09')).toEqual([])
    expect(titulos(container, '2025-11')).toEqual(['Viaxeira'])
  })

  it('con xanela propia dispoñible, a tarefa ábrese alí e non nesta xanela', async () => {
    const abertas = []
    const { container } = montar(plan0(), { abrirXanelaTarefa: async (datos) => { abertas.push(datos); return true } })
    abrirNova(container)
    await tick()
    expect(abertas).toHaveLength(1)
    expect(abertas[0]).toMatchObject({ nova: true, clave: '2025-09', bloq: false })
    expect(abertas[0].meses.length).toBe(10)
    expect(modal()).toBeNull()
  })

  it('se a xanela propia non se pode abrir, cae na pantalla completa desta xanela', async () => {
    const { container } = montar(plan0(), { abrirXanelaTarefa: async () => false })
    abrirNova(container)
    await tick()
    expect(modal()).not.toBeNull()
    const { container: c2 } = montar(plan0(), { abrirXanelaTarefa: async () => { throw new Error('sen xanelas') } })
    document.querySelector('.tarefa-x').click()
    await tick()
    abrirNova(c2)
    await tick()
    expect(modal()).not.toBeNull()
  })

  it('aplicarTarefaExterna mete no curso a tarefa gardada noutra xanela', () => {
    const { ed, container, cambios } = montar()
    ed.aplicarTarefaExterna({ tarefa: { id: 'x1', title: 'De fóra', description: '', color: 'rose' }, claveOrixe: '2025-09', mesDestino: '2025-10' })
    expect(titulos(container, '2025-10')).toEqual(['De fóra'])
    expect(cambios.at(-1).motivo).toBe('estrutura')
  })
})

describe('asistente de IA na pantalla da tarefa', () => {
  const boton = () => modal().querySelector('.tarefa-asistente')
  const panel = () => modal().querySelector('.asistente--incrustado')
  const abrirNova = (container) => mes(container, '2025-09').querySelector('.crono-mes__cab .crono-btn').click()

  function asistente(turnos) {
    const api = {
      IAAsistenteEstado: vi.fn(async () => ({ configurada: true, provedor: 'openai', preset: 'openai', modelo: 'm' })),
      IAFerramentasDoAmbito: vi.fn(async () => [
        { nome: 'editor_ler', tipo: 'lectura', onde: 'frontend' }, { nome: 'editor_cambiar', tipo: 'escritura', onde: 'frontend' },
        { nome: 'web_buscar', tipo: 'lectura', onde: 'backend' },
      ]),
      IATurnoAsistente: vi.fn(async () => turnos.shift()),
      IAChamarFerramenta: vi.fn(async () => ({ contido: 'resultados da web', erro: false })),
    }
    return { api, idioma: () => 'gl' }
  }

  it('sen asistente configurado a pantalla non leva o botón; co curso bloqueado tampouco', () => {
    const { container } = montar()
    abrirNova(container)
    expect(boton()).toBeNull()
    document.body.replaceChildren()
    const { container: c2 } = montar(gardarTarefa({ ...plan0(), isLocked: true }, '2025-09', { id: 'a', title: 'X', description: '', color: 'blue' }), { asistenteTarefa: asistente([]) })
    c2.querySelector('.crono-tarefa__corpo').click()
    expect(boton()).toBeNull()
  })

  it('co asistente: o botón abre un panel incrustado coas fichas da tarefa', async () => {
    const { container } = montar(plan0(), { asistenteTarefa: asistente([]) })
    abrirNova(container)
    expect(boton()).not.toBeNull()
    expect(panel().classList.contains('hidden')).toBe(true)
    boton().click()
    expect(panel().classList.contains('hidden')).toBe(false)
    await vi.waitFor(() => expect(panel().querySelectorAll('.asistente__chip').length).toBeGreaterThanOrEqual(7))
    const nomes = [...panel().querySelectorAll('.asistente__chip')].map((c) => c.textContent)
    for (const quero of ['Corrixir ortografía', 'Mellorar a redacción', 'Mellorar o formato', 'Referencias legais']) {
      expect(nomes.some((n) => n.includes(quero)), quero).toBe(true)
    }
    expect(panel().querySelector('.asistente__caixa')).not.toBeNull()
  })

  it('unha ficha edita o BORRADOR (non o curso) e o cambio pódese desfacer', async () => {
    const chamada = (id, nome, args) => ({ id, nome, args: JSON.stringify(args) })
    const turnos = [
      { texto: '', chamadas: [chamada('c1', 'editor_ler', {})], bruto: 'B1', provedor: 'openai' },
      { texto: '', chamadas: [chamada('c2', 'editor_cambiar', { titulo: 'Avaliación inicial', descricion: 'Texto **corrixido**' })], bruto: 'B2', provedor: 'openai' },
      { texto: 'Feito.', chamadas: [], bruto: 'B3', provedor: 'openai' },
    ]
    const asi = asistente(turnos)
    const p0 = gardarTarefa(plan0(), '2025-09', { id: 'a', title: 'Avaliacion', description: 'Texto mal escrito', color: 'blue' })
    const { container, cambios } = montar(p0, { asistenteTarefa: asi })
    container.querySelector('.crono-tarefa__corpo').click()
    boton().click()
    await vi.waitFor(() => expect(panel().querySelectorAll('.asistente__chip').length).toBeGreaterThan(0))
    ;[...panel().querySelectorAll('.asistente__chip')].find((c) => c.textContent.includes('Corrixir ortografía')).click()
    await vi.waitFor(() => expect(modal().querySelector('input[type=text]').value).toBe('Avaliación inicial'))
    await vi.waitFor(() => expect(panel().textContent).toContain('Feito.'))

    // O borrador cambiou...
    expect(modal().querySelector('textarea.crono-md').value).toBe('Texto **corrixido**')
    expect(modal().querySelector('.crono-md-vista strong').textContent).toBe('corrixido')
    expect(modal().querySelector('.tarefa-cab__estado').textContent).toMatch(/sen aplicar/)
    // ...pero o curso non ata Aplicar.
    expect(titulos(container, '2025-09')).toEqual(['Avaliacion'])
    expect(cambios).toHaveLength(0) // o curso non recibiu ningún cambio

    // A IA recibiu o ámbito, o catálogo da tarefa e o estado do borrador.
    const primeira = asi.api.IATurnoAsistente.mock.calls[0][0]
    expect(primeira.ambito).toBe('tarefa')
    expect(primeira.idioma).toBe('gl')
    expect(primeira.estado).toContain('Avaliacion')
    expect(asi.api.IAFerramentasDoAmbito).toHaveBeenCalledWith('tarefa')

    // Desfacer cambio da IA.
    const desfacer = modal().querySelector('.tarefa-desfacer-ia')
    expect(desfacer.hidden).toBe(false)
    desfacer.click()
    expect(modal().querySelector('input[type=text]').value).toBe('Avaliacion')
    expect(modal().querySelector('textarea.crono-md').value).toBe('Texto mal escrito')
    expect(desfacer.hidden).toBe(true)
  })

  it('despois de Aplicar, o cambio da IA entra no curso', async () => {
    const turnos = [
      { texto: '', chamadas: [{ id: 'c1', nome: 'editor_cambiar', args: JSON.stringify({ titulo: 'Novo título', cor: 'rose' }) }], bruto: 'B', provedor: 'openai' },
      { texto: 'ok', chamadas: [], bruto: 'B', provedor: 'openai' },
    ]
    const { container } = montar(plan0(), { asistenteTarefa: asistente(turnos) })
    abrirNova(container)
    boton().click()
    await vi.waitFor(() => expect(panel().querySelectorAll('.asistente__chip').length).toBeGreaterThan(0))
    panel().querySelector('.asistente__caixa').value = 'cambia o título'
    panel().querySelector('.asistente__enviar').click()
    await vi.waitFor(() => expect(modal().querySelector('input[type=text]').value).toBe('Novo título'))
    await vi.waitFor(() => expect(panel().textContent).toContain('ok'))
    expect(modal().querySelector('.crono-cor--activa').title).toBe('Rosa')
    modal().querySelector('.tarefa-aplicar').click()
    expect(titulos(container, '2025-09')).toEqual(['Novo título'])
  })

  it('Esc escribindo no asistente non pecha a pantalla da tarefa; pecha só o panel', async () => {
    const { container } = montar(plan0(), { asistenteTarefa: asistente([]) })
    abrirNova(container)
    boton().click()
    await vi.waitFor(() => expect(panel().querySelector('.asistente__caixa')).not.toBeNull())
    panel().querySelector('.asistente__caixa').dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true, cancelable: true }))
    expect(modal()).not.toBeNull()
    expect(panel().classList.contains('hidden')).toBe(true)
  })

  it('ao pechar a pantalla desaparece o panel', async () => {
    const { container } = montar(plan0(), { asistenteTarefa: asistente([]) })
    abrirNova(container)
    boton().click()
    modal().querySelector('.tarefa-x').click()
    await new Promise((r) => setTimeout(r, 0))
    expect(document.querySelector('.asistente')).toBeNull()
  })
})
