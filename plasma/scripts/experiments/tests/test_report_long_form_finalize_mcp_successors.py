from __future__ import annotations

import importlib.util
import json
import os
from pathlib import Path
import sys
import tempfile
from threading import Lock
import unittest
from unittest import mock


EXPERIMENTS = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(EXPERIMENTS))
from report_plan_mcp.judging import PLAN_DIMENSIONS
from report_plan_mcp.models import Fixture
from report_plan_mcp.product_path import ProductRunError, assert_public_product_path, product_commands

SPEC = importlib.util.spec_from_file_location(
    "report_long_form_finalize_mcp_successors",
    EXPERIMENTS / "report_long_form_finalize_mcp_successors.py",
)
successors = importlib.util.module_from_spec(SPEC)
assert SPEC.loader
sys.modules[SPEC.name] = successors
SPEC.loader.exec_module(successors)


def write_json(path: Path, value: object) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value), encoding="utf-8")


def fixture(root: Path, topic: str) -> Fixture:
    source = root / f"{topic}.txt"
    source.parent.mkdir(parents=True, exist_ok=True)
    source.write_text(topic, encoding="utf-8")
    return Fixture(topic, topic, topic, source, successors._sha256(source), "CC BY 4.0", "https://example.com/license", "2026-07-14T00:00:00Z")


class SuccessorControllerTests(unittest.TestCase):
    def test_action_graph_is_closed(self):
        expected = {
            "prepare-analysis", "assemble-itt", "analyze-quality",
            "prepare-operational", "smoke", "reliability", "audit",
        }
        with mock.patch.object(sys, "argv", ["controller", "--action", "prepare-analysis"]):
            self.assertEqual(successors.parse_args().action, "prepare-analysis")
        for action in expected:
            with mock.patch.object(sys, "argv", ["controller", "--action", action, *(["--auth-seed", "/tmp/auth"] if action == "prepare-operational" else [])]):
                self.assertEqual(successors.parse_args().action, action)
        with mock.patch.object(sys, "argv", ["controller", "--action", "judge"]), self.assertRaises(SystemExit):
            successors.parse_args()

    def test_action_context_rebinds_and_restores_19_20_19(self):
        with tempfile.TemporaryDirectory() as directory:
            home = Path(directory)
            original = (successors.predecessor.EXPERIMENT_ID, successors.base.EXPERIMENT_ID, successors.safety.EXPERIMENT_ID)
            for target in (successors.ANALYSIS, successors.OPERATIONAL, successors.ANALYSIS):
                with successors.action_context(target, home) as root:
                    root.mkdir(parents=True, exist_ok=True)
                    marker = root / f"{target.action_group}.txt"
                    if not marker.exists():
                        marker.write_text(target.experiment, encoding="utf-8")
                    self.assertEqual(successors.predecessor.EXPERIMENT_ID, target.experiment)
                    self.assertTrue(successors.safety.namespace("t", 1, "candidate", "long_form", "n").startswith(target.experiment))
            self.assertEqual((successors.predecessor.EXPERIMENT_ID, successors.base.EXPERIMENT_ID, successors.safety.EXPERIMENT_ID), original)
            self.assertEqual(len(list(successors.ANALYSIS.root(home).glob("*.txt"))), 1)
            self.assertEqual(len(list(successors.OPERATIONAL.root(home).glob("*.txt"))), 1)

    def test_action_context_rejects_cross_root_write(self):
        with tempfile.TemporaryDirectory() as directory:
            home = Path(directory)
            other = successors.OPERATIONAL.root(home)
            other.mkdir(parents=True)
            (other / "locked").write_text("before", encoding="utf-8")
            with self.assertRaisesRegex(ValueError, "non-target"):
                with successors.action_context(successors.ANALYSIS, home):
                    (other / "changed").write_text("after", encoding="utf-8")

    def test_recursive_lstat_rejects_symlink_and_hardlink(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory).resolve()
            source = root / "source"
            source.write_text("x", encoding="utf-8")
            self.assertTrue(successors._lstat_tree(root)["passed"])
            link = root / "link"
            link.symlink_to(source)
            with self.assertRaisesRegex(ValueError, "symlink"):
                successors._lstat_tree(root)
            link.unlink()
            os.link(source, root / "hard")
            with self.assertRaisesRegex(ValueError, "hard-linked"):
                successors._lstat_tree(root)

    def test_manifest_identity_rejects_wrong_experiment_namespace_and_root(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory).resolve()
            valid = {"experiment": successors.OPERATIONAL_ID, "namespace": f"{successors.OPERATIONAL_ID}-run", "database": str(root / "state/db")}
            successors._assert_manifest(valid, successors.OPERATIONAL, root)
            for field, value in (("experiment", "wrong"), ("namespace", "wrong-run"), ("database", "/outside/db")):
                changed = dict(valid)
                changed[field] = value
                with self.assertRaises(ValueError):
                    successors._assert_manifest(changed, successors.OPERATIONAL, root)

    def test_analysis_input_rejects_wrong_counts(self):
        with tempfile.TemporaryDirectory() as directory:
            home = Path(directory)
            source = successors._source_root(home)
            for name in (
                "protocol.lock.json", "prepare-gate.json", "quality-gate.json", "experiment-stopped.json",
                "packets.lock.json", "judge-gate.json",
            ):
                write_json(source / "control" / name, {})
            write_json(source / "judging/packets-mapping.private.json", [])
            with self.assertRaisesRegex(ValueError, "24 runs"):
                successors._analysis_inputs(home)

    def test_failed_prepare_is_sealed_and_same_id_cannot_be_reused(self):
        with tempfile.TemporaryDirectory() as directory:
            home = Path(directory)
            with mock.patch.object(successors, "_controller_commit", side_effect=ValueError("dirty")):
                with self.assertRaisesRegex(ValueError, "dirty"):
                    successors.prepare_analysis(home)
            marker = successors.ANALYSIS.root(home) / "control/experiment-stopped.json"
            self.assertTrue(json.loads(marker.read_text(encoding="utf-8"))["stopped"])
            with self.assertRaisesRegex(ValueError, "already exists"):
                successors.prepare_analysis(home)

    def test_validate_build_rejects_commit_and_hash_mismatch(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source, binary = root / "source.tar", root / "plasma"
            source.write_text("source", encoding="utf-8")
            binary.write_text("binary", encoding="utf-8")
            build = {
                "arm": "candidate", "commit": "wrong", "source_archive": str(source),
                "source_sha256": successors._sha256(source), "binary": str(binary),
                "binary_sha256": successors._sha256(binary), "version": "v",
            }
            with self.assertRaisesRegex(ValueError, "provenance"):
                successors._validate_build(build, "candidate", successors.CANDIDATE_COMMIT)
            build["commit"] = successors.CANDIDATE_COMMIT
            build["binary_sha256"] = "wrong"
            with self.assertRaisesRegex(ValueError, "hash"):
                successors._validate_build(build, "candidate", successors.CANDIDATE_COMMIT)

    def test_operational_source_rejects_duplicate_or_missing_topics(self):
        with tempfile.TemporaryDirectory() as directory:
            home = Path(directory)
            source = successors._source_root(home)
            fixtures = [fixture(source / "inputs", f"topic-{index}") for index in range(12)]
            smoke = fixture(source / "inputs", "smoke")
            write_json(source / "control/execution-schedule.json", {"entries": [{"topic": item.topic} for item in fixtures[:-1]] + [{"topic": fixtures[0].topic}]})
            with mock.patch.object(successors, "load_and_validate_fixtures", side_effect=(tuple(fixtures), (smoke,))):
                with self.assertRaisesRegex(ValueError, "12 unique"):
                    successors._operational_source(home)

    def test_analysis_uses_existing_itt_low_score_rule_and_exact_dimensions(self):
        records = []
        for topic in ("a", "b"):
            for arm in ("baseline", "candidate"):
                score = {"plan": {name: 4.0 for name in PLAN_DIMENSIONS}, "final": {name: 4.0 for name in successors.FINAL_DIMENSIONS}}
                records.append({
                    "topic": topic, "replicate": 1, "mode": "long_form", "arm": arm,
                    "started": True, "terminal_status": "itt_failure" if topic == "b" and arm == "candidate" else "completed",
                    "artifact_presence": 1, "machine_metrics": {}, "scores": score,
                })
        assembled = successors.assemble_itt(records)
        failed = next(row for row in assembled if row["terminal_status"] == "itt_failure")
        self.assertEqual(set(failed["scores"]["final"]), set(successors.FINAL_DIMENSIONS))
        self.assertEqual(set(failed["scores"]["final"].values()), {1.0})

    def test_operational_matrix_is_exact_smoke_two_then_candidate_twelve(self):
        with tempfile.TemporaryDirectory() as directory:
            home = Path(directory)
            root = successors.OPERATIONAL.root(home)
            root.mkdir(parents=True)
            fixtures = [fixture(root / "fixtures", f"topic-{index}") for index in range(12)]
            smoke = fixture(root / "fixtures", "smoke")
            config = {"experiment": successors.OPERATIONAL_ID, "auth_seeds": {"CODEX_HOME": str(root / "auth")}, "models": {"codex": "model"}}
            write_json(root / "config.json", config)
            write_json(root / "control/protocol.lock.json", {"experiment": successors.OPERATIONAL_ID})
            gate = {
                "experiment": successors.OPERATIONAL_ID, "passed": True,
                "baseline": {"commit": successors.BASELINE_COMMIT, "binary": "/bin/true"},
                "candidate": {"commit": successors.CANDIDATE_COMMIT, "binary": "/bin/true"},
            }
            protocol = {"experiment": successors.OPERATIONAL_ID}

            def completed(spec, item, auth, archive, used, namespaces, lock):
                namespace = f"{successors.OPERATIONAL_ID}-{item.topic}-{spec.arm}"
                return {
                    "experiment": successors.OPERATIONAL_ID, "topic": item.topic, "arm": spec.arm,
                    "mode": "long_form", "executor": "codex", "terminal_status": "completed",
                    "machine_metrics": {name: 0.0 for name in ("missing_canonical", "duplicate_canonical", "session_violation", "source_read_violation", "ref_scope_violation", "recovery_violation", "isolation_violation")} | {"artifact_presence": 1.0, "fallback_count": 0.0, "binding_violation": 0.0},
                    "finalizer_path": {}, "namespace": namespace,
                }

            with mock.patch.object(successors, "_require_prepared", return_value=(gate, protocol)), mock.patch.object(
                successors, "_source_inventories", return_value={}
            ), mock.patch.object(successors.safety, "snapshot_protected_paths", return_value={}), mock.patch.object(
                successors, "load_and_validate_fixtures", side_effect=((smoke,), tuple(fixtures))
            ), mock.patch.object(successors, "_execute_once", side_effect=completed), mock.patch.object(
                successors, "_run_spec", side_effect=lambda archive, cfg, locked, item, arm, nonce: mock.Mock(arm=arm)
            ):
                smoke_gate = successors.execute_operational_phase("smoke", 2, home)
                reliability_gate = successors.execute_operational_phase("reliability", 6, home)
            self.assertEqual(smoke_gate["expected_run_count"], 2)
            self.assertEqual({row["arm"] for row in smoke_gate["runs"]}, {"baseline", "candidate"})
            self.assertEqual(reliability_gate["expected_run_count"], 12)
            self.assertEqual({row["arm"] for row in reliability_gate["runs"]}, {"candidate"})

    def test_product_command_construction_is_real_web_codex_mcp_path_only(self):
        commands = product_commands(Path("/archive/plasma"), Path("/archive/run"), 6200, "http://127.0.0.1:6201", "long_form", "codex", "model", "high")
        assert_public_product_path(commands)
        flat = "\n".join(" ".join(command) for command in commands)
        for required in ("serve", "/reports", "long_form", "codex", "mcp"):
            self.assertIn(required, flat)
        for prohibited in ("planned", "report_h5", "designed", '"claude"'):
            self.assertNotIn(prohibited, flat)

    def test_started_failure_is_one_terminal_attempt_without_replacement(self):
        error = ProductRunError("failed", started=True, kind="runtime")
        manifest = mock.Mock()
        manifest.as_dict.return_value = {
            "experiment": successors.OPERATIONAL_ID, "namespace": f"{successors.OPERATIONAL_ID}-run",
            "database": "/tmp/archive/state/db", "artifact_root": "/tmp/archive/artifacts", "workdir": "/tmp/archive/work",
        }
        manifest.arm, manifest.port, manifest.connector_port = "candidate", 6200, 6201
        manifest.start_boundary, manifest.terminal_status = "started:product_cli_mission_create", "preflight"
        error.manifest = manifest
        terminal = mock.Mock()
        terminal.arm, terminal.port, terminal.connector_port = "candidate", 6200, 6201
        terminal.as_dict.return_value = {"terminal_status": "itt_failure"}
        with mock.patch.object(successors.base, "build_manifest", return_value=manifest), mock.patch.object(
            successors, "_assert_manifest"
        ), mock.patch.object(successors, "execute_product_run", side_effect=error), mock.patch.object(
            successors.base, "_failed_metrics", return_value={}
        ), mock.patch.object(successors, "replace", return_value=terminal):
            row = successors._execute_once(mock.Mock(), mock.Mock(), {}, Path("/tmp/archive"), {6200, 6201}, set(), Lock())
        self.assertEqual(row["terminal_status"], "itt_failure")
        self.assertEqual(len(row["attempts"]), 1)


if __name__ == "__main__":
    unittest.main()
