from __future__ import annotations

from copy import deepcopy
from dataclasses import replace
from pathlib import Path

from report_natural_voice_correction.archive import read_json, write_json_atomic
from report_natural_voice_correction.edits import StructuredResponse, parse_model_response
from report_natural_voice_examples.archive import ExperimentArchive

from . import config
from .hash_amendment import lock_protocol_amendment, validate_amendment
from .hash_common import (
    AMENDMENT_PATH,
    HashContractError,
    find_hash_corrections,
    sha256_file,
)


HASH_FIELDS = (
    "hash_normalized_output_path",
    "hash_normalized_output_sha256",
    "hash_contract_normalization_path",
    "hash_contract_normalization_sha256",
)


def normalize_response_hashes(
    archive: ExperimentArchive,
    run_dir: Path,
    original: str,
    response: StructuredResponse,
) -> tuple[StructuredResponse, dict[str, str]]:
    corrections = find_hash_corrections(response, original)
    if not corrections:
        return response, {}
    amendment_path = archive.root / AMENDMENT_PATH
    if not amendment_path.is_file():
        raise HashContractError("hash normalization requires the locked protocol amendment")
    validate_amendment(archive, read_json(amendment_path))
    by_line = {int(row["line_number"]): row for row in corrections}
    normalized = replace(response, edits=tuple(
        replace(edit, original_line_sha256=str(by_line[edit.line_number]["derived_sha256"]))
        if edit.line_number in by_line else edit
        for edit in response.edits
    ))
    raw_path = run_dir / "raw-output.json"
    value = _normalized_value(read_json(raw_path), corrections)
    normalized_path = run_dir / "hash-normalized-output.json"
    record_path = run_dir / "hash-contract-normalization.json"
    write_json_atomic(normalized_path, value)
    write_json_atomic(record_path, _normalization_record(
        archive, raw_path, normalized_path, corrections
    ))
    return normalized, {
        "hash_normalized_output_path": archive.rel(normalized_path),
        "hash_normalized_output_sha256": sha256_file(normalized_path),
        "hash_contract_normalization_path": archive.rel(record_path),
        "hash_contract_normalization_sha256": sha256_file(record_path),
    }


def validate_hash_artifacts(
    archive: ExperimentArchive,
    record: dict[str, object],
    run_dir: Path,
) -> None:
    raw_path = run_dir / "raw-output.json"
    original = archive.input_path(
        str(record["set_name"]), str(record["file_id"])
    ).read_text(encoding="utf-8")
    response = parse_model_response(raw_path.read_text(encoding="utf-8"))
    corrections = find_hash_corrections(response, original)
    present = {field for field in HASH_FIELDS if field in record}
    if not corrections:
        if present:
            raise HashContractError("unexpected hash normalization artifacts")
        return
    if present != set(HASH_FIELDS):
        raise HashContractError("hash normalization artifact fields are incomplete")
    validate_amendment(archive, read_json(archive.root / AMENDMENT_PATH))
    normalized_path = run_dir / "hash-normalized-output.json"
    normalization_path = run_dir / "hash-contract-normalization.json"
    _validate_artifact(archive, record, normalized_path, "hash_normalized_output")
    _validate_artifact(archive, record, normalization_path, "hash_contract_normalization")
    if read_json(normalized_path) != _normalized_value(read_json(raw_path), corrections):
        raise HashContractError("hash-normalized response changed")
    expected = _normalization_record(archive, raw_path, normalized_path, corrections)
    if read_json(normalization_path) != expected:
        raise HashContractError("hash normalization record changed")


def _normalized_value(
    value: dict[str, object],
    corrections: list[dict[str, object]],
) -> dict[str, object]:
    normalized = deepcopy(value)
    edits = normalized.get("edits")
    if not isinstance(edits, list):
        raise HashContractError("hash normalization requires an edits array")
    by_line = {int(row["line_number"]): row for row in corrections}
    for edit in edits:
        if isinstance(edit, dict) and edit.get("line_number") in by_line:
            correction = by_line[int(edit["line_number"])]
            if edit.get("original_line_sha256") != correction["claimed_sha256"]:
                raise HashContractError("claimed line hash changed before normalization")
            edit["original_line_sha256"] = correction["derived_sha256"]
    return normalized


def _normalization_record(
    archive: ExperimentArchive,
    raw_path: Path,
    normalized_path: Path,
    corrections: list[dict[str, object]],
) -> dict[str, object]:
    return {
        "experiment_id": config.EXPERIMENT_ID,
        "amendment_sha256": sha256_file(archive.root / AMENDMENT_PATH),
        "rule": config.HASH_AMENDMENT_RULE,
        "raw_output_sha256": sha256_file(raw_path),
        "normalized_output_sha256": sha256_file(normalized_path),
        "corrections": corrections,
        "edit_text_changed": False,
    }


def _validate_artifact(
    archive: ExperimentArchive,
    record: dict[str, object],
    path: Path,
    stem: str,
) -> None:
    if record.get(f"{stem}_path") != archive.rel(path) or not path.is_file():
        raise HashContractError(f"{stem} artifact path mismatch")
    if record.get(f"{stem}_sha256") != sha256_file(path):
        raise HashContractError(f"{stem} artifact hash mismatch")
