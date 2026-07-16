from __future__ import annotations

import importlib.util
import os
import sys
import tempfile
import unittest
from unittest import mock
from pathlib import Path
import json
import hashlib
import subprocess


EXPERIMENTS = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(EXPERIMENTS))
ENTRY_SPEC = importlib.util.spec_from_file_location("report_plan_mcp_experiment", EXPERIMENTS / "report_plan_mcp_experiment.py")
entry = importlib.util.module_from_spec(ENTRY_SPEC)
assert ENTRY_SPEC.loader
ENTRY_SPEC.loader.exec_module(entry)
JUDGE_SPEC = importlib.util.spec_from_file_location("codex_blind_judge", EXPERIMENTS / "codex_blind_judge.py")
codex_judge = importlib.util.module_from_spec(JUDGE_SPEC)
assert JUDGE_SPEC.loader
JUDGE_SPEC.loader.exec_module(codex_judge)

from report_plan_mcp.audit import REQUIRED_ZERO, arm_order, audit_lineage, collect_hard_metrics, hard_gate, validate_pair
from report_plan_mcp.builds import build_command, source_commands, version_command
from report_plan_mcp.judging import FINAL_DIMENSIONS, PLAN_DIMENSIONS, aggregate, aggregate_packet_scores, build_blind_packets, calibration_passes, needs_third_call, score_packets
from report_plan_mcp.fault_seed import materialize_isolation_environment
from report_plan_mcp.mcp_faults import FAULT_CASES, STATEFUL_CASES, fault_case_environment, run_fault_matrix, run_stdio
from report_plan_mcp.models import (
    ARCHIVE_SUFFIX, BASELINE_COMMIT, CANDIDATE_COMMIT, EXPERIMENT_ID, PREFLIGHT_MODEL, PREREGISTERED_TOPICS, Fixture, RunManifest, RunSpec,
    effort_for_mode, freeze_fixture_manifest, load_and_validate_fixtures, model_for_mode,
    validate_provider_efforts, validate_provider_models,
)
from report_plan_mcp.product_path import ProductRunError, assert_public_product_path, product_commands
from report_plan_mcp.provenance import _validate_phase_matrix, assemble_records, build_pairs
import report_plan_mcp.provenance as provenance
from report_plan_mcp.recovery import classify_failure, recovery_command, replacement_allowed
from report_plan_mcp.safety import (
    IsolationError,
    ensure_unique_namespace,
    isolated_environment,
    validate_archive,
    validate_endpoint,
    validate_environment,
    validate_run_paths,
)
from report_plan_mcp.statistics import analyze_confirmatory, exact_sign_pvalue, freeze_sample_size, guardrail, holm_adjust, mode_claim, overall_claim, paired_wilcoxon_pvalue, percentile_lower, reestimated_topics


class IsolationTests(unittest.TestCase):
    def test_rejects_noncanonical_and_sibling_archives(self):
        with tempfile.TemporaryDirectory() as directory:
            home = Path(directory)
            good = home / ARCHIVE_SUFFIX
            self.assertEqual(validate_archive(good, home), good.resolve())
            for bad in (home / "other", good.parent / "14-other"):
                with self.assertRaises(IsolationError):
                    validate_archive(bad, home)

    def test_rejects_dev_release_paths_nonloopback_and_ports(self):
        with tempfile.TemporaryDirectory() as directory:
            archive = Path(directory) / ARCHIVE_SUFFIX
            run = archive / "runs" / "one"
            with self.assertRaises(IsolationError):
                validate_run_paths(run, run / "runtime/dev-6002/plasma-ui-user.db", run / "artifacts", run / "work", archive)
            for host, port in (("0.0.0.0", 6200), ("127.0.0.1", 6002), ("127.0.0.1", 3002)):
                with self.assertRaises(IsolationError):
                    validate_endpoint(host, port)

    def test_effective_environment_is_archive_local_and_all_xdg_keys_are_replaced(self):
        with tempfile.TemporaryDirectory() as directory:
            run = Path(directory) / "run"
            inherited = {"HOME": "/real/home", "CODEX_HOME": "/real/codex", "XDG_CACHE_HOME": "/real/cache", "XDG_EXTRA": "/real/extra"}
            env = isolated_environment(run, inherited)
            validate_environment(env, run, inherited)
            self.assertEqual(entry.effective_child_environment(env), env)
            self.assertTrue(all(value is None or str(run) in value for value in env.values()))
            leaked = dict(env, CODEX_HOME="/real/codex")
            with self.assertRaises(IsolationError):
                validate_environment(leaked, run, inherited)

    def test_duplicate_namespace_is_rejected(self):
        existing = set()
        ensure_unique_namespace("one", existing)
        with self.assertRaises(IsolationError):
            ensure_unique_namespace("one", existing)


class ManifestTests(unittest.TestCase):
    @staticmethod
    def _write_prepare_gate(archive: Path, candidate: str = CANDIDATE_COMMIT, identical_binary: bool = False) -> None:
        baseline_source = archive / "source-manifests" / f"baseline-{BASELINE_COMMIT}.tar"
        candidate_source = archive / "source-manifests" / f"candidate-{candidate}.tar"
        baseline_binary = archive / "bin" / "baseline" / "plasma"
        candidate_binary = archive / "bin" / "candidate" / "plasma"
        for path, content in (
            (baseline_source, b"baseline source"), (candidate_source, b"candidate source"),
            (baseline_binary, b"baseline binary"), (candidate_binary, b"baseline binary" if identical_binary else b"candidate binary"),
        ):
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_bytes(content)
        gate = {
            "passed": True,
            "controller_commit": "controller-commit",
            "baseline_commit": BASELINE_COMMIT,
            "candidate_commit": candidate,
            "models": {"codex": "codex-model-id"},
            "efforts": {"codex": "high"},
            "fixture_manifest_sha256": "fixtures",
            "smoke_fixture_manifest_sha256": "smoke",
            "calibration_fixture_manifest_sha256": "calibration",
            "baseline": {"arm": "baseline", "commit": BASELINE_COMMIT, "source_archive": str(baseline_source), "source_sha256": hashlib.sha256(baseline_source.read_bytes()).hexdigest(), "binary": str(baseline_binary), "binary_sha256": hashlib.sha256(baseline_binary.read_bytes()).hexdigest()},
            "candidate": {"arm": "candidate", "commit": candidate, "source_archive": str(candidate_source), "source_sha256": hashlib.sha256(candidate_source.read_bytes()).hexdigest(), "binary": str(candidate_binary), "binary_sha256": hashlib.sha256(candidate_binary.read_bytes()).hexdigest()},
        }
        path = archive / "control" / "prepare-gate.json"
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(json.dumps(gate), encoding="utf-8")

    def test_prepare_rejects_dirty_or_unmatched_candidate_and_builds_clean_lock(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            repo, archive = root / "repo", root / ARCHIVE_SUFFIX
            (repo / "plasma/cmd/plasma").mkdir(parents=True)
            (repo / "plasma/go.mod").write_text("module example.com/plasma\n\ngo 1.22\n", encoding="utf-8")
            (repo / "plasma/cmd/plasma/main.go").write_text(
                'package main\nimport ("fmt"; "os")\nvar commit string\nfunc main(){if len(os.Args)>1 && os.Args[1]=="version" {fmt.Println(commit)}}\n',
                encoding="utf-8",
            )
            subprocess.run(["git", "init", "-q"], cwd=repo, check=True)
            subprocess.run(["git", "config", "user.email", "test@example.com"], cwd=repo, check=True)
            subprocess.run(["git", "config", "user.name", "Test"], cwd=repo, check=True)
            subprocess.run(["git", "add", "."], cwd=repo, check=True)
            subprocess.run(["git", "commit", "-qm", "baseline"], cwd=repo, check=True)
            baseline = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=repo, text=True).strip()
            (repo / "marker").write_text("candidate\n", encoding="utf-8")
            subprocess.run(["git", "add", "."], cwd=repo, check=True)
            subprocess.run(["git", "commit", "-qm", "candidate"], cwd=repo, check=True)
            candidate = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=repo, text=True).strip()
            (repo / "controller-marker").write_text("controller only\n", encoding="utf-8")
            subprocess.run(["git", "add", "."], cwd=repo, check=True)
            subprocess.run(["git", "commit", "-qm", "controller"], cwd=repo, check=True)
            controller = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=repo, text=True).strip()
            fixtures = []
            for topic in PREREGISTERED_TOPICS[:12]:
                source = archive / "fixtures" / f"{topic}.txt"
                source.parent.mkdir(parents=True, exist_ok=True)
                source.write_text(topic, encoding="utf-8")
                fixtures.append(self._fixture_row(topic, source))
            smoke_source = archive / "fixtures/smoke.txt"
            smoke_source.write_text("smoke", encoding="utf-8")
            calibration_rows = []
            for index in range(5):
                source = archive / "fixtures" / f"calibration-{index}.txt"
                source.write_text(f"calibration {index}", encoding="utf-8")
                calibration_rows.append(self._fixture_row(f"calibration-{index}", source))
            fixture_manifest, smoke_manifest, calibration_manifest = archive / "fixtures.json", archive / "smoke.json", archive / "calibration.json"
            fixture_manifest.write_text(json.dumps({"fixtures": fixtures}), encoding="utf-8")
            smoke_manifest.write_text(json.dumps({"fixtures": [self._fixture_row("smoke-topic", smoke_source)]}), encoding="utf-8")
            calibration_manifest.write_text(json.dumps({"fixtures": calibration_rows}), encoding="utf-8")
            config = {
                "controller_commit": controller,
                "candidate_commit": candidate,
                "models": {"codex": "codex-model-id"},
                "efforts": {"codex": "high"},
                "fixture_manifest": str(fixture_manifest),
                "smoke_fixture_manifest": str(smoke_manifest),
                "calibration_fixture_manifest": str(calibration_manifest),
            }
            with mock.patch.object(entry, "BASELINE_COMMIT", baseline), mock.patch.object(entry, "CANDIDATE_COMMIT", candidate):
                with self.assertRaisesRegex(ValueError, "frozen experiment candidate"):
                    entry._prepare(config | {"candidate_commit": baseline}, repo, archive)
                (repo / "dirty").write_text("dirty", encoding="utf-8")
                with self.assertRaisesRegex(ValueError, "clean worktree"):
                    entry._prepare(config, repo, archive)
                (repo / "dirty").unlink()
                prepared = entry._prepare(config, repo, archive)
            self.assertEqual(prepared["candidate"]["commit"], candidate)
            self.assertEqual(prepared["controller_commit"], controller)
            self.assertTrue(Path(prepared["candidate"]["binary"]).is_file())
            gate = json.loads((archive / "control" / "prepare-gate.json").read_text())
            self.assertEqual((gate["baseline_commit"], gate["candidate_commit"]), (baseline, candidate))
            self.assertNotEqual(gate["baseline"]["binary_sha256"], gate["candidate"]["binary_sha256"])
            for arm in ("baseline", "candidate"):
                self.assertTrue(gate[arm]["source_sha256"])
                self.assertTrue(gate[arm]["binary_sha256"])

    def test_provider_model_mapping_is_closed_selected_and_checked_before_prepare(self):
        self.assertEqual(EXPERIMENT_ID, "17-report-plan-mcp-focused-2026-07-14")
        self.assertEqual(ARCHIVE_SUFFIX.name, EXPERIMENT_ID)
        models = {"codex": "codex-model-id"}
        self.assertEqual(validate_provider_models(models), models)
        self.assertEqual(model_for_mode("planned", models), "codex-model-id")
        self.assertEqual(model_for_mode("long_form", models), "codex-model-id")
        self.assertEqual(PREFLIGHT_MODEL, "preflight-only-not-a-codex-model")
        fixture = Fixture("topic", "title", "objective", Path("source"), "hash", "license", "url")
        with tempfile.TemporaryDirectory() as directory:
            archive = Path(directory)
            self._write_prepare_gate(archive)
            config = {
                "controller_commit": "controller-commit",
                "candidate_commit": CANDIDATE_COMMIT,
                "models": models,
                "efforts": {"codex": "high"},
            }
            planned = entry._spec(config, archive, fixture, "baseline", "planned", 1, "n")
            long_form = entry._spec(config, archive, fixture, "candidate", "long_form", 1, "n")
            self.assertEqual((planned.executor, planned.model, planned.commit), ("codex", "codex-model-id", BASELINE_COMMIT))
            self.assertEqual((long_form.executor, long_form.model, long_form.commit), ("codex", "codex-model-id", CANDIDATE_COMMIT))
            self.assertEqual((planned.effort, long_form.effort), ("high", "high"))
            with self.assertRaisesRegex(ValueError, "frozen experiment candidate"):
                entry._prepare_gate(config | {"candidate_commit": "wrong-candidate"}, archive)

        malformed = (
            None,
            {},
            {},
            {"codex": ""},
            {"codex": 1},
            {"codex": "codex-model-id", "claude": "claude-model-id"},
        )
        for value in malformed:
            with self.subTest(value=value), self.assertRaises(ValueError):
                validate_provider_models(value)
        with self.assertRaisesRegex(ValueError, "single model config"):
            entry._provider_models({"model": "shared-model", "models": models})
        self.assertEqual(validate_provider_efforts({"codex": "high"}), {"codex": "high"})
        self.assertEqual(effort_for_mode("planned", {"codex": "high"}), "high")
        self.assertEqual(effort_for_mode("long_form", {"codex": "high"}), "high")
        for value in (None, {}, {"codex": ""}, {"codex": "low"}, {"codex": "high", "claude": "high"}):
            with self.subTest(efforts=value), self.assertRaises(ValueError):
                validate_provider_efforts(value)
        with self.assertRaisesRegex(ValueError, "single effort config"):
            entry._provider_efforts({"effort": "high", "efforts": {"codex": "high"}})
        with mock.patch.object(entry.subprocess, "check_output") as check_output:
            with self.assertRaisesRegex(ValueError, "models must contain exactly"):
                entry._prepare({"candidate_commit": "candidate"}, Path("repo"), Path("archive"))
            check_output.assert_not_called()
        with tempfile.TemporaryDirectory() as directory:
            archive = Path(directory)
            self._write_prepare_gate(archive, identical_binary=True)
            with self.assertRaisesRegex(ValueError, "binary hashes must differ"):
                entry._prepare_gate({
                    "controller_commit": "controller-commit",
                    "candidate_commit": CANDIDATE_COMMIT,
                    "models": {"codex": "codex-model-id"},
                    "efforts": {"codex": "high"},
                }, archive)

    def test_preflight_selects_explicit_provider_model(self):
        script = EXPERIMENTS / "report_plan_mcp_experiment.py"
        with tempfile.TemporaryDirectory() as directory:
            environment = dict(os.environ, HOME=directory)
            common = [
                sys.executable, str(script), "--preflight", "--dry-run",
                "--codex-model", "codex-model-id",
            ]
            planned = json.loads(subprocess.check_output([*common, "--mode", "planned"], text=True, env=environment))
            long_form = json.loads(subprocess.check_output([*common, "--mode", "long_form"], text=True, env=environment))
        self.assertEqual((planned["executor"], planned["model"]), ("codex", "codex-model-id"))
        self.assertEqual((long_form["executor"], long_form["model"]), ("codex", "codex-model-id"))

    @staticmethod
    def _fixture_row(topic: str, source: Path) -> dict[str, str]:
        return {
            "topic": topic, "title": "Title", "objective": "Objective", "source_bundle": str(source),
            "source_sha256": hashlib.sha256(source.read_bytes()).hexdigest(), "license": "CC0",
            "license_url": "https://example.com/license", "retrieved_at": "2026-07-13",
        }

    def test_build_and_fixture_freeze_contracts(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            archive = root / ARCHIVE_SUFFIX
            source = archive / "fixtures/source.txt"
            source.parent.mkdir(parents=True)
            source.write_text("public fixture", encoding="utf-8")
            digest = hashlib.sha256(source.read_bytes()).hexdigest()
            fixture_manifest = archive / "fixtures.json"
            fixture_manifest.write_text(json.dumps({"fixtures": [{
                "topic": PREREGISTERED_TOPICS[0], "title": "Title", "objective": "Objective",
                "source_bundle": str(source), "source_sha256": digest, "license": "CC0",
                "license_url": "https://example.com/license", "retrieved_at": "2026-07-13",
            }]}), encoding="utf-8")
            fixtures = load_and_validate_fixtures(fixture_manifest, archive, minimum=1)
            self.assertEqual(freeze_fixture_manifest(fixtures, archive / "fixtures.lock.json"), hashlib.sha256((archive / "fixtures.lock.json").read_bytes()).hexdigest())
            commands = source_commands(root, archive, "commit", "candidate")
            self.assertIn("git", commands[0])
            self.assertIn("archive", commands[0])
            self.assertIn("./cmd/plasma", build_command(archive / "source", archive / "bin/plasma", "commit"))
            self.assertEqual(version_command(archive / "bin/plasma")[-1], "version")

    def test_dry_run_manifest_uses_only_public_product_paths(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            commands = product_commands(
                root / "bin/plasma", root / "run", 6200, "http://127.0.0.1:6201",
                "planned", "codex", "codex-model-id", "high",
            )
            assert_public_product_path(commands)
            text = "\n".join(" ".join(command) for command in commands)
            for value in ("missions create", "sources attach-local", "serve", "/reports", "/events", "mcp"):
                self.assertIn(value, text)
            self.assertNotIn("internal/", text)

    def test_manifest_records_required_fields(self):
        with tempfile.TemporaryDirectory() as directory:
            home = Path(directory)
            archive = home / ARCHIVE_SUFFIX
            with self.assertRaisesRegex(ValueError, "must be codex"):
                RunSpec(PREREGISTERED_TOPICS[0], 1, "candidate", "long_form", "claude", "abc", archive / "bin/candidate/plasma", "claude-model-id", "high", "sources", 10, 20, "same_session", archive / "fixtures/topic-01", "hash", "rejected")
            spec = RunSpec(PREREGISTERED_TOPICS[0], 1, "candidate", "long_form", "codex", "abc", archive / "bin/candidate/plasma", "codex-model-id", "high", "sources", 10, 20, "same_session", archive / "fixtures/topic-01", "hash", "nonce")
            manifest_object = entry.build_manifest(spec, home, {"XDG_CACHE_HOME": "/raw"}, set(), set())
            manifest = manifest_object.as_dict()
            required = {"topic", "replicate", "arm", "mode", "executor", "commit", "binary_hash", "model", "effort", "source_policy", "source_bundle", "source_hash", "budgets", "selected_session_policy", "database", "artifact_root", "workdir", "port", "connector_port", "connector_url", "namespace", "child_environment", "mission_id", "process_id", "connector_process_id", "start_boundary", "terminal_status", "ledger_hash", "result_hash"}
            self.assertTrue(required.issubset(manifest))
            self.assertEqual(manifest["executor"], "codex")
            self.assertEqual(manifest["model"], "codex-model-id")
            self.assertNotIn(manifest["connector_port"], {3011, 6011})
            command_text = " ".join(" ".join(command) for command in manifest["commands"])
            self.assertIn("-agent codex,claude", command_text)
            self.assertIn("-liquid2-url http://127.0.0.1:", command_text)
            self.assertIn('"agent_executor":"codex"', command_text)
            self.assertIn('"agent_model":"codex-model-id"', command_text)
            self.assertIn('"agent_reasoning_effort":"high"', command_text)
            with self.assertRaisesRegex(ValueError, "effort must be high"):
                RunSpec(PREREGISTERED_TOPICS[0], 1, "candidate", "long_form", "codex", "abc", archive / "bin/candidate/plasma", "codex-model-id", "", "sources", 10, 20, "same_session", archive / "fixtures/topic-01", "hash", "invalid")
            self.assertIn(manifest["connector_url"], recovery_command(manifest_object, archive / "fixture"))
            self.assertIn("codex,claude", recovery_command(manifest_object, archive / "fixture"))
            self.assertEqual(len(PREREGISTERED_TOPICS), 24)


class ProtocolLogicTests(unittest.TestCase):
    def test_blind_payload_uses_arm_specific_plan_provenance(self):
        with tempfile.TemporaryDirectory() as directory:
            archive = Path(directory)

            def run_for(arm: str, events: list[dict[str, object]], namespace: str | None = None) -> tuple[dict[str, object], Path]:
                root = archive / "runs" / (namespace or arm)
                artifact = root / "artifacts" / "report.txt"
                artifact.parent.mkdir(parents=True)
                artifact.write_text(f"{arm} report", encoding="utf-8")
                ledger = root / "ledger.events.json"
                ledger.write_text(json.dumps({"events": events}), encoding="utf-8")
                return {
                    "namespace": namespace or arm, "arm": arm,
                    "result_hash": hashlib.sha256(artifact.read_bytes()).hexdigest(),
                    "ledger_hash": hashlib.sha256(ledger.read_bytes()).hexdigest(),
                }, ledger

            baseline, baseline_ledger = run_for("baseline", [{"EventType": "report.plan.created", "Payload": {"plan": {"origin": "json"}}}])
            candidate, _ = run_for("candidate", [
                {"EventType": "report.plan.submitted", "Payload": {"plan": {"origin": "mcp"}}},
                {"EventType": "report.plan.created", "Payload": {"plan": {"origin": "canonical"}}},
            ])
            self.assertEqual(provenance._judge_payload(archive, baseline)["plan"], {"origin": "json"})
            self.assertEqual(provenance._judge_payload(archive, candidate)["plan"], {"origin": "mcp"})

            leaked, _ = run_for("baseline", [{"EventType": "report.plan.submitted", "Payload": {"plan": {"origin": "mcp"}}}], "leaked")
            with self.assertRaisesRegex(ValueError, "candidate MCP submission"):
                provenance._judge_payload(archive, leaked)
            missing, _ = run_for("candidate", [{"EventType": "report.plan.created", "Payload": {"plan": {"origin": "json"}}}], "missing")
            with self.assertRaisesRegex(ValueError, "MCP submitted"):
                provenance._judge_payload(archive, missing)
            baseline_ledger.write_text("{}", encoding="utf-8")
            with self.assertRaisesRegex(ValueError, "ledger hash"):
                provenance._judge_payload(archive, baseline)

    def test_provenance_pipeline_uses_only_complete_immutable_experiment_files(self):
        with tempfile.TemporaryDirectory() as directory:
            archive = Path(directory)
            runs = []
            for topic_index in range(4):
                topic = f"topic-{topic_index}"
                for replicate in (1, 2):
                    for mode, executor in (("planned", "codex"), ("long_form", "codex")):
                        for arm in ("baseline", "candidate"):
                            namespace = f"{topic}-r{replicate}-{mode}-{arm}"
                            root = archive / "runs" / namespace
                            artifact = root / "artifacts/report.txt"
                            artifact.parent.mkdir(parents=True)
                            artifact.write_text("frozen report", encoding="utf-8")
                            result_hash = hashlib.sha256(artifact.read_bytes()).hexdigest()
                            ledger = root / "ledger.events.json"
                            event_type = "report.plan.created" if arm == "baseline" else "report.plan.submitted"
                            ledger.write_text(json.dumps({"events": [{"EventType": event_type, "Payload": {"plan": {"summary": "frozen plan"}}}]}), encoding="utf-8")
                            ledger_hash = hashlib.sha256(ledger.read_bytes()).hexdigest()
                            model = "codex-model-id"
                            commit = BASELINE_COMMIT if arm == "baseline" else CANDIDATE_COMMIT
                            binary_hash = hashlib.sha256((b"baseline binary" if arm == "baseline" else b"candidate binary")).hexdigest()
                            record = {
                                "experiment": "experiment", "topic": topic, "replicate": replicate, "arm": arm,
                                "mode": mode, "executor": executor, "commit": commit, "binary_hash": binary_hash, "model": model,
                                "effort": "high", "source_policy": "sources", "source_bundle": "fixture", "source_hash": "source", "budgets": {"tokens": 1},
                                "selected_session_policy": "same_session", "database": str(root / "state/plasma.db"),
                                "artifact_root": str(root / "artifacts"), "workdir": str(root / "workdir"), "port": 6200,
                                "connector_port": 6201, "connector_url": "http://127.0.0.1:6201", "namespace": namespace,
                                "child_environment": {"HOME": str(root / "home")}, "mission_id": "mis_test", "process_id": 10,
                                "connector_process_id": 11, "commands": [], "start_boundary": "started:product_cli_mission_create",
                                "terminal_status": "completed", "ledger_hash": ledger_hash, "result_hash": result_hash, "started": True,
                                "artifact_presence": 1, "machine_metrics": {"artifact_presence": 1},
                            }
                            (root / "manifest.terminal.json").write_text(json.dumps(record), encoding="utf-8")
                            runs.append(record)
            control = archive / "control"
            control.mkdir()
            ManifestTests._write_prepare_gate(archive)
            (control / "pilot-gate.json").write_text(json.dumps({"passed": True, "runs": runs}), encoding="utf-8")
            pairs, loaded = build_pairs(archive, ("pilot",))
            self.assertEqual(len(pairs), 16)
            self.assertEqual(len(loaded), 32)
            prepare_path = control / "prepare-gate.json"
            prepare_gate = json.loads(prepare_path.read_text())
            mismatched_gate = json.loads(json.dumps(prepare_gate))
            mismatched_gate["candidate_commit"] = "other-candidate"
            mismatched_gate["candidate"]["commit"] = "other-candidate"
            prepare_path.write_text(json.dumps(mismatched_gate), encoding="utf-8")
            with self.assertRaisesRegex(ValueError, "invalid commit provenance"):
                build_pairs(archive, ("pilot",))
            identical_gate = json.loads(json.dumps(prepare_gate))
            identical_gate["candidate"]["binary_sha256"] = identical_gate["baseline"]["binary_sha256"]
            prepare_path.write_text(json.dumps(identical_gate), encoding="utf-8")
            with self.assertRaisesRegex(ValueError, "identical arm binaries"):
                build_pairs(archive, ("pilot",))
            prepare_path.write_text(json.dumps(prepare_gate), encoding="utf-8")
            packets = archive / "judging/pilot-packets"
            build_blind_packets(pairs, packets, 110)
            scores = archive / "judging/pilot-scores"
            scores.mkdir()
            dimensions = PLAN_DIMENSIONS + FINAL_DIMENSIONS
            for packet in packets.glob("*.json"):
                packet_id = json.loads(packet.read_text())["packet_id"]
                value = {name: 3.0 for name in dimensions}
                (scores / f"{packet_id}.json").write_text(json.dumps({"packet_id": packet_id, "scores": {"A": value, "B": value}}), encoding="utf-8")
            records = assemble_records(archive, ("pilot",), scores, archive / "judging/pilot-packets-mapping.private.json")
            self.assertEqual(len(records), 32)
            next(scores.glob("*.json")).unlink()
            with self.assertRaises(ValueError):
                assemble_records(archive, ("pilot",), scores, archive / "judging/pilot-packets-mapping.private.json")

    def test_worker_ramp_is_enforced_before_execution(self):
        fixture = Fixture("smoke", "title", "objective", Path("source"), "hash", "license", "url")
        with tempfile.TemporaryDirectory() as directory:
            archive = Path(directory)
            with self.assertRaisesRegex(ValueError, "exactly two workers"):
                entry._execute_phase("smoke", {}, archive, (fixture,), 1)
            with self.assertRaisesRegex(ValueError, "at most six workers"):
                entry._execute_phase("pilot", {}, archive, (fixture,) * 4, 7)
            with self.assertRaisesRegex(ValueError, "between one and six"):
                entry._execute_phase("quality", {}, archive, (fixture,), 7)

    def test_focused_quality_requires_smoke_and_exactly_twelve_single_replicate_topics(self):
        fixture = Fixture("focused", "title", "objective", Path("source"), "hash", "license", "url")
        with tempfile.TemporaryDirectory() as directory:
            archive = Path(directory)
            with self.assertRaisesRegex(ValueError, "exactly 12"):
                entry._execute_phase("focused-quality", {}, archive, (fixture,) * 11, 1)
            with self.assertRaisesRegex(ValueError, "between one and six"):
                entry._execute_phase("focused-quality", {}, archive, (fixture,) * 12, 7)
            with mock.patch.object(entry, "require_gates", side_effect=ValueError("missing smoke")) as require:
                with self.assertRaisesRegex(ValueError, "missing smoke"):
                    entry._execute_phase("focused-quality", {}, archive, (fixture,) * 12, 1)
            require.assert_called_once_with(archive, "smoke-gate.json")
            with mock.patch.object(entry, "require_gates") as require:
                entry._require_focused_quality_gate(archive)
            require.assert_called_once_with(archive, "smoke-gate.json", "focused-quality-gate.json")

        rows = [
            {"topic": f"topic-{topic}", "replicate": 1, "mode": mode, "arm": arm, "executor": "codex"}
            for topic in range(12) for mode in ("planned", "long_form") for arm in ("baseline", "candidate")
        ]
        _validate_phase_matrix("focused-quality", rows)
        with self.assertRaisesRegex(ValueError, "incomplete"):
            _validate_phase_matrix("focused-quality", rows[:-1])

    def test_focused_judge_schema_is_blind_and_locked(self):
        dimensions = PLAN_DIMENSIONS + FINAL_DIMENSIONS
        packet = {"packet_id": "packet", "topic": "topic", "replicate": 1, "mode": "planned", "A": {"plan": {}, "report": "A"}, "B": {"plan": {}, "report": "B"}}
        self.assertEqual(codex_judge.validate_packet(packet), packet)
        scores = {label: {name: 3 for name in dimensions} for label in ("A", "B")}
        self.assertEqual(set(codex_judge.validate_scores(scores)), {"A", "B"})
        with self.assertRaisesRegex(ValueError, "private provenance"):
            codex_judge.validate_packet({**packet, "A": {"baseline": "leak"}})
        command = codex_judge._command("judge-model", "high", "rubric", Path("schema"), Path("output"))
        self.assertIn("--ephemeral", command)
        self.assertIn("--json", command)
        self.assertEqual(command[command.index("--sandbox") + 1], "read-only")
        codex_judge._assert_no_tools('{"type":"thread.started"}\n')
        with self.assertRaisesRegex(ValueError, "attempted to use a tool"):
            codex_judge._assert_no_tools('{"item":{"type":"function_call"}}\n')
        with self.assertRaisesRegex(ValueError, "attempted to use a tool"):
            codex_judge._assert_no_tools('{"item":{"type":"web_search_call"}}\n')
        self.assertEqual(entry._focused_judge_command({"model": "judge-model", "effort": "high"}, Path("rubric"))[3], "judge-model")
        with self.assertRaises(ValueError):
            entry._focused_judge_settings({"focused_judge": {"model": "judge-model", "effort": "low"}})

    def test_focused_itt_partial_packets_and_protocol_locks_are_recoverable(self):
        dimensions = PLAN_DIMENSIONS + FINAL_DIMENSIONS
        completed = {"topic": "topic", "replicate": 1, "mode": "planned", "arm": "baseline", "terminal_status": "completed"}
        failed = {"topic": "topic", "replicate": 1, "mode": "planned", "arm": "candidate", "terminal_status": "itt_failure"}
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            mapping = root / "mapping.json"
            mapping.write_text("[]", encoding="utf-8")
            with mock.patch.object(provenance, "build_pairs", return_value=([], [completed, failed])):
                records = assemble_records(root, ("focused-quality",), root / "scores", mapping)
            self.assertEqual(records[0]["itt_score_reason"], "paired-arm-not-completed")
            self.assertEqual(records[0]["scores"]["plan"], {name: 1.0 for name in PLAN_DIMENSIONS})
            self.assertNotIn("scores", records[1])

            packets = root / "judging/focused-packets"
            packets.mkdir(parents=True)
            (packets / "one.json").write_text("{}", encoding="utf-8")
            (root / "judging/focused-packets-mapping.private.json").write_text("[]", encoding="utf-8")
            packet_lock = entry._focused_packet_lock(root, "protocol-hash")
            self.assertEqual(packet_lock["packet_count"], 1)
            scores = root / "judging/focused-judge/attempt-2/scores"
            scores.mkdir(parents=True)
            (scores / "one.json").write_text("{}", encoding="utf-8")
            packet_lock_hash = hashlib.sha256(json.dumps(packet_lock, sort_keys=True).encode()).hexdigest()
            score_manifest = {"one.json": hashlib.sha256(b"{}").hexdigest()}
            (root / "control/focused-judge-gate.json").write_text(json.dumps({
                "passed": True, "attempt": 2, "protocol_lock_sha256": "protocol-hash",
                "packet_lock_sha256": packet_lock_hash,
                "score_manifest_sha256": hashlib.sha256(json.dumps(score_manifest, sort_keys=True).encode()).hexdigest(),
                "score_count": 1,
            }), encoding="utf-8")
            self.assertEqual(entry._focused_judge_gate(root, "protocol-hash", packet_lock)["attempt"], 2)

            repo = Path(__file__).resolve().parents[4]
            config = {"seed": 110, "focused_judge": {"model": "judge-model", "effort": "high"}}
            schedule = {"seed": 110, "entries": []}
            with mock.patch.object(entry, "_focused_controller_commit", return_value="quality-controller"), mock.patch.object(entry, "_focused_execution_schedule", return_value=schedule):
                lock, lock_hash = entry._focused_protocol_lock(config, root, repo)
            self.assertEqual(lock["seed"], 110)
            self.assertEqual(lock["quality_controller_commit"], "quality-controller")
            with mock.patch.object(entry, "_focused_controller_commit", return_value="quality-controller"), mock.patch.object(entry, "_focused_execution_schedule", return_value=schedule):
                with self.assertRaisesRegex(ValueError, "differs"):
                    entry._focused_protocol_lock({**config, "seed": 111}, root, repo)
            self.assertEqual(set(codex_judge._schema()["properties"]), {"A", "B"})
            self.assertEqual(len(lock["rubric_sha256"]), 64)
            self.assertEqual(lock["schema_sha256"], codex_judge.schema_sha256())

    def test_focused_schedule_is_seeded_and_counterbalanced_per_mode(self):
        fixtures = tuple(Fixture(f"topic-{index}", "title", "objective", Path("source"), "hash", "license", "url") for index in range(12))
        schedule = entry._build_focused_schedule(fixtures, 110)
        self.assertEqual(schedule, entry._build_focused_schedule(fixtures, 110))
        self.assertNotEqual(schedule, entry._build_focused_schedule(fixtures, 111))
        entries = schedule["entries"]
        self.assertEqual(len(entries), 24)
        for mode in ("planned", "long_form"):
            selected = [item for item in entries if item["mode"] == mode]
            self.assertEqual(sum(item["arms"][0] == "baseline" for item in selected), 6)
            self.assertEqual(sum(item["arms"][0] == "candidate" for item in selected), 6)

    def test_focused_controller_requires_clean_worktree(self):
        with mock.patch.object(entry.subprocess, "check_output", side_effect=["controller\n", ""]):
            self.assertEqual(entry._focused_controller_commit(Path("repo")), "controller")
        with mock.patch.object(entry.subprocess, "check_output", side_effect=["controller\n", " M file\n"]):
            with self.assertRaisesRegex(ValueError, "clean worktree"):
                entry._focused_controller_commit(Path("repo"))

    def test_focused_followup_accepts_only_matching_recovery_controller(self):
        original = {
            "seed": 110, "judge": {"model": "judge", "effort": "high"},
            "quality_controller_commit": "original", "execution_schedule_sha256": "schedule",
            "rubric_sha256": "rubric", "adapter_sha256": "adapter", "schema_sha256": "schema",
        }
        current = {**original, "quality_controller_commit": "recovery"}
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "control").mkdir()
            protocol_path = root / "control/focused-protocol.lock.json"
            protocol_path.write_text(json.dumps(original), encoding="utf-8")
            recovery = {
                "recovery_controller_commit": "recovery", "original_protocol_lock_sha256": hashlib.sha256(protocol_path.read_bytes()).hexdigest(),
                "terminal_manifest_sha256": {"run": "terminal"}, "initial_manifest_sha256": {"run": "initial"},
                "run_intervals_ns": {"run": [1, 2]}, "first_start_mtime_ns": 1, "protected_path_mtime_ns": {}, "workers": 6,
            }
            (root / "control/focused-quality-gate.json").write_text(json.dumps({"passed": True, "recovered": True}), encoding="utf-8")
            (root / "control/focused-quality-recovery.lock.json").write_text(json.dumps(recovery), encoding="utf-8")
            evidence = {"terminal_manifest_sha256": {"run": "terminal"}, "initial_manifest_sha256": {"run": "initial"}, "run_intervals_ns": {"run": [1, 2]}, "workers": 6}
            with mock.patch.object(entry, "_focused_protocol_values", return_value=current), mock.patch.object(entry, "_focused_run_evidence", return_value=evidence):
                lock, _ = entry._focused_followup_protocol_lock({}, root, Path("repo"))
            self.assertEqual(lock["quality_controller_commit"], "original")
            recovery["terminal_manifest_sha256"] = {}
            (root / "control/focused-quality-recovery.lock.json").write_text(json.dumps(recovery), encoding="utf-8")
            with mock.patch.object(entry, "_focused_protocol_values", return_value=current), mock.patch.object(entry, "_focused_run_evidence", return_value=evidence):
                with self.assertRaisesRegex(ValueError, "evidence differs"):
                    entry._focused_followup_protocol_lock({}, root, Path("repo"))
            recovery["terminal_manifest_sha256"] = {"run": "terminal"}
            recovery["recovery_controller_commit"] = "other"
            (root / "control/focused-quality-recovery.lock.json").write_text(json.dumps(recovery), encoding="utf-8")
            with mock.patch.object(entry, "_focused_protocol_values", return_value=current), mock.patch.object(entry, "_focused_run_evidence", return_value=evidence):
                with self.assertRaisesRegex(ValueError, "recovery controller"):
                    entry._focused_followup_protocol_lock({}, root, Path("repo"))

    def test_focused_followup_rejects_missing_or_changed_recovery_evidence(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "control").mkdir()
            protocol = {"quality_controller_commit": "original", "seed": 110}
            protocol_path = root / "control/focused-protocol.lock.json"
            protocol_path.write_text(json.dumps(protocol), encoding="utf-8")
            (root / "control/focused-quality-gate.json").write_text(json.dumps({"passed": True, "recovered": True}), encoding="utf-8")
            with mock.patch.object(entry, "_focused_protocol_values", return_value={**protocol, "quality_controller_commit": "recovery"}):
                with self.assertRaisesRegex(ValueError, "presence differ"):
                    entry._focused_followup_protocol_lock({}, root, Path("repo"))

    def test_focused_recovery_worker_count_uses_interval_overlap(self):
        self.assertEqual(entry._max_interval_overlap([(0, 5), (1, 4), (2, 3), (5, 6)]), 3)
        self.assertEqual(entry._max_interval_overlap([(0, 1), (1, 2)]), 1)

    def test_focused_judge_attempts_are_immutable_and_environment_is_contained(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            first_number, first = entry._next_focused_judge_attempt(root)
            second_number, second = entry._next_focused_judge_attempt(root)
            self.assertEqual((first_number, second_number), (1, 2))
            self.assertTrue(first.is_dir() and second.is_dir())
            source = root / "source-codex"
            source.mkdir()
            (source / "auth.json").write_text("auth", encoding="utf-8")
            contained = codex_judge.contained_environment(root / "call", {"CODEX_HOME": str(source), "PATH": "/bin", "PRIVATE_MAPPING": "forbidden"})
            self.assertEqual(set(contained), {"CODEX_HOME", "HOME", "TMPDIR", "PATH", "LANG"})
            self.assertTrue(Path(contained["CODEX_HOME"]).resolve().is_relative_to((root / "call").resolve()))
            self.assertNotIn("PRIVATE_MAPPING", contained)

    def test_failure_replacement_and_hard_gates(self):
        self.assertEqual(classify_failure(False, "port"), "pre_run_infrastructure")
        self.assertTrue(replacement_allowed(2, "pre_run_infrastructure"))
        self.assertFalse(replacement_allowed(3, "pre_run_infrastructure"))
        self.assertEqual(classify_failure(True, "timeout"), "itt")
        valid = {name: 0 for name in REQUIRED_ZERO} | {"artifact_presence": 1, "fallback_count": 0, "binding_violation": 0}
        self.assertTrue(hard_gate(valid, candidate=True))
        self.assertFalse(hard_gate(valid | {"fallback_count": 1}, candidate=True))
        with self.assertRaises(ValueError):
            hard_gate({"artifact_presence": 1}, candidate=True)

    def test_lineage_audit(self):
        self.assertTrue(audit_lineage([
            {"EventID": "s", "EventType": "report.plan.submitted", "Payload": {"plan_hash": "h", "tool_session_id": "t"}},
            {"EventID": "c", "EventType": "report.plan.created", "Payload": {"plan_submission": {"submission_event_id": "s", "plan_hash": "h", "tool_session_id": "t"}}},
        ]))

    def test_machine_audit_requires_observed_source_session_and_isolation(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory) / "namespace"
            manifest = {
                "database": str(root / "state/db"), "artifact_root": str(root / "artifacts"),
                "workdir": str(root / "work"), "namespace": "namespace", "port": 6200,
                "connector_port": 6201, "connector_url": "http://127.0.0.1:6201", "executor": "codex", "mode": "planned",
                "process_id": 10, "connector_process_id": 11,
                "child_environment": {"HOME": str(root / "home")}, "mission_id": "mis_1",
            }
            events = [
                {"EventID": "p", "MissionID": "mis_1", "EventType": "report.draft.pending", "Payload": {}},
                {"EventID": "r", "MissionID": "mis_1", "EventType": "mcp.tool.called", "Payload": {"tool_name": "plasma.sources.read"}},
                {"EventID": "s", "MissionID": "mis_1", "EventType": "report.plan.submitted", "Payload": {"plan_hash": "h", "tool_session_id": "t"}},
                {"EventID": "c", "MissionID": "mis_1", "EventType": "report.plan.created", "Payload": {"agent_session_id": "provider", "returned_agent_session_id": "provider", "plan_submission": {"submission_event_id": "s", "plan_hash": "h", "tool_session_id": "t"}}},
                {"EventID": "a", "MissionID": "mis_1", "EventType": "report.artifact.created", "Payload": {}},
            ]
            self.assertTrue(hard_gate(collect_hard_metrics(events, True, True, manifest), True))
            with self.assertRaises(ValueError):
                collect_hard_metrics([event for event in events if event["EventID"] != "r"], True, True, manifest)

    def test_pair_lock_and_randomization(self):
        common = {"topic": "t", "replicate": 1, "mode": "planned", "executor": "codex", "source_hash": "h", "model": "codex-model-id", "effort": "high", "source_policy": "sources", "budgets": {"tokens": 1}, "selected_session_policy": "same_session"}
        baseline = {**common, "arm": "baseline", "commit": BASELINE_COMMIT, "binary_hash": "baseline-binary"}
        candidate = {**common, "arm": "candidate", "commit": "candidate-commit", "binary_hash": "candidate-binary"}
        validate_pair(baseline, candidate)
        long_form = {**common, "mode": "long_form"}
        validate_pair({**long_form, "arm": "baseline", "commit": BASELINE_COMMIT, "binary_hash": "baseline-binary"}, {**long_form, "arm": "candidate", "commit": "candidate-commit", "binary_hash": "candidate-binary"})
        with self.assertRaisesRegex(ValueError, "paired run conditions differ"):
            validate_pair(baseline, {**candidate, "model": "other-codex-model"})
        with self.assertRaises(ValueError):
            validate_pair(baseline, {**candidate, "source_hash": "other"})
        with self.assertRaisesRegex(ValueError, "commits must differ"):
            validate_pair(baseline, {**candidate, "commit": BASELINE_COMMIT})
        with self.assertRaisesRegex(ValueError, "binary hashes must differ"):
            validate_pair(baseline, {**candidate, "binary_hash": "baseline-binary"})
        self.assertEqual(arm_order("t", 1, "planned", 110), arm_order("t", 1, "planned", 110))

    def test_judge_calibration_disagreement_and_aggregation(self):
        first = [1, 2, 3, 4, 5] * 4
        self.assertTrue(calibration_passes(first, first))
        self.assertFalse(needs_third_call([3, 3], [3, 3], [(0, 1)]))
        self.assertTrue(needs_third_call([1, 3], [3, 3], [(0, 1)]))
        self.assertEqual(aggregate([1, 3], [3, 5]), [2, 4])
        self.assertEqual(aggregate([1, 3], [5, 1], [3, 2]), [3, 2])
        dimensions = PLAN_DIMENSIONS + FINAL_DIMENSIONS
        first = {name: 3.0 for name in dimensions}
        second = {name: 3.5 for name in dimensions}
        self.assertEqual(set(aggregate_packet_scores(first, second)), set(dimensions))
        second[dimensions[0]] = 5.0
        with self.assertRaises(ValueError):
            aggregate_packet_scores(first, second)
        third = {name: 4.0 for name in dimensions}
        self.assertEqual(aggregate_packet_scores(first, second, third)[dimensions[0]], 4.0)

    def test_preregistered_statistics(self):
        self.assertGreaterEqual(percentile_lower([0.5] * 12, 110), 0.49)
        self.assertTrue(mode_claim([0.5] * 12, [0.4] * 12, 110))
        self.assertTrue(overall_claim(True, True))
        self.assertFalse(overall_claim(True, False))
        self.assertEqual(reestimated_topics(0.1), 12)
        self.assertEqual(exact_sign_pvalue([1] * 12), 2 / 4096)
        self.assertLess(paired_wilcoxon_pvalue([1] * 12), 0.01)
        self.assertEqual(holm_adjust([0.01, 0.04]), [0.02, 0.04])
        self.assertTrue(guardrail(-0.5, 0.1, 0.2))
        self.assertFalse(guardrail(-0.51, 0.1, 0.1))
        with self.assertRaises(ValueError):
            reestimated_topics(1.0)

    def test_sample_lock_and_confirmatory_analysis_fail_closed(self):
        with tempfile.TemporaryDirectory() as directory:
            lock = freeze_sample_size({name: [2.9, 3.0, 3.1, 3.0] for name in ("planned_final", "planned_plan", "long_form_final", "long_form_plan")}, Path(directory) / "lock.json")
            self.assertTrue(lock["locked"])
        metrics_base = {name: 0.0 for name in REQUIRED_ZERO} | {"artifact_presence": 1.0}
        records = []
        for topic in ("a", "b", "c", "d"):
            for mode in ("planned", "long_form"):
                for arm, score in (("baseline", 3.0), ("candidate", 3.1)):
                    metrics = dict(metrics_base)
                    if arm == "candidate":
                        metrics.update(fallback_count=0.0, binding_violation=0.0)
                    records.append({
                        "topic": topic, "replicate": 1, "mode": mode, "arm": arm, "started": True,
                        "terminal_status": "completed", "artifact_presence": 1, "machine_metrics": metrics,
                        "scores": {"plan": {name: score for name in PLAN_DIMENSIONS}, "final": {name: score for name in FINAL_DIMENSIONS}},
                    })
        result = analyze_confirmatory(records, 110)
        self.assertTrue(result["overall_claim"])
        broken = [dict(record) for record in records]
        broken[0] = dict(broken[0], machine_metrics={"artifact_presence": 1})
        with self.assertRaises(ValueError):
            analyze_confirmatory(broken, 110)

    def test_focused_analysis_retains_separate_mode_claims(self):
        metrics_base = {name: 0.0 for name in REQUIRED_ZERO} | {"artifact_presence": 1.0}
        records = []
        for topic in range(12):
            for mode, candidate_score in (("planned", 3.1), ("long_form", 2.7)):
                for arm, score in (("baseline", 3.0), ("candidate", candidate_score)):
                    metrics = dict(metrics_base)
                    if arm == "candidate":
                        metrics.update(fallback_count=0.0, binding_violation=0.0)
                    records.append({
                        "topic": f"topic-{topic}", "replicate": 1, "mode": mode, "arm": arm, "started": True,
                        "terminal_status": "completed", "artifact_presence": 1, "machine_metrics": metrics,
                        "scores": {"plan": {name: score for name in PLAN_DIMENSIONS}, "final": {name: score for name in FINAL_DIMENSIONS}},
                    })
        entry._require_focused_records(records)
        result = analyze_confirmatory(records, 110)
        self.assertTrue(result["mode_claims"]["planned"])
        self.assertFalse(result["mode_claims"]["long_form"])
        self.assertFalse(result["overall_claim"])
        failed = [dict(record) for record in records]
        failed[1] = {key: value for key, value in failed[1].items() if key != "scores"}
        failed[1]["terminal_status"] = "itt_failure"
        failed[1]["artifact_presence"] = 0
        failed[1]["machine_metrics"] = {**failed[1]["machine_metrics"], "missing_canonical": 1.0}
        self.assertFalse(analyze_confirmatory(failed, 110)["machine_gate"])

    def test_real_stdio_subprocess_fixture(self):
        with tempfile.TemporaryDirectory() as directory:
            binary = Path(directory) / "fixture"
            binary.write_text("#!/usr/bin/env python3\nimport json,sys\nfor line in sys.stdin:\n print(json.dumps({'echo':json.loads(line)}), flush=True)\n", encoding="utf-8")
            binary.chmod(0o755)
            messages = [{"jsonrpc": "2.0", "id": 1, "method": "tools/list"}]
            self.assertEqual(run_stdio(binary, [], messages, dict(os.environ))[0]["echo"], messages[0])

    def test_fault_subprocess_replaces_hostile_xdg_environment(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            case_root = root / "fault-case"
            inherited = dict(os.environ)
            inherited.update({"XDG_CACHE_HOME": "/hostile/cache", "XDG_CONFIG_HOME": "/hostile/config"})
            environment = fault_case_environment(case_root, inherited)
            case_root.mkdir()
            materialize_isolation_environment(environment)
            binary = root / "environment-fixture"
            binary.write_text(
                "#!/usr/bin/env python3\nimport json,os,sys\n"
                "for line in sys.stdin:\n print(json.dumps({'cache':os.environ.get('XDG_CACHE_HOME'),'config':os.environ.get('XDG_CONFIG_HOME')}),flush=True)\n",
                encoding="utf-8",
            )
            binary.chmod(0o755)
            observed = run_stdio(binary, [], [{"jsonrpc": "2.0", "id": 1, "method": "tools/list"}], environment)[0]
            for key in ("cache", "config"):
                self.assertTrue(Path(observed[key]).resolve().is_relative_to(case_root.resolve()), observed)
            self.assertNotIn("/hostile/", json.dumps(observed))

    def test_fault_matrix_seeds_stateful_cases_only_through_public_product_commands(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            binary = root / "plasma-fixture"
            binary.write_text(
                "#!/usr/bin/env python3\nimport json,sys\n"
                "if sys.argv[1]=='missions': print(json.dumps({'mission_id':'mis_fault'})); raise SystemExit(0)\n"
                "if '--fail' in sys.argv: print('conditional-failure',file=sys.stderr); raise SystemExit(2)\n"
                "for line in sys.stdin: print(json.dumps({'marker':'observed','request':json.loads(line)}),flush=True)\n",
                encoding="utf-8",
            )
            binary.chmod(0o755)
            cases = {}
            for name in FAULT_CASES:
                stateful = name in STATEFUL_CASES
                arguments = ["-db", "{case_root}/state.db"] if stateful else []
                expected = ["observed"]
                if name == "conditional_store_unavailable":
                    arguments.append("--fail")
                    expected = ["conditional-failure"]
                cases[name] = {
                    "binding_args": arguments, "messages": [{"jsonrpc": "2.0", "id": 1, "method": "tools/list"}],
                    "report_mode": "planned", "agent_executor": "codex",
                    "expected_fragments": expected, "forbidden_fragments": ["secret"],
                    **({"seed_commands": [["{binary}", "missions", "create", "-db", "{case_root}/state.db"]]} if stateful else {}),
                }
            with mock.patch("report_plan_mcp.fault_seed.seed_web_pending", return_value="evt_pending"):
                result = run_fault_matrix(binary, cases, {}, root / "faults")
            self.assertEqual(set(result), set(FAULT_CASES))
            for name in STATEFUL_CASES:
                evidence = result[name]["seed_evidence"]
                self.assertEqual(evidence[0], {"boundary": "missions", "returncode": 0})
                self.assertEqual(evidence[1]["boundary"], "web_report_start")

    def test_real_judge_subprocess_fixture(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            binary = root / "judge"
            dimensions = PLAN_DIMENSIONS + FINAL_DIMENSIONS
            binary.write_text("#!/usr/bin/env python3\nimport json,sys\njson.load(sys.stdin)\ns={name:3 for name in " + repr(dimensions) + "}\nprint(json.dumps({'A':s,'B':s}))\n", encoding="utf-8")
            binary.chmod(0o755)
            packet = root / "packet.json"
            packet.write_text(json.dumps({"packet_id": "packet", "A": "one", "B": "two"}), encoding="utf-8")
            result = score_packets([str(binary)], [packet], root / "scores", dict(os.environ))
            self.assertEqual(result[0]["technical_calls"], 2)
            self.assertEqual(set(result[0]["scores"]), {"A", "B"})
            self.assertEqual(set(result[0]["scores"]["A"]), set(dimensions))

    def test_pre_run_replacement_stops_after_third_and_itt_never_retries(self):
        fixture = Fixture("topic", "title", "objective", Path("source"), "hash", "license", "url")
        attempts = [RunManifest(
            experiment="experiment", topic="topic", replicate=1, arm="candidate", mode="planned", executor="codex",
            commit="commit", binary="binary", binary_hash="hash", model="model", effort="high", source_policy="sources",
            source_bundle="source", source_hash="hash", budgets={"tokens": 1, "seconds": 1}, selected_session_policy="same_session",
            database=f"root{index}/state/db", artifact_root=f"root{index}/artifacts", workdir=f"root{index}/work",
            port=6200 + index * 2, connector_port=6201 + index * 2, connector_url=f"http://127.0.0.1:{6201 + index * 2}",
            namespace=f"n{index}", child_environment={},
        ) for index in range(3)]
        with mock.patch.object(entry, "execute_product_run", side_effect=[ProductRunError("x", kind="port"), ProductRunError("x", kind="port"), ProductRunError("x", kind="port")]):
            terminal, _, history = entry._execute_attempts(attempts, fixture, {})
        self.assertEqual(len(history), 3)
        self.assertEqual(terminal.terminal_status, "pre_run_failure")
        with mock.patch.object(entry, "execute_product_run", side_effect=ProductRunError("x", started=True)) as execute:
            terminal, _, history = entry._execute_attempts(attempts, fixture, {})
        self.assertEqual(len(history), 1)
        self.assertEqual(execute.call_count, 1)
        self.assertEqual(terminal.terminal_status, "itt_failure")

    def test_focused_specs_allocate_ports_lazily_and_release_them(self):
        fixture = Fixture("topic", "title", "objective", Path("source"), "hash", "license", "url")
        spec = RunSpec("topic", 1, "candidate", "planned", "codex", "commit", Path("binary"), "model", "high", "sources", 1, 1, "same_session", Path("source"), "hash", "attempt")
        manifest = RunManifest(
            experiment="experiment", topic="topic", replicate=1, arm="candidate", mode="planned", executor="codex",
            commit="commit", binary="binary", binary_hash="hash", model="model", effort="high", source_policy="sources",
            source_bundle="source", source_hash="hash", budgets={"tokens": 1, "seconds": 1}, selected_session_policy="same_session",
            database="root/state/db", artifact_root="root/artifacts", workdir="root/work", port=6200, connector_port=6201,
            connector_url="http://127.0.0.1:6201", namespace="n", child_environment={},
        )
        used_ports = {6200, 6201}
        failure = ProductRunError("x", started=True)
        failure.manifest = manifest
        with mock.patch.object(entry, "build_manifest", return_value=manifest), mock.patch.object(
            entry, "execute_product_run", side_effect=failure
        ) as execute:
            terminal, _, history = entry._execute_specs([spec, spec, spec], fixture, {}, used_ports, set(), entry.Lock())
        self.assertEqual(execute.call_count, 1)
        self.assertEqual(history[0]["classification"], "itt")
        self.assertEqual(terminal.terminal_status, "itt_failure")
        self.assertEqual(used_ports, set())

    def test_focused_specs_record_completed_audit_failure_without_retry(self):
        fixture = Fixture("topic", "title", "objective", Path("source"), "hash", "license", "url")
        spec = RunSpec("topic", 1, "candidate", "planned", "codex", "commit", Path("binary"), "model", "high", "sources", 1, 1, "same_session", Path("source"), "hash", "attempt")
        manifest = RunManifest(
            experiment="experiment", topic="topic", replicate=1, arm="candidate", mode="planned", executor="codex",
            commit="commit", binary="binary", binary_hash="hash", model="model", effort="high", source_policy="sources",
            source_bundle="source", source_hash="hash", budgets={"tokens": 1, "seconds": 1}, selected_session_policy="same_session",
            database="root/state/db", artifact_root="root/artifacts", workdir="root/work", port=6200, connector_port=6201,
            connector_url="http://127.0.0.1:6201", namespace="n", child_environment={}, result_hash="result", terminal_status="completed",
        )
        used_ports = {6200, 6201}
        ledger = {"events": []}
        with mock.patch.object(entry, "build_manifest", return_value=manifest), mock.patch.object(entry, "execute_product_run", return_value=manifest), mock.patch.object(
            entry.json, "loads", return_value=ledger
        ), mock.patch.object(entry.Path, "read_text", return_value=json.dumps(ledger)), mock.patch.object(
            entry, "collect_hard_metrics", side_effect=ValueError("source-read trace is missing")
        ) as audit:
            terminal, metrics, history = entry._execute_specs([spec, spec], fixture, {}, used_ports, set(), entry.Lock())
        self.assertEqual(audit.call_count, 1)
        self.assertEqual(terminal.terminal_status, "completed")
        self.assertEqual(history[0]["classification"], "completed_audit_failure")
        self.assertEqual(metrics["source_read_violation"], 1.0)
        self.assertEqual(used_ports, set())


if __name__ == "__main__":
    unittest.main()
