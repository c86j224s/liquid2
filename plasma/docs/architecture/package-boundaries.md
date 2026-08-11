# Plasma Package Boundaries

This document defines the package and import rules summarized by the
[Architecture Map](README.md). These rules are normative for new work and for
behavior-preserving refactors. The synchronized Korean document is
[Package Boundaries Korean](package-boundaries.ko.md).

## Ownership Rules

| ID | Rule | Review question |
| --- | --- | --- |
| P1 | A Go package is the primary independent implementation unit. | Can its responsibility be described without “and”? |
| P2 | A package owns one capability or one narrow technical mechanism. | Do its files share one contract and reason to change? |
| P3 | A technical umbrella is not a capability. | Are unrelated Web, MCP, CLI, reporting, or SQLite features grouped only by technology? |
| P4 | Product rules and state transitions live with the owning capability. | Would changing this decision require editing a handler, SQL query, or provider adapter? |
| P5 | Replaceable implementations use consumer-owned ports. | Does the consumer import a concrete adapter type or adapter-owned policy? |
| P6 | Long-running work belongs to a runner owned by its capability. | Are retry, cancellation, recovery, or idempotency hidden in a transport? |
| P7 | `cmd/plasma` owns construction and wiring only. | Is reusable behavior implemented in a command or borrowed from Web? |
| P8 | Shared kernels stay small and semantically stable. | Is a `common`, `models`, or `helpers` package hiding an undecided owner? |

Package names and directory nesting follow ownership. A parent package may
provide a thin registration or composition surface, but it must not re-export
every child model and service or become another facade.

## Allowed Dependencies

| Caller | May depend on | Must not depend on |
| --- | --- | --- |
| `cmd/plasma` composition | Transport constructors, capability constructors, adapter constructors | Reusable feature policy |
| Feature transport adapter | Its protocol helpers and capability API | Sibling transports, concrete storage, connector, or provider implementations |
| Capability service | Its models, consumer-owned ports, small shared kernels, explicit APIs of collaborating capabilities | Web, MCP, CLI, or concrete adapters |
| Capability runner | Capability state and ports, provider port, clock/ID abstractions needed for replay | Request-bound state or transport-owned goroutines |
| Replaceable adapter | Consumer-owned port and model, external SDK or system library | Product policy or unrelated capability state |
| Shared kernel | Standard library and smaller primitives only | Transport, feature workflow, or concrete adapter packages |

Cross-capability calls are allowed only through a deliberate public API for a
real use case. They must not create an import cycle or expose another
capability's adapter.

## Boundary Review

Review a package boundary when any of these signals appears:

- its responsibility requires “and” to describe unrelated behavior;
- it exposes separate groups of models, errors, services, or stores to different
  consumers;
- feature changes repeatedly touch unrelated files in the package;
- tests and fixtures for several product capabilities accumulate together;
- the package has high fan-in because it re-exports shared contracts, or high
  fan-out because it coordinates unrelated systems;
- operational code grows beyond roughly twenty files or several thousand lines;
  or
- one test file or implementation file becomes a navigation bottleneck.

Counts are review triggers, not automatic violations. Keeping a large package
requires a written reason that its files still share one contract and one reason
to change.

The following changes do not resolve a boundary by themselves:

- splitting a large file while leaving all responsibilities in one package;
- creating child packages while the parent re-exports their complete API;
- moving unrelated types into `common`, `models`, `helpers`, or a service
  locator; or
- introducing interfaces that have no replaceable implementation or consumer
  boundary.

## Refactor Order

1. Characterize the public Web, MCP, CLI, event, persistence, error, and recovery
   behavior that must remain stable.
2. Name the capability owner and its consumer-side ports.
3. Move models and policy to the owner before moving transport or adapter code.
4. Move concrete implementations behind the new ports.
5. Update one product surface at a time and keep each wave independently
   testable and revertible.
6. Remove temporary aliases, re-exports, and compatibility facades in the same
   refactor series.
7. Add or update an architecture check for the dependency direction that was
   repaired.

Do not combine a boundary refactor with database schema changes, new product
behavior, or unrelated UX work unless that scope is explicitly approved.

## Comments, Tests, And Enforcement

- A non-trivial package comment states what the package owns, its neighboring
  boundary, and what does not belong there.
- Capability tests stay with product rules. Transport tests cover protocol
  mapping. Adapter tests cover infrastructure behavior. A small integration
  suite covers the composed path.
- Architecture checks should use the Go import graph to reject forbidden edges.
  Size signals should request review rather than fail a build mechanically.
- Documentation and code move together once implementation begins. Until a
  tracked refactor is complete, the architecture map must label current
  exceptions explicitly.

Run the current import boundary check from `plasma/`:

```sh
go test ./internal/architecturecheck
```

The checked-in baseline records exact file/import pairs for known debt. After
an intentional refactor removes an entry, regenerate the baseline and review
the diff; never use regeneration to accept unrelated new debt.

```sh
go test ./internal/architecturecheck -args -update
```

## Current Exceptions

Issue #66 tracks the broad `internal/app`, `internal/web`, `internal/mcp`,
and `internal/reporting` boundaries. Their current existence does not authorize
new dependencies with the same shape.

The first migration wave moved ledger, mission, artifact, source, stable error
models, and selected report execution boundaries to focused packages.
Starting with #111, `internal/reportworkflow` owns the product-fixed report
topologies, typed stage wiring, long-form prefix stages, final edit stages, and
legacy finalization stage. Each stage package owns its prompts, MCP allowlists,
provider execution, validation, and durable replay boundaries, while
`internal/reporting` keeps the durable final-edit contracts and artifact
lineage. Compatibility aliases in `internal/app` are temporary migration
surfaces, not new ownership.

The SQLite migration keeps one stable root facade for connection ownership,
migrations, maintenance, and cross-capability transactions. Feature repository
packages are implementation details of that facade: only the root package may
import them, and sibling repositories may not depend on one another.
