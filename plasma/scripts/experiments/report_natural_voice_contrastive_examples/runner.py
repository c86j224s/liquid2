from __future__ import annotations

from concurrent.futures import ThreadPoolExecutor
from pathlib import Path

from report_natural_voice_correction.archive import read_json, write_json_atomic

from . import config
from .archive import ExperimentArchive
from .prompts import lint_contrastive_prompt, load_prompt
from .records import RecordError, run_document, validate_record


class RunnerError(ValueError):
    pass


def run_calibration(archive: ExperimentArchive, prompt_path: Path) -> list[dict[str, object]]:
    archive.verify_source_seal()
    linted = lint_contrastive_prompt(archive, prompt_path)
    prompt = Path(str(linted["path"])).read_text(encoding="utf-8")
    prompt_sha = str(linted["sha256"])
    return [
        run_document(
            archive,
            "development",
            file_id,
            "contrastive",
            prompt,
            prompt_sha,
            ("calibration", prompt_sha),
        )
        for file_id in archive.development_file_ids()
    ]


def run_pilots(archive: ExperimentArchive, workers: int = 2) -> list[dict[str, object]]:
    archive.verify_source_seal()
    cells = [
        (file_id, arm)
        for file_id in archive.development_file_ids()
        for arm in config.ARMS
    ]

    def execute(cell: tuple[str, str]) -> dict[str, object]:
        file_id, arm = cell
        prompt, prompt_sha = load_prompt(archive, arm)
        return run_document(
            archive, "development", file_id, arm, prompt, prompt_sha, ("pilot",)
        )

    with ThreadPoolExecutor(max_workers=max(1, workers)) as pool:
        return list(pool.map(execute, cells))


def authorize_full(archive: ExperimentArchive) -> dict[str, object]:
    pilot_records: list[dict[str, str]] = []
    for file_id in archive.development_file_ids():
        for arm in config.ARMS:
            path = archive.root / "runs" / "pilot" / file_id / arm / "record.json"
            if not path.is_file():
                raise RunnerError("all four pilot cells must pass before full authorization")
            record = read_json(path)
            _validate_record(archive, record, "development", ("pilot",), file_id, arm)
            pilot_records.append({"path": archive.rel(path), "sha256": config.sha256_file(path)})
    path = archive.root / "analysis" / "pilot-acceptance-gate.json"
    stable = {
        "experiment_id": config.EXPERIMENT_ID,
        "status": "authorized_for_full",
        "protocol_lock_sha256": config.sha256_file(archive.root / "control" / "protocol.lock.json"),
        "pilot_records": pilot_records,
    }
    if path.exists():
        existing = read_json(path)
        if existing != stable:
            raise RunnerError("pilot authorization already exists with different evidence")
        return existing
    write_json_atomic(path, stable)
    return stable


def run_full(archive: ExperimentArchive, workers: int = 2) -> list[dict[str, object]]:
    archive.verify_source_seal()
    _validate_full_authorization(archive)
    lock = read_json(archive.root / "control" / "protocol.lock.json")
    schedule = lock.get("full_run_schedule")
    if not isinstance(schedule, list) or len(schedule) != 16:
        raise RunnerError("frozen full-run schedule must contain 16 cells")
    completed: list[dict[str, object]] = []
    pending: list[tuple[str, str]] = []
    for cell in schedule:
        if not isinstance(cell, dict):
            raise RunnerError("full-run schedule cell must be an object")
        file_id, arm = str(cell.get("file_id")), str(cell.get("arm"))
        run_dir = archive.root / "runs" / "full" / file_id / arm
        record_path = run_dir / "record.json"
        if record_path.is_file():
            record = read_json(record_path)
            _validate_record(archive, record, "evaluation", ("full",), file_id, arm)
            completed.append(record)
        elif run_dir.exists():
            raise RunnerError(f"failed or partial full-run cell cannot be retried: {file_id}/{arm}")
        else:
            pending.append((file_id, arm))

    def execute(cell: tuple[str, str]) -> dict[str, object]:
        file_id, arm = cell
        prompt, prompt_sha = load_prompt(archive, arm)
        return run_document(
            archive, "evaluation", file_id, arm, prompt, prompt_sha, ("full",)
        )

    with ThreadPoolExecutor(max_workers=max(1, workers)) as pool:
        completed.extend(pool.map(execute, pending))
    return completed


def _validate_full_authorization(archive: ExperimentArchive) -> None:
    gate_path = archive.root / "analysis" / "pilot-acceptance-gate.json"
    if not gate_path.is_file():
        raise RunnerError("full run requires pilot authorization")
    expected = authorize_full(archive)
    if read_json(gate_path) != expected:
        raise RunnerError("pilot authorization changed")


def _validate_record(*args: object) -> None:
    try:
        validate_record(*args)  # type: ignore[arg-type]
    except RecordError as exc:
        raise RunnerError(str(exc)) from exc
