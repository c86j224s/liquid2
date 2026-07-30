#!/usr/bin/env python3
"""Issue #179 natural curiosity-voice follow-up experiment runner."""

from __future__ import annotations

import report_section_contract_experiment as experiment


experiment.EXPERIMENT_ID = "34-report-curiosity-natural-voice-2026-07-25"
experiment.ARMS = ("baseline", "curiosity_led_explanation", "curiosity_natural_voice")
experiment.PROFILE_BY_ARM = {
    "baseline": "narrative-contract",
    "curiosity_led_explanation": "curiosity-led-explanation",
    "curiosity_natural_voice": "curiosity-natural-voice",
}


if __name__ == "__main__":
    raise SystemExit(experiment.main())
