#!/usr/bin/env python3
"""Issue #210 long-form report direction product-path experiment."""

from __future__ import annotations

import argparse
from concurrent.futures import ThreadPoolExecutor, as_completed
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
from threading import Lock
from typing import Any

import report_fanout_experiment as base


EXPERIMENT_ID = "210-report-direction-center"
ARMS = ("baseline", "candidate")
STRATEGIES = ("serial", "section_fanout")
TITLE = "Plasma 제품 아키텍처의 구조와 책임 경계"
OBJECTIVE = "제품 아키텍처 문서를 바탕으로 Plasma의 주요 구조, 상태, 작업 흐름을 독자가 이해할 수 있게 설명한다."
DIRECTION = (
    "소스 수명주기와 장기 작업의 상태 경계에 집중해, 운영자 관점이 아니라 제품 사용자가 연구 자료를 "
    "신뢰하고 재사용하는 흐름을 설명하라. 'MCP가 도메인 정책의 주인이다'라는 해석은 하지 말라. "
    "핵심 경계와 책임을 비교하는 표를 포함하되, 표가 설명을 대체하지 않게 하라."
)
GUIDANCE_PROFILE = "section-brief-cluster-memory-narrative-contract"


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--archive", type=Path, default=base.default_archive().parent / EXPERIMENT_ID)
    parser.add_argument("--arms", nargs="+", choices=ARMS, default=list(ARMS))
    parser.add_argument("--strategies", nargs="+", choices=STRATEGIES, default=list(STRATEGIES))
    parser.add_argument("--workers", type=int, default=2)
    parser.add_argument("--model", default="gpt-5.5")
    parser.add_argument("--effort", default="medium")
    parser.add_argument("--timeout-seconds", type=int, default=7200)
    return parser.parse_args()


def main() -> None:
    args = parse_args()
    archive = args.archive.expanduser().resolve()
    fixture = archive / "fixture/product-architecture.ko.md"
    binaries = {arm: archive / f"bin/plasma-{arm}" for arm in args.arms}
    for path in (fixture, *binaries.values()):
        if not path.is_file():
            raise FileNotFoundError(path)
    write_control(archive, fixture, binaries, args)

    specs = [(arm, strategy) for arm in args.arms for strategy in args.strategies]
    used_ports: set[int] = set()
    port_lock = Lock()
    results: list[dict[str, Any]] = []
    with ThreadPoolExecutor(max_workers=max(1, min(args.workers, len(specs)))) as pool:
        futures = [
            pool.submit(
                run_one,
                archive,
                fixture,
                binaries[arm],
                arm,
                strategy,
                args.model,
                args.effort,
                args.timeout_seconds,
                used_ports,
                port_lock,
            )
            for arm, strategy in specs
        ]
        for future in as_completed(futures):
            result = future.result()
            results.append(result)
            print(json.dumps({"arm": result["arm"], "strategy": result["strategy"], "status": result["status"]}, ensure_ascii=False), flush=True)
    base.write_json(archive / "run-summary.json", {"experiment": EXPERIMENT_ID, "results": results, "completed_at": base.utc_now()})


def write_control(archive: Path, fixture: Path, binaries: dict[str, Path], args: argparse.Namespace) -> None:
    base.write_json_new_or_same(archive / "control.json", {
        "experiment": EXPERIMENT_ID,
        "title": TITLE,
        "objective": OBJECTIVE,
        "direction": DIRECTION,
        "source": str(fixture),
        "source_sha256": base.sha256(fixture),
        "binaries": {arm: {"path": str(path), "sha256": base.sha256(path)} for arm, path in binaries.items()},
        "model": args.model,
        "effort": args.effort,
        "rigor": "strict",
        "generation_guidance_profile": GUIDANCE_PROFILE,
        "post_report_humanize": "enabled",
        "strategies": list(args.strategies),
    })


def run_one(
    archive: Path,
    fixture: Path,
    binary: Path,
    arm: str,
    strategy: str,
    model: str,
    effort: str,
    timeout_seconds: int,
    used_ports: set[int],
    port_lock: Lock,
) -> dict[str, Any]:
    run_root = archive / "runs" / f"{arm}-{strategy}"
    terminal = run_root / "manifest.terminal.json"
    if terminal.exists():
        return json.loads(terminal.read_text(encoding="utf-8"))
    run_root.mkdir(parents=True, exist_ok=False)
    for name in ("state", "logs", "workdir", "fixture"):
        (run_root / name).mkdir()
    source = run_root / "fixture" / fixture.name
    shutil.copy2(fixture, source)
    with port_lock:
        port = base.allocate_port(used_ports)
        connector_port = base.allocate_port(used_ports)
    env = base.isolated_environment(run_root)
    connector_log = (run_root / "logs/liquid2-stub.log").open("xb")
    serve_log = (run_root / "logs/serve.log").open("xb")
    connector = process = None
    manifest: dict[str, Any] = {
        "experiment": EXPERIMENT_ID,
        "arm": arm,
        "strategy": strategy,
        "binary": str(binary),
        "binary_sha256": base.sha256(binary),
        "source_sha256": base.sha256(source),
        "status": "started",
        "started_at": base.utc_now(),
    }
    base.write_json(run_root / "manifest.initial.json", manifest)
    try:
        connector = base.start_connector_stub(connector_port, env, connector_log)
        base.wait_health(f"http://127.0.0.1:{connector_port}", connector, 30)
        process = subprocess.Popen([
            str(binary), "serve", "-db", str(run_root / "state/plasma.db"), "-addr", f"127.0.0.1:{port}",
            "-liquid2-url", f"http://127.0.0.1:{connector_port}", "-local-source-root", f"fixture={source.parent}",
            "-agent", "codex", "-agent-workdir", str(run_root / "workdir"), "-agent-timeout", "0",
        ], env=env, stdout=serve_log, stderr=subprocess.STDOUT)
        base.wait_health(f"http://127.0.0.1:{port}", process, 30)
        mission = base.run_json([
            str(binary), "missions", "create", "-db", str(run_root / "state/plasma.db"),
            "-title", TITLE, "-objective", OBJECTIVE, "-json",
        ], env)
        mission_id = base.find_string(mission, "MissionID", "mission_id")
        base.run_json([
            str(binary), "sources", "attach-local", mission_id, "-db", str(run_root / "state/plasma.db"),
            "-root", "fixture", "-path", source.name, "-title", TITLE,
            "-local-source-root", f"fixture={source.parent}", "-json",
        ], env)
        body = {
            "title": TITLE,
            "report_mode": "long_form",
            "execution_strategy": strategy,
            "agent_executor": "codex",
            "agent_model": model,
            "agent_reasoning_effort": effort,
            "mcp_mode": "auto",
            "rigor_level": "strict",
            "generation_guidance_profile": GUIDANCE_PROFILE,
            "post_report_humanize": "enabled",
            "direction_hint": DIRECTION,
        }
        base.http_json(f"http://127.0.0.1:{port}/api/missions/{mission_id}/reports", body)
        events, status = base.poll_terminal(f"http://127.0.0.1:{port}", mission_id, process, timeout_seconds)
        base.write_json(run_root / "ledger.events.json", {"events": events})
        manifest.update({"mission_id": mission_id, "status": status, "completed_at": base.utc_now()})
        if status == "completed":
            artifact_id = base.final_artifact_id(events)
            report = base.http_bytes(f"http://127.0.0.1:{port}/api/missions/{mission_id}/artifacts/{artifact_id}/download")
            (run_root / "report.md").write_bytes(report)
            manifest.update({"artifact_id": artifact_id, "report_sha256": hashlib.sha256(report).hexdigest()})
        base.write_json(run_root / "manifest.terminal.json", manifest)
        return manifest
    except Exception as exc:
        manifest.update({"status": "failed", "error": str(exc), "completed_at": base.utc_now()})
        base.write_json(run_root / "manifest.terminal.json", manifest)
        return manifest
    finally:
        if process is not None:
            base.stop_process(process)
        if connector is not None:
            base.stop_process(connector)
        serve_log.close()
        connector_log.close()


if __name__ == "__main__":
    main()
