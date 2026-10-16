---
type: plan
title: Plan — Deseño dos documentos
slug: deseno-documentos
timestamp: 2026-10-16T00:00:00Z
topic: sdd
status: approved (autopiloto, 2026-10-16)
---

# Plan — deseno-documentos

> Spec: [../specs/deseno-documentos.md](../specs/deseno-documentos.md) · Constitución v1.2.1

## Arquitectura
- **Modelo** (`edicion/plantilla/deseno.go`): `Deseno` (letra, letras por papel, tamaño,
  interliñado, escala, cores, barras, interruptores, elementos con parámetros e LaTeX
  propio). `Plantilla.Deseno *Deseno` (`json:"deseno,omitempty"`). Normalización e
  validación (`NormalizarDeseno`) e JSON Schema (`EsquemaDeseno`).
- **Catálogo de letras** (`edicion/latex/letras.go`): combinacións → familia por papel
  (corpo, títulos, etiquetas, matemáticas) con ficheiros de TeX Live ou de Lúa; substitución
  por `kpsewhich`; equivalentes de pdflatex. `TipografiaPreambulo` pasa a definir
  `\setmainfont`, `\setsansfont` e `\fontchip` xuntas (fase 0).
- **Xerador** (`lua/lua/deseno_latex.go`): `Deseno` → bloque dinámico (cores) + estilo
  por elementos. `estiloPorDefectoLatex` pasa a ser o xerado co deseño por defecto.
  En vista previa, cada elemento deixa marcas de posición (`\pdfsavepos`) nun ficheiro `doc.lpos`.
- **Corrector** (`deseno_corrector.go`): contraste WCAG, penalizacións de liñas soltas,
  lectura de `Overfull \hbox` do log.
- **Cores do logotipo** (`deseno_cores.go`): `image` da biblioteca estándar, histograma de cores saturadas.
- **Importar deseño** (`deseno_importar.go`): PDF → `pdffonts` + cores da primeira páxina
  rasterizada; DOCX → `word/styles.xml` e `word/theme`.
- **Frontend** (`frontend/src/deseno/`): `modelo.js` (barras, interruptores, variantes:
  funcións puras con vitest), `editor.js` (bloque «Deseño»), `inspector.js` (clic na vista previa).
- **IA**: campo `deseno` en `camposPlantilla`; `DESENO-SKILL.md` en `lua/lua/help/` e
  binding `EsquemaDeseno`.

## Tarefas
| # | Tarefa | Done-check |
|---|---|---|
| T1 | Fase 0: letra coherente en todos os papeis, matemáticas, tamaño real, aviso en plantilla completa | `TestTipografiaPapeis`, `TestTamanoNormalsize`, `TestPdffontsCombinacion` (salta sen xelatex) |
| T2 | Modelo `Deseno` + normalización + esquema | `TestNormalizarDeseno`, `TestEsquemaDeseno` |
| T3 | Catálogo de letras con substitución | `TestLetrasCatalogo` |
| T4 | Xerador LaTeX por elementos + LaTeX propio por elemento | `TestEstiloDesenoPorDefecto`, `TestElementoLatexPropio` |
| T5 | Interruptores (B/N, aforrar, lectura fácil) | `TestDesenoBrancoNegro`, `TestDesenoAforrar` |
| T6 | Corrector (contraste, liñas soltas, overfull) | `TestCorrectorContraste`, `TestCorrectorOverfull` |
| T7 | Cores do logotipo | `TestCoresDoLogo` |
| T8 | Marcas de posición na vista previa | `TestMarcasPosicion` |
| T9 | Importar deseño de PDF e DOCX | `TestImportarDesenoDocx`, `TestImportarDesenoPdf` (salta sen ferramentas) |
| T10 | Frontend: modelo (barras, interruptores, variantes) | `deseno/modelo.test.js` |
| T11 | Frontend: bloque Deseño, inspector, variantes | `deseno/editor.test.js` |
| T12 | IA: campo deseno, skill e esquema | `TestEsquemaPlantillaLevaDeseno` |
| T13 | Textos (gl/es/en/pt), axuda, decisións | `i18n_test.go` |
| T14 | Verify e compilar a app | Definición de feito |
