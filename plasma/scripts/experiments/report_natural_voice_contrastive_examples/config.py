from __future__ import annotations

import hashlib
from pathlib import Path


EXPERIMENT_ID = "59-report-natural-voice-contrastive-examples-2026-07-31"
ARCHIVE_SUFFIX = Path("research-artifacts") / "liquid2" / "plasma" / "experiments" / EXPERIMENT_ID
MODEL = "gpt-5.5"
REASONING_EFFORT = "medium"
ARMS = ("control", "contrastive")
BASE_PROMPT_SHA256 = "4922b0cc2774dfe972c5403603f0dd8fe6a0e172ec2ef838fdc54ff039ee565f"
CONTROL_PROMPT_SHA256 = "f8d8362f08086a89e8abd1f388bcbf0ff105ed9b87e7f2544da5364298bac39d"
SCHEDULE_SEED = 5900

EXPERIMENT_57_SUFFIX = (
    Path("research-artifacts")
    / "liquid2"
    / "plasma"
    / "experiments"
    / "57-report-natural-voice-selective-acceptance-2026-07-30"
)
EXPERIMENT_58_SUFFIX = (
    Path("research-artifacts")
    / "liquid2"
    / "plasma"
    / "experiments"
    / "58-report-natural-voice-examples-2026-07-30"
)
BASE_PROMPT_SOURCE = EXPERIMENT_57_SUFFIX / "control" / "instruction-prompt.lock.md"
CONTROL_PROMPT_SOURCE = EXPERIMENT_58_SUFFIX / "control" / "prompts" / "examples.lock.md"

DEVELOPMENT_SOURCES = {
    "dev-01-wang-anshi-strict.md": EXPERIMENT_58_SUFFIX / "inputs" / "development" / "dev-01-wang-anshi-strict.md",
    "dev-02-go-raft-exploratory.md": EXPERIMENT_58_SUFFIX / "inputs" / "development" / "dev-02-go-raft-exploratory.md",
}

EVALUATION_SOURCES = {
    filename: EXPERIMENT_58_SUFFIX / "inputs" / "evaluation" / filename
    for filename in (
        "01-mortgage-baseline.md",
        "02-mortgage-reader.md",
        "03-hand-washing-baseline.md",
        "04-vaccination-reader.md",
        "05-road-safety-baseline.md",
        "06-road-safety-reader.md",
        "07-earthquake-baseline.md",
        "08-earthquake-curiosity.md",
    )
}

SUCCESS_CRITERIA = {
    "minimum_contrastive_wins": 5,
    "minimum_clear_or_large_contrastive_wins": 3,
    "maximum_clear_or_large_contrastive_losses": 0,
    "maximum_contrastive_semantic_drift": 0,
    "maximum_contrastive_claim_scope_drift": 0,
    "maximum_contrastive_citation_drift": 0,
}

CONTRACT_AMENDMENT_RULE = (
    "append each allowed edit category missing from diagnoses with the sorted unique "
    "line numbers of edits in that category; do not change any edit; fail if diagnoses exceed six"
)
HASH_AMENDMENT_RULE = (
    "replace original_line_sha256 only when line_number identifies an existing source line "
    "and original_line is byte-identical to that line; do not change line_number or text"
)


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
