package main

// Estrutura da bóveda de Obsidian (método Karpathy, «LLM Wiki»):
//
//	<bóveda>/
//	  pdf/      os PDF orixinais (copiados ou descargados)
//	  raw/      os PDF convertidos a Markdown (fontes; a IA só as le)
//	  wiki/     a wiki que constrúe e mantén a IA (index.md, log.md, páxinas)
//	  buxo.md   o esquema: convencións e fluxos de traballo da IA
//	  buxo.log  rexistro das operacións de Buxo (non é Markdown: Obsidian non o indexa)
//
// A estrutura non se crea ata que hai PDF que tratar.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	dirPDF          = "pdf"
	dirRaw          = "raw"
	dirWiki         = "wiki"
	ficheiroEsquema = "buxo.md"
	ficheiroLog     = "buxo.log"
)

// CrearEstrutura crea os cartafoles e buxo.md se faltan. Non pisa nada que xa
// exista (buxo.md pode estar editado polo usuario).
func CrearEstrutura(raiz, materia string) error {
	for _, d := range []string{dirPDF, dirRaw, dirWiki} {
		if err := os.MkdirAll(filepath.Join(raiz, d), 0o755); err != nil {
			return err
		}
	}
	esquema := filepath.Join(raiz, ficheiroEsquema)
	if _, err := os.Stat(esquema); err == nil {
		return nil
	}
	return os.WriteFile(esquema, []byte(textoEsquema(materia)), 0o644)
}

// textoEsquema é o contido inicial de buxo.md.
func textoEsquema(materia string) string {
	materia = strings.TrimSpace(materia)
	if materia == "" {
		materia = "a materia do proxecto"
	}
	return fmt.Sprintf(`# buxo.md · esquema da wiki

Segundo cerebro sobre **%s**, construído co patrón «LLM Wiki»: a wiki é un
artefacto persistente e acumulativo que escribe e mantén a IA a partir dos
documentos que o usuario achega. O coñecemento compílase unha vez e mantense ao
día; non se redescubre en cada pregunta.

## Capas

- **pdf/**: os PDF orixinais. Nunca se modifican.
- **raw/**: os documentos convertidos a Markdown, un ficheiro por orixinal (o
  frontmatter di de que ficheiro ou URL vén e como se converteu). As fórmulas
  van en LaTeX ($…$) e as figuras en **raw/imaxes/** (![[nome.png]]); os
  circuítos, tamén en bloques circuitikz. **Inmutables**: a IA só os le.
- **wiki/**: páxinas Markdown escritas e mantidas pola IA. É a única capa onde
  a IA escribe.
- **buxo.md**: este esquema. Edítao a man se queres cambiar as convencións.

## Ficheiros especiais (en wiki/)

- **wiki/index.md**: catálogo de TODAS as páxinas, cada unha cun enlace
  [[wikilink]] e un resumo dunha liña, agrupadas por categoría (fontes,
  conceptos, entidades, sínteses). Léese primeiro para atopar o relevante.
- **wiki/log.md**: rexistro cronolóxico só de engadir. Cada entrada comeza por
  «## [AAAA-MM-DD] ingest | título» (ou query / lint).

## Convencións das páxinas

- Nome de ficheiro en minúsculas, sen espazos (guións): wiki/conceptos/fotosintese.md.
- Cartafoles: wiki/fontes/ (un resumo por fonte de raw/), wiki/conceptos/,
  wiki/entidades/ (persoas, institucións, obras, lugares), wiki/sinteses/
  (comparativas, visións de conxunto, respostas valiosas gardadas).
- Frontmatter YAML en cada páxina: title, tags, sources (lista de ficheiros de
  raw/ que a sustentan) e updated (AAAA-MM-DD).
- Enlaces [[wikilink]] abundantes entre páxinas; cada páxina de fonte enlaza os
  conceptos e entidades que menciona, e viceversa. Cita sempre a fonte
  ([[fontes/nome]] ou o ficheiro de raw/).
- Se unha fonte contradí outra páxina, non a borres: anota a contradición nas
  dúas páxinas e di cal é máis recente ou fiable.
- Non inventes: se algo non está nas fontes, non o escribas.

## Fórmulas, figuras e probas

- Copia as fórmulas TAL CAL en LaTeX ($…$, $$…$$): nunca as pases a texto.
- As figuras de raw/ enlázanse co mesmo ![[nome.png]] (están en raw/imaxes/); os
  bloques circuitikz cópianse enteiros.
- Cada fonte que sexa un **exame, boletín ou solucionario** ten a súa páxina en
  **wiki/probas/** (wiki/probas/nome-da-fonte.md) para consultar as preguntas:
  - frontmatter: title, tipo (exame, boletin, solucionario, practica), materia,
    nivel, convocatoria ou data se se sabe, tags, sources, updated;
  - unha sección «### Pregunta N» por pregunta, co enunciado COMPLETO e literal
    (fórmulas, figuras, apartados e puntuación), unha liña «**Temas:**» con
    [[wikilinks]] aos conceptos e, se a fonte a trae, «**Solución:**» literal;
  - cada concepto ten unha sección «## Exercicios» con enlaces ás preguntas que o
    traballan ([[probas/nome#Pregunta 3]]).
- wiki/index.md ten unha categoría «Probas» con todas as páxinas de wiki/probas/.

## Fluxos

### Ingest (procesar unha fonte nova de raw/)
1. Le a fonte (por partes se é longa).
2. Escribe o seu resumo en wiki/fontes/.
3. Crea ou actualiza as páxinas de conceptos e entidades que toca (unha fonte
   pode afectar a 10-15 páxinas).
4. Actualiza wiki/index.md e engade unha entrada a wiki/log.md.

### Query
Busca no índice e nas páxinas, le o relevante e responde citando as páxinas. Se
a resposta ten valor, gárdaa como páxina nova en wiki/sinteses/.

### Lint
Revisa: contradicións, afirmacións obsoletas, páxinas orfas (sen enlaces
entrantes), conceptos mencionados sen páxina propia, enlaces rotos, lagoas.
Propón fontes ou preguntas novas.
`, materia)
}
