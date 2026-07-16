"""Machine hard-gate helpers for product-path results."""

from __future__ import annotations

import hashlib
from pathlib import Path
from typing import Mapping

from .models import FORBIDDEN_PORTS, executor_for_mode


REQUIRED_ZERO = (
    "missing_canonical",
    "duplicate_canonical",
    "session_violation",
    "source_read_violation",
    "ref_scope_violation",
    "recovery_violation",
    "isolation_violation",
)


def hard_gate(metrics: Mapping[str, float], candidate: bool) -> bool:
    keys = REQUIRED_ZERO + (("fallback_count", "binding_violation") if candidate else ())
    required = set(keys) | {"artifact_presence"}
    if not required.issubset(metrics):
        raise ValueError(f"hard-gate metrics missing: {sorted(required - set(metrics))}")
    return all(metrics[key] == 0 for key in keys) and metrics["artifact_presence"] == 1


def audit_lineage(events: list[Mapping[str, object]]) -> bool:
    submitted = [event for event in events if event.get("EventType") == "report.plan.submitted"]
    canonical = [event for event in events if event.get("EventType") == "report.plan.created"]
    if len(submitted) != 1 or len(canonical) != 1:
        return False
    submission_payload = _payload(submitted[0])
    canonical_payload = _payload(canonical[0])
    pointer = canonical_payload.get("plan_submission")
    if not isinstance(pointer, Mapping):
        return False
    return (
        pointer.get("submission_event_id") == submitted[0].get("EventID")
        and pointer.get("plan_hash") == submission_payload.get("plan_hash")
        and pointer.get("tool_session_id") == submission_payload.get("tool_session_id")
    )


def _payload(event: Mapping[str, object]) -> Mapping[str, object]:
    payload = event.get("Payload")
    if not isinstance(payload, Mapping):
        raise ValueError("ledger event payload is missing or not decoded")
    return payload


def collect_hard_metrics(
    events: list[Mapping[str, object]], artifact_present: bool, candidate: bool, manifest: Mapping[str, object],
) -> dict[str, float]:
    canonical_count = sum(event.get("EventType") == "report.plan.created" for event in events)
    required_types = {"report.draft.pending", "report.plan.created", "report.artifact.created"}
    present_types = {str(event.get("EventType")) for event in events}
    if not required_types.issubset(present_types):
        raise ValueError(f"required events missing: {sorted(required_types - present_types)}")
    canonical = [event for event in events if event.get("EventType") == "report.plan.created"]
    canonical_payload = _payload(canonical[0]) if len(canonical) == 1 else {}
    returned_session = canonical_payload.get("returned_agent_session_id")
    actual_session = canonical_payload.get("agent_session_id")
    if not isinstance(returned_session, str) or not returned_session or returned_session != actual_session:
        raise ValueError("canonical provider-session provenance is missing or inconsistent")
    if not _source_read_trace(events):
        raise ValueError("source-read trace is missing")
    isolation_ok = _manifest_isolation(manifest, events)
    lineage_ok = audit_lineage(events) if candidate else True
    if candidate and not _candidate_order(events):
        lineage_ok = False
    metrics = {
        "missing_canonical": float(canonical_count == 0),
        "duplicate_canonical": float(canonical_count > 1),
        "session_violation": 0.0,
        "source_read_violation": 0.0,
        "ref_scope_violation": 0.0 if not candidate or lineage_ok else 1.0,
        "recovery_violation": 0.0,
        "isolation_violation": 0.0 if isolation_ok else 1.0,
        "artifact_presence": float(artifact_present),
    }
    if candidate:
        metrics["fallback_count"] = 0.0 if lineage_ok else 1.0
        metrics["binding_violation"] = 0.0 if lineage_ok else 1.0
    return metrics


def _source_read_trace(events: list[Mapping[str, object]]) -> bool:
    for event in events:
        if event.get("EventType") == "source.observed":
            return True
        if event.get("EventType") != "mcp.tool.called":
            continue
        tool = _payload(event).get("tool_name")
        if tool in {"plasma.sources.read", "plasma.research.read"}:
            return True
    return False


def _candidate_order(events: list[Mapping[str, object]]) -> bool:
    submitted = [event for event in events if event.get("EventType") == "report.plan.submitted"]
    canonical = [event for event in events if event.get("EventType") == "report.plan.created"]
    if len(submitted) != 1 or len(canonical) != 1:
        return False
    try:
        submitted_index, canonical_index = events.index(submitted[0]), events.index(canonical[0])
    except ValueError:
        return False
    return submitted_index < canonical_index


def _manifest_isolation(manifest: Mapping[str, object], events: list[Mapping[str, object]]) -> bool:
    required = {"database", "artifact_root", "workdir", "namespace", "port", "connector_port", "connector_url", "process_id", "connector_process_id", "executor", "mode", "child_environment", "mission_id"}
    if not required.issubset(manifest):
        raise ValueError(f"isolation manifest missing: {sorted(required - set(manifest))}")
    database = Path(str(manifest["database"])).resolve()
    root = database.parent.parent
    ports = (int(manifest["port"]), int(manifest["connector_port"]))
    if root.name != manifest["namespace"] or len(set(ports)) != 2 or any(port in FORBIDDEN_PORTS or not 6200 <= port <= 6299 for port in ports):
        return False
    if manifest["connector_url"] != f"http://127.0.0.1:{ports[1]}":
        return False
    process_ids = (manifest["process_id"], manifest["connector_process_id"])
    if any(not isinstance(value, int) or value <= 0 for value in process_ids) or process_ids[0] == process_ids[1]:
        return False
    if manifest["executor"] != executor_for_mode(str(manifest["mode"])):
        return False
    for key in ("database", "artifact_root", "workdir"):
        if not Path(str(manifest[key])).resolve().is_relative_to(root):
            return False
    environment = manifest["child_environment"]
    if not isinstance(environment, Mapping) or any(value is not None and not Path(str(value)).resolve().is_relative_to(root) for value in environment.values()):
        return False
    mission_id = manifest["mission_id"]
    return bool(mission_id) and all(event.get("MissionID") == mission_id for event in events)


def arm_order(topic: str, replicate: int, mode: str, seed: int) -> tuple[str, str]:
    digest = hashlib.sha256(f"{seed}:{topic}:{replicate}:{mode}".encode()).digest()
    return ("candidate", "baseline") if digest[0] & 1 else ("baseline", "candidate")


def validate_pair(baseline: Mapping[str, object], candidate: Mapping[str, object]) -> None:
    fixed = ("topic", "replicate", "mode", "executor", "source_hash", "model", "effort", "source_policy", "budgets", "selected_session_policy")
    if any(baseline.get(key) != candidate.get(key) for key in fixed):
        raise ValueError("paired run conditions differ")
    if baseline.get("executor") != executor_for_mode(str(baseline.get("mode"))):
        raise ValueError("paired run executor differs from the frozen mode matrix")
    if not isinstance(baseline.get("model"), str) or not str(baseline["model"]).strip():
        raise ValueError("paired run model is blank")
    if baseline.get("arm") != "baseline" or candidate.get("arm") != "candidate":
        raise ValueError("paired run arms are invalid")
    if baseline.get("commit") == candidate.get("commit"):
        raise ValueError("baseline and candidate commits must differ")
    if baseline.get("binary_hash") == candidate.get("binary_hash"):
        raise ValueError("baseline and candidate binary hashes must differ")
