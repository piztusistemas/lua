// A «ferramenta» dun formulario propio: o que o controlador (informes/controlador.js)
// precisa saber dela. Cada formulario da biblioteca ten a súa (id «form:<id>»),
// co seu ficheiro, o seu borrador, o seu desfacer e o seu resultado, coma o
// cronograma e as actas.
import { createFormularioEditor } from './editor.js'
import { xeracionDe } from './problemas-modelo.js'
import {
  novoPlanFormulario, lerFicheiroFormulario, escribirFicheiroFormulario, nomeFicheiroSuxeridoFormulario, haiContidoFormulario,
} from './modelo.js'

// O contexto (centro, curso, quen asina...) lémbrase entre formularios: un
// documento novo sae co do último, que case sempre é o mesmo. Só entre
// formularios en PRODUCIÓN: un formulario en deseño (datos ficticios, e a IA
// pode velos) nin se enche co contexto real nin o pisa co seu.
const CLAVE_CONTEXTO = 'lua.formularios.contexto'

function contextoLembrado() {
  try {
    const c = JSON.parse(localStorage.getItem(CLAVE_CONTEXTO) || 'null')
    return c && typeof c === 'object' ? c : {}
  } catch {
    return {}
  }
}

function lembrarContexto(ctx) {
  try { localStorage.setItem(CLAVE_CONTEXTO, JSON.stringify(ctx)) } catch { /* sen persistencia */ }
}

// crearFerramentaFormulario devolve a ferramenta dun formulario. O esquema pode
// cambiar (o deseñador gárdao, ou o modo) sen recrear o controlador: actualizarEsquema.
//   aoDeseñar(id)       abre o deseñador
//   aoCambiarModo(id,m) pide pasar o formulario a 'deseno' ou 'produccion'
//   aoBorrarDatosProba(id) pide borrar os datos de proba da IA (en produción)
//   asistente           {api, idioma} do asistente de datos (só en modo deseño)
//   avisar(m, tipo)     mensaxes do asistente
export function crearFerramentaFormulario(esquemaInicial, { aoDeseñar, aoCambiarModo, aoBorrarDatosProba, asistente, avisar, problemas } = {}) {
  let esquema = esquemaInicial
  const f = {
    id: 'form:' + esquemaInicial.id,
    esFormulario: true,
    get esquema() { return esquema },
    actualizarEsquema(novo) { esquema = novo },
    get enProduccion() { return esquema.modo === 'produccion' },
    novoPlan: () => novoPlanFormulario(esquema.id, esquema.modo === 'produccion' ? contextoLembrado() : {}),
    lerFicheiro: (texto) => lerFicheiroFormulario(texto, esquema.id, esquema.nome),
    escribirFicheiro: escribirFicheiroFormulario,
    nomeFicheiroSuxerido: (plan) => nomeFicheiroSuxeridoFormulario(plan, esquema),
    crearEditor: (container, opts) => createFormularioEditor(container, {
      ...opts, obterEsquema: () => esquema,
      lembrarContexto: (ctx) => { if (esquema.modo === 'produccion') lembrarContexto(ctx) },
      aoDeseñar: () => aoDeseñar?.(esquema.id),
      aoCambiarModo: (modo) => aoCambiarModo?.(esquema.id, modo),
      aoBorrarDatosProba: () => aoBorrarDatosProba?.(esquema.id),
      asistente, avisar, problemas,
    }),
    validarDatas: () => '',
    haiContido: haiContidoFormulario,
    descricionCurso: (c) => c.formulario,
    // O que se lle manda ao backend (XerarInformeCronogramaRequest en Go): o
    // esquema vai xunto cos datos, así a vista previa non depende de que
    // estea gardado.
    peticion: (plan, { tipografia, nomeBase }) => ({
      ferramenta: 'formulario',
      formulario: { formulario: esquema, datos: { formulario: esquema.id, contexto: plan.contexto, valores: plan.valores, senReferencias: !!plan.senReferencias, xeracion: xeracionDe(plan) } },
      tipografia,
      nomeBase,
      plan: {},
    }),
  }
  return f
}
