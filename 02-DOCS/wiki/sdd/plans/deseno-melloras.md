---
type: plan
title: Plan — Melloras do deseño dos documentos
slug: deseno-melloras
timestamp: 2026-10-16T23:00:00Z
topic: sdd
status: approved (autopiloto pedido polo usuario, 2026-10-16)
---

# Plan — deseno-melloras

> Orixe: [análise crítica](../analysis/deseno-documentos.md) · Continúa: [deseno-documentos](deseno-documentos.md) · Constitución v1.2.1

## Decisións de alcance
- Atkinson Hyperlegible non se embebe (licenza propia, non OFL): «Lexible» segue con TeX Gyre Schola.
- Embébense (OFL): Fira Sans (4 estilos) + Fira Math, Libertinus Serif (4) + Libertinus Math. Escríbense ao carón do .tex só as que usa o documento.
- A cancelación da vista previa mata o proceso de LaTeX daquela compilación; todas as compilacións de informes levan tempo máximo.
- Cabeceira e pé visuais: tres zonas (esquerda, centro, dereita) con texto, variables e `@logo`/`@imaxe`. Van despois da maqueta (mandan sobre ela) só se se enchen.
- Portada: modelos con nome; o LaTeX da plantilla (`portada`) segue mandando se existe.
- F1 (proba con docentes) non se pode automatizar: déixase o protocolo escrito.

## Tarefas
| # | Tarefa | Done-check |
|---|---|---|
| T1 (A1) | Estudio de deseño: o deseño ocupa a súa propia columna ancha, vista previa máis grande, só o deseño aberto de primeiras; sen nome ao gardar abre Identidade | `editor.test.js`, captura |
| T2 (A2) | Vista previa ao abrir, salta á páxina da parte cambiada, progreso + cancelar; tempo máximo en LaTeX | `TestCancelarVistaPrevia`, `TestLatexTempoMaximo`, `editor.test.js` |
| T3 (A3) | Inspector visible: resalte e nome da parte baixo o rato | `inspector` en `modelo.test.js` (caixas), `editor.test.js` |
| T4 (A4) | Nomes: «Carácter do documento», «Letra de cada parte» | i18n |
| T5 (B1) | Portada por modelos (cor, sobria, oficial, sen portada) con posición do logotipo | `TestPortadaModelos` |
| T6 (B2) | Cabeceira e pé visuais (tres zonas, variables, logotipo, número de páxina) | `TestCabeceiraPeDeseno` |
| T7 (B3) | Word/ODT: cabeceira e pé do deseño se o bloque de Word está baleiro | `TestOfimaticaCabeceiraDoDeseno` |
| T8 (C1) | Embeber Fira e Libertinus (OFL) e escribir só as usadas | `TestFontesEmbebidas`, `TestPdffontsModerna` |
| T9 (C2) | Mostras «Aa» coa letra real na interface (@font-face) | captura |
| T10 (D1) | Kit do centro: gardar/abrir o deseño nun ficheiro `.luadeseno` e aplicalo a todas as plantillas propias | `TestAplicarDesenoATodas`, `TestFicheiroDeseno` |
| T11 (D2) | Barras reversibles: lembran os valores de partida | `modelo.test.js` |
| T12 (E1) | Comprobar o LaTeX dunha parte nun documento mínimo, co erro ao carón | `TestComprobarElementoDeseno` |
| T13 (E2) | Teclado: frechas na escolla de letra e de modelo de portada | `editor.test.js` |
| T14 (F2) | Arranxar `task dev` (script `build:dev`) | `task dev` arrinca |
| T15 (F1) | Protocolo de proba con docentes | `02-DOCS/wiki/sdd/analysis/deseno-proba-docentes.md` |
| T16 | Textos gl/es/en/pt, README, decisións, verify, compilar | Definición de feito |
