"""Deterministic reading-order metrics for issue 77 report candidates."""

from __future__ import annotations

import csv
import re
from collections import defaultdict
from pathlib import Path
from typing import Any


CORE_HEADING_RE = re.compile(r"^#{1,6}\s+.*(결론|핵심|요약|판단|권고|추천)", re.MULTILINE)
HEADING_RE = re.compile(r"^#{1,6}\s+(.+)$", re.MULTILINE)
PROCESS_PATTERNS = tuple(
    re.compile(pattern)
    for pattern in (
        r"조사\s*과정",
        r"자료\s*접근",
        r"검색\s*(과정|결과|한계)",
        r"수집\s*(과정|단계)",
        r"확인하지\s*못",
        r"찾지\s*못",
        r"원문에\s*접근",
        r"제공된\s*자료의\s*한계",
        r"이전\s*(조사\s*결과|실행)",
        r"working\s*memory",
    )
)
LIMIT_RE = re.compile(r"한계|불확실|주의|위험|검증\s*필요|확인\s*필요")
PRIOR_RESULT_RE = re.compile(r"이전\s*(조사\s*결과|실행)|working\s*memory")
RESULT_SOURCE_CONFLATION_RE = re.compile(r"원자료(?:의|에서)[^\n.]{0,30}이전\s*조사\s*결과")


def _pattern_hits(text: str) -> int:
    return sum(len(pattern.findall(text)) for pattern in PROCESS_PATTERNS)


def _first_match(patterns: tuple[re.Pattern[str], ...], text: str) -> int:
    positions = [match.start() for pattern in patterns if (match := pattern.search(text))]
    return min(positions) if positions else -1


def report_metrics(path: Path) -> dict[str, Any]:
    text = path.read_text(encoding="utf-8", errors="replace").strip()
    early_size = min(1500, max(500, len(text) // 4))
    early = text[:early_size]
    headings = HEADING_RE.findall(text)
    core_match = CORE_HEADING_RE.search(text)
    process_total = _pattern_hits(text)
    process_early = _pattern_hits(early)
    word_count = len(text.split())
    return {
        "path": str(path),
        "char_count": len(text),
        "word_count": word_count,
        "heading_count": len(headings),
        "first_headings": " | ".join(headings[:5]),
        "first_core_heading_char": core_match.start() if core_match else -1,
        "first_core_heading_ratio": round(core_match.start() / len(text), 4) if core_match and text else -1,
        "process_mentions_total": process_total,
        "process_mentions_early": process_early,
        "process_early_share": round(process_early / process_total, 4) if process_total else 0.0,
        "process_mentions_per_1000_words": round(process_total * 1000 / word_count, 4) if word_count else 0.0,
        "first_process_char": _first_match(PROCESS_PATTERNS, text),
        "prior_result_mentions_total": len(PRIOR_RESULT_RE.findall(text)),
        "result_source_conflation_total": len(RESULT_SOURCE_CONFLATION_RE.findall(text)),
        "limit_mentions_total": len(LIMIT_RE.findall(text)),
        "limit_mentions_early": len(LIMIT_RE.findall(early)),
    }


def aggregate_metrics(rows: list[dict[str, Any]]) -> list[dict[str, Any]]:
    numeric = (
        "char_count",
        "word_count",
        "heading_count",
        "first_core_heading_ratio",
        "process_mentions_total",
        "process_mentions_early",
        "process_early_share",
        "process_mentions_per_1000_words",
        "prior_result_mentions_total",
        "result_source_conflation_total",
        "limit_mentions_total",
        "limit_mentions_early",
    )
    groups: dict[tuple[str, str], list[dict[str, Any]]] = defaultdict(list)
    for row in rows:
        groups[(row["mode"], row["variant"])].append(row)
    result: list[dict[str, Any]] = []
    for (mode, variant), values in sorted(groups.items()):
        aggregate: dict[str, Any] = {"mode": mode, "variant": variant, "reports": len(values)}
        for key in numeric:
            valid = [float(value[key]) for value in values if float(value[key]) >= 0]
            aggregate[f"mean_{key}"] = round(sum(valid) / len(valid), 4) if valid else -1
        result.append(aggregate)
    return result


def write_csv(path: Path, rows: list[dict[str, Any]]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    fieldnames = sorted({key for row in rows for key in row})
    with path.open("w", encoding="utf-8", newline="") as handle:
        writer = csv.DictWriter(handle, fieldnames=fieldnames, extrasaction="ignore")
        writer.writeheader()
        writer.writerows(rows)
