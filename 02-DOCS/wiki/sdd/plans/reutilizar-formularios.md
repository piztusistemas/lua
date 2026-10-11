---
type: plan
title: Plan — Proxectos lixeiros, ventá de formularios e solucións só co resultado
slug: reutilizar-formularios
timestamp: 2026-10-11T00:00:00Z
topic: sdd
status: approved (autopiloto, 2026-10-11)
---

# Plan — reutilizar-formularios

> Spec: [../specs/reutilizar-formularios.md](../specs/reutilizar-formularios.md) · Constitución v1.2.1

## Arquitectura
- **Catálogo común** (`catalogo_formularios.go`): os formularios de `dirBibliotecaFormularios()` e os cartafoles no `biblioteca_formularios.json` que está ao seu lado (o de fóra dos proxectos: consérvanse os cartafoles de antes). Bindings: `CatalogoFormularios`, `CrearCarpetaCatalogo`, `RenomearCarpetaCatalogo`, `EliminarCarpetaCatalogo`, `MoverFormularioCatalogo`, `ReutilizarNoProxecto`.
- **Copia ao catálogo ao gardar**: `GardarFormulario`, `GardarDescricionFormulario` e `GardarIconaFormulario` chaman a `sincronizarFormularioCatalogo` se hai proxecto (actualiza o do mesmo id; se o do catálogo está en produción, entra coma copia).
- **Ventá** `frontend/src/formularios/catalogo.js` (botón + e «Engadir cronograma ou actas» da benvida).
- **Panel de proxectos**: crear sen ferramentas; tarxetas pregadas cunha cabeceira que desprega; fóra «Ferramentas…».
- **Columna**: «Quitar do proxecto…» no menú do botón dereito (substitúe a «Eliminar formulario» na columna; a biblioteca «⋯» segue igual).
- **Asistente**: `catalogo_ler`, `catalogo_mover` (crea o cartafol se non existe), `catalogo_cartafol_renomear`, `catalogo_cartafol_eliminar` (confirmar) en Go; `catalogo_reutilizar` no frontend.
- **Solucións**: `resultado-xunto` e `resultado-final` en `formulario_problemas.go` (texto do documento novo `FORM_RESULTADOS`).

## Tarefas
| # | Tarefa | Done-check |
|---|---|---|
| T1 | Solucións só co resultado | `TestSolucionsSoResultado`, `problemas-modelo.test.js` |
| T2 | Catálogo común e copia ao gardar | `TestCatalogoCartafolesEReutilizar`, `TestFormularioGardadoNoProxectoChegaAoCatalogo`, `TestReutilizarSenProxectoFalla` |
| T3 | Panel de proxectos lixeiro | `proxectos.test.js` |
| T4 | Ventá de formularios | `catalogo.test.js` |
| T5 | Quitar do proxecto na columna | `accions-ferramenta.js` + main |
| T6 | Ferramentas do asistente | `TestAsistenteMoveFormularioACartafolNovo`, `ferramentas.test.js` |
| T7 | Textos e documentación | `i18n_test.go`, README |
| T8 | Verify e compilar | Definición de feito |
