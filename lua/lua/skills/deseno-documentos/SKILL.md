---
name: deseno-documentos-lua
description: Deseñar o PDF dunha plantilla de Lúa (letra, tamaño, cores, espazos e cada parte do documento) cambiando só o seu deseño en JSON. Úsaa cando che pidan que un documento de Lúa sexa máis formal, máis lexible, doutra cor, que se pareza a outro ou que gaste menos papel.
---

# Deseño de documentos de Lúa

Lúa xera documentos docentes en PDF (cronograma anual, actas, formularios) con LaTeX.
O aspecto **non se escribe en LaTeX**: é un obxecto JSON, o **deseño**, e Lúa xera o
LaTeX a partir del. Ti só propós ese JSON. Lúa valídao e, se algo non vale, rexéitao
cunha mensaxe que di que corrixir.

## Como responder

- Devolve **só os campos que cambian** (un parche), non o deseño enteiro.
- Un campo baleiro (`""`) ou `0` volve ao valor de Lúa.
- Non toques os campos de `fixados`: decidiunos o usuario.
- Non escribas LaTeX (`elementos.*.latex`) salvo que o usuario o pida expresamente.

## Os campos

| Campo | Que é | Valores |
|---|---|---|
| `letra` | Combinación de letras para texto, títulos, etiquetas e fórmulas | `lua` (clara, a de sempre), `clasica` (tipo Times, formal), `elegante` (tipo Palatino), `lexible` (tipo Century, a máis fácil de ler), `moderna` (Fira Sans), `academica` (Libertinus, boa para fórmulas) |
| `letraCorpo`, `letraTitulos`, `letraEtiquetas` | Letra dun só papel | `termes`, `pagella`, `schola`, `bonum`, `libertinus`, `sourceserif`, `heros`, `adventor`, `firasans`, `opensans` |
| `letraMates` | Letra das fórmulas | `lm`, `termes`, `pagella`, `schola`, `bonum`, `fira`, `libertinus` |
| `tamano` | Tamaño do texto en pt | 8–20 (Lúa: 11). Todo escala con el |
| `escala` | Tamaño dos títulos respecto do texto | 0.6–1.8 |
| `interlinado` | Espazo entre liñas | 0.9–2.2 (1 = sinxelo) |
| `parrafo` | Espazo entre parágrafos (em) | 0–3 (Lúa: 0.55) |
| `marxe` | Marxe da páxina (mm) | 8–40 (Lúa: 20) |
| `corPrincipal` | Cor de títulos, liñas e rótulos | hex sen `#` |
| `corEscura` | Cor dos rótulos (se non, sae da principal) | hex |
| `corTexto` | Cor do texto | hex |
| `etiquetas` | Aspecto das etiquetas do cronograma | `degradado`, `planas`, `contorno` |
| `brancoNegro` | Preparar para imprimir en branco e negro | `true`/`false` |
| `aforrarPapel` | Compactar para gastar menos follas | `true`/`false` |
| `lecturaFacil` | Letra grande, máis espazo, sen xustificar | `true`/`false` |
| `portada` | Modelo da portada | `""` (de cor, a de Lúa), `sobria`, `oficial`, `sen` |
| `logoPortada` | Onde vai o logotipo na portada | `""`, `esquerda`, `centro`, `dereita` |
| `cabeceira`, `pe` | Texto das zonas `esquerda`, `centro`, `dereita` | texto con `{{CENTRO}}`, `{{CURSO}}`, `{{DEPARTAMENTO}}`…, `@logo`, `@imaxe nome.png` |
| `numeroPaxina` | Onde vai o número de páxina | `""` (dereita), `esquerda`, `centro`, `non` |
| `elementos.<id>` | Axuste dunha parte: `cor`, `escala` (0.5–2.5), `maiusculas`, `senFilete`, `latex` | ids: `texto`, `titulos`, `etiquetas`, `meses`, `trimestres`, `tarefas`, `actas_cabeceira`, `actas_apartados`, `actas_materias`, `sinaturas` |

## Bo deseño (regras que Lúa non comproba e ti si)

1. **Unha ou dúas familias de letra.** Escolle unha combinación (`letra`) antes de cambiar letras por papel.
2. **Contraste.** O texto de cor ten que lerse sobre branco (Lúa escurece as cores claras e avísao).
3. **Xerarquía.** Os títulos máis grandes ca o texto: `escala` entre 0.9 e 1.3 adoita bastar.
4. **Aire.** Para documentos longos, `interlinado` 1.1–1.25. Para impresión barata, `aforrarPapel`.
5. **Accesibilidade.** Para alumnado con dificultades de lectura: `lecturaFacil: true` e `letra: "lexible"`.
6. **Coherencia co centro.** Se hai logotipo, a `corPrincipal` sae del.

## Exemplos

«Máis formal e sobrio»:
```json
{"letra": "clasica", "etiquetas": "planas", "corPrincipal": "334155", "escala": 0.95}
```

«Que se lea mellor na pantalla do proxector»:
```json
{"tamano": 14, "interlinado": 1.25, "letra": "lexible"}
```

«Portada sobria e o logotipo e o centro na cabeceira»:
```json
{"portada": "sobria", "cabeceira": {"esquerda": "@logo {{CENTRO}}", "dereita": "{{DEPARTAMENTO}}"}}
```

«Os títulos en maiúsculas e sen a liña de cor»:
```json
{"elementos": {"titulos": {"maiusculas": true, "senFilete": true}}}
```
