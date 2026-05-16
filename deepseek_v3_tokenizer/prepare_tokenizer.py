#!/usr/bin/env python3
"""Build converted_tokenizer.json for Go (matches AutoTokenizer.encode).

The official tokenizer.json alone does not match transformers.AutoTokenizer:
LlamaTokenizerFast replaces the pretokenizer with Metaspace. This script exports
the effective runtime tokenizer.

  pip install transformers
  python3 prepare_tokenizer.py
"""
from __future__ import annotations

import sys

try:
    import transformers
except ImportError:
    print("Install transformers: pip install transformers", file=sys.stderr)
    raise SystemExit(1) from None

OUT = "converted_tokenizer.json"


def main() -> None:
    tok = transformers.AutoTokenizer.from_pretrained(".", trust_remote_code=True)
    tok.backend_tokenizer.save(OUT)
    sample = "Hello world"
    ids = tok.encode(sample)
    print(f"Wrote {OUT}")
    print(f"Sanity check encode({sample!r}): {ids}")


if __name__ == "__main__":
    main()
