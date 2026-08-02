from __future__ import annotations

import re


class GuardError(ValueError):
    def __init__(self, failures: list[str]) -> None:
        super().__init__("hard gates failed: " + ", ".join(failures))
        self.failures = failures


HEADING_RE = re.compile(r"^#{1,6}\s+")
LIST_RE = re.compile(r"^(\s*)([-*+]|\d+[.)])\s+(\[[ xX]\]\s+)?")
LINK_RE = re.compile(r"!?\[[^\]]*\]\([^)]+\)|https?://[^\s)<]+")
FOOTNOTE_RE = re.compile(r"\[\^[^\]]+\](?::[^\n]*)?")
BRACKET_RE = re.compile(r"\[[^\]\n]+\]")
INLINE_CODE_RE = re.compile(r"`[^`\n]+`")
QUOTE_RE = re.compile(r"\"[^\"\n]+\"|'[^'\n]+'|“[^”\n]+”|‘[^’\n]+’")
NUMBER_RE = re.compile(r"[-+]?\d+(?:[.,:/-]\d+)*(?:%|[A-Za-z가-힣]+)?")
LATIN_RE = re.compile(r"[A-Za-z][A-Za-z0-9._:+/#-]*[A-Za-z0-9]")
TERMINATOR_RE = re.compile(r"[.!?。？！]")
STRUCTURE_INLINE_RE = re.compile(r"(`+|\*\*|__|~~|!\[|\]\(|\[[^\]\n]*\])")
PRODUCT_RE = re.compile(r"\b(?:Codex|OpenAI|GPT-?\d+(?:\.\d+)?|GPT|Plasma|Liquid2|Liquid|H5|Markdown|Mermaid|Go|Raft)\b", re.IGNORECASE)
SOURCE_BULLET_RE = re.compile(r"^\s*[-*+]\s+(source|reference|citation|evidence|출처|참고|근거|인용)\s*[:：]", re.IGNORECASE)
HARD_GATE_REASON_CODES = (
    "line_count",
    "blank_line_positions",
    "nonempty_paragraph_shape",
    "heading_lines",
    "sentence_terminators_per_line",
    "code_fence_blocks",
    "table_lines",
    "blockquote_lines",
    "quoted_text",
    "source_bearing_lines",
    "footnotes",
    "bracket_tokens",
    "links_urls",
    "inline_code",
    "list_markers",
    "numbers_dates_percentages",
    "model_product_names",
    "latin_technical_tokens",
    "markdown_structure_tokens",
    "changed_line_budget",
    "changed_text_budget",
    "line_locality",
)


def validate_hard_gates(original: str, candidate: str) -> None:
    failures = hard_gate_failures(original, candidate)
    if failures:
        raise GuardError(failures)


def hard_gate_failures(original: str, candidate: str) -> list[str]:
    failures: list[str] = []
    original_lines = _lines(original)
    candidate_lines = _lines(candidate)
    _check(failures, "line_count", len(original_lines) == len(candidate_lines))
    _check(failures, "blank_line_positions", _blank_positions(original_lines) == _blank_positions(candidate_lines))
    _check(failures, "nonempty_paragraph_shape", _paragraph_shape(original_lines) == _paragraph_shape(candidate_lines))
    _check(failures, "heading_lines", _heading_lines(original) == _heading_lines(candidate))
    _check(failures, "sentence_terminators_per_line", _line_terminator_counts(original_lines) == _line_terminator_counts(candidate_lines))
    _check(failures, "code_fence_blocks", _code_fence_blocks(original) == _code_fence_blocks(candidate))
    _check(failures, "table_lines", _matching_lines(original, _is_table_line) == _matching_lines(candidate, _is_table_line))
    _check(failures, "blockquote_lines", _matching_lines(original, lambda line: line.strip().startswith(">")) == _matching_lines(candidate, lambda line: line.strip().startswith(">")))
    _check(failures, "quoted_text", _find_all(QUOTE_RE, _without_fences(original)) == _find_all(QUOTE_RE, _without_fences(candidate)))
    _check(failures, "source_bearing_lines", _source_bearing_lines(original) == _source_bearing_lines(candidate))
    _check(failures, "footnotes", _find_all(FOOTNOTE_RE, original) == _find_all(FOOTNOTE_RE, candidate))
    _check(failures, "bracket_tokens", _find_all(BRACKET_RE, original) == _find_all(BRACKET_RE, candidate))
    _check(failures, "links_urls", _find_all(LINK_RE, original) == _find_all(LINK_RE, candidate))
    _check(failures, "inline_code", _find_all(INLINE_CODE_RE, original) == _find_all(INLINE_CODE_RE, candidate))
    _check(failures, "list_markers", _list_markers(original) == _list_markers(candidate))
    _check(failures, "numbers_dates_percentages", _find_all(NUMBER_RE, original) == _find_all(NUMBER_RE, candidate))
    _check(failures, "model_product_names", _find_all(PRODUCT_RE, original) == _find_all(PRODUCT_RE, candidate))
    _check(failures, "latin_technical_tokens", _find_all(LATIN_RE, original) == _find_all(LATIN_RE, candidate))
    _check(failures, "markdown_structure_tokens", _markdown_structure_tokens(original) == _markdown_structure_tokens(candidate))
    failures.extend(_change_budget_failures(original, candidate, original_lines, candidate_lines))
    return failures


def _change_budget_failures(original: str, candidate: str, original_lines: list[str], candidate_lines: list[str]) -> list[str]:
    if len(original_lines) != len(candidate_lines):
        return []
    nonempty = sum(1 for line in original_lines if line.strip())
    changed_lines = 0
    changed_span = 0
    line_locality_failure = False
    for original_line, candidate_line in zip(original_lines, candidate_lines):
        a = original_line.strip()
        b = candidate_line.strip()
        if a == b:
            continue
        changed_lines += 1
        span, stable_prefix, stable_suffix = changed_middle_rune_metrics(a, b)
        changed_span += span
        if span > max_line_changed_runes(a, b):
            line_locality_failure = True
        if stable_prefix < 4 and span > 40:
            line_locality_failure = True
        if short_line_semantic_rewrite_risk(a, b, span, stable_prefix, stable_suffix):
            line_locality_failure = True
    failures: list[str] = []
    if changed_lines > max_changed_lines(nonempty):
        failures.append("changed_line_budget")
    if changed_span > max_changed_runes(original):
        failures.append("changed_text_budget")
    if line_locality_failure:
        failures.append("line_locality")
    return failures


def max_changed_lines(nonempty_lines: int) -> int:
    if nonempty_lines <= 0:
        return 0
    limit = nonempty_lines // 3
    if nonempty_lines >= 24 and limit < 8:
        limit = 8
    elif limit < 1:
        limit = 1
    return min(limit, 48)


def max_changed_runes(original: str) -> int:
    return min(max(len(original) // 4, 1200), 8000)


def max_line_changed_runes(a: str, b: str) -> int:
    longer = max(len(a), len(b))
    limit = longer // 2
    if longer <= 80:
        limit = max(limit, 18)
    elif longer <= 220:
        limit = max(limit, 72)
    else:
        limit = max(limit, 220)
    return min(limit, 480)


def short_line_semantic_rewrite_risk(a: str, b: str, span: int, stable_prefix: int, stable_suffix: int) -> bool:
    return max(len(a), len(b)) <= 120 and span >= 10 and stable_prefix < 8 and stable_suffix < 8


def changed_middle_rune_metrics(a: str, b: str) -> tuple[int, int, int]:
    prefix = 0
    while prefix < len(a) and prefix < len(b) and a[prefix] == b[prefix]:
        prefix += 1
    a_suffix = len(a)
    b_suffix = len(b)
    while a_suffix > prefix and b_suffix > prefix and a[a_suffix - 1] == b[b_suffix - 1]:
        a_suffix -= 1
        b_suffix -= 1
    stable_suffix = min(len(a) - a_suffix, len(b) - b_suffix)
    return max(a_suffix - prefix, b_suffix - prefix), prefix, stable_suffix


def _lines(text: str) -> list[str]:
    return text.rstrip("\n").split("\n")


def _blank_positions(lines: list[str]) -> list[int]:
    return [index for index, line in enumerate(lines, 1) if not line.strip()]


def _paragraph_shape(lines: list[str]) -> list[tuple[int, int]]:
    spans: list[tuple[int, int]] = []
    start: int | None = None
    for index, line in enumerate(lines, 1):
        if line.strip() and start is None:
            start = index
        if start is not None and (not line.strip() or index == len(lines)):
            end = index - 1 if not line.strip() else index
            spans.append((start, end))
            start = None
    return spans


def _without_fences(text: str) -> str:
    return "\n".join(line for _, line in _outside_fence_lines(text))


def _outside_fence_lines(text: str) -> list[tuple[int, str]]:
    out: list[tuple[int, str]] = []
    in_fence = False
    for index, line in enumerate(text.split("\n"), 1):
        if _is_fence_line(line):
            in_fence = not in_fence
            continue
        if not in_fence:
            out.append((index, line))
    return out


def _heading_lines(text: str) -> list[str]:
    return [line for _, line in _outside_fence_lines(text) if HEADING_RE.match(line.strip())]


def _line_terminator_counts(lines: list[str]) -> list[int]:
    return [len(TERMINATOR_RE.findall(line)) for line in lines]


def _code_fence_blocks(text: str) -> list[str]:
    blocks: list[str] = []
    current: list[str] = []
    in_fence = False
    for line in text.split("\n"):
        if _is_fence_line(line):
            current.append(line)
            if in_fence:
                blocks.append("\n".join(current))
                current = []
            in_fence = not in_fence
        elif in_fence:
            current.append(line)
    if current:
        blocks.append("\n".join(current))
    return blocks


def _matching_lines(text: str, keep: object) -> list[str]:
    return [line for _, line in _outside_fence_lines(text) if keep(line)]  # type: ignore[operator]


def _is_table_line(line: str) -> bool:
    stripped = line.strip()
    return stripped.startswith("|") and stripped.endswith("|")


def _is_fence_line(line: str) -> bool:
    stripped = line.strip()
    return stripped.startswith("```") or stripped.startswith("~~~")


def _source_bearing_lines(text: str) -> list[str]:
    out: list[str] = []
    in_source_section = False
    for index, line in _outside_fence_lines(text):
        stripped = line.strip()
        if HEADING_RE.match(stripped):
            in_source_section = _is_source_section_heading(stripped)
            if in_source_section:
                out.append(f"{index}:{line}")
            continue
        if in_source_section or _is_source_bearing_line(line):
            out.append(f"{index}:{line}")
    return out


def _is_source_bearing_line(line: str) -> bool:
    stripped = line.strip()
    lower = stripped.lower()
    prefixes = ("source:", "sources:", "reference:", "references:", "citation:", "citations:", "evidence:")
    korean = ("출처:", "출처：", "참고:", "참고：", "근거:", "근거：", "인용:", "인용：")
    return (
        lower.startswith(prefixes)
        or stripped.startswith(korean)
        or SOURCE_BULLET_RE.match(line) is not None
        or (stripped.startswith("[^") and "]:" in stripped)
    )


def _is_source_section_heading(line: str) -> bool:
    title = HEADING_RE.sub("", line).strip(" #").lower()
    return title in {
        "source", "sources", "reference", "references", "citation", "citations", "evidence",
        "출처", "참고", "참고자료", "참고 자료", "근거", "인용", "출처 및 참고자료", "출처와 참고자료",
    }


def _list_markers(text: str) -> list[str]:
    markers: list[str] = []
    for _, line in _outside_fence_lines(text):
        match = LIST_RE.match(line)
        if match:
            markers.append("".join(part for part in match.groups() if part))
    return markers


def _markdown_structure_tokens(text: str) -> list[str]:
    tokens: list[str] = []
    for index, line in enumerate(text.split("\n"), 1):
        stripped = line.strip()
        line_tokens: list[str] = []
        if _is_fence_line(line):
            line_tokens.append("fence:" + stripped.split()[0])
        if match := HEADING_RE.match(stripped):
            line_tokens.append("heading:" + match.group(0))
        if match := LIST_RE.match(line):
            line_tokens.append("list:" + "".join(part for part in match.groups() if part))
        if stripped.startswith(">"):
            line_tokens.append("blockquote")
        if _is_table_line(line):
            line_tokens.append("table:" + str(line.count("|")))
        line_tokens.extend(STRUCTURE_INLINE_RE.findall(line))
        if line_tokens:
            tokens.append(f"{index}:{'|'.join(line_tokens)}")
    return tokens


def _find_all(pattern: re.Pattern[str], text: str) -> list[str]:
    return pattern.findall(text)


def _check(failures: list[str], reason: str, ok: bool) -> None:
    if not ok:
        failures.append(reason)
