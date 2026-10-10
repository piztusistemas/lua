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

## T005 — 2026-10-10
- status: complete
- red (mutacións): clave inexistente, galego distinto do JSON, galego non literal → «3 problemas en 3 chamadas»; `%d` engadido en pt → «verbos de formato … (mesma orde)»
- green: sen mutacións, verde
- files: lua/lua/i18n_test.go
- decision: os verbos `%s %d %v %q %w` compáranse en orde; as variables `{x}` do frontend seguen comparándose ordenadas

## T006 — 2026-10-10
- status: complete
- red: TestPistaPaqueteLatexEnPortugues (texto en galego) e TestAvisoCambioDeMotorEnIngles (aviso en galego)
- green: os dous verdes e executados (non saltados); `go test ./...` en edicion e lua/lua verde; garda T005 verde
- files: edicion/documento/latex.go, edicion/plantilla/composicion.go, lua/lua/{debuxos_latex,informe_cronograma,latexdoc}.go, idiomas/*.json, idioma_mensaxes_test.go
- claves: 12 (go.latex.*, go.plantilla.variable_indefinida), 20 sitios
- decision: traducións por situación (ver decisions.md); F1 da análise aceptado: non hai test da mensaxe «non se atopou %s» (precisa un equipo sen motor LaTeX)

## T007 — 2026-10-10
- status: complete
- green: `go test ./...` en edicion e lua/lua verde; garda verde
- claves: 6 (go.maxima.*, go.problemas.*), 7 sitios
- F3 da análise: `edicion/xerar/problemas.go:301` busca «timeout»/«tempo»; a mensaxe de Maxima sen resposta non contén ningún, nin antes nin nas traducións (evitados a propósito) → comportamento igual
- nota: `probas.go:154` («non se cumpre » + c, concatenado) non é da lista; queda en galego

## T008 — 2026-10-10
- status: complete
- green: `go test ./...` e `go vet` en lua/lua verdes; `GOOS=windows go vet` limpo (linux non se pode comprobar desde macOS: Wails precisa cgo)
- claves: 1 (go.instalar.sen_pkexec), 2 sitios (installer.go, maxima.go)
