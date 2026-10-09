// Ferramentas do asistente da PANTALLA DE EDICIÓN DUNHA TAREFA (ámbito
// "tarefa", catálogo en asistente_ferramentas.go, ferramentasTarefa). Tocan só
// o borrador que hai no editor: o curso non se toca ata que o usuario prema
// Aplicar ou Gardar. A web (web_buscar/web_consultar) vai a Go.
import { PALETA } from '../informes/cronograma-modelo.js'

const CORES = PALETA.map((c) => c.clave)
const TAMANO_MAX = 30000

const erro = (m) => { throw new Error(m) }

// refsImaxe: os ids de imaxe que cita unha descrición (`![alt](img:ID`).
function refsImaxe(desc) {
  return [...String(desc).matchAll(/!\[[^\]]*\]\(img:([^\s)]+)/g)].map((m) => m[1])
}

// crearFerramentasTarefa devolve `executar(nome, args) -> {contido, erro}`.
//   editor  { ler(), cambiar({titulo,descricion,cor}), bloqueado }
//   api     { IAChamarFerramenta }
//   ui      { avisar(mensaxe, tipo), preguntar(pregunta, opcions) }
export function crearFerramentasTarefa({ editor, api, ui }) {
  const frontend = {
    editor_ler: () => editor.ler(),

    editor_cambiar(a) {
      if (editor.bloqueado) erro('o curso está bloqueado: esta tarefa só se pode ver')
      const cambios = {}
      if (a.titulo !== undefined) {
        const t = String(a.titulo).trim()
        if (!t) erro('o título non pode quedar baleiro')
        cambios.titulo = t
      }
      if (a.descricion !== undefined) cambios.descricion = String(a.descricion)
      if (a.cor !== undefined && a.cor !== '') {
        if (!CORES.includes(a.cor)) erro(`cor non válida: «${a.cor}». As posibles son: ${CORES.join(', ')}`)
        cambios.cor = a.cor
      }
      if (Object.keys(cambios).length === 0) erro('non indicaches ningún campo a cambiar')
      const antes = editor.ler()
      const avisos = []
      if (cambios.descricion !== undefined) {
        const novas = new Set(refsImaxe(cambios.descricion))
        const perdidas = refsImaxe(antes.descricion).filter((id) => !novas.has(id))
        if (perdidas.length) avisos.push(`a nova descrición xa non cita ${perdidas.length} imaxe(s) (${[...new Set(perdidas)].join(', ')}); se non era a intención, volve poñer a liña ![..](img:ID) ou o usuario pode desfacer o cambio`)
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
  // Só a web sae a Go: nada máis do backend se pode chamar dende aquí.
  const backendPermitido = new Set(['web_buscar', 'web_consultar'])

  const limitar = (s) => (s.length <= TAMANO_MAX ? s : s.slice(0, TAMANO_MAX) + `\n[... resultado cortado: ${s.length - TAMANO_MAX} caracteres máis ...]`)

  async function executar(nome, args) {
    try {
      if (frontend[nome]) {
        const v = await frontend[nome](args || {})
        return { contido: limitar(typeof v === 'string' ? v : JSON.stringify(v, null, 2)), erro: false }
      }
      if (!backendPermitido.has(nome)) return { contido: `a ferramenta «${nome}» non está dispoñible na pantalla de edición dunha tarefa`, erro: true }
      const r = await api.IAChamarFerramenta(nome, JSON.stringify(args || {}))
      return { contido: limitar(r.contido ?? ''), erro: !!r.erro }
    } catch (err) {
      return { contido: String(err?.message || err), erro: true }
    }
  }
  return { executar, nomesFrontend: Object.keys(frontend) }
}
