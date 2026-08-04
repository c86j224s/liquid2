// Package reportprompt owns report-generation prompt policy.
//
// It assembles stable guidance text, profile normalization, guidance hashes, and
// report composition strategy choices without depending on Web, CLI, MCP, or app
// transport packages. Callers own prompt envelopes, request orchestration, and
// persistence.
package reportprompt
