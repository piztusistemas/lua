---
type: plan
title: Plan — Trasladar
description: Plan de estrutura para a interface única de exportar e importar, o paquete .lua, o documento .json editable, as versións para desfacer e a migración do traballo solto.
tags: [sdd, plan, exportar, importar]
timestamp: 2026-10-11T00:00:00Z
topic: sdd
slug: trasladar
status: approved (autopiloto, 2026-10-11)
---

# Plan — Trasladar

> Spec: [../specs/trasladar.md](../specs/trasladar.md) · Constitution: [../constitution.md](../constitution.md) v1.2.1 · Status: approved in autopilot

## 0. Global Constraints
- Constitución 1.1-1.3 (Go + JS sen framework; ningunha dependencia nova: zip e JSON da biblioteca estándar), 2.1-2.4, 3.1-3.2 (textos nos catro idiomas), 5.1 (sen claves nos ficheiros), 5.2, 6.1-6.2, 8.1-8.2.
- Traballo directo en `main` (4.1b). Commits en galego con gitmoji, sen co-autoría de IA (4.3).
- Nunca se sobrescribe nada en silencio: toda escritura destrutiva vai precedida dunha versión gardada.

## 1. Context & constraints
- Os datos dun proxecto viven en `<proxectos>/<id>/` (ficha, `formularios/`, `datos/<ferramenta>-borrador.json`, `biblioteca_formularios.json`, `boveda/` con `pdf/`, `raw/`, `wiki/`). Fóra de proxecto, os borradores van a `<config>/lua/informes/`.
- O borrador xa é o «gardado en Lúa»: o controlador escríbeo 700 ms despois de cada cambio.
- Formatos existentes que hai que seguir lendo: curso de cronograma (`{plans:[…]}`, array ou obxecto con `config`/`monthsData`), actas (`{ferramenta:"actas", actas:[…]}`), documento de formulario (`{ferramenta:"formulario", formulario, datos}`), paquete de formulario (`formato:"lua-formulario"`), paquete de plantilla (`formato:"piztu-plantilla"` e históricos).
- A plantilla lémbrase por «ficheiro» (`depositoPlantillas.ParaFicheiro(clave)`). Sen ficheiros, a clave pasa a ser `lua://proxecto/<id>/<ferramenta>`.
- Wails v3 beta.14: `EnableFileDrop` + `data-file-drop-target` no DOM + evento `WindowFilesDropped` con rutas.

## 2. Architecture
```text
frontend/src/trasladar/modelo.js   (puro: preselección, resumo, textos)  ← tests vitest
frontend/src/trasladar/trasladar.js (xanela: Sacar | Meter, árbore, resumo, destino, cursos, desfacer)
main.js ── botón ⇅ / Ctrl+O / Ctrl+Maiús+S / soltar ficheiro / menús contextuais ──> abrirTrasladar(opcions)

Go:
trasladar.go           tipos, manifesto, clasificación e destinos (puro sobre bytes/rutas)
trasladar_paquete.go   escribir/ler o zip .lua con límites e rutas seguras
trasladar_meter.go     aplicar: proxectos, formularios, documentos, bóveda, plantillas, copia completa
trasladar_versions.go  versións para desfacer (máx. 5)
trasladar_migracion.go traballo solto → proxecto «O meu traballo» (unha vez, con marca)
trasladar_app.go       bindings de Wails e evento de ficheiros soltos
```
**Decisión principal:** toda a lóxica de ficheiros vive en Go (atomicidade, límites, rutas seguras) e o frontend só decide a interface. A clasificación dos `.json` vellos faise en Go para que «Meter» funcione igual dende o botón, ao soltar ou desde o asistente.

## 3. Interfaces & contracts
```text
SacarTraslado(sel SeleccionSacar) (ResultadoSacar, error)          // diálogo gardar; .lua
  SeleccionSacar{Tipo: "proxecto"|"todo"|"seleccion", Proxectos, Formularios, Plantillas []string, Datos bool}
SacarDocumentoJSON(ferramenta, contido string) (string, error)      // diálogo gardar; .json editable
SacarTenDatosProducion(sel) (bool, error)
ArboreTraslado() (ArboreTraslado, error)                            // proxectos, formularios, plantillas
EscollerFicheiroMeter() (string, error) · EscollerCartafolMeter() (string, error)
AnalizarMeter(ruta string) (ResumoMeter, error)
  ResumoMeter{Tipo, Nome, Elementos []ElementoMeter, Cursos int, Ferramenta, Destino, Volta *Volta, Produccion bool, Avisos}
AplicarMeter(ruta string, op OpcionsMeter) (ResultadoMeter, error)
  OpcionsMeter{Destino: "aberto"|"novo"|"", Cursos []int, Accion: "actualizar"|"copia"|""}
  ResultadoMeter{Proxectos []string, Formularios []string, Plantillas []string, Version string, Avisos, AbrirProxecto string}
DesfacerTraslado(version string) error
MigracionTraballoSolto() (ResultadoMigracion, error)                // idempotente; aviso unha soa vez
```
Ficheiro `.lua` = zip: `manifest.json` (`formato:"lua-paquete"`, `version:1`, contido, `completa`, `instalacion`), `proxectos/<id>/…`, `formularios/<id>.json`, `plantillas/<id>.json` (paquete de plantilla), e só na copia completa `configuracion/{settings.json sen claves, biblioteca_formularios.json, carpetas.json}`.
Documento `.json` = `{formato:"lua-documento", version:1, orixe:{instalacion, proxecto, ferramenta}, ferramenta, nome, documento:<o mesmo JSON ca o borrador>, formulario?, plantilla?}`.

## 4. Data model & flow
- **Settings.Instalacion**: identificador aleatorio desta instalación (recoñece a «volta»). Non é segredo.
- **Versións**: `<config>/lua/versions/<data>-<tipo>/meta.json` + copias. Restaurar volve copiar cada ruta ao seu sitio.
- **Meter**: extráese a un cartafol temporal, valídase todo e só despois se move ao destino (todo ou nada).
- **Migración**: marca `<config>/lua/.migracion-proxectos`; os borradores soltos pasan ao proxecto, cunha versión antes.
- **Sempre un proxecto**: ao arrincar sen proxecto, ábrese o último; se non hai ningún, créase «O meu traballo».

## 5. Testing strategy
| Aceptación | Nivel | Test |
|---|---|---|
| Sacar proxecto → .lua con formularios, datos, bóveda; meter noutro → igual | Go | `TestPaqueteProxectoIdaEVolta` |
| JSON editable, cambio, volta con Actualizar e Desfacer; Engadir como copia | Go | `TestDocumentoVoltaActualizaEDesfai`, `TestDocumentoVoltaComoCopia` |
| Formulario con datos → proxecto novo + biblioteca; sen datos → só biblioteca | Go | `TestMeterFormularioConESenDatos` |
| Produción avisa | Go + vitest | `TestSacarTenDatosProducion`, `modelo.test.js` |
| Sen claves | Go | `TestPaqueteSenClaves` |
| `.json` vellos | Go | `TestClasificarFormatosVellos` |
| Tres cursos, desmarcar un → dous proxectos | Go | `TestMeterVariosCursos` |
| Copia completa e restaurar recuperable | Go | `TestCopiaCompletaRestaurarEDesfacer` |
| Ficheiro danado non cambia nada | Go | `TestMeterDanadoNonCambiaNada`, `TestPaqueteRutasPerigosas` |
| Migración | Go | `TestMigracionTraballoSolto` |
| Entrada única / «Gardado en Lúa» | vitest + busca | `trasladar.test.js`, `controlador.test.js`; grep sen `btnOpen`/`btnSaveAs` |
| Teclado | vitest | Tab/Enter/Esc na xanela |
| Idiomas e axuda | Go | `i18n_test.go`; `help/` actualizada |

## 6. Sequencing
Versións → paquete → documento → clasificación → meter → migración → bindings → frontend (modelo, xanela, integración, controlador) → idiomas e axuda → verify.

## 7. Risks
- **Migración (alto):** mitigado con versión previa, marca idempotente e test.
- **Copia completa grande:** límites de tamaño; progreso só como estado (operación local).
- **Rutas no zip (seguridade):** rexéitanse rutas absolutas, `..` e ligazóns; límite de entradas e de tamaño descomprimido.

## Tasks
| # | Tarefa | Depende | Paralela | Done-check | Spec |
|---|---|---|---|---|---|
| T1 | Versións para desfacer (gardar, restaurar, podar a 5) | — | si | `go test -run Version` verde | Regras comúns |
| T2 | Paquete `.lua`: escribir e ler con límites e rutas seguras | — | si | `go test -run Paquete` verde | Sacar/Meter, sen claves |
| T3 | Documento `.json` editable e `Settings.Instalacion` | — | si | `go test -run DocumentoSacar` verde | JSON editable |
| T4 | Clasificación de ficheiros, destinos e detección da volta | T2,T3 | non | `go test -run Clasificar` verde | Táboa de destinos, `.json` vellos |
| T5 | Aplicar «Meter» (todos os tipos, todo ou nada, versións) | T1,T4 | non | `go test -run 'Meter|Copia'` verde | Táboa, regras, erros |
| T6 | Migración do traballo solto | T1 | non | `go test -run Migracion` verde | Migración |
| T7 | Bindings, soltar ficheiros, plantilla por documento | T5,T6 | non | `go build` e `go vet` limpos | Sitio único |
| T8 | `trasladar/modelo.js` | — | si | `npm test -- trasladar` verde | Preselección, aviso produción |
| T9 | Xanela Trasladar | T7,T8 | non | test DOM verde | Sitio único, teclado |
| T10 | Integración en main.js e index.html (fóra Abrir/Gardar/Gardar como, sempre un proxecto) | T9 | non | grep sen botóns vellos; vitest verde | Modelo, entrada única |
| T11 | Controlador sen ficheiro vinculado; asistente adaptado | T10 | non | `controlador.test.js` verde | Fóra o vinculado |
| T12 | Textos en gl/es/en/pt | T9-T11 | non | `go test -run I18n` verde | 3.2 |
| T13 | Axuda e decisións | T12 | non | `help/` e `decisions.md` actualizados | 8.1-8.2 |
| T14 | Verify | todo | non | Definición de feito | — |
