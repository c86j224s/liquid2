// Package reportrun owns Plasma's logical report-run registration and completion boundary.
//
// It classifies report ledger events into run membership, records delayed usage
// outcomes before one deterministic report.run.completed event, and performs
// DB-only recovery without resuming provider or executor work. It depends only on
// canonical ledger, mission, agent-usage, and report-usage contracts; report
// rendering, transports, and storage adapters remain outside this package.
// Run membership and completion code uses explicit lineage fields. It does not
// store event payloads or artifact bodies; those remain owned by the ledger and
// raw artifact stores.
package reportrun
