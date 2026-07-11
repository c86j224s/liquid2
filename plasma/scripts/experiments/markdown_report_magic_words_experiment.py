#!/usr/bin/env python3
"""Run the isolated issue 77 Markdown report prompt experiment."""

from __future__ import annotations

import argparse
import concurrent.futures
import hashlib
import json
import shutil
from collections import Counter, defaultdict
from pathlib import Path
from typing import Any

from markdown_report_magic_words_metrics import aggregate_metrics, report_metrics, write_csv
from markdown_report_magic_words_protocol import (
    EXPERIMENT_ID,
    JUDGE_AXES,
    JUDGE_SCHEMA,
    MODES,
    PAIRINGS,
    SAMPLES,
    VARIANTS,
    judge_prompt,
    report_prompt,
)
from markdown_report_magic_words_runner import (
    DEFAULT_ARCHIVE,
    SafetyError,
    cached_prompt_matches,
    codex_version,
    ensure_archive,
    run_codex,
    sha256_file,
    write_json,
)


DEFAULT_SOURCE_ARCHIVE = (
    Path.home()
    / "research-artifacts/liquid2/plasma/experiments/10-generation-time-tone-2026-07-07/sources"
)


def select(values: tuple[Any, ...], names: list[str] | None) -> list[Any]:
    if not names:
        return list(values)

    def identifier(value: Any) -> str:
        return getattr(value, "sample_id", getattr(value, "name", ""))

    selected = [value for value in values if identifier(value) in names]
    if len(selected) != len(set(names)):
        known = sorted(identifier(value) for value in values)
        raise SystemExit(f"unknown selection in {names}; known values: {known}")
    return selected


def validate_source_archive(source_archive: Path) -> Path:
    source_archive = source_archive.expanduser().resolve()
    expected_source_archive = DEFAULT_SOURCE_ARCHIVE.expanduser().resolve()
    if source_archive != expected_source_archive:
        raise SafetyError(f"source archive must be the fixed experiment 10 source directory: {source_archive}")
    if not source_archive.is_dir():
        raise SystemExit(f"source archive not found: {source_archive}")
    return source_archive


def prepare_sources(archive: Path, source_archive: Path, samples: list[Any]) -> dict[str, str]:
    source_archive = validate_source_archive(source_archive)
    texts: dict[str, str] = {}
    manifest: list[dict[str, Any]] = []
    for sample in samples:
        source = source_archive / sample.filename
        if not source.is_file():
            raise SystemExit(f"source file not found: {source}")
        target = archive / "sources" / sample.filename
        shutil.copyfile(source, target)
        texts[sample.sample_id] = target.read_text(encoding="utf-8")
        manifest.append(
            {
                "sample_id": sample.sample_id,
                "source_filename": sample.filename,
                "source_bytes": target.stat().st_size,
                "source_sha256": sha256_file(target),
            }
        )
    write_json(archive / "analysis" / "source-manifest-private.json", manifest)
    return texts


def generate(args: argparse.Namespace, archive: Path, samples: list[Any], modes: list[Any], variants: list[Any]) -> None:
    texts = prepare_sources(archive, args.source_archive, samples)
    jobs: list[tuple[Any, Any, Any, Path, str]] = []
    for sample in samples:
        for mode in modes:
            for variant in variants:
                output = archive / "reports" / sample.sample_id / mode.name / f"{variant.name}.md"
                prompt = report_prompt(sample, mode, variant, texts[sample.sample_id])
                run_id = f"generate-{sample.sample_id}-{mode.name}-{variant.name}"
                if not args.force and cached_prompt_matches(archive, run_id, prompt, output):
                    continue
                jobs.append((sample, mode, variant, output, prompt))

    def run(job: tuple[Any, Any, Any, Path, str]) -> None:
        sample, mode, variant, output, prompt = job
        run_codex(
            archive=archive,
            run_id=f"generate-{sample.sample_id}-{mode.name}-{variant.name}",
            prompt=prompt,
            output_path=output,
            model=args.model,
            reasoning_effort=args.reasoning_effort,
            timeout_seconds=args.timeout_seconds,
        )

    with concurrent.futures.ThreadPoolExecutor(max_workers=args.jobs) as executor:
        submitted = [(job, executor.submit(run, job)) for job in jobs]
        failures = []
        for job, future in submitted:
            try:
                future.result()
            except Exception as exc:  # noqa: BLE001 - preserve all run failures for the archive summary.
                failures.append({"run": [job[0].sample_id, job[1].name, job[2].name], "error": str(exc)})
        if failures:
            write_json(archive / "analysis" / "generation-failures.json", failures)
            raise SystemExit(f"{len(failures)} generation runs failed")


def _side_order(case_id: str, pass_index: int, first: str, second: str) -> tuple[str, str]:
    digest = hashlib.sha256(f"{case_id}:{pass_index}".encode("utf-8")).digest()
    return (first, second) if digest[0] % 2 == 0 else (second, first)


def judge(args: argparse.Namespace, archive: Path, samples: list[Any], modes: list[Any]) -> None:
    source_texts = prepare_sources(archive, args.source_archive, samples)
    schema_path = archive / "analysis" / "judge-schema.json"
    write_json(schema_path, JUDGE_SCHEMA)
    cases: list[dict[str, Any]] = []
    for sample in samples:
        for mode in modes:
            for pair_name, first, second in PAIRINGS:
                case_id = f"{sample.sample_id}-{mode.name}-{pair_name}"
                for pass_index in range(1, args.judge_passes + 1):
                    left, right = _side_order(case_id, pass_index, first, second)
                    left_path = archive / "reports" / sample.sample_id / mode.name / f"{left}.md"
                    right_path = archive / "reports" / sample.sample_id / mode.name / f"{right}.md"
                    if not left_path.is_file() or not right_path.is_file():
                        raise SystemExit(f"missing report for judge case: {left_path} or {right_path}")
                    output = archive / "judging" / case_id / f"pass-{pass_index:02d}.json"
                    cases.append(
                        {
                            "case_id": case_id,
                            "pass": pass_index,
                            "pair": pair_name,
                            "sample": sample,
                            "mode": mode,
                            "left": left,
                            "right": right,
                            "left_path": left_path,
                            "right_path": right_path,
                            "output": output,
                        }
                    )

    def run(case: dict[str, Any]) -> None:
        prompt = judge_prompt(
            sample=case["sample"],
            mode=case["mode"],
            source_text=source_texts[case["sample"].sample_id],
            left_report=case["left_path"].read_text(encoding="utf-8"),
            right_report=case["right_path"].read_text(encoding="utf-8"),
        )
        run_id = f"judge-{case['case_id']}-p{case['pass']:02d}"
        if cached_prompt_matches(archive, run_id, prompt, case["output"]):
            json.loads(case["output"].read_text(encoding="utf-8"))
            return
        run_codex(
            archive=archive,
            run_id=run_id,
            prompt=prompt,
            output_path=case["output"],
            output_schema=schema_path,
            model=args.judge_model or args.model,
            reasoning_effort=args.judge_reasoning_effort,
            timeout_seconds=args.timeout_seconds,
        )

    with concurrent.futures.ThreadPoolExecutor(max_workers=args.judge_jobs) as executor:
        futures = [executor.submit(run, case) for case in cases]
        for future in concurrent.futures.as_completed(futures):
            future.result()

    decode_path = archive / "analysis" / "blind-decode-private.json"
    existing_decode = json.loads(decode_path.read_text(encoding="utf-8")) if decode_path.is_file() else []
    decode_by_key = {(item["case_id"], item["pass"]): item for item in existing_decode}
    for case in cases:
        decode_by_key[(case["case_id"], case["pass"])] = {
            "case_id": case["case_id"],
            "pass": case["pass"],
            "pair": case["pair"],
            "sample": case["sample"].sample_id,
            "mode": case["mode"].name,
            "left": case["left"],
            "right": case["right"],
        }
    write_json(decode_path, [decode_by_key[key] for key in sorted(decode_by_key)])


def analyze(archive: Path, samples: list[Any], modes: list[Any], variants: list[Any]) -> None:
    rows: list[dict[str, Any]] = []
    for sample in samples:
        for mode in modes:
            for variant in variants:
                path = archive / "reports" / sample.sample_id / mode.name / f"{variant.name}.md"
                if path.is_file():
                    rows.append({"sample": sample.sample_id, "mode": mode.name, "variant": variant.name, **report_metrics(path)})
    write_csv(archive / "analysis" / "report-metrics.csv", rows)
    write_json(archive / "analysis" / "report-metrics-summary.json", aggregate_metrics(rows))

    overall: dict[tuple[str, str], Counter[str]] = defaultdict(Counter)
    axes: dict[tuple[str, str, str], Counter[str]] = defaultdict(Counter)
    decode_path = archive / "analysis" / "blind-decode-private.json"
    decode_items = json.loads(decode_path.read_text(encoding="utf-8")) if decode_path.is_file() else []
    decode_by_key = {(item["case_id"], item["pass"]): item for item in decode_items}
    for output in sorted((archive / "judging").glob("*/pass-*.json")):
        payload = json.loads(output.read_text(encoding="utf-8"))
        case_id = output.parent.name
        decode = decode_by_key.get((case_id, int(output.stem.split("-")[-1])))
        if not decode:
            continue
        winner_side = payload["overall_winner"]
        winner = decode[winner_side] if winner_side in {"left", "right"} else "tie"
        overall[(decode["mode"], decode["pair"])][winner] += 1
        for axis in JUDGE_AXES:
            side = payload["axis_winners"][axis]
            axis_winner = decode[side] if side in {"left", "right"} else "tie"
            axes[(decode["mode"], decode["pair"], axis)][axis_winner] += 1
    summary = {
        "overall": [
            {"mode": mode, "pair": pair, "counts": dict(counts)}
            for (mode, pair), counts in sorted(overall.items())
        ],
        "axes": [
            {"mode": mode, "pair": pair, "axis": axis, "counts": dict(counts)}
            for (mode, pair, axis), counts in sorted(axes.items())
        ],
    }
    write_json(archive / "analysis" / "preference-summary.json", summary)


def parser() -> argparse.ArgumentParser:
    result = argparse.ArgumentParser(description=__doc__)
    result.add_argument("--archive", type=Path, default=DEFAULT_ARCHIVE)
    result.add_argument("--source-archive", type=Path, default=DEFAULT_SOURCE_ARCHIVE)
    result.add_argument("--stage", choices=("generate", "judge", "analyze", "all"), default="all")
    result.add_argument("--sample", action="append")
    result.add_argument("--mode", action="append")
    result.add_argument("--variant", action="append")
    result.add_argument("--model", default="gpt-5.5")
    result.add_argument("--judge-model", default="")
    result.add_argument("--reasoning-effort", default="medium")
    result.add_argument("--judge-reasoning-effort", default="medium")
    result.add_argument("--jobs", type=int, default=2)
    result.add_argument("--judge-jobs", type=int, default=2)
    result.add_argument("--judge-passes", type=int, default=1)
    result.add_argument("--timeout-seconds", type=int, default=3600)
    result.add_argument("--dry-run", action="store_true")
    result.add_argument("--force", action="store_true", help="regenerate selected report outputs")
    return result


def main() -> None:
    args = parser().parse_args()
    archive = ensure_archive(args.archive)
    args.source_archive = validate_source_archive(args.source_archive)
    samples = select(SAMPLES, args.sample)
    modes = select(MODES, args.mode)
    variants = select(VARIANTS, args.variant)
    manifest = {
        "experiment_id": EXPERIMENT_ID,
        "archive": str(archive),
        "codex_version": codex_version(),
        "model": args.model,
        "reasoning_effort": args.reasoning_effort,
        "judge_model": args.judge_model or args.model,
        "judge_reasoning_effort": args.judge_reasoning_effort,
        "source_archive": str(args.source_archive),
        "samples": [sample.sample_id for sample in samples],
        "modes": [mode.name for mode in modes],
        "variants": [variant.name for variant in variants],
        "generation_runs": len(samples) * len(modes) * len(variants) if args.stage in {"generate", "all"} else 0,
        "judge_cases": len(samples) * len(modes) * len(PAIRINGS) * args.judge_passes if args.stage in {"judge", "all"} else 0,
    }
    write_json(archive / "analysis" / "run-manifest.json", manifest)
    if args.dry_run:
        print(json.dumps(manifest, ensure_ascii=False, indent=2))
        return
    if args.stage in {"generate", "all"}:
        generate(args, archive, samples, modes, variants)
    if args.stage in {"judge", "all"}:
        judge(args, archive, samples, modes)
    if args.stage in {"analyze", "all"}:
        analyze(archive, samples, modes, variants)


if __name__ == "__main__":
    try:
        main()
    except SafetyError as exc:
        raise SystemExit(f"safety error: {exc}")
