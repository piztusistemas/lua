// Ferramentas do asistente do EDITOR DE PLANTILLAS (ámbito "plantilla",
// catálogo en asistente_ferramentas.go, ferramentasPlantillaEditor). Tocan só o
// borrador que hai no editor: nada se garda no disco ata que o usuario prema
// Aplicar ou Gardar. Os adxuntos, a vista previa e a web van a Go, pero co
// borrador (e o id da plantilla) que pon esta capa: a IA non escolle sobre
// que plantilla traballa.

const MOTORES = ['', 'pdflatex', 'xelatex', 'lualatex']
const TEXTOS = ['nome', 'descricion', 'latex', 'markdown', 'portada', 'sinatura', 'introducion', 'estilo']
const TAMANO_MAX = 30000
const HEX = /^[0-9A-Fa-f]{6}$/

const erro = (m) => { throw new Error(m) }

// Campos de 'ofimatica': texto, cor hexadecimal ou número (con rango).
const OFI_TEXTO = ['fonte', 'portada', 'cabeceira', 'pe']
const OFI_COR = ['corPrincipal', 'corSecundaria', 'corTexto', 'corApagada', 'corLinha']
const OFI_NUM = {
  tamanoCorpo: [8, 16, 'pt'], interlinado: [1, 3, ''],
  marxeSuperior: [0.5, 6, 'cm'], marxeInferior: [0.5, 6, 'cm'], marxeEsquerda: [0.5, 6, 'cm'], marxeDereita: [0.5, 6, 'cm'],
}

// validarOfimatica devolve só os subcampos correctos, ou lanza un erro que
// explica cal falla (a IA le a mensaxe e corrixe).
export function validarOfimatica(o) {
  if (!o || typeof o !== 'object' || Array.isArray(o)) erro('ofimatica ten que ser un obxecto')
  const out = {}
  const descoñecidos = []
  for (const [k, v] of Object.entries(o)) {
    if (OFI_TEXTO.includes(k)) {
      out[k] = String(v ?? '').replace(/\r\n/g, '\n')
    } else if (OFI_COR.includes(k)) {
      let c = String(v ?? '').trim().replace(/^#/, '')
      if (c.length === 3) c = c.replace(/./g, (x) => x + x)
      if (c !== '' && !HEX.test(c)) erro(`${k}: cor non válida «${v}» (usa hexadecimal, p.ex. 1F3864)`)
      out[k] = c.toUpperCase()
    } else if (k in OFI_NUM) {
      const n = Number(v)
      const [min, max, unidade] = OFI_NUM[k]
      if (v === '' || v === null || Number.isNaN(n)) erro(`${k}: ten que ser un número`)
      if (n !== 0 && (n < min || n > max)) erro(`${k}: ${n} está fóra de rango (${min}-${max}${unidade ? ' ' + unidade : ''}; 0 = o por defecto)`)
      out[k] = n
    } else {
      descoñecidos.push(k)
    }
  }
  return { valores: out, descoñecidos }
}

function validarVariables(v) {
  if (!Array.isArray(v)) erro('variables ten que ser unha lista')
  const vistos = new Set()
  return v.map((x, i) => {
    if (!x || typeof x !== 'object') erro(`variables[${i}] non é un obxecto`)
    const nome = String(x.nome ?? '').trim().toUpperCase()
    if (!/^[A-Z][A-Z0-9_]*$/.test(nome)) erro(`variables[${i}].nome «${x.nome}» non vale: só letras maiúsculas, díxitos e _, e empeza por letra`)
    if (vistos.has(nome)) erro(`a variable ${nome} está repetida`)
    vistos.add(nome)
    return { nome, etiqueta: String(x.etiqueta ?? ''), valor: String(x.valor ?? '') }
  })
}

// crearFerramentasPlantilla devolve `executar(nome, args) -> {contido, erro}`.
//   editor  { ler(), borrador(), cambiar(cambios), id(), garantirGardada(), recargarImaxes(), motoresValidos? }
//   api     { IAChamarFerramenta }
//   ui      { avisar(mensaxe, tipo), preguntar(pregunta, opcions) }
export function crearFerramentasPlantilla({ editor, api, ui }) {
  const frontend = {
    plantilla_editor_ler: () => editor.ler(),

    plantilla_editor_cambiar(a) {
      const cambios = {}
      const avisos = []
      for (const k of TEXTOS) {
        if (a[k] === undefined) continue
        const v = String(a[k])
        if (k === 'nome' && !v.trim()) erro('o nome non pode quedar baleiro')
        cambios[k] = k === 'nome' || k === 'descricion' ? v.trim() : v.replace(/\r\n/g, '\n')
      }
      if (cambios.latex !== undefined) {
        const n = cambios.latex.split('{{CORPO}}').length - 1
        if (n === 0) erro('o LaTeX ten que levar {{CORPO}} (é onde vai o informe); non se cambiou nada')
        if (n > 1) erro('o LaTeX leva {{CORPO}} máis dunha vez: ten que aparecer só unha')
      }
      if (a.motor !== undefined) {
        const m = String(a.motor).trim().toLowerCase()
        if (!MOTORES.includes(m)) erro(`motor non válido: «${a.motor}» (pdflatex, xelatex, lualatex ou baleiro)`)
        cambios.motor = m
      }
      if (a.variables !== undefined) cambios.variables = validarVariables(a.variables)
      if (a.textos !== undefined) {
        // Só se cambian os indicados: partimos dos actuais e un baleiro quita o texto.
        if (!a.textos || typeof a.textos !== 'object' || Array.isArray(a.textos)) erro('textos ten que ser un obxecto clave -> texto')
        const claves = new Set((editor.ler().textosDispoñibles || []).map((x) => x.clave))
        const novos = { ...(editor.ler().textos || {}) }
        for (const [k, v] of Object.entries(a.textos)) {
          const clave = String(k).trim().toUpperCase()
          if (claves.size && !claves.has(clave)) erro(`o texto «${k}» non existe. Claves: ${[...claves].join(', ')}`)
          if (String(v ?? '').trim()) novos[clave] = String(v)
          else delete novos[clave]
        }
        cambios.textos = novos
      }
      if (a.ofimatica !== undefined) {
        const { valores, descoñecidos } = validarOfimatica(a.ofimatica)
        if (descoñecidos.length) avisos.push(`campos de ofimatica descoñecidos, ignorados: ${descoñecidos.join(', ')}`)
        if (Object.keys(valores).length) cambios.ofimatica = valores
      }
      const descoñecidosTop = Object.keys(a).filter((k) => ![...TEXTOS, 'motor', 'variables', 'textos', 'ofimatica', 'id'].includes(k))
      if (descoñecidosTop.length) avisos.push(`campos ignorados (non existen): ${descoñecidosTop.join(', ')}`)
      if (Object.keys(cambios).length === 0) erro('non indicaches ningún campo válido a cambiar')
      if (cambios.ofimatica && cambios.ofimatica.cabeceira !== undefined) {
        // Un aviso útil: as imaxes das que fala o deseño deben existir na plantilla.
        const tenImaxes = new Set(editor.ler().imaxes || [])
        for (const nome of [...String(cambios.ofimatica.cabeceira).matchAll(/@imaxe\s+(\S+)/g), ...String(cambios.ofimatica.pe || '').matchAll(/@imaxe\s+(\S+)/g)].map((m) => m[1])) {
          if (!tenImaxes.has(nome)) avisos.push(`a cabeceira/pé usa a imaxe «${nome}», que aínda non está no cartafol da plantilla`)
        }
      }
      editor.cambiar(cambios)
      return { cambiados: Object.keys(cambios), ...(avisos.length ? { avisos } : {}) }
    },

    usuario_avisar(a) {
      const m = String(a.mensaxe ?? '').trim()
      if (!m) erro('falta a mensaxe')
      ui.avisar(m, a.tipo === 'aviso' ? 'aviso' : 'info')
      return { amosado: true }
    },

    async usuario_preguntar(a) {
      const pregunta = String(a.pregunta ?? '').trim()
      if (!pregunta) erro('falta a pregunta')
      const opcions = Array.isArray(a.opcions) ? a.opcions.map(String).filter(Boolean) : []
      const r = await ui.preguntar(pregunta, opcions)
      return r === null || r === undefined ? { resposta: null, nota: 'o usuario non respondeu' } : { resposta: r }
    },
  }

  // Ferramentas de Go permitidas neste ámbito, e como se preparan os argumentos.
  const PROBAS = new Set(['plantilla_previsualizar', 'plantilla_comprobar'])
  const NA_PLANTILLA = new Set(['adxunto_gardar_en_plantilla', 'plantilla_imaxe_eliminar'])
  const LIBRES = new Set(['adxuntos_listar', 'adxunto_ler', 'adxunto_ver', 'adxunto_extraer_imaxes', 'web_buscar', 'web_consultar',
    // Bóveda de Obsidian (só chegan á IA se o documento ten bóveda conectada; ver boveda.go)
    'boveda_info', 'boveda_listar', 'boveda_ler', 'boveda_buscar', 'boveda_enlaces', 'boveda_pendentes', 'boveda_lint', 'boveda_escribir'])

  const limitar = (s) => (s.length <= TAMANO_MAX ? s : s.slice(0, TAMANO_MAX) + `\n[... resultado cortado: ${s.length - TAMANO_MAX} caracteres máis ...]`)

  async function chamarGo(nome, args) {
    const r = await api.IAChamarFerramenta(nome, JSON.stringify(args))
    return { contido: limitar(r.contido ?? ''), erro: !!r.erro, imaxes: r.imaxes || [] }
  }

  async function executar(nome, args) {
    try {
      if (frontend[nome]) {
        const v = await frontend[nome](args || {})
        return { contido: limitar(typeof v === 'string' ? v : JSON.stringify(v, null, 2)), erro: false }
      }
      if (PROBAS.has(nome)) {
        // Proba o BORRADOR (e o id, para atopar as súas imaxes): a IA só escolle formato e páxina.
        const b = editor.borrador()
        const extra = {}
        if (args && args.formato !== undefined) extra.formato = args.formato
        if (args && args.paxina !== undefined) extra.paxina = args.paxina
        if (args && args.documento !== undefined) extra.documento = args.documento
        return await chamarGo(nome, { ...b, ...extra })
      }
      if (NA_PLANTILLA.has(nome)) {
        // Sempre na plantilla que hai no editor; se aínda é nova ou de exemplo,
        // gárdase antes (ou o cartafol de imaxes non existiría).
        const ok = await editor.garantirGardada()
        if (!ok) return { contido: 'A plantilla non se puido gardar antes (revisa o nome e que o LaTeX leve {{CORPO}}): non se fixo nada.', erro: true }
        const r = await chamarGo(nome, { ...(args || {}), plantilla: editor.id() })
        await editor.recargarImaxes()
        return r
      }
      if (LIBRES.has(nome)) return await chamarGo(nome, args || {})
      return { contido: `a ferramenta «${nome}» non está dispoñible no editor de plantillas`, erro: true }
    } catch (err) {
      return { contido: String(err?.message || err), erro: true }
    }
  }
  return { executar, nomesFrontend: Object.keys(frontend) }
}
