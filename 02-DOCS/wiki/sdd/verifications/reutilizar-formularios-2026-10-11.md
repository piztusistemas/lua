---
type: verification
title: Verificación — reutilizar-formularios
slug: reutilizar-formularios
timestamp: 2026-10-11T00:00:00Z
topic: sdd
source_state: afe177f (dirty: os cambios desta funcionalidade, antes do seu commit)
verdict: pass (con límites anotados)
---

# Verificación — reutilizar-formularios (2026-10-11)

## Ordes
| Orde | Resultado |
|---|---|
| `gofmt -l .` | baleiro |
| `go vet ./...` | limpo |
| `go test ./... -count=1` (lua/lua) | ok (117 s) |
| `npx vitest run` | 34 ficheiros, 501 tests verdes (unha primeira execución deu 7 ficheiros en vermello por coincidir coa rexeneración dos bindings; repetida soa, verde) |
| bindings + `vite build` | compila |

## Criterios
| Criterio | Proba |
|---|---|
| Crear proxecto só co nome, baleiro | `proxectos.test.js` («crear un proxecto só pide o nome») |
| Sen lista de formularios nin «Ferramentas…» | `proxectos.test.js` |
| Proxectos pregados, desprégase ao premer | `proxectos.test.js` |
| + abre o catálogo (cronograma, actas, formularios por cartafoles) | `catalogo.test.js` |
| Marcar dous e Reutilizar | `catalogo.test.js`, `TestCatalogoCartafolesEReutilizar` |
| «Xa no proxecto» sen poder marcar | `catalogo.test.js` |
| Gardado nun proxecto → no catálogo | `TestFormularioGardadoNoProxectoChegaAoCatalogo` |
| Eliminar cartafol non borra formularios | `TestCatalogoCartafolesEReutilizar`, `catalogo.test.js` |
| Asistente move a un cartafol novo | `TestAsistenteMoveFormularioACartafolNovo` |
| Quitar do proxecto (columna) | Go: `quitarDoProxecto` (xa probado en `TestEngadirQuitarEEliminarProxecto`); o menú: só por código |
| Só o resultado, debaixo / ao final | `TestSolucionsSoResultado` |
| Opcións vellas iguais | `TestExpandirProblemas`, `TestFormularioProblemasPDF` |
| Catro idiomas | `i18n_test.go` |

## Límites
- O menú «Quitar do proxecto…» da columna e a recarga despois de reutilizar non teñen test de interface; comprobáronse no código, non abrindo a app.
