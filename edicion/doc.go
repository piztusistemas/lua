// Módulo común de edición e xeración de documentación de Piztu.
//
// Viven aquí as pezas que Yang e Lua comparten (e que compartirán os módulos
// futuros): utilidades de plataforma, plantillas, estilo e textos do
// documento, motor LaTeX, escritores ofimáticos nativos, recursos e mailo
// núcleo do asistente.
//
// Regras de deseño (ver PLAN-modulo-edicion.md):
//
//   - Este módulo NON importa Wails nin coñece `*App`, `Settings` nin rutas de
//     configuración do usuario. Todo o que necesita do exterior entra por
//     parámetro ou por interface.
//   - O dominio (exames, cronograma, actas) queda nas aplicacións.
//   - Maxima vai nun subpaquete OPCIONAL (`cas/`), non no núcleo: un módulo de
//     documentación non debe arrastrar un CAS que non usa.
//   - Todo símbolo que as aplicacións usen desde un adaptador común a todas as
//     plataformas ten que existir en TODAS as plataformas. Se unha capacidade
//     só existe nunha (p. ex. localizar o bundle .app en macOS), hai que dar
//     tamén un caso baleiro nos ficheiros da(s) outra(s) etiqueta(s): senón o
//     adaptador non compila fóra desa plataforma (aprendido con
//     BundleQueContenA).
//   - O frontend compartido vive en web/estatico/ e chega ás aplicacións coma
//     copia xerada (frontend/src/comun/ + MANIFEST.sha256) con
//     `cd edicion && go run ./cmd/sincronizarweb <destinos>`. As copias NON se
//     editan a man: cada aplicación ten un test que falla se derivan.
//   - Os tipos que cruzan a fronteira da aplicación (o estado de
//     actualización, o paquete de LaTeX pendente) expóñense nas app coma
//     ALIAS do tipo do módulo (`type X = paquete.Y`), non coma copias: así os
//     bindings de Wails seguen co mesmo nome e os mesmos campos JSON.
package edicion
