#!/usr/bin/env python3
"""Issue #153 report visual-aid experiment runner."""

from __future__ import annotations

import argparse
from concurrent.futures import ThreadPoolExecutor, as_completed
from datetime import datetime, timezone
import hashlib
import json
import random
import re
import shutil
import subprocess
from pathlib import Path
from threading import Lock
from typing import Any

import report_fanout_experiment as base


EXPERIMENT_ID = "23-report-visual-aids-2026-07-20"
SOURCE_FIXTURE_EXPERIMENT = "17-report-plan-mcp-focused-2026-07-14"
ARMS = ("baseline", "visual_supplement", "visual_plan")
PROFILE_BY_ARM = {
    "baseline": "g2",
    "visual_supplement": "visual-supplement",
    "visual_plan": "visual-plan",
}
MODES = ("planned", "long_form")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--action", choices=("prepare", "run", "analyze", "packets"), required=True)
    parser.add_argument("--workers", type=int, default=2)
    parser.add_argument("--limit", type=int, default=6)
    parser.add_argument("--modes", nargs="+", choices=MODES, default=list(MODES))
    parser.add_argument("--model", default="gpt-5.5")
    parser.add_argument("--effort", default="medium")
    parser.add_argument("--long-form-strategy", choices=("serial", "section_fanout"), default="section_fanout")
    parser.add_argument("--timeout-seconds", type=int, default=7200)
    parser.add_argument("--archive", type=Path, default=default_archive())
    parser.add_argument("--source-fixtures", type=Path, default=default_source_archive())
    parser.add_argument("--seed", type=int, default=15323)
    return parser.parse_args()


def default_archive() -> Path:
    return Path.home() / "research-artifacts/liquid2/plasma/experiments" / EXPERIMENT_ID


def default_source_archive() -> Path:
    return Path.home() / "research-artifacts/liquid2/plasma/experiments" / SOURCE_FIXTURE_EXPERIMENT


def prepare(args: argparse.Namespace) -> None:
    archive = args.archive.expanduser().resolve()
    source = args.source_fixtures.expanduser().resolve()
    archive.mkdir(parents=True, exist_ok=True)
    (archive / "bin").mkdir(exist_ok=True)
    fixtures = base.load_source_fixtures(source)
    base.write_json_new_or_same(
        archive / "fixtures.lock.json",
        {"fixtures": [base.fixture_to_json(base.copy_fixture(fixture, archive)) for fixture in fixtures]},
    )
    binary = archive / "bin" / "plasma"
    subprocess.run(["go", "build", "-o", str(binary), "./cmd/plasma"], cwd=base.plasma_root(), check=True)
    base.write_json(
        archive / "control.prepare.json",
        {
            "experiment": EXPERIMENT_ID,
            "source_fixture_experiment": SOURCE_FIXTURE_EXPERIMENT,
            "repo": str(base.repo_root()),
            "git_head": base.git("rev-parse", "HEAD"),
            "git_dirty": bool(base.git("status", "--porcelain")),
            "binary": str(binary),
            "binary_sha256": base.sha256(binary),
            "arms": list(ARMS),
            "profiles": PROFILE_BY_ARM,
            "modes_default": args.modes,
            "model_default": args.model,
            "effort_default": args.effort,
            "long_form_strategy_default": args.long_form_strategy,
            "prepared_at": utc_now(),
        },
    )
    print(json.dumps({"archive": str(archive), "fixtures": len(fixtures), "binary": str(binary)}, ensure_ascii=False))


def run(args: argparse.Namespace) -> None:
    archive = args.archive.expanduser().resolve()
    if not (archive / "bin/plasma").is_file():
        prepare(args)
    fixtures = base.load_fixtures(archive, args.limit)
    specs = [(fixture, mode, arm) for fixture in fixtures for mode in args.modes for arm in ARMS]
    random.Random(args.seed).shuffle(specs)
    used_ports: set[int] = set()
    port_lock = Lock()
    results: list[dict[str, Any]] = []
    with ThreadPoolExecutor(max_workers=max(1, args.workers)) as pool:
        futures = [
            pool.submit(
                run_one,
                archive,
                fixture,
                mode,
                arm,
                args.model,
                args.effort,
                args.long_form_strategy,
                args.timeout_seconds,
                used_ports,
                port_lock,
            )
            for fixture, mode, arm in specs
        ]
        for future in as_completed(futures):
            result = future.result()
            results.append(result)
            print(json.dumps({"topic": result["topic"], "mode": result["mode"], "arm": result["arm"], "status": result["status"]}, ensure_ascii=False), flush=True)
    base.write_json(archive / "run-summary.json", {"completed_at": utc_now(), "results": results})


def run_one(
    archive: Path,
    fixture: base.Fixture,
    mode: str,
    arm: str,
    model: str,
    effort: str,
    long_form_strategy: str,
    timeout_seconds: int,
    used_ports: set[int],
    port_lock: Lock,
) -> dict[str, Any]:
    run_root = archive / "runs" / f"{fixture.topic}-{mode}-{arm}"
    terminal = run_root / "manifest.terminal.json"
    if terminal.exists():
        return json.loads(terminal.read_text(encoding="utf-8"))
    if run_root.exists():
        shutil.rmtree(run_root)
    run_root.mkdir(parents=True, exist_ok=False)
    for path in ("state", "artifacts", "logs", "workdir", "fixture"):
        (run_root / path).mkdir()
    source = run_root / "fixture" / fixture.source_bundle.name
    shutil.copy2(fixture.source_bundle, source)
    binary = archive / "bin" / "plasma"
    with port_lock:
        port = base.allocate_port(used_ports)
        connector_port = base.allocate_port(used_ports)
    env = base.isolated_environment(run_root)
    connector_log = (run_root / "logs/liquid2-stub.log").open("xb")
    serve_log = (run_root / "logs/serve.log").open("xb")
    connector = process = None
    manifest = {
        "experiment": EXPERIMENT_ID,
        "topic": fixture.topic,
        "mode": mode,
        "arm": arm,
        "model": model,
        "effort": effort,
        "generation_guidance_profile": PROFILE_BY_ARM[arm],
        "long_form_strategy": long_form_strategy if mode == "long_form" else "",
        "database": str(run_root / "state/plasma.db"),
        "run_root": str(run_root),
        "port": port,
        "connector_port": connector_port,
        "binary": str(binary),
        "binary_sha256": base.sha256(binary),
        "status": "started",
        "started_at": utc_now(),
    }
    base.write_json(run_root / "manifest.initial.json", manifest)
    try:
        connector = base.start_connector_stub(connector_port, env, connector_log)
        base.wait_health(f"http://127.0.0.1:{connector_port}", connector, 30)
        process = subprocess.Popen(
            [
                str(binary),
                "serve",
                "-db",
                manifest["database"],
                "-addr",
                f"127.0.0.1:{port}",
                "-liquid2-url",
                f"http://127.0.0.1:{connector_port}",
                "-local-source-root",
                f"fixture={source.parent}",
                "-agent",
                "codex",
                "-agent-workdir",
                str(run_root / "workdir"),
                "-agent-timeout",
                "0",
            ],
            env=env,
            stdout=serve_log,
            stderr=subprocess.STDOUT,
        )
        base.wait_health(f"http://127.0.0.1:{port}", process, 30)
        mission = base.run_json(
            [
                str(binary),
                "missions",
                "create",
                "-db",
                manifest["database"],
                "-title",
                fixture.title,
                "-objective",
                fixture.objective,
                "-json",
            ],
            env,
        )
        mission_id = base.find_string(mission, "MissionID", "mission_id")
        base.run_json(
            [
                str(binary),
                "sources",
                "attach-local",
                mission_id,
                "-db",
                manifest["database"],
                "-root",
                "fixture",
                "-path",
                source.name,
                "-title",
                fixture.title,
                "-local-source-root",
                f"fixture={source.parent}",
                "-json",
            ],
            env,
        )
        body = {
            "title": fixture.title,
            "report_mode": mode,
            "agent_executor": "codex",
            "agent_model": model,
            "agent_reasoning_effort": effort,
            "generation_guidance_profile": PROFILE_BY_ARM[arm],
            "post_report_humanize": "disabled",
            "report_session_policy": "same_session",
        }
        if mode == "long_form":
            body["execution_strategy"] = long_form_strategy
        base.http_json(f"http://127.0.0.1:{port}/api/missions/{mission_id}/reports", body)
        events, status = base.poll_terminal(f"http://127.0.0.1:{port}", mission_id, process, timeout_seconds)
        base.write_json(run_root / "ledger.events.json", {"events": events})
        write_plan(run_root, events)
        manifest |= {"mission_id": mission_id, "status": status, "completed_at": utc_now()}
        if status == "completed":
            artifact_id = base.final_artifact_id(events)
            report = base.http_bytes(f"http://127.0.0.1:{port}/api/missions/{mission_id}/artifacts/{artifact_id}/download")
            (run_root / "report.md").write_bytes(report)
            manifest |= {"artifact_id": artifact_id, "report_sha256": hashlib.sha256(report).hexdigest()}
        metrics = base.collect_metrics(events, run_root / "report.md") | collect_visual_metrics(events, run_root / "report.md")
        base.write_json(run_root / "metrics.json", metrics)
        base.write_json(run_root / "manifest.terminal.json", manifest)
        return manifest
    except Exception as exc:
        manifest |= {"status": "failed", "error": str(exc), "completed_at": utc_now()}
        base.write_json(run_root / "manifest.terminal.json", manifest)
        return manifest
    finally:
        if process is not None:
            base.stop_process(process)
        if connector is not None:
            base.stop_process(connector)
        with port_lock:
            used_ports.discard(port)
            used_ports.discard(connector_port)
        serve_log.close()
        connector_log.close()


def write_plan(run_root: Path, events: list[dict[str, Any]]) -> None:
    event = base.first_event(events, "report.plan.created")
    payload = event.get("Payload", {}) if isinstance(event, dict) else {}
    plan = payload.get("plan") if isinstance(payload, dict) else None
    if isinstance(plan, dict):
        base.write_json(run_root / "plan.json", plan)


def collect_visual_metrics(events: list[dict[str, Any]], report_path: Path) -> dict[str, Any]:
    report = report_path.read_text(encoding="utf-8") if report_path.exists() else ""
    mermaid_fences = len(re.findall(r"(?im)^```mermaid\s*$", report))
    table_headers = len(re.findall(r"(?m)^\s*\|[^\n]+\|\s*\n\s*\|\s*:?-{3,}:?\s*\|", report))
    event_blob = json.dumps(events, ensure_ascii=False)
    validation_calls = event_blob.count("plasma.mermaid.validate")
    words = max(1, len(report.split()))
    return {
        "table_count": table_headers,
        "mermaid_fence_count": mermaid_fences,
        "visual_aid_count": table_headers + mermaid_fences,
        "visual_aids_per_1000_words": round(((table_headers + mermaid_fences) / words) * 1000, 3),
        "mermaid_validate_mentions": validation_calls,
        "has_unvalidated_mermaid_signal": mermaid_fences > validation_calls,
        "visual_candidate_line_count": count_visual_candidate_lines(report),
    }


def count_visual_candidate_lines(report: str) -> int:
    markers = ("|", "```mermaid", "flowchart", "graph ", "sequenceDiagram", "timeline", "stateDiagram")
    return sum(1 for line in report.splitlines() if any(marker in line for marker in markers))


def analyze(args: argparse.Namespace) -> None:
    archive = args.archive.expanduser().resolve()
    records = []
    for manifest_path in sorted((archive / "runs").glob("*/manifest.terminal.json")):
        manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
        if manifest.get("arm") not in ARMS:
            continue
        metrics_path = manifest_path.parent / "metrics.json"
        metrics = json.loads(metrics_path.read_text(encoding="utf-8")) if metrics_path.exists() else {}
        records.append(manifest | {"metrics": metrics})
    by_key: dict[tuple[str, str], dict[str, dict[str, Any]]] = {}
    for record in records:
        by_key.setdefault((record["topic"], record["mode"]), {})[record["arm"]] = record
    pairs: list[dict[str, Any]] = []
    candidate_arms = [arm for arm in ARMS if arm != "baseline"]
    for (topic, mode), arms in sorted(by_key.items()):
        if all(arm in arms and arms[arm]["status"] == "completed" for arm in ARMS):
            baseline = arms["baseline"]
            pair: dict[str, Any] = {
                "topic": topic,
                "mode": mode,
                "baseline_words": baseline["metrics"].get("final_word_count"),
                "baseline_visual_aids": baseline["metrics"].get("visual_aid_count"),
                "baseline_mermaid": baseline["metrics"].get("mermaid_fence_count"),
                "baseline_tables": baseline["metrics"].get("table_count"),
                "candidates": {},
            }
            for arm in candidate_arms:
                candidate = arms[arm]
                pair["candidates"][arm] = candidate_summary(candidate, baseline)
            pairs.append(pair)
    result = {
        "experiment": EXPERIMENT_ID,
        "records": len(records),
        "paired_completed": len(pairs),
        "failures": [record for record in records if record.get("status") != "completed"],
        "arm_summaries": summarize_arms(pairs),
        "pairs": pairs,
        "manual_review_note": "Automatic visual counts are observation signals only. Judge packets must be read as whole reports for usefulness, repetition, prose flow, and source grounding.",
    }
    base.write_json(archive / "analysis/aggregate.json", result)
    print(json.dumps(result, indent=2, ensure_ascii=False))


def candidate_summary(candidate: dict[str, Any], baseline: dict[str, Any]) -> dict[str, Any]:
    metrics = candidate.get("metrics", {})
    base_metrics = baseline.get("metrics", {})
    return {
        "words": metrics.get("final_word_count"),
        "visual_aids": metrics.get("visual_aid_count"),
        "mermaid": metrics.get("mermaid_fence_count"),
        "tables": metrics.get("table_count"),
        "visual_delta": delta(metrics.get("visual_aid_count"), base_metrics.get("visual_aid_count")),
        "word_ratio_over_baseline": ratio(metrics.get("final_word_count"), base_metrics.get("final_word_count")),
        "unvalidated_mermaid_signal": metrics.get("has_unvalidated_mermaid_signal"),
    }


def summarize_arms(pairs: list[dict[str, Any]]) -> dict[str, dict[str, Any]]:
    summaries: dict[str, dict[str, Any]] = {}
    for arm in (arm for arm in ARMS if arm != "baseline"):
        visual_deltas: list[float] = []
        word_ratios: list[float] = []
        unvalidated = 0
        for pair in pairs:
            summary = pair["candidates"].get(arm, {})
            visual_delta = summary.get("visual_delta")
            word_ratio = summary.get("word_ratio_over_baseline")
            if isinstance(visual_delta, (int, float)):
                visual_deltas.append(float(visual_delta))
            if isinstance(word_ratio, (int, float)):
                word_ratios.append(float(word_ratio))
            if summary.get("unvalidated_mermaid_signal"):
                unvalidated += 1
        summaries[arm] = {
            "completed_pairs": len(visual_deltas),
            "median_visual_delta": base.median(visual_deltas),
            "visual_increase_sign_p_one_sided": base.exact_one_sided_sign_test(sum(1 for value in visual_deltas if value > 0), sum(1 for value in visual_deltas if value < 0)),
            "median_word_ratio_over_baseline": base.median(word_ratios),
            "unvalidated_mermaid_signals": unvalidated,
        }
    return summaries


def packets(args: argparse.Namespace) -> None:
    archive = args.archive.expanduser().resolve()
    analysis = json.loads((archive / "analysis/aggregate.json").read_text(encoding="utf-8"))
    out = archive / "judging/packets"
    out.mkdir(parents=True, exist_ok=True)
    for stale in out.glob("*.json"):
        stale.unlink()
    mapping = {}
    rng = random.Random(args.seed)
    candidate_arms = [arm for arm in ARMS if arm != "baseline"]
    count = 0
    for pair in analysis["pairs"]:
        topic = pair["topic"]
        mode = pair["mode"]
        for candidate_arm in candidate_arms:
            labels = ["baseline", candidate_arm]
            rng.shuffle(labels)
            packet = {
                "packet_id": f"{EXPERIMENT_ID}-{topic}-{mode}-{candidate_arm}",
                "topic": topic,
                "candidate_arm": candidate_arm,
                "mode": mode,
                "review_questions": [
                    "Does the visual aid help understanding rather than repeat nearby prose?",
                    "Does the report still read as a coherent article?",
                    "Did tables or Mermaid replace source-grounded explanation?",
                    "Are Mermaid diagrams syntactically and visually plausible?",
                ],
            }
            for label, arm in zip(("A", "B"), labels):
                report = (archive / "runs" / f"{topic}-{mode}-{arm}" / "report.md").read_text(encoding="utf-8")
                packet[label] = {"report_markdown": report}
                mapping[f"{topic}:{mode}:{candidate_arm}:{label}"] = arm
            base.write_json(out / f"{topic}-{mode}-{candidate_arm}.json", packet)
            count += 1
    base.write_json(archive / "judging/private-mapping.json", mapping)
    print(json.dumps({"packets": count, "path": str(out)}, ensure_ascii=False))


def delta(candidate: object, baseline: object) -> float | None:
    if not isinstance(candidate, (int, float)) or not isinstance(baseline, (int, float)):
        return None
    return float(candidate) - float(baseline)


def ratio(numerator: object, denominator: object) -> float | None:
    if not isinstance(numerator, (int, float)) or not isinstance(denominator, (int, float)) or denominator == 0:
        return None
    return numerator / denominator


def utc_now() -> str:
    return datetime.now(timezone.utc).replace(tzinfo=None).isoformat(timespec="seconds") + "Z"


def main() -> int:
    args = parse_args()
    if args.action == "prepare":
        prepare(args)
    elif args.action == "run":
        run(args)
    elif args.action == "analyze":
        analyze(args)
    elif args.action == "packets":
        packets(args)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
