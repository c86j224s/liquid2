// Package mission owns the MCP mission.get and mission.update feature adapter.
//
// It decodes the mission tool wire payloads, preserves their validation and read
// ordering, and calls narrow canonical mission, source, and research-record ports.
// Root MCP policy such as allowlists, session guards, idempotency, tracing, and
// tool ordering remains in the parent mcp package.
package mission
