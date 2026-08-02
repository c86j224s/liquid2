from __future__ import annotations

from pathlib import Path
import random

from report_natural_voice_correction.archive import read_json, write_json_atomic

from . import config
from .archive import ExperimentArchive
from .records import RecordError, validate_record


class BlindError(ValueError):
    pass


MAGNITUDES = ("none", "slight", "clear", "large")


def make_blind_packets(
    archive: ExperimentArchive,
    seed: int | None = None,
) -> dict[str, object]:
    records = _full_records(archive)
    packet_dir = archive.root / "blind" / "packets"
    mapping_path = archive.root / "blind" / "private-mapping.lock.json"
    if mapping_path.exists() or packet_dir.exists():
        raise BlindError("blind packets already exist")
    packet_dir.mkdir(parents=True)
    rng = random.Random(seed) if seed is not None else random.SystemRandom()
    file_ids = list(archive.evaluation_file_ids())
    rng.shuffle(file_ids)
    mappings: list[dict[str, object]] = []
    for index, file_id in enumerate(file_ids, 1):
        packet_id = f"packet-{index:02d}"
        examples_slot = rng.choice(("A", "B"))
        control_slot = "B" if examples_slot == "A" else "A"
        bodies = {
            control_slot: _candidate(archive, records[(file_id, "control")]),
            examples_slot: _candidate(archive, records[(file_id, "examples")]),
        }
        packet_path = packet_dir / f"{packet_id}.json"
        write_json_atomic(packet_path, {
            "packet_id": packet_id,
            "documents": [
                {"slot": "A", "body": bodies["A"]},
                {"slot": "B", "body": bodies["B"]},
            ],
        })
        mappings.append({
            "packet_id": packet_id,
            "file_id": file_id,
            "control_slot": control_slot,
            "examples_slot": examples_slot,
            "packet_path": archive.rel(packet_path),
            "packet_sha256": config.sha256_file(packet_path),
        })
    mapping = {
        "experiment_id": config.EXPERIMENT_ID,
        "packet_count": len(mappings),
        "mappings": mappings,
    }
    write_json_atomic(mapping_path, mapping)
    return {
        "packet_count": len(mappings),
        "packet_ids": [row["packet_id"] for row in mappings],
        "mapping_path": archive.rel(mapping_path),
    }


def record_verdicts(archive: ExperimentArchive, verdicts_path: Path) -> dict[str, object]:
    lock_path = archive.root / "blind" / "host-verdicts.lock.json"
    if lock_path.exists():
        raise BlindError("host verdicts are already locked")
    packet_ids = validate_public_packets(archive)
    value = read_json(Path(verdicts_path))
    verdicts = value.get("verdicts")
    if not isinstance(verdicts, list):
        raise BlindError("verdict input must contain a verdicts array")
    locked: list[dict[str, str]] = []
    seen: set[str] = set()
    for row in verdicts:
        if not isinstance(row, dict) or set(row) != {"packet_id", "choice", "magnitude", "rationale"}:
            raise BlindError("verdict rows must contain packet_id, choice, magnitude, and rationale")
        packet_id = str(row["packet_id"])
        choice = str(row["choice"])
        magnitude = str(row["magnitude"])
        rationale = str(row["rationale"]).strip()
        if packet_id not in packet_ids or packet_id in seen:
            raise BlindError("verdict packet id is missing, duplicate, or unexpected")
        if choice not in {"A", "B", "tie"} or magnitude not in MAGNITUDES:
            raise BlindError("verdict choice or magnitude is invalid")
        if (choice == "tie") != (magnitude == "none"):
            raise BlindError("tie requires none; A/B choice requires slight, clear, or large")
        if not rationale or len(rationale) > 800:
            raise BlindError("verdict rationale must be short and non-empty")
        seen.add(packet_id)
        locked.append({
            "packet_id": packet_id,
            "choice": choice,
            "magnitude": magnitude,
            "rationale": rationale,
        })
    if seen != set(packet_ids):
        raise BlindError("all eight packet verdicts must be locked together")
    payload = {
        "experiment_id": config.EXPERIMENT_ID,
        "verdicts": sorted(locked, key=lambda row: row["packet_id"]),
    }
    write_json_atomic(lock_path, payload)
    validate_verdict_lock(archive)
    return {"locked_verdicts": len(locked), "path": archive.rel(lock_path)}


def validate_verdict_lock(archive: ExperimentArchive) -> dict[str, object]:
    packet_ids = validate_public_packets(archive)
    lock = read_json(archive.root / "blind" / "host-verdicts.lock.json")
    if set(lock) != {"experiment_id", "verdicts"} or lock.get("experiment_id") != config.EXPERIMENT_ID:
        raise BlindError("host verdict lock identity or schema mismatch")
    verdicts = lock.get("verdicts")
    if not isinstance(verdicts, list):
        raise BlindError("host verdict lock must contain a verdicts array")
    seen: set[str] = set()
    for row in verdicts:
        if not isinstance(row, dict) or set(row) != {"packet_id", "choice", "magnitude", "rationale"}:
            raise BlindError("host verdict lock row schema mismatch")
        packet_id = row["packet_id"]
        choice = row["choice"]
        magnitude = row["magnitude"]
        rationale = row["rationale"]
        if not isinstance(packet_id, str) or packet_id not in packet_ids or packet_id in seen:
            raise BlindError("host verdict lock packet coverage mismatch")
        if (
            not isinstance(choice, str)
            or not isinstance(magnitude, str)
            or choice not in {"A", "B", "tie"}
            or magnitude not in MAGNITUDES
        ):
            raise BlindError("host verdict lock choice or magnitude mismatch")
        if (choice == "tie") != (magnitude == "none"):
            raise BlindError("host verdict lock tie and magnitude mismatch")
        if not isinstance(rationale, str) or not rationale.strip() or len(rationale.strip()) > 800:
            raise BlindError("host verdict lock rationale mismatch")
        seen.add(packet_id)
    if seen != set(packet_ids):
        raise BlindError("host verdict lock must cover all eight packets")
    return lock


def validate_public_packets(archive: ExperimentArchive) -> list[str]:
    packet_dir = archive.root / "blind" / "packets"
    if not packet_dir.is_dir():
        raise BlindError("missing blind packet directory")
    packet_ids = [f"packet-{index:02d}" for index in range(1, 9)]
    expected = {f"{packet_id}.json" for packet_id in packet_ids}
    children = list(packet_dir.iterdir())
    if any(path.is_symlink() or not path.is_file() for path in children):
        raise BlindError("blind packet directory must contain regular files only")
    if {path.name for path in children} != expected:
        raise BlindError("blind packet file set mismatch")
    for packet_id in packet_ids:
        packet = read_json(packet_dir / f"{packet_id}.json")
        if set(packet) != {"packet_id", "documents"} or packet.get("packet_id") != packet_id:
            raise BlindError("blind packet schema mismatch")
        documents = packet.get("documents")
        if not isinstance(documents, list) or len(documents) != 2:
            raise BlindError("blind packet must contain two documents")
        if [row.get("slot") for row in documents if isinstance(row, dict)] != ["A", "B"]:
            raise BlindError("blind packet slots must be A then B")
        if any(not isinstance(row, dict) or set(row) != {"slot", "body"} or not isinstance(row["body"], str) for row in documents):
            raise BlindError("blind packet document schema mismatch")
    return packet_ids


def validate_private_mapping(archive: ExperimentArchive) -> dict[str, object]:
    packet_ids = validate_public_packets(archive)
    path = archive.root / "blind" / "private-mapping.lock.json"
    mapping = read_json(path)
    rows = mapping.get("mappings")
    if set(mapping) != {"experiment_id", "packet_count", "mappings"}:
        raise BlindError("private mapping schema mismatch")
    if mapping.get("experiment_id") != config.EXPERIMENT_ID or mapping.get("packet_count") != 8:
        raise BlindError("private mapping identity mismatch")
    if not isinstance(rows, list) or len(rows) != 8:
        raise BlindError("private mapping must contain eight rows")
    if [str(row.get("packet_id")) for row in rows if isinstance(row, dict)] != packet_ids:
        raise BlindError("private mapping packet order mismatch")
    if {str(row.get("file_id")) for row in rows if isinstance(row, dict)} != set(archive.evaluation_file_ids()):
        raise BlindError("private mapping evaluation corpus mismatch")
    for row in rows:
        if not isinstance(row, dict) or set(row) != {
            "packet_id", "file_id", "control_slot", "examples_slot", "packet_path", "packet_sha256"
        }:
            raise BlindError("private mapping row schema mismatch")
        if {row["control_slot"], row["examples_slot"]} != {"A", "B"}:
            raise BlindError("private mapping slots must be opposite")
        packet_path = archive.root / str(row["packet_path"])
        if not packet_path.is_file() or config.sha256_file(packet_path) != row["packet_sha256"]:
            raise BlindError("private mapping packet hash mismatch")
    return mapping


def _full_records(archive: ExperimentArchive) -> dict[tuple[str, str], dict[str, object]]:
    records: dict[tuple[str, str], dict[str, object]] = {}
    for file_id in archive.evaluation_file_ids():
        for arm in config.ARMS:
            path = archive.root / "runs" / "full" / file_id / arm / "record.json"
            if not path.is_file():
                raise BlindError("blind packets require all sixteen full-run records")
            record = read_json(path)
            try:
                validate_record(archive, record, "evaluation", ("full",), file_id, arm)
            except RecordError as exc:
                raise BlindError("blind packets require valid full-run records") from exc
            records[(file_id, arm)] = record
    return records


def _candidate(archive: ExperimentArchive, record: dict[str, object]) -> str:
    return (archive.root / str(record["candidate_path"])).read_text(encoding="utf-8")
