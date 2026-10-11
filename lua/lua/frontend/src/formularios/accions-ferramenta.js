// Que se pode facer con cada ferramenta da columna esquerda (cronograma,
// actas e cada formulario): as MESMAS accións no menú do botón dereito da
// icona e no menú «⋯» da biblioteca. Só devolve a lista; quen chama pinta o menú.
//
//   biblioteca()      estado actual (biblioteca-modelo.js)
//   gardar(nova)      persiste un novo estado (Promise)
//   eliminar(id)      borra un formulario, con confirmación (Promise)
//   exportar(id)      (opcional) garda o formulario nun ficheiro para compartilo (Promise)
//   cambiarIcona(id)  (opcional) abre a xanela para escoller a icona da columna (Promise)
//   quitar(id)        (opcional) saca a ferramenta do proxecto aberto, con confirmación (Promise).
//                     Se se dá, substitúe a «Eliminar formulario»: nun proxecto, quitar é o que se quere.
import { esIntegrada, estaFixa, estaAgochada, podeBorrar, fixar, desfixar, agochar } from './biblioteca-modelo.js'

export function itemsDeFerramenta(id, { t, biblioteca, gardar, eliminar, exportar, cambiarIcona, quitar }) {
  const b = biblioteca()
  const items = []
  const agochada = estaAgochada(b, id)
  items.push({
    texto: agochada ? t('formularios.menu.amosar', 'Amosar na columna') : t('formularios.menu.ocultar', 'Ocultar da columna'),
    titulo: agochada ? undefined : t('formularios.menu.ocultar.title', 'Non se borra: segue na biblioteca e podes volver amosalo'),
    aoPremer: () => gardar(agochar(b, id, !agochada)),
  })
  const itemQuitar = quitar ? {
    texto: t('formularios.menu.quitar', 'Quitar do proxecto…'), perigo: true,
    titulo: t('formularios.menu.quitar.title', 'Saca esta ferramenta e os seus datos do proxecto. A biblioteca común non cambia.'),
    aoPremer: () => quitar(id),
  } : null
  if (esIntegrada(id)) return itemQuitar ? [...items, { separador: true }, itemQuitar] : items

  items.unshift(estaFixa(b, id)
    ? { texto: t('formularios.menu.desfixar', 'Quitar da zona fixa'), titulo: t('formularios.menu.desfixar.title', 'Pasa á zona de abaixo (onde si se pode borrar)'), aoPremer: () => gardar(desfixar(b, id)) }
    : { texto: t('formularios.menu.fixar', 'Fixar na zona superior'), titulo: t('formularios.menu.fixar.title', 'Xunto ao cronograma e ás actas. Os fixos non se poden borrar, só ocultar'), aoPremer: () => gardar(fixar(b, id)) })
  if (cambiarIcona) {
    items.push({
      texto: t('formularios.menu.icona', 'Cambiar icona…'),
      titulo: t('formularios.menu.icona.title', 'Escolle a icona coa que sae o formulario na columna esquerda'),
      aoPremer: () => cambiarIcona(id),
    })
  }
  if (exportar) {
    items.push({
      texto: t('formularios.menu.exportar', 'Exportar formulario…'),
      titulo: t('formularios.menu.exportar.title', 'Garda o formulario nun ficheiro (con ou sen datos) para pasarllo a un compañeiro, que o importará á súa biblioteca'),
      aoPremer: () => exportar(id),
    })
  }
  items.push({ separador: true })
  if (itemQuitar) return [...items, itemQuitar]
  const borrable = podeBorrar(b, id)
  items.push({
    texto: t('formularios.menu.eliminar', 'Eliminar formulario…'), perigo: true, desactivado: !borrable,
    titulo: borrable ? undefined : t('formularios.menu.eliminar.fixo', 'Os formularios da zona fixa non se poden borrar: quítao da zona fixa primeiro (ou ocúltao).'),
    aoPremer: () => eliminar(id),
  })
  return items
}
