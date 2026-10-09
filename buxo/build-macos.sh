#!/usr/bin/env bash
# Compila e empaqueta buxo para macOS, para Apple Silicon e Intel á vez:
#   - buxo universal (arm64 + x86_64, unidos con lipo);
#   - markitdown-sidecar-arm64 e markitdown-sidecar-amd64, nativos (buxo
#     lanza o da CPU na que corre: nada pasa por Rosetta);
#   - dist/buxo_<versión>_darwin_{arm64,amd64}.zip (o mesmo zip cos dous nomes).
# Precisa o Python universal2 de python.org (ver sidecar/build.sh).
set -euo pipefail
cd "$(dirname "$0")"
export PATH="$PATH:$HOME/go/bin"
BIN=build/bin
sidecar/build.sh arm64
sidecar/build.sh amd64
task build ARCH=arm64 OUTPUT="$BIN/buxo-arm64"
task build ARCH=amd64 OUTPUT="$BIN/buxo-amd64"
lipo -create -output "$BIN/buxo" "$BIN/buxo-arm64" "$BIN/buxo-amd64"
rm -f "$BIN/buxo-arm64" "$BIN/buxo-amd64" "$BIN/markitdown-sidecar"
lipo -info "$BIN/buxo"
./package.sh
