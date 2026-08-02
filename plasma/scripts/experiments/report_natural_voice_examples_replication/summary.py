from __future__ import annotations

from report_natural_voice_correction.archive import write_json_atomic
from report_natural_voice_examples.archive import ExperimentArchive
from report_natural_voice_examples.blind import MAGNITUDES, validate_private_mapping, validate_verdict_lock
from report_natural_voice_examples.review import DRIFT_KEYS, validate_semantic_audit

from . import config
from .context import require_active


class SummaryError(ValueError):
    pass


MAGNITUDE_SCORES = {"none": 0, "slight": 1, "clear": 2, "large": 3}


def export_summary(archive: ExperimentArchive) -> dict[str, object]:
    require_active()
    path = archive.root / "analysis" / "public-summary.json"
    if path.exists():
        raise SummaryError("public summary already exists")
    mapping = validate_private_mapping(archive)
    verdict_lock = validate_verdict_lock(archive)
    audit_lock = validate_semantic_audit(archive)
    verdicts = verdict_lock.get("verdicts")
    rows = mapping.get("mappings")
    if not isinstance(verdicts, list) or not isinstance(rows, list):
        raise SummaryError("mapping or verdict rows missing")

    by_packet = {str(row["packet_id"]): row for row in rows if isinstance(row, dict)}
    counts = {"examples_wins": 0, "control_wins": 0, "ties": 0}
    strong = {"examples_wins": 0, "control_wins": 0}
    signed_magnitude_score = 0
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
        elif choice == row["examples_slot"]:
            winner = "examples"
            counts["examples_wins"] += 1
            signed_magnitude_score += MAGNITUDE_SCORES[magnitude]
            if magnitude in {"clear", "large"}:
                strong["examples_wins"] += 1
        else:
            winner = "control"
            counts["control_wins"] += 1
            signed_magnitude_score -= MAGNITUDE_SCORES[magnitude]
            if magnitude in {"clear", "large"}:
                strong["control_wins"] += 1
        decoded.append({
            "packet_id": packet_id,
            "file_id": str(row["file_id"]),
            "winner": winner,
            "magnitude": magnitude,
            "rationale": str(verdict["rationale"]),
        })
    if sum(counts.values()) != len(config.EVALUATION_SOURCES):
        raise SummaryError("summary verdict count does not match the evaluation corpus")

    drift_by_arm = drift_counts_by_arm(audit_lock)
    reading = {
        **counts,
        "clear_or_large_examples_wins": strong["examples_wins"],
        "clear_or_large_control_wins": strong["control_wins"],
        "signed_magnitude_score": signed_magnitude_score,
        "classification": _reading_classification(counts, signed_magnitude_score),
        "population_inference": "not_evaluated",
    }
    safety = {
        arm: {
            **counts_for_arm,
            "status": "no_drift_observed" if not any(counts_for_arm.values()) else "drift_observed",
        }
        for arm, counts_for_arm in drift_by_arm.items()
    }
    summary = {
        "experiment_id": config.EXPERIMENT_ID,
        "reading_efficacy": reading,
        "semantic_safety": safety,
        "product_readiness": {
            "status": "not_evaluated",
            "reason": "fresh-corpus prompt replication does not exercise the product path or user adoption boundary",
        },
        "interpretation_rules": config.INTERPRETATION_RULES,
        "decoded_verdicts": sorted(decoded, key=lambda row: row["packet_id"]),
    }
    write_json_atomic(path, summary)
    return summary


def drift_counts_by_arm(audit_lock: dict[str, object]) -> dict[str, dict[str, int]]:
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


def _reading_classification(counts: dict[str, int], score: int) -> str:
    if counts["examples_wins"] > counts["control_wins"] and score > 0:
        return "directional_support"
    if counts["control_wins"] > counts["examples_wins"] and score < 0:
        return "directional_contradiction"
    return "mixed"
