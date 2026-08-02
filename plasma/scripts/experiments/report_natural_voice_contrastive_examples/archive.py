from __future__ import annotations

from pathlib import Path
import shutil
from typing import Mapping

from report_natural_voice_correction.archive import read_json, write_json_atomic

from . import config


ARCHIVE_DIRS = (
    "analysis",
    "blind",
    "control/prompts",
    "inputs/development",
    "inputs/evaluation",
    "runs",
    "tmp-harness",
)


class ArchiveError(ValueError):
    pass


class ExperimentArchive:
    def __init__(
        self,
        root: Path,
        *,
        home: Path | None = None,
        development_sources: Mapping[str, Path] | None = None,
        evaluation_sources: Mapping[str, Path] | None = None,
        enforce_fixed_root: bool = True,
    ) -> None:
        self.home = (home or Path.home()).resolve()
        self.root = Path(root).expanduser().resolve()
        if enforce_fixed_root and self.root != config.fixed_archive_root(self.home):
            raise ArchiveError("archive must resolve exactly to the fixed experiment 59 archive")
        self.development_sources = self._resolve_sources(development_sources or config.DEVELOPMENT_SOURCES)
        self.evaluation_sources = self._resolve_sources(evaluation_sources or config.EVALUATION_SOURCES)

    @classmethod
    def from_path(cls, path: Path | None = None) -> "ExperimentArchive":
        expected = config.fixed_archive_root()
        candidate = expected if path is None else Path(path).expanduser().resolve()
        return cls(candidate)

    def _resolve_sources(self, sources: Mapping[str, Path]) -> dict[str, Path]:
        resolved: dict[str, Path] = {}
        for filename, path in sources.items():
            source = Path(path)
            resolved[filename] = source.resolve() if source.is_absolute() else (self.home / source).resolve()
        return resolved

    def ensure_layout(self) -> None:
        for name in ARCHIVE_DIRS:
            (self.root / name).mkdir(parents=True, exist_ok=True)

    def prepare(self) -> dict[str, object]:
        manifest_path = self.root / "control" / "source-manifest.lock.json"
        if manifest_path.exists():
            return self.verify_source_seal()
        if self.root.exists() and any(self.root.iterdir()):
            raise ArchiveError("refusing to prepare over a non-empty unsealed archive")
        self.ensure_layout()
        sets: dict[str, list[dict[str, str]]] = {}
        for set_name, sources in self._source_sets().items():
            rows: list[dict[str, str]] = []
            for filename, source in sources.items():
                if not source.is_file() or source.is_symlink():
                    raise ArchiveError(f"source must be a regular file: {filename}")
                destination = self.root / "inputs" / set_name / filename
                shutil.copyfile(source, destination)
                source_sha = config.sha256_file(source)
                destination_sha = config.sha256_file(destination)
                if source_sha != destination_sha:
                    raise ArchiveError(f"copied source hash mismatch: {filename}")
                rows.append({
                    "filename": filename,
                    "source_path": config.display_home_path(source, self.home),
                    "source_sha256": source_sha,
                    "destination_path": self.rel(destination),
                    "destination_sha256": destination_sha,
                })
            sets[set_name] = rows
        write_json_atomic(manifest_path, {
            "experiment_id": config.EXPERIMENT_ID,
            "invalid_material_used": False,
            "sets": sets,
        })
        return self.verify_source_seal()

    def verify_source_seal(self) -> dict[str, object]:
        manifest_path = self.root / "control" / "source-manifest.lock.json"
        if not manifest_path.is_file():
            raise ArchiveError("missing source-manifest.lock.json")
        manifest = read_json(manifest_path)
        if set(manifest) != {"experiment_id", "invalid_material_used", "sets"}:
            raise ArchiveError("source manifest schema mismatch")
        if manifest.get("experiment_id") != config.EXPERIMENT_ID or manifest.get("invalid_material_used") is not False:
            raise ArchiveError("source manifest identity or validity mismatch")
        sets = manifest.get("sets")
        if not isinstance(sets, dict) or set(sets) != {"development", "evaluation"}:
            raise ArchiveError("source manifest set mismatch")
        verified: dict[str, list[dict[str, str]]] = {}
        for set_name, sources in self._source_sets().items():
            rows = sets.get(set_name)
            if not isinstance(rows, list) or len(rows) != len(sources):
                raise ArchiveError(f"source manifest {set_name} rows mismatch")
            by_name = {str(row.get("filename")): row for row in rows if isinstance(row, dict)}
            if set(by_name) != set(sources):
                raise ArchiveError(f"source manifest {set_name} filenames mismatch")
            input_dir = self.root / "inputs" / set_name
            children = sorted(input_dir.iterdir(), key=lambda path: path.name)
            if any(path.is_symlink() or not path.is_file() for path in children):
                raise ArchiveError(f"{set_name} inputs must contain regular files only")
            if {path.name for path in children} != set(sources):
                raise ArchiveError(f"{set_name} input files mismatch")
            verified_rows: list[dict[str, str]] = []
            for filename, source in sources.items():
                row = by_name[filename]
                destination = input_dir / filename
                source_sha = config.sha256_file(source)
                destination_sha = config.sha256_file(destination)
                expected_path = config.display_home_path(source, self.home)
                if row.get("source_path") != expected_path or row.get("destination_path") != self.rel(destination):
                    raise ArchiveError(f"source manifest path mismatch: {filename}")
                if row.get("source_sha256") != source_sha or row.get("destination_sha256") != destination_sha:
                    raise ArchiveError(f"source manifest hash mismatch: {filename}")
                if source_sha != destination_sha:
                    raise ArchiveError(f"source and destination differ: {filename}")
                verified_rows.append({"filename": filename, "sha256": destination_sha})
            verified[set_name] = verified_rows
        return {"passed": True, "experiment_id": config.EXPERIMENT_ID, "sets": verified}

    def input_path(self, set_name: str, file_id: str) -> Path:
        sources = self._source_sets().get(set_name)
        if sources is None:
            raise ArchiveError(f"unknown input set: {set_name}")
        for filename in sources:
            if config.file_id(filename) == file_id:
                return self.root / "inputs" / set_name / filename
        raise ArchiveError(f"unknown {set_name} file id: {file_id}")

    def development_file_ids(self) -> tuple[str, ...]:
        return tuple(config.file_id(name) for name in self.development_sources)

    def evaluation_file_ids(self) -> tuple[str, ...]:
        return tuple(config.file_id(name) for name in self.evaluation_sources)

    def rel(self, path: Path) -> str:
        return str(path.resolve().relative_to(self.root))

    def _source_sets(self) -> dict[str, dict[str, Path]]:
        return {"development": self.development_sources, "evaluation": self.evaluation_sources}
