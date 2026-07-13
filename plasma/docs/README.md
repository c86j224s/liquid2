# Plasma Documentation

This directory documents Plasma as a product, an architecture, an operational
surface, and an experiment-driven research project.

The preferred documentation shape is:

- write the primary document in English
- keep a synchronized Korean counterpart beside it as `*.ko.md`
- explain experiment code names instead of hiding them
- separate current product rules from legacy notes, experiments, and future
  backlog ideas

> Migration note:
> Plasma documentation was written while the product was still being discovered.
> Some older documents are Korean-first or mix product decisions with experiment
> history. Do not treat that as the desired end state. Issue #67 established the
> current documentation structure; later cleanups should be tracked as separate
> backlog issues when they become concrete.

## Start Here

Read these first if you are trying to understand Plasma:

1. [Plasma README](../README.md) /
   [Plasma README Korean](../README.ko.md) - product overview and development
   commands.
2. [Glossary](glossary.md) / [Glossary Korean](glossary.ko.md) - product terms
   and experiment code names.
3. [C1 Default Loop](c1-default-loop.md) /
   [C1 Default Loop Korean](c1-default-loop.ko.md) - the current default product
   workflow.
4. [Product Architecture](product-architecture.md) /
   [Product Architecture Korean](product-architecture.ko.md) - durable product
   and backend boundaries.

## Current Product Rules

These documents describe how Plasma should behave now:

- [C1 Default Loop](c1-default-loop.md)
- [Product Architecture](product-architecture.md)
- [Product Flow](product-flow.md) - Korean-first bridge document that preserves
  product-flow history until a separate cleanup issue replaces it.
- [Automatic Investigation](automatic-investigation.md) - Korean-first
  legacy/current bridge document that preserves investigation-flow history until
  a separate cleanup issue replaces it.
- [Media And Document Source Implementation Design](media-source-implementation-design.md)
- [Token Diet Instrumentation](token-diet-instrumentation.md)
- [Mission Polling Measurement](mission-polling-measurement.md) /
  [Mission Polling Measurement Korean](mission-polling-measurement.ko.md)

## Source And Connector Work

These documents explain source intake and external-origin integration:

- [Confluence Source Integration](confluence-source-integration.md)
- [Confluence Live Validation Checklist](confluence-live-validation-checklist.md)
- [Media And Document Source Implementation Design](media-source-implementation-design.md)

Use the glossary distinction carefully:

- a connector is an adapter for an external origin
- a source is mission research material
- a raw artifact is stored content or extracted text
- a source snapshot is the accepted mission-level source record

## Legacy And Future Design Notes

These are useful background, but they are not the default product path:

- [Legacy Ledger Loop](legacy-ledger-loop.md)
- [Evidence Signal Model](evidence-signal-model.md) /
  [Evidence Signal Model Korean](evidence-signal-model.ko.md)

Evidence and claim records are not part of the current C1 default loop. If they
return, they should help source navigation, citation, uncertainty tracking, and
traceability. They must not become a gate that blocks investigation or report
generation.

## Experiments

Experiment summaries live under [experiments/](experiments/README.md). The
experiment directory should contain readable protocols, decision memos, and
small redacted metrics. Raw run payloads, screenshots, generated HTML, and
private corpora belong outside the repository under the artifact archive policy.

Important code-name families include:

- `C1`: the current default product loop
- `C0`, `PAL2`, `NAV`: controller strategy experiments
- `G2`, `H5`: report tone and humanization experiments
- `DH23`: designed HTML report rendering experiment
- `C4`: long-form report assembly experiment

See [Glossary](glossary.md) for definitions before using those labels in product
documents.

## Operations

Runtime and local artifact handling are documented in:

- [Plasma Artifact Archive](artifact-archive.md)
- repository-level [Configuration](../../docs/configuration.md)
- `plasma/README.md` for common development commands

## Documentation Maintenance Rules

- Prefer adding a short reader-facing introduction before dense design detail.
- Keep current behavior, historical notes, and future ideas visually separate.
- When moving files, update all links in this directory and in `plasma/README.md`.
- If a document references an experiment code name, explain it locally or link to
  the glossary.
- Korean counterparts should preserve the same meaning, not merely summarize the
  English source.
