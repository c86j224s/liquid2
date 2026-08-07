# Plasma MCP Mermaid 안내

URI: `plasma://docs/mcp/mermaid`

이 문서는 Plasma 보고서나 답변에 Mermaid diagram을 넣기 전에 적용할 점검 기준을 정의합니다.

## 기본 원칙

Mermaid diagram은 독자가 구조, 순서, 의존성, 비교 관계를 이해하는 데 도움이 될 때만 씁니다. 단순한 Markdown 목록으로 충분하면 diagram으로 바꾸지 않습니다.

Diagram source를 사용자에게 보여주기 전에 `plasma.mermaid.validate`를 호출합니다. 이 tool은 Plasma가 알고 있는 Mermaid parse 위험과 compatibility 위험을 정적으로 점검합니다.

## validate 결과 읽기

- `ok: true`: 알려진 정적 preflight 규칙을 통과했습니다. Browser render를 보장하지는 않습니다.
- `ok: false`: `errors`와 `warnings`를 읽고 source를 고친 뒤 다시 validate합니다.
- Warning만 있어도 보고서를 읽기 어렵게 만들 수 있는 pattern이면 고칩니다.

## 작성 규칙

- Node label, requirement text, 긴 설명은 parser가 잘못 해석하지 않도록 quote합니다.
- Label 안에 Markdown, HTML, 복잡한 punctuation을 많이 넣지 않습니다.
- Source body 전체나 긴 인용을 diagram 안에 넣지 않습니다. 짧은 label과 요약을 씁니다.
- ID에는 안정적인 ASCII token을 씁니다. 독자가 읽을 설명은 label에 둡니다.
- Diagram 자체는 evidence가 아닙니다. 보고서에서는 diagram이 요약하는 source와 evidence 연결을 본문에서 설명합니다.

## 검증 실패 시

검증 실패를 숨기고 diagram을 그대로 내보내지 않습니다. Parser 위험을 없애는 가장 작은 명확한 수정 후 다시 validate합니다. 계속 실패하면 diagram을 포기하고 일반 Markdown 구조를 씁니다.
