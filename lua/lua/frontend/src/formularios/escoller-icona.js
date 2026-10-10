// Xanela «Cambiar icona»: escolle a icona coa que sae un formulario na
// columna esquerda. Ábrese desde o menú do botón dereito (ou o «⋯» da
// biblioteca). Sen icona propia, a marcada é a que pon o programa.
//
//   escollerIcona({ t, nome, actual }) -> Promise<clave | null>   (null = cancelar)
import { CLAVES_ICONA, NOME_ICONA, svgIcona } from './iconas.js'

const ICONA_POR_DEFECTO = 'formulario'

export function escollerIcona({ t, nome, actual }) {
  return new Promise((resolve) => {
    const seleccionada = actual || ICONA_POR_DEFECTO
    const overlay = document.createElement('div')
    overlay.className = 'overlay overlay--dialog'
    const modal = document.createElement('div')
    modal.className = 'modal modal--dialog escoller-icona'
    modal.setAttribute('role', 'dialog')
    modal.setAttribute('aria-modal', 'true')
    overlay.appendChild(modal)

    const titulo = document.createElement('p')
    titulo.className = 'dialog-message'
    titulo.textContent = t('formularios.icona.escoller', 'Icona de «{nome}» na columna esquerda', { nome })
    modal.appendChild(titulo)

    let rematado = false
    function pechar(resultado) {
      if (rematado) return
      rematado = true
      document.removeEventListener('keydown', aoTecla, true)
      overlay.remove()
      resolve(resultado)
    }
    function aoTecla(e) {
      if (e.key === 'Escape') { e.preventDefault(); e.stopPropagation(); pechar(null) }
    }

    const grella = document.createElement('div')
    grella.className = 'dis-iconas escoller-icona__grella'
    grella.setAttribute('role', 'radiogroup')
    for (const clave of CLAVES_ICONA) {
      const b = document.createElement('button')
      b.type = 'button'
      b.className = 'dis-icona' + (clave === seleccionada ? ' dis-icona--sel' : '')
      b.dataset.icona = clave
      b.setAttribute('role', 'radio')
      b.setAttribute('aria-checked', String(clave === seleccionada))
      b.title = t('formularios.icona.' + clave, NOME_ICONA[clave])
      b.innerHTML = svgIcona(clave) // SVG noso, nunca do usuario
      b.addEventListener('click', () => pechar(clave))
      grella.appendChild(b)
    }
    modal.appendChild(grella)

    const accions = document.createElement('div')
    accions.className = 'modal-actions'
    const cancelar = document.createElement('button')
    cancelar.type = 'button'
    cancelar.textContent = t('dialog.cancel', 'Cancelar')
    cancelar.addEventListener('click', () => pechar(null))
    accions.appendChild(cancelar)
    modal.appendChild(accions)

    overlay.addEventListener('click', (e) => { if (e.target === overlay) pechar(null) })
    document.addEventListener('keydown', aoTecla, true)
    document.body.appendChild(overlay)
    grella.querySelector('.dis-icona--sel')?.focus()
  })
}
