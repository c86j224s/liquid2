# Confluence Live Validation Checklist

Status date: 2026-07-05

Live tenant validation status: pending live credentials.

Do not mark this integration live-validated until each applicable item has a
tenant result, timestamp, tester, and notes. Mock tests are useful regression
coverage, but they are not live usability validation.

## Connection

- [ ] Settings UI presents API token registration as the only 0.0 connection
      path: pending live credentials.
- [ ] API-token connection accepts email, API token, and site URL without asking
      the user for cloud id: pending live credentials.
- [ ] OAuth start/callback routes fail with a clear API-token-only message:
      pending live credentials.
- [ ] Multiple accessible sites: pending live credentials.
- [ ] Stored site lookup for API-token connections: pending live credentials.
- [ ] Rename connection: pending live credentials.
- [ ] Local revoke clears usability without breaking snapshots: pending live credentials.
- [ ] Delete/forget connection without breaking snapshots: pending live credentials.

## Discovery

- [ ] Spaces browse: pending live credentials.
- [ ] Space pages browse: pending live credentials.
- [ ] Page children browse: pending live credentials.
- [ ] Search within selected site/space: pending live credentials.
- [ ] Pagination cursor behavior: pending live credentials.

## Source Approval

- [ ] Candidate review metadata: pending live credentials.
- [ ] Candidate preview is not stored as source: pending live credentials.
- [ ] Full snapshot approval re-fetches page: pending live credentials.
- [ ] Version drift blocks approval: pending live credentials.
- [ ] Large page returns too-large result: pending live credentials.
- [ ] Range snapshot stores precise locator and selected content only: pending live credentials.
- [ ] Old full-page snapshot payloads remain readable: pending live credentials.

## Errors And Redaction

- [ ] 401 expired/invalid token: pending live credentials.
- [ ] 403 permission or missing scope: pending live credentials.
- [ ] 404 missing page/site: pending live credentials.
- [ ] 429 rate limited with retry-after handling: pending live credentials.
- [ ] Cloud-id mismatch: pending live credentials.
- [ ] Revoked/expired local connection: pending live credentials.
- [ ] No token, Authorization header, cookie, raw provider body, private page body, prompt, or raw MCP response leaks in UI/API/CLI/MCP/log review: pending live credentials.

## Updates

- [ ] No-update status: pending live credentials.
- [ ] Update-available metadata: pending live credentials.
- [ ] Update preview body is fetched only after explicit request: pending live credentials.
- [ ] Update approval creates a new snapshot: pending live credentials.
- [ ] Old snapshot remains readable after update: pending live credentials.
- [ ] Partial/range update requires range reselect: pending live credentials.
