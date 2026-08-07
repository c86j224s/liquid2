# Plasma MCP 보고서 작성 안내

URI: `plasma://docs/mcp/reporting`

이 문서는 Plasma MCP reporting tool의 경계를 설명합니다. Report는 source나 agent transcript가 아닙니다. 독자가 주제를 이해할 수 있도록 saved knowledge와 evidence를 조립한 result입니다.

## 공통 경계

- Report tool은 지정된 report stage에서만 사용합니다.
- Runner나 server가 제공한 mission, session, pending event, artifact binding을 바꾸지 않습니다.
- Source, evidence, saved knowledge를 작성 편의를 위해 재분류하지 않습니다.
- 보고서 문장에는 provider 응답, credentials, private URL, runtime identifier를 넣지 않습니다.

## Plan과 Requirement 도구

`plasma.report.plan.submit`은 보고서 구조와 의도를 durable하게 제출합니다. 하나의 plan submission은 runner가 승격할 수 있는 하나의 계획이어야 합니다. 같은 의미의 재시도에만 같은 idempotency key를 재사용합니다.

Requirement mapping tool은 이미 고정된 outline에 명시적인 사용자 출력 요구를 붙입니다. Outline을 다시 설계하거나 research 방향을 바꾸는 tool이 아닙니다.

## Part Assembly와 Edit

Part assembly tool은 immutable Section body 주변의 connective Markdown을 씁니다. Section body를 다시 쓰지 않고 introduction, transition, closing text를 다룹니다.

Part edit tool은 하나의 assembled Part를 isolated draft로 열고 bounded edit을 적용합니다. 다른 Part나 Section artifact를 직접 변경하지 않습니다.

## Long-form finalization

Long-form finalization과 final edit stage tool은 runner가 제공한 stage binding 안에서만 동작합니다. Writer, reader, style, gate, evidence-gate stage는 서로 다른 책임을 가집니다. 한 stage의 tool로 다른 stage의 policy를 대신 적용하지 않습니다.

Finalization은 단순한 파일 저장이 아닙니다. Ledger boundary를 통해 기록되는 제품 상태 전이입니다. 오류가 나면 현재 draft 또는 stage 상태를 다시 읽고, 다음 호출이 같은 재시도인지 새 변경인지 구분한 뒤 진행합니다.

## Patch Tool

Report patch tool은 보고서 전체를 prompt에 붙이지 않고 bounded slice와 작은 edit operation으로 기존 Markdown report artifact를 수정합니다. Patch tool은 새 source material을 읽는 research tool이 아닙니다. 기존 보고서의 안전한 edit boundary 안에서만 사용합니다.
