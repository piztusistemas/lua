// Idioma da interface de Buxo: o mesmo ca Lua (Lua pásallo ao lanzalo e o Go
// dío con Idioma()). O galego é a fonte: cada texto leva no código a súa clave
// e o seu galego, e os dicionarios de ../../idiomas só traen os outros idiomas.
import es from '../../idiomas/es.json'
import en from '../../idiomas/en.json'
import pt from '../../idiomas/pt.json'

const DICIONARIOS = { es, en, pt }
let dic = {}
let codigo = 'gl'

export function fixarIdioma(c) {
  codigo = DICIONARIOS[c] ? c : 'gl'
  dic = DICIONARIOS[codigo] || {}
}

export function idiomaActual() {
  return codigo
}

// t(clave, galego, vars): o texto no idioma activo ou, se falta, o galego.
// As variables {nome} substitúense polos valores de vars.
export function t(clave, galego, vars) {
  let v = dic[clave] ?? galego ?? clave
  if (vars) for (const k in vars) v = v.replaceAll(`{${k}}`, vars[k])
  return v
}

// aplicarTraducions traduce o HTML estático marcado con data-i18n*. O galego
// escrito no propio HTML é a reserva.
export function aplicarTraducions(raiz = document) {
  raiz.querySelectorAll('[data-i18n]').forEach((n) => { n.textContent = t(n.dataset.i18n, n.textContent) })
  raiz.querySelectorAll('[data-i18n-title]').forEach((n) => { n.title = t(n.dataset.i18nTitle, n.title) })
  raiz.querySelectorAll('[data-i18n-placeholder]').forEach((n) => { n.placeholder = t(n.dataset.i18nPlaceholder, n.placeholder) })
  raiz.querySelectorAll('[data-i18n-aria-label]').forEach((n) => {
    n.setAttribute('aria-label', t(n.dataset.i18nAriaLabel, n.getAttribute('aria-label')))
  })
}
