// Package articleexperiment owns the archive-local Article IL pilot harness.
//
// It validates frozen protocol, fixture, arm, and run-cell identities, then
// records immutable run inputs and terminal receipts around a caller-supplied
// executor. It does not own provider policy, Article prose, product events,
// SQLite integration, Web routes, or product runtime state.
package articleexperiment
