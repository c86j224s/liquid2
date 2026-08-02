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
    counts = {"contrastive_wins": 0, "control_wins": 0, "ties": 0}
    clear_or_large_contrastive_wins = 0
    clear_or_large_contrastive_losses = 0
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
        elif choice == row["contrastive_slot"]:
            winner = "contrastive"
            counts["contrastive_wins"] += 1
            if magnitude in {"clear", "large"}:
                clear_or_large_contrastive_wins += 1
        else:
            winner = "control"
            counts["control_wins"] += 1
            if magnitude in {"clear", "large"}:
                clear_or_large_contrastive_losses += 1
        decoded.append({
            "packet_id": packet_id,
            "file_id": str(row["file_id"]),
            "winner": winner,
            "magnitude": magnitude,
            "rationale": str(verdict["rationale"]),
        })
    if sum(counts.values()) != 8:
        raise SummaryError("summary requires eight verdicts")

    drift_by_arm = _drift_counts_by_arm(audit_lock)
    candidate_drift = drift_by_arm["contrastive"]
    criteria = config.SUCCESS_CRITERIA
    gates = {
        "preference": counts["contrastive_wins"] >= criteria["minimum_contrastive_wins"],
        "clear_or_large_wins": clear_or_large_contrastive_wins
        >= criteria["minimum_clear_or_large_contrastive_wins"],
        "clear_or_large_losses": clear_or_large_contrastive_losses
        <= criteria["maximum_clear_or_large_contrastive_losses"],
        "semantic_preservation": candidate_drift["semantic_drift"]
        <= criteria["maximum_contrastive_semantic_drift"],
        "claim_scope_preservation": candidate_drift["claim_scope_drift"]
        <= criteria["maximum_contrastive_claim_scope_drift"],
        "citation_preservation": candidate_drift["citation_drift"]
        <= criteria["maximum_contrastive_citation_drift"],
    }
    summary = {
        "experiment_id": config.EXPERIMENT_ID,
        **counts,
        "clear_or_large_contrastive_wins": clear_or_large_contrastive_wins,
        "clear_or_large_contrastive_losses": clear_or_large_contrastive_losses,
        "drift_by_arm": drift_by_arm,
        "success_criteria": criteria,
        "gates": gates,
        "screening_passed": all(gates.values()),
        "product_adoption_evaluated": False,
        "decoded_verdicts": sorted(decoded, key=lambda row: row["packet_id"]),
    }
    write_json_atomic(path, summary)
    return summary


def _drift_counts_by_arm(audit_lock: dict[str, object]) -> dict[str, dict[str, int]]:
    rows = audit_lock.get("audits")
    if not isinstance(rows, list):
        raise SummaryError("semantic audit rows missing")
    counts = {
        arm: {key.removesuffix("_lines"): 0 for key in DRIFT_KEYS}
        for arm in config.ARMS
    }
    for row in rows:
        if not isinstance(row, dict) or row.get("arm") not in counts:
            raise SummaryError("semantic audit arm mismatch")
        arm_counts = counts[str(row["arm"])]
        for key in DRIFT_KEYS:
            lines = row.get(key)
            if not isinstance(lines, list):
                raise SummaryError("semantic audit drift lines missing")
            arm_counts[key.removesuffix("_lines")] += len(lines)
    return counts
