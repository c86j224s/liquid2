#!/usr/bin/env python3
"""Run phase-2 issue 77 gap-stress report generation, judging, and analysis."""

from __future__ import annotations

import argparse
import concurrent.futures
import hashlib
import json
import shutil
from collections import Counter, defaultdict
from pathlib import Path
from typing import Any

import markdown_report_magic_words_runner as runner
from markdown_report_gap_stress_metrics import aggregate_metrics, report_metrics, write_csv
from markdown_report_gap_stress_protocol import (
    EXPERIMENT_ID,
    GAP_STRESS_SUBTREE,
    JUDGE_AXES,
    JUDGE_SCHEMA,
    MODES,
    PAIRINGS,
    SAMPLES,
    VARIANTS,
    judge_prompt,
    report_prompt,
)


DEFAULT_SOURCE_ARCHIVE = (
    Path.home()
    / "research-artifacts/liquid2/plasma/experiments/10-generation-time-tone-2026-07-07/sources"
)
DEFAULT_ARCHIVE = runner.DEFAULT_ARCHIVE / GAP_STRESS_SUBTREE


def validate_archive(path: Path) -> Path:
    archive = path.expanduser().resolve()
    if archive != DEFAULT_ARCHIVE.expanduser().resolve():
        raise runner.SafetyError(f"archive must be the phase-2 gap-stress subtree: {archive}")
    return archive


def validate_source_archive(path: Path) -> Path:
    source = path.expanduser().resolve()
    if source != DEFAULT_SOURCE_ARCHIVE.expanduser().resolve():
        raise runner.SafetyError(f"source archive must be the fixed experiment 10 source directory: {source}")
    if not source.is_dir():
        raise SystemExit(f"source archive not found: {source}")
    return source


def prepare_archive(path: Path) -> Path:
    archive = validate_archive(path)
    for name in ("analysis", "judging", "logs", "reports", "runs", "sources", "tmp-harness"):
        (archive / name).mkdir(parents=True, exist_ok=True)
    return archive


def select(values: tuple[Any, ...], names: list[str] | None) -> list[Any]:
    if not names:
        return list(values)
    selected = [
        value
        for value in values
        if getattr(value, "sample_id", getattr(value, "name", "")) in names
    ]
    if len(selected) != len(set(names)):
        raise SystemExit(f"unknown selection in {names}")
    return selected


def prepare_sources(
    archive: Path,
    source_archive: Path,
    samples: list[Any],
) -> dict[str, str]:
    texts: dict[str, str] = {}
    manifest: list[dict[str, str]] = []
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
                "source_sha256": runner.sha256_file(target),
            }
        )
    runner.write_json(archive / "analysis" / "source-manifest-private.json", manifest)
    return texts


def _side_order(case_id: str, first: str, second: str) -> tuple[str, str]:
    return (first, second) if hashlib.sha256(case_id.encode()).digest()[0] % 2 == 0 else (second, first)


def generate(
    args: argparse.Namespace,
    archive: Path,
    samples: list[Any],
    modes: list[Any],
    variants: list[Any],
) -> None:
    texts = prepare_sources(archive, args.source_archive, samples)
    jobs = []
    for sample in samples:
        for mode in modes:
            for variant in variants:
                output = archive / "reports" / sample.sample_id / mode.name / f"{variant.name}.md"
                prompt = report_prompt(sample, mode, variant, texts[sample.sample_id])
                run_id = f"generate-{sample.sample_id}-{mode.name}-{variant.name}"
                if args.force or not _cached(archive, run_id, prompt, output):
                    jobs.append((run_id, prompt, output))
    _execute(args, archive, jobs, args.jobs, None)


def _cached(archive: Path, run_id: str, prompt: str, output: Path) -> bool:
    records = archive / "runs" / "commands.jsonl"
    if not output.is_file() or not output.read_text(encoding="utf-8", errors="replace").strip() or not records.is_file():
        return False
    expected = runner.sha256_text(prompt)
    for line in reversed(records.read_text(encoding="utf-8", errors="replace").splitlines()):
        try:
            record = json.loads(line)
        except json.JSONDecodeError:
            continue
        if record.get("run_id") == run_id:
            return record.get("returncode") == 0 and record.get("prompt_sha256") == expected
    return False


def _execute(
    args: argparse.Namespace,
    archive: Path,
    jobs: list[tuple[str, str, Path]],
    workers: int,
    schema: Path | None,
) -> None:
    def one(job: tuple[str, str, Path]) -> None:
        run_id, prompt, output = job
        runner.run_codex(
            archive=archive,
            run_id=run_id,
            prompt=prompt,
            output_path=output,
            output_schema=schema,
            model=args.judge_model if schema and args.judge_model else args.model,
            reasoning_effort=args.judge_reasoning_effort if schema else args.reasoning_effort,
            timeout_seconds=args.timeout_seconds,
        )
    with concurrent.futures.ThreadPoolExecutor(max_workers=workers) as pool:
        futures = [pool.submit(one, job) for job in jobs]
        for future in futures:
            future.result()


def judge(
    args: argparse.Namespace,
    archive: Path,
    samples: list[Any],
    modes: list[Any],
    variants: list[Any],
) -> None:
    texts = prepare_sources(archive, args.source_archive, samples)
    schema = archive / "analysis" / "judge-schema.json"
    runner.write_json(schema, JUDGE_SCHEMA)
    jobs: list[tuple[str, str, Path]] = []
    decode: list[dict[str, Any]] = []
    selected_variants = {variant.name for variant in variants}
    for sample in samples:
        for mode in modes:
            for pair, first, second in PAIRINGS:
                if first not in selected_variants or second not in selected_variants:
                    continue
                case_id = f"{sample.sample_id}-{mode.name}-{pair}"
                left, right = _side_order(case_id, first, second)
                left_path = archive / "reports" / sample.sample_id / mode.name / f"{left}.md"
                right_path = archive / "reports" / sample.sample_id / mode.name / f"{right}.md"
                if not left_path.is_file() or not right_path.is_file():
                    raise SystemExit(f"missing report for judge case: {left_path} or {right_path}")
                output = archive / "judging" / case_id / "pass-01.json"
                prompt = judge_prompt(
                    sample=sample,
                    mode=mode,
                    source_text=texts[sample.sample_id],
                    left_report=left_path.read_text(encoding="utf-8"),
                    right_report=right_path.read_text(encoding="utf-8"),
                )
                if args.force or not _cached(archive, f"judge-{case_id}-p01", prompt, output):
                    jobs.append((f"judge-{case_id}-p01", prompt, output))
                decode.append(
                    {
                        "case_id": case_id,
                        "pass": 1,
                        "pair": pair,
                        "sample": sample.sample_id,
                        "mode": mode.name,
                        "left": left,
                        "right": right,
                    }
                )
    _execute(args, archive, jobs, args.judge_jobs, schema)
    runner.write_json(archive / "analysis" / "blind-decode-private.json", decode)


def analyze(archive: Path, samples: list[Any], modes: list[Any], variants: list[Any]) -> None:
    rows: list[dict[str, Any]] = []
    for sample in samples:
        for mode in modes:
            for variant in variants:
                path = archive / "reports" / sample.sample_id / mode.name / f"{variant.name}.md"
                if path.is_file():
                    rows.append(
                        {
                            "sample": sample.sample_id,
                            "mode": mode.name,
                            "variant": variant.name,
                            **report_metrics(path),
                        }
                    )
    write_csv(archive / "analysis" / "report-metrics.csv", rows)
    runner.write_json(archive / "analysis" / "report-metrics-summary.json", aggregate_metrics(rows))
    decode_path = archive / "analysis" / "blind-decode-private.json"
    decode_items = json.loads(decode_path.read_text(encoding="utf-8")) if decode_path.is_file() else []
    decode = {(item["case_id"], item["pass"]): item for item in decode_items}
    overall: dict[tuple[str, str], Counter[str]] = defaultdict(Counter)
    axes: dict[tuple[str, str, str], Counter[str]] = defaultdict(Counter)
    for output in (archive / "judging").glob("*/pass-*.json"):
        item = decode.get((output.parent.name, 1))
        if not item:
            continue
        payload = json.loads(output.read_text(encoding="utf-8"))
        overall[(item["mode"], item["pair"])][item.get(payload["overall_winner"], "tie")] += 1
        for axis in JUDGE_AXES:
            axes[(item["mode"], item["pair"], axis)][item.get(payload["axis_winners"][axis], "tie")] += 1
    runner.write_json(
        archive / "analysis" / "preference-summary.json",
        {
            "overall": [
                {"mode": key[0], "pair": key[1], "counts": dict(value)}
                for key, value in sorted(overall.items())
            ],
            "axes": [
                {"mode": key[0], "pair": key[1], "axis": key[2], "counts": dict(value)}
                for key, value in sorted(axes.items())
            ],
        },
    )


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
    result.add_argument("--jobs", type=int, default=1)
    result.add_argument("--judge-jobs", type=int, default=1)
    result.add_argument("--timeout-seconds", type=int, default=3600)
    result.add_argument("--dry-run", action="store_true")
    result.add_argument("--force", action="store_true")
    return result


def main() -> None:
    args = parser().parse_args()
    archive = validate_archive(args.archive)
    args.source_archive = validate_source_archive(args.source_archive)
    samples = select(SAMPLES, args.sample)
    modes = select(MODES, args.mode)
    variants = select(VARIANTS, args.variant)
    selected_variant_names = {variant.name for variant in variants}
    selected_pairs = [
        pair
        for pair in PAIRINGS
        if pair[1] in selected_variant_names and pair[2] in selected_variant_names
    ]
    manifest = {
        "experiment_id": EXPERIMENT_ID,
        "phase": "gap-stress",
        "archive": str(archive),
        "source_archive": str(args.source_archive),
        "codex_version": runner.codex_version(),
        "generation_model": args.model,
        "judge_model": args.judge_model or args.model,
        "generation_reasoning_effort": args.reasoning_effort,
        "judge_reasoning_effort": args.judge_reasoning_effort,
        "samples": [sample.sample_id for sample in samples],
        "modes": [mode.name for mode in modes],
        "variants": [variant.name for variant in variants],
        "generation_runs": len(samples) * len(modes) * len(variants)
        if args.stage in {"generate", "all"}
        else 0,
        "judge_cases": len(samples) * len(modes) * len(selected_pairs)
        if args.stage in {"judge", "all"}
        else 0,
    }
    if args.dry_run:
        print(json.dumps(manifest, ensure_ascii=False, indent=2))
        return
    archive = prepare_archive(archive)
    runner.write_json(archive / "analysis" / "run-manifest.json", manifest)
    # The phase-1 runner remains unmodified; this process narrows its archive guard to phase 2.
    runner.validate_archive = validate_archive
    if args.stage in {"generate", "all"}:
        generate(args, archive, samples, modes, variants)
    if args.stage in {"judge", "all"}:
        judge(args, archive, samples, modes, variants)
    if args.stage in {"analyze", "all"}:
        analyze(archive, samples, modes, variants)


if __name__ == "__main__":
    try:
        main()
    except runner.SafetyError as exc:
        raise SystemExit(f"safety error: {exc}")
