#!/usr/bin/env python3
"""Issue #179 section-direct reading follow-up experiment runner."""

from __future__ import annotations

from pathlib import Path

import report_section_contract_experiment as experiment


experiment.EXPERIMENT_ID = "37-report-section-direct-reading-2026-07-25"
experiment.ARMS = ("baseline", "section_direct_reading_voice")
experiment.PROFILE_BY_ARM = {
    "baseline": "edited-reading-voice",
    "section_direct_reading_voice": "section-direct-reading-voice",
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


experiment.base.load_fixtures = load_selected_fixtures


if __name__ == "__main__":
    raise SystemExit(experiment.main())
