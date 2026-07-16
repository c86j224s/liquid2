"""Fail-closed reconstruction of experiment pairs and ITT records."""

from __future__ import annotations

import hashlib
import json
from pathlib import Path
from statistics import mean
from typing import Mapping, Sequence

from .audit import validate_pair
from .judging import FINAL_DIMENSIONS, PLAN_DIMENSIONS
from .models import BASELINE_COMMIT, CANDIDATE_COMMIT


STRUCTURAL_GATES = ("smoke-gate.json",)
MANIFEST_FIELDS = (
    "experiment", "topic", "replicate", "arm", "mode", "executor", "commit", "binary_hash",
    "model", "effort", "source_policy", "source_bundle", "source_hash", "budgets", "selected_session_policy",
    "database", "artifact_root", "workdir", "port", "connector_port", "connector_url", "namespace",
    "child_environment", "mission_id", "process_id", "connector_process_id", "start_boundary",
    "terminal_status", "ledger_hash", "result_hash", "commands",
)


def require_gates(archive: Path, *names: str) -> list[dict[str, object]]:
    gates = []
    for name in names:
        path = archive / "control" / name
        value = _object(path)
        if value.get("passed") is not True:
            raise ValueError(f"required structural gate did not pass: {name}")
        gates.append(value)
    return gates


def build_pairs(
    archive: Path, phases: Sequence[str], *, baseline_commit: str = BASELINE_COMMIT,
    candidate_commit: str = CANDIDATE_COMMIT, modes: Sequence[str] = ("planned", "long_form"),
    phase_topic_counts: Mapping[str, int] | None = None, phase_replicates: Mapping[str, Sequence[int]] | None = None,
    both_arms_plan_mcp: bool = False,
) -> tuple[list[dict[str, object]], list[dict[str, object]]]:
    runs = load_runs(
        archive, phases, baseline_commit=baseline_commit, candidate_commit=candidate_commit,
        modes=modes, phase_topic_counts=phase_topic_counts, phase_replicates=phase_replicates,
    )
    grouped: dict[tuple[str, int, str], dict[str, dict[str, object]]] = {}
    for run in runs:
        key = (str(run["topic"]), int(run["replicate"]), str(run["mode"]))
        arm = str(run["arm"])
        if arm in grouped.setdefault(key, {}):
            raise ValueError(f"duplicate immutable run: {key} {arm}")
        grouped[key][arm] = run
    pairs: list[dict[str, object]] = []
    for key, arms in sorted(grouped.items()):
        if set(arms) != {"baseline", "candidate"}:
            raise ValueError(f"incomplete run pair: {key}")
        validate_pair(arms["baseline"], arms["candidate"])
        if all(arms[arm]["terminal_status"] == "completed" for arm in arms):
            pairs.append({
                "topic": key[0], "replicate": key[1], "mode": key[2],
                "baseline": _judge_payload(archive, arms["baseline"], both_arms_plan_mcp),
                "candidate": _judge_payload(archive, arms["candidate"], both_arms_plan_mcp),
            })
    return pairs, runs


def load_runs(
    archive: Path, phases: Sequence[str], *, baseline_commit: str = BASELINE_COMMIT,
    candidate_commit: str = CANDIDATE_COMMIT, modes: Sequence[str] = ("planned", "long_form"),
    phase_topic_counts: Mapping[str, int] | None = None, phase_replicates: Mapping[str, Sequence[int]] | None = None,
) -> list[dict[str, object]]:
    prepare = _prepare_provenance(archive, baseline_commit, candidate_commit)
    runs: list[dict[str, object]] = []
    for phase in phases:
        gate = require_gates(archive, f"{phase}-gate.json")[0]
        rows = gate.get("runs")
        if not isinstance(rows, list) or not rows:
            raise ValueError(f"{phase} gate has no runs")
        phase_runs: list[dict[str, object]] = []
        for row in rows:
            if not isinstance(row, dict):
                raise ValueError("run gate contains a non-object")
            terminal_path = archive / "runs" / str(row.get("namespace")) / "manifest.terminal.json"
            terminal = _object(terminal_path)
            if any(row.get(field) != terminal.get(field) for field in MANIFEST_FIELDS):
                raise ValueError(f"run gate differs from immutable terminal manifest: {terminal_path}")
            if terminal.get("terminal_status") not in {"completed", "itt_failure"}:
                raise ValueError("pre-run failures cannot enter a locked phase gate")
            record = dict(row)
            arm = str(record.get("arm"))
            build = prepare.get(arm)
            if not isinstance(build, Mapping) or record.get("commit") != build.get("commit") or record.get("binary_hash") != build.get("binary_sha256"):
                raise ValueError("run commit or binary hash differs from the immutable prepare gate")
            record["manifest_sha256"] = _sha256(terminal_path)
            runs.append(record)
            phase_runs.append(record)
        _validate_phase_matrix(phase, phase_runs, modes, phase_topic_counts, phase_replicates)
    _validate_run_matrix(runs, modes)
    return runs


def _prepare_provenance(archive: Path, baseline_commit: str, candidate_commit: str) -> dict[str, object]:
    gate = _object(archive / "control" / "prepare-gate.json")
    locked_candidate = gate.get("candidate_commit")
    if gate.get("passed") is not True or gate.get("baseline_commit") != baseline_commit or locked_candidate != candidate_commit:
        raise ValueError("immutable prepare gate has invalid commit provenance")
    builds: dict[str, Mapping[str, object]] = {}
    for arm, expected_commit in (("baseline", baseline_commit), ("candidate", candidate_commit)):
        build = gate.get(arm)
        if not isinstance(build, Mapping) or build.get("arm") != arm or build.get("commit") != expected_commit:
            raise ValueError(f"immutable prepare gate has invalid {arm} build provenance")
        if any(not isinstance(build.get(key), str) or not str(build[key]).strip() for key in ("source_sha256", "binary_sha256")):
            raise ValueError(f"immutable prepare gate has invalid {arm} hashes")
        builds[arm] = build
    if builds["baseline"]["binary_sha256"] == builds["candidate"]["binary_sha256"]:
        raise ValueError("immutable prepare gate contains identical arm binaries")
    return gate


def assemble_records(
    archive: Path, phases: Sequence[str], scores_root: Path, mapping_path: Path, *,
    baseline_commit: str = BASELINE_COMMIT, candidate_commit: str = CANDIDATE_COMMIT,
    modes: Sequence[str] = ("planned", "long_form"), phase_topic_counts: Mapping[str, int] | None = None,
    phase_replicates: Mapping[str, Sequence[int]] | None = None, both_arms_plan_mcp: bool = False,
) -> list[dict[str, object]]:
    _, runs = build_pairs(
        archive, phases, baseline_commit=baseline_commit, candidate_commit=candidate_commit,
        modes=modes, phase_topic_counts=phase_topic_counts, phase_replicates=phase_replicates,
        both_arms_plan_mcp=both_arms_plan_mcp,
    )
    mappings = json.loads(mapping_path.read_text(encoding="utf-8"))
    if not isinstance(mappings, list):
        raise ValueError("blind mapping must be a list")
    by_run: dict[tuple[str, int, str, str], Mapping[str, object]] = {}
    for mapping in mappings:
        if not isinstance(mapping, dict) or set(mapping) != {"packet_id", "topic", "replicate", "mode", "A", "B"}:
            raise ValueError("blind mapping shape is invalid")
        score = _object(scores_root / f"{mapping['packet_id']}.json")
        if score.get("packet_id") != mapping["packet_id"]:
            raise ValueError("score packet identity mismatch")
        score_arms = score.get("scores")
        if not isinstance(score_arms, dict) or set(score_arms) != {"A", "B"}:
            raise ValueError("score file lacks both blinded arms")
        for label in ("A", "B"):
            arm = mapping[label]
            if arm not in {"baseline", "candidate"}:
                raise ValueError("private arm mapping is invalid")
            by_run[(str(mapping["topic"]), int(mapping["replicate"]), str(mapping["mode"]), str(arm))] = score_arms[label]
    records: list[dict[str, object]] = []
    for run in runs:
        key = (str(run["topic"]), int(run["replicate"]), str(run["mode"]), str(run["arm"]))
        record = dict(run)
        if run["terminal_status"] == "completed":
            flat = by_run.pop(key, None)
            if flat is None:
                record["scores"] = _itt_low_scores()
                record["itt_score_reason"] = "paired-arm-not-completed"
            else:
                record["scores"] = _split_scores(flat)
        elif key in by_run:
            raise ValueError("failed ITT run unexpectedly has judge scores")
        records.append(record)
    if by_run:
        raise ValueError("score files contain runs outside the immutable run manifests")
    return records


def blinded_endpoint_values(records: Sequence[Mapping[str, object]]) -> dict[str, list[float]]:
    values = {name: [] for name in ("planned_final", "planned_plan", "long_form_final", "long_form_plan")}
    grouped: dict[tuple[str, str, str], list[float]] = {}
    for record in records:
        if record.get("terminal_status") != "completed":
            raise ValueError("pilot sample-size inputs require completed judged runs")
        scores = record.get("scores")
        for endpoint, dimensions in (("plan", PLAN_DIMENSIONS), ("final", FINAL_DIMENSIONS)):
            score_map = scores.get(endpoint) if isinstance(scores, Mapping) else None
            if not isinstance(score_map, Mapping) or set(score_map) != set(dimensions):
                raise ValueError("pilot score dimensions are incomplete")
            grouped.setdefault((str(record["topic"]), str(record["mode"]), endpoint), []).append(
                mean(float(score_map[name]) for name in dimensions)
            )
    for (topic, mode, endpoint), composites in grouped.items():
        if len(composites) != 4:
            raise ValueError(f"pilot topic requires two arms and two replicates: {topic} {mode} {endpoint}")
        values[f"{mode}_{endpoint}"].append(mean(composites))
    if any(len(endpoint) != 4 for endpoint in values.values()):
        raise ValueError("sample-size lock requires exactly four pilot topics")
    return values


def build_calibration_packets(archive: Path, destination: Path) -> list[Path]:
    runs = load_runs(archive, ("calibration",))
    if len(runs) < 20 or any(run["terminal_status"] != "completed" for run in runs):
        raise ValueError("calibration requires at least 20 completed immutable product runs")
    destination.mkdir(parents=True, exist_ok=False)
    paths: list[Path] = []
    for run in runs:
        payload = _judge_payload(archive, run)
        packet_id = hashlib.sha256(str(run["manifest_sha256"]).encode()).hexdigest()[:16]
        path = destination / f"{packet_id}.json"
        if path.exists():
            raise ValueError("duplicate calibration manifest hash")
        path.write_text(json.dumps({"packet_id": packet_id, **payload}, ensure_ascii=False, indent=2, sort_keys=True) + "\n", encoding="utf-8")
        paths.append(path)
    return sorted(paths)


def _judge_payload(archive: Path, run: Mapping[str, object], both_arms_plan_mcp: bool = False) -> dict[str, object]:
    root = archive / "runs" / str(run["namespace"])
    artifacts = list((root / "artifacts").glob("*"))
    if len(artifacts) != 1 or _sha256(artifacts[0]) != run.get("result_hash"):
        raise ValueError("collected artifact set/hash does not match the run manifest")
    ledger_path = root / "ledger.events.json"
    ledger = _object(ledger_path)
    events = ledger.get("events")
    if _sha256(ledger_path) != run.get("ledger_hash"):
        raise ValueError("collected ledger hash does not match the run manifest")
    if not isinstance(events, list):
        raise ValueError("run ledger events are missing")
    arm = run.get("arm")
    if arm == "baseline" and not both_arms_plan_mcp:
        if any(isinstance(event, dict) and event.get("EventType") == "report.plan.submitted" for event in events):
            raise ValueError("baseline ledger contains candidate MCP submission")
        created = [event for event in events if isinstance(event, dict) and event.get("EventType") == "report.plan.created"]
        if len(created) != 1 or not isinstance(created[0].get("Payload"), dict):
            raise ValueError("baseline ledger lacks one JSON-return plan")
        plan = created[0]["Payload"].get("plan")
        if not isinstance(plan, dict):
            raise ValueError("baseline JSON-return plan payload is missing")
    elif arm in {"baseline", "candidate"}:
        submitted = [event for event in events if isinstance(event, dict) and event.get("EventType") == "report.plan.submitted"]
        if len(submitted) != 1 or not isinstance(submitted[0].get("Payload"), dict):
            raise ValueError("candidate ledger lacks one MCP submitted plan")
        payload = submitted[0]["Payload"]
        plan = payload.get("plan") or payload.get("normalized_plan")
        if not isinstance(plan, dict):
            raise ValueError("candidate MCP submitted plan payload is missing")
    else:
        raise ValueError("run arm is invalid for blind plan extraction")
    return {"plan": plan, "report": artifacts[0].read_text(encoding="utf-8")}


def _validate_run_matrix(runs: Sequence[Mapping[str, object]], modes: Sequence[str] = ("planned", "long_form")) -> None:
    keys = [(row.get("topic"), row.get("replicate"), row.get("mode"), row.get("arm")) for row in runs]
    if len(keys) != len(set(keys)):
        raise ValueError("run matrix contains duplicate cells")
    for row in runs:
        if row.get("mode") not in set(modes) or row.get("arm") not in {"baseline", "candidate"}:
            raise ValueError("run matrix contains an unknown mode or arm")
        expected = "codex"
        if row.get("executor") != expected:
            raise ValueError("run executor differs from the frozen mode matrix")


def _validate_phase_matrix(
    phase: str, runs: Sequence[Mapping[str, object]], modes: Sequence[str] = ("planned", "long_form"),
    phase_topic_counts: Mapping[str, int] | None = None, phase_replicates: Mapping[str, Sequence[int]] | None = None,
) -> None:
    topics = {str(row["topic"]) for row in runs}
    expected_topics = (phase_topic_counts or {"smoke": 1, "pilot": 4, "focused-quality": 12}).get(phase)
    if expected_topics is not None and len(topics) != expected_topics:
        raise ValueError(f"{phase} run gate has the wrong topic count")
    replicates = tuple((phase_replicates or {}).get(phase, (1,) if phase in {"smoke", "focused-quality"} else (1, 2)))
    expected = {(topic, replicate, mode, arm) for topic in topics for replicate in replicates for mode in modes for arm in ("baseline", "candidate")}
    actual = {(str(row["topic"]), int(row["replicate"]), str(row["mode"]), str(row["arm"])) for row in runs}
    if actual != expected:
        raise ValueError(f"{phase} run gate matrix is incomplete or contains extra cells")


def _split_scores(value: object) -> dict[str, dict[str, float]]:
    if not isinstance(value, Mapping) or set(value) != set(PLAN_DIMENSIONS + FINAL_DIMENSIONS):
        raise ValueError("judge score dimensions are incomplete")
    return {
        "plan": {name: float(value[name]) for name in PLAN_DIMENSIONS},
        "final": {name: float(value[name]) for name in FINAL_DIMENSIONS},
    }


def _itt_low_scores() -> dict[str, dict[str, float]]:
    return {
        "plan": {name: 1.0 for name in PLAN_DIMENSIONS},
        "final": {name: 1.0 for name in FINAL_DIMENSIONS},
    }


def _object(path: Path) -> dict[str, object]:
    try:
        value = json.loads(path.read_text(encoding="utf-8"))
    except (FileNotFoundError, json.JSONDecodeError) as exc:
        raise ValueError(f"required immutable artifact is missing or invalid: {path}") from exc
    if not isinstance(value, dict):
        raise ValueError(f"required artifact is not an object: {path}")
    return value


def _sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()
