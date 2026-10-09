module edicion

go 1.25.0

// Dependencias mínimas, a propósito: o resto de capacidades (Wails, Maxima,
// LibreOffice...) entran polas aplicacións ou por subpaquetes opcionais.
//
//   - golang.org/x/sys: só o necesita plataforma/pathrefresh_windows.go (le o
//     PATH do Rexistro).
//   - golang.org/x/net: só o necesita plantilla/ia.go (parser de HTML para
//     quedar co texto visible dunha URL). As dúas aplicacións xa o traen.
//   - gopkg.in/yaml.v3: só o necesita ficha/ (o front matter das fichas de
//     LuaYang).
require golang.org/x/sys v0.47.0

require golang.org/x/net v0.58.0

require gopkg.in/yaml.v3 v3.0.1
