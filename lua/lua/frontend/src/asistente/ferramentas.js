// Ferramentas do asistente que se executan no frontend: son as que tocan o
// curso en edición e a interface (as demais, de backend, van a Go polo binding
// IAChamarFerramenta). O catálogo (nome, descrición, argumentos) vive en Go
// (asistente_ferramentas.go, ferramentasFrontend): aquí só está o código.
//
// Coma noutros módulos, é unha factoría illada: recibe as súas dependencias
// (o controlador do informe, os bindings, callbacks da interface) e nunca
// importa main.js, para poder probarse (ferramentas.test.js).
import { crearFerramentasActas } from './ferramentas-actas.js'
import {
  PALETA, calendario, validarDatas, novaTarefa, gardarTarefa, eliminarTarefa, moverTarefa,
  moverTarefaAMes, totalTarefas, tarefasForaDeRango, imaxesUsadas, novoId,
} from '../informes/cronograma-modelo.js'

const CORES = PALETA.map((c) => c.clave)
const TAMANO_MAX_RESULTADO = 30000

// Cada ferramenta devolve o dato (obxecto ou texto) ou lanza un Error con
// unha mensaxe que se lle amosa á IA para que o corrixa.

const erro = (m) => { throw new Error(m) }
const txt = (v) => (v === undefined || v === null ? undefined : String(v))
const nonBaleiro = (v, nome) => {
  const s = (txt(v) ?? '').trim()
  if (!s) erro(`falta ${nome}`)
  return s
}

// mesesValidos lista as claves do rango de datas para as mensaxes de erro.
function mesesValidos(plan) {
  return calendario(plan.config, plan.monthsData).map((m) => m.clave)
}

// atoparTarefa devolve {clave, tarefa} polo id, ou erro.
function atoparTarefa(plan, id) {
  const buscado = nonBaleiro(id, 'o id da tarefa')
  for (const [clave, mes] of Object.entries(plan.monthsData)) {
    const tarefa = mes.tasks.find((t) => t.id === buscado)
    if (tarefa) return { clave, tarefa }
  }
  return erro(`non existe ningunha tarefa co id «${buscado}» (usa curso_ler para ver os ids)`)
}

function validarCor(cor) {
  if (cor === undefined || cor === null || cor === '') return undefined
  if (!CORES.includes(cor)) erro(`cor non válida: «${cor}». As posibles son: ${CORES.join(', ')}`)
  return cor
}

function comprobarNonBloqueado(plan) {
  if (plan.isLocked) erro('o curso está bloqueado: para editalo primeiro hai que desbloquealo (curso_configurar con bloqueado=false)')
}

// vistaCurso: o curso en forma lixeira para a IA (sen imaxes en base64).
export function vistaCurso(plan, { ambito = 'todo', mes, ruta = null } = {}) {
  const c = plan.config
  const config = {
    anoAcademico: plan.academicYear, dataInicio: c.startDate, dataFin: c.endDate, centro: c.schoolName,
    localidade: c.location, departamento: c.department, xefe: c.headOfDepartment, cargoSinatura: c.signatoryRole,
    introducion: c.introText, bloqueado: !!plan.isLocked, tenLogotipo: !!c.schoolLogo,
  }
  const out = { ficheiro: ruta }
  if (ambito !== 'tarefas') out.curso = config
  if (ambito !== 'config') {
    const meses = calendario(plan.config, plan.monthsData)
    const convertir = (m) => ({
      mes: m.clave, nome: `${m.nome} ${m.ano}`,
      tarefas: m.tarefas.map((t) => ({
        id: t.id, titulo: t.title, cor: t.color || 'blue', descricion: t.description,
        ...(t.images && Object.keys(t.images).length ? { imaxes: Object.keys(t.images).length } : {}),
      })),
    })
    out.meses = (mes ? meses.filter((m) => m.clave === mes) : meses).map(convertir)
    if (mes && out.meses.length === 0) erro(`o mes ${mes} non está no rango do curso (${meses.map((m) => m.clave).join(', ') || 'sen rango válido'})`)
    const fora = tarefasForaDeRango(plan)
    if (fora > 0) out.avisoForaDeRango = `${fora} tarefa(s) están en meses fóra do rango de datas e non saen no informe`
    out.totalTarefas = totalTarefas(plan)
  }
  return out
}

// crearFerramentas devolve `executar(nome, args) -> {contido, erro}`.
//   informes   controlador (informes/controlador.js)
//   api        bindings: IAChamarFerramenta, ListarPlantillas...
//   ui         accións da interface (ver main.js, crearAsistente)
export function crearFerramentas({ informes, api, ui }) {
  const plan = () => informes.getPlan()
  const aplicar = (novo) => informes.substituirPlan(novo, 'estrutura')
  // As ferramentas do cronograma (curso_*, tarefa_*) só valen coa ferramenta do
  // cronograma activa; coas actas hai as súas (actas_*, acta_*).
  const soCronograma = (fn) => (...args) => {
    if (informes.ferramenta?.id === 'actas') erro('a ferramenta activa son as ACTAS: usa as ferramentas actas_* e acta_* (ou ferramenta_activa con ferramenta="cronograma" para traballar co cronograma)')
    return fn(...args)
  }

  const frontend = {
    curso_ler: (a) => vistaCurso(plan(), { ambito: a.ambito || 'todo', mes: txt(a.mes), ruta: informes.ruta() }),

    curso_configurar(a) {
      const p = plan()
      const config = { ...p.config }
      const cambios = []
      const mapa = {
        dataInicio: 'startDate', dataFin: 'endDate', centro: 'schoolName', localidade: 'location',
        xefe: 'headOfDepartment', cargoSinatura: 'signatoryRole', introducion: 'introText', departamento: 'department',
      }
      let novo = { ...p }
      // Desbloquear é o único que se pode facer con o curso bloqueado.
      if (a.bloqueado !== undefined) {
        novo.isLocked = a.bloqueado === true || a.bloqueado === 'true'
        cambios.push('bloqueado')
      }
      if (p.isLocked && novo.isLocked) comprobarNonBloqueado(p)
      for (const [k, campo] of Object.entries(mapa)) {
        if (a[k] === undefined) continue
        config[campo] = txt(a[k])
        cambios.push(k)
      }
      if (a.anoAcademico !== undefined) { novo.academicYear = txt(a.anoAcademico); cambios.push('anoAcademico') }
      if (a.dataInicio !== undefined || a.dataFin !== undefined) {
        const e = validarDatas(config.startDate, config.endDate)
        if (e) erro(e)
      }
      if (cambios.length === 0) erro('non indicaches ningún campo a cambiar')
      novo = { ...novo, config }
      aplicar(novo)
      return { cambiados: cambios }
    },

    tarefa_engadir(a) {
      const p = plan()
      comprobarNonBloqueado(p)
      const mes = nonBaleiro(a.mes, 'o mes')
      const validos = mesesValidos(p)
      if (!validos.includes(mes)) erro(`o mes ${mes} non está no rango do curso. Meses válidos: ${validos.join(', ') || '(ningún: revisa as datas do curso)'}`)
      const titulo = nonBaleiro(a.titulo, 'o título')
      const tarefa = { ...novaTarefa(validarCor(a.cor) || 'blue'), title: titulo, description: txt(a.descricion) ?? '' }
      aplicar(gardarTarefa(p, mes, tarefa))
      return { id: tarefa.id, mes, titulo }
    },

    tarefa_editar(a) {
      const p = plan()
      comprobarNonBloqueado(p)
      const { clave, tarefa } = atoparTarefa(p, a.id)
      const nova = { ...tarefa }
      const cambios = []
      const avisos = []
      if (a.titulo !== undefined) { nova.title = nonBaleiro(a.titulo, 'o título'); cambios.push('titulo') }
      if (a.cor !== undefined) { nova.color = validarCor(a.cor) || nova.color; cambios.push('cor') }
      if (a.descricion !== undefined) {
        nova.description = txt(a.descricion)
        const usadas = imaxesUsadas(nova.description, tarefa.images || {})
        const perdidas = Object.keys(tarefa.images || {}).filter((id) => !usadas[id])
        if (perdidas.length) avisos.push(`a nova descrición xa non usa ${perdidas.length} imaxe(s) (${perdidas.join(', ')}), así que se quitaron da tarefa; se non era a intención, vólvea poñer coa liña ![..](img:ID) ou usa historial(desfacer)`)
        if (Object.keys(usadas).length) nova.images = usadas
        else delete nova.images
        cambios.push('descricion')
      }
      let novoPlan = gardarTarefa(p, clave, nova)
      let mesFinal = clave
      if (a.mes !== undefined && a.mes !== clave) {
        const validos = mesesValidos(p)
        if (!validos.includes(a.mes)) erro(`o mes ${a.mes} non está no rango do curso. Meses válidos: ${validos.join(', ')}`)
        novoPlan = moverTarefaAMes(novoPlan, clave, a.mes, nova.id)
        mesFinal = a.mes
        cambios.push('mes')
      }
      if (cambios.length === 0) erro('non indicaches ningún campo a cambiar')
      aplicar(novoPlan)
      return { id: nova.id, mes: mesFinal, cambiados: cambios, ...(avisos.length ? { avisos } : {}) }
    },

    tarefa_eliminar(a) {
      const p = plan()
      comprobarNonBloqueado(p)
      const { clave, tarefa } = atoparTarefa(p, a.id)
      aplicar(eliminarTarefa(p, clave, tarefa.id))
      return { eliminada: tarefa.title || tarefa.id, mes: clave }
    },

    tarefa_mover(a) {
      const p = plan()
      comprobarNonBloqueado(p)
      const { clave, tarefa } = atoparTarefa(p, a.id)
      if (a.direccion !== 'arriba' && a.direccion !== 'abaixo') erro("direccion ten que ser 'arriba' ou 'abaixo'")
      const novo = moverTarefa(p, clave, tarefa.id, a.direccion === 'arriba' ? -1 : 1)
      if (novo === p || JSON.stringify(novo.monthsData[clave]) === JSON.stringify(p.monthsData[clave])) {
        return { movida: false, motivo: 'xa está no extremo do mes' }
      }
      aplicar(novo)
      return { movida: true, mes: clave }
    },

    tarefa_abrir(a) {
      const { clave, tarefa } = atoparTarefa(plan(), a.id)
      informes.abrirTarefa(clave, tarefa)
      return { aberta: tarefa.title || tarefa.id }
    },

    // ---- Imaxes adxuntas que pasan ao JSON do curso ----
    async curso_logotipo(a) {
      comprobarNonBloqueado(plan())
      if (a.quitar === true || a.quitar === 'true') {
        const p = plan()
        aplicar({ ...p, config: { ...p.config, schoolLogo: '' } })
        return { logotipo: 'quitado do curso' }
      }
      const adx = nonBaleiro(a.adxunto, 'o id do adxunto (ou quitar=true)')
      const uri = await api.AdxuntoDataURI(adx, 900)
      const p = plan()
      aplicar({ ...p, config: { ...p.config, schoolLogo: uri } })
      return { logotipo: 'gardado no JSON do curso', kb: Math.round(uri.length / 1370) }
    },

    async tarefa_imaxe(a) {
      comprobarNonBloqueado(plan())
      atoparTarefa(plan(), a.id) // falla antes de pedir a imaxe se a tarefa non existe
      const adx = nonBaleiro(a.adxunto, 'o id do adxunto')
      const uri = await api.AdxuntoDataURI(adx, 1600)
      const p = plan()
      const { clave, tarefa } = atoparTarefa(p, a.id)
      const id = 'i' + novoId()
      const pct = Math.min(100, Math.max(5, Math.round(Number(a.largura) || 60)))
      const alt = String(a.alt ?? '').replace(/[[\]()\n]/g, ' ').trim()
      let liña = `![${alt}](img:${id} =${pct}%)`
      if (a.alinear === 'esquerda' || a.alinear === 'dereita') liña = `:::${a.alinear}\n${liña}\n:::`
      const desc = tarefa.description || ''
      const description = a.posicion === 'inicio' ? `${liña}\n\n${desc}`.trimEnd() : `${desc}${desc ? '\n\n' : ''}${liña}`
      aplicar(gardarTarefa(p, clave, { ...tarefa, description, images: { ...(tarefa.images || {}), [id]: uri } }))
      return { imaxeId: id, mes: clave, largura: `${pct}%` }
    },

    async curso_novo() {
      await informes.novo()
      return { novo: true }
    },

    async curso_abrir() {
      await informes.abrir()
      return { ruta: informes.ruta() }
    },

    async curso_gardar(a) {
      const ok = a.comoNovo === true ? await informes.gardarComo() : await informes.gardar()
      return { gardado: !!ok, ruta: informes.ruta() }
    },

    async informe_xerar() {
      const ok = await informes.xerar()
      const r = informes.ultimoXerar()
      if (!ok) erro((r.erro || 'non se puido xerar o informe') + ' (o detalle está no rexistro interno: rexistro_ler)')
      return { ok: true, paxinas: r.paxinas, avisos: r.avisos }
    },

    async informe_exportar(a) {
      const f = a.formato
      const accions = {
        pdf: informes.exportarPDF, tex: informes.exportarTex, md: informes.exportarMarkdown,
        docx: informes.exportarDocx, odt: informes.exportarOdt,
      }
      if (!accions[f]) erro("formato non válido: usa pdf, tex, md, docx ou odt")
      const path = await accions[f](null)
      if (!path) {
        const e = informes.ultimoErroExportar()
        if (e) erro(e)
        return { exportado: false, motivo: 'o usuario cancelou o diálogo de gardar' }
      }
      return { exportado: true, ruta: path }
    },

    historial(a) {
      if (a.accion === 'desfacer') informes.desfacer()
      else if (a.accion === 'refacer') informes.refacer()
      else erro("accion ten que ser 'desfacer' ou 'refacer'")
      return { feito: a.accion }
    },

    async resultado_xanela(a) {
      if (a.accion === 'desacoplar') await ui.desacoplarResultado()
      else if (a.accion === 'acoplar') await ui.acoplarResultado()
      else erro("accion ten que ser 'desacoplar' ou 'acoplar'")
      return { feito: a.accion }
    },

    async plantilla_escoller(a) {
      if (a.id === undefined) erro('falta o id da plantilla (cadea baleira = sen plantilla)')
      await ui.escollerPlantilla(String(a.id))
      return { activa: String(a.id) }
    },

    axustes_texto(a) {
      const actual = ui.getTipografia()
      const cambios = {}
      if (a.tamanoPt !== undefined) {
        const n = Number(a.tamanoPt)
        if (!Number.isFinite(n) || n < 0 || n > 30) erro('tamanoPt ten que ser un número entre 0 e 30')
        cambios.tamanoPt = n
      }
      if (a.interlinado !== undefined) {
        const n = Number(a.interlinado)
        if (!Number.isFinite(n) || n < 0 || n > 3) erro('interlinado ten que ser un número entre 0 e 3')
        cambios.interlinado = n
      }
      if (a.fonte !== undefined) cambios.fonte = String(a.fonte)
      if (Object.keys(cambios).length === 0) return actual
      ui.setTipografia({ ...actual, ...cambios })
      return ui.getTipografia()
    },

    usuario_avisar(a) {
      ui.avisar(nonBaleiro(a.mensaxe, 'a mensaxe'), a.tipo === 'aviso' ? 'aviso' : 'info')
      return { amosado: true }
    },

    async usuario_preguntar(a) {
      const pregunta = nonBaleiro(a.pregunta, 'a pregunta')
      const opcions = Array.isArray(a.opcions) ? a.opcions.map(String).filter(Boolean) : []
      const resposta = await ui.preguntar(pregunta, opcions)
      if (resposta === null || resposta === undefined) return { resposta: null, nota: 'o usuario non respondeu' }
      return { resposta }
    },
  }

  // Coas actas activas as ferramentas propias do cronograma dan un erro claro.
  for (const nome of ['curso_ler', 'curso_configurar', 'tarefa_engadir', 'tarefa_editar', 'tarefa_eliminar', 'tarefa_mover', 'tarefa_abrir', 'curso_logotipo', 'tarefa_imaxe']) {
    frontend[nome] = soCronograma(frontend[nome])
  }
  Object.assign(frontend, crearFerramentasActas({ informes, ui }))

  const posDeBackend = (nome) => {
    // Gardar imaxes nunha plantilla de exemplo crea unha copia propia: o selector cámbiase.
    if (nome === 'plantilla_gardar' || nome === 'plantilla_eliminar' || nome === 'plantilla_duplicar' || nome === 'adxunto_gardar_en_plantilla') return ui.refrescarPlantillas?.()
    if (nome === 'opcions_cambiar') return ui.recargarOpcions?.()
    return undefined
  }

  // executar corre unha ferramenta e devolve sempre {contido, erro}: un fallo
  // non rompe o bucle, vai á IA para que o vexa.
  async function executar(nome, args) {
    // PRIVACIDADE: coa ferramenta activa nun formulario en modo PRODUCIÓN (datos
    // reais) a IA non toca nada: só pode marchar a outra ferramenta. O panel
    // nin se amosa neste caso (main.js); isto é o segundo seguro.
    if (informes.ferramenta?.enProduccion && nome !== 'ferramenta_activa') {
      return { contido: 'a ferramenta activa é un formulario en modo PRODUCIÓN: o asistente non ten acceso a el nin aos seus datos. Só o usuario pode pasalo a deseño.', erro: true }
    }
    try {
      if (frontend[nome]) {
        const v = await frontend[nome](args || {})
        return { contido: limitar(typeof v === 'string' ? v : JSON.stringify(v, null, 2)), erro: false }
      }
      const r = await api.IAChamarFerramenta(nome, JSON.stringify(args || {}))
      if (!r.erro) { try { await posDeBackend(nome) } catch { /* a interface non debe romper o resultado */ } }
      return { contido: limitar(r.contido ?? ''), erro: !!r.erro, imaxes: r.imaxes || [] }
    } catch (err) {
      return { contido: String(err?.message || err), erro: true }
    }
  }

  return { executar, nomesFrontend: Object.keys(frontend) }
}

function limitar(s) {
  if (s.length <= TAMANO_MAX_RESULTADO) return s
  return s.slice(0, TAMANO_MAX_RESULTADO) + `\n[... resultado cortado: ${s.length - TAMANO_MAX_RESULTADO} caracteres máis ...]`
}
