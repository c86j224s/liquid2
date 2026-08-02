from __future__ import annotations

from report_natural_voice_correction.archive import read_json, write_json_atomic
from report_natural_voice_correction.edits import parse_model_response
from report_natural_voice_examples.archive import ExperimentArchive

from . import config
from .hash_common import (
    AMENDMENT_PATH,
    HashContractError,
    find_hash_corrections,
    sha256_file,
)


def lock_protocol_amendment(archive: ExperimentArchive) -> dict[str, object]:
    path = archive.root / AMENDMENT_PATH
    if path.exists():
        lock = read_json(path)
        validate_amendment(archive, lock)
        return lock
    triggers = []
    for raw_path in sorted((archive.root / "runs" / "pilot").glob("*/*/raw-output.json")):
        file_id = raw_path.parents[1].name
        arm = raw_path.parent.name
        original = archive.input_path("development", file_id).read_text(encoding="utf-8")
        response = parse_model_response(raw_path.read_text(encoding="utf-8"))
        corrections = find_hash_corrections(response, original)
        if corrections:
            triggers.append({
                "file_id": file_id,
                "arm": arm,
                "raw_output_path": archive.rel(raw_path),
                "raw_output_sha256": sha256_file(raw_path),
                "corrections": corrections,
            })
    if not triggers:
        raise HashContractError("hash amendment requires preserved pilot trigger evidence")
    lock = {
        "experiment_id": config.EXPERIMENT_ID,
        "status": "locked_after_pilot_before_full_calls",
        "base_protocol_sha256": sha256_file(archive.root / "control" / "protocol.lock.json"),
        "rule": config.HASH_AMENDMENT_RULE,
        "scope": "all experiment 60 pilot and full cells",
        "triggers": triggers,
        "model_calls_retried": False,
        "prose_fields_changed": False,
    }
    write_json_atomic(path, lock)
    validate_amendment(archive, lock)
    return lock


def validate_amendment(archive: ExperimentArchive, lock: dict[str, object]) -> None:
    expected_keys = {
        "experiment_id", "status", "base_protocol_sha256", "rule", "scope",
        "triggers", "model_calls_retried", "prose_fields_changed",
    }
    if set(lock) != expected_keys:
        raise HashContractError("hash amendment schema mismatch")
    if (
        lock.get("experiment_id") != config.EXPERIMENT_ID
        or lock.get("status") != "locked_after_pilot_before_full_calls"
        or lock.get("base_protocol_sha256")
        != sha256_file(archive.root / "control" / "protocol.lock.json")
        or lock.get("rule") != config.HASH_AMENDMENT_RULE
        or lock.get("scope") != "all experiment 60 pilot and full cells"
        or lock.get("model_calls_retried") is not False
        or lock.get("prose_fields_changed") is not False
        or not isinstance(lock.get("triggers"), list)
        or not lock["triggers"]
    ):
        raise HashContractError("hash amendment identity or policy mismatch")
