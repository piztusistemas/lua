import { describe, it, expect, vi } from 'vitest'
import { crearFerramentasFormularioDatos, valorParaCampo, vistaDatos } from './ferramentas-formulario-datos.js'

const ESQUEMA = {
  id: 'notas', nome: 'Notas', modo: 'deseno',
  campos: [
    { id: 'grupo', tipo: 'texto', etiqueta: 'Grupo' },
    { id: 'nota', tipo: 'numero', etiqueta: 'Nota' },
    { id: 'data', tipo: 'data', etiqueta: 'Data' },
    { id: 'etapa', tipo: 'escolla', etiqueta: 'Etapa', opcions: ['ESO', 'Bacharelato'] },
    { id: 'ok', tipo: 'verificacion', etiqueta: 'Aprobado' },
    { id: 'foto', tipo: 'imaxe', etiqueta: 'Foto' },
    { id: 's', tipo: 'seccion', etiqueta: 'Alumnado' },
    { id: 'alumnos', tipo: 'taboa', etiqueta: 'Alumnos', columnas: [
      { id: 'nome', tipo: 'texto', etiqueta: 'Nome' },
      { id: 'exame', tipo: 'numero', etiqueta: 'Exame' },
      { id: 'pasa', tipo: 'verificacion', etiqueta: 'Pasa' },
    ] },
  ],
}

function montar(esquemaExtra = {}, planInicial = null, apiExtra = {}) {
  const estado = {
    esquema: { ...ESQUEMA, ...esquemaExtra },
    plan: planInicial || { formulario: 'notas', contexto: { centro: '', localidade: '', curso: '', departamento: '', responsable: '', cargo: '', logo: '' }, valores: {} },
    aplicados: 0,
  }
  const ui = { avisar: vi.fn(), preguntar: vi.fn(async () => 'Si') }
  const api = { IAChamarFerramenta: vi.fn(async () => ({ contido: '', erro: false })), ...apiExtra }
  const r = crearFerramentasFormularioDatos({
    esquema: () => estado.esquema, plan: () => estado.plan,
    aplicar: (novo) => { estado.plan = novo; estado.aplicados++ }, api, ui,
  })
  return { estado, ui, api, ...r }
}

describe('valorParaCampo', () => {
  const c = (tipo, extra = {}) => ({ id: 'x', tipo, etiqueta: 'X', ...extra })
  it('normaliza cada tipo como o faría o usuario', () => {
    expect(valorParaCampo(c('numero'), 7.5)).toBe('7.5')
    expect(valorParaCampo(c('numero'), '7,5')).toBe('7,5')
    expect(valorParaCampo(c('data'), '2026-03-15')).toBe('2026-03-15')
    expect(valorParaCampo(c('verificacion'), 'true')).toBe(true)
    expect(valorParaCampo(c('escolla', { opcions: ['a', 'b'] }), 'b')).toBe('b')
    expect(valorParaCampo(c('texto'), 42)).toBe('42')
    expect(valorParaCampo(c('texto'), null)).toBe('')
  })
  it('rexeita o que non vale co motivo', () => {
    expect(() => valorParaCampo(c('numero'), 'moito')).toThrow(/número/)
    expect(() => valorParaCampo(c('data'), '31/12/2026')).toThrow(/AAAA-MM-DD/)
    expect(() => valorParaCampo(c('data'), '2026-04-31')).toThrow(/data/) // non existe
    expect(() => valorParaCampo(c('escolla', { opcions: ['a'] }), 'z')).toThrow(/opcións/)
    expect(() => valorParaCampo(c('verificacion'), 'quizais')).toThrow(/true ou false/)
    expect(() => valorParaCampo(c('texto'), { a: 1 })).toThrow(/texto/)
    expect(() => valorParaCampo(c('imaxe'), 'data:image/png;base64,AAA')).toThrow(/imaxe/)
    expect(() => valorParaCampo(c('seccion'), 'x')).toThrow(/sección/)
  })
})

describe('formulario_datos_cambiar', () => {
  it('escribe valores validados e fusiona co que había', async () => {
    const { executar, estado } = montar({}, { formulario: 'notas', contexto: {}, valores: { grupo: '1º A' } })
    const r = await executar('formulario_datos_cambiar', { valores: { nota: 8, etapa: 'ESO', ok: true }, contexto: { centro: 'IES Ficticio' } })
    expect(r.erro).toBe(false)
    expect(estado.plan.valores).toEqual({ grupo: '1º A', nota: '8', etapa: 'ESO', ok: true })
    expect(estado.plan.contexto.centro).toBe('IES Ficticio')
    expect(estado.aplicados).toBe(1)
  })

  it('unha táboa substitúese enteira e valida cada cela', async () => {
    const { executar, estado } = montar()
    const filas = [{ nome: 'Ana', exame: 7.5, pasa: true }, { nome: 'Breixo', exame: '4', pasa: false }]
    expect((await executar('formulario_datos_cambiar', { valores: { alumnos: filas } })).erro).toBe(false)
    expect(estado.plan.valores.alumnos).toEqual([{ nome: 'Ana', exame: '7.5', pasa: true }, { nome: 'Breixo', exame: '4' }])
    const mala = await executar('formulario_datos_cambiar', { valores: { alumnos: [{ nome: 'X', inventada: 1 }] } })
    expect(mala.erro).toBe(true)
    expect(mala.contido).toMatch(/columna «inventada»/)
    expect(estado.plan.valores.alumnos.length).toBe(2) // non cambiou
  })

  it('se algo falla non cambia NADA (nin o que valía)', async () => {
    const { executar, estado } = montar()
    const r = await executar('formulario_datos_cambiar', { valores: { grupo: 'A', nota: 'moito' } })
    expect(r.erro).toBe(true)
    expect(r.contido).toMatch(/nota/)
    expect(estado.aplicados).toBe(0)
    expect(estado.plan.valores).toEqual({})
  })

  it('non inventa campos nin toca imaxes, seccións ou o logotipo', async () => {
    const { executar, estado } = montar()
    expect((await executar('formulario_datos_cambiar', { valores: { non_existe: 1 } })).contido).toMatch(/non existe ningún campo/)
    expect((await executar('formulario_datos_cambiar', { valores: { foto: 'data:image/png;base64,AAAA' } })).erro).toBe(true)
    expect((await executar('formulario_datos_cambiar', { valores: { s: 'x' } })).erro).toBe(true)
    expect((await executar('formulario_datos_cambiar', { contexto: { logo: 'data:image/png;base64,AAAA' } })).contido).toMatch(/logotipo/)
    expect((await executar('formulario_datos_cambiar', { contexto: { inventado: 'x' } })).erro).toBe(true)
    expect(estado.aplicados).toBe(0)
  })

  it('limpar baleira valores e datos do documento pero non o logotipo', async () => {
    const plan = { formulario: 'notas', contexto: { centro: 'Real', curso: '2025', logo: 'data:image/png;base64,AAAA' }, valores: { grupo: 'A' } }
    const { executar, estado } = montar({}, plan)
    expect((await executar('formulario_datos_cambiar', { limpar: true, valores: { nota: 5 } })).erro).toBe(false)
    expect(estado.plan.valores).toEqual({ nota: '5' })
    expect(estado.plan.contexto.centro).toBe('')
    expect(estado.plan.contexto.logo).toBe('data:image/png;base64,AAAA')
  })

  it('pide que se indique algo', async () => {
    const { executar } = montar()
    expect((await executar('formulario_datos_cambiar', {})).erro).toBe(true)
  })
})

describe('editar celas e filas', () => {
  const plan = () => ({ formulario: 'notas', contexto: {}, valores: { alumnos: [{ nome: 'Ana', exame: '7' }, { nome: 'Breixo', exame: '4' }, { nome: 'Carme' }] } })
  it('cambia e baleira celas soltas, validando', async () => {
    const { executar, estado } = montar({}, plan())
    const r = await executar('formulario_datos_cambiar', { celas: [{ campo: 'alumnos', fila: 2, columna: 'exame', valor: 5.5 }, { campo: 'alumnos', fila: 1, columna: 'nome', valor: null }] })
    expect(r.erro).toBe(false)
    expect(estado.plan.valores.alumnos).toEqual([{ exame: '7' }, { nome: 'Breixo', exame: '5.5' }, { nome: 'Carme' }])
    expect((await executar('formulario_datos_cambiar', { celas: [{ campo: 'alumnos', fila: 9, columna: 'nome', valor: 'x' }] })).contido).toMatch(/fila 9/)
    expect((await executar('formulario_datos_cambiar', { celas: [{ campo: 'alumnos', fila: 1, columna: 'exame', valor: 'moito' }] })).erro).toBe(true)
    expect((await executar('formulario_datos_cambiar', { celas: [{ campo: 'grupo', fila: 1, columna: 'x', valor: 1 }] })).contido).toMatch(/non é unha táboa/)
    expect(estado.aplicados).toBe(1)
  })
  it('quita filas polo seu número orixinal e non toca a orixinal se falla', async () => {
    const { executar, estado } = montar({}, plan())
    const orixinal = estado.plan.valores.alumnos
    expect((await executar('formulario_datos_cambiar', { quitar_filas: { alumnos: [1, 7] } })).erro).toBe(true)
    expect(estado.plan.valores.alumnos).toBe(orixinal)
    expect((await executar('formulario_datos_cambiar', { quitar_filas: { alumnos: [1, 3] } })).erro).toBe(false)
    expect(estado.plan.valores.alumnos).toEqual([{ nome: 'Breixo', exame: '4' }])
  })
})

describe('ler táboas grandes', () => {
  it('corta as táboas grandes e deixa lelas por tramos numerados', async () => {
    const alumnos = Array.from({ length: 100 }, (_, i) => ({ nome: `A${i + 1}` }))
    const { executar } = montar({}, { formulario: 'notas', contexto: {}, valores: { alumnos } })
    const v = JSON.parse((await executar('formulario_datos_ler', {})).contido)
    expect(v.valores.alumnos.length).toBe(40)
    expect(v.taboas.alumnos.filas_total).toBe(100)
    const t = JSON.parse((await executar('formulario_datos_ler', { campo: 'alumnos', desde: 41, cantidade: 10 })).contido)
    expect(t.valores.alumnos[0]).toEqual({ fila: 41, nome: 'A41' })
    expect(t.valores.alumnos.length).toBe(10)
    expect((await executar('formulario_datos_ler', { campo: 'inventado' })).erro).toBe(true)
  })
})

describe('marca de datos de proba', () => {
  it('escribir datos marca o documento; limpar sen datos novos quita a marca; un cambio que falla non a toca', async () => {
    const { executar, estado } = montar()
    await executar('formulario_datos_cambiar', { valores: { grupo: 'A' } })
    expect(estado.plan.datosProba).toBe(true)
    await executar('formulario_datos_cambiar', { valores: { nota: 'moito' } }) // falla
    expect(estado.plan.datosProba).toBe(true)
    await executar('formulario_datos_cambiar', { limpar: true })
    expect(estado.plan.datosProba).toBeUndefined()
    expect(estado.plan.valores).toEqual({})
    // Limpar e escribir na mesma chamada: hai datos de proba.
    await executar('formulario_datos_cambiar', { limpar: true, valores: { grupo: 'B' } })
    expect(estado.plan.datosProba).toBe(true)
  })
})

describe('formulario_datos_ler', () => {
  it('describe os campos e os datos, sen imaxes en base64', async () => {
    const plan = { formulario: 'notas', contexto: { centro: 'IES Ficticio', logo: 'data:image/png;base64,AAAA' }, valores: { grupo: '1º A', foto: 'data:image/png;base64,SEGREDO', alumnos: [{ nome: 'Ana' }] } }
    const { executar } = montar({}, plan)
    const r = await executar('formulario_datos_ler', {})
    expect(r.erro).toBe(false)
    expect(r.contido).not.toMatch(/SEGREDO|base64/)
    const v = JSON.parse(r.contido)
    expect(v.valores.grupo).toBe('1º A')
    expect(v.valores.foto).toMatch(/imaxe/)
    expect(v.contexto).toEqual({ centro: 'IES Ficticio' })
    expect(v.logotipo).toBe(true)
    expect(v.campos.find((c) => c.id === 'alumnos').columnas.map((c) => c.id)).toEqual(['nome', 'exame', 'pasa'])
    expect(vistaDatos(ESQUEMA, plan).campos.length).toBe(ESQUEMA.campos.length)
  })
})

describe('PRIVACIDADE: modo produción', () => {
  it('nega TODAS as ferramentas (e non le nin escribe nada)', async () => {
    const plan = { formulario: 'notas', contexto: { centro: 'Centro real' }, valores: { grupo: 'DATO PRIVADO' } }
    const { executar, estado, ui } = montar({ modo: 'produccion' }, plan)
    for (const nome of ['formulario_datos_ler', 'formulario_datos_cambiar', 'usuario_preguntar', 'usuario_avisar', 'calquera_outra']) {
      const r = await executar(nome, { valores: { grupo: 'x' }, pregunta: 'p', mensaxe: 'm' })
      expect(r.erro).toBe(true)
      expect(r.contido).toMatch(/PRODUCIÓN/)
      expect(r.contido).not.toMatch(/DATO PRIVADO|Centro real/)
    }
    expect(estado.aplicados).toBe(0)
    expect(estado.plan.valores.grupo).toBe('DATO PRIVADO')
    expect(ui.avisar).not.toHaveBeenCalled()
    expect(ui.preguntar).not.toHaveBeenCalled()
  })

  it('o modo mírase en CADA chamada: se pasa a produción co panel aberto, as seguintes fallan', async () => {
    const { executar, estado } = montar()
    expect((await executar('formulario_datos_ler', {})).erro).toBe(false)
    estado.esquema = { ...estado.esquema, modo: 'produccion' }
    expect((await executar('formulario_datos_ler', {})).erro).toBe(true)
  })

  it('un formulario sen modo (de antes) traballa coma deseño', async () => {
    const { executar } = montar({ modo: undefined })
    expect((await executar('formulario_datos_ler', {})).erro).toBe(false)
  })
})

describe('adxuntos (documentos de referencia)', () => {
  it('adxunto_ler, adxunto_ver, adxunto_extraer_imaxes e adxuntos_listar pasan a Go', async () => {
    const api = { IAChamarFerramenta: vi.fn(async (nome) => ({ contido: `resposta de ${nome}`, erro: false, imaxes: ['img1'] })) }
    const { executar } = montar({}, null, api)
    for (const nome of ['adxuntos_listar', 'adxunto_ler', 'adxunto_ver', 'adxunto_extraer_imaxes']) {
      const r = await executar(nome, { id: 'a1' })
      expect(api.IAChamarFerramenta).toHaveBeenCalledWith(nome, JSON.stringify({ id: 'a1' }))
      expect(r).toEqual({ contido: `resposta de ${nome}`, erro: false, imaxes: ['img1'] })
    }
  })
  it('as ferramentas boveda_* pasan a Go (consulta da bóveda de Obsidian)', async () => {
    const api = { IAChamarFerramenta: vi.fn(async (nome) => ({ contido: `resposta de ${nome}`, erro: false })) }
    const { executar } = montar({}, null, api)
    for (const nome of ['boveda_info', 'boveda_buscar', 'boveda_ler', 'boveda_enlaces', 'boveda_listar', 'boveda_pendentes', 'boveda_lint', 'boveda_escribir']) {
      const r = await executar(nome, { ruta: 'wiki/x.md' })
      expect(r.erro).toBe(false)
      expect(api.IAChamarFerramenta).toHaveBeenCalledWith(nome, JSON.stringify({ ruta: 'wiki/x.md' }))
    }
  })
  it('non deixa gardar nin eliminar adxuntos, nin buscar na web', async () => {
    const { executar, api } = montar()
    for (const nome of ['adxunto_eliminar', 'adxunto_gardar_en_plantilla', 'web_buscar', 'web_consultar']) {
      const r = await executar(nome, {})
      expect(r.erro).toBe(true)
      expect(r.contido).toMatch(/non está dispoñible/)
    }
    expect(api.IAChamarFerramenta).not.toHaveBeenCalled()
  })
  it('en produción tamén nega os adxuntos, sen chamar a Go', async () => {
    const { executar, api } = montar({ modo: 'produccion' })
    const r = await executar('adxunto_ler', { id: 'a1' })
    expect(r.erro).toBe(true)
    expect(r.contido).toMatch(/PRODUCIÓN/)
    expect(api.IAChamarFerramenta).not.toHaveBeenCalled()
  })
})

describe('outras ferramentas', () => {
  it('non ten ferramentas do deseñador, do curso nin da web', async () => {
    const { executar, nomesFrontend } = montar()
    expect([...nomesFrontend].sort()).toEqual(['formulario_datos_cambiar', 'formulario_datos_ler', 'formulario_xerar', 'usuario_avisar', 'usuario_preguntar'])
    for (const nome of ['formulario_cambiar', 'formulario_ler', 'curso_ler', 'web_consultar', 'informe_xerar']) {
      const r = await executar(nome, {})
      expect(r.erro).toBe(true)
      expect(r.contido).toMatch(/non está dispoñible/)
    }
  })
  it('avisar e preguntar pasan pola interface', async () => {
    const { executar, ui } = montar()
    expect((await executar('usuario_avisar', { mensaxe: 'Feito', tipo: 'aviso' })).erro).toBe(false)
    expect(ui.avisar).toHaveBeenCalledWith('Feito', 'aviso')
    const r = await executar('usuario_preguntar', { pregunta: 'Cantas filas?', opcions: ['5', '10'] })
    expect(JSON.parse(r.contido)).toEqual({ resposta: 'Si' })
  })
})
