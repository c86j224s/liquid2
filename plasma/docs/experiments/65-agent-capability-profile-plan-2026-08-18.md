# Minimal Agent Capability Profile Product Record

Status: implemented and offline-tested in the issue #349 worktree; not committed,
merged, deployed, or released.
Date: 2026-08-18.
Owner context: issue #349.
Review context: the earlier broad plan received adversarial Luna review; later code
reconnaissance narrowed the accepted product change to the minimum recorded here.

## Objective

Reuse Plasma's existing capability controls and apply the smallest caller-selected,
versioned capability profile to the normal product paths without another provider
experiment.

This record replaces the earlier over-scoped design in this file. The implemented
change does **not** add a separate operation-ID hierarchy, move ordinary Codex
inference to app-server, or require a complete runtime catalog assertion before each
call.

## Evidence Boundaries

### Existing mechanism evidence

An isolated Luna measurement previously established one narrow mechanism result:
when the actual provider-facing capability bundle was reduced, provider-reported
input usage also fell.

Across eleven balanced fresh-session pairs:

- every candidate call reported 4,206 fewer input tokens than its paired current
  call;
- semantic correctness passed in both arms for all eleven simple synthetic tasks;
- one representative pair exposed 3 versus 2 direct tools and 164 versus 13
  deferred tools; and
- that pair serialized 59,112 versus 39,947 model-input bytes and 101,159 versus
  44,622 wire-request bytes.

The measured candidate was a broad experimental bundle and incorrectly removed
product-required Plasma tools. The 4,206-token delta is therefore neither the
marginal effect of a specific tool nor the expected saving of the product profile.
It is not an adoption threshold.

### No new provider experiment

The implementation recorded here used unit, fake-provider, adapter, package, and
full offline Go tests only. It made no new Luna or other provider calls. No product
saving is claimed for the implemented profile.

### Artifact and privacy boundary

This record contains no raw prompt, provider request or response, provider JSONL,
provider session ID, credential, private mission content, real source content, or
private runtime path. Raw experiment material remains outside the repository.

## Product Capability Contract

Normal conversation and autonomous workflow steps retain all sixteen Plasma
research and control tools:

| Capability group | Tools | Decision |
| --- | --- | --- |
| Mission orientation | `plasma.research.outline`, `plasma.research.changes` | Keep |
| Mission discovery | `plasma.research.list`, `plasma.research.grep` | Keep |
| Mission grounding | `plasma.research.read`, `plasma.research.references` | Keep |
| Accepted source access | `plasma.sources.list`, `plasma.sources.read` | Keep |
| Live local source access | `plasma.sources.tree`, `plasma.sources.grep` | Keep |
| Source discovery | `plasma.sources.search` | Keep |
| Candidate review | `plasma.sources.candidates.propose`, `plasma.sources.candidates.read` | Keep |
| Presentation validation | `plasma.mermaid.validate` | Keep |
| Workflow control | `plasma.workflow.status`, `plasma.workflow.stop` | Keep |

The previous experimental candidate that removed source discovery, candidate review,
and workflow control tools is not a product profile.

Accepted live local-path sources continue to use Plasma source tools so reads remain
inside the approved source boundary and can create `source.observed` provenance.
Provider work-directory readers do not replace mission-bound source reads.

Source search still returns review candidates, not accepted sources. Candidate
proposal and staged-candidate read remain available; staged material remains
unapproved and outside the default report source set until the user accepts it.

## Implemented Minimal Profiles

The provider-neutral contract is one small typed pair:

```go
type ProfileID string

type Profile struct {
    ID       ProfileID
    Revision string
}
```

Revision `1` defines three profiles:

| Profile | Owner and use | Compatibility behavior |
| --- | --- | --- |
| `legacy.v1` | Historical sessions and callers that have not selected a profile | Preserves pre-profile behavior |
| `research.v1` | New conversation and workflow sessions | Keeps the Plasma research contract and removes supported unrelated ambient capability |
| `workflow_goal_draft.v1` | One request-local workflow-goal draft | Disables product tools and session persistence for the helper call |

The zero-value request resolves to immutable `legacy.v1`. This prevents existing
report and compatibility callers from silently changing behavior merely because the
new fields exist.

Profiles are selected by product callers. Prompts, user text, `mcp_mode`, and
heuristics do not choose them.

## Caller Wiring

### Conversation and workflow

- A new browser or CLI conversation selects `research.v1`.
- A new browser or CLI workflow selects `research.v1`.
- A resumed conversation or workflow resolves the profile persisted with the
  provider session.
- Workflow steps, automatic compaction, manual compaction, and retry after
  compaction carry the same profile ID and revision.
- Successful response and compaction events persist the profile identity.
- Unknown profile IDs and unsupported revisions fail before provider execution.

### Historical sessions

A historical successful response with a provider session ID but no profile fields
maps deterministically to `legacy.v1` revision `1`. It is not upgraded to
`research.v1` while being resumed.

### Reports

Existing report-stage MCP allowlists and report bindings remain the stage contract;
no duplicate operation-ID layer was added.

When a report uses an existing research session or its isolated fork, a small
executor wrapper applies the source session's profile to every report-stage request.
The wrapper preserves optional fork and fork-readiness interfaces. Report artifact
lineage allows the report session and the pre-report research session to inherit the
same profile during later resume, humanize, or patch work.

A report-only path with no recorded research lineage remains on `legacy.v1` for
compatibility. No narrower report-only profile was introduced.

### Workflow goal draft

The goal-draft route explicitly selects `workflow_goal_draft.v1`. It does not infer
that profile from the prompt.

## Provider Mapping

### Claude

For `research.v1`, the adapter reuses the existing exact Plasma MCP configuration,
built-in allowlist, deny set, slash-command disablement, and safe-mode isolation.
The sixteen Plasma research/control tools remain available.

For `workflow_goal_draft.v1`, the adapter:

- passes `--tools ""`;
- omits the Plasma MCP configuration;
- uses safe mode;
- disables slash commands; and
- uses a non-persistent helper session.

This is the implemented zero-tool goal-draft mapping for Claude at the CLI argument
and generated-MCP-config boundary.

### Codex

Ordinary inference remains on `codex exec` and `codex exec resume`. Native Codex
compaction remains on its existing app-server compaction path. The change does not
introduce app-server inference.

For `research.v1`, request-scoped Codex configuration disables supported ambient
features unused by Plasma:

```text
features.tool_suggest=false
features.recommended_plugins=false
tools.experimental_request_user_input.enabled=false
features.apps=false
features.plugins=false
features.multi_agent=false
features.shell_tool=false
features.view_image=false
tools.update_plan.enabled=false
skills.include_instructions=false
project_doc_max_bytes=0
```

The adapter intentionally does not use Codex `--ignore-user-config` for this
profile, because that can remove provider connection, authentication, and model
configuration needed to reach the configured service.

For `workflow_goal_draft.v1`, the adapter also omits Plasma MCP tools and uses an
ephemeral session. This is a **tool-minimized Codex goal-draft mapping**, not a
verified zero-tool catalog. Current public controls do not prove that every direct,
deferred, hidden, Code Mode, or apply-patch surface is absent. The product and this
document must not call the Codex mapping fully tool-free.

## Session Lineage Contract

The durable identity for this change is:

```text
executor + provider session ID + capability profile ID + profile revision
```

The profile is persisted on successful conversation/workflow responses and
compaction events. Resume resolves and validates that pair. An unknown profile or
revision fails loudly rather than falling back to the current default.

Report artifact events do not duplicate the profile fields through every report
pipeline type. Instead, restore follows the recorded same-session or pre-report
session lineage and inherits the already persisted profile. This keeps the change
small while preserving the provider-session invariant.

This implementation does not add:

- a separate capability operation ID;
- adapter-version or catalog fingerprints;
- complete direct/deferred/hidden catalog manifests;
- new usage telemetry; or
- automatic session migration.

## Offline Acceptance

Tests cover:

- zero-value resolution to `legacy.v1`;
- rejection of unknown profiles and revisions;
- Codex and Claude research-profile mapping;
- goal-draft tool disabling and ephemeral behavior;
- Claude `--tools ""` and omitted MCP configuration behavior;
- new browser and CLI conversation/workflow profile selection;
- workflow adapter propagation through normal steps and compaction;
- historical session restoration as `legacy.v1`;
- isolated/fresh report restoration to the pre-report research profile;
- report executor wrapper preservation of fork capabilities;
- workflow goal-draft request selection;
- existing report allowlist and provider-MCP acceptance suites; and
- package dependency boundaries.

The final gate is the repository's full offline Go suite:

```text
go test ./...
```

No live provider call is part of this acceptance record.

## Known Limits

- The Codex goal-draft profile is not proven to expose zero provider tools.
- Codex ambient MCP servers cannot be completely removed without also risking the
  configured provider connection; this change does not claim full ambient isolation.
- No complete provider-facing catalog assertion exists.
- No production token saving has been measured for the correct 16-tool profile.
- Historical sessions intentionally keep `legacy.v1` rather than receiving the new
  ambient reductions.
- Report-only sessions without research lineage intentionally remain legacy.
- The dormant proposal-extraction helper with no production caller is unchanged.

These are explicit limits, not passing assertions.

## Deferred Changes

The following are outside issue #349's minimal implementation:

- reducing any of the sixteen Plasma research/control tools;
- adding operation IDs;
- moving ordinary Codex inference to app-server;
- building full provider catalog manifests or fail-before-inference catalog checks;
- creating a narrower report-only profile;
- heuristic tool-free classification of ordinary user turns;
- changing development or release servers or databases;
- another provider experiment; and
- commit, pull request, merge, deployment, or release without a separate user gate.

## Adoption Decision

Adopt only the minimal typed profile, caller wiring, session lineage persistence,
existing report-contract reuse, Claude goal-draft tool disabling, and supported
Codex ambient reductions described above.

Do not adopt the prior experimental tool list or the over-scoped operation-ID,
app-server-inference, and complete-catalog architecture. The accepted cost is a small
profile type, provider mapping, lineage fields, and caller propagation. The accepted
limitation is that Codex is reduced only through supported request-scoped controls
and is not represented as completely tool-free or fully observable.
