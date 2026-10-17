---
type: decisions
title: lua_yang — Rexistro de decisións
description: Decisións significativas do proxecto, con data, opcións e porqué. Só engadir.
tags: [sdd, decisions]
topic: sdd
---

# Rexistro de decisións

## 2026-10-10 — Constitución v1.0.0 ratificada
- Decisión: `02-DOCS/wiki/sdd/constitution.md` pasa a v1.0.0.
- Porqué: o SDD necesita regras comprobables antes da primeira funcionalidade grande.

## 2026-10-10 — O produto é Lua
- Opcións: app LuaYang separada · Lua absorbe Yang.
- Decisión: Lua absorbe Yang; o produto é Lua máis `edicion/` e `buxo/`.
- Porqué: o usuario quere un programa único, e o traballo do 2026-10-09 xa vai dentro de Lua.

## 2026-10-10 — Autoría só humana
- Opcións: só o usuario · coautoría da IA nos commits.
- Decisión: só Martín Pardiñas Fondo; sen `Co-Authored-By` da IA.
- Porqué: decisión explícita do usuario.

## 2026-10-10 — Repositorio git na raíz
- Opcións: un repositorio que englobe todo · un repositorio na raíz que deixa fóra os que xa existen · submódulos.
- Decisión: o repositorio da raíz rastrexa o harness, `edicion/` e `buxo/`; `lua/`, `yang/` e `markitdown/` conservan o seu propio `.git` e quedan ignorados.
- Porqué: non se perde nin se mestura a historia dos repositorios que xa existen, e `edicion/` e `buxo/` pasan a ter control de versións.
- Pendente: `lua/.git` apunta a `github.com/piztutao/yang-src` (o mesmo remoto que `yang/`). O usuario indica `github.com/piztusistemas/lua`, que hoxe é público e só ten ficheiros de exemplo e de release. Falta decidir que se sobe alí.

## 2026-10-10 — Versión de Lua 26.10.3
- Decisión: Lua pasa a 26.10.3 en todos os ficheiros de versión do código (non nos binarios xa construídos).

## 2026-10-10 — Un só repositorio en GitHub
- Opcións: repositorio só de Lua · un repositorio con Lua, `edicion` e `buxo`.
- Decisión: un só repositorio, `github.com/piztusistemas/lua`, que une a historia local cos ficheiros que xa había publicados alí.
- Porqué: `lua/lua/go.mod` usa `../../edicion`, e por separado Lua non compila.
- A historia antiga de `lua/.git`, que apuntaba a `yang-src`, gárdase fóra do proxecto en `../lua-git-respaldo`.

## 2026-10-10 — Instalador en Wails v3
- Decisión: o instalador pasa de Wails v2 a v3, coma Lua e Buxo.
- Porqué: unha soa CLI (`wails3` + `task`) para todo o produto.
- Coidado: `wails3 task common:update:build-assets` reescribe o manifesto de Windows e quita `requireAdministrator`.

## 2026-10-10 — markitdown dentro do repositorio
- Opcións: submódulo git de microsoft/markitdown · copia dentro do repositorio.
- Decisión: copia dentro do repositorio, desde o commit `d9cae89`.
- Porqué: o submódulo perdería o plugin propio `markitdown-math` e obrigaría a un paso extra ao clonar. A historia anterior está en `../markitdown-git-respaldo`.

## 2026-10-10 — Alcance das traducións pendentes
- Opcións: traducir os 341 erros do Go · só os frecuentes · ningún; Buxo co idioma de Lua · con selector propio.
- Decisión: dúas specs (`buxo-idiomas`, `mensaxes-go-idiomas`). Só as mensaxes frecuentes do Go; Buxo segue o idioma de Lua; as respostas para a IA e os rexistros quedan en galego.
- Porqué: o custo de traducir erros raros que case ninguén ve non compensa; os frecuentes din como arranxar un problema.

## 2026-10-10 — Deseño das traducións do Go e de Buxo
- Decisión: un tradutor común novo, `edicion/idioma`, que usan Lua e Buxo. Os erros tradúcense ao lelos (`Error()`), non no frontend.
- Opcións descartadas: códigos de erro traducidos en JS (habería que tocar ~40 bindings); un tradutor propio en Buxo (duplicaría código, contra a regra 1.3).
- Lua pásalle o idioma a Buxo coa variable de entorno `BUXO_IDIOMA`, non cun argumento: un Buxo vello péchase cun argumento descoñecido.
- O rexistro de Buxo garda clave + argumentos: así o panel sae traducido e `buxo.log` en galego.

## 2026-10-10 — Traducir en cada situación, non ao final  (feature: mensaxes-go-idiomas, task: T006)
Context   — O test de idiomas exixe que cada clave estea nos catro JSON; engadir só o galego en T006-T012 e traducir en T013 deixaría os tests en vermello entre medias (constitución 2.2).
Options    — (a) traducir cada situación na súa tarefa; (b) relaxar o test ata T013.
Decision   — (a). T013 queda como comprobación final (74 claves en cada idioma).
Why        — Non se debilita un test para avanzar, e cada commit queda verde.

## 2026-10-10 — Nome visible «Lúa», nome técnico «lua»
- Decisión do usuario: na interface o aplicativo chámase «Lúa» (nome galego) e no apartado Sobre «Piztu Lúa». Cartafoles, rutas, ligazóns e identificadores seguen sendo «lua».
- Cambia a «Lúa»: textos dos catro idiomas (Lua, Buxo e instalador), título da xanela, CFBundleName de macOS, ProductName de Windows, textos de reserva no código e descricións para a IA.
- Queda «Lua»: `/Applications/Lua.app`, `Piztu/Lua`, `<config>/Lua/informes`, os accesos directos (`Lua.lnk`, `Name=Lua` do .desktop), `Options.Name` de Wails (identificador en Linux), a marca LaTeX `% --- engadido por Lua …`, o User-Agent e o repositorio.

## 2026-10-10 — Accesos directos e documentación tamén «Lúa»
- Decisión do usuario: corrixe a entrada anterior. Os accesos directos levan o nome visible «Lúa» (`Lúa.lnk` en Windows, `Name=Lúa` no .desktop de Linux) e o nome que ve Piztu no selector de módulos (`modulo.json` → "nome") tamén. A documentación viva (README, wiki, blog, constitución v1.2.1) di «Lúa».
- Seguen «lua»: rutas, cartafoles, executables, `lua.desktop`, `"id": "lua"` e os rexistros históricos do método SDD (specs, plans, análises, progreso, decisións), que describen o traballo tal como se fixo.
- Detalle técnico: ao instalar e desinstalar bórranse tamén os `Lua.lnk` de instalacións anteriores; o atallo créase con PowerShell `-EncodedCommand` (UTF-16) para que o til chegue intacto.

## 2026-10-10 — Deseñador de formularios: tubaxes e iconas
- «Tuberías» pasa a «Tubaxes» en galego (interface, textos para a IA e documentación viva). Claves e identificadores (`fluxo`, `tuberias`) non cambian; castelán, inglés e portugués seguen igual.
- A pestana Tubaxes amosa directamente o deseño, sen texto explicativo nin botón. O deseño do cableado automático é unha vista previa: o formulario segue en automático ata o primeiro cambio, que o personaliza. Por que: se se personalizase só por abrir a pestana, todo formulario que se mirase deixaría de seguir os campos novos sen que a persoa o decidise. «Volver ao cableado automático» só sae cando xa está personalizado.
- As iconas saen do deseñador. Cámbianse co botón dereito sobre o formulario na columna esquerda (ou no «⋯» da biblioteca) → «Cambiar icona…», nunha xanela propia. Novo método Go `GardarIconaFormulario(id, icona)`, que só toca a icona e vale tamén en produción (coma a descrición).

## 2026-10-10 — Título «Piztu Lúa» e cores dos formularios
- Decisión do usuario: o título das xanelas é «Piztu Lúa» (a principal) e «Piztu Lúa — …» (deseñador, plantilla, resultado, tarefa). A barra superior xa non repite o nome: só leva a mascota, sen recadro e a 48 px. O resto do nome visible segue sendo «Lúa».
- Cada formulario ten icona e cor propias (8 cores, ningunha vermella nin verde); o modo vai nun punto na esquina do botón. `GardarIconaFormulario(id, icona, cor)` garda as dúas. Os formularios sen cor gardada reparten as libres por orde de id, de xeito estable.

## 2026-10-11 — Trasladar: modelo de gardado, exportación e importación (spec `trasladar`)
- Decisións do usuario: (1) os datos pertencen sempre a un proxecto (fóra o traballo solto; migra só a «O meu traballo» con copia previa); (2) fóra o ficheiro vinculado con Ctrl+S, pero un documento pódese sacar como `.json` editable e volver meter ofrecendo «Actualizar»; (3) formato propio `.lua` que pode levar calquera combinación, sen deixar de ler os `.json` vellos.
- Un documento viaxa autosuficiente: datos, estrutura do formulario e plantilla. Ficheiros con varios cursos: lista marcada, un proxecto por curso.
- Opcións descartadas: só renomear botóns (non dá copia de proxecto nin bóveda); un tipo de ficheiro por contido (obriga ao usuario a saber que ficheiro é cada un).
- Risco anotado: a migración do traballo solto é o punto con máis perigo de perda de datos.

## 2026-10-11 — Trasladar: decisións de implementación (plan `trasladar`)
- Formato `.lua` = zip coa biblioteca estándar de Go (sen dependencias novas) e `manifest.json` (`formato: lua-paquete`, `version: 1`). Ao lelo rexéitanse rutas absolutas, con `..`, con `\` ou `:` e ligazóns; límites: 50 000 entradas, 1 GiB por entrada, 4 GiB en total.
- O `.json` editable dun documento (`formato: lua-documento`) leva `orixe.instalacion`, un identificador aleatorio novo en `Settings.Instalacion`: serve só para recoñecer a «volta» e ofrecer «Actualizar».
- Toda importación garda antes unha versión (máx. 5, en `<config>/lua/versions/`) do que pode cambiar e das rutas novas; se falla, restáurase (todo ou nada) e, se sae ben, a mesma versión é o «Desfacer».
- A plantilla de cada documento lémbrase pola ruta do seu ficheiro dentro do proxecto (`EscollerPlantillaDocumento`/`PlantillaDoDocumento`) e viaxa no paquete (`plantillaDocumento`).
- A copia completa restaura proxectos, biblioteca, plantillas e opcións, pero non a clave da IA, a instalación nin as bóvedas de Obsidian conectadas (son deste equipo).
- Quitáronse os bindings sen uso `AbrirInformeDialog`, `GardarInformeDialog`, `SaveFile`, `ExportarFormulario`, `ImportarFormulario`, `ExportarPlantilla` e `ImportarPlantilla`; as funcións puras dos formatos vellos quedan porque Trasladar segue léndoos.
- «Novo» xa non bota man dun ficheiro: baleira o documento do proxecto, con confirmación e Desfacer.

## 2026-10-11 — Proxectos lixeiros e catálogo común de formularios (spec `reutilizar-formularios`)
- Decisións do usuario: os proxectos crean baleiros; a ventá «Deseñar formulario» é a única porta para engadir ferramentas (tamén cronograma e actas); todo formulario gardado nun proxecto entra no catálogo común; quitar dun proxecto vai no botón dereito da columna; dúas opcións novas de solucións (só o resultado, debaixo ou ao final).
- Os cartafoles do catálogo gárdanse no `biblioteca_formularios.json` de fóra dos proxectos: así os cartafoles que xa existían seguen. Os cartafoles internos de cada proxecto non se ven nesta ventá.
- Na columna, «Quitar do proxecto…» substitúe a «Eliminar formulario…» (dentro dun proxecto, eliminar só borraba a copia sen sacala da ficha do proxecto).

## 2026-10-11 — Traballos na bóveda (spec `traballos-boveda`)
- Decisións do usuario: nota Markdown + PDF; en `wiki/traballos/`; botón e marca opcional «engadir sempre» (desmarcada por defecto); produción bloqueada.
- A bóveda é a que xa usa o asistente para ese documento (`bovedaDe`); se non hai ningunha, créase a do proxecto. Volver engadir substitúe.
- A marca vive no proxecto (`proxecto.json` → `bovedaAutomatica`), non no esquema do formulario: así non cambia o formulario (que en produción está conxelado) e cada proxecto decide.

## 2026-10-14 — Proxectos en cartafoles do equipo
- Decisións do usuario: «Crear» só crea o proxecto e abre o paso «Configurar» (cartafol, fontes de coñecemento, «Abrir proxecto» / «Agora non»); o proxecto novo sae despregado; os cartafoles son cartafoles reais do equipo e substitúen as etiquetas temáticas; o proxecto vai nun subcartafol co seu nome (`<cartafol>/<Nome>`, «Nome (2)» se se repite).
- Os proxectos gardados fóra anótanse en `<proxectos>/ubicacions.json` (id → ruta absoluta); `dirProxecto` consúltao e o resto de Lúa non cambia. Mover entre discos copia e despois borra. Un proxecto aberto non se move.
- Retíranse `ListarCarpetasProxectos`, `CrearCarpetaProxectos`, `RenomearCarpetaProxectos`, `EliminarCarpetaProxectos` e o campo `carpeta` do proxecto; `carpetas.json` xa non viaxa na copia completa (as copias vellas que o levan lense igual).
- Risco anotado: «Restaurar copia» volve deixar todos os proxectos no cartafol de Lúa; os que estaban fóra quedan no disco pero Lúa deixa de velos.

## 2026-10-14 — Inicio nos dous paneis e cor de cada proxecto
- Decisións do usuario: Lúa arrinca nunha pantalla de inicio que ocupa os dous paneis: no da esquerda, «Novo proxecto» e debaixo os proxectos creados ou metidos con Trasladar; os que non caben seguen no da dereita. Os paneis énchense co editor e o resultado ao abrir un proxecto. Cada proxecto ten a cor que escolla o usuario.
- Ao arrincar xa non se reabre o último proxecto nin se crea «O meu traballo» se non hai ningún (substitúe a suposición da spec `trasladar` «sempre hai un proxecto aberto» só para o arranque: no inicio non hai editor á vista, así que tampouco hai traballo solto). Desaparece a clave `lua.proxecto` de `localStorage`.
- A ventá flotante de Proxectos desaparece: o botón Proxectos da columna leva ao inicio e, se hai un proxecto aberto, volve a el (o proxecto non se pecha).
- O reparto entre paneis mídese no DOM (`cantasCaben` con alturas reais) e recalcúlase ao cambiar o tamaño. Se o panel da dereita non se ve (resultado noutra xanela), todo queda á esquerda con scroll. Opción descartada: CSS multicolumna, porque os dous paneis son elementos distintos co divisor no medio.
- A cor gárdase en `proxecto.json` → `cor` (`#rrggbb`, ou baleira = a de por defecto, que sae dunha paleta de oito segundo o id e non cambia cando nacen outros). Escóllese co selector de cor do sistema (accesible co teclado); viaxa cos paquetes porque vai na ficha. Novo binding `FixarCorProxecto`.

## 2026-10-16 — Deseño dos documentos coma datos (spec `deseno-documentos`)
- Decisión do usuario: automatizar o deseño do PDF para o profesorado sen perder o control; facer todas as fases en autopiloto.
- O deseño do PDF pasa a ser **datos** na plantilla (`deseno`, `edicion/plantilla/deseno.go`, versión 1) e Lúa xera o LaTeX (`lua/deseno_latex.go`). Opcións descartadas: seguir co LaTeX libre como única vía (é o problema de partida) e un editor WYSIWYG en HTML (non sería o PDF real).
- Prioridade: `estilo` (LaTeX libre) > `deseno` > Axustes do texto > deseño de Lúa. As plantillas existentes non cambian: as que teñen `estilo` seguen igual e o editor ofrece pasar ao visual.
- **Letra por papeis** (`edicion/latex/letras.go`): corpo (`\setmainfont`), títulos (`\setsansfont`), etiquetas (`\fontchip`) e fórmulas (`unicode-math`). Causa do erro «a letra non se reflicte»: só se cambiaba o corpo, os títulos quedaban en Latin Modern Sans, as etiquetas en Open Sans, «Sans» era igual ca o por defecto e o tamaño non chegaba a `\normalsize`. As fontes cárganse por ficheiro (TeX Live / `fonts/` de Lúa), con substituta se faltan.
- O tamaño vai con `scrextend` (`fontsize=`) e os títulos escalan co corpo. Sen `scrextend` ou `unicode-math` (TeX Live mínimo) o documento segue compilando: o tamaño vai tras `\begin{document}` e as fórmulas na letra de sempre.
- Unha plantilla completa que xa fixa a súa letra consérvaa e o resultado avisa; se non a fixa, recibe a do deseño.
- Interruptores do mundo real (branco e negro, aforrar papel, lectura fácil) aplícanse ao xerar e non tocan os valores gardados.
- Inspector: cada elemento escribe a súa posición (`\pdfsavepos`) en `doc.lpos`; vai sempre, non só na vista previa, para que a vista previa e o PDF final sexan idénticos.
- IA: o asistente cambia `deseno` (parche validado no frontend e normalizado no backend; respecta `fixados`). Para outras IA, `lua/skills/deseno-documentos/SKILL.md` + `EsquemaDeseno` (JSON Schema).
- Sen dependencias novas de execución (Go nin npm). Os paquetes de LaTeX novos (`scrextend`, `unicode-math`, fontes TeX Gyre Math, Fira, Libertinus, Source Serif) son opcionais e engadíronse á táboa de instalación baixo demanda.
- Fóra do alcance desta entrega (ver análise): PDF accesible etiquetado, catálogo de fontes descargable, deseño de Word/ODT desde o editor visual, e os exames de Yang (só reciben a letra por papeis).

## 2026-10-16 — Melloras do deseño (plan `deseno-melloras`)
- Decisión do usuario: executar en autopiloto o proxecto de melloras da análise crítica.
- **Estudio de deseño**: o deseño vai nunha columna propia, primeira e ancha; de primeiras só está aberto o deseño (unha plantilla nova abre tamén Identidade porque precisa nome). A vista previa faise ao abrir, é máis grande e salta á páxina da parte que se cambiou.
- **Tempo máximo e cancelación** (constitución 5.4 e 7.1): todas as pasadas de LaTeX dos informes van con `exec.CommandContext` e un máximo de 3 min; a vista previa do editor rexistra o seu cartafol e `CancelarVistaPrevia` mata o proceso. Unha compilación cancelada ou parada non proba outro motor.
- **Inspector visible**: resalte e nome da parte baixo o rato, a partir das marcas `doc.lpos`.
- **Portada por modelos** (`deseno.portada`: de cor, sobria, oficial, sen) e posición do logotipo; a portada en LaTeX da plantilla segue mandando. A portada de cor usa as cores do deseño.
- **Cabeceira e pé sen LaTeX** (`deseno.cabeceira`, `deseno.pe`, tres zonas con texto, variables, `@logo`, `@imaxe`; `deseno.numeroPaxina`). Van ao principio do corpo, despois da maqueta, para mandar sobre ela só cando se enchen. Word/ODT toman as zonas se o seu bloque está baleiro (unha caixa cun aliñamento: perde a distribución en tres zonas).
- **Letras embebidas**: Fira Sans e Fira Math, Libertinus Serif e Libertinus Math (OFL), como xa se facía con Open Sans; só se escriben ao carón do .tex as que usa o documento. Atkinson Hyperlegible descartouse (licenza propia, non OFL). A interface usa as mesmas fontes (e as TeX Gyre, GUST Font License) para as mostras «Aa». O binario medra uns 4 MB e o frontend 1,4 MB.
- **Kit do centro**: ficheiro `.luadeseno` (`formato: lua-deseno`, versión 1, só o deseño) e «Aplicar a todas as miñas plantillas» (as de exemplo non se tocan).
- **Barras reversibles**: lembran os valores de partida (`barras.base`) e restáuranos ao volver ao medio.
- **LaTeX dunha parte**: compróbase nun documento mínimo antes de usalo (`ComprobarElementoDeseno`); se falla, non se aplica salvo «Usar igualmente».
- **Nomes**: as barras chámanse «Carácter do documento»; «Letra de cada parte».
- `task dev` arranxado (script `build:dev` en `package.json`).


## 2026-10-17 — Números decimais nos documentos de problemas
- Informe do usuario: no PDF desaparecía a coma (ou o punto) decimal e o número de decimais disparábase (`0.30000000000000004`, `54.00000000000001`).
- Catro causas, catro arranxos:
  - **PDF**: babel-galician con `shorthands=off` deixa o «.» activo en modo matemático e sen definición, e `$12.75$` saía «1275». `latex.BabelLine` devolve o «.» ao código que garda o propio babel (vale tamén coa letra de unicode-math) e o «<»/«>» a carácter normal, ao empezar o documento e ao volver escoller o idioma. Descartado: fixar `\mathcode` a `"013A`, que rompe o punto das letras OpenType MATH; e quitar `shorthands=off`, que volvería romper os «<» dos TikZ.
  - **Rangos aleatorios**: calcúlanse en enteiros escalados ao número de decimais do rango (`rangoDecimal`), en Go e na liña de Maxima (`float((A + P*random(N))/10^d)`). Así `0,1 + 2·0,1` dá `0,3` e o extremo «ata» entra sempre (antes `floor((0.5-0.1)/0.1) = 3` deixaba fóra o 0,5). Os rangos enteiros sacan os mesmos números coa mesma semente ca antes.
  - **Resultados de Maxima**: o TeX de Maxima pasa por `numerosTex`, que quita o ruído do coma flotante (12 cifras significativas), os ceros de máis («80.0» → «80») e pon a coma decimal (`{,}`), coma xa facía o camiño en Go.
  - **Decimais**: a opción «Decimais nos resultados dos problemas» (xa existía para Maxima, con 0 = sen redondear) aplícase tamén ás contas en Go.
