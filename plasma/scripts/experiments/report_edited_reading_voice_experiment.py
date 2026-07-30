#!/usr/bin/env python3
"""Issue #179 edited reading-voice follow-up experiment runner."""

from __future__ import annotations

import report_section_contract_experiment as experiment


experiment.EXPERIMENT_ID = "36-report-edited-reading-voice-2026-07-25"
experiment.ARMS = ("baseline", "curiosity_natural_voice", "curiosity_tight_voice", "edited_reading_voice")
experiment.PROFILE_BY_ARM = {
    "baseline": "narrative-contract",
    "curiosity_natural_voice": "curiosity-natural-voice",
    "curiosity_tight_voice": "curiosity-tight-voice",
    "edited_reading_voice": "edited-reading-voice",
}


if __name__ == "__main__":
    raise SystemExit(experiment.main())
