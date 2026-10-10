---
type: progress
title: Progreso — buxo-idiomas
description: Rexistro de implementación por tarefa (só engadir).
tags: [sdd, progress]
timestamp: 2026-10-10T00:00:00Z
topic: sdd
slug: buxo-idiomas
---

# Progreso — buxo-idiomas

Fallbacks: sen config.yaml (ordes da constitución); sen subaxentes.

## T001 — 2026-10-10
- status: complete
- red: `go test -run TestIdiomaArrinque .` → `undefined: configurarIdioma`
- files: buxo/idioma_test.go

## T002 — 2026-10-10
- status: complete
- green: T001 verde; `go test ./...` e `go vet` en buxo verdes
- files: buxo/idioma.go, buxo/idiomas/{gl,es,en,pt}.json (clave inicial `ui.construir`), main.go (chamada tras flag.Parse)
