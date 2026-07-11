# Execution Status

Completed: protocol, source corpora, run manifests, score-matrix headers,
decision placeholders, and static validation.

Blocked: full primary execution and judging require a runnable provider-session
harness that can prove same-session final report generation, collect tool
traces, enforce hard-fail contamination rules live, and produce blinded judge
packets without leaking cell labels.

Next preflight command before primary execution:

```sh
python3 plasma/docs/experiments/03-controller-question-repertoire-2026-06-21/09-g0-controller-mcp-followup-2026-06-23/tools/validate_experiment.py
```

Post-run validation must use:

```sh
python3 plasma/docs/experiments/03-controller-question-repertoire-2026-06-21/09-g0-controller-mcp-followup-2026-06-23/tools/validate_experiment.py --phase post-run
```

During pilot or incremental execution, use:

```sh
python3 plasma/docs/experiments/03-controller-question-repertoire-2026-06-21/09-g0-controller-mcp-followup-2026-06-23/tools/validate_experiment.py --phase partial
```
