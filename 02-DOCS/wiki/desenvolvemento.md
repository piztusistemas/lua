---
type: article
title: Desenvolvemento de Lua
description: Como preparar o equipo, compilar, probar e publicar Lua, e as trampas xa coñecidas.
tags: [desenvolvemento, build, tests, release]
timestamp: 2026-10-10T00:00:00Z
topic: desenvolvemento
---

# Desenvolvemento de Lua

## Preparar o equipo

| Ferramenta | Para que | Instalación (macOS) |
|------------|----------|---------------------|
| Go 1.25+ | Todo o backend | `brew install go` |
| Node 20+ e npm | Frontend (Vite, vitest) | `brew install node` |
| Task | Tarefas de Wails v3 | `brew install go-task` |
| `wails3` | Lua, Buxo e o instalador | `go install github.com/wailsapp/wails/v3/cmd/wails3@latest` |
| Python 3.12 universal2 (python.org) | Sidecar de Buxo | instalador de python.org |
| MacTeX / TeX Live | PDF e tests de LaTeX | `brew install --cask mactex` |
| LibreOffice, Poppler, Maxima | Tests de integración | `brew install --cask libreoffice; brew install poppler maxima` |

En Debian/Ubuntu, `make deps` en `lua/lua_installer` instala o mesmo con `apt`.
En macOS hai `make deps-darwin`.

`markitdown/` vai dentro do repositorio, así que Buxo compila sen clonar nada máis.

## Executar en modo desenvolvemento

```bash
cd lua/lua && task dev      # wails3 dev con Vite e recarga en quente (porto 9245)
```

## Probar

A constitución (regra 2) pide estas ordes limpas en cada módulo que se toque:

```bash
gofmt -l . && go vet ./... && go test ./...   # en edicion, lua/lua e buxo
cd lua/lua/frontend && npm test               # vitest
```

- `go test` en `lua/lua` tarda uns 2 minutos, porque compila LaTeX e abre
  LibreOffice de verdade.
- Os tests que precisan `xelatex`, `soffice`, `pdftotext` ou Maxima fan
  `t.Skip` se a ferramenta falta. Un «ok» nun equipo sen elas proba menos.
- `i18n_test.go` falla se o frontend usa unha clave que non está en
  `idiomas/gl.json`, ou se `es`, `en` e `pt` non teñen as mesmas claves e
  variables ca o galego. Cada texto novo leva a súa clave nos catro JSON.
- O frontend compartido ten un test que falla se a copia de
  `frontend/src/comun/` non coincide con `edicion/web/estatico/`. Arránxase con
  `cd edicion && go run ./cmd/sincronizarweb <destinos>`, nunca editando a copia.

## Compilar

```bash
cd lua/lua && task build                      # build/bin/ da plataforma actual
cd lua/lua && task darwin:package:universal   # lua.app universal (arm64 + amd64)
cd buxo && sidecar/build.sh && task build EXTRA_TAGS=gtk3 && ./package.sh
cd lua/lua_installer && make build            # Lua + instalador (en macOS, universais)
```

En Linux, `EXTRA_TAGS=gtk3` forza GTK3/WebKit2GTK, porque o GTK4 de Wails v3
beta non compila en Debian 12.

## Publicar unha Release

1. Cambia a versión de Lua nos seis sitios (constitución, regra 4.2):
   `VERSION`, `wails.json`, `modulo.json`, `build/config.yml`,
   `build/darwin/Info.plist` e `build/darwin/Info.dev.plist`. Buxo leva a mesma versión.
2. Lua para macOS:
   ```bash
   cd lua/lua_installer
   make build                    # compila Lua universal e o instalador
   make zip-actualizacion-macos  # dist/lua_<v>_darwin_{arm64,amd64}.zip + .sha256
   ```
3. Buxo: `cd buxo && ./package.sh` → `dist/buxo_<v>_<so>_<arq>.zip` + `.sha256`.
4. Crea a Release coa etiqueta `<v>` en `piztusistemas/lua` e sube todos os zip,
   os seus `.sha256` e os instaladores (`lua-installer`, `lua-installer.exe`,
   `lua-installer-macos_<v>.zip`). A Release 26.10.2 é o modelo.
5. Márcaa como *Latest*: o actualizador só mira a última Release.

## Subir cambios

- Un só repositorio (`piztusistemas/lua`), directo a `main`.
- Commits en galego, en imperativo, con gitmoji. Autoría só humana, sen `Co-Authored-By` da IA.
- Credenciais: `gh auth setup-git` fai que `git push` use a sesión de `gh`.

## Trampas xa coñecidas

| Síntoma | Causa | Arranxo (2026-10-10) |
|---------|-------|----------------------|
| `localStorage.clear is not a function` en vitest | Node 25 trae un `localStorage` propio que tapa o de jsdom | `lua/lua/frontend/vitest.config.js` pasa `--no-experimental-webstorage` |
| O informe sae con pdflatex en vez de xelatex en macOS | fontspec non atopa «TeX Gyre Heros» polo nome | Cargar polo ficheiro: `setmainfontTeXGyre` en `latexdoc.go` e `edicion/documento` |
| LibreOffice: «source file could not be loaded» cun ODT de Lua | `mimetype` con descritor de datos (Go `CreateHeader`) e datas 1980-00-00 | `zip.CreateRaw` e `Modified` en `informe_odt.go` |
| `TestDirsModulos` falla en macOS | `XDG_CONFIG_HOME` só manda en Linux | O test compara con `os.UserConfigDir()` |
| Un adaptador común non compila noutra plataforma | Símbolo só definido nunha etiqueta de build | Dar tamén o caso baleiro nas outras (`edicion/doc.go`) |
| O instalador de Windows deixa de pedir administrador | `wails3 task common:update:build-assets` reescribe `build/windows/wails.exe.manifest` con `asInvoker` | Volver poñer `requireAdministrator` (o ficheiro avisa) |
| O instalador instala un Lua vello | Compilouse con `task` directamente e `files/` quedou dunha compilación anterior | Usar sempre `make build`; `version_test.go` falla se a versión embebida non é a de `lua/lua/VERSION` |
