#!/usr/bin/env python3
"""Issue #146-style long-form section-contract report experiment runner."""

from __future__ import annotations

import argparse
from concurrent.futures import ThreadPoolExecutor, as_completed
from datetime import datetime, timezone
import json
import random
import shutil
import subprocess
import sys
from pathlib import Path
from threading import Lock
import time
from typing import Any

import report_fanout_experiment as base


EXPERIMENT_ID = "22-report-section-contract-2026-07-17"
SOURCE_FIXTURE_EXPERIMENT = "17-report-plan-mcp-focused-2026-07-14"
ARMS = ("baseline", "section_brief", "section_brief_cluster_memory")
PROFILE_BY_ARM = {
    "baseline": "g2",
    "section_brief": "section-brief",
    "section_brief_cluster_memory": "section-brief-cluster-memory",
}


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--action", choices=("prepare", "run", "analyze", "packets"), required=True)
    parser.add_argument("--workers", type=int, default=2)
    parser.add_argument("--limit", type=int, default=6)
    parser.add_argument("--model", default="gpt-5.5")
    parser.add_argument("--effort", default="medium")
    parser.add_argument("--execution-strategy", choices=("serial", "section_fanout"), default="section_fanout")
    parser.add_argument("--timeout-seconds", type=int, default=7200)
    parser.add_argument("--archive", type=Path, default=default_archive())
    parser.add_argument("--source-fixtures", type=Path, default=default_source_archive())
    parser.add_argument("--seed", type=int, default=2217)
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
            "repo": str(base.repo_root()),
            "git_head": base.git("rev-parse", "HEAD"),
            "git_dirty": bool(base.git("status", "--porcelain")),
            "binary": str(binary),
            "binary_sha256": base.sha256(binary),
            "model_default": args.model,
            "effort_default": args.effort,
            "execution_strategy_default": args.execution_strategy,
            "prepared_at": utc_now(),
        },
    )
    print(json.dumps({"archive": str(archive), "fixtures": len(fixtures), "binary": str(binary)}, ensure_ascii=False))


def run(args: argparse.Namespace) -> None:
    archive = args.archive.expanduser().resolve()
    if not (archive / "bin/plasma").is_file():
        prepare(args)
    fixtures = base.load_fixtures(archive, args.limit)
    specs = [(fixture, arm) for fixture in fixtures for arm in ARMS]
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
                arm,
                args.model,
                args.effort,
                args.execution_strategy,
                args.timeout_seconds,
                used_ports,
                port_lock,
            )
            for fixture, arm in specs
        ]
        for future in as_completed(futures):
            result = future.result()
            results.append(result)
            print(json.dumps({"topic": result["topic"], "arm": result["arm"], "status": result["status"]}, ensure_ascii=False), flush=True)
    base.write_json(archive / "run-summary.json", {"completed_at": utc_now(), "results": results})


def run_one(
    archive: Path,
    fixture: base.Fixture,
    arm: str,
    model: str,
    effort: str,
    execution_strategy: str,
    timeout_seconds: int,
    used_ports: set[int],
    port_lock: Lock,
) -> dict[str, Any]:
    run_root = archive / "runs" / f"{fixture.topic}-{arm}"
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
    port = 0
    connector_port = 0
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
        "arm": arm,
        "model": model,
        "effort": effort,
        "generation_guidance_profile": PROFILE_BY_ARM[arm],
        "execution_strategy": execution_strategy,
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
            "report_mode": "long_form",
            "execution_strategy": execution_strategy,
            "agent_executor": "codex",
            "agent_model": model,
            "agent_reasoning_effort": effort,
            "generation_guidance_profile": PROFILE_BY_ARM[arm],
            "post_report_humanize": "disabled",
            "report_session_policy": "same_session",
        }
        base.http_json(f"http://127.0.0.1:{port}/api/missions/{mission_id}/reports", body)
        events, status = base.poll_terminal(f"http://127.0.0.1:{port}", mission_id, process, timeout_seconds)
        base.write_json(run_root / "ledger.events.json", {"events": events})
        write_plan(run_root, events)
        manifest |= {"mission_id": mission_id, "status": status, "completed_at": utc_now()}
        if status == "completed":
            artifact_id = base.final_artifact_id(events)
            report = base.http_bytes(f"http://127.0.0.1:{port}/api/missions/{mission_id}/artifacts/{artifact_id}/download")
            (run_root / "report.md").write_bytes(report)
            manifest |= {"artifact_id": artifact_id, "report_sha256": base.hashlib.sha256(report).hexdigest()}
        base.write_json(run_root / "metrics.json", base.collect_metrics(events, run_root / "report.md"))
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


def analyze(args: argparse.Namespace) -> None:
    archive = args.archive.expanduser().resolve()
    records = []
    for manifest_path in sorted((archive / "runs").glob("*/manifest.terminal.json")):
        manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
        metrics_path = manifest_path.parent / "metrics.json"
        metrics = json.loads(metrics_path.read_text(encoding="utf-8")) if metrics_path.exists() else {}
        records.append(manifest | {"metrics": metrics})
    by_topic: dict[str, dict[str, dict[str, Any]]] = {}
    for record in records:
        by_topic.setdefault(record["topic"], {})[record["arm"]] = record
    pairs = []
    candidate_arms = [arm for arm in ARMS if arm != "baseline"]
    for topic, arms in sorted(by_topic.items()):
        if all(arm in arms and arms[arm]["status"] == "completed" for arm in ARMS):
            baseline = arms["baseline"]
            pair: dict[str, Any] = {
                "topic": topic,
                "baseline_words": baseline["metrics"].get("final_word_count"),
                "baseline_parts": baseline["metrics"].get("part_count"),
                "baseline_sections": baseline["metrics"].get("section_count"),
                "baseline_wall_seconds": baseline["metrics"].get("wall_seconds"),
                "candidates": {},
            }
            for arm in candidate_arms:
                candidate = arms[arm]
                pair["candidates"][arm] = {
                    "words": candidate["metrics"].get("final_word_count"),
                    "parts": candidate["metrics"].get("part_count"),
                    "sections": candidate["metrics"].get("section_count"),
                    "wall_seconds": candidate["metrics"].get("wall_seconds"),
                    "word_ratio_over_baseline": ratio(candidate["metrics"].get("final_word_count"), baseline["metrics"].get("final_word_count")),
                    "section_ratio_over_baseline": ratio(candidate["metrics"].get("section_count"), baseline["metrics"].get("section_count")),
                }
            pairs.append(pair)
    result = {
        "experiment": EXPERIMENT_ID,
        "records": len(records),
        "paired_completed": len(pairs),
        "failures": [record for record in records if record.get("status") != "completed"],
        "pairs": pairs,
    }
    base.write_json(archive / "analysis/aggregate.json", result)
    print(json.dumps(result, indent=2, ensure_ascii=False))


def packets(args: argparse.Namespace) -> None:
    archive = args.archive.expanduser().resolve()
    analysis = json.loads((archive / "analysis/aggregate.json").read_text(encoding="utf-8"))
    out = archive / "judging/packets"
    out.mkdir(parents=True, exist_ok=True)
    mapping = {}
    rng = random.Random(args.seed)
    candidate_arms = [arm for arm in ARMS if arm != "baseline"]
    for pair in analysis["pairs"]:
        topic = pair["topic"]
        for candidate_arm in candidate_arms:
            labels = ["baseline", candidate_arm]
            rng.shuffle(labels)
            packet = {"packet_id": f"{EXPERIMENT_ID}-{topic}-{candidate_arm}", "topic": topic, "candidate_arm": candidate_arm, "replicate": 1, "mode": "long_form"}
            for label, arm in zip(("A", "B"), labels):
                report = (archive / "runs" / f"{topic}-{arm}" / "report.md").read_text(encoding="utf-8")
                packet[label] = {"report_markdown": report}
                mapping[f"{topic}:{candidate_arm}:{label}"] = arm
            base.write_json(out / f"{topic}-{candidate_arm}.json", packet)
    base.write_json(archive / "judging/private-mapping.json", mapping)
    print(json.dumps({"packets": len(analysis["pairs"]), "path": str(out)}, ensure_ascii=False))


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
