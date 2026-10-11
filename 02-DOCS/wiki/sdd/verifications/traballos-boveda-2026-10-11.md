---
type: verification
title: Verificación — traballos-boveda
slug: traballos-boveda
timestamp: 2026-10-11T00:00:00Z
topic: sdd
source_state: 3c6cb7a (dirty: os cambios desta funcionalidade, antes do seu commit)
verdict: pass (con límites anotados)
---

# Verificación — traballos-boveda (2026-10-11)

## Ordes
| Orde | Resultado |
|---|---|
| `gofmt -l .` / `go vet ./...` | limpos |
| `go test . -count=1` | ok (118 s). A primeira execución falou en `TestTraballoProduccionBloqueado`: o test buscaba o texto galego e outro test deixara o idioma en castelán; corrixido o test (compara coa mensaxe no idioma activo), sen cambiar código. |
| `npx vitest run` | 34 ficheiros, 504 tests verdes |

## Criterios
| Criterio | Proba |
|---|---|
| Sen bóveda → créase a do proxecto, con nota e PDF | `TestTraballoCreaABovedaDoProxectoESubstitue` |
| Engadir dúas veces → unha nota e un PDF | o mesmo test |
| Bóveda conectada se o proxecto non ten | `TestTraballoVaiABovedaConectada` |
| Nota co contido e ligazón ao PDF | `TestTraballoCreaABovedaDoProxectoESubstitue` |
| Marca «sempre» engade ao xerar; desmarcada non | `TestBovedaAutomaticaPorFormulario`, `controlador.test.js` («despois dun Xerar bo avisa»); a chamada automática en `main.js` só por código |
| Produción bloqueada (botón, marca) | `TestTraballoProduccionBloqueado` |
| Cronograma e actas | `TestTraballoCronograma` |
| Catro idiomas | `i18n_test.go` |

## Límites
- O botón, a marca e o seu estado en produción non teñen test de interface nin se viron na app aberta.
- O asistente non ten ferramenta para engadir á bóveda, así que o bloqueo «polo asistente» cúmprese porque non hai ese camiño.
