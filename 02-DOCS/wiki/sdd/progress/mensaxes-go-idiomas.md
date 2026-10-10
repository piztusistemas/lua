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

## T003 — 2026-10-10
- status: complete
- red: `go test -run TestIdiomaGo .` → 4 fallos esperados (idioma gl ao arrincar, dicionario sen rexistrar, gardar Opcións, idioma de Piztu)
- files: lua/lua/idioma_go_test.go

## T004 — 2026-10-10
- status: complete
- green: T003 verde; `go test ./...` en lua/lua verde
- files: lua/lua/idioma_go.go, app.go (startup), settings.go (SaveSettings)
- decision: o asistente garda co mesmo SaveSettings, así que abondan dous puntos (arrinque e SaveSettings), non tres
- decision: sen sync.Once ao rexistrar os dicionarios; cun Once, un test que restablece o tradutor deixaba o Go sen dicionarios (fallo visto na suite completa)
- check: PIZTU_CONTEXTO non existe nos tests → os tests seguen en galego
