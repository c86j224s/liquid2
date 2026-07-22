#!/usr/bin/env python3
"""Issue #174 visual affordance-priming experiment runner."""

from __future__ import annotations

import report_visual_reading_aid_preference_experiment as experiment


experiment.EXPERIMENT_ID = "31-report-visual-affordance-priming-2026-07-22"
experiment.ARMS = ("visual_plan", "visual_affordance_priming")
experiment.CANDIDATE_ARM = "visual_affordance_priming"
experiment.PROFILE_BY_ARM = {
    "visual_plan": "visual-plan",
    "visual_affordance_priming": "visual-affordance-priming",
}


if __name__ == "__main__":
    raise SystemExit(experiment.main())
