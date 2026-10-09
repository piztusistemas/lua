# Piztu Lua

Aplicación de escritorio para docentes: xera a documentación dos departamentos
(cronograma anual, actas, formularios propios) e problemas de matemáticas e
ciencias con Maxima, compilados en LaTeX, Word e OpenDocument. Lua absorbe o
que facía Yang.

Funciona en macOS, Windows e Linux, e a interface está en galego, castelán,
inglés e portugués.

## Que hai neste repositorio

| Cartafol | Que é |
|----------|-------|
| [`lua/lua`](lua/lua) | A aplicación (Go + Wails v3, frontend en JavaScript sen framework). |
| [`lua/lua_installer`](lua/lua_installer) | Instalador de Lua e das súas dependencias (Go + Wails v3). |
| [`edicion`](edicion) | Motor común: plantillas, LaTeX, Word/ODT, IA, Maxima, actualizacións, bóveda. |
| [`buxo`](buxo) | Módulo que converte PDF e documentos nunha bóveda de Obsidian coa IA. |
| [`markitdown`](markitdown) | Copia de MarkItDown (Microsoft, MIT) coa que Buxo converte documentos. |
| [`02-DOCS`](02-DOCS) | Documentación do proxecto: constitución, arquitectura, desenvolvemento. |
| [`exemplos`](exemplos) | Exemplos de documentos xerados con Lua. |

Que fai Lua, ferramenta a ferramenta: [`lua/README.md`](lua/README.md).
Como encaixan as pezas: [arquitectura](02-DOCS/wiki/arquitectura.md).

## Instalar

Descarga a última versión desde as
[Releases](https://github.com/piztusistemas/lua/releases). Lua comproba alí
as actualizacións e o módulo Buxo.

## Compilar desde o código

Necesitas Go 1.25+, Node 20+, [Task](https://taskfile.dev) e a CLI `wails3`.

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@latest
cd lua/lua
task dev          # modo desenvolvemento
task build        # binario en build/bin/
```

Para xerar PDF fai falla unha distribución LaTeX (TeX Live ou MacTeX). Maxima,
LibreOffice e ImageMagick son opcionais: se faltan, Lua avisa e di que instalar.

## Probar

```bash
(cd edicion && go test ./...)
(cd lua/lua && go test ./...)
(cd lua/lua/frontend && npm install && npm test)
(cd buxo && go test ./...)
```

Para compilar Buxo co seu conversor: `cd buxo && sidecar/build.sh` (precisa Python 3.12).

Detalle de cada orde e dos tests que precisan LaTeX ou LibreOffice:
[desenvolvemento](02-DOCS/wiki/desenvolvemento.md).

## Contribuír

As regras do proxecto (calidade, convencións, seguridade) están na
[constitución](02-DOCS/wiki/sdd/constitution.md). O código e a documentación
escríbense en galego.

## Licenza

Cada módulo ten o seu ficheiro `LICENSE`: [Lua](lua/lua/LICENSE) (AGPL-3.0),
[Buxo](buxo/LICENSE) (MIT).
