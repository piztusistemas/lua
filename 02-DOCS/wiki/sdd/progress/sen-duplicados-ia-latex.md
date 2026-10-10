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

## T004 — 2026-10-10
- status: complete
- green: TestCaracterizacionClienteIA verde (16 casos idénticos á referencia); `go test ./...` en lua/lua e edicion verdes
- cambios: Lúa usa edicion/ia (alias de tipos, configDe, chamarIA = 1 volta + limpeza de valados, chamarIAConversa = ia.Turno); `imaxeIA = ia.Imaxe`; novo `ia.Config.Validar()` para `comprobarIA`
- tests: os que chamaban ás funcións internas borradas (formato de fío por provedor, imaxes, asistente) pasan agora pola entrada pública chamarIA/chamarIAConversa coas mesmas comprobacións; ningún eliminado
- resultado: ia_client.go + ia_conversa.go de Lúa pasan de ~900 a 180 liñas, sen código de envío propio

## T005 — 2026-10-10
- status: complete
- 13 funcións + 9 dependencias movidas de edicion/documento/latex.go a edicion/latex/axudas.go (exportadas); documento queda con adaptadores dunha liña
- green: `go test ./...` en edicion verde; Yang (con -modfile apuntando a este edicion) igual á liña base

## T006 — 2026-10-10
- status: complete
- Lúa: latexdoc.go deixa as 13 copias (22 bloques) e queda con GeneratePDFResult e adaptadores dunha liña a edicion/latex; novo `latex.PresetsFonte()` para o test que percorría o mapa
- green: TestCaracterizacionLatex verde (141 entradas idénticas)

## T007 — 2026-10-10
- status: complete
- check: go vet e go test verdes en edicion, lua/lua e buxo; vitest de Lúa 487/487; Yang igual á liña base (con -modfile cara a este edicion)
