# Plasma MCP Source 안내

URI: `plasma://docs/mcp/sources`

이 문서는 Plasma MCP에서 source, source candidate, source snapshot, live local path observation을 구분하는 기준을 정의합니다.

## 용어 경계

- Source는 original research material입니다. 문서, URL, 파일, PDF, external repository 같은 원본 material을 뜻합니다.
- Evidence는 source의 특정 부분을 근거로 사용하는 것입니다.
- Result는 agent가 만든 summary, comparison, answer, conclusion, draft입니다. Result는 source가 아닙니다.
- Saved knowledge는 Plasma가 mission에 의도적으로 저장한 result나 claim입니다.
- Report는 saved knowledge와 evidence를 독자를 위해 조립한 output입니다.

Agent answer, controller output, report draft를 source로 재분류하지 않습니다. 필요하면 result가 어떤 source와 evidence에 의존하는지 설명합니다.

## Source snapshot

`plasma.sources.list`와 `plasma.sources.read`는 사용자가 mission에 받아들인 source snapshot을 다룹니다. Soft-removed 또는 superseded source는 사용자가 audit 또는 history review를 명시적으로 요청하지 않는 한 기본 읽기나 새 보고서 작성에 쓰지 않습니다.

PDF와 upload source에서는 원본 파일이 source입니다. MCP read tool은 원본 bytes를 그대로 반환하지 않고 bounded extracted text와 extraction metadata를 반환합니다.

## Connector Search와 Candidate

`plasma.sources.search` 결과는 candidate이지 source가 아닙니다. Search result, grep match, connector title은 읽고 판단하기 전까지 evidence나 saved knowledge가 아닙니다.

사용자 검토 가치가 있는 original material은 `plasma.sources.candidates.propose`로 기록합니다. 이 호출은 review candidate를 기록할 뿐입니다. Source snapshot을 만들지 않고 report의 default source set에도 넣지 않습니다.

`plasma.sources.candidates.read`는 staged unapproved candidate를 대화와 조사 목적으로 읽습니다. Candidate content는 기본 보고서 근거로 쓰기 전에 사용자 승인 source snapshot이 되어야 합니다.

## Live local path

Live local path source는 configured root의 `root_id`와 `relative_path`로만 지정합니다. Absolute filesystem path는 MCP tool input이나 문서 예시로 쓰지 않습니다.

Live material을 읽으면 새 source body snapshot을 만들지 않고 bounded observation metadata를 기록합니다. Report가 live material에 의존하면 가능한 경우 observed time, relative path, content hash, git metadata 같은 observation event metadata를 인용합니다. Source id만 단독으로 인용하지 않습니다.

## 안전한 보고

Source read를 요약할 때는 독자에게 필요한 짧은 부분만 사용합니다. Credentials, private URL, provider 응답, 전체 문서 body를 보고서나 오류 설명에 붙여 넣지 않습니다.
