"""Immutable manifest models for the report-plan MCP experiment."""

from __future__ import annotations

from dataclasses import asdict, dataclass, field
from pathlib import Path
from typing import Mapping
import hashlib
import json


EXPERIMENT_ID = "17-report-plan-mcp-focused-2026-07-14"
BASELINE_COMMIT = "15cde729f1dca1b6090711a095fdebc713257c6e"
CANDIDATE_COMMIT = "1b6239805f2dde41f7aaab36d8025812623da5a6"
ARCHIVE_SUFFIX = Path("research-artifacts/liquid2/plasma/experiments") / EXPERIMENT_ID
FORBIDDEN_PORTS = frozenset({3001, 3002, 3011, 6001, 6002, 6011})

TOPIC_DOMAINS = (
    "public-health-guidance",
    "transport-safety",
    "disaster-preparedness",
    "consumer-finance",
    "labor-statistics",
    "energy-efficiency",
    "climate-adaptation",
    "accessibility",
    "cybersecurity-guidance",
    "open-source-governance",
    "public-procurement",
    "education-policy",
)
PREREGISTERED_TOPICS = tuple(
    f"{domain}-{variant}" for domain in TOPIC_DOMAINS for variant in ("a", "b")
)
PROVIDER_EXECUTOR = "codex"
PREFLIGHT_MODEL = "preflight-only-not-a-codex-model"


@dataclass(frozen=True)
class RunSpec:
    topic: str
    replicate: int
    arm: str
    mode: str
    executor: str
    commit: str
    binary: Path
    model: str
    effort: str
    source_policy: str
    token_budget: int
    time_budget_seconds: int
    session_policy: str
    source_bundle: Path
    source_hash: str
    nonce: str

    def __post_init__(self) -> None:
        if self.executor != PROVIDER_EXECUTOR:
            raise ValueError("run executor must be codex")
        if self.effort != "high":
            raise ValueError("run effort must be high")
        if not self.model.strip():
            raise ValueError("run model must be non-blank")


@dataclass(frozen=True)
class RunManifest:
    experiment: str
    topic: str
    replicate: int
    arm: str
    mode: str
    executor: str
    commit: str
    binary: str
    binary_hash: str
    model: str
    effort: str
    source_policy: str
    source_bundle: str
    source_hash: str
    budgets: Mapping[str, int]
    selected_session_policy: str
    database: str
    artifact_root: str
    workdir: str
    port: int
    connector_port: int
    connector_url: str
    namespace: str
    child_environment: Mapping[str, str | None]
    mission_id: str | None = None
    process_id: int | None = None
    connector_process_id: int | None = None
    start_boundary: str = "not_started:first_cli_mutation_or_report_agent_request"
    terminal_status: str = "preflight"
    ledger_hash: str | None = None
    result_hash: str | None = None
    commands: tuple[tuple[str, ...], ...] = field(default_factory=tuple)

    def as_dict(self) -> dict[str, object]:
        return asdict(self)


def executor_for_mode(mode: str) -> str:
    if mode not in {"planned", "long_form"}:
        raise ValueError(f"unsupported report mode: {mode}")
    return PROVIDER_EXECUTOR


def validate_provider_models(value: object) -> dict[str, str]:
    if not isinstance(value, Mapping) or set(value) != {PROVIDER_EXECUTOR}:
        raise ValueError("models must contain exactly the codex key")
    model = value[PROVIDER_EXECUTOR]
    if not isinstance(model, str) or not model.strip():
        raise ValueError("codex model identifier must be a non-blank string")
    return {PROVIDER_EXECUTOR: model.strip()}


def model_for_mode(mode: str, value: object) -> str:
    return validate_provider_models(value)[executor_for_mode(mode)]


def validate_provider_efforts(value: object) -> dict[str, str]:
    if not isinstance(value, Mapping) or set(value) != {PROVIDER_EXECUTOR}:
        raise ValueError("efforts must contain exactly the codex key")
    effort = value[PROVIDER_EXECUTOR]
    if not isinstance(effort, str) or effort.strip().lower() != "high":
        raise ValueError("codex effort must be high")
    return {PROVIDER_EXECUTOR: "high"}


def effort_for_mode(mode: str, value: object) -> str:
    return validate_provider_efforts(value)[executor_for_mode(mode)]


@dataclass(frozen=True)
class Fixture:
    topic: str
    title: str
    objective: str
    source_bundle: Path
    source_sha256: str
    license: str
    license_url: str
    retrieved_at: str = ""


def load_and_validate_fixtures(
    path: Path, archive: Path, minimum: int = 12, maximum: int = 24, require_registered: bool = True,
) -> tuple[Fixture, ...]:
    raw = json.loads(path.read_text(encoding="utf-8"))
    rows = raw.get("fixtures") if isinstance(raw, dict) else None
    if not isinstance(rows, list) or len(rows) < minimum or len(rows) > maximum:
        raise ValueError(f"fixture manifest must contain {minimum}-{maximum} independent topics")
    fixtures: list[Fixture] = []
    topics: set[str] = set()
    for row in rows:
        if not isinstance(row, dict):
            raise ValueError("fixture entry must be an object")
        required = {"topic", "title", "objective", "source_bundle", "source_sha256", "license", "license_url", "retrieved_at"}
        if not required.issubset(row) or not all(str(row[key]).strip() for key in required):
            raise ValueError("fixture entry is incomplete")
        topic = str(row["topic"])
        safe_topic = topic and all(character.isalnum() or character in "-_" for character in topic)
        if not safe_topic or (require_registered and topic not in PREREGISTERED_TOPICS) or topic in topics:
            raise ValueError("fixture topic is unknown or duplicated")
        source = Path(str(row["source_bundle"])).expanduser().resolve()
        if not source.is_relative_to(archive.resolve()) or not source.is_file():
            raise ValueError("fixture source must be an archive-local file")
        if _sha256(source) != str(row["source_sha256"]):
            raise ValueError("fixture source hash mismatch")
        license_name = str(row["license"]).strip().lower()
        if license_name in {"unknown", "unverified", "none"}:
            raise ValueError("fixture license is not verified")
        fixtures.append(Fixture(topic, str(row["title"]), str(row["objective"]), source, str(row["source_sha256"]), str(row["license"]), str(row["license_url"]), str(row["retrieved_at"])))
        topics.add(topic)
    return tuple(fixtures)


def freeze_fixture_manifest(fixtures: tuple[Fixture, ...], destination: Path) -> str:
    payload = [{**asdict(item), "source_bundle": str(item.source_bundle)} for item in fixtures]
    encoded = (json.dumps({"fixtures": payload}, indent=2, sort_keys=True) + "\n").encode()
    destination.parent.mkdir(parents=True, exist_ok=True)
    with destination.open("xb") as handle:
        handle.write(encoded)
    return hashlib.sha256(encoded).hexdigest()


def _sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()
