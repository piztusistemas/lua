"""
Plugin registration for markitdown-math.
"""

from typing import Any
from markitdown import MarkItDown

from ._latex_service import LLMVisionLatexService
from ._pdf_converter_with_math import DEFAULT_DPI, PdfConverterWithMath

__plugin_interface_version__ = 1

# Runs before the built-in converters (priority 0.0) and before other PDF
# plugins such as markitdown-ocr (priority -1.0).
PRIORITY_MATH_PDF = -2.0


def register_converters(markitdown: MarkItDown, **kwargs: Any) -> None:
    """
    Register the math-aware PDF converter with MarkItDown.

    Args:
        markitdown: MarkItDown instance to register converters with
        **kwargs: Keyword arguments given to MarkItDown(), which may include:
            - llm_client: OpenAI-compatible client (required for LaTeX output)
            - llm_model: Vision-capable model name (e.g., 'gpt-4o')
            - pdf_math_prompt: Custom transcription prompt
            - pdf_math_mode: 'auto' (default), 'all' or 'off'
            - pdf_math_dpi: Resolution used to render pages (default 200)
    """
    llm_client = kwargs.get("llm_client")
    llm_model = kwargs.get("llm_model")

    latex_service: LLMVisionLatexService | None = None
    if llm_client and llm_model:
        latex_service = LLMVisionLatexService(
            client=llm_client,
            model=llm_model,
            prompt=kwargs.get("pdf_math_prompt"),
        )

    markitdown.register_converter(
        PdfConverterWithMath(
            latex_service=latex_service,
            mode=kwargs.get("pdf_math_mode") or "auto",
            dpi=kwargs.get("pdf_math_dpi") or DEFAULT_DPI,
        ),
        priority=PRIORITY_MATH_PDF,
    )
