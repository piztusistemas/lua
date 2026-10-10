# Entrada de blog — módulo Lúa

Este cartafol é o paquete completo da entrada de blog sobre o **módulo Lúa**:
o texto, os dous documentos de mostra e as dúas plantillas (estilos) de mostra,
ademais das imaxes listas para publicar.

## Contido

```
blog/
├── blog-modulo-lua.md          # A entrada (Markdown con frontmatter)
├── README.md                   # Este ficheiro
├── descargas/
│   ├── cronograma-2026-2027.pdf   # Documento de mostra: cronograma (8 páx.)
│   ├── actas-2026-2027.pdf        # Documento de mostra: actas (22 páx.)
│   ├── cronograma IES.json        # Plantilla (estilo) do cronograma
│   └── actas IES.json             # Plantilla (estilo) das actas
└── imaxes/
    ├── logo.png                   # Logotipo de Lúa (imaxe de cabeceira)
    ├── lua-cronograma-portada.png # Portada do cronograma (páx. 1)
    ├── lua-cronograma-anual.png   # Cronograma por trimestres (páx. 4)
    ├── lua-actas-portada.png      # Portada das actas (páx. 1)
    └── lua-actas-seguimento.png   # Acta co seguimento (páx. 3)
```

As catro imaxes `lua-*.png` están renderizadas a partir dos PDF de mostra
(100 ppp), para inserir na entrada sen depender de capturas de pantalla.

## Que son os `.json`

**Non son cursos**: son **plantillas** (estilos) no formato de intercambio de
Lúa (`"formato": "lua-plantilla"`). Importaranse desde a **biblioteca de
plantillas** do editor de Lúa. A descrición de cada unha:

| Ficheiro | Plantilla | Que fai |
|---|---|---|
| `cronograma IES.json` | `cronograma IES` | Estilo do cronograma; aumenta ~30 % o tamaño dos dous logotipos da cabeceira en PDF, Word e OpenDocument |
| `actas IES.json` | `actas IES` | Mesmo estilo coa portada adaptada a «Actas do Departamento» |

As imaxes (`conselleria_logo.png`, `logo.png`) van dentro do propio `.json` en
base64, así que cada ficheiro é autocontido.

## Publicación

A entrada segue o mesmo formato ca `blog-instalar-piztu.md` (frontmatter con
`title`, `slug` e `categoría`). Na cabeceira hai unha ligazón á imaxe
`imaxes/logo.png`; o corpo liga as imaxes `imaxes/lua-*.png` con **ruta
relativa**.

Para publicala en WordPress (piztu.org):

1. **Sube as descargas** á biblioteca de medios e, se queres que non dependan de
   GitHub, substitúe as ligazóns `https://github.com/piztusistemas/lua/raw/main/...`
   polas URLs resultantes.
2. **Sube as imaxes** de `imaxes/` á biblioteca de medios e reescribe as rutas
   relativas. Pódese facer dunha soa vez, por exemplo:

   ```bash
   # Axusta a BASE á carpeta de uploads do mes correspondente
   BASE="https://piztu.org/wp-content/uploads/2026/09"
   sed -i -E "s#\]\(imaxes/([^)]+)\)#](${BASE}/\1)#g" blog-modulo-lua.md
   ```

3. Revisa as ligazóns internas do repositorio (release e descargas) e publica.

## Ligazóns públicas xa dispoñibles

Os dous PDF e os dous `.json` están publicados no repositorio do proxecto, polo
que a entrada funciona tal cal sen subir nada:

- Repositorio: <https://github.com/piztusistemas/lua>
- Release 26.9.6: <https://github.com/piztusistemas/lua/releases/tag/26.9.6>
- Cronograma: <https://github.com/piztusistemas/lua/raw/main/cronograma-2026-2027.pdf>
- Actas: <https://github.com/piztusistemas/lua/raw/main/actas-2026-2027.pdf>
- Plantilla do cronograma: <https://github.com/piztusistemas/lua/raw/main/cronograma%20IES.json>
- Plantilla das actas: <https://github.com/piztusistemas/lua/raw/main/actas%20IES.json>
