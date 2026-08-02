from __future__ import annotations

from pathlib import Path

from report_natural_voice_correction.archive import read_json, write_json_atomic
from report_natural_voice_correction.edits import LineEdit, StructuredResponse, candidate_for_edits
from report_natural_voice_correction.guards import HARD_GATE_REASON_CODES, hard_gate_failures

from . import config
from .archive import ExperimentArchive


class PolicyError(ValueError):
    pass


def apply_selective_policy(
    archive: ExperimentArchive,
    run_dir: Path,
    file_id: str,
    arm: str,
    original: str,
    response: StructuredResponse,
) -> tuple[str, dict[str, object]]:
    accepted: list[LineEdit] = []
    decisions: list[dict[str, object]] = []
    for edit in response.edits:
        one_edit = candidate_for_edits(original, response, (edit,))
        failures = _known_failures(original, one_edit)
        disposition = "rejected" if failures else "accepted"
        if not failures:
            accepted.append(edit)
        decisions.append({
            "line_number": edit.line_number,
            "category": edit.category,
            "disposition": disposition,
            "reason_codes": failures,
        })

    decisions_path = run_dir / "edit-decisions.json"
    decisions_payload = {
        "experiment_id": config.EXPERIMENT_ID,
        "file_id": file_id,
        "arm": arm,
        "proposed_count": len(response.edits),
        "accepted_count": len(accepted),
        "rejected_count": len(response.edits) - len(accepted),
        "decisions": decisions,
    }
    _validate_decisions(decisions_payload, response)
    write_json_atomic(decisions_path, decisions_payload)

    candidate = candidate_for_edits(original, response, tuple(accepted))
    aggregate_failures = _known_failures(original, candidate)
    aggregate_path = run_dir / "aggregate-gates.json"
    write_json_atomic(aggregate_path, {
        "experiment_id": config.EXPERIMENT_ID,
        "file_id": file_id,
        "arm": arm,
        "proposed_count": len(response.edits),
        "accepted_count": len(accepted),
        "rejected_count": len(response.edits) - len(accepted),
        "aggregate_hard_gate_failures": aggregate_failures,
        "hard_gates_passed": not aggregate_failures,
    })
    if aggregate_failures:
        raise PolicyError("aggregate hard gates failed")
    return candidate, {
        "proposed_edit_count": len(response.edits),
        "accepted_edit_count": len(accepted),
        "rejected_edit_count": len(response.edits) - len(accepted),
        "edit_decisions_path": archive.rel(decisions_path),
        "edit_decisions_sha256": config.sha256_file(decisions_path),
        "aggregate_gates_path": archive.rel(aggregate_path),
        "aggregate_gates_sha256": config.sha256_file(aggregate_path),
        "aggregate_hard_gate_failures": aggregate_failures,
    }


def validate_policy_artifacts(archive: ExperimentArchive, record: dict[str, object], run_dir: Path) -> None:
    proposed = record.get("proposed_edit_count")
    accepted = record.get("accepted_edit_count")
    rejected = record.get("rejected_edit_count")
    if not all(isinstance(value, int) for value in (proposed, accepted, rejected)):
        raise PolicyError("edit counts must be integers")
    if proposed != accepted + rejected:  # type: ignore[operator]
        raise PolicyError("edit counts are inconsistent")
    decisions_path = _validate_artifact(archive, record, run_dir, "edit_decisions")
    aggregate_path = _validate_artifact(archive, record, run_dir, "aggregate_gates")
    decisions = read_json(decisions_path)
    aggregate = read_json(aggregate_path)
    if decisions.get("proposed_count") != proposed or decisions.get("accepted_count") != accepted:
        raise PolicyError("edit decision counts changed")
    if aggregate.get("hard_gates_passed") is not True or aggregate.get("aggregate_hard_gate_failures") != []:
        raise PolicyError("aggregate gate artifact changed or failed")


def _known_failures(original: str, candidate: str) -> list[str]:
    failures = hard_gate_failures(original, candidate)
    unknown = sorted(set(failures) - set(HARD_GATE_REASON_CODES))
    if unknown:
        raise PolicyError("unknown hard-gate reason: " + ", ".join(unknown))
    return failures


def _validate_decisions(payload: dict[str, object], response: StructuredResponse) -> None:
    decisions = payload["decisions"]
    if not isinstance(decisions, list) or len(decisions) != len(response.edits):
        raise PolicyError("every proposed edit must have one decision")
    expected = {edit.line_number: edit.category for edit in response.edits}
    seen: set[int] = set()
    for row in decisions:
        if not isinstance(row, dict) or set(row) != {"line_number", "category", "disposition", "reason_codes"}:
            raise PolicyError("edit decision schema mismatch")
        line = row["line_number"]
        if not isinstance(line, int) or line in seen or expected.get(line) != row["category"]:
            raise PolicyError("edit decision identity mismatch")
        reasons = row["reason_codes"]
        if not isinstance(reasons, list) or any(reason not in HARD_GATE_REASON_CODES for reason in reasons):
            raise PolicyError("edit decision reason mismatch")
        if (row["disposition"] == "accepted") != (reasons == []):
            raise PolicyError("edit decision disposition mismatch")
        seen.add(line)
    if seen != set(expected):
        raise PolicyError("edit decisions are incomplete")


def _validate_artifact(
    archive: ExperimentArchive,
    record: dict[str, object],
    run_dir: Path,
    stem: str,
) -> Path:
    path = run_dir / f"{stem.replace('_', '-')}.json"
    if record.get(f"{stem}_path") != archive.rel(path) or not path.is_file():
        raise PolicyError(f"{stem} artifact path mismatch")
    if record.get(f"{stem}_sha256") != config.sha256_file(path):
        raise PolicyError(f"{stem} artifact hash mismatch")
    return path
