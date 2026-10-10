---
type: analysis
title: Análise — mensaxes-go-idiomas
description: Contraste de constitución, spec, plan e tarefas antes de implementar.
tags: [sdd, analysis]
timestamp: 2026-10-10T00:00:00Z
topic: sdd
slug: mensaxes-go-idiomas
---

# Análise — mensaxes-go-idiomas

**GATE: PASS** — 0 CRITICAL · 0 HIGH · 2 MEDIUM · 3 LOW

Artefactos: [constitución v1.2.0](../constitution.md) · [spec](../specs/mensaxes-go-idiomas.md) · [plan e tarefas](../plans/mensaxes-go-idiomas.md)

## Mapa de cobertura

| REQ | Requisito da spec | Plan | Tarefas | Estado |
|---|---|---|---|---|
| R1 | Sen xelatex, aviso en inglés (Acept. 1) | §4 lista 1-2, §5 | T006 | AMBIGUOUS (ver F1) |
| R2 | Ficheiro non válido en castelán (Acept. 2) | §4 lista 33-37, §5 | T010 | covered |
| R3 | Paquete LaTeX en portugués co nome (Acept. 3) | §4 lista 4, §5 | T006 | covered |
| R4 | Galego idéntico (Acept. 4) | §0, §3, §5 | T001, T002, T006-T012 | covered |
| R5 | Test garda de traducións e variables (Acept. 5) | §5 | T005 | covered |
| R6 | Lista dentro das sete situacións (Acept. 6) | §4 | T005 (só por convención de claves) | AMBIGUOUS (F4) |
| R7 | IA sen clave en inglés (Acept. 7) | §4 lista 67, §5 | T012 | covered |
| R8 | Lista no plan para aprobar (Acept. 8) | §4 | — (aprobada) | covered |
| R9 | Cambio de idioma: mensaxes novas no idioma novo | §3 (tradución ao ler) | T001, T003, T004 | covered |
| R10 | Partes variables intactas; saída externa sen traducir | §0, §3 | T001 | covered |
| R11 | Yang sen cambios | §0, §7.4 | T014 | covered |

Sen DRIFT: toda tarefa traza a un requisito.

## Achados

| # | Sev. | Tipo | A (onde) | B (onde) | Conflito | Resolver en |
|---|---|---|---|---|---|---|
| F1 | MEDIUM | Cobertura | spec Acept. 1 («equipo sen xelatex») | tarefas T006 (test do aviso de cambio de motor) | O test proposto proba a mensaxe 2 (cambio automático de motor), non a 1 («non se atopou %s. Instala…»), que é a que sae cando falta xelatex. | `tasks` (axustar o test de T006) |
| F2 | MEDIUM | Contradición | spec Non-goals («os rexistros internos e os ficheiros de log») | plan §2-§3 (o erro tradúcese ao lelo) | Calquera log que escriba un erro da lista sairá no idioma activo. Medido: `lua/lua` non ten `log.Printf` con erros (0), así que hoxe non afecta, pero a regra non está escrita. | `plan` (dicir que os logs usan `idioma.Galego`) ou aceptar |
| F3 | LOW | Risco sen tarefa | plan §7.2 (comparacións de texto) | tarefas (ningunha o revisa) | Existe unha: `edicion/xerar/problemas.go:301` busca «timeout»/«tempo» no erro de Maxima. Ningún dos 74 textos os contén, así que non rompe, pero ninguén o revisa. | `tasks` (nota en T007) |
| F4 | LOW | Ambigüidade | spec Acept. 6 | T005 | O garda non comproba que cada clave pertenza a unha das sete situacións; depende da convención `go.<area>`. | `tasks` (o garda pode exixir un prefixo de área válido) ou aceptar |
| F5 | LOW | Duplicación | constitución 3.2 («gl.json é a fonte») | plan §2 (o galego tamén queda no código) | O mesmo texto vive en dous sitios; o garda T005 obriga a que coincidan, coma no frontend. | aceptar |

Constitución: sen violacións (1.3 en `edicion`; 2.1-2.3 nas comprobacións; 3.1; 4.1b `main`).
Portadores para implementadores illados: §0 presente; Interfaces en T002, T004 e T006-T012.

## Ruta recomendada

- **tasks:** F1 (test de T006 coa mensaxe 1), F3 (revisar `problemas.go:301` en T007), F4 opcional.
- **plan:** F2, unha liña en §3: os logs internos usan `idioma.Galego(err)`.
- Ou aceptar conscientemente F2-F5 e implementar.

## Decisión do usuario
- 2026-10-10 — Achados aceptados tal como están («acepto e programar xa»). Implementación: en caso de dúbida mana o §3 dos plans.
