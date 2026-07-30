#!/usr/bin/env python3
"""Fixed-input contracts for the issue #190 final-writer v2 experiment."""

from __future__ import annotations

import argparse
from contextlib import closing
from dataclasses import asdict, dataclass, replace
import hashlib
import json
from pathlib import Path, PurePosixPath
import random
import shutil
import sqlite3
import sys
from typing import Mapping, Sequence


EXPERIMENT_ID = "55-final-writer-v2-2026-07-29"
EVIDENCE_VERSION = "w6-b-product-reviewed-parts"
ARCHIVE_SUFFIX = Path("research-artifacts/liquid2/plasma/experiments") / EXPERIMENT_ID
PUBLIC_DOC_DIR = f"plasma/docs/experiments/{EXPERIMENT_ID}"

ARM_A = "A"
ARM_B = "B"
PIPELINE_V1 = "reader_style_gate_v1"
PIPELINE_V2 = "assembly_writer_reader_style_gate_v2"

FINAL_ASSEMBLY_CREATED = "report.final_assembly.created"
WRITER_STARTED = "report.final_edit.writer.started"
WRITER_SUBMITTED = "report.final_edit.writer.submitted"
READER_STARTED = "report.final_edit.reader.started"
READER_SUBMITTED = "report.final_edit.reader.submitted"
STYLE_STARTED = "report.final_edit.style.started"
STYLE_SUBMITTED = "report.final_edit.style.submitted"
GATE_STARTED = "report.final_edit.gate.started"
GATE_SUBMITTED = "report.final_edit.gate.submitted"

BLIND_LABELS = ("report_1", "report_2")
VALID_WINNERS = (ARM_A, ARM_B, "tie")
HARD_FAIL_COUNT_KEYS = (
    "information_loss_count",
    "citation_loss_count",
    "requirement_loss_count",
    "unsupported_external_fact_count",
)
HARD_FAIL_STATUS_KEYS = (
    "product_prompt_stage_parity",
    "pair_identity",
    "blinding",
    "archive_adoption",
    "stage_payload_contract",
)
BLINDING_LEAK_TERMS = (
    PIPELINE_V1,
    PIPELINE_V2,
    "final_write",
    "final writer",
    "current v1",
    "writer v2",
    "arm a",
    "arm b",
)
INVALID_UNSEALED_READING_DIR = Path("invalid/w6-b-unsealed-reading-2026-07-30")
BLIND_SEAL_PATH = f"control/blind_seal.{EVIDENCE_VERSION}.json"
PRE_MUTATION_RECEIPT = "control/w6-c-pre-mutation-sha256.json"
POST_MUTATION_RECEIPT = "control/w6-c-post-mutation-sha256.json"


@dataclass(frozen=True)
class StageContract:
    kind: str
    label: str
    tool_prefix: str | None
    tools: tuple[str, ...]
    required_events: tuple[str, ...]
    fork_from: str
    source_artifact: str
    canonicalizes: bool = False


@dataclass(frozen=True)
class ArmContract:
    arm: str
    name: str
    pipeline: str
    stages: tuple[StageContract, ...]


@dataclass(frozen=True)
class PairSpec:
    pair_id: str
    topic_id: str
    topic_title: str
    rigor: str
    fixed_part_manifest: str
    frozen_part_manifest_sha256_receipt: str


def default_archive_root(home: Path | None = None) -> Path:
    return (home or Path.home()) / ARCHIVE_SUFFIX


def repository_root(start: Path | None = None) -> Path:
    current = (start or Path(__file__)).resolve()
    for path in (current, *current.parents):
        if (path / ".git").exists():
            return path
    return Path.cwd().resolve()


def _is_relative_to(path: Path, parent: Path) -> bool:
    try:
        path.relative_to(parent)
        return True
    except ValueError:
        return False


def ensure_archive_outside_repo(archive_root: Path, repo_root: Path | None = None) -> Path:
    archive = archive_root.expanduser().resolve()
    repo = (repo_root or repository_root()).resolve()
    if archive == repo or _is_relative_to(archive, repo):
        raise ValueError("experiment archive must be outside the repository")
    return archive


def require_archive_contained(archive: Path, path: Path) -> Path:
    archive = archive.expanduser().resolve()
    resolved = path.expanduser().resolve()
    try:
        resolved.relative_to(archive)
    except ValueError as exc:
        raise ValueError(f"archive path escapes experiment archive: {path}") from exc
    if resolved == archive:
        raise ValueError(f"archive path points to archive root: {path}")
    return resolved


def tool_surface(prefix: str) -> tuple[str, str, str, str]:
    return (f"{prefix}.start", f"{prefix}.read", f"{prefix}.patch", f"{prefix}.submit")


def arm_contracts() -> dict[str, ArmContract]:
    return {
        ARM_A: ArmContract(
            arm=ARM_A,
            name="current_v1",
            pipeline=PIPELINE_V1,
            stages=(
                StageContract(
                    kind="reader_source_assembly",
                    label="reader source assembly",
                    tool_prefix=None,
                    tools=(),
                    required_events=(),
                    fork_from="none",
                    source_artifact="reviewed_part_artifacts",
                ),
                StageContract(
                    kind="reader_edit",
                    label="reader editor",
                    tool_prefix="plasma.report.long_form.reader_edit",
                    tools=tool_surface("plasma.report.long_form.reader_edit"),
                    required_events=(READER_STARTED, READER_SUBMITTED),
                    fork_from="report_plan_session",
                    source_artifact="reader_source_assembly",
                ),
                StageContract(
                    kind="style_edit",
                    label="optional style editor",
                    tool_prefix="plasma.report.long_form.style_edit",
                    tools=tool_surface("plasma.report.long_form.style_edit"),
                    required_events=(STYLE_STARTED, STYLE_SUBMITTED),
                    fork_from="reader_provider_session",
                    source_artifact="reader_edit",
                ),
                StageContract(
                    kind="corrective_gate",
                    label="corrective gate",
                    tool_prefix="plasma.report.long_form.final_edit",
                    tools=tool_surface("plasma.report.long_form.final_edit"),
                    required_events=(GATE_STARTED, GATE_SUBMITTED),
                    fork_from="report_plan_session",
                    source_artifact="style_edit",
                    canonicalizes=True,
                ),
            ),
        ),
        ARM_B: ArmContract(
            arm=ARM_B,
            name="final_writer_v2",
            pipeline=PIPELINE_V2,
            stages=(
                StageContract(
                    kind="final_assembly",
                    label="최종 조립",
                    tool_prefix=None,
                    tools=(),
                    required_events=(FINAL_ASSEMBLY_CREATED,),
                    fork_from="none",
                    source_artifact="reviewed_part_artifacts",
                ),
                StageContract(
                    kind="final_write",
                    label="최종 작성",
                    tool_prefix="plasma.report.long_form.final_write",
                    tools=tool_surface("plasma.report.long_form.final_write"),
                    required_events=(WRITER_STARTED, WRITER_SUBMITTED),
                    fork_from="report_plan_session",
                    source_artifact="final_assembly",
                ),
                StageContract(
                    kind="reader_edit",
                    label="reader editor",
                    tool_prefix="plasma.report.long_form.reader_edit",
                    tools=tool_surface("plasma.report.long_form.reader_edit"),
                    required_events=(READER_STARTED, READER_SUBMITTED),
                    fork_from="report_plan_session",
                    source_artifact="final_write",
                ),
                StageContract(
                    kind="style_edit",
                    label="optional style editor",
                    tool_prefix="plasma.report.long_form.style_edit",
                    tools=tool_surface("plasma.report.long_form.style_edit"),
                    required_events=(STYLE_STARTED, STYLE_SUBMITTED),
                    fork_from="reader_provider_session",
                    source_artifact="reader_edit",
                ),
                StageContract(
                    kind="corrective_gate",
                    label="corrective gate",
                    tool_prefix="plasma.report.long_form.final_edit",
                    tools=tool_surface("plasma.report.long_form.final_edit"),
                    required_events=(GATE_STARTED, GATE_SUBMITTED),
                    fork_from="report_plan_session",
                    source_artifact="style_edit",
                    canonicalizes=True,
                ),
            ),
        ),
    }


def pair_specs() -> tuple[PairSpec, ...]:
    topics = (
        ("wang-anshi-northern-song", "Wang Anshi and Northern Song reform memory"),
        ("go-raft-implementation-roadmap", "Go Raft implementation roadmap"),
    )
    return tuple(
        PairSpec(
            pair_id=f"{topic_id}-{rigor}",
            topic_id=topic_id,
            topic_title=title,
            rigor=rigor,
            fixed_part_manifest=f"fixed-inputs/{EVIDENCE_VERSION}/{topic_id}/{rigor}/parts.manifest.json",
            frozen_part_manifest_sha256_receipt=f"fixed-inputs/{EVIDENCE_VERSION}/{topic_id}/{rigor}/parts.manifest.sha256",
        )
        for topic_id, title in topics
        for rigor in ("exploratory", "strict")
    )


def planned_report_path(pair: PairSpec, arm: str) -> str:
    if arm not in (ARM_A, ARM_B):
        raise ValueError(f"unknown arm {arm!r}")
    return f"runs/{EVIDENCE_VERSION}/{pair.pair_id}/{arm}/report.md"


def planned_check_path(pair: PairSpec, arm: str) -> str:
    if arm not in (ARM_A, ARM_B):
        raise ValueError(f"unknown arm {arm!r}")
    return f"checks/{EVIDENCE_VERSION}/{pair.pair_id}/{arm}.machine_check.json"


def pair_specs_by_id() -> dict[str, PairSpec]:
    return {pair.pair_id: pair for pair in pair_specs()}


def build_manifest(archive_root: Path | None = None, repo_root: Path | None = None) -> dict[str, object]:
    archive = ensure_archive_outside_repo(archive_root or default_archive_root(), repo_root)
    arms = {key: _jsonable(contract) for key, contract in arm_contracts().items()}
    pairs = []
    for pair in pair_specs():
        pair_row = _jsonable(pair)
        pair_row["arm_inputs"] = {
            ARM_A: {
                "pipeline": PIPELINE_V1,
                "frozen_part_manifest": pair.fixed_part_manifest,
                "frozen_part_manifest_sha256_receipt": pair.frozen_part_manifest_sha256_receipt,
            },
            ARM_B: {
                "pipeline": PIPELINE_V2,
                "frozen_part_manifest": pair.fixed_part_manifest,
                "frozen_part_manifest_sha256_receipt": pair.frozen_part_manifest_sha256_receipt,
            },
        }
        pair_row["planned_outputs"] = {
            arm: {
                "report_markdown": planned_report_path(pair, arm),
                "machine_check": planned_check_path(pair, arm),
            }
            for arm in (ARM_A, ARM_B)
        }
        pairs.append(pair_row)
    manifest: dict[str, object] = {
        "experiment_id": EXPERIMENT_ID,
        "evidence_version": EVIDENCE_VERSION,
        "status": "w6_b_prepared_not_run",
        "archive_root": str(archive),
        "public_doc_dir": PUBLIC_DOC_DIR,
        "arms": arms,
        "pairs": pairs,
        "blind_labels": list(BLIND_LABELS),
        "blind_assignment": {
            "default": "private_local_randomness",
            "test_seed": "explicit --blind-seed only",
        },
        "blind_mapping_path": f"control/blind_mapping.{EVIDENCE_VERSION}.json",
        "blind_reading_pack_dir": f"reading-packs/{EVIDENCE_VERSION}/blind",
        "manual_adjudication_path": "control/manual-adjudication.json",
        "invalidated_prior_evidence": "control/w6-a-invalid-directional.json",
        "fixed_input_rules": [
            "A uses the current reader_style_gate_v1 product path.",
            "B uses assembly_writer_reader_style_gate_v2.",
            "A and B consume the same frozen reviewed Part artifact manifest for each pair.",
            "No raw prompts, provider outputs, session IDs, source corpora, or bulky reports are written to Git.",
        ],
        "hard_fail_count_keys": list(HARD_FAIL_COUNT_KEYS),
        "hard_fail_status_keys": list(HARD_FAIL_STATUS_KEYS),
        "acceptance_rule": {
            "candidate_arm": ARM_B,
            "minimum_equal_or_better_pairs": 3,
            "total_pairs": 4,
            "hard_fail_policy": "reject on any hard information, citation, requirement, unsupported external fact, prompt/stage parity, pair identity, or blinding failure",
            "structural_regression_policy": "reject on repeated v2 structural regression across both rigor levels of the same topic",
        },
    }
    validate_manifest(manifest, repo_root=repo_root)
    return manifest


def _jsonable(value: object) -> object:
    if hasattr(value, "__dataclass_fields__"):
        return asdict(value)
    return value


def _require_archive_relative(value: object) -> None:
    if not isinstance(value, str) or not value.strip():
        raise ValueError("archive path must be a non-empty string")
    path = PurePosixPath(value)
    if path.is_absolute() or ".." in path.parts:
        raise ValueError(f"archive path must be relative and confined: {value}")


def validate_manifest(manifest: Mapping[str, object], repo_root: Path | None = None) -> None:
    if manifest.get("experiment_id") != EXPERIMENT_ID:
        raise ValueError("manifest experiment_id changed")
    if manifest.get("evidence_version") != EVIDENCE_VERSION:
        raise ValueError("manifest evidence_version changed")
    ensure_archive_outside_repo(Path(str(manifest.get("archive_root", ""))), repo_root)
    if manifest.get("public_doc_dir") != PUBLIC_DOC_DIR:
        raise ValueError("manifest public_doc_dir changed")
    arms = manifest.get("arms")
    if not isinstance(arms, Mapping) or set(arms) != {ARM_A, ARM_B}:
        raise ValueError("manifest must contain exactly A and B arms")
    if arms[ARM_A]["pipeline"] != PIPELINE_V1 or arms[ARM_B]["pipeline"] != PIPELINE_V2:  # type: ignore[index]
        raise ValueError("manifest arm pipelines changed")
    pairs = manifest.get("pairs")
    if not isinstance(pairs, Sequence) or isinstance(pairs, (str, bytes)) or len(pairs) != 4:
        raise ValueError("manifest must contain four fixed-input pairs")
    expected_pairs = pair_specs_by_id()
    expected_pair_ids = set(expected_pairs)
    seen_pair_ids: set[str] = set()
    for row in pairs:
        if not isinstance(row, Mapping):
            raise ValueError("manifest pair must be an object")
        pair_id = str(row.get("pair_id", ""))
        if pair_id not in expected_pairs:
            raise ValueError("manifest pair set changed")
        seen_pair_ids.add(pair_id)
        expected = expected_pairs[pair_id]
        if row.get("topic_id") != expected.topic_id or row.get("rigor") != expected.rigor:
            raise ValueError(f"pair {pair_id} topic or rigor changed")
        if row.get("fixed_part_manifest") != expected.fixed_part_manifest:
            raise ValueError(f"pair {pair_id} fixed Part manifest path changed")
        if row.get("frozen_part_manifest_sha256_receipt") != expected.frozen_part_manifest_sha256_receipt:
            raise ValueError(f"pair {pair_id} frozen Part receipt path changed")
        _require_archive_relative(row.get("fixed_part_manifest"))
        _require_archive_relative(row.get("frozen_part_manifest_sha256_receipt"))
        inputs = row.get("arm_inputs")
        outputs = row.get("planned_outputs")
        if not isinstance(inputs, Mapping) or set(inputs) != {ARM_A, ARM_B}:
            raise ValueError(f"pair {pair_id} must bind A and B inputs")
        for arm, pipeline in ((ARM_A, PIPELINE_V1), (ARM_B, PIPELINE_V2)):
            input_row = inputs[arm]  # type: ignore[index]
            if input_row["pipeline"] != pipeline:  # type: ignore[index]
                raise ValueError(f"pair {pair_id} {arm} pipeline changed")
            if input_row["frozen_part_manifest"] != expected.fixed_part_manifest:  # type: ignore[index]
                raise ValueError(f"pair {pair_id} {arm} fixed Part manifest path changed")
            if input_row["frozen_part_manifest_sha256_receipt"] != expected.frozen_part_manifest_sha256_receipt:  # type: ignore[index]
                raise ValueError(f"pair {pair_id} {arm} frozen Part receipt path changed")
        if inputs[ARM_A]["frozen_part_manifest"] != inputs[ARM_B]["frozen_part_manifest"]:  # type: ignore[index]
            raise ValueError(f"pair {pair_id} does not share one frozen Part manifest")
        if not isinstance(outputs, Mapping) or set(outputs) != {ARM_A, ARM_B}:
            raise ValueError(f"pair {pair_id} must declare A and B outputs")
        for arm in (ARM_A, ARM_B):
            output = outputs[arm]  # type: ignore[index]
            if output["report_markdown"] != planned_report_path(expected, arm):  # type: ignore[index]
                raise ValueError(f"pair {pair_id} {arm} report path changed")
            if output["machine_check"] != planned_check_path(expected, arm):  # type: ignore[index]
                raise ValueError(f"pair {pair_id} {arm} machine-check path changed")
            _require_archive_relative(output["report_markdown"])  # type: ignore[index]
            _require_archive_relative(output["machine_check"])  # type: ignore[index]
    if seen_pair_ids != expected_pair_ids:
        raise ValueError("manifest pair set changed")


def expected_stage_contracts(arm: str, style_enabled: bool) -> tuple[StageContract, ...]:
    contracts = arm_contracts()
    if arm not in contracts:
        raise ValueError(f"unknown arm {arm!r}")
    stages = contracts[arm].stages
    if not style_enabled:
        stages = tuple(stage for stage in stages if stage.kind != "style_edit")
    gate_source = "style_edit" if style_enabled else "reader_edit"
    return tuple(replace(stage, source_artifact=gate_source) if stage.kind == "corrective_gate" else stage for stage in stages)


def validate_stage_trace(arm: str, trace: Sequence[Mapping[str, object]], style_enabled: bool) -> list[str]:
    contracts = arm_contracts()
    if arm not in contracts:
        raise ValueError(f"unknown arm {arm!r}")
    contract = contracts[arm]
    expected = expected_stage_contracts(arm, style_enabled)
    errors: list[str] = []
    if len(trace) != len(expected):
        errors.append(f"stage count {len(trace)} != expected {len(expected)}")
    for index, want in enumerate(expected):
        raw = trace[index] if index < len(trace) else {}
        if not isinstance(raw, Mapping):
            errors.append(f"stage {index} trace row is not an object")
            raw = {}
        got = raw
        if got.get("stage") != want.kind:
            errors.append(f"stage {index} kind {got.get('stage')!r} != {want.kind!r}")
        if got.get("label") != want.label:
            errors.append(f"stage {index} label {got.get('label')!r} != {want.label!r}")
        if got.get("pipeline") != contract.pipeline:
            errors.append(f"stage {index} pipeline {got.get('pipeline')!r} != {contract.pipeline!r}")
        if got.get("fork_from") != want.fork_from:
            errors.append(f"stage {index} fork_from {got.get('fork_from')!r} != {want.fork_from!r}")
        if got.get("source_artifact") != want.source_artifact:
            errors.append(f"stage {index} source_artifact {got.get('source_artifact')!r} != {want.source_artifact!r}")
        if got.get("canonicalizes") is not want.canonicalizes:
            errors.append(f"stage {index} canonicalizes {got.get('canonicalizes')!r} != {want.canonicalizes!r}")
        tools_value = got.get("tools")
        if not isinstance(tools_value, Sequence) or isinstance(tools_value, (str, bytes)):
            errors.append(f"stage {index} tools must be a list")
            tools = ()
        else:
            tools = tuple(str(tool) for tool in tools_value)
        if tools != want.tools:
            errors.append(f"stage {index} tools {tools!r} != {want.tools!r}")
        events_value = got.get("events")
        if not isinstance(events_value, Sequence) or isinstance(events_value, (str, bytes)):
            errors.append(f"stage {index} events must be a list")
            events = set()
        else:
            events = set(str(event) for event in events_value)
        missing = [event for event in want.required_events if event not in events]
        if missing:
            errors.append(f"stage {index} missing events: {', '.join(missing)}")
    return errors


def _require_bool(record: Mapping[str, object], key: str) -> bool:
    if key not in record or type(record[key]) is not bool:
        raise ValueError(f"{key} must be an explicit boolean")
    return bool(record[key])


def _require_count(record: Mapping[str, object], key: str) -> int:
    if key not in record or type(record[key]) is not int:
        raise ValueError(f"{key} must be an explicit non-negative integer")
    value = record[key]
    if value < 0:
        raise ValueError(f"{key} must be an explicit non-negative integer")
    return value


def _require_status(record: Mapping[str, object], key: str) -> str:
    if key not in record or not isinstance(record[key], str):
        raise ValueError(f"{key} must be an explicit status")
    status = record[key].strip().lower()
    if status not in ("pass", "fail"):
        raise ValueError(f"{key} has an invalid status")
    return status


def _require_stage_trace_errors(record: Mapping[str, object]) -> list[str]:
    if "stage_trace_errors" not in record:
        raise ValueError("stage_trace_errors must be explicit")
    value = record["stage_trace_errors"]
    if not isinstance(value, Sequence) or isinstance(value, (str, bytes)):
        raise ValueError("stage_trace_errors must be a list")
    return [str(item) for item in value]


def hard_fail_reasons(record: Mapping[str, object]) -> list[str]:
    reasons: list[str] = []
    for key in HARD_FAIL_COUNT_KEYS:
        value = _require_count(record, key)
        if value > 0:
            reasons.append(f"{key}={value}")
    for key in HARD_FAIL_STATUS_KEYS:
        status = _require_status(record, key)
        if status != "pass":
            reasons.append(f"{key}={status}")
    if _require_stage_trace_errors(record):
        reasons.append("stage_trace_errors")
    return reasons


def validate_pair_result_record(record: Mapping[str, object], expected_pair_ids: set[str]) -> str:
    if not isinstance(record, Mapping):
        raise ValueError("pair result must be an object")
    pair_id = record.get("pair_id")
    if not isinstance(pair_id, str) or pair_id not in expected_pair_ids:
        raise ValueError("pair result must name one preregistered pair")
    winner = record.get("winner")
    if not isinstance(winner, str) or winner not in VALID_WINNERS:
        raise ValueError(f"pair {pair_id} winner must be A, B, or tie")
    _require_bool(record, "hard_fail")
    _require_bool(record, "v2_structural_regression")
    hard_fail_reasons(record)
    return pair_id


def validate_manual_adjudication(record: Mapping[str, object]) -> None:
    if record.get("experiment_id") != EXPERIMENT_ID or record.get("evidence_version") != EVIDENCE_VERSION:
        raise ValueError("manual adjudication identity is invalid")
    if not isinstance(record.get("adjudicated_at"), str) or not str(record.get("adjudicated_at")).strip():
        raise ValueError("manual adjudication timestamp is required")
    pairs = record.get("pairs")
    expected = set(pair_specs_by_id())
    if not isinstance(pairs, Sequence) or isinstance(pairs, (str, bytes)) or len(pairs) != len(expected):
        raise ValueError("manual adjudication must cover exactly the four pairs")
    seen: set[str] = set()
    for item in pairs:
        if not isinstance(item, Mapping):
            raise ValueError("manual adjudication pair must be an object")
        pair_id = item.get("pair_id")
        if not isinstance(pair_id, str) or pair_id not in expected or pair_id in seen:
            raise ValueError("manual adjudication pair set is invalid")
        seen.add(pair_id)
        winner = item.get("direct_reading_winner")
        if winner not in VALID_WINNERS:
            raise ValueError(f"manual adjudication winner is invalid for {pair_id}")
        unsupported = item.get("unsupported_external_facts")
        if not isinstance(unsupported, Mapping) or set(unsupported) != {ARM_A, ARM_B}:
            raise ValueError(f"manual adjudication unsupported fact map is invalid for {pair_id}")
        for arm, value in unsupported.items():
            if type(value) is not int or value < 0:
                raise ValueError(f"manual adjudication unsupported fact count is invalid for {pair_id} {arm}")
        _require_bool(item, "v2_structural_regression")
        for key in ("inference_boundary", "reading_notes"):
            if not isinstance(item.get(key), str) or not str(item.get(key)).strip():
                raise ValueError(f"manual adjudication {key} is required for {pair_id}")
    if seen != expected:
        raise ValueError("manual adjudication must cover exactly the four pairs")


def validate_manual_adjudication_seal(record: Mapping[str, object], archive: Path) -> None:
    seal = load_blind_seal(archive)
    if record.get("blind_mapping_sha256") != seal["blind_mapping_sha256"]:
        raise ValueError("manual adjudication blind mapping digest mismatch")
    if record.get("blind_pack_sha256") != seal["blind_pack_sha256"]:
        raise ValueError("manual adjudication blind pack digest mismatch")


def load_manual_adjudication(archive_root: Path | None = None) -> Mapping[str, object]:
    archive = ensure_archive_outside_repo(archive_root or default_archive_root())
    path = archive / "control" / "manual-adjudication.json"
    if not path.is_file():
        raise ValueError(f"manual adjudication is missing: {path}")
    record = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(record, Mapping):
        raise ValueError("manual adjudication must be an object")
    validate_manual_adjudication(record)
    validate_manual_adjudication_seal(record, archive)
    return record


def acceptance_result(pair_results: Sequence[Mapping[str, object]]) -> dict[str, object]:
    expected_pairs = pair_specs_by_id()
    if not isinstance(pair_results, Sequence) or isinstance(pair_results, (str, bytes)) or len(pair_results) != len(expected_pairs):
        raise ValueError("acceptance calculation requires exactly the four preregistered pairs")
    seen_pairs: set[str] = set()
    for row in pair_results:
        pair_id = validate_pair_result_record(row, set(expected_pairs))
        if pair_id in seen_pairs:
            raise ValueError(f"duplicate pair result: {pair_id}")
        seen_pairs.add(pair_id)
    if seen_pairs != set(expected_pairs):
        raise ValueError("acceptance calculation requires exactly the four preregistered pairs")
    equal_or_better = 0
    hard_failures: list[str] = []
    regressions_by_topic: dict[str, set[str]] = {}
    for row in pair_results:
        pair_id = str(row["pair_id"])
        pair = expected_pairs[pair_id]
        winner = str(row["winner"]).strip()
        if winner in (ARM_B, "tie"):
            equal_or_better += 1
        reasons = hard_fail_reasons(row)
        if reasons or row["hard_fail"] is True:
            hard_failures.append(pair_id)
        if row["v2_structural_regression"] is True:
            regressions_by_topic.setdefault(pair.topic_id, set()).add(pair.rigor)
    repeated = sorted(topic for topic, rigors in regressions_by_topic.items() if {"exploratory", "strict"}.issubset(rigors))
    accepted = equal_or_better >= 3 and not hard_failures and not repeated
    return {
        "accepted": accepted,
        "candidate_arm": ARM_B,
        "equal_or_better_pairs": equal_or_better,
        "hard_failure_pairs": hard_failures,
        "repeated_structural_regression_topics": repeated,
}


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def _validate_sha256(value: object, context: str) -> str:
    if not isinstance(value, str):
        raise ValueError(f"{context} SHA-256 must be a string")
    digest = value.strip().lower()
    if len(digest) != 64 or any(char not in "0123456789abcdef" for char in digest):
        raise ValueError(f"{context} SHA-256 is invalid")
    return digest


def read_sha256_receipt(path: Path) -> str:
    if not path.is_file():
        raise ValueError(f"frozen Part manifest receipt is missing: {path}")
    parts = path.read_text(encoding="utf-8").strip().split()
    if not parts:
        raise ValueError(f"frozen Part manifest receipt is empty: {path}")
    first = parts[0]
    return _validate_sha256(first, "frozen Part manifest receipt")


def _require_string_list(value: object, context: str) -> list[str]:
    if not isinstance(value, Sequence) or isinstance(value, (str, bytes)):
        raise ValueError(f"{context} must be a non-empty string list")
    result = [str(item).strip() for item in value]
    if not result or any(not item for item in result):
        raise ValueError(f"{context} must be a non-empty string list")
    return result


def validate_frozen_part_manifest(path: Path, pair: PairSpec, archive: Path) -> None:
    try:
        manifest = json.loads(path.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        raise ValueError(f"frozen Part manifest is not valid JSON: {path}") from exc
    if not isinstance(manifest, Mapping):
        raise ValueError(f"frozen Part manifest must be an object: {path}")
    if (
        manifest.get("experiment_id") != EXPERIMENT_ID
        or manifest.get("pair_id") != pair.pair_id
        or manifest.get("topic_id") != pair.topic_id
        or manifest.get("rigor") != pair.rigor
    ):
        raise ValueError(f"frozen Part manifest identity mismatch for {pair.pair_id}")
    if manifest.get("source") != "product_reviewed_parts_from_upstream_section_fanout":
        raise ValueError(f"frozen Part manifest for {pair.pair_id} was not produced by the product-reviewed-Part prep path")
    prep = manifest.get("prep")
    if not isinstance(prep, Mapping):
        raise ValueError(f"frozen Part manifest prep provenance is missing for {pair.pair_id}")
    if prep.get("product_path") != "section_fanout_plan_requirement_sections_part_assembly_part_author":
        raise ValueError(f"frozen Part manifest prep product path changed for {pair.pair_id}")
    if prep.get("discarded_final_report") is not True:
        raise ValueError(f"prep final report discard receipt is missing for {pair.pair_id}")
    for key in ("mission_id", "pending_event_id", "plan_event_id", "db_path", "ledger_events_path", "ledger_events_sha256"):
        if not isinstance(prep.get(key), str) or not str(prep.get(key)).strip():
            raise ValueError(f"prep provenance {key} is missing for {pair.pair_id}")
    _validate_sha256(prep.get("ledger_events_sha256"), f"prep ledger receipt {pair.pair_id}")
    for key in ("source_snapshot_ids", "source_artifact_ids", "source_event_ids"):
        _require_string_list(prep.get(key), f"prep provenance {key} for {pair.pair_id}")
    parts = manifest.get("parts")
    if not isinstance(parts, Sequence) or isinstance(parts, (str, bytes)) or not parts:
        raise ValueError(f"frozen Part manifest for {pair.pair_id} must contain product-reviewed Parts")
    for index, part in enumerate(parts, start=1):
        if not isinstance(part, Mapping):
            raise ValueError(f"frozen Part {index} for {pair.pair_id} must be an object")
        markdown = part.get("markdown")
        if not isinstance(markdown, str) or not markdown.strip() or not any("\uac00" <= char <= "\ud7a3" for char in markdown):
            raise ValueError(f"frozen Part {index} for {pair.pair_id} must be Korean Markdown")
        if part.get("part_index") != index or not isinstance(part.get("title"), str) or not str(part.get("title")).strip():
            raise ValueError(f"frozen Part {index} for {pair.pair_id} has invalid identity")
        if part.get("sha256") != hashlib.sha256(markdown.encode("utf-8")).hexdigest():
            raise ValueError(f"frozen Part {index} digest mismatch for {pair.pair_id}")
        if not isinstance(part.get("word_count"), int) or int(part.get("word_count")) <= 0:
            raise ValueError(f"frozen Part {index} word count is invalid for {pair.pair_id}")
    validate_prep_provenance(archive, manifest, pair)


def validate_prep_provenance(archive: Path, manifest: Mapping[str, object], pair: PairSpec) -> None:
    prep = manifest.get("prep")
    if not isinstance(prep, Mapping):
        raise ValueError(f"prep provenance is missing for {pair.pair_id}")
    db_path = require_archive_contained(archive, Path(str(prep["db_path"])))
    ledger_path = require_archive_contained(archive, Path(str(prep["ledger_events_path"])))
    if not db_path.is_file():
        raise ValueError(f"prep DB is missing for {pair.pair_id}")
    if not ledger_path.is_file():
        raise ValueError(f"prep ledger export is missing for {pair.pair_id}")
    if sha256_file(ledger_path) != str(prep["ledger_events_sha256"]):
        raise ValueError(f"prep ledger receipt mismatch for {pair.pair_id}")
    db_events, raw_artifacts, snapshots = load_prep_sqlite(db_path)
    exported_events = json.loads(ledger_path.read_text(encoding="utf-8"))
    if not isinstance(exported_events, Sequence) or isinstance(exported_events, (str, bytes)):
        raise ValueError(f"prep ledger export must be a list for {pair.pair_id}")
    validate_prep_ledger_replay(db_events, exported_events, prep, manifest, pair)
    validate_prep_sources(archive, raw_artifacts, snapshots, prep, pair)
    validate_prep_part_artifacts(raw_artifacts, db_events, prep, manifest, pair)


def load_prep_sqlite(db_path: Path) -> tuple[list[dict[str, object]], dict[str, dict[str, object]], dict[str, dict[str, object]]]:
    uri = f"file:{db_path.as_posix()}?mode=ro&immutable=1"
    with closing(sqlite3.connect(uri, uri=True)) as conn:
        conn.row_factory = sqlite3.Row
        events = [
            {
                "event_id": row["event_id"],
                "mission_id": row["mission_id"],
                "sequence": row["sequence"],
                "event_type": row["event_type"],
                "payload": json.loads(row["payload_json"]),
            }
            for row in conn.execute(
                "select event_id, mission_id, sequence, event_type, payload_json from plasma_ledger_events order by sequence"
            )
        ]
        artifacts = {
            row["artifact_id"]: {
                "artifact_id": row["artifact_id"],
                "mission_id": row["mission_id"],
                "sha256": row["sha256"],
                "filename": row["filename"],
                "content": bytes(row["content_blob"]),
            }
            for row in conn.execute(
                "select artifact_id, mission_id, sha256, filename, content_blob from plasma_raw_artifacts"
            )
        }
        snapshots = {
            row["snapshot_id"]: {
                "snapshot_id": row["snapshot_id"],
                "mission_id": row["mission_id"],
                "external_source_id": row["external_source_id"],
                "connector_id": row["connector_id"],
                "connector_version": row["connector_version"],
                "content_hash_value": row["content_hash_value"],
            }
            for row in conn.execute(
                "select snapshot_id, mission_id, connector_id, external_source_id, connector_version, content_hash_value from plasma_source_snapshots"
            )
        }
    return events, artifacts, snapshots


def event_payload(event: Mapping[str, object]) -> Mapping[str, object]:
    payload = event.get("Payload", event.get("payload"))
    if isinstance(payload, str):
        payload = json.loads(payload)
    if not isinstance(payload, Mapping):
        return {}
    return payload


def canonical_payload(value: object) -> str:
    return json.dumps(value, ensure_ascii=False, sort_keys=True, separators=(",", ":"))


def validate_prep_ledger_replay(
    db_events: Sequence[Mapping[str, object]],
    exported_events: Sequence[Mapping[str, object]],
    prep: Mapping[str, object],
    manifest: Mapping[str, object],
    pair: PairSpec,
) -> None:
    if not db_events or len(db_events) != len(exported_events):
        raise ValueError(f"prep DB and ledger export counts differ for {pair.pair_id}")
    for db_event, exported in zip(db_events, exported_events):
        if (
            db_event["event_id"] != exported.get("EventID")
            or db_event["mission_id"] != exported.get("MissionID")
            or db_event["sequence"] != exported.get("Sequence")
            or db_event["event_type"] != exported.get("EventType")
            or canonical_payload(db_event["payload"]) != canonical_payload(event_payload(exported))
        ):
            raise ValueError(f"prep DB and ledger export diverge for {pair.pair_id}")
    counts: dict[str, int] = {}
    by_id: dict[str, Mapping[str, object]] = {}
    for event in db_events:
        counts[str(event["event_type"])] = counts.get(str(event["event_type"]), 0) + 1
        by_id[str(event["event_id"])] = event
    source_events = _require_string_list(prep.get("source_event_ids"), f"prep source_event_ids for {pair.pair_id}")
    parts = manifest["parts"]  # type: ignore[index]
    required = {
        "source.snapshotted": len(source_events),
        "report.draft.pending": 1,
        "report.plan.created": 1,
        "report.requirements.started": 1,
        "report.requirements.mapped": 1,
        "report.section.created": 1,
        "report.part_plan.created": len(parts),  # type: ignore[arg-type]
        "report.part_assembly.submitted": len(parts),  # type: ignore[arg-type]
        "report.part.created": len(parts),  # type: ignore[arg-type]
        "report.part_edit.started": len(parts),  # type: ignore[arg-type]
        "report.part.edited": len(parts),  # type: ignore[arg-type]
    }
    for event_type, minimum in required.items():
        if counts.get(event_type, 0) < minimum:
            raise ValueError(f"prep ledger missing {event_type} for {pair.pair_id}")
    if counts.get("report.artifact.created", 0):
        raise ValueError(f"prep final report was not discarded for {pair.pair_id}")
    for event_id, event_type in (
        (prep["pending_event_id"], "report.draft.pending"),
        (prep["plan_event_id"], "report.plan.created"),
        *[(event_id, "source.snapshotted") for event_id in source_events],
    ):
        event = by_id.get(str(event_id))
        if not event or event["event_type"] != event_type:
            raise ValueError(f"prep ledger event {event_id} missing for {pair.pair_id}")


def validate_prep_sources(
    archive: Path,
    artifacts: Mapping[str, Mapping[str, object]],
    snapshots: Mapping[str, Mapping[str, object]],
    prep: Mapping[str, object],
    pair: PairSpec,
) -> None:
    source_snapshot_ids = _require_string_list(prep.get("source_snapshot_ids"), f"source snapshots for {pair.pair_id}")
    source_artifact_ids = _require_string_list(prep.get("source_artifact_ids"), f"source artifacts for {pair.pair_id}")
    if len(source_snapshot_ids) != len(source_artifact_ids):
        raise ValueError(f"prep source provenance cardinality mismatch for {pair.pair_id}")
    for snapshot_id, artifact_id in zip(source_snapshot_ids, source_artifact_ids):
        snapshot = snapshots.get(snapshot_id)
        artifact = artifacts.get(artifact_id)
        if not snapshot or not artifact:
            raise ValueError(f"prep source snapshot/artifact missing for {pair.pair_id}")
        if snapshot["mission_id"] != prep["mission_id"] or artifact["mission_id"] != prep["mission_id"]:
            raise ValueError(f"prep source mission mismatch for {pair.pair_id}")
        if snapshot["connector_id"] != "experiment-archive" or snapshot["connector_version"] != EVIDENCE_VERSION:
            raise ValueError(f"prep source connector mismatch for {pair.pair_id}")
        source_path = require_archive_contained(archive, archive / "source-corpora" / str(snapshot["external_source_id"]))
        source_bytes = source_path.read_bytes()
        if artifact["content"] != source_bytes or artifact["sha256"] != hashlib.sha256(source_bytes).hexdigest():
            raise ValueError(f"prep source artifact byte mismatch for {pair.pair_id}")
        if snapshot["content_hash_value"] != artifact["sha256"]:
            raise ValueError(f"prep source snapshot hash mismatch for {pair.pair_id}")


def validate_prep_part_artifacts(
    artifacts: Mapping[str, Mapping[str, object]],
    db_events: Sequence[Mapping[str, object]],
    prep: Mapping[str, object],
    manifest: Mapping[str, object],
    pair: PairSpec,
) -> None:
    edited_artifacts = {
        str(event_payload(event).get("artifact_id"))
        for event in db_events
        if event["event_type"] == "report.part.edited"
        and event_payload(event).get("pending_event_id") == prep["pending_event_id"]
        and event_payload(event).get("plan_event_id") == prep["plan_event_id"]
    }
    for part in manifest["parts"]:  # type: ignore[index]
        if not isinstance(part, Mapping):
            raise ValueError(f"prep Part row is malformed for {pair.pair_id}")
        artifact_id = str(part.get("artifact_id", ""))
        artifact = artifacts.get(artifact_id)
        if not artifact:
            raise ValueError(f"prep Part artifact missing for {pair.pair_id}")
        markdown = str(part["markdown"])
        if artifact["mission_id"] != prep["mission_id"] or artifact["content"] != markdown.encode("utf-8") or artifact["sha256"] != part["sha256"]:
            raise ValueError(f"prep Part artifact byte mismatch for {pair.pair_id}")
        if artifact_id not in edited_artifacts:
            raise ValueError(f"prep Part edited event missing for {pair.pair_id}")


def load_frozen_part_receipts(archive_root: Path | None = None, repo_root: Path | None = None) -> dict[str, dict[str, dict[str, str]]]:
    manifest = build_manifest(archive_root, repo_root=repo_root)
    archive = Path(str(manifest["archive_root"]))
    receipts: dict[str, dict[str, dict[str, str]]] = {}
    for pair in pair_specs():
        manifest_path = archive / pair.fixed_part_manifest
        receipt_path = archive / pair.frozen_part_manifest_sha256_receipt
        if not manifest_path.is_file():
            raise ValueError(f"frozen Part manifest is missing: {manifest_path}")
        expected_digest = read_sha256_receipt(receipt_path)
        actual_digest = sha256_file(manifest_path)
        if actual_digest != expected_digest:
            raise ValueError(f"frozen Part manifest digest mismatch for {pair.pair_id}")
        validate_frozen_part_manifest(manifest_path, pair, archive)
        receipts[pair.pair_id] = {
            arm: {
                "pipeline": PIPELINE_V1 if arm == ARM_A else PIPELINE_V2,
                "frozen_part_manifest": pair.fixed_part_manifest,
                "frozen_part_manifest_sha256_receipt": pair.frozen_part_manifest_sha256_receipt,
                "frozen_part_manifest_sha256": actual_digest,
            }
            for arm in (ARM_A, ARM_B)
        }
    validate_fixed_input_receipts(manifest, receipts, repo_root=repo_root)
    return receipts


def validate_fixed_input_receipts(manifest: Mapping[str, object], receipts: Mapping[str, Mapping[str, Mapping[str, object]]], repo_root: Path | None = None) -> None:
    validate_manifest(manifest, repo_root=repo_root)
    pairs = {str(row["pair_id"]): row for row in manifest["pairs"]}  # type: ignore[index]
    if set(receipts) != set(pairs):
        raise ValueError("frozen Part receipts must cover exactly the four pairs")
    for pair_id, row in pairs.items():
        pair_receipts = receipts[pair_id]
        if not isinstance(pair_receipts, Mapping) or set(pair_receipts) != {ARM_A, ARM_B}:
            raise ValueError(f"frozen Part receipts for {pair_id} must bind A and B")
        expected_digest = ""
        for arm in (ARM_A, ARM_B):
            receipt = pair_receipts[arm]
            expected_input = row["arm_inputs"][arm]  # type: ignore[index]
            if receipt.get("pipeline") != expected_input["pipeline"]:  # type: ignore[index]
                raise ValueError(f"frozen Part receipt {pair_id} {arm} pipeline changed")
            if receipt.get("frozen_part_manifest") != expected_input["frozen_part_manifest"]:  # type: ignore[index]
                raise ValueError(f"frozen Part receipt {pair_id} {arm} manifest path changed")
            if receipt.get("frozen_part_manifest_sha256_receipt") != expected_input["frozen_part_manifest_sha256_receipt"]:  # type: ignore[index]
                raise ValueError(f"frozen Part receipt {pair_id} {arm} receipt path changed")
            digest = _validate_sha256(receipt.get("frozen_part_manifest_sha256"), f"frozen Part receipt {pair_id} {arm}")
            if expected_digest and digest != expected_digest:
                raise ValueError(f"frozen Part manifest SHA-256 differs between A and B for {pair_id}")
            expected_digest = digest


def build_blind_mapping(pairs: Sequence[PairSpec] | None = None, seed: int | None = None) -> dict[str, dict[str, str]]:
    rng = random.Random(seed) if seed is not None else random.SystemRandom()
    result: dict[str, dict[str, str]] = {}
    for pair in pairs or pair_specs():
        arms = [ARM_A, ARM_B]
        rng.shuffle(arms)
        result[pair.pair_id] = {BLIND_LABELS[0]: arms[0], BLIND_LABELS[1]: arms[1]}
    validate_blind_mapping(result, pairs or pair_specs())
    return result


def validate_blind_mapping(mapping: Mapping[str, Mapping[str, str]], pairs: Sequence[PairSpec] | None = None) -> None:
    expected = {pair.pair_id for pair in pairs or pair_specs()}
    if set(mapping) != expected:
        raise ValueError("blind mapping pair set changed")
    for pair_id, labels in mapping.items():
        if set(labels) != set(BLIND_LABELS) or set(labels.values()) != {ARM_A, ARM_B}:
            raise ValueError(f"blind mapping for {pair_id} must map two labels to A and B")


def load_or_create_blind_mapping(
    path: Path,
    pairs: Sequence[PairSpec] | None = None,
    seed: int | None = None,
) -> dict[str, dict[str, str]]:
    selected_pairs = pairs or pair_specs()
    if path.exists():
        raw = json.loads(path.read_text(encoding="utf-8"))
        if not isinstance(raw, Mapping):
            raise ValueError("blind mapping must be an object")
        mapping = {str(pair_id): {str(label): str(arm) for label, arm in labels.items()} for pair_id, labels in raw.items()}
        validate_blind_mapping(mapping, selected_pairs)
        return mapping
    mapping = build_blind_mapping(selected_pairs, seed=seed)
    write_json(path, mapping)
    return mapping


def _check_blinding_text(markdown: str) -> None:
    folded = markdown.lower()
    for term in BLINDING_LEAK_TERMS:
        if term.lower() in folded:
            raise ValueError(f"reading pack leaks arm identity term: {term}")


def render_pair_reading_pack(pair: PairSpec, mapping: Mapping[str, Mapping[str, str]], report_bodies: Mapping[str, str]) -> str:
    validate_blind_mapping({pair.pair_id: mapping[pair.pair_id]}, (pair,))
    lines = [
        f"# Blind Pair: {pair.topic_title}",
        "",
        f"- Pair ID: `{pair.pair_id}`",
        f"- Rigor: `{pair.rigor}`",
        "- Read the two Markdown reports directly before scoring.",
        "",
    ]
    for label in BLIND_LABELS:
        arm = mapping[pair.pair_id][label]
        body = report_bodies[arm].strip()
        _check_blinding_text(body)
        lines.extend((f"## {label.replace('_', ' ').title()}", "", body, ""))
    markdown = "\n".join(lines).rstrip() + "\n"
    _check_blinding_text(markdown)
    return markdown


def write_json(path: Path, value: object) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, ensure_ascii=False, indent=2, sort_keys=True) + "\n", encoding="utf-8")


def write_manifest(archive_root: Path | None = None) -> Path:
    manifest = build_manifest(archive_root)
    archive = Path(str(manifest["archive_root"]))
    path = archive / "control" / "manifest.json"
    write_json(path, manifest)
    return path


def write_blind_packs(archive_root: Path | None = None, seed: int | None = None) -> list[Path]:
    manifest = build_manifest(archive_root)
    archive = Path(str(manifest["archive_root"]))
    mapping = load_or_create_blind_mapping(archive / str(manifest["blind_mapping_path"]), seed=seed)
    written: list[Path] = []
    for pair in pair_specs():
        bodies = {
            arm: (archive / planned_report_path(pair, arm)).read_text(encoding="utf-8")
            for arm in (ARM_A, ARM_B)
        }
        output = archive / str(manifest["blind_reading_pack_dir"]) / f"{pair.pair_id}.md"
        output.parent.mkdir(parents=True, exist_ok=True)
        output.write_text(render_pair_reading_pack(pair, mapping, bodies), encoding="utf-8")
        written.append(output)
    index = archive / str(manifest["blind_reading_pack_dir"]) / "README.md"
    index.write_text(render_reading_pack_index(written), encoding="utf-8")
    written.append(index)
    write_blind_seal(archive, mapping, written)
    return written


def blind_mapping_path(archive: Path) -> Path:
    return archive / f"control/blind_mapping.{EVIDENCE_VERSION}.json"


def blind_pack_dir(archive: Path) -> Path:
    return archive / f"reading-packs/{EVIDENCE_VERSION}/blind"


def compute_blind_pack_digests(paths: Sequence[Path]) -> dict[str, str]:
    result: dict[str, str] = {}
    for path in paths:
        if path.name == "README.md":
            continue
        result[path.stem] = sha256_file(path)
    expected = set(pair_specs_by_id())
    if set(result) != expected:
        raise ValueError("blind pack digests must cover exactly the four pairs")
    return result


def write_blind_seal(archive: Path, mapping: Mapping[str, Mapping[str, str]], paths: Sequence[Path]) -> Path:
    mapping_path = blind_mapping_path(archive)
    require_archive_contained(archive, mapping_path)
    validate_blind_mapping(mapping)
    pack_digests = compute_blind_pack_digests(paths)
    seal = {
        "experiment_id": EXPERIMENT_ID,
        "evidence_version": EVIDENCE_VERSION,
        "blind_mapping_path": f"control/blind_mapping.{EVIDENCE_VERSION}.json",
        "blind_mapping_sha256": sha256_file(mapping_path),
        "blind_pack_dir": f"reading-packs/{EVIDENCE_VERSION}/blind",
        "blind_pack_sha256": pack_digests,
    }
    path = archive / BLIND_SEAL_PATH
    write_json(path, seal)
    return path


def load_blind_seal(archive_root: Path | None = None) -> Mapping[str, object]:
    archive = ensure_archive_outside_repo(archive_root or default_archive_root())
    path = archive / BLIND_SEAL_PATH
    if not path.is_file():
        raise ValueError(f"blind seal is missing: {path}")
    seal = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(seal, Mapping):
        raise ValueError("blind seal must be an object")
    mapping_path = blind_mapping_path(archive)
    if seal.get("experiment_id") != EXPERIMENT_ID or seal.get("evidence_version") != EVIDENCE_VERSION:
        raise ValueError("blind seal identity mismatch")
    if seal.get("blind_mapping_sha256") != sha256_file(mapping_path):
        raise ValueError("blind seal mapping digest mismatch")
    pack_paths = [blind_pack_dir(archive) / f"{pair.pair_id}.md" for pair in pair_specs()]
    if seal.get("blind_pack_sha256") != compute_blind_pack_digests(pack_paths):
        raise ValueError("blind seal pack digest mismatch")
    return seal


def validate_reading_results(archive_root: Path | None = None) -> Mapping[str, object]:
    archive = ensure_archive_outside_repo(archive_root or default_archive_root())
    path = archive / "control" / "reading-results.json"
    if not path.is_file():
        raise ValueError(f"reading results are missing: {path}")
    record = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(record, Mapping):
        raise ValueError("reading results must be an object")
    if record.get("experiment_id") != EXPERIMENT_ID or record.get("evidence_version") != EVIDENCE_VERSION:
        raise ValueError("reading result identity mismatch")
    seal = load_blind_seal(archive)
    if record.get("blind_mapping_sha256") != seal["blind_mapping_sha256"]:
        raise ValueError("reading result blind mapping digest mismatch")
    if record.get("blind_pack_sha256") != seal["blind_pack_sha256"]:
        raise ValueError("reading result blind pack digest mismatch")
    pairs = record.get("pairs")
    if not isinstance(pairs, Sequence) or isinstance(pairs, (str, bytes)):
        raise ValueError("reading results pairs are missing")
    acceptance_result(pairs)  # type: ignore[arg-type]
    return record


def immutable_input_paths(archive: Path) -> list[Path]:
    paths = [archive / planned_report_path(pair, arm) for pair in pair_specs() for arm in (ARM_A, ARM_B)]
    paths.extend(sorted((archive / "fixed-inputs" / EVIDENCE_VERSION).glob("**/*")))
    paths.extend(sorted((archive / "source-corpora").glob("**/*")))
    files = [require_archive_contained(archive, path) for path in paths if path.is_file()]
    if len([path for path in files if path.name == "report.md"]) != 8:
        raise ValueError("immutable input receipt must cover eight report.md files")
    if not any("fixed-inputs" in path.parts for path in files) or not any("source-corpora" in path.parts for path in files):
        raise ValueError("immutable input receipt must cover fixed-input and source files")
    return sorted(files)


def capture_immutable_input_hashes(archive_root: Path | None = None) -> dict[str, str]:
    archive = ensure_archive_outside_repo(archive_root or default_archive_root())
    return {path.relative_to(archive).as_posix(): sha256_file(path) for path in immutable_input_paths(archive)}


def write_input_receipt(archive_root: Path | None, receipt_rel: str) -> Path:
    archive = ensure_archive_outside_repo(archive_root or default_archive_root())
    path = archive / receipt_rel
    write_json(
        path,
        {
            "experiment_id": EXPERIMENT_ID,
            "evidence_version": EVIDENCE_VERSION,
            "files": capture_immutable_input_hashes(archive),
        },
    )
    return path


def verify_input_receipts_unchanged(archive_root: Path | None = None) -> None:
    archive = ensure_archive_outside_repo(archive_root or default_archive_root())
    pre_path = archive / PRE_MUTATION_RECEIPT
    post_path = archive / POST_MUTATION_RECEIPT
    if not pre_path.is_file() or not post_path.is_file():
        raise ValueError("W6-C pre/post immutable input receipts are required")
    pre = json.loads(pre_path.read_text(encoding="utf-8"))
    post = json.loads(post_path.read_text(encoding="utf-8"))
    if pre.get("files") != post.get("files") or pre.get("files") != capture_immutable_input_hashes(archive):
        raise ValueError("W6-C immutable report/fixed-input/source hashes changed")


def verify_invalid_unsealed_copy(archive_root: Path | None = None) -> None:
    archive = ensure_archive_outside_repo(archive_root or default_archive_root())
    invalid = archive / INVALID_UNSEALED_READING_DIR
    if not invalid.is_dir():
        raise ValueError(f"invalid unsealed archive copy is missing: {invalid}")
    required_pairs = pair_specs()
    path_pairs: list[tuple[Path, Path]] = [
        (archive / "control" / f"blind_mapping.{EVIDENCE_VERSION}.json", invalid / "control" / f"blind_mapping.{EVIDENCE_VERSION}.json"),
        (archive / "control" / "manual-adjudication.json", invalid / "control" / "manual-adjudication.json"),
        (archive / "control" / "reading-results.json", invalid / "control" / "reading-results.json"),
    ]
    for pair in required_pairs:
        path_pairs.append((blind_pack_dir(archive) / f"{pair.pair_id}.md", invalid / "reading-packs" / f"{pair.pair_id}.md"))
    path_pairs.append((blind_pack_dir(archive) / "README.md", invalid / "reading-packs" / "README.md"))
    for pair in required_pairs:
        for arm in (ARM_A, ARM_B):
            path_pairs.append((archive / planned_check_path(pair, arm), invalid / "checks" / pair.pair_id / f"{arm}.machine_check.json"))
    for active, copied in path_pairs:
        require_archive_contained(archive, active)
        require_archive_contained(archive, copied)
        if not active.is_file() or not copied.is_file():
            raise ValueError(f"invalid copy is incomplete: {copied}")
        if sha256_file(active) != sha256_file(copied):
            raise ValueError(f"invalid copy digest mismatch: {copied}")


def remove_unsealed_active_controls(archive: Path) -> None:
    for rel in (
        f"control/blind_mapping.{EVIDENCE_VERSION}.json",
        "control/manual-adjudication.json",
        "control/reading-results.json",
        BLIND_SEAL_PATH,
    ):
        path = archive / rel
        require_archive_contained(archive, path)
        if path.exists():
            path.unlink()
    pack_dir = blind_pack_dir(archive)
    require_archive_contained(archive, pack_dir)
    if pack_dir.exists():
        shutil.rmtree(pack_dir)


def seal_fresh_blind_packs(archive_root: Path | None = None, seed: int | None = None) -> Mapping[str, object]:
    archive = ensure_archive_outside_repo(archive_root or default_archive_root())
    verify_invalid_unsealed_copy(archive)
    write_input_receipt(archive, PRE_MUTATION_RECEIPT)
    remove_unsealed_active_controls(archive)
    written = write_blind_packs(archive, seed=seed)
    first = load_blind_seal(archive)
    written_again = write_blind_packs(archive, seed=seed)
    second = load_blind_seal(archive)
    if first != second or [sha256_file(path) for path in written] != [sha256_file(path) for path in written_again]:
        raise ValueError("fresh blind pack generation is not stable across verification call")
    write_input_receipt(archive, POST_MUTATION_RECEIPT)
    verify_input_receipts_unchanged(archive)
    return first


def render_reading_pack_index(paths: Sequence[Path]) -> str:
    lines = ["# Blind Reading Pack Index", ""]
    for path in sorted(paths):
        if path.name != "README.md":
            lines.append(f"- [{path.stem}]({path.name})")
    return "\n".join(lines).rstrip() + "\n"


def parse_args(argv: Sequence[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--action",
        choices=(
            "manifest",
            "write-manifest",
            "check-fixed-inputs",
            "check-manual-adjudication",
            "check-reading-results",
            "write-blind-packs",
            "check-blind-seal",
            "verify-invalid-copy",
            "seal-fresh-blind-packs",
            "write-pre-receipt",
            "write-post-receipt",
            "verify-inputs-unchanged",
        ),
        required=True,
    )
    parser.add_argument("--archive-root", type=Path, default=default_archive_root())
    parser.add_argument("--blind-seed", type=int, default=None, help="Optional deterministic seed for tests; omit for private local randomness.")
    return parser.parse_args(argv)


def main(argv: Sequence[str] | None = None) -> int:
    args = parse_args(argv or sys.argv[1:])
    if args.action == "manifest":
        print(json.dumps(build_manifest(args.archive_root), ensure_ascii=False, indent=2, sort_keys=True))
        return 0
    if args.action == "write-manifest":
        print(write_manifest(args.archive_root))
        return 0
    if args.action == "check-fixed-inputs":
        print(json.dumps(load_frozen_part_receipts(args.archive_root), ensure_ascii=False, indent=2, sort_keys=True))
        return 0
    if args.action == "check-manual-adjudication":
        print(json.dumps(load_manual_adjudication(args.archive_root), ensure_ascii=False, indent=2, sort_keys=True))
        return 0
    if args.action == "check-reading-results":
        print(json.dumps(validate_reading_results(args.archive_root), ensure_ascii=False, indent=2, sort_keys=True))
        return 0
    if args.action == "write-blind-packs":
        for path in write_blind_packs(args.archive_root, args.blind_seed):
            print(path)
        return 0
    if args.action == "check-blind-seal":
        print(json.dumps(load_blind_seal(args.archive_root), ensure_ascii=False, indent=2, sort_keys=True))
        return 0
    if args.action == "verify-invalid-copy":
        verify_invalid_unsealed_copy(args.archive_root)
        print("ok")
        return 0
    if args.action == "seal-fresh-blind-packs":
        print(json.dumps(seal_fresh_blind_packs(args.archive_root, args.blind_seed), ensure_ascii=False, indent=2, sort_keys=True))
        return 0
    if args.action == "write-pre-receipt":
        print(write_input_receipt(args.archive_root, PRE_MUTATION_RECEIPT))
        return 0
    if args.action == "write-post-receipt":
        print(write_input_receipt(args.archive_root, POST_MUTATION_RECEIPT))
        return 0
    if args.action == "verify-inputs-unchanged":
        verify_input_receipts_unchanged(args.archive_root)
        print("ok")
        return 0
    raise AssertionError(f"unhandled action {args.action}")


if __name__ == "__main__":
    raise SystemExit(main())
