#!/usr/bin/env python3
"""Run the isolated product-path G2/H5 Plasma report experiment.

Raw experiment outputs stay under ~/research-artifacts and must not be committed.
The script intentionally drives the Plasma CLI instead of calling internals.
"""

from __future__ import annotations

import argparse
import concurrent.futures
import csv
import hashlib
import json
import math
import os
import random
import shutil
import sqlite3
import subprocess
import sys
import time
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path
from typing import Any


EXPERIMENT_ID = "11-product-path-g2-2026-07-07"
DEFAULT_ARCHIVE = Path.home() / "research-artifacts/liquid2/plasma/experiments" / EXPERIMENT_ID
DEFAULT_SOURCE_ARCHIVE = Path.home() / "research-artifacts/liquid2/plasma/experiments/10-generation-time-tone-2026-07-07/sources"
FORBIDDEN_DB_SUFFIXES = (
    "/tmp/plasma-ui-user.db",
    "/runtime/dev-6002/plasma-ui-user.db",
    "/runtime/release-3002/plasma-ui-user.db",
    "/Library/Application Support/Plasma/plasma.db",
)


@dataclass(frozen=True)
class Sample:
    sample_id: str
    slug: str
    title: str
    objective: str
    source_root: Path


@dataclass(frozen=True)
class Variant:
    name: str
    guidance: str
    humanize: bool
    final_kind: str


VARIANTS = (
    Variant("P0-current", "none", False, "raw"),
    Variant("P0-H5", "none", True, "humanized"),
    Variant("G2-current", "g2", False, "raw"),
    Variant("G2-H5", "g2", True, "humanized"),
)

PAIRINGS = (
    ("primary", "G2-H5", "P0-H5"),
    ("generation-guide", "G2-current", "P0-current"),
    ("h5-positive-control", "P0-H5", "P0-current"),
    ("h5-after-g2", "G2-H5", "G2-current"),
    ("bridge", "G2-current", "P0-H5"),
)

META_REPORT_MARKERS = (
    "본문을 읽지 못",
    "본문 텍스트가 확보되지",
    "원자료 본문으로 확인되지",
    "실질적인 결론은 내릴 수 없습니다",
    "후속 작성에 필요한 조치",
)


def repo_root() -> Path:
    return Path(__file__).resolve().parents[3]


def plasma_root() -> Path:
    return Path(__file__).resolve().parents[2]


def ensure_dirs(archive: Path) -> None:
    for name in (
        "bin",
        "source-roots",
        "manifests",
        "blocks",
        "runs",
        "artifacts",
        "analysis",
        "logs",
        "workdirs",
        "tmp-harness",
    ):
        (archive / name).mkdir(parents=True, exist_ok=True)


def reset_run_outputs(archive: Path) -> None:
    for name in ("blocks", "runs", "artifacts", "blind", "judging", "workdirs", "tmp-harness"):
        path = archive / name
        if path.exists():
            shutil.rmtree(path)
        path.mkdir(parents=True, exist_ok=True)
    for name in (
        "product-path-audit.jsonl",
        "run-summary.json",
        "blind-skipped.json",
        "blind-decode-private.csv",
        "blind-packet-summary.json",
        "preference-results.jsonl",
        "preference-summary.json",
    ):
        path = archive / "analysis" / name
        if path.exists():
            path.unlink()


def write_json(path: Path, value: Any) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")


def write_csv(path: Path, fieldnames: list[str], rows: list[dict[str, Any]]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", encoding="utf-8", newline="") as handle:
        writer = csv.DictWriter(handle, fieldnames=fieldnames, extrasaction="ignore")
        writer.writeheader()
        for row in rows:
            writer.writerow(row)


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


def safe_sample_id(index: int, source: Path) -> str:
    slug = source.stem.lower()
    allowed = []
    for ch in slug:
        if ch.isalnum() or ch in ("-", "_"):
            allowed.append(ch)
        else:
            allowed.append("-")
    clean = "-".join("".join(allowed).split("-"))
    return f"s{index:02d}-{clean or 'sample'}"


def sample_title(slug: str) -> str:
    title = slug.replace("-", " ").replace("_", " ").strip()
    return title[:1].upper() + title[1:] if title else "Product path report sample"


def prepare_sources(archive: Path, source_archive: Path) -> list[Sample]:
    if not source_archive.exists():
        raise SystemExit(f"source archive not found: {source_archive}")
    source_files = sorted(path for path in source_archive.glob("*.md") if path.is_file())
    if not source_files:
        raise SystemExit(f"no .md source files under {source_archive}")

    samples: list[Sample] = []
    manifest_rows: list[dict[str, Any]] = []
    for index, source in enumerate(source_files, start=1):
        sample_id = safe_sample_id(index, source)
        source_root = archive / "source-roots" / sample_id
        source_root.mkdir(parents=True, exist_ok=True)
        target = source_root / "source.md"
        shutil.copyfile(source, target)
        digest = sha256_file(target)
        (archive / "manifests" / f"{sample_id}.sources.sha256").write_text(
            f"{digest}  source.md\n", encoding="utf-8"
        )
        title = sample_title(source.stem)
        objective = (
            f"{title} 자료를 Plasma 등록 소스로만 읽고, 구체적인 사실, 맥락, "
            "불확실성, 비교 포인트를 보존한 한국어 보고서를 작성한다."
        )
        sample = Sample(sample_id, source.stem, title, objective, source_root)
        samples.append(sample)
        manifest_rows.append(
            {
                "sample_id": sample_id,
                "slug": source.stem,
                "title": title,
                "source_file_count": 1,
                "source_bytes": target.stat().st_size,
                "sha256": digest,
            }
        )
    write_json(archive / "manifests/sample-manifest-private.json", manifest_rows)
    return samples


def load_samples(archive: Path, source_archive: Path, limit: int | None) -> list[Sample]:
    samples = prepare_sources(archive, source_archive)
    if limit is not None:
        samples = samples[:limit]
    return samples


def build_plasma(archive: Path) -> Path:
    binary = archive / "bin" / "plasma"
    cmd = ["go", "build", "-o", str(binary), "./cmd/plasma"]
    result = subprocess.run(cmd, cwd=plasma_root(), text=True, capture_output=True)
    (archive / "logs/build.stdout.log").write_text(result.stdout, encoding="utf-8")
    (archive / "logs/build.stderr.log").write_text(result.stderr, encoding="utf-8")
    if result.returncode != 0:
        raise SystemExit(f"go build failed; see {archive / 'logs/build.stderr.log'}")
    return binary


def command_db_path(args: list[str]) -> Path | None:
    for index, arg in enumerate(args):
        if arg == "-db" and index + 1 < len(args):
            return Path(args[index + 1])
        if arg.startswith("-db="):
            return Path(arg.split("=", 1)[1])
    return None


def assert_archive_db(archive: Path, db_path: Path) -> None:
    resolved_archive = archive.resolve()
    resolved_db = db_path.resolve()
    if resolved_archive not in resolved_db.parents:
        raise RuntimeError(f"refusing non-archive DB path: {db_path}")
    db_text = str(resolved_db)
    for suffix in FORBIDDEN_DB_SUFFIXES:
        if db_text.endswith(suffix):
            raise RuntimeError(f"refusing forbidden DB path: {db_path}")


def run_cli(
    archive: Path,
    run_id: str,
    cmd: list[str],
    *,
    cwd: Path | None = None,
    timeout_seconds: int | None = None,
) -> dict[str, Any]:
    db_path = command_db_path(cmd)
    if db_path is not None:
        assert_archive_db(archive, db_path)
    env = os.environ.copy()
    env["PLASMA_RUNTIME_MODE"] = "release"
    env["TMPDIR"] = str(archive / "tmp-harness" / run_id)
    Path(env["TMPDIR"]).mkdir(parents=True, exist_ok=True)
    started = datetime.now(timezone.utc)
    completed: datetime | None = None
    proc = subprocess.run(
        cmd,
        cwd=cwd or plasma_root(),
        env=env,
        text=True,
        capture_output=True,
        timeout=timeout_seconds,
    )
    completed = datetime.now(timezone.utc)
    log_prefix = archive / "logs" / run_id
    log_prefix.parent.mkdir(parents=True, exist_ok=True)
    (log_prefix.with_suffix(".stdout.log")).write_text(proc.stdout, encoding="utf-8")
    (log_prefix.with_suffix(".stderr.log")).write_text(proc.stderr, encoding="utf-8")
    record = {
        "run_id": run_id,
        "cmd": cmd,
        "cwd": str(cwd or plasma_root()),
        "returncode": proc.returncode,
        "started_at": started.isoformat(),
        "completed_at": completed.isoformat(),
        "duration_seconds": (completed - started).total_seconds(),
        "stdout_log": str(log_prefix.with_suffix(".stdout.log")),
        "stderr_log": str(log_prefix.with_suffix(".stderr.log")),
    }
    append_jsonl(archive / "runs/commands.jsonl", record)
    if proc.returncode != 0:
        raise RuntimeError(f"command failed for {run_id}; see {log_prefix.with_suffix('.stderr.log')}")
    return record


def parse_json_file(path: Path) -> dict[str, Any]:
    return json.loads(path.read_text(encoding="utf-8"))


def deep_get(value: Any, *keys: str) -> Any:
    current = value
    for key in keys:
        if not isinstance(current, dict):
            return None
        current = current.get(key)
    return current


def first_deep_string(value: Any, key_names: tuple[str, ...]) -> str:
    if isinstance(value, dict):
        for key in key_names:
            candidate = value.get(key)
            if isinstance(candidate, str) and candidate:
                return candidate
        for child in value.values():
            found = first_deep_string(child, key_names)
            if found:
                return found
    if isinstance(value, list):
        for child in value:
            found = first_deep_string(child, key_names)
            if found:
                return found
    return ""


def artifact_id_from_report(report: dict[str, Any], variant: Variant) -> str:
    if variant.final_kind == "humanized":
        for path in (
            ("humanized", "Artifact", "ArtifactID"),
            ("humanized", "artifact", "artifact_id"),
            ("humanized", "artifact", "ArtifactID"),
        ):
            value = deep_get(report, *path)
            if isinstance(value, str) and value:
                return value
        return first_deep_string(report.get("humanized", {}), ("ArtifactID", "artifact_id"))
    for path in (("artifact", "ArtifactID"), ("artifact", "artifact_id")):
        value = deep_get(report, *path)
        if isinstance(value, str) and value:
            return value
    return first_deep_string(report.get("artifact", {}), ("ArtifactID", "artifact_id"))


def raw_artifact_id_from_report(report: dict[str, Any]) -> str:
    for path in (("artifact", "ArtifactID"), ("artifact", "artifact_id")):
        value = deep_get(report, *path)
        if isinstance(value, str) and value:
            return value
    return first_deep_string(report.get("artifact", {}), ("ArtifactID", "artifact_id"))


def mission_id_from_json(created: dict[str, Any]) -> str:
    for path in (("projection", "mission_id"), ("mission", "MissionID"), ("mission", "mission_id")):
        value = deep_get(created, *path)
        if isinstance(value, str) and value:
            return value
    return first_deep_string(created, ("mission_id", "MissionID"))


def extract_artifact(db_path: Path, artifact_id: str, output_path: Path) -> dict[str, Any]:
    with sqlite3.connect(db_path) as conn:
        row = conn.execute(
            "SELECT media_type, byte_size, sha256, filename, content_blob FROM plasma_raw_artifacts WHERE artifact_id = ?",
            (artifact_id,),
        ).fetchone()
    if not row:
        raise RuntimeError(f"artifact not found: {artifact_id} in {db_path}")
    media_type, byte_size, sha256, filename, content = row
    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_bytes(content)
    return {
        "artifact_id": artifact_id,
        "media_type": media_type,
        "byte_size": byte_size,
        "sha256": sha256,
        "filename": filename,
        "extracted_path": str(output_path),
    }


def load_events(db_path: Path) -> list[dict[str, Any]]:
    with sqlite3.connect(db_path) as conn:
        rows = conn.execute(
            "SELECT sequence, event_id, event_type, producer_type, producer_id, payload_json, created_at "
            "FROM plasma_ledger_events ORDER BY sequence"
        ).fetchall()
    events: list[dict[str, Any]] = []
    for sequence, event_id, event_type, producer_type, producer_id, payload_json, created_at in rows:
        try:
            payload = json.loads(payload_json)
        except json.JSONDecodeError:
            payload = {}
        events.append(
            {
                "sequence": sequence,
                "event_id": event_id,
                "event_type": event_type,
                "producer_type": producer_type,
                "producer_id": producer_id,
                "payload": payload,
                "created_at": created_at,
            }
        )
    return events


def source_read_trace_exists(events: list[dict[str, Any]]) -> bool:
    event_types = {event["event_type"] for event in events}
    if "source.observed" in event_types:
        return True
    for event in events:
        text = json.dumps(event.get("payload", {}), ensure_ascii=False)
        if "plasma.sources.read" in text or "plasma.research.read" in text or "observation_event_id" in text:
            return True
    return False


def candidate_quality_failures(path: Path) -> list[str]:
    failures: list[str] = []
    text = path.read_text(encoding="utf-8", errors="replace")
    if len(text.strip()) < 800:
        failures.append("candidate_too_short")
    marker_hits = sum(1 for marker in META_REPORT_MARKERS if marker in text)
    if marker_hits >= 2:
        failures.append("meta_source_unread_report")
    if any(label in text for label in ("P0-current", "P0-H5", "G2-current", "G2-H5", "h5-full-report-tone-pass")):
        failures.append("experiment_label_leakage")
    return failures


def assert_variant_events(events: list[dict[str, Any]], variant: Variant) -> list[str]:
    event_types = [event["event_type"] for event in events]
    failures: list[str] = []
    if "source.snapshotted" not in event_types and "source.local_path.attached" not in event_types:
        failures.append("missing_source_registration_event")
    if "report.artifact.created" not in event_types:
        failures.append("missing_report_artifact_created")
    if variant.humanize:
        if "report.humanize.pending" not in event_types or "report.artifact.exported" not in event_types:
            failures.append("missing_h5_events")
    else:
        if "report.humanize.pending" in event_types or "report.artifact.exported" in event_types:
            failures.append("unexpected_h5_events")
    payloads = [event["payload"] for event in events if event["event_type"] in {"report.draft.pending", "report.artifact.created"}]
    expected_profile = "" if variant.guidance == "none" else variant.guidance
    for payload in payloads:
        if payload.get("generation_guidance_profile", "") != expected_profile:
            failures.append("guidance_profile_mismatch")
            break
        if bool(payload.get("humanize_enabled")) != variant.humanize:
            failures.append("humanize_flag_mismatch")
            break
    return failures


def create_seed_db(archive: Path, plasma_bin: Path, sample: Sample, replicate: int) -> tuple[Path, str]:
    block_dir = archive / "blocks" / f"{sample.sample_id}-r{replicate:02d}"
    block_dir.mkdir(parents=True, exist_ok=True)
    seed_db = block_dir / "seed.db"
    if seed_db.exists():
        seed_db.unlink()
    mission_json = block_dir / "mission.json"
    attach_json = block_dir / "source-attach.json"
    run_cli(
        archive,
        f"{sample.sample_id}-r{replicate:02d}-mission",
        [
            str(plasma_bin),
            "missions",
            "create",
            "-db",
            str(seed_db),
            "-title",
            sample.title,
            "-objective",
            sample.objective,
            "-json",
        ],
    )
    shutil.copyfile(archive / "logs" / f"{sample.sample_id}-r{replicate:02d}-mission.stdout.log", mission_json)
    mission_id = mission_id_from_json(parse_json_file(mission_json))
    if not mission_id:
        raise RuntimeError(f"mission id not found in {mission_json}")
    root_spec = f"sample={sample.source_root}"
    run_cli(
        archive,
        f"{sample.sample_id}-r{replicate:02d}-attach",
        [
            str(plasma_bin),
            "sources",
            "attach-local",
            mission_id,
            "-db",
            str(seed_db),
            "-local-source-root",
            root_spec,
            "-root",
            "sample",
            "-path",
            ".",
            "-title",
            f"{sample.title} source packet",
            "-json",
        ],
    )
    shutil.copyfile(archive / "logs" / f"{sample.sample_id}-r{replicate:02d}-attach.stdout.log", attach_json)
    return seed_db, mission_id


def run_variant(
    archive: Path,
    plasma_bin: Path,
    sample: Sample,
    replicate: int,
    seed_db: Path,
    mission_id: str,
    variant: Variant,
    agent: str,
    agent_timeout: str,
) -> dict[str, Any]:
    run_label = f"{sample.sample_id}-r{replicate:02d}-{variant.name}"
    variant_dir = archive / "blocks" / f"{sample.sample_id}-r{replicate:02d}" / variant.name
    variant_dir.mkdir(parents=True, exist_ok=True)
    run_db = variant_dir / "run.db"
    if run_db.exists():
        run_db.unlink()
    shutil.copyfile(seed_db, run_db)
    workdir = archive / "workdirs" / run_label
    workdir.mkdir(parents=True, exist_ok=True)
    root_spec = f"sample={sample.source_root}"
    cmd = [
        str(plasma_bin),
        "reports",
        "draft",
        mission_id,
        "-db",
        str(run_db),
        "-local-source-root",
        root_spec,
        "-title",
        sample.title,
        "-mode",
        "planned",
        "-agent",
        agent,
        "-mcp-mode",
        "auto",
        "-agent-workdir",
        str(workdir),
        "-agent-timeout",
        agent_timeout,
        f"-humanize={str(variant.humanize).lower()}",
        "-experimental-generation-guidance",
        variant.guidance,
        "-report-session-policy",
        "auto",
        "-wait",
        "-json",
    ]
    started = time.monotonic()
    run_cli(archive, run_label, cmd)
    duration = time.monotonic() - started
    report_json = variant_dir / "report-draft.json"
    shutil.copyfile(archive / "logs" / f"{run_label}.stdout.log", report_json)
    report = parse_json_file(report_json)
    raw_artifact_id = raw_artifact_id_from_report(report)
    if not raw_artifact_id:
        raise RuntimeError(f"raw artifact id not found for {run_label}")
    artifact_id = artifact_id_from_report(report, variant)
    if not artifact_id:
        raise RuntimeError(f"artifact id not found for {run_label}")
    artifact_path = archive / "artifacts" / f"{sample.sample_id}-r{replicate:02d}" / f"{variant.name}.md"
    raw_artifact_path = archive / "artifacts" / f"{sample.sample_id}-r{replicate:02d}" / f"{variant.name}.raw.md"
    raw_artifact_info = extract_artifact(run_db, raw_artifact_id, raw_artifact_path)
    artifact_info = extract_artifact(run_db, artifact_id, artifact_path)
    events = load_events(run_db)
    event_audit = assert_variant_events(events, variant)
    quality_failures = candidate_quality_failures(artifact_path)
    trace_ok = source_read_trace_exists(events)
    row = {
        "sample_id": sample.sample_id,
        "replicate": replicate,
        "variant": variant.name,
        "guidance": variant.guidance,
        "humanize": variant.humanize,
        "final_kind": variant.final_kind,
        "mission_id": mission_id,
        "db_path": str(run_db),
        "workdir": str(workdir),
        "report_json": str(report_json),
        "raw_artifact": raw_artifact_info,
        "artifact": artifact_info,
        "duration_seconds": duration,
        "source_read_trace": trace_ok,
        "event_audit_failures": event_audit,
        "candidate_quality_failures": quality_failures,
        "event_types": [event["event_type"] for event in events],
    }
    append_jsonl(archive / "analysis/product-path-audit.jsonl", row)
    return row


def run_experiment(args: argparse.Namespace) -> None:
    archive = args.archive.expanduser().resolve()
    source_archive = args.source_archive.expanduser().resolve()
    ensure_dirs(archive)
    reset_run_outputs(archive)
    plasma_bin = build_plasma(archive)
    samples = load_samples(archive, source_archive, args.sample_limit)
    env_record = {
        "experiment_id": EXPERIMENT_ID,
        "archive": str(archive),
        "source_archive": str(source_archive),
        "repo_root": str(repo_root()),
        "plasma_root": str(plasma_root()),
        "plasma_binary": str(plasma_bin),
        "started_at": datetime.now(timezone.utc).isoformat(),
        "agent": args.agent,
        "agent_timeout": args.agent_timeout,
        "jobs": args.jobs,
        "replicates": args.replicates,
        "sample_ids": [sample.sample_id for sample in samples],
        "variants": [variant.__dict__ for variant in VARIANTS],
    }
    write_json(archive / "logs/run-environment.json", env_record)
    rows: list[dict[str, Any]] = []
    for replicate in range(1, args.replicates + 1):
        for sample in samples:
            seed_db, mission_id = create_seed_db(archive, plasma_bin, sample, replicate)
            with concurrent.futures.ThreadPoolExecutor(max_workers=args.jobs) as pool:
                futures = [
                    pool.submit(
                        run_variant,
                        archive,
                        plasma_bin,
                        sample,
                        replicate,
                        seed_db,
                        mission_id,
                        variant,
                        args.agent,
                        args.agent_timeout,
                    )
                    for variant in VARIANTS
                ]
                for future in concurrent.futures.as_completed(futures):
                    rows.append(future.result())
    write_json(archive / "analysis/run-summary.json", summarize_rows(rows))
    print(f"completed {len(rows)} variant runs")
    print(f"archive: {archive}")
    print(f"summary: {archive / 'analysis/run-summary.json'}")


def summarize_rows(rows: list[dict[str, Any]]) -> dict[str, Any]:
    by_variant: dict[str, dict[str, Any]] = {}
    for row in rows:
        entry = by_variant.setdefault(
            row["variant"],
            {
                "runs": 0,
                "source_read_trace": 0,
                "event_audit_failures": {},
                "candidate_quality_failures": {},
                "total_bytes": 0,
                "total_duration_seconds": 0.0,
            },
        )
        entry["runs"] += 1
        if row["source_read_trace"]:
            entry["source_read_trace"] += 1
        for failure in row["event_audit_failures"]:
            entry["event_audit_failures"][failure] = entry["event_audit_failures"].get(failure, 0) + 1
        for failure in row.get("candidate_quality_failures", []):
            entry["candidate_quality_failures"][failure] = entry["candidate_quality_failures"].get(failure, 0) + 1
        entry["total_bytes"] += int(row["artifact"]["byte_size"])
        entry["total_duration_seconds"] += float(row["duration_seconds"])
    for entry in by_variant.values():
        runs = max(1, entry["runs"])
        entry["mean_bytes"] = entry["total_bytes"] / runs
        entry["mean_duration_seconds"] = entry["total_duration_seconds"] / runs
    return {
        "experiment_id": EXPERIMENT_ID,
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "variant_runs": len(rows),
        "variants": by_variant,
        "hard_failures": [
            {
                "sample_id": row["sample_id"],
                "replicate": row["replicate"],
                "variant": row["variant"],
                "failures": row["event_audit_failures"],
                "source_read_trace": row["source_read_trace"],
            }
            for row in rows
            if row["event_audit_failures"] or not row["source_read_trace"]
            or row.get("candidate_quality_failures")
        ],
    }


def read_jsonl(path: Path) -> list[dict[str, Any]]:
    if not path.exists():
        return []
    rows: list[dict[str, Any]] = []
    with path.open("r", encoding="utf-8") as handle:
        for line in handle:
            line = line.strip()
            if line:
                rows.append(json.loads(line))
    return rows


def variant_index(rows: list[dict[str, Any]]) -> dict[tuple[str, int, str], dict[str, Any]]:
    indexed: dict[tuple[str, int, str], dict[str, Any]] = {}
    for row in rows:
        indexed[(row["sample_id"], int(row["replicate"]), row["variant"])] = row
    return indexed


def row_is_judgeable(row: dict[str, Any] | None) -> bool:
    if row is None:
        return False
    return (
        bool(row.get("source_read_trace"))
        and not row.get("event_audit_failures")
        and not row.get("candidate_quality_failures")
    )


def blind_side_order(case_id: str, left_variant: str, right_variant: str) -> tuple[str, str]:
    digest = hashlib.sha256(f"{case_id}:{left_variant}:{right_variant}".encode("utf-8")).digest()
    if digest[0] % 2 == 0:
        return left_variant, right_variant
    return right_variant, left_variant


def make_judge_schema(archive: Path) -> Path:
    schema = {
        "type": "object",
        "additionalProperties": False,
        "properties": {
            "overall_winner": {"type": "string", "enum": ["left", "right", "tie"]},
            "tone_winner": {"type": "string", "enum": ["left", "right", "tie"]},
            "coverage_winner": {"type": "string", "enum": ["left", "right", "tie"]},
            "source_safety_winner": {"type": "string", "enum": ["left", "right", "tie"]},
            "verbosity_risk": {"type": "string", "enum": ["left", "right", "both", "neither"]},
            "hard_fail_observed": {"type": "boolean"},
            "rationale": {"type": "string"},
        },
        "required": [
            "overall_winner",
            "tone_winner",
            "coverage_winner",
            "source_safety_winner",
            "verbosity_risk",
            "hard_fail_observed",
            "rationale",
        ],
    }
    path = archive / "judging/judge-output.schema.json"
    write_json(path, schema)
    return path


def judge_prompt(left_path: Path, right_path: Path, rubric_path: Path) -> str:
    return f"""두 개의 블라인드 한국어 보고서를 비교해 주세요.

중요한 제한:
- 후보 이름, 실험 이름, 내부 구현 방식을 추측하지 마세요.
- 더 짧은 글을 자동으로 선호하지 마세요.
- 출처 안전성과 구체적 내용 보존이 말투보다 우선입니다.
- 자연스러운 한국어는 중요하지만, 보고서의 사실, 구조, 비교 포인트, 불확실성을 줄이면 안 됩니다.
- 긴 원문 인용은 하지 마세요.

반드시 아래 세 파일을 직접 읽고 비교하세요. 파일을 읽지 않은 상태에서 추측으로 답하면 판정 실패입니다.

읽을 파일:
- 왼쪽 보고서: {left_path.name}
- 오른쪽 보고서: {right_path.name}
- 평가 기준: {rubric_path.name}

반드시 JSON 하나만 출력하세요."""


def write_blind_case(
    archive: Path,
    case_number: int,
    pair_name: str,
    sample_id: str,
    replicate: int,
    left_row: dict[str, Any],
    right_row: dict[str, Any],
) -> dict[str, Any]:
    case_id = f"case-{case_number:04d}"
    display_left_variant, display_right_variant = blind_side_order(
        f"{pair_name}:{sample_id}:r{replicate:02d}", left_row["variant"], right_row["variant"]
    )
    row_by_variant = {left_row["variant"]: left_row, right_row["variant"]: right_row}
    case_dir = archive / "blind/round-01" / case_id
    case_dir.mkdir(parents=True, exist_ok=True)
    left_path = case_dir / "left.md"
    right_path = case_dir / "right.md"
    rubric_path = case_dir / "rubric.md"
    left_path.write_text(Path(row_by_variant[display_left_variant]["artifact"]["extracted_path"]).read_text(encoding="utf-8"), encoding="utf-8")
    right_path.write_text(Path(row_by_variant[display_right_variant]["artifact"]["extracted_path"]).read_text(encoding="utf-8"), encoding="utf-8")
    rubric_path.write_text(
        "\n".join(
            [
                "# Blind Report Rubric",
                "",
                "Choose the report a Plasma user should receive.",
                "",
                "Priority order:",
                "1. Source and evidence safety: does not invent unsupported certainty, keeps uncertainty visible.",
                "2. Coverage preservation: keeps concrete facts, comparison points, caveats, and useful detail.",
                "3. Report usefulness: coherent structure, readable flow, practical synthesis.",
                "4. Korean tone: natural, not stiff, but still report-like.",
                "",
                "Hard fail if a report is empty, visibly truncated, mentions hidden experiment labels,",
                "or replaces the report with meta commentary about the task.",
            ]
        )
        + "\n",
        encoding="utf-8",
    )
    return {
        "case_id": case_id,
        "pair": pair_name,
        "sample_id": sample_id,
        "replicate": replicate,
        "left_variant": display_left_variant,
        "right_variant": display_right_variant,
        "left_path": str(left_path),
        "right_path": str(right_path),
        "rubric_path": str(rubric_path),
        "case_dir": str(case_dir),
    }


def prepare_blind_packets(archive: Path) -> list[dict[str, Any]]:
    rows = read_jsonl(archive / "analysis/product-path-audit.jsonl")
    if not rows:
        raise SystemExit(f"no candidate audit rows found: {archive / 'analysis/product-path-audit.jsonl'}")
    indexed = variant_index(rows)
    sample_replicates = sorted({(row["sample_id"], int(row["replicate"])) for row in rows})
    cases: list[dict[str, Any]] = []
    skipped: list[dict[str, Any]] = []
    case_number = 1
    for sample_id, replicate in sample_replicates:
        for pair_name, first_variant, second_variant in PAIRINGS:
            first = indexed.get((sample_id, replicate, first_variant))
            second = indexed.get((sample_id, replicate, second_variant))
            if not row_is_judgeable(first) or not row_is_judgeable(second):
                skipped.append(
                    {
                        "pair": pair_name,
                        "sample_id": sample_id,
                        "replicate": replicate,
                        "first_variant": first_variant,
                        "second_variant": second_variant,
                        "reason": "candidate_missing_or_failed_hard_gate",
                    }
                )
                continue
            cases.append(write_blind_case(archive, case_number, pair_name, sample_id, replicate, first, second))
            case_number += 1
    write_json(archive / "analysis/blind-skipped.json", skipped)
    write_csv(
        archive / "analysis/blind-decode-private.csv",
        [
            "case_id",
            "pair",
            "sample_id",
            "replicate",
            "left_variant",
            "right_variant",
            "left_path",
            "right_path",
            "rubric_path",
        ],
        cases,
    )
    write_json(
        archive / "analysis/blind-packet-summary.json",
        {
            "generated_at": datetime.now(timezone.utc).isoformat(),
            "case_count": len(cases),
            "skipped_count": len(skipped),
            "pairs": list(PAIRINGS),
        },
    )
    return cases


def judge_case(
    archive: Path,
    schema_path: Path,
    case: dict[str, Any],
    pass_index: int,
    judge_model: str,
) -> dict[str, Any]:
    case_dir = Path(case["case_dir"])
    out_dir = archive / "judging" / case["case_id"]
    out_dir.mkdir(parents=True, exist_ok=True)
    output_path = out_dir / f"pass-{pass_index:02d}.json"
    stdout_path = out_dir / f"pass-{pass_index:02d}.stdout.log"
    stderr_path = out_dir / f"pass-{pass_index:02d}.stderr.log"
    if output_path.exists():
        try:
            payload = json.loads(output_path.read_text(encoding="utf-8"))
            return decode_judge_result(case, pass_index, payload, "cached")
        except json.JSONDecodeError:
            output_path.unlink()

    cmd = [
        "codex",
        "--ask-for-approval",
        "never",
        "--sandbox",
        "read-only",
        "exec",
        "--ephemeral",
        "--ignore-rules",
        "--skip-git-repo-check",
        "-C",
        str(case_dir),
        "--output-schema",
        str(schema_path),
        "-o",
        str(output_path),
    ]
    if judge_model:
        cmd.extend(["-m", judge_model])
    cmd.append(judge_prompt(case_dir / "left.md", case_dir / "right.md", case_dir / "rubric.md"))
    started = datetime.now(timezone.utc)
    proc = subprocess.run(cmd, text=True, capture_output=True)
    completed = datetime.now(timezone.utc)
    stdout_path.write_text(proc.stdout, encoding="utf-8")
    stderr_path.write_text(proc.stderr, encoding="utf-8")
    command_record = {
        "case_id": case["case_id"],
        "pass": pass_index,
        "cmd": cmd,
        "returncode": proc.returncode,
        "started_at": started.isoformat(),
        "completed_at": completed.isoformat(),
        "duration_seconds": (completed - started).total_seconds(),
        "stdout_log": str(stdout_path),
        "stderr_log": str(stderr_path),
        "output_path": str(output_path),
    }
    append_jsonl(archive / "judging/judge-commands.jsonl", command_record)
    if proc.returncode != 0:
        return {
            **case,
            "pass": pass_index,
            "status": "failed",
            "error": f"judge command failed with returncode {proc.returncode}",
        }
    try:
        payload = json.loads(output_path.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        return {**case, "pass": pass_index, "status": "failed", "error": f"invalid judge json: {exc}"}
    return decode_judge_result(case, pass_index, payload, "ok")


def decode_judge_result(case: dict[str, Any], pass_index: int, payload: dict[str, Any], status: str) -> dict[str, Any]:
    winner_side = payload.get("overall_winner", "tie")
    if winner_side == "left":
        winner_variant = case["left_variant"]
    elif winner_side == "right":
        winner_variant = case["right_variant"]
    else:
        winner_variant = "tie"
    return {
        **case,
        "pass": pass_index,
        "status": status,
        "overall_winner_side": winner_side,
        "overall_winner_variant": winner_variant,
        "tone_winner": payload.get("tone_winner", "tie"),
        "coverage_winner": payload.get("coverage_winner", "tie"),
        "source_safety_winner": payload.get("source_safety_winner", "tie"),
        "verbosity_risk": payload.get("verbosity_risk", "neither"),
        "hard_fail_observed": bool(payload.get("hard_fail_observed")),
        "rationale": str(payload.get("rationale", ""))[:1000],
    }


def exact_one_sided_sign_test(successes: int, failures: int) -> float | None:
    n = successes + failures
    if n == 0:
        return None
    probability = 0.0
    for k in range(successes, n + 1):
        probability += math.comb(n, k) * (0.5**n)
    return probability


def summarize_judgments(results: list[dict[str, Any]]) -> dict[str, Any]:
    by_case: dict[str, list[dict[str, Any]]] = {}
    for result in results:
        if result.get("status") in {"ok", "cached"}:
            by_case.setdefault(result["case_id"], []).append(result)

    pair_samples: dict[str, dict[str, Any]] = {}
    decision_counts: dict[str, dict[str, int]] = {}
    for case_id, case_results in by_case.items():
        first = case_results[0]
        pair = first["pair"]
        left_variant = first["left_variant"]
        right_variant = first["right_variant"]
        counts = {left_variant: 0, right_variant: 0, "tie": 0}
        for result in case_results:
            winner = result["overall_winner_variant"]
            counts[winner if winner in counts else "tie"] += 1
        case_winner = "tie"
        majority_threshold = len(case_results) // 2 + 1
        for variant in (left_variant, right_variant):
            if counts[variant] >= majority_threshold:
                case_winner = variant
        entry = pair_samples.setdefault(pair, {"cases": [], "sample_wins": {}, "ties": 0})
        entry["cases"].append(
            {
                "case_id": case_id,
                "sample_id": first["sample_id"],
                "replicate": first["replicate"],
                "left_variant": left_variant,
                "right_variant": right_variant,
                "vote_counts": counts,
                "case_winner": case_winner,
            }
        )
        if case_winner == "tie":
            entry["ties"] += 1
        else:
            entry["sample_wins"][case_winner] = entry["sample_wins"].get(case_winner, 0) + 1
        decisions = decision_counts.setdefault(pair, {})
        for variant, count in counts.items():
            decisions[variant] = decisions.get(variant, 0) + count

    pair_summaries: dict[str, Any] = {}
    for pair_name, first_variant, second_variant in PAIRINGS:
        entry = pair_samples.get(pair_name, {"cases": [], "sample_wins": {}, "ties": 0})
        first_wins = int(entry["sample_wins"].get(first_variant, 0))
        second_wins = int(entry["sample_wins"].get(second_variant, 0))
        pair_summaries[pair_name] = {
            "first_variant": first_variant,
            "second_variant": second_variant,
            "case_count": len(entry["cases"]),
            "first_sample_wins": first_wins,
            "second_sample_wins": second_wins,
            "sample_ties": entry["ties"],
            "one_sided_p_first_beats_second": exact_one_sided_sign_test(first_wins, second_wins),
            "decision_counts": decision_counts.get(pair_name, {}),
            "cases": entry["cases"],
        }
    return {
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "judge_results": len(results),
        "failed_judgments": [result for result in results if result.get("status") == "failed"],
        "pairs": pair_summaries,
    }


def run_judging(args: argparse.Namespace) -> None:
    archive = args.archive.expanduser().resolve()
    cases = prepare_blind_packets(archive)
    if not cases:
        raise SystemExit("no judgeable blind cases were created")
    preference_path = archive / "analysis/preference-results.jsonl"
    if preference_path.exists():
        preference_path.unlink()
    schema_path = make_judge_schema(archive)
    results: list[dict[str, Any]] = []
    jobs = max(1, args.judge_jobs)
    passes = max(1, args.judge_passes)
    with concurrent.futures.ThreadPoolExecutor(max_workers=jobs) as pool:
        futures = [
            pool.submit(judge_case, archive, schema_path, case, pass_index, args.judge_model)
            for case in cases
            for pass_index in range(1, passes + 1)
        ]
        for future in concurrent.futures.as_completed(futures):
            result = future.result()
            results.append(result)
            append_jsonl(preference_path, result)
    summary = summarize_judgments(results)
    write_json(archive / "analysis/preference-summary.json", summary)
    print(f"judged {len(results)} passes over {len(cases)} blind cases")
    print(f"summary: {archive / 'analysis/preference-summary.json'}")


def main(argv: list[str]) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("command", choices=("run", "smoke", "judge"))
    parser.add_argument("--archive", type=Path, default=DEFAULT_ARCHIVE)
    parser.add_argument("--source-archive", type=Path, default=DEFAULT_SOURCE_ARCHIVE)
    parser.add_argument("--sample-limit", type=int, default=None)
    parser.add_argument("--replicates", type=int, default=1)
    parser.add_argument("--jobs", type=int, default=2)
    parser.add_argument("--agent", default="codex")
    parser.add_argument("--agent-timeout", default="75m")
    parser.add_argument("--judge-passes", type=int, default=5)
    parser.add_argument("--judge-jobs", type=int, default=2)
    parser.add_argument("--judge-model", default="")
    args = parser.parse_args(argv)
    if args.command == "judge":
        run_judging(args)
        return 0
    if args.command == "smoke":
        args.sample_limit = 1 if args.sample_limit is None else args.sample_limit
        args.replicates = 1
        args.jobs = min(args.jobs, 2)
    if args.jobs < 1:
        raise SystemExit("--jobs must be >= 1")
    if args.replicates < 1:
        raise SystemExit("--replicates must be >= 1")
    run_experiment(args)
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
