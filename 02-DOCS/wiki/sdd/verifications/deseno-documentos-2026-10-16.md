---
type: verification
title: Verificación — deseno-documentos
slug: deseno-documentos
timestamp: 2026-10-16T22:40:00Z
topic: sdd
verdict: PASS (con riscos anotados)
---

# Verificación — deseno-documentos (2026-10-16)

| Comprobación | Resultado |
|---|---|
| `gofmt -l` (edicion, lua/lua) | baleiro |
| `go vet ./...` (edicion, lua/lua) | limpo |
| `go test ./...` edicion | verde |
| `go test ./...` lua/lua | verde (≈140 s, con xelatex, pdffonts e pdftoppm) |
| `npm test` (vitest) | 542/542 |
| Claves novas en gl/es/en/pt | 137 claves; `TestIdiomasTenAsMesmasClavesEVariables` verde |
| Compilar a app (`task build`) | `build/bin/lua` (16 out. 22:35) |

## Criterios de aceptación

| AC | Evidencia |
|---|---|
| AC1 letra en todos os papeis | `TestPdffontsCombinacion`: só Termes/Heros no PDF, sen LMSans |
| AC2 tamaño real | `TestTamanoNormalsize`, `TestTamanoCompila` |
| AC3 viaxa coa plantilla | `TestPlantillaLevaDesenoEnJSON`, test do editor (gardar leva `deseno`) |
| AC4 branco e negro | `TestDesenoBrancoNegro` |
| AC5 corrector de contraste | `TestCorrectorContraste` |
| AC6 inspector | `TestMarcasPosicion`, `editor.test.js` (clic no título abre Títulos) |
| AC7 IA validada | `modelo.test.js` (validarParche), `ferramentas-plantilla.js` |
| AC8 cores do logotipo | `TestCoresDoLogo` |
| AC9 PDF en Palatino → Elegante | `TestImportarDesenoPdf` |
| AC10 idiomas | i18n verde |

## Revisión visual
PDF reais con seis deseños (Lúa, Clásica, Lexible+lectura fácil, Elegante B/N,
Moderna, cor clara+aforrar) e actas en Académica. Atopáronse e arranxáronse: caixas
de meses e etiquetas que non medraban co tamaño; interliñado amplo dentro das
etiquetas; redacción do aviso de contraste; e un erro anterior a este traballo
(portada «Cronogramaanual», sen espazo) máis a portada fixa en índigo.

## Non verificado
- A xanela real de Wails: `task dev` falla antes deste traballo (falta o script
  `build:dev` en `package.json`). A interface probouse con jsdom e cunha páxina
  de proba no navegador co editor real e un backend simulado.
- Yang (repositorio propio) usa `edicion/latex`: `TipografiaPreambulo` agora
  define as letras de todos os papeis; non se probou alí.
- Ningunha proba con docentes.
