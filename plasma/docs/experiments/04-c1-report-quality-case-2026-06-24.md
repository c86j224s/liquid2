# C1 Report Quality Case - 2026-06-24

This note records a product-use case where the C1 loop produced a noticeably
better Oda Nobunaga report. It is not a controlled benchmark. It is a concrete
case study to preserve what worked, what still failed, and what Plasma should
optimize for next.

## Case

- Mission ID: `mis_20260618144638_2de7d6f3`
- Topic: Oda Nobunaga's life
- Better report artifact: `art_20260624085849_f122578a`
- Better report event: `evt_20260624085849_13594e4b`
- Report title: `오다 노부나가의 생애 리포트`
- Agent session: `019edb33-a7e9-7900-9f5b-60330dbdcb1f`
- Report tool session: `ses_20260624085029_03495011`
- Rigor level: `exploratory`

The immediately relevant bounded workflow run was
`wfr_20260624073636_d281eb2c`, started from the instruction `다른 전투들도 상세
조사.` It completed seven steps before pausing on the 20 minute duration limit
that was active at the time.

## Observable Difference

The latest report improved over the previous Markdown artifact in ways that were
visible both qualitatively and mechanically.

| Metric | Previous artifact | Better artifact |
|---|---:|---:|
| Artifact ID | `art_20260623223529_11907b0c` | `art_20260624085849_f122578a` |
| Characters | 8,858 | 14,756 |
| Lines | 115 | 167 |
| Markdown headings | 13 | 15 |
| Public URL links | 0 | 34 |
| Internal footnote-style refs | 44 | 0 |
| `信長公記` mentions | 1 | 7 |
| `신장공기` mentions | 2 | 16 |
| `전승` mentions | 2 | 6 |
| `데도리가와` mentions | 0 | 7 |

The size increase was not the main improvement. The better report read like an
article rather than a record dump. It opened with a source-reading frame,
separated stable facts from transmitted stories, cited public URLs instead of
internal Plasma IDs, and used uncertainty as a narrative asset rather than
flattening it away.

## Why This Worked

### Same Session Continuity

The successful report was generated from the same Codex provider session that
had handled the preceding conversation and bounded workflow steps. Plasma did
not ask an isolated writer to reconstruct the mission from a large prompt pack.
The report writer had the same conversation memory and could use MCP reads for
details.

This supports the C1 decision: report generation should normally run in the same
agent session as the investigation, with Plasma tools available for lookup.

### Investigation Before Report Writing

The preceding workflow did useful shaping work:

- Step 1 reframed "other battles" into battle categories and identified gaps.
- Step 2 read pinned `信長公記` text for Nagashino, Tennoji, Kizuura, and later
  northern-campaign sections.
- Step 3 treated Tedorigawa as a source conflict instead of accepting the popular
  "Uesugi victory" story at face value.
- Steps 4-7 tried to locate Uesugi-side letters, `歴代古案`, NDL, CiNii, and book
  catalog records, then preserved the failure path and next search direction.

This was not just more collection. It created a useful report stance: "read
Nobunaga-side records closely, then mark where opposing-side or later-source
support is still needed."

### Source Framing Was Part Of The Report

The better report started by explaining how to read the sources. It described
`信長公記` as the central source while warning that it is still a Nobunaga-side
record. It then used Britannica and wiki-style summaries as secondary
orientation, not as equal primary evidence.

That source hierarchy gave the report a stronger voice without becoming
overconfident.

### Uncertainty Made The Report Richer

The report did not avoid uncertain material. It included transmitted stories and
popular images, but labeled them as stories, later reception, or contested
interpretation. This matched the desired C1 direction: do not block useful
material from entering the conversation; make the report explain how reliable or
tentative each layer is.

## What Still Failed

### Liquid2 Search Was Unavailable

Several workflow turns reported `127.0.0.1:3000 connection refused` for
`plasma.sources.search`. The agent kept working through pinned sources and web
search, but the connected-source path was not healthy.

Product implication: a successful C1 run should not require Liquid2 to be up,
but connector failure must be visible and should not silently weaken source
discovery.

### Bibliographic Leads Were Not Reviewable

The agent found useful bibliographic leads, such as:

- Yada Toshifumi, `上杉謙信－政虎一世中忘失すべからず候`
- `上杉家御年譜 第1巻 謙信公`
- `上杉謙信書翰`
- `越佐史料`

Because it could not find exact NDL/CiNii/publisher URLs, it correctly avoided
creating normal URL source candidates. However, these leads were left only inside
conversation results and workflow next instructions.

Product implication: C1 needs a lightweight review surface for "source search
leads" or "bibliographic leads" that are not yet source candidates. These should
not become sources and should not require evidence/claim machinery, but they
should be visible enough for the user or a later workflow to continue.

### This Is Not A Statistical Result

This case supports the earlier report-generation A/B direction, but it is still
a single product-use case. The measured difference is useful as a regression
target and design signal, not as proof that every C1 report will improve.

## Product Guidance

Use this case as a quality target for future C1 report work.

1. Keep report generation in the same provider session by default.
2. Keep the prompt thin; optimize the MCP/source reading surface instead.
3. Let bounded workflow build the agent's understanding before asking for a
   report.
4. Prefer public source citations and source-reading explanations over internal
   Plasma IDs.
5. Preserve uncertainty, source hierarchy, and contested interpretations in the
   report instead of filtering them out too early.
6. Treat agent answers as results. Do not turn results into sources.
7. Add a reviewable surface for source-search leads that are not yet source
   candidates.
8. Make connector failures visible, but do not let one failed connector end the
   whole investigation when other paths remain.

## Regression Signals

A future Oda-style report should be considered weaker if it:

- leaks internal Plasma IDs as public citations,
- uses no public URL links when sources are available,
- collapses primary sources, secondary summaries, and popular stories into one
  flat authority level,
- avoids contested or uncertain material instead of labeling it,
- ignores preceding workflow results from the same session,
- or depends on a large pasted prompt pack instead of reading through MCP tools.
