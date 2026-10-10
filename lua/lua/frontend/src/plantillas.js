// Biblioteca de PLANTILLAS (os "estilos" dos centros: cabeceira co logotipo e
// o nome do centro, pé cos teléfonos, marxes...). O backend está en
// plantillas.go; aquí só hai UI, e toda a comunicación pasa polo obxecto
// `api` que inxecta main.js (este módulo non importa nunca os bindings de
// Wails directamente, así se pode probar sen backend).
//
// Dúas ventás:
//   - abrirBibliotecaPlantillas: a lista. Escoller cal se aplica, crear,
//     duplicar, editar, eliminar, e tamén importar/exportar unha plantilla
//     (ficheiro .json: texto + imaxes) para compartila cos
//     compañeiros - ver ExportarPlantilla/ImportarPlantilla en
//     plantillas_intercambio.go.
//   - o editor (abrirEditorPlantilla, tamén exportado para "✨ Crear con
//     IA"): nome, LaTeX, Markdown, variables, imaxes, vista previa real.
//
// O marcador {{CORPO}} é o contrato con plantillas.go: onde el estea vai o
// documento xerado. A UI insiste nel (axuda, validación e o botón que o
// insire) porque é o único erro que deixa unha plantilla inservible.

import { confirmar } from './comun/dialogs.js'
import { el, boton } from './plantillas-util.js'
import { montarEditorPlantilla } from './plantillas-editor.js'

function crearOverlay(clase) {
  const overlay = document.createElement('div')
  overlay.className = 'overlay hidden'
  const modal = document.createElement('div')
  modal.className = 'modal' + (clase ? ' ' + clase : '')
  overlay.appendChild(modal)
  document.body.appendChild(overlay)
  return overlay
}

// ---------- Ventá 1: a biblioteca ----------

let bibliotecaOverlay = null

// abrirBibliotecaPlantillas amosa a lista. onEscoller avísalle a main.js de
// que cambiou a plantilla activa (para refrescar o selector da barra e
// volver xerar), e chámase tamén tras gardar/eliminar porque calquera das
// dúas pode cambiar o que hai que amosar aí.
export async function abrirBibliotecaPlantillas({ t, api, ficheiroActual, onEscoller }) {
  if (!bibliotecaOverlay) {
    bibliotecaOverlay = crearOverlay('modal--plantillas')
    bibliotecaOverlay.addEventListener('click', (e) => {
      if (e.target === bibliotecaOverlay) bibliotecaOverlay.classList.add('hidden')
    })
  }
  const overlay = bibliotecaOverlay
  const modal = overlay.querySelector('.modal')
  const pechar = () => overlay.classList.add('hidden')

  // `aviso` sobrevive aos repintados (pintar() baleira o modal enteiro): o
  // que importa/exporta escríbeo aquí e pintar() vólveo poñer na barra de
  // estado, así o "Plantilla importada" ou o erro non desaparecen ao
  // recargar a lista.
  let aviso = ''
  const mostrarAviso = (msg) => { aviso = msg || '' }

  async function pintar() {
    modal.innerHTML = ''
    modal.appendChild(el('h2', null, '📐 ' + t('plantillas.title', 'Plantillas (estilos dos centros)')))
    modal.appendChild(el('p', null, t('plantillas.intro',
      'A plantilla decide a maqueta do informe: cabeceira co logotipo e o nome do centro, pé cos teléfonos e o nome de quen orienta, marxes… O curso segue sendo o mesmo. Aplícase á vista previa e a todo o que exportes: PDF, LaTeX, Markdown, Word e OpenDocument. Exporta unha plantilla e importa noutro equipo para que todo o departamento use o mesmo estilo.')))

    // Vai baixo a barra, pero decláloa antes: os manexadores dos botóns
    // (importar, exportar) escriben nela.
    const barraEstado = el('p', 'ai-row__status plantillas-toolbar-estado')
    barraEstado.textContent = aviso

    const barra = el('div', 'plantillas-toolbar')
    barra.appendChild(boton('➕ ' + t('plantillas.new', 'Nova en branco'), null, async () => {
      pechar()
      await abrirEditorPlantilla({ t, api, plantilla: null, onGardada: recargar })
    }))
    barra.appendChild(boton('✨ ' + t('plantillas.newAI', 'Crear con IA'), null, async () => {
      pechar()
      await abrirEditorPlantilla({ t, api, plantilla: null, abrirIA: true, onGardada: recargar })
    }))
    // Importar: ao lado de "Crear con IA". Trae unha plantilla que compartiu
    // un compañeiro (ficheiro .json: texto + imaxes) e engádea á
    // biblioteca coma unha máis.
    barra.appendChild(boton('📥 ' + t('plantillas.import', 'Importar'), null, async () => {
      mostrarAviso('')
      barraEstado.textContent = ''
      try {
        const nova = await api.importar()
        if (!nova || !nova.id) return // diálogo cancelado
        mostrarAviso(t('plantillas.imported', 'Plantilla "{nome}" importada.', { nome: nova.nome }))
        await pintar()
        if (onEscoller) onEscoller()
      } catch (err) {
        barraEstado.textContent = t('plantillas.importError', 'Non se puido importar: {msg}',
          { msg: err && err.message ? err.message : err })
      }
    }))
    modal.appendChild(barra)
    modal.appendChild(barraEstado)

    let estado
    try {
      estado = await api.listar()
    } catch (err) {
      modal.appendChild(el('p', 'ai-row__status', String(err)))
      return
    }

    const lista = el('div', 'plantillas-lista')
    lista.appendChild(fila({
      nome: t('plantillas.none', 'Ningunha'),
      descricion: t('plantillas.noneDesc', 'O informe sae coa maqueta por defecto de Lúa.'),
      activa: !estado.activa,
      accions: [],
      onUsar: () => escoller(''),
    }))
    for (const p of estado.plantillas || []) {
      lista.appendChild(fila({
        nome: p.nome,
        descricion: p.descricion || '',
        deFabrica: p.deFabrica,
        activa: estado.activa === p.id,
        onUsar: () => escoller(p.id),
        accions: [
          boton(t('plantillas.edit', 'Editar'), null, async () => {
            pechar()
            await abrirEditorPlantilla({ t, api, plantilla: p, onGardada: recargar })
          }),
          boton(t('plantillas.duplicate', 'Duplicar'), null, async () => {
            await api.duplicar(p.id)
            await pintar()
          }),
          // Exportar: cada plantilla da biblioteca pódese gardar nun ficheiro
          // .json para pasarllo a un compañeiro.
          boton('📤 ' + t('plantillas.export', 'Exportar'), null, async () => {
            mostrarAviso('')
            barraEstado.textContent = ''
            try {
              const ruta = await api.exportar(p.id)
              if (!ruta) return // diálogo cancelado
              barraEstado.textContent = t('plantillas.exported', 'Exportada a {ruta}', { ruta })
            } catch (err) {
              barraEstado.textContent = t('plantillas.exportError', 'Non se puido exportar: {msg}',
                { msg: err && err.message ? err.message : err })
            }
          }),
          boton(t('plantillas.delete', 'Eliminar'), 'danger', async () => {
            // confirmar() e non window.confirm: en macOS o nativo devolve
            // sempre false sen preguntar (ver dialogs.js).
            if (!await confirmar(t('plantillas.confirmDelete', 'Eliminar a plantilla "{nome}"?', { nome: p.nome }))) return
            await api.eliminar(p.id)
            await pintar()
            if (onEscoller) onEscoller()
          }),
        ],
      }))
    }
    modal.appendChild(lista)

    const accions = el('div', 'modal-actions')
    accions.appendChild(boton(t('plantillas.close', 'Pechar'), 'primary', pechar))
    modal.appendChild(accions)
  }

  function fila({ nome, descricion, deFabrica, activa, onUsar, accions }) {
    const card = el('div', 'plantilla-card' + (activa ? ' plantilla-card--activa' : ''))
    const info = el('div', 'plantilla-card__info')
    const titulo = el('div', 'plantilla-card__nome', nome)
    if (deFabrica) titulo.appendChild(el('span', 'plantilla-card__badge', t('plantillas.builtin', 'de exemplo')))
    if (activa) titulo.appendChild(el('span', 'plantilla-card__badge plantilla-card__badge--activa', t('plantillas.inUse', 'en uso')))
    info.appendChild(titulo)
    if (descricion) info.appendChild(el('div', 'plantilla-card__desc', descricion))
    card.appendChild(info)

    const btns = el('div', 'plantilla-card__accions')
    btns.appendChild(boton(activa ? t('plantillas.inUse', 'en uso') : t('plantillas.use', 'Usar'), 'primary', async () => {
      if (activa) return
      await onUsar()
      await pintar()
    }))
    for (const b of accions || []) btns.appendChild(b)
    card.appendChild(btns)
    return card
  }

  async function escoller(id) {
    await api.escoller(id, ficheiroActual ? ficheiroActual() : '')
    if (onEscoller) onEscoller(id)
  }

  async function recargar() {
    await pintar()
    overlay.classList.remove('hidden')
    if (onEscoller) onEscoller()
  }

  await pintar()
  overlay.classList.remove('hidden')
}

// ---------- Ventá 2: o editor dunha plantilla ----------

let editorOverlay = null

// abrirEditorPlantilla edita unha plantilla existente ou crea unha nova
// (plantilla null). O editor (plantillas-editor.js) traballa sobre un BORRADOR:
// ata premer Aplicar ou Gardar non se toca nada do disco, agás as imaxes.
//
// Se hai xanelas nativas (api.abrirXanela, ver xanelaplantilla.go) ábrese nunha
// xanela propia, maximizada, que se pode levar a outro monitor e deixar aberta
// mentres se proba (Aplicar garda sen pechar). Se non (proba en navegador), ábrese
// a pantalla completa dentro desta xanela.
export async function abrirEditorPlantilla({ t, api, plantilla, abrirIA, onGardada }) {
  if (api.abrirXanela) {
    let aberta = false
    try { aberta = await api.abrirXanela(plantilla ? plantilla.id : '', JSON.stringify({ plantilla, abrirIA: !!abrirIA, documento: api.documentoActual?.() ?? null })) } catch (_) { /* cae ao overlay */ }
    if (aberta) return
  }
  if (!editorOverlay) {
    editorOverlay = crearOverlay('modal--plantilla-editor')
    editorOverlay.classList.add('tarefa-overlay', 'plantilla-overlay')
  }
  const overlay = editorOverlay
  const modal = overlay.querySelector('.modal')
  overlay.classList.remove('hidden')
  await montarEditorPlantilla(modal, {
    t, api, plantilla, abrirIA, documento: api.documentoActual?.() ?? null, aoGardada: onGardada,
    aoPechar: () => overlay.classList.add('hidden'),
  })
}
