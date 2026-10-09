# Buxo

Módulo común de Lua (e, máis adiante, Yang) que constrúe unha **bóveda de
Obsidian** co método de Karpathy («LLM Wiki») a partir de PDF.

```
<bóveda>/
  pdf/      os PDF orixinais (copiados do equipo ou descargados)
  raw/      os PDF convertidos a Markdown (con cabeceira da fonte)
  wiki/     a wiki que constrúe a IA (index.md, log.md, fontes/, conceptos/...)
  buxo.md   o esquema que segue a IA (editable)
  buxo.log  rexistro das operacións
```

## Que se pode subir (coma NotebookLM)

PDF, imaxes (fotos de exames, esquemas), Markdown, Word, PowerPoint, Excel, HTML
e EPUB. Todo acaba en `raw/` coma Markdown:

- **Texto normal** (PDF de texto, ofimática): markitdown, gratis e rápido.
- **Fórmulas, debuxos e escaneados**: a IA **le as páxinas** coma imaxes
  (`vision.go`) e transcríbeas a Markdown con **LaTeX** (`$…$`, `$$…$$`). As
  figuras (gráficas, xeometría, circuítos) recórtanse en `raw/imaxes/` e
  enlázanse con `![[…]]`; os circuítos reprodúcense ademais en ` ```circuitikz `.
  En modo **Automático** Buxo decide só cando fai falla (símbolos matemáticos,
  texto esnaquizado, figuras no PDF ou pouco texto); tamén se pode pedir
  **Sempre coa IA** ou **Só texto**.
- **Markdown**: vai tal cal; se ten fórmulas en texto plano (x^2, sqrt(x)...), a
  IA pásaas a LaTeX sen tocar o resto.

Ao construír a bóveda, cada exame, boletín ou solucionario ten a súa páxina en
`wiki/probas/`, cunha sección por pregunta (enunciado literal con fórmulas e
figuras, temas enlazados e solución se a hai), e os conceptos enlazan os
exercicios que os traballan.

A estrutura non se crea ata que hai un ficheiro. Cada PDF que se engade (ficheiro ou
URL) cópiase/descárgase a `pdf/` e convértese **sen intervención** a `raw/`
cun sidecar de [markitdown](../markitdown); se un falla, anótase no rexistro e
sigue co seguinte. Unha URL dunha páxina (non dun PDF) pásalla á IA, que busca
nela os PDF que valen para o proxecto. «Construír a bóveda» fai que a IA lea
cada fonte de `raw/` e escriba a wiki.

## Como funciona con Lua

Buxo non garda a clave da IA nin toca máis ca a súa bóveda: Lua lánzao
(`buxo --boveda <cartafol> --materia <texto> --id <id>`) con `PIZTU_API_URL` e
`PIZTU_API_TOKEN`, e todo o que precisa (turnos de IA, ferramentas `boveda_*`)
pídeo á API local de Lua (`edicion/api`). Sen esa API, ou sen unha IA
configurada, Buxo avisa e non fai nada: Lua só o ofrece cando hai IA.

## Compilar e empaquetar

```bash
sidecar/build.sh                      # markitdown-sidecar (usa ../markitdown)
PATH=$PATH:$HOME/go/bin task build EXTRA_TAGS=gtk3
./package.sh                          # dist/buxo_<versión>.zip + .sha256
```

Publica os dous ficheiros nunha Release de `piztusistemas/lua`: Lua instala
sozo `buxo_<versión>.zip` (comprobando o SHA-256) no seu cartafol de módulos.

## Licenza

Buxo é software libre baixo licenza **MIT** (ver `LICENSE`), os mesmos termos ca
[MarkItDown](https://github.com/microsoft/markitdown), que leva dentro do
conversor. Os avisos de terceiros (MarkItDown e os paquetes Python do conversor)
están en `THIRD_PARTY_LICENSES.txt`. Ambos van no zip do módulo e pódense
consultar desde Buxo co botón «Licenza».

## Versións e Releases

Buxo viaxa na Release de Lua coa **mesma versión**: cada Release de
`piztusistemas/lua` leva, ademais dos ficheiros de Lua, o módulo
`buxo_<versión>_<so>_<arq>.zip` e o seu `.sha256`. Lua (e Yang) miran só a
**última** Release publicada e, dentro dela, o zip da súa plataforma; se esa
Release non o leva, non se descarga nada.
