// Package reportrun owns Plasma's logical report-run storage contract.
//
// The package classifies report ledger events into run membership using only
// explicit lineage fields. It does not store event payloads or artifact bodies;
// those remain owned by the ledger and raw artifact stores.
package reportrun
