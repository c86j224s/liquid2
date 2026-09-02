// Package reporthumanize owns the legacy transport-neutral H5 report
// humanization pass.
//
// The package runs the same-session MCP patch agent, validates the finalized
// Markdown artifact, writes H5 terminal events idempotently, and recovers
// finalized patches after restart. HTTP and CLI callers still own request
// normalization, executor selection, route/flag orchestration, and run locks.
//
// Deprecated: manual/post-canonical H5 export remains only for historical
// artifacts and direct API or CLI compatibility. The active long-form
// post_report_humanize path is a separate pre-canonical style-edit stage in the
// reporting final-edit pipeline; it does not use this package.
package reporthumanize
