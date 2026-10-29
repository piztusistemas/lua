import { describe, it, expect, afterEach } from 'vitest'
import { htmlMates, configurarFormulas, normalizarEscalaFormulas } from './mates-vista.js'

describe('htmlMates', () => {
  it('compón as fórmulas e o Markdown', () => {
    const h = htmlMates('**1.** A área é $\\pi r^2$.\n\n$$x = \\dfrac{1}{2}$$\n\n- uno\n- dous')
    expect(h).toContain('<strong>1.</strong>')
    expect(h).toContain('class="katex"')
    expect(h).toContain('katex-display')
    expect(h).toContain('<ul><li>uno</li><li>dous</li></ul>')
    expect(htmlMates('Resolve $ {x^2+x-2} = 0 $.')).toContain('katex')
  })
  it('escapa o HTML e respecta os prezos', () => {
    const h = htmlMates('<b>x</b> custa 5$ ou 6$ e \\$7')
    expect(h).toContain('&lt;b&gt;')
    expect(h).not.toContain('katex')
    expect(h).toContain('$7')
  })
  it('pinta os circuítos que compilou Go', () => {
    const md = 'Circuíto:\n\n```circuitikz\n\\draw (0,0) to[R] (2,0);\n```\n\nFin'
    expect(htmlMates(md)).toContain('Debuxando')
    const h = htmlMates(md, { 'circuitikz\n\\draw (0,0) to[R] (2,0);': { imaxe: 'data:image/svg+xml;base64,AAA', anchoPx: 120.4 } })
    expect(h).toContain('<img src="data:image/svg+xml;base64,AAA" style="width:120px"')
    expect(h).toContain('Fin')
    expect(htmlMates(md, { 'circuitikz\n\\draw (0,0) to[R] (2,0);': { erro: 'Undefined control sequence' } })).toContain('erro de TikZ')
  })
})

describe('opcións das fórmulas (Opcións → Fórmulas nos documentos)', () => {
  afterEach(() => configurarFormulas({}))

  it('por defecto, as fórmulas coma sempre', () => {
    const h = htmlMates('Divide $\\frac{a}{b}$')
    expect(h).not.toContain('font-size')
    expect(h).toContain('mfrac')
  })

  it('a tamaño completo, as fórmulas en liña van en displaystyle', () => {
    const compacta = htmlMates('$\\frac{a}{b}$')
    configurarFormulas({ formulasEstilo: 'grandes' })
    const grande = htmlMates('$\\frac{a}{b}$')
    expect(grande).not.toBe(compacta)
    expect(grande).not.toContain('font-size')
  })

  it('a escala agranda só as fórmulas', () => {
    configurarFormulas({ formulasEscala: 130 })
    const h = htmlMates('Texto $x$')
    expect(h).toContain('font-size:130%')
    expect(h.startsWith('<p>Texto ')).toBe(true)
  })

  it('as opcións pódense pasar para unha mostra sen cambiar as globais', () => {
    expect(htmlMates('$x$', null, { formulasEscala: 150 })).toContain('font-size:150%')
    expect(htmlMates('$x$')).not.toContain('font-size')
  })

  it('normaliza a escala', () => {
    expect([0, 95, 120, 124, 126, 300, 'x'].map(normalizarEscalaFormulas)).toEqual([100, 100, 120, 120, 130, 150, 100])
  })
})
