from __future__ import annotations

from pathlib import Path

from report_natural_voice_correction.archive import read_json, write_json_atomic

from . import config
from .archive import ExperimentArchive
from .blind import validate_private_mapping, validate_verdict_lock
from .records import validate_record


class ReviewError(ValueError):
    pass


DRIFT_KEYS = ("semantic_drift_lines", "claim_scope_drift_lines", "citation_drift_lines")


def make_semantic_audit_pack(archive: ExperimentArchive) -> dict[str, object]:
    validate_verdict_lock(archive)
    validate_private_mapping(archive)
    path = archive.root / "analysis" / "semantic-audit-pack.json"
    if path.exists():
        raise ReviewError("semantic audit pack already exists")
    cells: list[dict[str, object]] = []
    for file_id in archive.evaluation_file_ids():
        original = archive.input_path("evaluation", file_id).read_text(encoding="utf-8").splitlines()
        for arm in config.ARMS:
            run_dir = archive.root / "runs" / "full" / file_id / arm
            record = read_json(run_dir / "record.json")
            validate_record(archive, record, "evaluation", ("full",), file_id, arm)
            decisions = read_json(run_dir / "edit-decisions.json").get("decisions")
            if not isinstance(decisions, list):
                raise ReviewError("edit decisions missing")
            edits: list[dict[str, object]] = []
            candidate = (archive.root / str(record["candidate_path"])).read_text(encoding="utf-8").splitlines()
            for row in decisions:
                if not isinstance(row, dict) or row.get("disposition") != "accepted":
                    continue
                line_number = int(row["line_number"])
                start = max(0, line_number - 2)
                end = min(len(original), line_number + 1)
                edits.append({
                    "line_number": line_number,
                    "category": row["category"],
                    "original_context": original[start:end],
                    "candidate_context": candidate[start:end],
                })
            edits.sort(key=lambda edit: int(edit["line_number"]))
            cells.append({"file_id": file_id, "arm": arm, "accepted_edits": edits})
    payload = {"experiment_id": config.EXPERIMENT_ID, "cells": cells}
    write_json_atomic(path, payload)
    return {"cells": len(cells), "accepted_edits": sum(len(row["accepted_edits"]) for row in cells), "path": archive.rel(path)}


def record_semantic_audit(archive: ExperimentArchive, audit_path: Path) -> dict[str, object]:
    lock_path = archive.root / "analysis" / "semantic-audit.lock.json"
    if lock_path.exists():
        raise ReviewError("semantic audit is already locked")
    pack = read_json(archive.root / "analysis" / "semantic-audit-pack.json")
    expected = _expected_lines(pack)
    value = read_json(Path(audit_path))
    rows = value.get("audits")
    if not isinstance(rows, list):
        raise ReviewError("semantic audit input must contain an audits array")
    locked = _validate_audit_rows(rows, expected)
    payload = {
        "experiment_id": config.EXPERIMENT_ID,
        "audit_pack_sha256": config.sha256_file(archive.root / "analysis" / "semantic-audit-pack.json"),
        "audits": sorted(locked, key=lambda row: (str(row["file_id"]), str(row["arm"]))),
    }
    write_json_atomic(lock_path, payload)
    return {"locked_cells": len(locked), "path": archive.rel(lock_path)}


def validate_semantic_audit(archive: ExperimentArchive) -> dict[str, object]:
    pack_path = archive.root / "analysis" / "semantic-audit-pack.json"
    lock = read_json(archive.root / "analysis" / "semantic-audit.lock.json")
    if set(lock) != {"experiment_id", "audit_pack_sha256", "audits"}:
        raise ReviewError("semantic audit lock schema mismatch")
    if lock.get("experiment_id") != config.EXPERIMENT_ID:
        raise ReviewError("semantic audit identity mismatch")
    if lock.get("audit_pack_sha256") != config.sha256_file(pack_path):
        raise ReviewError("semantic audit pack changed after review")
    expected = _expected_lines(read_json(pack_path))
    _validate_audit_rows(lock.get("audits"), expected)
    return lock


def _validate_audit_rows(
    rows: object,
    expected: dict[tuple[str, str], list[int]],
) -> list[dict[str, object]]:
    if not isinstance(rows, list):
        raise ReviewError("semantic audit must contain an audits array")
    seen: set[tuple[str, str]] = set()
    validated: list[dict[str, object]] = []
    for row in rows:
        if not isinstance(row, dict) or set(row) != {
            "file_id", "arm", "reviewed_line_numbers", *DRIFT_KEYS, "notes"
        }:
            raise ReviewError("semantic audit row schema mismatch")
        key = (str(row["file_id"]), str(row["arm"]))
        if key not in expected or key in seen:
            raise ReviewError("semantic audit cell is missing, duplicate, or unexpected")
        reviewed = row["reviewed_line_numbers"]
        if reviewed != expected[key]:
            raise ReviewError("semantic audit must cover every accepted edit exactly")
        for drift_key in DRIFT_KEYS:
            lines = row[drift_key]
            if (
                not isinstance(lines, list)
                or any(type(line) is not int for line in lines)
                or lines != sorted(set(lines))
                or any(line not in reviewed for line in lines)
            ):
                raise ReviewError("drift lines must be a unique ordered subset of reviewed lines")
        notes = row["notes"]
        if not isinstance(notes, str) or not notes.strip() or len(notes.strip()) > 1000:
            raise ReviewError("semantic audit notes must be short and non-empty")
        validated.append({**row, "notes": notes.strip()})
        seen.add(key)
    if seen != set(expected):
        raise ReviewError("all sixteen semantic audit cells must be locked together")
    return validated


def _expected_lines(pack: dict[str, object]) -> dict[tuple[str, str], list[int]]:
    cells = pack.get("cells")
    if pack.get("experiment_id") != config.EXPERIMENT_ID or not isinstance(cells, list):
        raise ReviewError("semantic audit pack identity mismatch")
    expected: dict[tuple[str, str], list[int]] = {}
    for cell in cells:
        if not isinstance(cell, dict) or not isinstance(cell.get("accepted_edits"), list):
            raise ReviewError("semantic audit pack cell mismatch")
        key = (str(cell.get("file_id")), str(cell.get("arm")))
        lines = [int(edit["line_number"]) for edit in cell["accepted_edits"] if isinstance(edit, dict)]
        if key in expected or lines != sorted(set(lines)):
            raise ReviewError("semantic audit pack lines are duplicate or unordered")
        expected[key] = lines
    if set(expected) != {(file_id, arm) for file_id in config.file_id_set() for arm in config.ARMS}:
        raise ReviewError("semantic audit pack corpus mismatch")
    return expected
