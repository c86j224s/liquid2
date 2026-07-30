#!/usr/bin/env python3
"""Issue #179 Part connective-economy follow-up experiment runner."""

from __future__ import annotations

import json
from pathlib import Path
import re
import sqlite3

import report_section_contract_experiment as experiment


experiment.EXPERIMENT_ID = "38-report-part-connective-economy-2026-07-26"
experiment.ARMS = ("baseline", "part_connective_economy_voice")
experiment.PROFILE_BY_ARM = {
    "baseline": "section-direct-reading-voice",
    "part_connective_economy_voice": "part-connective-economy-voice",
}

SELECTED_TOPICS = (
    "public-health-guidance-b",
    "consumer-finance-a",
    "labor-statistics-a",
    "climate-adaptation-a",
    "accessibility-a",
    "public-procurement-b",
)
POSITION_MARKERS = (
    "이 부는",
    "이 부에서는",
    "이 부에서",
    "이 부의",
    "이 절은",
    "이 절에서는",
    "이 절에서",
    "이 절의",
    "다음 질문",
    "다음 절",
    "앞 절",
    "앞에서",
    "뒤에서",
    "이 보고서",
)
SECTION_FILENAME = re.compile(r"-part-\d+-section-\d+\.md$")
PART_FILENAME = re.compile(r"-part-\d+\.md$")

_load_all_fixtures = experiment.base.load_fixtures
_analyze_reports = experiment.analyze


def load_selected_fixtures(archive: Path, limit: int) -> list[experiment.base.Fixture]:
    fixtures = _load_all_fixtures(archive, 10_000)
    by_topic = {fixture.topic: fixture for fixture in fixtures}
    missing = [topic for topic in SELECTED_TOPICS if topic not in by_topic]
    if missing:
        raise ValueError(f"selected fixtures are missing: {', '.join(missing)}")
    return [by_topic[topic] for topic in SELECTED_TOPICS[:limit]]


def analyze_with_stage_metrics(args: object) -> None:
    _analyze_reports(args)
    archive = args.archive.expanduser().resolve()
    records = []
    for manifest_path in sorted((archive / "runs").glob("*/manifest.terminal.json")):
        manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
        if manifest.get("status") != "completed" or manifest.get("arm") not in experiment.ARMS:
            continue
        metrics = collect_stage_metrics(manifest_path.parent / "state/plasma.db")
        records.append({"topic": manifest["topic"], "arm": manifest["arm"], **metrics})
    result = {
        "experiment": experiment.EXPERIMENT_ID,
        "records": records,
        "aggregate_by_arm": {
            arm: aggregate_stage_metrics([record for record in records if record["arm"] == arm])
            for arm in experiment.ARMS
        },
    }
    experiment.base.write_json(archive / "analysis/stage-aggregate.json", result)
    print(json.dumps(result, indent=2, ensure_ascii=False))


def collect_stage_metrics(database: Path) -> dict[str, object]:
    stages = {
        "section": {"artifacts": 0, "characters": 0, "position_markers": 0},
        "part": {"artifacts": 0, "characters": 0, "position_markers": 0},
        "final": {"artifacts": 0, "characters": 0, "position_markers": 0},
    }
    with sqlite3.connect(database) as connection:
        rows = connection.execute("select filename, content_blob from plasma_raw_artifacts").fetchall()
    for filename, blob in rows:
        stage = artifact_stage(str(filename))
        text = bytes(blob).decode("utf-8")
        stages[stage]["artifacts"] += 1
        stages[stage]["characters"] += len(text)
        stages[stage]["position_markers"] += sum(text.count(marker) for marker in POSITION_MARKERS)
    section_chars = stages["section"]["characters"]
    connective_chars = stages["part"]["characters"] - section_chars
    return {
        "stages": stages,
        "part_connective_characters": connective_chars,
        "part_connective_ratio": experiment.ratio(connective_chars, section_chars),
        "part_introduced_position_markers": stages["part"]["position_markers"] - stages["section"]["position_markers"],
        "final_character_delta": stages["final"]["characters"] - stages["part"]["characters"],
    }


def artifact_stage(filename: str) -> str:
    if SECTION_FILENAME.search(filename):
        return "section"
    if PART_FILENAME.search(filename):
        return "part"
    return "final"


def aggregate_stage_metrics(records: list[dict[str, object]]) -> dict[str, object]:
    section_chars = sum(record["stages"]["section"]["characters"] for record in records)
    part_chars = sum(record["stages"]["part"]["characters"] for record in records)
    return {
        "completed": len(records),
        "section_characters": section_chars,
        "part_characters": part_chars,
        "part_connective_characters": part_chars - section_chars,
        "part_connective_ratio": experiment.ratio(part_chars - section_chars, section_chars),
        "section_position_markers": sum(record["stages"]["section"]["position_markers"] for record in records),
        "part_position_markers": sum(record["stages"]["part"]["position_markers"] for record in records),
        "part_introduced_position_markers": sum(record["part_introduced_position_markers"] for record in records),
    }


experiment.base.load_fixtures = load_selected_fixtures
experiment.analyze = analyze_with_stage_metrics


if __name__ == "__main__":
    raise SystemExit(experiment.main())
