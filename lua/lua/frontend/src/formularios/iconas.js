// Iconas que pode escoller o usuario para o seu formulario (a columna
// esquerda e o deseñador). Cada unha é o interior dun SVG 24x24 de liña.
// A clave gárdase no esquema (campo `icona`); unha clave descoñecida cae na
// icona «formulario».

const ICONAS = {
  formulario: '<path d="M6.5 3h8l4 4v13.2a.8.8 0 0 1-.8.8H6.5a.8.8 0 0 1-.8-.8V3.8a.8.8 0 0 1 .8-.8z"/><path d="M14.5 3v4h4"/><path d="M8.5 11.5h7M8.5 14.5h7M8.5 17.5h4"/>',
  lista: '<path d="M9 6.5h11M9 12h11M9 17.5h11"/><path d="M4 6.5h.01M4 12h.01M4 17.5h.01" stroke-width="2.6"/>',
  calendario: '<rect x="3.5" y="5" width="17" height="15.5" rx="2.2"/><path d="M3.5 10h17M8 3v4M16 3v4"/>',
  persoas: '<circle cx="9" cy="8.5" r="3.2"/><path d="M3.5 19.5c.4-3.3 2.6-5.2 5.5-5.2s5.1 1.9 5.5 5.2"/><path d="M16 5.6a3 3 0 0 1 0 5.8M17.5 14.6c1.8.5 3 2.1 3.3 4.9"/>',
  libro: '<path d="M4.5 5.5c2.8-.9 5.4-.6 7.5 1v13c-2.1-1.6-4.7-1.9-7.5-1z"/><path d="M19.5 5.5c-2.8-.9-5.4-.6-7.5 1v13c2.1-1.6 4.7-1.9 7.5-1z"/>',
  estrela: '<path d="m12 3.6 2.6 5.4 5.9.8-4.3 4.1 1 5.9L12 17l-5.2 2.8 1-5.9L3.5 9.8l5.9-.8z"/>',
  correo: '<rect x="3.5" y="5.5" width="17" height="13" rx="2"/><path d="m4 7 8 6 8-6"/>',
  aviso: '<path d="M12 4 3 19.5h18z"/><path d="M12 10v4.5M12 17.2h.01" stroke-width="2"/>',
  grafico: '<path d="M4 19.5h16"/><path d="M7 19.5V12M12 19.5V6.5M17 19.5v-9"/>',
  lapis: '<path d="M4 20l1-4L16.5 4.5a1.8 1.8 0 0 1 2.5 0l.5.5a1.8 1.8 0 0 1 0 2.5L8 19z"/><path d="m14.5 6.5 3 3"/>',
  maleta: '<rect x="3.5" y="7.5" width="17" height="12" rx="2"/><path d="M9 7.5V6a1.5 1.5 0 0 1 1.5-1.5h3A1.5 1.5 0 0 1 15 6v1.5M3.5 13h17"/>',
  reloxo: '<circle cx="12" cy="12" r="8.5"/><path d="M12 7v5l3.2 2"/>',
  camara: '<path d="M4 8h3l1.5-2.5h7L17 8h3v11H4z"/><circle cx="12" cy="13" r="3.4"/>',
  corazon: '<path d="M12 20s-7.5-4.6-7.5-10.2A4.3 4.3 0 0 1 12 7.3a4.3 4.3 0 0 1 7.5 2.5C19.5 15.4 12 20 12 20z"/>',
  bandeira: '<path d="M5.5 21V4"/><path d="M5.5 5c3.5-1.6 6.5 1.6 10 0 1.2-.5 2.2-.5 3 0v8c-.8-.5-1.8-.5-3 0-3.5 1.6-6.5-1.6-10 0"/>',
  mochila: '<path d="M8 9.5a4 4 0 0 1 8 0"/><rect x="5.5" y="9.5" width="13" height="11" rx="3"/><path d="M9 15h6"/>',
}

export const CLAVES_ICONA = Object.keys(ICONAS)

export const NOME_ICONA = {
  formulario: 'Documento', lista: 'Lista', calendario: 'Calendario', persoas: 'Persoas', libro: 'Libro', estrela: 'Estrela',
  correo: 'Correo', aviso: 'Aviso', grafico: 'Gráfico', lapis: 'Lapis', maleta: 'Maleta', reloxo: 'Reloxo',
  camara: 'Cámara', corazon: 'Corazón', bandeira: 'Bandeira', mochila: 'Mochila',
}

// svgIcona devolve o SVG completo da icona (só para innerHTML: o contido é noso, nunca do usuario).
export function svgIcona(clave) {
  const corpo = ICONAS[clave] || ICONAS.formulario
  return `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${corpo}</svg>`
}

// Iconas dos tipos de campo (paleta e tarxetas do deseñador), en 20x20.
const TIPO = {
  texto: '<path d="M4 6h12M10 6v9M7.5 15h5"/>',
  textolongo: '<path d="M4 5.5h12M4 9h12M4 12.5h12M4 16h7"/>',
  numero: '<path d="M8 4 6.5 16M13.5 4 12 16M4.5 8h12M3.5 12h12"/>',
  data: '<rect x="3.5" y="4.5" width="13" height="12" rx="1.8"/><path d="M3.5 8.5h13M7 3v3M13 3v3"/>',
  escolla: '<rect x="3.5" y="5" width="13" height="10" rx="1.8"/><path d="m7.5 9.5 2.5 2.5 2.5-2.5"/>',
  verificacion: '<rect x="4" y="4" width="12" height="12" rx="2"/><path d="m7.2 10.2 2.2 2.2 3.6-4.4"/>',
  imaxe: '<rect x="3.5" y="4.5" width="13" height="11" rx="1.8"/><circle cx="7.5" cy="8.5" r="1.3"/><path d="m4 14 4-3.5 3 2.5 2.5-2 3 3"/>',
  taboa: '<rect x="3.5" y="4.5" width="13" height="11" rx="1.6"/><path d="M3.5 8.5h13M3.5 12h13M9 4.5v11"/>',
  problemas: '<path d="M4 5.5h5M4 9h3"/><path d="M11 5.5h5"/><path d="M4.5 15.5 7 12.5l1.6 2.2L11 11"/><path d="M12.5 12.5h4M14.5 10.5v4"/>',
  seccion: '<path d="M4 6h12"/><path d="M4 11h12M4 14.5h8" opacity=".45"/>',
}

export function svgTipo(tipo) {
  return `<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${TIPO[tipo] || TIPO.texto}</svg>`
}

// Cores da icona de cada formulario, para distinguilos dunha ollada na columna
// esquerda. Ningunha é vermella nin verde: esas son as do modo (deseño /
// produción), que vai nun punto á parte. A clave gárdase no esquema (`cor`).
const CORES = {
  azul: 'oklch(52% 0.17 255)',
  laranxa: 'oklch(62% 0.17 55)',
  violeta: 'oklch(50% 0.2 300)',
  ciano: 'oklch(56% 0.11 215)',
  rosa: 'oklch(56% 0.2 350)',
  ambar: 'oklch(66% 0.15 80)',
  indigo: 'oklch(45% 0.17 275)',
  marron: 'oklch(48% 0.08 50)',
}

export const CLAVES_COR = Object.keys(CORES)

export const NOME_COR = {
  azul: 'Azul', laranxa: 'Laranxa', violeta: 'Violeta', ciano: 'Ciano',
  rosa: 'Rosa', ambar: 'Ámbar', indigo: 'Índigo', marron: 'Marrón',
}

export const valorCor = (clave) => CORES[clave] || CORES.azul

const corValida = (f) => (CORES[f?.cor] ? f.cor : '')

// coresFormularios devolve {id: clave de cor} para todos os esquemas. Os que
// xa teñen cor quedan coa súa; os de antes (sen cor) reparten, por orde de id,
// as cores que non usa ningún dos outros. Así ningún cambia de cor cando nace
// un formulario novo (colle unha cor libre, ver aparenciaLibre) nin cando un
// dos de antes garda a cor que xa amosaba.
export function coresFormularios(esquemas) {
  const out = {}
  const sen = []
  for (const f of esquemas || []) {
    if (corValida(f)) out[f.id] = f.cor
    else sen.push(f.id)
  }
  const usadas = new Set(Object.values(out))
  const libres = CLAVES_COR.filter((k) => !usadas.has(k))
  const quendas = libres.length ? libres : CLAVES_COR
  sen.sort().forEach((id, i) => { out[id] = quendas[i % quendas.length] })
  return out
}

// aparenciaLibre escolle a icona e a cor dun formulario novo: as menos usadas
// polos que xa hai (a primeira, se empatan).
export function aparenciaLibre(esquemas) {
  const lista = esquemas || []
  const cores = coresFormularios(lista)
  const menosUsada = (claves, usadas) => {
    const n = Object.fromEntries(claves.map((k) => [k, 0]))
    for (const k of usadas) if (k in n) n[k]++
    return claves.reduce((m, k) => (n[k] < n[m] ? k : m), claves[0])
  }
  return {
    icona: menosUsada(CLAVES_ICONA, lista.map((f) => f.icona || 'formulario')),
    cor: menosUsada(CLAVES_COR, Object.values(cores)),
  }
}
