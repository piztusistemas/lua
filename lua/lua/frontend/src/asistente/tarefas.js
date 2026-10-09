// As tarefas que o asistente sabe facer: fichas coa instrución que se lle manda
// á IA. Hai unhas de fábrica (TAREFAS_BASE) e as que engada o propio usuario
// (persistidas en localStorage, unha preferencia local coma o divisor do
// editor). O texto que o usuario escriba na caixa de instrucións engádese á
// ficha ao executala.

export const TAREFAS_BASE = [
  {
    id: 'ortografia', icona: '✓', clave: 'asistente.tarefa.ortografia',
    titulo: 'Corrixir ortografía',
    instrucion: 'Le o curso enteiro (datos do centro, introdución e todas as tarefas) e corrixe as faltas de ortografía, gramática, puntuación e tipografía. Cambia só o necesario, sen alterar o sentido, o ton nin o formato Markdown (nin as imaxes). Ao rematar, resume que cambiaches.',
  },
  {
    id: 'redaccion', icona: '✎', clave: 'asistente.tarefa.redaccion',
    titulo: 'Mellorar a redacción',
    instrucion: 'Mellora a redacción das descricións das tarefas e da introdución: máis clara, coherente e profesional, sen cambiar datos, nomes nin cifras e sen inventar contido novo. Respecta o formato Markdown e as imaxes. Ao rematar, resume os cambios.',
  },
  {
    id: 'introducion', icona: '¶', clave: 'asistente.tarefa.introducion',
    titulo: 'Completar a introdución',
    instrucion: 'Le o curso e a plantilla activa (plantilla_ler) e redacta ou completa o texto da introdución do informe, adaptado ao centro, ao curso e ás actuacións do cronograma. Se xa hai texto, respéctao e complétao. Gárdao no curso co campo introducion de curso_configurar e móstrame o resultado.',
  },
  {
    id: 'lexislacion', icona: '§', clave: 'asistente.tarefa.lexislacion',
    titulo: 'Buscar lexislación',
    instrucion: 'Le as tarefas do curso e busca na web a lexislación educativa vixente que sustenta as actuacións (estatal e da Xunta de Galicia: LOMLOE, decretos de currículo e de orientación, normas do DOG e do BOE...). Consulta as fontes oficiais. Preséntame unha lista con cada norma (nome exacto, data e enderezo) e a que tarefas se aplica, e despois pregúntame se quero engadir a referencia normativa ás descricións das tarefas. Non inventes números de decreto: se non o comprobas nunha fonte, dío.',
  },
  {
    id: 'formato', icona: '✦', clave: 'asistente.tarefa.formato',
    titulo: 'Mellorar formato e estética',
    instrucion: 'Revisa o formato e a estética do documento: estrutura con Markdown (títulos ##, listas, negritas) as descricións longas ou pouco claras, asigna cores coherentes ás tarefas segundo o tipo de actuación, ordena as tarefas dentro de cada mes se procede e revisa a introdución. Non cambies o contido nin os datos. Despois xera o informe e comproba os avisos.',
  },
  {
    id: 'plantilla', icona: '▤', clave: 'asistente.tarefa.plantilla',
    titulo: 'Mellorar a plantilla',
    instrucion: 'Le a plantilla activa (plantilla_ler) e revisa a súa maqueta: cabeceira, pé, portada, sinatura e introdución. Propóñeme melloras estéticas e de contido usando variables ({{CENTRO}}, {{CURSO}}, {{XEFE}}...) e, se estou de acordo, aplícaas con plantilla_gardar e comproba que compila con plantilla_comprobar.',
  },
  {
    id: 'revisar', icona: '☰', clave: 'asistente.tarefa.revisar',
    titulo: 'Revisar o curso',
    instrucion: 'Revisa o curso e dime o que falta ou é incoherente: datos do centro sen encher, meses sen tarefas, tarefas sen descrición, títulos repetidos, tarefas fóra de rango, datas... Non cambies nada: dáme unha lista priorizada de melloras.',
  },
  {
    id: 'xerar', icona: '▶', clave: 'asistente.tarefa.xerar',
    titulo: 'Xerar e comprobar o informe',
    instrucion: 'Xera o informe co compilador actual e revisa os avisos ou erros. Se hai algún problema (LaTeX, imaxes, plantilla, LaTeX ausente), diagnostícao con sistema_estado se fai falta e propón ou aplica a solución.',
  },
]

// As fichas do asistente da PANTALLA DE EDICIÓN DUNHA TAREFA: traballan só
// sobre o borrador desa tarefa (ferramentas editor_ler/editor_cambiar).
export const TAREFAS_TAREFA = [
  {
    id: 'ortografia', icona: '✓', clave: 'asistente.tt.ortografia',
    titulo: 'Corrixir ortografía',
    instrucion: 'Le a tarefa aberta (editor_ler) e corrixe as faltas de ortografía, gramática, puntuación e tipografía do título e da descrición. Cambia só o necesario, sen alterar o sentido, o ton, as marcas Markdown nin as imaxes. Aplica o resultado co editor_cambiar e resume o que cambiaches.',
  },
  {
    id: 'redaccion', icona: '✎', clave: 'asistente.tt.redaccion',
    titulo: 'Mellorar a redacción',
    instrucion: 'Le a tarefa aberta (editor_ler) e mellora a redacción da descrición: máis clara, precisa e profesional, sen cambiar datos, nomes nin cifras e sen inventar contido novo. Respecta o formato Markdown e as imaxes. Aplícao co editor_cambiar e resume os cambios.',
  },
  {
    id: 'formato', icona: '✦', clave: 'asistente.tt.formato',
    titulo: 'Mellorar o formato',
    instrucion: 'Le a tarefa aberta (editor_ler) e mellora o formato da descrición co Markdown admitido: estrutura en parágrafos curtos, listas onde haxa enumeracións, negrita para os datos clave (datas, destinatarios, responsables) e subtítulos ## se a descrición é longa. Non cambies o contido nin as imaxes. Aplícao co editor_cambiar.',
  },
  {
    id: 'lei', icona: '§', clave: 'asistente.tt.lei',
    titulo: 'Referencias legais',
    instrucion: 'Le a tarefa aberta (editor_ler) e busca na web a normativa que a sustenta, estatal e da Xunta de Galicia (BOE, DOG, edu.xunta.gal): usa web_buscar con site:boe.es, site:xunta.gal ou site:edu.xunta.gal e le as fontes con web_consultar. Engade ao final da descrición unha sección "## Referencias normativas" cunha lista (nome exacto da norma, número/data de publicación e ligazón), só coas normas que comprobases nunha fonte. Se non atopas nada seguro, dío no chat en vez de inventar.',
  },
  {
    id: 'completar', icona: '¶', clave: 'asistente.tt.completar',
    titulo: 'Completar a descrición',
    instrucion: 'Le a tarefa aberta (editor_ler). Se a descrición está baleira ou é moi curta, redacta unha descrición axeitada a partir do título (obxectivo, destinatarios, temporalización e responsables, só no que se poida deducir sen inventar datos concretos; pregunta co usuario_preguntar se falta información importante). Se xa ten texto, complétao respectando o que hai. Aplícao co editor_cambiar.',
  },
  {
    id: 'conciso', icona: '↧', clave: 'asistente.tt.conciso',
    titulo: 'Facela máis concisa',
    instrucion: 'Le a tarefa aberta (editor_ler) e resume a descrición para que ocupe menos sen perder datos importantes (datas, cifras, responsables, referencias). Conserva o formato Markdown e as imaxes. Aplícao co editor_cambiar e dime canto acurtaches.',
  },
]

// As fichas do asistente do EDITOR DE PLANTILLAS: traballan só sobre o borrador
// da plantilla aberta (plantilla_editor_ler / plantilla_editor_cambiar) e miran o
// resultado con plantilla_previsualizar. Se non se di o formato, tócanse o PDF
// (LaTeX) e Word/OpenDocument (ofimatica).
export const TAREFAS_PLANTILLA = [
  {
    id: 'crear', icona: '✨', clave: 'asistente.tp.crear',
    titulo: 'Crear ou renovar a plantilla',
    instrucion: 'Constrúe ou renova a plantilla aberta a partir do que che dea: a DESCRICIÓN que escriba na caixa, unha URL de referencia (le a páxina con web_consultar e toma dela o estilo e os datos do centro), documentos modelo adxuntos (PDF, .tex, .md, .txt, Word, imaxes: adxunto_ler / adxunto_ver) ou un papel timbrado. Le antes o borrador (plantilla_editor_ler): se xa ten LaTeX escrito, tómao coma punto de partida en vez de empezar de cero e non quites variables nin imaxes que xa haxa. Completa o nome e a descrición, o LaTeX da maqueta (con {{CORPO}}), a portada, a cabeceira e o pé, as variables dos datos fixos (teléfono, correo...) e o deseño de Word/OpenDocument (ofimatica), pon o logotipo se hai adxunto (adxunto_gardar_en_plantilla) e comproba o resultado con plantilla_previsualizar en pdf e docx. Se non me deron ningunha descrición, adxunto nin URL, pregúntame que plantilla quero (usuario_preguntar).',
  },
  {
    id: 'portada', icona: '▣', clave: 'asistente.tp.portada',
    titulo: 'Modificar a portada',
    instrucion: 'Le a plantilla aberta (plantilla_editor_ler) e mellora a PORTADA: máis elegante e ben aliñada, co logotipo do curso (\\logocentro en LaTeX, @logo en Word/ODT), o título, o centro, o departamento, o curso, a localidade e a data mediante variables. Cambia a "portada" (LaTeX, coa maqueta levando {{PORTADA}}) e a portada de "ofimatica". Non quites imaxes nin datos que xa haxa. Mira o resultado con plantilla_previsualizar (páxina 1, en pdf e en docx) e corrixe ata que quede ben.',
  },
  {
    id: 'letra', icona: 'Aa', clave: 'asistente.tp.letra',
    titulo: 'Cambiar a letra',
    instrucion: 'Le a plantilla aberta (plantilla_editor_ler) e cambia o TIPO e o TAMAÑO DE LETRA. Se non me dixeron cal, pregúntame (usuario_preguntar) cunhas opcións (p.ex. Calibri, Georgia, Times, Arial, Verdana) ou propón unha lexible para un documento oficial. En Word/OpenDocument vai en ofimatica.fonte e tamanoCorpo (a letra ten que estar instalada no equipo de quen abra o documento: avísao); no PDF cambia a familia no LaTeX cun paquete habitual de TeX Live (sen fontes do sistema, salvo que sexa necesario e o motor sexa xelatex). Comproba con plantilla_previsualizar.',
  },
  {
    id: 'interlinado', icona: '≡', clave: 'asistente.tp.interlinado',
    titulo: 'Axustar o interliñado',
    instrucion: 'Le a plantilla aberta (plantilla_editor_ler) e axusta o INTERLIÑADO (espazo entre liñas). Se non me dixeron canto, pregúntame (usuario_preguntar: sinxelo 1, 1.15, 1.5, dobre 2). En Word/OpenDocument vai en ofimatica.interlinado; no PDF, con \\linespread{..}\\selectfont ou setspace no LaTeX. Comproba con plantilla_previsualizar e dime o que puxeches.',
  },
  {
    id: 'marxes', icona: '▭', clave: 'asistente.tp.marxes',
    titulo: 'Axustar as marxes',
    instrucion: 'Le a plantilla aberta (plantilla_editor_ler) e axusta as MARXES da páxina. Se non me dixeron cales, pregúntame (usuario_preguntar: estreitas 1.5 cm, normais 2.5 cm, anchas 3 cm; ou superior/inferior/esquerda/dereita). En Word/OpenDocument vai en ofimatica.marxeSuperior/Inferior/Esquerda/Dereita (cm); no PDF, con \\geometry{...} no LaTeX. Comproba con plantilla_previsualizar que a cabeceira e o pé seguen ben.',
  },
  {
    id: 'cabeceira', icona: '⤒', clave: 'asistente.tp.cabeceira',
    titulo: 'Modificar a cabeceira',
    instrucion: 'Le a plantilla aberta (plantilla_editor_ler) e mellora a CABECEIRA das páxinas: logotipos (os do cartafol de imaxes da plantilla; se hai adxuntos, mira adxuntos_listar) e texto con variables ({{CENTRO}}, {{DEPARTAMENTO}}, {{CURSO}}...), ben aliñados e cun filete se queda ben. En Word/OpenDocument vai en ofimatica.cabeceira (@imaxe, texto, @esquerda/@centro/@dereita) e no PDF no \\fancyhead do LaTeX. Comproba con plantilla_previsualizar (a páxina 2 xa ten cabeceira).',
  },
  {
    id: 'pe', icona: '⤓', clave: 'asistente.tp.pe',
    titulo: 'Modificar o pé de páxina',
    instrucion: 'Le a plantilla aberta (plantilla_editor_ler) e mellora o PÉ DE PÁXINA: datos do centro (variables como {{TELEFONO}} ou {{CORREO}}; se non existen, pregúntame os valores con usuario_preguntar e crea as variables) e o número de páxina. En Word/OpenDocument vai en ofimatica.pe (o número engádese solo) e no PDF no \\fancyfoot do LaTeX. Comproba con plantilla_previsualizar.',
  },
  {
    id: 'cores', icona: '◐', clave: 'asistente.tp.cores',
    titulo: 'Cores corporativas',
    instrucion: 'Le a plantilla aberta (plantilla_editor_ler) e axusta as CORES: pregúntame (usuario_preguntar) a cor principal do centro ou tómaa dun logotipo/documento adxunto (adxunto_ver). En Word/OpenDocument vai en ofimatica (corPrincipal, corSecundaria, corTexto, corApagada, corLinha, en hexadecimal) e no PDF nas cores da portada e do LaTeX. Procura contraste suficiente para imprimir en branco e negro. Comproba con plantilla_previsualizar.',
  },
  {
    id: 'logotipos', icona: '🖼', clave: 'asistente.tp.logotipos',
    titulo: 'Poñer os logotipos adxuntos',
    instrucion: 'Mira os ficheiros que xuntei a esta conversa (adxuntos_listar; imaxes ou PDF con logotipos), gárdaos na plantilla con adxunto_gardar_en_plantilla (varios á vez) e colócaos ben: na cabeceira (ofimatica.cabeceira con @imaxe; no LaTeX con \\includegraphics no \\fancyhead) e na portada se procede. Se non hai adxuntos, dío e pídeme que os xunte co clip. Comproba con plantilla_previsualizar.',
  },
  {
    id: 'imitar', icona: '⎘', clave: 'asistente.tp.imitar',
    titulo: 'Copiar o deseño dun documento adxunto',
    instrucion: 'Mira o documento (imaxe, PDF, Word ou papel timbrado) que xuntei a esta conversa (adxunto_ver / adxunto_ler) e reproduce o seu deseño na plantilla aberta: cabeceira, pé, portada, cores, tipo de letra, marxes e logotipos (adxunto_extraer_imaxes + adxunto_gardar_en_plantilla), pondo os datos fixos coma variables. Aplícao ao PDF e a Word/OpenDocument. Compara o resultado con plantilla_previsualizar e axusta. Se non hai adxunto, dío e pídeme que o xunte co clip.',
  },
  {
    id: 'sinatura', icona: '✍', clave: 'asistente.tp.sinatura',
    titulo: 'Sinatura e introdución',
    instrucion: 'Le a plantilla aberta (plantilla_editor_ler) e revisa a SINATURA (última folla) e o texto da INTRODUCIÓN por defecto: que usen as variables ({{XEFE}}, {{CARGO}}, {{CENTRO}}, {{LOCALIDADE}}, {{DATA_LONGA}}...), estean ben redactadas en galego e ben presentadas. Cambia só o necesario e comproba con plantilla_previsualizar (última páxina).',
  },
  {
    id: 'revisar', icona: '✔', clave: 'asistente.tp.revisar',
    titulo: 'Revisar e probar a plantilla',
    instrucion: 'Le a plantilla aberta (plantilla_editor_ler), compila o borrador (plantilla_comprobar) e mira a portada e unha páxina interior en pdf e en docx (plantilla_previsualizar). Se non compila ou hai algo feo (cortes, solapamentos, logotipos desproporcionados, variables sen valor, marcadores {{...}} sen substituír), corrixeo. Ao final dime, en poucas liñas, o que arranxaches e o que recomendas.',
  },
]

const CLAVE_PROPIAS = 'lua.asistente.tarefas'

// lerPropias devolve as tarefas engadidas polo usuario.
export function lerPropias(almacen = globalThis.localStorage, clave = CLAVE_PROPIAS) {
  try {
    const v = JSON.parse(almacen.getItem(clave) || '[]')
    return (Array.isArray(v) ? v : [])
      .filter((x) => x && typeof x.titulo === 'string' && typeof x.instrucion === 'string' && x.titulo.trim() && x.instrucion.trim())
      .map((x) => ({ id: String(x.id || x.titulo), titulo: x.titulo.trim(), instrucion: x.instrucion.trim(), propia: true }))
  } catch {
    return []
  }
}

function gardarPropias(lista, almacen, clave = CLAVE_PROPIAS) {
  try { almacen.setItem(clave, JSON.stringify(lista.map(({ id, titulo, instrucion }) => ({ id, titulo, instrucion })))) } catch { /* sen persistencia */ }
}

// engadirPropia garda unha tarefa nova do usuario e devolve a lista actualizada.
export function engadirPropia(titulo, instrucion, almacen = globalThis.localStorage, clave = CLAVE_PROPIAS) {
  const t = String(titulo || '').trim()
  const i = String(instrucion || '').trim()
  if (!t || !i) throw new Error('A tarefa necesita un nome e unha instrución.')
  const lista = lerPropias(almacen, clave)
  lista.push({ id: 'p' + Date.now().toString(36) + Math.random().toString(36).slice(2, 5), titulo: t, instrucion: i, propia: true })
  gardarPropias(lista, almacen, clave)
  return lista
}

export function eliminarPropia(id, almacen = globalThis.localStorage, clave = CLAVE_PROPIAS) {
  const lista = lerPropias(almacen, clave).filter((x) => x.id !== id)
  gardarPropias(lista, almacen, clave)
  return lista
}

// instrucionFinal xunta a ficha coas instrucións adicionais do usuario.
export function instrucionFinal(tarefa, extra) {
  const e = String(extra || '').trim()
  return e ? `${tarefa.instrucion}\n\nInstrucións adicionais do usuario: ${e}` : tarefa.instrucion
}

// TAREFAS_FORMULARIO son as fichas do asistente do DESEÑADOR DE FORMULARIOS
// (ámbito «formulario»). Traballan sobre o borrador do formulario aberto
// (formulario_ler / formulario_cambiar) e miran o resultado con formulario_probar.
export const TAREFAS_FORMULARIO = [
  {
    id: 'documento', icona: '📄', clave: 'asistente.tf.documento',
    titulo: 'Deseñar a partir dun documento adxunto',
    instrucion: 'Deseña o formulario e as tuberías a partir do DOCUMENTO XA FEITO que che adxuntei (un informe, unha acta, unha folla de notas, unha ficha...). Sigue o método de formulario_axuda: le o documento (adxunto_ler) e mira as súas páxinas (adxunto_ver); separa o FIXO (cabeceira, textos legais: son da plantilla, non campos) do VARIABLE (datas, nomes, números, observacións, listas: campos); as listas en filas son UNHA táboa; o que o documento CALCULA (totais, medias, porcentaxes, aprobado/suspenso) vai nunha tubería, e se non hai nada calculado, filtrado ou condicional NON fagas tubería. Respecta a orde e os títulos das seccións. Constrúe con formulario_cambiar, proba con formulario_probar usando datos de mostra tirados do propio documento, COMPARA a páxina renderizada co orixinal e corrixe as diferenzas. Se hai algo ambiguo, pregúntame unha soa cousa clara (usuario_preguntar). Se non adxuntei ningún documento, pídemo (usuario_preguntar).',
  },
  {
    id: 'crear', icona: '✨', clave: 'asistente.tf.crear',
    titulo: 'Crear un formulario desde unha descrición',
    instrucion: 'Deseña o formulario que che describa na caixa (que datos hai que recoller e que documento debe saír). Le antes formulario_axuda e o borrador (formulario_ler): se xa ten campos, parte deles. Elixe o tipo axeitado de cada campo (data, numero, escolla, verificacion, textolongo, taboa), dálles ids claros e etiquetas en galego, agrupa con seccións e engade tubería só se hai cálculos, filtros ou condicións. Proba con formulario_probar e ensínamo. Se non me deron ningunha descrición, pregúntame que formulario quero (usuario_preguntar).',
  },
  {
    id: 'tuberias', icona: '⚙', clave: 'asistente.tf.tuberias',
    titulo: 'Engadir tuberías (cálculos, filtros, táboas)',
    instrucion: 'Le o formulario aberto (formulario_ler) e engade as TUBERÍAS que fagan falta: o que se poida CALCULAR (medias, notas ponderadas, porcentaxes, contas de filas), FILTRAR ou ORDENAR nunha táboa, compor nunha frase ou amosar só se se cumpre algo. Se me dixeron que quero, faino; se non, propóñeme o que ten sentido para estes campos (usuario_preguntar cunha lista curta). Le antes formulario_axuda para usar ben os tipos de nodo. Non quites campos nin bloques que xa había e conserva a orde do documento. Proba con formulario_probar con datos de mostra que dean números comprobables a man e dime as contas.',
  },
  {
    id: 'probar', icona: '🔍', clave: 'asistente.tf.probar',
    titulo: 'Revisar e probar o formulario',
    instrucion: 'Revisa o formulario aberto (formulario_ler): tipos de campo axeitados, etiquetas claras, campos obrigatorios, táboas ben definidas e, se hai tubería, que non haxa nodos con erro nin bloques que non saian. Proba con formulario_probar cunhas filas de mostra realistas e mira a páxina renderizada. Di o que atopes e corrixe o que sexa claramente un fallo; o que sexa unha preferencia, pregúntamo antes.',
  },
]

// TAREFAS_FORMULARIO_DATOS son as fichas do asistente dos DATOS dun formulario
// en modo deseño (ámbito «formulario_datos»): datos ficticios para probar.
export const TAREFAS_FORMULARIO_DATOS = [
  {
    id: 'encher', icona: '✨', clave: 'asistente.tfd.encher',
    titulo: 'Encher con datos ficticios',
    instrucion: 'Le o formulario (formulario_datos_ler) e énchoo enteiro con datos FICTICIOS verosímiles: un valor en cada campo (respectando o seu tipo, as opcións das escollas e as columnas das táboas) e os datos do documento (centro, curso, quen asina...). Nas táboas pon entre 4 e 8 filas variadas. Non cambies o que xa estea escrito salvo que llo pida na caixa. Ao rematar, resume o que enchiches.',
  },
  {
    id: 'taboas', icona: '▦', clave: 'asistente.tfd.taboas',
    titulo: 'Engadir filas ficticias ás táboas',
    instrucion: 'Le o formulario (formulario_datos_ler) e engade filas ficticias verosímiles ás táboas: se non me din cantas, engade 10 a cada unha, conservando as filas que xa haxa (manda a lista completa de filas, as de antes e as novas). Non toques os outros campos.',
  },
  {
    id: 'baleirar', icona: '🧹', clave: 'asistente.tfd.baleirar',
    titulo: 'Baleirar os datos',
    instrucion: 'Pregúntame con usuario_preguntar se estou seguro de baleirar TODOS os datos do formulario e, se me responde que si, baleiraos con formulario_datos_cambiar (limpar=true). Non cambies nada máis.',
  },
]
