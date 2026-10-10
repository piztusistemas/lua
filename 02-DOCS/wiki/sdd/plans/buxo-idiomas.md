---
type: plan
title: Plan — Buxo no idioma de Lua
description: Plan de estrutura para que a interface, o panel de rexistro e os erros visibles de Buxo saian no idioma activo de Lua.
tags: [sdd, plan, i18n, buxo]
timestamp: 2026-10-10T00:00:00Z
topic: sdd
slug: buxo-idiomas
status: draft
---

# Plan — Buxo no idioma de Lua

> Spec: [../specs/buxo-idiomas.md](../specs/buxo-idiomas.md) · Constitution: [../constitution.md](../constitution.md) · Status: draft
> Last updated: 2026-10-10

## 0. Global Constraints

- **Idiomas:** exactamente `gl`, `es`, `en`, `pt`. O galego é a fonte; calquera outro código, ou ningún, equivale a `gl`.
- **Galego intacto:** con `gl`, todo texto visible é byte a byte o de hoxe (spec, aceptación 5).
- **`buxo.log` en galego sempre**, aínda que o panel saia noutro idioma.
- **Compatibilidade entre versións:** un Lua novo pode lanzar un Buxo vello e viceversa. O idioma viaxa nunha variable de entorno, `BUXO_IDIOMA`, nunca nun argumento: un Buxo vello péchase cun argumento descoñecido e ignora unha variable.
- **Respostas para a IA en galego:** as mensaxes das ferramentas `boveda_*` e os erros que só le a IA non se traducen.
- **Constitución:** 2.1-2.3 (gofmt, go vet e go test limpos; test antes do código), 3.1 (identificadores e comentarios en galego), 3.2 (ningún texto de interface cableado), 4.3 (autoría só humana, sen `Co-Authored-By`).

## 1. Context & constraints

- Aceptación da spec: 1 (castelán ao abrir), 2 (erro de conversión en inglés), 3 (sen Lua → galego), 4 (test que detecta textos sen traducir), 5 (galego idéntico), 6 (panel en portugués e `buxo.log` en galego).
- Hoxe Buxo ten tres fontes de texto visible: o HTML e o JS do frontend (uns 25 textos, incluído `util.js`), as liñas do rexistro que emite o Go (38 chamadas) e os erros que devolven ao frontend os métodos de Buxo (~20 dos 33 `Errorf`).
- Lua xa garda o idioma en `Settings.Idioma` e lanza Buxo en `buxo_modulo.go`.
- Fóra: selector propio, contido da wiki, licenza, mensaxes do Go de Lua (spec irmá).

## 2. Architecture

```text
Lua (Settings.Idioma) --BUXO_IDIOMA--> proceso Buxo
                                          |
                     +--------------------+--------------------+
                     v                    v                    v
              Tradutor (Go)        Rexistro (Go)          Frontend (JS)
          idiomas/*.json embebidos  liña = clave+args      t(clave, galego)
                     |              ├─> panel: idioma activo   |
                     |              └─> buxo.log: galego       |
                     +--------- Idioma() exposto -------------->
```

- **Lanzador en Lua** (interno, Lua) — engade `BUXO_IDIOMA=<idioma efectivo>` ao entorno ao lanzar Buxo: `Settings.Idioma` ou, se está baleiro, `IdiomaPiztu()`.
- **Dicionarios** (interno, Buxo) — `buxo/idiomas/{gl,es,en,pt}.json`, unha clave por texto, coma `lua/lua/idiomas`. Úsanos o Go (embebidos) e o frontend (importados).
- **Tradutor** (interno, `edicion/idioma`, compartido co plan irmán) — Buxo rexistra os seus dicionarios e fixa o idioma unha vez ao arrincar desde `BUXO_IDIOMA`; traduce unha clave cun formato de reserva en galego.
- **Rexistro** (interno, Go) — cada liña leva a clave, o formato galego e os argumentos. Escribe en `buxo.log` co galego e emite ao panel co idioma activo.
- **Erros visibles** (interno, Go) — os erros que chegan á persoa levan clave e argumentos, e o rexistro e as respostas ao frontend tradúcenos. O texto `Error()` segue en galego para os logs e para a IA.
- **Frontend** (interno, JS) — pide o código de idioma ao Go e aplica `t()` ao HTML (`data-i18n*`) e aos textos de `main.js` e `util.js`, coma Lua.

**Decisión principal:** o rexistro garda **clave + argumentos** e non texto xa formatado. Só así unha mesma liña sae en galego no ficheiro e noutro idioma no panel, como pide a aceptación 6. A alternativa, traducir o texto xa feito, non pode reconstruír os argumentos. Descartada.

## 3. Interfaces & contracts

```text
Ao arrincar: idioma.Rexistrar(gl|es|en|pt, buxo/idiomas/*.json) e idioma.SetIdioma(BUXO_IDIOMA)
  - baleiro ou descoñecido → "gl"

idioma.T(clave, formatoGalego, args...) / idioma.Errorf(...)   (contrato no plan irmán, §3)
  - os argumentos insírense igual en todos os idiomas (mesmos verbos e na mesma orde)

Rexistro.Info|Aviso|Erro(clave, formatoGalego, args...)
  - buxo.log  ← Sprintf(formatoGalego, args)       (sempre galego)
  - panel     ← idioma.T(clave, formatoGalego, args)     (idioma activo)
  - se un argumento é un erro traducible, no panel vai traducido e no ficheiro en galego

Erros visibles: idioma.Errorf(clave, formatoGalego, args)
  - Error() → idioma activo; o rexistro escribe o ficheiro co formato galego da mesma clave
  - os erros que só le a IA seguen con fmt.Errorf en galego

App.Idioma() -> código   (binding para o frontend)

Frontend: t(clave, galego, vars?) e aplicarTraducions() sobre data-i18n, data-i18n-title,
          data-i18n-placeholder, coma lua/lua/frontend/src/main.js
```

## 4. Data model & flow

- **Dicionarios:** catro JSON planos `clave → texto`. Uns 80 textos: ~25 do frontend, 38 formatos do rexistro, ~6 motivos de lectura con IA, ~15 erros visibles e o título do diálogo de ficheiros.
- **Prefixos:** `ui.*` frontend, `rex.*` rexistro, `erro.*` erros visibles, `dialogo.*` diálogos.
- **Fluxo:** Lua → `BUXO_IDIOMA` → Buxo arrinca → idioma fixo durante a sesión (non hai cambio en quente, segundo a spec) → o frontend pide `Idioma()` e carga o dicionario.
- **Historial:** ao reabrir a xanela, o panel repíntase desde o historial en memoria. O historial garda clave + argumentos, así que se repinta no idioma activo.

## 5. Testing strategy

| Aceptación | Nivel | Que comproba | Que se finxe |
|---|---|---|---|
| 1, 3 | unidade Go | o arrinque con `BUXO_IDIOMA` = es / baleiro / `xx` deixa `idioma.Actual()` = es / gl / gl | a variable de entorno |
| 1 | unidade JS (vitest) | `aplicarTraducions` co dicionario `es` pon os textos en castelán no DOM | o binding `Idioma()` |
| 2 | unidade Go | un erro de conversión de `idioma.Errorf`, co idioma `en`, dá o texto inglés e conserva o argumento | — |
| 4 | unidade Go | toda clave usada no Go e no frontend existe en `gl.json`; os catro JSON teñen as mesmas claves e os mesmos verbos e variables; o formato galego do código coincide co de `gl.json` | — (le os ficheiros) |
| 5 | unidade Go | con `gl`, cada formato traducido é idéntico ao formato galego de hoxe | — |
| 6 | unidade Go | `Rexistro` con idioma `pt`: a liña emitida está en portugués e a do ficheiro en galego | o emisor do panel e un ficheiro temporal |
| Lua lanza Buxo | unidade Go en Lua | o comando de Buxo leva `BUXO_IDIOMA` igual ao idioma efectivo (`Settings.Idioma` ou `IdiomaPiztu()`) | sen lanzar o proceso |

O test da aceptación 4 é o garda permanente: falla se alguén engade un texto visible sen clave nos catro idiomas. Primeiro hai que velo fallar.

## 6. Sequencing & dependencies

1. `edicion/idioma` (paso 1 do plan irmán, faise unha vez) e os dicionarios de Buxo, rexistrados ao arrincar desde `BUXO_IDIOMA`, cos seus tests.
2. `Rexistro` con clave + argumentos e as 38 chamadas migradas. Depende de 1.
3. Os ~15 erros visibles pasan a `idioma.Errorf`; o rexistro escribe o ficheiro en galego. Depende de 1 e 2.
4. Frontend: `t()`, `data-i18n*` no HTML, `main.js` e `util.js`, e o binding `Idioma()`. Depende de 1. Pódese facer en paralelo con 2-3.
5. Test garda (aceptación 4) e traducións es/en/pt de todas as claves. Depende de 2, 3 e 4.
6. Lua: `BUXO_IDIOMA` no lanzador, co seu test. Independente de 1-5.
7. Verificación: suites de Buxo e Lua; Buxo aberto a man con `BUXO_IDIOMA=en` e captura do panel, ou proba do usuario se non hai captura.

## 7. Risks & open decisions

1. **Formatos con argumentos en galego** (o «motivo» da lectura con IA, os erros que chegan de Lua). *Disparador:* unha liña traducida cun anaco galego dentro. *Impacto:* medio. *Mitigación:* os motivos de Buxo pasan a ser claves; os que veñen de Lua quedan en galego, como acorda a spec.
2. **Desfase entre o formato do código e `gl.json`.** *Disparador:* alguén cambia un no código e non no JSON. *Mitigación:* o test da aceptación 4 compara os dous.
3. **Verbos de formato distintos entre idiomas** (`%d` nun, `%s` noutro). *Impacto:* texto roto en tempo de execución. *Mitigación:* o test compara a secuencia de verbos.
4. **Buxo vello con Lua novo.** Retirado: a variable de entorno ignórase.
5. **Non poder ver a xanela** (este entorno non fai capturas). *Mitigación:* os tests cobren o comportamento; a proba visual faina o usuario.

**Decisións abertas:** ningunha.

**Illamento:** a constitución (4.1b) fixa o traballo directo en `main`; non se abre rama.
