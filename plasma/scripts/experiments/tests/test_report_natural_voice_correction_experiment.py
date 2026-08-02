from __future__ import annotations

import importlib.util
import json
import os
from pathlib import Path
from types import SimpleNamespace
import subprocess
import sys
import tempfile
import unittest
from unittest import mock


EXPERIMENTS = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(EXPERIMENTS))

from report_natural_voice_correction import codex_runner, config, edits, guards
from report_natural_voice_correction.archive import ARCHIVE_DIRS, ExperimentArchive, write_json_atomic
from report_natural_voice_correction.blind import BlindError, make_blind_packets, record_host_verdicts
from report_natural_voice_correction.codex_runner import (
    EXPERIMENT_57_PILOT_FILES,
    PILOT_FILES,
    RunnerError,
    freeze_prompt,
    lint_prompt_file,
    run_full,
    run_one_document,
    run_pilot,
)
from report_natural_voice_correction.summary import export_public_summary


SCRIPT = EXPERIMENTS / "report_natural_voice_correction_experiment.py"
NO_BYTECODE_ENV = {**os.environ, "PYTHONDONTWRITEBYTECODE": "1"}
SPEC = importlib.util.spec_from_file_location("report_natural_voice_correction_entrypoint", SCRIPT)
entrypoint = importlib.util.module_from_spec(SPEC)
assert SPEC and SPEC.loader
SPEC.loader.exec_module(entrypoint)


def make_archive(root: Path, contents: dict[str, str], experiment: str = "56") -> ExperimentArchive:
    for name in ARCHIVE_DIRS:
        (root / name).mkdir(parents=True, exist_ok=True)
    expected: dict[str, str] = {}
    rows: list[dict[str, str]] = []
    for filename, text in contents.items():
        sha = config.sha256_text(text)
        expected[filename] = sha
        (root / "inputs" / filename).write_text(text, encoding="utf-8")
        rows.append({"filename": filename, "source_sha256": sha, "destination_sha256": sha})
    write_json_atomic(root / "control" / "source-manifest.lock.json", {
        "experiment_id": config.experiment_id(experiment),
        "invalid_material_used": False,
        "files": rows,
    })
    return ExperimentArchive(root, expected, experiment)


def response_for(original: str, replacement: str | None = None) -> dict[str, object]:
    original_line = edits.split_document_lines(original)[0]
    return {
        "document_sha256": config.sha256_text(original),
        "diagnoses": [
            {"category": "translationese_nominalization", "evidence_line_numbers": [1]},
            {"category": "formulaic_connection", "evidence_line_numbers": [1]},
            {"category": "uniform_cadence", "evidence_line_numbers": [1]},
        ],
        "edits": [{
            "line_number": 1,
            "original_line_sha256": config.sha256_text(original_line),
            "original_line": original_line,
            "replacement_line": replacement or original_line.replace("합니다.", "해요."),
            "category": "translationese_nominalization",
            "safety_rationale": "line-local wording only",
        }],
    }


def response_for_edits(original: str, replacements_by_line: dict[int, str]) -> dict[str, object]:
    lines = edits.split_document_lines(original)
    return {
        "document_sha256": config.sha256_text(original),
        "diagnoses": [
            {"category": "translationese_nominalization", "evidence_line_numbers": sorted(replacements_by_line)},
            {"category": "formulaic_connection", "evidence_line_numbers": [1]},
            {"category": "uniform_cadence", "evidence_line_numbers": [1]},
        ],
        "edits": [
            {
                "line_number": line_number,
                "original_line_sha256": config.sha256_text(lines[line_number - 1]),
                "original_line": lines[line_number - 1],
                "replacement_line": replacement,
                "category": "translationese_nominalization",
                "safety_rationale": "line-local wording only",
            }
            for line_number, replacement in sorted(replacements_by_line.items())
        ],
    }


def write_archive_prompt(archive: ExperimentArchive, relative_path: str, text: str) -> Path:
    path = archive.root / relative_path
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(text, encoding="utf-8")
    return path


def write_frozen_prompt_lock(archive: ExperimentArchive, text: str) -> str:
    prompt_sha = config.sha256_text(text)
    path = archive.root / "control" / "instruction-prompt.lock.md"
    path.write_text(text, encoding="utf-8")
    if archive.experiment == "57":
        prompt_path = write_archive_prompt(archive, "control/prompts/refined-01.md", text)
        write_json_atomic(archive.root / "control" / "prompt-freeze.lock.json", {
            "corpus_hashes": [
                {"file_id": filename[:-3], "sha256": sha}
                for filename, sha in sorted(archive.expected_files.items())
            ],
            "experiment_id": config.EXPERIMENT_57_ID,
            "fixed_pilot_pair": dict(EXPERIMENT_57_PILOT_FILES),
            "inherited_from_experiment_id": config.EXPERIMENT_56_ID,
            "instruction_prompt_lock_path": archive.rel(path),
            "lock_kind": "pre_call_instruction_prompt_freeze",
            "locked_at_utc": "2026-07-30T02:53:42Z",
            "model": config.MODEL,
            "no_policy_iteration_in_experiment_57": True,
            "no_prompt_iteration_in_experiment_57": True,
            "prompt_path": archive.rel(prompt_path),
            "prompt_sha256": prompt_sha,
            "reasoning_effort": config.REASONING_EFFORT,
            "refinement_history": {
                "additional_experiment_57_refinement_count": 0,
                "history": ["initial", "refined-01"],
                "inherited_refinement_count": 1,
            },
            "selected_prompt_source": "refined-01",
            "status": "frozen_before_any_experiment_57_model_call",
        })
    else:
        write_json_atomic(archive.root / "control" / "prompt-freeze.lock.json", {
            "experiment_id": archive.experiment_id,
            "instruction_prompt_path": archive.rel(path),
            "instruction_prompt_sha256": prompt_sha,
        })
    return prompt_sha


def write_pilot_record(
    archive: ExperimentArchive,
    prompt_sha: str,
    step: str,
    *,
    file_id: str | None = None,
    record_sha: str | None = None,
    passed: bool = True,
) -> Path:
    expected_file_id = codex_runner.PILOT_FILES_BY_EXPERIMENT[archive.experiment][step]
    path = archive.root / "runs" / "pilot" / step / prompt_sha / expected_file_id / "record.json"
    write_json_atomic(path, {
        "file_id": file_id or expected_file_id,
        "instruction_prompt_sha256": record_sha or prompt_sha,
        "hard_gates_passed": passed,
    })
    return path


def write_success_record(
    archive: ExperimentArchive,
    file_id: str,
    phase: list[str],
    *,
    prompt_sha: str = "prompt-sha",
    candidate_text: str | None = None,
) -> Path:
    filename = archive.filename_for_file_id(file_id)
    original = archive.input_path_for_file_id(file_id).read_text(encoding="utf-8")
    candidate_body = candidate_text if candidate_text is not None else original
    run_dir = archive.root / "runs" / Path(*phase) / file_id
    run_dir.mkdir(parents=True, exist_ok=True)
    raw_output = run_dir / "raw-output.json"
    raw_output.write_text("{}\n", encoding="utf-8")
    command = run_dir / "codex-command.json"
    write_json_atomic(command, {"command": ["codex"], "returncode": 0, "stdout": "", "stderr": "", "duration_seconds": 0})
    rendered = run_dir / "rendered-prompt.txt"
    rendered.write_text("rendered prompt\n", encoding="utf-8")
    candidate = run_dir / "candidate.md"
    candidate.write_text(candidate_body, encoding="utf-8")
    record: dict[str, object] = {
        "experiment_id": archive.experiment_id,
        "phase": phase,
        "file_id": file_id,
        "source_filename": filename,
        "document_sha256": config.sha256_text(original),
        "instruction_prompt_sha256": prompt_sha,
        "rendered_prompt_sha256": config.sha256_file(rendered),
        "input_sha256": config.sha256_file(rendered),
        "model": config.MODEL,
        "reasoning_effort": config.REASONING_EFFORT,
        "raw_output_path": archive.rel(raw_output),
        "raw_output_sha256": config.sha256_file(raw_output),
        "candidate_path": archive.rel(candidate),
        "candidate_sha256": config.sha256_file(candidate),
        "hard_gates_passed": True,
        "codex_command_path": archive.rel(command),
        "codex_command_sha256": config.sha256_file(command),
        "rendered_prompt_path": archive.rel(rendered),
    }
    if archive.experiment == "57":
        decisions = run_dir / "edit-decisions.json"
        write_json_atomic(decisions, {
            "experiment_id": archive.experiment_id,
            "file_id": file_id,
            "proposed_count": 0,
            "accepted_count": 0,
            "rejected_count": 0,
            "decisions": [],
        })
        aggregate = run_dir / "aggregate-gates.json"
        write_json_atomic(aggregate, {
            "experiment_id": archive.experiment_id,
            "file_id": file_id,
            "proposed_count": 0,
            "accepted_count": 0,
            "rejected_count": 0,
            "aggregate_hard_gate_failures": [],
            "hard_gates_passed": True,
        })
        record.update({
            "proposed_edit_count": 0,
            "accepted_edit_count": 0,
            "rejected_edit_count": 0,
            "edit_decisions_path": archive.rel(decisions),
            "edit_decisions_sha256": config.sha256_file(decisions),
            "aggregate_gates_path": archive.rel(aggregate),
            "aggregate_gates_sha256": config.sha256_file(aggregate),
            "aggregate_hard_gate_failures": [],
        })
    record_path = run_dir / "record.json"
    write_json_atomic(record_path, record)
    return record_path


class NaturalVoiceCorrectionExperimentTests(unittest.TestCase):
    def test_cli_help_lists_all_w2_actions(self) -> None:
        output = subprocess.check_output([sys.executable, str(SCRIPT), "--help"], text=True, env=NO_BYTECODE_ENV)
        for action in config.CLI_ACTIONS:
            self.assertIn(action, output)
        self.assertIn("--experiment", output)
        self.assertIn("{56,57}", output)
        with mock.patch.object(sys, "argv", [str(SCRIPT), "--action", "verify-source-seal"]):
            self.assertEqual(entrypoint.parse_args().experiment, "56")

    def test_archive_resolution_is_fixed_and_hash_mismatch_fails_closed(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            home = Path(directory)
            with mock.patch("report_natural_voice_correction.config.Path.home", return_value=home):
                self.assertEqual(config.resolve_archive(), (home / config.ARCHIVE_SUFFIX).resolve())
                self.assertEqual(config.resolve_archive(experiment="57"), (home / config.archive_suffix("57")).resolve())
                with self.assertRaisesRegex(ValueError, "fixed experiment 56 archive"):
                    config.resolve_archive(home / "other")
                with self.assertRaisesRegex(ValueError, "fixed experiment 57 archive"):
                    config.resolve_archive(home / "other", experiment="57")
                with self.assertRaisesRegex(ValueError, "experiment must be 56 or 57"):
                    config.resolve_archive(experiment="58")
                with self.assertRaisesRegex(ValueError, "fixed experiment 56 archive"):
                    ExperimentArchive(home / "other")
                with self.assertRaisesRegex(ValueError, "fixed experiment 57 archive"):
                    ExperimentArchive(home / "other", experiment="57")
            archive = make_archive(Path(directory) / "archive", {"doc.md": "sealed\n"})
            self.assertTrue(archive.verify_source_seal()["passed"])
            (archive.root / "inputs" / "doc.md").write_text("changed\n", encoding="utf-8")
            with self.assertRaisesRegex(ValueError, "SHA mismatch"):
                archive.verify_source_seal()

    def test_prompt_lint_rejects_repo_prompt_and_forbidden_outputs(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            prompt = Path(directory) / "prompt.md"
            prompt.write_text("Return structured line edits only.\n", encoding="utf-8")
            self.assertRegex(str(lint_prompt_file(prompt)["sha256"]), r"^[0-9a-f]{64}$")
            prompt.write_text("Return rewritten_document.\n", encoding="utf-8")
            with self.assertRaisesRegex(RunnerError, "forbidden token"):
                lint_prompt_file(prompt)
        with self.assertRaisesRegex(RunnerError, "Git worktree"):
            lint_prompt_file(SCRIPT)

    def test_response_parser_is_strict_and_rejects_rewrites_duplicates_and_unknown_categories(self) -> None:
        original = "문장이 조금 딱딱합니다.\n"
        valid = response_for(original)
        parsed = edits.parse_model_response(valid)
        self.assertEqual(parsed.edits[0].line_number, 1)
        invalid = dict(valid)
        invalid["rewritten_document"] = "free form"
        with self.assertRaisesRegex(ValueError, "extra"):
            edits.parse_model_response(invalid)
        duplicate = dict(valid)
        duplicate["edits"] = [valid["edits"][0], dict(valid["edits"][0])]
        with self.assertRaisesRegex(ValueError, "duplicate line_number"):
            edits.parse_model_response(duplicate)
        unknown = dict(valid)
        unknown["diagnoses"] = [dict(row) for row in valid["diagnoses"]]
        unknown["diagnoses"][0]["category"] = "document_specific_blacklist"
        with self.assertRaisesRegex(ValueError, "allowed enum"):
            edits.parse_model_response(unknown)
        spaced = json.loads(json.dumps(valid))
        spaced["edits"][0]["safety_rationale"] = "  line local only  "
        self.assertEqual(edits.parse_model_response(spaced).edits[0].safety_rationale, "line local only")
        blank = json.loads(json.dumps(valid))
        blank["edits"][0]["safety_rationale"] = "   "
        with self.assertRaisesRegex(ValueError, "safety_rationale"):
            edits.parse_model_response(blank)
        self.assertNotIn("uniqueItems", json.dumps(edits.response_schema()))
        duplicate_evidence = json.loads(json.dumps(valid))
        duplicate_evidence["diagnoses"][0]["evidence_line_numbers"] = [1, 1]
        with self.assertRaisesRegex(ValueError, "must not contain duplicates"):
            edits.parse_model_response(duplicate_evidence)
        self.assertEqual(edits.response_schema()["properties"]["edits"]["maxItems"], 24)

    def test_response_parser_rejects_25_edits_before_candidate_generation(self) -> None:
        original = "".join(f"문장 {index}입니다.\n" for index in range(1, 26))
        oversized = response_for_edits(original, {
            index: f"문장 {index}이에요."
            for index in range(1, 26)
        })
        with self.assertRaisesRegex(ValueError, "no more than 24"):
            edits.parse_model_response(oversized)

    def test_run_full_rejects_prompt_overrides_and_requires_frozen_prompt(self) -> None:
        fake_archive = mock.Mock()
        fake_archive.verify_source_seal.return_value = {"passed": True}
        args = SimpleNamespace(
            action="run-full",
            experiment="56",
            archive=None,
            prompt=Path("/path/to/prompt.md"),
            pilot_step="first",
            verdicts=None,
            seed=None,
        )
        with mock.patch.object(entrypoint, "parse_args", return_value=args), mock.patch.object(
            entrypoint.ExperimentArchive, "from_path", return_value=fake_archive
        ), mock.patch.object(entrypoint, "run_full") as run_full_mock:
            with self.assertRaisesRegex(RunnerError, "--prompt"):
                entrypoint.main()
            run_full_mock.assert_not_called()
        process = subprocess.run(
            [sys.executable, str(SCRIPT), "--action", "run-full", "--prompt-file", "/path/to/prompt.md"],
            text=True,
            capture_output=True,
            env=NO_BYTECODE_ENV,
        )
        self.assertNotEqual(process.returncode, 0)
        self.assertIn("unrecognized arguments", process.stderr)

    def test_run_full_uses_only_frozen_prompt_lock_and_rejects_sha_mismatch(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            archive = make_archive(Path(directory) / "archive", {"doc.md": "문장이 조금 딱딱합니다.\n"})
            prompt_body = archive.root / "control" / "instruction-prompt.lock.md"
            prompt_body.write_text("Frozen prompt.\n", encoding="utf-8")
            write_json_atomic(archive.root / "control" / "prompt-freeze.lock.json", {
                "instruction_prompt_path": archive.rel(prompt_body),
                "instruction_prompt_sha256": config.sha256_text("Different prompt.\n"),
            })
            fake_run = mock.Mock()
            with self.assertRaisesRegex(RunnerError, "frozen prompt SHA-256 mismatch"):
                run_full(archive, fake_run)
            fake_run.assert_not_called()

    def test_experiment_57_rejects_prompt_overrides_freeze_and_uses_fixed_pilots(self) -> None:
        self.assertEqual(config.EXPERIMENT_57_FROZEN_PROMPT_SHA256, "4922b0cc2774dfe972c5403603f0dd8fe6a0e172ec2ef838fdc54ff039ee565f")
        self.assertEqual(EXPERIMENT_57_PILOT_FILES, {
            "first": "02-wang-anshi-strict-read-A",
            "second": "03-go-raft-exploratory-read-B",
        })
        with tempfile.TemporaryDirectory() as directory:
            original = "문장이 조금 딱딱합니다.\n"
            file_id = EXPERIMENT_57_PILOT_FILES["first"]
            archive = make_archive(Path(directory) / "archive", {f"{file_id}.md": original}, experiment="57")
            prompt_override = write_archive_prompt(archive, "control/prompts/refined-01.md", "Refined prompt.\n")
            with self.assertRaisesRegex(RunnerError, "--prompt"):
                run_pilot(archive, prompt_override, "first", mock.Mock())
            with self.assertRaisesRegex(RunnerError, "freeze-prompt"):
                freeze_prompt(archive, prompt_override)
            prompt_sha = write_frozen_prompt_lock(archive, "Frozen prompt.\n")
            calls: list[list[str]] = []

            def fake_run(command: list[str], input: str, text: bool, capture_output: bool) -> SimpleNamespace:
                calls.append(command)
                Path(command[command.index("-o") + 1]).write_text(json.dumps(response_for(original), ensure_ascii=False), encoding="utf-8")
                return SimpleNamespace(returncode=0, stdout="", stderr="")

            with mock.patch.object(config, "EXPERIMENT_57_FROZEN_PROMPT_SHA256", prompt_sha):
                record = run_pilot(archive, None, "first", fake_run)
            self.assertEqual(len(calls), 1)
            self.assertEqual(record["experiment_id"], config.EXPERIMENT_57_ID)
            self.assertEqual(record["file_id"], "02-wang-anshi-strict-read-A")
            self.assertEqual(record["phase"], ["pilot", "first", prompt_sha])

    def test_experiment_57_load_prompt_accepts_actual_w4b_schema_and_rejects_mismatches(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            archive = make_archive(Path(directory) / "archive", {
                f"{EXPERIMENT_57_PILOT_FILES['first']}.md": "첫 문장입니다.\n",
                f"{EXPERIMENT_57_PILOT_FILES['second']}.md": "둘째 문장입니다.\n",
            }, experiment="57")
            prompt_sha = write_frozen_prompt_lock(archive, "Frozen prompt.\n")
            with mock.patch.object(config, "EXPERIMENT_57_FROZEN_PROMPT_SHA256", prompt_sha):
                text, sha = codex_runner.load_prompt(archive, None)
            self.assertEqual(text, "Frozen prompt.\n")
            self.assertEqual(sha, prompt_sha)
            lock_path = archive.root / "control" / "prompt-freeze.lock.json"
            lock = json.loads(lock_path.read_text(encoding="utf-8"))
            bad_status = dict(lock)
            bad_status["status"] = "draft"
            write_json_atomic(lock_path, bad_status)
            with mock.patch.object(config, "EXPERIMENT_57_FROZEN_PROMPT_SHA256", prompt_sha):
                with self.assertRaisesRegex(RunnerError, "status mismatch"):
                    codex_runner.load_prompt(archive, None)
            write_json_atomic(lock_path, lock)
            bad_pair = json.loads(json.dumps(lock))
            bad_pair["fixed_pilot_pair"]["first"] = "01-wang-anshi-exploratory-read-A"
            write_json_atomic(lock_path, bad_pair)
            with mock.patch.object(config, "EXPERIMENT_57_FROZEN_PROMPT_SHA256", prompt_sha):
                with self.assertRaisesRegex(RunnerError, "fixed_pilot_pair"):
                    codex_runner.load_prompt(archive, None)
            write_json_atomic(lock_path, lock)
            (archive.root / str(lock["prompt_path"])).write_text("tampered\n", encoding="utf-8")
            with mock.patch.object(config, "EXPERIMENT_57_FROZEN_PROMPT_SHA256", prompt_sha):
                with self.assertRaisesRegex(RunnerError, "prompt_path SHA"):
                    codex_runner.load_prompt(archive, None)

    def test_experiment_57_selective_acceptance_records_every_edit_and_reason_codes(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            original = "첫 문장이 조금 딱딱합니다.\n수치는 50%입니다.\n"
            archive = make_archive(Path(directory) / "archive", {"doc.md": original}, experiment="57")
            response = response_for_edits(original, {
                1: "첫 문장이 조금 딱딱해요.",
                2: "수치는 51%입니다.",
            })

            def fake_run(command: list[str], input: str, text: bool, capture_output: bool) -> SimpleNamespace:
                Path(command[command.index("-o") + 1]).write_text(json.dumps(response, ensure_ascii=False), encoding="utf-8")
                return SimpleNamespace(returncode=0, stdout="", stderr="")

            record = run_one_document(archive, "doc", "Prompt", config.sha256_text("Prompt"), ("pilot", "selective"), fake_run)
            self.assertEqual(record["proposed_edit_count"], 2)
            self.assertEqual(record["accepted_edit_count"], 1)
            self.assertEqual(record["rejected_edit_count"], 1)
            self.assertEqual(record["aggregate_hard_gate_failures"], [])
            decisions = json.loads((archive.root / str(record["edit_decisions_path"])).read_text(encoding="utf-8"))
            self.assertEqual(decisions["proposed_count"], decisions["accepted_count"] + decisions["rejected_count"])
            self.assertEqual(len(decisions["decisions"]), 2)
            rejected = [row for row in decisions["decisions"] if row["disposition"] == "rejected"]
            self.assertEqual(len(rejected), 1)
            self.assertIn("numbers_dates_percentages", rejected[0]["reason_codes"])
            self.assertTrue(set(rejected[0]["reason_codes"]).issubset(set(guards.HARD_GATE_REASON_CODES)))
            aggregate = json.loads((archive.root / str(record["aggregate_gates_path"])).read_text(encoding="utf-8"))
            self.assertTrue(aggregate["hard_gates_passed"])
            candidate = (archive.root / str(record["candidate_path"])).read_text(encoding="utf-8")
            self.assertIn("첫 문장이 조금 딱딱해요.", candidate)
            self.assertIn("수치는 50%입니다.", candidate)
            self.assertRegex(str(record["edit_decisions_sha256"]), r"^[0-9a-f]{64}$")
            self.assertRegex(str(record["aggregate_gates_sha256"]), r"^[0-9a-f]{64}$")

    def test_experiment_57_decision_validator_rejects_missing_unknown_and_silent_drops(self) -> None:
        original = "첫 문장이 조금 딱딱합니다.\n수치는 50%입니다.\n"
        response = edits.parse_model_response(response_for_edits(original, {
            1: "첫 문장이 조금 딱딱해요.",
            2: "수치는 51%입니다.",
        }))
        valid_payload = {
            "proposed_count": 2,
            "accepted_count": 1,
            "rejected_count": 1,
            "decisions": [
                {"line_number": 1, "category": "translationese_nominalization", "disposition": "accepted", "reason_codes": []},
                {
                    "line_number": 2,
                    "category": "translationese_nominalization",
                    "disposition": "rejected",
                    "reason_codes": ["numbers_dates_percentages"],
                },
            ],
        }
        codex_runner._validate_edit_decisions_payload(response, valid_payload)
        missing_reason = json.loads(json.dumps(valid_payload))
        missing_reason["decisions"][1]["reason_codes"] = []
        with self.assertRaisesRegex(RunnerError, "non-empty"):
            codex_runner._validate_edit_decisions_payload(response, missing_reason)
        unknown_reason = json.loads(json.dumps(valid_payload))
        unknown_reason["decisions"][1]["reason_codes"] = ["invented_gate"]
        with self.assertRaisesRegex(RunnerError, "unknown hard-gate"):
            codex_runner._validate_edit_decisions_payload(response, unknown_reason)
        silent_drop = json.loads(json.dumps(valid_payload))
        silent_drop["decisions"] = silent_drop["decisions"][:1]
        with self.assertRaisesRegex(RunnerError, "every proposed edit"):
            codex_runner._validate_edit_decisions_payload(response, silent_drop)

    def test_experiment_57_aggregate_failure_preserves_evidence_no_candidate_and_blocks_retry(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            original = "첫 문장이 조금 딱딱합니다.\n둘째 문장이 조금 딱딱합니다.\n"
            archive = make_archive(Path(directory) / "archive", {"doc.md": original}, experiment="57")
            response = response_for_edits(original, {
                1: "첫 문장이 조금 딱딱해요.",
                2: "둘째 문장이 조금 딱딱해요.",
            })
            phase = ("pilot", "aggregate")
            run_dir = archive.root / "runs" / "pilot" / "aggregate" / "doc"

            def fake_run(command: list[str], input: str, text: bool, capture_output: bool) -> SimpleNamespace:
                Path(command[command.index("-o") + 1]).write_text(json.dumps(response, ensure_ascii=False), encoding="utf-8")
                return SimpleNamespace(returncode=0, stdout="", stderr="")

            with self.assertRaisesRegex(RunnerError, "aggregate hard gates failed"):
                run_one_document(archive, "doc", "Prompt", config.sha256_text("Prompt"), phase, fake_run)
            self.assertTrue((run_dir / "raw-output.json").is_file())
            self.assertTrue((run_dir / "codex-command.json").is_file())
            self.assertTrue((run_dir / "edit-decisions.json").is_file())
            self.assertTrue((run_dir / "aggregate-gates.json").is_file())
            self.assertFalse((run_dir / "candidate.md").exists())
            self.assertFalse((run_dir / "record.json").exists())
            aggregate = json.loads((run_dir / "aggregate-gates.json").read_text(encoding="utf-8"))
            self.assertFalse(aggregate["hard_gates_passed"])
            self.assertIn("changed_line_budget", aggregate["aggregate_hard_gate_failures"])
            retry = mock.Mock()
            with self.assertRaisesRegex(RunnerError, "already attempted"):
                run_one_document(archive, "doc", "Prompt", config.sha256_text("Prompt"), phase, retry)
            retry.assert_not_called()

    def test_experiment_57_run_full_requires_evidence_bound_authorization_and_uses_fresh_calls(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            original = "문장이 조금 딱딱합니다.\n"
            archive = make_archive(Path(directory) / "archive", {
                f"{EXPERIMENT_57_PILOT_FILES['first']}.md": original,
                f"{EXPERIMENT_57_PILOT_FILES['second']}.md": original,
            }, experiment="57")
            prompt_sha = write_frozen_prompt_lock(archive, "Frozen prompt.\n")
            no_call = mock.Mock()
            with mock.patch.object(config, "EXPERIMENT_57_FROZEN_PROMPT_SHA256", prompt_sha):
                with self.assertRaisesRegex(RunnerError, "pilot-acceptance-gate"):
                    run_full(archive, no_call)
            no_call.assert_not_called()
            write_json_atomic(archive.root / "analysis" / "pilot-acceptance-gate.json", {
                "experiment_id": config.EXPERIMENT_57_ID,
                "status": "authorized_for_full",
            })
            with mock.patch.object(config, "EXPERIMENT_57_FROZEN_PROMPT_SHA256", prompt_sha):
                with self.assertRaisesRegex(RunnerError, "prompt SHA"):
                    run_full(archive, no_call)
            no_call.assert_not_called()
            first_record = write_success_record(
                archive,
                EXPERIMENT_57_PILOT_FILES["first"],
                ["pilot", "first", prompt_sha],
                prompt_sha=prompt_sha,
            )
            write_json_atomic(archive.root / "analysis" / "pilot-acceptance-gate.json", {
                "experiment_id": config.EXPERIMENT_57_ID,
                "status": "authorized_for_full",
                "instruction_prompt_sha256": prompt_sha,
                "fixed_pilot_pair": dict(EXPERIMENT_57_PILOT_FILES),
                "pilot_records": {
                    "first": {"path": archive.rel(first_record), "sha256": config.sha256_file(first_record)},
                },
            })
            with mock.patch.object(config, "EXPERIMENT_57_FROZEN_PROMPT_SHA256", prompt_sha):
                with self.assertRaisesRegex(RunnerError, "first and second"):
                    run_full(archive, no_call)
            second_record = write_success_record(
                archive,
                EXPERIMENT_57_PILOT_FILES["second"],
                ["pilot", "second", prompt_sha],
                prompt_sha=prompt_sha,
            )
            write_json_atomic(archive.root / "analysis" / "pilot-acceptance-gate.json", {
                "experiment_id": config.EXPERIMENT_57_ID,
                "status": "authorized_for_full",
                "instruction_prompt_sha256": prompt_sha,
                "fixed_pilot_pair": dict(EXPERIMENT_57_PILOT_FILES),
                "pilot_records": {
                    "first": {"path": archive.rel(first_record), "sha256": config.sha256_file(first_record)},
                    "second": {"path": archive.rel(second_record), "sha256": config.sha256_file(second_record)},
                },
            })
            calls: list[list[str]] = []

            def fake_run(command: list[str], input: str, text: bool, capture_output: bool) -> SimpleNamespace:
                calls.append(command)
                Path(command[command.index("-o") + 1]).write_text(json.dumps(response_for(original), ensure_ascii=False), encoding="utf-8")
                return SimpleNamespace(returncode=0, stdout="", stderr="")

            with mock.patch.object(config, "EXPERIMENT_57_FROZEN_PROMPT_SHA256", prompt_sha):
                records = run_full(archive, fake_run)
            self.assertEqual(len(records), 2)
            self.assertEqual(len(calls), 2)
            self.assertEqual(records[0]["experiment_id"], config.EXPERIMENT_57_ID)
            self.assertEqual(records[0]["phase"], ["full"])
            self.assertTrue((archive.root / "runs" / "full" / EXPERIMENT_57_PILOT_FILES["first"] / "record.json").is_file())

    def test_experiment_57_run_full_blocks_if_later_full_run_dir_exists_before_first_subprocess(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            original = "문장이 조금 딱딱합니다.\n"
            archive = make_archive(Path(directory) / "archive", {
                f"{EXPERIMENT_57_PILOT_FILES['first']}.md": original,
                f"{EXPERIMENT_57_PILOT_FILES['second']}.md": original,
            }, experiment="57")
            prompt_sha = write_frozen_prompt_lock(archive, "Frozen prompt.\n")
            first_record = write_success_record(
                archive,
                EXPERIMENT_57_PILOT_FILES["first"],
                ["pilot", "first", prompt_sha],
                prompt_sha=prompt_sha,
            )
            second_record = write_success_record(
                archive,
                EXPERIMENT_57_PILOT_FILES["second"],
                ["pilot", "second", prompt_sha],
                prompt_sha=prompt_sha,
            )
            write_json_atomic(archive.root / "analysis" / "pilot-acceptance-gate.json", {
                "experiment_id": config.EXPERIMENT_57_ID,
                "status": "authorized_for_full",
                "instruction_prompt_sha256": prompt_sha,
                "fixed_pilot_pair": dict(EXPERIMENT_57_PILOT_FILES),
                "pilot_records": {
                    "first": {"path": archive.rel(first_record), "sha256": config.sha256_file(first_record)},
                    "second": {"path": archive.rel(second_record), "sha256": config.sha256_file(second_record)},
                },
            })
            later_full_dir = archive.root / "runs" / "full" / EXPERIMENT_57_PILOT_FILES["second"]
            later_full_dir.mkdir(parents=True)
            subprocess_run = mock.Mock()
            with mock.patch.object(config, "EXPERIMENT_57_FROZEN_PROMPT_SHA256", prompt_sha):
                with self.assertRaisesRegex(RunnerError, "already attempted"):
                    run_full(archive, subprocess_run)
            subprocess_run.assert_not_called()
            first_full_dir = archive.root / "runs" / "full" / EXPERIMENT_57_PILOT_FILES["first"]
            self.assertFalse(first_full_dir.exists())

    def test_pilot_paths_are_prompt_sha_scoped_and_duplicate_blocks_before_model_call(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            original = "문장이 조금 딱딱합니다.\n"
            file_id = PILOT_FILES["first"]
            archive = make_archive(Path(directory) / "archive", {f"{file_id}.md": original})
            initial = write_archive_prompt(archive, "control/prompts/initial.md", "Initial prompt.\n")
            refined = write_archive_prompt(archive, "control/prompts/refined-01.md", "Refined prompt.\n")
            initial_sha = config.sha256_text("Initial prompt.\n")
            refined_sha = config.sha256_text("Refined prompt.\n")
            calls: list[str] = []
            bad = response_for(original)
            bad["edits"][0]["original_line"] = "다른 줄"

            def failing_run(command: list[str], input: str, text: bool, capture_output: bool) -> SimpleNamespace:
                calls.append("initial")
                Path(command[command.index("-o") + 1]).write_text(json.dumps(bad, ensure_ascii=False), encoding="utf-8")
                return SimpleNamespace(returncode=0, stdout="", stderr="")

            with self.assertRaisesRegex(ValueError, "byte-identical"):
                run_pilot(archive, initial, "first", failing_run)
            initial_raw = archive.root / "runs" / "pilot" / "first" / initial_sha / file_id / "raw-output.json"
            self.assertTrue(initial_raw.is_file())
            duplicate_run = mock.Mock()
            with self.assertRaisesRegex(RunnerError, "already attempted"):
                run_pilot(archive, initial, "first", duplicate_run)
            duplicate_run.assert_not_called()

            def refined_run(command: list[str], input: str, text: bool, capture_output: bool) -> SimpleNamespace:
                calls.append("refined")
                Path(command[command.index("-o") + 1]).write_text(json.dumps(response_for(original), ensure_ascii=False), encoding="utf-8")
                return SimpleNamespace(returncode=0, stdout="", stderr="")

            record = run_pilot(archive, refined, "first", refined_run)
            refined_raw = archive.root / "runs" / "pilot" / "first" / refined_sha / file_id / "raw-output.json"
            self.assertTrue(refined_raw.is_file())
            self.assertNotEqual(initial_raw.parent, refined_raw.parent)
            self.assertEqual(record["phase"], ["pilot", "first", refined_sha])
            self.assertEqual(calls, ["initial", "refined"])

    def test_freeze_prompt_requires_same_sha_success_records_and_writes_exact_metadata(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            archive = make_archive(Path(directory) / "archive", {
                f"{PILOT_FILES['first']}.md": "첫 문장입니다.\n",
                f"{PILOT_FILES['second']}.md": "둘째 문장입니다.\n",
            })
            initial = write_archive_prompt(archive, "control/prompts/initial.md", "Initial prompt.\n")
            prompt_sha = config.sha256_text("Initial prompt.\n")
            with self.assertRaisesRegex(RunnerError, "missing successful first pilot"):
                freeze_prompt(archive, initial)
            self.assertFalse((archive.root / "control" / "instruction-prompt.lock.md").exists())
            first_record = write_pilot_record(archive, prompt_sha, "first")
            with self.assertRaisesRegex(RunnerError, "missing successful second pilot"):
                freeze_prompt(archive, initial)
            write_pilot_record(archive, prompt_sha, "second", record_sha=config.sha256_text("Other prompt.\n"))
            with self.assertRaisesRegex(RunnerError, "instruction_prompt_sha256 mismatch"):
                freeze_prompt(archive, initial)
            write_pilot_record(archive, prompt_sha, "second", file_id="wrong-file")
            with self.assertRaisesRegex(RunnerError, "file_id mismatch"):
                freeze_prompt(archive, initial)
            second_record = write_pilot_record(archive, prompt_sha, "second")
            lock = freeze_prompt(archive, initial)
            self.assertEqual(set(lock), {
                "created_at_utc",
                "experiment_id",
                "instruction_prompt_path",
                "instruction_prompt_sha256",
                "model",
                "pilot_file_ids",
                "pilot_record_paths",
                "reasoning_effort",
                "refinement_count",
                "selected_prompt_source",
            })
            self.assertEqual(lock["experiment_id"], config.EXPERIMENT_ID)
            self.assertEqual(lock["instruction_prompt_sha256"], prompt_sha)
            self.assertEqual(lock["instruction_prompt_path"], "control/instruction-prompt.lock.md")
            self.assertEqual(lock["selected_prompt_source"], "control/prompts/initial.md")
            self.assertEqual(lock["pilot_file_ids"], PILOT_FILES)
            self.assertEqual(lock["pilot_record_paths"], {
                "first": archive.rel(first_record),
                "second": archive.rel(second_record),
            })
            self.assertEqual(lock["refinement_count"], 0)
            self.assertNotIn(str(archive.root), json.dumps(lock))
            self.assertEqual(config.sha256_file(archive.root / "control" / "instruction-prompt.lock.md"), prompt_sha)

    def test_freeze_prompt_accepts_only_initial_or_single_refined_archive_source(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            archive = make_archive(Path(directory) / "archive", {
                f"{PILOT_FILES['first']}.md": "첫 문장입니다.\n",
                f"{PILOT_FILES['second']}.md": "둘째 문장입니다.\n",
            })
            refined = write_archive_prompt(archive, "control/prompts/refined-01.md", "Refined prompt.\n")
            refined_sha = config.sha256_text("Refined prompt.\n")
            write_pilot_record(archive, refined_sha, "first")
            write_pilot_record(archive, refined_sha, "second")
            lock = freeze_prompt(archive, refined)
            self.assertEqual(lock["selected_prompt_source"], "control/prompts/refined-01.md")
            self.assertEqual(lock["refinement_count"], 1)
        with tempfile.TemporaryDirectory() as directory:
            archive = make_archive(Path(directory) / "archive", {
                f"{PILOT_FILES['first']}.md": "첫 문장입니다.\n",
                f"{PILOT_FILES['second']}.md": "둘째 문장입니다.\n",
            })
            bad_prompt = write_archive_prompt(archive, "control/prompts/refined-02.md", "Bad prompt.\n")
            with self.assertRaisesRegex(RunnerError, "initial.md or control/prompts/refined-01.md"):
                freeze_prompt(archive, bad_prompt)

    def test_codex_command_record_and_atomic_candidate_write_with_fake_boundary(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory) / "archive"
            original = "문장이 조금 딱딱합니다.\n"
            archive = make_archive(root, {"doc.md": original})
            calls: list[list[str]] = []

            def fake_run(command: list[str], input: str, text: bool, capture_output: bool) -> SimpleNamespace:
                calls.append(command)
                raw_path = Path(command[command.index("-o") + 1])
                raw_path.write_text(json.dumps(response_for(original), ensure_ascii=False), encoding="utf-8")
                return SimpleNamespace(returncode=0, stdout="ok", stderr="")

            prompt = "Return structured line edits only."
            record = run_one_document(
                archive,
                "doc",
                prompt,
                config.sha256_text(prompt),
                ("pilot", "first"),
                fake_run,
            )
            self.assertEqual(len(calls), 1)
            self.assertEqual(calls[0][:4], ["codex", "--sandbox", "read-only", "exec"])
            self.assertIn("-m", calls[0])
            self.assertIn(config.MODEL, calls[0])
            self.assertIn('model_reasoning_effort="medium"', calls[0])
            self.assertEqual(record["model"], config.MODEL)
            self.assertEqual(record["reasoning_effort"], config.REASONING_EFFORT)
            self.assertRegex(str(record["raw_output_sha256"]), r"^[0-9a-f]{64}$")
            self.assertTrue((archive.root / str(record["candidate_path"])).is_file())
            with self.assertRaisesRegex(RunnerError, "already attempted"):
                run_one_document(archive, "doc", prompt, config.sha256_text(prompt), ("pilot", "first"), fake_run)
            self.assertEqual(len(calls), 1)

    def test_nonzero_codex_attempt_preserves_ledger_and_blocks_retry_before_subprocess(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            archive = make_archive(Path(directory) / "archive", {"doc.md": "문장이 조금 딱딱합니다.\n"})
            calls = 0

            def failing_run(command: list[str], input: str, text: bool, capture_output: bool) -> SimpleNamespace:
                nonlocal calls
                calls += 1
                return SimpleNamespace(returncode=2, stdout="", stderr="failed")

            phase = ("pilot", "nonzero")
            run_dir = archive.root / "runs" / "pilot" / "nonzero" / "doc"
            with self.assertRaisesRegex(RunnerError, "codex command failed"):
                run_one_document(archive, "doc", "Prompt", config.sha256_text("Prompt"), phase, failing_run)
            self.assertEqual(calls, 1)
            self.assertTrue(run_dir.is_dir())
            self.assertTrue((run_dir / "rendered-prompt.txt").is_file())
            self.assertTrue((run_dir / "codex-command.json").is_file())
            self.assertFalse((run_dir / "raw-output.json").exists())
            retry = mock.Mock()
            with self.assertRaisesRegex(RunnerError, "already attempted"):
                run_one_document(archive, "doc", "Prompt", config.sha256_text("Prompt"), phase, retry)
            retry.assert_not_called()

    def test_candidate_is_not_written_on_line_or_hash_mismatch(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory) / "archive"
            original = "문장이 조금 딱딱합니다.\n"
            archive = make_archive(root, {"doc.md": original})
            bad = response_for(original)
            bad["edits"][0]["original_line"] = "다른 줄"

            def fake_run(command: list[str], input: str, text: bool, capture_output: bool) -> SimpleNamespace:
                Path(command[command.index("-o") + 1]).write_text(json.dumps(bad, ensure_ascii=False), encoding="utf-8")
                return SimpleNamespace(returncode=0, stdout="", stderr="")

            with self.assertRaisesRegex(ValueError, "byte-identical"):
                run_one_document(archive, "doc", "Prompt", config.sha256_text("Prompt"), ("pilot", "bad"), fake_run)
            self.assertFalse((archive.root / "runs" / "pilot" / "bad" / "doc" / "candidate.md").exists())

    def test_candidate_is_not_written_on_evidence_range_or_blank_safety(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            archive = make_archive(Path(directory) / "archive", {"doc.md": "문장이 조금 딱딱합니다.\n"})
            out_of_range = response_for("문장이 조금 딱딱합니다.\n")
            out_of_range["diagnoses"][0]["evidence_line_numbers"] = [2]

            def out_of_range_run(command: list[str], input: str, text: bool, capture_output: bool) -> SimpleNamespace:
                Path(command[command.index("-o") + 1]).write_text(json.dumps(out_of_range, ensure_ascii=False), encoding="utf-8")
                return SimpleNamespace(returncode=0, stdout="", stderr="")

            with self.assertRaisesRegex(ValueError, "evidence_line_numbers"):
                run_one_document(archive, "doc", "Prompt", config.sha256_text("Prompt"), ("pilot", "range"), out_of_range_run)
            self.assertFalse((archive.root / "runs" / "pilot" / "range" / "doc" / "candidate.md").exists())
            blank_safety = response_for("문장이 조금 딱딱합니다.\n")
            blank_safety["edits"][0]["safety_rationale"] = "   "

            def blank_safety_run(command: list[str], input: str, text: bool, capture_output: bool) -> SimpleNamespace:
                Path(command[command.index("-o") + 1]).write_text(json.dumps(blank_safety, ensure_ascii=False), encoding="utf-8")
                return SimpleNamespace(returncode=0, stdout="", stderr="")

            with self.assertRaisesRegex(ValueError, "safety_rationale"):
                run_one_document(archive, "doc", "Prompt", config.sha256_text("Prompt"), ("pilot", "safety"), blank_safety_run)
            self.assertFalse((archive.root / "runs" / "pilot" / "safety" / "doc" / "candidate.md").exists())

    def test_legacy_marker_gates_removed_but_line_locality_budget_retained(self) -> None:
        self.assertEqual([], guards.hard_gate_failures("검증한 문장입니다.\n", "살핀 문장입니다.\n"))
        failures = guards.hard_gate_failures("abcdefghij klmnopqrst.\n", "uvwxyzabcd efghijklmn.\n")
        self.assertIn("line_locality", failures)

    def test_source_bearing_bullet_prefixes_are_protected_without_source_false_positive(self) -> None:
        self.assertEqual(["source_bearing_lines"], guards.hard_gate_failures("- source: 값.\n", "- source: 말.\n"))
        self.assertEqual(["source_bearing_lines"], guards.hard_gate_failures("- 출처: 값.\n", "- 출처: 말.\n"))
        self.assertEqual([], guards.hard_gate_failures("- 항목은 딱딱합니다.\n", "- 항목은 딱딱해요.\n"))

    def test_hard_gates_report_each_protected_class_and_budget(self) -> None:
        cases = {
            "heading_lines": ("# Title\n\nBody.\n", "# Other\n\nBody.\n"),
            "line_count": ("Body.\n", "Body.\nExtra.\n"),
            "blank_line_positions": ("A.\n\nB.\n", "A.\nX.\nB.\n"),
            "nonempty_paragraph_shape": ("A.\n\nB.\n", "A.\nX.\nB.\n"),
            "sentence_terminators_per_line": ("A.\n", "A..\n"),
            "code_fence_blocks": ("```mermaid\ngraph TD\n```\n", "```mermaid\ngraph LR\n```\n"),
            "table_lines": ("| A | B |\n", "| A | C |\n"),
            "blockquote_lines": ("> quoted.\n", "> changed.\n"),
            "quoted_text": ("He said \"quote\".\n", "He said \"other\".\n"),
            "source_bearing_lines": ("Source: example\n", "Source: other\n"),
            "footnotes": ("Fact.[^1]\n\n[^1]: note\n", "Fact.[^2]\n\n[^2]: note\n"),
            "bracket_tokens": ("Fact [A].\n", "Fact [B].\n"),
            "links_urls": ("See [x](https://example.com/a).\n", "See [x](https://example.com/b).\n"),
            "inline_code": ("Use `code`.\n", "Use `code2`.\n"),
            "list_markers": ("- item.\n", "* item.\n"),
            "numbers_dates_percentages": ("On 2026-07-30, 50% passed.\n", "On 2026-07-31, 50% passed.\n"),
            "model_product_names": ("GPT-5.5 stays.\n", "GPT-4.1 stays.\n"),
            "latin_technical_tokens": ("ABC123 token.\n", "XYZ123 token.\n"),
            "markdown_structure_tokens": ("plain word.\n", "**plain** word.\n"),
            "changed_line_budget": ("첫 문장입니다.\n둘째 문장입니다.\n", "첫 문장이에요.\n둘째 문장이에요.\n"),
            "changed_text_budget": ("가" * 6000 + ".\n", "나" * 6000 + ".\n"),
            "line_locality": ("abcdefghij klmnopqrst.\n", "uvwxyzabcd efghijklmn.\n"),
        }
        for reason, (original, candidate) in cases.items():
            with self.subTest(reason=reason):
                self.assertIn(reason, guards.hard_gate_failures(original, candidate))

    def test_blind_packets_require_8_passes_and_ties_count_as_candidate_losses(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory) / "archive"
            contents = {f"doc-{index}.md": f"원문 {index}.\n" for index in range(8)}
            archive = make_archive(root, contents)
            with self.assertRaisesRegex(BlindError, "8 hard-gate pass"):
                make_blind_packets(archive, seed=1)
            for filename in contents:
                file_id = filename[:-3]
                write_success_record(archive, file_id, ["full"])
            result = make_blind_packets(archive, seed=7)
            self.assertEqual(result["packet_count"], 8)
            packet_text = (archive.root / "blind" / "packets" / "packet-01.json").read_text(encoding="utf-8")
            self.assertNotIn("source_filename", packet_text)
            self.assertNotIn("candidate_slot", packet_text)
            verdicts_path = archive.root / "blind" / "host-verdicts-input.json"
            write_json_atomic(verdicts_path, {
                "verdicts": [
                    {"packet_id": f"packet-{index:02d}", "choice": "tie", "rationale": "no clear preference"}
                    for index in range(1, 9)
                ]
            })
            lock = record_host_verdicts(archive, verdicts_path)
            self.assertEqual(lock["locked_verdicts"], 8)
            public = export_public_summary(archive)
            self.assertEqual(public["candidate_wins"], 0)
            self.assertEqual(public["candidate_losses"], 8)
            self.assertEqual(public["ties"], 8)

    def test_blind_and_summary_records_use_selected_experiment_identity_and_lock_names(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory) / "archive"
            contents = {f"doc-{index}.md": f"원문 {index}.\n" for index in range(8)}
            archive = make_archive(root, contents, experiment="57")
            for filename in contents:
                file_id = filename[:-3]
                write_success_record(archive, file_id, ["full"], prompt_sha=config.EXPERIMENT_57_FROZEN_PROMPT_SHA256)
            packets = make_blind_packets(archive, seed=11)
            self.assertEqual(packets["mapping_path"], "blind/private-mapping.lock.json")
            mapping = json.loads((archive.root / "blind" / "private-mapping.lock.json").read_text(encoding="utf-8"))
            self.assertEqual(mapping["experiment_id"], config.EXPERIMENT_57_ID)
            verdicts_path = archive.root / "blind" / "host-verdicts.input.json"
            write_json_atomic(verdicts_path, {
                "verdicts": [
                    {"packet_id": f"packet-{index:02d}", "choice": "tie", "rationale": "no clear preference"}
                    for index in range(1, 9)
                ]
            })
            verdict_lock = record_host_verdicts(archive, verdicts_path)
            self.assertEqual(verdict_lock["path"], "blind/host-verdicts.lock.json")
            locked = json.loads((archive.root / "blind" / "host-verdicts.lock.json").read_text(encoding="utf-8"))
            self.assertEqual(locked["experiment_id"], config.EXPERIMENT_57_ID)
            public = export_public_summary(archive)
            self.assertEqual(public["experiment_id"], config.EXPERIMENT_57_ID)
            self.assertTrue((archive.root / "analysis" / "public-summary.json").is_file())
            self.assertEqual(
                sorted(path.name for path in (archive.root / "blind").iterdir()),
                ["host-verdicts.input.json", "host-verdicts.lock.json", "packets", "private-mapping.lock.json"],
            )
            self.assertEqual(sorted(path.name for path in (archive.root / "analysis").iterdir()), ["public-summary.json"])

    def test_verdict_lock_does_not_read_private_mapping_and_summary_validates_it(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            archive = make_archive(
                Path(directory) / "archive",
                {f"doc-{index}.md": f"원문 {index}.\n" for index in range(8)},
            )
            for file_id in archive.expected_file_ids():
                write_success_record(archive, file_id, ["full"])
            make_blind_packets(archive, seed=5)
            verdicts_path = archive.root / "blind" / "host-verdicts.input.json"
            write_json_atomic(verdicts_path, {
                "verdicts": [
                    {"packet_id": f"packet-{index:02d}", "choice": "tie", "rationale": "no clear preference"}
                    for index in range(1, 9)
                ]
            })
            with mock.patch(
                "report_natural_voice_correction.blind.validate_private_mapping_packets",
                side_effect=AssertionError("private mapping must stay sealed"),
            ):
                self.assertEqual(record_host_verdicts(archive, verdicts_path)["locked_verdicts"], 8)

            mapping_path = archive.root / "blind" / "private-mapping.lock.json"
            mapping = json.loads(mapping_path.read_text(encoding="utf-8"))
            mapping["mappings"][0]["candidate_slot"] = mapping["mappings"][0]["original_slot"]
            write_json_atomic(mapping_path, mapping)
            with self.assertRaisesRegex(BlindError, "slots must be opposite"):
                export_public_summary(archive)

    def test_blind_full_records_reject_wrong_experiment_and_stale_candidate(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            archive = make_archive(Path(directory) / "archive", {f"doc-{index}.md": f"원문 {index}.\n" for index in range(8)})
            for file_id in archive.expected_file_ids():
                write_success_record(archive, file_id, ["full"])
            first = archive.expected_file_ids()[0]
            record_path = archive.root / "runs" / "full" / first / "record.json"
            record = json.loads(record_path.read_text(encoding="utf-8"))
            record["experiment_id"] = config.EXPERIMENT_57_ID
            write_json_atomic(record_path, record)
            with self.assertRaisesRegex(BlindError, "valid full"):
                make_blind_packets(archive, seed=1)
        with tempfile.TemporaryDirectory() as directory:
            archive = make_archive(Path(directory) / "archive", {f"doc-{index}.md": f"원문 {index}.\n" for index in range(8)})
            for file_id in archive.expected_file_ids():
                write_success_record(archive, file_id, ["full"])
            first = archive.expected_file_ids()[0]
            candidate = archive.root / "runs" / "full" / first / "candidate.md"
            candidate.write_text("tampered candidate\n", encoding="utf-8")
            with self.assertRaisesRegex(BlindError, "valid full"):
                make_blind_packets(archive, seed=1)

    def test_blind_packet_hash_tamper_is_detected_when_private_mapping_is_opened(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            archive = make_archive(Path(directory) / "archive", {f"doc-{index}.md": f"원문 {index}.\n" for index in range(8)})
            for file_id in archive.expected_file_ids():
                write_success_record(archive, file_id, ["full"])
            make_blind_packets(archive, seed=2)
            packet_path = archive.root / "blind" / "packets" / "packet-01.json"
            packet = json.loads(packet_path.read_text(encoding="utf-8"))
            packet["documents"][0]["body"] += "tampered"
            write_json_atomic(packet_path, packet)
            verdicts_path = archive.root / "blind" / "host-verdicts.input.json"
            write_json_atomic(verdicts_path, {
                "verdicts": [
                    {"packet_id": f"packet-{index:02d}", "choice": "tie", "rationale": "no clear preference"}
                    for index in range(1, 9)
                ]
            })
            self.assertEqual(record_host_verdicts(archive, verdicts_path)["locked_verdicts"], 8)
            with self.assertRaisesRegex(BlindError, "packet hash"):
                export_public_summary(archive)


if __name__ == "__main__":
    unittest.main()
