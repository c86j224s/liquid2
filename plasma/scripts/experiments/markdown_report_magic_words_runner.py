"""Archive and Codex subprocess boundary for the issue 77 experiment."""

from __future__ import annotations

import hashlib
import json
import shutil
import subprocess
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

from markdown_report_magic_words_protocol import EXPERIMENT_ID


ARCHIVE_ROOT = Path.home() / "research-artifacts/liquid2/plasma/experiments"
DEFAULT_ARCHIVE = ARCHIVE_ROOT / EXPERIMENT_ID


class SafetyError(RuntimeError):
    pass


def utc_now() -> str:
    return datetime.now(timezone.utc).isoformat()


def sha256_text(value: str) -> str:
    return hashlib.sha256(value.encode("utf-8")).hexdigest()


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def write_json(path: Path, value: Any) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, ensure_ascii=False, indent=2, sort_keys=True) + "\n", encoding="utf-8")


def append_jsonl(path: Path, value: Any) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("a", encoding="utf-8") as handle:
        handle.write(json.dumps(value, ensure_ascii=False, sort_keys=True) + "\n")


def cached_prompt_matches(archive: Path, run_id: str, prompt: str, output_path: Path) -> bool:
    if not output_path.is_file() or not output_path.read_text(encoding="utf-8", errors="replace").strip():
        return False
    records_path = validate_archive(archive) / "runs" / "commands.jsonl"
    if not records_path.is_file():
        return False
    expected = sha256_text(prompt)
    for line in reversed(records_path.read_text(encoding="utf-8", errors="replace").splitlines()):
        try:
            record = json.loads(line)
        except json.JSONDecodeError:
            continue
        if record.get("run_id") == run_id:
            return record.get("returncode") == 0 and record.get("prompt_sha256") == expected
    return False


def validate_archive(path: Path) -> Path:
    archive = path.expanduser().resolve()
    root = ARCHIVE_ROOT.expanduser().resolve()
    try:
        relative = archive.relative_to(root)
    except ValueError as exc:
        raise SafetyError(f"archive must stay under {root}: {archive}") from exc
    if relative.parts != (EXPERIMENT_ID,):
        raise SafetyError(f"archive must be the isolated {EXPERIMENT_ID} directory: {archive}")
    return archive


def ensure_archive(path: Path) -> Path:
    archive = validate_archive(path)
    for name in ("analysis", "blind", "judging", "logs", "reports", "runs", "sources", "tmp-harness"):
        (archive / name).mkdir(parents=True, exist_ok=True)
    return archive


def codex_version() -> str:
    binary = shutil.which("codex")
    if not binary:
        raise SafetyError("codex binary not found on PATH")
    proc = subprocess.run([binary, "--version"], text=True, capture_output=True, check=False)
    if proc.returncode != 0:
        raise SafetyError(f"codex --version failed: {proc.stderr.strip()}")
    return proc.stdout.strip()


def run_codex(
    *,
    archive: Path,
    run_id: str,
    prompt: str,
    output_path: Path,
    model: str,
    reasoning_effort: str,
    timeout_seconds: int,
    output_schema: Path | None = None,
) -> dict[str, Any]:
    archive = validate_archive(archive)
    binary = shutil.which("codex")
    if not binary:
        raise SafetyError("codex binary not found on PATH")
    workdir = archive / "tmp-harness" / run_id
    workdir.mkdir(parents=True, exist_ok=True)
    output_path.parent.mkdir(parents=True, exist_ok=True)
    stdout_path = archive / "logs" / f"{run_id}.stdout.log"
    stderr_path = archive / "logs" / f"{run_id}.stderr.log"
    command = [
        binary,
        "--sandbox",
        "read-only",
        "exec",
        "--ephemeral",
        "--ignore-user-config",
        "--ignore-rules",
        "--skip-git-repo-check",
        "-C",
        str(workdir),
        "-m",
        model,
        "-c",
        f'model_reasoning_effort="{reasoning_effort}"',
        "-o",
        str(output_path),
    ]
    if output_schema is not None:
        command.extend(["--output-schema", str(output_schema)])
    command.append("-")
    started = utc_now()
    proc = subprocess.run(
        command,
        input=prompt,
        text=True,
        capture_output=True,
        timeout=timeout_seconds,
        check=False,
    )
    stdout_path.write_text(proc.stdout, encoding="utf-8")
    stderr_path.write_text(proc.stderr, encoding="utf-8")
    record = {
        "run_id": run_id,
        "command": command,
        "prompt_sha256": sha256_text(prompt),
        "prompt_bytes": len(prompt.encode("utf-8")),
        "returncode": proc.returncode,
        "started_at": started,
        "completed_at": utc_now(),
        "output_path": str(output_path),
        "stdout_log": str(stdout_path),
        "stderr_log": str(stderr_path),
    }
    append_jsonl(archive / "runs" / "commands.jsonl", record)
    if proc.returncode != 0:
        raise RuntimeError(f"{run_id} failed; see {stderr_path}")
    if not output_path.exists() or not output_path.read_text(encoding="utf-8").strip():
        raise RuntimeError(f"{run_id} produced no final output")
    return record
