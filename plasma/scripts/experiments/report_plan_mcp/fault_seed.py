"""Public CLI/Web state preparation for MCP fault cases."""

from __future__ import annotations

import json
from pathlib import Path
import subprocess
import time
from typing import Mapping
from urllib import request

from .runtime import http_json, start_connector_stub, stop_process, wait_health
from .safety import FIXED_ENV, allocate_port, validate_environment


def seed_public_state(
    binary: Path, case_root: Path, commands: list[object], case: Mapping[str, object],
    environment: Mapping[str, str], used_ports: set[int],
) -> tuple[list[dict[str, object]], dict[str, str]]:
    if not case_root.is_absolute():
        raise ValueError("stateful fault case requires an absolute isolated case_root")
    case_root.mkdir(parents=True, exist_ok=False)
    validate_environment(environment, case_root, environment)
    materialize_isolation_environment(environment)
    evidence: list[dict[str, object]] = []
    bindings = {"case_root": str(case_root), "binary": str(binary)}
    allowed = {"missions", "sources", "reports", "mcp"}
    for raw in commands:
        if not isinstance(raw, list) or not raw:
            raise ValueError("seed command must be an argv list")
        argv = [render_string(str(value), bindings) for value in raw]
        if argv[0] != str(binary) or len(argv) < 2 or argv[1] not in allowed:
            raise ValueError("fault seed must use a public plasma CLI/product boundary")
        completed = subprocess.run(argv, text=True, capture_output=True, env=environment)
        if completed.returncode != 0:
            raise RuntimeError(f"fault seed command failed for {case_root.name}: {completed.stderr.strip()}")
        evidence.append({"boundary": argv[1], "returncode": completed.returncode})
        if argv[1] == "missions":
            bindings["mission_id"] = find_string(json.loads(completed.stdout), "MissionID", "mission_id")
    if "mission_id" not in bindings:
        raise ValueError("stateful fault seed must create a mission through the public CLI")
    mode = str(case.get("report_mode", "planned"))
    executor = str(case.get("agent_executor", "codex" if mode == "planned" else "claude"))
    pending_id = seed_web_pending(binary, case_root, bindings["mission_id"], mode, executor, environment, used_ports)
    bindings["pending_event_id"] = pending_id
    evidence.append({"boundary": "web_report_start", "pending_event_id": pending_id, "report_mode": mode, "agent_executor": executor})
    return evidence, bindings


def seed_web_pending(
    binary: Path, case_root: Path, mission_id: str, mode: str, executor: str,
    environment: Mapping[str, str], used_ports: set[int],
) -> str:
    port, connector_port = allocate_port(used_ports), allocate_port(used_ports)
    blocker = case_root / "provider-blocker.py"
    blocker.write_text("#!/usr/bin/env python3\nimport sys\nsys.stdin.read()\n", encoding="utf-8")
    blocker.chmod(0o700)
    validate_environment(environment, case_root, environment)
    materialize_isolation_environment(environment)
    run_env = dict(environment)
    connector_log = (case_root / "connector.log").open("xb")
    serve_log = (case_root / "serve.log").open("xb")
    connector = start_connector_stub(connector_port, run_env, connector_log)
    command = [
        str(binary), "serve", "-db", str(case_root / "state.db"), "-addr", f"127.0.0.1:{port}",
        "-liquid2-url", f"http://127.0.0.1:{connector_port}", "-agent", "codex,claude",
        "-codex-command", str(blocker), "-claude-command", str(blocker), "-agent-workdir", str(case_root / "workdir"),
    ]
    process = subprocess.Popen(command, env=run_env, stdout=serve_log, stderr=subprocess.STDOUT)
    base = f"http://127.0.0.1:{port}"
    try:
        wait_health(f"http://127.0.0.1:{connector_port}", connector, 30)
        wait_health(base, process, 30)
        payload = json.dumps({"title": "Fault seed", "report_mode": mode, "agent_executor": executor}).encode()
        with request.urlopen(request.Request(f"{base}/api/missions/{mission_id}/reports", data=payload, headers={"Content-Type": "application/json"}), timeout=10):
            pass
        deadline = time.monotonic() + 10
        while time.monotonic() < deadline:
            events = http_json(f"{base}/api/missions/{mission_id}/events").get("events")
            if isinstance(events, list):
                pending = [event.get("EventID") for event in events if isinstance(event, dict) and event.get("EventType") == "report.draft.pending"]
                if len(pending) == 1 and isinstance(pending[0], str):
                    return pending[0]
            time.sleep(0.05)
        raise RuntimeError("public Web fault seed did not expose one pending event")
    finally:
        stop_process(process)
        stop_process(connector)
        serve_log.close()
        connector_log.close()


def materialize_isolation_environment(environment: Mapping[str, str]) -> None:
    for key, value in environment.items():
        if key in FIXED_ENV or key.startswith("XDG_"):
            Path(value).mkdir(parents=True, exist_ok=True)


def render_string(value: str, bindings: Mapping[str, str]) -> str:
    for key, replacement in bindings.items():
        value = value.replace("{" + key + "}", replacement)
    return value


def render_value(value: object, bindings: Mapping[str, str]) -> object:
    if isinstance(value, str):
        return render_string(value, bindings)
    if isinstance(value, list):
        return [render_value(item, bindings) for item in value]
    if isinstance(value, dict):
        return {key: render_value(item, bindings) for key, item in value.items()}
    return value


def find_string(value: object, *keys: str) -> str:
    if isinstance(value, dict):
        for key in keys:
            found = value.get(key)
            if isinstance(found, str) and found:
                return found
        for nested in value.values():
            try:
                return find_string(nested, *keys)
            except ValueError:
                pass
    raise ValueError(f"seed output omitted {keys}")
