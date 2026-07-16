# Closed Smoke Record

The frozen #16 configuration used Codex for planned mode and Claude for
long-form mode. The immutable prepare gate completed, and the two planned
baseline/candidate product paths completed. The two long-form paths failed
before MCP plan submission because Claude authentication returned HTTP 401.

The prepare and smoke records are retained without modification. Their provider
lock cannot be changed and `_write_new` prevents replacing them, so #16 is not
eligible for a corrected rerun or quality analysis. Experiment 17 is the
separate Codex-only successor with a new immutable archive identity.

Raw sources, reports, prompts, ledgers, provider state, session identifiers,
and blind mappings remain outside Git under the experiment archive policy.
