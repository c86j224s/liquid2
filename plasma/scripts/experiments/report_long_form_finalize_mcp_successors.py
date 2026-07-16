#!/usr/bin/env python3
"""Locked analysis and operational successors for issue #110 experiments 19/20."""

from __future__ import annotations

import argparse
from concurrent.futures import ThreadPoolExecutor, as_completed
from contextlib import contextmanager
from dataclasses import dataclass, replace
import hashlib
import json
import os
from pathlib import Path
import shutil
import stat
import subprocess
from threading import Lock
from typing import Iterator, Mapping, Sequence

import report_long_form_finalize_mcp_experiment as predecessor
import report_plan_mcp.safety as safety
import report_plan_mcp_experiment as base
from report_plan_mcp.audit import hard_gate
from report_plan_mcp.builds import export_and_build, sha256_file
from report_plan_mcp.judging import FINAL_DIMENSIONS
from report_plan_mcp.models import Fixture, RunSpec, freeze_fixture_manifest, load_and_validate_fixtures
from report_plan_mcp.product_path import ProductRunError, execute_product_run
from report_plan_mcp.provenance import assemble_records
from report_plan_mcp.statistics import assemble_itt, guardrail, percentile_lower, topic_endpoint_differences, write_aggregate


ANALYSIS_ID = "19-report-long-form-finalize-itt-analysis-2026-07-14"
OPERATIONAL_ID = "20-report-long-form-finalize-operational-reliability-2026-07-14"
SOURCE_ID = "18-report-long-form-finalize-mcp-2026-07-14"
BASELINE_COMMIT = "1b6239805f2dde41f7aaab36d8025812623da5a6"
SOURCE_CANDIDATE_COMMIT = "4bc3ac07fab93f31d9447c0a83802f6628bd9623"
CANDIDATE_COMMIT = "8a054e6d7d1e50a9ebeb72b6bf6b933303264dc1"
ARCHIVE_BASE = Path("research-artifacts/liquid2/plasma/experiments")
SOURCE_ARCHIVE_SHA256 = "10df1d5d3c37cf4492ba80a818c0f63ed207bd2b1ecacaae8c24c3f8b290694f"
SOURCE_PUBLIC_SHA256 = "54126e8ad29db8bd225faf8a6d5c9037367282f2a91bc22fdef86a505fd436bf"
PREDECESSOR_ARCHIVE_SHA256 = "165db30bb8538bc9b3e646822574d33d8ff5c9619d06b5b3fe1ced9147c39eee"
PREDECESSOR_PUBLIC_SHA256 = "c70d3e0a8eecb471979b295b989dcfecff78bf61b67efda429b4c48fd41e1f2f"
SEED = 110
DRAWS = 10_000
MARGIN = -0.25
COMPLETENESS_MEAN_MARGIN = -0.50
COMPLETENESS_LOW_RATE_INCREASE = 0.10
PRODUCT_PATHS = ("plasma/internal", "plasma/cmd", "plasma/go.mod", "plasma/go.sum")


@dataclass(frozen=True)
class Target:
    experiment: str
    action_group: str

    def root(self, home: Path) -> Path:
        return (home / ARCHIVE_BASE / self.experiment).resolve()

    @property
    def suffix(self) -> Path:
        return ARCHIVE_BASE / self.experiment


ANALYSIS = Target(ANALYSIS_ID, "analysis")
OPERATIONAL = Target(OPERATIONAL_ID, "operational")


def _repo() -> Path:
    return Path(__file__).resolve().parents[3]


def _git(*args: str) -> str:
    return subprocess.check_output(["git", *args], cwd=_repo(), text=True).strip()


def _object(path: Path) -> dict[str, object]:
    value = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(value, dict):
        raise ValueError(f"required object is invalid: {path}")
    return value


def _write_new(path: Path, value: object) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("x", encoding="utf-8") as handle:
        json.dump(value, handle, indent=2, sort_keys=True)
        handle.write("\n")


def _sha256(path: Path) -> str:
    return sha256_file(path)


def _inventory(root: Path) -> str:
    if not root.is_dir():
        raise ValueError(f"inventory root is missing: {root}")
    digest = hashlib.sha256()
    for path in sorted(root.rglob("*")):
        mode = path.lstat().st_mode
        if stat.S_ISLNK(mode):
            raise ValueError(f"inventory contains a symlink: {path}")
        if stat.S_ISREG(mode):
            digest.update(str(path.relative_to(root)).encode())
            digest.update(str(path.stat().st_size).encode())
            digest.update(_sha256(path).encode())
    return digest.hexdigest()


def _lstat_tree(root: Path) -> dict[str, object]:
    if not root.is_dir() or root.is_symlink():
        raise ValueError(f"lstat root must be a real directory: {root}")
    files = 0
    for path in (root, *sorted(root.rglob("*"))):
        metadata = path.lstat()
        if stat.S_ISLNK(metadata.st_mode):
            raise ValueError(f"symlink is prohibited: {path}")
        if stat.S_ISREG(metadata.st_mode):
            files += 1
            if metadata.st_nlink != 1:
                raise ValueError(f"hard-linked regular file is prohibited: {path}")
    return {"passed": True, "regular_file_count": files, "inventory_sha256": _inventory(root)}


def _source_root(home: Path) -> Path:
    return (home / ARCHIVE_BASE / SOURCE_ID).resolve()


def _other(target: Target) -> Target:
    return OPERATIONAL if target is ANALYSIS else ANALYSIS


@contextmanager
def action_context(target: Target, home: Path) -> Iterator[Path]:
    root, other_root = target.root(home), _other(target).root(home)
    other_before = _inventory(other_root) if other_root.exists() else None
    saved = (
        predecessor.EXPERIMENT_ID, predecessor.ARCHIVE_SUFFIX,
        base.EXPERIMENT_ID, safety.EXPERIMENT_ID, safety.ARCHIVE_SUFFIX,
    )
    predecessor.EXPERIMENT_ID, predecessor.ARCHIVE_SUFFIX = target.experiment, target.suffix
    base.EXPERIMENT_ID = target.experiment
    safety.EXPERIMENT_ID, safety.ARCHIVE_SUFFIX = target.experiment, target.suffix
    try:
        if predecessor.archive_root(home) != root or safety.canonical_archive(home) != root:
            raise ValueError("successor archive identity binding failed")
        probe = safety.namespace("identity", 1, "candidate", "long_form", "probe")
        if not probe.startswith(f"{target.experiment}-"):
            raise ValueError("successor namespace identity binding failed")
        try:
            yield root
        finally:
            other_after = _inventory(other_root) if other_root.exists() else None
            if other_after != other_before:
                raise ValueError("non-target successor archive changed")
    finally:
        predecessor.EXPERIMENT_ID, predecessor.ARCHIVE_SUFFIX, base.EXPERIMENT_ID, safety.EXPERIMENT_ID, safety.ARCHIVE_SUFFIX = saved


def _assert_manifest(value: Mapping[str, object], target: Target, root: Path) -> None:
    if value.get("experiment") != target.experiment:
        raise ValueError("manifest experiment identity mismatch")
    namespace = value.get("namespace")
    if namespace is not None and not str(namespace).startswith(f"{target.experiment}-"):
        raise ValueError("manifest namespace identity mismatch")
    for key in ("database", "artifact_root", "workdir"):
        path = value.get(key)
        if path is not None and not Path(str(path)).resolve().is_relative_to(root):
            raise ValueError(f"manifest {key} escapes the successor root")


def _source_inventories(home: Path) -> dict[str, str]:
    values = {
        "17_raw": _inventory((home / ARCHIVE_BASE / predecessor.SOURCE_EXPERIMENT_ID).resolve()),
        "18_raw": _inventory(_source_root(home)),
        "17_public": _inventory(_repo() / "plasma/docs/experiments" / predecessor.SOURCE_EXPERIMENT_ID),
        "18_public": _inventory(_repo() / "plasma/docs/experiments" / SOURCE_ID),
    }
    expected = {
        "17_raw": PREDECESSOR_ARCHIVE_SHA256, "18_raw": SOURCE_ARCHIVE_SHA256,
        "17_public": PREDECESSOR_PUBLIC_SHA256, "18_public": SOURCE_PUBLIC_SHA256,
    }
    if values != expected:
        raise ValueError("#17/#18 immutable input inventory mismatch")
    return values


def _controller_commit() -> str:
    if _git("status", "--porcelain"):
        raise ValueError("prepare requires a clean controller worktree")
    commit = _git("rev-parse", "HEAD")
    if commit == CANDIDATE_COMMIT:
        raise ValueError("controller and product candidate commits must be distinct")
    if _git("diff", "--name-only", CANDIDATE_COMMIT, "--", *PRODUCT_PATHS):
        raise ValueError("live product code differs from the locked candidate")
    return commit


def _seal_stopped(root: Path, stage: str, error: Exception) -> None:
    marker = root / "control/experiment-stopped.json"
    if root.exists() and not marker.exists():
        _write_new(marker, {"experiment": root.name, "stopped": True, "stage": stage, "reason": str(error)})


def _analysis_inputs(home: Path) -> dict[str, object]:
    source = _source_root(home)
    names = (
        "control/protocol.lock.json", "control/prepare-gate.json", "control/quality-gate.json",
        "control/experiment-stopped.json", "control/packets.lock.json", "control/judge-gate.json",
        "judging/packets-mapping.private.json",
    )
    paths = [source / name for name in names]
    terminals = sorted((source / "runs").glob("*quality*/manifest.terminal.json"))
    scores = sorted((source / "judging/scores").glob("*.json"))
    quality = _object(source / "control/quality-gate.json")
    rows = quality.get("runs")
    if not isinstance(rows, list) or len(rows) != 24:
        raise ValueError("#18 quality gate must contain exactly 24 runs")
    failed = [row for row in rows if isinstance(row, dict) and row.get("terminal_status") == "itt_failure"]
    completed = [row for row in rows if isinstance(row, dict) and row.get("terminal_status") == "completed"]
    if len(completed) != 23 or len(failed) != 1 or failed[0].get("arm") != "candidate" or failed[0].get("started") is not True:
        raise ValueError("#18 terminal classification differs from the locked 23+1 ITT result")
    metrics = failed[0].get("machine_metrics")
    if not isinstance(metrics, dict) or metrics.get("binding_violation") != 1.0:
        raise ValueError("#18 ITT failure is not the locked report-plan binding failure")
    mapping = json.loads((source / "judging/packets-mapping.private.json").read_text(encoding="utf-8"))
    if not isinstance(mapping, list) or len(mapping) != 11 or len(terminals) != 24 or len(scores) != 11:
        raise ValueError("#18 terminal/mapping/score counts differ from preregistration")
    packet_gate, judge_gate = _object(source / "control/packets.lock.json"), _object(source / "control/judge-gate.json")
    if packet_gate.get("pair_count") != 11 or judge_gate.get("passed") is not True or judge_gate.get("score_count") != 11:
        raise ValueError("#18 packet or judge lock is invalid")
    paths.extend(terminals)
    paths.extend(scores)
    return {
        "experiment": ANALYSIS_ID, "source_experiment": SOURCE_ID,
        "files": {str(path.relative_to(source)): _sha256(path) for path in paths},
        "terminal_count": 24, "completed_count": 23, "itt_failure_count": 1,
        "mapping_count": 11, "score_count": 11,
    }


def prepare_analysis(home: Path = Path.home()) -> dict[str, object]:
    target = ANALYSIS
    with action_context(target, home) as root:
        if root.exists():
            raise ValueError("#19 archive already exists and cannot be repaired or reused")
        root.mkdir(parents=True)
        try:
            controller = _controller_commit()
            inventories = _source_inventories(home)
            inputs = _analysis_inputs(home)
            inputs.update(controller_commit=controller, source_inventories=inventories)
            _write_new(root / "control/input.lock.json", inputs)
            protocol = {
                "experiment": target.experiment, "controller_commit": controller,
                "source_experiment": SOURCE_ID, "source_candidate_commit": SOURCE_CANDIDATE_COMMIT,
                "current_candidate_commit": CANDIDATE_COMMIT, "seed": SEED, "draws": DRAWS,
                "margin": MARGIN, "completeness_mean_margin": COMPLETENESS_MEAN_MARGIN,
                "completeness_low_rate_increase": COMPLETENESS_LOW_RATE_INCREASE,
                "scored_pairs": 11, "itt_failure_pairs": 1, "topic_pairs": 12,
                "provider_runs": 0, "judge_runs": 0,
            }
            _write_new(root / "control/protocol.lock.json", protocol)
            integrity = _lstat_tree(root)
            gate = {
                "experiment": target.experiment, "passed": True, "controller_commit": controller,
                "input_lock_sha256": _sha256(root / "control/input.lock.json"),
                "protocol_lock_sha256": _sha256(root / "control/protocol.lock.json"),
                "post_prepare_link_integrity": integrity,
            }
            _write_new(root / "control/prepare-gate.json", gate)
            _lstat_tree(root)
            return gate
        except Exception as exc:
            _seal_stopped(root, "prepare-analysis", exc)
            raise


def _copy_fixture(fixture: Fixture, root: Path) -> Fixture:
    destination = root / "fixtures" / fixture.source_bundle.name
    destination.parent.mkdir(parents=True, exist_ok=True)
    shutil.copy2(fixture.source_bundle, destination)
    return replace(fixture, source_bundle=destination)


def _operational_source(home: Path) -> tuple[list[Fixture], Fixture, list[dict[str, object]]]:
    source = _source_root(home)
    fixtures = list(load_and_validate_fixtures(source / "fixtures.lock.json", source, minimum=12, maximum=12))
    smoke = load_and_validate_fixtures(source / "smoke-fixture.lock.json", source, minimum=1, maximum=1, require_registered=False)[0]
    schedule = _object(source / "control/execution-schedule.json").get("entries")
    if not isinstance(schedule, list):
        raise ValueError("#18 execution schedule is invalid")
    topics = [str(row.get("topic")) for row in schedule if isinstance(row, dict)]
    if len(fixtures) != 12 or len(set(topics)) != 12 or {item.topic for item in fixtures} != set(topics):
        raise ValueError("#18 operational fixture topics must be exactly 12 unique entries")
    return fixtures, smoke, [dict(row) for row in schedule if isinstance(row, dict)]


def _validate_build(build: Mapping[str, object], arm: str, commit: str) -> None:
    required = ("source_archive", "source_sha256", "binary", "binary_sha256", "version")
    if build.get("arm") != arm or build.get("commit") != commit or any(not str(build.get(key, "")).strip() for key in required):
        raise ValueError(f"{arm} build provenance is incomplete")
    if _sha256(Path(str(build["source_archive"]))) != build["source_sha256"] or _sha256(Path(str(build["binary"]))) != build["binary_sha256"]:
        raise ValueError(f"{arm} source or binary hash differs from its build manifest")
    actual_version = subprocess.check_output([str(build["binary"]), "version"], text=True).strip()
    if actual_version != build["version"]:
        raise ValueError(f"{arm} executable version output differs from its build manifest")


def prepare_operational(auth_seed: Path, home: Path = Path.home()) -> dict[str, object]:
    target = OPERATIONAL
    with action_context(target, home) as root:
        if root.exists():
            raise ValueError("#20 archive already exists and cannot be repaired or reused")
        root.mkdir(parents=True)
        try:
            controller = _controller_commit()
            inventories = _source_inventories(home)
            fixtures, smoke, schedule = _operational_source(home)
            auth_seed = auth_seed.expanduser().resolve()
            _lstat_tree(auth_seed)
            _lstat_tree(_source_root(home) / "fixtures")
            copied_fixtures = [_copy_fixture(item, root) for item in fixtures]
            copied_smoke = _copy_fixture(smoke, root)
            fixture_hash = freeze_fixture_manifest(tuple(copied_fixtures), root / "fixtures.lock.json")
            smoke_hash = freeze_fixture_manifest((copied_smoke,), root / "smoke-fixture.lock.json")
            copied_auth = root / "auth-seeds/codex"
            shutil.copytree(auth_seed, copied_auth, symlinks=False)
            baseline = export_and_build(_repo(), root, BASELINE_COMMIT, "baseline")
            candidate = export_and_build(_repo(), root, CANDIDATE_COMMIT, "candidate")
            _validate_build(baseline, "baseline", BASELINE_COMMIT)
            _validate_build(candidate, "candidate", CANDIDATE_COMMIT)
            if baseline["binary_sha256"] == candidate["binary_sha256"]:
                raise ValueError("baseline and candidate binaries must differ")
            source_config = _object(_source_root(home) / "config.json")
            model = str(source_config.get("models", {}).get("codex", "")).strip() if isinstance(source_config.get("models"), dict) else ""
            if not model:
                raise ValueError("#18 frozen Codex model is missing")
            config = {
                "experiment": target.experiment, "controller_commit": controller,
                "baseline_commit": BASELINE_COMMIT, "candidate_commit": CANDIDATE_COMMIT,
                "models": {"codex": model}, "efforts": {"codex": "high"},
                "source_policy": "mission-sources-only", "session_policy": "same_session",
                "token_budget": 120000, "time_budget_seconds": 7200,
                "auth_seeds": {"CODEX_HOME": str(copied_auth)},
            }
            _write_new(root / "config.json", config)
            _write_new(root / "control/execution-schedule.json", {
                "seed": SEED, "mode": "long_form",
                "entries": [{"topic": row["topic"], "arm": "candidate"} for row in schedule],
            })
            protocol = {
                "experiment": target.experiment, "controller_commit": controller,
                "baseline_commit": BASELINE_COMMIT, "candidate_commit": CANDIDATE_COMMIT,
                "executor": "codex", "model": model, "effort": "high", "mode": "long_form",
                "session_policy": "same_session", "source_policy": "mission-sources-only",
                "token_budget": 120000, "time_budget_seconds": 7200,
                "smoke_runs": 2, "reliability_runs": 12,
                "topics": [row["topic"] for row in schedule],
                "fixture_manifest_sha256": fixture_hash, "smoke_fixture_manifest_sha256": smoke_hash,
                "source_inventories": inventories, "product_tests_passed": True,
            }
            _write_new(root / "control/protocol.lock.json", protocol)
            integrity = _lstat_tree(root)
            gate = {
                "experiment": target.experiment, "passed": True, "controller_commit": controller,
                "baseline_commit": BASELINE_COMMIT, "candidate_commit": CANDIDATE_COMMIT,
                "baseline": baseline, "candidate": candidate,
                "protocol_lock_sha256": _sha256(root / "control/protocol.lock.json"),
                "post_prepare_link_integrity": integrity,
            }
            _write_new(root / "control/prepare-gate.json", gate)
            _lstat_tree(root)
            return gate
        except Exception as exc:
            _seal_stopped(root, "prepare-operational", exc)
            raise


def _require_prepared(target: Target, root: Path) -> tuple[dict[str, object], dict[str, object]]:
    if (root / "control/experiment-stopped.json").exists():
        raise ValueError("successor is stopped and cannot be resumed")
    gate, protocol = _object(root / "control/prepare-gate.json"), _object(root / "control/protocol.lock.json")
    _assert_manifest(gate, target, root)
    _assert_manifest(protocol, target, root)
    if gate.get("passed") is not True or gate.get("controller_commit") != _git("rev-parse", "HEAD"):
        raise ValueError("prepare gate or clean controller HEAD lock is invalid")
    if _git("status", "--porcelain") or _git("diff", "--name-only", CANDIDATE_COMMIT, "--", *PRODUCT_PATHS):
        raise ValueError("action requires a clean controller worktree and unchanged product code")
    return gate, protocol


def assemble_analysis_itt(home: Path = Path.home()) -> dict[str, object]:
    with action_context(ANALYSIS, home) as root:
        try:
            _, protocol = _require_prepared(ANALYSIS, root)
            inputs = _analysis_inputs(home)
            locked = _object(root / "control/input.lock.json")
            if inputs["files"] != locked.get("files") or _source_inventories(home) != locked.get("source_inventories"):
                raise ValueError("#18 analysis input differs from its immutable lock")
            source = _source_root(home)
            records = assemble_records(
                source, ("quality",), source / "judging/scores", source / "judging/packets-mapping.private.json",
                baseline_commit=BASELINE_COMMIT, candidate_commit=SOURCE_CANDIDATE_COMMIT,
                modes=("long_form",), phase_topic_counts={"quality": 12},
                phase_replicates={"quality": (1,)}, both_arms_plan_mcp=True,
            )
            itt = assemble_itt(records)
            keys = {(str(row["topic"]), str(row["arm"])) for row in itt}
            if len(itt) != 24 or len(keys) != 24 or len({row["topic"] for row in itt}) != 12:
                raise ValueError("ITT assembly must contain exactly 12 complete topic pairs")
            failed = [row for row in itt if row["terminal_status"] != "completed"]
            if len(failed) != 1 or any(value != 1.0 for value in failed[0]["scores"]["final"].values()):
                raise ValueError("ITT failure did not receive the preregistered low scores")
            payload = {"experiment": ANALYSIS_ID, "records": itt}
            _write_new(root / "analysis/itt-records.private.json", payload)
            gate = {
                "experiment": ANALYSIS_ID, "passed": True, "record_count": 24,
                "topic_pair_count": 12, "scored_pair_count": 11, "itt_failure_pair_count": 1,
                "itt_records_sha256": _sha256(root / "analysis/itt-records.private.json"),
                "protocol_lock_sha256": _sha256(root / "control/protocol.lock.json"),
                "seed": protocol["seed"],
            }
            _write_new(root / "control/itt-gate.json", gate)
            return gate
        except Exception as exc:
            _seal_stopped(root, "assemble-itt", exc)
            raise


def _content_preserving_audit() -> dict[str, object]:
    unchanged = (
        "plasma/internal/reporting/long_form_finalization_assembly.go",
        "plasma/internal/web/server.go",
    )
    hashes: dict[str, str] = {}
    for path in unchanged:
        before = subprocess.check_output(["git", "show", f"{SOURCE_CANDIDATE_COMMIT}:{path}"], cwd=_repo())
        after = subprocess.check_output(["git", "show", f"{CANDIDATE_COMMIT}:{path}"], cwd=_repo())
        if before != after:
            raise ValueError("mechanical final report assembly changed after experiment 18")
        hashes[path] = hashlib.sha256(after).hexdigest()
    route_diff = _git("diff", "-U0", SOURCE_CANDIDATE_COMMIT, CANDIDATE_COMMIT, "--", "plasma/internal/web/report_routes.go")
    forbidden = ("agentLongFormFinalizePrompt", "opening_markdown contains", "closing_markdown contains")
    if any(value in route_diff for value in forbidden):
        raise ValueError("successful final report prompt or boundary instructions changed")
    return {"passed": True, "from_commit": SOURCE_CANDIDATE_COMMIT, "to_commit": CANDIDATE_COMMIT, "unchanged_hashes": hashes}


def analyze_quality(home: Path = Path.home()) -> dict[str, object]:
    with action_context(ANALYSIS, home) as root:
        try:
            _, protocol = _require_prepared(ANALYSIS, root)
            itt_gate = _object(root / "control/itt-gate.json")
            records_path = root / "analysis/itt-records.private.json"
            if itt_gate.get("passed") is not True or _sha256(records_path) != itt_gate.get("itt_records_sha256"):
                raise ValueError("analysis requires the immutable passed ITT gate")
            wrapper = _object(records_path)
            _assert_manifest(wrapper, ANALYSIS, root)
            records = wrapper.get("records")
            if not isinstance(records, list):
                raise ValueError("ITT records are missing")
            itt = assemble_itt(records)
            differences = topic_endpoint_differences(itt, "long_form", "final")
            if len(differences) != 12:
                raise ValueError("quality analysis requires exactly 12 topic differences")
            completeness = {
                arm: [float(row["scores"]["final"]["completeness"]) for row in itt if row["arm"] == arm]
                for arm in ("baseline", "candidate")
            }
            mean_difference = sum(completeness["candidate"]) / 12 - sum(completeness["baseline"]) / 12
            low_rate = {arm: sum(value <= 2 for value in values) / 12 for arm, values in completeness.items()}
            lower = percentile_lower(differences, int(protocol["seed"]), int(protocol["draws"]))
            noninferiority = lower >= float(protocol["margin"])
            completeness_pass = guardrail(mean_difference, low_rate["baseline"], low_rate["candidate"])
            content_audit = _content_preserving_audit()
            result = {
                "experiment": ANALYSIS_ID, "source_experiment": SOURCE_ID,
                "current_candidate_commit": CANDIDATE_COMMIT, "topic_pair_count": 12,
                "scored_pair_count": 11, "itt_failure_pair_count": 1,
                "final_dimension_count": len(FINAL_DIMENSIONS), "bootstrap_draws": DRAWS,
                "mean_difference": sum(differences) / 12, "one_sided_95_lower": lower,
                "margin": MARGIN, "noninferiority_passed": noninferiority,
                "completeness_mean_difference": mean_difference,
                "completeness_baseline_low_rate": low_rate["baseline"],
                "completeness_candidate_low_rate": low_rate["candidate"],
                "completeness_passed": completeness_pass,
                "content_preserving_audit": content_audit,
                "passed": noninferiority and completeness_pass and content_audit["passed"],
            }
            write_aggregate(result, root / "analysis/aggregate.json")
            _write_new(root / "control/quality-analysis-gate.json", {
                "experiment": ANALYSIS_ID, "passed": result["passed"],
                "aggregate_sha256": _sha256(root / "analysis/aggregate.json"),
                "input_lock_sha256": _sha256(root / "control/input.lock.json"),
                "itt_gate_sha256": _sha256(root / "control/itt-gate.json"),
            })
            if not result["passed"]:
                _seal_stopped(root, "analyze-quality", ValueError("preregistered quality gate failed"))
            return result
        except Exception as exc:
            _seal_stopped(root, "analyze-quality", exc)
            raise


def _run_spec(root: Path, config: Mapping[str, object], gate: Mapping[str, object], fixture: Fixture, arm: str, nonce: str) -> RunSpec:
    build = gate.get(arm)
    if not isinstance(build, Mapping):
        raise ValueError(f"missing {arm} build lock")
    return RunSpec(
        fixture.topic, 1, arm, "long_form", "codex", str(build["commit"]), Path(str(build["binary"])),
        str(config["models"]["codex"]), "high", "mission-sources-only", 120000, 7200,
        "same_session", fixture.source_bundle, fixture.source_sha256, nonce,
    )


def _execute_once(spec: RunSpec, fixture: Fixture, auth: Mapping[str, Path], root: Path, used_ports: set[int], namespaces: set[str], lock: Lock) -> dict[str, object]:
    with lock:
        manifest = base.build_manifest(spec, Path.home(), dict(os.environ), used_ports, namespaces)
    _assert_manifest(manifest.as_dict(), OPERATIONAL, root)
    try:
        marker = "report_long_form_final_completed" if manifest.arm == "candidate" else None
        terminal = execute_product_run(manifest, fixture, auth_seeds=auth, completion_log_marker=marker)
        ledger = _object(Path(terminal.database).parent.parent / "ledger.events.json")
        events = ledger.get("events")
        if not isinstance(events, list):
            raise ValueError("run ledger events are missing")
        metrics = base.collect_hard_metrics(events, terminal.result_hash is not None, terminal.arm == "candidate", terminal.as_dict())
        finalizer = predecessor.assert_finalizer_path(terminal.as_dict())
        return {**terminal.as_dict(), "started": True, "artifact_presence": 1, "machine_metrics": metrics, "finalizer_path": finalizer, "attempts": [{"attempt": 1, "classification": "completed", "namespace": terminal.namespace}]}
    except ProductRunError as exc:
        current = exc.manifest or manifest
        started = exc.started or current.start_boundary.startswith("started:")
        terminal = replace(current, terminal_status="itt_failure" if started else "pre_run_failure")
        return {**terminal.as_dict(), "started": started, "artifact_presence": 0, "machine_metrics": base._failed_metrics(terminal.arm == "candidate"), "finalizer_path": {"error": exc.kind}, "attempts": [{"attempt": 1, "classification": "itt" if started else "pre_run_infrastructure", "namespace": terminal.namespace, "kind": exc.kind}]}
    finally:
        with lock:
            used_ports.discard(manifest.port)
            used_ports.discard(manifest.connector_port)


def execute_operational_phase(phase: str, workers: int, home: Path = Path.home()) -> dict[str, object]:
    if phase not in {"smoke", "reliability"}:
        raise ValueError("unsupported operational phase")
    with action_context(OPERATIONAL, home) as root:
        try:
            gate, protocol = _require_prepared(OPERATIONAL, root)
            if phase == "smoke" and workers != 2:
                raise ValueError("smoke requires exactly two workers")
            if phase == "reliability" and not 1 <= workers <= 6:
                raise ValueError("reliability requires one to six workers")
            if phase == "reliability" and _object(root / "control/smoke-gate.json").get("passed") is not True:
                raise ValueError("reliability requires the corrected smoke gate")
            config = _object(root / "config.json")
            _assert_manifest(config, OPERATIONAL, root)
            fixtures = load_and_validate_fixtures(
                root / ("smoke-fixture.lock.json" if phase == "smoke" else "fixtures.lock.json"), root,
                minimum=1 if phase == "smoke" else 12, maximum=1 if phase == "smoke" else 12,
                require_registered=phase != "smoke",
            )
            cells = [(fixtures[0], arm) for arm in ("baseline", "candidate")] if phase == "smoke" else [(fixture, "candidate") for fixture in fixtures]
            if len(cells) != (2 if phase == "smoke" else 12):
                raise ValueError("operational phase matrix count is invalid")
            protected = (
                home / "research-artifacts/liquid2/plasma/runtime/dev-6002/plasma-ui-user.db",
                home / "Library/Application Support/Plasma/plasma.db",
            )
            protected_before = safety.snapshot_protected_paths(protected, root)
            source_before = _source_inventories(home)
            auth = {"CODEX_HOME": Path(str(config["auth_seeds"]["CODEX_HOME"]))}
            used_ports: set[int] = set()
            namespaces: set[str] = set()
            lock = Lock()
            results: list[dict[str, object]] = []
            with ThreadPoolExecutor(max_workers=workers) as pool:
                futures = [pool.submit(_execute_once, _run_spec(root, config, gate, fixture, arm, f"{phase}-a1"), fixture, auth, root, used_ports, namespaces, lock) for fixture, arm in cells]
                for future in as_completed(futures):
                    results.append(future.result())
            expected = {(fixture.topic, arm) for fixture, arm in cells}
            actual = {(str(row["topic"]), str(row["arm"])) for row in results}
            all_invariants = actual == expected and all(
                row["terminal_status"] == "completed" and hard_gate(row["machine_metrics"], row["arm"] == "candidate")
                and "error" not in row["finalizer_path"] for row in results
            )
            unchanged = _source_inventories(home) == source_before and safety.snapshot_protected_paths(protected, root) == protected_before
            phase_gate = {
                "experiment": OPERATIONAL_ID, "phase": phase, "passed": all_invariants and unchanged,
                "workers": workers, "runs": results, "matrix_complete": actual == expected,
                "all_invariants_passed": all_invariants, "protected_paths_unchanged": unchanged,
                "expected_run_count": len(cells), "completed_run_count": sum(row["terminal_status"] == "completed" for row in results),
                "protocol_lock_sha256": _sha256(root / "control/protocol.lock.json"),
            }
            _write_new(root / f"control/{phase}-gate.json", phase_gate)
            if not phase_gate["passed"]:
                _seal_stopped(root, phase, ValueError(f"{phase} gate failed"))
            return phase_gate
        except Exception as exc:
            _seal_stopped(root, phase, exc)
            raise


def audit_operational(home: Path = Path.home()) -> dict[str, object]:
    with action_context(OPERATIONAL, home) as root:
        try:
            gate, protocol = _require_prepared(OPERATIONAL, root)
            smoke, reliability = _object(root / "control/smoke-gate.json"), _object(root / "control/reliability-gate.json")
            if smoke.get("passed") is not True or reliability.get("passed") is not True:
                raise ValueError("operational audit requires passed smoke and reliability gates")
            smoke_rows, rows = smoke.get("runs"), reliability.get("runs")
            if not isinstance(smoke_rows, list) or len(smoke_rows) != 2 or not isinstance(rows, list) or len(rows) != 12:
                raise ValueError("operational terminal matrix is incomplete")
            if {(row.get("arm"), row.get("mode"), row.get("executor")) for row in rows} != {("candidate", "long_form", "codex")}:
                raise ValueError("reliability matrix contains a prohibited cell")
            for row in [*smoke_rows, *rows]:
                _assert_manifest(row, OPERATIONAL, root)
                terminal = root / "runs" / str(row["namespace"]) / "manifest.terminal.json"
                if _object(terminal).get("result_hash") != row.get("result_hash"):
                    raise ValueError("terminal manifest differs from its gate")
                finalizer = predecessor.assert_finalizer_path(row)
                if finalizer != row.get("finalizer_path") or len(row.get("attempts", [])) != 1:
                    raise ValueError("finalizer or no-replacement evidence differs from the gate")
                expected_commit = BASELINE_COMMIT if row["arm"] == "baseline" else CANDIDATE_COMMIT
                build = gate[row["arm"]]
                if row.get("commit") != expected_commit or row.get("binary_hash") != build.get("binary_sha256"):
                    raise ValueError("run product provenance differs from the prepare lock")
            integrity = _lstat_tree(root)
            result = {
                "experiment": OPERATIONAL_ID, "passed": True,
                "smoke_run_count": 2, "smoke_completed_count": 2,
                "reliability_denominator": 12, "reliability_completed_count": 12,
                "all_invariants_count": 12, "candidate_commit": CANDIDATE_COMMIT,
                "controller_commit": gate["controller_commit"], "post_run_link_integrity": integrity,
                "source_inventories": _source_inventories(home), "product_tests_passed": protocol["product_tests_passed"],
            }
            _write_new(root / "analysis/aggregate.json", result)
            _write_new(root / "control/audit-gate.json", {
                "experiment": OPERATIONAL_ID, "passed": True,
                "aggregate_sha256": _sha256(root / "analysis/aggregate.json"),
                "smoke_gate_sha256": _sha256(root / "control/smoke-gate.json"),
                "reliability_gate_sha256": _sha256(root / "control/reliability-gate.json"),
            })
            return result
        except Exception as exc:
            _seal_stopped(root, "audit", exc)
            raise


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--action", required=True, choices=(
        "prepare-analysis", "assemble-itt", "analyze-quality",
        "prepare-operational", "smoke", "reliability", "audit",
    ))
    parser.add_argument("--auth-seed", type=Path)
    parser.add_argument("--workers", type=int)
    args = parser.parse_args()
    if args.action == "prepare-operational" and args.auth_seed is None:
        parser.error("prepare-operational requires --auth-seed")
    if args.action != "prepare-operational" and args.auth_seed is not None:
        parser.error("only prepare-operational accepts --auth-seed")
    if args.action == "smoke" and args.workers not in (None, 2):
        parser.error("smoke requires --workers 2")
    if args.action == "reliability" and args.workers is not None and not 1 <= args.workers <= 6:
        parser.error("reliability workers must be between 1 and 6")
    if args.action not in {"smoke", "reliability"} and args.workers is not None:
        parser.error("only smoke and reliability accept --workers")
    return args


def main() -> int:
    args = parse_args()
    if args.action == "prepare-analysis":
        result = prepare_analysis()
    elif args.action == "assemble-itt":
        result = assemble_analysis_itt()
    elif args.action == "analyze-quality":
        result = analyze_quality()
    elif args.action == "prepare-operational":
        result = prepare_operational(args.auth_seed)
    elif args.action in {"smoke", "reliability"}:
        result = execute_operational_phase(args.action, args.workers or (2 if args.action == "smoke" else 6))
    else:
        result = audit_operational()
    print(json.dumps(result, indent=2, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
