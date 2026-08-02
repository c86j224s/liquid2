from __future__ import annotations

from dataclasses import dataclass
import json
from typing import Any

from . import config


class EditError(ValueError):
    pass


@dataclass(frozen=True)
class Diagnosis:
    category: str
    evidence_line_numbers: tuple[int, ...]


@dataclass(frozen=True)
class LineEdit:
    line_number: int
    original_line_sha256: str
    original_line: str
    replacement_line: str
    category: str
    safety_rationale: str


@dataclass(frozen=True)
class StructuredResponse:
    document_sha256: str
    diagnoses: tuple[Diagnosis, ...]
    edits: tuple[LineEdit, ...]


def response_schema() -> dict[str, object]:
    return {
        "type": "object",
        "additionalProperties": False,
        "required": ["document_sha256", "diagnoses", "edits"],
        "properties": {
            "document_sha256": {"type": "string", "pattern": "^[0-9a-f]{64}$"},
            "diagnoses": {
                "type": "array",
                "minItems": 3,
                "maxItems": 6,
                "items": {
                    "type": "object",
                    "additionalProperties": False,
                    "required": ["category", "evidence_line_numbers"],
                    "properties": {
                        "category": {"type": "string", "enum": list(config.ALLOWED_CATEGORIES)},
                        "evidence_line_numbers": {
                            "type": "array",
                            "minItems": 1,
                            "items": {"type": "integer", "minimum": 1},
                        },
                    },
                },
            },
            "edits": {
                "type": "array",
                "maxItems": 24,
                "items": {
                    "type": "object",
                    "additionalProperties": False,
                    "required": [
                        "line_number",
                        "original_line_sha256",
                        "original_line",
                        "replacement_line",
                        "category",
                        "safety_rationale",
                    ],
                    "properties": {
                        "line_number": {"type": "integer", "minimum": 1},
                        "original_line_sha256": {"type": "string", "pattern": "^[0-9a-f]{64}$"},
                        "original_line": {"type": "string"},
                        "replacement_line": {"type": "string"},
                        "category": {"type": "string", "enum": list(config.ALLOWED_CATEGORIES)},
                        "safety_rationale": {"type": "string", "minLength": 1},
                    },
                },
            },
        },
    }


def parse_model_response(raw: str | dict[str, Any]) -> StructuredResponse:
    value = json.loads(raw) if isinstance(raw, str) else raw
    if not isinstance(value, dict):
        raise EditError("model response must be a JSON object")
    _require_keys(value, {"document_sha256", "diagnoses", "edits"}, "response")
    document_sha = _require_hex64(value.get("document_sha256"), "document_sha256")

    diagnoses_value = value.get("diagnoses")
    if not isinstance(diagnoses_value, list) or not 3 <= len(diagnoses_value) <= 6:
        raise EditError("diagnoses must contain 3 to 6 objects")
    categories: list[str] = []
    diagnoses: list[Diagnosis] = []
    for item in diagnoses_value:
        if not isinstance(item, dict):
            raise EditError("diagnosis must be an object")
        _require_keys(item, {"category", "evidence_line_numbers"}, "diagnosis")
        category = _require_category(item.get("category"))
        if category in categories:
            raise EditError("diagnosis category must not repeat")
        evidence = _require_line_numbers(item.get("evidence_line_numbers"), "evidence_line_numbers")
        categories.append(category)
        diagnoses.append(Diagnosis(category, tuple(evidence)))

    edits_value = value.get("edits")
    if not isinstance(edits_value, list):
        raise EditError("edits must be an array")
    if len(edits_value) > 24:
        raise EditError("edits must contain no more than 24 objects")
    seen_lines: set[int] = set()
    parsed_edits: list[LineEdit] = []
    for item in edits_value:
        if not isinstance(item, dict):
            raise EditError("edit must be an object")
        _require_keys(
            item,
            {"line_number", "original_line_sha256", "original_line", "replacement_line", "category", "safety_rationale"},
            "edit",
        )
        line_number = _require_positive_int(item.get("line_number"), "line_number")
        if line_number in seen_lines:
            raise EditError("duplicate line_number edits are not allowed")
        seen_lines.add(line_number)
        category = _require_category(item.get("category"))
        if category not in categories:
            raise EditError("edit category must appear in diagnoses")
        original_line = _require_single_line_string(item.get("original_line"), "original_line")
        replacement_line = _require_single_line_string(item.get("replacement_line"), "replacement_line")
        parsed_edits.append(LineEdit(
            line_number=line_number,
            original_line_sha256=_require_hex64(item.get("original_line_sha256"), "original_line_sha256"),
            original_line=original_line,
            replacement_line=replacement_line,
            category=category,
            safety_rationale=_require_non_empty_stripped_string(item.get("safety_rationale"), "safety_rationale"),
        ))
    return StructuredResponse(document_sha, tuple(diagnoses), tuple(parsed_edits))


def split_document_lines(text: str) -> list[str]:
    return text.rstrip("\n").split("\n")


def join_document_lines(lines: list[str], original_text: str) -> str:
    return "\n".join(lines) + ("\n" if original_text.endswith("\n") else "")


def apply_response(original_text: str, response: StructuredResponse) -> str:
    return candidate_for_edits(original_text, response, response.edits)


def candidate_for_edits(original_text: str, response: StructuredResponse, line_edits: tuple[LineEdit, ...]) -> str:
    if response.document_sha256 != config.sha256_text(original_text):
        raise EditError("document_sha256 does not match original document")
    lines = split_document_lines(original_text)
    _validate_diagnosis_evidence_line_numbers(response, len(lines))
    seen_lines: set[int] = set()
    candidate = list(lines)
    for edit in line_edits:
        if edit.line_number in seen_lines:
            raise EditError("duplicate line_number edits are not allowed")
        seen_lines.add(edit.line_number)
        index = edit.line_number - 1
        if index < 0 or index >= len(lines):
            raise EditError("edit line_number is outside the document")
        if lines[index] != edit.original_line:
            raise EditError("edit original_line is not byte-identical to the target line")
        if config.sha256_text(lines[index]) != edit.original_line_sha256:
            raise EditError("edit original_line_sha256 does not match the target line")
        candidate[index] = edit.replacement_line
    return join_document_lines(candidate, original_text)


def _validate_diagnosis_evidence_line_numbers(response: StructuredResponse, line_count: int) -> None:
    for diagnosis in response.diagnoses:
        for line_number in diagnosis.evidence_line_numbers:
            if line_number > line_count:
                raise EditError("diagnosis evidence_line_numbers entry is outside the document")


def _require_keys(value: dict[str, Any], expected: set[str], label: str) -> None:
    keys = set(value)
    if keys != expected:
        extra = sorted(keys - expected)
        missing = sorted(expected - keys)
        raise EditError(f"{label} keys mismatch; extra={extra}, missing={missing}")


def _require_string(value: Any, field: str) -> str:
    if not isinstance(value, str):
        raise EditError(f"{field} must be a string")
    return value


def _require_non_empty_stripped_string(value: Any, field: str) -> str:
    text = _require_string(value, field).strip()
    if not text:
        raise EditError(f"{field} must be non-empty after strip")
    return text


def _require_single_line_string(value: Any, field: str) -> str:
    text = _require_string(value, field)
    if "\n" in text or "\r" in text:
        raise EditError(f"{field} must be a single line")
    return text


def _require_positive_int(value: Any, field: str) -> int:
    if not isinstance(value, int) or isinstance(value, bool) or value < 1:
        raise EditError(f"{field} must be a 1-based integer")
    return value


def _require_line_numbers(value: Any, field: str) -> list[int]:
    if not isinstance(value, list) or not value:
        raise EditError(f"{field} must be a non-empty array")
    lines = [_require_positive_int(item, field) for item in value]
    if len(lines) != len(set(lines)):
        raise EditError(f"{field} must not contain duplicates")
    return lines


def _require_category(value: Any) -> str:
    if not isinstance(value, str) or value not in config.ALLOWED_CATEGORIES:
        raise EditError("category is not in the allowed enum")
    return value


def _require_hex64(value: Any, field: str) -> str:
    if not isinstance(value, str) or not config.HEX64_RE.match(value):
        raise EditError(f"{field} must be a 64-character lowercase hex SHA-256")
    return value
