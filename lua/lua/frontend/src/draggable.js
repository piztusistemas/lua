// makeDraggable permite arrastrar `modal` premendo e arrastrando `handle`
// (a cabeceira coma asa). Úsao o panel flotante "Axustes do texto", para
// poder apartalo e ver a previsualización mentres se afinan os valores.
export function makeDraggable(modal, handle) {
  handle.addEventListener('mousedown', (e) => {
    if (e.button !== 0) return
    const rect = modal.getBoundingClientRect()
    modal.style.position = 'fixed'
    modal.style.margin = '0'
    modal.style.left = rect.left + 'px'
    modal.style.top = rect.top + 'px'
    const startX = e.clientX
    const startY = e.clientY
    const startLeft = rect.left
    const startTop = rect.top

    function onMove(ev) {
      modal.style.left = startLeft + ev.clientX - startX + 'px'
      modal.style.top = startTop + ev.clientY - startY + 'px'
    }
    function onUp() {
      window.removeEventListener('mousemove', onMove)
      window.removeEventListener('mouseup', onUp)
    }
    window.addEventListener('mousemove', onMove)
    window.addEventListener('mouseup', onUp)
    e.preventDefault()
  })
}
