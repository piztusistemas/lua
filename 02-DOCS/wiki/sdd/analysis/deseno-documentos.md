---
type: analysis
title: Análise crítica e proxecto de melloras — deseño dos documentos
slug: deseno-documentos
timestamp: 2026-10-16T22:45:00Z
topic: sdd
---

# Análise crítica — deseño dos documentos (entrega do 2026-10-16)

Feita sobre o código, os PDF reais xerados e capturas do editor nun navegador
(non na xanela de Wails: ver a verificación).

## 1. Usabilidade e facilidade de uso

**O que funciona**
- A queixa de partida está resolta: unha letra chega a texto, títulos, etiquetas e
  fórmulas (comprobado con `pdffonts`).
- Os cambios habituais (letra, tamaño, cor, branco e negro, aforrar papel) son
  dun clic, con nomes de docente e sen LaTeX.
- As combinacións amosan unha mostra «Aa» e unha descrición curta: escóllese sen
  saber que é unha serifa.

**Problemas observados**
1. **O deseño non é o protagonista da pantalla.** O bloque vai nunha columna
   estreita ao carón de «Identidade»; hai que baixar moito para chegar aos
   interruptores e ás partes do documento. A vista previa, que é o que importa,
   ocupa só un terzo.
2. **A vista previa non é inmediata.** Ao abrir o editor está baleira ata
   premer «Vista previa». Cada cambio recompila o documento enteiro (2–5 s), sen
   barra de progreso nin cancelar, e a vista amosa a portada primeiro aínda que
   o cambio estea na páxina 4.
3. **O inspector non se ve.** Premer na vista previa funciona, pero nada o
   indica (só un *tooltip*): non hai resalte ao pasar o rato.
4. **As partes máis visibles seguen en LaTeX.** Portada, cabeceira, pé e
   sinatura do PDF non teñen controis visuais: para mover o logotipo hai que
   tocar LaTeX ou pedírllelo á IA.
5. **Choque de nomes.** As barras chámanse «Estilo» e o bloque de LaTeX libre
   «Estilo LaTeX»; «papel» (letra de cada papel) é xerga.
6. **A mostra «Aa» é unha aproximación** coas fontes do sistema: en Windows
   «Elegante» e «Académica» poden verse iguais na interface.
7. **As barras sobrescriben valores.** Mover «Formal» cambia a letra; volver ao
   medio restaura a de Lúa, non a que había. Hai «Desfacer», pero sorprende.
8. **As variantes son lentas**: cada rolda compila dous documentos enteiros.

## 2. Control do produto final

**O que funciona**
- Control fino sen perder o sinxelo: cada parte ten «Editar en LaTeX» (parte do
  LaTeX xerado e substitúe só esa parte) e o LaTeX libre enteiro segue existindo.
- Transparencia: o deseño vese e edítase como JSON; a IA só pode cambiar datos
  validados; os cadeados protexen o que decide o usuario.
- O corrector avisa sempre do que cambia («Arranxei…»).

**Límites**
1. Un LaTeX propio dun elemento só se valida ao compilar o documento enteiro; o
   erro non sinala o elemento.
2. Word/ODT só herdan letra, tamaño, interliñado e cores: seguen sendo dous
   editores para un mesmo documento.
3. **Fontes:** nun TeX Live mínimo (o que pon o instalador en Debian) faltan
   Fira e Libertinus. Lúa usa a substituta e avisa, pero «Moderna» e
   «Académica» acaban vendo como «Lúa» e «Clásica»: o control prometido non se
   cumpre nese equipo.
4. Unha plantilla completa que fixa a súa letra non a cambia (avísase): correcto,
   pero o docente non sabe por que a súa plantilla «non obedece».

## 3. Riscos técnicos
- `exec` de LaTeX sen tempo máximo e vista previa sen cancelar (constitución 5.4
  e 7.1): xa era así; agora pésalle máis porque se compila a cada cambio.
- `task dev` non funciona (falta `build:dev`), así que non se probou na xanela real.
- Yang comparte `edicion/latex` e recibe a letra por papeis sen probas alí.
- Erro anterior atopado e arranxado: a portada por defecto perdía os espazos do
  título (o tamaño cambiaba dentro do nodo de TikZ) e ía sempre en índigo.

# Proxecto de melloras

Ordenado por impacto para o profesorado. Cada punto é unha spec curta.

## Fase A — «Estudio de deseño» (impacto alto, esforzo medio)
- **A1 Pantalla propia para o deseño:** controis á esquerda (unha columna, sen
  scroll para o esencial: letra, tamaño, cor, interruptores) e vista previa
  grande á dereita. Os bloques técnicos pasan a «Avanzado».
- **A2 Vista previa viva e rápida:** compilar ao abrir; documento de mostra curto
  (unha páxina por parte); saltar á páxina do elemento cambiado; progreso e
  cancelar (cumpre 7.1); tempo máximo para LaTeX (5.4).
- **A3 Inspector visible:** resalte da parte baixo o rato (rectángulos a partir
  das marcas) e etiqueta co seu nome.
- **A4 Nomes:** «Estilo» → «Carácter do documento»; «Letra de cada papel» →
  «Letra de cada parte»; revisión de vocabulario con docentes.

## Fase B — Sen LaTeX nas partes visibles (impacto alto, esforzo alto)
- **B1 Portada como elemento:** 3–4 modelos (cor completa, sobria, con foto,
  oficial) con parámetros (posición do logotipo, cor, textos).
- **B2 Cabeceira e pé visuais:** logotipos e textos en tres posicións, a mesma
  linguaxe ca a de Word (`@imaxe`, `@esquerda`…), que xere o LaTeX.
- **B3 Un só deseño para PDF, Word e ODT:** o deseño manda e Word/ODT tradúceo;
  o bloque de Word queda para excepcións.

## Fase C — Fiabilidade das letras (impacto medio, esforzo baixo)
- **C1 Embeber as letras das combinacións** (OFL: Fira Sans e Math, Libertinus,
  Atkinson Hyperlegible para «Lexible») como xa se fai con Open Sans: o PDF sae
  igual en calquera equipo.
- **C2 Mostras reais:** servir esas mesmas fontes á interface para que «Aa» sexa a letra de verdade.

## Fase D — Kit do centro (impacto medio, esforzo medio)
- **D1 Deseño do centro:** exportar e importar só o deseño (con logotipo) e
  «aplicar a todas as miñas plantillas»; a persoa que coordina prepárao unha vez.
- **D2 Barras que lembran o punto de partida** de todos os valores (coma a da cor).

## Fase E — Control avanzado seguro (impacto medio, esforzo baixo)
- **E1 Validar o LaTeX dun elemento** compilando só esa parte nun documento
  mínimo, co erro ao carón do elemento.
- **E2 Teclado:** frechas na escolla de letra e foco ao abrir un elemento desde o inspector.

## Fase F — Validación (imprescindible antes de dar por boa a interface)
- **F1 Proba con 5 docentes**, con tarefas cronometradas: cambiar a letra, a
  cor do centro, imprimir en B/N, poñer os títulos en maiúsculas, copiar o
  deseño dun PDF. Éxito: < 1 minuto por tarefa, sen axuda e sen ver LaTeX.
- **F2 Arranxar `task dev`** para probar na xanela real.
