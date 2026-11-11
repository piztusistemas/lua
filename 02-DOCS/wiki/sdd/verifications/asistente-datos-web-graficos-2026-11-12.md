---
type: verification
title: Verificación — asistente-datos-web-graficos — 2026-11-12
description: Gate de probas antes do commit e da release 26.11.1 (web e gráficos no asistente de datos).
tags: [sdd, verification, ftd]
timestamp: 2026-11-12T00:30:00Z
topic: sdd
slug: asistente-datos-web-graficos
source_state: 33621ee (dirty: cambios sen commit da funcionalidade e da versión 26.11.1)
---

# Verificación — asistente-datos-web-graficos — 2026-11-12

Carril FTD: as comprobacións son o checklist de `02-DOCS/wiki/ftd/asistente-datos-web-graficos.md`.

## Gate (config.yaml), unha execución nova tras o último cambio
- `gofmt -l .` → baleiro
- `go vet ./...` → limpo
- `go test ./...` → ok lua-wails (165,6 s), ok lua-wails/grafico
- `go test -run 'I18n|Idioma' ./...` → ok (claves en gl, es, en e pt)
- `npm test` → 38 ficheiros, 583 tests en verde

## Comprobacións do FTD
- [x] O ámbito ten a web — TestAmbitoFormularioDatosFerramentas
- [x] Formularios web e POST — TestFormulariosDeHTML*, TestWebEnviarFormulario* (servidor local; os enderezos locais rexéitanse)
- [x] Porta do frontend e produción negada — ferramentas-formulario-datos.test.js
- [x] pgfplots en PDF, Word/ODT e plantillas — TestPgfplotsNosPreambulos, TestGraficoPgfplotsCompila, TestFormularioConGraficoPgfplotsCompilaPDF (pdflatex e xelatex) e páxina renderizada revisada
- [x] Calquera fonte — TestTextoDeHTMLSeparaAsCelasDasTaboas (vermello antes), TestTextoWebCSVeBinarios
- [x] Fluxo real contra datosclima.es (test temporal, 2026-11-11)
- [ ] Pedido completo cunha IA de verdade dentro da app — SEN VERIFICAR (precisa a clave da API e a app)
- [ ] Gráfico na vista previa do editor en modo deseño — SEN VERIFICAR (no PDF si)

## Capas non executadas
- **NO-APLICA** — `edicion/` e `buxo/`: non se tocaron.
- **HERRAMIENTA-AUSENTE** — staticcheck, govulncheck: non están no gate de config.yaml; non se executaron.
- **SUSTITUIDA** — o comportamento coa IA: probouse cada ferramenta por separado e o fluxo real con código, non co modelo. Non detecta que a IA non siga as regras do prompt.

## Achados descartados
- ninguno

## Veredicto: FAIL — 2 comprobacións manuais abertas (IA real na app, vista previa). O gate automático está enteiro en verde. O usuario pediu publicar a 26.11.1 sabendo que estas dúas non se probaron (resposta do 2026-11-11).
