"""
PDF converter that transcribes pages containing mathematics into Markdown
with embedded LaTeX, using an LLM vision model.
"""

import io
from typing import Any, BinaryIO, Literal

import pdfminer.high_level
import pdfplumber
import pypdfium2

from markitdown import DocumentConverterResult, StreamInfo
from markitdown.converters import PdfConverter

from ._latex_service import LLMVisionLatexService
from ._math_detection import DEFAULT_MIN_MATH_CHARS, page_has_math

MathMode = Literal["auto", "all", "off"]

DEFAULT_DPI = 200


def render_page_png(pdf: pypdfium2.PdfDocument, page_index: int, dpi: int) -> bytes:
    page = pdf[page_index]
    try:
        image = page.render(scale=dpi / 72).to_pil()
    finally:
        page.close()
    buffer = io.BytesIO()
    image.save(buffer, format="PNG")
    return buffer.getvalue()


class PdfConverterWithMath(PdfConverter):
    """
    Converts PDFs to Markdown, writing mathematics as LaTeX.

    Modes (constructor argument or `pdf_math_mode` keyword at convert time):
    - "auto": only pages detected as mathematical go through the LLM; the
      other pages use regular text extraction.
    - "all": every page goes through the LLM.
    - "off": behave exactly like the built-in PdfConverter.

    Without an LLM service the converter also behaves like the built-in one.
    """

    def __init__(
        self,
        latex_service: LLMVisionLatexService | None = None,
        mode: MathMode = "auto",
        dpi: int = DEFAULT_DPI,
        min_math_chars: int = DEFAULT_MIN_MATH_CHARS,
    ):
        super().__init__()
        self.latex_service = latex_service
        self.mode = mode
        self.dpi = dpi
        self.min_math_chars = min_math_chars

    def convert(
        self,
        file_stream: BinaryIO,
        stream_info: StreamInfo,
        **kwargs: Any,
    ) -> DocumentConverterResult:
        latex_service = kwargs.get("latex_service") or self.latex_service
        mode: MathMode = kwargs.get("pdf_math_mode") or self.mode
        dpi: int = kwargs.get("pdf_math_dpi") or self.dpi

        if mode not in ("auto", "all", "off"):
            raise ValueError(
                f"Invalid pdf_math_mode {mode!r}; expected 'auto', 'all' or 'off'."
            )

        pdf_bytes = file_stream.read()

        if latex_service is None or mode == "off":
            return super().convert(io.BytesIO(pdf_bytes), stream_info, **kwargs)

        with pdfplumber.open(io.BytesIO(pdf_bytes)) as pdf:
            page_count = len(pdf.pages)
            math_pages: set[int] = set()
            for page_idx, page in enumerate(pdf.pages):
                if mode == "all" or page_has_math(page, self.min_math_chars):
                    math_pages.add(page_idx)
                page.close()

        if not math_pages:
            return super().convert(io.BytesIO(pdf_bytes), stream_info, **kwargs)

        chunks: list[str] = []
        pdfium_doc = pypdfium2.PdfDocument(pdf_bytes)
        try:
            for page_idx in range(page_count):
                content = None
                if page_idx in math_pages:
                    png = render_page_png(pdfium_doc, page_idx, dpi)
                    content = latex_service.transcribe(png)
                if content is None:
                    content = pdfminer.high_level.extract_text(
                        io.BytesIO(pdf_bytes), page_numbers=[page_idx]
                    )
                if content.strip():
                    chunks.append(content.strip())
        finally:
            pdfium_doc.close()

        return DocumentConverterResult(markdown="\n\n".join(chunks))
