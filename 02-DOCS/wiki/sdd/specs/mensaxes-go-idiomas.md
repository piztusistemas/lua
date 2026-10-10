---
type: spec
title: Spec — Mensaxes frecuentes de Lua no idioma activo
description: WHAT e WHY para que os erros e avisos máis comúns que xera Lua por dentro aparezan no idioma escollido (gl, es, en, pt).
tags: [sdd, spec, i18n, lua, edicion]
timestamp: 2026-10-10T00:00:00Z
topic: sdd
slug: mensaxes-go-idiomas
status: clarified
---

# Spec — Mensaxes frecuentes de Lua no idioma activo

> Slug: `mensaxes-go-idiomas` · Status: clarified · Creada: 2026-10-10
> Inherits: [constitution](../constitution.md) (v1.2.0, regra 3.2: ningún texto de interface cableado)
> Irmá: [buxo-idiomas](buxo-idiomas.md)

## Problem & why
A interface de Lua xa está traducida, pero as mensaxes que xera Lua por dentro
(erros e avisos ao compilar, exportar, instalar ou abrir ficheiros) saen sempre
en galego. Son xusto as que a persoa ten que entender para saber que facer: «non
se atopou xelatex», «falta o paquete de LaTeX…», «non se puido converter…».
Medido o 2026-10-10: 341 mensaxes deste tipo poden chegar á pantalla, pero a
maioría son casos raros.

## Cost of not building it
Pouco en número, alto no momento: quen usa Lua noutro idioma queda sen entender
a mensaxe que lle di como arranxar un problema, e pode abandonar ou pedir
axuda. Lua segue incumprindo a regra 3.2 da constitución nesas mensaxes.

## The cheapest alternative
Traducir as 341. Isto non é barato, pero é a alternativa obvia, e descartouse:
moitos erros raros (ZIP corrupto, permiso denegado nun cartafol interno) case
ninguén os verá, e cada mensaxe nova obrigaría a escribir catro textos. A máis
barata de todas é non facer nada; descartouse porque deixa sen entender
precisamente os avisos que din como arranxar algo.

## Goals
- As mensaxes das situacións frecuentes (ver Behaviour) aparecen no idioma activo de Lua.
- Queda escrito cales son esas mensaxes, e un test impide que unha delas quede sen tradución.

## Non-goals / out of scope
- Os erros raros que non están na lista: seguen en galego (decisión do usuario, 2026-10-10).
- As respostas das ferramentas do asistente para a IA (unhas 50): a persoa non as ve, e a IA responde no seu idioma.
- Os rexistros internos e os ficheiros de log.
- A saída de programas externos que Lua mostra tal cal (log de LaTeX, mensaxes de LibreOffice ou Maxima).
- Os textos dos documentos xerados: xa dependen da plantilla («Textos do documento»).
- Buxo: é a especificación irmá.

## Users & context
Docentes que usan Lua en castelán, inglés ou portugués e se atopan cun aviso ou
un erro ao xerar, exportar, instalar dependencias ou abrir ficheiros.

## Behaviour
- Main path: unha lista escollida de mensaxes, unhas 70, sae no idioma activo.
  Escóllense dentro destas sete situacións, só as que ve case todo o mundo;
  a lista apróbaa o usuario mensaxe a mensaxe antes de traducir:
  1. compilar a PDF: motor LaTeX que falta, paquete que falta, cambio automático de motor;
  2. Maxima: detectar, instalar e fallos ao calcular;
  3. instalar dependencias desde Lua (LaTeX, ImageMagick e similares);
  4. exportar e previsualizar (PDF, TeX, Markdown, Word, OpenDocument);
  5. abrir e gardar cursos, formularios, proxectos e plantillas, incluído un ficheiro non válido;
  6. actualizacións de Lua e do módulo Buxo;
  7. erros do provedor de IA: falta a clave ou o modelo, sen conexión, resposta non válida, límite superado.
- Edge cases:
  - A parte variable dunha mensaxe (unha ruta, o nome dun paquete, un número) mantense tal cal dentro do texto traducido.
  - A persoa cambia de idioma: as mensaxes que aparecen desde ese momento saen no idioma novo.
  - Unha mensaxe da lista sen tradución nun idioma: sae en galego, coma no resto de Lua.
  - Unha mensaxe que non está na lista: sae en galego, coma hoxe.
  - Unha operación longa que remata despois de cambiar de idioma: a mensaxe sae no idioma activo cando aparece.
  - Unha mensaxe que ve a persoa e que tamén se lle pasa á IA: vai no idioma activo; a IA enténdea igual.
- Error paths: se unha mensaxe inclúe a saída dun programa externo, esa parte non se traduce e o texto que a rodea si.

## Acceptance criteria
- Given Lua en inglés e un equipo sen xelatex, When a persoa xera un informe que o necesita, Then o aviso que ve está en inglés.
- Given Lua en castelán, When a persoa abre un ficheiro que non é un curso válido, Then o erro está en castelán.
- Given Lua en portugués, When falta un paquete de LaTeX, Then a mensaxe está en portugués e inclúe o nome exacto do paquete.
- Given Lua en galego, When aparece calquera mensaxe, Then o texto é idéntico ao de hoxe.
- Given a lista de mensaxes frecuentes, When se pasan os tests de Lua, Then fallan se unha mensaxe da lista non ten tradución nalgún dos catro idiomas ou se as súas partes variables non coinciden entre idiomas.
- Given a lista aprobada, When se conta, Then cada mensaxe pertence a unha das sete situacións de Behaviour.
- Given Lua en inglés e sen clave de API configurada, When a persoa usa o asistente, Then o erro está en inglés.
- Given o plan, When se presenta, Then inclúe a lista completa de mensaxes escollidas para que o usuario a aprobe antes de traducir.

## Points to clarify
- **suposición tomada** — Unhas 70 mensaxes escollidas. *Base:* as sete situacións suman ~209 mensaxes (reconto do 2026-10-10), e escóllense as frecuentes. *Riesgo:* se a lista escollida pasa moito de 70, revísase co usuario.
- **suposición tomada** — As mensaxes que comparten Lua e outras apps a través de `edicion` tradúcense igual para todas. *Base:* constitución 1.3, o motor común non se copia. *Riesgo:* Yang usaría as mesmas traducións, o que é aceptable.
- **decisión diferida** — Traducir os erros raros: quedan en galego neste ciclo. Revísase se alguén informa dun concreto.
- **decisión diferida** — O instalador ten as súas propias traducións e queda fóra.

## Revisions
- 2026-10-10 — Borrador. Alcance decidido polo usuario: só as frecuentes; respostas para a IA e rexistros quedan en galego.
- 2026-10-10 — Aprobada polo usuario («aprobo»), fase a fase (sen autopiloto).

## Clarifications
- 2026-10-10 — «Frecuente» → lista escollida de ~70 dentro das situacións, aprobada mensaxe a mensaxe no plan. *Porqué:* as seis situacións suman ~186 e o usuario quere só as que ve case todo o mundo. (Decisión do usuario; a suposición de 60-80 rompeu e substituíuse.)
- 2026-10-10 — Engádese a situación 7, erros do provedor de IA. *Porqué:* aparecen sempre que a IA falla ou está mal configurada. (Decisión do usuario.)
- 2026-10-10 — A área non formulable (mensaxes que ve a persoa e tamén a IA) gradúase a comportamento: van no idioma activo. *Porqué:* a IA entende os catro idiomas.
- 2026-10-10 — Por defecto (sen obxección): cambio de idioma a metade dunha operación → idioma activo ao aparecer a mensaxe; o instalador fica fóra (ten traducións propias). Motor común en `edicion` → resolto pola constitución 1.3.
