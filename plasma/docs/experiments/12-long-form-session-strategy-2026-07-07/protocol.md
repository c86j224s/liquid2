# Protocol

Raw archive:
`~/research-artifacts/liquid2/plasma/experiments/12-long-form-session-strategy-2026-07-07/`

## Isolation Rules

The experiment must not use development or release Plasma databases. It uses
archive-local SQLite databases, archive-local provider homes, and loopback-only
experiment server ports.

Product code, product defaults, browser UI, CLI behavior, MCP defaults, and
runtime server scripts are not part of the experiment surface.

## Safety Rules

The experiment uses archive-local command wrappers and rejects unsafe command
shapes:

- no product default DB path;
- no development or release DB path;
- no non-loopback server bind;
- no 3000-range release port;
- no raw provider home;
- no report generation command without an explicit archive-local DB.

This rule exists because an earlier stopped execution attempted to start a
Plasma server without explicit `-db` and `-addr`. The stopped command did not
start, but the experiment treats that as a hard safety incident.

## A0 Path

`A0-current-chain` represents the product-style long-form path:

1. planning starts the report session chain;
2. section generation receives the previous report session id;
3. part assembly receives the latest report session id;
4. final framing receives the latest report session id;
5. the same report-session chain grows throughout the long-form run.

This path is useful as the baseline because it exposes the context-pressure
failure class.

## B Path

`B-independent-sections` represents the divide-and-conquer candidate:

1. plan the report structure;
2. draft each section in an independent provider conversation;
3. save each section as a durable report-part result;
4. assemble parts and the final report from section outputs without rewriting
   section bodies.

Section outputs are report parts, not sources. The report must still cite
original source material.

## C Follow-Ups

The C-series variants were added after B proved structurally safer but too thin:

- `C1` tested whether a reframe-only connective pass improved readability.
- `C2` made section drafting denser.
- `C3` fixed duplicate heading assembly.
- `C4` normalized final heading assembly while preserving section files.

## Measurement

The primary automated measurements are:

- final report word count;
- source section word count;
- heading drift count;
- adjacent duplicate heading count;
- estimated input, uncached input, and output token ratios;
- run completion and artifact presence.

Natural prose quality still requires human review. The automated measurements
do not prove that the report is beautiful or narratively strong.

