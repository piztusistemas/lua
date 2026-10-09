# MarkItDown Math Plugin

MarkItDown plugin that converts PDFs containing mathematics to Markdown with the formulas written as LaTeX:

```markdown
## Quadratic formula

The roots of $ax^2+bx+c=0$ are given by

$$ x = \frac{-b \pm \sqrt{b^2-4ac}}{2a}. $$
```

Text extracted from a PDF loses the structure of formulas (fractions, roots, sub/superscripts), so this plugin renders each page that contains mathematics to an image and asks a vision LLM to transcribe it. It uses the same `llm_client` / `llm_model` pattern that MarkItDown already supports for image descriptions.

## How it works

1. Each page is checked for mathematics: characters set in math fonts (TeX `CMMI`/`CMSY`/`CMEX`, AMS `MSAM`/`MSBM`, Cambria Math, STIX Math, ...) or mathematical Unicode symbols (`∑`, `∫`, `≤`, `𝑥`, ...).
2. Pages with mathematics are rendered with `pypdfium2` and transcribed by the LLM into Markdown with inline `$...$` and display `$$...$$` math.
3. Pages without mathematics keep MarkItDown's regular text extraction, so no LLM calls are spent on them.
4. If an LLM call fails, that page falls back to regular text extraction and a warning is emitted.

Without an `llm_client` the plugin behaves exactly like the built-in PDF converter.

## Installation

```bash
pip install markitdown-math[llm]   # includes the openai client
```

## Usage

### Command line

```bash
export OPENAI_API_KEY=...
markitdown-math paper.pdf -o paper.md --model gpt-4o
```

Options:

| Option | Description |
|--------|-------------|
| `-m`, `--model` | Vision-capable model (default `$MARKITDOWN_MATH_MODEL` or `gpt-4o`) |
| `--base-url` | Any OpenAI-compatible endpoint (default `$OPENAI_BASE_URL`) |
| `--mode` | `auto` (only pages with math), `all` (every page) or `off` |
| `--dpi` | Page render resolution (default 200) |
| `--prompt` | Custom transcription prompt |

### Python API

```python
from markitdown import MarkItDown
from openai import OpenAI

md = MarkItDown(
    enable_plugins=True,
    llm_client=OpenAI(),
    llm_model="gpt-4o",
    pdf_math_mode="auto",  # optional: "auto" | "all" | "off"
    pdf_math_dpi=200,      # optional
)
result = md.convert("paper.pdf")
print(result.markdown)
```

`pdf_math_mode` can also be passed to `convert()` to override it for a single document.

Use `"all"` for scanned PDFs: they have no text layer, so mathematics cannot be detected automatically.

## Running tests

```bash
cd packages/markitdown-math
pip install -e ".[llm]" pytest
pytest
```

The tests use a mock LLM client, so no API key is needed. The fixture `tests/test_files/math.pdf` is built from `math.tex` with `pdflatex`.
