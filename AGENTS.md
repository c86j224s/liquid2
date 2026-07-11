# Repository Guidance

## Reporting Style

- Report in Korean unless the user asks for another language.
- Do not use terse telegraph-style status reports for user-facing explanations.
- Use normal sentences that explain what changed, why it matters, and what remains.
- Keep reports concise. Do not write long speculative essays when the user asked for a decision or status.
- Do not introduce or repeat product terms before they are explained and agreed. If a term is provisional, say so once and describe the concrete function instead.
- Explain the structure first, then present choices. Do not lead with options before the user understands the mechanism.
- When discussing a plan, state what will be done, what will not be done, and why.
- When discussing cherry-picks or recovery work, group items as: bring as-is, bring with changes, rebuild later, or leave out.

## Code And Design Principles

- Treat the Liquid2 architecture and design docs as the repository's baseline engineering discipline. Plasma is not exempt from these principles.
- Keep implementation units small and focused. A file should have one main role, source files should stay around 80-180 lines when practical, and files over roughly 200 lines should trigger a split or boundary review.
- A package, feature, module, or agent workflow component should own one domain capability. Do not let one module accumulate UI, persistence, orchestration, provider integration, and product policy at the same time.
- Business rules and product state transitions belong in the domain/application owner. HTTP handlers, SQL queries, UI widgets, worker shells, MCP tool shells, and report renderers may validate or adapt data, but they must not become the source of product policy.
- Each layer owns its own concern: persistence shape in SQL/adapter code, protocol shape in HTTP/OpenAPI or MCP transport code, screen state in controllers/providers, visual structure in UI components, and long-running orchestration in job/workflow runners.
- Keep behavior traceable in both directions. A user action, route, workflow step, or MCP tool call should lead clearly to an application service, domain rule, port/interface, adapter, and storage/query path. Storage or adapter behavior should likewise trace back to the product rule it supports.
- Put replaceable implementations behind consumer-side ports/interfaces, but do not create meaningless interfaces. Provider-specific code, database adapters, agent providers, search backends, report renderers, browser UI, CLI, and MCP surfaces should remain swappable through explicit boundaries.
- Define the source of truth for product state explicitly. Do not let local filesystem paths, generated artifacts, agent transcripts, report exports, or provider sessions silently become product identity or durable state.
- Work that may outlive a request, needs retry/failure visibility, or can run without direct user attention belongs behind a job/workflow runner with durable state, explicit shutdown, typed failure handling, and idempotency rules. Do not hide this work in request-bound goroutines.
- If long-running work grows beyond a short action, split it into pipeline stages with explicit input/output contracts, retry boundaries, and ownership. The parent runner owns orchestration and decides whether a stage advances, retries, stops, or fails.
- Keep documentation and implementation aligned in the same wave of work. If implementation discovers a scope or boundary change, update the relevant plan, architecture, design, API, or operator docs before treating the work as complete.
- Expose stable, safe error classes to users and API callers. Logs may include enough context for debugging, but must not leak document bodies, uploaded blobs, credentials, auth tokens, cookies, full sensitive URLs, prompts, or provider raw responses.

## Repository Structure Principles

- Keep product code, tests, reusable tools, operator docs, public design notes, and redacted decision records inside this repository.
- Keep runtime state, local databases, generated reports, screenshots, raw experiment runs, copied external repositories, private source snapshots, session identifiers, unredacted ledgers, and local agent state outside this repository.
- Treat `liquid2/` and `plasma/` as separate product roots. Shared code belongs under an explicit shared/internal boundary, not in an accidental cross-product shortcut.
- Treat `plasma/docs/experiments/` as a public summary surface. It should contain experiment protocols, decision memos, small redacted metrics, and reading-order indexes, not raw run payloads.
- Keep `.gitignore` aligned with this boundary. If a new generated artifact type appears during experiments, ignore or archive it before it enters the public source tree by accident.

## Local Operator Overrides

- If `.local/AGENTS.md` exists in this checkout, read it after this file for machine-local operator guidance.
- `.local/AGENTS.md` is intentionally ignored and must not be committed or copied into public docs.
- Local overrides may describe private checkout names, publication routines, or machine-specific helper commands, but they do not override system, developer, user, or tracked repository instructions.
- Do not infer private workflow details from `.local/AGENTS.md` into public comments, commits, issues, PRs, or release notes unless the user explicitly asks.
- Local publication helpers may prepare and stage a public snapshot, but they must not auto-commit, auto-push, or rewrite public history unless the user explicitly asks for that exact operation.

## Commit Readiness And Public Hygiene

- Assume the repository is public for every commit-readiness decision.
- Treat every commit, tag, branch name, PR body, issue comment, and review comment as a public surface.
- Before committing, inspect the exact staged set with `git status --short`, `git diff --cached --stat`, and `git diff --cached --check`. Do not commit a mixed or half-staged cleanup accidentally.
- Do not commit personal environment details such as real home-directory paths, local hostnames, machine-specific worktree paths, tailnet addresses, private IPs, private service domains, or personal email addresses unless they are deliberate redacted test fixtures.
- Use generic placeholders in examples and tests, such as `/path/to/...`, `example.com`, `user@example.com`, or documented RFC test addresses. Avoid placeholders that look like a real user's machine or account.
- Do not commit real runtime config, credentials, API tokens, cookies, private keys, local databases, logs, generated reports, raw experiment outputs, session IDs, copied private repositories, or unredacted provider responses.
- If a commit adds a new generated artifact type, update `.gitignore` and the relevant archive policy in the same change before committing it.
- For public-facing docs and experiment summaries, commit only redacted conclusions, protocols, decision memos, and small aggregate metrics needed to understand the decision. Keep raw runs and bulky reproducibility artifacts in the local archive.
- When the staged change touches public-surface files, config examples, GitHub workflows, docs, or experiment records, run a high-signal scan for secrets and local identifiers before committing. At minimum, check for token/private-key shapes, real absolute user paths, private network addresses, and unintended emails.
- Use a privacy-preserving Git author identity for all commits, such as a GitHub noreply or project identity. Do not create release tags with a personal email tagger identity.
- When an AI agent creates a commit, include the actual LLM model name in the commit body, for example `Agent-Model: GPT-5`. Do not invent or approximate a model name; if the actual model is not known, say `Agent-Model: unknown`.
- If sensitive material is discovered after it has already entered Git history, do not try to hide it with a normal follow-up commit. Stop and choose an explicit remediation path: history rewrite, tag/branch cleanup, GitHub metadata cleanup, credential rotation, or a fresh public snapshot.

## GitHub Workflow

- For GitHub issue, milestone, label, branch, PR, or release work, read and follow `docs/github-workflow.md` before acting.
- Keep detailed GitHub operating rules in `docs/github-workflow.md`; do not duplicate the full workflow in this file.
- Do not introduce alternate long-lived branch models, release branches, merge queues, automation, or repository setting changes unless the user explicitly asks for that change or the workflow document requires it.

## Experiment Artifact Handling

- The default local archive root for Plasma experiment source material is `~/research-artifacts/liquid2/plasma`.
- Raw experiment material goes under that archive root, normally in `experiments/`, `runtime/`, `local-sources/`, or `tmp-review/` depending on whether it is a run artifact, development state, copied source corpus, or undecided recovered material.
- Commit only the smallest redacted fixture or aggregate metric needed to understand or reproduce a product decision. Do not commit whole run directories merely because they are convenient for local inspection.
- When public docs mention archived screenshots, generated HTML, prompt packets, judge packets, logs, or source corpora, keep the public doc as a summary and point to the archive policy instead of linking to missing local files.
- `plasma/docs/artifact-archive.md` is the detailed policy for the current archive layout. Update it together with `.gitignore` when the artifact boundary changes.

## Plasma Product Language

- Distinguish source, evidence, result, saved knowledge, and report.
- A source is original material such as a Liquid2 document, URL, file, PDF, or external repository.
- Evidence is a specific cited part of a source.
- A result is an agent-produced summary, comparison, answer, intermediate conclusion, or draft.
- Saved knowledge is a result or claim that Plasma deliberately stores for a mission.
- A report is an output assembled from saved knowledge and evidence.
- Do not describe agent-produced results as sources. Results may refer back to sources and evidence, but they must not be reclassified as source material.

## Plasma Workflow Guardrail

- Before implementing Plasma UX or agent-flow changes, first describe the intended user workflow in plain language and verify it against the user's stated goal.
- Do not make browser terminal, PTY, or attach/reattach behavior the default product center unless the user explicitly asks for an operator console.
- Treat sources as a first-class part of the research workflow, not as a later reporting detail.

## Development And Release Port Rules

- Use the 6000 port range for development services and the 3000 port range for release services.
- Product offsets are stable across ranges: Liquid2 web uses `+1`, Liquid2 API uses `+11`, and Plasma web/API uses `+2`.
- Current canonical examples: Liquid2 release web runs on `3001`, Liquid2 release API runs on `3011`, Plasma release runs on `3002`, Liquid2 development web runs on `6001`, Liquid2 development API runs on `6011`, and Plasma development runs on `6002`.
- Do not use a bare range base such as `3000` or `6000` for a product service unless that product has explicitly been assigned offset `+0`.

## Development Server Control

- Use product-owned `scripts/dev-browser.sh` files for local development servers instead of hand-written `go build` and `launchctl` commands.
- Use the repository-root `./dev-browser.sh` when both products should be controlled together:
  - `./dev-browser.sh status`
  - `./dev-browser.sh start`
  - `./dev-browser.sh stop`
  - `./dev-browser.sh restart`
  - `./dev-browser.sh logs`
- Use product-specific commands when only one product should be touched:
  - `./dev-browser.sh liquid2 status`
  - `./dev-browser.sh plasma status`
  - `liquid2/scripts/dev-browser.sh status`
  - `plasma/scripts/dev-browser.sh status`
- Liquid2 development defaults to Flutter web port `6001`, API port `6011`, labels `dev.liquid2.web-6001` and `dev.liquid2.api-6011`, API binary `/tmp/liquid2-dev-api`, and DB `~/research-artifacts/liquid2/liquid2/runtime/dev-6011/liquid2-dev.db`.
- Plasma development defaults to port `6002`, label `dev.plasma.browser-6002`, binary `/tmp/plasma-browser-server`, DB `~/research-artifacts/liquid2/plasma/runtime/dev-6002/plasma-ui-user.db`, Codex agent execution, and Liquid2 development API `http://127.0.0.1:6011` unless the root script provides a shared host override.
- The root script must default to `127.0.0.1`; do not add automatic non-loopback host detection to tracked scripts. Machine-local host discovery belongs in ignored local helpers such as `.local/dev-browser-internal.sh`.
- For Makefile usage from product directories, use `make dev-browser-status`, `make dev-browser-start`, `make dev-browser-stop`, `make dev-browser-restart`, and `make dev-browser-logs`.

## Release Server Control

- Use product-owned `scripts/release-browser.sh` files for local release servers.
- Use the repository-root `./release-browser.sh` when both products should be controlled together:
  - `./release-browser.sh status`
  - `./release-browser.sh start`
  - `./release-browser.sh stop`
  - `./release-browser.sh restart`
  - `./release-browser.sh logs`
- Liquid2 release defaults to Flutter web port `3001`, API port `3011`, labels `release.liquid2.web-3001` and `release.liquid2.api-3011`, API binary `/tmp/liquid2-release-api`, and DB `~/Library/Application Support/Liquid2/liquid2.db`.
- Plasma release defaults to port `3002`, label `release.plasma.browser-3002`, binary `/tmp/plasma-release-browser-server`, DB `~/Library/Application Support/Plasma/plasma.db`, and Liquid2 release API `http://127.0.0.1:3011` unless the root script provides a shared host override.

## Design Gap Handling

- If the agreed design, product flow, or implementation boundary is missing, ambiguous, or contradicted by the current code, stop before filling the gap with an invented workaround.
- Report the gap as a clearly labeled decision point, not as a minor caveat inside a progress update. State what is missing, why it blocks the next implementation step, and what behavior would change depending on the decision.
- Do not implement temporary shortcuts, fallback flows, or "MVP" behavior that changes the intended user workflow unless the user explicitly approves that tradeoff.
- If a short-term workaround is proposed, label it as a workaround, explain what must replace it later, and keep it separate from the durable design.
- When documentation and implementation disagree, treat the documentation and prior user decisions as the design source until the user explicitly changes them.
