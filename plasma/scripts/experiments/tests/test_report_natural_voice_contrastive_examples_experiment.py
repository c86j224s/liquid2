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
from report_natural_voice_contrastive_examples import config
from report_natural_voice_contrastive_examples.archive import ArchiveError, ExperimentArchive
from report_natural_voice_contrastive_examples.blind import (
    BlindError,
    make_blind_packets,
    record_verdicts,
    validate_verdict_lock,
)
from report_natural_voice_contrastive_examples.prompts import (
    CONTRASTIVE_CATEGORIES,
    PromptError,
    freeze_protocol,
    load_prompt,
)
from report_natural_voice_contrastive_examples.records import (
    RecordError,
    run_document,
    validate_record,
)
from report_natural_voice_contrastive_examples.recovery import (
    resume_full_contract,
    resume_pilot_contract,
)
from report_natural_voice_contrastive_examples.review import (
    ReviewError,
    make_semantic_audit_pack,
    record_semantic_audit,
    validate_semantic_audit,
)
from report_natural_voice_contrastive_examples.runner import authorize_full, run_full, run_pilots
from report_natural_voice_contrastive_examples.summary import export_summary


BASE = "# Base\n\nKeep the document intact.\n"
CONTROL = BASE.rstrip() + "\n\n# Target voice examples\n\nExisting sealed control.\n"


def contrastive_prompt(base: str = BASE) -> str:
    sections = []
    for category in CONTRASTIVE_CATEGORIES:
        sections.append(
            f"## {category}\n"
            "Edit before:\n기존 문장입니다.\n"
            "Edit after:\n고친 문장입니다.\n"
            "Leave unchanged:\n그대로 둘 문장입니다.\n"
            "Forbidden before:\n판단을 유보합니다.\n"
            "Forbidden after:\n사실로 확인합니다.\n"
            "Why forbidden:\n판단 강도를 바꿉니다."
        )
    return base.rstrip() + "\n\n# Contrastive edit decisions\n\n" + "\n\n".join(sections) + "\n"


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
        base = self.home / config.BASE_PROMPT_SOURCE
        base.parent.mkdir(parents=True, exist_ok=True)
        base.write_text(BASE, encoding="utf-8")
        self.prompt = self.home / "candidate.md"
        self.prompt.write_text(contrastive_prompt(), encoding="utf-8")

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
        with (
            mock.patch.object(config, "BASE_PROMPT_SHA256", config.sha256_text(BASE)),
            mock.patch.object(config, "CONTROL_PROMPT_SHA256", config.sha256_text(CONTROL)),
        ):
            freeze_protocol(archive, self.prompt)
        return archive


def fake_codex(command: list[str], **kwargs: object) -> SimpleNamespace:
    rendered = str(kwargs["input"])
    document_sha = next(line.split(": ", 1)[1] for line in rendered.splitlines() if line.startswith("document_sha256: "))
    numbered = [line for line in rendered.splitlines() if "\t" in line]
    evidence = [int(numbered[index].split("\t", 1)[0]) for index in (0, 1, 2)]
    line_number, line_sha, original_line = numbered[2].split("\t", 2)
    payload = {
        "document_sha256": document_sha,
        "diagnoses": [
            {"category": "translationese_nominalization", "evidence_line_numbers": [evidence[0]]},
            {"category": "formulaic_connection", "evidence_line_numbers": [evidence[1]]},
            {"category": "uniform_cadence", "evidence_line_numbers": [evidence[2]]},
        ],
        "edits": [{
            "line_number": int(line_number),
            "original_line_sha256": line_sha,
            "original_line": original_line,
            "replacement_line": "이 문장은 실행을 검토합니다.",
            "category": "translationese_nominalization",
            "safety_rationale": "명사화를 직접 동사로 바꾸고 판단 범위는 유지한다.",
        }],
    }
    output = Path(command[command.index("-o") + 1])
    output.write_text(json.dumps(payload, ensure_ascii=False), encoding="utf-8")
    return SimpleNamespace(returncode=0, stdout="", stderr="")


def fake_codex_missing_diagnosis(command: list[str], **kwargs: object) -> SimpleNamespace:
    rendered = str(kwargs["input"])
    document_sha = next(line.split(": ", 1)[1] for line in rendered.splitlines() if line.startswith("document_sha256: "))
    numbered = [line for line in rendered.splitlines() if "\t" in line]
    line_number, line_sha, original_line = numbered[2].split("\t", 2)
    payload = {
        "document_sha256": document_sha,
        "diagnoses": [
            {"category": "formulaic_connection", "evidence_line_numbers": [1]},
            {"category": "uniform_cadence", "evidence_line_numbers": [2]},
            {"category": "process_narration", "evidence_line_numbers": [3]},
        ],
        "edits": [{
            "line_number": int(line_number),
            "original_line_sha256": line_sha,
            "original_line": original_line,
            "replacement_line": "이 문장은 실행을 검토합니다.",
            "category": "translationese_nominalization",
            "safety_rationale": "명사화를 직접 동사로 바꾸고 판단 범위는 유지한다.",
        }],
    }
    output = Path(command[command.index("-o") + 1])
    output.write_text(json.dumps(payload, ensure_ascii=False), encoding="utf-8")
    return SimpleNamespace(returncode=0, stdout="", stderr="")


def fake_codex_bad_line_hash(command: list[str], **kwargs: object) -> SimpleNamespace:
    result = fake_codex(command, **kwargs)
    output = Path(command[command.index("-o") + 1])
    payload = read_json(output)
    correct = payload["edits"][0]["original_line_sha256"]
    payload["edits"][0]["original_line_sha256"] = (
        ("0" if correct[0] != "0" else "1") + correct[1:]
    )
    output.write_text(json.dumps(payload, ensure_ascii=False), encoding="utf-8")
    return result


class NaturalVoiceContrastiveExamplesExperimentTest(unittest.TestCase):
    def setUp(self) -> None:
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.harness = Harness(Path(self.temp.name))
        self.control_patch = mock.patch.object(
            config, "CONTROL_PROMPT_SHA256", config.sha256_text(CONTROL)
        )
        self.control_patch.start()
        self.addCleanup(self.control_patch.stop)
        self.base_patch = mock.patch.object(
            config, "BASE_PROMPT_SHA256", config.sha256_text(BASE)
        )
        self.base_patch.start()
        self.addCleanup(self.base_patch.stop)

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

    def test_missing_diagnosis_is_normalized_without_retrying_model(self) -> None:
        archive = self.harness.prepare_and_freeze()
        file_id = archive.development_file_ids()[0]
        prompt, prompt_sha = load_prompt(archive, "contrastive")
        with self.assertRaisesRegex(RecordError, "locked protocol amendment"):
            run_document(
                archive, "development", file_id, "contrastive", prompt, prompt_sha,
                ("pilot",), subprocess_run=fake_codex_missing_diagnosis,
            )
        run_dir = archive.root / "runs" / "pilot" / file_id / "contrastive"
        raw_path = run_dir / "raw-output.json"
        amendment = {
            "experiment_id": config.EXPERIMENT_ID,
            "status": "locked_before_contract_normalization",
            "base_protocol_sha256": config.sha256_file(archive.root / "control" / "protocol.lock.json"),
            "trigger": {
                "raw_output_path": archive.rel(raw_path),
                "raw_output_sha256": config.sha256_file(raw_path),
                "added_diagnoses": [{
                    "category": "translationese_nominalization",
                    "evidence_line_numbers": [3],
                }],
            },
            "rule": config.CONTRACT_AMENDMENT_RULE,
            "scope": "all_experiment_59_cells",
            "model_calls_retried": False,
        }
        amendment_path = archive.root / "control" / "protocol-amendment-01.lock.json"
        amendment_path.write_text(json.dumps(amendment), encoding="utf-8")
        record = resume_pilot_contract(archive, file_id, "contrastive")
        self.assertEqual(record["accepted_edit_count"], 1)
        self.assertIn("contract_normalization_path", record)
        normalized_path = archive.root / str(record["normalized_output_path"])
        normalized = read_json(normalized_path)
        self.assertEqual(normalized["edits"], read_json(raw_path)["edits"])
        self.assertEqual(normalized["diagnoses"][-1]["category"], "translationese_nominalization")

    @mock.patch("report_natural_voice_contrastive_examples.records.subprocess.run", side_effect=fake_codex)
    def test_bad_line_hash_is_normalized_from_exact_source_without_retry(self, mocked: mock.Mock) -> None:
        archive = self.harness.prepare_and_freeze()
        run_pilots(archive)
        authorize_full(archive)
        file_id = archive.evaluation_file_ids()[0]
        prompt, prompt_sha = load_prompt(archive, "contrastive")
        with self.assertRaisesRegex(RecordError, "locked protocol amendment"):
            run_document(
                archive, "evaluation", file_id, "contrastive", prompt, prompt_sha,
                ("full",), subprocess_run=fake_codex_bad_line_hash,
            )
        model_call_count = mocked.call_count
        run_dir = archive.root / "runs" / "full" / file_id / "contrastive"
        raw_path = run_dir / "raw-output.json"
        raw_hash = config.sha256_file(raw_path)
        raw = read_json(raw_path)
        edit = raw["edits"][0]
        derived = config.sha256_text(edit["original_line"])
        correction = {
            "line_number": edit["line_number"],
            "claimed_sha256": edit["original_line_sha256"],
            "derived_sha256": derived,
        }
        amendment = {
            "experiment_id": config.EXPERIMENT_ID,
            "status": "locked_before_hash_normalization",
            "base_protocol_sha256": config.sha256_file(archive.root / "control" / "protocol.lock.json"),
            "trigger": {
                "file_id": file_id,
                "raw_output_path": archive.rel(raw_path),
                "raw_output_sha256": raw_hash,
                "corrections": [correction],
            },
            "rule": config.HASH_AMENDMENT_RULE,
            "scope": "all_experiment_59_cells",
            "model_calls_retried": False,
        }
        amendment_path = archive.root / "control" / "protocol-amendment-02.lock.json"
        amendment_path.write_text(json.dumps(amendment), encoding="utf-8")

        record = resume_full_contract(archive, file_id, "contrastive")

        self.assertEqual(mocked.call_count, model_call_count)
        self.assertEqual(config.sha256_file(raw_path), raw_hash)
        self.assertEqual(record["accepted_edit_count"], 1)
        normalized = read_json(archive.root / str(record["hash_normalized_output_path"]))
        self.assertEqual(normalized["edits"][0]["original_line_sha256"], derived)
        self.assertEqual(normalized["edits"][0]["original_line"], edit["original_line"])
        self.assertEqual(normalized["edits"][0]["replacement_line"], edit["replacement_line"])

    @mock.patch("report_natural_voice_contrastive_examples.records.subprocess.run", side_effect=fake_codex)
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
                "choice": row["contrastive_slot"],
                "magnitude": "clear",
                "rationale": "대조형 예시 팔의 문장 호흡이 더 자연스럽다.",
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
                "reviewed_line_numbers": [edit["line_number"] for edit in cell["accepted_edits"]],
                "semantic_drift_lines": [edit["line_number"] for edit in cell["accepted_edits"]]
                if cell["arm"] == "control" else [],
                "claim_scope_drift_lines": [edit["line_number"] for edit in cell["accepted_edits"]]
                if cell["arm"] == "control" else [],
                "citation_drift_lines": [],
                "notes": "대조군 이탈은 보고하되 실험군 안전성 문턱에는 포함하지 않는다.",
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
        self.assertTrue(summary["screening_passed"])
        self.assertFalse(summary["product_adoption_evaluated"])
        self.assertEqual(summary["contrastive_wins"], 8)
        self.assertEqual(summary["clear_or_large_contrastive_wins"], 8)
        self.assertEqual(summary["drift_by_arm"]["control"]["semantic_drift"], 8)
        self.assertEqual(summary["drift_by_arm"]["contrastive"]["semantic_drift"], 0)

    @mock.patch("report_natural_voice_contrastive_examples.records.subprocess.run", side_effect=fake_codex)
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

    @mock.patch("report_natural_voice_contrastive_examples.records.subprocess.run", side_effect=fake_codex)
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
