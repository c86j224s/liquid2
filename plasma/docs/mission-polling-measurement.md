# Mission Polling Measurement

Issue #96 changes only the active-work observation path. A selected mission
still loads the established full-detail representation; the browser does not
use a cache or a new reconciliation endpoint.

## Reproducible Fixture

`TestMissionPollingLargeFixtureMetrics` creates one mission with 240
non-activity ledger events containing representative evidence-sized payloads
and one in-flight turn, registered with the server so detail recovery leaves it
open. It measures HTTP response bytes for the existing full detail and
`/api/missions/{id}/activity` read surfaces. It makes no external network calls
and writes its SQLite database below Go's test temporary directory.

Run it with:

```sh
cd plasma
go test ./internal/web -run '^TestMissionPollingLargeFixtureMetrics$' -count=1 -v
```

## Recorded Result

The baseline is a historical measurement recorded from `a9ba837` before this
change; it is not an assertion executed by the current test suite because that
commit did not contain this harness. The after values below are emitted by the
command above. Timing is intentionally excluded from the comparison.

| Observation | Baseline | After #96 |
| --- | ---: | ---: |
| fixture events | 240 | 240 |
| full detail response | 449,087 B | about 449,200 B |
| activity response | 1,081 B | 1,205 B |
| unchanged pending poll requests | 4 | 1 |
| advanced/gap/restart pending poll requests | 4 | 2 maximum |

The four baseline requests were mission detail, mission list, Confluence
connections, and mission Confluence access. `TestSelectedMissionActivityPollUsesCursorBeforeDetailFallback`
executes the current browser functions from their real initial detail state:
an unchanged cursor makes one activity request. A valid advance, a cursor gap
or regression, an incompatible cursor, or a changed server instance makes one
activity request plus one selected-mission detail request. It does not reload
the mission list or Confluence settings.

The additional activity bytes are the typed cursor schema, sequence, and server
instance identifier. Full detail preserves its existing fields and adds the
same `activity_cursor`, allowing the selected mission to seed polling without a
second request. Generated timestamps can vary the raw detail response by a few
bytes, so the test asserts the stable property: activity stays below 5% of full
detail. Static browser tests assert unchanged, advanced, gap, and restarted
cursor request behavior using request counts and paths rather than timing.
