// A ferramenta "Cronograma anual": o que o controlador (controlador.js) precisa
// saber dela - o modelo de datos, o editor e a petición ao backend.
import { createCronogramaEditor } from './cronograma-editor.js'
import {
  novoPlan, validarDatas, lerFicheiro, escribirFicheiro, nomeFicheiroSuxerido, totalTarefas,
} from './cronograma-modelo.js'

export const FERRAMENTA_CRONOGRAMA = {
  id: 'cronograma',
  novoPlan,
  lerFicheiro,
  escribirFicheiro,
  nomeFicheiroSuxerido,
  crearEditor: createCronogramaEditor,
  validarDatas: (plan) => validarDatas(plan.config.startDate, plan.config.endDate),
  haiContido(plan) {
    const c = plan.config
    return totalTarefas(plan) > 0 || !!(c.schoolName || c.location || c.department || c.headOfDepartment || c.schoolLogo || c.introText || c.signatoryRole)
  },
  descricionCurso: (c, t) =>
    `${c.academicYear || '—'} · ${c.config.schoolName || t('informes.sen.centro', 'sen nome de centro')} (${totalTarefas(c)} ${t('informes.tarefas.abrev', 'tarefas')})`,
  // O que se lle manda ao backend (XerarInformeCronogramaRequest en Go).
  peticion: (plan, { tipografia, nomeBase }) => ({ ferramenta: 'cronograma', plan, tipografia, nomeBase }),
}
