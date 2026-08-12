package missionusage

const AggregationVersion = "mission_usage.v3"

// Summary is the read-only token usage view for one mission. Partial is true
// when at least one recorded agent call cannot be included exactly.
type Summary struct {
	AggregationVersion    string             `json:"aggregation_version"`
	UsageRecordCount      int64              `json:"usage_record_count"`
	UsageAvailableCount   int64              `json:"usage_available_count"`
	UsageUnavailableCount int64              `json:"usage_unavailable_count"`
	SessionCount          int64              `json:"session_count"`
	CounterResetCount     int64              `json:"counter_reset_count"`
	FailedCallCount       int64              `json:"failed_call_count"`
	FailedTotalTokens     int64              `json:"failed_total_tokens"`
	InputTokens           int64              `json:"input_tokens"`
	CachedInputTokens     int64              `json:"cached_input_tokens"`
	UncachedInputTokens   int64              `json:"uncached_input_tokens"`
	OutputTokens          int64              `json:"output_tokens"`
	ReasoningOutputTokens int64              `json:"reasoning_output_tokens"`
	TotalTokens           int64              `json:"total_tokens"`
	UsagePartial          bool               `json:"usage_partial"`
	PerCall               Percentiles        `json:"per_call"`
	Categories            []Category         `json:"categories"`
	WorkflowRuns          []WorkflowRunUsage `json:"workflow_runs,omitempty"`
}

// Percentiles summarize corrected total tokens per available agent call.
type Percentiles struct {
	P50 int64 `json:"p50"`
	P90 int64 `json:"p90"`
	Max int64 `json:"max"`
}

// Category groups agent calls by the product capability that requested them.
// Unknown future surfaces remain visible under other instead of being dropped.
type Category struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	CallCount   int64  `json:"call_count"`
	TotalTokens int64  `json:"total_tokens"`
}

// WorkflowRunUsage groups corrected workflow-step increments by autonomous
// research run. WorkflowRunID is stable API identity; user interfaces should
// prefer an ordinal label and must not expose provider session identifiers.
type WorkflowRunUsage struct {
	WorkflowRunID         string      `json:"workflow_run_id"`
	CallCount             int64       `json:"call_count"`
	SessionCount          int64       `json:"session_count"`
	ResumedCallCount      int64       `json:"resumed_call_count"`
	InputTokens           int64       `json:"input_tokens"`
	CachedInputTokens     int64       `json:"cached_input_tokens"`
	UncachedInputTokens   int64       `json:"uncached_input_tokens"`
	OutputTokens          int64       `json:"output_tokens"`
	ReasoningOutputTokens int64       `json:"reasoning_output_tokens"`
	TotalTokens           int64       `json:"total_tokens"`
	AgentModel            string      `json:"agent_model,omitempty"`
	ReasoningEffort       string      `json:"reasoning_effort,omitempty"`
	PerCall               Percentiles `json:"per_call"`
}
