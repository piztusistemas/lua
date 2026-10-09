  function pintar() {
    const e = estimarProgreso(agora() - inicio, esperado)
    enche.style.width = e.pct + '%'
    contedor.setAttribute('aria-valuenow', String(e.pct))
  }

// Progreso da compilación (botón Xerar): só unha barra, sen texto nin porcentaxe.
//
// LaTeX non di canto leva, así que o progreso é unha ESTIMACIÓN: parte do tempo
// que tardou a última vez (gárdase por ferramenta, con media móbil) e a barra
// sobe ata ~90% cando chega a ese tempo; se aínda non rematou, achégase a 99%
// sen chegar, para que non minta. Ao rematar a barra vai ao 100%.

// estimarProgreso: a porcentaxe (0-99) e os segundos que faltan, segundo o
// tempo transcorrido e o esperado (ms).
export function estimarProgreso(transcorridoMs, esperadoMs) {
  const T = Math.max(esperadoMs || 0, 1500)
  const x = Math.max(0, transcorridoMs) / T
  const pct = x <= 1 ? 90 * (1 - (1 - x) ** 2) : 90 + 9 * (1 - Math.exp(-(x - 1) * 2))
  return {
    pct: Math.min(99, Math.floor(pct)),
    faltanS: Math.max(1, Math.ceil((T - transcorridoMs) / 1000)),
    pasado: transcorridoMs > T,
  }
}

// novaEstimacion: media móbil do tempo esperado, con límites razoables.
export function novaEstimacion(anteriorMs, medidoMs) {
  if (!Number.isFinite(medidoMs) || medidoMs < 300 || medidoMs > 180000) return anteriorMs
  if (!anteriorMs) return Math.round(medidoMs)
  return Math.round(anteriorMs * 0.5 + medidoMs * 0.5)
}

const ESPERADO_INICIAL_MS = 8000

// crearProgreso monta a lóxica sobre os nodos xa existentes:
//   contedor  o div role=progressbar (hidden ata que se compila)
//   enche     o span que se ensancha
//   t         tradución (só para a etiqueta accesible da barra)
//   almacen   {get(clave), set(clave, valor)} (localStorage, inxectable)
//   agora     () => ms (inxectable para as probas)
export function crearProgreso({ contedor, enche, t, almacen, agora = () => Date.now(), setInterval: si = setInterval, clearInterval: ci = clearInterval }) {
  let inicio = 0
  let esperado = ESPERADO_INICIAL_MS
  let temporizador = null
  let clave = ''

  function pintar() {
    const e = estimarProgreso(agora() - inicio, esperado)
    enche.style.width = e.pct + '%'
    contedor.setAttribute('aria-valuenow', String(e.pct))
  }

  return {
    // comezar(id): id identifica o tipo de documento (a estimación é de cada un).
    comezar(id = 'xeral') {
      clave = 'lua.xerar.ms.' + id
      esperado = Number(almacen.get(clave)) || ESPERADO_INICIAL_MS
      inicio = agora()
      contedor.setAttribute('aria-label', t('informes.status.compilando', 'Compilando o documento…'))
      contedor.hidden = false
      document.body.classList.add('xerando')
      pintar()
      if (temporizador) ci(temporizador)
      temporizador = si(pintar, 200)
    },
    // rematar(ok): quita a barra; se saíu ben, aprende canto tardou.
    rematar(ok = true) {
      if (temporizador) ci(temporizador)
      temporizador = null
      if (ok && clave) almacen.set(clave, String(novaEstimacion(Number(almacen.get(clave)) || 0, agora() - inicio)))
      enche.style.width = '100%'
      contedor.setAttribute('aria-valuenow', '100')
      contedor.hidden = true
      document.body.classList.remove('xerando')
    },
    activo: () => temporizador !== null,
  }
}
