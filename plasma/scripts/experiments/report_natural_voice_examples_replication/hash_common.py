from __future__ import annotations

import hashlib
from pathlib import Path

from report_natural_voice_correction.edits import StructuredResponse, split_document_lines


class HashContractError(ValueError):
    pass


AMENDMENT_PATH = Path("control") / "protocol-amendment-01.lock.json"


def find_hash_corrections(
    response: StructuredResponse,
    original: str,
) -> list[dict[str, object]]:
    lines = split_document_lines(original)
    corrections = []
    for edit in response.edits:
        if 1 <= edit.line_number <= len(lines) and edit.original_line == lines[edit.line_number - 1]:
            derived = sha256_text(edit.original_line)
            if edit.original_line_sha256 != derived:
                corrections.append({
                    "line_number": edit.line_number,
                    "claimed_sha256": edit.original_line_sha256,
                    "derived_sha256": derived,
                })
    return corrections


def sha256_file(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def sha256_text(value: str) -> str:
    return hashlib.sha256(value.encode("utf-8")).hexdigest()
