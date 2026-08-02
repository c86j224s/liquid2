from __future__ import annotations

from copy import deepcopy
import json
from pathlib import Path

from report_natural_voice_correction.archive import read_json, write_json_atomic, write_text_atomic
from report_natural_voice_correction.edits import StructuredResponse, parse_model_response

from . import config
from .archive import ExperimentArchive
from .hash_contract import HashContractError, normalize_hashes, validate_hash_artifacts


class ContractError(ValueError):
    pass


NORMALIZATION_FIELDS = (
    "normalized_output_path",
    "normalized_output_sha256",
    "contract_normalization_path",
    "contract_normalization_sha256",
)


def parse_with_amendment(
    archive: ExperimentArchive,
    raw_path: Path,
    run_dir: Path,
    original: str,
) -> tuple[StructuredResponse, dict[str, str]]:
    raw = read_json(raw_path)
    additions = _missing_diagnoses(raw)
    if not additions:
        value = raw
        diagnosis_fields: dict[str, str] = {}
    else:
        _validate_amendment(archive, raw_path, additions)
        value = _normalized(raw, additions)
        normalized_path = run_dir / "normalized-output.json"
        normalization_path = run_dir / "contract-normalization.json"
        write_text_atomic(
            normalized_path,
            json.dumps(value, ensure_ascii=False, indent=2, sort_keys=True) + "\n",
        )
        write_json_atomic(normalization_path, {
            "experiment_id": config.EXPERIMENT_ID,
            "rule": config.CONTRACT_AMENDMENT_RULE,
            "raw_output_sha256": config.sha256_file(raw_path),
            "normalized_output_sha256": config.sha256_file(normalized_path),
            "added_diagnoses": additions,
            "edits_changed": False,
        })
        diagnosis_fields = {
            "normalized_output_path": archive.rel(normalized_path),
            "normalized_output_sha256": config.sha256_file(normalized_path),
            "contract_normalization_path": archive.rel(normalization_path),
            "contract_normalization_sha256": config.sha256_file(normalization_path),
        }
    value, hash_fields = normalize_hashes(archive, value, raw_path, run_dir, original)
    return parse_model_response(value), {**diagnosis_fields, **hash_fields}


def validate_response_artifacts(
    archive: ExperimentArchive,
    record: dict[str, object],
    run_dir: Path,
    raw_path: Path,
    original: str,
) -> None:
    present = {field for field in NORMALIZATION_FIELDS if field in record}
    additions = _missing_diagnoses(read_json(raw_path))
    if not present:
        if additions:
            raise ContractError("missing diagnosis normalization artifacts")
        value = read_json(raw_path)
    else:
        if present != set(NORMALIZATION_FIELDS) or not additions:
            raise ContractError("normalization artifact fields are incomplete or unexpected")
        _validate_amendment(archive, raw_path, additions)
        normalized_path = run_dir / "normalized-output.json"
        normalization_path = run_dir / "contract-normalization.json"
        _validate_recorded_artifact(archive, record, normalized_path, "normalized_output")
        _validate_recorded_artifact(archive, record, normalization_path, "contract_normalization")
        expected = _normalized(read_json(raw_path), additions)
        value = read_json(normalized_path)
        if value != expected:
            raise ContractError("normalized response changed")
        normalization = read_json(normalization_path)
        if normalization != {
            "experiment_id": config.EXPERIMENT_ID,
            "rule": config.CONTRACT_AMENDMENT_RULE,
            "raw_output_sha256": config.sha256_file(raw_path),
            "normalized_output_sha256": config.sha256_file(normalized_path),
            "added_diagnoses": additions,
            "edits_changed": False,
        }:
            raise ContractError("contract normalization record changed")
    try:
        value = validate_hash_artifacts(archive, record, run_dir, raw_path, value, original)
    except HashContractError as exc:
        raise ContractError(str(exc)) from exc
    parse_model_response(value)


def _missing_diagnoses(raw: dict[str, object]) -> list[dict[str, object]]:
    diagnoses = raw.get("diagnoses")
    edits = raw.get("edits")
    if not isinstance(diagnoses, list) or not isinstance(edits, list):
        return []
    diagnosed = {
        row.get("category")
        for row in diagnoses
        if isinstance(row, dict) and isinstance(row.get("category"), str)
    }
    by_category: dict[str, set[int]] = {}
    for row in edits:
        if not isinstance(row, dict):
            continue
        category = row.get("category")
        line_number = row.get("line_number")
        if isinstance(category, str) and category not in diagnosed and type(line_number) is int:
            by_category.setdefault(category, set()).add(line_number)
    return [
        {"category": category, "evidence_line_numbers": sorted(lines)}
        for category, lines in sorted(by_category.items())
    ]


def _normalized(
    raw: dict[str, object],
    additions: list[dict[str, object]],
) -> dict[str, object]:
    value = deepcopy(raw)
    diagnoses = value.get("diagnoses")
    if not isinstance(diagnoses, list) or len(diagnoses) + len(additions) > 6:
        raise ContractError("missing diagnoses cannot be normalized within the six-category limit")
    diagnoses.extend(deepcopy(additions))
    return value


def _validate_amendment(
    archive: ExperimentArchive,
    raw_path: Path,
    additions: list[dict[str, object]],
) -> None:
    path = archive.root / "control" / "protocol-amendment-01.lock.json"
    if not path.is_file():
        raise ContractError("contract normalization requires the locked protocol amendment")
    lock = read_json(path)
    expected_keys = {
        "experiment_id", "status", "base_protocol_sha256", "trigger",
        "rule", "scope", "model_calls_retried",
    }
    if set(lock) != expected_keys:
        raise ContractError("protocol amendment schema mismatch")
    if (
        lock.get("experiment_id") != config.EXPERIMENT_ID
        or lock.get("status") != "locked_before_contract_normalization"
        or lock.get("base_protocol_sha256")
        != config.sha256_file(archive.root / "control" / "protocol.lock.json")
        or lock.get("rule") != config.CONTRACT_AMENDMENT_RULE
        or lock.get("scope") != "all_experiment_59_cells"
        or lock.get("model_calls_retried") is not False
    ):
        raise ContractError("protocol amendment identity or policy mismatch")
    trigger = lock.get("trigger")
    if not isinstance(trigger, dict) or set(trigger) != {
        "raw_output_path", "raw_output_sha256", "added_diagnoses",
    }:
        raise ContractError("protocol amendment trigger mismatch")
    trigger_path = archive.root / str(trigger["raw_output_path"])
    trigger_additions = _missing_diagnoses(read_json(trigger_path)) if trigger_path.is_file() else []
    if (
        not trigger_path.is_file()
        or config.sha256_file(trigger_path) != trigger["raw_output_sha256"]
        or trigger.get("added_diagnoses") != trigger_additions
    ):
        raise ContractError("protocol amendment trigger evidence changed")
    if raw_path == trigger_path and additions != trigger_additions:
        raise ContractError("triggering diagnosis normalization changed")


def _validate_recorded_artifact(
    archive: ExperimentArchive,
    record: dict[str, object],
    path: Path,
    stem: str,
) -> None:
    if record.get(f"{stem}_path") != archive.rel(path) or not path.is_file():
        raise ContractError(f"{stem} artifact path mismatch")
    if record.get(f"{stem}_sha256") != config.sha256_file(path):
        raise ContractError(f"{stem} artifact hash mismatch")
