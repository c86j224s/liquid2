// Package startuprecovery sequences finite startup reconciliation steps.
//
// Steps are finite, durable DB-only reconciliation. Capability packages own
// recovery rules and transaction/idempotency boundaries. Callers own
// composition, logging, and fatal/nonfatal policy. Steps must not call
// providers, connectors, or external APIs; start background work; or depend on
// process-local locks. Composition enforces this contract, not the callback
// type.
package startuprecovery
