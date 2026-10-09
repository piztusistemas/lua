#!/usr/bin/env bash
# Builds the embedded markitdown sidecar with the latest markitdown + deps.
set -euo pipefail
cd "$(dirname "$0")"
python3 -m venv .venv
. .venv/bin/activate
pip install -U pip
pip install -U "markitdown[pdf]" pyinstaller
pyinstaller --onefile --clean --noconfirm --name markitdown-sidecar \
  --collect-all magika --collect-all markitdown \
  --distpath ../embedded --workpath .build --specpath .build md_convert.py
pip show markitdown | grep -i '^version'
