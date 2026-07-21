#!/usr/bin/env python3
"""Issue #160 visual-type selection experiment runner."""

from __future__ import annotations

import argparse
from concurrent.futures import ThreadPoolExecutor, as_completed
from dataclasses import dataclass
from datetime import datetime, timezone
import hashlib
import json
import random
import re
import shutil
import subprocess
from pathlib import Path
from threading import Lock
from typing import Any

import report_fanout_experiment as base


EXPERIMENT_ID = "25-report-visual-type-selection-2026-07-21"
ARMS = ("visual_plan", "visual_type_manual")
PROFILE_BY_ARM = {
    "visual_plan": "visual-plan",
    "visual_type_manual": "visual-type-manual",
}
MODES = ("planned", "long_form")


@dataclass(frozen=True)
class Fixture:
    topic: str
    title: str
    objective: str
    source_bundle: Path
    source_sha256: str
    expected_visual_families: tuple[str, ...]


@dataclass(frozen=True)
class FixtureSpec:
    topic: str
    title: str
    objective: str
    expected_visual_families: tuple[str, ...]
    body: str


FIXTURE_SPECS = (
    FixtureSpec(
        topic="fictional-equity-dashboard",
        title="Fictional equity dashboard visual-structure packet",
        objective="Explain what changed in the fictional equity dataset and choose visuals only when they clarify the source-backed structure.",
        expected_visual_families=("table", "source_backed_chart", "timeline"),
        body="""# Source packet: fictional equity dashboard

This is a synthetic fixture for report-generation experiments. It is not market
data and must not be treated as investment advice.

## Daily observations

| Day | Open | High | Low | Close | Volume | 5-day average close | Note |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| D1 | 101.2 | 104.5 | 100.9 | 103.8 | 1.2M | 101.9 | Baseline volume |
| D2 | 103.6 | 105.1 | 102.2 | 104.6 | 1.1M | 102.7 | Supplier briefing |
| D3 | 104.5 | 107.9 | 104.1 | 107.2 | 1.8M | 103.9 | Guidance rumor |
| D4 | 107.0 | 108.4 | 105.8 | 106.1 | 1.5M | 104.6 | Profit-taking |
| D5 | 106.4 | 109.6 | 105.9 | 109.0 | 2.1M | 106.1 | Contract disclosed |
| D6 | 109.2 | 111.5 | 108.7 | 110.8 | 2.6M | 107.5 | Analyst upgrades |
| D7 | 110.5 | 112.0 | 107.4 | 108.2 | 2.4M | 108.3 | Sector rotation |
| D8 | 108.0 | 109.1 | 104.8 | 105.4 | 2.9M | 107.9 | Margin warning |
| D9 | 105.6 | 107.2 | 103.9 | 106.7 | 2.2M | 108.0 | Stabilization |
| D10 | 106.8 | 110.4 | 106.3 | 109.8 | 2.7M | 108.2 | Customer order update |

## Interpretation constraints

- The packet supports comparing close, volume, and event markers.
- It does not support causal claims about future price movement.
- A compact table or simple source-backed chart can help if it stays close to
  the listed values.
- A diagram that invents investor psychology or hidden demand would be wrong.
""",
    ),
    FixtureSpec(
        topic="industry-capacity-statistics",
        title="Industry capacity and bottleneck statistics packet",
        objective="Summarize the industry capacity picture and choose visual forms that clarify comparisons, bottlenecks, and timing.",
        expected_visual_families=("table", "source_backed_chart", "timeline"),
        body="""# Source packet: industry capacity and bottlenecks

This packet is synthetic and uses fictional regional names.

## Capacity table

| Region | 2025 capacity | 2026 planned capacity | Utilization | Lead time | Main bottleneck |
| --- | ---: | ---: | ---: | ---: | --- |
| North Fabrication Belt | 48 units | 55 units | 92% | 18 weeks | Lithography tools |
| East Assembly Corridor | 61 units | 70 units | 88% | 11 weeks | Skilled technicians |
| South Materials Hub | 37 units | 42 units | 95% | 22 weeks | Specialty substrates |
| West Packaging Cluster | 44 units | 53 units | 81% | 9 weeks | Power expansion permits |

## Timeline

- 2025 Q4: two regions report utilization above 90%.
- 2026 Q1: tool shipment delays extend North lead time from 14 to 18 weeks.
- 2026 Q2: South Materials Hub prioritizes existing customers over spot demand.
- 2026 Q3: West Packaging Cluster expects permit decision.

## Reading notes

- The useful structure is comparison across regions and a short bottleneck
  timeline.
- The packet does not rank company-level winners.
- The lead-time numbers and utilization percentages are source-backed.
""",
    ),
    FixtureSpec(
        topic="agent-benchmark-matrix",
        title="Agent benchmark matrix and failure-mode packet",
        objective="Explain benchmark trade-offs across agents without flattening latency, accuracy, tool reliability, and cost into one score.",
        expected_visual_families=("table", "source_backed_chart", "quadrant"),
        body="""# Source packet: agent benchmark matrix

The agents and scores are fictional. This fixture tests dense benchmark
reporting and visual selection.

## Benchmark results

| Agent | Planning tasks pass@1 | Tool tasks pass@1 | Median latency | Tool-call error rate | Mean context tokens | Cost index |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Atlas-S | 74% | 68% | 41s | 4.2% | 92k | 1.00 |
| Boreal-M | 81% | 72% | 58s | 3.1% | 118k | 1.34 |
| Cygnus-L | 86% | 77% | 96s | 5.7% | 171k | 2.40 |
| Delta-R | 79% | 81% | 64s | 2.2% | 130k | 1.62 |
| Echo-F | 70% | 66% | 32s | 6.4% | 76k | 0.82 |

## Failure-mode counts over 200 runs

| Failure mode | Atlas-S | Boreal-M | Cygnus-L | Delta-R | Echo-F |
| --- | ---: | ---: | ---: | ---: | ---: |
| Wrong tool arguments | 11 | 8 | 15 | 5 | 17 |
| Scope drift | 14 | 10 | 9 | 12 | 18 |
| Premature completion | 9 | 7 | 5 | 8 | 13 |
| Overlong context | 3 | 6 | 14 | 7 | 2 |

## Interpretation constraints

- There is no single winner unless the report names the chosen priority.
- A table is necessary for exact figures.
- A quadrant-style framing may help compare reliability and speed, but it must
  not invent exact coordinates beyond the listed metrics.
""",
    ),
    FixtureSpec(
        topic="architecture-dependency-graph",
        title="Complex architecture dependency graph packet",
        objective="Explain architecture dependencies, critical paths, and blast-radius risk using visuals only where they make the graph easier to understand.",
        expected_visual_families=("dependency_graph", "table"),
        body="""# Source packet: complex architecture dependency graph

This synthetic architecture packet describes a research workflow service.

## Components

- Browser UI sends user actions to Gateway.
- Gateway routes mission commands to Mission API and report commands to Report API.
- Mission API owns mission metadata in Primary SQL.
- Source API owns source records and snapshots in Primary SQL and Blob Store.
- Report API schedules report work through Report Runner.
- Report Runner starts Planner Agent, Section Agent Pool, Assembly Agent, and Finalizer Agent.
- Planner Agent and Section Agent Pool read approved source snapshots through MCP Source Tools.
- MCP Source Tools read Primary SQL metadata and Blob Store content through read-only ports.
- Event Ledger records user-visible lifecycle events.
- Activity Projection reads Event Ledger and serves lightweight UI polling.
- Search Index is rebuilt asynchronously from approved source snapshots.
- Notification Worker consumes Report Events from Event Bus.

## Dependency table

| From | To | Mode | Criticality | Failure impact |
| --- | --- | --- | --- | --- |
| Browser UI | Gateway | sync HTTP | high | User cannot start or observe work |
| Gateway | Mission API | sync HTTP | high | Mission creation and edits fail |
| Gateway | Report API | sync HTTP | high | Report requests cannot start |
| Mission API | Primary SQL | sync SQL | high | Durable mission state unavailable |
| Source API | Blob Store | sync object I/O | high | Source snapshots cannot be read or written |
| Report API | Report Runner | async job handoff | high | Requests are accepted but never progress |
| Report Runner | Planner Agent | subprocess/MCP | high | No plan is produced |
| Report Runner | Section Agent Pool | subprocess/MCP | medium | Long-form reports slow or fail by section |
| Section Agent Pool | MCP Source Tools | MCP | high | Sections cannot inspect approved sources |
| MCP Source Tools | Blob Store | read-only object I/O | high | Source content is unavailable to agents |
| Report Runner | Event Ledger | append event | high | UI cannot explain report progress |
| Event Ledger | Activity Projection | async read model | medium | Polling becomes stale but durable events remain |
| Source API | Search Index | async rebuild | low | Search freshness degrades |
| Report Runner | Event Bus | async publish | medium | Notifications fail but report can finish |

## Notes

- Report generation depends on read-only source access and append-only event
  recording.
- Search Index is not on the critical path for already attached sources.
- Activity Projection is a derived view, not the source of truth.
- A stable flowchart with grouped subsystems may explain this better than prose.
- A C4-style diagram would be tempting, but a Mermaid flowchart fallback is safer
  if compatibility is uncertain.
""",
    ),
    FixtureSpec(
        topic="protocol-lifecycle",
        title="Protocol and lifecycle packet",
        objective="Describe a staged source-review protocol and decide whether sequence or state visuals help more than prose.",
        expected_visual_families=("sequence", "state", "timeline"),
        body="""# Source packet: staged source-review protocol

## Actors

- User
- Conversation Agent
- Source Candidate Service
- Snapshot Worker
- Review UI
- Mission Ledger

## Happy path

1. User asks the Conversation Agent to look for source material.
2. Conversation Agent proposes a source candidate with title, URL, and reason.
3. Source Candidate Service records the candidate as proposed.
4. Snapshot Worker starts a best-effort fetch and records fetching.
5. Snapshot Worker records fetched or failed.
6. User opens Review UI and accepts or rejects the candidate.
7. Accepted candidates become mission sources.
8. Mission Ledger records the decision and snapshot outcome.

## State model

Candidate states are proposed, fetching, fetched, fetch_failed, accepted,
rejected, and withdrawn. Only proposed, fetched, and fetch_failed can be accepted
or rejected. Accepted and rejected are terminal.

## Constraints

- The agent may read fetched candidate material during conversation, but it must
  still tell the user that the source is not approved yet.
- The report writer can cite only approved mission sources.
""",
    ),
    FixtureSpec(
        topic="scenario-risk-portfolio",
        title="Scenario risk portfolio packet",
        objective="Explain scenario risk and uncertainty while choosing visuals that clarify scenario structure without implying unsupported precision.",
        expected_visual_families=("table", "source_backed_chart", "timeline"),
        body="""# Source packet: scenario risk portfolio

The figures are synthetic. The packet is designed to test visual restraint.

## Scenario table

| Scenario | Probability band | Revenue pressure | Margin pressure | Operational risk | Time horizon |
| --- | --- | ---: | ---: | ---: | --- |
| Base continuity | 45-55% | -1% to +2% | 0% to -1% | Low | 2 quarters |
| Demand air pocket | 20-30% | -6% to -9% | -2% to -4% | Medium | 1 quarter |
| Supply disruption | 10-15% | -3% to -5% | -5% to -8% | High | 2-3 quarters |
| Policy acceleration | 10-20% | +4% to +7% | -1% to +1% | Medium | 3 quarters |

## Cross-scenario notes

- Supply disruption has lower probability but highest operational risk.
- Demand air pocket is the fastest downside case.
- Policy acceleration has upside revenue but execution uncertainty.
- Probability bands are broad and should not be converted into exact expected
  values.

## Timeline anchors

- Month 1: demand data releases.
- Month 2: supplier allocation decision.
- Month 3: policy funding review.
- Month 6: margin impact becomes visible.
""",
    ),
)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--action", choices=("prepare", "run", "analyze", "packets"), required=True)
    parser.add_argument("--workers", type=int, default=2)
    parser.add_argument("--limit", type=int, default=len(FIXTURE_SPECS))
    parser.add_argument("--modes", nargs="+", choices=MODES, default=list(MODES))
    parser.add_argument("--topics", nargs="+", default=None)
    parser.add_argument("--arms", nargs="+", choices=ARMS, default=list(ARMS))
    parser.add_argument("--model", default="gpt-5.5")
    parser.add_argument("--effort", default="medium")
    parser.add_argument("--long-form-strategy", choices=("serial", "section_fanout"), default="section_fanout")
    parser.add_argument("--timeout-seconds", type=int, default=7200)
    parser.add_argument("--archive", type=Path, default=default_archive())
    parser.add_argument("--seed", type=int, default=16025)
    return parser.parse_args()


def default_archive() -> Path:
    return Path.home() / "research-artifacts/liquid2/plasma/experiments" / EXPERIMENT_ID


def prepare(args: argparse.Namespace) -> None:
    archive = args.archive.expanduser().resolve()
    archive.mkdir(parents=True, exist_ok=True)
    (archive / "bin").mkdir(exist_ok=True)
    fixtures = write_synthetic_fixtures(archive)
    base.write_json_new_or_same(
        archive / "fixtures.lock.json",
        {"fixtures": [fixture_to_json(fixture) for fixture in fixtures]},
    )
    binary = archive / "bin" / "plasma"
    subprocess.run(["go", "build", "-o", str(binary), "./cmd/plasma"], cwd=base.plasma_root(), check=True)
    base.write_json(
        archive / "control.prepare.json",
        {
            "experiment": EXPERIMENT_ID,
            "repo": str(base.repo_root()),
            "git_head": base.git("rev-parse", "HEAD"),
            "git_dirty": bool(base.git("status", "--porcelain")),
            "binary": str(binary),
            "binary_sha256": base.sha256(binary),
            "arms": list(ARMS),
            "profiles": PROFILE_BY_ARM,
            "modes_default": args.modes,
            "model_default": args.model,
            "effort_default": args.effort,
            "long_form_strategy_default": args.long_form_strategy,
            "prepared_at": utc_now(),
        },
    )
    print(json.dumps({"archive": str(archive), "fixtures": len(fixtures), "binary": str(binary)}, ensure_ascii=False))


def write_synthetic_fixtures(archive: Path) -> list[Fixture]:
    fixtures: list[Fixture] = []
    for spec in FIXTURE_SPECS:
        path = archive / "fixtures" / f"{spec.topic}.md"
        write_text_new_or_same(path, spec.body.strip() + "\n")
        digest = base.sha256(path)
        fixtures.append(
            Fixture(
                spec.topic,
                spec.title,
                spec.objective,
                path,
                digest,
                spec.expected_visual_families,
            )
        )
    return fixtures


def load_fixtures(archive: Path, limit: int) -> list[Fixture]:
    manifest = json.loads((archive / "fixtures.lock.json").read_text(encoding="utf-8"))
    fixtures = []
    for row in manifest["fixtures"][:limit]:
        path = Path(row["source_bundle"]).expanduser().resolve()
        digest = str(row["source_sha256"])
        if base.sha256(path) != digest:
            raise ValueError(f"fixture hash mismatch: {path}")
        fixtures.append(
            Fixture(
                str(row["topic"]),
                str(row["title"]),
                str(row["objective"]),
                path,
                digest,
                tuple(str(item) for item in row.get("expected_visual_families", [])),
            )
        )
    return fixtures


def run(args: argparse.Namespace) -> None:
    archive = args.archive.expanduser().resolve()
    if not (archive / "bin/plasma").is_file():
        prepare(args)
    fixtures = select_fixtures(archive, args.limit, args.topics)
    specs = [(fixture, mode, arm) for fixture in fixtures for mode in args.modes for arm in args.arms]
    random.Random(args.seed).shuffle(specs)
    used_ports: set[int] = set()
    port_lock = Lock()
    results: list[dict[str, Any]] = []
    with ThreadPoolExecutor(max_workers=max(1, args.workers)) as pool:
        futures = [
            pool.submit(
                run_one,
                archive,
                fixture,
                mode,
                arm,
                args.model,
                args.effort,
                args.long_form_strategy,
                args.timeout_seconds,
                used_ports,
                port_lock,
            )
            for fixture, mode, arm in specs
        ]
        for future in as_completed(futures):
            result = future.result()
            results.append(result)
            print(json.dumps({"topic": result["topic"], "mode": result["mode"], "arm": result["arm"], "status": result["status"]}, ensure_ascii=False), flush=True)
    base.write_json(archive / "run-summary.json", {"completed_at": utc_now(), "results": results})


def select_fixtures(archive: Path, limit: int, topics: list[str] | None) -> list[Fixture]:
    fixtures = load_fixtures(archive, len(FIXTURE_SPECS) if topics else limit)
    if not topics:
        return fixtures
    wanted = set(topics)
    selected = [fixture for fixture in fixtures if fixture.topic in wanted]
    missing = sorted(wanted - {fixture.topic for fixture in selected})
    if missing:
        raise ValueError(f"unknown fixture topics: {', '.join(missing)}")
    return selected


def run_one(
    archive: Path,
    fixture: Fixture,
    mode: str,
    arm: str,
    model: str,
    effort: str,
    long_form_strategy: str,
    timeout_seconds: int,
    used_ports: set[int],
    port_lock: Lock,
) -> dict[str, Any]:
    run_root = archive / "runs" / f"{fixture.topic}-{mode}-{arm}"
    terminal = run_root / "manifest.terminal.json"
    if terminal.exists():
        return json.loads(terminal.read_text(encoding="utf-8"))
    if run_root.exists():
        shutil.rmtree(run_root)
    run_root.mkdir(parents=True, exist_ok=False)
    for path in ("state", "artifacts", "logs", "workdir", "fixture"):
        (run_root / path).mkdir()
    source = run_root / "fixture" / fixture.source_bundle.name
    shutil.copy2(fixture.source_bundle, source)
    binary = archive / "bin" / "plasma"
    with port_lock:
        port = base.allocate_port(used_ports)
        connector_port = base.allocate_port(used_ports)
    env = base.isolated_environment(run_root)
    connector_log = (run_root / "logs/liquid2-stub.log").open("xb")
    serve_log = (run_root / "logs/serve.log").open("xb")
    connector = process = None
    manifest = {
        "experiment": EXPERIMENT_ID,
        "topic": fixture.topic,
        "mode": mode,
        "arm": arm,
        "model": model,
        "effort": effort,
        "generation_guidance_profile": PROFILE_BY_ARM[arm],
        "long_form_strategy": long_form_strategy if mode == "long_form" else "",
        "database": str(run_root / "state/plasma.db"),
        "run_root": str(run_root),
        "port": port,
        "connector_port": connector_port,
        "binary": str(binary),
        "binary_sha256": base.sha256(binary),
        "status": "started",
        "started_at": utc_now(),
        "expected_visual_families": list(fixture.expected_visual_families),
    }
    base.write_json(run_root / "manifest.initial.json", manifest)
    try:
        connector = base.start_connector_stub(connector_port, env, connector_log)
        base.wait_health(f"http://127.0.0.1:{connector_port}", connector, 30)
        process = subprocess.Popen(
            [
                str(binary),
                "serve",
                "-db",
                manifest["database"],
                "-addr",
                f"127.0.0.1:{port}",
                "-liquid2-url",
                f"http://127.0.0.1:{connector_port}",
                "-local-source-root",
                f"fixture={source.parent}",
                "-agent",
                "codex",
                "-agent-workdir",
                str(run_root / "workdir"),
                "-agent-timeout",
                "0",
            ],
            env=env,
            stdout=serve_log,
            stderr=subprocess.STDOUT,
        )
        base.wait_health(f"http://127.0.0.1:{port}", process, 30)
        mission = base.run_json(
            [
                str(binary),
                "missions",
                "create",
                "-db",
                manifest["database"],
                "-title",
                fixture.title,
                "-objective",
                fixture.objective,
                "-json",
            ],
            env,
        )
        mission_id = base.find_string(mission, "MissionID", "mission_id")
        base.run_json(
            [
                str(binary),
                "sources",
                "attach-local",
                mission_id,
                "-db",
                manifest["database"],
                "-root",
                "fixture",
                "-path",
                source.name,
                "-title",
                fixture.title,
                "-local-source-root",
                f"fixture={source.parent}",
                "-json",
            ],
            env,
        )
        body = {
            "title": fixture.title,
            "report_mode": mode,
            "agent_executor": "codex",
            "agent_model": model,
            "agent_reasoning_effort": effort,
            "generation_guidance_profile": PROFILE_BY_ARM[arm],
            "post_report_humanize": "disabled",
            "report_session_policy": "same_session",
        }
        if mode == "long_form":
            body["execution_strategy"] = long_form_strategy
        base.http_json(f"http://127.0.0.1:{port}/api/missions/{mission_id}/reports", body)
        events, status = base.poll_terminal(f"http://127.0.0.1:{port}", mission_id, process, timeout_seconds)
        base.write_json(run_root / "ledger.events.json", {"events": events})
        write_plan(run_root, events)
        manifest |= {"mission_id": mission_id, "status": status, "completed_at": utc_now()}
        if status == "completed":
            artifact_id = base.final_artifact_id(events)
            report = base.http_bytes(f"http://127.0.0.1:{port}/api/missions/{mission_id}/artifacts/{artifact_id}/download")
            (run_root / "report.md").write_bytes(report)
            manifest |= {"artifact_id": artifact_id, "report_sha256": hashlib.sha256(report).hexdigest()}
        metrics = base.collect_metrics(events, run_root / "report.md") | collect_visual_metrics(fixture, events, run_root / "report.md")
        base.write_json(run_root / "metrics.json", metrics)
        base.write_json(run_root / "manifest.terminal.json", manifest)
        return manifest
    except Exception as exc:
        manifest |= {"status": "failed", "error": str(exc), "completed_at": utc_now()}
        base.write_json(run_root / "manifest.terminal.json", manifest)
        return manifest
    finally:
        if process is not None:
            base.stop_process(process)
        if connector is not None:
            base.stop_process(connector)
        with port_lock:
            used_ports.discard(port)
            used_ports.discard(connector_port)
        serve_log.close()
        connector_log.close()


def write_plan(run_root: Path, events: list[dict[str, Any]]) -> None:
    event = base.first_event(events, "report.plan.created")
    payload = event.get("Payload", {}) if isinstance(event, dict) else {}
    plan = payload.get("plan") if isinstance(payload, dict) else None
    if isinstance(plan, dict):
        base.write_json(run_root / "plan.json", plan)


def collect_visual_metrics(fixture: Fixture, events: list[dict[str, Any]], report_path: Path) -> dict[str, Any]:
    report = report_path.read_text(encoding="utf-8") if report_path.exists() else ""
    mermaid_types = detect_mermaid_types(report)
    table_headers = len(re.findall(r"(?m)^\s*\|[^\n]+\|\s*\n\s*\|\s*:?-{3,}:?\s*\|", report))
    event_blob = json.dumps(events, ensure_ascii=False)
    validation_calls = event_blob.count("plasma.mermaid.validate")
    alignment_hits, alignment_missing = visual_alignment(fixture.expected_visual_families, table_headers, mermaid_types)
    words = max(1, len(report.split()))
    return {
        "table_count": table_headers,
        "mermaid_fence_count": sum(mermaid_types.values()),
        "mermaid_type_counts": mermaid_types,
        "visual_aid_count": table_headers + sum(mermaid_types.values()),
        "visual_aids_per_1000_words": round(((table_headers + sum(mermaid_types.values())) / words) * 1000, 3),
        "mermaid_validate_mentions": validation_calls,
        "has_unvalidated_mermaid_signal": sum(mermaid_types.values()) > validation_calls,
        "expected_visual_families": list(fixture.expected_visual_families),
        "visual_alignment_hits": alignment_hits,
        "visual_alignment_missing": alignment_missing,
        "visual_alignment_score": len(alignment_hits),
    }


def detect_mermaid_types(report: str) -> dict[str, int]:
    counts: dict[str, int] = {}
    for match in re.finditer(r"(?ims)^```mermaid\s*\n\s*([A-Za-z0-9_-]+)", report):
        diagram_type = match.group(1)
        counts[diagram_type] = counts.get(diagram_type, 0) + 1
    return counts


def visual_alignment(expected: tuple[str, ...], table_count: int, mermaid_types: dict[str, int]) -> tuple[list[str], list[str]]:
    types = {key for key, count in mermaid_types.items() if count > 0}
    hits: list[str] = []
    missing: list[str] = []
    for family in expected:
        ok = False
        if family == "table":
            ok = table_count > 0
        elif family == "source_backed_chart":
            ok = bool(types & {"pie", "quadrantChart", "xychart-beta"})
        elif family == "quadrant":
            ok = "quadrantChart" in types
        elif family == "dependency_graph":
            ok = bool(types & {"flowchart", "graph", "classDiagram", "erDiagram", "architecture", "C4Context"})
        elif family == "sequence":
            ok = "sequenceDiagram" in types
        elif family == "state":
            ok = "stateDiagram-v2" in types
        elif family == "timeline":
            ok = "timeline" in types
        if ok:
            hits.append(family)
        else:
            missing.append(family)
    return hits, missing


def analyze(args: argparse.Namespace) -> None:
    archive = args.archive.expanduser().resolve()
    records = []
    for manifest_path in sorted((archive / "runs").glob("*/manifest.terminal.json")):
        manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
        if manifest.get("arm") not in ARMS:
            continue
        metrics_path = manifest_path.parent / "metrics.json"
        metrics = json.loads(metrics_path.read_text(encoding="utf-8")) if metrics_path.exists() else {}
        records.append(manifest | {"metrics": metrics})
    by_key: dict[tuple[str, str], dict[str, dict[str, Any]]] = {}
    for record in records:
        by_key.setdefault((record["topic"], record["mode"]), {})[record["arm"]] = record
    pairs: list[dict[str, Any]] = []
    for (topic, mode), arms in sorted(by_key.items()):
        if all(arm in arms and arms[arm]["status"] == "completed" for arm in ARMS):
            baseline = arms["visual_plan"]
            candidate = arms["visual_type_manual"]
            pairs.append(
                {
                    "topic": topic,
                    "mode": mode,
                    "baseline_visual_aids": baseline["metrics"].get("visual_aid_count"),
                    "baseline_mermaid_types": baseline["metrics"].get("mermaid_type_counts"),
                    "baseline_alignment_score": baseline["metrics"].get("visual_alignment_score"),
                    "candidate_visual_aids": candidate["metrics"].get("visual_aid_count"),
                    "candidate_mermaid_types": candidate["metrics"].get("mermaid_type_counts"),
                    "candidate_alignment_score": candidate["metrics"].get("visual_alignment_score"),
                    "candidate": candidate_summary(candidate, baseline),
                }
            )
    result = {
        "experiment": EXPERIMENT_ID,
        "records": len(records),
        "paired_completed": len(pairs),
        "failures": [record for record in records if record.get("status") != "completed"],
        "arm_summaries": summarize_arms(records),
        "candidate_summary": summarize_candidate(pairs),
        "pairs": pairs,
        "manual_review_note": "Automatic counts only show whether useful structures appeared. Read whole reports to judge usefulness, readability, and source grounding.",
    }
    base.write_json(archive / "analysis/aggregate.json", result)
    print(json.dumps(result, indent=2, ensure_ascii=False))


def candidate_summary(candidate: dict[str, Any], baseline: dict[str, Any]) -> dict[str, Any]:
    metrics = candidate.get("metrics", {})
    base_metrics = baseline.get("metrics", {})
    return {
        "words": metrics.get("final_word_count"),
        "visual_aids": metrics.get("visual_aid_count"),
        "mermaid_types": metrics.get("mermaid_type_counts"),
        "alignment_score": metrics.get("visual_alignment_score"),
        "alignment_hits": metrics.get("visual_alignment_hits"),
        "alignment_missing": metrics.get("visual_alignment_missing"),
        "visual_delta": delta(metrics.get("visual_aid_count"), base_metrics.get("visual_aid_count")),
        "alignment_delta": delta(metrics.get("visual_alignment_score"), base_metrics.get("visual_alignment_score")),
        "word_ratio_over_baseline": ratio(metrics.get("final_word_count"), base_metrics.get("final_word_count")),
        "unvalidated_mermaid_signal": metrics.get("has_unvalidated_mermaid_signal"),
    }


def summarize_arms(records: list[dict[str, Any]]) -> dict[str, dict[str, Any]]:
    summaries: dict[str, dict[str, Any]] = {}
    for arm in ARMS:
        arm_records = [record for record in records if record.get("arm") == arm and record.get("status") == "completed"]
        summaries[arm] = {
            "completed": len(arm_records),
            "median_visual_aids": base.median([float(record["metrics"].get("visual_aid_count", 0)) for record in arm_records]),
            "median_alignment_score": base.median([float(record["metrics"].get("visual_alignment_score", 0)) for record in arm_records]),
            "mermaid_type_totals": merge_mermaid_type_counts(arm_records),
            "unvalidated_mermaid_signals": sum(1 for record in arm_records if record["metrics"].get("has_unvalidated_mermaid_signal")),
        }
    return summaries


def summarize_candidate(pairs: list[dict[str, Any]]) -> dict[str, Any]:
    visual_deltas: list[float] = []
    alignment_deltas: list[float] = []
    word_ratios: list[float] = []
    for pair in pairs:
        summary = pair["candidate"]
        visual_delta = summary.get("visual_delta")
        alignment_delta = summary.get("alignment_delta")
        word_ratio = summary.get("word_ratio_over_baseline")
        if isinstance(visual_delta, (int, float)):
            visual_deltas.append(float(visual_delta))
        if isinstance(alignment_delta, (int, float)):
            alignment_deltas.append(float(alignment_delta))
        if isinstance(word_ratio, (int, float)):
            word_ratios.append(float(word_ratio))
    return {
        "completed_pairs": len(pairs),
        "median_visual_delta": base.median(visual_deltas),
        "visual_increase_sign_p_one_sided": base.exact_one_sided_sign_test(sum(1 for value in visual_deltas if value > 0), sum(1 for value in visual_deltas if value < 0)),
        "median_alignment_delta": base.median(alignment_deltas),
        "alignment_increase_sign_p_one_sided": base.exact_one_sided_sign_test(sum(1 for value in alignment_deltas if value > 0), sum(1 for value in alignment_deltas if value < 0)),
        "median_word_ratio_over_baseline": base.median(word_ratios),
    }


def merge_mermaid_type_counts(records: list[dict[str, Any]]) -> dict[str, int]:
    merged: dict[str, int] = {}
    for record in records:
        counts = record.get("metrics", {}).get("mermaid_type_counts", {})
        if not isinstance(counts, dict):
            continue
        for key, value in counts.items():
            if isinstance(value, (int, float)):
                merged[str(key)] = merged.get(str(key), 0) + int(value)
    return merged


def packets(args: argparse.Namespace) -> None:
    archive = args.archive.expanduser().resolve()
    analysis = json.loads((archive / "analysis/aggregate.json").read_text(encoding="utf-8"))
    out = archive / "judging/packets"
    out.mkdir(parents=True, exist_ok=True)
    for stale in out.glob("*.json"):
        stale.unlink()
    mapping = {}
    rng = random.Random(args.seed)
    count = 0
    for pair in analysis["pairs"]:
        topic = pair["topic"]
        mode = pair["mode"]
        labels = list(ARMS)
        rng.shuffle(labels)
        packet = {
            "packet_id": f"{EXPERIMENT_ID}-{topic}-{mode}",
            "topic": topic,
            "mode": mode,
            "review_questions": [
                "Does the report choose visual types that match the source structure?",
                "Do quantitative, benchmark, or architecture visuals preserve exact source-backed facts?",
                "Are diagrams and tables useful supplements rather than decorative or repetitive blocks?",
                "For architecture dependency material, does the visual make relationships and blast-radius risk easier to understand?",
                "Does the prose still read as a coherent report after the visuals are added?",
            ],
        }
        for label, arm in zip(("A", "B"), labels):
            report = (archive / "runs" / f"{topic}-{mode}-{arm}" / "report.md").read_text(encoding="utf-8")
            packet[label] = {"report_markdown": report}
            mapping[f"{topic}:{mode}:{label}"] = arm
        base.write_json(out / f"{topic}-{mode}.json", packet)
        count += 1
    base.write_json(archive / "judging/private-mapping.json", mapping)
    print(json.dumps({"packets": count, "path": str(out)}, ensure_ascii=False))


def write_text_new_or_same(path: Path, value: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    if path.exists() and path.read_text(encoding="utf-8") != value:
        raise RuntimeError(f"existing file differs: {path}")
    if not path.exists():
        path.write_text(value, encoding="utf-8")


def fixture_to_json(fixture: Fixture) -> dict[str, Any]:
    return {
        "topic": fixture.topic,
        "title": fixture.title,
        "objective": fixture.objective,
        "source_bundle": str(fixture.source_bundle),
        "source_sha256": fixture.source_sha256,
        "expected_visual_families": list(fixture.expected_visual_families),
    }


def delta(candidate: object, baseline: object) -> float | None:
    if not isinstance(candidate, (int, float)) or not isinstance(baseline, (int, float)):
        return None
    return float(candidate) - float(baseline)


def ratio(numerator: object, denominator: object) -> float | None:
    if not isinstance(numerator, (int, float)) or not isinstance(denominator, (int, float)) or denominator == 0:
        return None
    return numerator / denominator


def utc_now() -> str:
    return datetime.now(timezone.utc).replace(tzinfo=None).isoformat(timespec="seconds") + "Z"


def main() -> int:
    args = parse_args()
    if args.action == "prepare":
        prepare(args)
    elif args.action == "run":
        run(args)
    elif args.action == "analyze":
        analyze(args)
    elif args.action == "packets":
        packets(args)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
