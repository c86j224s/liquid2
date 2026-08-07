# Plasma MCP Error Guide

URI: `plasma://docs/mcp/errors`

This guide explains how Plasma MCP clients should interpret errors and choose
the next action.

## Error Layers

Plasma MCP has two error layers.

- JSON-RPC protocol error: the request envelope, method, or params shape is
  invalid. For example, an unknown method returns a method-not-found error.
- Tool execution error: the tool name and JSON-RPC request are valid, but Plasma
  product rules, binding, input values, or connector state prevent successful
  execution.

Tool execution errors are returned through `isError: true` in the `tools/call`
result and the `error` field inside Plasma's tool response envelope. That
envelope is the stable tool contract. Clients should inspect `error.error_kind`,
`error.message`, `error.retryable`, and `error.related_object_ids`.

## Common error_kind Values

- `validation`: input values, mission binding, session binding, or allowed scope
  do not match. Fix the input before retrying.
- `approval_required`: the state transition requires user approval. Agents must
  not bypass it.
- `conflict`: a different change conflicts with the same idempotency key or draft
  state. Read the current state and construct a new request.
- `binding`: the reporting stage or session boundary is not enabled for this MCP
  server instance. Do not rewrite runner-provided binding values.
- `internal`: the server failed internally. If `retryable` is false, repeating
  the same request is unlikely to help.

## Resource URI Errors

`resources/read` separates invalid resource URI input from unknown resources.

- Invalid resource URI input returns JSON-RPC `-32602`. A blank URI is invalid
  params. A malformed URI such as `"not a uri"` is also invalid URI input.
- An unknown but well-formed resource returns JSON-RPC `-32002` with `resource
  not found`. For example, `plasma://docs/mcp/unknown` is well-formed but is not
  part of Plasma's static resource catalog.
- These examples are intentionally public-safe. They do not include mission,
  session, source, or runtime identifiers.

`resources/list` is a static non-paginated catalog. Absent params, `null`, `{}`,
and an empty cursor are accepted. Malformed params or a non-empty cursor return
JSON-RPC `-32602` with `invalid params`.

## Retry Guidance

`retryable: true` means the agent may retry after improving input or waiting for
a temporary condition. It does not guarantee success. `retryable: false` means
the same arguments should not be repeated unchanged.

For mutating tools with an idempotency key, keep the same key for the same
meaningful request. Sending a different change with the same key may produce a
conflict.

## Public Safety

Error messages must be safe to show to users and agents. Do not paste
credentials, cookies, private keys, provider responses, sensitive URLs, or full
source bodies into documents or reports. When reporting an error, use the safe
`error_kind`, summary message, and related object ids.
