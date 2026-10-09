---
type: decisions
title: lua_yang — Rexistro de decisións
description: Decisións significativas do proxecto, con data, opcións e porqué. Só engadir.
tags: [sdd, decisions]
topic: sdd
---

# Rexistro de decisións

## 2026-10-10 — Constitución v1.0.0 ratificada
- Decisión: `02-DOCS/wiki/sdd/constitution.md` pasa a v1.0.0.
- Porqué: o SDD necesita regras comprobables antes da primeira funcionalidade grande.

## 2026-10-10 — O produto é Lua
- Opcións: app LuaYang separada · Lua absorbe Yang.
- Decisión: Lua absorbe Yang; o produto é Lua máis `edicion/` e `buxo/`.
- Porqué: o usuario quere un programa único, e o traballo do 2026-10-09 xa vai dentro de Lua.

## 2026-10-10 — Autoría só humana
- Opcións: só o usuario · coautoría da IA nos commits.
- Decisión: só Martín Pardiñas Fondo; sen `Co-Authored-By` da IA.
- Porqué: decisión explícita do usuario.

## 2026-10-10 — Repositorio git na raíz
- Opcións: un repositorio que englobe todo · un repositorio na raíz que deixa fóra os que xa existen · submódulos.
- Decisión: o repositorio da raíz rastrexa o harness, `edicion/` e `buxo/`; `lua/`, `yang/` e `markitdown/` conservan o seu propio `.git` e quedan ignorados.
- Porqué: non se perde nin se mestura a historia dos repositorios que xa existen, e `edicion/` e `buxo/` pasan a ter control de versións.
- Pendente: `lua/.git` apunta a `github.com/piztutao/yang-src` (o mesmo remoto que `yang/`). O usuario indica `github.com/piztusistemas/lua`, que hoxe é público e só ten ficheiros de exemplo e de release. Falta decidir que se sobe alí.

## 2026-10-10 — Versión de Lua 26.10.3
- Decisión: Lua pasa a 26.10.3 en todos os ficheiros de versión do código (non nos binarios xa construídos).

## 2026-10-10 — Un só repositorio en GitHub
- Opcións: repositorio só de Lua · un repositorio con Lua, `edicion` e `buxo`.
- Decisión: un só repositorio, `github.com/piztusistemas/lua`, que une a historia local cos ficheiros que xa había publicados alí.
- Porqué: `lua/lua/go.mod` usa `../../edicion`, e por separado Lua non compila.
- A historia antiga de `lua/.git`, que apuntaba a `yang-src`, gárdase fóra do proxecto en `../lua-git-respaldo`.

## 2026-10-10 — Instalador en Wails v3
- Decisión: o instalador pasa de Wails v2 a v3, coma Lua e Buxo.
- Porqué: unha soa CLI (`wails3` + `task`) para todo o produto.
- Coidado: `wails3 task common:update:build-assets` reescribe o manifesto de Windows e quita `requireAdministrator`.

## 2026-10-10 — markitdown dentro do repositorio
- Opcións: submódulo git de microsoft/markitdown · copia dentro do repositorio.
- Decisión: copia dentro do repositorio, desde o commit `d9cae89`.
- Porqué: o submódulo perdería o plugin propio `markitdown-math` e obrigaría a un paso extra ao clonar. A historia anterior está en `../markitdown-git-respaldo`.
