#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
from pathlib import Path

from report_natural_voice_contrastive_examples.archive import ExperimentArchive
from report_natural_voice_contrastive_examples.blind import make_blind_packets, record_verdicts
from report_natural_voice_contrastive_examples.prompts import freeze_protocol, lint_contrastive_prompt
from report_natural_voice_contrastive_examples.recovery import (
    resume_full_contract,
    resume_pilot_contract,
)
from report_natural_voice_contrastive_examples.review import make_semantic_audit_pack, record_semantic_audit
from report_natural_voice_contrastive_examples.runner import authorize_full, run_calibration, run_full, run_pilots
from report_natural_voice_contrastive_examples.summary import export_summary


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Run sealed natural-voice contrastive example experiment 59")
    parser.add_argument("action", choices=(
        "prepare", "lint-prompt", "calibrate", "freeze", "run-pilots", "authorize-full",
        "resume-pilot-contract", "resume-full-contract", "run-full",
        "make-blind-packets", "lock-verdicts",
        "make-audit-pack", "lock-audit", "summarize", "verify",
    ))
    parser.add_argument("--archive", type=Path)
    parser.add_argument("--prompt", type=Path)
    parser.add_argument("--input", type=Path)
    parser.add_argument("--file-id")
    parser.add_argument("--arm", choices=("control", "contrastive"))
    parser.add_argument("--workers", type=int, default=2)
    parser.add_argument("--seed", type=int)
    return parser.parse_args()


def main() -> None:
    args = parse_args()
    archive = ExperimentArchive.from_path(args.archive)
    if args.action == "prepare":
        result = archive.prepare()
    elif args.action == "lint-prompt":
        result = lint_contrastive_prompt(archive, _required(args.prompt, "--prompt"))
    elif args.action == "calibrate":
        result = run_calibration(archive, _required(args.prompt, "--prompt"))
    elif args.action == "freeze":
        result = freeze_protocol(archive, _required(args.prompt, "--prompt"))
    elif args.action == "run-pilots":
        result = run_pilots(archive, workers=args.workers)
    elif args.action == "authorize-full":
        result = authorize_full(archive)
    elif args.action == "resume-pilot-contract":
        result = resume_pilot_contract(
            archive,
            _required_text(args.file_id, "--file-id"),
            _required_text(args.arm, "--arm"),
        )
    elif args.action == "resume-full-contract":
        result = resume_full_contract(
            archive,
            _required_text(args.file_id, "--file-id"),
            _required_text(args.arm, "--arm"),
        )
    elif args.action == "run-full":
        result = run_full(archive, workers=args.workers)
    elif args.action == "make-blind-packets":
        result = make_blind_packets(archive, seed=args.seed)
    elif args.action == "lock-verdicts":
        result = record_verdicts(archive, _required(args.input, "--input"))
    elif args.action == "make-audit-pack":
        result = make_semantic_audit_pack(archive)
    elif args.action == "lock-audit":
        result = record_semantic_audit(archive, _required(args.input, "--input"))
    elif args.action == "summarize":
        result = export_summary(archive)
    else:
        result = archive.verify_source_seal()
    print(json.dumps(result, ensure_ascii=False, indent=2, sort_keys=True))


def _required(value: Path | None, flag: str) -> Path:
    if value is None:
        raise SystemExit(f"{flag} is required for this action")
    return value


def _required_text(value: str | None, flag: str) -> str:
    if value is None:
        raise SystemExit(f"{flag} is required for this action")
    return value


if __name__ == "__main__":
    main()
