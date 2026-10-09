---
type: constitution
title: lua_yang — Constitución
description: Regras non negociables que obedecen todas as fases SDD.
tags: [sdd, constitution]
timestamp: 2026-10-10T00:00:00Z
topic: sdd
version: v1.0.0
ratified: 2026-10-10
last_amended: 2026-10-10
---

# lua_yang — Constitución

> **v1.0.0 — ratificada o 2026-10-10.** Emendas: só engadir, coa versión e o porqué no rexistro.
> Cada regra é comprobable: `analyze`, `verify` e `review` cítana como «constitución vX.Y.Z, regra N».

## Feitos de partida

- Obxectivo: **o programa Lua** (`lua/lua`). Lua absorbe Yang (Maxima, problemas, sementes); non hai app LuaYang separada.
  Lua depende de `edicion/` (motor común) e de `buxo/` (módulo de lectura de documentos); os tres forman o produto.
  Yang mantense só como fonte da que se migran capacidades.
- Tipo: software, en crecemento. Os cambios van directos á rama principal.
- Módulos: `edicion/` (motor común), `lua/lua`, `yang/yang`, `buxo/`, instaladores; `markitdown/` é código de terceiros.

## 1. Stack

1. O backend é Go (versión de `go.mod`, hoxe 1.25) e as apps de escritorio usan Wails v3.
2. O frontend é JavaScript sen framework, con Vite. Engadir unha dependencia de execución (Go ou npm) require unha entrada en `decisions.md`.
3. A lóxica que usa máis dunha app vive en `edicion/` e consómese con `replace`; nunca se copia entre apps.
4. Maxima, LaTeX e o sidecar Python son opcionais: se faltan, a app segue funcionando e explica en que falla e como instalalo.
5. `markitdown/` non se edita á man; os cambios van no build do sidecar (`buxo/sidecar/build.sh`).
6. Plataformas: macOS, Windows e Linux. Sen móbil.

## 2. Calidade

1. `gofmt -l` e `go vet ./...` saen limpos en cada módulo tocado.
2. `go test ./...` pasa en cada módulo tocado. Os fallos coñecidos están na táboa de excepcións e a lista non pode medrar.
3. Cada corrección de erro trae un test que fallaba antes. Cada funcionalidade nova trae polo menos un test do seu comportamento (TDD dentro de SDD).
4. A lóxica de frontend con tests pasa `npm test` (vitest).
5. Nada que dependa da IA, de Maxima ou de LaTeX necesita rede nin ferramentas instaladas para os tests unitarios: vai detrás dunha interface con dobre de proba. Os tests de integración que si as precisan sáltanse (`t.Skip`) se faltan.

| Excepción coñecida | Desde | Saída prevista |
|---|---|---|
| `lua/lua`: os tests do paquete principal non compilan (`pathHelperDirsMacOS` non existe) | 2026-10-10 | primeira tarefa: sen isto non corre ningún dos ~35 tests de Lua |
| `lua/lua`: `gofmt` sinala `biblioteca_formularios*.go` | 2026-10-10 | formatar na mesma tarefa |
| `edicion/modulo`: `TestDirsModulos` falla porque le a instalación real de Yang | 2026-10-10 | illar o test do directorio do usuario |
| Tests ODT / xelatex de Lua (rotos segundo a sesión do 2026-10-09; sen verificar ata que compilen) | 2026-10-09 | revisar cando compilen os tests de Lua |

## 3. Convencións

1. Identificadores, comentarios e documentación en galego normativo, como o código existente.
2. Ningún texto de interface vai cableado: ten clave en `idiomas/gl.json` (fonte da verdade). Unha funcionalidade non está feita ata que `es`, `en` e `pt` teñen as mesmas claves.
3. As fórmulas escríbense en LaTeX (`$…$`); a interface debúxaas con KaTeX e as exportacións saen da mesma fonte.
4. Dentro de `$…$`, unha variable `{x}` pegada a un comando, `^`, `_` ou `}` é LaTeX; o valor substitúese só con espazo (`$ {x} $`).
5. Commits en galego, en imperativo, cun gitmoji (o garda `.rsc/gitmoji-guard.mjs` compróbao).

## 4. Ramas e entregas

1. O traballo vai directo á rama `main` de cada repositorio (decidido na instalación). Hai dous: o da raíz (harness, `edicion/`, `buxo/`) e `lua/` (código de Lua). `yang/` e `markitdown/` teñen o seu propio repositorio e non se mesturan.
2. A versión é CalVer `AA.MM.N`. En Lua escríbese igual en `VERSION`, `wails.json`, `modulo.json`, `build/config.yml` e `build/darwin/Info*.plist`; unha release cambia todos á vez.
3. A autoría dos commits e PR é só de Martín Pardiñas Fondo: sen `Co-Authored-By` dunha IA nin pé «generated with». Compróbase na fase `ship`.

## 5. Seguridade e privacidade

1. Ningunha clave de API vai ao repositorio, aos rexistros nin aos ficheiros exportados; gárdase na configuración do usuario, fóra do proxecto.
2. Os datos do docente e do alumnado quedan no equipo. Só saen cara a un provedor de IA cando o usuario lanza unha acción de IA, e as importacións masivas amosan antes o custo estimado.
3. Maxima, LaTeX e calquera binario externo lánzanse con `exec` e argumentos separados, nunca a través dun shell. LaTeX compílase sen `--shell-escape`.
4. Cada chamada a un proceso externo ou á IA ten tempo máximo.

## 6. Interface e accesibilidade

1. Toda acción principal pódese facer co teclado.
2. Os erros amósanse no idioma do usuario e din que facer, non só que fallou.

## 7. Rendemento

1. As operacións longas (IA, Maxima, compilación LaTeX) nunca bloquean a interface: amosan progreso e pódense cancelar.

## 8. Coñecemento e decisións

1. Cada decisión significativa engádese a `02-DOCS/wiki/sdd/decisions.md` con data, opcións e porqué. Esta constitución é o rexistro de máis rango.
2. Un cambio visible para o usuario actualiza a axuda (`help/`) se a afecta.

## Definición de feito (o que comproba `verify`)

- [ ] `gofmt -l` baleiro e `go vet ./...` limpo nos módulos tocados.
- [ ] `go test ./...` verde nos módulos tocados (salvo excepcións da táboa, sen novas).
- [ ] `npm test` verde se se tocou frontend con tests.
- [ ] Test novo para cada erro corrixido e cada comportamento novo.
- [ ] Claves novas presentes en `gl`, `es`, `en` e `pt`.
- [ ] Sen claves nin datos persoais no diff nin nos rexistros.
- [ ] Decisións novas en `decisions.md`; axuda actualizada se cambia o que ve o usuario.

## Rexistro de emendas (só engadir)

| Data | Versión | Cambio | Porqué |
|------|---------|--------|--------|
| 2026-10-09 | draft | Esqueleto xerado por `rsc onboard`. | O SDD necesita unha constitución. |
| 2026-10-10 | draft | Borrador completo coas regras propostas. | Fase `constitution`. |
| 2026-10-10 | v1.0.0 | Ratificada: obxectivo = Lua (absorbe Yang, usa `edicion` e `buxo`); autoría só humana; dous repositorios git; excepcións de tests medidas. | Ratificación explícita do usuario. |
