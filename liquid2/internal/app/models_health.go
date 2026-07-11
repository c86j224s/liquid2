package app

// Health reports whether the API process can serve requests.
type Health struct {
	// OK is true when the process-level health check passes.
	OK bool `json:"ok"`
}
