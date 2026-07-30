#!/usr/bin/env python3
"""Issue #189 subject-direct long-form synthesis experiment runner."""

from __future__ import annotations

import json
from pathlib import Path

import report_section_contract_experiment as experiment
import report_subject_direct_synthesis_metrics as stage_metrics


experiment.EXPERIMENT_ID = "39-report-subject-direct-synthesis-2026-07-27"
experiment.ARMS = ("current_default", "rich_control", "subject_direct_candidate")
experiment.PROFILE_BY_ARM = {
    "current_default": "part-connective-economy-voice",
    "rich_control": "section-brief-cluster-memory-narrative-contract",
    "subject_direct_candidate": "part-connective-subject-direct-synthesis-voice",
}

SELECTED_TOPICS = (
    "public-health-guidance-b",
    "consumer-finance-a",
    "labor-statistics-a",
    "climate-adaptation-a",
    "accessibility-a",
    "public-procurement-b",
)
_load_all_fixtures = experiment.base.load_fixtures


def load_selected_fixtures(archive: Path, limit: int) -> list[experiment.base.Fixture]:
    fixtures = _load_all_fixtures(archive, 10_000)
    by_topic = {fixture.topic: fixture for fixture in fixtures}
    missing = [topic for topic in SELECTED_TOPICS if topic not in by_topic]
    if missing:
        raise ValueError(f"selected fixtures are missing: {', '.join(missing)}")
    return [by_topic[topic] for topic in SELECTED_TOPICS[:limit]]


def analyze_with_subject_direct_metrics(args: object) -> None:
    archive = args.archive.expanduser().resolve()
    records = []
    for manifest_path in sorted((archive / "runs").glob("*/manifest.terminal.json")):
        manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
        if manifest.get("arm") not in experiment.ARMS:
            continue
        metrics_path = manifest_path.parent / "metrics.json"
        metrics = json.loads(metrics_path.read_text(encoding="utf-8")) if metrics_path.exists() else {}
        stage = stage_metrics.collect_stage_metrics(manifest_path.parent / "state/plasma.db", experiment.ratio) if manifest.get("status") == "completed" else {}
        records.append({"topic": manifest["topic"], "arm": manifest["arm"], "status": manifest.get("status"), "metrics": metrics, **stage})
    pairs = build_pairs(records)
    aggregate = {
        "experiment": experiment.EXPERIMENT_ID,
        "records": len(records),
        "paired_completed": len(pairs),
        "failures": [record for record in records if record.get("status") != "completed"],
        "pairs": pairs,
    }
    stage_result = {
        "experiment": experiment.EXPERIMENT_ID,
        "records": [record for record in records if record.get("status") == "completed"],
        "aggregate_by_arm": {
            arm: stage_metrics.aggregate_stage_metrics(
                [record for record in records if record["arm"] == arm and record.get("status") == "completed"],
                experiment.ratio,
            )
            for arm in experiment.ARMS
        },
    }
    experiment.base.write_json(archive / "analysis/aggregate.json", aggregate)
    experiment.base.write_json(archive / "analysis/stage-aggregate.json", stage_result)
    print(json.dumps(aggregate, indent=2, ensure_ascii=False))


def expected_matrix_status(archive: Path, limit: int) -> dict[str, object]:
    rows = []
    for topic in SELECTED_TOPICS[:limit]:
        for arm in experiment.ARMS:
            manifest_path = archive / "runs" / f"{topic}-{arm}" / "manifest.terminal.json"
            if not manifest_path.exists():
                rows.append({"topic": topic, "arm": arm, "state": "missing", "manifest": str(manifest_path)})
                continue
            manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
            status = manifest.get("status")
            state = "completed" if status == "completed" else "failed" if status == "failed" else "non_completed"
            rows.append({"topic": topic, "arm": arm, "state": state, "status": status, "manifest": str(manifest_path)})
    return {
        "expected": len(SELECTED_TOPICS[:limit]) * len(experiment.ARMS),
        "completed": [row for row in rows if row["state"] == "completed"],
        "missing": [row for row in rows if row["state"] == "missing"],
        "non_completed": [row for row in rows if row["state"] == "non_completed"],
        "failed": [row for row in rows if row["state"] == "failed"],
        "rows": rows,
    }


def assert_packet_matrix_completed(archive: Path, limit: int) -> dict[str, object]:
    status = expected_matrix_status(archive, limit)
    blockers = status["missing"] or status["non_completed"] or status["failed"]
    if blockers:
        experiment.base.write_json(archive / "judging/packet-matrix-status.json", status)
        raise RuntimeError(
            "refusing packet generation until expected matrix is completed: "
            f"missing={len(status['missing'])} non_completed={len(status['non_completed'])} failed={len(status['failed'])}"
        )
    return status


def build_pairs(records: list[dict[str, object]]) -> list[dict[str, object]]:
    by_topic: dict[str, dict[str, dict[str, object]]] = {}
    for record in records:
        by_topic.setdefault(str(record["topic"]), {})[str(record["arm"])] = record
    pairs = []
    controls = ("rich_control", "subject_direct_candidate")
    for topic, arms in sorted(by_topic.items()):
        if all(arm in arms and arms[arm].get("status") == "completed" for arm in experiment.ARMS):
            current = arms["current_default"]
            pair = {
                "topic": topic,
                "current_default_words": current["metrics"].get("final_word_count"),
                "current_default_parts": current["metrics"].get("part_count"),
                "current_default_sections": current["metrics"].get("section_count"),
                "current_default_wall_seconds": current["metrics"].get("wall_seconds"),
                "comparisons": {},
            }
            for arm in controls:
                candidate = arms[arm]
                pair["comparisons"][arm] = {
                    "words": candidate["metrics"].get("final_word_count"),
                    "parts": candidate["metrics"].get("part_count"),
                    "sections": candidate["metrics"].get("section_count"),
                    "wall_seconds": candidate["metrics"].get("wall_seconds"),
                    "word_ratio_over_current_default": experiment.ratio(candidate["metrics"].get("final_word_count"), current["metrics"].get("final_word_count")),
                    "section_ratio_over_current_default": experiment.ratio(candidate["metrics"].get("section_count"), current["metrics"].get("section_count")),
                }
            pairs.append(pair)
    return pairs


def packets_subject_direct(args: object) -> None:
    archive = args.archive.expanduser().resolve()
    matrix_status = assert_packet_matrix_completed(archive, args.limit)
    out = archive / "judging/packets"
    out.mkdir(parents=True, exist_ok=True)
    mapping = {}
    rng = experiment.random.Random(args.seed)
    for topic in SELECTED_TOPICS[: args.limit]:
        for candidate_arm in ("rich_control", "subject_direct_candidate"):
            labels = ["current_default", candidate_arm]
            rng.shuffle(labels)
            packet = {
                "packet_id": f"{experiment.EXPERIMENT_ID}-{topic}-{candidate_arm}",
                "topic": topic,
                "candidate_arm": candidate_arm,
                "replicate": 1,
                "mode": "long_form",
            }
            for label, arm in zip(("A", "B"), labels):
                report = (archive / "runs" / f"{topic}-{arm}" / "report.md").read_text(encoding="utf-8")
                packet[label] = {"report_markdown": report}
                mapping[f"{topic}:{candidate_arm}:{label}"] = arm
            experiment.base.write_json(out / f"{topic}-{candidate_arm}.json", packet)
    experiment.base.write_json(archive / "judging/private-mapping.json", mapping)
    experiment.base.write_json(archive / "judging/packet-matrix-status.json", matrix_status)
    print(json.dumps({"packets": len(mapping) // 2, "path": str(out)}, ensure_ascii=False))


experiment.base.load_fixtures = load_selected_fixtures
experiment.analyze = analyze_with_subject_direct_metrics
experiment.packets = packets_subject_direct


if __name__ == "__main__":
    raise SystemExit(experiment.main())
