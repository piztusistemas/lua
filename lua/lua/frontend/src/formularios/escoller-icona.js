// Xanela «Cambiar icona»: escolle a icona e a cor coas que sae un formulario
// na columna esquerda. Ábrese desde o menú do botón dereito (ou o «⋯» da
// biblioteca). Sen icona propia, a marcada é a que pon o programa. Premer
// unha cor só a escolle (as iconas pintanse nela); premer unha icona, ou
// Gardar, pecha coa icona e a cor marcadas.
//
//   escollerIcona({ t, nome, actual, cor }) -> Promise<{ icona, cor } | null>   (null = cancelar)
import { CLAVES_ICONA, NOME_ICONA, CLAVES_COR, NOME_COR, svgIcona, valorCor } from './iconas.js'

const ICONA_POR_DEFECTO = 'formulario'

export function escollerIcona({ t, nome, actual, cor }) {
  return new Promise((resolve) => {
    const seleccionada = actual || ICONA_POR_DEFECTO
    let corSel = CLAVES_COR.includes(cor) ? cor : CLAVES_COR[0]
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

    const cores = document.createElement('div')
    cores.className = 'dis-cores escoller-icona__cores'
    cores.setAttribute('role', 'radiogroup')
    cores.setAttribute('aria-label', t('formularios.icona.cor', 'Cor da icona'))
    const grella = document.createElement('div')
    grella.className = 'dis-iconas escoller-icona__grella'
    grella.setAttribute('role', 'radiogroup')
    function marcarCor() {
      grella.style.setProperty('--cor-formulario', valorCor(corSel))
      for (const b of cores.children) {
        const sel = b.dataset.cor === corSel
        b.classList.toggle('dis-cor--sel', sel)
        b.setAttribute('aria-checked', String(sel))
      }
    }
    for (const clave of CLAVES_COR) {
      const b = document.createElement('button')
      b.type = 'button'
      b.className = 'dis-cor'
      b.dataset.cor = clave
      b.setAttribute('role', 'radio')
      b.style.setProperty('--cor-formulario', valorCor(clave))
      b.title = t('formularios.cor.' + clave, NOME_COR[clave])
      b.setAttribute('aria-label', b.title)
      b.addEventListener('click', () => { corSel = clave; marcarCor() })
      cores.appendChild(b)
    }
    modal.appendChild(cores)
    for (const clave of CLAVES_ICONA) {
      const b = document.createElement('button')
      b.type = 'button'
      b.className = 'dis-icona' + (clave === seleccionada ? ' dis-icona--sel' : '')
      b.dataset.icona = clave
      b.setAttribute('role', 'radio')
      b.setAttribute('aria-checked', String(clave === seleccionada))
      b.title = t('formularios.icona.' + clave, NOME_ICONA[clave])
      b.innerHTML = svgIcona(clave) // SVG noso, nunca do usuario
      b.addEventListener('click', () => pechar({ icona: clave, cor: corSel }))
      grella.appendChild(b)
    }
    modal.appendChild(grella)
    marcarCor()

    const accions = document.createElement('div')
    accions.className = 'modal-actions'
    const cancelar = document.createElement('button')
    cancelar.type = 'button'
    cancelar.textContent = t('dialog.cancel', 'Cancelar')
    cancelar.addEventListener('click', () => pechar(null))
    accions.appendChild(cancelar)
    const gardar = document.createElement('button')
    gardar.type = 'button'
    gardar.className = 'primary'
    gardar.textContent = t('formularios.icona.gardar', 'Gardar')
    gardar.addEventListener('click', () => pechar({ icona: seleccionada, cor: corSel }))
    accions.appendChild(gardar)
    modal.appendChild(accions)

    overlay.addEventListener('click', (e) => { if (e.target === overlay) pechar(null) })
    document.addEventListener('keydown', aoTecla, true)
    document.body.appendChild(overlay)
    grella.querySelector('.dis-icona--sel')?.focus()
  })
}
