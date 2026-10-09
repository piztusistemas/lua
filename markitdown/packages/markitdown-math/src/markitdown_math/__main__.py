"""
Command line entry point: convert a PDF to Markdown with embedded LaTeX.

    markitdown-math paper.pdf -o paper.md --model gpt-4o

The OpenAI client reads OPENAI_API_KEY from the environment. Any
OpenAI-compatible endpoint can be used with --base-url (or OPENAI_BASE_URL).
"""

import argparse
import os
import sys

from markitdown import MarkItDown

from ._pdf_converter_with_math import DEFAULT_DPI


def main() -> None:
    parser = argparse.ArgumentParser(
        prog="markitdown-math",
        description="Convert a PDF to Markdown, writing mathematics as LaTeX.",
    )
    parser.add_argument("filename", help="PDF file to convert")
    parser.add_argument("-o", "--output", help="Output file (default: stdout)")
    parser.add_argument(
        "-m",
        "--model",
        default=os.environ.get("MARKITDOWN_MATH_MODEL", "gpt-4o"),
        help="Vision-capable model (default: $MARKITDOWN_MATH_MODEL or gpt-4o)",
    )
    parser.add_argument(
        "--base-url",
        default=os.environ.get("OPENAI_BASE_URL"),
        help="OpenAI-compatible API base URL (default: $OPENAI_BASE_URL)",
    )
    parser.add_argument(
        "--mode",
        choices=["auto", "all", "off"],
        default="auto",
        help="auto: only pages with detected math use the LLM; all: every page; off: plain extraction",
    )
    parser.add_argument(
        "--dpi",
        type=int,
        default=DEFAULT_DPI,
        help=f"Resolution used to render pages (default: {DEFAULT_DPI})",
    )
    parser.add_argument("--prompt", help="Custom transcription prompt")
    args = parser.parse_args()

    llm_client = None
    if args.mode != "off":
        try:
            from openai import OpenAI
        except ImportError:
            sys.exit(
                "The openai package is required: pip install 'markitdown-math[llm]'"
            )
        llm_client = OpenAI(base_url=args.base_url)

    markitdown = MarkItDown(
        enable_plugins=True,
        llm_client=llm_client,
        llm_model=args.model,
        pdf_math_mode=args.mode,
        pdf_math_dpi=args.dpi,
        pdf_math_prompt=args.prompt,
    )
    markdown = markitdown.convert(args.filename).markdown

    if args.output:
        with open(args.output, "w", encoding="utf-8") as f:
            f.write(markdown)
    else:
        sys.stdout.write(markdown + "\n")


if __name__ == "__main__":
    main()
