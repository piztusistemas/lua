---
author: Martín Pardiñas
branch: main
status: in-progress
---
# Tamaño das fórmulas nos documentos (Opcións)

## Intent
Unha fracción dentro dunha liña (`$\frac{a}{b}$`) sae moi pequena no documento, porque LaTeX a compón en tamaño «de texto». O usuario quere poder agrandar as fórmulas sen cambiar o tamaño do texto, e fixalo unha vez para todos os seus traballos.

## Scope
- Dentro: un apartado en Opcións, «Fórmulas nos documentos», con dous controis:
  - **Estilo:** compactas (como agora) ou a tamaño completo (`\displaystyle`, coma `\dfrac`).
  - **Tamaño respecto ao texto:** de 100 % a 150 %, en pasos de 10 %.
- Aplícase aos formularios (exames, solucións, solucionario) en PDF, TeX, Word e ODT, e á vista das fórmulas no editor.
- Fóra: o Markdown exportado (non ten tamaños). Cronograma e actas seguen coma agora, porque non reciben as Opcións.

## Decisións
- Escala con `\scalebox` sobre cada fórmula e non con `\DeclareMathSizes`. A primeira non depende das letras nin do motor (pdflatex ou xelatex). A segunda precisa letras con calquera tamaño e coñecer o tamaño do corpo.
- `latexTramosCrono` escribe `\luaformula{…}` e `\luaformulacentrada{…}`. As macros defínense ao principio do corpo, xunto a `babelListquotFix`. Así chegan tamén cunha plantilla con `\documentclass` propio. Por defecto fan o mesmo ca antes (`$…$`, `\[…\]`). Sen `graphicx`, a escala ignórase.
- Word e ODT: a imaxe de cada fórmula compílase con `\displaystyle` (se é a tamaño completo) e o seu tamaño multiplícase pola porcentaxe.

## Checklist
- [x] Go: `opcionsFormulas` (normalizar e definicións LaTeX) e `latexTramosCrono` con `\luaformula`. Proba: tests unitarios en vermello e despois en verde.
- [x] Go: as Opcións (`Settings`) chegan ao formulario (PDF/TeX e Word/ODT). Proba: test que o `.tex` leva as definicións e que as pezas de Word levan `\displaystyle`.
- [x] Compilación real con pdflatex e xelatex, a 100 % e a 150 %. Proba: compila, e unha imaxe da páxina amosa a fórmula grande co texto igual e as liñas sen montarse.
- [x] Frontend: o apartado en Opcións garda e le os valores, e a vista do editor (KaTeX) aplica estilo e escala. Proba: test vitest de mates-vista.
- [x] Claves nos catro idiomas, suites, compilación e axuda (README).

## Evidence
- Go: os cinco tests de `formulas_estilo_test.go` fallaban por non compilar (`undefined: opcionsFormulas`) e despois pasan. Comproban a normalización, as definicións LaTeX (por defecto idénticas ás de antes), `\luaformula` nos tramos, que o formulario leva as opcións no corpo e as pezas de Word con `\displaystyle`.
- `TestParseMathCrono` esperaba `\[a + b` literal. Cambiou a `\luaformulacentrada{a + b`, porque o cambio é intencionado.
- Compilación real: un exame con fraccións en liña e un sumatorio centrado, con pdflatex e con xelatex, a 100 % compactas, 130 % compactas e 150 % grandes. Os seis compilan (código 0). Nas imaxes da páxina o texto mide o mesmo, as fórmulas medran, as liñas sepáranse soas sen montarse e as fórmulas comparten liña base co texto (recorte a 160 ppp).
- O resto da suite de Go compila de verdade cronograma, actas e formularios con plantillas e pasa. As definicións por defecto chegan a todos os camiños.
- Frontend: os cinco tests novos de `mates-vista.test.js` fallaban e agora pasan. `npm test` → 581.
- Idiomas: `TestClavesDoFrontendEstanEnGalego` e `TestIdiomasTenAsMesmasClavesEVariables` pasan.
- `gofmt` baleiro, `go vet` limpo, `go test -tags gtk3 ./...` → `ok 166s`. Compilación: `build/bin/lua` (2026-10-29 14:05).
- Sen verificar: o aspecto do diálogo de Opcións na app real (non hai captura).

## Next
- Que o usuario o probe. Pendente de commit.
