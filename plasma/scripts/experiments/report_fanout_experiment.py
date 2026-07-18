#!/usr/bin/env python3
"""Issue #103 section-fanout long-form report experiment runner."""

from __future__ import annotations

import argparse
from concurrent.futures import ThreadPoolExecutor, as_completed
from dataclasses import dataclass
from datetime import datetime, timezone
import hashlib
import json
import math
import os
from pathlib import Path
import random
import shutil
import socket
import subprocess
import sys
from threading import Lock
import time
from typing import Any
from urllib import request


EXPERIMENT_ID = "21-report-fanout-2026-07-16"
SOURCE_FIXTURE_EXPERIMENT = "17-report-plan-mcp-focused-2026-07-14"
ARMS = ("serial", "section_fanout")


@dataclass(frozen=True)
class Fixture:
    topic: str
    title: str
    objective: str
    source_bundle: Path
    source_sha256: str


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--action", choices=("prepare", "run", "analyze", "packets"), required=True)
    parser.add_argument("--workers", type=int, default=2)
    parser.add_argument("--limit", type=int, default=24)
    parser.add_argument("--model", default="gpt-5.5")
    parser.add_argument("--effort", default="medium")
    parser.add_argument("--timeout-seconds", type=int, default=7200)
    parser.add_argument("--archive", type=Path, default=default_archive())
    parser.add_argument("--source-fixtures", type=Path, default=default_source_archive())
    parser.add_argument("--seed", type=int, default=10321)
    return parser.parse_args()


def default_archive() -> Path:
    return Path.home() / "research-artifacts/liquid2/plasma/experiments" / EXPERIMENT_ID


def default_source_archive() -> Path:
    return Path.home() / "research-artifacts/liquid2/plasma/experiments" / SOURCE_FIXTURE_EXPERIMENT


def repo_root() -> Path:
    return Path(__file__).resolve().parents[3]


def plasma_root() -> Path:
    return Path(__file__).resolve().parents[2]


def prepare(args: argparse.Namespace) -> None:
    archive = args.archive.expanduser().resolve()
    source = args.source_fixtures.expanduser().resolve()
    archive.mkdir(parents=True, exist_ok=True)
    (archive / "bin").mkdir(exist_ok=True)
    fixtures = load_source_fixtures(source)
    write_json_new_or_same(archive / "fixtures.lock.json", {"fixtures": [fixture_to_json(copy_fixture(fixture, archive)) for fixture in fixtures]})
    binary = archive / "bin" / "plasma"
    subprocess.run(["go", "build", "-o", str(binary), "./cmd/plasma"], cwd=plasma_root(), check=True)
    write_json(archive / "control.prepare.json", {
        "experiment": EXPERIMENT_ID,
        "repo": str(repo_root()),
        "git_head": git("rev-parse", "HEAD"),
        "git_dirty": bool(git("status", "--porcelain")),
        "binary": str(binary),
        "binary_sha256": sha256(binary),
        "model_default": args.model,
        "effort_default": args.effort,
        "prepared_at": utc_now(),
    })
    print(json.dumps({"archive": str(archive), "fixtures": len(fixtures), "binary": str(binary)}, ensure_ascii=False))


def load_source_fixtures(source: Path) -> list[Fixture]:
    manifest = json.loads((source / "fixtures.lock.json").read_text(encoding="utf-8"))
    rows = manifest.get("fixtures")
    if not isinstance(rows, list) or len(rows) < 24:
        raise ValueError("source fixture lock must contain at least 24 fixtures")
    fixtures: list[Fixture] = []
    for row in rows[:24]:
        path = Path(str(row["source_bundle"])).expanduser().resolve()
        if not path.is_file():
            raise ValueError(f"fixture source missing: {path}")
        digest = str(row["source_sha256"])
        if sha256(path) != digest:
            raise ValueError(f"fixture hash mismatch: {path}")
        fixtures.append(Fixture(str(row["topic"]), str(row["title"]), str(row["objective"]), path, digest))
    return fixtures


def copy_fixture(fixture: Fixture, archive: Path) -> Fixture:
    target = archive / "fixtures" / fixture.source_bundle.name
    target.parent.mkdir(parents=True, exist_ok=True)
    if target.exists():
        if sha256(target) != fixture.source_sha256:
            raise ValueError(f"existing copied fixture differs: {target}")
    else:
        shutil.copy2(fixture.source_bundle, target)
    return Fixture(fixture.topic, fixture.title, fixture.objective, target, fixture.source_sha256)


def load_fixtures(archive: Path, limit: int) -> list[Fixture]:
    manifest = json.loads((archive / "fixtures.lock.json").read_text(encoding="utf-8"))
    fixtures = []
    for row in manifest["fixtures"][:limit]:
        path = Path(row["source_bundle"]).expanduser().resolve()
        fixtures.append(Fixture(row["topic"], row["title"], row["objective"], path, row["source_sha256"]))
    return fixtures


def run(args: argparse.Namespace) -> None:
    archive = args.archive.expanduser().resolve()
    if not (archive / "bin/plasma").is_file():
        prepare(args)
    fixtures = load_fixtures(archive, args.limit)
    specs = [(fixture, arm) for fixture in fixtures for arm in ARMS]
    rng = random.Random(args.seed)
    rng.shuffle(specs)
    used_ports: set[int] = set()
    port_lock = Lock()
    results = []
    with ThreadPoolExecutor(max_workers=max(1, args.workers)) as pool:
        futures = [pool.submit(run_one, archive, fixture, arm, args.model, args.effort, args.timeout_seconds, used_ports, port_lock) for fixture, arm in specs]
        for future in as_completed(futures):
            result = future.result()
            results.append(result)
            print(json.dumps({"topic": result["topic"], "arm": result["arm"], "status": result["status"]}, ensure_ascii=False), flush=True)
    write_json(archive / "run-summary.json", {"completed_at": utc_now(), "results": results})


def run_one(archive: Path, fixture: Fixture, arm: str, model: str, effort: str, timeout_seconds: int, used_ports: set[int], port_lock: Lock) -> dict[str, Any]:
    run_root = archive / "runs" / f"{fixture.topic}-{arm}"
    terminal = run_root / "manifest.terminal.json"
    if terminal.exists():
        return json.loads(terminal.read_text(encoding="utf-8"))
    run_root.mkdir(parents=True, exist_ok=False)
    for path in ("state", "artifacts", "logs", "workdir", "fixture"):
        (run_root / path).mkdir()
    source = run_root / "fixture" / fixture.source_bundle.name
    shutil.copy2(fixture.source_bundle, source)
    binary = archive / "bin" / "plasma"
    with port_lock:
        port = allocate_port(used_ports)
        connector_port = allocate_port(used_ports)
    env = isolated_environment(run_root)
    connector_log = (run_root / "logs/liquid2-stub.log").open("xb")
    serve_log = (run_root / "logs/serve.log").open("xb")
    connector = process = None
    started = utc_now()
    manifest = {
        "experiment": EXPERIMENT_ID, "topic": fixture.topic, "arm": arm, "model": model, "effort": effort,
        "database": str(run_root / "state/plasma.db"), "run_root": str(run_root), "port": port,
        "connector_port": connector_port, "binary": str(binary), "binary_sha256": sha256(binary),
        "strategy": "" if arm == "serial" else "section_fanout", "status": "started", "started_at": started,
    }
    write_json(run_root / "manifest.initial.json", manifest)
    try:
        connector = start_connector_stub(connector_port, env, connector_log)
        wait_health(f"http://127.0.0.1:{connector_port}", connector, 30)
        process = subprocess.Popen([
            str(binary), "serve", "-db", manifest["database"], "-addr", f"127.0.0.1:{port}",
            "-liquid2-url", f"http://127.0.0.1:{connector_port}",
            "-local-source-root", f"fixture={source.parent}", "-agent", "codex",
            "-agent-workdir", str(run_root / "workdir"), "-agent-timeout", "0",
        ], env=env, stdout=serve_log, stderr=subprocess.STDOUT)
        wait_health(f"http://127.0.0.1:{port}", process, 30)
        mission = run_json([
            str(binary), "missions", "create", "-db", manifest["database"], "-title", fixture.title,
            "-objective", fixture.objective, "-json",
        ], env)
        mission_id = find_string(mission, "MissionID", "mission_id")
        run_json([
            str(binary), "sources", "attach-local", mission_id, "-db", manifest["database"],
            "-root", "fixture", "-path", source.name, "-title", fixture.title,
            "-local-source-root", f"fixture={source.parent}", "-json",
        ], env)
        body = {
            "title": fixture.title, "report_mode": "long_form", "agent_executor": "codex",
            "agent_model": model, "agent_reasoning_effort": effort,
            "post_report_humanize": "disabled", "report_session_policy": "same_session",
        }
        if arm == "section_fanout":
            body["execution_strategy"] = "section_fanout"
        http_json(f"http://127.0.0.1:{port}/api/missions/{mission_id}/reports", body)
        events, status = poll_terminal(f"http://127.0.0.1:{port}", mission_id, process, timeout_seconds)
        write_json(run_root / "ledger.events.json", {"events": events})
        manifest |= {"mission_id": mission_id, "status": status, "completed_at": utc_now()}
        if status == "completed":
            artifact_id = final_artifact_id(events)
            report = http_bytes(f"http://127.0.0.1:{port}/api/missions/{mission_id}/artifacts/{artifact_id}/download")
            (run_root / "report.md").write_bytes(report)
            manifest |= {"artifact_id": artifact_id, "report_sha256": hashlib.sha256(report).hexdigest()}
        write_json(run_root / "metrics.json", collect_metrics(events, run_root / "report.md"))
        write_json(run_root / "manifest.terminal.json", manifest)
        return manifest
    except Exception as exc:
        manifest |= {"status": "failed", "error": str(exc), "completed_at": utc_now()}
        write_json(run_root / "manifest.terminal.json", manifest)
        return manifest
    finally:
        if process is not None:
            stop_process(process)
        if connector is not None:
            stop_process(connector)
        serve_log.close()
        connector_log.close()


def analyze(args: argparse.Namespace) -> None:
    archive = args.archive.expanduser().resolve()
    records = []
    for manifest_path in sorted((archive / "runs").glob("*/manifest.terminal.json")):
        manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
        metrics_path = manifest_path.parent / "metrics.json"
        metrics = json.loads(metrics_path.read_text(encoding="utf-8")) if metrics_path.exists() else {}
        records.append(manifest | {"metrics": metrics})
    by_topic = {}
    for record in records:
        by_topic.setdefault(record["topic"], {})[record["arm"]] = record
    pairs = []
    for topic, arms in sorted(by_topic.items()):
        if all(arm in arms and arms[arm]["status"] == "completed" for arm in ARMS):
            serial, fanout = arms["serial"], arms["section_fanout"]
            pairs.append({
                "topic": topic,
                "serial_wall_seconds": serial["metrics"].get("wall_seconds"),
                "fanout_wall_seconds": fanout["metrics"].get("wall_seconds"),
                "speedup_seconds": serial["metrics"].get("wall_seconds", 0) - fanout["metrics"].get("wall_seconds", 0),
                "serial_words": serial["metrics"].get("final_word_count"),
                "fanout_words": fanout["metrics"].get("final_word_count"),
                "serial_preservation_ratio": serial["metrics"].get("preservation_ratio"),
                "fanout_preservation_ratio": fanout["metrics"].get("preservation_ratio"),
            })
    speed_diffs = [pair["speedup_seconds"] for pair in pairs if isinstance(pair["speedup_seconds"], (int, float)) and pair["speedup_seconds"] != 0]
    result = {
        "experiment": EXPERIMENT_ID,
        "records": len(records),
        "paired_completed": len(pairs),
        "failures": [record for record in records if record.get("status") != "completed"],
        "speed_sign_p_one_sided": exact_one_sided_sign_test(sum(1 for value in speed_diffs if value > 0), sum(1 for value in speed_diffs if value < 0)),
        "median_speedup_seconds": median(speed_diffs),
        "pairs": pairs,
    }
    write_json(archive / "analysis/aggregate.json", result)
    print(json.dumps(result, indent=2, ensure_ascii=False))


def packets(args: argparse.Namespace) -> None:
    archive = args.archive.expanduser().resolve()
    analysis = json.loads((archive / "analysis/aggregate.json").read_text(encoding="utf-8"))
    out = archive / "judging/packets"
    out.mkdir(parents=True, exist_ok=True)
    mapping = {}
    rng = random.Random(args.seed)
    for pair in analysis["pairs"]:
        topic = pair["topic"]
        labels = list(ARMS)
        rng.shuffle(labels)
        packet = {"packet_id": f"{EXPERIMENT_ID}-{topic}", "topic": topic, "replicate": 1, "mode": "long_form"}
        for label, arm in zip(("A", "B"), labels):
            report = (archive / "runs" / f"{topic}-{arm}" / "report.md").read_text(encoding="utf-8")
            packet[label] = {"report_markdown": report}
            mapping[f"{topic}:{label}"] = arm
        write_json(out / f"{topic}.json", packet)
    write_json(archive / "judging/private-mapping.json", mapping)
    print(json.dumps({"packets": len(analysis["pairs"]), "path": str(out)}, ensure_ascii=False))


def collect_metrics(events: list[dict[str, Any]], report_path: Path) -> dict[str, Any]:
    pending = first_event(events, "report.draft.pending")
    terminal = first_event(events, "report.artifact.created") or first_event(events, "report.draft.failed")
    payload = terminal.get("Payload", {}) if terminal else {}
    report = report_path.read_text(encoding="utf-8") if report_path.exists() else ""
    return {
        "wall_seconds": event_delta_seconds(pending, terminal),
        "plan_duration_ms": stage_duration(events, "report.plan.created"),
        "section_sum_duration_ms": sum_stage_duration(events, "report.section.created"),
        "section_max_duration_ms": max_stage_duration(events, "report.section.created"),
        "part_sum_duration_ms": sum_stage_duration(events, "report.part.created"),
        "part_max_duration_ms": max_stage_duration(events, "report.part.created"),
        "final_duration_ms": payload.get("duration_ms"),
        "section_count": payload.get("section_count"),
        "part_count": payload.get("part_count"),
        "section_word_count": payload.get("section_word_count"),
        "final_word_count": len(report.split()),
        "preservation_ratio": payload.get("preservation_ratio"),
    }


def first_event(events: list[dict[str, Any]], event_type: str) -> dict[str, Any] | None:
    for event in events:
        if event.get("EventType") == event_type:
            return event
    return None


def event_delta_seconds(start: dict[str, Any] | None, end: dict[str, Any] | None) -> float | None:
    if not start or not end:
        return None
    started = parse_time(start.get("CreatedAt") or start.get("created_at"))
    ended = parse_time(end.get("CreatedAt") or end.get("created_at"))
    if not started or not ended:
        return None
    return max(0.0, (ended - started).total_seconds())


def parse_time(value: object) -> datetime | None:
    if not isinstance(value, str) or not value:
        return None
    return datetime.fromisoformat(value.replace("Z", "+00:00"))


def stage_duration(events: list[dict[str, Any]], event_type: str) -> int | None:
    event = first_event(events, event_type)
    if not event:
        return None
    value = event.get("Payload", {}).get("duration_ms")
    return int(value) if isinstance(value, (int, float)) else None


def sum_stage_duration(events: list[dict[str, Any]], event_type: str) -> int:
    return sum(int(event.get("Payload", {}).get("duration_ms") or 0) for event in events if event.get("EventType") == event_type)


def max_stage_duration(events: list[dict[str, Any]], event_type: str) -> int:
    values = [int(event.get("Payload", {}).get("duration_ms") or 0) for event in events if event.get("EventType") == event_type]
    return max(values) if values else 0


def exact_one_sided_sign_test(successes: int, failures: int) -> float | None:
    n = successes + failures
    if n == 0:
        return None
    observed = successes
    return sum(math.comb(n, k) for k in range(observed, n + 1)) / (2**n)


def median(values: list[float]) -> float | None:
    if not values:
        return None
    ordered = sorted(values)
    mid = len(ordered) // 2
    if len(ordered) % 2:
        return ordered[mid]
    return (ordered[mid - 1] + ordered[mid]) / 2


def isolated_environment(run_root: Path) -> dict[str, str]:
    env = os.environ.copy()
    env["HOME"] = str(run_root / "home")
    env["TMPDIR"] = str(run_root / "tmp")
    env["CODEX_HOME"] = str(run_root / "provider/codex")
    for key in ("HOME", "TMPDIR", "CODEX_HOME"):
        Path(env[key]).mkdir(parents=True, exist_ok=True)
    source_codex = Path(os.environ.get("CODEX_HOME", Path.home() / ".codex")).expanduser()
    seed_codex_home(source_codex, Path(env["CODEX_HOME"]))
    return env


def seed_codex_home(source: Path, target: Path) -> None:
    if not source.exists():
        return
    for dirname in ("sessions", "log", "tmp", ".tmp"):
        (target / dirname).mkdir(parents=True, exist_ok=True)
    for filename in ("auth.json", "config.toml", "fleet.config.toml", "installation_id", "models_cache.json", ".personality_migration"):
        source_path = source / filename
        target_path = target / filename
        if not source_path.exists() or target_path.exists():
            continue
        target_path.symlink_to(source_path)


def allocate_port(used_ports: set[int]) -> int:
    for port in range(6400, 6500):
        if port in used_ports:
            continue
        with socket.socket() as probe:
            try:
                probe.bind(("127.0.0.1", port))
            except OSError:
                continue
        used_ports.add(port)
        return port
    raise RuntimeError("no experiment port available")


def start_connector_stub(port: int, env: dict[str, str], log: object) -> subprocess.Popen[bytes]:
    script = """
import http.server, json, sys
class Handler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        body = json.dumps({"status":"isolated"}).encode()
        self.send_response(200 if self.path in ("/", "/health", "/api/health") else 503)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers(); self.wfile.write(body)
    def log_message(self, fmt, *args):
        print(fmt % args, flush=True)
http.server.ThreadingHTTPServer(("127.0.0.1", int(sys.argv[1])), Handler).serve_forever()
"""
    return subprocess.Popen([sys.executable, "-u", "-c", script, str(port)], env=env, stdout=log, stderr=subprocess.STDOUT)


def wait_health(base: str, process: subprocess.Popen[bytes], timeout: int) -> None:
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        if process.poll() is not None:
            raise RuntimeError(f"server exited during health check: {process.returncode}")
        try:
            http_json(f"{base}/api/health")
            return
        except Exception:
            time.sleep(0.2)
    raise RuntimeError("health check timed out")


def poll_terminal(base: str, mission_id: str, process: subprocess.Popen[bytes], timeout: int) -> tuple[list[dict[str, Any]], str]:
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        if process.poll() is not None:
            raise RuntimeError(f"server exited before terminal state: {process.returncode}")
        payload = http_json(f"{base}/api/missions/{mission_id}/events")
        events = payload.get("events")
        if not isinstance(events, list):
            raise RuntimeError("events response omitted events")
        kinds = [event.get("EventType") for event in events if isinstance(event, dict)]
        if "report.artifact.created" in kinds:
            return events, "completed"
        if "report.draft.failed" in kinds:
            return events, "failed"
        time.sleep(1)
    raise RuntimeError("report polling timed out")


def final_artifact_id(events: list[dict[str, Any]]) -> str:
    event = first_event(events, "report.artifact.created")
    if not event or not isinstance(event.get("Payload"), dict):
        raise RuntimeError("final artifact event missing")
    value = event["Payload"].get("artifact_id")
    if not isinstance(value, str) or not value:
        raise RuntimeError("artifact id missing")
    return value


def http_json(url: str, body: dict[str, Any] | None = None) -> dict[str, Any]:
    data = None if body is None else json.dumps(body).encode()
    headers = {} if data is None else {"Content-Type": "application/json"}
    with request.urlopen(request.Request(url, data=data, headers=headers), timeout=30) as response:
        value = json.load(response)
    if not isinstance(value, dict):
        raise RuntimeError("HTTP response is not JSON object")
    return value


def http_bytes(url: str) -> bytes:
    with request.urlopen(url, timeout=60) as response:
        return response.read()


def run_json(command: list[str], env: dict[str, str]) -> dict[str, Any]:
    completed = subprocess.run(command, env=env, check=True, capture_output=True, text=True)
    value = json.loads(completed.stdout)
    if not isinstance(value, dict):
        raise RuntimeError("CLI output is not JSON object")
    return value


def find_string(value: Any, *keys: str) -> str:
    if isinstance(value, dict):
        for key in keys:
            found = value.get(key)
            if isinstance(found, str) and found:
                return found
        for nested in value.values():
            try:
                return find_string(nested, *keys)
            except RuntimeError:
                pass
    raise RuntimeError(f"missing string key: {keys}")


def stop_process(process: subprocess.Popen[bytes]) -> None:
    if process.poll() is not None:
        return
    process.terminate()
    try:
        process.wait(timeout=5)
    except subprocess.TimeoutExpired:
        process.kill()
        process.wait(timeout=5)


def fixture_to_json(fixture: Fixture) -> dict[str, str]:
    return {"topic": fixture.topic, "title": fixture.title, "objective": fixture.objective, "source_bundle": str(fixture.source_bundle), "source_sha256": fixture.source_sha256}


def write_json(path: Path, value: Any) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, indent=2, sort_keys=True, ensure_ascii=False) + "\n", encoding="utf-8")


def write_json_new_or_same(path: Path, value: Any) -> None:
    encoded = json.dumps(value, indent=2, sort_keys=True, ensure_ascii=False) + "\n"
    path.parent.mkdir(parents=True, exist_ok=True)
    if path.exists() and path.read_text(encoding="utf-8") != encoded:
        raise RuntimeError(f"existing file differs: {path}")
    if not path.exists():
        path.write_text(encoded, encoding="utf-8")


def sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def utc_now() -> str:
    return datetime.now(timezone.utc).replace(tzinfo=None).isoformat(timespec="seconds") + "Z"


def git(*args: str) -> str:
    return subprocess.check_output(["git", *args], cwd=repo_root(), text=True).strip()


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
