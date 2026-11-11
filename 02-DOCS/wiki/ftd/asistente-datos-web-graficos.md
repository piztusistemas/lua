---
author: Martín Pardiñas
branch: main
status: in-progress
---
# Asistente de datos: web, formularios web e gráficos

## Intent
O docente pediulle ao asistente de datos os gráficos das temperaturas de Vimianzo (2000-2026) sacados de datosclima.es, e o asistente respondeu que non tiña acceso á web. Hai tres bloqueos:
1. O ámbito `formulario_datos` non ten `web_buscar` nin `web_consultar`. Quitáronse a propósito, por privacidade.
2. datosclima.es dá os datos só despois de enviar un formulario POST (provincia, despois estación e despois datas; a acción `Tempestad.php` ponse con JavaScript). `web_consultar` só fai GET e non deixa ver os campos dos formularios.
3. O asistente non pode poñer gráficos no formulario.

## Scope
- Dentro:
  - Web no asistente de datos (backend e porta do frontend), con regras no prompt.
  - `web_consultar` con `formularios: true`: lista os formularios da páxina (acción, método, campos, opcións) e as accións que poñen os scripts.
  - Ferramenta nova `web_enviar_formulario` (POST ou GET con campos), coas mesmas proteccións ca `web_consultar`: só enderezos públicos, 4 MB e tempo máximo. É de tipo `escritura`, porque envía datos fóra.
  - Gráficos: `pgfplots` dentro dos bloques ```tikz (preámbulo do PDF, plantillas completas e recortes de Word/ODT/vista) e regras de gráficos para a IA.
- Calquera fonte pública, non só datosclima.es (petición do usuario durante o traballo): celas das táboas separadas, CSV e JSON tal cal, e aviso claro cos ficheiros binarios.
- Fóra: un tipo de campo «gráfico» novo; descargar ficheiros CSV ou ZIP; executar o JavaScript das páxinas.

## Decisións
- A privacidade segue en pé: en modo produción o asistente non existe e as súas ferramentas néganse. A web só funciona en modo deseño, que leva datos ficticios ou públicos.
- Os gráficos son pgfplots dentro de ```tikz e non un tipo novo de bloque. Así reutilizan a compilación, a caché e os avisos que xa teñen os circuítos.
- `web_enviar_formulario` é `escritura` e non `lectura`: con «pedir confirmación de todo» pregunta antes de enviar.

## Checklist
- [x] Go: o ámbito `formulario_datos` ten `web_buscar`, `web_consultar` e `web_enviar_formulario`. Proba: `TestAmbitoFormularioDatosFerramentas` (14 ferramentas) en vermello e despois en verde.
- [x] Go: `formulariosDeHTML`, `webEnviarFormulario` e `formularios=true` nas dúas ferramentas. Proba: tests con servidor local; os enderezos locais rexéitanse.
- [x] Frontend: a porta `LIBRES` deixa pasar a web; en produción segue negada. Proba: vitest 24/24 no ficheiro.
- [x] LaTeX: `preambuloPgfplots` nos tres preámbulos (coma decimal, sen separador de miles). Proba: `TestPgfplotsNosPreambulos`, `TestGraficoPgfplotsCompila` e `TestFormularioConGraficoPgfplotsCompilaPDF` (pdflatex e xelatex); revisei a páxina renderizada.
- [x] Prompt: regras WEB e GRÁFICOS. Proba: test do texto do prompt.
- [x] Proba real contra datosclima.es: provincia → «1399» = VIMIANZO → `Tempestad.php` devolve as temperaturas de 2005.
- [x] Calquera fonte: celas das táboas con « | », CSV tal cal e erro nos binarios. Proba: `TestTextoDeHTMLSeparaAsCelasDasTaboas` (vermello: «200513,435,0») e `TestTextoWebCSVeBinarios`.
- [x] Rexistro de decisións, README, gofmt, vet, go test e npm test.
- [ ] Proba na aplicación cunha IA de verdade (pedido completo de Vimianzo 2000-2026). Non se fixo: precisa a app aberta e a clave da API.
- [ ] Vista previa do editor: non se comprobou que un gráfico nun texto longo se vexa na vista do deseño (no PDF xerado si se ve).

## Evidence
- `go test ./...` en `lua/lua`: ok (158 s). `npm test`: 38 ficheiros, 583 tests en verde. `gofmt -l` baleiro e `go vet` limpo.
- Fluxo real (test temporal, xa borrado): o paso 1 lista Provincia co aviso «reenvía o formulario»; o paso 2 atopa «1399» = VIMIANZO e a acción `https://datosclima.es/Aemethistorico/Tempestad.php`; o paso 3 devolve «Temperatura Máxima más alta Registrada: 35.0 17-06-2005».
- PDF do formulario con gráfico (pdflatex): eixe cos anos 1998-2022, eixe Y 13 / 13,5 / 14, lenda e tendencia.
- Límite da fonte: o histórico de datosclima.es chega ata 2012; desde 2013 está noutra sección (`Aemet2013/`). A IA ten que consultar as dúas.

- Instalada en `/Applications/Lua.app` o 2026-11-12: `task darwin:package:universal`. Cambiouse só `MacOS/lua` e `Info.plist`; o executable vello quedou en `lua.anterior`. Buxo segue dentro e asinado, `codesign --verify --deep` dá ok e a app arranca. A versión segue sendo 26.10.9.

## Next
Probalo na aplicación co pedido de Vimianzo e mirar a vista previa do gráfico no editor.
