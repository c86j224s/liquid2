from __future__ import annotations

from report_natural_voice_correction.archive import write_json_atomic

from . import config
from .archive import ExperimentArchive
from .blind import MAGNITUDES, validate_private_mapping, validate_verdict_lock
from .review import DRIFT_KEYS, validate_semantic_audit


class SummaryError(ValueError):
    pass


def export_summary(archive: ExperimentArchive) -> dict[str, object]:
    path = archive.root / "analysis" / "public-summary.json"
    if path.exists():
        raise SummaryError("public summary already exists")
    mapping = validate_private_mapping(archive)
    verdict_lock = validate_verdict_lock(archive)
    audit_lock = validate_semantic_audit(archive)
    verdicts = verdict_lock.get("verdicts")
    rows = mapping.get("mappings")
    if verdict_lock.get("experiment_id") != config.EXPERIMENT_ID:
        raise SummaryError("verdict identity mismatch")
    if not isinstance(verdicts, list) or not isinstance(rows, list):
        raise SummaryError("mapping or verdict rows missing")
    by_packet = {str(row["packet_id"]): row for row in rows if isinstance(row, dict)}
    counts = {"examples_wins": 0, "examples_losses": 0, "ties": 0}
    clear_or_large_examples_wins = 0
    clear_or_large_examples_losses = 0
    decoded: list[dict[str, str]] = []
    for verdict in verdicts:
        if not isinstance(verdict, dict) or str(verdict.get("packet_id")) not in by_packet:
            raise SummaryError("verdict references unknown packet")
        packet_id = str(verdict["packet_id"])
        choice = str(verdict["choice"])
        magnitude = str(verdict["magnitude"])
        row = by_packet[packet_id]
        if magnitude not in MAGNITUDES:
            raise SummaryError("verdict magnitude changed")
        if choice == "tie":
            winner = "tie"
            counts["ties"] += 1
            counts["examples_losses"] += 1
        elif choice == row["examples_slot"]:
            winner = "examples"
            counts["examples_wins"] += 1
            if magnitude in {"clear", "large"}:
                clear_or_large_examples_wins += 1
        else:
            winner = "control"
            counts["examples_losses"] += 1
            if magnitude in {"clear", "large"}:
                clear_or_large_examples_losses += 1
        decoded.append({
            "packet_id": packet_id,
            "file_id": str(row["file_id"]),
            "winner": winner,
            "magnitude": magnitude,
            "rationale": str(verdict["rationale"]),
        })
    if counts["examples_wins"] + counts["examples_losses"] != 8:
        raise SummaryError("summary requires eight verdicts")
    drift_counts = _drift_counts(audit_lock)
    criteria = config.SUCCESS_CRITERIA
    gates = {
        "preference": counts["examples_wins"] >= criteria["minimum_example_wins"],
        "clear_or_large_wins": clear_or_large_examples_wins >= criteria["minimum_clear_or_large_example_wins"],
        "clear_or_large_losses": clear_or_large_examples_losses <= criteria["maximum_clear_or_large_example_losses"],
        "semantic_preservation": drift_counts["semantic_drift"] <= criteria["maximum_semantic_drift"],
        "claim_scope_preservation": drift_counts["claim_scope_drift"] <= criteria["maximum_claim_scope_drift"],
        "citation_preservation": drift_counts["citation_drift"] <= criteria["maximum_citation_drift"],
    }
    summary = {
        "experiment_id": config.EXPERIMENT_ID,
        **counts,
        "clear_or_large_examples_wins": clear_or_large_examples_wins,
        "clear_or_large_examples_losses": clear_or_large_examples_losses,
        **drift_counts,
        "success_criteria": criteria,
        "gates": gates,
        "experiment_passed": all(gates.values()),
        "decoded_verdicts": sorted(decoded, key=lambda row: row["packet_id"]),
    }
    write_json_atomic(path, summary)
    return summary


def _drift_counts(audit_lock: dict[str, object]) -> dict[str, int]:
    rows = audit_lock.get("audits")
    if not isinstance(rows, list):
        raise SummaryError("semantic audit rows missing")
    return {
        key.removesuffix("_lines"): sum(len(row[key]) for row in rows if isinstance(row, dict))
        for key in DRIFT_KEYS
    }
