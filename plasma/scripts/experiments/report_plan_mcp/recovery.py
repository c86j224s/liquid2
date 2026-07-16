"""Process-level crash/restart injections using public HTTP observations."""

from __future__ import annotations

from pathlib import Path
import subprocess

from .models import RunManifest
from .product_path import ProductRunError, _artifact_id, _child_environment, _http_bytes, _poll_terminal, _start_connector_stub, _stop_process, _wait_health, _write_new_json

RECOVERY_CASES = {
    "after_submit_before_validated_exit": "old submission remains stale and cannot advance progress",
    "after_canonical": "canonical plan is reused without duplicate creation",
    "after_first_section": "completed stages and final artifact are not duplicated",
}

RECOVERY_EVENT = {
    "after_submit_before_validated_exit": "report.plan.submitted",
    "after_canonical": "report.plan.created",
    "after_first_section": "report.section.created",
}


def classify_failure(started: bool, kind: str) -> str:
    if started:
        return "itt"
    if kind in {"build", "directory", "environment", "port", "health"}:
        return "pre_run_infrastructure"
    return "itt"


def replacement_allowed(attempt: int, classification: str) -> bool:
    return classification == "pre_run_infrastructure" and attempt <= 2


def validate_crash_observation(case: str, events: list[dict[str, object]]) -> None:
    kinds = [event.get("EventType") for event in events]
    target = RECOVERY_EVENT[case]
    if target not in kinds:
        raise ProductRunError(f"crash target was not observed: {target}")
    if case == "after_submit_before_validated_exit" and "report.plan.created" in kinds:
        raise ProductRunError("submit crash missed the pre-canonical window")


def audit_recovery(case: str, before: list[dict[str, object]], after: list[dict[str, object]]) -> None:
    for event_type in ("report.plan.created", "report.artifact.created"):
        if sum(event.get("EventType") == event_type for event in after) != 1:
            raise ProductRunError(f"recovery produced missing or duplicate {event_type}", started=True)
    for event_type in ("report.section.created", "report.part.created"):
        events = [event for event in after if event.get("EventType") == event_type]
        identities = [_stage_identity(event) for event in events]
        if any(not identity for identity in identities) or len(identities) != len(set(identities)):
            raise ProductRunError(f"recovery duplicated {event_type}", started=True)
    if case == "after_submit_before_validated_exit":
        old_ids = {event.get("EventID") for event in before if event.get("EventType") == "report.plan.submitted"}
        canonical = next(event for event in after if event.get("EventType") == "report.plan.created")
        payload = canonical.get("Payload")
        pointer = payload.get("plan_submission") if isinstance(payload, dict) else None
        if not isinstance(pointer, dict) or pointer.get("submission_event_id") in old_ids:
            raise ProductRunError("stale submission was promoted during recovery", started=True)


def _stage_identity(event: dict[str, object]) -> str:
    payload = event.get("Payload")
    if not isinstance(payload, dict):
        return ""
    for key in ("section_id", "part_id", "frame_id", "stage_id"):
        value = payload.get(key)
        if isinstance(value, str) and value:
            return value
    artifact_id = payload.get("artifact_id")
    if isinstance(artifact_id, str) and artifact_id:
        return artifact_id
    if "section_index" in payload:
        return f"{payload.get('part_index')}:{payload.get('section_index')}"
    if "part_index" in payload:
        return str(payload["part_index"])
    return ""


def resume_crashed_run(manifest: RunManifest, mission_id: str) -> dict[str, object]:
    run_root = Path(manifest.database).parent.parent
    environment = _child_environment(manifest.child_environment)
    fixture = run_root / "fixture"
    command = recovery_command(manifest, fixture)
    log = (run_root / "logs" / "serve.recovery.log").open("xb")
    connector_log = (run_root / "logs" / "liquid2-stub.recovery.log").open("xb")
    connector = _start_connector_stub(manifest.connector_port, environment, connector_log)
    process = subprocess.Popen(command, env=environment, stdout=log, stderr=subprocess.STDOUT)
    base = f"http://127.0.0.1:{manifest.port}"
    try:
        _wait_health(manifest.connector_url, connector, 30, started=True)
        _wait_health(base, process, 30, started=True)
        events, status = _poll_terminal(base, mission_id, process, manifest.budgets["seconds"], None)
        if status != "completed":
            raise ProductRunError("recovery did not complete")
        artifact_id = _artifact_id(events)
        artifact = _http_bytes(f"{base}/api/missions/{mission_id}/artifacts/{artifact_id}/download")
        result = {"status": status, "events": events, "artifact_id": artifact_id, "artifact_bytes": len(artifact)}
        _write_new_json(run_root / "recovery.result.json", result)
        return result
    finally:
        _stop_process(process)
        _stop_process(connector)
        log.close()
        connector_log.close()


def recovery_command(manifest: RunManifest, fixture: Path) -> list[str]:
    return [
        manifest.binary, "serve", "-db", manifest.database, "-addr", f"127.0.0.1:{manifest.port}",
        "-liquid2-url", manifest.connector_url, "-local-source-root", f"fixture={fixture}",
        "-agent", "codex,claude", "-agent-workdir", manifest.workdir,
    ]
