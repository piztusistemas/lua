.PHONY: build clean deps deps-darwin check-go check-wails check-wails3 files files-darwin copia-macos zip-actualizacion-macos help

BINARY = lua-installer
VERSION := $(shell tr -d '[:space:]' < VERSION)

# Sistema no que se está a compilar. O instalador non se pode
# compilar de forma cruzada (Wails precisa a webview nativa de cada
# plataforma), así que o SO anfitrión decide tamén o SO de destino: en macOS
# hai que empaquetar Lua coma un bundle .app e non coma un binario solto,
# e iso cambia o contido de files/ (ver files-darwin).
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
FILES_TARGET = files-darwin
else
FILES_TARGET = files
endif

## build: Compila o instalador (require Go, wails e node/npm)
## Fronte da propia interface do instalador (frontend/) — non confundir con
## "files"/"files-darwin", que empaquetan o Lua que se instala. Mesmo esquema
## ca installer/Makefile.
build: check-go check-wails assets/logo.png build/appicon.png $(FILES_TARGET)
	go mod tidy
ifeq ($(UNAME_S),Darwin)
	wails build -platform darwin/universal
	@lipo -info build/bin/$(BINARY).app/Contents/MacOS/$(BINARY)
	@$(MAKE) --no-print-directory copia-macos
else
	wails build
endif
	@echo ""
ifeq ($(UNAME_S),Darwin)
	@echo "✅ Compilado: ./lua_installer/build/bin/$(BINARY).app"
	@echo "   Ábreo cun dobre clic, ou:  open ./lua_installer/build/bin/$(BINARY).app"
	@echo "   NON o executes con sudo: en macOS instálase en /Applications sen"
	@echo "   elevar, e Homebrew négase a funcionar coma root."
else
	@echo "✅ Compilado: ./lua_installer/build/bin/$(BINARY)"
	@echo "   Executa con:  sudo ./lua_installer/build/bin/$(BINARY)"
endif

## copia-macos: Refresca lua-installer-macos.app dende o recén compilado.
## Ese nome é o que se distribúe (di explicitamente para que SO é), pero
## "wails build" só produce lua-installer.app: ata agora a copia facíase á
## man e quedaba conxelada. Iso xa causou un fallo real — instalouse un Lua
## de dous días antes, SEN as plantillas de fábrica de plantillas_fabrica.go,
## e a biblioteca de plantillas saía baleira. Ao ir dentro de "build" xa non
## pode desfasarse.
##
## "cp -R" (non "cp"): Contents/_CodeSignature ten que viaxar tamén, senón a
## copia queda sen firmar e Gatekeeper bloquéaa (mesmo motivo ca en
## files-darwin). Verificado: a firma adhoc sobrevive ao cp -R.
copia-macos:
	@rm -rf build/bin/$(BINARY)-macos.app
	@cp -R build/bin/$(BINARY).app build/bin/$(BINARY)-macos.app
	@echo "✅ Copia de distribución ao día: build/bin/$(BINARY)-macos.app"
	@mkdir -p dist
	@rm -f dist/$(BINARY)-macos_$(VERSION).zip
	@# ditto (non zip): conserva permisos, firma e estrutura do .app.
	@ditto -c -k --keepParent --norsrc --noextattr --noqtn build/bin/$(BINARY)-macos.app dist/$(BINARY)-macos_$(VERSION).zip
	@cd dist && shasum -a 256 $(BINARY)-macos_$(VERSION).zip > $(BINARY)-macos_$(VERSION).zip.sha256
	@echo "✅ Zip: dist/$(BINARY)-macos_$(VERSION).zip"
	@cat dist/$(BINARY)-macos_$(VERSION).zip.sha256
	@$(MAKE) --no-print-directory zip-actualizacion-macos

## zip-actualizacion-macos: o asset co que se actualiza un Lua xa instalado
## (botón "i"): lua_<versión>_darwin_{arm64,amd64}.zip co executable "lua"
## (universal, o mesmo zip cos dous nomes). Sen el, Lua non ve versións novas:
## busca assets "lua_*" e o instalador chámase "lua-installer-*".
zip-actualizacion-macos:
	@test -f files/Contents/MacOS/lua || (echo "❌ falta files/Contents/MacOS/lua: executa make build" && exit 1)
	@rm -rf dist/.lua-exe && mkdir -p dist/.lua-exe
	@cp files/Contents/MacOS/lua dist/.lua-exe/lua && chmod 755 dist/.lua-exe/lua
	@for ARQ in arm64 amd64; do \
		NOME=lua_$(VERSION)_darwin_$$ARQ.zip; \
		rm -f dist/$$NOME; \
		ditto -c -k --norsrc --noextattr --noqtn dist/.lua-exe dist/$$NOME; \
		(cd dist && shasum -a 256 $$NOME > $$NOME.sha256 && cat $$NOME.sha256); \
	done
	@rm -rf dist/.lua-exe

## assets/logo.png: Copia o logo dende Lua
## Depende de ../lua/logo.png (non abondaba con "assets/logo.png:"
## sen prerequisito: unha vez existise o ficheiro, Make xa non volvía copialo
## aínda que ../lua/logo.png cambiase despois).
assets/logo.png: ../lua/logo.png
	@mkdir -p assets
	@cp ../lua/logo.png assets/logo.png

## build/appicon.png: icona da xanela/taskbar de Wails e logo do frontend
## (mesma orixe ca assets/logo.png, ver nota de enriba).
build/appicon.png: ../lua/logo.png
	@mkdir -p build frontend/src/assets/images
	@cp ../lua/logo.png build/appicon.png
	@cp ../lua/logo.png frontend/src/assets/images/logo.png

## files: Compila Lua e xunta en files/ só o necesario en tempo de execución.
## Todo o demais (help/, codeini_default.ini, icona…) xa vai embebido no
## propio binario "lua" (ver lua/main.go e lua/app.go,
## go:embed). Fonte: ../lua (versión con editor Blockly; ../lua
## quedou como versión anterior, xa non se empaqueta aquí).
## Lua Blockly volveu a Wails v3 (ver historial deste ficheiro - pasara por
## v2 un tempo) - compílase con "task build" (CLI wails3/Taskfile, ver
## lua/Taskfile.yml), non "wails build" (v2). EXTRA_TAGS=gtk3
## fai falla en Linux: é o tag que evita GTK4 (o predeterminado de Wails v3
## beta, que NON compila en Debian 12) forzando o backend GTK3/WebKit2GTK -
## SEN VERIFICAR NUNCA en Debian 12 real dende que se escribiu isto, proba
## primeiro con "task build EXTRA_TAGS=gtk3" a man se falla aquí.
files: check-go check-wails3
	@rm -rf files
	@mkdir -p files
	@cd ../lua && task build EXTRA_TAGS=gtk3
	@cp ../lua/build/bin/lua files/lua
	@cp ../lua/logo.png files/logo.png
	@cp ../lua/modulo.json files/modulo.json
	@chmod +x files/lua
	@echo "✅ lua compilado e ficheiros copiados a lua_installer/files/"

## files-darwin: Igual ca "files" pero para macOS, onde Lua non é un binario
## solto senón un bundle Lua.app. Empaquétase con "task package" (non "task
## build": esa só produce o executable) e cópiase o CONTIDO do bundle —
## Contents/ enteiro — dentro de files/.
##
## Por que o contido e non o Lua.app completo: destDir en macOS xa é
## /Applications/Lua.app (ver install.go), e extractFiles volca files/
## directamente aí dentro. Se en files/ houbese un "Lua.app/", quedaría un
## /Applications/Lua.app/Lua.app. E facer que destDir fose /Applications
## sería moito peor: a desinstalación fai os.RemoveAll(destDir).
##
## "cp -R" e non "cp": Contents/_CodeSignature ten que viaxar tamén (ver a
## nota do //go:embed all:files en install.go).
##
## Universal (arm64 + x86_64, "task darwin:package:universal"), coma o propio
## instalador ("wails build -platform darwin/universal"): un só .app que corre
## nativo en Apple Silicon e en Intel. Un Lua só x86_64 corría por Rosetta nos
## Mac Apple Silicon, e macOS avisa de que deixará de admitir esas apps.
##
## EXTRA_TAGS=gtk3 NON se pasa aquí: é un arranxo exclusivo de Linux (evitar
## GTK4); en macOS o backend é WKWebView e ese tag non pinta nada.
files-darwin: check-go check-wails3
	@rm -rf files
	@mkdir -p files
	@rm -rf ../lua/build/bin/lua.app
	@cd ../lua && task darwin:package:universal
	@cp -R ../lua/build/bin/lua.app/Contents files/Contents
	@chmod +x files/Contents/MacOS/lua
	@lipo -info files/Contents/MacOS/lua
	@echo "✅ Lua.app empaquetado e copiado a lua_installer/files/"

## deps-darwin: Dependencias para COMPILAR este instalador nun Mac.
## Non fai falla nada equivalente a libgtk-3-dev: en macOS a webview
## (WKWebView) forma parte do sistema. Si fan falla as Command Line Tools de
## Xcode, que son as que traen o compilador de C que precisa cgo e o
## "codesign" co que se asinan os bundles.
deps-darwin:
	@xcode-select -p > /dev/null 2>&1 || xcode-select --install
	@which brew > /dev/null 2>&1 || (echo "❌ Instala Homebrew primeiro: https://brew.sh" && exit 1)
	brew install go node
	go install github.com/wailsapp/wails/v2/cmd/wails@latest
	go install github.com/wailsapp/wails/v3/cmd/wails3@latest
	brew install go-task
	@echo ""
	@echo "✅ Dependencias instaladas. Engade $$(go env GOPATH)/bin ao PATH se fai falla."
	@echo "   Agora executa: make build"

## deps: Instala as dependencias de sistema necesarias para COMPILAR este
## instalador (non confundir coas dependencias que o instalador INSTALA no
## equipo do profesor — LaTeX, ver install.go). Só Debian/Ubuntu:
## nun Mac usa "make deps-darwin".
deps:
	sudo apt-get update
	sudo apt-get install -y \
		golang-go \
		nodejs \
		npm \
		gcc \
		pkg-config \
		libgtk-3-dev \
		libwebkit2gtk-4.0-dev
	go install github.com/wailsapp/wails/v2/cmd/wails@latest
	go install github.com/wailsapp/wails/v3/cmd/wails3@latest
	sudo snap install task --classic || sudo apt-get install -y task
	@echo ""
	@echo "✅ Dependencias instaladas. Agora executa: make build"

## clean: Elimina o binario compilado e ficheiros temporais
clean:
	rm -rf build/bin
	rm -f assets/logo.png build/appicon.png frontend/src/assets/images/logo.png
	rm -rf frontend/dist frontend/wailsjs
	rm -rf files

check-go:
	@which go > /dev/null 2>&1 || (echo "❌ Go non está instalado. Executa: make deps" && exit 1)

check-wails:
	@which wails > /dev/null 2>&1 || (echo "❌ wails non está instalado. Executa: go install github.com/wailsapp/wails/v2/cmd/wails@latest" && exit 1)

## check-wails3: para "files" (compila lua, que está en Wails v3).
check-wails3:
	@which wails3 > /dev/null 2>&1 || (echo "❌ wails3 non está instalado. Executa: go install github.com/wailsapp/wails/v3/cmd/wails3@latest" && exit 1)
	@which task > /dev/null 2>&1 || (echo "❌ task non está instalado (require wails3 build). Ver https://taskfile.dev/installation/" && exit 1)

help:
	@echo ""
	@echo "Instalador de Lua — Comandos dispoñibles:"
	@echo ""
	@grep -E '^## ' Makefile | sed 's/## /  /'
	@echo ""
