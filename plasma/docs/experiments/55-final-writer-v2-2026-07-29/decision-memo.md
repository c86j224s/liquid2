# Final Writer V2 Decision Memo

Date: 2026-07-29

Product decision updated: 2026-07-30

Status: product adoption approved; quality comparison remains mixed/inconclusive

## Decision

Adopt `assembly_writer_reader_style_gate_v2` for new planned narrative
long-form reports while preserving stored `reader_style_gate_v1` and legacy
replay behavior. The sealed model readers disagreed on three of four pairs, so
the quality comparison remains mixed and inconclusive; neither reading is
averaged into a tie or selected after the mapping reveal.

The adoption is an explicit user product decision under a narrower criterion.
The purpose of this change is to separate final assembly, final writing,
reader-facing editing, optional style editing, and corrective gating so each
responsibility can improve independently. The host direct reread found two B
wins and two ties, with no observed regression. Preserving current quality is
sufficient for this responsibility-separation change; immediate prose-quality
superiority is not claimed.

This is a bounded product decision informed by the corrected W6-B four-pair
fixed-reviewed-Part experiment, not a claim that the candidate has been
validated across all report topics or has passed a complete blind human
adjudication. The earlier W6-A execution remains archived only as invalid
directional evidence because its frozen Parts were hand-authored and did not
come from the upstream product path.

## Evidence Summary

All eight reports passed the hard gates before reading:

- same frozen reviewed Part manifest digest for A and B within each pair,
- no citation tag loss,
- no protected requirement loss,
- stage/tool trace matched the declared product path,
- archive adoption verified report hash, database and ledger lineage, final
  artifacts, and stage payload contracts,
- direct adjudication found zero unsupported external facts in either arm,
- blind packs were generated without a deterministic public seed.

The first reading preferred B in three pairs and found one pair tied:

| Pair | Topic | Rigor | Preferred Arm |
| --- | --- | --- | --- |
| `wang-anshi-northern-song-exploratory` | Wang Anshi and Northern Song reform memory | Exploratory | B |
| `wang-anshi-northern-song-strict` | Wang Anshi and Northern Song reform memory | Strict | B |
| `go-raft-implementation-roadmap-exploratory` | Go Raft implementation roadmap | Exploratory | B |
| `go-raft-implementation-roadmap-strict` | Go Raft implementation roadmap | Strict | Tie |

The mapping file was later found to have been rewritten after the first manual
adjudication, so that table is not a sealed blind result and cannot drive
acceptance. An independent host reread after the mapping was revealed reached a more
conservative result of two B wins and two ties. It kept B for Wang strict and
Go Raft exploratory, and judged Wang exploratory and Go Raft strict tied. The Wang
exploratory downgrade reflects a real trade-off: B improved entry and
transitions but added source-interpreter phrasing that weakens the direct
authorial stance. This secondary, non-blind check does not replace the blind
result; it remains diagnostic evidence of no observed B regression.

## Reading Notes

For the Wang Anshi pairs, B produced a more coherent Korean reading artifact.
It kept the fiscal, military, education, Shenzong, Sima Guang, Green Sprouts,
labor-service, local-burden, and factional-memory anchors while giving the
reader a clearer explanation of why reform ambition and implementation risk
belong in the same judgment.

For the Go Raft pairs, B gave the more useful roadmap. It preserved the fixed
Raft anchors and tied milestones to testable safety properties, crash recovery,
network partition caveats, storage boundaries, membership changes, snapshotting,
and operator observability without adding vendor-specific or unsupported
performance claims. The strict Go pair was judged a tie because B had stronger
safety-boundary synthesis while A had slightly cleaner surface formatting.

## Caveats

The experiment used fixed reviewed Parts and terminal pipeline execution, so it
does not measure upstream research, section drafting, or source-selection
quality.

No prompt-only correction was applied. The optional style stage was not invoked
because the run kept `post_report_humanize` at the product default disabled
setting.

The sealed reread used independent model readers rather than a completed human
blind panel. Their disagreement remains part of the record. Product adoption
does not convert that disagreement into experimental proof of quality gain.

Raw reports, ledgers, prompts, traces, session identifiers, private blind
mapping, and databases remain outside Git under the experiment archive policy.

## W6-C Sealed Reading

Both model readers confirmed that they read both reports in all four pairs end
to end and that no choice changed on rereading. After resolving the sealed
neutral labels, one reader selected A twice and B twice; the other selected A
three times and B once. They agreed only on A for the Wang Anshi exploratory
pair. The disagreement centers on whether added framing makes the report easier
to remember or instead becomes repetitive source-management narration.

This is a real reader-preference conflict, not a report tie. The automated
acceptance aggregate remains unwritten; the explicit user product decision is
recorded separately from that unresolved experimental comparison.
