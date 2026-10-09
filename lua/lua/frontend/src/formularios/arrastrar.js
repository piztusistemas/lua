// Arrastre por eventos de punteiro (rato, dedo, lapis). Non usa o DnD de HTML5
// porque nas webviews de escritorio (WebKitGTK, WebView2) é pouco fiable e
// mais cando a app tamén acepta ficheiros soltos.
//
//   iniciarArrastre(eventoPointerdown, {
//     texto,                // o que leva o «fantasma» pegado ao punteiro
//     aoMover(x, y),        // mentres se arrastra (para marcar onde caería)
//     aoSoltar(x, y),       // ao soltar despois de arrastrar de verdade
//     aoRematar(),          // sempre ao final (para limpar as marcas)
//   })
//
// Un simple clic (sen mover máis de LIMIAR px) NON é un arrastre: non chama a
// aoSoltar, así que o mesmo elemento pode ter tamén acción de clic.

const LIMIAR = 6

export function iniciarArrastre(ev, { texto = '', aoMover, aoSoltar, aoRematar } = {}) {
  if (ev.button !== undefined && ev.button !== 0) return
  const x0 = ev.clientX
  const y0 = ev.clientY
  let fantasma = null
  let arrastrando = false

  const limpar = () => {
    window.removeEventListener('pointermove', mover, true)
    window.removeEventListener('pointerup', soltar, true)
    window.removeEventListener('pointercancel', cancelar, true)
    window.removeEventListener('keydown', tecla, true)
    fantasma?.remove()
    document.body.classList.remove('arrastrando')
    aoRematar?.()
  }
  function mover(e) {
    if (!arrastrando) {
      if (Math.hypot(e.clientX - x0, e.clientY - y0) < LIMIAR) return
      arrastrando = true
      document.body.classList.add('arrastrando')
      fantasma = document.createElement('div')
      fantasma.className = 'fantasma-arrastre'
      fantasma.textContent = texto
      document.body.append(fantasma)
    }
    fantasma.style.transform = `translate(${e.clientX + 12}px, ${e.clientY + 12}px)`
    e.preventDefault()
    aoMover?.(e.clientX, e.clientY)
  }
  function soltar(e) {
    const foi = arrastrando
    if (foi) {
      // O clic que dispara o navegador ao soltar tras un arrastre non conta.
      const enguli = (c) => { c.stopPropagation(); c.preventDefault() }
      window.addEventListener('click', enguli, { capture: true, once: true })
      setTimeout(() => window.removeEventListener('click', enguli, true), 0)
    }
    limpar()
    if (foi) aoSoltar?.(e.clientX, e.clientY)
  }
  function cancelar() { limpar() }
  function tecla(e) { if (e.key === 'Escape') { e.stopPropagation(); limpar() } }

  window.addEventListener('pointermove', mover, true)
  window.addEventListener('pointerup', soltar, true)
  window.addEventListener('pointercancel', cancelar, true)
  window.addEventListener('keydown', tecla, true)
}
