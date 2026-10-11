---
type: verification
title: Verificación — Trasladar
slug: trasladar
timestamp: 2026-10-11T00:00:00Z
topic: sdd
verdict: pass (con límites anotados)
---

# Verificación — Trasladar (2026-10-11)

## Ordes
| Orde | Resultado |
|---|---|
| `gofmt -l .` (lua/lua) | baleiro |
| `go vet ./...` (lua/lua) | limpo |
| `go test ./... -count=1` (lua/lua) | ok (124 s) e `grafico` ok |
| `npx vitest run` (lua/lua/frontend) | 33 ficheiros, 492 tests, todos verdes |
| `wails3 generate bindings` + `npx vite build` | compila |

`edicion/` e `buxo/` non se tocaron.

## Criterios de aceptación
| Criterio | Proba |
|---|---|
| Sacar proxecto → .lua; meter noutro equipo → igual (bóveda incluída) | `TestPaqueteProxectoIdaEVolta` |
| JSON editable → Actualizar e Desfacer | `TestDocumentoVoltaActualizaEDesfai` |
| Engadir como copia non cambia o orixinal | `TestDocumentoVoltaComoCopia` |
| Formulario con datos → proxecto novo + biblioteca; sen datos → biblioteca | `TestMeterFormularioConESenDatos` |
| Aviso de produción ao sacar | `TestSacarTenDatosProducion`, `trasladar.test.js` |
| Sen claves en ningún ficheiro | `TestPaqueteSenClaves` |
| `.json` vellos entran | `TestClasificarFormatosVellos` |
| Tres cursos, desmarcar un → dous proxectos | `TestMeterVariosCursos` |
| Copia completa e recuperar o estado anterior | `TestCopiaCompletaRestaurarEDesfacer` |
| Ficheiro danado non cambia nada | `TestMeterDanadoNonCambiaNada`, `TestPaqueteRutasPerigosas` |
| Migración a «O meu traballo», unha vez | `TestMigracionTraballoSolto`, `TestMigracionSenNadaSoMarca` |
| Entrada única (fóra Abrir/Gardar/Gardar como) | `index.html` sen `btnOpen`/`btnSave`/`btnSaveAs`; os menús de formularios e plantillas abren Trasladar |
| «Gardado en Lúa», nunca «sen gardar» | `controlador.test.js`; `main.js` sen `status.unsaved` |
| Teclado (Tab, frechas nas lapelas, Esc) | `trasladar.test.js` (Esc); lapelas con `role=tab` e frechas |
| Catro idiomas e axuda | `i18n_test.go` verde; `lua/README.md` (sección Trasladar) |
| Nunca se pisa un documento do proxecto aberto | `TestMeterNonPisaODocumentoDoProxectoAberto` (mata o mutante que quita a garda) |

## Límites (o que non se probou)
- Non se probou a xanela real de Wails: soltar ficheiros (`EnableFileDrop`), os diálogos nativos de gardar e abrir e o arranque coa migración comprobáronse por código e por tests unitarios, non nunha execución da app.
- O teclado completo (Tab por toda a xanela) non ten test automático.
