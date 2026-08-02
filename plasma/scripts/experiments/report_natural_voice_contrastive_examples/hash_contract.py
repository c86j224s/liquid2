from __future__ import annotations

from copy import deepcopy
import json
from pathlib import Path

from report_natural_voice_correction.archive import read_json, write_json_atomic, write_text_atomic
from report_natural_voice_correction.edits import split_document_lines

from . import config
from .archive import ExperimentArchive


class HashContractError(ValueError):
    pass


HASH_NORMALIZATION_FIELDS = (
    "hash_normalized_output_path",
    "hash_normalized_output_sha256",
    "hash_contract_normalization_path",
    "hash_contract_normalization_sha256",
)


def normalize_hashes(
    archive: ExperimentArchive,
    value: dict[str, object],
    raw_path: Path,
    run_dir: Path,
    original: str,
) -> tuple[dict[str, object], dict[str, str]]:
    corrections = _hash_corrections(value, original)
    if not corrections:
        return value, {}
    _validate_amendment(archive, raw_path, corrections, original)
    normalized = _normalized(value, corrections)
    normalized_path = run_dir / "hash-normalized-output.json"
    normalization_path = run_dir / "hash-contract-normalization.json"
    write_text_atomic(
        normalized_path,
        json.dumps(normalized, ensure_ascii=False, indent=2, sort_keys=True) + "\n",
    )
    write_json_atomic(normalization_path, _normalization_record(
        archive, value, raw_path, normalized_path, corrections
    ))
    return normalized, {
        "hash_normalized_output_path": archive.rel(normalized_path),
        "hash_normalized_output_sha256": config.sha256_file(normalized_path),
        "hash_contract_normalization_path": archive.rel(normalization_path),
        "hash_contract_normalization_sha256": config.sha256_file(normalization_path),
    }


def validate_hash_artifacts(
    archive: ExperimentArchive,
    record: dict[str, object],
    run_dir: Path,
    raw_path: Path,
    input_value: dict[str, object],
    original: str,
) -> dict[str, object]:
    present = {field for field in HASH_NORMALIZATION_FIELDS if field in record}
    corrections = _hash_corrections(input_value, original)
    if not present:
        if corrections:
            raise HashContractError("missing hash normalization artifacts")
        return input_value
    if present != set(HASH_NORMALIZATION_FIELDS) or not corrections:
        raise HashContractError("hash normalization artifact fields are incomplete or unexpected")
    _validate_amendment(archive, raw_path, corrections, original)
    normalized_path = run_dir / "hash-normalized-output.json"
    normalization_path = run_dir / "hash-contract-normalization.json"
    _validate_artifact(archive, record, normalized_path, "hash_normalized_output")
    _validate_artifact(archive, record, normalization_path, "hash_contract_normalization")
    expected = _normalized(input_value, corrections)
    actual = read_json(normalized_path)
    if actual != expected:
        raise HashContractError("hash-normalized response changed")
    expected_record = _normalization_record(
        archive, input_value, raw_path, normalized_path, corrections
    )
    if read_json(normalization_path) != expected_record:
        raise HashContractError("hash normalization record changed")
    return actual


def _hash_corrections(
    value: dict[str, object],
    original: str,
) -> list[dict[str, object]]:
    edits = value.get("edits")
    if not isinstance(edits, list):
        return []
    source_lines = split_document_lines(original)
    corrections: list[dict[str, object]] = []
    for row in edits:
        if not isinstance(row, dict):
            continue
        line_number = row.get("line_number")
        copied_line = row.get("original_line")
        claimed = row.get("original_line_sha256")
        if (
            type(line_number) is not int
            or line_number < 1
            or line_number > len(source_lines)
            or not isinstance(copied_line, str)
            or copied_line != source_lines[line_number - 1]
            or not isinstance(claimed, str)
            or len(claimed) != 64
        ):
            continue
        derived = config.sha256_text(copied_line)
        if claimed != derived:
            corrections.append({
                "line_number": line_number,
                "claimed_sha256": claimed,
                "derived_sha256": derived,
            })
    return corrections


def _normalized(
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


def _validate_amendment(
    archive: ExperimentArchive,
    raw_path: Path,
    corrections: list[dict[str, object]],
    original: str,
) -> None:
    path = archive.root / "control" / "protocol-amendment-02.lock.json"
    if not path.is_file():
        raise HashContractError("hash normalization requires the locked protocol amendment")
    lock = read_json(path)
    if set(lock) != {
        "experiment_id", "status", "base_protocol_sha256", "trigger",
        "rule", "scope", "model_calls_retried",
    }:
        raise HashContractError("hash amendment schema mismatch")
    if (
        lock.get("experiment_id") != config.EXPERIMENT_ID
        or lock.get("status") != "locked_before_hash_normalization"
        or lock.get("base_protocol_sha256")
        != config.sha256_file(archive.root / "control" / "protocol.lock.json")
        or lock.get("rule") != config.HASH_AMENDMENT_RULE
        or lock.get("scope") != "all_experiment_59_cells"
        or lock.get("model_calls_retried") is not False
    ):
        raise HashContractError("hash amendment identity or policy mismatch")
    trigger = lock.get("trigger")
    if not isinstance(trigger, dict) or set(trigger) != {
        "file_id", "raw_output_path", "raw_output_sha256", "corrections",
    }:
        raise HashContractError("hash amendment trigger mismatch")
    trigger_path = archive.root / str(trigger["raw_output_path"])
    trigger_value = read_json(trigger_path) if trigger_path.is_file() else {}
    trigger_original = archive.input_path("evaluation", str(trigger["file_id"])).read_text(encoding="utf-8")
    trigger_corrections = _hash_corrections(trigger_value, trigger_original)
    if (
        not trigger_path.is_file()
        or config.sha256_file(trigger_path) != trigger["raw_output_sha256"]
        or trigger.get("corrections") != trigger_corrections
    ):
        raise HashContractError("hash amendment trigger evidence changed")
    if raw_path == trigger_path and corrections != trigger_corrections:
        raise HashContractError("triggering hash normalization changed")


def _normalization_record(
    archive: ExperimentArchive,
    input_value: dict[str, object],
    raw_path: Path,
    normalized_path: Path,
    corrections: list[dict[str, object]],
) -> dict[str, object]:
    canonical = json.dumps(input_value, ensure_ascii=False, sort_keys=True, separators=(",", ":"))
    return {
        "experiment_id": config.EXPERIMENT_ID,
        "rule": config.HASH_AMENDMENT_RULE,
        "raw_output_sha256": config.sha256_file(raw_path),
        "input_response_sha256": config.sha256_text(canonical),
        "normalized_output_sha256": config.sha256_file(normalized_path),
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
    if record.get(f"{stem}_sha256") != config.sha256_file(path):
        raise HashContractError(f"{stem} artifact hash mismatch")
