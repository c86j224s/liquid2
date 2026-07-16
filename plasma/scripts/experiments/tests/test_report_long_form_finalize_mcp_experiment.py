from __future__ import annotations

import importlib.util
import json
from pathlib import Path
import sys
import tempfile
import unittest
from unittest import mock

EXPERIMENTS = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(EXPERIMENTS))
from report_plan_mcp.provenance import _validate_phase_matrix
import report_plan_mcp.product_path as product_path

SPEC = importlib.util.spec_from_file_location(
    "report_long_form_finalize_mcp_experiment",
    EXPERIMENTS / "report_long_form_finalize_mcp_experiment.py",
)
experiment = importlib.util.module_from_spec(SPEC)
assert SPEC.loader
SPEC.loader.exec_module(experiment)


def write_json(path: Path, value: object) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value), encoding="utf-8")


class LongFormFinalizeExperimentTests(unittest.TestCase):
    def source_experiment(self, root: Path, topics: int = 12) -> None:
        rows, entries = [], []
        for index in range(topics):
            topic = f"topic-{index}"
            source = root / "fixtures" / f"{topic}.txt"
            source.parent.mkdir(parents=True, exist_ok=True)
            source.write_text(topic, encoding="utf-8")
            rows.append({
                "topic": topic, "title": topic, "objective": topic,
                "source_bundle": str(source), "source_sha256": experiment._sha256(source),
                "license": "CC BY-SA 4.0", "license_url": "https://example.com/license",
                "retrieved_at": "2026-07-14T00:00:00Z",
            })
            entries.append({
                "topic": topic, "mode": "long_form",
                "arms": ["baseline", "candidate"] if index < 6 else ["candidate", "baseline"],
            })
            entries.append({"topic": topic, "mode": "planned", "arms": ["baseline", "candidate"]})
        write_json(root / "fixtures.lock.json", {"fixtures": rows})
        write_json(root / "control/focused-execution-schedule.json", {"seed": 110, "entries": entries})

    def test_preflight_selects_exact_codex_long_form_matrix(self):
        with tempfile.TemporaryDirectory() as directory:
            source = Path(directory)
            self.source_experiment(source)
            with mock.patch.object(experiment, "source_root", return_value=source), mock.patch.object(
                experiment, "archive_root", return_value=source.parent / "archive"
            ):
                result = experiment.preflight("codex-model")
        self.assertEqual(result["quality_runs"], 24)
        self.assertEqual(len(result["topics"]), 12)
        self.assertEqual(result["executor"], "codex")
        self.assertEqual(result["mode"], "long_form")

    def test_provenance_adapter_accepts_only_one_long_form_replicate(self):
        rows = [
            {"topic": f"topic-{index}", "replicate": 1, "mode": "long_form", "arm": arm}
            for index in range(12) for arm in ("baseline", "candidate")
        ]
        _validate_phase_matrix(
            "quality", rows, ("long_form",), {"quality": 12}, {"quality": (1,)},
        )
        with self.assertRaisesRegex(ValueError, "incomplete"):
            _validate_phase_matrix(
                "quality", rows[:-1], ("long_form",), {"quality": 12}, {"quality": (1,)},
            )

    def test_schedule_rejects_duplicate_missing_and_unbalanced_topics(self):
        with tempfile.TemporaryDirectory() as directory:
            source = Path(directory)
            self.source_experiment(source, topics=11)
            with mock.patch.object(experiment, "source_root", return_value=source):
                with self.assertRaisesRegex(ValueError, "exactly 12"):
                    experiment._source_rows()
            self.source_experiment(source, topics=12)
            schedule = json.loads((source / "control/focused-execution-schedule.json").read_text())
            long_rows = [row for row in schedule["entries"] if row["mode"] == "long_form"]
            long_rows[-1]["topic"] = long_rows[0]["topic"]
            write_json(source / "control/focused-execution-schedule.json", {"entries": long_rows})
            with mock.patch.object(experiment, "source_root", return_value=source):
                with self.assertRaisesRegex(ValueError, "unique"):
                    experiment._source_rows()

    def test_config_rejects_claude_and_prepare_gate_rejects_wrong_commit(self):
        with tempfile.TemporaryDirectory() as directory:
            archive = Path(directory)
            write_json(archive / "config.json", {
                "experiment": experiment.EXPERIMENT_ID,
                "candidate_commit": experiment.CANDIDATE_COMMIT,
                "models": {"codex": "c", "claude": "x"}, "efforts": {"codex": "high"},
            })
            with mock.patch.object(experiment, "archive_root", return_value=archive):
                with self.assertRaisesRegex(ValueError, "exactly one Codex"):
                    experiment._config()
            write_json(archive / "control/prepare-gate.json", {
                "passed": True, "baseline_commit": "wrong", "candidate_commit": experiment.CANDIDATE_COMMIT,
            })
            with mock.patch.object(experiment, "archive_root", return_value=archive):
                with self.assertRaisesRegex(ValueError, "prepare gate"):
                    experiment._prepare_gate()

    def test_finalizer_path_requires_candidate_trace_canonical_and_sentinel(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "state").mkdir()
            (root / "logs").mkdir()
            database = root / "state/plasma.db"
            artifact = {
                "EventID": "evt_final", "EventType": "report.artifact.created",
                "Payload": {"artifact_id": "art_final", "tool_session_id": "tool_1"},
            }
            events = [
                {"EventID": "evt_plan", "EventType": "report.plan.submitted", "Payload": {
                    "plan_hash": "hash", "tool_session_id": "tool_plan",
                }},
                {"EventID": "evt_created", "EventType": "report.plan.created", "Payload": {
                    "plan_submission": {"submission_event_id": "evt_plan", "plan_hash": "hash", "tool_session_id": "tool_plan"},
                }}, artifact,
                {"EventType": "mcp.tool.called", "Payload": {
                    "tool_name": "plasma.report.long_form.finalize", "tool_session_id": "tool_1", "success": True,
                    "result": {"created_event_ids": ["evt_final"]},
                }},
            ]
            write_json(root / "ledger.events.json", {"events": events})
            (root / "logs/serve.log").write_text(
                'report_long_form_final_completed artifact_id="art_final" event_id="evt_final" '
                "attempt_count=1 canonical=true sentinel_ok=true\n", encoding="utf-8",
            )
            result = experiment.assert_finalizer_path({"database": str(database), "arm": "candidate"})
            self.assertEqual(result["finalizer_calls"], 1)
            self.assertTrue(result["sentinel"])
            events[-1]["Payload"]["result"]["created_event_ids"] = []
            write_json(root / "other.json", {})
            (root / "ledger.events.json").write_text(json.dumps({"events": events}), encoding="utf-8")
            with self.assertRaisesRegex(ValueError, "canonical artifact"):
                experiment.assert_finalizer_path({"database": str(database), "arm": "candidate"})

    def test_baseline_rejects_finalizer_call(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "state").mkdir()
            (root / "logs").mkdir()
            events = [
                {"EventID": "evt_plan", "EventType": "report.plan.submitted", "Payload": {"plan_hash": "h", "tool_session_id": "t"}},
                {"EventType": "report.plan.created", "Payload": {"plan_submission": {"submission_event_id": "evt_plan", "plan_hash": "h", "tool_session_id": "t"}}},
                {"EventType": "report.artifact.created", "Payload": {"artifact_id": "a"}},
                {"EventType": "mcp.tool.called", "Payload": {"tool_name": "plasma.report.long_form.finalize"}},
            ]
            write_json(root / "ledger.events.json", {"events": events})
            with self.assertRaisesRegex(ValueError, "baseline"):
                experiment.assert_finalizer_path({"database": str(root / "state/plasma.db"), "arm": "baseline"})

    def test_product_path_can_wait_for_safe_completion_marker(self):
        process = mock.Mock()
        process.poll.return_value = None
        events = {"events": [{"EventType": "report.artifact.created"}]}
        with tempfile.TemporaryDirectory() as directory:
            log = Path(directory) / "serve.log"
            log.write_text("", encoding="utf-8")
            with mock.patch.object(product_path, "_http_json", return_value=events), mock.patch.object(
                product_path.time, "monotonic", side_effect=(0.0, 2.0)
            ), self.assertRaisesRegex(Exception, "timed out"):
                product_path._poll_terminal("http://127.0.0.1:1", "mission", process, 1, None, log, "completed")
            log.write_text("completed", encoding="utf-8")
            with mock.patch.object(product_path, "_http_json", return_value=events), mock.patch.object(
                product_path.time, "monotonic", side_effect=(0.0, 0.1)
            ):
                returned, status = product_path._poll_terminal("http://127.0.0.1:1", "mission", process, 1, None, log, "completed")
            self.assertEqual(returned, events["events"])
            self.assertEqual(status, "completed")


if __name__ == "__main__":
    unittest.main()
