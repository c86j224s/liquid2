from __future__ import annotations

import hashlib
import re
from pathlib import Path


EXPERIMENT_56_ID = "56-report-natural-voice-correction-2026-07-30"
EXPERIMENT_57_ID = "57-report-natural-voice-selective-acceptance-2026-07-30"
EXPERIMENT_ID = EXPERIMENT_56_ID
SOURCE_EXPERIMENT_ID = "55-final-writer-v2-2026-07-29"
EXPERIMENT_57_FROZEN_PROMPT_SHA256 = "4922b0cc2774dfe972c5403603f0dd8fe6a0e172ec2ef838fdc54ff039ee565f"
EXPERIMENT_CHOICES = ("56", "57")
EXPERIMENT_IDS_BY_SELECTOR = {
    "56": EXPERIMENT_56_ID,
    "57": EXPERIMENT_57_ID,
}
EXPERIMENT_ARCHIVE_SUFFIXES = {
    "56": Path("research-artifacts") / "liquid2" / "plasma" / "experiments" / EXPERIMENT_56_ID,
    "57": Path("research-artifacts") / "liquid2" / "plasma" / "experiments" / EXPERIMENT_57_ID,
}
ARCHIVE_SUFFIX = EXPERIMENT_ARCHIVE_SUFFIXES["56"]
SOURCE_SUFFIX = (
    Path("research-artifacts")
    / "liquid2"
    / "plasma"
    / "experiments"
    / SOURCE_EXPERIMENT_ID
    / "reading-packs"
    / "w6-b-product-reviewed-parts"
    / "blind-split-ab"
)

MODEL = "gpt-5.5"
REASONING_EFFORT = "medium"

ALLOWED_CATEGORIES = (
    "translationese_nominalization",
    "formulaic_connection",
    "uniform_cadence",
    "process_narration",
    "inflated_abstraction",
    "redundant_framing",
)

CLI_ACTIONS = (
    "verify-source-seal",
    "lint-prompt",
    "run-pilot",
    "freeze-prompt",
    "run-full",
    "make-blind-packets",
    "record-host-verdicts",
    "export-public-summary",
)

EXPECTED_SHA256_BY_FILENAME = {
    "01-wang-anshi-exploratory-read-A.md": "9d61c098c5db590bbf6b9f5a2565621e6c678114fb7fe8d6a08b6259f74baf30",
    "01-wang-anshi-exploratory-read-B.md": "998a4802eea587a5d303938910fee5eed3ee729b8c37d33e384ffc4beb4f4e3c",
    "02-wang-anshi-strict-read-A.md": "053f8c1ab78d7fcd8f1ae5812caea93845f0ed5955dc45279a86e16cc13c584c",
    "02-wang-anshi-strict-read-B.md": "d4fc2e21030f5de11d1325d6cc549494113f3176bd2bd12d3693993a81321ab7",
    "03-go-raft-exploratory-read-A.md": "55bb158d23c580f9aa476a97ae5bb72549ce4b52408415b94a7e11a055b79035",
    "03-go-raft-exploratory-read-B.md": "bcb288f42f312626d3eab62da6a71b7084296397bbbf922f3d72e3c4e739b7fd",
    "04-go-raft-strict-read-A.md": "d41b3584d07dc1a07f31cdde76b07be30c7c3766c2c7823ea2f4417e3dbb3291",
    "04-go-raft-strict-read-B.md": "3df830e8b2ce6a9f2c8cbbd9c4d1e36e9500f15b34fac339ca8c9aeaa4d1dd2d",
}
EXPECTED_FILENAMES = tuple(EXPECTED_SHA256_BY_FILENAME)
HEX64_RE = re.compile(r"^[0-9a-f]{64}$")


def repo_root() -> Path:
    return Path(__file__).resolve().parents[4]


def resolve_experiment(experiment: str = "56") -> str:
    selector = str(experiment)
    if selector not in EXPERIMENT_CHOICES:
        raise ValueError("experiment must be 56 or 57")
    return selector


def experiment_id(experiment: str = "56") -> str:
    return EXPERIMENT_IDS_BY_SELECTOR[resolve_experiment(experiment)]


def archive_suffix(experiment: str = "56") -> Path:
    return EXPERIMENT_ARCHIVE_SUFFIXES[resolve_experiment(experiment)]


def fixed_archive_root(home: Path | None = None, experiment: str = "56") -> Path:
    return ((home or Path.home()) / archive_suffix(experiment)).resolve()


def fixed_source_root(home: Path | None = None) -> Path:
    return ((home or Path.home()) / SOURCE_SUFFIX).resolve()


def resolve_archive(path: str | Path | None = None, home: Path | None = None, experiment: str = "56") -> Path:
    selector = resolve_experiment(experiment)
    expected = fixed_archive_root(home, selector)
    candidate = expected if path is None else Path(path).expanduser().resolve()
    if candidate != expected:
        raise ValueError(f"archive must resolve exactly to the fixed experiment {selector} archive")
    return candidate


def file_id_for_filename(filename: str) -> str:
    if not filename.endswith(".md"):
        raise ValueError(f"corpus filename must be Markdown: {filename}")
    return filename[:-3]


def sha256_bytes(value: bytes) -> str:
    return hashlib.sha256(value).hexdigest()


def sha256_text(value: str) -> str:
    return sha256_bytes(value.encode("utf-8"))


def sha256_file(path: Path) -> str:
    return sha256_bytes(path.read_bytes())


def is_relative_to(path: Path, root: Path) -> bool:
    try:
        path.resolve().relative_to(root.resolve())
        return True
    except ValueError:
        return False
