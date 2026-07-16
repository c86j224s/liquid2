#!/usr/bin/env python3
"""Subprocess-only controller for the preregistered issue #110 experiment."""

from __future__ import annotations

import argparse
from concurrent.futures import ThreadPoolExecutor, as_completed
from dataclasses import replace
import hashlib
import json
import os
from pathlib import Path
import random
import shutil
import subprocess
import sys
from threading import Lock
from typing import Mapping

from codex_blind_judge import schema_sha256
from report_plan_mcp.audit import collect_hard_metrics, hard_gate
from report_plan_mcp.builds import export_and_build, sha256_file
from report_plan_mcp.judging import build_blind_packets, calibrate_dimensions, calibrate_with_command, score_packets
from report_plan_mcp.models import (
    BASELINE_COMMIT, CANDIDATE_COMMIT, EXPERIMENT_ID, PREFLIGHT_MODEL, PREREGISTERED_TOPICS, Fixture, RunManifest, RunSpec,
    effort_for_mode, executor_for_mode, freeze_fixture_manifest, load_and_validate_fixtures, model_for_mode,
    validate_provider_efforts, validate_provider_models,
)
from report_plan_mcp.product_path import ProductRunError, assert_public_product_path, execute_product_run, product_commands
from report_plan_mcp.provenance import STRUCTURAL_GATES, assemble_records, blinded_endpoint_values, build_pairs, require_gates
from report_plan_mcp.safety import (
    allocate_port, canonical_archive, ensure_unique_namespace, isolated_environment, namespace,
    snapshot_protected_paths, validate_archive, validate_endpoint, validate_environment, validate_run_paths,
)
from report_plan_mcp.statistics import analyze_confirmatory, freeze_sample_size, write_aggregate


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--action", choices=("preflight", "prepare", "smoke", "calibration-runs", "pilot", "quality", "pilot-packets", "pilot-judge", "packets", "judge", "calibrate", "lock", "analyze", "focused-quality", "focused-recover-quality", "focused-packets", "focused-judge", "focused-analyze"))
    parser.add_argument("--config", type=Path)
    parser.add_argument("--workers", type=int, default=2)
    parser.add_argument("--preflight", action="store_true")
    parser.add_argument("--dry-run", action="store_true")
    parser.add_argument("--topic", choices=PREREGISTERED_TOPICS, default=PREREGISTERED_TOPICS[0])
    parser.add_argument("--replicate", type=int, choices=(1, 2), default=1)
    parser.add_argument("--arm", choices=("baseline", "candidate"), default="candidate")
    parser.add_argument("--mode", choices=("planned", "long_form"), default="planned")
    parser.add_argument("--codex-model")
    parser.add_argument("--nonce", default="preflight")
    args = parser.parse_args()
    if args.preflight:
        args.action = "preflight"
    if not args.action:
        parser.error("--action is required")
    if args.action == "preflight" and not args.dry_run:
        parser.error("preflight requires --dry-run")
    if args.action != "preflight" and args.config is None:
        parser.error("this action requires --config")
    if args.action != "preflight" and args.codex_model is not None:
        parser.error("provider model flags are preflight-only; run actions use the frozen config models")
    return args


def effective_child_environment(env: dict[str, str | None]) -> dict[str, str | None]:
    child = os.environ.copy()
    for key, value in env.items():
        if value is None:
            child.pop(key, None)
        else:
            child[key] = value
    keys = sorted(env)
    script = "import json,os,sys; print(json.dumps({k:os.environ.get(k) for k in sys.argv[1:]}))"
    output = subprocess.check_output([sys.executable, "-c", script, *keys], env=child, text=True)
    return json.loads(output)


def build_manifest(spec: RunSpec, home: Path, inherited: dict[str, str], used_ports: set[int], namespaces: set[str]) -> RunManifest:
    expected_executor = executor_for_mode(spec.mode)
    expected_effort = effort_for_mode(spec.mode, {"codex": "high"})
    if spec.executor != expected_executor or spec.effort != expected_effort:
        raise ValueError("run executor or effort differs from the frozen report-mode mapping")
    archive = validate_archive(canonical_archive(home), home)
    run_namespace = namespace(spec.topic, spec.replicate, spec.arm, spec.mode, spec.nonce)
    ensure_unique_namespace(run_namespace, namespaces)
    run_root = archive / "runs" / run_namespace
    database, artifact, workdir = run_root / "state" / "plasma.db", run_root / "artifacts", run_root / "workdir"
    validate_run_paths(run_root, database, artifact, workdir, archive)
    port = allocate_port(used_ports)
    connector_port = allocate_port(used_ports)
    validate_endpoint("127.0.0.1", port)
    validate_endpoint("127.0.0.1", connector_port)
    connector_url = f"http://127.0.0.1:{connector_port}"
    environment = isolated_environment(run_root, inherited)
    validate_environment(environment, run_root, inherited)
    effective = effective_child_environment(environment)
    if effective != environment:
        raise ValueError("effective child environment differs from the manifest")
    commands = product_commands(
        spec.binary, run_root, port, connector_url, spec.mode, spec.executor, spec.model, spec.effort,
    )
    assert_public_product_path(commands)
    binary_hash = sha256_file(spec.binary) if spec.binary.is_file() else "pending-build"
    return RunManifest(
        experiment=EXPERIMENT_ID, topic=spec.topic, replicate=spec.replicate, arm=spec.arm, mode=spec.mode, executor=spec.executor,
        commit=spec.commit, binary=str(spec.binary), binary_hash=binary_hash, model=spec.model, effort=spec.effort,
        source_policy=spec.source_policy, source_bundle=str(spec.source_bundle), source_hash=spec.source_hash,
        budgets={"tokens": spec.token_budget, "seconds": spec.time_budget_seconds}, selected_session_policy=spec.session_policy,
        database=str(database), artifact_root=str(artifact), workdir=str(workdir), port=port,
        connector_port=connector_port, connector_url=connector_url, namespace=run_namespace,
        child_environment=effective, commands=commands,
    )


def _config(path: Path) -> dict[str, object]:
    value = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(value, dict):
        raise ValueError("config must be an object")
    return value


def _provider_models(config: Mapping[str, object]) -> dict[str, str]:
    if "model" in config:
        raise ValueError("single model config is unsupported; use the closed models mapping")
    return validate_provider_models(config.get("models"))


def _provider_efforts(config: Mapping[str, object]) -> dict[str, str]:
    if "effort" in config:
        raise ValueError("single effort config is unsupported; use the closed efforts mapping")
    return validate_provider_efforts(config.get("efforts"))


def _prepare_gate(config: Mapping[str, object], archive: Path) -> dict[str, object]:
    gate = _config(archive / "control" / "prepare-gate.json")
    required_gate = {
        "passed", "controller_commit", "baseline_commit", "candidate_commit", "models", "efforts", "baseline", "candidate",
        "fixture_manifest_sha256", "smoke_fixture_manifest_sha256", "calibration_fixture_manifest_sha256",
    }
    if set(gate) != required_gate or gate.get("passed") is not True:
        raise ValueError("immutable prepare gate is invalid")
    candidate_commit = str(config.get("candidate_commit", ""))
    controller_commit = str(config.get("controller_commit", ""))
    if gate.get("controller_commit") != controller_commit:
        raise ValueError("prepare gate controller lock does not match the experiment config")
    if gate.get("models") != _provider_models(config) or gate.get("efforts") != _provider_efforts(config):
        raise ValueError("prepare gate provider settings do not match the experiment config")
    if candidate_commit != CANDIDATE_COMMIT:
        raise ValueError("candidate commit does not match the frozen experiment candidate")
    if gate.get("baseline_commit") != BASELINE_COMMIT or gate.get("candidate_commit") != CANDIDATE_COMMIT:
        raise ValueError("prepare gate commit lock does not match the experiment config")
    builds: dict[str, Mapping[str, object]] = {}
    for arm, commit in (("baseline", BASELINE_COMMIT), ("candidate", candidate_commit)):
        build = gate.get(arm)
        if not isinstance(build, Mapping):
            raise ValueError(f"prepare gate omits the {arm} build")
        required = {"arm", "commit", "source_archive", "source_sha256", "binary", "binary_sha256"}
        if not required.issubset(build) or build.get("arm") != arm or build.get("commit") != commit:
            raise ValueError(f"prepare gate {arm} build provenance is invalid")
        if any(not isinstance(build.get(key), str) or not str(build[key]).strip() for key in ("source_sha256", "binary_sha256")):
            raise ValueError(f"prepare gate {arm} hashes are invalid")
        source_archive, binary = Path(str(build["source_archive"])), Path(str(build["binary"]))
        if not source_archive.resolve().is_relative_to(archive.resolve()) or not binary.resolve().is_relative_to(archive.resolve()):
            raise ValueError(f"prepare gate {arm} build paths escape the experiment archive")
        if source_archive.resolve() != (archive / "source-manifests" / f"{arm}-{commit}.tar").resolve() or binary.resolve() != (archive / "bin" / arm / "plasma").resolve():
            raise ValueError(f"prepare gate {arm} build paths differ from the locked layout")
        if not source_archive.is_file() or sha256_file(source_archive) != build["source_sha256"]:
            raise ValueError(f"prepare gate {arm} source archive hash does not match")
        if not binary.is_file() or sha256_file(binary) != build["binary_sha256"]:
            raise ValueError(f"prepare gate {arm} binary hash does not match")
        builds[arm] = build
    if builds["baseline"]["binary_sha256"] == builds["candidate"]["binary_sha256"]:
        raise ValueError("baseline and candidate binary hashes must differ")
    return gate


def _write_new(path: Path, value: object) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("x", encoding="utf-8") as handle:
        json.dump(value, handle, indent=2, sort_keys=True)
        handle.write("\n")


def _prepare(config: Mapping[str, object], repo: Path, archive: Path) -> dict[str, object]:
    _provider_models(config)
    _provider_efforts(config)
    candidate = str(config["candidate_commit"])
    controller = str(config["controller_commit"])
    if candidate != CANDIDATE_COMMIT:
        raise ValueError("candidate commit does not match the frozen experiment candidate")
    if subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=repo, text=True).strip() != controller:
        raise ValueError("controller commit does not match HEAD")
    if subprocess.check_output(["git", "status", "--porcelain"], cwd=repo, text=True).strip():
        raise ValueError("candidate build requires a clean worktree")
    fixtures = load_and_validate_fixtures(Path(str(config["fixture_manifest"])), archive)
    smoke = load_and_validate_fixtures(Path(str(config["smoke_fixture_manifest"])), archive, minimum=1, maximum=1, require_registered=False)
    calibration = load_and_validate_fixtures(Path(str(config["calibration_fixture_manifest"])), archive, minimum=5, maximum=24, require_registered=False)
    if smoke[0].topic in {fixture.topic for fixture in fixtures}:
        raise ValueError("smoke fixture must not overlap quality topics")
    reserved = {fixture.topic for fixture in fixtures} | {smoke[0].topic}
    if any(fixture.topic in reserved for fixture in calibration):
        raise ValueError("calibration fixtures must not overlap smoke or quality topics")
    fixture_hash = freeze_fixture_manifest(fixtures, archive / "fixtures.lock.json")
    smoke_hash = freeze_fixture_manifest(smoke, archive / "smoke-fixture.lock.json")
    calibration_hash = freeze_fixture_manifest(calibration, archive / "calibration-fixtures.lock.json")
    baseline_build = export_and_build(repo, archive, BASELINE_COMMIT, "baseline")
    candidate_build = export_and_build(repo, archive, candidate, "candidate")
    result = {
        "passed": True,
        "controller_commit": controller,
        "baseline_commit": BASELINE_COMMIT,
        "candidate_commit": candidate,
        "models": _provider_models(config),
        "efforts": _provider_efforts(config),
        "fixture_manifest_sha256": fixture_hash,
        "smoke_fixture_manifest_sha256": smoke_hash,
        "calibration_fixture_manifest_sha256": calibration_hash,
        "baseline": baseline_build,
        "candidate": candidate_build,
    }
    if baseline_build["binary_sha256"] == candidate_build["binary_sha256"]:
        raise ValueError("baseline and candidate builds are identical")
    _write_new(archive / "control" / "prepare-gate.json", result)
    return result


def _spec(config: Mapping[str, object], archive: Path, fixture: Fixture, arm: str, mode: str, replicate: int, nonce: str) -> RunSpec:
    gate = _prepare_gate(config, archive)
    commit = str(gate[f"{arm}_commit"])
    executor = executor_for_mode(mode)
    return RunSpec(
        fixture.topic, replicate, arm, mode, executor, commit, archive / "bin" / arm / "plasma",
        model_for_mode(mode, config.get("models")), effort_for_mode(mode, config.get("efforts")), str(config.get("source_policy", "mission-sources-only")),
        int(config.get("token_budget", 120000)), int(config.get("time_budget_seconds", 7200)),
        str(config.get("session_policy", "same_session")), fixture.source_bundle, fixture.source_sha256, nonce,
    )


def _execute_attempts(attempts: list[RunManifest], fixture: Fixture, auth_seeds: Mapping[str, Path]) -> tuple[RunManifest, dict[str, float], list[dict[str, object]]]:
    history: list[dict[str, object]] = []
    for attempt, manifest in enumerate(attempts, start=1):
        try:
            terminal = execute_product_run(manifest, fixture, auth_seeds=auth_seeds)
            events = json.loads((Path(terminal.database).parent.parent / "ledger.events.json").read_text(encoding="utf-8"))["events"]
            metrics = collect_hard_metrics(events, terminal.result_hash is not None, terminal.arm == "candidate", terminal.as_dict())
            history.append({"attempt": attempt, "namespace": manifest.namespace, "classification": "completed"})
            return terminal, metrics, history
        except ProductRunError as exc:
            classification = "itt" if exc.started else "pre_run_infrastructure"
            history.append({"attempt": attempt, "namespace": manifest.namespace, "classification": classification, "kind": exc.kind})
            current = exc.manifest or manifest
            terminal = replace(
                current,
                start_boundary="started:product_cli_mission_create" if exc.started else current.start_boundary,
                terminal_status="itt_failure" if exc.started else "pre_run_failure",
            )
            if exc.started or attempt == 3:
                return terminal, _failed_metrics(terminal.arm == "candidate"), history
    raise AssertionError("attempt loop did not return")


def _execute_specs(
    specs: list[RunSpec], fixture: Fixture, auth_seeds: Mapping[str, Path],
    used_ports: set[int], namespaces: set[str], allocation_lock: Lock,
) -> tuple[RunManifest, dict[str, float], list[dict[str, object]]]:
    history: list[dict[str, object]] = []
    for attempt, spec in enumerate(specs, start=1):
        with allocation_lock:
            manifest = build_manifest(spec, Path.home(), dict(os.environ), used_ports, namespaces)
        try:
            terminal = execute_product_run(manifest, fixture, auth_seeds=auth_seeds)
            events = json.loads((Path(terminal.database).parent.parent / "ledger.events.json").read_text(encoding="utf-8"))["events"]
            try:
                metrics = collect_hard_metrics(events, terminal.result_hash is not None, terminal.arm == "candidate", terminal.as_dict())
                classification = "completed"
            except ValueError as exc:
                metrics = _failed_metrics(terminal.arm == "candidate")
                classification = "completed_audit_failure"
                history.append({"attempt": attempt, "namespace": manifest.namespace, "classification": classification, "kind": str(exc)})
                return terminal, metrics, history
            history.append({"attempt": attempt, "namespace": manifest.namespace, "classification": classification})
            return terminal, metrics, history
        except ProductRunError as exc:
            classification = "itt" if exc.started else "pre_run_infrastructure"
            history.append({"attempt": attempt, "namespace": manifest.namespace, "classification": classification, "kind": exc.kind})
            current = exc.manifest or manifest
            terminal = replace(
                current,
                start_boundary="started:product_cli_mission_create" if exc.started else current.start_boundary,
                terminal_status="itt_failure" if exc.started else "pre_run_failure",
            )
            if exc.started or attempt == 3:
                return terminal, _failed_metrics(terminal.arm == "candidate"), history
        finally:
            with allocation_lock:
                used_ports.discard(manifest.port)
                used_ports.discard(manifest.connector_port)
    raise AssertionError("attempt loop did not return")


def _auth_seeds(config: Mapping[str, object], archive: Path) -> dict[str, Path]:
    seeds = config.get("auth_seeds", {})
    if not isinstance(seeds, Mapping):
        raise ValueError("auth_seeds must be an object")
    result: dict[str, Path] = {}
    for key in ("CODEX_HOME", "CLAUDE_CONFIG_DIR"):
        raw = seeds.get(key)
        if raw is None:
            continue
        source = Path(str(raw)).expanduser().resolve()
        if not source.is_relative_to(archive.resolve()) or not source.is_dir():
            raise ValueError(f"{key} auth seed must be an archive-local directory")
        result[key] = source
    return result


def _judge_environment(config: Mapping[str, object], archive: Path, phase: str) -> dict[str, str]:
    root = archive / "judging" / f"runtime-{phase}"
    overrides = isolated_environment(root, dict(os.environ))
    validate_environment(overrides, root, dict(os.environ))
    seeds = _auth_seeds(config, archive)
    for key, value in overrides.items():
        if not value:
            continue
        target = Path(value)
        if key in seeds:
            target.parent.mkdir(parents=True, exist_ok=True)
            shutil.copytree(seeds[key], target)
        else:
            target.mkdir(parents=True, exist_ok=True)
    environment = dict(os.environ)
    for key, value in overrides.items():
        if value is None:
            environment.pop(key, None)
        else:
            environment[key] = value
    _write_new(archive / "judging" / f"environment-{phase}.json", overrides)
    return environment


def _focused_judge_settings(config: Mapping[str, object]) -> dict[str, str]:
    judge = config.get("focused_judge")
    if not isinstance(judge, Mapping) or set(judge) != {"model", "effort"}:
        raise ValueError("focused_judge must contain exactly model and effort")
    model, effort = judge["model"], judge["effort"]
    if not isinstance(model, str) or not model.strip() or effort != "high":
        raise ValueError("focused_judge requires a non-blank model and high effort")
    return {"model": model.strip(), "effort": "high"}


def _focused_judge_command(settings: Mapping[str, object], rubric: Path) -> list[str]:
    return [sys.executable, str(Path(__file__).with_name("codex_blind_judge.py")), "--model", str(settings["model"]), "--effort", str(settings["effort"]), "--rubric", str(rubric)]


def _focused_controller_commit(repo: Path) -> str:
    commit = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=repo, text=True).strip()
    if subprocess.check_output(["git", "status", "--porcelain"], cwd=repo, text=True).strip():
        raise ValueError("focused quality controller requires a clean worktree")
    return commit


def _sha256(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def _build_focused_schedule(fixtures: tuple[Fixture, ...], seed: int) -> dict[str, object]:
    topics = sorted(fixture.topic for fixture in fixtures)
    if len(topics) != 12 or len(set(topics)) != 12:
        raise ValueError("focused execution schedule requires 12 unique topics")
    entries: list[dict[str, object]] = []
    for mode in ("planned", "long_form"):
        ordered = list(topics)
        random.Random(f"{seed}:{mode}:arm-order").shuffle(ordered)
        for index, topic in enumerate(ordered):
            arms = ["baseline", "candidate"] if index < 6 else ["candidate", "baseline"]
            entries.append({"topic": topic, "mode": mode, "arms": arms})
    random.Random(f"{seed}:pair-order").shuffle(entries)
    return {"seed": seed, "entries": entries}


def _focused_execution_schedule(config: Mapping[str, object], archive: Path, fixtures: tuple[Fixture, ...] | None = None) -> dict[str, object]:
    selected = fixtures or load_and_validate_fixtures(archive / "fixtures.lock.json", archive, minimum=12)[:12]
    schedule = _build_focused_schedule(selected, int(config.get("seed", 110)))
    path = archive / "control" / "focused-execution-schedule.json"
    if path.exists():
        if _config(path) != schedule:
            raise ValueError("focused execution schedule differs from the frozen schedule")
    else:
        _write_new(path, schedule)
    return schedule


def _focused_protocol_values(config: Mapping[str, object], archive: Path, repo: Path) -> dict[str, object]:
    settings = _focused_judge_settings(config)
    schedule = _focused_execution_schedule(config, archive)
    rubric = repo / "plasma/docs/experiments/17-report-plan-mcp-focused-2026-07-14/focused-rubric.md"
    adapter = Path(__file__).with_name("codex_blind_judge.py")
    if not rubric.is_file() or not adapter.is_file():
        raise ValueError("focused rubric or judge adapter is missing")
    return {
        "seed": int(config.get("seed", 110)), "judge": settings,
        "quality_controller_commit": _focused_controller_commit(repo),
        "execution_schedule_sha256": hashlib.sha256(json.dumps(schedule, sort_keys=True).encode()).hexdigest(),
        "rubric_sha256": _sha256(rubric), "adapter_sha256": _sha256(adapter),
        "schema_sha256": schema_sha256(),
    }


def _focused_protocol_lock(config: Mapping[str, object], archive: Path, repo: Path) -> tuple[dict[str, object], str]:
    lock = _focused_protocol_values(config, archive, repo)
    path = archive / "control" / "focused-protocol.lock.json"
    if path.exists():
        if _config(path) != lock:
            raise ValueError("focused protocol lock differs from the requested configuration")
    else:
        _write_new(path, lock)
    return lock, _sha256(path)


def _focused_followup_protocol_lock(config: Mapping[str, object], archive: Path, repo: Path) -> tuple[dict[str, object], str]:
    path = archive / "control" / "focused-protocol.lock.json"
    lock = _config(path)
    current = _focused_protocol_values(config, archive, repo)
    recovery_path = archive / "control" / "focused-quality-recovery.lock.json"
    quality_gate = _config(archive / "control" / "focused-quality-gate.json")
    recovered = quality_gate.get("recovered") is True
    if recovered != recovery_path.exists():
        raise ValueError("focused recovery gate and recovery lock presence differ")
    if not recovered:
        if lock != current:
            raise ValueError("focused protocol lock differs from the requested configuration")
        return lock, _sha256(path)
    recovery = _config(recovery_path)
    required = {
        "recovery_controller_commit", "original_protocol_lock_sha256", "terminal_manifest_sha256",
        "initial_manifest_sha256", "run_intervals_ns", "first_start_mtime_ns",
        "protected_path_mtime_ns", "workers",
    }
    if set(recovery) != required or recovery.get("original_protocol_lock_sha256") != _sha256(path):
        raise ValueError("focused recovery lock is invalid")
    if recovery.get("recovery_controller_commit") != current["quality_controller_commit"]:
        raise ValueError("focused recovery controller differs from the current clean controller")
    for key, value in current.items():
        if key != "quality_controller_commit" and lock.get(key) != value:
            raise ValueError("focused recovery protocol settings differ from the original lock")
    evidence = _focused_run_evidence(archive)
    for key in ("terminal_manifest_sha256", "initial_manifest_sha256", "run_intervals_ns", "workers"):
        if recovery.get(key) != evidence[key]:
            raise ValueError("focused recovery evidence differs from the preserved run artifacts")
    return lock, _sha256(path)


def _focused_packet_lock(archive: Path, protocol_lock_sha256: str) -> dict[str, object]:
    packet_root = archive / "judging" / "focused-packets"
    packets = sorted(packet_root.glob("*.json"))
    mapping = archive / "judging" / "focused-packets-mapping.private.json"
    if not mapping.is_file():
        raise ValueError("focused private packet mapping is missing")
    lock = {
        "protocol_lock_sha256": protocol_lock_sha256,
        "packet_count": len(packets),
        "packets": {path.name: _sha256(path) for path in packets},
        "private_mapping_sha256": _sha256(mapping),
    }
    path = archive / "control" / "focused-packets.lock.json"
    if path.exists():
        if _config(path) != lock:
            raise ValueError("focused packet lock differs from immutable packet files")
    else:
        _write_new(path, lock)
    return lock


def _focused_judge_gate(archive: Path, protocol_lock_sha256: str, packet_lock: Mapping[str, object]) -> dict[str, object]:
    gate = _config(archive / "control" / "focused-judge-gate.json")
    required = {"passed", "attempt", "protocol_lock_sha256", "packet_lock_sha256", "score_manifest_sha256", "score_count"}
    if set(gate) != required or gate.get("passed") is not True or gate.get("protocol_lock_sha256") != protocol_lock_sha256:
        raise ValueError("focused judge completion gate is invalid")
    packet_hash = hashlib.sha256(json.dumps(packet_lock, sort_keys=True).encode()).hexdigest()
    if gate.get("packet_lock_sha256") != packet_hash or gate.get("score_count") != packet_lock.get("packet_count"):
        raise ValueError("focused judge completion gate differs from the packet lock")
    attempt = gate.get("attempt")
    if not isinstance(attempt, int) or attempt < 1:
        raise ValueError("focused judge completion gate attempt is invalid")
    scores = archive / "judging" / "focused-judge" / f"attempt-{attempt}" / "scores"
    manifest = {path.name: _sha256(path) for path in sorted(scores.glob("*.json"))}
    if len(manifest) != int(gate["score_count"]) or hashlib.sha256(json.dumps(manifest, sort_keys=True).encode()).hexdigest() != gate.get("score_manifest_sha256"):
        raise ValueError("focused judge completion gate differs from immutable score files")
    return gate


def _require_focused_records(records: list[dict[str, object]]) -> None:
    if len(records) != 48:
        raise ValueError("focused quality analysis requires exactly 48 runs")
    topics = {str(record.get("topic")) for record in records}
    if len(topics) != 12 or {record.get("replicate") for record in records} != {1}:
        raise ValueError("focused quality analysis requires 12 topics and replicate 1")


def _require_focused_quality_gate(archive: Path) -> None:
    require_gates(archive, *STRUCTURAL_GATES, "focused-quality-gate.json")


def _next_focused_judge_attempt(archive: Path) -> tuple[int, Path]:
    root = archive / "judging" / "focused-judge"
    existing = [int(path.name.removeprefix("attempt-")) for path in root.glob("attempt-*") if path.is_dir() and path.name.removeprefix("attempt-").isdigit()]
    attempt = max(existing, default=0) + 1
    destination = root / f"attempt-{attempt}"
    destination.mkdir(parents=True, exist_ok=False)
    return attempt, destination


def _execute_phase(name: str, config: Mapping[str, object], archive: Path, fixtures: tuple[Fixture, ...], workers: int) -> dict[str, object]:
    if name == "smoke":
        if workers != 2 or len(fixtures) != 1:
            raise ValueError("smoke requires exactly one topic and exactly two workers")
        replicates = (1,)
    elif name == "calibration":
        if not 1 <= workers <= 6 or len(fixtures) < 5:
            raise ValueError("calibration requires at least five frozen topics and at most six workers")
        require_gates(archive, *STRUCTURAL_GATES)
        replicates = (1, 2)
    elif name == "pilot":
        if not 1 <= workers <= 6 or len(fixtures) != 4:
            raise ValueError("pilot requires exactly four topics and at most six workers")
        require_gates(archive, *STRUCTURAL_GATES)
        calibration = _config(archive / "judging" / "calibration.lock.json")
        if not calibration or any(not isinstance(value, Mapping) or value.get("passed") is not True for value in calibration.values()):
            raise ValueError("pilot requires a passed immutable judge calibration lock")
        replicates = (1, 2)
    elif name == "focused-quality":
        if not 1 <= workers <= 6 or len(fixtures) != 12:
            raise ValueError("focused quality requires exactly 12 frozen topics and workers between one and six")
        require_gates(archive, *STRUCTURAL_GATES)
        replicates = (1,)
    else:
        if not 1 <= workers <= 6:
            raise ValueError("quality workers must be between one and six")
        require_gates(archive, *STRUCTURAL_GATES, "pilot-gate.json")
        lock = json.loads((archive / "analysis" / "sample-size.lock.json").read_text(encoding="utf-8"))
        if lock.get("locked") is not True or len(fixtures) != int(lock["final_topics"]) - 4:
            raise ValueError("quality requires all structural gates and the locked topic count")
        replicates = (1, 2)
    configured = config.get("protected_paths", [])
    if not isinstance(configured, list):
        raise ValueError("protected_paths must be a list")
    protected = (
        Path.home() / "research-artifacts/liquid2/plasma/runtime/dev-6002/plasma-ui-user.db",
        Path.home() / "Library/Application Support/Plasma/plasma.db",
        *(Path(str(value)) for value in configured),
    )
    before_protected = snapshot_protected_paths(protected, archive)
    used_ports: set[int] = set()
    namespaces: set[str] = set()
    allocation_lock = Lock()
    auth_seeds = _auth_seeds(config, archive)
    manifests: list[tuple[list[RunManifest], Fixture]] = []
    spec_groups: list[tuple[list[RunSpec], Fixture]] = []
    if name == "focused-quality":
        schedule = _focused_execution_schedule(config, archive, fixtures)
        fixtures_by_topic = {fixture.topic: fixture for fixture in fixtures}
        cells = [
            (fixtures_by_topic[str(entry["topic"])], str(entry["mode"]), str(arm))
            for entry in schedule["entries"]  # type: ignore[index]
            for arm in entry["arms"]  # type: ignore[index]
        ]
    else:
        cells = [(fixture, mode, arm) for fixture in fixtures for mode in ("planned", "long_form") for arm in ("baseline", "candidate")]
    for replicate in replicates:
        for fixture, mode, arm in cells:
            specs = []
            for attempt in range(1, 4):
                specs.append(_spec(config, archive, fixture, arm, mode, replicate, f"{name}-a{attempt}"))
            if name == "focused-quality":
                spec_groups.append((specs, fixture))
            else:
                attempts = [build_manifest(spec, Path.home(), dict(os.environ), used_ports, namespaces) for spec in specs]
                manifests.append((attempts, fixture))
    results: list[dict[str, object]] = []
    with ThreadPoolExecutor(max_workers=workers) as pool:
        if name == "focused-quality":
            futures = {
                pool.submit(_execute_specs, specs, fixture, auth_seeds, used_ports, namespaces, allocation_lock): specs
                for specs, fixture in spec_groups
            }
        else:
            futures = {pool.submit(_execute_attempts, attempts, fixture, auth_seeds): attempts for attempts, fixture in manifests}
        for future in as_completed(futures):
            terminal, metrics, history = future.result()
            results.append({**terminal.as_dict(), "started": terminal.start_boundary.startswith("started:"), "artifact_presence": int(terminal.result_hash is not None), "machine_metrics": metrics, "attempts": history})
    contamination = snapshot_protected_paths(protected, archive) != before_protected
    if contamination:
        for record in results:
            record["machine_metrics"]["isolation_violation"] = 1.0  # type: ignore[index]
    all_success = all(record["terminal_status"] == "completed" and hard_gate(record["machine_metrics"], record["arm"] == "candidate") for record in results)  # type: ignore[arg-type]
    if name == "focused-quality":
        expected = {(fixture.topic, 1, mode, arm) for fixture in fixtures for mode in ("planned", "long_form") for arm in ("baseline", "candidate")}
        actual = {(str(record["topic"]), int(record["replicate"]), str(record["mode"]), str(record["arm"])) for record in results}
        matrix_complete = actual == expected
        pre_run_blocker = any(record["terminal_status"] == "pre_run_failure" for record in results)
        passed = not contamination and matrix_complete and not pre_run_blocker
        summary = {
            "phase": name, "workers": workers, "runs": results, "protected_paths_unchanged": not contamination,
            "matrix_complete": matrix_complete, "all_success": all_success, "passed": passed,
        }
    else:
        summary = {"phase": name, "workers": workers, "runs": results, "protected_paths_unchanged": not contamination, "passed": all_success}
    _write_new(archive / "control" / f"{name}-gate.json", summary)
    return summary


def _failed_metrics(candidate: bool) -> dict[str, float]:
    metrics = {name: 1.0 for name in ("missing_canonical", "duplicate_canonical", "session_violation", "source_read_violation", "ref_scope_violation", "recovery_violation", "isolation_violation")}
    metrics["artifact_presence"] = 0.0
    if candidate:
        metrics.update(fallback_count=1.0, binding_violation=1.0)
    return metrics


def _max_interval_overlap(intervals: list[tuple[int, int]]) -> int:
    events = sorted(((stamp, delta) for start, end in intervals for stamp, delta in ((start, 1), (end, -1))), key=lambda item: (item[0], item[1]))
    active = maximum = 0
    for _, delta in events:
        active += delta
        maximum = max(maximum, active)
    return maximum


def _focused_run_evidence(archive: Path) -> dict[str, object]:
    terminal_paths = sorted((archive / "runs").glob("*focused-quality*/manifest.terminal.json"))
    initial_paths = sorted((archive / "runs").glob("*focused-quality*/manifest.initial.json"))
    if len(terminal_paths) != 48 or len(initial_paths) != 48 or {path.parent for path in initial_paths} != {path.parent for path in terminal_paths}:
        raise ValueError("focused recovery requires 48 matching initial and terminal manifests")
    terminal_by_namespace = {path.parent.name: path for path in terminal_paths}
    initial_by_namespace = {path.parent.name: path for path in initial_paths}
    intervals: dict[str, list[int]] = {}
    for run_namespace in sorted(terminal_by_namespace):
        start = initial_by_namespace[run_namespace].stat().st_mtime_ns
        end = terminal_by_namespace[run_namespace].stat().st_mtime_ns
        if end < start:
            raise ValueError("focused recovery found a terminal timestamp before its initial manifest")
        intervals[run_namespace] = [start, end]
    workers = _max_interval_overlap([(value[0], value[1]) for value in intervals.values()])
    return {
        "terminal_paths": terminal_paths,
        "initial_paths": initial_paths,
        "terminal_manifest_sha256": {name: _sha256(path) for name, path in terminal_by_namespace.items()},
        "initial_manifest_sha256": {name: _sha256(path) for name, path in initial_by_namespace.items()},
        "run_intervals_ns": intervals,
        "workers": workers,
    }


def _recover_focused_quality(config: Mapping[str, object], archive: Path, repo: Path) -> dict[str, object]:
    schedule = _focused_execution_schedule(config, archive)
    protocol_path = archive / "control" / "focused-protocol.lock.json"
    protocol = _config(protocol_path)
    if protocol.get("execution_schedule_sha256") != hashlib.sha256(json.dumps(schedule, sort_keys=True).encode()).hexdigest():
        raise ValueError("focused recovery schedule differs from the original protocol lock")
    evidence = _focused_run_evidence(archive)
    terminal_paths = evidence["terminal_paths"]
    expected = {
        (str(entry["topic"]), 1, str(entry["mode"]), str(arm))
        for entry in schedule["entries"]  # type: ignore[index]
        for arm in entry["arms"]  # type: ignore[index]
    }
    results: list[dict[str, object]] = []
    for path in terminal_paths:  # type: ignore[union-attr]
        terminal = _config(path)
        key = (str(terminal.get("topic")), int(terminal.get("replicate", 0)), str(terminal.get("mode")), str(terminal.get("arm")))
        if key not in expected:
            raise ValueError("focused recovery found a terminal outside the frozen schedule")
        status = terminal.get("terminal_status")
        if status == "completed":
            events = _config(Path(str(terminal["database"])).parent.parent / "ledger.events.json").get("events")
            if not isinstance(events, list):
                raise ValueError("focused recovery found an invalid ledger")
            try:
                metrics = collect_hard_metrics(events, terminal.get("result_hash") is not None, terminal.get("arm") == "candidate", terminal)
                classification, kind = "completed", None
            except ValueError as exc:
                metrics = _failed_metrics(terminal.get("arm") == "candidate")
                classification, kind = "completed_audit_failure", str(exc)
        elif status == "itt_failure":
            metrics = _failed_metrics(terminal.get("arm") == "candidate")
            classification, kind = "itt", "recovered-terminal"
        else:
            raise ValueError("focused recovery cannot admit a pre-run terminal")
        history = {"attempt": 1, "namespace": terminal["namespace"], "classification": classification}
        if kind is not None:
            history["kind"] = kind
        results.append({**terminal, "started": True, "artifact_presence": int(terminal.get("result_hash") is not None), "machine_metrics": metrics, "attempts": [history]})
    actual = {(row["topic"], row["replicate"], row["mode"], row["arm"]) for row in results}
    if actual != expected:
        raise ValueError("focused recovery terminal matrix is incomplete or duplicated")
    initial_paths = evidence["initial_paths"]
    first_start = min(path.stat().st_mtime_ns for path in initial_paths)  # type: ignore[union-attr]
    configured = config.get("protected_paths", [])
    if not isinstance(configured, list):
        raise ValueError("protected_paths must be a list")
    protected = (
        Path.home() / "research-artifacts/liquid2/plasma/runtime/dev-6002/plasma-ui-user.db",
        Path.home() / "Library/Application Support/Plasma/plasma.db",
        *(Path(str(value)) for value in configured),
    )
    if any(not path.expanduser().is_file() or path.expanduser().stat().st_mtime_ns >= first_start for path in protected):
        raise ValueError("focused recovery cannot prove protected paths predate the first run")
    workers = int(evidence["workers"])
    if workers != 6:
        raise ValueError("focused recovery cannot prove the configured six-worker launch")
    recovery = {
        "recovery_controller_commit": _focused_controller_commit(repo),
        "original_protocol_lock_sha256": _sha256(protocol_path),
        "terminal_manifest_sha256": evidence["terminal_manifest_sha256"],
        "initial_manifest_sha256": evidence["initial_manifest_sha256"],
        "run_intervals_ns": evidence["run_intervals_ns"],
        "first_start_mtime_ns": first_start,
        "protected_path_mtime_ns": {str(path.expanduser()): path.expanduser().stat().st_mtime_ns for path in protected},
        "workers": workers,
    }
    _write_new(archive / "control" / "focused-quality-recovery.lock.json", recovery)
    all_success = all(row["terminal_status"] == "completed" and hard_gate(row["machine_metrics"], row["arm"] == "candidate") for row in results)  # type: ignore[arg-type]
    summary = {
        "phase": "focused-quality", "workers": workers, "runs": results,
        "protected_paths_unchanged": True, "matrix_complete": True,
        "all_success": all_success, "passed": True, "recovered": True,
    }
    _write_new(archive / "control" / "focused-quality-gate.json", summary)
    return summary


def main() -> int:
    args = parse_args()
    home, repo = Path.home(), Path(__file__).resolve().parents[3]
    archive = validate_archive(canonical_archive(home), home)
    if args.action == "preflight":
        preview_models = validate_provider_models({
            "codex": args.codex_model if args.codex_model is not None else PREFLIGHT_MODEL,
        })
        commit = BASELINE_COMMIT if args.arm == "baseline" else subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=repo, text=True).strip()
        preview_efforts = {"codex": "high"}
        spec = RunSpec(args.topic, args.replicate, args.arm, args.mode, executor_for_mode(args.mode), commit, archive / "bin" / args.arm / "plasma", model_for_mode(args.mode, preview_models), effort_for_mode(args.mode, preview_efforts), "mission-sources-only", 120000, 7200, "same_session", archive / "fixtures" / args.topic, "locked-before-run", args.nonce)
        print(json.dumps(build_manifest(spec, home, dict(os.environ), set(), set()).as_dict(), indent=2, sort_keys=True))
        return 0
    config = _config(args.config)
    _provider_models(config)
    _provider_efforts(config)
    if args.action != "prepare":
        _prepare_gate(config, archive)
    if args.action == "prepare":
        result = _prepare(config, repo, archive)
    elif args.action in {"smoke", "calibration-runs", "pilot", "quality", "focused-quality"}:
        fixtures = load_and_validate_fixtures(archive / "fixtures.lock.json", archive, minimum=12)
        if args.action == "smoke":
            selected = load_and_validate_fixtures(archive / "smoke-fixture.lock.json", archive, minimum=1, maximum=1, require_registered=False)
        elif args.action == "calibration-runs":
            selected = load_and_validate_fixtures(archive / "calibration-fixtures.lock.json", archive, minimum=5, require_registered=False)
        elif args.action == "pilot":
            selected = fixtures[:4]
        elif args.action == "focused-quality":
            selected = fixtures[:12]
            if len(selected) != 12:
                raise ValueError("focused quality requires exactly 12 frozen quality topics")
            _focused_protocol_lock(config, archive, repo)
        else:
            lock = json.loads((archive / "analysis" / "sample-size.lock.json").read_text(encoding="utf-8"))
            selected = fixtures[4:int(lock["final_topics"])]
        phase = "calibration" if args.action == "calibration-runs" else args.action
        result = _execute_phase(phase, config, archive, selected, args.workers)
    elif args.action == "focused-recover-quality":
        result = _recover_focused_quality(config, archive, repo)
    elif args.action == "focused-packets":
        _require_focused_quality_gate(archive)
        pairs, runs = build_pairs(archive, ("focused-quality",))
        _require_focused_records(runs)
        lock, lock_hash = _focused_followup_protocol_lock(config, archive, repo)
        result = build_blind_packets(pairs, archive / "judging" / "focused-packets", int(lock["seed"]))
        if len(result) != len(pairs):
            raise ValueError("focused packet set differs from completed baseline/candidate pairs")
        _focused_packet_lock(archive, lock_hash)
    elif args.action == "focused-judge":
        _require_focused_quality_gate(archive)
        settings, lock_hash = _focused_followup_protocol_lock(config, archive, repo)
        packet_lock = _focused_packet_lock(archive, lock_hash)
        packets = sorted((archive / "judging" / "focused-packets").glob("*.json"))
        if len(packets) != packet_lock["packet_count"]:
            raise ValueError("focused judge packet files differ from the packet lock")
        attempt, attempt_root = _next_focused_judge_attempt(archive)
        packet_lock_hash = hashlib.sha256(json.dumps(packet_lock, sort_keys=True).encode()).hexdigest()
        _write_new(attempt_root / "attempt.lock.json", {"attempt": attempt, "protocol_lock_sha256": lock_hash, "packet_lock_sha256": packet_lock_hash})
        rubric = repo / "plasma/docs/experiments/17-report-plan-mcp-focused-2026-07-14/focused-rubric.md"
        result = score_packets(_focused_judge_command(settings["judge"], rubric), packets, attempt_root / "scores", _judge_environment(config, archive, f"focused-attempt-{attempt}"))
        if len(result) != len(packets):
            raise ValueError("focused judge did not score every immutable blind packet")
        score_manifest = {path.name: _sha256(path) for path in sorted((attempt_root / "scores").glob("*.json"))}
        _write_new(archive / "control" / "focused-judge-gate.json", {
            "passed": True, "attempt": attempt, "protocol_lock_sha256": lock_hash,
            "packet_lock_sha256": packet_lock_hash,
            "score_manifest_sha256": hashlib.sha256(json.dumps(score_manifest, sort_keys=True).encode()).hexdigest(),
            "score_count": len(score_manifest),
        })
    elif args.action == "focused-analyze":
        _require_focused_quality_gate(archive)
        lock, lock_hash = _focused_followup_protocol_lock(config, archive, repo)
        packet_lock = _focused_packet_lock(archive, lock_hash)
        judge_gate = _focused_judge_gate(archive, lock_hash, packet_lock)
        scores = archive / "judging" / "focused-judge" / f"attempt-{judge_gate['attempt']}" / "scores"
        records = assemble_records(archive, ("focused-quality",), scores, archive / "judging" / "focused-packets-mapping.private.json")
        _require_focused_records(records)
        result = {
            **analyze_confirmatory(records, int(lock["seed"])),
            "seed": lock["seed"], "focused_protocol_lock_sha256": lock_hash,
            "focused_packet_lock_sha256": hashlib.sha256(json.dumps(packet_lock, sort_keys=True).encode()).hexdigest(),
        }
        write_aggregate(result, archive / "analysis" / "focused-aggregate.json")
    elif args.action in {"pilot-packets", "packets"}:
        require_gates(archive, *STRUCTURAL_GATES, "pilot-gate.json")
        phases = ("pilot",) if args.action == "pilot-packets" else ("pilot", "quality")
        if args.action == "packets":
            require_gates(archive, "quality-gate.json")
        pairs, _ = build_pairs(archive, phases)
        destination = archive / "judging" / ("pilot-packets" if args.action == "pilot-packets" else "packets")
        result = build_blind_packets(pairs, destination, int(config.get("seed", 110)))
    elif args.action in {"pilot-judge", "judge"}:
        command = config.get("judge_command")
        if not isinstance(command, list) or not command:
            raise ValueError("judge_command must be a non-empty argv list")
        phase = "pilot" if args.action == "pilot-judge" else "experimental"
        packet_name = "pilot-packets" if args.action == "pilot-judge" else "packets"
        score_name = "pilot-scores" if args.action == "pilot-judge" else "scores"
        require_gates(archive, *STRUCTURAL_GATES, "pilot-gate.json")
        if args.action == "judge":
            require_gates(archive, "quality-gate.json")
        packets = sorted((archive / "judging" / packet_name).glob("*.json"))
        if not packets:
            raise ValueError("immutable blind packets are missing")
        result = score_packets([str(value) for value in command], packets, archive / "judging" / score_name, _judge_environment(config, archive, phase))
    elif args.action == "calibrate":
        command = config.get("judge_command")
        if isinstance(command, list) and command:
            from report_plan_mcp.provenance import build_calibration_packets
            require_gates(archive, *STRUCTURAL_GATES, "calibration-gate.json")
            packets = build_calibration_packets(archive, archive / "judging" / "calibration-packets")
            result = calibrate_with_command([str(value) for value in command], packets, archive / "judging" / "calibration.lock.json", _judge_environment(config, archive, "calibration"))
        else:
            raise ValueError("calibration requires a subprocess judge and archive-owned calibration packets")
        if not all(value["passed"] for value in result.values()):
            raise ValueError("judge calibration failed")
        if not (archive / "judging" / "calibration.lock.json").exists():
            _write_new(archive / "judging" / "calibration.lock.json", result)
    elif args.action == "lock":
        require_gates(archive, *STRUCTURAL_GATES, "pilot-gate.json")
        records = assemble_records(archive, ("pilot",), archive / "judging" / "pilot-scores", archive / "judging" / "pilot-packets-mapping.private.json")
        result = freeze_sample_size(blinded_endpoint_values(records), archive / "analysis" / "sample-size.lock.json")
    else:
        require_gates(archive, *STRUCTURAL_GATES, "pilot-gate.json", "quality-gate.json")
        lock = _config(archive / "analysis" / "sample-size.lock.json")
        if lock.get("locked") is not True:
            raise ValueError("analysis requires the immutable sample-size lock")
        records = assemble_records(archive, ("pilot", "quality"), archive / "judging" / "scores", archive / "judging" / "packets-mapping.private.json")
        if len({record["topic"] for record in records}) != int(lock["final_topics"]):
            raise ValueError("analysis run matrix differs from the sample-size lock")
        result = analyze_confirmatory(records, int(config.get("seed", 110)))
        write_aggregate(result, archive / "analysis" / "aggregate.json")
    print(json.dumps(result, indent=2, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
