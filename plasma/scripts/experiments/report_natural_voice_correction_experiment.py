#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
from pathlib import Path

from report_natural_voice_correction import config
from report_natural_voice_correction.archive import ExperimentArchive
from report_natural_voice_correction.blind import make_blind_packets, record_host_verdicts
from report_natural_voice_correction.codex_runner import RunnerError, freeze_prompt, lint_prompt_file, run_full, run_pilot
from report_natural_voice_correction.summary import export_public_summary


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Issue #207 sealed natural-voice correction experiment runner.")
    parser.add_argument("--action", required=True, choices=config.CLI_ACTIONS)
    parser.add_argument("--experiment", choices=config.EXPERIMENT_CHOICES, default="56")
    parser.add_argument("--archive", type=Path, default=None)
    parser.add_argument("--prompt", type=Path)
    parser.add_argument("--pilot-step", choices=("first", "second"), default="first")
    parser.add_argument("--verdicts", type=Path)
    parser.add_argument("--seed", type=int)
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    archive = ExperimentArchive.from_path(args.archive, args.experiment)
    result: object
    if args.action == "verify-source-seal":
        result = archive.verify_source_seal()
    elif args.action == "lint-prompt":
        _require(args.prompt, "--prompt is required for lint-prompt")
        result = lint_prompt_file(args.prompt)
    elif args.action == "freeze-prompt":
        _require(args.prompt, "--prompt is required for freeze-prompt")
        archive.verify_source_seal()
        result = freeze_prompt(archive, args.prompt)
    elif args.action == "run-pilot":
        archive.verify_source_seal()
        result = run_pilot(archive, args.prompt, args.pilot_step)
    elif args.action == "run-full":
        archive.verify_source_seal()
        if args.prompt is not None:
            raise RunnerError("run-full uses only the frozen prompt; --prompt is not allowed")
        result = run_full(archive)
    elif args.action == "make-blind-packets":
        result = make_blind_packets(archive, args.seed)
    elif args.action == "record-host-verdicts":
        _require(args.verdicts, "--verdicts is required for record-host-verdicts")
        result = record_host_verdicts(archive, args.verdicts)
    elif args.action == "export-public-summary":
        result = export_public_summary(archive)
    else:
        raise AssertionError(f"unhandled action: {args.action}")
    print(json.dumps(result, ensure_ascii=False, indent=2, sort_keys=True))
    return 0


def _require(value: object, message: str) -> None:
    if value is None:
        raise SystemExit(message)


if __name__ == "__main__":
    raise SystemExit(main())
