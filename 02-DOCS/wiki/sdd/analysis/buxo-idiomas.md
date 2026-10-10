---
type: analysis
title: Análise — buxo-idiomas
description: Contraste de constitución, spec, plan e tarefas antes de implementar.
tags: [sdd, analysis]
timestamp: 2026-10-10T00:00:00Z
topic: sdd
slug: buxo-idiomas
---

# Análise — buxo-idiomas

**GATE: PASS** — 0 CRITICAL · 0 HIGH · 1 MEDIUM · 1 LOW

Artefactos: [constitución v1.2.0](../constitution.md) · [spec](../specs/buxo-idiomas.md) · [plan e tarefas](../plans/buxo-idiomas.md)

## Mapa de cobertura

| REQ | Requisito da spec | Plan | Tarefas | Estado |
|---|---|---|---|---|
| R1 | Buxo en castelán desde Lua (Acept. 1) | §2-§3 | T001, T002, T006, T009 | covered |
| R2 | Erro de conversión en inglés (Acept. 2) | §3, §5 | T005 | covered |
| R3 | Sen Lua → galego (Acept. 3) | §3, §5 | T001 | covered |
| R4 | Test que detecta textos sen traducir (Acept. 4) | §5 | T007 | covered |
| R5 | Galego idéntico (Acept. 5) | §0 | T004 (tests existentes) | covered |
| R6 | Panel en portugués e `buxo.log` en galego (Acept. 6) | §2-§3 | T003, T004 | covered |
| R7 | Diálogos de ficheiros no idioma de Lua | §4 | T006 | covered |
| R8 | Sen cambio en quente | §4 | (por deseño, sen tarefa) | covered |
| R9 | Historial do panel ao reabrir no idioma activo | §4 | T004 (non o nomea) | AMBIGUOUS (G2) |
| R10 | Respostas para a IA en galego | §0 | T005 (só migra os visibles) | covered |

Sen DRIFT.

## Achados

| # | Sev. | Tipo | A (onde) | B (onde) | Conflito | Resolver en |
|---|---|---|---|---|---|---|
| G1 | MEDIUM | Contradición interna | plan §2, «Erros visibles»: «O texto `Error()` segue en galego para os logs e para a IA» | plan §3 e plan irmán §3: `Error()` → idioma activo; `idioma.Galego(err)` para o log | Quedou unha frase do primeiro borrador. Un implementador que lea §2 faría o contrario ca §3. | `plan` (corrixir a frase de §2) |
| G2 | LOW | Portador incompleto | plan §4 (o historial garda clave + argumentos) | T004 Interfaces (só `Liña{Hora, Nivel, Texto}`) | Un implementador illado pode gardar no historial o texto xa traducido, e o panel repintaríase mal tras un cambio. Hoxe non hai cambio en quente, así que o efecto é mínimo. | `tasks` (engadir o historial ás Interfaces de T004) |

Constitución: sen violacións. Dependencia entre plans (T001 ← mensaxes-go-idiomas T002; T009 ← mensaxes-go-idiomas T004) escrita nas dúas táboas.

## Ruta recomendada

- **plan:** G1, unha frase.
- **tasks:** G2, unha liña nas Interfaces de T004.

## Decisión do usuario
- 2026-10-10 — Achados aceptados tal como están («acepto e programar xa»). Implementación: en caso de dúbida mana o §3 dos plans.
