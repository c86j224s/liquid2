#!/usr/bin/env python3
"""Safety harness for the Plasma long-form session strategy experiment.

The raw experiment archive stays under ~/research-artifacts. This tracked
script owns command safety: every Plasma command that can touch storage must
name an archive-local DB, and every server command must bind a loopback
6000-range experiment port.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import shutil
import signal
import socket
import sqlite3
import subprocess
import sys
import time
import urllib.error
import urllib.request
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path
from typing import Any


EXPERIMENT_ID = "12-long-form-session-strategy-2026-07-07"
DEFAULT_ARCHIVE = Path.home() / "research-artifacts/liquid2/plasma/experiments" / EXPERIMENT_ID
DEFAULT_SAMPLE_ID = "s01"
DEFAULT_A0_PORT = 6202
DEFAULT_B_TIMEOUT = "150m"

FORBIDDEN_DB_FRAGMENTS = (
    "/Library/Application Support/Plasma/plasma.db",
    "/tmp/plasma-ui-user.db",
    "/runtime/dev-6002/plasma-ui-user.db",
    "/runtime/release-3002/plasma-ui-user.db",
)

FORBIDDEN_PORT_RANGES = (
    range(3000, 4000),
)

PROVIDER_ENV_KEYS = (
    "CODEX_HOME",
    "XDG_CONFIG_HOME",
    "XDG_DATA_HOME",
    "XDG_CACHE_HOME",
)

DB_REQUIRED_COMMANDS = {
    "health",
    "missions",
    "turns",
    "sources",
    "workflow",
    "reports",
    "mcp",
    "serve",
}

AGENT_TOUCHING_COMMANDS = {
    "turns",
    "workflow",
    "reports",
    "serve",
}


@dataclass(frozen=True)
class SafetyResult:
    name: str
    ok: bool
    detail: str


class SafetyError(RuntimeError):
    pass


def utc_now() -> str:
    return datetime.now(timezone.utc).isoformat()


def repo_root() -> Path:
    return Path(__file__).resolve().parents[3]


def plasma_root() -> Path:
    return Path(__file__).resolve().parents[2]


def write_json(path: Path, value: Any) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, ensure_ascii=False, indent=2, sort_keys=True) + "\n", encoding="utf-8")


def append_jsonl(path: Path, value: Any) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("a", encoding="utf-8") as handle:
        handle.write(json.dumps(value, ensure_ascii=False, sort_keys=True) + "\n")


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def ensure_archive_dirs(archive: Path) -> None:
    for name in (
        "analysis",
        "logs",
        "runs",
        "servers",
        "tmp-harness",
        "workdirs",
        "provider-homes",
    ):
        (archive / name).mkdir(parents=True, exist_ok=True)


def resolve_archive(path: Path) -> Path:
    return path.expanduser().resolve()


def is_relative_to(path: Path, parent: Path) -> bool:
    try:
        path.resolve().relative_to(parent.resolve())
    except ValueError:
        return False
    return True


def arg_values(args: list[str], name: str) -> list[str]:
    values: list[str] = []
    for index, arg in enumerate(args):
        if arg == name and index + 1 < len(args):
            values.append(args[index + 1])
        if arg.startswith(name + "="):
            values.append(arg.split("=", 1)[1])
    return values


def arg_value(args: list[str], name: str) -> str:
    values = arg_values(args, name)
    if len(values) > 1:
        raise SafetyError(f"duplicate singleton flag {name}: {values}")
    return values[0] if values else ""


def has_arg(args: list[str], name: str) -> bool:
    return any(arg == name or arg.startswith(name + "=") for arg in args)


def parse_addr(addr: str) -> tuple[str, int]:
    if not addr:
        raise SafetyError("serve command requires explicit -addr")
    if addr.count(":") != 1:
        raise SafetyError(f"serve -addr must be host:port, got {addr!r}")
    host, port_text = addr.rsplit(":", 1)
    try:
        port = int(port_text)
    except ValueError as exc:
        raise SafetyError(f"serve -addr port must be numeric, got {addr!r}") from exc
    return host, port


def validate_loopback_experiment_addr(addr: str) -> None:
    host, port = parse_addr(addr)
    if host not in {"127.0.0.1", "localhost", "::1"}:
        raise SafetyError(f"serve -addr must bind loopback only, got {addr!r}")
    if port < 6000 or port > 6999:
        raise SafetyError(f"serve -addr must use 6000-range experiment port, got {port}")
    if any(port in blocked for blocked in FORBIDDEN_PORT_RANGES):
        raise SafetyError(f"serve -addr must not use release port range, got {port}")


def validate_archive_db(archive: Path, db_text: str) -> Path:
    if not db_text:
        raise SafetyError("command requires explicit -db")
    db_path = Path(db_text).expanduser()
    resolved = db_path.resolve()
    if not is_relative_to(resolved, archive):
        raise SafetyError(f"refusing non-archive DB path: {db_text}")
    resolved_text = str(resolved)
    for fragment in FORBIDDEN_DB_FRAGMENTS:
        if fragment in resolved_text:
            raise SafetyError(f"refusing forbidden DB path: {db_text}")
    return resolved


def validate_provider_env(archive: Path, env: dict[str, str], *, required: bool) -> None:
    missing = [key for key in PROVIDER_ENV_KEYS if not env.get(key)]
    if missing and required:
        raise SafetyError(f"provider env missing: {', '.join(missing)}")
    for key in PROVIDER_ENV_KEYS:
        value = env.get(key, "")
        if not value:
            continue
        if not is_relative_to(Path(value).expanduser(), archive):
            raise SafetyError(f"{key} must be archive-local, got {value}")


def resolve_codex_bin() -> Path:
    value = shutil.which("codex")
    if not value:
        raise SafetyError("codex binary not found on PATH")
    return Path(value).resolve()


def codex_version(codex_bin: Path) -> str:
    proc = subprocess.run(
        [str(codex_bin), "--version"],
        text=True,
        capture_output=True,
        check=False,
        timeout=10,
    )
    if proc.returncode != 0:
        return f"unavailable: {proc.stderr.strip()}"
    return proc.stdout.strip()


def validate_plasma_command(archive: Path, cmd: list[str], env: dict[str, str]) -> None:
    if len(cmd) < 2:
        raise SafetyError("Plasma command must include a subcommand")
    subcommand = cmd[1]
    if subcommand in {"--help", "-h", "help"}:
        return
    if subcommand not in DB_REQUIRED_COMMANDS and subcommand != "version":
        raise SafetyError(f"unexpected Plasma subcommand in experiment harness: {subcommand}")
    if subcommand in DB_REQUIRED_COMMANDS:
        validate_archive_db(archive, arg_value(cmd, "-db"))
    if subcommand == "serve":
        validate_loopback_experiment_addr(arg_value(cmd, "-addr"))
    if subcommand in AGENT_TOUCHING_COMMANDS:
        validate_provider_env(archive, env, required=True)


def safe_env(archive: Path, provider_home: Path | None = None) -> dict[str, str]:
    env = os.environ.copy()
    env["LONGFORM_ARCHIVE"] = str(archive)
    env["PLASMA_RUNTIME_MODE"] = "release"
    env["TMPDIR"] = str(archive / "tmp-harness")
    if provider_home is not None:
        env["CODEX_HOME"] = str(provider_home / "codex")
        env["XDG_CONFIG_HOME"] = str(provider_home / "xdg-config")
        env["XDG_DATA_HOME"] = str(provider_home / "xdg-data")
        env["XDG_CACHE_HOME"] = str(provider_home / "xdg-cache")
    env["CODEX_BIN"] = str(resolve_codex_bin())
    return env


def safe_run(
    archive: Path,
    run_id: str,
    cmd: list[str],
    *,
    env: dict[str, str],
    cwd: Path | None = None,
    timeout_seconds: int | None = None,
) -> subprocess.CompletedProcess[str]:
    validate_plasma_command(archive, cmd, env)
    started = utc_now()
    proc = subprocess.run(
        cmd,
        cwd=str(cwd or plasma_root()),
        env=env,
        text=True,
        capture_output=True,
        timeout=timeout_seconds,
    )
    completed = utc_now()
    log_prefix = archive / "logs" / run_id
    log_prefix.parent.mkdir(parents=True, exist_ok=True)
    (log_prefix.with_suffix(".stdout.log")).write_text(proc.stdout, encoding="utf-8")
    (log_prefix.with_suffix(".stderr.log")).write_text(proc.stderr, encoding="utf-8")
    append_jsonl(
        archive / "runs/safe-commands.jsonl",
        {
            "run_id": run_id,
            "cmd": cmd,
            "cwd": str(cwd or plasma_root()),
            "returncode": proc.returncode,
            "started_at": started,
            "completed_at": completed,
            "stdout_log": str(log_prefix.with_suffix(".stdout.log")),
            "stderr_log": str(log_prefix.with_suffix(".stderr.log")),
        },
    )
    if proc.returncode != 0:
        raise RuntimeError(f"command failed for {run_id}; see {log_prefix.with_suffix('.stderr.log')}")
    return proc


def safe_popen(
    archive: Path,
    run_id: str,
    cmd: list[str],
    *,
    env: dict[str, str],
    cwd: Path | None = None,
) -> subprocess.Popen[str]:
    validate_plasma_command(archive, cmd, env)
    log_prefix = archive / "servers" / run_id
    log_prefix.mkdir(parents=True, exist_ok=True)
    stdout_handle = (log_prefix / "server.stdout").open("w", encoding="utf-8")
    stderr_handle = (log_prefix / "server.stderr").open("w", encoding="utf-8")
    proc = subprocess.Popen(
        cmd,
        cwd=str(cwd or plasma_root()),
        env=env,
        text=True,
        stdout=stdout_handle,
        stderr=stderr_handle,
        start_new_session=True,
    )
    (log_prefix / "server.pid").write_text(str(proc.pid), encoding="utf-8")
    append_jsonl(
        archive / "runs/safe-commands.jsonl",
        {
            "run_id": run_id,
            "cmd": cmd,
            "cwd": str(cwd or plasma_root()),
            "pid": proc.pid,
            "started_at": utc_now(),
            "stdout_log": str(log_prefix / "server.stdout"),
            "stderr_log": str(log_prefix / "server.stderr"),
        },
    )
    return proc


def stop_process(proc: subprocess.Popen[str]) -> None:
    if proc.poll() is not None:
        return
    try:
        os.killpg(proc.pid, signal.SIGTERM)
    except ProcessLookupError:
        return
    try:
        proc.wait(timeout=10)
    except subprocess.TimeoutExpired:
        try:
            os.killpg(proc.pid, signal.SIGKILL)
        except ProcessLookupError:
            pass
        proc.wait(timeout=10)


def check_port_free(host: str, port: int) -> None:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.settimeout(0.5)
        if sock.connect_ex((host, port)) == 0:
            raise SafetyError(f"port already in use: {host}:{port}")


def http_json(url: str, *, method: str = "GET", payload: dict[str, Any] | None = None, timeout: int = 10) -> dict[str, Any]:
    data = None
    headers = {"Accept": "application/json"}
    if payload is not None:
        data = json.dumps(payload).encode("utf-8")
        headers["Content-Type"] = "application/json"
    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    with urllib.request.urlopen(req, timeout=timeout) as response:
        raw = response.read()
    return json.loads(raw.decode("utf-8"))


def wait_http_ready(base_url: str, timeout_seconds: int = 30) -> None:
    deadline = time.monotonic() + timeout_seconds
    last_error = ""
    while time.monotonic() < deadline:
        try:
            urllib.request.urlopen(base_url + "/", timeout=2).read()
            return
        except Exception as exc:  # noqa: BLE001 - readiness probe records all failures.
            last_error = str(exc)
            time.sleep(0.5)
    raise RuntimeError(f"server did not become ready: {last_error}")


def load_parity(archive: Path, sample_id: str) -> dict[str, Any]:
    path = archive / "analysis" / f"{sample_id}-research-parity.json"
    if not path.exists():
        raise FileNotFoundError(f"missing research parity manifest: {path}")
    return json.loads(path.read_text(encoding="utf-8"))


def require_archive_db_from_manifest(archive: Path, manifest: dict[str, Any], key: str) -> Path:
    value = str(manifest.get(key) or "")
    if not value:
        raise SafetyError(f"parity manifest missing {key}")
    return validate_archive_db(archive, value)


def copytree_replace(src: Path, dst: Path) -> None:
    if dst.exists():
        shutil.rmtree(dst)
    shutil.copytree(src, dst)


def copy_provider_auth_only(src_root: Path, dest_root: Path) -> None:
    src_codex = src_root / "codex" if (src_root / "codex").is_dir() else src_root
    dest_codex = dest_root / "codex"
    if dest_root.exists():
        shutil.rmtree(dest_root)
    for name in ("codex", "xdg-config", "xdg-data", "xdg-cache"):
        (dest_root / name).mkdir(parents=True, exist_ok=True)
    for rel in (
        "auth.json",
        "config.toml",
        "models_cache.json",
        "version.json",
        "rules/default.rules",
    ):
        src = src_codex / rel
        if not src.is_file():
            continue
        dst = dest_codex / rel
        dst.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(src, dst)
    forbidden = (
        "sessions",
        "history.jsonl",
        "session_index.jsonl",
        "logs",
        "logs_2.sqlite",
        "logs_2.sqlite-wal",
        "logs_2.sqlite-shm",
        "state_5.sqlite",
        "state_5.sqlite-wal",
        "state_5.sqlite-shm",
        "memories_1.sqlite",
        "goals_1.sqlite",
        "shell_snapshots",
    )
    copied = [str(dest_codex / rel) for rel in forbidden if (dest_codex / rel).exists()]
    if copied:
        raise SafetyError(f"forbidden provider conversation state copied: {copied}")


def validate_archive_path(archive: Path, path_text: str, name: str, *, must_exist: bool = False) -> Path:
    if not path_text:
        raise SafetyError(f"{name} is required")
    resolved = Path(path_text).expanduser().resolve()
    if not is_relative_to(resolved, archive):
        raise SafetyError(f"{name} must be archive-local, got {path_text}")
    if must_exist and not resolved.exists():
        raise SafetyError(f"{name} does not exist: {path_text}")
    return resolved


def reject_symlinks(path: Path, name: str) -> None:
    if path.is_symlink():
        raise SafetyError(f"{name} must not be a symlink: {path}")
    if path.is_dir():
        for child in path.rglob("*"):
            if child.is_symlink():
                raise SafetyError(f"{name} contains symlink: {child}")


def validate_local_source_root(archive: Path, spec: str) -> None:
    if "=" not in spec:
        raise SafetyError(f"local source root must be name=path, got {spec!r}")
    _name, path_text = spec.split("=", 1)
    validate_archive_path(archive, path_text, "local source root", must_exist=True)


def validate_b_runner_command(archive: Path, cmd: list[str], env: dict[str, str]) -> None:
    if len(cmd) < 2:
        raise SafetyError("B runner command requires a subcommand")
    runner = validate_archive_path(archive, cmd[0], "B runner", must_exist=True)
    expected_runner = archive / "tmp-runner" / "longform_b_runner"
    if runner != expected_runner.resolve():
        raise SafetyError(f"B runner must use archived runner, got {runner}")
    runner_text = runner.read_text(encoding="utf-8", errors="replace")
    if "plasma.workflow." in runner_text:
        raise SafetyError("B runner must not expose workflow tools")
    if cmd[1] not in {"draft-section", "assemble-part", "frame"}:
        raise SafetyError(f"unexpected B runner subcommand: {cmd[1]}")
    validate_archive_db(archive, arg_value(cmd, "--db"))
    validate_archive_path(archive, arg_value(cmd, "--plasma"), "plasma binary", must_exist=True)
    validate_archive_path(archive, arg_value(cmd, "--plan"), "plan", must_exist=True)
    validate_archive_path(archive, arg_value(cmd, "--prompt-snapshot"), "prompt snapshot", must_exist=True)
    validate_archive_path(archive, arg_value(cmd, "--agent-workdir"), "agent workdir")
    validate_archive_path(archive, arg_value(cmd, "--manifest"), "manifest")
    validate_local_source_root(archive, arg_value(cmd, "--local-source-root"))
    for flag in (
        "--out",
        "--connective-out",
        "--frame-out",
        "--section-hashes",
        "--part-hashes",
        "--sections-dir",
        "--parts-dir",
    ):
        value = arg_value(cmd, flag)
        if value:
            validate_archive_path(archive, value, flag)
    validate_provider_env(archive, env, required=True)


def safe_b_runner(
    archive: Path,
    run_id: str,
    cmd: list[str],
    *,
    env: dict[str, str],
    cwd: Path | None = None,
    timeout_seconds: int | None = None,
) -> subprocess.CompletedProcess[str]:
    validate_b_runner_command(archive, cmd, env)
    started = utc_now()
    proc = subprocess.run(
        cmd,
        cwd=str(cwd or plasma_root()),
        env=env,
        text=True,
        capture_output=True,
        timeout=timeout_seconds,
    )
    completed = utc_now()
    log_prefix = archive / "logs" / run_id
    log_prefix.parent.mkdir(parents=True, exist_ok=True)
    log_prefix.with_suffix(".stdout.log").write_text(proc.stdout, encoding="utf-8")
    log_prefix.with_suffix(".stderr.log").write_text(proc.stderr, encoding="utf-8")
    append_jsonl(
        archive / "runs/safe-commands.jsonl",
        {
            "run_id": run_id,
            "kind": "b_runner",
            "cmd": cmd,
            "codex_bin": env.get("CODEX_BIN", ""),
            "codex_version": codex_version(Path(env["CODEX_BIN"])),
            "cwd": str(cwd or plasma_root()),
            "returncode": proc.returncode,
            "started_at": started,
            "completed_at": completed,
            "stdout_log": str(log_prefix.with_suffix(".stdout.log")),
            "stderr_log": str(log_prefix.with_suffix(".stderr.log")),
        },
    )
    if proc.returncode != 0:
        raise RuntimeError(f"B runner failed for {run_id}; see {log_prefix.with_suffix('.stderr.log')}")
    return proc


def sqlite_event_types(db_path: Path) -> list[str]:
    with sqlite3.connect(f"file:{db_path}?mode=ro&immutable=1", uri=True) as conn:
        rows = conn.execute("SELECT event_type FROM plasma_ledger_events ORDER BY sequence").fetchall()
    return [row[0] for row in rows]


def sqlite_count(db_path: Path, table: str) -> int:
    if not re.fullmatch(r"[A-Za-z0-9_]+", table):
        raise SafetyError(f"invalid sqlite table name: {table}")
    with sqlite3.connect(f"file:{db_path}?mode=ro&immutable=1", uri=True) as conn:
        row = conn.execute(f"SELECT COUNT(*) FROM {table}").fetchone()
    return int(row[0] if row else 0)


def latest_a0_run_dir(archive: Path, sample_id: str) -> Path:
    candidates = sorted((archive / "runs" / sample_id).glob("A0-safe-smoke-*"), reverse=True)
    for candidate in candidates:
        if (candidate / "run.db").is_file():
            return candidate
    raise FileNotFoundError(f"missing A0-safe-smoke run for sample {sample_id}")


def latest_completed_b_run_dir(archive: Path, sample_id: str) -> Path:
    candidates = sorted((archive / "runs" / sample_id).glob("B-safe-smoke-*"), reverse=True)
    for candidate in candidates:
        result_path = candidate / "b-smoke-result.json"
        if not result_path.is_file():
            continue
        try:
            result = json.loads(result_path.read_text(encoding="utf-8"))
        except json.JSONDecodeError:
            continue
        if result.get("status") == "completed" and (candidate / "sections").is_dir() and (candidate / "plan.json").is_file():
            return candidate
    raise FileNotFoundError(f"missing completed B-safe-smoke run for sample {sample_id}")


def extract_latest_report_plan(db_path: Path) -> dict[str, Any]:
    with sqlite3.connect(db_path) as conn:
        row = conn.execute(
            """
            SELECT payload_json
            FROM plasma_ledger_events
            WHERE event_type = 'report.plan.created'
            ORDER BY sequence DESC
            LIMIT 1
            """
        ).fetchone()
    if row is None:
        raise SafetyError(f"missing report.plan.created event in {db_path}")
    payload = json.loads(row[0])
    plan = payload.get("plan")
    if not isinstance(plan, dict) or not isinstance(plan.get("parts"), list):
        raise SafetyError(f"report plan payload is malformed in {db_path}")
    return plan


def mission_id_from_file(path: Path) -> str:
    data = json.loads(path.read_text(encoding="utf-8"))
    for keys in (
        ("mission", "MissionID"),
        ("mission", "mission_id"),
        ("projection", "mission_id"),
        ("projection", "MissionID"),
    ):
        current: Any = data
        for key in keys:
            if not isinstance(current, dict):
                current = None
                break
            current = current.get(key)
        if isinstance(current, str) and current:
            return current
    raise SafetyError(f"missing mission id in {path}")


def response_session_id_from_file(path: Path) -> str:
    data = json.loads(path.read_text(encoding="utf-8"))
    payload = data.get("response_event", {}).get("Payload", {})
    if isinstance(payload, str):
        payload = json.loads(payload)
    if not isinstance(payload, dict):
        raise SafetyError(f"malformed response payload in {path}")
    session_id = str(payload.get("agent_session_id") or "")
    if not session_id:
        raise SafetyError(f"missing pre-report provider session id in {path}")
    return session_id


def count_matching_session_files(provider_home: Path, session_id: str) -> int:
    if not session_id:
        return 0
    return sum(1 for path in provider_home.rglob("*") if path.is_file() and session_id in path.name)


def write_research_parity_manifest(archive: Path, sample_id: str, mission_id: str, research_db: Path, a0_db: Path, b_base_db: Path, research_turn_path: Path, research_home: Path) -> dict[str, Any]:
    research_sha = sha256_file(research_db)
    a0_sha = sha256_file(a0_db)
    b_sha = sha256_file(b_base_db)
    session_id = response_session_id_from_file(research_turn_path)
    manifest = {
        "sample_id": sample_id,
        "mission_id": mission_id,
        "research_db": str(research_db),
        "a0_db": str(a0_db),
        "b_base_db": str(b_base_db),
        "research_db_sha256": research_sha,
        "a0_db_sha256": a0_sha,
        "b_base_db_sha256": b_sha,
        "db_split_byte_identical": research_sha == a0_sha == b_sha,
        "source_snapshot_count": sqlite_count(research_db, "plasma_source_snapshots"),
        "raw_artifact_count": sqlite_count(research_db, "plasma_raw_artifacts"),
        "ledger_event_count": sqlite_count(research_db, "plasma_ledger_events"),
        "pre_report_provider_session_id": session_id,
        "matching_research_session_files": count_matching_session_files(research_home, session_id),
    }
    write_json(archive / "analysis" / f"{sample_id}-research-parity.json", manifest)
    return manifest


def prepare_sample(args: argparse.Namespace) -> None:
    archive = resolve_archive(args.archive)
    ensure_archive_dirs(archive)
    plasma_bin = Path(args.plasma).expanduser().resolve() if args.plasma else archive / "bin" / "plasma"
    if not plasma_bin.exists():
        raise SystemExit(f"missing archive Plasma binary: {plasma_bin}")
    results = collect_preflight_results(archive, plasma_bin, args.sample_id, require_parity=False)
    if not all(result.ok for result in results):
        write_json(
            archive / "analysis" / f"{args.sample_id}-prepare-preflight.json",
            {
                "archive": str(archive),
                "plasma": str(plasma_bin),
                "sample_id": args.sample_id,
                "checked_at": utc_now(),
                "results": [result.__dict__ for result in results],
                "ok": False,
            },
        )
        raise SystemExit(1)

    block_dir = archive / "blocks" / args.sample_id
    seed_db = validate_archive_path(archive, str(block_dir / "seed.db"), "seed DB", must_exist=True)
    mission_path = validate_archive_path(archive, str(block_dir / "mission.json"), "mission file", must_exist=True)
    source_root = validate_archive_path(archive, str(archive / "source-roots" / args.sample_id), "source root", must_exist=True)
    reject_symlinks(source_root, "source root")
    mission_id = mission_id_from_file(mission_path)

    research_db = block_dir / "research.db"
    for suffix in ("", "-wal", "-shm"):
        target = Path(str(research_db) + suffix)
        if target.exists():
            target.unlink()
    shutil.copyfile(seed_db, research_db)

    research_home = archive / "provider-homes" / args.sample_id / "research"
    auth_source = validate_archive_path(archive, str(args.auth_provider_home.expanduser().resolve()), "auth provider home", must_exist=True)
    reject_symlinks(auth_source, "auth provider home")
    copy_provider_auth_only(auth_source, research_home)

    prompt_path = validate_archive_path(archive, str(args.research_prompt.expanduser().resolve()), "research prompt", must_exist=True)
    prompt_text = prompt_path.read_text(encoding="utf-8")
    workdir = archive / "workdirs" / f"{args.sample_id}-research"
    if workdir.exists():
        shutil.rmtree(workdir)
    workdir.mkdir(parents=True, exist_ok=True)
    root_spec = f"sample={source_root}"
    env = safe_env(archive, research_home)
    proc = safe_run(
        archive,
        f"{args.sample_id}-research-turn",
        [
            str(plasma_bin),
            "turns",
            "send",
            mission_id,
            "-db", str(research_db),
            "-local-source-root", root_spec,
            "-text", prompt_text,
            "-agent", args.agent,
            "-mcp-mode", "auto",
            "-agent-workdir", str(workdir),
            "-agent-timeout", args.agent_timeout,
            "-wait",
            "-json",
        ],
        env=env,
        cwd=plasma_root(),
        timeout_seconds=parse_duration_seconds(args.agent_timeout),
    )
    research_turn_path = block_dir / "research-turn.json"
    research_turn_path.write_text(proc.stdout, encoding="utf-8")

    a0_db = archive / "runs" / args.sample_id / "A0" / "run.db"
    b_base_db = archive / "runs" / args.sample_id / "B" / "base.db"
    a0_db.parent.mkdir(parents=True, exist_ok=True)
    b_base_db.parent.mkdir(parents=True, exist_ok=True)
    shutil.copyfile(research_db, a0_db)
    shutil.copyfile(research_db, b_base_db)
    manifest = write_research_parity_manifest(archive, args.sample_id, mission_id, research_db, a0_db, b_base_db, research_turn_path, research_home)
    out = {
        "status": "completed",
        "sample_id": args.sample_id,
        "mission_id": mission_id,
        "research_db": str(research_db),
        "research_turn": str(research_turn_path),
        "parity_manifest": str(archive / "analysis" / f"{args.sample_id}-research-parity.json"),
        "manifest": manifest,
        "completed_at": utc_now(),
    }
    write_json(block_dir / "prepare-sample-result.json", out)
    print(json.dumps(out, ensure_ascii=False, indent=2))


def prompt_snapshot_dir(archive: Path, explicit: Path | None = None) -> Path:
    if explicit is not None:
        return validate_archive_path(archive, str(explicit), "prompt snapshot dir", must_exist=True)
    base = archive / "prompts" / "product-snapshots"
    children = sorted(path for path in base.iterdir() if path.is_dir()) if base.is_dir() else []
    if len(children) != 1:
        raise SafetyError(f"expected exactly one prompt snapshot dir under {base}, found {len(children)}")
    required = {"section-draft.md", "part-assembly.md", "frame.md", "generation-guidance-g2.md"}
    missing = sorted(name for name in required if not (children[0] / name).is_file())
    if missing:
        raise SafetyError(f"prompt snapshot is missing files: {', '.join(missing)}")
    return children[0]


def prepare_b_run(archive: Path, sample_id: str, run_label: str, a0_run_dir: Path | None, prompt_dir: Path | None) -> dict[str, Any]:
    parity = load_parity(archive, sample_id)
    b_base_db = require_archive_db_from_manifest(archive, parity, "b_base_db")
    mission_id = parity["mission_id"]
    run_dir = archive / "runs" / sample_id / run_label
    base_db_path = run_dir / "base.db"
    if run_dir.exists():
        shutil.rmtree(run_dir)
    for name in ("sections", "parts", "connective", "frame", "provider-homes", "workdirs", "dbs"):
        (run_dir / name).mkdir(parents=True, exist_ok=True)
    shutil.copyfile(b_base_db, base_db_path)

    source_a0 = validate_archive_path(archive, str(a0_run_dir or latest_a0_run_dir(archive, sample_id)), "A0 run dir", must_exist=True)
    plan = extract_latest_report_plan(source_a0 / "run.db")
    plan_path = run_dir / "plan.json"
    write_json(plan_path, plan)

    research_home = archive / "provider-homes" / sample_id / "research"
    if not research_home.exists():
        raise FileNotFoundError(f"missing research provider home: {research_home}")
    research_home = validate_archive_path(archive, str(research_home), "research provider home", must_exist=True)
    reject_symlinks(research_home, "research provider home")
    return {
        "mission_id": mission_id,
        "title": f"{sample_id} long-form report",
        "base_db_path": base_db_path,
        "run_dir": run_dir,
        "plan": plan,
        "plan_path": plan_path,
        "prompt_dir": prompt_snapshot_dir(archive, prompt_dir),
        "research_home": research_home,
        "root_spec": f"sample={archive / 'source-roots' / sample_id}",
    }


def prepare_b_reframe_run(archive: Path, sample_id: str, run_label: str, source_b_run_dir: Path | None, prompt_dir: Path | None) -> dict[str, Any]:
    parity = load_parity(archive, sample_id)
    b_base_db = require_archive_db_from_manifest(archive, parity, "b_base_db")
    mission_id = parity["mission_id"]
    run_dir = archive / "runs" / sample_id / run_label
    base_db_path = run_dir / "base.db"
    if run_dir.exists():
        shutil.rmtree(run_dir)
    for name in ("sections", "parts", "connective", "frame", "provider-homes", "workdirs", "dbs"):
        (run_dir / name).mkdir(parents=True, exist_ok=True)
    shutil.copyfile(b_base_db, base_db_path)

    source_run = validate_archive_path(
        archive,
        str(source_b_run_dir or latest_completed_b_run_dir(archive, sample_id)),
        "source B run dir",
        must_exist=True,
    )
    reject_symlinks(source_run / "sections", "source B sections")
    shutil.copytree(source_run / "sections", run_dir / "sections", dirs_exist_ok=True)
    shutil.copyfile(source_run / "plan.json", run_dir / "plan.json")
    plan = json.loads((run_dir / "plan.json").read_text(encoding="utf-8"))

    research_home = archive / "provider-homes" / sample_id / "research"
    if not research_home.exists():
        raise FileNotFoundError(f"missing research provider home: {research_home}")
    research_home = validate_archive_path(archive, str(research_home), "research provider home", must_exist=True)
    reject_symlinks(research_home, "research provider home")
    return {
        "mission_id": mission_id,
        "title": f"{sample_id} long-form report",
        "base_db_path": base_db_path,
        "run_dir": run_dir,
        "source_b_run_dir": source_run,
        "plan": plan,
        "plan_path": run_dir / "plan.json",
        "prompt_dir": prompt_snapshot_dir(archive, prompt_dir),
        "research_home": research_home,
        "root_spec": f"sample={archive / 'source-roots' / sample_id}",
    }


def scan_for_unsafe_serve(archive: Path) -> list[SafetyResult]:
    results: list[SafetyResult] = []
    scan_roots = [archive / "tmp-runner", archive / "servers"]
    pattern = re.compile(r"(\bplasma\b|\$[{]?[A-Z_]*PLASMA[}]?|ARCHIVE/bin/plasma)[\"']?\s+serve\b")
    for root in scan_roots:
        if not root.exists():
            continue
        for path in root.rglob("*"):
            if not path.is_file() or path.stat().st_size > 2_000_000:
                continue
            if path.suffix.lower() not in {"", ".sh", ".py", ".json", ".jsonl", ".command", ".cmd"}:
                continue
            try:
                text = path.read_text(encoding="utf-8", errors="replace")
            except OSError:
                continue
            logical_text = re.sub(r"\\\n\s*", " ", text)
            for line_number, line in enumerate(logical_text.splitlines(), start=1):
                if "serve" not in line:
                    continue
                if not pattern.search(line):
                    continue
                if "safe-plasma" not in line and "SAFE_PLASMA" not in line:
                    results.append(SafetyResult("unsafe_serve_static_scan", False, f"{path}:{line_number}: raw serve bypasses wrapper: {line.strip()}"))
                    continue
                if "-db" not in line or "-addr" not in line:
                    results.append(SafetyResult("unsafe_serve_static_scan", False, f"{path}:{line_number}: serve is missing -db or -addr: {line.strip()}"))
    return results


def run_internal_safety_tests(archive: Path, plasma_bin: Path) -> list[SafetyResult]:
    tests: list[tuple[str, list[str], dict[str, str], bool]] = [
        (
            "reject_argumentless_serve",
            [str(plasma_bin), "serve"],
            safe_env(archive, archive / "provider-homes/test"),
            False,
        ),
        (
            "reject_default_db",
            [str(plasma_bin), "serve", "-db", str(Path.home() / "Library/Application Support/Plasma/plasma.db"), "-addr", "127.0.0.1:6202"],
            safe_env(archive, archive / "provider-homes/test"),
            False,
        ),
        (
            "reject_release_port",
            [str(plasma_bin), "serve", "-db", str(archive / "tmp-harness/test.db"), "-addr", "127.0.0.1:3002"],
            safe_env(archive, archive / "provider-homes/test"),
            False,
        ),
        (
            "reject_non_loopback",
            [str(plasma_bin), "serve", "-db", str(archive / "tmp-harness/test.db"), "-addr", "100.64.0.1:6202"],
            safe_env(archive, archive / "provider-homes/test"),
            False,
        ),
        (
            "accept_archive_loopback_serve",
            [str(plasma_bin), "serve", "-db", str(archive / "tmp-harness/test.db"), "-addr", "127.0.0.1:6202"],
            safe_env(archive, archive / "provider-homes/test"),
            True,
        ),
        (
            "reject_duplicate_db_override",
            [
                str(plasma_bin),
                "serve",
                "-db", str(archive / "tmp-harness/test.db"),
                "-addr", "127.0.0.1:6202",
                "-db", str(Path.home() / "Library/Application Support/Plasma/plasma.db"),
            ],
            safe_env(archive, archive / "provider-homes/test"),
            False,
        ),
        (
            "reject_duplicate_addr_override",
            [
                str(plasma_bin),
                "serve",
                "-db", str(archive / "tmp-harness/test.db"),
                "-addr", "127.0.0.1:6202",
                "-addr", "127.0.0.1:3002",
            ],
            safe_env(archive, archive / "provider-homes/test"),
            False,
        ),
    ]
    results: list[SafetyResult] = []
    for name, cmd, env, should_accept in tests:
        try:
            validate_plasma_command(archive, cmd, env)
            accepted = True
            detail = "accepted"
        except Exception as exc:  # noqa: BLE001 - this is a safety-test harness.
            accepted = False
            detail = str(exc)
        ok = accepted == should_accept
        results.append(SafetyResult(name, ok, detail))
    return results


def collect_preflight_results(archive: Path, plasma_bin: Path, sample_id: str, *, require_parity: bool) -> list[SafetyResult]:
    results = run_internal_safety_tests(archive, plasma_bin)
    results.extend(scan_for_unsafe_serve(archive))
    running = subprocess.run(
        ["ps", "-axo", "pid,command"],
        text=True,
        capture_output=True,
        check=False,
    )
    suspicious = [
        line for line in running.stdout.splitlines()
        if str(archive) in line and any(marker in line for marker in ("plasma serve", "longform_b_runner", "codex exec", "plasma mcp"))
    ]
    results.append(SafetyResult("no_archive_processes_running", not suspicious, "\n".join(suspicious) or "none"))
    runner_path = archive / "tmp-runner" / "longform_b_runner"
    if runner_path.exists():
        runner_text = runner_path.read_text(encoding="utf-8", errors="replace")
        results.append(SafetyResult("b_runner_workflow_tools_disabled", "plasma.workflow." not in runner_text, str(runner_path)))
    parity_path = archive / "analysis" / f"{sample_id}-research-parity.json"
    if not require_parity:
        return results
    results.append(SafetyResult("sample_parity_manifest_exists", parity_path.exists(), str(parity_path)))
    if parity_path.exists():
        try:
            parity = load_parity(archive, sample_id)
            for key in ("research_db", "a0_db", "b_base_db"):
                require_archive_db_from_manifest(archive, parity, key)
            results.append(SafetyResult("sample_parity_db_paths_archive_local", True, str(parity_path)))
        except Exception as exc:  # noqa: BLE001 - preflight must report every safety failure.
            results.append(SafetyResult("sample_parity_db_paths_archive_local", False, str(exc)))
    return results


def preflight(args: argparse.Namespace) -> None:
    archive = resolve_archive(args.archive)
    ensure_archive_dirs(archive)
    plasma_bin = Path(args.plasma).expanduser().resolve() if args.plasma else archive / "bin" / "plasma"
    if not plasma_bin.exists():
        raise SystemExit(f"missing archive Plasma binary: {plasma_bin}")
    results = collect_preflight_results(archive, plasma_bin, args.sample_id, require_parity=True)
    out = {
        "archive": str(archive),
        "plasma": str(plasma_bin),
        "sample_id": args.sample_id,
        "checked_at": utc_now(),
        "results": [result.__dict__ for result in results],
        "ok": all(result.ok for result in results),
    }
    write_json(archive / "analysis" / "safety-preflight.json", out)
    print(json.dumps(out, ensure_ascii=False, indent=2))
    if not out["ok"]:
        raise SystemExit(1)


def prepare_a0_run(archive: Path, sample_id: str, run_label: str) -> dict[str, Any]:
    parity = load_parity(archive, sample_id)
    research_db = require_archive_db_from_manifest(archive, parity, "research_db")
    require_archive_db_from_manifest(archive, parity, "a0_db")
    require_archive_db_from_manifest(archive, parity, "b_base_db")
    mission_id = parity["mission_id"]
    run_dir = archive / "runs" / sample_id / run_label
    db_path = run_dir / "run.db"
    if run_dir.exists():
        shutil.rmtree(run_dir)
    run_dir.mkdir(parents=True, exist_ok=True)
    shutil.copyfile(research_db, db_path)
    research_home = archive / "provider-homes" / sample_id / "research"
    provider_home = archive / "provider-homes" / sample_id / run_label
    if not research_home.exists():
        raise FileNotFoundError(f"missing research provider home: {research_home}")
    copytree_replace(research_home, provider_home)
    return {
        "mission_id": mission_id,
        "title": f"{sample_id} long-form report",
        "db_path": db_path,
        "run_dir": run_dir,
        "provider_home": provider_home,
        "root_spec": f"sample={archive / 'source-roots' / sample_id}",
    }


def field_value(mapping: dict[str, Any], *names: str) -> Any:
    for name in names:
        if name in mapping:
            return mapping[name]
    return None


def normalize_event(event: dict[str, Any]) -> tuple[str, dict[str, Any]]:
    event_type = field_value(event, "event_type", "EventType", "type", "Type")
    payload = field_value(event, "payload", "Payload", "payload_json", "PayloadJSON")
    if isinstance(payload, str):
        try:
            payload = json.loads(payload)
        except json.JSONDecodeError:
            payload = {}
    if not isinstance(payload, dict):
        payload = {}
    return str(event_type or ""), payload


def nested_id(mapping: dict[str, Any], *paths: tuple[str, ...]) -> str:
    for path in paths:
        current: Any = mapping
        for key in path:
            if not isinstance(current, dict):
                current = None
                break
            current = field_value(current, key, key[:1].upper() + key[1:])
        if isinstance(current, str) and current:
            return current
    return ""


def parse_duration_seconds(value: str) -> int | None:
    text = (value or "").strip().lower()
    if not text:
        return None
    match = re.fullmatch(r"(\d+(?:\.\d+)?)(ms|s|m|h)?", text)
    if not match:
        raise SafetyError(f"invalid duration: {value}")
    amount = float(match.group(1))
    unit = match.group(2) or "s"
    if unit == "ms":
        return max(1, int(amount / 1000))
    if unit == "s":
        return int(amount)
    if unit == "m":
        return int(amount * 60)
    if unit == "h":
        return int(amount * 3600)
    raise SafetyError(f"invalid duration unit: {value}")


def stage_env(archive: Path, research_home: Path, stage_home: Path) -> dict[str, str]:
    copy_provider_auth_only(research_home, stage_home)
    return safe_env(archive, stage_home)


def run_b_stage(
    archive: Path,
    prepared: dict[str, Any],
    run_label: str,
    stage_name: str,
    subcommand: str,
    extra_args: list[str],
    agent_timeout: str,
) -> None:
    stage_home = prepared["run_dir"] / "provider-homes" / stage_name
    workdir = prepared["run_dir"] / "workdirs" / stage_name
    stage_db = prepared["run_dir"] / "dbs" / stage_name / "run.db"
    stage_db.parent.mkdir(parents=True, exist_ok=True)
    shutil.copyfile(prepared["base_db_path"], stage_db)
    env = stage_env(archive, prepared["research_home"], stage_home)
    cmd = [
        str(archive / "tmp-runner" / "longform_b_runner"),
        subcommand,
        "--plasma", str(archive / "bin" / "plasma"),
        "--db", str(stage_db),
        "--mission-id", prepared["mission_id"],
        "--local-source-root", prepared["root_spec"],
        "--plan", str(prepared["plan_path"]),
        "--title", prepared["title"],
        "--rigor-level", "balanced",
        "--generation-guidance-profile", "g2",
        "--prompt-snapshot", str(prepared["prompt_dir"] / {
            "draft-section": "section-draft.md",
            "assemble-part": "part-assembly.md",
            "frame": "frame.md",
        }[subcommand]),
        "--agent-workdir", str(workdir),
        "--agent-timeout", agent_timeout,
        "--manifest", str(prepared["run_dir"] / "manifest.jsonl"),
        *extra_args,
    ]
    safe_b_runner(
        archive,
        f"{run_label}-{stage_name}",
        cmd,
        env=env,
        cwd=plasma_root(),
        timeout_seconds=parse_duration_seconds(agent_timeout),
    )


def stage_db_has_workflow_events(db_path: Path) -> bool:
    with sqlite3.connect(db_path) as conn:
        row = conn.execute("SELECT COUNT(*) FROM plasma_ledger_events WHERE event_type LIKE 'workflow.%'").fetchone()
    return bool(row and row[0])


def validate_b_completion(archive: Path, prepared: dict[str, Any], rows: list[dict[str, Any]]) -> dict[str, int]:
    plan = prepared["plan"]
    run_dir = prepared["run_dir"]
    expected_sections = sum(len(part.get("sections", [])) for part in plan["parts"])
    expected_parts = len(plan["parts"])
    counts = {
        name: sum(1 for row in rows if row.get("record_type") == name)
        for name in ("report.section.created", "report.part.created", "report.artifact.created", "report.stage.failed")
    }
    failures: list[str] = []
    if counts["report.section.created"] != expected_sections:
        failures.append(f"expected {expected_sections} sections, got {counts['report.section.created']}")
    if counts["report.part.created"] != expected_parts:
        failures.append(f"expected {expected_parts} parts, got {counts['report.part.created']}")
    if counts["report.artifact.created"] != 1:
        failures.append(f"expected 1 final artifact, got {counts['report.artifact.created']}")
    if counts["report.stage.failed"] != 0:
        failures.append(f"stage failures recorded: {counts['report.stage.failed']}")
    expected_section_keys = {
        (part_index, section_index)
        for part_index, part in enumerate(plan["parts"], start=1)
        for section_index, _section in enumerate(part.get("sections", []), start=1)
    }
    actual_section_keys = {
        (int(row.get("part_index") or 0), int(row.get("section_index") or 0))
        for row in rows
        if row.get("record_type") == "report.section.created"
    }
    if actual_section_keys != expected_section_keys:
        failures.append(
            "section identity mismatch: "
            f"missing={sorted(expected_section_keys - actual_section_keys)} "
            f"extra={sorted(actual_section_keys - expected_section_keys)}"
        )
    expected_part_keys = set(range(1, expected_parts + 1))
    actual_part_keys = {
        int(row.get("part_index") or 0)
        for row in rows
        if row.get("record_type") == "report.part.created"
    }
    if actual_part_keys != expected_part_keys:
        failures.append(
            "part identity mismatch: "
            f"missing={sorted(expected_part_keys - actual_part_keys)} "
            f"extra={sorted(actual_part_keys - expected_part_keys)}"
        )
    final_path = run_dir / "final.md"
    if not final_path.is_file() or final_path.stat().st_size == 0:
        failures.append(f"missing final artifact: {final_path}")
    workflow_dbs: list[str] = []
    for row in rows:
        db_text = str(row.get("db_path") or "")
        if not db_text:
            continue
        db_path = validate_archive_path(archive, db_text, "B stage db", must_exist=True)
        if not is_relative_to(db_path, run_dir / "dbs"):
            failures.append(f"B stage db outside run dbs dir: {db_path}")
        if stage_db_has_workflow_events(db_path):
            workflow_dbs.append(str(db_path))
    if workflow_dbs:
        failures.append(f"workflow events appeared in B stage DBs: {workflow_dbs}")
    if failures:
        raise SafetyError("; ".join(failures))
    return counts


def b_smoke(args: argparse.Namespace) -> None:
    archive = resolve_archive(args.archive)
    ensure_archive_dirs(archive)
    plasma_bin = Path(args.plasma).expanduser().resolve() if args.plasma else archive / "bin" / "plasma"
    preflight_args = argparse.Namespace(archive=archive, plasma=plasma_bin, sample_id=args.sample_id)
    preflight(preflight_args)

    run_label = f"B-safe-smoke-{datetime.now(timezone.utc).strftime('%Y%m%d%H%M%S')}"
    prepared = prepare_b_run(
        archive,
        args.sample_id,
        run_label,
        args.a0_run_dir.expanduser().resolve() if args.a0_run_dir else None,
        args.prompt_snapshot_dir.expanduser().resolve() if args.prompt_snapshot_dir else None,
    )
    plan = prepared["plan"]
    started = utc_now()
    try:
        for part_index, part in enumerate(plan["parts"], start=1):
            for section_index, _section in enumerate(part.get("sections", []), start=1):
                run_b_stage(
                    archive,
                    prepared,
                    run_label,
                    f"P{part_index:02d}-S{section_index:02d}",
                    "draft-section",
                    [
                        "--part-index", str(part_index),
                        "--section-index", str(section_index),
                        "--out", str(prepared["run_dir"] / "sections" / f"P{part_index:02d}-S{section_index:02d}.md"),
                    ],
                    args.agent_timeout,
                )
        for part_index, _part in enumerate(plan["parts"], start=1):
            run_b_stage(
                archive,
                prepared,
                run_label,
                f"P{part_index:02d}",
                "assemble-part",
                [
                    "--part-index", str(part_index),
                    "--sections-dir", str(prepared["run_dir"] / "sections"),
                    "--section-hashes", str(prepared["run_dir"] / "connective" / f"P{part_index:02d}-section-hashes.json"),
                    "--out", str(prepared["run_dir"] / "parts" / f"P{part_index:02d}.md"),
                    "--connective-out", str(prepared["run_dir"] / "connective" / f"P{part_index:02d}.json"),
                ],
                args.agent_timeout,
            )
        run_b_stage(
            archive,
            prepared,
            run_label,
            "frame",
            "frame",
            [
                "--parts-dir", str(prepared["run_dir"] / "parts"),
                "--part-hashes", str(prepared["run_dir"] / "frame" / "part-hashes.json"),
                "--out", str(prepared["run_dir"] / "final.md"),
                "--frame-out", str(prepared["run_dir"] / "frame" / "frame.json"),
            ],
            args.agent_timeout,
        )
        manifest_rows = []
        manifest_path = prepared["run_dir"] / "manifest.jsonl"
        if manifest_path.exists():
            manifest_rows = [json.loads(line) for line in manifest_path.read_text(encoding="utf-8").splitlines() if line.strip()]
        record_counts = validate_b_completion(archive, prepared, manifest_rows)
        result = {
            "status": "completed",
            "sample_id": args.sample_id,
            "run_label": run_label,
            "mission_id": prepared["mission_id"],
            "variant": "B-independent-sections",
            "base_db_path": str(prepared["base_db_path"]),
            "run_dir": str(prepared["run_dir"]),
            "plan_path": str(prepared["plan_path"]),
            "manifest_path": str(manifest_path),
            "final_path": str(prepared["run_dir"] / "final.md"),
            "section_count": sum(len(part.get("sections", [])) for part in plan["parts"]),
            "part_count": len(plan["parts"]),
            "record_counts": record_counts,
            "started_at": started,
            "completed_at": utc_now(),
        }
        write_json(prepared["run_dir"] / "b-smoke-result.json", result)
        print(json.dumps(result, ensure_ascii=False, indent=2))
    except Exception as exc:
        result = {
            "status": "failed",
            "sample_id": args.sample_id,
            "run_label": run_label,
            "mission_id": prepared["mission_id"],
            "variant": "B-independent-sections",
            "base_db_path": str(prepared["base_db_path"]),
            "run_dir": str(prepared["run_dir"]),
            "error": str(exc),
            "started_at": started,
            "failed_at": utc_now(),
        }
        write_json(prepared["run_dir"] / "b-smoke-result.json", result)
        raise


def b_reframe(args: argparse.Namespace) -> None:
    archive = resolve_archive(args.archive)
    ensure_archive_dirs(archive)
    plasma_bin = Path(args.plasma).expanduser().resolve() if args.plasma else archive / "bin" / "plasma"
    preflight_args = argparse.Namespace(archive=archive, plasma=plasma_bin, sample_id=args.sample_id)
    preflight(preflight_args)

    run_prefix = getattr(args, "run_label_prefix", "") or "C1-reframe"
    run_label = f"{run_prefix}-{datetime.now(timezone.utc).strftime('%Y%m%d%H%M%S')}"
    prepared = prepare_b_reframe_run(
        archive,
        args.sample_id,
        run_label,
        args.source_b_run_dir.expanduser().resolve() if args.source_b_run_dir else None,
        args.prompt_snapshot_dir.expanduser().resolve() if args.prompt_snapshot_dir else None,
    )
    plan = prepared["plan"]
    started = utc_now()
    source_hashes = {
        str(path.relative_to(prepared["run_dir"] / "sections")): sha256_file(path)
        for path in sorted((prepared["run_dir"] / "sections").glob("*.md"))
    }
    try:
        for part_index, _part in enumerate(plan["parts"], start=1):
            run_b_stage(
                archive,
                prepared,
                run_label,
                f"P{part_index:02d}",
                "assemble-part",
                [
                    "--part-index", str(part_index),
                    "--sections-dir", str(prepared["run_dir"] / "sections"),
                    "--section-hashes", str(prepared["run_dir"] / "connective" / f"P{part_index:02d}-section-hashes.json"),
                    "--out", str(prepared["run_dir"] / "parts" / f"P{part_index:02d}.md"),
                    "--connective-out", str(prepared["run_dir"] / "connective" / f"P{part_index:02d}.json"),
                ],
                args.agent_timeout,
            )
        run_b_stage(
            archive,
            prepared,
            run_label,
            "frame",
            "frame",
            [
                "--parts-dir", str(prepared["run_dir"] / "parts"),
                "--part-hashes", str(prepared["run_dir"] / "frame" / "part-hashes.json"),
                "--out", str(prepared["run_dir"] / "final.md"),
                "--frame-out", str(prepared["run_dir"] / "frame" / "frame.json"),
            ],
            args.agent_timeout,
        )
        manifest_path = prepared["run_dir"] / "manifest.jsonl"
        manifest_rows = [json.loads(line) for line in manifest_path.read_text(encoding="utf-8").splitlines() if line.strip()] if manifest_path.exists() else []
        counts = {
            name: sum(1 for row in manifest_rows if row.get("record_type") == name)
            for name in ("report.part.created", "report.artifact.created", "report.stage.failed")
        }
        expected_parts = len(plan["parts"])
        current_hashes = {
            str(path.relative_to(prepared["run_dir"] / "sections")): sha256_file(path)
            for path in sorted((prepared["run_dir"] / "sections").glob("*.md"))
        }
        failures: list[str] = []
        if counts["report.part.created"] != expected_parts:
            failures.append(f"expected {expected_parts} parts, got {counts['report.part.created']}")
        if counts["report.artifact.created"] != 1:
            failures.append(f"expected 1 final artifact, got {counts['report.artifact.created']}")
        if counts["report.stage.failed"] != 0:
            failures.append(f"stage failures recorded: {counts['report.stage.failed']}")
        if source_hashes != current_hashes:
            failures.append("section hashes changed during reframe")
        if not (prepared["run_dir"] / "final.md").is_file():
            failures.append("missing final.md")
        if failures:
            raise SafetyError("; ".join(failures))
        result = {
            "status": "completed",
            "sample_id": args.sample_id,
            "run_label": run_label,
            "mission_id": prepared["mission_id"],
            "variant": run_prefix,
            "source_b_run_dir": str(prepared["source_b_run_dir"]),
            "base_db_path": str(prepared["base_db_path"]),
            "run_dir": str(prepared["run_dir"]),
            "plan_path": str(prepared["plan_path"]),
            "manifest_path": str(manifest_path),
            "final_path": str(prepared["run_dir"] / "final.md"),
            "section_count": sum(len(part.get("sections", [])) for part in plan["parts"]),
            "part_count": expected_parts,
            "record_counts": counts,
            "section_hashes_preserved": True,
            "started_at": started,
            "completed_at": utc_now(),
        }
        write_json(prepared["run_dir"] / "reframe-result.json", result)
        print(json.dumps(result, ensure_ascii=False, indent=2))
    except Exception as exc:
        result = {
            "status": "failed",
            "sample_id": args.sample_id,
            "run_label": run_label,
            "mission_id": prepared["mission_id"],
            "variant": run_prefix,
            "source_b_run_dir": str(prepared["source_b_run_dir"]),
            "base_db_path": str(prepared["base_db_path"]),
            "run_dir": str(prepared["run_dir"]),
            "error": str(exc),
            "started_at": started,
            "failed_at": utc_now(),
        }
        write_json(prepared["run_dir"] / "reframe-result.json", result)
        raise


def a0_smoke(args: argparse.Namespace) -> None:
    archive = resolve_archive(args.archive)
    ensure_archive_dirs(archive)
    plasma_bin = Path(args.plasma).expanduser().resolve() if args.plasma else archive / "bin" / "plasma"
    preflight_args = argparse.Namespace(archive=archive, plasma=plasma_bin, sample_id=args.sample_id)
    preflight(preflight_args)
    host = "127.0.0.1"
    port = args.port
    validate_loopback_experiment_addr(f"{host}:{port}")
    check_port_free(host, port)
    run_label = f"A0-safe-smoke-{datetime.now(timezone.utc).strftime('%Y%m%d%H%M%S')}"
    prepared = prepare_a0_run(archive, args.sample_id, run_label)
    env = safe_env(archive, prepared["provider_home"])
    workdir = archive / "workdirs" / f"{args.sample_id}-{run_label}"
    workdir.mkdir(parents=True, exist_ok=True)
    cmd = [
        str(plasma_bin),
        "serve",
        "-db", str(prepared["db_path"]),
        "-addr", f"{host}:{port}",
        "-agent", args.agent,
        "-agent-workdir", str(workdir),
        "-agent-timeout", args.agent_timeout,
        "-local-source-root", prepared["root_spec"],
    ]
    proc = safe_popen(archive, f"{args.sample_id}-{run_label}", cmd, env=env, cwd=plasma_root())
    base_url = f"http://{host}:{port}"
    try:
        wait_http_ready(base_url, timeout_seconds=30)
        request_payload = {
            "title": prepared["title"],
            "report_mode": "long_form",
            "report_session_policy": "isolated_fork",
            "post_report_humanize": "disabled",
            "agent_executor": args.agent,
            "mcp_mode": "auto",
            "rigor_level": "balanced",
            "generation_guidance_profile": "g2",
        }
        start = http_json(
            f"{base_url}/api/missions/{prepared['mission_id']}/reports",
            method="POST",
            payload=request_payload,
            timeout=20,
        )
        write_json(prepared["run_dir"] / "report-start.json", start)
        pending_event_id = nested_id(start, ("pending_event", "event_id"), ("PendingEvent", "EventID"), ("event", "event_id"), ("Event", "EventID"))
        deadline = time.monotonic() + args.wait_seconds
        terminal: dict[str, Any] | None = None
        while time.monotonic() < deadline:
            detail = http_json(f"{base_url}/api/missions/{prepared['mission_id']}", timeout=20)
            write_json(prepared["run_dir"] / "mission-detail-latest.json", detail)
            events = detail.get("events") or detail.get("ledger_events") or []
            for event in events:
                event_type, payload = normalize_event(event)
                if pending_event_id and payload.get("pending_event_id") != pending_event_id:
                    continue
                if event_type in {"report.artifact.created", "report.draft.failed"}:
                    terminal = event
                    break
            if terminal is not None:
                break
            if proc.poll() is not None:
                break
            time.sleep(args.poll_interval)
        final_detail = http_json(f"{base_url}/api/missions/{prepared['mission_id']}", timeout=20)
        write_json(prepared["run_dir"] / "mission-detail-final.json", final_detail)
        event_types = sqlite_event_types(prepared["db_path"])
        result = {
            "sample_id": args.sample_id,
            "run_label": run_label,
            "mission_id": prepared["mission_id"],
            "db_path": str(prepared["db_path"]),
            "provider_home": str(prepared["provider_home"]),
            "port": port,
            "pending_event_id": pending_event_id,
            "terminal_event": terminal,
            "server_returncode": proc.poll(),
            "event_types": event_types,
            "completed_at": utc_now(),
        }
        write_json(prepared["run_dir"] / "a0-smoke-result.json", result)
        print(json.dumps(result, ensure_ascii=False, indent=2))
    finally:
        stop_process(proc)


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--archive", type=Path, default=DEFAULT_ARCHIVE)
    parser.add_argument("--plasma", type=Path, default=None)
    sub = parser.add_subparsers(dest="command", required=True)

    pre = sub.add_parser("preflight", help="run static and process safety checks")
    pre.add_argument("--sample-id", default=DEFAULT_SAMPLE_ID)
    pre.set_defaults(func=preflight)

    prep = sub.add_parser("prepare-sample", help="create sample research DB and parity manifest from a seed DB")
    prep.add_argument("--sample-id", default=DEFAULT_SAMPLE_ID)
    prep.add_argument("--agent", default="codex")
    prep.add_argument("--agent-timeout", default="45m")
    prep.add_argument("--research-prompt", type=Path, default=DEFAULT_ARCHIVE / "prompts" / "pre-report-research.md")
    prep.add_argument("--auth-provider-home", type=Path, default=DEFAULT_ARCHIVE / "provider-homes" / DEFAULT_SAMPLE_ID / "research")
    prep.set_defaults(func=prepare_sample)

    smoke = sub.add_parser("a0-smoke", help="run A0 real smoke through safe serve wrapper")
    smoke.add_argument("--sample-id", default=DEFAULT_SAMPLE_ID)
    smoke.add_argument("--port", type=int, default=DEFAULT_A0_PORT)
    smoke.add_argument("--agent", default="codex")
    smoke.add_argument("--agent-timeout", default="150m")
    smoke.add_argument("--wait-seconds", type=int, default=45 * 60)
    smoke.add_argument("--poll-interval", type=float, default=5.0)
    smoke.set_defaults(func=a0_smoke)

    b = sub.add_parser("b-smoke", help="run B independent-section smoke through safe runner wrapper")
    b.add_argument("--sample-id", default=DEFAULT_SAMPLE_ID)
    b.add_argument("--a0-run-dir", type=Path, default=None)
    b.add_argument("--prompt-snapshot-dir", type=Path, default=None)
    b.add_argument("--agent-timeout", default=DEFAULT_B_TIMEOUT)
    b.set_defaults(func=b_smoke)

    c1 = sub.add_parser("b-reframe", help="rerun part/frame connective stages over an existing B section set")
    c1.add_argument("--sample-id", default=DEFAULT_SAMPLE_ID)
    c1.add_argument("--source-b-run-dir", type=Path, default=None)
    c1.add_argument("--prompt-snapshot-dir", type=Path, required=True)
    c1.add_argument("--run-label-prefix", default="C1-reframe")
    c1.add_argument("--agent-timeout", default=DEFAULT_B_TIMEOUT)
    c1.set_defaults(func=b_reframe)
    return parser


def main() -> int:
    parser = build_parser()
    args = parser.parse_args()
    try:
        args.func(args)
    except SafetyError as exc:
        print(f"safety: {exc}", file=sys.stderr)
        return 2
    except KeyboardInterrupt:
        print("interrupted", file=sys.stderr)
        return 130
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
