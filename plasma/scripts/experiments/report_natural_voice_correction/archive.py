from __future__ import annotations

import json
import os
from pathlib import Path
from typing import Mapping

from . import config


ARCHIVE_DIRS = ("analysis", "blind", "control", "inputs", "logs", "runs", "tmp-harness")


class ArchiveError(ValueError):
    pass


class ExperimentArchive:
    def __init__(self, root: Path, expected_files: Mapping[str, str] | None = None, experiment: str = "56") -> None:
        self.experiment = config.resolve_experiment(experiment)
        self.experiment_id = config.experiment_id(self.experiment)
        self.archive_suffix = config.archive_suffix(self.experiment)
        self.root = Path(root).expanduser().resolve()
        self.fixed_corpus = expected_files is None
        if self.fixed_corpus and self.root != config.fixed_archive_root(experiment=self.experiment):
            raise ArchiveError(f"archive must resolve exactly to the fixed experiment {self.experiment} archive")
        self.expected_files = dict(expected_files or config.EXPECTED_SHA256_BY_FILENAME)

    @classmethod
    def from_path(cls, path: str | Path | None = None, experiment: str = "56") -> "ExperimentArchive":
        return cls(config.resolve_archive(path, experiment=experiment), experiment=experiment)

    def ensure_layout(self) -> None:
        for name in ARCHIVE_DIRS:
            (self.root / name).mkdir(parents=True, exist_ok=True)

    def rel(self, path: Path) -> str:
        return str(Path(path).resolve().relative_to(self.root))

    def expected_filenames(self) -> tuple[str, ...]:
        return tuple(self.expected_files)

    def expected_file_ids(self) -> tuple[str, ...]:
        return tuple(config.file_id_for_filename(name) for name in self.expected_filenames())

    def filename_for_file_id(self, file_id: str) -> str:
        for filename in self.expected_filenames():
            if config.file_id_for_filename(filename) == file_id:
                return filename
        raise ArchiveError(f"unlisted corpus file id: {file_id}")

    def input_path_for_file_id(self, file_id: str) -> Path:
        return self.root / "inputs" / self.filename_for_file_id(file_id)

    def verify_source_seal(self) -> dict[str, object]:
        manifest_path = self.root / "control" / "source-manifest.lock.json"
        if not manifest_path.is_file():
            raise ArchiveError("missing source-manifest.lock.json")
        manifest = read_json(manifest_path)
        if manifest.get("experiment_id") != self.experiment_id:
            raise ArchiveError("source manifest experiment_id mismatch")
        if self.fixed_corpus:
            expected_source = "~/" + config.SOURCE_SUFFIX.as_posix()
            expected_destination = "~/" + self.archive_suffix.as_posix()
            if manifest.get("source_directory") != expected_source:
                raise ArchiveError("source manifest source_directory mismatch")
            if manifest.get("destination_directory") != expected_destination:
                raise ArchiveError("source manifest destination_directory mismatch")
        if manifest.get("invalid_material_used") is not False:
            raise ArchiveError("source manifest indicates invalid material use")
        if "/invalid/" in json.dumps(manifest, ensure_ascii=False):
            raise ArchiveError("source manifest contains an invalid material path")
        rows = manifest.get("files")
        if not isinstance(rows, list):
            raise ArchiveError("source manifest files must be a list")
        row_names = [row.get("filename") for row in rows if isinstance(row, dict)]
        if len(row_names) != len(set(row_names)):
            raise ArchiveError("source manifest contains duplicate filenames")
        if set(row_names) != set(self.expected_files):
            raise ArchiveError("source manifest filename set mismatch")

        inputs = self.root / "inputs"
        if not inputs.is_dir():
            raise ArchiveError("missing inputs directory")
        children = sorted(inputs.iterdir(), key=lambda path: path.name)
        if any(child.is_symlink() for child in children):
            raise ArchiveError("inputs must not contain symlinks")
        actual_files = [child.name for child in children if child.is_file()]
        if len(actual_files) != len(children):
            raise ArchiveError("inputs contains unexpected non-file entries")
        if set(actual_files) != set(self.expected_files):
            raise ArchiveError("inputs filename set mismatch")

        locked: list[dict[str, str]] = []
        for row in rows:
            if not isinstance(row, dict):
                raise ArchiveError("source manifest file entry must be an object")
            filename = str(row["filename"])
            expected_sha = self.expected_files[filename]
            source_sha = str(row.get("source_sha256"))
            dest_sha = str(row.get("destination_sha256"))
            if source_sha != expected_sha or dest_sha != expected_sha:
                raise ArchiveError(f"locked SHA mismatch for {filename}")
            actual_sha = config.sha256_file(inputs / filename)
            if actual_sha != expected_sha:
                raise ArchiveError(f"input SHA mismatch for {filename}")
            locked.append({"filename": filename, "sha256": actual_sha})
        return {
            "passed": True,
            "experiment_id": self.experiment_id,
            "archive": str(self.root),
            "files": sorted(locked, key=lambda row: row["filename"]),
        }


def read_json(path: Path) -> dict[str, object]:
    value = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(value, dict):
        raise ArchiveError(f"required JSON object is invalid: {path}")
    return value


def write_json_atomic(path: Path, value: object) -> None:
    text = json.dumps(value, ensure_ascii=False, indent=2, sort_keys=True) + "\n"
    write_text_atomic(path, text)


def write_text_atomic(path: Path, text: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    tmp = path.with_name(f".{path.name}.tmp")
    tmp.write_text(text, encoding="utf-8")
    os.replace(tmp, path)
