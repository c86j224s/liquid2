# Runtime Configuration

Liquid2 and Plasma read runtime settings from TOML files. Config files are
intended for local paths, provider credentials, and runtime integration values
that should not be committed.

The Go binaries parse TOML directly. Browser scripts do not parse app runtime
config; they only select development or release mode and control launchd or
Flutter. File selection, default values, and status output are owned by each
app.

## Liquid2 Files

Liquid2 has two runtime surfaces, so new configuration should use two files:

- `liquid2/config.api.toml` for the API server
- `liquid2/config.web.toml` for the Flutter web server

The old unified `liquid2/config.toml` file is still supported as a fallback.

Development runtime precedence, highest first:

1. Environment variables and explicit CLI arguments
2. `liquid2/config.web.toml`
3. `liquid2/config.api.toml`
4. `liquid2/config.toml`
5. `~/.config/liquid2-web-dev/config.toml`
6. `~/.config/liquid2-api-dev/config.toml`
7. `~/.config/liquid2-dev/config.toml`
8. App defaults for development mode

Release runtime precedence, highest first:

1. Environment variables and explicit CLI arguments
2. `~/.config/liquid2-web/config.toml`
3. `~/.config/liquid2-api/config.toml`
4. `~/.config/liquid2/config.toml`
5. App defaults for release mode

The dev script sets only `LIQUID2_RUNTIME_MODE=dev`; the release script sets
only `LIQUID2_RUNTIME_MODE=release`. The scripts ask `liquid2-api status` for
the resolved API URL, web host/port, database, and environment label instead of
parsing TOML themselves.

Tracked examples:

- `liquid2/config.api.toml.example`
- `liquid2/config.web.toml.example`
- `liquid2/config.toml.example` for the unified fallback shape

## Liquid2 Sections

Split API files use responsibility-based sections:

```toml
[server]
addr = "127.0.0.1"
port = 6011
cors_origins = ["http://127.0.0.1:6001"]

[paths]
db_path = "/path/to/liquid2.db"
export_dir = "/path/to/exports"
backup_dir = "/path/to/backups"

[runtime]
jobs_enabled = true
seed_demo = false

[logging]
level = "info"
format = "text"
source = false

[translation]
provider = ""

# Translation provider settings are opt-in. Add this table only when
# `provider = "codex"` should send document text through the Codex CLI.
# [translation.codex]
# command = "codex"
# model = ""
# timeout_seconds = 300

```

Split web files use only the web server and browser runtime label settings:

```toml
[server]
addr = "127.0.0.1"
port = 6001

[runtime]
environment_label = "DEV"
```

Unified fallback files use the same section names below product tables, such as
`[liquid2-api.server]`, `[liquid2-api.paths]`, `[liquid2-web.server]`, and
`[liquid2-web.runtime]`.

Existing top-level keys, `[liquid2]`, `[liquid2-api]`, and `[liquid2-web]` flat
tables remain accepted for compatibility. Launchd settings such as service
label, plist path, binary path, and stdout/stderr paths are controlled by the
browser scripts' process-control constants, not Liquid2 runtime TOML.

## Plasma Files

Plasma continues to use one product config file because it currently has one
local browser/server process.

Development runtime precedence, highest first:

1. Environment variables and explicit CLI arguments
2. `plasma/config.toml`
3. `~/.config/plasma-dev/config.toml`
4. App defaults for development mode

Release runtime precedence, highest first:

1. Environment variables and explicit CLI arguments
2. `~/.config/plasma/config.toml`
3. App defaults for release mode

Plasma browser scripts do not parse runtime config and do not inject settings
such as `db_path`, `addr`, `agent`, or `liquid2_url` as CLI arguments. The dev
script sets only `PLASMA_RUNTIME_MODE=dev`; the release script sets only
`PLASMA_RUNTIME_MODE=release`. File selection, default values, and status
output are owned by the Plasma app. Use `plasma status` or the browser scripts'
`status` command to inspect the resolved URL, database, mode, Liquid2 URL, and
agent setting.

Tracked example:

- `plasma/config.toml.example`

## Plasma Sections

Plasma separates server, paths, agent, local source, and connector settings:

```toml
[plasma-server]
addr = "127.0.0.1"
port = 6002
liquid2_url = "http://127.0.0.1:6011"

[plasma-paths]
db_path = "/path/to/plasma.db"
static_dir = "auto"

[plasma-agents]
enabled = ["codex", "claude"]
workdir = "/tmp/plasma-agent-workdir"
timeout = "10m"

[plasma-agents.codex]
command = "codex"
# Interactive Codex sessions use Plasma's gpt-5.6-terra / medium default.
# Model and reasoning effort are selected per mission in Plasma.
# These remain specialized workflow-goal drafting settings.
workflow_goal_model = ""
workflow_goal_reasoning_effort = "low"

[plasma-agents.claude]
command = "claude"
model = "haiku"
max_budget_usd = ""

[plasma-local-sources.roots]
workspace = "/path/to/repository-parent"
docs = "/path/to/documents"
```

For `static_dir`, omit the key to serve the embedded assets bundled in the
binary. Use `static_dir = "auto"` only when the development server should read
the repository static directory from disk, or provide an explicit directory path.
Boolean-like aliases such as `on`, `off`, `1`, `default`, or `none` are
rejected so the runtime mode is not ambiguous.

`[plasma-local-sources.roots]` is a map from root ID to an allowlisted parent
directory. For source attachment, choose a `root` such as `workspace` and pass a
relative path inside that parent, for example `liquid2/README.md`. This lets one
root expose several repositories without leaking absolute paths to the UI, CLI,
or MCP agent.

Confluence is API-token only in the 0.0 product path. Register Confluence
connections from Plasma Settings with an Atlassian email, API token, and site
URL.

Legacy OAuth keys may still be parsed by older code paths, but OAuth start and
exchange flows are disabled for the 0.0 product path. Do not add
`[plasma-confluence-oauth]` to normal development or release config unless the
OAuth product path is explicitly reintroduced:

```toml
[plasma-confluence-oauth]
client_id = ""
client_secret = ""
redirect_uri = "http://127.0.0.1:6002/api/settings/connectors/confluence/oauth/callback"
scopes = ["read:confluence-content.all", "offline_access"]
authorize_url = ""
token_url = ""
discovery_url = ""
```

If OAuth is reintroduced later, the browser Settings panel can pass the global
Settings callback URL when it starts OAuth. The callback must return a small
Korean browser page rather than raw JSON and must not expose OAuth state, code,
tokens, client secret, Authorization headers, or provider response bodies.

Confluence source intake uses the same source rules in browser and CLI:
discovery results and previews are candidates/results, while accepted sources
are connector-fetched snapshots pinned by `cloud_id`, `page_id`, page version,
and optional plain-text range offsets. Live Atlassian validation is documented
in `plasma/docs/confluence-live-validation-checklist.md`; leave it marked
pending credentials until a real tenant run is completed.

Existing top-level keys, `[plasma]`, and older `[plasma-server]` path or
path keys remain accepted for compatibility. Launchd settings such as service
label, plist path, binary path, and stdout/stderr paths are controlled by the
browser scripts' process-control constants, not Plasma runtime TOML.
