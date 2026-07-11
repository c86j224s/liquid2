# Autonomous Research Trends - 2026-06

This note records the external research and product scan that informed the
Plasma controller and automatic-investigation direction in June 2026.

The scan was run separately from the C0/PAL2/NAV experiment. Its purpose was to
check whether Plasma's direction aligns with current autonomous research,
deep-research, agentic-RAG, and research-product patterns.

This note is a documentation synthesis, not a Plasma mission source snapshot.
The linked product pages and papers are the external references; agent-produced
summaries and experiment outputs are treated as results, not source material.

## Product Trend

Modern research products are converging on slow, tool-using, source-cited
research runs rather than single-turn chatbot answers.

OpenAI Deep Research presents research as a multi-step internet investigation
that can take minutes and produce cited reports. The important product signal
for Plasma is not "make one giant prompt"; it is "run an agentic investigation
that can search, read, pivot, and synthesize over time." See
[OpenAI Deep Research](https://openai.com/index/introducing-deep-research/) and
the [Deep Research system card](https://cdn.openai.com/deep-research-system-card.pdf).

Anthropic's Research system uses an orchestrator-worker multi-agent pattern for
broad research tasks. The useful lesson is proportionality: multi-agent research
helps when the task is broad enough to justify parallel search, but it also adds
coordination cost and should not become the default for every turn. See
[How we built our multi-agent research system](https://www.anthropic.com/engineering/multi-agent-research-system).

Gemini Deep Research has moved toward API-level research agents that plan,
execute, synthesize cited reports, support collaborative planning, connect to
MCP servers, accept files, and produce visualizations. This directly supports
Plasma's MCP-first research IDE direction: the agent should receive tools and a
thin workflow contract, not a large stuffed prompt. See
[Gemini Deep Research Agent](https://ai.google.dev/gemini-api/docs/deep-research)
and [Deep Research model docs](https://ai.google.dev/gemini-api/docs/models/deep-research-preview-04-2026).

Microsoft Copilot Researcher combines a deep-research model with Microsoft 365
orchestration and work-context search. The useful Plasma lesson is connector
plurality: Liquid2, URLs, local paths, external repositories, and future private
workspaces should all be source surfaces behind replaceable connectors. See
[Introducing Researcher and Analyst](https://www.microsoft.com/en-us/microsoft-365/blog/2025/03/25/introducing-researcher-and-analyst-in-microsoft-365-copilot/).

NotebookLM remains important as a source-grounded notebook product. Its lesson
for Plasma is still the source boundary: user-selected original material should
stay distinct from agent-produced answers and reports. See
[NotebookLM](https://notebooklm.google/) and
[Google's NotebookLM research updates](https://blog.google/innovation-and-ai/products/notebooklm/better-research-notebooklm/).

## Research Pattern

The literature also points toward a pipeline, not a monolithic prompt.

`Deep Research: A Survey of Autonomous Research Agents` describes deep research
as planning, question development, web exploration, and report generation. For
Plasma this means "better reports" and "better investigation" should be tested
as separate concerns. See [arXiv:2508.12752](https://arxiv.org/abs/2508.12752).

`Deep Research Agents: A Systematic Examination And Roadmap` highlights dynamic
reasoning, long-horizon planning, multi-hop retrieval, iterative tool use,
structured analytical reports, static versus dynamic workflows, single-agent
versus multi-agent composition, and MCP-style extensibility. This supports
Plasma's current direction: a same-session agent with mission-bound tools is a
valid default, and larger parallel research should remain an optional expansion.
See [arXiv:2506.18096](https://arxiv.org/abs/2506.18096).

ReAct remains a useful base pattern: reasoning, acting, observing, and then
continuing from what the tool returned. Plasma's corresponding product form is
the mission ledger plus MCP tool calls. See
[ReAct](https://arxiv.org/abs/2210.03629).

Self-RAG and CRAG both argue against indiscriminate retrieval. Self-RAG's lesson
is that retrieval should happen when useful and then be reflected on. CRAG's
lesson is that retrieved material can be wrong or weak, so the system needs a
way to evaluate and recover from poor retrieval. Plasma should therefore make
source access easy and observable, but should not force every turn through a
heavy retrieval or controller path. See
[Self-RAG](https://openreview.net/forum?id=hSyW5go0v8) and
[CRAG](https://arxiv.org/abs/2401.15884).

Anthropic's context-engineering guidance is consistent with the same principle:
curate a small useful tool surface and avoid bloated prompts or ambiguous tool
sets. This reinforces Plasma's "thin guidance plus MCP/source reads" rule. See
[Effective context engineering for AI agents](https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents).

## Plasma Implications

The scan supports the broad Plasma direction:

- Keep Plasma's center as mission ledger + same agent session + MCP/source read
  tools + report artifacts.
- Treat browser UI, CLI, MCP, and future adapters as replaceable clients over
  the same ledger.
- Keep sources as original material. Agent answers, controller messages, and
  reports are results or artifacts, not sources.
- Prefer thin guidance and mission-bound tools over large injected recall
  payloads, report-only corpora, or prebuilt report packs.
- Use bounded automatic investigation as a continuation of the same mission,
  not as a separate product mode.
- Consider parallel or multi-agent research only for larger tasks where breadth
  justifies the cost.

The scan does not support adding a strong always-on controller. A controller
should not become a second researcher, source producer, judge, or report
writer. If used, it should act like a user who notices stagnation, excessive
narrowness, missing source access, or drift, and then sends a short steering
turn.

## Connection To The 2026-06-26 Experiment

The corrected 45-run C0/PAL2/NAV experiment found no controller variant worth
shipping as a default. NAV was statistically worse than C0 in the aggregate,
and PAL2 was inconclusive.

That result is compatible with the external research scan. Current systems need
planning, retrieval, verification, and synthesis, but that does not imply that
every turn should receive a structured controller prompt. Plasma should keep the
default loop simple and add controller behavior only after a concrete failure
mode is detected and experimentally validated.
