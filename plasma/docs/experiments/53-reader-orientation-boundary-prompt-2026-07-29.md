# Reader Orientation Boundary Prompt Experiment

Date: 2026-07-29

Status: corrected prompt accepted after host blind review, user re-reading, and
the separate full-product pipeline acceptance

Related issue: #190

## Question

Can the existing reader-edit stage make a report easier to follow by preserving
explanation that adds understanding while removing prose that merely narrates
the report's structure or writing process?

This was a prompt-only experiment. It did not add a workflow stage, MCP tool,
state transition, persistence rule, or recovery path.

## Product-Path Boundary

Each comparison held one upstream manuscript fixed and ran both prompts through
the product reader-edit MCP sequence:

1. start the bound reader edit,
2. read the complete manuscript,
3. apply reader-facing patches,
4. submit the edit once.

The pair used the same manuscript SHA, model, reasoning effort, report-plan
binding, and tool permissions. The style pass, integrity gate, and canonical
finalization were excluded so that any difference could be attributed to the
reader prompt. This verifies the real reader-stage path, not the complete
end-to-end report workflow.

## First Candidate And Failure

The first candidate asked the editor to explain the subject directly, preserve
explanatory value, and reduce repeated meta-signposting. Its blind result was
one win, two losses, and one tie across the four-cell matrix.

Direct reading showed that `meta-signposting` was too broad. In some outputs the
editor removed a useful report-level opening together with genuinely redundant
section roadmaps. The instruction therefore failed to distinguish orientation
that helps a reader from narration that only describes the document.

## Corrected Candidate

The correction made that boundary explicit:

- keep or create one brief report-level opening that states the subject,
  central question, and main answer or evidence boundary;
- let later transitions follow the subject and the reader's next question;
- remove repeated section roadmaps and writing-process narration;
- preserve transitions that add context, logic, or stakes;
- keep or add supported explanation when it clarifies a concept, causal link,
  condition, example, or technical detail;
- do not optimize for brevity by itself.

No other editor prompt or workflow behavior changed.

## Matrix And Result

| Topic | Rigor | Host blind result for corrected candidate |
| --- | --- | --- |
| Public health | Exploratory | Win |
| Public health | Strict | Win |
| Software engineering | Exploratory | Win |
| Software engineering | Strict | Win |

All eight reader runs completed and submitted exactly once. Each output differed
from its fixed input, and the execution checks found no style, gate, or
canonicalization event in the reader-only comparison.

Before revealing the arm mapping, the host read every A manuscript in full and
reviewed the complete A-to-B diff for each pair. The corrected candidate won all
four pairs. Its most consistent advantage was preserving or improving a useful
whole-report opening while still removing duplicated headings or local process
narration.

The user's initial blind choices were `B B B A`, which mapped to two candidate
wins and two baseline wins. After the mapping was revealed, the user re-read the
two exploratory pairs and changed pair 03 and then pair 01 to A. The final
considered choices were `A B A A`, matching the host choices and preferring the
corrected candidate in all four pairs. Because those revisions happened after
the reveal, they are recorded as considered preferences rather than blind wins.

The re-reading also refined the criterion. In pairs 01 and 03, the baseline had
some advantages in immediate tone or directness. The corrected candidate was
ultimately preferred because it carried the subject through one sustained flow,
helped the reader remain aware of the current question, and accumulated
information in a form that was easier to remember.

## Interpretation

The correction is a strong directional result for this bounded causal test. It
does not establish statistical significance: the matrix was run once, the host
was the only evaluator whose final four choices were all made blind, and the
user's final two revisions followed the mapping reveal. It also does not replace
an end-to-end product comparison.

The candidate remains in the Issue #190 worktree because it is narrower and
better supported than the first version. The recorded product-writing criterion
now distinguishes useful cognitive orientation from document self-narration and
separates immediate tone preference from sustained long-form comprehension.

The follow-up
[full-pipeline acceptance](54-reader-full-pipeline-acceptance-2026-07-29.md)
then ran this selected prompt through Part editing, reader editing, style
editing, the corrective integrity gate, and canonical finalization. One
exploratory and one strict report both preserved the reader-stage improvement,
so the prompt was accepted for Issue #190 integration without another prompt
change.

## Archive Evidence

Raw reports, run databases, provider traces, and blind packets remain outside
the repository under:

- `~/research-artifacts/liquid2/plasma/experiments/52-reader-explanatory-value-prompt-2026-07-29/`
- `~/research-artifacts/liquid2/plasma/experiments/53-reader-orientation-boundary-prompt-2026-07-29/`

This repository stores only the redacted protocol, aggregate result, and
decision boundary.
