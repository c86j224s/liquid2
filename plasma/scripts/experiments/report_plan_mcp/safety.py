"""Fail-closed archive, environment, port, and namespace isolation."""

from __future__ import annotations

import os
import socket
from pathlib import Path
from typing import Mapping

from .models import ARCHIVE_SUFFIX, EXPERIMENT_ID, FORBIDDEN_PORTS


FIXED_ENV = ("HOME", "TMPDIR", "CODEX_HOME", "CLAUDE_CONFIG_DIR")


class IsolationError(ValueError):
    pass


def canonical_archive(home: Path) -> Path:
    return (home / ARCHIVE_SUFFIX).resolve()


def validate_archive(archive: Path, home: Path) -> Path:
    expected = canonical_archive(home)
    if archive.resolve() != expected:
        raise IsolationError("archive must be the fixed issue #110 experiment root")
    return expected


def isolated_environment(run_root: Path, inherited: Mapping[str, str]) -> dict[str, str | None]:
    env: dict[str, str | None] = {
        "HOME": str(run_root / "home"),
        "TMPDIR": str(run_root / "tmp"),
        "CODEX_HOME": str(run_root / "provider" / "codex"),
        "CLAUDE_CONFIG_DIR": str(run_root / "provider" / "claude"),
    }
    for key in inherited:
        if key.startswith("XDG_"):
            env[key] = str(run_root / "xdg" / key.removeprefix("XDG_").lower())
    return env


def validate_environment(env: Mapping[str, str | None], run_root: Path, inherited: Mapping[str, str]) -> None:
    required = set(FIXED_ENV) | {key for key in inherited if key.startswith("XDG_")}
    if not required.issubset(env):
        raise IsolationError("effective child environment omits an inherited isolation key")
    root = run_root.resolve()
    for key in required:
        value = env[key]
        if value is not None and not Path(value).resolve().is_relative_to(root):
            raise IsolationError(f"{key} escapes the run root")


def validate_run_paths(run_root: Path, database: Path, artifact: Path, workdir: Path, archive: Path) -> None:
    root = run_root.resolve()
    if not root.is_relative_to(archive.resolve()):
        raise IsolationError("run root is outside the experiment archive")
    for label, path in (("database", database), ("artifact", artifact), ("workdir", workdir)):
        if not path.resolve().is_relative_to(root):
            raise IsolationError(f"{label} escapes the run root")
    forbidden = ("dev-6002", "release-3002", "Application Support/Plasma", "plasma-ui-user.db")
    if any(marker in str(database) for marker in forbidden):
        raise IsolationError("development or release database is prohibited")


def validate_endpoint(host: str, port: int) -> None:
    if host not in {"127.0.0.1", "::1", "localhost"}:
        raise IsolationError("experiment server must use loopback")
    if port in FORBIDDEN_PORTS or not 6200 <= port <= 6299:
        raise IsolationError("port is outside the isolated allocation range")


def allocate_port(used: set[int]) -> int:
    for port in range(6200, 6300):
        if port in used or port in FORBIDDEN_PORTS:
            continue
        with socket.socket() as probe:
            try:
                probe.bind(("127.0.0.1", port))
            except OSError:
                continue
        used.add(port)
        return port
    raise IsolationError("no isolated experiment port is available")


def namespace(topic: str, replicate: int, arm: str, mode: str, nonce: str) -> str:
    value = f"{EXPERIMENT_ID}-{topic}-r{replicate}-{arm}-{mode}-{nonce}"
    if any(not (character.isalnum() or character in "-_") for character in value):
        raise IsolationError("namespace contains unsafe characters")
    return value


def ensure_unique_namespace(value: str, existing: set[str]) -> None:
    if value in existing:
        raise IsolationError("duplicate run namespace")
    existing.add(value)


def snapshot_protected_paths(paths: tuple[Path, ...], archive: Path) -> dict[str, tuple[bool, int]]:
    snapshot: dict[str, tuple[bool, int]] = {}
    for path in paths:
        resolved = path.expanduser().resolve()
        if resolved.is_relative_to(archive.resolve()):
            raise IsolationError("protected path must be outside the experiment archive")
        exists = resolved.exists()
        snapshot[str(resolved)] = (exists, resolved.stat().st_mtime_ns if exists else 0)
    return snapshot
