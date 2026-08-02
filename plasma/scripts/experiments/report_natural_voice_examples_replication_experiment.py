#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
from pathlib import Path

from report_natural_voice_examples.archive import ExperimentArchive
from report_natural_voice_examples.blind import make_blind_packets, record_verdicts
from report_natural_voice_examples.review import make_semantic_audit_pack, record_semantic_audit
from report_natural_voice_examples.runner import authorize_full, run_full, run_pilots
from report_natural_voice_examples_replication.context import activated
from report_natural_voice_examples_replication.hash_contract import lock_protocol_amendment
from report_natural_voice_examples_replication.prompts import freeze_protocol
from report_natural_voice_examples_replication.recovery import resume_failed_pilots
from report_natural_voice_examples_replication.summary import export_summary


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Run fresh-corpus replication experiment 60")
    parser.add_argument("action", choices=(
        "prepare", "freeze", "run-pilots", "authorize-full", "run-full",
        "amend-hash-contract", "resume-pilots",
        "make-blind-packets", "lock-verdicts", "make-audit-pack", "lock-audit",
        "summarize", "verify",
    ))
    parser.add_argument("--archive", type=Path)
    parser.add_argument("--input", type=Path)
    parser.add_argument("--workers", type=int, default=2)
    parser.add_argument("--seed", type=int)
    return parser.parse_args()


def main() -> None:
    args = parse_args()
    with activated():
        archive = ExperimentArchive.from_path(args.archive)
        if args.action == "prepare":
            result = archive.prepare()
        elif args.action == "freeze":
            result = freeze_protocol(archive)
        elif args.action == "run-pilots":
            result = run_pilots(archive, workers=args.workers)
        elif args.action == "amend-hash-contract":
            result = lock_protocol_amendment(archive)
        elif args.action == "resume-pilots":
            result = resume_failed_pilots(archive)
        elif args.action == "authorize-full":
            result = authorize_full(archive)
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


if __name__ == "__main__":
    main()
