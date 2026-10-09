// @vitest-environment jsdom
import { describe, it, expect, vi } from 'vitest'
import { estimarProgreso, novaEstimacion, crearProgreso } from './progreso.js'

describe('estimarProgreso', () => {
  it('sobe sen parar, non chega ao 100 e achégase ao 90 no tempo esperado', () => {
    let anterior = -1
    for (const ms of [0, 500, 1000, 2000, 4000, 8000, 9000, 12000, 30000, 300000]) {
      const { pct } = estimarProgreso(ms, 8000)
      expect(pct).toBeGreaterThanOrEqual(anterior)
      expect(pct).toBeLessThanOrEqual(99)
      anterior = pct
    }
    expect(estimarProgreso(0, 8000).pct).toBe(0)
    expect(estimarProgreso(8000, 8000).pct).toBe(90)
    expect(estimarProgreso(8000, 8000).pasado).toBe(false)
  })
  it('os segundos que faltan baixan e nunca chegan a 0; pasado o tempo esperado avisa', () => {
    expect(estimarProgreso(0, 8000).faltanS).toBe(8)
    expect(estimarProgreso(6500, 8000).faltanS).toBe(2)
    const tarde = estimarProgreso(20000, 8000)
    expect(tarde.faltanS).toBe(1)
    expect(tarde.pasado).toBe(true)
  })
  it('un tempo esperado absurdo (0 ou moi pequeno) ten un mínimo', () => {
    expect(estimarProgreso(500, 0).pct).toBeLessThan(90)
  })
})

describe('novaEstimacion', () => {
  it('fai media co anterior e ignora medidas absurdas', () => {
    expect(novaEstimacion(0, 6000)).toBe(6000)
    expect(novaEstimacion(6000, 10000)).toBe(8000)
    expect(novaEstimacion(6000, 50)).toBe(6000)
    expect(novaEstimacion(6000, 999999)).toBe(6000)
    expect(novaEstimacion(6000, NaN)).toBe(6000)
  })
})

describe('crearProgreso', () => {
  function montar(inicial = {}) {
    document.body.innerHTML = '<div id="c" hidden><span id="e"></span></div>'
    const datos = { ...inicial }
    let agora = 1000
    const t = (c, f) => f
    let tic = null
    const p = crearProgreso({
      contedor: document.getElementById('c'), enche: document.getElementById('e'), t,
      almacen: { get: (k) => datos[k], set: (k, v) => { datos[k] = v } },
      agora: () => agora, setInterval: (f) => { tic = f; return 1 }, clearInterval: () => { tic = null },
    })
    return { p, datos, avanzar: (ms) => { agora += ms; tic && tic() }, c: document.getElementById('c'), e: document.getElementById('e') }
  }

  it('amosa só a barra (sen texto) e ao rematar quítaa e aprende canto tardou', () => {
    const m = montar({ 'lua.xerar.ms.actas': '10000' })
    m.p.comezar('actas')
    expect(m.c.hidden).toBe(false)
    expect(document.body.classList.contains('xerando')).toBe(true)
    expect(m.c.textContent).toBe('') // nin «compilando», nin porcentaxe, nin tempo
    expect(m.c.getAttribute('aria-label')).toBe('Compilando o documento…') // só accesible, non visible
    m.avanzar(5000)
    expect(m.e.style.width).toBe('67%')
    expect(m.c.getAttribute('aria-valuenow')).toBe('67')
    expect(m.c.textContent).toBe('')
    m.avanzar(9000)
    m.p.rematar(true)
    expect(m.c.hidden).toBe(true)
    expect(document.body.classList.contains('xerando')).toBe(false)
    expect(m.datos['lua.xerar.ms.actas']).toBe(String(Math.round(10000 * 0.5 + 14000 * 0.5)))
    expect(m.p.activo()).toBe(false)
  })

  it('un fallo non cambia a estimación; sen historial usa un tempo por defecto', () => {
    const m = montar()
    m.p.comezar('cronograma')
    m.avanzar(3000)
    expect(m.e.style.width).not.toBe('0%')
    m.p.rematar(false)
    expect(m.datos['lua.xerar.ms.cronograma']).toBeUndefined()
  })
})
