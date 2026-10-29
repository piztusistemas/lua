---
author: Martín Pardiñas
branch: main
status: in-progress
---
# Desanclar unha pregunta para editala a pantalla completa

## Intent
Editar os textos dunha pregunta na tarxeta estreita da reixa é incómodo. O usuario quere desanclar unha pregunta e editala nun espazo grande. Escolleu un panel a pantalla completa dentro da xanela de Lúa, non unha xanela do sistema.

## Scope
- Dentro: un botón «Desanclar» na cabeceira de cada pregunta. Abre un panel fixo que cobre a xanela: a pregunta á esquerda (o enunciado ocupa o alto dispoñible e a vista vai ao lado) e a súa solución á dereita, cos mesmos controis de sempre. «Anclar» ou Esc devólvea á reixa. Se o formulario se repinta, a pregunta segue desanclada (`estado.solta`).
- Fóra: xanela propia do sistema e varias preguntas desancladas á vez.

## Checklist
- [x] Test: o botón desancla, a pregunta e a solución pasan ao panel, e Esc ou «Anclar» volven á reixa. Proba: `npm test` (problemas.test.js).
- [x] Claves novas nos catro idiomas. Proba: `go test -tags gtk3 -run I18n .` en `lua/lua`.
- [x] Aspecto: captura do panel nun Chrome automatizado.
- [x] Suite e compilación. Proba: `npm test`, `go test -tags gtk3 ./...` e `wails3 task build EXTRA_TAGS=gtk3`.

## Evidence
- Vermello antes: o test «desancla unha pregunta…» fallaba con `Cannot read properties of null (reading 'click')`, porque non había botón.
- Verde despois: `npm test` → 573 tests pasan. O test comproba que se abre o panel, que leva a pregunta e a solución, que o foco vai ao enunciado, que segue aberto tras repintar o formulario, e que Esc e «Anclar» o pechan.
- Idiomas: `TestClavesDoFrontendEstanEnGalego` e `TestIdiomasTenAsMesmasClavesEVariables` pasan.
- Chrome automatizado a 1280×760: o panel cobre a xanela. O editor de fórmulas (Ctrl+M) ábrese por riba. O primeiro Esc pecha só o editor e o segundo ancla a pregunta.
- `go test -tags gtk3 ./...` → `ok lua-wails 158s`. `wails3 task build EXTRA_TAGS=gtk3` → `build/bin/lua` (2026-10-29 12:56).

## Next
- Que o usuario o probe no executable. Pendente de commit.
