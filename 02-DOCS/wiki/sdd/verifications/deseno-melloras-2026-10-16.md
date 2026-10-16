---
type: verification
title: Verificación — deseno-melloras
slug: deseno-melloras
timestamp: 2026-10-16T23:12:00Z
topic: sdd
verdict: PASS (F1 pendente: precisa docentes)
---

# Verificación — deseno-melloras (2026-10-16)

| Comprobación | Resultado |
|---|---|
| `gofmt -l`, `go vet` (edicion, lua/lua) | limpos |
| `go test ./...` edicion e lua/lua | verdes |
| `npm test` (vitest) | 552/552 |
| Idiomas gl/es/en/pt | i18n verde |
| `task darwin:package` | `build/bin/lua.app` (16 out. 23:10) |
| `task dev` | arrinca (Vite listo e xanela dev aberta) |

| Tarefa | Evidencia |
|---|---|
| T1 estudio | `editor.test.js` (columnas, só deseño aberto, plantilla nova abre identidade), captura |
| T2 vista previa | `TestLatexTempoMaximo`, `TestCancelarVistaPrevia`, test de Cancelar no editor |
| T3 inspector visible | test de resalte, captura co nome da parte |
| T4 nomes | i18n |
| T5 portada | `TestPortadaModelos`, PDF reais (sobria, oficial) revisados |
| T6 cabeceira e pé | `TestCabeceiraPeDeseno`, PDF real revisado |
| T7 Word/ODT | `TestOfimaticaCabeceiraDoDeseno` |
| T8 letras embebidas | `TestFontesEmbebidas`, `TestPdffontsModerna` (sen Fira en TeX Live) |
| T9 mostras reais | captura |
| T10 kit | `TestFicheiroDeseno`, `TestAplicarDesenoATodas`, test do editor |
| T11 barras | `modelo.test.js` |
| T12 comprobar LaTeX | `TestComprobarElementoDeseno`, test do editor |
| T13 teclado | test de frechas |
| T14 task dev | arrinca |
| T15 protocolo | `analysis/deseno-proba-docentes.md` |

## Non verificado
- F1: a proba con docentes (protocolo escrito, sen executar).
- A interface na xanela real de Wails só se viu arrincar; as interaccións probáronse con jsdom e cunha páxina de proba no navegador.
- Yang non se probou coas letras novas de `edicion/latex`.
