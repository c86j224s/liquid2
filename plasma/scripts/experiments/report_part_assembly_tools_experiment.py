#!/usr/bin/env python3
"""Issue #152 long-form part-assembly edit-tool experiment runner."""

from __future__ import annotations

import argparse
from concurrent.futures import ThreadPoolExecutor, as_completed
from datetime import datetime, timezone
import hashlib
import json
import random
import shutil
import sqlite3
import subprocess
from pathlib import Path
from threading import Lock
import time
from typing import Any

import report_fanout_experiment as base


EXPERIMENT_ID = "26-report-assembly-edit-tools-2026-07-21"
SOURCE_FIXTURE_EXPERIMENT = "17-report-plan-mcp-focused-2026-07-14"
PART_ASSEMBLY_EVENT = "report.part_assembly.submitted"
ARMS = ("visual_plan", "part_assembly_edit_tools")
USAGE_KEYS = (
    "input_tokens",
    "cached_input_tokens",
    "uncached_input_tokens",
    "output_tokens",
    "reasoning_output_tokens",
    "total_tokens",
)
PROFILE_BY_ARM = {
    "visual_plan": "visual-plan",
    "part_assembly_edit_tools": "part-assembly-edit-tools",
}


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--action", choices=("prepare", "run", "analyze", "packets"), required=True)
    parser.add_argument("--fixed-plan", action="store_true", help="reuse one frozen plan per topic for both arms")
    parser.add_argument("--workers", type=int, default=2)
    parser.add_argument("--limit", type=int, default=6)
    parser.add_argument("--arms", nargs="+", choices=ARMS, default=list(ARMS))
    parser.add_argument("--model", default="gpt-5.5")
    parser.add_argument("--effort", default="medium")
    parser.add_argument("--execution-strategy", choices=("serial", "section_fanout"), default="section_fanout")
    parser.add_argument("--timeout-seconds", type=int, default=7200)
    parser.add_argument("--archive", type=Path, default=default_archive())
    parser.add_argument("--source-fixtures", type=Path, default=default_source_archive())
    parser.add_argument("--seed", type=int, default=15226)
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
            "model_default": args.model,
            "effort_default": args.effort,
            "execution_strategy_default": args.execution_strategy,
            "prepared_at": utc_now(),
        },
    )
    print(json.dumps({"archive": str(archive), "fixtures": len(fixtures), "binary": str(binary)}, ensure_ascii=False))


def run(args: argparse.Namespace) -> None:
    if args.fixed_plan:
        run_fixed_plan(args)
        return
    archive = args.archive.expanduser().resolve()
    if not (archive / "bin/plasma").is_file():
        prepare(args)
    fixtures = base.load_fixtures(archive, args.limit)
    arms = tuple(dict.fromkeys(args.arms))
    specs = [(fixture, arm) for fixture in fixtures for arm in arms]
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
            manifest |= {"artifact_id": artifact_id, "report_sha256": hashlib.sha256(report).hexdigest()}
        metrics = base.collect_metrics(events, run_root / "report.md") | collect_part_assembly_metrics(events)
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


def run_fixed_plan(args: argparse.Namespace) -> None:
    archive = args.archive.expanduser().resolve()
    if not (archive / "bin/plasma").is_file():
        prepare(args)
    fixtures = base.load_fixtures(archive, args.limit)
    random.Random(args.seed).shuffle(fixtures)
    used_ports: set[int] = set()
    port_lock = Lock()
    results: list[dict[str, Any]] = []
    with ThreadPoolExecutor(max_workers=max(1, args.workers)) as pool:
        futures = [
            pool.submit(
                run_fixed_plan_topic,
                archive,
                fixture,
                args.model,
                args.effort,
                args.execution_strategy,
                args.timeout_seconds,
                args.seed + index,
                used_ports,
                port_lock,
            )
            for index, fixture in enumerate(fixtures)
        ]
        for future in as_completed(futures):
            result = future.result()
            results.append(result)
            print(json.dumps({"topic": result["topic"], "status": result["status"]}, ensure_ascii=False), flush=True)
    base.write_json(archive / "fixed-run-summary.json", {"completed_at": utc_now(), "results": results})


def run_fixed_plan_topic(
    archive: Path,
    fixture: base.Fixture,
    model: str,
    effort: str,
    execution_strategy: str,
    timeout_seconds: int,
    seed: int,
    used_ports: set[int],
    port_lock: Lock,
) -> dict[str, Any]:
    topic_root = archive / "fixed-runs" / fixture.topic
    terminal = topic_root / "manifest.terminal.json"
    if terminal.exists():
        return json.loads(terminal.read_text(encoding="utf-8"))
    if topic_root.exists():
        shutil.rmtree(topic_root)
    topic_root.mkdir(parents=True, exist_ok=False)

    seed_root = topic_root / "seed"
    seed_manifest = create_seed_plan_run(
        archive,
        seed_root,
        fixture,
        model,
        effort,
        execution_strategy,
        min(900, timeout_seconds),
        used_ports,
        port_lock,
    )
    arms = list(ARMS)
    random.Random(seed).shuffle(arms)
    arm_results = []
    for arm in arms:
        arm_root = topic_root / arm
        copy_seed_root(seed_root, arm_root)
        clear_copied_seed_outputs(arm_root)
        patch_fixed_plan_arm(arm_root / "state/plasma.db", arm)
        result = resume_fixed_plan_arm(
            archive,
            arm_root,
            fixture,
            arm,
            model,
            effort,
            execution_strategy,
            timeout_seconds,
            seed_manifest["mission_id"],
            used_ports,
            port_lock,
        )
        arm_results.append(result)
        if result.get("status") != "completed":
            break

    status = "completed" if len(arm_results) == len(ARMS) and all(item.get("status") == "completed" for item in arm_results) else "failed"
    manifest = {
        "experiment": EXPERIMENT_ID,
        "topic": fixture.topic,
        "status": status,
        "mode": "fixed_plan",
        "seed": seed_manifest,
        "arms": arm_results,
        "completed_at": utc_now(),
    }
    base.write_json(terminal, manifest)
    return manifest


def create_seed_plan_run(
    archive: Path,
    run_root: Path,
    fixture: base.Fixture,
    model: str,
    effort: str,
    execution_strategy: str,
    timeout_seconds: int,
    used_ports: set[int],
    port_lock: Lock,
) -> dict[str, Any]:
    prepare_run_root(run_root, fixture)
    source = run_root / "fixture" / fixture.source_bundle.name
    binary = archive / "bin" / "plasma"
    with port_lock:
        port = base.allocate_port(used_ports)
        connector_port = base.allocate_port(used_ports)
    env = base.isolated_environment(run_root)
    connector_log = (run_root / "logs/liquid2-stub.seed.log").open("xb")
    serve_log = (run_root / "logs/serve.seed.log").open("xb")
    connector = process = None
    manifest = {
        "experiment": EXPERIMENT_ID,
        "topic": fixture.topic,
        "arm": "seed_plan",
        "model": model,
        "effort": effort,
        "generation_guidance_profile": PROFILE_BY_ARM["visual_plan"],
        "execution_strategy": execution_strategy,
        "database": str(run_root / "state/plasma.db"),
        "run_root": str(run_root),
        "status": "started",
        "started_at": utc_now(),
    }
    base.write_json(run_root / "manifest.initial.json", manifest)
    try:
        connector = base.start_connector_stub(connector_port, env, connector_log)
        base.wait_health(f"http://127.0.0.1:{connector_port}", connector, 30)
        process = start_plasma(binary, run_root, manifest["database"], port, connector_port, env, serve_log)
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
            "generation_guidance_profile": PROFILE_BY_ARM["visual_plan"],
            "post_report_humanize": "disabled",
            "report_session_policy": "same_session",
        }
        base.http_json(f"http://127.0.0.1:{port}/api/missions/{mission_id}/reports", body)
        events = poll_for_event(f"http://127.0.0.1:{port}", mission_id, process, "report.plan.created", timeout_seconds)
        manifest |= {"mission_id": mission_id, "status": "plan_created", "completed_at": utc_now()}
        base.write_json(run_root / "ledger.events.json", {"events": events})
        write_plan(run_root, events)
        assert_seed_plan_is_clean(events)
        plan_event = base.first_event(events, "report.plan.created")
        manifest |= {
            "plan_event_id": plan_event.get("EventID") if plan_event else "",
            "plan_signature": plan_signature_from_events(events),
        }
        base.write_json(run_root / "manifest.terminal.json", manifest)
        return manifest
    except Exception as exc:
        manifest |= {"status": "failed", "error": str(exc), "completed_at": utc_now()}
        base.write_json(run_root / "manifest.terminal.json", manifest)
        raise
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


def resume_fixed_plan_arm(
    archive: Path,
    run_root: Path,
    fixture: base.Fixture,
    arm: str,
    model: str,
    effort: str,
    execution_strategy: str,
    timeout_seconds: int,
    mission_id: str,
    used_ports: set[int],
    port_lock: Lock,
) -> dict[str, Any]:
    binary = archive / "bin" / "plasma"
    with port_lock:
        port = base.allocate_port(used_ports)
        connector_port = base.allocate_port(used_ports)
    env = base.isolated_environment(run_root)
    connector_log = (run_root / f"logs/liquid2-stub.{arm}.log").open("xb")
    serve_log = (run_root / f"logs/serve.{arm}.log").open("xb")
    connector = process = None
    manifest = {
        "experiment": EXPERIMENT_ID,
        "topic": fixture.topic,
        "arm": arm,
        "mode": "fixed_plan",
        "model": model,
        "effort": effort,
        "generation_guidance_profile": PROFILE_BY_ARM[arm],
        "execution_strategy": execution_strategy,
        "database": str(run_root / "state/plasma.db"),
        "run_root": str(run_root),
        "mission_id": mission_id,
        "status": "started",
        "started_at": utc_now(),
    }
    base.write_json(run_root / "manifest.resume.json", manifest)
    try:
        connector = base.start_connector_stub(connector_port, env, connector_log)
        base.wait_health(f"http://127.0.0.1:{connector_port}", connector, 30)
        process = start_plasma(binary, run_root, manifest["database"], port, connector_port, env, serve_log)
        base.wait_health(f"http://127.0.0.1:{port}", process, 30)
        base.http_json(f"http://127.0.0.1:{port}/api/missions/{mission_id}")
        events, status = base.poll_terminal(f"http://127.0.0.1:{port}", mission_id, process, timeout_seconds)
        base.write_json(run_root / "ledger.events.json", {"events": events})
        write_plan(run_root, events)
        manifest |= {
            "status": status,
            "completed_at": utc_now(),
            "plan_signature": plan_signature_from_events(events),
        }
        if status == "completed":
            artifact_id = base.final_artifact_id(events)
            report = base.http_bytes(f"http://127.0.0.1:{port}/api/missions/{mission_id}/artifacts/{artifact_id}/download")
            (run_root / "report.md").write_bytes(report)
            manifest |= {"artifact_id": artifact_id, "report_sha256": hashlib.sha256(report).hexdigest()}
        metrics = base.collect_metrics(events, run_root / "report.md") | collect_part_assembly_metrics(events)
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


def prepare_run_root(run_root: Path, fixture: base.Fixture) -> None:
    if run_root.exists():
        shutil.rmtree(run_root)
    run_root.mkdir(parents=True, exist_ok=False)
    for path in ("state", "artifacts", "logs", "workdir", "fixture"):
        (run_root / path).mkdir()
    shutil.copy2(fixture.source_bundle, run_root / "fixture" / fixture.source_bundle.name)


def copy_seed_root(seed_root: Path, arm_root: Path) -> None:
    def ignore_volatile_provider_files(path: str, names: list[str]) -> set[str]:
        relative = Path(path).relative_to(seed_root)
        if relative == Path("provider/codex"):
            return {name for name in names if name in {".tmp", "tmp"}}
        return set()

    shutil.copytree(seed_root, arm_root, ignore=ignore_volatile_provider_files)


def clear_copied_seed_outputs(run_root: Path) -> None:
    for name in ("ledger.events.json", "metrics.json", "plan.json", "report.md"):
        path = run_root / name
        if path.exists():
            path.unlink()
    for path in run_root.glob("manifest*.json"):
        path.unlink()


def start_plasma(binary: Path, run_root: Path, database: str, port: int, connector_port: int, env: dict[str, str], log: object) -> subprocess.Popen[bytes]:
    return subprocess.Popen(
        [
            str(binary),
            "serve",
            "-db",
            database,
            "-addr",
            f"127.0.0.1:{port}",
            "-liquid2-url",
            f"http://127.0.0.1:{connector_port}",
            "-local-source-root",
            f"fixture={run_root / 'fixture'}",
            "-agent",
            "codex",
            "-agent-workdir",
            str(run_root / "workdir"),
            "-agent-timeout",
            "0",
        ],
        env=env,
        stdout=log,
        stderr=subprocess.STDOUT,
    )


def poll_for_event(base_url: str, mission_id: str, process: subprocess.Popen[bytes], event_type: str, timeout_seconds: int) -> list[dict[str, Any]]:
    deadline = time.monotonic() + timeout_seconds
    while time.monotonic() < deadline:
        if process.poll() is not None:
            raise RuntimeError(f"server exited before {event_type}: {process.returncode}")
        payload = base.http_json(f"{base_url}/api/missions/{mission_id}/events")
        events = payload.get("events")
        if not isinstance(events, list):
            raise RuntimeError("events response omitted events")
        if any(isinstance(event, dict) and event.get("EventType") == event_type for event in events):
            return events
        if any(isinstance(event, dict) and event.get("EventType") == "report.draft.failed" for event in events):
            raise RuntimeError(f"report failed before {event_type}")
        time.sleep(0.25)
    raise RuntimeError(f"timed out waiting for {event_type}")


def assert_seed_plan_is_clean(events: list[dict[str, Any]]) -> None:
    if not base.first_event(events, "report.plan.created"):
        raise RuntimeError("seed run did not create a report plan")
    if any(event.get("EventType") == "report.section.created" for event in events):
        raise RuntimeError("seed run completed a section before it could be frozen")
    if any(event.get("EventType") == "report.part.created" for event in events):
        raise RuntimeError("seed run completed a part before it could be frozen")
    if any(event.get("EventType") == "report.artifact.created" for event in events):
        raise RuntimeError("seed run completed the report before it could be frozen")


def patch_fixed_plan_arm(db_path: Path, arm: str) -> None:
    profile = PROFILE_BY_ARM[arm]
    created_at = utc_now()
    with sqlite3.connect(db_path) as conn:
        rows = conn.execute(
            """
            SELECT event_id, event_type, payload_json
            FROM plasma_ledger_events
            WHERE event_type IN ('report.draft.pending', 'report.plan.created')
            ORDER BY sequence
            """
        ).fetchall()
        if len(rows) < 2:
            raise RuntimeError("fixed-plan clone is missing pending or plan events")
        for event_id, event_type, payload_json in rows:
            payload = json.loads(payload_json)
            payload["generation_guidance_profile"] = profile
            payload["fixed_plan_experiment"] = True
            payload["fixed_plan_seed_profile"] = PROFILE_BY_ARM["visual_plan"]
            if event_type == "report.plan.created":
                payload["fixed_plan_seed_event_id"] = event_id
            conn.execute(
                "UPDATE plasma_ledger_events SET payload_json = ? WHERE event_id = ?",
                (json.dumps(payload, ensure_ascii=False, sort_keys=True), event_id),
            )
            conn.execute(
                "UPDATE plasma_ledger_events SET created_at = ? WHERE event_id = ?",
                (created_at, event_id),
            )
        conn.commit()


def plan_signature_from_events(events: list[dict[str, Any]]) -> dict[str, Any]:
    event = base.first_event(events, "report.plan.created")
    payload = event.get("Payload", {}) if isinstance(event, dict) else {}
    plan = payload.get("plan") if isinstance(payload, dict) else None
    if not isinstance(plan, dict):
        return {"sha256": "", "parts": 0, "sections": 0}
    parts = plan.get("parts")
    part_count = len(parts) if isinstance(parts, list) else 0
    section_count = 0
    if isinstance(parts, list):
        for part in parts:
            sections = part.get("sections") if isinstance(part, dict) else None
            if isinstance(sections, list):
                section_count += len(sections)
    encoded = json.dumps(plan, ensure_ascii=False, sort_keys=True).encode("utf-8")
    return {"sha256": hashlib.sha256(encoded).hexdigest(), "parts": part_count, "sections": section_count}


def write_plan(run_root: Path, events: list[dict[str, Any]]) -> None:
    event = base.first_event(events, "report.plan.created")
    payload = event.get("Payload", {}) if isinstance(event, dict) else {}
    plan = payload.get("plan") if isinstance(payload, dict) else None
    if isinstance(plan, dict):
        base.write_json(run_root / "plan.json", plan)


def collect_part_assembly_metrics(events: list[dict[str, Any]]) -> dict[str, Any]:
    submissions = [event for event in events if event.get("EventType") == PART_ASSEMBLY_EVENT]
    part_events = [event for event in events if event.get("EventType") == "report.part.created"]
    section_events = [event for event in events if event.get("EventType") == "report.section.created"]
    part_indices: set[int] = set()
    transition_count = 0
    intro_count = 0
    closing_count = 0
    submitted_section_total = 0
    for event in submissions:
        payload = event.get("Payload", {})
        if not isinstance(payload, dict):
            continue
        if isinstance(payload.get("part_index"), int):
            part_indices.add(payload["part_index"])
        if isinstance(payload.get("section_count"), int):
            submitted_section_total += payload["section_count"]
        assembly = payload.get("assembly", {})
        if not isinstance(assembly, dict):
            continue
        if str(assembly.get("intro", "")).strip():
            intro_count += 1
        if str(assembly.get("closing", "")).strip():
            closing_count += 1
        transitions = assembly.get("transitions", [])
        if isinstance(transitions, list):
            transition_count += len(transitions)
    return {
        "part_created_count": len(part_events),
        "section_created_count": len(section_events),
        "part_assembly_submission_count": len(submissions),
        "part_assembly_submitted_part_count": len(part_indices),
        "part_assembly_submitted_section_total": submitted_section_total,
        "part_assembly_intro_count": intro_count,
        "part_assembly_transition_count": transition_count,
        "part_assembly_closing_count": closing_count,
    }


def analyze(args: argparse.Namespace) -> None:
    archive = args.archive.expanduser().resolve()
    records = []
    manifest_glob = "fixed-runs/*/*/manifest.terminal.json" if args.fixed_plan else "runs/*/manifest.terminal.json"
    for manifest_path in sorted(archive.glob(manifest_glob)):
        manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
        if manifest.get("arm") not in ARMS:
            continue
        metrics_path = manifest_path.parent / "metrics.json"
        metrics = json.loads(metrics_path.read_text(encoding="utf-8")) if metrics_path.exists() else {}
        records.append(manifest | {"metrics": metrics, "usage": collect_agent_usage(manifest_path.parent / "ledger.events.json")})
    by_topic: dict[str, dict[str, dict[str, Any]]] = {}
    for record in records:
        by_topic.setdefault(record["topic"], {})[record["arm"]] = record
    pairs: list[dict[str, Any]] = []
    for topic, arms in sorted(by_topic.items()):
        baseline = arms.get("visual_plan")
        candidate = arms.get("part_assembly_edit_tools")
        if not baseline or not candidate:
            continue
        if baseline.get("status") != "completed" or candidate.get("status") != "completed":
            continue
        pairs.append(pair_summary(topic, baseline, candidate))
    result = {
        "experiment": EXPERIMENT_ID,
        "records": len(records),
        "paired_completed": len(pairs),
        "failures": [record for record in records if record.get("status") != "completed"],
        "candidate_summary": summarize_pairs(pairs),
        "pairs": pairs,
        "manual_review_note": "Automatic metrics only check operation and rough report scale. Whole reports must be read for part-to-part flow, Korean naturalness, source-backed detail, caveat placement, and whether connective edits help rather than smooth over substance.",
    }
    (archive / "analysis").mkdir(parents=True, exist_ok=True)
    analysis_path = archive / ("analysis/fixed-aggregate.json" if args.fixed_plan else "analysis/aggregate.json")
    base.write_json(analysis_path, result)
    print(json.dumps(result, indent=2, ensure_ascii=False))


def pair_summary(topic: str, baseline: dict[str, Any], candidate: dict[str, Any]) -> dict[str, Any]:
    baseline_metrics = baseline.get("metrics", {})
    candidate_metrics = candidate.get("metrics", {})
    part_count = candidate_metrics.get("part_created_count")
    submission_count = candidate_metrics.get("part_assembly_submission_count")
    return {
        "topic": topic,
        "baseline_words": baseline_metrics.get("final_word_count"),
        "candidate_words": candidate_metrics.get("final_word_count"),
        "word_ratio_over_baseline": ratio(candidate_metrics.get("final_word_count"), baseline_metrics.get("final_word_count")),
        "baseline_wall_seconds": baseline_metrics.get("wall_seconds"),
        "candidate_wall_seconds": candidate_metrics.get("wall_seconds"),
        "wall_ratio_over_baseline": ratio(candidate_metrics.get("wall_seconds"), baseline_metrics.get("wall_seconds")),
        "baseline_preservation_ratio": baseline_metrics.get("preservation_ratio"),
        "candidate_preservation_ratio": candidate_metrics.get("preservation_ratio"),
        "preservation_delta": delta(candidate_metrics.get("preservation_ratio"), baseline_metrics.get("preservation_ratio")),
        "candidate_part_count": part_count,
        "candidate_part_assembly_submissions": submission_count,
        "candidate_part_assembly_complete": isinstance(part_count, int) and part_count > 0 and submission_count == part_count,
        "same_plan_signature": baseline.get("plan_signature", {}).get("sha256") == candidate.get("plan_signature", {}).get("sha256"),
        "plan_signature": candidate.get("plan_signature"),
        "candidate_connective_units": {
            "intro": candidate_metrics.get("part_assembly_intro_count"),
            "transition": candidate_metrics.get("part_assembly_transition_count"),
            "closing": candidate_metrics.get("part_assembly_closing_count"),
        },
        "usage": usage_pair_summary(baseline.get("usage", {}), candidate.get("usage", {})),
    }


def summarize_pairs(pairs: list[dict[str, Any]]) -> dict[str, Any]:
    word_ratios = [float(pair["word_ratio_over_baseline"]) for pair in pairs if isinstance(pair.get("word_ratio_over_baseline"), (int, float))]
    wall_ratios = [float(pair["wall_ratio_over_baseline"]) for pair in pairs if isinstance(pair.get("wall_ratio_over_baseline"), (int, float))]
    preservation_deltas = [float(pair["preservation_delta"]) for pair in pairs if isinstance(pair.get("preservation_delta"), (int, float))]
    total_token_ratios = [
        float(pair["usage"]["total_tokens_ratio_over_baseline"])
        for pair in pairs
        if isinstance(pair.get("usage"), dict) and isinstance(pair["usage"].get("total_tokens_ratio_over_baseline"), (int, float))
    ]
    uncached_token_ratios = [
        float(pair["usage"]["uncached_input_tokens_ratio_over_baseline"])
        for pair in pairs
        if isinstance(pair.get("usage"), dict) and isinstance(pair["usage"].get("uncached_input_tokens_ratio_over_baseline"), (int, float))
    ]
    return {
        "completed_pairs": len(pairs),
        "candidate_part_assembly_complete_count": sum(1 for pair in pairs if pair.get("candidate_part_assembly_complete") is True),
        "median_word_ratio_over_baseline": base.median(word_ratios),
        "median_wall_ratio_over_baseline": base.median(wall_ratios),
        "median_preservation_delta": base.median(preservation_deltas),
        "median_total_tokens_ratio_over_baseline": base.median(total_token_ratios),
        "median_uncached_input_tokens_ratio_over_baseline": base.median(uncached_token_ratios),
        "word_longer_sign_p_one_sided": base.exact_one_sided_sign_test(
            sum(1 for value in word_ratios if value > 1),
            sum(1 for value in word_ratios if value < 1),
        ),
        "preservation_nonnegative_sign_p_one_sided": base.exact_one_sided_sign_test(
            sum(1 for value in preservation_deltas if value >= 0),
            sum(1 for value in preservation_deltas if value < 0),
        ),
    }


def collect_agent_usage(ledger_path: Path) -> dict[str, Any]:
    totals = {key: 0 for key in USAGE_KEYS}
    if not ledger_path.exists():
        return totals | {"usage_events": 0}
    data = json.loads(ledger_path.read_text(encoding="utf-8"))
    events = data.get("events") if isinstance(data, dict) else data
    if not isinstance(events, list):
        return totals | {"usage_events": 0}
    usage_events = 0
    for event in events:
        if not isinstance(event, dict):
            continue
        payload = event.get("Payload") or event.get("payload") or {}
        if not isinstance(payload, dict):
            continue
        provider_usage = (payload.get("agent_usage") or {}).get("provider_usage") or {}
        if not isinstance(provider_usage, dict):
            continue
        usage_events += 1
        for key in USAGE_KEYS:
            value = provider_usage.get(key)
            if isinstance(value, (int, float)):
                totals[key] += int(value)
    return totals | {"usage_events": usage_events}


def usage_pair_summary(baseline: dict[str, Any], candidate: dict[str, Any]) -> dict[str, Any]:
    summary: dict[str, Any] = {
        "baseline_usage_events": baseline.get("usage_events"),
        "candidate_usage_events": candidate.get("usage_events"),
    }
    for key in USAGE_KEYS:
        summary[f"baseline_{key}"] = baseline.get(key)
        summary[f"candidate_{key}"] = candidate.get(key)
        summary[f"{key}_ratio_over_baseline"] = ratio(candidate.get(key), baseline.get(key))
    return summary


def packets(args: argparse.Namespace) -> None:
    archive = args.archive.expanduser().resolve()
    analysis_path = archive / ("analysis/fixed-aggregate.json" if args.fixed_plan else "analysis/aggregate.json")
    analysis = json.loads(analysis_path.read_text(encoding="utf-8"))
    out = archive / ("judging/fixed-packets" if args.fixed_plan else "judging/packets")
    out.mkdir(parents=True, exist_ok=True)
    for stale in out.glob("*.json"):
        stale.unlink()
    mapping = {}
    rng = random.Random(args.seed)
    count = 0
    for pair in analysis["pairs"]:
        topic = pair["topic"]
        labels = list(ARMS)
        rng.shuffle(labels)
        packet = {
            "packet_id": f"{EXPERIMENT_ID}-{topic}",
            "topic": topic,
            "mode": "long_form",
            "review_questions": [
                "Does the candidate preserve concrete source-backed detail and caveats?",
                "Does the candidate improve part-to-part and section-to-section flow?",
                "Does the report still read as natural Korean long-form prose?",
                "Does the candidate avoid replacing section substance with generic connective summaries?",
            ],
        }
        for label, arm in zip(("A", "B"), labels):
            if args.fixed_plan:
                report_path = archive / "fixed-runs" / topic / arm / "report.md"
            else:
                report_path = archive / "runs" / f"{topic}-{arm}" / "report.md"
            report = report_path.read_text(encoding="utf-8")
            packet[label] = {"report_markdown": report}
            mapping[f"{topic}:{label}"] = arm
        base.write_json(out / f"{topic}.json", packet)
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
