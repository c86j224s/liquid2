#!/usr/bin/env python3
"""Ephemeral Codex adapter that rejects observed tool use while scoring blind pairs."""

from __future__ import annotations

import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
from typing import Mapping


DIMENSIONS = (
    "depth", "breadth", "goal_preservation", "investigation_discipline",
    "source_safety", "coverage", "usefulness", "tone", "flow", "consistency",
    "non_repetition", "heading_stability", "completeness",
)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--model", required=True)
    parser.add_argument("--effort", required=True, choices=("high",))
    parser.add_argument("--rubric", type=Path, required=True)
    return parser.parse_args()


def _schema() -> dict[str, object]:
    scores = {
        "type": "object",
        "additionalProperties": False,
        "required": list(DIMENSIONS),
        "properties": {name: {"type": "number", "minimum": 1, "maximum": 5} for name in DIMENSIONS},
    }
    return {
        "type": "object",
        "additionalProperties": False,
        "required": ["A", "B"],
        "properties": {"A": scores, "B": scores},
    }


def schema_sha256() -> str:
    return hashlib.sha256(json.dumps(_schema(), sort_keys=True, separators=(",", ":")).encode()).hexdigest()


def validate_packet(value: object) -> dict[str, object]:
    if not isinstance(value, dict) or set(value) != {"packet_id", "topic", "replicate", "mode", "A", "B"}:
        raise ValueError("blind judge input must be one complete blind packet")
    _reject_private_mapping(value)
    return value


def validate_scores(value: object) -> dict[str, dict[str, float]]:
    if not isinstance(value, dict) or set(value) != {"A", "B"}:
        raise ValueError("judge response must contain exactly A and B")
    result: dict[str, dict[str, float]] = {}
    for label in ("A", "B"):
        scores = value[label]
        if not isinstance(scores, dict) or set(scores) != set(DIMENSIONS):
            raise ValueError("judge response dimensions are incomplete")
        if any(not isinstance(score, (int, float)) or not 1 <= float(score) <= 5 for score in scores.values()):
            raise ValueError("judge scores must be numeric values in [1,5]")
        result[label] = {name: float(scores[name]) for name in DIMENSIONS}
    return result


def _reject_private_mapping(value: object) -> None:
    private_markers = ("baseline", "candidate", "mapping", "commit", "ledger", "manifest", "session", "transport")
    if isinstance(value, Mapping):
        for key, nested in value.items():
            if any(marker in str(key).lower() for marker in private_markers):
                raise ValueError("blind judge input leaks private provenance")
            _reject_private_mapping(nested)
    elif isinstance(value, list):
        for nested in value:
            _reject_private_mapping(nested)


def _command(model: str, effort: str, rubric: str, schema: Path, output: Path) -> list[str]:
    return [
        "codex", "exec", "--ephemeral", "--ignore-user-config", "--ignore-rules",
        "--sandbox", "read-only", "--skip-git-repo-check", "-m", model,
        "-c", f'model_reasoning_effort="{effort}"', "--output-schema", str(schema),
        "--output-last-message", str(output), "--json",
        "Apply this frozen rubric independently to A and B. Return only the requested JSON scores. "
        "Do not use tools, commands, or external sources.\n\n" + rubric,
    ]


def contained_environment(root: Path, inherited: Mapping[str, str]) -> dict[str, str]:
    home, temporary, codex_home = root / "home", root / "tmp", root / "codex"
    for path in (home, temporary):
        path.mkdir(parents=True)
    source = inherited.get("CODEX_HOME")
    if source:
        shutil.copytree(source, codex_home)
    else:
        codex_home.mkdir()
    return {
        "CODEX_HOME": str(codex_home), "HOME": str(home), "TMPDIR": str(temporary),
        "PATH": inherited.get("PATH", os.defpath), "LANG": inherited.get("LANG", "C"),
    }


def _assert_no_tools(events: str) -> None:
    forbidden = ("command_execution", "function_call", "mcp_tool_call", "tool_call", "web_search")
    for line in events.splitlines():
        event = json.loads(line)
        if any(marker in event_type for event_type in _event_types(event) for marker in forbidden):
            raise ValueError("Codex judge attempted to use a tool")


def _event_types(value: object) -> list[str]:
    if isinstance(value, Mapping):
        current = [value["type"]] if isinstance(value.get("type"), str) else []
        return current + [item for nested in value.values() for item in _event_types(nested)]
    if isinstance(value, list):
        return [item for nested in value for item in _event_types(nested)]
    return []


def main() -> int:
    args = parse_args()
    packet = validate_packet(json.load(sys.stdin))
    rubric = args.rubric.read_text(encoding="utf-8")
    with tempfile.TemporaryDirectory(prefix="plasma-blind-judge-") as directory:
        root = Path(directory)
        schema, output = root / "schema.json", root / "response.json"
        schema.write_text(json.dumps(_schema()), encoding="utf-8")
        completed = subprocess.run(
            _command(args.model, args.effort, rubric, schema, output), input=json.dumps(packet), text=True,
            stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=True, cwd=root,
            env=contained_environment(root, dict(os.environ)),
        )
        _assert_no_tools(completed.stdout)
        print(json.dumps(validate_scores(json.loads(output.read_text(encoding="utf-8"))), sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
