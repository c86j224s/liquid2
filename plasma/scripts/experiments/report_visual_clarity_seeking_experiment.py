#!/usr/bin/env python3
"""Issue #174 visual clarity-seeking experiment runner."""

from __future__ import annotations

import report_visual_reading_aid_preference_experiment as experiment


experiment.EXPERIMENT_ID = "30-report-visual-clarity-seeking-2026-07-22"
experiment.ARMS = ("visual_plan", "visual_clarity_seeking")
experiment.CANDIDATE_ARM = "visual_clarity_seeking"
experiment.PROFILE_BY_ARM = {
    "visual_plan": "visual-plan",
    "visual_clarity_seeking": "visual-clarity-seeking",
}


if __name__ == "__main__":
    raise SystemExit(experiment.main())
