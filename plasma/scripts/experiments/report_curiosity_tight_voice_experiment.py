#!/usr/bin/env python3
"""Issue #179 tight curiosity-voice follow-up experiment runner."""

from __future__ import annotations

import report_section_contract_experiment as experiment


experiment.EXPERIMENT_ID = "35-report-curiosity-tight-voice-2026-07-25"
experiment.ARMS = ("baseline", "curiosity_led_explanation", "curiosity_natural_voice", "curiosity_tight_voice")
experiment.PROFILE_BY_ARM = {
    "baseline": "narrative-contract",
    "curiosity_led_explanation": "curiosity-led-explanation",
    "curiosity_natural_voice": "curiosity-natural-voice",
    "curiosity_tight_voice": "curiosity-tight-voice",
}


if __name__ == "__main__":
    raise SystemExit(experiment.main())
