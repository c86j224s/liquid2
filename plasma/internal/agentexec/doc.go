// Package agentexec owns transport-neutral Plasma agent provider execution.
//
// It defines the provider request/result contract, Codex and Claude process
// adapters, MCP server process configuration, and provider session fork/readiness
// behavior. Web, CLI, and other transports remain responsible for their own
// prompts, request parsing, and product orchestration.
package agentexec
