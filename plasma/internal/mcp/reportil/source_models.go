package reportil

type ReportILSourcesListInput struct{}

type ReportILSourcesListOutput struct {
	CatalogSHA256 string                   `json:"catalog_sha256"`
	Stage         string                   `json:"stage"`
	Attempt       int                      `json:"attempt"`
	Sources       []ReportILSourceListItem `json:"sources"`
}

type ReportILSourceListItem struct {
	SourceKey     string `json:"source_key"`
	ReadableBytes int    `json:"readable_bytes"`
	Extraction    string `json:"extraction"`
}

type ReportILSourcesReadInput struct {
	SourceKey string `json:"source_key"`
	Offset    int    `json:"offset"`
	MaxBytes  int    `json:"max_bytes"`
}

type ReportILSourceQuoteInput struct {
	SourceKey string `json:"source_key"`
	Quote     string `json:"quote"`
}

type ReportILSourceQuoteOutput struct {
	SourceReceipt string `json:"source_receipt"`
	SourceKey     string `json:"-"`
	Offset        int    `json:"-"`
	ByteSize      int    `json:"-"`
	Sha256        string `json:"-"`
	CatalogSHA256 string `json:"-"`
	Stage         string `json:"-"`
}

type ReportILSourcesReadOutput struct {
	SourceKey        string `json:"source_key"`
	CatalogSHA256    string `json:"catalog_sha256"`
	Stage            string `json:"stage"`
	Content          string `json:"content"`
	Offset           int    `json:"offset"`
	NextOffset       int    `json:"next_offset,omitempty"`
	ContentLength    int    `json:"content_length"`
	Truncated        bool   `json:"truncated"`
	Extraction       string `json:"extraction"`
	AttemptReadBytes int    `json:"attempt_read_bytes"`
	AttemptMaxBytes  int    `json:"attempt_max_bytes"`
}

type ReportILSourceBatchItem struct {
	SourceKey     string `json:"source_key"`
	Content       string `json:"content"`
	Offset        int    `json:"offset"`
	NextOffset    int    `json:"next_offset,omitempty"`
	ContentLength int    `json:"content_length"`
	Truncated     bool   `json:"truncated"`
	Extraction    string `json:"extraction"`
}

type ReportILSourcesBatchReadOutput struct {
	CatalogSHA256    string                    `json:"catalog_sha256"`
	Stage            string                    `json:"stage"`
	Sources          []ReportILSourceBatchItem `json:"sources"`
	RemainingSources int                       `json:"remaining_sources"`
	AttemptReadBytes int                       `json:"attempt_read_bytes"`
	AttemptMaxBytes  int                       `json:"attempt_max_bytes"`
}
