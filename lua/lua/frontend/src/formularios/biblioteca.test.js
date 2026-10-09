// @vitest-environment jsdom
import { describe, it, expect, vi, afterEach } from 'vitest'
import { abrirBibliotecaFormularios } from './biblioteca.js'
import { bibliotecaBaleira as bibliotecaPorDefecto, INTEGRADAS, fixar, agochar, moverACarpeta } from './biblioteca-modelo.js'

// Fixture: o cronograma e as actas activos na zona fixa (o por defecto é só na biblioteca).
const bibliotecaBaleira = () => ({ ...bibliotecaPorDefecto(), fixos: [...INTEGRADAS], agochados: [] })

afterEach(() => { document.body.replaceChildren() })
const t = (k, f, v) => { let s = f ?? k; if (v) for (const x in v) s = s.replaceAll(`{${x}}`, v[x]); return s }
const tick = () => new Promise((r) => setTimeout(r, 0))

const FORMS = [
  { id: 'acta', nome: 'Acta de titoría', icona: 'libro', modo: 'deseno', descricion: 'Reunión mensual', campos: [{ tipo: 'texto' }, { tipo: 'seccion' }, { tipo: 'data' }] },
  { id: 'ficha', nome: 'Ficha do alumnado', icona: 'persoas', modo: 'produccion', descricion: '', campos: [{ tipo: 'texto' }] },
  { id: 'exame', nome: 'Exame', icona: 'lapis', descricion: 'Notas do exame', campos: [] },
]

function abrir(inicial = { ...bibliotecaBaleira(), carpetas: [{ id: 'tit', nome: 'Titoría' }, { id: 'exa', nome: 'Exames' }], carpeta: { acta: 'tit', exame: 'exa' } }, extra = {}) {
  const estado = { bib: inicial, formularios: FORMS.map((f) => ({ ...f })) }
  const accions = {
    formularios: () => estado.formularios, biblioteca: () => estado.bib,
    integradas: [{ id: 'cronograma', nome: 'Cronograma' }, { id: 'actas', nome: 'Actas' }],
    gardar: vi.fn(async (nova) => { estado.bib = nova }),
    crearCarpeta: vi.fn(async (nome) => { estado.bib = { ...estado.bib, carpetas: [...estado.bib.carpetas, { id: nome.toLowerCase(), nome }] } }),
    gardarDescricion: vi.fn(async (id, texto) => { estado.formularios.find((f) => f.id === id).descricion = texto }),
    abrir: vi.fn(async () => {}),
    eliminar: vi.fn(async (tid) => { estado.formularios = estado.formularios.filter((f) => 'form:' + f.id !== tid); return true }),
    confirmar: vi.fn(async () => true), preguntar: vi.fn(async () => 'Novo'),
    ...extra,
  }
  const fin = abrirBibliotecaFormularios({ t, accions })
  return { estado, accions, fin }
}
const q = (s) => document.querySelector(s)
const qa = (s) => [...document.querySelectorAll(s)]
const nomes = () => qa('.bib-form .bib-form__tit strong').map((e) => e.textContent)
const carpeta = (nome) => qa('.bib-carp').find((b) => b.textContent.startsWith(nome))

describe('biblioteca de formularios', () => {
  it('lista todos os formularios con nome, modo, descrición e cantidade de campos (sen contar seccións)', () => {
    abrir()
    expect(nomes()).toEqual(['Acta de titoría', 'Ficha do alumnado', 'Exame'])
    const acta = qa('.bib-form')[0]
    expect(acta.querySelector('.bib-desc').value).toBe('Reunión mensual')
    expect(acta.querySelector('.bib-chip--deseno').textContent).toBe('Deseño')
    expect(acta.textContent).toContain('2 campos')
    expect(qa('.bib-form')[1].querySelector('.bib-chip--produccion').textContent).toBe('Produción')
    expect(qa('.bib-form')[1].querySelector('.bib-form__icona--produccion')).not.toBeNull()
  })

  it('os cartafoles filtran e contan; «Sen cartafol» ensina os que non están en ningún', () => {
    abrir()
    expect(carpeta('Todos').textContent).toContain('3')
    expect(carpeta('Sen cartafol').textContent).toContain('1')
    carpeta('Titoría').click()
    expect(nomes()).toEqual(['Acta de titoría'])
    carpeta('Sen cartafol').click()
    expect(nomes()).toEqual(['Ficha do alumnado'])
    carpeta('Todos').click()
    expect(nomes().length).toBe(3)
  })

  it('busca por nome ou descrición (sen acentos)', () => {
    abrir()
    const b = q('.bib-busca')
    b.value = 'notas'; b.dispatchEvent(new Event('input'))
    expect(nomes()).toEqual(['Exame'])
    b.value = 'TITORIA'; b.dispatchEvent(new Event('input'))
    expect(nomes()).toEqual(['Acta de titoría'])
    b.value = 'zzz'; b.dispatchEvent(new Event('input'))
    expect(q('.bib-baleira').textContent).toMatch(/Ningún formulario coincide/)
  })

  it('a descrición gárdase ao rematar de editala', async () => {
    const { accions } = abrir()
    const d = qa('.bib-desc')[1]
    d.value = 'Datos reais do alumnado'
    d.dispatchEvent(new Event('change'))
    await tick()
    expect(accions.gardarDescricion).toHaveBeenCalledWith('ficha', 'Datos reais do alumnado')
  })

  it('mover un formulario a un cartafol co selector', async () => {
    const { accions, estado } = abrir()
    const sel = qa('.bib-sel')[1] // Ficha, sen cartafol
    expect(sel.value).toBe('')
    sel.value = 'tit'
    sel.dispatchEvent(new Event('change'))
    await tick()
    expect(accions.gardar.mock.calls[0][0].carpeta.ficha).toBe('tit')
    expect(estado.bib.carpeta.ficha).toBe('tit')
  })

  it('crear un cartafol (pregunta o nome) e rexeita un repetido sen chamar a Go', async () => {
    const { accions } = abrir()
    q('.bib-nova').click()
    await tick()
    expect(accions.crearCarpeta).toHaveBeenCalledWith('Novo')
    expect(carpeta('Novo')).toBeTruthy()
    accions.preguntar.mockResolvedValueOnce(' titoría ')
    q('.bib-nova').click()
    await tick()
    expect(accions.crearCarpeta).toHaveBeenCalledTimes(1)
    expect(q('.bib-erro').hidden).toBe(false)
    expect(q('.bib-erro').textContent).toMatch(/Xa hai un cartafol/)
  })

  it('renomear e eliminar un cartafol: o eliminar pide confirmación e NON borra formularios', async () => {
    const { accions, estado } = abrir()
    accions.preguntar.mockResolvedValueOnce('Titorías 1º')
    qa('.bib-carp-fila')[0].querySelector('[data-foco^="r:"]').click()
    await tick()
    expect(estado.bib.carpetas[0].nome).toBe('Titorías 1º')

    accions.confirmar.mockResolvedValueOnce(false)
    qa('.bib-carp-fila')[0].querySelector('[data-foco^="e:"]').click()
    await tick()
    expect(estado.bib.carpetas.length).toBe(2) // cancelou

    qa('.bib-carp-fila')[0].querySelector('[data-foco^="e:"]').click()
    await tick()
    expect(accions.confirmar.mock.calls[1][0]).toMatch(/NON se borran/)
    expect(estado.bib.carpetas.map((c) => c.id)).toEqual(['exa'])
    expect(estado.bib.carpeta.acta).toBeUndefined()
    expect(estado.formularios.length).toBe(3) // seguen todos
    expect(nomes().length).toBe(3)
  })

  it('Abrir vai á ferramenta e pecha a biblioteca', async () => {
    const { accions, fin } = abrir()
    qa('.bib-form')[0].querySelector('[data-foco^="a:"]').click()
    await fin
    expect(accions.abrir).toHaveBeenCalledWith('form:acta')
    expect(q('.bib-overlay')).toBeNull()
  })

  it('o menú «⋯» é o mesmo que o do botón dereito: fixar, ocultar e eliminar', async () => {
    const { accions } = abrir()
    qa('.bib-mais')[1].click()
    expect([...document.querySelectorAll('.menu-ctx__item')].map((i) => i.textContent)).toEqual(['Fixar na zona superior', 'Ocultar da columna', 'Eliminar formulario…'])
    document.querySelectorAll('.menu-ctx__item')[0].click()
    await tick()
    expect(accions.gardar.mock.calls[0][0].fixos).toContain('form:ficha')
    // Agora está fixo: chip «Fixo» e non se pode borrar.
    expect(qa('.bib-form')[1].textContent).toContain('Fixo')
    qa('.bib-mais')[1].click()
    const borrar = [...document.querySelectorAll('.menu-ctx__item')].find((i) => i.textContent.startsWith('Eliminar'))
    expect(borrar.getAttribute('aria-disabled')).toBe('true')
    borrar.click()
    expect(accions.eliminar).not.toHaveBeenCalled()
  })

  it('eliminar desde o menú borra o formulario e desaparece da lista', async () => {
    const { accions } = abrir()
    qa('.bib-mais')[2].click()
    ;[...document.querySelectorAll('.menu-ctx__item')].find((i) => i.textContent.startsWith('Eliminar')).click()
    await tick()
    expect(accions.eliminar).toHaveBeenCalledWith('form:exame')
    expect(nomes()).toEqual(['Acta de titoría', 'Ficha do alumnado'])
  })

  it('un formulario agochado sae marcado, e o cronograma/actas pódense ocultar e amosar desde o pé', async () => {
    const { accions } = abrir(agochar(bibliotecaBaleira(), 'form:exame'))
    expect(qa('.bib-form')[2].textContent).toContain('Agochado')
    const btnCrono = qa('.bib-integrada')[0].querySelector('button')
    expect(btnCrono.textContent).toBe('Ocultar da columna')
    btnCrono.click()
    await tick()
    expect(accions.gardar.mock.calls[0][0].agochados).toContain('cronograma')
    expect(qa('.bib-integrada')[0].querySelector('button').textContent).toBe('Amosar na columna')
  })

  it('se Go rexeita un cambio, amosa o erro no diálogo e non se rompe', async () => {
    const { accions } = abrir()
    accions.gardar.mockRejectedValueOnce(new Error('tes que deixar polo menos unha ferramenta á vista'))
    qa('.bib-integrada')[0].querySelector('button').click()
    await tick()
    expect(q('.bib-erro').hidden).toBe(false)
    expect(q('.bib-erro').textContent).toMatch(/polo menos unha ferramenta/)
    qa('.bib-integrada')[0].querySelector('button').click() // xa funciona outra vez
    await tick()
    expect(q('.bib-erro').hidden).toBe(true)
  })

  it('despois de cambiar algo o foco volve ao mesmo control (teclado)', async () => {
    abrir()
    const sel = qa('.bib-sel')[1]
    sel.focus()
    sel.value = 'tit'
    sel.dispatchEvent(new Event('change'))
    await tick()
    expect(document.activeElement.dataset.foco).toBe('s:ficha')
    expect(document.activeElement).toBe(qa('.bib-sel')[1]) // o novo (repintado), non o vello
  })

  it('sen formularios ensina como crear o primeiro; Esc pecha', async () => {
    const { fin, estado } = abrir(bibliotecaBaleira(), { formularios: () => [] })
    expect(q('.bib-baleira').textContent).toMatch(/Aínda non tes formularios/)
    expect(estado).toBeTruthy()
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    await fin
    expect(q('.bib-overlay')).toBeNull()
  })

  it('un cartafol baleiro di como encher; Esc non pecha se hai un menú aberto encima', () => {
    abrir({ ...bibliotecaBaleira(), carpetas: [{ id: 'baleiro', nome: 'Baleiro' }] })
    carpeta('Baleiro').click()
    expect(q('.bib-baleira').textContent).toMatch(/cartafol está baleiro/)
    qa('.bib-mais').length // sen tarxetas
    carpeta('Todos').click()
    qa('.bib-mais')[0].click()
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    expect(q('.menu-ctx')).toBeNull() // pechou o menú
    expect(q('.bib-overlay')).not.toBeNull() // pero non a biblioteca
  })
})

// Non usados directamente, pero comproban que o modelo e a vista coinciden.
describe('coherencia co modelo', () => {
  it('un formulario fixado e agochado leva os dous chips', () => {
    let b = fixar(bibliotecaBaleira(), 'form:acta')
    b = agochar(b, 'form:acta')
    b = moverACarpeta(b, 'acta', '')
    abrir(b)
    const txt = qa('.bib-form')[0].textContent
    expect(txt).toContain('Fixo')
    expect(txt).toContain('Agochado')
  })

  describe('intercambio con compañeiros', () => {
    it('sen accións de intercambio non hai botón nin elemento de menú', () => {
      abrir()
      expect(q('[data-foco="imp"]')).toBeNull()
      qa('.bib-mais')[1].click()
      expect([...document.querySelectorAll('.menu-ctx__item')].some((i) => i.textContent.startsWith('Exportar'))).toBe(false)
    })

    it('«Importar formulario…» chama a Go, repinta e amosa os avisos', async () => {
      const importar = vi.fn(async () => ({ avisos: ['Xa tiñas un formulario con ese identificador.'] }))
      const { estado } = abrir(undefined, { importar })
      q('[data-foco="imp"]').click()
      await tick()
      expect(importar).toHaveBeenCalledOnce()
      expect(q('.bib-aviso').hidden).toBe(false)
      expect(q('.bib-aviso').textContent).toContain('Xa tiñas un formulario')
      expect(q('.bib-erro').hidden).toBe(true) // un aviso non é un erro
      // Un formulario importado aparece na lista ao repintar.
      estado.formularios.push({ id: 'novo', nome: 'Novo importado', icona: 'lapis', campos: [] })
      importar.mockImplementationOnce(async () => ({ avisos: [] }))
      q('[data-foco="imp"]').click()
      await tick()
      expect(nomes()).toContain('Novo importado')
      expect(q('.bib-aviso').hidden).toBe(true) // o aviso anterior límpase
    })

    it('cancelar o diálogo (null) non amosa nada, e un erro de Go sae na capa', async () => {
      const importar = vi.fn(async () => null)
      abrir(undefined, { importar })
      q('[data-foco="imp"]').click()
      await tick()
      expect(q('.bib-erro').hidden).toBe(true)
      importar.mockImplementationOnce(async () => { throw new Error('o ficheiro non é un formulario de Lua') })
      q('[data-foco="imp"]').click()
      await tick()
      expect(q('.bib-erro').textContent).toContain('non é un formulario de Lua')
    })

    it('o menú de cada formulario ofrece «Exportar formulario…» co seu id', async () => {
      const exportar = vi.fn(async () => {})
      abrir(undefined, { exportar })
      qa('.bib-mais')[1].click()
      const item = [...document.querySelectorAll('.menu-ctx__item')].find((i) => i.textContent.startsWith('Exportar'))
      expect(item).toBeTruthy()
      item.click()
      await tick()
      expect(exportar).toHaveBeenCalledWith('form:ficha')
    })
  })
})
