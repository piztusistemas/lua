import { describe, it, expect } from 'vitest'
import { readFileSync } from 'node:fs'
import { PALETA, chave } from './editor-formulas.js'

// As claves da paleta constrúense ('formulas.p.' + chave(título)) e o test
// de Go (i18n_test.go) non as ve: compróbanse aquí nos catro idiomas.
const idioma = (c) => JSON.parse(readFileSync(new URL(`../../../idiomas/${c}.json`, import.meta.url), 'utf8'))

describe('paleta do editor de fórmulas', () => {
  const claves = PALETA.flatMap(([id, , items]) => [
    'formulas.tab.' + id,
    ...items.filter(([, , , traducir = true]) => traducir).map(([titulo]) => 'formulas.p.' + chave(titulo)),
  ])

  it('as claves non se repiten', () => {
    expect(new Set(claves).size).toBe(claves.length)
  })

  for (const c of ['gl', 'es', 'en', 'pt']) {
    it(`${c}.json ten todas as claves`, () => {
      const d = idioma(c)
      expect(claves.filter((k) => !d[k])).toEqual([])
    })
  }

  it('chave quita tiles e signos', () => {
    expect(chave('Raíz n-ésima')).toBe('raiz-n-esima')
    expect(chave('Matriz 2×2')).toBe('matriz-2-2')
  })
})
