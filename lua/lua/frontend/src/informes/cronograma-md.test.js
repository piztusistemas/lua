import { describe, it, expect } from 'vitest'
import {
  envolver, prefixoBloque, alinear, tamanoLetra, ligazon, inserirImaxe,
  parseInline, parseMarkdown, markdownAHtml,
} from './cronograma-md.js'

const est = (texto, ini = 0, fin = ini) => ({ texto, ini, fin })

describe('barra de formato: envolver', () => {
  it('envolve a selección', () => {
    const r = envolver(est('un texto aquí', 3, 8), '**', '**')
    expect(r.texto).toBe('un **texto** aquí')
    expect(r.texto.slice(r.ini, r.fin)).toBe('texto')
  })
  it('sen selección insire un marcador seleccionado', () => {
    const r = envolver(est('ab', 1), '*', '*', 'texto')
    expect(r.texto).toBe('a*texto*b')
    expect(r.texto.slice(r.ini, r.fin)).toBe('texto')
  })
  it('tamanoLetra usa a marca [fs=..]', () => {
    expect(tamanoLetra(est('x', 0, 1), 'gr').texto).toBe('[fs=gr]x[/fs]')
  })
})

describe('barra de formato: prefixoBloque', () => {
  it('pon o prefixo en cada liña seleccionada e despois quítao (alterna)', () => {
    const a = prefixoBloque(est('un\ndous\ntres', 0, 8), '- ')
    expect(a.texto).toBe('- un\n- dous\ntres')
    const b = prefixoBloque(est(a.texto, a.ini, a.fin), '- ')
    expect(b.texto).toBe('un\ndous\ntres')
  })
  it('sen selección afecta só á liña do cursor', () => {
    expect(prefixoBloque(est('a\nb\nc', 2), '## ').texto).toBe('a\n## b\nc')
  })
})

describe('barra de formato: alinear', () => {
  it('envolve a liña nun bloque ::: e o parser recoñéceo', () => {
    const r = alinear(est('antes\n\ncentrado\n\ndespois', 8), 'centro')
    expect(r.texto).toContain(':::centro\ncentrado\n:::')
    const bl = parseMarkdown(r.texto)
    expect(bl.find((b) => b.tramos?.[0]?.texto === 'centrado').alin).toBe('center')
    expect(bl.find((b) => b.tramos?.[0]?.texto === 'despois').alin).toBe('')
  })
  it('se xa está nun bloque, cambia o nome en vez de aniñar', () => {
    const a = alinear(est('hola', 0, 4), 'centro')
    const b = alinear(est(a.texto, a.ini, a.fin), 'dereita')
    expect(b.texto).toBe(':::dereita\nhola\n:::')
  })
  it('ignora nomes descoñecidos', () => {
    const e = est('x')
    expect(alinear(e, 'raro')).toBe(e)
  })
})

describe('barra de formato: ligazón e imaxe', () => {
  it('ligazon usa a selección coma etiqueta', () => {
    expect(ligazon(est('ver web ok', 4, 7), 'https://x.gal').texto).toBe('ver [web](https://x.gal) ok')
  })
  it('inserirImaxe pon a imaxe nunha liña propia', () => {
    const r = inserirImaxe(est('antes despois', 6), 'i1', 'foto')
    expect(r.texto).toBe('antes \n![foto](img:i1)\ndespois')
    expect(parseMarkdown(r.texto).some((b) => b.tipo === 'img' && b.imaxe.ref === 'i1')).toBe(true)
  })
})

// Mesmos casos ca TestParseInlineEnxeCrono (informe_cronograma_test.go): se
// un parser cambia hai que cambiar o outro.
describe('parseInline (paridade co parser Go)', () => {
  const marcado = (s) => parseInline(s).map((x) => {
    let p = x.texto
    if (x.negrita) p = `[b]${p}[/b]`
    if (x.cursiva) p = `[i]${p}[/i]`
    return p
  }).join('')
  const casos = [
    ['un **negro** e *cursivo*', 'un [b]negro[/b] e [i]cursivo[/i]'],
    ['un __negro__ e _cursivo_', 'un [b]negro[/b] e [i]cursivo[/i]'],
    ['**a *b* c**', '[b]a [/b][i][b]b[/b][/i][b] c[/b]'],
    ['caracteres _ ^ ~ e mais', 'caracteres _ ^ ~ e mais'],
    ['2 * 3 = 6', '2 * 3 = 6'],
    ['un **negro sen pechar', 'un **negro sen pechar'],
    ['ver nome_ficheiro_final.txt', 'ver nome_ficheiro_final.txt'],
    ['ver _cursiva_ e nome_x', 'ver [i]cursiva[/i] e nome_x'],
    ['in*tra*palabra', 'in[i]tra[/i]palabra'],
  ]
  for (const [entrada, esperado] of casos) {
    it(JSON.stringify(entrada), () => expect(marcado(entrada)).toBe(esperado))
  }
  it('tamaño de letra e clave descoñecida', () => {
    const tr = parseInline('a [fs=gr]grande **negro**[/fs] b')
    expect(tr.find((x) => x.texto === 'negro')).toMatchObject({ escala: 1.3, negrita: true })
    expect(parseInline('[fs=zzz]x[/fs]')[0].texto).toContain('fs=zzz')
  })
})

describe('parseMarkdown', () => {
  const md = '# Título\n\nUn parágrafo\nen dúas liñas.\n\n- un\n- dous\n1. tres\n\n> cita\n\n:::centro\ncentrado\n:::\n![logo](img:a1 =60%)\n![f](img:b =120x80)'
  it('recoñece os tipos de bloque na mesma orde ca o parser Go', () => {
    expect(parseMarkdown(md).map((b) => b.tipo)).toEqual(['h', 'p', 'li', 'li', 'li', 'q', 'p', 'img', 'img'])
  })
  it('une as liñas dun parágrafo con espazo', () => {
    expect(parseMarkdown(md)[1].tramos[0].texto).toBe('Un parágrafo en dúas liñas.')
  })
  it('o peche ::: devolve o aliñamento por defecto', () => {
    const bl = parseMarkdown(md)
    expect(bl[6].alin).toBe('center')
    expect(bl[7].alin).toBe('')
  })
  it('le tamaños de imaxe en % e en mm', () => {
    const bl = parseMarkdown(md)
    expect(bl[7].imaxe).toMatchObject({ ref: 'a1', anchoPct: 60 })
    expect(bl[8].imaxe).toMatchObject({ anchoMM: 120, altoMM: 80 })
  })
})

describe('markdownAHtml', () => {
  it('xera listas agrupadas, títulos e énfase', () => {
    const h = markdownAHtml('# T\n\n- a\n- b\n\n1. c\n\n**n** e _c_')
    expect(h).toContain('<h4>T</h4>')
    expect(h).toContain('<ul><li>a</li><li>b</li></ul>')
    expect(h).toContain('<ol><li>c</li></ol>')
    expect(h).toContain('<strong>n</strong>')
    expect(h).toContain('<em>c</em>')
  })
  it('escapa o HTML do texto', () => {
    const h = markdownAHtml('<script>alert(1)</script> & "x"')
    expect(h).not.toContain('<script>')
    expect(h).toContain('&lt;script&gt;')
    expect(h).toContain('&amp;')
  })
  it('só deixa ligazóns http(s) e mailto', () => {
    expect(markdownAHtml('[a](https://x.gal)')).toContain('href="https://x.gal"')
    expect(markdownAHtml('[a](javascript:alert(1))')).not.toContain('href')
  })
  it('amosa a imaxe do mapa e avisa se falta', () => {
    const uri = 'data:image/png;base64,AAAA'
    expect(markdownAHtml('![f](img:a =40%)', { a: uri })).toContain(`src="${uri}"`)
    expect(markdownAHtml('![f](img:a =40%)', { a: uri })).toContain('width:40%')
    expect(markdownAHtml('![f](img:nope)', {})).toContain('imaxe non atopada')
  })
  it('non deixa pasar un src que non sexa data:image', () => {
    expect(markdownAHtml('![f](img:a)', { a: 'javascript:alert(1)' })).toContain('imaxe non atopada')
  })
  it('aplica aliñamento e tamaño de letra', () => {
    expect(markdownAHtml(':::dereita\nx\n:::')).toContain('text-align:right')
    expect(markdownAHtml('[fs=gr]x[/fs]')).toContain('font-size:1.3em')
  })
})
