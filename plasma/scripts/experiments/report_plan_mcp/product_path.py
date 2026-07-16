"""Public product-path command manifest; it never imports product internals."""

from __future__ import annotations

from dataclasses import replace
import hashlib
import json
from pathlib import Path
import shutil
import subprocess
import time
from typing import Any, Mapping
from urllib import error

from .models import Fixture, RunManifest
from .runtime import (
    ProductRunError, child_environment as _child_environment, find_string as _find_string,
    http_bytes as _http_bytes, http_json as _http_json, post_json as _post_json,
    run_json as _run_json, start_connector_stub as _start_connector_stub,
    stop_process as _stop_process, wait_health as _wait_health,
)


def product_commands(
    binary: Path, run_root: Path, port: int, connector_url: str,
    mode: str, executor: str, model: str, effort: str,
) -> tuple[tuple[str, ...], ...]:
    base = f"http://127.0.0.1:{port}"
    database = run_root / "state" / "plasma.db"
    fixture = run_root / "fixture"
    return (
        (str(binary), "missions", "create", "-db", str(database), "-title", "{topic_title}", "-objective", "{objective}", "-json"),
        (str(binary), "sources", "attach-local", "{mission_id}", "-db", str(database), "-root", "fixture", "-path", ".", "-title", "{source_title}", "-local-source-root", f"fixture={fixture}", "-json"),
        (str(binary), "serve", "-db", str(database), "-addr", f"127.0.0.1:{port}", "-liquid2-url", connector_url, "-local-source-root", f"fixture={fixture}", "-agent", "codex,claude", "-agent-workdir", str(run_root / "workdir")),
        ("curl", "--fail", "--json", json.dumps({
            "title": "{report_title}", "report_mode": mode, "agent_executor": executor,
            "agent_model": model, "agent_reasoning_effort": effort,
        }, separators=(",", ":")), f"{base}/api/missions/{{mission_id}}/reports"),
        ("curl", "--fail", f"{base}/api/missions/{{mission_id}}"),
        ("curl", "--fail", f"{base}/api/missions/{{mission_id}}/events"),
        (str(binary), "mcp"),
    )


def assert_public_product_path(commands: tuple[tuple[str, ...], ...]) -> None:
    flat = "\n".join(" ".join(command) for command in commands)
    required = ("missions create", "sources attach-local", " serve ", "/reports", "/events", " mcp")
    if any(value not in flat for value in required):
        raise ValueError("command manifest omits a public product boundary")
    if "internal/" in flat or "_test" in flat:
        raise ValueError("command manifest contains an internal or test bypass")


def execute_product_run(
    manifest: RunManifest,
    fixture: Fixture,
    crash_after: str | None = None,
    auth_seeds: Mapping[str, Path] | None = None,
    completion_log_marker: str | None = None,
) -> RunManifest:
    if _sha256(Path(manifest.binary)) != manifest.binary_hash:
        raise ProductRunError("binary hash differs from the frozen manifest", kind="build")
    if _sha256(fixture.source_bundle) != manifest.source_hash:
        raise ProductRunError("fixture hash differs from the frozen manifest", kind="directory")
    run_root = Path(manifest.database).parent.parent
    if run_root.exists():
        raise ProductRunError("immutable run root already exists")
    for path in (Path(manifest.database).parent, Path(manifest.artifact_root), Path(manifest.workdir), run_root / "fixture", run_root / "logs"):
        path.mkdir(parents=True, exist_ok=False)
    for key, seed in (auth_seeds or {}).items():
        target = manifest.child_environment.get(key)
        if not target:
            raise ProductRunError(f"auth seed has no isolated target: {key}", kind="environment")
        shutil.copytree(seed, Path(target))
    source = run_root / "fixture" / fixture.source_bundle.name
    shutil.copy2(fixture.source_bundle, source)
    environment = _child_environment(manifest.child_environment)
    initial = run_root / "manifest.initial.json"
    _write_new_json(initial, manifest.as_dict())
    serve = [
        manifest.binary, "serve", "-db", manifest.database, "-addr", f"127.0.0.1:{manifest.port}",
        "-liquid2-url", manifest.connector_url, "-local-source-root", f"fixture={source.parent}",
        "-agent", "codex,claude", "-agent-workdir", manifest.workdir,
    ]
    connector_log = (run_root / "logs" / "liquid2-stub.log").open("xb")
    serve_log = (run_root / "logs" / "serve.log").open("xb")
    connector: subprocess.Popen[bytes] | None = None
    process: subprocess.Popen[bytes] | None = None
    try:
        connector = _start_connector_stub(manifest.connector_port, environment, connector_log)
        _wait_health(manifest.connector_url, connector, 30, started=False)
        process = subprocess.Popen(serve, env=environment, stdout=serve_log, stderr=subprocess.STDOUT)
        manifest = replace(manifest, process_id=process.pid, connector_process_id=connector.pid)
        base = f"http://127.0.0.1:{manifest.port}"
        _wait_health(base, process, 30, started=False)

        # The ITT boundary is the first durable product mutation, after isolated infrastructure is ready.
        manifest = replace(manifest, start_boundary="started:product_cli_mission_create")
        mission = _run_json([
            manifest.binary, "missions", "create", "-db", manifest.database, "-title", fixture.title,
            "-objective", fixture.objective, "-json",
        ], environment, started=True)
        try:
            mission_id = _find_string(mission, "MissionID", "mission_id")
        except ProductRunError as exc:
            exc.started = True
            raise
        manifest = replace(manifest, mission_id=mission_id)
        _run_json([
            manifest.binary, "sources", "attach-local", mission_id, "-db", manifest.database,
            "-root", "fixture", "-path", source.name, "-title", fixture.title,
            "-local-source-root", f"fixture={source.parent}", "-json",
        ], environment, started=True)
        _post_json(f"{base}/api/missions/{mission_id}/reports", {
            "title": fixture.title, "report_mode": manifest.mode, "agent_executor": manifest.executor,
            "agent_model": manifest.model, "agent_reasoning_effort": manifest.effort,
            "report_session_policy": manifest.selected_session_policy,
        }, started=True)
        events, status = _poll_terminal(
            base, mission_id, process, manifest.budgets["seconds"], crash_after,
            run_root / "logs/serve.log", completion_log_marker,
        )
        ledger_path = run_root / "ledger.events.json"
        _write_new_json(ledger_path, {"events": events})
        manifest = replace(manifest, ledger_hash=_sha256(ledger_path))
        if status.startswith("crashed:"):
            terminal = replace(manifest, terminal_status=status)
            _write_new_json(run_root / "manifest.terminal.json", terminal.as_dict())
            return terminal
        artifact_id = _artifact_id(events)
        artifact = _http_bytes(f"{base}/api/missions/{mission_id}/artifacts/{artifact_id}/download")
        artifact_path = Path(manifest.artifact_root) / f"{artifact_id}.bin"
        with artifact_path.open("xb") as handle:
            handle.write(artifact)
        result_hash = hashlib.sha256(artifact).hexdigest()
        terminal = replace(manifest, terminal_status="completed", result_hash=result_hash)
        _write_new_json(run_root / "manifest.terminal.json", terminal.as_dict())
        return terminal
    except (OSError, error.URLError) as exc:
        started = manifest.start_boundary.startswith("started:")
        message = "product run failed after the ITT boundary" if started else "isolated product infrastructure failed before the ITT boundary"
        failure = ProductRunError(message, started=started, kind="runtime" if started else "health")
        failure.manifest = manifest
        _write_failure_manifest(run_root, manifest, failure.started)
        raise failure from exc
    except ProductRunError as exc:
        exc.manifest = manifest
        _write_failure_manifest(run_root, manifest, exc.started)
        raise
    finally:
        if process is not None:
            _stop_process(process)
        if connector is not None:
            _stop_process(connector)
        serve_log.close()
        connector_log.close()


def _poll_terminal(
    base: str, mission_id: str, process: subprocess.Popen[bytes], timeout: int, crash_after: str | None,
    completion_log: Path | None = None, completion_log_marker: str | None = None,
) -> tuple[list[dict[str, Any]], str]:
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        if process.poll() is not None:
            raise ProductRunError(f"serve exited before terminal state: {process.returncode}", started=True)
        payload = _http_json(f"{base}/api/missions/{mission_id}/events")
        events = payload.get("events")
        if not isinstance(events, list):
            raise ProductRunError("events endpoint omitted events", started=True)
        kinds = [event.get("EventType") for event in events if isinstance(event, dict)]
        if crash_after and crash_after in kinds:
            _stop_process(process)
            return events, f"crashed:{crash_after}"
        marker_ready = completion_log_marker is None or (
            completion_log is not None and completion_log_marker in completion_log.read_text(encoding="utf-8")
        )
        if "report.artifact.created" in kinds and marker_ready:
            return events, "completed"
        if "report.draft.failed" in kinds:
            raise ProductRunError("report reached failed terminal state", started=True)
        time.sleep(0.1)
    raise ProductRunError("report polling timed out", started=True)


def _artifact_id(events: list[dict[str, Any]]) -> str:
    matches = [event for event in events if event.get("EventType") == "report.artifact.created"]
    if len(matches) != 1 or not isinstance(matches[0].get("Payload"), dict):
        raise ProductRunError("exactly one artifact event is required")
    artifact_id = matches[0]["Payload"].get("artifact_id")
    if not isinstance(artifact_id, str) or not artifact_id:
        raise ProductRunError("artifact event omitted artifact_id")
    return artifact_id


def _write_new_json(path: Path, value: object) -> None:
    with path.open("x", encoding="utf-8") as handle:
        json.dump(value, handle, indent=2, sort_keys=True)
        handle.write("\n")


def _write_failure_manifest(run_root: Path, manifest: RunManifest, started: bool) -> None:
    path = run_root / "manifest.terminal.json"
    if not path.exists():
        status = "itt_failure" if started else "pre_run_failure"
        _write_new_json(path, replace(manifest, terminal_status=status).as_dict())


def _sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()
