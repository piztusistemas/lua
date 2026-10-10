// @vitest-environment jsdom
import { describe, it, expect, afterEach } from 'vitest'
import { t, fixarIdioma, aplicarTraducions } from './i18n.js'
import { textoEstado } from './util.js'

afterEach(() => fixarIdioma('gl'))

describe('i18n de Buxo (spec buxo-idiomas, aceptación 1)', () => {
  it('aplica o dicionario es ao HTML marcado', () => {
    document.body.innerHTML = `
      <button id="b" data-i18n="ui.construir" data-i18n-title="ui.estado.listo" title="A IA le os documentos convertidos e constrúe a wiki">Construír a bóveda</button>
      <textarea id="x" data-i18n-placeholder="ui.instrucion.ph" placeholder="Ex.: só os exames de 2023 en adiante; ignora as solucións"></textarea>`
    fixarIdioma('es')
    aplicarTraducions(document)
    expect(document.getElementById('b').textContent).toBe('Construir la bóveda')
    expect(document.getElementById('b').title).toMatch(/La IA lee/)
    expect(document.getElementById('x').placeholder).toMatch(/^Ej\.:/)
  })
  it('os textos que xera o JS tamén cambian', () => {
    fixarIdioma('en')
    expect(textoEstado({ iaConfigurada: false, fontes: 0 })).toMatch(/AI key/)
    expect(t('ui.ia_modelo', 'IA: {modelo}', { modelo: 'gpt' })).toBe('AI: gpt')
  })
  it('idioma descoñecido ou clave sen tradución → galego', () => {
    fixarIdioma('xx')
    expect(t('ui.construir', 'Construír a bóveda')).toBe('Construír a bóveda')
    fixarIdioma('es')
    expect(t('ui.non.existe', 'texto galego')).toBe('texto galego')
  })
})
