---
type: spec
title: Spec — Buxo no idioma de Lua
description: WHAT e WHY para que a interface de Buxo apareza no idioma activo de Lua (gl, es, en, pt).
tags: [sdd, spec, i18n, buxo]
timestamp: 2026-10-10T00:00:00Z
topic: sdd
slug: buxo-idiomas
status: draft
---

# Spec — Buxo no idioma de Lua

> Slug: `buxo-idiomas` · Status: draft · Creada: 2026-10-10
> Inherits: [constitution](../constitution.md) (v1.2.0, regra 3.2: ningún texto de interface cableado)
> Irmá: [mensaxes-go-idiomas](mensaxes-go-idiomas.md)

## Problem & why
Lua xa se pode usar en galego, castelán, inglés e portugués, pero Buxo, o
módulo que Lua abre para converter documentos nunha bóveda, aparece sempre en
galego. Quen escolleu outro idioma en Lua atópase cunha xanela noutra lingua
xusto no paso onde máis decisións ten que tomar: que ficheiros subir, cando
construír a wiki e que fallou.

## Cost of not building it
Pouco en número: hoxe a maioría das persoas usan Lua en galego. Para quen non,
Buxo é a única pantalla enteira que non entende, e os erros de conversión
(«non se puido converter…») quedan sen ler. Ademais, Buxo incumpre a regra 3.2
da constitución mentres non teña traducións.

## The cheapest alternative
Deixar Buxo en galego e explicar na axuda de Lua o que significa cada botón.
Non abonda: Buxo ten uns 25 textos visibles, e traducilos custa pouco máis ca
escribir esa explicación, que ademais habería que manter igual.

## Goals
- Buxo aparece no mesmo idioma que Lua tiña activo ao abrilo.
- Ningún texto visible da interface de Buxo queda sen tradución nos catro idiomas.

## Non-goals / out of scope
- Un selector de idioma propio en Buxo (decisión do usuario, 2026-10-10).
- O contido que a IA escribe na bóveda (wiki, resumos): vai na lingua da materia, non na da interface.
- O ficheiro de rexistro `buxo.log` no disco.
- As mensaxes do Go de Lua e de `edicion`: son a especificación irmá.

## Users & context
Docentes que usan Lua en castelán, inglés ou portugués e abren Buxo desde un
proxecto de Lua para converter PDF, imaxes ou documentos nunha bóveda de Obsidian.

## Behaviour
- Main path: a persoa ten Lua en inglés e abre Buxo; todos os botóns,
  etiquetas, axudas e avisos de Buxo saen en inglés.
- Edge cases:
  - Buxo ábrese sen Lua (a man, para probas): sae en galego.
  - Lua pide un idioma que Buxo non coñece: sae en galego.
  - A un texto concreto fáltalle a tradución: ese texto sae en galego e o resto no idioma pedido.
  - A persoa cambia o idioma de Lua con Buxo xa aberto: Buxo segue no idioma que tiña ata que se volve abrir.
- Error paths: as mensaxes de erro que Buxo mostra na súa xanela (conversión
  fallida, sen IA, sen conexión con Lua) saen no idioma de Lua.

## Acceptance criteria
- Given Lua en castelán, When a persoa abre Buxo desde un proxecto, Then o título, os botóns e as axudas de Buxo están en castelán.
- Given Lua en inglés, When unha conversión falla en Buxo, Then a mensaxe de erro visible está en inglés.
- Given Buxo aberto sen Lua, When aparece a xanela, Then está en galego.
- Given un texto visible novo na interface de Buxo sen tradución nos catro idiomas, When se pasan os tests de Buxo, Then fallan e din que texto falta.
- Given Lua en galego, When se abre Buxo, Then a interface é idéntica á de hoxe.

## Points to clarify
- **pregunta abierta** — As liñas do panel de rexistro que se ven dentro da xanela de Buxo («Convertido X», «Erro en Y») contan como interface (traducir) ou como rexistro interno (galego)? Recomendo traducilas: a persoa ve ese panel.
- **suposición tomada** — Buxo non cambia de idioma en quente. *Base:* Lua abre Buxo como programa á parte e a persoa adoita cambiar de idioma unha soa vez. *Riesgo:* se se pide cambio en quente, fai falla unha canle de aviso entre Lua e Buxo.
- **suposición tomada** — O galego segue sendo a fonte das traducións, coma en Lua. *Base:* constitución 3.2 e `lua/lua/idiomas/README.md`. *Riesgo:* ningún coñecido.
- **decisión diferida** — Traducir o diálogo de licenza (texto legal MIT e avisos de terceiros): queda en inglés orixinal.

## Revisions
- 2026-10-10 — Borrador. Alcance decidido polo usuario: Buxo segue o idioma de Lua, sen selector propio; o rexistro interno queda en galego.
