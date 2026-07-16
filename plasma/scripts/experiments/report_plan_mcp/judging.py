"""Frozen judge calibration and disagreement aggregation."""

from __future__ import annotations

from collections import Counter
import hashlib
import json
from pathlib import Path
import random
import subprocess
from typing import Mapping
from typing import Sequence


PLAN_DIMENSIONS = ("depth", "breadth", "goal_preservation", "investigation_discipline")
FINAL_DIMENSIONS = ("source_safety", "coverage", "usefulness", "tone", "flow", "consistency", "non_repetition", "heading_stability", "completeness")


def within_one_rate(first: Sequence[int], second: Sequence[int]) -> float:
    return sum(abs(a - b) <= 1 for a, b in zip(first, second, strict=True)) / len(first)


def quadratic_weighted_kappa(first: Sequence[int], second: Sequence[int], levels: int = 5) -> float:
    if len(first) != len(second) or not first:
        raise ValueError("paired non-empty judge scores are required")
    observed = sum((a - b) ** 2 for a, b in zip(first, second)) / (len(first) * (levels - 1) ** 2)
    left, right = Counter(first), Counter(second)
    expected = sum(left[a] * right[b] * (a - b) ** 2 for a in range(1, levels + 1) for b in range(1, levels + 1))
    expected /= len(first) ** 2 * (levels - 1) ** 2
    return 1.0 if expected == 0 and observed == 0 else 1.0 - observed / expected


def calibration_passes(first: Sequence[int], second: Sequence[int]) -> bool:
    return len(first) >= 20 and quadratic_weighted_kappa(first, second) >= 0.70 and within_one_rate(first, second) >= 0.90


def calibrate_dimensions(first: Sequence[dict[str, int]], second: Sequence[dict[str, int]]) -> dict[str, dict[str, float | bool]]:
    if len(first) != len(second) or len(first) < 20:
        raise ValueError("at least 20 paired calibration packets are required")
    dimensions = set(first[0])
    expected = set(PLAN_DIMENSIONS + FINAL_DIMENSIONS)
    if dimensions != expected or any(set(row) != expected for row in (*first, *second)):
        raise ValueError("calibration dimensions are incomplete or inconsistent")
    result: dict[str, dict[str, float | bool]] = {}
    for dimension in sorted(dimensions):
        left = [row[dimension] for row in first]
        right = [row[dimension] for row in second]
        kappa = quadratic_weighted_kappa(left, right)
        within = within_one_rate(left, right)
        result[dimension] = {"weighted_kappa": kappa, "within_one_rate": within, "passed": kappa >= 0.70 and within >= 0.90}
    return result


def needs_third_call(first: Sequence[float], second: Sequence[float], composites: Sequence[tuple[int, ...]]) -> bool:
    if any(abs(a - b) >= 2 for a, b in zip(first, second, strict=True)):
        return True
    return any(abs(sum(first[i] for i in group) / len(group) - sum(second[i] for i in group) / len(group)) > 0.75 for group in composites)


def aggregate(first: Sequence[float], second: Sequence[float], third: Sequence[float] | None = None) -> list[float]:
    if third is None:
        return [(a + b) / 2 for a, b in zip(first, second, strict=True)]
    return [sorted(values)[1] for values in zip(first, second, third, strict=True)]


def aggregate_packet_scores(
    first: dict[str, float], second: dict[str, float], third: dict[str, float] | None = None,
) -> dict[str, float]:
    dimensions = PLAN_DIMENSIONS + FINAL_DIMENSIONS
    if set(first) != set(dimensions) or set(second) != set(dimensions) or (third is not None and set(third) != set(dimensions)):
        raise ValueError("judge packet dimensions are incomplete")
    left, right = [first[name] for name in dimensions], [second[name] for name in dimensions]
    composites = (tuple(range(len(PLAN_DIMENSIONS))), tuple(range(len(PLAN_DIMENSIONS), len(dimensions))))
    disagreement = needs_third_call(left, right, composites)
    if disagreement != (third is not None):
        raise ValueError("third judge call presence does not match the frozen disagreement rule")
    values = aggregate(left, right, None if third is None else [third[name] for name in dimensions])
    if any(not 1 <= value <= 5 for value in values):
        raise ValueError("judge scores must be in [1,5]")
    return dict(zip(dimensions, values, strict=True))


def build_blind_packets(pairs: Sequence[dict[str, object]], destination: Path, seed: int) -> list[dict[str, object]]:
    destination.mkdir(parents=True, exist_ok=False)
    packets: list[dict[str, object]] = []
    mappings: list[dict[str, str]] = []
    for pair in pairs:
        for key in ("topic", "replicate", "mode", "baseline", "candidate"):
            if key not in pair:
                raise ValueError(f"blind pair missing {key}")
        identity = f"{pair['topic']}:{pair['replicate']}:{pair['mode']}"
        local = random.Random(f"{seed}:{identity}")
        order = ["baseline", "candidate"]
        local.shuffle(order)
        packet_id = hashlib.sha256(identity.encode()).hexdigest()[:16]
        packet = {"packet_id": packet_id, "topic": pair["topic"], "replicate": pair["replicate"], "mode": pair["mode"], "A": pair[order[0]], "B": pair[order[1]]}
        _assert_blind_keys(packet)
        _write_new(destination / f"{packet_id}.json", packet)
        packets.append(packet)
        mappings.append({
            "packet_id": packet_id, "topic": str(pair["topic"]), "replicate": int(pair["replicate"]),
            "mode": str(pair["mode"]), "A": order[0], "B": order[1],
        })
    _write_new(destination.parent / f"{destination.name}-mapping.private.json", mappings)
    return packets


def _assert_blind_keys(value: object) -> None:
    forbidden = ("baseline", "candidate", "session", "commit", "transport", "ledger", "manifest", "sha256", "path")
    if isinstance(value, dict):
        for key, nested in value.items():
            if any(marker in str(key).lower() for marker in forbidden):
                raise ValueError(f"blind packet leaks provenance key {key}")
            _assert_blind_keys(nested)
    elif isinstance(value, list):
        for nested in value:
            _assert_blind_keys(nested)


def invoke_judge(command: Sequence[str], packet: Mapping[str, object], environment: Mapping[str, str]) -> dict[str, float]:
    completed = subprocess.run(command, input=json.dumps(packet), text=True, capture_output=True, env=environment, check=True)
    value = json.loads(completed.stdout)
    if not isinstance(value, dict) or any(not isinstance(score, (int, float)) for score in value.values()):
        raise ValueError("judge response must be a numeric score object")
    return {str(name): float(score) for name, score in value.items()}


def invoke_pair_judge(command: Sequence[str], packet: Mapping[str, object], environment: Mapping[str, str]) -> dict[str, dict[str, float]]:
    completed = subprocess.run(command, input=json.dumps(packet), text=True, capture_output=True, env=environment, check=True)
    value = json.loads(completed.stdout)
    if not isinstance(value, dict) or set(value) != {"A", "B"}:
        raise ValueError("experimental judge response must contain exactly A and B")
    result: dict[str, dict[str, float]] = {}
    for arm in ("A", "B"):
        scores = value[arm]
        if not isinstance(scores, dict) or set(scores) != set(PLAN_DIMENSIONS + FINAL_DIMENSIONS):
            raise ValueError("experimental judge dimensions are incomplete")
        if any(not isinstance(score, (int, float)) for score in scores.values()):
            raise ValueError("experimental judge scores must be numeric")
        result[arm] = {str(name): float(score) for name, score in scores.items()}
    return result


def score_packets(
    command: Sequence[str], packet_paths: Sequence[Path], destination: Path, environment: Mapping[str, str],
) -> list[dict[str, object]]:
    destination.mkdir(parents=True, exist_ok=False)
    results: list[dict[str, object]] = []
    for path in packet_paths:
        packet = json.loads(path.read_text(encoding="utf-8"))
        first = invoke_pair_judge(command, packet, environment)
        second = invoke_pair_judge(command, packet, environment)
        third: dict[str, dict[str, float]] | None = None
        arm_needs_third: dict[str, bool] = {}
        for label in ("A", "B"):
            try:
                aggregate_packet_scores(first[label], second[label])
                arm_needs_third[label] = False
            except ValueError as exc:
                if "third judge call presence" not in str(exc):
                    raise
                arm_needs_third[label] = True
        if any(arm_needs_third.values()):
            third = invoke_pair_judge(command, packet, environment)
        scores = {
            label: aggregate_packet_scores(first[label], second[label], third[label] if third is not None and arm_needs_third[label] else None)
            for label in ("A", "B")
        }
        result = {"packet_id": packet.get("packet_id"), "technical_calls": 3 if third else 2, "scores": scores}
        _write_new(destination / f"{packet['packet_id']}.json", result)
        results.append(result)
    return results


def calibrate_with_command(
    command: Sequence[str], packet_paths: Sequence[Path], destination: Path, environment: Mapping[str, str],
) -> dict[str, dict[str, float | bool]]:
    if len(packet_paths) < 20:
        raise ValueError("at least 20 calibration packets are required")
    first, second = [], []
    for path in packet_paths:
        packet = json.loads(path.read_text(encoding="utf-8"))
        left, right = invoke_judge(command, packet, environment), invoke_judge(command, packet, environment)
        if any(not value.is_integer() or not 1 <= value <= 5 for value in (*left.values(), *right.values())):
            raise ValueError("calibration scores must be ordinal integers in [1,5]")
        first.append({name: int(value) for name, value in left.items()})
        second.append({name: int(value) for name, value in right.items()})
    result = calibrate_dimensions(first, second)
    _write_new(destination, result)
    return result


def _write_new(path: Path, value: object) -> None:
    with path.open("x", encoding="utf-8") as handle:
        json.dump(value, handle, ensure_ascii=False, indent=2, sort_keys=True)
        handle.write("\n")
