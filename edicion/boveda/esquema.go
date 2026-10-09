package boveda

import (
	"fmt"
	"strings"
)

// TextoEsquema é o contido inicial de buxo.md.
func TextoEsquema(materia string) string {
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
- **raw/**: os PDF convertidos a Markdown, un ficheiro por PDF (a primeira liña
  de frontmatter di de que PDF ou URL vén). **Inmutables**: a IA só os le.
- **wiki/**: páxinas Markdown escritas e mantidas pola IA. É a única capa onde
  a IA escribe.
- **fichas/**: as fichas de LuaYang (exercicios, exames, bancos e formularios).
  A wiki enlázaas, pero non as modifica.
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
  wiki/exercicios/ (unha nota por exercicio das fontes, cos conceptos que
  traballa e unha ligazón [[…]] á súa ficha se existe),
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
