#!/usr/bin/env bash
# Empaqueta o módulo: buxo_<versión>_<so>_<arq>.zip (+ .sha256 en formato sha256sum), o
# asset que Lua descarga das Releases de piztusistemas/lua. Leva o executable,
# o conversor (markitdown-sidecar), modulo.json e as licenzas (LICENSE e
# THIRD_PARTY_LICENSES.txt: o aviso de MarkItDown ten que ir con el), todo nun
# único cartafol buxo/ (en macOS, con ditto) ou na raíz do zip.
#
# Antes: `task build EXTRA_TAGS=gtk3` e `sidecar/build.sh` (deixan os dous
# binarios en build/bin). Compila cada plataforma na súa máquina.
#
# macOS: usa ./build-macos.sh, que deixa buxo universal (arm64+x86_64) e os
# dous sidecars nativos (markitdown-sidecar-arm64 / -amd64). O zip é o mesmo
# para as dúas CPU e publícase cos dous nomes (darwin_arm64 e darwin_amd64),
# que son os que busca Lua.
set -euo pipefail
cd "$(dirname "$0")"
VERSION="$(tr -d '[:space:]' < VERSION)"
BIN=build/bin
EXE=buxo; SCS=(markitdown-sidecar)
# Plataformas desta máquina (linux_amd64, windows_amd64...): Lua descarga o zip da súa.
PLATS=("$(go env GOOS)_$(go env GOARCH)")
if [[ "${OS:-}" == "Windows_NT" ]]; then EXE=buxo.exe; SCS=(markitdown-sidecar.exe); fi
if [[ "$(go env GOHOSTOS)" == "darwin" ]]; then
  SCS=(markitdown-sidecar-arm64 markitdown-sidecar-amd64)
  PLATS=(darwin_arm64 darwin_amd64)
  ARCHS="$(lipo -archs "$BIN/$EXE" 2>/dev/null || true)"
  [[ "$ARCHS" == *arm64* && "$ARCHS" == *x86_64* ]] || {
    echo "$BIN/$EXE non é universal ($ARCHS): usa ./build-macos.sh" >&2; exit 1; }
fi
for f in "$EXE" "${SCS[@]}"; do [[ -f "$BIN/$f" ]] || { echo "falta $BIN/$f" >&2; exit 1; }; done
OUT="dist"; mkdir -p "$OUT"
TMP="$(mktemp -d)"; trap 'rm -rf "$TMP"' EXIT
STAGE="$TMP/buxo"; mkdir "$STAGE"
for f in "$EXE" "${SCS[@]}"; do cp "$BIN/$f" "$STAGE/"; chmod 755 "$STAGE/$f"; done
cp LICENSE THIRD_PARTY_LICENSES.txt "$STAGE/"
sed "s/\"version\": \"[^\"]*\"/\"version\": \"$VERSION\"/" modulo.json > "$STAGE/modulo.json"
for PLAT in "${PLATS[@]}"; do
  NOME="buxo_${VERSION}_${PLAT}.zip"
  rm -f "$OUT/$NOME"
  if [[ "$(go env GOHOSTOS)" == "darwin" ]]; then
    # ditto conserva os permisos de execución; --keepParent mete todo en buxo/
    # (Lua quita ese cartafol raíz único ao extraer). Sen forks de recursos nin
    # atributos estendidos, para que non aparezan ficheiros ._* no módulo.
    ditto -c -k --keepParent --norsrc --noextattr --noqtn "$STAGE" "$OUT/$NOME"
  else
    (cd "$STAGE" && zip -q -r "$OLDPWD/$OUT/$NOME" .)
  fi
  (cd "$OUT" && sha256sum "$NOME" > "$NOME.sha256")
  echo "$OUT/$NOME"; cat "$OUT/$NOME.sha256"
done
