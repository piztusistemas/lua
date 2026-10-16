---
type: spec
title: Spec — Deseño dos documentos sen LaTeX e sen perder o control
description: WHAT e WHY para que o profesorado deseñe o PDF (letra, cores, tamaños, elementos) cun editor visual, con IA e con axudas automáticas, mantendo o LaTeX a un clic.
tags: [sdd, spec, plantillas, deseño, tipografía, pdf]
timestamp: 2026-10-16T00:00:00Z
topic: sdd
slug: deseno-documentos
status: planned (autopiloto pedido polo usuario o 2026-10-16)
---

# Spec — Deseño dos documentos

> Slug: `deseno-documentos` · Status: planned · Creada: 2026-10-16
> Inherits: [constitution](../constitution.md) v1.2.1

## Problem & why
O aspecto do PDF depende de LaTeX (bloque «Estilo» da plantilla) e dun panel
«Axustes do texto» que só cambia parte da letra. O profesorado non sabe LaTeX.
Ademais, cambiar a letra non se ve ben no PDF:

- só cambia o corpo; os títulos seguen en Latin Modern Sans e as etiquetas en Open Sans;
- «Sans» con xelatex é a mesma letra ca a por defecto;
- as fórmulas non cambian de letra;
- o tamaño non escala títulos nin notas;
- unha plantilla completa ignora a letra sen avisar;
- a letra gárdase no navegador, non na plantilla, e non viaxa con ela.

## Cost of not building it
Os documentos saen con tres familias de letra mesturadas, o profesorado depende
de quen sabe LaTeX e as plantillas compartidas non saen igual noutro equipo.

## The cheapest alternative
Arranxar só a letra (fase 0). Resolve a queixa concreta, pero o deseño segue en LaTeX.

## Goals
1. **A letra escollida chega a todo o documento**: corpo, títulos, etiquetas e fórmulas.
2. **O deseño vive na plantilla** como datos (`deseno`), non en LaTeX nin no navegador.
3. **Editor visual «Deseño»** dentro do editor de plantillas, o primeiro bloque á vista:
   - letra por combinacións con nome (Lúa, Clásica, Elegante, Lexible, Moderna);
   - tamaño, cor principal e cor secundaria;
   - **tomar as cores do logotipo**;
   - **tres barras**: Formal↔Informal, Compacto↔Aireado, Sobrio↔Colorido;
   - **interruptores do mundo real**: branco e negro, aforrar papel, lectura fácil;
   - **elementos** (títulos, etiquetas, meses, trimestres, tarefas, actas, sinaturas) con poucos controis e «Editar en LaTeX»;
   - **premer na vista previa** abre o elemento que hai nese punto;
   - **probar variantes**: dúas propostas lado a lado; escóllese unha.
4. **Corrector automático**: Lúa evita liñas soltas, escurece cores sen contraste e
   avisa de texto que sae da marxe, e dio («Arranxei N cousas»).
5. **IA**: o asistente da plantilla cambia o deseño como datos validados; existe un
   documento de deseño (`DESENO-SKILL.md`) e un JSON Schema que valen para calquera IA.
6. **Fai que se pareza a isto**: a partir dun PDF ou Word, Lúa propón letra e cores.
7. **Control total**: o LaTeX libre segue existindo (por elemento ou enteiro).

## Non-goals / out of scope
- Deseño de Word/ODT desde o mesmo editor visual: comparten letra, tamaño e cores
  (o deseño escribe tamén `ofimatica`), pero o resto de Word segue no seu bloque.
- PDF accesible etiquetado (PDF/UA): o soporte de LaTeX aínda é experimental.
- Catálogo de fontes descargable: úsanse as de TeX Live e as que leva Lúa, con substitución se faltan.
- Exames de Yang (`edicion/documento`): só reciben o arranxo da letra (fase 0).

## Users & context
Docentes sen LaTeX que queren un documento correcto e coherente; persoas «manitas»
que preparan a plantilla do departamento e queren control fino.

## Behaviour
- Unha plantilla sen `deseno` e sen `estilo` sae co deseño de Lúa (o de sempre, coa letra coherente).
- Se a plantilla ten `estilo` (LaTeX libre), manda o LaTeX libre; o editor amósao como «Deseño en LaTeX libre» e ofrece «Pasar ao editor visual».
- Se a plantilla é completa e declara a súa letra, a letra do deseño non se aplica e o resultado avisa.
- A barra «Axustes do texto» da xanela principal segue valendo para un cambio rápido; se a plantilla ten deseño, o deseño manda e o panel dío.
- As fontes cárganse por ficheiro (TeX Live / Lúa). Se falta unha, úsase a máis parecida e avísase.

## Acceptance criteria
- AC1 Coa combinación «Clásica», `pdffonts` do PDF só amosa TeX Gyre Termes (+ math) e as etiquetas; non aparece Latin Modern Sans.
- AC2 Co tamaño 13 pt, `\normalsize` é 13 pt e os títulos medran na mesma proporción.
- AC3 O deseño gárdase en `plantillas.json` e viaxa ao exportar a plantilla.
- AC4 Branco e negro: ningunha cor do bloque dinámico ten saturación.
- AC5 O corrector escurece unha cor principal sen contraste (ratio < 4.5 sobre branco) e avisa.
- AC6 Premer nun título da vista previa abre o elemento «Títulos».
- AC7 O asistente pode cambiar `deseno` e un valor fóra do esquema rexéitase.
- AC8 As cores dun logotipo PNG dan unha cor principal non gris.
- AC9 Un PDF de mostra con Palatino propón a combinación «Elegante».
- AC10 Todas as claves novas en gl, es, en e pt.
