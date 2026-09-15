# Confluence OAuth retirement

Refs #486. Confluence product connections remain API/access-token-only.

Removed unreachable OAuth URL/code-exchange CLI implementations and unused CLI
OAuth config builders; removed unregistered Web OAuth start/callback/return-page
handlers and the unreachable refresh-token execution branch; removed the unused
connector OAuth client and its execution-only tests.

The CLI command/flag parsing tombstones still explain that OAuth is unsupported.
Existing HTTP unsupported responses and API-token connection/search/read behavior
remain. Legacy OAuthConfig/config keys and persisted connection fields are kept
for compatibility; no DB/schema migration, stored credential deletion or OAuth
reactivation occurs. Shared scope normalization, discovery and endpoint validation
used by token/access compatibility paths are retained.

Validation: full uncached product tests and go vet ./... pass. Existing token
connection and disabled OAuth CLI/HTTP tests remain. OAuth execution tests were
removed with their implementation, not skipped. No runtime state was changed by
this branch; merge/deployment is a separate operation.
