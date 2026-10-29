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

## Ampliación (2026-10-29, segunda ronda)
Tras probalo, o usuario pide:
- Que a vista do documento xerado e o botón Xerar sigan á vista: o panel cobre só o panel de edición, non toda a xanela.
- Un botón de asistente na pregunta desanclada para crear ou mellorar enunciados e fórmulas. Escolleu un **asistente propio da pregunta**, que funciona en deseño e en produción. Ten accións rápidas (mellorar a redacción, números → variables, revisar fórmulas, crear unha parecida) e unha caixa libre. A IA só recibe esa pregunta e devolve unha proposta que se acepta ou se descarta.
- Decisión: «cancelar» descarta a resposta pendente. As chamadas á IA de Lúa (`ia.Turno`) teñen tempo máximo pero non contexto cancelable, coma no asistente que xa hai.
- Arranxo relacionado: as instrucións do Asistente datos (`formulario_datos_ia.go`) dicían «Lua pon 1., 2.» e xa non é certo.

## Checklist
- [x] Test: o botón desancla, a pregunta e a solución pasan ao panel, e Esc ou «Anclar» volven á reixa. Proba: `npm test` (problemas.test.js).
- [x] Claves novas nos catro idiomas. Proba: `go test -tags gtk3 -run I18n .` en `lua/lua`.
- [x] Aspecto: captura do panel nun Chrome automatizado.
- [x] Suite e compilación. Proba: `npm test`, `go test -tags gtk3 ./...` e `wails3 task build EXTRA_TAGS=gtk3`.

- [x] O panel encaixa no panel de edición (`.pane-editor`) e segue o seu tamaño. Proba: test «cobre só o panel de edición» en vermello e despois en verde.
- [x] Instrucións do Asistente datos sen «Lua pon 1., 2.». Proba: `go vet` limpo e a liña nova en `formulario_datos_ia.go`.
- [x] Go: binding `AsistentePregunta`. Le o JSON da IA, valida os nomes das variables, comproba que o enunciado se calcula e, se falla, devólvelle o erro á IA (ata 2 rondas). Proba: tests con dobre de IA.
- [x] Frontend: botón «Asistente» na pregunta desanclada, con accións, caixa libre, proposta con vista e os botóns Aceptar, Descartar e Cancelar. «Crear unha parecida» engade a pregunta despois. Proba: test en problemas.test.js cun dobre da API.
- [x] Claves nos catro idiomas, suite completa, captura e compilación.

## Evidence
- Vermello antes: o test «desancla unha pregunta…» fallaba con `Cannot read properties of null (reading 'click')`, porque non había botón.
- Verde despois: `npm test` → 573 tests pasan. O test comproba que se abre o panel, que leva a pregunta e a solución, que o foco vai ao enunciado, que segue aberto tras repintar o formulario, e que Esc e «Anclar» o pechan.
- Idiomas: `TestClavesDoFrontendEstanEnGalego` e `TestIdiomasTenAsMesmasClavesEVariables` pasan.
- Chrome automatizado a 1280×760: o panel cobre a xanela. O editor de fórmulas (Ctrl+M) ábrese por riba. O primeiro Esc pecha só o editor e o segundo ancla a pregunta.
- `go test -tags gtk3 ./...` → `ok lua-wails 158s`. `wails3 task build EXTRA_TAGS=gtk3` → `build/bin/lua` (2026-10-29 12:56).

- Encaixe: o test «cobre só o panel de edición» fallaba (`expected '' to be '60px'`) e agora pasa (7/7 en problemas.test.js).
- Asistente datos: `formulario_datos_ia.go` di agora «Non numeres as preguntas salvo que cho pidan: Lua non lles pon número no documento». `gofmt -l` baleiro e `go vet` limpo.

- Go: os cinco `TestAsistentePregunta*` (`problemas_asistente_test.go`) pasan cun dobre da IA. Comproban que a proposta se calcula, que os fallos volven á IA (2 rondas), que se queda cos fallos se non se arranxan, que se rexeitan as accións e os nomes que non valen, e que a variable leva paso 1 se falta. Os tests escribíronse antes ca o código, pero o «vermello» foi un erro de compilación, non un fallo observado.
- Frontend: os dous tests do asistente fallaban antes («cannot read … click») e pasan despois. Comproban que o botón sae só desanclada, que se pide coa acción e a pregunta, que a proposta non cambia o plan ata Aceptar, que «parecida» engade despois e que Descartar e Cancelar non cambian nada.
- Chrome automatizado, 1600×860, cunha IA de proba: a barra (Asistente, Xerar) e o documento seguen á vista. Ao redimensionar a 1200, o panel mide o mesmo ca o editor (624 px, top 52). Aceptar substitúe o enunciado.
- Suites: `npm test` → 576 pasan. `gofmt -l` baleiro, `go vet` limpo, `go test -tags gtk3 ./...` → `ok lua-wails 167s`. Compilación: `build/bin/lua` (2026-10-29 13:20).

## Next
- Que o usuario o probe cunha IA de verdade. Pendente de commit.
