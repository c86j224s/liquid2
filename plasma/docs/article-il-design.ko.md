# Article IL 설계

상태: Issue #461 설계 기준선. 아직 글 제품은 구현되지 않았다.

## 목적

Plasma는 사용자가 미션 조사에서 실제로 읽을 만한 결과를 만들 수 있어야 한다. 독자는 모든 원천을 먼저 읽지 않고도 새로운 정보, 실제로 쓸 수 있는 방법, 달라진 관점 또는 읽는 재미를 얻어야 한다.

이 목적은 기존 보고서의 말투를 가볍게 바꾸는 것과 다르다. 보고서는 검토, 포괄성, 참조, 의사결정을 우선한다. 글은 독자가 궁금증을 느끼고, 설명을 받고, 이해를 바꾸고, 쓸모 있는 보상에 도달하는 순서를 우선한다. 두 결과 모두 원천에 근거해야 한다.

따라서 `글 만들기`는 `보고서 만들기`와 나란한 제품 capability다. 보고서 mode, report pipeline family, 보고서 rigor profile 또는 말투 preset이 아니다.

## 결정

목표 제품은 별도 Article identity, lifecycle, projection, artifact와 Web surface를 갖는다. 중립적인 source access, provider, workspace, IL, rendering, storage, recovery mechanics는 재사용할 수 있지만, `report.*` 이벤트나 report-run identity를 Article의 alias로 사용하지 않는다.

영구적인 제품 통합 전에는 archive-local prototype gate를 통과해야 한다. Article IL이 현재 Report IL과 최소 독자지향 Report IL 대조군 모두와 구분되는 독서 가치가 있는지 frozen fixture에 한정된 방향성 증거를 먼저 수집한다. 동률이거나 열세면 영구 Article table, event, route, UI를 추가하기 전에 제품화를 중단한다. Pilot은 인과적·통계적 우월성을 입증하지 않으며 frozen fixture 밖으로 일반화하지 않는다.

이 gate는 보고서 말투 옵션으로 돌아가는 것이 아니다. 별도 capability의 영구 lifecycle과 유지 비용을 실제 독서 가치로 정당화하는 절차다.

## 첫 제품 범위

첫 범위는 source-grounded 장편 설명 글 하나다. 짧은 SNS post, 튜토리얼 전용 topology, style preset, 여러 작성자의 산문 fan-out, 자동 게시를 포함하지 않는다.

하나의 provider-session lineage가 모든 prose node와 전체 정보 공개 순서를 유일하게 소유한다. Source 읽기와 사실 검증에는 별도 역할을 사용할 수 있지만 reader-facing prose를 작성하거나 교체할 수 없다. 독립 작성한 Section, paragraph, block 또는 passage를 기계적으로 조립해 최종 글을 만들지 않는다.

사용자는 다음만 입력한다.

- `audience`: 누가 읽으며 무엇을 알고 있을 것으로 보는지
- `reader_promise`: 읽은 뒤 무엇을 이해하거나 할 수 있어야 하는지
- `emphasis`: 특히 살리고 싶은 발견, 관점 또는 방법. 선택 입력

Archive prototype과 첫 제품 범위의 target language는 한국어(`ko`)로 고정한다. 다른 언어가 검증된 제품 계약을 갖기 전까지 language 선택은 미룬다. 중심 질문, throughline, 발견 순서, 결말은 세 사용자 입력과 frozen source를 바탕으로 시스템이 만든다. 첫 범위에서는 model, reasoning, validation profile, topology, language, output format을 Article 사용자 설정으로 노출하지 않는다.

## 제품화 Gate

작업은 두 단계로 진행한다.

### Phase A: archive-local 검증

Phase A는 사용자의 product database, product event namespace, route 또는 UI에 Article 상태를 추가하지 않는다. 전용 archive-local pilot harness가 필요하며, prebuilt Document를 렌더링할 뿐인 기존 Phase 0 CLI는 이 harness가 아니다. Harness는 현재 provider, trace, artifact mechanics를 사용자 runtime DB에 손대지 않고 실행하기 위해 Issue #461 archive 아래의 isolated experiment SQLite database와 request-local tool workspace를 사용할 수 있다. 모든 임시 DB와 workspace를 run manifest에 기록해야 한다.

R arm은 isolated fixture DB에서 변경하지 않은 현재 product path를 실행한다. E와 A arm은 동일한 canonical source bytes와 hash 위에서 명시적인 archive-runner adapter를 사용한다. Harness는 하나의 arm-independent input contract를 freeze하고, 각 arm의 서로 다른 topology, prompt, tool, output contract를 실험 변수로 기록한다. 이 결합된 arm 차이에서 topology 효과만 분리했다고 주장하지 않는다.

재사용 가능한 테스트와 실험 코드는 저장소에 둘 수 있지만, 실제 run, prompt, manuscript, screenshot, isolated database와 raw judgment는 Git 밖의 Issue #461 artifact archive에 보관한다. 세 arm end-to-end 실행, frozen-input hash 동일성, 실패 보존, manifest finalization을 증명하는 harness preflight가 통과하기 전에는 pilot을 시작할 수 없다.

동일한 frozen input으로 세 arm을 비교한다.

- `R`: 현재 채택된 Report IL
- `E`: 대조군으로만 사용하는 최소 독자지향 Report IL 확장
- `A`: archive-local Article IL prototype

pilot은 서로 다른 가치 유형의 fixture 세 개를 사용한다. 하나는 방법/how-to, 하나는 mechanism/insight, 하나는 discovery/context다. 모든 arm은 같은 source catalog, audience, reader promise, emphasis, language, model, reasoning effort와 제한된 resource contract를 받는다.

Protocol은 9개 fixture/arm cell마다 정확히 하나의 run ID를 사전에 배정한다. 각 run은 모든 bounded technical attempt를 기록한다. Manual rerun, replacement manuscript, 추가 semantic attempt 또는 결과를 본 뒤 arm 변경이 생기면 pilot version 전체가 무효다. 유리한 attempt를 선택할 수 없다. PASS에는 usable terminal manifest와 complete audit가 있는 9개 run이 모두 필요하다. Failed/missing arm, input 변경, blinding leak, 불완전한 usage/failure receipt, 분류되지 않은 실패 또는 사용할 수 없는 manuscript가 하나라도 있으면 pilot은 `INCONCLUSIVE`다. 이를 생략하거나 다른 arm의 패배로 취급하거나 PASS로 계산할 수 없다.

Generation 전에 R/E/A identity 각각을 code revision, exact prompt, tool allowlist, schema, topology, provider-call/repair budget, renderer와 output-length contract를 포함한 manifest digest로 고정한다. 직접 제품화 비교는 blinded A/E content lane이며 R은 contextual benchmark라서 투표하지 않는다. 대칭적인 fixture-level 규칙에서 A win은 A와 E가 matching input hash, complete audit, 중대한 factual/source defect 0건을 갖고, 사용자의 blinded A/E 선택이 A를 선호하며, fixture memory 또는 transfer task에서 A가 E보다 나쁘지 않고, 사전 정의한 세 reading dimension 중 최소 두 개에서 A를 선호하며 어느 dimension도 E를 선호하지 않을 때만 성립한다. Pilot PASS에는 최소 두 개의 A fixture win, E win 0개, 세 번째 fixture의 유효한 tie 또는 A win이 필요하다. `INCONCLUSIVE` fixture가 하나라도 있으면 전체 pilot도 `INCONCLUSIVE`다. E는 label을 바꾼 같은 규칙을 사용한다. Mixed result는 재량으로 PASS가 될 수 없다. Protocol에 이름이 있는 모든 Luna panel seat가 complete judgment를 반환하지 않으면 fixture는 `INCONCLUSIVE`다. 모든 judgment와 contradiction은 전체 독자 집단의 사람 증거가 아닌 방향성 증거로 보존한다.

모든 run receipt, audit, blind judgment가 끝난 뒤 harness는 `analysis/decision.candidate.json`과 SHA-256을 만든다. 그 digest를 Issue #461에 게시한다. 이후 repository owner인 사용자의 댓글은 exact digest를 명시하고 제품화를 명시적으로 승인하거나 거절해야 한다. Pre-run, generic, digest-free comment는 무효다. 최종 `analysis/decision.lock.json`은 candidate digest, decision comment URL/ID, author, timestamp, 9개 run ID와 rule version을 binding하고, 자체 digest도 이후 Issue #461 댓글에 게시한다. Archive를 write-once storage로 신뢰하지 않는다. Phase B preflight는 모든 file/record digest를 다시 계산하여 두 GitHub 댓글과 일치하는지 검증한다. Record outcome이 `PASS`이고 모든 binding이 검증되기 전에는 Phase B를 시작할 수 없다. 사용자의 판단은 프로젝트 decision authority이지 broader target audience가 동의한다는 증거가 아니다.

Pilot은 즉시 반응과 표현된 의향을 기록한다. 장기 기억, 실제 공유 행동, 인과적·통계적 우월성 또는 전체 독자 집단의 효과를 주장하지 않는다.

Pilot이 통과하면 더 큰 follow-up 비교를 제안할 수 있다. pilot이 존재한다는 이유만으로 자동 실행하지 않는다. Locked rule에서 Article IL이 최소 Report 확장과 동률이거나 열세면 영구 Article capability를 구현하지 않는다.

### Phase B: 제품 capability

Phase A 통과 후에만 Article 전용 durable event, projection, run membership, route, UI, retry, delete와 release 동작을 추가한다.

## Article Intent

승인된 요청은 immutable Article Intent revision을 만든다. 제품 통합 시에는 provider 작업을 시작하기 전에 intent revision과 최초 pending attempt를 원자적으로 기록해야 한다.

Durable meaning은 작게 유지한다.

```text
ArticleIntent
  audience
  reader_promise
  emphasis?
  language
```

제목은 작성자가 소유한다. source ID, source body, model setting, validation profile, output choice, central question, throughline, pipeline configuration은 Article Intent에 들어가지 않는다.

retry는 같은 Intent hash를 재사용한다. 독자 약속을 바꾸면 기존 attempt의 의미를 변경하지 않고 새 Intent revision과 새 run을 만든다.

## Article Narrative

Article Narrative는 얇은 authoring contract다. prose template도 아니고, 좋은 글임을 형식적으로 증명하는 체계도 아니다.

다음을 기록한다.

- 독자의 출발 상태
- 읽은 뒤 의도한 변화
- 중심 질문과 throughline
- 순서가 있는 소수의 movement
- 각 movement의 독자-facing 역할과 구체적인 보상
- 다음 movement가 필요한 이유
- 자료가 허용할 때만 쓰는 선택적 질문과 뒤의 callback
- 결말의 의무
- 의도적 제외 범위

movement는 neutral document node의 연속된 범위에 대응한다. paragraph보다 크고, 렌더된 Section과 반드시 일치하지는 않는다. schema는 scene, question, surprise, reversal, loop 또는 Section 개수를 강제하지 않는다. 모든 paragraph에 payoff record를 요구하지도 않는다.

사실을 담은 leaf는 기존의 검증된 source-span receipt와 binding mechanics를 사용한다. 이 receipt는 Article/Report IL authoring contract이며 Plasma의 legacy Evidence 또는 Claim record가 아니다. Article Narrative가 두 번째 claim/source 모델을 중복하거나 legacy Evidence gate를 다시 도입하면 안 된다. Compiler는 Narrative metadata를 prose로 바꾸지 않는다.

## Prose 이전의 Evidence

Article 작성자는 frozen source catalog와 server-verified editorial input을 사용한다. 이 입력은 prose로 압축되기 전에 연결된 사실, 관계, 방법, 수치, 날짜, 불확실성, exact source anchor를 보존한다.

재사용할 mechanics는 다음과 같다.

- opaque source identity와 catalog hash
- bounded, forward-only, UTF-8-safe read
- 서버가 복원하는 exact quote receipt
- content-free trace와 failure payload
- revisioned server-owned workspace
- finalize 전 전체 재독
- mission, media type, hash, byte size, producer에 결합된 typed artifact

기존 Report IL `EditorialMemory` 정책을 그대로 Article 정책으로 일반화하지 않는다. Workspace와 anchor mechanics는 유용하지만, report-oriented account checklist는 Article dossier 설계의 입력일 뿐 generic truth가 아니다. Prototype dossier는 factual event, relationship, method/procedure, quantitative result, condition/caveat, conflict, uncertainty의 closed account-kind set을 정의한다. 각 kind는 필수 typed element를 가지며 모든 element가 하나 이상의 exact source-span receipt에 binding된다. 서버는 account/element/anchor coverage를 검증하고 누락된 필수 element를 거부한다. 이는 frozen fixture truth ledger에 대한 structural completeness를 증명하지만 provider의 semantic account가 보편적으로 완전함을 증명하지 않는다. Factual audit과 사용자 독서는 여전히 필요하다.

Source body를 expanded prompt packet에 복사하지 않는다. Dossier artifact는 source tool로 등록한 bounded exact source excerpt를 포함할 수 있고 downstream role은 bounded dossier/source tool을 통해서만 그 excerpt를 읽을 수 있다. Excerpt bytes는 하나의 storage owner만 가지며 claim inventory, audit record, ledger event, manifest, 일반 cache, export, ordinary artifact preview 또는 administrative/read-tool response로 복사하지 않는다. 이 표면에는 opaque span ID, source key, range, hash, access receipt와 closed verdict만 남긴다. Manifest는 dossier artifact를 sensitive archive content로 분류하고 identity/hash만 기록하며 excerpt 자체를 담지 않는다. Trace, failure, public summary와 Git에도 그 bytes를 넣지 않는다.

Source-grounded editorial dossier는 source-bound account의 projection이자 내부 authoring artifact이며 Source가 아니다. 모든 factual account는 accepted mission Source snapshot 또는 accepted live-reference observation과 exact source-span receipt를 가리켜야 한다. 그 Source는 외부 원본 자료 또는 사용자가 제공한 원본 bytes까지 이어지는 검증된 provenance를 유지해야 하며, 상태 label이나 사용자 승인만으로 original provenance를 만들 수 없다. Agent result, saved knowledge, report, Article 또는 다른 생성 artifact에서 유래한 content는 Source snapshot으로 감싼다고 factual authority가 되지 않는다. 일반 Source contract에 따라 별도의 original material로 제공되고 검증되어야 한다. Staged candidate, raw artifact, connector search result, agent result 또는 saved knowledge item만으로는 binding을 충족할 수 없다. Mission result는 angle이나 hypothesis를 안내할 수 있지만 Source나 factual authority가 되지 않는다.

기존 source key와 source-span receipt는 identity, reading, range, lineage를 증명하지만 claim이 인용 source span에서 실제로 성립하는지 자체를 증명하지 않는다. 따라서 Article prototype은 완전한 factual-claim inventory를 추가한다. Accepted Article IL에서 어떤 target에든 나타날 수 있는 모든 authored string이 inventory에 등장한다. 여기에는 title, standfirst, prose, list item, quotation, table caption/header/cell, code annotation, equation caption, figure caption과 alt text, link label, disclosure text, conditional/accessibility rendering이 포함된다. Authored content가 없는 deterministic renderer-owned interface label만 제외한다. 각 inventoried string은 모든 factual claim의 exact text와 hash, document node와 field, source-span receipt, support relation, uncertainty와 claim-strength class를 나열하거나, auditor가 완전히 검토한 string에 factual claim이 없다는 verdict를 제출해야 한다. Author가 nonfactual label을 붙여 leaf를 제외할 수 없다. Direct quotation은 canonical source bytes와 exact match해야 한다. Server receipt는 모든 reader-facing leaf와 listed claim이 audit packet에 포함됐고 모든 span/hash가 frozen catalog에 속함을 증명한다.

Factual auditor는 모든 leaf, claim inventory entry와 bound source span을 읽고 claim별 closed verdict와 leaf-coverage verdict를 제출해야 한다. Claim verdict는 directly supported, bounded inference, overstated, contradicted, unsupported 중 하나다. 누락된 entry, 검토하지 않은 leaf나 claim, exact하지 않은 quotation은 hard failure다.

Severity mapping은 closed contract다. `contradicted`, `unsupported`, fabricated quotation/scene, 지원되지 않은 identity/number/date/condition/causation, material caveat 또는 uncertainty 손실은 최소 H1이다. 중심 thesis, 핵심 how-to, safety 또는 material decision을 뒤집으면 H0다. `overstated` claim이 conclusion, action 또는 reader understanding을 materially 바꾸면 H1이며, 그렇지 않으면 repairable이지만 correction 없이는 진행할 수 없다. `bounded inference`는 prose가 inference임을 드러내고 cited span이 그 bounded reading을 합리적으로 지지할 때만 통과한다. 모든 severity decision은 claim ID, rule code, auditor verdict와 source-span receipt에 binding된다. A candidate에 H0/H1이 하나라도 있으면 `A_HARD_FAIL`이다.

Mechanical validation은 inventory, span/hash/catalog, coverage와 verdict binding integrity를 증명하지만 semantic entailment를 증명한다고 주장하지 않는다. 독립 audit과 이후 사용자 독서는 여전히 필요한 한계이며 manifest는 auditor/model lineage를 기록한다.

## Prototype Topology

archive-local prototype은 다음 고정 topology를 사용한다.

```text
frozen inputs
  -> source-grounded editorial dossier
  -> Article Narrative + 한 명의 주 작성자
  -> immutable Article IL revision
  -> reader diagnosis ----\
  -> factual audit --------+-> 조건부 author repair 최대 1회
  -> confirmation reads ---/
  -> accepted Article IL
  -> Markdown + reading-first self-contained HTML + PDF
  -> archive manifest와 receipt
```

Article Narrative와 전체 manuscript는 author receipt에 기록된 하나의 exact author provider-session lineage가 소유한다. Reader와 factual-audit 역할은 독립된 fresh session과 read-only tool만 사용하고 prose를 제출할 수 없다. Revision과 node에 결합된 finding만 반환한다.

Repair가 필요 없으면 최초 author revision이 바로 다음 단계로 진행한다. Repair가 필요하면 runner가 reader와 factual-audit finding을 모두 모은 뒤 하나의 repair batch를 연다. 이 batch는 기록된 exact author session을 resume하고 검증된 session lineage를 보존해야 한다. Continuation을 사용할 수 없거나 다른 lineage가 반환되면 새 author를 붙이지 않고 attempt를 실패시킨다. 새 revision은 다시 전체 읽기·감사한다. 각 attempt는 repair count 0 또는 1을 기록하며 confirmation finding이 남으면 terminal이다. 두 번째 semantic repair loop는 없으며 어떤 report writer로도 fallback하지 않는다.

## 편집 경계

첫 repair contract는 국소적인 독서 결함을 고치는 장치이지, 근본적으로 잘못 기획된 Article을 구제하는 장치가 아니다.

Prototype v1은 기존 prose node 안의 exact once-only local text replacement만 허용한다. 하나의 atomic batch는 최대 6개 replacement와 총 4 KiB의 replacement text만 가질 수 있다. Exact old/new UTF-8 value는 각각 최대 1 KiB이며 replacement span은 base node Unicode code point의 최대 25%만 포함할 수 있다. 완성된 post-edit node의 code-point 길이는 base node의 ±15% 안에 있어야 하며, base node와의 rune-level Levenshtein distance는 base 길이의 25%를 넘을 수 없다. 하나의 node는 한 번만 대상이 된다. Replacement는 node를 추가·삭제하거나, node 순서와 movement mapping을 바꾸거나, source/evidence reference를 변경하거나, node 전체·거의 전체 또는 원고 전체를 교체할 수 없다.

모든 replacement는 exact old/new UTF-8 text를 기록하고, base Article revision, canonical Markdown projection bytes의 SHA-256, frozen source-catalog hash, node ID, expected pre-edit node digest와 변경되지 않은 evidence binding에 결합된다. Operation은 서로 다른 node만 대상으로 하며 canonical document-node 순서로 제출해야 한다. 중복 대상과 겹치는 replacement는 거부한다. 서버는 그 순서대로 적용하고 exact operation list, post-edit node digest, accepted Article IL digest와 canonical Markdown digest를 기록하여 accepted revision을 byte-for-byte로 replay하고 감사할 수 있게 한다.

이 기계적 검사는 operation과 lineage integrity를 증명하지만 semantic equivalence를 증명하지 않는다. 의미 보존은 독립 factual audit의 hard gate다. Factual audit은 stale catalog, 없거나 여러 번 나타나는 old text, byte/count budget을 넘는 operation, factual meaning, claim strength, causation, quotation, number, date, name, condition, caveat 또는 uncertainty의 변화를 거부한다. Opening과 conclusion도 이 요구사항의 예외가 아니다.

Node 이동, 구조 operation으로서의 반복 삭제, split, merge, cross-movement reorder와 node 삭제는 prototype v1 범위 밖이다. 이후 prototype이 독서 가치와 결정론적 movement·source·lineage 검증을 증명한 뒤에만 추가할 수 있다. 전체 정보 공개 순서가 잘못됐다면 editor가 몰래 재구성하지 않고 attempt를 실패시키며, 다음 실험에서 Article Narrative나 author contract를 개선한다.

## 검증 계층

검증은 세 계층으로 나눈다.

### Hard failure

독자 선호와 무관하게 accepted Article을 막는다.

- frozen source input 변경 또는 사용 불가
- source가 뒷받침하지 않는 factual content
- 창작된 scene, 감정, quote, cause, name, date, number 또는 condition
- 중요한 caveat, uncertainty 또는 claim-strength 손실
- source binding, revision, node, artifact, privacy 또는 lineage 실패
- malformed/unfinalized workspace
- silent fallback 또는 report reclassification

### Repairable finding

Node-local opening, clarity, transition, repetition, terminology 또는 conclusion 문제만 한 번의 bounded repair 대상으로 삼을 수 있다. Mechanical validation은 exact Article/source lineage를 보존해야 하며, 새 revision이 진행하려면 독립 factual audit이 결과의 meaning과 source support를 승인해야 한다.

### Human/advisory judgment

재미, 즉시 기억, 표현된 공유 욕구, 전체 voice, 실제로 읽을 가치가 있는지는 whole-read 판단으로 남긴다. 하나의 자동 점수로 환원하지 않는다. 자동 reader는 방향성 증거와 모순을 제공할 수 있지만, 사용자의 독서는 프로젝트 go/no-go 판단이지 전체 독자 집단의 효과, 장기 기억 또는 실제 공유 행동의 증거가 아니다.

## 출력 계약

하나의 accepted Article IL revision에서 다음을 만든다.

- 내부 intermediate `article-il.json`
- canonical portable text artifact `article.md`
- reading-first self-contained artifact `article.html`
- derivative `article.pdf`
- manifest와 content-free review receipt

HTML 첫 viewport에는 title, 필요하면 짧은 standfirst, substantive prose가 나온다. summary card, TOC, relationship map, pipeline graph, model setting, artifact lineage 또는 source inventory로 시작하지 않는다. reference와 provenance는 제공하되 독서보다 앞세우지 않는다.

같은 accepted IL bytes에서 Markdown과 HTML은 deterministic해야 한다. PDF는 같은 IL과 renderer receipt에 binding하지만 byte-identical 출력을 전제하지 않는다. 현재 archive `Run`은 PDF renderer identity를 기록하지만 현재 product manifest는 이를 버리므로, Article prototype harness는 현재 제품 출력에 이미 있다고 가정하지 않고 Chrome product/revision을 자체 manifest에 보존해야 한다. Equation과 Mermaid는 외부 subresource 없이 self-contained 상태를 유지하면서 embedded JavaScript를 필요로 할 수 있다.

## Gate 통과 후 제품 아키텍처

최종 package 이름은 이 문서의 그림이 아니라 실제 소유권을 따른다. 제품 통합은 최소한 다음 경계를 보존해야 한다.

- Article이 Intent, Narrative, lifecycle, progress, artifact lineage, UI meaning을 소유한다.
- 작은 neutral document/compiler seam은 Report IL output behavior를 characterization하고 실제 소비자 둘이 필요로 한 뒤에만 추출한다.
- Article-owned consumer port가 source access, provider execution, workspace finalization, durable storage를 concrete adapter에서 분리한다.
- Article package가 Report policy, Report workflow, Report event 또는 report-run identity를 import하지 않는다.
- Report package가 Article을 fallback으로 import하지 않는다.
- SQLite child repository는 root-only implementation detail로 유지한다.

깨끗한 그림을 얻기 위해 처음부터 광범위한 `agentexec`, source access 또는 compiler 재구성을 하지 않는다. 영구 abstraction을 만들기 전에 archive prototype으로 제품을 증명한다.

## Gate 통과 후 Durable Lifecycle

제품 구현은 별도 `article.*` event namespace와 logical Article-run membership을 사용한다. lifecycle pattern은 재사용하지만 report event 이름은 재사용하지 않는다.

다음 동작이 필요하다.

- `article_run_id`가 immutable input facts를 소유하고, 각 execution attempt는 별도 identity와 parent/origin lineage를 가짐
- input-facts SHA-256이 exact Intent revision, source catalog와 selection, 허용된 editorial-input artifact, provider/model/tool/schema/budget/output contract를 binding
- initial create는 mission+user scope에서 unique한 client request key를 받고 canonical request fingerprint와 함께 저장함. 같은 key와 같은 입력의 replay는 원래 Intent/run을 반환하고, 다른 입력으로 같은 key를 쓰면 conflict
- provider work 전 Intent, initial pending attempt, request-key record와 turn/workflow/report/Article의 mission-level 중복 실행 방지를 원자적으로 commit
- 현재 one-runner-per-database 전제 아래 process-local cancel은 즉시 signal이지만 durable `article.attempt.canceled`가 terminal state
- success, failure, canceled terminal은 하나의 conditional boundary를 사용해 정확히 하나만 승리하고 canceled terminal은 recovery 진행을 막음
- 비용이 큰 stage가 완전히 finalize·validate된 뒤에만 run input-facts hash에 binding된 content-addressed checkpoint 생성
- `resume_failed`는 source attempt가 정확히 하나의 failed terminal을 가질 때만 가능하며 새 pending attempt를 만들기 전에 intent/input/checkpoint hash를 검증. Canceled, completed, purged, ambiguous attempt는 대상이 될 수 없음
- restart attempt는 audit을 위해 run/origin/parent lineage를 유지하지만 hard traversal boundary를 표시하고, 하나의 checkpoint walker가 그 node에서 멈춰 ancestor를 검색하지 않음
- exact-author repair continuation이 중단되면 terminal. Post-repair confirmation failure는 `repair_exhausted`를 기록하고 resume을 허용하지 않음
- 하나의 Article run은 명시적인 user-requested complete restart를 최대 1회만 허용하며, 새 attempt와 새 author를 사용하고 과거 checkpoint를 재사용하지 않음. 두 번째 전체 재작성에는 새 Intent revision과 새 Article run이 필요하여 restart가 무제한 best-of-N author search가 될 수 없음
- 하나의 SQLite transaction이 모든 final artifact byte, manifest, Article-run membership과 success terminal을 저장하며 rollback은 아무것도 노출하지 않음
- content-free typed failure와 truthful usage receipt
- revision/facts-hash delete preview와 content-free tombstone. Preview와 delete는 open attempt와 process-local owner가 없는 terminal run에만 허용. Delete는 하나의 transaction에서 revision/facts를 재검증하고 tombstone 기록과 소유 event/artifact membership purge를 수행하며, 같은 conditional boundary가 tombstone 이후 늦게 도착한 worker terminal을 거부

Durable worker lease, heartbeat, fencing, multi-process scheduling, 범용 execution-kernel 재작성은 첫 범위 밖이다.

## Gate 통과 후 Web Surface

Plasma는 `리포트` 옆에 별도 `글` 탭을 추가한다.

기본 form은 세 가지 Article Intent 입력만 묻는다. Progress는 사용자 언어로 표현한다.

1. 자료 읽기
2. 독자가 따라갈 흐름 찾기
3. 글쓰기
4. 전체 글 읽고 다듬기
5. 읽기용 결과 준비

완료 card는 title, reader promise, 읽기, Markdown/HTML/PDF 다운로드를 앞세운다. Pipeline stage, artifact ID, hash, source selection, lineage는 secondary operator/provenance detail에 둔다.

Article이 실패하면 사용자는 가능한 경우 verified checkpoint에서 재시도, frozen input에서 처음부터 다시 시작, reader promise를 새 Intent revision과 run으로 수정, 또는 attempt 중단 중 다음 행동을 명시적으로 선택한다. Failed attempt를 completed report로 바꾸지 않는다. 이후 사용자 acceptance action은 Issue #461의 제품 gate로 유지하며 자동 reader stage 안에 숨기지 않는다.

직접 사용자 편집은 redpen의 interaction idea를 참고할 수 있지만 report-specific route, event, workcopy identity 또는 delete semantics를 재사용하면 안 된다. Archive prototype과 첫 생성 Article에는 필요하지 않다.

## 명시적 비범위

- 기존 Report path 교체 또는 rename
- Article을 `report_mode`, `reportpipeline`, rigor/tone selector에 추가
- Report-to-Article 또는 Article-to-Report fallback
- 짧은 SNS post, 플랫폼별 copy, 자동 게시
- 첫 범위의 tutorial 전용 topology
- 여러 style preset 또는 실제 작가 문체 모사
- 독립 작성한 prose Section
- Article의 자동 source 등록
- browser raw IL 편집
- 창작된 drama 또는 필수 narrative curve
- 하나의 자동 fun/engagement score
- 광범위한 cross-product infrastructure rewrite
- 별도 release 결정 전 public snapshot, tag 또는 release

## 이슈와 릴리스 규칙

Issue #461 하나가 설계, 실험, 구현, 검증, 사용자 review를 소유한다. 서브이슈는 만들지 않는다. 여러 짧은 PR과 experiment run은 허용하지만 이 이슈의 checklist와 comment lineage 안에서 관리한다.

사용자가 candidate를 읽고 독립 가치를 확인하고, internal `main` runtime에서 최종 동작을 승인하기 전에는 `issue:completed`, `release:ready` 또는 close로 진행하지 않는다.
