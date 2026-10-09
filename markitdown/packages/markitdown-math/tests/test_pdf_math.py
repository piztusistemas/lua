"""
Tests for markitdown-math.

tests/test_files/math.pdf is compiled from math.tex with pdflatex: page 1 is
plain prose, page 2 contains inline and displayed formulas.
"""

from pathlib import Path
from types import SimpleNamespace
from typing import Any

import pdfplumber
import pytest

from markitdown import MarkItDown, StreamInfo
from markitdown_math import PdfConverterWithMath, LLMVisionLatexService
from markitdown_math._latex_service import strip_code_fence
from markitdown_math._math_detection import is_math_char, is_math_font, page_has_math

MATH_PDF = Path(__file__).parent / "test_files" / "math.pdf"

_LATEX = "## Quadratic formula\n\n$$ x = \\frac{-b \\pm \\sqrt{b^2-4ac}}{2a} $$"


class MockCompletions:
    def __init__(self, content: str | None = _LATEX, error: Exception | None = None):
        self.content = content
        self.error = error
        self.calls: list[dict[str, Any]] = []

    def create(self, **kwargs: Any) -> Any:
        self.calls.append(kwargs)
        if self.error is not None:
            raise self.error
        message = SimpleNamespace(content=self.content)
        return SimpleNamespace(choices=[SimpleNamespace(message=message)])


def mock_client(**kwargs: Any) -> Any:
    return SimpleNamespace(chat=SimpleNamespace(completions=MockCompletions(**kwargs)))


def _convert(converter: PdfConverterWithMath, **kwargs: Any) -> str:
    with open(MATH_PDF, "rb") as f:
        return converter.convert(f, StreamInfo(extension=".pdf"), **kwargs).markdown


# Detection


def test_math_fonts() -> None:
    for name in ["XQOOLT+CMMI10", "CMSY7", "CMEX10", "MSBM10", "CambriaMath"]:
        assert is_math_font(name), name
    for name in ["BEGZQY+CMR10", "CMBX12", "Helvetica", "TimesNewRomanPSMT", None]:
        assert not is_math_font(name), name


def test_math_chars() -> None:
    assert is_math_char("∑")
    assert is_math_char("≤")
    assert is_math_char("𝑥")  # MATHEMATICAL ITALIC SMALL X
    assert not is_math_char("−")  # minus sign is common in prose
    assert not is_math_char("a")
    assert not is_math_char("")


def test_page_has_math() -> None:
    with pdfplumber.open(MATH_PDF) as pdf:
        assert not page_has_math(pdf.pages[0])
        assert page_has_math(pdf.pages[1])


# Service


def test_strip_code_fence() -> None:
    assert strip_code_fence("```markdown\n$x$\n```") == "$x$"
    assert strip_code_fence("```\n$x$\n```") == "$x$"
    assert strip_code_fence("$x$ and ```code```") == "$x$ and ```code```"


def test_service_sends_png_and_prompt() -> None:
    client = mock_client()
    service = LLMVisionLatexService(client, "test-model", prompt="PROMPT")
    assert service.transcribe(b"\x89PNG") == _LATEX

    call = client.chat.completions.calls[0]
    assert call["model"] == "test-model"
    content = call["messages"][0]["content"]
    assert content[0] == {"type": "text", "text": "PROMPT"}
    assert content[1]["image_url"]["url"].startswith("data:image/png;base64,")


def test_service_returns_none_on_error() -> None:
    service = LLMVisionLatexService(mock_client(error=RuntimeError("boom")), "m")
    with pytest.warns(UserWarning, match="boom"):
        assert service.transcribe(b"") is None


# Converter


def test_auto_mode_only_sends_math_pages() -> None:
    client = mock_client()
    converter = PdfConverterWithMath(LLMVisionLatexService(client, "m"))
    markdown = _convert(converter)

    assert len(client.chat.completions.calls) == 1
    assert markdown.startswith("Plain page")
    assert "ordinary prose" in markdown
    assert markdown.endswith(_LATEX)


def test_all_mode_sends_every_page() -> None:
    client = mock_client()
    converter = PdfConverterWithMath(LLMVisionLatexService(client, "m"), mode="all")
    markdown = _convert(converter)

    assert len(client.chat.completions.calls) == 2
    assert markdown == f"{_LATEX}\n\n{_LATEX}"


def test_mode_override_at_convert_time() -> None:
    client = mock_client()
    converter = PdfConverterWithMath(LLMVisionLatexService(client, "m"))
    markdown = _convert(converter, pdf_math_mode="off")

    assert client.chat.completions.calls == []
    assert markdown == MarkItDown().convert(MATH_PDF).markdown


def test_invalid_mode() -> None:
    converter = PdfConverterWithMath(LLMVisionLatexService(mock_client(), "m"))
    with pytest.raises(ValueError, match="pdf_math_mode"):
        _convert(converter, pdf_math_mode="sometimes")


def test_without_service_matches_builtin() -> None:
    markitdown = MarkItDown()
    expected = markitdown.convert(MATH_PDF).markdown
    assert _convert(PdfConverterWithMath()) == expected


def test_llm_failure_falls_back_to_text() -> None:
    client = mock_client(error=RuntimeError("down"))
    converter = PdfConverterWithMath(LLMVisionLatexService(client, "m"))
    with pytest.warns(UserWarning):
        markdown = _convert(converter)
    assert "Quadratic" in markdown
    assert "ordinary prose" in markdown


def test_plugin_registration() -> None:
    client = mock_client()
    markitdown = MarkItDown(enable_plugins=True, llm_client=client, llm_model="m")
    markdown = markitdown.convert(MATH_PDF).markdown

    assert len(client.chat.completions.calls) == 1
    assert markdown.endswith(_LATEX)
