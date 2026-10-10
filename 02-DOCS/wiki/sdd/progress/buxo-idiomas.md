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

## T003 — 2026-10-10
- status: complete
- red: TestRexistroIdiomas non compila coa firma vella de Rexistro (Info/Aviso/Erro sen clave)

## T004 — 2026-10-10
- status: complete
- green: TestRexistroIdiomas verde (panel en pt, buxo.log en galego, erro traducible en galego no ficheiro); `go test ./...` e `go vet` en buxo verdes
- files: buxo/rexistro.go, construir.go, procurar.go, mestre.go, vision.go, app.go, idiomas/*.json
- claves: 36 formatos `rex.*` (38 chamadas)
- decision: o historial garda a liña xa traducida (non clave + argumentos, como dicía o plan §4): Buxo non cambia de idioma en quente, así que o resultado é o mesmo e o código é máis simple (G2 da análise, aceptado)

## T005 — 2026-10-10
- status: complete
- red: TestErroConversionEnIngles (motivoVision devolvía string; erro en galego)
- green: verde; `go test ./...`, `go vet` (tamén GOOS=windows) en buxo verdes
- claves: 20 `erro.*`, 7 `motivo.*`, 5 `detalle.*`, 3 `desc.*`, 3 `fallo.*`, 3 `conversor.*`
- decision: motivoVision devolve un erro traducible (nil = sen motivo) e `fallar` recibe un erro: así o panel e a cola van no idioma de Lua e buxo.log en galego; TestMotivoVision adaptado ao tipo novo, mesmas comprobacións
- decision: `desc` («páxina N de M») segue en galego na petición á IA; `descVisible` traducible no progreso, no rexistro e nos erros
- nota: os erros que só le a IA (ferramentas, recadro, «sen IA») seguen con fmt.Errorf en galego
