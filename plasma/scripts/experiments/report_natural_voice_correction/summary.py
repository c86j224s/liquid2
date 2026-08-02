from __future__ import annotations

from . import config
from .archive import ExperimentArchive, read_json, write_json_atomic
from .blind import validate_private_mapping_packets


class SummaryError(ValueError):
    pass


def export_public_summary(archive: ExperimentArchive) -> dict[str, object]:
    mapping = validate_private_mapping_packets(archive)
    verdict_lock = read_json(archive.root / "blind" / "host-verdicts.lock.json")
    if mapping.get("experiment_id") != archive.experiment_id:
        raise SummaryError("private mapping experiment_id mismatch")
    if verdict_lock.get("experiment_id") != archive.experiment_id:
        raise SummaryError("host verdict lock experiment_id mismatch")
    mappings = mapping.get("mappings")
    verdicts = verdict_lock.get("verdicts")
    if not isinstance(mappings, list) or not isinstance(verdicts, list):
        raise SummaryError("mapping and verdict locks must contain arrays")
    candidate_slot_by_packet = {
        str(row["packet_id"]): str(row["candidate_slot"])
        for row in mappings
        if isinstance(row, dict) and "packet_id" in row and "candidate_slot" in row
    }
    if len(candidate_slot_by_packet) != 8:
        raise SummaryError("private mapping must contain 8 candidate slots")
    candidate_wins = 0
    candidate_losses = 0
    ties = 0
    opaque_rationales: list[dict[str, str]] = []
    for verdict in verdicts:
        if not isinstance(verdict, dict):
            raise SummaryError("verdict must be an object")
        packet_id = str(verdict["packet_id"])
        choice = str(verdict["choice"])
        rationale = str(verdict["rationale"])
        if packet_id not in candidate_slot_by_packet:
            raise SummaryError("verdict references an unknown packet")
        if choice == "tie":
            ties += 1
            candidate_losses += 1
        elif choice == candidate_slot_by_packet[packet_id]:
            candidate_wins += 1
        else:
            candidate_losses += 1
        opaque_rationales.append({"packet_id": packet_id, "choice": choice, "rationale": rationale})
    if candidate_wins + candidate_losses != 8:
        raise SummaryError("summary requires exactly 8 locked verdicts")
    summary = {
        "experiment_id": archive.experiment_id,
        "packet_count": 8,
        "candidate_wins": candidate_wins,
        "candidate_losses": candidate_losses,
        "ties": ties,
        "ties_count_as": "candidate_loss",
        "adoption_threshold": "candidate_wins >= 7 and zero semantic/citation drift",
        "preference_gate_passed": candidate_wins >= 7,
        "opaque_rationales": sorted(opaque_rationales, key=lambda row: row["packet_id"]),
    }
    write_json_atomic(archive.root / "analysis" / "public-summary.json", summary)
    return summary
