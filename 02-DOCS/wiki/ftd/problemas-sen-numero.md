---
author: Martín Pardiñas
branch: main
status: in-progress
---
# Problemas sen número de pregunta por defecto

## Intent
No documento xerado, cada pregunta dun campo «Problemas» saía con «**1.**», «**2.**»… diante do enunciado. O usuario non quere ese número por defecto. Se o quere, escríbeo no propio enunciado coma texto.

## Scope
- Dentro: o enunciado do exame (`expandirProblemas`, `lua/lua/formulario_problemas.go`). Vale para PDF, TeX, Markdown, Word e ODT, que comparten os bloques.
- Os puntos seguen indo diante do enunciado: «*(2 pt)* …».
- Fóra: o solucionario e a lista de resultados ao final seguen numerados («1.», «2.»), porque sen iso non se sabe a que pregunta vai cada solución. Os avisos internos («pregunta 3: …») tamén seguen.

## Checklist
- [x] Test que falla antes: o enunciado non empeza por «**1.**». Proba: `go test -run TestExpandirProblemas` en vermello antes e en verde despois.
- [x] Quitar o número en `expandirProblemas`; sen puntos nin texto, non se escribe un bloque baleiro. Proba: o mesmo test.
- [x] Suite completa. Proba: `gofmt -l`, `go vet` e `go test -tags gtk3 ./...` en `lua/lua`.
- [x] Recompilar o executable. Proba: `task build EXTRA_TAGS=gtk3` remata e crea `build/bin/lua`.

## Evidence
- Vermello antes: `TestExpandirProblemas` fallaba con `enunciado: "**1.** *(2 pt)* Un móbil percorre $120$ m…"`.
- Verde despois: `go test -tags gtk3 -run TestExpandirProblemas .` → `ok`. O test comproba tamén que unha pregunta sen puntos empeza directamente polo enunciado.
- Suite: `gofmt -l .` baleiro, `go vet -tags gtk3 ./...` limpo, `go test -tags gtk3 ./...` → `ok lua-wails 157s`, `ok lua-wails/grafico`.
- Compilación: `wails3 task build EXTRA_TAGS=gtk3` → `build/bin/lua` (2026-10-29 12:49).
- O frontend non numera as preguntas en ningunha vista (busca sen resultados).

## Next
- Que o usuario probe un documento xerado. Pendente de commit.
