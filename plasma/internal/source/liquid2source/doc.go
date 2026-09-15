// Package liquid2source owns the Liquid2 source connector contract and its
// normalization and snapshot content-selection rules. It encodes snapshot payloads
// and locators without owning connector reads or persistence. It depends on source, artifact, ledger,
// and product error contracts; HTTP and application orchestration stay outside.
package liquid2source
