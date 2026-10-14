---
type: spec
title: Spec — Proxectos lixeiros, ventá de formularios para reutilizar e solucións só co resultado
description: WHAT e WHY para simplificar o panel de proxectos, xestionar os formularios dun proxecto desde unha ventá con cartafoles (manual e co asistente) e engadir solucións só co resultado.
tags: [sdd, spec, proxectos, formularios, problemas]
timestamp: 2026-10-11T00:00:00Z
topic: sdd
slug: reutilizar-formularios
status: planned (aprobada polo usuario o 2026-10-11; resto da cadea en autopiloto)
---

# Spec — Proxectos lixeiros, ventá de formularios e solucións só co resultado

> Slug: `reutilizar-formularios` · Status: planned · Creada: 2026-10-11
> Inherits: [constitution](../constitution.md) v1.2.1 · Continúa: [trasladar](trasladar.md) (os datos sempre nun proxecto)

## Problem & why
O panel de Proxectos mestura dúas tarefas: crear e abrir proxectos, e escoller os
seus formularios. A lista de formularios do «Novo proxecto» medra con cada
formulario e faise incómoda. Os proxectos existentes amósanse enteiros, cos seus
botóns, e ocupan moito. Ademais, un formulario creado dentro dun proxecto non
chega á biblioteca común, así que non se pode reutilizar noutro. E nos exames
con problemas non hai forma de dar só o resultado de cada apartado sen a
resolución paso a paso.

## Cost of not building it
O panel de proxectos faise máis lento de usar con cada formulario novo. Os
formularios creados en proxectos quedan illados: para usalos noutro proxecto hai
que volver deseñalos ou trasladalos a man. Quen quere un exame con resultados
para autocorrección ten que editar o documento fóra.

## The cheapest alternative
Pregar a lista de formularios do «Novo proxecto» e engadir un buscador. Reduce o
problema visual, pero segue sen biblioteca común nin cartafoles, e non resolve as
solucións.

## Goals
- Crear un proxecto pide só o nome (e a materia opcional): crea un proxecto baleiro.
- Os proxectos existentes saen pregados (só nome, data e materia); ao premer, desprégase o resto.
- A xestión dos formularios dun proxecto faise nun só sitio: a ventá que abre «Deseñar formulario».
- Esa ventá amosa a biblioteca común (cronograma, actas e todos os formularios creados) en cartafoles; o usuario marca e preme «Reutilizar».
- Os cartafoles xestiónanse a man e tamén co asistente.
- Todo formulario que se garda nun proxecto queda tamén na biblioteca común.
- Nos documentos con problemas hai dúas opcións novas: só o resultado debaixo de cada pregunta, e só os resultados ao final.

## Non-goals / out of scope
- Cambiar o deseñador de formularios por dentro.
- Sincronizar un formulario reutilizado coa biblioteca despois (o proxecto leva a súa copia, como ata agora).
- Borrar formularios da biblioteca común desde un proxecto (quitar dun proxecto non toca a biblioteca).
- Cambiar as opcións de solucións que xa existen.

## Users & context
Docentes con varios proxectos que reutilizan os mesmos formularios (fichas,
exames, informes) e queren telos ordenados por temas.

## Behaviour
### Panel de Proxectos
- «Novo proxecto»: nome e materia; «Crear» crea un proxecto sen ferramentas (sen abrilo), que sae despregado, e a columna pasa a «Configurar»: cartafol do equipo onde se garda («Escoller cartafol…»), fontes de coñecemento (Buxo) e «Abrir proxecto» ou «Agora non».
- Cada proxecto existente sae nunha liña pregada (nome, data, materia). Ao premer desprégase: ferramentas, bóveda e botóns (Abrir, cartafol, Fontes de coñecemento, Eliminar).
- Non hai botón «Ferramentas…».

### Ventá de formularios («Deseñar formulario», botón + da columna)
- Amosa a biblioteca común: arriba Cronograma e Actas; despois os formularios, agrupados en cartafoles (e «Sen cartafol»).
- Os que xa están no proxecto aberto saen marcados como «Xa no proxecto» e non se poden marcar outra vez.
- O usuario marca un ou varios e preme «Reutilizar»: entran no proxecto aberto (cunha copia) e aparecen na columna.
- Botón «Novo formulario» abre o deseñador (en branco ou co asistente, como agora).
- Cartafoles: crear, renomear, eliminar (os formularios pasan a «Sen cartafol») e mover un formulario a un cartafol.
- Busca por nome.

### Columna da esquerda
- Botón dereito → «Quitar do proxecto»: saca a ferramenta do proxecto (e os seus datos, con confirmación). A biblioteca común non cambia.

### Biblioteca común
- Gardar un formulario no deseñador dentro dun proxecto deixa tamén unha copia na biblioteca común (se xa existe co mesmo id e non está en produción, actualízase).

### Asistente
- Pode listar a biblioteca común e os seus cartafoles, crear e renomear cartafoles, mover formularios entre cartafoles e reutilizar formularios no proxecto aberto. Os cambios que non se poden desfacer piden confirmación, como as demais accións.

### Solucións dos problemas
- Opcións: Non (exame) · Debaixo de cada pregunta · Solucionario ao final · **Só o resultado, debaixo de cada pregunta** · **Só os resultados, ao final**.
- As dúas novas amosan só o resultado final de cada pregunta, sen a resolución.
- Unha pregunta con solución pero sen resultado separado: avisa e non amosa nada para esa pregunta.

## Acceptance criteria
- Given o panel de proxectos, When se crea un proxecto co nome «X», Then existe un proxecto «X» sen ferramentas e queda aberto.
- Given o panel de proxectos, When se abre, Then non hai lista de formularios no «Novo proxecto» nin botón «Ferramentas…».
- Given proxectos existentes, When se abre o panel, Then cada un sae pregado; When se preme nun, Then amosa os seus botóns.
- Given un proxecto aberto, When se preme +, Then abre a ventá coa biblioteca común (cronograma, actas e formularios por cartafoles).
- Given a ventá, When se marcan dous formularios e se preme «Reutilizar», Then os dous están no proxecto e na columna.
- Given un formulario que xa está no proxecto, When se abre a ventá, Then sae como «Xa no proxecto» e non se pode marcar.
- Given un formulario gardado no deseñador dentro dun proxecto, When se abre a ventá noutro proxecto, Then ese formulario aparece na biblioteca.
- Given un cartafol con formularios, When se elimina, Then os formularios seguen na biblioteca, en «Sen cartafol».
- Given o asistente, When se lle pide mover un formulario a un cartafol novo, Then o cartafol existe e o formulario está nel.
- Given un formulario na columna, When se escolle «Quitar do proxecto» e se confirma, Then sae do proxecto e segue na biblioteca.
- Given un problema con resultado, When se xera con «Só o resultado, debaixo», Then o documento ten o resultado baixo a pregunta e non a resolución.
- Given o mesmo, When se xera con «Só os resultados, ao final», Then hai unha lista final só cos resultados.
- Given as opcións vellas, When se xera, Then o documento sae igual ca antes.
- Given os catro idiomas, Then os textos novos existen en gl, es, en e pt.

## Points to clarify
- **suposición tomada** — Os proxectos parten de baleiro e a ventá de formularios é a única porta para engadir ferramentas (cronograma e actas incluídos). *Base:* respostas do usuario o 2026-10-11.
- **suposición tomada** — Os cartafoles son da biblioteca común (non de cada proxecto). *Base:* a ventá amosa a biblioteca común. *Risco:* os cartafoles que xa existían dentro dalgún proxecto deixan de verse nesa ventá.
- **suposición tomada** — A copia na biblioteca ao gardar non pisa un formulario da biblioteca en produción: entón crea unha copia «Nome (2)». *Base:* regra de non pisar nada (trasladar).
- **decisión diferida** — Arrastrar formularios entre cartafoles co rato: de momento, cun selector.
