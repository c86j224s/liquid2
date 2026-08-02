from __future__ import annotations

import random
from pathlib import Path
from typing import Any

from . import config
from .archive import ExperimentArchive, read_json, write_json_atomic
from .codex_runner import RunnerError, validate_success_record_artifacts


class BlindError(ValueError):
    pass


def make_blind_packets(archive: ExperimentArchive, seed: int | None = None) -> dict[str, object]:
    archive.verify_source_seal()
    records = _full_pass_records(archive)
    packet_dir = archive.root / "blind" / "packets"
    mapping_path = archive.root / "blind" / "private-mapping.lock.json"
    if mapping_path.exists() or packet_dir.exists():
        raise BlindError("blind packets already exist")
    packet_dir.mkdir(parents=True)
    rng = random.Random(seed) if seed is not None else random.SystemRandom()
    mappings: list[dict[str, object]] = []
    packet_ids: list[str] = []
    for index, record in enumerate(records, 1):
        packet_id = f"packet-{index:02d}"
        candidate_slot = rng.choice(("A", "B"))
        original_slot = "B" if candidate_slot == "A" else "A"
        original_text = (archive.root / "inputs" / str(record["source_filename"])).read_text(encoding="utf-8")
        candidate_path = archive.root / str(record["candidate_path"])
        candidate_text = candidate_path.read_text(encoding="utf-8")
        documents = {
            original_slot: original_text,
            candidate_slot: candidate_text,
        }
        packet_path = packet_dir / f"{packet_id}.json"
        write_json_atomic(packet_path, {
            "packet_id": packet_id,
            "documents": [
                {"slot": "A", "body": documents["A"]},
                {"slot": "B", "body": documents["B"]},
            ],
        })
        mappings.append({
            "packet_id": packet_id,
            "file_id": record["file_id"],
            "source_filename": record["source_filename"],
            "original_slot": original_slot,
            "candidate_slot": candidate_slot,
            "packet_path": archive.rel(packet_path),
            "packet_sha256": config.sha256_file(packet_path),
        })
        packet_ids.append(packet_id)
    mapping = {
        "experiment_id": archive.experiment_id,
        "packet_count": len(packet_ids),
        "packet_ids": packet_ids,
        "mappings": mappings,
    }
    write_json_atomic(mapping_path, mapping)
    return {"packet_count": len(packet_ids), "packet_ids": packet_ids, "mapping_path": archive.rel(mapping_path)}


def record_host_verdicts(archive: ExperimentArchive, verdicts_path: Path) -> dict[str, object]:
    if (archive.root / "analysis" / "public-summary.json").exists():
        raise BlindError("host verdicts cannot be changed after public summary export")
    lock_path = archive.root / "blind" / "host-verdicts.lock.json"
    if lock_path.exists():
        raise BlindError("host verdicts are already locked")
    packet_ids = validate_public_packets(archive)
    value = read_json(Path(verdicts_path))
    verdicts = value.get("verdicts")
    if not isinstance(verdicts, list):
        raise BlindError("verdicts file must contain a verdicts array")
    seen: set[str] = set()
    locked: list[dict[str, str]] = []
    for item in verdicts:
        if not isinstance(item, dict) or set(item) != {"packet_id", "choice", "rationale"}:
            raise BlindError("each verdict must contain packet_id, choice, and rationale only")
        packet_id = str(item["packet_id"])
        choice = str(item["choice"])
        rationale = str(item["rationale"]).strip()
        if packet_id not in packet_ids or packet_id in seen:
            raise BlindError("verdict packet_id is missing, duplicate, or unexpected")
        if choice not in {"A", "B", "tie"}:
            raise BlindError("verdict choice must be A, B, or tie")
        if not rationale or len(rationale) > 500:
            raise BlindError("verdict rationale must be short and non-empty")
        seen.add(packet_id)
        locked.append({"packet_id": packet_id, "choice": choice, "rationale": rationale})
    if seen != set(packet_ids):
        raise BlindError("all 8 packet verdicts must be locked together")
    payload = {"experiment_id": archive.experiment_id, "verdicts": sorted(locked, key=lambda row: row["packet_id"])}
    write_json_atomic(lock_path, payload)
    return {"locked_verdicts": len(locked), "path": archive.rel(lock_path)}


def _full_pass_records(archive: ExperimentArchive) -> list[dict[str, Any]]:
    records: list[dict[str, Any]] = []
    for file_id in archive.expected_file_ids():
        path = archive.root / "runs" / "full" / file_id / "record.json"
        if not path.is_file():
            raise BlindError("blind packets require 8 hard-gate pass records")
        record = read_json(path)
        try:
            validate_success_record_artifacts(archive, record, ["full"], file_id)
        except RunnerError as exc:
            raise BlindError("blind packets require valid full hard-gate pass records") from exc
        records.append(record)
    if len(records) != 8:
        raise BlindError("blind packets require exactly 8 pass records")
    return records


def validate_private_mapping_packets(archive: ExperimentArchive) -> dict[str, Any]:
    public_packet_ids = validate_public_packets(archive)
    mapping = read_json(archive.root / "blind" / "private-mapping.lock.json")
    if set(mapping) != {"experiment_id", "packet_count", "packet_ids", "mappings"}:
        raise BlindError("private mapping schema mismatch")
    if mapping.get("experiment_id") != archive.experiment_id:
        raise BlindError("private mapping experiment_id mismatch")
    if mapping.get("packet_count") != 8:
        raise BlindError("private mapping packet_count mismatch")
    packet_ids = mapping.get("packet_ids")
    mappings = mapping.get("mappings")
    if not isinstance(packet_ids, list) or len(packet_ids) != 8:
        raise BlindError("private mapping must contain exactly 8 packet ids")
    if not isinstance(mappings, list) or len(mappings) != 8:
        raise BlindError("private mapping must contain exactly 8 mapping rows")
    expected_packet_ids = [str(packet_id) for packet_id in packet_ids]
    if expected_packet_ids != public_packet_ids:
        raise BlindError("private mapping packet ids do not match public packets")
    expected_file_ids = dict(zip(public_packet_ids, archive.expected_file_ids(), strict=True))
    seen: set[str] = set()
    for row in mappings:
        if not isinstance(row, dict):
            raise BlindError("private mapping row must be an object")
        required = {
            "packet_id",
            "file_id",
            "source_filename",
            "original_slot",
            "candidate_slot",
            "packet_path",
            "packet_sha256",
        }
        if set(row) != required:
            raise BlindError("private mapping row schema mismatch")
        packet_id = str(row["packet_id"])
        if packet_id not in expected_packet_ids or packet_id in seen:
            raise BlindError("private mapping packet_id is missing, duplicate, or unexpected")
        file_id = str(row["file_id"])
        if file_id != expected_file_ids[packet_id]:
            raise BlindError("private mapping file_id mismatch")
        if row.get("source_filename") != archive.filename_for_file_id(file_id):
            raise BlindError("private mapping source_filename mismatch")
        if {row.get("original_slot"), row.get("candidate_slot")} != {"A", "B"}:
            raise BlindError("private mapping original and candidate slots must be opposite")
        expected_path = f"blind/packets/{packet_id}.json"
        if row.get("packet_path") != expected_path:
            raise BlindError("private mapping packet_path mismatch")
        packet_path = archive.root / expected_path
        if not packet_path.is_file() or config.sha256_file(packet_path) != row.get("packet_sha256"):
            raise BlindError("private mapping packet hash mismatch")
        seen.add(packet_id)
    if seen != set(expected_packet_ids):
        raise BlindError("private mapping packets do not match packet ids")
    return mapping


def validate_public_packets(archive: ExperimentArchive) -> list[str]:
    packet_dir = archive.root / "blind" / "packets"
    if not packet_dir.is_dir():
        raise BlindError("missing blind packet directory")
    packet_ids = [f"packet-{index:02d}" for index in range(1, 9)]
    expected_names = {f"{packet_id}.json" for packet_id in packet_ids}
    children = sorted(packet_dir.iterdir(), key=lambda path: path.name)
    if any(child.is_symlink() for child in children):
        raise BlindError("blind packets must not contain symlinks")
    actual_names = {child.name for child in children if child.is_file()}
    if len(actual_names) != len(children) or actual_names != expected_names:
        raise BlindError("blind packet file set mismatch")
    for packet_id in packet_ids:
        _validate_public_packet(packet_dir / f"{packet_id}.json", packet_id)
    return packet_ids


def _validate_public_packet(packet_path: Path, packet_id: str) -> None:
    packet = read_json(packet_path)
    if set(packet) != {"packet_id", "documents"} or packet.get("packet_id") != packet_id:
        raise BlindError("blind packet schema mismatch")
    documents = packet.get("documents")
    if not isinstance(documents, list) or len(documents) != 2:
        raise BlindError("blind packet must contain exactly two documents")
    slots: set[str] = set()
    for document in documents:
        if not isinstance(document, dict) or set(document) != {"slot", "body"}:
            raise BlindError("blind packet document schema mismatch")
        slot = document.get("slot")
        if slot not in {"A", "B"} or slot in slots or not isinstance(document.get("body"), str):
            raise BlindError("blind packet document slot/body mismatch")
        slots.add(str(slot))
    if slots != {"A", "B"}:
        raise BlindError("blind packet must contain A and B slots")
