// Axudas de DOM e ficheiros que comparten a biblioteca (plantillas.js) e o
// editor de plantillas (plantillas-editor.js).

export function el(tag, clase, texto) {
  const n = document.createElement(tag)
  if (clase) n.className = clase
  if (texto !== undefined) n.textContent = texto
  return n
}

export function boton(texto, clase, onClick) {
  const b = el('button', clase, texto)
  b.type = 'button'
  if (onClick) b.addEventListener('click', onClick)
  return b
}

export function ficheiroABase64(file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result).split(',')[1] || '')
    reader.onerror = () => reject(reader.error)
    reader.readAsDataURL(file)
  })
}

// inserirNoCursor mete texto onde estea o cursor do textarea (ou ao final se
// nunca tivo foco) - o mesmo que fan os botóns de etiquetas do editor de
// código.
export function inserirNoCursor(area, texto) {
  const inicio = area.selectionStart ?? area.value.length
  const fin = area.selectionEnd ?? area.value.length
  area.value = area.value.slice(0, inicio) + texto + area.value.slice(fin)
  area.focus()
  area.selectionStart = area.selectionEnd = inicio + texto.length
  // Un cambio feito por código non dispara "input": avísase á man (marca de
  // cambios sen aplicar do editor).
  area.dispatchEvent(new Event('input', { bubbles: true }))
}
