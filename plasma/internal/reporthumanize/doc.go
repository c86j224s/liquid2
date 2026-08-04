// Package reporthumanize owns the transport-neutral H5 report humanization pass.
//
// The package runs the same-session MCP patch agent, validates the finalized
// Markdown artifact, writes H5 terminal events idempotently, and recovers
// finalized patches after restart. HTTP and CLI callers still own request
// normalization, executor selection, route/flag orchestration, and run locks.
package reporthumanize
