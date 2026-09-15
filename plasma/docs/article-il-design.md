# Article IL Design

Status: Issue #461 design baseline; the Article product is not implemented yet.

## Purpose

Plasma should let a user turn mission research into something worth reading. The
reader should gain new information, a usable method, a changed perspective, or
enjoyment without having to read every source first.

This is different from making an existing report sound casual. Reports optimize
for review, coverage, reference, and decision support. Articles optimize the
order in which a reader becomes curious, receives an explanation, changes their
understanding, and reaches a useful payoff. Both must remain source-grounded.

`글 만들기` is therefore a product capability beside `보고서 만들기`, not a
report mode, report pipeline family, rigor profile, or tone preset.

## Decision

The intended product has its own Article identity, lifecycle, projection,
artifacts, and Web surface. It may reuse neutral source-access, provider,
workspace, IL, rendering, storage, and recovery mechanics, but it must not reuse
`report.*` events or report-run identity as aliases.

Permanent product integration is gated on an archive-local prototype. Plasma
will first collect fixture-bound directional evidence on whether Article IL has
reader value beyond both the current Report IL and a minimally reader-oriented
Report IL control. A tie or loss stops productization before permanent Article
tables, events, routes, or UI are added. The pilot does not establish causal or
statistical superiority and must not be generalized beyond its frozen fixtures.

This gate is not a retreat to a report tone option. It tests whether the
separate capability is worth its permanent lifecycle and maintenance cost.

## First Slice

The first slice is one source-grounded, long-form explanatory article. It does
not include a short social post, a tutorial-specific topology, style presets,
multi-author prose fan-out, or automatic publishing.

One and only one provider-session lineage owns every prose node and the complete
disclosure order. Source reading and factual checks may use separate roles, but
they cannot author or replace any reader-facing prose. No independently authored
Section, paragraph, block, or passage may be mechanically assembled into the
final Article.

The user supplies only:

- `audience`: who the Article is for and what they likely know;
- `reader_promise`: what they should understand or be able to do after reading;
- `emphasis`: an optional discovery, perspective, or method to preserve.

The archive prototype and first product slice write in Korean (`ko`). Language
selection is deferred until another language has a tested product contract. The
system derives the central question, throughline, discovery order, and ending
from the three user inputs and the frozen sources. Model, reasoning, validation
profile, topology, language, and output-format controls are not user-facing
Article settings in the first slice.

## Productization Gate

The work proceeds in two phases.

### Phase A: archive-local proof

Phase A adds no Article state to the user's product database, product event
namespace, routes, or UI. It requires a dedicated archive-local pilot harness;
the existing Phase 0 CLI only renders a prebuilt Document and is not this
harness. The harness may use an isolated experiment SQLite database and
request-local tool workspaces under the Issue #461 archive so current provider,
trace, and artifact mechanics can run without touching the user's runtime DB.
It must record every temporary DB and workspace in the run manifest.

The R arm runs the unchanged current product path against an isolated fixture DB.
The E and A arms use explicit archive-runner adapters over the same canonical
source bytes and hashes. The harness freezes one arm-independent input contract,
then records each arm's different topology, prompt, tool, and output contracts as
experimental variables. It must not claim that topology effects are isolated
from those coupled arm differences.

Reusable test and experiment code may enter Git, but runs, prompts, manuscripts,
screenshots, isolated databases, and raw judgments remain under the Issue #461
artifact archive outside Git. The pilot cannot start until the harness has an
end-to-end preflight proving all three arms, frozen-input hash equality, failure
retention, and manifest finalization.

It compares three arms over identical frozen inputs:

- `R`: the current accepted Report IL path;
- `E`: a minimal reader-oriented Report IL extension used only as a control;
- `A`: the archive-local Article IL prototype.

The pilot uses three fixtures: one method/how-to subject, one
mechanism/insight subject, and one discovery/context subject. Each arm receives
the same source catalog, audience, reader promise, emphasis, language, model,
reasoning effort, and bounded resource contract.

The protocol preassigns exactly one run ID to each of the nine fixture/arm
cells. Each run records every bounded technical attempt. A manual rerun,
replacement manuscript, extra semantic attempt, or post-result arm change
invalidates the complete pilot version; investigators cannot select a favorable
attempt. PASS requires nine usable terminal manifests and complete audits.
Any failed or missing arm, changed input, blinding leak, incomplete usage/failure
receipt, unclassified failure, or unusable manuscript makes the pilot
`INCONCLUSIVE`; it cannot be omitted, treated as another arm's loss, or counted
as a pass.

Before generation, each R/E/A identity is fixed by a manifest digest covering
code revision, exact prompt, tool allowlist, schema, topology, provider-call and
repair budget, renderer, and output-length contract. The direct productization
comparison is the blinded A/E content lane; R is contextual and cannot vote.
The symmetric per-fixture rule defines an A win only when both A and E have
matching input hashes, complete audits, no material factual/source defect, the
user's blinded A/E choice prefers A, A is not worse on the fixture memory or
transfer task, and A is preferred on at least two of three predeclared reading
dimensions with none preferring E. A passes the pilot only with at least two A
fixture wins, no E win, and a valid tie or A win in the third fixture. Any
`INCONCLUSIVE` fixture makes the complete pilot `INCONCLUSIVE`. E uses the same
rule with labels reversed. Any mixed result is not a discretionary PASS. Every Luna
panel seat named by the protocol must return a complete judgment or the fixture
is `INCONCLUSIVE`; all judgments and contradictions remain directional evidence,
not population-level human evidence.

After all run receipts, audits, and blind judgments are complete, the harness
writes `analysis/decision.candidate.json` and its SHA-256. That digest is posted
to Issue #461. The user's later repository-owner comment must name the exact
digest and explicitly accept or reject productization. A pre-run, generic, or
digest-free comment is invalid. The final `analysis/decision.lock.json` binds the
candidate digest, decision comment URL/ID, author, timestamp, all nine run IDs,
and the rule version; its own digest is posted in a later Issue #461 comment.
The archive is not trusted as write-once storage: Phase B preflight recomputes all
file and record digests and requires them to match both GitHub comments. Phase B
cannot begin unless the record outcome is `PASS` and every binding verifies. The
user's judgment is project decision authority, not evidence that a broader
target audience agrees.

The pilot records immediate responses and stated intentions. It does not claim
long-term memory, actual sharing behavior, causal or statistical superiority,
or population-level audience effects.

If the pilot passes, a larger follow-up comparison may be proposed. It must not
run merely because the pilot exists. If Article IL ties or loses to the minimal
Report extension under the locked rule, the permanent Article capability is not
implemented.

### Phase B: product capability

Only after Phase A passes may Plasma add Article-specific durable events,
projections, run membership, routes, UI, retry, deletion, and release behavior.

## Article Intent

An accepted request creates an immutable Article Intent revision. During
product integration, the intent revision and initial pending attempt must be
recorded atomically before provider work starts.

The durable meaning is deliberately small:

```text
ArticleIntent
  audience
  reader_promise
  emphasis?
  language
```

The title belongs to the author. Source IDs, source bodies, model settings,
validation profiles, output choices, central question, throughline, and
pipeline configuration do not belong in Article Intent.

A retry reuses the exact Intent hash. Changing the promise creates a new Intent
revision and a new run rather than mutating the meaning of an existing attempt.

## Article Narrative

Article Narrative is a thin authoring contract, not a prose template and not a
formal proof of good writing.

It records:

- the reader's starting state;
- the intended change after reading;
- the central question and throughline;
- a small ordered set of movements;
- each movement's reader-facing job and concrete payoff;
- why the next movement is needed;
- optional questions and later callbacks when the material warrants them;
- the ending obligation;
- deliberate exclusions.

A movement maps to one contiguous range of neutral document nodes. It is larger
than a paragraph and need not match a rendered Section. The schema does not
require a number of scenes, questions, surprises, reversals, loops, or Sections.
It does not assign a payoff record to every paragraph.

Factual leaves continue to use the existing verified source-span receipt and
binding mechanics. These receipts are Article/Report IL authoring contracts, not
Plasma's legacy Evidence or Claim records. Article Narrative must not duplicate
a second claim/source model or reintroduce a legacy Evidence gate. The compiler
never turns Narrative metadata into prose.

## Evidence Before Prose

The Article author consumes a frozen source catalog and a server-verified
editorial input that preserves connected facts, relationships, methods,
numbers, dates, uncertainty, and exact source anchors before prose compression.

The reusable mechanics are:

- opaque source identities and a catalog hash;
- bounded, forward-only, UTF-8-safe reads;
- exact quote receipts reconstructed by the server;
- content-free traces and failure payloads;
- revisioned server-owned workspaces;
- complete reread before finalization;
- typed artifacts bound by mission, media type, hash, byte size, and producer.

The existing Report IL `EditorialMemory` policy is not automatically the Article
policy. Its workspace and anchor mechanics are useful; its report-oriented
account checklist is an input to the Article dossier design, not a generic truth.
The prototype dossier defines a closed account-kind set for factual event,
relationship, method/procedure, quantitative result, condition/caveat, conflict,
and uncertainty. Each kind has required typed elements and every element binds
to one or more exact source-span receipts. The server validates account/element/
anchor coverage and rejects omitted required elements. This proves structural
completeness against the frozen fixture truth ledger, not that the provider's
semantic account is universally complete; factual audit and user reading remain
necessary.

Source bodies are not copied into expanded prompt packets. The dossier artifact
may contain bounded exact source excerpts registered through the source tool, and
downstream roles may read those excerpts only through bounded dossier/source
tools. Those excerpt bytes have one storage owner and are never copied into
claim inventories, audit records, ledger events, manifests, general caches,
exports, ordinary artifact previews, or administrative/read-tool responses.
Those surfaces retain only opaque span IDs, source keys, ranges, hashes, access
receipts, and closed verdicts. The manifest classifies the dossier artifact as
sensitive archive content and records its identity/hash, not its excerpts;
traces, failures, public summaries, and Git never contain the bytes.

The source-grounded editorial dossier is an internal authoring artifact and a
projection of source-bound accounts, not a Source. Every factual account in it
must point to an accepted mission Source snapshot or an accepted live-reference
observation and exact source-span receipts. That Source must retain verified
provenance to original external material or bytes supplied by the user; a status
label or user approval alone cannot manufacture original provenance. Content
derived from an agent result, saved knowledge, report, Article, or other generated
artifact cannot become factual authority by being wrapped in a Source snapshot.
It must be separately supplied and verified as original material under the normal
Source contract. A staged candidate, raw artifact, connector search result,
agent result, or saved knowledge item cannot satisfy the binding by itself. A
mission result may guide an angle or hypothesis, but it does not become a Source
or factual authority.

Existing source keys and source-span receipts prove identity, reading, range,
and lineage; they do not by themselves prove that a claim follows from the
quoted span. The Article prototype therefore adds a complete factual-claim
inventory. Every authored string in the accepted Article IL that can appear in any target
appears in the inventory: title, standfirst, prose, list item, quotation, table
caption/header/cell, code annotation, equation caption, figure caption and alt
text, link label, disclosure text, and any conditional or accessible rendering.
Only deterministic renderer-owned interface labels with no authored content are
excluded. Each inventoried string either lists every factual claim's exact text
and hash, document node and field, source-span receipts, support relation,
uncertainty, and claim-strength class, or receives an auditor verdict that the
fully reviewed string contains no factual claim. The author cannot exempt a leaf by labeling it nonfactual. Direct
quotations must match exact canonical source bytes. A server receipt proves that
every reader-facing leaf and every listed claim was included in the audit packet
and that every cited span/hash belongs to the frozen catalog.

The factual auditor must read every leaf, every claim inventory entry, and every
bound source span and submit one closed verdict per claim plus a leaf-coverage
verdict. Claim verdicts are directly supported, bounded inference, overstated,
contradicted, or unsupported. Missing entries, unreviewed leaves or claims, and
non-exact quotations are hard failures.

The severity mapping is closed. `contradicted` and `unsupported`, fabricated
quotation or scene, unsupported identity/number/date/condition/causation, and
material caveat or uncertainty loss are at least H1. They are H0 when they
reverse the central thesis, core how-to, safety, or material decision. An
`overstated` claim is H1 when it materially changes a conclusion, action, or
reader understanding; otherwise it is repairable and still cannot advance
without correction. `bounded inference` passes only when the prose marks it as
inference and the cited spans reasonably support that bounded reading. Every
severity decision binds the claim ID, rule code, auditor verdict, and source-span
receipts. Any H0/H1 in an A candidate yields `A_HARD_FAIL`.

Mechanical validation proves inventory, span/hash/catalog, coverage, and verdict
binding integrity; it does not claim to prove semantic entailment. The
independent audit and the user's later reading remain necessary limitations, and
the manifest records the auditor/model lineage.

## Prototype Topology

The archive-local prototype uses this fixed topology:

```text
frozen inputs
  -> source-grounded editorial dossier
  -> Article Narrative + one dominant author
  -> immutable Article IL revision
  -> reader diagnosis ----\
  -> factual audit --------+-> conditional author repair (at most once)
  -> confirmation reads ---/
  -> accepted Article IL
  -> Markdown + reading-first self-contained HTML + PDF
  -> archive manifest and receipts
```

The Article Narrative and complete manuscript are owned by one exact author
provider-session lineage recorded in the author receipt. Reader and factual-audit
roles use independent fresh sessions, have read-only tools, and cannot submit
prose. They return revision- and node-bound findings.

If no repair is needed, the original author revision advances directly. If a
repair is needed, the runner collects all reader and factual-audit findings
before opening one repair batch. That batch must resume the exact recorded author
session and preserve its validated session lineage. If continuation is
unavailable or returns a different lineage, the attempt fails rather than
assigning a new author. The repaired revision is read and audited again. Each
attempt records a repair count of zero or one; any remaining confirmation
finding is terminal. There is no second semantic repair loop and no fallback to
any report writer.

## Editing Boundary

The first repair contract exists to fix local reading defects, not to rescue a
fundamentally misplanned Article.

Prototype v1 permits only an exact, once-only local text replacement inside an
existing prose node. One atomic batch may contain at most six replacements and
at most 4 KiB of replacement text in total. Each exact old and new UTF-8 value
is at most 1 KiB, and the replaced span may cover at most 25% of the base node's
Unicode code points. The complete post-edit node must remain within 15% of the
base node's code-point length, and its rune-level Levenshtein distance from the
base node may not exceed 25% of the base length. A node may be targeted once. A
replacement cannot insert or remove a node, change node order or movement
mapping, alter source/evidence references, replace a complete or near-complete
node, or replace the complete manuscript.

Every replacement records exact old and new UTF-8 text and is bound to the base
Article revision, the SHA-256 of the canonical Markdown projection bytes, the
frozen source-catalog hash, node ID, expected pre-edit node digest, and unchanged
evidence bindings. Operations target distinct nodes and must appear in canonical
document-node order; duplicate targets or overlapping replacements are rejected.
The server applies them in that order and records the exact operation list,
post-edit node digests, accepted Article IL digest, and canonical Markdown digest
so the accepted revision can be replayed and audited byte for byte.

These mechanical checks prove operation and lineage integrity, not semantic
equivalence. The independent factual audit remains the hard semantic gate. It
rejects a stale catalog, missing or non-unique old text, an operation outside its
byte/count budget, or any change to factual meaning, claim strength, causation,
quotation, number, date, name, condition, caveat, or uncertainty. Opening and
conclusion text receive no exception from these requirements.

Node movement, repetition deletion as a structural operation, split, merge,
cross-movement reorder, and node deletion are outside prototype v1. They may be
added only if a later prototype demonstrates reader value and deterministic
movement, source, and lineage validation. A globally wrong disclosure order
fails the attempt and should improve the Article Narrative or author contract
in the next experiment rather than be silently reconstructed by an editor.

## Validation Hierarchy

Validation has three levels.

### Hard failures

Hard failures block an accepted Article regardless of reader preference:

- changed or unavailable frozen source input;
- unsupported factual content;
- invented scene, feeling, quotation, cause, name, date, number, or condition;
- material caveat, uncertainty, or claim-strength loss;
- source-binding, revision, node, artifact, privacy, or lineage failure;
- malformed or unfinalized workspace;
- silent fallback or report reclassification.

### Repairable findings

A single bounded repair may address node-local opening, clarity, transition,
repetition, terminology, or conclusion defects. Mechanical validation must
preserve the exact Article/source lineage, and the independent factual audit
must accept the resulting meaning and source support before the revision can
advance.

### Human and advisory judgment

Fun, immediate memory, stated sharing desire, broad voice, and whether the
Article is worth reading remain whole-read judgments. They are not reduced to
one automatic score. Automated readers may provide directional evidence and
contradictions; the user's reading is the project go/no-go decision, not evidence
of population-level reader effects, long-term memory, or actual sharing behavior.

## Output Contract

One accepted Article IL revision produces:

- `article-il.json` as an internal intermediate;
- `article.md` as the canonical portable text artifact;
- `article.html` as a reading-first self-contained artifact;
- `article.pdf` as a derivative;
- a manifest and content-free review receipts.

The HTML first viewport contains the title, an optional short standfirst, and
substantive prose. It does not lead with a summary card, table of contents,
relationship map, pipeline graph, model settings, artifact lineage, or source
inventory. References and provenance remain available but subordinate to
reading.

Markdown and HTML must be deterministic for identical accepted IL bytes. PDF is
bound to the same IL and renderer receipt; byte-identical PDF output is not
assumed. The current archive `Run` records PDF renderer identity, while the
current product manifest drops it, so the Article prototype harness must carry
Chrome product/revision into its own manifest rather than assuming current
product output already does. Equation and Mermaid support may require embedded
JavaScript while remaining free of external subresource dependencies.

## Product Architecture After a Passing Gate

The final package names follow demonstrated ownership rather than this document
alone. At minimum, product integration must preserve these boundaries:

- Article owns Intent, Narrative, lifecycle, progress, artifact lineage, and UI
  meaning.
- A small neutral document/compile seam may be extracted only after Report IL
  output behavior is characterized and two real consumers require it.
- Article-owned consumer ports isolate source access, provider execution,
  workspace finalization, and durable storage from concrete adapters.
- Article packages do not import Report policy, Report workflow, Report events,
  or report-run identity.
- Report packages do not import Article as a fallback.
- SQLite child repositories remain root-only implementation details.

Do not begin with a broad `agentexec`, source-access, or compiler
reorganization merely to obtain a clean diagram. The archive prototype should
prove the product before permanent abstractions are created.

## Durable Lifecycle After a Passing Gate

The product implementation uses its own `article.*` event namespace and logical
Article-run membership. It reuses lifecycle patterns, not report event names.

Required behavior includes:

- `article_run_id` owns immutable input facts; each execution attempt has a
  separate identity and parent/origin lineage;
- the input-facts SHA-256 binds the exact Intent revision, source catalog and
  selection, allowed editorial-input artifacts, provider/model/tool/schema/
  budget/output contracts;
- initial creation receives a client request key unique within mission and user
  scope and stores it with a canonical request fingerprint; exact replay returns
  the original Intent/run, while the same key with different inputs conflicts;
- Intent, initial pending attempt, request-key record, and mission-level
  exclusion against overlapping turn, workflow, report, or Article work are
  committed atomically before provider work;
- process-local cancellation is the immediate signal under the current
  one-runner-per-database assumption, but durable
  `article.attempt.canceled` is a terminal state;
- success, failure, and canceled terminal writes use one conditional boundary so
  exactly one wins; a canceled terminal prevents recovery from advancing;
- content-addressed checkpoints bind the run input-facts hash and appear only
  after expensive stages fully finalize and validate;
- `resume_failed` requires the source attempt to have exactly one failed terminal
  and validates intent/input/checkpoint hashes before creating its new pending
  attempt; canceled, completed, purged, or ambiguous attempts are ineligible;
- a restart attempt keeps run/origin/parent lineage for audit but marks a hard
  traversal boundary; the single checkpoint walker stops at that node and never
  searches its ancestors;
- an interrupted exact-author repair continuation is terminal; post-repair
  confirmation failure records `repair_exhausted` and permits no resume;
- one Article run permits at most one explicit user-requested complete restart,
  with a new attempt, new author, and no prior checkpoint reuse. A second full
  reauthoring requires a new Intent revision and new Article run, so restart
  cannot become an unbounded best-of-N author search;
- one SQLite transaction stores every final artifact byte, manifest, Article-run
  membership, and success terminal; rollback exposes none of them;
- content-free typed failures and truthful usage receipts;
- revision/facts-hash delete preview and a content-free tombstone. Preview and
  delete require a terminal run with no open attempt or process-local owner.
  Delete rechecks revision/facts in one transaction that writes the tombstone and
  purges owned event/artifact membership; the tombstone makes every late worker
  terminal ineligible at the same conditional boundary.

Durable worker leases, heartbeats, fencing, multi-process scheduling, and a
general execution-kernel rewrite are outside the first slice.

## Web Surface After a Passing Gate

Plasma adds a separate `글` tab beside `리포트`.

The default form asks for the three Article Intent inputs. Progress is presented
in user language:

1. reading the material;
2. finding the reader's path;
3. writing the Article;
4. reading and refining the whole Article;
5. preparing reading outputs.

The completed card foregrounds the title, the reader promise, Read, and
Markdown/HTML/PDF downloads. Pipeline stages, artifact IDs, hashes, source
selection, and lineage belong in a secondary operator/provenance detail.

A failed Article gives the user an explicit next action: retry from a verified
checkpoint when available, restart from frozen inputs, revise the reader promise
as a new Intent revision and run, or abandon the attempt. A failed attempt is
never relabeled as a completed report. The later user-acceptance action remains
an Issue #461 product gate; it is not hidden inside automatic reader stages.

Direct user editing may reuse interaction ideas from redpen, but it must not
reuse report-specific routes, events, workcopy identity, or deletion semantics.
It is not required for the archive prototype or first generated Article.

## Explicit Non-goals

- replacing or renaming existing Report paths;
- adding Article to `report_mode`, `reportpipeline`, or a rigor/tone selector;
- Report-to-Article or Article-to-Report fallback;
- short social posts, platform-specific copy, or automatic publishing;
- tutorial-specific topology in the first slice;
- multiple style presets or imitation of a real writer;
- independently authored prose Sections;
- automatic Article-as-source registration;
- raw IL editing in the browser;
- invented drama or a required narrative curve;
- one automatic fun or engagement score;
- a broad cross-product infrastructure rewrite;
- public snapshot, tag, or release before a separate release decision.

## Issue and Release Rule

Issue #461 owns design, experiment, implementation, validation, and user review.
No subissues are created. Several short PRs and experiment runs may be used, but
each remains in this issue's checklist and comment lineage.

The issue cannot become `issue:completed` or `release:ready` until the user has
read the candidate, confirmed its independent value, exercised the runtime, and
accepted the final behavior on internal `main`.
