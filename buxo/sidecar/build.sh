#!/usr/bin/env bash
# Compila o sidecar markitdown-sidecar co markitdown do cartafol irmán
# (lua_yang/markitdown) e déixao en ../build/bin (ao lado do executable de buxo).
#
# En macOS: `sidecar/build.sh arm64` ou `sidecar/build.sh amd64` deixa
# markitdown-sidecar-<arq>, nativo desa CPU (buxo escolle o seu ao arrincar).
# PyInstaller non pode facer un só sidecar universal2 (numpy non ten wheels
# universal2), así que se compilan os dous, cada un no seu venv, co Python
# universal de python.org (o de Homebrew só é arm64) lanzado con `arch`.
set -euo pipefail
cd "$(dirname "$0")"
MARKITDOWN="${MARKITDOWN:-../../markitdown/packages/markitdown}"
NOME=markitdown-sidecar
VENV=.venv
WORK=.build
RUN=()
PY="${PYTHON:-python3}"
if [[ "$(uname -s)" == "Darwin" ]]; then
  ARQ="${1:-$(go env GOARCH)}"
  case "$ARQ" in
    arm64) MACH=arm64 ;;
    amd64|x86_64) ARQ=amd64; MACH=x86_64 ;;
    *) echo "arquitectura descoñecida: $ARQ (arm64 ou amd64)" >&2; exit 1 ;;
  esac
  PY="${PYTHON:-/Library/Frameworks/Python.framework/Versions/3.12/bin/python3}"
  lipo -archs "$PY" | grep -qw "$MACH" || {
    echo "$PY non ten $MACH: instala o Python universal2 de python.org ou pon PYTHON=" >&2; exit 1; }
  NOME="$NOME-$ARQ"; VENV=".venv-$ARQ"; WORK=".build-$ARQ"
  RUN=(arch "-$MACH")
  if [[ "$MACH" == x86_64 && "$(uname -m)" == arm64 ]]; then
    # Baixo Rosetta, PyInstaller chama a lipo/codesign... e herdan x86_64, pero
    # as Command Line Tools só traen libxcrun arm64: lánzanse en arm64.
    FERR="$(mktemp -d)"; trap 'rm -rf "$FERR"' EXIT
    for t in lipo codesign otool install_name_tool; do
      printf '#!/bin/sh\nexec /usr/bin/arch -arm64 /usr/bin/%s "$@"\n' "$t" > "$FERR/$t"
      chmod +x "$FERR/$t"
    done
    export PATH="$FERR:$PATH"
  fi
fi
"${RUN[@]}" "$PY" -m venv "$VENV"
"${RUN[@]}" "$VENV/bin/python" -m pip install -U pip
# --prefer-binary: cryptography e onnxruntime xa non publican wheels para
# Intel (x86_64) en macOS; así cóllese a última versión que os ten en vez de
# intentar compilala.
"${RUN[@]}" "$VENV/bin/python" -m pip install -U --prefer-binary "${MARKITDOWN}[pdf,docx,pptx,xlsx]" pyinstaller
"${RUN[@]}" "$VENV/bin/python" -m PyInstaller --onefile --clean --noconfirm --name "$NOME" \
  --collect-all magika --collect-all markitdown \
  --distpath ../build/bin --workpath "$WORK" --specpath "$WORK" md_convert.py
