from __future__ import annotations

from report_natural_voice_correction.archive import read_json, write_json_atomic, write_text_atomic
from report_natural_voice_correction.edits import parse_model_response
from report_natural_voice_examples import config as base_config
from report_natural_voice_examples import records as base_records
from report_natural_voice_examples.archive import ExperimentArchive
from report_natural_voice_examples.prompts import load_prompt


def resume_failed_pilots(archive: ExperimentArchive) -> list[dict[str, object]]:
    records = []
    for file_id in archive.development_file_ids():
        for arm in base_config.ARMS:
            run_dir = archive.root / "runs" / "pilot" / file_id / arm
            record_path = run_dir / "record.json"
            if record_path.is_file():
                record = read_json(record_path)
                base_records.validate_record(
                    archive, record, "development", ("pilot",), file_id, arm
                )
            else:
                record = resume_pilot_attempt(archive, file_id, arm)
            records.append(record)
    return records


def resume_pilot_attempt(
    archive: ExperimentArchive,
    file_id: str,
    arm: str,
) -> dict[str, object]:
    run_dir = archive.root / "runs" / "pilot" / file_id / arm
    original_path = archive.input_path("development", file_id)
    original = original_path.read_text(encoding="utf-8")
    prompt, prompt_sha = load_prompt(archive, arm)
    rendered_path = run_dir / "rendered-prompt.txt"
    raw_path = run_dir / "raw-output.json"
    command_path = run_dir / "codex-command.json"
    if not all(path.is_file() for path in (rendered_path, raw_path, command_path)):
        raise base_records.RecordError("pilot resume requires the preserved raw attempt")
    if any((run_dir / name).exists() for name in (
        "record.json", "candidate.md", "edit-decisions.json", "aggregate-gates.json",
    )):
        raise base_records.RecordError("pilot resume requires an unfinalized attempt")
    if rendered_path.read_text(encoding="utf-8") != base_records.render_prompt(prompt, file_id, original):
        raise base_records.RecordError("preserved rendered prompt changed")
    if read_json(command_path).get("returncode") != 0:
        raise base_records.RecordError("preserved model command did not succeed")

    response = parse_model_response(raw_path.read_text(encoding="utf-8"))
    candidate, policy_fields = base_records.apply_selective_policy(
        archive, run_dir, file_id, arm, original, response
    )
    candidate_path = run_dir / "candidate.md"
    write_text_atomic(candidate_path, candidate)
    record = {
        "experiment_id": base_config.EXPERIMENT_ID,
        "phase": ["pilot"],
        "set_name": "development",
        "file_id": file_id,
        "source_filename": original_path.name,
        "arm": arm,
        "document_sha256": base_config.sha256_text(original),
        "instruction_prompt_sha256": prompt_sha,
        "model": base_config.MODEL,
        "reasoning_effort": base_config.REASONING_EFFORT,
        "hard_gates_passed": True,
    }
    record.update(base_records._artifact_fields(
        archive, rendered_path, raw_path, candidate_path, command_path
    ))
    record.update(policy_fields)
    write_json_atomic(run_dir / "record.json", record)
    base_records.validate_record(
        archive, record, "development", ("pilot",), file_id, arm
    )
    return record
