---
type: plan
title: Plan — Un só cliente de IA e unhas soas funcións LaTeX
description: Plan de estrutura para quitar as copias do cliente de IA e das funcións auxiliares de LaTeX de Lúa, deixándoas só en edicion.
tags: [sdd, plan, refactor]
timestamp: 2026-10-10T00:00:00Z
topic: sdd
slug: sen-duplicados-ia-latex
status: approved (autopiloto, 2026-10-10)
---

# Plan — Un só cliente de IA e unhas soas funcións LaTeX

> Spec: [../specs/sen-duplicados-ia-latex.md](../specs/sen-duplicados-ia-latex.md) · Constitution: [../constitution.md](../constitution.md) · Status: approved in autopilot

## 0. Global Constraints
- **Comportamento idéntico:** `testdata/ia_caracterizacion.golden.json` (13+3 casos: peticións, cabeceiras, corpos, resultados e erros dos tres provedores) non cambia nin un byte; o preámbulo de tipografía e as pistas de LaTeX son as mesmas.
- **Sen tocar Yang:** `yang/` non se edita; a súa suite dá a liña base (build failed por `pathHelperDirsMacOS` + `TestSetDecimaisRoundsFloatingResults`).
- **APIs de `edicion` compatibles:** nada que xa exportan `edicion/ia`, `edicion/latex` ou `edicion/documento` cambia de firma.
- **Tipos que cruzan a fronteira de Wails:** en Lúa quedan coma alias (`type X = paquete.Y`), non copias (`edicion/doc.go`).
- **Constitución:** 1.3, 2.1-2.3, 3.1, 4.3 (sen `Co-Authored-By`). Traballo na rama `refactor-duplicados` (`.worktrees/refactor-duplicados`), porque outra sesión usa `main`.

## 1. Context & constraints
- Medido o 2026-10-10: `lua/lua/ia_conversa.go` é idéntico a `edicion/ia/conversa.go` agás o nome do campo de imaxes; as chamadas simples de `ia_client.go` envían exactamente o mesmo ca unha volta de conversa cunha soa mensaxe e sen ferramentas, e só se diferencian en que limpan os valados ``` da resposta. As 13 funcións de `lua/lua/latexdoc.go` son idénticas ás de `edicion/documento/latex.go`.
- `edicion/latex` é a «infraestrutura LaTeX común» e non importa nada de `edicion` (sen ciclos).

## 2. Architecture
```text
lua/lua ──alias de tipos + configDe(Settings)──> edicion/ia (Turno, tipos)   ← único cliente
lua/lua ──adaptador de unha liña por función──> edicion/latex (tipografía, motores, pistas)
edicion/documento ─────────────────────────────> edicion/latex                ← sen copia propia
```
- **`edicion/ia`** gaña o campo `Imaxes []string` en `IAMensaxe` (o que manda Buxo pola API), co mesmo JSON ca en Lúa.
- **`lua/lua/ia_conversa.go`** queda en: alias dos tipos, `configDe(Settings) ia.Config` (resolve o máximo de tokens: Anthropic 8192 se non hai valor; tempo de Opcións), `chamarIAConversa` → `ia.Turno`, e `chamarIA` → `ia.Turno` cunha mensaxe + `limparValadosMarkdown`. `ia_client.go` perde os tres clientes; consérvanse `ProvedorIA` e `baseURLPorDefecto` como alias/adaptadores se se usan.
- **`edicion/latex`** recibe as 13 funcións exportadas (`EngineUnicodeNativo`, `TipografiaPreambulo`…) e o tipo `TipografiaOpts`; `edicion/documento` e Lúa usan esas. En Lúa, `latexdoc.go` queda só con `GeneratePDFResult` e adaptadores dunha liña.

**Decisión principal:** adaptadores finos en Lúa (o patrón de `edicion_plataforma.go`) en vez de cambiar centos de chamadas: o diff queda pequeno e revisable, e non hai lóxica duplicada.

## 3. Interfaces & contracts
```text
ia.Turno(c ia.Config, sistema, msgs []ia.IAMensaxe, ferr []ia.IAFerramenta) (ia.IATurno, error)   (xa existe)
configDe(s Settings) ia.Config      // Provedor, BaseURL, APIKey, Modelo, MaxTokens resolto, TimeoutSegundos
chamarIA(s, sistema, peticion) (string, error)        // = Turno(1 msg, sen ferramentas) + limparValadosMarkdown; non erra por truncado
latex.TipografiaOpts; latex.TipografiaPreambulo(t *TipografiaOpts, engine string) string; … (mesmos corpos)
```

## 4. Data model & flow
Sen datos novos. O JSON de `IAMensaxe`, `IATurno`, `IAFerramenta` e `TipografiaOpts` que viaxa ao frontend e se garda nas conversas non cambia (alias).

## 5. Testing strategy
| Aceptación | Nivel | Que |
|---|---|---|
| 1, 2 | caracterización (Lúa) | `TestCaracterizacionClienteIA` contra a referencia gravada antes de refactorizar |
| 3 | caracterización (Lúa) | `TestCaracterizacionLatex`: preámbulos e pistas para un conxunto fixo de entradas, gravados antes |
| 4 | busca | ningún `func turno*`, `chamarGemini/Anthropic/OpenAICompatible`, `postJSONIA` nin as 13 funcións LaTeX con corpo en `lua/lua` |
| 5 | suites | `go test ./...` en edicion, lua/lua, buxo e instalador; vitest de Lúa e Buxo; Yang = liña base |

## 6. Sequencing
1. Caracterizacións gravadas (IA feita; LaTeX). 2. IA en `edicion`: campo `Imaxes`. 3. Lúa usa `edicion/ia`. 4. LaTeX a `edicion/latex` e `documento` úsao. 5. Lúa usa `edicion/latex`. 6. Verificación.

## 7. Risks
1. **Tests vellos de `ia_client_test.go` proban funcións que desaparecen** (formato de fío por provedor). *Mitigación:* a caracterización cobre o mesmo con máis detalle; elimínanse eses tests (non hai código que probar) e quedan os de validación e despacho, adaptados.
2. **Exportar en `edicion/latex` choca con nomes existentes.** *Mitigación:* revisar antes; `go vet` e Yang.
3. **Imaxes nas mensaxes:** o campo privado `imaxes` pasa a `ImaxesIA`. *Mitigación:* a caracterización inclúe imaxes en mensaxes e resultados.

## Tasks
<!-- generated by tasks on 2026-10-10; IDs are stable, do not renumber -->
| ID | [P] | Task | Done-check | Depends-on | Trace |
| --- | --- | --- | --- | --- | --- |
| T001 |  | Gravar caracterización da IA | `CARACTERIZAR=1 go test -run TestCaracterizacionClienteIA` → 16 casos; sen a variable, verde | — | Acept. 1, 2 |
| T002 |  | Gravar caracterización de LaTeX | idem con `TestCaracterizacionLatex` | — | Acept. 3 |
| T003 |  | Engadir `Imaxes []string` a `ia.IAMensaxe` | `go test ./ia/` en edicion verde | — | plan §2 |
| T004 |  | Lúa usa `edicion/ia` (alias, `configDe`, `chamarIA`, `chamarIAConversa`); borrar copias | T001 verde; `go test ./...` en lua/lua verde; busca de Acept. 4 baleira para a IA | T001, T003 | Acept. 1, 2, 4 |
| T005 |  | Mover as 13 funcións e `TipografiaOpts` a `edicion/latex`; `documento` úsaas | `go test ./...` en edicion verde; Yang = liña base | T002 | Acept. 3, 5 |
| T006 |  | Lúa usa `edicion/latex` (alias + adaptadores); borrar copias | T002 verde; `go test ./...` en lua/lua verde; busca de Acept. 4 baleira para LaTeX | T005 | Acept. 3, 4 |
| T007 |  | Verificación completa | suites de edicion, lua/lua, buxo, instalador e vitest verdes; Yang = liña base; gofmt/vet limpos | all | Acept. 5 |

**Previsión de revisión:** −1 200/+300 liñas aprox. (sobre todo borrado), risco medio (IA real non probada contra os provedores), entrega nun commit por tarefa na rama `refactor-duplicados`, e fusión con `main` só tras a túa revisión.
