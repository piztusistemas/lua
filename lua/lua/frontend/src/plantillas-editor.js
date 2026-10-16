// O EDITOR DE PLANTILLAS: unha pantalla grande, en bloques e columnas, que se
// pode montar onde se queira:
//   - como pantalla completa dentro da xanela principal (plantillas.js,
//     abrirEditorPlantilla: proba en navegador ou se non hai xanelas nativas), ou
//   - como páxina enteira dunha xanela propia do sistema, que se pode levar a
//     outro monitor (main.js, entrarModoPlantilla; ver xanelaplantilla.go).
//
// Traballa sobre un BORRADOR: ata premer Aplicar (garda sen pechar) ou Gardar
// (garda e pecha) non se toca a plantilla do disco, agás as imaxes, que se soben
// ao momento. Aplicar refresca a vista previa do propio editor e avisa á xanela
// principal (api.aplicada) para que a biblioteca e o informe usen xa a nova
// versión: así se pode ir axustando a plantilla de maneira continua, sen pechar,
// probar, abrir e volver modificar.
//
// Tres columnas de bloques (Datos e recursos | PDF/LaTeX | Word/OpenDocument), a
// vista previa á dereita e, se hai IA configurada, o asistente da plantilla
// (asistente/ferramentas-plantilla.js) coas fichas por defecto (portada, letra,
// interliñado, marxes, cabeceira, pé...) e as que engada o usuario.
//
// Hai moitos bloques, así que só están á vista os esenciais: unha fila de botóns
// (o «lanzador», baixo a cabeceira) chama a cada un dos demais, que se despregan
// ao premelos e se pechan co seu ✕ ou co mesmo botón. Cada columna desaparece se
// non lle queda ningún bloque á vista, e o que se ten aberto lémbrase.
//
// O marcador {{CORPO}} é o contrato con plantillas.go: onde el estea vai o
// documento xerado. A UI insiste nel (axuda, validación e o botón que o
// insire) porque é o único erro que deixa unha plantilla inservible.
import { confirmar } from './comun/dialogs.js'
import { el, boton, ficheiroABase64, inserirNoCursor } from './plantillas-util.js'
import { crearPanelAsistente } from './asistente/panel.js'
import { crearFerramentasPlantilla } from './asistente/ferramentas-plantilla.js'
import { TAREFAS_PLANTILLA } from './asistente/tarefas.js'
import { logoProvedor } from './asistente/logos.js'
import { montarBloqueDeseno } from './deseno/editor.js'
import { marcaEnPunto, caixaMarca } from './deseno/modelo.js'

const MARCADOR_CORPO = '{{CORPO}}'
const MARCADOR_PORTADA = '{{PORTADA}}'
const MARCADOR_SINATURA = '{{SINATURA}}'
const CLAVE_AUTO_VISTA = 'lua.plantilla.autoVista'
const CLAVE_BLOCOS = 'lua.plantilla.blocos'
// Os bloques á vista de primeiras: o resto chámase co lanzador. O deseño vai
// primeiro: é o que cambia o aspecto do PDF sen saber LaTeX.
const BLOCOS_POR_DEFECTO = ['deseno']
// Quen xa tiña bloques gardados de antes do bloque Deseño veo unha vez aberto.
const CLAVE_DESENO_VISTO = 'lua.plantilla.desenoVisto'

const ICONA_X = '<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round"><path d="M5 5l10 10M15 5L5 15"/></svg>'

// PLANTILLA_BALEIRA é o punto de partida dunha plantilla nova: un fragmento
// (sen \documentclass) cunha cabeceira e un pé mínimos, para que quen a abra
// vexa de primeiras como se usan {{CORPO}} e as variables que enche Lua.
export const PLANTILLA_BALEIRA = `\\pagestyle{fancy}
\\fancyhf{}
\\fancyhead[L]{\\bfseries {{CENTRO}} }
\\fancyhead[R]{ {{DEPARTAMENTO}} }
\\fancyfoot[C]{\\thepage}

{{PORTADA}}

{{CORPO}}

{{SINATURA}}
`

function lerAutoVista() {
  try { return localStorage.getItem(CLAVE_AUTO_VISTA) !== '0' } catch { return true }
}
function gardarAutoVista(v) {
  try { localStorage.setItem(CLAVE_AUTO_VISTA, v ? '1' : '0') } catch { /* sen persistencia */ }
}
function lerBlocosAbertos() {
  try {
    const bruto = JSON.parse(localStorage.getItem(CLAVE_BLOCOS) || 'null')
    if (Array.isArray(bruto)) {
      const set = new Set(bruto.map(String))
      if (!localStorage.getItem(CLAVE_DESENO_VISTO)) {
        set.add('deseno')
        localStorage.setItem(CLAVE_DESENO_VISTO, '1')
      }
      return set
    }
  } catch { /* sen persistencia */ }
  return new Set(BLOCOS_POR_DEFECTO)
}
function gardarBlocosAbertos(set) {
  try { localStorage.setItem(CLAVE_BLOCOS, JSON.stringify([...set])) } catch { /* sen persistencia */ }
}
// ofiLimpa: o bloque de Word/OpenDocument sen valores baleiros, ordenado, para
// comparar o borrador co gardado.
function ofiLimpa(o) {
  const out = {}
  for (const k of Object.keys(o || {}).sort()) {
    const v = o[k]
    if (v !== '' && v !== 0 && v !== undefined && v !== null) out[k] = v
  }
  return out
}

// montarEditorPlantilla(container, opcións) monta o editor en `container` e
// devolve { cambiado, pedirPechar, destruir, focus }.
//   t, api       tradución e bindings (ver apiPlantillas en main.js). Opcionais en
//                api: previsualizarOfi, fontes, aplicada(plantilla), asistente
//                ({api, idioma}: activa o asistente de IA)
//   plantilla    a que se edita (null = nova)
//   documento    a petición do documento que se está editando (vista previa con el); null se non hai
//   abrirIA      abre xa o asistente ("Crear con IA" da biblioteca)
//   aoGardada    (plantilla) cando se garda e se pecha (Gardar)
//   aoPechar     () hai que quitar a pantalla (xa confirmado o descarte)
//   aoCambiarSucio  (bool) cando hai/deixa de haber cambios sen aplicar
export async function montarEditorPlantilla(container, {
  t, api, plantilla, abrirIA, documento = null, aoGardada, aoPechar = () => {}, aoCambiarSucio = () => {},
}) {
  // Os deseños por defecto (portada, sinatura, introdución) viven no backend
  // (informe_fragmentos.go): unha soa fonte de verdade. Se non se poden ler
  // (proba sen backend), os botóns "Usar o de Lua" quedan sen efecto.
  let porDefecto = { portada: '', sinatura: '', introducion: '', portadaOfimatica: '', estilo: '', textos: [] }
  try {
    if (api.fragmentosPorDefecto) porDefecto = { ...porDefecto, ...(await api.fragmentosPorDefecto()) }
  } catch (_) { /* sen deseños por defecto */ }

  // Copia de traballo. `imaxes` mantense sincronizada co backend en canto se
  // sobe/borra unha. O obxecto `ofi` (Word/OpenDocument) mantén SEMPRE a mesma
  // identidade: os campos capturan esta referencia.
  let actual = plantilla
    ? JSON.parse(JSON.stringify(plantilla))
    : {
        id: '', nome: '', descricion: '', latex: PLANTILLA_BALEIRA, markdown: '', motor: '', variables: [], imaxes: [], ofimatica: {},
        portada: porDefecto.portada, sinatura: porDefecto.sinatura, introducion: porDefecto.introducion,
        estilo: '', textos: {}, deseno: {},
      }
  actual.portada = actual.portada || ''
  actual.sinatura = actual.sinatura || ''
  actual.introducion = actual.introducion || ''
  actual.estilo = actual.estilo || ''
  actual.deseno = actual.deseno || {}
  actual.variables = actual.variables || []
  actual.imaxes = actual.imaxes || []
  const ofi = { ...(actual.ofimatica || {}) }
  actual.ofimatica = ofi
  // Textos do documento que esta plantilla cambia (clave -> texto). Mesma
  // identidade sempre: os campos capturan este obxecto.
  const textos = { ...(actual.textos || {}) }
  actual.textos = textos

  // Os bloques rexístranse aquí para que o lanzador os poida chamar (ver
  // montarLanzador, máis abaixo). `id` é a súa clave para lembrar se estaba aberto.
  const registroBlocos = []
  const blocosAbertos = lerBlocosAbertos()
  const bloque = (titulo, id, grupo, etq) => {
    const b = el('section', 'plantilla-bloque')
    b.dataset.bloque = id
    const cab = el('div', 'plantilla-bloque__cab')
    cab.appendChild(el('h3', 'plantilla-seccion', titulo))
    const x = boton('✕', 'plantilla-bloque__x', () => alternarBloque(id, false))
    x.title = t('plantillas.bloque.pechar', 'Pechar este bloque')
    x.setAttribute('aria-label', t('plantillas.bloque.pechar', 'Pechar este bloque'))
    cab.appendChild(x)
    b.appendChild(cab)
    registroBlocos.push({ id, grupo, etq: etq || titulo, nodo: b })
    return b
  }
  let alternarBloque = () => {}

  let pechado = false
  let ocupado = false
  let compilando = false

  container.classList.add('plantilla-pantalla')
  container.setAttribute('role', 'dialog')
  container.setAttribute('aria-modal', 'true')

  // Amosar os cambios de Word/OpenDocument nos campos (tras gardar, desfacer...).
  const refrescadoresOfi = []
  const pintarOfimatica = () => refrescadoresOfi.forEach((f) => f())

  // ---------- Identidade: nome, descrición e compilador ----------
  const nomeLabel = el('label', 'modal-label', t('plantillas.name', 'Nome'))
  const nomeInput = el('input')
  nomeInput.type = 'text'
  nomeInput.value = actual.nome
  nomeLabel.appendChild(nomeInput)

  const descLabel = el('label', 'modal-label', t('plantillas.description', 'Descrición (unha liña)'))
  const descInput = el('input')
  descInput.type = 'text'
  descInput.value = actual.descricion
  descLabel.appendChild(descInput)

  const motorLabel = el('label', 'modal-label', t('plantillas.engine', 'Compilador'))
  const motorSelect = el('select')
  for (const [valor, texto] of [
    ['', t('plantillas.engineDefault', 'O de Opcións (por defecto)')],
    ['pdflatex', 'pdflatex'],
    ['xelatex', 'xelatex'],
    ['lualatex', 'lualatex'],
  ]) {
    const opt = el('option', null, texto)
    opt.value = valor
    motorSelect.appendChild(opt)
  }
  motorSelect.value = actual.motor || ''
  motorLabel.appendChild(motorSelect)

  const bloqueIdentidade = bloque(t('plantillas.bloque.identidade', 'Identidade'), 'identidade', 'datos', t('plantillas.chip.identidade', 'Identidade'))
  bloqueIdentidade.append(nomeLabel, descLabel, motorLabel)

  // ---------- Variables ----------
  const bloqueVariables = bloque(t('plantillas.variables', 'Variables'), 'variables', 'datos', t('plantillas.chip.variables', 'Variables'))
  const varsWrap = el('div', 'plantilla-vars')
  bloqueVariables.appendChild(varsWrap)
  const varsBarra = el('div', 'plantillas-toolbar')
  varsBarra.appendChild(boton('➕ ' + t('plantillas.addVar', 'Engadir variable'), null, () => {
    actual.variables.push({ nome: '', etiqueta: '', valor: '' })
    pintarVariables()
    marcarCambios()
  }))
  bloqueVariables.appendChild(varsBarra)

  function pintarVariables() {
    varsWrap.innerHTML = ''
    if (actual.variables.length === 0) {
      varsWrap.appendChild(el('p', 'plantilla-axuda', t('plantillas.noVars',
        'Sen variables. Engade unha para que un dato fixo (o teléfono, o correo do centro…) se poida cambiar sen tocar o LaTeX.')))
      return
    }
    actual.variables.forEach((v, i) => {
      const fila = el('div', 'plantilla-var')
      const nome = el('input', 'plantilla-var__nome')
      nome.type = 'text'
      nome.placeholder = t('plantillas.varName', 'NOME')
      nome.value = v.nome
      nome.addEventListener('input', () => { v.nome = nome.value })
      const etiqueta = el('input')
      etiqueta.type = 'text'
      etiqueta.placeholder = t('plantillas.varLabel', 'Etiqueta')
      etiqueta.value = v.etiqueta
      etiqueta.addEventListener('input', () => { v.etiqueta = etiqueta.value })
      const valor = el('input')
      valor.type = 'text'
      valor.placeholder = t('plantillas.varValue', 'Valor')
      valor.value = v.valor
      valor.addEventListener('input', () => { v.valor = valor.value })
      fila.appendChild(nome)
      fila.appendChild(etiqueta)
      fila.appendChild(valor)
      fila.appendChild(boton(t('plantillas.insert', 'Inserir'), null, () => {
        inserirNoCursor(latexArea, '{{' + (v.nome || 'NOME').toUpperCase() + '}}')
      }))
      fila.appendChild(boton('🗑', 'danger', () => {
        actual.variables.splice(i, 1)
        pintarVariables()
        marcarCambios()
      }))
      varsWrap.appendChild(fila)
    })
  }
  pintarVariables()

  // ---------- Imaxes ----------
  const bloqueImaxes = bloque('🖼️ ' + t('plantillas.images', 'Imaxes da plantilla'), 'imaxes', 'datos', t('plantillas.chip.imaxes', 'Imaxes'))
  bloqueImaxes.appendChild(el('p', 'plantilla-axuda', t('plantillas.imagesHelp',
    'Gárdanse coa plantilla (non a carón do curso), así que valen para calquera cronograma, estea onde estea o ficheiro. Pódense engadir varias á vez: imaxes (PNG, JPEG, GIF, WebP, TIFF…, e RAW, SVG ou HEIC se hai conversor) e PDF, que non perden calidade. No LaTeX usa o nome tal cal, ex. \\includegraphics{logotipo.pdf}; en Word/ODT, @imaxe logotipo.pdf 10.')))
  const imaxesWrap = el('div', 'plantilla-imaxes')
  bloqueImaxes.appendChild(imaxesWrap)

  const imaxeFila = el('div', 'image-upload-row')
  const imaxeEstado = el('span', 'image-upload-row__status')
  imaxeFila.appendChild(imaxeEstado)
  const imaxeLabel = el('label', 'image-upload-row__btn image-upload-row__btn--destacado', '🖼️ ' + t('plantillas.addImage', 'Engadir imaxe…'))
  const imaxeInput = el('input')
  imaxeInput.type = 'file'
  imaxeInput.multiple = true
  // Sen restrición de tipo demasiado estrita: WebKitGTK oculta no selector o
  // que non encaixe (un PDF), e aquí valen imaxes, PDF vectoriais e formatos
  // que Lua converte (RAW, SVG, HEIC...).
  imaxeInput.accept = 'image/*,application/pdf,.pdf,.svg,.tif,.tiff,.heic,.avif,.webp,.dng,.cr2,.nef,.arw'
  imaxeInput.hidden = true
  imaxeInput.addEventListener('change', async () => {
    const ficheiros = [...(imaxeInput.files || [])]
    imaxeInput.value = ''
    if (!ficheiros.length) return
    const erros = []
    try {
      // Unha plantilla sen gardar aínda non ten onde meter a imaxe: gárdase
      // primeiro (co que haxa escrito) e despois sóbense.
      if (!actual.id) {
        imaxeEstado.textContent = t('plantillas.savingFirst', 'Gardando a plantilla antes de engadir a imaxe…')
        await gardar({ pechar: false })
        if (!actual.id) return
      }
      for (const file of ficheiros) {
        imaxeEstado.textContent = t('plantillas.uploading', 'Subindo…') + ' ' + file.name
        try {
          const nome = await api.subirImaxe({ id: actual.id, fileName: file.name, dataB64: await ficheiroABase64(file) })
          actual.imaxes.push(nome)
          pintarImaxes()
        } catch (err) {
          erros.push(`${file.name}: ${err && err.message ? err.message : err}`)
        }
      }
    } catch (err) {
      erros.push(String(err && err.message ? err.message : err))
    }
    // Un ficheiro que falla non impide subir os demais.
    imaxeEstado.textContent = erros.join(' · ')
  })
  imaxeLabel.appendChild(imaxeInput)
  imaxeFila.appendChild(imaxeLabel)
  bloqueImaxes.appendChild(imaxeFila)

  function pintarImaxes() {
    imaxesWrap.innerHTML = ''
    if (actual.imaxes.length === 0) {
      imaxesWrap.appendChild(el('p', 'plantilla-axuda', t('plantillas.noImages', 'Aínda non hai imaxes.')))
      return
    }
    for (const nome of actual.imaxes) {
      // Tres liñas: 1) a imaxe co seu nome (e o botón de borrar), 2) inserir no
      // LaTeX / Markdown, 3) usala na cabeceira / portada de Word e ODT.
      const fila = el('div', 'plantilla-imaxe')
      const cab = el('div', 'plantilla-imaxe__cab')
      const eImaxe = /\.(png|jpe?g|gif|webp|bmp|tiff?|pdf)$/i.test(nome)
      const mini = el('img', 'plantilla-imaxe__mini')
      mini.alt = ''
      mini.hidden = true
      cab.appendChild(mini)
      if (eImaxe && api.vistaImaxe) {
        Promise.resolve(api.vistaImaxe(actual.id, nome)).then((uri) => {
          if (uri) { mini.src = uri; mini.hidden = false }
        }).catch(() => {})
      }
      cab.appendChild(el('code', null, nome))
      if (/\.pdf$/i.test(nome)) {
        const etq = el('span', 'plantilla-imaxe__etq', 'PDF')
        etq.title = t('plantillas.pdfVectorial', 'Vectorial: no PDF do informe non perde calidade. En Word/OpenDocument úsase unha imaxe de píxeles da súa páxina 1.')
        cab.appendChild(etq)
      }
      cab.appendChild(boton('🗑', 'danger', async () => {
        await api.eliminarImaxe(actual.id, nome)
        actual.imaxes = actual.imaxes.filter((n) => n !== nome)
        pintarImaxes()
      }))
      fila.appendChild(cab)

      const linhaPdf = el('div', 'plantilla-imaxe__acc')
      linhaPdf.appendChild(boton(t('plantillas.insertLatex', 'Inserir no LaTeX'), null, () => {
        inserirNoCursor(latexArea, '\\includegraphics[height=1.5cm]{' + nome + '}')
      }))
      linhaPdf.appendChild(boton(t('plantillas.insertMd', 'Inserir no Markdown'), null, () => {
        inserirNoCursor(mdArea, '![](images/' + nome + ')')
      }))
      fila.appendChild(linhaPdf)

      // Word/OpenDocument: a portada e a cabeceira son da plantilla e usan @imaxe.
      const linhaOfi = el('div', 'plantilla-imaxe__acc')
      linhaOfi.appendChild(boton(t('plantillas.insertOfiCab', 'Na cabeceira de Word/ODT'), null, () => {
        const v = cabInput.value.replace(/\s+$/, '')
        cabInput.value = (v ? v + '\n' : '') + '@imaxe ' + nome + ' 10'
        ofi.cabeceira = cabInput.value
        marcarCambios()
      }))
      linhaOfi.appendChild(boton(t('plantillas.insertOfiPortada', 'Na portada de Word/ODT'), null, () => {
        const v = ofiPortadaArea.value.replace(/\s+$/, '') || porDefecto.portadaOfimatica.replace(/\s+$/, '')
        ofiPortadaArea.value = (v ? v + '\n' : '') + '@imaxe ' + nome + ' 30'
        ofi.portada = ofiPortadaArea.value
        marcarCambios()
      }))
      fila.appendChild(linhaOfi)
      imaxesWrap.appendChild(fila)
    }
  }

  // ---------- PDF: a maqueta LaTeX ----------
  // ---------- Deseño do PDF (sen LaTeX) ----------
  // O bloque créase aquí para que vaia o primeiro no lanzador; móntase máis
  // abaixo, cando xa existe o bloque Estilo (o LaTeX libre manda sobre el).
  const bloqueDeseno = bloque('🎨 ' + t('plantillas.bloque.deseno', 'Deseño do PDF'), 'deseno', 'pdf', t('plantillas.chip.deseno', 'Deseño'))
  bloqueDeseno.classList.add('plantilla-bloque--deseno')
  const corpoDeseno = el('div', 'deseno')
  bloqueDeseno.appendChild(corpoDeseno)

  const bloqueLatex = bloque(t('plantillas.bloque.latex', 'Maqueta do PDF (LaTeX)'), 'latex', 'pdf', t('plantillas.chip.latex', 'Maqueta'))
  const latexLabel = el('label', 'modal-label', t('plantillas.latex', 'LaTeX da plantilla'))
  const latexArea = el('textarea', 'plantilla-codigo plantilla-codigo--grande')
  latexArea.rows = 20
  latexArea.spellcheck = false
  latexArea.value = actual.latex
  latexLabel.appendChild(latexArea)
  bloqueLatex.appendChild(latexLabel)

  const axuda = el('p', 'plantilla-axuda')
  axuda.innerHTML = t('plantillas.latexHelp',
    'Escribe <code>{{CORPO}}</code> onde queiras que vaia o informe (índice, introdución, cronograma e desenvolvemento). Se non poñes <code>\\documentclass</code>, Lúa engade o seu preámbulo (abonda para cabeceira e pé; xa carga <code>fancyhdr</code>); se o poñes, mandas ti e Lúa só engade os paquetes e as definicións que precisa o informe. A primeira e a última páxina deséñanse abaixo e entran onde poñas <code>{{PORTADA}}</code> e <code>{{SINATURA}}</code>; sen eses marcadores Lúa pon a súa portada e a súa sinatura. Os datos fixos do centro ponos coma variables: <code>{{TELEFONO}}</code>, <code>{{CORREO}}</code>… Lúa enche soas, a partir do curso e do centro, <code>{{CENTRO}}</code>, <code>{{LOCALIDADE}}</code>, <code>{{CURSO}}</code>, <code>{{DEPARTAMENTO}}</code>, <code>{{XEFE}}</code> (ou <code>{{ORIENTADOR}}</code>), <code>{{CARGO}}</code>, <code>{{TITULO}}</code>, <code>{{DATA}}</code>, <code>{{DATA_LONGA}}</code>, <code>{{DATA_INICIO}}</code>, <code>{{DATA_FIN}}</code>, <code>{{ANO}}</code> e <code>{{FICHEIRO}}</code>; se unha variable co mesmo nome ten valor na plantilla, gaña ese valor. O logotipo do curso pónse con <code>\\logocentro[height=2cm]</code>.')
  bloqueLatex.appendChild(axuda)

  const latexBarra = el('div', 'plantillas-toolbar')
  latexBarra.appendChild(boton(t('plantillas.insertBody', 'Inserir {{CORPO}}'), null, () => {
    inserirNoCursor(latexArea, MARCADOR_CORPO)
  }))
  latexBarra.appendChild(boton(t('plantillas.insertCover', 'Inserir {{PORTADA}}'), null, () => {
    inserirNoCursor(latexArea, MARCADOR_PORTADA)
  }))
  latexBarra.appendChild(boton(t('plantillas.insertSignature', 'Inserir {{SINATURA}}'), null, () => {
    inserirNoCursor(latexArea, MARCADOR_SINATURA)
  }))
  bloqueLatex.appendChild(latexBarra)

  // ---------- Portada, sinatura e introdución: o deseño destas partes vai na plantilla ----------
  function campoParte(pai, { clave, etiqueta, axudaTexto, botonTexto, filas, codigo }) {
    const label = el('label', 'modal-label', etiqueta)
    const area = el('textarea', codigo ? 'plantilla-parte' : '')
    area.rows = filas
    area.spellcheck = !codigo
    area.value = actual[clave] || ''
    area.addEventListener('input', () => { actual[clave] = area.value })
    label.appendChild(area)
    pai.appendChild(label)
    pai.appendChild(el('p', 'plantilla-axuda', axudaTexto))
    const barra = el('div', 'plantillas-toolbar')
    barra.appendChild(boton(botonTexto, null, () => {
      if (!porDefecto[clave]) return
      area.value = porDefecto[clave]
      actual[clave] = area.value
      marcarCambios()
    }))
    pai.appendChild(barra)
    return area
  }
  const bloquePortada = bloque(t('plantillas.bloque.portada', 'Portada do PDF'), 'portada', 'pdf', t('plantillas.chip.portada', 'Portada'))
  const portadaArea = campoParte(bloquePortada, {
    clave: 'portada', codigo: true, filas: 12,
    etiqueta: t('plantillas.cover', 'Portada (LaTeX)'),
    axudaTexto: t('plantillas.coverHelp',
      'Deseño da primeira páxina. Usa as mesmas variables (centro, curso, xefe/a de departamento, cargo, data…) e o logotipo do curso con \\logocentro. Debe levar {{PORTADA}} no LaTeX de arriba; baleiro, Lúa usa a súa portada. Só se aplica ao PDF e ao .tex.'),
    botonTexto: t('plantillas.useDefaultCover', 'Usar a portada de Lúa'),
  })
  const bloqueSinatura = bloque(t('plantillas.bloque.sinatura', 'Sinatura do PDF'), 'sinatura', 'pdf', t('plantillas.chip.sinatura', 'Sinatura'))
  const sinaturaArea = campoParte(bloqueSinatura, {
    clave: 'sinatura', codigo: true, filas: 10,
    etiqueta: t('plantillas.signature', 'Sinatura (LaTeX)'),
    axudaTexto: t('plantillas.signatureHelp',
      'Deseño do peche do documento (data, quen asina e cargo). Vai ao final, sen folla propia; se a queres aparte, empeza o fragmento con \\clearpage. Debe levar {{SINATURA}} no LaTeX de arriba; baleiro, Lúa usa a súa sinatura. Só se aplica ao PDF e ao .tex.'),
    botonTexto: t('plantillas.useDefaultSignature', 'Usar a sinatura de Lúa'),
  })
  const bloqueIntro = bloque(t('plantillas.bloque.intro', 'Introdución'), 'intro', 'pdf', t('plantillas.chip.intro', 'Introdución'))
  const introArea = campoParte(bloqueIntro, {
    clave: 'introducion', codigo: false, filas: 8,
    etiqueta: t('plantillas.intro2', 'Texto da introdución'),
    axudaTexto: t('plantillas.introHelp',
      'Texto da sección «Introdución». Podes usar variables coma {{CENTRO}}, {{CURSO}} e {{DEPARTAMENTO}}; unha liña en branco separa parágrafos. Se o curso ten a súa propia introdución, esa gaña. Baleiro = o texto de Lúa.'),
    botonTexto: t('plantillas.useDefaultIntro', 'Usar o texto de Lúa'),
  })

  // ---------- Estilo: o deseño do documento, en LaTeX ----------
  const bloqueEstilo = bloque(t('plantillas.bloque.estilo', 'Estilo do documento (LaTeX libre)'), 'estilo', 'pdf', t('plantillas.chip.estilo', 'Estilo LaTeX'))
  const estiloArea = campoParte(bloqueEstilo, {
    clave: 'estilo', codigo: true, filas: 18,
    etiqueta: t('plantillas.estilo', 'Estilo (LaTeX)'),
    axudaTexto: t('plantillas.estiloHelp',
      'O deseño do documento vai aquí, non no código de Lúa: formato dos parágrafos e dos títulos, as etiquetas do cronograma (\\chip, \\mesbadge, ambiente trimestre), a cabeceira das actas (\\actacab, \\actasecc, \\actasinaturas...). Baleiro, Lúa usa o seu estilo; se escribes algo, substitúeo enteiro (parte do de Lúa co botón de abaixo). As cores das tarefas (tf-*, tt-*, tb-*, tc-*), gris-linea, gris-texto, \\fontchip e \\logocentro vén sempre de Lúa. Só se aplica ao PDF e ao .tex.'),
    botonTexto: t('plantillas.useDefaultStyle', 'Usar o estilo de Lúa'),
  })
  estiloArea.addEventListener('input', () => desenoUI?.pintar())

  // O deseño vive en actual.deseno; cada cambio refresca a vista previa (se
  // está activo «Actualizar ao aplicar»), sen esperar a Aplicar.
  let temporizadorDeseno = null
  // A parte do documento que máis cambiou: a vista previa salta á súa páxina.
  let parteCambiada = ''
  const desenoUI = montarBloqueDeseno(corpoDeseno, {
    t, api,
    nomePlantilla: () => nomeInput.value.trim(),
    ler: () => actual.deseno || {},
    cambiar: (d, parte) => {
      actual.deseno = d
      if (parte) parteCambiada = parte
      marcarCambios()
      clearTimeout(temporizadorDeseno)
      temporizadorDeseno = setTimeout(() => { if (autoVista.checked && !vista.hidden && formatoVista.value === 'pdf') previsualizar() }, 700)
    },
    estiloLibre: () => estiloArea.value,
    quitarEstiloLibre: () => {
      estiloArea.value = ''
      actual.estilo = ''
      marcarCambios()
    },
    imaxeLogo: async () => {
      const imaxes = actual.imaxes || []
      const nome = imaxes.find((n) => /logo/i.test(n) && /\.(png|jpe?g|gif)$/i.test(n)) || imaxes.find((n) => /\.(png|jpe?g|gif)$/i.test(n))
      if (!nome || !actual.id || !api.vistaImaxe) return null
      return api.vistaImaxe(actual.id, nome)
    },
    previsualizarCon: api.previsualizar ? (d) => api.previsualizar({ ...datosActuais(), deseno: d }, mostraVista.value, documento) : null,
    pedirIA: api.asistente ? (texto) => { asistente?.abrir(); asistente?.escribir?.(texto) } : null,
  })

  // ---------- Textos: todos os textos fixos do documento ----------
  const bloqueTextos = bloque(t('plantillas.bloque.textos', 'Textos do documento'), 'textos', 'datos', t('plantillas.chip.textos', 'Textos'))
  bloqueTextos.appendChild(el('p', 'plantilla-axuda', t('plantillas.textosHelp',
    'Lúa xera o contido; os textos fixos do documento (títulos das seccións, «Sen tarefas», os rótulos das actas...) cámbianse aquí, en PDF, Word, OpenDocument e Markdown. Baleiro = o texto de Lúa (o que se ve gris). Valen as variables {{CENTRO}}, {{CURSO}}, {{DEPARTAMENTO}}... e as que indica cada texto.')))
  const textosFiltro = el('input')
  textosFiltro.type = 'search'
  textosFiltro.placeholder = t('plantillas.textosFiltro', 'Buscar un texto…')
  bloqueTextos.appendChild(textosFiltro)
  const textosWrap = el('div', 'plantilla-textos')
  bloqueTextos.appendChild(textosWrap)
  const GRUPOS_TEXTOS = [
    ['comun', t('plantillas.textos.comun', 'Comúns')],
    ['cronograma', t('plantillas.textos.cronograma', 'Cronograma anual')],
    ['actas', t('plantillas.textos.actas', 'Actas do departamento')],
    ['formulario', t('plantillas.textos.formulario', 'Formularios propios')],
  ]
  function pintarTextos() {
    textosWrap.innerHTML = ''
    if (!porDefecto.textos.length) {
      textosWrap.appendChild(el('p', 'plantilla-axuda', t('plantillas.textosSen', 'Non se puido ler a lista de textos.')))
      return
    }
    for (const [grupo, titulo] of GRUPOS_TEXTOS) {
      const lista = porDefecto.textos.filter((x) => x.grupo === grupo)
      if (!lista.length) continue
      const det = el('details', 'plantilla-textos__grupo')
      det.open = grupo === 'comun'
      det.appendChild(el('summary', null, titulo + ' (' + lista.length + ')'))
      for (const x of lista) {
        const fila = el('label', 'plantilla-texto')
        fila.dataset.busca = (x.etiqueta + ' ' + x.clave + ' ' + x.defecto).toLowerCase()
        fila.appendChild(el('span', 'plantilla-texto__etq', x.etiqueta))
        const linha = el('span', 'plantilla-texto__linha')
        const input = el('input')
        input.type = 'text'
        input.placeholder = x.defecto
        input.value = textos[x.clave] || ''
        input.addEventListener('input', () => {
          if (input.value.trim()) textos[x.clave] = input.value
          else delete textos[x.clave]
        })
        const reiniciar = boton('↺', null, () => { input.value = ''; delete textos[x.clave]; marcarCambios() })
        reiniciar.title = t('plantillas.textoPorDefecto', 'Volver ao texto de Lúa')
        linha.append(input, reiniciar)
        fila.appendChild(linha)
        if (x.axuda) fila.appendChild(el('small', 'plantilla-texto__axuda', x.axuda))
        det.appendChild(fila)
      }
      textosWrap.appendChild(det)
    }
    filtrarTextos()
  }
  function filtrarTextos() {
    const q = textosFiltro.value.trim().toLowerCase()
    for (const fila of textosWrap.querySelectorAll('.plantilla-texto')) fila.hidden = !!q && !fila.dataset.busca.includes(q)
    for (const det of textosWrap.querySelectorAll('.plantilla-textos__grupo')) {
      if (q) det.open = [...det.querySelectorAll('.plantilla-texto')].some((f) => !f.hidden)
    }
  }
  textosFiltro.addEventListener('input', filtrarTextos)
  pintarTextos()

  // ---------- Markdown ----------
  const bloqueMd = bloque(t('plantillas.bloque.markdown', 'Markdown (.md)'), 'markdown', 'pdf', t('plantillas.chip.markdown', 'Markdown'))
  const mdLabel = el('label', 'modal-label', t('plantillas.markdown', 'Markdown equivalente (opcional: .md)'))
  const mdArea = el('textarea', 'plantilla-codigo')
  mdArea.rows = 6
  mdArea.spellcheck = false
  mdArea.value = actual.markdown
  mdLabel.appendChild(mdArea)
  bloqueMd.appendChild(mdLabel)
  bloqueMd.appendChild(el('p', 'plantilla-axuda', t('plantillas.markdownHelp',
    'Mesmo {{CORPO}} e mesmas variables, pero en Markdown. Se o deixas baleiro, o .md sae sen maqueta. As imaxes referéncianse aquí coma images/logo.png.')))

  // ---------- Word e OpenDocument: a estética destes formatos tamén é da plantilla ----------
  const bloqueTipo = bloque(t('plantillas.ofi.titulo', 'Word e OpenDocument (.docx / .odt)'), 'ofi', 'ofi', t('plantillas.chip.ofi', 'Letra e marxes'))
  bloqueTipo.appendChild(el('p', 'plantilla-axuda', t('plantillas.ofi.axuda',
    'Lúa xera a información do documento; o aspecto en Word e OpenDocument (tipo de letra, cores, portada, cabeceira e pé) é da plantilla. Todo o que deixes baleiro usa o deseño por defecto de Lúa. Podes usar as mesmas variables ({{CENTRO}}, {{CURSO}}…).')))
  const ofiGrade = el('div', 'plantilla-ofi-grade')

  const fonteInput = el('input')
  fonteInput.type = 'text'
  fonteInput.placeholder = 'Arial'
  fonteInput.setAttribute('list', 'plantillaFontes')
  fonteInput.addEventListener('input', () => { ofi.fonte = fonteInput.value })
  refrescadoresOfi.push(() => { fonteInput.value = ofi.fonte || '' })
  const fonteLabel = el('label', 'modal-label', t('plantillas.ofi.fonte', 'Tipo de letra'))
  fonteLabel.appendChild(fonteInput)
  const fontesLista = el('datalist')
  fontesLista.id = 'plantillaFontes'
  fonteLabel.appendChild(fontesLista)
  ofiGrade.appendChild(fonteLabel)
  if (api.fontes) {
    Promise.resolve(api.fontes()).then((lista) => {
      for (const f of (lista || []).slice(0, 400)) {
        const o = el('option')
        o.value = typeof f === 'string' ? f : (f && f.nome) || ''
        if (o.value) fontesLista.appendChild(o)
      }
    }).catch(() => {})
  }

  function campoNumero(clave, etiqueta, { min, max, step, placeholder }) {
    const input = el('input')
    input.type = 'number'
    input.min = String(min)
    input.max = String(max)
    input.step = String(step)
    input.placeholder = placeholder
    input.addEventListener('input', () => { ofi[clave] = parseFloat(input.value) || 0 })
    refrescadoresOfi.push(() => { input.value = ofi[clave] ? String(ofi[clave]) : '' })
    const label = el('label', 'modal-label', etiqueta)
    label.appendChild(input)
    ofiGrade.appendChild(label)
    return input
  }
  campoNumero('tamanoCorpo', t('plantillas.ofi.tamano', 'Tamaño do texto (pt)'), { min: 8, max: 16, step: 0.5, placeholder: '11' })
  campoNumero('interlinado', t('plantillas.ofi.interlinado', 'Interliñado (1 = sinxelo)'), { min: 1, max: 3, step: 0.05, placeholder: '1.15' })
  campoNumero('marxeSuperior', t('plantillas.ofi.marxeSuperior', 'Marxe superior (cm)'), { min: 0.5, max: 6, step: 0.1, placeholder: '2.5' })
  campoNumero('marxeInferior', t('plantillas.ofi.marxeInferior', 'Marxe inferior (cm)'), { min: 0.5, max: 6, step: 0.1, placeholder: '2.5' })
  campoNumero('marxeEsquerda', t('plantillas.ofi.marxeEsquerda', 'Marxe esquerda (cm)'), { min: 0.5, max: 6, step: 0.1, placeholder: '2.5' })
  campoNumero('marxeDereita', t('plantillas.ofi.marxeDereita', 'Marxe dereita (cm)'), { min: 0.5, max: 6, step: 0.1, placeholder: '2.5' })
  bloqueTipo.appendChild(ofiGrade)
  bloqueTipo.appendChild(el('p', 'plantilla-axuda', t('plantillas.ofi.marxesAxuda',
    'Deixa baleiro o interliñado e as marxes para usar os de Lúa (1,15 e 2,5 cm en Word; 2 cm en OpenDocument).')))

  const bloqueCores = bloque(t('plantillas.bloque.cores', 'Cores de Word/OpenDocument'), 'cores', 'ofi', t('plantillas.chip.cores', 'Cores'))
  const CORES_OFI = [
    ['corPrincipal', t('plantillas.ofi.corPrincipal', 'Cor dos títulos e da portada'), '1F3864'],
    ['corSecundaria', t('plantillas.ofi.corSecundaria', 'Cor dos subtítulos'), '2E5496'],
    ['corTexto', t('plantillas.ofi.corTexto', 'Cor do texto'), '212529'],
    ['corApagada', t('plantillas.ofi.corApagada', 'Cor dos textos secundarios'), '595959'],
    ['corLinha', t('plantillas.ofi.corLinha', 'Cor dos filetes'), '8EAADB'],
  ]
  const coresGrade = el('div', 'plantilla-ofi-grade')
  for (const [clave, etiqueta, defecto] of CORES_OFI) {
    const linha = el('label', 'modal-label plantilla-ofi-cor', etiqueta)
    const cor = el('input')
    cor.type = 'color'
    cor.addEventListener('input', () => { ofi[clave] = cor.value.replace('#', '').toUpperCase() })
    refrescadoresOfi.push(() => { cor.value = '#' + (ofi[clave] || defecto) })
    const reiniciar = boton('↺', null, () => { ofi[clave] = ''; cor.value = '#' + defecto; marcarCambios() })
    reiniciar.title = t('plantillas.ofi.porDefecto', 'Volver á cor por defecto')
    linha.appendChild(cor)
    linha.appendChild(reiniciar)
    coresGrade.appendChild(linha)
  }
  bloqueCores.appendChild(coresGrade)

  const bloqueCabPe = bloque(t('plantillas.bloque.cabpe', 'Cabeceira e pé de Word/OpenDocument'), 'cabpe', 'ofi', t('plantillas.chip.cabpe', 'Cabeceira e pé'))
  const cabInput = el('textarea')
  cabInput.rows = 4
  cabInput.placeholder = '{{CENTRO}} · {{DEPARTAMENTO}}'
  cabInput.addEventListener('input', () => { ofi.cabeceira = cabInput.value })
  refrescadoresOfi.push(() => { cabInput.value = ofi.cabeceira || '' })
  const cabLabel = el('label', 'modal-label', t('plantillas.ofi.cabeceira', 'Cabeceira (en todas as páxinas menos a portada)'))
  cabLabel.appendChild(cabInput)
  bloqueCabPe.appendChild(cabLabel)
  const peInput = el('textarea')
  peInput.rows = 3
  peInput.placeholder = 'Curso {{CURSO}}'
  peInput.addEventListener('input', () => { ofi.pe = peInput.value })
  refrescadoresOfi.push(() => { peInput.value = ofi.pe || '' })
  const peLabel = el('label', 'modal-label', t('plantillas.ofi.pe', 'Texto do pé (á beira do número de páxina)'))
  peLabel.appendChild(peInput)
  bloqueCabPe.appendChild(peLabel)
  bloqueCabPe.appendChild(el('p', 'plantilla-axuda', t('plantillas.ofi.caixaAxuda',
    'Na cabeceira e no pé pódense pór varios logotipos en liña: unha liña por elemento, coma «@imaxe logo1.png 10», «@imaxe logo2.pdf 10» (alto en mm; PNG, JPEG ou PDF do cartafol de imaxes desta plantilla), texto con variables e «@esquerda», «@centro» ou «@dereita».')))

  const bloqueOfiPortada = bloque(t('plantillas.bloque.ofiportada', 'Portada de Word/OpenDocument'), 'ofiportada', 'ofi', t('plantillas.chip.ofiportada', 'Portada'))
  const ofiPortadaLabel = el('label', 'modal-label', t('plantillas.ofi.portada', 'Portada de Word/OpenDocument'))
  const ofiPortadaArea = el('textarea', 'plantilla-parte')
  ofiPortadaArea.rows = 10
  ofiPortadaArea.spellcheck = false
  ofiPortadaArea.placeholder = porDefecto.portadaOfimatica
  ofiPortadaArea.addEventListener('input', () => { ofi.portada = ofiPortadaArea.value })
  refrescadoresOfi.push(() => { ofiPortadaArea.value = ofi.portada || '' })
  ofiPortadaLabel.appendChild(ofiPortadaArea)
  bloqueOfiPortada.appendChild(ofiPortadaLabel)
  bloqueOfiPortada.appendChild(el('p', 'plantilla-axuda', t('plantillas.ofi.portadaAxuda',
    'Unha liña por elemento: @logo 35 (logotipo do curso, alto en mm), @imaxe nome.png 30 (unha imaxe desta plantilla), @espazo 12, # Título, ## Subtítulo, --- (liña), ### Liña destacada e texto normal. Baleiro = a portada por defecto.')))
  const ofiBarra = el('div', 'plantillas-toolbar')
  ofiBarra.appendChild(boton(t('plantillas.ofi.usarPortada', 'Usar a portada de Lúa'), null, () => {
    if (!porDefecto.portadaOfimatica) return
    ofiPortadaArea.value = porDefecto.portadaOfimatica
    ofi.portada = ofiPortadaArea.value
    marcarCambios()
  }))
  bloqueOfiPortada.appendChild(ofiBarra)
  pintarOfimatica()
  pintarImaxes()

  // ---------- Vista previa ----------
  const previaWrap = el('div', 'plantilla-previa')
  const formatoVista = el('select', 'plantilla-vista__formato')
  formatoVista.title = t('plantillas.vista.formato', 'Formato que se previsualiza')
  for (const [v, txt] of [['pdf', 'PDF (LaTeX)'], ['docx', 'Word (.docx)'], ['odt', 'OpenDocument (.odt)']]) {
    const o = el('option', null, txt)
    o.value = v
    formatoVista.appendChild(o)
  }
  // A vista previa faise SEMPRE co documento que se está editando. Só se non hai
  // ningún (o editor aberto sen ferramenta activa) usa o cronograma de exemplo.
  const mostraVista = { get value() { return documento ? 'actual' : 'cronograma' } }
  const autoVista = el('input')
  autoVista.type = 'checkbox'
  autoVista.checked = lerAutoVista()
  autoVista.addEventListener('change', () => gardarAutoVista(autoVista.checked))
  const autoLabel = el('label', 'plantilla-vista__auto')
  autoLabel.append(autoVista, el('span', null, t('plantillas.vista.auto', 'Actualizar ao aplicar')))
  const avisoVista = el('p', 'plantilla-vista__aviso', t('plantillas.vista.vella', 'Hai cambios que aínda non se ven aquí.'))
  avisoVista.hidden = true
  const refrescarBtn = boton('↻ ' + t('plantillas.vista.actualizar', 'Actualizar'), null, () => previsualizar())
  // Mentres compila: progreso e Cancelar (mata a compilación de LaTeX).
  const progresoVista = el('span', 'plantilla-vista__progreso', t('plantillas.previewing', 'Compilando a vista previa…'))
  progresoVista.hidden = true
  progresoVista.setAttribute('role', 'status')
  const cancelarVistaBtn = boton('✕ ' + t('plantillas.vista.cancelar', 'Cancelar'), 'plantilla-vista__cancelar', () => {
    vistaPendente = false
    api.cancelarVista?.()
  })
  cancelarVistaBtn.hidden = true
  const vistaCab = el('div', 'plantilla-vista__cab')
  vistaCab.append(el('strong', null, t('plantillas.vista.titulo', 'Vista previa')), formatoVista, refrescarBtn, autoLabel, progresoVista, cancelarVistaBtn)
  const vista = el('aside', 'plantilla-vista')
  vista.append(vistaCab, avisoVista, previaWrap)
  formatoVista.addEventListener('change', () => { if (autoVista.checked) previsualizar() })

  // ---------- Pé: estado e accións ----------
  // Clase propia ademais de .ai-row__status (que xa usa o xerador de IA deste
  // mesmo editor): son dous avisos distintos e teñen que poder distinguirse,
  // tanto ao estilalos coma nas probas.
  const estado = el('p', 'ai-row__status plantilla-estado')
  const previaBtn = boton('👁 ' + t('plantillas.preview', 'Vista previa'), null, () => previsualizar())
  const cancelarBtn = boton(t('plantillas.cancel', 'Cancelar'), null, () => cancelar())
  const aplicarBtn = boton(t('plantillas.apply', 'Aplicar'), 'tarefa-aplicar', () => gardar({ pechar: false }))
  aplicarBtn.title = t('plantillas.apply.title', 'Gardar a plantilla sen pechar este editor e actualizar a vista previa (Ctrl+S)')
  const gardarBtn = boton(t('plantillas.save', 'Gardar'), 'primary', () => gardar({ pechar: true }))
  gardarBtn.title = t('plantillas.save.title', 'Gardar a plantilla e pechar o editor (Ctrl+Intro)')
  const accions = el('div', 'modal-actions tarefa-pe plantilla-pe')
  accions.append(estado, previaBtn, cancelarBtn, aplicarBtn, gardarBtn)

  // ---------- Cabeceira da pantalla ----------
  const cabTitulo = el('h2', 'tarefa-cab__titulo')
  const estadoCab = el('span', 'tarefa-cab__estado')
  estadoCab.setAttribute('aria-live', 'polite')
  const pintarTitulo = () => {
    cabTitulo.textContent = actual.id
      ? '📐 ' + t('plantillas.editTitle', 'Editar plantilla')
      : '📐 ' + t('plantillas.newTitle', 'Plantilla nova')
  }
  pintarTitulo()
  const botonX = el('button', 'tarefa-x')
  botonX.type = 'button'
  botonX.innerHTML = ICONA_X
  botonX.title = t('informes.tarefa.pechar', 'Pechar')
  botonX.setAttribute('aria-label', t('informes.tarefa.pechar', 'Pechar'))
  botonX.addEventListener('click', () => cancelar())
  const btnVista = boton('👁 ' + t('plantillas.vista.mostrar', 'Vista previa'), 'plantilla-cab-btn', () => {
    vista.hidden = !vista.hidden
    btnVista.setAttribute('aria-pressed', String(!vista.hidden))
  })
  btnVista.setAttribute('aria-pressed', 'true')
  btnVista.title = t('plantillas.vista.alternar', 'Amosar ou agochar a vista previa')
  const btnDesfacerIA = boton(t('plantillas.desfacer.ia', '↶ Desfacer cambio da IA'), 'tarefa-desfacer-ia', () => {
    const anterior = pilaIA.pop()
    if (anterior) fixarBorrador(anterior, { substituirOfi: true })
    btnDesfacerIA.hidden = pilaIA.length === 0
  })
  btnDesfacerIA.hidden = true
  btnDesfacerIA.title = t('plantillas.desfacer.ia.title', 'Volver ao estado anterior ao último cambio da IA')

  const logoAsistente = el('span', 'tarefa-asistente__logo')
  const asistenteOpc = api.asistente || null
  const btnAsistente = asistenteOpc ? boton('', 'tarefa-asistente', () => asistente?.alternar()) : null
  if (btnAsistente) {
    btnAsistente.title = t('plantillas.asistente.title', 'Asistente de IA da plantilla: cambia a portada, a letra, o interliñado, as marxes, a cabeceira, o pé...')
    btnAsistente.append(logoAsistente, el('span', null, t('plantillas.asistente.label', 'Asistente de deseño')))
  }
  const cabeceira = el('header', 'tarefa-cab')
  cabeceira.append(cabTitulo, estadoCab, btnDesfacerIA, btnVista, ...(btnAsistente ? [btnAsistente] : []), botonX)

  // ---------- Columnas e lanzador de bloques ----------
  const columna = (titulo, ...blocos) => {
    const c = el('div', 'plantilla-columna')
    c.appendChild(el('h2', 'plantilla-columna__tit', titulo))
    c.append(...blocos)
    return c
  }
  const colDatos = columna(t('plantillas.col.datos', 'Datos e recursos'), bloqueIdentidade, bloqueTextos, bloqueVariables, bloqueImaxes)
  // O deseño vai nunha columna propia, a primeira e a máis ancha: é o que se
  // usa máis e o que non precisa LaTeX.
  const colDeseno = columna(t('plantillas.col.deseno', 'Deseño'), bloqueDeseno)
  colDeseno.classList.add('plantilla-columna--deseno')
  const colPdf = columna(t('plantillas.col.pdf', 'PDF e LaTeX'), bloqueLatex, bloquePortada, bloqueSinatura, bloqueIntro, bloqueEstilo, bloqueMd)
  const colOfi = columna(t('plantillas.col.ofi', 'Word e OpenDocument'), bloqueTipo, bloqueCores, bloqueCabPe, bloqueOfiPortada)
  const columnas = el('div', 'plantilla-columnas')
  columnas.append(colDeseno, colDatos, colPdf, colOfi)

  // O lanzador: un botón por bloque, agrupados como as columnas. Premer un
  // botón despraga (ou pecha) o seu bloque; unha columna sen bloques á vista
  // agóchase.
  const lanzador = el('nav', 'plantilla-lanzador')
  lanzador.setAttribute('aria-label', t('plantillas.lanzador', 'Bloques da plantilla'))
  const chips = new Map()
  for (const [grupo, titulo] of [['datos', t('plantillas.col.datos', 'Datos e recursos')], ['pdf', t('plantillas.col.pdf', 'PDF e LaTeX')], ['ofi', t('plantillas.col.ofi', 'Word e OpenDocument')]]) {
    const g = el('div', 'plantilla-lanzador__grupo')
    g.appendChild(el('span', 'plantilla-lanzador__tit', titulo))
    for (const r of registroBlocos.filter((x) => x.grupo === grupo)) {
      const chip = boton(r.etq, 'plantilla-chip', () => alternarBloque(r.id))
      chip.dataset.bloque = r.id
      chip.title = t('plantillas.chip.title', 'Amosar ou agochar: {nome}', { nome: r.etq })
      chips.set(r.id, chip)
      g.appendChild(chip)
    }
    lanzador.appendChild(g)
  }
  function aplicarBlocos() {
    for (const r of registroBlocos) {
      const aberto = blocosAbertos.has(r.id)
      r.nodo.hidden = !aberto
      const chip = chips.get(r.id)
      chip.setAttribute('aria-pressed', String(aberto))
      chip.setAttribute('aria-expanded', String(aberto))
    }
    for (const c of [colDeseno, colDatos, colPdf, colOfi]) c.hidden = !c.querySelector('.plantilla-bloque:not([hidden])')
    columnas.classList.toggle('plantilla-columnas--so-deseno', !colDeseno.hidden && [colDatos, colPdf, colOfi].every((c) => c.hidden))
  }
  alternarBloque = (id, abrir) => {
    const abrirAgora = abrir === undefined ? !blocosAbertos.has(id) : abrir
    if (abrirAgora) blocosAbertos.add(id)
    else blocosAbertos.delete(id)
    gardarBlocosAbertos(blocosAbertos)
    aplicarBlocos()
    if (abrirAgora) {
      const nodo = registroBlocos.find((r) => r.id === id)?.nodo
      nodo?.scrollIntoView?.({ block: 'nearest' })
    }
  }
  aplicarBlocos()

  const medio = el('div', 'tarefa-medio plantilla-medio')
  const centro = el('div', 'plantilla-centro')
  centro.append(lanzador, columnas)
  medio.append(centro, vista)
  container.replaceChildren(cabeceira, medio, accions)

  // ---------- Borrador ----------
  function datosActuais() {
    return {
      ...actual,
      nome: nomeInput.value.trim(),
      descricion: descInput.value.trim(),
      latex: latexArea.value,
      markdown: mdArea.value,
      motor: motorSelect.value,
      variables: actual.variables,
      portada: portadaArea.value,
      sinatura: sinaturaArea.value,
      introducion: introArea.value,
      estilo: estiloArea.value,
      textos: { ...textos },
      ofimatica: { ...ofi },
      deseno: actual.deseno && Object.keys(actual.deseno).length ? actual.deseno : null,
    }
  }

  const instantanea = () => {
    const d = datosActuais()
    return JSON.stringify([d.nome, d.descricion, d.latex, d.markdown, d.motor, d.portada, d.sinatura, d.introducion, d.estilo,
      (d.variables || []).map((v) => [v.nome, v.etiqueta, v.valor]), ofiLimpa(d.ofimatica), ofiLimpa(d.textos), d.deseno || {}])
  }
  let inicial = instantanea()
  let sucioAvisado = false
  const cambiado = () => instantanea() !== inicial
  let vistaDe = ''
  const pintarAvisoVista = () => { avisoVista.hidden = !vistaDe || vistaDe === instantanea() }
  function marcarCambios() {
    if (pechado) return
    const sucio = cambiado()
    const msgSucio = t('plantillas.sen.aplicar', '● Cambios sen aplicar')
    // Só se toca a mensaxe da cabeceira para poñer/quitar o aviso de cambios:
    // un «✓ Gardado» ou un aviso do asistente non se borran por un clic calquera.
    if (sucio) estadoCab.textContent = msgSucio
    else if (estadoCab.textContent === msgSucio) estadoCab.textContent = ''
    if (sucio !== sucioAvisado) { sucioAvisado = sucio; aoCambiarSucio(sucio) }
    pintarAvisoVista()
  }
  container.addEventListener('input', marcarCambios)
  container.addEventListener('change', marcarCambios)
  container.addEventListener('click', () => setTimeout(marcarCambios, 0))

  // ---------- Vista previa: compilar o borrador ----------
  // Inspector: onde está cada parte do deseño nas páxinas da última vista previa.
  let marcasVista = []
  let vistaPendente = false
  async function previsualizar() {
    if (pechado) return
    // Se xa se está compilando, vólvese compilar ao rematar (o deseño cambiou).
    if (compilando) { vistaPendente = true; return }
    compilando = true
    previaBtn.disabled = true
    refrescarBtn.disabled = true
    progresoVista.hidden = false
    cancelarVistaBtn.hidden = !api.cancelarVista
    estado.textContent = t('plantillas.previewing', 'Compilando a vista previa…')
    previaWrap.innerHTML = ''
    const formato = formatoVista.value
    const instante = instantanea()
    try {
      const datos = datosActuais()
      let res
      if (formato === 'pdf') res = await api.previsualizar(datos, mostraVista.value, documento)
      else if (api.previsualizarOfi) res = await api.previsualizarOfi(datos, formato, mostraVista.value, documento)
      else throw new Error(t('plantillas.vista.sen.ofi', 'A vista previa de Word/OpenDocument non está dispoñible.'))
      estado.textContent = (res.warnings || []).join(' · ')
      marcasVista = formato === 'pdf' ? (res.marcas || []) : []
      previaWrap.classList.toggle('plantilla-previa--inspector', marcasVista.length > 0)
      ;(res.pageImages || []).forEach((img, i) => {
        // Cada páxina vai nunha folla co resalte do inspector por riba.
        const folla = el('div', 'plantilla-previa__folla')
        const im = el('img', 'plantilla-previa__paxina')
        im.src = img
        im.dataset.paxina = String(i + 1)
        if (marcasVista.length) im.title = t('deseno.inspector', 'Preme nunha parte para cambiar o seu deseño')
        const resalte = el('div', 'plantilla-previa__resalte')
        resalte.hidden = true
        resalte.appendChild(el('span', 'plantilla-previa__resalte-nome'))
        folla.append(im, resalte)
        previaWrap.appendChild(folla)
      })
      saltarAParte()
      if (!(res.pageImages || []).length) {
        previaWrap.appendChild(el('p', 'plantilla-axuda', t('plantillas.previewNoPages',
          'Compilou, pero non se puido debuxar a vista previa (falta pdftoppm).')))
      }
      vistaDe = instante
      pintarAvisoVista()
    } catch (err) {
      const msg = String(err && err.message ? err.message : err)
      estado.textContent = /cancelada/.test(msg) ? t('plantillas.vista.cancelada', 'Vista previa cancelada.') : msg
    } finally {
      compilando = false
      previaBtn.disabled = false
      refrescarBtn.disabled = false
      progresoVista.hidden = true
      cancelarVistaBtn.hidden = true
      if (vistaPendente) { vistaPendente = false; previsualizar() }
    }
  }

  // A vista previa amosa a páxina onde está a parte que se acaba de cambiar.
  function saltarAParte() {
    if (!parteCambiada) return
    const parte = parteCambiada
    parteCambiada = ''
    const pax = parte === 'portada' ? 1 : marcasVista.find((m) => m.elemento === parte)?.paxina
    const folla = pax && previaWrap.querySelector(`.plantilla-previa__paxina[data-paxina="${pax}"]`)?.parentElement
    if (folla && previaWrap.scrollTo) previaWrap.scrollTo({ top: folla.offsetTop - previaWrap.offsetTop - 8, behavior: 'smooth' })
  }

  // Inspector: a marca (parte do documento) que hai baixo o punteiro.
  const marcaBaixo = (e) => {
    const im = e.target.closest?.('.plantilla-previa__paxina')
    if (!im || !marcasVista.length) return null
    const r = im.getBoundingClientRect()
    const pax = Number(im.dataset.paxina)
    return { im, marca: marcaEnPunto(marcasVista, pax, (e.clientX - r.left) / r.width, (e.clientY - r.top) / r.height) }
  }
  // Ao pasar o rato resáltase a parte e dise o seu nome: así se ve que se pode premer.
  previaWrap.addEventListener('mousemove', (e) => {
    for (const r of previaWrap.querySelectorAll('.plantilla-previa__resalte')) r.hidden = true
    const b = marcaBaixo(e)
    if (!b || !b.marca) return
    const res = b.im.parentElement.querySelector('.plantilla-previa__resalte')
    const c = caixaMarca(marcasVista, b.marca)
    Object.assign(res.style, { left: c.x * 100 + '%', top: c.y * 100 + '%', width: c.w * 100 + '%', height: c.h * 100 + '%' })
    res.firstChild.textContent = t(`deseno.el.${b.marca.elemento}`, b.marca.elemento)
    res.hidden = false
  })
  previaWrap.addEventListener('mouseleave', () => {
    for (const r of previaWrap.querySelectorAll('.plantilla-previa__resalte')) r.hidden = true
  })
  // Premer nunha páxina da vista previa abre a parte do deseño que hai alí.
  previaWrap.addEventListener('click', (e) => {
    const b = marcaBaixo(e)
    if (!b) return
    alternarBloque('deseno', true)
    desenoUI.abrirElemento(b.marca?.elemento || 'texto')
  })

  // ---------- Gardar / Aplicar ----------
  async function gardar({ pechar: pecharAoRematar }) {
    if (ocupado || pechado) return false
    const datos = datosActuais()
    if (!datos.nome) {
      alternarBloque('identidade', true)
      nomeInput.focus()
      estado.textContent = t('plantillas.needName', 'Dálle un nome á plantilla.')
      return false
    }
    if (!datos.latex.includes(MARCADOR_CORPO)) {
      estado.textContent = t('plantillas.needBody', 'Falta {{CORPO}}: sen el Lúa non sabe onde meter o documento.')
      return false
    }
    ocupado = true
    aplicarBtn.disabled = gardarBtn.disabled = true
    try {
      const gardada = await api.gardar(datos)
      actual = { ...gardada, variables: gardada.variables || [], imaxes: gardada.imaxes || [] }
      // Os campos capturan `ofi`: mantense o obxecto e cámbiase o seu contido.
      for (const k of Object.keys(ofi)) delete ofi[k]
      Object.assign(ofi, gardada.ofimatica || {})
      actual.ofimatica = ofi
      for (const k of Object.keys(textos)) delete textos[k]
      Object.assign(textos, gardada.textos || {})
      actual.textos = textos
      actual.deseno = gardada.deseno || {}
      desenoUI.pintar()
      pintarTextos()
      pintarVariables()
      pintarImaxes()
      pintarOfimatica()
      pintarTitulo()
      inicial = instantanea()
      if (sucioAvisado) { sucioAvisado = false; aoCambiarSucio(false) }
      estadoCab.textContent = t('plantillas.aplicado', '✓ Gardado')
      estado.textContent = t('plantillas.saved', 'Plantilla gardada.')
      // Avisa á xanela principal (biblioteca, selector, informe) de cada gardado.
      // Sen esperar: volver xerar o informe tarda e o editor non debe quedar parado.
      if (api.aplicada) Promise.resolve().then(() => api.aplicada(actual)).catch(() => { /* a xanela principal non o precisa */ })
      if (pecharAoRematar) {
        pechar()
        if (aoGardada) await aoGardada(actual)
      } else if (autoVista.checked && !vista.hidden) {
        previsualizar()
      }
      return true
    } catch (err) {
      estado.textContent = String(err && err.message ? err.message : err)
      return false
    } finally {
      ocupado = false
      aplicarBtn.disabled = gardarBtn.disabled = false
    }
  }

  function pechar() {
    pechado = true
    asistente?.destruir()
    document.removeEventListener('keydown', onKeydown, true)
    container.classList.remove('plantilla-pantalla')
    aoPechar()
  }

  async function cancelar() {
    if (ocupado) return
    if (cambiado()) {
      const ok = await confirmar(t('plantillas.descartar', 'Hai cambios sen gardar nesta plantilla. Descartalos?'),
        { aceptar: t('informes.descartar', 'Descartar'), cancelar: t('informes.seguir', 'Seguir editando') })
      if (!ok) return
    }
    pechar()
  }

  // Esc en captura para gañarlle ao manexador global de main.js; Ctrl+Intro
  // garda e pecha, Ctrl+S aplica.
  function onKeydown(e) {
    // Escribindo no asistente, as teclas son del (Esc pecha só o seu panel).
    if (e.target && e.target.closest && e.target.closest('.asistente')) return
    if (document.querySelector('.overlay--dialog:not(.tarefa-overlay)')) return // un diálogo por riba manda
    if (e.key === 'Escape') { e.preventDefault(); e.stopPropagation(); cancelar() }
    else if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) { e.preventDefault(); e.stopPropagation(); gardar({ pechar: true }) }
    else if ((e.key === 's' || e.key === 'S') && (e.ctrlKey || e.metaKey)) { e.preventDefault(); e.stopPropagation(); gardar({ pechar: false }) }
  }
  document.addEventListener('keydown', onKeydown, true)

  // ---------- Asistente de IA ----------
  // Traballa sobre o BORRADOR: cada cambio seu apílase para poder desfacelo.
  const pilaIA = []
  const instantaneaBorrador = () => {
    const d = datosActuais()
    return JSON.parse(JSON.stringify({
      nome: d.nome, descricion: d.descricion, latex: d.latex, markdown: d.markdown, motor: d.motor, portada: d.portada,
      sinatura: d.sinatura, introducion: d.introducion, estilo: d.estilo, textos: d.textos, variables: d.variables, ofimatica: d.ofimatica,
      deseno: d.deseno || {},
    }))
  }
  function fixarBorrador(c, { substituirOfi = false } = {}) {
    if (c.nome !== undefined) nomeInput.value = c.nome
    if (c.descricion !== undefined) descInput.value = c.descricion
    if (c.latex !== undefined) latexArea.value = c.latex
    if (c.markdown !== undefined) mdArea.value = c.markdown
    if (c.motor !== undefined) motorSelect.value = c.motor
    for (const [clave, area] of [['portada', portadaArea], ['sinatura', sinaturaArea], ['introducion', introArea], ['estilo', estiloArea]]) {
      if (c[clave] !== undefined) { area.value = c[clave]; actual[clave] = c[clave] }
    }
    if (c.textos !== undefined) {
      for (const k of Object.keys(textos)) delete textos[k]
      for (const [k, v] of Object.entries(c.textos || {})) if (String(v).trim()) textos[k] = String(v)
      pintarTextos()
    }
    if (c.variables !== undefined) {
      actual.variables = c.variables.map((v) => ({ nome: v.nome, etiqueta: v.etiqueta || '', valor: v.valor || '' }))
      pintarVariables()
    }
    if (c.ofimatica !== undefined) {
      if (substituirOfi) for (const k of Object.keys(ofi)) delete ofi[k]
      Object.assign(ofi, c.ofimatica)
      pintarOfimatica()
    }
    if (c.deseno !== undefined) {
      actual.deseno = c.deseno || {}
      desenoUI.pintar()
    }
    if (c.estilo !== undefined) desenoUI.pintar()
    marcarCambios()
  }

  // garantirGardada: as ferramentas que tocan o cartafol de imaxes precisan unha
  // plantilla propia xa gardada (unha nova, ou de exemplo, gárdase antes).
  async function garantirGardada() {
    if (actual.id && !actual.deFabrica) return true
    return gardar({ pechar: false })
  }
  async function recargarImaxes() {
    if (!actual.id || !api.listar) return
    const est = await api.listar()
    const p = (est.plantillas || []).find((x) => x.id === actual.id)
    if (p) { actual.imaxes = p.imaxes || []; pintarImaxes() }
  }

  const editorParaIA = {
    ler() {
      const d = datosActuais()
      const saida = {
        id: actual.id || null, gardada: !!actual.id, deFabrica: !!actual.deFabrica,
        nome: d.nome, descricion: d.descricion, motor: d.motor, latex: d.latex, portada: d.portada, sinatura: d.sinatura,
        introducion: d.introducion, markdown: d.markdown, variables: d.variables,
        estilo: d.estilo, textos: ofiLimpa(d.textos),
        imaxes: [...actual.imaxes], ofimatica: ofiLimpa(d.ofimatica), cambiosSenAplicar: cambiado(),
        // O deseño do PDF coma datos: é o que hai que cambiar para letra, cores,
        // tamaños e espazos (o campo «deseno» de plantilla_editor_cambiar).
        deseno: d.deseno || {},
        desenoLibre: !!d.estilo.trim(),
      }
      // Baleiro = o de Lua: dáse a base para partir dela e as claves dos textos.
      if (!d.estilo.trim() && porDefecto.estilo) saida.estiloPorDefecto = porDefecto.estilo
      saida.textosDispoñibles = porDefecto.textos.map((x) => ({ clave: x.clave, grupo: x.grupo, defecto: x.defecto }))
      // Baleiro = o de Lua: dáse a base para partir dela se a IA quere retocala.
      if (!d.portada.trim() && porDefecto.portada) saida.portadaPorDefecto = porDefecto.portada
      if (!d.sinatura.trim() && porDefecto.sinatura) saida.sinaturaPorDefecto = porDefecto.sinatura
      if (!ofi.portada && porDefecto.portadaOfimatica) saida.ofimaticaPortadaPorDefecto = porDefecto.portadaOfimatica
      return saida
    },
    borrador() {
      const d = datosActuais()
      return {
        id: actual.id || '', nome: d.nome, descricion: d.descricion, latex: d.latex, markdown: d.markdown, motor: d.motor,
        portada: d.portada, sinatura: d.sinatura, introducion: d.introducion, estilo: d.estilo, textos: { ...textos },
        variables: d.variables, ofimatica: { ...ofi }, deseno: d.deseno,
      }
    },
    id: () => actual.id,
    cambiar(c) {
      if (pechado) throw new Error('o editor desta plantilla xa está pechado')
      pilaIA.push(instantaneaBorrador())
      btnDesfacerIA.hidden = false
      fixarBorrador(c)
    },
    garantirGardada,
    recargarImaxes,
  }
  const resumoParaIA = () => {
    const d = datosActuais()
    return [
      `- Plantilla «${d.nome || '(sen nome)'}» ${actual.id ? `(gardada, id ${actual.id})` : '(nova, aínda sen gardar)'}${actual.deFabrica ? ', de exemplo: ao gardala créase unha copia propia' : ''}; compilador: ${d.motor || 'o de Opcións'}.`,
      `- ${d.variables.length} variable(s) propias; imaxes no cartafol: ${actual.imaxes.length ? actual.imaxes.join(', ') : 'ningunha'}.`,
      `- ${cambiado() ? 'Hai cambios sen aplicar no editor.' : 'Sen cambios pendentes.'} Deseño de Word/OpenDocument: ${Object.keys(ofiLimpa(ofi)).length ? Object.keys(ofiLimpa(ofi)).join(', ') : 'o por defecto'}.`,
    ].join('\n')
  }

  let asistente = null
  if (asistenteOpc && btnAsistente) {
    const ferr = crearFerramentasPlantilla({
      editor: editorParaIA,
      api: { IAChamarFerramenta: asistenteOpc.api.IAChamarFerramenta },
      ui: {
        avisar: (m) => { estadoCab.textContent = m },
        preguntar: (p, o) => asistente.preguntar(p, o),
      },
    })
    asistente = crearPanelAsistente({
      t,
      api: {
        ...asistenteOpc.api, // inclúe os adxuntos (clip, arrastrar, pegar)
        IAFerramentas: () => asistenteOpc.api.IAFerramentasDoAmbito('plantilla'),
        IATurnoAsistente: (req) => asistenteOpc.api.IATurnoAsistente({ ...req, ambito: 'plantilla', idioma: asistenteOpc.idioma?.() || '' }),
      },
      documentoBoveda: 'plantilla',
      executar: ferr.executar,
      estadoPrograma: resumoParaIA,
      tarefas: TAREFAS_PLANTILLA,
      claveTarefas: 'lua.asistente.tarefas.plantilla',
      montarEn: medio,
      textos: {
        titulo: t('asistente.tp.titulo', 'Asistente de deseño'),
        benvida: t('asistente.tp.benvida', 'Traballo só sobre esta plantilla: podo crela desde unha descrición, unha URL ou un documento modelo, e cambio a portada, a letra, o interliñado, as marxes, a cabeceira, o pé, as cores e os logotipos, tanto no PDF coma en Word/OpenDocument. Os cambios entran no borrador e podes velos na vista previa; ti decides se os aplicas.'),
        benvidaNota: t('asistente.tp.nota', 'Xunta cun clip un logotipo ou un documento modelo para que o copie. Podes desfacer o último cambio co botón «Desfacer cambio da IA».'),
        caixaPh: t('asistente.tp.ph', 'Pídeme o que queiras sobre a plantilla: «letra Georgia 12», «marxes de 3 cm», «pon o logotipo á esquerda da cabeceira»… Se premes unha ficha, isto engádese como instrución adicional.'),
      },
      aoCambiar: ({ ocupado: ocup, preset }) => {
        logoAsistente.innerHTML = logoProvedor(preset)
        btnAsistente.classList.toggle('tarefa-asistente--ocupado', ocup)
      },
    })
  }

  // "Crear con IA" (biblioteca) abre o editor co asistente xa á vista: describir a
  // plantilla, dar unha URL ou xuntar un documento modelo faise nel.
  if (abrirIA && asistente) asistente.abrir()
  else requestAnimationFrame(() => {
    // Unha plantilla nova precisa nome: a identidade vai á vista. Se non, o
    // foco vai ao deseño, que é o que se vén cambiar.
    if (!actual.id) { alternarBloque('identidade', true); nomeInput.focus() } else corpoDeseno.querySelector('[role="radio"][tabindex="0"]')?.focus()
  })
  // A vista previa faise xa ao abrir: o deseño vese desde o primeiro momento.
  if (api.previsualizar && autoVista.checked && !abrirIA) setTimeout(() => { if (!pechado) previsualizar() }, 0)

  return {
    cambiado,
    focus: () => nomeInput.focus(),
    // Quita a pantalla sen preguntar (a xanela pechouse por outro lado).
    destruir() {
      pechado = true
      asistente?.destruir()
      document.removeEventListener('keydown', onKeydown, true)
      container.replaceChildren()
      container.classList.remove('plantilla-pantalla')
    },
    // Repite o pechado normal (con confirmación se hai cambios).
    pedirPechar: cancelar,
    // Para as probas e o asistente.
    aplicar: () => gardar({ pechar: false }),
  }
}
