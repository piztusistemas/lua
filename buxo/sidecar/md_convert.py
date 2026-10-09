"""Sidecar de Buxo: envolve Microsoft MarkItDown (código en ../../markitdown).

Uso: md_convert <cartafol_saída> <ficheiro> [<ficheiro> ...]
Escribe un obxecto JSON por liña en stdout: {"input","output","ok","error"}.
Cada ficheiro dá o seu <nome>.md (se xa existe, sobrescríbese: reconvertir un
PDF dá o mesmo resultado). Un ficheiro que falla non para os demais.
"""
import json
import os
import sys

from markitdown import MarkItDown


def emit(**kw):
    sys.stdout.write(json.dumps(kw, ensure_ascii=False) + "\n")
    sys.stdout.flush()


def main():
    if len(sys.argv) < 3:
        emit(ok=False, error="uso: md_convert <cartafol_saída> <ficheiro>...")
        return 2
    out_dir = sys.argv[1]
    os.makedirs(out_dir, exist_ok=True)
    md = MarkItDown(enable_plugins=False)
    for src in sys.argv[2:]:
        try:
            text = md.convert(src).text_content
            if not text or not text.strip():
                raise ValueError("o PDF non ten texto extraíble (¿está escaneado?)")
            name = os.path.splitext(os.path.basename(src))[0] + ".md"
            dst = os.path.join(out_dir, name)
            tmp = dst + ".parte"
            with open(tmp, "w", encoding="utf-8") as f:
                f.write(text)
            os.replace(tmp, dst)
            emit(input=src, output=dst, ok=True)
        except Exception as e:  # noqa: BLE001
            emit(input=src, ok=False, error=f"{type(e).__name__}: {e}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
