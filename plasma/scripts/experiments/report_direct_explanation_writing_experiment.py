#!/usr/bin/env python3
"""Issue #179 direct-explanation writing experiment runner."""

from __future__ import annotations

import report_section_contract_experiment as experiment


experiment.EXPERIMENT_ID = "33-report-direct-explanation-writing-2026-07-24"
experiment.ARMS = ("baseline", "reader_paragraph_contract", "curiosity_led_explanation")
experiment.PROFILE_BY_ARM = {
    "baseline": "narrative-contract",
    "reader_paragraph_contract": "reader-paragraph-contract",
    "curiosity_led_explanation": "curiosity-led-explanation",
}


if __name__ == "__main__":
    raise SystemExit(experiment.main())
