# SPDX-FileCopyrightText: 2025-present Contributors
# SPDX-License-Identifier: MIT

"""
markitdown-math: math plugin for MarkItDown

Converts PDFs containing mathematics to Markdown with embedded LaTeX using an
LLM vision model.
"""

from ._plugin import __plugin_interface_version__, register_converters
from .__about__ import __version__
from ._latex_service import DEFAULT_PROMPT, LLMVisionLatexService
from ._math_detection import page_has_math
from ._pdf_converter_with_math import PdfConverterWithMath

__all__ = [
    "__version__",
    "__plugin_interface_version__",
    "register_converters",
    "DEFAULT_PROMPT",
    "LLMVisionLatexService",
    "PdfConverterWithMath",
    "page_has_math",
]
