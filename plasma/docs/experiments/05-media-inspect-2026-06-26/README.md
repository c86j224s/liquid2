# Media Inspect Experiment Design - 2026-06-26

This experiment decides how Plasma should let agents use image and document
media during research and report writing.

For the Korean decision summary, see
[`decision-memo-ko.md`](./decision-memo-ko.md).

The design starts from one verified fact in the current product: Plasma can
store and expose source pages, HTML, metadata, and transcribed text, but it does
not yet give the agent a default tool that visually inspects image bytes or OCRs
document scans. If an agent describes a map, portrait, or scanned book today, it
is usually reasoning from page title, surrounding text, provider metadata,
search results, or upstream transcriptions, not from Plasma-side visual
inspection.

## Product Question

Should Plasma stay metadata-only for media sources, or should it add explicit
inspection tools that let an agent see images and document scans when needed?

The intended product direction is not to stuff image bytes into ordinary MCP
reads. The question is whether a separate, explicit inspection surface improves
research quality enough to justify the cost, latency, and provenance complexity.

## What Will Be Compared

### M0: Metadata-only Media Sources

M0 is the planned default source path.

The agent can list and read media source metadata:

- source snapshot ID
- artifact ID when pinned
- canonical source URL
- provider page URL
- direct media URL when known
- media kind
- MIME type
- byte size
- dimensions or duration when known
- license and attribution

The agent cannot inspect image pixels, OCR scan images, or receive base64 media
bytes through the default read path.

Expected strength: cheap, safe, provenance-friendly, and enough for citation or
report embedding.

Expected weakness: visual descriptions may be shallow or wrong unless the
source page already contains useful text.

### M1: Image Inspect Tool

M1 keeps M0, then adds an explicit image inspection tool.

The tool takes a media source snapshot or artifact reference and returns bounded
observations about the visual content. It should not reclassify those
observations as sources. The original image remains the source; the inspection
result is an agent/tool-produced result with provenance back to the image.

Expected strength: maps, portraits, diagrams, screenshots, and visual evidence
can be described more accurately.

Expected weakness: cost and latency increase, and tool-produced observations
must be labeled so they do not masquerade as source text.

### M1C: Conditional Image Inspect

M1C is the product-shaped version of M1.

The agent first reads only source metadata, provider pages, and transcribed
text. It must then decide whether visual inspection is necessary. If it is
necessary, it requests inspect for concrete source IDs and receives the image
inspection result in a second turn.

Expected strength: preserves the cheap metadata-first path while still allowing
visual reasoning when the mission actually requires it.

Expected weakness: adds a decision point; if the agent fails to request inspect
when the task needs it, the final report stays shallow.

### M2: Document Image Inspect and OCR Tool

M2 keeps M0, then adds a document-image path for scans.

The tool should first resolve a stable page image or IIIF manifest when
available. It may then use OCR or vision inspection. Provider-supplied
transcriptions such as Wikisource page text must be labeled separately from
Plasma-generated OCR.

Expected strength: scanned books, archival pages, and image-only PDFs become
usable when no clean text source exists.

Expected weakness: OCR errors and viewer-shell pages can mislead the agent if
the tool does not distinguish actual page content from provider UI metadata.

### M2C: Conditional Document OCR/Vision Inspect

M2C is the product-shaped version of M2.

The agent first reads document metadata, provider pages, and any provider
transcription. It requests document OCR/vision inspect only when the scan itself
must be observed to answer the mission. Provider transcription and
Plasma-generated OCR/vision observations must stay separate.

Expected strength: avoids paying OCR/vision cost for cases where provider text
is enough.

Expected weakness: depends on the agent recognizing that a viewer shell,
metadata page, or adjacent transcription is not the scanned document body.

## Corpora

Use small corpora that exercise different failure modes.

### C1: Oda Nobunaga Visual Research

Use known mission material around Oda Nobunaga, including Wikimedia Commons map
and portrait pages.

Checks:

- Can the agent distinguish a Commons file page from the actual image bytes?
- Does it avoid claiming to have read the map visually in M0?
- Does M1 improve descriptions of maps, territories, portraits, or battle
  diagrams?
- Does the final report include visual material with useful attribution?

### C2: Document Scan Versus Transcribed Text

Use one provider page that exposes transcribed text and one scan/viewer page
where the stored HTML is mostly a viewer shell.

Checks:

- Can the agent label provider transcription separately from scan OCR?
- Does it avoid treating a viewer shell as the book body?
- Does M2 improve extraction when only page images are available?
- Does the report state uncertainty when OCR is weak?

### C3: Performance Benchmark Graph Reading

Use a generated benchmark chart whose metadata explains the axes and color
legend, but intentionally omits the datapoints and the winner.

Checks:

- Does M0 avoid pretending that it can judge plotted performance without seeing
  the graph?
- Does M1 help judge which implementation is more stable at high load?
- Does M1C request inspect when the mission asks for graph-derived trend and
  tradeoff analysis?
- Does the agent identify that the better implementation changes by throughput
  range instead of giving a single oversimplified winner?
- Does the agent cite the original file/source rather than the inspection
  result?
- Does the report treat graph reading as an inspection result tied to the chart
  source, not as a new source?

### C4: Benchmark Variance and Error-Bar Reading

Use a generated benchmark bar chart with black error bars. Metadata explains the
metric, color legend, and what an error bar means, but not the actual bar
heights, spread, or winner.

Checks:

- Does M1C request inspect when the user's question depends on bar height and
  error-bar size?
- Does the report avoid reducing the decision to mean latency only?
- Does the agent describe the tradeoff between typical latency and measurement
  stability?

### C5: Dual-Axis Benchmark Tradeoff Reading

Use a generated dual-axis chart where throughput improves while error rate
increases later in the run. Metadata explains the two axes and line colors, but
not the turning point.

Checks:

- Does M1C request inspect when the answer depends on the relative movement of
  both axes?
- Does the report catch that a throughput improvement can hide a reliability
  regression?
- Does the agent avoid treating the two axes as one shared unit?

### C6: Truncated-Axis Benchmark Caution Reading

Use a generated pass-rate bar chart whose y-axis starts near 94 percent instead
of zero. Metadata tells the agent the axis is truncated, while the chart image
contains the visual exaggeration and relative differences.

Checks:

- Does M1C request inspect when the task asks how the visual presentation should
  be interpreted?
- Does the report warn that a visually large gap may represent a small absolute
  difference?
- Does the agent preserve both facts: higher pass rate is better, but the chart
  presentation requires caution?

## Method

Run each corpus across M0, always-inspect variants, and conditional-inspect
variants where applicable. After M0 is eliminated for graph-dependent
benchmark interpretation, follow-up graph corpora can run only M1 versus M1C.

PDF is intentionally out of scope for this experiment. PDF should be evaluated
later as a separate document-source experiment because text PDFs, scanned PDFs,
mixed PDFs, and paper/report PDFs require different extraction and inspection
tools.

For each run:

1. Start from the same mission objective and attached sources.
2. Use the same provider, model class, timeout, and report mode.
3. Let the agent investigate through Plasma tools only.
4. Generate both Markdown and interactive HTML report artifacts when the run
   reaches report generation.
5. Store transcripts, MCP tool logs, source manifests, report artifacts, and
   judge scores under the experiment directory.

Minimum pilot:

- 3 seeds per applicable corpus/variant pair.

Before the full pilot, run a one-seed harness smoke test with
`plasma/scripts/experiments/media-inspect/run_experiment.py`. This smoke test
uses Codex image attachments as an explicit stand-in for future Plasma inspect
tools. That means it can test whether inspection observations help and whether
the agent preserves the source/result boundary, but it does not prove the final
product implementation of `plasma.media.inspect` or document OCR.

Validation target before productizing:

- 6 seeds per applicable corpus/variant pair. Six paired blocks are the first
  useful threshold for a two-sided sign test to show a clean all-win pattern at
  p < 0.05. Stop earlier only if a variant hits a hard failure pattern that
  invalidates it.

## Metrics

### Source and Metadata Quality

- Correct source identity: source page, direct media URL, and artifact ID are
  not confused.
- License and attribution completeness.
- Duplicate handling: repeated media does not create user-facing constraint
  failures.
- Removed or inactive sources are not used by default.

### Visual or OCR Usefulness

- Number of accurate visual observations that were unavailable from metadata.
- Number of unsupported visual claims.
- Clear labeling of "not inspected", "provider transcription", "OCR result",
  and "vision observation".
- Ability to say "unknown" instead of inventing detail.

### Report Quality

- Media is used when it helps the argument or explanation.
- Markdown keeps canonical URLs and attribution visible.
- Interactive HTML embeds pinned images in self-contained mode when policy
  allows.
- Audio/video remain linked or provider-embedded, not inlined as blobs.
- The report does not treat inspection results as original sources.

### Cost and Latency

- Total run time.
- Number of media inspect/OCR calls.
- Bytes fetched and bytes stored.
- Report artifact size, especially self-contained HTML size.

## Hard Fail Conditions

A run is invalid and must be retried or rejected if it does any of the following:

- Claims visual inspection in M0 without an inspect tool call.
- Treats an agent/tool-produced caption, OCR, or visual observation as the
  original source.
- Treats a provider viewer shell as the scanned document body.
- Hides missing license or attribution information.
- Inlines audio/video bytes into a default self-contained report.
- Uses a removed source as if it were active.
- Fetches media through a path that bypasses the secure URL fetch boundary.

## Expected Product Implications

If M0 is good enough for most report embedding cases, Plasma should keep
metadata-only reads as the default.

If M1 materially improves map, portrait, diagram, or screenshot research without
causing unsupported claims, Plasma should add `plasma.media.inspect` as an
explicit optional tool.

If M2 materially improves scan-based research, Plasma should add a document path
that resolves IIIF/page images first and records OCR or vision output as
inspection results, not sources.

If C3 shows that inspect materially improves chart interpretation, Plasma should
let agents explicitly inspect graphs, charts, screenshots, and benchmark images
when the user's question depends on plotted visual structure. Metadata remains
the default read path; graph-derived observations remain results linked back to
the original chart source.

If C4-C6 show that M1C keeps quality close to M1 while reliably requesting
inspection, conditional inspect should be the product default. Always-inspect
should remain a comparison ceiling or an explicit user override, not the default
path.

None of these variants should change the source/result boundary:

- The media file, page, or scan is the source.
- Provider transcription can be a text source when it is the published material.
- Plasma-generated OCR, captions, and visual descriptions are results that point
  back to the source.
