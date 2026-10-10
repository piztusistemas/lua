// Ferramentas do asistente do DESEÑADOR DE FORMULARIOS (ámbito "formulario",
// catálogo en formulario_ia.go). Tocan só o borrador que hai no deseñador:
// nada se garda no disco ata que o usuario prema «Gardar formulario».
//
// Todo cambio pasa POLA VALIDACIÓN DE GO (formulario_validar) antes de entrar
// no borrador: a IA equivócase con ids, tipos e ligazóns, e a mensaxe de erro
// exacta é o que lle permite corrixir. Se non valida, non se cambia nada.
import { CLAVES_ICONA } from '../formularios/iconas.js'

const TAMANO_MAX = 30000
const erro = (m) => { throw new Error(m) }
const CLAVES = ['nome', 'descricion', 'titulo', 'icona', 'portada', 'sinatura', 'campos', 'fluxo']

// crearFerramentasFormulario devolve `executar(nome, args) -> {contido, erro, imaxes}`.
//   editor  { ler(), borrador(), cambiar(formulario), datos() }   (o deseñador)
//   api     { IAChamarFerramenta, FluxoAutomaticoDe }
//   ui      { avisar(mensaxe, tipo), preguntar(pregunta, opcions) }
export function crearFerramentasFormulario({ editor, api, ui }) {
  const chamarGo = async (nome, args) => {
    const r = await api.IAChamarFerramenta(nome, JSON.stringify(args))
    const contido = r.contido ?? ''
    return {
      contido: contido.length <= TAMANO_MAX ? contido : contido.slice(0, TAMANO_MAX) + `\n[... resultado cortado: ${contido.length - TAMANO_MAX} caracteres máis ...]`,
      erro: !!r.erro, imaxes: r.imaxes || [],
    }
  }

  const frontend = {
    async formulario_ler() {
      const b = editor.borrador()
      const saida = {
        nome: b.nome, descricion: b.descricion || '', titulo: b.titulo || '', icona: b.icona || 'formulario',
        portada: !!b.portada, sinatura: !!b.sinatura, campos: b.campos, gardado: !!b.id, id: b.id || null,
        automatico: !b.fluxo, fluxo: b.fluxo || null,
      }
      if (!b.fluxo && api.FluxoAutomaticoDe) {
        try {
          saida.fluxoAutomatico = await api.FluxoAutomaticoDe(b)
          saida.nota = 'Sen tubaxe propia: o documento sae co cableado automático (fluxoAutomatico, só de referencia). Se queres cambiar algo que non sexa o tipo dos campos, pon un fluxo propio.'
        } catch { /* só é de axuda */ }
      }
      const d = editor.datos?.()
      if (d && Object.keys(d.valores || {}).length) saida.datosActuais = d.valores
      return saida
    },

    async formulario_cambiar(a) {
      const dados = CLAVES.filter((k) => a[k] !== undefined)
      if (!dados.length) erro('non indicaches ningún campo a cambiar (nome, descricion, titulo, icona, portada, sinatura, campos, fluxo)')
      const descoñecidos = Object.keys(a).filter((k) => !CLAVES.includes(k))
      const cand = JSON.parse(JSON.stringify(editor.borrador()))
      for (const k of dados) {
        const v = a[k]
        if (k === 'campos') {
          if (!Array.isArray(v)) erro('campos ten que ser unha lista')
          cand.campos = v
        } else if (k === 'fluxo') {
          if (v === null || v === 'automatico' || v === 'automático') delete cand.fluxo
          else if (typeof v === 'object' && !Array.isArray(v)) cand.fluxo = v
          else erro('fluxo ten que ser un obxecto {nodos, documento}, ou "automatico" para volver ao cableado automático')
        } else if (k === 'icona') {
          if (!CLAVES_ICONA.includes(v)) erro(`icona non válida «${v}». Usa: ${CLAVES_ICONA.join(', ')}`)
          cand.icona = v
        } else if (k === 'portada' || k === 'sinatura') {
          cand[k] = v === true || v === 'true'
        } else {
          cand[k] = String(v ?? '').trim()
        }
      }
      // Se cambian os campos e hai tubería propia, axústase aos campos novos/quitados.
      const sincronizar = a.campos !== undefined && a.fluxo === undefined && !!cand.fluxo
      const r = JSON.parse((await api.IAChamarFerramenta('formulario_validar', JSON.stringify({ formulario: cand, sincronizar }))).contido)
      if (!r.ok) erro(`o formulario non valida, non se cambiou nada: ${r.erro}`)
      editor.cambiar(r.formulario)
      const saida = { cambiados: dados, campos: r.formulario.campos.length, automatico: r.automatico }
      if (sincronizar) saida.nota = 'A tubaxe axustouse aos campos: revisa que os campos novos saian onde queres.'
      if (r.problemas?.length) saida.problemas = r.problemas
      if (descoñecidos.length) saida.ignorados = descoñecidos
      return saida
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

  // Ferramentas de Go permitidas neste ámbito.
  const LIBRES = new Set(['formulario_axuda', 'adxuntos_listar', 'adxunto_ler', 'adxunto_ver', 'adxunto_extraer_imaxes', 'web_buscar', 'web_consultar',
    // Bóveda de Obsidian (só chegan á IA se o documento ten bóveda conectada; ver boveda.go)
    'boveda_info', 'boveda_listar', 'boveda_ler', 'boveda_buscar', 'boveda_enlaces', 'boveda_pendentes', 'boveda_lint', 'boveda_escribir'])

  async function executar(nome, args) {
    try {
      if (frontend[nome]) {
        const v = await frontend[nome](args || {})
        const s = typeof v === 'string' ? v : JSON.stringify(v, null, 2)
        return { contido: s.length <= TAMANO_MAX ? s : s.slice(0, TAMANO_MAX) + '\n[... cortado ...]', erro: false }
      }
      if (nome === 'formulario_probar') {
        // Proba o BORRADOR: a IA só escolle os datos de mostra, o formato e a páxina.
        const extra = { formulario: editor.borrador() }
        for (const k of ['datos', 'paxina', 'formato']) if (args && args[k] !== undefined) extra[k] = args[k]
        return await chamarGo(nome, extra)
      }
      if (LIBRES.has(nome)) return await chamarGo(nome, args || {})
      return { contido: `a ferramenta «${nome}» non está dispoñible no deseñador de formularios`, erro: true }
    } catch (err) {
      return { contido: String(err?.message || err), erro: true }
    }
  }
  return { executar, nomesFrontend: Object.keys(frontend) }
}
