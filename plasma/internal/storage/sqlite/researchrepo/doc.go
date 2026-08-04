// Package researchrepo persists research records for evidence, claims, questions,
// options, and proposal bundles.
//
// It owns research-record SQL only. Root sqlite.Store keeps cross-capability
// transaction ownership and calls the narrow Tx helpers here for atomic writes.
package researchrepo
