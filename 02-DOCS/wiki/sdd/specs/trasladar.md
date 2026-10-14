---
type: spec
title: Spec — Trasladar: un só sitio para sacar e meter cousas en Lúa
description: WHAT e WHY dunha interface única para exportar e importar datos, formularios, plantillas, proxectos, bóvedas e copias de seguridade, cos datos sempre dentro dun proxecto.
tags: [sdd, spec, exportar, importar, proxectos, boveda]
timestamp: 2026-10-11T00:00:00Z
topic: sdd
slug: trasladar
status: planned (aprobada polo usuario o 2026-10-11; resto da cadea en autopiloto)
---

# Spec — Trasladar: un só sitio para sacar e meter cousas en Lúa

> Slug: `trasladar` · Status: planned · Creada: 2026-10-11 · Aprobada: «Aprobada, lévaa ata o final en autopiloto» (2026-10-11)
> Inherits: [constitution](../constitution.md) v1.2.1 (5.1 claves fóra das exportacións; 5.2 datos no equipo; 3.2 textos en catro idiomas; 6.1 teclado; 6.2 erros que din que facer; 8.2 axuda)

## Problem & why
O usuario non sabe que garda cada botón. «Gardar» e «Gardar como» escriben os
datos nun ficheiro externo, pero parecen obrigatorios para non perder o
traballo, que en realidade Lúa xa garda só. «Gardar formulario» garda a
estrutura. «Exportar formulario» e «Exportar»/«Importar» de plantillas están
noutros sitios. Un proxecto enteiro (formularios, datos e bóveda) non se pode
sacar de Lúa nin para compartilo nin como copia de seguridade. O resultado:
dúbidas en cada acción, medo a perder traballo e ningunha forma de facer copia.

## Cost of not building it
O docente segue sen copia de seguridade: se o equipo falla, perde proxectos e
bóvedas (os PDF e a wiki que construíu). Non pode pasarlle un proxecto a un
compañeiro. Os botóns de gardar seguen xerando dúbidas e consultas. Non hai
perda de datos inmediata: o traballo en curso xa se garda só.

## The cheapest alternative
Cambiar só os textos: «Gardar» → «Exportar datos…» e un indicador «Gardado en
Lúa». Aclara a ambigüidade de gardar, pero deixa as importacións e exportacións
espalladas e non dá forma de sacar un proxecto, unha bóveda nin unha copia
completa.

## Goals
- Todo o que se pode sacar de Lúa ou meter nela faise dende un único sitio, co mesmo patrón para calquera tipo de contido.
- Sacar o que está aberto custa un clic; meter un ficheiro custa soltalo ou escollelo e confirmar.
- Lúa decide onde vai cada cousa que entra, e crea un proxecto só cando fai falta.
- Os datos sempre pertencen a un proxecto; o usuario sabe en todo momento que o seu traballo está gardado en Lúa.
- Un documento (cronograma, actas ou formulario propio) viaxa co que precisa para abrirse igual noutro equipo: datos, estrutura do formulario e plantilla.
- Un documento pódese sacar como ficheiro editable de texto (`.json`), modificar fóra e volver meter actualizando o orixinal.
- Hai copia de seguridade completa e restauración.

## Non-goals / out of scope
- Sincronización na nube ou entre equipos en tempo real.
- Copias de seguridade automáticas programadas (pódese pedir despois).
- Editar un ficheiro externo «en vivo» (o ficheiro vinculado con Ctrl+S desaparece).
- Importar CSV nunha táboa: segue dentro da táboa, porque é edición.
- Conectar unha bóveda de Obsidian externa: segue sendo un vínculo, en Opcións, non un traslado.
- Converter o cronograma ou as actas en formularios propios.
- Exportar o resultado xerado (PDF, Word, ODT, Markdown, LaTeX): é outra función e non cambia.

## Users & context
Docentes que usan Lúa no seu equipo para documentación de curso (cronogramas,
actas, formularios propios) agrupada en proxectos, ás veces cunha bóveda de
fontes. Queren pasarlle material a compañeiros, traer o que outros lles pasan,
gardar unha copia por se acaso e, ás veces, retocar datos fóra de Lúa. Inclúe
quen recibe ficheiros de Piztu Orienta con varios cursos.

## Behaviour

### O modelo
- Dentro de Lúa todo se garda só. A interface amosa sempre o estado («Gardado en Lúa») e nunca un «sen gardar» alarmante.
- Sempre hai un proxecto aberto: non existe traballo solto. *(2026-10-14: Lúa arrinca no inicio, sen proxecto aberto e sen editor á vista; os datos seguen sendo sempre dun proxecto. Ver `decisions.md`, «Inicio nos dous paneis».)*
- A palabra «Gardar» só aparece en editores que se poden descartar (deseñador de formularios, editor de plantillas) e sempre co nome do obxecto.

### O sitio único: Trasladar
- Un botón da barra e un atallo de teclado abren Trasladar. Soltar un ficheiro ou cartafol en calquera parte de Lúa abre Trasladar co ficheiro xa escollido.
- Os botóns e menús espallados de abrir, gardar como, exportar e importar formularios e exportar e importar plantillas desaparecen. Un menú contextual pode abrir Trasladar coa selección feita, pero non fai a acción por si mesmo.
- **Sacar.** Vén preseleccionado o que está aberto. Opcións: este documento, este proxecto, todo Lúa (copia de seguridade) ou escoller nunha árbore (proxectos, formularios, plantillas, bóvedas). Unha marca decide se van os datos escritos.
- **Formato ao sacar.** Por defecto, un ficheiro Lúa (`.lua`) que pode levar calquera combinación. Para un só documento ofrécese tamén «JSON editable» (`.json`), que leva datos, estrutura e plantilla nun ficheiro de texto e non leva PDF nin bóveda.
- **Meter.** Lúa le o ficheiro e amosa en resumo o que leva e onde vai ir, antes de facer nada. Un botón confirma.

### Onde vai o que entra
| Entra | Vai a |
|---|---|
| Un proxecto | Proxecto novo |
| Formulario sen datos | Biblioteca |
| Formulario con datos | Proxecto novo co nome do formulario; o formulario tamén queda na biblioteca |
| Cronograma ou actas que non veñen deste Lúa | O proxecto aberto; se xa ten ese documento, proxecto novo |
| Documento que saíu deste Lúa e volve | Ofrécese «Actualizar» (por defecto) ou «Engadir como copia» |
| PDF, Markdown ou cartafol de bóveda | A bóveda do proxecto aberto |
| Plantilla | Biblioteca de plantillas |
| Copia de seguridade completa | Restaurar todo |
| Ficheiro con varios cursos | Lista con todos marcados; cada curso escollido crea o seu proxecto |

### Regras comúns
- Nunca se sobrescribe nada en silencio. Un nome repetido entra como «Nome (2)». Un formulario idéntico a un existente reutilízase.
- Actualizar un documento ou restaurar unha copia garda antes a versión anterior e permite desfacer.
- Só hai unha pregunta adicional cando os datos poden ser reais (formulario en PRODUCIÓN) ou cando se restaura unha copia completa.
- As claves de API nunca saen nun ficheiro.
- Os ficheiros `.json` vellos (cursos, formularios e plantillas exportados por versións anteriores) seguen entrando.

### Migración
- Ao abrir a nova versión por primeira vez, o traballo solto (cronograma, actas e datos de formularios fóra de proxectos) pasa a un proxecto «O meu traballo». Antes faise unha copia. Lúa avisa unha vez do que moveu.
- Un curso que estaba vinculado a un ficheiro externo queda dentro do proxecto; o ficheiro externo non se toca.

### Erros
- Un ficheiro que non é de Lúa, está danado ou é dunha versión futura descoñecida: non entra nada, e a mensaxe di que pasou e que facer.
- Se falla a metade, non queda nada a medias: ou entra todo o que se confirmou, ou nada.
- Sen espazo ou sen permiso ao sacar: mensaxe co motivo; non se crea un ficheiro incompleto.

## Acceptance criteria
- Given un proxecto aberto, When o usuario abre Trasladar e confirma «Sacar» sen cambiar nada, Then obtén un ficheiro `.lua` con ese proxecto: formularios, datos e bóveda.
- Given ese ficheiro `.lua`, When se mete noutro Lúa, Then aparece un proxecto novo co mesmo contido e a bóveda cos mesmos PDF e páxinas.
- Given un cronograma aberto, When se saca como «JSON editable», se cambia un dato no ficheiro e se mete de novo, Then Lúa ofrece «Actualizar» por defecto e, ao aceptar, o cronograma amosa o dato novo e pódese desfacer.
- Given o mesmo caso, When o usuario escolle «Engadir como copia», Then o orixinal non cambia e aparece un proxecto novo coa copia.
- Given ningún formulario con ese nome, When se mete un formulario con datos, Then hai un proxecto novo co nome do formulario cos datos e o formulario está na biblioteca.
- Given un formulario sen datos, When se mete, Then só aparece na biblioteca e non se crea proxecto.
- Given un formulario con datos de PRODUCIÓN, When se saca marcando os datos, Then Lúa avisa antes de que os datos saen do equipo.
- Given calquera ficheiro sacado, When se inspecciona o contido, Then non hai ningunha clave de API.
- Given un `.json` de curso, formulario ou plantilla exportado por unha versión anterior, When se mete, Then entra no destino que marca a táboa.
- Given un ficheiro de Piztu Orienta con tres cursos, When se mete e se desmarca un, Then aparecen dous proxectos novos.
- Given unha copia completa, When se restaura, Then o estado é o da copia e o estado anterior pódese recuperar.
- Given un ficheiro danado, When se mete, Then non cambia nada en Lúa e a mensaxe di que facer, no idioma do usuario.
- Given traballo solto dunha versión anterior, When se abre a nova versión, Then ese traballo está nun proxecto «O meu traballo», con todos os datos, e aparece un aviso unha soa vez.
- Given calquera pantalla de Lúa, When o usuario busca abrir, gardar como, exportar ou importar formularios ou plantillas, Then a única entrada é Trasladar (ou un menú contextual que abre Trasladar).
- Given calquera lugar con datos, When o usuario mira o estado, Then ve «Gardado en Lúa» e nunca «sen gardar».
- Given o teclado só, When o usuario abre Trasladar, saca e mete un ficheiro, Then pode facelo todo sen rato (salvo soltar ficheiros, que ten alternativa con selector).
- Given os catro idiomas, When se revisan os textos novos, Then existen en `gl`, `es`, `en` e `pt`, e a axuda explica Trasladar.

## Points to clarify
- **suposición tomada** — «Documento autosuficiente»: cronograma, actas e formulario propio viaxan con datos, estrutura e plantilla. *Base:* resposta do usuario o 2026-10-11. *Risco:* se a plantilla leva imaxes grandes, o `.json` editable faise pesado.
- **suposición tomada** — Ao volver un documento que saíu deste Lúa ofrécese «Actualizar» por defecto e gárdase a versión anterior. *Base:* resposta do usuario. *Risco:* se o recoñecemento falla, entra como copia; hai que decidir que conta como «o mesmo documento».
- **suposición tomada** — O traballo solto migra só a «O meu traballo», con copia previa e aviso. *Base:* resposta do usuario. *Risco:* se xa existe un proxecto con ese nome, entra como «O meu traballo (2)».
- **suposición tomada** — Ficheiros con varios cursos: lista con todos marcados, un proxecto por curso. *Base:* resposta do usuario.
- **suposición tomada** — Ao abrir Lúa sen proxectos (primeira vez), créase un proxecto baleiro para empezar. *Base:* «sempre hai un proxecto aberto». *Risco:* pode sorprender a quen só quere probar. *(Substituída o 2026-10-14: xa non se crea; o inicio amosa «Novo proxecto».)*
- **suposición tomada** (clarify, autopiloto) — Destino por defecto segundo a táboa; o resumo de «Meter» ten un selector «Destino» (proxecto aberto / proxecto novo) para cambialo sen máis preguntas. *Base:* intervención mínima sen quitar control. *Risco:* ningún; un clic opcional.
- **suposición tomada** (clarify, autopiloto) — Un `.json` vello de curso ou de actas segue a mesma regra ca un documento: o proxecto aberto se non ten ese documento, se non proxecto novo. *Base:* coherencia coa táboa.
- **suposición tomada** (clarify, autopiloto) — «Desfacer» garda as 5 últimas versións anteriores (actualizacións, restauracións e migración); a máis vella bórrase ao crear a sexta. *Base:* abonda para corrixir un erro recente sen encher o disco. *Risco:* unha copia completa grande ocupa espazo cinco veces.
- **decisión diferida** — Copias de seguridade automáticas periódicas: fóra deste ciclo.
- **área non formulable** — Compatibilidade cara adiante: que fai unha versión vella de Lúa cun `.lua` dunha versión nova. Hai algo aquí, pero aínda non se pode enunciar ben.

## Revisions
- 2026-10-11 — clarify en autopiloto: pechadas as tres preguntas abertas como suposicións (destino con selector, `.json` vello coma documento, 5 versións para desfacer).
