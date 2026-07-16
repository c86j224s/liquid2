# Contributing

This repository is currently operated as a private development workspace. Public
contribution intake is not open until the repository owner explicitly publishes
a public contribution process.

## Before Sending Changes

- Keep product boundaries separate: `liquid2/` and `plasma/` are separate
  product roots.
- Do not commit local config files, databases, reports, raw experiment runs, or
  private source snapshots.
- Use the product-owned scripts in `SETUP.md` for build and server control.
- For GitHub issue and PR workflow rules, follow `docs/github-workflow.md`.

## Pull Requests

When public pull requests are enabled, each PR should include:

- the linked issue or reason for the change
- a short user-visible summary
- verification commands and results
- any known risk or follow-up work

Security-sensitive reports should follow `SECURITY.md` instead of public PR or
issue discussion.
