// Package reportexecution owns the durable execution lifecycle for Plasma report jobs.
//
// The package coordinates pending, terminal, cancellation, retry visibility, and
// execution request normalization. It deliberately does not own report writing,
// editing, prompt construction, or final rendering policy.
package reportexecution
