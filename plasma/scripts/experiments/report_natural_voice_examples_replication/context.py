from __future__ import annotations

from contextlib import contextmanager
from typing import Iterator

from report_natural_voice_examples import config as base_config
from report_natural_voice_examples import records as base_records
from report_natural_voice_examples.policy import (
    apply_selective_policy as apply_base_policy,
    validate_policy_artifacts as validate_base_policy_artifacts,
)

from . import config
from .hash_contract import normalize_response_hashes, validate_hash_artifacts


_CONFIG_KEYS = (
    "EXPERIMENT_ID",
    "ARCHIVE_SUFFIX",
    "MODEL",
    "REASONING_EFFORT",
    "ARMS",
    "CONTROL_PROMPT_SHA256",
    "SCHEDULE_SEED",
    "CONTROL_PROMPT_SOURCE",
    "DEVELOPMENT_SOURCES",
    "EVALUATION_SOURCES",
    "SUCCESS_CRITERIA",
)


@contextmanager
def activated() -> Iterator[None]:
    """Bind experiment 58's sealed machinery to experiment 60 for one action."""
    original = {key: getattr(base_config, key) for key in _CONFIG_KEYS}
    replacements = {
        "EXPERIMENT_ID": config.EXPERIMENT_ID,
        "ARCHIVE_SUFFIX": config.ARCHIVE_SUFFIX,
        "MODEL": config.MODEL,
        "REASONING_EFFORT": config.REASONING_EFFORT,
        "ARMS": config.ARMS,
        "CONTROL_PROMPT_SHA256": config.CONTROL_PROMPT_SHA256,
        "SCHEDULE_SEED": config.SCHEDULE_SEED,
        "CONTROL_PROMPT_SOURCE": config.CONTROL_PROMPT_SOURCE,
        "DEVELOPMENT_SOURCES": config.DEVELOPMENT_SOURCES,
        "EVALUATION_SOURCES": config.EVALUATION_SOURCES,
        "SUCCESS_CRITERIA": config.SUCCESS_CRITERIA,
    }
    original_apply = base_records.apply_selective_policy
    original_validate = base_records.validate_policy_artifacts

    def apply_policy(archive, run_dir, file_id, arm, original_text, response):
        normalized, hash_fields = normalize_response_hashes(
            archive, run_dir, original_text, response
        )
        candidate, policy_fields = apply_base_policy(
            archive, run_dir, file_id, arm, original_text, normalized
        )
        return candidate, {**policy_fields, **hash_fields}

    def validate_policy(archive, record, run_dir):
        validate_base_policy_artifacts(archive, record, run_dir)
        validate_hash_artifacts(archive, record, run_dir)

    try:
        for key, value in replacements.items():
            setattr(base_config, key, value)
        base_records.apply_selective_policy = apply_policy
        base_records.validate_policy_artifacts = validate_policy
        yield
    finally:
        base_records.apply_selective_policy = original_apply
        base_records.validate_policy_artifacts = original_validate
        for key, value in original.items():
            setattr(base_config, key, value)


def require_active() -> None:
    if base_config.EXPERIMENT_ID != config.EXPERIMENT_ID:
        raise RuntimeError("experiment 60 context is not active")
