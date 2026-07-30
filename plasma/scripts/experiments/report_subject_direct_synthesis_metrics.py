"""Advisory stage metrics for issue #189 subject-direct synthesis runs."""

from __future__ import annotations

from pathlib import Path
import re
import sqlite3


SOURCE_NARRATOR_PATTERNS = tuple(
    re.compile(pattern, re.IGNORECASE)
    for pattern in (
        r"\b(?:source|document|article|report|material)s?\s+(?:says?|states?|notes?|describes?|explains?|shows?|mentions?|reports?)\b",
        r"\baccording to (?:the )?(?:source|document|article|report|material)\b",
        r"\bofficial documentation\s+(?:says?|states?|notes?|describes?|explains?|shows?|mentions?|reports?)\b",
        r"\b(?!official documentation\b)[A-Z][A-Za-z0-9._ -]{1,40}\s+(?:document|docs?|book|guide|manual)\s+(?:says?|states?|notes?|describes?|explains?|shows?|mentions?|reports?)\b",
        r"[A-Z][A-Za-z0-9_-]*(?:\s+[A-Z][A-Za-z0-9_-]*){0,5}(?:\s*(?:문서|자료|책|가이드))?(?:는|은|가|이)\s*[^.?!\n]{0,80}(?:말한다|설명한다|보여준다|언급한다|제시한다|다룬다)",
        r"(?<![A-Za-z0-9] )(?:자료|문서|출처|보고서|글|본문)(?:는|은|가|이)\s*[^.?!\n]{0,80}(?:말한다|설명한다|보여준다|언급한다|제시한다|다룬다)",
        r"(?:자료|문서|출처|보고서|글|본문)에\s*(?:따르면|의하면)",
    )
)
EXTERNAL_MARKDOWN_LINK_PATTERN = re.compile(r"(?<!!)\[[^\]\n]{1,120}\]\(https?://[^)\s]+\)")
PARENTHETICAL_PATTERN = re.compile(r"\(([^()\n]{2,120})\)")
PARENTHETICAL_CITATION_SHAPE = re.compile(
    r"^(?P<label>[^,]+?)(?P<comma>,?)\s+(?:18|19|20)\d{2}[a-z]?$"
)
LATIN_NAME_PATTERN = re.compile(
    r"(?:[A-Z][A-Za-z.'-]{1,}|[A-Z]{2,})"
    r"(?:\s+(?:(?:for|of|and|the|&)\s+)?(?:[A-Z][A-Za-z.'-]{1,}|[A-Z]{2,})){0,7}"
)
LATIN_ET_AL_PATTERN = re.compile(r"[A-Z][A-Za-z.'-]{1,}\s+et\s+al\.")
LATIN_ACRONYM_PATTERN = re.compile(r"[A-Z]{2,}")
LATIN_ORGANIZATION_WORD_PATTERN = re.compile(
    r"\b(?:Agency|Association|Academies|Administration|Bank|Bureau|Centers|Commission|Committee|Council|Department|Foundation|Group|Institute|Ministry|Office|Organization|Society|University)\b"
)
KOREAN_NAME_PATTERN = re.compile(r"[가-힣][가-힣A-Za-z0-9·(). -]{1,60}")
DATE_NOTE_PATTERN = re.compile(r"^(?:in|as of|q[1-4]|fy|jan(?:uary)?|feb(?:ruary)?|mar(?:ch)?|apr(?:il)?|may|jun(?:e)?|jul(?:y)?|aug(?:ust)?|sep(?:t(?:ember)?)?|oct(?:ober)?|nov(?:ember)?|dec(?:ember)?)\s+\b(?:18|19|20)\d{2}\b$", re.IGNORECASE)
FOOTNOTE_CITATION_PATTERN = re.compile(r"\[\^[A-Za-z0-9_.:-]+\](?=:)?")


def collect_stage_metrics(database: Path, ratio) -> dict[str, object]:
    stage_by_artifact = artifact_stage_map(database)
    stages = {
        "section": empty_stage_metrics(),
        "part": empty_stage_metrics(),
        "final": empty_stage_metrics(),
        "unmapped": empty_stage_metrics(),
    }
    with sqlite3.connect(database) as connection:
        rows = connection.execute("select artifact_id, content_blob from plasma_raw_artifacts").fetchall()
    for artifact_id, blob in rows:
        stage = stage_by_artifact.get(str(artifact_id), "unmapped")
        text = bytes(blob).decode("utf-8")
        update_stage_metrics(stages[stage], text)
    return {
        "stages": stages,
        "part_connective_characters": stages["part"]["characters"] - stages["section"]["characters"],
        "final_character_delta": stages["final"]["characters"] - stages["part"]["characters"],
        "section_to_final_character_ratio": ratio(stages["final"]["characters"], stages["section"]["characters"]),
    }


def empty_stage_metrics() -> dict[str, int]:
    return {
        "artifacts": 0,
        "characters": 0,
        "words": 0,
        "citation_count": 0,
        "source_narrator_candidates": 0,
        "heading_count": 0,
    }


def update_stage_metrics(metrics: dict[str, int], text: str) -> None:
    metrics["artifacts"] += 1
    metrics["characters"] += len(text)
    metrics["words"] += len(text.split())
    metrics["citation_count"] += count_citations(text)
    metrics["source_narrator_candidates"] += count_source_narrator_candidates(text)
    metrics["heading_count"] += sum(1 for line in text.splitlines() if line.startswith("#"))


def count_citations(text: str) -> int:
    return (
        len(EXTERNAL_MARKDOWN_LINK_PATTERN.findall(text))
        + count_parenthetical_citations(text)
        + len(FOOTNOTE_CITATION_PATTERN.findall(text))
    )


def count_parenthetical_citations(text: str) -> int:
    return sum(1 for value in PARENTHETICAL_PATTERN.findall(text) if is_parenthetical_citation(value))


def is_parenthetical_citation(value: str) -> bool:
    inner = " ".join(value.strip().split())
    if DATE_NOTE_PATTERN.fullmatch(inner):
        return False
    match = PARENTHETICAL_CITATION_SHAPE.fullmatch(inner)
    if not match:
        return False
    label = match.group("label")
    has_comma = bool(match.group("comma"))
    if KOREAN_NAME_PATTERN.fullmatch(label):
        return has_comma
    if LATIN_ET_AL_PATTERN.fullmatch(label) or LATIN_ACRONYM_PATTERN.fullmatch(label):
        return True
    if not LATIN_NAME_PATTERN.fullmatch(label):
        return False
    return has_comma or bool(LATIN_ORGANIZATION_WORD_PATTERN.search(label))


def count_source_narrator_candidates(text: str) -> int:
    return sum(len(pattern.findall(text)) for pattern in SOURCE_NARRATOR_PATTERNS)


def artifact_stage_map(database: Path) -> dict[str, str]:
    mapping: dict[str, str] = {}
    with sqlite3.connect(database) as connection:
        rows = connection.execute(
            """
            select event_type, payload_json
            from plasma_ledger_events
            where event_type in ('report.section.created', 'report.part.created', 'report.artifact.created')
            """
        ).fetchall()
    for event_type, payload_json in rows:
        artifact_id = artifact_id_from_payload(str(payload_json))
        if artifact_id:
            mapping[artifact_id] = event_stage(str(event_type))
    return mapping


def artifact_id_from_payload(payload_json: str) -> str:
    import json

    try:
        payload = json.loads(payload_json)
    except json.JSONDecodeError:
        return ""
    value = payload.get("artifact_id") if isinstance(payload, dict) else None
    return str(value).strip() if isinstance(value, str) else ""


def event_stage(event_type: str) -> str:
    if event_type == "report.section.created":
        return "section"
    if event_type == "report.part.created":
        return "part"
    return "final"


def aggregate_stage_metrics(records: list[dict[str, object]], ratio) -> dict[str, object]:
    section_chars = sum(record["stages"]["section"]["characters"] for record in records)
    final_chars = sum(record["stages"]["final"]["characters"] for record in records)
    return {
        "completed": len(records),
        "section_characters": section_chars,
        "part_characters": sum(record["stages"]["part"]["characters"] for record in records),
        "final_characters": final_chars,
        "section_to_final_character_ratio": ratio(final_chars, section_chars),
        "section_citation_count": sum(record["stages"]["section"]["citation_count"] for record in records),
        "part_citation_count": sum(record["stages"]["part"]["citation_count"] for record in records),
        "final_citation_count": sum(record["stages"]["final"]["citation_count"] for record in records),
        "section_source_narrator_candidates": sum(record["stages"]["section"]["source_narrator_candidates"] for record in records),
        "part_source_narrator_candidates": sum(record["stages"]["part"]["source_narrator_candidates"] for record in records),
        "final_source_narrator_candidates": sum(record["stages"]["final"]["source_narrator_candidates"] for record in records),
        "final_words": sum(record["stages"]["final"]["words"] for record in records),
        "final_headings": sum(record["stages"]["final"]["heading_count"] for record in records),
        "metrics_section_count": sum((record.get("metrics") or {}).get("section_count") or 0 for record in records),
        "metrics_part_count": sum((record.get("metrics") or {}).get("part_count") or 0 for record in records),
    }
