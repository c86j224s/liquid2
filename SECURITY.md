# Security Policy

Liquid2 and Plasma are local-first applications. Do not expose local development
or release servers to an untrusted network unless an explicit authentication and
network boundary has been added for that deployment.

## Supported Versions

The project is pre-1.0. Security fixes apply to the current `main` line unless a
release note says otherwise.

## Reporting A Vulnerability

Please open a private security advisory on GitHub when available. Before any
public repository launch, the repository owner must enable GitHub private
vulnerability reporting or publish another private security contact route.

If neither private advisories nor a private contact route are available, do not
publish vulnerability details in a public issue. The repository is not ready for
public vulnerability intake until that route exists.

Do not include live credentials, tokens, private documents, database files, or
full sensitive URLs in a public issue.

## Local Data

Runtime configuration, local SQLite databases, generated reports, uploaded
sources, and raw experiment artifacts are local user data. They should not be
committed to this repository. See `docs/configuration.md` and
`plasma/docs/artifact-archive.md` for the expected local storage boundary.
