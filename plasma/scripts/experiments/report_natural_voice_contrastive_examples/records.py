from __future__ import annotations

import json
from pathlib import Path
import subprocess
import time
from typing import Any, Callable

from report_natural_voice_correction.archive import write_json_atomic, write_text_atomic
from report_natural_voice_correction.edits import response_schema, split_document_lines
from report_natural_voice_correction.guards import hard_gate_failures

from . import config
from .archive import ExperimentArchive
from .contract import ContractError, parse_with_amendment, validate_response_artifacts
from .policy import apply_selective_policy, validate_policy_artifacts


class RecordError(ValueError):
    pass


def run_document(
    archive: ExperimentArchive,
    set_name: str,
    file_id: str,
    arm: str,
    prompt: str,
    prompt_sha: str,
    phase: tuple[str, ...],
    subprocess_run: Callable[..., Any] | None = None,
) -> dict[str, object]:
    original_path = archive.input_path(set_name, file_id)
    original = original_path.read_text(encoding="utf-8")
    run_dir = archive.root / "runs" / Path(*phase) / file_id / arm
    if run_dir.exists():
        raise RecordError(f"model call already attempted: {'/'.join(phase)}/{file_id}/{arm}")
    run_dir.mkdir(parents=True)
    schema_path = ensure_schema(archive)
    rendered_path = run_dir / "rendered-prompt.txt"
    raw_path = run_dir / "raw-output.json"
    command_path = run_dir / "codex-command.json"
    write_text_atomic(rendered_path, render_prompt(prompt, file_id, original))
    command = codex_command(archive, raw_path, schema_path)
    started = time.time()
    execute = subprocess_run or subprocess.run
    process = execute(
        command,
        input=rendered_path.read_text(encoding="utf-8"),
        text=True,
        capture_output=True,
    )
    write_json_atomic(command_path, {
        "command": command,
        "returncode": int(getattr(process, "returncode", 1)),
        "stdout": str(getattr(process, "stdout", "")),
        "stderr": str(getattr(process, "stderr", "")),
        "duration_seconds": round(time.time() - started, 3),
    })
    if getattr(process, "returncode", 1) != 0 or not raw_path.is_file():
        raise RecordError("codex command failed or produced no output")
    return _finalize_run(
        archive, set_name, file_id, arm, prompt_sha, phase,
        original_path, original, run_dir, rendered_path, raw_path, command_path,
    )


def _finalize_run(
    archive: ExperimentArchive,
    set_name: str,
    file_id: str,
    arm: str,
    prompt_sha: str,
    phase: tuple[str, ...],
    original_path: Path,
    original: str,
    run_dir: Path,
    rendered_path: Path,
    raw_path: Path,
    command_path: Path,
) -> dict[str, object]:
    try:
        response, contract_fields = parse_with_amendment(
            archive, raw_path, run_dir, original
        )
    except (ContractError, ValueError) as exc:
        raise RecordError(str(exc)) from exc
    candidate, policy_fields = apply_selective_policy(
        archive, run_dir, file_id, arm, original, response
    )
    candidate_path = run_dir / "candidate.md"
    write_text_atomic(candidate_path, candidate)
    record = {
        "experiment_id": config.EXPERIMENT_ID,
        "phase": list(phase),
        "set_name": set_name,
        "file_id": file_id,
        "source_filename": original_path.name,
        "arm": arm,
        "document_sha256": config.sha256_text(original),
        "instruction_prompt_sha256": prompt_sha,
        "model": config.MODEL,
        "reasoning_effort": config.REASONING_EFFORT,
        "hard_gates_passed": True,
    }
    record.update(_artifact_fields(archive, rendered_path, raw_path, candidate_path, command_path))
    record.update(contract_fields)
    record.update(policy_fields)
    write_json_atomic(run_dir / "record.json", record)
    validate_record(archive, record, set_name, phase, file_id, arm)
    return record


def validate_record(
    archive: ExperimentArchive,
    record: dict[str, object],
    set_name: str,
    phase: tuple[str, ...],
    file_id: str,
    arm: str,
) -> None:
    original_path = archive.input_path(set_name, file_id)
    original = original_path.read_text(encoding="utf-8")
    if phase and phase[0] == "calibration":
        if len(phase) != 2 or arm != "contrastive":
            raise RecordError("calibration record phase or arm mismatch")
        expected_prompt_sha = phase[1]
    else:
        from .prompts import load_prompt

        _, expected_prompt_sha = load_prompt(archive, arm)
    required = {
        "experiment_id": config.EXPERIMENT_ID,
        "phase": list(phase),
        "set_name": set_name,
        "file_id": file_id,
        "source_filename": original_path.name,
        "arm": arm,
        "document_sha256": config.sha256_text(original),
        "instruction_prompt_sha256": expected_prompt_sha,
        "model": config.MODEL,
        "reasoning_effort": config.REASONING_EFFORT,
        "hard_gates_passed": True,
    }
    if arm not in config.ARMS or any(record.get(key) != value for key, value in required.items()):
        raise RecordError("run record identity mismatch")
    run_dir = archive.root / "runs" / Path(*phase) / file_id / arm
    candidate_path = _validate_artifact(archive, record, run_dir, "candidate", ".md")
    raw_path = _validate_artifact(archive, record, run_dir, "raw_output", ".json")
    _validate_artifact(archive, record, run_dir, "rendered_prompt", ".txt")
    _validate_artifact(archive, record, run_dir, "codex_command", ".json")
    if hard_gate_failures(original, candidate_path.read_text(encoding="utf-8")):
        raise RecordError("candidate no longer passes hard gates")
    try:
        validate_response_artifacts(archive, record, run_dir, raw_path, original)
    except (ContractError, ValueError) as exc:
        raise RecordError(str(exc)) from exc
    validate_policy_artifacts(archive, record, run_dir)


def ensure_schema(archive: ExperimentArchive) -> Path:
    path = archive.root / "control" / "structured-response.schema.json"
    text = json.dumps(response_schema(), ensure_ascii=False, indent=2, sort_keys=True) + "\n"
    if path.exists() and path.read_text(encoding="utf-8") != text:
        raise RecordError("response schema changed")
    if not path.exists():
        write_text_atomic(path, text)
    return path


def render_prompt(prompt: str, file_id: str, document: str) -> str:
    lines = [
        prompt.rstrip(),
        "",
        "SEALED_DOCUMENT",
        f"file_id: {file_id}",
        f"document_sha256: {config.sha256_text(document)}",
        "numbered_lines:",
    ]
    lines.extend(
        f"{number}\t{config.sha256_text(line)}\t{line}"
        for number, line in enumerate(split_document_lines(document), 1)
    )
    return "\n".join(lines) + "\n"


def codex_command(archive: ExperimentArchive, output: Path, schema: Path) -> list[str]:
    return [
        "codex", "--sandbox", "read-only", "exec", "--ephemeral", "--ignore-user-config",
        "--ignore-rules", "--skip-git-repo-check", "-C", str(archive.root / "tmp-harness"),
        "-m", config.MODEL, "-c", f'model_reasoning_effort="{config.REASONING_EFFORT}"',
        "-o", str(output), "--output-schema", str(schema), "-",
    ]


def _artifact_fields(archive: ExperimentArchive, *paths: Path) -> dict[str, str]:
    fields: dict[str, str] = {}
    for path in paths:
        stem = path.name.rsplit(".", 1)[0].replace("-", "_")
        fields[f"{stem}_path"] = archive.rel(path)
        fields[f"{stem}_sha256"] = config.sha256_file(path)
    return fields


def _validate_artifact(
    archive: ExperimentArchive,
    record: dict[str, object],
    run_dir: Path,
    stem: str,
    suffix: str,
) -> Path:
    path = run_dir / f"{stem.replace('_', '-')}{suffix}"
    if record.get(f"{stem}_path") != archive.rel(path) or not path.is_file():
        raise RecordError(f"{stem} artifact path mismatch")
    if record.get(f"{stem}_sha256") != config.sha256_file(path):
        raise RecordError(f"{stem} artifact hash mismatch")
    return path
