# 제3 IL 보고서 파이프라인 Phase 0

Issue: #357

상태: archive-local experimental prototype

## 목적

이 prototype은 기존 일반·장문 보고서 pipeline과 연결되지 않는 제3 경로의 첫 compiler boundary를 검증한다.

```text
Narrative/Discourse Plan
  → paragraph prose leaf 기반 Semantic Document IL
  → linear manuscript
  → whole-manuscript flow attestation
  → Markdown | static self-contained HTML | minimal PDF
```

제품 report workflow, DB, durable event, prompt, provider, API, UI를 사용하거나 변경하지 않는다. 기존 일반·장문 결과는 이후 paired study에서 control로만 사용한다.

## 세 실험 arm

- `T0`: thin semantic IL만 사용한다.
- `T1`: Narrative Contract와 thin semantic IL을 묶는다.
- `T2`: T1에 전체 linear manuscript를 실제로 읽은 flow attestation을 추가한다.

Narrative Contract는 문장 template이 아니다. throughline, reader journey, Section role·dependency·handoff, continuity term, open-loop/callback, repetition policy를 기록한다. compiler는 이 값을 사용해 transition 문장이나 prose를 만들지 않는다.

## 저장소에 포함되는 것

- `internal/reportilphase0/schemas/`: experimental JSON Schema
- `internal/reportilphase0/`: cross-object validator, Markdown/HTML/PDF compiler, receipt writer
- `cmd/plasma-report-il-phase0/`: archive-local CLI
- unit/adversarial tests

실제 fixture, generated reports, screenshots, manifests, PDF, failure packet은 저장소 밖 `~/research-artifacts/liquid2/plasma/experiments/issue-357-report-il-phase0/` 아래에 둔다.

## 실행

Plasma module 또는 그 하위에서 실행한다.

```sh
go run ./cmd/plasma-report-il-phase0 \
  -archive-root "$HOME/research-artifacts/liquid2/plasma/experiments/issue-357-report-il-phase0" \
  -bundle "$HOME/research-artifacts/liquid2/plasma/experiments/issue-357-report-il-phase0/fixtures/t2.json" \
  -run-id manual-t2 \
  -require-pdf
```

필수 입력:

- `archive-root`: repository 밖 durable archive root
- `bundle`: archive root 안의 T0/T1/T2 JSON bundle
- `run-id`: 재사용하지 않는 safe slug

선택 입력:

- `chrome-path`: PDF에 사용할 Chrome/Chromium executable
- `require-pdf`: PDF 실패 시 blocker packet을 남긴 뒤 command도 실패

## output

각 run은 새 `runs/<run-id>/`에 다음을 저장한다.

- `input-bundle.json`
- `narrative-contract.json` — T1/T2
- `semantic-il.json`
- `flow-attestation.json` — T2
- `report.md`
- `report.html`
- `report.pdf` 또는 `pdf-blocker.json`
- `manifest.json`

manifest는 schema version과 별도로 experimental compiler version, input/output hashes, target media type·byte size, PDF renderer identity, degradation/blocker를 기록한다.

`report.html`은 core content를 build time에 materialize하고 CSS와 image bytes를 포함한다. script와 외부 subresource는 사용하지 않으며 CSP를 문서에 포함한다.

PDF는 Phase 0에서 Chrome print backend 한 종류만 preflight한다. tagged PDF와 outline 생성을 요청하고 생성 파일을 parser로 연다. 이 결과는 PDF/UA conformance나 최종 engine 선택을 의미하지 않는다.

## validation

실행 전에 다음을 fail-closed로 검사한다.

- schema version과 unknown field
- T0/T1/T2의 Narrative/attestation cardinality
- unique node/reference/asset identity
- parent와 rhetorical relation target
- Section role/dependency/transition/open-loop/callback relation
- dependency cycle
- block kind와 payload 일치
- table shape
- asset hash, license, alt, active/external SVG
- 사용되지 않은 asset/reference와 evidence/reference binding
- portable Markdown cross-reference의 explicit fallback/degradation receipt
- coverage target, requirement binding, duplicate requirement identity
- 등록 profile 없는 document extension
- archive/run symlink containment과 immutable run directory
- flow attestation의 document/revision/linear projection hash
- accepted attestation의 unresolved required finding

invalid input은 run directory를 만들지 않는다. target가 exact representation을 제공하지 못하면 `degradation_receipt` 또는 PDF blocker를 남긴다.

## 현재 비목표

- provider가 Narrative Contract나 prose를 작성하게 하는 것
- 기존 report family와 제품 state에 연결하는 것
- 기존 Markdown을 IL로 migration하는 것
- 장문 partial regeneration
- full rhetorical AST
- PDF engine bake-off와 PDF/UA 선언
- UI selector, canary, default 변경

Phase 0 이후에는 T0/T1/T2 frozen packet을 먼저 수작업 whole-read하고, 별도 issue에서 provider-backed paired generation study 여부를 결정한다.
