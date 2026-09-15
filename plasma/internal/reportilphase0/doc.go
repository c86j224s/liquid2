// Package reportilphase0 owns Plasma's independent experimental Report IL
// authoring, validation, and target-compilation pipeline.
//
// It keeps Report-specific source, Narrative, authoring, reader, checkpoint,
// manifest, and rendering policy independent from the classic report authoring
// graph. Product adapters connect it to report execution, providers, the mission
// ledger, artifact storage, and Web UI; those integrations must not turn this
// package into a generic report or Article workflow.
package reportilphase0
