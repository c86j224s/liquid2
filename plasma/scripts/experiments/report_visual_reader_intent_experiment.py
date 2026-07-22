#!/usr/bin/env python3
"""Issue #174 visual reader-intent experiment runner."""

from __future__ import annotations

import report_visual_reading_aid_preference_experiment as experiment


experiment.EXPERIMENT_ID = "29-report-visual-reader-intent-2026-07-22"
experiment.ARMS = ("visual_plan", "visual_reader_intent")
experiment.CANDIDATE_ARM = "visual_reader_intent"
experiment.PROFILE_BY_ARM = {
    "visual_plan": "visual-plan",
    "visual_reader_intent": "visual-reader-intent",
}


if __name__ == "__main__":
    raise SystemExit(experiment.main())
