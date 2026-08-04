// Package reportrepo persists reports, report versions, report blocks, and report
// promotion state.
//
// It owns report SQL only. Root sqlite.Store keeps cross-capability transaction
// ownership and calls the narrow Tx insert helpers here for atomic writes.
package reportrepo
