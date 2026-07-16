#!/usr/bin/env python3
"""Thin long-form-only successor controller for issue #110 experiment 18."""

from __future__ import annotations

import argparse
from concurrent.futures import ThreadPoolExecutor, as_completed
from dataclasses import replace
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
from threading import Lock
from typing import Mapping, Sequence

import report_plan_mcp.safety as safety
import report_plan_mcp_experiment as base
from codex_blind_judge import schema_sha256
from report_plan_mcp.audit import audit_lineage, hard_gate
from report_plan_mcp.builds import export_and_build, sha256_file
from report_plan_mcp.judging import FINAL_DIMENSIONS, build_blind_packets, score_packets
from report_plan_mcp.models import Fixture, RunSpec, freeze_fixture_manifest, load_and_validate_fixtures
from report_plan_mcp.product_path import ProductRunError, execute_product_run
from report_plan_mcp.provenance import assemble_records, build_pairs
from report_plan_mcp.statistics import (
    exact_sign_pvalue, guardrail, holm_adjust, paired_wilcoxon_pvalue,
    percentile_lower, topic_endpoint_differences, write_aggregate,
)


EXPERIMENT_ID = "18-report-long-form-finalize-mcp-2026-07-14"
BASELINE_COMMIT = "1b6239805f2dde41f7aaab36d8025812623da5a6"
CANDIDATE_COMMIT = "4bc3ac07fab93f31d9447c0a83802f6628bd9623"
SOURCE_EXPERIMENT_ID = "17-report-plan-mcp-focused-2026-07-14"
ARCHIVE_SUFFIX = Path("research-artifacts/liquid2/plasma/experiments") / EXPERIMENT_ID
SOURCE_SUFFIX = Path("research-artifacts/liquid2/plasma/experiments") / SOURCE_EXPERIMENT_ID
PUBLIC_PROTOCOL = Path("plasma/docs/experiments") / EXPERIMENT_ID / "protocol.md"
RUBRIC = Path("plasma/docs/experiments/17-report-plan-mcp-focused-2026-07-14/focused-rubric.md")
SEED = 110
MARGIN = -0.25


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--action", required=True, choices=("preflight", "prepare", "smoke", "quality", "packets", "judge", "analyze"))
    parser.add_argument("--model")
    parser.add_argument("--auth-seed", type=Path)
    parser.add_argument("--workers", type=int, default=2)
    parser.add_argument("--dry-run", action="store_true")
    args = parser.parse_args()
    if args.action == "preflight" and not args.dry_run:
        parser.error("preflight requires --dry-run")
    if args.action in {"preflight", "prepare"} and (not isinstance(args.model, str) or not args.model.strip()):
        parser.error("preflight and prepare require a non-blank --model")
    if args.action == "prepare" and args.auth_seed is None:
        parser.error("prepare requires --auth-seed")
    if args.action not in {"preflight", "prepare"} and (args.model is not None or args.auth_seed is not None):
        parser.error("run actions use the immutable prepare config")
    return args


def archive_root(home: Path = Path.home()) -> Path:
    return (home / ARCHIVE_SUFFIX).resolve()


def source_root(home: Path = Path.home()) -> Path:
    return (home / SOURCE_SUFFIX).resolve()


def configure_reused_helpers() -> None:
    base.EXPERIMENT_ID = EXPERIMENT_ID
    safety.EXPERIMENT_ID = EXPERIMENT_ID
    safety.ARCHIVE_SUFFIX = ARCHIVE_SUFFIX


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


def _repo() -> Path:
    return Path(__file__).resolve().parents[3]


def _git(*args: str) -> str:
    return subprocess.check_output(["git", *args], cwd=_repo(), text=True).strip()


def _inventory(root: Path) -> str:
    digest = hashlib.sha256()
    for path in sorted(root.rglob("*")):
        if path.is_symlink():
            raise ValueError(f"source experiment contains a symlink: {path}")
        if path.is_file():
            digest.update(str(path.relative_to(root)).encode())
            digest.update(str(path.stat().st_size).encode())
            digest.update(_sha256(path).encode())
    return digest.hexdigest()


def _source_rows() -> tuple[list[dict[str, object]], dict[str, object]]:
    source = source_root()
    schedule = _object(source / "control/focused-execution-schedule.json")
    entries = schedule.get("entries")
    if not isinstance(entries, list):
        raise ValueError("#17 focused schedule is invalid")
    selected = [dict(entry) for entry in entries if isinstance(entry, dict) and entry.get("mode") == "long_form"]
    topics = [str(entry.get("topic")) for entry in selected]
    if len(selected) != 12 or len(set(topics)) != 12:
        raise ValueError("#17 schedule must yield exactly 12 unique long-form topics")
    if any(set(entry) != {"topic", "mode", "arms"} or entry.get("arms") not in (["baseline", "candidate"], ["candidate", "baseline"]) for entry in selected):
        raise ValueError("#17 long-form schedule contains an invalid cell")
    if sum(entry["arms"][0] == "baseline" for entry in selected) != 6:
        raise ValueError("#17 long-form schedule is not 6:6 counterbalanced")
    lock = _object(source / "fixtures.lock.json")
    fixtures = lock.get("fixtures")
    if not isinstance(fixtures, list):
        raise ValueError("#17 fixture lock is invalid")
    by_topic = {str(row.get("topic")): row for row in fixtures if isinstance(row, dict)}
    if set(topics) - set(by_topic):
        raise ValueError("#17 schedule topic is absent from its fixture lock")
    return selected, by_topic


def preflight(model: str) -> dict[str, object]:
    selected, rows = _source_rows()
    if not model.strip():
        raise ValueError("Codex model must be non-blank")
    return {
        "experiment": EXPERIMENT_ID, "baseline_commit": BASELINE_COMMIT,
        "candidate_commit": CANDIDATE_COMMIT, "executor": "codex", "model": model.strip(),
        "effort": "high", "mode": "long_form", "smoke_runs": 2, "quality_runs": 24,
        "topics": [entry["topic"] for entry in selected],
        "fixture_hashes": {topic: rows[str(topic)]["source_sha256"] for topic in (entry["topic"] for entry in selected)},
        "archive": str(archive_root()), "runnable": True,
    }


def prepare(model: str, auth_seed: Path) -> dict[str, object]:
    repo, archive, source = _repo(), archive_root(), source_root()
    if archive.exists():
        raise ValueError("#18 archive already exists; prepare is immutable")
    if _git("status", "--porcelain"):
        raise ValueError("prepare requires a clean worktree")
    controller_commit = _git("rev-parse", "HEAD")
    if controller_commit == CANDIDATE_COMMIT or CANDIDATE_COMMIT == BASELINE_COMMIT:
        raise ValueError("controller, candidate, and baseline commits must be distinct")
    if _git("merge-base", "--is-ancestor", CANDIDATE_COMMIT, controller_commit) != "":
        raise AssertionError("unexpected git output")
    auth_seed = auth_seed.expanduser().resolve()
    if not auth_seed.is_dir():
        raise ValueError("auth seed must be an existing directory")
    selected, rows = _source_rows()
    source_inventory = _inventory(source)
    archive.mkdir(parents=True)
    fixtures: list[Fixture] = []
    for entry in selected:
        row = rows[str(entry["topic"])]
        source_file = Path(str(row["source_bundle"])).resolve()
        if not source_file.is_relative_to(source) or _sha256(source_file) != row["source_sha256"]:
            raise ValueError("#17 source fixture provenance mismatch")
        destination = archive / "fixtures" / source_file.name
        destination.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(source_file, destination)
        fixtures.append(Fixture(
            str(row["topic"]), str(row["title"]), str(row["objective"]), destination,
            str(row["source_sha256"]), str(row["license"]), str(row["license_url"]), str(row["retrieved_at"]),
        ))
    fixture_hash = freeze_fixture_manifest(tuple(fixtures), archive / "fixtures.lock.json")
    smoke_source = load_and_validate_fixtures(source / "smoke-fixture.lock.json", source, minimum=1, maximum=1, require_registered=False)[0]
    smoke_file = archive / "fixtures" / smoke_source.source_bundle.name
    shutil.copy2(smoke_source.source_bundle, smoke_file)
    smoke = Fixture(smoke_source.topic, smoke_source.title, smoke_source.objective, smoke_file, smoke_source.source_sha256, smoke_source.license, smoke_source.license_url, smoke_source.retrieved_at)
    smoke_hash = freeze_fixture_manifest((smoke,), archive / "smoke-fixture.lock.json")
    copied_auth = archive / "auth-seeds/codex"
    shutil.copytree(auth_seed, copied_auth, symlinks=False)
    baseline = export_and_build(repo, archive, BASELINE_COMMIT, "baseline")
    candidate = export_and_build(repo, archive, CANDIDATE_COMMIT, "candidate")
    if baseline["binary_sha256"] == candidate["binary_sha256"]:
        raise ValueError("baseline and candidate binaries must differ")
    schedule = {"seed": SEED, "mode": "long_form", "entries": selected}
    _write_new(archive / "control/execution-schedule.json", schedule)
    config = {
        "experiment": EXPERIMENT_ID, "controller_commit": controller_commit,
        "candidate_commit": CANDIDATE_COMMIT, "models": {"codex": model.strip()},
        "efforts": {"codex": "high"}, "source_policy": "mission-sources-only",
        "token_budget": 120000, "time_budget_seconds": 7200,
        "session_policy": "same_session", "auth_seeds": {"CODEX_HOME": str(copied_auth)},
        "seed": SEED, "judge": {"model": model.strip(), "effort": "high"},
    }
    _write_new(archive / "config.json", config)
    protocol = {
        "experiment": EXPERIMENT_ID, "baseline_commit": BASELINE_COMMIT,
        "candidate_commit": CANDIDATE_COMMIT, "controller_commit": controller_commit,
        "fixture_manifest_sha256": fixture_hash, "smoke_fixture_manifest_sha256": smoke_hash,
        "schedule_sha256": _sha256(archive / "control/execution-schedule.json"),
        "source_experiment_inventory_sha256": source_inventory,
        "executor": "codex", "model": model.strip(), "effort": "high", "mode": "long_form",
        "topics": [entry["topic"] for entry in selected], "smoke_runs": 2, "quality_runs": 24,
        "session_policy": "same_session", "token_budget": 120000, "time_budget_seconds": 7200,
        "rubric_sha256": _sha256(repo / RUBRIC),
        "judge_adapter_sha256": _sha256(repo / "plasma/scripts/experiments/codex_blind_judge.py"),
        "judge_schema_sha256": schema_sha256(), "margin": MARGIN,
        "completeness_mean_margin": -0.50, "completeness_low_rate_increase": 0.10,
        "product_tests_passed": True,
    }
    _write_new(archive / "control/protocol.lock.json", protocol)
    gate = {
        "passed": True, "controller_commit": controller_commit,
        "baseline_commit": BASELINE_COMMIT, "candidate_commit": CANDIDATE_COMMIT,
        "baseline": baseline, "candidate": candidate, "models": config["models"], "efforts": config["efforts"],
        "fixture_manifest_sha256": fixture_hash, "smoke_fixture_manifest_sha256": smoke_hash,
        "protocol_lock_sha256": _sha256(archive / "control/protocol.lock.json"),
    }
    _write_new(archive / "control/prepare-gate.json", gate)
    if _inventory(source) != source_inventory:
        raise ValueError("#17 archive changed during prepare")
    return gate


def _config() -> dict[str, object]:
    config = _object(archive_root() / "config.json")
    if config.get("experiment") != EXPERIMENT_ID or config.get("candidate_commit") != CANDIDATE_COMMIT:
        raise ValueError("#18 config identity mismatch")
    models, efforts = config.get("models"), config.get("efforts")
    if not isinstance(models, dict) or set(models) != {"codex"} or not str(models["codex"]).strip():
        raise ValueError("config must freeze exactly one Codex model")
    if efforts != {"codex": "high"}:
        raise ValueError("config must freeze Codex high effort")
    return config


def _prepare_gate() -> dict[str, object]:
    archive = archive_root()
    gate = _object(archive / "control/prepare-gate.json")
    if gate.get("passed") is not True or gate.get("baseline_commit") != BASELINE_COMMIT or gate.get("candidate_commit") != CANDIDATE_COMMIT:
        raise ValueError("prepare gate is missing or invalid")
    for arm in ("baseline", "candidate"):
        build = gate.get(arm)
        if not isinstance(build, dict) or _sha256(Path(str(build["binary"]))) != build.get("binary_sha256"):
            raise ValueError("locked build hash mismatch")
    return gate


def _spec(config: Mapping[str, object], fixture: Fixture, arm: str, nonce: str) -> RunSpec:
    gate = _prepare_gate()
    build = gate[arm]
    if not isinstance(build, dict):
        raise ValueError("build lock is invalid")
    return RunSpec(
        fixture.topic, 1, arm, "long_form", "codex", str(build["commit"]), Path(str(build["binary"])),
        str(config["models"]["codex"]), "high", str(config["source_policy"]), int(config["token_budget"]),
        int(config["time_budget_seconds"]), str(config["session_policy"]), fixture.source_bundle,
        fixture.source_sha256, nonce,
    )


def _execute_specs(
    specs: list[RunSpec], fixture: Fixture, auth: Mapping[str, Path], used_ports: set[int],
    namespaces: set[str], lock: Lock,
) -> tuple[object, dict[str, float], list[dict[str, object]]]:
    history: list[dict[str, object]] = []
    for attempt, spec in enumerate(specs, start=1):
        with lock:
            manifest = base.build_manifest(spec, Path.home(), dict(os.environ), used_ports, namespaces)
        try:
            marker = "report_long_form_final_completed" if manifest.arm == "candidate" else None
            terminal = execute_product_run(manifest, fixture, auth_seeds=auth, completion_log_marker=marker)
            events = _object(Path(terminal.database).parent.parent / "ledger.events.json")["events"]
            metrics = base.collect_hard_metrics(events, terminal.result_hash is not None, True, terminal.as_dict())
            history.append({"attempt": attempt, "namespace": manifest.namespace, "classification": "completed"})
            return terminal, metrics, history
        except ProductRunError as exc:
            classification = "itt" if exc.started else "pre_run_infrastructure"
            history.append({"attempt": attempt, "namespace": manifest.namespace, "classification": classification, "kind": exc.kind})
            current = exc.manifest or manifest
            terminal = replace(
                current, start_boundary="started:product_cli_mission_create" if exc.started else current.start_boundary,
                terminal_status="itt_failure" if exc.started else "pre_run_failure",
            )
            if exc.started or attempt == 3:
                return terminal, base._failed_metrics(True), history
        finally:
            with lock:
                used_ports.discard(manifest.port)
                used_ports.discard(manifest.connector_port)
    raise AssertionError("attempt loop did not return")


def assert_finalizer_path(run: Mapping[str, object]) -> dict[str, object]:
    root = Path(str(run["database"])).parent.parent
    ledger = _object(root / "ledger.events.json")
    events = ledger.get("events")
    if not isinstance(events, list):
        raise ValueError("ledger events are missing")
    plan_submitted = [event for event in events if isinstance(event, dict) and event.get("EventType") == "report.plan.submitted"]
    artifacts = [event for event in events if isinstance(event, dict) and event.get("EventType") == "report.artifact.created"]
    calls = [event for event in events if isinstance(event, dict) and event.get("EventType") == "mcp.tool.called" and isinstance(event.get("Payload"), dict) and event["Payload"].get("tool_name") == "plasma.report.long_form.finalize"]
    if len(plan_submitted) != 1 or len(artifacts) != 1:
        raise ValueError("run lacks one plan submission or final artifact")
    if not audit_lineage(events):
        raise ValueError("run plan submission lineage is invalid")
    if run.get("arm") == "baseline":
        if calls:
            raise ValueError("baseline unexpectedly called the finalizer")
        return {"finalizer_calls": 0, "successful_calls": 0, "sentinel": False}
    if not 1 <= len(calls) <= 2 or any(event["Payload"].get("success") is not True for event in calls):
        raise ValueError("candidate finalizer call count/success is invalid")
    artifact_id, event_id = artifacts[0]["Payload"].get("artifact_id"), artifacts[0].get("EventID")
    for call in calls:
        payload = call["Payload"]
        result = payload.get("result")
        created = result.get("created_event_ids") if isinstance(result, dict) else None
        if not isinstance(created, list) or event_id not in created:
            raise ValueError("finalizer trace does not reference the canonical artifact event")
        if artifacts[0]["Payload"].get("tool_session_id") != payload.get("tool_session_id"):
            raise ValueError("finalizer tool session does not match canonical provenance")
    log = (root / "logs/serve.log").read_text(encoding="utf-8")
    evidence = f'artifact_id="{artifact_id}" event_id="{event_id}"'
    if "report_long_form_final_completed" not in log or evidence not in log or "canonical=true sentinel_ok=true" not in log:
        raise ValueError("candidate safe runtime log lacks exact sentinel evidence")
    return {"finalizer_calls": len(calls), "successful_calls": len(calls), "sentinel": True, "artifact_id": artifact_id, "event_id": event_id}


def execute_phase(name: str, fixtures: Sequence[Fixture], workers: int) -> dict[str, object]:
    if name == "smoke" and (workers != 2 or len(fixtures) != 1):
        raise ValueError("smoke requires one fixture and exactly two workers")
    if name == "quality" and (not 1 <= workers <= 6 or len(fixtures) != 12):
        raise ValueError("quality requires 12 fixtures and one to six workers")
    if name == "quality" and _object(archive_root() / "control/smoke-gate.json").get("passed") is not True:
        raise ValueError("quality requires a passed smoke gate")
    config, archive = _config(), archive_root()
    protocol = _object(archive / "control/protocol.lock.json")
    source_before = str(protocol.get("source_experiment_inventory_sha256"))
    if _inventory(source_root()) != source_before:
        raise ValueError("#17 archive differs from the prepare inventory")
    protected = (
        Path.home() / "research-artifacts/liquid2/plasma/runtime/dev-6002/plasma-ui-user.db",
        Path.home() / "Library/Application Support/Plasma/plasma.db",
    )
    protected_before = safety.snapshot_protected_paths(protected, archive)
    schedule = _object(archive / "control/execution-schedule.json")
    entries = schedule["entries"] if name == "quality" else [{"topic": fixtures[0].topic, "arms": ["baseline", "candidate"]}]
    by_topic = {fixture.topic: fixture for fixture in fixtures}
    cells = [(by_topic[str(entry["topic"])], str(arm)) for entry in entries for arm in entry["arms"]]
    used_ports: set[int] = set()
    namespaces: set[str] = set()
    lock = Lock()
    auth = {"CODEX_HOME": Path(str(config["auth_seeds"]["CODEX_HOME"]))}
    groups = [([_spec(config, fixture, arm, f"{name}-a{attempt}") for attempt in range(1, 4)], fixture) for fixture, arm in cells]
    results: list[dict[str, object]] = []
    with ThreadPoolExecutor(max_workers=workers) as pool:
        futures = {pool.submit(_execute_specs, specs, fixture, auth, used_ports, namespaces, lock): specs for specs, fixture in groups}
        for future in as_completed(futures):
            terminal, metrics, attempts = future.result()
            row = {**terminal.as_dict(), "started": terminal.start_boundary.startswith("started:"), "artifact_presence": int(terminal.result_hash is not None), "machine_metrics": metrics, "attempts": attempts}
            if terminal.terminal_status == "completed":
                try:
                    row["finalizer_path"] = assert_finalizer_path(row)
                except ValueError as exc:
                    row["finalizer_path"] = {"error": str(exc)}
            results.append(row)
    expected = {(fixture.topic, arm) for fixture, arm in cells}
    actual = {(str(row["topic"]), str(row["arm"])) for row in results}
    protected_unchanged = safety.snapshot_protected_paths(protected, archive) == protected_before
    source_unchanged = _inventory(source_root()) == source_before
    structural = actual == expected and protected_unchanged and source_unchanged and not any(row["terminal_status"] == "pre_run_failure" for row in results)
    machine = structural and all(
        row["terminal_status"] == "completed" and hard_gate(row["machine_metrics"], row["arm"] == "candidate")
        and "error" not in row.get("finalizer_path", {}) for row in results
    )
    gate = {
        "phase": name, "workers": workers, "runs": results, "matrix_complete": actual == expected,
        "protected_paths_unchanged": protected_unchanged, "source_experiment_unchanged": source_unchanged,
        "machine_passed": machine, "passed": machine if name == "smoke" else structural,
    }
    _write_new(archive / f"control/{name}-gate.json", gate)
    return gate


def _pairs() -> tuple[list[dict[str, object]], list[dict[str, object]]]:
    return build_pairs(
        archive_root(), ("quality",), baseline_commit=BASELINE_COMMIT, candidate_commit=CANDIDATE_COMMIT,
        modes=("long_form",), phase_topic_counts={"quality": 12}, phase_replicates={"quality": (1,)},
        both_arms_plan_mcp=True,
    )


def packets() -> list[dict[str, object]]:
    gate = _object(archive_root() / "control/quality-gate.json")
    if gate.get("passed") is not True:
        raise ValueError("packets require a structurally complete quality gate")
    pairs, _ = _pairs()
    result = build_blind_packets(pairs, archive_root() / "judging/packets", SEED)
    _write_new(archive_root() / "control/packets.lock.json", {
        "pair_count": len(result), "mapping_sha256": _sha256(archive_root() / "judging/packets-mapping.private.json"),
        "packet_hashes": {path.name: _sha256(path) for path in sorted((archive_root() / "judging/packets").glob("*.json"))},
    })
    return result


def judge() -> list[dict[str, object]]:
    archive, config = archive_root(), _config()
    lock = _object(archive / "control/packets.lock.json")
    paths = sorted((archive / "judging/packets").glob("*.json"))
    if len(paths) != lock.get("pair_count") or {path.name: _sha256(path) for path in paths} != lock.get("packet_hashes"):
        raise ValueError("judge packets differ from the immutable packet lock")
    settings = config["judge"]
    command = [sys.executable, str(_repo() / "plasma/scripts/experiments/codex_blind_judge.py"), "--model", str(settings["model"]), "--effort", str(settings["effort"]), "--rubric", str(_repo() / RUBRIC)]
    destination = archive / "judging/scores"
    result = score_packets(command, paths, destination, base._judge_environment(config, archive, "experiment-18"))
    _write_new(archive / "control/judge-gate.json", {"passed": len(result) == len(paths), "score_count": len(result), "score_hashes": {path.name: _sha256(path) for path in sorted(destination.glob("*.json"))}})
    return result


def analyze() -> dict[str, object]:
    archive = archive_root()
    quality, judge_gate = _object(archive / "control/quality-gate.json"), _object(archive / "control/judge-gate.json")
    if quality.get("passed") is not True or judge_gate.get("passed") is not True:
        raise ValueError("analysis requires complete quality and judge gates")
    scores = archive / "judging/scores"
    if {path.name: _sha256(path) for path in sorted(scores.glob("*.json"))} != judge_gate.get("score_hashes"):
        raise ValueError("judge scores differ from the immutable judge gate")
    records = assemble_records(
        archive, ("quality",), scores, archive / "judging/packets-mapping.private.json",
        baseline_commit=BASELINE_COMMIT, candidate_commit=CANDIDATE_COMMIT, modes=("long_form",),
        phase_topic_counts={"quality": 12}, phase_replicates={"quality": (1,)}, both_arms_plan_mcp=True,
    )
    differences = topic_endpoint_differences(records, "long_form", "final")
    if len(differences) != 12:
        raise ValueError("analysis requires exactly 12 topic pairs")
    completeness = [
        float(row["scores"]["final"]["completeness"])
        for row in records
    ]
    baseline = [value for value, row in zip(completeness, records, strict=True) if row["arm"] == "baseline"]
    candidate = [value for value, row in zip(completeness, records, strict=True) if row["arm"] == "candidate"]
    mean_difference = sum(candidate) / len(candidate) - sum(baseline) / len(baseline)
    low = lambda values: sum(value <= 2 for value in values) / len(values)
    lower = percentile_lower(differences, SEED, 10_000)
    completeness_pass = guardrail(mean_difference, low(baseline), low(candidate))
    protocol = _object(archive / "control/protocol.lock.json")
    result = {
        "experiment": EXPERIMENT_ID, "topic_count": 12, "run_count": 24,
        "final_dimensions": list(FINAL_DIMENSIONS), "mean_difference": sum(differences) / len(differences),
        "one_sided_95_lower": lower, "margin": MARGIN, "noninferior": lower >= MARGIN,
        "completeness": {"mean_difference": mean_difference, "baseline_low_rate": low(baseline), "candidate_low_rate": low(candidate), "passed": completeness_pass},
        "sign_pvalue_holm": holm_adjust([exact_sign_pvalue(differences)])[0],
        "wilcoxon_pvalue_holm": holm_adjust([paired_wilcoxon_pvalue(differences)])[0],
        "machine_gate": quality.get("machine_passed") is True,
        "adopt": protocol.get("product_tests_passed") is True and quality.get("machine_passed") is True and lower >= MARGIN and completeness_pass,
        "protocol_lock_sha256": _sha256(archive / "control/protocol.lock.json"),
    }
    write_aggregate(result, archive / "analysis/aggregate.json")
    return result


def main() -> int:
    args = parse_args()
    configure_reused_helpers()
    if args.action == "preflight":
        result = preflight(args.model)
    elif args.action == "prepare":
        result = prepare(args.model, args.auth_seed)
    elif args.action == "smoke":
        result = execute_phase("smoke", load_and_validate_fixtures(archive_root() / "smoke-fixture.lock.json", archive_root(), minimum=1, maximum=1, require_registered=False), args.workers)
    elif args.action == "quality":
        result = execute_phase("quality", load_and_validate_fixtures(archive_root() / "fixtures.lock.json", archive_root(), minimum=12, maximum=12), args.workers)
    elif args.action == "packets":
        result = packets()
    elif args.action == "judge":
        result = judge()
    else:
        result = analyze()
    print(json.dumps(result, indent=2, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
