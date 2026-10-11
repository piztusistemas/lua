---
type: spec
title: Spec — Engadir traballos á bóveda
description: WHAT e WHY para que un documento feito en Lúa (boletín, exame, cronograma, actas) se poida engadir á bóveda de Obsidian, creando a do proxecto se non hai ningunha, sempre por decisión do usuario.
tags: [sdd, spec, boveda, obsidian, formularios]
timestamp: 2026-10-11T00:00:00Z
topic: sdd
slug: traballos-boveda
status: planned (aprobada polo usuario o 2026-10-11; resto da cadea en autopiloto)
---

# Spec — Engadir traballos á bóveda

> Slug: `traballos-boveda` · Status: planned · Creada: 2026-10-11
> Inherits: [constitution](../constitution.md) v1.2.1 (5.2: os datos do alumnado só chegan á IA por unha acción de IA)

## Problem & why
Os boletíns, exames e demais documentos que se fan en Lúa quedan só no proxecto
ou nun PDF solto. A bóveda de coñecemento (Obsidian) non os coñece, así que nin
o docente os atopa xunto ao resto do material nin o asistente pode usalos para
preparar outros (un exame parecido, un boletín de repaso, a programación).

## Cost of not building it
O docente ten que copiar a man cada documento á súa bóveda, ou renuncia a
telo alí. O asistente prepara material novo sen ver o que xa se fixo.

## The cheapest alternative
Exportar o Markdown e o PDF co que xa hai e copialos á man no cartafol da
bóveda. Funciona, pero require saber onde está a bóveda, repetilo en cada
cambio e, se o proxecto non ten bóveda, crear a estrutura a man.

## Goals
- Un documento pódese engadir á bóveda cun botón.
- Cada formulario pode marcar «engadir sempre á bóveda ao xerar»; por defecto está desmarcado.
- Se non hai bóveda, Lúa crea a do proxecto soa ao engadir o primeiro traballo.
- Nada vai á bóveda sen que o usuario o pida (botón ou marca).
- Os datos dun formulario en produción nunca van á bóveda.

## Non-goals / out of scope
- Sincronizar cambios feitos en Obsidian de volta a Lúa.
- Escoller o cartafol dentro da bóveda (vai sempre a `wiki/traballos/`).
- Gardar o historial de versións dun traballo na bóveda.

## Users & context
Docentes que preparan boletíns, exames, fichas e informes, e que usan (ou
queren usar) a bóveda como memoria do seu material e como fonte do asistente.

## Behaviour
- **Que bóveda:** a mesma que xa usa o asistente para ese documento: a do proxecto aberto; se o proxecto non ten, a de Obsidian conectada a ese tipo de documento; se non hai ningunha, Lúa crea a do proxecto (coa estrutura de bóveda) e avisa.
- **Que se garda:** unha nota Markdown co documento (o mesmo contido ca a exportación a Markdown, cun encabezado co proxecto, a data e o tipo de documento) e o PDF xerado ao lado, ligado desde a nota.
- **Onde:** `wiki/traballos/<nome do documento>.md` e `.pdf`. Volver engadir o mesmo documento substitúe a nota e o PDF (son o documento de agora). Engádese unha liña ao rexistro da wiki (`log.md`).
- **Botón:** «Engadir á bóveda», xunto ás exportacións. Para engadir o PDF, se o documento non está xerado ou cambiou, xérase antes.
- **Marca por formulario:** «Engadir sempre á bóveda ao xerar». Con ela, cada Xerar que saia ben engade o traballo.
- **Produción:** nun formulario en modo produción, o botón e a marca están desactivados e explican por que: o asistente le a bóveda e os datos reais non deben chegarlle.
- **Erros:** se non se pode escribir na bóveda (permisos, sen espazo), o documento segue igual e a mensaxe di que pasou; non queda unha nota a medias.

## Acceptance criteria
- Given un proxecto sen bóveda e un formulario en deseño xerado, When se preme «Engadir á bóveda», Then o proxecto ten bóveda (con `raw/` e `wiki/`) e `wiki/traballos/` ten a nota e o PDF do documento.
- Given un proxecto con bóveda, When se engade o mesmo documento dúas veces, Then hai unha soa nota e un só PDF, co contido da segunda vez.
- Given un proxecto sen bóveda propia e unha bóveda de Obsidian conectada aos formularios, When se engade un traballo, Then vai á bóveda conectada.
- Given a nota creada, When se abre, Then leva o contido do documento e unha ligazón ao seu PDF.
- Given un formulario coa marca «engadir sempre» activada, When se xera ben, Then o traballo queda na bóveda sen premer o botón.
- Given a marca desactivada (por defecto), When se xera, Then non se escribe nada na bóveda.
- Given un formulario en produción, When se tenta engadir á bóveda (botón, marca ou asistente), Then non se escribe nada e a mensaxe explica o motivo.
- Given o cronograma ou as actas, When se preme «Engadir á bóveda», Then tamén se engaden.
- Given os catro idiomas, Then os textos novos existen en gl, es, en e pt.

## Points to clarify
- **suposición tomada** — Vale para calquera documento (formularios, cronograma e actas), non só para exames. *Base:* «boletín, exame, etc.» e custa o mesmo. *Risco:* ningún.
- **suposición tomada** — A marca «engadir sempre» é dos formularios; o cronograma e as actas só teñen o botón. *Base:* a petición fala de formularios.
- **suposición tomada** — Volver engadir substitúe (sen versións). *Base:* a bóveda debe ter o documento actual. *Risco:* quen queira gardar cada modelo dun exame con sementes distintas terá que cambiarlle o nome.
- **suposición tomada** — `wiki/traballos/` (resposta do usuario): a IA pode editar esas notas ao manter a wiki. *Risco:* unha nota editada pola IA substitúese ao volver engadir o documento.
