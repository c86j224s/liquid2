"""Deterministic layout and integrity metrics for gap-stress reports."""

from __future__ import annotations

import csv
import re
from collections import defaultdict
from pathlib import Path
from typing import Any


HEADING_RE = re.compile(r"^(#{1,6})[ \t]+(.+?)\s*#*\s*$", re.MULTILINE)
CORE_RE = re.compile(r"결론|핵심|요약|판단|권고|추천")
LIMIT_HEADING_RE = re.compile(
    r"정보\s*(?:한계|제한|공백)(?:\s*(?:및|과|와)\s*영향)?|"
    r"(?:조사|자료)\s*(?:범위|한계|제한|공백)|"
    r"(?:누락(?:된)?|남은|미해결)\s*(?:정보|사항|과제|공백|질문|근거)|"
    r"(?:한계|제한)\s*(?:및|과|와)\s*영향"
)
PROCESS_RE = re.compile(
    r"조사\s*(?:과정|실패|시도)|검색\s*(?:과정|실패|시도)|"
    r"(?:자료|원문|페이지|문서)\s*접근\s*(?:불가|실패|거부|하지\s*못)|"
    r"접근\s*(?:권한\s*)?(?:거부|불가|실패)|권한\s*거부|세션\s*만료|"
    r"(?:후보|자료|예제)[^\n.]{0,24}제외|(?:확인|찾)지\s*못"
)
GAP_RE = re.compile(
    r"누락\s*(?:정보|자료|근거|증거)|(?:근거|증거|자료)[^\n.]{0,24}"
    r"(?:부족|부재|없(?:다|어)|확인되지)|미해결|정보\s*공백|"
    r"원자료만으로[^\n.]{0,24}(?:결정되지|해결되지)|확정하지\s*못"
)
CONFLATION_RE = re.compile(
    r"(?:원자료|출처|source)(?:의|에서|에\s*포함된)[^\n.]{0,45}"
    r"(?:조사\s*결과|investigation_result|이전\s*결과)|"
    r"(?:조사\s*결과|investigation_result|이전\s*결과)[^\n.]{0,45}"
    r"(?:원자료|출처|source)(?:로|라고|처럼)",
    re.IGNORECASE,
)


def _headings(text: str) -> list[tuple[int, int, int, str]]:
    return [
        (match.start(), match.end(), len(match.group(1)), match.group(2))
        for match in HEADING_RE.finditer(text)
    ]


def _first_heading(headings: list[tuple[int, int, int, str]], pattern: re.Pattern[str]) -> int:
    return next((start for start, _, _, title in headings if pattern.search(title)), -1)


def _limitation_span(
    text: str,
    headings: list[tuple[int, int, int, str]],
) -> tuple[int, int, int]:
    for index, (start, _, level, title) in enumerate(headings):
        if not LIMIT_HEADING_RE.search(title):
            continue
        for next_start, _, next_level, _ in headings[index + 1:]:
            if next_level <= level:
                return index, start, next_start
        return index, start, len(text)
    return -1, -1, -1


def _hits(pattern: re.Pattern[str], text: str) -> int:
    return len(pattern.findall(text))


def report_metrics(path: Path) -> dict[str, Any]:
    text = path.read_text(encoding="utf-8", errors="replace").strip()
    headings = _headings(text)
    core_start = _first_heading(headings, CORE_RE)
    limit_index, limit_start, limit_end = _limitation_span(text, headings)
    early_end = min(1500, max(500, len(text) // 4))
    outside = text if limit_start < 0 else text[:limit_start] + text[limit_end:]
    limit_size = max(0, limit_end - limit_start)
    return {
        "path": str(path),
        "char_count": len(text),
        "word_count": len(text.split()),
        "heading_count": len(headings),
        "first_headings": " | ".join(title for _, _, _, title in headings[:5]),
        "first_core_heading_char": core_start,
        "first_core_heading_ratio": round(core_start / len(text), 4) if text and core_start >= 0 else -1,
        "limitation_section_present": int(limit_start >= 0),
        "limitation_heading_index": limit_index,
        "first_limitation_heading_char": limit_start,
        "limitation_heading_late_ratio": round(limit_start / len(text), 4) if text and limit_start >= 0 else -1,
        "limitation_before_core": int(limit_start >= 0 and (core_start < 0 or limit_start < core_start)),
        "limitation_section_chars": limit_size,
        "limitation_section_share": round(limit_size / len(text), 4) if text and limit_start >= 0 else -1,
        "early_process_mentions": _hits(PROCESS_RE, text[:early_end]),
        "process_mentions_outside_limitation": _hits(PROCESS_RE, outside),
        "gap_mentions_outside_limitation": _hits(GAP_RE, outside),
        "source_result_conflation_total": _hits(CONFLATION_RE, text),
    }


def aggregate_metrics(rows: list[dict[str, Any]]) -> list[dict[str, Any]]:
    excluded = {"path", "sample", "mode", "variant", "first_headings"}
    numeric = [key for key in rows[0] if key not in excluded] if rows else []
    groups: dict[tuple[str, str], list[dict[str, Any]]] = defaultdict(list)
    for row in rows:
        groups[(row["mode"], row["variant"])].append(row)

    summaries = []
    for (mode, variant), values in sorted(groups.items()):
        summary: dict[str, Any] = {"mode": mode, "variant": variant, "reports": len(values)}
        for key in numeric:
            valid = [float(row[key]) for row in values if float(row[key]) >= 0]
            summary[f"mean_{key}"] = round(sum(valid) / len(valid), 4) if valid else -1
        summaries.append(summary)
    return summaries


def write_csv(path: Path, rows: list[dict[str, Any]]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    fields = sorted({key for row in rows for key in row})
    with path.open("w", encoding="utf-8", newline="") as handle:
        writer = csv.DictWriter(handle, fieldnames=fields)
        writer.writeheader()
        writer.writerows(rows)
