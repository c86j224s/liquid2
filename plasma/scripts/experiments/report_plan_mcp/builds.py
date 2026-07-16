"""Reproducible source and binary build manifests."""

from __future__ import annotations

import hashlib
import json
from pathlib import Path
import shutil
import subprocess
import tarfile


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def source_commands(repo: Path, archive: Path, commit: str, arm: str) -> tuple[tuple[str, ...], ...]:
    source = archive / "local-sources" / arm
    tarball = archive / "source-manifests" / f"{arm}-{commit}.tar"
    return (
        ("git", "-C", str(repo), "archive", "--format=tar", "-o", str(tarball), commit),
        ("tar", "-xf", str(tarball), "-C", str(source)),
    )


def build_command(source: Path, binary: Path, commit: str) -> tuple[str, ...]:
    return ("go", "build", "-trimpath", "-ldflags", f"-X main.commit={commit}", "-o", str(binary), "./cmd/plasma")


def version_command(binary: Path) -> tuple[str, ...]:
    return (str(binary), "version")


def export_and_build(repo: Path, archive: Path, commit: str, arm: str) -> dict[str, str]:
    source = archive / "local-sources" / arm
    binary = archive / "bin" / arm / "plasma"
    tarball = archive / "source-manifests" / f"{arm}-{commit}.tar"
    if source.exists():
        shutil.rmtree(source)
    source.mkdir(parents=True)
    binary.parent.mkdir(parents=True, exist_ok=True)
    tarball.parent.mkdir(parents=True, exist_ok=True)
    subprocess.run(source_commands(repo, archive, commit, arm)[0], check=True)
    with tarfile.open(tarball) as bundle:
        bundle.extractall(source, filter="data")
    subprocess.run(build_command(source / "plasma", binary, commit), cwd=source / "plasma", check=True)
    version = subprocess.check_output(version_command(binary), text=True).strip()
    result = {
        "arm": arm,
        "commit": commit,
        "source_archive": str(tarball),
        "source_sha256": sha256_file(tarball),
        "binary": str(binary),
        "binary_sha256": sha256_file(binary),
        "version": version,
    }
    manifest = archive / "source-manifests" / f"{arm}-build.json"
    with manifest.open("x", encoding="utf-8") as handle:
        json.dump(result, handle, indent=2, sort_keys=True)
        handle.write("\n")
    return result
