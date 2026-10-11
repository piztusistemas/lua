---
type: plan
title: Plan — Engadir traballos á bóveda
slug: traballos-boveda
timestamp: 2026-10-11T00:00:00Z
topic: sdd
status: approved (autopiloto, 2026-10-11)
---

# Plan — traballos-boveda

> Spec: [../specs/traballos-boveda.md](../specs/traballos-boveda.md) · Constitución v1.2.1 (5.2)

## Arquitectura
- `boveda_traballos.go`: `EngadirTraballoBoveda(req, pdfBase64, nome)` escolle a bóveda con `bovedaDe` (proxecto → conectada ao tipo de documento → crea a do proxecto con `raw/`, `wiki/`, `pdf/`), compón o Markdown con `markdownInformeConPlantilla` (o mesmo da exportación) e escribe `wiki/traballos/<nome>.pdf` e despois `<nome>.md` (frontmatter + ligazón `[[<nome>.pdf|PDF]]`), e engade unha liña a `wiki/log.md`. Bloquea produción mirando a petición e o formulario gardado.
- `Proxecto.BovedaAutomatica` (lista de formularios) con `BovedaAutomaticaDe` e `FixarBovedaAutomatica` (rexeita activar en produción).
- Frontend: `controlador.engadirABoveda(nome, boton)` (xera antes se non está ao día) e o gancho `ui.despoisDeXerar`; en `main.js` o botón «Bóveda» e a marca «sempre» (só formularios; desactivados en produción; agochados na xanela de resultado).

## Tarefas
| # | Tarefa | Done-check |
|---|---|---|
| T1 | Engadir á bóveda e crear a do proxecto | `TestTraballoCreaABovedaDoProxectoESubstitue`, `TestTraballoVaiABovedaConectada`, `TestTraballoCronograma` |
| T2 | Bloqueo de produción | `TestTraballoProduccionBloqueado` |
| T3 | Marca «sempre» | `TestBovedaAutomaticaPorFormulario` |
| T4 | Controlador e botón | `controlador.test.js` (describe «bóveda») |
| T5 | Textos, axuda, verify e compilar | `i18n_test.go`, README |
