from __future__ import annotations

from report_natural_voice_correction.archive import read_json

from .archive import ExperimentArchive
from .prompts import load_prompt
from .records import RecordError, _finalize_run, render_prompt


def resume_pilot_contract(
    archive: ExperimentArchive,
    file_id: str,
    arm: str,
) -> dict[str, object]:
    return _resume_contract(
        archive, "development", ("pilot",), file_id, arm
    )


def resume_full_contract(
    archive: ExperimentArchive,
    file_id: str,
    arm: str,
) -> dict[str, object]:
    from .runner import authorize_full

    authorize_full(archive)
    return _resume_contract(
        archive, "evaluation", ("full",), file_id, arm
    )


def _resume_contract(
    archive: ExperimentArchive,
    set_name: str,
    phase: tuple[str, ...],
    file_id: str,
    arm: str,
) -> dict[str, object]:
    original_path = archive.input_path(set_name, file_id)
    original = original_path.read_text(encoding="utf-8")
    prompt, prompt_sha = load_prompt(archive, arm)
    run_dir = archive.root / "runs" / phase[0] / file_id / arm
    rendered_path = run_dir / "rendered-prompt.txt"
    raw_path = run_dir / "raw-output.json"
    command_path = run_dir / "codex-command.json"
    required = (rendered_path, raw_path, command_path)
    if not all(path.is_file() for path in required):
        raise RecordError("contract resume requires the preserved raw attempt")
    finalized = (
        "record.json", "candidate.md", "edit-decisions.json", "aggregate-gates.json",
    )
    if any((run_dir / name).exists() for name in finalized):
        raise RecordError("contract resume requires an unfinalized attempt")
    if rendered_path.read_text(encoding="utf-8") != render_prompt(prompt, file_id, original):
        raise RecordError("preserved rendered prompt changed")
    command = read_json(command_path)
    if command.get("returncode") != 0:
        raise RecordError("preserved model command did not succeed")
    return _finalize_run(
        archive, set_name, file_id, arm, prompt_sha, phase,
        original_path, original, run_dir, rendered_path, raw_path, command_path,
    )
