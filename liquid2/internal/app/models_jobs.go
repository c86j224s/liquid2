package app

const (
	JobKindScrapeURL         = "scrape_url"
	JobKindTranslateDocument = "translate_document"
	JobKindPollFeed          = "poll_feed"
	JobKindExtractUploadText = "extract_upload_text"
	JobStatusQueued          = "queued"
	JobStatusRunning         = "running"
	JobStatusCompleted       = "completed"
	JobStatusFailed          = "failed"
)

// Job describes background work status.
type Job struct {
	// ID is the stable job identifier.
	ID string `json:"id"`
	// Kind identifies the worker operation.
	Kind string `json:"kind"`
	// Status is the current lifecycle state.
	Status string `json:"status"`
	// PayloadJSON stores adapter-owned job payload data.
	PayloadJSON string `json:"-"`
	// Error contains the safe failure message when failed.
	Error *string `json:"error"`
	// Attempts counts starts from queued to running.
	Attempts int64 `json:"attempts"`
	// CreatedAt is the creation timestamp in Unix milliseconds.
	CreatedAt int64 `json:"createdAt"`
	// UpdatedAt is the last update timestamp in Unix milliseconds.
	UpdatedAt int64 `json:"updatedAt"`
	// StartedAt is set when a worker starts the job.
	StartedAt *int64 `json:"startedAt"`
	// FinishedAt is set when the job reaches a terminal state.
	FinishedAt *int64 `json:"finishedAt"`
}

// JobFilters contains job list criteria.
type JobFilters struct {
	// Status filters by lifecycle state.
	Status string
	// Kind filters by job kind.
	Kind string
	// Limit caps the number of returned jobs.
	Limit int
}

// JobList is a job list response.
type JobList struct {
	// Items contains the current page of jobs.
	Items []Job `json:"items"`
}
