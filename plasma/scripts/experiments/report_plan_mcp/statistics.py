"""Preregistered topic-level non-inferiority calculations."""

from __future__ import annotations

import math
import json
from pathlib import Path
import random
from statistics import mean, stdev
from typing import Mapping, Sequence

from .judging import FINAL_DIMENSIONS, PLAN_DIMENSIONS
from .audit import hard_gate


MARGIN = -0.25


def percentile_lower(differences: Sequence[float], seed: int, draws: int = 10_000) -> float:
    if not differences:
        raise ValueError("topic differences are required")
    rng = random.Random(seed)
    means = sorted(sum(rng.choice(differences) for _ in differences) / len(differences) for _ in range(draws))
    return means[math.floor(0.05 * (draws - 1))]


def mode_claim(final_differences: Sequence[float], plan_differences: Sequence[float], seed: int) -> bool:
    if percentile_lower(final_differences, seed) < MARGIN:
        return False
    return percentile_lower(plan_differences, seed + 1) >= MARGIN


def overall_claim(planned: bool, long_form: bool) -> bool:
    return planned and long_form


def reestimated_topics(blinded_sd: float) -> int:
    required = math.ceil(((1.645 + 0.842) * blinded_sd / 0.25) ** 2)
    rounded = max(12, math.ceil(required / 4) * 4)
    if rounded > 24:
        raise ValueError("blinded sample-size re-estimation exceeds feasibility cap")
    return rounded


def exact_sign_pvalue(differences: Sequence[float]) -> float:
    nonzero = [value for value in differences if value != 0]
    if not nonzero:
        return 1.0
    positives = sum(value > 0 for value in nonzero)
    tail = min(positives, len(nonzero) - positives)
    probability = sum(math.comb(len(nonzero), count) for count in range(tail + 1)) / 2 ** len(nonzero)
    return min(1.0, 2 * probability)


def holm_adjust(pvalues: Sequence[float]) -> list[float]:
    order = sorted(range(len(pvalues)), key=pvalues.__getitem__)
    adjusted = [0.0] * len(pvalues)
    running = 0.0
    for rank, index in enumerate(order):
        running = max(running, min(1.0, (len(pvalues) - rank) * pvalues[index]))
        adjusted[index] = running
    return adjusted


def guardrail(mean_difference: float, baseline_low_rate: float, candidate_low_rate: float) -> bool:
    return mean_difference >= -0.50 and candidate_low_rate - baseline_low_rate <= 0.10


def paired_wilcoxon_pvalue(differences: Sequence[float]) -> float:
    values = [(abs(value), value > 0) for value in differences if value != 0]
    if not values:
        return 1.0
    ordered = sorted(enumerate(values), key=lambda item: item[1][0])
    ranks = [0.0] * len(values)
    index = 0
    while index < len(ordered):
        end = index + 1
        while end < len(ordered) and ordered[end][1][0] == ordered[index][1][0]:
            end += 1
        rank = (index + 1 + end) / 2
        for position in range(index, end):
            ranks[ordered[position][0]] = rank
        index = end
    doubled = [int(round(rank * 2)) for rank in ranks]
    observed = sum(rank for rank, (_, positive) in zip(doubled, values, strict=True) if positive)
    counts = {0: 1}
    for rank in doubled:
        updated = dict(counts)
        for total, count in counts.items():
            updated[total + rank] = updated.get(total + rank, 0) + count
        counts = updated
    total_rank = sum(doubled)
    tail = min(observed, total_rank - observed)
    extreme = sum(count for score, count in counts.items() if score <= tail or score >= total_rank - tail)
    return min(1.0, extreme / (2 ** len(doubled)))


def assemble_itt(records: Sequence[Mapping[str, object]]) -> list[dict[str, object]]:
    output: list[dict[str, object]] = []
    required = {"topic", "replicate", "mode", "arm", "started", "terminal_status", "artifact_presence", "machine_metrics"}
    for raw in records:
        if not required.issubset(raw):
            raise ValueError(f"run record missing: {sorted(required - set(raw))}")
        record = dict(raw)
        scores = record.get("scores")
        if record["terminal_status"] != "completed":
            if not record["started"]:
                raise ValueError("pre-run infrastructure failure cannot enter ITT")
            scores = {"plan": {name: 1.0 for name in PLAN_DIMENSIONS}, "final": {name: 1.0 for name in FINAL_DIMENSIONS}}
            record["artifact_presence"] = 0
        _validate_scores(scores)
        record["scores"] = scores
        output.append(record)
    return output


def topic_endpoint_differences(records: Sequence[Mapping[str, object]], mode: str, endpoint: str) -> list[float]:
    dimensions = PLAN_DIMENSIONS if endpoint == "plan" else FINAL_DIMENSIONS
    grouped: dict[tuple[str, str], list[float]] = {}
    for record in records:
        if record["mode"] != mode:
            continue
        score_map = record["scores"][endpoint]  # type: ignore[index]
        composite = mean(float(score_map[name]) for name in dimensions)
        grouped.setdefault((str(record["topic"]), str(record["arm"])), []).append(composite)
    topics = sorted({topic for topic, _ in grouped})
    differences: list[float] = []
    for topic in topics:
        if (topic, "baseline") not in grouped or (topic, "candidate") not in grouped:
            raise ValueError(f"topic pair is incomplete: {topic} {mode}")
        differences.append(mean(grouped[(topic, "candidate")]) - mean(grouped[(topic, "baseline")]))
    return differences


def freeze_sample_size(blinded_endpoint_values: Mapping[str, Sequence[float]], destination: Path) -> dict[str, object]:
    if set(blinded_endpoint_values) != {"planned_final", "planned_plan", "long_form_final", "long_form_plan"}:
        raise ValueError("all four blinded endpoints are required")
    dispersions = {key: stdev(values) for key, values in blinded_endpoint_values.items() if len(values) >= 4}
    if len(dispersions) != 4:
        raise ValueError("four pilot topic values are required per endpoint")
    largest = max(dispersions.values())
    topics = reestimated_topics(largest)
    lock = {"blinded_sd": largest, "endpoint_sds": dispersions, "final_topics": topics, "locked": True}
    destination.parent.mkdir(parents=True, exist_ok=True)
    with destination.open("x", encoding="utf-8") as handle:
        json.dump(lock, handle, indent=2, sort_keys=True)
        handle.write("\n")
    return lock


def analyze_confirmatory(records: Sequence[Mapping[str, object]], seed: int) -> dict[str, object]:
    itt = assemble_itt(records)
    machine_pass = all(hard_gate(record["machine_metrics"], record["arm"] == "candidate") for record in itt)  # type: ignore[arg-type]
    endpoints: dict[str, dict[str, object]] = {}
    sign_values: list[float] = []
    wilcoxon_values: list[float] = []
    claims: dict[str, bool] = {}
    for mode_index, mode in enumerate(("planned", "long_form")):
        final = topic_endpoint_differences(itt, mode, "final")
        plan = topic_endpoint_differences(itt, mode, "plan")
        final_lower = percentile_lower(final, seed + mode_index * 10)
        plan_lower = percentile_lower(plan, seed + mode_index * 10 + 1)
        final_pass = final_lower >= MARGIN
        claims[mode] = final_pass and plan_lower >= MARGIN
        for endpoint, values, lower in (("final", final, final_lower), ("plan", plan, plan_lower)):
            key = f"{mode}_{endpoint}"
            endpoints[key] = {
                "topic_count": len(values), "mean_difference": mean(values), "lower_ci": lower,
                "noninferior": lower >= MARGIN,
                "confirmatory_tested": endpoint == "final" or final_pass,
            }
            sign_values.append(exact_sign_pvalue(values))
            wilcoxon_values.append(paired_wilcoxon_pvalue(values))
    guardrails = {mode: _mode_guardrail(itt, mode) for mode in ("planned", "long_form")}
    return {
        "endpoints": endpoints,
        "mode_claims": claims,
        "machine_gate": machine_pass,
        "guardrails": guardrails,
        "overall_claim": machine_pass and all(guardrails.values()) and overall_claim(claims["planned"], claims["long_form"]),
        "sign_holm": holm_adjust(sign_values),
        "wilcoxon_holm": holm_adjust(wilcoxon_values),
    }


def write_aggregate(result: Mapping[str, object], destination: Path) -> None:
    destination.parent.mkdir(parents=True, exist_ok=True)
    with destination.open("x", encoding="utf-8") as handle:
        json.dump(result, handle, indent=2, sort_keys=True)
        handle.write("\n")


def _mode_guardrail(records: Sequence[Mapping[str, object]], mode: str) -> bool:
    selected = [record for record in records if record["mode"] == mode]
    if not selected:
        raise ValueError(f"no records for {mode}")
    for endpoint, dimension in (("plan", "goal_preservation"), ("final", "completeness")):
        baseline = [float(record["scores"][endpoint][dimension]) for record in selected if record["arm"] == "baseline"]  # type: ignore[index]
        candidate = [float(record["scores"][endpoint][dimension]) for record in selected if record["arm"] == "candidate"]  # type: ignore[index]
        if not baseline or not candidate:
            raise ValueError(f"guardrail pair is incomplete: {mode} {dimension}")
        if not guardrail(mean(candidate) - mean(baseline), _low_rate(baseline), _low_rate(candidate)):
            return False
    return True


def _low_rate(values: Sequence[float]) -> float:
    return sum(value <= 2 for value in values) / len(values)


def _validate_scores(scores: object) -> None:
    if not isinstance(scores, Mapping) or set(scores) != {"plan", "final"}:
        raise ValueError("plan and final score maps are required")
    for endpoint, dimensions in (("plan", PLAN_DIMENSIONS), ("final", FINAL_DIMENSIONS)):
        values = scores[endpoint]
        if not isinstance(values, Mapping) or set(values) != set(dimensions):
            raise ValueError(f"{endpoint} dimensions are incomplete")
        if any(not 1 <= float(value) <= 5 for value in values.values()):
            raise ValueError("judge scores must be in [1,5]")
