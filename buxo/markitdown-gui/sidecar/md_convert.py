"""Sidecar that wraps Microsoft MarkItDown. Embedded in the Go executable.

Usage: md_convert <output_dir> <file> [<file> ...]
Emits one JSON object per line on stdout: {"input","output","ok","error"}.
"""
import json
import os
import sys

from markitdown import MarkItDown


def emit(**kw):
    sys.stdout.write(json.dumps(kw, ensure_ascii=False) + "\n")
    sys.stdout.flush()


def unique_path(path):
    base, ext = os.path.splitext(path)
    n = 1
    while os.path.exists(path):
        path = f"{base}_{n}{ext}"
        n += 1
    return path


def main():
    if len(sys.argv) < 3:
        emit(ok=False, error="uso: md_convert <directorio_saída> <ficheiro>...")
        return 2
    out_dir = sys.argv[1]
    os.makedirs(out_dir, exist_ok=True)
    md = MarkItDown(enable_plugins=False)
    for src in sys.argv[2:]:
        try:
            text = md.convert(src).text_content
            name = os.path.splitext(os.path.basename(src))[0] + ".md"
            dst = unique_path(os.path.join(out_dir, name))
            with open(dst, "w", encoding="utf-8") as f:
                f.write(text)
            emit(input=src, output=dst, ok=True)
        except Exception as e:  # noqa: BLE001
            emit(input=src, ok=False, error=f"{type(e).__name__}: {e}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
