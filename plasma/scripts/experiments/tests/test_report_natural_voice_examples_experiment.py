from __future__ import annotations

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
from report_natural_voice_examples import config
from report_natural_voice_examples.archive import ArchiveError, ExperimentArchive
from report_natural_voice_examples.blind import (
    BlindError,
    make_blind_packets,
    record_verdicts,
    validate_verdict_lock,
)
from report_natural_voice_examples.prompts import EXAMPLE_CATEGORIES, PromptError, freeze_protocol
from report_natural_voice_examples.records import RecordError, validate_record
from report_natural_voice_examples.review import (
    ReviewError,
    make_semantic_audit_pack,
    record_semantic_audit,
    validate_semantic_audit,
)
from report_natural_voice_examples.runner import authorize_full, run_full, run_pilots
from report_natural_voice_examples.summary import export_summary


CONTROL = "# Control\n\nKeep the document intact.\n"


def example_prompt(control: str = CONTROL) -> str:
    sections = []
    for category in EXAMPLE_CATEGORIES:
        sections.append(
            f"## {category}\nBefore:\n기존 문장입니다.\nAfter:\n고친 문장입니다.\nPreserve:\n그대로 둘 문장입니다."
        )
    return control.rstrip() + "\n\n# Target voice examples\n\n" + "\n\n".join(sections) + "\n"


def source_text(index: int) -> str:
    return (
        f"# 문서 {index}\n\n"
        "이 문장은 실행에 대한 검토를 진행합니다.\n\n"
        "따라서 이 문장은 결과를 설명합니다.\n\n"
        "마지막 문장은 판단을 정리합니다.\n"
    )


class Harness:
    def __init__(self, base: Path) -> None:
        self.home = base / "home"
        self.root = self.home / config.ARCHIVE_SUFFIX
        self.development: dict[str, Path] = {}
        self.evaluation: dict[str, Path] = {}
        for index, filename in enumerate(config.DEVELOPMENT_SOURCES, 1):
            path = self.home / "sources" / "development" / filename
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(source_text(index), encoding="utf-8")
            self.development[filename] = path
        for index, filename in enumerate(config.EVALUATION_SOURCES, 11):
            path = self.home / "sources" / "evaluation" / filename
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(source_text(index), encoding="utf-8")
            self.evaluation[filename] = path
        control = self.home / config.CONTROL_PROMPT_SOURCE
        control.parent.mkdir(parents=True, exist_ok=True)
        control.write_text(CONTROL, encoding="utf-8")
        self.prompt = self.home / "candidate.md"
        self.prompt.write_text(example_prompt(), encoding="utf-8")

    def archive(self) -> ExperimentArchive:
        return ExperimentArchive(
            self.root,
            home=self.home,
            development_sources=self.development,
            evaluation_sources=self.evaluation,
        )

    def prepare_and_freeze(self) -> ExperimentArchive:
        archive = self.archive()
        archive.prepare()
        with mock.patch.object(config, "CONTROL_PROMPT_SHA256", config.sha256_text(CONTROL)):
            freeze_protocol(archive, self.prompt)
        return archive


def fake_codex(command: list[str], **kwargs: object) -> SimpleNamespace:
    rendered = str(kwargs["input"])
    document_sha = next(line.split(": ", 1)[1] for line in rendered.splitlines() if line.startswith("document_sha256: "))
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


class NaturalVoiceExamplesExperimentTest(unittest.TestCase):
    def setUp(self) -> None:
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.harness = Harness(Path(self.temp.name))
        self.control_patch = mock.patch.object(
            config, "CONTROL_PROMPT_SHA256", config.sha256_text(CONTROL)
        )
        self.control_patch.start()
        self.addCleanup(self.control_patch.stop)

    def test_prepare_rejects_partial_archive_and_detects_source_tamper(self) -> None:
        archive = self.harness.archive()
        archive.root.mkdir(parents=True)
        (archive.root / "partial").write_text("x", encoding="utf-8")
        with self.assertRaisesRegex(ArchiveError, "non-empty unsealed"):
            archive.prepare()
        (archive.root / "partial").unlink()
        archive.prepare()
        archive.input_path("development", archive.development_file_ids()[0]).write_text("changed", encoding="utf-8")
        with self.assertRaisesRegex(ArchiveError, "hash mismatch|differ"):
            archive.verify_source_seal()

    def test_protocol_freeze_is_idempotent_and_schedule_tamper_fails(self) -> None:
        archive = self.harness.prepare_and_freeze()
        first = freeze_protocol(archive, self.harness.prompt)
        second = freeze_protocol(archive, self.harness.prompt)
        self.assertEqual(first, second)
        path = archive.root / "control" / "protocol.lock.json"
        lock = read_json(path)
        lock["schedule_seed"] = 1
        path.write_text(json.dumps(lock), encoding="utf-8")
        with self.assertRaisesRegex(PromptError, "schedule"):
            freeze_protocol(archive, self.harness.prompt)

    @mock.patch("report_natural_voice_examples.records.subprocess.run", side_effect=fake_codex)
    def test_full_pipeline_locks_blind_magnitude_audit_and_passes(self, _: mock.Mock) -> None:
        archive = self.harness.prepare_and_freeze()
        run_pilots(archive)
        first_gate = authorize_full(archive)
        self.assertEqual(first_gate, authorize_full(archive))
        self.assertEqual(len(run_full(archive, workers=3)), 16)
        make_blind_packets(archive, seed=58)

        mapping = read_json(archive.root / "blind" / "private-mapping.lock.json")
        verdict_input = archive.root / "blind" / "verdicts.input.json"
        verdict_input.write_text(json.dumps({"verdicts": [
            {
                "packet_id": row["packet_id"],
                "choice": row["examples_slot"],
                "magnitude": "clear",
                "rationale": "예시 팔의 문장 호흡이 더 자연스럽다.",
            }
            for row in mapping["mappings"]
        ]}, ensure_ascii=False), encoding="utf-8")
        record_verdicts(archive, verdict_input)
        make_semantic_audit_pack(archive)
        pack = read_json(archive.root / "analysis" / "semantic-audit-pack.json")
        audit_input = archive.root / "analysis" / "semantic-audit.input.json"
        audit_input.write_text(json.dumps({"audits": [
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
        record_semantic_audit(archive, audit_input)
        audit_lock_path = archive.root / "analysis" / "semantic-audit.lock.json"
        audit_lock = read_json(audit_lock_path)
        audit_lock["audits"][0]["unexpected"] = True
        audit_lock_path.write_text(json.dumps(audit_lock, ensure_ascii=False), encoding="utf-8")
        with self.assertRaisesRegex(ReviewError, "schema"):
            validate_semantic_audit(archive)
        del audit_lock["audits"][0]["unexpected"]
        audit_lock_path.write_text(json.dumps(audit_lock, ensure_ascii=False), encoding="utf-8")
        summary = export_summary(archive)
        self.assertTrue(summary["experiment_passed"])
        self.assertEqual(summary["examples_wins"], 8)
        self.assertEqual(summary["clear_or_large_examples_wins"], 8)

    @mock.patch("report_natural_voice_examples.records.subprocess.run", side_effect=fake_codex)
    def test_verdicts_do_not_open_private_mapping_and_reject_weak_contract(self, _: mock.Mock) -> None:
        archive = self.harness.prepare_and_freeze()
        run_pilots(archive)
        authorize_full(archive)
        run_full(archive)
        make_blind_packets(archive, seed=2)
        mapping_path = archive.root / "blind" / "private-mapping.lock.json"
        hidden = mapping_path.with_suffix(".hidden")
        mapping_path.rename(hidden)
        verdict_input = archive.root / "blind" / "verdicts.input.json"
        verdict_input.write_text(json.dumps({"verdicts": [
            {"packet_id": f"packet-{index:02d}", "choice": "tie", "magnitude": "none", "rationale": "차이가 없다."}
            for index in range(1, 9)
        ]}, ensure_ascii=False), encoding="utf-8")
        record_verdicts(archive, verdict_input)
        validate_verdict_lock(archive)
        verdict_lock_path = archive.root / "blind" / "host-verdicts.lock.json"
        verdict_lock = read_json(verdict_lock_path)
        verdict_lock["verdicts"][0]["choice"] = "unknown"
        verdict_lock_path.write_text(json.dumps(verdict_lock, ensure_ascii=False), encoding="utf-8")
        with self.assertRaisesRegex(BlindError, "choice"):
            validate_verdict_lock(archive)
        mapping_path.write_bytes(hidden.read_bytes())
        with self.assertRaisesRegex(BlindError, "already locked"):
            record_verdicts(archive, verdict_input)

    @mock.patch("report_natural_voice_examples.records.subprocess.run", side_effect=fake_codex)
    def test_record_validation_rejects_prompt_sha_tamper(self, _: mock.Mock) -> None:
        archive = self.harness.prepare_and_freeze()
        run_pilots(archive)
        file_id = archive.development_file_ids()[0]
        path = archive.root / "runs" / "pilot" / file_id / "control" / "record.json"
        record = read_json(path)
        record["instruction_prompt_sha256"] = "0" * 64
        with self.assertRaisesRegex(RecordError, "identity"):
            validate_record(archive, record, "development", ("pilot",), file_id, "control")


if __name__ == "__main__":
    unittest.main()
