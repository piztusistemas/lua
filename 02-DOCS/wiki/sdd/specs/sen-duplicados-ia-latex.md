---
type: spec
title: Spec — Un só cliente de IA e unhas soas funcións LaTeX
description: WHAT e WHY para que o cliente de IA e as funcións auxiliares de LaTeX existan unha soa vez, en edicion, sen cambiar o comportamento de Lúa.
tags: [sdd, spec, refactor, ia, latex]
timestamp: 2026-10-10T00:00:00Z
topic: sdd
slug: sen-duplicados-ia-latex
status: approved (autopiloto, 2026-10-10)
---

# Spec — Un só cliente de IA e unhas soas funcións LaTeX

> Slug: `sen-duplicados-ia-latex` · Status: approved in autopilot · Creada: 2026-10-10
> Inherits: [constitution](../constitution.md) v1.2.1 (regra 1.3: o común vive en `edicion` e non se copia)

## Problem & why
O código que fala cos provedores de IA (Gemini, Claude e compatibles con OpenAI)
está escrito cinco veces: tres chamadas simples e un cliente de conversa en
Lúa, e outro cliente de conversa en `edicion`. As funcións auxiliares de LaTeX
(tipografía, motores, pistas de erro) están copiadas enteiras entre Lúa e
`edicion`. Cada arranxo hai que facelo varias veces: traducir 8 erros da IA
obrigou a cambiar 41 sitios, e o erro das fontes TeX Gyre arranxouse dúas veces.
Isto incumpre a regra 1.3 da constitución.

## Cost of not building it
Cada provedor novo, cambio de API ou arranxo da IA custa cinco veces máis e
deixa a porta aberta a que unha copia quede sen arranxar (xa pasou coas
mensaxes e coas fontes). Para quen usa Lúa non hai custo inmediato: o
comportamento hoxe é correcto.

## The cheapest alternative
Deixar as copias e engadir un test que compare que seguen sendo iguais. Detecta
a diverxencia, pero non evita facer cada cambio varias veces; só avisa de que se
esqueceu algún.

## Goals
- Hai un só cliente de IA para os tres provedores, en `edicion`, e Lúa úsao para todo: conversa con ferramentas e chamadas simples.
- Hai unha soa copia das funcións auxiliares de LaTeX, en `edicion`, e Lúa úsaa.
- Lúa compórtase exactamente igual ca antes.

## Non-goals / out of scope
- Yang: non se modifica o seu código (decisión do usuario). Só se comproba que non aparecen fallos novos respecto da súa liña base.
- Engadir provedores novos ou cambiar o que se lles envía.
- Outras duplicacións ou partir `main.js` (outro ciclo).
- Buxo: xa usa a IA a través da API de Lúa.

## Users & context
Quen mantén Lúa: cada cambio na IA ou en LaTeX ten que facerse nun só sitio.
Quen usa Lúa non debe notar ningunha diferenza.

## Behaviour
- Main path: para cada provedor, a petición que sae de Lúa (enderezo, cabeceiras, corpo, modelo, límite de tokens, imaxes e ferramentas) e a forma de interpretar a resposta son idénticas ás de antes; o mesmo para o texto LaTeX que xera Lúa (preámbulo de tipografía, motores, pistas de erro).
- Edge cases: sen clave ou sen modelo, os erros son os mesmos (e no idioma activo, coma agora); a resposta cortada polo límite de tokens dá o mesmo aviso; os bloques ```…``` das respostas límpanse coma antes onde se limpaban.
- Error paths: erros HTTP e respostas non válidas producen as mesmas mensaxes ca hoxe.

## Acceptance criteria
- Given cada provedor (Gemini, Anthropic, OpenAI compatible) e un servidor de proba, When Lúa fai unha chamada simple e unha volta de conversa con ferramentas e imaxes, Then a petición recibida (ruta, cabeceiras relevantes e corpo JSON) é idéntica á que se gravou antes da refactorización.
- Given as mesmas respostas de proba, When Lúa as interpreta, Then o resultado (texto, chamadas a ferramentas, erros) é idéntico ao de antes.
- Given as opcións de tipografía e os logs de LaTeX de proba, When Lúa xera o preámbulo e as pistas de erro, Then o texto é idéntico ao de antes.
- Given o código final, When se buscan as funcións de envío aos provedores e as auxiliares de LaTeX en `lua/lua`, Then non hai ningunha: só existen en `edicion`.
- Given as suites, When se executan, Then `edicion`, `lua/lua`, `buxo` e o instalador pasan, e Yang dá a mesma liña base ca antes.

## Points to clarify
- **suposición tomada** — As copias de conversa son idénticas agás o nome dun campo de imaxes. *Base:* diff do 2026-10-10 (16 liñas, só nomes). *Riesgo:* se aparece unha diferenza de comportamento, decídese co usuario cal se mantén.
- **suposición tomada** — `lua/latexdoc.go` non ten funcións propias: todas existen en `edicion/documento/latex.go`. *Base:* comparación de nomes do 2026-10-10. *Riesgo:* se o corpo dalgunha difire, as probas de caracterización amosarano.
- **decisión diferida** — Yang segue coas súas copias; quitalas é outro ciclo, no seu repositorio.
- **área no formulable** — As chamadas simples de Lúa (`chamarIA`) teñen detalles propios (timeouts, limpeza de Markdown) que aínda non revisei liña a liña; o plan dirá se `edicion/ia` precisa unha opción nova para cubrilos.

## Revisions
- 2026-10-10 — Escrita e aprobada en autopiloto (o usuario escolleu «Autopiloto» e «Yang fóra, só non rompelo»).
