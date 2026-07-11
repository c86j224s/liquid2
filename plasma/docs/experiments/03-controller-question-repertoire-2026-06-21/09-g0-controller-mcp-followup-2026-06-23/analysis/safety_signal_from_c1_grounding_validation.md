# Safety Signal From C1 Grounding Validation

Referenced paths:

- `.fleet/plans/plasma-c1-grounding-validation.md`
- `plasma/docs/experiments/03-controller-question-repertoire-2026-06-21/08-c1-grounding-validation-2026-06-22/`
- `plasma/scripts/c1-controller-runner.py`
- `plasma/scripts/grounding-validation.py`

Allowed use:

- Safety prior.
- Audit checklist.
- Non-inferiority threshold input.
- Stop-condition motivation.

Disallowed use:

- Merging into new primary score matrices.
- Claiming controller adoption or rejection.
- Claiming MCP surface adoption or rejection.
- Treating older runs as product validation evidence.

Summary: in the older artifact, no-controller had better grounding/provenance
and controller variants showed increased unverifiable/provenance risk. This is a
reason to measure safety carefully, not a reason to abandon controller work.
