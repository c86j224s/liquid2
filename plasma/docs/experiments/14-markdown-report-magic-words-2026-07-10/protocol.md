# Protocol

Raw archive:
`~/research-artifacts/liquid2/plasma/experiments/14-markdown-report-magic-words-2026-07-10/`

## Product Question

The experiment has two controlled questions:

1. Under complete-source conditions, do short Korean writing cues improve short
   and long Markdown reports?
2. When research has unavailable sources, rejected candidates, and unresolved
   gaps, does an explicit output-structure instruction keep the substantive
   report first and collect research limitations in one concise late section?

The second question was added after clarifying that the observed problem is not
failure to put a conclusion first. The problem is a long account of research
difficulty occupying the opening and pushing the report body down. The desired
shape keeps limitations out of the body and collects them separately after the
substantive content.

## Isolation

- Work runs from a dedicated issue 77 worktree and branch.
- Raw sources, generated reports, prompt hashes, model logs, blind packets,
  judge outputs, and private decode maps stay under the archive root. Phase 2
  writes only below `gap-stress/` within that root.
- The harness starts no server and opens no product database.
- Development and release Plasma runtimes, product defaults, and experiment 13
  are outside the experiment surface.
- Codex runs are ephemeral, read-only, and do not persist provider sessions.

## Phase 1: Complete Sources

Two archived source packets from experiment 10 are reused without modification:

1. A consumer purchase decision with prices, conditions, and time-sensitive
   caveats.
2. A technical OAuth/OIDC server design with implementation and security
   constraints.

Each input is generated in `short` and `long` modes. One run is made for every
sample, mode, and variant, for 20 report candidates total.

Each sample also receives the same short prior-investigation result in every
variant. This working-memory result recreates the observed pressure to discuss
source-access or verification difficulty before the report body. It is clearly
separated from the original source and may not be cited as source material. The
actual source body is present for every run.

### Variants

| ID | Controlled addition |
|---|---|
| `B0-baseline` | No added expression or composition instruction. |
| `W1-step-calm` | `차근차근`, `차분하게`. |
| `W2-human-flow` | `사람에게 잘 읽히는 문장으로`, `자연스러운 흐름으로`. |
| `C1-conclusion-first` | Put the conclusion and core judgment first; move investigation process and method commentary to a short late section when needed. |
| `C2-combined` | Combine W1, W2, and C1. |

All variants share the existing G2 direction: natural Korean must not be
obtained by deleting concrete facts, conditions, source distinctions, caveats,
or uncertainty.

### Measures

Deterministic metrics record report size, heading count, the position of the
first conclusion-like heading, early investigation-process mentions, and the
distribution of limitation terms.

Blind pairwise review compares each candidate with the baseline and compares
the combined variant with the explicit composition variant. Review axes are:

- early access to the core judgment;
- investigation-process exposure control;
- readability and natural flow;
- detail preservation;
- uncertainty integrity;
- source/result integrity.

The blind judge receives the same original source and the separately labeled
prior-investigation result used by generation. Variant identities remain
hidden. This allows the judge to penalize a report that cites or relabels an
agent-produced result as source material.

The phase is directional rather than statistically conclusive: it uses two
topics, one generation per cell, and one model-judge pass per comparison.

## Phase 2: Gap Stress

Phase 2 reuses the same two original sources and the same `short` and `long`
modes. For each topic, a structured `investigation_result` separately records:

- one source that could not be accessed;
- one candidate rejected as unsuitable evidence;
- one unresolved gap that can affect the decision.

This record is an agent-produced result, not source material. The generation
and judge prompts preserve that distinction. Four variants produce 16 reports:

| ID | Controlled addition |
|---|---|
| `B0-baseline` | No added reader-flow or placement instruction. |
| `R1-reader-flow` | Reader-oriented sentences and natural section flow only. |
| `L1-separate-late-limitations` | Substantive body first; research failure, rejection, and gaps only in one concise late `정보 한계와 영향` section; no chronological search narrative. |
| `C1-combined` | Combine reader-flow wording with the separate-late-limitations instruction. |

Blind review covers five comparisons per topic and mode, for 20 cases:

- reader flow versus baseline;
- separate late limitations versus baseline;
- combined versus baseline;
- combined versus reader flow;
- combined versus separate late limitations.

Deterministic measures record the first core and limitation heading positions,
limitation-section size and report share, early research-process mentions,
process and gap narrative outside the limitation section, and source/result
conflation. Blind review compares body-first order, limitation separation,
limitation concision, gap honesty, readability, detail preservation, and
source/result integrity.

Phase 2 is still a direct source-to-report stress fixture. It does not execute
the real Plasma research and report workflow and therefore cannot by itself
prove that the exact product-path opening failure is fixed.

## Commands

```sh
python3 plasma/scripts/experiments/markdown_report_magic_words_experiment.py --dry-run
python3 plasma/scripts/experiments/markdown_report_magic_words_experiment.py --stage all
python3 plasma/scripts/experiments/markdown_report_gap_stress_experiment.py --dry-run
python3 plasma/scripts/experiments/markdown_report_gap_stress_experiment.py --stage all --jobs 1 --judge-jobs 1
```

Each dry run validates exact archive and source-archive boundaries and prints
its matrix without model calls. Each full run resumes outputs only when the
stored prompt hash matches. Phase 2 defaults to one generation and one judge
worker so concurrent experiments do not share mutable runtime state.
