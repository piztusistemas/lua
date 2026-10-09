"""
Heuristics for deciding whether a PDF page contains mathematical notation.

Only pages that look mathematical are sent to the (comparatively slow and
costly) LLM; the rest keep MarkItDown's regular text extraction.
"""

import re
from typing import Any

# Base names (subset prefix such as "ABCDEF+" removed) of fonts that are only
# used for typesetting mathematics: TeX Computer Modern math fonts, AMS symbol
# fonts, Euler, RSFS, and OpenType math fonts (Cambria Math, STIX Two Math,
# Latin Modern Math, ...).
MATH_FONT_PATTERN = re.compile(
    r"^(CMMI|CMMIB|CMSY|CMBSY|CMEX|MSAM|MSBM|EUFM|EUFB|EURM|EURB|EUSM|EUSB|EUEX"
    r"|RSFS|ESINT|STMARY|LMMATH|TXSY|TXEX|TXMI|PXSY|PXEX|PXMI|MTMI|MTSY|MTEX)"
    r"|MATH",
    re.IGNORECASE,
)

# Code points that are almost exclusively used in mathematical notation.
# U+2212 (minus sign) is deliberately excluded because it is common in prose
# and tables.
_MATH_CODEPOINT_RANGES = (
    (0x2200, 0x2211),  # Mathematical Operators (part 1)
    (0x2213, 0x22FF),  # Mathematical Operators (part 2)
    (0x27C0, 0x27EF),  # Miscellaneous Mathematical Symbols-A
    (0x2980, 0x29FF),  # Miscellaneous Mathematical Symbols-B
    (0x2A00, 0x2AFF),  # Supplemental Mathematical Operators
    (0x1D400, 0x1D7FF),  # Mathematical Alphanumeric Symbols
)

# Minimum number of math symbols on a page before it counts as mathematical.
# A single stray symbol (e.g. "∞" in a sentence) should not trigger the LLM.
DEFAULT_MIN_MATH_CHARS = 3


def _base_font_name(fontname: str) -> str:
    # Embedded subset fonts are named like "XQOOLT+CMMI10".
    return fontname.split("+", 1)[-1]


def is_math_font(fontname: str | None) -> bool:
    if not fontname:
        return False
    return MATH_FONT_PATTERN.search(_base_font_name(fontname)) is not None


def is_math_char(text: str | None) -> bool:
    if not text:
        return False
    return any(
        start <= ord(ch) <= end for ch in text for start, end in _MATH_CODEPOINT_RANGES
    )


def page_has_math(page: Any, min_math_chars: int = DEFAULT_MIN_MATH_CHARS) -> bool:
    """
    Return True if a pdfplumber page appears to contain mathematics.

    A character counts as mathematical if it is set in a math font or is a
    mathematical Unicode symbol.
    """
    count = 0
    for char in page.chars:
        if is_math_font(char.get("fontname")) or is_math_char(char.get("text")):
            count += 1
            if count >= min_math_chars:
                return True
    return False
