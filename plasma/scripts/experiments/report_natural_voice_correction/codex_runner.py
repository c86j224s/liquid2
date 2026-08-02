from __future__ import annotations

import json
from pathlib import Path
import subprocess
import time
from typing import Any, Callable

from . import config
from .archive import ExperimentArchive, read_json, write_json_atomic, write_text_atomic
from .edits import (
    LineEdit,
    StructuredResponse,
    apply_response,
    candidate_for_edits,
    parse_model_response,
    response_schema,
    split_document_lines,
)
from .guards import HARD_GATE_REASON_CODES, hard_gate_failures, validate_hard_gates


PILOT_FILES = {
    "first": "01-wang-anshi-exploratory-read-A",
    "second": "04-go-raft-strict-read-B",
}
EXPERIMENT_57_PILOT_FILES = {
    "first": "02-wang-anshi-strict-read-A",
    "second": "03-go-raft-exploratory-read-B",
}
PILOT_FILES_BY_EXPERIMENT = {
    "56": PILOT_FILES,
    "57": EXPERIMENT_57_PILOT_FILES,
}
FREEZABLE_PROMPT_SOURCES = {
    Path("control/prompts/initial.md"): 0,
    Path("control/prompts/refined-01.md"): 1,
}


class RunnerError(ValueError):
    pass


def lint_prompt_file(path: Path) -> dict[str, object]:
    resolved = Path(path).expanduser().resolve()
    if not resolved.is_file():
        raise RunnerError("prompt path must be an existing file")
    if config.is_relative_to(resolved, config.repo_root()):
        raise RunnerError("prompt file must stay outside the Git worktree")
    text = resolved.read_text(encoding="utf-8")
    if not text.strip():
        raise RunnerError("prompt must not be blank")
    forbidden = ["/invalid/", "rewritten_document", "AI detector", "ai detector", "detector score"]
    forbidden.extend(config.EXPECTED_SHA256_BY_FILENAME)
    for token in forbidden:
        if token in text:
            raise RunnerError(f"prompt contains forbidden token: {token}")
    return {"path": str(resolved), "sha256": config.sha256_text(text), "bytes": len(text.encode("utf-8"))}


def freeze_prompt(archive: ExperimentArchive, prompt_path: Path) -> dict[str, object]:
    if archive.experiment == "57":
        raise RunnerError("freeze-prompt is not allowed for experiment 57")
    linted = lint_prompt_file(prompt_path)
    selected_source, refinement_count = _freeze_prompt_source(archive, Path(str(linted["path"])))
    text = Path(str(linted["path"])).read_text(encoding="utf-8")
    prompt_sha = str(linted["sha256"])
    pilot_record_paths = _validate_freeze_pilot_records(archive, prompt_sha)
    prompt_lock = archive.root / "control" / "prompt-freeze.lock.json"
    prompt_body_path = archive.root / "control" / "instruction-prompt.lock.md"
    if prompt_lock.exists():
        existing = read_json(prompt_lock)
        _validate_existing_freeze_lock(archive, existing, prompt_sha, selected_source, pilot_record_paths, refinement_count)
        return existing
    write_text_atomic(prompt_body_path, text)
    if config.sha256_file(prompt_body_path) != prompt_sha:
        raise RunnerError("frozen prompt body SHA-256 mismatch after write")
    lock = {
        "experiment_id": archive.experiment_id,
        "instruction_prompt_sha256": prompt_sha,
        "instruction_prompt_path": archive.rel(prompt_body_path),
        "selected_prompt_source": selected_source,
        "model": config.MODEL,
        "reasoning_effort": config.REASONING_EFFORT,
        "pilot_file_ids": _pilot_files(archive),
        "pilot_record_paths": pilot_record_paths,
        "refinement_count": refinement_count,
        "created_at_utc": _utc_now(),
    }
    write_json_atomic(prompt_lock, lock)
    return lock


def load_prompt(archive: ExperimentArchive, prompt_path: Path | None) -> tuple[str, str]:
    if prompt_path is not None:
        if archive.experiment == "57":
            raise RunnerError("experiment 57 uses only the frozen prompt; prompt overrides are not allowed")
        linted = lint_prompt_file(prompt_path)
        text = Path(str(linted["path"])).read_text(encoding="utf-8")
        return text, str(linted["sha256"])
    lock_path = archive.root / "control" / "prompt-freeze.lock.json"
    if not lock_path.is_file():
        raise RunnerError("frozen prompt lock is required")
    lock = read_json(lock_path)
    if archive.experiment == "57":
        return _load_experiment_57_prompt(archive, lock)
    return _load_experiment_56_prompt(archive, lock)


def _load_experiment_56_prompt(archive: ExperimentArchive, lock: dict[str, object]) -> tuple[str, str]:
    locked_experiment = lock.get("experiment_id")
    if locked_experiment is not None and locked_experiment != archive.experiment_id:
        raise RunnerError("frozen prompt lock experiment_id mismatch")
    prompt_path = archive.root / str(lock["instruction_prompt_path"])
    text = prompt_path.read_text(encoding="utf-8")
    sha = config.sha256_text(text)
    if sha != lock.get("instruction_prompt_sha256"):
        raise RunnerError("frozen prompt SHA-256 mismatch")
    return text, sha


def _load_experiment_57_prompt(archive: ExperimentArchive, lock: dict[str, object]) -> tuple[str, str]:
    expected_keys = {
        "corpus_hashes",
        "experiment_id",
        "fixed_pilot_pair",
        "inherited_from_experiment_id",
        "instruction_prompt_lock_path",
        "lock_kind",
        "locked_at_utc",
        "model",
        "no_policy_iteration_in_experiment_57",
        "no_prompt_iteration_in_experiment_57",
        "prompt_path",
        "prompt_sha256",
        "reasoning_effort",
        "refinement_history",
        "selected_prompt_source",
        "status",
    }
    if set(lock) != expected_keys:
        raise RunnerError("experiment 57 frozen prompt lock schema mismatch")
    if lock.get("experiment_id") != config.EXPERIMENT_57_ID:
        raise RunnerError("experiment 57 frozen prompt lock experiment_id mismatch")
    if lock.get("status") != "frozen_before_any_experiment_57_model_call":
        raise RunnerError("experiment 57 frozen prompt lock status mismatch")
    if lock.get("prompt_sha256") != config.EXPERIMENT_57_FROZEN_PROMPT_SHA256:
        raise RunnerError("experiment 57 frozen prompt lock prompt_sha256 mismatch")
    if lock.get("model") != config.MODEL or lock.get("reasoning_effort") != config.REASONING_EFFORT:
        raise RunnerError("experiment 57 frozen prompt lock model or reasoning_effort mismatch")
    if lock.get("fixed_pilot_pair") != EXPERIMENT_57_PILOT_FILES:
        raise RunnerError("experiment 57 frozen prompt lock fixed_pilot_pair mismatch")
    refinement = lock.get("refinement_history")
    if not isinstance(refinement, dict):
        raise RunnerError("experiment 57 frozen prompt lock refinement_history mismatch")
    if refinement.get("inherited_refinement_count") != 1:
        raise RunnerError("experiment 57 inherited_refinement_count mismatch")
    if refinement.get("additional_experiment_57_refinement_count") != 0:
        raise RunnerError("experiment 57 additional refinement count mismatch")
    if lock.get("no_prompt_iteration_in_experiment_57") is not True:
        raise RunnerError("experiment 57 no_prompt_iteration flag mismatch")
    if lock.get("no_policy_iteration_in_experiment_57") is not True:
        raise RunnerError("experiment 57 no_policy_iteration flag mismatch")
    prompt_source_path = _archive_relative_file(archive, lock.get("prompt_path"), "prompt_path")
    prompt_lock_path = _archive_relative_file(archive, lock.get("instruction_prompt_lock_path"), "instruction_prompt_lock_path")
    if config.sha256_file(prompt_source_path) != config.EXPERIMENT_57_FROZEN_PROMPT_SHA256:
        raise RunnerError("experiment 57 prompt_path SHA-256 mismatch")
    if config.sha256_file(prompt_lock_path) != config.EXPERIMENT_57_FROZEN_PROMPT_SHA256:
        raise RunnerError("experiment 57 frozen prompt SHA-256 mismatch")
    text = prompt_lock_path.read_text(encoding="utf-8")
    return text, config.EXPERIMENT_57_FROZEN_PROMPT_SHA256


def render_prompt(instruction_prompt: str, file_id: str, document_text: str) -> str:
    lines = [
        instruction_prompt.rstrip(),
        "",
        "SEALED_DOCUMENT",
        f"file_id: {file_id}",
        f"document_sha256: {config.sha256_text(document_text)}",
        "numbered_lines:",
    ]
    for line_number, line in enumerate(split_document_lines(document_text), 1):
        lines.append(f"{line_number}\t{config.sha256_text(line)}\t{line}")
    return "\n".join(lines) + "\n"


def codex_command(archive: ExperimentArchive, raw_output_path: Path, schema_path: Path) -> list[str]:
    return [
        "codex",
        "--sandbox",
        "read-only",
        "exec",
        "--ephemeral",
        "--ignore-user-config",
        "--ignore-rules",
        "--skip-git-repo-check",
        "-C",
        str(archive.root / "tmp-harness"),
        "-m",
        config.MODEL,
        "-c",
        'model_reasoning_effort="medium"',
        "-o",
        str(raw_output_path),
        "--output-schema",
        str(schema_path),
        "-",
    ]


def run_pilot(
    archive: ExperimentArchive,
    prompt_path: Path | None,
    pilot_step: str,
    subprocess_run: Callable[..., Any] = subprocess.run,
) -> dict[str, object]:
    pilot_files = _pilot_files(archive)
    if pilot_step not in pilot_files:
        raise RunnerError("pilot_step must be first or second")
    if archive.experiment == "57" and prompt_path is not None:
        raise RunnerError("experiment 57 run-pilot uses the frozen prompt; --prompt is not allowed")
    prompt, prompt_sha = load_prompt(archive, prompt_path)
    return run_one_document(archive, pilot_files[pilot_step], prompt, prompt_sha, ("pilot", pilot_step, prompt_sha), subprocess_run)


def run_full(
    archive: ExperimentArchive,
    subprocess_run: Callable[..., Any] = subprocess.run,
) -> list[dict[str, object]]:
    prompt, prompt_sha = load_prompt(archive, None)
    if archive.experiment == "57":
        _validate_full_authorization(archive, prompt_sha)
        _validate_fresh_full_run_set(archive)
    return [
        run_one_document(archive, file_id, prompt, prompt_sha, ("full",), subprocess_run)
        for file_id in archive.expected_file_ids()
    ]


def run_one_document(
    archive: ExperimentArchive,
    file_id: str,
    instruction_prompt: str,
    instruction_prompt_sha: str,
    phase: tuple[str, ...],
    subprocess_run: Callable[..., Any] = subprocess.run,
) -> dict[str, object]:
    archive.verify_source_seal()
    archive.ensure_layout()
    document_path = archive.input_path_for_file_id(file_id)
    document_text = document_path.read_text(encoding="utf-8")
    document_sha = config.sha256_text(document_text)
    run_dir = archive.root / "runs" / Path(*phase) / file_id
    record_path = run_dir / "record.json"
    raw_output_path = run_dir / "raw-output.json"
    candidate_path = run_dir / "candidate.md"
    edit_decisions_path = run_dir / "edit-decisions.json"
    aggregate_gates_path = run_dir / "aggregate-gates.json"
    if run_dir.exists():
        raise RunnerError(f"model call already attempted for {file_id} in {'/'.join(phase)}")
    if (
        record_path.exists()
        or raw_output_path.exists()
        or candidate_path.exists()
        or edit_decisions_path.exists()
        or aggregate_gates_path.exists()
    ):
        raise RunnerError(f"model call already recorded for {file_id} in {'/'.join(phase)}")
    run_dir.mkdir(parents=True, exist_ok=False)
    schema_path = ensure_response_schema(archive)
    rendered = render_prompt(instruction_prompt, file_id, document_text)
    rendered_path = run_dir / "rendered-prompt.txt"
    write_text_atomic(rendered_path, rendered)
    command = codex_command(archive, raw_output_path, schema_path)
    started = time.time()
    process = subprocess_run(command, input=rendered, text=True, capture_output=True)
    command_path = run_dir / "codex-command.json"
    command_record = {
        "command": command,
        "returncode": int(getattr(process, "returncode", 1)),
        "stdout": str(getattr(process, "stdout", "")),
        "stderr": str(getattr(process, "stderr", "")),
        "duration_seconds": round(time.time() - started, 3),
    }
    write_json_atomic(command_path, command_record)
    if command_record["returncode"] != 0:
        raise RunnerError("codex command failed")
    if not raw_output_path.is_file():
        raise RunnerError("codex did not create the raw output file")
    raw_text = raw_output_path.read_text(encoding="utf-8")
    response = parse_model_response(raw_text)
    policy_fields: dict[str, object] = {}
    if archive.experiment == "57":
        candidate_text, policy_fields = _run_selective_acceptance_policy(archive, run_dir, file_id, document_text, response)
    else:
        candidate_text = apply_response(document_text, response)
        validate_hard_gates(document_text, candidate_text)
    write_text_atomic(candidate_path, candidate_text)
    record = {
        "experiment_id": archive.experiment_id,
        "phase": list(phase),
        "file_id": file_id,
        "source_filename": document_path.name,
        "document_sha256": document_sha,
        "instruction_prompt_sha256": instruction_prompt_sha,
        "rendered_prompt_sha256": config.sha256_text(rendered),
        "input_sha256": config.sha256_text(rendered),
        "model": config.MODEL,
        "reasoning_effort": config.REASONING_EFFORT,
        "raw_output_path": archive.rel(raw_output_path),
        "raw_output_sha256": config.sha256_text(raw_text),
        "candidate_path": archive.rel(candidate_path),
        "candidate_sha256": config.sha256_text(candidate_text),
        "hard_gates_passed": True,
        "codex_command_path": archive.rel(command_path),
        "codex_command_sha256": config.sha256_file(command_path),
        "rendered_prompt_path": archive.rel(rendered_path),
    }
    record.update(policy_fields)
    write_json_atomic(record_path, record)
    return record


def ensure_response_schema(archive: ExperimentArchive) -> Path:
    schema_path = archive.root / "control" / "structured-response.schema.json"
    schema_text = json.dumps(response_schema(), ensure_ascii=False, indent=2, sort_keys=True) + "\n"
    if schema_path.exists() and schema_path.read_text(encoding="utf-8") != schema_text:
        raise RunnerError("existing response schema does not match runner schema")
    if not schema_path.exists():
        write_text_atomic(schema_path, schema_text)
    return schema_path


def _freeze_prompt_source(archive: ExperimentArchive, prompt_path: Path) -> tuple[str, int]:
    try:
        relative = prompt_path.resolve().relative_to(archive.root)
    except ValueError as exc:
        raise RunnerError("freeze prompt must be inside the archive") from exc
    if relative not in FREEZABLE_PROMPT_SOURCES:
        raise RunnerError("freeze prompt must be control/prompts/initial.md or control/prompts/refined-01.md")
    return relative.as_posix(), FREEZABLE_PROMPT_SOURCES[relative]


def _validate_freeze_pilot_records(archive: ExperimentArchive, prompt_sha: str) -> dict[str, str]:
    record_paths: dict[str, str] = {}
    for step, file_id in _pilot_files(archive).items():
        record_path = archive.root / "runs" / "pilot" / step / prompt_sha / file_id / "record.json"
        if not record_path.is_file():
            raise RunnerError(f"missing successful {step} pilot record for prompt SHA")
        record = read_json(record_path)
        if record.get("hard_gates_passed") is not True:
            raise RunnerError(f"{step} pilot record did not pass hard gates")
        if record.get("instruction_prompt_sha256") != prompt_sha:
            raise RunnerError(f"{step} pilot record instruction_prompt_sha256 mismatch")
        if record.get("file_id") != file_id:
            raise RunnerError(f"{step} pilot record file_id mismatch")
        record_paths[step] = archive.rel(record_path)
    return record_paths


def _validate_existing_freeze_lock(
    archive: ExperimentArchive,
    existing: dict[str, object],
    prompt_sha: str,
    selected_source: str,
    pilot_record_paths: dict[str, str],
    refinement_count: int,
) -> None:
    expected_keys = {
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
    }
    if set(existing) != expected_keys:
        raise RunnerError("existing prompt freeze lock metadata is incomplete")
    if existing.get("experiment_id") != archive.experiment_id:
        raise RunnerError("existing prompt freeze lock experiment_id mismatch")
    if existing.get("instruction_prompt_sha256") != prompt_sha:
        raise RunnerError("prompt freeze already exists with a different SHA-256")
    if existing.get("instruction_prompt_path") != "control/instruction-prompt.lock.md":
        raise RunnerError("existing prompt freeze lock instruction_prompt_path mismatch")
    if existing.get("selected_prompt_source") != selected_source:
        raise RunnerError("prompt freeze already exists with a different selected prompt source")
    if existing.get("model") != config.MODEL:
        raise RunnerError("existing prompt freeze lock model mismatch")
    if existing.get("reasoning_effort") != config.REASONING_EFFORT:
        raise RunnerError("existing prompt freeze lock reasoning_effort mismatch")
    if existing.get("pilot_file_ids") != _pilot_files(archive):
        raise RunnerError("existing prompt freeze lock pilot_file_ids mismatch")
    if existing.get("pilot_record_paths") != pilot_record_paths:
        raise RunnerError("prompt freeze already exists with different pilot records")
    if existing.get("refinement_count") != refinement_count:
        raise RunnerError("prompt freeze already exists with a different refinement_count")
    prompt_body_path = archive.root / str(existing.get("instruction_prompt_path"))
    if not prompt_body_path.is_file() or config.sha256_file(prompt_body_path) != prompt_sha:
        raise RunnerError("existing frozen prompt body SHA-256 mismatch")


def _utc_now() -> str:
    return time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())


def _pilot_files(archive: ExperimentArchive) -> dict[str, str]:
    return dict(PILOT_FILES_BY_EXPERIMENT[archive.experiment])


def _validate_full_authorization(archive: ExperimentArchive, prompt_sha: str) -> None:
    gate_path = archive.root / "analysis" / "pilot-acceptance-gate.json"
    if not gate_path.is_file():
        raise RunnerError("experiment 57 run-full requires analysis/pilot-acceptance-gate.json")
    gate = read_json(gate_path)
    if gate.get("experiment_id") != config.EXPERIMENT_57_ID or gate.get("status") != "authorized_for_full":
        raise RunnerError("experiment 57 pilot acceptance gate is not authorized_for_full")
    if gate.get("instruction_prompt_sha256") != prompt_sha:
        raise RunnerError("experiment 57 pilot acceptance gate prompt SHA mismatch")
    if gate.get("fixed_pilot_pair") != EXPERIMENT_57_PILOT_FILES:
        raise RunnerError("experiment 57 pilot acceptance gate fixed_pilot_pair mismatch")
    pilot_records = gate.get("pilot_records")
    if not isinstance(pilot_records, dict) or set(pilot_records) != {"first", "second"}:
        raise RunnerError("experiment 57 pilot acceptance gate must include first and second pilot_records")
    for step, file_id in EXPERIMENT_57_PILOT_FILES.items():
        entry = pilot_records[step]
        if not isinstance(entry, dict) or set(entry) != {"path", "sha256"}:
            raise RunnerError("experiment 57 pilot record gate entries must contain path and sha256 only")
        expected_path = f"runs/pilot/{step}/{prompt_sha}/{file_id}/record.json"
        if entry.get("path") != expected_path:
            raise RunnerError(f"experiment 57 {step} pilot record path mismatch")
        record_path = archive.root / expected_path
        if not record_path.is_file() or config.sha256_file(record_path) != entry.get("sha256"):
            raise RunnerError(f"experiment 57 {step} pilot record SHA-256 mismatch")
        record = read_json(record_path)
        validate_success_record_artifacts(
            archive,
            record,
            ["pilot", step, prompt_sha],
            file_id,
            prompt_sha,
            require_selective_artifacts=True,
        )


def _validate_fresh_full_run_set(archive: ExperimentArchive) -> None:
    artifact_names = (
        "record.json",
        "raw-output.json",
        "candidate.md",
        "edit-decisions.json",
        "aggregate-gates.json",
        "rendered-prompt.txt",
        "codex-command.json",
    )
    for file_id in archive.expected_file_ids():
        run_dir = archive.root / "runs" / "full" / file_id
        if run_dir.exists() or any((run_dir / name).exists() for name in artifact_names):
            raise RunnerError(f"experiment 57 full run already attempted for {file_id}")


def validate_success_record_artifacts(
    archive: ExperimentArchive,
    record: dict[str, object],
    expected_phase: list[str],
    expected_file_id: str,
    prompt_sha: str | None = None,
    require_selective_artifacts: bool | None = None,
) -> None:
    expected_filename = archive.filename_for_file_id(expected_file_id)
    document_path = archive.input_path_for_file_id(expected_file_id)
    document_text = document_path.read_text(encoding="utf-8")
    if record.get("experiment_id") != archive.experiment_id:
        raise RunnerError("run record experiment_id mismatch")
    if record.get("phase") != expected_phase:
        raise RunnerError("run record phase mismatch")
    if record.get("file_id") != expected_file_id or record.get("source_filename") != expected_filename:
        raise RunnerError("run record source identity mismatch")
    if record.get("document_sha256") != config.sha256_text(document_text):
        raise RunnerError("run record source document SHA-256 mismatch")
    expected_prompt_sha = prompt_sha or (config.EXPERIMENT_57_FROZEN_PROMPT_SHA256 if archive.experiment == "57" else None)
    if expected_prompt_sha is not None and record.get("instruction_prompt_sha256") != expected_prompt_sha:
        raise RunnerError("run record instruction_prompt_sha256 mismatch")
    if record.get("model") != config.MODEL or record.get("reasoning_effort") != config.REASONING_EFFORT:
        raise RunnerError("run record model or reasoning_effort mismatch")
    if record.get("hard_gates_passed") is not True:
        raise RunnerError("run record hard_gates_passed mismatch")
    run_dir = Path("runs") / Path(*expected_phase) / expected_file_id
    candidate_path = _validate_hashed_artifact(archive, record, "candidate_path", "candidate_sha256", run_dir / "candidate.md")
    raw_output_path = _validate_hashed_artifact(archive, record, "raw_output_path", "raw_output_sha256", run_dir / "raw-output.json")
    command_path = _validate_hashed_artifact(archive, record, "codex_command_path", "codex_command_sha256", run_dir / "codex-command.json")
    rendered_path = _validate_hashed_artifact(archive, record, "rendered_prompt_path", "rendered_prompt_sha256", run_dir / "rendered-prompt.txt")
    if not raw_output_path.is_file() or not command_path.is_file() or not rendered_path.is_file():
        raise RunnerError("run record required artifact is missing")
    candidate_text = candidate_path.read_text(encoding="utf-8")
    if hard_gate_failures(document_text, candidate_text):
        raise RunnerError("run record candidate no longer passes aggregate hard gates")
    needs_selective = archive.experiment == "57" if require_selective_artifacts is None else require_selective_artifacts
    if needs_selective:
        _validate_selective_record_artifacts(archive, record, run_dir)


def _validate_selective_record_artifacts(archive: ExperimentArchive, record: dict[str, object], run_dir: Path) -> None:
    if record.get("aggregate_hard_gate_failures") != []:
        raise RunnerError("run record aggregate hard gate failures mismatch")
    proposed = record.get("proposed_edit_count")
    accepted = record.get("accepted_edit_count")
    rejected = record.get("rejected_edit_count")
    if not isinstance(proposed, int) or not isinstance(accepted, int) or not isinstance(rejected, int):
        raise RunnerError("run record edit counts must be integers")
    if proposed != accepted + rejected:
        raise RunnerError("run record edit counts must satisfy proposed = accepted + rejected")
    decisions_path = _validate_hashed_artifact(
        archive,
        record,
        "edit_decisions_path",
        "edit_decisions_sha256",
        run_dir / "edit-decisions.json",
    )
    aggregate_path = _validate_hashed_artifact(
        archive,
        record,
        "aggregate_gates_path",
        "aggregate_gates_sha256",
        run_dir / "aggregate-gates.json",
    )
    decisions = read_json(decisions_path)
    _validate_decisions_artifact_payload(decisions)
    if decisions.get("experiment_id") != archive.experiment_id:
        raise RunnerError("edit decisions experiment_id mismatch")
    if decisions.get("proposed_count") != proposed or decisions.get("accepted_count") != accepted or decisions.get("rejected_count") != rejected:
        raise RunnerError("edit decisions counts mismatch")
    aggregate = read_json(aggregate_path)
    if aggregate.get("experiment_id") != archive.experiment_id:
        raise RunnerError("aggregate gates experiment_id mismatch")
    if aggregate.get("hard_gates_passed") is not True or aggregate.get("aggregate_hard_gate_failures") != []:
        raise RunnerError("aggregate gates did not pass")
    if aggregate.get("proposed_count") != proposed or aggregate.get("accepted_count") != accepted or aggregate.get("rejected_count") != rejected:
        raise RunnerError("aggregate gates counts mismatch")


def _validate_decisions_artifact_payload(payload: dict[str, object]) -> None:
    decisions = payload.get("decisions")
    proposed = payload.get("proposed_count")
    accepted = payload.get("accepted_count")
    rejected = payload.get("rejected_count")
    if not isinstance(decisions, list):
        raise RunnerError("edit decisions artifact must contain a decisions array")
    if not isinstance(proposed, int) or not isinstance(accepted, int) or not isinstance(rejected, int):
        raise RunnerError("edit decisions artifact counts must be integers")
    if proposed != accepted + rejected or proposed != len(decisions):
        raise RunnerError("edit decisions artifact counts are inconsistent")
    seen_lines: set[int] = set()
    accepted_seen = 0
    rejected_seen = 0
    for item in decisions:
        if not isinstance(item, dict) or set(item) != {"line_number", "category", "disposition", "reason_codes"}:
            raise RunnerError("edit decisions artifact entry schema mismatch")
        line_number = item["line_number"]
        disposition = item["disposition"]
        reason_codes = item["reason_codes"]
        if not isinstance(line_number, int) or line_number in seen_lines:
            raise RunnerError("edit decisions artifact line_number mismatch")
        if disposition not in {"accepted", "rejected"}:
            raise RunnerError("edit decisions artifact disposition mismatch")
        if not isinstance(reason_codes, list) or any(not isinstance(reason, str) for reason in reason_codes):
            raise RunnerError("edit decisions artifact reason_codes mismatch")
        if any(reason not in set(HARD_GATE_REASON_CODES) for reason in reason_codes):
            raise RunnerError("edit decisions artifact unknown reason code")
        if disposition == "accepted":
            if reason_codes:
                raise RunnerError("accepted edit decisions artifact reason_codes must be empty")
            accepted_seen += 1
        else:
            if not reason_codes:
                raise RunnerError("rejected edit decisions artifact reason_codes must be non-empty")
            rejected_seen += 1
        seen_lines.add(line_number)
    if accepted_seen != accepted or rejected_seen != rejected:
        raise RunnerError("edit decisions artifact disposition counts mismatch")


def _validate_hashed_artifact(
    archive: ExperimentArchive,
    record: dict[str, object],
    path_key: str,
    sha_key: str,
    expected_relative: Path,
) -> Path:
    expected = expected_relative.as_posix()
    if record.get(path_key) != expected:
        raise RunnerError(f"run record {path_key} mismatch")
    path = archive.root / expected_relative
    if not path.is_file():
        raise RunnerError(f"run record {path_key} target is missing")
    if record.get(sha_key) != config.sha256_file(path):
        raise RunnerError(f"run record {sha_key} mismatch")
    return path


def _archive_relative_file(archive: ExperimentArchive, value: object, label: str) -> Path:
    if not isinstance(value, str) or not value or Path(value).is_absolute():
        raise RunnerError(f"experiment 57 frozen prompt lock {label} must be a relative file path")
    path = (archive.root / value).resolve()
    if not config.is_relative_to(path, archive.root):
        raise RunnerError(f"experiment 57 frozen prompt lock {label} escapes archive")
    if not path.is_file():
        raise RunnerError(f"experiment 57 frozen prompt lock {label} is missing")
    return path


def _run_selective_acceptance_policy(
    archive: ExperimentArchive,
    run_dir: Path,
    file_id: str,
    document_text: str,
    response: StructuredResponse,
) -> tuple[str, dict[str, object]]:
    decisions: list[dict[str, object]] = []
    accepted_edits: list[LineEdit] = []
    for edit in response.edits:
        single_edit_candidate = candidate_for_edits(document_text, response, (edit,))
        reason_codes = _hard_gate_reason_codes(document_text, single_edit_candidate)
        if reason_codes:
            decisions.append({
                "line_number": edit.line_number,
                "category": edit.category,
                "disposition": "rejected",
                "reason_codes": reason_codes,
            })
        else:
            accepted_edits.append(edit)
            decisions.append({
                "line_number": edit.line_number,
                "category": edit.category,
                "disposition": "accepted",
                "reason_codes": [],
            })
    accepted_count = len(accepted_edits)
    rejected_count = len(response.edits) - accepted_count
    decisions_payload = {
        "experiment_id": archive.experiment_id,
        "file_id": file_id,
        "proposed_count": len(response.edits),
        "accepted_count": accepted_count,
        "rejected_count": rejected_count,
        "decisions": decisions,
    }
    _validate_edit_decisions_payload(response, decisions_payload)
    edit_decisions_path = run_dir / "edit-decisions.json"
    write_json_atomic(edit_decisions_path, decisions_payload)
    candidate_text = candidate_for_edits(document_text, response, tuple(accepted_edits))
    aggregate_failures = _hard_gate_reason_codes(document_text, candidate_text)
    aggregate_payload = {
        "experiment_id": archive.experiment_id,
        "file_id": file_id,
        "proposed_count": len(response.edits),
        "accepted_count": accepted_count,
        "rejected_count": rejected_count,
        "aggregate_hard_gate_failures": aggregate_failures,
        "hard_gates_passed": aggregate_failures == [],
    }
    aggregate_gates_path = run_dir / "aggregate-gates.json"
    write_json_atomic(aggregate_gates_path, aggregate_payload)
    record_fields = {
        "proposed_edit_count": len(response.edits),
        "accepted_edit_count": accepted_count,
        "rejected_edit_count": rejected_count,
        "edit_decisions_path": archive.rel(edit_decisions_path),
        "edit_decisions_sha256": config.sha256_file(edit_decisions_path),
        "aggregate_gates_path": archive.rel(aggregate_gates_path),
        "aggregate_gates_sha256": config.sha256_file(aggregate_gates_path),
        "aggregate_hard_gate_failures": aggregate_failures,
    }
    if aggregate_failures:
        raise RunnerError("aggregate hard gates failed")
    return candidate_text, record_fields


def _hard_gate_reason_codes(original: str, candidate: str) -> list[str]:
    failures = hard_gate_failures(original, candidate)
    unknown = sorted(set(failures) - set(HARD_GATE_REASON_CODES))
    if unknown:
        raise RunnerError("hard gate returned unknown reason code: " + ", ".join(unknown))
    return failures


def _validate_edit_decisions_payload(response: StructuredResponse, payload: dict[str, object]) -> None:
    decisions = payload.get("decisions")
    if not isinstance(decisions, list):
        raise RunnerError("edit decisions must contain a decisions array")
    proposed_count = payload.get("proposed_count")
    accepted_count = payload.get("accepted_count")
    rejected_count = payload.get("rejected_count")
    if proposed_count != len(response.edits):
        raise RunnerError("edit decisions proposed_count mismatch")
    if not isinstance(accepted_count, int) or not isinstance(rejected_count, int):
        raise RunnerError("edit decisions accepted/rejected counts must be integers")
    if proposed_count != accepted_count + rejected_count:
        raise RunnerError("edit decisions counts must satisfy proposed = accepted + rejected")
    expected_category_by_line = {edit.line_number: edit.category for edit in response.edits}
    seen_lines: set[int] = set()
    accepted_seen = 0
    rejected_seen = 0
    allowed_reasons = set(HARD_GATE_REASON_CODES)
    for item in decisions:
        if not isinstance(item, dict) or set(item) != {"line_number", "category", "disposition", "reason_codes"}:
            raise RunnerError("each edit decision must contain line_number, category, disposition, and reason_codes only")
        line_number = item["line_number"]
        category = item["category"]
        disposition = item["disposition"]
        reason_codes = item["reason_codes"]
        if not isinstance(line_number, int) or line_number not in expected_category_by_line or line_number in seen_lines:
            raise RunnerError("edit decision line_number is missing, duplicate, or unexpected")
        if category != expected_category_by_line[line_number]:
            raise RunnerError("edit decision category mismatch")
        if disposition not in {"accepted", "rejected"}:
            raise RunnerError("edit decision disposition must be accepted or rejected")
        if not isinstance(reason_codes, list) or any(not isinstance(reason, str) for reason in reason_codes):
            raise RunnerError("edit decision reason_codes must be a string array")
        if any(reason not in allowed_reasons for reason in reason_codes):
            raise RunnerError("edit decision contains an unknown hard-gate reason code")
        if disposition == "accepted":
            if reason_codes:
                raise RunnerError("accepted edit decision reason_codes must be empty")
            accepted_seen += 1
        else:
            if not reason_codes:
                raise RunnerError("rejected edit decision reason_codes must be non-empty")
            rejected_seen += 1
        seen_lines.add(line_number)
    if seen_lines != set(expected_category_by_line):
        raise RunnerError("edit decisions do not account for every proposed edit")
    if accepted_seen != accepted_count or rejected_seen != rejected_count:
        raise RunnerError("edit decisions disposition counts mismatch")
