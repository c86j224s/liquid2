// Package reportpatch owns transport-neutral report patch agent bindings.
//
// The package selects the provider session used for a patch run and builds the
// ordinary patch prompt/tool surface. HTTP and CLI callers adapt their request
// state before calling this package; transport schemas, storage, and reporting
// policy remain outside this boundary.
package reportpatch
