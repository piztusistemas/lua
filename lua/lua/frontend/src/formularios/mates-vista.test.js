import { describe, it, expect } from 'vitest'
import { htmlMates } from './mates-vista.js'

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
