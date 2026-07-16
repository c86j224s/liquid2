"""Product `plasma mcp` stdio fault driver."""

from __future__ import annotations

import json
from pathlib import Path
import subprocess
from typing import Mapping

from .fault_seed import render_string, render_value, seed_public_state
from .fault_seed import materialize_isolation_environment
from .safety import isolated_environment, validate_environment

FAULT_CASES = (
    "tool_unbound",
    "binding_mismatch",
    "unknown_report_mode",
    "malformed_plan",
    "missing_ref",
    "wrong_kind_ref",
    "cross_mission_ref",
    "ineligible_ref",
    "validation_attempts_exhausted",
    "protocol_parse_no_attempt",
    "idempotent_replay",
    "idempotency_conflict",
    "conditional_store_unavailable",
)

STATEFUL_CASES = frozenset(FAULT_CASES) - {
    "tool_unbound", "unknown_report_mode", "malformed_plan", "protocol_parse_no_attempt",
}


def run_stdio(binary: Path, arguments: list[str], messages: list[object], environment: Mapping[str, str]) -> list[dict[str, object]]:
    payload = "".join(json.dumps(message, separators=(",", ":")) + "\n" for message in messages)
    completed = subprocess.run([str(binary), "mcp", *arguments], input=payload, text=True, capture_output=True, env=environment)
    if completed.returncode != 0:
        raise RuntimeError(f"plasma mcp exited {completed.returncode}: {completed.stderr.strip()}")
    lines = [json.loads(line) for line in completed.stdout.splitlines() if line.strip()]
    if len(lines) != len(messages):
        raise RuntimeError("plasma mcp response count mismatch")
    return lines


def run_fault_matrix(binary: Path, cases: Mapping[str, Mapping[str, object]], environment: Mapping[str, str], fault_root: Path) -> dict[str, object]:
    required = set(FAULT_CASES)
    if not required.issubset(cases):
        raise ValueError(f"fault cases missing: {sorted(required - set(cases))}")
    results: dict[str, object] = {}
    used_ports: set[int] = set()
    for name in sorted(required):
        case = cases[name]
        case_root = (fault_root / name).resolve()
        case_environment = fault_case_environment(case_root, environment)
        if name in STATEFUL_CASES:
            commands = case.get("seed_commands")
            if not isinstance(commands, list) or not commands:
                raise ValueError(f"stateful fault case {name} requires public product seed_commands")
            evidence, bindings = seed_public_state(binary, case_root, commands, case, case_environment, used_ports)
        else:
            case_root.mkdir(parents=True, exist_ok=False)
            materialize_isolation_environment(case_environment)
            evidence = []
            bindings = {"case_root": str(case_root), "binary": str(binary)}
        arguments = case.get("binding_args")
        messages = case.get("messages")
        expected = case.get("expected_fragments")
        forbidden = case.get("forbidden_fragments", [])
        if not isinstance(arguments, list) or not isinstance(messages, list) or not messages:
            raise ValueError(f"fault case {name} has no executable subprocess contract")
        rendered_arguments = [render_string(str(value), bindings) for value in arguments]
        messages = render_value(messages, bindings)
        if name in STATEFUL_CASES and "-db" not in rendered_arguments:
            raise ValueError(f"stateful fault case {name} must bind an isolated database")
        if "-db" in rendered_arguments:
            database = Path(rendered_arguments[rendered_arguments.index("-db") + 1]).resolve()
            if not database.is_relative_to(case_root):
                raise ValueError(f"fault case {name} database escapes its isolated root")
        if not isinstance(expected, list) or not expected or not isinstance(forbidden, list):
            raise ValueError(f"fault case {name} has no fail-closed response assertions")
        if name == "conditional_store_unavailable":
            payload = "".join(json.dumps(message, separators=(",", ":")) + "\n" for message in messages)
            completed = subprocess.run([str(binary), "mcp", *rendered_arguments], input=payload, text=True, capture_output=True, env=case_environment)
            if completed.returncode == 0:
                raise RuntimeError("conditional store failure case unexpectedly succeeded")
            observed = completed.stderr
            _assert_fragments(name, observed, expected, forbidden)
            results[name] = {"returncode": completed.returncode, "failed_closed": True, "seed_evidence": evidence}
        else:
            responses = run_stdio(binary, rendered_arguments, messages, case_environment)
            _assert_fragments(name, json.dumps(responses, sort_keys=True), expected, forbidden)
            results[name] = {"responses": responses, "failed_closed": True, "seed_evidence": evidence}
    return results


def fault_case_environment(case_root: Path, inherited: Mapping[str, str]) -> dict[str, str]:
    overrides = isolated_environment(case_root, inherited)
    validate_environment(overrides, case_root, inherited)
    environment = dict(inherited)
    for key, value in overrides.items():
        if value is None:
            environment.pop(key, None)
        else:
            environment[key] = value
    validate_environment(environment, case_root, inherited)
    return environment


def _assert_fragments(name: str, observed: str, expected: list[object], forbidden: list[object]) -> None:
    if any(str(value) not in observed for value in expected) or any(str(value) in observed for value in forbidden):
        raise RuntimeError(f"fault case {name} did not satisfy its frozen response assertions")
