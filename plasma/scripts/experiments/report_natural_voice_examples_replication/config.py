from __future__ import annotations

from pathlib import Path


EXPERIMENT_ID = "60-report-natural-voice-examples-replication-2026-07-31"
ARCHIVE_SUFFIX = Path("research-artifacts") / "liquid2" / "plasma" / "experiments" / EXPERIMENT_ID
MODEL = "gpt-5.5"
REASONING_EFFORT = "medium"
ARMS = ("control", "examples")
SCHEDULE_SEED = 6000

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
SOURCE_SUFFIX = (
    Path("research-artifacts")
    / "liquid2"
    / "plasma"
    / "experiments"
    / "27-report-human-writer-multitopic-ab-2026-07-15"
    / "expanded36-reserve12-r4"
)

CONTROL_PROMPT_SOURCE = EXPERIMENT_57_SUFFIX / "control" / "instruction-prompt.lock.md"
EXAMPLES_PROMPT_SOURCE = EXPERIMENT_58_SUFFIX / "control" / "prompts" / "examples.lock.md"
CONTROL_PROMPT_SHA256 = "4922b0cc2774dfe972c5403603f0dd8fe6a0e172ec2ef838fdc54ff039ee565f"
EXAMPLES_PROMPT_SHA256 = "f8d8362f08086a89e8abd1f388bcbf0ff105ed9b87e7f2544da5364298bac39d"

DEVELOPMENT_SOURCES = {
    "pilot-01-community-education.md": SOURCE_SUFFIX / "06-C05-long_form-candidate" / "artifacts" / "report.md",
    "pilot-02-intangible-cultural-heritage.md": (
        SOURCE_SUFFIX / "27-H05-long_form-candidate" / "artifacts" / "report.md"
    ),
}

EVALUATION_SOURCES = {
    "01-adult-education.md": SOURCE_SUFFIX / "03-C04-long_form-candidate" / "artifacts" / "report.md",
    "02-inflation.md": SOURCE_SUFFIX / "11-E04-long_form-candidate" / "artifacts" / "report.md",
    "03-request-for-proposal.md": SOURCE_SUFFIX / "19-G04-long_form-candidate" / "artifacts" / "report.md",
    "04-silk-road.md": SOURCE_SUFFIX / "22-H04-long_form-candidate" / "artifacts" / "report.md",
    "05-public-transport.md": SOURCE_SUFFIX / "30-L04-long_form-candidate" / "artifacts" / "report.md",
    "06-token-usage-metering.md": SOURCE_SUFFIX / "35-P04-long_form-candidate" / "artifacts" / "report.md",
    "07-flood-management.md": SOURCE_SUFFIX / "43-S04-long_form-candidate" / "artifacts" / "report.md",
    "08-multifactor-authentication.md": SOURCE_SUFFIX / "46-T04-long_form-candidate" / "artifacts" / "report.md",
}

# These are interpretation rules, not a single adoption gate. The historical
# runner records this object under its success_criteria field for seal parity.
INTERPRETATION_RULES = {
    "decision_mode": "separate_dimensions_without_overall_pass",
    "reading_efficacy": {
        "directional_support": "examples_wins > control_wins and signed_magnitude_score > 0",
        "directional_contradiction": "control_wins > examples_wins and signed_magnitude_score < 0",
        "otherwise": "mixed",
        "population_inference": "not_evaluated",
    },
    "semantic_safety": {
        "candidate_observation": "no_drift_observed only when every examples-arm drift count is zero",
        "control_and_candidate_reported_separately": True,
    },
    "product_readiness": "not_evaluated_by_this_replication",
}
SUCCESS_CRITERIA = INTERPRETATION_RULES

HASH_AMENDMENT_RULE = (
    "replace original_line_sha256 only when line_number identifies an existing source line "
    "and original_line is byte-identical to that line; do not change line_number or text"
)
