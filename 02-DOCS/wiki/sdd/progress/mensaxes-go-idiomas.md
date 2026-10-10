---
type: progress
title: Progreso — mensaxes-go-idiomas
description: Rexistro de implementación por tarefa (só engadir).
tags: [sdd, progress]
timestamp: 2026-10-10T00:00:00Z
topic: sdd
slug: mensaxes-go-idiomas
---

# Progreso — mensaxes-go-idiomas

Fallbacks: sen `02-DOCS/wiki/sdd/config.yaml` (ordes da constitución: gofmt, go vet, go test, npm test); sen subaxentes (revisión feita na sesión).

## T001 — 2026-10-10
- status: complete
- red: `cd edicion && go test ./idioma/` → build failed (paquete inexistente)
- files: edicion/idioma/idioma_test.go
- nota: commit xunto con T002 para non deixar `main` en vermello (constitución 2.2)

## T002 — 2026-10-10
- status: complete
- green: `go test -race ./idioma/` ok; `go vet` e `gofmt -l` limpos; `go test ./...` en edicion verde
- triangulation: erro traducido que envolve outro → `Galego()` todo en galego, `errors.Is` chega dentro
- files: edicion/idioma/idioma.go
- decision: `Restablecer()` exportada para que os tests de Lua e Buxo poidan volver ao estado inicial
- blocker: none
