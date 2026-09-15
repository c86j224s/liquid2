# Automatic Compaction Scope

Plasma uses automatic compaction only for a resumed Web conversation turn when the provider reports that the model context is exhausted. The turn keeps the existing provider session: it runs one compact request, appends the compacted-session event, and retries the original request once.

The shared conversation capability owns the trigger predicate, strict same-session validation, phase reporting, elapsed durations, and callback order. It does not own provider execution, observation, or ledger storage. Workflow execution retains its separate empty-session fallback and its existing proactive compaction behavior.

Automatic compaction does not create a new session. A missing or different returned session ID is recorded as an invalid input, and a failed compaction append does not start the retry. Manual compaction remains a separate Web path.
