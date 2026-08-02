from __future__ import annotations

from pathlib import Path

from report_natural_voice_examples.archive import ExperimentArchive
from report_natural_voice_examples.prompts import PromptError, freeze_protocol as freeze_base_protocol

from . import config
from .context import require_active


def freeze_protocol(archive: ExperimentArchive) -> dict[str, object]:
    require_active()
    prompt = (archive.home / config.EXAMPLES_PROMPT_SOURCE).resolve()
    if not prompt.is_file() or prompt.is_symlink():
        raise PromptError("sealed experiment 58 example prompt is missing")
    if _sha256_file(prompt) != config.EXAMPLES_PROMPT_SHA256:
        raise PromptError("sealed experiment 58 example prompt changed")
    return freeze_base_protocol(archive, prompt)


def _sha256_file(path: Path) -> str:
    import hashlib

    return hashlib.sha256(path.read_bytes()).hexdigest()
