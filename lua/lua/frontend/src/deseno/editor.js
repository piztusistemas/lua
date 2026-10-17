// O BLOQUE «DESEÑO» do editor de plantillas: o deseño do PDF sen LaTeX.
//
// Vai por capas, da máis sinxela á de máis control:
//   1. letra (combinacións con nome), tamaño e cor; «tomar as cores do
//      logotipo» e «copiar o deseño dun documento»;
//   2. tres barras (formal, aire, cor) e os interruptores do mundo real
//      (branco e negro, aforrar papel, lectura fácil);
//   3. «Cada parte do documento»: os elementos, cada un cos seus poucos
//      controis e, ao final, «Editar en LaTeX» (o control total);
//   4. letras por papel (corpo, títulos, etiquetas, fórmulas).
// Ademais: desfacer (historial), fixar campos (as barras, as variantes e a IA
// non os tocan) e «Probar variantes» (escoller entre dúas propostas).
//
// O deseño é datos (modelo.js); este ficheiro só pinta e chama a `cambiar`.

import { el, boton, ficheiroABase64 } from '../plantillas-util.js'
import { confirmar } from '../comun/dialogs.js'
import {
  COMBINACIONS, elementosDe, CONTROIS_ELEMENTO, aplicarBarra, aplicarParche, limpar, normalizarCor, valor, variantes, describir, fixado, contrasteBranco,
  validarParche, paxinaRepresentativa, parteDeCampo,
} from './modelo.js'

// A mostra «Aa» de cada combinación vai na letra de verdade: as mesmas fontes
// ca o PDF, servidas á interface (style.css, @font-face «Lua …»).
const MOSTRA_LETRA = {
  lua: '"Lua Heros", "Helvetica Neue", Arial, sans-serif',
  clasica: '"Lua Termes", "Times New Roman", serif',
  elegante: '"Lua Pagella", Palatino, serif',
  lexible: '"Lua Schola", "Century Schoolbook", serif',
  moderna: '"Lua Fira", "Fira Sans", sans-serif',
  academica: '"Lua Libertinus", "Libertinus Serif", serif',
}
const MODELOS_PORTADA = ['', 'sobria', 'oficial', 'sen']

// grupoRadio: nun grupo de opcións (role=radio) só a marcada recibe o foco co
// tabulador e as frechas moven a escolla (o patrón de teclado dos radios).
function grupoRadio(grupo) {
  const opcions = [...grupo.querySelectorAll('[role="radio"]')]
  const marcada = opcions.find((o) => o.getAttribute('aria-checked') === 'true') || opcions[0]
  for (const o of opcions) o.tabIndex = o === marcada ? 0 : -1
  grupo.addEventListener('keydown', (e) => {
    const i = opcions.indexOf(document.activeElement)
    if (i < 0) return
    const paso = { ArrowRight: 1, ArrowDown: 1, ArrowLeft: -1, ArrowUp: -1 }[e.key]
    if (!paso) return
    e.preventDefault()
    const seguinte = opcions[(i + paso + opcions.length) % opcions.length]
    seguinte.click()
    // Volveuse pintar o bloque: o foco vai á opción nova.
    requestAnimationFrame(() => {
      const nova = document.querySelector(`[data-radio="${seguinte.dataset.radio}"]`)
      nova?.focus()
    })
  })
}
const CORES_RAPIDAS = ['4F46E5', '1D4ED8', '0F766E', '15803D', 'B91C1C', 'C2410C', '7C3AED', '334155']
const HISTORIAL_MAX = 50

// montarBloqueDeseno(nodo, opcións) pinta o bloque en `nodo`.
//   t, api     tradución e bindings (combinacions, coresImaxe, importarDeseno, estiloElemento, fontes)
//   ler()      o deseño actual (obxecto, pode ser {})
//   cambiar(d) fixa o deseño novo (o editor marca cambios e refresca a vista previa)
//   estiloLibre()       o LaTeX libre (bloque Estilo) da plantilla
//   quitarEstiloLibre() baleira o bloque Estilo
//   imaxeLogo()         promesa co data URI do logotipo da plantilla (ou null)
//   previsualizarCon(d) promesa co resultado de compilar a plantilla co deseño d
//   ferramenta()        a ferramenta do documento que se edita (só se mostran as súas partes)
//   pedirIA(texto)      abre o asistente coa caixa xa empezada (opcional)
//   nomePlantilla()     o nome da plantilla (para o ficheiro do deseño)
// `cambiar(d, parte)` recibe tamén a parte do documento que máis cambia (a
// vista previa salta á súa páxina).
// Devolve { pintar, abrirElemento, desfacer }.
export function montarBloqueDeseno(nodo, { t, api, ler, cambiar, estiloLibre = () => '', quitarEstiloLibre = () => {}, imaxeLogo = async () => null, previsualizarCon = null, imaxes = () => [], ferramenta = () => '', pedirIA = null, nomePlantilla = () => '' }) {
  const historial = []
  let coresLogo = []
  let combinacions = COMBINACIONS.map((id) => ({ id, nome: t(`deseno.letra.${id}`, id), descricion: t(`deseno.letra.${id}.desc`, '') }))
  Promise.resolve(api.combinacions?.()).then((l) => {
    if (Array.isArray(l) && l.length) {
      combinacions = l.map((c) => ({ id: c.id, nome: t(`deseno.letra.${c.id}`, c.nome), descricion: t(`deseno.letra.${c.id}.desc`, c.descricion) }))
      pintar()
    }
  }).catch(() => { /* sen backend: os nomes de tradución */ })

  // Cada cambio garda o estado anterior para poder desfacelo.
  function mudar(novo, parte = '') {
    const antes = ler() || {}
    const despois = limpar(novo)
    if (JSON.stringify(antes) === JSON.stringify(despois)) return
    historial.push(antes)
    if (historial.length > HISTORIAL_MAX) historial.shift()
    cambiar(despois, parte)
    pintar()
  }
  function desfacer() {
    const anterior = historial.pop()
    if (anterior === undefined) return
    cambiar(anterior, '')
    pintar()
  }
  const poñer = (campo, v) => mudar({ ...(ler() || {}), [campo]: v }, parteDeCampo(campo))

  // Botón de fixar: un cadeado ao carón de cada control principal.
  function cadeado(campo) {
    const d = ler() || {}
    const b = boton(fixado(d, campo) ? '🔒' : '🔓', 'deseno-cadeado', () => {
      const actual = ler() || {}
      const fix = new Set(actual.fixados || [])
      if (fix.has(campo)) fix.delete(campo)
      else fix.add(campo)
      mudar({ ...actual, fixados: [...fix].sort() })
    })
    const fixo = fixado(d, campo)
    b.setAttribute('aria-pressed', String(fixo))
    b.title = fixo
      ? t('deseno.fixado.si', 'Fixado: as barras, as variantes e a IA non o cambian. Preme para soltalo.')
      : t('deseno.fixado.non', 'Fixar: que as barras, as variantes e a IA non o cambien.')
    b.setAttribute('aria-label', b.title)
    return b
  }

  const fila = (titulo, campo, ...fillos) => {
    const f = el('div', 'deseno-fila')
    const cab = el('div', 'deseno-fila__cab')
    cab.appendChild(el('span', 'deseno-fila__tit', titulo))
    if (campo) cab.appendChild(cadeado(campo))
    f.append(cab, ...fillos)
    return f
  }

  let elementoAberto = ''
  let detallesAbertos = false

  // ---- Cabeceira e pé ----
  const PEZAS = [
    ['@logo', '🖼️', t('deseno.peza.logo', 'Logotipo')],
    ['{{CENTRO}}', '🏫', t('deseno.peza.centro', 'Nome do centro')],
    ['{{DEPARTAMENTO}}', '🏷️', t('deseno.peza.departamento', 'Departamento')],
    ['{{CURSO}}', '📅', t('deseno.peza.curso', 'Curso')],
    ['{{TITULO}}', '📄', t('deseno.peza.titulo', 'Título do documento')],
  ]
  const ZONAS = [['esquerda', t('deseno.zona.esquerda', 'Esquerda')], ['centro', t('deseno.zona.centro', 'Centro')], ['dereita', t('deseno.zona.dereita', 'Dereita')]]
  // Modelos para empezar: un clic e a cabeceira queda feita.
  const MODELOS_CABECEIRA = [
    [t('deseno.cab.m.logo', 'Logotipo e centro'), { esquerda: '@logo', dereita: '{{CENTRO}}' }, true],
    [t('deseno.cab.m.centro', 'Centro e curso'), { esquerda: '{{CENTRO}}', dereita: '{{CURSO}}' }, true],
    [t('deseno.cab.m.todo', 'Centro, departamento e curso'), { esquerda: '{{CENTRO}}', centro: '{{DEPARTAMENTO}}', dereita: '{{CURSO}}' }, true],
    [t('deseno.cab.m.nada', 'Sen cabeceira'), {}, false],
  ]

  function pezaDe(linha) {
    const f = PEZAS.find((p) => p[0] === linha)
    if (f) return { ico: f[1], nome: f[2] }
    if (linha.startsWith('@imaxe')) return { ico: '🖼️', nome: linha.replace('@imaxe', '').trim() || t('deseno.peza.imaxe', 'Imaxe') }
    return { ico: '✏️', nome: linha }
  }

  function pintarCabeceiraPe(d) {
    const caixa = el('div', 'deseno-cabpe')
    caixa.appendChild(el('p', 'plantilla-axuda', t('deseno.cabpe.axuda2', 'Escolle que vai en cada esquina da cabeceira e do pé. O logotipo é o do curso; se queres outra imaxe, sóbea en «Imaxes da plantilla». Tamén vale para Word e OpenDocument.')))

    const zonasDe = (campo) => ({ esquerda: '', centro: '', dereita: '', ...(ler()?.[campo] || {}) })
    const gardarZonas = (campo, z) => {
      const lz = limpar(z)
      mudar({ ...(ler() || {}), [campo]: Object.keys(lz).length ? lz : '' }, parteDeCampo(campo))
    }

    for (const [campo, campoFilete, titulo, ico] of [
      ['cabeceira', 'cabeceiraFilete', t('deseno.cabeceira', 'Cabeceira'), '⬆️'],
      ['pe', 'peFilete', t('deseno.pe', 'Pé'), '⬇️'],
    ]) {
      const sec = el('section', `deseno-folla deseno-folla--${campo}`)
      const cab = el('div', 'deseno-folla__cab')
      cab.appendChild(el('strong', null, `${ico} ${titulo}`))
      sec.appendChild(cab)

      // Modelos (só na cabeceira) e «Baleirar».
      if (campo === 'cabeceira') {
        const m = el('div', 'deseno-folla__modelos')
        m.appendChild(el('span', 'deseno-folla__ou', t('deseno.cab.empezar', 'Empezar cun modelo:')))
        for (const [nome, zonas, filete] of MODELOS_CABECEIRA) {
          m.appendChild(boton(nome, 'deseno-chip', () => mudar({ ...(ler() || {}), cabeceira: Object.keys(zonas).length ? zonas : '', cabeceiraFilete: filete }, 'tarefas')))
        }
        sec.appendChild(m)
      }

      // A folla: tres zonas e, se se pide, a liña horizontal.
      const z = zonasDe(campo)
      const folla = el('div', 'deseno-folla__papel')
      const franxa = el('div', 'deseno-folla__franxa')
      for (const [zona, nomeZona] of ZONAS) {
        const col = el('div', `deseno-folla__zona deseno-folla__zona--${zona}`)
        col.appendChild(el('span', 'deseno-folla__zonanome', nomeZona))
        const linhas = (z[zona] || '').split('\n').map((x) => x.trim()).filter(Boolean)
        const pezas = el('div', 'deseno-folla__pezas')
        linhas.forEach((linha, idx) => {
          const pz = pezaDe(linha)
          const chip = el('span', 'deseno-peza')
          chip.append(el('span', null, `${pz.ico} ${pz.nome}`))
          const x = boton('✕', 'deseno-peza__x', () => {
            const novas = linhas.filter((_, k) => k !== idx)
            gardarZonas(campo, { ...z, [zona]: novas.join('\n') })
          })
          x.setAttribute('aria-label', t('deseno.peza.quitar', 'Quitar {nome}', { nome: pz.nome }))
          chip.appendChild(x)
          pezas.appendChild(chip)
        })
        col.appendChild(pezas)

        // Engadir unha peza.
        const sel = el('select', 'deseno-folla__engadir')
        sel.setAttribute('aria-label', `${titulo}: ${nomeZona}`)
        const o0 = el('option', null, t('deseno.peza.engadir', '＋ Engadir…'))
        o0.value = ''
        sel.appendChild(o0)
        const engadir = (txt) => {
          const novas = [...linhas, txt]
          gardarZonas(campo, { ...z, [zona]: novas.join('\n') })
        }
        for (const [cod, i2, nome] of PEZAS) {
          const o = el('option', null, `${i2} ${nome}`)
          o.value = cod
          sel.appendChild(o)
        }
        for (const im of imaxes()) {
          if (!/\.(png|jpe?g|gif|pdf)$/i.test(im)) continue
          const o = el('option', null, `🖼️ ${im}`)
          o.value = `@imaxe ${im}`
          sel.appendChild(o)
        }
        sel.addEventListener('change', () => { if (sel.value) engadir(sel.value) })
        col.appendChild(sel)

        const txt = el('input', 'deseno-folla__texto')
        txt.type = 'text'
        txt.placeholder = t('deseno.peza.propio', 'Ou escribe un texto…')
        txt.setAttribute('aria-label', `${titulo}: ${nomeZona}: ${t('deseno.peza.propio', 'Ou escribe un texto…')}`)
        txt.addEventListener('keydown', (e) => {
          if (e.key === 'Enter' && txt.value.trim()) { e.preventDefault(); engadir(txt.value.trim()) }
        })
        txt.addEventListener('change', () => { if (txt.value.trim()) engadir(txt.value.trim()) })
        col.appendChild(txt)
        franxa.appendChild(col)
      }
      folla.appendChild(franxa)
      if (d[campoFilete]) folla.appendChild(el('div', 'deseno-folla__filete'))
      sec.appendChild(folla)

      // Opcións: liña horizontal (e, no pé, onde vai o número de páxina).
      const opc = el('div', 'deseno-folla__opcions')
      const lf = el('label', 'deseno-interruptor deseno-interruptor--curto')
      const cf = el('input')
      cf.type = 'checkbox'
      cf.checked = !!d[campoFilete]
      cf.addEventListener('change', () => poñer(campoFilete, cf.checked))
      lf.append(cf, el('span', null, campo === 'cabeceira' ? t('deseno.filete.cab', '➖ Liña horizontal debaixo da cabeceira') : t('deseno.filete.pe', '➖ Liña horizontal enriba do pé')))
      opc.appendChild(lf)
      if (campo === 'pe') {
        const num = el('div', 'deseno-folla__numero')
        num.setAttribute('role', 'radiogroup')
        num.setAttribute('aria-label', t('deseno.num', 'Número de páxina'))
        num.appendChild(el('span', null, t('deseno.num', 'Número de páxina')))
        for (const [v, txtv] of [['esquerda', t('deseno.zona.esquerda', 'Esquerda')], ['centro', t('deseno.zona.centro', 'Centro')], ['', t('deseno.zona.dereita', 'Dereita')], ['non', t('deseno.num.non', 'Sen número')]]) {
          const b = boton(txtv, 'deseno-chip', () => poñer('numeroPaxina', v))
          b.setAttribute('role', 'radio')
          b.setAttribute('aria-checked', String((d.numeroPaxina || '') === v))
          num.appendChild(b)
        }
        opc.appendChild(num)
      }
      if (!zonasDe(campo).esquerda && !zonasDe(campo).centro && !zonasDe(campo).dereita && !d[campoFilete]) {
        sec.appendChild(el('p', 'plantilla-axuda', t('deseno.cabpe.baleira', 'Baleira: usa a da maqueta.')))
      }
      sec.appendChild(opc)
      caixa.appendChild(sec)
    }
    return caixa
  }

  function pintar() {
    const d = ler() || {}
    const partes = []

    // Aviso de LaTeX libre: manda sobre o editor visual.
    if (estiloLibre().trim()) {
      const aviso = el('div', 'deseno-aviso')
      aviso.appendChild(el('p', null, t('deseno.libre', 'Esta plantilla ten o deseño en LaTeX libre (bloque «Estilo»): manda sobre o que escollas aquí. Para usar o editor visual hai que deixar ese bloque baleiro.')))
      aviso.appendChild(boton(t('deseno.libre.pasar', 'Pasar ao editor visual'), null, () => {
        historial.push(ler() || {})
        quitarEstiloLibre()
        pintar()
      }))
      partes.push(aviso)
    }

    partes.push(el('p', 'plantilla-axuda', t('deseno.axuda', 'Escolle e mira a vista previa. Preme en calquera parte da vista previa para cambiar esa parte.')))

    // 1. Letra
    const letras = el('div', 'deseno-letras')
    letras.setAttribute('role', 'radiogroup')
    letras.setAttribute('aria-label', t('deseno.letra', 'Letra'))
    const letraActual = d.letra || 'lua'
    for (const c of combinacions) {
      const b = boton('', 'deseno-letra', () => poñer('letra', c.id === 'lua' ? '' : c.id))
      b.dataset.radio = 'letra-' + c.id
      b.setAttribute('role', 'radio')
      b.setAttribute('aria-checked', String(c.id === letraActual))
      const mostra = el('span', 'deseno-letra__mostra', 'Aa')
      mostra.style.fontFamily = MOSTRA_LETRA[c.id] || 'inherit'
      b.append(mostra, el('span', 'deseno-letra__nome', c.nome), el('span', 'deseno-letra__desc', c.descricion))
      letras.appendChild(b)
    }
    grupoRadio(letras)
    partes.push(fila(t('deseno.letra', 'Letra'), 'letra', letras))

    // Tamaño
    const tam = valor(d, 'tamano')
    const tamTxt = el('output', 'deseno-tamano__valor', `${tam} pt`)
    const menos = boton('−', 'deseno-tamano__btn', () => poñer('tamano', Math.max(8, Math.round((tam - 0.5) * 2) / 2)))
    const mais = boton('+', 'deseno-tamano__btn', () => poñer('tamano', Math.min(20, Math.round((tam + 0.5) * 2) / 2)))
    menos.setAttribute('aria-label', t('deseno.tamano.menos', 'Letra máis pequena'))
    mais.setAttribute('aria-label', t('deseno.tamano.mais', 'Letra máis grande'))
    const tamanoBox = el('div', 'deseno-tamano')
    tamanoBox.append(menos, tamTxt, mais)
    partes.push(fila(t('deseno.tamano', 'Tamaño do texto'), 'tamano', tamanoBox))

    // Cor
    const cores = el('div', 'deseno-cores')
    const corActual = normalizarCor(d.corPrincipal) || '4F46E5'
    const mostra = (hex, etiqueta) => {
      const b = boton('', 'deseno-cor', () => poñer('corPrincipal', hex === '4F46E5' ? '' : hex))
      b.style.background = '#' + hex
      b.setAttribute('aria-pressed', String(hex === corActual))
      b.title = etiqueta || '#' + hex
      b.setAttribute('aria-label', b.title)
      return b
    }
    for (const c of coresLogo) cores.appendChild(mostra(c, t('deseno.cor.logo', 'Cor do logotipo') + ' #' + c))
    if (coresLogo.length) cores.appendChild(el('span', 'deseno-cores__sep'))
    for (const c of CORES_RAPIDAS) cores.appendChild(mostra(c))
    const outra = el('input', 'deseno-cor__outra')
    outra.type = 'color'
    outra.value = '#' + corActual.toLowerCase()
    outra.title = t('deseno.cor.outra', 'Outra cor')
    outra.setAttribute('aria-label', outra.title)
    outra.addEventListener('change', () => poñer('corPrincipal', normalizarCor(outra.value)))
    cores.appendChild(outra)
    const logoBtn = boton('🎨 ' + t('deseno.cor.tomarLogo', 'Tomar as cores do logotipo'), 'deseno-accion', () => tomarCoresLogo())
    const corFila = fila(t('deseno.cor', 'Cor'), 'corPrincipal', cores, logoBtn)
    if (d.corPrincipal && contrasteBranco(normalizarCor(d.corPrincipal)) < 4.5) {
      corFila.appendChild(el('p', 'deseno-nota', t('deseno.cor.clara', 'É unha cor clara: nos rótulos Lúa usará unha versión máis escura para que se lean ben.')))
    }
    partes.push(corFila)

    // 2. Barras
    const barras = el('div', 'deseno-barras')
    for (const [id, esq, der] of [
      ['formal', t('deseno.barra.informal', 'Informal'), t('deseno.barra.formal', 'Formal')],
      ['aire', t('deseno.barra.compacto', 'Compacto'), t('deseno.barra.aireado', 'Aireado')],
      ['cor', t('deseno.barra.sobrio', 'Sobrio'), t('deseno.barra.colorido', 'Colorido')],
    ]) {
      const b = el('label', 'deseno-barra')
      const r = el('input')
      r.type = 'range'
      r.min = '-2'
      r.max = '2'
      r.step = '1'
      r.value = String(d.barras?.[id] || 0)
      r.setAttribute('aria-label', `${esq} – ${der}`)
      r.addEventListener('change', () => mudar(aplicarBarra(ler() || {}, id, Number(r.value)), id === 'cor' ? 'etiquetas' : 'titulos'))
      b.append(el('span', 'deseno-barra__esq', esq), r, el('span', 'deseno-barra__der', der))
      barras.appendChild(b)
    }
    partes.push(fila(t('deseno.barras', 'Carácter do documento'), null, barras))

    // Interruptores
    const inter = el('div', 'deseno-interruptores')
    for (const [campo, txt, axuda] of [
      ['brancoNegro', t('deseno.bn.titulo', '🖨️ Vou imprimir en branco e negro'), t('deseno.bn.axuda', 'As cores pasan a grises que se distinguen ben.')],
      ['aforrarPapel', t('deseno.aforrar.titulo', '📄 Aforrar papel'), t('deseno.aforrar.axuda', 'Marxes e espazos máis pequenos.')],
      ['lecturaFacil', t('deseno.lectura.titulo', '👓 Lectura fácil'), t('deseno.lectura.axuda', 'Letra grande, máis espazo entre liñas, sen xustificar nin cortar palabras.')],
    ]) {
      const l = el('label', 'deseno-interruptor')
      const c = el('input')
      c.type = 'checkbox'
      c.setAttribute('role', 'switch')
      c.checked = !!d[campo]
      c.addEventListener('change', () => poñer(campo, c.checked))
      const txts = el('span', 'deseno-interruptor__txt')
      txts.append(el('strong', null, txt), el('small', null, axuda))
      l.append(c, txts)
      inter.appendChild(l)
    }
    partes.push(fila(t('deseno.para', 'Para que é'), null, inter))

    // Portada: modelos con nome e onde vai o logotipo.
    const portadas = el('div', 'deseno-portadas')
    portadas.setAttribute('role', 'radiogroup')
    portadas.setAttribute('aria-label', t('deseno.portada', 'Portada'))
    for (const m of MODELOS_PORTADA) {
      const b = boton('', 'deseno-portada', () => poñer('portada', m))
      b.dataset.radio = 'portada-' + (m || 'cor')
      b.setAttribute('role', 'radio')
      b.setAttribute('aria-checked', String((d.portada || '') === m))
      const mini = el('span', `deseno-portada__mini deseno-portada__mini--${m || 'cor'}`)
      mini.style.setProperty('--cor', '#' + (normalizarCor(d.corPrincipal) || '4F46E5'))
      b.append(mini, el('span', null, t(`deseno.portada.${m || 'cor'}`, m || 'cor')))
      portadas.appendChild(b)
    }
    grupoRadio(portadas)
    const logo = el('label', 'deseno-control')
    const logoSel = el('select')
    for (const [v, txt] of [['', t('deseno.logo.modelo', 'Onde diga o modelo')], ['esquerda', t('deseno.logo.esquerda', 'Á esquerda')], ['centro', t('deseno.logo.centro', 'No centro')], ['dereita', t('deseno.logo.dereita', 'Á dereita')]]) {
      const o = el('option', null, txt)
      o.value = v
      logoSel.appendChild(o)
    }
    logoSel.value = d.logoPortada || ''
    logoSel.disabled = d.portada === 'sen'
    logoSel.addEventListener('change', () => poñer('logoPortada', logoSel.value))
    logo.append(el('span', null, t('deseno.logo', 'Logotipo')), logoSel)
    partes.push(fila(t('deseno.portada', 'Portada'), 'portada', portadas, logo))

    // Cabeceira e pé: unha folla de mostra con tres zonas onde se colocan
    // pezas (logotipo, centro, departamento, curso...), sen escribir códigos.
    partes.push(fila(t('deseno.cabpe', 'Cabeceira e pé'), 'cabeceira', pintarCabeceiraPe(d)))

    // Accións
    const accions = el('div', 'deseno-accions')
    if (previsualizarCon) accions.appendChild(boton('🔀 ' + t('deseno.variantes', 'Probar variantes'), 'deseno-accion', () => abrirVariantes()))
    accions.appendChild(boton('📄 ' + t('deseno.copiar', 'Copiar o deseño dun documento…'), 'deseno-accion', () => copiarDeDocumento()))
    const desf = boton('↶ ' + t('deseno.desfacer', 'Desfacer'), 'deseno-accion', () => desfacer())
    desf.disabled = historial.length === 0
    accions.appendChild(desf)
    accions.appendChild(boton('↺ ' + t('deseno.restablecer', 'Deseño de Lúa'), 'deseno-accion', () => mudar({})))
    partes.push(accions)

    // Kit do centro: levar este deseño a outras plantillas e a outras persoas.
    const kit = el('div', 'deseno-accions deseno-kit')
    kit.appendChild(el('span', 'deseno-kit__tit', t('deseno.kit', 'Deseño do centro')))
    if (api.gardarDesenoFicheiro) kit.appendChild(boton('💾 ' + t('deseno.kit.gardar', 'Gardar nun ficheiro…'), 'deseno-accion', () => gardarFicheiro()))
    if (api.lerDesenoFicheiro) kit.appendChild(boton('📂 ' + t('deseno.kit.abrir', 'Abrir un ficheiro…'), 'deseno-accion', () => abrirFicheiro()))
    if (api.aplicarDesenoATodas) kit.appendChild(boton('🏫 ' + t('deseno.kit.todas', 'Aplicar a todas as miñas plantillas…'), 'deseno-accion', () => aplicarATodas()))
    if (kit.children.length > 1) partes.push(kit)

    // 3. Cada parte do documento
    const det = el('details', 'deseno-detalles')
    det.open = detallesAbertos || !!elementoAberto
    det.addEventListener('toggle', () => { detallesAbertos = det.open })
    det.appendChild(el('summary', null, t('deseno.elementos', 'Cada parte do documento')))
    const lista = el('div', 'deseno-elementos')
    for (const id of elementosDe(ferramenta(), d)) lista.appendChild(pintarElemento(id, d))
    det.appendChild(lista)
    det.appendChild(pintarLetrasPorPapel(d))
    det.appendChild(pintarJSON(d))
    partes.push(det)

    nodo.replaceChildren(...partes)
  }

  async function gardarFicheiro() {
    try {
      const ruta = await api.gardarDesenoFicheiro(ler() || {}, nomePlantilla())
      if (ruta) avisar(t('deseno.kit.gardado', 'Gardado en {ruta}. Quen o abra no seu editor terá este mesmo deseño.', { ruta }))
    } catch (err) {
      avisar(String(err?.message || err))
    }
  }
  async function abrirFicheiro() {
    const f = await escollerFicheiro('.luadeseno,application/json')
    if (!f) return
    try {
      const d = await api.lerDesenoFicheiro(await f.text())
      mudar(d || {}, 'titulos')
      avisar(t('deseno.kit.aberto', 'Deseño aberto. Aplica a plantilla para gardalo.'))
    } catch (err) {
      avisar(String(err?.message || err))
    }
  }
  async function aplicarATodas() {
    const ok = await confirmar(t('deseno.kit.todas.confirmar', 'Poñer este deseño en todas as túas plantillas? As de exemplo non se tocan. Cada plantilla perde o deseño que tiña.'),
      { aceptar: t('deseno.kit.todas.si', 'Aplicar a todas'), cancelar: t('deseno.kit.todas.non', 'Cancelar') })
    if (!ok) return
    try {
      const n = await api.aplicarDesenoATodas(ler() || {})
      avisar(t('deseno.kit.todas.feito', 'Feito: {n} plantilla(s) levan agora este deseño.', { n }))
    } catch (err) {
      avisar(String(err?.message || err))
    }
  }

  function pintarElemento(id, d) {
    const e = d.elementos?.[id] || {}
    const caixa = el('div', 'deseno-elemento')
    caixa.dataset.elemento = id
    const aberto = elementoAberto === id
    const cab = boton('', 'deseno-elemento__cab', () => {
      elementoAberto = aberto ? '' : id
      pintar()
    })
    cab.setAttribute('aria-expanded', String(aberto))
    cab.append(el('span', null, t(`deseno.el.${id}`, id)))
    const marcas = []
    if (e.latex) marcas.push(t('deseno.el.latexPropio', 'LaTeX propio'))
    else if (Object.keys(e).length) marcas.push(t('deseno.el.cambiado', 'cambiado'))
    if (marcas.length) cab.append(el('span', 'deseno-elemento__marca', marcas.join(' · ')))
    caixa.appendChild(cab)
    if (!aberto) return caixa

    const corpo = el('div', 'deseno-elemento__corpo')
    const mudarEl = (cambio) => mudar(aplicarParche(ler() || {}, { elementos: { [id]: cambio } }), id)
    const controis = CONTROIS_ELEMENTO[id] || []
    if (id === 'texto') {
      corpo.appendChild(el('p', 'plantilla-axuda', t('deseno.el.texto.axuda', 'A letra, o tamaño, o espazo entre liñas e as marxes do texto cámbianse arriba (Letra, Tamaño, Estilo).')))
    }
    if (controis.includes('cor')) {
      const l = el('label', 'deseno-control')
      const c = el('input')
      c.type = 'color'
      c.value = '#' + (normalizarCor(e.cor) || '000000').toLowerCase()
      c.addEventListener('change', () => mudarEl({ cor: normalizarCor(c.value) }))
      l.append(el('span', null, t('deseno.el.cor', 'Cor')), c)
      if (e.cor) l.appendChild(boton(t('deseno.el.corDeseno', 'A do deseño'), null, () => mudarEl({ cor: '' })))
      corpo.appendChild(l)
    }
    if (controis.includes('escala')) {
      const l = el('label', 'deseno-control')
      const r = el('input')
      r.type = 'range'
      r.min = '50'
      r.max = '250'
      r.step = '5'
      r.value = String(Math.round((e.escala || 1) * 100))
      const out = el('output', null, `${r.value} %`)
      r.addEventListener('input', () => { out.textContent = `${r.value} %` })
      r.addEventListener('change', () => mudarEl({ escala: Number(r.value) === 100 ? 0 : Number(r.value) / 100 }))
      l.append(el('span', null, t('deseno.el.tamano', 'Tamaño')), r, out)
      corpo.appendChild(l)
    }
    for (const [campo, txt] of [['maiusculas', t('deseno.el.maiusculas', 'EN MAIÚSCULAS')], ['filete', t('deseno.el.filete', 'Coa liña de cor')]]) {
      if (!controis.includes(campo)) continue
      const l = el('label', 'deseno-control deseno-control--check')
      const c = el('input')
      c.type = 'checkbox'
      c.checked = campo === 'filete' ? !e.senFilete : !!e.maiusculas
      c.addEventListener('change', () => mudarEl(campo === 'filete' ? { senFilete: !c.checked } : { maiusculas: c.checked }))
      l.append(c, el('span', null, txt))
      corpo.appendChild(l)
    }
    const accions = el('div', 'deseno-elemento__accions')
    if (pedirIA) {
      accions.appendChild(boton('💬 ' + t('deseno.el.ia', 'Pedirllo á IA'), null, () => pedirIA(t('deseno.el.ia.prefixo', 'Na parte «{nome}» do documento: ', { nome: t(`deseno.el.${id}`, id) }))))
    }
    if (Object.keys(e).length) accions.appendChild(boton('↺ ' + t('deseno.el.restablecer', 'Como no deseño'), null, () => mudarEl(null)))
    corpo.appendChild(accions)

    // Control total: o LaTeX deste elemento.
    const latexDet = el('details', 'deseno-latex')
    latexDet.open = !!e.latex
    latexDet.appendChild(el('summary', null, t('deseno.el.latex', 'Editar en LaTeX')))
    const area = el('textarea', 'deseno-codigo')
    area.rows = 10
    area.spellcheck = false
    area.value = e.latex || ''
    area.setAttribute('aria-label', t('deseno.el.latex', 'Editar en LaTeX'))
    latexDet.addEventListener('toggle', async () => {
      if (latexDet.open && !area.value && api.estiloElemento) {
        try { area.value = await api.estiloElemento({ ...(ler() || {}), elementos: undefined }, id) } catch { /* sen backend */ }
      }
    })
    const latexAccions = el('div', 'deseno-elemento__accions')
    const latexErro = el('p', 'deseno-nota')
    latexErro.hidden = true
    latexErro.setAttribute('role', 'status')
    // Antes de usalo compróbase nun documento mínimo: o erro sae aquí, ao
    // carón da parte, e non ao xerar o documento enteiro.
    const comprobar = async () => {
      if (!api.comprobarElemento) return ''
      latexErro.hidden = false
      latexErro.textContent = t('deseno.el.latex.comprobando', 'Comprobando…')
      try {
        const msg = await api.comprobarElemento(ler() || {}, id, area.value)
        latexErro.textContent = msg ? t('deseno.el.latex.erro', 'Non compila: {erro}', { erro: msg }) : t('deseno.el.latex.ok', 'Compila ben.')
        latexErro.classList.toggle('deseno-nota--ok', !msg)
        return msg
      } catch (err) {
        latexErro.textContent = String(err?.message || err)
        return String(err?.message || err)
      }
    }
    const usarIgual = boton(t('deseno.el.latex.usarIgual', 'Usar igualmente'), null, () => mudarEl({ latex: area.value }))
    usarIgual.hidden = true
    latexAccions.append(
      boton(t('deseno.el.latex.usar', 'Usar este LaTeX'), null, async () => {
        const erro = await comprobar()
        if (erro) { usarIgual.hidden = false; return }
        mudarEl({ latex: area.value })
      }),
      usarIgual,
      boton(t('deseno.el.latex.quitar', 'Volver ao xerado'), null, () => { area.value = ''; mudarEl({ latex: '' }) }),
    )
    latexDet.append(el('p', 'plantilla-axuda', t('deseno.el.latex.axuda', 'Este LaTeX substitúe só esta parte. Os nomes dos macros e os seus argumentos teñen que quedar iguais.')), area, latexAccions, latexErro)
    corpo.appendChild(latexDet)
    caixa.appendChild(corpo)
    return caixa
  }

  // O deseño coma datos: velo, editalo ou levalo a outra IA (a skill de deseño
  // máis o deseño actual) e pegar o que responda.
  function pintarJSON(d) {
    const caixa = el('div', 'deseno-json')
    caixa.appendChild(el('h4', null, t('deseno.json', 'O deseño coma datos (para outra IA)')))
    caixa.appendChild(el('p', 'plantilla-axuda', t('deseno.json.axuda', 'Copia as instrucións e o deseño, pégaos en calquera IA (ChatGPT, Gemini, Claude…) co que queres, e pega aquí o JSON que che devolva.')))
    const area = el('textarea', 'deseno-codigo')
    area.rows = 6
    area.spellcheck = false
    area.value = JSON.stringify(d, null, 2)
    area.setAttribute('aria-label', t('deseno.json', 'O deseño coma datos (para outra IA)'))
    const accions = el('div', 'deseno-elemento__accions')
    accions.append(
      boton('📋 ' + t('deseno.json.copiar', 'Copiar instrucións para outra IA'), null, async () => {
        let skill = ''
        try { skill = (await api.skillDeseno?.()) || '' } catch { /* sen backend */ }
        const texto = `${skill}\n\n## Deseño actual\n\n\`\`\`json\n${JSON.stringify(ler() || {}, null, 2)}\n\`\`\`\n\nResponde só co JSON dos campos que cambian.\n`
        try {
          await navigator.clipboard.writeText(texto)
          avisar(t('deseno.json.copiado', 'Copiado. Pégao na outra IA e engade o que queres cambiar.'))
        } catch {
          area.value = texto
          area.select()
          avisar(t('deseno.json.copiaMan', 'Non se puido copiar: o texto está na caixa, cópiao á man.'))
        }
      }),
      boton(t('deseno.json.aplicar', 'Aplicar este JSON'), null, () => {
        try {
          const bruto = area.value.slice(area.value.indexOf('{'), area.value.lastIndexOf('}') + 1)
          const { parche, avisos } = validarParche(JSON.parse(bruto), ler() || {})
          mudar(aplicarParche(ler() || {}, parche))
          if (avisos.length) avisar(avisos.join(' · '))
        } catch (err) {
          avisar(t('deseno.json.erro', 'O JSON non vale: {erro}', { erro: String(err?.message || err) }))
        }
      }),
    )
    caixa.append(area, accions)
    return caixa
  }

  // Letras por papel: para quen quere escoller cada unha.
  let fontesSistema = null
  function pintarLetrasPorPapel(d) {
    const caixa = el('div', 'deseno-papeis')
    caixa.appendChild(el('h4', null, t('deseno.papeis', 'Letra de cada parte')))
    const familias = [['', t('deseno.papel.combinacion', 'A da combinación')], ['termes', 'Termes (Times)'], ['pagella', 'Pagella (Palatino)'], ['schola', 'Schola (Century)'],
      ['bonum', 'Bonum (Bookman)'], ['libertinus', 'Libertinus'], ['sourceserif', 'Source Serif'], ['heros', 'Heros (Helvetica)'], ['adventor', 'Adventor'], ['firasans', 'Fira Sans'], ['opensans', 'Open Sans']]
    const mates = [['', t('deseno.papel.combinacion', 'A da combinación')], ['lm', 'Latin Modern'], ['termes', 'Termes Math'], ['pagella', 'Pagella Math'], ['schola', 'Schola Math'], ['bonum', 'Bonum Math'], ['fira', 'Fira Math'], ['libertinus', 'Libertinus Math']]
    for (const [campo, txt, opcions] of [
      ['letraCorpo', t('deseno.papel.corpo', 'Texto'), familias], ['letraTitulos', t('deseno.papel.titulos', 'Títulos'), familias],
      ['letraEtiquetas', t('deseno.papel.etiquetas', 'Etiquetas e rótulos'), familias], ['letraMates', t('deseno.papel.mates', 'Fórmulas'), mates],
    ]) {
      const l = el('label', 'deseno-control')
      const s = el('select')
      for (const [v, n] of opcions) {
        const o = el('option', null, n)
        o.value = v
        s.appendChild(o)
      }
      if (campo !== 'letraMates') {
        const g = el('optgroup')
        g.label = t('deseno.papel.sistema', 'Do equipo (non viaxa a outros equipos)')
        for (const f of fontesSistema || []) {
          const o = el('option', null, f)
          o.value = f
          g.appendChild(o)
        }
        const actual = d[campo] || ''
        if (actual && !opcions.some(([v]) => v === actual) && !(fontesSistema || []).includes(actual)) {
          const o = el('option', null, actual)
          o.value = actual
          g.appendChild(o)
        }
        s.appendChild(g)
        s.addEventListener('focus', cargarFontes, { once: true })
      }
      s.value = d[campo] || ''
      s.addEventListener('change', () => poñer(campo, s.value))
      l.append(el('span', null, txt), s)
      caixa.appendChild(l)
    }
    return caixa
  }
  async function cargarFontes() {
    if (fontesSistema !== null || !api.fontes) return
    try { fontesSistema = (await api.fontes()) || [] } catch { fontesSistema = [] }
    pintar()
  }

  // ---------- Accións con backend ----------
  async function tomarCoresLogo() {
    let datos = null
    try { datos = await imaxeLogo() } catch { datos = null }
    if (!datos) {
      // Sen logotipo na plantilla: escóllese unha imaxe.
      const f = await escollerFicheiro('image/png,image/jpeg,image/gif')
      if (!f) return
      datos = await ficheiroABase64(f)
    }
    try {
      coresLogo = ((await api.coresImaxe(datos)) || []).slice(0, 5)
      if (!coresLogo.length) {
        avisar(t('deseno.cor.logoSen', 'O logotipo non ten cores (é branco, negro ou gris).'))
        pintar()
        return
      }
      mudar({ ...(ler() || {}), corPrincipal: fixado(ler(), 'corPrincipal') ? ler().corPrincipal : coresLogo[0] }, 'etiquetas')
    } catch (err) {
      avisar(String(err?.message || err))
    }
  }

  async function copiarDeDocumento() {
    const f = await escollerFicheiro('.pdf,.docx,application/pdf,application/vnd.openxmlformats-officedocument.wordprocessingml.document')
    if (!f) return
    try {
      const res = await api.importarDeseno(f.name, await ficheiroABase64(f))
      const proposta = res?.deseno || {}
      // Os campos fixados quedan como estaban.
      const actual = ler() || {}
      const novo = { ...actual }
      for (const [k, v] of Object.entries(proposta)) {
        if (k === 'version' || fixado(actual, k)) continue
        novo[k] = v
      }
      mudar(novo, 'titulos')
      avisar(t('deseno.copiado', 'Copiado: {que}', { que: (res?.explicacion || []).join('; ') }))
    } catch (err) {
      avisar(String(err?.message || err))
    }
  }

  let aviso = null
  function avisar(m) {
    if (!aviso || !aviso.isConnected) {
      aviso = el('p', 'deseno-nota')
      aviso.setAttribute('role', 'status')
    }
    aviso.textContent = m
    nodo.appendChild(aviso)
  }

  // ---------- Variantes ----------
  async function abrirVariantes() {
    let rolda = 0
    const overlay = el('div', 'overlay overlay--dialog deseno-variantes')
    const caixa = el('div', 'modal deseno-variantes__caixa')
    caixa.setAttribute('role', 'dialog')
    caixa.setAttribute('aria-modal', 'true')
    const titulo = el('h3', null, t('deseno.variantes.titulo', 'Cal che gusta máis?'))
    const que = el('p', 'plantilla-axuda')
    const columnas = el('div', 'deseno-variantes__cols')
    const pe = el('div', 'modal-actions')
    const outra = boton(t('deseno.variantes.ningunha', 'Ningunha: outra cousa'), null, () => { rolda++; rolda_() })
    const rematar = boton(t('deseno.variantes.rematar', 'Rematar'), 'primary', () => pechar())
    pe.append(outra, rematar)
    caixa.append(titulo, que, columnas, pe)
    overlay.appendChild(caixa)
    document.body.appendChild(overlay)
    const onKey = (e) => { if (e.key === 'Escape') { e.stopPropagation(); pechar() } }
    document.addEventListener('keydown', onKey, true)
    function pechar() {
      document.removeEventListener('keydown', onKey, true)
      overlay.remove()
      xeracion++
      api.cancelarVista?.()
    }
    let xeracion = 0
    async function rolda_() {
      const v = variantes(ler() || {}, rolda, coresLogo)
      if (!v) {
        que.textContent = t('deseno.variantes.todoFixado', 'Todo está fixado: solta algún cadeado para ver variantes.')
        columnas.replaceChildren()
        return
      }
      const esta = ++xeracion
      que.textContent = t(`deseno.variantes.${v.dimension}`, v.dimension)
      columnas.replaceChildren()
      for (const proposta of [v.a, v.b]) {
        const col = el('div', 'deseno-variantes__col')
        const imx = el('div', 'deseno-variantes__imx', t('deseno.variantes.compilando', 'Compilando…'))
        const escoller = boton(t('deseno.variantes.esta', 'Prefiro esta'), 'primary', () => { mudar(proposta, parteDeCampo({ cor: 'corPrincipal', aire: 'interlinado' }[v.dimension] || v.dimension)); rolda++; rolda_() })
        escoller.disabled = true
        col.append(el('p', 'deseno-variantes__desc', describir(proposta, t)), imx, escoller)
        columnas.appendChild(col)
        previsualizarCon(proposta).then((res) => {
          if (esta !== xeracion) return
          imx.replaceChildren()
          const paxina = (res?.pageImages || [])[paxinaRepresentativa(res) - 1]
          if (paxina) {
            const im = el('img')
            im.src = paxina
            im.alt = describir(proposta, t)
            imx.appendChild(im)
          } else {
            imx.textContent = (res?.warnings || []).join(' · ') || t('deseno.variantes.senImaxe', 'Sen vista previa')
          }
          escoller.disabled = false
        }).catch((err) => {
          if (esta === xeracion) imx.textContent = String(err?.message || err)
        })
      }
    }
    rolda_()
    rematar.focus()
  }

  // abrirElemento: o inspector da vista previa abre aquí a parte que se premeu.
  function abrirElemento(id) {
    elementoAberto = id
    detallesAbertos = true
    pintar()
    const n = nodo.querySelector(`[data-elemento="${id}"]`)
    n?.scrollIntoView?.({ block: 'center', behavior: 'smooth' })
    n?.classList.add('deseno-elemento--destacado')
    setTimeout(() => n?.classList.remove('deseno-elemento--destacado'), 1600)
    n?.querySelector('.deseno-elemento__cab')?.focus()
  }

  pintar()
  return { pintar, abrirElemento, desfacer }
}

function escollerFicheiro(accept) {
  return new Promise((resolve) => {
    const i = document.createElement('input')
    i.type = 'file'
    i.accept = accept
    i.addEventListener('change', () => resolve(i.files?.[0] || null))
    i.click()
  })
}
