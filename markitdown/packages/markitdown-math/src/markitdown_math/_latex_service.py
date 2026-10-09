"""
LLM Vision service that transcribes a rendered PDF page into Markdown with
embedded LaTeX.
"""

import base64
import re
from typing import Any
from warnings import warn

DEFAULT_PROMPT = (
    "Transcribe this document page into Markdown.\n"
    "- Write every mathematical expression in LaTeX: inline math between single "
    "dollar signs ($...$) and displayed equations between double dollar signs "
    "($$...$$) on their own lines.\n"
    "- Keep equation numbers with \\tag{...} inside the displayed equation.\n"
    "- Preserve headings, paragraphs, lists and tables using Markdown syntax.\n"
    "- Keep the original language and wording; do not translate, summarise or "
    "solve anything.\n"
    "- Omit running headers, footers and page numbers.\n"
    "- Return only the Markdown, without code fences or commentary."
)

# Some models wrap their whole answer in ```markdown ... ``` despite the prompt.
_CODE_FENCE_PATTERN = re.compile(
    r"^\s*```(?:markdown|md|latex|tex)?\s*\n(.*?)\n\s*```\s*$",
    re.DOTALL | re.IGNORECASE,
)


def strip_code_fence(text: str) -> str:
    match = _CODE_FENCE_PATTERN.match(text)
    return match.group(1) if match else text


class LLMVisionLatexService:
    """Transcribes page images to Markdown + LaTeX using an OpenAI-compatible client."""

    def __init__(
        self,
        client: Any,
        model: str,
        prompt: str | None = None,
    ) -> None:
        self.client = client
        self.model = model
        self.prompt = prompt or DEFAULT_PROMPT

    def transcribe(self, png_bytes: bytes) -> str | None:
        """
        Return the page transcription, or None if the LLM call failed so the
        caller can fall back to plain text extraction.
        """
        data_uri = "data:image/png;base64," + base64.b64encode(png_bytes).decode(
            "utf-8"
        )
        try:
            response = self.client.chat.completions.create(
                model=self.model,
                messages=[
                    {
                        "role": "user",
                        "content": [
                            {"type": "text", "text": self.prompt},
                            {"type": "image_url", "image_url": {"url": data_uri}},
                        ],
                    }
                ],
            )
            text = response.choices[0].message.content
        except Exception as e:
            warn(f"LaTeX transcription failed, falling back to text extraction: {e}")
            return None

        if not text or not text.strip():
            return None
        return strip_code_fence(text.strip()).strip()
