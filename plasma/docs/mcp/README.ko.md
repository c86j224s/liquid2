# Plasma MCP 문서

[English](README.md)

이 디렉터리는 Plasma MCP 안내를 위한 사람이 읽는 저장소 및 Pages 표면입니다.

런타임 MCP resource는 `plasma/internal/mcpdocs`가 embed하는 코드 소유 복사본입니다.
`resources/read`는 그 embed 복사본만 제공합니다. 이 디렉터리의 영어 파일은 embed된
런타임 복사본과 byte-for-byte로 같아야 하며, 둘 중 하나가 drift되면 package test가
명확하게 실패합니다. 한국어 counterpart는 사람이 읽기 위한 문서이며 MCP resource
catalog에는 추가하지 않습니다.

이 문서는 공개 저장소에 안전한 정적 안내만 담습니다. Mission data, source body,
session identifier, ledger content, provider response, credential, private URL,
runtime state를 포함하면 안 됩니다.

## MCP Resource URI

| Resource URI | 영어 canonical | 한국어 counterpart |
| --- | --- | --- |
| `plasma://docs/mcp/tools` | [tools.md](tools.md) | [tools.ko.md](tools.ko.md) |
| `plasma://docs/mcp/errors` | [errors.md](errors.md) | [errors.ko.md](errors.ko.md) |
| `plasma://docs/mcp/reporting` | [reporting.md](reporting.md) | [reporting.ko.md](reporting.ko.md) |
| `plasma://docs/mcp/sources` | [sources.md](sources.md) | [sources.ko.md](sources.ko.md) |
| `plasma://docs/mcp/mermaid` | [mermaid.md](mermaid.md) | [mermaid.ko.md](mermaid.ko.md) |

Resource set은 `resources/list`로 확인하고, 하나의 영어 Markdown 문서는 URI로
`resources/read`를 호출해 가져옵니다.
