# Protocol

1. Run only isolated experiment servers on `127.0.0.1:6010+`.
2. Use a per-run DB under `<experiment-runtime-root>`.
3. Use a per-run profile directory and agent session namespace.
4. Export only redacted ledger events and metadata summaries.
5. Treat usage telemetry, MCP traces, wrapper metadata, and reports as generated results,
   not sources or evidence.
6. Minimum baseline cannot start until harness smoke passes.
7. Minimum baseline runs use isolated HTTP/API product paths on `127.0.0.1:6010+`, not the live `6002` server.
8. Direct MCP tool-only provider usage is recorded as unavailable until an external wrapper is implemented.
9. Report-isolation measurements require a successful R1 preflight; unavailable preflight output must skip paired isolation runs.
10. `isolated_fork` is a product capability for non-one-take reports. Experiments may still
    pass it explicitly so the variant remains pinned instead of relying on automatic selection.
11. Report-isolation parallel runs are paired fixture blocks, not same-DB/same-mission causal
    product measurements. Use them to validate session-chain isolation, runtime isolation, and
    parallel stability before making primary product-effect claims.

## Current Smoke Command

```bash
python3 plasma/scripts/experiments/token-diet-measurement/run_experiment.py --phase smoke
```
