from __future__ import annotations

from contextlib import contextmanager
import json
from pathlib import Path
from types import SimpleNamespace
import sys
import tempfile
import unittest
from unittest import mock


EXPERIMENTS = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(EXPERIMENTS))

from report_natural_voice_correction.archive import read_json
from report_natural_voice_examples import config as base_config
from report_natural_voice_examples.archive import ExperimentArchive
from report_natural_voice_examples.blind import make_blind_packets, record_verdicts
from report_natural_voice_examples.prompts import EXAMPLE_CATEGORIES, PromptError, load_prompt
from report_natural_voice_examples.review import make_semantic_audit_pack, record_semantic_audit
from report_natural_voice_examples.runner import authorize_full, run_full, run_pilots
from report_natural_voice_examples.records import run_document
from report_natural_voice_examples_replication import config
from report_natural_voice_examples_replication.context import activated
from report_natural_voice_examples_replication.hash_contract import (
    HashContractError,
    lock_protocol_amendment,
)
from report_natural_voice_examples_replication.prompts import freeze_protocol
from report_natural_voice_examples_replication.recovery import resume_pilot_attempt
from report_natural_voice_examples_replication.summary import drift_counts_by_arm, export_summary


CONTROL = "# Control\n\nKeep the document intact.\n"


def example_prompt() -> str:
    sections = []
    for category in EXAMPLE_CATEGORIES:
        sections.append(
            f"## {category}\nBefore:\n기존 문장입니다.\nAfter:\n고친 문장입니다.\n"
            "Preserve:\n그대로 둘 문장입니다."
        )
    return CONTROL.rstrip() + "\n\n# Target voice examples\n\n" + "\n\n".join(sections) + "\n"


def source_text(index: int) -> str:
    return (
        f"# 문서 {index}\n\n"
        "이 문장은 실행에 대한 검토를 진행합니다.\n\n"
        "따라서 이 문장은 결과를 설명합니다.\n\n"
        "마지막 문장은 판단을 정리합니다.\n"
    )


def fake_codex(command: list[str], **kwargs: object) -> SimpleNamespace:
    rendered = str(kwargs["input"])
    document_sha = next(
        line.split(": ", 1)[1]
        for line in rendered.splitlines()
        if line.startswith("document_sha256: ")
    )
    numbered = [line for line in rendered.splitlines() if "\t" in line]
    evidence = [int(numbered[index].split("\t", 1)[0]) for index in (0, 1, 2)]
    payload = {
        "document_sha256": document_sha,
        "diagnoses": [
            {"category": "translationese_nominalization", "evidence_line_numbers": [evidence[0]]},
            {"category": "formulaic_connection", "evidence_line_numbers": [evidence[1]]},
            {"category": "uniform_cadence", "evidence_line_numbers": [evidence[2]]},
        ],
        "edits": [],
    }
    output = Path(command[command.index("-o") + 1])
    output.write_text(json.dumps(payload, ensure_ascii=False), encoding="utf-8")
    return SimpleNamespace(returncode=0, stdout="", stderr="")


def fake_bad_hash(command: list[str], **kwargs: object) -> SimpleNamespace:
    rendered = str(kwargs["input"])
    document_sha = next(
        line.split(": ", 1)[1]
        for line in rendered.splitlines()
        if line.startswith("document_sha256: ")
    )
    numbered = [line for line in rendered.splitlines() if "\t" in line]
    line_number, _, original_line = numbered[2].split("\t", 2)
    payload = {
        "document_sha256": document_sha,
        "diagnoses": [
            {"category": "translationese_nominalization", "evidence_line_numbers": [1]},
            {"category": "formulaic_connection", "evidence_line_numbers": [2]},
            {"category": "uniform_cadence", "evidence_line_numbers": [3]},
        ],
        "edits": [{
            "line_number": int(line_number),
            "original_line_sha256": "0" * 64,
            "original_line": original_line,
            "replacement_line": "이 문장은 실행을 검토합니다.",
            "category": "translationese_nominalization",
            "safety_rationale": "뜻을 유지하면서 명사화를 줄인다.",
        }],
    }
    output = Path(command[command.index("-o") + 1])
    output.write_text(json.dumps(payload, ensure_ascii=False), encoding="utf-8")
    return SimpleNamespace(returncode=0, stdout="", stderr="")


class Harness:
    def __init__(self, base: Path) -> None:
        self.home = base / "home"
        for index, path in enumerate(
            [*config.DEVELOPMENT_SOURCES.values(), *config.EVALUATION_SOURCES.values()],
            1,
        ):
            source = self.home / path
            source.parent.mkdir(parents=True, exist_ok=True)
            source.write_text(source_text(index), encoding="utf-8")
        control = self.home / config.CONTROL_PROMPT_SOURCE
        control.parent.mkdir(parents=True, exist_ok=True)
        control.write_text(CONTROL, encoding="utf-8")
        prompt = self.home / config.EXAMPLES_PROMPT_SOURCE
        prompt.parent.mkdir(parents=True, exist_ok=True)
        prompt.write_text(example_prompt(), encoding="utf-8")

    @contextmanager
    def active_archive(self):
        with (
            mock.patch.object(config, "CONTROL_PROMPT_SHA256", base_config.sha256_text(CONTROL)),
            mock.patch.object(config, "EXAMPLES_PROMPT_SHA256", base_config.sha256_text(example_prompt())),
            activated(),
        ):
            yield ExperimentArchive(self.home / config.ARCHIVE_SUFFIX, home=self.home)


class NaturalVoiceExamplesReplicationTest(unittest.TestCase):
    def setUp(self) -> None:
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.harness = Harness(Path(self.temp.name))

    def test_activation_restores_experiment_58_config(self) -> None:
        original_id = base_config.EXPERIMENT_ID
        with self.harness.active_archive():
            self.assertEqual(base_config.EXPERIMENT_ID, config.EXPERIMENT_ID)
        self.assertEqual(base_config.EXPERIMENT_ID, original_id)

    def test_freeze_rejects_changed_experiment_58_prompt(self) -> None:
        with self.harness.active_archive() as archive:
            archive.prepare()
            prompt = self.harness.home / config.EXAMPLES_PROMPT_SOURCE
            prompt.write_text(prompt.read_text(encoding="utf-8") + "changed\n", encoding="utf-8")
            with self.assertRaisesRegex(PromptError, "changed"):
                freeze_protocol(archive)

    def test_hash_amendment_preserves_raw_attempt_and_changes_only_hash(self) -> None:
        with self.harness.active_archive() as archive:
            archive.prepare()
            freeze_protocol(archive)
            file_id = archive.development_file_ids()[0]
            prompt, prompt_sha = load_prompt(archive, "control")
            with self.assertRaisesRegex(HashContractError, "amendment"):
                run_document(
                    archive, "development", file_id, "control", prompt, prompt_sha,
                    ("pilot",), subprocess_run=fake_bad_hash,
                )
            amendment = lock_protocol_amendment(archive)
            self.assertFalse(amendment["model_calls_retried"])
            record = resume_pilot_attempt(archive, file_id, "control")
            self.assertIn("hash_contract_normalization_path", record)

            run_dir = archive.root / "runs" / "pilot" / file_id / "control"
            raw = read_json(run_dir / "raw-output.json")
            normalized = read_json(run_dir / "hash-normalized-output.json")
            self.assertEqual(raw["edits"][0]["replacement_line"], normalized["edits"][0]["replacement_line"])
            self.assertEqual(raw["edits"][0]["original_line"], normalized["edits"][0]["original_line"])
            self.assertNotEqual(
                raw["edits"][0]["original_line_sha256"],
                normalized["edits"][0]["original_line_sha256"],
            )

    @mock.patch("report_natural_voice_examples.records.subprocess.run", side_effect=fake_codex)
    def test_pipeline_separates_replication_safety_and_readiness(self, _: mock.Mock) -> None:
        with self.harness.active_archive() as archive:
            archive.prepare()
            freeze_protocol(archive)
            self.assertEqual(len(run_pilots(archive)), 4)
            authorize_full(archive)
            self.assertEqual(len(run_full(archive, workers=3)), 16)
            make_blind_packets(archive, seed=60)

            mapping = read_json(archive.root / "blind" / "private-mapping.lock.json")
            verdict_path = archive.root / "blind" / "verdicts.input.json"
            verdicts = []
            for index, row in enumerate(mapping["mappings"]):
                examples_win = index < 5
                verdicts.append({
                    "packet_id": row["packet_id"],
                    "choice": row["examples_slot"] if examples_win else row["control_slot"],
                    "magnitude": "clear" if examples_win else "slight",
                    "rationale": "문장 호흡과 단어 선택을 전체 문서 기준으로 비교했다.",
                })
            verdict_path.write_text(
                json.dumps({"verdicts": verdicts}, ensure_ascii=False), encoding="utf-8"
            )
            record_verdicts(archive, verdict_path)

            make_semantic_audit_pack(archive)
            pack = read_json(archive.root / "analysis" / "semantic-audit-pack.json")
            audit_path = archive.root / "analysis" / "semantic-audit.input.json"
            audit_path.write_text(json.dumps({"audits": [
                {
                    "file_id": cell["file_id"],
                    "arm": cell["arm"],
                    "reviewed_line_numbers": [],
                    "semantic_drift_lines": [],
                    "claim_scope_drift_lines": [],
                    "citation_drift_lines": [],
                    "notes": "수락된 편집이 없어 보존 이탈도 없다.",
                }
                for cell in pack["cells"]
            ]}, ensure_ascii=False), encoding="utf-8")
            record_semantic_audit(archive, audit_path)

            summary = export_summary(archive)
            self.assertEqual(summary["reading_efficacy"]["classification"], "directional_support")
            self.assertEqual(summary["reading_efficacy"]["signed_magnitude_score"], 7)
            self.assertEqual(summary["semantic_safety"]["examples"]["status"], "no_drift_observed")
            self.assertEqual(summary["product_readiness"]["status"], "not_evaluated")
            self.assertNotIn("experiment_passed", summary)
            self.assertNotIn("screening_passed", summary)

    def test_drift_counts_remain_arm_specific(self) -> None:
        counts = drift_counts_by_arm({"audits": [
            {
                "arm": "control",
                "semantic_drift_lines": [3, 8],
                "claim_scope_drift_lines": [3],
                "citation_drift_lines": [],
            },
            {
                "arm": "examples",
                "semantic_drift_lines": [5],
                "claim_scope_drift_lines": [],
                "citation_drift_lines": [5],
            },
        ]})
        self.assertEqual(counts["control"]["semantic_drift"], 2)
        self.assertEqual(counts["examples"]["semantic_drift"], 1)
        self.assertEqual(counts["examples"]["citation_drift"], 1)


if __name__ == "__main__":
    unittest.main()
