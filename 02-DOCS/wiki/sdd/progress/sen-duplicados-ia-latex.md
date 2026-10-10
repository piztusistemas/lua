---
type: progress
title: Progreso — sen-duplicados-ia-latex
description: Rexistro de implementación por tarefa (só engadir).
tags: [sdd, progress]
timestamp: 2026-10-10T00:00:00Z
topic: sdd
slug: sen-duplicados-ia-latex
---

# Progreso — sen-duplicados-ia-latex

Autopiloto (aprobación do usuario: «Autopiloto»). Rama `refactor-duplicados` en `.worktrees/refactor-duplicados`.

## T001 — 2026-10-10
- status: complete
- caracterización IA: 16 casos (3 provedores × simple/conversa × max 0/1234, erros de configuración e HTTP 429) en `lua/lua/testdata/ia_caracterizacion.golden.json`; estable en execucións repetidas
- achado: a chamada simple envía o mesmo ca unha volta de conversa cunha mensaxe e sen ferramentas (comprobado nos 6 casos); só difire en que limpa os valados ``` da resposta

## T002 — 2026-10-10
- status: complete
- caracterización LaTeX: 141 entradas (tipografía, motores, pistas e erros de LaTeX) en `lua/lua/testdata/latex_caracterizacion.golden.json`; estable

## T003 — 2026-10-10
- status: complete
- red: `ia.IAMensaxe` sen campo `Imaxes` (build failed)
- green: TestIAMensaxeLevaImaxesEnJSON verde; `go test ./...` en edicion verde
