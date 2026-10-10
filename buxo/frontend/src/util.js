// Textos que dependen do estado de Buxo (separados para poder probalos).
import { t } from './i18n.js'

export function textoEstado(e) {
  if (!e.iaConfigurada) return t('ui.estado.sen_ia', 'Configura unha clave de IA en Lúa (Opcións) para construír a bóveda.')
  if (e.fontes === 0) return t('ui.estado.sen_fontes', 'Aínda non hai PDF convertidos: engade ficheiros primeiro.')
  return t('ui.estado.listo', 'A IA le os documentos convertidos e constrúe a wiki')
}

export function textoAxudaURLs(e) {
  return e.iaConfigurada
    ? t('ui.urls.axuda_ia', 'Unha URL que apunta a un PDF descárgase directamente; se é unha páxina, a IA busca nela os PDF que valen para o proxecto e aplica a instrución de filtrado, se a escribes.')
    : t('ui.urls.axuda_sen_ia', 'Sen IA só se descargan as URL que apuntan directamente a un PDF.')
}

// resumoProgreso: contas do conxunto de tarefas. Unha tarefa feita ou con erro
// xa percorreu o seu camiño (o erro non a deixa en suspenso); a que corre
// aporta unha fracción que crece cos pasos sen chegar nunca a 1.
export function fraccionTarefa(t) {
  if (t.estado === 'feita' || t.estado === 'erro') return 1
  if (t.estado === 'executando') return Math.min(0.9, t.pasos / (t.pasos + 6))
  return 0
}

export function resumoProgreso(tarefas) {
  const total = tarefas.length
  const rematadas = tarefas.filter((t) => t.estado === 'feita' || t.estado === 'erro').length
  const erros = tarefas.filter((t) => t.estado === 'erro').length
  const suma = tarefas.reduce((s, t) => s + fraccionTarefa(t), 0)
  return { total, rematadas, erros, pct: total ? Math.round((suma / total) * 100) : 0 }
}
