---
type: plan
title: Plan — Mensaxes frecuentes de Lua no idioma activo
description: Plan de estrutura para traducir ~74 erros e avisos frecuentes que xera o Go de Lua e edicion, coa lista para aprobar.
tags: [sdd, plan, i18n, lua, edicion]
timestamp: 2026-10-10T00:00:00Z
topic: sdd
slug: mensaxes-go-idiomas
status: draft
---

# Plan — Mensaxes frecuentes de Lua no idioma activo

> Spec: [../specs/mensaxes-go-idiomas.md](../specs/mensaxes-go-idiomas.md) · Constitution: [../constitution.md](../constitution.md) · Status: draft
> Last updated: 2026-10-10

## 0. Global Constraints

- **Idiomas:** `gl`, `es`, `en`, `pt`. O galego é a fonte e a reserva.
- **Idioma efectivo de Lua:** `Settings.Idioma` e, se está baleiro, `IdiomaPiztu()`. É o mesmo que xa usa o frontend.
- **Galego intacto:** con `gl`, cada mensaxe da lista é byte a byte a de hoxe.
- **Só a lista aprobada** (§4) se traduce; o resto queda como está.
- **Partes variables intactas:** os argumentos (rutas, paquetes, números, erros envoltos) insírense tal cal; os catro idiomas teñen os mesmos verbos de formato na mesma orde.
- **Yang e calquera app que non rexistre dicionarios:** seguen en galego sen cambios.
- **Constitución:** 1.3 (o común vai en `edicion`), 2.1-2.3, 3.1, 3.2, 4.3 (sen `Co-Authored-By`).

## 1. Context & constraints

- Aceptación da spec: 1 (xelatex en inglés), 2 (ficheiro non válido en castelán), 3 (paquete LaTeX en portugués co nome), 4 (galego idéntico), 5 (test garda), 6 (lista dentro das sete situacións), 7 (IA sen clave en inglés), 8 (lista no plan).
- Hoxe o Go «nunca traduce» (comentario de `Settings.Idioma`): todas as mensaxes son `fmt.Errorf`/`Sprintf` en galego, repartidas entre `lua/lua` e `edicion`.
- `edicion` non coñece a configuración de Lua (`edicion/doc.go`): o idioma tenllo que dicir a app.
- Fóra: erros raros, respostas para a IA, logs, saída de programas externos, instalador, Buxo.

## 2. Architecture

```text
Lua: idioma efectivo ──SetIdioma──┐        Lua: idiomas/*.json (claves go.*) ──Rexistrar──┐
                                  v                                                       v
                        edicion/idioma (tradutor común, global ao proceso)
                          T(clave, galego, args)      → texto no idioma activo
                          Errorf(clave, galego, args) → erro que se traduce ao lelo
                                  ^                                   ^
                 lua/lua: ~45 sitios da lista          edicion: ~30 sitios da lista
```

- **`edicion/idioma`** (interno, novo) — garda o idioma activo e os dicionarios rexistrados; traduce con reserva ao galego. Non sabe nada de Lua. Tamén o usará Buxo (plan irmán).
- **Lua** (interno) — embebe `idiomas/*.json`, rexístraos ao arrincar e chama `SetIdioma` co idioma efectivo ao arrincar, ao gardar Opcións e cando o asistente cambia o idioma.
- **Sitios da lista** (internos) — cambian `fmt.Errorf("…")` por `idioma.Errorf("go.clave", "…")` e os avisos por `idioma.T(...)`. O texto galego queda no código, coma o fallback de `t()` no frontend.

**Decisión principal:** traducir **no Go, no momento de ler a mensaxe**, e non no frontend. Os erros chegan ao frontend como texto plano, e moitos pasan por varias capas envoltos con `%w`. Se o Go devolvese claves, habería que cambiar todos os bindings e o seu manexo no JS. Un erro que se traduce cando se le cumpre ademais o caso de cambiar de idioma a metade dunha operación (spec, Behaviour). Alternativa descartada: códigos de erro e tradución en JS. Exixiría tocar 40 bindings.

## 3. Interfaces & contracts

```text
idioma.Rexistrar(codigo, dicionario map[clave]formato)   // a app, ao arrincar
idioma.SetIdioma(codigo)                                 // descoñecido ou baleiro → "gl"
idioma.Actual() -> codigo

idioma.T(clave, formatoGalego, args...) -> string
  - formato do idioma activo se existe a clave; se non, formatoGalego
  - mesmo resultado ca fmt.Sprintf para "gl"

idioma.Errorf(clave, formatoGalego, args...) -> error
  - Error(): tradución no idioma activo NO MOMENTO de chamalo
  - errors.Is / errors.As / errors.Unwrap funcionan coma con fmt.Errorf e %w
  - para "gl", Error() == fmt.Errorf(formatoGalego, args...).Error()
  - idioma.Galego(err) -> texto en galego dese erro, aínda que o idioma activo sexa outro
    (para logs como buxo.log; para un erro normal devolve err.Error())

Seguro para uso concorrente (as compilacións e descargas van en gorutinas).
```

## 4. Data model & flow — a lista para aprobar

Claves con prefixo `go.` en `lua/lua/idiomas/{gl,es,en,pt}.json`. O frontend non as usa. Un texto repetido en varios sitios é unha soa clave. **74 textos.**

**1. Compilar a PDF (12)**
1. non se atopou %s. Instala unha distribución LaTeX (p.ex. 'sudo apt install %s') — Lua e edicion
2. %s non conseguiu compilar este informe; usouse %s automaticamente. Podes cambiar o motor por defecto en Opcións.
3. a plantilla pide a fonte «%s», que non está instalada neste equipo: instálaa, ou cambia a fonte (…) — Lua e edicion
4. falta o paquete de LaTeX «%s»: instálao (p.ex. 'sudo tlmgr install %s') ou quítao da plantilla — Lua e edicion
5. o documento usa un ambiente LaTeX que non existe («%s»); … — Lua e edicion
6. a imaxe «%s» non se puido ler (%v): omitiuse
7. a imaxe «%s» ten un formato que LaTeX non le (só PNG, JPEG e GIF): omitiuse
8. a imaxe «%s» (GIF) non se puido converter: omitiuse
9. non se puido xerar a vista previa das páxinas: %v
10. non se puideron copiar as imaxes da plantilla: %w — Lua e edicion
11. LaTeX non deu ningunha páxina
12. a plantilla usa {{%s}}, que non está definida: quedou baleira

**2. Maxima (6)**
13. Maxima non respondeu en %s - a sesión reiniciouse automaticamente, podes tentalo de novo
14. non se puido iniciar Maxima: %w
15. non se cumpre «%s»
16. non se cumpre «%s» (Maxima di %s)
17. %s: {%s} está nunha fórmula pero non é ningunha variable
18. pregunta %s: aínda non ten solución

**3. Instalar dependencias (1)**
19. Non se atopou 'pkexec' para pedir permisos de administrador. Instala manualmente nun terminal cos comandos de %s. — LaTeX e Maxima

**4. Exportar, previsualizar e ler documentos (12)**
20. non se atopou pdftotext (paquete poppler-utils) para ler o PDF
21. para ler «%s» fai falta LibreOffice, que non está instalado: convérteo a PDF, DOCX ou ODT
22. LibreOffice non puido converter «%s»: %w
23. non se atopou pdftoppm (paquete poppler-utils) para debuxar a vista previa
24. non se atopou pdftoppm (paquete poppler-utils) para ver as páxinas
25. a imaxe «%s» non se puido usar (%v): omitiuse
26. o logotipo non se puido usar (%v): omitiuse
27. o deseño de Word/OpenDocument da plantilla usa a imaxe «%s», que non se puido usar (%v): omitiuse
28. a plantilla aínda non ten imaxes (gárdaa primeiro)
29. cor non válida: %q (usa un valor hexadecimal coma 1F3864)
30. non se atopou pandoc (precísase para as fichas en Markdown)
31. o ficheiro está danado ou non é un documento válido: %w

**5. Abrir e gardar (21)**
32. o ficheiro é demasiado grande (máximo %d MB)
33. o ficheiro non é un JSON válido: %w
34. este ficheiro é un documento cos DATOS dun formulario, non o formulario en si: ábreo desde o formulario ao que pertence
35. o ficheiro non é un formulario de Lua (falta o marcador «%s»)
36. o ficheiro ten a versión %d do formato e este Lua só entende ata a %d: actualiza Lua
37. o formulario do ficheiro non é válido: %w
38. Xa tiñas un formulario con ese identificador: o importado entrou coma copia, «%s».
39. %d datos do ficheiro non correspondían a ningún campo do formulario e non se cargaron.
40. o formulario entrou na biblioteca pero non se puideron cargar os seus datos: %w
41. o ficheiro non é unha plantilla de %s válida: %w
42. o ficheiro non é unha plantilla de %s
43. esta plantilla fíxose cunha versión máis nova de %s: actualiza %s para importala
44. plantilla importada, pero a imaxe %q non se puido gardar: %w
45. a plantilla non ten código LaTeX
46. o formulario non ten nome
47. «%s» non ten etiqueta
48. «%s» é unha escolla sen opcións
49. a táboa «%s» non ten columnas
50. xa hai un cartafol chamado «%s»
51. o cartafol necesita un nome
52. o deseñador deste formulario está aberto con cambios sen gardar: gárdaos ou péchao antes de pasalo a produción

**6. Actualizacións e módulos (14)**
53. non se puido descargar a actualización: %w
54. non se puido descargar a actualización: HTTP %d
55. o ficheiro descargado non é un zip válido: %w
56. esta Release non publica un .sha256 do .zip — non se pode verificar a súa integridade, así que se cancela a actualización
57. o SHA-256 do .zip descargado non coincide co publicado na Release — podería estar adulterado; cancélase a actualización
58. a actualización automática aínda non está soportada en Windows
59. non se puido instalar o binario novo: %w
60. non se puido reiniciar %s: %w
61. non hai ningunha actualización dispoñible
62. Buxo precisa unha IA configurada en Opcións: %w
63. Buxo aínda non está instalado (descárgase ao arrincar Lua; mira Opcións)
64. non se puido lanzar Buxo: %w
65. a última Release de %s non leva ningún «%s*.zip» para esta plataforma (¿sen rede?)
66. non se puido instalar o módulo: %w

**7. Provedor de IA (8)**
67. falta a clave da API de IA (configúraa en Opcións) — Lua e edicion
68. falta o modelo de IA (configúrao en Opcións) — Lua e edicion
69. non se puido contactar coa IA: %w — 4 sitios
70. resposta non válida da IA (HTTP %d) — 6 sitios
71. a IA non devolveu resultado (HTTP %d) — 6 sitios
72. resposta non válida da IA: %w
73. erro da IA: %s
74. non hai ningún provedor de IA configurado

**Fóra a propósito** (exemplos): as mensaxes que nomean ferramentas da IA (`adxunto_ver`), os erros internos de ZIP, rutas e executables (`non se puido crear %q`), a validación de identificadores internos e os erros de `edicion/ficha` (formato de Yang).

**Fluxo:** arrinque → rexistro dos catro dicionarios → `SetIdioma(efectivo)` → cada `Error()`/`T()` le o idioma activo. Ao gardar Opcións ou cambiar o idioma desde o asistente → `SetIdioma`.

## 5. Testing strategy

| Aceptación | Nivel | Que comproba | Que se finxe |
|---|---|---|---|
| 4 | unidade, `edicion/idioma` | con `gl`, `T` e `Errorf` dan exactamente o de `fmt.Sprintf`/`fmt.Errorf` | dicionario de proba |
| 1, 2, 3, 7 | unidade, `edicion/idioma` | con `en`/`es`/`pt`, devolve a tradución cos argumentos; clave que falta → galego; idioma descoñecido → galego | dicionario de proba |
| — | unidade, `edicion/idioma` | `errors.Is` e `Unwrap` atravesan un `Errorf` con `%w`; cambiar de idioma despois de crear o erro cambia o texto | — |
| — | unidade, `edicion/idioma` | uso concorrente sen carreiras (`go test -race`) | — |
| 1, 3 | unidade, Lua | `compilarComMotores` co idioma `en` devolve o aviso de cambio de motor en inglés; `pistaLatexAccionable` dá o paquete en portugués co nome | binario LaTeX finxido / log de proba |
| 2 | unidade, Lua | importar un ficheiro que non é un formulario, co idioma `es`, devolve o erro en castelán | ficheiro temporal |
| 7 | unidade, Lua | `ia_conversa` sen clave co idioma `en` → erro en inglés | configuración baleira |
| 5, 6 | unidade, Lua (estende `i18n_test.go`) | toda chamada `idioma.T`/`Errorf` de Lua e edicion ten a clave en `gl.json`; o galego do código coincide co de `gl.json`; os catro idiomas teñen os mesmos verbos (`%s %d %v %q %w`) na mesma orde | — (le os fontes) |
| Opcións | unidade, Lua | gardar Opcións con `idioma: "pt"` deixa `idioma.Actual() == "pt"`; idioma baleiro → `IdiomaPiztu()` | sen interface |

## 6. Sequencing & dependencies

1. `edicion/idioma` cos seus tests. Compártese co plan de Buxo: faise unha soa vez, primeiro.
2. Lua: embeber e rexistrar dicionarios e `SetIdioma` nos tres puntos (arrinque, Opcións, asistente), con tests. Depende de 1.
3. Garda: estender `i18n_test.go` para as claves `go.*` e os verbos `%w %q %v`. Ten que fallar antes do paso 4. Depende de 1.
4. Migrar os sitios da lista por situación (1→7), engadindo `gl.json` en cada paso. Cada situación é independente das outras e verifícase co garda. Depende de 2 e 3.
5. Traducións es/en/pt das 74 claves. Depende de 4.
6. Tests de aceptación por situación (§5). Despois, suite completa de Lua, edicion e Yang (Yang debe seguir igual).

## 7. Risks & open decisions

1. **Mensaxes envoltas a medias.** *Disparador:* un erro da lista envolto por outro que non está na lista («X: <tradución>»). *Impacto:* medio, unha mestura de idiomas nunha liña. *Mitigación:* as envolturas máis comúns xa están na lista (62, 66); o resto acéptase porque a spec deixa os erros raros en galego.
2. **Comparacións de texto no código ou nos tests** (`strings.Contains(err.Error(), "non se atopou")`). *Disparador:* unha comparación que deixa de casar cando o idioma non é gl. *Mitigación:* buscalas antes de migrar cada sitio; os tests corren en gl por defecto.
3. **Estado global.** *Disparador:* dous tests que cambian o idioma en paralelo. *Mitigación:* os tests que cambian o idioma restáurano e non usan `t.Parallel`.
4. **Yang afectado.** Retirado: Yang non rexistra dicionarios, así que todo sae en galego. Compróbase coa súa suite.
5. **Lista mal escollida.** *Mitigación:* apróbaa o usuario aquí; engadir unha clave máis despois é barato.

**Decisións abertas:** a aprobación da lista do §4.

**Illamento:** a constitución (4.1b) fixa o traballo directo en `main`; non se abre rama.
