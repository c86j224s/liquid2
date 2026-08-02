from __future__ import annotations

import hashlib
from pathlib import Path


EXPERIMENT_ID = "58-report-natural-voice-examples-2026-07-30"
ARCHIVE_SUFFIX = Path("research-artifacts") / "liquid2" / "plasma" / "experiments" / EXPERIMENT_ID
MODEL = "gpt-5.5"
REASONING_EFFORT = "medium"
ARMS = ("control", "examples")
CONTROL_PROMPT_SHA256 = "4922b0cc2774dfe972c5403603f0dd8fe6a0e172ec2ef838fdc54ff039ee565f"
SCHEDULE_SEED = 5800

EXPERIMENT_57_SUFFIX = (
    Path("research-artifacts")
    / "liquid2"
    / "plasma"
    / "experiments"
    / "57-report-natural-voice-selective-acceptance-2026-07-30"
)
EXPERIMENT_33_SUFFIX = (
    Path("research-artifacts")
    / "liquid2"
    / "plasma"
    / "experiments"
    / "33-report-direct-explanation-writing-2026-07-24"
)
CONTROL_PROMPT_SOURCE = EXPERIMENT_57_SUFFIX / "control" / "instruction-prompt.lock.md"

DEVELOPMENT_SOURCES = {
    "dev-01-wang-anshi-strict.md": EXPERIMENT_57_SUFFIX / "inputs" / "02-wang-anshi-strict-read-A.md",
    "dev-02-go-raft-exploratory.md": EXPERIMENT_57_SUFFIX / "inputs" / "03-go-raft-exploratory-read-B.md",
}

EVALUATION_SOURCES = {
    "01-mortgage-baseline.md": EXPERIMENT_33_SUFFIX / "runs" / "consumer-finance-b-baseline" / "report.md",
    "02-mortgage-reader.md": EXPERIMENT_33_SUFFIX / "runs" / "consumer-finance-b-reader_paragraph_contract" / "report.md",
    "03-hand-washing-baseline.md": EXPERIMENT_33_SUFFIX / "runs" / "public-health-guidance-a-baseline" / "report.md",
    "04-vaccination-reader.md": EXPERIMENT_33_SUFFIX / "runs" / "public-health-guidance-b-reader_paragraph_contract" / "report.md",
    "05-road-safety-baseline.md": EXPERIMENT_33_SUFFIX / "runs" / "transport-safety-a-baseline" / "report.md",
    "06-road-safety-reader.md": EXPERIMENT_33_SUFFIX / "runs" / "transport-safety-a-reader_paragraph_contract" / "report.md",
    "07-earthquake-baseline.md": EXPERIMENT_33_SUFFIX / "runs" / "disaster-preparedness-b-baseline" / "report.md",
    "08-earthquake-curiosity.md": EXPERIMENT_33_SUFFIX / "runs" / "disaster-preparedness-b-curiosity_led_explanation" / "report.md",
}

SUCCESS_CRITERIA = {
    "minimum_example_wins": 7,
    "minimum_clear_or_large_example_wins": 6,
    "maximum_clear_or_large_example_losses": 0,
    "maximum_semantic_drift": 0,
    "maximum_claim_scope_drift": 0,
    "maximum_citation_drift": 0,
}


def fixed_archive_root(home: Path | None = None) -> Path:
    return ((home or Path.home()) / ARCHIVE_SUFFIX).resolve()


def file_id(filename: str) -> str:
    if not filename.endswith(".md"):
        raise ValueError(f"expected Markdown filename: {filename}")
    return filename[:-3]


def sha256_bytes(value: bytes) -> str:
    return hashlib.sha256(value).hexdigest()


def sha256_text(value: str) -> str:
    return sha256_bytes(value.encode("utf-8"))


def sha256_file(path: Path) -> str:
    return sha256_bytes(path.read_bytes())


def display_home_path(path: Path, home: Path) -> str:
    return "~/" + path.resolve().relative_to(home.resolve()).as_posix()


def file_id_set() -> tuple[str, ...]:
    return tuple(file_id(name) for name in EVALUATION_SOURCES)
