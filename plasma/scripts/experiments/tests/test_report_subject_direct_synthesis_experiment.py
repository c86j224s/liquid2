from __future__ import annotations

import importlib.util
import sqlite3
from pathlib import Path
import sys
import tempfile
import unittest


EXPERIMENTS = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(EXPERIMENTS))
SPEC = importlib.util.spec_from_file_location(
    "report_subject_direct_synthesis_experiment",
    EXPERIMENTS / "report_subject_direct_synthesis_experiment.py",
)
experiment = importlib.util.module_from_spec(SPEC)
assert SPEC.loader
SPEC.loader.exec_module(experiment)
METRICS_SPEC = importlib.util.spec_from_file_location(
    "report_subject_direct_synthesis_metrics",
    EXPERIMENTS / "report_subject_direct_synthesis_metrics.py",
)
metrics = importlib.util.module_from_spec(METRICS_SPEC)
assert METRICS_SPEC.loader
METRICS_SPEC.loader.exec_module(metrics)


class SubjectDirectSynthesisExperimentTests(unittest.TestCase):
    def test_arms_are_exact_issue_189_matrix(self):
        self.assertEqual(
            experiment.experiment.ARMS,
            ("current_default", "rich_control", "subject_direct_candidate"),
        )
        self.assertEqual(
            experiment.experiment.PROFILE_BY_ARM,
            {
                "current_default": "part-connective-economy-voice",
                "rich_control": "section-brief-cluster-memory-narrative-contract",
                "subject_direct_candidate": "part-connective-subject-direct-synthesis-voice",
            },
        )

    def test_stage_metrics_collect_advisory_locator_citations_and_size(self):
        with tempfile.TemporaryDirectory() as directory:
            database = Path(directory) / "plasma.db"
            with sqlite3.connect(database) as connection:
                connection.execute("create table plasma_raw_artifacts(artifact_id text, filename text, content_blob blob)")
                connection.execute("create table plasma_ledger_events(event_type text, payload_json text)")
                connection.executemany(
                    "insert into plasma_raw_artifacts(artifact_id, filename, content_blob) values (?, ?, ?)",
                    [
                        ("art_section", "topic-part-1-section-1.md", b"# S\n\nThe policy expands coverage [Doc](https://example.com)."),
                        ("art_part", "topic-part-1.md", "자료는 정책이 확대된다고 설명한다. (Agency, 2026) [^std-future]".encode()),
                        ("art_final", "topic.md", "The source says the policy expands coverage.\n\n## Caveat".encode()),
                        ("art_other", "other.md", b"Unrelated final-looking artifact [Other](https://example.com)."),
                    ],
                )
                connection.executemany(
                    "insert into plasma_ledger_events(event_type, payload_json) values (?, ?)",
                    [
                        ("report.section.created", '{"artifact_id":"art_section"}'),
                        ("report.part.created", '{"artifact_id":"art_part"}'),
                        ("report.artifact.created", '{"artifact_id":"art_final"}'),
                    ],
                )
            result = metrics.collect_stage_metrics(database, experiment.experiment.ratio)
        self.assertEqual(result["stages"]["section"]["artifacts"], 1)
        self.assertEqual(result["stages"]["part"]["citation_count"], 2)
        self.assertEqual(result["stages"]["final"]["source_narrator_candidates"], 1)
        self.assertEqual(result["stages"]["final"]["heading_count"], 1)
        self.assertEqual(result["stages"]["final"]["artifacts"], 1)
        self.assertEqual(result["stages"]["unmapped"]["artifacts"], 1)
        self.assertEqual(result["stages"]["unmapped"]["citation_count"], 1)
        self.assertIsNotNone(result["section_to_final_character_ratio"])

    def test_citation_metrics_count_only_evidence_locators(self):
        included = (
            "External evidence [Doc](https://example.com/report). "
            "The archive says so (Agency, 2026). "
            "Prior work agrees (Smith et al. 2024). "
            "The standard follows this pattern (WHO 2023). "
            "Korean guidance matches it (세계보건기구, 2024). "
            "Rust async has a standardization track.[^std-future]\n\n"
            "[^std-future]: Standard future note."
        )
        excluded = (
            "![Chart](https://example.com/chart.png) "
            "[Section](#local-anchor) "
            "[Relative](../notes.md) "
            "[Internal](/missions/mis_1/artifacts/art_1) "
            "(not a citation) "
            "(https://example.com/raw)"
        )
        self.assertEqual(metrics.count_citations(included), 7)
        self.assertEqual(metrics.count_citations(excluded), 0)

    def test_parenthetical_citations_exclude_plain_dates_and_periods(self):
        included = (
            "(Agency, 2026) (Smith et al. 2024) (WHO 2023) (세계보건기구, 2024) "
            "(World Health Organization, 2023) "
            "(Centers for Disease Control and Prevention, 2024) "
            "(National Academies 2020)"
        )
        excluded = (
            "(in 2026) (Q4 2026) (FY 2026) (as of 2026) (May 2026) (May 2026 update) "
            "(Spring 2026) (Summer 2026) (Update 2026) (Release 2026)"
        )
        self.assertEqual(metrics.count_parenthetical_citations(included), 7)
        self.assertEqual(metrics.count_parenthetical_citations(excluded), 0)

    def test_source_narrator_detector_is_advisory_count_only(self):
        text = "The source says the mechanism changes. Tokio 문서는 런타임을 설명한다. official documentation explains the API."
        self.assertEqual(metrics.count_source_narrator_candidates(text), 3)

    def test_packet_matrix_records_missing_failed_and_non_completed(self):
        with tempfile.TemporaryDirectory() as directory:
            archive = Path(directory)
            self.write_manifest(archive, "public-health-guidance-b", "current_default", "completed")
            self.write_manifest(archive, "public-health-guidance-b", "rich_control", "failed")
            self.write_manifest(archive, "public-health-guidance-b", "subject_direct_candidate", "started")
            with self.assertRaisesRegex(RuntimeError, "refusing packet generation"):
                experiment.assert_packet_matrix_completed(archive, 2)
            status = self.read_json(archive / "judging/packet-matrix-status.json")
        self.assertEqual(len(status["missing"]), 3)
        self.assertEqual(len(status["failed"]), 1)
        self.assertEqual(len(status["non_completed"]), 1)

    def test_packet_matrix_accepts_only_complete_expected_matrix(self):
        with tempfile.TemporaryDirectory() as directory:
            archive = Path(directory)
            for topic in experiment.SELECTED_TOPICS[:1]:
                for arm in experiment.experiment.ARMS:
                    self.write_manifest(archive, topic, arm, "completed")
            status = experiment.assert_packet_matrix_completed(archive, 1)
        self.assertEqual(status["expected"], 3)
        self.assertEqual(len(status["completed"]), 3)
        self.assertFalse(status["missing"] or status["failed"] or status["non_completed"])

    @staticmethod
    def write_manifest(archive: Path, topic: str, arm: str, status: str) -> None:
        path = archive / "runs" / f"{topic}-{arm}" / "manifest.terminal.json"
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(f'{{"topic":"{topic}","arm":"{arm}","status":"{status}"}}', encoding="utf-8")

    @staticmethod
    def read_json(path: Path) -> dict[str, object]:
        import json

        return json.loads(path.read_text(encoding="utf-8"))


if __name__ == "__main__":
    unittest.main()
