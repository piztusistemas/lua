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

## 2026-10-10 — Alcance das traducións pendentes
- Opcións: traducir os 341 erros do Go · só os frecuentes · ningún; Buxo co idioma de Lua · con selector propio.
- Decisión: dúas specs (`buxo-idiomas`, `mensaxes-go-idiomas`). Só as mensaxes frecuentes do Go; Buxo segue o idioma de Lua; as respostas para a IA e os rexistros quedan en galego.
- Porqué: o custo de traducir erros raros que case ninguén ve non compensa; os frecuentes din como arranxar un problema.

## 2026-10-10 — Deseño das traducións do Go e de Buxo
- Decisión: un tradutor común novo, `edicion/idioma`, que usan Lua e Buxo. Os erros tradúcense ao lelos (`Error()`), non no frontend.
- Opcións descartadas: códigos de erro traducidos en JS (habería que tocar ~40 bindings); un tradutor propio en Buxo (duplicaría código, contra a regra 1.3).
- Lua pásalle o idioma a Buxo coa variable de entorno `BUXO_IDIOMA`, non cun argumento: un Buxo vello péchase cun argumento descoñecido.
- O rexistro de Buxo garda clave + argumentos: así o panel sae traducido e `buxo.log` en galego.

## 2026-10-10 — Traducir en cada situación, non ao final  (feature: mensaxes-go-idiomas, task: T006)
Context   — O test de idiomas exixe que cada clave estea nos catro JSON; engadir só o galego en T006-T012 e traducir en T013 deixaría os tests en vermello entre medias (constitución 2.2).
Options    — (a) traducir cada situación na súa tarefa; (b) relaxar o test ata T013.
Decision   — (a). T013 queda como comprobación final (74 claves en cada idioma).
Why        — Non se debilita un test para avanzar, e cada commit queda verde.

## 2026-10-10 — Nome visible «Lúa», nome técnico «lua»
- Decisión do usuario: na interface o aplicativo chámase «Lúa» (nome galego) e no apartado Sobre «Piztu Lúa». Cartafoles, rutas, ligazóns e identificadores seguen sendo «lua».
- Cambia a «Lúa»: textos dos catro idiomas (Lua, Buxo e instalador), título da xanela, CFBundleName de macOS, ProductName de Windows, textos de reserva no código e descricións para a IA.
- Queda «Lua»: `/Applications/Lua.app`, `Piztu/Lua`, `<config>/Lua/informes`, os accesos directos (`Lua.lnk`, `Name=Lua` do .desktop), `Options.Name` de Wails (identificador en Linux), a marca LaTeX `% --- engadido por Lua …`, o User-Agent e o repositorio.
