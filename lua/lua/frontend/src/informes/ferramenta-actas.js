// A ferramenta "Actas do departamento": o que o controlador (controlador.js)
// precisa saber dela - o modelo de datos, o editor e a petición ao backend.
import { createActasEditor } from './actas-editor.js'
import {
  novoPlanActas, validarDatas, lerFicheiroActas, escribirFicheiroActas, nomeFicheiroSuxeridoActas,
  totalActas, ambitosDisponibles,
} from './actas-modelo.js'

export const FERRAMENTA_ACTAS = {
  id: 'actas',
  novoPlan: novoPlanActas,
  lerFicheiro: lerFicheiroActas,
  escribirFicheiro: escribirFicheiroActas,
  nomeFicheiroSuxerido: nomeFicheiroSuxeridoActas,
  crearEditor: createActasEditor,
  // Que se xera e imprime: o curso enteiro (todas as actas) ou unha acta.
  ambitoPorDefecto: 'curso',
  ambitosDisponibles,
  validarDatas: (plan) => validarDatas(plan.config.startDate, plan.config.endDate),
  haiContido(plan) {
    const c = plan.config
    return totalActas(plan) > 0 || c.subjects.length > 0 || c.members.length > 0
      || !!(c.schoolName || c.location || c.department || c.headOfDepartment || c.secretary || c.schoolLogo || c.signatoryRole)
  },
  descricionCurso: (c, t) =>
    `${c.academicYear || '—'} · ${c.config.department || c.config.schoolName || t('informes.sen.centro', 'sen nome de centro')} (${totalActas(c)} ${t('actas.actas.abrev', 'actas')})`,
  // O que se lle manda ao backend (XerarInformeCronogramaRequest en Go).
  peticion: (plan, { tipografia, nomeBase, ambito }) => ({ ferramenta: 'actas', actas: plan, ambito: ambito || 'curso', tipografia, nomeBase, plan: {} }),
}
