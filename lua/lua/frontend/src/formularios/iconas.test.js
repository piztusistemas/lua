import { describe, it, expect } from 'vitest'
import { CLAVES_COR, CLAVES_ICONA, coresFormularios, aparenciaLibre, valorCor } from './iconas.js'

describe('cores dos formularios', () => {
  it('os de antes (sen cor) reciben cores distintas, por orde de id', () => {
    const cores = coresFormularios([{ id: 'c' }, { id: 'a' }, { id: 'b', cor: 'rosa' }])
    expect(cores).toEqual({ a: CLAVES_COR[0], c: CLAVES_COR[1], b: 'rosa' })
  })

  it('gardar a cor que xa amosaba un dos de antes non lles cambia a cor aos outros', () => {
    const antes = [{ id: 'a' }, { id: 'b' }, { id: 'c' }]
    const cores = coresFormularios(antes)
    const despois = coresFormularios([{ id: 'a' }, { id: 'b', cor: cores.b }, { id: 'c' }])
    expect(despois).toEqual(cores)
  })

  it('unha cor descoñecida trátase coma se non tivese', () => {
    expect(coresFormularios([{ id: 'a', cor: 'vermello' }]).a).toBe(CLAVES_COR[0])
    expect(valorCor('vermello')).toBe(valorCor('azul'))
  })

  it('crear un formulario novo non lles cambia a cor aos de antes', () => {
    const antes = [{ id: 'x' }, { id: 'y' }]
    const novo = { id: 'n', ...aparenciaLibre(antes) }
    const despois = coresFormularios([...antes, novo])
    expect(despois.x).toBe(coresFormularios(antes).x)
    expect(despois.y).toBe(coresFormularios(antes).y)
  })

  it('un formulario novo leva a forma e a cor menos usadas', () => {
    const esquemas = [
      { id: 'a', icona: 'lapis' }, { id: 'b', icona: 'lapis' }, { id: 'c', icona: CLAVES_ICONA[0] },
    ]
    const ap = aparenciaLibre(esquemas)
    expect(['lapis', CLAVES_ICONA[0]]).not.toContain(ap.icona)
    expect(Object.values(coresFormularios(esquemas))).not.toContain(ap.cor)
  })

  it('sen formularios, a primeira de cada', () => {
    expect(aparenciaLibre([])).toEqual({ icona: CLAVES_ICONA[0], cor: CLAVES_COR[0] })
  })
})
