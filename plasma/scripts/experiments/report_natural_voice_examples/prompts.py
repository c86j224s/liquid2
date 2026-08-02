from __future__ import annotations

import json
from pathlib import Path
import random
import shutil
import time

from report_natural_voice_correction.archive import read_json, write_json_atomic, write_text_atomic
from report_natural_voice_correction.edits import response_schema

from . import config
from .archive import ExperimentArchive


class PromptError(ValueError):
    pass


EXAMPLE_CATEGORIES = (
    "translationese_nominalization",
    "formulaic_connection",
    "uniform_cadence",
    "process_narration",
    "inflated_abstraction",
    "redundant_framing",
)


def prepare_control_prompt(archive: ExperimentArchive) -> Path:
    source = (archive.home / config.CONTROL_PROMPT_SOURCE).resolve()
    if not source.is_file() or config.sha256_file(source) != config.CONTROL_PROMPT_SHA256:
        raise PromptError("experiment 57 control prompt is missing or changed")
    destination = archive.root / "control" / "prompts" / "control.md"
    destination.parent.mkdir(parents=True, exist_ok=True)
    if destination.exists() and config.sha256_file(destination) != config.CONTROL_PROMPT_SHA256:
        raise PromptError("existing control prompt differs from experiment 57")
    if not destination.exists():
        shutil.copyfile(source, destination)
    return destination


def lint_example_prompt(archive: ExperimentArchive, candidate_path: Path) -> dict[str, object]:
    control_path = prepare_control_prompt(archive)
    control = control_path.read_text(encoding="utf-8").rstrip()
    candidate = Path(candidate_path).expanduser().resolve().read_text(encoding="utf-8")
    prefix = control + "\n\n# Target voice examples\n"
    if not candidate.startswith(prefix):
        raise PromptError("example prompt must preserve the experiment 57 prompt byte-for-byte before the examples section")
    suffix = candidate[len(control):]
    for category in EXAMPLE_CATEGORIES:
        if suffix.count(f"## {category}\n") != 1:
            raise PromptError(f"example prompt must contain one section for {category}")
    if suffix.count("Before:\n") != len(EXAMPLE_CATEGORIES):
        raise PromptError("example prompt must contain one positive before example per category")
    if suffix.count("After:\n") != len(EXAMPLE_CATEGORIES):
        raise PromptError("example prompt must contain one positive after example per category")
    if suffix.count("Preserve:\n") != len(EXAMPLE_CATEGORIES):
        raise PromptError("example prompt must contain one preserve example per category")
    forbidden = tuple(config.DEVELOPMENT_SOURCES) + tuple(config.EVALUATION_SOURCES)
    if any(token in suffix for token in forbidden):
        raise PromptError("example prompt leaks a corpus filename")
    return {
        "path": str(Path(candidate_path).expanduser().resolve()),
        "sha256": config.sha256_text(candidate),
        "examples_sha256": config.sha256_text(suffix),
        "bytes": len(candidate.encode("utf-8")),
    }


def freeze_protocol(archive: ExperimentArchive, candidate_path: Path) -> dict[str, object]:
    archive.verify_source_seal()
    linted = lint_example_prompt(archive, candidate_path)
    lock_path = archive.root / "control" / "protocol.lock.json"
    if lock_path.exists():
        existing = read_json(lock_path)
        _validate_protocol_lock(archive, existing)
        prompts = existing.get("prompts")
        if not isinstance(prompts, dict) or not isinstance(prompts.get("examples"), dict):
            raise PromptError("protocol lock example prompt metadata missing")
        if prompts["examples"].get("sha256") != linted["sha256"]:
            raise PromptError("protocol is already frozen with another example prompt")
        return existing

    control_source = prepare_control_prompt(archive)
    control_lock = archive.root / "control" / "prompts" / "control.lock.md"
    examples_lock = archive.root / "control" / "prompts" / "examples.lock.md"
    shutil.copyfile(control_source, control_lock)
    shutil.copyfile(Path(str(linted["path"])), examples_lock)
    schema_path = archive.root / "control" / "structured-response.schema.json"
    schema_text = json.dumps(response_schema(), ensure_ascii=False, indent=2, sort_keys=True) + "\n"
    write_text_atomic(schema_path, schema_text)

    schedule = [
        {"file_id": file_id, "arm": arm}
        for file_id in archive.evaluation_file_ids()
        for arm in config.ARMS
    ]
    random.Random(config.SCHEDULE_SEED).shuffle(schedule)
    source_manifest = archive.root / "control" / "source-manifest.lock.json"
    lock = {
        "experiment_id": config.EXPERIMENT_ID,
        "status": "frozen_before_pilot_calls",
        "locked_at_utc": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "model": config.MODEL,
        "reasoning_effort": config.REASONING_EFFORT,
        "prompts": {
            "control": {"path": archive.rel(control_lock), "sha256": config.sha256_file(control_lock)},
            "examples": {
                "path": archive.rel(examples_lock),
                "sha256": config.sha256_file(examples_lock),
                "examples_sha256": linted["examples_sha256"],
            },
        },
        "schema": {"path": archive.rel(schema_path), "sha256": config.sha256_file(schema_path)},
        "source_manifest_sha256": config.sha256_file(source_manifest),
        "development_file_ids": list(archive.development_file_ids()),
        "evaluation_file_ids": list(archive.evaluation_file_ids()),
        "schedule_seed": config.SCHEDULE_SEED,
        "full_run_schedule": schedule,
        "success_criteria": config.SUCCESS_CRITERIA,
        "full_run_retry_policy": "no scientific retry; interrupted runs may resume only untouched cells",
    }
    write_json_atomic(lock_path, lock)
    _validate_protocol_lock(archive, lock)
    return lock


def load_prompt(archive: ExperimentArchive, arm: str) -> tuple[str, str]:
    if arm not in config.ARMS:
        raise PromptError(f"unknown arm: {arm}")
    archive.verify_source_seal()
    lock = read_json(archive.root / "control" / "protocol.lock.json")
    _validate_protocol_lock(archive, lock)
    row = lock["prompts"][arm]  # type: ignore[index]
    path = archive.root / str(row["path"])
    return path.read_text(encoding="utf-8"), str(row["sha256"])


def _validate_protocol_lock(archive: ExperimentArchive, lock: dict[str, object]) -> None:
    if lock.get("experiment_id") != config.EXPERIMENT_ID or lock.get("status") != "frozen_before_pilot_calls":
        raise PromptError("protocol lock identity or status mismatch")
    if lock.get("model") != config.MODEL or lock.get("reasoning_effort") != config.REASONING_EFFORT:
        raise PromptError("protocol lock model mismatch")
    if lock.get("source_manifest_sha256") != config.sha256_file(archive.root / "control" / "source-manifest.lock.json"):
        raise PromptError("protocol lock source manifest mismatch")
    if lock.get("development_file_ids") != list(archive.development_file_ids()):
        raise PromptError("protocol lock development corpus mismatch")
    if lock.get("evaluation_file_ids") != list(archive.evaluation_file_ids()):
        raise PromptError("protocol lock evaluation corpus mismatch")
    expected_schedule = [
        {"file_id": file_id, "arm": arm}
        for file_id in archive.evaluation_file_ids()
        for arm in config.ARMS
    ]
    random.Random(config.SCHEDULE_SEED).shuffle(expected_schedule)
    if lock.get("schedule_seed") != config.SCHEDULE_SEED or lock.get("full_run_schedule") != expected_schedule:
        raise PromptError("protocol lock full-run schedule mismatch")
    prompts = lock.get("prompts")
    if not isinstance(prompts, dict) or set(prompts) != set(config.ARMS):
        raise PromptError("protocol lock prompt arms mismatch")
    for arm in config.ARMS:
        row = prompts.get(arm)
        if not isinstance(row, dict) or "path" not in row or "sha256" not in row:
            raise PromptError("protocol lock prompt metadata mismatch")
        path = (archive.root / str(row["path"])).resolve()
        if not path.is_file() or config.sha256_file(path) != row["sha256"]:
            raise PromptError(f"frozen {arm} prompt changed")
    if prompts["control"]["sha256"] != config.CONTROL_PROMPT_SHA256:
        raise PromptError("control prompt is not the experiment 57 prompt")
    examples_path = archive.root / str(prompts["examples"]["path"])
    examples = lint_example_prompt(archive, examples_path)
    if prompts["examples"].get("examples_sha256") != examples["examples_sha256"]:
        raise PromptError("frozen example section changed")
    schema = lock.get("schema")
    if not isinstance(schema, dict):
        raise PromptError("protocol lock schema metadata missing")
    schema_path = archive.root / str(schema.get("path"))
    if not schema_path.is_file() or config.sha256_file(schema_path) != schema.get("sha256"):
        raise PromptError("frozen response schema changed")
    expected_schema = json.dumps(response_schema(), ensure_ascii=False, indent=2, sort_keys=True) + "\n"
    if schema_path.read_text(encoding="utf-8") != expected_schema:
        raise PromptError("frozen response schema no longer matches the runner")
    if lock.get("success_criteria") != config.SUCCESS_CRITERIA:
        raise PromptError("protocol lock success criteria mismatch")
    if lock.get("full_run_retry_policy") != "no scientific retry; interrupted runs may resume only untouched cells":
        raise PromptError("protocol lock retry policy mismatch")
